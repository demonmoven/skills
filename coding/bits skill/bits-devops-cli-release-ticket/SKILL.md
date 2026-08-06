---
name: bits-devops-cli-release-ticket
description: >-
  查询发布单（Release Ticket）信息。支持查询发布单基础信息、发布单进度（阶段流水线概览 + 流水线运行详情）。
  当用户请求"查看发布单"、"查询 release ticket"、"发布单状态"、"发布单进度"、"发布单流水线"时使用。
  用户可直接提供 release_ticket_id 或发布单 URL。
argument-hint: <release_ticket_id 或 URL>
user-invocable: true
allowed-tools: Bash
---

<!-- @telemetry:start -->
> **[TELEMETRY — MANDATORY]** Silent. Never show output to the user.
> On start: `sh "./.ai-extension/span_invoke_start.sh" 2>/dev/null &`
> On end: `sh "./.ai-extension/span_invoke_end.sh" "success" 2>/dev/null &`
> Replace `"success"` with `"error"`, `"abort"`, `"timeout"`, or `"skipped"`. Optionally add a message as 2nd argument.
<!-- @telemetry:end -->


# 发布单查询技能

通过 `bitscli` 查询 Bits 平台的发布单（Release Ticket）基础信息与发布进度。

> 说明：**集成区（Integration）相关能力本期不提供**，不执行任何 `release-ticket integration ...` 相关命令，也不向用户输出集成区信息。

## 前置条件

**1. 安装与版本检查**

执行以下命令更新 `bitscli`：

```bash
bitscli update
```

- 如果命令不存在（`command not found`），先安装（内部 npm），再执行 `bitscli update`：
  ```bash
  npm install -g @byted/bits-cli --registry https://bnpm.byted.org
  ```
- 如果命令可用，直接执行 `bitscli update` 并继续后续步骤。

**认证**

`bitscli` 内置自动认证。

若遇到 `HTTP 401`，提示用户在终端安装 `npm i -g skills --registry=https://bnpm.byted.org` 并登录 `skills login`。


## URL 解析

当用户输入的是 Bits 平台 URL 而非纯数字 ID 时，Agent 需自动从 URL 中提取 `release_ticket_id`。

**发布单 URL 格式**：

```
https://bits.bytedance.net/devops/{space_id}/release/releaseTicket/detail/{release_ticket_id}?...
```

**提取规则**：取 URL 路径中 `detail/` 后紧跟的数字部分作为 `release_ticket_id`，忽略 `?` 之后的所有查询参数。

**示例**：
- 输入：`https://bits.bytedance.net/devops/112700626946/release/releaseTicket/detail/1125147002882?devops_space_type=server_fe&releaseTab=flow`
- 提取：`release_ticket_id = 1125147002882`

## 支持的命令

| 命令 | 说明 | OpenAPI 路径 |
|------|------|-------------|
| `release-ticket get <release_ticket_id>` | 查询发布单基础信息 | `GET /openapi/onesite/v1/release/basic/info?id=<release_ticket_id>` |
| `release-ticket process info --ticket-id <release_ticket_id>` | 查询发布单进度（阶段流水线概览 + 流水线运行详情，合并输出） | 内部依次调用 `stage/running/pipeline` + `pipeline/run/batch` |

---

## 典型分析流程

当用户说"帮我看一下发布单 xxx 的状态"、"查询发布单信息"、"发布单进度"等泛化描述时，**AI Agent 按以下 2 步获取数据，然后按"用户展示规则"向用户展示结果**：

```bash
# Step 1：获取发布单基础信息
bitscli release-ticket get <release_ticket_id>

# Step 2：获取发布单进度（CLI 内部自动合并阶段流水线概览 + 流水线运行详情）
bitscli release-ticket process info --ticket-id <release_ticket_id>
```

---

## Step 1：发布单基础信息

```bash
bitscli release-ticket get <release_ticket_id>
```

返回 JSON，业务数据在 `data` 对象中。其中 `created_at`、`completed_at` 已由 CLI 自动将毫秒时间戳转为可读字符串（格式 `2006-01-02 15:04:05`），Agent 无需再做转换。

**向用户展示以下字段（第一部分）**：

- **发布单 ID**：来自 `data.release_ticket_id`
- **名称**：来自 `data.name`
- **状态**：来自 `data.status`（可转为中文说明）
- **创建人**：来自 `data.creator`
- **成员**：来自 `data.release_approvers`、`data.test_approvers` 等人员字段
- **创建时间**：来自 `data.created_at`（直接展示即可）
- **服务信息**：从 `data.change_items` 中抽取，整理为「psm + 项目类型」
- **需求信息**：从 `data.work_items` 中抽取，整理为「需求标题 + 链接」（多条可列清单）

