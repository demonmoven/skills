# gloop 可选设计阶段 Spec — Design Phase

> 版本：v0.1
> 状态：已实现
> 相关文档：`refactor-spec-v1.md`（架构重构主 spec）、`hotl-spec-v0.1.md`（HOTL 升级 spec）
> 定位：专项 spec，聚焦设计-执行两阶段拆分，是管道化架构的扩展

---

## 实现状态（2026-06-22）

本 spec 已按保守路径落地：

- D.-1 PhaseDef 数据驱动：`buildPhaseConfig` 已从 `PhaseDef` 读取 `Name / Role / ReadOnly / EndSignal / AllowedTools`，默认两阶段和历史 session key 保持兼容。
- D.0 可选四阶段 pipeline：支持 `with_design_phase`，生成 `warrior_design → mage_design_review → warrior_execute → mage_implementation_review` 四阶段管道；默认 quest 仍走两阶段。
- D.1 设计交付物：`PhaseReviewArtifact` 增加 `Kind / PhaseName / SessionID`；`design_plan` 作为 phase artifact kind，可通过 `FindPhaseArtifactByKind` 查找，并注入执行阶段 `<design_plan>`。
- D.2 实现评审对齐：`mage_implementation_review` 会注入 `design_plan` 和 `design_alignment_review` 机制块，要求法师检查实现与方案一致性。
- D.3 触发条件：已支持 CLI `--with-design-phase`、HTTP/API `with_design_phase`、automation `with_design_phase`。quest type / intensity 仍只作为推荐方向，不默认自动开启。
- D.4 设计到执行闭环（v0.1.2 增量）：支持 `auto_spawn_execute`，design quest 成功后可自动创建并启动 execute quest；父 quest 持久化 `child_execute_quest_id` / `spawn_error`，并通过 `parent_quest_id` 兜底保证幂等。quick design 第一版不自动 spawn，避免 quick 跳过评审边界混淆。
- D.5 前端入口（v0.1.2 增量）：创建委托 / 创建 automation / automation 编辑页已从 `execute/design` 字段升级为“直接执行 / 先设计再执行 / 只出方案”流程选择，并暴露“方案通过后自动执行”开关。

刻意未做：
- 不做自动复杂度判断。
- 不做多方案对抗。
- 不引入 per-phase rework_count，v0.1 继续使用全局 rework_count。
- 不直接复用 design quest 的 workspace_path 作为 child 工作区；child 会基于父级 base working dir 和 workspace mode 重新准备隔离工作区。
- spawn 失败只记录 `spawn_error` + `quest.note` + 页面提示，暂未接外部通知。

关键提交：`a569a34`, `eaee3f6`, `d42e5f9`, `3b57ca9`, `bc40290`。

---

## 一、背景与动机

### 1.1 现状

当前 gloop 的剑士阶段（warrior phase）是"设计 + 实现"混在一起的：

```
warrior phase（设计 + 实现 同阶段）
    ↓
mage review（只看最终交付）
```

剑士在同一个 phase 里：
1. 理解需求
2. 自己做技术方案
3. 按自己的方案实现
4. 提交交付物

法师只看最终结果，不提前评审方案方向。

### 1.2 问题

对于复杂 quest，这种模式有几个问题：

1. **返工成本高** — 方案方向错了，要等全部实现完了法师才发现。改方案意味着推倒重来，代价大。
2. **方案不可追溯** — 设计思路在 agent 的上下文里，没有独立的方案交付物。出了问题不知道是设计错了还是实现错了。
3. **评审聚焦不清** — 法师既要审方案又要审实现，两个维度混在一起，容易漏。
4. **不利于 Loop Engineering** — 设计循环和执行循环是两种不同的认知模式，混在一起每个循环的出口条件都不清晰。

简单 quest 没问题（方案简单，做错了返工成本也低），但复杂 quest 效率瓶颈很明显。

### 1.3 目标

在管道化架构的基础上，引入**可选的设计阶段（Design Phase）**：

- 复杂 quest 可以在执行前插入一个独立的设计评审阶段
- 剑士先出方案，法师审方案，方案过了再进执行
- 提前纠偏，降低返工成本
- 方案有独立交付物，可追溯、可对比

---

## 二、目标与非目标

