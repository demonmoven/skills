# 3. 研发范式

> **本章目标**：理解 Agent 时代的两大研发范式——从需求到代码落地的完整交付（Pre Task Completion）和交付后的质量收敛与知识复利（Post Task Completion）。

---

## 概述

在 Agent 时代，"写代码"不再是瓶颈，**如何驱动完整的交付过程**和**确保交付后的质量闭环**才是关键。

研发范式分为两个阶段：

| 阶段 | 定位 | 子阶段 | 说明 | 代表工具 |
|------|------|--------|------|----------|
| **Pre Task Completion** | 从需求到代码落地的完整交付 | | 需求构建 → 调研 → 技术方案 → 任务拆解 → 编码/单测落地 | |
| | | └─ SDD (Spec-Driven) | 规格驱动的全流程交付 | exec-plan / speckit / openspec |
| | | └─ TDD (Test-Driven) | 测试驱动的全流程交付 | superpowers |
| **Post Task Completion** | 交付后的质量收敛与知识复利 | | 代码已落地，进入验收和沉淀阶段 | |
| | | └─ E2E | 后/前端功能验证 + 代码质量门禁 | fixloop / e2e-fe / vv |
| | | └─ UI 还原 | 设计稿与实现的视觉一致性 | x-ui-workflow |
| | | └─ Compound 复利 | 知识归档，每次迭代更快 | Compound Engineering Plugin |

---

## 为什么需要研发范式

### Vibe Coding 的局限

Vibe Coding（即兴编程）适合快速原型和小任务，但在以下场景容易失控：

| 场景 | Vibe Coding 的问题 |
|------|-------------------|
| 大型功能开发 | Agent 缺乏全局视角，容易改偏 |
| 多人协作 | 没有共享的需求文档，理解不一致 |
| 质量要求高 | 没有验收标准，"能跑就行" |
| 需要可追溯 | 没有留下设计决策记录 |

### 文档驱动的价值

```text
Vibe:   "帮我加个登录功能" → Agent 自由发挥 → 结果不可预测
SDD:    需求 → Spec → 技术方案 → 任务拆解 → Agent 编码 → 结果可验证、可追溯
TDD:    需求 → 测试用例 → Agent 实现 → 测试通过 → 结果自动验收
```

---

## 子章节

- 3.1. [Pre Task Completion](031-pre-task-completion.md)
  - 3.1.1. [SDD（Spec-Driven Development）](031a-sdd.md)
    - 3.1.1.1. [exec-plan](031a1-exec-plan.md)
    - 3.1.1.2. [speckit](031a2-speckit.md)
    - 3.1.1.3. [openspec](031a3-openspec.md)
  - 3.1.2. [TDD（Test-Driven Development）](031b-tdd.md)
    - 3.1.2.1. [superpowers](031b1-superpowers.md)
- 3.2. [Post Task Completion](032-post-task-completion.md)
  - 3.2.1. [E2E](032a-e2e.md)
  - 3.2.2. [UI 还原](032b-ui.md)
  - 3.2.3. [Compound（复利工程）](032c-compound.md)

---

[上一章：Agent 技术栈](020-agent-tech-stack.md) | [下一章：观测 & 评测](040-observation-evaluation.md) | [返回目录](000-index.md)
