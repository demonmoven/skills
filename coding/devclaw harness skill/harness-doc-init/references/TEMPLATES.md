# 文档模板

本文件提供 harness-init 生成的每类文档的标准模板。
`{{variable}}` 表示需要从代码分析中填入的值，`{{?section}}...{{/section}}` 表示按条件包含的章节。

---

## 1. 根 AGENTS.md 模板

```markdown
# AGENTS.md — 仓库文档总入口

> **本文件是 {{project_name}} 仓库的总路由，不是百科全书。**
> Agent 默认从这里开始；人类读者可先看 `README.md`，再回到这里继续下钻。

{{?language_note}}
- {{language_note}}
{{/language_note}}

---

## 如何使用本文件

- Agent：先读本文件，确定要进入哪个模块入口或专题文档
- 人类读者：先读 `README.md` 了解项目，再用本文件或 `docs/AGENTS.md` 找模块和专题文档
- 需要代码结构和边界：读 `ARCHITECTURE.md`
- 需要各模块的本地规则：进入对应模块 `AGENTS.md`
{{?has_docs}}
- 需要专题文档、操作手册、参考资料：进入 `docs/`
{{/has_docs}}

---

## 项目简介

{{project_description}}

当前仓库的核心资产：

{{module_list}}

---

{{?has_docs}}
## 文档体系

| 层级 | 入口 | 作用 |
|---|---|---|
| L0 总入口 | `AGENTS.md` | 仓库任务路由、协作约束、阅读顺序 |
| L0 人类入口 | `README.md` | 项目介绍、快速开始 |
{{?has_architecture}}
| L0 架构入口 | `ARCHITECTURE.md` | 代码地图、边界、不变量 |
{{/has_architecture}}
{{?has_docs_index}}
| L1 中层索引 | `docs/AGENTS.md` | 专题文档、模块入口汇总 |
{{/has_docs_index}}
| L2 模块入口 | {{module_agents_list}} | 各模块的本地入口 |
{{?has_reference_guidance}}
| L3 专题知识库 | `docs/reference/`、`docs/guidance/` | 事实说明、操作手册 |
{{/has_reference_guidance}}
{{?has_plans}}
| L3 设计决策 | `docs/plans/` 或 `docs/adr/` | 设计决策记录与执行计划 |
{{/has_plans}}

---
{{/has_docs}}

## 知识导航（Progressive Disclosure）

根据你要做的事，按需查阅：

| 我要做什么 | 去哪里看 |
|---|---|
| 我是第一次进入仓库，先理解项目是什么 | `README.md` |
{{?has_architecture}}
| 我想快速理解整体架构与代码边界 | `ARCHITECTURE.md` |
{{/has_architecture}}
{{navigation_rows}}

---

## 核心约束（所有 Agent 必须遵守）

{{constraints}}

---

{{?has_hooks}}
## 代码质量回压（Backpressure）

{{hook_description}}
{{/has_hooks}}
```

---

## 2. ARCHITECTURE.md 模板

```markdown
# {{project_name}} Architecture

> 本文档只负责解释代码架构、模块边界和不变量，不承担完整文档导航职责。
> 仓库总入口见 `AGENTS.md`，人类阅读入口见 `README.md`。

## Bird's Eye View

{{birds_eye_description}}

```text
{{architecture_diagram}}
```

## Code Map

本节简要介绍各重要目录和数据结构。
注意 **Architecture Invariant** 部分——它们通常描述代码中**刻意不存在**的东西。

### `{{module_path}}/` — {{module_description}}

{{module_details}}

**Architecture Invariant:** {{invariant}}

{{repeat_for_each_module}}

## Cross-Cutting Concerns

### {{concern_name}}

{{concern_description}}
```

---

## 3. 模块 AGENTS.md 模板

