# Init Templates — Git-only 初始化模板

初始化计费/结算 Git 知识库时用于创建 `AGENTS.md`、`wiki/INDEX.md`、`wiki/LOG.md` 和 `wiki/registry/*` 的骨架模板。

---

## AGENTS.md 模板

```markdown
# AGENTS

本文档只记录这套 LLM Wiki 的结构、规则和例外，不记录具体业务事实。具体业务知识应沉淀到 Source / Module / Scenario / PlatformCapability / Application / CodeComponent / Data / Implementation / Map / Overview / Comparison / Query Feedback。

当前知识库采用 Git-only 模式：raw 只保存原始资料引用，wiki 全部为本地 Markdown，INDEX 给人看，REGISTRY 给机器读。飞书只作为 raw 原始资料读取来源，不作为 wiki 存储后端。

## 核心规则

- INDEX 是人工导航页，不承担全量注册；REGISTRY 及其分册才是机器侧全量入口。
- Source 分类唯一以 raw 为准：每个 Source 的分类必须与原始素材所在的 `raw/分类` 一致。
- Source 页面必须保留原始文档名：标题统一为 `Source：<原始文档名>`，文件名默认使用 `<原始文档名>.md`（仅清理文件系统非法字符）。
- 所有事实性内容必须能回溯到 raw 或 Source；综合判断必须和原始事实区分开。
- 金额、币种、主体、账期、费用项、费率、规则版本、结算周期、账单/结算单状态等高风险事实必须能回溯 Source/raw；证据不足时写“待确认”或 `UNKNOWN`。
- 默认增量维护；只有页面严重漂移、结构损坏或重建 REGISTRY 分册时才整页覆盖。
- 先定 raw，再谈 wiki：import、ingest、registry、lint 的上游依据都是 `raw/`，不是 INDEX。
- wiki 是知识合集的索引层 / 关键摘要层，不是 raw 原文对应的飞书文档镜像；只沉淀可查询、可复用、可回溯的关键摘要、结构化关系、证据边界、阅读路径和不确定性。

## Git 模式业务分层默认规则

- `raw/` 按原始材料类型组织，不按业务架构层拆 raw。
- `wiki/` 按计费/结算业务架构模型组织：平台系统 → 业务场景 → 平台能力 → 应用 → 数据 / 技术实现。
- 默认链路：`Module/System -> Scenario -> PlatformCapability -> Application -> Data/Implementation -> Source -> RawRef`。
- 默认平台系统仅包括：计费、结算；可以更新既有两类，但除非用户明确指定新增 module / 新增平台系统，不得创建第三类 Module。
- 旧专题、旧概念、旧实体层已废弃：长期查询入口归入 `scenarios/`，能力/规则/流程归入 `platform-capabilities/`，PSM/职责对象归入 `applications/`。

## 主干目录职责

### 代码仓库语义索引规则

- Application 作为应用/代码仓库的一对一入口，沉淀 repo URL/path、commit、阅读路径和场景矩阵。
- CodeComponent 是稳定代码模块 / 包 / 组件级语义索引，不按文件逐个建页。
- Implementation 按业务场景或平台能力拆分，使用系统时序图体现代码动作、上下游交互、领域动作、单据生命周期/状态流转。
- Data 按场景/平台能力持久化拆分；共享表可以在多个 Data 页出现，但必须说明使用阶段和证据。
- “系统依赖”只列下游业务服务 / PSM / SDK，不列中间件/运行时依赖。


- `modules/`：顶层平台系统，仅计费、结算两类；可更新，不默认新增。
- `scenarios/`：计费 / 结算领域的业务场景和长期查询入口，例如计费规则生效、账单生成、账单重算、结算单生成、结算出款、对账差异、差错补偿等；可以由多个 Source 支撑，沉淀跨系统、跨层复用的术语、模式和阅读路径。
- `platform-capabilities/`：计费 / 结算平台对外或业务可复用的能力、规则和流程，例如费用项定义、计费规则、账单生成、结算单生成、结算周期、差错处理等。
- `applications/`：PSM 维度的应用职责、边界、上下游、接口、任务和消息；除非用户明确指定新增 PSM，默认不新增 Application。
- `data/`：数据库表结构、索引信息、Redis Key、字段等具体存储结构；只有明确涉及现有表/索引修正或新增表/Redis Key 时才变动。
- `implementations/`：领域模型和系统 PSM 之间的交互链路细节；领域模型需要关联到 `data/` 的具体表/Key/索引，缺少存储证据时标注“待确认”。
- `maps/`：只维护跨层映射。
- `overviews/`、`comparisons/` 是辅助知识层，服务主干，不替代主干；主要由 query 阶段在用户确认后产生，ingest 阶段默认不主动创建。

## 新增与更新阈值

- 业务分层页面是知识合集索引，不是 Source 引用计数器。新 Source 命中已有 Scenario / PlatformCapability / Application / Data / Implementation / Module 时，先判断是否改变该页面的定义、边界、契约、规则、职责、证据边界、长期阅读路径或关键映射；没有改变时，只在 Source、必要的上层页面或 Map 中建立引用，不更新该页面正文。
- 越底层、越稳定的知识索引，更新阈值越高。若新 Source 只是使用或引用已有 Application / Data / Implementation，不改变职责边界、输入输出、适用范围、核心机制、存储结构或关键规则，则不得更新稳定页面正文。
- 新增 Scenario 或 PlatformCapability 前，必须按第一性原理向用户说明并征询确认：为什么现有目录不足以承载、解决什么长期复用问题、覆盖哪些来源、证据是否充分、有哪些不确定性。
- 若 Source 没有明确指定新增 PSM，不新增 Application；若没有明确指定新增平台系统，不新增 Module。

## 命名与格式规则

- 飞书来源导入时，必须优先用 `lark-cli` 或 `bytedcli feishu docs fetch-doc` 只读获取标题和正文摘要，根据内容匹配当前仓库已有 raw 分类；无法读取、内容不足或无法稳定命中分类时，再咨询用户。
- 飞书 raw 文件名默认必须与飞书文档标题一致，仅清理文件系统非法字符；“白皮书”类文档默认归入 `raw/系统白皮书`。
- Source 标题统一为 `Source：<原始文档名>`；Source 文件默认位于 `wiki/sources/<分类>/<原始文档名>.md`。
- PSM Application 文件名使用 PSM 下划线格式，例如 `caijing.bytepay.charge` → `caijing_bytepay_charge.md`。
- Application 页面必须保留 `## 关联数据`、`## 关联平台能力`、`## 证据来源` 三个章节；已废弃层的历史引用应忽略或移除。
- Data 页面应在前置 `## 来源` 章节统一声明表结构来源；字段/索引表默认继承该来源，不逐行重复来源，只有字段级来源差异、冲突或待确认时才在备注中说明。

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
```

---

## INDEX.md 初始模板

```markdown
# INDEX

