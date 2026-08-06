# gloop HOTL 升级 Spec — 从人在环里到人在环上

> 版本：v0.1
> 状态：已实现（保守落地；v0.1.2 增加 quick no-effect auto-complete）
> 相关文档：`refactor-spec-v1.md`（架构重构主 spec）
> 定位：专项 spec，聚焦人机交互流程轻量化，是主 spec 的补充而非替代

---

## 实现状态（2026-06-21）

本 spec 已按“先地基、后放权”的原则完成保守落地。核心链路已经进入代码：

- **Policy Engine / ReviewPolicy**：已落地统一 facts、decision、`policy.decision` 审计事件，以及 context/none 类 automation allowlist 的保守 auto-pass。
- **Review + Apply 合并**：已支持 `verdict=pass + apply=true`，CLI/API 可一键通过并应用；apply 使用 `apply_status=pending` intent，服务启动会恢复 pending apply。
- **Waiting Input**：已新增 `waiting_input` 状态、`answers.jsonl`、`gloop quest answer` / `/answer` API；`quest_ask` 已从 blocked 改为 waiting_input，回答会注入下一轮 agent；超时支持 `continue/block/escalate`。
- **Recovery Policy**：已落地 `RecoveryState`、blocked `reason_code` 标准化、recovery `policy.decision` 审计，以及 `agent_consecutive_errors` / `no_progress` 的单次保守 auto-retry。
- **Notification Policy**：已作为 EventSubscriber 输入过滤层落地；默认只推 action-required（blocked / user_review / waiting_input），success/failed/progress 默认不打扰。
- **Automation Trust Tier**：已落地 trust state、recent outcomes、history、人工锁定，以及 human-approved / policy-auto-pass / apply 等 outcome 记录。

### v0.1.2 增量对齐（2026-06-22）

- **CompletionPolicy / quick no-effect auto-complete**：新增 `CompletionPolicy`，在 quick / 单阶段 execute 完成后、进入 `user_review` 之前做保守判定。只有 `allow_quick_auto_complete=true`、无 workspace diff、无 L2、无外部副作用、零返工时才直接 `CompleteExecute` 到 success。
- **独立审计字段**：auto-complete 使用 `auto_completed_by_policy` 与 `policy_action=auto_complete`，不复用 `auto_passed_by_policy`，避免把“无评审自动完成”误记成“评审自动通过”。
- **配置入口**：`allow_quick_auto_complete` 已接入 automation API、automation edit CLI、Dashboard automation 创建/编辑、`POST /api/quests` 和发起委托表单的一次性 override。
- **Review + Apply UX**：有 workspace diff 时默认主操作为“通过并应用”，同时提供“通过但不应用”；安全拦截时提示“评审已通过，应用被安全检查拦截”并允许 force apply。

刻意暂缓的部分：

- `tier_2+ verified` 暂不替代 allowlist 参与 auto-pass。原因是自动升权必须建立在更完整的独立验证和抽样审计上，不能让 auto-pass 的 success 自我强化。
- 通知合并窗口 / 静默窗口暂未做，只先完成策略过滤层。
- `skip_phase` 暂未开放。默认禁用仍是正确边界。
- 低风险 diff auto-pass 仍未做，继续作为独立安全 RFC；本 spec 不移动 workspace diff 必须人审的硬护栏。
- `user_review` 超时策略化未做，继续 data-driven follow-up。

相关提交：`778af14`, `bff49e3`, `77e4083`, `98725b5`, `0d768ba`, `ef87a10`, `18bab34`, `d59d90e`, `1432376`, `498ad20`, `1d89fa5`。

## 一、背景与动机

### 1.1 现状

gloop 当前的人机交互模式是 **HITL（Human-in-the-Loop）**：人深度嵌在循环里，每个 quest 至少需要一次人工决策才能收尾。具体表现为：

1. **User Review 是默认必经之路** — 法师评审通过后，无论 quest 类型、风险高低，都必须等用户点 pass/reject 才能进入终态。
2. **Blocked 是"垃圾桶状态"** — 超时、无进展、连续错误、认证失败、剑士提问… 所有异常全进 blocked，统一等人 triage。
3. **评审与 Apply 是两步操作** — 用户先 pass 进入 success，再手动 apply 改动，对于 coding quest 两步决策高度冗余。
4. **Automation 默认全量确认** — 所有自动化创建的 quest 默认进 inbox 等人确认，即使跑了几百次都成功的自动化也不例外。
5. **通知噪音大** — 事件驱动的通知粒度太细，人被频繁 ping 但大部分不需要行动。

### 1.2 问题

- **人的时间是最贵的资源**，却被消耗在大量低价值的形式主义确认上
- **不符合 Loop Engineering 理念** — Loop Engineering 的核心是人设方向、机器跑循环，而不是每个循环都要人盖章
- **自动化价值打折扣** — 自动化 quest 仍然需要人点 start / 点 pass，只自动化了"执行"没自动化"决策"
- **体验断点多** — 一个 quest 可能在 inbox / blocked / user_review 之间多次跳转，人要反复介入

### 1.3 目标

从 **HITL（Human-in-the-Loop）** 升级为 **HOTL（Human-on-the-Loop）**：

- 人从"每个循环都要确认"变成"只设规则 + 处理异常"
- 大部分 quest 自动闭环，人只在置信度低、风险高、出问题时介入
- 介入方式从"点按钮"变成"调策略"

---

## 二、目标与非目标

### 2.1 目标

1. **默认路径去人工化**：低风险、高置信度的 quest 自动闭环（不进 user_review）
2. **异常自动化降级**：大部分异常场景自动重试/降级，不直接进 blocked
3. **操作合并**：减少单次介入的操作步骤（review+apply 合并）
4. **信任积累机制**：自动化任务按历史表现动态调整信任级别，越跑越不用管
5. **通知降噪**：只推送需要人决策的通知，通知里直接带操作入口

### 2.2 非目标

- ❌ 不引入 AI agent 替代人做最终决策（法师是 AI 评审，但法师评审通过的低风险 quest 直接过，不等于 AI 替代人）
- ❌ 不做"全自动无人值守" — 人仍然是最高裁决者，只是从"每轮都盖"变成"随时可以介入"
- ❌ 不改变 gloop dumb runtime 的定位 — 所有策略都是配置驱动的规则，没有智能
- ❌ 不引入新的 agent 角色类型（比如 QA 角色、产品经理角色）。剑士/法师是阶段配置差异，不是两套 agent 架构
- ❌ 不单独拆分剑士/法师为两套独立系统 — 二者共享同一执行引擎，差异仅在 PhaseDef 配置
- ❌ 不做跨 quest 的 workflow 编排（那是 Phase 4 的事）

---

## 三、设计原则

### 3.1 默认安全（Safe by Default）

- 权限宁紧勿松：新增的自动跳过 user_review 等能力默认关闭，需要显式配置开启
- 高风险操作永远要人确认：涉及数据删除、权限变更、生产环境部署等场景，禁止自动闭环
- 人随时可以接管：任何自动路径都保留人工介入的入口

