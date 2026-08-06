# 3.1.2 TDD（Test-Driven Development）

> **本节目标**：理解 Agent 时代 TDD 的新价值和工作流。

---

## 概念与历史

### 传统 TDD

TDD（Test-Driven Development）由 Kent Beck 在 2003 年系统化提出，核心循环：

```text
    ┌──────────────┐     ┌──────────────────────┐     ┌──────────────┐
    │  Red         │     │  Green               │     │  Refactor    │
    │  写失败测试    │ ──→ │  写最少代码使测试通过    │ ──→ │  重构代码     │
    └──────────────┘     └──────────────────────┘     └───────┬──────┘
           ↑                                                  │
           └──────────────────────────────────────────────────┘
```

### Agent 时代的 TDD

在 Agent 时代，TDD 获得了新的价值：**测试用例 = Agent 的验收标准**，且 TDD 同样覆盖完整交付链路。

```text
传统 TDD:  人写测试 → 人写实现 → 人重构
Agent TDD: 需求 → 测试设计 → Agent 实现至通过 → Agent 重构 → 验收
```

Agent 的优势：
- Agent 不会觉得"写测试麻烦"——写测试和写实现对它来说成本一样
- Agent 可以快速 Red-Green 循环直到全部通过
- 测试为 Agent 提供了**机械化的成功标准**——不再依赖人判断"做没做完"

### 与 SDD 的互补关系

| 维度 | SDD | TDD |
|------|-----|-----|
| 驱动物 | Spec 文档 | 测试用例 |
| 验收方式 | 人工对照 Spec | 自动运行测试 |
| Agent 自主性 | 中（需要人 Review Spec） | 高（测试自动判断对错） |
| 适合场景 | 功能开发、API 设计、跨模块 | 算法、Bug 修复、边界处理、重构 |

两者不互斥——SDD 流程中的编码阶段也可以用 TDD 驱动。

---

## 业界工具

| 工具 | 定位 | 核心特性 |
|------|------|----------|
| **[superpowers](031b1-superpowers.md)** (obra) | Agent TDD 标杆框架 | Mandatory TDD + SubAgent 驱动 + 严格 Red-Green-Refactor |

---

## 子章节

- 3.1.2.1. [superpowers（TDD 视角）](031b1-superpowers.md)

---

## 选型：何时用 TDD vs SDD

```text
"我要做一个新功能"         → SDD（speckit / openspec / gstack）
"我要修一个 Bug"           → TDD（superpowers）：先写测试复现，再修复
"我要重构一段代码"         → TDD（superpowers）：先写测试保护，再重构
"我要做性能优化"           → TDD：先写性能基准测试，再优化
"新功能中的核心算法"       → SDD + TDD 组合：SDD 定义接口，TDD 驱动实现
```

---

[上一节：SDD](031a-sdd.md) | [返回上级：Pre Task Completion](031-pre-task-completion.md)