### 2.1 目标

1. **降低复杂 quest 的返工成本** — 方案问题在设计阶段暴露，不用等实现完才改
2. **方案交付物独立化** — 设计有单独的 artifact，可追溯、可评审、可对比
3. **管道化原生支持** — 作为可选 phase 插入，不改变默认两阶段管道
4. **评审分层** — 方案评审和实现评审分开，各有各的标准和重点

### 2.2 非目标

- ❌ 不改变默认管道 — 简单 quest 仍然是"执行 → 评审"两阶段，设计阶段是可选的
- ❌ 不引入新的 agent 角色 — 设计阶段还是剑士做、法师审，角色分工不变，只是阶段拆细了
- ❌ 不做多方案并行/对抗式设计 — v0.1 只做单方案设计评审，多方案是远期探索
- ❌ 不改变 HOTL 的人机边界 — user_review 仍然在最终交付后，设计阶段的评审也是法师审，不是人审
- ❌ 不做"产品经理角色" — 需求澄清仍然由剑士通过 quest_ask 完成，不引入专门的需求分析阶段

---

## 三、设计原则

### 3.1 可选，不强制

- 设计阶段是**可选的 phase**，默认不开
- 简单 quest 不开，复杂 quest 按需开
- 开启方式：quest 创建时指定 `--with-design-phase`，或 quest type / automation 配置自动开启

### 3.2 管道化原生，但有前置依赖

- 方向上：设计阶段就是管道里多一个 phase，和现有 phase 平级，复用同一套执行引擎（runPhase）、状态机、存储结构
- **但有前置依赖：需要先把 PhaseDef 从静态硬编码改成数据驱动**
  - 当前 `buildPhaseConfig` 还是 `switch phaseIdx { 0 → execute, 1 → review }` 的硬编码
  - design phase 需要的是 `warrior_design / mage_design_review / warrior_execute / mage_implementation_review` 四种 phase 角色
  - 必须让 `PhaseDef.Role`（execute / review）+ `PhaseDef.Name` + `PhaseDef.EndSignal` 来驱动配置构建，不能再靠 phaseIdx 猜
- 不搞特殊架构，不加新概念，但要承认"加 phase 不是改个配置那么简单"，有一个 PhaseDef 数据驱动的前置工作

### 3.3 设计交付物是一等公民

- 设计输出有独立的 artifact，不是上下文里的"想法"
- 有明确的格式和评审标准
- 可以独立查看、对比、审计

### 3.4 评审分层，各有侧重

- **设计评审**：看方向对不对、方案行不行、边界有没有考虑到
- **实现评审**：看代码质量、有没有 bug、和方案是否一致
- 两次评审的标准不一样，prompt 不一样，关注点不一样

---

## 四、总体架构

### 4.1 管道结构

不开设计阶段（默认）：
```
phase 0: warrior（设计+实现） → phase 1: mage_review → user_review
```

开设计阶段：
```
phase 0: warrior_design（出方案）
    ↓ 设计评审
phase 1: mage_design_review（审方案）
    ↓ 方案通过
phase 2: warrior_execute（按方案实现）
    ↓ 实现评审
phase 3: mage_implementation_review（审代码）
    ↓
user_review
```

本质是把原来的一个 warrior phase 拆成两个（design + execute），一个 mage review 拆成两个（design review + implementation review）。

### 4.2 在现有架构中的位置

```
┌─────────────────────────────────────────────────────────┐
│                    Quest Pipeline                       │
│                                                         │
│  [Design Phase]  [Execute Phase]  [Review Phases]       │
│      (可选)           (必有)           (必有)            │
│                                                         │
│  warrior_design → mage_design_review →                  │
│                           ↓ pass                        │
│                      warrior_execute →                  │
│                           ↓                             │
│                      mage_review → user_review          │
│                                                         │
└───────────────────────┬─────────────────────────────────┘
                        │
                Policy Engine（HOTL）
                        │
                  Event Bus
```

### 4.3 与两角色的关系

- **剑士**：有两个形态 — design 剑士（出方案）和 execute 剑士（按方案实现）
- **法师**：有两个形态 — design 评审法师（审方案）和 implementation 评审法师（审实现）
- 都是同一套执行引擎的不同配置，不是新角色
- 同一个 adventurer 可以配置不同 phase 的不同 prompt / 工具权限 / 预算

