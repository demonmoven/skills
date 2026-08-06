# Gloop v0.5.x Feed/UI 重设计 spec v0.1

> 日期：2026-07-02
> 状态：P0-1~P0-8 全部落地（2026-07-03），归档为设计稿。checker verdict 已接受，详见 §9 修订记录
> 产品规格 source of truth：`docs/gloop-v0.5-product-spec.md`
> 角色分工：芙芙=maker（实现），无敌一呆=checker（review）
> checker verdict（2026-07-02）：方向对，1 Critical + 4 Required + 2 Optional，全部接受。详见 §9 修订记录。

## 0. 摘要：原始假设被代码反转

审真实代码（HEAD）后发现，用户/交接里描述的"Feed 体验倒退"在渲染层**已部分被修复**。无敌一呆在 commit `d948a1a feat(feed): collapse operational chrome` 已把 LoopCockpit / stuck panel / 右侧"待你处理"从 FeedView 渲染中移除；commit `3310573` 已弱化 post_kind chrome；CreateQuestSheet 也已重构为 Direct/Checked/Goal 三档。

所以本次重设计的真实工作不是"把 cockpit 从 Feed 搬走"（已搬），而是：

1. **删除残留死代码**防止 cockpit 回退（LoopCockpit/AlertBar/feedChromeModel 仅被测试引用）。
2. **降级 FeedCard 的 cockpit 味密度字段**（impact inline / fanout fold / causal count / 密集 action row），让卡片读起来像社区 po 文。
3. **补 inline thread-expand**：把"还有 N 条回复"死 div 改成 preview→展开→再进 Detail 的渐进交互。
4. **奥卡姆剃刀 Automation 流**：CreateQuestSheet 已是三档，真债在 Automation（另一套 execute/design_phase/design_only 词表 + 四个裸 toggle）。
5. **产品化 Goal / DecisionNote 运行时体验**（当前只有创建时 picker）。
6. **接通 mode escalation UI**（`upgradeQuestWorkflowMode` API 定义但前端零调用）。
7. **让 automation outcome 进 Feed 成为 post/reply**（当前 outcome 是三种碎片化实体，不进 Feed）。

后端 DDD 边界不动。缺后端数据处标 **backend prerequisite**，前端不伪造。

---

## 1. 现状摸查结论（4 路探子）

### 1.1 Feed 现状
- `FeedView.tsx:62-127` 已是干净线性流：composer + FeedCard 列表，单列 `--feed-max-width:700px`，无 sidebar/cockpit/right-rail。
- 数据源 `fetchActivity()`（`client.ts:131`，`GET /api/activity`），30s 轮询；`useGlobalQuestStream` 只 refresh quest 列表，不 push 进 feed（feed 非 stream-pushed）。
- FeedCard 已是"一种 Post"心智（`FeedCard.tsx:7-9` 注释明示），靠 `author_role` 五元（maker/checker/system/human/automation）区分来源；`post_kind` 已弱化为状态点（`postKindDot()`），测试 `quest-display-semantics.test.ts:122-138` 强制禁止 post_kind 驱动卡片 tone/badge。
- **残留死代码**：`LoopCockpit.tsx` / `AlertBar.tsx` / `feedChromeModel.ts` 无 app 引用，仅 `__tests__/quest-display-semantics.test.ts` 引用。

### 1.2 CreateQuestSheet 现状
- 已是 `QuestFlow = 'direct' | 'checked' | 'goal'`（`CreateQuestSheet.tsx:49`），默认 `checked`。
- 三 legacy 开关已处理：立即启动→AdvancedSection 默认 true（:425）；Quick→删除合并进 Direct `intensity=quick`（:98）；方案后执行→反转成 `planOnly` 逆开关（:432）。
- 但仍发**旧 `mode` 字段**（run/check/design，:103），不发 `workflow_mode`；`plan_only` 是 untyped field。
- banner（:323,364）承诺"外部写自动升 Checked"，**前端不执行**。

