# Spec：Gloop Mage Review — 剑士输出的抗偏差层

> **版本**：v0.2.1（2026-06-23，基于 v0.2 的第二轮 review 结论修订）
> **v0.2 → v0.2.1 修订摘要**：
> - `LatestMageReview` 匹配规则升级为兼容函数：`ReviewedBy === quest.mage_id` 优先；`ReviewedBy === "mage"` 作为 legacy 兼容并显式标注；不再漏判历史/旁路数据
> - 色板 × verdict × score 查表顺序全局唯一，fixtures 全部重新对齐（无自相矛盾）
> - Phase 1 测试策略收敛为纯函数单测（3 个 select/state/tone 函数 TS 断言），不引入测试运行器；React 部分靠 build + 手工 fixtures 页面
> - `SourcePhase`（uint）升级为 `ReviewSource` 结构（`source_role / source_class / source_phase_idx / source_adventurer_id / source_session_id`），Phase 1 就解耦「身份」和「phase 位置」
> - `StructuredReview` 校验失败 → 丢弃结构化、保留基础 review 字段 + 写入 invalid event + UI 降级提示，绝不阻断 verdict/comment 主信号
> - ExecutionTrace 跳转从 `quest.phases` 里匹配 mage 的 role/class/adventurer_id 取 phase_idx/session_id，fallback 到 1；不再 hardcode `phase=1`
> **核心边界（继承 v0.2，不变）**：
> - 严格遵守 **PRDv2 § 核心原则「系统做机制，agent 做决策」**。前端 / 平台 **绝不做任何语义猜测**，展示的每一个结构化信号都必须来自「agent 明确提交的 typed 字段」或「平台写入的 typed 字段」。
> - 前端 keyword/正则抽取 disagreements/risks → 不做。
> - 把「生成可信信号」和「展示可信信号」严格拆成两条线。Phase 1 只做后者。
> - **定位通用 Agent Loop 编排平台**，不窄化到 coding-agent。diff / coding 是 target.kind 的一个实例，不是边界。
> **根目标**：把法师从"另一个说话的人"变成「系统内置的第二只眼」，将用户对 Quest 的决策成本从「读 2 段长文自己拼答案」降到「扫 3 个数字 + 看 review 原文」。

---

## 0. 可信信号来源白名单（最高优先级原则，先读这条）

本 spec 里任何「前端展示的结构化信号」，只能来自以下白名单来源。**不在白名单里的，一律展示原文，绝不语义合成。**

### 0.1 Phase 1 已存在的可信 typed 字段（零后端改动）

| 字段 | 来源 | 定位 / 备注 |
|---|---|---|
| `quest.mage_id` | `internal/fsstore/quests.go` `QuestMeta.MageID` | 本 Quest 绑定的法师 adventurer id（可信）。可能为空字符串 |
| `quest.warrior_id` | `QuestMeta.WarriorID` | 剑士 adventurer id |
| `quest.mage_score` | `QuestMeta.MageScore` | 法师评分（1-10，0=未评分）。**Phase 1 只作展示 + 色板查表输入，不做加权判断的其他输入** |
| `quest.intensity` | `QuestMeta.Intensity` | quick/standard/deep/adversarial。`quick` = 跳法师的可信信号 |
| `quest.phases[]` | `QuestMeta.Phases` / `[]PhaseTask`（见 `internal/fsstore/quests.go:28-45`） | 每个 phase 含 `phase_idx / role / class / adventurer_id / session_id / status / turns`。**Phase 1 用作 mage 身份 + phase 位置的真实溯源，不再 hardcode phase=1** |
| `quest.final_verdict` / `final_comment` / `finalized_by` / `auto_passed_by_policy` / `auto_completed_by_policy` | `QuestMeta.Final*` | **最终决策字段**（finalized_by ∈ {user, policy, automation}）。**绝不能用来推导 Mage verdict**。quick 模式下 policy 自动完成会写这些字段，写了也和法师无关 |
| `reviews[]`（`GET /api/quests/:id` body 的 `reviews` key，`LoadReviews` 返回） | `internal/fsstore/quests.go` `ReviewRecord`：`Ts / Verdict / Comment / RewriteHints / ReviewedBy / Score / RepairCount / TotalIssueCount` | 多轮返工累积的评审记录。注意：**`ReviewedBy` 当前有两种真实存在的写法**（§0.1.1 详述） |
| `checks[]`（`GET /api/quests/:id` body 的 `checks` key） | `QuestStore.LoadChecks` 读取的 JSONL。前端类型 `Record<string, unknown>` | 仅作为 `checks` 面板数据。**在 contract 未扩展 `source` 字段之前，前端绝不能做「法师检查 / 平台检查」的归属判断**。只允许做纯展示优化（分栏、排序、折叠） |
| `quest.rework_count` / `max_rework` | `QuestMeta` | 返工进度，可信 |

#### 0.1.1 Phase 1：`selectLatestMageReview(reviews[], quest)` 兼容函数（修复 v0.2 的漏判）

当前代码库里 `reviews[].ReviewedBy` 存在两种合法写法：
- 宏循环（`internal/orchestrator/macro_loop.go:392-398`）写 `ReviewedBy: mage.ID` → adventurer id 字符串
- domain service（`internal/domain/quest/service.go:493-500`）写 `ReviewedBy: "mage"` → 字面量 `"mage"`

**Phase 1 的识别规则必须同时兼容两者，并在 UI 上区分来源（审计 + 后续收敛）：**

```
函数: selectLatestMageReview(reviews, quest) -> LatestMageReviewMatch | null

  type LatestMageReviewMatch = {
    record: ReviewRecord
    source_kind: 'exact_adventurer' | 'legacy_role_literal'
  }

  1. 过滤 reviews：保留 ReviewedBy ∈ { quest.mage_id, "mage" } 的记录；
     如果 quest.mage_id 非空且 ReviewedBy === quest.mage_id → exact
     如果 quest.mage_id 非空且 ReviewedBy === "mage"   → legacy（domain service 历史写法）
     quest.mage_id 空 → 不匹配任何记录（因为没有 adventurer 绑定，legacy "mage" 语义不可信）
  2. 在剩余记录里取 max(Ts) = latest
  3. 返回 { record: latest, source_kind } ；空 → null
```

