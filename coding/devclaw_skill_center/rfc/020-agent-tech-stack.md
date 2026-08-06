# 2. Agent 技术栈

> **本章目标**：理解 AI Coding Agent 的核心技术栈——上下文怎么管、流程怎么控、能力怎么扩展。

---

## 概述

Agent 时代的工程师不再只写业务代码，还需要理解和掌握 Agent 的技术栈。这个技术栈决定了 Agent 能看到什么（上下文）、怎么干活（流程）、以及如何扩展能力（命名空间/包管理）。

```text
┌─────────────────────────────────────────┐
│            Agent 技术栈全景              │
│                                         │
│  ┌───────────────────────────────────┐  │
│  │  2.1 上下文控制                    │  │
│  │  "Agent 能看到什么"                │  │
│  │  AGENTS.md · MCP · SubAgent ·     │  │
│  │  Slash Command / Skill            │  │
│  └───────────────────────────────────┘  │
│                                         │
│  ┌───────────────────────────────────┐  │
│  │  2.5 流程控制                      │  │
│  │  "Agent 怎么干活"                  │  │
│  │  SubAgent 编排 · Hooks 回调        │  │
│  └───────────────────────────────────┘  │
│                                         │
│  ┌───────────────────────────────────┐  │
│  │  2.6 命名空间 / 包管理             │  │
│  │  "Agent 能力怎么扩展"              │  │
│  │  Plugin · Marketplace              │  │
│  └───────────────────────────────────┘  │
└─────────────────────────────────────────┘
```

---

## 子章节

- 2.1. [上下文控制](021-context-control.md)
  - 2.1.1. [Agents 文档体系](021a-agents-docs.md)
  - 2.1.2. [MCP vs CLI](021b-mcp-vs-cli.md)
  - 2.1.3. [SubAgent](021c-subagent.md)
  - 2.1.4. [Slash Command vs Skill](021d-slash-command-vs-skill.md)
- 2.5. [流程控制](025-flow-control.md)
  - 2.5.1. [SubAgent（流程编排视角）](025a-subagent-flow.md)
  - 2.5.2. [Hooks](025b-hooks.md)
- 2.6. [命名空间 / 包管理](026-namespace-pkg.md)
  - 2.6.1. [Plugin / Marketplace](026a-plugin-marketplace.md)

---

## 为什么要理解 Agent 技术栈

在 Agent 时代，工程效率的瓶颈已经从"写代码的速度"转移到"给 Agent 提供正确上下文的能力"。一个不了解技术栈的工程师可能会：

- 把所有信息堆进一个大 prompt，导致上下文溢出
- 不知道用 SubAgent 并行化任务，白白等待串行执行
- 手动重复操作，而不是封装成 Skill 复用
- 忽略 Hooks 的存在，让 Agent 提交低质量代码

理解技术栈 = 理解 Agent 的"扩展框架"。

---

[上一章：AI Coding 代际演进](010-ai-coding-evolution.md) | [下一章：研发范式](030-dev-paradigm.md) | [返回目录](000-index.md)