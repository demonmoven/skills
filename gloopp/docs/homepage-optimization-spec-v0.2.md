# Gloop 首页优化 Spec v0.2

> 基于当前代码实现（focus / board / list 三形态）的重新 Review，对照 refactor-spec-v1 / hotl-spec-v0.1 / design-phase-spec-v0.1 三份后端架构 spec，输出首页（Quests 入口页）的优化方向与实施规格。
> 状态：已废弃（superseded by `homepage-optimization-spec-v0.2.1.md`）

> 说明：本文件保留为 v0.2 过渡稿。当前实现对齐、nullable 处理和 P0/P1/P2 完成情况以 `homepage-optimization-spec-v0.2.1.md` 为准。

**本版原则：**
- P0 只用**已有字段**做"可解释首页"，不用后端愿景当 UI 合同
- P1 补**前后端契约**后做 phase / artifact 可视化
- P2 定义**策略视图模型**后上 HOTL 控制面
- 安全操作一律后置，先解释再操作，先单项再批量

---

## 1. 当前实现现状评估

### 1.1 三形态概览

当前首页 (`QuestsBoard.tsx`) 已实现三种视图模式，共享数据层与 QuestCard 组件，差异在布局与信息密度：

| 模式 | 定位 | 核心布局 | 信息密度 | 默认分组 |
|------|------|----------|----------|----------|
| **Focus** | 指挥中心 / 每日第一眼 | 双列：需关注 + 进行中 + 近期结果 | 中（卡片） | 按优先级（attention priority） |
| **Board** | 全局看板 / 流程视角 | 水平滚动看板，按状态分列 | 中低（卡片） | 按状态（status kanban） |
| **List** | 高密度列表 / 批量浏览 | 垂直分组列表，行式布局 | 高（行） | 按状态（attention 优先） |

### 1.2 做得好的地方 ✅

1. **三形态分层清晰**：从"盯盘"（focus）→"全局"（board）→"批量浏览"（list），覆盖不同使用场景
2. **状态可视化完整**：9+ 种 quest 状态均有对应视觉标识
3. **Focus 优先级排序**：`focusPriority()` 按 user_review > blocked > apply_failed > waiting_input 排序
4. **实时更新**：SSE 全局流 + 250ms 防抖刷新
5. **筛选/排序/分组体系完整**：groupBy / sortBy / visibleFields 可配
6. **Attention 机制**：user_review / blocked / apply_failed 有 amber 左侧强调条

### 1.3 核心问题

当前首页本质上仍是"任务列表视图，面向的是**2 阶段简单循环**（执行 → 评审）。三份后端重构 spec 引入的核心概念，在首页上几乎没有承载：

| 架构能力 | 首页现状 | 缺失影响 |
|----------|----------|----------|
| **阶段流水线** | 只展示状态，无阶段概念 | 用户看不到 quest 处于哪个 phase、离出口还有多远、返工历史 |
| **结构化产物** | 卡片只显示 query 文本，outputs 数组未被使用 | 用户看不到设计方案 / 评审报告 / diff 等产物摘要 |
| **HOTL 策略痕迹** | `auto_passed_by_policy` / `policy_decision_id` 字段存在但未展示 | 用户不知道哪些是策略自动过的、为什么过了 |
| **Effect Type** | 有 `effect_type` 字段存在但未在首页展示 | workspace_diff / external_side_effect 等安全信号不可见 |
| **恢复尝试** | `resume_count` 存在但未展示，`recovery_state 在独立 store | 重试了几次、是不是自动恢复的，都不可见 |
| **Waiting Input 详情** | 状态有但 `waiting_input` 对象未在前端类型化 | agent 问了什么、等多久了不直观 |
| **设计阶段** | 只有 type=design 标签 | 4 阶段流水线完全不可见 |

### 1.4 字段现状：前端类型与后端数据的 Gap

重要：前端 `QuestMeta` 类型与后端实际返回的数据存在多处不一致，以下字段**前端声明了但后端不返回**：

| 前端声明字段 | 实际状态 |
|-------------|----------|
| `turns_used` | ❌ 不存在。回合数在 `phases[].turns（每 phase） |
| `max_turns` | ❌ 不存在。后端有 `max_turns_per_phase_override` |
| `duration_used_ms` | ❌ 不存在。需从 `started_at_ms` + `completed_at_ms` 计算 |
| `max_duration_ms` | ❌ 不存在。后端有 `max_duration_per_quest_ms_override` |
| `notes` | ❌ 不存在。Notes 在事件流中，不在 quest meta 上没有 |
| `comments` | ❌ 不存在。同上 |