**UI 上的呈现差异**：`source_kind === 'legacy_role_literal'` 时，法师名字旁加小字 tag「来源：legacy 写法」。Phase 1.5 引入 ReviewSource 结构之后，这一类会被正确标注，tag 消失。Phase 1 保证"不展示成蓝条（法师参与中）而误判"——即 **宁可显示 legacy 也不能把它当成"法师没评审"**。

### 0.2 Phase 1.5 新增的 typed contract（微小后端改动 + agent 协议调整）

本阶段新增的字段**只做加字段**，不破坏 Phase 1。

#### 0.2.1 `ReviewRecord.ReviewSource`（替代 v0.2 的 SourcePhase uint）

把「身份」和「phase 位置」解耦（修复 v0.2 把 phase 当身份用的错）：

```go
type ReviewSource struct {
    SourceRole         string `json:"source_role"`           // warrior / mage / user / policy / automation
    SourceClass        string `json:"source_class"`          // 冒险者职业：warrior/mage（对应 model.AdventurerClass）
    SourcePhaseIdx     int    `json:"source_phase_idx"`      // 阶段索引（仅定位，非身份；未来多 phase pipeline 不影响）
    SourceAdventurerID string `json:"source_adventurer_id"`  // 冒险者 ID。非 agent 提交（user/policy/automation）可空
    SourceSessionID    string `json:"source_session_id"`    // 该次评审对应的 session id，前端跳转 ExecutionTrace 直接用
}
```

平台在 `AppendReview` 时（宏循环 + domain service 两条路径都要）按已知上下文填完整；Phase 1.5 起前端判断 Mage Review 用：

```
source.Role === "mage" || source.Class === "mage" || source.AdventurerID === quest.mage_id
```

展示人名用 `source_adventurer_id`；跳转 ExecutionTrace 用 `source_phase_idx + source_session_id`。`phase_idx` 只是定位，不再参与身份判断。

#### 0.2.2 `ReviewRecord.StructuredReview`（Mage agent / CLI review 协议写入，可选）

typed 结构，见 §4.1。**有此字段，Phase 2 才开启 disagreements / cross-panel 信号。**

#### 0.2.3 `ReviewRecord.ReviewArtifactID`（可选）

如果法师把结构化评审写成 artifact 交付，这里挂 artifact id。

#### 0.2.4 `QuestCheck.Source`（enum：`platform` / `warrior` / `mage`）

`LoadChecks` 的每条 check 在写入时由 writer 带。**有此字段，checks 面板才允许分「平台 / 法师 / 剑士」做 UI 区分**；没有就统一展示。

### 0.3 Phase 2+ 的 typed contract 扩展

`StructuredReview.Evidence[]` / `TargetKind` / `Confidence` 等。见 §4 / §7。

> **铁律**：任何一个 UI 组件，只要要使用某字段做"判断"而不是"原样展示"，这个字段的类型就必须先写进 §0 的白名单。没进白名单的，展示原文 + 标注「来源：法师自由文本」，不做高亮/置顶/过滤/加权。

---

## 1. 背景 / 问题定义（为什么做）

### 1.1 当前实现的问题

Gloop 的剑士（warrior）+ 法师（mage）双代理架构，本质是「**视角冗余度 = 2**」——同一个 Quest 有两个独立信号源，这是产品核心竞争力之一。但当前前端把两个信号源当"并列展示的两个人"处理：

| 问题 | 量化影响（估算） |
|---|---|
| 法师在结论优先级里排第 4（final_comment → 剑士 → summary → **法师**），多数 Quest 里法师**根本不说话** | ≈60% 的 Quest 用户读不到法师输出 |
| 法师消息和剑士消息并列，用户必须**自己 diff 两段长文找分歧** | 用户心智成本 ≥ 读 1.5× 全文 |
| 法师和剑士内容高度重叠（法师基于剑士输出写评论），前端不做视觉层级区分 | 用户被迫重读 ≈70% 的法师文本 |
| `mage_score` /10 躺在侧边栏元数据里，不参与任何 UI 决策 | 法师最有价值的量化信号完全浪费 |
| quick 模式下"快速模式 · 无评审"对用户是一句空话，**没有行为引导** | 用户在没有抗偏差层时，决策风险未知 |
| final_comment / mage review / user review 三者来源混淆 | 用户不知道他读的终审意见是谁写的 |
| `ReviewedBy` 存在两种写法 → 法师 review 偶发被漏判成「法师参与中 · 暂未提交」 | 感知 bug，信任度下降 |

### 1.2 第一性原理根目标

> **Phase 1 的可工程化目标**：用户首屏 ≤ 1 秒能回答下面 3 个问题，而不需要往下滚 / 切换 Tab：
> 1. 这单有法师吗？（quick / 没配 / 法师已评审 / 有法师但没评审 — 四种状态一眼区分；并且对 legacy `ReviewedBy === "mage"` 写法不会误判为「没评审」）
> 2. 法师的 verdict / score 是什么？（不进入执行轨迹 / 不搜索消息）
> 3. 法师原文评论在哪里点？（一次点击直达折叠原文 + 跳转 ExecutionTrace 对应 phase/session）
>
> **Phase 1.5+ 的体验目标**（非工程阻塞项，放到 rollout 验证）：消费法师评审的心智成本 ≤ 30 秒，法师信息信噪比 ≥ 60%。

体验指标（3 秒心智成本 / 80% 信噪比 / 10ms 渲染 / 15KB gzip）全部移到 rollout 验证，不阻塞 Phase 1 工程验收。

---

## 2. 产品定位 / 核心原则

### 2.1 法师在 Gloop 里到底是什么

法师是**针对「完整 Quest」的独立评审信号源**（Anti-bias Layer）。
- "完整 Quest" 的含义：它评审的对象是剑士 phase 结束时对 Quest 目标的声称达成 + 交付证据链的全域一致性。**代码 diff 是 target.kind 的一个实例（artifact / message / check / workspace_effect / policy_target 都是合法的 target kind），不是产品边界。**
- 通用含义：同样的 Mage Review 机制在代码任务里显示 diff，在文档任务里显示文本修改，在运营任务里显示工单变更 — UI 不依赖 target kind，只依赖 typed contract。

### 2.2 法师的行为约束（从 §0 边界导出）

- **不是结论来源**。主要结论永远来自剑士（或最终用户终审）。法师绝不抢话语权。
- **只做两件事**：
  1. 提交 typed verdict / score / comment（Phase 1 已有）
  2. 通过 StructuredReview 协议提交 typed evidence（Phase 1.5+）
