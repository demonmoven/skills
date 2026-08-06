# Harness Doc Fix — 文档治理

> **本文件是 prompt 指令**，由 `<skill_dir>/actions/doc-fix/doc-fix.md` 加载。原属于 `harness-bootstrap` 项目的 `harness-doc-fix` 演进 skill，已合并对应的 `references/doc-fix-workflow.md` 详细参考（见文件末尾"附录"段）。

以代码为唯一真相来源，修改文档中与代码不符的部分。只修改文档，绝不修改代码。

## 何时使用

| 场景 | 触发条件 | 分支策略 |
|------|----------|----------|
| 定期触发 | 用户手动调用（触发词：`文档治理`、`doc fix`） | 创建独立分支，提独立 MR |
| ExecPlan 完成后 | Agent 将 ExecPlan 标记为 completed | 跟随当前用户分支 |
| 结构变更 | 新增/删除/重命名顶层目录、package 或核心模块 | 跟随当前用户分支 |
| 用户主动 | 用户指定要修复某份文档或对齐某个模块的文档 | 按用户指示决定 |

## 检查范围

对以下 6 项逐一检查，识别文档与代码的偏差：

### 1. ARCHITECTURE.md vs 实际目录结构

读取 `ARCHITECTURE.md` 的 Code Map 部分，提取列出的所有模块路径。扫描仓库的实际目录结构（顶层目录 + `packages/`、`internal/`、`src/` 等关键子目录）。对比两者：

- **幽灵模块**：Code Map 中列出但目录中已不存在的模块 → 需从文档中移除或标记为废弃
- **未记录模块**：目录中存在但 Code Map 中缺失的模块 → 需在文档中补充描述

### 2. code-patterns.md vs 实际代码模式

读取 `docs/reference/code-patterns.md`，提取已记录的所有模式。抽样检查最近 20 个 commit 涉及的业务逻辑文件。对比：

- **新未记录模式**：代码中出现了文档未记录的新编码模式 → 补充到文档
- **过时已记录模式**：文档中记录的模式在代码中已不再使用 → 标记为废弃或移除

### 3. invariants.md 路径有效性

读取 `docs/rules/invariants.md`，提取每条不变量中引用的文件路径和目录路径。逐一检查这些路径在仓库中是否仍然存在：

- 路径不存在 → 更新为当前正确路径，或标记该不变量需要重新评估

### 4. integrations.md vs 实际依赖

读取 `docs/reference/integrations.md`，提取记录的外部集成清单。扫描代码中的实际外部依赖（import 语句、SDK 调用、API 客户端配置）。对比：

- **仅文档有**：文档中记录但代码中已不再使用的集成 → 从文档中移除
- **仅代码有**：代码中使用但文档中未记录的集成 → 补充到文档

### 5. AGENTS.md 导航完整性

扫描 `docs/` 下所有子目录（`docs/reference/`、`docs/guidance/`、`docs/quality/`、`docs/rules/` 等）中的 `.md` 文件。读取根 `AGENTS.md` 和 `docs/AGENTS.md`，检查每个文档文件是否在导航表中有对应条目：

- 未链接的文件 → 补充导航条目

### 6. knowledge-gaps.md open 条目评估

读取 `docs/quality/knowledge-gaps.md`，对每个 `open` 状态的条目：

- 检查条目描述的知识缺口是否已被后续代码变更或文档更新填补
- 如果已填补 → 标记为 `resolved` 并注明解决方式
- 如果仍然 open → 保持不变，但检查优先级是否仍然合理

## ExecPlan 完成后的知识沉淀

当 ExecPlan 完成后触发时，额外执行以下 6 项知识沉淀检查：

