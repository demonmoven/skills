#!/usr/bin/env python3
"""
打点数据上报脚本

Usage:
    # 使用默认生成的 JSON 数据上报
    python report_skill.py --skill "api-test"

    # customs-only 步骤上报
    python report_skill.py step-start --step-id S1_VREGION --step-name "确认测试 VRegion"
    python report_skill.py step-finish --step-id S1_VREGION --result success

    # 从文件读取 JSON 数据上报
    python report_skill.py --file metrics.json

    # 直接传递 JSON 字符串
    python report_skill.py --json '{"report_info": {...}}'
"""

import argparse
import hashlib
import json
import os
from pathlib import Path
import sys
import tempfile
import time
from typing import Any, Optional
import uuid

# 默认上报 URL
DEFAULT_REPORT_URL = "https://ms-explorer.byted.org/explorer/v5/bam_skills/report"
FIXED_SKILL = "api-test"
FIXED_MODE = "Fast call"
SKILL_ACTION = "skill"
STEP_ACTION_PREFIX = "step_"

# 环境变量名称
ENV_SOURCE = "EXEC_SOURCE"
ENV_SESSION_ID = "EXEC_SESSION_ID"
ENV_STATE_ROOT = "REPORT_SKILL_STATE_ROOT"
ENV_PLATFORM = "REPORT_SKILL_PLATFORM"

# 本地状态文件配置
STATE_DIR_NAME = "api_test_report_skill"
SKILL_STATE_KEY = "__skill__"
STATE_META_SESSION_ID = "__state_meta__"
CURRENT_SESSION_PREFIX = "__current_session__"

REMOVED_CUSTOM_KEYS = {"event_type", "step_id"}

# 执行结果
RESULT_SUCCESS = "success"
RESULT_FAILED = "failed"
RESULT_SKIPPED = "skipped"
RESULT_CHOICES = (RESULT_SUCCESS, RESULT_FAILED, RESULT_SKIPPED)

PLATFORM_ALIAS_MAP = {
    ".trae": "trae",
    ".trae-cn": "trae-cn",
    ".claude": "claude",
    ".cursor": "cursor",
    ".windsurf": "windsurf",
    ".gemini": "gemini",
}


def normalize_platform_name(raw_name: str) -> str:
    """规范化 agent/platform 名称。"""
    normalized = (raw_name or "").strip()
    if not normalized:
        return ""
    if normalized in PLATFORM_ALIAS_MAP:
        return PLATFORM_ALIAS_MAP[normalized]
    if normalized.startswith("."):
        return normalized[1:]
    return normalized


def derive_platform_from_path(file_path: Path) -> str:
    """根据脚本路径推导调用该 skill 的 agent 名称。"""
    for part in file_path.parts:
        if part in PLATFORM_ALIAS_MAP:
            return PLATFORM_ALIAS_MAP[part]

    parts = file_path.parts
    for index, part in enumerate(parts):
        if part == "skills" and index > 0:
            candidate = normalize_platform_name(parts[index - 1])
            if candidate and candidate not in {"skills", "skills-demo", "bam_skill"}:
                return candidate

    return "unknown"


def resolve_platform(explicit_platform: str = "") -> str:
    """解析上报 platform，优先显式传参，其次环境变量，最后按脚本路径推导。"""
    normalized = normalize_platform_name(explicit_platform)
    if normalized:
        return normalized

    for env_name in (ENV_PLATFORM, "AGENT_PLATFORM", "BAM_SKILL_AGENT"):
        env_value = normalize_platform_name(os.environ.get(env_name, ""))
        if env_value:
            return env_value

    raw_path = Path(__file__)
    derived = derive_platform_from_path(raw_path)
    if derived != "unknown":
        return derived

    return derive_platform_from_path(raw_path.resolve())


def build_step_action(step_id: str) -> str:
    """构造 step 级 action。"""
    return f"{STEP_ACTION_PREFIX}{step_id.strip()}"


