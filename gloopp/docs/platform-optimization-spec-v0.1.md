# gloop 平台优化 Spec — Recovery / Trace / Trust / Dashboard

> 版本：v0.1
> 状态：已实现（保守落地；v0.1.2 补充 workspace copy 稳定性）
> 相关文档：`refactor-spec-v1.md`、`hotl-spec-v0.1.md`
> 不包含：`design-phase-spec-v0.1.md` 已单独推进，本 spec 不重复定义 Design Phase
> 定位：专项优化 spec，收敛近期 review 后剩余的平台工程优化项

---

## 实现状态（2026-06-21）

本 spec 已按“先地基、后放权”的原则完成保守落地：

- **Phase 0：地基清理**
  已新增 `events.EvtQuestWaitingInput`，替换 waiting_input 字符串字面量；`RecoveryState` 已预置 `add_turns` / `add_duration_minutes` / `cooldown_seconds` / `max_attempts`；Automation Trust 已增加 `outcome_type` / `dedup_key` / `outcome_keys`，并通过 `AppendOutcome` 区分 stage outcome 与 terminal outcome。

- **Phase 1：Recovery Policy 参数化**
  `RecoveryPolicy.Decide` 已输出 retry 执行参数；Orchestrator 读取 decision 参数并写入 RecoveryState 快照；auto retry 不再硬编码预算，而是读取 state 参数；`max_attempts` 由 Orchestrator 基于 decision 参数 + state attempt 统一判断。

- **Phase 2：Recovery / Trust 事件化结算**
  Engine `publish` 后已触发 settlement hook，事件只是触发器，结算时重新读取 quest meta 作为事实源；recovery running 会根据当前 quest 状态结算为 `succeeded` / `failed` / `pending`；Automation Trust 也接入 quest success / failed / cancelled / apply_failed 事件，并依赖 terminal outcome 去重避免重复计数。

- **Phase 3：全轮次 Trace Contract**
  `/api/quests/{id}/trace` 已支持 `scope=current|all`、`round`、`limit`、`truncated`、`next_cursor` 响应字段；默认仍是 current scope，`scope=all` 用于 RCA / 调试。真实 cursor 翻页先保留为空，后续如数据量继续扩大再实现。

- **Phase 4：Notification 去重与恢复摘要**
  EventSubscriber 已支持内存级 5 分钟 dedup，按 `quest_id + event_type` 去重，不会让 blocked 吞掉 waiting_input；user_review 通知会在 recovery succeeded + retry 时附加恢复摘要。去重状态不持久化，重启后失效属于 v0.1 接受边界。

- **Phase 5：Dashboard 首屏减重**
  Dashboard 已做 tab-level lazy loading，低频页面拆成独立 chunks；`loadAll` 不再首屏拉 automations / skills / prompts，进入对应 tab 时再加载。主 JS 从约 580KB 降到约 267KB，Vite 大 chunk 警告消失。

- **Phase 6：测试日志收敛**
  Orchestrator 测试默认使用 silent logger；原先绕过 `e.log` 的标准库 `log.Printf` 已收敛到 engine logger；`waitForStatus` 失败时会 dump quest snapshot 和最近事件，保留失败诊断能力。

- **v0.1.2 workspace copy 稳定性补充**
  Workspace copy 已跳过系统目录 `.Trash` / `.Trashes` / `.Spotlight-V100` / `.fseventsd`，避免桌面/挂载目录里的系统垃圾目录权限异常导致 copy workspace 准备失败；已补 `TestWorkspaceCopySkipsSystemTrashDir` 回归。

刻意暂缓的部分：

- **Trust 自动降级策略**暂未真正放权执行。v0.1 只完成 outcome 去重、terminal/stage 分层和事件 settlement，为“只降不升”策略打地基；自动降级本身建议另起提交实现。
- **Trace cursor 翻页**暂未实现真实 cursor，仅保留响应字段。当前 `limit + scope + round` 已覆盖主要调试场景。
- **Notification dedup 持久化**暂未做。当前为内存级去重，服务重启后可能重复通知。

相关提交：

