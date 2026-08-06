#!/usr/bin/env python3
from __future__ import \
    annotations

import argparse
import base64
import json
import os
import sys
import time
import subprocess
from datetime import datetime, timezone
from pathlib import Path
from typing import Iterable

try:
    import requests
except ImportError as exc:
    print("Missing dependency: requests. Install with: python3 -m pip install requests", file=sys.stderr)
    raise


def _expand_path(path: str) -> Path:
    return Path(os.path.expanduser(path)).resolve()


def _default_cache_path() -> Path:
    env_path = os.environ.get("STREAMLOG_TOKEN_CACHE")
    if env_path:
        return _expand_path(env_path)
    return _expand_path("~/.cache/streamlog-token.json")


def _default_result_dir() -> Path:
    env_path = os.environ.get("STREAMLOG_RESULT_CACHE")
    if env_path:
        return _expand_path(env_path)
    return _expand_path("~/.cache/streamlog-results")


def _load_cache(path: Path) -> dict:
    if not path.exists():
        raise FileNotFoundError(str(path))
    with path.open("r", encoding="utf-8") as f:
        return json.load(f)


def _mask(value: str) -> str:
    if not value:
        return ""
    if len(value) <= 10:
        return "***"
    return f"{value[:4]}...{value[-4:]}"


def _jwt_expiry(token: str):
    parts = (token or "").split(".")
    if len(parts) != 3:
        return None
    try:
        payload = parts[1] + "=="
        data = json.loads(base64.urlsafe_b64decode(payload.encode("utf-8")).decode("utf-8"))
        exp = data.get("exp")
        if isinstance(exp, int):
            return exp
    except Exception:
        return None
    return None


def _write_cache_result(result_dir: Path, logid: str, data: dict) -> Path:
    result_dir.mkdir(parents=True, exist_ok=True)
    ts = datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ")
    safe_logid = "".join(ch for ch in logid if ch.isalnum() or ch in ("-", "_"))
    filename = f"{safe_logid}_{ts}.json" if safe_logid else f"trace_{ts}.json"
    out_path = result_dir / filename
    with out_path.open("w", encoding="utf-8") as f:
        json.dump(data, f, indent=2, ensure_ascii=False)
    return out_path


def _ts_to_iso(ts_raw: str | int | None) -> str:
    if ts_raw is None:
        return ""
    try:
        ts = int(ts_raw)
    except Exception:
        return str(ts_raw)
    if ts > 10**14:
        sec = ts / 1e6
    elif ts > 10**11:
        sec = ts / 1e3
    else:
        sec = ts
    return datetime.fromtimestamp(sec, tz=timezone.utc).isoformat()


def _normalize_trace_response(data: dict) -> list[dict]:
    """Flatten streamlog trace response to per-log json objects.

    Each record collapses `kv_list` into a key/value dict.
    """

    def _flatten_records(d: dict) -> Iterable[dict]:
        items = (d.get("data") or {}).get("items") or []
        for item in items:
            group = item.get("group") or {}
            group_psm = group.get("psm")
            for entry in item.get("value") or []:
                kv_list = entry.get("kv_list") or []
                kv = {it.get("key"): it.get("value") for it in kv_list if isinstance(it, dict)}
                ts_raw = kv.get("__timestamp")
                yield {
                    "id": entry.get("id") or kv.get("__id") or "",
                    "level": entry.get("level") or kv.get("_level") or "",
                    "timestamp_raw": ts_raw,
                    "timestamp": _ts_to_iso(ts_raw),
                    "psm": kv.get("_psm") or group_psm or "",
                    "spanid": kv.get("_spanid") or "",
                    "logid": kv.get("__logid") or "",
                    "msg": kv.get("_msg") or "",
                    "kv": kv,
                }

    return list(_flatten_records(data))


def _run_login_flow(cache_path: Path) -> dict | None:
    script_path = Path(__file__).resolve().parent / "login_and_cache_token.py"
    cmd = [sys.executable, str(script_path), "--cache-path", str(cache_path)]
    try:
        subprocess.run(cmd, check=False)
    except Exception as exc:
        print(f"Auto-login failed: {exc}", file=sys.stderr)
        return None
    if cache_path.exists():
        try:
            return _load_cache(cache_path)
        except Exception:
            return None
    return None


