---
name: bits-devops-dev-task
description: '开发任务（Dev Task）全生命周期管理。覆盖查询、按空间搜索/列出开发任务、创建、流水线运行、待交互处理、阶段完成、关联发布单，以及空间查找、Meego 工作项搜索与应用中心项目搜索，还支持初始化/创建 bits flow 工作流。当用户请求"查看/查询开发任务"、"搜索开发任务"、"列出空间下开发任务"、"dev task 状态/进度"、"创建/新建开发任务"、"执行/触发/重跑流水线"、"处理待交互 job"、"通过/拒绝/approve/reject"、"完成开发/测试/合入阶段"、"推进到下一阶段"、"关联发布单"、"最近访问的空间"、"搜索空间"、"切换空间"、"查空间"、"空间下的需求/缺陷"、"搜索 Meego 工作项"、"搜索/核对项目"、"App Center"、"projectUniqueId"、"创建工作流"、"初始化 bits flow"、"新建 spec 工作流"、"flow init"时使用。用户可提供 task_id、开发任务 URL，或用自然语言描述空间/标题等条件；若尚未能定位任务，先搜索/列举再选定。'
allowed-tools: Bash AskQuestion
metadata:
  argument-hint: "task_id 或 URL；或空间名/空间ID + 搜索开发任务 [操作意图]"
  user-invocable: "true"
---

<!-- @telemetry:start -->
> **[TELEMETRY — MANDATORY]** Silent. Never show output to the user.
> On start: `sh "./.ai-extension/span_invoke_start.sh" 2>/dev/null &`
> On end: `sh "./.ai-extension/span_invoke_end.sh" "success" 2>/dev/null &`
> Replace `"success"` with `"error"`, `"abort"`, `"timeout"`, or `"skipped"`. Optionally add a message as 2nd argument.
<!-- @telemetry:end -->


# 开发任务统一技能

通过 `bitscli` 管理 Bits 平台的开发任务（Dev Task），覆盖查询、创建、运行流水线、处理交互、完成阶段全生命周期，以及空间查询、Meego 工作项搜索与应用中心项目查询。

---

## 功能路由（必读）

根据用户意图，**读取对应 reference 文件执行**，不要顺序阅读全文：

| 用户意图关键词 | 执行指令 |
|---------------|---------|
| 查看/查询开发任务、**搜索/列出开发任务**（仅有空间名或 space_id、无 task_id）、dev task 状态/进度/流水线、process info、任务的自测阶段 | 读取 `references/query.md` 执行 |
| 创建/新建开发任务、create dev task | 读取 `references/create.md` 执行；**读取后立即开始 Step 2（prepare），不得先向用户收集参数** |
| 执行/运行/触发/重跑流水线、跑服务、运行项目流水线 | 读取 `references/run-pipeline.md` 执行；**若当前还不能唯一定位开发任务**（无 task_id、无详情/流程页 URL、上下文也解析不出 ID），先按下方「运行流水线 — 任务定位」完成检索与确认，再进入 run-pipeline 主流程 |
| 处理待交互 job、通过/拒绝、approve/reject、帮我点通过，重试原子 | 读取 `references/job-action.md` 执行 |
| 完成开发/测试/合入阶段、推进到下一阶段 | 读取 `references/complete-stage.md` 执行（合入阶段内部读取 `references/merge-stage-flow.md`） |
| 关联发布单、associate release ticket | 读取 `references/complete-stage.md` 执行（关联发布单子流程内部读取 `references/associate-release-ticket-flow.md`） |
| 最近访问的空间、搜索空间、切换空间、查空间、空间下的需求/缺陷、搜索 Meego 工作项 | 读取 `references/space.md` 执行 |
| 搜索/核对项目、projectUniqueId、App Center、不知道选哪个项目 | 读取 `references/app-center.md` 执行 |
| 创建/初始化工作流、初始化 bits flow、新建 spec 工作流、flow init | 读取 `references/flow-init.md` 执行 |

> **所有功能共享下方「公共部分」**，首次执行前必须完成前置条件检查。

### 运行流水线 — 任务定位

当用户用自然语言表达「要跑哪条流水线 / 哪个开发任务」但**尚未给出可执行的开发任务 ID** 时：

