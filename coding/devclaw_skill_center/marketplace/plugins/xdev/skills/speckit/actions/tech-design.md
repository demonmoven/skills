# 技术方案设计

## 参数

- `FEATURE_NAME`（必需）：功能特性名称，用于定位 specs 目录

---

## 概述

本 action 编排技术方案设计的全流程，从 Spec 文档和代码库出发，自动生成完整的技术方案设计文档：

```
Spec + Tech Guidance + 代码仓库
    ↓
    ↓ 并发组 A（×3）
    ↓   ├─ analyze    — 代码库现状分析
    ↓   ├─ research   — 技术方案调研
    ↓   └─ mining     — 隐性需求挖掘
    ↓   产出: analyze.md, research.md, mining.md
    ↓
    ↓ mining-diff — Mining 差异化发现提取
    ↓   产出: mining-raw.md（原 mining.md 重命名）, mining.md（差异化版）
    ↓
    ↓ plan — 架构设计 + 技术实现方案（串行执行 5 个子任务）
    ↓   ├─ data-model     — 数据模型设计
    ↓   ├─ contracts      — API 契约设计
    ↓   ├─ configuration  — 配置设计
    ↓   ├─ integration    — 系统集成设计
    ↓   └─ plan           — 技术实现方案
    ↓   产出: data-model.md, contracts.md, configuration.md, integration.md, plan.md
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
TECH_GUIDANCE_DOC = {SPECS_DIR}/tech-guidance.md
```

继续校验：

1. `SPEC_DOC`（必需）文件存在
   - 不存在 → 提示用户："`SPEC_DOC` 文件不存在：{路径}，请确认 spec.md 文件是否已生成"
2. `TECH_GUIDANCE_DOC`（可选）文件是否存在
   - 存在 → 记录路径，后续传入各子任务
   - 不存在 → 跳过，不报错，后续传入空值
3. `CONSTITUTION_DOC`（必需）文件存在
   - 不存在 → 提示用户："`CONSTITUTION_DOC` 文件不存在：{路径}，请在项目根目录下准备 constitution/constitution.md 文件"

全部校验通过后，进入执行阶段。

## 输出参数

1. 各独立技术方案文件：`data-model.md`、`contracts.md`、`configuration.md`、`integration.md`、`plan.md`

---

## 执行流程

> **`<skill_dir>` 约定**：以下所有路径中 `<skill_dir>` 指代本 action 所属的 speckit skill 目录的绝对路径。

### 子任务执行方式

所有子任务统一通过 **Agent 工具**（`subagent_type="general-purpose"`）执行。每个 Agent 的 prompt 指示 subagent 读取对应的 prompt 文件并直接执行其中的指令。

**Agent prompt 模板**：

```
你需要执行一个技术设计子任务。

以下是你的输入变量（均为绝对路径）：
{KEY_VALUE_PAIRS}

请读取并严格按照以下指令文件执行：{PROMPT_FILE}

以下参数已全部校验通过，直接使用，无需校验、无需向用户提问。
执行完成后，报告产出文件路径。
```

### Step 1：并发组 A — 前期调研（×3）

使用 Agent 工具**在同一条消息中**同时发起以下 3 个子任务，实现并行执行。3 个 Agent 调用必须放在同一条消息中发送，以确保真正并发：

**1a. analyze — 代码库现状分析**

```
Agent(subagent_type="general-purpose", prompt="
你需要执行一个技术设计子任务。

以下是你的输入变量（均为绝对路径）：
SPEC_DOC={SPEC_DOC}
TECH_GUIDANCE_DOC={TECH_GUIDANCE_DOC}
CONSTITUTION_DOC={CONSTITUTION_DOC}
TARGET_SRC_DIR={TARGET_SRC_DIR}
ANALYZE_DOC={SPECS_DIR}/analyze.md
CODEBASE_LOCATOR_AGENT_DOC=<skill_dir>/agents/codebase-locator.md
CODEBASE_ANALYZER_AGENT_DOC=<skill_dir>/agents/codebase-analyzer.md
CODEBASE_PATTERN_FINDER_AGENT_DOC=<skill_dir>/agents/codebase-pattern-finder.md

请读取并严格按照以下指令文件执行：<skill_dir>/prompts/analyze_prompt.md

以下参数已全部校验通过，直接使用，无需校验、无需向用户提问。
执行完成后，报告产出文件路径。
")
```

**产出**：`{SPECS_DIR}/analyze.md`

**1b. research — 技术方案调研**

```
Agent(subagent_type="general-purpose", prompt="
你需要执行一个技术设计子任务。

以下是你的输入变量（均为绝对路径）：
SPEC_DOC={SPEC_DOC}
TECH_GUIDANCE_DOC={TECH_GUIDANCE_DOC}
CONSTITUTION_DOC={CONSTITUTION_DOC}
TARGET_SRC_DIR={TARGET_SRC_DIR}
RESEARCH_DOC={SPECS_DIR}/research.md

请读取并严格按照以下指令文件执行：<skill_dir>/prompts/research_prompt.md

以下参数已全部校验通过，直接使用，无需校验、无需向用户提问。
执行完成后，报告产出文件路径。
")
```

**产出**：`{SPECS_DIR}/research.md`

**1c. mining — 隐性需求挖掘**