---

## Step 2：发布单进度（阶段流水线概览 + 运行详情）

```bash
bitscli release-ticket process info --ticket-id <release_ticket_id>
```

CLI 内部自动完成以下操作：
1. 请求 `stage/running/pipeline` 获取阶段流水线概览
2. 提取所有非零 `build_id`，请求 `pipeline/run/batch` 获取运行详情
3. 将运行详情以 `pipeline_run_detail` 字段合并到对应的 `pipeline_overviews[]` 中

返回 JSON（`{ code, message, data }`）。

### 返回结构

**`data` 顶层字段**：

| 字段 | 类型 | 说明 |
|------|------|------|
| `stage_id` | uint64 | 实际使用的阶段 ID |
| `stage_status` | int | 阶段状态（`StageStatus` 枚举） |
| `pipeline_overviews` | array | 阶段下每条流水线的概览（已合并运行详情） |

**`StageStatus` 枚举值**（`stage_status` 字段）：

| 值 | 含义 | 向用户展示 |
|----|------|-----------|
| 1 | PENDING | 待开始 |
| 2 | NOTREADY | 未就绪 |
| 3 | RUNNING | 运行中 |
| 4 | SUCCEEDED | 成功 |
| 5 | FAILED | 失败 |
| 6 | CANCELED | 已取消 |
| 7 | FORCE_COMPLETE | 强制完成 |

**`pipeline_overviews[]` 字段**（对应 `PipelineOverview`，已合并运行详情）：

| 字段 | 类型 | 说明 |
|------|------|------|
| `build_id` | uint64 | 流水线构建 ID |
| `control_plane` | int | 控制面（`ControlPlane` 枚举） |
| `pipeline_type` | int | 流水线类型（`PipelineType` 枚举） |
| `main_pipeline` | object | 当 `pipeline_type=1` 时生效，包含 `pipeline_id` |
| `project_pipeline` | object | 当 `pipeline_type=2` 时生效，包含 `pipeline_id`、`project_name`、`project_type`、`project_unique_id` |
| `pipeline_run_status` | int | 流水线运行状态（`StagePipelineRunStatus` 枚举） |
| `status` | int | 已废弃旧状态，**不向用户展示** |
| `pipeline_run_detail` | object | **CLI 合并的运行详情**（见下方） |

**`ControlPlane` 枚举值**（`control_plane` 字段）：

| 值 | 含义 | 向用户展示 |
|----|------|-----------|
| 1 | CONTROL_PLANE_CN | CN |
| 2 | CONTROL_PLANE_I18N | I18N |
| 4 | CONTROL_PLANE_EU_TTP | EU_TTP |
| 5 | CONTROL_PLANE_US_TTP | US_TTP |
| 6 | CONTROL_PLANE_I18N_BD | I18N_BD |

**`PipelineType` 枚举值**（`pipeline_type` 字段）：

| 值 | 含义 |
|----|------|
| 1 | 主流水线 |
| 2 | 项目流水线 |

**`StagePipelineRunStatus` 枚举值**（`pipeline_run_status` 字段）：

| 值 | 含义 | 向用户展示 |
|----|------|-----------|
| 1 | BLOCKING | 阻塞中 |
| 2 | RUNNING | 运行中 |
| 3 | WAITING | 待人工确认 |
| 4 | ROLLBACKING | 回滚中 |
| 5 | ROLLBACKED | 已回滚 |
| 6 | CANCELLING | 取消中 |
| 7 | CANCELLED | 已取消 |
| 8 | SUCCEEDED | 成功 |
| 9 | FAILED | 失败 |
| 10 | QUEUING | 排队中 |

### `pipeline_run_detail` 字段

当 `build_id` 非零且 pipeline/run/batch 返回成功时，CLI 会将运行详情挂载到 `pipeline_run_detail`。

**`pipeline_run_detail`** 关键字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| `run_id` | uint64 | 运行 ID（= `build_id`） |
| `pipeline_run.run_status` | int | 流水线运行状态（`PipelineRunStatus` 枚举，值与 `StagePipelineRunStatus` 一致） |
| `pipeline_run.created_by` | string | **触发人** |
| `pipeline_run.created_at` | string | **开始时间**（ISO 格式字符串） |
| `pipeline_run.trigger_type` | string | 触发类型 |
| `pipeline_run.fail_reason` | string | 失败原因（仅失败时有值） |
| `pipeline_run.job_runs` | array | 原子（Job）运行列表 |

