# gloop HITL 环节精简 Spec v0.2 — 减少冗余决策点，扩大自动闭环

> 版本：v0.2
> 状态：v0.1.2 候选实现已对齐（P0 / P1 主链路完成，P2 延后）
> 前置依赖：`hotl-spec-v0.1.md`（Policy Engine / ReviewPolicy / RecoveryPolicy 已落地）
> 定位：HOTL v0.1 的增量优化 spec，聚焦在"减少人工决策点数量"和"合并冗余操作"
>
> **v0.1 → v0.2 修订点**：
> - P0.1 从"实现项"改为"验收微调项"（review+apply 合并后端已落地，前端默认已开启）
> - P0.2 改名改语义：`quick_no_diff_auto_pass` → `quick_no_effect_auto_complete`，触发点在 quick execute phase 完成后，不是 review 阶段
> - P1 低风险 diff auto-pass 从 v0.1 主线移出，改为独立安全 RFC（安全模型迁移，不能混在体验优化里）
> - P1 design→execute auto spawn 补充幂等、child 状态、继承策略、失败重试契约
> - P2 user_review 超时：澄清现状（已有 BlockUserReviewTimeout scheduler），默认不改动

> **当前实现对齐状态（2026-06-22）**：
> - 已实现：review+apply 一步操作 UX、quick no-effect auto-complete、design auto-spawn execute、API/CLI/前端配置入口、策略审计字段、关键单测/E2E。
> - 部分实现：design auto-spawn 的 child 继承是保守继承，不完全复用原 design 工作区；spawn 失败记录为 quest note + `spawn_error`，未接外部通知。
> - 未实现：review action 选择 localStorage 记忆、user_review 超时策略化、低风险 diff auto-pass、Trust Tier 驱动 auto-pass。

---

## 一、背景与动机

### 1.1 现状

HOTL v0.1 已经落地了 Policy Engine、auto-pass、recovery policy、waiting_input 等基础能力，人机边界的基础设施已经就位。但实际跑下来，主流程上仍然存在几处**可以合并或消除的人工决策点**：

1. **最终审核 + 应用变更是两步独立操作** — 用户在 `user_review` 点 "通过"，quest 进入 `success`，如果有 workspace diff，还得再点一次 "应用变更"。两个决策点在用户心智上是同一个决策："我认可结果并要落盘"。

2. **User Review 是绝对必经之路** — 即使是 quick 模式、无 diff、低风险的 quest，也必须人点一下通过才能收尾。quick 模式的定位就是"快"，还要人确认一次违背了 quick 的初衷。

3. **Design → Execute 是手动断点** — design quest 完成后，用户要手动点 "按此方案执行" 才能生成 execute quest。设计-执行本是一条链路，中间却有个手动断点。

4. **user_review 超时 → blocked 是多余跳转** — user_review 超时了不直接处理，而是先转 blocked，用户要先 "解 blocked" 才能继续审核。状态机上"正确"，UX 上冗余。

5. **auto-pass 的口子开得太窄** — 目前只有 allowlisted 的 context_store automation 能 auto-pass，条件极其苛刻。大量"明明很安全"的 quest 还是要人点一下。

### 1.2 问题

- **主流程人工决策点多**：一个有 workspace diff 的普通 coding quest，从创建到落地要经过 **创建 → user_review → apply** 至少 3 次人工交互（不含异常路径）。
- **操作碎片化**：review 和 apply 在后端已经是串联调用，但前端 UX 上还可以更清晰（按钮文案、失败态、force apply 流程）。
- **quick 模式仍然要人审**：quick 模式的定位就是"快速简单任务"，还跳过法师评审直接到 user_review，等于没有任何质量 Gate 还要人点一下确认，形式主义。
- **Design → Execute 是手动断点**：design quest 完成后，用户要手动点 "按此方案执行" 才能生成 execute quest。设计-执行本是一条链路，中间却有个手动断点。
- **user_review 超时转 blocked 可能冗余**：审核超时了不直接处理，先转 blocked，用户要多一步操作才能继续。但目前线上这条路径触发量未知，需要先验证是不是真实问题。

### 1.3 目标

