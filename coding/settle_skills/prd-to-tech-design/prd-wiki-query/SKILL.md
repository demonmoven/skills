---
name: prd-wiki-query
description: 使用强制的 llm-wiki-git skill 检索计费/结算 LLM Wiki，为 PRD 技术方案定位业务场景、平台能力、应用/PSM、数据、实现、映射、历史技术方案、Source 页面和 raw 原始证据。该 skill 既可作为 prd-feature-split 内部的 Wiki 查询子流程，也可用于功能拆分后的补充调查/重查。
---

# PRD 知识库查询

## 概述

PRD 生成技术方案流程中的知识库查询能力，负责所有 LLM Wiki 检索。本能力必须使用 `llm-wiki-git` skill。

当前编排中，`prd-feature-split` 会把本能力作为子流程前置调用：先通过 LLM Wiki 和代码/配置/接口了解系统现状，再产出正式功能点拆分。本 skill 也可以在功能拆分之后作为补充调查/重查入口使用。

## 输入与本地产物

- 作为 `prd-feature-split` 子流程使用时，输入必须读取 `prd2tech/<需求目录>/prd-understand.md`，并围绕其中的需求摘要、范围边界、待确认问题、资金/账户影响和接口/数据影响建立查询清单。
- 作为补充调查/重查使用时，输入必须读取 `prd2tech/<需求目录>/prd-understand.md`、`prd2tech/<需求目录>/investigation.md` 和 `prd2tech/<需求目录>/prd-feature-split.md`。如果这些文件经过用户微调，以本地文件内容作为当前调查清单；不要用旧对话摘要覆盖。
- 调查结果必须写入或更新 `prd2tech/<需求目录>/investigation.md`，供功能点拆分、绘图和技术方案生成阶段复用。
- 重跑时如果只修改了功能点拆分或调查结论，应更新 `investigation.md`，并在必要时同步更新 `prd-feature-split.md` 以及后续绘图/技术方案产物。

## 强制规则

每次 LLM Wiki 查询都必须加载并遵循 `llm-wiki-git` skill。不得用临时读取 registry 文件替代。

调用方式必须表达为：

```text
llm-wiki-git query "<围绕 PRD/功能点/待确认问题组织的自然语言问题>"
```

问题应包含 PRD 标题/场景、关键业务动作、资金/账户疑问、接口/字段/PSM 线索和希望区分的结论类型，例如“已有能力、本次需要增强、本次需要新增、代码链路、历史方案、待确认风险”。

语义检索路径必须按以下层级展开：

```text
Module/System -> Scenario -> PlatformCapability -> Application -> Data/Implementation -> Source -> RawRef
```

- `Module/System`：先判断属于计费、结算或二者协同；不要新增模块，除非用户明确要求。
- `Scenario`：定位业务场景和长期查询入口，如结算、周期结算、分账、月付贴息等。
- `PlatformCapability`：定位平台能力和规则边界，如正向结算、逆向退结算、可用金额、资金平衡校验、周期汇总等。
- `Application`：定位 PSM/应用职责和上下游边界，如 `settle_center`、`settle`、`bytepay_charge`。
- `Data/Implementation`：查找表/字段/Redis Key/领域模型/实现链路；没有页面时标记缺口并转代码确认。
- `Source`：读取 Source 摘要、历史方案、白皮书和需求文档摘要。
- `RawRef`：当 Source 摘要不足、涉及高风险资金事实或存在来源冲突时，回查 raw 引用的原始材料。

落地读取时仍从 Git Wiki 的机器入口开始：`wiki/registry/REGISTRY.md` -> 相关注册分册 -> 具体页面 -> Source/raw。Registry 是定位文件的入口；上面的 `Module/System -> ... -> RawRef` 是必须遵循的语义分析路径。

知识库默认是本地 Git Wiki 仓库 `charge_settle_llm_wiki`（默认路径 `/Users/bytedance/go/src/code.byted.org/charge_settle_llm_wiki`）。优先使用本地仓库检索。知识库是线索/索引层，不是最终事实源。

## Wiki 的作用和边界

Wiki 只提供：

- 功能点的业务语义和历史背景。
- 功能点对应的候选平台能力、能力边界和系统职责线索。
- 候选 PSM、候选仓库、上下游线索。
- 历史技术方案和相似 PRD 线索。
- 已有平台能力、已有系统职责和可复用链路。
- 哪些问题可由现状确认，哪些仍需用户确认。