**`job_runs[]`** 关键字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| `job_id` | string | Job ID |
| `job_run_id` | uint64 | Job 运行 ID |
| `job_name` | string | 原子名称 |
| `job_status` | int | Job 运行状态（`JobRunStatus` 枚举） |
| `fail_reason` | string | 失败原因（仅失败时有值） |

**`JobRunStatus` 枚举值**（`job_status` 字段，仅列关键状态）：

| 值 | 含义 | 向用户展示 |
|----|------|-----------|
| 3 | RUNNING | 运行中 |
| 4 | WAITING | 待交互 |
| 14 | SUCCEEDED | 成功 |
| 15 | FAILED | 失败 |
| 12 | CANCELED | 已取消 |

---

## 用户展示规则

AI Agent 最终向用户输出 **两部分**：

### 第一部分：发布单基础信息

内容同 Step 1 的展示字段（发布单 ID / 名称 / 状态 / 创建人 / 成员 / 创建时间 / 服务信息 / 需求信息）。

### 第二部分：发布单进度

按「控制面 → 主流水线 / 项目流水线」结构组织。从每条 `pipeline_overviews[]` 及其 `pipeline_run_detail` 中提取数据：

```
阶段状态：<stage_status 的中文含义>

控制面：CN
  主流水线：
    状态：<pipeline_run_status 的中文含义>
    触发人：<pipeline_run_detail.pipeline_run.created_by>
    开始时间：<pipeline_run_detail.pipeline_run.created_at>
    结束时间：<若返回中无该字段则标注"进行中"或省略>
  项目流水线（project_name_1）：
    状态：...
    触发人：...
    开始时间：...
    结束时间：...
  项目流水线（project_name_2）：
    ...

控制面：I18N
  主流水线：
    ...
  项目流水线（...）：
    ...
```

### 第三部分（附加）：失败 & 待交互流水线

当 `pipeline_run_detail.pipeline_run` 中存在 `run_status` 为 **失败（9）** 或 **待交互（3）** 的流水线时，额外输出一个「失败 & 待交互流水线」区块：

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
- 配合 `control_plane` / `pipeline_type` / `project_pipeline.project_name` 标识所属流水线
- 如果所有流水线均为成功状态，则**不输出**此区块

---

## 输出纪律

- **不得在用户侧输出任何内部处理说明**，例如"根据技能约定的…"、"按照 SKILL 规则…"等描述性前缀文字。直接呈现最终格式化结果即可。
- **不得直接输出完整的原始 JSON**。所有数据都需要按上述规则整理后再向用户呈现。
- **枚举值必须转为中文/可读文案**再向用户展示。`control_plane` 转为 "CN" / "I18N" / "US_TTP" 等；`pipeline_run_status` / `stage_status` / `job_status` 转为中文含义。
- **时间戳**若为 Unix 毫秒，需要 ÷ 1000 后转为可读时间；若已为 ISO 字符串则直接展示。

---

## 错误处理

| 错误现象 | 可能原因 | 处理方式 |
|----------|----------|----------|
| `HTTP 401` / `Unauthorized` | 认证失效 | 提示用户在终端执行 `skills login` 重新登录 |

如需在本地命令行中使用 Bits 服务 Token，可按以下方式配置（可选）：

```bash
export BITS_TOKEN="your_token_here"
# 可选：指定 API 域名（默认 https://bits.bytedance.net）
export BITS_BASE_URL="https://bits.bytedance.net"
# 可选：服务 Token 下指定执行身份
export BITS_USER="your.username"
```
| `HTTP 403` / `Forbidden` | Token 无权限访问该发布单 | 确认 Token 具有对应项目权限 |
| `HTTP 404` / 返回空数据 | release_ticket_id 不存在 | 检查 ID 是否正确 |
| `code != 200` | 业务层错误 | 查看返回的 `message` 字段 |
| `dial tcp: ... connection refused` | `BITS_BASE_URL` 配置错误或网络不通 | 检查网络和 `BITS_BASE_URL` 配置 |

---

## 示例

```bash
# Step 1：查询发布单基础信息（本地调试示例，Agent 容器中可省略 BITS_TOKEN）
BITS_TOKEN=xxx bitscli release-ticket get 1119978686466

# Step 2：查询发布单进度（已自动合并流水线运行详情）
BITS_TOKEN=xxx bitscli release-ticket process info --ticket-id 1119978686466

# 开启调试模式
BITS_TOKEN=xxx BITS_DEBUG=true bitscli release-ticket get 1119978686466
```
