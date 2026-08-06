# 功能三：运行流水线

> **入口判定**：用户说"执行流水线"、"跑 cdapi"、"运行 bits.cd.api"、"触发 xxx 流水线"、"重跑项目流水线"等。
> 用户可提供 dev_task_id、开发任务详情或流程页 URL；若只给项目口语别名，须结合上下文或追问任务。
> **尚不能唯一定位开发任务时**（无 task_id、无详情/流程页 URL、上下文也解析不出 ID）：**不要**把「请粘贴链接或纯数字 ID」当作唯一路径；应先按 `references/query.md`「按空间搜索开发任务列表」（必要时 `references/space.md`）检索并让用户确认任务，再进入下文 Step 1。详见 `SKILL.md`「运行流水线 — 任务定位」。
> **覆盖范围**：含项目流水线、RD 自测/阶段流水线。

在已能确定 `dev_basic_id` 的前提下，采用**两阶段**流程：先 prepare 校验参数，用户确认后再 submit 真正触发。

> **⛔ 强制规则 — 不可绕过的确认机制**
>
> 1. **禁止复用旧文件**：工作区中可能存在上一次运行遗留的 `bits_pipeline_run_prepare.json`，**严禁**直接读取该文件执行 submit。每次运行都必须从 Step 1 开始完整执行（若本次需要先定位任务，则 Step 0 → Step 1…）。
> 2. **禁止跳步**：必须严格按 Step 0（若需要）→ Step 1 → Step 2 → Step 3 → Step 4 顺序执行。
> 3. **用户确认是唯一的 submit 前提**：只有在 Step 3 向用户展示完整参数清单表格、且用户**在当前对话中明确回复确认**后，才能执行 Step 4 的 `--submit`。

## URL 解析

从 `https://bits.bytedance.net/devops/{space_id}/develop/detail/{dev_basic_id}/flow?pipelineId={pipeline_id}&stage={stage_query}&task={task_query}` 中取 `dev_basic_id` 等字段。

- **`stage` 和 `task` 的 query 值必须是 snake_case 小写**。服务端使用 PascalCase 存储，注意格式转换。示例：
  - `DevDevelopStage` → `dev_develop_stage`
  - `DevDevelopStageSelfTestTask` → `dev_develop_stage_self_test_task`

## 核心流程

### Step 0：尚未定位开发任务 ID 时（可选）

当 **Step 1 无法从用户输入与上下文中得到 `dev_basic_id`**（无数字 ID、无含 `detail/{id}` 的 Bits URL）时，**必须先**完成本步，**不得**直接进入 Step 2（prepare 依赖 `--dev-basic-id`）。

1. 从用户自然语言中提取检索条件：**空间**（名称或数字 ID）、**标题/业务关键词**（可选）、**状态**（可选）；条件不足时可简短追问「在哪个空间」或引导使用最近访问空间（见 `references/space.md`）。
2. 按 `references/query.md`「按空间搜索开发任务列表」执行检索，将结果按该节展示规则呈现；命中多条时由用户**明确选定一条**（或补充条件后重搜）。
3. 用户确认后，将对应条目的 **开发任务 ID** 作为后续步骤中的 `dev_basic_id` 来源，进入 Step 1。

> **与「禁止提前收集信息」的关系**：下文「禁止以请先提供控制面/空间 ID 为由停下来」针对的是 **已具备 `dev_basic_id`、准备进入 prepare** 的场景；**本步**解决的是「还没有任务 ID」——此时允许且应当通过检索/追问**先唯一定位任务**，与「提前索要控制面」不是同一类问题。

### 整体编排（循环 prepare → 展示表格 → 用户确认 → submit）

```
用户表达意图
    ↓
尚无 dev_basic_id？ ──是──→ Step 0：按 query.md 搜索列表 → 用户确认任务 ─┐
    └──否──────────────────────────────────────────────────────────────┘
    ↓
Step 1: 解析用户输入，构造已知参数（含 dev_basic_id）
    ↓
┌──→ Step 2: 调用 pipeline run（prepare 阶段）
│        ↓
│    Step 3: 解析 prepare 输出，展示【完整参数清单表格】（每次必须展示）
│        ↓
│        ├── 有缺失必填项 → 表格中标记 ❌，提示用户补充
│        │       ↓
│        │   用户补充参数 ──→ 回到 Step 2 重新 prepare ─┘
│        │
│        └── ready === true → 表格展示所有参数已就绪 ✅，请求用户确认
│                ↓
│           用户确认"是" → Step 4: 调用 pipeline run --submit
│           用户说"否"/要修改 ──→ 回到 Step 2（携带修改后的参数）─┘
```

