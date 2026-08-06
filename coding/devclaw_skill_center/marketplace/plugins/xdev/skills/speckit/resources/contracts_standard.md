# Contracts 文档：新增 Agent 评估器与 VibeWeb 评测能力

***

## 1. 评测对象（EvalTarget）- 输出产物配置

### 1.1 注册应用（含输出产物配置）

**api path**: `/api/foundation/v1/applications`
**方法**: `POST`
**文件位置**: `idl/thrift/coze/loop/foundation/coze.loop.foundation.application.thrift`
**变更类型**: **改**
**对应用户操作**: User Story 1 - 配置自定义 HTTP 应用的输出产物

**请求: 改**

```thrift
struct RegisterApplicationRequest {  
    1: required i64 space_id (api.js_conv="true", go.tag='json:"space_id"')  
    2: required string name  
    3: optional string description  
    4: required application.HTTPConfig http_config  
    // ... 省略：现有字段 ...  
    50: optional application.ApplicationOutputSchema output_schema  // 【新增】

    255: optional base.Base Base  
}  
```

**响应: 不变(间接受到影响)** - 直接依赖(application/Application)有变更：新增字段 12 output_schema

```thrift
struct RegisterApplicationResponse {  
    1: optional application.Application application

    255: base.BaseResp BaseResp  
}  
```

**Service 定义: 不变**

```thrift
service ApplicationService {  
    // ... 省略：现有代码 ...  
    RegisterApplicationResponse RegisterApplication(1: RegisterApplicationRequest request)  
        (api.post = "/api/foundation/v1/applications")  
    // ... 省略：现有代码 ...  
}  
```

**备注**:

* 扩展现有接口，新增 `output_schema` 字段（字段 50）

* 输出产物配置遵循评测集 EvaluationSetSchema 设计，详见 1.4 节

***

### 1.2 更新应用（含输出产物配置）

**api path**: `/api/foundation/v1/applications/:id`
**方法**: `PUT`
**文件位置**: `idl/thrift/coze/loop/foundation/coze.loop.foundation.application.thrift`
**变更类型**: **改**
**对应用户操作**: User Story 1 - 配置自定义 HTTP 应用的输出产物

**请求: 改**

```thrift
struct UpdateApplicationRequest {  
    1: required i64 id (api.path="id", api.js_conv="true", go.tag='json:"id"')  
    2: required i64 space_id (api.js_conv="true", go.tag='json:"space_id"')  
    3: optional string name  
    4: optional string description  
    5: optional application.HTTPConfig http_config  
    // ... 省略：现有字段 ...  
    50: optional application.ApplicationOutputSchema output_schema  // 【新增】

    255: optional base.Base Base  
}  
```

**响应: 不变(间接受到影响)** - 直接依赖(application/Application)有变更：新增字段 12 output_schema

```thrift
struct UpdateApplicationResponse {  
    1: optional application.Application application

    255: base.BaseResp BaseResp  
}  
```

**Service 定义: 不变**

```thrift
service ApplicationService {  
    // ... 省略：现有代码 ...  
    UpdateApplicationResponse UpdateApplication(1: UpdateApplicationRequest request)  
        (api.put = "/api/foundation/v1/applications/:id")  
    // ... 省略：现有代码 ...  
}  
```

**备注**:

* 支持更新输出产物配置

***

### 1.3 获取应用详情

**api path**: `/api/foundation/v1/applications/:id`
**方法**: `GET`
**文件位置**: `idl/thrift/coze/loop/foundation/coze.loop.foundation.application.thrift`
**变更类型**: **不变(间接受到影响)**
**对应用户操作**: User Story 1 - 配置自定义 HTTP 应用的输出产物

**请求: 不变**

```thrift
struct GetApplicationRequest {  
    1: required i64 id (api.path="id", api.js_conv="true", go.tag='json:"id"')  
    2: required i64 space_id (api.js_conv="true", go.tag='json:"space_id"')

    255: optional base.Base Base  
}  
```

**响应: 不变(间接受到影响)** - 直接依赖(application/Application)有变更：新增字段 12 output_schema

```thrift
struct GetApplicationResponse {  
    1: optional application.Application application

    255: base.BaseResp BaseResp  
}  
```

**Service 定义: 不变**

```thrift
service ApplicationService {  
    // ... 省略：现有代码 ...  
    GetApplicationResponse GetApplication(1: GetApplicationRequest request)  
        (api.get = "/api/foundation/v1/applications/:id")  
    // ... 省略：现有代码 ...  
}  
```

