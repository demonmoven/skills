# Gloop v0.2.24

## 修复：网络超时被误分类为 unknown，走 blocked 而非快速重试

### 现象

qst_2606253040 剑士阶段连续 6 次 "Request timed out"（上游 LLM API 超时），前 3 次触发 blocked，blocked 恢复后又 3 次超时才成功。每次超时跑 5-7 分钟，总卡顿 30+ 分钟。

### 根因

`classifyAgentError` 的 transient 匹配列表只认 `"timeout"`，不认 `"timed out"`。relay CLI 输出的 `"Request timed out"` 小写后是 `"request timed out"`，不包含 `"timeout"`，被分类为 `"unknown"`。

- `"unknown"` 走 `consecutiveErrors >= maxConsecutiveAgentErrors(3)` 路径 → 3 次直接 blocked
- `"transient"` 走 `retryableErrorStreak < maxRetryableAgentErrors(3)` 快速重试路径（带退避，3 次都失败才 blocked）

同一个语义的两种拼写（"timeout" vs "timed out"），导致网络超时走了重流程（blocked + cooldown 恢复），而非轻流程（秒级退避重试）。

### 修复

**`classifyAgentError`**：transient case 加 `"timed out"` 匹配。`"Request timed out"` → transient → 快速重试。

**`agentRetryDelay`**：线性 1s/2s/3s → 指数退避 2s/4s/8s。网络超时后上游可能需要 10-30 秒恢复，线性 1-3 秒太短会撞同样的超时；指数退避给上游恢复时间，总耗时 14 秒可接受。

### 为什么这是全局最优

- **语义对齐**："timed out" 和 "timeout" 是同一语义，都该走 transient 快速重试
- **轻流程优先**：瞬抖走秒级退避重试（14 秒内自愈），持续故障才走 blocked + cooldown（避免空耗 phase 预算）
- **指数退避**：给上游恢复时间，避免短间隔重试撞同样的超时
- **3 次重试不变**：瞬抖 3 次内通常恢复；3 次都超时说明上游持续故障，blocked 是对的（保护 phase 预算）

### 变更

- `internal/orchestrator/micro_loop.go`：
  - `classifyAgentError` transient case 加 `"timed out"` 匹配
  - `agentRetryDelay` 线性 → 指数退避（2s/4s/8s）
- `internal/orchestrator/agent_recovery_test.go`：加 "timed out" 测试用例
