#!/usr/bin/env python3
"""Initialize a Git-only LLM Wiki repository.

This script creates only local files. It never creates or writes Lark/Feishu
folders, docs, wiki nodes, or shortcuts.
"""
from __future__ import annotations

import argparse
import json
import subprocess
from pathlib import Path

from common import (
    BILLING_SETTLEMENT_MODULES,
    DEFAULT_MAPS,
    DEFAULT_RAW_SUBDIRS,
    REGISTRY_FIXED,
    TRACKED_EMPTY_WIKI_DIRS,
    WIKI_DIRS,
    append_log,
    ensure_dir,
    ensure_gitkeep,
    now,
    write_if_missing,
)


def registry_page(title: str) -> str:
    return f"""# {title}

| 标题 | 类型 | Doc | 分类/目录 | 最后更新 | 关联 |
|---|---|---|---|---|---|
"""


def agents_template(wiki_name: str) -> str:
    return f"""# AGENTS

本文档只记录这套 LLM Wiki 的结构、规则和例外，不记录具体业务事实。具体业务知识应沉淀到 Source / Module / Scenario / PlatformCapability / Application / CodeComponent / Data / Implementation / Map / Overview / Comparison / Query Feedback。

当前知识库 `{wiki_name}` 采用 Git-only 模式：raw 只保存原始资料引用，wiki 全部为本地 Markdown，INDEX 给人看，REGISTRY 给机器读。飞书只作为 raw 原始资料读取来源，不作为 wiki 存储后端。

## 核心规则

- INDEX 是人工导航页，不承担全量注册；REGISTRY 及其分册才是机器侧全量入口。
- Source 分类唯一以 raw 为准：每个 Source 的分类必须与原始素材所在的 `raw/分类` 一致。
- Source 页面必须保留原始文档名：标题统一为 `Source：<原始文档名>`，文件名默认使用 `<原始文档名>.md`（仅清理文件系统非法字符）。
- 所有事实性内容必须能回溯到 raw 或 Source；综合判断必须和原始事实区分开。
- 默认增量维护；只有页面严重漂移、结构损坏或重建 REGISTRY 分册时才整页覆盖。
- 先定 raw，再谈 wiki：import、ingest、registry、lint 的上游依据都是 `raw/`，不是 INDEX。
- wiki 是知识合集的索引层 / 关键摘要层，不是 raw 原文对应的飞书文档镜像；只沉淀可查询、可复用、可回溯的关键摘要、结构化关系、证据边界、阅读路径和不确定性。

## Git 模式业务分层默认规则

- `raw/` 按原始材料类型组织，不按业务架构层拆 raw。
- `wiki/` 按业务架构模型组织：平台系统 → 业务场景 → 平台能力 → 应用 → 数据 / 技术实现。
- 默认链路：`Module/System -> Scenario -> PlatformCapability -> Application -> Data/Implementation -> Source -> RawRef`。
- 默认平台系统仅包括：计费、结算；可以更新既有两类，但除非用户明确指定新增 module / 新增平台系统，不得创建第三类 Module。
- 旧专题、旧概念、旧实体层已废弃：长期查询入口归入 `scenarios/`，能力/规则/流程归入 `platform-capabilities/`，PSM/职责对象归入 `applications/`。

## 主干目录职责

- `modules/`：顶层平台系统，仅计费、结算两类；可更新，不默认新增。
- `scenarios/`：计费 / 结算领域的业务场景和长期查询入口，例如计费规则生效、账单生成、账单重算、结算单生成、结算出款、对账差异、差错补偿等；可以由多个 Source 支撑，沉淀跨系统、跨层复用的术语、模式和阅读路径。
- `platform-capabilities/`：计费 / 结算平台对外或业务可复用的能力、规则和流程，例如费用项定义、计费规则、账单生成、结算单生成、结算周期、差错处理等。
- `applications/`：PSM 维度的应用职责、边界、上下游、接口、任务和消息；代码仓库知识中 Application 也作为应用/代码仓库一对一入口维护，沉淀仓库地址、验证 commit、阅读路径和场景矩阵；除非用户明确指定新增 PSM，默认不新增 Application。
- `code-components/`：代码模块 / 包 / 组件级语义索引，回答某块稳定代码模块是什么、在哪、如何修改、关联哪些实现链路；不得按文件逐个镜像代码。
- `data/`：数据库表结构、索引信息、Redis Key、字段等具体存储结构；代码仓库知识中的 Data 应按场景/平台能力持久化拆分，说明该场景读写哪些表、哪些字段语义不同；共享表可以出现在多个场景 Data 中，但必须说明使用阶段和证据。
- `implementations/`：领域模型和系统 PSM 之间的交互链路细节；Implementation 是动态链路视图，原则上按业务场景或平台能力拆分，需体现不同场景/能力的入口、领域模型、系统时序图、下游系统依赖、系统实现、数据读写、状态/幂等/补偿差异。
- `implementations/` 中的“系统依赖”只表示下游业务服务 / PSM / SDK 服务；Kitex、MySQL/GORM、Redis、TCC、RocketMQ、Chronos、配置、日志、metrics 等中间件/运行时依赖不得放入“系统依赖”，只在必要时作为代码机制出现。
- `maps/`：只维护跨层映射；代码知识优先维护 Application -> Scenario/PlatformCapability/Implementation、Implementation -> CodeComponent、Implementation -> Data 的索引。
- `overviews/`、`comparisons/` 是辅助知识层，服务主干，不替代主干；主要由 query 阶段在用户确认后产生，ingest 阶段默认不主动创建。

## 新增与更新阈值

- 业务分层页面是知识合集索引，不是 Source 引用计数器。新 Source 命中已有 Scenario / PlatformCapability / Application / Data / Implementation / Module 时，先判断是否改变该页面的定义、边界、契约、规则、职责、证据边界、长期阅读路径或关键映射；没有改变时，只在 Source、必要的上层页面或 Map 中建立引用，不更新该页面正文。
- 越底层、越稳定的知识索引，更新阈值越高。若新 Source 只是使用或引用已有 Application / Data / Implementation，不改变职责边界、输入输出、适用范围、核心机制、存储结构或关键规则，则不得更新稳定页面正文。
- 新增 Scenario 或 PlatformCapability 前，必须按第一性原理向用户说明并征询确认：为什么现有目录不足以承载、解决什么长期复用问题、覆盖哪些来源、证据是否充分、有哪些不确定性。
- 若 Source 没有明确指定新增 PSM，不新增 Application；若没有明确指定新增平台系统，不新增 Module。

## 命名与格式规则

- 飞书来源导入时，必须优先用 `lark-cli` 只读获取标题和正文摘要，根据内容匹配当前仓库已有 raw 分类；无法读取、内容不足或无法稳定命中分类时，再咨询用户。
- 飞书 raw 文件名默认必须与飞书文档标题一致，仅清理文件系统非法字符；“白皮书”类文档默认归入 `raw/系统白皮书`。
- Source 标题统一为 `Source：<原始文档名>`；Source 文件默认位于 `wiki/sources/<分类>/<原始文档名>.md`。
- PSM Application 文件名使用 PSM 下划线格式，例如 `caijing.bytepay.charge` → `caijing_bytepay_charge.md`。
- Application 页面必须保留 `## 关联数据`、`## 关联平台能力`、`## 证据来源` 三个章节；已废弃层的历史引用应忽略或移除。
- Data 页面应在前置 `## 来源` 章节统一声明表结构来源；金额、币种、主体、账期、费用项、费率、规则版本、结算周期、结算单状态等高风险字段必须能回溯 Source/raw；字段/索引表默认继承该来源，不逐行重复来源，只有字段级来源差异、冲突或待确认时才在备注中说明。

## 默认流程规则

- 默认流程是 `import -> ingest`。除非用户明确说明“只导入不摄入 / 暂不摄入”，否则 import 成功后默认继续 ingest。
- import 的关键结果是把资料放到正确的 `raw/分类`。
- ingest 的关键结果是保证 `raw -> Source -> REGISTRY` 对齐，并在确有长期复用价值和证据变化时维护业务分层页面。
- query 默认读取顺序是：`REGISTRY -> 对应 REGISTRY 分册 -> 具体页面`；必要时回查 raw 原文对应的飞书文档。

## Query 与纠错回流规则

- query 阶段默认只读，不应因为“顺手修一下”直接写回。
- query 中涉及任何页面变更时，原则上都应先获得用户明确确认。
- 唯一例外：用户明确指出现有页面内容有误并要求修正，可以直接修改对应页面。
- Query Feedback 单独维护在 `wiki/query_feedback/`，并注册到 `REGISTRY-QueryFeedback`；按天维度维护。

## 日常分享规则

- 用户只给一句话或几句话、但没有正式文档/附件/链接时，默认先落到 `raw/日常分享`，并按天维度维护。
- 即使内容涉及风险、防控、值班、技术方案等主题，也不能仅按关键词猜到其他 raw 目录。
"""


