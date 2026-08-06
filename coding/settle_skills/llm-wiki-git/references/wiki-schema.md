# Wiki Schema — Git-only 知识库结构定义

本文档定义计费/结算 LLM Wiki Git-only 模式的目录结构、页面类型规范和注册规则。

## 核心结构

```text
<repo>/
├── AGENTS.md
├── raw/
│   ├── 需求文档/
│   ├── 技术方案/
│   ├── ADR决策/
│   ├── 系统白皮书/
│   ├── 数据文档/
│   ├── 接口文档/
│   ├── 值班记录/
│   ├── 复盘报告/
│   ├── 团队规约/
│   ├── 风险防控/
│   └── 日常分享/
├── wiki/
│   ├── INDEX.md
│   ├── LOG.md
│   ├── registry/
│   │   ├── REGISTRY.md
│   │   ├── REGISTRY-Sources-<分类>.md
│   │   ├── REGISTRY-Modules.md
│   │   ├── REGISTRY-Scenarios.md
│   │   ├── REGISTRY-PlatformCapabilities.md
│   │   ├── REGISTRY-Applications.md
│   │   ├── REGISTRY-CodeComponents.md
│   │   ├── REGISTRY-Data.md
│   │   ├── REGISTRY-Implementations.md
│   │   ├── REGISTRY-Maps.md
│   │   ├── REGISTRY-Overviews.md
│   │   ├── REGISTRY-Comparisons.md
│   │   └── REGISTRY-QueryFeedback.md
│   ├── sources/
│   ├── modules/
│   ├── scenarios/
│   ├── platform-capabilities/
│   ├── applications/
│   ├── code-components/
│   ├── data/
│   ├── implementations/
│   ├── maps/
│   ├── overviews/
│   ├── comparisons/
│   └── query_feedback/
└── .llm-wiki/                 # 可选
    └── config.json            # 可选元数据，不是入口
```

## 双入口规则

- `wiki/INDEX.md`：人工导航页，只放说明、推荐入口、常用页面，不承载全量 Source 注册。
- `wiki/registry/REGISTRY.md`：机器入口，登记分册入口、统计、状态。
- `wiki/registry/REGISTRY-*.md`：机器分册，登记对应目录的全量页面。

默认检索顺序：

```text
REGISTRY.md -> REGISTRY-* -> 具体 Markdown 页面 -> 必要时 raw 原文对应的飞书文档
```

## raw/ 职责

- 保存原始资料引用，不保存飞书原文。
- 是 Source 分类的唯一上游真相。
- `raw/` 按材料类型组织，不按平台系统、主题或架构层组织。
- 每个 raw 文件必须是 Markdown + YAML frontmatter。

raw 引用文件必须至少包含：

```yaml
---
type: raw_ref
title: "<原始标题>"
raw_category: "技术方案"
source_kind: "lark_doc|lark_wiki|external_url|local_file|note|code_repo"
lark_url: "https://..."      # 飞书来源可选但推荐
doc_id: "<可选，docx obj_token>"
wiki_token: "<可选，wiki node token>"
obj_type: "docx|file|sheet|url|note"
repo_url: "<code repo URL，可选>"
local_path: "<本地代码仓库路径，可选>"
module_path: "<模块路径，可选>"
branch: "<分支，可选>"
commit: "<commit/revision，可选>"
modules: []                  # 仅允许默认计费/结算，除非用户明确新增平台系统
architecture_layers: []
business_scenarios: []
platform_capabilities: []
systems: []
imported_at: "YYYY-MM-DD HH:mm"
updated_at: "YYYY-MM-DD HH:mm"
---
```

## wiki/ 职责

`wiki/` 是索引层 / 关键摘要层，不是 raw 原文对应的飞书文档镜像。任何 wiki 页面都只保留用于检索、判断、导航和复用的关键摘要、结构化关系、证据链接与不确定性；全量事实细节通过 `raw/` 引用回查原文。

### 证据层

- `sources/`：raw 原始资料的结构化摘要 / 索引，必须能回溯到 raw；推荐按 raw 分类存放为 `wiki/sources/<分类>/<原始文档名>.md`，Source 标题统一为 `Source：<原始文档名>`，历史平铺 `wiki/sources/<slug>.md` 可兼容。Source 只保留关键摘要、分层标签、证据边界和不确定性，不复制 raw 全量信息。

### 业务架构主干目录