- `0d34e17 chore: prepare recovery and trust contracts`
- `e88b82b feat: parameterize recovery retry policy`
- `9f5bdb1 feat: settle recovery and trust from quest events`
- `b1318e5 fix: support trace all scope limits`
- `f8bb8a0 feat: dedupe action notifications`
- `b76dcff feat: lazy load dashboard tabs`

---

## 一、背景与动机

最近一轮 review 和修复已经解决了几类直接错误：

- quick / 单阶段 quest 的 orphan recovery 不再强制要求 mage
- recovery retry 不再在恢复启动后立刻标记 succeeded，而是在 runtime 释放时按 quest 结果结算
- recovery retry 增加保守预算余量，避免刚恢复就再次撞上 guardrail
- 返工轮次 artifact 读取不再硬编码 `warrior_0`

这些修复把最明显的生命周期 bug 拉回了安全线。但从平台长期演进看，还有几类问题需要继续收敛：

1. **Recovery 语义仍分散**
   现在 recovery state 结算发生在 runtime release 路径中。它比提前成功正确，但还不是最干净的事实模型。真正的事实源应该是 quest 状态事件，而不是某个 goroutine 的清理路径。

2. **Policy 决策缺少可执行参数**
   `RecoveryPolicy.Decide` 目前主要返回 action / reason。预算增量、冷却时间、最大尝试次数等执行参数还散在 orchestrator 中，审计时只能看到"做了 retry"，看不到"为什么加这些预算"。

3. **Trace / Evidence 更偏当前轮，不够适合 RCA**
   返工链路里，真正有价值的是 `warrior_0..N`、`mage_0..N` 的完整因果时间线。当前 UI/API 已经能展示部分证据，但还缺一个明确的全轮次 trace contract。

4. **Automation Trust Tier 只有地基，缺闭环策略**
   trust state、recent outcomes、history 已存在，但自动升降级还没有形成产品语义。尤其是自动升级风险高，应该从"只降不升"开始。

5. **通知策略能降噪，但还不够稳**
   当前只通知 blocked / user_review / waiting_input，方向正确。但同一 quest 的短时间重复通知、恢复后摘要、再次阻塞升级等场景还没有明确策略。

6. **Dashboard bundle 已到拆包线**
   `npm --prefix web run build` 通过，但 Vite 提示主 JS chunk 超过 500KB。随着 Stats / Knowledge / Settings / QuestDetail 继续变重，首屏加载会逐步恶化。

7. **测试日志过吵**
   `go test ./...` 通过，但 orchestrator 测试会输出大量 INFO/WARN。失败时有帮助，常规通过时会降低 CI 可读性。

本 spec 的目标是把这些问题从散点 TODO 收敛为可拆分、可验证的工程任务。

---

## 二、目标与非目标

### 2.1 目标

1. **Recovery 事件化结算**
   recovery outcome 由 quest 状态变化统一驱动，避免隐藏在 runtime cleanup 中。

2. **Policy 决策参数化**
   recovery policy 不只返回 action，还返回预算增量、冷却时间、最大尝试次数等可审计执行参数。

3. **全轮次 Trace Contract**
   建立覆盖所有 rework round 的 trace API / frontend 数据结构，用于 review、RCA、调试和用户解释。

4. **Automation Trust 降级闭环**
   先做保守闭环：连续失败或用户 reject 自动降级；升级必须人工确认。

5. **通知去重与摘要**
   减少重复打扰，同时在关键状态恢复后给出必要摘要。

6. **Dashboard 首屏减重**
   对非首屏重页面做 route-level lazy loading，降低初始 JS chunk。

7. **测试输出收敛**
   常规通过时减少测试日志，失败时仍能保留足够诊断信息。

### 2.2 非目标

- 不实现 Design Phase；相关内容继续走 `design-phase-spec-v0.1.md`
- 不改变剑士 / 法师两角色模型
- 不引入分布式事件队列
- 不让 trust tier 自动升级到更高权限
- 不改变 agent runtime 策略，例如 Relay / Claude / Codex 的内部优化参数
- 不把 Dashboard 做成全新前端架构；只做渐进式拆包和契约补强

---

## 三、设计原则

### 3.1 状态事件是事实源

