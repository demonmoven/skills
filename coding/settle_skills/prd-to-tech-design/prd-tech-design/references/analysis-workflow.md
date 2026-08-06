# 分析流程

起草技术方案前使用本参考。目标是避免过早输出方案：先理解需求、补齐缺失信息；随后在功能点拆分阶段先用 LLM Wiki 和代码/配置/接口了解系统现状、已有能力和历史方案，再拆分功能点；最后进入绘图和方案生成。

## 0. 工作目录与可重跑中间产物

每个 PRD 需求必须使用独立工作目录：`prd2tech/<需求目录>/`。`<需求目录>` 由 PRD 标题、项目名、需求编号或文档 token 生成，保持可读、稳定、适合作为目录名。

阶段产物写入该目录，作为用户可人工微调、后续可重跑的事实输入：

| 文件 | 阶段 | 说明 |
| --- | --- | --- |
| `prd-understand.md` | 需求理解 | 基于 PRD 正文和评论/回复的结构化需求摘要、业务场景模型、业务动作候选、范围、业务规则、资金/账户影响、信息流、正文与评论冲突处理、待确认问题 |
| `investigation.md` | 阶段 2 前置调查 | Wiki 线索、系统现状、已有能力、历史方案、按功能点 ID 组织的 evidence table、代码/配置/接口初步证据、待确认项 |
| `prd-feature-split.md` | 功能点拆分 | 基于 `investigation.md` 的业务动作到能力域映射、功能点矩阵、调查任务、已有/增强/新增能力分类、改动点定位表、候选链路、资金流/信息流影响 |
| `sequence-diagram.md` | 时序/流程图 | action、participant、判断分支、同步/异步边界、流程 diff |
| `funds-flow-diagram.md` | 资金流图 | 主体、账户、借贷方向、金额赋值、真实动账标识、字段链路 |
| `tech-design.md` | 技术方案草稿 | 创建飞书/Lark 文档前的 Markdown 草稿 |

重跑规则：

- 若用户已修改 `prd-understand.md` 或 `prd-feature-split.md`，重跑时必须优先读取本地微调后的文件，并从受影响阶段继续，不要无条件从 PRD 原文重新覆盖。
- 如果 PRD 原文与本地中间产物冲突，除非用户要求重新解析 PRD，否则以本地微调文件为准，并在后续产物中标注“基于本地微调结果”。
- 阶段 1 完成后必须写 `prd-understand.md`；阶段 2 必须先写 `investigation.md`，再写 `prd-feature-split.md`。这些文件是硬性产物，不得只在对话中输出。
- 不自动删除旧产物；需要重算时覆盖同一路径，保持路径稳定，便于用户继续微调和重跑。

## 1. 需求完整性门禁

PRD 理解必须同时读取正文和评论/回复：

- 飞书/Lark PRD 必须读取文档评论和回复；正文与评论共同作为需求输入。
- 如果正文和评论冲突，以评论/回复中的最新明确结论为准。
- 如果评论之间互相冲突，或评论表述含糊且会影响正确设计，必须列为待确认；涉及资金流则进入资金流阻塞门禁。
- 已被评论明确否定的正文表述，不得作为产品事实进入需求摘要；在 `prd-understand.md` 的“正文与评论冲突处理”中记录覆盖关系。

以下任一信息缺失且会阻塞正确设计时，起草前必须先提澄清问题：

- 业务场景不清：谁触发流程、在哪个产品入口、面向哪类商户/用户/订单。
- 范围边界不清：地区、渠道、支付方式、商户类型、结算类型、业务线的包含/排除范围。
- 成功标准缺失：预期产物、状态变化、时效、SLA、对账或风控目标。
- 资金/账务行为不清：借贷账户、退款路径、结算时点、付款方/收款方、会计主体、是否真实动账。
- 集成目标缺失：上下游系统、API、消息 topic、回调、离线表或配置负责人。
- 影响存量流量、字段、存储、消息或规则时，兼容/灰度约束缺失。

