# Query — 知识查询

## 目标

从 Git 仓库中的本地 Markdown 计费/结算知识库回答问题。默认只读，不写回；只有用户确认归档或明确要求修正时才修改文件。

## 前置条件

- 已确定 Git 仓库路径，或当前目录可识别为 LLM Wiki Git 仓库。
- 已读取 `AGENTS.md`。

## 核心规则

- query 默认读取顺序是：`REGISTRY -> 对应 REGISTRY 分册 -> 具体页面`；必要时回查 raw 原文对应的飞书文档。query 中的 3–5 个页面只是第一轮候选集，不是总读取上限；跨层追踪、事实核验、字段 / 接口 / 流程 / 职责边界问题必须继续沿 `wiki -> Source -> raw` 分层扩展。
- query 时若 `REGISTRY / wiki 页面 / Source 摘要` 未能直接、完整回答用户问题，不要咨询用户是否需要，必须直接继续回查关联 raw 原文对应的飞书文档；raw 为飞书引用时需要继续使用可用的飞书文档读取命令继续读取飞书原文，且 raw 回查不限制数量。
- 只有在已获得明确结论，或高相关 raw 原文对应的飞书文档已核查完毕但原文仍无法访问、未命中或证据不足时，query 才可停止。
- 不得因只读取了部分 raw、单个 raw、单一类型 raw（如仅 PRD）或获得阶段性线索后提前停止；只要结论仍可能被其他 raw 补充或修正，就必须继续扩展读取。
- query 过程中不得反复向用户确认是否继续查询、是否继续回查 raw、是否继续读取原文；除非用户明确要求停止，否则默认持续查询直到满足停止条件。
- query 时应根据问题类型选择优先证据源：产品目标 / 页面交互优先查需求文档；接口约束 / 字段规则 / 条数上限 / 状态流转优先查技术方案、数据文档或实现证据；金额、币种、主体、账期、费用项、费率、规则版本、结算周期、账单/结算单状态等高风险事实必须查 Source/raw。

## 步骤

### 1. 读取入口

- 读取 `wiki/registry/REGISTRY.md`。
- 根据问题类型读取相关 `REGISTRY-*.md` 分册。
- `wiki/INDEX.md` 只作为人工导航和辅助上下文，不作为全量发现入口。

### 2. 定位候选页面

| 问题类型 | 优先页面 |
|---|---|
| 事实核验 | Source / raw |
| 平台系统范围 | Module |
| 业务流程 / 场景 | Scenario |
| 能力 / 规则 / 流程 | PlatformCapability |
| 系统职责 / 上下游 / PSM 边界 / 代码仓库入口 | Application |
| MySQL 表 / Redis Key / 索引 / 存储字段 | Data |
| 代码模块 / 包 / 组件职责 | CodeComponent |
| 领域模型 / 技术链路 / PSM 交互 / 接口 / 幂等 / 事务 / 容灾 | Implementation |
| 跨层追踪 | Map |
| 领域对象 / PSM 职责对象 | Application |
| 专题 / 长期查询入口 | Scenario / Overview |
| 对比 | Comparison |
| 历史纠错 | QueryFeedback |

### 3. 读取页面

- 默认先读取 3–5 个最相关页面作为第一轮候选集；这不是总读取上限。wiki 页面是索引 / 摘要层，不保证包含 raw 全量细节。
- query 采用“分层扩展”策略：先从 REGISTRY 定位入口页，再按问题需要沿 `Map -> Module/Scenario/PlatformCapability -> Application/CodeComponent/Data/Implementation -> Source -> raw` 逐层展开。每一层可先选 3–5 个最相关页面，不限制整个 query 的总页面数。
- 若问题需要完整覆盖、跨层追踪、事实核验、冲突消解、字段 / 接口 / 流程 / 职责边界级答案，或第一轮页面无法充分回答，必须继续扩展读取相关 wiki 页面、Source 和 raw。
- query 默认不能停在 wiki / Source 摘要层：若本地 wiki 页面或 Source 摘要无法直接、完整回答用户问题，必须继续回查 raw 引用对应的原始资料；raw 为 `lark_doc` / `lark_wiki` 时，应读取飞书原文。只有原文无法访问、原文也未命中、证据仍不足，或继续扩展只会增加重复证据时，才可以回答“未找到明确依据”。
- 查询领域模型、状态机、接口时，除非问题明确询问具体 MySQL/Redis 表、索引、字段，否则优先读取 `implementations/`，再沿链接查 `data/`。
- 查询表结构、索引、Redis Key、字段落库时，优先读取 `data/`；若 `data/` 无命中，不要把 Implementation 中的领域字段当作已落库事实，应回查 Source/raw 或标注“未找到明确存储证据”。
- 如果是事实性问题，必须至少回查一个 Source 或 raw 原始资料；当用户追问字段、流程细节、完整方案、职责边界或原文依据时，应通过 raw 引用回查原文，而不是假设 wiki 摘要已经完整覆盖。
- 停止扩展的条件：已覆盖问题要求的层级、已找到足够支撑答案的 Source/raw、继续扩展只会增加重复证据、不再命中强相关页面，或高相关 raw 原文对应的飞书文档已核查完毕但证据仍不足、需要明确标注不确定性。