在 HOTL v0.1 的基础上，**主流程平均人工介入次数从 2~3 次降到 0~1 次**：

- P0.1：验收现有 review+apply 串联能力，做 UX 微调（按钮文案、失败态、force apply 流）
- P0.2：quick 模式 + 无副作用 → 自动完成（零人工介入的最短路径跑通）
- P1：design → execute 自动 spawn（设计流闭环，补充幂等/继承/失败契约）
- P2：user_review 超时策略化（先调研现状，再决定是否做）

> **关于低风险 diff auto-pass**：原本 P1 的"低风险变更自动通过"涉及安全模型迁移（从"有 diff 必须人审"到"高风险 diff 必须人审"），不在本 v0.1 spec 范围内，单独开安全 RFC 推进。

---

## 二、目标与非目标

### 2.1 目标

1. **最短路径零人工**：quick 模式 + 无副作用 + 无 diff → 自动完成，不进 user_review
2. **Review + Apply 体验打磨**：确认后端 `resolve + apply` 串联、前端 `apply !== false` 默认逻辑顺畅，优化按钮文案、失败态、force apply 流
3. **设计流闭环**：design quest 通过后可自动 spawn execute quest（带幂等、继承策略、失败处理）
4. **减少冗余状态跳转**：评估 user_review → blocked 的跳转是否必要，必要则策略化

### 2.2 非目标

- ❌ 不重做状态机（不在 `user_review` 和 `success` 之间插新状态）
- ❌ 不引入 AI agent 替代人做最终决策（auto-complete 是规则驱动，不是 AI 驱动）
- ❌ 不做跨 quest 的 workflow 编排（那是 Phase 4 的事）
- ❌ 不改变 HOTL v0.1 的安全底线（L2 / 外部副作用 / workspace diff → 永远要人审）
  - ⚠️ **注意**：低风险 diff auto-pass 属于安全模型迁移，不在本 spec 范围内，单独开安全 RFC
- ❌ 不做 notification 优化（那是 HOTL v0.1 的另一块，不在本 spec 范围）

---

## 三、设计原则

### 3.1 安全底线不动摇

HOTL v0.1 的硬护栏 **全部保留**，本 spec 不挪动任何安全边界：

| 护栏 | 动作 | 说明 |
|------|------|------|
| L2 权限（`allow_l2 = true`） | 永远 require_user | 高权限必须人审 |
| 外部副作用（`effect_type = external_side_effect`） | 永远 require_user | 外部影响必须人审 |
| Workspace diff（`has_workspace_diff = true`） | 永远 require_user | 有工作区改动必须人审 |

> **关于"低风险 diff 自动过**：这是安全模型迁移（从"有 diff 必须人审"到"高风险 diff 必须人审"），不在本 v0.1 spec 范围内。单独开安全 RFC 评审，不混在体验优化里推进。

本 spec 所有新增的自动能力，都只在 "无 diff + 无 L2 + 无外部副作用" 的安全空间内做文章。

### 3.2 一个决策 = 一次操作

用户心智上的一个决策（"我认可这个结果"），应该只需要一次点击。不要因为内部状态拆分（review state / apply state），让用户点两次。

### 3.3 默认安全，可配置放宽

所有新增的自动能力**默认关闭**，需要显式配置才开启。用户要主动选择"让这个自动化任务自动通过"，而不是反过来。

### 3.4 渐进式，可回滚

每项优化都可以独立开关、独立上线。出了问题可以单独关闭某一项，不影响其他。

---

## 四、现状盘点：所有 HITL 环节

### 4.1 主流程必现环节

| # | 环节 | 状态变化 | 触发条件 | 人做什么 | 能否合并/消除 |
|---|------|---------|---------|---------|-------------|
| 1 | 创建委托 | (none) → pending | 用户主动创建 | 填需求、选参数 | 不能（总得有输入） |
| 2 | 启动委托 | pending → running | auto_start=false 时 | 点"启动" | 已通过 auto_start 优化（默认 true） |
| 3 | 最终审核 | reviewing → user_review | 法师评审结束 | 点 通过/返工/拒绝 | **可通过 auto-pass 部分消除** |
| 4 | 应用变更 | success + diff | quest 成功且有 workspace diff | 点 应用/放弃 | **可与最终审核合并** |

