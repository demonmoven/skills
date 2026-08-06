# 前端改造 Spec v0.1 — Design Phase + PhaseDef 数据驱动

> 配套文档：`design-phase-spec-v0.1.md`
> 目标：让前端适配 4 阶段管道（设计 → 设计评审 → 执行 → 实现评审），同时把两阶段硬编码假设全部拆掉，真正做到 PhaseDef 数据驱动。
> 状态：已实现（v0.1.2 前端继续补齐 HITL 流程控件）

## 实现状态（2026-06-22）

本 spec 的前端主链路已经落地：

- **P0 selector 边界**：`web/src/domain/questSelectors.ts` 已承载 `questPhaseNodes`、`questPhaseView`、`questTimelineEntries`、`groupEventsByPhase`、`getDesignPlan`、`questActionState` 等业务计算，`QuestDetail.tsx` 不再手写三段式时间线。
- **P0 多阶段详情页**：`QuestDetail` 使用 `PhaseProgressBar` + 数据驱动 timeline，支持默认两阶段、设计四阶段和旧数据 fallback。
- **P0 设计方案展示**：`DesignPlanCard` / design doc panel 已接入，支持 `design_summary` / design doc 结构化展示和 execute prompt 折叠查看。
- **P1 创建入口**：`CreateQuestSheet` / `CreateAutomationSheet` 已从内部 `execute/design` 字段升级为"直接执行 / 先设计再执行 / 只出方案"流程选择，并接入 `with_design_phase`、`allow_quick_auto_complete`、`auto_spawn_execute`。
- **P1 首页/看板承载**：`QuestsBoard` 已展示当前 phase、返工次数、waiting input 摘要、policy chips、effect type、outputs/design summary，并支持 `groupBy=phase` 的阶段看板视图。
- **P1 ContextPack 标签**：`domain/contextLabels.ts` 已集中管理上下文块显示名，`ContextPackView` 不再散落硬编码。

仍未完全实现或刻意暂缓：

- **P2 高级交互**：阶段分组的更强视觉层次、产物摘要弹窗、批量操作等仍未做。
- **浏览器级视觉验收**：本轮只跑 `npm --prefix web run build`，未新增 Playwright/截图验收。
- **QuestDetail 继续瘦身**：已经抽出大量 selector/trace adapter，但页面仍偏大，后续还可继续拆组件。

---

## 一、背景与目标

### 1.1 背景

后端即将做两件事：
1. **PhaseDef 数据驱动改造**（D.-1）：`buildPhaseConfig` 从硬编码 `switch phaseIdx` 改为读 `PhaseDef`
2. **Design Phase 上线**（D.0/D.1）：支持 4 阶段管道 `warrior_design → mage_design_review → warrior_execute → mage_implementation_review`

前端目前的状态是：**部分数据驱动，部分还留着两阶段硬编码**。看板已经能动态列化了，但详情页时间线、文案、状态判断还有不少"剑士/法师"两段式假设。Design Phase 一上，这些地方会直接出 bug 或显示错乱。

### 1.2 目标

**P0（必须做）**：
- 前端能正确渲染 4 阶段 quest，不报错、不显示"Phase 3"这种无意义名字
- 设计方案（design_plan artifact）作为一等交付物有独立展示位
- 时间线从硬编码三段式改为数据驱动，支持任意阶段数
- 阶段进度指示器（step bar）— 提前做，用于验证 selector 输出正确性

**P1（应该做）**：
- 阶段进度指示器（顶部 step bar）
- 创建委托时的设计阶段推荐提示
- ContextPack block 名友好化

**P2（可以做）**：
- 设计方案结构化渲染（按章节折叠）
- 阶段分组（设计组 / 执行组）视觉区分
- 设计评审 vs 实现评审的图标/颜色区分

### 1.3 非目标

- 不重做前端架构（不引入状态管理库、不重构路由）
- 不做移动端适配优化
- **不优先新增后端 API**：优先用现有 API 和事件流；若 design_plan 等数据无法稳定读取，允许补一个 artifact-by-kind API。API 是手段不是目标，但前端不为后端语义缺口背锅。

---

## 二、设计原则

### 2.0 页面只做组合，逻辑下沉到 selector / component

这是本 spec 的最高优先级原则。

