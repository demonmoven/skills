# Gloop 重构 Spec v1

> 状态：Implemented（v1 scope closed; Phase 4 workflow still deferred, v0.1.2 adds limited design→execute auto-spawn）
> 日期：2026-06-20
> 负责人：lihuanyu.0w0
> 关联讨论：飞书话题群 #gloop-design

---

## 1. 背景与动机

### 1.1 当前架构快照

Gloop 是一个**带制检分离（Maker-Checker Separation）的 Agent Loop Runtime**。

**核心机制（实现后）：**
- **双职业**：剑士（Warrior，生产者）+ 法师（Mage，评审者）
- **宏循环**：剑士执行 → 法师评审 → 返工或通过（状态机驱动）
- **微循环**：单阶段内 agent 回合制执行（消息 → 推理 → 工具调用 → 结果回灌）
- **ContextPack**：带信任分层的上下文 IR（platform / user / agent_output / mixed）
- **自动证据注入**：法师评审前平台自动运行 L0 验证命令，结果作为最高信任级证据
- **gloop CLI**：agent 通过 shell 命令与 gloop 机制交互（phase done / review / note add 等），通过信号文件异步传递
- **Platform Tools 系统**：内部 handler 复用层（CLI 底层调用它，测试也用它）
- **事件总线**：内存 fan-out，quest 状态变更发事件；同时持久化带单调 ID 的 events.jsonl
- **QuestService**：domain 层状态机，状态迁移字段统一走语义化 API
- **Pipeline**：PhaseDef 静态定义 + meta 持久化快照；默认两阶段与线性多评审阶段可恢复
- **Automation**：定时触发 + FanOut 多 quest 并行
- **Skill 系统**：agent-side instruction package，gloop 只分发不解释

**Agent ↔ Gloop 交互模型：**

```
gloop ──阶段级 context pack（每条阶段开始时发一次）──▶ agent
           （中途只做机制 guardrail，不干预策略）

agent ──shell 执行 gloop 命令──▶ 信号文件 ──gloop 轮询──▶ 消费信号
       （主路径，生产环境，异步）

agent ──native tool 调用──▶ 直接执行，gloop 只旁观记录（审计/证据用）
```

**domain / fsstore 双模型现状（v1 收口后）：**
- domain 层：`quest.QuestMeta` 保留 `Pipeline`/`CurrentPhaseIdx`/`Phases []PhaseRun` 运行态视图。
- fsstore 层：`QuestMeta` 持久化 `pipeline_version/current_phase_idx/pipeline_name/phase_count/pipeline_def_hash/pipeline_def/phases`。
- `QuestRepositoryAdapter` 双向转换 pipeline snapshot；自定义 pipeline 不再退回默认两阶段。
- 真实事实源仍是 fsstore meta，完整 domain aggregate 重构推迟到后续探索。

**v0.1.2 增量说明：**
- 已新增一个受限的 design→execute auto-spawn：仅在单个 design quest 成功后，根据 `auto_spawn_execute` 创建一个 execute child，并用 `child_execute_quest_id` + `parent_quest_id` 做幂等。
- 这不是完整 Phase 4 跨 quest workflow automation：没有 workflow instance、没有全局事件日志驱动的多节点编排，也没有 design→execute→test 级联状态机。
- auto-spawn 作为 post-success action 存在，失败不改变 design quest 终态，只记录 `spawn_error` / `quest.note` 并在详情页提示。

### 1.2 核心问题

经过多轮设计 review + 代码走查，识别出三个层面的问题：

**架构层（已处理）：**
1. **宏循环混杂 coding 特化逻辑** — CommitWorkspaceChanges / applyWorkspace / updateDiffSummary 等 coding 相关逻辑散落在宏循环主路径，与"通用编排层"定位矛盾
2. **Pipeline 化不彻底** — domain 层有 PhaseDef，但不持久化；ReworkTargetIdx() 硬编码返回 0；宏循环里还是 phase 0 / phase 1 硬编码 switch
3. **状态写入点不统一** — QuestService 设计上是单一入口，但 orchestrator 仍有直接写 fsstore 的情况；domain/fsstore 双模型，真实事实源不清
4. **Post-Processor 边界模糊** — auto-apply 这类会影响状态裁决的同步副作用，和通知、diff cache 这类 best-effort 后置动作混在一起
5. **Platform Tools 定位模糊** — 对外暴露的是 CLI，内部有 platformtools 包，名字和概念容易混淆

**上下文传递层（已处理）：**
1. **P0：结构化 deliverables 断流** — 剑士通过 `gloop phase done --deliverables` 提交了结构化产出物，存到了 `q.Outputs`，但构建法师 context pack 时完全没加载
2. **P1：返工只传 hints** — 法师的完整 comment / score / verdict 没传给返工剑士，导致挤牙膏式返工
3. **P2：platform_tool_evidence 噪音大** — 所有工具调用一股脑列出来，没有分类和结构，法师需要自己筛选有效证据
4. **P3：artifact 结构过薄** — PhaseReviewArtifact 只有两个字符串字段，完全没有结构化能力

**Loop Engineering 落地层（v1 已处理 / Phase 4 推迟）：**
1. **Automation 只有定时触发** — 缺少事件驱动型（quest 通过 / 失败 / 阶段结束）
2. **事件总线可靠性不足** — in-memory fan-out，慢订阅方丢事件；automation 直接订阅不可靠
3. **State 管理不够干净** — 虽然有状态机，但 meta.json 仍被多处直接读写，不完全是"唯一真相源"
4. **文件式上下文契约未闭环** — `gloop context get` 只是占位文案，没有真实 CLI 实现
5. **pipeline 不持久化** — 重启或恢复时依赖两阶段假设，多阶段管道无法正确恢复

### 1.3 设计约束

重构必须遵守以下硬约束：

1. **Gloop 是 dumb runtime** — 没有智能，所有判断由 agent 完成
2. **制检分离是特色** — 剑士/法师双阶段要保留并强化
3. **少干预 agent** — 能让 agent 自己发现的就不主动塞
4. **不强制任务类型** — 不给用户增加选择任务类型的负担
5. **架构简洁** — 不搞三层架构、不搞任务指纹等复杂概念
6. **gloop CLI 是唯一对外机制入口** — 平台工具是内部实现，不直接暴露给 agent
7. **向后兼容** — 现有 quest 数据、配置、CLI 接口尽量不破坏
8. **同步 vs 异步边界清晰** — 影响状态裁决的副作用走同步路径，best-effort 走异步事件