### 3.2 策略驱动，非硬编码

- 所有"什么时候自动 / 什么时候要人"的决策，都通过 policy 配置表达
- policy 可以按 quest type / intensity / source / workspace 等维度组合
- 不用改代码，改配置就能调整人机边界

### 3.3 渐进式放权

信任不是一次性给的，是逐步积累的：

```
新 automation → inbox 确认 → 连续成功 N 次 → auto_start → 再成功 N 次 → 跳过 user_review → …
```

失败则回退到上一级。类似 CI/CD 的渐进式发布理念。

### 3.4 可观测性优先

自动路径必须留痕：
- 为什么跳过了 user_review（命中了哪条 policy）
- 为什么自动重试了（触发了什么恢复策略）
- 自动化决策的 audit log 完整可追溯

出了问题能快速回答"为什么机器做了这个决定"。

---

## 四、总体架构

### 4.1 HOTL 三层模型

```
┌─────────────────────────────────────────┐
│            人 (Human on the Loop)       │
│  设策略 / 审异常 / 定方向 / 随时接管      │
└───────────▲──────────────▲──────────────┘
            │              │
            │ 异常/低置信   │ 策略配置
            │              │
┌───────────┴──────────────┴──────────────┐
│           Policy Engine (策略层)         │
│  review_policy / recovery_policy        │
│  trust_tiers / notification_policy      │
└───────────▲──────────────▲──────────────┘
            │              │
            │ 规则匹配      │ 状态事件
            │              │
┌───────────┴──────────────┴──────────────┐
│         State Machine (状态机层)         │
│  pending / running / reviewing /        │
│  waiting_input / blocked / user_review  │
└─────────────────────────────────────────┘
```

核心变化：在状态机之上加一层 **Policy Engine**，由它决定"这件事要不要人管"。状态机只负责"状态能怎么迁"，Policy 负责"什么时候迁、迁到哪"。

### 4.2 人机边界重定义

| 维度 | HITL（现状） | HOTL（目标） |
|------|-------------|-------------|
| 人的角色 | 每个 loop 都审批 | 设规则 + 处理异常 + 随时接管 |
| User Review | 每个 quest 必经 | 只有高风险/低置信度才进 |
| Blocked | 异常直接进等人处理 | 先自动重试/降级，不行再进 |
| Automation | 默认要人确认才能跑 | 按信任级别动态调整 |
| 通知 | 事件全量推 | 只推需要行动的，带操作入口 |
| 介入次数 | 每个 quest 2-3 次 | 平均 0.x 次 |

### 4.3 与剑士/法师两角色的关系

HOTL 升级**不改变**两角色的分工，也不把它们拆成两套独立系统。明确认知：

- **剑士 ≠ 一种 agent，法师 ≠ 另一种 agent** — 二者共享同一套执行引擎（`runPhase`）、同一套状态机、同一套存储结构。差异仅在于 `PhaseDef` 配置（工具权限、读写模式、结束信号、上下文内容、hint 策略）。
- **制检分离是 gloop 的核心特色，不弱化** — 法师是 AI Checker，人是最终裁决者。HOTL 调节的是"人什么时候介入"，不是"要不要法师"。
- **HOTL 横跨两个阶段** — Recovery Policy 同时作用于剑士阶段和法师阶段（超时、无进展等异常都可以自动恢复）；Review Policy 作用于法师评审之后（决定要不要进 user_review）。
- **角色是阶段的属性，不是人的属性** — 同一个 adventurer 配置可以在不同阶段扮演不同角色。未来加 QA 阶段、安全审计阶段，也只是往 pipeline 里加 PhaseDef，不需要改架构。

一句话：**两角色 = 管道化的两个阶段实例**，HOTL 在管道之上加策略层，不改变管道本身。

### 4.4 与主 spec 的关系

本 spec 是 `refactor-spec-v1.md` 的补充专项：
- 依赖 **Phase 2（宏循环管道化）** — policy 需要挂在 PhaseDef / QuestDef 上
- 可以在 Phase 2 之后作为 **Phase 2.5** 实施
- 不改变主 spec 的架构方向，是在其基础上的"人机流程轻量化"
- 部分小改动（如 design quest 跳过 user_review）可以提前到 Phase 1 验证

---

## 五、详细设计

### 5.0 Policy Engine 通用契约

所有 HOTL 策略（Review Policy / Recovery Policy / Notification Policy / Trust Tiers）共享同一套 Policy Engine 契约。这是 HOTL 的基础设施，先有契约再有策略。

#### 5.0.1 核心数据结构

```go
// PolicyDecision 是策略引擎的统一输出
type PolicyDecision struct {
    PolicyName  string         // 命中的策略名称
    Action      string         // 决策动作：auto_pass / require_user / retry / skip_phase / ...
    Reason      string         // 人类可读的决策原因
    Conditions  map[string]any // 命中的条件（用于 audit）

    // 审计字段
    DecidedAt    time.Time // 决策时间
    DecisionID   string    // 决策唯一 ID（用于幂等/追踪）
    InputHash    string    // 输入事实的 hash，用于校验决策可重现
    SafeByDefault bool     // 是否走了安全兜底路径
}

// PolicyAuditEvent 是每次决策的审计事件
type PolicyAuditEvent struct {
    QuestID    string
    PhaseIdx   int
    Decision   PolicyDecision
    InputFacts map[string]any // 决策时的输入事实快照
}
```

##### 事件类型与持久化

每次 policy 决策统一发布 **`policy.decision`** 事件，写入两个地方：

1. **Quest 级**：append 到 `workspace/quests/<qid>/events.jsonl`（每个 quest 自己的事件流）
2. **全局级**：append 到 `workspace/events/global.jsonl`（全局事件流，event_type = policy.decision，和现有全局事件共用同一套 cursor / 查询 / 审计工具）

事件 payload 固定字段：
```json
{
  "event_type": "policy.decision",
  "decision_id": "pol_dec_xxx",
  "quest_id": "q_xxx",
  "phase_idx": 1,
  "policy_name": "context_store_automation_auto_pass",
  "policy_action": "auto_pass",
  "input_hash": "sha256:abc123...",
  "safe_by_default": false,
  "decided_at": "2026-06-20T10:00:00Z"
}
```

> 设计原则：append-only、幂等（decision_id 去重）、可重放（input_hash 校验）。
> 和现有 event log / workflow cursor 机制对齐，不搞特殊存储。

#### 5.0.2 输入事实（Input Facts）

所有 policy 都从同一组事实中读取，不直接读 quest 对象。事实层是 policy 和状态机之间的边界。

