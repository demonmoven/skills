# -*- coding: utf-8 -*-
"""
Meego 鉴权预检原子(auto-develop Step 0.3,2026-06-03 新增)。

定位:在主流程进入 Step 1(PRD↔Meego 解析)**之前**完成 Meego 登录,
避免"解析到一半发现没登录 → 中段断点 → 回头扫码"的劣体验。

设计要点:
1. 默认走 device flow 扫码(沙箱无本地 keychain,SSO/cookie 路径不可达)
2. 已登录态秒过 — `bytedcli meego login --check` 命中则直接 return success
3. 未登录 / token 过期 → 拉 QR + 上传 + 输出 handshake 上下文,**不阻塞**,
   由路由层负责把 QR + URL + user_code 推给用户,等用户「继续」回执后
   再调 `complete_login()` 收尾
4. 不内置任何团队相关字段;project_keys 来自 profile,**只为 check 阶段做诊断打印**

两段式调用约定:
    handshake = begin_login(qr_dir="/tmp/auto-dev/meego-login")
    if handshake["status"] == "already_authenticated":
        # 路由层直接进入 Step 0.4
        pass
    elif handshake["status"] == "qr_pending":
        # 路由层:upload handshake["qr_path"] → 渲染 + 等用户回执
        # 用户「继续」后:
        complete_login(handshake["complete_token"])
"""

import json
import os
import subprocess
from pathlib import Path
from typing import Any, Dict, Optional


def _env_with_npm_bin() -> Dict[str, str]:
    env = os.environ.copy()
    env["PATH"] = os.path.expanduser("~/.npm-global/bin") + ":" + env.get("PATH", "")
    return env


def _run_bytedcli(args: list, timeout: int = 60) -> Dict[str, Any]:
    """Run `bytedcli --json <args>` and return parsed JSON envelope."""
    cmd = ["bytedcli", "--json"] + args
    r = subprocess.run(
        cmd, capture_output=True, text=True, env=_env_with_npm_bin(), timeout=timeout
    )
    raw = r.stdout or "{}"
    try:
        payload = json.loads(raw)
    except json.JSONDecodeError as e:
        raise RuntimeError(
            f"bytedcli {' '.join(args)} returned non-JSON (exit={r.returncode}): "
            f"{e}; raw_head={raw[:200]!r}; stderr_head={r.stderr[:200]!r}"
        )
    return payload


def begin_login(qr_dir: str = "/tmp/auto-dev/meego-login") -> Dict[str, Any]:
    """
    Probe auth state; if already authenticated → short-circuit.
    Otherwise kick off device-flow `--begin` and return handshake context for
    the route layer to render QR + wait for user ack.

    判定「已登录」靠 `--begin` 返回 payload 的 `reused_session=True` /
    `authenticated=True` 信号(bytedcli 0.71+ 行为:已登录时不重新拉 QR,
    直接复用 session,无副作用)。这是当前最稳的探测方式 —— 旧路径
    `meego auth status` 不存在(被 CLI 当 help 处理)、`login --check`
    在 0.71 已被移除。

    Return shape:
        {"status": "already_authenticated", "endpoint": str}
      | {"status": "qr_pending",
         "qr_path": str,                      # local PNG path
         "verification_uri_complete": str,    # clickable URL fallback
         "user_code": str,                    # 6-char code to paste if URL pre-fill fails
         "complete_token": str,               # opaque token for complete_login()
         "expires_in_sec": int}
      | {"status": "error", "message": str}
    """
    Path(qr_dir).mkdir(parents=True, exist_ok=True)
    qr_path = str(Path(qr_dir) / "qr.png")

    try:
        payload = _run_bytedcli(
            [
                "meego", "login", "--begin",
                "--qr-image", qr_path,
                "--no-terminal-qr",
            ],
            timeout=30,
        )
    except Exception as e:
        return {"status": "error", "message": f"meego login --begin failed: {e}"}

    if payload.get("status") != "success":
        msg = (payload.get("error") or {}).get("message") or "unknown"
        return {"status": "error", "message": f"meego login --begin: {msg}"}

    data = payload.get("data") or {}

    # 短路 1:已登录态(reused_session=True / authenticated=True 且无 complete_token)
    if data.get("authenticated") is True and data.get("reused_session") is True:
        return {
            "status": "already_authenticated",
            "endpoint": data.get("endpoint") or "",
            "expires_at": data.get("expires_at"),
        }

    complete_token = data.get("complete_token") or data.get("device_code")
    if not complete_token:
        # 兜底:已登录态的另一种 payload 形态(authenticated=True 但 reused_session 缺失)
        if data.get("authenticated") is True:
            return {
                "status": "already_authenticated",
                "endpoint": data.get("endpoint") or "",
                "expires_at": data.get("expires_at"),
            }
        return {
            "status": "error",
            "message": f"meego login --begin: missing complete_token in payload {data}",
        }

    return {
        "status": "qr_pending",
        "qr_path": qr_path,
        "verification_uri_complete": data.get("verification_uri_complete")
        or data.get("verification_uri")
        or "",
        "user_code": data.get("user_code") or "",
        "complete_token": complete_token,
        "expires_in_sec": int(data.get("expires_in") or data.get("expires_in_sec") or 600),
    }


def check_authenticated() -> bool:
    """
    Lightweight probe — calls begin_login() and checks for already_authenticated.
    Side-effect-free for already-logged-in state (bytedcli reuses session, no QR pulled).
    """
    result = begin_login()
    return result.get("status") == "already_authenticated"


def complete_login(complete_token: str, timeout: int = 30) -> Dict[str, Any]:
    """
    Finalize device flow after user confirms QR scan.

    Return shape:
        {"status": "success", "authenticated": True, "endpoint": str}
      | {"status": "error", "message": str}
    """
    try:
        payload = _run_bytedcli(
            ["meego", "login", "--complete", complete_token],
            timeout=timeout,
        )
    except Exception as e:
        return {"status": "error", "message": f"meego login --complete failed: {e}"}

    if payload.get("status") != "success":
        msg = (payload.get("error") or {}).get("message") or "unknown"
        return {"status": "error", "message": f"meego login --complete: {msg}"}

    data = payload.get("data") or {}
    return {
        "status": "success",
        "authenticated": bool(data.get("authenticated")),
        "endpoint": data.get("endpoint") or "",
        "expires_at": data.get("expires_at"),
    }


if __name__ == "__main__":
    import sys
    if len(sys.argv) >= 2 and sys.argv[1] == "complete" and len(sys.argv) >= 3:
        print(json.dumps(complete_login(sys.argv[2]), ensure_ascii=False, indent=2))
    else:
        print(json.dumps(begin_login(), ensure_ascii=False, indent=2))
