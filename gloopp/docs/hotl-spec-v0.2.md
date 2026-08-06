# gloop HOTL 演进 Spec v0.2 — checker 权威化与自主闭环

> 版本：v0.2.0
> 状态：已落地（v0.2.0，五环全部实现并通过测试）
> 前置：v0.1（`hotl-spec-v0.1.md`）已落地 Policy Engine / ReviewPolicy / RecoveryPolicy / WaitingInput / Trust Tier / Notification Policy
> 参考框架：[Addy Osmani — Loop Engineering](https://addyosmani.com/blog/loop-engineering/)

---

## 一、第一性原理

HOTL（Human-on-the-Loop）的本质：**agent 自主闭环，人 on the loop**。从这条原理推出三条不可动摇的原则：

1. **checker 是权威，不是建议者** — checker（法师）pass = 最优解。人没有能力审 agent 产出（comprehension debt），user_review 作为"人审质量"是伪命题。人审不动，只是橡皮图章，最终导致 cognitive surrender。

2. **人是感知者，不是审批者** — 人不需要判断产出"好不好"（那是 checker 的活，人做不了），只需要感知 quest"做了什么、产生了什么影响"。人的价值在于方向决策和异常处理，不在于逐行 review。

3. **人只介入 agent 搞不定的事** — blocked（agent 求助）和 waiting_input（agent 缺信息）。这两个是 agent 主动求助，不是人审质量。

## 二、与 v0.1 的核心差异

v0.1 的保守立场是"workspace_diff 永远 require_user"（HardGuardrailReason 第一条）。这在 v0.1 阶段是正确的——当时 checker 的评分机制还不成熟，score 是纯记录零门禁权重。

v0.2 的核心转变：

| 维度 | v0.1（保守） | v0.2（自主闭环） |
|------|-------------|-----------------|
| checker pass 的语义 | "及格可接受"，仍需人审 | "最优解"，自主闭环 |
| workspace_diff | 永远 require_user（硬护栏） | checker pass → 自主 apply |
| user_review | 默认必经节点 | 降级为可选回溯入口 |
| score 的作用 | 纯记录，零门禁权重 | checker 内部决策工具（决定给 pass 还是 request_changes） |
| 人的角色 | 审批者（每个 quest 终审） | 感知者（感知影响，处理异常） |
| HardGuardrailReason | 阻断放行 | 标记"需感知影响"（影响通知强度，不阻断放行） |

## 三、方案（五环因果链）

每环是下一环的前提，倒过来做没有意义。

### 环 0：checker 权威化（地基）

**目标**：让 checker pass 成为可信的"最优解"信号，而非"及格可接受"。

**改动**：

1. **ReviewPolicy.Decide**：`verdict=pass` → `auto_pass`，不再只对 context automation 放行，不再按 score 分级。所有 quest 一视同仁。

2. **score 回归本职**：score 是 checker 内部决定给 pass 还是 request_changes 的工具（SKILL.md 已定义：9-10 优秀、7-8 良好、5-6 及格、1-4 较差）。pass 之后 score 不再当放行门槛——因为 checker 给了 pass 就意味着认为这是最优解。

3. **HardGuardrailReason 语义调整**：L2/外部副作用不再阻断 pass（质量是 checker 判断的），但标记 `requires_impact_notification=true`——apply 后必须发影响通知，让人感知。workspace_diff 从硬护栏中移除。

4. **取消 isContextAutomationQuest 特殊分支**：handleReviewPass 里不再只对 context automation 调 decideReviewPolicy，所有 quest 都走 policy 决策。

5. **trust 反馈环**：auto_pass 的 quest 如果后来发现问题（apply 失败、follow-up quest 修复了同一问题），ConsecutiveFailures 累计到阈值后自动收紧——但这不是 v0.2 的阻断条件，是后续观察项。

**为什么这是地基**：没有它，后面所有环都在把"不靠谱的通过"往外送。checker 权威化后，pass 就是可信的，自主闭环才成立。

### 环 1：自主闭环（checker pass → apply → 感知通知）

**目标**：checker pass 后不再 MoveToUserReview，直接自主 apply + 发影响通知。

**改动**：

1. **handleReviewPass 主路径改写**：
   - 移除 `MoveToUserReview` 作为默认终态路径
   - pass → policy 决策 → auto_pass → 自主 apply（复用 `applyWorkspace` 逻辑）
   - apply 成功 → quest 直接 success（`FinalizedBy=policy`）
   - apply 失败 → quest 进 user_review（apply 失败是人需要知道的异常，不是质量审批）

2. **影响通知**：apply 成功后通过 EventBus 发布 `EvtQuestApplied`，ConnectorSubscriber（环 4）或 EventSubscriber 发飞书卡片：
   - 改了哪些文件、多少行
   - quest 做了什么事（query 摘要）
   - 影响范围（effect_type / side_effect_level）
   - 可回溯入口（quest detail 链接）

3. **user_review 降级为可选回溯入口**：
   - 不再阻断主流程
   - quest trace 完整保留，人想看随时能看
   - 列表页不再把自主闭环的 quest 放在"待审"区

4. **保留的人工介入路径**：
   - blocked：agent 搞不定，人必须介入
   - waiting_input：agent 缺信息，人必须回答
   - apply 失败：技术问题，人需要处理
   - request_changes 达到 MaxRework：返工耗尽，人决定方向

**人的体验变化**：从"收到一堆待审 quest 逐个点通过"变成"收到影响通知，想看可以看，不看也没事"。

### 环 2：event 触发补全（跨 quest 工作流）

**目标**：放开 event automation 的 spawn quest 限制，让"quest A success → 触发 quest B"成为可能。

**改动**：

1. **runEventAutomationAction 增加 spawn_quest action**：当前只有 note/通知，新增创建 quest 的能力。

2. **AutomationConfig 增加 EventAction 配置**：
   ```
   event_trigger: {
     on: "quest.success" | "quest.applied" | "quest.blocked" | ...,
     action: "spawn_quest",
     spawn_automation: "xxx",  // 触发哪个 automation
   }
   ```

3. **安全限制**：
   - spawn 的 quest 继承触发 automation 的风险等级
   - 链式深度限制（max_depth=3，防止无限链式）
   - 同一事件不重复触发（去重，复用现有 state 追踪）

4. **前端暴露**：Automations 页面增加 event 触发配置入口。

**场景**：修 bug 成功 → 自动触发回归测试；context 更新成功 → 自动触发 inbox triage。

### 环 3：/goal 模式（持续运行 + 独立完成检查）

**目标**：新增 goal 触发器，quest 持续运行直到条件满足。

**改动**：

1. **新增 TriggerGoal**：quest 跑完一轮后不直接终态，由独立小模型检查"目标是否达成"。

2. **制造者/检查者分离延伸到停止条件**：
   - 剑士执行 → 法师评审 → 独立完成度检查模型判定 `done` / `continue`
   - `done` → 进入环 1 的自主闭环
   - `continue` → 自动返工继续（不消耗 MaxRework，goal 模式有独立的 max_iterations）

3. **CompletionChecker**：
   - 独立于法师的检查者（可以是轻量模型）
   - 输入：quest goal + 当前产出
   - 输出：`done`（目标达成）/ `continue`（还需迭代）+ 原因

4. **配置**：
   ```
   trigger: "goal",
   goal: "所有测试通过且无 lint 警告",
   max_iterations: 10,
   ```

**对齐 Addy 框架**："after every turn a separate small model checks whether you are done" — 制造者和检查者分离应用于停止条件本身。

### 环 4：triage 分拣 + Connectors

**目标**：人感知全局而非逐个审批。

**triage 分拣**：

1. **inbox 只保留 blocked / waiting_input**：人必须介入的。
2. **自主闭环的 quest 进"影响日志"分区**：默认折叠，可回溯。
3. **inbox-triage skill 更新**：扫描时区分"需人处理"和"已自主闭环"。
4. **列表页重构**：
   - "需处理"分区：blocked / waiting_input，默认展开
   - "影响日志"分区：自主闭环的 quest，默认折叠，显示影响摘要

**Connectors**：

1. **新增 `internal/connectors/` 包**：ConnectorSubscriber 仿照 notifications.EventSubscriber 订阅 EventBus。

2. **监听 `EvtQuestApplied`**：当前已定义但无人处理，正好补上。

3. **第一批 connector**：
   - Git connector：apply 后 commit + push + 开 MR
   - 飞书 connector：复用 LarkCliNotifier 发影响卡片

4. **配置驱动**：AutomationConfig 增加 `connectors` 字段，quest 继承 automation 的 connector 配置；user quest 可在创建时选。

## 四、不改的

- 三角色（剑士/法师/用户）不变
- 预算/超时/返工机制不变
- blocked / waiting_input 的人工介入机制不变
- worktree 隔离不变
- Policy Engine 基础设施（facts / decision / audit）不变
- RecoveryPolicy 不变（blocked 恢复仍按 v0.1 逻辑）

## 五、因果关系

```
环 0（checker 权威化）→ 环 1（自主闭环）→ 环 2（event 触发）→ 环 3（/goal）→ 环 4（triage + connectors）
```

- 环 0 让 pass 可信
- 环 1 让可信的 pass 自主闭环
- 环 2 让闭环的 quest 能串成工作流
- 环 3 让工作流能持续运行直到目标达成
- 环 4 让人感知全局而非逐个审批

## 六、风险与应对

### 6.1 checker 误判导致有问题的交付自主 apply

**风险**：checker 给了 pass 但实际有问题，自主 apply 后影响 base 代码。

**应对**：
- apply 仍走 SafetyCheck（现有机制不取消）
- apply 产生 backup，可回滚
- trust 反馈环：apply 后发现问题会收紧后续 auto_pass
- 人可以随时创建 follow-up quest 修复
- 高风险 quest（adversarial 强度）可配置仍 require_user 作为过渡

### 6.2 人失去对系统行为的感知

**风险**：自主闭环后人不知道系统在做什么。

**应对**：
- 影响通知是必须的（环 1）
- triage 分拣让人能看到全局（环 4）
- 每日摘要：汇总当天自主闭环的 quest
- quest trace 完整可回溯

### 6.3 event 链式触发失控

**风险**：quest A → B → C → ... 无限链式。

**应对**：
- max_depth 限制（默认 3）
- 同事件去重
- 链式 quest 继承风险等级，高风险不自动链式

## 七、落地记录（v0.2.0）

### 7.1 关键工程决策：HOTLAutoClose 特性开关

v0.2 的核心行为变化（checker pass → 自主闭环）与 v0.1 测试套件编码的 HITL 流程直接冲突——约 30 个 orchestrator E2E 测试断言 `user_review` 终态及随后的 `Apply`/`Approve` 操作。重写这 30 个测试不仅工作量巨大，还会丢失作为回归网的 v0.1 行为快照。

**决策**：引入 `GlobalConfig.HOTLAutoClose bool` 特性开关：
- **生产默认 `true`**（`DefaultGlobalConfig` 设 true）——HOTL 作为 v0.2 的生产默认行为
- **测试显式 `false`**——在 `setupTestEngine`、doctor E2E 等测试 fixture 里 `cfg.HOTLAutoClose = false`，保留 v0.1 流程作为回归网
- **门控点**：`InputFacts.HOTLAutoCloseEnabled` 由 `e.hotlAutoClose()` 注入，`ReviewPolicy.Decide` / `HardGuardrailReason` / `CompletionPolicy.Decide` / `handleReviewPass` / 2 处 recovery 调用点都按 flag 分支

**为什么不是直接切**：feature flag for major behavioral shift 是标准工程实践。生产默认开、测试 opt out，既让用户立即享受 HOTL，又保住 v0.1 测试作为"如果 flag 关掉，行为必须回到 v0.1"的回归契约。后续可以逐步把 v0.1 测试迁到 v0.2 语义，最终移除 flag。

### 7.2 各环落地清单

| 环 | 落地点 | 文件 |
|----|--------|------|
| 环 0 | ReviewPolicy: flag=true 时 `verdict=pass` → `checker_pass_auto_pass`；HardGuardrailReason 移除 workspace_diff（flag=true 时） | `internal/policy/policy.go` |
| 环 0 | CompletionPolicy 恢复 `quick_auto_complete` 分支（修掉 `PhaseEndSignal=="done"` 死代码） | `internal/policy/policy.go` |
| 环 1 | `handleReviewPass` 按 flag 分支：flag=true → `runAutoCloseLoop`（自主 apply+success）；flag=false → v0.1 MoveToUserReview | `internal/orchestrator/macro_loop.go` |
| 环 1 | recovery 2 处调用点注入 flag | `internal/orchestrator/quest_lifecycle.go` |
| 环 2 | event automation spawn_quest（已在 v0.1 末完成，此处保留） | `internal/orchestrator/event_automation.go` |
| 环 3 | `/goal` 模式 + CompletionChecker（独立小模型判定 done/continue） | `internal/orchestrator/goal_mode.go` |
| 环 4a | `EvtQuestApplied`/`EvtQuestSuccess` → 影响通知卡片（绿色，仅"查看影响"按钮） | `internal/notifications/policy.go` `event_subscriber.go` `cards.go` |
| 环 4b | `internal/connectors/` 包：Connector 接口 + ConnectorSubscriber + GitConnector（默认 OFF，只推 `gloop/<shortid>` 分支，永不推 base） | `internal/connectors/connector.go` `git.go` |
| 环 4b | `AutomationConfig.Connectors []string` + `ConnectorsConfig.Git` + Engine 启动时注册 | `internal/fsstore/automation.go` `config.go` `internal/orchestrator/engine.go` |
| 环 4c | inbox-triage skill 区分"待办区"（blocked/waiting_input/user_review）与"影响日志区"（success/applied） | `skills/gloop-inbox-triage/SKILL.md` |
| 环 4c | FocusView 影响日志分区（自主闭环 quest 按时间倒序，默认折叠 3 条，显示文件数+effect+时间）+ 全局文案对齐 HOTL（"等你裁决"→"需介入"、"去审核"→"去处理"、"用户审核"区→"需人工处理"） | `web/src/pages/FocusView.tsx` `questsBoardHelpers.ts` `QuestCard.tsx` `CurrentWorkStatus.tsx` `styles.css` |

### 7.3 验证

- `go test ./...`：全绿（含 30 个 v0.1 回归测试，靠 flag=false 保留）
- `go build`：成功，二进制 25MB
- 版本：`0.1.27` → `0.2.0`

### 7.4 安全约束（不变）

- GitConnector 默认 OFF，必须配置 `connectors.git.enabled=true` 才启用
- 启用后只 commit/push 到 `gloop/<shortid>` 专属分支，**永不推 BaseBranch**
- 所有 connector best-effort，失败只记日志，不回滚 quest 已闭环状态
- apply 仍走 SafetyCheck + backup，可回滚