如果一个状态可以从 quest 状态变化推导，就不要把它藏在某条执行路径里。Recovery outcome、trust outcome、通知摘要都应该优先消费状态事件。

### 3.2 Policy 只做可解释的决定

Policy 不执行副作用，但必须输出足够解释副作用的参数：

- 为什么 retry
- 最多 retry 几次
- 追加多少预算
- 是否需要冷却
- 什么情况下升级给人

这些参数要进入 `policy.decision` 的 payload，方便 UI 和审计读取。

### 3.3 只降不升，升级找人

Automation trust 可以自动降级，因为降级是收紧权限；自动升级会扩大权限，必须走人工确认。

### 3.4 首屏只加载首屏

Dashboard 首屏不应该为 Stats / Knowledge / Settings / QuestDetail 的全部逻辑买单。重页面和低频页面应该 lazy load。

### 3.5 测试输出服务于失败诊断

测试通过时日志越少越好；失败时能看到关键事件、状态变更和路径证据。

---

## 四、总体方案

### 4.1 优化域

| 域 | 当前问题 | 目标形态 |
|---|---|---|
| Recovery | outcome 结算依赖 runtime release | quest 状态事件统一结算 |
| Policy | action 与执行参数分散 | decision 携带执行参数 |
| Trace | 当前轮证据为主 | 全 rework round 时间线 |
| Trust | 有 state，无降级闭环 | 失败自动降级，升级人工确认 |
| Notification | 只做事件类型过滤 | 去重、摘要、再次阻塞升级 |
| Dashboard | 主 bundle 超 500KB | 非首屏 route lazy loading |
| Tests | orchestrator 日志过多 | 默认安静，失败可诊断 |

### 4.2 建议实施顺序

```
Phase 0: 地基清理
    ↓
Phase 1: Recovery Policy 参数化
    ↓
Phase 2: Recovery / Trust 事件化结算
    ↓
Phase 3: 全轮次 Trace Contract
    ↓
Phase 4: Notification 去重与摘要
    ↓
Phase 5: Dashboard 拆包
    ↓
Phase 6: 测试日志收敛
```

Recovery 和 Trust 影响平台安全边界，优先级高于 UI 性能。Dashboard 拆包价值明确，但不应该和 lifecycle / policy 改动混在一个提交里。

### 4.3 Phase 0 地基清理

在进入 Recovery / Trust 核心改造前，先做 1-2 个小提交，降低后续实现的混杂度：

1. **事件常量补齐**
   把 `quest.waiting_input` 从 `events.EventType("quest.waiting_input")` 字符串字面量提升为 `events.EvtQuestWaitingInput`。通知、server、orchestrator 统一引用常量。

2. **RecoveryState 参数字段预置**
   先给 `RecoveryState` 增加预算/尝试参数快照字段，但 Phase 0 不改变行为。Phase 1 再把这些字段接入 policy decision。

3. **Trust outcome 去重规则落地**
   明确 trust outcome 的唯一键和终态语义，先实现去重 helper，再做自动降级。否则 Phase 2/4 很容易把同一个 quest 记成多次失败。

Phase 0 的目标不是做新能力，而是把后续几期会反复踩的基础契约钉住。

---

## 五、详细设计

### 5.1 Recovery Policy 参数化

#### 5.1.1 数据结构

在 `policy.Decision` 或 recovery 专用 decision payload 中增加执行参数：

```go
type RecoveryActionParams struct {
    AddTurns           int    `json:"add_turns,omitempty"`
    AddDurationMinutes int    `json:"add_duration_minutes,omitempty"`
    CooldownSeconds    int    `json:"cooldown_seconds,omitempty"`
    MaxAttempts        int    `json:"max_attempts,omitempty"`
    EscalateAfter      int    `json:"escalate_after,omitempty"`
}
```

推荐先放入 `Decision.Conditions`，避免一次性扩大通用 Policy ABI：

```json
{
  "policy_name": "no_progress_retry",
  "action": "retry",
  "conditions": {
    "blocked_reason_code": "no_progress",
    "recovery_count": 0,
    "add_turns": 5,
    "add_duration_minutes": 15,
    "max_attempts": 1
  }
}
```

