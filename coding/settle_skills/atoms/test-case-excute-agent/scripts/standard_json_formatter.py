import argparse
import json
import os
import re
from typing import Any, Dict, List, Optional, Tuple, Set


def _safe_str(value: Any) -> str:
    """将任意对象安全转为字符串，避免 None 带来的异常。"""

    if value is None:
        return ""
    try:
        return str(value)
    except Exception:  # noqa: BLE001
        return ""


def _try_parse_json(text: str) -> Optional[Dict[str, Any]]:
    """尝试将字符串解析为 JSON 对象，失败则返回 None。"""

    try:
        obj = json.loads(text)
    except (TypeError, json.JSONDecodeError):  # noqa: PERF203
        return None
    if isinstance(obj, dict):
        return obj
    return None


def _extract_module_from_goal(goal: str) -> str:
    """从 goal 文本中提取模块信息，优先匹配方括号、书名号等包裹内容。"""

    if not goal:
        return ""
    patterns = [
        r"【([^】]+)】",
        r"\[([^\[\]]+)\]",
        r"《([^》]+)》",
        r"「([^」]+)」",
        r"『([^』]+)』",
    ]
    for pattern in patterns:
        m = re.search(pattern, goal)
        if m:
            return m.group(1).strip()
    return ""


def _extract_title_from_goal(goal: str) -> str:
    """从 goal 文本中提取标题信息，优先匹配第一个枚举项（1、 / 1. / (1) / （1））。"""

    if not goal:
        return ""

    # 常见枚举形式："1、xxx"、"1. xxx"、"(1) xxx"、"（1）xxx"
    patterns = [
        r"(?:^|[：:\n])\s*1[、.]\s*([^\n]+)",
        r"(?:^|[：:\n])\s*\(1\)\s*([^\n]+)",
        r"(?:^|[：:\n])\s*（1）\s*([^\n]+)",
    ]
    for pattern in patterns:
        m = re.search(pattern, goal)
        if m:
            return m.group(1).strip()

    return ""


def _extract_case_id_from_text(text: str) -> str:
    """从原始文本中提取 case_id 字段，支持 case_id 与 caseID_ 前缀形式。"""

    if not text:
        return ""

    # 1) 优先匹配形式类似于 "case_id":"xxx"、"case_id: xxx"、"case_id = xxx"
    m = re.search(r"case_id[\"'\s:=：]*([A-Za-z0-9_\-]+)", text)
    if m:
        return m.group(1).strip()

    # 2) 回退匹配形如 caseID_xxx 的片段
    m = re.search(r"(caseID_[A-Za-z0-9_]+)", text)
    if m:
        return m.group(1).strip()

    return ""


def _extract_case_basic_fields(test_case_str: Any) -> Tuple[str, str, str]:
    """从 test_case_str 中提取 case_id/module/title 三个基础字段。"""

    raw_text = _safe_str(test_case_str).strip()

    case_id = ""
    module = ""
    title = ""

    obj = _try_parse_json(raw_text)

    goal_text = ""
    if obj is not None:
        # 直接从 JSON 中读取显式字段
        case_id = _safe_str(obj.get("case_id") or obj.get("caseId")).strip()
        module = _safe_str(obj.get("module") or obj.get("模块")).strip()
        title = _safe_str(obj.get("title") or obj.get("标题")).strip()
        goal_text = _safe_str(obj.get("goal") or obj.get("用例描述") or obj.get("desc")).strip()

    # 若 JSON 中未获取到 case_id，则尝试从原始文本中匹配
    if not case_id:
        case_id = _extract_case_id_from_text(raw_text)

    # 模块：JSON 中没有时，尝试从 goal 文本中提取
    if not module:
        if not goal_text:
            goal_text = raw_text
        module = _extract_module_from_goal(goal_text)

    # 标题：JSON 中没有时，优先从 goal 文本中的第一个枚举项提取
    if not title:
        if not goal_text:
            goal_text = raw_text
        title = _extract_title_from_goal(goal_text)

    return case_id or "", module or "", title or ""


