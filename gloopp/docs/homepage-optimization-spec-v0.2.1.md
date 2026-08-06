# Gloop 首页优化 Spec v0.2.1

> 基于当前代码实现（focus / board / list 三形态）的重新 Review，对照 refactor-spec-v1 / hotl-spec-v0.1 / design-phase-spec-v0.1 三份后端架构 spec，输出首页（Quests 入口页）的优化方向与实施规格。
> 状态：部分实现（P0 主体 + 部分 P1/P2 只读展示已落地，控制面/批量操作未做）

**本版原则：**
- P0 只用**已有字段**做"可解释首页"，不用后端愿景当 UI 合同
- P1 补**前后端契约**后做 phase / artifact 可视化
- P2 定义**策略视图模型**后上 HOTL 控制面
- 安全操作一律后置，先解释再操作，先单项再批量
- 所有 P0 展示 nullable-safe，字段缺失不报错不空白

---

## 0. 实现状态（2026-06-22）

按当前 `v0.1.2` 候选实现对齐：

### 已实现

- **Focus attention 分区**：`QuestsBoard.tsx` 已拆出等待审核、等待输入、已阻塞、应用失败等 attention sections，并保留近期待处理/进行中/近期结果的 Focus command center。
- **停止原因透明化**：卡片已展示 `waiting_input.question_text`、`failure_attribution`、`blocked_reason`、`resume_count`、`apply_error` 等 nullable-safe 文案。
- **策略痕迹 / Effect Type / 结案方式**：卡片已展示 `policy_view.last_policy_decision`、`auto_passed_by_policy`、`auto_completed_by_policy`、`finalized_by`、`effect_type` 等 chips。
- **产物与设计摘要**：卡片已展示 outputs 计数、类型 chips 和 `design_summary` 摘要；完整内容仍进入详情页。
- **阶段展示**：卡片已展示当前 phase 名称、`Phase x/y`、返工次数；Board 已支持 `groupBy=phase` 的阶段视图。
- **P0/P1 字段类型化**：`web/src/api/types.ts` 已补齐 pipeline、phases、waiting_input、failure_attribution、policy_view、effect_type、auto-complete/auto-spawn 等字段。

### 部分实现

- **状态 pill 可点击过滤**：当前已有视图切换和 filters，但未做完整“点击 pill → 自动滚动到对应列”的强交互。
- **阶段视图**：已支持按 phase 分组，但还不是完整的 pipeline_hash 分组和高级阶段泳道分析。
- **产物 chips 增强**：已展示计数/类型；安全摘要弹窗未做。
- **设计任务视觉区分**：已有 design summary / design doc / phase 信息展示，但未做完整设计任务专属视觉体系。

### 未实现 / 延后

- **P2 HOTL 控制面**：信任层级调整、安全底线详情、恢复策略控制、单项快速操作、批量操作均未做。
- **精确等待时长指标**：没有基于 events.jsonl 做 attention duration 统计。
- **批量操作**：没有进入 List 批量模式。

## 1. 当前实现现状评估

### 1.1 三形态概览

当前首页（`QuestsBoard.tsx`）已实现三种视图模式，共享数据层与 QuestCard 组件，差异在布局与信息密度：

| 模式 | 定位 | 核心布局 | 信息密度 | 默认分组 |
|------|------|----------|----------|----------|
| **Focus** | 指挥中心 / 每日第一眼 | 双列：需关注 + 进行中 + 近期结果 | 中（卡片） | 按优先级（attention priority） |
| **Board** | 全局看板 / 流程视角 | 水平滚动看板，按状态分列 | 中低（卡片） | 按状态（status kanban） |
| **List** | 高密度列表 / 批量浏览 | 垂直分组列表，行式布局 | 高（行） | 按状态（attention 优先） |

### 1.2 做得好的地方

1. **三形态分层清晰**：从"盯盘"（focus）→"全局"（board）→"批量浏览"（list），覆盖不同使用场景
2. **状态可视化完整**：9+ 种 quest 状态均有对应视觉标识
3. **Focus 优先级排序**：`focusPriority()` 已覆盖 blocked / apply_failed / failed / running 等高风险项，但 waiting_input 尚未被 Focus attention 充分承载
4. **实时更新**：SSE 全局流 + 250ms 防抖刷新
5. **筛选/排序/分组体系完整**：groupBy / sortBy / visibleFields 可配
6. **Attention 机制**：user_review / blocked / apply_failed 有 amber 左侧强调条

