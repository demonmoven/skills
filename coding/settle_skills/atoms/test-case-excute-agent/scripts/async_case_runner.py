import argparse
import datetime
import json
import os
import re
import sys
import time
from typing import Any, Dict, List, Optional, Tuple
from urllib.parse import urlparse, parse_qs

import requests

from report_md_formatter import generate_markdown
from standard_json_formatter import format_to_standard


# 固定的 HTTP 重试次数（提交与轮询内部使用，不暴露为脚本参数）
MAX_HTTP_RETRIES = 3

# 异步执行服务固定地址
BASE_URL = "https://e9r97mvz.fn.bytedance.net"

# 用于校验 bits_url 路径中的 caseDetailId
CASE_DETAIL_REGEX = re.compile(r"/caseDetail/(\d+)")


def _parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="批量测试用例异步发起 + 任务状态轮询 + Markdown 报告生成",
    )

    # 脚本运行参数（不进入请求体）
    parser.add_argument(
        "--user",
        required=True,
        help="用户邮箱前缀，必填，例如 yangchen.shine。服务端会据此换取用户 token 并记录触发人",
    )
    parser.add_argument(
        "--messages-file",
        help="包含字符串数组的 JSON 文件路径，文件内容必须是 JSON 数组字符串，例如 [\"case1\", \"case2\"]",
    )
    parser.add_argument(
        "--bits-url",
        dest="bits_url",
        help=(
            "Bits 用例树长链接。若传入，必须包含非空的 projectId 查询参数，且路径中能解析到 "
            "`/caseDetail/数字` 形式的 caseDetailId；否则脚本会在本地校验阶段失败并退出"
        ),
    )
    parser.add_argument(
        "--env",
        help=(
            "执行环境，可选，例如 prod 或 boe_xxx。若不提供，则脚本不会在请求体中设置 env 字段，"
            "由服务端按 boe_* 规则或默认 prod 处理"
        ),
    )
    parser.add_argument(
        "--poll-interval",
        type=int,
        default=5,
        help="轮询状态接口的时间间隔（秒），默认 5",
    )
    parser.add_argument(
        "--max-wait",
        type=int,
        default=1200,
        help="最大等待时间（秒），超过后视为超时，默认 1200",
    )
    parser.add_argument(
        "--report-dir",
        default="reports",
        help="报告输出目录，默认 reports/（相对当前工作目录）",
    )
    parser.add_argument(
        "--emit-standard-json",
        dest="emit_standard_json",
        action="store_true",
        default=True,
        help="是否生成标准化执行记录 JSON，默认开启；如需关闭，可配合 --no-emit-standard-json 使用",
    )
    parser.add_argument(
        "--no-emit-standard-json",
        dest="emit_standard_json",
        action="store_false",
        help="关闭标准化执行记录 JSON 生成（兼容开关参数）",
    )
    parser.add_argument(
        "--standard-json-path",
        help=(
            "标准化执行记录 JSON 输出路径，可选。未提供时默认输出到 "
            "<report-dir>/execution_records_{task_id}.json"
        ),
    )

    return parser.parse_args()


def _load_messages_from_file(path: str) -> List[str]:
    """从 JSON 文件加载 messages 列表。

    要求文件内容为 JSON 数组字符串，例如:
    [
      "case1",
      "case2"
    ]
    """

    with open(path, "r", encoding="utf-8") as f:
        content = f.read()
    try:
        data = json.loads(content)
    except json.JSONDecodeError as e:
        raise ValueError(f"messages-file 不是合法 JSON：{e}") from e

    if not isinstance(data, list):
        raise ValueError("messages-file 必须是 JSON 数组字符串，例如 [\"case1\", \"case2\"]")

    return [str(x) for x in data]


def _build_messages(args: argparse.Namespace) -> List[str]:
    """构造 messages 列表，优先级：--messages-file > --bits-url
    - 两个参数必须至少传一个
    """
    messages: List[str] = []

    # 优先从文件加载
    if args.messages_file:
        messages_from_file = _load_messages_from_file(args.messages_file)
        messages.extend(messages_from_file)

    return messages