### Step 1：解析用户输入

**前置**：已能通过用户输入、URL 或 **Step 0** 得到 `dev_basic_id`。若仍无，回到 Step 0，不得空跑 Step 2。

从用户的自然语言描述中提取可用参数：

| 参数 | 来源 |
|------|------|
| `--dev-basic-id` | 用户提供的开发任务 ID、URL 中解析，或 Step 0 选定任务 |
| `--task-name` | 用户指定的阶段 Task fixedName（通常不需要，会自动获取） |
| `--control-plane` | 用户指定的控制面（通常不需要，单控制面会自动选定） |
| `--selected-projects` | 用户指定的项目范围（默认全量） |

**重要约束**：Step 1 只提取用户**已明确提供**的参数。缺什么参数由 Step 3 的 prepare 结果告知用户。

> **⛔ 禁止提前收集信息**：在 **已具备 `dev_basic_id`** 的前提下，无论用户是否还提供了其他参数，Step 1 结束后必须**立即进入 Step 2 执行 prepare**，不得以"请先提供控制面/空间 ID"为由停下来向用户提问。（尚未定位任务时，不适用本条，应先完成 Step 0。）

### Step 2：调用 prepare（参数校验）

将已知参数拼入命令并执行。**不带 `--submit`**，此命令仅做参数校验与自动补全，返回 JSON。

```bash
# 用户仅提供了 dev_basic_id 的最常见场景
bitscli devops dev-task pipeline run --dev-basic-id <id>

# 用户还指定了控制面
bitscli devops dev-task pipeline run \
  --dev-basic-id <id> \
  --control-plane CN
```

### Step 3：解析 prepare 输出，展示完整参数清单表格

**核心原则**：**每次 prepare 调用后，都必须向用户展示完整的参数清单表格**，无论 `ready` 为 `true` 还是 `false`。

prepare 输出 JSON 结构：

```json
{
  "ready": true,
  "params": {
    "dev_basic_id":      { "value": 123,   "required": true,  "source": "user_input", "is_display": true },
    "task_name":         { "value": "DevDevelopStageSelfTestTask", "required": true, "source": "auto", "is_display": true, "display": "RD 自测" },
    "task_id":           { "value": 456,   "required": true,  "source": "auto", "is_display": false },
    "control_plane":     { "value": "CN",  "required": true,  "source": "auto", "is_display": true },
    "selected_projects": { "value": null,  "required": false, "source": "default", "is_display": true, "display": "全量" }
  },
  "missing_required": [],
  "available_control_planes": ["CN", "I18N"],
  "runnable_projects_preview": [
    { "project_type": "TCE", "project_name": "示例服务 A" },
    { "project_type": "FAAS", "project_name": "示例函数 B" }
  ]
}
```

> **说明**：`params` 中**不包含** `space_id`。prepare 内部仍会按开发任务拉取详情做校验；**submit** 时再根据 `dev_basic_id` 查询开发任务详情解析空间并触发流水线。

> **`runnable_projects_preview`（可选）**：当「项目范围」为全量（未传 `--selected-projects`）且已能确定**控制面**与 **Task** 时，CLI 会调用与 `dev-task pipeline change-items` 相同的接口，把当前可运行项目预览写入快照。字段含义：`project_type` 为项目类型简写（如 TCE、FAAS）；`project_name` 为展示名。**不向用户复述 JSON 字段名**，下表用中文列展示即可。

**展示规则**：

