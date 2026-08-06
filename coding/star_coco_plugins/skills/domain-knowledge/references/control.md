# 治理域核心参考知识

# 总体介绍

治理域（风控治理）通过“决策校验（读、判）\+处罚封禁（写、处置）\+申诉解封（逆向恢复）\+查询与白名单（读、豁免）”为账号、交易、资金、达人等上层业务域提供风险治理能力。职责边界明确：治理域负责风险识别与处置的通用能力沉淀，不承接具体业务流程，不直接拥有业务域数据，仅通过标准接口影响业务域的功能开放与关闭。

与上层业务域的接口关系（仅正向调用，接口交互）：

- 账号域：通过 MakeDecision/CheckPermission 提供行为判定；通过 InnerCheckControlStatus 提供封禁状态查询；处罚/解封由 BatchControl/BatchRelease 发起并影响账号能力开放。

- 交易域（任务、履约、作品子域）：基于 EventKey 执行 Decision Flow 输出拦截码与原因；命中后触发 ControlPoint 对发单、接单等能力进行处置。

- 资金域：处罚项如“禁止提现”由治理域接口控制，查询与处置均通过接口交互，不直接访问资金域数据表。

- 达人域：接单资格、投稿上传等由治理域判定并拦截/放行；处罚与解封仅通过接口影响达人域的能力。

治理闭环目标（端到端）：发现 → 决策 → 拦截/处罚 → 查询 → 申诉解封 → 反馈迭代。

合规与依赖约束（必须遵守）：

- 领域间仅通过接口交互；上层可调用下层，同层可互调，禁止反向调用。

- 聚合层不直接读写数据表；事件图定义与处罚记录等数据由治理域下层模块维护与访问。

- 严格遵守《星图决策校验模块开发手册》《封禁处罚模块开发手册》，先方案后开发、算子/控制点优先复用。

- 外部平台集成均为接口交互：dolphin（规则）、shark（策略）、eagleye（处罚/解封/查询）。

# 业务词典

**（代码仓库:ad/star\_control）**

<table><tbody>
<tr>
<td>

业务用语

（PRD、日常表述中表达）

</td>
<td>

含义

</td>
<td>

标准名

</td>
<td>

对应领域实体/模型



</td>
<td>

对应数据模型

</td>
</tr>
<tr>
<td>

封禁/处罚/违规事件/黑名单

</td>
<td>

定义：对主体（账号、达人、订单等）下发限制或取消能力的处置行为。

别名：封禁、黑名单、处置。

适用范围：账号、交易、资金、达人等对象。

跨域关系：仅通过接口影响上层业务域功能；由治理域通过 BatchControl/ControlPoint 发起，账号、交易、资金、达人域被动接收能力变更。



</td>
<td>

处罚



</td>
<td>

business/control\_point/control\_point\.go\.ControlPoint



</td>
<td>

StarAccessControlRecord（dal/model/star\_access\_control\_record\.go）

StarControlEventRecord（dal/model/star\_control\_event\_record\.go）

单条StarControlEventRecord对应多条StarAccessControlRecord，用于关联一次违规产生的多次处罚

</td>
</tr>
<tr>
<td>

投诉

</td>
<td>

定义：客户或达人对某业务对象进行投诉

</td>
<td>

投诉

</td>
<td>

business/complaint/core/complaint\_record\.go:ComplaintRecord

</td>
<td>

StarComplaint（dal/model/star\_complaint\.go）

StarComplaintHandle

dal/model/star\_complaint\_handle\.gen\.go:14

</td>
</tr>
<tr>
<td>

拦截，鉴权，决策，权限校验

决策流/事件图

</td>
<td>

定义：基于算子编排的有向无环图，用于完成一次决策的参数加载与判断。

别名：Event DAG、事件决策图。

适用范围：账号、交易、资金、达人行为判定。

跨域关系：上层通过 MakeDecision/CheckPermission 调用；治理域加载 star\_event\_permission\_rule 定义。