def resolve_action(explicit_action: str = "", *, step_id: str = "") -> str:
    """解析上报 action。"""
    normalized_step_id = (step_id or "").strip()
    if normalized_step_id:
        return build_step_action(normalized_step_id)

    normalized_action = (explicit_action or "").strip()
    if normalized_action.startswith(STEP_ACTION_PREFIX):
        return normalized_action
    if normalized_action == SKILL_ACTION:
        return SKILL_ACTION
    return SKILL_ACTION


def sanitize_customs(customs: Any) -> dict[str, Any]:
    """清洗 customs，仅保留允许上报的字段。"""
    if not isinstance(customs, dict):
        return {}

    sanitized: dict[str, Any] = {}
    for key, value in customs.items():
        if key in REMOVED_CUSTOM_KEYS:
            continue
        sanitized[key] = value
    return sanitized


def normalize_report_info(report_info: dict[str, Any], platform: str = "", action: str = "") -> dict[str, Any]:
    """强制收敛 api-test 的上报字段，避免被调用方覆盖固定值。"""
    normalized = dict(report_info or {})
    raw_customs = normalized.get("customs")
    step_id = ""
    if isinstance(raw_customs, dict):
        step_id = str(raw_customs.get("step_id") or "").strip()
    normalized["skill"] = FIXED_SKILL
    normalized["mode"] = FIXED_MODE
    normalized["action"] = resolve_action(action or str(normalized.get("action") or ""), step_id=step_id)
    normalized["platform"] = resolve_platform(platform or str(normalized.get("platform") or ""))
    normalized["customs"] = sanitize_customs(raw_customs)
    return normalized


def coerce_custom_value(raw_value: str) -> Any:
    """将命令行 customs 值尽量转成基础类型。"""
    value = raw_value.strip()
    if value.lstrip("-").isdigit():
        return int(value)

    lowered = value.lower()
    if lowered == "true":
        return True
    if lowered == "false":
        return False
    return value


def parse_custom_items(items: Optional[list[str]]) -> dict[str, Any]:
    """解析 `--custom key=value` 列表。"""
    customs: dict[str, Any] = {}
    for item in items or []:
        if "=" not in item:
            print(f"Warning: ignore invalid custom '{item}', expected key=value")
            continue

        key, value = item.split("=", 1)
        key = key.strip()
        if not key:
            print(f"Warning: ignore invalid custom '{item}', key is empty")
            continue
        customs[key] = coerce_custom_value(value)
    return customs


def ensure_session_id(session_id: str) -> tuple[str, bool]:
    """确保存在 session_id；缺失时生成新的 UUID。"""
    normalized = (session_id or "").strip()
    if normalized:
        return normalized, False
    return str(uuid.uuid4()), True


def build_current_session_key(platform: str) -> str:
    """构造当前活跃 session 元信息键。"""
    normalized_platform = normalize_platform_name(platform) or "unknown"
    return f"{CURRENT_SESSION_PREFIX}:{normalized_platform}"


def get_state_dir() -> Path:
    """获取步骤耗时状态文件目录。"""
    root = os.environ.get(ENV_STATE_ROOT, tempfile.gettempdir())
    state_dir = Path(root) / STATE_DIR_NAME
    state_dir.mkdir(parents=True, exist_ok=True)
    return state_dir


def build_state_path(session_id: str, state_key: str) -> Path:
    """为 session + state_key 生成稳定的本地状态文件路径。"""
    digest = hashlib.sha256(f"{session_id}:{state_key}".encode("utf-8")).hexdigest()
    return get_state_dir() / f"{digest}.json"


def write_state(session_id: str, state_key: str, data: dict[str, Any]) -> None:
    """写入本地状态文件。"""
    path = build_state_path(session_id, state_key)
    with path.open("w", encoding="utf-8") as handle:
        json.dump(data, handle, ensure_ascii=False, indent=2)


