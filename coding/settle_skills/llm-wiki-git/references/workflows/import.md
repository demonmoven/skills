# Import — 导入原始素材引用

## 目标

import 的首要目标是把原始资料登记到正确的 `raw/分类`。Git-only 模式下，raw 只保存引用和元数据，不复制飞书原文，不创建飞书快捷方式，不移动飞书文档。

批量 Feishu import 的目标与逐个导入一致：为每份原始文档创建独立 raw_ref。批量只是操作层能力，用于一次处理同一知识块相关的多份文档，并为后续 batch ingest 暴露潜在冲突；不创建合并 raw 文件。

## 前置条件

- 已确定 Git 仓库路径，或当前目录可识别为 LLM Wiki Git 仓库。
- 已读取仓库内 `AGENTS.md`。
- 用户提供以下之一：一个飞书文档 URL/token、一次最多 10 个飞书文档 URL/token、本地文件路径、外部 URL、短文本输入。

## raw 分类原则

- raw 子目录以当前仓库中实际存在的目录为准。
- import 的第一步必须是确定目标 `raw/分类`。
- 对飞书 docx/wiki URL 或 token，必须优先用 `lark-cli` 或 `bytedcli feishu docs fetch-doc` 只读获取标题、节点元数据和正文内容，再根据内容匹配 raw 分类；不要只凭 URL 或 token 询问用户。
- 只有在读取工具不可用/未认证/无权限、原文内容不足，或分类匹配置信度不足时，才向用户咨询 raw 分类。
- raw 按材料类型归档，不按平台系统、主题或架构层归档。
- 平台系统、架构层、场景、平台能力/规则/流程、系统等作为 frontmatter 标签记录，不作为 raw 子目录。

默认分类：`需求文档`、`技术方案`、`ADR决策`、`系统白皮书`、`数据文档`、`接口文档`、`值班记录`、`复盘报告`、`团队规约`、`风险防控`、`日常分享`、`代码仓库`。

## 素材类型识别

| 用户输入 | source_kind | 处理 |
|---|---|---|
| 单个飞书 docx/wiki URL 或 token | `lark_doc` / `lark_wiki` / `lark_auto` | 优先只读获取标题和正文内容，根据内容匹配 raw 分类；无法读取或无法稳定命中时再咨询用户；随后创建 raw 引用 |
| 1–10 个飞书 docx/wiki URL 或 token | `lark_auto`（推荐）/ `lark_doc` / `lark_wiki` | 逐文档只读获取标题、正文和元数据，逐文档确定 raw 分类与文件名；全批次预检成功后写入 raw 引用 |
| 本地文件路径 | `local_file` | 默认只记录路径；用户明确要求时才复制到仓库或 Git LFS |
| 代码仓库路径 / Git URL | `code_repo` | 登记 repo URL/path、branch/commit、Application/PSM、扫描范围；raw 不复制代码全文 |
| 外部 HTTP(S) URL | `external_url` | 记录 URL、标题、抓取时间；默认不缓存全文 |
| 几句话口述 | `note` | 写入 `raw/日常分享/YYYY-MM-DD.md`，可保存用户原话 |

## 批量 Feishu 导入

- 支持一次性导入 1–10 个飞书 doc/wiki URL；超过 10 个必须拆批。
- 批量导入当前只适用于飞书来源；本地文件、外部 URL、note 仍按单素材处理。
- 每个飞书文档独立读取标题、节点元数据和正文摘要，独立确定 raw 分类和 raw 文件名。
- `--raw-category` 若由用户显式提供，可作为本批次所有文档的共同 raw 分类；否则逐文档自动分类。
- 批量导入必须先完成全批次预检；任一文档无法读取、无法分类或发生路径/来源结构冲突时，不应部分写入。
- 批量导入成功后，除非用户明确“只导入不摄入”，默认把所有 raw_ref 交给批量 ingest。
- 批量 import 不判断自然语言语义冲突；语义冲突由后续批量 ingest 读取全批次原文后判断。

## 步骤

可用本地脚本创建/更新 raw 引用。

单文档示例：

```bash
python3 scripts/import_raw.py \
  --repo <REPO_ROOT> \
  --title "<原始标题>" \        # 可省略：飞书来源会优先自动获取
  --raw-category 技术方案 \      # 可省略：飞书来源会优先按标题/正文自动匹配
  --source-kind lark_doc \
  --lark-url "<飞书 URL>" \
  --slug <stable-slug> \        # 可选；飞书来源默认使用飞书文档标题作为 raw 文件名
  --modules "计费" \
  --architecture-layers "业务架构,应用架构,数据架构" \
  --business-scenarios "账单生成" \
  --platform-capabilities "计费规则,账单生成" \
  --systems "caijing.bytepay.charge"
```

批量 Feishu 示例：

```bash
python3 scripts/import_raw.py \
  --repo <REPO_ROOT> \
  --source-kind lark_auto \
  --lark-url "<飞书 URL 1>" \
  --lark-url "<飞书 URL 2>" \
  --lark-url "<飞书 URL 3>" \
  --modules "计费" \
  --business-scenarios "账单生成"
```

脚本只写 `raw/<分类>/<slug>.md` 和 `wiki/LOG.md`。飞书来源会用可用只读工具读取标题/正文用于分类，但不复制原文、不写入飞书。脚本完成后，除非用户明确“只导入不摄入”，继续执行 ingest；批量模式下继续执行批量 ingest。

### 1. 读取仓库结构

- 使用用户指定仓库路径或当前仓库。
- 扫描 `raw/` 下一级目录，得到可选 raw 分类。
- 读取 `AGENTS.md`，确认分类和维护规则。