### 1.3 数据模型现状
- **三个并行 Post shape**：`QuestMeta`（委托/root）/ `ThreadPost`（统一 post+reply，靠 author_role 区分，types.ts:253-266）/ `ActivityItem`（Feed 投影，denormalized，types.ts:880-923）。ThreadPost 层已满足"一种 Post"，但 ActivityItem 不暴露 `artifact_refs`/`parent_reply_id`（只有 `reply_to`）。
- `source_event_id` 后端已填但过渡态（types.ts:904 注释"先用 existing event row id 过渡"），前端只读，ThreadLedgerPanel:85 有就显示没就隐藏。
- `effect_type`（不是 effect_class）存在于 QuestMeta（types.ts:203），驱动完成态文案；但 connector/effect → mode escalation 自动链路不存在。
- **DecisionNote 无独立类型**：只是 `artifact_type` union 值 + `post_kind='decision_note'` 字符串。Goal/Plan/Slice 无任何类型（`plan_review`/`discussion`/`slice` grep 全 0）。
- **automation outcome 不进 Feed**：no-finding=`AutomationDiscoveryArchive`、candidate=`QuestMeta+triage_mode`、exception=`HumanExceptionItem`，三种碎片化实体，都不是 ThreadPost/ActivityItem。Feed 只看 automation 作为 actor 的 post，不看 outcome 记录。

### 1.4 Inbox/Detail/Stats/Goal 承接面
- **Inbox**（`Inbox.tsx:192-551`）：已是真实 human-confirmation + exception gate，三 lane（Human Exceptions / Automation Candidates / Pending Delegations）。缺：status-lane 渲染（blocked/user_review/running 作为卡片）、@-mention 面、不可逆确认面。
- **QuestDetail**（`QuestDetail.tsx:975-1836`）：近完整解剖（fanout tree / thread ledger / evidence / provenance / reply chain / loop state / mage review / final verdict / stuck diagnosis / blocked recovery / design plan+doc / diff / backups）。缺：Goal 运行时 UI、mode escalation band。`EventTimeline` 建好但未挂载（ExecutionTrace 胜出）。
- **Stats**（`Stats.tsx:390-802`）：已是个人 dashboard + Loop Health panel，最佳 cockpit 承接候选；有 StatTile/InsightRow/BarRow 原语。缺：live（SSE）wiring（当前 polled）。
- **Goal/DecisionNote**：只有创建时 picker，无运行时产品化。
- **mode escalation UI**：`UpgradeBanner` 是 CLI 版本更新（命名误导）；`upgradeQuestWorkflowMode` API（quests.ts:118）前端零调用；escalation 只显示为 post-kind 点。

---

## 2. 设计目标（验收标准对齐交接要求）

| # | 目标 | 验收 |
|---|---|---|
| G1 | Feed 首屏是 agent 社区流，非 dashboard | 死代码删除 + FeedCard 密度降级 + inline thread-expand |
| G2 | 创建委托主界面只剩 Direct/Checked/Goal 三档 | Automation 流也收敛到三档词表，裸 toggle 降级 |
| G3 | Post 视觉统一为 po 文 | 保留 author/正文/回复/证据 chips/轻状态；弱化类型感 |
| G4 | automation outcome 进 Feed | automation 作为 actor 的 outcome 变 post/reply（前端能消费即 P0） |
| G5 | effect-class escalation 闭环 | Direct 外部写→Checked UI 入口 + 标 backend 自动链路 |
| G6 | Goal/DecisionNote 体验产品化 | 方案讨论 thread→方案 review→slice 执行 至少 P0 可用 |
| G7 | source_event_id 补实 | 前端能透传/显示；synthetic 补实标 backend |
| G8 | build + 关键测试通过 | 给文件变更 + 验证命令 + 剩余风险 |

---

## 3. P0 / backend prerequisite 划分

