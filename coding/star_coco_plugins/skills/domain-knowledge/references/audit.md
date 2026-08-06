# 星图审核网关数据模型汇总

## 概述

本文档梳理审核网关关键业务实体和存储实体的结构信息和关联关系，以供后续开发参考使用

## 业务实体

- AMU（审核最小单元）：携带节点 ID、前后继、是否 OrCheck、是否 NeedNotify 与行为接口（IAMU）。

- IAMU（审核最小单元行为接口）：AuditSend、CheckParam、GetAmuAuditResult、GetResultExtra、Abandon、GetAuditEventDetail、显示名等。

- IConsultUnit（审核申诉单元行为接口）：CheckConsultationStatus、SubmitConsultation、SaveResult等

## db实体总览

<div class="callout">

共有 10 个核心实体，其中 **StarAuditEvent** 作为审核过程的核心事实表，**StarAuditSceneAmu** 作为场景与最小审核单元（AMU）的元数据表；**StarAuditTask** 在部分流程中与 Event 共享主键（同一值），并通过 **resource\_id** 关联 **StarResources**。白名单相关实体（Note/NoteExtra）以 Note 为主表，Extra 存储多业务维度的扩展绑定。

</div>

### StarAuditEvent（star\_audit\_event）审核事件

- 表：`star\_audit\_event`

- 含义：每个审核实体的某一审核的信息，包含审核参数，审核结果和送审审出时间等

- 主键：`ID int64`

- 关键字段与标签：

    - `AMUID int32 gorm:\&\#34;column:audit\_min\_unit\&\#34;`

    - `AuditScene int64`（与场景记录关联）

    - `ObjectID int64`，`ObjectType`（自定义枚举：Video/Article/Component/Anchor/Material）

    - 审核状态枚举：`Status`（Init/Auditing/Expired/Received/Done/Fail/Ignore/Cancel），`AuditStatus int8` \(Auditing/Pass/Reject/Cancel\)

- 约束与用法：大量按 `object\_id`、`audit\_min\_unit`、`audit\_scene` 过滤；软删通过嵌入 Base 的 `Deleted` 字段实现

### StarAuditChangeRecord（star\_audit\_change\_record）审核事件变更记录

- 表：`star\_audit\_change\_record`

- 含义：每个审核事件如果出现审核结果变更，则记录在此

- 主键：`ID int64`

- 外键候选：`EventId int64` → `star\_audit\_event\.ID`

- 关键字段：`OriginalAuditStatus`、`CurrentAuditStatus`、`OriginalBanReason`、`CurrentBanReason`、`Operator`

### StarAuditSceneAmu（star\_audit\_scene\_amu）审核场景

- 表：`star\_audit\_scene\_amu`

- 含义：审核场景的一次打包送审包含多个审核事件，带有审核事件的先后顺序和拓扑关系

- 主键：`ID int64`

- 关键字段：`SceneID int64`（场景唯一标识）、`ObjectType`、`SceneName`、`Expression`、`Version`、`Env`

- 用法要点：DAO 既支持按 `ID` 获取当前记录，也支持按 `SceneID` 获取最新版本（`ORDER BY version DESC`）

### StarAuditTask（star\_audit\_task）老审核任务

- 表：`star\_audit\_task`

- 含义：老审核链路，已废弃

- 主键：`ID int64`

- 外键候选：`ResourceId int64` → `star\_resources\.id`

- 关键字段：`OrderId`、`ItemId`、`ResourceOrigin`、`Category`、`AuditStatus`、`RejectReason`

- 特殊约束：在多个流程中与 **StarAuditEvent** 使用同一 ID 作为跨表关联主键（共享 ID）

### StarAuditConsultation（star\_audit\_consultation）审核申诉

- 表：`star\_audit\_consultation`

- 含义：审核申诉记录，用户对审核结果不理解或有异议，发起申诉并完成申诉流转

- 主键：`ID int64`

- 关键字段：`ObjectID`、`ObjectType`（kitex 生成的枚举）、`AuditBindId`、`AuditStatus`、`AuditParam`

### StarAuditMisRecord（star\_audit\_mis\_record）后台操作记录

- 表：`star\_audit\_mis\_record`

- 含义：管理员在管理后台修改审核结果，重新送审，废弃送审等行为的记录

- 主键：`ID int64`

- 关键字段：`SceneId`、`ObjectId`、`Detail`、`Operator`

- 关系推断：`SceneId` 与 `StarAuditSceneAmu\.SceneID` 存在语义上的场景对应（见关系表中的“推断关系”）

### StarAuditPriority（star\_audit\_priority）审核优先权益

- 表：`star\_audit\_priority`

- 含义：客户/主体维度的审核优先级配置

- 主键：`ID int64`

- 关键字段与枚举：`StarId`、`CompanyId`、`EffectiveDimension`、`TaskType`、`Priority`、`Quota`、`Status`、`StartTime`、`EndTime`

### StarAuditWhitelistNote（star\_editor\_whitelist\_note）审核备注

- 表：`star\_editor\_whitelist\_note`

- 含义：客户/主体维度的审核备注信息，包含品牌，资质，豁免等，需要给审核人员展示

- 主键：`ID int64`

- 关键字段与枚举：`CompanyId`、`TaskType`、`NoteStatus`（Banned/Valid）

- 备注：表名与结构名不同步（TableName 指向 editor 前缀）

### StarAuditWhitelistNoteExtra（star\_whitelist\_note\_extra）审核备注额外信息

- 表：`star\_whitelist\_note\_extra`

- 含义：客户/主体维度的审核备注信息，包含品牌，资质，豁免等，需要给审核人员展示

- 主键：`ID int64`

