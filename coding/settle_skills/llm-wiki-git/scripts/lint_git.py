#!/usr/bin/env python3
"""Lint a Git-only LLM Wiki repository locally.

This is a read-only checker. It does not access Feishu/Lark and does not modify
files. Use it before manual/LLM deep lint to catch structural issues quickly.
"""
from __future__ import annotations

import argparse
import json
import os
import re
from pathlib import Path

from common import parse_frontmatter

RE_MD_LINK = re.compile(r"\[[^\]]+\]\((?!https?://|mailto:|#)([^)]+\.md(?:#[^)]+)?)\)")


def rel(repo: Path, p: Path) -> str:
    return str(p.relative_to(repo))


def status(errors: list, warnings: list) -> str:
    if errors:
        return "ERROR"
    if warnings:
        return "WARNING"
    return "PASS"


def link_target_exists(base_file: Path, href: str) -> bool:
    href = href.split("#", 1)[0]
    target = (base_file.parent / href).resolve()
    return target.exists()


def registry_links(registry_file: Path) -> list[str]:
    text = registry_file.read_text(encoding="utf-8", errors="replace")
    return [m.group(1).split("#", 1)[0] for m in RE_MD_LINK.finditer(text)]


def main() -> None:
    ap = argparse.ArgumentParser(description="Read-only local lint for Git-only LLM Wiki")
    ap.add_argument("--repo", required=True, help="LLM Wiki Git repository root")
    ap.add_argument("--json", action="store_true", help="Print full JSON report")
    args = ap.parse_args()

    repo = Path(args.repo).expanduser().resolve()
    errors: list[dict[str, str]] = []
    warnings: list[dict[str, str]] = []
    stats: dict[str, object] = {}

    required = [repo / "raw", repo / "wiki", repo / "wiki" / "registry", repo / "wiki" / "registry" / "REGISTRY.md"]
    for p in required:
        if not p.exists():
            errors.append({"dimension": "D1", "file": rel(repo, p) if repo in p.parents or p == repo else str(p), "message": "required path missing"})

    raw_files = sorted((repo / "raw").glob("*/*.md")) if (repo / "raw").exists() else []
    wiki_files = sorted(p for p in (repo / "wiki").rglob("*.md")) if (repo / "wiki").exists() else []
    registry_files = sorted((repo / "wiki" / "registry").glob("REGISTRY*.md")) if (repo / "wiki" / "registry").exists() else []
    source_files = sorted(p for p in (repo / "wiki" / "sources").rglob("*.md")) if (repo / "wiki" / "sources").exists() else []

    stats["raw_files"] = len(raw_files)
    stats["wiki_files"] = len(wiki_files)
    stats["source_files"] = len(source_files)
    stats["registry_files"] = len(registry_files)
    raw_by_category: dict[str, int] = {}

    for p in raw_files:
        raw_by_category[p.parent.name] = raw_by_category.get(p.parent.name, 0) + 1
        text = p.read_text(encoding="utf-8", errors="replace")
        fm, body = parse_frontmatter(text)
        for key in ["type", "title", "raw_category", "source_kind"]:
            if not fm.get(key):
                errors.append({"dimension": "D4", "file": rel(repo, p), "message": f"raw frontmatter missing {key}"})
        if fm.get("type") and fm.get("type") != "raw_ref":
            errors.append({"dimension": "D4", "file": rel(repo, p), "message": "raw type must be raw_ref"})
        if fm.get("raw_category") and fm.get("raw_category") != p.parent.name:
            errors.append({"dimension": "D4", "file": rel(repo, p), "message": f"raw_category {fm.get('raw_category')} != path category {p.parent.name}"})
        if fm.get("source_kind") in {"lark_doc", "lark_wiki"} and not (fm.get("lark_url") or fm.get("doc_id") or fm.get("wiki_token")):
            errors.append({"dimension": "D4", "file": rel(repo, p), "message": "lark raw ref missing lark_url/doc_id/wiki_token"})
        if not body.strip():
            warnings.append({"dimension": "D1", "file": rel(repo, p), "message": "raw body is empty"})

    source_to_registry: set[str] = set()
    bad_registry_links: list[dict[str, str]] = []
    for reg in registry_files:
        for href in registry_links(reg):
            target = (reg.parent / href).resolve()
            if not target.exists():
                bad_registry_links.append({"file": rel(repo, reg), "link": href})
            elif repo / "wiki" / "sources" in target.parents:
                source_to_registry.add(rel(repo, target))
    for item in bad_registry_links:
        errors.append({"dimension": "D2", "file": item["file"], "message": f"registry broken link: {item['link']}"})

    for p in source_files:
        text = p.read_text(encoding="utf-8", errors="replace")
        fm, body = parse_frontmatter(text)
        if fm.get("type") != "source":
            errors.append({"dimension": "D4", "file": rel(repo, p), "message": "Source type must be source"})
        raw_ref = fm.get("raw_ref")
        if not raw_ref:
            errors.append({"dimension": "D2", "file": rel(repo, p), "message": "Source missing raw_ref"})
        else:
            raw_path = (p.parent / raw_ref).resolve()
            if not raw_path.exists():
                errors.append({"dimension": "D2", "file": rel(repo, p), "message": f"raw_ref not found: {raw_ref}"})
            elif fm.get("raw_category") and raw_path.parent.name != fm.get("raw_category"):
                errors.append({"dimension": "D4", "file": rel(repo, p), "message": f"Source raw_category {fm.get('raw_category')} != raw path category {raw_path.parent.name}"})
        if rel(repo, p) not in source_to_registry:
            errors.append({"dimension": "D5", "file": rel(repo, p), "message": "Source not registered in REGISTRY-Sources-*"})
        if len(body.strip()) < 20:
            warnings.append({"dimension": "D1", "file": rel(repo, p), "message": "Source body looks empty"})

    # General local Markdown link check for wiki pages except registry already handled.
    for p in wiki_files:
        text = p.read_text(encoding="utf-8", errors="replace")
        for m in RE_MD_LINK.finditer(text):
            href = m.group(1)
            if not link_target_exists(p, href):
                errors.append({"dimension": "D2", "file": rel(repo, p), "message": f"broken markdown link: {href}"})

    # Unregistered non-registry wiki pages.
    registry_text = "\n".join(r.read_text(encoding="utf-8", errors="replace") for r in registry_files)
    for p in wiki_files:
        if "/registry/" in rel(repo, p) or rel(repo, p) in {"wiki/INDEX.md", "wiki/LOG.md"}:
            continue
        if rel(repo, p) not in registry_text and os.path.relpath(p, start=repo / "wiki" / "registry") not in registry_text:
            warnings.append({"dimension": "D3", "file": rel(repo, p), "message": "wiki page may be unregistered"})


    # CodeComponent structural checks.
    code_component_files = sorted((repo / "wiki" / "code-components").glob("*.md")) if (repo / "wiki" / "code-components").exists() else []
    stats["code_component_files"] = len(code_component_files)
    code_components_registry = repo / "wiki" / "registry" / "REGISTRY-CodeComponents.md"
    registry_text_all = registry_text
    for p in code_component_files:
        text = p.read_text(encoding="utf-8", errors="replace")
        fm, body = parse_frontmatter(text)
        if fm.get("type") != "code_component":
            errors.append({"dimension": "D4", "file": rel(repo, p), "message": "CodeComponent type must be code_component"})
        if rel(repo, p) not in registry_text_all and os.path.relpath(p, start=repo / "wiki" / "registry") not in registry_text_all:
            errors.append({"dimension": "D3", "file": rel(repo, p), "message": "CodeComponent not registered in REGISTRY-CodeComponents"})
        title = fm.get("title") or ""
        h1 = ""
        for line in body.splitlines():
            if line.startswith("# "):
                h1 = line[2:].strip()
                break
        if title and h1 and title != h1:
            warnings.append({"dimension": "D9", "file": rel(repo, p), "message": f"CodeComponent title/frontmatter mismatch: {title} != {h1}"})
        if "../applications/" not in text and "applications:" not in text:
            warnings.append({"dimension": "D13", "file": rel(repo, p), "message": "CodeComponent missing Application reference"})
        if "../sources/" not in text:
            warnings.append({"dimension": "D13", "file": rel(repo, p), "message": "CodeComponent missing Source evidence link"})
    if code_component_files and not code_components_registry.exists():
        errors.append({"dimension": "D3", "file": rel(repo, code_components_registry), "message": "REGISTRY-CodeComponents.md missing while code-components exist"})

    # Code repository Source checks.
    for p in raw_files:
        text = p.read_text(encoding="utf-8", errors="replace")
        fm, _ = parse_frontmatter(text)
        if fm.get("source_kind") == "code_repo":
            if not (fm.get("repo_url") or fm.get("local_path") or fm.get("repo_path")):
                errors.append({"dimension": "D4", "file": rel(repo, p), "message": "code_repo raw missing repo_url/local_path/repo_path"})
            if not (fm.get("commit") or fm.get("branch")):
                warnings.append({"dimension": "D12", "file": rel(repo, p), "message": "code_repo raw missing commit/branch revision"})

    stats["raw_by_category"] = raw_by_category
    report = {
        "ok": not errors,
        "repo": str(repo),
        "status": status(errors, warnings),
        "dimensions": {
            "D1_empty_or_required_paths": "ERROR" if any(e["dimension"] == "D1" for e in errors) else ("WARNING" if any(w["dimension"] == "D1" for w in warnings) else "PASS"),
            "D2_broken_links": "ERROR" if any(e["dimension"] == "D2" for e in errors) else "PASS",
            "D3_registration": "ERROR" if any(e["dimension"] in {"D3", "D5", "D6"} for e in errors) else ("WARNING" if any(w["dimension"] == "D3" for w in warnings) else "PASS"),
            "D4_raw_source_category": "ERROR" if any(e["dimension"] == "D4" for e in errors) else "PASS",
            "D7_D13_deep_content": "PARTIAL",
        },
        "stats": stats,
        "errors": errors,
        "warnings": warnings,
        "note": "This script is structural/local lint only. D9-D13 semantic conflict/staleness checks still require LLM deep lint over page bodies.",
    }

    if args.json:
        print(json.dumps(report, ensure_ascii=False, indent=2))
    else:
        print(f"LLM Wiki lint: {report['status']} ({repo})")
        print(json.dumps(report["dimensions"], ensure_ascii=False, indent=2))
        print(json.dumps(report["stats"], ensure_ascii=False, indent=2))
        if errors:
            print("\nERRORS:")
            for e in errors:
                print(f"- [{e['dimension']}] {e['file']}: {e['message']}")
        if warnings:
            print("\nWARNINGS:")
            for w in warnings:
                print(f"- [{w['dimension']}] {w['file']}: {w['message']}")

    raise SystemExit(0 if not errors else 1)


if __name__ == "__main__":
    main()
