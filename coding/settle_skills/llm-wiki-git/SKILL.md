---
name: llm-wiki-git
description: "LLM Wiki：在 Git 仓库中构建和维护计费/结算 LLM 知识库。Git-only 模式：raw/ 只保存原始资料引用（飞书 URL/token/doc_id、本地路径、外部链接等），wiki/ 全部为本地 Markdown，registry/ 为机器入口，INDEX.md 为人工导航。飞书/Lark 仅作为 raw 原始资料读取来源，不作为 wiki 存储后端，不同步、不发布、不写回飞书。支持 init、import、ingest、query、lint。触发词：llm wiki, git wiki, 计费知识库, 结算知识库, 清结算知识库, 知识库摄入, 知识库查询, wiki ingest, wiki query, wiki lint, wiki init, wiki import, 添加素材, 导入素材"
---

# LLM Wiki Git-only — 计费/结算

在 **Git 仓库** 中构建和维护一个面向计费、结算两个平台型系统的 LLM 结构化知识库。当前 skill 只支持 **Git 模式**：

- `raw/`：原始素材索引层，只保存引用和元数据，不复制飞书原文。
- `wiki/`：知识索引 / 关键摘要层，全部为本地 Markdown 文件；只沉淀可查询、可复用、可回溯的关键摘要和结构化索引，不复制 raw 全量信息。
- `wiki/registry/`：机器可读注册入口。
- `wiki/INDEX.md`：人工导航入口。
- `wiki/LOG.md`：操作日志。

飞书/Lark 只允许作为 `raw/` 原始资料读取来源：可以读取 raw 引用中的飞书文档内容，但**不把 wiki 写入飞书、不同步到飞书、不维护飞书发布目标、不依赖全局飞书 wiki 配置**。

## 前置检查

在执行 init/import/ingest/query/lint 之前，按操作需要检查本地依赖：

- 本地只读/写 `wiki/`、`raw/`、`registry/`、`INDEX.md`、`LOG.md` 时，不需要 `lark-cli`。
- 只有在 import / ingest / query / lint 需要读取 raw 引用中的飞书原始资料时，才检查 `lark-cli` 与认证。
- 如果飞书读取报 refresh token expired 或未认证，应提示用户运行：`! bytedcli feishu login --force-oauth`；必要时补充 `! bytedcli auth login --session --feishu`。

## Git-only 硬约束

- 仓库路径就是知识库入口；不读取、不要求 `~/.llm_wiki.setting.json`。
- 可选 `.llm-wiki/config.json` 只作为仓库内元数据，不是入口。
- `raw/` 不保存飞书原文，只保存 Markdown 引用文件和 YAML frontmatter。
- `wiki/` 下所有知识页都是本地 Markdown，是 Git 版本管理的唯一事实源。
- `wiki/registry/REGISTRY.md` 和 `REGISTRY-*.md` 是机器入口；`wiki/INDEX.md` 只给人导航。
- 本地 wiki 页面之间统一使用相对 Markdown 链接。
- raw 原始飞书 URL 只出现在 `raw/<分类>/*.md` 引用文件中；wiki 正文应引用 raw 文件或 Source 页面。
- wiki 是索引层，不是 raw 的镜像：Source / Scenario / Capability / Application 只保留关键摘要、判断框架、阅读路径、证据链接和不确定性；完整事实细节需要通过 raw 引用回查原文。
- 禁止创建飞书目录、飞书 wiki 节点、Drive shortcut、Wiki shortcut，禁止把 wiki 层写入飞书。
- 清理 wiki 业务页面时只清理文件，不删除固定 schema 目录；固定空目录必须保留 `.gitkeep`，避免 Git 提交后目录丢失。

## 目录与分层

### raw/ 按材料类型归档

`raw/` 回答“这份原始资料是什么类型”，不回答“它属于哪个平台系统”。默认分类：