LLM Wiki 人工导航入口。

> 全量注册请查看 `wiki/registry/REGISTRY.md` 和 `REGISTRY-*` 分册；此处只保留常用入口和阅读路径。

## 目录说明

| 目录 | 说明 |
|---|---|
| raw/ | 原始资料引用，按材料类型归档 |
| wiki/sources/ | Source 摘要 |
| wiki/modules/ | 平台系统 |
| wiki/scenarios/ | 业务场景 |
| wiki/platform-capabilities/ | 能力 / 规则 / 流程 |
| wiki/applications/ | 应用架构 / 系统职责 / 代码仓库入口 |
| wiki/code-components/ | 代码模块 / 包 / 组件级语义索引 |
| wiki/data/ | 数据库表结构 / 索引 / Redis Key |
| wiki/implementations/ | 技术实现 |
| wiki/maps/ | 跨层映射 |
| wiki/registry/ | 机器注册入口 |

## 推荐入口

- [REGISTRY](registry/REGISTRY.md)
- [计费](modules/charge.md)
- [结算](modules/settlement.md)
```

---

## LOG.md 初始模板

```markdown
# LOG

最新操作在最下方。
```

---

## REGISTRY.md 初始模板

```markdown
# REGISTRY

机器注册入口。INDEX 只做人类导航，不承载全量注册。

