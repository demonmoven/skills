# Field Metadata: 流水线通知 / Pipeline Notification
本文档用于让 AI 理解「流水线通知」如何配置、如何在运行时生效、以及各字段对应的实际行为。
说明：本文聚焦 **Bits 流水线**（自由流水线）的通知功能，仅在 CI 流水线出现或仅为 CI 兼容的属性/行为在此不展开。

1. ## 通知系统综述


流水线通知从大面上分为两类：

- **飞书通知（Lark）**：发送交互式卡片消息到用户或群。

- **Webhook**：对外发起自定义 HTTP(S) 请求（或其他动作类型）。


整体链路（简化）：

1. 用户在 DSL 中配置通知（Pipeline 或 Job 维度）。

2. 引擎状态变化回调后，`pipelinerpc` 组装状态变化事件 `pevent.StatusChangeEvent`，携带本次应生效的通知列表。

3. `pipelineeventrpc` 消费事件，按 `Notification.type` 分流：
1. `LARK`：构建飞书卡片并发送

2. `WEBHOOK`：发起 HTTP 请求（异步）


关键入口代码（便于定位）：

- 事件结构与字段：`internal/pipeline/pevent/pevent.go:44`

- 事件消费入口：`app/pipelineeventrpc/biz/service/event/handler.go:27`

- 通知分流与处理：`app/pipelineeventrpc/biz/service/event/service.go:32`


2. ## DSL 中通知配置位置（静态配置）


通知可以配置在两个层级：

### Pipeline 级通知

- 字段：`Pipeline.notifications[]`

- 含义：在 **PipelineRun** 状态变化时触发。

- 参考定义：`idls/byted/devinfra/pipeline/dsl/pipeline.proto:62`


### Job 级通知

- 字段：`Job.notifications[]`

- 含义：在 **JobRun** 状态变化时触发。

- 参考定义：`idls/byted/devinfra/pipeline/dsl/pipeline.proto:236`


3. ## 运行时实体


运行时通知依附于状态变化事件 `pevent.StatusChangeEvent`。该结构是 `pipelineeventrpc` 消费通知的 **唯一输入**：

### 3.1 `pevent.StatusChangeEvent`

```Go
// 省略了 json tag 与少量缓存字段，仅保留通知相关字段
type StatusChangeEvent struct {
  SpaceId     uint64
  EventType   string        // "pipeline" or "job"
  TriggerInfo *TriggerInfo  // 触发方式信息，用于 trigger filter / 卡片展示
  Operator    string        // 操作人（通常等于触发人）

  PipelineRun *PipelineRun  // Pipeline 事件必有
  JobRun      *JobRun       // Job 事件才有

  UpdatedAt   time.Time

  Repost      int           // 被延迟重投递次数（超时/重复通知依赖）
  SceneType   platformpb.SceneType // 研发流程流水线标记（部分场景静默通知）
}
```

字段类型说明：

- `TriggerInfo *TriggerInfo`：对象（JSON）
- 样例：

```JSON
{"trigger_type": 4, "trigger_context": ""}
```

- `PipelineRun *PipelineRun` / `JobRun *JobRun`：对象（JSON）
- 样例：

```JSON
{"pipeline_run": {"run_id": 1, "status_str": "start", "notifications": []}}
```

- `UpdatedAt time.Time`：RFC3339 时间字符串（JSON）
- 样例：`"2024-03-08T10:45:11.798867544+08:00"`

- `SceneType platformpb.SceneType`：枚举（整数，PB enum 的数值）
- 样例：`0`（表示 `SCENE_TYPE_UNSPECIFIED`）


与通知强相关的访问方法：

- `GetNotifications()`：根据 `EventType` 返回 `PipelineRun.Notifications` 或 `JobRun.Notifications`（`internal/pipeline/pevent/pevent.go:107`）。

- `GetStatus()`：根据 `EventType` 返回 `PipelineRun.StatusStr` 或 `JobRun.StatusStr`（`internal/pipeline/pevent/pevent.go:127`）。


### 3.2 `pevent.PipelineRun`

