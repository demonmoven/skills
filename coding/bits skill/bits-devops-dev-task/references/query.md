# 功能一：查询开发任务

> **入口判定**
> - **已知 task_id**（或详情页 URL）：用户说"查看/查询开发任务"、"dev task 状态/进度"、"开发任务流水线"、"process info"、"任务的自测阶段"等 → 走下方 [典型分析流程](#典型分析流程)。
> - **仅有空间、无 task_id**：用户说"搜索开发任务"、"列出某空间下的开发任务"、"空间里有哪些开发任务"、"按标题找开发任务"等 → 走 [按空间搜索开发任务列表](#按空间搜索开发任务列表)。
> - **要执行/触发流水线但尚不能唯一定位任务**：用户说"运行/重跑流水线"、"跑 xxx 项目流水线"等，且**未提供** task_id、详情页或流程页 URL、当前对话与上下文也**解析不出** `dev_basic_id` → **先**走 [按空间搜索开发任务列表](#按空间搜索开发任务列表)（必要时配合 `references/space.md` 解析空间），列出候选并由用户确认唯一任务后，再读取 `references/run-pipeline.md` 执行 prepare/submit。

## 目录

- [支持的命令](#支持的命令)
- [按空间搜索开发任务列表](#按空间搜索开发任务列表)
- [典型分析流程](#典型分析流程) — Step 1 基础信息 + Step 2 进度
- [查询展示规则](#查询展示规则)

## 支持的命令

| 命令 | 说明 | OpenAPI 路径 |
|------|------|-------------|
| `dev-task get <task_id>` | 查询基础信息 | `GET /openapi/onesite/v1/dev/detail/info?dev_basic_id=<task_id>` |
| `dev-task template get --id <template_id>` | 查询模版详情 | `GET /openapi/onesite/v1/space/dev_template/get?id=<template_id>` |
| `dev-task process info --task-id <task_id>` | 查询 Task 列表及流水线信息（合并输出） | 内部依次调用 `dev/process/info` + `pipeline/run/batch` |
| `dev-task list` | 按空间等条件搜索开发任务列表（ListDevTask） | `POST /api/v1/dev/task/list` |

## 按空间搜索开发任务列表

当用户**没有提供开发任务 ID**，但提供了**空间**（名称或已知的 `space_id`），需要**枚举或检索**该空间下的开发任务时使用。

### 前置：解析 `space_id`

- 若用户已给出数字 **空间 ID**，直接使用。
- 若仅给出空间名称/关键词，先读取 `references/space.md`，用 `bitscli devops space search --keyword <关键词>` 获取目标空间的 `id`；若匹配多条，请用户确认后再继续。

### 命令

```bash
bitscli devops dev-task list \
  --space-id <space_id> \
  [--title <标题关键词>] \
  [--state <状态>]
```

| 选项 | 必填 | 说明 |
|------|------|------|
| `--space-id` | 是 | 空间 ID。 |
| `--title` | 否 | 标题模糊匹配；不传则不过滤标题。 |
| `--state` | 否 | 开发任务状态，对应列表条件里的 `state` + `in`。**不传时 CLI 侧默认按 `opened` 组装请求**。取值见下表。 |

**`--state` 取值**（与平台开发任务状态一致）：

| 取值 | 含义 |
|------|------|
| `initial` | 初始 |
| `opened` | 进行中（CLI 未传 `--state` 时的默认） |
| `closed` | 已关闭 |
| `finished` | 已完成 |

### 行为说明（Agent 须知）

- CLI 会先请求当前登录用户（`GET /openapi/onesite/v1/user/me`），将 **username** 写入请求体中的 **aboutUsers**（与 Web 端列表行为一致）；**无需**、也**不要**让用户传该参数。
- 若无法解析当前用户，CLI 会返回明确错误，可引导用户检查鉴权或先执行 `bitscli devops auth whoami` 排查。

### 成功响应与展示

- 业务成功：`code === 200` 且 `message` 为成功语义；开发任务在 **`data.tasks`** 数组中（可能为空数组）。
- **不得向用户直接粘贴整段 JSON**。将每条任务整理为表格或清单，至少包含：
  - **开发任务 ID**：`id`
  - **标题**：`title`
  - **状态**：`status`（英文枚举需转为可读中文，如 opened→进行中，与上表一致）
  - **创建人**：`creator`
  - **创建时间**：`createdAt` 若为 Unix 秒时间戳，需转为可读时间
- 若用户需要打开详情，可按 SKILL.md「公共：URL 解析」拼装：`https://bits.bytedance.net/devops/{space_id}/develop/detail/{task_id}`（`space_id` 可用本次请求的 `--space-id` 或条目中的 `spaceId`）。

### 失败与空列表

- `code !== 200`：向用户说明接口返回的 `message`（或 CLI 打印的错误文案），勿堆砌字段名。
- `data.tasks` 为空：明确告知当前条件下无匹配任务，可建议放宽标题关键词或调整状态筛选。

## 典型分析流程

当用户说"帮我看一下开发任务 xxx 的状态"、"查询开发任务信息"、"这个 Dev Task 现在进度怎么样"等**且已能定位 task_id** 时，**AI Agent 按以下 2 步获取数据，然后按"查询展示规则"向用户展示结果**：

```bash
# Step 1：获取开发任务基础信息
bitscli devops dev-task get <task_id>

# Step 2：获取开发任务进度（CLI 内部自动合并 Task 流水线概览 + 流水线运行详情）
bitscli devops dev-task process info --task-id <task_id>
```

## Step 1：开发任务基础信息

```bash
bitscli devops dev-task get <task_id>
```

返回 JSON（`{ code, message, data }`），`code=200` 且 `message=success` 表示成功。`data.basic_info.created_at`、`data.basic_info.completed_at` 已由 CLI 自动将毫秒时间戳转为可读字符串（格式 `2006-01-02 15:04:05`），Agent 无需再做转换。

**向用户展示以下字段**：

- **开发任务 ID**：来自 `data.basic_info.id`
- **名称**：来自 `data.basic_info.name` / `data.basic_info.title`
- **状态**：来自 `data.basic_info.status`
- **创建人**：来自 `data.basic_info.creator`
- **成员**：来自 `data.basic_info.members`（只需展示成员列表及角色，如 RD / QA）
- **创建时间**：来自 `data.basic_info.created_at`（直接展示即可）
- **服务信息**：从 `data.changes` 中抽取，优先按照「每个项目一行」的形式整理为「psm + 项目类型 + 开发分支」：
  - 遍历 `data.changes[]`，取 `project_info.psm` 作为服务标识，`project_info.type` 作为项目类型
  - 在对应 `build_configs[].scm_dependencies[]` 中找到 `is_main=true` 的依赖，使用其 `revision`（或分支信息）作为开发分支
- **需求信息**：从 `data.work_items` 中抽取，遍历 `work_items[]`，使用 `name` 作为标题、`url` 作为链接，整理为「需求标题 + 链接」清单（多条可列清单）

## Step 2：开发任务进度（Task 列表 + 流水线运行详情）

```bash
bitscli devops dev-task process info --task-id <task_id>
```

CLI 内部自动完成：
1. 请求 `dev/process/info` 获取 Task 列表及流水线概览
2. 遍历所有 `tasks[].pipelines[]`，提取所有非零 `build_id`，请求 `pipeline/run/batch` 获取运行详情
3. 将运行详情以 `pipeline_run_detail` 字段合并到对应的 `pipelines[]` 中

### 返回结构

**`data` 顶层字段**：

| 字段 | 类型 | 说明 |
|------|------|------|
| `dev_basic_id` | int64 | 开发任务 ID |
| `tasks` | array | 开发任务下 Task 列表 |

**`tasks[]` 字段**：

| 字段 | 类型 | 说明 |
|------|------|------|
| `task_id` | int64 | Task ID |
| `task_name` | string | Task 名称（如 `DevDevelopStageSelfTestTask`） |
| `stage_name` | string | Task 阶段名称（如 `DevDevelopStage`） |
| `status` | string | Task 状态（如 `succeeded`、`running`、`failed`） |
| `task_type` | string | Task 类型 |
| `pipelines` | array | Task 下流水线信息列表（已合并运行详情） |

**Task 选择规则**：从 `data.tasks` 中选出 **一个目标 Task** 进行展示：
- **优先选择进行中的 Task**（`status` 非终态，如非 `succeeded` / `failed` / `canceled`）
- **兜底规则**：若没有进行中的 Task，取 `data.tasks` 数组的 **最后一条**

**`pipelines[]` 字段**：

| 字段 | 类型 | 说明 |
|------|------|------|
| `pipeline_id` | int64 | 流水线 ID |
| `build_id` | int64 | 流水线构建 ID |
| `control_plane` | string | 控制面（如 `CONTROL_PLANE_CN`、`CONTROL_PLANE_I18N`） |
| `status` | string | 流水线状态（如 `SUCCESS`、`FAILED`、`RUNNING`） |
| `main_pipeline` | bool | 是否为主流水线 |
| `project_name` | string | 项目名称（项目流水线时有值） |
| `psm` | string | 流水线 PSM |
| `pipeline_run_detail` | object | CLI 合并的运行详情 |

### `pipeline_run_detail` 字段

当 `build_id` 非零且 `pipeline/run/batch` 返回成功时，CLI 会将运行详情挂载到 `pipeline_run_detail`。

**`pipeline_run_detail`** 关键字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| `run_id` | uint64 | 运行 ID（= `build_id`） |
| `pipeline_run.run_status` | int | 流水线运行状态（`PipelineRunStatus` 枚举，见 SKILL.md 公共部分） |
| `pipeline_run.created_by` | string | 触发人 |
| `pipeline_run.created_at` | string | 开始时间（ISO 格式字符串） |
| `pipeline_run.trigger_type` | string | 触发类型 |
| `pipeline_run.fail_reason` | string | 失败原因（仅失败时有值） |
| `pipeline_run.job_runs` | array | 原子（Job）运行列表 |

**`job_runs[]`** 关键字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| `job_id` | string | Job ID |
| `job_run_id` | uint64 | Job 运行 ID |
| `job_name` | string | 原子名称 |
| `job_status` | int | Job 运行状态（`JobRunStatus` 枚举，见 SKILL.md 公共部分） |
| `fail_reason` | string | 失败原因（仅失败时有值） |

## 查询展示规则

AI Agent 最终向用户输出 **两部分**：

### 第一部分：开发任务基础信息

内容同 Step 1 的展示字段。

### 第二部分：开发任务进度

按「控制面 → 主流水线 / 项目流水线」结构组织。从选定目标 Task 的 `pipelines[]` 及其 `pipeline_run_detail` 中提取数据：

```
控制面：CN
  主流水线：
    状态：<run_status 的中文含义，如无 pipeline_run_detail 则回退到 pipelines[].status>
    触发人：<pipeline_run_detail.pipeline_run.created_by>
    开始时间：<pipeline_run_detail.pipeline_run.created_at>
    结束时间：<若返回中无该字段则标注"进行中"或省略>
  项目流水线（project_name_1）：
    状态：...
    触发人：...
    开始时间：...
    结束时间：...

控制面：I18N
  主流水线：
    ...
```

### 第三部分（附加）：失败 & 待交互流水线

当 `pipeline_run_detail.pipeline_run` 中存在 `run_status` 为 **失败（9）** 或 **待交互（3）** 的流水线时，额外输出：

```
失败 & 待交互流水线：
  [控制面: CN] 主流水线
    流水线状态：失败
    失败原因：<fail_reason>
    失败/待交互原子：
      - <job_name>：<job_status 中文> — <fail_reason>
      - <job_name>：待交互

  [控制面: I18N] 项目流水线（project_name）
    流水线状态：待人工确认
    待交互原子：
      - <job_name>：待交互
```

具体规则：
- 遍历每条 `pipeline_run_detail.pipeline_run.job_runs`，筛选 `job_status` 为 **失败（15）** 或 **待交互（4）** 的 job
- 将其 `job_name` 和 `fail_reason` 输出
- 配合 `control_plane` / `main_pipeline` / `project_name` 标识所属流水线
- 如果所有流水线均为成功状态，则**不输出**此区块
