# Gloop v0.2.14

## 本地存储维护：四项清理落地

从第一性原理审视本地存储，修了四个无上限增长 / 残留不清理的问题。

### 1. scheduler 审计日志轮转

`scheduler/events.jsonl` 只追加不轮转，10 天累积 15M（16.8 万行），无上限增长。

- 新增 `fsstore.RotateJSONLIfBig(path, maxBytes, keep)`：超阈值时 rename 轮转，保留最近 N 份，超出删最老。
- scheduler tick 每分钟检查一次，超 10M 轮转，保留 1 份。

### 2. recovery_state 终态清理

recovery_state 只对 running/blocked quest 有意义，quest 进终态后残留成孤儿（实测 16 个残留，1 个纯孤儿 meta 已不存在）。

- 新增 `Root.DeleteRecoveryState(qid)`。
- scheduler tick 加 `cleanupStaleRecoveryStates`：扫所有 recovery_state，对应 quest 已终态（success/failed/cancelled）或 meta 不存在的删掉。

### 3. bin/versions 旧版本清理

18 个历史版本二进制 611M，无保留策略。

- 启动时 `PruneOldVersions`：按版本号降序保留最近 3 个（含当前版本），超出删除。
- 实测：18 → 4，清理 15 个旧版本。

### 4. cleanup_backup TTL

workspace 清理时的 backup 目录无 TTL，残留不消。

- 启动时 `CleanupOldBackups`：目录名末尾 unix 时间戳超 7 天的删除。
- 兼容 `cleanup_backup_1782143508` 和 `cleanup_backup_blocked_apply_failed_1782143776` 两种命名（正则提取末尾数字）。

### 不动的（已合理）

- worktree work/ 目录 16M/quest：git worktree add 的正常 checkout，共享 base repo .git 对象存储。retention=7 天自动清。
- per-quest events.jsonl / sessions/：随 quest 终态 + workspace retention 一起清。
