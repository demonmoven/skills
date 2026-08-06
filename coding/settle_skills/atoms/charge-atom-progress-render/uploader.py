"""Pluggable upload channel abstraction for charge-atom-progress-render.

为什么独立成模块:
    render.py 原先把"上传文件 → 拿 URL"硬编码到三个 _try_* 函数里
    (mira_runtime SDK / mira CLI / web-hosting CLI),且只覆盖 Mira 沙箱单一
    平台。当 atom 需要在 Coze/devbox/其他 Agent 运行时(无 mcp__runtime__upload_file)
    上跑时,加新通道必须改 render.py 主流程,违反开闭。

本模块定义统一上传接口 UploadChannel,各通道实现自己的 available() + upload(),
publish_file() 按 priority 顺序尝试已注册通道,任一成功即返回,全部失败抛 UploadError。

新增平台只需:
    1. 写一个 XxxChannel(UploadChannel) 子类
    2. 调 register(XxxChannel())
    3. 不动 render.py

通道注册顺序(由 _build_default_registry 控制):
    1. WebHostingChannel    永久态,产物用 hosting CLI 部署到 data.bytedance.net
                             触发条件: env DATA_AGENT_TITAN_PASSPORT_ID
                             或显式设 ENABLE_WEB_HOSTING=1(放宽给 devbox 跑)
    2. MiraRuntimeChannel   优先,Mira 沙箱内默认可用(import mira_runtime)
    3. MiraCliChannel       fallback,mira / bytedcli CLI 任一可用
    4.(预留)CozeChannel    扣子/Coze 运行时上传通道,接入时实现一个子类即可
"""

from __future__ import annotations

import json
import os
import re
import shutil
import subprocess
from abc import ABC, abstractmethod
from pathlib import Path
from typing import List, Optional


class UploadError(Exception):
    """所有上传通道都失败时抛出。"""


# ---------- 通道抽象 ----------

class UploadChannel(ABC):
    """单个上传通道的契约。新增平台只需实现一个子类。

    所有方法都不能抛异常到调用方(除了显式声明的);失败统一返回空串/False。
    """

    #: 通道展示名,出现在 publish 结果的 "channel" 字段里,用户/排障可用
    name: str = "unknown"

    #: 优先级,数字小的先尝试。同优先级按注册顺序。
    priority: int = 100

    @abstractmethod
    def available(self) -> bool:
        """返回 True 表示当前环境下"可以尝试"调用 upload。

        实现应**只做轻量探测**(env / which 命令 / import 模块),不可触发网络
        或 120s 阻塞。否则会拖慢主链路。
        """

    @abstractmethod
    def upload_file(self, local: Path) -> str:
        """上传单个文件,成功返回可公开 fetch 的 URL,失败返回空串。

        实现不应抛异常 —— 任何错误捕获后返回 "" 让 dispatcher 回退到下一条通道。
        """

    # ---- 可选:整目录发布(web-hosting 这种"目录 → 一个 app URL"模型用) ----

    def supports_dir(self) -> bool:
        """通道是否支持整目录发布(默认 False,只有 web-hosting 这种 SPA 部署支持)。"""
        return False

    def upload_dir(self, local_dir: Path, *, app_name: str) -> dict:
        """整目录发布。返回 {"app_url": "...", "json_url": "..."} 或 {}。"""
        return {}


# ---------- 通道实现 ----------

class MiraRuntimeChannel(UploadChannel):
    """Mira 沙箱内的默认通道,通过 mira_runtime python SDK 上传。

    SDK 内部最终走 mcp__runtime__upload_file,但**封装在 Python import 后面**,
    因此从 render.py 视角看 atom **不直接依赖 MCP 工具**,可以在任何能 import
    到 mira_runtime 的 Agent runtime 里跑(不只是 Mira 主端)。
    """

    name = "mira-runtime"
    priority = 10

    def available(self) -> bool:
        try:
            import importlib
            importlib.import_module("mira_runtime")
            return True
        except Exception:
            return False

    def upload_file(self, local: Path) -> str:
        try:
            from mira_runtime import upload_file as _upload  # type: ignore
            result = _upload(str(local))
            if isinstance(result, dict):
                return result.get("url") or result.get("URL") or ""
            if isinstance(result, str):
                return result
        except Exception:
            return ""
        return ""


