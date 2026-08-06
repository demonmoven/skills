# 合入阶段特殊流程

当 Step 1 校验通过且 `目标阶段 === DevMergeStage` 时，**跳过 Step 2/3/4**，改为执行以下流程：

## 目录

- [M-Step 1: 提示合入操作暂不支持](#m-step-1-提示合入操作暂不支持)
- [M-Step 2: 获取合入详情数据](#m-step-2-获取合入详情数据)
- [M-Step 3: 展示合入队列](#m-step-3-展示合入队列如有)
- [M-Step 4: 展示 MergeTable](#m-step-4-展示-mergetable)
- [M-Step 5: 附加失败变更摘要](#m-step-5-附加失败变更摘要如有)
- [M-Step 6: 输出界面链接](#m-step-6-输出界面链接)

## M-Step 1: 提示合入操作暂不支持

向用户输出提示：

```text
⚠️ 合入阶段的「操作合入」功能即将支持，当前请前往 Bits 界面完成合入操作。
```

## M-Step 2: 获取合入详情数据

**并行执行**以下三个命令：

```bash
# 获取合入状态列表
bitscli devops dev-task process merge-info --dev-task-id <dev_task_id>

# 获取变更详情（含 deployConfigs，用于提取项目 + 控制面）
bitscli devops dev-task get <dev_task_id>

# 获取评审/CI 基础状态（用于“代码评审状态”“Codebase CI 状态”两列）
bitscli devops dev-task process review-basic-info --dev-task-id <dev_task_id>
```

从 `merge-info` 响应读取 `data.changeMergeInfoList` 和 `data.mergingDevTaskList`。

从 `dev-task get` 响应读取 `data.changes[]`，构建 **changeId → deployConfigs** 映射：遍历 `data.changes[]`，以 `changes[].id` 为 key，`changes[].deploy_configs[]` 为 value。每个 `deploy_config` 含 `project_info`（项目名、项目类型）和 `build_configs[]`（含 `control_panel` 控制面）。

从 `review-basic-info` 响应读取 `data.reviewBasicInfos[0].codeChangeReviewBasicInfos[]`，构建 **changeId → reviewInfo** 映射：以 `codeChangeReviewBasicInfos[].devBasicChangeId` 对齐 `changeMergeInfo.changeId`。

## M-Step 3: 展示合入队列（如有）

若 `mergingDevTaskList` 非空，先展示前置合入队列：

```text
📋 合入队列：当前有 {count} 个前置开发任务正在合入，合入后可继续合入当前变更

  - {title}  创建人: {author}  影响仓库: {relatedRepos用、分隔}
  - ...
```

若 `mergingDevTaskList` 为空，不输出此区块。

## M-Step 4: 展示 MergeTable

将 `changeMergeInfoList` 按前端逻辑拆行：对每条 `changeMergeInfo`，根据其 `changeId` 查找 `dev-task get` 返回的 `changes[]` 中对应 change 的 `deploy_configs[]`，按「项目（`project_info.type` + `project_info.name`）× 控制面（`build_configs[].control_panel`）」拆分为多行。

> 若某个 `changeId` 在 `changes[]` 中找不到（例如 change 已删除），则该 change 只输出一行，项目和控制面列显示 `-`。

**MergeTable 列定义与取值**：

| 列名 | 取值来源 | 说明 |
|---|---|---|
| 合码状态 | `changeMergeInfo.mergeStatus` | 转为中文（见下方枚举映射） |
| 仓库名称 | `changeMergeInfo.manifest.codeElement.repoPath` | 无则显示 `-` |
| 开发分支 → 目标分支 | `manifest.codeElement.sourceBranch` → `manifest.codeElement.targetBranch` | 若有 `lastestCommit.id`，附加 `(commit: <前8位>)` |
| 项目 | `deploy_config.project_info.name` | 前缀附加项目类型简称，如 `[TCE] xxx` |
| 控制面 | `build_config.control_panel` | 转为可读值（见下方映射） |
| 代码评审状态 | `reviewInfo.reviewSkipped ? 'skipped' : reviewInfo.status` | 展示图标 + 文案 + “查看”链接 |
| Codebase CI 状态 | `reviewInfo.ciCheckSkipped ? 'skipped' : reviewInfo.ciChckStatus` | 展示图标 + 文案 + “查看”链接 |

**合码状态（MergeStatus）中文映射**：

| 枚举值 | 中文 |
|---|---|
| pending (1) | 未开始 |
| merging (2) | 合入中 |
| merged (3) | 已合入 |
| queuing (4) | 队列中 |
| failed (5) | 合入失败 |
| merged_on_codebase (6) | 其他途径合入 |
| closed_on_codebase (7) | 已关闭 |

**控制面（control_panel）映射**：

| 值 | 展示 |
|---|---|
| 1 / `CONTROL_PLANE_CN` | CN |
| 2 / `CONTROL_PLANE_I18N` | I18N |
| 3 / `CONTROL_PLANE_EU_TTP` | EU_TTP |
| 4 / `CONTROL_PLANE_US_TTP` | US_TTP |
| 5 / `CONTROL_PLANE_I18N_BD` | I18N_BD |
| 其他 | 原值 |

**项目类型简称映射**：

| type 值 | 简称 |
|---|---|
| 1 / `PROJECT_TYPE_TCE` | TCE |
| 2 / `PROJECT_TYPE_FAAS` | FAAS |
| 3 / `PROJECT_TYPE_CRONJOB` | CRONJOB |
| 4 / `PROJECT_TYPE_WEB` | WEB |
| 6 / `PROJECT_TYPE_HYBRID` | HYBRID |
| 7 / `PROJECT_TYPE_CUSTOM` | CUSTOM |
| 其他 | 原值 |

**代码评审状态映射**（与前端一致）：

| 值 | 展示 |
|---|---|
| skipped | 跳过 |
| RUNNING | 待我 Review（如不区分 needReview，可统一展示“进行中”） |
| REJECTED | 已驳回 |
| APPROVED / ALWAYS_APPROVED | 已完成 |
| FETCHING / PENDING / 其他 | 未开始 |

**Codebase CI 状态映射**（与前端一致）：

| 值 | 展示 |
|---|---|
| skipped | 跳过 |
| running | 进行中 |
| failed | 失败 |
| succeeded | 成功 |
| exceptional | 异常 |
| unstarted | 未开始 |
| 其他 | 进行中 |

**“查看”链接规则**（与前端一致）：

- 代码评审状态列：跳转到变更详情页  
  `https://bits.bytedance.net/devops/{space_id}/develop/detail/{dev_task_id}/change/{change_id}?devops_space_type=server_fe`
- Codebase CI 状态列：跳转到变更 checks 页  
  `https://bits.bytedance.net/devops/{space_id}/develop/detail/{dev_task_id}/change/{change_id}/checks?devops_space_type=server_fe`
- 其中 `space_id` 来自 `dev-task get` 的 `data.basic_info.space_id`，`change_id` 为当前行对应的 `changeMergeInfo.changeId`。

**输出格式**（Markdown 表格）：

```text
📋 合入详情

| 合码状态 | 仓库名称 | 开发分支 → 目标分支 | 项目 | 控制面 | 代码评审状态 | Codebase CI 状态 |
|---|---|---|---|---|---|---|
| 已合入 | repo-a | feat/xxx → master (commit: a1b2c3d4) | [TCE] my-service | CN | 已完成 [查看](.../change/1001) | 成功 [查看](.../change/1001/checks) |
| 已合入 | ^ | ^ | [TCE] my-service | I18N | ^ | ^ |
| 合入中 | repo-b | feat/yyy → master (commit: e5f6g7h8) | [WEB] web-app | CN | 待我 Review [查看](.../change/1002) | 进行中 [查看](.../change/1002/checks) |
```

> - 同一个 change 拆出多行时，第 2 行起的「合码状态」「仓库名称」「开发分支 → 目标分支」列用 `^` 表示与上一行相同（等效于前端的 rowSpan 合并）。
> - 「代码评审状态」「Codebase CI 状态」同样是 change 维度的合并列；同一 change 的第 2 行起可用 `^`。
> - 若 `changeMergeInfoList` 为空，输出 `暂无合入信息`。
> - 若 `mergeStatus` 为 `failed`，在合码状态后附加失败原因：`合入失败: {errorMessageZh || errorMessageEn || description}`。
> - 若某条 change 缺失 `reviewInfo`，两列均显示 `-`（不展示“查看”）。

## M-Step 5: 附加失败变更摘要（如有）

遍历 `changeMergeInfoList`，若存在 `mergeStatus` 为 `failed(5)` 或 `closed_on_codebase(7)` 的条目，额外输出一个摘要：

```text
⚠️ 异常变更：
  - {repoPath}：合入失败 — {errorMessageZh || errorMessageEn}
  - {repoPath}：已关闭
```

若无异常变更，不输出此区块。

## M-Step 6: 输出界面链接

```text
🔗 前往 Bits 界面操作合入：https://bits.bytedance.net/devops/{space_id}/develop/detail/{dev_task_id}?devops_space_type=server_fe&stage=dev_code_merge_stage
```

> `space_id` 从 Step 1 的 `dev-task get` 返回的 `data.basic_info.space_id` 获取，或从用户输入的 URL 中提取。

**合入阶段特殊流程到此结束，不执行 Step 2/3/4。**