Wiki 不决定最终实现事实，代码仓库才是权威来源。禁止只读 Wiki 或只读代码就产出技术改造点。Wiki 用于定位阅读路径，所有改动点都必须由代码确认后才能进入结论。

## 查询目标

重点检索：

- 场景和别名。
- 平台能力和能力边界。
- 平台能力、能力边界，以及与 `prd-feature-split.md` 中能力域的对应关系。
- 应用/PSM 及职责边界。
- 场景、能力、应用、数据、Source 之间的映射。
- 历史技术方案和相似 PRD。
- PRD 待确认问题在现有系统中的对应规则、默认行为、已有约束和历史决策。
- 现有能力、本次需要增强能力、本次需要新增能力、无需改动能力的候选分类。
- 字段、存储、接口、流程相关的数据/实现页面。
- Wiki 摘要不足时，回查 Source/raw 原始证据。

## 证据纪律

- Wiki 页面只能作为索引和阅读路径，不能作为高风险事实的最终证明。
- 涉及账户流、结算阶段边界、金额/币种/主体、状态流转、IDL 字段、BP/XML、存储事实时，必须继续回查 Source/raw 或代码证据。
- 来源冲突或缺少 raw/代码证据时，标记 `待确认`。
- 代码仓库是最终事实源。不得仅基于 Wiki 阅读或单独代码阅读输出技术改造点；Wiki 给线索，代码定事实。
- 以功能点 ID 为主键做调查。每条 Wiki 线索、代码证据、链路结论、资金流/信息流结论都必须能回溯到一个或多个功能点 ID。
- 禁止把 Wiki 命中的历史方案结论直接转换成技术改造点；必须先拆为 `claim`，再由代码证据确认。

## 代码调查路径

针对每个功能点 ID，按以下路径调查，缺失的层级要说明“不涉及/未找到/待确认”：

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

不要只做关键词命中清单。调查结论必须回答：当前能力是否存在、能力归属哪个 PSM/repo、入口在哪里、关键分支在哪里、是否影响资金/数据/接口/配置/消息、这次应新增/扩展/复用/编排/配置/兼容/观测/补偿哪类能力。

## Evidence Table（必须）

`investigation.md` 必须为每个功能点输出 evidence table：

| 功能点 ID | Claim | Wiki 线索 | 代码证据 | 验证状态 | 置信度 | 是否可进入技术方案 | 说明 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| F001 |  | 页面/Source/历史方案 | repo/path/file.go:function 或 XML/IDL/配置/表 | confirmed/contradicted/not_found/pending | high/medium/low | yes/no |  |

- `confirmed` 且有代码证据的改动点才可以进入技术方案的确定性改动。
- `pending`、`not_found` 或只有 Wiki 线索的项只能进入 `待确认事项`、`风险` 或 `调查不足`，不得包装成最终方案。
- 如果 Wiki 与代码冲突，以代码为准，并在说明中记录冲突。

## 输出

输出写入 `prd2tech/<需求目录>/investigation.md`，内容包括：

- 命中的场景/能力/应用/映射。
- 相关历史方案和可复用决策。
- 候选 PSM/仓库。
- Source/raw 证据列表。
- Wiki 推导出的假设和未知项。
- 已有功能、本次需要增强功能、本次需要新增功能、无需改动功能的候选分类。
- 建议的代码检索关键词。
- 按功能点 ID 输出的 evidence table。
- 明确的 `需代码确认` 清单：每个候选改动点/事实在进入技术方案前都必须由代码确认。
- 按功能点输出改动点定位图谱：改动仓库（PSM/repo）、改动接口（RPC/HTTP/IDL 方法）、受影响链路（入口->编排->下游->持久化/消息）和改动类型（接口/逻辑/配置/数据/消息/缓存/任务）。每条必须带代码定位证据（文件/函数/分支/配置）；只有 Wiki 证据的条目保持 `待确认`。
- 对 `prd-understand.md` 中每个待确认问题给出处理结论：`Wiki/代码已确认`、`仍需用户确认`、`保留待确认但不阻塞` 或 `资金流待确认（阻塞）`。