def read_state(session_id: str, state_key: str) -> Optional[dict[str, Any]]:
    """读取本地状态文件。"""
    path = build_state_path(session_id, state_key)
    if not path.exists():
        return None

    try:
        with path.open("r", encoding="utf-8") as handle:
            loaded = json.load(handle)
    except (OSError, json.JSONDecodeError) as error:
        print(f"Warning: failed to read state file {path}: {error}")
        return None

    return loaded if isinstance(loaded, dict) else None


def remove_state(session_id: str, state_key: str) -> None:
    """删除本地状态文件。"""
    path = build_state_path(session_id, state_key)
    try:
        path.unlink(missing_ok=True)
    except OSError as error:
        print(f"Warning: failed to remove state file {path}: {error}")


def build_state_key(step_id: Optional[str] = None) -> str:
    """构造状态文件键。"""
    return step_id or SKILL_STATE_KEY


def write_current_session(platform: str, session_id: str) -> None:
    """记录当前活跃 session_id，供同一会话内后续命令复用。"""
    write_state(
        STATE_META_SESSION_ID,
        build_current_session_key(platform),
        {
            "meta_type": "current_session",
            "platform": normalize_platform_name(platform) or "unknown",
            "session_id": session_id,
            "updated_at_ms": current_timestamp_ms(),
        },
    )


def read_current_session(platform: str) -> Optional[dict[str, Any]]:
    """读取当前活跃 session 元信息。"""
    return read_state(STATE_META_SESSION_ID, build_current_session_key(platform))


def remove_current_session(platform: str) -> None:
    """删除当前活跃 session 元信息。"""
    remove_state(STATE_META_SESSION_ID, build_current_session_key(platform))


def has_active_execution_state(session_id: str) -> bool:
    """判断当前 session 是否仍存在未完成的本地执行状态。"""
    state_dir = get_state_dir()
    for path in state_dir.glob("*.json"):
        try:
            with path.open("r", encoding="utf-8") as handle:
                loaded = json.load(handle)
        except (OSError, json.JSONDecodeError):
            continue

        if not isinstance(loaded, dict):
            continue
        if loaded.get("meta_type") == "current_session":
            continue
        if loaded.get("session_id") == session_id and loaded.get("state_type") in {"step", "skill"}:
            return True
    return False


def resolve_session_id(explicit_session_id: str, platform: str) -> tuple[str, bool]:
    """解析或创建可复用的 session_id，确保同一会话内 start/finish 对齐。"""
    normalized = (explicit_session_id or "").strip()
    normalized_platform = normalize_platform_name(platform) or "unknown"
    if normalized:
        write_current_session(normalized_platform, normalized)
        return normalized, False

    current_session = read_current_session(normalized_platform) or {}
    current_session_id = str(current_session.get("session_id") or "").strip()
    if current_session_id and has_active_execution_state(current_session_id):
        write_current_session(normalized_platform, current_session_id)
        return current_session_id, False

    generated_session_id = str(uuid.uuid4())
    write_current_session(normalized_platform, generated_session_id)
    return generated_session_id, True


def current_timestamp_ms() -> int:
    """返回当前毫秒级时间戳。"""
    return int(time.time() * 1000)


def compute_duration_ms(started_at_ms: Optional[int], ended_at_ms: Optional[int] = None) -> int:
    """计算耗时，保证返回非负整数。"""
    if started_at_ms is None:
        return 0

    end_ms = ended_at_ms if ended_at_ms is not None else current_timestamp_ms()
    return max(0, int(end_ms - int(started_at_ms)))


def build_event_customs(
    base_customs: Optional[dict[str, Any]],
    *,
    step_name: Optional[str] = None,
    parent_step_id: Optional[str] = None,
    result: Optional[str] = None,
    duration_ms: Optional[int] = None,
    error_type: Optional[str] = None,
    skip_reason: Optional[str] = None,
) -> dict[str, Any]:
    """组装 customs-only 的事件字段。"""
    customs = dict(base_customs or {})
    if step_name:
        customs["step_name"] = step_name
    if parent_step_id:
        customs["parent_step_id"] = parent_step_id
    if result:
        customs["result"] = result
    if duration_ms is not None:
        customs["duration_ms"] = max(0, int(duration_ms))
    if error_type:
        customs["error_type"] = error_type
    if skip_reason:
        customs["skip_reason"] = skip_reason

    return customs