### 4.2 异常路径偶发环节

| # | 环节 | 触发条件 | 人做什么 | 能否优化 |
|---|------|---------|---------|---------|
| 5 | Agent 提问 | warrior 用 ask_user | 回答问题 | 可批量/智能推荐（P3） |
| 6 | 阻塞恢复 | 错误/超时/预算耗尽 | 继续 / 转审核 / 取消 | recovery policy 已覆盖常见场景 |
| 7 | 返工僵局 | 反复返工无进展 | 终审 | 已是安全阀门，不能消 |
| 8 | 应用安全拦截 | apply 前扫出高危变更 | 确认风险并强制应用 | 安全护栏，不能消 |
| 9 | 应用失败 | apply 失败 | 排查重试 | 可自动重试（P2） |
| 10 | 设计→执行 spawn | design quest 成功 | 手动 spawn execute | **可自动（P1）** |
| 11 | user_review 超时转 blocked | user_review 超时 | 解 blocked 再审核 | **可消除冗余跳转（P2）** |

---

## 五、详细设计

### 5.0 前置确认：现有能力盘点

HOTL v0.1 已经有的，本 spec 直接复用：

| 能力 | 状态 | 本 spec 怎么用 |
|------|------|--------------|
| ReviewPolicy / auto-pass | ✅ 已落地 | 扩 policy 覆盖范围 |
| HardGuardrail（L2 / 外部副作用 / workspace diff 硬护栏） | ✅ 已落地 | **保留，不动** |
| `verdict=pass + apply=true` 一步操作 | ✅ 已支持（CLI/API） | **前端 UI 上接起来** |
| RecoveryPolicy | ✅ 已落地 | 不改动 |
| Trust Tier | ✅ 已落地（保守） | v0.1 暂不用于 auto-pass |
| waiting_input | ✅ 已落地 | 不改动 |

### 5.1 P0.1：Review + Apply 体验验收与微调

#### 5.1.1 现状（已验证）

后端和前端的 review+apply 串联**已经落地**，不是新能力：

- **后端**：`POST /quests/:id/resolve` 接口在 `verdict=pass && req.Apply` 时，先 `ResolveUserReview` 再 `ApplyQuest`，apply 有 warnings 返回给前端。
- **前端**：`confirmReview()` 里 `reviewAction.verdict === 'pass'` 时，`payload.apply = reviewAction.apply !== false`（默认 true）。
- **安全检查流**：apply 命中 high severity 时，前端通过 `warningsFromError` 解析，弹 `setApplyWarning`（带 `fromReview: true` 标记），用户可以强制应用。

也就是说，spec 里写的"P0 新能力"大部分已经存在，本项改为**验收 + UX 微调**。

#### 5.1.2 验收清单（按当前实现更新）

| 验收项 | 预期 | 现状 |
|--------|------|------|
| user_review 按钮默认"通过并应用" | ✅ 符合预期 | ✅ 已实现：有 workspace diff 时主按钮为"通过并应用" |
| 有 diff 时按钮文案明确提示"会应用" | 文案应包含"应用"语义 | ✅ 已实现：确认弹窗说明"先通过终审，再立即应用工作区变更" |
| 无 diff 时按钮就是"通过" | 不显示 apply 相关文案 | ✅ 已实现：无 diff 时显示普通"通过" |
| apply 失败时错误提示清楚 | 告诉用户 "通过了但 apply 失败" | ✅ 已实现：安全检查拦截 sheet 文案为"评审已通过，应用被安全检查拦截" |
| force apply 流程顺畅（从 review 触发的） | 用户能在 warnings sheet 里直接强制应用 | ✅ 已实现：`fromReview` 场景可在同一 sheet 里强制应用 |
| "仅通过（不应用）" 选项存在 | 下拉或 checkbox | ✅ 已实现：有 diff 时提供"通过但不应用"按钮 |
| 选择记忆（localStorage） | 记住用户上次选了仅通过还是通过+应用 | ❌ 未实现：当前不记忆用户选择，保持每次显式选择 |