```text
raw/
├── 代码仓库/          # 代码仓库引用，保存 repo/path/revision/扫描范围，不复制代码
├── 需求文档/
├── 技术方案/
├── ADR决策/
├── 系统白皮书/
├── 数据文档/
├── 接口文档/
├── 值班记录/
├── 复盘报告/
├── 团队规约/
├── 风险防控/
└── 日常分享/
```

**Source 分类唯一以 `raw/分类` 为准。** 禁止根据标题、摘要、关键词、主题猜 Source 分类。

### wiki/ 按计费/结算业务架构建模

业务架构类知识优先进入主干目录：

```text
wiki/
├── sources/                  # raw 的结构化摘要 / 证据层
├── modules/                  # 顶层平台系统，仅计费、结算；可更新既有两类，默认不新增
├── scenarios/                # 业务场景 / 长期查询入口
├── platform-capabilities/     # 能力 / 规则 / 流程
├── applications/             # 应用架构 / 系统职责 / PSM；代码仓库知识中也是 repo 入口
├── code-components/          # 代码模块 / 包 / 组件级语义索引
├── data/                     # 数据库表结构 / 索引 / Redis Key
├── implementations/          # 技术架构 / 技术实现
├── maps/                     # 跨层映射
├── overviews/                # 辅助：query 阶段沉淀的一页式综述
├── comparisons/              # 辅助：query 阶段沉淀的对比分析
├── query_feedback/           # 辅助：纠错反馈，按天汇总
└── registry/                 # 机器注册入口
```

默认关系链：

```text
Module/System -> Scenario -> PlatformCapability -> Application -> Data/Implementation -> Source -> RawRef
代码仓库链路：Application(repo入口) -> Scenario/PlatformCapability -> Implementation(场景/能力链路) -> Data(场景持久化) / CodeComponent(稳定代码模块) -> Source -> RawRef
```

业务拆解语义：

```text
业务场景 -> 平台能力 / 规则 / 流程 -> 应用职责 -> 技术实现
平台系统 -> 应用架构 -> 存储结构 -> 技术架构
```

默认且仅允许的顶层平台系统：`计费`、`结算`。可以基于 Source 证据更新这两类已有 `wiki/modules/` 页面；若非用户明确指定新增 module / 新增平台系统，不得创建新的 Module。

主干目录新增约束：

- `scenarios/` 与 `platform-capabilities/` 新增页面前，必须按第一性原理向用户说明并征询确认：为什么现有目录不足以承载、解决什么长期复用问题、覆盖哪些来源、证据是否充分、有哪些不确定性。
- `applications/` 必须具体到 PSM 维度；除非用户明确指定新增 PSM，否则不新增 Application。代码仓库知识中 Application 也是应用/代码仓库一对一入口，沉淀 repo URL/path、commit、仓库阅读入口和场景矩阵。
- `code-components/` 保存稳定代码模块 / 包 / 组件的语义索引，不按文件镜像代码；用于回答某块代码是什么、在哪、如何修改、关联哪些实现链路。
- `data/` 只保存 MySQL 表、Redis Key/表、索引、字段等数据库存储结构；代码仓库知识中 Data 应按场景/平台能力持久化拆分，说明该场景读写哪些表和字段语义差异。
- `implementations/` 保存领域模型和 PSM 交互链路细节；代码仓库知识中 Implementation 应按业务场景/平台能力拆分，使用系统时序图体现代码动作、上下游交互、领域动作、单据生命周期/状态流转。
- `implementations/` 的“系统依赖”只表示下游业务服务 / PSM / SDK；不得把 Kitex、MySQL/GORM、Redis、TCC、RocketMQ、Chronos 等中间件/运行时作为系统依赖沉淀。
- `maps/` 仅维护跨层映射；代码仓库知识优先维护 Application -> Scenario/PlatformCapability/Implementation、Implementation -> CodeComponent、Implementation -> Data。

### 计费/结算证据纪律

