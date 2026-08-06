# Field Metadata: 流水线触发器 / Pipeline Trigger
本文覆盖「自由流水线/平台流水线」触发器（`dsl.Trigger`），忽略仅在 CI 流水线中出现的结构（例如 `dsl.CICronTrigger`、`SaveCICronTriggers` 等接口字段）。

## 1) 触发器能力综述

流水线触发器用于在满足条件时触发流水线执行。

- 默认动作：触发对应流水线执行（RunPipelineLite / RunGitPipeline）。

- 同样还有 HTTP Action 作为触发器的动作，但是 AI 生成流水线的场景咱不支持这个配置，相关字段留空。


从触发来源看，大面上分两类：

1. **Git 事件触发**


- MR 事件触发：`MRTrigger`（创建/更新/合入等）

- Push 事件触发：`GitPushTrigger`（push、delete、tag pushed）


2. **定时触发**


- 纯定时触发：`CronTrigger`

- 以及：Git 触发器也支持“时间约束”`TimeRestriction.cron`，效果是：**Git 事件发生后不立即跑流水线，而是写入一条待调度事件，等到下一个 cron 时间点再触发执行**。


## 2) 触发器配置的 proto

### 2.1 触发器顶层结构：`dsl.Trigger`

```ProtoBuf
// idls/byted/devinfra/pipeline/dsl/trigger.proto

message Trigger {
  oneof trigger {
    MRTrigger mr = 2;
    GitPushTrigger git_push = 3;
    CronTrigger cron = 5;
  }

  // 触发器变量组
  bc.varstore.VarGroup var_group = 14;
  // 对应的模版触发器id
  uint64 template_trigger_id = 20;

  bool disabled = 100;

  // trigger a HTTP(S) request
  // exclusive with trigger a pipeline, if there is http action
  // it will not trigger pipeline
  HttpAction http_action = 30;

  string frontend_key = 200;
}
```

#### 通用行为（结合 `app/triggerrpc` 推断）

- `disabled=true`：触发器会被过滤掉，不会执行。

- `http_action != nil`：触发器会转为“Webhook 执行器”，仅调用 HTTP，不再触发流水线。

- `var_group`：会被存入变量系统；触发时会创建一次变量赋值（assignment），并把 `assignment_id` 传给下游流水线执行。

- `frontend_key`：目前主要用于迁移/回填映射，不影响触发逻辑。

- `template_trigger_id`：会被持久化，但在触发执行路径中未看到直接使用（更像是模板关联/审计/回溯字段）。


### 2.2 Git 触发器：pattern、事件、时间约束、机器人策略