---

## 2. 目标与非目标

### 2.1 目标

- **T1**：宏循环完全去 coding 特化，成为通用阶段管道（基于现有 Pipeline 演进）
- **T2**：剑士/法师上下文传递结构化、证据化，返工信息完整
- **T3**：Loop Engineering 六要素在 gloop 中清晰落地（基于现有能力补齐）
- **T4**：状态管理完全走 QuestService，事件可订阅，自动化可事件触发
- **T5**：同步副作用（auto-apply 等）的状态语义不被破坏；异步订阅有可靠恢复机制

### 2.2 非目标

- ❌ 不做智能任务分类（agent 自行判断）
- ❌ 不内置评审方法论（只给协议，不给标准）
- ❌ 不内置执行方法论（agent 自行探索，skill 提供参考）
- ❌ 不搞 intake/sensors/actuators 三层架构
- ❌ 不搞任务指纹、意图识别
- ❌ 不搞"统一工具 ABI"（原生工具和平台机制就是两条线，不混）
- ❌ 不搞分布式事件队列（单机单用户，持久化 log + cursor 足够）

---

## 3. 设计原则

| 原则 | 含义 |
|------|------|
| **机制优于策略** | 只规定"怎么交差"，不规定"怎么干活" |
| **信任分层** | 所有上下文带 source/trust 标签，platform > user > agent_output |
| **最少干预** | 能让 agent 自己发现的，就不要主动塞 |
| **数据驱动阶段** | 阶段配置用数据描述，不用接口多态（已有 Pipeline 基础） |
| **状态可恢复** | 任何时刻断电，重启后能从断点续跑 |
| **可观测性优先** | 所有决策都留事件痕迹，事后可审计 |
| **渐进式重构** | 分阶段落地，每阶段都可独立交付和验证 |
| **CLI-first** | 对 agent 只暴露 gloop CLI 作为机制入口，平台工具是内部实现细节 |
| **同步裁决、异步派生** | 影响状态终态的副作用走同步路径；通知、派生任务走异步事件 |
| **小步迁移** | 优先做增量补充，不搞大爆炸式结构迁移；能兼容就不重写 |

---

## 4. 总体架构

### 4.1 重构后架构图

```
┌──────────────────────────────────────────────────────────────────────┐
│                           Gloop Runtime                               │
│                                                                      │
│  ┌──────────────┐       ┌───────────────────────────┐               │
│  │  Automation  │──────▶│     QuestService          │               │
│  │ (cron+事件)  │       │  Domain State Machine     │               │
│  │              │       │  （状态迁移唯一入口）        │               │
│  └──────────────┘       └────────────┬──────────────┘               │
│                                      │                                │
│  ┌──────────────┐                   │                                │
│  │  Event Bus   │◀──────────────────┘  （状态变更发事件）            │
│  │ in-memory    │──after-commit───────────────────┐                  │
│  │ fan-out      │                                 │                  │
│  └──────┬───────┘                                 ▼                  │
│         │                           ┌──────────────────────┐        │
│         │                           │ After-Commit         │        │
│         │                           │ Subscribers          │        │
│         │                           │  · 通知               │        │
│         │                           │  · diff cache         │        │
│         │                           │  · 派生任务(弱保证)    │        │
│         │                           │  · 统计               │        │
│         │                           └──────────────────────┘        │
│         │                                                           │
│         ▼                                                           │
│  ┌─────────────────┐          ┌───────────────────┐                 │
│  │ events.jsonl    │          │    Macro Loop     │                 │
│  │ 持久化事件日志   │          │  （通用阶段管道）  │                 │
│  └────────┬────────┘          └─────────┬─────────┘                 │
│           │                             │                           │
│           │                ┌────────────┴──────────────┐            │
│           │                ▼                           ▼            │
│           │         ┌───────────┐   Artifact    ┌───────────┐       │
│           │         │  Phase 0  │──────────────▶│  Phase 1  │       │
│           │         │  (剑士)    │               │  (法师)    │       │
│           │         └───────────┘               └─────┬─────┘       │
│           │               ▲                            │             │
│           │               │─────────rework─────────────┘             │
│           │                                                           │
│  ┌────────▼───────────────────────────────────────────────────┐      │
│  │              Artifact / State Store                        │      │
│  │  meta.json + sessions + work + signals + reviews.jsonl     │      │
│  └───────────────────────────────────────────────────────────┘      │
│                                                                      │
│  · Pre-Completion Processors（同步，在状态裁决前执行）               │
│    - workspace snapshot / commit                                     │
│    - auto-apply（失败会停在可处理状态）                              │
│                                                                      │
│  · Event-Cursor Automation（可靠事件触发，基于 events.jsonl）        │
│    - checkpoint / cursor 持久化                                      │
│    - 去重 key: automation_id + event_id                              │
│                                                                      │
│  ┌─────────────┐         ┌───────────────────────────┐              │
│  │ gloop CLI   │◀────────│   Platform Tools Layer    │              │
│  │ （对外入口） │         │  （内部 handler 复用层）   │              │
│  └─────────────┘         └───────────────────────────┘              │
└──────────────────────────────────────────────────────────────────────┘
```

### 4.2 Loop Engineering 六要素映射（修正版）

| 要素 | Gloop 现状 | 重构动作 |
|------|-----------|---------|
| **Automations** | ✅ 已有 cron + FanOut，5 个预置模板 | 增加事件型 automation（基于持久化 event log + cursor，可靠执行） |
| **Worktrees** | ✅ per-quest workspace 隔离 | 保留，git worktree 模式加强 |
| **Skills** | ✅ agent-side instruction package | 保留，索引注入 system prompt（已实现） |
| **Plugins / Tools** | ⚠️ 原生工具 + 平台机制双轨，概念容易混 | **明确边界**：原生工具是 agent 能力，gloop CLI 是机制入口，不做"统一 ABI" |
| **Sub-agents** | ⚠️ FanOut = 宏循环级并行，单阶段内 sub-agent 是 agent 自己的事 | **明确边界**：fan-out 是 gloop 级（多 quest 并行），sub-agent 是 agent 级（单 quest 内自行 spawn） |
| **State** | ⚠️ 有 QuestService + events + meta，但 domain/fsstore 双模型，pipeline 不持久化 | **收口**：状态迁移字段统一走 QuestService；pipeline 定义持久化到 meta；事件型副作用有 cursor 可恢复 |

