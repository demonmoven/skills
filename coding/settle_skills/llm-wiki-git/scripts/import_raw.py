#!/usr/bin/env python3
"""Import raw references into a Git-only LLM Wiki repository.

This script writes local Git files under raw/ and wiki/LOG.md. For Feishu/Lark
sources, it may read title/content with lark-cli to classify the raw category,
but it never copies full original content and never creates/updates Feishu/Lark
resources.
"""
from __future__ import annotations

import argparse
import json
import re
import shutil
import subprocess
from pathlib import Path
from urllib.parse import urlsplit, urlunsplit

from common import append_log, ensure_dir, now, parse_frontmatter, slugify, yaml_list, yaml_quote

MAX_BATCH_SIZE = 10

LIST_FIELDS = [
    "modules",
    "architecture_layers",
    "business_scenarios",
    "platform_capabilities",
    "systems",
]

CATEGORY_KEYWORDS = {
    "需求文档": ["prd", "需求", "产品需求", "验收标准", "用户故事", "原型", "交互", "排期", "产品方案"],
    "技术方案": ["技术方案", "概要设计", "详细设计", "架构设计", "实现方案", "接口改造", "流程图", "时序图", "rpc", "db", "数据库", "代码", "上线方案", "回滚方案"],
    "ADR决策": ["adr", "决策", "decision", "备选方案", "取舍", "结论", "decision record"],
    "系统白皮书": ["白皮书", "系统架构", "总体架构", "架构全景", "a2架构", "a3架构", "架构白皮书"],
    "数据文档": ["数据模型", "表结构", "字段", "指标", "埋点", "数仓", "sql", "etl", "schema"],
    "接口文档": ["接口文档", "api", "入参", "出参", "request", "response", "endpoint", "http接口", "rpc接口"],
    "值班记录": ["值班", "oncall", "报警处理", "值班记录", "排查记录", "告警"],
    "复盘报告": ["复盘", "事故", "rca", "根因", "改进项", "postmortem", "故障复盘"],
    "团队规约": ["规范", "规约", "sop", "流程", "研发规范", "团队约定", "操作手册"],
    "风险防控": ["风险", "防控", "安全", "越权", "漏洞", "扫描", "治理", "风控", "权限"],
    "日常分享": ["分享", "笔记", "零散记录", "会议纪要", "纪要"],
}


class UserInputNeeded(Exception):
    def __init__(self, payload: dict[str, object]) -> None:
        super().__init__(str(payload.get("reason") or payload))
        self.payload = payload


def split_csv(value: str | None) -> list[str]:
    if not value:
        return []
    return [x.strip() for x in value.split(",") if x.strip()]


def frontmatter(data: dict[str, object]) -> str:
    lines = ["---"]
    for key, value in data.items():
        if isinstance(value, list):
            lines.append(f"{key}:" + yaml_list(value))
        else:
            lines.append(f"{key}: {yaml_quote(str(value))}")
    lines.append("---")
    return "\n".join(lines) + "\n"


def run_json(cmd: list[str]) -> dict[str, object]:
    p = subprocess.run(cmd, text=True, capture_output=True, check=False)
    if p.returncode != 0:
        raise RuntimeError((p.stderr or p.stdout or "").strip() or f"command failed: {' '.join(cmd)}")
    try:
        return json.loads(p.stdout or "{}")
    except json.JSONDecodeError as e:
        raise RuntimeError(f"invalid JSON from {' '.join(cmd)}: {e}") from e


def find_first_key(obj: object, keys: set[str]) -> str:
    if isinstance(obj, dict):
        for k, v in obj.items():
            if k in keys and isinstance(v, (str, int)):
                s = str(v).strip()
                if s:
                    return s
        for v in obj.values():
            found = find_first_key(v, keys)
            if found:
                return found
    elif isinstance(obj, list):
        for v in obj:
            found = find_first_key(v, keys)
            if found:
                return found
    return ""