def index_template() -> str:
    return """# INDEX

LLM Wiki 人工导航入口。

> 全量注册请查看 `wiki/registry/REGISTRY.md` 和 `REGISTRY-*` 分册；此处只保留常用入口和阅读路径。

## 目录说明

- `wiki/code-components/`：代码模块 / 包 / 组件级语义索引。

## 推荐入口

- [REGISTRY](registry/REGISTRY.md)
- [计费](modules/charge.md)
- [结算](modules/settlement.md)
"""


def root_registry(raw_subdirs: list[str]) -> str:
    ts = now()
    rows = []
    for sub in raw_subdirs:
        rows.append(f"| [Sources-{sub}](REGISTRY-Sources-{sub}.md) | raw/{sub} Source | 0 | {ts} |")
    fixed_map = [
        ("Modules", "业务模块"),
        ("Scenarios", "业务场景"),
        ("PlatformCapabilities", "平台能力"),
        ("Applications", "应用架构"),
        ("CodeComponents", "代码组件"),
        ("Data", "数据架构"),
        ("Implementations", "技术实现"),
        ("Maps", "跨层映射"),
        ("Overviews", "综述"),
        ("Comparisons", "对比"),
        ("QueryFeedback", "纠错反馈"),
    ]
    for name, desc in fixed_map:
        rows.append(f"| [{name}](REGISTRY-{name}.md) | {desc} | 0 | {ts} |")
    return "# REGISTRY\n\n机器注册入口。INDEX 只做人类导航，不承载全量注册。\n\n| 分册 | 说明 | 数量 | 最后更新 |\n|---|---|---:|---|\n" + "\n".join(rows) + "\n"