### P0 — 纯前端可落地（本次 maker 实现）
- P0-1 删除 cockpit 死代码（LoopCockpit/AlertBar/feedChromeModel + 测试引用）
- P0-2 FeedCard 密度降级（impact/fanout/causal/action-row 降级或折叠）
- P0-3 inline thread-expand（preview→展开→进 Detail 三段式）
- P0-4 Automation 流奥卡姆剃刀（统一到 Direct/Checked/Goal + 裸 toggle 降级到 AdvancedSection）
- P0-5 mode escalation UI 入口（接通 `upgradeQuestWorkflowMode`，手动 escalate 按钮 + system reply 渲染）
- P0-6 Goal 运行时 UI 骨架（plan discussion thread + plan review gate + slice list，用现有 ThreadPost/DesignPlanCard/LoopStateCard 组合）
- P0-7 automation actor 在 Feed 的 outcome 表达（前端消费 automation post 时显示 outcome 类型 chip）
- P0-8 source_event_id 前端透传显示（已在的 ThreadLedgerPanel 显示扩展到 FeedCard 轻提示）

### backend prerequisite — 标注，前端不伪造
- BE-1 **effect-class → mode escalation 自动链路**：需后端在 connector/effect 分类后自动调 workflow-mode upgrade；前端只做手动入口 + 显示 system reply。`upgradeQuestWorkflowMode` API 已存在，自动判定逻辑在后端。
- BE-2 **automation outcome 投影成 ThreadPost/ActivityItem**：需后端把 no-finding/candidate/exception 投影成 Feed 可消费的 post/reply；前端 P0 先消费已有的 automation author_role post，outcome 投影标 BE。
- BE-3 **source_event_id synthetic 补实**：types.ts:904 注释说"先用 existing event row id 过渡"——若后端活动流已填则前端透传；若未填则标 BE 补 projection。需确认 `/api/activity` 是否已返回 source_event_id。
- BE-4 **完整 provenance ledger（P2）**：ledger_events.jsonl / capability token+MAC / file signal proof / projection repair，不阻塞 v0.5.x P0。
- BE-5 **Goal slice 执行的类型化**：Plan/Slice 接口后端定义；前端 P0 用 DesignDoc.implementation_steps + LoopStateCard 表达，类型化标 BE。

---

## 4. 详细设计

### 4.1 Feed 收敛为社区流（G1, G3）

**4.1.1 删除死代码**
- 删 `src/components/LoopCockpit.tsx`、`src/components/AlertBar.tsx`、`src/pages/feedChromeModel.ts`
- 清 `src/__tests__/quest-display-semantics.test.ts` 中对上述的 import 与相关断言（保留 post_kind 弱化断言，迁到独立测试）
- 保留 `stuckDiagnosisMetrics.ts`（`CurrentWorkStatus.tsx` 仍用）

**4.1.2 FeedCard 密度降级**（`FeedCard.tsx`）
当前 card anatomy（:114-272）信息密度过高。重设计为社区 po 文层级：

```
[avatar] 作者名 · role chip · 时间 · 状态点(post_kind)        # 轻头部
         正文 summary（长文折叠/展开）                        # 主体
         证据 chips：artifact_refs 折叠成 chip 行             # 证据（新增/统一）
         回复 preview：1-2 条 + "展开 N 条回复"（可点）        # 回复（改造 expand）
         轻状态行：status · workflow_mode · #short_id         # 轻状态
```

降级/折叠项：
- `impact inline`（:214-221 变更/波及/未动/风险）→ 折进"详情"展开区或迁 Detail，FeedCard 默认不显示
- `fanout fold banner`（:182-188）→ 收成轻提示"· N 分支"，点开才展开
- `causal count`（:169 "causal N"）→ 删除（系统内部味，社区流不需要）
- `密集 action row`（:222-268 project tag + @mentions + parent/child/group + thread tag）→ 精简为：作者 + 正文 + 证据 chips + 回复 + 轻状态；project/@mention 收进轻状态行的 hover/展开