### 4.4 前置依赖：PhaseDef 数据驱动

设计阶段不是"加个配置就能跑"的纯配置改动。它依赖一个前置基础设施：**PhaseDef 驱动的 phase config 构建**。

**现状问题**：
- `buildPhaseConfig` 靠 `switch phaseIdx` 硬编码：0 → execute, 1 → review, default → review
- phase 角色、名称、结束信号、prompt 内容都和索引绑定，不是数据驱动
- 加新 phase 需要改代码，不是改配置

**现状已有的结构**（不能再造一套平行的）：
- `domain/quest.PhaseDef`：phase 定义（模板层），已有 Index / Name / DisplayName / Role / Class / ReadOnly / EndSignal / ExitCriteria 等字段
- `fsstore.PhaseTask`：phase 运行时状态（实例层），有 Status / Turns / ReworkCount 等
- `QuestMeta.PipelineDef []PhaseTask`：管道定义（名字叫 def，但类型是 PhaseTask，语义混用）
- `QuestMeta.Phases []PhaseTask`：运行时 phase 实例

**D.-1 要做的事：统一语义，补齐字段，让 buildPhaseConfig 真的从 PhaseDef 读**

1. **语义对齐**：`domain/quest.PhaseDef` 是管道定义的真相源；`fsstore.PipelineDef` 从 `[]PhaseTask` 改为从 `PhaseDef` 生成实例，不要再定义一套字段。
2. **字段补齐**：`PhaseDef` 增加 / 明确以下字段（如已有则直接复用）：
   - `Name`：阶段标识（如 `warrior_design` / `mage_design_review`）
   - `Role`：`PhaseRoleExecute` / `PhaseRoleReview`
   - `ReadOnly`：是否只读阶段
   - `EndSignal`：结束阶段的信号工具名（`phase_checkpoint` / `review_quest`）
   - `AllowedTools`：允许的工具列表（可空，跟随 adventurer）
   - `DefaultHints`：默认 hint 次数
3. **buildPhaseConfig 改造**：从 `switch phaseIdx` 改为接收 `*quest.PhaseDef`，根据 PhaseDef 的字段构建配置。Name / Role / ReadOnly / EndSignal 全部从 PhaseDef 读，不再硬编码。
4. **不新增平行字段**：QuestMeta 不新增 `PhaseDefs` 字段，统一用 `PipelineDef`（或 domain 层的 `Pipeline`）承载管道定义。

```go
// buildPhaseConfig 的目标签名：
func (e *Engine) buildPhaseConfig(
    q *fsstore.QuestMeta,
    adv *fsstore.AdventurerFile,
    phaseDef *quest.PhaseDef,   // 从 PhaseDef 读配置，不再靠 phaseIdx 推断
    input phaseInput,
) *phaseConfig
```

**为什么这是前置**：
- 没有它，四阶段跑起来会"名不正、prompt 不对、结束信号不匹配"
- 这是管道化架构的收尾工作，也是 Design Phase 的地基
- 做完这件事，不仅 Design Phase 能接，未来加 QA phase、安全审计 phase 也都顺理成章

> 注意：这个前置工作属于主 spec Phase 2 的补完，不是 Design Phase 特有。
> 但 Design Phase 是第一个真正用到"多 phase 角色"的场景，所以在这里明确列出。
> 实现时**绝对不能**再新增一套平行的 PhaseDef 字段，必须在现有结构上统一语义。

---

## 五、详细设计

### 5.1 触发条件

设计阶段什么时候开启：

| 触发方式 | v0.1 状态 | 说明 | 示例 |
|---------|-----------|------|------|
| 显式指定 | ✅ 支持 | 创建 quest 时手动开 | `gloop quest create --with-design-phase` |
| Automation 配置 | ✅ 支持 | 某个 automation 的 quest 自动开 | `automation.config.with_design_phase = true` |
| Quest Type 推荐 | 🟡 仅提示 | 某些 quest type 推荐开，但不默认自动开 | `quest_type=system_design` 时 CLI 提示"建议开启设计阶段" |
| Intensity 推荐 | 🟡 仅提示 | 高复杂度推荐开，但不默认自动开 | `intensity=deep` 时 CLI 提示"建议开启设计阶段" |
| 智能判断（远期） | ❌ 不做 | 根据 quest 描述自动判断复杂度 | 描述里有"重构""架构"等关键词自动开 |