```Go
type PipelineRun struct {
  RunID        uint64
  PipelineID   uint64
  PipelineName string
  Psm          string

  EngineRunID  int64
  RunSeq       uint64
  RunParams    map[string]interface{} // 运行参数（包含触发信息、提交人等）

  Status       dslpb.PipelineRunStatus
  StatusStr    string                  // 供通知 when.status 匹配的字符串状态

  RepoName     string
  BranchOrTag  string
  SpaceID      uint64
  YamlFilename string

  Notifications []*dslpb.Notification  // 本次事件需要处理的通知列表（pipeline 维度）
  CreatedBy     string                 // 触发/创建人（notifier_types: builder 等依赖）
  ControlPanel  pentity.ControlPanel   // 变量/引擎查询需要
}
```

字段类型说明：

- `RunParams map[string]interface{}`：任意 JSON 对象（map）
- 常见用途：承载触发上下文、提交人列表等（例如 `commits_uploader`）。

- 样例：

```JSON
{
  "build_reason": "xxx 手动触发",
  "commits_uploader": [
    {"username": "alice"},
    {"username": "bob"}
  ],
  "build_scene_info": {"scene_name": "发布单", "scene_url": "https://..."}
}
```

- `Status dslpb.PipelineRunStatus`：枚举（整数，PB enum 的数值）
- 说明：通知匹配主要使用 `StatusStr`（字符串），而不是 `Status`。

- `Notifications []*dslpb.Notification`：数组（Notification 对象列表）
- 样例：

```JSON
[
  {
    "name": "默认流水线事件",
    "type": 2,
    "when": {"status": ["start", "failed"]},
    "lark": {"notifier_types": ["builder"], "users": ["alice"]}
  }
]
```

- `ControlPanel pentity.ControlPanel`：枚举（int8，JSON 中通常表现为整数）
- 取值：`0=cn`、`1=us-ttp`（默认）、`2=eu-ttp`、`3=cn-stage`（`internal/pipeline/pentity/control_panel.go:8`）

- 样例：`1`


### 3.3 `pevent.JobRun`

```Go
type JobRun struct {
  ID            uint64
  JobID         string
  JobBuilder    string     // job 触发人（notifier_types: job_builder 依赖）
  EngineRunID   int64      // job 实例 id（用于查变量/引擎状态等）
  NameEN        string
  NameZH        string
  PipelineRunID uint64

  Status        dslpb.JobRunStatus
  StatusStr     string     // 供通知 when.status 匹配的字符串状态

  Notifications []*dslpb.Notification  // 本次事件需要处理的通知列表（job 维度）

  FailType      engineSDK.FailType
  FailReason    string
  Outputs       engineSDK.Outputs
  ActionOutputs []engineSDK.Outputs    // waiting 状态下的卡片交互按钮来源
  AtomUID       string
  ControlPanel  pentity.ControlPanel
  Tried         int
  Retry         int
}
```

字段类型说明：

- `Status dslpb.JobRunStatus`：枚举（整数，PB enum 的数值）
- 说明：通知匹配主要使用 `StatusStr`（字符串）。

- `Outputs engineSDK.Outputs`：任意 JSON 对象（通常是 `map[string]interface{}`）
- 样例：

```JSON
{"fail_notice_value": {"message": "xxx", "i18nCategory": {"zh": {"message": "失败"}}}}
```

- `ActionOutputs []engineSDK.Outputs`：数组（每个元素是一个对象），用于 `waiting` 状态下构造飞书卡片交互按钮
- 样例：

```JSON
[
  {
    "title": "Approve",
    "method": "post",
    "url": "/api/v1/user_define_method",
    "body": {"is_pass": "true", "job": 10001, "method_name": "user_define_method"}
  }
]
```

- `FailType engineSDK.FailType`：枚举（字符串/数值由 go-sdk 定义，卡片会展示 fail\_type 文本）


### 3.4 `pevent.TriggerInfo`

```Go
type TriggerInfo struct {
  TriggerType    platformpb.TriggerType // 触发类型（手动/定时/Git 等）
  TriggerContext []byte
}
```

字段类型说明：

- `TriggerType platformpb.TriggerType`：枚举（整数）
- 样例：`4`（例如手动触发）