class MiraCliChannel(UploadChannel):
    """通过 mira / bytedcli 二进制上传。covers mira/bytedcli 各种子命令名变体。"""

    name = "mira-cli"
    priority = 20

    def available(self) -> bool:
        return any(shutil.which(b) for b in ("mira", "bytedcli"))

    def upload_file(self, local: Path) -> str:
        for bin_ in ("mira", "bytedcli"):
            if not shutil.which(bin_):
                continue
            candidates = [
                [bin_, "upload", str(local)],
                [bin_, "files", "upload", str(local)],
                [bin_, "tos", "upload", str(local)],
            ]
            for cmd in candidates:
                try:
                    proc = subprocess.run(cmd, capture_output=True, text=True, timeout=60)
                    if proc.returncode == 0 and proc.stdout:
                        m = re.search(r'https?://\S+', proc.stdout)
                        if m:
                            return m.group(0).strip().rstrip(',";')
                except Exception:
                    continue
        return ""


class WebHostingChannel(UploadChannel):
    """通过 web-hosting CLI 把 dist 目录部署到 data.bytedance.net。

    触发条件(满足任一即认为 available):
        - env DATA_AGENT_TITAN_PASSPORT_ID 已设(无人值守 CI 模式)
        - env ENABLE_WEB_HOSTING=1            (devbox 用户显式启用)
        - HOSTING_CLI_BIN 已指向可用 hosting 二进制(任一情况下都生效)

    沙箱内**默认关闭**,避免 hosting CLI 触发 TAE 浏览器登录 120s 阻塞。
    """

    name = "web-hosting"
    priority = 5  # 优先级最高,部署成功后 URL 是永久态

    def available(self) -> bool:
        has_titan = bool(os.environ.get("DATA_AGENT_TITAN_PASSPORT_ID"))
        explicit = os.environ.get("ENABLE_WEB_HOSTING") == "1"
        if not (has_titan or explicit):
            return False
        return bool(self._resolve_bin())

    def supports_dir(self) -> bool:
        return True

    def upload_file(self, local: Path) -> str:
        """web-hosting 不支持单文件上传,落空。整目录发布走 upload_dir。"""
        return ""

    def upload_dir(self, local_dir: Path, *, app_name: str) -> dict:
        bin_ = self._resolve_bin()
        if not bin_:
            return {}
        try:
            proc = subprocess.run(
                [
                    bin_, "deploy",
                    "--project", str(local_dir),
                    "--dist", str(local_dir),
                    "--skip-build",
                    "--name", app_name,
                    "--no-need-auth",
                    "--json",
                ],
                capture_output=True, text=True, timeout=180,
            )
        except Exception:
            return {}
        if proc.returncode != 0:
            return {}
        out = proc.stdout or ""
        report = None
        m = re.search(r'DEPLOY_REPORT_START\s*(\{.*?\})\s*DEPLOY_REPORT_END', out, re.DOTALL)
        if m:
            try:
                report = json.loads(m.group(1))
            except Exception:
                report = None
        if report is None:
            try:
                report = json.loads(out.strip())
            except Exception:
                return {}
        app_url = report.get("appUri") or report.get("accessUrl") or ""
        if not app_url:
            return {}
        public_base = report.get("publicBaseUrl") or ""
        obj_prefix = report.get("objectPrefix") or ""
        if public_base and obj_prefix:
            json_url = f"{public_base.rstrip('/')}/{obj_prefix.rstrip('/')}/progress.json"
        else:
            json_url = report.get("samplePublicUrl") or ""
        return {"app_url": app_url, "json_url": json_url}

    def _resolve_bin(self) -> str:
        """定位 hosting CLI。优先级: env > skill bundle 内置 tgz > PATH。"""
        env_bin = os.environ.get("HOSTING_CLI_BIN")
        if env_bin and Path(env_bin).exists():
            return env_bin
        # SKILL bundle 内的 hosting-cli.tgz(/data/plugins/custom/skills/web-hosting/)
        tgz = Path("/data/plugins/custom/skills/web-hosting/hosting-cli.tgz")
        if tgz.exists() and shutil.which("npx"):
            # 用 wrapper 调 npx 一次性执行,避免每次解压(npx 自己有 cache)
            return self._make_npx_wrapper(tgz)
        return shutil.which("hosting") or ""

    @staticmethod
    def _make_npx_wrapper(tgz: Path) -> str:
        """生成一个一次性 wrapper 脚本,把 npx -y -p <tgz> hosting 当作单二进制使用。
        缓存到 /tmp,后续复用同一 wrapper,避免重复 stat。
        """
        wrapper = Path("/tmp/charge-progress-hosting-wrapper.sh")
        if not wrapper.exists():
            wrapper.write_text(
                "#!/usr/bin/env bash\n"
                f'exec npx -y -p "{tgz}" hosting "$@"\n'
            )
            wrapper.chmod(0o755)
        return str(wrapper)