- **页面组件（QuestDetail.tsx / QuestsBoard.tsx）只做组合渲染**：把 selector 的数据喂给 component，不写业务逻辑。
- **复杂计算逻辑必须放在 `domain/` 下的 selector 里**，可测试、可复用、可独立演进。
- **通用 UI 逻辑放在 `components/` 下**，样式 + 交互，不涉及 quest 业务语义。
- QuestDetail.tsx 已经 1600+ 行了，这次 refactor 的目标之一是**把它往瘦了做**，不是继续往里面塞。

反模式（这次要消灭的）：
- ❌ 页面组件里手写 "if idx === 0 就是剑士" 这种两阶段假设
- ❌ 页面组件里拼时间线节点数据
- ❌ 页面组件里直接解析 artifact kind

正确模式：
- ✅ `questPhaseNodes(quest)` → 返回标准 phase 节点列表
- ✅ `getDesignPlan(quest)` → 返回设计方案数据或 null
- ✅ `phaseDisplayName(quest, idx)` → 返回友好阶段名
- ✅ 页面只管 `map` 渲染，不管数据怎么来的

### 2.1 渐进式，不推倒重来

和后端的 D.-1 → D.0 → D.1 节奏对齐，前端也分阶段拆。每一步都能独立上线，不阻塞后端。

### 2.2 数据驱动，不靠状态推断

状态（running/reviewing）只是大类，**具体是什么阶段看 `current_phase_idx + pipeline_def`**。前端不能再出现"if status === 'reviewing' 就是法师评审"这种两阶段假设。

### 2.3 向后兼容，旧数据也能看

- 没有 `pipeline_def` 的老 quest 按两阶段渲染（fallback 逻辑保留）
- 两阶段 quest 和四阶段 quest 在同一个列表面板里共存
- 不做数据迁移，纯渲染层兼容

### 2.4 设计方案是一等公民，但不特殊到改架构

- design_plan artifact 有独立展示位、独立样式
- 但它本质还是 phase artifact，走同一套读取/渲染逻辑
- 不新增独立的数据模型层，只是 kind=design_plan 的渲染特殊化

---

## 三、现状分析

### 3.1 已经数据驱动的部分 ✅

| 模块 | 文件 | 状态 |
|------|------|------|
| QuestsBoard 看板列化 | `pages/QuestsBoard.tsx` | ✅ 从 pipeline_def 动态生成列 |
| phase view selector | `domain/questSelectors.ts:questPhaseView` | ✅ 从 phases[] 读 |
| 创建委托开关 | `components/CreateQuestSheet.tsx` | ✅ 已有 withDesignPhase |
| automation 配置 | `components/CreateAutomationSheet.tsx` | ✅ 已有 with_design_phase |
| API 类型定义 | `api/types.ts:QuestPhaseTask` | ✅ 字段基本齐全 |

### 3.2 还在硬编码两阶段的部分 ❌

| 模块 | 文件 | 问题 |
|------|------|------|
| QuestDetail 时间线 | `pages/QuestDetail.tsx:phaseEntries` | ❌ 硬编码剑士→法师→用户三段 |
| 状态文案 | `pages/QuestDetail.tsx:mageReviewCopy` | ❌ "法师评审"默认指实现评审 |
| 状态徽标 | `components/StatusBadge.tsx` | ⚠️ 只显示状态大类，不显示阶段名 |
| phaseName fallback | `domain/questSelectors.ts:phaseNameFallback` | ❌ idx 0=剑士, idx 1=法师，超过就显示 Phase N |
| ContextPack block 名 | `components/ContextPackView.tsx:blockKindLabel` | ⚠️ 缺 design_plan 等新 block |
| 动作判断 | `domain/questSelectors.ts:questActionState` | ⚠️ canReview 只看 status==='user_review'，理论上没问题但需要确认 |

### 3.3 新增能力需要补的 ➕

- 设计方案（design_plan artifact）独立展示卡片
- 阶段进度指示器（step bar）
- 创建委托时的推荐提示
- 设计阶段 / 执行阶段的视觉分组

---

## 四、详细设计

### 4.1 时间线数据驱动化（P0，分两步）

**现状问题**：`phaseEntries` 是手写三段式（剑士执行 → 法师评审 → 用户审核），4 阶段 quest 放进去会丢阶段。

**改造方案**：分两步走，先立 selector 边界，再做事件归属。

#### 4.1.1 P0a：phase 主节点数据驱动

先把时间线的骨架从硬编码改成从 `quest.phases[]` 动态生成。核心是 `questPhaseNodes` selector 先写对，页面只负责渲染。