def rel_from(target: Path, base_file: Path) -> str:
    import os
    return os.path.relpath(target, start=base_file.parent)


def append_registry_row(registry: Path, title: str, kind: str, target: Path, category: str, related: str = "") -> None:
    ensure_dir(registry.parent)
    if not registry.exists():
        registry.write_text(registry_page(registry.stem), encoding="utf-8")
    doc_rel = rel_from(target, registry)
    text = registry.read_text(encoding="utf-8", errors="replace")
    lines = [ln for ln in text.splitlines() if f"]({doc_rel})" not in ln]
    row = f"| {title} | {kind} | [{doc_rel}]({doc_rel}) | {category} | {now()} | {related} |"
    lines.append(row)
    registry.write_text("\n".join(lines).rstrip() + "\n", encoding="utf-8")


def main() -> None:
    ap = argparse.ArgumentParser(description="Initialize a Git-only LLM Wiki repository")
    ap.add_argument("--repo", required=True, help="Repository root")
    ap.add_argument("--wiki-name", default="llm-wiki", help="Wiki name")
    ap.add_argument("--raw-subdirs", default=",".join(DEFAULT_RAW_SUBDIRS), help="Comma-separated raw subdirs")
    ap.add_argument("--with-billing-settlement-defaults", action="store_true", help="Create default billing/settlement module/map skeleton pages")
    ap.add_argument("--with-bytepay-defaults", action="store_true", help="Deprecated alias of --with-billing-settlement-defaults")
    ap.add_argument("--git-init", action="store_true", help="Run git init if .git is missing")
    args = ap.parse_args()

    repo = Path(args.repo).expanduser().resolve()
    raw_subdirs = [x.strip() for x in args.raw_subdirs.split(",") if x.strip()]
    ensure_dir(repo)

    if args.git_init and not (repo / ".git").exists():
        subprocess.run(["git", "init"], cwd=repo, check=True)

    created: list[str] = []
    for sub in raw_subdirs:
        ensure_dir(repo / "raw" / sub)
    for d in WIKI_DIRS:
        ensure_dir(repo / "wiki" / d)
    # Git does not track empty directories. Keep intentionally empty layer dirs
    # visible after users clear generated pages but want the schema directories retained.
    for d in TRACKED_EMPTY_WIKI_DIRS:
        ensure_gitkeep(repo / "wiki" / d)

    if write_if_missing(repo / "AGENTS.md", agents_template(args.wiki_name)):
        created.append("AGENTS.md")
    if write_if_missing(repo / "wiki" / "INDEX.md", index_template()):
        created.append("wiki/INDEX.md")
    if write_if_missing(repo / "wiki" / "LOG.md", "# LOG\n\n最新操作在最下方。\n"):
        created.append("wiki/LOG.md")
    if write_if_missing(repo / "wiki" / "registry" / "REGISTRY.md", root_registry(raw_subdirs)):
        created.append("wiki/registry/REGISTRY.md")

    for sub in raw_subdirs:
        path = repo / "wiki" / "registry" / f"REGISTRY-Sources-{sub}.md"
        if write_if_missing(path, registry_page(f"REGISTRY-Sources-{sub}")):
            created.append(str(path.relative_to(repo)))
    for name in REGISTRY_FIXED:
        path = repo / "wiki" / "registry" / name
        if write_if_missing(path, registry_page(name[:-3])):
            created.append(str(path.relative_to(repo)))

    if args.with_billing_settlement_defaults or args.with_bytepay_defaults:
        for slug, title in BILLING_SETTLEMENT_MODULES:
            content = f"---\ntype: module\ntitle: \"Module: {title}\"\ncreated_at: \"{now()}\"\nupdated_at: \"{now()}\"\n---\n\n# Module: {title}\n\n## 范围与边界\n\n- 覆盖范围：待补充\n- 不覆盖：待补充\n\n## 核心场景\n\n待补充\n\n## 证据来源\n\n待补充\n"
            path = repo / "wiki" / "modules" / f"{slug}.md"
            if write_if_missing(path, content):
                created.append(str(path.relative_to(repo)))
            append_registry_row(repo / "wiki" / "registry" / "REGISTRY-Modules.md", f"Module: {title}", "module", path, "modules")
        for slug, title in DEFAULT_MAPS:
            content = f"---\ntype: map\ntitle: \"Map: {title}\"\ncreated_at: \"{now()}\"\nupdated_at: \"{now()}\"\n---\n\n# Map: {title}\n\n| 上层对象 | 关系 | 下层对象 | 证据 |\n|---|---|---|---|\n"
            path = repo / "wiki" / "maps" / f"{slug}.md"
            if write_if_missing(path, content):
                created.append(str(path.relative_to(repo)))
            append_registry_row(repo / "wiki" / "registry" / "REGISTRY-Maps.md", f"Map: {title}", "map", path, "maps")

    ensure_dir(repo / ".llm-wiki")
    config = repo / ".llm-wiki" / "config.json"
    if write_if_missing(config, json.dumps({
        "wiki_name": args.wiki_name,
        "storage_type": "git",
        "raw_subdirs": raw_subdirs,
        "created_at": now(),
    }, ensure_ascii=False, indent=2) + "\n"):
        created.append(".llm-wiki/config.json")

    append_log(repo, f"---\n\n### {now()} — INIT\n\n**操作**: 初始化 Git-only LLM Wiki\n**创建文件**: {len(created)} 个\n")
    print(json.dumps({"ok": True, "repo": str(repo), "created": created}, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
