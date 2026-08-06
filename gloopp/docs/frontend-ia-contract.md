# Gloop Frontend IA Contract (P0 Draft v3)

## 1. Scope

覆盖前端信息架构、action ownership、fact ownership。不设计视觉、不写组件方案、不展开后端实现。标注 backend prerequisite 和当前违规项。

## 2. Core Invariants

1. **同一事实只由一个 selector/domain helper 派生**，页面不得自行解释 quest 状态。
2. **同一决策只有一个主入口**，其他 surface 只能显示状态、轻 CTA、deep link 到主入口。
3. **Runtime warning 不是 Human Exception**；`runtime.idle_warning` / `runtime.protocol_warning` 归 runtime/system status，不进入人工异常队列。
4. **缺 backend 数据的能力不得在前端伪造**，必须记为 prerequisite。

## 3. Surface Contract

| Surface | 一句话职责 | 用户问题 | 允许的主 action | 禁止承担的 action | 事实来源 | 入口优先级 |
|---------|-----------|---------|----------------|------------------|---------|-----------|
| **Feed** | 社区流叙事：谁做了什么、产出了什么 | "最近发生了什么？" | 创建 quest (composer)、展开 thread、查看 adventurer | candidate 确认/取消、review verdict、blocked retry | Activity / Thread Ledger projection | 日常浏览 |
| **Board/List** | 全局状态视图：所有 quest 的当前位置和健康度 | "全局进展如何？" | 分组/排序/过滤、切换视图、deep link 到 Detail | 任何决策 action（accept/reject/review/retry） | QuestMeta + `domain/questSelectors.ts` | 日常监控 |
| **Inbox** | 待我决策的队列：需要人类拍板才能继续的事项 | "现在需要我做什么？" | automation candidate accept/reject/edit、deep link 到 Detail | waiting_input 直接回答、review verdict、blocked retry（这些跳 Detail） | pending candidate queue + HumanException view（非独立 store） | 高频决策 |
| **Quest Detail** | 单个 quest 的完整上下文、trace、操作面板 | "这个 quest 具体怎么样了？" | start/stop、review approve/reject/apply、waiting_input 回答、blocked retry、discard、spawn fanout | — (这是终极操作面) | QuestMeta + events + artifacts + trace | 深度操作 |
| **Knowledge** | 日常知识操作：浏览维度、预览内容、手动刷新、dim 导入导出 | "agent 当前知道什么？" | context refresh、context dim import/export | 全局配置包导入导出、context 策略配置 | context dims / summary / export preview APIs | 日常操作 |
| **Resources** | 可配置资源浏览：能力注册表和协议模板 | "有哪些可用的 skill/prompt？" | 浏览/搜索/查看详情 | 创建/编辑/删除（如果未来支持） | skills / prompts registry APIs | 参考查阅 |
| **Adventurers** | actor inventory：warrior/mage 人格配置和状态 | "谁在干活？" | create/edit/retire、set default、activate | agent 命令行配置（归 Settings） | adventurer / executor APIs | 中频管理 |
| **Automations** | 自动化规则 CRUD 和运行状态 | "什么会自动触发？" | create/edit/delete、enable/disable、手动触发、模板复制 | quest 级决策 | automation config / run history APIs | 中频管理 |
| **Stats** | 运营指标概览：吞吐、成功率、卡住老化 | "系统健康吗？" | 时间范围切换、drilldown 到 quest list | raw audit 展示、quest 操作 | stats aggregate APIs（后续 time-series contract） | 定期查看 |
| **Settings** | 控制面配置：agent 命令、allowlist、主题、context 策略、全局配置包导入导出 | "系统怎么配？" | agent 命令配置、allowlist 编辑、主题切换、context 策略、全局 config package import/export、workspace 清理 | 日常知识操作（归 Knowledge）、adventurer 人格编辑（归 Adventurers） | config / executor / policy APIs | 低频管理 |

## 4. Action Ownership Matrix

| Action | Primary Surface | Secondary (deep link / status only) |
|--------|----------------|--------------------------------------|
| quest create | **QuestsBoard (FAB/Composer)** | Automations (模板创建) |
| quest open/drilldown | **Quest Detail** | Board/List/Feed/Inbox/Stats (全部 deep link) |
| quest start/stop | **Quest Detail** | — |
| quest discard | **Quest Detail** | — |
| quest spawn fanout | **Quest Detail** | — |
| automation candidate accept/reject/edit | **Inbox** | Feed (显示 candidate 标签 + deep link 到 Inbox) |
| waiting_input answer | **Quest Detail** | Inbox (显示卡片 + deep link 到 Detail) |
| user_review approve/reject/apply | **Quest Detail** | Inbox (显示卡片 + deep link 到 Detail) |
| blocked recovery/retry | **Quest Detail** | Inbox (显示卡片 + deep link 到 Detail) |
| automation create/edit/delete | **Automations** | — |
| automation enable/disable | **Automations** | — |
| automation manual trigger | **Automations** | — |
| automation template copy | **Automations** | — |
| adventurer create/edit/retire | **Adventurers** | — |
| adventurer set default | **Adventurers** | — |
| adventurer activate | **Adventurers** | — |
| context refresh | **Knowledge** | — |
| context dim import/export | **Knowledge** | — |
| config package import/export | **Settings** | — |
| agent command allowlist edit | **Settings** | — |
| theme change | **Settings** | — |
| workspace cleanup | **Settings** | — |
| resource browse/search/view | **Resources** | — |