**备注**:

* 返回包含输出产物配置的完整应用信息

***

### 1.4 公共结构定义

#### 1.4.1 Application Common 定义

**文件位置**: `idl/thrift/coze/loop/foundation/domain/common.thrift`
**变更类型**: **增**

```thrift
namespace go coze.loop.foundation.domain.common

typedef string ApplicationContentType(ts.enum="true")  // 【新增】

const ApplicationContentType ApplicationContentType_Text = "Text"  // 【新增】  
const ApplicationContentType ApplicationContentType_Image = "Image"  // 【新增】  
const ApplicationContentType ApplicationContentType_Audio = "Audio"  // 【新增】  
const ApplicationContentType ApplicationContentType_MultiPart = "MultiPart"  // 【新增】  
const ApplicationContentType ApplicationContentType_MultiPartVariable = "multi_part_variable"  // 【新增】

struct ApplicationUserInfo {  // 【新增】  
    1: optional string name  
    2: optional string en_name  
    3: optional string avatar_url  
    4: optional string avatar_thumb  
    5: optional string open_id  
    // ... 省略：其他字段 ...  
}

struct ApplicationBaseInfo {  // 【新增】  
    1: optional ApplicationUserInfo created_by  
    2: optional ApplicationUserInfo updated_by  
    3: optional i64 created_at (api.js_conv="true", go.tag = 'json:"created_at"')  
    4: optional i64 updated_at (api.js_conv="true", go.tag = 'json:"updated_at"')  
    5: optional i64 deleted_at (api.js_conv="true", go.tag = 'json:"deleted_at"')  
}  
```

#### 1.4.2 Application Output Schema 定义

**文件位置**: `idl/thrift/coze/loop/foundation/domain/application.thrift`
**变更类型**: **增**

```thrift
include "common.thrift"  
include "../../data/domain/dataset.thrift"

enum ApplicationFieldDisplayFormat {  // 【新增】  
    PlainText = 1  
    Markdown = 2  
    JSON = 3  
    YAML = 4  
    Code = 5  
}

enum ApplicationFieldStatus {  // 【新增】  
    Available = 1  
    Deleted = 2  
}

enum ApplicationSchemaKey {  // 【新增】  
    String = 1  
    Integer = 2  
    Float = 3  
    Bool = 4  
    Message = 5  
    // ... 省略：其他字段 ...  
}

enum ApplicationFieldTransformationType {  // 【新增】  
    RemoveExtraFields = 1  
}

struct ApplicationMultiModalSpec {  // 【新增】  
    1: optional i64 max_file_count  
    2: optional i64 max_file_size  
    3: optional list<string> supported_formats  
    4: optional i32 max_part_count  
}

struct ApplicationFieldTransformationConfig {  // 【新增】  
    1: optional ApplicationFieldTransformationType trans_type  
    2: optional bool global  
}

struct ApplicationFieldSchema {  // 【新增】  
    1: optional string key  
    2: optional string name  
    3: optional string description  
    4: optional common.ApplicationContentType content_type  
    5: optional ApplicationFieldDisplayFormat default_format  
    // ... 省略：其他字段 ...  
}

struct ApplicationOutputSchema {  // 【新增】  
    1: optional i64 id  
    2: optional i32 app_id  
    3: optional i64 workspace_id  
    4: optional i64 application_id  
    10: optional list<ApplicationFieldSchema> field_schemas  
    100: optional common.ApplicationBaseInfo base_info  
}  
```

#### 1.4.3 Application 实体扩展

**文件位置**: `idl/thrift/coze/loop/foundation/domain/application.thrift`
**变更类型**: **改**

```thrift
struct Application {  
    // ... 现有字段 1-11 ...  
    12: optional ApplicationOutputSchema output_schema  // 【新增】  
}  
```

***

## 2. 评估器 - Agent 评估器

### 2.1 创建 Agent 评估器

**api path**: `/api/evaluation/v1/evaluators`
**方法**: `POST`
**文件位置**: `idl/thrift/coze/loop/evaluation/coze.loop.evaluation.evaluator.thrift`
**变更类型**: **不变(间接受到影响)**
**对应用户操作**: User Story 2 - 创建和管理 Agent 评估器