---

## 5. 详细设计

### 5.1 通用阶段管道（宏循环重构）

#### 5.1.1 现状

- Domain 层已有 `Pipeline` / `PhaseDef` 数据结构
- 但 `ReworkTargetIdx()` 硬编码返回 0
- `pipeline.go` 里 `buildPhaseConfig()` 还是硬编码 `switch phaseIdx`
- `macro_loop.go` 里直接写 `phase 0 = 剑士 / phase 1 = 法师` 的逻辑还不少
- Coding 特化逻辑（commit / apply / diff-summary）散落在宏循环主路径
- Pipeline 不持久化，每次从 `DefaultPipeline()` 派生；恢复逻辑假设两阶段

#### 5.1.2 PhaseDef 演进（基于现有结构扩展）

在已有 `PhaseDef` 基础上增加字段，不重新造：

```go
// 已有字段（保留）:
//   Index, Name, DisplayName, Role, Class, Goal, ReadOnly, DefaultHints, DefaultMaxTurns

// 新增字段：
type PhaseDef struct {
    // ... 已有字段 ...

    // 阶段结束信号标识（检测到这个信号算阶段结束）
    // 注意：不是"工具名"，是信号标识。agent 通过 gloop CLI 写信号文件触发。
    // 可以对应多个 CLI 子命令（如 phase done / phase fail 都算结束信号）
    EndSignal string

    // 返工目标阶段索引（-1 = 不支持返工）
    // 替代硬编码的 ReworkTargetIdx() = 0
    ReworkTo int

    // 阶段退出条件（机制层只做布尔判断，不做策略指导）
    ExitCriteria PhaseExitCriteria

    // 预完成处理器（同步执行，影响状态裁决）
    PreCompletionProcessors []string

    // 提交后订阅者（异步执行，best-effort，不影响状态）
    AfterCommitSubscribers []string
}

type PhaseExitCriteria struct {
    // 最低通过分数（review 阶段用，0 = 不限制）
    // 平台只比较法师显式提交的 score，不自行评估质量
    MinScore int
}
```

默认两阶段管道（向后兼容，`ReworkTo` 默认 0）：

```go
func DefaultPipeline() Pipeline {
    return Pipeline{
        {
            Index:       0,
            Name:        "warrior_execute",
            DisplayName: "剑士执行",
            Role:        PhaseRoleExecute,
            Class:       model.ClassWarrior,
            Goal:        "按照用户需求完成任务，交付可验证的产出",
            ReadOnly:    false,
            EndSignal:   "phase_checkpoint", // 信号标识，不是工具名
            ReworkTo:    0,
            DefaultHints: 2,
            PreCompletionProcessors: []string{"workspace_commit"}, // conditional: 仅 coding 任务
            AfterCommitSubscribers:   []string{"diff_summary"},
        },
        {
            Index:       1,
            Name:        "mage_review",
            DisplayName: "法师评审",
            Role:        PhaseRoleReview,
            Class:       model.ClassMage,
            Goal:        "评审剑士的产出，给出通过/返工/拒绝结论",
            ReadOnly:    true,
            EndSignal:   "review_quest", // 法师用 gloop review 提交，信号标识独立
            ReworkTo:    0, // 返工回到 phase 0
            DefaultHints: 2,
            ExitCriteria: PhaseExitCriteria{MinScore: 0}, // 按强度动态算
            PreCompletionProcessors: []string{"auto_apply"}, // conditional: 仅 automation policy 授权时
            AfterCommitSubscribers:   []string{"review_notify"},
        },
    }
}
```

**关于 conditional processor**：
PhaseDef 上声明的 processor 是"可选处理器"，不是"无条件执行"。
每个 processor 有自己的启用条件判断：
- `workspace_commit`：仅 coding 任务（有 workspace_diff）且非只读阶段执行
- `auto_apply`：仅 `hasAutoApply(q)` 为 true（来自 automation policy 授权）时执行
- 未命中条件的 processor 直接 no-op，不报错

这样设计的原因：
- Pipeline 定义描述"这个阶段可能发生哪些副作用"
- 具体执不执行由 processor 自己的 guard condition 判断
- 不需要为每种 quest 类型造一套独立的 pipeline 定义

#### 5.1.3 Pipeline 持久化契约（Phase 0.5）

**问题**：当前 pipeline 定义不持久化到 meta.json，每次从 `DefaultPipeline()` 派生。如果引入 3 阶段管道或自定义管道，重启后恢复、前端展示、状态推断都会退回到"两阶段假设"。

**设计**：在 meta.json 中保存 pipeline 相关字段，作为恢复和展示的唯一事实源。

```go
// fsstore.QuestMeta 新增字段（旧数据自动补齐）
type QuestMeta struct {
    // ... 已有字段 ...

    PipelineVersion  int         `json:"pipeline_version"`  // 管道版本号，用于迁移
    CurrentPhaseIdx  int         `json:"current_phase_idx"` // 显式保存，不再从 Status 推断
    PipelineName     string      `json:"pipeline_name"`     // 管道名称，default / custom / ...
    PhaseCount       int         `json:"phase_count"`       // 阶段总数，前端展示用
}
```

**迁移策略**：
- 旧 quest 加载时，`QuestRepositoryAdapter` 检测到 `pipeline_version == 0` 时，按 `DefaultPipeline()` 补齐字段并写回 meta
- 状态推断优先级：`CurrentPhaseIdx` 字段 > Status 推断（向后兼容兜底）
- `fsstore.QuestMeta.CurrentPhaseIdx()` 方法保留，优先读字段，fallback 到推断

**domain 侧同步 + 恢复一致性保证**：

分两种 pipeline 类型处理：

| Pipeline 类型 | 持久化策略 | 恢复方式 |
|--------------|-----------|---------|
| **内置管道**（default / design_review / ...） | 存 `pipeline_name` + `pipeline_version` | 从注册表按 name + version 查找定义 |
| **自定义管道**（custom，3 阶段等） | 存 `pipeline_name = "custom"` + 完整 `pipeline_def` snapshot | 直接从 meta 读取定义，不依赖注册表 |