不要询问能从 PRD、LLM Wiki、本地仓库或引用文档中低成本发现的信息。需求理解阶段只阻塞会影响正确设计的关键问题；非阻塞待确认问题进入阶段 2 后先通过 Wiki/Source/raw/代码调查确认，再只问剩余决策阻塞点。每轮问题尽量不超过 5 个，并按影响分组。

如果用户要求带缺口继续，必须写明假设并标注 `待确认`。

## 2. 现状调查与功能点拆分

功能点拆分必须以 `prd2tech/<需求目录>/prd-understand.md` 为输入。如果该文件被用户微调，以微调后的内容为准。本阶段必须先进行 LLM Wiki 和代码/配置/接口调查，再输出正式功能点矩阵。

### 2.1 LLM Wiki 和代码现状调查

所有 LLM Wiki 检索都必须使用 `llm-wiki-git`。加载该 skill 并遵循其查询流程；不得只手动读取 registry 文件来替代知识库检索。

调用方式必须写成：

```text
llm-wiki-git query "<围绕 PRD/功能点/待确认问题组织的自然语言问题>"
```

问题应包含 PRD 标题/场景、关键业务动作、资金/账户疑问、接口/字段/PSM 线索和希望区分的结论类型，例如已有能力、本次需要增强、本次需要新增、代码链路、历史方案、待确认风险。

语义检索路径必须遵循：

```text
Module/System -> Scenario -> PlatformCapability -> Application -> Data/Implementation -> Source -> RawRef
```

含义：

- `Module/System`：先判断属于计费、结算或二者协同。
- `Scenario`：定位业务场景和长期查询入口。
- `PlatformCapability`：定位平台能力、规则和边界。
- `Application`：定位 PSM/应用职责、上下游和接口边界。
- `Data/Implementation`：定位数据模型、领域模型、实现链路；没有页面时标记缺口并转代码确认。
- `Source`：读取 Source 摘要、历史方案、白皮书和需求文档摘要。
- `RawRef`：当 Source 摘要不足、涉及高风险资金事实或存在来源冲突时，回查 raw 引用的原始材料。

落地读取时仍从 Git Wiki 机器入口定位文件：`wiki/registry/REGISTRY.md` -> 相关注册分册 -> 具体页面 -> Source/raw。Registry 是文件入口，不替代上述语义分析路径。

查询路径保持为：

1. `wiki/registry/REGISTRY.md`
2. 相关注册分册：
   - `REGISTRY-Scenarios.md`
   - `REGISTRY-PlatformCapabilities.md`
   - `REGISTRY-Applications.md`
   - `REGISTRY-Data.md`
   - `REGISTRY-Implementations.md`
   - `REGISTRY-Maps.md`
   - `REGISTRY-Sources-技术方案.md`
   - `REGISTRY-Sources-需求文档.md`
3. 具体 Wiki 页面和 Source 页面。
4. 仅当 Wiki 摘要不足或事实声明需要更强证据时，回查 raw/source 材料。

调查目标：

- 对 `prd-understand.md` 中的待确认问题逐项处理：`Wiki/代码已确认`、`仍需用户确认`、`保留待确认但不阻塞` 或 `资金流待确认（阻塞）`。
- 找到相关业务场景、平台能力、应用/PSM、历史方案、Source/raw 证据和候选代码阅读路径。
- 初步定位代码仓库、入口接口、IDL、BP/XML、消息、表模型、常量、开关和关键函数。
- 明确现有能力、本次需要增强能力、本次需要新增能力、无需改动能力和待确认能力。
- 按功能点 ID 形成 evidence table，记录 claim、Wiki 线索、代码证据、验证状态、置信度和是否可进入技术方案。

调查结果必须写入 `prd2tech/<需求目录>/investigation.md`。

### 2.2 业务动作到能力域拆分

功能点拆分不是 PRD 段落改写，而是从业务语义映射到系统能力变化：

```text
结构化 PRD 理解
  -> 业务场景
  -> 业务动作
  -> 能力域
  -> 系统能力功能点
  -> Wiki/代码调查任务
```