**请求: 不变(间接受到影响)** - 间接依赖(evaluator/Evaluator -> current_version/EvaluatorVersion -> evaluator_content/EvaluatorContent)有变更：新增字段 104 agent_evaluator

```thrift
struct CreateEvaluatorRequest {  
    1: required evaluator.Evaluator evaluator (api.body='evaluator')  
    100: optional string cid (api.body='cid')

    255: optional base.Base Base  
}  
```

**响应: 不变**

```thrift
struct CreateEvaluatorResponse {  
    1: optional i64 evaluator_id (api.body='evaluator_id', api.js_conv='true', go.tag='json:"evaluator_id"')

    255: base.BaseResp BaseResp  
}  
```

**Service 定义: 不变**

```thrift
service EvaluatorService {  
    // ... 省略：现有代码 ...  
    CreateEvaluatorResponse CreateEvaluator(1: CreateEvaluatorRequest request)  
        (api.post = "/api/evaluation/v1/evaluators")  
    // ... 省略：现有代码 ...  
}  
```

**备注**:

* 扩展现有 CreateEvaluator 接口，新增 `EvaluatorType.Agent = 4`

* 创建时需要在 `evaluator.current_version.evaluator_content` 中设置 `agent_evaluator` 字段(字段104)

* 从 `user_prompt` 中的 `{{变量名}}` 自动提取 InputSchema 并设置到 evaluator_content.input_schemas

* 从 `output_rules` 自动构建 OutputSchema 并设置到 evaluator_content.output_schemas

***

### 2.2 创建自定义技能

**api path**: `/api/evaluation/v1/skills`
**方法**: `POST`
**文件位置**: `idl/thrift/coze/loop/evaluation/coze.loop.evaluation.evaluator.thrift`
**变更类型**: **增**
**对应用户操作**: User Story 2 - 创建和管理 Agent 评估器

**请求: 新增**

```thrift
struct CreateSkillRequest {  // 【新增】  
    1: required i64 workspace_id (api.js_conv="true", go.tag='json:"workspace_id"')  
    2: optional string skill_name (vt.min_size = "1", vt.max_size = "100")  
    3: optional string description (vt.max_size = "500")  
    4: optional string instruction (vt.min_size = "10")  
    5: optional string file_path

    255: optional base.Base Base  
}  
```

**响应: 新增**

```thrift
struct CreateSkillResponse {  // 【新增】  
    1: optional i64 skill_id (api.js_conv="true", go.tag='json:"skill_id"')

    255: base.BaseResp BaseResp  
}  
```

**Service 定义: 新增**

```thrift
service EvaluatorService {  
    // ... 省略：现有代码 ...  
    CreateSkillResponse CreateSkill(1: CreateSkillRequest request)  // 【新增】  
        (api.post = "/api/evaluation/v1/skills")  
    // ... 省略：现有代码 ...  
}  
```

**备注**:

* 用户上传 zip 包后，系统解析 SKILL.md 提取 name、description 和 instruction

* 技能唯一性由 名称 + created_by 确定

***

### 2.3 查询技能列表

**api path**: `/api/evaluation/v1/skills`
**方法**: `GET`
**文件位置**: `idl/thrift/coze/loop/evaluation/coze.loop.evaluation.evaluator.thrift`
**变更类型**: **增**
**对应用户操作**: User Story 2 - 创建和管理 Agent 评估器

**请求: 新增**

```thrift
struct ListSkillsRequest {  // 【新增】  
    1: required i64 workspace_id (api.query='workspace_id', api.js_conv='true', go.tag='json:"workspace_id"')  
    2: optional list<SkillType> skill_types (api.query='skill_types')  
    3: optional list<string> created_bys (api.query='created_bys')  
    4: optional i32 page_num (api.query='page_num')  
    5: optional i32 page_size (api.query='page_size')

    255: optional base.Base Base  
}  
```

**响应: 新增**

```thrift
struct ListSkillsResponse {  // 【新增】  
    1: optional list<Skill> skills  
    2: optional i32 total

    255: base.BaseResp BaseResp  
}  
```

**Service 定义: 新增**

```thrift
service EvaluatorService {  
    // ... 省略：现有代码 ...  
    ListSkillsResponse ListSkills(1: ListSkillsRequest request)  // 【新增】  
        (api.get = "/api/evaluation/v1/skills")  
    // ... 省略：现有代码 ...  
}  
```

***

### 2.4 删除自定义技能