- 外键候选：`NoteId int64` → `star\_editor\_whitelist\_note\.ID`

- 关键字段与枚举：`BizId`、`BizType`（Customer/Brand/Author/Order）

### StarResources（star\_resources）审核实体信息

- 表：`star\_resources`

- 含义：视频，直播，图文的基础信息

- 主键：`Id int64 gorm:\&\#34;primary\_key\&\#34;`

- 关键字段与标签：

    - `Status`、`ProcessStatus`、`ResourceType` 等均为 kitex 枚举（含 `gorm:\&\#34;column:\.\.\.\&\#34;` 标签）

    - `OrderOrDemandId`、`ResourceVersion`、`ItemId`、`ExternalResourceId`、`ResourceUri` 等归档字段

- 外键候选：`ItemId int64` → `star\_audit\_event\.ObjectID`

<table><tbody><tr>
<td>

**实体用途速览：**

- StarAuditEvent：审核事实与结果的中心表，承载 `audit\_min\_unit` 与 `audit\_scene`。

- StarAuditSceneAmu：场景\-最小审核单元的元数据，含表达式、版本与环境。

- StarAuditTask：与 Event 在达尔文物料流程中共享主键 ID，且通过 `resource\_id` 指向资源。

- StarResources：素材/视频等资源元数据，供任务与业务查询使用。

- Note/NoteExtra：白名单主表与扩展维度。

- Consultation：基于对象与类型的外部工单关联记录。

</td>
<td>

**关键属性速览：**

- Event：`object\_id`、`audit\_min\_unit`、`audit\_scene`、`audit\_param`、`audit\_info`

- SceneAmu：`scene\_id`、`expression`、`version`、`env`

- Task：`resource\_id`、`order\_id`、`item\_id`、`audit\_status`

- Resources：`id`、`resource\_uri`、`resource\_type`、`extra\_audit\_source\_list`

- Consultation：`object\_id`、`object\_type`、`audit\_bind\_id`

- Note/Extra：`note\_status`、`biz\_type`

</td>
</tr></tbody></table>

## 关系与连接（含证据）

<div class="callout">

关键结论：

- **Event↔Task**：在达尔文流程中采用“**共享主键**”的设计（`event\.id == task\.id`），方便以同一主键串联审核事件与对应的任务。

- **Task→Resources**：以 `resource\_id` 指向资源实体，业务层会批量拉取资源信息再回填到任务结果。

- **Event→SceneAmu**：以 `audit\_scene` 关联到场景记录的 **ID**（非 `scene\_id`），场景的版本管理通过 `scene\_id` 聚合最新记录。

</div>

### 关系汇总表

<table><tbody>
<tr>
<td>

源实体

</td>
<td>

关系类型

</td>
<td>

目标实体

</td>
<td>

基数

</td>
<td>

连接键

</td>
<td>

证据路径

</td>
</tr>
<tr>
<td>

StarAuditChangeRecord

</td>
<td>

多对一

</td>
<td>

StarAuditEvent

</td>
<td>

N:1

</td>
<td>

event\_id → star\_audit\_event\.id

</td>
<td>

business/audit\_change/audit\_change\_record\.go

dal/dao/star\_audit\_change\_record\.go

</td>
</tr>
<tr>
<td>

StarAuditEvent

</td>
<td>

多对一

</td>
<td>

StarAuditSceneAmu

</td>
<td>

N:1

</td>
<td>

audit\_scene → star\_audit\_scene\_amu\.id

</td>
<td>

business/amu/aweme\_safety\_audit\_amu\.go

dal/dao/star\_audit\_scene\_amu\.go

</td>
</tr>
<tr>
<td>

StarAuditTask

</td>
<td>

多对一

</td>
<td>

StarResources

</td>
<td>

N:1

</td>
<td>

resource\_id → star\_resources\.id

</td>
<td>

service/darwin\.go（批量资源查询与回填）

dal/model/star\_audit\_task\.go

</td>
</tr>
<tr>
<td>

StarAuditTask

</td>
<td>

一对一（共享主键）

</td>
<td>

StarAuditEvent

</td>
<td>

1:1

</td>
<td>

id（共享）

</td>
<td>

service/darwin\.go（eventIds 取自 task\.ID；按 id 查询 Event）

service/darwin\.go/SaveDarwinAuditResult（按 TaskId 查询 Event）

business/audit\_consultation/audit\_consultation\.go（taskId = event\.ID）

</td>
</tr>
<tr>
<td>

StarAuditWhitelistNoteExtra

</td>
<td>

多对一

</td>
<td>

StarAuditWhitelistNote

</td>
<td>

N:1

</td>
<td>

note\_id → star\_editor\_whitelist\_note\.id

</td>
<td>

dal/dao/star\_audit\_whitelist\_note\_extra\.go（按 note\_id 查询）

</td>
</tr>
<tr>
<td>

StarAuditConsultation

</td>
<td>

可选关联

</td>
<td>

StarAuditEvent

</td>
<td>

0\.\.N:0\.\.N

</td>
<td>

object\_id 相等，且 `ConsultationType → AMU` 映射匹配

</td>
<td>

business/audit\_consultation/audit\_consultation\.go（GetAuditEventByObjectIDAndAmuID，taskId=event\.ID）

</td>
</tr>
<tr>
<td>

StarAuditMisRecord

</td>
<td>

多对一（组合键）

</td>
<td>

StarAuditEvent

</td>
<td>

N:1

</td>
<td>

\(scene\_id, object\_id\) → \(audit\_scene, object\_id\)

</td>
<td>

service/audit\.go（CreateAuditMisRecord：sceneId=event\.AuditScene，objectId=event\.ObjectID；行 718、756）

</td>
</tr>
</tbody></table>