## 分册

| 分册 | 说明 | 数量 | 最后更新 |
|---|---|---:|---|
| [Sources-需求文档](REGISTRY-Sources-需求文档.md) | raw/需求文档 Source | 0 | {{YYYY-MM-DD HH:mm}} |
| [Sources-技术方案](REGISTRY-Sources-技术方案.md) | raw/技术方案 Source | 0 | {{YYYY-MM-DD HH:mm}} |
| [Sources-ADR决策](REGISTRY-Sources-ADR决策.md) | raw/ADR决策 Source | 0 | {{YYYY-MM-DD HH:mm}} |
| [Sources-系统白皮书](REGISTRY-Sources-系统白皮书.md) | raw/系统白皮书 Source | 0 | {{YYYY-MM-DD HH:mm}} |
| [Sources-数据文档](REGISTRY-Sources-数据文档.md) | raw/数据文档 Source | 0 | {{YYYY-MM-DD HH:mm}} |
| [Sources-接口文档](REGISTRY-Sources-接口文档.md) | raw/接口文档 Source | 0 | {{YYYY-MM-DD HH:mm}} |
| [Sources-值班记录](REGISTRY-Sources-值班记录.md) | raw/值班记录 Source | 0 | {{YYYY-MM-DD HH:mm}} |
| [Sources-复盘报告](REGISTRY-Sources-复盘报告.md) | raw/复盘报告 Source | 0 | {{YYYY-MM-DD HH:mm}} |
| [Sources-团队规约](REGISTRY-Sources-团队规约.md) | raw/团队规约 Source | 0 | {{YYYY-MM-DD HH:mm}} |
| [Sources-风险防控](REGISTRY-Sources-风险防控.md) | raw/风险防控 Source | 0 | {{YYYY-MM-DD HH:mm}} |
| [Sources-日常分享](REGISTRY-Sources-日常分享.md) | raw/日常分享 Source | 0 | {{YYYY-MM-DD HH:mm}} |
| [Modules](REGISTRY-Modules.md) | 平台系统 | 0 | {{YYYY-MM-DD HH:mm}} |
| [Scenarios](REGISTRY-Scenarios.md) | 业务场景 | 0 | {{YYYY-MM-DD HH:mm}} |
| [PlatformCapabilities](REGISTRY-PlatformCapabilities.md) | 能力 / 规则 / 流程 | 0 | {{YYYY-MM-DD HH:mm}} |
| [Applications](REGISTRY-Applications.md) | 应用架构 | 0 | {{YYYY-MM-DD HH:mm}} |
| [CodeComponents](REGISTRY-CodeComponents.md) | 代码组件 | 0 | {{YYYY-MM-DD HH:mm}} |
| [Data](REGISTRY-Data.md) | 数据架构 | 0 | {{YYYY-MM-DD HH:mm}} |
| [Implementations](REGISTRY-Implementations.md) | 技术实现 | 0 | {{YYYY-MM-DD HH:mm}} |
| [Maps](REGISTRY-Maps.md) | 跨层映射 | 0 | {{YYYY-MM-DD HH:mm}} |
| [Overviews](REGISTRY-Overviews.md) | 综述 | 0 | {{YYYY-MM-DD HH:mm}} |
| [Comparisons](REGISTRY-Comparisons.md) | 对比 | 0 | {{YYYY-MM-DD HH:mm}} |
| [QueryFeedback](REGISTRY-QueryFeedback.md) | 纠错反馈 | 0 | {{YYYY-MM-DD HH:mm}} |
```

## REGISTRY 分册模板

```markdown
# REGISTRY-{{TYPE}}

| 标题 | 类型 | Doc | 分类/目录 | 最后更新 | 关联 |
|---|---|---|---|---|---|
```
