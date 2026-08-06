---
name: settle-center-recognition
description: 在功能点改动分析前，识别 PRD 如何映射到 settle_center 的入口接口和 XML 产品流程。凡 PRD 涉及 settle_center（结算中心）能力时使用，用于判断端内/端外商户、准确 RPC 入口，以及由哪个 conf/mapping 条件命中哪个 conf/product XML 流程。mapping/流程结论必须来自代码和 conf/mapping 条件，不能仅凭 LLM Wiki 断定。
---

# settle_center 流程识别

## 概述

这是所有涉及 `settle_center`（结算中心）改动的前置识别门禁。凡功能点涉及 settle_center，必须在 `prd-feature-split` 产出改动点前执行。本阶段从代码确认请求使用哪个入口接口（端内/端外）以及实际执行哪个 XML 产品流程，确保后续改动分析修改的是正确流程。

Repo: `settle_center` (default `/Users/bytedance/go/src/code.byted.org/settle_center`).

## 何时运行

- PRD 提到或隐含结算、退结算、分账、退分账、独立结算、补差、账户决策，或直接提到 settle_center / 结算中心 / `caijing.bytepay.settle_center`。
- `prd-feature-split` 的功能点将 `settle_center` 列为候选 PSM。
- 输出 settle_center 技术改造点前必须完整运行本阶段。不得基于猜测流程分析改动点。

## 硬性规则（必须）

实际执行的 XML 流程由运行时 `conf/mapping/**` 的 `<condition>` 匹配决定，不由 Wiki 摘要决定。禁止依据 LLM Wiki、页面标题或名称相似性断定执行哪个 XML。必须读取真实 mapping 条件和产品引擎代码确认流程。Wiki 只能提供业务语义和候选线索。

## 步骤 1：判断端内/端外商户和入口接口

先判断 PRD 场景是端内还是端外/收单，再映射到入口接口。事实来源是 `idl/bytepay_settle_center.thrift` 中的 `service SettleCenterService`。

- 端内 (in-app, switched into settle center) interfaces, route tag `CentralRouteMethod`:
  - `SettleInternal` — 端内结算
  - `RefundSettleInternal` — 端内退结算
  - `RefundSettleReverseInternal` — 端内退款关闭
  - `RefundAccountDecision` — 端内退款账户咨询
- 端外 / 收单标准链路 interfaces, route tag `SetRouteMethod`:
  - `Settle` — 结算
  - `RefundSettle` — 退结算
  - `RefundSettleConsult` — 退结算咨询（不落单）
  - `IndependentSettle` — 独立结算
  - `RefundSplit` / `SplitOrderQuery` / `RefundSplitQuery` — 分账相关

需要从 PRD 提取并在代码中确认的区分信号：

- `BizIdentitySceneInfo.AcquireMode` ∈ {`direct`, `bankAgent`, `platformBusiness`, `import`} (constants in `kitex_gen/.../bytepay_settle_center.go`, e.g. `AcquireModeDirect="direct"`). 端外标准收单通常 `acquireMode=='direct'`.
- `SwitchToSettleCenter` ext key (`ReqOrderExtensionKeySwitchToSettleCenter="SwitchToSettleCenter"`); 端内切流链路常以 `$switchToSettleCenter=='True'` 命中 internal mapping.
- `refundType` ext key (`ReqExtInfoKeyRefundType`): `STANDARD_REFUND` / `APPOINTED_ACCOUNT_REFUND` / `ADVANCE_REFUND`.

输出命中的接口，并给出 thrift 行号证据，以及 PRD 隐含的 AcquireMode / switch / refundType 取值。

## 步骤 2：通过 conf/mapping 条件定位 XML 流程

流程由 `core/product/sc_product_engine.go` 中的 `CoordinateEngine.Coordinate` 选择：

1. `buildSceneKey` assembles the routing scene from request fields: `SellProductCode`, `SettleBizProduct`, `ServiceType`, `BizCode`, `BackendProductCode`, `AcquireMode`, `State`, `FlowType`, `RefundType`, `SwitchToSettleCenter`, `InstanceEnv`, `SBusinessCodePermittedEnv`.
2. `c.engine.MappingBiz(mapCtx)` matches these against `conf/mapping/**` `<scene>/<mapping>/<condition>` expressions and returns a product id.
3. `c.engine.Process(product, bpCtx)` runs the matched `conf/product/**` XML.

`conf/mapping/` 下的目录含义：

