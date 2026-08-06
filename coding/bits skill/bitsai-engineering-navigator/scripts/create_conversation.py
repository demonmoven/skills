#!/usr/bin/env python3
# -*- coding: utf-8 -*-

"""Create a BitsAI Q&A system conversation.

This script wraps the HTTP API:
  POST https://bitsai.bytedance.net/api/quoraid/v1/conversations

It prints the `conversation_id` to stdout on success.

Security notes:
- Do NOT hardcode JWT/cookies in code.
- Read tokens from a local file with proper permission.
"""

from __future__ import annotations

import argparse
import json
import sys
from typing import Any, Dict, Optional

from shared import build_headers, env, http_post_json

DEFAULT_ENDPOINT = "https://bitsai.bytedance.net/api/quoraid/v1/conversations"


def _mask_secret(value: str, keep: int = 6) -> str:
    if not value:
        return ""
    if len(value) <= keep * 2:
        return "***"
    return f"{value[:keep]}***{value[-keep:]}"


def create_conversation(
    app_id: str,
    jwt_token: str,
    timeout_s: int,
    insecure: bool,
) -> Dict[str, Any]:
    headers = build_headers(jwt_token)
    payload = {
        "app_id": app_id,
    }
    status, body = http_post_json(
        url=DEFAULT_ENDPOINT,
        payload=payload,
        headers=headers,
        timeout_s=timeout_s,
        insecure=insecure,
    )

    if not body:
        raise RuntimeError(f"HTTP {status} with empty body")

    try:
        obj = json.loads(body.decode("utf-8"))
    except Exception as e:
        raise RuntimeError(
            f"HTTP {status} but response is not JSON: {e}; body={body[:200]!r}"
        )

    # Attach status for debugging but keep API payload intact.
    if isinstance(obj, dict):
        obj.setdefault("_http_status", status)
    return obj


def _build_arg_parser() -> argparse.ArgumentParser:
    p = argparse.ArgumentParser(
        description="调用 BitsAI conversations API 创建问答系统会话，并输出 conversation_id",
        formatter_class=argparse.ArgumentDefaultsHelpFormatter,
    )
    p.add_argument(
        "--timeout",
        type=int,
        default=int(env("BITS_AI_TIMEOUT") or 20),
        help="请求超时时间（秒）",
    )
    p.add_argument(
        "--insecure",
        action="store_true",
        help="跳过 TLS 证书校验（仅排查网络问题时使用）",
    )
    return p


def main(argv: Optional[list[str]] = None) -> int:
    args = _build_arg_parser().parse_args(argv)

    jwt_token = env("X_JWT_TOKEN")
    if not jwt_token:
        print("缺少环境变量 X_JWT_TOKEN", file=sys.stderr)
        return 2

    try:
        resp = create_conversation(
            app_id="5c0fcd6b-f9e6-4a36-8098-4791c3e2e0d3",
            jwt_token=jwt_token,
            timeout_s=args.timeout,
            insecure=args.insecure,
        )
    except Exception as e:
        # Never print full secrets. Only show whether headers were provided.
        print(f"请求失败：{e}", file=sys.stderr)
        return 1

    conversation_id = None
    if isinstance(resp, dict):
        data = resp.get("data")
        if isinstance(data, dict):
            conversation_id = data.get("conversation_id")

    if not conversation_id:
        print("未在返回值中找到 data.conversation_id：", file=sys.stderr)
        print(json.dumps(resp, ensure_ascii=False, indent=2), file=sys.stderr)
        return 1

    print(conversation_id)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