```go
type PolicyInputFacts struct {
    // Quest 基础事实
    QuestID      string
    QuestType    string // code / design / research / ...
    Intensity    string // quick / standard / deep
    Source       string // manual / automation / fanout
    WorkspacePath string

    // 副作用事实（核心判断依据）
    EffectType        string // none / workspace_diff / context_store / external_side_effect
    HasWorkspaceDiff  bool   // 是否有待应用的工作区改动
    SideEffectLevel   string // none / low / medium / high
    HasExternalOutput bool   // 是否有外部输出
    AllowL2           bool   // 是否启用 L2 高权限

    // 阶段事实
    CurrentPhaseIdx   int
    CurrentPhaseType  string // execute / review
    PhaseReadOnly     bool
    PhaseEndSignal    string

    // Automation 事实
    AutomationID     string
    AutomationTier   string // tier_0 / tier_1 / tier_2 / tier_3
    AutomationTags   []string
    IsOfficialAutomation bool

    // 质量事实
    ReviewVerdict    string // pass / request_changes / reject
    ReviewScore      int    // 1-10
    HasAutoEvidence  bool   // 是否有自动证据注入
    EvidencePassed   bool   // 自动证据是否通过

    // 历史事实
    ReworkCount      int
    RecoveryCount    int    // 已尝试的恢复次数
    BlockedReason    string // 如果是 blocked 状态
}
```

#### 5.0.2.1 PolicyFactBuilder

Policy 不直接读 quest 对象，也不直接读 adventurer、automation config。所有事实统一由 `PolicyFactBuilder` 构建。

```go
// PolicyFactStore 是 FactBuilder 依赖的数据源抽象，
// 不直接传路径字符串，避免实现层到处拼路径。
// 具体实现可以包装 fsstore.QuestStore / fsstore.Root。
type PolicyFactStore interface {
    GetQuestMeta(questID string) (*fsstore.QuestMeta, error)
    GetAdventurer(questID string, advID string) (*fsstore.AdventurerFile, error)
    GetAutomationConfig(automationID string) (AutomationConfig, error)
    ComputeWorkspaceDiff(questID string) (WorkspaceDiffSummary, error)
    GetAutoEvidence(questID string, sessionID string) (AutoEvidenceSummary, error)
    ListEvents(questID string) ([]PolicyAuditEvent, error)
}

type PolicyFactBuilder struct {
    store PolicyFactStore // 数据源抽象，不是字符串路径
}

// BuildReviewFacts 为 review 决策点构建事实
func (b *PolicyFactBuilder) BuildReviewFacts(
    q *fsstore.QuestMeta,
    adv *fsstore.AdventurerFile,
    reviewVerdict string,
    reviewScore int,
) (*PolicyInputFacts, error)

// BuildRecoveryFacts 为 recovery 决策点构建事实
func (b *PolicyFactBuilder) BuildRecoveryFacts(
    q *fsstore.QuestMeta,
    blockedReason string,
    recoveryCount int,
) (*PolicyInputFacts, error)

// BuildNotificationFacts 为 notification 决策点构建事实
func (b *PolicyFactBuilder) BuildNotificationFacts(
    event PolicyAuditEvent,
) (*PolicyInputFacts, error)
```

设计原则：
- **单一入口**：所有 policy 决策点都通过 FactBuilder 拿事实，不自己拼
- **可重放**：相同输入 → 相同 facts → 相同决策。facts 的 hash 记录在 audit log 里
- **只读**：FactBuilder 不修改任何状态，只做读取和计算
- **增量安全**：新增事实维度时，只改 FactBuilder，policy 层不用动
- **Golden test 友好**：可以直接构造 PolicyInputFacts 来测 policy，不需要起整个 quest

#### 5.0.2.2 事实来源映射

每个事实字段都有明确的来源，policy 不关心数据从哪来：

| 事实字段 | 来源 |
|---------|------|
| QuestID / QuestType / Intensity / Source | quest meta |
| EffectType / HasWorkspaceDiff / SideEffectLevel | quest outputs + workspace diff 计算 |
| AllowL2 / HasExternalOutput | quest config |
| CurrentPhaseType / PhaseReadOnly | phase 配置 |
| AutomationID / Tier / Tags / IsOfficial | automation config |
| ReviewVerdict / ReviewScore | 法师评审结果 |
| HasAutoEvidence / EvidencePassed | auto evidence 注入结果 |
| ReworkCount / RecoveryCount | quest 历史统计 |

> **为什么要单独一层 FactBuilder？**
> 如果 policy 直接读 quest 对象，那 quest 结构一变，所有 policy 都要改。
> FactBuilder 是防腐层，把"存储结构"和"策略决策所需的事实"解耦。
> 未来换存储、加字段、改计算逻辑，都只动 Builder，不动 policy。

#### 5.0.3 安全契约

Policy Engine 必须遵守以下安全契约：

1. **默认安全（Safe by Default）**：任何 policy 匹配失败、配置错误、条件不完整时，必须走最保守的动作（require_user / blocked），不能赌。
2. **高风险操作永不自动**：涉及 `effect_type=external_side_effect`、`allow_l2=true`、生产环境部署等场景，禁止 auto_pass / skip_phase。
3. **可重放**：相同的输入事实 + 相同的 policy 配置，必须产出相同的决策。输入事实的 hash 要记录在 audit log 里。
4. **审计完整**：每次决策都必须产生 PolicyAuditEvent，包含完整的输入事实快照和决策结果。
5. **人可随时接管**：任何自动决策的结果，人都可以撤销或覆盖。撤销有独立的 audit log。

#### 5.0.4 接入点

Policy Engine 挂在状态机迁移的**决策点**上，不侵入状态机本身：

```
状态机迁移前 → 收集 InputFacts → Policy Engine 决策 → 执行迁移
```

具体接入点：
- **Review 决策点**：法师评审通过后，决定进 user_review 还是直接 success
- **Recovery 决策点**：异常发生时，决定自动恢复还是进 blocked
- **Notification 决策点**：事件发布后，决定要不要推送通知
- **Trust Tier 决策点**：quest 结束后，决定 automation 是升级还是降级

状态机只管"状态能不能迁"，Policy Engine 管"什么时候迁、迁到哪"。两层解耦。

---

### 5.1 评审策略分级（Review Policy）

#### 5.1.1 核心概念

`review_policy` 决定"法师评审通过后，要不要进 user_review"。

每条 policy 是一个 **条件 → 动作** 的规则：