def collect_text(obj: object, limit: int = 120000) -> str:
    chunks: list[str] = []

    def walk(x: object) -> None:
        if sum(len(c) for c in chunks) > limit:
            return
        if isinstance(x, str):
            s = x.strip()
            if s:
                chunks.append(s)
        elif isinstance(x, dict):
            for key in ("title", "name", "markdown", "content", "text", "plain_text"):
                if key in x:
                    walk(x[key])
            for v in x.values():
                walk(v)
        elif isinstance(x, list):
            for v in x:
                walk(v)

    walk(obj)
    return "\n".join(chunks)[:limit]


def extract_lark_token(value: str | None) -> str:
    if not value:
        return ""
    m = re.search(r"/(?:wiki|docx|docs|doc)/([A-Za-z0-9]+)", value)
    if m:
        return m.group(1)
    m = re.search(r"\b([A-Za-z0-9]{12,})\b", value)
    return m.group(1) if m else ""


def fetch_lark_readonly(lark_url: str | None, doc_id: str | None = None, wiki_token: str | None = None) -> dict[str, str]:
    """Fetch Lark/Wiki title/content for classification. Read-only; never writes back."""
    if shutil.which("lark-cli") is None:
        raise RuntimeError("lark-cli not found")

    token_or_url = lark_url or wiki_token or doc_id
    if not token_or_url:
        raise RuntimeError("missing lark URL/token")

    meta: dict[str, object] = {}
    try:
        meta = run_json(["lark-cli", "wiki", "+node-get", "--as", "user", "--token", token_or_url, "--format", "json"])
    except Exception:
        meta = {}

    title = find_first_key(meta, {"title", "name"})
    obj_token = find_first_key(meta, {"obj_token", "doc_token", "document_id", "doc_id", "token"}) or doc_id or ""
    obj_type = find_first_key(meta, {"obj_type", "type"})
    node_token = find_first_key(meta, {"node_token", "wiki_token"}) or wiki_token or extract_lark_token(lark_url)

    doc: dict[str, object] = {}
    errors: list[str] = []
    for candidate in [obj_token, token_or_url]:
        if not candidate:
            continue
        try:
            doc = run_json([
                "lark-cli", "docs", "+fetch", "--as", "user", "--api-version", "v2",
                "--doc", candidate, "--format", "json",
            ])
            break
        except Exception as e:
            errors.append(str(e))
    if not doc and errors:
        raise RuntimeError("; ".join(errors))

    title = title or find_first_key(doc, {"title", "name"}) or extract_lark_token(lark_url) or "未命名飞书文档"
    content = collect_text({"meta": meta, "doc": doc})
    return {
        "title": title,
        "content": content,
        "doc_id": obj_token or doc_id or "",
        "wiki_token": node_token or "",
        "obj_type": obj_type or "",
    }


def classify_raw_category(text: str, available: list[str]) -> tuple[str, list[tuple[str, int, list[str]]]]:
    lower = text.lower()
    # Strong title/content rules: material type beats general technical keywords.
    # A title containing 白皮书 / 架构白皮书 must be classified as 系统白皮书 if that raw category exists.
    if "系统白皮书" in available and ("白皮书" in lower or "架构白皮书" in lower):
        return "系统白皮书", [("系统白皮书", 999, ["白皮书"])]
    scored: list[tuple[str, int, list[str]]] = []
    for cat in available:
        hits: list[str] = []
        score = 0
        for kw in CATEGORY_KEYWORDS.get(cat, []):
            n = lower.count(kw.lower())
            if n:
                hits.append(kw)
                score += min(n, 3) * (3 if kw.lower() in lower[:1000] else 1)
        if score:
            scored.append((cat, score, hits[:8]))
    scored.sort(key=lambda x: x[1], reverse=True)
    if not scored:
        return "", []
    top = scored[0]
    second = scored[1][1] if len(scored) > 1 else 0
    if top[1] >= 3 and top[1] >= second + 2:
        return top[0], scored
    return "", scored


def sanitize_filename_stem(title: str) -> str:
    # Keep the Lark document title as much as possible; only remove filesystem-illegal characters.
    stem = re.sub(r'[\\/:*?"<>|]+', '-', title).strip().strip('.')
    stem = re.sub(r'\s+', ' ', stem)
    return stem[:120].strip() or "未命名飞书文档"