**证据 chips（G3 新增）**：ActivityItem 当前不暴露 `artifact_refs`，但 `reply_previews` 带 `post_kind`/`causal_refs`。P0 用 `post_kind` + `causal_refs.length` 派生轻证据 chip（如"已交付"/"已评审"/"有证据"），BE-2 后补真实 artifact_refs。

**4.1.3 inline thread-expand**（G1 关键）
- `FeedCard.tsx:206-211`"还有 N 条回复"死 div → 改成可点 button，触发本地 `expanded` state
- **P0 范围**：展开**已返回的 `reply_previews`**（`/api/activity` 仅给有限条，`api_activity.go:69`）+ "查看完整 thread"按钮。完整展开（>preview 条数）需调用 `getQuestDetail` 拉 `thread_posts`，标为 API 依赖，P0 不承诺"纯前端展开全部"。
- 三段式：preview（默认 1-2 条）→ 展开（已返回 previews 全部）→ "查看完整 thread"按钮 → `onOpen` 进 Detail（或调 getQuestDetail inline 拉全量 thread_posts）
- 整卡点击不再强制 `onOpen`：改为仅点击"查看完整 thread"或卡片标题区才跳 Detail，正文区可展开不跳转

### 4.2 创建委托三档（G2）

**4.2.1 CreateQuestSheet 收尾**（已是三档，补语义）
- **创建口继续发旧 `mode`（run/check/design）**——后端 `POST /api/quests` 只认 `mode`，`createQuestReq` 无 `workflow_mode` 字段（checker Critical）。`workflow_mode` 只用于 `/api/quests/{id}/workflow-mode` 升级口。
- `plan_only` 加进 `QuestMeta` 类型（当前 untyped，走 index signature）
- banner 文案"外部写自动升 Checked"加 disclaimer：手动 escalate 可用，自动判定依赖后端（BE-1）

**4.2.2 Automation 流奥卡姆剃刀**（真债）
`CreateAutomationSheet.tsx` + `Automations.tsx` 当前用 `AutomationFlow = 'execute'|'design_phase'|'design_only'`，与委托的三档不一致。重设计：
- automation flow 映射到三档：`execute`→Direct、`design_phase`→Checked(带方案)、`design_only`→Goal(plan only)
- 四个裸 toggle 降级进 AdvancedSection：
  - `auto_apply`（审查通过自动应用）→ Goal/Checked 高级选项
  - `allow_quick_auto_complete`（Quick 自主闭环）→ Direct 默认行为，删除裸 toggle
  - `auto_spawn_execute`（方案后自动执行）→ Goal 默认行为，反转成 planOnly 逆开关
  - `allow_l2`（外部副作用）→ 高级选项，与 escalation 联动
- `autoStart` 默认对齐：automation 也默认 true（当前 CreateAutomationSheet:52 默认 false，与"发 root post 默认启动"矛盾）

### 4.3 mode escalation UI（G5）

**4.3.1 接通 `upgradeQuestWorkflowMode`**
- QuestDetail header（:977-994）加 escalate action：Direct→Checked、Checked→Goal 手动按钮（单向升）
- 调用 `upgradeQuestWorkflowMode(qid, {workflow_mode, reason})`（quests.ts:118）
- 升级后 backend 发 system reply（spec 要求"升级是 system reply"），前端在 thread ledger / feed 渲染 `system_escalation` post_kind（已支持，FeedCard:25）

**4.3.2 escalation band**（QuestDetail 新组件）
- 在 CurrentWorkStatus 上方加 `EscalationBand`：显示当前 mode + 升级历史 + "为何升级"reason
- Direct 外部写自动升 Checked 的**自动**判定标 BE-1，前端只显示结果（system_escalation post）

### 4.4 Goal / DecisionNote 产品化（G6）