#### 5.1.3 微调方向（基于验收结果决定是否做）

- 按钮文案从"通过"改成"通过并应用"（有 diff 时），更明确
- 加下拉选项"仅通过（不应用）"
- apply 失败态的提示优化：明确区分"评审已通过"和"应用失败"是两件事
- fromReview 的 force apply 完成后回到 success 状态，提示"已通过并强制应用"

当前未做 localStorage 记忆，原因是 review/apply 是有副作用决策，默认每次显式确认更稳。后续如果要做，也应只记忆 UI 偏好，不改变默认安全语义。

#### 5.1.4 不做的事

- 不新增后端接口（能力已有）
- 不改变状态机
- 不挪动安全护栏

### 5.2 P0.2：Quick 模式 + 无副作用 → 自动完成

#### 5.2.1 问题

quick 模式**跳过法师评审**（`skipMageReview = true`），warrior 跑完直接进 user_review。也就是说：
- 没有 AI 评审环节，质量全靠人
- 但人在 user_review 能看到的也就是 warrior 输出
- 对于无副作用（no effect）的 quick quest，人要点的那个"通过"按钮纯形式主义

这是最容易做、也最安全的零人工闭环场景。

#### 5.2.2 方案

**不是** "quick review auto-pass"（因为 quick 根本没有 review 阶段）。

而是：在 quick execute phase 完成后、准备 MoveToUserReview 之前，加一层 **CompletionPolicy** 判断，如果满足"无副作用"条件，直接 `CompleteQuest` 进入 success，不进 user_review。

**Policy 名称**：`quick_no_effect_auto_complete`

**命中条件（全部满足）**：
1. `skip_mage_review = true`（本质条件，包含 quick 模式和 MageID 为空两种情况）
2. `effect_type = none`（无任何副作用）
3. `has_workspace_diff = false`
4. `allow_l2 = false`
5. `has_external_output = false`（防御性断言，和 effect_type=none 双重校验）
6. `rework_count = 0`（一次跑完，没返工过）
7. `auto_complete_allowed = true`（automation 配置显式授权）

满足 → `action = auto_complete`（直接 success）

#### 5.2.3 触发点

在 macro_loop 的 quick path 里，`MoveToUserReview` 之前插入 policy 判断：

```go
if skipMageReview {
    canDecideCompletion := e.refreshDiffSummaryForPolicy(ctx, qid)
    if latest, err := qs.LoadQuest(qid); err == nil {
        q = latest
    }
    decision, facts := e.decideCompletionPolicy(q)
    if decision.Action == policy.ActionAutoComplete {
        e.publishPolicyDecision(q, decision, facts)
        // 直接 CompleteExecute，不进 user_review
        if _, err := e.questService.CompleteExecute(qid, quest.CompleteExecuteOptions{...}); err != nil {
            // fallback 到 user_review
        }
        return warriorOutput, q, true
    }
    // 否则走原逻辑：MoveToUserReview
    ...
}
```

#### 5.2.4 默认值与开关

- **默认：关闭**（quick 模式虽然定位快，但 auto-complete 是行为变更，默认保守）
- **配置字段名**：`allow_quick_auto_complete`（与 `allow_hotl_auto_pass` 保持命名风格一致）
- 配置层级：
  - automation 级别：`allow_quick_auto_complete = true`（在 automation config 里）
  - quest override：创建 quest 时可指定（和其他 automation 字段一样的 override 机制）
  - 暂不做 global 级开关（先用 automation 级验证，需要的话再加）
- 为什么默认关闭：quick 模式的"快速"是指"快速出结果"，不是"快速闭环"。自动闭环是额外授权，需要用户主动开。

#### 5.2.5 暴露入口（P0.2 必做项）

配置字段加了之后，必须同步暴露到以下入口，否则用户用不了：

| 入口 | 说明 |
|------|------|
| **HTTP API** | `POST /api/automations` create / `PATCH /api/automations/:id` 支持 `allow_quick_auto_complete`；response 里返回 |
| **CLI** | `gloop automation edit <id> --allow-quick-auto-complete` / `--no-allow-quick-auto-complete` |
| **前端自动化设置页** | automation 编辑页加开关，在 HOTL / 自动闭环分组里 |
| **Quest create override** | `POST /api/quests` 和发起委托表单支持一次性覆盖 |