```typescript
// domain/questSelectors.ts — 纯数据计算，可单测
export type PhaseNode = {
  phaseIdx: number
  name: string
  displayName: string
  role: 'execute' | 'review' | string
  class_?: string       // warrior / mage / user
  status: PhaseStatus
  turns: number
  reworkCount: number
  startedAtMs?: number
  endedAtMs?: number
  sessionId?: string
  artifactKinds: string[]   // 该阶段产出的 artifact kind 列表
}

export function questPhaseNodes(quest: QuestMeta | null | undefined): PhaseNode[] {
  const phases = Array.isArray(quest?.phases) ? quest.phases : []
  const pipelineDef = quest?.pipeline_def || []
  return phases.map((phase, idx) => {
    const def = pipelineDef[idx] || phase
    return {
      phaseIdx: idx,
      name: def.name || phase.name || `phase_${idx}`,
      displayName: def.display_name || phase.display_name || phaseNameFallback(idx, quest?.status),
      role: def.role || phase.role || 'execute',
      class_: def.class || (def.role === 'review' ? 'mage' : 'warrior'),
      status: phase.status || 'pending',
      turns: Number(phase.turns) || 0,
      reworkCount: Number(phase.rework_count ?? quest?.rework_count ?? 0) || 0,
      startedAtMs: phase.started_at_ms,
      endedAtMs: phase.ended_at_ms,
      sessionId: phase.session_id,
      artifactKinds: [],  // 后续从 outputs/events 聚合
    }
  })
}
```

**时间线节点渲染规则**（组件层，纯展示）：
- `role === 'execute'` + `class_ === 'warrior'` → 剑士色（蓝）+ 剑图标
- `role === 'review'` + `class_ === 'mage'` → 法师色（紫）+ 法杖图标
- `role === 'review'` + `class_ === 'user'` → 用户色 + 人图标
- `status === 'running'` → 节点高亮 + pulse 动画
- `status === 'done'` → 节点打勾
- `status === 'pending'` → 节点置灰
- `status === 'failed'` → 节点红色 + 警告图标

**页面层改动最小化**：
- `QuestDetail.tsx` 里把原来手写的 `phaseEntries` 换成 `questPhaseNodes(quest).map(...)`
- 不新增业务逻辑，只做 `map` 渲染
- 样式类名从 `phase-entry-warrior` 改为 `phase-entry-execute` 或直接读 `role`

**文件改动（P0a）**：
- `domain/questSelectors.ts`：新增 `questPhaseNodes`、`PhaseNode` 类型
- `pages/QuestDetail.tsx`：`phaseEntries` 替换为 selector 调用
- 组件层：`TimelinePhaseNode` 接收 `PhaseNode` 渲染

#### 4.1.2 P0b：事件归属到 phase

agent_update / review 等事件按 `session_id` 归属到对应 phase 节点下。

**归属规则**：
- 事件有 `session_id` → 匹配 `phase.session_id` 归入对应阶段
- 事件有 `phase_idx` → 直接按阶段号归属
- 都没有 → 归入当前运行阶段（fallback，不报错）
- notes / comments / failure_attribution 等全局事件挂在时间线最外层，不入 phase

**向后兼容**：
- 如果 `quest.phases` 为空或只有两阶段 → 走现有 fallback 逻辑
- 老 quest 显示效果不变

**文件改动（P0b）**：
- `domain/questSelectors.ts`：新增 `groupEventsByPhase(events, phases)` 辅助
- `pages/QuestDetail.tsx`：事件渲染改为按 phase 分组

### 4.2 设计方案展示卡片（P0）

design_plan 是一等交付物，需要在详情页有独立展示位，不是混在 outputs 里。

**展示位置**：
- QuestDetail 顶部（状态徽章下方，时间线上方）
- 折叠卡片，默认展开，显示方案摘要
- 点击展开查看完整方案

**数据契约（严格顺序）**：

1. **优先：按 kind 查找**（一等公民路径）
   - 从 `quest.outputs` / phase artifact 中找 `kind === 'design_plan'`
   - 这是后端 D.1 之后的标准路径

2. **次优先：按 phase role + name 定位**（过渡期 fallback）
   - 从 `quest.phases` 中找 `role === 'execute' && name === 'warrior_design'` 的 phase
   - 再从该 phase 的产出中找 design_plan
   - **注意：不是 name contains "design" 的模糊匹配**，必须精确匹配已知 phase 名

