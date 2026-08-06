#!/usr/bin/env python3
"""Create Source skeletons from raw references and register them locally.

This helper does not summarize source content by itself. The LLM should read the
raw reference/original material, fill the Source body and update downstream wiki
pages. The script only guarantees raw -> Source -> REGISTRY bookkeeping.
"""
from __future__ import annotations

import argparse
import json
import re
from pathlib import Path

from common import append_log, ensure_dir, now, parse_frontmatter, safe_filename_stem, yaml_list, yaml_quote

MAX_BATCH_SIZE = 10

LIST_FIELDS = [
    "modules",
    "architecture_layers",
    "business_scenarios",
    "platform_capabilities",
    "systems",
]


def get_list_from_fm(text: str, key: str) -> list[str]:
    m = re.search(rf"^{re.escape(key)}:\n((?:  - .+\n?)*)", text, flags=re.M)
    if not m:
        return []
    return [line.strip()[2:].strip() for line in m.group(1).splitlines() if line.strip().startswith("- ")]


def frontmatter(data: dict[str, object]) -> str:
    lines = ["---"]
    for key, value in data.items():
        if isinstance(value, list):
            lines.append(f"{key}:" + yaml_list(value))
        else:
            lines.append(f"{key}: {yaml_quote(str(value))}")
    lines.append("---")
    return "\n".join(lines) + "\n"


def rel_from(target: Path, base_file: Path) -> str:
    return target.relative_to(base_file.parent) if False else __import__("os").path.relpath(target, start=base_file.parent)


def registry_row(title: str, doc_rel: str, raw_category: str, raw_rel: str) -> str:
    ts = now()
    return f"| {title} | source | [{doc_rel}]({doc_rel}) | {raw_category} | {ts} | raw: [{raw_rel}]({raw_rel}) |\n"


def upsert_registry(registry: Path, row: str, source_rel_from_registry: str) -> None:
    ensure_dir(registry.parent)
    if not registry.exists():
        registry.write_text("# " + registry.stem + "\n\n| 标题 | 类型 | Doc | 分类/目录 | 最后更新 | 关联 |\n|---|---|---|---|---|---|\n", encoding="utf-8")
    text = registry.read_text(encoding="utf-8", errors="replace")
    # Remove previous rows pointing to the same source doc.
    lines = [ln for ln in text.splitlines() if f"]({source_rel_from_registry})" not in ln]
    if lines and lines[-1].strip():
        lines.append(row.rstrip())
    else:
        lines.append(row.rstrip())
    registry.write_text("\n".join(lines).rstrip() + "\n", encoding="utf-8")


def print_json_exit(payload: dict[str, object], code: int = 2) -> None:
    print(json.dumps(payload, ensure_ascii=False, indent=2))
    raise SystemExit(code)


def resolve_raw_path(repo: Path, raw_arg: str) -> Path:
    raw_path = Path(raw_arg).expanduser()
    if not raw_path.is_absolute():
        raw_path = repo / raw_path
    return raw_path.resolve()


def resolve_source_raw_ref(source_path: Path, raw_ref: str) -> Path | None:
    if not raw_ref:
        return None
    try:
        raw_path = Path(raw_ref).expanduser()
        if not raw_path.is_absolute():
            raw_path = source_path.parent / raw_path
        return raw_path.resolve()
    except OSError:
        return None


def prepare_item(repo: Path, raw_arg: str, source_slug: str | None = None, title_arg: str | None = None, index: int = 1) -> dict[str, object]:
    raw_path = resolve_raw_path(repo, raw_arg)
    if not raw_path.exists():
        raise FileNotFoundError(f"raw reference not found: {raw_path}")
    if repo not in raw_path.parents:
        raise ValueError(f"raw reference must be inside repo: {raw_path}")

    raw_text = raw_path.read_text(encoding="utf-8", errors="replace")
    raw_fm, _ = parse_frontmatter(raw_text)
    raw_category = raw_fm.get("raw_category") or raw_path.parent.name
    if raw_path.parent.name != raw_category:
        raise ValueError(f"raw_category mismatch: frontmatter={raw_category}, path={raw_path.parent.name}, raw={raw_path.relative_to(repo)}")
    raw_title = raw_fm.get("title") or raw_path.stem
    source_name = raw_title
    title = title_arg or f"Source：{source_name}"
    if not (title.startswith("Source：") or title.startswith("Source:")):
        title = f"Source：{title}"
    slug = source_slug or safe_filename_stem(raw_path.stem)
    source_path = repo / "wiki" / "sources" / raw_category / f"{slug}.md"

    return {
        "index": index,
        "raw_arg": raw_arg,
        "raw_path": raw_path,
        "raw_text": raw_text,
        "raw_category": raw_category,
        "raw_title": raw_title,
        "title": title,
        "source_path": source_path,
    }


