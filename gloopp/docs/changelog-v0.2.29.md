# Gloop v0.2.29

## 优化：workspace 生命周期管理 — 终态自动清理 + 孤儿兜底

### 现象

worktree 模式的 quest 在终态后 workspace 不回收：
- 52 个 worktree quest 留下 35 个 worktree 实体 + 67 个 gloop/* 分支
- 13G 空间被占用，base repo 的 git worktree list / git branch 被污染

### 根因

1. `CleanupOrphanWorktrees` 是死代码 — 定义了没接线
2. 孤儿判定只认"quest 不存在"，不认"quest 已终态"
3. 终态自动清理只覆盖 apply 成功路径，failed/cancelled/未 apply 的 success 不清
4. 分支清理只在显式 discard 时跑，终态自动路径不删分支

### 修复

**1. 终态自动 cleanup**

quest 进 success/failed/cancelled 时自动调 `Cleanup`（worktree remove + branch -D + 删 work 目录）。
覆盖所有 6 个终态入口：failQuest、completeReviewedQuest、auto-complete、StopQuest（排队取消 + 运行取消）、RejectInboxItem。
readonly 模式跳过（无独立 workspace）。幂等。

**2. 孤儿 worktree 兜底清理接线**

- daemon 启动时跑一次 `CleanupOrphanWorkspaces`
- scheduler 每小时（整点附近）跑一次

**3. 孤儿判定扩展**

从"quest 不存在"扩展到"quest 不存在 / quest 已终态 / worktree 路径不一致"。

**4. 分支独立清理**

新增 `cleanOrphanBranches`：扫描 `gloop/*` 分支，qid 不存在或已终态的分支直接 `git branch -D`。
独立于 worktree 实体 — 即使 worktree 目录已手动删但分支残留，也能清掉。

### 变更

- `internal/orchestrator/quest_lifecycle.go`：
  - 新增 `cleanupWorkspaceIfTerminal`：终态后清理 workspace
  - 新增 `CleanupOrphanWorkspaces`：Engine 层包装，供 server/scheduler 调用
  - `StopQuest` 两处 CancelQuest 后加 `cleanupWorkspaceIfTerminal`
- `internal/orchestrator/macro_loop.go`：
  - `failQuest` / `completeReviewedQuest` / auto-complete 路径加 `cleanupWorkspaceIfTerminal`
- `internal/orchestrator/automation.go`：
  - `RejectInboxItem` 加 `cleanupWorkspaceIfTerminal`
- `internal/orchestrator/scheduler.go`：
  - tick 里每小时调 `CleanupOrphanWorkspaces`
- `internal/server/server.go`：
  - `startBackgroundRecoveries` 加 `CleanupOrphanWorkspaces`
- `internal/fsstore/workspace_cleanup.go`：
  - `CleanupOrphanWorktrees` 末尾调 `cleanOrphanBranches`
  - 新增 `cleanOrphanBranches`：独立扫描清理孤儿分支
  - `findOrphanWorktreesInBase` 加"quest 已终态"判定
- `internal/model/model.go`：
  - 新增 `ImpactSummary` 类型（agent 主动声明的影响，供后续 HOTL 使用）