def generate_metrics_data(
    *,
    source: str = "",
    version: str = "",
    session_id: str = "",
    customs: Optional[dict[str, Any]] = None,
    platform: str = "",
    action: str = "",
) -> dict[str, Any]:
    """生成统一的 api-test 上报 JSON 数据。"""
    data = {
        "report_info": normalize_report_info(
            {
                "skill": FIXED_SKILL,
                "mode": FIXED_MODE,
                "action": action,
                "platform": platform,
                "source": source,
                "version": version,
                "session_id": session_id,
                "customs": customs or {},
            },
            platform=platform,
            action=action,
        )
    }
    return data


def load_json_from_file(file_path: str) -> dict[str, Any]:
    """从文件加载 JSON 数据。"""
    try:
        with open(file_path, "r", encoding="utf-8") as handle:
            return json.load(handle)
    except FileNotFoundError:
        print(f"Error: File not found: {file_path}")
        sys.exit(1)
    except json.JSONDecodeError as error:
        print(f"Error: Invalid JSON in file {file_path}: {error}")
        sys.exit(1)


def load_json_from_string(json_str: str) -> dict[str, Any]:
    """从字符串解析 JSON 数据。"""
    try:
        return json.loads(json_str)
    except json.JSONDecodeError as error:
        print(f"Error: Invalid JSON string: {error}")
        sys.exit(1)


def parse_headers(items: Optional[list[str]]) -> dict[str, str]:
    """解析自定义请求头。"""
    headers: dict[str, str] = {}
    for item in items or []:
        if ":" not in item:
            print(f"Warning: ignore invalid header '{item}', expected 'Key: Value'")
            continue
        key, value = item.split(":", 1)
        headers[key.strip()] = value.strip()
    return headers


def normalize_loaded_data(
    data: dict[str, Any],
    fallback_session_id: str,
    platform: str = "",
) -> dict[str, Any]:
    """对外部传入的 JSON 数据做最小归一化。"""
    normalized = dict(data)
    report_info = normalized.get("report_info")
    if not isinstance(report_info, dict):
        return normalized

    normalized_report_info = normalize_report_info(
        report_info,
        platform=platform,
        action=str(report_info.get("action") or ""),
    )
    session_id, generated = ensure_session_id(
        str(normalized_report_info.get("session_id") or fallback_session_id)
    )
    if generated:
        print(f"Info: generated session_id '{session_id}' for metrics reporting")
    normalized_report_info["session_id"] = session_id
    normalized["report_info"] = normalized_report_info
    return normalized


def report_metrics(
    url: str,
    data: dict[str, Any],
    headers: Optional[dict] = None,
    timeout: int = 10,
) -> bool:
    """
    上报打点数据

    Args:
        url: 上报接口 URL
        data: 要上报的 JSON 数据
        headers: 自定义请求头
        timeout: 请求超时时间(秒)

    Returns:
        bool: 上报是否成功
    """
    try:
        import requests
    except ModuleNotFoundError:
        print("Error: Missing dependency 'requests'. Please install it before reporting metrics.")
        return False

    default_headers = {
        "Content-Type": "application/json",
        "Accept": "application/json",
    }
    if headers:
        default_headers.update(headers)

    try:
        response = requests.post(
            url,
            json=data,
            headers=default_headers,
            timeout=timeout,
        )
        response.raise_for_status()

        print(f"Report success: {response.status_code}")
        print(f"Response: {response.text}")
        return True

    except requests.exceptions.Timeout:
        print(f"Error: Request timeout after {timeout}s")
        return False
    except requests.exceptions.ConnectionError as error:
        print(f"Error: Connection failed: {error}")
        return False
    except requests.exceptions.HTTPError as error:
        print(f"Error: HTTP error: {error}")
        print(f"Response body: {response.text}")
        return False
    except Exception as error:  # pylint: disable=broad-except
        print(f"Error: Unexpected error: {error}")
        return False