**v0.1 原则：保守触发只有两种自动开启**：
1. 显式 `--with-design-phase` 标志
2. automation 配置中 `with_design_phase = true`

Quest Type 和 Intensity **只做推荐提示**，不默认自动开启。原因：
- `intensity=deep` 不等于"需要设计阶段"，可能只是预算高但任务简单
- 自动开启会拖慢简单但高预算的任务
- 先积累数据后再考虑是否自动开

> 设计阶段默认不开，按需开启。v0.1 先验证价值，再决定是否扩大自动触发范围。

### 5.2 设计交付物（Design Artifact）

设计交付物是**平台级一等公民**，不是随便的文本输出。但它不引入新的 artifact 存储体系，而是**现有 phase artifact 的 kind 扩展**。

**和现有 artifact 系统的关系**：

| 层级 | 现有结构 | 设计阶段扩展 |
|------|---------|-------------|
| 存储层 | `PhaseReviewArtifact`（Assistant + evidence + outputs） | 加 `Kind` 字段，值为 `design_plan` |
| 存储路径 | `artifacts/<sid>.json`（phase artifact 文件） | 复用同一路径，不新增加密型 |
| 读取接口 | `PhaseArtifactForReview(qid, sid)` | 增加按 kind 查找的辅助函数：`FindPhaseArtifactByKind(qid, kind)` |
| 元数据层 | `QuestArtifact`（quest inputs/outputs） | 不扩展，design_plan 是 phase artifact，不是 quest-level output |

> 选型说明：v0.1 **不新建第三套 artifact**。直接在 `PhaseReviewArtifact` 上加 `Kind` 字段，
> 因为设计方案本质上就是 warrior_design phase 的 phase artifact，只是有特殊语义（design_plan），
> 执行阶段需要按 kind 查找并注入，而不是按 session_id 找"上一阶段的输出"。

**Artifact 契约**：

| 字段 | 值 | 说明 |
|------|-----|------|
| Kind | `design_plan` | phase artifact 的类型标记，存在 `PhaseReviewArtifact.Kind` 字段 |
| PhaseName | `warrior_design` | 产生该 artifact 的 phase 名称（来自 PhaseDef.Name） |
| Storage Key | `artifacts/<session_id>.json` | 复用 phase artifact 存储路径，session_id 由 PhaseDef + rework_count 生成 |
| Injection Entry | 执行阶段 prompt 的 `<design_plan>` 块 | 执行剑士上下文注入的固定位置 |
| Review Target | mage_design_review 的主评审对象 | 设计法师的核心评审输入 |
| 读取方式 | `FindPhaseArtifactByKind(qid, "design_plan")` | 按 kind 查找最近一轮的 design_plan artifact |

**谁写、谁读**：
- 写入：warrior_design phase 结束时，`phase_checkpoint` 提交，平台保存 phase artifact 时带上 `Kind: "design_plan"`
- 读取 1：mage_design_review phase 启动时，按 kind 查找 design_plan，作为主评审对象
- 读取 2：warrior_execute phase 启动时，按 kind 查找 design_plan，注入到 `<design_plan>` block
- 读取 3：mage_implementation_review phase 启动时，按 kind 查找 design_plan，用于"方案一致性"检查

**v0.1 交付物格式**（Markdown 结构化，包裹在 artifact 的 content 字段中）：

```markdown
# 方案设计

## 1. 问题理解
- 需求核心是什么
- 约束条件有哪些

## 2. 方案选型
- 方案 A：...（优缺点）
- 方案 B：...（优缺点）
- 最终选择：方案 A，原因是...

## 3. 详细设计
- 核心模块划分
- 关键接口定义
- 数据结构设计

## 4. 风险与边界
- 已知风险
- 不做的范围（out of scope）

## 5. 实施计划
- 分几步做
- 每步的预期产出
```

> 这是默认模板，可以按 quest type / adventurer 配置自定义。