设计原则：
- 默认管道只存 name + version，避免 bloated meta
- 自定义管道必须 snapshot 完整 `[]PhaseDef` 到 meta（或单独 `pipeline.json`），保证重启后用**创建时的定义**恢复，不受注册表变更影响
- 注册表的 pipeline 版本号单调递增，旧 quest 用旧版本恢复
- `pipeline_def_hash` 作为校验：如果从注册表查到的 hash 和 meta 里存的不一致，说明注册表变了，fallback 到 snapshot 定义

**一致性校验（启动时/恢复时必跑）：**
- 校验 `phase_count == len(Phases[])`（或 pipeline snapshot 的阶段数）
- 不一致直接报错 + 标记 quest 为 `blocked`，不能靠补齐逻辑掩盖脏数据
- 校验 `current_phase_idx < phase_count`
- 自定义管道额外校验 `pipeline_def_hash == hash(pipeline_def)`

**验收**：
1. 一个 3 阶段自定义 quest 运行到 phase 2 时被杀掉，重启后能正确恢复到 phase 2
2. 修改注册表中的 default pipeline 定义后，旧 quest 重启仍用创建时的版本恢复，不被新版本影响
3. 手动把 meta 的 phase_count 改成 3 但 Phases 只有 2 个，恢复时报错并 blocked，不静默补齐

#### 5.1.4 Pre-Completion vs After-Commit 边界

这是本次重构最关键的边界划分，直接影响状态一致性。

| 维度 | Pre-Completion Processor | After-Commit Subscriber |
|------|-------------------------|------------------------|
| **执行时机** | 状态变更**之前**，同步执行 | 状态变更**之后**，异步订阅 |
| **失败影响** | 失败会阻止状态变更，或进入可处理的失败状态 | 失败不影响主状态，重试或丢弃 |
| **顺序保证** | 严格按定义顺序执行 | 不保证顺序，fan-out |
| **错误语义** | 同步返回错误，调用方处理 | 只记日志，不回滚 |
| **典型用途** | workspace commit / snapshot、auto-apply | 通知、diff cache、统计、派生任务 |
| **可靠性** | 必须成功（或有明确失败路径） | best-effort，可丢可重放 |

**Auto-apply 示例（同步裁决路径）：**

```
当前实现（正确，保留同步语义）：
  1. 法师评审通过 → verdict = pass
  2. MoveToUserReview (状态: user_review)
  3. 同步 applyWorkspace()
     - 成功 → CompleteAutoApplySuccess (状态: success)
     - 失败 → recordApplyFailure (停在 user_review，ApplyStatus=failed)

重构后仍保持同步，不事件化：
  1. review 阶段结束，verdict = pass
  2. 执行 PreCompletionProcessors:
     - auto_apply processor 同步跑 apply
     - 失败 → 返回错误，宏循环停在 user_review 状态
  3. 所有 pre-completion 成功 → QuestService.MarkSuccess()
  4. 发 quest.success 事件 → after-commit subscribers 异步跑
```

**为什么 auto-apply 不能事件化**：
- `quest.success` 事件发出后，前端、通知、统计都会看到"已成功"
- 如果 auto-apply 作为 subscriber 在事件后执行且失败，就会出现"已成功但又失败"的补偿状态
- 状态补偿会扩散到所有消费方，复杂度指数级上升
- 结论：**影响终态裁决的副作用，必须走同步预完成路径**

#### 5.1.5 宏循环管道化

`runMacroLoop` 改造成遍历 `Pipeline.Phases` 的通用循环：

```
当前状态（硬编码两阶段）：
  runWarriorPhase → runMagePhase → check verdict → rework / pass

目标状态（管道化）：
  for phaseIdx := currentPhase; phaseIdx < len(pipeline); phaseIdx++ {
      phase := pipeline[phaseIdx]

      // 1. 执行阶段
      outcome := runAgentPhase(phaseIdx, input)

      // 2. 同步执行 pre-completion processors（conditional，不满足的 no-op）
      for _, procName := range phase.PreCompletionProcessors {
          err := runPreCompletionProcessor(procName, q, outcome)
          if err != nil {
              // 进入可处理失败状态，不继续推进
              handlePreCompletionFailure(q, phase, procName, err)
              return
          }
      }

      // 3. 推进状态（QuestService 语义化方法，不搞 MoveToNextPhase 上帝方法）
      if outcome.IsRework {
          // 评审阶段要求返工
          questService.RequestReworkToPhase(qid, phase.ReworkTo, outcome.ReworkHints)
          phaseIdx = phase.ReworkTo - 1 // 下次循环跳到返工目标
      } else if phaseIdx == len(pipeline)-1 {
          // 最后一个阶段完成 → quest 成功
          questService.CompleteQuest(qid, outcome)
      } else if outcome.IsUserReview {
          // 需要人工终审
          questService.MoveToUserReview(qid)
          return // 暂停，等用户操作
      } else {
          // 进入下一个阶段
          questService.CompletePhase(qid, phaseIdx)
      }

      // 4. 发事件 → after-commit subscribers 异步执行（不等待）
      publish(PhaseEndedEvent)
  }
```

**QuestService API 边界（语义化方法，不造上帝方法）：**

| 方法 | 语义 | 状态迁移 |
|------|------|---------|
| `StartQuest` | 启动 quest | pending → running |
| `CompletePhase` | 完成一个执行阶段，进入下一阶段 | running phase N → running phase N+1 |
| `RequestReworkToPhase` | 评审阶段要求返工到指定阶段 | reviewing → running + rework_count++ |
| `MoveToUserReview` | 需要人工终审 | reviewing / running → user_review |
| `ResolveUserReview` | 用户评审完成 | user_review → success / failed / rework |
| `CompleteQuest` | quest 成功完成 | running / user_review → success |
| `FailQuest` | quest 失败 | any → failed |
| `BlockQuest` | quest 阻塞 | running / reviewing → blocked |
| `CancelQuest` | 取消 quest | any → cancelled |

每个方法只承载明确的状态语义，不做"通用阶段推进"。

注意：
- 返工目标由 `PhaseDef.ReworkTo` 指定，不再硬编码回 phase 0
- 状态推进统一走 QuestService，不直接写 fsstore
- after-commit subscribers 通过 EventBus 触发，宏循环不等待
- `MoveToNextPhase` 这种过宽的上帝方法不实现，每个状态迁移用语义化方法

### 5.2 结构化 Artifact 与上下文传递

> 策略：小步快跑，增量补充。不搞大爆炸式 PhaseArtifact 结构迁移。
> 先解决最痛的断流问题，再逐步结构化。

#### 5.2.1 Phase 1：增量补全（不动大结构）