- **同意 = 不浪费 UI 空间**。如果 Mage review 只有 comment 且和剑士结论重叠度高，UI 只展示 verdict/score，comment 默认折叠。不做语义去重替换。
- **渗透全页，但只在 typed evidence 存在时渗透**。没 evidence 时，只能做「侧边栏 → 评审带锚点跳转」这种纯机制联动，不做业务语义联动。
- **缺失时诚实降级**。抗偏差层 = 0 时，用户看到的不是"无法师"，而是「本单未经过独立评审，请谨慎决策」。

### 2.3 三个角色的固定分工（不让用户困惑的底线）

| 角色 | 产出形式 | 在决策中的权重 | 视觉层级 |
|---|---|---|---|
| ⚔ 剑士 | 自由文本 + 证据链（artifact / check / diff / ...）按协议提交 | 主信息源 · ~60% | 主卡 · 全尺寸 · 默认展开 |
| 🧙 法师 | typed ReviewRecord（verdict / score / comment）+ 可选 StructuredReview | 偏差校验 · ~25% | 附随带 · 半尺寸 · 默认折叠 · **粘在剑士卡下方**（从属层级固定） |
| 🧑 用户 / policy / automation | FinalVerdict + Comment（finalized_by ∈ {user, policy, automation}） | 最终责任人 · ~15% | 顶栏终态 + 底部操作区 · 主 CTA |

> **重要**：`quest.final_*` 永远是终态。在 Mage Review UI 里，它的展示是「最终结论卡」，和 Mage Review 是两条独立线，不互相推断。v0.1 把 final_verdict 当 Mage verdict 用是错的，已修复。

---

## 3. 用户故事（按 §0 白名单重构）

**US-1**（P0 · Phase 1）
> 作为 reviewer，打开任意 Quest 首屏不用滚，能一眼区分「quick / 没配法师 / 法师已评审（含 legacy ReviewedBy="mage" 写法）/ 法师已参与未评审」四种状态，并看到法师 verdict + score。

**US-2**（P0 · Phase 1）
> 作为 reviewer，当法师 verdict ≠ pass 时，剑士结论卡边框/横幅按 §5.1 全局唯一直方表警告，不会漏掉；但不会改变剑士结论内容、不会抢话语权。

**US-3**（P0 · Phase 1）
> 作为 reviewer，点击 MageReviewBand 能展开法师评论原文（Markdown 渲染）+ 一键跳到 ExecutionTrace 的法师对应 phase（**通过 quest.phases 匹配 mage 身份取 phase_idx/session_id，不是 hardcode 1**）。

**US-4**（P0 · Phase 1）
> 作为 quick 模式或未配置法师的用户，我看到「⚡ 快速模式 · 无独立评审 · 仅用于确定性任务」的明确降级提示 + 建议操作（升级到标准模式 / 去配法师），不是一句空话。

**US-5**（P0 · Phase 1.5）
> 当法师通过协议提交了 `ReviewRecord.StructuredReview.disagreements[]`（typed），前端展开 MageReviewBand 时，按 dimension + severity 的卡片形式展示每条异议，**每条有「来源：mage structured review v1」的 audit 标识**。不是猜的。
> 如果 structured_review 校验失败（超长、schema 不匹配等），整条 review 的 verdict/comment/score 仍正常保存；UI 显示 typed 部分「结构化评审无效，已降级为原文」；不阻断核心评审流程。

**US-6**（P1 · Phase 2）
> 当 typed `Evidence[].kind == 'diff_file' | 'artifact' | 'warrior_message' | 'check'` 存在时，ChangesPanel / OutputsPanel / ExecutionTrace / checks 面板**按 evidence 引用做高亮 / 置顶 / 默认展开**。没 evidence 的面板不做任何猜测性联动。

**US-7**（P2 · Phase 3）
> 有了 typed coverage / confidence / cross-panel evidence，再考虑段落级去重和"未读异议"的 CTA 提醒。这两件事依赖可信 typed signal，不提前做。

---

## 4. 数据契约（严格按 §0 白名单分层）

### 4.1 Phase 1.5：StructuredReview（写入方：Mage agent / CLI review 协议；消费方：前端 Phase 2）

只定义 contract，Phase 1 完全不用它（可以是 `null`，前端回退到纯 comment 展示）。

```typescript
// ==============================================================
// 关键：target kinds 是通用枚举，不含 code/coding 字眼
// ==============================================================
type EvidenceTargetKind =
  | 'warrior_message'     // 剑士 phase 的某条消息
  | 'artifact'            // 任意交付产物（文件 / 图片 / 文本 / 补丁）
  | 'check'               // 一个 QuestCheck
  | 'diff_file'           // workspace diff 中某个具体文件
  | 'workspace_effect'    // 变更面板的 effect_type（workspace/readonly/context_store/…）
  | 'policy_target'       // policy 决策相关的目标
  | 'quest'               // 整份 quest 级别的异议（scope、目标）
  | 'design_doc'          // 设计文档的某一节
  | 'other'

// ==============================================================
// Mage Review 的维度分类（通用，不绑定代码领域）
// ==============================================================
type MageDimension =
  | 'correctness'     // 正确性（事实 / 逻辑 / 协议一致性）
  | 'completeness'    // 完整性（是否覆盖全部目标）
  | 'boundary'        // 边界 / 异常路径 / 边缘情况
  | 'risk'            // 回归 / 性能 / 成本 / 副作用风险
  | 'validation'      // 验证 / 测试 / 证据链不足
  | 'coherence'       // 内部一致性 / 证据和结论矛盾
  | 'scope'           // scope 漂移 / 越权 / 未涉及承诺
  | 'style'           // 可维护性 / 规范性（不影响正确性）
  | 'other'

type MageSeverity = 'high' | 'med' | 'low' | 'info'

type EvidenceRef = {
  id: string                     // 稳定且唯一的 id。warrior_message=session id+turn；artifact=artifact.id；check=check.id；diff_file=slug(path)
  kind: EvidenceTargetKind
  // 可选的次级引用。例：
  //   kind=artifact 时 field='md' anchor='section-2.3'
  //   kind=diff_file 时 anchor='hunk-3'
  //   kind=warrior_message 时 field='paragraph[1]'
  anchor?: string
  field?: string
  snippet?: string               // ≤200 字的原文摘录（审计用途，UI 可选展开）
}

type MageDisagreement = {
  id: string
  dimension: MageDimension
  severity: MageSeverity
  target_kind: EvidenceTargetKind   // 这条异议指向哪种对象
  claim: string                     // ≤140字，主展示文案（mage 写的，非前端合成）
  evidence: EvidenceRef[]           // 数组，允许一个异议跨多 target
  suggestion?: string               // 可操作建议（≤140字，mage 写）
}

type MageExtraRisk = {
  id: string
  kind: 'safety' | 'regression' | 'perf' | 'security' | 'legal' | 'operational' | 'other'
  severity: MageSeverity
  detail: string
  evidence: EvidenceRef[]           // 风险对应的证据
}

type MageEndorsement = {
  id: string
  dimension: MageDimension
  reason: string                    // ≤80字。最多 3 条，UI 不允许超过这个数量显示（否则"同意"变噪音）
  evidence: EvidenceRef[]
}

type MageCheckReference = {    // 法师自己跑 / 触发的 check，引用 checks[] 的 id
  check_id: string
  expected?: 'pass' | 'fail' | 'warn' | 'any'
  comment?: string
}

type StructuredReview = {
  schema_version: 'v1'

  // 溯源 & 审计
  reviewed_warrior_phase_idx: number      // = 0（warrior phase）。未来多 phase 时扩展
  reviewed_warrior_session_id?: string
  reviewed_warrior_latest_turn?: number   // 审到剑士哪一轮，前端展示用
  coverage_ratio?: [number, number]       // [已覆盖论点, 总论点数]。允许 mage 提交。前端只有在非 nil 时才显示覆盖率
  confidence?: 'high' | 'med' | 'low'     // 仅 mage 自评。不影响平台决策，只显示

  // 核心三块：
  disagreements: MageDisagreement[]
  risks_extra: MageExtraRisk[]
  endorsements: MageEndorsement[]

  // 指向 checks[] 的引用（注意：不是 copy，是引用）
  referenced_checks?: MageCheckReference[]

  // 机器可读的原文指针（方便 UI 做「回到法师原文 comment 的这一段」）
  free_comment_range?: { start_char: number; end_char: number }
}
```