- 金额、币种、主体、账期、费用项、费率、计价口径、规则版本、结算周期、结算单状态、账单状态、出款状态等高风险事实必须能回溯到 Source/raw。
- 不得凭通用支付、清结算或账务经验补全计费/结算口径。
- 本地 wiki / Source 摘要未直接、完整回答问题时，应继续读取 raw 引用对应的原始资料；raw 为飞书引用时读取飞书原文，不应直接以“知识库未找到”结束。
- 证据不足或来源冲突时，必须显式标记“待确认 / UNKNOWN / 来源冲突”，不要静默合并。


## 代码仓库语义索引

当原始资料是代码仓库时，使用 `raw/代码仓库` + `source_kind=code_repo` 登记 repo/path/revision/扫描范围。raw 和 wiki 都不得复制代码全文，只保存语义摘要、路径、符号和证据边界。

代码仓库摄入默认建模：

```text
Application(repo入口)
  -> Scenario / PlatformCapability
  -> Implementation(按场景/能力拆分的执行链路)
  -> Data(按场景/能力拆分的持久化表/Key)
  -> CodeComponent(稳定代码模块/包/组件)
  -> Source(code snapshot)
  -> raw/代码仓库
```

要求：

- Application 是应用和代码仓库的一对一入口；先确认 Application，再确认对应 repo。
- CodeComponent 是稳定代码模块语义，不是 file-per-page。
- Implementation 必须按场景/平台能力拆分，体现不同入口、领域模型、系统时序图、下游服务、数据读写和状态流转差异。
- Data 必须按场景/能力持久化拆分；共享表可出现在多个 Data 页面，但必须说明使用阶段和证据。
- “系统依赖”只列下游业务服务 / PSM / SDK；中间件只作为代码机制出现，不作为依赖知识维护。
- 系统时序图优先使用 Mermaid `sequenceDiagram`，边必须能回溯到代码路径/符号或标记“待确认”。

## 配置与仓库识别

Git-only 模式按以下优先级选择知识库：

1. 用户本次请求明确指定的 Git 仓库路径。
2. 当前对话上下文已确定的仓库路径。
3. 从当前工作目录向上查找可识别的 LLM Wiki Git 仓库。

最小识别条件：

- 存在 `raw/`
- 存在 `wiki/registry/`
- 存在 `wiki/INDEX.md` 或 `wiki/registry/REGISTRY.md`

如果没有指定仓库路径，且当前目录无法识别为 LLM Wiki Git 仓库，应要求用户提供仓库路径。

## 五大操作

除 init 外，执行其他操作前必须读取仓库内 `AGENTS.md`。若 AGENTS 要求先查知识库或特定 skill 路由，必须优先遵循。

| 操作 | 说明 | 详细步骤 |
|------|------|---------|
| **init** | 创建 Git 知识库目录树、AGENTS.md、INDEX.md、LOG.md、REGISTRY 分册 | [init.md](references/workflows/init.md) |
| **import** | 将一个素材或一次最多 10 个飞书素材登记到 `raw/<分类>/`；raw 只保存引用和标签，不保存飞书原文 | [import.md](references/workflows/import.md) |
| **ingest** | 从一个 raw 引用或一次最多 10 个 raw 引用读取原文，创建/更新 Source 与业务分层页面，同步 REGISTRY；明显来源冲突时阻塞语义写入并提示用户确认 | [ingest.md](references/workflows/ingest.md) |
| **query** | 从 REGISTRY 定位本地 Markdown，必要时回查 raw 原文对应的飞书文档，综合回答 | [query.md](references/workflows/query.md) |
| **lint** | 检查 raw / Source / REGISTRY / INDEX / 业务分层链路一致性 | [lint.md](references/workflows/lint.md) |

## 业务分层更新阈值

