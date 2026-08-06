# 2.1.3 SubAgent（上下文隔离视角）

> **本节目标**：理解 SubAgent 如何通过上下文隔离来保护主对话的上下文窗口。

---

## 概念与历史

### 什么是 SubAgent

**SubAgent** 是主 Agent 启动的子 Agent 实例。每个 SubAgent 拥有**独立的上下文窗口**，完成任务后只向主 Agent 返回精简的结果。

SubAgent 在技术栈中有两重身份：
- **上下文控制工具**（本节）：通过隔离保护主上下文不被污染
- **流程控制工具**（[2.5.1](025a-subagent-flow.md)）：通过并行/串行编排实现任务分解

### 核心价值：上下文隔离

```text
Without SubAgent:
  主 Agent 上下文窗口: [任务A的大量中间数据 + 任务B的大量中间数据 + ...]
  → 上下文溢出，遗忘早期约束

With SubAgent:
  主 Agent 上下文窗口: [任务A的结论(50行) + 任务B的结论(30行)]
  SubAgent-A 上下文: [任务A的全部细节] → 返回精简结论后释放
  SubAgent-B 上下文: [任务B的全部细节] → 返回精简结论后释放
```

---

## 在 Claude Code 中的实现

Claude Code 通过 `Agent` 工具创建 SubAgent，支持多种 agent 类型：

| Agent 类型 | 用途 | 工具权限 |
|-----------|------|----------|
| `general-purpose` | 通用任务 | 全部工具 |
| `Explore` | 代码库探索和搜索 | 只读工具（Glob/Grep/Read 等） |
| `Plan` | 架构设计和实施规划 | 只读工具 |
| `claude-code-guide` | Claude Code 使用指南查询 | 只读 + WebFetch |

### 关键特性

- **独立上下文**：SubAgent 看不到主对话的历史消息
- **结果精简**：SubAgent 完成后返回单条消息，大量中间过程不进入主上下文
- **可并行**：多个独立 SubAgent 可同时执行
- **Worktree 隔离**：`isolation: "worktree"` 让 SubAgent 在独立的 git worktree 中工作

---

## 最佳实践

### 何时使用 SubAgent 做上下文隔离

1. **探索性搜索**：需要翻遍大量文件才能找到答案
2. **独立调研**：结果独立于其他任务，只需最终结论
3. **大输出处理**：工具返回大量数据，需要先过滤摘要

### 如何写好 SubAgent 的 prompt

```markdown
Bad:  "帮我看看这个 bug"
Good: "搜索 /src/auth/ 目录下所有包含 session.expired 的文件，
       确认 token 刷新逻辑是否在 middleware 还是 handler 中实现。
       报告：文件路径、行号、实现位置。200字以内。"
```

关键：**像给刚进入团队的同事写便条**——说清楚目标、范围、输出格式。

---

## Good Case vs Bad Case

### Good Case

- 用 `Explore` SubAgent 搜索代码库中所有使用某个 API 的位置，只返回文件列表
- 用 `Plan` SubAgent 独立设计方案，主 Agent 拿到方案后决策

### Bad Case

| 错误 | 后果 |
|------|------|
| 对简单的单文件搜索也用 SubAgent | 徒增延迟，直接用 Grep 更快 |
| SubAgent prompt 过于模糊 | SubAgent 做了大量无用功，返回低质量结果 |
| 让 SubAgent 做决策（"帮我 fix 这个 bug"） | 不带上下文就让 SubAgent 做判断，容易出错 |

---

[上一节：MCP vs CLI](021b-mcp-vs-cli.md) | [下一节：Slash Command vs Skill →](021d-slash-command-vs-skill.md) | [返回上级：上下文控制](021-context-control.md)
