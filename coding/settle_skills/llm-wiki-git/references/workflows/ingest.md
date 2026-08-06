# Ingest — 摄入源文档

## 目标

把 `raw/` 中的原始资料引用转成可回溯、可注册、可查询的本地 Markdown 知识。摄入完成后必须保证：

```text
raw -> Source -> REGISTRY
```

并尽量维护计费/结算业务分层链路：

```text
Module/System -> Scenario -> PlatformCapability -> Application -> Data/Implementation -> Source -> RawRef
```

批量 ingest 的最终沉淀结果应与逐个文档导入并正确处理冲突后的结果一致。批量的额外价值是把同一知识块相关文档作为一个证据集合提前比较：发现明显冲突时阻塞语义写入，让用户确认正确来源/口径，避免错误知识沉淀。

## 前置条件

- 已确定 Git 仓库路径。
- 已读取 `AGENTS.md`。
- 待摄入素材已存在于 `raw/<分类>/<slug>.md`；批量模式支持一次最多 10 个 raw 引用。

## 步骤

可先用本地脚本完成 Source 骨架与 REGISTRY-Sources 分册注册。

单 raw 示例：

```bash
python3 scripts/ingest_source.py \
  --repo <REPO_ROOT> \
  --raw raw/技术方案/<raw_slug>.md \
  --source-slug <source-slug>
```

批量 raw 示例：

```bash
python3 scripts/ingest_source.py \
  --repo <REPO_ROOT> \
  --raw raw/技术方案/<文档1>.md \
  --raw raw/技术方案/<文档2>.md \
  --raw raw/系统白皮书/<文档3>.md
```

该脚本只做 `raw -> Source -> REGISTRY-Sources-<分类>` 的本地 bookkeeping，不替代原文读取、摘要抽取、业务分层页面维护、maps 更新和语义判断。明显语义冲突由 LLM ingest workflow 读取全批次原文后判断；冲突未解决前，不填充 Source 关键事实、不更新业务分层页面、不更新 maps，也不把冲突事实写成确定结论。

### 1. 读取仓库与注册信息

- 读取 `wiki/registry/REGISTRY.md` 与相关 `REGISTRY-*.md`。
- 遍历 `wiki/sources|modules|scenarios|platform-capabilities|applications|data|implementations|maps|comparisons|overviews|query_feedback/*.md`，建立标题、slug、raw_ref 映射。
- 扫描 `raw/`，建立 raw 引用映射。

### 2. 确定 ingest 范围

- 用户指定 raw 文件：只处理该文件。
- 用户指定多个 raw 文件：按一个批次处理，最多 10 个；批次内每个 raw 独立创建/更新 Source，但语义抽取和业务分层更新必须统一分析。
- 用户指定 raw 目录：处理该分类下素材；若目录内待处理素材超过 10 个，应分批处理并说明批次边界。
- 用户未指定：可根据最近 import 的 raw 文件或待处理清单处理。

ingest 前必须明确 raw 文件属于哪个 `raw/分类`，后续 Source 和 REGISTRY 分类必须继承该分类。

### 3. 读取 raw 原始内容

- 先读取 raw 引用文件 frontmatter。
- 若 `source_kind = lark_doc|lark_wiki`，用 `doc_id` / `wiki_token` / `lark_url` 读取飞书原文。
- 若只有 URL 没有可读取 token，先尝试解析；解析失败则标记 BLOCKED，说明缺少可读取 token。
- 若读取工具提示 Feishu OAuth refresh token expired，提示用户运行 `! bytedcli feishu login --force-oauth` 后重试。
- 若 `source_kind = external_url`，按 URL 读取或仅使用已有摘要，取决于用户授权和网络可用性。
- 若 `source_kind = local_file`，读取本地文件或记录无法读取原因。
- 若 `source_kind = note`，raw 文件正文就是原始输入。

禁止把飞书原文复制进 raw 或 wiki。wiki 是索引层，只能抽取关键事实、关键摘要、结构化关系、证据边界和不确定性；完整细节需要保留在 raw 原文对应的飞书文档中供后续回查。

### 4. 创建或更新 Source 骨架

Source 文件推荐：`wiki/sources/<分类>/<原始文档名>.md`，文件名必须保留原始文档名，仅清理文件系统非法字符。除非用户明确指定，不要使用简称或重新生成语义 slug。若历史仓库已采用 `wiki/sources/<slug>.md`，可继续兼容，但新增 / 重命名时应迁移到原始文档名文件；REGISTRY-Sources 分册仍按 raw 分类拆分。