</td>
<td>

决策



</td>
<td>

business/tool/core/dag\_context\.go:DAGContext

business/tool/core/dag\_core\.go:ExecutorNode

business/tool/core/dag\_core\.go:ExecutorNode



</td>
<td>

StarEventPermissionRule（dal/model/star\_event\_permission\_rule\.go）



</td>
</tr>
<tr>
<td>

算子/loader/加载器

</td>
<td>

定义：原子功能单元，包含数据加载、快捷运算处理、dolphin、shark调用等类型。

适用范围：参数读取、规则判断、数据处理。

跨域关系：属于决策的一部分

</td>
<td>

Executor



</td>
<td>

business/tool/executor/executor\_origin\.go:5\.Executor

</td>
<td>



</td>
</tr>
<tr>
<td>

白名单

</td>
<td>

定义：为指定主体开放或豁免部分能力的名单。

别名：豁免列表。

适用范围：账号、达人、客户、订单。

跨域关系：通过 InnerCheckWhiteList、InnerCheckIdInWhitelist、InnerCheckIdInMultiWhitelist 等接口供上层业务域鉴权或判定，治理域维护标签与对象。

</td>
<td>

Whitelist

</td>
<td>

StarWhitelist

</td>
<td>

StarWhitelist（dal/model/star\_whitelist\.go）

StarWhitelistLabel（dal/model/star\_whitelist\_label\.go）

</td>
</tr>
<tr>
<td>

申诉/解封

</td>
<td>

定义：对处罚发起复核请求，审核通过后解除限制。

别名：解封申诉。

适用范围：所有处罚项。

跨域关系：通过 CreateControlEventAppeal/ReleaseByEvent 接口与上层业务域交互，仅接口方式。

</td>
<td>

Appeal/Release

</td>
<td>

暂无模型，数据模型为StarControlAppealRecord



</td>
<td>

StarControlAppealRecord（dal/model/star\_control\_appeal\_record\.go

</td>
</tr>
<tr>
<td>

Dolphin（规则）

</td>
<td>

定义：公司内部规则引擎平台，承载动态决策表达式。

别名：规则平台。

适用范围：决策判断。

跨域关系：治理域通过算子调用；外部平台（接口交互）。

</td>
<td>

Dolphin

</td>
<td>

外部平台（接口交互）\.Dolphin



</td>
<td>



</td>
</tr>
<tr>
<td>

Shark（策略）

</td>
<td>

定义：集团风控策略配置平台，支持处罚策略管理。

别名：策略平台。

适用范围：策略管理与协同。

跨域关系：治理域在鉴权收敛与处罚联动中调用；外部平台（接口交互）。

</td>
<td>

Shark

</td>
<td>

外部平台（接口交互）\.Shark

</td>
<td>



</td>
</tr>
<tr>
<td>

Eagleye（处罚/解封/查询）

</td>
<td>

定义：集团处罚中心平台，提供处罚、解封、查询能力。

别名：鹰眼。

适用范围：运营接入与联动。

跨域关系：治理域作为服务方对接；外部平台（接口交互）。

</td>
<td>

Eagleye

</td>
<td>

外部平台（接口交互）\.Eagleye

</td>
<td>



</td>
</tr>
</tbody></table>

# 领域模型

## PlantUML源码