> 注意：`allow_quick_auto_complete` 和 `allow_hotl_auto_pass` 是两个独立开关。前者控制 execute 阶段 auto-complete，后者控制 review 阶段 auto-pass。两者互不依赖，可以单独开启。

#### 5.2.6 审计

和 ReviewPolicy auto-pass 一致的审计契约，但字段独立、语义不同：

- `policy.decision` 事件，`policy_name = "quick_no_effect_auto_complete"`
- `policy_action = "auto_complete"`
- `auto_completed_by_policy = "quick_no_effect_auto_complete"`（独立字段，不复用 auto_passed_by_policy）
- `finalized_by = "policy"`
- `review_verdict` 字段：**空**（因为没有 review 阶段，不要伪造 pass）

> ⚠️ **关键原则**：不伪造 review_verdict=pass。quick auto-complete 是"无副作用任务自动完成"，不是"评审自动通过"。审计里用独立字段 `auto_completed_by_policy`，和 `auto_passed_by_policy` 严格区分，避免统计和信任等级计算混淆。

#### 5.2.6 与现有 ReviewPolicy 的关系

- ReviewPolicy 作用于 **review phase 完成后**（法师评审完了）
- CompletionPolicy 作用于 **execute phase 完成后且无后续 review 阶段时**（quick 模式、或未来的单阶段委托）
- 二者共享同一套 InputFacts 和 Decision 结构
- 二者都受 HardGuardrail 约束（L2 / 外部副作用 / workspace diff → 永远 require_user）

#### 5.2.7 为什么只做 quick，不做 standard

- quick 模式本身跳过法师评审，说明用户对这类任务的质量预期就是"够用就行"
- standard 模式有法师评审把关，法师都过了的任务要不要人审，是另一个问题（属于低风险 diff auto-pass 的安全 RFC 范畴）
- 先从最安全的场景切入，跑通数据再扩

### 5.3 P1：Design → Execute 自动 Spawn

#### 5.3.1 问题

design quest 通过后，用户要手动点 "按此方案执行" 才能生成 execute quest。设计和执行本是一个完整流程，中间却有个手动断点。

这是明确的 HITL 冗余环节：用户已经同意了方案，接下来的执行就是顺理成章的事，不需要再点一次。

#### 5.3.2 方案

加一个 `auto_spawn_execute` 选项：

- design quest 配置 `auto_spawn_execute = true`
- design quest 成功（auto-pass 或人工通过都算）后，**自动创建**对应的 execute quest
- execute quest 保守继承 design quest 的 agent、intensity、workspace mode 和 base working dir；具体继承/重建规则见 5.3.5
- execute quest 的 `parent_quest_id` 指向 design quest
- execute quest 默认 `auto_start = true`（自动启动）

#### 5.3.3 幂等契约（必须）

**同一 design quest 最多 spawn 一个 execute child。**

- design quest 上存 `child_execute_quest_id` 字段
- spawn 前先检查该字段，非空则跳过（幂等）
- spawn 成功后写入该字段
- 如果 child 被用户取消了，不自动重 spawn（用户手动决定要不要再来一次）

**幂等兜底：** 除了字段检查，还可以用 `parent_quest_id = {design_id} + type = execute` 反查 quest 列表作为兜底校验。
即使 `child_execute_quest_id` 因异常没写进去，反查也能发现已经 spawn 过了，避免重复。

#### 5.3.4 触发位置与实现方式

spawn 是 design quest 成功后的**后置副作用**，不参与 quest 本身的状态裁决：

- **触发位置**：after-commit subscriber / post-success action
  - 在 `CompleteQuest` 状态迁移成功、事务提交之后触发
  - 不在状态机内部，也不在 `CompleteQuest` 方法里（避免 spawn 失败影响 quest 终态）
- **与状态机的关系**：spawn 不参与 quest 的 success 判定
  - design quest 的 success 只由 review verdict 决定，和 spawn 成不成功没关系
  - spawn 是"顺便做的事"，失败了 design quest 还是 success