#### 5.1.2 参数归属

Recovery 的职责分两层：

| 层 | 负责什么 | 不负责什么 |
|---|---|---|
| Policy | 根据 facts 输出 action 与执行参数，例如 `max_attempts`、`add_turns`、`add_duration_minutes` | 不直接执行 retry，不修改 quest，不持久化状态 |
| Orchestrator | 读取 decision 参数 + recovery state attempt 计数，判断是否执行，并落盘执行快照 | 不再硬编码 retry 预算，不重复实现 policy 判断 |

推荐调整：`RecoveryPolicy.Decide` 不再用内部硬编码 `RecoveryCount >= 3` 直接拦截，而是输出 `max_attempts`。Orchestrator 根据 `state.AttemptIndex` 和 `decision.Conditions.max_attempts` 决定是否继续 retry；超过上限时发布 require-user 的 policy decision 并停止自动恢复。

这样避免两个真相源：

- 最大尝试次数只来自 decision 参数
- 当前尝试次数只来自 recovery state
- 是否执行由 orchestrator 统一判断

#### 5.1.3 RecoveryState 参数快照

RecoveryState 需要保存执行时使用的参数，避免将来默认值变化后无法解释历史行为：

```go
type RecoveryState struct {
    // existing fields...
    AddTurns           int `json:"add_turns,omitempty"`
    AddDurationMinutes int `json:"add_duration_minutes,omitempty"`
    CooldownSeconds    int `json:"cooldown_seconds,omitempty"`
    MaxAttempts        int `json:"max_attempts,omitempty"`
}
```

这些字段是**执行快照**，不是配置源。配置源是 policy decision。

#### 5.1.4 策略规则

| reason_code | action | add_turns | add_duration_minutes | max_attempts | 说明 |
|---|---:|---:|---:|---:|---|
| `agent_consecutive_errors` | retry | 5 | 15 | 1 | 短暂 agent 执行失败，保守重试一次 |
| `no_progress` | retry | 5 | 15 | 1 | 给 agent 一次额外推进机会 |
| `duration_exceeded` | require_user | 0 | 0 | 0 | 总时长超限需要人判断 |
| auth 类错误 | require_user | 0 | 0 | 0 | 登录/授权由人确认 |
| L2 / external side effect | require_user | 0 | 0 | 0 | 高风险副作用不自动恢复 |

#### 5.1.5 验收标准

- `policy.decision` 事件能看到 retry 的预算参数
- `autoRetryBlockedQuest` 不再硬编码预算参数，而是读取 decision / recovery state
- recovery state 记录执行参数快照，后续审计不依赖当前代码默认值
- 超过 `max_attempts` 时不再 retry，而是进入 require-user 路径
- 现有 `go test ./internal/policy ./internal/orchestrator` 通过

---

### 5.2 Recovery / Trust 事件化结算

#### 5.2.1 Recovery settlement

新增一个订阅器或 orchestrator 内部 handler，消费 quest 状态事件：

- `quest.user_review` / `quest.success`：如果 recovery state 为 running，则标记 `succeeded`
- `quest.failed` / `quest.cancelled`：如果 recovery state 为 running，则标记 `failed`
- `quest.blocked`：如果 recovery state 为 running 且再次 blocked，则标记 `failed` 或 `pending`，并记录 `LastError`

当前 runtime release settlement 可以保留为兼容兜底，但事件结算应成为主路径。

#### 5.2.2 竞态与幂等

事件化结算必须显式处理乱序和重复：

1. **终态幂等**
   如果 recovery state 已是 `succeeded` / `failed`，任何后续事件都忽略。终态不可回滚。

2. **按 quest 当前状态二次确认**
   事件到达时不要只相信事件 payload；必须重新读取 quest meta，以当前持久化状态为准。事件只是触发器，quest meta 是事实源。

3. **状态优先级**
   如果事件乱序，按 quest 当前状态结算：

   | 当前 quest 状态 | recovery running 的结算 |
   |---|---|
   | `success` / `user_review` | `succeeded` |
   | `failed` / `cancelled` | `failed` |
   | `blocked` | 见 5.2.3 |
   | `running` / `reviewing` / `waiting_input` | 保持 `running` |