**api path**: `/api/evaluation/v1/skills/:skill_id`
**方法**: `DELETE`
**文件位置**: `idl/thrift/coze/loop/evaluation/coze.loop.evaluation.evaluator.thrift`
**变更类型**: **增**
**对应用户操作**: User Story 2 - 创建和管理 Agent 评估器

**请求: 新增**

```thrift
struct DeleteSkillRequest {  // 【新增】  
    1: required i64 skill_id (api.path='skill_id', api.js_conv="true", go.tag='json:"skill_id"')

    255: optional base.Base Base  
}  
```

**响应: 新增**

```thrift
struct DeleteSkillResponse {  // 【新增】  
    255: base.BaseResp BaseResp  
}  
```

**Service 定义: 新增**

```thrift
service EvaluatorService {  
    // ... 省略：现有代码 ...  
    DeleteSkillResponse DeleteSkill(1: DeleteSkillRequest request)  // 【新增】  
        (api.delete = "/api/evaluation/v1/skills/:skill_id")  
    // ... 省略：现有代码 ...  
}  
```

**备注**:

* 仅允许删除自定义技能，预置技能不可删除

***

### 2.5 异步调试 Agent 评估器

**api path**: `/api/evaluation/v1/evaluators/async_debug`
**方法**: `POST`
**文件位置**: `idl/thrift/coze/loop/evaluation/coze.loop.evaluation.evaluator.thrift`
**变更类型**: **增**
**对应用户操作**: User Story 5 - 调试 Agent 评估器

**请求: 新增**

```thrift
struct AsyncDebugEvaluatorRequest {  // 【新增】  
    1: required i64 workspace_id (api.body='workspace_id', api.js_conv='true', go.tag='json:"workspace_id"')  
    2: optional evaluator.EvaluatorContent evaluator_content (api.body='evaluator_content')  
    3: optional evaluator.EvaluatorInputData input_data (api.body='input_data')

    255: optional base.Base Base  
}  
```

**响应: 新增**

```thrift
struct AsyncDebugEvaluatorResponse {  // 【新增】  
    1: optional i64 evaluator_record_id (api.body='evaluator_record_id', api.js_conv='true', go.tag='json:"evaluator_record_id"')

    255: base.BaseResp BaseResp  
}  
```

**Service 定义: 新增**

```thrift
service EvaluatorService {  
    // ... 省略：现有代码 ...  
    AsyncDebugEvaluatorResponse AsyncDebugEvaluator(1: AsyncDebugEvaluatorRequest request)  // 【新增】  
        (api.post = "/api/evaluation/v1/evaluators/async_debug")  
    // ... 省略：现有代码 ...  
}  
```

**备注**:

* 调试弹窗自动加载 User Prompt 中的所有变量名

* 调试 Session 超时时间为 30 分钟

***

### 2.5.1 获取评估器运行结果

**api path**: `/api/evaluation/v1/evaluator_records/:evaluator_record_id`
**方法**: `GET`
**文件位置**: `idl/thrift/coze/loop/evaluation/coze.loop.evaluation.evaluator.thrift`
**变更类型**: **不变(间接受到影响)**
**对应用户操作**: User Story 5 - 调试 Agent 评估器

**请求: 不变**

```thrift
struct GetEvaluatorRecordRequest {  
    1: required i64 workspace_id (api.query='workspace_id', api.js_conv='true', go.tag='json:"workspace_id"')  
    2: required i64 evaluator_record_id (api.path='evaluator_record_id', api.js_conv='true', go.tag='json:"evaluator_record_id"')  
    3: optional bool include_deleted (api.query='include_deleted')

    255: optional base.Base Base  
}  
```

**响应: 不变(间接受到影响)** - 直接依赖(record/EvaluatorRecord)有变更：新增字段 21 running_data；间接依赖(record/EvaluatorRecord -> evaluator_output_data/EvaluatorOutputData -> evaluator_result/EvaluatorResult)有变更：新增字段 4 additional_artifacts

```thrift
struct GetEvaluatorRecordResponse {  
    1: required evaluator.EvaluatorRecord record (api.body='record')

    255: base.BaseResp BaseResp  
}  
```

**Service 定义: 不变**

```thrift
service EvaluatorService {  
    // ... 省略：现有代码 ...  
    GetEvaluatorRecordResponse GetEvaluatorRecord(1: GetEvaluatorRecordRequest req)  
        (api.get = "/api/evaluation/v1/evaluator_records/:evaluator_record_id")   
    // ... 省略：现有代码 ...  
}  
```

