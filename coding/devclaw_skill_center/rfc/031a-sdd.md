# 3.1.1 SDD（Spec-Driven Development）

> **本节目标**：理解 SDD 的核心理念——Spec as Code，以及业界主流工具的对比和选型。

---

## 概念：Spec as Code

**SDD（Spec-Driven Development，规格驱动开发）** 的核心范式转变是：

> **人的精力从"写代码"转移到"写 Spec"——Spec 成为一等公民，代码是 Spec 的派生物。**

```text
传统模式:  人写代码 → 代码是一等公民 → Spec 是可选的文档
SDD 模式:  人写 Spec → Spec 是一等公民 → 代码是 Agent 按 Spec 派生的产物
```

Spec 不仅是"需求描述"，而是一份**版本化的、可验收的交付契约**：

- **要做什么**：功能描述、用户故事
- **接口是什么**：API 定义、数据模型、字段类型
- **怎么验收**：量化的验收标准（测试通过、覆盖率、lint 零告警）
- **不做什么**：明确的 Out of Scope

Spec as Code 意味着 Spec 以 Markdown 文件存储在仓库中，和代码一起进入 Git 版本管理——而非散落在飞书文档、Confluence 或 Slack 讨论中。

---

## SDD 全流程（最细粒度）

一个完整的 SDD 交付链路包含以下阶段：

```text
Phase 1: 需求构建 (Propose)
  用户描述需求 → Agent 整理结构化需求文档 → 确认理解一致

Phase 2: 调研/探索 (Explore)
  分析代码库 → 联网搜索 → 技术选型 → 影响面评估 → 输出调研报告

Phase 3: Spec 编写 (Specify)
  生成规格文档（接口、模型、验收标准）→ 用户 Review → 确认

Phase 4: Spec 审阅 (Review Spec)
  HITL 交互式审阅：检查完整性、一致性、可行性

Phase 5: 技术方案 (Design)
  基于 Spec 设计架构、数据流、模块划分 → 用户 Review

Phase 6: 技术方案审阅 (Review Design)
  多角色审查：工程视角、设计视角、产品视角

Phase 7: 任务拆解 (Plan / Task Split)
  拆分为可执行的 Phase/Task → 明确每步的验收标准

Phase 8: 编码/测试落地 (Implement)
  Agent 按 Phase 逐步实现 + 单测 → commit + push

Phase 9: 验收 (Verify)
  逐条核对 Spec 验收标准 → 标记完成

Phase 10: 归档 (Archive)
  归档已完成的 Spec 和交付物，沉淀到项目知识库
```

### 各工具的阶段覆盖

不同工具对上述 10 个阶段做了不同的取舍——有的合并了阶段，有的拆得更细：


| 阶段 | exec-plan | OpenSpec | Spec Kit | superpowers | Compound |
|------|-----------|----------|----------|-------------|----------|
| 1. 需求构建 | plan.md 中合并 | propose | — | brainstorm | ideate → brainstorm |
| 2. 调研探索 | — (轻量跳过) | explore | — | brainstorm | — |
| 3. Spec 编写 | plan.md 中合并 | propose | specify | write-plan | — |
| 4. Spec 审阅 | — (轻量跳过) | — | review | — | review |
| 5. 技术方案 | plan.md 中合并 | design.md | plan | write-plan | — |
| 6. 方案审阅 | — (轻量跳过) | — | review | — | review |
| 7. 任务拆解 | milestone 即任务 | tasks.md | tasks | write-plan | — |
| 8. 编码落地 | Agent 自主 | apply | implement | execute-plan | work |
| 9. 验收 | pre-push 守护 | verify | — | verification | review |
| 10. 归档 | 自动归档 | archive | — | finish | compound |


---

## 业界主流工具对比


| 工具 | Spec 文档数 | 难度 | 设计哲学 | 核心工作流 | 选型指南 |
|------|-----------|------|----------|----------|----------|
| **[exec-plan](031a1-exec-plan.md)** (OpenAI) | **1** (plan.md) | 入门 | Lightweight SDD：单文档、短流程，Vibe Coding / Plan Mode 的工程化替代 | 需求 → plan.md → milestone 执行 → 验收 | 敏捷小需求（≤1 人天），Vibe Coding 的可追溯替代 |
| **[OpenSpec](031a2-openspec.md)** (Fission-AI) | **~4** (proposal + specs + design + tasks) | 新手 | Artifact-centric：Spec 是一等公民；Delta Spec 支持 Brownfield | propose → explore → apply → verify → archive | 小中型项目，标准化 SDD，20+ Agent 兼容 |
| **[Spec Kit](031a3-speckit.md)** (GitHub) | **~5** (constitution + spec + plan + tasks + impl) | 熟练 | Constitution-driven：先定义项目宪法，再驱动全流程；文档更细致完整 | constitution → specify → plan → tasks → implement | 中大型项目，需要强约束和完整文档链 |
| **[superpowers](031a4-superpowers.md)** (obra) | **~2** (brainstorm 产出 + plan) | 新手(仅 brainstorm) / 老手(全流程) | Process-centric：Socratic 需求精炼 + SubAgent 驱动执行 | brainstorm → write-plan → execute-plan → review → debug | 需求不明确时的探索精炼 + TDD 编码 |
| **[Compound](031a5-compound.md)** (Every.to) | **~3** (brainstorm 需求 + plan + solutions) | 老手 | Feedback-loop：80% Plan+Review, 20% Work，每次迭代更快 | brainstorm → plan → work → review → compound | 持续迭代，追求复利效应（Week 8 达 4x） |


---

## 子章节

- 3.1.1.1. [exec-plan](031a1-exec-plan.md)
- 3.1.1.2. [OpenSpec](031a2-openspec.md)
- 3.1.1.3. [Spec Kit](031a3-speckit.md)
- 3.1.1.4. [superpowers（SDD 视角）](031a4-superpowers.md)
- 3.1.1.5. [Compound Engineering（SDD 视角）](031a5-compound.md)

---

[下一节：TDD →](031b-tdd.md) | [返回上级：Pre Task Completion](031-pre-task-completion.md)