能力域包括：接入、准入校验、规则决策、金额计算、流程编排、资金处理、数据持久化、下游集成、配置灰度、查询运营、异常补偿、可观测。

能力变化类型包括：新增、扩展、复用、编排、配置、兼容、观测、补偿。

先创建业务动作到能力域映射表：

| 动作 ID | 业务动作 | 动作类型 | 能力域 | 拆出的功能点 ID | 拆分理由 | 合并/拆分说明 |
| --- | --- | --- | --- | --- | --- | --- |
| A001 |  | 受理/校验/决策/计算/生成/编排/执行/持久化/通知/查询/配置/补偿 |  | F001 |  |  |

### 2.3 功能点矩阵

完成 `investigation.md` 后再创建功能点表：

| 功能点 ID | 功能点 | PRD 来源 | 业务动作 | 能力域 | 能力变化类型 | 触发场景 | 输入/条件 | 现有能力 | 本次变化 | 输出/状态 | 候选 PSM | 资金影响 | 信息流影响 | 结论 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| F001 |  |  |  |  | 新增/扩展/复用/编排/配置/兼容/观测/补偿 |  |  | 待定位 | 已有能力复用/本次需要增强/本次需要新增/无需改动/待确认 |  | 待定位 | 待分析 | 待分析 | 待确认 |

并为每个功能点创建调查任务：

| 功能点 ID | 系统能力假设 | Wiki 检索任务 | 代码调查任务 | 必须确认的证据 | 开放问题 |
| --- | --- | --- | --- | --- | --- |
| F001 |  | 场景/能力/应用/历史方案/Source | IDL/入口/handler/流程配置/常量/模型/下游/测试 | 文件/函数/IDL/XML/配置/表/常量 |  |

规则：

- 按业务能力和实现归属拆分，不按 PRD 段落拆分。
- 一个 PRD 规则影响多个能力域时拆开；多个 PRD 规则落到同一系统能力时可合并。
- 拆分必须基于 `investigation.md`，不能只基于 PRD 原文。
- 查询/只读能力与写入/状态变更能力分开。
- 同步请求链路、异步任务链路、消息/回调链路、离线/对账链路分开。
- 同一 PRD 中同时出现计价/计费、分账、结算明细生成、周期汇总、凭证/账务、退款/退结算、通知、对账时，必须分开分析。
- 资金方向、账户、金额口径、结算时点、是否真实动账必须单独拆功能点，不能隐藏在“流程改造”中。
- 灰度、幂等、补偿、监控、查询/运营可见性如果影响落地或风险控制，应拆为能力点。
- 每个功能点标记为：`已有能力复用`、`本次需要增强`、`本次需要新增`、`无需改动`、`待确认`。
- 拆分结果必须写入 `prd2tech/<需求目录>/prd-feature-split.md`，包含前置调查摘要、功能点矩阵、改动点定位表、候选链路、候选资金流/信息流、候选 PSM/仓库和阻塞问题。

## 3. 检索关键词与历史方案对比

检索关键词同时来自 PRD 原文和 Wiki 规范术语：

- 场景名和别名。
- 平台能力名称。
- PSM 名称和服务别名。
- 产品术语、字段名、枚举值、资金驱动类型、biz code、账户常量、消息名、表名、IDL 方法名。

历史方案检索：

- 优先检索 `raw/技术方案`、`wiki/sources/技术方案`、`REGISTRY-Sources-技术方案.md`。
- 对比历史范围、假设、受影响 PSM、流程图、账户/资金规则、数据模型、风险章节和灰度约束。
- 不要照搬历史结论；说明哪些仍适用、哪些已变化、哪些未知。

## 4. PSM 和仓库定位

使用多种信号将功能点映射到 PSM：

- Wiki 应用页和映射页。
- PRD 中命名的系统或 API。
- IDL package/service/method 名称。
- 代码常量、handler、converter、BP/XML 流程文件、定时任务、MQ topic、表模型、配置 key。
- 历史技术方案中的影响系统表。

检查仓库时：