**备注**:

* 复用现有 `GetEvaluatorRecord` 接口

* 需要为该接口添加 API 路径暴露


***

### 2.6 获取调试日志（轮询）

**api path**: `/api/evaluation/v1/evaluators/debug/:evaluator_record_id/logs`
**方法**: `GET`
**文件位置**: `idl/thrift/coze/loop/evaluation/coze.loop.evaluation.evaluator.thrift`
**变更类型**: **增**
**对应用户操作**: User Story 5 - 调试 Agent 评估器

**请求: 新增**

```thrift
struct GetAgentDebugLogsRequest {  // 【新增】  
    1: required i64 workspace_id (api.query='workspace_id', api.js_conv='true', go.tag='json:"workspace_id"')  
    2: required i64 evaluator_record_id (api.path='evaluator_record_id', api.js_conv='true', go.tag='json:"evaluator_record_id"')

    255: optional base.Base Base  
}  
```

**响应: 新增**

```thrift
struct GetAgentDebugLogsResponse {  // 【新增】  
    1: optional list<string> log_lines (api.body='log_lines')

    255: base.BaseResp BaseResp  
}  
```

**Service 定义: 新增**

```thrift
service EvaluatorService {  
    // ... 省略：现有代码 ...  
    GetAgentDebugLogsResponse GetAgentDebugLogs(1: GetAgentDebugLogsRequest request)  // 【新增】  
        (api.get = "/api/evaluation/v1/evaluators/debug/:evaluator_record_id/logs")  
    // ... 省略：现有代码 ...  
}  
```

**备注**:

* 前端轮询获取增量日志

* 左侧消息瀑布框实时展示 Agent 的消息和思考过程

***

### 2.7 取消调试

**api path**: `/api/evaluation/v1/evaluators/debug/:evaluator_record_id/cancel`
**方法**: `POST`
**文件位置**: `idl/thrift/coze/loop/evaluation/coze.loop.evaluation.evaluator.thrift`
**变更类型**: **增**
**对应用户操作**: User Story 5 - 调试 Agent 评估器

**请求: 新增**

```thrift
struct CancelAgentDebugRequest {  // 【新增】  
    1: required i64 workspace_id (api.query='workspace_id', api.js_conv='true', go.tag='json:"workspace_id"')  
    2: required i64 evaluator_record_id (api.path='evaluator_record_id', api.js_conv='true', go.tag='json:"evaluator_record_id"')

    255: optional base.Base Base  
}  
```

**响应: 新增**

```thrift
struct CancelAgentDebugResponse {  // 【新增】  
    255: base.BaseResp BaseResp  
}  
```

**Service 定义: 新增**

```thrift
service EvaluatorService {  
    // ... 省略：现有代码 ...  
    CancelAgentDebugResponse CancelAgentDebug(1: CancelAgentDebugRequest request)  // 【新增】  
        (api.post = "/api/evaluation/v1/evaluators/debug/:evaluator_record_id/cancel")  
    // ... 省略：现有代码 ...  
}  
```

***

### 2.8 列出评估器列表

**api path**: `/api/evaluation/v1/evaluators/list`
**方法**: `POST`
**文件位置**: `idl/thrift/coze/loop/evaluation/coze.loop.evaluation.evaluator.thrift`
**变更类型**: **不变(间接受到影响)**
**对应用户操作**: User Story 4 - 运行包含 Agent 评估器的评测实验

**请求: 不变(间接受到影响)** - 直接依赖(evaluator_type/EvaluatorType)有变更：新增枚举值 Agent=4

```thrift
struct ListEvaluatorsRequest {  
    1: required i64 workspace_id (api.body='workspace_id', api.js_conv='true', go.tag='json:"workspace_id"')  
    2: optional string search_name (api.body='search_name')  
    3: optional list<i64> creator_ids (api.body='creator_ids', api.js_conv='true', go.tag='json:"creator_ids"')  
    4: optional list<evaluator.EvaluatorType> evaluator_type (api.body='evaluator_type')  
    5: optional bool with_version (api.body='with_version')  
    // ... 省略：其他字段 ...

    255: optional base.Base Base  
}  
```

**响应: 不变(间接受到影响)** - 间接依赖(evaluators/Evaluator -> current_version/EvaluatorVersion -> evaluator_content/EvaluatorContent)有变更：新增字段 104 agent_evaluator

