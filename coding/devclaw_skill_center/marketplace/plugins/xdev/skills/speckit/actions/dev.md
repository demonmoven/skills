# 开发

## 参数

- `FEATURE_NAME`（必需）：功能特性名称，用于定位 specs 目录

---

## 概述

本 action 编排 2 个阶段，从技术方案出发，自动完成任务拆分和逐 Phase 代码开发：

```
Spec + Constitution + 技术方案文件（5 个独立文件）
    ↓
    ↓ Step 1: task-split — 任务拆分
    ↓   产出: subtasks_1.md, subtasks_2.md, ...
    ↓
    ↓ Step 2: phase-dev — 逐 Phase 串行开发
    ↓   ├─ Phase 1: subtasks_1.md
    ↓   ├─ Phase 2: subtasks_2.md
    ↓   └─ ...
    ↓   每个 Phase 独立一个 subagent 执行
    ↓
  完成
```

---

## 入参校验

读取并执行 `<skill_dir>/prompts/resolve_feature_name.md`，传入以下参数：
- `ACTION_TYPE = "other"`
- `ARG = {用户传入的参数}`

执行完成后获得 `FEATURE_NAME`、`PRIMARY_REPO`。

推导内部变量（使用 `PRIMARY_REPO` 作为工作目录，而非 `$(pwd)`）：

```text
CWD = {PRIMARY_REPO}
SPECS_DIR = {CWD}/docs/xdev/speckit/{FEATURE_NAME}
TARGET_SRC_DIR = {CWD}
SPEC_DOC = {SPECS_DIR}/spec.md
CONSTITUTION_DOC = {CWD}/constitution/constitution.md
RESEARCH_DOC = {SPECS_DIR}/research.md
MINING_DOC = {SPECS_DIR}/mining.md
DATA_MODEL_DOC = {SPECS_DIR}/data-model.md
CONTRACTS_DOC = {SPECS_DIR}/contracts.md
CONFIGURATION_DOC = {SPECS_DIR}/configuration.md
INTEGRATION_DOC = {SPECS_DIR}/integration.md
PLAN_DOC = {SPECS_DIR}/plan.md
```

继续校验：

1. `SPEC_DOC`（必需）文件存在
   - 不存在 → 提示用户："`SPEC_DOC` 文件不存在：{路径}，请确认 spec.md 文件是否已生成"
2. `CONSTITUTION_DOC`（必需）文件存在
   - 不存在 → 提示用户："`CONSTITUTION_DOC` 文件不存在：{路径}，请在项目根目录下准备 constitution/constitution.md 文件"
3. `RESEARCH_DOC`（必需）文件存在
   - 不存在 → 提示用户："`RESEARCH_DOC` 文件不存在：{路径}，请确认 research.md 文件是否已生成"
4. `MINING_DOC`（必需）文件存在
   - 不存在 → 提示用户："`MINING_DOC` 文件不存在：{路径}，请确认 mining.md 文件是否已生成"
5. 以下 5 个技术方案文件（必需）全部存在：
   - `DATA_MODEL_DOC` — 不存在 → 提示用户："`DATA_MODEL_DOC` 文件不存在：{路径}，请确认 data-model.md 文件是否已生成"
   - `CONTRACTS_DOC` — 不存在 → 提示用户："`CONTRACTS_DOC` 文件不存在：{路径}，请确认 contracts.md 文件是否已生成"
   - `CONFIGURATION_DOC` — 不存在 → 提示用户："`CONFIGURATION_DOC` 文件不存在：{路径}，请确认 configuration.md 文件是否已生成"
   - `INTEGRATION_DOC` — 不存在 → 提示用户："`INTEGRATION_DOC` 文件不存在：{路径}，请确认 integration.md 文件是否已生成"
   - `PLAN_DOC` — 不存在 → 提示用户："`PLAN_DOC` 文件不存在：{路径}，请确认 plan.md 文件是否已生成"

全部校验通过后，进入执行阶段。

## 输出参数

1. 开发完成的代码仓库（当前工作目录），每个 Phase 完成后自动 commit + push

---

## 执行流程

> **`<skill_dir>` 约定**：以下所有路径中 `<skill_dir>` 指代本 action 所属的 speckit skill 目录的绝对路径。

### 子任务执行方式

所有子任务统一通过 **Agent 工具**（`subagent_type="general-purpose"`）执行。每个 Agent 的 prompt 指示 subagent 读取对应的 prompt 文件并直接执行其中的指令。

### Step 1：task-split — 任务拆分

串行执行，等待 Agent 返回后再进入下一步：

```
Agent(subagent_type="general-purpose", prompt="
你需要执行一个任务拆分子任务。

以下是你的输入变量（均为绝对路径）：
SPEC_DOC={SPEC_DOC}
CONSTITUTION_DOC={CONSTITUTION_DOC}
RESEARCH_DOC={RESEARCH_DOC}
MINING_DOC={MINING_DOC}
DATA_MODEL_DOC={DATA_MODEL_DOC}
CONTRACTS_DOC={CONTRACTS_DOC}
CONFIGURATION_DOC={CONFIGURATION_DOC}
INTEGRATION_DOC={INTEGRATION_DOC}
PLAN_DOC={PLAN_DOC}
SPECS_DIR={SPECS_DIR}
TASKS_TEMPLATE_DOC=<skill_dir>/resources/subtasks_template.md

请读取并严格按照以下指令文件执行：<skill_dir>/prompts/task_split_prompt.md

以下参数已全部校验通过，直接使用，无需校验、无需向用户提问。
执行完成后，报告产出文件路径。
")
```