| # | 检查项 | 条件 | 更新目标 |
|---|--------|------|----------|
| 1 | 引入了新的代码模式？ | ExecPlan 中建立了新的编码惯例、错误处理方式、组件结构等 | `docs/reference/code-patterns.md` |
| 2 | 涉及架构决策？ | ExecPlan 中做了有替代方案的设计选择 | 创建新 ADR → `docs/reference/adr/ADR-NNN.md` |
| 3 | 踩了新坑或发现调试技巧？ | 实现过程中遇到了非直觉的问题或找到了排查方法 | `docs/guidance/debugging-playbook.md` 或 `docs/guidance/common-pitfalls.md` |
| 4 | 填补了已知知识缺口？ | ExecPlan 实现过程中解答了 knowledge-gaps.md 中的某个 open 条目 | `docs/quality/knowledge-gaps.md`（将对应条目标记为 resolved） |
| 5 | 改变了模块边界？ | 新增/删除/重命名了顶层目录、package 或核心模块 | `ARCHITECTURE.md` 的 Code Map 部分 |
| 6 | 新增或修改了不变量？ | 引入了新的跨模块约束，或发现已有不变量需要调整 | `docs/rules/invariants.md` |

## 工作流

### Phase 1: 扫描

执行上述 6 项检查范围扫描，收集所有偏差。如果是 ExecPlan 完成后触发，同时执行 6 项知识沉淀检查。

产出：偏差清单（每项包含：文件、位置、偏差类型、具体描述）。

### Phase 2: 优先级排序与确认

对偏差清单按优先级排序：

| 优先级 | 判定条件 | 示例 |
|--------|----------|------|
| 高 | 文档与代码直接矛盾 | Code Map 描述的模块职责与实际代码完全不同 |
| 中 | 文档缺失，信息不完整 | 新增模块未记录、新集成未文档化 |
| 低 | 文档不精确但不误导 | 描述措辞过时但大方向正确、路径写法略有出入 |

展示排序后的偏差清单，等待用户确认后再执行修复。

### Phase 3: 逐项修复

对用户确认的每项偏差：

1. **读取代码**：确认代码中的实际状态
2. **修复文档**：按代码实际状态更新文档（只改文档，不改代码）
3. **标注来源**：在更新内容中添加 HTML 注释标注
4. **提交**：每项修复独立 commit，格式 `docs: fix {文档名} - {修复描述}`

### Phase 4: 分支策略

根据触发场景决定收尾方式：

| 触发场景 | 操作 |
|---------|------|
| 定期手动触发 | 创建独立分支 `docs/doc-fix-YYYY-MM-DD`，push 并提 MR |
| ExecPlan 完成后触发 | 跟随当前用户分支，commit 直接加入 |
| 结构变更触发 | 跟随当前用户分支 |
| 用户主动触发 | 按用户指示决定 |

## 关键原则

1. **代码即真相**：当文档与代码不一致时，永远以代码为准修改文档。绝不因为文档这样写就去改代码。
2. **增量更新**：只更新变化的部分，不重写整个文档。修改应精确定位到相关段落。
3. **标注来源**：每次更新使用 HTML 注释标注触发原因，格式见本文件末尾"附录：doc-fix workflow 详细参考"段。
4. **保持简洁**：新增内容聚焦于模式、约束和决策，避免过度详细的实现细节。遵循"只写不太会变的东西"原则。
5. **关闭缺口**：每次更新后，检查 `docs/quality/knowledge-gaps.md` 是否有条目可以关闭。如果本次更新解答了某个缺口，将其标记为 resolved。

## 参考

详细的更新格式、ADR 模板、来源标注规范和完整工作流示例见本文件末尾"附录：doc-fix workflow 详细参考"段。


---

# 附录：doc-fix workflow 详细参考

# Doc Fix Workflow — 文档治理详细参考

本文档是 harness-doc-fix skill 的详细参考，包含各类文档的更新格式、ADR 模板、来源标注规范和完整工作流示例。

## 各类文档的更新格式

### code-patterns.md

每个新增模式使用以下格式，添加到 `docs/reference/code-patterns.md` 对应分类的末尾：