### 1.3 核心问题

当前首页本质上仍是"任务列表视图"，面向的是**2 阶段简单循环**（执行 → 评审）。三份后端重构 spec 引入的核心概念，在首页上几乎没有承载：

| 架构能力 | 首页现状 | 缺失影响 |
|----------|----------|----------|
| **阶段流水线** | 已展示当前 phase 与 phase board，尚未做完整 pipeline_hash 泳道分析 | 用户能看到当前阶段，但高级流程分析仍不足 |
| **结构化产物** | 已展示 outputs 计数 / 类型 chip / design summary，完整内容进详情页 | 首页仍不直接预览产物内容 |
| **HOTL 策略痕迹** | 已展示 `policy_view` / auto-pass / auto-complete chips | 控制面和策略调整仍未做 |
| **Effect Type** | 已展示 effect type chip | 安全底线详情页/解释面板未做 |
| **恢复尝试** | 已展示 `resume_count` / failure attribution；recovery summary chip 部分支持 | 仍缺完整恢复策略控制面 |
| **Waiting Input 详情** | 已类型化并展示问题摘要、asker/phase | 快速内联回复未做 |
| **设计阶段** | 已通过 phase line、design summary、design doc/execute quest 关联展示 | 首页专属设计视觉体系仍未完整落地 |

### 1.4 字段现状：前端类型与后端数据的 Gap

重要：前端 `QuestMeta` 类型与后端实际返回的数据存在多处不一致。

#### 前端声明了但后端不返回的字段

| 前端声明字段 | 实际状态 |
|-------------|----------|
| `turns_used` | ❌ 不存在。回合数在 `phases[].turns`（每 phase） |
| `max_turns` | ❌ 不存在。后端有 `max_turns_per_phase_override` |
| `duration_used_ms` | ❌ 不存在。需从 `started_at_ms` + `completed_at_ms` 计算 |
| `max_duration_ms` | ❌ 不存在。后端有 `max_duration_per_quest_ms_override` |
| `notes` | ❌ 不存在。Notes 在事件流中，不在 quest meta 上 |
| `comments` | ❌ 不存在。同上 |

#### 后端返回且已类型化的关键字段

以下字段已在 `web/src/api/types.ts` 类型化，并被 `QuestsBoard` / `QuestDetail` / selectors 使用：

| 后端实际字段 | 前端状态 |
|-------------|----------|
| `pipeline_version` | ✅ 已类型化 |
| `current_phase_idx` | ✅ 已类型化 |
| `phase_count` | ✅ 已类型化 |
| `pipeline_name` | ✅ 已类型化 |
| `pipeline_def_hash` | ✅ 已类型化 |
| `pipeline_def` | ✅ 已类型化（QuestPhaseTask[]） |
| `phases` | ✅ 已类型化（QuestPhaseTask[]） |
| `waiting_input` 对象 | ✅ 已类型化（WaitingInputState） |
| `blocked_reason_code` | ✅ 已类型化 |
| `resume_count` | ✅ 已类型化 |
| `finalized_by` | ✅ 已类型化 |
| `policy_view` / `last_policy_decision` | ✅ 已类型化 |
| `auto_completed_by_policy` | ✅ 已类型化 |
| `auto_spawn_execute` / `child_execute_quest_id` / `spawn_error` | ✅ 已类型化 |

---

## 2. 优化目标与原则

### 2.1 设计目标

1. **降低人的介入判断成本**：首页第一眼回答三个问题——"为什么停了、我该不该管、点哪里处理"
2. **承载架构能力**：让 refactor-spec 的核心概念（阶段流水线、HOTL 痕迹、设计阶段）在首页可感知
3. **保持三形态分层**：不推翻现有 focus / board / list 框架，在每层上做增量增强
4. **渐进式暴露复杂度**：默认简洁，需要时可展开，不破坏"轻量首页"体验
5. **解释先行，操作后置**：先让用户看懂系统在干什么、为什么，再谈放权和批量操作

### 2.2 设计原则

- **默认简洁，按需展开**：核心信息一目了然，详情点击/hover 查看
- **语义化视觉**：每种架构概念有对应的视觉语言，与现有设计系统一致
- **只读优先**：P0 全只读，P1 无新增操作，P2 才上控制面
- **Fallback 友好**：旧数据 / 缺数据时优雅降级，不报错不空白
- **不用 emoji，用 Icon**：图标体系走 lucide / 语义图标，不用 emoji 文案
- **Nullable-safe**：所有新增展示都处理字段缺失 / null / undefined 的情况

