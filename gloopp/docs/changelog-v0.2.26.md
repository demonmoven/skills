# Gloop v0.2.26

## 修复：CLI signal 提交时 assistant message 丢失 + auto-apply 成功路径 diff summary 丢失

### 现象

qst_2606253040 完成后：
1. 执行轨迹里看不到剑士/法师说了什么（"输出"tab 不出现）
2. 代码 diff 看不到（diff_stat 为空）

### 根因

**1. assistant message 丢失（系统性 bug）**

`micro_loop.go` 的 `consumePhaseSignal` 检测到 agent 通过 CLI signal（`gloop phase done`）
提交阶段结论时，直接 `return finalContent, sig, nil`——跳过了 line 452 的 assistant
message 记录。

agent 的 streaming 内容只做了 SSE 实时推送（`EvtTokenDelta`），从未持久化到 session
jsonl。所有通过 CLI signal 提交阶段结论的 quest 都会丢 assistant message。

**2. diff summary 丢失（系统性 bug）**

`runAutoApplyProcessor` / `runAutoCloseLoop` 在 apply 成功后直接
`cleanupAppliedWorkspace` 清理 work 目录，**从没计算 diff summary 存 meta**。
只在 apply 失败时才触发 `diff_summary` subscriber。

apply 成功 → work 目录被清 → meta 里 `diff_stat` 为空 → 前端无 diff 可展示。

### 修复

**assistant message**：
- stream goroutine 增加 `streamDelta` 累积器，边推送 SSE 边累积 delta
- `consumePhaseSignal` return 之前，用 `resp.Message.Content`（如果 SendMessage
  成功返回）或 `streamDelta.String()`（如果被信号中断，resp 为零值）补记
  assistant message
- 标记 `finish_reason: "phase_signal"` 和 `recovered: true`（从 stream 恢复）

**diff summary**：
- `runAutoApplyProcessor` / `runAutoCloseLoop` 在 `applyWorkspace` 之前同步调用
  `updateDiffSummary(qid)`，确保 work 目录被 cleanup 之前 diff 已存 meta

### 变更

- `internal/orchestrator/micro_loop.go`：
  - stream goroutine 增加 `streamDelta` 累积
  - `consumePhaseSignal` return 前补记 assistant message
- `internal/orchestrator/macro_loop.go`：
  - `runAutoApplyProcessor`：apply 前同步 `updateDiffSummary`
  - `runAutoCloseLoop`：apply 前同步 `updateDiffSummary`
- `web/src/theme.ts`：默认主题 obsidian → emerald