```markdown
<!-- 来源: ExecPlan {name}, {date} -->
### {模式名称}

**适用场景**：{何时使用此模式}

**结构**：
{代码结构描述或伪代码}

**示例文件**：
- `path/to/example1.ts` — {说明}
- `path/to/example2.ts` — {说明}
```

过时模式的处理：不直接删除，而是添加废弃标记：

```markdown
### ~~{旧模式名称}~~

> **已废弃**（{date}）：已被 [{新模式名称}](#{新模式锚点}) 替代。
> <!-- 来源: 文档治理 - 模式过时清理, {date} -->
```

### debugging-playbook.md

在 `docs/guidance/debugging-playbook.md` 对应问题类别的表格中新增行：

```markdown
<!-- 来源: ExecPlan {name}, {date} -->
| {症状描述} | {可能原因} | {排查步骤} |
```

格式要求：
- **症状**：用户或日志中可观察到的现象，简洁描述
- **可能原因**：导致该症状的根本原因
- **排查步骤**：编号列表，从最可能到最不可能排列

### common-pitfalls.md

在 `docs/guidance/common-pitfalls.md` 对应场景分类的表格中新增行：

```markdown
<!-- 来源: ExecPlan {name}, {date} -->
| {场景} | {坑点描述} | {正确做法} |
```

格式要求：
- **场景**：在什么情况下容易踩坑
- **坑点描述**：具体的错误做法或误解
- **正确做法**：应该怎么做，附代码片段（如需要）

### ARCHITECTURE.md Code Map

Code Map 使用缩进文件树格式，每行包含路径和行内注释：

```markdown
## Code Map

project-root/
  src/
    api/          # HTTP 入口层，路由定义和请求验证
    core/         # 核心业务逻辑，不依赖外部框架
    infra/        # 基础设施层，数据库和外部服务适配
    utils/        # 通用工具函数
  packages/
    auth-sdk/     # OAuth 2.0 客户端封装
    shared/       # 跨 package 共享类型和常量
  docs/           # harness 文档体系
```

新增模块时，在对应位置插入新行并标注来源：

```markdown
<!-- 来源: 结构变更 - 新增 packages/notification/, {date} -->
    notification/ # 通知服务，消息推送路由和交付
```

删除模块时，直接移除对应行（不保留废弃标记，Code Map 只反映当前状态）。

## ADR 模板和编号规则

### ADR 模板

创建新 ADR 文件 `docs/reference/adr/ADR-NNN.md`，使用以下模板：

```markdown
# ADR-NNN: {决策标题}

- 日期: {YYYY-MM-DD}
- 状态: accepted
- 来源: ExecPlan {plan-name}

## 背景

{为什么需要做这个决策——从 ExecPlan 的问题陈述中提取。描述面临的问题和约束条件。}

## 决策

{选择了什么方案——从 ExecPlan 的设计方案中提取。清晰陈述最终选择。}

## 替代方案

{考虑过但未选择的方案——从 ExecPlan 中提取。每个替代方案说明为什么被排除。}

### 方案 A: {名称}

{描述}

**排除原因**：{原因}

### 方案 B: {名称}

{描述}

**排除原因**：{原因}

## 后果

{这个决策的影响和约束——从 ExecPlan 实现过程中总结。}

- **正面**：{带来的好处}
- **负面**：{引入的约束或成本}
- **需关注**：{后续需要注意的事项}
```

### ADR 编号规则

1. 扫描 `docs/reference/adr/` 目录下已有的 ADR 文件
2. 提取最大编号（如 `ADR-005` → 最大编号为 5）
3. 新 ADR 编号 = 最大编号 + 1，三位数字补零（如 `006`）
4. 如果目录为空或不存在，从 `001` 开始

## knowledge-gaps.md 处理流程

### 从 open 到 resolved

当确认某个知识缺口已被填补时，执行以下更新：

**更新前**（在 Open 表中）：

