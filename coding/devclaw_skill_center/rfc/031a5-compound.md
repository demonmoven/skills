# 3.1.1.5 Compound Engineering（SDD 视角）

> **本节目标**：理解 Compound Engineering 的 plan → work → review 循环如何作为一种 SDD 范式运作。
> 复利沉淀（compound 步骤）的详细内容见 [3.2.3 Compound（复利工程）](032c-compound.md)。

---

## 概念与来源

**Compound Engineering**（[EveryInc/compound-engineering-plugin](https://github.com/EveryInc/compound-engineering-plugin)，当前版本 cli-v2.66.1）由 Every.to 提出，核心理念是"每次迭代让下次更快"。

从 SDD 视角看，Compound Engineering 的 **plan → work → review** 前三步本身就是一套 SDD 流程——只是它更强调 **80% 精力在 Plan 和 Review，20% 在执行**，以及每次循环的知识回流。

---

## Slash Command

### 核心工作流（6 个阶段）

| 命令 | 精力占比 | 说明 | 产物 |
|------|----------|------|------|
| `/ce:ideate` | — | **发散探索**——发散式构想 + 对抗性过滤 | —（对话输出候选改进点） |
| `/ce:brainstorm` | — | **需求精炼**——协作对话探索需求 | `docs/brainstorms/{name}-requirements.md` |
| `/ce:plan` | ~40% | **计划编写**——含 confidence check + 深化 | `docs/plans/YYYY-MM-DD-NNN-{type}-{name}-plan.md` |
| `/ce:work` | ~20% | **编码执行**——worktree + SubAgent 调度 | 代码变更 + commit |
| `/ce:review` | ~30% | **多角色审查**——17 人格 + confidence 门控 | —（对话输出 / autofix 代码修改） |
| `/ce:compound` | ~10% | **知识沉淀**——文档化已解决的问题 | `docs/solutions/{category}/{slug}-{date}.md` |

### 一键全自动

| 命令 | 说明 | 产物 |
|------|------|------|
| `/lfg` | **全自动管线**——plan → work → review → compound 一键串联 | 同上述各阶段产物之和 |

### 辅助命令

| 命令 | 说明 |
|------|------|
| `/ce-debug` | 系统化根因调试 |
| `/ce-setup` | 环境诊断与配置 |
| `/ce-sessions` | 搜索历史 coding agent session |
| `/ce-update` | 检查/修复 plugin 版本 |
| `/ce-optimize` | 指标驱动的迭代优化循环 |
| `/ce-pr-description` | 生成价值导向的 PR 描述 |
| `/ce-slack-research` | 搜索 Slack 获取组织上下文 |
| `/git-commit` | 精心编写的 git commit |
| `/git-commit-push-pr` | commit + push + 开 PR 一步到位 |
| `/git-worktree` | Git worktree 并行开发管理 |
| `/resolve-pr-feedback` | 处理 PR review 反馈 |
| `/todo-create` | 创建持久化工作项 |
| `/todo-resolve` | 批量完成已审批的 todo |
| `/document-review` | 通过人格 Agent 审查需求/计划文档 |
| `/onboarding` | 为新贡献者生成 ONBOARDING.md |

**标准工作流**：`/ce:brainstorm → /ce:plan → /ce:work → /ce:review → /ce:compound`
**快速模式**：`/lfg`（全自动）

---

## SDD 相关工作流

### Plan（~40% 精力）

> **核心思想**：Plan 特别强调**上下文准备**——不只是"写清楚要做什么"，还要"准备好 Agent 工作需要的所有信息"。含 confidence check：Agent 评估自己对计划的信心水平，不足则深化。

```text
/ce:plan
  → 明确本次迭代的目标和范围
  → 整理 Agent 需要的上下文（哪些文档、哪些代码、哪些约束）
  → Confidence Check：Agent 评估信心 → 不足则自动深化
  → 输出：结构化的任务定义 + 上下文准备
```

### Work（~20% 精力）

> **核心思想**：如果 Plan 做得好，Work 几乎是自动化的。支持 worktree 隔离和三种 SubAgent 调度模式（inline/serial/parallel）。

```text
/ce:work
  → Agent 在 worktree 中执行编码（人类监督和引导）
  → SubAgent 调度模式：inline（同进程）/ serial（串行）/ parallel（并行）
  → 每个 task 完成后自动运行系统级测试检查
```

### Review（~30% 精力）

> **核心思想**：不是简单 code review，而是 17 个 reviewer 人格的多角色审查，含 confidence 门控——低信心发现自动过滤。支持 interactive / autofix / report-only / headless 四种模式。

```text
/ce:review
  → 17 个 reviewer 人格分别审查（安全、性能、可读性、架构...）
  → Confidence 门控：每个发现附带信心评分，低信心自动过滤
  → Merge/Dedup 管线：合并重复发现
  → 模式选择：interactive（交互修复）/ autofix（自动修复）/ report-only / headless
```

---

## 中间产物

### Brainstorm 产出：需求文档

> **核心思想**：协作对话的结构化输出——从发散探索收敛到可执行的需求定义。

```markdown
<!-- 存放路径: docs/brainstorms/{name}-requirements.md -->

# {Feature Name} Requirements

## Problem Statement
<!-- 概要：要解决的问题 -->

## Requirements
<!-- 概要：功能和非功能需求 -->

## Constraints
<!-- 概要：技术和业务约束 -->
```

### Plan 产出：实施计划

> **核心思想**：计划不只是 task 列表，还包含 Requirements Trace（需求追溯）、Scope Boundaries（范围边界）、Key Technical Decisions（技术决策）——确保 Plan 和需求的可追溯性。

```markdown
<!-- 存放路径: docs/plans/YYYY-MM-DD-NNN-{type}-{name}-plan.md -->
---
title: {Plan Title}
type: feature | bugfix | refactor | exploration
status: draft | approved | in-progress | completed
date: YYYY-MM-DD
origin: {brainstorm 来源}
---

# {Plan Title}

## Overview
<!-- 概要：目标和背景 -->

## Problem Frame
<!-- 概要：问题定义和边界 -->

## Requirements Trace
<!-- 概要：追溯到 brainstorm 需求文档 -->

## Scope Boundaries
### In Scope
### Out of Scope

## Key Technical Decisions
<!-- 概要：技术选型和理由 -->

## Implementation Units
### Unit 1: {单元名}
- **Goal**: {目标}
- **Requirements**: {关联需求}
- **Files**: {涉及文件}
- **Approach**: {实现方式}
- **Test Scenarios**: {测试场景}
- **Verification**: {验证方式}

## Risks & Dependencies
<!-- 概要：风险和外部依赖 -->
```

| 维度 | 要求 |
|------|------|
| YAML Frontmatter | 必须包含 title/type/status/date/origin |
| Requirements Trace | 每个 Implementation Unit 必须追溯到需求 |
| Scope Boundaries | 明确 In Scope / Out of Scope |
| Confidence Check | Plan 完成后 Agent 自评信心，不足则深化 |

### Compound 产出：知识沉淀

> **核心思想**：将解决过的问题结构化沉淀，让下次遇到类似问题时更快——这是"复利效应"的载体。

```markdown
<!-- 存放路径: docs/solutions/{category}/{slug}-{date}.md -->
---
title: {解决方案标题}
category: {分类}
date: YYYY-MM-DD
---

# {Solution Title}

## Problem
<!-- 概要：遇到的问题 -->

## Solution
<!-- 概要：如何解决 -->

## Key Insight
<!-- 概要：可复用的洞察 -->
```

---

## 与其他 SDD 工具的定位差异

| 维度 | Compound Engineering | exec-plan（轻量） | OpenSpec / Spec Kit（完整） |
|------|---------------------|-----------------|--------------------------|
| 精力分配 | 80% Plan+Review, 20% Work | Plan 轻量，执行为主 | 按阶段均匀 |
| 迭代模式 | 短循环快速迭代 | 单次轻量交付 | 完整流程一次走完 |
| 上下文管理 | Plan 阶段特别强调上下文准备 | plan.md 自包含 | 多文档体系 |
| Review 深度 | 17 人格多角色审查 | 无 Review 阶段 | 单一 Review |
| 核心价值 | 每次迭代更快（复利效应） | Vibe Coding 的可追溯替代 | 单次交付的质量和可追溯性 |
| 适用场景 | 持续迭代、探索性开发 | 明确需求的一次性交付 | 标准化工程交付 |

Compound Engineering 适合**持续迭代**场景——每个 plan/work/review 循环不追求一步到位，而是追求每次比上次快。

---

[上一节：superpowers（SDD 视角）](031a4-superpowers.md) | [返回上级：SDD](031a-sdd.md)