```ProtoBuf
enum PatternSyntax {
  PATTERN_SYNTAX_UNSPECIFIED = 0;
  // glob: https://www.digitalocean.com/community/tools/glob
  PATTERN_SYNTAX_GLOB = 1;
  // re2: https://github.com/google/re2/wiki
  PATTERN_SYNTAX_REGEX = 2;
}

enum MREvent {
  MR_EVENT_UNSPECIFIED = 0;
  // 创建
  MR_EVENT_OPENED = 1;
  // 源分支代码变更
  MR_EVENT_PUSHED = 2;
  // 合入
  MR_EVENT_MERGED = 3;
  // 关闭
  MR_EVENT_CLOSED = 4;
  // 恢复
  MR_EVENT_REOPENED = 5;
  // 属性更新
  MR_EVENT_UPDATED = 6;
}

// bot triggered event strategy:
// in the pipeline senario default is ignore such event
enum BotEventStrategy {
  BOT_EVENT_STRATEGY_UNSPECIFIED = 0;
  BOT_EVENT_STRATEGY_IGNORED = 1;        // ignore such event on bot triggered event
  BOT_EVENT_STRATEGY_LATEST_AUTHOR = 2;  // find latest human author of the commit
}

message TimeRestriction {
  // NOTICE: group_wait and cron are mutually exclusive.
  //
  // If this trigger happens multi times during group_wait,
  // the Pipeline will run only once with the last trigger.
  // 0 means the Pipeline will run immediately when trigger happens.
  //
  // format: <duration>
  string group_wait = 1;

  // NOTICE: group_wait and cron are mutually exclusive.
  //
  // format: <cron>
  string cron = 2;

  // The timezone used for cron expression.
  // Common timezone examples:
  // - "America/New_York": US East
  // - "America/Los_Angeles": US West
  // - "Asia/Shanghai": CN/SG
  string timezone = 3;
}

message MRTrigger {
  uint64 id = 1;
  string name = 2;

  // format: [domain/]group_name/repository_name
  // domain is optional.
  // By default, Codebase or Gerrit domain will be applied automatically based on where the repository is.
  string repository = 3;
  repeated string repositories = 13;

  repeated MREvent events = 4;

  // pattern_syntax specifies which syntax used in string matching,
  // works for these fields:
  // - source_branches
  // - target_branches
  // - paths
  // - incremental_paths
  PatternSyntax pattern_syntax = 5;

  // format: <glob>|<regex>
  repeated string source_branches = 6;

  // format: <glob>|<regex>
  repeated string target_branches = 7;

  // format: <glob>|<regex>
  repeated string paths = 8;

  // format: <glob>|<regex>
  // specify changed paths between versions within an MR
  repeated string incremental_paths = 9;

  // format: <glob>|<regex>
  repeated string mr_titles = 10;

  // format: <glob>|<regex>
  repeated string commit_messages = 11;

  TimeRestriction time_restriction = 12;
  // listen to bot trigger event
  BotEventStrategy bot_event_strategy = 20;
}

enum GitPushEvent {
  GIT_PUSH_EVENT_UNSPECIFIED = 0;
  GIT_PUSH_EVENT_PUSHED = 1;
  GIT_PUSH_EVENT_DELETED = 2;
  GIT_PUSH_EVENT_TAG_PUSHED = 3;
}

message GitPushTrigger {
  uint64 id = 1;
  string name = 2;

  // format: [domain/]group_name/repository_name
  // domain is optional.
  // By default, Codebase or Gerrit domain will be applied automatically based where the repository is.
  string repository = 3;
  repeated string repositories = 13;

  repeated GitPushEvent events = 4;

  // pattern_syntax specifies which syntax used in string matching,
  // works for these fields:
  // - branches
  // - paths
  // - tags
  PatternSyntax pattern_syntax = 5;

  // format: <glob>|<regex>
  repeated string branches = 6;

  // format: <glob>|<regex>
  repeated string tags = 7;

  // format: <glob>|<regex>
  repeated string paths = 8;

  // format: <glob>|<regex>
  repeated string commit_messages = 10;

  TimeRestriction time_restriction = 11;

  // filter actor in codebase pushed message
  repeated string actor_usernames = 12;

  // listen to bot trigger event
  BotEventStrategy bot_event_strategy = 20;
}
```

#### Git 触发器字段解释（结合代码推断）

- `id`
	- `0` 表示新建；保存时会生成 trigger\_id。

- `name`
	- 触发器名称；用于展示与审计。

- `repository` 与 `repositories`
	- 保存时会做“互补”：
		- `repository` 非空但不在 `repositories` 里，会 append 进去；

		- `repository` 为空且 `repositories` 非空，会用第一个补齐。

- `events`
	- 必须包含当前事件，否则不触发。

- `pattern_syntax` + 各类匹配字段（分支/路径/标题/commit message）
	- `PATTERN_SYNTAX_REGEX`：走 RE2 正则匹配。

	- `PATTERN_SYNTAX_GLOB`：走 glob 匹配。

	- MR 匹配字段：`source_branches` / `target_branches` / `paths` / `mr_titles` / `commit_messages`。

	- Push 匹配字段：`branches`（branch push）/ `tags`（tag push）/ `paths` / `commit_messages`。

	- `incremental_paths`：当前过滤逻辑未使用该字段，需要确认预期。

- `actor_usernames`
	- Push 场景可按“推送人用户名”过滤；配置后如果当前 push 人不在列表里会直接拦截。

- `time_restriction.cron` / `timezone`
	- 配置了 `cron` 时，Git 事件不会立即触发流水线，而是写入一条待调度事件，等到下一次 cron 时间点再触发执行。

- `time_restriction.group_wait`
	- 当前未在 `app/triggerrpc` 中发现使用点，需要确认语义/是否废弃。

- `bot_event_strategy`
	- 当触发人是系统用户/机器人：
		- 若策略不是 `LATEST_AUTHOR`，直接拦截；

		- 若是 `LATEST_AUTHOR`，会尝试从 MR commits 中找到最近真人作者作为触发人继续执行。


### 2.3 定时触发器：`CronTrigger`

```ProtoBuf
message CronTrigger {
  uint64 id = 1;
  string name = 2;
  string cron = 3;
  string timezone = 4;
  string triggered_by = 5;
}
```

#### 字段解释（结合代码推断）

- `cron`
	- 必填；保存时会校验非空且不允许带 `TZ=`（时区必须走字段 `timezone`）。

	- 下一次触发时间由 cron + timezone 计算。

