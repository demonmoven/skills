# 3.1.1.4 superpowers（SDD 视角）

> **本节目标**：理解 superpowers 在 SDD 链路中的价值——Socratic 需求精炼和 SubAgent 驱动的编排执行。
> TDD 相关内容详见 [3.1.2.1 superpowers（TDD 视角）](031b1-superpowers.md)。

---

## 概念与来源

**superpowers**（[obra/superpowers](https://github.com/obra/superpowers)，MIT 协议）由 Jesse Vincent（Prime Radiant）开发，是一个 14 Skill 的 Agent 工作流框架。

虽然 superpowers 以 TDD 闻名，但它的 **brainstorm → write-plan → execute-plan** 链路本身就是一套完整的 SDD 流程——从需求精炼到计划编写再到 SubAgent 驱动执行。

---

## Slash Command

superpowers 提供 14 个 Skills，按 SDD 相关性分组如下：

### SDD 核心流程

| 命令 | 说明 | 产物 |
|------|------|------|
| `/brainstorm` | **需求精炼入口**——Socratic 对话，反复提问帮助用户明确需求和设计决策 | `docs/superpowers/specs/YYYY-MM-DD-{topic}-design.md` |
| `/write-plan` | 基于 brainstorm 产出生成结构化实施计划 | `docs/superpowers/plans/YYYY-MM-DD-{feature}.md` |
| `/execute-plan` | 按计划逐 task 执行（非 SubAgent 平台用） | 代码变更 + commit |
| `/subagent-driven-development` | **SubAgent 并行执行**——每个 task 启动独立 SubAgent，两阶段 Review | 代码变更 + commit（每 task 一个） |

### 开发辅助

| 命令 | 说明 | 产物 |
|------|------|------|
| `/test-driven-development` | RED-GREEN-REFACTOR 循环强制（详见 [TDD 视角](031b1-superpowers.md)） | 测试文件 + 实现代码 |
| `/systematic-debugging` | 4 阶段根因分析流程 | 修复代码 + 回归测试 |
| `/verification-before-completion` | 完成前验证——确认修复确实生效 | —（对话输出验证结论） |
| `/dispatching-parallel-agents` | 并发 SubAgent 调度 | —（编排控制，无独立产物） |

### Git / Review 工作流

| 命令 | 说明 | 产物 |
|------|------|------|
| `/using-git-worktrees` | 设计确认后创建隔离 worktree 分支 | git worktree + feature branch |
| `/finishing-a-development-branch` | 验证测试通过，提供 merge/PR/keep/discard 选项 | merge / PR（视用户选择） |
| `/requesting-code-review` | 提交 review 前的自检清单 | —（对话输出自检结果） |
| `/receiving-code-review` | 响应 review 反馈的流程 | 代码修改 + commit |

### 元技能

| 命令 | 说明 | 产物 |
|------|------|------|
| `/writing-skills` | 创建新 skill 的最佳实践 | 新 `SKILL.md` 文件 |
| `/using-superpowers` | superpowers 系统介绍 | —（对话输出） |

**标准 SDD 工作流**：`/brainstorm → /using-git-worktrees → /write-plan → /subagent-driven-development → /finishing-a-development-branch`

---

## SDD 相关工作流

### Phase 1: brainstorm（需求精炼）

> **核心思想**：不是简单的"描述需求"，而是 Socratic 对话——通过反复提问帮助用户发现自己也没想清楚的问题。

```text
/brainstorm "用户搜索功能优化"
  → Agent 提问：搜索的范围是什么？是否需要模糊匹配？
  → 用户回答
  → Agent 提问：搜索结果的排序策略？是否需要分页？
  → 用户回答
  → Agent 提出替代方案：考虑过用 Elasticsearch 吗？
  → 最终产出：明确的问题定义 + 设计决策
```

### Phase 2: write-plan（计划编写）

> **核心思想**：计划的标准是"清晰到一个没有项目上下文的初级工程师也能执行"。

```text
/write-plan
  → 基于 brainstorm 产出，生成结构化计划
  → 每个 task 包含：目标、具体步骤、验收标准、测试要求
  → 任务间依赖关系明确
```

### Phase 3: execute-plan / subagent-driven-development

> **核心思想**：SubAgent 隔离执行 + 两阶段 Review（Spec 合规 → 代码质量）。

```text
/subagent-driven-development
  → 主 Agent 读取计划，按 task 启动 SubAgent
  → SubAgent A: 执行 Task 1（独立上下文）
  → SubAgent B: 执行 Task 2（独立上下文）
  → 主 Agent: 逐个 Review SubAgent 的产出（两阶段审查）
  → 每个 task 完成后 commit
```

---

## 中间产物

### Plan 文档

> **核心思想**：以日期命名、存放在固定目录、每个 task 粒度 2-5 分钟——粒度极细。

```markdown
<!-- 存放路径: docs/superpowers/plans/YYYY-MM-DD-{feature-name}.md -->

# {Feature Name} Plan

## Overview
<!-- 概要：要解决的问题和期望结果 -->

## Tasks
### Task 1: {任务名}
- **Goal**: {目标}
- **Steps**: {具体步骤}
- **Acceptance Criteria**: {验收标准}
- **Test Requirements**: {测试要求}
- **Dependencies**: {依赖的其他 task}
```

| 维度 | 要求 |
|------|------|
| 粒度 | 每个 task 2-5 分钟可完成（比其他工具更细） |
| 自包含性 | "初级工程师也能执行"——无需额外上下文 |
| 文件路径 | 每个 task 必须写明涉及的具体文件路径 |
| 测试要求 | 每个 task 必须包含测试要求（与 TDD 强绑定） |

### Design 文档

```markdown
<!-- 存放路径: docs/superpowers/specs/YYYY-MM-DD-{topic}-design.md -->

# {Topic} Design

## Problem Statement
## Design Decisions
## Trade-offs
```

---

## 与其他 SDD 工具的定位差异

| 维度 | superpowers | OpenSpec / Spec Kit |
|------|-------------|---------------------|
| 需求精炼 | Socratic 对话，强交互 | 模板化 Spec 编写 |
| 计划粒度 | 2-5 分钟/task（极细） | 2-4 小时/task |
| 执行方式 | SubAgent 并行 + 两阶段 Review | Agent 串行执行 |
| 适用场景 | 需求不明确时的探索 + 编码 | 需求已明确的标准化交付 |

### 难度分层

superpowers 的难度取决于使用深度：
- **新手**：只用 `/brainstorm` 做需求精炼——零门槛，Socratic 对话自然引导
- **老手**：全流程 brainstorm → plan → execute → review → debug——需要理解 SubAgent 编排和 TDD 强制机制

新手建议从 brainstorm 开始，搭配 exec-plan 或 OpenSpec 完成后续流程；老手可以用 superpowers 全流程替代。

---

[上一节：Spec Kit](031a3-speckit.md) | [下一节：Compound Engineering（SDD 视角）→](031a5-compound.md) | [返回上级：SDD](031a-sdd.md)