**Contract 设计 + 校验策略（修复 v0.2 的尾巴摇狗问题）**：

Review 提交分**两层校验**，绝对不允许结构化部分失败阻断主信号：

| 层级 | 校验内容 | 失败 → 行为 |
|---|---|---|
| 基础 review 字段 | `verdict ∈ {pass, request_changes, reject}`；`comment` 长度 ≤ 200KB；`ts` 合法；`reviewed_by` / `review_source` 合法 | **整条拒绝**，返回协议错误。这一层很少失败 |
| `StructuredReview`（可选字段） | `schema_version === 'v1'`；`disagreements.length ≤ 20`；`endorsements.length ≤ 3`；每个 `evidence.length ≤ 5`；`id` 去重；所有枚举合法 | **只丢弃 StructuredReview**（置 nil），基础字段照常写入。同时 append 一条 `review.structured_invalid` event（记录原 structured JSON + 错误原因，供 mage/agent 侧排查）。UI 上显示「结构化评审字段校验未通过，已降级为原文（审计：event_id）」小字 tag |

结果：**哪怕 mage 把 structured 部分写崩了，`request_changes` + comment 原文也一定能到达用户**。尾巴绝不能摇狗。

---

## 5. UI / UX 规格

### 5.1 色板查表（全局唯一，verdict × score → 颜色。修复 v0.2 的自相矛盾）

> **查表顺序从上到下**（先命中先返回，全局唯一）：
>
> | # | 条件（全是白名单字段） | 边框 / 标签色 | 背景 | 剑士卡横幅 |
> |:--|---|---|---|---|
> | 1 | 无 LatestMageReview **但** `mage_id` 非空（法师参与中） | `#0ea5e9` 蓝 | `rgba(14,165,233,0.06)` | 蓝色弱提示：「法师暂未提交评审」 |
> | 2 | 有 LatestMageReview，`verdict === reject` | `#7f1d1d` 深暗红 | `rgba(127,29,29,0.10)` | 红色强横幅：「法师否决，请复核」 |
> | 3 | 有 LatestMageReview，`verdict === request_changes` | `#dc2626` 红 | `rgba(239,68,68,0.08)` | 红色横幅：「法师要求返工」 |
> | 4 | 有 LatestMageReview，`verdict === pass` 且 `score === 0 或 score undefined/null`（未评分） | `#64748b` 灰绿 | `rgba(100,116,139,0.06)` | 无横幅（或极弱文字「法师通过 · 未评分」） |
> | 5 | 有 LatestMageReview，`verdict === pass` 且 `0 < score < 5` | `#dc2626` 红 | `rgba(239,68,68,0.08)` | 红色横幅：「法师通过但评分偏低，请复核」 |
> | 6 | 有 LatestMageReview，`verdict === pass` 且 `5 ≤ score < 8` | `#ca8a04` 黄 | `rgba(234,179,8,0.06)` | 黄色横幅：「法师通过 · 带条件」 |
> | 7 | 有 LatestMageReview，`verdict === pass` 且 `score ≥ 8` | `#16a34a` 绿 | `rgba(34,197,94,0.06)` | 无横幅（高可信） |
> | 8 | `intensity === quick` **或** 无 `mage_id`（降级态 · 抗偏差层缺失） | `#64748b` 灰 | `rgba(100,116,139,0.06)` | 灰色降级条：见 §5.6 |
>
> **注意**：`verdict === request_changes 时 score=6`（v0.2 F2 的旧假设）→ 查表命中第 3 条（红），不是黄。fixture 全部按此表重算，确保不矛盾。
> **注意**：`score=0`（未评分）不触发第 5 条（score<5），因为第 4 条在前面先命中了。fixture F4 走第 4 条，不是红。

### 5.2 组件新增 / 调整清单（按 Phase 分层）

#### Phase 1（零后端改动，只依赖 §0.1 白名单）