**平台级识别的意义**：
- 设计 artifact 不只是"上一阶段的输出"，平台知道它是方案，能做特殊处理
- 执行阶段注入时，有固定的 `<design_plan>` 注入位置，不是混在 previous_output 里
- 实现评审时，平台知道要拿 design artifact 和实现 artifact 做一致性对比
- 后续扩展（设计质量评分、方案库沉淀）都基于 kind 识别

**设计交付物的定位**：
- 是执行阶段的"施工图"，执行阶段按图施工
- 是设计评审的核心依据，法师按这个审
- 是实现评审的对照物，实现评审要检查"有没有按方案做"
- 是 quest 级别的可沉淀资产，不是一次性中间产物

### 5.3 设计评审

设计评审法师和实现评审法师是两套不同的配置：

| 维度 | 设计评审 | 实现评审 |
|------|---------|---------|
| 评审对象 | 设计方案（design_plan artifact） | 代码 + 最终交付 + 方案一致性 |
| 关注点 | 方向、可行性、边界 | 质量、正确性、一致性 |
| 工具权限 | 只读（Read/Glob/Grep） | 只读（同现有法师） |
| 评审标准 | 方案对不对、全不全 | 代码好不好、对不对、和方案一致不一致 |
| Verdict | pass / request_changes / reject | pass / request_changes / reject |
| Pass 后去向 | 推进到下一个 execute phase | 进 user_review（或 HOTL policy） |

**设计评审的通过标准**（v0.1 建议）：
1. 问题理解是否准确
2. 方案是否可行（技术上能落地）
3. 边界是否清晰（知道什么不做）
4. 风险是否识别到了
5. 和现有架构是否冲突

> 注意：设计评审不追求"完美方案"，追求"方向没错 + 边界清晰 + 能落地"。
> 完美主义会导致设计阶段反复迭代，反而拖慢进度。

**评审流向的核心规则**：
- **设计评审 pass → 推进到 execute phase，**不进 user_review**。设计评审是中间环节，不是终审。**
- **实现评审 pass → 才是最后一个 review phase，进 user_review（或走 HOTL policy 决定是否需要人工）。**
- 任何 review phase reject → 直接进 user_review（reject 是终局性的，由人拍板）。
- 判断"pass 后往哪走"的依据是 `CurrentPhaseIdx` + `PhaseDef` 列表，不是状态本身。

### 5.4 状态机变化

设计阶段不引入新状态类型，复用现有状态模型。

> **关键澄清**：状态（running / reviewing）只是"正在执行 / 正在评审"的大类，**具体是哪个阶段由 `CurrentPhaseIdx + PhaseDef` 决定**，不能靠状态本身推断。
> 多阶段场景下，状态推断两阶段语义是错的。所有 phase 推进逻辑都必须读 `CurrentPhaseIdx`，再查 `PhaseDef` 拿到角色/名称/结束信号。

**状态流转**：
```
pending
  ↓ start（phase_idx = 0）
running（phase 0: design）
  ↓ phase_checkpoint 提交方案
reviewing（phase 1: design_review）
  ↓ review_quest: pass → phase_idx++
running（phase 2: execute）
  ↓ phase_checkpoint 提交实现
reviewing（phase 3: implementation_review）
  ↓ review_quest: pass → 最后一个 review phase
user_review
  ↓ user pass
success
```

如果设计评审 request_changes：
```
reviewing（design review）
  ↓ request_changes → phase_idx -= 1
running（design phase，返工）
  ↓ 重新提交方案
reviewing（design review）
  ... 循环直到 pass 或 reject
```

> 核心状态契约：
> - **设计评审 pass → 推进到下一个 execute phase，不进 user_review**
> - **实现评审 pass → 才是最后一个 review phase，进 user_review（或走 HOTL policy）**
> - 判断"是不是最后一个 review phase"的依据是 `CurrentPhaseIdx` 和 `PhaseDef` 列表，不是状态本身

### 5.5 执行阶段如何使用设计方案

设计交付物作为执行阶段的输入，通过**固定注入入口**进入剑士上下文，而不是混在普通 previous_output 里。

```go
// 执行阶段的 build user message 时，按 kind 查找设计 artifact
designArtifact := findPhaseArtifactByKind(q, "design_plan")
pack := prompt.BuildExecuteContextPackWith(q, designArtifact, ...)

// 注入位置：执行 prompt 的 <design_plan> 块，结构化、可定位
// <design_plan>
//   # 方案设计
//   ...
// </design_plan>
```