- **实现方式二选一**（推荐前者）：
  1. 事件订阅：quest success 事件触发后，异步执行 spawn
  2. 宏循环后置：macro_loop 里 CompleteQuest 成功返回后，再调 spawn 逻辑

#### 5.3.5 child 状态与继承策略

child quest 的字段继承规则（用实际字段名）：

| 字段 | 继承策略 | 说明 |
|------|----------|------|
| `warrior_id` | 继承 | 用同一个 warrior 执行 |
| `mage_id` | 继承 | 用同一个 mage 评审 |
| `intensity` | 继承 | quick design 不允许 auto-spawn；手动 spawn 会按父级强度继承 |
| `max_rework` | 未单独继承 | 当前 child 跟随全局/创建默认配置 |
| `workspace_mode` | 继承 | |
| `workspace_path` | 不直接继承 | 重新按父级 `base_working_dir` 和 `workspace_mode` 创建隔离工作区 |
| `base_working_dir` | 继承 | 传入父级 base working dir |
| `base_branch` | 间接继承 | 由新工作区准备逻辑重新记录 |
| `base_commit` | 间接继承 | 由新工作区准备逻辑基于当前 base 重新记录 |
| `created_by` | 不直接继承 | 当前按新 quest 创建来源记录 |
| `query` / title | design doc 的 `execute_prompt` | 不额外拼接固定前缀 |
| `description` | 未单独写字段 | 执行语义进入 child query / prompt |
| `parent_quest_id` | 设为 design quest 的 ID | 反向关联 |
| `type` | 设为 `execute` | 不从 design 继承，强制改写 |
| `allow_quick_auto_complete` | **不继承**，默认 false | child 不自动继承 quick auto-complete |
| `auto_spawn_execute` | **不继承**，默认 false | 防止递归 spawn |
| `allow_l2` | **不继承** | quest 本身不带该权限，仍由 automation / command policy 控制 |
| `has_external_output` / 外部副作用类 | **不继承** | 由 child 实际产出重新判定 |
| `automation_id` | 当前不直接继承 | child 作为独立执行 quest 创建 |

> ⚠️ 安全原则：**automation 权限类字段（allow_l2、外部副作用、context automation 标记）默认不继承**，执行阶段收回到最严格。
> 用户如果确实需要 L2 执行，可以手动给 execute quest 开。
> 注意：不继承的是 automation 权限字段，不是 agent executor（warrior/mage 是继承的）。

#### 5.3.6 失败重试契约

spawn 是 design quest 成功后的**后置动作**，失败不影响 design quest 本身的终态：

- design quest 始终是 success（即使 spawn 失败了）
- spawn 失败时：
  1. 记录 `spawn_error` 在 design quest 的 metadata 里
  2. 记录一条 `quest.note` 事件，`scope = auto_spawn_execute`
  3. 前端详情页显示 spawn 失败信息；底部仍可用"按此方案执行"手动重试
- 重试：用户手动点"按此方案执行"重试；系统**不自动重试**（默认不重试，避免死循环）
- 未实现：外部通知（飞书等）未接入本 spec 的 spawn 失败路径。

#### 5.3.7 配置位置与默认值

- 创建 design quest 时的表单里加一个开关："方案通过后自动执行"
- automation 配置里也加同样的开关
- **默认：关闭**（用户显式开启才自动 spawn）
- 原因：自动 spawn 会消耗算力、可能产生副作用，默认保守

#### 5.3.8 和 quick auto-complete 的组合

> 注意：design quest 是 quick 模式的话，走的是法师评审 → user_review 路径，**不**走 quick_no_effect_auto_complete（因为 design quest 一定有输出产物，不是 no effect）。

但如果 standard design quest 开启了 auto-pass（未来低风险 diff auto-pass RFC 落地后）+ auto_spawn_execute，整条链路可以做到：

```
用户创建 design quest
  → warrior_design 跑方案
  → mage_design_review 评审 → pass
  → 命中 auto-pass policy → 自动通过
  → 触发 auto_spawn_execute → 创建 execute quest
  → execute quest 自动启动
  → warrior_execute 执行
  → mage_execute_review 评审 → pass
  → 命中 auto-pass policy → 自动通过
  → user_review → 用户最终验收
```