```thrift
struct ListEvaluatorsResponse {  
    1: optional list<evaluator.Evaluator> evaluators (api.body='evaluators', go.tag='json:"evaluators"')  
    10: optional i64 total (api.body='total', api.js_conv='true', go.tag='json:"total"')

    255: base.BaseResp BaseResp  
}  
```

**Service 定义: 不变**

```thrift
service EvaluatorService {  
    // ... 省略：现有代码 ...  
    ListEvaluatorsResponse ListEvaluators(1: ListEvaluatorsRequest request)  
        (api.post = "/api/evaluation/v1/evaluators/list")  
    // ... 省略：现有代码 ...  
}  
```

**备注**:

* 评估器列表中包含 Agent 评估器选项，与 LLM 评估器、Code 评估器混排显示

* EvaluatorType 枚举新增 Agent=4，筛选时可传入 evaluator_type=[Agent]

***

### 2.8.1 获取评估器详情

**api path**: `/api/evaluation/v1/evaluators/:evaluator_id`
**方法**: `GET`
**文件位置**: `idl/thrift/coze/loop/evaluation/coze.loop.evaluation.evaluator.thrift`
**变更类型**: **不变(间接受到影响)**
**对应用户操作**: User Story 4 - 运行包含 Agent 评估器的评测实验

**请求: 不变**

```thrift
struct GetEvaluatorRequest {  
    1: required i64 workspace_id (api.query='workspace_id', api.js_conv='true', go.tag='json:"workspace_id"')  
    2: optional i64 evaluator_id (api.path='evaluator_id', api.js_conv='true', go.tag='json:"evaluator_id"')  
    3: optional bool include_deleted (api.query='include_deleted')

    255: optional base.Base Base  
}  
```

**响应: 不变(间接受到影响)** - 间接依赖(evaluator/Evaluator -> current_version/EvaluatorVersion -> evaluator_content/EvaluatorContent)有变更：新增字段 104 agent_evaluator

```thrift
struct GetEvaluatorResponse {  
    1: optional evaluator.Evaluator evaluator (api.body='evaluator')

    255: base.BaseResp BaseResp  
}  
```

**Service 定义: 不变**

```thrift
service EvaluatorService {  
    // ... 省略：现有代码 ...  
    GetEvaluatorResponse GetEvaluator(1: GetEvaluatorRequest request)  
        (api.get = "/api/evaluation/v1/evaluators/:evaluator_id")  
    // ... 省略：现有代码 ...  
}  
```

**备注**:

* 返回的 Evaluator 中 current_version.evaluator_content 可能包含 agent_evaluator 字段

***

### 2.8.2 更新评估器草稿

**api path**: `/api/evaluation/v1/evaluators/:evaluator_id/update_draft`
**方法**: `PATCH`
**文件位置**: `idl/thrift/coze/loop/evaluation/coze.loop.evaluation.evaluator.thrift`
**变更类型**: **不变(间接受到影响)**
**对应用户操作**: User Story 4 - 运行包含 Agent 评估器的评测实验

**请求: 不变(间接受到影响)** - 直接依赖(evaluator_content/EvaluatorContent)有变更：新增字段 104 agent_evaluator；直接依赖(evaluator_type/EvaluatorType)有变更：新增枚举值 Agent=4

```thrift
struct UpdateEvaluatorDraftRequest {  
    1: required i64 evaluator_id (api.path='evaluator_id', api.js_conv='true', go.tag='json:"evaluator_id"')  
    2: required i64 workspace_id (api.body='workspace_id', api.js_conv='true', go.tag='json:"workspace_id"')  
    3: required evaluator.EvaluatorContent evaluator_content (api.body='evaluator_content', go.tag='json:"evaluator_content"')  
    4: required evaluator.EvaluatorType evaluator_type (api.body='evaluator_type', go.tag='json:"evaluator_type"')

    255: optional base.Base Base  
}  
```

**响应: 不变(间接受到影响)** - 间接依赖(evaluator/Evaluator -> current_version/EvaluatorVersion -> evaluator_content/EvaluatorContent)有变更：新增字段 104 agent_evaluator

```thrift
struct UpdateEvaluatorDraftResponse {  
    1: optional evaluator.Evaluator evaluator (api.body='evaluator')

    255: base.BaseResp BaseResp  
}  
```