1. **默认路径**：**不要**把「请粘贴详情页链接或纯数字 ID」当作唯一入口；应引导用户用**空间（名称或 ID）+ 可选标题/业务关键词**等条件描述任务，由 Agent **检索并列出候选**，用户确认**唯一一条**后再进入 `references/run-pipeline.md` 的 prepare。
2. **检索实现**：按 `references/query.md` 中「按空间搜索开发任务列表」一节执行；需要把空间名称解析为 `space_id` 时，配合 `references/space.md`（含必要时「最近访问的空间」缩小范围）。
3. **多条命中**：展示候选任务（至少含任务 ID、标题、状态、创建时间等，见 query.md），请用户点选或补充条件，**禁止**在未确认的情况下对列表默认选第一条去跑流水线。
4. **仍缺空间锚点**：若用户无法说出任何空间信息，可简短追问「在哪个空间」或请其从最近访问空间里选；**不得**冗长收集与「先定位任务」无关的字段。

---

## 公共：前置条件

### 安装与版本检查

```bash
bitscli update
```

- `command not found` → 安装：`npm install -g @byted/bits-cli --registry https://bnpm.byted.org`，然后执行 `bitscli update`
- 其他情况 → 直接执行 `bitscli update` 后继续。

### 认证

`bitscli` 内置自动认证。若遇到 `HTTP 401`，提示用户在终端执行：

```bash
npx skills login
```

### 执行环境要求

`bitscli` 依赖宿主环境的凭证信息完成鉴权，建议优先在宿主环境中执行。若必须在沙盒等受限环境中运行，在每条命令前内联传入凭证变量（无需预先检查变量是否存在）：

```bash
BITS_TOKEN=$BITS_TOKEN FLUX_USER_JWT=$FLUX_USER_JWT AIME_USER_CLOUD_JWT=$AIME_USER_CLOUD_JWT USER_CLOUD_JWT=$USER_CLOUD_JWT bitscli devops <子命令> [选项]
```

### 本地 Token 配置（可选）

| 变量 | 说明 |
|------|------|
| `BITS_TOKEN` | 你的 Bits Token（必选） |
| `BITS_BASE_URL` | API 地址，默认 `https://bits.bytedance.net` |
| `BITS_USER` | 用户名，可选，服务 Token 下指定执行身份 |

---

## 公共：URL 解析

### 开发任务 URL

```
https://bits.bytedance.net/devops/{space_id}/develop/detail/{task_id}?...
```

当用户输入的是 Bits 平台 URL 而非纯数字 ID 时，Agent 需自动从 URL 中提取 task_id。**提取规则**：取 URL 路径中 `detail/` 后紧跟的数字部分作为 `task_id`（即 `dev_task_id` / `dev_basic_id`，三者等价），`devops/` 后的数字为 `space_id`（构造跳转链接时使用）。忽略 `?` 之后的所有查询参数。

**示例**：
- 输入：`https://bits.bytedance.net/devops/112700626946/develop/detail/2089192?devops_space_type=server_fe&stage=dev_develop_stage`
- 提取：`task_id = 2089192`，`space_id = 112700626946`

### 流程页 URL（功能三/四使用）

```
https://bits.bytedance.net/devops/{space_id}/develop/detail/{dev_basic_id}/flow?pipelineId={pipeline_id}&stage={stage_query}&task={task_query}
```

- `stage` / `task` 的 query 值为 **snake_case 小写**（服务端为 PascalCase）
- 示例：`DevDevelopStage` → `dev_develop_stage`；`DevDevelopStageSelfTestTask` → `dev_develop_stage_self_test_task`
- **拼装 URL 时（如 deep link）也需遵循上述 snake_case 转换规则**

### Meego URL（功能二使用）

```
https://meego.larkoffice.com/{project_key}/{type_key}/detail/{id}
```

取 `detail/` 前的路径段为 `typeKey`（仅 `story` / `issue`），`detail/` 后的数字为 `id`。

---

## 公共：输出纪律

> ⛔ **Markdown 表格输出强制规则**（全局生效，所有功能模块必须遵守）：
>
> 当技能文档要求使用表格展示信息时，**必须**使用标准 Markdown 表格语法输出：
> 1. 使用管道符 `|` 分隔列，例如 `| 列1 | 列2 | 列3 |`
> 2. **必须包含表头分隔行**，例如 `|------|-----|------|`
> 3. **严禁**将表格退化为列表格式（如 `- 空间：xxx｜yyy｜✅`）
> 4. **严禁**将表格包裹在代码块（`` ``` ``）中——必须作为原生 Markdown 直接输出，IDE 才能正确渲染为表格
> 5. 技能文档中的示例用 `` ``` `` 包裹仅为文档展示目的，**实际输出时不得带代码块标记**