```go
type ReviewPolicy struct {
    Name      string        // 策略名称
    Condition ReviewCondition // 匹配条件
    Action    ReviewAction   // 命中后的动作
}

type ReviewCondition struct {
    // Quest 基础属性
    QuestType   []string // quest type: code / design / research / ...
    Intensity   []string // quick / standard / deep
    Source      []string // manual / automation / fanout

    // 副作用与风险（核心判断依据）
    EffectType        []string // none / workspace_diff / context_store / external_side_effect / ...
    HasWorkspaceDiff  *bool    // 是否有待应用的工作区改动
    SideEffectLevel   []string // none / low / medium / high
    HasExternalOutput *bool    // 是否有外部输出（发帖、写数据库等）
    AllowL2           *bool    // 是否启用 L2 高权限

    // Automation 相关
    AutomationID    []string // 哪个 automation 创建的
    AutomationTags  []string // automation 的标签
    AutomationTier  []string // automation 的信任等级（tier_0 / tier_1 / tier_2 / tier_3）
    IsOfficialAutomation *bool // 是否官方内置 automation

    // 派生字段（可由上面的字段计算出来，也可直接用）
    RiskLevel   []string // low / medium / high

    // 更多维度按需加
}

type ReviewAction string

const (
    ReviewActionRequireUser ReviewAction = "require_user" // 进 user_review（现状，默认安全）
    ReviewActionAutoPass    ReviewAction = "auto_pass"   // 直接 success，跳过 user_review
)
```

> **关于 auto_reject**：v0.1 不提供 auto_reject 动作。
> 原因：自动拒绝的风险远高于自动通过——会影响统计、信任等级、触发通知和下游自动化。
> HOTL 的目标是减少低价值确认，不是让机器替人否定结果。
> 法师评审已经给出 request_changes / reject 的，走正常流程即可，不需要 policy 再自动拒绝一层。
> 未来可以考虑作为内部审计工具（标记异常 quest），但不作为可配置动作。

#### 5.1.2 Policy 来源与优先级

HOTL 决策是**平台/任务/automation 的人机边界**，不是某个冒险者的人格偏好。不能因为换了一个法师就改变 user_review 行为。

Policy 来源按优先级从高到低：

| 优先级 | 来源 | 说明 | 谁可以改 |
|-------|------|------|---------|
| 1（最高） | **quest override** | 单次 quest 显式指定的 policy（用户手动触发时附带） | 创建 quest 的用户 |
| 2 | **automation policy** | 某个 automation 绑定的 policy（授权边界） | automation 的所有者 |
| 3 | **global policy** | 全局默认 policy（安全基线） | 平台管理员 |
| 4（最低） | **adventurer suggestion** | 冒险者配置里的 policy 建议，只作参考，不直接生效 | adventurer 作者 |

> **关键原则**：adventurer 只能提供 policy 建议（suggestion），不能作为最终 policy source。
> 人机边界是平台责任，不是 agent 个性。

最终 policy 列表的合并逻辑：
1. 从低优先级到高优先级依次合并
2. 高优先级的 policy 可以新增、覆盖、或禁用低优先级的
3. **global safety floor 是硬护栏（hard guardrail），任何来源都不能覆盖**，包括 quest override**

> **Global Safety Floor（硬底线）**

以下条件永远 require_user，任何 policy source 都不能改成 auto_pass：

| 条件 | 动作 | 说明 |
|------|------|------|
| `effect_type = workspace_diff` | require_user | 有工作区改动必须人审 |
| `allow_l2 = true` | require_user | 高权限必须人审 |
| `effect_type = external_side_effect` | require_user | 外部副作用必须人审 |

quest override 只能在 guardrail 允许的动作集合内选择。
比如 effect_type=context_store 的 quest，quest override 可以选 require_user 或 auto_pass；
但 effect_type=workspace_diff 的 quest，quest override 只能选 require_user，选 auto_pass 会被 guardrail 拦截。

guardrail 命中时在 Policy Engine 入口统一校验，不在各 policy source 都绕不过去。

默认值（global policy）保持现状（安全优先）：

```go
GlobalDefaultReviewPolicies = []ReviewPolicy{
    {
        Name:      "default_require_user",
        Condition: ReviewCondition{}, // 空条件 = 匹配所有
        Action:    ReviewActionRequireUser,
    },
}
```

#### 5.1.3 裁决流程

```
法师评审通过
    ↓
按优先级匹配 review_policy（先匹配先命中）
    ↓
命中 auto_pass → 直接 success，记录 policy 名称 + 审计日志
命中 require_user → 进 user_review（现状）
都不命中 → 走 default（require_user，安全兜底）
```

#### 5.1.4 审计与可追溯

自动 pass 的 quest，在 success 记录中标记：
- `auto_passed_by_policy: "context_store_automation_auto_pass"`
- `review_verdict: "pass"`（法师的结论）
- `finalized_by: "policy"`（而不是 user）
- `policy_decision_id: "xxx"`（关联到 PolicyAuditEvent）

##### 关于"人能不能撤销"

**v0.1 不承诺 success 后可以 reject/撤销**。

原因：
1. success 是终态，状态机不支持终态向外迁移。破坏终态语义会引入一堆回滚问题（apply 要不要回滚？exp 要不要扣？trust tier 要不要降级？下游通知要不要撤回？）
2. HOTL 的核心是**默认安全 + 政策收紧**，不是"先放行再撤销"。如果 policy 有疑虑，就应该 require_user，而不是 auto_pass 了再让人撤销。

人对 auto-pass 结果有异议时，可以：
- **创建 follow-up quest**：在原 quest 基础上继续修改
- **标记 audit_failed**：打一个审计失败标签，用于统计和信任回退
- **触发 trust tier 降级**：如果是 automation 的 quest，标记后会触发对应 automation 的信任等级降级
- **回滚 apply**：如果已经 apply 了，可以手动 revert（这是 workspace 操作，不是状态机操作）

未来 v0.2 可以考虑新增 `disputed` 状态或 `human_override` 事件，但 v0.1 不做。先把"为什么自动通过"说清楚，再谈"自动通过了能不能反悔"。

#### 5.1.5 Quick 模式的处理

现状 quick 模式跳过法师，但仍然要过 user_review。

**注意**：quick = 预算少、节奏快，**不等于**风险低、不需要人审。一个 quick coding quest 仍然可能改工作区、产生 workspace diff、甚至触发 apply。不能把"用户选了快"解读成"用户放弃终审"。

因此：
- quick 模式**默认仍然 require_user**（和 Safe by Default 原则一致）
- 用户可以显式配置 policy 让 quick + 特定条件（如 effect_type=none）的 quest 自动 pass
- quick 模式的价值是省掉法师评审的时间，不是省掉人最终拍板的责任

### 5.2 阻塞恢复策略（Block Recovery Policy）

#### 5.2.1 问题

现状：任何异常直接进 blocked，等人来 triage。但很多异常是临时性的，可以自动处理。

#### 5.2.2 恢复策略分层

```
异常发生
  ↓
匹配 recovery_policy
  ↓
一级恢复：自动重试（retry）
  ↓ 不行
二级恢复：自动降级（degrade）
  ↓ 不行
三级恢复：自动提交终审（escalate_to_user_review）
  ↓ 不行（或不适用）
进入 blocked（最终兜底）
```

#### 5.2.3 策略定义