3. **兜底：不显示**
   - 两种方式都找不到 → 不渲染 DesignPlanCard
   - 不做启发式猜测，避免把非设计方案当成设计方案展示

```typescript
// domain/questSelectors.ts
export function getDesignPlan(quest: QuestMeta | null | undefined): QuestArtifact | null {
  if (!quest) return null

  // 1. 按 kind 精确查找（标准路径）
  const outputs = Array.isArray(quest.outputs) ? quest.outputs : []
  const byKind = outputs.find(a => a.kind === 'design_plan')
  if (byKind) return byKind

  // 2. 按 phase name 精确匹配（过渡期 fallback）
  const phases = Array.isArray(quest.phases) ? quest.phases : []
  const designPhase = phases.find(p => p.name === 'warrior_design')
  if (designPhase?.session_id) {
    // TODO: 从 events / session outputs 中找
    // 过渡期先用 outputs 里模糊匹配 session_id
  }

  return null
}
```

> **设计原则**：kind 是一等契约，name 是兼容兜底。绝不做 "name 包含 design" 这种模糊匹配——否则任何叫 design_review 的文件都会被当成设计方案，体验反而更差。

**展示形式**：
```
┌─────────────────────────────────┐
│ 📋 设计方案              [展开] │
│                                 │
│  问题理解：xxxx                 │
│  方案选型：方案 A（理由）       │
│  风险边界：xxxx                 │
│                                 │
│  设计阶段 · {turns} turns       │
└─────────────────────────────────┘
```

**结构化渲染（P2 优化）**：
- 解析 Markdown 标题层级
- 按章节折叠展开
- 章节图标（问题/方案/风险/计划）

**文件改动**：
- `components/DesignPlanCard.tsx`（新增）
- `pages/QuestDetail.tsx`：引入 DesignPlanCard，有设计方案时显示
- `domain/questSelectors.ts`：新增 `getDesignPlan(quest)` 选择器

### 4.3 阶段进度指示器（P0，F.-1 后半段做）

在 QuestDetail 顶部加一个 step bar，直观显示当前在哪个阶段。

**为什么放 F.-1**：
- 组件很轻，但能直接验证 `questPhaseView` 和 `questPhaseNodes` selector 的输出对不对
- 相当于 selector 写完后的"可视化自测"，比纯单测更直观
- 早做早发现问题，不然后面时间线改完再发现 selector 错了，改起来更痛

**样式**：横向步骤条，节点数 = phase 数
- 已完成：绿色 + ✓
- 进行中：高亮 + pulse
- 未开始：灰色

**数据来源**：`questPhaseNodes(quest)` — 和时间线复用同一份数据，保证一致性

**向后兼容**：两阶段 quest 显示 2 步条（执行 → 评审），不强行显示 4 步。

**组件契约**：
```typescript
// components/PhaseProgressBar.tsx
type Props = {
  phases: PhaseNode[]     // 从 questPhaseNodes 来
  currentIdx: number      // 从 questPhaseView 来
}
// 组件内部只做渲染，不做任何业务计算
```

**文件改动**：
- `components/PhaseProgressBar.tsx`（新增，纯展示组件）
- `pages/QuestDetail.tsx`：顶部引入
- `domain/questSelectors.ts`：复用 `questPhaseNodes`

### 4.4 状态文案和徽章优化（P1）

**StatusBadge**：
- 现状：只显示状态大类（running / reviewing）
- 改造：hover 时 tooltip 显示当前阶段名（如"设计中"、"设计评审中"）
- 主徽章文字不变（保持简洁），用不同深浅区分同状态的不同阶段

**mageReviewCopy 文案**：
- 现状：默认"法师评审"
- 改造：根据 `current_phase_idx` 和 `pipeline_def` 判断是设计评审还是实现评审
- 设计评审通过 → "设计评审通过，即将进入执行阶段"
- 实现评审通过 → "等待你的确认"（现有文案）

**文件改动**：
- `components/StatusBadge.tsx`：加 tooltip 显示阶段名
- `pages/QuestDetail.tsx`：`mageReviewCopy` 改为数据驱动

### 4.5 创建委托推荐提示（P1，克制版）

配合后端 spec 的"保守触发"原则，不默认自动开，但给提示。

**触发条件**：
- quest_type 选择了 system_design / refactor 等复杂类型
- intensity 选择了 deep / adversarial
- 且 with_design_phase 未开启

**提示形式**：
- 表单内 info 条："当前任务类型/强度较高，建议开启设计阶段以提前评审方案"
- 附一个"一键开启"按钮
- 不阻塞提交