- `TriggerContext []byte`：字节数组（JSON 中通常是 base64 字符串）
- 样例：`""`（空字符串表示无上下文）


额外说明：除 DSL 固化通知外，运行时还可能追加“实例级通知”（例如 pipeline\_run 维度的临时订阅），在事件构建时拼入 `PipelineRun.Notifications`（`app/pipelinerpc/biz/service/pipeline_event/event_builder.go:168`）。

## 4\. Notification 结构（字段解释）

通知结构定义：`idls/byted/devinfra/pipeline/dsl/notification.proto:7`。
为避免只靠“文件路径引用”导致上下文缺失，这里把通知系统用到的主要 proto 结构体与字段完整列出（按发送链路实际使用程度排序）。

### 4.1 `Notification`（顶层）

```ProtoBuf
message Notification {
  NotifyWhen when = 1;              // 触发条件
  LarkNotification lark = 2;        // 飞书通知配置
  Webhook webhook = 3;              // webhook 配置

  string name = 4;                  // 通知名称（日志/展示）
  NotificationType type = 5;        // LARK / WEBHOOK

  uint64 id = 6;                    // 通知 id（存储/管理）
  uint64 template_notification_id = 7; // 模板通知 id（存储/模板；当前无消费方）
  TriggerType trigger_type = 8;     // v1 yaml 兼容字段
}
```

字段类型说明：

- `when: NotifyWhen`：对象
- 样例：`{"status": ["start", "failed"], "timeout": 600}`

- `lark: LarkNotification`：对象（当 `type=LARK` 时必填）
- 样例：`{"users": ["alice"], "groups": ["oc_xxx"]}`

- `webhook: Webhook`：对象（当 `type=WEBHOOK` 时必填）
- 样例：`{"action_type": 1, "http_action": {"url": "https://...", "method": "POST"}}`

- `type: NotificationType`：枚举（PB enum 数值）
- 约定：`2=LARK`、`1=WEBHOOK`。


通知对象（JSON）完整样例（偏 DSL/PB 侧）：

```JSON
{
  "name": "默认流水线事件",
  "type": 2,
  "when": {"status": ["start", "failed"], "timeout": 0},
  "lark": {
    "notifier_types": ["builder"],
    "users": ["alice"],
    "groups": ["oc_123456"],
    "notification_method": 1,
    "group_notification_method": 0
  }
}
```

`Notification.type` 的执行分流：`app/pipelineeventrpc/biz/service/event/service.go:79`。

### 4.2 `NotificationType`（通知类型）

```ProtoBuf
enum NotificationType {
  NOTIFICATION_TYPE_UNSPECIFIED = 0;
  NOTIFICATION_TYPE_WEBHOOK = 1;
  NOTIFICATION_TYPE_LARK = 2;
}
```

### 4.1 顶层字段

- `when`：何时触发（条件）

- `lark`：飞书通知配置

- `webhook`：Webhook 配置

- `name`：通知名称（通常用于展示/日志）

- `type`：通知类型（`LARK` / `WEBHOOK`）

- `id`：通知 id（更多用于存储/管理；发送链路不依赖）

- `template_notification_id`：模板通知 id（当前没有消费方；发送链路不依赖）

- `trigger_type`：v1 yaml 兼容字段（发送链路不依赖）


### 4.2 `NotifyWhen`（触发条件）

定义：

```ProtoBuf
message NotifyWhen {
  repeated string status = 1; // format: <status>
  int32 timeout = 2;          // format: <duration>, unit: second
}
```

- `status[]`：命中哪些状态时触发（使用 `status_str` 进行匹配）
- 判断逻辑：`internal/pipeline/pevent/pevent.go:187`

- `timeout`：超时触发（秒）
- 实现逻辑：不是“到点直接发”，而是“在开始事件时延迟投递一条超时事件”，到点后再判断是否已终态/是否仍需触发。

- 参考：`app/pipelineeventrpc/biz/service/event/filter.go:120`


补充口径（timeout 计时起点）：

- **Pipeline**：从 `status_str == "start"` 的事件开始计时。

- **Job**：从收到该 job 的 `running` 事件开始计时。


字段类型说明：

- `status: repeated string`：字符串数组
- 样例：`["start", "succeeded", "failed"]`

