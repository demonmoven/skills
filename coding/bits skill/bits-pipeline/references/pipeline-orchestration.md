# Field Metadata: 流水线编排 / Pipeline Orchestration
这个知识库覆盖了生成流水线编排所需要的字段知识。
本文档仅包含自由流水线中与编排相关的字段，忽略了仅在 CI 流水线中出现的字段结构，CI 的字段在生成编排的时候留空即可。

## Proto 定义

流水线编排的数据结构在 protobuf 中的定义如下：

```ProtoBuf
message Pipeline {
  // Schema version
  // - v1: https://bytedance.feishu.cn/wiki/wikcn98ZWMAHiIwkxhJNeBaH5Bd#
  // - v2: current version
  string schema_version = 1;

  // Identity of this pipeline, hidden from user-perspective
  uint64 id = 2;

  // Name of this pipeline
  // Value: <multi_lang_string>
  I18nString name = 3 [(infra.i18n) = {
    key_format: "#id/name",
  }];

  // Description of this pipeline
  // Value: <multi_lang_string>
  I18nString desc = 4 [(infra.i18n) = {
    key_format: "#id/desc",
  }];

  // Defines stages in this pipeline, stages run in sequential
  // A stage is composed of jobs
  repeated Stage stages = 6;
}

message Stage {
  // Identity of this stage, unique in a pipeline, hidden from user-perspective
  string id = 1;

  // Name of this stage
  // <multi_lang_string>
  I18nString name = 2;

  // Run jobs in this stage only when a condition is met
  // 产品侧不展示
  string if = 3;

  // Specify jobs in this stage
  // By default, jobs in a stage are run in parallel
  // format: <identity>: <job>
  repeated Job jobs = 4;
}

message I18nString {
  // 默认文案的内容
  string value = 1;
  // value中的文案的语言代码, 如 en, zh等
  string lang = 2;
  // 多语言文案
  // key: 语言代码
  // value: 对应语言的文本内容
  map<string, string> texts = 3;
}

message Job {
  // id of job
  string id = 1;

  // Name of this job
  // <multi_lang_string>
  I18nString name = 2;

  // Run jobs in this stage only when a condition is met.
  string if = 3;
  // 跳过执行，与if互斥 https://meego.larkoffice.com/bitsdevops/story/detail/4970005816
  string if_skip = 107;

  // Specify ids of jobs that this job is depending on
  // (-- api-linter: core::0158::response-plural-first-field=disabled --)
  repeated string depends_on = 4;

  // 功能同 if. 用于 UI 模式下, 设置用户不可见的 if 条件, 如选择器场景.
  // 当 if 和 extra_if 同时设置, 它们之间是 And 关系.
  string extra_if = 5;

  // Start this job manually
  bool manual = 6;

  // for atom ref: [10, 20)
  // e.g., 'job_atom/scm_compile'
  string uses = 10;

  // atom inputs
  google.protobuf.Struct inputs = 11;

  // true is support do some operation in case of job-run timeout
  bool support_timeout = 100;

  // <duration>
  // default: 1 hour(for script-type jobs), 90 days(for service-type jobs),unit seconds
  // max: same as default
  int32 timeout = 101;

  OnFailedRetryPolicy retry = 102;

  // specify actions to do when job failed
  OnFailed on_failed = 103;

  // specify actions to do on timeout
  // by default, script-type jobs will fail on timeout, and for service-type jobs, it's a nop
  OnTimeout on_timeout = 104;

  // Specify manual operations
  JobOperations manual_operations = 105;

  // Specify whether the jobs depending on current job to run or not when if evaluated to false.
  OnIgnored on_ignored = 106;

  // Enable rollback or not
  bool enable_pipeline_rollback = 120;

  // atom notification configs
  repeated Notification notifications = 200;

  // Specify feature env the job runs on，for atom ppe test only
  string run_env = 350;

  // Support pipeline-job schedule set can not actionable operations
  repeated JobRunOperation disable_operations = 351;
}
```

## 编排结构概述

> Pipeline/Stage/Job 如何组织编排

### 总体编排模型

> Stage 串行 + 同一个 Stage 内的 Job 并行/依赖

流水线（Pipeline）的编排有三个层级，分为流水线、阶段（Stage）和原子（Job）。流水线一般由多个阶段顺序执行，每个阶段包含若干可并行或串行的原子。
流水线是组织一次 CI/CD 流程的顶层容器，通过 Pipeline.stages 字段定义多个执行阶段。