def execute_report(data: dict[str, Any], *, headers: dict[str, str], timeout: int, dry_run: bool) -> bool:
    """统一输出并执行上报。"""
    print(f"Data: {json.dumps(data, ensure_ascii=False, indent=2)}")

    report_info = data.get("report_info")
    customs = report_info.get("customs") if isinstance(report_info, dict) else None
    if not isinstance(customs, dict) or not customs:
        print("Info: skip report because customs is empty after normalization")
        return True

    if dry_run:
        print("\n[Dry-run mode] 不执行实际上报")
        return True

    return report_metrics(
        url=DEFAULT_REPORT_URL,
        data=data,
        headers=headers if headers else None,
        timeout=timeout,
    )


def add_shared_report_args(parser: argparse.ArgumentParser) -> None:
    """为子命令添加通用上报参数。"""
    parser.add_argument(
        "--source",
        default=os.environ.get(ENV_SOURCE, ""),
        help=f"来源 (默认: 环境变量 {ENV_SOURCE} 或 '')",
    )
    parser.add_argument(
        "--version",
        "-v",
        default="0.0.1",
        help="版本号",
    )
    parser.add_argument(
        "--session-id",
        default=os.environ.get(ENV_SESSION_ID, ""),
        help=f"会话ID (默认: 环境变量 {ENV_SESSION_ID})",
    )
    parser.add_argument(
        "--platform",
        default=os.environ.get(ENV_PLATFORM, ""),
        help=f"调用该 skill 的 agent 名称；默认优先读环境变量 {ENV_PLATFORM}，否则按脚本路径推导",
    )
    parser.add_argument(
        "--custom",
        "-c",
        action="append",
        help="自定义字段，格式: 'key=value'，可多次使用",
    )
    parser.add_argument(
        "--header",
        "-H",
        action="append",
        help="自定义请求头，格式: 'Key: Value'，可多次使用",
    )
    parser.add_argument(
        "--timeout",
        "-t",
        type=int,
        default=10,
        help="请求超时时间(秒)，默认10",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="只打印数据，不实际上报",
    )


def build_step_start_parser() -> argparse.ArgumentParser:
    """构造 step-start 子命令解析器。"""
    parser = argparse.ArgumentParser(description="记录步骤开始事件并写入本地状态文件")
    parser.add_argument("--step-id", required=True, help="步骤 ID，例如 S1_VREGION")
    parser.add_argument("--step-name", required=True, help="步骤名称")
    parser.add_argument("--parent-step-id", help="父步骤 ID，例如 S5_TEST")
    add_shared_report_args(parser)
    return parser


def build_step_finish_parser() -> argparse.ArgumentParser:
    """构造 step-finish 子命令解析器。"""
    parser = argparse.ArgumentParser(description="记录步骤结束事件并自动计算耗时")
    parser.add_argument("--step-id", required=True, help="步骤 ID，例如 S1_VREGION")
    parser.add_argument("--step-name", help="步骤名称；未传时尝试从状态文件回填")
    parser.add_argument("--parent-step-id", help="父步骤 ID；未传时尝试从状态文件回填")
    parser.add_argument("--result", required=True, choices=RESULT_CHOICES, help="步骤执行结果")
    parser.add_argument("--error-type", help="失败时的错误分类")
    parser.add_argument("--skip-reason", help="跳过时的原因")
    parser.add_argument("--duration-ms", type=int, help="手工指定耗时；未传时尝试从状态文件自动计算")
    add_shared_report_args(parser)
    return parser


def build_skill_start_parser() -> argparse.ArgumentParser:
    """构造 skill-start 子命令解析器。"""
    parser = argparse.ArgumentParser(description="记录 skill 开始事件并写入本地状态文件")
    add_shared_report_args(parser)
    return parser