**4.4.1 Goal 运行时三段式**（QuestDetail 内）
基于现有资产组合，不造新底层，**P0 不发明写入语义**（checker Required）：
- **方案讨论 thread**：复用 `ThreadLedgerPanel`（已支持 decision_note/system_escalation kind），Goal 模式下置顶讨论区
- **方案 review gate**：在 `DesignPlanCard`（:1175）下方加 `PlanReviewGate` 组件——**P0 只展示**已有 design doc + thread ledger + `spawnExecuteFromDesignQuest`（:578 已有）入口；**不发明** approve/request_changes/decision 写入语义。需要创建/编辑 DecisionNote 或方案 review verdict，列 BE 前置。
- **slice execution**：用 `DesignDoc.implementation_steps`（types.ts:611）渲染 slice list，每个 slice 关联 `LoopStateCard`（:1162）的状态；slice 类型化标 BE-5

**4.4.2 DecisionNote 体验**
- 当前只是 post_kind 点。重设计：DecisionNote 作为 thread 的 pinned outcome（spec 原话），在 thread 顶部 pin 显示
- `ThreadLedgerPanel.tsx:26` 已有 label，**P0 只显示已有 `decision_note` post**（checker Required）——扩展为可展开的 pinned card（作者/正文/证据 chips/轻状态 统一 Post 视觉）；创建/编辑 DecisionNote 列 BE 前置
- FeedCard 对 decision_note post 显示"结案"轻提示

### 4.5 automation as Feed actor（G4）

- automation 已是一等 author_role（FeedCard:16）。后端 `orchestrator/automation.go:443` + `engine_e2e_test.go:2082` 已验证 **`automation.run` 投影成 ThreadPost**（`AuthorRole=automation, Kind=automation_run`）。
- **P0 收窄**（checker Required）：只展示已有 automation post 的轻状态，**不从 `source`+`post_kind` 硬推** no_finding/candidate/exception chip。`no_finding` 仍在 `fsstore/automation.go:365` Outcome + discovery archive，不是 post。
- automation outcome **完整投影成 post/reply** 标 BE-2：no_finding/candidate/exception 三种碎片实体需后端投影进 `/api/activity`。
- Inbox 的 Automation Candidates lane 保留（干预面），但 Feed 也能看到 automation 的社区动态

### 4.6 source_event_id（G7）

- 前端透传：`ActivityItem`/reply preview 已在透传 source_event_id（checker Optional 确认）。放轻状态行/hover，**不做响提示**（不占主视觉）。
- ThreadLedgerPanel:85 已显示 `event #N`；FeedCard 轻状态行对齐该样式，hover 可见。

---

## 5. 文件变更预估

### 新增
- `src/components/EscalationBand.tsx` — mode 升级 band
- `src/components/PlanReviewGate.tsx` — Goal 方案 review 闸门
- `src/components/DecisionNoteCard.tsx` — DecisionNote pinned card（或扩 ThreadLedgerPanel）
- `src/components/EvidenceChips.tsx` — 证据 chips 行

### 修改
- `src/components/FeedCard.tsx` — 密度降级 + inline expand + 证据 chips + source_event_id 提示
- `src/pages/FeedView.tsx` — expand 状态传递（若需 lift state）
- `src/components/CreateQuestSheet.tsx` — 发 workflow_mode + plan_only 类型化 + banner disclaimer
- `src/components/CreateAutomationSheet.tsx` — 三档词表 + 裸 toggle 降级 + autoStart 默认
- `src/pages/Automations.tsx` — 同上 editor 侧
- `src/pages/QuestDetail.tsx` — escalate action + EscalationBand + PlanReviewGate + DecisionNote pin
- `src/api/types.ts` — QuestMeta 加 plan_only/triage_mode/candidate_source 显式类型；ActivityItem 加 artifact_refs?(若 BE-2)
- `src/__tests__/quest-display-semantics.test.ts` — 清死代码引用 + 新增 expand/证据 chip 断言

### 删除
- `src/components/LoopCockpit.tsx`
- `src/components/AlertBar.tsx`
- `src/pages/feedChromeModel.ts`

