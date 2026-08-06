# 2.5.1 SubAgent（流程编排视角）

> **本节目标**：理解 SubAgent 在流程编排中的三种模式——并行、串行、隔离。

---

## 概念

在 [2.1.3](021c-subagent.md) 中我们讨论了 SubAgent 的上下文隔离价值。本节聚焦它的另一重身份：**流程编排工具**。

SubAgent 让主 Agent 能像项目经理一样"分活"：

```text
主 Agent（项目经理）
  ├── SubAgent-A：调研技术方案（并行）
  ├── SubAgent-B：搜索相关代码（并行）
  └── 等待 A、B 返回 → 主 Agent 综合判断 → 开始实施
```

---

## 三种编排模式

### 模式 1：并行执行

多个独立任务同时启动，不存在依赖关系：

```text
主 Agent ──┬──→ SubAgent-A: 搜索前端代码
           ├──→ SubAgent-B: 搜索后端代码
           └──→ SubAgent-C: 搜索配置文件
                    ↓
           所有结果汇总后，主 Agent 决策
```

**适用场景**：独立的搜索/调研任务

### 模式 2：串行执行

任务之间有依赖，后一个需要前一个的结果：

```text
主 Agent ──→ SubAgent-A: 出设计方案
                ↓ (方案结果)
             SubAgent-B: 基于方案实现代码
                ↓ (代码结果)
             SubAgent-C: 编写测试
```

**适用场景**：有先后顺序的工作流

### 模式 3：Worktree 隔离

SubAgent 在独立的 git worktree 中工作，完全不影响主分支：

```text
主 Agent (main 分支)
  └──→ SubAgent (临时 worktree 分支)
           ├── 实现功能
           ├── 运行测试
           └── 返回结果 + 分支名
       主 Agent 决定是否合并
```

**适用场景**：探索性实现、风险较高的修改

---

## 最佳实践

### 何时并行 vs 串行

| 信号 | 选择 |
|------|------|
| 任务之间无数据依赖 | **并行** |
| 需要前一步的输出作为后一步的输入 | **串行** |
| 不确定方案是否可行 | **Worktree 隔离** |
| 需要多角度独立评估 | **并行**（避免相互影响） |

### 如何控制 SubAgent 的输出质量

1. **限制输出长度**：在 prompt 中明确"200字以内"
2. **指定输出格式**："报告：文件路径、行号、一句话总结"
3. **给足上下文**：SubAgent 看不到主对话，必须在 prompt 中交代清楚

---

## Good Case vs Bad Case

### Good Case

```
需求：调研项目中所有的 API 端点

主 Agent 并行启动 3 个 Explore SubAgent：
- SubAgent-1: 搜索 backend/ 下的 handler
- SubAgent-2: 搜索 idl/ 下的 thrift 定义
- SubAgent-3: 搜索 docs/reference/ 下的 API 文档

3 个 Agent 并行执行，总耗时 = max(单个耗时)
```

### Bad Case

| 错误 | 后果 |
|------|------|
| 所有任务都串行 SubAgent | 等待时间 = sum(所有任务)，浪费 |
| 并行的 SubAgent 修改相同文件 | 结果冲突，需要手动合并 |
| 不用 Worktree 做破坏性修改 | 失败后需要手动恢复 |

---

[下一节：Hooks →](025b-hooks.md) | [返回上级：流程控制](025-flow-control.md)
