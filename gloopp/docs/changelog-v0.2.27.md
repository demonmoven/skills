# Gloop v0.2.27

## 优化：transient 重试不重复记录 context_pack / 评论注入

### 现象

qst_2606258845 轨迹里连续三个完全相同的剑士上下文块，实际是 transient 超时重试
导致同一条 context_pack 消息被记录三次。

### 根因

transient 重试（`micro_loop.go`）把 msg 塞回 `pendingMsgs` 头部，下一 turn 重新
走完整的「turn_summary + context_pack + 评论注入」记录流程。重试语义是"把同一条
消息再发一次"，不是"注入新上下文"，但代码把重试当新 turn 处理，导致：
1. context_pack 重复记录（内容字节级相同）
2. turn_summary 重复记录（"turn 2 start" 实际是 turn 1 的重试）
3. 评论注入有重复推进 cursor 风险

### 修复（第一性原理）

transient 重试时给 msg 打 `_retry: true` 标记。所有注入类记录（context_pack /
user_comment / user_answer）在 `isRetry` 时跳过，不重复 append session row。
turn_summary 保留但标注 `retry: true`，让前端能区分真实 turn 和重试。

语义对齐：重试是"把同一条消息再发一次"，不产生新的上下文注入事件。

### 变更

- `internal/orchestrator/micro_loop.go`：
  - msg 取出后判断 `_retry` 标记
  - transient/rate_limit/session_lost 重试塞回时打 `_retry: true`
  - context_pack / user_comment / user_answer 记录跳过 retry msg
  - turn_summary 标注 `retry: true`