4. **runtime release 兜底**
   runtime release settlement 暂时保留，处理事件订阅器未启动、测试路径或老数据迁移场景。主路径稳定后再考虑移除。

#### 5.2.3 再次 blocked 的明确语义

当 quest 在 recovery running 后再次进入 blocked：

| blocked reason | recovery state | 说明 |
|---|---|---|
| 与本次 recovery 原始 reason 相同 | `failed` | 同一问题恢复失败，停止自动恢复 |
| 高风险 reason，例如 auth / L2 / external side effect / duration_exceeded | `failed` | 必须交给人 |
| 新的 recoverable reason 且未超过 max_attempts | `pending` | 允许新的 recovery decision 重新评估 |
| 缺少 reason_code | `failed` | 无法解释，不自动继续 |

实现上优先使用 `BlockedReasonCode`，不要用自然语言 `BlockedReason` 做主判断。

#### 5.2.4 Orphan recovery 路径

`RecoverOrphanedQuests` 仍然通过 `launchRunningLoop` 进入正常 runtime，因此成功恢复后会触发同一套事件结算。对于无法启动 runtime、直接被 `markOrphanBlocked` 降级的 quest：

- 如果已有 recovery state 为 running，则结算为 `failed`
- 如果没有 recovery state，则只记录 blocked + policy.decision，不伪造 recovery outcome

这避免 orphan recovery 成为事件化 settlement 的旁路。

#### 5.2.5 Trust settlement

Automation trust outcome 同样应以状态事件为输入：

- human pass → success / independent
- human reject → failure / independent
- policy auto-pass → success / policy
- apply failed → failure / platform
- repeated blocked / recovery failed → failure / platform

#### 5.2.6 验收标准

- 直接通过 `QuestService` 改变状态也能结算 recovery / trust
- runtime release 不再是唯一结算路径
- 新增测试覆盖非 runtime 路径状态变化
- trust outcome 不重复记录同一 quest 的同一终态
- 乱序事件不会把 succeeded/failed recovery state 回滚
- 再次 blocked 的同因/异因路径都有测试覆盖

---

### 5.3 全轮次 Trace Contract

#### 5.3.1 API

新增或扩展现有 trace API：

```
GET /api/quests/{id}/trace?scope=all
GET /api/quests/{id}/trace?scope=current
```

默认建议保持当前行为，`scope=all` 返回所有阶段轮次。

#### 5.3.2 数据契约

```ts
type QuestTraceScope = 'current' | 'all'

interface QuestTraceEntry {
  id?: number
  kind: 'event' | 'message' | 'tool' | 'evidence' | 'comment' | 'context_pack'
  ts: number
  sid?: string
  phase?: number
  round?: number
  type?: string
  payload?: unknown
  content?: string
  tool_name?: string
  tool_args?: string
  status?: string
  result?: string
  meta?: unknown
}
```

`round` 从 session id 中解析：

- `warrior_0` → round 0
- `mage_0` → round 0
- `warrior_1` → round 1
- `mage_1` → round 1

#### 5.3.3 分页与数据量

`scope=all` 不能简单把每个 session 的 200 行拼起来后一次性返回。返工多轮时数据会快速膨胀。

建议 API 支持：

```
GET /api/quests/{id}/trace?scope=all&limit=200&cursor=...
GET /api/quests/{id}/trace?scope=all&round=2&limit=200
```

语义：

- `limit` 限制返回 entry 数，不是每个 session 的行数
- `cursor` 使用稳定的 `(ts, source, id)` 或后端 opaque cursor
- `round` 可选，用于只看某轮
- 默认 `scope=current`，避免 QuestDetail 首次打开就加载全量历史
- `scope=all` 定位为调试 / RCA，不保证一次返回所有原始日志

服务端需要返回：

```json
{
  "items": [],
  "next_cursor": "...",
  "truncated": true
}
```

#### 5.3.4 UI

QuestDetail 的 evidence / trace 视图增加：

- All rounds / Current round 切换
- 按 round 分组
- 每个 round 展示 Warrior / Mage 两列或两段
- 保留 native tool evidence 的折叠详情
- 大日志分页加载，不在首屏展开全部 raw output