def _validate_bits_url(bits_url: str) -> Tuple[bool, str]:
    """校验 bits_url 是否满足：

    - query 中包含非空 projectId；
    - path 中能够解析到 /caseDetail/{数字} 形式的 caseDetailId。
    """

    if not bits_url:
        return True, ""

    try:
        parsed = urlparse(bits_url)
    except Exception as e:  # noqa: BLE001
        return False, f"URL 解析失败：{e}"

    # 校验 projectId
    query = parse_qs(parsed.query or "")
    project_ids = query.get("projectId") or []
    project_id = project_ids[0] if project_ids else ""
    if not project_id:
        return False, "bits_url 缺少 projectId 查询参数，例如 ?projectId=123456"

    # 校验 caseDetailId
    if not CASE_DETAIL_REGEX.search(parsed.path or ""):
        return False, "bits_url 路径中未找到 /caseDetail/{数字}，无法解析 caseDetailId"

    return True, ""


def _extract_task_info(resp_data: Any) -> Tuple[Optional[str], Optional[str]]:
    """从 NewChatCompletions 响应中解析 task_id / status。

    兼容：
    - {"task_id": "...", "status": "processing"}
    - {"code": 200, "data": {"task_id": "...", "status": "processing"}}
    """

    if not isinstance(resp_data, dict):
        return None, None

    if "task_id" in resp_data:
        return str(resp_data.get("task_id")), resp_data.get("status")

    data = resp_data.get("data")
    if isinstance(data, dict) and "task_id" in data:
        return str(data.get("task_id")), data.get("status")

    return None, None


def _normalize_task_status(raw_status: Any) -> Optional[str]:
    if raw_status is None:
        return None
    s = str(raw_status).lower()
    if s in {"success", "succeeded", "done", "finished", "completed"}:
        return "success"
    if s in {"failed", "fail", "error", "exception"}:
        return "failed"
    if s in {"timeout", "timedout"}:
        return "timeout"
    if s in {"processing", "running", "pending", "in_progress"}:
        return "processing"
    return None


def _interpret_task_data(
    data: Any,
) -> Tuple[str, Optional[Dict[str, Any]], Optional[str], Optional[str]]:
    """解析任务详情 data，返回 (status, batch_result, parse_error, task_error)。

    解析规则：
    - 状态优先读取 ``task_status``，兼容 ``status``，并统一归一到 ``success`` / ``failed`` / ``timeout`` / ``processing``；
    - 错误信息优先读取 ``error_msg``，兼容 ``error`` / ``error_message``；
    - 结果字段兼容 ``result`` / ``task_result`` / ``Result`` / ``taskResult``，支持 dict 与 JSON 字符串两种形态；
    - 当 data 本体即为 batchResult（含 ``total_cases`` / ``case_results``）时，直接作为 batchResult 使用。
    """

    parse_error: Optional[str] = None
    task_error: Optional[str] = None

    if data is None:
        # 未查到任务或任务仍在初始化阶段，视为 processing
        return "processing", None, None, None

    # data 为 dict 的通用情况（TaskResponse.data 为 AgentTask 或 batchResult）
    if isinstance(data, dict):
        # 1) 标准化任务状态：优先 task_status，其次 status
        raw_status = data.get("task_status")
        if raw_status is None:
            raw_status = data.get("status")
        status = _normalize_task_status(raw_status)

        # 2) 任务错误信息：优先 error_msg，其次 error / error_message
        for key in ("error_msg", "error", "error_message"):
            val = data.get(key)
            if val:
                task_error = str(val)
                break

        # 3) data 本体就是 batchResult（total_cases + case_results）
        if "total_cases" in data and "case_results" in data:
            if status is None:
                status = "success"
            return status, data, None, task_error

        # 4) 解析 result / task_result / Result / taskResult
        raw_result: Any = None
        result_key_used: Optional[str] = None
        for key in ("result", "task_result", "Result", "taskResult"):
            if key in data:
                raw_result = data.get(key)
                result_key_used = key
                break

        result_obj: Any = None
        if result_key_used is not None:
            if isinstance(raw_result, (dict, list)):
                result_obj = raw_result
            elif isinstance(raw_result, str):
                try:
                    result_obj = json.loads(raw_result)
                except json.JSONDecodeError as e:
                    # JSON 字符串解析失败，记录 parse_error，后续会体现在报告顶部
                    parse_error = f"{result_key_used} 字段 JSON 解析失败: {e}"
            else:
                # 其他基础类型（如数字/布尔）直接透传
                result_obj = raw_result

        # 4.1 result 解析出标准 batchResult
        if isinstance(result_obj, dict) and "total_cases" in result_obj and "case_results" in result_obj:
            if status is None:
                status = "success"
            return status, result_obj, parse_error, task_error

        # 4.2 虽非标准 batchResult，但仍返回给上游兜底展示
        if result_obj is not None:
            if status is None:
                status = "success"
            return status, result_obj, parse_error, task_error

        # 5) 没有可用的 result，尝试根据状态与错误信息推断
        if status is None:
            # 有错误信息则视为失败，否则视为仍在处理中
            status = "failed" if task_error else "processing"

        if status in {"failed", "timeout"} or task_error:
            return status, None, parse_error, task_error

        # 默认认为任务仍在处理中
        return status, None, parse_error, task_error

    # data 是字符串，尝试整体解析（兼容 data 直接是 batchResult 字符串的场景）
    if isinstance(data, str):
        try:
            result_obj = json.loads(data)
        except json.JSONDecodeError as e:  # noqa: PERF203
            parse_error = f"data JSON 解析失败: {e}"
            return "failed", None, parse_error, None

        if isinstance(result_obj, dict) and "total_cases" in result_obj and "case_results" in result_obj:
            return "success", result_obj, None, None
        return "success", result_obj, None, None

    # 其他结构暂不支持
    parse_error = f"不支持的 data 类型: {type(data)}"
    return "failed", None, parse_error, None


