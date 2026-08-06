# 知识演进编排器

> **本文件是 prompt 指令**，由 `<skill_dir>/actions/evolve/evolve.md` 加载。原属于 `harness-bootstrap` 项目的 `harness-evolve` 演进 skill，已合并对应的 `references/evolution-triggers.md` 详细参考（见文件末尾"附录"段）。
>
> **重要**：原文中提到的 `harness-debt-fix` / `harness-doc-fix` / `harness-lint-promote` 三个 skill，迁移后均已成为 harness skill 的 action：调用方式分别为 `/harness debt-fix`、`/harness doc-fix`、`/harness lint-promote`。

harness-evolve 是知识演进的**编排层**。它负责检测代码变更事件、生成健康检查报告、将执行工作路由到专门的 skill。

**harness-evolve 自身不修改任何代码或文档。** 它只做三件事：检测、报告、路由。

## 核心理念

- **写一次不够，持续演进才是价值**：初始化（harness-bootstrap）建立骨架，演进（harness-evolve）协调持续更新
- **事件驱动，非定期扫描**：文档更新由具体的代码变更事件触发，而非靠人记住去做
- **增量积累，复利效应**：每次小更新的价值随时间累积，最终形成高度精准的仓库知识库

## 执行 Skill

harness-evolve 将实际工作路由到以下三个执行 skill：

| Skill | 触发词 | 职责 |
|-------|--------|------|
| **harness-doc-fix** | `文档治理`、`doc fix`、`文档对齐` | 以代码为准修复文档——更新 ARCHITECTURE.md、AGENTS.md、code-patterns、ADR 等 |
| **harness-debt-fix** | `技术债治理`、`debt fix`、`修复技术债` | 识别+修复技术债——记录 debt-log、修复 invariant 违反、清理已知债务 |
| **harness-lint-promote** | `规范 lint 化`、`lint promote`、`Promotion Rule` | 将文档规范转化为自动化 lint 规则——编写 lint rule、配置 pre-commit hook、设置 CI check |

## 场景 1: 开发过程中的事件驱动判断

当 ExecPlan 完成或检测到结构变更时，harness-evolve 负责判断需要触发哪些执行 skill。

### ExecPlan 完成后

**触发条件**：ExecPlan 标记为 completed（从 `docs/plans/active/` 移入 `docs/plans/completed/`）。

读取完成的 ExecPlan，逐项评估以下 7 项检查清单：

| # | 检查项 | 命中条件 | 路由 |
|---|--------|----------|------|
| 1 | 引入了新的代码模式？ | 新的编码惯例、错误处理方式、组件结构等 | → **harness-doc-fix** |
| 2 | 涉及架构决策？ | 有替代方案的设计选择，影响未来开发方向 | → **harness-doc-fix** |
| 3 | 踩了新坑或发现调试技巧？ | 非直觉的问题、有效的排查方法 | → **harness-doc-fix** |
| 4 | 填补了已知知识缺口？ | 解答了 knowledge-gaps.md 中的 open 条目 | → **harness-doc-fix** |
| 5 | 改变了模块边界？ | 新增/删除/重命名顶层目录或核心模块 | → **harness-doc-fix** |
| 6 | 新增或修改了不变量？ | 新的跨模块约束，或已有不变量需调整 | → **harness-doc-fix** |
| 7 | 发现 invariant 被违反？ | 代码违反了 invariants.md 中的约束 | → **harness-debt-fix**；若同一 invariant 累计 ≥3 次违反 → 同时路由 **harness-lint-promote** |

**执行方式**：

1. 评估完 7 项后，汇总命中项
2. 向用户报告检测结果及建议的执行 skill
3. 用户确认后，调用对应的执行 skill

> 详细的事件检测信号见本文件末尾"附录：evolution triggers 详细参考"段。

### 结构变更时

**触发条件**：开发过程中检测到以下结构性变化：

- 新增顶层目录或新增 package/module
- 新增外部服务/API 集成
- 删除或重命名主要模块
- 新增或移除基础设施组件（数据库、消息队列、缓存等）

**路由**：→ **harness-doc-fix**

### 代码模式偏离时

**触发条件**：发现代码偏离了 `docs/reference/code-patterns.md` 中记录的模式。

**判断与路由**：