**现状**：`PhaseReviewArtifact` 只有 `Assistant`（字符串摘要）和 `PlatformToolEvidence`（字符串）两个字段。剑士的结构化 deliverables 存在 `q.Outputs` 里，法师 context 完全没用到。

**Phase 1 改动（最小侵入）：**

1. `PhaseArtifactForReview` 返回值增加 `Outputs []QuestArtifact`，从 `q.Outputs` 读取（source = warrior_phase）
2. 法师 context pack 增加 `warrior_deliverables` block，内容是结构化产出物清单
3. 对 `kind=file` / 带 `path` 的 deliverable 做存在性自动核验，结果写入 block
4. `warrior_artifact`（自然语言摘要）保留，向后兼容

**为什么不直接搞大的 PhaseArtifact 结构迁移：**
- 旧数据迁移成本高，容易出问题
- deliverables 已经有 `q.Outputs` 这个事实源，不需要另造一份
- 先验证"结构化 deliverables 注入后法师评审质量是否提升"，再决定要不要继续结构化

#### 5.2.2 Evidence 分类（增量）

**现状**：`PlatformToolEvidence` 是一坨字符串，混合了 auto collected、native tool calls、platform mechanisms 三种不同信任级的证据。

**Phase 1 改动：**

`PhaseArtifactForReview` 返回时，把证据按信任级拆成三个分区：

| 分区 | Trust | 来源 | 说明 |
|------|-------|------|------|
| `auto_collected` | platform | `collectAutoEvidence` 跑的命令 | git diff / go build / go vet 等，平台主动跑的，可信度最高 |
| `platform_mechanisms` | platform | gloop CLI signal、phase_checkpoint 等 | 平台机制调用记录，gloop 自己产生的 |
| `native_tool_calls` | agent_output | ACP observed native tool calls | agent 主动调用的原生工具，仅作参考，可信度低于平台产生的 |

法师 context pack 中：
- `platform_auto_evidence` block（trust=platform）放 auto_collected
- `warrior_native_tool_calls` block（trust=agent_output）放 native_tool_calls
- `warrior_artifact` 里自然语言摘要保留
- `platform_mechanisms` 内容较短，可直接放进 artifact block

**注意**：ACP native tool observed 是 agent 输出侧事实，**不能叫 platform 证据**。它的可信度低于 gloop 自己跑的 auto evidence 和 CLI signal。

#### 5.2.3 后续可选项（Phase 2+，视价值再定）

如果 Phase 1 验证了结构化的价值，可以考虑进一步升级 PhaseArtifact 结构：

```go
// 这是 Phase 2+ 的可选方向，不是 Phase 1 范围
type PhaseArtifact struct {
    Summary        string              // 人类可读摘要（向后兼容）
    Deliverables   []DeliverableItem   // 结构化产出物
    Evidence       PhaseEvidence       // 分区证据
    PhaseIdx       int                 // 阶段元数据
    PhaseName      string
    DurationMs     int64
    TurnCount      int
    Status         string
}
```

只有当 Phase 1 证明"结构化确实有用"时，才做完整迁移。避免先陷入迁移泥潭。

#### 5.2.4 返工上下文补全

**现状**：剑士返工轮次只能拿到 `ReviewHints`（法师写的修改建议），拿不到完整的 comment / score / verdict。

**改动**：
- 返工剑士 context pack 增加 `rework_full_review` block，包含上一轮法师的完整评审（verdict + score + comment + hints）
- 数据来源：`reviews.jsonl` 中最新的一条 ReviewRecord
- `ReviewRecord` 结构已有 verdict/comment/rewrite_hints/score 等字段，直接读就行

新增 blocks：

| Block 名称 | Trust | 说明 |
|-----------|-------|------|
| `rework_full_review` | agent_output | 上一轮法师的完整评审（verdict + score + comment + hints） |
| `previous_round_summary` | agent_output | 上一轮剑士的交付摘要（已存在，移到 core blocks） |

目的：让剑士理解**整体质量差距**，而不只是改几个点，减少挤牙膏式返工。

#### 5.2.5 法师 Context Pack 重构（Phase 1 后状态）

**核心 block（始终直接注入）：**

| Block 名称 | Trust | 说明 |
|-----------|-------|------|
| `quest_user_intent` | user | 用户原始需求 |
| `warrior_deliverables` | agent_output | 结构化产出物清单（checker 的 checklist） + 存在性核验结果 |
| `review_protocol` | platform | 评审协议和 CLI 用法 |

**Fallback 策略**：如果剑士没有提交结构化 deliverables（`q.Outputs` 为空），则将 `warrior_artifact`（自然语言摘要）提升为核心 block，保证法师第一眼总能看到评审对象。

**证据 block（文件式上下文，按需读取）：**

| Block 名称 | Trust | 说明 |
|-----------|-------|------|
| `platform_auto_evidence` | platform | 自动采集的可信证据（git diff / build / vet 等） |
| `warrior_native_tool_calls` | agent_output | 剑士所有原生工具调用记录 |
| `warrior_summary` | agent_output | 剑士自然语言总结（原 warrior_artifact 文本） |
| `review_history` | mixed | 返工历史脉络（前几轮摘要 + 评审意见） |
| `acceptance_criteria` | platform | 验收标准（如有） |
| `review_requirements` | platform | 评审强度说明、质量分阈值等 |

设计思路：
- 核心 block 少而精，让法师第一眼看到"要评审什么"（deliverables）
- 详细证据写文件，法师需要时自己读
- 符合「最少干预」原则：平台只保证信息可达，不强迫 agent 全量消费
- 符合 CLI-first 原则：文件是最自然的交互载体

#### 5.2.6 deliverables 存在性自动核验

平台在构建法师 artifact 时，对 `kind=file` 且带 `path` 的 deliverable 自动检查文件是否存在。

- 纯检查，不做智能判断
- 存在 ≠ 正确，法师仍需自行验证内容
- 减轻法师"有没有这个文件"的基础核验负担
- 验证逻辑很轻量，不增加明显开销
- 核验结果写进 `warrior_deliverables` block（带 ✅/❌ 标记）

### 5.3 状态管理与事件驱动

#### 5.3.1 QuestService 收口

**现状**：QuestService 设计上是"状态变更的单一入口"，但实际落地不完全：
- orchestrator 层仍有直接写 fsstore 的情况（如 `SaveQuest` 更新 apply 状态）
- domain/fsstore 双模型，真实事实源不清
- 状态迁移字段和非状态字段混在一起

