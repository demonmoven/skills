#!/usr/bin/env python3
"""Common helpers for LLM Wiki Git-only scripts."""
from __future__ import annotations

import re
from datetime import datetime
from pathlib import Path

DEFAULT_RAW_SUBDIRS = [
    "需求文档",
    "技术方案",
    "ADR决策",
    "系统白皮书",
    "数据文档",
    "接口文档",
    "值班记录",
    "复盘报告",
    "团队规约",
    "风险防控",
    "日常分享",
    "代码仓库",
]

WIKI_DIRS = [
    "sources",
    "modules",
    "scenarios",
    "platform-capabilities",
    "applications",
    "code-components",
    "data",
    "implementations",
    "maps",
    "overviews",
    "comparisons",
    "query_feedback",
    "registry",
]

TRACKED_EMPTY_WIKI_DIRS = [
    "modules",
    "scenarios",
    "platform-capabilities",
    "code-components",
    "implementations",
    "maps",
    "overviews",
    "comparisons",
    "query_feedback",
]

REGISTRY_FIXED = [
    "REGISTRY-Modules.md",
    "REGISTRY-Scenarios.md",
    "REGISTRY-PlatformCapabilities.md",
    "REGISTRY-Applications.md",
    "REGISTRY-CodeComponents.md",
    "REGISTRY-Data.md",
    "REGISTRY-Implementations.md",
    "REGISTRY-Maps.md",
    "REGISTRY-Overviews.md",
    "REGISTRY-Comparisons.md",
    "REGISTRY-QueryFeedback.md",
]

BILLING_SETTLEMENT_MODULES = [
    ("charge", "计费"),
    ("settlement", "结算"),
]

# Deprecated compatibility alias for older callers/imports. New code should use
# BILLING_SETTLEMENT_MODULES and --with-billing-settlement-defaults.
BYTEPAY_MODULES = BILLING_SETTLEMENT_MODULES

DEFAULT_MAPS = [
    ("module-scenario-map", "平台系统到业务场景映射"),
    ("scenario-platform-map", "业务场景到平台能力映射"),
    ("platform-application-map", "平台能力到应用系统映射"),
    ("application-data-map", "应用系统到数据模型映射"),
    ("application-business-map", "应用到业务场景能力映射"),
    ("implementation-codecomponent-map", "实现链路到代码组件映射"),
    ("implementation-data-map", "实现链路到数据模型映射"),
    ("module-capability-map", "平台系统能力总览"),
    ("end-to-end-billing-settlement-flow", "端到端计费结算链路"),
]


def now() -> str:
    return datetime.now().strftime("%Y-%m-%d %H:%M")


def slugify(text: str) -> str:
    text = text.strip().lower()
    # Keep ASCII words/numbers; collapse the rest into '-'. For Chinese-only titles,
    # callers should pass --slug for a stable semantic slug.
    slug = re.sub(r"[^a-z0-9]+", "-", text).strip("-")
    return slug or "untitled"


def safe_filename_stem(text: str) -> str:
    """Return a filesystem-safe filename stem while preserving the source title.

    Source pages should keep the original document name whenever possible; this
    helper only removes path separators and characters that are unsafe on common
    filesystems instead of transliterating Chinese titles into opaque slugs.
    """
    text = text.strip()
    text = re.sub(r'[\\/:*?"<>|]+', "_", text)
    text = re.sub(r"\s+", " ", text).strip(" .")
    return text or slugify(text) or "untitled"


def ensure_dir(path: Path) -> None:
    path.mkdir(parents=True, exist_ok=True)


def ensure_gitkeep(path: Path) -> None:
    """Keep intentionally empty wiki directories visible to Git."""
    ensure_dir(path)
    gitkeep = path / ".gitkeep"
    if not gitkeep.exists():
        gitkeep.write_text("", encoding="utf-8")


def write_if_missing(path: Path, content: str) -> bool:
    if path.exists():
        return False
    ensure_dir(path.parent)
    path.write_text(content, encoding="utf-8")
    return True


def append_log(repo: Path, entry: str) -> None:
    log = repo / "wiki" / "LOG.md"
    ensure_dir(log.parent)
    if not log.exists():
        log.write_text("# LOG\n\n最新操作在最下方。\n", encoding="utf-8")
    with log.open("a", encoding="utf-8") as f:
        f.write("\n" + entry.rstrip() + "\n")


def yaml_quote(value: str) -> str:
    return '"' + value.replace('"', '\\"') + '"'


def yaml_list(values: list[str]) -> str:
    if not values:
        return "[]"
    return "\n" + "\n".join(f"  - {v}" for v in values)


def parse_frontmatter(text: str) -> tuple[dict[str, str], str]:
    if not text.startswith("---\n"):
        return {}, text
    end = text.find("\n---\n", 4)
    if end == -1:
        return {}, text
    raw = text[4:end]
    body = text[end + 5 :]
    data: dict[str, str] = {}
    for line in raw.splitlines():
        if not line.strip() or line.startswith(" ") or line.startswith("-"):
            continue
        if ":" in line:
            k, v = line.split(":", 1)
            data[k.strip()] = v.strip().strip('"')
    return data, body