```go
type RecoveryPolicy struct {
    Name      string
    Condition RecoveryCondition // 什么异常
    Actions   []RecoveryAction  // 按顺序尝试的恢复动作
}

type RecoveryCondition struct {
    BlockReason []string // blocked_reason: timeout / no_progress / auth_error / ...
    PhaseType   []string // execute / review
}

type RecoveryAction struct {
    Type  RecoveryActionType
    Params map[string]any
}

type RecoveryActionType string

const (
    // 恢复当前阶段，不改变阶段边界，最安全
    RecoveryRetry           RecoveryActionType = "retry"             // 重试当前阶段（重置部分状态）
    RecoveryResumeSamePhase RecoveryActionType = "resume_same_phase" // 从断点继续当前阶段

    // 降级当前阶段，不跨阶段
    RecoveryDegrade RecoveryActionType = "degrade" // 降级（降 intensity、加 hint 等）

    // 升级到人的决策，交给人
    RecoveryEscalate RecoveryActionType = "escalate_to_user_review" // 升级到 user_review

    // 跳过阶段，最危险 —— 默认禁用，必须显式 policy 声明 + 满足安全条件
    // 只允许跳过 L0 / 只读 / 无副作用阶段
    // 必须记录完整 audit trail：为什么跳过、命中了什么 policy、跳过的阶段有没有副作用
    RecoverySkipPhase RecoveryActionType = "skip_phase"
)
```

> **安全原则**：`skip_phase` 默认禁用。启用时必须同时满足：
> 1. 目标阶段是 L0 或只读模式
> 2. 该阶段无 side effect（不写文件、不触发外部 API、不修改 workspace）
> 3. policy 显式声明允许
> 4. 完整审计日志记录

#### 5.2.4 各类异常的默认恢复策略

| 异常类型 | 默认策略 | 说明 |
|---------|---------|------|
| 阶段超时 (timeout) | retry ×2（各加 50% 时长）→ escalate | 超时可能是预算给少了，先加时试试 |
| 无进展 (no_progress) | degrade ×1（降 intensity）→ escalate | agent 卡住了，降强度可能有帮助 |
| 连续错误 (consecutive_errors) | retry ×2 → escalate | 可能是临时故障，先重试 |
| 认证错误 (auth_error) | 直接 blocked | 认证问题必须人处理，不能自动 |
| 总时长超限 (duration_exceeded) | 直接 escalate | 总预算用完了，直接给人看结果 |

#### 5.2.5 恢复状态持久化（RecoveryState）

自动恢复必须是可恢复的。服务重启后，需要知道：
- 当前恢复到第几步了
- 上一个 action 有没有执行完
- 下一次重试在什么时候
- 原始异常是什么

```go
type RecoveryState struct {
    // 定位
    QuestID    string // 所属 quest
    PhaseIdx   int    // 发生在哪个阶段
    SessionID  string // 发生在哪个会话

    // 策略信息
    PolicyName    string // 命中的 recovery policy 名称
    AttemptIndex  int    // 当前是第几次恢复尝试（从 0 开始）
    ActionIndex   int    // 当前执行到 policy 里的第几个 action
    InputHash     string // 触发恢复时的输入事实 hash，用于幂等判断

    // 状态
    Status         RecoveryStatus // pending / running / succeeded / failed
    LastAction     string         // 上一个执行的 action
    LastError      string         // 上一次失败的错误
    NextRetryAt    time.Time      // 下一次重试时间（指数退避用）
    StartedAt      time.Time      // 第一次开始恢复的时间
    LastAttemptAt  time.Time      // 最近一次尝试的时间

    // 原始异常
    BlockReason    string // 原始 blocked_reason
    OriginalError  string // 原始错误信息
}

type RecoveryStatus string

const (
    RecoveryStatusPending   RecoveryStatus = "pending"    // 等待执行恢复动作
    RecoveryStatusRunning   RecoveryStatus = "running"    // 正在执行恢复动作
    RecoveryStatusSucceeded RecoveryStatus = "succeeded"  // 恢复成功，已回到 running
    RecoveryStatusFailed    RecoveryStatus = "failed"     // 所有恢复动作都失败了，已进 blocked
)
```

恢复流程的幂等性保证：
1. 每次恢复前先检查 `RecoveryState`，如果已经有同一 `input_hash` 的记录，直接用已有进度
2. 单文件写入使用 **temp + rename 原子写**（和现有 WriteJSON 一致）
3. 跨文件更新（如状态迁移 + RecoveryState 更新）不保证事务，必须设计为**可重放 + 幂等补偿**：先写 RecoveryState，再执行状态迁移；重启扫描 pending/running 的记录，重新尝试
4. 恢复动作本身必须幂等：同一 attempt_index 重复执行不会产生额外副作用

#### 5.2.6 恢复次数限制

每个 quest 有全局恢复次数上限（默认 3 次），避免无限重试。超过上限直接进 blocked。

每次恢复都记录在 `RecoveryState` 中，包含 attempt_index、action、reason、result。

### 5.3 等待输入状态（Waiting for Input）

#### 5.3.1 问题

现状：剑士通过 `quest_ask` 向用户提问时，quest 进入 blocked 状态。但"等用户回答问题"和"等用户 triage 阻塞"语义完全不同：

- Blocked = 系统卡住了，需要人决定怎么办（continue / cancel / 终审）
- Waiting for input = 系统在等一个具体输入，拿到了就自动继续

混在一起会导致：
1. 用户看到 blocked 不知道是"出问题了"还是"在等我回答"
2. 回复问题需要用 comment，语义不对
3. 提问超时也没有自动处理（永远等下去）

#### 5.3.2 设计

新增独立状态 `waiting_input`：

```
     ┌─────────────┐
     │   running   │
     └──────┬──────┘
            │ quest_ask
            ▼
     ┌─────────────┐
     │waiting_input│  ← 新增状态
     └──────┬──────┘
            │ 用户回复 / 超时自动继续
            ▼
     ┌─────────────┐
     │   running   │
```

状态机迁移：

```go
model.QuestStatusRunning: {
    // ... 现有迁移 ...
    model.QuestStatusWaitingInput,  // 新增：提问等待输入
}

model.QuestStatusWaitingInput: {
    model.QuestStatusRunning,    // 收到回复 / 超时自动继续
    model.QuestStatusBlocked,    // 超时且重试后仍无响应（兜底）
    model.QuestStatusCancelled,  // 用户取消
}
```

#### 5.3.3 自动超时机制

`quest_ask` 时可以指定超时时间（默认 1 小时），超时后：
- 自动注入一条"用户未在规定时间内回复，请基于现有信息继续"的 system message
- 自动恢复 running 状态
- agent 自己决定接下来怎么办

agent 也可以在 `quest_ask` 里指定超时策略：
- `continue`（默认）：超时自动继续
- `block`：超时进 blocked
- `escalate`：超时提交 user_review

#### 5.3.4 用户交互与 Answer 存储

用户回复有两种方式：
1. `gloop quest answer --question-id xxx --answer "..."`（推荐，语义明确）
2. `gloop quest comment "..."`（兼容现有方式，blocked 下评论自动继续的逻辑保留）

