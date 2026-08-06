#!/usr/bin/env python3
"""
触发 qagents_next 用例生成 workflow 脚本（同步 / 异步双模式）

模式说明：
- sync（默认）：直接调用 RunWorkflowSync 接口，等待同步返回；不再调用 GetWorkflowTaskDetail
- async：调用 RunWorkflow 异步接口取得 workflowTaskID，再按固定间隔轮询 GetWorkflowTaskDetail
        直到 workflowTask.status == 3 后取出 end_0 节点 output

无论同步还是异步，最终标准输出都会包含一段：
    BITS_CASE_RESULT={"bits_case_url":"<...>"}
便于上层稳定抓取。

行为约定（共同）：
- 触发接口在脚本运行期内**仅会被调用一次**
- 不会自行重试触发，超时或失败由调用方决定是否重新执行脚本

示例：
    # 默认同步
    python3 scripts/trigger_workflow.py \
        --prd-link "https://xxxxx/docx/yyyyyy" \
        --prd-name "支付补丁需求 PRD"

    # 显式异步
    python3 scripts/trigger_workflow.py --mode async \
        --prd-link "https://xxxxx/docx/yyyyyy" \
        --prd-name "支付补丁需求 PRD"
"""

import argparse
import json
import sys
import time
from urllib import request as urlrequest, error as urlerror

DEFAULT_SYNC_API_URL = "https://qagents-next-api.bytedance.net/apin/qagents_next/workflow/RunWorkflowSync"
DEFAULT_ASYNC_API_URL = "https://qagents-next-api.bytedance.net/apin/qagents_next/workflow/RunWorkflow"
DEFAULT_DETAIL_URL = "https://qagents-next-api.bytedance.net/apin/qagents_next/workflow/GetWorkflowTaskDetail"
DEFAULT_WORKFLOW_ID = "10670"
DEFAULT_AUTH_TOKEN = "AHhnnlOprYywGXXY"
DEFAULT_TIMEOUT = 600  # seconds，单次 HTTP 请求超时
DEFAULT_POLL_INTERVAL = 30  # seconds，异步模式轮询间隔
DEFAULT_MAX_POLLS = 120  # 异步模式最大轮询次数（约 1 小时，30s × 120）
DEFAULT_MODE = "sync"


# ---------------------------------------------------------------------------
# 通用工具
# ---------------------------------------------------------------------------

def _http_post_json(url: str, payload: dict, auth_token: str, timeout: int) -> dict:
    """通用的 POST JSON 调用，返回 {status_code, response}。"""
    body = json.dumps(payload, ensure_ascii=False).encode("utf-8")
    req = urlrequest.Request(
        url,
        data=body,
        method="POST",
        headers={
            "Content-Type": "application/json",
            "Authorization": f"Bearer {auth_token}",
        },
    )

    try:
        with urlrequest.urlopen(req, timeout=timeout) as resp:
            status = resp.status
            raw = resp.read().decode("utf-8", errors="replace")
    except urlerror.HTTPError as e:
        status = e.code
        raw = e.read().decode("utf-8", errors="replace") if e.fp else str(e)
    except urlerror.URLError as e:
        return {"status_code": None, "response": {"error": f"URLError: {e.reason}"}}

    try:
        parsed = json.loads(raw)
    except json.JSONDecodeError:
        parsed = {"raw": raw}

    return {"status_code": status, "response": parsed}


def _build_payload(prd_link: str, prd_name: str, workflow_id: str) -> dict:
    """组装触发接口请求体（sync / async 共用同一格式）。"""
    if not prd_link:
        raise ValueError("prd_link 不能为空")
    if not prd_name:
        raise ValueError("prd_name 不能为空（应为 prd_link 对应的文档标题）")
    inner_input = json.dumps(
        {"prd_link": prd_link, "prd_name": prd_name},
        ensure_ascii=False,
    )
    return {"workflowID": workflow_id, "input": inner_input}


def _normalize_output(raw_output):
    """output 可能是 dict 或被再次序列化的 JSON 字符串，统一还原成对象。"""
    if raw_output is None:
        return None
    if isinstance(raw_output, dict):
        return raw_output
    if isinstance(raw_output, str):
        try:
            return json.loads(raw_output)
        except json.JSONDecodeError:
            return raw_output
    return raw_output