def preflight_items(repo: Path, items: list[dict[str, object]], batch: bool, overwrite: bool) -> list[dict[str, object]]:
    conflicts: list[dict[str, object]] = []
    seen_raws: dict[Path, dict[str, object]] = {}
    seen_sources: dict[Path, dict[str, object]] = {}

    for item in items:
        raw_path = item["raw_path"]
        source_path = item["source_path"]
        assert isinstance(raw_path, Path)
        assert isinstance(source_path, Path)

        if raw_path in seen_raws:
            conflicts.append({
                "type": "duplicate_raw_in_batch",
                "raw": str(raw_path.relative_to(repo)),
                "items": [seen_raws[raw_path].get("index"), item.get("index")],
                "suggestion": "批次内 raw 重复，请去重后重试。",
            })
        else:
            seen_raws[raw_path] = item

        if source_path in seen_sources:
            other = seen_sources[source_path]
            conflicts.append({
                "type": "target_source_path_collision",
                "target": str(source_path.relative_to(repo)),
                "items": [
                    {"index": other.get("index"), "raw": str(other["raw_path"].relative_to(repo))},
                    {"index": item.get("index"), "raw": str(raw_path.relative_to(repo))},
                ],
                "suggestion": "多个 raw 将生成同一个 Source 文件，请先确认文档命名或使用单文档模式指定 source-slug。",
            })
        else:
            seen_sources[source_path] = item

        if source_path.exists() and not overwrite:
            source_fm, _ = parse_frontmatter(source_path.read_text(encoding="utf-8", errors="replace"))
            existing_raw_ref = source_fm.get("raw_ref") or ""
            existing_raw_path = resolve_source_raw_ref(source_path, existing_raw_ref)
            if existing_raw_path and existing_raw_path != raw_path:
                conflicts.append({
                    "type": "existing_source_points_to_different_raw",
                    "source": str(source_path.relative_to(repo)),
                    "existing_raw_ref": existing_raw_ref,
                    "planned_raw": str(raw_path.relative_to(repo)),
                    "suggestion": "已有 Source 指向不同 raw，未确认前不能更新注册。",
                })
            elif batch and not existing_raw_ref:
                conflicts.append({
                    "type": "existing_source_raw_ref_unknown",
                    "source": str(source_path.relative_to(repo)),
                    "planned_raw": str(raw_path.relative_to(repo)),
                    "suggestion": "已有 Source 缺少 raw_ref，批量模式无法安全确认匹配关系，请先补齐或单独处理。",
                })
    return conflicts