**Answer 存储**：`gloop quest answer` 产生的回答 append-only 写入：
```
workspace/quests/<qid>/answers.jsonl
```

每条 answer 结构：
```go
type AnswerRecord struct {
    AnswerID   string    // 唯一 ID（answer_xxx）
    QuestionID string    // 对应哪个问题
    Content    string    // 回答内容
    Source     string    // cli / api / lark / comment_bridge
    CreatedAt  time.Time
}
```

读取和推进：
- `WaitingInputState.LastAnswerID` 从 `answers.jsonl` 推进
- 恢复时从 `LastAnswerID` 之后的记录继续处理
- comment 兼容逻辑不变（从 comments.jsonl 按 LastCommentSeq 推进）
- 两种路径互相独立，都可以触发 waiting_input → running 的恢复

> 设计原则：append-only、可追溯、不修改历史。answer 和 comment 是两条独立的事件流，互不干扰。

#### 5.3.5 状态契约（持久化字段）

`waiting_input` 状态必须持久化完整上下文，确保服务重启后可恢复：

```go
type WaitingInputState struct {
    QuestionID    string    // 提问 ID，用于去重和关联
    QuestionText  string    // 问题正文
    PhaseIdx      int       // 提问发生在哪个阶段
    SessionID     string    // 提问发生在哪个会话
    PhaseType     string    // execute / review
    Asker         string    // 谁提的问（剑士/法师）

    TimeoutMs     int64     // 超时时间
    TimeoutAction string    // 超时时的动作：continue / block / escalate
    ResumePhaseIdx int      // 恢复后回到哪个阶段（通常和 PhaseIdx 相同）

    // 用于追踪用户回复
    LastCommentSeq int      // 上次处理到的评论序号（递增整数，恢复时从 LastCommentSeq+1 开始）
    LastAnswerID   string   // 已处理的最后一条 answer 的 ID（answer 有独立的 ID 体系）
    ProcessedAnswerIDs []string // 已处理的 answer ID 列表（防重，冗余但安全）

    // 审计
    AskedAt       time.Time // 提问时间
}
```

恢复流程：
1. 服务重启后，扫描所有 `waiting_input` 状态的 quest
2. 检查是否超时 — 超时执行对应 timeout_action
3. 检查是否有新的 answer / comment — 有则处理并恢复 running
4. 没有新消息则继续等待

### 5.4 评审与应用合并（Review + Apply）

#### 5.4.1 问题

对于 coding 类 quest，用户终审的操作链是：
```
user_review → pass → success → apply → 完成
```
中间"pass"和"apply"是两步操作，但大部分时候用户的决策是一致的：
- 觉得好 → pass 然后 apply
- 觉得不好 → request_changes 或 reject

两步操作 = 两次上下文切换 = 两倍摩擦。

#### 5.4.2 设计

**模型**：verdict 是质量裁决，apply 是副作用动作。两者是独立维度，不混合。verdict 枚举保持纯净：`pass / request_changes / reject`。

User Review 界面/CLI 增加 **`--apply` 选项**（和 `--verdict pass` 搭配使用）：

```bash
# 现状：只通过不应用
gloop quest review --verdict pass --comment "不错"

# 新增：通过并应用（两个独立参数，不是新 verdict）
gloop quest review --verdict pass --apply --comment "不错"
```

执行路径：
```
verdict=pass, apply=true
  ↓
1. 单个 meta.json 通过 temp+rename 原子写入：
   status = success
   apply_intent = pending
   （注意：这里的原子写只覆盖 quest meta 单文件，
    apply 产生的 workspace patch backup 是另一层可恢复证据，
    两者不是同一个事务。）
  ↓
2. 执行 apply（和手动 apply 一样的安全检查流程）
  ↓
3a. apply 成功 → 更新 apply_status=applied
3b. apply 失败 → 更新 apply_status=failed + apply_error
    （success 状态保留，不回滚）
```

> **可恢复性**：服务重启后，扫描所有 `apply_intent=pending` 的 quest，继续执行 apply。
> 不会出现"pass 了但 apply 没做"的情况，也不会重复 apply。
>
> 注意：这里不用数据库事务的语义。Gloop 是 JSON 文件存储，单文件 temp+rename 原子写，跨文件用"两阶段持久化 + 重启恢复"保证最终一致。

#### 5.4.3 注意事项

- **verdict 枚举保持纯净**：`pass / request_changes / reject`。apply 是附加参数，不污染领域裁决。
- 这是**新增选项**，不改变现有 pass / reject / request_changes 的行为
- apply 失败不会回滚 success 状态（质量通过和能否应用是两件事）
- 只对 coding / 有 workspace 的 quest 有效，design 类 quest 不显示这个选项
- 和现有 API 的 `approve-and-apply` 语义一致，不是新概念

### 5.5 自动化信任分级（Automation Trust Tiers）

#### 5.5.1 问题

现状：所有 automation 创建的 quest 默认 `auto_start=false`，进 inbox 等人确认。

但一个自动化跑了几百次都成功，每次还要人点 start，就是纯粹的噪音。Loop Engineering 的要义是**信任积累后逐步放权**。

#### 5.5.2 信任等级

```
Tier 3 — 全自动：auto_start + auto_pass user_review + auto_apply
  ↑ 连续成功 N 次
Tier 2 — 自动跑但要人审：auto_start + 正常走 user_review
  ↑ 连续成功 N 次
Tier 1 — 确认后跑：auto_start=false，进 inbox 确认（现状）
  ↑ 失败回退
Tier 0 — 暂停：连续失败多次，自动暂停该 automation，等人排查
```

每个等级的升级/降级条件可配置，默认：
- 升级：连续 **独立验证成功** 10 次升一级
- 降级：失败 1 次降一级
- 暂停：连续失败 3 次暂停

> **关键约束**：信任升级只能用"有独立验证的成功"计数，auto-pass 自己产生的 success 不算数。
> 否则会有自我强化问题：一旦 auto_pass 开了，success 可能只是 policy 自己制造出来的，不等于质量过关。

**可计入升级的成功类型**（独立验证）：
- `human_approved` — 人工在 user_review 中点 pass
- `sampled_audit_pass` — 被抽样复审通过的 auto-pass quest
- `test_doctor_pass` — 自动化测试 / doctor 检查通过
- `apply_success` — apply 到 base 成功（说明改动实际可合入）
- `reviewer_verdict_pass` — 法师评审通过（只用于 tier_0→tier_1 的升级，更高 tier 不算）

**不可计入升级的成功类型**（自我循环）：
- `policy_auto_pass` — policy 自动跳过 user_review 的 success
- `auto_apply_success` — auto_apply 成功（可作为附加加权，但不能单独作为升级依据）

升级规则示例：
- Tier 0 → Tier 1：法师 pass 就算（10 次）
- Tier 1 → Tier 2：需要人工 pass 或抽样审计 pass（10 次）
- Tier 2 → Tier 3：需要人工 pass + apply 成功（10 次）

