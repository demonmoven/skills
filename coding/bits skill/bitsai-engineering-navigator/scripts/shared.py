#!/usr/bin/env python3
# -*- coding: utf-8 -*-

"""Shared helpers for bitsai-engineering-navigator scripts."""

from __future__ import annotations

import base64
import datetime
import json
import os
import ssl
import urllib.error
import urllib.request
from typing import Any, Dict, Optional, Tuple

QUORAID_ENTRY_POINT_HEADER = "Quoraid-Entry-Point"
QUORAID_ENTRY_POINT_VALUE = "agent_skill"


def env(name: str) -> Optional[str]:
    v = os.environ.get(name)
    return v if v else None


def _base64url_decode(s: str) -> bytes:
    s = s.strip()
    pad = (-len(s)) % 4
    if pad:
        s += "=" * pad
    return base64.urlsafe_b64decode(s.encode("utf-8"))


def parse_jwt_payload(jwt_token: str) -> Dict[str, Any]:
    """Parse JWT payload (middle segment) without verifying signature."""

    if not isinstance(jwt_token, str) or not jwt_token:
        raise ValueError("JWT 为空")
    parts = jwt_token.split(".")
    if len(parts) < 2:
        raise ValueError("JWT 格式非法")
    payload_b64 = parts[1]
    try:
        payload_raw = _base64url_decode(payload_b64)
        obj = json.loads(payload_raw.decode("utf-8"))
    except Exception as e:
        raise ValueError(f"JWT payload 解析失败：{e}")
    if not isinstance(obj, dict):
        raise ValueError("JWT payload 不是对象")
    return obj


def jwt_expire_time(jwt_token: str) -> datetime.datetime:
    payload = parse_jwt_payload(jwt_token)
    exp = payload.get("exp")
    if not isinstance(exp, (int, float)):
        raise ValueError("JWT payload 缺少 exp 字段")
    return datetime.datetime.fromtimestamp(int(exp))


def is_jwt_expired(jwt_token: str, *, skew_seconds: int = 60) -> bool:
    """Return True if token is expired (with a safety skew)."""

    exp_time = jwt_expire_time(jwt_token)
    now = datetime.datetime.now()
    return (exp_time - now).total_seconds() <= float(skew_seconds)


def build_headers(jwt_token: str, *, accept_sse: bool = False) -> Dict[str, str]:
    headers: Dict[str, str] = {
        "Content-Type": "application/json",
        "X-JWT-TOKEN": jwt_token,
        QUORAID_ENTRY_POINT_HEADER: QUORAID_ENTRY_POINT_VALUE,
    }
    if accept_sse:
        headers["Accept"] = "text/event-stream"
    return headers


def build_ssl_context(insecure: bool) -> Optional[ssl.SSLContext]:
    if not insecure:
        return None
    ctx = ssl.create_default_context()
    ctx.check_hostname = False
    ctx.verify_mode = ssl.CERT_NONE
    return ctx


def http_post_json(
    *,
    url: str,
    payload: Dict[str, Any],
    headers: Dict[str, str],
    timeout_s: int,
    insecure: bool,
) -> Tuple[int, bytes]:
    data = json.dumps(payload, ensure_ascii=False).encode("utf-8")
    req = urllib.request.Request(url=url, data=data, headers=headers, method="POST")

    context = build_ssl_context(insecure)

    try:
        with urllib.request.urlopen(req, timeout=timeout_s, context=context) as resp:
            return int(getattr(resp, "status", 200)), resp.read()
    except urllib.error.HTTPError as e:
        # Read body for error details.
        body = b""
        try:
            body = e.read()
        except Exception:
            body = b""
        return int(getattr(e, "code", 0) or 0), body