**克制原则（重要）**：
- ✅ 用户点了"知道了"关闭 → **本次会话内不再弹出**
- ✅ 用户手动切过 with_design_phase 开关 → 说明已经注意到了，不再提示
- ✅ 同一条 quest 创建流程里只提示一次
- ❌ 不做刷新就弹、切换字段就弹的骚扰式提示
- ❌ 不用 toast / modal，只用内联 info 条，不抢焦点

**记忆方式**：
- 用组件内 state 记录 `dismissed`，够用就行
- 不持久化到 localStorage（没那么重要，别过度设计）
- 关闭 CreateQuestSheet 再打开，状态重置——也合理，因为是新的创建场景

**文件改动**：
- `components/CreateQuestSheet.tsx`：加推荐提示条 + dismissed 状态
- `components/CreateAutomationSheet.tsx`：类似处理

### 4.6 ContextPack block 名集中配置（P1）

给新的 context block 加上友好名称，不然用户看到 `design_plan` 会懵。

**集中配置原则**：
- block 名映射**不能散**在各个组件里
- 统一抽到 `domain/contextLabels.ts`（或 `components/util.ts`，视规模定）
- 所有需要显示 block 名的地方都从这里读，保证一致

```typescript
// domain/contextLabels.ts — 唯一真相源
export const BLOCK_KIND_LABELS: Record<string, string> = {
  // 现有
  'files': '文件上下文',
  'project_summary': '项目摘要',
  'codebase_index': '代码索引',
  'skill_prompt': '技能提示',
  'user_instructions': '用户指令',
  // 设计阶段新增
  'design_plan': '设计方案',
  'design_review_requirements': '设计评审要求',
  'design_evidence_pack': '设计评审证据包',
  // 评审阶段
  'implementation_review_summary': '实现评审摘要',
  'review_evidence_pack': '评审证据包',
}

export function blockKindLabel(kind: string): string {
  return BLOCK_KIND_LABELS[kind] || kind
}
```

**新增映射**：
- `design_plan` → "设计方案"
- `design_review_requirements` → "设计评审要求"
- `design_evidence_pack` → "设计评审证据包"
- `implementation_review` 相关的和现有 review 区分

**文件改动**：
- `domain/contextLabels.ts`（新增，或放 `components/util.ts`）
- `components/ContextPackView.tsx`：`blockKindLabel` 从集中配置导入
- `KEY_BLOCKS` 数组也从集中配置读
- 检查其他组件里有没有硬编码 block 名的，统一迁过来

### 4.7 phaseNameFallback 改造（P0）

现状：`idx 0=剑士执行, idx 1=法师评审, idx>=2 显示 Phase N`

改造：从 `pipeline_def` 读 name/display_name，没有才 fallback。

```typescript
function phaseNameFallback(quest: QuestMeta, idx: number): string {
  const def = quest.pipeline_def?.[idx]
  if (def?.display_name) return def.display_name
  if (def?.name) return def.name
  if (def?.role === 'review') return `评审 ${idx + 1}`
  if (def?.role === 'execute') return `执行 ${idx + 1}`
  return `阶段 ${idx + 1}`
}
```

**文件改动**：
- `domain/questSelectors.ts`：`phaseNameFallback` 改为接收 quest 而不是裸 idx

### 4.8 看板列名优化（P1）

现状：列名直接用 `def.display_name || def.name`，如果后端没填 display_name，可能显示 `warrior_design` 这种程序名。

优化：前端加一层 display name 映射兜底。

```typescript
const PHASE_NAME_FALLBACK: Record<string, string> = {
  'warrior_design': '方案设计',
  'mage_design_review': '设计评审',
  'warrior': '剑士执行',
  'warrior_execute': '执行',
  'mage_review': '实现评审',
  'mage_implementation_review': '实现评审',
  'user_review': '用户审核',
}
```

**文件改动**：
- `pages/QuestsBoard.tsx`：列名渲染加 fallback 映射
- 或抽到 `components/util.ts` 共享

---

## 五、实施路线

> **先立 selector 边界，再改页面。** 核心不是多写几个组件，而是先把数据计算和页面渲染的边界划清。否则前端会继续变成一坨，后面每加一个 phase 都要痛一次。

### Milestone F.-1：Selector 基础设施（P0，1.5 天）

目标：把核心数据计算逻辑从页面抽到 `domain/`，用轻组件验证输出。