以下字段**后端返回了但前端未类型化**（数据可达但未使用**：

| 后端实际字段 | 前端状态 |
|-------------|----------|
| `pipeline_version` | ✅ 可达，未类型化 |
| `current_phase_idx` | ✅ 可达，未类型化 |
| `phase_count` | ✅ 可达，未类型化 |
| `pipeline_name` | ✅ 可达，未类型化 |
| `pipeline_def` | ✅ 可达，未类型化（PhaseTask[]） |
| `phases` | ✅ 可达，未类型化（PhaseTask[]） |
| `waiting_input` 对象 | ✅ 可达，未类型化（WaitingInputState） |
| `blocked_reason_code` | ✅ 可达，未类型化 |
| `resume_count` | ✅ 可达，未类型化 |
| `finalized_by` | ✅ 可达，未类型化 |

---

## 2. 优化目标与原则

### 2.1 设计目标

1. **降低人的介入判断成本**：首页第一眼回答三个问题——"为什么停了、我该不该管、点哪里处理"
2. **承载架构能力**：让 refactor-spec 的核心概念（阶段流水线、HOTL 痕迹、设计阶段）在首页可感知
3. **保持三形态分层**：不推翻现有 focus / board / list 框架，在每层上做增量增强
4. **渐进式暴露复杂度**：默认简洁，需要时可展开，不破坏"轻量首页"体验
5. **先解释先行，操作后置**：先让用户看懂系统在干什么、为什么，再谈放权和批量操作

### 2.2 设计原则

- **默认简洁，按需展开**：核心信息一目了然，详情点击/hover 查看
- **语义化视觉**：每种架构概念有对应的视觉语言，与现有设计系统一致
- **只读优先**：P0 全只读，P1 单项安全操作，P2 批量操作
- **Fallback 友好**：旧数据 / 缺数据时优雅降级，不报错不空白
- **不用 emoji，用 Icon**：图标体系走 lucide / 语义图标，不用 emoji 文案

---

## 3. P0：可解释首页（只用现有字段）

**目标：让用户一眼看懂"为什么停、我该不该管、点哪里处理**

### 3.1 Focus 视图：需关注分区细化

#### 现状：一个大的"需关注"列表，所有 attention 状态混在一起。

#### 优化：拆分为四个语义分区，每个分区有独立标题和解释文案。

| 分区 | 包含状态 | 视觉标识 | 解释文案 |
|------|----------|----------|--------|
| **等待审核** | user_review | amber attention bar + shield icon | 需要你做最终审核 / apply |
| **等待输入** | waiting_input | amber attention bar + message-circle icon | Agent 在问你问题 |
| **已阻塞** | blocked | red border + alert-triangle icon | 执行卡住了，需要排查 |
| **应用失败** | apply_failed | red attention bar + x-circle icon | Apply 环节出错了 |

每个分区：
- 独立折叠块，空状态自动隐藏
- 分区标题右侧显示数量徽章
- 分区顺序按优先级：等待审核 > 已阻塞 > 应用失败 > 等待输入
- 点击分区标题可点击跳转到 Board 视图对应列（或在 List 对应分组）

**实现要点：
- 分区标题旁加一个小的 info tooltip，解释"这个状态是什么意思、为什么需要你"
- 不新增任何操作按钮，点击卡片照常进详情页

