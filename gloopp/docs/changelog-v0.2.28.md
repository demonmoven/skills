# Gloop v0.2.28

## 优化：创建委托秒回 + copy 模式跳过 node_modules

### 现象

创建委托后需要等几秒（worktree 0.1-0.2s，copy 6-7s，gloop 自身仓库 worktree 30s），
HTTP 请求阻塞在 workspace 准备阶段。

### 根因

1. `createQuest` handler `auto_start=true` 时同步调 `StartQuest` → `launchQuest` →
   `Prepare`（git worktree add / copyDir），整个 workspace 准备在 HTTP 请求内完成
2. copy 模式全量复制工作区，只跳过 `.git`/`.gloop`，**node_modules 不跳过**——
   几百 MB 逐文件复制很慢

### 修复

**1. workspace 准备异步化**

`StartQuest` 改为：同步校验冒险者（mismatched class 等快速失败）→ reserveQuestRuntime →
`go launchQuestAsync()`。HTTP 立即返回，workspace 准备在 goroutine 里跑。

`launchQuestAsync`：
- Prepare 失败 → BlockQuest + SSE 通知
- adventurer 选失败 → BlockQuest + SSE 通知
- 成功 → SaveQuest → workspace.created SSE → StartQuest → runMacroLoop

前端通过 SSE 感知 workspace.created / quest.started，创建委托秒回。

`startNextQueued` 等已有 goroutine 的路径仍用同步 `launchQuest`（不阻塞 HTTP）。

**2. copy 模式跳过 node_modules**

`shouldSkipWorkspacePath` 硬编码跳过 `node_modules`（和 `.git`/`.gloop` 一样）。
agent 在 work 目录自己 `npm install`。

### 变更

- `internal/orchestrator/quest_lifecycle.go`：
  - `StartQuest` 异步化：同步校验 adventurer + `go launchQuestAsync`
  - 新增 `launchQuestAsync`：goroutine 内 Prepare + 启动，失败时 BlockQuest + SSE
  - `reserveQuestRuntime` 返回 nil 时幂等返回（不报错）
- `internal/fsstore/workspace_ignore.go`：
  - `shouldSkipWorkspacePath` 跳过 `node_modules`
- `internal/orchestrator/engine_e2e_test.go`：
  - inbox 测试适配异步：`waitForNotStatus` 等 status 离开 pending
  - 新增 `waitForNotStatus` helper
- `internal/orchestrator/scheduler_test.go`：
  - `TestConcurrency_DuplicateStartIgnored` 适配幂等语义