```markdown
# AGENTS.md — {{module_path}}/ {{module_description}}

> 本文件是 `{{module_path}}/` 模块的本地入口。
> 仓库总入口见 `{{root_ref}}/AGENTS.md`，架构边界见 `{{root_ref}}/ARCHITECTURE.md`。

## 模块定位

{{module_positioning}}

## 知识导航

| 我要做什么 | 去哪里看 |
|---|---|
{{module_navigation_rows}}

## 目录结构

```text
{{directory_tree}}
```

## 本地约束

{{local_constraints}}

## 常用命令

```bash
{{commands}}
```

{{?has_key_files}}
## 关键文件

{{key_files_list}}
{{/has_key_files}}

{{?has_further_reading}}
## 进一步阅读

{{further_reading_links}}
{{/has_further_reading}}
```

---

## 4. docs/AGENTS.md 模板

```markdown
# {{project_name}} 文档索引

> 本目录承接仓库专题文档和中层导航。
> Agent 总入口见 `../AGENTS.md`，人类入口见 `../README.md`。

## 1. 阅读顺序

- Agent：`../AGENTS.md` -> 本页 -> 模块入口 / 专题文档
- 人类：`../README.md` -> 本页 -> `../ARCHITECTURE.md` 或对应模块文档
- 需要代码边界：直接看 `../ARCHITECTURE.md`

## 2. 文档分层

| 层级 | 位置 | 用途 |
|---|---|---|
| 总入口 | `../AGENTS.md` | 仓库任务路由、协作约束 |
| 人类入口 | `../README.md` | 项目介绍、快速开始 |
{{?has_architecture}}
| 架构入口 | `../ARCHITECTURE.md` | 代码地图、边界、不变量 |
{{/has_architecture}}
| 模块入口 | {{module_refs}} | 进入具体模块 |
| 参考资料 | `reference/` | 稳定事实说明 |
| 操作手册 | `guidance/` | SOP、联调与验证流程 |
{{?has_plans}}
| 设计决策 | `plans/` 或 `adr/` | 设计决策记录与执行计划 |
{{/has_plans}}

## 3. 按任务导航

| 场景 | 文档 |
|---|---|
{{task_navigation_rows}}

## 4. 当前专题文件

```text
{{docs_file_tree}}
```

## 5. 维护规则

- 模块入口型文档保留在模块根目录，统一使用 `AGENTS.md`
- 所有操作手册、事实说明统一收敛到 `docs/guidance/` 或 `docs/reference/`
{{?has_plans}}
- `plans/` / `adr/` 由设计决策流程单独维护
{{/has_plans}}
- 如果模块是独立发布的包（npm/crate/PyPI），可同时保留 README.md 作为包发布元数据
```

---

## 5. 约束章节模板

以下是核心约束的典型写法参考，按需选用：

### 生成代码约束（适用于有代码生成的项目）

```markdown
### 生成代码与路由文件

- 对标注 `DO NOT EDIT` 的生成文件，默认不直接修改
- 涉及 IDL / protobuf / swagger 产物时，优先通过生成流程更新
- 若必须改生成相关代码，先确认该文件是否为"可保留自定义逻辑"的入口
```

### 文件与函数长度约束

```markdown
### 文件与函数长度

- 代码文件行数上限：`{{max_file_lines}}`（由 {{checker}} 检查）
- {{language}} 函数长度上限：`{{max_func_lines}}` 行（由 {{linter}} 检查，测试文件豁免）
```

### 测试约束

```markdown
### 测试与覆盖率

- 以 `{{coverage_script}}` 输出为准
- 任何测试命令中出现 `SKIP` 都应视为异常信号，不得按"测试通过"处理
- Agent 必须在反馈中明确报告被跳过的测试项和跳过原因
```

### 依赖方向约束

```markdown
### 模块依赖方向

- `{{shared_layer}}` 不能反向依赖 `{{business_modules}}`
- 依赖方向：{{dependency_direction}}
- 禁止跨模块"横向抄逻辑"，共用逻辑先下沉到共享层
```

---

## 6. Hook 回压章节模板

```markdown
## 代码质量回压（Backpressure）

本项目通过 pre-commit hooks 保证提交质量。

**首次 clone 或 hook 缺失时**，执行：

```bash
{{hook_install_command}}
```

当前检查项：

| 检查项 | 说明 |
|--------|------|
{{hook_table_rows}}
```