```markdown
| KG-003 | Auth | ARCHITECTURE.md | OAuth 流程的 token 刷新策略未记录 | 实现后补充 | open |
```

**更新后**（移至 Resolved 表）：

```markdown
<!-- 来源: ExecPlan implement-auth-flow, 2026-04-01 -->
| KG-003 | Auth | ARCHITECTURE.md | OAuth 流程的 token 刷新策略未记录 | 已在 code-patterns.md 中记录 token 刷新模式 | resolved |
```

操作步骤：
1. 从 Open 表中删除该行
2. 在 Resolved 表中添加该行，Status 改为 `resolved`，"How to Fill" 改为实际的解决方式
3. 在该行上方添加来源标注

## 来源标注规范

所有由 harness-doc-fix 触发的文档更新必须使用 HTML 注释标注来源。格式：

```markdown
<!-- 来源: {场景标识}, {YYYY-MM-DD} -->
```

### 场景标识

根据触发场景使用不同的标识：

| 触发场景 | 场景标识格式 | 示例 |
|---------|-------------|------|
| ExecPlan 完成后 | `ExecPlan {plan-name}` | `<!-- 来源: ExecPlan implement-auth-flow, 2026-04-01 -->` |
| 结构变更 | `结构变更 - {变更描述}` | `<!-- 来源: 结构变更 - 新增 packages/auth-sdk/, 2026-04-01 -->` |
| 定期文档治理 | `文档治理 - {描述}` | `<!-- 来源: 文档治理 - 季度文档对齐, 2026-04-01 -->` |

标注位置：放在被更新内容的紧上方，不要放在文件开头或离内容很远的位置。

## 完整工作流示例

### 场景 A: 定期触发

用户执行 `文档治理`，harness-doc-fix 以独立 MR 方式修复文档偏差。

```
用户: 文档治理

=== Phase 1: 扫描 ===

执行 6 项检查：

1. ARCHITECTURE.md vs 实际目录结构
   → 发现幽灵模块: packages/legacy-auth/（已删除 2 周前）
   → 发现未记录模块: packages/notification/（1 周前新增）

2. code-patterns.md vs 实际代码模式
   → 发现新未记录模式: Structured Error Handling（最近 8 个 commit 中有 5 个使用）
   → 发现过时模式: 旧的 error code 方式，代码中已无使用

3. invariants.md 路径有效性
   → INV-004 引用的 src/legacy/auth.ts 已不存在

4. integrations.md vs 实际依赖
   → 仅文档有: Sentry SDK（已从 package.json 移除）
   → 仅代码有: DataDog APM（代码中已集成但文档未记录）

5. AGENTS.md 导航完整性
   → docs/guidance/common-pitfalls.md 缺少导航条目

6. knowledge-gaps.md open 条目评估
   → KG-005 (Error handling 策略) 已被新的 Structured Error Handling 模式填补

=== Phase 2: 优先级排序 ===

| # | 优先级 | 文件 | 偏差描述 |
|---|--------|------|----------|
| 1 | 高 | ARCHITECTURE.md | 幽灵模块 packages/legacy-auth/ |
| 2 | 高 | integrations.md | Sentry SDK 已移除但文档仍在 |
| 3 | 中 | ARCHITECTURE.md | 未记录模块 packages/notification/ |
| 4 | 中 | integrations.md | DataDog APM 未记录 |
| 5 | 中 | code-patterns.md | Structured Error Handling 未记录 |
| 6 | 中 | code-patterns.md | 旧 error code 模式需标记废弃 |
| 7 | 低 | invariants.md | INV-004 路径过时 |
| 8 | 低 | AGENTS.md | common-pitfalls.md 缺导航 |
| 9 | 低 | knowledge-gaps.md | KG-005 可关闭 |

→ 展示清单，等待用户确认

=== Phase 3: 逐项修复 ===

用户确认全部 9 项。逐项执行：

1. 读取 ARCHITECTURE.md → 移除 packages/legacy-auth/ 行 → commit
2. 读取 integrations.md → 移除 Sentry SDK 行 → commit
3. 读取代码确认 notification 模块职责 → 在 ARCHITECTURE.md Code Map 补充 → commit
4. 读取 DataDog 集成代码 → 在 integrations.md 补充行 → commit
5. 读取使用 Structured Error Handling 的代码文件 → 在 code-patterns.md 新增模式 → commit
6. 在 code-patterns.md 将旧 error code 模式标记为废弃 → commit
7. 查找 INV-004 涉及的模块当前路径 → 更新 invariants.md → commit
8. 在 docs/AGENTS.md 补充 common-pitfalls.md 导航条目 → commit
9. 在 knowledge-gaps.md 将 KG-005 从 open 移至 resolved → commit

=== Phase 4: 收尾 ===

定期触发 → 创建独立分支:
  git checkout -b docs/doc-fix-2026-04-01
  git push -u origin docs/doc-fix-2026-04-01
  → 提 MR，标题: "docs: 文档治理 - 9 项偏差修复"

报告：已修复 9 项文档偏差，MR 已创建。
```