**Selector 层（主菜）**：
- `questPhaseNodes(quest)` — phase 主节点数据计算，从 `phases[] + pipeline_def` 生成
- `phaseNameFallback` 改造：接收 quest 而不是裸 idx，从 pipeline_def 读
- `questActionState` 检查：确认对多阶段的兼容性（理论上没问题，补单测）
- `getDesignPlan(quest)` 骨架：按 kind 优先、phase name 兜底的查找逻辑
- 单元测试：2 阶段 / 4 阶段 / 无 pipeline_def 三种情况

**组件验证（甜点，顺便做）**：
- `PhaseProgressBar` 组件初版 — 直接用 `questPhaseNodes` 的输出渲染
- 目的：**可视化验证 selector 对不对**，比纯单测直观
- 纯展示组件，不包含任何业务计算

**产出物**：
- `domain/questSelectors.ts` 新增 2-3 个 selector + 对应类型
- `components/PhaseProgressBar.tsx` 轻组件
- QuestDetail 顶部先把进度条挂上，验证数据链路通了

### Milestone F.0a：时间线主节点数据驱动（P0，1.5 天）

目标：把 `phaseEntries` 从硬编码改成从 `questPhaseNodes` 读，QuestDetail 页面瘦下来。

- `QuestDetail.tsx` 中 `phaseEntries` 替换为 `questPhaseNodes(quest).map(...)`
- 时间线节点样式按 `role + class_` 区分（execute=剑士蓝, review=法师紫）
- 状态渲染（running/done/pending/failed）和现有一致
- 子节点（事件）暂时先挂在当前 phase 下，不做强归属
- 向后兼容：老 quest 显示效果不变
- 视觉回归测试：两阶段 quest 和之前看起来一样

**关键交付**：`QuestDetail.tsx` 行数不增反减（把计算逻辑迁走了）。

### Milestone F.0b：事件归属到 phase（P0，1 天）

目标：agent_update / review 事件按 session_id 正确归属到各阶段下。

- `groupEventsByPhase(events, phases)` selector：按 session_id 分组
- 时间线渲染从"平铺事件"改为"phase 节点 + 子事件"
- 归属规则：session_id 精确匹配 → phase_idx 匹配 → 当前运行阶段兜底
- 全局事件（notes/comments/failure_attribution）不入 phase，挂最外层
- 向后兼容：两阶段 quest 的事件归属和之前一致

**为什么拆成 F.0a + F.0b**：
- F.0a 是"骨架立住"，价值最大、风险可控，可以先上线
- F.0b 是"细节对齐"，可以跟在 F.0a 后面，也可以并行
- 不拆的话一次改太多，容易出回归 bug

### Milestone F.1：设计方案展示（P0，2 天）

- `DesignPlanCard` 组件
- `getDesignPlan` selector 完善（kind 优先 + phase name 兜底）
- QuestDetail 页面集成（有设计方案才显示，不占位置）
- 折叠交互 + Markdown 渲染
- 空状态 / 加载状态处理
- 如果 artifact 内容不在 meta 里，补一个 artifact-by-kind API 调用（和后端对齐）

### Milestone F.2：体验优化（P1，2 天）

- 状态徽章 tooltip 显示阶段名
- mageReviewCopy 文案数据驱动化
- 创建委托推荐提示（克制版，关了就不再弹）
- ContextPack block 名集中配置
- 看板列名 fallback 优化
- `domain/contextLabels.ts` 集中配置 block 名

### Milestone F.3：高级特性（P2，可选，2 天）

- 设计方案结构化渲染（按章节折叠）
- 阶段分组视觉（设计组 vs 执行组 有分隔）
- 设计评审 vs 实现评审的图标/颜色进一步区分

### Milestone F.3：高级特性（P2，可选，2 天）

- 设计方案结构化渲染（按章节折叠）
- 阶段分组视觉（设计组 vs 执行组 有分隔）
- 设计评审 vs 实现评审的图标/颜色进一步区分

---

## 六、依赖与对齐

### 后端依赖

| 前端里程碑 | 依赖后端里程碑 | 说明 |
|-----------|---------------|------|
| F.-1 | D.-1（弱依赖） | PhaseDef 数据驱动完成后，pipeline_def 字段才完整可信；但 selector 可以用 mock 先写 |
| F.0a | D.0（弱依赖） | 主节点数据驱动不依赖 4 阶段数据，两阶段也能验证；联调需要 D.0 |
| F.0b | D.0 | 事件归属需要真实的多阶段 session_id 才能测准 |
| F.1 | D.1 | design_plan artifact 的 kind 字段需要后端写入 |
| F.2 | D.1/D.2 | 文案和提示依赖设计阶段的完整语义 |
| F.3 | D.2+ | 锦上添花，不阻塞 |