设计到执行的中间环节全部零人工，用户只需要最后验收一次。

### 5.4 P2：User Review 超时策略调研与优化

#### 5.4.1 现状（先澄清）

scheduler 里已经有 `BlockUserReviewTimeout`：user_review 状态的 quest 超时后自动转 blocked。

但有几个问题需要先搞清楚，再决定改不改：

1. **超时时间是多少？** 配置在哪？默认值？
2. **实际触发频率有多高？** 有没有统计数据？
3. **用户真的被这个困扰吗？** 还是说"超时转 blocked"是合理的安全机制？
4. **blocked 之后用户的操作路径是什么？** 是"解除阻塞 → 回到 user_review → 再点通过"吗？

在拿到这些数据之前，**不做任何改动**。P2 先做调研。

#### 5.4.2 可能的优化方向（调研后选）

**方向 A：超时不转 blocked，直接按默认策略处理**

- `timeout_action = auto_reject`：超时自动拒绝
- `timeout_action = keep_waiting`（默认）：不做任何操作，一直等
- 适用于：超时触发频率很高、但大部分 quest 最终都是用户手动处理的场景

**方向 B：blocked 状态下直接显示 review 操作**

- 用户在 blocked 页面就能直接看到评审意见、直接点通过/拒绝
- 不需要先"解除阻塞"再操作
- 适用于：超时转 blocked 是必要的安全机制，但操作路径可以简化

**方向 C：维持现状，只优化文案和提示**

- 告诉用户"为什么 blocked"、"怎么继续"
- 适用于：问题没那么严重，不值得改逻辑

#### 5.4.3 为什么放 P2

- 问题是否真实存在还不确定，需要数据验证
- 即使存在，ROI 也不如前面几项高
- 改动可能涉及状态机变更，风险相对大

---

## 六、实施计划

> 状态：P0 / P1 已落地并通过验证；P2 保持 data-driven follow-up，不阻塞 v0.2 收工。

按优先级分三批上线，每批都可以独立发布。

### 第一批（P0，~1 周）— 立竿见影

- [x] **5.1 Review + Apply 体验验收与微调**：验收现有能力，完成按钮文案、"通过但不应用"、失败态和 force apply 流
- [ ] **5.1 localStorage 选择记忆**：未做；出于副作用决策安全性，暂不作为 v0.1.2 必需项
- [x] **5.2 Quick 模式无副作用自动完成**：新增 `quick_no_effect_auto_complete` policy（CompletionPolicy）
- [x] **配置入口（P0.2 必做）**：`allow_quick_auto_complete` 暴露到 HTTP API / CLI / 前端设置页 / quest create override
- [x] **测试（P0.2 必做）**：policy 单元测试 + 状态机测试 + service 测试 + E2E 验证
- [x] 配套：auto-complete quest 在看板/详情页有明确标识（"已自动完成"标签，区别于 auto-pass）
- [x] 额外 UI 优化：创建委托 / 自动化从内部 `execute/design` 字段改为"直接执行 / 先设计再执行 / 只出方案"流程选择

### 第二批（P1，~2 周）— 扩大自动闭环

- [x] **5.3 Design → Execute 自动 spawn**：`auto_spawn_execute` 配置 + 自动创建逻辑 + 幂等契约
- [x] 配套：spawn 失败的元数据、quest note、详情页提示和手动重试入口
- [x] 配套：child quest 的状态关联展示
- [x] 安全边界：quick design 第一版不自动 spawn，避免 quick 跳过评审和 design review 边界混淆
- [ ] 外部通知：spawn 失败未接入 NotificationPolicy，后续如需要再补

### 第三批（P2，~1 周）— 体验打磨（数据驱动）

- [ ] **5.4 User review 超时调研**：拉数据、确认问题存在性和严重度（deferred / data-driven follow-up）
- [ ] 根据调研结果选择优化方向并实施（deferred / data-driven follow-up）

### 移出本 spec / 独立 RFC