```
Agent(subagent_type="general-purpose", prompt="
你需要执行一个技术设计子任务。

以下是你的输入变量（均为绝对路径）：
SPEC_DOC={SPEC_DOC}
TECH_GUIDANCE_DOC={TECH_GUIDANCE_DOC}
CONSTITUTION_DOC={CONSTITUTION_DOC}
TARGET_SRC_DIR={TARGET_SRC_DIR}
MINING_DOC={SPECS_DIR}/mining.md

请读取并严格按照以下指令文件执行：<skill_dir>/prompts/mining_prompt.md

以下参数已全部校验通过，直接使用，无需校验、无需向用户提问。
执行完成后，报告产出文件路径。
")
```

**产出**：`{SPECS_DIR}/mining.md`

> **等待全部完成**：3 个 Agent 全部返回后，再进入 Step 2。如果某个子任务失败，记录失败原因后继续执行后续步骤。

### Step 2：mining-diff — Mining 差异化发现提取

串行执行，等待 Agent 返回后再进入下一步：

```
Agent(subagent_type="general-purpose", prompt="
你需要执行一个技术设计子任务。

以下是你的输入变量（均为绝对路径）：
SPEC_DOC={SPEC_DOC}
ANALYZE_DOC={SPECS_DIR}/analyze.md
RESEARCH_DOC={SPECS_DIR}/research.md
MINING_DOC={SPECS_DIR}/mining.md
MINING_DIFF_DOC={SPECS_DIR}/mining-diff.md

请读取并严格按照以下指令文件执行：<skill_dir>/prompts/mining_diff_prompt.md

以下参数已全部校验通过，直接使用，无需校验、无需向用户提问。
执行完成后，报告产出文件路径。
")
```

**产出**：`{SPECS_DIR}/mining-diff.md`

Agent 返回后，执行文件重命名：

```bash
mv "{SPECS_DIR}/mining.md" "{SPECS_DIR}/mining-raw.md" && mv "{SPECS_DIR}/mining-diff.md" "{SPECS_DIR}/mining.md"
```

### Step 3：plan — 架构设计 + 技术实现方案

串行执行，由一个 Agent 内部按顺序依次执行 5 个 prompt。等待 Agent 返回后进入报告：

```
Agent(subagent_type="general-purpose", prompt="
你需要按顺序执行 5 个技术设计子任务。每个子任务有对应的指令文件，请逐一读取并执行。

以下是你的输入变量（均为绝对路径）：
SPEC_DOC={SPEC_DOC}
TECH_GUIDANCE_DOC={TECH_GUIDANCE_DOC}
CONSTITUTION_DOC={CONSTITUTION_DOC}
TARGET_SRC_DIR={TARGET_SRC_DIR}
ANALYZE_DOC={SPECS_DIR}/analyze.md
RESEARCH_DOC={SPECS_DIR}/research.md
MINING_DOC={SPECS_DIR}/mining.md
DATA_MODEL_DOC={SPECS_DIR}/data-model.md
CONTRACTS_DOC={SPECS_DIR}/contracts.md
CONTRACTS_STANDARD_DOC=<skill_dir>/resources/contracts_standard.md
CONFIGURATION_DOC={SPECS_DIR}/configuration.md
INTEGRATION_DOC={SPECS_DIR}/integration.md
PLAN_DOC={SPECS_DIR}/plan.md

请按以下顺序逐一执行（每个任务完成后再执行下一个）：

1. 读取并执行 <skill_dir>/prompts/data_model_prompt.md → 产出 DATA_MODEL_DOC
2. 读取并执行 <skill_dir>/prompts/contracts_prompt.md → 产出 CONTRACTS_DOC
3. 读取并执行 <skill_dir>/prompts/configuration_prompt.md → 产出 CONFIGURATION_DOC
4. 读取并执行 <skill_dir>/prompts/integration_prompt.md → 产出 INTEGRATION_DOC
5. 读取并执行 <skill_dir>/prompts/plan_prompt.md → 产出 PLAN_DOC

以下参数已全部校验通过，直接使用，无需校验、无需向用户提问。
每个子任务执行完成后，报告产出文件路径，然后继续下一个。
全部完成后，汇报所有产出文件路径。
")
```

**产出**：
- `{SPECS_DIR}/data-model.md`
- `{SPECS_DIR}/contracts.md`
- `{SPECS_DIR}/configuration.md`
- `{SPECS_DIR}/integration.md`
- `{SPECS_DIR}/plan.md`

### Step 4：报告结果

输出生成摘要：
- 各子任务执行状态（成功/失败）
- 最终产出文件路径列表

---

## 产出物

| 产出文件 | 何时产出 | 说明 |
|---------|---------|------|
| `analyze.md` | Step 1 | 代码库现状分析文档 |
| `research.md` | Step 1 | 技术方案调研文档 |
| `mining-raw.md` | Step 2 | 隐性需求挖掘原始文档（重命名自 mining.md） |
| `mining.md` | Step 2 | Mining 差异化发现文档（仅保留独特发现） |
| `data-model.md` | Step 3 | 数据模型设计文档 |
| `contracts.md` | Step 3 | API 接口契约文档 |
| `configuration.md` | Step 3 | 配置设计文档 |
| `integration.md` | Step 3 | 系统集成设计文档 |
| `plan.md` | Step 3 | 技术实现方案文档 |