**可以提前做的**：
- F.-1 的 selector 全部可以用 mock 数据写 + 单测，不等后端
- F.0a 的时间线骨架可以先写，两阶段 quest 就能验证
- 不用等后端全部做完再开工，错峰并行

### API 依赖

优先用现有 API 和事件流。若有缺口再补，不硬扛：

- `quest.pipeline_def[]` — 阶段定义 ✅
- `quest.phases[]` — 阶段运行时状态 ✅
- `quest.current_phase_idx` — 当前阶段索引 ✅
- artifact API — 读取 phase artifact（设计方案）✅

**当前 API 对齐结论（2026-06-22）**：
- `QuestPhaseTask.role` / `pipeline_def` / `phases` 已在 `web/src/api/types.ts` 类型化，前端 selector 直接消费。
- `design_plan` 优先通过 `quest.outputs[].kind` 查找，fallback 到 `warrior_design` phase 关联 artifact；详情页也可读取 design doc 结构化内容。
- 暂未新增 `GET /quests/:id/artifacts?kind=design_plan` 过滤 API；现有 meta / design doc / artifact 读取路径已覆盖当前展示需求。

---

## 七、Selector-Component 边界

### 7.1 分层原则

```
┌─────────────────────────────┐
│  pages/QuestDetail.tsx      │  ← 只做组合：import selector + component，render
│  （页面层，尽量薄）         │
└─────────┬───────────────────┘
          │
┌─────────▼───────────────────┐
│  domain/questSelectors.ts   │  ← 纯函数：QuestMeta → ViewModel
│  （数据计算层，可单测）     │     questPhaseNodes, getDesignPlan, questPhaseView...
└─────────┬───────────────────┘
          │
┌─────────▼───────────────────┐
│  components/                │  ← 纯展示：props → UI
│  （组件层，无业务语义）     │     PhaseProgressBar, DesignPlanCard, StatusBadge...
└─────────────────────────────┘
```

### 7.2 什么该放哪

| 内容 | 放哪 | 理由 |
|------|------|------|
| phase 节点数据计算 | `domain/questSelectors.ts` | 纯逻辑，可单测，可复用 |
| 设计方案查找逻辑 | `domain/questSelectors.ts` | 数据契约相关，集中管理 |
| 阶段名 fallback 映射 | `domain/questSelectors.ts` 或 `domain/contextLabels.ts` | 配置集中，不散落 |
| 时间线节点样式 | `components/Timeline.tsx` 或内联 | 纯展示，和业务无关 |
| 卡片折叠交互 | `components/DesignPlanCard.tsx` | 组件内部行为，不影响数据 |
| "要不要显示推荐提示"的判断 | `domain/questSelectors.ts` 或 `components/util.ts` | 规则可复用，可测试 |
| 页面布局组合 | `pages/QuestDetail.tsx` | 页面就是干这个的 |

### 7.3 验收标准

- `QuestDetail.tsx` 里找不到 `if (phaseIdx === 0)` 这种硬编码判断
- 新增 phase 相关逻辑时，优先改 selector，不是改页面
- selector 都有对应的单元测试
- 组件 props 都是 plain data，不接收 quest 整对象（按需传字段）

---

## 八、风险与应对

### 8.1 时间线重构成 bug 重灾区

**风险**：时间线是详情页最复杂的部分，重构容易引入回归（事件顺序错乱、子节点归属错误等）。

**应对**：
- 保留旧的 phaseEntries 逻辑作为 fallback 开关
- 用 feature flag 控制：新 pipeline_def 存在且有效 → 走新逻辑；否则走旧逻辑
- 准备两阶段 quest 的视觉对比测试

### 8.2 设计方案数据读取不及时

**风险**：设计阶段跑完后，artifact 可能没即时同步到 quest meta 里，前端看不到。

**应对**：
- 优先从 quest meta 的 phases / outputs 读
- meta 里没有但应该有的时候，调 artifact API 主动拉
- 事件流里有 phase_checkpoint / review_quest 事件时，触发一次刷新

### 8.3 多阶段 quest 导致界面拥挤

