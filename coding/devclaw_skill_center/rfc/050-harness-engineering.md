# 5. Harness Engineering

> **本章目标**：理解 Harness Engineering 的核心理念、三大支柱和实践方法——这是 AI Native 时代的工程范式基石。

---

## 概念与历史

### 起源

**Harness Engineering** 由 OpenAI 在 2025 年提出，Martin Fowler 随后将其系统化分析。核心洞察：

> **Agent 的表现上限不取决于模型本身，而取决于它工作的环境有多好。**

### 类比

```text
传统软件工程的比喻：
  程序员 = 赛车手
  IDE/工具 = 赛车

Harness Engineering 的比喻：
  Agent = 赛马
  Harness = 缰绳 + 马具 + 赛道
  工程师 = 驯马师（设计缰绳的人）
```

**工程师的核心工作**从"写代码"转变为"搭建环境、说清意图、建立反馈闭环"——让 Agent 高效且可控地完成编码。

### 核心等式

```
更多的 AI 自治 = 更严格的运行时约束
```

Agent 越自主，环境约束就要越严格。这不是矛盾，而是**自由和边界的统一**。

---

## 三大支柱

```text
              ┌──────────────────────────────────┐
              │      Harness Engineering          │
              └──────────┬───────────────────────┘
                         │
         ┌───────────────┼───────────────┐
         │               │               │
    ┌────▼────┐    ┌─────▼─────┐   ┌─────▼─────┐
    │ 上下文  │    │  架构约束  │   │  垃圾回收  │
    │  工程   │    │           │   │           │
    └─────────┘    └───────────┘   └───────────┘
    Agent 看到       Agent 不能       Agent 产出
    正确的信息       做什么          的质量把关
```

---

### 支柱一：上下文工程（Context Engineering）

**问题**：Agent 进入一个陌生仓库，不知道项目架构、编码规范、业务约束。

**解决方案**：分层文档体系 + 渐进式披露。

详见 [2.1 上下文控制](021-context-control.md) 和 [2.1.1 Agents 文档体系](021a-agents-docs.md)。

核心原则：
- **给 Agent 一张地图，不是一本千页大辞典**
- **仓库即唯一真相源**：Agent 看不到的信息等于不存在
- **知识版本化**：所有决策、规范、设计必须以 Markdown 形式入 Git

实践：
| 实践 | 作用 |
|------|------|
| L0-L4 五层文档体系 | 渐进式披露 |
| AGENTS.md 知识导航表 | Agent 快速定位 |
| ARCHITECTURE.md 不变量 | 明确边界 |
| ExecPlan 生命周期管理 | 中断恢复 |

---

### 支柱二：架构约束（Architecture Constraints）

**问题**：Agent 善于生成代码，但容易违反项目中隐含的架构规则。

**解决方案**：将隐含规则显式化为不变量，并通过自动化层级强制执行。

详见 [2.5.2 Hooks](025b-hooks.md)。

四层约束模型：

| 层级 | 手段 | 作用时机 | 举例 |
|------|------|----------|------|
| **文档层** | AGENTS.md 不变量 | Agent 阅读时 | "handler 不直接访问 DB" |
| **Lint 层** | golangci-lint / biome | 编码时 | funlen ≤ 120 行 |
| **Hook 层** | pre-commit hooks | 提交时 | AI Guard（占位符/幻觉/长度） |
| **CI 层** | CI pipeline | 合并时 | 覆盖率门禁 |

**升级规则**：当同一规范被违反 3 次以上，应将 enforcement 提升到下一层级。

实践：
| 实践 | 作用 |
|------|------|
| 18 条 pre-commit hook | 提交时拦截 |
| AI Guard（3 条 AI 专项） | 拦截 Agent 特有问题 |
| DDD 四层架构 | 依赖方向管控 |
| IDL 契约驱动 | API 变更唯一来源 |

---

### 支柱三：垃圾回收（Garbage Collection）

**问题**：Agent 生成的代码可能包含占位符、幻觉标记、技术债。随时间推移，代码库质量持续下降。

**解决方案**：通过 AI Guard hooks + 周期性扫描 + 技术债治理，实现自动化的代码质量维护。

详见 [3.2.3 Compound 复利工程](032c-compound.md)。

实践：
| 实践 | 作用 |
|------|------|
| AI Guard hooks | 提交时拦截占位符/幻觉 |
| tech-debt-checker Skill | 5 条架构级技术债检测规则 |
| check-active-plans.sh | pre-push 守护计划完成度 |
| Compound 复利动作 | 每次问题解决后沉淀防护 |