| 组件 | 状态 | 说明 |
|---|---|---|
| `selectLatestMageReview()` | **新增 · 纯函数**（纯 TS 单测） | 按 §0.1.1 的兼容函数实现；返回值结构含 `source_kind` 便于审计 tag |
| `mageReviewTone()` | **新增 · 纯函数**（纯 TS 单测） | 输入 `LatestMageReviewMatch | null + quest`，输出 §5.1 的色板对象（tone id + css class + 横幅文案）。纯查表，零副作用 |
| `mageReviewState()` | **新增 · 纯函数**（纯 TS 单测） | 输入 `LatestMageReviewMatch | null + quest + phases[]`，输出：显示态（`reviewed / reviewing / no_mage / no_review_quick`）+ legacy tag + phase_idx/session_id 用于跳转。零副作用 |
| `<MageReviewBand />` | **新增** | 粘在剑士结论卡下的折叠评审带。折叠态：法师 id + verdict 胶囊 + score 胶囊 + legacy tag（仅 source_kind=legacy 显示）+ 降级提示；展开态：review comment Markdown + rewrite_hints + repair_count/total_issue_count + 「跳到法师轨迹（通过 `mageReviewState().phase_idx/session_id`，不是 hardcode 1）」按钮 |
| `<MageScoreChip />` | **新增** | `7.8/10` + verdict 胶囊的小胶囊。复用于侧边栏 / 列表页（未来）/ Quest 摘要卡 |
| `AdventurerConclusionCard`（剑士结论卡） | **改** | 边框 + 横幅区域，按 `mageReviewTone()` 的返回值挂 CSS class（不改内容、不改排序、不改折叠态） |
| `ResourcePanel / 侧边栏` | **改** | 法师信息行 → `<MageScoreChip />` + 锚点跳 MageReviewBand；intensity=quick 时显式写"未执行评审" |
| `<FinalVerdictCard />`（**必做，非可选**） | **新增** | 把 `quest.final_*` 单独做成一张终态卡，剑士卡**之上**或顶栏。**明确标注 finalized_by**（user / policy / automation），和 Mage Review 完全分离。v0.2 混淆的根治 |
| `ExecutionTrace`（机制联动，**不改内容**） | **小改** | 暴露 `scrollToPhase(phase_idx, session_id?)` API（不再暴露 `scrollToMageTurn`）；Phase 1 就不 hardcode phase=1；滚动位置通过 `mageReviewState()` 计算 |

#### Phase 1.5（§0.2 新字段可用后开启）

| 组件 | 状态 | 说明 |
|---|---|---|
| `<MageDisagreementCard />` | **新增（条件渲染：StructuredReview 存在时）** | 卡片：dimension tag + severity 颜色 + claim + evidence 引用展开 + suggestion |
| `<MageRiskCard />` | **新增（条件渲染）** | 卡片：kind tag + severity 颜色 + detail + evidence |
| `<EndorsementStrip />` | **新增（条件渲染且 endorsements.length>0）** | 横向亮点列表（≤3 颗星），不占大空间 |
| `QuestChecksPanel`（checks 展示优化） | **改（条件：QuestCheck.source 存在）** | 分 Tab `平台检查` / `剑士检查` / `法师检查`（按 source 枚举直接映射，不 keyword 猜）。source 不存在 → 不分栏，保持现状 |
| StructuredReview 降级 tag | **新增** | 如果 review 存在 `structured_invalid` event（§4.1 校验降级），在 MageReviewBand 顶部显示小字 tag「结构化评审字段校验未通过 · 已降级为原文 · 查看原因」（链接到 event） |

#### Phase 2（§4.1 EvidenceRef 完整时开启）

| 组件 | 状态 | 说明 |
|---|---|---|
| `ChangesSection` | **改（条件 Evidence.kind=diff_file / workspace_effect）** | 按 target_kind = diff_file 的 evidence 做：`border-left: 3px solid {severity}`；≤2 条时默认展开；>2 条时仅高亮。**绝不排序，除非 evidence.ref 里显式带 priority 字段**（避免前端引入判断语义） |
| `QuestOutputsPanel` | **改（条件 kind=artifact）** | 引用的 artifact 卡片加 🔖 `法师提及` tag；≤2 条时默认展开 preview。不做排序修改 |
| `ExecutionTrace` | **改（条件 kind=warrior_message）** | 引用的消息左侧加 severity 色竖条 + 锚点 id，允许从 DisagreementCard 直接跳转。不做自动展开 |
| `ChecksPanel` | **改（条件 kind=check）** | 引用的 check 卡片加 🔖 tag；如果 Mage referenced_checks.expected='fail' 且实际 pass，额外显示「✓ 法师预期失败，检查已通过」 |
| audit 标签 | **全卡片** | 每张结构化卡片右下角 `来源：法师 structured_review v1` |

#### Phase 3（体验增强，非阻塞，按需排期）

| 项 | 说明 |
|---|---|
| paragraph 级折叠辅助 | 依赖 `free_comment_range` + evidence.snippet。默认展开，折叠是用户主动操作，不自动省略 |
| CTA 未读 High 异议弱提醒 | 依赖 disagreement.id seen（localStorage）。不弹窗、不禁用 |
| coverage 展示 | 依赖 coverage_ratio 非 nil |
| confidence 展示 | 依赖 confidence 非 nil（3 个点：高 3 亮 / 中 2 亮 / 低 1 亮） |
| rollout 体验指标 | 心智成本、信噪比、决策耗时对比 |

### 5.3 ExecutionTrace 跳转（Phase 1 就解耦 phase=1）

修复 v0.2 的 hardcode 问题。纯函数 `mageReviewState` 计算跳转目标：

```
magePhase := quest.phases.find(p =>
  p.role === "mage" ||
  String(p.class) === "mage" ||   // model.AdventurerClass
  (quest.mage_id !== "" && p.adventurer_id === quest.mage_id)
)
如果找到 → 返回 { phase_idx: magePhase.phase_idx, session_id: magePhase.session_id }
否则（理论上不会发生，防御性 fallback）→ 返回 { phase_idx: 1, session_id: undefined }
```

前端 ExecutionTrace 的 `scrollToPhase(phase_idx, session_id?)` 实现：
- 如果传 session_id → 按 session_id 滚（跨 pipeline 版本仍稳定）
- 否则按 phase_idx 滚
- 都找不到 → 滚到第一条 phase=1 的消息

### 5.4 MageReviewBand 折叠态（默认态 · Phase 1）