#### 5.3.5 验收标准

- 返工 N 次的 quest 能看到 0..N 所有 warrior/mage sessions
- trace entries 按 timestamp 稳定排序
- 大输出默认折叠，UI 不被长日志撑爆
- API 测试覆盖 `scope=all`
- 分页 / limit 测试覆盖多轮 session
- 前端 build 通过

---

### 5.4 Automation Trust 降级闭环

#### 5.4.1 Outcome 语义

Trust outcome 分两层记录：

1. **Verification stage outcome**
   记录每个阶段事实，例如 `policy_auto_pass`、`apply_failed`、`human_approved`。这些可以多条存在，用于审计。

2. **Quest terminal trust outcome**
   每个 automation quest 只产生一条终态 trust outcome，用于连续成功/失败计数和降级判断。

自动降级只能基于 terminal trust outcome，不能基于所有 stage outcome。否则一个 quest 的 `apply_failed` + `human_rejected` 可能把失败计数加两次。

#### 5.4.2 去重键

推荐新增结构：

```go
type AutomationTrustOutcome struct {
    QuestID          string `json:"quest_id"`
    OutcomeType      string `json:"outcome_type"` // terminal | stage
    Outcome          string `json:"outcome"`      // success | failure
    VerificationType string `json:"verification_type,omitempty"`
    Independent      bool   `json:"independent"`
    DedupKey         string `json:"dedup_key"`
    TsMs             int64  `json:"ts_ms"`
}
```

去重规则：

| 用途 | dedup key |
|---|---|
| terminal outcome | `quest_id + \":terminal\"` |
| stage outcome | `quest_id + \":stage:\" + verification_type` |
| 降级 history | `quest_id + \":downgrade:\" + reason` |

`ConsecutiveFailures` / `ConsecutiveIndependentSuccess` 只由 terminal outcome 更新。

#### 5.4.3 策略

先做只降不升：

| 条件 | 动作 |
|---|---|
| human reject | 降一级 |
| apply failed | 降一级 |
| 连续 2 次 blocked | 降一级 |
| 连续 2 次 recovery failed | 降一级 |
| automation 被人工锁定 | 不自动变更 |

升级只允许人工操作：

- UI 显示建议："连续 N 次 independent success，可考虑升级"
- 用户点击确认后调用 lock/update API
- history 写入 `manual_upgrade`

#### 5.4.4 验收标准

- locked automation 不会被自动降级
- 降级写入 history，包含 from / to / reason / quest_id
- UI 能显示最近降级原因
- 单个 quest 不重复触发多次降级
- 一个 quest 的多个 stage outcome 不会重复增加连续失败计数

---

### 5.5 Notification 去重与摘要

#### 5.5.1 去重窗口

同一 quest + 同一 action-required 类型，在短窗口内只发一次：

```go
type NotificationDedupKey struct {
    QuestID string
    Type    events.EventType
}
```

建议默认窗口：5 分钟。

#### 5.5.2 去重状态存储

v0.1 先采用内存级去重，接受服务重启后去重窗口丢失。理由：

- Gloop 是本地单用户 runtime，不引入额外持久化复杂度
- 通知重复的损害可控，且比漏通知更可接受
- 持久化去重需要清理策略、时间索引和迁移，先不扩大 scope

边界必须写清楚：重启后 5 分钟内可能重复发送 action-required 通知。若后续用户反馈明显，再升级为 workspace 下的轻量 JSONL / state 文件。

#### 5.5.3 恢复摘要

当 blocked quest 被恢复并进入 `user_review`，可以发一条摘要：

- 原 blocked reason
- recovery policy
- 本次恢复是否成功
- 当前需要用户做什么

#### 5.5.4 再次阻塞升级

如果同一 quest 在 recovery 后再次 blocked：

- severity 从 `action_required` 升级为 `needs_attention`
- 通知内容说明"自动恢复后再次阻塞"

#### 5.5.5 验收标准

- 同一 quest 短时间重复 blocked 不刷屏
- waiting_input 不被 blocked 去重误伤
- 恢复摘要只在状态实际推进后发送
- notifier 为 noop 时无副作用
- 测试明确覆盖服务重启后去重状态丢失的可接受边界