**产出**：`{SPECS_DIR}/subtasks_1.md`、`{SPECS_DIR}/subtasks_2.md`、...

> **等待完成**：Agent 返回后，检查 `{SPECS_DIR}/` 目录下是否存在至少一个 `subtasks_*.md` 文件。如果不存在，记录失败原因后停止执行。

### Step 2：phase-dev — 逐 Phase 串行开发

**2a. 扫描 Phase 列表**

扫描 `{SPECS_DIR}/` 目录下所有 `subtasks_*.md` 文件，提取 Phase 编号并按编号排序：

```bash
ls {SPECS_DIR}/subtasks_*.md 2>/dev/null | sort -t_ -k2 -n
```

记录：
- `{phase_numbers}`：排序后的 Phase 编号列表（如 [1, 2, 3]）
- `{phase_count}`：Phase 总数

**2b. 智能续接判断**

对每个 `subtasks_N.md` 文件，检查其内容判断状态：

1. 读取文件内容，检查是否存在"## 开发状态"章节：
   - 如果有，且包含"✅ 已完成" → 标记为**已完成**，跳过
   - 如果有，且包含"❌ 未完成" → 标记为**待续接**
   - 如果没有"## 开发状态" → 标记为**未开始**

2. 从编号最小的"待续接"或"未开始"的 Phase 开始执行

3. 跳过所有"已完成"的 Phase

记录：`{start_phase}`（从哪个 Phase 开始执行）

**2c. 逐 Phase 执行（核心循环）**

从 `{start_phase}` 开始，对后续每个未完成的 Phase `{phase_number}` **串行**执行以下步骤：

**调用 phase-dev subagent**：

```
Agent(subagent_type="general-purpose", prompt="
你需要执行一个开发子任务。

以下是你的输入变量（均为绝对路径）：
TASK_PHASE=subtasks_{phase_number}.md
SPECS_DIR={SPECS_DIR}
SPEC_DOC={SPEC_DOC}
CONSTITUTION_DOC={CONSTITUTION_DOC}
RESEARCH_DOC={RESEARCH_DOC}
MINING_DOC={MINING_DOC}
DATA_MODEL_DOC={DATA_MODEL_DOC}
CONTRACTS_DOC={CONTRACTS_DOC}
CONFIGURATION_DOC={CONFIGURATION_DOC}
INTEGRATION_DOC={INTEGRATION_DOC}
PLAN_DOC={PLAN_DOC}

请读取并严格按照以下指令文件执行：<skill_dir>/prompts/phase_dev_prompt.md

以下参数已全部校验通过，直接使用，无需校验、无需向用户提问。
执行完成后，报告产出文件路径。
")
```

**解析返回结果**：

Agent 完成后，读取 `{SPECS_DIR}/subtasks_{phase_number}.md` 文件中的"## 开发状态"章节，判断结果：

- **成功完成**（包含"✅ 已完成"）→ 继续下一个 Phase
- **阻塞性询问**（包含"❌ 未完成"）→ 进入下方的**阻塞问题解答流程**
- **异常失败**（无开发状态 或 Agent 本身报错）→ 记录失败原因，**停止执行**

> **注意**：每个 Phase 必须等待上一个 Phase 完成后才能开始。Phase 之间严格串行，不可并行。

**阻塞问题解答流程**（交互式）：

当 phase-dev 返回阻塞时，主 Agent 负责向用户转问并收集回答，更新 subtasks 文件后重新续接开发：

1. **提取阻塞问题**：从 `{SPECS_DIR}/subtasks_{phase_number}.md` 的"## 开发状态"章节中，提取"**阻塞性问题**"列表

2. **向用户转问**：将阻塞问题转述给用户，格式如下：
   ```
   Phase {phase_number} 开发过程中遇到了以下需要您确认的问题：

   {阻塞性问题列表，原样展示}

   请针对以上问题给出您的回答或确认。
   ```

3. **交互式回答循环**：用户会逐条或批量回答问题，每次收到用户回答后：
   - **更新 subtasks 文件**：将用户的回答追加到 `{SPECS_DIR}/subtasks_{phase_number}.md` 的"## 开发状态"章节中"**阻塞性问题**"下方，格式为：
     ```markdown
     **用户回答**：
     1. 回答一：xxx
     2. 回答二：yyy
     ```
   - **询问是否还有补充**：`还有其他需要补充的吗？确认无误后我将继续开发。`

4. **用户确认后续接开发**：用户表示确认后，重新调用 phase-dev subagent 执行同一个 Phase（续接模式，phase-dev prompt 中的续接开发检查会自动读取用户回答并继续）。回到 Step 2c 的核心循环继续处理。

### Step 3：报告结果

输出执行摘要：
- 各 Phase 执行状态（成功/阻塞/失败/跳过）
- 最终完成的 Phase 数量

---

## 产出物

| 产出文件 | 何时产出 | 说明 |
|---------|---------|------|
| `subtasks_1.md` ... `subtasks_N.md` | Step 1 | 各 Phase 的任务拆分文档 |
| 代码仓库 commits | Step 2 | 每个 Phase 完成后自动 commit + push |