- `timezone`
	- 保存时会做“时区别名归一”，目前支持：
		- `UTC`

		- `UTC+8` / `Asia/Shanghai`

		- `UTC-7` / `UTC-8` / `America/Los_Angeles`

		- `UTC-4` / `UTC-5` / `America/New_York`

	- 其他时区会报错“不支持”。

- `triggered_by`
	- 定时触发时作为执行用户名下发给流水线。

	- 当编辑已有触发器且关键字段发生变化，系统会把 `triggered_by` 改成操作者。

	- 执行时如果为空，会 fallback 到 `CreatedBy`。


## 3) “动作”：触发流水线

这部分不是“配置字段”，但会影响你设计 `var_group` 的变量引用：

- Git 事件触发 run\_params 常见键：
	- `repo_id` / `repo_name` / `repo_url` / `commit_id`

	- `git_event`（字符串化事件类型）

	- `active_trigger`（trigger\_id）

	- MR 场景还会有：`git_mr_id`、`git_mr_title`、`source_branch`、`target_branch`、`work_item_ids` 等

- Cron 触发 run\_params 更轻量：`active_trigger`、`cron_tab`、`type`/`type_cn` 等


## 4) 非基础类型：`var_group`（变量组）用法补充

`dsl.Trigger.var_group` 类型为 `bc.varstore.VarGroup`。为了让 AI 能“正确产出可保存的配置”，这里把相关 proto 一并放出（精简到触发器最常用的结构）。

### 4.1 VarGroup / StringInMultiLang / VarDefinition（proto）

```ProtoBuf
// idls/byted/bc/varstore/shared.proto
message StringInMultiLang {
  // 必填默认值
  string value = 1;
  string cn = 2;
  string en = 3;
  string lang = 4;
  map<string, string> texts = 5;
  string starling_key = 10;
}
```

```ProtoBuf
// idls/byted/bc/varstore/variable.proto

enum VarKind {
  VAR_KIND_UNSPECIFIED = 0;
  VAR_KIND_BOOL = 1;
  VAR_KIND_INTEGER = 2;
  VAR_KIND_STRING = 3;
  // json_array
  VAR_KIND_ARRAY = 4;
  // json_object
  VAR_KIND_OBJECT = 5;
  VAR_KIND_MAP = 11;
  // 超长的只读值
  VAR_KIND_TOS_FILE = 12;
}

message VarValue {
  oneof value {
    bool boolean = 1;
    int64 number = 2;
    string text = 3;
    string json_array = 4;
    string json_object = 5;
  }
  // 由服务端编码并赋值, 调用方设置无效.
  string raw_json = 10;
}

enum UIInputType {
  UI_INPUT_TYPE_UNSPECIFIED = 0;
  UI_INPUT_TYPE_TEXT = 1;
  UI_INPUT_TYPE_SELECT = 2;
  UI_INPUT_TYPE_MULTI_SELECT = 3;
  UI_INPUT_TYPE_GIT_BRANCH = 4;
  UI_INPUT_TYPE_MEEGO_TASK = 5;
  UI_INPUT_TYPE_MULTILINE_TEXT = 6;
  UI_INPUT_TYPE_PERSON = 7;
  UI_INPUT_TYPE_TIMESTAMP = 8;
}

message Option {
  string name = 1;
  string label = 2;
}

message VarUIOption {
  UIInputType input_type = 1;
  // Deprecated
  repeated string options = 2;
  bool required = 3;
  bool final = 4;
  repeated Option new_options = 5;
}

message VarDefinition {
  // ^[a-zA-Z][a-zA-Z0-9_]*$
  string name = 1;

  VarKind kind = 2;

  // 对应 UI 中 "描述" 字段
  StringInMultiLang short_desc = 3;

  // 对应 UI 中 "解释" 字段
  StringInMultiLang long_desc = 4;

  // 默认值, 用于自定义变量
  VarValue default_value = 5;

  // 由服务端设置该值, 调用方设置无效.
  string full_name = 6;

  bool sensitive = 10;
  VarUIOption ui_option = 11;
  bool deprecated = 12;

  // 在变量替换前，是否需要先json反序列化，仅kind为json类型时生效
  bool need_load_when_render = 13;
}
```