---

## 3. P0：可解释首页（无新增后端 API）

**目标：让用户一眼看懂"为什么停、我该不该管、点哪里处理"**

**后端依赖：无新增后端 API。使用已有的 QuestMeta 字段，前端做类型补齐 + nullable fallback。**

### 3.1 Focus 视图：需关注分区细化

#### 现状

一个大的"需关注"列表，所有 attention 状态混在一起。

#### 优化

拆分为四个语义分区，每个分区有独立标题和解释文案。

| 分区 | 包含状态 | 视觉标识 | 解释文案 |
|------|----------|----------|----------|
| **等待审核** | user_review | amber attention bar + shield icon | 需要你做最终审核 / apply |
| **等待输入** | waiting_input | amber attention bar + message-circle icon | Agent 在问你问题 |
| **已阻塞** | blocked | red border + alert-triangle icon | 执行卡住了，需要排查 |
| **应用失败** | apply_failed | red attention bar + x-circle icon | Apply 环节出错了 |

每个分区：
- 独立折叠块，空状态自动隐藏
- 分区标题右侧显示数量徽章
- 分区顺序按优先级：等待审核 > 已阻塞 > 应用失败 > 等待输入
- 点击分区标题跳转到 Board 视图对应列（或 List 对应分组）

**实现要点：**
- 分区标题旁加一个小的 info tooltip，解释"这个状态是什么意思、为什么需要你"
- 不新增任何操作按钮，点击卡片照常进详情页
- 所有字段 nullable：`blocked_reason` 为空时显示"已阻塞"通用文案

### 3.2 卡片增强：停止原因透明化

对 **user_review / blocked / waiting_input / apply_failed** 四种 attention 状态的卡片，增加"为什么停了"的解释行。

#### 3.2.1 Blocked 卡片

现有：只显示 `blocked_reason` 字符串。

增强：
- 第一行：`failure_attribution.category` + `failure_attribution.reason`（结构化展示）
- 第二行（`recoverable === true` 时显示）：可自动恢复 · 已重试 `resume_count` 次
- 第二行（`recoverable === false` 时显示）：需人工介入
- 如果 `failure_attribution` 不存在，fallback 到 `blocked_reason` 纯文本

**字段来源：** `failure_attribution`、`resume_count`、`blocked_reason_code`（均已在前端类型化）

**Nullable 策略：**
- `failure_attribution` 为 null 时：只显示 `blocked_reason`
- `blocked_reason` 也为空时：显示"已阻塞"通用文案
- `resume_count` 为 null / 0 时：不显示重试次数行

#### 3.2.2 Waiting Input 卡片

现有：只显示"等待输入"状态提示。

增强：
- 显示问题摘要：`waiting_input.question_text` 的前 60 字符（单行省略）
- 显示提问者：`${asker}（${phase_type}）`
- 显示已等待时长（从 `asked_at` 计算）

**字段来源：** `waiting_input` 对象（已在前端类型化）

**Nullable 策略：**
- `waiting_input` 为 null 时：只显示状态标签，不显示问题摘要
- `question_text` 为空时：显示"等待输入"通用文案

#### 3.2.3 User Review 卡片

现有：状态提示 + final_verdict 等。

增强：
- 显示 `effect_type` 徽章：如果是 `workspace_diff` / `external_side_effect` 等，显示对应图标 + label
- 显示 `auto_passed_by_policy` 标记：如果非空，显示"策略 ${auto_passed_by_policy} 建议通过"
- 显示 `workspace_diff_pending` 标记：有 diff 待审核

**字段来源：** `effect_type`（已有）、`auto_passed_by_policy`（已有）、`workspace_diff_pending`（已有）

**Nullable 策略：**
- 每个标记字段为空时直接不显示该标记，不占位
- `effect_type` 为 `none` 或空时不显示

#### 3.2.4 Apply Failed 卡片

现有：显示 `apply_error` 等。

增强：
- 显示 `apply_error` 精简版（前 80 字符）
- 显示 `finalized_by`：谁触发的 apply（user / policy / automation）

**字段来源：** `apply_error`、`finalized_by`（均已在前端类型化）

**Nullable 策略：**
- `apply_error` 为空时：显示"应用失败"通用文案
- `finalized_by` 为空时：不显示触发者

### 3.3 卡片底部：策略痕迹

在卡片 footer 区域增加轻量策略痕迹（小字）。

#### 3.3.1 已完成卡片（success / failed）