## 5. Fact Ownership Matrix

| Fact | Owner Module | 状态 |
|------|-------------|------|
| attention level (`blocked` \| `waiting_input` \| `review` \| `apply_failed` \| `runtime_warning` \| `none`) | `domain/questSelectors.ts` → `deriveQuestAttention()` | **to create** (当前散落在 QuestCard.tsx / CurrentWorkStatus.tsx) |
| progress line text | `domain/questSelectors.ts` → `deriveQuestProgressLine()` | **to create** (当前散落在 QuestCard.tsx / CurrentWorkStatus.tsx) |
| stuck diagnosis | `features/quest/stuckDiagnosisMetrics.ts` → `getStuckDiagnosis()` | 已存在 |
| actor role meta (label/icon/avatar/tone) | `domain/actorMeta.ts` | **to create** (当前 FeedCard.tsx 和 ThreadLedgerPanel.tsx 各有一份 ROLE_META，shape 不一致) |
| status label | `components/util.ts` → `statusLabel()` | 已存在；P1a 评估是否迁入 domain 或通过 domain-facing wrapper 暴露 |
| phase label | `domain/questSelectors.ts` → `phaseDisplayName()` | 已存在 |
| runtime health | `domain/questSelectors.ts` → `deriveRuntimeHealth()` | **to create** (当前无统一派生；runtime warning 散落在各处) |
| apply failed | `domain/questSelectors.ts` → `hasApplyFailed()` | 已存在 |
| human exception classification | `features/quest/stuckDiagnosisMetrics.ts` → `humanExceptionSourceKind()` | 已存在 |
| quest action availability | `domain/questSelectors.ts` → `questActionState()` | 已存在 |
| review verdict options | `domain/questSelectors.ts` → `reviewActionDraft()` | 已存在 |
| resource usage | `domain/questSelectors.ts` → `questResourceUsage()` | 已存在 |

## 6. Current Violations (必须修复)

| # | 违规项 | 当前位置（稳定符号 + 行号） | 违反的 Invariant | 目标 | 修复阶段 |
|---|--------|--------------------------|-----------------|------|---------|
| V1 | Feed 直接渲染 automation candidate 确认/取消按钮 | `FeedCard.tsx` — `resolveCandidate()` (L134-145) + `.feed-candidate-actions` 按钮块 (L328-349) | #2 同一决策只有一个主入口 | Feed 只显示 candidate 标签 + deep link 到 Inbox | P1a |
| V2 | actor role meta 两处定义且 shape 不一致 | `FeedCard.tsx` — `ROLE_META` const (L14, 含 avatar field) vs `ThreadLedgerPanel.tsx` — `ROLE_META` const (L11, 含 cls field) | #1 同一事实只由一个 selector 派生 | 统一到 `domain/actorMeta.ts`，允许 presentation variant | P1a |
| V3 | attention/progress 散落在多个组件自行计算 | `QuestCard.tsx` — 内联 attention/progress 派生；`CurrentWorkStatus.tsx` — 内联 attention/progress 派生 | #1 同一事实只由一个 selector 派生 | 统一到 `domain/questSelectors.ts` 的 `deriveQuestAttention()` / `deriveQuestProgressLine()` | P1a |
| V4 | Desktop 主 nav 缺少 Settings 入口 | `App.tsx` — `NAV` 不含 `settings`；`App.tsx` — `MOBILE_NAV` 已含 `settings`；`AppSidebar.tsx` — `sidebar-foot` 仍有 Settings persistent entry。 | — (信息架构问题，非 invariant 违规) | Desktop Settings 进入主 nav persistent entry；footer 不再重复 persistent entry；移动端保持不变 | P2 |

## 7. Navigation Proposal

按工作流而非后端资源名组织：

```
Monitor（监控）
  ├── Feed
  ├── Board
  └── List
Decide（决策）
  └── Inbox
Automate（自动化）
  └── Automations
Knowledge（知识）
  └── Knowledge
Actors（执行者）
  └── Adventurers
Resources（资源）
  └── Resources (内部 Skills / Prompts 两个 tab)
Observe（观察）
  └── Stats
Settings（设置）
  └── Settings ← 从 sidebar-foot 提升到主 nav
```

**Create 的归属**：Create 是全局 primary action，通过 Feed composer / FAB 承载，不是一个 persistent nav surface。未来 Command Palette 也暴露为命令。