- `direct/` — 收单直连/端外标准链路（`$acquireMode=='direct'`，再按 `$refundType` 区分标准/非标准退款、按 `$state` 区分 SUCCESS/WAITING）。
- `internal/` — 端内切流链路（`$switchToSettleCenter=='True'`）。
- `bank_agent/`, `platform_business/`, `import/`, `old_default/` — 其它收单模式与历史默认。

处理步骤（全部来自代码/配置，不得靠 Wiki 猜测）：

- 读取候选 mapping XML，列出所有 `<condition>`（mapping 层和 `<product condition=...>`）。
- 用 PRD 隐含字段值匹配这些条件，识别命中的 `<scene>` 和最终解析到的 `conf/product/**` 流程。
- 如果多个 scene 都可能命中，列出它们和区分字段，不要按名称相似性选择。
- 表达式含糊时，到 `core/product/config_domain/config_constant.go` 确认 mapping 变量名（如 `MappingAcquireMode="$acquireMode"`）。

## 步骤 3：输出

输出供 `prd-feature-split` / `prd-design-generate` 使用的识别结果：

- 商户类型: 端内 / 端外（及判定依据字段与取值）。
- 入口接口: thrift 方法名 + 路由标签 + `idl/bytepay_settle_center.thrift` 行号证据。
- 命中的 mapping 场景: `conf/mapping/<dir>/<file>.xml` 的 `<scene id>` 与触发它的 `<condition>` 列表（含字段实际取值）。
- 执行的 XML 产品流程: `conf/product/<...>.xml`（含按 `$state`/`condition` 分叉的多个产品）。
- 未决项: 若 PRD 缺少决定路由的字段（acquireMode / switchToSettleCenter / refundType / serviceType / bizCode / backendProductCode / state），列为 `待确认`，并说明它如何改变命中的流程。

## 步骤 4：XML 编排改动落地要求（供改动分析使用）

若识别结论表明需要改动 settle_center 的 `conf/product/**` 或 `conf/mapping/**` XML 编排，必须在后续改动点分析中给出**完整可落地的目标 XML 片段**，而不是文字描述。完整 XML 至少包含：

- 新增/修改的 `<action>` 完整属性：`name`、`actionType`、`assignFundOrder`、`executePhase`（consult/confirm）、`async`、`index` 等。
- 控制流条件：`<if test=...>`、`<product condition=...>`、mapping 层 `<condition>` 表达式（含字段实际取值）。
- action 间依赖：`assignFundOrder` 指向的上游 action 名，以及生成（FUND_GEN）与驱动（FUND_DRIVEN）、consult 与 confirm 阶段的先后关系。
- 在现有 XML 文件中的精确插入位置，并指明可参照的同类已有流程文件作为模板，标注新增（+）/修改（~）/删除（-）。

### 同步受理 + 异步驱动资金流规则（必须）

如果目标资金流属于同步受理、异步驱动链路，识别阶段必须同时定位 accept 和 execute 两类 XML：

- 同步受理 XML：通常由 `flowType=RPC_*_ACCEPT*` 命中，例如 `RPC_REFUND_ACCEPT_AND_CONSULT_SYNC`。
- 异步执行 XML：通常由 `flowType=RPC_*_EXECUTE_ASYNC` 命中，例如 `RPC_REFUND_EXECUTE_ASYNC`。

后续改动分析必须要求：

- 在同步受理 XML 中新增/修改对应资金流的 consult 阶段编排：`*_FUND_GEN` + `*_FUND_DRIVEN executePhase="consult"`。
- 在异步执行 XML 中新增/修改对应资金流的 confirm 阶段编排：`*_FUND_DRIVEN executePhase="confirm"`，并通过 `assignFundOrder` 依赖同步受理/执行 consult 阶段生成的 fund order。
- 不得只在 execute XML 中新增 confirm，也不得跳过 accept 阶段 consult；否则同步受理阶段无法完成资金流咨询和返回预计算/受理结果。
- 参考模板必须优先读取并引用同类已有流程，例如 `conf/product/general_settle_refund_accept.xml` 的 consult 编排和 `conf/product/general_refund_execute.xml` 的 confirm 编排。

本阶段只识别并指明“应给出完整 XML 及参照模板”，不直接改写 XML 文件（改写属于实现阶段）。

## 边界

- 本阶段不修改 mapping/product XML，只做识别。
- 不从 Wiki/历史方案文本直接断定执行流程；代码 + conf/mapping 条件才是权威。
- 遵守结算 Wiki 的 `AGENTS.md`：除非用户明确新增模块，否则模块范围只包括计费和结算。