**重构目标（更准确的表述）：**
- **状态迁移字段只能通过 QuestService**：Status、CurrentPhaseIdx、ReworkCount 等
- **非状态字段可以由专门 store 写**：artifact、review log、session row、apply backup 等，但要有清晰 ownership
- Orchestrator 不直接调用 `fsstore.SaveQuest` 来改状态
- 所有状态变更触发事件（已实现，确保不遗漏）

**不追求的目标**：
- ❌ 所有写操作都走 QuestService（那会把 QuestService 变成上帝对象）
- ❌ 立即消除 domain/fsstore 双模型（那是更大的重构，超出本 spec 范围）

#### 5.3.2 事件型 Automation（可靠执行版本）

> 不能直接基于内存 EventBus 承诺可靠执行。
> 内存 bus 适合通知、前端、统计；automation 触发需要持久化事实源 + 可恢复。

**前置依赖：事件 ID 化**

当前 `events.Event` / `QuestEventRow` 只有 Timestamp/Type/QuestID/SessionID/Payload，没有稳定单调的 event id。Timestamp 不能当 cursor（同毫秒多事件、排序、去重都会出问题）。

Phase 3 必须先补事件 ID 化：

```go
// fsstore.QuestEventRow 新增字段
type QuestEventRow struct {
    ID        int64           `json:"id"`         // 单调递增，每个 quest 内从 1 开始
    Timestamp int64           `json:"ts"`
    Type      string          `json:"type"`
    QuestID   string          `json:"qid"`
    SessionID string          `json:"sid,omitempty"`
    Payload   json.RawMessage `json:"payload,omitempty"`
}

// events.Event 同步新增 ID 字段
type Event struct {
    ID        int64
    Type      EventType
    QuestID   string
    SessionID string
    Payload   json.RawMessage
    Timestamp int64
}
```

事件 ID 生成规则：
- 每个 quest 内独立单调递增，从 1 开始
- AppendEvent 时读取当前最大 ID + 1
- 重启后从 events.jsonl 最后一行恢复 max_id
- cursor 表示为 `(quest_id, event_id)` 对

**跨 quest 监听的全局事件日志**：

如果 automation 需要跨 quest 监听（如"所有 coding quest 成功后通知"），需要全局事件日志：
- 文件：`events/global.jsonl`（或按天分片）
- 每条事件有全局单调 `global_id`
- 每个 quest 事件 append 时同步写全局日志
- 跨 quest automation 用 `global_id` 作 cursor

Phase 3 先做**单 quest 内事件自动化**（cursor = quest_id + event_id），全局事件日志作为可选项。

**可靠消费架构**：

```
内存 EventBus（实时，best-effort）
    │
    ├── best-effort subscribers：通知 / 前端 / 统计（直接订阅，可丢）
    │
    └── Automation Engine（实时加速 + 持久化补偿）
            │
            ├── 实时路径：从内存 bus 拿到事件 → 检查 cursor → 执行 → 更新 cursor
            │
            └── 补偿路径：启动/恢复时，从 events.jsonl 扫描 cursor 之后的事件
                     按 event_id 去重，保证 at-least-once

持久化事实源：events.jsonl（append-only，崩溃安全，带单调 ID）
cursor 持久化：automation_state.json（每个 automation 一个 cursor: quest_id + last_event_id）
去重 key：automation_id + quest_id + event_id
```

**关键数据结构：**

```go
type AutomationTrigger struct {
    Type   TriggerType       // schedule / event
    Cron   string            // schedule 类型用
    Event  events.EventType  // event 类型用：quest.success / quest.failed / phase.ended
    Filter map[string]string // 事件过滤条件
    Scope  AutomationScope   // per_quest / global
}

type AutomationState struct {
    AutomationID  string `json:"automation_id"`
    LastQuestID   string `json:"last_quest_id,omitempty"`  // per-quest scope 的 cursor
    LastEventID   int64  `json:"last_event_id"`            // cursor，最后处理的事件 ID
    LastGlobalID  int64  `json:"last_global_id,omitempty"` // global scope 的 cursor
    LastRunTs     int64  `json:"last_run_ts"`
    RunCount      int    `json:"run_count"`
}

// 去重：automation_id + quest_id + event_id 作为唯一键
// 已执行过的事件直接跳过
```

**Automation Engine vs After-Commit Subscriber 的边界：**

| 维度 | After-Commit Subscriber | Automation Engine |
|------|------------------------|-------------------|
| 消费源 | 内存 EventBus | 持久化 event log + 内存 bus 加速 |
| 可靠性 | best-effort，可丢 | at-least-once，可恢复，可去重 |
| 状态 | 无状态 | 有 cursor 状态 |
| 典型用途 | 通知、UI、统计、diff cache | 派生任务、自动执行、工作流串联 |
| 失败处理 | 记日志，不重试 | 重试 + 死信 |

注意：Phase 2 交付物里的 after-commit subscribers 是**轻量 best-effort**，不是可靠事件自动化。可靠事件自动化是 Phase 3 的独立模块，基于 event log + cursor。

**Phase 3 范围：per-quest 可靠事件自动化**

Phase 3 先做**单 quest 内**的可靠事件自动化：
- cursor = `(quest_id, event_id)`
- 消费源：单个 quest 的 `events.jsonl`
- 典型场景：phase.ended 触发 triage / note、quest.failed 触发 notify、quest.success 触发本 quest 后续派生动作

**跨 quest workflow automation（design → execute → test 串联）** 不在 Phase 3 范围内，需要全局事件日志 + workflow instance 状态，放到 Phase 4 再做。
- 顺序保证：按 event_id 升序处理

**为什么不用消息队列：**
- 单机单用户场景，events.jsonl 足够
- 引入 Kafka/Redis 之类的组件完全没必要
- cursor + append log 是最简单的可靠消费模式

**内置事件模板（Phase 3 per-quest 范围）：**
- `phase.ended → triage` — 阶段结束后自动分类/打标签（本 quest 内）
- `quest.failed → notify` — 失败后发通知（本 quest 内）
- `quest.success → apply-followup` — coding 任务成功后触发后续验证（本 quest 内，如跑测试）
  - 注意：这是**事件型派生任务**，不是 auto-apply 主体。auto-apply 主体是同步 pre-completion 处理器