### 2. 获取标题/内容并确定 raw 分类和 slug

- 若素材是飞书 docx/wiki URL 或 token：
  1. 检查可用读取工具及认证。若 bytedcli 返回 refresh token expired，提示用户运行 `! bytedcli feishu login --force-oauth`。
  2. Wiki URL 可先读取节点元数据，拿到 `title`、`obj_token/doc_id`、`obj_type`、`node_token/wiki_token` 等。
  3. 再读取文档标题和正文内容用于分类判断。读取只是事实核验，不写回飞书，不把原文复制进 raw/wiki。
  4. 用标题 + 正文片段匹配当前仓库已有 raw 分类。推荐高置信命中规则：
     - `需求文档`：PRD、需求、产品需求、验收标准、用户故事、原型、交互、排期。
     - `技术方案`：技术方案、概要设计、详细设计、架构设计、实现方案、接口改造、流程图、时序图、RPC、DB。
     - `ADR决策`：ADR、决策、Decision、备选方案、取舍、结论。
     - `系统白皮书`：白皮书、系统架构、总体架构、A2/A3 架构、架构全景。
     - `数据文档`：数据模型、表结构、字段、指标、数仓、SQL、ETL。
     - `接口文档`：接口文档、API、入参、出参、request、response、endpoint。
     - `值班记录`：值班、oncall、报警处理、值班记录、排查记录。
     - `复盘报告`：复盘、事故、RCA、根因、改进项、postmortem。
     - `团队规约`：规范、规约、SOP、流程、研发规范、团队约定。
     - `风险防控`：风险、防控、安全、越权、漏洞、扫描、治理。
     - `日常分享`：分享、笔记、零散记录、会议纪要。
  5. 若最高分明显高于其他分类，则自动使用该分类，并在 `wiki/LOG.md` 记录分类依据；否则列出候选分类和证据，向用户确认。
- 对批量 Feishu 输入，以上步骤逐文档执行；若任一文档需要用户判断，则停止本批次写入，统一报告需要补充的信息。
- 对本地文件、外部 URL、短文本输入，仍根据可读内容和用户描述判断；拿不准则问用户。
- 飞书来源的 raw 文件名默认使用飞书文档标题：`raw/<分类>/<飞书标题>.md`；仅清理 `/\:*?"<>|` 等文件系统非法字符，并保留中文标题。
- 非飞书来源仍可生成稳定 slug：小写英文、数字、短横线；中文标题可生成语义英文 slug。
- 若目标 raw 文件已存在，优先增量更新 frontmatter，不覆盖正文备注；若已存在 raw 指向不同飞书来源，批量模式必须阻塞并要求用户确认。

### 3. 创建 raw 引用文件

飞书来源示例：

```markdown
---
type: raw_ref
title: "抖音支付计收费-白皮书"
raw_category: "系统白皮书"
source_kind: "lark_doc"
lark_url: "https://..."
doc_id: "xxx"
wiki_token: "yyy"
obj_type: "docx"
modules:
  - 计费
architecture_layers:
  - 业务架构
  - 应用架构
  - 数据架构
business_scenarios:
  - 账单生成
  - 费用项计费
platform_capabilities:
  - 计费规则
  - 账单生成
systems:
  - caijing.bytepay.charge
imported_at: "YYYY-MM-DD HH:mm"
updated_at: "YYYY-MM-DD HH:mm"
---

# 抖音支付计收费-白皮书

- 原始飞书地址：<https://...>
- 备注：本文件只保存 raw 引用，不保存原文。
```

本地文件 / 外部链接 / note 按相同结构记录 `source_kind` 和路径/URL/原文。

### 4. 追加 LOG

在 `wiki/LOG.md` 记录：

- 素材标题；批量模式列出全部标题
- source_kind
- raw 分类
- raw 引用文件路径
- 是否继续 ingest
- 批量模式的结构冲突预检结果

单文档使用 `IMPORT`，批量使用 `IMPORT-BATCH`。

### 5. 默认继续 ingest

除非用户明确说“只导入不摄入 / 暂不摄入”，否则 import 成功后默认继续执行 ingest。批量导入成功后，默认把本批次全部 raw_refs 作为同一 ingest 批次处理；后续 ingest 需要统一检查多文档之间的升级、替代和冲突关系。

## 禁止事项

- 禁止创建 Wiki shortcut / Drive shortcut。
- 禁止移动或复制飞书原文。
- 禁止把飞书内容粘贴进 raw 引用文件。
- 禁止根据标题/关键词把 Source 分类改到不同 raw 目录。
- 禁止在批量预检失败后部分写入 raw_ref。


## 代码仓库导入

代码仓库作为原始素材时，使用 `source_kind=code_repo`，默认登记到 `raw/代码仓库/`。raw 只保存仓库引用、版本和扫描边界，不复制源代码。

示例：

```bash
python3 scripts/import_raw.py \
  --repo <REPO_ROOT> \
  --source-kind code_repo \
  --title "<应用>代码仓库快照" \
  --raw-category 代码仓库 \
  --repo-url "https://code.example.com/org/repo" \
  --local-path "/path/to/repo" \
  --module-path "module/path" \
  --branch master \
  --commit "<commit>" \
  --application "<Application>" \
  --systems "<system>" \
  --include-path "handler.go" \
  --include-path "core/"
```

导入预检应关注同一 Application 指向多个 repo、同一 repo 不同 revision 未标注版本、扫描范围缺失等问题。后续 ingest 由 LLM 读取代码并生成 Application / CodeComponent / Implementation / Data / Map。