### 3.2 卡片增强：停止原因透明化

对 **user_review / blocked / waiting_input / apply_failed 四种 attention 状态的卡片，增加"为什么停了"的解释行。

#### 3.2.1 Blocked 卡片

现有：只显示 blocked_reason 字符串。

增强：
- 第一行：`failure_attribution.category + failure_attribution.reason（结构化展示）
- 第二行（如果 recoverable 为 true 时显示：`可自动恢复 · 已重试 ${resume_count} 次`
- 第二行（如果 recoverable 为 false 时显示：`需人工介入`

**字段来源**：`failure_attribution`（已有类型）、`resume_count`（后端已有、前端未类型化，P0 直接用类型补齐）、`blocked_reason_code`（可用于分类图标）

#### 3.2.2 Waiting Input 卡片

现有：只显示 "等待输入"状态提示。

增强：
- 显示问题摘要：`waiting_input.question_text` 的前 60 字符（单行省略）
- 显示提问者：`${asker} (${phase_type 或 role})`
- 显示已等待时长

**字段来源**：`waiting_input` 对象（后端已有、前端未类型化，P0 先类型补齐）

#### 3.2.3 User Review 卡片

现有：状态提示 + final_verdict 等。

增强：
- 显示 `effect_type` 徽章：如果是 `workspace_diff` / `external_side_effect` 等，显示对应图标 + label
- 显示 `auto_passed_by_policy` 标记：如果非空，显示"策略 ${auto_passed_by_policy} 建议通过"
- 显示 `workspace_diff_pending` 标记：有 diff 待审核

**字段来源**：`effect_type`（已有）、`auto_passed_by_policy`（已有）、`workspace_diff_pending`（已有）

#### 3.2.4 Apply Failed 卡片

现有：显示 apply_error 等。

增强：
- 显示 `apply_error` 精简版（前 80 字符）
- 显示 `finalized_by` 是 user / policy / automation（谁触发的 apply）

**字段来源**：`apply_error`（已有）、`finalized_by`（后端已有、前端未类型化）

### 3.3 卡片底部：策略痕迹

在卡片 footer 区域增加轻量策略痕迹（小字）。

#### 3.3.1 已完成卡片（success / failed）

增加一行：
- 如果 `auto_passed_by_policy` 非空：显示 `策略通过 · ${policy_name
- 如果 `finalized_by` 为 policy：显示 `策略结案 · ${policy_name}
- 如果 `finalized_by` 为 user：显示 `人工结案`
- 如果 `finalized_by` 为 automation：显示 `自动结案`

**字段来源**：`auto_passed_by_policy`、`finalized_by`

#### 3.3.2 进行中卡片（running / reviewing）

增加一行 effect_type 小徽章（小型 badge：
- `workspace_diff`：文件变更图标 + "有文件变更
- `external_side_effect`：外部影响图标 + "有外部副作用"
- `context_store`：知识存储图标 + "有上下文存储"
- `none`：不显示

**字段来源**：`effect_type`

### 3.4 产物摘要

#### 现状
`outputs` 数组已有但首页完全没用到。

#### 优化
卡片底部增加产物计数 chips（只显示数量 + 类型，不预览内容）：
- `📄 ${outputs.length} 个产物`（输出数目统计
- 如果有 design_summary 的 quest，显示 design_summary 第一行作为摘要（1 行，省略号截断）

**字段来源**：`outputs`（已有类型 QuestArtifact[]）、`design_summary`（已有）

**安全边界**：
- P0 只显示计数和类型 chips，不渲染产物内容
- design_summary 只显示第一行纯文本，不做 markdown 渲染
- 点击产物详情一律进详情页或已有 modal

### 3.5 状态 Pill 可点击过滤

Focus 视图顶部的 4 个状态药丸（待执行 / 执行中 / 评审中 / 需关注），点击后：
- 切换到 Board 视图
- 自动滚动到对应列
- 或者在当前视图应用对应状态

实现方式：点击 pill 后设置 filter 到对应状态 + 切到对应视图

### 3.6 前端类型补齐（P0 必须做的类型工作

P0 需要新增 / 修正前端类型，与后端实际数据对齐：

**新增类型：**

```typescript
// Phase / pipeline
pipeline_version: number
current_phase_idx: number
phase_count: number
pipeline_name: string

// Waiting input
waiting_input: WaitingInputState | null

// Blocked / recovery
blocked_reason_code: string
resume_count: number

// Finalization
finalized_by: 'user' | 'policy' | 'automation' | ''
```

**修正 / 弃用：**
- `turns_used`、`max_turns`、`duration_used_ms`、`max_duration_ms`、`notes`、`comments` — 标记为 deprecated 或移除

---

## 4. P1：阶段与产物可视化（补前后端契约）

**目标：让用户看懂 quest 在流水线中的位置和产物有什么**

### 4.1 Board 视图：阶段视图

#### 4.1.1 看板双模式

工具栏增加切换：**按状态（默认）/ 按阶段

- **按状态**：保持现状，10 列，面向执行监控
- **按阶段**：按 pipeline_def 分列，面向流程理解

#### 4.1.2 阶段列生成规则

阶段列**不硬编码，动态生成：

1. 从当前所有 quest 的 `pipeline_def` 中收集所有出现的 phase 定义
2. 按 `phase_idx` 排序
3. 每列显示 `display_name`（或 `name`）+ 数量徽章
4. 列的 role（warrior/mage）用颜色区分

对于不同 pipeline 的 quest 混在同一块板上时：
- 优先显示最常见的 pipeline 作为主列
- 不常见的 pipeline 的 phase 如果与主 pipeline 对应不上时，合并展示
- 或按 pipeline 分组折叠

#### 4.1.3 Fallback 策略

**重要**：不是所有 quest 都有完整 pipeline 数据。

- 有 `pipeline_version` 且 > 0 且有 `phases` 数组非空 → 显示阶段视图
- 没有 pipeline 数据的 quest → 在阶段视图中归到"状态视图"的对应状态列，或显示"旧数据"标记
- UI 不假设所有 quest 都有完整 pipeline

### 4.2 卡片增强：阶段进度

#### 4.2.1 阶段指示器

卡片顶部 / 行显示阶段进度：
- 格式：`Phase ${current_phase_idx + 1}/${phase_count} · ${phase_name}`
- 或用进度条形式：显示已完成 phase 比例
- 显示在当前 phase 的停留时长（从 phases[].started_at_ms 计算）

**字段来源**：`current_phase_idx`、`phase_count`、`phases[].name` / `display_name`、`phases[].started_at_ms`

#### 4.2.2 返工标记

如果 `rework_count > 0`，显示返工标记：
- 图标 + `${rework_count} 次返工
- 悬停显示返工历史（从哪一 phase 返回到哪一 phase）

### 4.3 产物 Chips 增强

P0 的产物计数 chips 增强：
- 按 kind 分类显示：`file` / document` / `image` 等
- 点击产物 chip 打开**安全摘要弹窗
  - 产物列表（名称 + 大小 + 来源
  - 纯文本列表，不渲染内容
  - "查看详情" 按钮跳详情页

**安全边界**：
- 不直接渲染 artifact 内容（避免 XSS 风险）
- 只显示元数据（名称、大小、类型、来源）
- 完整预览进详情页

### 4.4 设计任务的视觉区分

设计类 quest（type=design）：
- 使用 mage 语义色（紫色系，与现有 mage 色一致）
- 细左边线（类似 attention bar 样式，紫色）
- 顶部 wand / badge "设计" badge
- 不用渐变边框，不用花哨装饰
- 阶段名称用 pipeline_def 里的实际 phase name，不硬编码"战士设计 → 法师设计评审"

---

## 5. P2：HOTL 控制面（定义策略视图模型后）

**目标：让用户能管理信任层级、调整策略、做批量操作**

### 5.1 前提：后端契约补齐

P2 全部依赖后端新增 quest 级 policy view model。需要后端定义并实现：

```typescript
// Quest 级策略视图模型（需后端新增
effective_trust_tier: 'tier_0' | 'tier_1' | 'tier_2' | 'tier_3'
// 注意：来源可能是从 automation 快照 + 当前策略决策事实
safety_floor_reason: string | null  // 安全底线触发原因，null 表示未触发
last_policy_decision: {
  id: string
  policy_name: string
  decision: 'pass' | 'escalate' | 'block' | 'retry'
  reason: string
  at_ms: number
} | null
recovery_state_summary: {
  strategy: 'retry' | 'degrade' | 'escalate' | 'skip_phase'
  attempt: number
  max_attempts: number
  next_attempt_at_ms: number | null
  status: 'active' | 'exhausted' | 'waiting'
} | null
```

**注意**：这些不是前端不能直接从现有 QuestMeta 推导。需要后端聚合后。

### 5.2 信任层级显示

- 所有 quest 卡片/行显示信任层级徽章：
  - t0：手动（灰色 / 已暂停
  - t1：确认运行（琥珀色）
  - t2：自动 + 审（蓝色/品牌色）
  - t3：全自动（绿色/成功色）

- 徽章点击可调整层级（弹窗确认）：
  - 调高："确认调高到 t2"
  - 调低："确认调低到 t1"
  - 每次调整有风险解释文案
  - 调整后有审计事件

**安全要求**：
- 必须有后端 API 支持
- 必须有确认对话框
- 必须有审计事件
- 必须有失败处理

### 5.3 安全底线提示

对于被安全底线拦截的 quest：
- 卡片上显示 shield 图标 + "安全底线" 标记
- hover 显示具体原因：`workspace_diff` / `external_side_effect` / `allow_l2`
- 说明"不受信任层级影响，始终需要人工审核

### 5.4 恢复策略可视化

对于 blocked 且有 active recovery 的 quest：
- 显示恢复策略类型 + 进度：重试 / 已重试 N/M 次
- 下次重试时间倒计时
- "立即重试"按钮（需确认）
- "跳过此阶段"按钮（需确认 + 风险提示）

### 5.5 单项快速操作

P2 才做，P0/P1 都不做。

支持的快速操作（卡片 hover 显示）：

| 操作 | 状态 | 安全等级 | 确认要求 |
|------|------|----------|----------|
| 查看 diff | user_review | 低（只读） | 否 |
| 立即重试 | blocked | 中 | 是 |
| 跳过阶段 | blocked | 高 | 是 + 风险解释 |
| 停止 | running | 中 | 是 |
| 开始 | pending | 低 | 否 |
| 取消 | pending | 中 | 是 |
| 调整信任层级 | 所有 | 高 | 是 + 影响说明 |

**所有操作都走后端对应 API，有成功失败都有 toast 反馈。

### 5.6 批量操作

P2 后期才做，且在 List 视图中做。

- 进入"批量模式后每行显示复选框
- 支持的批量操作（取决于选中项状态）：
  - 批量通过审核（user_review 状态）
  - 批量重试（blocked 状态）
  - 批量取消（pending / running）
  - 批量调整信任层级

**安全要求**：
- 必须有明确的数量确认（选中 N 项，其中 M 项可执行成功，K 项失败
- 必须有审计事件
- 必须有部分失败的处理（部分成功部分失败怎么展示）
- 必须有"仅对相同状态的操作

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

- **Attention / 琥珀色：`--attention`（user_review、waiting_input、t1）
- **Danger / 红色：`--danger`（blocked、apply_failed）
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

### 8.1 无需埋点的本地指标（直接从数据计算）

| 指标 | 计算方式 | 意义 |
|------|----------|------|
| Attention 队列平均等待时长 | user_review / blocked / waiting_input 从进入 attention 到处理的平均时长 | 反映人工介入的响应速度 |
| Waiting input 平均等待时长 | 同上，仅 waiting_input | Agent 等人类回答的速度 |
| Blocked 恢复率 | blocked → running 的比例 / 自动恢复比例 | 反映恢复策略效果 |
| 策略自动通过率 | auto_passed_by_policy 非空 / 已完成 quest 比例 | 反映 HOTL 自动化程度 |
| 各阶段平均停留时长 | phases[].ended_at - phases[].started_at 平均 | 反映流水线瓶颈 |

### 8.2 可选 UI Telemetry（可选，本地事件）

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
| **P0** | 可解释首页 | 需关注分区、停止原因透明化、策略痕迹、产物计数、状态 pill 可点击 | 无（只用已有字段，前端类型补齐即可 | 无新增后端改动 |
| **P1** | 阶段与产物可视化 | 阶段视图、阶段进度、产物 chips + 摘要、设计任务视觉区分 | pipeline_def / phases 前端类型化 + 确认字段稳定 |
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
| 阶段进度条 | ✅ PhaseDef / ReworkTo | | |
| 返工可视化 | ✅ ReworkTo | | |
| 信任层级显示与调整 | | ✅ Trust Tiers | |
| 安全底线提示 | | ✅ Global Safety Floor | |
| 恢复策略可视化 | | ✅ RecoveryPolicy | |
| 等待输入详情 | | ✅ waiting_input 状态 | |
| 设计任务视觉区分 | | | ✅ 核心 |
| 快速操作 | ✅ QuestService API | ✅ Review+Apply | |
| 批量操作 | ✅ 批量 API | ✅ 批量审核 | |

## 附录 B：字段使用字段清单

### P0 已存在的字段

| 字段 | 前端类型状态 | 用途 |
|------|-------------|------|
| `failure_attribution` | ✅ 已有类型 | blocked 原因结构化展示 |
| `blocked_reason` | ✅ 已有 | blocked 原因文本 |
| `waiting_input` 对象 | ⚠️ 未类型化 | waiting input 问题文本、提问者 |
| `effect_type` | ✅ 已有 | effect 类型徽章 |
| `auto_passed_by_policy` | ✅ 已有 | 策略自动通过标记 |
| `policy_decision_id` | ✅ 已有 | 策略决策 ID（link |
| `finalized_by` | ⚠️ 未类型化 | 结案方式 |
| `apply_error` | ✅ 已有 | apply 失败原因 |
| `outputs` | ✅ 已有类型 | 产物计数 |
| `design_summary` | ✅ 已有 | 设计摘要 |
| `resume_count` | ⚠️ 未类型化 | 恢复重试次数 |
| `workspace_diff_pending` | ✅ 已有 | diff 待审核标记 |

### P1 需要类型化字段

| 字段 | 状态 | 用途 |
|------|------|------|
| `pipeline_version` | ⚠️ 未类型化 | 阶段视图开关判断 |
| `current_phase_idx` | ⚠️ 未类型化 | 当前阶段索引 |
| `phase_count` | ⚠️ 未类型化 | 总阶段数 |
| `pipeline_name` | ⚠️ 未类型化 | pipeline 名称 |
| `pipeline_def` | ⚠️ 未类型化 | 阶段定义（列生成） |
| `phases` | ⚠️ 未类型化 | 阶段运行时状态 |

### P2 需要后端新增

| 字段 | 状态 | 用途 |
|------|------|------|
| `effective_trust_tier | ❌ 需新增 | 信任层级显示与调整 |
| `safety_floor_reason` | ❌ 需新增 | 安全底线触发原因 |
| `last_policy_decision` | ❌ 需新增 | 最近策略决策 |
| `recovery_state_summary` | ❌ 需新增 | 恢复状态摘要 |