```ProtoBuf
// idls/byted/bc/varstore/group.proto（节选：完整 message 在该文件中）

enum VarGroupNamespace {
  VAR_GROUP_NAMESPACE_UNSPECIFIED = 0;
  // 系统变量组: key_prefix: 'sys.'
  VAR_GROUP_NAMESPACE_SYS = 1;
  // 自定义变量组: key_prefix: 'custom.'
  VAR_GROUP_NAMESPACE_CUSTOM = 2;
  // 兼容旧的ci变量
  VAR_GROUP_NAMESPACE_OLD_CI_PIPELINE = 3;
}

enum SysProvider {
  SYS_PROVIDER_UNSPECIFIED = 0;
  SYS_PROVIDER_WORKSPACE = 1;
  SYS_PROVIDER_BUILD = 2;
  SYS_PROVIDER_TRAIN = 3;
  SYS_PROVIDER_REQUIREMENT = 4;
  SYS_PROVIDER_TEMPLATE = 5;
  SYS_PROVIDER_GIT = 6;
  SYS_PROVIDER_WORKFLOW = 7;
  SYS_PROVIDER_RELEASE_TICKET = 8;
  SYS_PROVIDER_DEVELOPMENT_TASK = 9;
  SYS_PROVIDER_MESSAGE = 10;
  SYS_PROVIDER_PIPELINE = 11;
  SYS_PROVIDER_CI_PIPELINE = 12;
  SYS_PROVIDER_CI_PIPELINE_ENVIRONMENT_VARIABLE = 13;
}

enum CustomScope {
  CUSTOM_SCOPE_UNSPECIFIED = 0;
  CUSTOM_SCOPE_WORKSPACE = 1;
  CUSTOM_SCOPE_PIPELINE = 2;
  CUSTOM_SCOPE_DEV_TASK = 3;
  CUSTOM_SCOPE_RELEASE_TICKET = 4;
  CUSTOM_SCOPE_PROJECT = 5;
  CUSTOM_SCOPE_GIT_REPO = 6;
  CUSTOM_SCOPE_BYTE_TREE = 7;
}

message VarGroup {
  uint64 group_id = 1;
  StringInMultiLang name = 2;
  int64 version = 3;
  StringInMultiLang description = 4;

  VarGroupNamespace namespace = 5;

  uint64 workspace_id = 6;
  uint64 bits_workspace_id = 11;

  SysProvider provider = 7;
  CustomScope custom_scope = 12;

  string key_prefix = 8;

  repeated VarDefinition var_definitions = 9;

  int64 constraints = 10;

  string created_by = 100;
  int64 created_at = 101;
  string updated_by = 103;
  int64 updated_at = 104;
}
```

### 4.2 VarGroup 在触发器里的语义（结合代码推断）

- 保存触发器时：会创建/升级变量组，并把返回的 `group_id/version` 绑定到触发器实体。

- 触发执行时：会用 `(run_params, trigger_id, var_group_id, username)` 创建一次 assignment（变量赋值快照），然后把 `assignment_id` 传给下游流水线执行。


### 4.3 非基础类型示例（便于 AI 生成）

#### 示例 A：Push 触发（触发流水线）

```JSON
{
  "trigger": {
    "git_push": {
      "id": 0,
      "name": "push-main",
      "repositories": ["devinfra/hagrid"],
      "events": ["GIT_PUSH_EVENT_PUSHED"],
      "pattern_syntax": "PATTERN_SYNTAX_GLOB",
      "branches": ["main"],
      "paths": ["**/*.go"]
    }
  },
  "disabled": false
}
```

#### 示例 B：Cron 触发（跑流水线），并指定触发人

```JSON
{
  "trigger": {
    "cron": {
      "id": 0,
      "name": "every-night",
      "cron": "0 2 * * *",
      "timezone": "Asia/Shanghai",
      "triggered_by": "alice"
    }
  },
  "disabled": false
}
```

#### 示例 C：var\_group（自定义变量组）

```JSON
{
  "group_id": 0,
  "name": {"value": "触发器自定义变量"},
  "description": {"value": "用于触发器运行时注入/校验的自定义变量"},
  "namespace": "VAR_GROUP_NAMESPACE_CUSTOM",
  "custom_scope": "CUSTOM_SCOPE_PIPELINE",
  "key_prefix": "custom.",
  "var_definitions": [
    {
      "name": "notify_channel",
      "kind": "VAR_KIND_STRING",
      "short_desc": {"value": "通知频道"},
      "ui_option": {"input_type": "UI_INPUT_TYPE_TEXT", "required": true}
    }
  ]
}
```

## 5) 重要约束（AI 生成配置时要遵守）

- **单流水线触发器数量上限**：保存时校验触发器数量上限为 10。

- **timezone 不允许写在 cron 表达式里**：含 `TZ=` 会被拒绝；时区必须走字段 `timezone`。

- **机器人触发策略**：希望 bot push / bot mr 也能触发时，需要显式设置 `bot_event_strategy=LATEST_AUTHOR`，否则会被拦截。

- **不打扰名单**：Cron/Git 都有“不打扰用户名单”过滤器；该名单不在 trigger proto 中配置，属于系统侧策略。