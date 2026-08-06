# Gloop v0.2.13

## 补完预算迁移：max_consecutive_agent_errors

v0.2.12 把 `MaxConsecutiveAgentErrors` 代码默认值改成 3，但 `applyDefaults` 只在零值时填默认——用户 config.json 已显式设 2 的不会被动。本版补上迁移：

对 config version < 0.2.12 的旧配置，`MaxConsecutiveAgentErrors` < 3 的提升到 3。relay/aiden CLI 偶发 rate limit / transient 错误，2 次就 block 会让正常 quest 频繁失败，3 次容忍一次抖动。

v0.2.12 + v0.2.13 合起来完整覆盖三项预算调优：
- `max_turns_per_phase`: 6 → 20（v0.2.12 迁移）
- `max_duration_per_phase_ms`: 10min → 20min（v0.2.12 迁移）
- `max_consecutive_agent_errors`: 2 → 3（v0.2.13 迁移）