- `timeout: int32`：整数（秒）
- 样例：`1800`

<br>


### 4.3 `LarkNotification`（飞书通知配置）

```ProtoBuf
message LarkNotification {
  repeated string notifier_types = 1; // 关注人类型/角色（运行时会解析成 users）
  repeated string users = 2;          // 用户（运行时会转 email）
  repeated string groups = 3;         // 群（chat_id / open_chat_id）

  NotificationMethod notification_method = 6;       // 个人消息策略
  NotificationMethod group_notification_method = 7; // 群消息策略

  Repeat repeat = 8;                 // 重复通知
  repeated LarkCard cards = 11;      // 自定义卡片内容

  // deprecated 字段省略（CI 兼容）

  bool with_trigger_filter = 16;     // 是否按触发类型过滤
  repeated TriggerFilter trigger_types = 17; // 允许的触发类型集合
}
```

字段类型说明：

- `notifier_types: repeated string`：字符串数组（关注人类型/角色）
- 样例：`["builder", "commits_uploader"]`

- `users: repeated string`：字符串数组（用户标识；运行时会转 email）
- 样例：`["alice", "bob"]`

- `groups: repeated string`：字符串数组（群 chat\_id 或 open\_chat\_id）
- 样例：`["oc_123456"]`

- `notification_method / group_notification_method: NotificationMethod`：枚举（PB enum 数值）
- 样例：`1`（MENTIONED）、`2`（URGENT）、`3`（IGNORE\_PERSONAL）

- `repeat: Repeat`：对象
- 样例：`{"count": 3, "interval": 300}`

- `cards: repeated LarkCard`：对象数组
- 样例：`[{"title": "链接", "content": "[点击查看](https://...)"}]`

- `with_trigger_filter: bool`：布尔
- 样例：`true`

- `trigger_types: repeated TriggerFilter`：枚举数组
- 样例：`[1, 2]`（MANUAL/CRON）


### 4.4 `NotificationMethod`（发送方式：@/加急/忽略个人）

```ProtoBuf
enum NotificationMethod {
  NOTIFICATION_METHOD_UNSPECIFIED = 0;
  NOTIFICATION_METHOD_MENTIONED = 1;         // 在卡片正文中 @ users
  NOTIFICATION_METHOD_URGENT = 2;            // 加急（应用内加急）
  NOTIFICATION_METHOD_IGNORE_PERSONAL = 3;   // 不发个人消息
}
```

### 4.5 `Repeat`（重复通知）

```ProtoBuf
message Repeat {
  int32 count = 1;     // 重复次数
  int32 interval = 2;  // 间隔（秒）
}
```

字段类型说明：

- `count: int32`：整数（重复次数，不包含首次命中）
- 样例：`3`（总计 4 次通知）

- `interval: int32`：整数（秒）
- 样例：`300`


### 4.6 `LarkCard`（自定义卡片字段）

```ProtoBuf
message LarkCard {
  string title = 1;   // format: <expr_string>
  string content = 2; // format: {<expr_string>|<lark_markdown>}
}
```

字段类型说明：

- `title: string`：字符串（可包含表达式）
- 样例：`"构建号"`

- `content: string`：字符串（可为 markdown，亦可包含表达式）
- 样例：`"No.{{sys.pipeline.run_seq}}"`


### 4.7 `Webhook` / `HttpAction` / `PipelineAction`

```ProtoBuf
message Webhook {
  WebhookActionType action_type = 1;
  HttpAction http_action = 2;
  PipelineAction pipeline_action = 3;
}

enum WebhookActionType {
  WEBHOOK_ACTION_TYPE_UNSPECIFIED = 0;
  WEBHOOK_ACTION_TYPE_HTTP = 1;
  WEBHOOK_ACTION_TYPE_PIPELINE = 2;
}

message HttpAction {
  string url = 1;
  string method = 2;               // GET/POST/PUT
  map<string, string> headers = 3;
  string body = 4;                 // json string（或表达式渲染后字符串）
}

message PipelineAction {
  int32 pipeline_id = 1;
}
```

字段类型说明：

- `action_type: WebhookActionType`：枚举（PB enum 数值）
- 样例：`1`（HTTP）