---

## 6. 验证

```bash
cd /data00/home/lihuanyu.0w0/agent-workspace/gloop/web
npm run build              # tsc + vite build
npm test                   # mage-review + quest-answer-api + quest-display-semantics
```

关键测试：`quest-display-semantics.test.ts`（post_kind 弱化 + ledger provenance），改动后须保持绿。inline expand 新增 `>3 replies` 展开行为断言（checker Required）。

**提交前证据**（checker Required）：除 build/test 外，给 Feed expand / Automation 三档 / QuestDetail escalation·Goal 的浏览器截图或录屏。

---

## 7. 剩余风险

1. **ActivityItem 不暴露 artifact_refs**：证据 chips P0 用 post_kind/causal_refs 派生，真实 artifact_refs 依赖 BE-2。
2. **Goal slice 类型化**：P0 用 DesignDoc.implementation_steps 表达，语义弱；完整 slice 执行跟踪依赖 BE-5。
3. **Feed 非 stream-pushed**：当前 30s 轮询；inline expand 的完整展开依赖 `getQuestDetail` 拉 thread_posts（API 依赖，非纯前端）。
4. **死代码删除可能影响测试覆盖率门槛**：需确认 CI 无硬覆盖率门。

---

## 8. 实施顺序（slice 化，每 slice 跑 build/test/截图给 checker 审）

**slice 1**（先行，checker 建议先审）：
- 1-A 删死代码（LoopCockpit/AlertBar/feedChromeModel + 测试引用）
- 1-B FeedCard 降密度
- 1-C inline thread-expand（展开已返回 previews + 查看完整 thread）
- 1 验证：build + test + Feed expand 截图 → 给 checker 审

**slice 2**（slice 1 审过后）：
- 2-A Automation 流奥卡姆剃刀（三档词表 + 裸 toggle 降级 + autoStart 默认）
- 2-B mode escalation UI（接通 upgradeQuestWorkflowMode + EscalationBand，手动入口，自动判定标 BE-1）

**slice 3**（最重，最后）：
- 3-A Goal 运行时三段式骨架（PlanReviewGate 只展示，不发明写入语义）
- 3-B DecisionNote pinned card（只显示已有 decision_note post）
- 3-C automation actor 轻状态 + source_event_id hover 透传

每 slice 独立提交，便于 checker 分段 review。

---

## 9. 修订记录

### 2026-07-02 checker verdict 接受（无敌一呆）
- **Critical**：创建口继续发旧 `mode`（run/check/design），不发 `workflow_mode`；`workflow_mode` 只用于升级口。后端 `POST /api/quests` 只认 `mode`，`createQuestReq` 无 `workflow_mode`。→ §4.2.1 修正。
- **Required 1**：inline expand 不能"纯前端展开全部"。`/api/activity` reply_previews 有限（`api_activity.go:69`）。P0 = 展开已返回 previews + 查看完整 thread；完整展开调 getQuestDetail。→ §4.1.3 修正。
- **Required 2**：automation outcome P0 收窄。`automation.run` 已投成 ThreadPost（`orchestrator/automation.go:443`），但 no_finding 仍在 discovery archive。不从 source+post_kind 硬推 outcome chip。→ §4.5 修正。
- **Required 3**：PlanReviewGate 不发明 approve/request_changes/decision 写入语义。P0 只复用 design doc + thread ledger + spawnExecuteFromDesignQuest 展示。DecisionNote pinned card 只显示已有 post。→ §4.4 修正。
- **Required 4**：验证加浏览器截图/录屏 + >3 replies expand 测试断言。→ §6 修正。
- **Optional 1**：source_event_id 已透传，放轻状态/hover 不做响提示。→ §4.6 修正。
- **Optional 2**：EvidenceChips/DecisionNoteCard 只在复用≥2处才抽。→ §5 注明。

每步可独立提交，便于 checker 分段 review。