- wiki 是知识合集索引，不是 Source 引用计数器。新 Source 命中已有 Scenario / PlatformCapability / Application / Data / Implementation / Module 时，先判断是否改变该页面的定义、边界、契约、规则、职责、证据边界、长期阅读路径或关键映射；没有改变时，只在 Source、必要的上层页面或 Map 中建立引用，不更新该页面正文。
- 越底层、越稳定的知识索引，更新阈值越高。若新 Source 只是使用或引用已有 Application / Data / Implementation，不改变职责边界、输入输出、适用范围、核心机制、存储结构或关键规则，则不得更新稳定页面正文。
- 新增 Scenario 或 PlatformCapability 前，必须按第一性原理向用户说明并征询确认：为什么现有目录不足以承载、解决什么长期复用问题、覆盖哪些来源、证据是否充分、有哪些不确定性。
- 若 Source 没有明确指定新增 PSM，不新增 Application；若没有明确指定新增平台系统，不新增 Module。

## 默认生成与变更约束

- 默认工作流是 `import -> ingest`。除非用户明确说明“只导入不摄入 / 暂不摄入”，否则 import 成功后应自动继续 ingest；批量 import 成功后默认把本批次成功导入的 raw_refs 一次性交给批量 ingest。
- import / ingest 支持一次性处理最多 10 个飞书文档；批量只是操作批次，不改变 schema：每个原始文档仍独立创建一个 raw_ref 和一个 Source，不创建合并 raw 或合并 Source。
- import 的第一步是尽量从原始资料读取标题与正文来确定 `raw/分类`：飞书 Wiki/Doc URL 必须优先用 `lark-cli` 或 `bytedcli feishu docs fetch-doc` 只读获取节点信息、标题和正文摘要，再按内容匹配当前仓库已有 raw 分类；飞书来源的 raw 文件名默认必须与飞书文档标题一致（仅清理文件系统非法字符）；只有无法读取、内容不足或无法稳定命中分类时，才咨询用户。
- ingest 的关键结果不只是创建 Source，而是保证 `raw -> Source -> REGISTRY` 对齐；写入 wiki 时只抽取关键摘要和索引信息，不搬运 raw 全量内容。
- 批量 ingest 必须先建立本批次证据矩阵；若多份文档或既有 wiki 页面在同一事实键、规则、接口、字段、状态、职责边界、版本口径上存在明显冲突，必须主动告警并阻塞语义写入，等待用户确认哪个来源/版本/适用范围为准后再继续。未解决前不得把冲突内容写成确定知识；必要时只保留 raw_ref / Source 骨架 / REGISTRY-Sources 的结构性记录，并在报告中说明业务页面未写入。
- Source 页面必须保留原始文档名：标题统一为 `Source：<原始文档名>`，文件名默认使用 `<原始文档名>.md`（仅清理文件系统非法字符），不要默认生成简称。
- 生成本地 Markdown 文件名时，默认忽略名称中的所有空格（含半角/全角可见空格），以避免检索和链接命中不稳定；正文标题仍保留原始名称展示。
- 除 `applications/`、`data/`、`modules/` 外，其他 `wiki/` 业务/辅助页面文件名应尽量与页面标题中的中文命名一致，优先使用中文文件名；只有文件系统不兼容、名称过长、历史兼容或用户明确指定时，才使用英文 slug 或缩写。
- ingest 默认不把全量 Source 回灌到 INDEX。
- query 默认只读；任何页面变更原则上需要用户确认。唯一例外：用户明确指出现有页面错误并要求修正。
- 长期专题归入 Scenario，能力 / 规则 / 流程归入 PlatformCapability，职责对象归入 Application。
- Scenario 页面优先采用“专题入口”形态：元信息/主题范围/证据边界、摘要、覆盖材料矩阵、推荐阅读路径、关键决策或边界提醒、关联能力、关联 Source；不把每份 raw 的完整流程和字段细节搬进 Scenario。
- Overview / Comparison 创建前必须说明长期价值、证据来源、影响边界与不确定性，并等待用户确认。
- Query Feedback 单独维护在 `wiki/query_feedback/`，注册到 `REGISTRY-QueryFeedback`，按天汇总。
- AGENTS 只记录结构、规则和例外，不记录具体业务事实；业务知识沉淀到 wiki 页面。