- `http_action: HttpAction`：对象
- 样例：`{"url": "https://example.com/hook", "method": "POST", "headers": {"x-foo": "bar"}, "body": "{\"k\":\"v\"}"}`

- `headers: map<string,string>`：对象（key/value 都是字符串）
- 样例：`{"Content-Type": "application/json", "x-pipeline-proxy": "row"}`

- `body: string`：字符串（通常承载 JSON 文本；可包含表达式）
- 样例：`"{\"build_id\": {{sys.pipeline.run_id}}, \"status\": \"{{sys.pipeline.status}}\"}"`


### 4.8 `NotificationGroup`（通知组：group\_id + version + snapshot）

```ProtoBuf
message NotificationGroup {
  uint64 group_id = 1;                 // 0 表示创建新 group
  uint64 version = 2;
  repeated Notification notifications = 3;
}
```

## 5\. 状态枚举（`when.status` 可用值）

`when.status` 使用 **字符串状态**，定义于 `internal/pipeline/pevent/pevent.go:18`，常见包括：

- `start`

- `running`

- `waiting`

- `blocking`

- `succeeded`

- `failed`

- `cancelled` / `cancelling`

- `rollbacking` / `rollbacked`

- `ignored` / `skipped`


建议：配置时以产品侧透出的状态字符串为准；若需严格对齐，可直接对照上述常量。

## 6\. 飞书通知（Lark）

### 6.1 收件人配置

定义：`idls/byted/devinfra/pipeline/dsl/notification.proto:65`

- `lark.users[]`：用户列表（运行时会做去重并转换为 email，再发送）
- 去重与转 email：`app/pipelineeventrpc/biz/service/notify/service.go:49`

- `lark.groups[]`：群列表
- 支持数字 chat\_id，发送前会转换为 open\_chat\_id：`app/pipelineeventrpc/biz/provider/lark/service.go:302`


### 6.2 `notifier_types[]`（关注人类型/角色）

字段：`lark.notifier_types[]`
用途：用于配置“关注人类型”，运行时会解析为具体用户并 **追加** 到 `lark.users[]`。
解析逻辑：`app/pipelineeventrpc/biz/service/event/notification_helper.go:40`
从代码可推断的内置值：

- `builder` / `pipeline_builder` / `stream_builder` / `{{sys.pipeline.trigger_user}}`
- 解释：映射为 `pipelineRun.CreatedBy`（触发/创建人）

- `commits_uploader`
- 解释：从 `pipelineRun.RunParams["commits_uploader"]` 解析提交人 username，追加为通知用户

- `job_builder`
- 解释：Job 事件时使用 `jobRun.JobBuilder`（原子/Job 的触发人）

- `bits*` / `custom_role*`
- 解释：视为“空间角色”，会通过权限系统查 principals 并追加


### 6.3 发送方式（个人/群）

字段：

- `lark.notification_method`：个人消息的策略

- `lark.group_notification_method`：群消息的策略


枚举：

```ProtoBuf
enum NotificationMethod {
  // Default notification method
  NOTIFICATION_METHOD_UNSPECIFIED = 0;
  // Mention users/groups in Lark message
  NOTIFICATION_METHOD_MENTIONED = 1;
  // Send a Lark message with urgency
  NOTIFICATION_METHOD_URGENT = 2;
//  // Send a Lark message with urgency and short messages
//  NOTIFICATION_METHOD_URGENT_WITH_SMS = 3;
//  // Send a Lark message with urgency and phone call
//  NOTIFICATION_METHOD_URGENT_WITH_PHONE_CALL = 4;
  // Don't send a Lark message personally
  NOTIFICATION_METHOD_IGNORE_PERSONAL = 3;
}
```

目前实现关注的值：

- `MENTIONED`：卡片正文中追加 `<at email=...></at>`（实现位于卡片构建：`app/pipelineeventrpc/biz/service/card/bits.go:318`）

- `URGENT`：消息发送成功后调用“应用内加急”接口（`app/pipelineeventrpc/biz/service/notify/service.go:141`）

- `IGNORE_PERSONAL`：不发送个人消息，只发送群消息（`app/pipelineeventrpc/biz/service/notify/service.go:75`）


