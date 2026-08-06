import argparse
import datetime
import json
import os
from typing import Any, Dict, List, Optional


def _now_str() -> str:
    return datetime.datetime.now().strftime("%Y-%m-%d %H:%M:%S")


def _safe_json_dumps(obj: Any) -> str:
    try:
        return json.dumps(obj, ensure_ascii=False, indent=2)
    except TypeError:
        # 回退为 str，避免因为不可序列化导致整个报告失败
        return str(obj)


def _is_case_success(case: Dict[str, Any]) -> bool:
    if isinstance(case.get("success"), bool):
        return bool(case["success"])
    status = case.get("status") or case.get("case_status")
    if status is None:
        # 如果有 error 字段且非空，默认认为失败
        if case.get("error") or case.get("error_message"):
            return False
        return True
    s = str(status).lower()
    if s in {"success", "succeeded", "done", "finished", "completed"}:
        return True
    if s in {"failed", "fail", "error", "exception"}:
        return False
    # 未知状态时，保守认为失败
    return False


def _get_first_nonempty(data: Dict[str, Any], keys: List[str], default: str = "") -> str:
    for k in keys:
        if k in data and data[k] not in (None, ""):
            return str(data[k])
    return default


def _escape_backticks(text: str) -> str:
    return text.replace("`", "\`")


def _format_step_records_table(records: List[Dict[str, Any]]) -> List[str]:
    lines: List[str] = []
    lines.append('<table header-row="true" header-col="false">')
    lines.append("  <tr>")
    lines.append("    <td>步骤</td>")
    lines.append("    <td>工具</td>")
    lines.append("    <td>状态</td>")
    lines.append("    <td>决策原因</td>")
    lines.append("    <td>输入</td>")
    lines.append("    <td>输出</td>")
    lines.append("  </tr>")

    for record in records:
        step = _get_first_nonempty(record, ["步骤名称", "step_name", "step", "步骤"])
        tool = _get_first_nonempty(record, ["工具名称", "tool_name", "tool"])
        # 优先取预期执行状态
        exp_status = record.get("expected_success", "").lower()
        if exp_status in ("success", "succeeded"):
            status = "✅ 成功"
        elif exp_status in ("failed", "fail", "error"):
            status = "❌ 失败"
        else:
            # 兜底取原有status字段
            raw_status = _get_first_nonempty(record, ["status", "状态"])
            status = raw_status if raw_status else "未知"
        reason = _get_first_nonempty(record, ["reason", "决策原因", "说明"])
        tool_input = _get_first_nonempty(record, ["工具执行输入", "input", "tool_input"])
        tool_output = _get_first_nonempty(record, ["工具执行输出", "output", "tool_output"])

        lines.append("  <tr>")
        lines.append(f"    <td>{step}</td>")
        lines.append(f"    <td>{tool}</td>")
        lines.append(f"    <td>{status}</td>")
        lines.append(f"    <td>{reason}</td>")
        if tool_input:
            lines.append(f"    <td>`{_escape_backticks(tool_input)}`</td>")
        else:
            lines.append("    <td></td>")
        if tool_output:
            lines.append(f"    <td>`{_escape_backticks(tool_output)}`</td>")
        else:
            lines.append("    <td></td>")
        lines.append("  </tr>")

    lines.append("</table>")
    return lines


def _format_case(index: int, case: Dict[str, Any]) -> List[str]:
    lines: List[str] = []
    lines.append("---")
    lines.append(f"### 用例 {index}")
    lines.append("")

    success = _is_case_success(case)
    status_text = "成功" if success else "失败"
    status_icon = "✅" if success else "❌"
    lines.append(f"{status_icon} **状态**: {status_text}")

    error_msg = case.get("error") or case.get("error_message") or ""
    if error_msg:
        lines.append(f"**错误信息**: {error_msg}")

    summary = case.get("summary") or ""
    if summary:
        lines.append(f"**执行结果摘要**: {summary}")

    lines.append("")

    test_case_str = case.get("test_case_str") or case.get("test_case") or ""
    if test_case_str:
        lines.append("#### 📥 测试用例")
        lines.append("```json")
        lines.append(str(test_case_str).strip())
        lines.append("```")
        lines.append("")

    final_plan = case.get("final_plan")
    if final_plan is not None:
        lines.append("#### ✅ 最终方案")
        lines.append("```json")
        lines.append(_safe_json_dumps(final_plan))
        lines.append("```")
        lines.append("")

    step_records = case.get("step_records") or []
    if step_records:
        lines.append("#### 🔧 执行记录")
        lines.extend(_format_step_records_table(step_records))
        lines.append("")

    token_usage = case.get("token_usage") or {}
    if token_usage:
        lines.append("#### 💰 Token 使用统计")
        prompt = token_usage.get("prompt_tokens", "未知")
        completion = token_usage.get("completion_tokens", "未知")
        total = token_usage.get("total_tokens", "未知")
        call_count = token_usage.get("call_count", token_usage.get("calls", "未知"))
        lines.append(f"- Prompt Tokens: {prompt}")
        lines.append(f"- Completion Tokens: {completion}")
        lines.append(f"- Total Tokens: {total}")
        lines.append(f"- 大模型调用次数: {call_count}")
        lines.append("")

    return lines