def _collect_all_log_ids(value: Any, max_depth: int = 3, seen_str: Optional[Set[str]] = None) -> Set[str]:
    """
    递归收集所有符合条件的logid：
    1. key包含 log/Log/LOG（不区分大小写）
    2. value非空且为字符串
    3. 嵌套JSON解析，限制深度
    4. 自动去重
    """
    log_ids: Set[str] = set()
    if seen_str is None:
        seen_str = set()

    if max_depth <= 0:
        return log_ids

    # 处理字典：匹配key包含log，提取value
    if isinstance(value, dict):
        for key, val in value.items():
            if isinstance(key, str) and "log" in key.lower():
                log_str = _safe_str(val).strip()
                # 非空字符串判定为logid
                if log_str:
                    log_ids.add(log_str)
            # 递归处理子节点
            log_ids.update(_collect_all_log_ids(val, max_depth - 1, seen_str))
        return log_ids

    # 处理列表
    if isinstance(value, list):
        for item in value:
            log_ids.update(_collect_all_log_ids(item, max_depth, seen_str))
        return log_ids

    # 处理字符串：尝试解析嵌套JSON
    if isinstance(value, str):
        s = value.strip()
        if not s or s in seen_str:
            return log_ids
        seen_str.add(s)
        try:
            nested_obj = json.loads(s)
            log_ids.update(_collect_all_log_ids(nested_obj, max_depth - 1, seen_str))
        except (json.JSONDecodeError, TypeError):
            pass
        return log_ids

    return log_ids


def _extract_logid_from_nested_json(text: str, max_depth: int = 3) -> str:
    """
    解析嵌套JSON，收集所有匹配的logid，去重后逗号分隔返回
    """
    try:
        root = json.loads(text)
    except (TypeError, json.JSONDecodeError):
        return ""

    log_id_set = _collect_all_log_ids(root, max_depth)
    return ",".join(sorted(log_id_set)) if log_id_set else ""


_LOGID_TEXT_PATTERN = re.compile(r"(?:重试\s*logid|logid)[\s：:]*([0-9A-Za-z]+)", re.IGNORECASE)
# 匹配所有包含log的关键词，提取后面的字符串
_ALL_LOG_TEXT_PATTERN = re.compile(r"log[^：:\n]*[：:\s]+([0-9a-zA-Z]+)", re.IGNORECASE)


def _extract_logid_from_text(text: str) -> str:
    """从文本中提取所有logid，去重后逗号分隔"""
    if not text:
        return ""

    log_ids = set()
    # 匹配原有规则
    matches = _LOGID_TEXT_PATTERN.findall(text)
    log_ids.update(matches)
    # 匹配所有包含log的关键词
    all_matches = _ALL_LOG_TEXT_PATTERN.findall(text)
    log_ids.update(all_matches)
    # 纯字母数字串也视为logid
    stripped = text.strip()
    if re.fullmatch(r"[0-9A-Za-z]+", stripped):
        log_ids.add(stripped)

    return ",".join(sorted(log_ids)) if log_ids else ""


def _extract_log_id(tool_output: Any) -> str:
    """从工具执行输出中提取 logid，优先嵌套JSON，其次文本正则，返回所有去重后的logid"""
    text = _safe_str(tool_output)
    if not text:
        return ""

    # 1) 从嵌套JSON中提取所有logid
    log_ids_from_json = _extract_logid_from_nested_json(text)
    if log_ids_from_json:
        return log_ids_from_json

    # 2) 从文本中提取所有logid
    log_ids_from_text = _extract_logid_from_text(text)
    if log_ids_from_text:
        return log_ids_from_text

    return ""


def _get_first_nonempty(data: Dict[str, Any], keys: List[str]) -> str:
    """从字典中按顺序获取第一个非空键的字符串值。"""

    for key in keys:
        if key in data and data[key] not in (None, ""):
            return _safe_str(data[key])
    return ""


def _extract_steps(case_result: Dict[str, Any]) -> List[Dict[str, Any]]:
    """从 case_result 中提取步骤列表，优先 final_plan.steps，其次 initial_plan.steps。"""

    steps: Any = None
    final_plan = case_result.get("final_plan") or {}
    if isinstance(final_plan, dict):
        steps = final_plan.get("steps")
    if steps is None:
        initial_plan = case_result.get("initial_plan") or {}
        if isinstance(initial_plan, dict):
            steps = initial_plan.get("steps")
    if isinstance(steps, list):
        return steps
    return []