增加一行结案方式标记：
- 如果 `auto_passed_by_policy` 非空：显示"策略通过 · ${auto_passed_by_policy}"
- 如果 `finalized_by` 为 `policy`：显示"策略结案"
- 如果 `finalized_by` 为 `user`：显示"人工结案"
- 如果 `finalized_by` 为 `automation`：显示"自动结案"
- 都没有时：不显示该行

**字段来源：** `auto_passed_by_policy`、`finalized_by`

#### 3.3.2 进行中卡片（running / reviewing）

增加一行 effect_type 小徽章（小型 badge）：
- `workspace_diff`：file-diff 图标 + "有文件变更"
- `external_side_effect`：external-link 图标 + "有外部副作用"
- `context_store`：database 图标 + "有上下文存储"
- `none` 或空：不显示

**字段来源：** `effect_type`

### 3.4 产物摘要

#### 现状

`outputs` 数组已有但首页完全没用到。

#### 优化

卡片底部增加产物计数 chips（只显示数量 + 类型，不预览内容）：
- `file-text` 图标 + `${outputs.length} 个产物`（输出数目统计）
- 如果有 `design_summary` 的 quest，显示 `design_summary` 第一行作为摘要（1 行，省略号截断）

**字段来源：** `outputs`（已有类型 `QuestArtifact[]`）、`design_summary`（已有）

**安全边界：**
- P0 只显示计数和类型 chips，不渲染产物内容
- `design_summary` 只显示第一行纯文本，不做 markdown 渲染
- 点击产物详情一律进详情页或已有 modal

**Nullable 策略：**
- `outputs` 为空数组或 null 时：不显示产物 chip
- `design_summary` 为空或 null 时：不显示摘要行

### 3.5 状态 Pill 可点击过滤

Focus 视图顶部的 4 个状态药丸（待执行 / 执行中 / 评审中 / 需关注），点击后：
- 切换到 Board 视图
- 应用对应状态过滤
- 自动滚动到对应列

实现方式：点击 pill 后设置 filter 到对应状态 + 切到对应视图。

### 3.6 前端类型补齐 + 迁移策略

P0 必须完成的类型工作：补齐类型 + 处理旧字段迁移。

#### 3.6.1 新增类型

```typescript
// Phase / pipeline (基础字段，P0 先类型化，P1 才用)
pipeline_version: number | null
current_phase_idx: number | null
phase_count: number | null
pipeline_name: string | null
pipeline_def_hash: string | null

// Waiting input
waiting_input: WaitingInputState | null

// Blocked / recovery
blocked_reason_code: string | null
resume_count: number | null

// Finalization
finalized_by: 'user' | 'policy' | 'automation' | '' | null
```

`WaitingInputState` 类型：
```typescript
interface WaitingInputState {
  question_id: string
  question_text: string
  phase_idx: number
  session_id?: string
  phase_type?: string
  asker?: string
  timeout_ms?: number
  timeout_action?: string
  resume_phase_idx?: number
  asked_at?: string  // ISO time string, 需转 ms 计算；非法或空值 fallback 到 quest activity time
}
```

#### 3.6.2 旧字段迁移策略

以下字段前端类型声明了但后端不返回，需明确替换策略，不能让现有页面继续用错误字段：

| 旧字段 | 迁移策略 | 影响范围 |
|--------|----------|----------|
| `turns_used` | 回合数从 `phases[].turns` 聚合（所有 phase 回合总和），或只展示当前 phase 的 turns。P0 先修正类型 + 修正现有展示逻辑。 | QuestsBoard budget line、QuestDetail |
| `max_turns` | 预算上限从 `max_turns_per_phase_override`（每 phase）或 config 默认值取。没有全局 max_turns；如果当前组件拿不到 config，只展示已用回合，不硬展示 `/ 上限`。进度条不应用回合数伪装完成度。 | QuestsBoard budget line |
| `duration_used_ms` | 已用时长从 `started_at_ms` + 当前时间（running）或 `completed_at_ms`（已结束）计算。 | QuestsBoard、QuestDetail |
| `max_duration_ms` | 时长上限从 `max_duration_per_quest_ms_override` 或 config 默认值取；如果当前组件拿不到 config，只展示已用时长，不硬展示 `/ 上限`。 | budget 展示 |
| `notes` | 从事件流 / trace 接口获取，不在 quest meta 上。现有引用需修正。 | QuestDetail |
| `comments` | 同上。 | QuestDetail |