```Plain Text
@startuml
' 治理域-领域模型-决策详图（严格基于代码）
' 约束：领域仅通过接口交互；上层→下层，同层互调；禁止反向调用；聚合层不直接读写DB
'
' 源代码清单（仓库与文件路径）：
' - ad/star_control
'   - business/tool/core/dag_context.go
'   - business/tool/core/dag_core.go
'   - business/tool/core/dag_conf.go
'   - business/tool/executor/executor_origin.go
'   - business/tool/graph_pool.go
'   - service/permission.go
'   - service/control.go
'   - service/appeal.go
'   - business/control_point/control_point.go
'   - dal/model/star_event_permission_rule.go
'   - dal/model/star_access_control_record.go
'   - dal/model/star_control_event_record.go
'   - dal/model/star_control_appeal_record.go
'   - business/complaint/core/complaint_record.go
'   - dal/model/star_whitelist.go
'   - dal/model/star_whitelist_label.go
'
class DAGContext {
  -DecisionId: string
  -OriginalInput: string
  -Executor: executor.Executor
  -Graph: ExecutorGraph
  -SubOutput: map[string]string
  -SubError: map[string]error
  -Progress: map[string]bool
  +Run(): error
  +GetResult(): (string, map[string]string, error)
  +GetSimpleResultWithDolphin(): (string, error)
}

interface Executor {
  +Key(): string
  +VerifyInputParams(params: map[string]any): bool
  +DefaultConfig(): map[string]any
  +Execute(ctx: context.Context, param: map[string]any): (any, error)
  +GetDescription(): string
  +GetInputExample(): string
  +GetOutputExample(): string
}

class ExecutorNode {
  -Id: string
  -Key: string
  -Alias: string
  -Input: map[string]any
  -Output: map[string]string
  -Required: bool
  -UpStream: []*ExecutorNode
  -DownStream: []*ExecutorNode
  -Context: ExecutorGraph
  -Condition: Condition
  +GetId(ctx: context.Context): string
  +Exec(ctx: DAGContext): void
}

class ExecutorGraph {
  -EventName: string
  -Nodes: []*ExecutorNode
  -DefaultResult: executor.Executor
  -Persist: bool
  +Exec(ctx: DAGContext): error
  +Shutdown(err: error): void
  +AddNode(node: *ExecutorNode): void
}

class GraphPool {
  +GetGraph(ctx: context.Context, key: string, env: string): ExecutorGraph
}

interface ControlPoint {
  +GetID(): int32
  +IsNeedRelease(): bool
  +GetEntityType(): model.EntityType
  +GetName(): string
  +Control(ctx, entityId: int64, reason: string, operator: string, extra: string): (model.RecordStatus, error)
  +Release(ctx, entityId: int64, extra: string): error
  +BuildBatchControlReqs(ctx, req: control.PunishReq): ([]control.BatchControlReq, error)
  +CheckAuth(ctx, entityIds: []int64, starId: int64): bool
}

class StarAccessControlRecord {
  +ID: int64
  +Operator: string
  +ControlEntityType: model.EntityType
  +Reason: string
  +Detail: string
  +ControlPoint: int32
  +Status: model.RecordStatus
  +EntityId: int64
  +EventId: int64
  +ControlEventId: int64
  +ExpectControlTime: time.Time
  +ActualControlTime: time.Time
  +ExpectReleaseTime: time.Time
  +ActualReleaseTime: time.Time
  +LogId: string
}

class StarControlEventRecord {
  +ID: int64
  +EventName: string
  +ControlInfo: string
  +LogId: string
  +PunishTicket: string
}

class StarControlAppealRecord {
  +ID: int64
  +EntityId: int64
  +ControlEntityType: model.EntityType
  +ControlPoint: int32
  +SubjectID: int64
  +Detail: string
  +AppealInfo: string
  +AppealStatus: AppealStatus
  +RejectReason: string
  +AppealResultTime: time.Time
  +AuthorId: int64
  +Operator: int64
  +OperatorName: string
}

class ComplaintRecord {
  +Id: int64
  +OrderId: int64
  +DemandId: int64
  +ItemId: int64
  +FromUserId: int64
  +ToUserId: int64
  +BizType: ComplaintBizType
  +Status: ComplaintStatus
  +ActionCategoryByReleaseUser: string
  +Detail: string
  +Cert: Cert
  +CanCancelBalance: bool
}

class StarWhitelistLabel {
  +ID: int64
  +SystemType: int32
  +UserRole: int8
  +TargetType: int32
  +Status: int8
  +WhitelistLabel: int32
  +Level: int32
  +Name: string
}

class StarWhitelist {
  +ID: int64
  +SystemType: int32
  +TargetType: int32
  +WhitelistLabel: int32
  +TargetId: int64
  +OperatorId: int64
  +OperatorName: string
  +ExpirationTime: time.Time
  +EventId: int64
}

class StarEventPermissionRule {
  +ID: int64
  +EventName: string
  +ParamFillingConf: string
  +ForbiddenConf: string
  +SharkConf: string
  +DolphinEvent: string
  +DecisionGraph: string
  +Env: string
  +Version: int32
}

class DecisionGraph {
  +EventKey: string
  +Nodes: []control.Node
  +Edges: []control.Edge
  +DefaultResult: map[string]any
}

class DolphinRuleEngine <<external>>
class SharkStrategy <<external>>
class EagleyeControl <<external>>

' 关系（严格依据代码调用/字段引用）
DAGContext --> ExecutorGraph
ExecutorGraph o-- ExecutorNode
ExecutorNode --> Executor
DolphinExecutor ..> DolphinRuleEngine
Shark ..> SharkStrategy
ControlPoint ..> EagleyeControl
GraphPool --> StarEventPermissionRule
GraphPool --> ExecutorGraph
DecisionGraph --> ExecutorGraph
ExecutorGraph --> StarEventPermissionRule
DAGContext --> StarEventPermissionRule
DAGContext --> DolphinExecutor
ExecutorGraph --> DolphinExecutor
ExecutorGraph --> Shark

@enduml

```