def _deep_find_bits_case_url(obj):
    """递归查找 obj 中第一个非空的 bits_case_url 字段值。"""
    if obj is None:
        return None
    if isinstance(obj, dict):
        url = obj.get("bits_case_url")
        if isinstance(url, str) and url.strip():
            return url.strip()
        for v in obj.values():
            found = _deep_find_bits_case_url(v)
            if found:
                return found
        return None
    if isinstance(obj, list):
        for v in obj:
            found = _deep_find_bits_case_url(v)
            if found:
                return found
        return None
    if isinstance(obj, str):
        # 兼容 output 本身是 JSON 字符串
        try:
            parsed = json.loads(obj)
        except (json.JSONDecodeError, ValueError):
            return None
        return _deep_find_bits_case_url(parsed)
    return None


# ---------------------------------------------------------------------------
# 触发：同步 / 异步
# ---------------------------------------------------------------------------

def trigger_workflow_sync(prd_link: str,
                          prd_name: str,
                          workflow_id: str = DEFAULT_WORKFLOW_ID,
                          api_url: str = DEFAULT_SYNC_API_URL,
                          auth_token: str = DEFAULT_AUTH_TOKEN,
                          timeout: int = DEFAULT_TIMEOUT) -> dict:
    """同步触发 RunWorkflowSync。**只调用一次**，不轮询。"""
    payload = _build_payload(prd_link, prd_name, workflow_id)
    result = _http_post_json(api_url, payload, auth_token, timeout)
    return {
        "status_code": result["status_code"],
        "request": {
            "url": api_url,
            "workflowID": workflow_id,
            "prd_link": prd_link,
            "prd_name": prd_name,
        },
        "response": result["response"],
    }


def trigger_workflow_async(prd_link: str,
                           prd_name: str,
                           workflow_id: str = DEFAULT_WORKFLOW_ID,
                           api_url: str = DEFAULT_ASYNC_API_URL,
                           auth_token: str = DEFAULT_AUTH_TOKEN,
                           timeout: int = DEFAULT_TIMEOUT) -> dict:
    """异步触发 RunWorkflow，返回应包含 workflowTaskID。**只调用一次**。"""
    payload = _build_payload(prd_link, prd_name, workflow_id)
    result = _http_post_json(api_url, payload, auth_token, timeout)
    return {
        "status_code": result["status_code"],
        "request": {
            "url": api_url,
            "workflowID": workflow_id,
            "prd_link": prd_link,
            "prd_name": prd_name,
        },
        "response": result["response"],
    }


# ---------------------------------------------------------------------------
# 异步轮询相关
# ---------------------------------------------------------------------------

def _extract_workflow_task_id(trigger_resp: dict):
    """从异步触发响应中提取 workflowTaskID。"""
    if not isinstance(trigger_resp, dict):
        return None
    resp = trigger_resp.get("response")
    if isinstance(resp, dict):
        for key in ("workflowTaskID", "workflow_task_id", "WorkflowTaskID"):
            if key in resp and resp[key] not in (None, "", 0):
                return resp[key]
        data = resp.get("data")
        if isinstance(data, dict):
            for key in ("workflowTaskID", "workflow_task_id", "WorkflowTaskID"):
                if key in data and data[key] not in (None, "", 0):
                    return data[key]
    return None


def _extract_workflow_task(detail_resp: dict):
    """从 GetWorkflowTaskDetail 响应中定位 workflowTask 对象（第一层）。"""
    if not isinstance(detail_resp, dict):
        return None
    resp = detail_resp.get("response")
    if not isinstance(resp, dict):
        return None
    if isinstance(resp.get("workflowTask"), dict):
        return resp["workflowTask"]
    data = resp.get("data")
    if isinstance(data, dict) and isinstance(data.get("workflowTask"), dict):
        return data["workflowTask"]
    return None