**决议：**

- **Resources**: 选方案 A — 合并为一个顶级 nav item "Resources"，内部 Skills / Prompts 子 tab。理由：两者都是只读/半只读资源浏览，不是高频操作面；未来 Command Palette 提供直达路径。
- **Adventurers**: 独立顶级入口。理由：Adventurers 是可调度的 actor inventory，和 Skills/Prompts（agent 可消费的 resource）不是一类东西。不放进 "Agents" 分组，避免二级概念负担。
- **Settings**: 选方案 A — 单一主 nav 入口，不保留 footer 重复 persistent entry。实现时验证 collapsed/mobile/keyboard 路径可达。移动端 MOBILE_NAV 已有 settings，保持不变。

## 8. Non-goals (P0 不做)

- Command Palette / 全局搜索 (→ P3)
- Automation 批量启停 (→ P1.5，可并行小项)
- **Quest 批量操作** (explicitly out-of-scope；需 backend 幂等 + 审计 contract)
- Stats 图表/趋势线 (→ P4c，需 backend time-series query model)
- Audit UI (→ P4b，需 event ledger)
- Notifications UI (→ P4b，需 event ledger)
- 协作评论/指派 (→ 不在当前路线图)
- Onboarding 引导 (→ 术语降频优先，P3 后再考虑)

## 9. Acceptance Criteria

1. 任意一个 surface 允许的主 action 都能在 Action Ownership Matrix 找到唯一 primary surface。
2. 任意一个 quest 状态展示字段都能在 Fact Ownership Matrix 找到 owner。
3. Feed/Inbox/Board/Detail 对 blocked/waiting_input/user_review/apply_failed 的职责不冲突。
4. Knowledge (日常操作) vs Settings (策略配置) 的边界清楚；context dim import/export 和 config package import/export 归属不同 surface。
5. 当前违规项 (V1-V4) 有明确的目标状态和修复阶段。
6. 读完文档后，P1a 的 selector 抽取和 V1/V2/V3 修复能直接开工。
7. 读完文档后，P2 的导航重排能直接开工。

---

**Backend Prerequisites**:
- Event Ledger (P4a): 需后端定义统一事件模型 (actor/entity/action/correlation_id/risk/before-after/artifact refs)
- Stats time-series (P4c): 需后端支持按时间范围的聚合查询
- Quest 批量操作 (out-of-scope): 需后端幂等性保证 + 审计记录

---

## 10. Workspace / Harness Boundary (2026-07 决议)

**核心结论：Gloop 不承诺 harness sandbox。**

- `workspace_mode` 是 **artifact boundary metadata**（产物来源/隔离级别），不是用户可配置的 harness 安全策略。
- 前端不再暴露 workspace_mode 选择器（Inbox edit / CreateQuest / CreateAutomation / Automations edit / Settings default 均已移除）。
- `workspace_retention_days` 保留为 artifact lifecycle/cleanup 配置，不属于 harness 范畴。
- 真正的 filesystem sandbox / tool 越界 enforcement 是 harness runtime 的职责，不在 Gloop 前端或 Gloop authority orchestration 层。
- Connector 可用性由 backend config 决定（`settings.connectors.*.enabled`），前端不再用 `workspace_mode !== 'readonly'` 做二次判断。

## 11. Phase Authority Audit (2026-07)

**原则：action authority 应收敛到 runtime-derived selectors，而非纯 `quest.status`。**

已完成的前端收紧：
- `questActionState()` 现在读取 `phases[]` + `current_phase_idx` + `pipeline_version`，与 `questPhaseView()` 共用 `resolveCurrentPhaseIdx()` fallback。
- `canStop`: pipeline 可用时要求 current phase 实际为 running；`current_phase_idx` 缺失时 fallback 到第一个 running phase。
- `isRunning`: `isEnded` 短路，终态 quest 不会被 stale running phase 误判。
- `QuestDetail` workflow upgrade 按钮：runtime-active gate 阻止执行中升档。

### Backend Follow-up

| # | 项 | 说明 | 状态 |
|---|----|------|------|
| PA-BE-1 | `UpgradeWorkflowMode` 拒绝 active runtime | 后端拒绝 running/reviewing/queued/starting 状态升档，防止 API 直调绕过前端 gate。runtime 内部 Direct→Checked 升档走 `upgradeWorkflowModeDuringRuntime()` 不受限 | ✅ 已完成 |
| PA-BE-2 | Goal spawn explicit authority | 当前 `canSpawn` 依赖 `design + success + !child` 隐含 contract；后端应提供 `allowed_actions` / `can_spawn_execute` 或显式 phase completion authority | 待办 |

**前端不伪造 backend authority**：如果后端未提供 `allowed_actions`，前端保持当前基于 `quest.status` + phase fields 的 best-effort 派生，不发明不存在的权限字段。
