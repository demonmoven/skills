# 3.2.3 Compound（复利工程）

> **本节目标**：理解 Compound Engineering 的核心理念——每一次工程迭代都应该让下一次更容易，而非更难。

---

## 概念与历史

### 什么是 Compound Engineering

**Compound Engineering（复利工程）** 由 Every.to 提出并系统化，是 Agent 时代的研发效率方法论。

核心命题：

> **Each unit of engineering work should make subsequent units easier—not harder.**
> 每一次工程迭代都应该让下一次更容易，而非更难。

这和传统软件开发的经验恰好相反——传统模式下，代码库越大、越复杂，后续开发越慢（技术债累积）。Compound Engineering 通过系统化的知识回流，**反转这个趋势**：

```text
传统模式：
  代码量增长 → 复杂度增长 → 开发速度下降 → 越来越慢

Compound 模式：
  代码量增长 → 知识同步增长 → Agent 的上下文越来越丰富 → 越来越快
```

### 类比

就像金融复利一样，收益会产生新的收益：

```text
金融复利：  本金 × (1 + 利率)^时间           = 指数增长
工程复利：  开发效率 × (1 + 每次知识沉淀)^迭代次数  = 指数提速
```

关键区别在于：**AI 工程让你今天更快，Compound Engineering 让你明天更快，以及之后的每一天都更快。**

### 量化效果

根据 Every.to 的实践数据：
- 第 8 周起，团队通常报告 **2-3x** 的速度提升
- 随着模式持续积累，最终可达 **4-7x** 的效率增益

---

## Compound 循环

Compound Engineering 的核心是一个四步循环，约 **80% 的精力用于 Plan 和 Review，20% 用于执行**：

```text
                ┌──────────┐
                │  Plan    │  ← 规划：明确目标、拆解任务、准备上下文
                └────┬─────┘
                     ↓
                ┌──────────┐
                │  Work    │  ← 执行：Agent 编码（这一步是最短的）
                └────┬─────┘
                     ↓
                ┌──────────┐
                │  Review  │  ← 审查：验证质量、发现问题、提炼洞察
                └────┬─────┘
                     ↓
                ┌──────────┐
                │ Compound │  ← 复利：沉淀知识，让下次迭代更快
                └────┬─────┘
                     ↓
               下一次迭代更快
                     ↓
                ┌──────────┐
                │  Plan    │  ← 这次规划更快，因为有上次的知识积累
                └──────────┘
```

### 每一步做什么

| 步骤 | 时间占比 | 关键动作 |
|------|----------|----------|
| **Plan** | ~40% | 明确目标、整理上下文、准备 Agent 需要的信息 |
| **Work** | ~20% | Agent 执行编码（人类监督和引导） |
| **Review** | ~30% | 验证结果、发现偏差、提炼可复用的模式 |
| **Compound** | ~10% | 沉淀知识产物，更新文档/约束/测试 |

---

## "复利"沉淀什么

每次迭代完成后，Compound 步骤产出的知识产物会回流到下次迭代的上下文中：

| 产物类型 | 说明 | 回流方式 |
|----------|------|----------|
| **文档更新** | 新发现的坑、模式、最佳实践 | 写入 AGENTS.md / docs/guidance/ |
| **约束规则** | 反复违反的规范 | 提升为 lint 规则或 pre-commit hook |
| **测试用例** | Bug 复现、边界条件 | 写入测试套件，防止回归 |
| **Prompt 模式** | 有效的 Agent 指令模板 | 封装为 Skill 或 Slash Command |
| **架构决策** | 设计取舍和原因 | 写入 ARCHITECTURE.md 或 ADR |

### 复利的飞轮效应

```text
第 1 次迭代：从零开始，Agent 对项目一无所知
                → Compound：写好 AGENTS.md + ARCHITECTURE.md

第 2 次迭代：Agent 有了基础上下文，少走弯路
                → Compound：踩到 SQLite 兼容性坑，写入 pitfalls 文档

第 3 次迭代：Agent 自动避开已知坑，效率更高
                → Compound：发现某个模式反复出现，封装为 Skill

第 N 次迭代：Agent 有丰富的上下文 + 自动化约束 + 专用 Skill
                → 开发速度是第 1 次的数倍
```

---

## Slash Command

### 复利相关

| 命令 | 说明 | 产物 |
|------|------|------|
| `/ce:compound` | **知识沉淀入口**——将本次迭代解决的问题结构化文档化 | `docs/solutions/{category}/{slug}-{date}.md` |
| `/ce:compound-refresh` | 刷新已有的 solutions 文档（更新过时内容） | 更新 `docs/solutions/` 下已有文件 |

### 完整循环（详见 [SDD 视角](031a5-compound.md)）

| 命令 | 说明 | 产物 |
|------|------|------|
| `/ce:brainstorm` | 需求精炼 | `docs/brainstorms/{name}-requirements.md` |
| `/ce:plan` | 计划编写（含 confidence check） | `docs/plans/YYYY-MM-DD-NNN-{type}-{name}-plan.md` |
| `/ce:work` | 编码执行（worktree + SubAgent） | 代码变更 + commit |
| `/ce:review` | 多角色审查（17 人格） | —（对话输出 / autofix 代码修改） |
| `/ce:compound` | 知识沉淀 | `docs/solutions/{category}/{slug}-{date}.md` |
| `/lfg` | 全自动管线（plan → work → review → compound） | 同上述各阶段产物之和 |