- 如果偏离代表**模式演变**（新方式更好）→ **harness-doc-fix**（更新文档记录新模式）
- 如果偏离代表**模式违规**（代码有误）→ **harness-debt-fix**（记录+修复违规）

## 场景 2: 定期健康检查

**触发条件**：用户手动调用（触发词：`文档健康检查`、`doc health check`、`harness health`）。

执行 5 个维度的检查，**仅生成报告，不执行修复**：

| # | 检查维度 | 检查方法 | 风险判定 |
|---|----------|----------|----------|
| 1 | 知识缺口老化 | 检查 knowledge-gaps.md 中 High Priority 条目的创建日期 | >30 天未处理 → 过期 |
| 2 | ARCHITECTURE.md 过时 | 对比 Code Map 模块列表与当前实际目录结构 | 存在不匹配 → 需更新 |
| 3 | 代码模式一致性 | 抽样最近 20 个 commit 涉及的文件，对比 code-patterns.md | 发现偏离 → 判断演变/违规 |
| 4 | 不变量合规 | 对 invariants.md 中每条不变量执行验证检查 | 发现违反 → 标记 |
| 5 | 导航完整性 | 检查 docs/ 下文件是否都在 AGENTS.md 导航表中 | 发现遗漏 → 标记 |

> 详细的检查方法见本文件末尾"附录：evolution triggers 详细参考"段（场景 2 部分）。

## 健康检查报告格式

检查完成后生成以下格式的报告：

```markdown
# 文档健康度报告

生成时间: YYYY-MM-DD HH:MM
检查仓库: {repo-name}

## 总览

| 维度 | 状态 | 详情 |
|------|------|------|
| 知识缺口 | ⚠ / ✓ | N 个高优先级缺口过期 |
| 架构文档 | ⚠ / ✓ | N 个模块描述过时 |
| 代码模式 | ⚠ / ✓ | N 个模式偏离 |
| 不变量合规 | ⚠ / ✓ | N 条不变量违反 |
| 导航完整性 | ⚠ / ✓ | N 个文档缺少导航 |

## 建议操作

1. [高优] {问题描述} → 建议执行 **harness-doc-fix** / **harness-debt-fix**
2. [中优] {问题描述} → 建议执行 **harness-doc-fix**
3. [低优] {问题描述} → 建议执行 **harness-doc-fix**
4. [信息] invariant {ID} 累计违反 N 次 → 若 ≥3 次，建议执行 **harness-lint-promote**
```

每条建议必须明确指向一个具体的执行 skill，让用户可以直接调用。

## 重要约束

**harness-evolve 绝不直接修改代码或文档。** 它的职责边界是：

| 可以做 | 不可以做 |
|--------|----------|
| 检测代码变更事件 | 修改任何源代码文件 |
| 读取文档内容做对比 | 写入或编辑任何文档 |
| 生成健康检查报告 | 执行文档更新操作 |
| 推荐应调用的执行 skill | 直接执行 debt-log 记录 |
| 评估 invariant 违反次数 | 编写或修改 lint 规则 |

所有实际修改工作由对应的执行 skill 完成：
- 文档更新 → **harness-doc-fix**
- 技术债修复 → **harness-debt-fix**
- 规范 lint 化 → **harness-lint-promote**


---

# 附录：evolution triggers 详细参考

# Evolution Triggers — 事件检测与路由参考

本文档是 harness-evolve 编排器的详细参考，描述三类事件的检测信号和路由规则。

> 本文档只包含检测条件和路由目标。具体的更新格式和执行细节已移至对应的执行 skill（harness-doc-fix / harness-debt-fix / harness-lint-promote）。

## 一、ExecPlan 完成事件

### 触发识别

当以下任一条件成立时，进入 ExecPlan 完成检查：

- ExecPlan 文件从 `docs/plans/active/` 移入 `docs/plans/completed/`
- 用户明确表示一个 ExecPlan 已完成
- Agent 在执行 ExecPlan 最后一个里程碑的过程中

### 7 项检查清单

#### 1. 新代码模式

**何时命中**：建立了团队之前未使用过的编码惯例。
**信号**：引入新设计模式（Repository Pattern、Event Sourcing）、新文件组织惯例、新错误处理方式、新异步模式
**路由** → **harness-doc-fix**

#### 2. 架构决策