def _extract_end_node_output(workflow_task: dict):
    """在 workflowTask 中查找 end_0 节点并取出 output。兼容多种结构。"""
    if not isinstance(workflow_task, dict):
        return None

    target_keys = ("end_0",)

    # 形态 1：直接挂在 workflowTask 上
    for k in target_keys:
        if k in workflow_task:
            node = workflow_task[k]
            if isinstance(node, dict):
                if "output" in node:
                    return _normalize_output(node["output"])
            else:
                return _normalize_output(node)

    # 形态 2：nodeOutputs / outputs / nodes 字段
    for container_key in ("nodeOutputs", "node_outputs", "outputs", "nodes", "Nodes"):
        container = workflow_task.get(container_key)
        if container is None:
            continue
        if isinstance(container, dict):
            for k in target_keys:
                if k in container:
                    val = container[k]
                    if isinstance(val, dict) and "output" in val:
                        return _normalize_output(val["output"])
                    return _normalize_output(val)
        if isinstance(container, list):
            for item in container:
                if not isinstance(item, dict):
                    continue
                node_id = (
                    item.get("nodeID")
                    or item.get("node_id")
                    or item.get("nodeId")
                    or item.get("key")
                    or item.get("id")
                    or item.get("name")
                )
                if node_id in target_keys:
                    if "output" in item:
                        return _normalize_output(item["output"])
                    return _normalize_output(item)

    return None


def poll_task_detail(workflow_task_id,
                     detail_url: str = DEFAULT_DETAIL_URL,
                     auth_token: str = DEFAULT_AUTH_TOKEN,
                     interval: int = DEFAULT_POLL_INTERVAL,
                     max_polls: int = DEFAULT_MAX_POLLS,
                     timeout: int = DEFAULT_TIMEOUT) -> dict:
    """按固定间隔轮询任务详情，直到 workflowTask.status == 3 或达到上限。"""
    payload = {"workflowTaskID": str(workflow_task_id)}
    last_snapshot = None
    for attempt in range(1, max_polls + 1):
        result = _http_post_json(detail_url, payload, auth_token, timeout)
        snapshot = {
            "attempt": attempt,
            "status_code": result["status_code"],
            "response": result["response"],
        }
        last_snapshot = snapshot

        workflow_task = _extract_workflow_task(snapshot)
        status = None
        if isinstance(workflow_task, dict):
            status = workflow_task.get("status")

        if status == 3:
            end_output = _extract_end_node_output(workflow_task)
            return {
                "found": True,
                "attempts": attempt,
                "status": status,
                "end_node_output": end_output,
                "last_response": snapshot,
            }

        if attempt < max_polls:
            time.sleep(interval)

    return {
        "found": False,
        "attempts": max_polls,
        "status": None,
        "end_node_output": None,
        "last_response": last_snapshot,
    }


# ---------------------------------------------------------------------------
# 主流程
# ---------------------------------------------------------------------------

def _print_final(final: dict, bits_case_url):
    """统一输出：完整 JSON + 末尾追加 BITS_CASE_RESULT={"bits_case_url":"..."}。"""
    print(json.dumps(final, ensure_ascii=False, indent=2))
    result_obj = {"bits_case_url": bits_case_url or ""}
    print("\nBITS_CASE_RESULT=" + json.dumps(result_obj, ensure_ascii=False))


def run_sync(args) -> int:
    trigger_result = trigger_workflow_sync(
        prd_link=args.prd_link,
        prd_name=args.prd_name,
        workflow_id=args.workflow_id,
        api_url=args.api_url or DEFAULT_SYNC_API_URL,
        auth_token=args.auth_token,
        timeout=args.timeout,
    )

    status = trigger_result.get("status_code")
    if not (isinstance(status, int) and 200 <= status < 300):
        _print_final(
            {
                "mode": "sync",
                "stage": "trigger_failed",
                "trigger_result": trigger_result,
                "error": "RunWorkflowSync 接口非 2xx，终止",
            },
            None,
        )
        return 1

    # sync 接口的产物结构与 end_0.output 一致；递归查找 bits_case_url
    bits_case_url = _deep_find_bits_case_url(trigger_result.get("response"))

    _print_final(
        {
            "mode": "sync",
            "stage": "completed" if bits_case_url else "completed_without_url",
            "trigger_result": trigger_result,
            "bits_case_url": bits_case_url,
        },
        bits_case_url,
    )
    return 0 if bits_case_url else 3