def build_source_content(repo: Path, item: dict[str, object], ts: str) -> tuple[str, str, str, str, dict[str, object]]:
    raw_path = item["raw_path"]
    source_path = item["source_path"]
    raw_text = item["raw_text"]
    raw_category = str(item["raw_category"])
    title = str(item["title"])
    assert isinstance(raw_path, Path)
    assert isinstance(source_path, Path)
    assert isinstance(raw_text, str)

    data: dict[str, object] = {
        "type": "source",
        "title": title,
        "raw_ref": rel_from(raw_path, source_path),
        "raw_category": raw_category,
    }
    for key in LIST_FIELDS:
        data[key] = get_list_from_fm(raw_text, key)
    data["created_at"] = ts
    data["updated_at"] = ts

    raw_fm, _ = parse_frontmatter(raw_text)
    if raw_fm.get("source_kind") == "code_repo":
        body = f"""# {title}

## 元数据

- 原始来源：[{raw_path.relative_to(repo)}]({rel_from(raw_path, source_path)})
- Raw 分类：`raw/{raw_category}`
- repo_url：{raw_fm.get('repo_url') or '待补充'}
- local_path：{raw_fm.get('local_path') or raw_fm.get('repo_path') or '待补充'}
- module_path：{raw_fm.get('module_path') or '待补充'}
- branch / commit：{raw_fm.get('branch') or '待补充'} / {raw_fm.get('commit') or '待补充'}
- Application：{raw_fm.get('application') or raw_fm.get('psm') or '待补充'}

## 摘要

待读取代码仓库后补充。注意：本 Source 只沉淀代码仓库结构化摘要、证据矩阵、场景/能力线索和证据边界，不复制源代码全文。

## 代码证据矩阵

| 问题域 | 代表路径 / 符号 | 说明 |
|---|---|---|
| 应用入口 | 待补充 | 服务/应用入口、handler、IDL、任务入口等。 |
| 代码组件 | 待补充 | 稳定代码模块 / 包 / 组件。 |
| 场景实现 | 待补充 | 按场景/平台能力拆分的 Implementation 线索。 |
| 持久化 | 待补充 | 具体 DAO / 表 / Redis Key 证据。 |
| 下游系统依赖 | 待补充 | 只记录下游业务服务 / PSM / SDK。 |

## 场景证据矩阵

| 场景 / 能力 | 入口 | 核心实现 | Data | Implementation | 证据边界 |
|---|---|---|---|---|---|
| 待补充 | 待补充 | 待补充 | 待补充 | 待补充 | 待确认 |

## 业务分层标签

- 业务模块：{', '.join(data.get('modules', [])) or '待补充'}
- 架构层：{', '.join(data.get('architecture_layers', [])) or '待补充'}
- 业务场景：{', '.join(data.get('business_scenarios', [])) or '待补充'}
- 平台能力：{', '.join(data.get('platform_capabilities', [])) or '待补充'}
- 系统：{', '.join(data.get('systems', [])) or '待补充'}

## 证据边界 / 待确认

- 代码证据绑定具体 repo/revision；后续代码变更需要重新核验。
- 代码事实可证明当前实现、字段、调用和约束；不得单独证明金额、费率、账户主体、费用归属、账期、资金方向或会计口径。
"""
    else:
        body = f"""# {title}

## 摘要

待摄入原文后补充。注意：本 Source 只沉淀结构化事实和摘要，不复制飞书/外部原文全文。

## 关键事实

- 待补充

## 业务分层标签

- 业务模块：{', '.join(data.get('modules', [])) or '待补充'}
- 架构层：{', '.join(data.get('architecture_layers', [])) or '待补充'}
- 业务场景：{', '.join(data.get('business_scenarios', [])) or '待补充'}
- 平台能力：{', '.join(data.get('platform_capabilities', [])) or '待补充'}
- 系统：{', '.join(data.get('systems', [])) or '待补充'}

## 原始资料

- RawRef: [{raw_path.relative_to(repo)}]({rel_from(raw_path, source_path)})

## 待确认 / 不确定性

- 待补充；若批量 ingest 发现明显来源冲突，用户确认前不得在此写成确定结论。
"""
    registry = repo / "wiki" / "registry" / f"REGISTRY-Sources-{raw_category}.md"
    source_rel_from_registry = rel_from(source_path, registry)
    raw_rel_from_registry = rel_from(raw_path, registry)
    return body, source_rel_from_registry, raw_rel_from_registry, str(registry.relative_to(repo)), data


def write_item(repo: Path, item: dict[str, object], ts: str, overwrite: bool) -> dict[str, object]:
    raw_path = item["raw_path"]
    source_path = item["source_path"]
    raw_category = str(item["raw_category"])
    title = str(item["title"])
    assert isinstance(raw_path, Path)
    assert isinstance(source_path, Path)

    body, source_rel_from_registry, raw_rel_from_registry, registry_rel, data = build_source_content(repo, item, ts)

    if source_path.exists() and not overwrite:
        action = "保留已有 Source，仅更新注册"
    else:
        ensure_dir(source_path.parent)
        source_path.write_text(frontmatter(data) + "\n" + body, encoding="utf-8")
        action = "创建 Source 骨架" if not overwrite else "覆盖 Source 骨架"

    registry = repo / registry_rel
    upsert_registry(registry, registry_row(title, source_rel_from_registry, raw_category, raw_rel_from_registry), source_rel_from_registry)
    return {
        "ok": True,
        "raw_ref": str(raw_path.relative_to(repo)),
        "source": str(source_path.relative_to(repo)),
        "registry": registry_rel,
        "action": action,
        "title": title,
    }


