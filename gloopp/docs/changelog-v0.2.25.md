# Gloop v0.2.25

## 修复：domain round-trip 清空剑士产出物，委托完成后看不到交付物

### 现象

qst_2606253040 法师评审通过、auto-apply 成功后，meta.json 的 `outputs` 字段为空。
server.log 显示"剑士产出物已登记 count=9/7"，但最终 meta 里 outputs 字段不存在。
前端产物声明区为空，用户"看不到委托做了啥"。

### 根因

`Outputs`（剑士声明的交付物）只在 fsstore 层 `QuestMeta` 建模，domain 层
`quest.QuestMeta` 没有 `Outputs` 字段。`QuestMetaToDomain` / `QuestMetaFromDomain`
这对转换函数不映射 `Outputs`（以及 `Connectors`、`PromptVersion`、`GoalIterations` 等
fsstore 独有字段）。

触发链：
1. 剑士执行结束 → `extractWarriorDeliverables` → 直接走 fsstore `AddDeclaredOutput`
   写入磁盘 → meta.json 此时有 outputs（server.log 记录 count=9）
2. 法师评审通过 → `runAutoApplyProcessor` → `MoveToUserReview` → questService 走
   domain `repo.Save` → `QuestMetaFromDomain` 重建 fsstore meta（Outputs 为空）→
   `SaveQuest` 覆盖磁盘 → outputs 被清空
3. `CompleteAutoApplySuccess` 再次 round-trip，依然空

reviews 没丢是因为它存在单独的 `reviews.jsonl`，不经过 meta。

### 影响面

任何经过 domain 状态机的 quest（HOTL auto-apply / auto-complete / user_review 都走
domain），剑士声明的产出物都会被清空。同时 `Connectors`（HOTL connector）、
`PromptVersion`（prompt 快照）、`GoalIterations`（goal 模式迭代）等 fsstore 独有字段
也会被 round-trip 清空。

### 修复

**第一性原理**：domain 只管状态机字段，fsstore 独有的附属数据（Outputs/Connectors/
PromptVersion/Goal*/WorkspaceDowngrade）归 fsstore 直接路径管理。domain 状态机无权
修改也无权清空这些字段——round-trip 时应无条件保留磁盘已有值。

`QuestRepositoryAdapter.Save` 改为 merge 模式：`QuestMetaFromDomain` 重建后，先 Load
磁盘 meta，把 domain 不管的附属字段从磁盘值回填，再 `SaveQuest`。这样 domain 状态机
字段由 domain 主导，fsstore 附属字段由磁盘主导，互不干扰。

fsstore 直接路径（`AddDeclaredOutput` / `ClearOutputs` / `SaveOutputArtifact`）仍然
直接操作 `Outputs`，不受影响。

### 变更

- `internal/fsstore/quest_repository_adapter.go`：
  - `Save` 新增 `preserveFsstoreOnlyFields` merge 步骤
  - 保留 `Outputs` / `Connectors` / `PromptVersion` / `PromptOverrides` /
    `PromptOverrideMap` / `WorkspaceDowngrade` / `GoalIterations` /
    `GoalMaxIterations` / `GoalCondition`
- `internal/fsstore/outputs_test.go`：
  - 加 `TestAdapterSavePreservesFsstoreOnlyFields` 回归测试