def build_skill_finish_parser() -> argparse.ArgumentParser:
    """构造 skill-finish 子命令解析器。"""
    parser = argparse.ArgumentParser(description="记录 skill 结束事件并自动计算耗时")
    parser.add_argument("--result", default=RESULT_SUCCESS, choices=RESULT_CHOICES, help="skill 执行结果")
    parser.add_argument("--error-type", help="失败时的错误分类")
    parser.add_argument("--skip-reason", help="跳过时的原因")
    parser.add_argument("--duration-ms", type=int, help="手工指定耗时；未传时尝试从状态文件自动计算")
    add_shared_report_args(parser)
    return parser


def run_step_start(argv: list[str]) -> int:
    """执行 step-start 子命令。"""
    args = build_step_start_parser().parse_args(argv)
    resolved_platform = resolve_platform(args.platform)
    session_id, generated = resolve_session_id(args.session_id, resolved_platform)
    if generated:
        print(f"Info: generated session_id '{session_id}' for step-start")

    base_customs = parse_custom_items(args.custom)
    write_state(
        session_id,
        build_state_key(args.step_id),
        {
            "state_type": "step",
            "session_id": session_id,
            "started_at_ms": current_timestamp_ms(),
            "step_id": args.step_id,
            "step_name": args.step_name,
            "parent_step_id": args.parent_step_id or "",
            "source": args.source,
            "version": args.version,
            "platform": resolved_platform,
            "custom_context": base_customs,
        },
    )
    print(
        "State recorded locally for step-start: "
        f"session_id={session_id}, step_id={args.step_id}, platform={resolved_platform}"
    )
    if args.dry_run:
        print("[Dry-run mode] step-start is local-only; no report sent")
    return 0


def run_step_finish(argv: list[str]) -> int:
    """执行 step-finish 子命令。"""
    args = build_step_finish_parser().parse_args(argv)
    requested_platform = resolve_platform(args.platform)
    session_id, generated = resolve_session_id(args.session_id, requested_platform)
    if generated:
        print(f"Info: generated session_id '{session_id}' for step-finish")

    headers = parse_headers(args.header)
    base_customs = parse_custom_items(args.custom)
    stored_state = read_state(session_id, build_state_key(args.step_id))
    stored_custom_context = (stored_state or {}).get("custom_context")
    if isinstance(stored_custom_context, dict):
        merged_customs = dict(stored_custom_context)
        merged_customs.update(base_customs)
        base_customs = merged_customs

    step_name = args.step_name or (stored_state or {}).get("step_name") or ""
    parent_step_id = args.parent_step_id or (stored_state or {}).get("parent_step_id") or ""
    resolved_platform = resolve_platform(
        args.platform or str((stored_state or {}).get("platform") or "")
    )
    if args.duration_ms is not None:
        duration_ms = max(0, int(args.duration_ms))
    else:
        started_at_ms = (stored_state or {}).get("started_at_ms")
        if started_at_ms is None:
            print(f"Warning: no state found for step '{args.step_id}', fallback duration_ms=0")
        duration_ms = compute_duration_ms(started_at_ms)

    if args.result == RESULT_FAILED and not args.error_type:
        print("Warning: result=failed but --error-type is empty")
    if args.result == RESULT_SKIPPED and not args.skip_reason:
        print("Warning: result=skipped but --skip-reason is empty")

    customs = build_event_customs(
        base_customs,
        step_name=step_name,
        parent_step_id=parent_step_id,
        result=args.result,
        duration_ms=duration_ms,
        error_type=args.error_type,
        skip_reason=args.skip_reason,
    )
    data = generate_metrics_data(
        source=args.source,
        version=args.version,
        session_id=session_id,
        customs=customs,
        platform=resolved_platform,
        action=build_step_action(args.step_id),
    )

    success = execute_report(data, headers=headers, timeout=args.timeout, dry_run=args.dry_run)
    if success:
        remove_state(session_id, build_state_key(args.step_id))
        if not has_active_execution_state(session_id):
            remove_current_session(resolved_platform)
    else:
        print(f"Info: keep step state for retry: {args.step_id}")
    return 0 if success else 1


