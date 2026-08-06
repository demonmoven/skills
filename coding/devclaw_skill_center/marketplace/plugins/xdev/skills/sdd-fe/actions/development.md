---
name: development
description: 按照 subtasks 文档执行前端开发任务。Use when user asks to 开始开发, 执行开发, run development, implement phase, 实现阶段, start coding.
---

# 用户 args 解析

优先从 args 里解析相应的参数，如果不存在，则按以下规则处理：

1. **SpecsDir**：plan.md、feature_spec.md、subtasks_*.md 所在目录。用户未提供时**必须询问**。
2. **PhaseNumber**（可选）：要执行第几个 Phase。用户未提供时**进入全量开发模式**，自动扫描所有 Phase 连续执行，无需询问。
3. **RepoDir**：前端代码仓库路径。用户未提供时，**默认使用当前工作目录（`pwd`）**，无需询问。
4. **ConstitutionFile**：constitution 规约文档文件路径。用户未提供时**必须询问**。

> SpecsDir 和 ConstitutionFile 每个变量单独询问，确保用户明确输入后再继续下一步。

# Development

读取 plan.md、feature_spec.md、subtasks_{N}.md 等文档，按阶段执行前端开发任务。

## 参数

| 参数 | 必填 | 说明 | 默认值 | 示例 |
|------|------|------|--------|------|
| `SpecsDir` | ✅ 必填 | 规格文档目录（含 plan.md、feature_spec.md、subtasks_*.md） | — | `/path/to/specs` |
| `PhaseNumber` | ❌ 选填 | 要执行的阶段编号；省略则自动扫描全部 Phase 连续执行 | 全量模式 | `1` |
| `RepoDir` | ❌ 选填 | 前端代码仓库路径，默认为当前工作目录（`pwd`） | 当前工作目录（`pwd`） | `/path/to/frontend-repo` |
| `ConstitutionFile` | ✅ 必填 | constitution 规约文档文件路径 | — | `/path/to/constitution.md` |

### 参数校验

- `SpecsDir` 和 `ConstitutionFile` 为**必填**，用户未提供时**必须询问获取**。
- `PhaseNumber` 为**选填**，未提供时进入**全量开发模式**。
- `RepoDir` 为**选填**，用户未提供时默认使用当前工作目录（`pwd`）。
- 读取 `{ConstitutionFile}` 作为规约。

### 询问模板

如果用户未提供必填参数，询问：

```
需要以下信息来执行开发任务：
1. 📂 规格文档目录 (SpecsDir) — plan.md、feature_spec.md、subtasks_*.md 所在目录？
2. 📜 规约文档路径 (ConstitutionFile) — constitution 规约文档文件路径？

💡 PhaseNumber 未提供，将自动扫描全部 Phase 连续执行
💡 RepoDir 未提供，将默认使用当前工作目录：{pwd}
```

---

## 执行模式判定

参数齐全后，根据 `PhaseNumber` 是否提供决定执行模式：

| PhaseNumber | 模式 | 行为 |
|-------------|------|------|
| 已提供（如 `1`） | 单阶段模式 | 只执行指定 Phase |
| 未提供 | **全量开发模式** | 扫描 SpecsDir 内所有 `subtasks_{N}.md`，列出全部 Phase，制定 todolist，连续执行所有 Phase |

---

## 全量开发模式（PhaseNumber 未提供时）

### 1. 扫描 Phase 列表

在 `{SpecsDir}` 目录下扫描所有匹配 `subtasks_*.md` 的文件，提取 Phase 编号并**按编号升序排列**。

示例：若发现 `subtasks_1.md`、`subtasks_2.md`、`subtasks_3.md`，则 Phase 列表为 `[1, 2, 3]`。

### 2. 制定 todolist

根据扫描结果，使用 TodoWrite 工具创建开发 todolist，格式为：

```
- Phase 1: {从 subtasks_1.md 标题或首行提取的 Phase 描述}
- Phase 2: {从 subtasks_2.md 标题或首行提取的 Phase 描述}
- Phase 3: {从 subtasks_3.md 标题或首行提取的 Phase 描述}
...
```

### 3. 连续执行所有 Phase

按 Phase 编号**从小到大依次执行**，每个 Phase 执行完整的「单阶段执行流程」（见下方）。
- 完成一个 Phase 后，立即标记该 Phase 的 todo 为 completed
- **不等待用户确认**，直接继续下一个 Phase
- 全部 Phase 完成后，输出整体总结

---

## 单阶段执行流程（子任务内执行）

> 无论是单阶段模式还是全量模式，每个 Phase 都按此流程执行。

### 1. 加载上下文

- **必须读取**：`{SpecsDir}/subtasks_{PhaseNumber}.md` — 当前阶段完整任务列表
- **必须读取**：`{SpecsDir}/plan.md` — 技术方案、组件实现方案
- **必须读取**：`{SpecsDir}/feature_spec.md` — 需求规格和 User Stories
- **必须读取**：`{ConstitutionFile}` — 作为规约
- **按需读取**：API 文档（如当前阶段涉及 API）

### 2. 解析任务结构

从 `subtasks_{PhaseNumber}.md` 中识别 Phase 类型：
- **i18n**：国际化文案处理
- **API**：API Schema 更新与类型定义
- **UI 代码迁移**：将 UI 预览代码迁移至业务包
- **组件功能实现**：在 UI 代码基础上添加业务逻辑
- **User Story 集成**：按用户故事进行页面集成

### 3. 按任务计划逐个执行