def stable_slug(title: str, lark_url: str | None = None, doc_id: str | None = None, wiki_token: str | None = None, source_kind: str | None = None) -> str:
    if source_kind in {"lark_doc", "lark_wiki", "lark_auto"}:
        return sanitize_filename_stem(title)
    slug = slugify(title)
    if slug != "untitled":
        return slug[:80].strip("-") or slug
    token = extract_lark_token(lark_url) or wiki_token or doc_id
    if token:
        return slugify(token)[:80]
    return slug


def raw_body(args: argparse.Namespace) -> str:
    lines = [f"# {args.title}", "", "本文件只保存 raw 引用与元数据，不保存飞书/外部原文。", ""]
    if args.lark_url:
        lines.append(f"- 原始飞书地址：<{args.lark_url}>")
    if args.external_url:
        lines.append(f"- 外部地址：<{args.external_url}>")
    if args.local_path:
        label = "本地代码仓库" if args.source_kind == "code_repo" else "本地文件"
        lines.append(f"- {label}：`{args.local_path}`")
    if getattr(args, "repo_path", ""):
        lines.append(f"- 本地代码仓库：`{args.repo_path}`")
    if getattr(args, "repo_url", ""):
        lines.append(f"- 代码仓库：<{args.repo_url}>")
    if getattr(args, "module_path", ""):
        lines.append(f"- module_path：`{args.module_path}`")
    if getattr(args, "branch", ""):
        lines.append(f"- branch：`{args.branch}`")
    if getattr(args, "commit", ""):
        lines.append(f"- commit：`{args.commit}`")
    if getattr(args, "application", ""):
        lines.append(f"- Application：`{args.application}`")
    if getattr(args, "psm", ""):
        lines.append(f"- PSM：`{args.psm}`")
    if getattr(args, "include_path", None):
        lines.append("- include_path：" + ", ".join(f"`{x}`" for x in args.include_path))
    if getattr(args, "exclude_path", None):
        lines.append("- exclude_path：" + ", ".join(f"`{x}`" for x in args.exclude_path))
    if args.doc_id:
        lines.append(f"- doc_id：`{args.doc_id}`")
    if args.wiki_token:
        lines.append(f"- wiki_token：`{args.wiki_token}`")
    if args.obj_type:
        lines.append(f"- obj_type：`{args.obj_type}`")
    if args.note:
        lines.extend(["", "## 备注 / 原始短文本", "", args.note.rstrip(), ""])
    return "\n".join(lines).rstrip() + "\n"


def print_json_exit(payload: dict[str, object], code: int = 2) -> None:
    print(json.dumps(payload, ensure_ascii=False, indent=2))
    raise SystemExit(code)


def normalize_url(value: str) -> str:
    value = value.strip()
    try:
        parts = urlsplit(value)
    except ValueError:
        return value.rstrip("/")
    if not parts.scheme or not parts.netloc:
        return value.rstrip("/")
    return urlunsplit((parts.scheme.lower(), parts.netloc.lower(), parts.path.rstrip("/"), "", ""))


def source_identity(args: argparse.Namespace) -> str:
    if getattr(args, "source_kind", "") == "code_repo":
        repo_id = getattr(args, "repo_url", "") or getattr(args, "local_path", "") or getattr(args, "repo_path", "")
        revision = getattr(args, "commit", "") or getattr(args, "revision", "") or getattr(args, "branch", "")
        if repo_id:
            return "code_repo:" + normalize_url(str(repo_id)) + (f"@{revision}" if revision else "")
    for prefix, value in (
        ("doc_id", args.doc_id),
        ("wiki_token", args.wiki_token),
        ("token", extract_lark_token(args.lark_url)),
        ("url", normalize_url(args.lark_url or "")),
    ):
        if value:
            return f"{prefix}:{value}"
    return ""


def infer_lark_source_kind(requested: str, lark_url: str | None, obj_type: str | None = None) -> str:
    if requested != "lark_auto":
        return requested
    lower_url = (lark_url or "").lower()
    lower_type = (obj_type or "").lower()
    if "/wiki/" in lower_url or lower_type == "wiki":
        return "lark_wiki"
    if any(part in lower_url for part in ("/docx/", "/docs/", "/doc/")) or lower_type in {"doc", "docx", "docs"}:
        return "lark_doc"
    # lark_auto is only a command-level convenience. If the URL shape is not
    # enough to distinguish doc/wiki, default to lark_doc instead of blocking a
    # readable document; raw/source correctness still comes from saved URL/tokens.
    return "lark_doc"