- `quest.success → spawn-child` — 成功后 spawn 子 quest（但子 quest 后续事件不在本 automation 跟踪范围内）

**跨 quest workflow（如 design → execute → test 全链路串联）**：Phase 4 范围，需要全局事件日志 + workflow instance id，不在本 spec 详述。

### 5.4 Platform Tools 定位澄清

#### 5.4.1 现状与问题

`platformtools` 包名字容易让人误解为"平台提供给 agent 的工具集"，但实际上：
- Agent 不直接调用这些"工具"
- Agent 通过 `gloop CLI` 调用机制
- CLI 底层复用 `platformtools` 的 handler 逻辑
- 测试时 mock executor 直接调用 `platformtools`

#### 5.4.2 重构后定位

- **对外（agent 侧）**：只暴露 `gloop CLI` 作为机制入口，不提"平台工具"
- **对内（实现侧）**：`platformtools` 包是"机制 handler 层"，CLI 和测试都复用它
- **名字可以不改**（避免破坏性重构），但文档和注释要写清楚定位
- **Phase 2 完成 `EndTool` → `EndSignal` 重命名**：代码里 `PhaseDef.EndTool` 字段改名为 `EndSignal`，保留兼容 shim（旧字段名 alias 或迁移逻辑）

**不做的事：**
- ❌ 不搞"统一工具 ABI"把原生工具和平台机制混在一起
- ❌ 不把平台机制包装成 agent tool use 格式暴露
- ❌ 不改变 agent 侧的调用方式

### 5.5 文件式上下文

#### 5.5.1 现状（实现后）

- `context_file_mode` 是显式配置开关，`GLOOP_CONTEXT_FILE_MODE` 仍作为紧急覆盖
- `WriteFileContext` 写 `.gloop/context/*.md`
- `gloop context get <block-name>` 已实现，支持 `--json`
- 实际可用的读取方式包括 `cat .gloop/context/<filename>` 和 `gloop context get <block-name>`
- File-L4 实验验证了质量提升 + token 节省，但也增加了耗时

#### 5.5.2 分阶段落地

**Phase 1-3 实现结果**：
- 保持默认关闭，不强行全量开启
- 修复 `WriteFileContext` 文案，明确 `gloop context get`
- 实现 `gloop context get <block-name>`，支持 `--work-dir` 与 `--json`
- 新增 `context_file_mode` 配置开关，保留 `GLOOP_CONTEXT_FILE_MODE` 紧急覆盖
- 全量默认开启仍需真实 agent 灰度数据，未在 v1 scope 内强推

**为什么不急着默认开启：**
- 不同 executor 对工作目录、shell、只读模式的行为不同
- agent 不一定会主动读文件（尤其是 ACP 模式下的 agent）
- 如果默认开启但 agent 不读，关键信息就丢了
- "降 token" 的前提是"信息可达"，先保证可达再谈优化

---

## 6. 里程碑

按依赖关系和价值密度排序，小步快跑，每步都可验证。

### 开工顺序建议

Phase 编号按依赖关系排列，但实际开发可以并行/调整：

| 顺序 | 阶段 | 前置依赖 | 说明 |
|------|------|---------|------|
| 1 | **Phase 1** | 无 | 上下文断流修复，纯 prompt / context 层改动，价值最高、风险最低，可直接开工 |
| 2 | **Phase 0.5** | Phase 2 之前 | Pipeline 持久化契约，是 Phase 2（宏循环管道化）的前置，不需要现在做 |
| 3 | **Phase 2** | Phase 0.5 | 宏循环管道化 + post-processor，需要数据层先对齐 |
| 4 | **Phase 3** | Phase 2 | Loop Engineering 补全，per-quest 可靠事件自动化 |

下面按阶段编号顺序排列（0.5 → 1 → 2 → 3），但实际开工从 Phase 1 开始。

### Phase 0.5：统一 Pipeline 持久化契约（1 天）

**目标**：先把数据层的事实源对齐，不然后面改宏循环都是空中楼阁。

**交付物：**
- [x] `fsstore.QuestMeta` 增加 `pipeline_version` / `current_phase_idx` / `pipeline_name` / `phase_count` / `pipeline_def_hash` 字段
- [x] 自定义管道持久化完整 `pipeline_def` snapshot（meta）
- [x] `QuestRepositoryAdapter` 旧数据自动补齐（`DefaultPipeline()` 推导 + 写回）
- [x] `CurrentPhaseIdx()` 优先读字段，fallback 到 Status 推断
- [x] 默认 pipeline 版本固定为 v1；完整注册表多版本管理推迟到后续
- [x] 单元测试覆盖旧数据迁移、3 阶段自定义管道、坏数据拒绝

**验收标准**：
- meta.json 里能看到 pipeline 相关字段
- 一个跑到 phase 2 的 3 阶段 quest 重启后正确恢复
- 修改注册表 default pipeline 后，旧 quest 重启仍用创建时的版本

### Phase 1：上下文传递修复（2-3 天）

**目标**：解决最痛的信息断流问题，立竿见影提升评审质量。
**策略**：增量补充，不搞大结构迁移。
**顺序**：deliverables 断流 → 返工补全 → evidence 分类

**交付物：**
- [x] `PhaseArtifactForReview` 返回值增加 `Outputs []QuestArtifact`（从 q.Outputs 读，source=warrior_phase）
- [x] 法师 context pack 增加 `warrior_deliverables` block（核心 block）
- [x] deliverables 文件存在性自动核验（本地文件 verified，外链 external）
- [x] **Fallback**：没有 outputs 时，`warrior_artifact`（自然语言摘要）提升为核心 block
- [x] 返工上下文补全：剑士 pack 注入完整上一轮 review（verdict + score + comment + hints）
- [x] Evidence 分类（auto_collected / platform_mechanisms / native_tool_calls）
- [x] 旧数据兼容：没有 Outputs 时 deliverables block 不显示，fallback 到 warrior_artifact
- [x] 单元测试覆盖

**验收标准**：
- 法师能拿到结构化 deliverables checklist + 存在性核验标记
- 返工轮次剑士能看到完整评审意见，不只 hints
- Evidence 按信任级分三区展示
- 老数据/老 agent 交付时，法师仍能看到 warrior_artifact 作为评审对象

### Phase 2：宏循环管道化 + Post-Processor（1 周）

**目标**：宏循环变成通用阶段管道，coding 逻辑下沉为 pre-completion / after-commit 两类处理器。
**前置依赖**：Phase 0.5（pipeline 持久化契约）

