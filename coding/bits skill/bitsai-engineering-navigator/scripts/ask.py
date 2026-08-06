#!/usr/bin/env python3
# -*- coding: utf-8 -*-

"""Ask BitsAI in an existing conversation (SSE streaming).

This script wraps the HTTP API:
  POST https://bitsai.bytedance.net/api/quoraid/v1/conversations/{conversation_id}/messages/stream

It consumes the SSE stream and returns ONLY the final answer:
- Keep the last event whose type is `chunk`, then output payload.data.
- If an event whose type is `error` appears, output formatted error info.

Security notes:
- Do NOT hardcode JWT/cookies in code.
- Read tokens from a local file with proper permission.
"""

from __future__ import annotations

import argparse
import json
import re
import sys
import urllib.error
import urllib.request
from dataclasses import dataclass
from typing import Any, Dict, Iterable, List, Optional, Tuple

from shared import build_headers, build_ssl_context, env

DEFAULT_APP_ID = "5c0fcd6b-f9e6-4a36-8098-4791c3e2e0d3"
DEFAULT_MODEL_ID = "default"
DEFAULT_BASE = "https://bitsai.bytedance.net/api/quoraid/v1"

# Match tags like:
# - [citation: 2]
# - [citation:2]
# - [citation:    12]
# (Only single number; no comma list)
_CITATION_TAG_RE = re.compile(r"\[citation:\s*\d+\s*\]", re.IGNORECASE)


# <bits-context data-type='project' data-name='bits.ai.quoraid' data-resource-type='TCE' data-id='bits.ai.quoraid' data-url=''/>


@dataclass
class _SSEEvent:
    event: Optional[str]
    data: str


def _iter_sse_events(resp) -> Iterable[_SSEEvent]:
    """Parse SSE stream from an HTTPResponse-like object.

    Minimal SSE parser:
    - Collect `event:` and all `data:` lines until a blank line, then yield.
    - Ignore other fields.
    """

    event_type: Optional[str] = None
    data_lines: List[str] = []

    while True:
        line_b = resp.readline()
        if not line_b:
            # EOF: flush buffered event if any.
            if event_type is not None or data_lines:
                yield _SSEEvent(event=event_type, data="\n".join(data_lines))
            return

        try:
            line = line_b.decode("utf-8", errors="replace")
        except Exception:
            line = str(line_b)

        # SSE uses \n as line separator; keep it simple.
        line = line.rstrip("\r\n")

        if line == "":
            if event_type is not None or data_lines:
                yield _SSEEvent(event=event_type, data="\n".join(data_lines))
            event_type = None
            data_lines = []
            continue

        if line.startswith(":"):
            # Comment line.
            continue

        if line.startswith("event:"):
            event_type = line[len("event:") :].strip() or None
            continue

        if line.startswith("data:"):
            data_lines.append(line[len("data:") :].lstrip())
            continue

        # Ignore id:, retry:, and unknown fields.


def _maybe_json(s: str) -> Any:
    s = s.strip()
    if not s:
        return None
    try:
        return json.loads(s)
    except Exception:
        return None


def _extract_event_type_and_payload(evt: _SSEEvent) -> Tuple[Optional[str], Any]:
    """Return (event_type, payload).

    Supports both:
    - SSE `event:` field + JSON in `data:`
    - JSON wrapper style in `data:` like {"type":"chunk","data":{...}}
    """

    payload = _maybe_json(evt.data)
    if evt.event:
        return evt.event, payload

    if isinstance(payload, dict):
        t = payload.get("type") or payload.get("Type")
        if isinstance(t, str) and t:
            return t, payload.get("data") if "data" in payload else payload.get("Data")

    return None, payload


def _format_error_payload(p: Any) -> str:
    if not isinstance(p, dict):
        return f"code=? stage=? message=? stage_message=? session_id=? role=? conversation_id=? (raw={p!r})"

    code = p.get("code") or p.get("Code") or ""
    stage = p.get("stage") or p.get("Stage") or ""
    message = p.get("message") or p.get("Message") or ""
    stage_message = p.get("stage_message") or p.get("StageMessage")
    session_id = p.get("session_id") or p.get("SessionID") or ""
    role = p.get("role") or p.get("Role") or ""
    conversation_id = p.get("conversation_id") or p.get("ConversationID") or ""

    sm = "" if stage_message is None else str(stage_message)
    return (
        f"code={code} stage={stage} message={message} "
        f"stage_message={sm} session_id={session_id} role={role} conversation_id={conversation_id}"
    )


def _strip_citation_tags(text: str) -> str:
    # Remove patterns like "[citation: 2]".
    cleaned = _CITATION_TAG_RE.sub("", text)
    # Avoid leaving awkward extra spaces; keep newlines intact.
    cleaned = re.sub(r"[ \t]{2,}", " ", cleaned)
    cleaned = re.sub(r" ?\n ?", "\n", cleaned)
    return cleaned.strip()