def _post_new_chat_completions(
    payload: Dict[str, Any],
) -> Tuple[str, str, Dict[str, Any]]:
    """调用 NewChatCompletions 发起异步任务。"""

    url = BASE_URL.rstrip("/") + "/api/v4/bots/chat/completions"
    last_error: Optional[str] = None

    for attempt in range(1, MAX_HTTP_RETRIES + 1):
        try:
            resp = requests.post(url, json=payload, timeout=30)
        except requests.RequestException as e:  # noqa: PERF203
            last_error = f"第 {attempt} 次调用发起接口失败: {e}"
            if attempt < MAX_HTTP_RETRIES:
                time.sleep(1)
                continue
            raise RuntimeError(last_error) from e

        try:
            data = resp.json()
        except json.JSONDecodeError as e:  # noqa: PERF203
            last_error = f"发起接口返回非 JSON：{e}，响应内容: {resp.text[:200]}"
            if attempt < MAX_HTTP_RETRIES:
                time.sleep(1)
                continue
            raise RuntimeError(last_error) from e

        if resp.status_code != 200:
            last_error = f"发起接口返回非 200 状态码: {resp.status_code}, body={data!r}"
            if attempt < MAX_HTTP_RETRIES:
                time.sleep(1)
                continue
            raise RuntimeError(last_error)

        task_id, status = _extract_task_info(data)
        if not task_id:
            last_error = f"无法从发起接口响应中解析 task_id，响应内容: {data!r}"
            if attempt < MAX_HTTP_RETRIES:
                time.sleep(1)
                continue
            raise RuntimeError(last_error)

        return task_id, status or "processing", data

    raise RuntimeError(last_error or "调用发起接口失败")