除 `applications/`、`data/`、`modules/` 外，其他 `wiki/` 业务/辅助页面文件名应尽量与页面标题中的中文命名一致，优先使用中文文件名；只有文件系统不兼容、名称过长、历史兼容或用户明确指定时，才使用英文 slug 或缩写。

frontmatter 至少包含：

```yaml
---
type: source
title: "Source：<原始文档名>"
raw_ref: "../../raw/<分类>/<raw_slug>.md"
raw_category: "<分类>"
modules: []
architecture_layers: []
business_scenarios: []
platform_capabilities: []
systems: []
created_at: "YYYY-MM-DD HH:mm"
updated_at: "YYYY-MM-DD HH:mm"
---
```

Source 正文至少包含：

- 原始来源：相对链接到 raw 引用文件
- 文档名：保留原始文档名，页面标题统一为 `Source：<原始文档名>`
- 原始素材目录 / Raw 分类
- 3–5 句关键摘要
- 3–7 条关键要点
- 提取的平台系统 / 场景 / 能力 / 应用 / 数据 / 技术实现
- 金额、币种、主体、账期、费用项、费率、规则版本、结算周期、账单/结算单状态等高风险事实的证据边界
- 不确定性和待确认项

Source 不是原文备份，不要记录 raw 的全量章节、全量表格、全量字段、完整流程。若某些细节只在原文中偶尔需要，应保留为“可回查线索”而不是复制到 wiki。

已有 Source 优先根据 `raw_ref` 匹配更新，不只按标题匹配。

### 5. 批量 ingest 的证据矩阵与冲突预检

批量 ingest 时，创建 Source 骨架后，不要按文档顺序逐个静默更新业务页面；必须先把本批次所有原文读完，抽取批次证据矩阵，再统一判断是否更新 Source 摘要和业务分层页面。

证据矩阵至少记录：

- Source / raw_ref
- 文档标题、更新时间、是否声明版本/升级/替代关系
- 涉及的平台系统、业务场景、能力/规则/流程、PSM、接口、数据对象、实现链路
- 高风险事实：金额、币种、主体、账期、费用项、费率、计价口径、规则版本、结算周期、结算单状态、账单状态、出款状态等
- 每条事实的适用范围、版本、时间、证据位置或可回查线索

冲突类型：

1. **批次内冲突**：本批次多份文档对同一事实键 / 规则 / 字段 / 状态 / 职责边界给出不同结论，且没有明确版本、适用范围或升级替代关系。
2. **升级替代**：新文档明确声明替代、升级、废弃、版本变更或生效时间；这不是普通冲突，但必须在 Source 和下游页面标注版本/时效边界。
3. **既有 wiki 冲突**：本批次 Source 与已有 Source / Scenario / PlatformCapability / Application / Data / Implementation 中的事实、定义、边界或结论不一致。
4. **证据不足**：多个文档看似冲突，但缺少范围、版本或上下文，无法判断是否同一事实键。

处理规则：

- 不得静默合并冲突事实。
- 对高风险计费/结算事实，默认按 ERROR 级别告警。
- 对职责边界、接口契约、状态流转、表结构、索引、Redis Key 等稳定页面事实，冲突未解决前不得覆盖既有结论。
- 发现明显冲突时，必须阻塞后续语义写入，向用户询问哪个文档/版本/适用范围为准；用户确认前不覆盖稳定业务页面，也不把冲突事实写入 Source 摘要或下游页面作为确定结论。
- 可以先完成 raw_ref、Source 骨架、REGISTRY-Sources 对齐；但必须在报告中明确业务知识沉淀被阻塞，等待用户确认后再继续。
- 若文档存在明确升级/替代关系，应保留旧事实的适用范围，并记录新事实的版本、生效时间和证据，不要简单删除旧事实。
- 用户确认冲突处理后，再按确认结果继续写入；最终结果应与逐个导入这些文档、并在冲突处人工纠正后的结果一致。

发现冲突时，必须在结果中主动输出 Conflict Alerts，包含：

- 冲突 ID
- 冲突类型：批次内冲突 / 既有 wiki 冲突 / 升级替代 / 证据不足
- 严重级别：ERROR / WARNING
- 冲突事实键：例如 `费率口径`、`结算周期`、`Application: xxx 的职责边界`、`表字段含义`
- 涉及 Source/raw
- 各来源的冲突陈述
- 证据或可回查线索
- 建议处理：保留多版本、标记待确认、以新版本为准但保留旧版本范围、请求用户判断
- 本次实际动作：已写入 Source 骨架 / 已阻塞 Source 摘要与业务页面更新 / 等待用户确认