- ❌ ~~低风险变更自动通过（Change Risk Classification）~~ → 独立安全 RFC，属于安全模型迁移
- ❌ ~~Trust Tier 用于 auto-pass~~ → HOTL v0.1 已暂缓，后续再说

---

## 七、风险与边界

### 7.1 安全风险

| 风险 | 严重度 | 缓解措施 |
|------|--------|---------|
| quick auto-complete 漏判（把有副作用当成无副作用） | 中 | 1. HardGuardrail 三层校验（L2/外部副作用/workspace diff）全部命中 require_user<br>2. policy decision 有独立 unit test<br>3. 默认关闭，需显式开启<br>4. 所有 auto-complete 都有审计日志 |
| auto_spawn_execute 意外创建大量 quest | 中 | 1. 默认关闭<br>2. 幂等保证（每个 design 最多一个 child）<br>3. spawn 失败不阻塞、不重试 |
| auto-spawn 的 execute quest 继承了过高权限 | 高 | 1. 权限类字段（allow_l2、external_side_effect）不继承，默认最严格<br>2. 有审计可追溯 |

### 7.2 体验风险

- **"通过并应用" 可能让用户困惑**：为什么通过了还要应用？→ 按钮文案要清晰，下拉选项里明确"仅通过"。
- **auto-complete 的 quest 可能被忽略**：用户没注意到 quest 已经自动完成了 → 看板上要有明确标识，通知策略可以针对 auto-complete 做特殊处理（比如静默通知）。
- **auto-spawn 后用户不知道去哪找 execute quest** → design quest 详情页直接显示 child quest 卡片/链接。

### 7.3 不在本 spec 范围

- 变更风险分级 / 低风险 diff auto-pass（安全模型迁移，独立 RFC）
- Agent 提问的批量处理 / 智能推荐（属于交互体验优化，另开 spec）
- Notification 优化（属于 HOTL v0.1 的另一块）
- Apply 失败的智能自愈（属于更高级的自动化，后续再考虑）
- Trust Tier 用于 auto-pass（HOTL v0.1 刻意暂缓了，本 spec 也不碰）

---

## 八、成功指标

上线后跟踪以下指标，验证优化效果：

| 指标 | 现状（估） | 目标 | 对应项目 |
|------|-----------|------|---------|
| 平均每个 quest 的人工介入次数 | 2~3 次 | 0.5~1 次 | 整体 |
| user_review 中点击"通过+应用"占比 | 待验收 | > 70%（说明合并操作符合预期） | P0.1 |
| quick 模式 quest 的 auto-complete 率 | 0% | > 30%（无副作用 quick quest 自动闭环） | P0.2 |
| 设计→执行自动 spawn 使用率 | 0% | > 30% 的 design quest 开启 | P1 |
| user_review → blocked 跳转次数 | 待调研 | （调研后定目标） | P2 |

---

## 九、相关文件

- `docs/hotl-spec-v0.1.md` — HOTL 基础 spec（前置依赖）
- `internal/policy/policy.go` — ReviewPolicy / RecoveryPolicy（新增 CompletionPolicy）
- `internal/policy/facts.go` — 输入事实构建
- `internal/domain/quest/service.go` — quest 状态变更服务（CompleteQuest / MoveToUserReview）
- `internal/orchestrator/macro_loop.go` — macro loop 调度（quick path 插入 policy 判断）
- `internal/orchestrator/scheduler.go` — scheduler（BlockUserReviewTimeout）
- `internal/server/api_quests.go` — /resolve 接口（review+apply 串联）与 quest create override
- `internal/server/api_automations.go` — automation 策略配置 API
- `internal/server/api_test.go` — automation 策略字段、quest create override、multipart override 回归测试
- `internal/orchestrator/engine_e2e_test.go` — quick auto-complete 与 auto-spawn E2E 回归
- `web/src/components/CreateQuestSheet.tsx` — 发起委托表单的流程与一次性策略开关
- `web/src/components/CreateAutomationSheet.tsx` — 创建 automation 的流程与策略开关
- `web/src/pages/Automations.tsx` — automation 编辑页策略开关
- `web/src/pages/QuestDetail.tsx` — 详情页 + user_review 操作栏
- `web/src/domain/questSelectors.ts` — 状态 selectors