**重要：** P0 必须处理现有页面中 `turns_used` / `max_turns` / `duration_used_ms` / `max_duration_ms` 的引用，否则展示是空的 / 错误的。进度条不要用"预算消耗比例"伪装成完成度——它不是完成度，只是资源消耗指示，用 text / meta line 展示即可。

---

## 4. P1：阶段与产物可视化（补前后端契约）

**目标：让用户看懂 quest 在流水线中的位置和产物有什么**

**后端依赖：确认 pipeline_def / phases / pipeline_def_hash 字段语义稳定；可选补 phase_history 或 rework_events 用于返工历史展示。**

### 4.1 Board 视图：阶段视图

#### 4.1.1 看板双模式

工具栏增加切换：**按状态（默认）/ 按阶段**

- **按状态**：保持现状，10 列，面向执行监控
- **按阶段**：按 pipeline_def 分列，面向流程理解

#### 4.1.2 阶段列生成规则

阶段列**不硬编码，动态生成**。核心原则：**按 `pipeline_def_hash` 分组，同 hash 内按 `phase_idx` 成列**。

```
阶段视图
├─ Pipeline: default-execute (hash: abc123)   ← 泳道 1
│  ├─ 战士执行  →  法师评审  →  用户审核  →  完成
│  └─ (每张卡片是一个 quest)
│
└─ Pipeline: design-full (hash: def456)       ← 泳道 2
   ├─ 战士设计  →  法师设计评审  →  战士执行  →  法师实现评审  →  用户审核  →  完成
   └─ (每张卡片是一个 quest)
```

生成规则：
1. 按 `pipeline_def_hash` 对所有 quest 分组（相同 hash = 相同 pipeline 定义）
2. 每组内从 `pipeline_def` 中提取 phase 列表，按 `phase_idx` 排序成列
3. 每列显示 `display_name`（或 `name`） + 数量徽章
4. 列的 role（warrior/mage）用颜色区分（warrior 金 / mage 紫）
5. 多个 pipeline 时，垂直排列成多个泳道（每个泳道是一组阶段列）
6. 当 pipeline 泳道超过 3 个时，按 quest 数保留主泳道展开，其余泳道默认折叠

**不跨 pipeline 硬合并 phase**——不同 pipeline 的 `phase_idx` 语义可能完全不同，不能对齐合并。

#### 4.1.3 Fallback 策略

**重要：** 不是所有 quest 都有完整 pipeline 数据。UI 必须处理以下情况：

- 有 `pipeline_version` 且 > 0 且 `phases` 数组非空 → 显示阶段视图
- 没有 pipeline 数据的 quest（`pipeline_version` 为空或 0）：
  - 在阶段视图中归到"旧数据"单独列
  - 或在卡片上显示"阶段信息不可用"标记
  - UI 不假设所有 quest 都有完整 pipeline
- `pipeline_def` 缺失但 `phases` 存在 → 用 `phases` 数组推断列
- 单一场景下 quest 属于不同 pipeline 的情况较少，优先展示主 pipeline，其他折叠

### 4.2 卡片增强：阶段进度

#### 4.2.1 阶段指示器

卡片上显示阶段进度信息：
- 格式：`Phase ${current_phase_idx + 1}/${phase_count} · ${phase_name}`
- 或用文字进度条形式：显示已完成 phase 比例（纯文字 / 细线，不用粗进度条伪装完成度）
- 显示在当前 phase 的停留时长（从 `phases[current_phase_idx].started_at_ms` 计算）

**字段来源：** `current_phase_idx`、`phase_count`、`phases[].name` / `display_name`、`phases[].started_at_ms`

**Nullable 策略：**
- `current_phase_idx` 或 `phase_count` 为空时：不显示阶段指示器
- `phases` 数组为空时：fallback 到只显示状态

#### 4.2.2 返工标记

如果 `rework_count > 0`，显示返工标记：
- rotate-ccw 图标 + `${rework_count} 次返工`
- P1 只显示次数，**不展示返工历史路径**（历史路径需要 `phase_history` 或 `rework_events` 数据，当前 quest meta 上没有，放入 P2 或后端补契约）

### 4.3 产物 Chips 增强

P0 的产物计数 chips 增强：
- 按 `kind` 分类显示：`file` / `document` / `image` / `log` / `archive` 等
- 点击产物 chip 打开**安全摘要弹窗**
  - 产物列表（名称 + 大小 + 来源）
  - 纯文本列表，不渲染内容
  - "查看详情"按钮跳详情页