- `Pipeline.stages` 定义阶段列表；默认执行顺序为“stage 串行”。
	
- `Stage.jobs` 定义阶段内的 job 列表；默认执行顺序为“job 并行”。当 job 设置 `depends_on` 时，会把阶段内的并行关系收敛为一个 DAG（有向无环图），用于约束启动顺序。同一个阶段中 job 的执行顺序还可能会受到 Job.If 以及 Job.If_skip 的影响，这个参照下文。
	

### 条件执行与跳过

> if / extra_if/ if_skip

- `Stage.if`、`Job.if`：表达式条件满足才执行（阶段/任务级）。表达式语法由 `Pipeline.expr_syntax` 指定，CI 流水线使用 JET 语法，自由流水线使用类 Python 语法。
	
- `Job.extra_if`：与 `if` 语义相同，但用于 UI 模式下存储“用户不可见”的条件（如选择器默认条件）；当 `if` 与 `extra_if` 同时设置时，两者是 AND 关系。
	
- `Job.if_skip`：用于“跳过执行”的显式开关，与 `if` 互斥（两者不应同时生效）。
	

### Job 的执行

> Uses / steps

- `Job.uses` + `Job.inputs`：自由流水线的 Job 通常是服务型原子，引用一个“原子/动作”（atom/action）作为 job 的执行体；`inputs` 以 `google.protobuf.Struct` 承载动态入参（适合不同原子拥有不同的参数 schema），入参的格式由各个原子自定义。
	
- `Job.steps`：job 内的多步执行序列，与 `uses` 互斥。用于 CI 流水线场景。
	

## 核心字段定义

### 流水线（Pipeline）

- `schema_version`：DSL schema 版本（例如 v1/v2）。类型：`string`。
	
- `id`：流水线标识。类型：`uint64`。说明：`Pipeline.id` 与平台侧 `pipelineId` 相同，是一条流水线的 uniq_id；与 `pipelineVersion` 不同（同一条流水线每次编辑 `pipelineVersion` 递增）。
	
- `name`：流水线名称，多语言字符串。类型：`I18nString`；示例：`{"value":"发布流水线","lang":"zh","texts":{"zh":"发布流水线","en":"Release Pipeline"}}`。
	
- `desc`：流水线描述，多语言字符串。类型：`I18nString`；示例：`{"value":"用于生产发布","lang":"zh","texts":{"zh":"用于生产发布","en":"For production release"}}`。
	
- `stages`：阶段列表。默认 stage 串行、stage 内 job 并行；可用 `depends_on` 描述 job DAG。类型：`Stage[]`；示例：`[{"id":"build","jobs":[{"id":"compile"}]}]`。
	

### 阶段（Stage）

- `id`：(必填) 阶段标识，要求在同一条 pipeline 内唯一。通常格式为 'stage_' + 数字，或者 4 位随机的 16 进制字符，比如 'stage_1' 或者 'ecc2'。类型：`string`。
	
- `name`：(必填) 阶段名称，多语言字符串。类型：`I18nString`；示例：`{"value":"构建","lang":"zh","texts":{"zh":"构建","en":"Build"}}`。
	
- `if`：(必填) 阶段级条件表达式。用户没有配置入口，如果不需要特定条件，请使用 `""` 作为默认值进行填充。类型：`string`；示例：`"{{ eq .vars.custom.ENV "prod" }}"` 或 `true`。
	
- `jobs`：(必填) 阶段内 job 列表，列表不能为空（minItems: 1）。类型：`Job[]`；示例：`[{"id":"compile"},{"id":"test","depends_on":["compile"]}]`。
	

### 原子（Job）

- `id`：(必填) job 标识，要求在同一条 pipeline 内唯一。通常格式为 'jobname_' + 数字，或者 4 位随机的 16 进制字符，比如 'scm_compile_1' 或者 'ecc2'。在 YAML/DSL 场景通常作为 map key/引用 key（用于 `depends_on`、运行态 JobRun 的 jobId 等）。类型：`string`。
	
- `name`：(必填) job 名称，多语言字符串。类型：`I18nString`；示例：`{"value":"构建","lang":"zh","texts":{"zh":"构建","en":"Build"}}`。
	
- `if`：(必填) job 级条件表达式（语法由 `Pipeline.expr_syntax` 决定）。如果不需要特定条件，请使用 `""` 作为默认值进行填充。类型：`string`；示例：`"{{ .vars.custom.RUN_TESTS }}"`。
	
