# Gloop v0.2.9

## P0：竞态根治 — answer 后 warrior 不再失活

**根因**：`releaseQuestRuntime` 此前先 `flushCommentCursor` + `settleRunningRecoveryState`，最后才 `delete(e.running, qid)`。在这个窗口内若 `AppendUserAnswer` 进来，会发现 `e.running[qid]` 仍在但 runtime 已在收尾，answer 写进了一个即将被 cancel 的 runtime，warrior 永远不恢复 → 委托僵尸在 waiting_input。

**修复**：
- `releaseQuestRuntime`：把 `delete(e.running, qid)` 提到最前，收尾逻辑改用已持有的 `rt` 指针，不再回查 running map。
- `flushCommentCursor`：改签名直接接收 `rt *questRuntime`，不依赖 running map（releaseQuestRuntime 已先 delete）。

效果：answer 与 release 互斥窗口消除，无论谁先到都能正确落到稳定状态。

## P1：automation quest 短超时 — 僵尸 waiting_input 自动收敛

automation quest 无人 answer，原 24h 超时太长会僵尸。`RecoverWaitingInputTimeouts` 对 `SourceAutomation` quest 叠加 5min 短超时：

```
deadline = AskedAt + TimeoutMs（默认 24h）
if SourceAutomation: autoDeadline = AskedAt + 5min；取更早者
```

5min 足够内部循环感知恢复，又不会让自动化委托长时间占住看板。

## 前端：首页 failed/cancelled 默认折叠

终态历史默认折叠，让首页 focus 在需处理的 quest 上。点击分组头仍可展开查看。

## 评估为不必要（不做才是全局最优）

- **P2 warrior 提问质量前置校验**：易误杀正常提问，且 P1 的 5min 短超时已兜底自动化僵尸场景，收益不抵风险。
- **P3 统一 quest 存活检查**：合并检查路径是伪优化，现有 reserve/release + answer 互斥已覆盖，职责清晰不该合并。