**安全边界：**
- 不直接渲染 artifact 内容（避免 XSS 风险）
- 只显示元数据（名称、大小、类型、来源）
- 完整预览进详情页

### 4.4 设计任务的视觉区分

设计类 quest（`type=design`）：
- 使用 mage 语义色（紫色系，与现有 mage 色一致）
- 细左边线（类似 attention bar 样式，紫色）
- 顶部 wand 图标 + "设计" badge
- 不用渐变边框，不用花哨装饰
- 阶段名称用 `pipeline_def` 里的实际 phase name，不硬编码"战士设计 → 法师设计评审"

---

## 5. P2：HOTL 控制面（定义策略视图模型后）

**目标：让用户能管理信任层级、调整策略、做批量操作**

**前提：后端定义并实现 quest 级 policy view model。前端不能从现有 QuestMeta 推导这些数据。**

### 5.1 后端契约：Quest 级策略视图模型

需要后端新增并返回以下字段（挂在 quest meta 上，或独立 API）：

```typescript
// Quest 级策略视图模型（需后端新增）
effective_trust_tier: 'tier_0' | 'tier_1' | 'tier_2' | 'tier_3' | null
// 注意：来源可能是 automation 快照 + 当前策略决策事实聚合

safety_floor_reason: string | null
// 安全底线触发原因，null 表示未触发

last_policy_decision: {
  id: string
  policy_name: string
  action: string  // 原始 action 字符串，前端做已知映射
  // 已知 action: pass / escalate / block / retry / require_user /
  //               auto_pass / auto_apply / pending_apply / continue
  // 未知 action: 直接显示原始 label
  reason: string
  at_ms: number
} | null

recovery_state_summary: {
  strategy: 'retry' | 'degrade' | 'escalate' | 'skip_phase' | string
  attempt: number
  max_attempts: number
  next_attempt_at_ms: number | null
  status: 'active' | 'exhausted' | 'waiting'
} | null
```

**设计原则：** `action` 和 `strategy` 用 `string` 原始值 + 前端已知映射，不要过早收窄为前端枚举。平台策略还在演进，UI 不该卡死语义。未知值直接显示原始 label。

### 5.2 信任层级显示

- 所有 quest 卡片/行显示信任层级徽章：
  - t0：手动（灰色 / 已暂停）
  - t1：确认运行（琥珀色）
  - t2：自动 + 审（蓝色 / 品牌色）
  - t3：全自动（绿色 / 成功色）

- 徽章点击可调整层级（弹窗确认）：
  - 调高："确认调高到 t2？" + 影响说明
  - 调低："确认调低到 t1？" + 影响说明
  - 每次调整有风险解释文案
  - 调整后有审计事件

**安全要求：**
- 必须有后端 API 支持
- 必须有确认对话框
- 必须有审计事件
- 必须有失败处理

### 5.3 安全底线提示

对于被安全底线拦截的 quest：
- 卡片上显示 shield-alert 图标 + "安全底线" 标记
- hover 显示具体原因：`workspace_diff` / `external_side_effect` / `allow_l2`
- 说明"不受信任层级影响，始终需要人工审核"

### 5.4 恢复策略可视化

对于 blocked 且有 active recovery 的 quest：
- 显示恢复策略类型 + 进度：已重试 N/M 次
- 下次重试时间倒计时
- "立即重试"按钮（需确认）
- "跳过此阶段"按钮（需确认 + 风险提示）

### 5.5 单项快速操作

P2 才做，P0 / P1 都不做。

支持的快速操作（卡片 hover 显示）：

| 操作 | 适用状态 | 安全等级 | 确认要求 |
|------|----------|----------|----------|
| 查看 diff | user_review | 低（只读） | 否 |
| 立即重试 | blocked | 中 | 是 |
| 跳过阶段 | blocked | 高 | 是 + 风险解释 |
| 停止 | running | 中 | 是 |
| 开始 | pending | 低 | 否 |
| 取消 | pending | 中 | 是 |
| 调整信任层级 | 所有 | 高 | 是 + 影响说明 |

**所有操作都走后端对应 API，成功失败都有 toast 反馈。**

### 5.6 批量操作

P2 后期才做，且在 List 视图中做。

- 进入"批量模式"后每行显示复选框
- 支持的批量操作（取决于选中项状态）：
  - 批量通过审核（user_review 状态）
  - 批量重试（blocked 状态）
  - 批量取消（pending / running）
  - 批量调整信任层级