def clone_args(args: argparse.Namespace, lark_url: str | None) -> argparse.Namespace:
    values = vars(args).copy()
    values["lark_url"] = lark_url
    return argparse.Namespace(**values)


def prepare_item(args: argparse.Namespace, repo: Path, available_categories: list[str], lark_url: str | None = None, index: int = 1) -> dict[str, object]:
    item_args = clone_args(args, lark_url)
    classification_note = ""
    fetched: dict[str, str] = {}

    if item_args.source_kind in {"lark_doc", "lark_wiki", "lark_auto"} and (
        not item_args.title or not item_args.raw_category or not item_args.doc_id or not item_args.wiki_token or not item_args.obj_type or item_args.source_kind == "lark_auto"
    ):
        try:
            fetched = fetch_lark_readonly(item_args.lark_url, item_args.doc_id, item_args.wiki_token)
            if not item_args.title:
                item_args.title = fetched.get("title") or item_args.title
            if not item_args.doc_id and fetched.get("doc_id"):
                item_args.doc_id = fetched["doc_id"]
            if not item_args.wiki_token and fetched.get("wiki_token"):
                item_args.wiki_token = fetched["wiki_token"]
            if not item_args.obj_type and fetched.get("obj_type"):
                item_args.obj_type = fetched["obj_type"]
            item_args.source_kind = infer_lark_source_kind(item_args.source_kind, item_args.lark_url, item_args.obj_type)
            if not item_args.raw_category:
                matched, scored = classify_raw_category((item_args.title or "") + "\n" + fetched.get("content", ""), available_categories)
                if matched:
                    item_args.raw_category = matched
                    classification_note = "自动分类: " + matched + "; evidence=" + json.dumps(scored[:3], ensure_ascii=False)
                else:
                    raise UserInputNeeded({
                        "ok": False,
                        "needs_user_input": True,
                        "reason": "无法根据 lark-cli 获取的标题/正文稳定命中 raw 分类，请用户确认 raw_category。",
                        "index": index,
                        "lark_url": item_args.lark_url,
                        "title": item_args.title,
                        "available_categories": available_categories,
                        "candidates": scored[:5],
                    })
        except UserInputNeeded:
            raise
        except Exception as e:
            if not item_args.title or not item_args.raw_category:
                raise UserInputNeeded({
                    "ok": False,
                    "needs_user_input": True,
                    "reason": f"无法通过 lark-cli 读取飞书标题/正文：{e}。请用户提供 title 与 raw_category，或先完成 lark-cli 认证/授权。",
                    "index": index,
                    "lark_url": item_args.lark_url,
                    "available_categories": available_categories,
                }) from e
            item_args.source_kind = infer_lark_source_kind(item_args.source_kind, item_args.lark_url, item_args.obj_type)
            classification_note = f"lark-cli 读取失败，使用用户提供信息: {e}"
    elif item_args.source_kind == "lark_auto":
        item_args.source_kind = infer_lark_source_kind(item_args.source_kind, item_args.lark_url, item_args.obj_type)

    if item_args.source_kind == "code_repo":
        if not item_args.title:
            item_args.title = item_args.application or item_args.psm or Path(item_args.local_path or item_args.repo_path or item_args.repo_url or "代码仓库").name or "代码仓库"
        if not item_args.raw_category and "代码仓库" in available_categories:
            item_args.raw_category = "代码仓库"
    if not item_args.title:
        raise SystemExit("--title is required unless it can be fetched from Lark")
    if not item_args.raw_category:
        raise SystemExit("--raw-category is required unless it can be auto-classified from Lark content")

    raw_dir = repo / "raw" / item_args.raw_category
    if not raw_dir.exists():
        raise SystemExit(f"raw category does not exist: {raw_dir}")

    slug = item_args.slug or (now().split()[0] if item_args.source_kind == "note" and item_args.raw_category == "日常分享" else stable_slug(item_args.title, item_args.lark_url, item_args.doc_id, item_args.wiki_token, item_args.source_kind))
    path = raw_dir / f"{slug}.md"
    return {
        "args": item_args,
        "path": path,
        "path_rel": str(path.relative_to(repo)),
        "classification_note": classification_note,
        "identity": source_identity(item_args),
        "index": index,
    }