**何时命中**：做了有替代方案的设计选择，且影响未来开发方向。
**信号**：ExecPlan 中有"Design Decisions"或"替代方案"内容、选择了特定方案放弃其他方案、引入新外部依赖
**路由** → **harness-doc-fix**

#### 3. 调试知识

**何时命中**：遇到非直觉问题或找到有效排查方法。
**信号**：Progress 中记录了"踩坑/排查过程"、某步骤耗时远超预期、发现未记录的系统行为
**路由** → **harness-doc-fix**

#### 4. 知识缺口关闭

**何时命中**：实现解答了 `docs/quality/knowledge-gaps.md` 中的某个 open 条目。
**信号**：ExecPlan 涉及领域与某 open 缺口 Area 匹配、获得的信息能回答缺口描述
**路由** → **harness-doc-fix**

#### 5. 模块边界变更

**何时命中**：涉及新增/删除/重命名顶层目录、package 或核心模块。
**信号**：`packages/`、`internal/`、`cmd/` 等关键目录下出现新建/删除/重命名/合并拆分
**路由** → **harness-doc-fix**

#### 6. 不变量变更

**何时命中**：引入新的跨模块约束，或已有不变量需要调整。
**信号**：新增跨模块隐式约束、发现已记录不变量描述不准确、废弃某条不变量
**路由** → **harness-doc-fix**

#### 7. Invariant 违反

**何时命中**：代码违反了 `docs/rules/invariants.md` 中的约束。
**信号**：import/调用/数据流与 invariant 矛盾、因违反 invariant 导致 bug、Code review 指出违反
**路由**：
- → **harness-debt-fix**
- 若同一 invariant 在 debt-log.md 中累计 ≥3 次 `invariant-violation` → 同时 **harness-lint-promote**

### 汇总路由表

| 命中项 | 路由目标 |
|--------|----------|
| #1 ~ #6 | **harness-doc-fix** |
| #7 | **harness-debt-fix** |
| #7 且同一 invariant ≥3 次 | **harness-debt-fix** + **harness-lint-promote** |

## 二、结构变更事件

### 触发识别

Agent 在开发过程中执行以下操作时触发：

- 创建新的顶层目录或 package
- 引入新的外部服务 SDK 或 API 客户端
- 删除或重命名主要模块/目录
- 添加新基础设施组件（数据库、消息队列、缓存等）

### 变更分类与检测信号

| 变更类型 | 检测信号 |
|----------|----------|
| 新增模块/Package | 出现新的顶层子目录或 packages/ 下新目录 |
| 删除模块 | 主要子目录被移除 |
| 重命名模块 | 目录名变更（git 检测为 rename） |
| 新增外部集成 | 引入新的 SDK 依赖、API 客户端代码、配置项 |
| 基础设施变更 | 新增数据库连接、消息队列配置、缓存层等 |

### 路由

所有结构变更事件 → **harness-doc-fix**

## 三、健康检查事件

### 触发识别

用户手动调用，触发词：`文档健康检查`、`doc health check`、`harness health`、`检查文档健康度`

### 5 维度检测方法

#### 1. 知识缺口老化

**方法**：读取 knowledge-gaps.md，对 open 条目检查创建日期。High Priority >30 天或 Medium >60 天 → 过期。
**路由** → **harness-doc-fix**

#### 2. ARCHITECTURE.md 过时

**方法**：提取 Code Map 模块路径，对比仓库实际目录。有但不存在 → 幽灵模块；存在但没记录 → 未记录模块。
**路由** → **harness-doc-fix**

#### 3. 代码模式一致性

**方法**：提取 code-patterns.md 已记录模式，抽样最近 20 commit 涉及的 5-10 个业务文件做对比。判断偏离是演变还是违规。
**路由**：演变 → **harness-doc-fix**；违规 → **harness-debt-fix**

#### 4. 不变量合规

**方法**：对 invariants.md 中可自动验证的条目执行检查（import 方向、命名惯例、目录边界）。统计 debt-log.md 中违反次数。
**路由**：违反 → **harness-debt-fix**；累计 ≥3 次 → 追加 **harness-lint-promote**

#### 5. 导航完整性

**方法**：扫描 `docs/reference/`、`docs/guidance/`、`docs/quality/`、`docs/rules/` 下 `.md` 文件，检查是否都在 AGENTS.md 导航表中。
**路由** → **harness-doc-fix**