- 优先使用 `rg --files` 和 `rg`。
- 检索 `.go`、`.thrift`、`.xml`、`.yaml`、`.json`、SQL/model 文件和测试。
- 追踪足够的调用方/被调方，确认归属 PSM 和改动点。
- 跨仓库链路需记录入口、编排、下游调用、持久化、异步/消息路径、响应/回调。

如果本地缺少仓库，报告缺失仓库/路径，并先基于可用 Wiki/Source 证据继续。只有缺失仓库阻塞核心设计决策时才向用户要路径。

## 4b. 按功能点 ID 的代码调查和 Evidence Table

调查必须以功能点 ID 为主键，不要只按仓库或关键词散列记录。每个功能点按以下路径确认：

```text
IDL / API 定义
  -> Handler / Controller / 消息入口
  -> Service / Domain Logic
  -> Workflow / BP XML / Config / TCC
  -> Constants / Enum / BizCode / Account Type
  -> DAO / Model / DB
  -> MQ / Task / Async Job
  -> Downstream RPC / HTTP
  -> Unit Tests / Integration Tests / Mocks
```

`investigation.md` 必须包含 evidence table：

| 功能点 ID | Claim | Wiki 线索 | 代码证据 | 验证状态 | 置信度 | 是否可进入技术方案 | 说明 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| F001 |  | 页面/Source/历史方案 | repo/path/file.go:function 或 XML/IDL/配置/表 | confirmed/contradicted/not_found/pending | high/medium/low | yes/no |  |

- `confirmed` 且有代码证据的 claim 才能进入技术方案确定性改动。
- `pending`、`not_found` 或只有 Wiki 线索的 claim 只能进入 `待确认事项`、`风险` 或 `调查不足`。
- 如果 Wiki 与代码冲突，以代码为准，并记录冲突。

## 5. 链路、资金流和信息流

业务链路：

- 参与方和角色。
- 触发条件和前置条件。
- 主路径、异常路径、重试/冲正路径。
- 既有能力复用与新增行为。

系统调用链：

- 上游入口、API/RPC 方法、请求/响应字段。
- 编排 PSM 和下游 PSM。
- 同步/异步边界。
- 幂等键、状态机、重试、超时/降级。
- 复杂实现逻辑使用流程图或流程图 diff，不用伪代码。图中展示判断分支、状态流转、幂等/重试点、变更代码/配置节点。

资金流：

- 是否真实动账，还是仅生成账务/明细记录。
- 借方账户、贷方账户、账户常量、biz code、fund-driven type、settlement-driven type。
- 正向、逆向/退款、周期结算、汇总/凭证边界。
- 对账、资损检查、补偿/冲正路径。
- 复杂流程图中突出资金流金额赋值：金额来源字段、赋值公式、币种、借贷账户、biz code、fund type，以及节点是真实动账还是仅生成明细。

信息流：

- 来自 PRD/API/消息/配置的来源字段。
- 决策字段和派生字段。
- 持久化字段/表。
- 下游字段/消息/回调/离线产出。
- 新增/变更字段兼容性。
- 复杂流程图中突出信息流改动：来源字段 -> 决策/派生字段 -> 持久化字段/表 -> 下游 RPC/MQ/回调/离线字段，并标注新增/变更字段和兼容行为。

计费/结算场景不要把这些内容混成一张图。好的方案应区分产品/业务流、系统调用链、资金/账务流和信息/数据流。

绘图子 skill：

- 链路分析后，如果 PRD/方案涉及业务流程、系统调用、BP/XML 编排、异步消息、配置路由或分支逻辑，使用 `prd-sequence-diagram`。输出必须展示 action、participant、分支条件、同步/异步边界、异常/补偿路径和变更 diff 标记。
- 资金流分析后，如果 PRD/方案涉及资金移动、账务/明细生成、账户决策、退款/结算/扣费、汇总/凭证、机构记账或任意账户级影响，使用 `prd-funds-flow-diagram`。输出必须展示主体、账户（待结算户/现金户/垫资户/手续费户/机构户等）、借贷方向、金额来源、是否真实动账、biz code/fund type 和代码证据。
- 资金流子 skill 同时返回 `信息流分析` 所需字段链路，用于保持资金流和信息流配套。
- 绘图输出分别写入 `prd2tech/<需求目录>/sequence-diagram.md` 和 `prd2tech/<需求目录>/funds-flow-diagram.md`，并由技术方案生成阶段直接读取。