#### 5.5.3 人工锁定

人可以手动锁定某个 automation 的信任级别：
- `"我就是要每次都确认"` → 锁定在 Tier 1
- `"这个自动化很成熟了，直接跑"` → 锁定在 Tier 2 或 3

锁定后不随成功率波动。

#### 5.5.4 审计

每个 automation 维护完整的历史记录：
- 总运行次数 / 成功率 / 平均耗时
- 最近 10 次结果摘要
- 信任等级变更历史（什么时候升/降/锁定，原因是什么）

人随时可以查看和调整。

### 5.6 通知降噪（Notification Policy）

#### 5.6.1 问题

现状：quest 关键事件（开始、完成、阻塞、用户终审等）自动发通知。事件多了就是噪音，人会忽略所有通知。

#### 5.6.2 通知分级

| 级别 | 事件类型 | 默认是否推送 | 说明 |
|------|---------|-------------|------|
| **需要行动** | user_review / blocked / inbox | ✅ 推送 | 人必须做点什么才能推进 |
| **重要结果** | quest 成功 / 失败 / 取消 | ⚠️ 可配置 | 人想知道结果但不需要立刻行动 |
| **进度通知** | 阶段切换 / 返工 / 自动恢复 | ❌ 默认不推 | 纯进度，无决策价值 |
| **调试信息** | 日志 / 事件详情 | ❌ 不推 | 查问题时看 dashboard |

#### 5.6.3 通知合并

同一时间窗口内的多个同类通知合并：
- 多个 quest 同时完成 → 一条摘要通知："3 个 quest 已完成，2 成功 1 失败"
- 多个 automation 触发 → 不逐个通知，按时间窗口汇总

#### 5.6.4 通知带操作入口

需要行动的通知（user_review / blocked / inbox）直接带操作按钮：
- User Review 通知：`[通过] [要求返工] [拒绝]`
- Blocked 通知：`[继续] [提交终审] [取消]`
- Inbox 通知：`[接受] [拒绝] [编辑]`

不用点开详情页再操作，一步到位。

#### 5.6.5 接入边界

通知 policy 是 **EventSubscriber 的输入过滤 + 聚合层**，**不是**状态机的决策点，**不影响**任何状态迁移。

> **核心定位**：Notification Policy 是输出端的过滤器，不是状态机的一部分。
> 它决定"用户看得到哪些事件"，不决定"quest 怎么走"。

架构位置：
```
状态变更 → 发布事件 → Event Bus → [Notification Policy 过滤层] → Notifier → 用户
                                    ↑
                             纯过滤 + 聚合
                             不回写状态
                             不影响流程
```

具体来说：
- 所有状态迁移点只管发事件，不判断"要不要通知"
- NotificationPolicy 挂在 EventSubscriber 前面，作为统一的过滤器
- 通知合并、分级、去重、静默窗口都在这一层做
- 新增事件类型默认走 default policy（action_required 级别的推送，其他不推），不需要改业务代码
- 通知 policy 失败（配置错误 / 匹配异常）不影响 quest 流程，兜底为"不推送"

和其他 Policy 的关系：
| Policy 类型 | 接入位置 | 影响范围 | 失败兜底 |
|------------|---------|---------|---------|
| Review Policy | 法师评审后 → 状态迁移前 | 决定进 user_review 还是 success | 进 user_review（安全） |
| Recovery Policy | 异常发生 → 状态迁移前 | 决定自动恢复还是进 blocked | 进 blocked（安全） |
| Notification Policy | EventSubscriber 输入侧 | 决定用户收不收通知 | 不推送（安全） |
| Trust Tier Policy | quest 结束后 → 等级更新前 | 决定自动化信任升降 | 不升级（安全） |

这样做的好处：
- 单一真相源：通知策略只在一个地方定义
- 可观测：所有通知决策都有 audit log
- 好测试：policy 层可以独立做 golden test

---

## 六、里程碑计划

**实施原则**：先建最小地基，再上最保守策略，逐步放开。**不要一上来就做全套 HOTL。**

分两波推进：

- **第一波（Phase 2.5.0 ~ 2.5.1）**：Policy Engine Contract + 最保守 Review Policy。验证链路，风险可控。
- **第二波（Phase 2.5.2 ~ 2.5.6）**：Recovery / Waiting Input / Trust Tier / Notification 等进阶能力。第一波跑稳了再上。

### Phase 2.5：HOTL 升级

依赖：主 spec Phase 2（宏循环管道化）完成

---

#### 第一波：最小地基（约 3-5 天）

##### Milestone 2.5.0：Policy Engine 基础设施（2-3 天）

先把地基打好，再建策略。

- `PolicyInputFacts` + `PolicyFactBuilder` — 统一事实层，防腐隔离（基于 PolicyFactStore 抽象，不是字符串路径）
- `PolicyDecision` 数据结构 + `policy.decision` 事件
- `PolicyAuditEvent` 持久化：`quests/<qid>/events.jsonl` + 全局 `workspace/events/global.jsonl`（event_type=policy.decision，和全局事件流共用）
- Policy 匹配框架（条件 → 动作，优先级匹配，四级来源合并）
- **Global Safety Floor（硬护栏）**：workspace_diff / allow_l2 / external_side_effect 永远 require_user，任何 source 都不能覆盖
- Golden test 框架（输入事实 → 期望决策 → 比对）
- 默认安全兜底：任何异常都回落到最保守动作

##### Milestone 2.5.1：Review Policy 保守落地（2 天）

只实现一个最保守的策略，验证整个链路通不通。

**允许 auto_pass 的唯一条件**：
- source = automation
- effect_type = context_store / none
- 无 workspace_diff
- automation_id 在 **auto_pass_allowlist** 中

> `official` 只是来源属性，不是可信充分条件；`tier_2+` 要等 Trust Tier 模块上线后再接入。

**默认策略**：require_user（全量要人审，和现状一致）

- ReviewPolicy 条件维度补齐（effect_type / workspace_diff / side_effect_level / allow_l2 等）
- 实现"context_store 类 automation allowlist 可 auto_pass"的保守策略
- Golden test 覆盖：
  - `workspace_diff 永远不默认 auto_pass`
  - `allow_l2 永远不默认 auto_pass`
  - `external_side_effect 永远不默认 auto_pass`
  - `不在 allowlist 的 automation 不 auto_pass`
- 审计日志 + audit_failed 标记能力
- CLI/Server API 适配

---

#### 第二波：进阶能力（第一波跑稳后再排期）

##### Milestone 2.5.2：Block Recovery Policy（2-3 天）

- RecoveryPolicy 配置和匹配逻辑
- `retry` / `resume_same_phase` / `degrade` / `escalate_to_user_review` 动作
- `skip_phase` 动作（默认禁用，满足 L0/只读/无副作用才可启用）
- `RecoveryState` 持久化（temp+rename 原子写 + 幂等补偿）
- 超时/无进展/连续错误的默认恢复策略
- 恢复次数全局限制
- 审计日志