**风险**：4 阶段 + 每个阶段下的子事件，时间线变得很长，重要信息被淹没。

**应对**：
- 已完成的 phase 默认折叠（只显示标题 + 结果摘要）
- 当前阶段默认展开
- 用户可以手动展开/折叠各阶段
- 设计方案卡片是独立的，不在时间线里挤

### 8.4 两阶段和四阶段视觉不统一

**风险**：两阶段老 quest 和四阶段新 quest 放在一起，看起来像两个产品。

**应对**：
- 时间线的节点样式统一（同样的节点大小、同样的图标风格）
- 阶段数不同，但视觉语言一致
- 看板列数不同没关系，列宽自适应

---

## 九、验证指标

### 9.1 功能正确性

- [x] 两阶段 quest 显示效果和重构前一致（通过 legacy fallback 保持可读）
- [x] 四阶段 quest 正确渲染 4 个时间节点（`questPhaseNodes` + `PhaseProgressBar`）
- [x] 设计方案卡片在有 design_plan / design doc 时显示，没有时不显示
- [x] 阶段进度条和当前阶段一致（复用 `questPhaseView.currentIdx`）
- [x] 设计评审通过 → 推进到执行阶段，不显示"等待用户确认"
- [x] 实现评审通过 → 显示"等待用户确认"

### 9.2 体验指标

- [x] 用户能一眼看出当前在哪个阶段（详情页 step bar + 首页 phase line）
- [x] 设计方案可读性：不用展开时间线就能看到方案摘要
- [ ] 创建委托时，复杂任务的用户有 ≥ 30% 会看到并考虑开启设计阶段（需要真实使用数据，未验证）

### 9.3 代码质量

- [x] 前端不再在主渲染路径硬编码"idx 0=剑士, idx 1=法师"；旧数据仅保留 legacy fallback
- [x] `pipeline_def` / `phases` 是阶段信息的主要真相源
- [ ] 新增组件有单元测试覆盖（当前主要依赖 `npm --prefix web run build` 和 Go/API/E2E 测试，前端单测未补）
- [x] 向后兼容：没有 pipeline_def 的老数据不报错

---

## 十、相关文件清单

### 新增文件
- `web/src/components/DesignPlanCard.tsx`
- `web/src/components/PhaseProgressBar.tsx`
- `web/src/domain/contextLabels.ts`（集中配置 block 名 / phase 名映射）

### 修改文件
- `web/src/pages/QuestDetail.tsx` — 时间线重构 + 设计方案卡片集成 + 进度条
- `web/src/pages/QuestsBoard.tsx` — 列名 fallback 优化
- `web/src/domain/questSelectors.ts` — phaseNameFallback / questPhaseNodes / getDesignPlan / groupEventsByPhase
- `web/src/components/StatusBadge.tsx` — 加 tooltip
- `web/src/components/ContextPackView.tsx` — block 名从集中配置读
- `web/src/components/CreateQuestSheet.tsx` — 推荐提示（克制版）
- `web/src/components/CreateAutomationSheet.tsx` — 推荐提示（克制版）
- `web/src/components/util.ts` — 或迁移到 domain/contextLabels.ts
- `web/src/api/types.ts` — QuestPhaseTask 字段确认，Kind 补充

---

## 十一、落地状态

状态：已落地。

完成范围：
- F.-1：`questPhaseNodes` / `phaseDisplayName` / `getDesignPlan` 已集中到 `domain/questSelectors.ts`，旧 quest 通过 legacy fallback 保持可读。
- F.0a：`QuestDetail` 主时间线已改为基于 phase nodes 渲染，不再写死两阶段主节点。
- F.0b：`groupEventsByPhase` 已按 `session_id`、`phase_idx`、当前阶段兜底归属 agent update 事件。
- F.1：`DesignPlanCard` 已接入，`design_plan` 按 kind 精确优先查找，文本类 artifact 可展开渲染 Markdown 正文。
- F.2：状态徽章 tooltip 显示当前阶段名；终审提示文案按当前评审阶段生成；创建委托/自动化增加克制版设计阶段推荐提示；ContextPack block 名已集中到 `domain/contextLabels.ts`。
- 看板列名复用 `phaseDisplayName`，避免显示 `warrior_design` / `mage_design_review` 这类内部名。

验证：
- `npm run build`
- `go test ./...`

未做范围：
- F.3 的高级视觉增强不作为本 spec 收工条件，包括多阶段折叠、设计/执行阶段分组视觉、设计方案章节级折叠。