执行剑士的 prompt 里明确：
- "你有一份设计方案（<design_plan> 块），请按照方案实现"
- "如果发现方案有问题，可以提出修改意见，但不要偷偷改方案"
- "大的方案变更需要走设计返工（重新进 design phase），不能自行拍板"

> 注入入口是契约的一部分：执行阶段知道设计方案在哪个 block 里，
> 平台知道如何检查方案有没有被注入，后续自动化工具也能定位。

### 5.6 设计 vs 实现的责任边界

这是设计阶段最容易模糊的地方，需要明确：

| 责任 | 谁负责 |
|------|--------|
| 方案正确 | 设计剑士 + 设计法师 |
| 实现正确 | 执行剑士 + 实现法师 |
| 方案和实现一致 | 执行剑士 + 实现法师 |
| 方案变更 | 设计阶段返工，不是执行阶段自己改 |

**实现评审的额外检查项**：实现是否和设计方案一致。如果不一致，实现法师可以 request_changes，要求执行剑士要么按方案改，要么走设计返工流程。

### 5.7 返工与迭代

设计阶段和执行阶段都可以独立返工：

1. **设计返工**：设计评审 request_changes → 回到设计阶段改方案 → 再审
2. **执行返工**：实现评审 request_changes → 回到执行阶段改代码 → 再审
3. **执行阶段发现方案问题**：执行剑士可以提交 block / 提交设计变更建议 → 由法师和用户决定是改方案还是继续按原方案

v0.1 不做"执行阶段自动回退到设计阶段"的自动化路径。执行阶段发现方案问题，走 blocked → 用户 triage → 回退到设计 phase 的手动路径。自动化回退放远期。

> **v0.1 返工模型的明确妥协**：理想模型是每个 phase 独立的 rework_count，设计返工不影响执行阶段的轮次。v0.1 复用全局 rework_count 是**兼容性妥协**，不是理想模型。见 5.8 详细说明。

### 5.8 存储结构与返工计数

复用现有 phase artifact 存储，不新增存储类型：

- 设计阶段 artifact：`quests/<qid>/artifacts/warrior_design_0.json`
- 设计评审 artifact：`quests/<qid>/artifacts/mage_design_review_0.json`
- 执行阶段 artifact：`quests/<qid>/artifacts/warrior_execute_0.json`
- 实现评审 artifact：`quests/<qid>/artifacts/mage_review_0.json`

#### v0.1 简化方案：全局 rework_count

**v0.1 用全局 rework_count**，每个 phase 的 artifact 名里带上同一个 rework_count。

```
第一轮：
  warrior_design_0 → mage_design_review_0 → warrior_execute_0 → mage_review_0

如果设计返工 1 次：
  warrior_design_1 → mage_design_review_1 → warrior_execute_1 → mage_review_1
```

即：设计返工一次，后面所有阶段的轮次都 +1。这样和现有 rework_count 机制兼容，不用改存储结构。

> **兼容性妥协，不是理想模型**：
> 
> - 理想模型：每个 phase 有独立的 rework_count，设计返工只增加 design phase 的轮次，不影响 execute phase。执行阶段已经跑过的 artifact 可以保留并复用。
> - v0.1 模型：全局一个 rework_count，设计返工导致执行阶段也要"被返工"（轮次 +1 但其实还没开始）。
> - 为什么妥协：现有存储、状态、prompt 渲染都基于全局 rework_count，改成分阶段的改动面比较大。v0.1 先验证设计阶段本身的价值，返工计数优化放在 v0.2。
> - 代价：设计返工 N 次，artifact 数量是 4×N，比理想模型多。但 v0.1 阶段返工次数不会多，代价可接受。
> - 迁移路径：v0.2 引入 per-phase rework_count 时，artifact 存储路径需要升级，增加迁移逻辑。

---

## 六、实施路线

### Phase 0：可选设计阶段（预计 1-2 周）

依赖：主 spec Phase 2（宏循环管道化）完成 + **PhaseDef 数据驱动改造**完成（本 spec 的前置依赖，详见 4.4 节）