- **不得在用户侧输出任何内部处理说明**，例如"根据技能约定的…"、"按照 SKILL 规则…"等描述性前缀文字。直接呈现最终格式化结果即可。
- **不得直接输出完整的原始 JSON**。所有数据都需要按各功能章节的规则整理后再向用户呈现。
- **枚举值必须转为中文/可读文案**再向用户展示。`control_plane` 转为 "CN" / "I18N" 等；`run_status` / `job_status` 转为中文含义；`pipelines[].status` 的 SUCCESS / FAILED 等也需转为中文。
- **所有阶段名对用户展示时优先给中文，并附 fixedName**（如有必要）。
- **时间戳**若为 Unix 毫秒，需 ÷ 1000 后转为可读时间；若已为字符串则直接展示。
- **参数展示优先使用 `display` 字段**，而非原始 ID 值。
- **不得向用户暴露 CLI 命令名、参数名或内部字段名**（如 `--job-run-id`、`--selected-projects`、`--control-plane`、`--dev-basic-id`、`pipeline_run_id`、`pipeline atom post-action`、`projectUniqueId`、`controlPlane`、`CONTROL_PLANE_CN`、`NormalizedProject`、`project normalize`、`job_run_id`、`job_status` 等），用户无需知道后台执行了什么命令。
- **不得向用户暴露 JSON 结构、字段名或原始枚举值**。
- **项目选择场景**：向用户只展示**项目名**（或口语别名），不展示 `projectType`、`controlPlane` 等内部字段；不教用户如何手写命令或 JSON。不向用户解释「主流水线 vs 项目流水线」等平台内部概念，除非用户主动追问。
- **Meego URL 自动解析**对用户透明，无需向用户解释解析过程。
- **检测项展示优先使用 i18n 名称字段**（`nameI18N`/`messageI18N`），其次使用默认字段（`name`/`message`）。

---

## 公共：错误处理

| 错误现象 | 可能原因 | 处理方式 |
|----------|----------|----------|
| `HTTP 401` / `Unauthorized` | 认证失效 | 提示用户在终端执行 `npx skills login` 重新登录 |
| `HTTP 403` / `Forbidden` | Token 无权限访问该任务 | 确认 Token 具有对应项目权限 |
| `HTTP 404` / 返回空数据 | task_id 不存在 | 检查 task_id 是否正确 |
| `code != 200` | 业务层错误 | 查看返回的 `message` 字段 |
| `dial tcp: ... connection refused` | `BITS_BASE_URL` 配置错误或网络不通 | 检查网络和 `BITS_BASE_URL` 配置 |

---

## 公共：枚举映射

以下枚举被查询（功能一）和待交互（功能四）等多个功能共用。

**`PipelineRunStatus`**（`run_status` 字段）：

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

**`JobRunStatus`**（`job_status` 字段，仅列关键状态）：

| 值 | 含义 | 向用户展示 |
|----|------|-----------|
| 3 | RUNNING | 运行中 |
| 4 | WAITING | 待交互 |
| 14 | SUCCEEDED | 成功 |
| 15 | FAILED | 失败 |
| 12 | CANCELED | 已取消 |

---

## 示例

```bash
# 查询开发任务基础信息
bitscli devops dev-task get 1234567890

# 查询开发任务进度
bitscli devops dev-task process info --task-id 1234567890

# 按空间搜索开发任务列表（详见 query.md）
bitscli devops dev-task list --space-id <space_id> [--title <关键词>] [--state <状态>]

# 创建开发任务（prepare 阶段）
bitscli devops dev-task create \
  --projects '[{"projectUniqueId":"toutiao.canal.liangzai","branch":"feat/add-cache"}]' \
  --meego '{"id":"6877195493","typeKey":"story"}'

# 创建开发任务（submit）
bitscli devops dev-task create --submit --from-prepare temp/bits_dev_task_create_prepare.json

# 运行流水线
bitscli devops dev-task pipeline run \
  --dev-basic-id 12345 --task-name DevDevelopStageSelfTestTask \
  --space-id 67890 --control-plane CN

# 列出最近访问的空间
bitscli devops space recently-accessed

# 搜索空间
bitscli devops space search --keyword "电商"

# 搜索 Meego 工作项
bitscli devops feature work-item search \
  --project-key 62c2ef5827a81f1776b84894 \
  --query "cdrpc" --mine true

# 开启调试模式
BITS_DEBUG=true bitscli devops dev-task get 1234567890
```