**Service 定义: 不变**

```thrift
service EvaluatorService {  
    // ... 省略：现有代码 ...  
    UpdateEvaluatorDraftResponse UpdateEvaluatorDraft(1: UpdateEvaluatorDraftRequest request)  
        (api.patch = "/api/evaluation/v1/evaluators/:evaluator_id/update_draft")  
    // ... 省略：现有代码 ...  
}  
```

**备注**:

* 请求中的 evaluator_content 可以设置 agent_evaluator 字段来配置 Agent 评估器

* evaluator_type 可以传入 Agent=4 来指定评估器类型

***

### 2.8.3 提交评估器版本

**api path**: `/api/evaluation/v1/evaluators/:evaluator_id/submit_version`
**方法**: `POST`
**文件位置**: `idl/thrift/coze/loop/evaluation/coze.loop.evaluation.evaluator.thrift`
**变更类型**: **不变(间接受到影响)**
**对应用户操作**: User Story 4 - 运行包含 Agent 评估器的评测实验

**请求: 不变**

```thrift
struct SubmitEvaluatorVersionRequest {  
    1: required i64 workspace_id (api.body='workspace_id', api.js_conv='true', go.tag='json:"workspace_id"')  
    2: required i64 evaluator_id (api.path='evaluator_id', api.js_conv='true', go.tag='json:"evaluator_id"')  
    3: required string version (api.body='version')  
    4: optional string description (api.body='description')  
    100: optional string cid (api.body='cid')

    255: optional base.Base Base  
}  
```

**响应: 不变(间接受到影响)** - 间接依赖(evaluator/Evaluator -> current_version/EvaluatorVersion -> evaluator_content/EvaluatorContent)有变更：新增字段 104 agent_evaluator

```thrift
struct SubmitEvaluatorVersionResponse {  
    1: optional evaluator.Evaluator evaluator (api.body='evaluator')

    255: base.BaseResp BaseResp  
}  
```

**Service 定义: 不变**

```thrift
service EvaluatorService {  
    // ... 省略：现有代码 ...  
    SubmitEvaluatorVersionResponse SubmitEvaluatorVersion(1: SubmitEvaluatorVersionRequest request)  
        (api.post = "/api/evaluation/v1/evaluators/:evaluator_id/submit_version")  
    // ... 省略：现有代码 ...  
}  
```

**备注**:

* 向后兼容 Agent 评估器版本提交

* 返回的 Evaluator 中 current_version.evaluator_content 可能包含 agent_evaluator 字段

***

### 2.9 公共结构定义

#### 2.9.1 EvaluatorType 枚举扩展

**文件位置**: `idl/thrift/coze/loop/evaluation/domain/evaluator.thrift`
**变更类型**: **改**

```thrift
enum EvaluatorType {  
    Prompt = 1  
    Code = 2  
    CustomRPC = 3  
    Agent = 4  // 【新增】  
}  
```

#### 2.9.2 EvaluatorContent 结构扩展

**文件位置**: `idl/thrift/coze/loop/evaluation/domain/evaluator.thrift`
**变更类型**: **改**

```thrift
struct EvaluatorContent {  
    1: optional bool receive_chat_history (go.tag = 'mapstructure:"receive_chat_history"')  
    2: optional list<common.ArgsSchema> input_schemas (go.tag = 'mapstructure:"input_schemas"')  
    3: optional list<common.ArgsSchema> output_schemas (go.tag = 'mapstructure:"output_schemas"')  
    101: optional PromptEvaluator prompt_evaluator (go.tag ='mapstructure:"prompt_evaluator"')  
    102: optional CodeEvaluator code_evaluator  
    103: optional CustomRPCEvaluator custom_rpc_evaluator  
    104: optional AgentEvaluator agent_evaluator  // 【新增】  
}  
```

#### 2.9.3 Agent 评估器相关结构

**文件位置**: `idl/thrift/coze/loop/evaluation/domain/evaluator.thrift`
**变更类型**: **增**

```thrift
struct AgentEvaluator {  // 【新增】  
    1: optional common.ModelConfig model_config  
    2: optional list<Skill> skills  
    3: optional string user_prompt  
    4: optional OutputRules output_rules  
}

struct Skill {  // 【新增】  
    1: optional i64 skill_id  
    2: optional i64 workspace_id  
    3: optional SkillType skill_type  
    4: optional string skill_name  
    5: optional string description  
    6: optional string instruction  
    7: optional string file_path  
}

struct OutputRules {  // 【新增】  
    1: optional string score_rule  
    2: optional string reason_rule  
    3: optional string additional_output_desc  
}  
```