---

### 5.6 Dashboard Tab-Level Lazy Loading

#### 5.6.1 拆分对象

当前 Dashboard 是自定义 tab-based routing，不是 React Router。因此这里的准确目标是 **tab-level lazy loading**。

优先 lazy load 页面组件：

- `Stats`
- `Knowledge`
- `Settings`
- `QuestDetail`
- `Automations`

保留首屏必要模块同步加载：

- Shell / Sidebar / Topbar
- Quest list summary
- API client / routing hooks

#### 5.6.2 数据加载下沉

只拆组件不够。`useDashboardData` 当前在 App 顶层加载大量数据，低频页面即使 lazy load 组件，也可能仍在首屏请求数据。

需要配套拆数据：

| 数据 | 建议归属 |
|---|---|
| quests / inbox 轻量列表 | App / Board 首屏保留 |
| stats | `Stats` 页面内加载 |
| knowledge/context | `Knowledge` 页面内加载 |
| settings/providers/prompts/skills | 对应页面内加载或按需加载 |
| quest trace/evidence | `QuestDetail` 打开后加载 |

兼容策略：可以先把 `useDashboardData` 拆为内部 feature hooks，保持外部 API 兼容，再逐步下沉到页面。

#### 5.6.3 约束

- 不改变 URL routing 行为
- loading state 使用现有视觉系统
- 不引入新的路由库
- 不把页面拆成过度细碎的 chunks

#### 5.6.4 验收标准

- `npm --prefix web run build` 通过
- 主 JS chunk 明显低于当前 565KB
- App 首屏不再请求 Stats / Knowledge / Settings 的重数据
- 页面切换 loading 不闪烁、不撑布局
- mobile viewport 下无重叠

---

### 5.7 测试日志收敛

#### 5.7.1 方案

测试默认使用 silent logger，优先覆盖 `setupTestEngine` 注入到 Engine 的 logger：

```go
logger := slog.New(slog.NewTextHandler(io.Discard, nil))
```

对需要诊断的测试，保留可选 event dump helper：

```go
dumpQuestEventsOnFailure(t, root, qid)
```

注意事项：

- 先确认测试没有依赖日志文本做断言
- 只改测试 logger，不改生产 logger
- 如果 fsstore / policy 未来出现独立 logger，再分别提供注入点；当前不为降噪而扩大生产接口

#### 5.7.2 验收标准

- `go test ./internal/orchestrator` 通过时输出显著减少
- 失败时仍能打印对应 quest 的 events / session rows 摘要
- 不影响生产 logger

---

## 六、实施计划

> 当前状态：Phase 0-6 主体已完成。下方 checklist 已按实现回填；未勾选项是明确暂缓项，不代表当前实现缺口。

### Phase 0: 地基清理

- [x] `quest.waiting_input` 提升为 `events.EvtQuestWaitingInput`
- [x] `RecoveryState` 增加预算参数快照字段，暂不改变行为
- [x] 定义并实现 trust outcome dedup helper
- [x] 补事件常量和 trust 去重单测

### Phase 1: Recovery Policy 参数化

- [x] 在 policy decision conditions 中加入 retry 执行参数
- [x] RecoveryState 保存参数快照
- [x] auto retry 从 state / decision 中读取预算参数
- [x] max attempts 由 decision 参数 + state attempt index 统一判断
- [x] 补 policy + orchestrator 测试

### Phase 2: Recovery / Trust 事件化结算

- [x] 抽出 recovery settlement helper
- [x] 接入 quest 状态事件
- [x] trust outcome 去重
- [x] 处理乱序事件、终态幂等、再次 blocked 语义
- [x] orphan recovery blocked 旁路接入 settlement
- [x] 补直接状态变更路径测试

### Phase 3: 全轮次 Trace

- [x] API 支持 `scope=all`
- [x] 后端解析 session round
- [x] 支持 limit / round 参数；`next_cursor` 字段已保留
- [x] 前端 trace adapter 支持 round
- [x] QuestDetail 增加 all/current 切换
- [ ] 真实 cursor 翻页暂未做，当前仅保留响应字段