## Git 命令速查

| 操作 | 命令 |
|------|------|
| 初始化仓库 | `python3 scripts/init_git.py --repo <REPO_ROOT> --wiki-name <NAME> --with-billing-settlement-defaults` |
| 初始化仓库（旧参数兼容） | `python3 scripts/init_git.py --repo <REPO_ROOT> --wiki-name <NAME> --with-bytepay-defaults` |
| 导入 raw 引用 | `python3 scripts/import_raw.py --repo <REPO_ROOT> --title "<标题>" --raw-category 技术方案 --source-kind lark_wiki --lark-url "<URL>" --slug <slug>` |
| 批量导入 Feishu raw 引用（最多 10 个） | `python3 scripts/import_raw.py --repo <REPO_ROOT> --source-kind lark_auto --lark-url "<URL1>" --lark-url "<URL2>"` |
| 创建 Source 骨架并注册 | `python3 scripts/ingest_source.py --repo <REPO_ROOT> --raw raw/技术方案/<原始文档名>.md` |
| 批量创建 Source 骨架并注册（最多 10 个） | `python3 scripts/ingest_source.py --repo <REPO_ROOT> --raw raw/技术方案/<文档1>.md --raw raw/技术方案/<文档2>.md` |
| 本地结构 lint | `python3 scripts/lint_git.py --repo <REPO_ROOT> --json` |
| 查看工作区 | `git status --short` |
| 查找页面 | `rg -n "关键词" wiki raw` |
| 查看分层页面 | `find wiki/modules wiki/scenarios wiki/platform-capabilities wiki/applications wiki/data wiki/implementations wiki/maps -type f -name '*.md'` |
| 创建 raw 引用 | 在 `raw/<分类>/<slug>.md` 写入 YAML frontmatter，正文只放原始链接与备注 |
| 查看变更 | `git diff -- raw wiki .llm-wiki AGENTS.md` |

## Lark 操作命令速查

| 操作 | 命令 |
|------|------|
| 读取文档 | `bytedcli --json feishu docs fetch-doc <URL_OR_DOC_TOKEN>` 或 `lark-cli docs +fetch --as user --doc <DOC_ID>` |
| 搜索文档 | `lark-cli docs +search --query "关键词"` |

## scripts/ 本地工具

`scripts/` 下的工具只做本地 Git 文件维护，不访问或写入飞书：

- `scripts/init_git.py`：创建 Git-only 知识库骨架、AGENTS、INDEX、LOG、REGISTRY 分册和可选计费/结算默认模块。
- `scripts/import_raw.py`：创建/更新一个或一次最多 10 个飞书 `raw/<分类>/<slug>.md` 引用文件，并追加 `wiki/LOG.md`；批量模式只检测结构冲突，不做语义判断。
- `scripts/ingest_source.py`：从一个或一次最多 10 个 raw 引用创建 `wiki/sources/<分类>/<原始文档名>.md` Source 骨架，并注册到 `REGISTRY-Sources-<分类>.md`；批量模式只做 raw -> Source -> REGISTRY bookkeeping。
- `scripts/lint_git.py`：只读检查 raw / Source / REGISTRY / 链接的一致性；语义冲突与过时检测仍需按 lint workflow 由 LLM 深检。

这些脚本不替代 LLM 的抽取、综合、分层建模能力；ingest/query/lint 的语义判断仍必须遵循 workflow 和仓库 `AGENTS.md`。

## 参考文档

- [Wiki Schema](references/wiki-schema.md)
- [Git Adapter](references/adapter/git.md)
- [Init Templates](references/templates/init.md)
- [Page Templates](references/templates/pages.md)
- [init workflow](references/workflows/init.md)
- [import workflow](references/workflows/import.md)
- [ingest workflow](references/workflows/ingest.md)
- [query workflow](references/workflows/query.md)
- [lint workflow](references/workflows/lint.md)