***

## 3. 实验 - Agent 评估器实验

### 3.1 批量查询实验结果

**api path**: `/api/evaluation/v1/experiments/results/batch_get`
**方法**: `POST`
**文件位置**: `idl/thrift/coze/loop/evaluation/coze.loop.evaluation.expt.thrift`
**变更类型**: **不变(间接受到影响)**
**对应用户操作**: User Story 4 - 运行包含 Agent 评估器的评测实验

**请求: 不变**

```thrift
struct BatchGetExperimentResultRequest {  
    1: required i64 workspace_id (api.query='workspace_id', api.js_conv='true', go.tag='json:"workspace_id"')  
    2: required list<i64> experiment_ids (api.body='experiment_ids', api.js_conv='true', go.tag='json:"experiment_ids"')  
    3: optional i64 baseline_experiment_id (api.body='baseline_experiment_id', api.js_conv='true', go.tag='json:"baseline_experiment_id"')  
    10: optional map<i64, expt.ExperimentFilter> filters (api.body = 'filters', go.tag = 'json:"filters"')  
    // ... 省略：其他字段 ...

    255: optional base.Base Base  
}  
```

**响应: 不变(间接受到影响)** - 间接依赖(item_results/ItemResult -> evaluator_results/EvaluatorResult)有变更：新增字段 4 additional_artifacts

```thrift
struct BatchGetExperimentResultResponse {  
    1: required list<expt.ColumnEvalSetField> column_eval_set_fields (api.body = "column_eval_set_fields")  
    2: optional list<expt.ColumnEvaluator> column_evaluators (api.body = "column_evaluators")  
    3: optional list<expt.ExptColumnEvaluator> expt_column_evaluators (api.body = "expt_column_evaluators")  
    4: optional list<expt.ExptColumnAnnotation> expt_column_annotations (api.body = "expt_column_annotations")  
    5: optional list<expt.ExptColumnEvalTarget> expt_column_eval_target (api.body = "expt_column_eval_target")  
    // ... 省略：其他字段 ...

    255: base.BaseResp BaseResp  
}  
```

**Service 定义: 不变**

```thrift
service ExperimentService {  
    // ... 省略：现有代码 ...  
    BatchGetExperimentResultResponse BatchGetExperimentResult(1: BatchGetExperimentResultRequest req)  
        (api.post = "/api/evaluation/v1/experiments/results/batch_get")  
    // ... 省略：现有代码 ...  
}  
```

**备注**:

* Agent 评估器可输出额外文件（markdown/图片/html）

* 附加产物通过 `additional_artifacts` 字段返回

***

### 3.2 公共结构定义

#### 3.2.1 EvaluatorRecord 结构扩展

**文件位置**: `idl/thrift/coze/loop/evaluation/domain/evaluator.thrift`
**变更类型**: **改**

```thrift
struct EvaluatorRecord {  
    1: optional i64 id (api.js_conv = 'true', go.tag = 'json:"id"')  
    // ... 省略：现有字段 2-12 ...  
    20: optional map<string, string> ext  
    21: optional EvaluatorRunningData evaluator_running_data  // 【新增】  
}  
```

#### 3.2.2 EvaluatorRunningData 结构

**文件位置**: `idl/thrift/coze/loop/evaluation/domain/evaluator.thrift`
**变更类型**: **增**

```thrift
struct EvaluatorRunningData {  // 【新增】  
    1: optional string vnc_url  
}  
```

#### 3.2.3 EvaluatorResult 结构扩展

**文件位置**: `idl/thrift/coze/loop/evaluation/domain/evaluator.thrift`
**变更类型**: **改**

```thrift
struct EvaluatorResult {  
    1: optional double score  
    2: optional Correction correction  
    3: optional string reasoning  
    4: optional list<AdditionalArtifact> additional_artifacts  // 【新增】  
}  
```

#### 3.2.4 AdditionalArtifact 结构

**文件位置**: `idl/thrift/coze/loop/evaluation/domain/evaluator.thrift`
**变更类型**: **增**

```thrift
struct AdditionalArtifact {  // 【新增】  
    1: optional string artifact_name  
    2: optional string artifact_url  
}  
```