def ask(
    *,
    conversation_id: str,
    input_text: str,
    app_id: str,
    model_id: str,
    jwt_token: str,
    timeout_s: int,
    insecure: bool,
    enable_full_resp: bool,
) -> str:
    if not conversation_id:
        raise ValueError("conversation_id 不能为空")
    if not input_text:
        raise ValueError("input_text 不能为空")
    if not jwt_token:
        raise ValueError("缺少 JWT 令牌")

    url = f"{DEFAULT_BASE}/conversations/{conversation_id}/messages/stream"
    headers = build_headers(jwt_token, accept_sse=True)
    payload: Dict[str, Any] = {
        "input": input_text,
        "app_id": app_id,
        "context": {
            "enable_web_search": False,
            "enable_bytedance_knowledge": True,
            "enable_thinking": False,
            "bits_spaces_list": [],
            "enable_agent_mode": True,
            "cmds": [],
            "attachments": [],
            "tools": [
                "lark_search",
                "knowledge_retrieval",
                "meego_search",
                "bitsdata_retrieval",
                "codebase_search",
                "appcenter_retrieval",
                "test_case_search",
            ],
        },
        "model_id": model_id,
        "conversation_id": conversation_id,
        "enable_full_resp": bool(enable_full_resp),
    }

    data = json.dumps(payload, ensure_ascii=False).encode("utf-8")
    req = urllib.request.Request(url=url, data=data, headers=headers, method="POST")

    ctx = build_ssl_context(insecure)

    last_chunk_text: Optional[str] = None
    try:
        with urllib.request.urlopen(req, timeout=timeout_s, context=ctx) as resp:
            for sse_evt in _iter_sse_events(resp):
                evt_type, evt_payload = _extract_event_type_and_payload(sse_evt)

                if evt_type == "error":
                    return _format_error_payload(evt_payload)

                if evt_type != "chunk":
                    continue

                # chunk payload expected to be a dict like {"data": "answer", ...}
                if isinstance(evt_payload, dict):
                    ans = evt_payload.get("data")
                    if isinstance(ans, str):
                        last_chunk_text = ans
                elif isinstance(evt_payload, str):
                    # Some services may emit a raw string.
                    last_chunk_text = evt_payload

    except urllib.error.HTTPError as e:
        body = b""
        try:
            body = e.read()
        except Exception:
            body = b""
        detail = body.decode("utf-8", errors="replace")[:500]
        raise RuntimeError(f"HTTP {getattr(e, 'code', '')}: {detail}")

    if last_chunk_text is None:
        raise RuntimeError("未在 SSE 流中收到 chunk 事件")
    return _strip_citation_tags(last_chunk_text)


def _build_arg_parser() -> argparse.ArgumentParser:
    p = argparse.ArgumentParser(
        description="调用 BitsAI SSE messages/stream API，并输出最终回答（最后一个 chunk.data）",
        formatter_class=argparse.ArgumentDefaultsHelpFormatter,
    )
    p.add_argument(
        "conversation_id",
        help="create_conversation.py 输出的 conversation_id",
    )
    p.add_argument(
        "input",
        nargs="?",
        default=None,
        help="问题内容（未提供则从 stdin 读取）",
    )
    p.add_argument(
        "--timeout",
        type=int,
        default=int(env("BITS_AI_TIMEOUT") or 120),
        help="请求超时时间（秒）",
    )
    p.add_argument(
        "--insecure",
        action="store_true",
        help="跳过 TLS 证书校验（仅排查网络问题时使用）",
    )
    p.add_argument(
        "--app-id",
        default=DEFAULT_APP_ID,
        help="BitsAI app_id",
    )
    p.add_argument(
        "--model-id",
        default=DEFAULT_MODEL_ID,
        help="模型ID",
    )
    p.add_argument(
        "--no-full-resp",
        action="store_true",
        help="将 enable_full_resp 设为 false",
    )
    return p


def main(argv: Optional[List[str]] = None) -> int:
    args = _build_arg_parser().parse_args(argv)

    input_text = args.input
    if input_text is None:
        input_text = sys.stdin.read().strip()

    jwt_token = env("X_JWT_TOKEN")
    if not jwt_token:
        print("缺少环境变量 X_JWT_TOKEN", file=sys.stderr)
        return 2

    try:
        ans = ask(
            conversation_id=args.conversation_id,
            input_text=input_text,
            app_id=args.app_id,
            model_id=args.model_id,
            jwt_token=jwt_token,
            timeout_s=args.timeout,
            insecure=args.insecure,
            enable_full_resp=not args.no_full_resp,
        )
    except Exception as e:
        print(f"请求失败：{e}", file=sys.stderr)
        return 1

    print(ans)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