def find_existing_raw_by_identity(repo: Path, identity: str, target: Path) -> Path | None:
    if not identity:
        return None
    for raw_file in (repo / "raw").glob("*/*.md"):
        if raw_file.resolve() == target.resolve():
            continue
        fm, _ = parse_frontmatter(raw_file.read_text(encoding="utf-8", errors="replace"))
        ns = argparse.Namespace(
            lark_url=fm.get("lark_url") or "",
            doc_id=fm.get("doc_id") or "",
            wiki_token=fm.get("wiki_token") or "",
        )
        if source_identity(ns) == identity:
            return raw_file
    return None


def preflight_items(repo: Path, items: list[dict[str, object]], overwrite: bool) -> list[dict[str, object]]:
    conflicts: list[dict[str, object]] = []
    seen_identities: dict[str, dict[str, object]] = {}
    seen_paths: dict[Path, dict[str, object]] = {}

    for item in items:
        path = item["path"]  # type: ignore[assignment]
        args = item["args"]  # type: ignore[assignment]
        assert isinstance(path, Path)
        assert isinstance(args, argparse.Namespace)
        identity = str(item.get("identity") or "")

        if identity and identity in seen_identities:
            conflicts.append({
                "type": "duplicate_input_source",
                "identity": identity,
                "items": [seen_identities[identity].get("index"), item.get("index")],
                "suggestion": "批次内存在重复飞书来源，请去重后重试。",
            })
        elif identity:
            seen_identities[identity] = item

        if path in seen_paths:
            other = seen_paths[path]
            conflicts.append({
                "type": "target_path_collision",
                "target": str(path.relative_to(repo)),
                "items": [
                    {"index": other.get("index"), "title": getattr(other["args"], "title", ""), "lark_url": getattr(other["args"], "lark_url", "")},
                    {"index": item.get("index"), "title": args.title, "lark_url": args.lark_url},
                ],
                "suggestion": "多份文档将写入同一个 raw 文件；请确认是否重复，或改名/拆分后重试。",
            })
        else:
            seen_paths[path] = item

        existing = find_existing_raw_by_identity(repo, identity, path) if identity else None
        if existing:
            conflicts.append({
                "type": "same_source_existing_elsewhere",
                "identity": identity,
                "existing_raw": str(existing.relative_to(repo)),
                "planned_raw": str(path.relative_to(repo)),
                "suggestion": "同一飞书来源已登记到其他 raw 文件，请先确认是否迁移或复用既有 raw。",
            })

        if path.exists() and not overwrite:
            old_fm, _ = parse_frontmatter(path.read_text(encoding="utf-8", errors="replace"))
            old_ns = argparse.Namespace(
                lark_url=old_fm.get("lark_url") or "",
                doc_id=old_fm.get("doc_id") or "",
                wiki_token=old_fm.get("wiki_token") or "",
            )
            old_identity = source_identity(old_ns)
            if old_identity and identity and old_identity != identity:
                conflicts.append({
                    "type": "existing_raw_points_to_different_source",
                    "target": str(path.relative_to(repo)),
                    "existing_identity": old_identity,
                    "planned_identity": identity,
                    "suggestion": "目标 raw 文件已指向另一个飞书来源，未确认前不能增量更新。",
                })
            old_category = old_fm.get("raw_category")
            if old_category and old_category != args.raw_category:
                conflicts.append({
                    "type": "existing_raw_category_mismatch",
                    "target": str(path.relative_to(repo)),
                    "existing_raw_category": old_category,
                    "planned_raw_category": args.raw_category,
                    "suggestion": "目标 raw 文件分类与本次计划分类不一致，请先确认分类。",
                })
    return conflicts