```
┌─ FinalVerdictCard（终态卡 · 顶部，仅 quest.final_verdict 存在时渲染）
│  Status: [pass · 策略自动完成]   Finalized by: policy (auto_complete_quick)
│  Comment: "剑士完成全部任务，quick 模式自动完成"
│
├─ AdventurerConclusionCard（剑士主卡；边框按 §5.1 查表，仅提示色）
│  TL;DR  …
│  行动项 …
│  后果聚合 …
│
├─ MageReviewBand（折叠）─────────────────────────────────────────────┤
│  🧙 法师 · 莉莉丝   [verdict: request_changes]   ⭐ 6 / 10         │
│  [12 条问题 · 已修复 3 · 还剩 9]   [来源: legacy_role_literal] ← 仅 legacy 显示
│                                                    [展开评论 ▾]     │
│  ── 降级态（no_mage / quick）替换 ──                                │
│  ⚡ 快速模式 · **无独立评审** · 仅用于确定性任务  [升级到标准模式 ▸]  │
└─────────────────────────────────────────────────────────────────────┘
```

- 高度 ≤ 3 行（不含展开内容）
- legacy tag：仅 `source_kind === 'legacy_role_literal'` 时显示，tooltip 写「本条 review 的 ReviewedBy 记录为字面量 "mage"，建议后端升级为 adventurer_id 写法」。Phase 1.5 起自动消失
- `[升级到标准模式]`：如 Quest 已完成 → tooltip "下次建议使用标准模式"，按钮 disabled

### 5.5 MageReviewBand 展开态

Phase 1（无 StructuredReview）：**只显示原文 comment + rewrite_hints + 修复进度**，绝不做任何语义抽取。
Phase 1.5（有 StructuredReview）：在原文上方渲染 typed 卡片区（§5.2 Phase 1.5），原文仍在下方保留完整可读。
StructuredReview 校验降级（§4.1）：在 typed 区位置显示红色降级 tag，不渲染卡片。

### 5.6 四种降级态（Phase 1 全量覆盖）

| 场景（全来自白名单字段） | 折叠态显示 | 额外行为 |
|---|---|---|
| `intensity === 'quick'` | 灰条：「⚡ 快速模式 · **无独立评审** · 仅用于确定性任务」 | tooltip 解释；条末尾 `[升级到标准模式 ▸]`（Quest 已完成 → tooltip + disabled） |
| `intensity !== 'quick' && !quest.mage_id` | 灰条：「⚠ 未配置法师 · **缺少抗偏差层**」 | 条末尾 `[去配置法师 ▸]` 直跳配置 |
| `intensity !== 'quick' && quest.mage_id` **但** 无 LatestMageReview（空 reviews 或匹配不到） | 蓝条 §5.1 #1：「⏳ 法师参与中 · 暂未提交评审」 | 条末尾「[查看法师执行轨迹 ▸]」→ 按 §5.3 跳转函数算 phase |
| 有 LatestMageReview，但 `comment === ''` 且 StructuredReview 为 nil | 正常色，但显示「法师提交了 verdict + score，未附评论」 + 隐藏展开按钮 | — |
| Quest 已用户终审（finalized_by=user/policy/automation） | MageReviewBand **仍然显示**，允许用户回顾；FinalVerdictCard 在剑士卡上方展示真值 | CTA 区无提示按钮 |

---

## 6. 全域去重（按边界收敛范围）

v0.2 已经删除了所有段落级去重替换；v0.2.1 保持 §6 不动：

- 保留：消息 ↔ artifact 的 normalize + fastHash 弱关联（chip，不改内容）
- 保留：结论来源 = summary artifact 时，产物卡片默认折叠 + tag（只改 open 属性，不改文本）
- 新增 Phase 1：剑士最终消息 vs final_comment / user review comment 的弱匹配 → chip「已在最终结论展示」
- 删除（移 Phase 3）：剑士 vs 法师段落替换、structured claims vs 原文高亮联动。Phase 3 只有在 `free_comment_range` typed 字段存在时，做可选折叠（默认展开，不自动省略）

---

## 7. 分阶段落地计划（按 contract 成熟度切）

### Phase 1（~1.5 天，**零后端改动**，本周开工）
- [ ] 3 个纯函数：`selectLatestMageReview / mageReviewTone / mageReviewState`（§5.2 Phase 1）
- [ ] 纯函数单测（§8.2）
- [ ] `<MageReviewBand />` 折叠 / 展开（verdict × score × repair/total × comment markdown 渲染 + legacy tag + §5.3 跳转）
- [ ] `<MageScoreChip />` + 侧边栏法师行升级
- [ ] `<FinalVerdictCard />`（必做）
- [ ] 剑士结论卡边框 + 横幅（仅 class 切换，不改内容）
- [ ] ExecutionTrace `scrollToPhase(idx, session?)` API
- [ ] 4 种降级态
- [ ] §6 的 final_comment chip 弱提示
- [ ] 手工 fixtures 页面（6 个 fixtures JSON）
- [ ] `npm run build` 0 tsc error

### Phase 1.5（~0.5 天后端 + ~1 天前端）
- [ ] 后端：`ReviewRecord.ReviewSource`（宏循环 + service 两条写入路径都补；AppendReview 时如果字段缺就按上下文尽量填）
- [ ] 后端：`ReviewRecord.StructuredReview`（schema v1）+ 分层校验（§4.1）+ `review.structured_invalid` event 写入
- [ ] 后端：`QuestCheck.Source`（writer 写入，缺省 nil → 前端不分栏）
- [ ] Mage agent / CLI review 协议：如果要提交 structured_review，按 §4.1 schema v1
- [ ] 前端：`<MageDisagreementCard />` / `<MageRiskCard />` / `<EndorsementStrip />` 条件渲染 + audit tag
- [ ] 前端：StructuredReview invalid 降级 tag 显示
- [ ] 前端：checks 分 Tab（仅 source 非空才分）

### Phase 2（~1 天前端。EvidenceRef 有真实数据时开启）
- [ ] ChangesPanel diff 文件高亮（kind=diff_file / workspace_effect）
- [ ] OutputsPanel artifact tag + 默认展开（kind=artifact）
- [ ] ExecutionTrace warrior_message 竖条 + 锚点（kind=warrior_message）
- [ ] ChecksPanel tag + expected 联动（kind=check）

### Phase 3（体验增强，按需）
- [ ] paragraph 级折叠辅助（free_comment_range）
- [ ] CTA 未读 High 异议弱提醒
- [ ] coverage / confidence 展示
- [ ] rollout 体验指标采集

---

## 8. 验收标准（v0.2.1 重写：对齐实际测试策略）

### 8.1 测试策略（修复 v0.2 引入测试运行器的问题）