# ---------- registry + dispatcher ----------

_REGISTRY: List[UploadChannel] = []


def register(channel: UploadChannel) -> None:
    """注册一个新通道。同名重复注册会覆盖(便于测试/mock)。"""
    global _REGISTRY
    _REGISTRY = [c for c in _REGISTRY if c.name != channel.name]
    _REGISTRY.append(channel)
    _REGISTRY.sort(key=lambda c: c.priority)


def _build_default_registry() -> None:
    register(WebHostingChannel())
    register(MiraRuntimeChannel())
    register(MiraCliChannel())


_build_default_registry()


def registered_channels() -> List[str]:
    """返回当前注册通道名,供调试 / 日志/ stdout 透出。"""
    return [c.name for c in _REGISTRY]


def publish_file(local: Path, *, strict: bool = False) -> str:
    """对外主入口:上传单个文件,返回公网 URL。

    按 priority 顺序遍历已注册通道,任一 available + 上传成功即返回。
    web-hosting 通道在 single-file 模式下不参与(它只支持 dir)。

    失败时:
        - strict=True  → 抛 UploadError(附带尝试过的通道列表)
        - strict=False → 返回空串
    """
    if not local.exists():
        raise UploadError(f"待上传文件不存在: {local}")

    tried = []
    for ch in _REGISTRY:
        if not ch.available():
            continue
        # 整目录通道在 single-file 入口不参与
        if ch.supports_dir() and not getattr(ch, "_force_single_file", False):
            continue
        tried.append(ch.name)
        url = ""
        try:
            url = ch.upload_file(local)
        except Exception:
            url = ""
        if url:
            return url

    if strict:
        raise UploadError(
            f"上传失败,所有通道均不可用或失败 (tried={tried}, registered={registered_channels()}): {local}\n"
            f"如需启用 web-hosting,请 export DATA_AGENT_TITAN_PASSPORT_ID 或 ENABLE_WEB_HOSTING=1。"
        )
    return ""


def publish_dir(local_dir: Path, *, app_name: str) -> dict:
    """对外主入口:整目录发布(仅 web-hosting 类通道支持)。

    成功返回 {"app_url": "...", "json_url": "...", "channel": "<name>"},
    任一通道支持 + 部署成功即返回。
    全失败返回 {}(调用方应回退到 publish_file 逐文件上传)。
    """
    if not local_dir.exists() or not local_dir.is_dir():
        return {}
    for ch in _REGISTRY:
        if not ch.supports_dir():
            continue
        if not ch.available():
            continue
        try:
            result = ch.upload_dir(local_dir, app_name=app_name)
        except Exception:
            result = {}
        if result.get("app_url"):
            result["channel"] = ch.name
            return result
    return {}