- `depends_on`：(必填) 当前 job 依赖的 job id 列表。类型：`string[]`；示例：`["compile","lint"]`。说明：仅允许同一 `Stage` 内的 job 之间建立依赖，不支持跨 stage 依赖。
	
- `uses`：(必填) 引用原子/动作作为 job 执行体（与 `steps` 互斥）。类型：`string` (minLength: 1)；示例：`"job_atom/scm_compile"`。
	
- `extra_if`：(可选) UI 模式下的“隐藏条件”，与 `if` 是 AND 关系。类型：`string`；示例：`"{{ ne .vars.custom.CHANNEL "canary" }}"`。
	
- `if_skip`：(可选) 跳过执行标记；与 `if` 互斥。类型：`string`；示例：`"true"`。
	
- `manual`：(可选) 手动 job（人工确认/人工触发 gate）。类型：`boolean`。
	
- `inputs`：(可选) 原子/动作入参（动态结构），具体格式需要参照原子的使用文档来填充。类型：`object`；示例：`{"revision":"main","enable_cache":true,"timeout_min":30}`。
	
- `runs_on`：runner 规格/执行环境选择。类型：`RunnerSpec`；示例：`{"engine":"RUNNER_ENGINE_KUBERNETES","bytenv":"online","resource_combo":"m1.large","labels":["biz:demo"],"image":"debian_buster:1.0.0","working_directory":"/home/code"}`。
	
- `env`：CI Job 独有的属性，job 级环境变量。类型：`map<string,string>`；示例：`{"CI":"true","GOMAXPROCS":"4"}`。
	
- `steps`：CI Job 独有的属性，脚本步骤列表（与 `uses` 互斥）。类型：`Step[]`；示例：`[{"id":"clone","uses":"step_atom/clone_code"},{"id":"build","commands":["make build"]}]`。
	
- `services`：CI Job 独有的属性，sidecar/container service 声明（常见于 Kubernetes）。类型：`ContainerService[]`；示例：`[{"id":"redis","image":"redis:7","env":{"ALLOW_EMPTY_PASSWORD":"yes"},"commands":["redis-server"]}]`。
	
- `allow_push`：CI Job 独有的属性，是否赋予 job 执行过程中的 git push 权限。类型：`bool`。
	
- `support_timeout`：(可选) 是否开启超时后的额外处理能力。类型：`boolean`。
	
- `timeout`：(可选) 超时时间（单位秒）。类型：`integer`；范围：0 - 7776000。
	
- `retry`：(可选) 失败重试策略。类型：`object`；示例：`{"max":2,"interval":60}`。
    - `max`: (enum: 1, 2, 3, 4, 5)
    - `interval`: (string | number)
	
- `on_failed`：(可选) 失败处理策略。类型：`enum`；可选值：`"ON_FAILED_UNSPECIFIED"`, `"fail"`, `"retry"`, `"continue"`, `"ignore"`。
	
- `on_timeout`：(可选) 超时处理策略。类型：`enum`；可选值：`"ON_TIMEOUT_UNSPECIFIED"`, `"fail"`, `"skip"`, `"cancel"`。
	
- `on_ignored`：(可选) 当 job 被 ignored（例如 `if` 结果为 false）时，下游依赖 job 是否继续运行。类型：`enum`；可选值：`"ON_IGNORED_UNSPECIFIED"`, `"abort"`, `"continue"`。
	
- `enable_pipeline_rollback`：(可选) 是否允许回滚。类型：`boolean`。
	
- `can_operations`：(可选) 允许的操作列表。类型：`array | object | null`。
	
- `disable_operations`：(可选) 限制 JobRun 层面的可操作项。类型：`array | object | null`。
	
- `auto_go_module_proxy`：(可选) 是否自动开启 Go Module Proxy。类型：`boolean`。
	
- `notifications`：(可选) job 级通知。类型：`array`。
    - `type`: `enum` ("webhook", "lark")
    - `name`: `string` (maxLength: 99)
    - `when`: `object` (status: array of enums, timeout: integer)
    - `lark`: `object` (users, notifier_types, notification_method, groups, group_notification_method, cards, repeat)
    - `webhook`: `object` (action_type, http_action, pipelineAction)
	
- `run_env`：(可选) 指定运行环境。类型：`string`。
	
- `job_run_operations_used`：(可选) 指示以 `job_run_operations` 或 `disable_operations` 为准。类型：`string`。
	
- `job_run_operations`：(可选) 带权限信息的 JobRun 可操作项列表。类型：`array | object | null`。