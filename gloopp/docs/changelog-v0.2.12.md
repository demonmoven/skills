# Gloop v0.2.12

## 预算限制全局调优

从第一性原理重新审视并发与预算限制，结合实际 quest 数据（7 个 failed 里 5 个是 automation quest，phase 实际执行才 2-4 分钟就被切断）定位到根因：早期 config 把回合/时长设得过紧。

### 代码默认值调整

- `MaxConsecutiveAgentErrors`: 2 → 3。relay/aiden CLI 偶发 rate limit / transient 错误，2 次就 block 会让正常 quest 频繁失败。3 次容忍一次 transient 抖动，仍有 retryableErrorStreak 单独计数兜底真·连续失败。

### 配置迁移（config version < 0.2.12 自动生效）

早期 config 把 `max_turns_per_phase` / `max_duration_per_phase_ms` 设得过紧（6 回合 / 10 分钟），导致稍复杂的 phase（读几个文件 + 改码 + 跑测试）被频繁切断 failed。对 < 0.2.12 的旧配置，把低于合理下限的值向上提升：

- `MaxTurnsPerPhase`: < 20 的提升到 20（给复杂 phase 喘息空间，仍远低于代码默认 50 的安全网）
- `MaxDurationPerPhaseMs`: < 20min 的提升到 20min（匹配 20 turn 的合理执行时长）

已是合理值的旧配置不动；0.2.12+ 新配置用户显式设的值不覆盖（尊重用户意图）。

### 不动的（已合理）

- `MaxConcurrent` = 6（默认）/ 3（用户 config）：本地 CLI provider，每 quest 一 warrior 一 mage 串行，3-6 并发受单机资源 + API rate 限制，合理。
- `MaxTurnsPerPhase` 默认 50：技术安全网，主预算由 guardrails（时长/无进展/连续错误）决定。
- `MaxDurationPerPhaseMs` 默认 30min / `MaxDurationPerQuestMs` 默认 3h：上限合理。
- `MaxNoProgressTurns` = 3 / `MaxReworkPerQuest` = 2：合理。