### 4. 默认回查 raw 原文对应的飞书文档

- 先读取 `raw/<分类>/<slug>.md`。
- 若 `source_kind = lark_doc|lark_wiki`，按 `doc_id` / `wiki_token` / `lark_url` 读取飞书原文。
- 当 wiki / Source 摘要没有直接答案时，默认回查 raw 原文对应的飞书文档，不需要等待用户继续确认。
- raw 回查不限制数量；只要结论仍可能被更多 raw 补充或修正，就继续扩展读取。
- 回查飞书仅用于事实核验，不把原文复制回 `raw/` 或 `wiki/`。
- 如果读取飞书失败且错误为 refresh token expired，应提示用户运行 `! bytedcli feishu login --force-oauth` 后重试，并在回答中标记该来源未核验。

### 5. 综合回答

回答中区分：

- 原始事实
- 综合判断
- 建议 / 推断
- 不确定点
- 来源冲突

计费/结算问题尤其要区分：

- 计费金额、账单金额、结算金额、对账金额。
- 账期、结算周期、规则生效时间、规则版本。
- 费用项、费率、计价口径、主体、币种。
- 账单状态、结算单状态、出款状态。

如果来源冲突，应显式指出，不要静默合并。证据不足时写“待确认”或 `UNKNOWN`。

### 6. 归档或修正

- query 默认不修改知识库。
- 若结果值得沉淀为 Overview / Comparison，必须先说明价值、依据 Source、影响范围和不确定性，并等待用户确认。
- 长期专题归入 Scenario，能力 / 规则 / 流程归入 PlatformCapability，职责对象归入 Application。
- 用户明确指出现有页面错误并要求修正时，可以直接修改相关页面，并更新 REGISTRY / LOG / QueryFeedback。

### 7. 更新 LOG

若本次 query 执行了归档或修正，追加 `wiki/LOG.md`。纯只读查询可不写 LOG，除非 AGENTS 要求记录。

## Query Feedback

- query 前检查相关 `wiki/query_feedback/` 和 `REGISTRY-QueryFeedback`。
- 命中历史纠错时，优先参考最新反馈。
- 用户指出回答或页面错误时，先修正当前回答，再记录或更新 QueryFeedback；必要时同步修正相关页面。


### 代码仓库查询路径

- 问“要改哪个仓库/应用”时：优先读 Application，再沿 Application 的场景矩阵读 Implementation/Data。
- 问“某场景/能力怎么实现”时：读 Scenario/PlatformCapability -> Map -> Implementation -> CodeComponent -> Source/raw 代码仓库。
- 问“某个包/函数/模块负责什么”时：读 `REGISTRY-CodeComponents` 和对应 CodeComponent；必要时回查 Source 中的代码路径。
- 问“字段是否落库/表如何使用”时：只以 Data 为准；Implementation/CodeComponent 中的领域字段不能当作落库事实。
- 问“系统依赖”时：只回答下游业务服务 / PSM / SDK；中间件/运行时不作为系统依赖回答。
- 若 wiki 摘要不足，必须回查 raw/代码仓库 中记录的 repo/path/revision 和具体代码路径，且说明代码证据绑定的 commit/revision。