当前 web 工程没有 `vitest` / `jest` / testing-library。v0.2 写的 snapshot 测试不现实。v0.2.1 改成：
- **Phase 1 阻塞项：3 个纯函数的 TS 单测**，用最简单的断言手写（不引入测试运行器）
- **React 组件**：`npm run build` 通过 + 手工 fixtures 页面 + code review DOM 属性断言
- **Phase 1.5+**：再评估是否引入 vitest，不在 Phase 1 的阻塞路径上

### 8.2 纯函数 TS 单测（Phase 1 阻塞项。写在 `web/src/__tests__/mage-review.ts`）

测试实现方式：用「TS 文件 + Node 原生 assert」的极简自运行脚本，`npm run test:mage-review` 调用 `tsx web/src/__tests__/mage-review.ts`（或 build 后用 node 跑）。不引入 vitest。

纯函数列表和断言集：

#### A. `selectLatestMageReview(reviews, quest)`

用例：
| 用例名 | reviews 构造 | quest 构造 | 预期 |
|---|---|---|---|
| A1 单条 exact 匹配 | `[{ts:100, reviewed_by: 'm-1', ...}]` | `mage_id: 'm-1'` | `{source_kind: 'exact_adventurer', record.ts=100}` |
| A2 单条 legacy 匹配 | `[{ts:100, reviewed_by: 'mage', ...}]` | `mage_id: 'm-1'` | `{source_kind: 'legacy_role_literal', record.ts=100}` |
| A3 exact 优先于 legacy | `[{ts:90, reviewed_by: 'm-1'}, {ts:200, reviewed_by: 'mage'}]` | `mage_id: 'm-1'` | 返回 `ts=200`（max Ts 优先；source_kind 按匹配到那条记录的判定——因为 legacy reviewed_by='mage' 在有 mage_id 的情况下判定为 legacy。本题 exact 和 legacy 各一条，max Ts=200 是 legacy → 返回 legacy） |
| A4 无 mage_id 时 legacy 不匹配 | `[{ts:100, reviewed_by: 'mage'}]` | `mage_id: ''` | `null`（没有 adventurer 绑定，legacy 语义不可信） |
| A5 无 mage_id 时 exact 也不匹配 | `[{ts:100, reviewed_by: 'm-1'}]` | `mage_id: ''` | `null` |
| A6 空 reviews | `[]` | `mage_id: 'm-1'` | `null` |
| A7 全部是 user review | `[{ts:100, reviewed_by: 'u-alice'}, {ts:200, reviewed_by: 'policy:auto-complete'}]` | `mage_id: 'm-1'` | `null` |
| A8 多轮返工 latest 优先 | `[ {ts:100, reviewed_by:'m-1', verdict:'request_changes'}, {ts:200, reviewed_by:'m-1', verdict:'pass'} ]` | `mage_id: 'm-1'` | 返回 verdict=pass 那条 |

#### B. `mageReviewTone(match, quest)`

用例（和 §5.1 查表一一对应，fixture F1-F6 同步）：
| 用例名 | match | quest | 预期 tone id | 预期颜色 |
|---|---|---|---|---|
| B1 reject → 深红 | `{verdict:reject, score:4}` | intensity=standard, mage_id='m-1' | `reject` | `#7f1d1d` |
| B2 request_changes (score=6) → 红 | `{verdict:request_changes, score:6}` | intensity=standard | `request_changes` | `#dc2626` |
| B3 pass + score=9 → 绿 | `{verdict:pass, score:9}` | intensity=standard | `pass_high` | `#16a34a` |
| B4 pass + score=6 → 黄 | `{verdict:pass, score:6}` | intensity=standard | `pass_mid` | `#ca8a04` |
| B5 pass + score=3 → 红 | `{verdict:pass, score:3}` | intensity=standard | `pass_low` | `#dc2626` |
| B6 pass + score=0 → 灰绿（未评分） | `{verdict:pass, score:0}` | intensity=standard | `pass_unscored` | `#64748b`（灰绿） |
| B7 null match + mage_id → 蓝（参与中） | `null` | intensity=standard, mage_id='m-1' | `reviewing` | `#0ea5e9` |
| B8 null match + quick → 灰 | `null` | intensity=quick | `no_review_quick` | `#64748b` |
| B9 null match + 无 mage_id → 灰（no_mage） | `null` | intensity=standard, mage_id='' | `no_mage` | `#64748b` |

#### C. `mageReviewState(match, quest)`

用例：
| 用例 | quest.phases | 预期跳转目标 phase_idx | 预期 display_state |
|---|---|---|---|
| C1 phases 里 role=mage phase_idx=2（多 phase 场景） | `[{idx:0,role:warrior}, {idx:1,role:auditor}, {idx:2,role:mage,session_id:s-42}]` | `phase_idx=2, session_id=s-42` | `reviewed` |
| C2 phases 里 adventurer_id 匹配（未来 adventurer_id 变化） | `[{idx:0,role:warrior,adv_id:w-1}, {idx:1,role:reviewer,adv_id:m-1,s_id:s-9}]` quest.mage_id='m-1' | `phase_idx=1, session_id=s-9` | `reviewed` |
| C3 phases 空数组 → fallback 到 1 | `[]` | `phase_idx=1, session_id=undefined` | 同上 |

**断言实现**：每个用例写成 `assert.deepStrictEqual(actual, expected, caseName)`。脚本最终打印 `PASSED: N / N`。非零退出码表示失败，可以挂到 `npm run build` 之后做 CI 验证（如果团队想），但 Phase 1 先不强制挂 build 链。

### 8.3 React 组件验收（Phase 1 阻塞项）

- [ ] `npm run build` tsc 0 error + vite build 通过
- [ ] 手工 fixtures 页面（`web/src/__fixtures__/mage-review/fixtures.tsx`）渲染 F1~F6 的 6 个 QuestDetail 样例（用 mock data），**人工对照 §5.1 色板**确认：