- `modules/`：顶层平台系统，当前默认且仅允许两类：`计费`、`结算`。import / ingest / query / lint 可以基于 Source 证据更新这两类已有 Module 页面；但除非用户明确指定“新增 module / 新增平台系统”，否则不得创建第三类 Module。Source frontmatter 或正文中可以记录模块标签。
- `scenarios/`：业务场景和长期查询入口，指计费/结算领域对外或跨系统复用的问题域，例如端内计收费，端外计收费，月付息费，通用平台计费、端内结算，端外结算，分账等。Scenario 可能由多个 Source 支撑，并长期作为查询入口；页面必须沉淀主题范围、证据边界、关键摘要、覆盖材料矩阵、推荐阅读路径、关键决策或边界提醒、关联能力、关联 Source 与不确定性；不要把每份 raw 的完整流程、字段、方案细节复制进 Scenario。新增 Scenario 前必须说明为什么现有目录不足以承载、解决什么长期复用问题、覆盖哪些来源、证据是否充分、有哪些不确定性，并向用户咨询。
- `platform-capabilities/`：计费/结算平台对外或业务可复用的能力、规则和流程。计费示例：计价，收费，退费，计费配置查询；结算示例：收款账户决策，结算，退计算，分账，退分账。页面必须沉淀能力定义、所属平台系统、适用场景、接口/入口、核心规则、能力边界、关键机制、与相关能力的关系、承载应用、证据来源与不确定性。新增 PlatformCapability 前必须说明为什么现有目录不足以承载、解决什么长期复用问题、覆盖哪些来源、证据是否充分、有哪些不确定性，并向用户咨询。
- `applications/`：应用架构 / 系统职责，具体到 PSM 维度。代码仓库知识中 Application 也是应用/代码仓库一对一入口，承载 repo URL/path、commit、入口、场景矩阵、上下游边界。
- `code-components/`：代码模块 / 包 / 组件级语义索引，描述稳定代码区域的职责、代码位置、输入输出、约束、关联 Implementation/Data；不得按文件镜像代码。
- `data/`：数据库数据结构层，只放数据库表结构、索引信息，以及明确的 MySQL / Redis 表、Key、索引、字段等存储结构。代码仓库知识中 Data 应按场景/平台能力持久化拆分，说明该场景读写哪些表、使用阶段和字段语义差异。
- `implementations/`：技术实现层，主要放领域模型和系统 PSM 之间的交互链路细节。代码仓库知识中 Implementation 应按业务场景/平台能力拆分，使用系统时序图体现代码动作、上下游交互、领域动作、单据生命周期/状态流转；“系统依赖”只列下游业务服务 / PSM / SDK。
- `maps/`：跨层映射，维护 Module / Scenario / PlatformCapability / Application / CodeComponent / Data / Implementation 之间的映射关系。

默认链路：

```text
Module/System -> Scenario -> PlatformCapability -> Application -> Data/Implementation -> Source -> RawRef
```

业务拆解语义：

```text
业务场景 -> 平台能力 / 规则 / 流程 -> 应用职责 -> 技术实现
平台系统 -> 应用架构 -> 存储结构 -> 技术架构
```

### 辅助输出层目录

`applications/` 承载 PSM / 职责对象，`platform-capabilities/` 承载能力 / 规则 / 流程，`scenarios/` 承载长期查询入口的业务场景专题。

- `overviews/`：query 阶段沉淀的一页式综述，通常需要用户确认后创建。
- `comparisons/`：query 阶段沉淀的对比分析，通常需要用户确认后创建。
- `query_feedback/`：query 纠错反馈，按天汇总。

## 业务分层更新阈值

- 业务分层页面是知识合集索引，不是 Source 引用计数器。新 Source 命中已有 Scenario / PlatformCapability / Application / Data / Implementation / Module 时，先判断是否改变该页面的定义、边界、契约、规则、职责、证据边界、长期阅读路径或关键映射；没有改变时，只在 Source、必要的上层页面或 Map 中建立引用，不更新该页面正文。
- 越底层、越稳定的知识索引，更新阈值越高。若新 Source 只是使用或引用已有 Application / Data / Implementation，不改变职责边界、输入输出、适用范围、核心机制、存储结构或关键规则，则不得更新稳定页面正文。
- 新增 Scenario 或 PlatformCapability 前，必须按第一性原理向用户说明并征询确认：为什么现有目录不足以承载、解决什么长期复用问题、覆盖哪些来源、证据是否充分、有哪些不确定性。
- 若 Source 没有明确指定新增 PSM，不新增 Application；若没有明确指定新增平台系统，不新增 Module。