**安全要求：**
- 必须有明确的数量确认（选中 N 项，其中 M 项可执行）
- 必须处理部分成功 / 部分失败的展示
- 必须有审计事件
- 必须有"仅对相同状态的操作"约束

---

## 6. 视觉与交互规范

### 6.1 图标体系

使用 lucide / 现有 Icon 组件语义，不用 emoji。

| 概念 | Icon 名 | 用途 |
|------|--------|------|
| 等待审核 | shield-check / shield | user_review |
| 等待输入 | message-circle / help-circle | waiting_input |
| 已阻塞 | alert-triangle | blocked |
| 应用失败 | x-circle | apply_failed |
| 策略通过 | check-circle | auto_passed_by_policy |
| 产物 | file-text | outputs |
| 设计方案 | wand / sparkles | design type |
| 返工 | rotate-ccw | rework |
| 信任层级 | shield / zap | trust tier |
| 安全底线 | shield-alert | safety floor |
| 恢复 / 重试 | refresh-cw | recovery |
| 外部副作用 | globe / external-link | external_side_effect |
| 文件变更 | git-branch / file-diff | workspace_diff |

### 6.2 颜色体系

沿用现有 Mew 风格（黑/白/灰 + 单强调色 + 语义色），不新增主色调：

- **Attention / 琥珀色**：`--attention`（user_review、waiting_input、t1）
- **Danger / 红色**：`--danger`（blocked、apply_failed）
- **Success / 绿色**：`--success`（success、t3）
- **Brand / 蓝色**：`--brand`（t2、策略通过）
- **Mage / 紫色**：`--mage-violet`（设计类、法师阶段）
- **Warrior / 金色**：`--warrior-gold`（战士阶段）

不用渐变边框，不用花哨装饰。

### 6.3 动效

- 状态切换：轻量 fade + slide（< 200ms）
- 新 attention 项出现：amber 轻微闪烁一次（吸引注意但不打扰）
- 不做呼吸动效（表示"系统在工作"）

---

## 7. 信息架构

保持现有导航结构不变：
委托 → 收件箱 → 自动化 → 知识 → 冒险者 → 技能 → 提示词 → 统计

首页定位从"任务列表"升级为"循环指挥中心"：
- 一眼看全系统运行状态
- 快速理解需要人工介入的事项
- 了解各阶段流转情况
- （P2）管理自动化信任程度

---

## 8. 成功度量

### 8.1 本地可计算指标（无需埋点）

直接从 quest 数据 / 事件数据计算得到：

| 指标 | 计算方式 | 数据来源 | 意义 |
|------|----------|----------|------|
| Attention 队列平均等待时长 | user_review / blocked / waiting_input 从进入 attention 到处理的平均时长 | 优先 events.jsonl 状态变迁事件；没有事件时只能用当前状态的 `updated_at_ms` 作为进入状态时间近似，或不展示该指标，近似值标记为 estimated | 反映人工介入的响应速度 |
| Waiting input 平均等待时长 | 同上，仅 waiting_input | 同上 | Agent 等人类回答的速度 |
| Blocked 恢复率 | blocked → running 的比例 / 自动恢复比例 | events.jsonl 状态变迁事件 | 反映恢复策略效果 |
| 策略自动通过率 | `auto_passed_by_policy` 非空 / 已完成 quest 比例 | quest meta | 反映 HOTL 自动化程度 |
| 各阶段平均停留时长 | `phases[].ended_at_ms` - `phases[].started_at_ms` 平均 | `phases` 数组 | 反映流水线瓶颈 |

**事件边界说明：**
- 优先使用 `events.jsonl` 中的状态变迁事件计算精确时长
- 没有事件数据时，退化使用 quest meta 的 `updated_at_ms` / `started_at_ms` / `completed_at_ms` 近似计算
- 近似值在 UI 上标记为"约"或"估计"，不冒充精确值

### 8.2 可选 UI Telemetry（本地事件）

如果后续加本地前端事件统计（不发远端）：

| 事件 | 意义 |
|------|------|
| `dashboard_open` | 首页打开次数 |
| `quest_detail_open` | 进入详情页次数 |
| `quick_action_used` | 快速操作使用次数 |
| `view_toggle` | 视图切换次数 |

---

## 9. 实施路线图