def run_skill_start(argv: list[str]) -> int:
    """执行 skill-start 子命令。"""
    args = build_skill_start_parser().parse_args(argv)
    resolved_platform = resolve_platform(args.platform)
    session_id, generated = resolve_session_id(args.session_id, resolved_platform)
    if generated:
        print(f"Info: generated session_id '{session_id}' for skill-start")

    base_customs = parse_custom_items(args.custom)

    write_state(
        session_id,
        build_state_key(),
        {
            "state_type": "skill",
            "session_id": session_id,
            "started_at_ms": current_timestamp_ms(),
            "source": args.source,
            "version": args.version,
            "platform": resolved_platform,
            "custom_context": base_customs,
        },
    )
    print(f"State recorded locally for skill-start: session_id={session_id}, platform={resolved_platform}")
    if args.dry_run:
        print("[Dry-run mode] skill-start is local-only; no report sent")
    return 0


def run_skill_finish(argv: list[str]) -> int:
    """执行 skill-finish 子命令。"""
    args = build_skill_finish_parser().parse_args(argv)
    requested_platform = resolve_platform(args.platform)
    session_id, generated = resolve_session_id(args.session_id, requested_platform)
    if generated:
        print(f"Info: generated session_id '{session_id}' for skill-finish")

    headers = parse_headers(args.header)
    base_customs = parse_custom_items(args.custom)
    stored_state = read_state(session_id, build_state_key())
    stored_custom_context = (stored_state or {}).get("custom_context")
    if isinstance(stored_custom_context, dict):
        merged_customs = dict(stored_custom_context)
        merged_customs.update(base_customs)
        base_customs = merged_customs
    resolved_platform = resolve_platform(
        args.platform or str((stored_state or {}).get("platform") or "")
    )

    if args.duration_ms is not None:
        duration_ms = max(0, int(args.duration_ms))
    else:
        started_at_ms = (stored_state or {}).get("started_at_ms")
        if started_at_ms is None:
            print("Warning: no state found for skill-start, fallback duration_ms=0")
        duration_ms = compute_duration_ms(started_at_ms)

    if args.result == RESULT_FAILED and not args.error_type:
        print("Warning: result=failed but --error-type is empty")
    if args.result == RESULT_SKIPPED and not args.skip_reason:
        print("Warning: result=skipped but --skip-reason is empty")

    customs = build_event_customs(
        base_customs,
        result=args.result,
        duration_ms=duration_ms,
        error_type=args.error_type,
        skip_reason=args.skip_reason,
    )
    data = generate_metrics_data(
        source=args.source,
        version=args.version,
        session_id=session_id,
        customs=customs,
        platform=resolved_platform,
        action=SKILL_ACTION,
    )

    success = execute_report(data, headers=headers, timeout=args.timeout, dry_run=args.dry_run)
    if success:
        remove_state(session_id, build_state_key())
        if not has_active_execution_state(session_id):
            remove_current_session(resolved_platform)
    else:
        print("Info: keep skill state for retry")
    return 0 if success else 1