## 图例与关键接口

- CheckPermission：接口权限检查（service/permission\.go: CheckPermission）。

- MakeDecision：事件判定决策（service/permission\.go: MakeDecision；业务图加载：business/tool/graph\_pool\.go → business/tool/core/dag\_conf\.go/core\.dag\_core\.go）。

- InnerCheckControlStatus：封禁状态查询（service/control\.go: InnerCheckControlStatus；读取 StarAccessControlRecord）。

- BatchControl / BatchRelease：批量处罚 / 批量解封（service/control\.go；控制点抽象：business/control\_point/control\_point\.go）。

- CreateControlAppeal：创建申诉记录（service/appeal\.go / business/admin/control\_appeal\.go；模型：dal/model/star\_control\_appeal\_record\.go）。

- ReleaseByEvent：按处罚事件解封（service/control\.go: ReleaseByEvent；事件模型：dal/model/star\_control\_event\_record\.go）。

- 投诉实体：ComplaintRecord（business/complaint/core/complaint\_record\.go）。

- 事件图定义：star\_event\_permission\_rule（dal/model/star\_event\_permission\_rule\.go；manager/star\_event\_permission\_rule\.go；聚合层不直连 DB）。

- 外部平台：Dolphin（business/tool/executor/dolphin\_executor\.go → webarch\_dolphin\_decision）、Shark（business/tool/executor/shark\.go → sharkgo）、Eagleye（business/control\_point/control\_point\.go → webarch\_punish\_center\_proxy），均以接口交互



# 接口

若需要接口信息，读取接口文档

[治理域接口文档](https://bytedance.larkoffice.com/wiki/B3ZIwfCAfiVaCikhhgFc6TLrn4g)



# 开发手册目录

**若当前任务所属模块存在开发手册，则必须严格遵守手册进行开发**

<table><tbody>
<tr>
<td>

模块

</td>
<td>

手册

</td>
</tr>
<tr>
<td>

处罚封禁

</td>
<td>

[封禁处罚模块开发手册](https://bytedance.larkoffice.com/wiki/Fq8VwYLi9ifRWFkLZT1cR9Wmn0F)

</td>
</tr>
<tr>
<td>

决策校验

</td>
<td>

[星图决策校验模块开发手册](https://bytedance.larkoffice.com/wiki/LER3wVVIlixBTwk9Ewfc6y5Qnab)

</td>
</tr>
</tbody></table>