**交付物：**
- [x] PhaseDef 新增字段：`EndSignal`（注意：从 `EndTool` 重命名，保留兼容 shim） / `ReworkTo` / `ExitCriteria` / `PreCompletionProcessors` / `AfterCommitSubscribers`
- [x] `PhaseDef.EndTool` → `EndSignal` 字段兼容迁移
- [x] `ReworkTargetIdx()` 从硬编码改为读 `PhaseDef.ReworkTo`
- [x] 宏循环从硬编码两阶段改为由 pipeline definition 驱动默认两阶段 + 线性多评审阶段
- [x] **QuestService 语义化阶段 API**：CompletePhase / RequestReworkToPhase / CompleteQuest，不造 MoveToNextPhase 上帝方法
- [x] Pre-completion processor 接口 + 2 个内置处理器（workspace_commit / auto_apply）
  - 注意：processor 是 conditional 的，未授权/非适用场景下 no-op
- [x] After-commit subscriber 薄层 + diff_summary；review_notify 由现有 notifications.EventSubscriber 处理
  - 注意：after-commit subscribers 是 best-effort，不是可靠事件自动化
- [x] EventBus 对接通知类 subscriber；diff_summary 走 after-commit helper
- [x] QuestService 收口：状态迁移字段统一走 QuestService 语义化 API
- [x] 3 阶段线性评审管道可配置、可运行、可恢复
- [x] E2E/golden 测试覆盖：2 阶段默认 + 3 阶段自定义 + 返工 + 恢复 + auto-apply 失败场景

**验收标准**：
- 可以配置一个 3 阶段管道（执行 → 初审 → 复审）跑通全流程
- 服务重启后能从第 2/3 阶段正确恢复
- coding 逻辑（commit / apply）完全不在宏循环主路径里
- auto-apply 失败仍停在 user_review 状态（同步语义保持）
- 前端/展示层不再依赖"两阶段假设"，从 pipeline 定义动态展示
- QuestService 没有 MoveToNextPhase 这种过宽的通用方法

### Phase 3：Loop Engineering 补全（1-2 周）

**目标**：六要素完整落地，per-quest 自动化能力上台阶。
**范围**：单 quest 内可靠事件自动化；跨 quest workflow（design→execute→test 全链路）不在本 phase，放 Phase 4。
**前置依赖**：Phase 2

**交付物：**
- [x] 事件 ID 化：`QuestEventRow.ID` 单调递增；API trace 输出 ID
  - 旧 events.jsonl 兼容：无 ID 的事件按行号派生伪 ID，新事件从 max+1 开始
- [x] 可靠事件型 automation 框架（per-quest 范围）
  - 消费源：单个 quest 的 events.jsonl
  - cursor：`(quest_id, event_id)`
  - 去重 key：`automation_id + quest_id + event_id`
  - 重启补偿扫描：从 events.jsonl 恢复 cursor 之后的事件
  - 与 after-commit best-effort subscribers 明确分层
- [x] 内置 per-quest event actions：triage note / notify event / default note；apply-followup 保留为 Phase 4 workflow 范围
- [x] `gloop context get` CLI 命令实现
- [x] 文件式上下文配置开关 + 环境变量覆盖；默认全量开启推迟到真实 agent 灰度后
- [x] Fan-out 与 sub-agent 边界文档化
- [x] Worktree 模式加强（评审前 snapshot worktree，保证 diff 包含未跟踪文件）
- [x] Platform Tools 定位文档化（澄清是内部机制层，不是对外工具）
- [x] 完整的 Loop Engineering 映射文档

**验收标准**：
- 单 quest 内：`phase.ended` 事件触发 triage note，服务重启后不丢、不重复
- `gloop context get` CLI 能正常读取文件式上下文
- 文件式上下文灰度开启后，质量不下降、token 开销下降
- 明确文档化：per-quest automation 的边界，以及跨 quest workflow 需要 Phase 4 的全局事件日志

---

## 7. 风险与缓解

| 风险 | 影响 | 概率 | 缓解措施 |
|------|------|------|---------|
| 重构破坏现有 quest 数据 | 高 | 中 | 向后兼容设计，旧数据自动迁移，提供回滚脚本；Phase 0.5 先做数据层对齐 |
| 上下文模式切换影响 agent 行为 | 中 | 中 | 灰度发布，保留开关，先在测试 quest 上验证；Phase 1 不默认开启文件模式 |
| 宏循环重构引入回归 bug | 高 | 中 | 分阶段落地，每阶段充分 E2E 测试，保留旧路径可回退 |
| auto-apply 事件化破坏状态语义 | 高 | 低 | **明确不事件化**，走 pre-completion 同步路径，本 spec 已修正 |
| 事件型 automation 丢事件 / 重复触发 | 中 | 中 | 事件 ID 化 + 持久化 cursor + events.jsonl 补偿扫描 + 去重 key；at-least-once 语义 |
| 事件 ID 迁移破坏旧 events.jsonl 格式 | 中 | 低 | 旧文件兼容：无 ID 的事件按行号派生伪 ID，新事件从 max+1 开始 |
| 注册表 pipeline 变更导致旧 quest 恢复异常 | 中 | 中 | 自定义管道 snapshot 完整定义；内置管道带版本号，旧 quest 用旧版本恢复 |
| 结构化 artifact 增加 token | 低 | 低 | 文件式上下文抵消：核心 block 减少 + 证据按需读取；Phase 1 先只加 deliverables block |
| 3 阶段管道恢复失败 | 中 | 中 | Phase 0.5 先做持久化契约；验收标准明确要求重启恢复测试 |
| 工作量比预估大 | 中 | 中 | 每个 Phase 都可独立交付和验证；Phase 1 先做，验证价值后再继续 |

---

## 8. 后续探索方向（不列入本 spec）

- **Phase 4：跨 quest workflow automation** — 全局事件日志 + global_id + workflow instance 状态，支撑 design→execute→test 全链路串联
- 多 quest fan-out 的负载均衡和资源隔离
- Human-in-the-loop 阶段（人工评审作为一个 phase）
- Plugin 系统（第三方接入自定义后置处理器 / 阶段类型）
- 跨 quest 的状态/记忆共享
- Quest 模板市场（常用管道配置分享）
- 完整的 domain aggregate 重构（彻底消除 fsstore/domain 双模型）