---

## 中间产物：Solution 文档

> **核心思想**：将解决过的问题结构化沉淀，让 Agent 下次遇到类似问题时直接查阅——这是"复利效应"的物理载体。

### 模板

```markdown
<!-- 存放路径: docs/solutions/{category}/{slug}-{date}.md -->
---
title: {解决方案标题}
category: {分类，如 database / auth / performance / deployment}
date: YYYY-MM-DD
tags: [{关键词1}, {关键词2}]
---

# {Solution Title}

## Problem
<!-- 概要：遇到的问题是什么，在什么场景下触发 -->

## Root Cause
<!-- 概要：根因分析——为什么会出现这个问题 -->

## Solution
<!-- 概要：如何解决，关键代码/配置/操作 -->

## Key Insight
<!-- 概要：可复用的洞察——下次遇到类似问题时的判断依据 -->

## Prevention
<!-- 概要：如何防止再次发生——hook / lint / test / 文档 -->
```

### 模板规范

| 维度 | 要求 |
|------|------|
| YAML Frontmatter | 必须包含 title / category / date |
| Problem | 必须描述触发场景，而非只说"XX 报错了" |
| Key Insight | 必须提炼可复用的判断依据，不能只是"改了 XX 就好了" |
| Prevention | 明确防护手段的优先级：自动化（hook/test） > 文档 |
| 分类目录 | 按技术域分类存放，便于 Agent 检索 |

### 示例

```markdown
---
title: SQLite WAL 模式在 Docker 容器中的兼容性问题
category: database
date: 2026-03-15
tags: [sqlite, docker, wal]
---

# SQLite WAL 模式在 Docker 容器中的兼容性问题

## Problem
在 Docker 容器内使用 SQLite WAL 模式时，容器重启后数据丢失。
仅在使用 tmpfs 挂载的场景下触发。

## Root Cause
WAL 模式依赖 mmap，而 tmpfs 的 mmap 实现在容器重启时不保证持久化 WAL 日志。

## Solution
将 SQLite 数据文件挂载到宿主机的持久化目录，而非 tmpfs。

## Key Insight
使用 WAL 模式时，必须确保底层文件系统支持持久化 mmap——
判断依据：如果 `mount` 命令输出包含 `tmpfs`，则不能用 WAL。

## Prevention
- 已添加 CI 检查：容器启动脚本中检测挂载类型，tmpfs 时自动回退到 DELETE 模式
- 写入本文档供后续 Agent 参考
```

---

## 最佳实践

### Compound 的关键原则

1. **每次迭代都要问**："这次学到了什么？怎么让下次更快？"
2. **知识必须入库**：放在个人笔记里的知识 = 不存在（Agent 看不到）
3. **自动化优于文档**：能变成 hook/lint/test 的规则就不要只写文档
4. **第三次就该自动化**：同一个问题出现第三次，说明文档没用，必须升级为自动化防护

### 什么值得做 Compound

| 信号 | Compound 动作 |
|------|--------------|
| 同一个 bug 出现第 2 次 | 写回归测试 + 文档 |
| Agent 反复犯同一个错误 | 加 hook 或约束 |
| 一个有效的 prompt 模式 | 封装为 Skill |
| 配置/环境问题导致的耗时调试 | 写入 debugging-playbook |
| 一次性的手误 | 不需要 Compound |

---

## Good Case vs Bad Case

### Good Case

```text
团队采用 Compound Engineering 后的第 8 周：

Week 1:  新项目初始化，Agent 什么都不知道          → 效率 1x
Week 2:  AGENTS.md + ARCHITECTURE.md 就位         → 效率 1.5x
Week 4:  3 个常见坑已文档化 + 2 个 hook 已安装     → 效率 2x
Week 6:  核心工作流封装为 3 个 Skill               → 效率 3x
Week 8:  Agent 上下文丰富，约束完善，Skill 齐全     → 效率 4x+

每一周都比上一周更快——这就是复利。
```

### Bad Case

| 错误 | 后果 |
|------|------|
| 修完 bug 就走，不做任何 Compound | 同样的 bug 3 个月后再次出现，从头排查 |
| 知识沉淀在个人笔记/飞书文档中 | Agent 看不到，无法利用 |
| 只做文档不做自动化 | 依赖人/Agent 自觉遵守，第三次还是会犯 |
| 过度 Compound（每个小问题都大张旗鼓） | 文档膨胀，噪音淹没信号，Compound 步骤反而拖慢迭代 |
| 不做 Review 直接 Compound | 沉淀了错误的知识，误导后续迭代 |

---

## 延伸阅读

- [Compound Engineering: The Definitive Guide](https://every.to/source-code/compound-engineering-the-definitive-guide) — Every.to 官方完整指南
- [Compound Engineering: How Every Codes With Agents](https://every.to/chain-of-thought/compound-engineering-how-every-codes-with-agents) — 方法论起源
- [Compound Engineering Plugin](https://github.com/EveryInc/compound-engineering-plugin) — Claude Code / Codex 官方插件

---

[上一节：UI 还原](032b-ui.md) | [返回上级：Post Task Completion](032-post-task-completion.md)