def build_default_parser() -> argparse.ArgumentParser:
    """构造默认模式解析器，兼容历史调用方式。"""
    parser = argparse.ArgumentParser(
        description="打点数据上报脚本",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  %(prog)s --skill "api-test"                    # 基础上报
  %(prog)s --skill "api-test" --mode "Fast call" --version "0.0.1"
  %(prog)s --skill "api-test" --custom "key1=123" --custom "key2=value"
  %(prog)s --file metrics.json                   # 从文件读取 JSON 上报
  %(prog)s --json '{"report_info": {...}}'       # 直接传递 JSON 字符串
  %(prog)s step-start --step-id S1_VREGION --step-name "确认测试 VRegion"  # 仅写本地开始时间
  %(prog)s step-finish --step-id S1_VREGION --result success               # 读取开始时间并上报
  %(prog)s --dry-run                             # 预览数据，不实际上报
        """,
    )
    parser.add_argument("--file", "-f", help="JSON 数据文件路径")
    parser.add_argument("--json", "-j", help="JSON 字符串数据")
    parser.add_argument(
        "--skill",
        "-s",
        required=False,
        default=FIXED_SKILL,
        help=f"技能名称固定为 {FIXED_SKILL}；若传入其他值会被自动纠正",
    )
    parser.add_argument(
        "--mode",
        "-m",
        default=FIXED_MODE,
        help=f"运行模式固定为 {FIXED_MODE}；若传入其他值会被自动纠正",
    )
    parser.add_argument(
        "--source",
        default=os.environ.get(ENV_SOURCE, ""),
        help=f"来源 (默认: 环境变量 {ENV_SOURCE} 或 '')",
    )
    parser.add_argument("--version", "-v", default="0.0.1", help="版本号")
    parser.add_argument(
        "--session-id",
        default=os.environ.get(ENV_SESSION_ID, ""),
        help=f"会话ID (默认: 环境变量 {ENV_SESSION_ID})",
    )
    parser.add_argument(
        "--platform",
        default=os.environ.get(ENV_PLATFORM, ""),
        help=f"调用该 skill 的 agent 名称；默认优先读环境变量 {ENV_PLATFORM}，否则按脚本路径推导",
    )
    parser.add_argument(
        "--custom",
        "-c",
        action="append",
        help="自定义字段，格式: 'key=value'，可多次使用，例如: -c 'test1=111' -c 'test2=222'",
    )
    parser.add_argument(
        "--header",
        "-H",
        action="append",
        help="自定义请求头，格式: 'Key: Value'，可多次使用",
    )
    parser.add_argument("--timeout", "-t", type=int, default=10, help="请求超时时间(秒)，默认10")
    parser.add_argument("--dry-run", action="store_true", help="只打印数据，不实际上报")
    return parser


def run_default(argv: list[str]) -> int:
    """执行默认兼容模式。"""
    args = build_default_parser().parse_args(argv)

    if args.skill != FIXED_SKILL:
        print(
            f"Info: override --skill from '{args.skill}' to '{FIXED_SKILL}' "
            "for api-test metrics reporting"
        )
        args.skill = FIXED_SKILL
    if args.mode != FIXED_MODE:
        print(
            f"Info: override --mode from '{args.mode}' to '{FIXED_MODE}' "
            "for api-test metrics reporting"
        )
        args.mode = FIXED_MODE

    headers = parse_headers(args.header)
    resolved_platform = resolve_platform(args.platform)
    session_id, generated = resolve_session_id(args.session_id, resolved_platform)
    if generated:
        print(f"Info: generated session_id '{session_id}' for metrics reporting")

    if args.file:
        data = normalize_loaded_data(load_json_from_file(args.file), session_id, platform=resolved_platform)
    elif args.json:
        data = normalize_loaded_data(load_json_from_string(args.json), session_id, platform=resolved_platform)
    else:
        customs = parse_custom_items(args.custom)
        data = generate_metrics_data(
            source=args.source,
            version=args.version,
            session_id=session_id,
            customs=customs if customs else None,
            platform=resolved_platform,
            action=SKILL_ACTION,
        )

    success = execute_report(data, headers=headers, timeout=args.timeout, dry_run=args.dry_run)
    return 0 if success else 1


def main() -> None:
    """主入口。"""
    argv = sys.argv[1:]
    if not argv:
        sys.exit(run_default([]))

    subcommand = argv[0]
    if subcommand == "step-start":
        sys.exit(run_step_start(argv[1:]))
    if subcommand == "step-finish":
        sys.exit(run_step_finish(argv[1:]))
    if subcommand == "skill-start":
        sys.exit(run_skill_start(argv[1:]))
    if subcommand == "skill-finish":
        sys.exit(run_skill_finish(argv[1:]))

    sys.exit(run_default(argv))


if __name__ == "__main__":
    main()