**实施原则**：先做最小可用版本，验证"提前设计评审"的价值，再迭代。

#### Milestone D.-1：PhaseDef 数据驱动（前置依赖，2-3 天）

这是设计阶段的基础设施，不直接产出设计阶段能力，但设计阶段必须建立在它之上。

**核心原则：统一现有语义，不新增平行的 PhaseDef 字段。**

具体工作：
- 对齐 `domain/quest.PhaseDef` 和 `fsstore.PhaseTask` / `PipelineDef` 的语义，明确 def 层（配置模板）和 instance 层（运行时状态）的分工
- `domain/quest.PhaseDef` 补齐 / 明确关键字段：Name / Role / ReadOnly / EndSignal / AllowedTools / DefaultHints
- `buildPhaseConfig` 从硬编码 `switch phaseIdx` 改为接收 `*quest.PhaseDef`，按 PhaseDef 字段构建配置
- `phaseRoleForIdx` / `phaseNameForIdx` / `phaseDisplayNameForIdx` 改为从 PhaseDef 列表查表
- QuestMeta 不新增 `PhaseDefs` 字段，复用 `PipelineDef`（或 domain 层 `Pipeline`）承载管道定义
- 默认两阶段行为完全不变，只是内部从硬编码改为数据驱动
- 单元测试覆盖：默认 2 阶段、自定义 4 阶段、phase 推进与回退

> 这条是硬前置。D.0 必须建立在 D.-1 完成的基础上，否则四阶段一跑就会撞在 `buildPhaseConfig` 的硬编码上。

#### Milestone D.0：管道扩展（1-2 天）

在 PhaseDef 基础设施上，扩展支持 4 阶段 pipeline：

- Quest 创建时根据 `with_design_phase` 配置决定使用 2 阶段还是 4 阶段
- 4 阶段 PhaseDef 列表：warrior_design → mage_design_review → warrior_execute → mage_implementation_review
- 宏循环 phase 推进逻辑：不依赖固定的 phase 0/1，依赖 PhaseDef.Role 判断下一个 phase 是 execute 还是 review
- "最后一个 review phase 进 user_review" 的判断基于 PhaseDef 列表位置，不是状态推断
- 不改变默认两阶段行为

#### Milestone D.1：设计阶段落地（2-3 天）

- warrior_design phase 配置（工具权限、prompt、结束信号 phase_checkpoint）
- 设计交付物模板（结构化 Markdown，phase artifact 带 `Kind: "design_plan"`）
- PhaseReviewArtifact 增加 `Kind` 字段，支持按 kind 查找 phase artifact
- mage_design_review phase 配置（评审标准 prompt，结束信号 review_quest）
- 设计方案注入到执行阶段上下文（`<design_plan>` 固定注入入口，按 kind 查找）
- 设计返工支持（和现有全局 rework_count 机制对齐）
- 设计评审 pass → 推进到 execute phase，不进 user_review

#### Milestone D.2：实现评审对齐（1-2 天）

- 实现评审增加"方案一致性"检查项
- mage_review 的 prompt 区分"有设计方案"和"无设计方案"两种模式
- 有设计方案时，实现评审要对照方案检查
- 实现评审阶段也能按 kind 找到 design_plan artifact

#### Milestone D.3：触发条件（1 天）

- `--with-design-phase` CLI 参数（显式开启）
- automation 配置支持 `with_design_phase = true`
- quest type / intensity 触发只做推荐提示，不默认自动开（CLI 输出建议，不自动启用）

### 可前置的小改动（Phase 1 就能做）

以下改动不依赖管道化，可以提前验证：
1. **剑士 prompt 增加"先想方案再动手"的引导** — 不需要新 phase，纯 prompt 优化，看看效果
2. **法师评审 prompt 增加"方案方向"的评审维度** — 在现有评审里拆分方案和实现两个评分项

> 注意：这些是低成本实验，不是 Design Phase 的替代品。
> 真正的设计阶段需要独立 artifact + 独立评审 + 返工闭环。

---

## 七、风险与应对

### 7.1 设计阶段变成"甩锅层"

**风险**：方案和实现都做对了还好，出了问题就互相甩锅 — "方案没说清楚" vs "实现没看懂"。