##### Milestone 2.5.3：Waiting Input 状态（1-2 天）

- 新增 `waiting_input` 状态和状态机迁移
- `WaitingInputState` 持久化契约（question_id / timeout / resume_phase / last_comment_seq / last_answer_id 等）
- Answer 存储：`quests/<qid>/answers.jsonl`（append-only）
- `quest_ask` 改为进 `waiting_input` 而非 blocked
- 超时自动继续机制
- CLI: `gloop quest answer` 命令
- 评论兼容逻辑保留（blocked 下评论自动继续的逻辑不变）
- 重启恢复：扫描 waiting_input quest，处理超时和新回复

##### Milestone 2.5.4：Review + Apply 合并（1 天）

- `--verdict pass --apply` 参数支持（不是新 verdict，是附加参数）
- 两阶段可恢复 apply：先写 apply_intent=pending，再执行，成功写 applied / 失败写 failed
- 重启恢复：扫描 apply_intent=pending 的 quest，继续执行
- CLI / Server API 适配
- 保持 verdict 枚举纯净

##### Milestone 2.5.5：Automation Trust Tiers（2-3 天）

- 信任等级定义（tier_0 ~ tier_3）
- 升级/降级逻辑（**只吃独立验证的成功，不吃 auto-pass 自己的成功**）
- 人工锁定能力
- 历史记录与审计
- 和 ReviewPolicy 打通：tier_2+ verified 可替代 allowlist
- 配置入口

##### Milestone 2.5.6：Notification Policy（1-2 天）

- 通知分级与默认策略
- 通知合并 / 去重 / 静默窗口机制
- 作为 EventSubscriber 的输入过滤层实现（不侵入状态机，不散落各处）
- 失败兜底：不推送（不影响 quest 流程）
- 通知带操作入口（如果前端/飞书卡片支持）

### 可前置的小改动（Phase 1 就能做）

以下改动不依赖管道化，可以提前验证：
1. **超时自动追加 hint 而不是直接 blocked** — 小步验证恢复策略的思路
2. **法师评审 + auto_apply 成功路径梳理** — 为后续 auto_pass 铺路

> 注意：design quest 跳过 user_review、quick 模式 auto_pass 等，**不要**作为 Phase 1 的前置改动。
> 原因：风险边界不清，应该等 Policy Engine 基础设施建好后，作为保守策略的扩展再上。

---

## 七、风险与应对

### 7.1 自动 pass 放过了有问题的交付

**风险**：policy 配置不当，导致本应人审的 quest 自动通过了。

**应对**：
- 默认全量 require_user，auto_pass 需要显式开启
- 自动 pass 的 quest 保留人工撤销能力
- 可以配置"抽样复审"：自动 pass 的 quest 按一定比例随机抽回 user_review，用于校验 policy 合理性
- Design 类 quest（纯文本、不碰代码）风险最低，可以先放开验证

### 7.2 自动重试导致资源浪费 / 死循环

**风险**：恢复策略不断重试，消耗 token 和时间，但问题根本解决不了。

**应对**：
- 全局恢复次数上限（默认 3 次）
- 同类异常不重复触发相同恢复动作（比如超时重试两次还超时，就升级，不再重试）
- 恢复动作有指数退避（如果适用）
- 总时长预算仍然是硬上限，到了就停

### 7.3 信任等级升级太快，放了不该放的权

**风险**：自动化连续走运几次就升上去了，遇到真问题时人不在。

**应对**：
- 升级条件保守（默认 10 次连续成功才升一级）
- 降级条件严格（失败 1 次就降一级）
- 高等级（Tier 3 全自动）需要人工显式解锁，不能纯靠自动升级
- 信任等级变更都有通知，人随时可以干预

### 7.4 通知少了，人会漏事

**风险**：降噪过度，重要的事反而没注意到。

**应对**：
- "需要行动"级别的通知永远推送，这是底线
- 提供每日摘要：每天早上推一条"昨天 N 个 quest 完成，M 个等待你处理"
- 降噪是可配置的，觉得少了可以调回去

---

## 八、后续探索与可选项

以下方向 v0.1 不做，按"近期可实验"和"远期探索"分类，留给未来评估。

### 8.1 近期可实验（第一波跑稳后可以小步验证）

1. **可选设计阶段（Design Phase）** — 复杂 quest 在执行前插入一个独立的设计评审阶段。剑士先出方案大纲，法师评审方案方向，过了再进执行。好处是提前纠偏，坏处是多一轮迭代。适用于高复杂度 / 高返工成本的 quest。管道化架构天然支持，只需加 phase 定义。

2. **auto_reject 动作** — 自动拒绝低质量交付。v0.1 不做：自动拒绝的风险远高于自动通过，会影响统计、信任等级、触发下游动作。HOTL 的目标是减少低价值确认，不是让机器替人否定结果。

3. **基于风险的动态评审** — 根据 quest 的实际改动范围（影响了哪些文件、多少行、什么类型）动态决定要不要 user_review，而不是只看 type/intensity。

### 8.2 远期探索（架构变化较大，需要单独评估）

4. **多方案对抗式设计** — 同一 quest 并行跑多个剑士出不同方案，法师交叉评审，择优落地。本质是把 maker-checker 从"单方案迭代"升级为"多方案竞争"。成本高（token × N），适合高价值 quest。

5. **人机协同评审** — 法师评审 + 自动测试 + 人抽查，三者结合，人只看法师不确定的部分。

6. **自然语言策略配置** — 人用自然语言说"设计类 quest 不用给我看"，系统自动翻译成 policy。

7. **跨 quest 异常模式识别** — 批量失败时自动暂停相关 automation，避免雪崩。

8. **人的工作量仪表盘** — 展示"这周你点了多少次 pass / 平均响应时间 / 哪些 automation 最占你时间"，帮助人发现优化空间。

---

## 九、与主 spec 的映射关系

| 主 spec 阶段 | HOTL 改动 | 依赖关系 |
|------------|-----------|---------|
| Phase 0.5：Pipeline 持久化 | 无 | - |
| Phase 1：上下文传递修复 | 部分可前置（design/quick 跳过 user_review 可先做） | 独立，不依赖 |
| Phase 2：宏循环管道化 | 全部 HOTL 特性的基础（policy 需要挂在管道上） | HOTL Phase 2.5 依赖主 spec Phase 2 |
| Phase 3：per-quest 自动化 | 信任分级与事件自动化可以协同 | 部分联动 |
| Phase 4：跨 quest workflow | 未来探索 | 远期 |

建议实施顺序：
```
Phase 1（上下文修复）
  ↓ 同时可以顺手做
Phase 1.5（小 HOTL 验证：design/quick 跳过 user_review）
  ↓
Phase 2（宏循环管道化）
  ↓
Phase 2.5（HOTL 全量升级）
  ↓
Phase 3（事件自动化 + 信任分级联动）
```