def main() -> None:
    ap = argparse.ArgumentParser(description="Create Source skeleton and update REGISTRY-Sources-* locally")
    ap.add_argument("--repo", required=True, help="LLM Wiki Git repository root")
    ap.add_argument("--raw", action="append", required=True, help="Raw reference path, absolute or relative to repo; repeat up to 10 times for batch ingest")
    ap.add_argument("--source-slug", help="Optional Source filename stem; defaults to original raw document name")
    ap.add_argument("--title", help="Source title; defaults to raw title")
    ap.add_argument("--overwrite", action="store_true", help="Overwrite existing Source skeleton")
    args = ap.parse_args()

    repo = Path(args.repo).expanduser().resolve()
    raw_args = args.raw or []
    batch = len(raw_args) > 1

    if len(raw_args) > MAX_BATCH_SIZE:
        print_json_exit({
            "ok": False,
            "needs_user_input": True,
            "reason": f"一次批量 ingest 最多支持 {MAX_BATCH_SIZE} 个 raw 引用。",
            "count": len(raw_args),
            "max": MAX_BATCH_SIZE,
        })
    if batch and (args.source_slug or args.title):
        forbidden = []
        if args.source_slug:
            forbidden.append("--source-slug")
        if args.title:
            forbidden.append("--title")
        print_json_exit({
            "ok": False,
            "needs_user_input": True,
            "reason": "批量 ingest 不能使用单文档标量参数，避免多个 raw 写入同一个 Source 或标题。",
            "forbidden_args": forbidden,
        })

    try:
        items = [prepare_item(repo, raw_arg, args.source_slug, args.title, idx) for idx, raw_arg in enumerate(raw_args, start=1)]
    except (FileNotFoundError, ValueError) as e:
        if batch:
            print_json_exit({"ok": False, "needs_user_input": True, "reason": str(e)})
        raise SystemExit(str(e)) from e

    conflicts = preflight_items(repo, items, batch, args.overwrite)
    if conflicts:
        print_json_exit({
            "ok": False,
            "needs_user_input": True,
            "reason": "ingest 预检发现结构冲突，未写入 Source 或 REGISTRY。",
            "batch": batch,
            "conflicts": conflicts,
        })

    ts = now()
    results = [write_item(repo, item, ts, args.overwrite) for item in items]

    if batch:
        registries = sorted({str(result["registry"]) for result in results})
        lines = [
            "---",
            "",
            f"### {ts} — INGEST-SOURCE-BATCH",
            "",
            "**操作**: 批量创建/注册 Source 骨架",
            f"**数量**: {len(results)} / {MAX_BATCH_SIZE}",
            "**raw -> Source**:",
        ]
        for result in results:
            lines.append(f"- `{result['raw_ref']}` -> `{result['source']}`")
        lines.append("**更新 REGISTRY**:")
        for registry in registries:
            lines.append(f"- `{registry}`")
        lines.extend([
            "**结构冲突**: 无",
            "**语义冲突**: 脚本未判断；LLM ingest workflow 需读取全批次原文并在明显冲突时阻塞语义写入。",
        ])
        append_log(repo, "\n".join(lines) + "\n")
        print(json.dumps({
            "ok": True,
            "batch": True,
            "count": len(results),
            "repo": str(repo),
            "results": results,
            "registries": registries,
            "warnings": [],
            "conflicts": [],
        }, ensure_ascii=False, indent=2))
    else:
        result = results[0]
        append_log(repo, f"---\n\n### {ts} — INGEST-SOURCE\n\n**操作**: {result['action']}\n**raw**: `{result['raw_ref']}`\n**source**: `{result['source']}`\n**registry**: `{result['registry']}`\n")
        print(json.dumps({"ok": True, "repo": str(repo), "raw_ref": result["raw_ref"], "source": result["source"], "registry": result["registry"], "action": result["action"]}, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