### 6.4 重复通知（Repeat）

字段：`lark.repeat.count` / `lark.repeat.interval`（定义：`idls/byted/devinfra/pipeline/dsl/notification.proto:57`）
口径：`repeat.count` **不包含首次命中**。

- 例：配置 `count=3` 时，整体会收到 `1（首次） + 3（重复） = 4` 次通知。


实现方式：通过延迟重新投递事件来实现重复（`app/pipelineeventrpc/biz/service/event/filter.go:87`）。

### 6.5 自定义卡片内容（LarkCard）

字段：`lark.cards[]`（定义：`idls/byted/devinfra/pipeline/dsl/notification.proto:128`）

- `title`：卡片字段标题（支持表达式）

- `content`：卡片字段内容（支持表达式/markdown）


渲染方式：构建卡片时把每个 `card` 以 `**[title]**：content` 的形式拼到正文（`app/pipelineeventrpc/biz/service/card/card_builder.go:185`）。

### 6.6 触发方式过滤（Trigger Filter）

字段：

- `lark.with_trigger_filter`：是否启用触发方式过滤

- `lark.trigger_types[]`：允许的触发类型集合（枚举 `TriggerFilter`）


实现：不匹配则直接跳过通知（abort）

- 逻辑：`app/pipelineeventrpc/biz/service/event/filter.go:143`

- 映射：`app/pipelineeventrpc/biz/service/event/filter.go:155`


## 7\. Webhook 通知

定义：`idls/byted/devinfra/pipeline/dsl/notification.proto:138`

### 7.1 动作类型

- `webhook.action_type = HTTP`：发起 HTTP(S) 请求（已实现）

- `webhook.action_type = PIPELINE`：触发流水线（当前自由流水线场景 **不执行发送**；实现为空，属于明确的“无效果配置”）


HTTP 执行逻辑：`app/pipelineeventrpc/biz/service/webhook/service.go:43`

### 7.2 HttpAction 字段

- `url`：请求 URL（需包含协议头）

- `method`：支持 `GET/POST/PUT`（大小写不敏感）

- `headers`：自定义请求头
- 会默认补齐 `Content-Type: application/json`

- 会注入一组“事件上下文 header”（便于下游识别当前事件）

- 参考：`app/pipelineeventrpc/biz/service/webhook/service.go:134`

- `body`：请求 body（string）
- 若为空，会对 pipeline 事件补兼容默认 body：`{"build_id":..., "status":...}`

- 参考：`app/pipelineeventrpc/biz/service/webhook/service.go:177`


#### 7.2.1 Webhook 默认注入的上下文 Headers（重要）
当 `HttpAction.headers` 未显式覆盖时，系统会在请求头中注入一组默认字段（兼容历史 ByteCycle，同时提供 Bits 自己的字段）。这些字段会随事件类型不同而不同：
**Job 事件（****`event.EventType == "job"`** **）注入：** 

| Header Key | 取值来源（运行时字段） |
| --- | --- |
| `status` | `event.JobRun.StatusStr` |
| `bits-job-status` | `event.JobRun.StatusStr` |
| `instance-id` | `strconv.FormatInt(event.JobRun.EngineRunID, 10)` |
| `instance_id` | `strconv.FormatInt(event.JobRun.EngineRunID, 10)` |
| `bits-job-id` | `strconv.FormatInt(event.JobRun.EngineRunID, 10)` |
| `bits-job-name` | `event.JobRun.NameZH` |
| `bits-job-name-i18n` | `event.JobRun.NameEN` |

**Pipeline 事件（****`event.EventType == "pipeline"`** **）注入：** 