**应对**：
- 设计交付物的标准要明确、结构化，减少模糊地带
- 实现评审有"方案一致性"检查项，不一致就打回
- 设计返工有明确的触发条件，不是执行做不好就赖方案

### 7.2 设计评审磨洋工，方案永远通不过

**风险**：设计阶段反复迭代，"完美主义"导致进度停滞。

**应对**：
- 设计评审的通过标准要明确：方向对、边界清、能落地，不追求完美
- 设计阶段有独立的预算和 rework 次数限制
- 超过限制自动提交 user_review，让人来拍板

### 7.3 简单 quest 开了设计阶段反而慢

**风险**：本来一小时能做完的小 quest，加了设计评审反而花了两小时。

**应对**：
- 设计阶段默认不开，按需开启
- 提供清晰的开启指南：什么类型、什么复杂度的 quest 建议开
- 可以做 A/B 实验，对比有/无设计阶段的总耗时

### 7.4 方案和实现两张皮

**风险**：设计写了一套，执行时另一套，设计交付物形同虚设。

**应对**：
- 执行阶段 prompt 明确要求按方案实现
- 实现评审检查方案一致性
- 大的方案变更必须走设计返工，不能执行阶段偷偷改
- 自动检测：可以对比设计里提到的文件/接口和实际改动有没有偏差（远期）

---

## 八、后续探索

### 8.1 近期可实验

1. **设计阶段自动触发** — 根据 quest 描述的复杂度智能判断要不要开设计阶段
2. **执行阶段自动回退设计** — 执行时发现方案有重大问题，自动触发设计返工
3. **设计质量评分** — 对设计交付物做结构化评分，用于积累设计质量数据

### 8.2 远期探索

4. **多方案对抗式设计** — 并行跑两个设计剑士出不同方案，法师对比评审，选优进入执行
5. **设计模式库** — 积累常见问题的参考方案，设计阶段可以引用
6. **架构一致性检查** — 自动检查设计方案和项目整体架构是否一致
7. **设计评审 + 自动验证** — 设计评审前自动跑一些轻量检查（如接口是否存在、数据结构是否兼容）

---

## 九、与其他 spec 的映射关系

### 9.1 与主 spec（refactor-spec-v1.md）

| 主 spec 阶段 | Design Phase 改动 | 依赖关系 |
|------------|-------------------|---------|
| Phase 0.5：Pipeline 持久化 | 无 | - |
| Phase 1：上下文传递修复 | 设计方案注入执行上下文 | 可前置做 prompt 优化实验 |
| Phase 2：宏循环管道化 | 全部 Design Phase 特性的基础 | Design Phase 依赖管道化（动态 phase 数量） |
| Phase 3：per-quest 自动化 | automation 配置可以指定是否开设计阶段 | 联动 |
| Phase 4：跨 quest workflow | 未来探索（设计方案作为 workflow 节点输出） | 远期 |

### 9.2 与 HOTL spec（hotl-spec-v0.1.md）

| HOTL 特性 | Design Phase 交互 | 说明 |
|-----------|-------------------|------|
| Review Policy | 两个评审点都可以配置 policy | 设计评审和实现评审独立配置 auto_pass 策略 |
| Recovery Policy | 设计阶段和执行阶段各自的异常恢复 | 独立恢复，不跨阶段 |
| Trust Tiers | automation 的信任等级可以影响是否开设计阶段 | 高信任 automation 可以跳过设计阶段 |
| Notification Policy | 设计评审通过 / 打回也是通知事件 | 复用现有事件体系 |

> 核心原则：Design Phase 和 HOTL 是正交的两个方向。
> 一个是"管道里多几个 phase"，一个是"phase 之间的闸门谁来守"。
> 两者可以独立演进，也可以组合使用。

---

## 十、验证指标

上线后要回答的几个问题：

1. **返工次数减少了吗？** — 有设计阶段 vs 没有设计阶段的 quest，平均 rework_count 对比
2. **总耗时减少了吗？** — 复杂 quest 的端到端耗时对比
3. **最终质量提升了吗？** — user_review 的 pass 率、用户满意度
4. **设计评审的命中率** — 设计阶段发现的问题中，有多少是"如果没发现会导致执行返工"的真问题

先有数据，再决定要不要推广、要不要深化。
