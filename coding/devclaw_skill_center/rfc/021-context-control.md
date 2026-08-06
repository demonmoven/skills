# 2.1 上下文控制

> **本节目标**：理解"上下文"为什么是 Agent 时代的第一公民，掌握管理上下文的四大工具。

---

## 概念

**上下文（Context）** 是 Agent 的"工作记忆"。模型的推理能力再强，看不到正确的信息就无法产出正确的结果。

在 AI Native 的类比中：**Context = RAM**。上下文窗口是有限的，如何在有限的窗口中放入最相关的信息，是 Agent 时代工程师的核心技能。

### 上下文的三个层次

| 层次 | 说明 | 管理工具 |
|------|------|----------|
| **持久上下文** | 项目级别，始终存在 | AGENTS.md / CLAUDE.md / ARCHITECTURE.md |
| **按需上下文** | 任务触发时加载 | Slash Command / Skill / MCP |
| **临时上下文** | 单次对话内产生 | SubAgent 返回结果、工具调用输出 |

### 为什么上下文控制是第一优先级

```text
上下文质量 ────→ Agent 输出质量
                    │
上下文过少 ────→ Agent 猜测、幻觉
上下文过多 ────→ Agent 遗忘、失焦
上下文错误 ────→ Agent 信心满满地做错
```

核心原则：**精准投喂，按需披露**（Progressive Disclosure）。

---

## 子章节

- 2.1.1. [Agents 文档体系](021a-agents-docs.md)
- 2.1.2. [MCP vs CLI](021b-mcp-vs-cli.md)
- 2.1.3. [SubAgent](021c-subagent.md)
- 2.1.4. [Slash Command vs Skill](021d-slash-command-vs-skill.md)

---

## Good Case vs Bad Case

### Good Case：渐进式披露

```
Agent 进入仓库
  → 读 AGENTS.md（~100行，知道项目结构和约束）
    → 按任务需要读 backend/AGENTS.md（模块级细节）
      → 按需调用 /speckit 获取 SDD 工作流指令
```

每一步只加载必要的信息，上下文窗口利用率最高。

### Bad Case：上下文洪泛

```
Agent 进入仓库
  → 读完所有 README.md（上万行）
    → 读完所有源代码文件
      → 上下文溢出，开始遗忘早期信息
        → 输出不符合项目规范
```

---

[上一节：Agent 技术栈](020-agent-tech-stack.md) | [下一节：流程控制](025-flow-control.md) | [返回目录](000-index.md)