### 场景 B: ExecPlan 完成后触发

ExecPlan `implement-payment-flow` 完成后，检查知识沉淀并跟随当前分支提交。

```
ExecPlan implement-payment-flow 已完成。

=== Phase 1: 扫描（知识沉淀 6 项检查） ===

逐项评估 ExecPlan 内容：

1. 引入了新的代码模式？
   ✓ 命中：引入了 Saga 模式处理支付事务
   → 更新目标: docs/reference/code-patterns.md

2. 涉及架构决策？
   ✓ 命中：选择了 Saga 而非 Two-Phase Commit
   → 更新目标: 创建 ADR-004

3. 踩了新坑或发现调试技巧？
   ✓ 命中：发现 idempotency key 在并发场景下的 race condition
   → 更新目标: docs/guidance/common-pitfalls.md

4. 填补了已知知识缺口？
   ✗ 未命中

5. 改变了模块边界？
   ✗ 未命中（在已有 packages/payment/ 内实现）

6. 新增或修改了不变量？
   ✗ 未命中

=== Phase 2: 确认 ===

3 项需要更新：
| # | 优先级 | 更新目标 | 内容 |
|---|--------|----------|------|
| 1 | 中 | code-patterns.md | Saga 模式 |
| 2 | 中 | ADR-004 | Saga vs Two-Phase Commit |
| 3 | 中 | common-pitfalls.md | idempotency key race condition |

→ 展示清单，等待用户确认

=== Phase 3: 逐项修复 ===

用户确认。执行 3 项更新：

1. 在 code-patterns.md 的"事务处理"节新增 Saga 模式：
   <!-- 来源: ExecPlan implement-payment-flow, 2026-04-01 -->
   ### Saga 模式
   **适用场景**：跨服务的分布式事务，需要补偿回滚
   **结构**：...
   **示例文件**：packages/payment/saga/payment-saga.ts
   → commit: "docs: fix code-patterns.md - 新增 Saga 模式"

2. 创建 docs/reference/adr/ADR-004.md：
   Saga vs Two-Phase Commit 的选择
   → commit: "docs: add ADR-004 Saga vs Two-Phase Commit"

3. 在 common-pitfalls.md 新增行：
   <!-- 来源: ExecPlan implement-payment-flow, 2026-04-01 -->
   | 支付并发处理 | idempotency key 未做分布式锁导致重复扣款 | 使用 Redis SETNX 确保 key 唯一性 |
   → commit: "docs: fix common-pitfalls.md - 新增 idempotency key 并发坑点"

=== Phase 4: 收尾 ===

ExecPlan 完成后触发 → 跟随当前用户分支。
3 个 commit 已加入当前分支。

报告：已完成 3 项知识沉淀更新（code-patterns、ADR-004、common-pitfalls）。
```