## 计费结算证据要求

- 金额、币种、主体、账期、费用项、费率、规则版本、结算周期、结算单状态、账单状态、出款状态等高风险事实必须能回溯到 Source/raw；证据不足时写“待确认”或 `UNKNOWN`。
- 不得凭通用支付或清结算经验补全计费/结算口径。
- 来源冲突时必须显式标记冲突，不得静默合并。
- 批量 ingest 发现明显来源冲突时，必须先阻塞语义写入并让用户确认正确来源/版本/适用范围；确认前不得把冲突事实写成确定结论或覆盖稳定业务页面。

## 批量导入 / 批量摄入规则

批量导入 / 摄入是操作层概念，不改变 schema：

- 一次最多 10 个飞书文档。
- 每个原始文档仍独立创建一个 `raw_ref`。
- 每个 `raw_ref` 仍独立对应一个 Source 页面。
- 不创建合并 raw 页面或合并 Source 页面。
- Source 分类仍唯一继承 `raw/<分类>`。
- 批量导入最终沉淀结果应与逐个导入这些文档并正确处理冲突后的结果一致。
- 明显冲突必须先阻塞并让用户确认；确认后的冲突、版本边界、升级替代关系再沉淀到 Source、下游页面、LOG 和报告。

## REGISTRY 拆分规则

- `REGISTRY-Sources-*` 的拆分维度固定为 `raw/` 子目录。
- 主干目录对应：`REGISTRY-Modules`、`REGISTRY-Scenarios`、`REGISTRY-PlatformCapabilities`、`REGISTRY-Applications`、`REGISTRY-CodeComponents`、`REGISTRY-Data`、`REGISTRY-Implementations`、`REGISTRY-Maps`。
- 辅助输出层对应：`REGISTRY-Overviews`、`REGISTRY-Comparisons`、`REGISTRY-QueryFeedback`。不再设置额外专题 / 概念 / 实体注册分册。

## 页面类型

除 `applications/`、`data/`、`modules/` 外，其他 `wiki/` 业务/辅助页面文件名应尽量与页面标题中的中文命名一致，优先使用中文文件名；只有文件系统不兼容、名称过长、历史兼容或用户明确指定时，才使用英文 slug 或缩写。

### Source

- 标题：`Source：<原始文档名>`，必须保留原始文档名；`Source:` 旧英文冒号格式只兼容历史页面，新增页面统一使用中文冒号 `Source：`。
- 目录：推荐 `wiki/sources/<分类>/`，兼容 `wiki/sources/`
- 文件名：默认使用 `<原始文档名>.md`，仅清理文件系统非法字符；不要默认改成简称或语义 slug。
- 必须包含 `raw_ref`、`raw_category`。
- 必须显式写明 `原始素材目录` / `Raw 分类` / `raw/<分类>` 中至少一种回溯字段。
- Source 分类必须与 raw 文件所在目录一致。

### Module

- 标题：`Module: <平台系统>`
- 目录：`wiki/modules/`
- 用于沉淀顶层平台系统范围、目标用户、核心场景、上下游边界和能力地图。当前默认且仅允许 `计费`、`结算` 两类；流程可以基于 Source 证据更新这两类已有页面，但若非用户明确指定新增 module，不得创建其他 Module 页面。

### Scenario

- 标题：`Scenario: <业务场景>`
- 目录：`wiki/scenarios/`
- 文件名：默认使用 `<业务场景>.md`，优先中文，并与标题中的业务场景命名一致。
- 用于沉淀计费 / 结算的具体业务场景和项目型专题入口，例如端内计收费，端外计收费，月付息费，通用平台计费、端内结算，端外结算，分账等。Scenario 可以由多个 Source 支撑，并长期作为查询入口；页面应包含主题范围、证据边界、关键摘要、覆盖材料矩阵、推荐阅读路径、关键决策或边界提醒、关联能力、关联 Source 与不确定性；新增前必须说明长期复用价值、证据来源、影响范围和不确定性，并向用户咨询。

### PlatformCapability

- 标题：`PlatformCapability: <平台能力/规则/流程>`
- 目录：`wiki/platform-capabilities/`
- 文件名：默认使用 `<平台能力>.md`，优先中文，并与标题中的平台能力命名一致。
- 用于沉淀计费/结算平台对外或业务可复用的能力定义、适用范围、入口、规则和边界。计费包括：计价，收费，退费，计费配置管理；结算包括：收款账户决策，结算，退计算，分账，退分账。页面应包含能力定义、所属平台系统、适用场景、接口/入口、核心规则、能力边界、关键机制、相关能力关系、承载应用、证据来源与不确定性；新增前必须说明长期复用价值、证据来源、影响范围和不确定性，并向用户咨询。