若无冲突，明确写“未发现批次内或既有 wiki 来源冲突”，再继续语义写入。

### 6. 处理业务架构主干页面

基于 Source 证据识别并维护：

- Module/System（允许更新既有 `计费`、`结算` 两类；除非用户明确要求新增，否则不创建新的 `wiki/modules/` 页面）
- Scenario
- PlatformCapability
- Application
- Data
- Implementation
- Map

更新阈值：

- 先判断“是否需要更新业务分层页面”，不要把业务分层页面当作 Source 引用计数器。新 Source 命中已有页面时，只有当它改变该页面的定义、边界、契约、规则、职责、证据边界、长期阅读路径或关键映射，才更新页面正文。
- 如果新 Source 只是使用、引用或佐证已有页面，不改变上述内容，则只更新 Source 的关联、必要的上层入口或 `maps/`，不更新被命中的稳定页面正文。
- 越底层、越稳定的页面更新阈值越高；若不改变 Application / Data / Implementation 的职责边界、输入输出、适用范围、核心机制、存储结构或关键规则，只能建立引用，不得更新正文。
- 批量 ingest 中若证据矩阵已发现明显冲突，冲突涉及的业务分层页面必须停止写入，等待用户确认；不受冲突影响的页面可继续更新，但报告中必须说明边界。

规则：

- Module 规则：`wiki/modules/` 当前默认且仅允许 `计费`、`结算` 两类；ingest 可以基于 Source 证据更新这两类已有 Module 页面；除非用户明确指定新增 module / 新增平台系统，否则不得创建第三类 Module。
- Scenario 规则：Scenario 是计费/结算领域的具体业务场景，也可以是项目型专题入口。生成/更新时按 Scenario 模板保留：元信息（主题范围、证据边界）、摘要、覆盖材料矩阵、推荐阅读路径、关键决策 / 边界提醒、关联平台能力、承载应用、关联数据、关键实现、证据来源、不确定性、变更记录。Scenario 是索引页，不复制每份 raw 的完整流程、字段或方案细节。新增 Scenario 前必须先回答：为什么现有目录不足以承载、解决什么长期复用问题、覆盖哪些来源、证据是否充分、有哪些不确定性，并向用户咨询。
- PlatformCapability 规则：平台能力承载计费/结算平台对外或业务可复用的能力、规则和流程。生成/更新时按 PlatformCapability 模板保留：能力定义、所属平台系统、适用场景、接口/入口、核心规则、能力边界、关键机制、与相关能力的关系、承载应用、边界与不确定性、证据来源。新增 PlatformCapability 前必须先回答：为什么现有目录不足以承载、解决什么长期复用问题、覆盖哪些来源、证据是否充分、有哪些不确定性，并向用户咨询。
- Application 规则：Application 必须具体到 PSM 维度；除非用户明确指定新增 PSM，默认不新增 Application。代码仓库知识中 Application 同时作为应用/代码仓库入口；仅当某个 PSM 职责、repo 入口、边界 / 上下游 / 服务能力被 Source 明确改变/补充时更新。
- Application 格式规则：更新 PSM 页面时按模板组织 `Summary`、`代码仓库`、`核心职责`、`主要能力`、`上下游与边界`、`典型场景`；同时必须保留 LLM Wiki 追踪章节 `关联数据`、`关联平台能力`、`证据来源`。
- Data 规则：Data 只放数据库表结构、索引信息，以及明确的 MySQL / Redis 表、Key、索引、字段等存储结构。代码仓库知识中的 Data 应按场景/平台能力持久化拆分，说明共享表在该场景的使用阶段和字段语义差异；领域对象没有存储证据时不得放入 Data。
- Implementation 规则：Implementation 主要放领域模型和系统 PSM 之间的交互链路细节；代码仓库知识中的 Implementation 应按场景/平台能力拆分，使用系统时序图体现代码动作、上下游交互、领域动作、单据生命周期和状态流转；领域模型必须关联到 `data/` 中的具体表结构 / Redis Key / 索引信息，缺少存储证据时写“待确认”。
- 先抽取链路：`业务场景 -> 平台能力 / 规则 / 流程 -> 应用 -> 数据/技术实现`。
- 只能基于 Source 或 raw 原文对应的飞书文档中有证据的信息生成；不确定项写“待确认”。即使有证据，也只沉淀可复用关键摘要和索引信息，不搬运 raw 全量内容。
- 维护双向链接，例如 Scenario 链 PlatformCapability，PlatformCapability 反链 Scenario；PlatformCapability 链 Application，Application 反链 PlatformCapability。
- 更新 `wiki/maps/` 中对应映射表。
- 同步更新对应 REGISTRY 分册。