def main():
    parser = argparse.ArgumentParser(description="Query streamlog trace by log id.")
    parser.add_argument("--logid", required=True, help="log id to query")
    parser.add_argument("--scan-span-min", type=int, default=10, help="scan span in minutes")
    parser.add_argument(
        "--psm-list",
        default="",
        help="comma-separated psm list (e.g. a.b.c,d.e.f).",
    )
    parser.add_argument(
        "--vregion",
        default="China-Pay,China-Pay2,China-North",
        help="comma-separated vregion string (default China-Pay,China-Pay2,China-North).",
    )
    parser.add_argument(
        "--endpoint",
        default="https://logservice-zg.byted.org/streamlog/platform/microservice/v1/query/trace",
        help="trace query endpoint",
    )
    parser.add_argument(
        "--cache-path",
        default=str(_default_cache_path()),
        help="token cache file path",
    )
    parser.add_argument(
        "--cache-dir",
        default=str(_default_result_dir()),
        help="result cache directory",
    )
    parser.add_argument(
        "--auto-login",
        action="store_true",
        help="If token cache is missing/expired/unauthorized, open login flow to refresh token.",
    )
    parser.add_argument("--timeout", type=int, default=30, help="http timeout seconds")
    args = parser.parse_args()

    cache_path = _expand_path(args.cache_path)
    try:
        cache = _load_cache(cache_path)
    except FileNotFoundError:
        if args.auto_login:
            cache = _run_login_flow(cache_path)
            if not cache:
                print("Auto-login did not produce a token cache.", file=sys.stderr)
                return 2
        else:
            print(f"Token cache not found: {cache_path}. Run login_and_cache_token.py first.", file=sys.stderr)
            return 2

    jwt = cache.get("jwt_token") or ""
    custom_identity = cache.get("custom_identity") or ""
    user_agent = cache.get("user_agent") or "Mozilla/5.0"

    exp = _jwt_expiry(jwt)
    if exp and exp < int(time.time()):
        exp_dt = datetime.fromtimestamp(exp, tz=timezone.utc).isoformat()
        if args.auto_login:
            cache = _run_login_flow(cache_path)
            if not cache:
                print(f"Token expired at {exp_dt}. Auto-login failed.", file=sys.stderr)
                return 3
            jwt = cache.get("jwt_token") or ""
            custom_identity = cache.get("custom_identity") or ""
            user_agent = cache.get("user_agent") or "Mozilla/5.0"
        else:
            print(f"Token expired at {exp_dt}. Please re-login to refresh.", file=sys.stderr)
            return 3

    headers = {
        "Accept": "application/json, text/plain, */*",
        "Content-Type": "application/json",
        "Origin": "https://cloud.bytedance.net",
        "Referer": "https://cloud.bytedance.net/",
        "User-Agent": user_agent,
        "X-Jwt-Token": jwt,
    }
    if custom_identity:
        headers["x-custom-identity"] = custom_identity

    payload = {
        "logid": args.logid,
        "scan_span_in_min": args.scan_span_min,
        "vregion": args.vregion,
    }
    if args.psm_list:
        payload["psm_list"] = [s for s in args.psm_list.split(",") if s]

    resp = requests.post(args.endpoint, headers=headers, json=payload, timeout=args.timeout)
    if resp.status_code in (401, 403):
        if args.auto_login:
            cache = _run_login_flow(cache_path)
            if not cache:
                print("Unauthorized. Auto-login failed.", file=sys.stderr)
                return 4
            jwt = cache.get("jwt_token") or ""
            custom_identity = cache.get("custom_identity") or ""
            user_agent = cache.get("user_agent") or "Mozilla/5.0"
            headers["X-Jwt-Token"] = jwt
            if custom_identity:
                headers["x-custom-identity"] = custom_identity
            resp = requests.post(args.endpoint, headers=headers, json=payload, timeout=args.timeout)
            if resp.status_code in (401, 403):
                print("Unauthorized after auto-login. Please retry manually.", file=sys.stderr)
                return 4
        else:
            print("Unauthorized. Please re-login to refresh token.", file=sys.stderr)
            return 4
    if not resp.ok:
        print(f"Request failed: {resp.status_code} {resp.text[:500]}", file=sys.stderr)
        return 5

    data = resp.json()
    cache_out = _write_cache_result(_expand_path(args.cache_dir), args.logid, data)
    print(f"Cached response to {cache_out}")

    # 固定行为：
    # - 原始响应永远缓存到 cache-dir
    # - 控制台默认输出归一化后的日志 jsonl（每条日志一行；kv_list 已折叠为 kv 字典）
    for record in _normalize_trace_response(data):
        print(json.dumps(record, ensure_ascii=False))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