### Application

- 标题：`Application: <PSM>`
- 目录：`wiki/applications/`
- 用于沉淀 PSM 维度的系统职责、上游/下游、接口、任务、消息、归属能力和边界。除非用户明确指定新增 PSM，默认不新增 Application；仅在某个 PSM 职责发生变更，或 Source 明确补充该 PSM 边界 / 上下游 / 服务能力时更新。
- 页面格式默认包含：`Summary`、`代码仓库`、`核心职责`、`主要能力`、`上下游与边界`、`典型场景`、`关联数据`、`关联平台能力`、`证据来源`。其中 `关联数据`、`关联平台能力`、`证据来源` 是 LLM Wiki 固定追踪章节，即使参考外部 PSM 模板也必须保留。


### CodeComponent

- 标题：`CodeComponent: <应用><组件语义>`
- 目录：`wiki/code-components/`
- 用于沉淀稳定代码模块 / 包 / 组件的语义索引，包含代码位置、关键符号、输入输出、核心流程、修改约束、关联 Implementation/Data/Source。
- CodeComponent 是语义单元，不是按文件逐个建页；同一 CodeComponent 可被多个 Implementation 复用。

### Data

- 标题：`Data: <数据模型/状态/表>`
- 目录：`wiki/data/`
- 用于沉淀数据库表结构、索引信息，以及明确的 MySQL / Redis 表、Key、索引、字段等存储结构。只有文档给出具体存储结构，或涉及现有表结构 / 索引修正、新增表 / Redis Key 时才创建或更新；普通领域对象、状态机、字段语义不应放入 data，除非它们落到具体存储结构。Data 页面应在前置“来源”章节统一声明表结构来源；字段/索引表默认继承该来源，金额、币种、主体、账期、费用项、费率、规则版本、结算周期、结算单状态等高风险字段必须能回溯 Source/raw。

### Implementation

- 标题：`Implementation: <技术实现>`
- 目录：`wiki/implementations/`
- 用于沉淀领域模型和系统 PSM 之间的交互链路细节，包括接口、时序、幂等、事务、异常恢复、容灾、降级、灰度、可观测性等。领域模型页面必须关联到 data 中的具体表结构 / Redis Key / 索引信息；若缺少存储证据，应写明待确认。

### Map

- 标题：`Map: <映射名称>`
- 目录：`wiki/maps/`
- 用于维护跨层映射，如 Module-Scenario、Scenario-PlatformCapability、PlatformCapability-Application、Application-Data/Implementation。

### Overview / Comparison

- 目录：`wiki/overviews/`、`wiki/comparisons/`
- 由 query 阶段推荐归档产生，通常需要用户确认；ingest 阶段默认不主动创建。

### QueryFeedback

- 目录：`wiki/query_feedback/`
- 按天汇总维护，同日新增反馈优先增量追加。
- 注册到 `REGISTRY-QueryFeedback`。

## 元数据约定

每个 wiki 页面建议包含 YAML frontmatter：

```yaml
---
type: source|module|scenario|platform_capability|application|data|implementation|map|overview|comparison|query_feedback
title: "..."
created_at: "YYYY-MM-DD HH:mm"
updated_at: "YYYY-MM-DD HH:mm"
---
```

Source 页面至少包含：

```yaml
---
type: source
title: "Source：xxx"
raw_ref: "../../raw/技术方案/xxx.md"
raw_category: "技术方案"
created_at: "YYYY-MM-DD HH:mm"
updated_at: "YYYY-MM-DD HH:mm"
---
```

## 引用规则

- wiki 页面之间使用相对 Markdown 链接。
- wiki 正文不直接粘贴飞书原始 URL，应链接到 raw 引用文件或 Source 页面。
- raw 引用文件允许保存原始飞书 URL、doc_id、wiki_token。
- REGISTRY 中 Doc 列使用相对 Markdown 链接。

## 一致性原则

```text
raw -> Source -> REGISTRY -> INDEX
```

- Source 分类冲突时，以 raw 为准。
- REGISTRY 与 INDEX 冲突时，以 REGISTRY 为准。
- 业务分层关系冲突时，以 Source / raw 可回溯事实为准。