### 代码仓库摄入分支

当 raw 的 `source_kind = code_repo` 时，ingest 在 `raw -> Source -> REGISTRY-Sources-代码仓库` 对齐后，按以下语义步骤维护代码知识：

1. 解析 repo URL/path、branch/commit、Application/PSM、include/exclude paths；代码证据必须绑定 revision。
2. 更新 Source：记录代码证据矩阵、场景证据矩阵、证据边界；不得复制源代码全文。
3. 更新 Application：Application 作为应用/代码仓库入口，保留 repo、commit、阅读入口、场景到 Implementation/Data 矩阵。
4. 创建/更新 CodeComponent：只为稳定语义模块建页，例如 handler/facade、core service、domain model、DAO、业务消息、内部任务、下游 client；不得按文件逐个建页。
5. 创建/更新 Implementation：按业务场景或平台能力拆分，使用系统 `sequenceDiagram` 表达上游/入口/领域转换/core service/持久化/下游业务服务/状态流转；没有差异的通用段落不要重复。
6. 创建/更新 Data：按场景/平台能力持久化拆分；共享表可以重复出现，但必须说明该场景使用阶段、DAO/model 证据、状态字段和查询/更新条件。
7. 维护 Map：Application -> Scenario/PlatformCapability/Implementation，Implementation -> CodeComponent，Implementation -> Data。
8. 更新 REGISTRY：`REGISTRY-CodeComponents`、`REGISTRY-Applications`、`REGISTRY-Implementations`、`REGISTRY-Data`、`REGISTRY-Maps`。

代码仓库摄入的“系统依赖”只列下游业务服务 / PSM / SDK；Kitex、DB/GORM、Redis、TCC、RocketMQ、Chronos 等中间件/运行时不作为系统依赖沉淀。

### 7. 处理辅助输出层

ingest 阶段按以下承载关系归类：

- 职责对象归入 `applications/`。
- 能力 / 规则 / 流程归入 `platform-capabilities/`。
- 长期查询入口的业务场景专题归入 `scenarios/`。

Overview / Comparison 主要由 query 阶段在用户确认后沉淀；ingest 阶段默认不主动创建。

如果确实建议创建 Overview / Comparison，必须说明：为什么主干目录不足以承载、解决什么长期复用问题、覆盖哪些来源、证据是否充分、有哪些不确定性，并向用户咨询。

### 8. 维护 REGISTRY

必须更新：

- `wiki/registry/REGISTRY-Sources-<分类>.md`
- `wiki/registry/REGISTRY.md`

如创建/更新业务分层页面，也更新：

- `REGISTRY-Modules`
- `REGISTRY-Scenarios`
- `REGISTRY-PlatformCapabilities`
- `REGISTRY-Applications`
- `REGISTRY-Data`
- `REGISTRY-Implementations`
- `REGISTRY-Maps`

如创建/更新 Overview / Comparison / QueryFeedback，也更新对应 REGISTRY 分册。

REGISTRY 中 Doc 列使用相对 Markdown 链接，分类列写 `raw/<分类>`。

### 9. 维护 INDEX 与 LOG

- `wiki/LOG.md`：记录 ingest 行为、涉及 raw 文档、结果页面；批量模式必须记录批次内 raw -> Source、结构冲突状态、是否存在 Conflict Alerts。
- `wiki/INDEX.md`：只更新人工导航入口，不承载全量 Source。

### 10. 结果报告

向用户说明：

- 创建/更新了哪些 Source。
- 创建/更新了哪些业务分层页面。
- 是否建议创建辅助层页面。
- 更新了哪些 REGISTRY 分册。
- 本次改动的本地文件路径。
- Conflict Alerts：若发现明显冲突，列出冲突并说明已阻塞哪些写入；若无冲突，明确写“未发现来源冲突”。
- 建议 review diff 后提交 Git。

## 注意事项

- Source 分类与 raw 目录不一致时，以 raw 为准。
- 不要通过 INDEX 判断 Source 是否已全量注册。
- 不确定事实写“待确认”，不要编造。
- 批量 ingest 的结果应与逐个导入并正确处理冲突后的结果一致；不得因为批量模式改变知识建模结果。
- 明显冲突未解决前，不要把冲突事实写成确定知识，也不要覆盖稳定业务页面。