def _poll_task_until_done(
    task_id: str,
    poll_interval: int,
    max_wait: int,
) -> Dict[str, Any]:
    """轮询任务状态直至结束，返回聚合结果。"""

    url = BASE_URL.rstrip("/") + "/api/v3/bots/task/status"

    start_time = time.time()
    attempts = 0
    last_json: Optional[Dict[str, Any]] = None
    last_error: Optional[str] = None
    batch_result: Optional[Dict[str, Any]] = None
    final_status: Optional[str] = None

    while True:
        elapsed = time.time() - start_time
        if elapsed > max_wait:
            final_status = "timeout"
            if not last_error:
                last_error = f"轮询超时：等待 {max_wait} 秒后任务仍未完成。"
            break

        attempts += 1
        try:
            resp = requests.get(url, params={"task_id": task_id}, timeout=30)
        except requests.RequestException as e:  # noqa: PERF203
            last_error = f"第 {attempts} 次查询任务状态失败: {e}"
            if attempts >= MAX_HTTP_RETRIES:
                final_status = "failed"
                break
            time.sleep(poll_interval)
            continue

        try:
            json_data = resp.json()
        except json.JSONDecodeError as e:  # noqa: PERF203
            last_error = f"任务状态接口返回非 JSON：{e}，响应内容: {resp.text[:200]}"
            if attempts >= MAX_HTTP_RETRIES:
                final_status = "failed"
                break
            time.sleep(poll_interval)
            continue

        last_json = json_data
        code = json_data.get("code")
        if code is not None and code != 200:
            last_error = f"任务状态接口返回 code={code}, message={json_data.get('message')}"
            final_status = "failed"
            break

        data = json_data.get("data")
        status, batch_result_candidate, parse_error, task_error = _interpret_task_data(data)

        if task_error:
            last_error = task_error
        if parse_error:
            # 解析失败也需要记录，用于报告
            last_error = parse_error

        if batch_result_candidate is not None:
            batch_result = batch_result_candidate

        if status in {"success", "failed", "timeout"}:
            final_status = status
            break

        # 未结束，继续轮询
        time.sleep(poll_interval)

    if final_status is None:
        final_status = "failed"

    if final_status == "success" and batch_result is None:
        # 任务标记成功但结果缺失，视为失败并给出明确错误
        final_status = "failed"
        if not last_error:
            last_error = "任务状态为 success，但 data/result 中未找到 batchResult。"

    return {
        "status": final_status,
        "batch_result": batch_result,
        "raw": last_json,
        "error": last_error,
    }


def _ensure_dir(path: str) -> None:
    os.makedirs(path, exist_ok=True)


def _now_str() -> str:
    return datetime.datetime.now().strftime("%Y-%m-%d %H:%M:%S")


def _generate_error_report(task_id: str, env: str, status: str, error_msg: str) -> str:
    """在任务提交或轮询失败时生成简要错误报告。"""

    lines: List[str] = []
    lines.append("# 批量测试执行报告")
    lines.append(f"**生成时间**: {_now_str()}")
    if task_id:
        lines.append(f"**任务 ID**: {task_id}")
    if env:
        lines.append(f"**环境**: {env}")

    if status:
        status_map = {
            "success": "成功",
            "failed": "失败",
            "timeout": "超时",
        }
        status_label = status_map.get(status.lower(), status)
        lines.append(f"**任务状态**: {status_label}")

    if error_msg:
        lines.append(f"**任务错误**: {error_msg}")

    lines.append("")
    lines.append("## 📊 执行摘要")
    lines.append("- **总用例数**: 0")
    lines.append("- **成功**: 0")
    lines.append("- **失败**: 0")
    lines.append("- **成功率**: 0.00%")
    lines.append("")
    lines.append("## ❌ 执行失败")
    if error_msg:
        lines.append("")
        lines.append(f"**错误信息**: {error_msg}")
    return "\n".join(lines)