## 5b. 改动点定位（功能点 -> 仓库 / 接口 / 链路 -> 分类改动）

完成链路、资金流、信息流分析后，必须按功能点汇总改动点定位。这是调查结论进入技术方案「改动点分析」章节的桥梁。对每个功能点：

1. 从功能点 ID 出发，识别受影响的 **repository/PSM**、改动 **interface**（RPC/HTTP/IDL 方法）和影响链路（entry -> orchestration -> downstream -> persistence/message）。
2. 按类型拆分具体改动：接口（RPC/HTTP/IDL）、业务逻辑、配置（conf/mapping/XML/tcc/开关）、数据（表/索引/字段）、消息（MQ/事件）、缓存、任务（定时/异步）。
3. 每条改动必须附带 **代码证据**（file/function/branch/config），并标记新增 / 修改 / 无需改动 / 待确认。任何改动点都不能只依赖 Wiki 推断。

以下两张表直接进入技术方案：

| 功能点 | 改动仓库（PSM/repo） | 改动接口（RPC/HTTP/IDL 方法） | 受影响链路（入口->编排->下游->持久化/消息） | 涉及改动类型 | 结论 |
| --- | --- | --- | --- | --- | --- |
|  |  |  |  | 接口/逻辑/配置/数据/消息/缓存/任务 | 新增/修改/无需改动/待确认 |

| 改动类型 | 所属仓库/PSM | 具体改动点 | 代码定位（文件/函数/分支/配置） | 改动说明（前 -> 后） | 关联功能点 |
| --- | --- | --- | --- | --- | --- |
| 接口/逻辑/配置/数据/消息/缓存/任务 |  |  |  |  |  |

一个功能点可能跨多个仓库和多种改动类型，需要分别列出，不要合并成一条。

对于 settle_center conf/product 或 conf/mapping XML 编排改动，改动点必须带完整目标 XML 片段（完整 `<action>` 属性：name/actionType/assignFundOrder/executePhase/async/index；`<if test>`/`<product condition>`/mapping `<condition>` 条件；action 依赖和 consult/confirm 顺序；在现有文件中的插入位置），并引用同类既有流程文件作为模板，标注新增（+）/修改（~）/删除（-）。不能只用文字描述 XML 改动。

如果该 XML 资金流属于同步受理、异步驱动场景，必须同时覆盖同步受理和异步执行两段流程：同步受理 XML（例如 `flowType=RPC_*_ACCEPT*`）必须编排资金流 consult，即 `*_FUND_GEN` + `*_FUND_DRIVEN executePhase="consult"`；异步执行 XML（例如 `flowType=RPC_*_EXECUTE_ASYNC`）必须编排对应 confirm，即 `*_FUND_DRIVEN executePhase="confirm"`，并用 `assignFundOrder` 串联 consult 阶段 fund order。参考 `general_settle_refund_accept.xml` 和 `general_refund_execute.xml`；不得只在 execute XML 中补 confirm。

## 6. Evidence Discipline

Use evidence labels in the draft:

- `PRD`: requirement statement, product rule, business goal.
- `Wiki`: scenario/capability/application/data map.
- `历史方案`: prior design or similar scenario.
- `代码`: file path, symbol, IDL method, constant, table model, config key, BP/XML node.
- `推断`: reasoned design inference that still needs owner review.
- `待确认`: missing decision or unavailable source.

High-risk claims requiring code or raw/source evidence:

- Which account is charged/refunded.
- Whether a flow triggers settlement summary, voucher/accounting, or only detail generation.
- Idempotency and status transitions.
- Data migration, schema/index, or compatibility rules.
- Message exactly-once/at-least-once behavior.
- Rollout, gray, fallback, or emergency plan.
