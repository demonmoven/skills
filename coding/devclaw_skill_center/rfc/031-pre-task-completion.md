# 3.1 Pre Task Completion

> **本节目标**：理解 Pre Task Completion 的完整含义——从需求构建到代码落地的全流程交付，以及 SDD 和 TDD 两种驱动方式。

---

## 概念

**Pre Task Completion** 是从需求到代码落地的**完整交付过程**，而非仅指"编码之前"的准备工作。它涵盖：

```text
需求构建 → 调研 → 技术方案 → 任务拆解 → 编码/单测落地
   │         │        │          │           │
   └─────────┴────────┴──────────┴───────────┘
              Pre Task Completion 的完整范围
```

核心思想：**用文档（Spec 或测试用例）驱动整个交付链路**，从需求理解到最终代码落地，每一步都有明确的中间产物和验收标准。

### 为什么叫"Pre"

"Pre"指的是在任务被标记为"完成"（Completion）之前的所有工作——不是"编码之前"，而是**"交付之前"**。包括编码本身。

与之对应的 Post Task Completion 关注的是交付之后的事：E2E 验收、UI 还原、知识复利。

---

## 完整的交付链路

```text
Phase 1: 需求构建
  用户描述需求 → Agent 整理结构化需求文档 → 确认理解一致

Phase 2: 调研
  Agent 分析现有代码库 & 联网搜索 & 选型决策 → 识别影响面 → 输出调研报告

Phase 3: 技术方案
  Agent 基于调研生成技术方案 → 用户 Review → 确认方案

Phase 4: 任务拆解
  技术方案 → 拆分为可执行的 Phase/Task → 明确每步的验收标准

Phase 5: 编码/单测落地
  Agent 按 Phase 逐步实现 → 每个 Phase 编码+单测 → commit + push
```

---

## 两种驱动方式


| 方法      | 核心产物          | 驱动方式                       | 适用场景                  |
| ------- | ------------- | -------------------------- | --------------------- |
| **SDD** | Spec 文档（需求规格） | 需求 → Spec → 技术方案 → 拆解 → 编码 | 功能开发、API 设计、跨模块改动     |
| **TDD** | 测试用例          | 需求 → 测试用例 → 实现至通过 → 重构     | 算法逻辑、Bug 修复、边界处理、重构保护 |


### SDD vs TDD 的选择

| 场景 | 推荐 | 难度 |
|------|------|------|
| 小需求想留痕，不想 Vibe | **exec-plan**（1 个文档） | 入门 |
| 小中型新功能 | **OpenSpec**（~4 个文档） | 新手 |
| 中大型项目、需要强约束 | **Spec Kit**（~5 个文档） | 熟练 |
| 需求不明确，需要探索 | **superpowers /brainstorm** | 新手 |
| 持续迭代，追求复利 | **Compound Engineering** | 老手 |
| 修 Bug | **TDD（superpowers）**：先写测试复现 | 新手 |
| 重构保护 | **TDD（superpowers）**：先写测试确保行为不变 | 新手 |
| 一行配置改动 | **直接 Vibe Coding** | — |

两者不互斥——SDD 流程中的编码阶段也可以用 TDD 驱动。

---

## 子章节

- 3.1.1. [SDD（Spec-Driven Development）](031a-sdd.md)
  - 3.1.1.1. [exec-plan](031a1-exec-plan.md)
  - 3.1.1.2. [OpenSpec](031a2-openspec.md)
  - 3.1.1.3. [Spec Kit](031a3-speckit.md)
  - 3.1.1.4. [superpowers（SDD 视角）](031a4-superpowers.md)
  - 3.1.1.5. [Compound Engineering（SDD 视角）](031a5-compound.md)
- 3.1.2. [TDD（Test-Driven Development）](031b-tdd.md)
  - 3.1.2.1. [superpowers（TDD 视角）](031b1-superpowers.md)

---

## Good Case vs Bad Case

### Good Case

```text
需求：为用户模块添加批量导入功能

SDD 全流程：
  1. 需求构建：整理需求文档（导入格式、字段、校验规则）
  2. 调研：Agent 分析现有用户模块代码，输出影响面报告
  3. 技术方案：定义 API 接口、数据模型、错误处理策略 → Review
  4. 任务拆解：拆为 3 个 Phase（模型层 → API 层 → 测试）
  5. 编码落地：Agent 按 Phase 逐步实现 + 单测 → commit
```

### Bad Case

```text
需求：为用户模块添加批量导入功能

Vibe 流程：
  1. "帮我加个批量导入功能"
  2. Agent 自由发挥，用了 CSV 格式
  3. 产品说要 Excel 格式
  4. 重来一遍，这次 Agent 没加校验
  5. 测试发现空字段会崩溃
  6. 又改一轮...
```

---

[下一节：Post Task Completion](032-post-task-completion.md) | [返回上级：研发范式](030-dev-paradigm.md)