def main() -> None:
    args = _parse_args()

    user = (args.user or "").strip()
    if not user:
        print("参数错误: --user 为必填，需传入用户邮箱前缀，例如 foo.bar", file=sys.stderr)
        sys.exit(1)

    bits_url = (args.bits_url or "").strip()
    # 校验参数：messages-file 和 bits-url 必须至少传一个
    if not args.messages_file and not bits_url:
        print("参数错误: 必须传入 --messages-file 或 --bits-url 其中至少一个参数", file=sys.stderr)
        sys.exit(1)
        
    if bits_url:
        ok, reason = _validate_bits_url(bits_url)
        if not ok:
            print(f"参数错误: 无效的 --bits-url：{reason}", file=sys.stderr)
            sys.exit(1)

    try:
        messages = _build_messages(args)
    except ValueError as e:
        print(f"参数错误: {e}", file=sys.stderr)
        sys.exit(1)

    # 构造请求体：优先级 messages-file > bits-url
    # 传入了 messages-file 则使用文件内容，否则使用 bits-url
    env = (args.env or "").strip() or None

    payload: Dict[str, Any] = {
        "messages": messages,
        "user": user,
    }
    # 仅当未传入 messages-file 时，才使用 bits_url
    if bits_url and not args.messages_file:
        payload["bits_url"] = bits_url
    if env:
        payload["env"] = env

    task_id: str = ""
    submit_status: str = ""
    final_status: str = "failed"
    batch_result: Optional[Dict[str, Any]] = None
    error_msg: str = ""

    try:
        task_id, submit_status, _ = _post_new_chat_completions(
            payload=payload,
        )
    except Exception as e:  # noqa: BLE001
        # 发起阶段失败，直接生成失败报告
        error_msg = f"任务发起失败: {e}"
        final_status = "failed"
    else:
        poll_result = _poll_task_until_done(
            task_id=task_id,
            poll_interval=args.poll_interval,
            max_wait=args.max_wait,
        )
        final_status = str(poll_result.get("status") or "failed")
        batch_result = poll_result.get("batch_result")
        error_msg = poll_result.get("error") or ""

    _ensure_dir(args.report_dir)
    timestamp = datetime.datetime.now().strftime("%Y%m%d_%H%M%S")

    if task_id:
        report_filename = f"execution_report_{task_id}.md"
        summary_filename = f"summary_{task_id}.json"
    else:
        report_filename = f"execution_report_{timestamp}.md"
        summary_filename = f"summary_{timestamp}.json"

    report_path = os.path.join(args.report_dir, report_filename)
    summary_path = os.path.join(args.report_dir, summary_filename)

    env_for_report = env or ""

    if batch_result is not None:
        meta: Dict[str, Any] = {
            "task_id": task_id,
            "env": env_for_report,
            "generated_at": _now_str(),
            "status": final_status,
        }
        if error_msg:
            meta["error"] = error_msg
        md_text = generate_markdown(batch_result, meta)
    else:
        md_text = _generate_error_report(task_id, env_for_report, final_status, error_msg)

    with open(report_path, "w", encoding="utf-8") as f:
        f.write(md_text)

    # 标准化执行记录 JSON 产出（默认开启），在已有 batch_result 的前提下追加生成
    standard_json_relpath: Optional[str] = None
    if getattr(args, "emit_standard_json", True) and batch_result is not None:
        if args.standard_json_path:
            standard_json_path = args.standard_json_path
            if not os.path.isabs(standard_json_path):
                standard_json_path_to_use = standard_json_path
            else:
                standard_json_path_to_use = standard_json_path
        else:
            if task_id:
                json_filename = f"execution_records_{task_id}.json"
            else:
                json_filename = f"execution_records_{timestamp}.json"
            standard_json_path_to_use = os.path.join(args.report_dir, json_filename)

        standard_parent = os.path.dirname(standard_json_path_to_use)
        if standard_parent:
            os.makedirs(standard_parent, exist_ok=True)

        try:
            format_to_standard(batch_result, standard_json_path_to_use, task_id=task_id or None)
            standard_json_relpath = os.path.relpath(standard_json_path_to_use).replace("\\", "/")
        except Exception:  # noqa: BLE001
            # 标准 JSON 生成失败不影响主流程
            standard_json_relpath = None

    output_obj: Dict[str, Any] = {
        "task_id": task_id,
        "status": final_status,
        "report_md": os.path.relpath(report_path).replace("\\", "/"),
    }
    if error_msg:
        output_obj["error"] = error_msg
    if submit_status and not error_msg:
        output_obj["submit_status"] = submit_status
    if env_for_report:
        output_obj["env"] = env_for_report
    if standard_json_relpath:
        output_obj["standard_json"] = standard_json_relpath

    with open(summary_path, "w", encoding="utf-8") as f:
        json.dump(output_obj, f, ensure_ascii=False, indent=2)

    print(json.dumps(output_obj, ensure_ascii=False))


if __name__ == "__main__":
    main()