### Phase 4: Notification

- [x] 加去重缓存
- [x] 明确 v0.1 去重为内存级
- [x] 加恢复摘要通知
- [ ] 再次阻塞升级暂未做
- [x] 补 notifier noop / fake 测试

### Phase 5: Dashboard 拆包

- [x] tab-level lazy load 重页面
- [x] 将低频页面数据加载从 App 顶层下沉
- [x] 增加稳定 loading surface
- [x] build 后检查 chunk size
- [x] smoke 检查关键页面

### Phase 6: 测试日志

- [x] setupTestEngine 使用 silent logger
- [x] 失败 dump helper
- [x] 清理无意义测试输出

---

## 七、验证策略

每个 phase 至少运行：

```bash
go test ./...
npm --prefix web run build
```

涉及前端 UI 的 phase 还需要：

```bash
npm --prefix web exec tsc -- --noEmit
```

涉及 lifecycle / recovery 的 phase 重点测试：

```bash
go test ./internal/orchestrator -run 'Recovery|Blocked|Trust|Trace|Orphan'
go test ./internal/policy
```

涉及 notification 的 phase 重点测试：

```bash
go test ./internal/notifications ./internal/orchestrator
```

---

## 八、边界

### Always

- 保持 backend DDD 依赖方向：`cmd -> server/cli -> orchestrator -> fsstore/domain/model`
- 所有状态变更继续走 QuestService 或明确的 orchestrator 应用服务边界
- 自动化权限默认收紧，不做自动升权
- 所有 policy 自动决策必须产生可审计事件

### Ask First

- 改变 quest 状态机合法迁移
- 改变 automation trust tier 默认值
- 引入新依赖或路由库
- 改变 agent runtime 启动参数

### Never

- 不在 UI 里推断安全策略绕过 backend
- 不把 L2 / external side effect 纳入自动恢复
- 不用 mock agent 作为业务路径默认选项
- 不为了降噪吞掉失败诊断信息

---

## 九、开放问题

| 问题 | 推荐倾向 | 理由 |
|---|---|---|
| Recovery retry 预算固定还是按 intensity 分级？ | v0.1 固定 5 turns / 15 min，后续再分级 | 固定值更容易审计；分级会扩大策略面 |
| `max_attempts` 谁判断？ | Orchestrator 按 decision 参数 + state attempt 判断 | 避免 policy 和 orchestrator 双真相源 |
| 再次 blocked 后 recovery 是 failed 还是 pending？ | 同因 failed，异因且 recoverable 才 pending | 同一问题反复出现说明自动恢复失败 |
| Trust outcome 一个 quest 记几条？ | stage 可多条，terminal 只一条 | 审计和计数分离，避免重复降级 |
| Trust 降级是否影响 `auto_start`？ | v0.1 先只影响 `allow_hotl_auto_pass` / tier，不自动关 auto_start | 降级先收紧自动通过，不直接打断自动化运行 |
| Notification 去重是否持久化？ | v0.1 内存级 | 重启重复通知可接受，先不加持久化复杂度 |
| Trace `scope=all` 是否默认启用？ | 默认 current，QuestDetail 显式切 all | 控制首屏和 API 数据量 |
| Dashboard bundle budget 设多少？ | 主 chunk 先压到 < 450KB，再评估 < 400KB | 先用低风险拆包获得确定收益 |

---

## 十、建议下一步

v0.1 的平台优化主线已经完成保守落地。后续建议单独开三个小规格或任务推进：

1. **Automation Trust 自动降级策略**
   基于当前 terminal outcome 去重地基，实现“只降不升”：human reject / apply failed / 连续 blocked / recovery failed 触发降级，locked automation 不自动变更。

2. **Trace 真分页**
   如果实际 quest trace 数据继续增长，再把 `next_cursor` 从保留字段升级为真实 cursor。优先保持当前 `scope + round + limit` 的简单模型。

3. **Notification 持久化去重**
   只有当重启后重复通知真的造成明显干扰时再做。当前内存级去重足够符合本地单用户 runtime 的复杂度边界。

这些都不应该和 Design Phase 或 Dashboard 视觉优化混在一个提交里。