| Header Key | 取值来源（运行时字段） |
| --- | --- |
| `bc-build-id` | `strconv.FormatUint(event.PipelineRun.RunID, 10)` |
| `bc-build-status` | `event.PipelineRun.StatusStr` |
| `bc-pipeline-id` | `strconv.FormatUint(event.PipelineRun.PipelineID, 10)` |
| `bc-build-no` | `strconv.FormatUint(event.PipelineRun.RunSeq, 10)` |
| `bc-pipeline-name` | `event.PipelineRun.PipelineName` |
| `bc-build-operator` | `event.PipelineRun.CreatedBy` |
| `bits-pipeline-run-id` | `strconv.FormatUint(event.PipelineRun.RunID, 10)` |
| `bits-pipeline-run-status` | `event.PipelineRun.StatusStr` |
| `bits-pipeline-id` | `strconv.FormatUint(event.PipelineRun.PipelineID, 10)` |
| `bits-pipeline-run-no` | `strconv.FormatUint(event.PipelineRun.RunSeq, 10)` |
| `bits-pipeline-name` | `event.PipelineRun.PipelineName` |
| `bits-pipeline-name-i18n` | `event.PipelineRun.PipelineName`（当前实现中与中文同值） |
| `bits-pipeline-run-trigger-user` | `event.PipelineRun.CreatedBy` |

以上 header key 常量定义在：`app/pipelineeventrpc/biz/constvar/constvar.go:43`，注入逻辑在：`app/pipelineeventrpc/biz/service/webhook/service.go:134`。

#### 7.2.2 跨区代理（ROW/CN 网络）
若请求头里出现 `x-pipeline-proxy: row`（大小写不敏感的 header 名、value 区分大小写），系统会将请求改写为走统一代理地址，并把原始请求信息封装进代理 body 中（`app/pipelineeventrpc/biz/service/webhook/service.go:235`）。

### 7.3 成功/失败判定与记录
- `status_code == 200` 视为成功，否则视为失败并记录事件
- 记录逻辑：`app/pipelineeventrpc/biz/service/webhook/service.go:190`


## 8\. 变量与表达式（配置中写 `{{ ... }}`）
通知支持在字符串字段中写表达式 `{{ ... }}`，运行时会基于上下文变量进行渲染。

### 8.1 是否触发渲染
判断方式：把 Notification 序列化为 JSON，只要包含 `{{` 与 `}}` 就尝试渲染（`app/pipelineeventrpc/biz/utils/var.go:14`）。

### 8.2 变量来源（粗粒度）
渲染用的 `varMap`（变量字典）由 `pipelineeventrpc` 在处理通知时构造，构造逻辑会用到以下字段（均来自 `pevent.StatusChangeEvent`）：
- `event.EventType`：区分 pipeline / job（决定用 pipeline 变量还是 job 变量）
- `event.PipelineRun.StatusStr`：判断是否为 pipeline 的 `start` 状态（`start` 时走 pipeline 变量来源）
- `event.PipelineRun.ControlPanel`：查询引擎/变量时需要
- `event.JobRun.EngineRunID`：Job 事件需要用它查询 job var map
- `event.JobRun.ControlPanel`：同上
实际策略：

- **Pipeline 事件**：
- 当 `event.PipelineRun.StatusStr == "start"`：变量优先从 **pipeline 级 variables** 获取。
- 否则：尝试定位“最近运行的 jobs”，对每个 job 拉取 job var map 并合并为一个 `varMap`（用于渲染通知）。
- **Job 事件**：
- 直接拉取当前 job 的 var map（key 为空间/引擎变量系统定义的变量名），作为 `varMap`。

### 8.3 渲染落点
- `lark.users[]` / `lark.groups[]`：会先单独渲染一次，并支持把 `"a,b"` 逗号分隔展开为多用户（`app/pipelineeventrpc/biz/service/event/notification_helper.go:166`）
- 整个 `Notification`：随后对整个通知对象进行渲染，意味着 `webhook.http_action.body`、`lark.cards[].content` 等理论上都可包含表达式（`app/pipelineeventrpc/biz/service/event/notification_helper.go:155`）。

## 9\. 通知过滤与触发机制（实现细节）
通知是否执行由 `pipelineeventrpc` 的过滤器决定：`app/pipelineeventrpc/biz/service/event/filter.go:19`

- Webhook：仅校验 `when.status` 命中
- Lark：
- `when.status` 命中（普通通知）
- `when.timeout`（超时通知）
- `lark.repeat`（重复通知）
- Job 自动重试失败：只在最后一次失败时通知（abort 逻辑：`app/pipelineeventrpc/biz/service/event/filter.go:70`）