def write_item(repo: Path, item: dict[str, object], ts: str) -> dict[str, object]:
    args = item["args"]
    path = item["path"]
    assert isinstance(args, argparse.Namespace)
    assert isinstance(path, Path)

    data: dict[str, object] = {
        "type": "raw_ref",
        "title": args.title,
        "raw_category": args.raw_category,
        "source_kind": args.source_kind,
    }
    for key in ["lark_url", "doc_id", "wiki_token", "obj_type", "local_path", "repo_path", "repo_url", "module_path", "branch", "commit", "application", "psm", "external_url"]:
        value = getattr(args, key, None)
        if value:
            data[key] = value
    for key in ["include_path", "exclude_path"]:
        value = getattr(args, key, None)
        if value:
            data[key] = value
    for key in LIST_FIELDS:
        values = split_csv(getattr(args, key))
        data[key] = values
    data["imported_at"] = ts
    data["updated_at"] = ts

    if path.exists() and not args.overwrite:
        old_fm, old_body = parse_frontmatter(path.read_text(encoding="utf-8", errors="replace"))
        old_fm.update({k: v for k, v in data.items() if v != []})
        old_fm["updated_at"] = ts
        body = old_body.rstrip()
        if args.note:
            body += f"\n\n## {ts}\n\n{args.note.rstrip()}\n"
        path.write_text(frontmatter(old_fm) + "\n" + body.rstrip() + "\n", encoding="utf-8")
        action = "更新 raw 引用"
    else:
        ensure_dir(path.parent)
        path.write_text(frontmatter(data) + "\n" + raw_body(args), encoding="utf-8")
        action = "创建 raw 引用"

    return {
        "ok": True,
        "title": args.title,
        "source_kind": args.source_kind,
        "raw_category": args.raw_category,
        "raw_ref": str(path.relative_to(repo)),
        "action": action,
        "classification_note": item.get("classification_note") or "用户显式提供或非飞书来源规则判断",
    }


def batch_ingest_command(repo: Path, raw_refs: list[str]) -> str:
    return "python3 scripts/ingest_source.py --repo " + str(repo) + " " + " ".join(f"--raw {ref}" for ref in raw_refs)