tech-debt-checker 的 5 条架构级规则：

| 规则 | 检测内容 | 严重度 |
|------|----------|--------|
| duplicate-persistence | 重复持久化实现 | 高 |
| divergent-impl | 同一功能多处独立实现 | 高 |
| model-divergence | 前后端数据模型分裂 | 中高 |
| scattered-state | 配置/状态散落多文件 | 中 |
| unabstracted-flow | 相似流程未抽象 | 中 |

---

## Harness 的 5 个 Action

`harness` Skill 通过 5 个 action 将三大支柱一键落地：

| Action | 用途 | 说明 |
|--------|------|------|
| `init` | 新仓库工程化初始化 | 6 phase 全流程（分析→hooks→docs→AGENTS.md→ARCH.md→验证） |
| `debt-fix` | 技术债治理 | 每次最多修 3 项，逐 commit 验证 |
| `doc-fix` | 修复文档 | 以代码为准，修复文档与代码不一致处 |
| `evolve` | 健康检查 | 检测+报告+路由建议（**不修改任何代码或文档**） |
| `lint-promote` | 规范升级 | 把反复违反的文档级规范提升为自动化 lint 规则 |

### 典型使用流程

```bash
# 1. 新项目初始化
/harness init

# 2. 定期健康检查
/harness evolve          # 只做检测，不改东西

# 3. 根据报告执行
/harness doc-fix         # 修文档
/harness debt-fix        # 修技术债
/harness lint-promote    # 升级规范
```

---

## DevClaw 的 Harness 实践

DevClaw 作为 AI 研发工作台项目，本身就是 Harness Engineering 的实验场：

| Harness 原则 | DevClaw 落地 |
|---|---|
| AGENTS.md 当目录用 | 五层文档体系（L0—L4），知识导航表 |
| 仓库即唯一真相源 | 16 篇 reference + 5 篇 guidance + 30+ ExecPlan |
| 计划是一等公民 | 完整生命周期管理 + pre-push 守护 |
| 架构约束自动执行 | 18 条 pre-commit hook + 3 条 AI Guard |
| 刚性分层架构 | DDD 四层 + IDL 契约驱动 + 前端模块边界 |
| 报错即指导 | 中文错误提示 |
| 让 Agent 看见应用 | VictoriaLogs/Traces + 结构化日志 |
| 垃圾回收式维护 | tech-debt-checker Skill |

---

## Good Case vs Bad Case

### Good Case

```text
新项目接入 Harness：
  /harness init
    → Phase 1: 自动检测技术栈（Go + React）
    → Phase 2: 安装适配的 pre-commit hooks
    → Phase 3: 创建 docs/ 骨架
    → Phase 4: 生成 AGENTS.md（100行导航）
    → Phase 5: 生成 ARCHITECTURE.md（200行架构）
    → Phase 6: 18点验证清单

  之后 Agent 进入仓库：
    → 先读 AGENTS.md → 知道项目结构
    → 按知识导航表找到要改的模块
    → 按 ARCHITECTURE.md 的不变量约束编码
    → Pre-commit hook 自动拦截问题
    → 结果：高质量、符合规范的代码
```

### Bad Case

| 错误 | 后果 |
|------|------|
| 不做 Harness 直接让 Agent 写代码 | Agent 不知道项目规范，产出的代码"能跑但不符合架构" |
| 只写 AGENTS.md 不装 hooks | Agent 知道规则但不遵守（自觉性不可靠） |
| 只装 hooks 不写文档 | Agent 被 hook 拦截但不知道怎么修 |
| 用 `/harness init` 做业务开发 | harness 是工程化初始化工具，不是需求实现工具 |
| 跳过 evolve 直接改 | 不知道当前健康状态，改了可能更糟 |

---

## 总结

Harness Engineering 是 AI Native 时代的工程基石。它回答了一个核心问题：

> **怎么让 Agent 在你的代码库里安全、高效、可控地工作？**

答案是三个支柱的组合：
1. **上下文工程**：让 Agent 看到正确的信息
2. **架构约束**：让 Agent 知道不能做什么
3. **垃圾回收**：自动清理 Agent 产出中的问题

这三者缺一不可。光有上下文没有约束，Agent 知道规则但可能违反；光有约束没有上下文，Agent 被 hook 拦截但不知道怎么改；光有前两者没有垃圾回收，代码库会随时间慢慢腐化。

---

[上一章：观测 & 评测](040-observation-evaluation.md) | [返回目录](000-index.md)