1. **展示文案优先级**：有 `display` 字段优先使用 `display`；否则展示 `value`
2. **仅展示 `is_display === true` 的参数**（`task_id` 等 `is_display === false` 的不展示）；**不要**把「空间 ID」列入 prepare 参数表（快照中已无该字段；空间在提交阶段由 CLI 根据开发任务解析）
3. **可运行项目预览**：若存在 `runnable_projects_preview` 且长度 ≥ 1，在参数表**之后**追加一张「本次全量运行将覆盖的项目」表（或同级说明），列：**项目类型** | **项目名称**，一行一项。若该字段缺失或为空数组，不编造、不提示「无列表」。
4. **来源标注**：
   - `user_input` → 「用户指定」
   - `auto` → 「自动获取」
   - `default` → 「默认值」
   - `missing` → 「❌ 未填写（必填）」
5. **状态列**：已填写 → `✅`；缺失必填项 → `❌ 缺失`
6. **多控制面提示**：当 `available_control_planes` 存在且长度 > 1 时，展示候选列表让用户选择

**展示格式 — 有缺失必填项时（`ready === false`）**：

```
运行流水线 — 参数校验结果

| 参数 | 值 | 来源 | 状态 |
|------|-----|------|------|
| 开发任务 ID | 123456 | 用户指定 | ✅ |
| Task | RD 自测（DevDevelopStageSelfTestTask） | 自动获取 | ✅ |
| 控制面 | — | — | ❌ 缺失 |
| 项目范围 | 全量 | 默认值 | ✅ |

⚠️ 以下必填参数仍需补充：
- **控制面**（`control_plane`）：可选值为 CN, I18N，请选择

请补充以上缺失项后，我会重新校验。
```

**展示格式 — 所有参数就绪时（`ready === true`）**：

```
运行流水线 — 参数校验通过 ✅

| 参数 | 值 | 来源 | 状态 |
|------|-----|------|------|
| 开发任务 ID | 123456 | 用户指定 | ✅ |
| Task | RD 自测（DevDevelopStageSelfTestTask） | 自动获取 | ✅ |
| 控制面 | CN | 自动获取 | ✅ |
| 项目范围 | 全量 | 默认值 | ✅ |

所有必填参数已就绪，是否确认执行？
```

### Step 3.5（可选）：用户要求只跑部分项目

当用户明确要求只跑部分项目（如「只跑 cdapi」「跑 xxx 项目」），需查询可运行项目列表：

1. 使用 `task_id` 和 `task_name` 从 prepare 结果中获取
2. 拉取可运行项目列表：

```bash
bitscli devops dev-task pipeline change-items \
  --dev-basic-id <dev_basic_id> \
  --task-id <task_id> \
  --task-name <task_name> \
  --control-panel <control_plane>
```

3. 标准化：

```bash
bitscli devops dev-task project normalize --input '<change-items 的 stdout JSON>'
```

4. 从返回的 `projects` 数组中，按用户描述过滤出目标项；若无法唯一确定，让用户多选。
5. 用户确认后，将最终数组作为 `--selected-projects` 传入下一次 prepare。

### Step 4：submit 执行

> **⛔ 前置条件**：Step 4 仅允许在以下条件**全部满足**时执行：
> - Step 2 的 prepare 已在**当前对话**中执行过
> - Step 3 的参数清单表格已向用户展示
> - 用户在**当前对话**中明确回复确认

```bash
bitscli devops dev-task pipeline run --submit --from-prepare temp/bits_pipeline_run_prepare.json
```

**成功**（`code == 200`）：

向用户报告：流水线已触发，附 deep link（按上文 URL 解析规则拼装，`task_name` 需转 snake_case）：

```
✅ 流水线已触发！
链接：https://bits.bytedance.net/devops/{space_id}/develop/detail/{dev_basic_id}/flow?task={task_name_snake_case}
```

**失败**（`code != 200`）：

复述 CLI 返回的 `message`，按以下方向归类：

| 分类 | 说明 |
|------|------|
| 权限不足 | 需申请权限 |
| 流水线预检 | 配置/门禁不满足 |
| 环境/项目 | 泳道/集群/部署目标问题 |
| 变更/分支 | 分支不存在/变更未关联 |
| 自定义变量 | 必填未填/类型非法 |
| RBAC/IAM | 资源鉴权 |
| 环境配置 | `env` 为空被拒 → 引导到流程页手动配置 |
