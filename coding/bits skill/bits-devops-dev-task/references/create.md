# 功能二：创建开发任务

> **入口判定**：用户说"创建开发任务"、"新建 dev task"、"帮我建一个开发任务"、"create dev task"。

## 目录

- [用户输入解析规则](#用户输入解析规则)
- [核心流程](#核心流程) — Step 1 解析 → Step 2 prepare → Step 3 展示 → Step 4 submit
- [参数默认值说明](#参数默认值说明)
- [创建专属错误](#创建专属错误)
- [完整示例](#完整示例)

采用**两阶段**流程：先 prepare 校验参数，用户确认后再 submit 真正创建。

> **⛔ 强制规则 — 不可绕过的确认机制**
>
> 1. **禁止复用旧文件**：工作区中可能存在上一次运行遗留的 `bits_dev_task_create_prepare.json` 或 `bits_dev_task_create_request.json`，**严禁**直接读取这些文件执行 submit。每次创建都必须从 Step 1 开始完整执行。
> 2. **禁止跳步**：必须严格按 Step 1 → Step 2 → Step 3 → Step 4 顺序执行，不得以任何理由（包括"文件已存在""参数已齐全"等）跳过中间步骤。
> 3. **用户确认是唯一的 submit 前提**：只有在 Step 3 向用户展示完整参数清单表格、且用户**在当前对话中明确回复确认**后，才能执行 Step 4 的 `--submit`。Agent 自行判断"参数正确"不构成确认。

## 用户输入解析规则

### Meego URL 解析

当用户输入 Meego URL 时，Agent 须自动解析出 `typeKey` 和 `id`，构造 `--meego` 参数：`--meego '{"id":"6877195493","typeKey":"story"}'`

| 用户输入 | 解析结果 |
|----------|----------|
| `https://meego.larkoffice.com/bitsdevops/story/detail/6877195493` | `{"id":"6877195493","typeKey":"story"}` |
| `https://meego.larkoffice.com/bitsdevops/issue/detail/1234567890` | `{"id":"1234567890","typeKey":"issue"}` |

### 项目参数解析

用户可能以自然语言描述项目信息，Agent 须将其结构化为 `--projects` JSON 数组：

- **projectUniqueId**：项目唯一标识，如 `toutiao.canal.liangzai`
- **branch**：开发分支名，如 `feat/xxx`
- **type**：项目类型简写，默认 `TCE`（可选值：`TCE`、`FAAS`、`CRONJOB`、`WEB`、`HYBRID`、`CUSTOM`）
- **controlPanel**：控制面，默认 `CN`（可选值：`CN`、`I18N`、`EU_TTP`、`US_TTP`、`I18N_BD`）

示例：用户说"项目 toutiao.canal.liangzai，分支 feat/123"，构造为：

```bash
--projects '[{"projectUniqueId":"toutiao.canal.liangzai","branch":"feat/123"}]'
```

## 核心流程

### 整体编排（循环 prepare → 展示表格 → 用户确认 → submit）

```
用户表达意图
    ↓
Step 1: 解析用户输入，构造已知参数
    ↓
┌──→ Step 2: 调用 dev-task create（prepare 阶段）
│        ↓
│    Step 3: 解析 prepare 输出，展示【完整参数清单表格】（每次必须展示）
│        ↓
│        ├── 有缺失必填项 → 表格中标记 ❌，提示用户补充
│        │       ↓
│        │   用户补充参数 ──→ 回到 Step 2 重新 prepare ─┘
│        │
│        └── ready === true → 表格展示所有参数已就绪 ✅，请求用户确认
│                ↓
│           用户确认"是" → Step 4: 调用 dev-task create --submit
│           用户说"否"/要修改 ──→ 回到 Step 2（携带修改后的参数）─┘
```

**关键规则**：每次执行 Step 2 → Step 3 时，**都必须向用户展示完整的参数清单表格**，无论参数是否完整。这确保用户在每一轮循环中都能清楚看到当前所有参数的状态和变化。

### Step 1：解析用户输入

从用户的自然语言描述中提取可用参数：

| 参数 | 来源 |
|------|------|
| `--name` | 用户直接提供的任务名称 |
| `--meego` | 用户提供的 Meego URL（按上述规则解析）或直接给出的 Meego ID + 类型 |
| `--projects` | 用户提供的项目名 + 分支（可能分多次补充） |
| `--space-id` | 用户指定的空间 ID（通常不需要，会从最近任务自动获取） |
| `--teamflow-id` / `--workflow-id` | 用户指定的流程/模版（通常不需要，会从最近任务自动获取） |

**重要约束**：Step 1 只提取用户**已明确提供**的参数。用户未提及 `meego` 时，**不得主动追问**——是否需要 `meego` 由 Step 3 根据 prepare 输出决定。

> **⛔ 禁止提前收集信息**：无论用户是否提供了任何参数，Step 1 结束后必须**立即进入 Step 2 执行 prepare**，不得以"请先提供项目/分支/Meego 等信息"为由停下来向用户提问。缺什么参数由 Step 3 的 prepare 结果告知用户，**prepare 是唯一的参数收集入口**。

### Step 2：调用 prepare（参数校验）

将已知参数拼入命令并执行（**只传用户已提供的参数**）。**不带 `--submit`**，此命令仅做参数校验，返回 JSON。

> **⛔ 关键规则：用户未提供任何参数时，直接以零参数调用 prepare（不附加任何选项），不得提前向用户要求补充参数。**

```bash
# 用户未提供任何参数时（最常见场景：用户仅说"创建开发任务"）
bitscli devops dev-task create

# 用户提供了 meego 的场景
bitscli devops dev-task create \
  --projects '[{"projectUniqueId":"xxx","branch":"feat/123"}]' \
  --name '任务名称' \
  --meego '{"id":"6877195493","typeKey":"story"}'

# 用户只提供了项目/分支，未提供 meego 的场景
bitscli devops dev-task create \
  --projects '[{"projectUniqueId":"xxx","branch":"feat/123"}]' \
  --name '任务名称'
```

### Step 3：解析 prepare 输出，展示完整参数清单表格

**核心原则**：**每次 prepare 调用后，都必须向用户展示完整的参数清单表格**，无论 `ready` 为 `true` 还是 `false`。这样用户在每一轮循环中都能清楚看到所有参数的当前状态。

prepare 输出 JSON 结构：

```json
{
  "ready": false,
  "params": {
    "space_id":  { "value": 13740, "required": true, "source": "default", "is_display": true, "display": "某某空间名" },
    "template":  { "value": {}, "required": true, "source": "default", "is_display": true, "display": "研发流程名 - 模版名" },
    "name":      { "value": "任务名称", "required": true, "source": "user_input", "is_display": true },
    "meego":     { "value": {}, "required": false, "source": "user_input", "is_display": false },
    "projects":  { "value": [], "required": true, "source": "user_input", "is_display": true }
  },
  "missing_required": ["name"],
  "template_constraints": { "workitem_must": false }
}
```

**展示规则**：

1. **展示文案优先级**：有 `display` 字段优先使用 `display`；否则展示 `value`
2. **展示顺序**（固定，**不跟随 JSON key 顺序**）：`space_id`（空间）→ `template`（研发流程/模版）→ `name`（任务名称）→ `meego`（Meego 工作项）→ `projects`（变更项目）→ `env`（环境）
3. **来源标注**：
   - `user_input` → 「用户指定」
   - `default` → 根据参数区分：`space_id`/`template` →「默认值（来自最近创建的开发任务）」；`env` →「默认值（来自研发流程模版）」；其他 →「默认值」
   - `default_from_space_workflow`（仅 `template`）→ 「默认值（来自于空间流程的第一个）」
   - `meego` → 「自动获取自 Meego 工作项」
   - `common` → 「常用项目（自动填充）」
   - `missing` → 「❌ 未填写（必填）」
   - `space_id.display` 应展示空间名称，来源为 `GetSpaceInfoByID` 返回的 `nameRaw.value`
4. **仅展示 `is_display === true` 的参数**，`is_display === false` 的参数一律不在表格中展示**（无论是否已填写）**
5. **状态列**：已填写 → `✅`；缺失必填项（在 `missing_required` 中）→ `❌ 缺失`
6. **缺失项醒目提示**：表格下方对 `missing_required` 中的每一项，明确告知用户需要补充什么
7. **项目缺失时的常用项目提示**：当 `missing_required` 包含 `projects` 时，提示用户可通过 `bitscli devops dev-task common-projects` 查询常用项目。若 prepare 已包含 `common_projects` 数组，可直接展示。prepare 阶段若仅查到 1 个常用项目，会自动填充为默认值（`source` 为 `common`），此时仅需用户补充分支名
8. **Meego 展示与必填判定**：`meego` 是否展示由 `is_display` 决定。仅当 `meego.is_display === true` 且缺失时才提示用户补充；`meego.is_display === false` 时，**不在表格中展示，也不得要求用户提供 Meego URL**
9. **项目类型候选重点关注**：当 `projects[].typeCandidates` 存在且长度大于 1 时，必须标记为「需重点确认」，清晰展示候选列表，并明确询问用户最终采用的 `type`。**在用户确认前不得直接执行 submit**

**展示格式 — 有缺失必填项时（`ready === false`）**：

> 表格输出须遵守 SKILL.md「公共：输出纪律」中的 Markdown 表格强制规则。下方 `` ``` `` 仅为文档展示模板，实际输出时不得带代码块标记。

```
创建开发任务 — 参数校验结果

| 参数 | 值 | 来源 | 状态 |
|------|-----|------|------|
| 空间 | 某某空间名 | 默认值（来自最近创建的开发任务） | ✅ |
| 研发流程/模版 | 标准研发流程 - 标准开发任务模版 | 默认值（来自最近创建的开发任务） | ✅ |
| 任务名称 | — | — | ❌ 缺失 |
| 变更项目 | toutiao.canal.liangzai (feat/123, TCE, CN) | 用户指定 | ✅ |
| 环境 | ppe_bits_workflow_show_case | 默认值（来自研发流程模版） | ✅ |

⚠️ 以下必填参数仍需补充：
- **任务名称**（`name`）：请提供开发任务名称

请补充以上缺失项后，我会重新校验。
```

**展示格式 — 所有参数就绪时（`ready === true`）**：

```
创建开发任务 — 参数校验通过 ✅

| 参数 | 值 | 来源 | 状态 |
|------|-----|------|------|
| 空间 | 某某空间名 | 默认值（来自最近创建的开发任务） | ✅ |
| 研发流程/模版 | 标准研发流程 - 标准开发任务模版 | 默认值（来自最近创建的开发任务） | ✅ |
| 任务名称 | feat: 新功能 | 自动获取自 Meego 工作项 | ✅ |
| 变更项目 | toutiao.canal.liangzai (feat/123, TCE, CN) | 用户指定 | ✅ |
| 环境 | ppe_bits_workflow_show_case | 默认值（来自研发流程模版） | ✅ |

所有必填参数已就绪，是否确认创建？
```

> 注：Meego 工作项行仅在 `meego.is_display === true` 时才出现。**若 `meego.is_display === false`，则该行不展示。**
> 注：若某个项目返回多个 `typeCandidates`**（例如 `["TCE","WEB"]`）**，需在确认区单独高亮该项目并提示「请重点确认项目类型」。
> **重要**：无论参数是否完整，每次 prepare 后都必须展示上述完整表格。用户补充参数后重新 prepare 时，同样展示更新后的完整表格，让用户看到参数的变化。

### Step 3.5（可选）：用户要求切换流程/模版

当用户说"我想换个流程"或"用另一个模版"时，先查询可用列表：

```bash
bitscli devops dev-task workflows --workspace-id <space_id>
```

返回 JSON 中 `data.workflows` 数组，每项包含：

| 字段 | 说明 |
|------|------|
| `teamFlowID` | 研发流程 ID |
| `teamFlowName` | 研发流程名称 |
| `workflowId` | 开发任务模版 ID |
| `name` | 开发任务模版名称 |

向用户展示为可选列表：

```
当前空间可用的研发流程和模版：
1. 标准研发流程 - 标准开发任务模版
2. 标准研发流程 - 快速开发任务模版
3. 定制研发流程 - 自定义模版

请选择序号，或直接说模版名称。
```

用户选择后，将 `teamFlowID` 和 `workflowId` 作为 `--teamflow-id` 和 `--workflow-id` 传入下一次 prepare。

### Step 4：submit 创建

> **⛔ 前置条件**：Step 4 仅允许在以下条件**全部满足**时执行：
> - Step 2 的 prepare 已在**当前对话**中执行过（不得复用旧文件）
> - Step 3 的参数清单表格已向用户展示
> - 用户在**当前对话**中明确回复确认（如"是""确认""创建"）
>
> **违反以上任一条件直接执行 submit 属于严重错误。**

```bash
bitscli devops dev-task create --submit --from-prepare temp/bits_dev_task_create_prepare.json
```

**成功输出**：

```json
{
  "success": true,
  "dev_task_id": 123456,
  "url": "https://bits.bytedance.net/devops/..."
}
```

向用户展示：

```
✅ 开发任务创建成功！
- 任务 ID：123456
- 链接：https://bits.bytedance.net/devops/...
```

**失败输出**：

```json
{
  "success": false,
  "log_id": "20260322123456ABCDEF1234567890",
  "error": "创建失败原因",
  "create_form_url": "https://bits.bytedance.net/..."
}
```

向用户展示：

```
❌ 创建失败：<error 内容>
log_id：<log_id>
如需手动创建，可访问：<create_form_url>
```

**⛔ 确认步骤不可跳过**：即使 `ready === true`、即使工作区已有 prepare 文件，也必须先在当前对话中展示参数清单、等待用户明确确认后才能执行 submit。

## 参数默认值说明

| 参数 | 默认值来源 | 说明 |
|------|------------|------|
| `space_id` | 用户最近创建的开发任务 | 自动获取，通常无需用户指定 |
| `template`（teamflowId + workflowId） | 用户最近创建的开发任务 / 当前空间流程首个模版 | 默认从最近创建的开发任务获取；若用户切换了 `space_id` 且未指定模版，则取该空间流程列表的第一个 |
| `name` | Meego 工作项名称 | 传了 `--meego` 且未传 `--name` 时，自动从 Meego 拉取 |
| `projects[].type` | 自动推导 | 优先使用用户输入；未输入时按 `projectUniqueId` 推导（可选：`TCE`、`FAAS`、`CRONJOB`、`WEB`、`HYBRID`、`CUSTOM`）。若存在 `typeCandidates` 且多值，需提示用户确认 |
| `projects[].controlPanel` | `CN` | 控制面默认值 |
| `meego` 是否必填 | 取决于模版配置 | prepare 输出 `template_constraints.workitem_must` 反映该规则 |

## 创建专属错误

| 错误现象 | 可能原因 | 处理方式 |
|----------|----------|----------|
| prepare 输出 `missing_required` 含 `meego` | 模版要求关联工作项 | 当前模版要求关联 Meego 工作项，提示用户提供 Meego URL 或 ID |
| `--projects` 解析失败 | JSON 格式非法 | 检查并修正 JSON 格式 |
| submit 返回 `success: false` | 创建失败 | 展示 `error` 和 `log_id`，并提供 `create_form_url` 作为手动兜底 |

## 完整示例

**用户**："帮我创建一个开发任务，项目 toutiao.canal.liangzai，分支 feat/add-cache，meego 是 https://meego.larkoffice.com/bitsdevops/story/detail/6877195493"

**Agent 操作**：

1. 解析 Meego URL → `{"id":"6877195493","typeKey":"story"}`
2. 执行 prepare：
   ```bash
   bitscli devops dev-task create \
     --projects '[{"projectUniqueId":"toutiao.canal.liangzai","branch":"feat/add-cache"}]' \
     --meego '{"id":"6877195493","typeKey":"story"}'
   ```
3. 解析输出，发现 `ready === true`（name 自动从 Meego 拉取），向用户展示参数清单并请求确认
4. 用户确认后执行 submit：
   ```bash
   bitscli devops dev-task create --submit --from-prepare temp/bits_dev_task_create_prepare.json
   ```
5. 返回创建结果链接给用户