- **逐任务执行**：按 `subtasks_{PhaseNumber}.md` 定义的顺序逐个实现，不跳过、不并行
- **直接当作 todo**：subtasks 中的 tasks 直接当 todo 依次实现，不额外生成 todo
- **标记完成**：完成每个 task 后，将 `- [ ]` 改为 `- [X]`
- **逐任务提交**：每完成一个 task 的代码实现后，**必须立即执行 git commit**，规则如下：
  1. 先 `git add` 本次 task 涉及的所有变更文件（在 `{RepoDir}` 目录下执行）
  2. 执行 `git commit`，commit message 格式：`feat({feature}): {TaskID} {task 简述}`
     - `{feature}`：从 subtasks 文档标题或 plan.md 中提取的需求/功能名称（kebab-case）
     - `{TaskID}`：当前 task 的编号，如 `T001`、`T002`
     - `{task 简述}`：当前 task 的核心描述，简明扼要（英文）
     - 示例：`feat(prompt-debug): T001 update API schema for debug endpoints`
     - 示例：`feat(prompt-debug): T003 implement DebugPanel component skeleton`
  3. **不要使用 `git push`**，只做本地 commit
  4. 如果某个 task 无代码变更（如纯验证类 task），则跳过 commit，在标记完成时注明「无代码变更，跳过 commit」

### 4. 代码实现规则

- **context 召回**：根据 task 描述中的引用（行号、文件路径），从 `{SpecsDir}` 目录下召回相关内容
- **文案来源**：只能使用 spec 文档中的文案，不允许自己生成
- **[US*] 标记**：包含 `[US*]` 的是集成类任务，需使用已开发完成的组件/API/hooks 集成
- **[D2C] 标记**：包含 `[D2C]` 的必须使用 ui-coder subagent 实现 UI 布局

### 5. Phase 完成后验证

- 确认所有任务均已完成（所有 `- [ ]` 已改为 `- [X]`）
- 检查实现是否符合 plan.md 和 feature_spec.md 规格要求

### 6. 生成实现总结

在 `{SpecsDir}/subtasks_{PhaseNumber}_implement_summary.md` 文件中追加该 Phase 的实现总结：

- **组件开发类 Phase**：组件实现情况、预览 URL、人工测试关注点
- **User Story 集成类 Phase**：实现情况、变更文件列表、工程规范检查、人工验收链路
- **API/i18n 类 Phase**：实现概要、变更清单、测试验证方式

## 🚫 禁止事项

| # | 禁止事项 | 正确做法 |
|---|----------|----------|
| 1 | 跳过 task 或并行实现 | 严格按顺序逐个执行 |
| 2 | 自己额外生成 todo | 直接用 subtasks 中的 tasks |
| 3 | 自己编造文案 | 只用 spec 文档中的文案 |
| 4 | 单阶段模式下进入下一阶段 | 单阶段模式只完成指定 PhaseNumber |
| 5 | 凭记忆使用基础组件 API | 先查询组件详情再使用 |
| 6 | 全量模式下等待用户确认才继续下一 Phase | 全量模式必须自动连续执行，完成一个立即开始下一个 |

## 输出说明

| 输出文件 | 说明 |
|----------|------|
| 代码变更 | 在 `{RepoDir}` 中的实际代码修改 |
| `{SpecsDir}/subtasks_{PhaseNumber}.md` | 更新 task 完成状态（每个 Phase 各自更新） |
| `{SpecsDir}/subtasks_{PhaseNumber}_implement_summary.md` | Phase 实现总结（每个 Phase 各自生成） |

## 完成总结（必须）

### 单阶段模式

Phase 开发完毕后，**必须输出一份简短总结**：

```
✅ Phase {PhaseNumber} 开发完成

📂 代码仓库：{RepoDir}
📋 任务完成：{已完成数}/{总任务数}
📝 实现总结：{SpecsDir}/subtasks_{PhaseNumber}_implement_summary.md

变更概要：
  - 新增文件：X 个
  - 修改文件：Y 个
  - 核心功能：{功能描述}
```

### 全量开发模式

所有 Phase 开发完毕后，**必须输出一份整体总结**：

```
✅ 全部 Phase 开发完成（共 {总Phase数} 个阶段）

📂 代码仓库：{RepoDir}
📋 各阶段完成情况：
  - Phase 1: ✅ {已完成数}/{总任务数} — {功能描述}
  - Phase 2: ✅ {已完成数}/{总任务数} — {功能描述}
  - Phase 3: ✅ {已完成数}/{总任务数} — {功能描述}
  ...

📝 实现总结文件：
  - {SpecsDir}/subtasks_1_implement_summary.md
  - {SpecsDir}/subtasks_2_implement_summary.md
  - {SpecsDir}/subtasks_3_implement_summary.md
  ...

变更概要：
  - 新增文件：X 个
  - 修改文件：Y 个
  - 核心功能：{整体功能描述}
```

# 直接开始下一步

## 单阶段模式 → 引导下一步

### 情况1：还有下一个 Phase → 继续 development

直接按照下面的指令开始后续的工作！

```prompt
执行下一步 devclaw-sdd-fe development
  SpecsDir: {SpecsDir}
  PhaseNumber: {PhaseNumber+1}
  RepoDir: {RepoDir}
  ConstitutionFile: {ConstitutionFile}
```

### 情况2：所有 Phase 都完成了

引导语模板（必须原样输出，仅替换花括号变量）：

```
🚀 所有 Phase 开发完成！
```

## 全量开发模式

全量开发完成后，引导语模板（必须原样输出，仅替换花括号变量）：

```
🚀 所有 Phase 开发完成！
```