def generate_markdown(batch_result: Dict[str, Any], meta: Optional[Dict[str, Any]] = None) -> str:
    """将 batchResult(JSON) 转为 Markdown 文本。

    batchResult 结构示例：
    {
      "total_cases": 3,
      "success_cases": 1,
      "failed_cases": 2,
      "case_results": [ ... ]
    }
    """

    meta = meta or {}

    lines: List[str] = []
    title = meta.get("title", "批量测试执行报告")
    generated_at = meta.get("generated_at") or _now_str()

    lines.append(f"# {title}")
    lines.append(f"**生成时间**: {generated_at}")

    task_id = meta.get("task_id")
    if task_id:
        lines.append(f"**任务 ID**: {task_id}")

    env = meta.get("env")
    if env:
        lines.append(f"**环境**: {env}")

    status = meta.get("status")
    if status:
        status_map = {
            "success": "成功",
            "failed": "失败",
            "timeout": "超时",
        }
        status_str = str(status)
        status_label = status_map.get(status_str.lower(), status_str)
        lines.append(f"**任务状态**: {status_label}")

    error_summary = meta.get("error")
    if error_summary:
        lines.append(f"**任务错误**: {error_summary}")

    extra = meta.get("extra")
    if extra:
        lines.append(str(extra))

    lines.append("")

    case_results: List[Dict[str, Any]] = batch_result.get("case_results") or []
    total_cases = batch_result.get("total_cases")
    if total_cases is None:
        total_cases = len(case_results)

    success_cases = batch_result.get("success_cases")
    if success_cases is None and case_results:
        success_cases = sum(1 for c in case_results if _is_case_success(c))

    failed_cases = batch_result.get("failed_cases")
    if failed_cases is None and total_cases is not None and success_cases is not None:
        failed_cases = max(total_cases - success_cases, 0)

    if success_cases is None:
        success_cases = 0
    if failed_cases is None:
        failed_cases = max(total_cases - success_cases, 0)

    success_rate = 0.0
    if total_cases:
        success_rate = success_cases * 100.0 / float(total_cases)

    lines.append("## 📊 执行摘要")
    lines.append(f"- **总用例数**: {total_cases}")
    lines.append(f"- **成功**: {success_cases}")
    lines.append(f"- **失败**: {failed_cases}")
    lines.append(f"- **成功率**: {success_rate:.2f}%")
    lines.append("")

    lines.append("## 📋 用例执行详情")
    lines.append("")

    for idx, case in enumerate(case_results, start=1):
        lines.extend(_format_case(idx, case))

    return "\n".join(lines)


def _load_json_file(path: str) -> Dict[str, Any]:
    with open(path, "r", encoding="utf-8") as f:
        return json.load(f)


def _ensure_parent_dir(path: str) -> None:
    parent = os.path.dirname(path)
    if parent:
        os.makedirs(parent, exist_ok=True)


def main() -> None:
    parser = argparse.ArgumentParser(
        description="将 batchResult(JSON) 转为 Markdown 执行报告",
    )
    parser.add_argument("--input", "-i", required=True, help="batchResult JSON 文件路径")
    parser.add_argument("--output", "-o", required=True, help="输出 Markdown 报告路径")
    parser.add_argument("--task-id", help="任务 ID，可选")
    parser.add_argument("--env", help="环境信息，例如 prod/boe_xxx，可选")

    args = parser.parse_args()

    batch_result = _load_json_file(args.input)
    meta: Dict[str, Any] = {
        "task_id": args.task_id,
        "env": args.env,
    }

    md_text = generate_markdown(batch_result, meta)

    _ensure_parent_dir(args.output)
    with open(args.output, "w", encoding="utf-8") as f:
        f.write(md_text)


if __name__ == "__main__":
    main()