def run_async(args) -> int:
    trigger_result = trigger_workflow_async(
        prd_link=args.prd_link,
        prd_name=args.prd_name,
        workflow_id=args.workflow_id,
        api_url=args.api_url or DEFAULT_ASYNC_API_URL,
        auth_token=args.auth_token,
        timeout=args.timeout,
    )

    status = trigger_result.get("status_code")
    if not (isinstance(status, int) and 200 <= status < 300):
        _print_final(
            {
                "mode": "async",
                "stage": "trigger_failed",
                "trigger_result": trigger_result,
                "error": "RunWorkflow 异步触发接口非 2xx，终止",
            },
            None,
        )
        return 1

    workflow_task_id = _extract_workflow_task_id(trigger_result)
    if not workflow_task_id:
        _print_final(
            {
                "mode": "async",
                "stage": "trigger_no_task_id",
                "trigger_result": trigger_result,
                "error": "未能从触发响应中解析出 workflowTaskID",
            },
            None,
        )
        return 1

    poll_result = poll_task_detail(
        workflow_task_id=workflow_task_id,
        detail_url=args.detail_url,
        auth_token=args.auth_token,
        interval=args.poll_interval,
        max_polls=args.max_polls,
        timeout=args.timeout,
    )

    bits_case_url = _deep_find_bits_case_url(poll_result.get("end_node_output"))

    final = {
        "mode": "async",
        "stage": "completed" if poll_result["found"] else "timeout",
        "workflow_task_id": workflow_task_id,
        "trigger_result": trigger_result,
        "poll_attempts": poll_result["attempts"],
        "status": poll_result.get("status"),
        "end_node_output": poll_result.get("end_node_output"),
        "last_detail_response": poll_result["last_response"],
        "bits_case_url": bits_case_url,
    }
    _print_final(final, bits_case_url)

    if not poll_result["found"]:
        return 2
    return 0 if bits_case_url else 3


def main():
    parser = argparse.ArgumentParser(
        description="触发 qagents_next 用例生成 workflow（同步 / 异步双模式），最终输出 {\"bits_case_url\":\"...\"}"
    )
    parser.add_argument("--mode", choices=("sync", "async"), default=DEFAULT_MODE,
                        help="调用模式：sync(默认，调 RunWorkflowSync) 或 async(调 RunWorkflow + 轮询)")
    parser.add_argument("--prd-link", required=True,
                        help="PRD 文档链接（必填）")
    parser.add_argument("--prd-name", required=True,
                        help="PRD 文档标题名称（必填，应为 --prd-link 对应文档的标题）")
    parser.add_argument("--workflow-id", default=DEFAULT_WORKFLOW_ID,
                        help=f"workflow ID，默认 {DEFAULT_WORKFLOW_ID}")
    parser.add_argument("--api-url", default=None,
                        help="触发接口地址；不传则按 --mode 自动选择 RunWorkflowSync 或 RunWorkflow")
    parser.add_argument("--detail-url", default=DEFAULT_DETAIL_URL,
                        help="GetWorkflowTaskDetail 接口地址（仅 async 模式使用）")
    parser.add_argument("--auth-token", default=DEFAULT_AUTH_TOKEN,
                        help="Bearer Token / Access Key")
    parser.add_argument("--timeout", type=int, default=DEFAULT_TIMEOUT,
                        help=f"单次 HTTP 超时秒数，默认 {DEFAULT_TIMEOUT}s")
    parser.add_argument("--poll-interval", type=int, default=DEFAULT_POLL_INTERVAL,
                        help=f"异步轮询间隔秒数，默认 {DEFAULT_POLL_INTERVAL}s（每 30 秒一次）")
    parser.add_argument("--max-polls", type=int, default=DEFAULT_MAX_POLLS,
                        help=f"异步最大轮询次数，默认 {DEFAULT_MAX_POLLS} 次")
    args = parser.parse_args()

    if args.mode == "sync":
        sys.exit(run_sync(args))
    else:
        sys.exit(run_async(args))


if __name__ == "__main__":
    main()