def _extract_tool_exec_results(step_records: Any) -> List[Dict[str, str]]:
    """从 step_records 中提取工具执行记录列表。"""

    results: List[Dict[str, str]] = []
    if not isinstance(step_records, list):
        return results

    for record in step_records:
        if not isinstance(record, dict):
            continue
        tool_name = _get_first_nonempty(record, ["tool_name", "工具名称"])
        tool_input = _get_first_nonempty(record, ["tool_input", "工具执行输入"])
        tool_output = _get_first_nonempty(record, ["tool_output", "工具执行输出"])
        description = _get_first_nonempty(record, ["reason", "description"])
        exec_status = _get_first_nonempty(record, ["expected_success"])
        log_id = _extract_log_id(tool_output)

        results.append(
            {
                "tool_name": tool_name,
                "tool_url": "",
                "tool_input": tool_input,
                "tool_output": tool_output,
                "description": description,
                "exec_status": exec_status,
                "log_id": log_id,
            }
        )

    return results


def format_to_standard(batch_result: Dict[str, Any], output_path: str, task_id: Optional[str] = None) -> Dict[str, Any]:
    """将轮询得到的 batch_result 转换为标准化执行记录 JSON 文件并返回摘要。

    返回的摘要包含记录条数与最终输出路径，调用方可用于后续归档或日志记录。
    """

    standard_records: List[Dict[str, Any]] = []

    try:
        case_results: Any
        if isinstance(batch_result, dict):
            case_results = batch_result.get("case_results")
        elif isinstance(batch_result, list):
            case_results = batch_result
        else:
            case_results = []

        if not isinstance(case_results, list):
            case_results = []

        for case in case_results:
            if not isinstance(case, dict):
                continue

            test_case_str = case.get("test_case_str") or case.get("test_case") or ""
            case_id, module, title = _extract_case_basic_fields(test_case_str)

            steps = _extract_steps(case)
            step_records = case.get("step_records") or []
            tool_exec_result = _extract_tool_exec_results(step_records)

            record: Dict[str, Any] = {
                "case_id": case_id,
                "module": module,
                "title": title,
                "steps": steps if isinstance(steps, list) else [],
                "tool_exec_result": tool_exec_result,
            }
            standard_records.append(record)
    except Exception:  # noqa: BLE001
        # 所有异常兜底为生成空数组文件，避免抛出异常中断主流程
        standard_records = []

    # 确保输出目录存在
    parent = os.path.dirname(output_path)
    if parent:
        os.makedirs(parent, exist_ok=True)

    with open(output_path, "w", encoding="utf-8") as f:
        json.dump(standard_records, f, ensure_ascii=False, indent=2)

    summary: Dict[str, Any] = {
        "task_id": task_id or "",
        "record_count": len(standard_records),
        "output_path": os.path.abspath(output_path),
    }
    return summary


def _load_json_file(path: str) -> Dict[str, Any]:
    """从本地 JSON 文件中加载 batch_result。"""

    with open(path, "r", encoding="utf-8") as f:
        return json.load(f)


def _parse_args() -> argparse.Namespace:
    """解析命令行参数，用于离线将 batch_result 转为标准化执行记录 JSON。"""

    parser = argparse.ArgumentParser(
        description="将批量执行结果 batch_result 转换为标准化执行记录 JSON 文件",
    )
    parser.add_argument("--input", "-i", required=True, help="batch_result JSON 文件路径")
    parser.add_argument("--output", "-o", required=True, help="标准化执行记录 JSON 输出路径")
    parser.add_argument("--task-id", help="任务 ID，可选，仅用于输出摘要信息")
    return parser.parse_args()


def main() -> None:
    """命令行入口：读取输入 JSON，生成标准化执行记录文件并打印摘要。"""

    args = _parse_args()
    batch_result = _load_json_file(args.input)
    summary = format_to_standard(batch_result, args.output, task_id=args.task_id)
    print(json.dumps(summary, ensure_ascii=False))


if __name__ == "__main__":
    main()