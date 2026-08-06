# 3.1.2.1 superpowers（TDD 视角）

> **本节目标**：理解 superpowers 在 TDD 链路中的价值——严格 Red-Green-Refactor + SubAgent 驱动执行。
> SDD 相关内容详见 [3.1.1.4 superpowers（SDD 视角）](031a4-superpowers.md)。

---

## 概念与来源

**superpowers**（[obra/superpowers](https://github.com/obra/superpowers)，MIT 协议）由 Jesse Vincent（Prime Radiant）开发，是一个 14 Skill 的 Agent 工作流框架。

核心理念：**Mandatory workflows, not suggestions——TDD 不是建议，而是 Agent 的硬约束**。Agent 在 superpowers 框架下必须严格遵循 Red-Green-Refactor 循环，不可跳过任何步骤。

### Agent TDD vs 传统 TDD

| 步骤 | 传统 TDD | superpowers Agent TDD |
|------|----------|----------------------|
| 理解需求 | 人脑理解 | Agent 先读代码、文档、已有测试 |
| 写测试 | 人手写 | Agent 生成（覆盖 happy path + 边界 + 异常） |
| 实现 | 人写 | Agent 写，自动运行测试验证 |
| 重构 | 人判断 | Agent 重构，测试保护 |
| 文档 | 经常忽略 | 强制更新（规范要求） |

---

## Slash Command

superpowers 的 14 个 Skills 中，以下与 TDD 直接相关：

### TDD 核心

| 命令 | 说明 | 产物 |
|------|------|------|
| `/test-driven-development` | **TDD 主入口**——强制 Red-Green-Refactor 循环 | 测试文件 + 实现代码 + commit |
| `/systematic-debugging` | 4 阶段根因分析：复现 → 假设 → 验证 → 修复 | 修复代码 + 回归测试 |
| `/verification-before-completion` | 完成前验证——确认修复确实生效，无回归 | —（对话输出验证结论） |

### 执行编排

| 命令 | 说明 | 产物 |
|------|------|------|
| `/subagent-driven-development` | SubAgent 并行执行——每个 task 独立 TDD 循环 + 两阶段 Review | 代码变更 + commit（每 task 一个） |
| `/execute-plan` | 按计划逐 task 执行（非 SubAgent 平台用） | 代码变更 + commit |
| `/dispatching-parallel-agents` | 并发 SubAgent 调度 | —（编排控制，无独立产物） |

### SDD 相关（详见 [SDD 视角](031a4-superpowers.md)）

| 命令 | 说明 | 产物 |
|------|------|------|
| `/brainstorm` | Socratic 需求精炼 | `docs/superpowers/specs/YYYY-MM-DD-{topic}-design.md` |
| `/write-plan` | 结构化计划编写 | `docs/superpowers/plans/YYYY-MM-DD-{feature}.md` |
| `/using-git-worktrees` | Worktree 隔离开发 | git worktree + feature branch |
| `/finishing-a-development-branch` | 分支收尾 | merge / PR（视用户选择） |
| `/requesting-code-review` | 提交 review 前自检 | —（对话输出自检结果） |
| `/receiving-code-review` | 响应 review 反馈 | 代码修改 + commit |
| `/writing-skills` | 创建新 skill | 新 `SKILL.md` 文件 |
| `/using-superpowers` | 系统介绍 | —（对话输出） |

**TDD 标准工作流**：`/brainstorm → /write-plan → /test-driven-development → /verification-before-completion`

**SubAgent TDD 工作流**：`/write-plan → /subagent-driven-development`（每个 SubAgent 内部自动执行 TDD 循环）

---

## TDD 工作流

### Red-Green-Refactor 循环

> **核心思想**：严格强制——Agent 必须先看到测试失败（Red），再写最少代码使之通过（Green），最后重构。每一步都不可跳过。

**前置依赖**：需要先完成 SDD 前序流程，确保 Agent 知道"测什么"：
- `/brainstorm` 产出的**问题定义和设计决策**——决定测试的范围和边界
- `/write-plan` 产出的 **Plan 文档**（含 Test Requirements 字段）——每个 task 明确要覆盖的测试场景

```text
前置：/brainstorm → /write-plan（产出 Plan 文档，每个 task 含 Test Requirements）
                        ↓
/test-driven-development（按 Plan 中的 task 逐个执行）

Step 1: Understand
  → 读取 Plan 中当前 task 的 Goal + Test Requirements
  → 深度理解需求和现有代码

Step 2: Red（写失败测试）
  → 按 Test Requirements 写测试用例（happy path + 边界条件 + 异常路径）
  → 运行测试 → 确认失败（RED）

Step 3: Green（最少实现）
  → 写最少的代码使测试通过
  → 运行测试 → 确认通过（GREEN）

Step 4: Refactor（重构）
  → 重构代码，保持测试通过
  → 运行测试 → 确认仍然通过

Step 5: Commit
  → 提交代码
```

### SubAgent 驱动的 TDD

> **核心思想**：每个 task 由独立 SubAgent 执行 TDD 循环，主 Agent 做两阶段 Review（Spec 合规 → 代码质量）。

**前置依赖**：比单人 TDD 更严格——SubAgent 没有对话上下文，完全依赖 Plan 文档驱动：
- `/brainstorm` 产出的**问题定义和设计决策**
- `/write-plan` 产出的 **Plan 文档**——SubAgent 唯一的输入，必须自包含到"初级工程师也能执行"
- （可选）`/using-git-worktrees` 创建的**隔离 worktree**——SubAgent 在独立分支上工作，避免互相干扰

```text
前置：/brainstorm → /write-plan → (可选) /using-git-worktrees
                                      ↓
/subagent-driven-development

主 Agent 读取 Plan 文档
  → 为每个 task 启动独立 SubAgent（传入 task 的 Goal + Steps + Test Requirements）
  → SubAgent 内部执行完整 TDD 循环（Red → Green → Refactor）
  → SubAgent 完成后，主 Agent 两阶段审查：
      Stage 1: Spec 合规性（对照 Plan 中的 Acceptance Criteria）
      Stage 2: 代码质量（写的好不好）
  → 审查通过 → commit
  → 审查不通过 → SubAgent 重做
```

### 系统化调试

> **核心思想**：不猜不试——4 阶段结构化根因分析，每一步有明确的输入输出。

```text
/systematic-debugging

Phase 1: 复现
  → 稳定复现问题，确认不是偶发

Phase 2: 假设
  → 基于证据提出根因假设（最多 3 个）

Phase 3: 验证
  → 逐个验证假设，排除到唯一根因

Phase 4: 修复
  → 修复 + 写回归测试 + 确认无副作用
```

---

## 中间产物

### 测试覆盖策略

> **核心思想**：TDD 的测试不是"有就行"，而是必须覆盖四个维度——happy path、边界条件、异常路径、回归保护。

| 维度 | 说明 | 示例 |
|------|------|------|
| Happy Path | 正常输入正常输出 | 上传 1MB jpg → 返回 URL |
| 边界条件 | 空值、极值、临界值 | 上传 0 字节文件 / 恰好 5MB 文件 |
| 异常路径 | 无效输入、网络错误、超时 | 上传非图片文件 → 返回 400 |
| 回归保护 | 确保现有功能不受影响 | 修改上传逻辑后，下载仍正常 |

### Plan 文档（TDD 增强）

> **核心思想**：superpowers 的 Plan 中每个 task 必须包含测试要求——这是与纯 SDD Plan 的关键差异。

```markdown
<!-- 存放路径: docs/superpowers/plans/YYYY-MM-DD-{feature-name}.md -->

# {Feature Name} Plan

## Overview
<!-- 概要：要解决的问题和期望结果 -->

## Tasks
### Task 1: {任务名}
- **Goal**: {目标}
- **Steps**: {具体步骤}
- **Test Requirements**: {必须覆盖的测试场景——happy path / 边界 / 异常}
- **Acceptance Criteria**: {验收标准}
- **Dependencies**: {依赖的其他 task}
```

| 维度 | 要求 |
|------|------|
| 粒度 | 每个 task 2-5 分钟可完成（比其他工具更细） |
| 测试先行 | 每个 task 必须包含 Test Requirements 字段 |
| 自包含性 | "初级工程师也能执行"——无需额外上下文 |
| 文件路径 | 每个 task 必须写明涉及的具体文件路径 |

---

## 适用场景

| 场景 | 推荐度 | 说明 |
|------|--------|------|
| Bug 修复 | 强烈推荐 | 先写复现测试，再修复，确保不再复发 |
| 核心业务逻辑 | 强烈推荐 | TDD 提供机械化验收标准 |
| 重构 | 推荐 | 先写测试保护，再重构 |
| 工具函数/库函数 | 推荐 | 输入输出明确，天然适合 TDD |
| 快速原型 | 不推荐 | 用 Vibe Coding 更快 |
| UI 组件 | 不推荐 | 用 E2E / 视觉测试更合适 |

---

## 局限

- 专注 TDD，对需求分析和技术方案阶段覆盖较弱（这些由 SDD 工具补充）
- SubAgent 链过长时可能出现上下文衰减

---

[返回上级：TDD](031b-tdd.md)