| # | Fixture 名 | 关键白名单字段 | §5.1 查表命中 # | 预期 UI 断言 |
|---|---|---|---|---|
| F1 | `pass_high.json` | mage_id='m-1'；review=[reviewed_by='m-1', verdict=pass, score=9]；final_verdict=pass, finalized_by=user | #7 | 剑士卡绿边框；MageReviewBand 绿 verdict + 9/10；FinalVerdictCard 显示「用户 · 通过」 |
| F2 | `request_changes_mid.json` | mage_id='m-1'；review=[reviewed_by='mage' ← **legacy 写法** , verdict=request_changes, score=6, repair=3, total=12] | #3 | **红边框**（不再是 v0.2 的黄）；红色横幅「法师要求返工」；MageReviewBand 显示 `legacy_role_literal` tag +「已修复 3 / 共 12」 |
| F3 | `reject_low.json` | mage_id='m-1'；review=[reviewed_by='m-1', verdict=reject, score=3] | #2 | 深暗红边框；verdict 胶囊深深红色 |
| F4 | `pass_unscored.json` | mage_id='m-1'；review=[reviewed_by='m-1', verdict=pass, score=0] | #4 | **灰绿边框**（不再是红，因为第 4 条比第 5 条先命中）；score 胶囊显示「未评分 / —」 |
| F5 | `no_mage.json` | mage_id=''; final_verdict=pass, finalized_by=policy, auto_completed_by_policy=xxx | #8 no_mage | 灰降级条「未配置法师」；FinalVerdictCard 「策略 · 通过 · xxx」；LatestMageReview = null（因为 mage_id 空） |
| F6 | `quick_no_review.json` | intensity=quick; mage_id=''; final_verdict=pass, finalized_by=policy, auto_completed_by_policy=xxx | #8 quick | 灰降级条「快速模式 · 无独立评审」；FinalVerdictCard 「策略 · 通过」 |

> **F2 和 F4 是本次 v0.2.1 最关键的修 bug fixture**——分别对应 reviewer 指出的「request_changes + score=6 误判成黄」和「score=0 误判成红」两个色板矛盾。

### 8.4 行为验收（Phase 1 阻塞项）
- [ ] FinalVerdictCard 的 finalized_by 文案严格区分 user/policy/automation
- [ ] F5 / F6（两个没法师的 case），无论 reviews 里有没有 policy 记录，LatestMageReview 都必须是 null
- [ ] F2（legacy ReviewedBy）不被误判成「法师参与中 · 暂未提交」——必须显示为已评审
- [ ] 点击「跳到法师轨迹」时：C1/C2 场景下跳到正确 phase_idx（不是 1）；C3 场景 fallback 到 1 且不崩
- [ ] 全域去重只在 §6 列出的 3 处生效，其他位置未改 textContent

### 8.5 非功能验收
- [ ] Phase 1 新增代码 gzip ≤ 10KB（不含 Shiki chunk）
- [ ] MageReviewBand 未展开时渲染 ≤ 5ms

---

## 9. 开放问题（收敛为 3 个可拍板项）

| # | 问题 | v0.2.1 默认值（不同意再改） | 待决定 |
|---|---|---|---|
| 1 | FinalVerdictCard 放剑士卡**上面**（作为整份 Quest 的当前状态头）还是**侧边栏**？ | 放剑士卡上面。因为 finalized_by=policy 的自动完成终态是用户第一眼要看到的事实 | 是 |
| 2 | MageReviewBand 放剑士卡**下面**（从属层级）还是**上面**（先看风险）？ | 放下面。保持「剑士=主结论来源」心智，降低学习成本 | 是 |
| 3 | Phase 1 的 `test:mage-review` 脚本要不要直接挂到 `npm run build` 前面？ | 先不挂。纯函数脚本 + 手工 fixture 页面跑通就行；Phase 1.5 起再挂 CI。避免给当前 build 流程加未知变量 | 是 |

---

## 10. 红线 — 设计不允许触碰的三条（防回归）

**红线 1：平台 / 前端绝不做任何语义质量判断。**
- disagreements / risks / endorsements / dimension / severity / evidence / check 归属 — 只要是"语义判断"，必须是 agent 明确通过 typed 协议提交。前端 keyword/正则抽取 = 越界，不允许。
- 唯一例外：PRDv2 明确允许的 3 种平台判断（确定性环境判断 / 协议完整性判断 / 安全策略判断）。这 3 种判断在代码里必须注释标注「PRDv2 §核心原则的例外：确定性 / 协议 / 安全」。

**红线 2：法师不能变成阻塞用户操作的门禁。**
- Phase 3 的 CTA 未读异议提醒只能是弱文字提醒，**绝不能禁用按钮 / 弹强制确认 / 要求用户点过所有卡片**。
- 任何 future 版本想加"法师 reject 时不能点通过" → 必须走 spec 修订，且过 PRD 合规 review。v0.2.1 不允许。

**红线 3：法师绝不抢剑士的话语权。**
- MageReviewBand 永远在剑士结论卡下方，视觉尺寸 / 字体更小
- 法师 verdict 即使 reject，剑士结论卡也**只改边框颜色 + 横幅**，绝不折叠 / 隐藏 / 替换剑士原文
- StructuredReview 有内容时，卡片区放在 Mage 原文之前，但原文永远在卡片区下方可展开读到完整评论

---

## 附：这样设计会影响 Gloop 的定位吗？（继承 v0.2 不修改）

> **结论不变**：**会，但方向是正的——它让 Gloop 从「一个能跑的通用 Agent Loop 编排器」变成「一个带可信审计机制的通用 Agent Loop 编排器」。定位上是升级，不是偏移。** v0.2 已经把 coding-agent 窄化问题修正到位，v0.2.1 不重复。

### 定位安全的三条底线（继承 v0.2 + 对照本次 reviewer 6 条修 bug）

v0.2.1 修了 6 条 bug，没有一条涉及定位改变——都是「让可信信号更可信、让 contract 更解耦」的工程性修订。三条红线守住了，定位就是正向升级；守不住，定位会从「平台」窄化成「某个领域 reviewer 工具」。

---

## 下一步建议

1. **30 分钟 tri-party review v0.2.1**（产品 + 后端 + 前端），只需要确认 4 件事：
   - §0.1.1 `selectLatestMageReview` 的 legacy 兼容策略 + legacy tag 做法
   - §5.1 的全局唯一直方表（特别是 F2=红、F4=灰绿）
   - §0.2.1 `ReviewSource` 结构（5 个字段）在宏循环 + service 两个写入路径都能正确填充，不会有上下文缺失
   - §4.1 StructuredReview 分层校验（结构化失败只降级、不阻断）写入 side effect 的实现位置
2. review 通过 → 我直接开 Phase 1 落地（1.5 天：3 纯函数 + TS 断言脚本 + 4 组件 + 6 fixtures 页面）
3. Phase 1 合并上线 → 排队 Phase 1.5 的后端 PR → Mage agent 侧 structured_review 有真实数据 → 开 Phase 2