| 阶段 | 目标 | 主要工作 | 后端依赖 |
|------|------|----------|----------|
| **P0** | 可解释首页 | 需关注分区、停止原因透明化、策略痕迹、产物计数、状态 pill 可点击、旧字段迁移 | 无新增后端 API（前端类型补齐 + nullable fallback） |
| **P1** | 阶段与产物可视化 | 阶段视图（按 pipeline_hash 分组）、阶段进度、产物 chips + 摘要弹窗、设计任务视觉区分 | pipeline_def / phases 字段语义确认；可选补 phase_history |
| **P2** | HOTL 控制面 | 信任层级显示与调整、安全底线提示、恢复策略可视化、单项快速操作、批量操作 | quest 级 policy view model、操作 API、审计事件 |

---

## 附录 A：与三份 Spec 的对应关系

| 优化项 | refactor-spec-v1 | hotl-spec | design-phase |
|--------|:----------------:|:----------:|:------------:|
| 停止原因透明化 | ✅ 状态机 | ✅ 策略可解释性 | |
| 策略痕迹展示 | | ✅ ReviewPolicy | |
| Effect type 展示 | | ✅ Global Safety Floor | |
| 产物计数 + 摘要 | ✅ 结构化产物 | | ✅ 设计产物 |
| 阶段视图 | ✅ Phase Pipeline | | ✅ 4 阶段流水线 |
| 阶段进度 | ✅ PhaseDef / ReworkTo | | |
| 返工可视化 | ✅ ReworkTo | | |
| 信任层级显示与调整 | | ✅ Trust Tiers | |
| 安全底线提示 | | ✅ Global Safety Floor | |
| 恢复策略可视化 | | ✅ RecoveryPolicy | |
| 等待输入详情 | | ✅ waiting_input 状态 | |
| 设计任务视觉区分 | | | ✅ 核心 |
| 快速操作 | ✅ QuestService API | ✅ Review+Apply | |
| 批量操作 | ✅ 批量 API | ✅ 批量审核 | |

## 附录 B：字段使用清单

### B.1 P0 已存在的字段

| 字段 | 前端类型状态 | 用途 | Nullable 处理 |
|------|-------------|------|--------------|
| `failure_attribution` | ✅ 已有类型 | blocked 原因结构化展示 | 为空时 fallback 到 blocked_reason |
| `blocked_reason` | ✅ 已有 | blocked 原因文本 | 为空时显示通用文案 |
| `waiting_input` 对象 | ✅ 已类型化 | waiting input 问题文本、提问者 | 为空时只显示状态标签 |
| `effect_type` | ✅ 已有 | effect 类型徽章 | 为 none / 空时不显示 |
| `auto_passed_by_policy` | ✅ 已有 | 策略自动通过标记 | 为空时不显示 |
| `policy_decision_id` | ✅ 已有 | 策略决策 ID（link） | 为空时不显示 |
| `finalized_by` | ✅ 已类型化 | 结案方式 | 为空时不显示 |
| `apply_error` | ✅ 已有 | apply 失败原因 | 为空时显示通用文案 |
| `outputs` | ✅ 已有类型 | 产物计数 | 为空时不显示 |
| `design_summary` | ✅ 已有 | 设计摘要 | 为空时不显示 |
| `resume_count` | ✅ 已类型化 | 恢复重试次数 | 为 0 / null 时不显示 |
| `workspace_diff_pending` | ✅ 已有 | diff 待审核标记 | 为 false / null 时不显示 |

### B.2 P1 需要类型化字段

| 字段 | 状态 | 用途 |
|------|------|------|
| `pipeline_version` | ✅ 已类型化 | 阶段视图开关判断 |
| `current_phase_idx` | ✅ 已类型化 | 当前阶段索引 |
| `phase_count` | ✅ 已类型化 | 总阶段数 |
| `pipeline_name` | ✅ 已类型化 | pipeline 名称 |
| `pipeline_def_hash` | ✅ 已类型化 | pipeline 分组依据 |
| `pipeline_def` | ✅ 已类型化 | 阶段定义（列生成） |
| `phases` | ✅ 已类型化 | 阶段运行时状态 |

### B.3 P2 需要后端新增

| 字段 | 状态 | 用途 |
|------|------|------|
| `effective_trust_tier` | ✅ policy_view 已返回；调整 API 未做 | 信任层级显示与调整 |
| `safety_floor_reason` | ✅ policy_view 已返回 | 安全底线触发原因 |
| `last_policy_decision` | ✅ policy_view 已返回 | 最近策略决策 |
| `recovery_state_summary` | ✅ policy_view 已返回；控制面未做 | 恢复状态摘要 |
