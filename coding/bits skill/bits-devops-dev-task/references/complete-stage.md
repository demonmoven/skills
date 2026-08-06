# 功能五：完成阶段 & 关联发布单

> **入口判定**：用户说"完成开发阶段"、"完成测试阶段"、"完成合入阶段"、"推进到下一阶段"、"关联发布单"等。用户可直接提供 `dev_task_id` 或开发任务 URL。

用于在 Dev Task 中完成某个阶段或关联发布单。支持阶段：**开发**、**测试**、**合入**；支持独立操作：**关联发布单**。
本功能严格遵循：**获取数据 → 校验阶段 → 校验任务 + 准出 → 汇总判定 → 执行或阻断**。

## 目录

- [输入解析](#输入解析)
- [关联发布单流程](#关联发布单流程)
- [核心流程（阶段完成）](#核心流程阶段完成) — Step 1 ~ Step 5
- [准出检测项展示规则](#准出检测项展示规则)
- [用户提示规范](#用户提示规范)
- [错误处理](#错误处理)
- [输出纪律](#输出纪律)
- [阶段名中文映射](#阶段名中文映射)

---

## 输入解析

### 1) 开发任务 ID 解析

用户可传数字 ID，或 URL：

`https://bits.bytedance.net/devops/{space_id}/develop/detail/{dev_task_id}?...`

提取 `detail/` 后的数字作为 `dev_task_id`（即 `dev_basic_id`，二者等价），同时提取 `devops/` 后的数字作为 `space_id`（后续构建跳转链接时使用）。

> **`space_id` 不可知时**：若用户仅提供数字 ID（无 URL），则 `space_id` 未知。此时所有需要 `space_id` 的跳转链接均以 `{space_id}` 占位，并在链接旁注明「请将链接中 `{space_id}` 替换为实际空间 ID」。

### 2) 目标操作标准化

将用户表达映射为目标操作：

| 用户输入 | 目标操作 |
|---|---|
| 开发 / dev / develop | 阶段完成 → `DevDevelopStage` |
| 测试 / 准入 / test / access | 阶段完成 → `DevAccessStage` |
| 合入 / merge | 阶段完成 → `DevMergeStage` |
| 推进阶段 / 推进 / 完成阶段 / **未指定具体阶段** | 先执行命令 A 获取 workflow，以 `data.currentStage` 作为目标阶段，再继续后续步骤 |
| 关联发布单 / associate release ticket | **关联发布单**（独立流程） |

若用户直接给 fixedName（如 `DevDevelopStage`），按原值使用。

> **重要**：当用户说"推进阶段"、"帮我推进"等但**未明确指定**开发/测试/合入时，不要猜测阶段，必须先获取 workflow 数据，以 `data.currentStage` 为目标阶段。

> **注意**：后端 `currentStage` 返回的合入阶段 fixedName 可能为 `DevMergeStage` 或 `DevCodeMergeStage`，两者均视为合入阶段。在判断 `currentStage` 时需同时匹配这两个值。

**若目标操作为「关联发布单」，直接跳转到「关联发布单流程」章节，不执行核心流程 Step 1–5。**

---

## 关联发布单流程

当用户输入"关联发布单"时，执行以下独立流程（不走阶段完成的 Step 1–5）。

**读取 `references/associate-release-ticket-flow.md` 执行关联发布单流程。**

---

## 核心流程（阶段完成）

> ⛔ **执行约束（必读）**
> 1. 必须严格按 **Step 1 → 2 → 3 → 4** 顺序执行，禁止跳过任何步骤。
> 2. Step 3 结束后，必须**先向用户输出检查结论**（格式见 Step 3 末尾），再进入 Step 4 分支。禁止跳过结论输出直接调用 `complete-stage`，也禁止"先试调用、失败了再回头查原因"。

### Step 1（强制）: 获取数据

根据目标阶段是否已知，决定调用策略：

**① 目标阶段已知（`DevDevelopStage` 或 `DevAccessStage`）**：**并行**发起以下两个命令（两者无数据依赖）：

```bash
# 命令 A：获取 workflow（含阶段、任务状态）
bitscli devops dev-task process workflow --dev-task-id <dev_task_id>

# 命令 B：获取准出检测信息
bitscli devops gatekeeper checkpoint-info --dev-basic-id <dev_task_id> --stage <目标阶段 fixedName>
```

**② 目标阶段未知（用户未指定具体阶段）**：先**单独执行命令 A**，从返回的 `data.currentStage` 确定目标阶段后，再按上方规则决定是否补充执行命令 B：

- 若 `currentStage` 为 `DevDevelopStage` 或 `DevAccessStage` → 继续执行命令 B
- 若 `currentStage` 为 `DevMergeStage` / `DevCodeMergeStage` → 不执行命令 B（合入阶段走独立流程）

**③ 目标阶段已知且为 `DevMergeStage`**：**仅执行命令 A**，不执行命令 B（合入阶段走独立流程，不做准出校验）。

所有必要命令返回后，进入 Step 2。

---

### Step 2（强制）: 校验 currentStage

从命令 A（workflow）的返回数据读取：

- `data.currentStage`：当前进行中的阶段 fixedName
- `data.stages[]`：所有阶段信息（含 `status`、`tasks[]`）

**必须校验**：

- 若 `data.currentStage !== 目标阶段 fixedName`：
  - **立即终止**，不执行任何后续步骤；
  - 向用户提示阶段不一致（文案见「用户提示规范 → 阶段不一致」）。
- 若一致且**目标阶段为 `DevMergeStage` / `DevCodeMergeStage`**（合入阶段）：
  - 先检查该 stage 的 `status`：若为 `succeeded(2)` 或 `skipped(3)`，直接提示 `当前开发任务的合入阶段已完成。\n\n请前往[开发任务详情](https://bits.bytedance.net/devops/{space_id}/develop/detail/{dev_task_id}?devops_space_type=server_fe)查看` 并结束。
  - 否则，读取 `references/merge-stage-flow.md` 执行合入阶段特殊流程，**不执行 Step 3/4/5**。
- 若一致且目标阶段不是合入阶段 → 进入 Step 3。

---

### Step 3（强制）: 检查任务状态 + 解析准出结果

本步骤**不产生新的 API 调用**，仅解析 Step 1 已获取的数据。

#### 3a: 检查当前阶段 Tasks 状态（来自命令 A）

在 `data.stages[]` 中定位 `fixedName == data.currentStage` 的 stage，检查其 `tasks[]`：

- 判定为"已完成"：`succeeded(2)`、`skipped(3)`
- 判定为"未完成"：`pending(0)`、`running(1)`、`failed(4)`、`canceled(5)`、`held(6)`

**遍历全部 tasks**，收集所有未完成的 Task 及其 `canSkip` 字段（不要在发现第一个未完成时就终止）。

Task 状态中文映射：

| 枚举值 | 中文 |
|---|---|
| 0 (pending) | 未开始 |
| 1 (running) | 运行中 |
| 4 (failed) | 失败 |
| 5 (canceled) | 已取消 |
| 6 (held) | 暂停 |

#### 3b: 解析准出检测结果（来自命令 B）

无论 3a 是否有未完成 Task，都**必须解析命令 B 的返回**（需收集全部阻断信息后统一输出）：

- `passFlag == true`：准出通过
- `passFlag != true`：准出未通过，收集未通过的检测项信息（用于 Step 4b 展示）

#### 3c: 输出检查结论（强制，不可跳过）

Step 3a 和 3b 解析完成后，**必须先向用户输出以下检查结论**，再进入 Step 4：

```text
当前「<目标阶段中文>」阶段推进检查：
- 阶段任务：<N> 项全部完成 ✅ / <M> 项未完成 ❌（<未完成任务名列表>）
- 准出检测：已通过 ✅ / 未通过 ❌（<passedCount>/<totalCount>）
```

> **这段结论是 `complete-stage` 的前置输出**——Agent 必须基于真实数据填充此模板，填充后根据结论自然分支到 Step 4a 或 Step 4b。没有这段输出，禁止执行任何后续动作。

---

### Step 4: 执行或阻断

基于 Step 3c 输出的检查结论分支：

- **若阶段任务全部完成且准出已通过**（两项均 ✅）→ 执行阶段完成（Step 4a）
- **若任一未通过**（存在 ❌）→ **不调用 `complete-stage`**，直接展示阻断文案（Step 4b）+ 操作菜单（Step 4c）

#### Step 4a: 执行阶段完成

```bash
bitscli devops dev-task process complete-stage --dev-task-id <dev_task_id> --stage <目标阶段 fixedName>
```

**命令返回错误时的兜底处理**：

> 此分支仅用于竞态条件（Step 1–3 数据获取后状态发生变化），不是跳过前置校验的替代路径。

若命令执行失败（如"不允许推进"、"stage not allowed"等），不要直接输出错误信息。必须：

1. 重新从 Step 1 开始获取最新 workflow + 准出数据
2. 按 Step 2–3 重新校验并解析阻断项
3. 按 Step 4b + Step 4c 格式展示阻断项和可用操作

#### Step 4b: 输出阻断文案

见「[用户提示规范 → 阻断输出](#阻断输出task-未完成-和或-准出未通过)」，按该节定义的格式输出。

附加规则（Step 4b 专用）：

- 若某个未完成 Task 的 `canSkip == true`，在该 Task 条目下追加一行：`（该任务支持跳过，是否跳过？）`
- 若有未完成 Task 但准出通过，不展示准出部分

#### Step 4c: 展示可用操作菜单

Step 4b 输出完毕后，根据阻断内容动态构建操作菜单，**不要自动执行任何操作**，等待用户明确指示。

菜单项生成规则（仅展示适用选项，编号连续）：

| 条件 | 菜单项 |
|---|---|
| 存在 `canSkip == true` 的未完成 Task | 逐个列出可跳过任务 |
| 准出中发布单检查未关联（`releaseTicketNotAssociated == true`） | 关联发布单 |
| 准出未通过（`passFlag != true`） | 重新运行全部检测项 |
| 准出未通过（`passFlag != true`） | 重新运行指定检测项 |
| 存在 `skippable == true` 的检测项 | 逐个列出可跳过检测项 |
| 始终 | 暂不操作 |

**展示格式**：

```text
可用操作：
  a) 跳过任务「<任务显示名>」(<taskFixedName>)
  b) 关联发布单
  c) 重新运行全部检测项
  d) 重新运行指定检测项（请指定序号或名称）
  e) 跳过检测项「<检测项名称>」（可跳过）
  f) 暂不操作
请告诉我你要执行哪些操作（可多选，如 "a,c" 或 "跳过任务并重跑检测"）：
```

> **核心原则**：Step 4b + Step 4c 作为一个整体输出后**立即停止**，等待用户回复。不要自动执行跳过、关联、重跑等任何操作。

---

### Step 5: 执行用户选择的操作

等待用户回复后，按用户指示执行对应操作。支持用户一次选择多个操作，按顺序依次执行。

#### 5a) 跳过任务

```bash
bitscli devops dev-task process skip-task --dev-task-id <dev_task_id> --stage <目标阶段 fixedName> --task-name <taskFixedName>
```

执行后输出：`已跳过任务「<任务显示名>」(<taskFixedName>)。`

#### 5b) 关联发布单

读取 `references/associate-release-ticket-flow.md` 执行关联发布单流程。

#### 5c) 重新运行全部检测项

使用 Step 3b 返回的 `info.checkPointID`：

```bash
bitscli devops gatekeeper retry-checkpoint --checkpoint-id <checkPointID>
```

执行后输出：`已发起重新运行全部检测项（checkpointID: <checkPointID>），请稍后再次查看检测状态。`

#### 5d) 重新运行单个检测项

用户指定要重跑的检测项（通过序号或名称匹配），使用该检测项的 `checkID`：

```bash
bitscli devops gatekeeper retry-check --check-id <checkID>
```

执行后输出：`已发起重新运行检测项「<检测项名称>」（checkID: <checkID>），请稍后再次查看检测状态。`

#### 5e) 跳过检测项

仅当目标检测项的 `skippable == true` 时可执行：

```bash
bitscli devops gatekeeper skip-check --check-id <checkID> --skip-info '{"reason":"manual_skip"}' --skip-reason '用户手动跳过'
```

- 若 `skippable != true`，提示：`该检测项「<检测项名称>」不支持跳过。`
- 跳过成功后输出：`已跳过检测项「<检测项名称>」。`
- 跳过失败时输出错误信息。

#### 5f) 暂不操作

直接结束，不执行任何操作。

#### 执行后重新校验

所有用户选择的操作执行完毕后：

1. 输出已执行操作的简要汇总（格式见「用户提示规范 → 执行操作后的输出」）
2. **重新从 Step 1 开始执行**完整流程，校验阶段状态是否已可完成
3. 若 3a + 3b 均通过 → 执行 Step 4a 完成阶段
4. 若仍有阻断 → 按 Step 4b + Step 4c 格式再次展示，等待用户下一步指示
5. **最多自动重新校验 2 轮**（不含用户主动发起的重试）；2 轮后仍有阻断，提示用户前往 Bits 界面手动处理

> **注意**：
> - 可重跑的检测项状态：失败(8)、异常(10)、未执行(5)。对于执行中(4)、排队中(7)、待交互(6) 的检测项，重跑无意义，若用户选择了这些项，提示「该检测项当前处于 <状态中文>，无需重跑」。
> - 可跳过的检测项：由 `skippable == true` 决定，与状态无关。

---

## 准出检测项展示规则

当 `passFlag != true` 时，展示准出检测概况及**未通过**的检测项明细。

### 整体概况行

```text
准出检测未通过  进度 <passedCount>/<totalCount>
```

- `totalCount` = `checkList.length`
- `passedCount` = status 为 成功(3)、跳过(1)、警告(9) 的检测项数量之和

### 筛选与排序

**仅展示未通过的检测项**。以下 status 视为未通过：5（未执行）、6（待交互）、7（排队中）、8（失败）、10（异常）、4（执行中）。

排序权重（靠前先展示）：失败(8) → 异常(10) → 待交互(6) → 执行中(4) → 排队中(7) → 未执行(5)。同状态内按 `priority` 升序。

### 单条检测项格式

每条检测项作为 Markdown 子列表项输出（`- ` 前缀），确保渲染时独立换行：

```markdown
- <图标> <检测项名称>  [建议优先处理]  [可跳过]  [状态标签] <描述信息>
```

**图标规则**：

| 状态标签 | 图标 | 说明 |
|---|---|---|
| `[阻塞]` | ❌ | 失败/异常/待交互，需要处理 |
| `[不阻塞]` | ⚠️ | 失败但不影响推进 |
| `[检测中]` / `[待开始]` | ⏳ | 进行中或等待态，无需图标强调 |

> 内部需记录每条检测项的 `checkID`（取自 `checkList[].checkID`）和 `skippable`（取自 `checkList[].skippable`），用于 Step 5 重跑/跳过时匹配目标，但**不向用户展示** `checkID`。

**检测项名称**：优先 `nameI18N`，其次 `name`。

**建议优先处理**（可选）：仅当 `status ∈ {6,7,8,10}` 且 `checkLevel == CHECK_LEVEL_ERROR` 且 `priority == 0` 时展示。

**可跳过**（可选）：仅当 `skippable == true` 时展示 `[可跳过]` 标记。

**状态标签**（对齐前端 `block-status.tsx`）：

| status | 标签 |
|---|---|
| 4（执行中） | `[检测中]` |
| 6（待交互）/ 7（排队中） | `[阻塞]` |
| 8（失败）/ 10（异常）且 `checkLevel == CHECK_LEVEL_ERROR` | `[阻塞]` |
| 8（失败）/ 10（异常）且 `checkLevel != CHECK_LEVEL_ERROR` | `[不阻塞]` |
| 5（未执行）/ 其他 | `[待开始]` |

**描述信息**：优先使用 `messageI18N`，其次 `message`。若非空直接使用。若为空，则按以下各检测项类型规则从 `checkResult` 构建。

---

### 流水线运行检测（checkItemType=30）

数据源：`checkResult.stagePipelineRunningCheckResult.taskResults[]`

遍历 `taskResults`，跳过 `taskStatus` 为 succeeded/skipped 的子阶段，对每个未完成子阶段统计 `items[]` 的 `pipelineStatus`：

| pipelineStatus | 归类 |
|---|---|
| NotStarted / Canceled | notStartedCount |
| Running / Canceling | runningCount |
| Succeed | successCount |
| Pending | pendingCount |
| Failed | failedCount |

文案拼接：
- 基础：`当前 {successCount}/{totalCount} 个流水线完成`
- 若有失败：`；{failedCount} 条流水线运行失败`
- 若有待人工确认：`；{pendingCount} 条流水线待人工确认`
- 若有未运行：`，{notStartedCount} 条流水线待运行`
- 若有运行中：`，{runningCount} 个运行中`
- 若有失败或待确认：末尾追加名称列表 `：[CN]主流水线、[CN]项目名`（主流水线用 `[CN]主流水线`，项目流水线用 `[CN]<projectName>`，多个用 `、` 分隔）
- 多子阶段时每段前加 `「<taskDisplayName>」`，多段用 `；` 分隔；单子阶段不加前缀

示例：

```markdown
- ❌ 流水线运行检测  建议优先处理  [阻塞] 当前 1/3 个流水线完成；2 条流水线待人工确认：[CN]主流水线、[CN]lies.kefu.c_platform_module
```

---

### 代码可合入性检查（checkItemType=2）

数据源：`checkResult.codeConflictCheckResult.conflictDevChanges[]`

将 `conflictDevChanges` 按 `failReason` 分组，按以下规则生成文案，多条原因用 `；` 分隔：

| failReason | 文案模板 |
|---|---|
| `closed` / `abandoned` | 仓库 {repoNames} 的 MR 已经被关闭，请在变更列表中查看详情 |
| `merged` / `submitted` | 仓库 {repoNames} 的 MR 已经被合入，请在变更列表中移除对应项目后重试 |
| `repo_archived` | 仓库 {repoNames} 的 MR 已经被归档，请在变更列表中移除对应项目 |
| `branch_missing` / `missing_branch` | 仓库 {repoNames} 的目标分支/开发分支不存在，请在代码服务中创建分支 |
| `no_mr` | 仓库 {repoNames} 未创建 MR |
| `wip` / `draft` / `is_draft` | 仓库 {repoNames} 的 MR 处于草稿状态 |
| `no_commits` | 仓库 {repoNames} 未发现有增量代码，无需合入，请提交代码或移除对应仓库 |
| `conflict` | 仓库 {repoNames} 存在代码冲突，前往 [开发任务]({devTaskChangeUrl}) 解决冲突 |
| `rebase_needed` | 仓库 {repoNames} 的开发分支落后于目标分支，请尽快完成分支同步 |
| `branch_protected` | 仓库 {repoNames} 的目标分支设置了保护策略，请确保机器人具备"允许合并"权限 |
| `not_allowed` | 你没有仓库 {repoNames} 的权限 |
| `thread_unresolved` / `discussions_unresolved` | 仓库 {repoNames} 的 MR 中存在未解决的评论 |
| `mr_creation_failed` | 仓库 {repoNames} 的 MR 创建失败 |
| 其他 | {repoNames} 变更不可合入 |

> `{repoNames}` = 同一 failReason 下所有 `devChangeName`（仓库名），用 `、` 分隔。
>
> `{devTaskChangeUrl}` = `https://bits.bytedance.net/devops/{space_id}/develop/detail/{dev_task_id}/change/{devChangeId}?devops_space_type=server_fe`
> 其中 `space_id` 和 `dev_task_id` 从输入解析获取，`devChangeId` 取自 `conflictDevChanges[].devChangeId`。
> 当同一 failReason 下有多个仓库冲突时，为每个仓库单独生成一条带各自链接的文案，用 `；` 分隔。

示例：

```markdown
- ❌ 代码可合入性检查  [阻塞] 仓库 repo-a 存在代码冲突，前往[开发任务](https://bits.bytedance.net/devops/12345/develop/detail/6789/change/1001?devops_space_type=server_fe)解决冲突；仓库 repo-b 存在代码冲突，前往[开发任务](https://bits.bytedance.net/devops/12345/develop/detail/6789/change/1002?devops_space_type=server_fe)解决冲突；仓库 repo-c 的开发分支落后于目标分支，请尽快完成分支同步
```

---

### 发布单可合入性检查（checkItemType=6）

数据源：`checkResult.releaseTicketCheckResult`

按以下优先级生成文案：

1. **未关联发布单**（`releaseTicketNotAssociated == true`）：

   输出阻塞文案及候选列表（候选列表来自 `references/associate-release-ticket-flow.md` 中的"查询可关联发布单"步骤），格式：

   ```text
   当前开发任务未关联发布单，请关联一个处于"集成中"的发布单
   可关联发布单（集成中）：
   1. [集成中]发布单-A  2026-03-30 10:00:00  alice  研发流程-主干
   2. [集成中]发布单-B  2026-03-29 18:23:11  bob  研发流程-主干
   ```

   将"关联发布单"作为 Step 4c 操作菜单中的一个选项展示，**等待用户在操作菜单中选择后**再执行关联，不自动触发。

2. **有不可合入项目**（`unmergeableProjects[]` 非空）：按 `failReason` 分组拼接，用 `；` 分隔：
   - `FailReasonAlreadyDeployed`：`项目 {projectNames} 在发布单中已经部署成功，无法再次部署`
   - `FailReasonRollBack`：`项目 {projectNames} 在发布单中已经回滚，无法再次部署`
   - `FailReasonDifferentMainSCM`：`{projectNames} 在集成区存在相同控制面的不同主仓`
   - `FailReasonProjectIntegrationLock`：`项目+控制面已锁定：{projectNames} 无法合入，请前往发布单联系发布负责人解锁`

3. **集成分支被占用**（`integrationBranchOccupiedInfos[]` 非空）：
   `{projectNames} 项目下，目标分支在其他发布单 {releaseTicketName} 中被使用；前往该发布单完成或取消发布`

4. **集成区状态不允许合入**：`当前发布单集成区状态为 {integrationStatus}，不允许合入`

---

### Codebase CI（checkItemType=4）

数据源：`checkResult.codebaseCICheckResult.totalDevChanges[]`

每个 devChange 含 `checkRuns[]`，按仓库聚合状态后统计：

| 仓库状态 | 计数 |
|---|---|
| 有 Failed 的 checkRun | failedCount |
| 有 NeedOperate 的 checkRun（无 Failed） | needOperateCount |
| 有 Running 的 checkRun（无 Failed/NeedOperate） | runningCount |
| 全部 Success 或 Skipped | passedCount |

文案拼接：
- 未执行状态时：`绑定发布单或创建 MR 后触发检测`
- 执行中时：`{passedCount}/{totalCount} 个仓库已通过`
- 异常时：按顺序拼接非零项，用 `；` 分隔：
  - `{failedCount} 个仓库失败`
  - `{needOperateCount} 个仓库需要手动操作`
  - `{runningCount} 个仓库进行中`
  - 末尾追加已通过信息（如有）：`，{passedCount} 个仓库成功`

示例：

```markdown
- ❌ Codebase CI  [阻塞] 2 个仓库失败；1 个仓库需要手动操作，1 个仓库成功
- ⏳ Codebase CI  [待开始] 绑定发布单或创建 MR 后触发检测
```

---

### Code Review（checkItemType=3）

数据源：`checkResult.crCheckResult`

按以下优先级生成文案：

1. **未执行**（status=5）：`待 MR 创建后开始检测`
2. **有未通过 MR**（`notPassDevChangeCount > 0`）：
   - 有评审人（`reviewInfos` 中有未通过且含 reviewersInfo）：`存在 {notPassDevChangeCount} 个 MR 评审未通过，请联系评审人去处理`
   - 无评审人：`存在 {notPassDevChangeCount} 个 MR 评审未通过，仓库未配置评审人`

示例：

```markdown
- ❌ Code Review  [阻塞] 存在 2 个 MR 评审未通过，请联系评审人去处理
- ⏳ Code Review  [待开始] 待 MR 创建后开始检测
```

---

### 其他检测项

对于上述未列出的 `checkItemType`（如安全合规、质量卡点等），**不使用** `messageI18N`/`message` 原文（原文通常为英文且不含有效操作入口），统一使用以下描述文案：

```text
检测未通过，请点击[开发任务详情]({devTaskUrl})处理。
```

> `{devTaskUrl}` = `https://bits.bytedance.net/devops/{space_id}/develop/detail/{dev_task_id}?devops_space_type=server_fe`
>
> 使用 Markdown 链接语法，「开发任务详情」为可点击文本，不向用户暴露原始 URL。

示例：

```markdown
- ❌ 安全合规  [阻塞] 检测未通过，请点击[开发任务详情](https://bits.bytedance.net/devops/12345/develop/detail/6789?devops_space_type=server_fe)处理。
- ⚠️ 质量卡点  [不阻塞] 检测未通过，请点击[开发任务详情](https://bits.bytedance.net/devops/12345/develop/detail/6789?devops_space_type=server_fe)处理。
```

---

## 用户提示规范

### 阶段不一致

```text
当前进行中的阶段是「<当前阶段中文>」(<currentStage>)，与您要求完成的阶段「<目标阶段中文>」(<targetStage>) 不一致。
为避免误操作，本次未执行阶段完成。请确认阶段后重试。

请前往[开发任务详情](https://bits.bytedance.net/devops/{space_id}/develop/detail/{dev_task_id}?devops_space_type=server_fe)查看
```

### 阻断输出（Task 未完成 和/或 准出未通过）

```markdown
下列事项处理完成后，即可完成<目标阶段中文>阶段：

1. <任务显示名>(<fixedName>) <状态中文>
   <任务显示名>子阶段暂未完成，当前处于<状态中文>
   （该任务支持跳过，是否跳过？）            ← 仅 canSkip == true 时追加

2. 准出检测未通过  进度 0/5
   - ❌ 流水线运行检测  建议优先处理  [阻塞] 当前 1/3 个流水线完成；2 条流水线待人工确认：[CN]主流水线、[CN]lies.kefu.c_platform_module
   - ❌ 代码可合入性检查  [阻塞] 仓库 repo-a 存在代码冲突，前往[开发任务](https://bits.bytedance.net/devops/{space_id}/develop/detail/{dev_task_id}/change/{devChangeId}?devops_space_type=server_fe)解决冲突
   - ❌ 发布单可合入性检查  [阻塞] 当前开发任务未关联发布单，请关联一个处于"集成中"的发布单
     可关联发布单（集成中）：
     1. [集成中]发布单-A  2026-03-30 10:00:00  alice  研发流程-主干
     2. [集成中]发布单-B  2026-03-29 18:23:11  bob  研发流程-主干
   - ⏳ Codebase CI  [待开始] 绑定发布单或创建 MR 后触发检测
   - ⏳ Code Review  [待开始] 待 MR 创建后开始检测

请前往[开发任务详情](https://bits.bytedance.net/devops/{space_id}/develop/detail/{dev_task_id}?devops_space_type=server_fe)查看
```

编号规则：先列出所有未完成 Task（每个 Task 一条），若准出未通过则追加一条。若无未完成 Task 但准出未通过，直接从编号 1 开始。若有未完成 Task 但准出通过，不展示准出部分。

### 执行操作后的输出

```text
已执行以下操作：
- <操作描述>
- <操作描述>

<重新校验结果>
```

- 若重新校验后 3a + 3b 均通过 → `<重新校验结果>` 使用「执行成功」格式
- 若仍有阻断 → `<重新校验结果>` 使用阻断输出 + Step 4c 操作菜单格式

操作描述模板：

| 动作 | 描述格式 |
|---|---|
| 关联发布单 | `已关联发布单「<发布单名称>」` |
| 跳过任务 | `已跳过任务「<任务显示名>」(<taskFixedName>)` |
| 重跑检测项 | `已发起重新运行检测项「<检测项名称>」` |
| 跳过检测项 | `已跳过检测项「<检测项名称>」` |

> **严格遵守**：
> - 「已执行以下操作」部分仅简要列出已执行的动作，每条一行，不加额外解释。
> - 阻断项列表必须严格使用上方定义的格式（含 emoji、状态标签、描述文案），禁止用自然语言改写或省略。
> - 禁止在阻断项之后添加"你需要处理"、"建议你"、"接下来你需要"等自创建议段落。
> - 禁止添加"处理完后回复我 xxx"之类的引导语。

### 执行成功

```text
已完成「<目标阶段中文>」阶段操作。
开发任务：<dev_task_id>

请前往[开发任务详情](https://bits.bytedance.net/devops/{space_id}/develop/detail/{dev_task_id}?devops_space_type=server_fe)查看
```

---

## 错误处理

| 错误现象 | 可能原因 | 处理方式 |
|----------|----------|----------|
| `complete-stage` 返回"不允许推进"/"未完成" | 竞态条件（Step 1–3 数据获取后状态已变化） | 禁止直接输出错误。重新执行 Step 1–3 获取最新数据，按 Step 4b + Step 4c 格式展示阻断项 |
| `gatekeeper checkpoint-info` 返回异常 | 准出接口调用失败 | 展示错误信息，建议用户前往页面手动查看 |

---

## 输出纪律

- **每轮完整输出的末尾**追加一次：`请前往[开发任务详情](https://bits.bytedance.net/devops/{space_id}/develop/detail/{dev_task_id}?devops_space_type=server_fe)查看`（`{space_id}`、`{dev_task_id}` 替换为实际值；若 `space_id` 未知则保留占位并提示用户替换）。一轮输出中仅出现一次，不要在中间步骤重复。
- **必须严格使用本文档定义的输出模板**，禁止自由发挥。
- **禁止自动执行操作**。展示阻断文案和操作菜单后，必须等待用户明确指示再执行。
- **禁止**添加"你需要处理"、"建议你"、"接下来你需要"等自创建议段落。
- **禁止**添加"处理完后回复我 xxx"之类的引导语。
- **禁止**用自然语言改写阻断项（如不要写"Code Review 仍未通过"，要写 `- ❌ Code Review [阻塞] 存在 x 个 MR 评审未通过，请联系评审人去处理`）。
- 不要向用户输出原始 JSON、CLI 命令名、内部字段名。
- 不要输出"根据技能规则"之类的内部说明。
- 所有阶段名对用户展示时优先给中文，并附 fixedName（如有必要）。
- 枚举值必须转为中文/可读文案再向用户展示。
- 检测项展示优先使用 i18n 名称字段（`nameI18N`/`messageI18N`），其次使用默认字段（`name`/`message`）。

---

## 阶段名中文映射

| fixedName | 中文 |
|---|---|
| `DevDevelopStage` | 开发 |
| `DevAccessStage` | 测试 |
| `DevMergeStage` / `DevCodeMergeStage` | 合入 |