def main() -> None:
    ap = argparse.ArgumentParser(description="Create/update raw reference in a Git-only LLM Wiki")
    ap.add_argument("--repo", required=True, help="LLM Wiki Git repository root")
    ap.add_argument("--title", help="Material title; for Lark sources, omitted title is fetched with lark-cli")
    ap.add_argument("--raw-category", help="Target raw category; for Lark sources, omitted category is auto-classified from fetched content")
    ap.add_argument("--source-kind", required=True, choices=["lark_auto", "lark_doc", "lark_wiki", "local_file", "external_url", "note", "code_repo", "other"], help="Raw source kind")
    ap.add_argument("--slug", help="Stable file slug; recommended for Chinese titles")
    ap.add_argument("--lark-url", action="append", help="Original Feishu/Lark URL; repeat up to 10 times for batch Feishu import")
    ap.add_argument("--doc-id", help="Feishu doc_id/document_id")
    ap.add_argument("--wiki-token", help="Feishu wiki token")
    ap.add_argument("--obj-type", help="Feishu object type, e.g. docx")
    ap.add_argument("--local-path", help="Local file path reference; for code_repo this may be the local repository path")
    ap.add_argument("--repo-path", help="Local code repository path (alias/metadata for source_kind=code_repo)")
    ap.add_argument("--repo-url", help="Code repository URL for source_kind=code_repo")
    ap.add_argument("--module-path", help="Code module path, e.g. Go module or package root")
    ap.add_argument("--branch", help="Code repository branch for source_kind=code_repo")
    ap.add_argument("--commit", help="Code repository commit/revision for source_kind=code_repo")
    ap.add_argument("--application", help="Application/PSM wiki entry associated with a code_repo raw ref")
    ap.add_argument("--psm", help="PSM name associated with a code_repo raw ref")
    ap.add_argument("--include-path", action="append", help="Included code path for source_kind=code_repo; repeatable")
    ap.add_argument("--exclude-path", action="append", help="Excluded code path for source_kind=code_repo; repeatable")
    ap.add_argument("--external-url", help="External URL reference")
    ap.add_argument("--modules", help="Comma-separated business modules")
    ap.add_argument("--architecture-layers", help="Comma-separated architecture layers")
    ap.add_argument("--business-scenarios", help="Comma-separated business scenarios")
    ap.add_argument("--platform-capabilities", help="Comma-separated platform capabilities")
    ap.add_argument("--systems", help="Comma-separated systems/applications")
    ap.add_argument("--note", help="Short note/original brief text")
    ap.add_argument("--overwrite", action="store_true", help="Overwrite existing raw reference body")
    args = ap.parse_args()

    repo = Path(args.repo).expanduser().resolve()
    if not (repo / "raw").exists() or not (repo / "wiki").exists():
        raise SystemExit(f"Not an LLM Wiki Git repo: {repo}")
    available_categories = sorted([p.name for p in (repo / "raw").iterdir() if p.is_dir()])

    lark_urls = args.lark_url or []
    batch = len(lark_urls) > 1
    if len(lark_urls) > MAX_BATCH_SIZE:
        print_json_exit({
            "ok": False,
            "needs_user_input": True,
            "reason": f"一次批量导入最多支持 {MAX_BATCH_SIZE} 个飞书文档。",
            "count": len(lark_urls),
            "max": MAX_BATCH_SIZE,
        })
    if batch:
        if args.source_kind not in {"lark_auto", "lark_doc", "lark_wiki"}:
            print_json_exit({"ok": False, "needs_user_input": True, "reason": "批量导入当前只支持飞书来源。", "source_kind": args.source_kind})
        forbidden = [name for name in ["title", "slug", "doc_id", "wiki_token", "obj_type"] if getattr(args, name)]
        if forbidden:
            print_json_exit({
                "ok": False,
                "needs_user_input": True,
                "reason": "批量导入不能使用单文档标量参数，避免误套用到多篇文档。",
                "forbidden_args": ["--" + name.replace("_", "-") for name in forbidden],
            })

    urls_for_items = lark_urls if lark_urls else [None]
    items: list[dict[str, object]] = []
    try:
        for idx, url in enumerate(urls_for_items, start=1):
            items.append(prepare_item(args, repo, available_categories, url, idx))
    except UserInputNeeded as e:
        print_json_exit(e.payload)

    conflicts = preflight_items(repo, items, args.overwrite)
    if conflicts:
        print_json_exit({
            "ok": False,
            "needs_user_input": True,
            "reason": "导入预检发现结构冲突，未写入文件。",
            "batch": batch,
            "conflicts": conflicts,
        })

    ts = now()
    results = [write_item(repo, item, ts) for item in items]

    if batch:
        raw_refs = [str(r["raw_ref"]) for r in results]
        lines = [
            "---",
            "",
            f"### {ts} — IMPORT-BATCH",
            "",
            "**操作**: 批量导入 raw 引用",
            f"**数量**: {len(results)} / {MAX_BATCH_SIZE}",
            "**素材**:",
        ]
        for result in results:
            lines.append(f"- `{result['raw_ref']}` — {result['title']} — {result['source_kind']} — {result['classification_note']}")
        lines.extend([
            "**结构冲突**: 无",
            "**后续**: 默认继续批量 ingest，除非用户明确只导入不摄入。",
        ])
        append_log(repo, "\n".join(lines) + "\n")
        print(json.dumps({
            "ok": True,
            "batch": True,
            "count": len(results),
            "repo": str(repo),
            "raw_refs": raw_refs,
            "results": results,
            "warnings": [],
            "conflicts": [],
            "next": {"default_ingest_command": batch_ingest_command(repo, raw_refs)},
        }, ensure_ascii=False, indent=2))
    else:
        result = results[0]
        append_log(repo, f"---\n\n### {ts} — IMPORT\n\n**操作**: {result['action']}\n**标题**: {result['title']}\n**source_kind**: {result['source_kind']}\n**raw 分类**: {result['raw_category']}\n**raw 文件**: `{result['raw_ref']}`\n**分类依据**: {result['classification_note']}\n**后续**: 默认继续 ingest，除非用户明确只导入不摄入。\n")
        print(json.dumps({"ok": True, "repo": str(repo), "raw_ref": result["raw_ref"], "raw_category": result["raw_category"], "title": result["title"], "action": result["action"]}, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
