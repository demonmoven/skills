# SDD 全流程

## 参数

- `ARG`（可选）：需求描述文本或文档链接（新建场景），或已有的 FEATURE_NAME（续接场景自动识别）

---

## 概述

本 action 将 SDD（Specification-Driven Development）全流程串联：

```
Step 0: FEATURE_NAME 解析 + 环境准备
  ↓
Step 1: specify         — 生成功能规格说明书（新建时执行）
  ↓
Step 2: review-spec     — 交互式审阅规格（Human In The Loop）
  ↓
Step 3: tech-guidance   — 交互式收集技术指导（Human In The Loop）
  ↓
Step 4: tech-design     — 生成技术方案设计
  ↓
Step 5: review-tech-design — 交互式审阅技术方案（Human In The Loop）
  ↓
Step 6: dev             — 任务拆分 + 逐 Phase 开发
  ↓
完成
```

### 关键特征

- **带参数 = 新建**：直接创建新 feature 并从头跑全流程
- **不带参数 = 续接**：自动识别已有 feature 并从断点继续
- **交互式步骤**（Step 2/3/5）需要用户参与
- **自动化步骤**（Step 1/4/6）通过子任务执行

---

## 运行环境适配

本 action 可在两种环境下运行，调用子 action 的方式不同：

| 环境 | 交互式步骤（Human In The Loop） | 自动化步骤 |
|------|-------------------------------|-----------|
| **Claude Code** | 主 Agent 读取并执行对应 action 文件 | 使用 Agent 工具派发 subagent 执行 |
| **OpenClaw** | 主 Agent 直接执行对应 action | 使用 `sessions_spawn` 派发子 Agent |

**判断当前环境**：检查是否有 `sessions_spawn` 工具可用。有则为 OpenClaw 环境，无则为 Claude Code 环境。

---

## 执行流程

> **`<skill_dir>` 约定**：以下路径中 `<skill_dir>` 指代本 action 所属的 speckit skill 目录的绝对路径。

### Step 0：FEATURE_NAME 解析 + 环境准备

#### 0a. FEATURE_NAME 解析

读取并执行 `<skill_dir>/prompts/resolve_feature_name.md`，传入以下参数：
- `ACTION_TYPE = "run"`
- `ARG = {用户传入的参数}`

执行完成后获得 `FEATURE_NAME`、`SPECS_DIR`、`IS_NEW_FEATURE`、`PRIMARY_REPO`。

> 注：resolve_feature_name 内部已经调用了 resolve_workspace.md，自动完成了多仓库扫描 / 主仓库选择 / 分支切换。

#### 0b. 推导变量

使用 `PRIMARY_REPO` 作为工作目录，而非 `$(pwd)`：

```text
CWD = {PRIMARY_REPO}
CONSTITUTION_DOC = {CWD}/constitution/constitution.md
```

#### 0c. 校验 CONSTITUTION_DOC

检查 `CONSTITUTION_DOC` 文件是否存在：
- 不存在 → 提示用户："`CONSTITUTION_DOC` 文件不存在：{路径}，请在项目根目录下准备 constitution/constitution.md 文件"

#### 0d. 断点续传检测

检测 `docs/xdev/speckit/{FEATURE_NAME}/` 下已有产物，判断从哪一步开始：

| 检查条件 | 从哪一步开始 |
|----------|-------------|
| `IS_NEW_FEATURE = true` | Step 1: specify |
| `feature.md` 不存在 | Step 1: specify（异常状态，重新执行） |
| `spec.md` 不存在 | Step 1: specify |
| `spec.md` 存在但用户要求重审 | Step 2: review-spec |
| `tech-guidance.md` 不存在 | Step 3: tech-guidance |
| `plan.md` 不存在 | Step 4: tech-design |
| `plan.md` 存在但用户要求重审 | Step 5: review-tech-design |
| `subtasks_*.md` 不存在 | Step 6: dev |
| `subtasks_*.md` 部分有"✅ 已完成" | Step 6: dev（自动续接） |

#### 0e. TODO 列表

使用 `TodoWrite` 工具创建 TODO 列表，覆盖全流程各步骤。执行过程中实时更新状态（pending → in_progress → completed）。续接执行时，将已完成的步骤直接标记为 completed。

---

### Step 1：specify — 生成功能规格说明书（自动化）

读取并执行 `<skill_dir>/actions/specify.md`，传入参数：
- 新建场景：FEATURE_NAME 已由 Step 0 确定，feature.md 已就绪
- 续接场景：传入 `{FEATURE_NAME}`

**等待完成**，确认产出文件存在：
- `{SPECS_DIR}/spec.md`

---

### Step 2：review-spec — 交互式审阅规格（交互式）

读取并执行 `<skill_dir>/actions/review-spec.md`，传入参数 `{FEATURE_NAME}`。

这是一个交互式步骤，用户会与 Agent 对话修订 `spec.md`。等待用户确认审阅完毕后，继续下一步。

**产出**：修订后的 `{SPECS_DIR}/spec.md`

---

### Step 3：tech-guidance — 交互式收集技术指导（交互式）

读取并执行 `<skill_dir>/actions/tech-guidance.md`，传入参数 `{FEATURE_NAME}`。

这是一个交互式步骤，用户提供技术选型、约束条件等指导信息。如果用户没有额外输入，可直接跳过。

**产出**：`{SPECS_DIR}/tech-guidance.md`

---

### Step 4：tech-design — 生成技术方案设计（自动化）

读取并执行 `<skill_dir>/actions/tech-design.md`，传入参数 `{FEATURE_NAME}`。

tech-design action 内部会自动编排：并发组 A → mining-diff → 串行 plan。

**等待完成**，确认产出文件存在：
- `{SPECS_DIR}/data-model.md`
- `{SPECS_DIR}/contracts.md`
- `{SPECS_DIR}/configuration.md`
- `{SPECS_DIR}/integration.md`
- `{SPECS_DIR}/plan.md`

---

### Step 5：review-tech-design — 交互式审阅技术方案（交互式）

读取并执行 `<skill_dir>/actions/review-tech-design.md`，传入参数 `{FEATURE_NAME}`。

这是一个交互式步骤，用户审阅 5 个技术方案文件。等待用户确认审阅完毕后，继续下一步。

**产出**：修订后的 5 个技术方案文件

---

### Step 6：dev — 任务拆分 + 逐 Phase 开发（自动化）

读取并执行 `<skill_dir>/actions/dev.md`，传入参数 `{FEATURE_NAME}`。

dev action 内部会：
1. 执行任务拆分（task-split）→ 生成 `subtasks_*.md`
2. 逐 Phase 串行开发（phase-dev）→ 每个 Phase 完成后自动 commit + push
3. 遇到阻塞问题时，自动向用户转问并收集回答后续接开发

**等待完成**。

---

### Step 7：报告结果

输出全流程执行摘要：

```
SDD 全流程完成

代码仓库：当前工作目录
规格文档：docs/xdev/speckit/{FEATURE_NAME}/

产出文件：
  - feature.md         — 原始需求描述留底
  - spec.md            — 功能规格说明书
  - tech-guidance.md   — 技术指导文档
  - data-model.md      — 数据模型设计
  - contracts.md       — API 接口契约
  - configuration.md   — 配置设计
  - integration.md     — 系统集成设计
  - plan.md            — 技术实现方案
  - subtasks_*.md      — 任务拆分文档
  - 代码变更           — 已提交到远端分支
```

---

## 产出物总览

| 产出文件 | 产出步骤 | 说明 |
|---------|---------|------|
| `feature.md` | Step 0/1 | 原始需求描述留底 |
| `prd.md` | Step 1 | 完整需求文档（用户输入 + 引用文档展开内容） |
| `constitution/constitution.md` | 用户提前准备（项目根目录） | 代码库规约文档 |
| `spec.md` | Step 1 → Step 2 | 功能规格说明书（经审阅修订） |
| `tech-guidance.md` | Step 3 | 技术指导文档 |
| `analyze.md` | Step 4 | 代码库现状分析 |
| `research.md` | Step 4 | 技术方案调研 |
| `mining-raw.md` | Step 4 | 隐性需求挖掘（原始版） |
| `mining.md` | Step 4 | 隐性需求挖掘（差异化版） |
| `data-model.md` | Step 4 | 数据模型设计 |
| `contracts.md` | Step 4 | API 接口契约 |
| `configuration.md` | Step 4 | 配置设计 |
| `integration.md` | Step 4 | 系统集成设计 |
| `plan.md` | Step 4 → Step 5 | 技术实现方案（经审阅修订） |
| `subtasks_*.md` | Step 6 | 各 Phase 任务拆分文档 |
| 代码变更 | Step 6 | 每个 Phase 自动 commit + push |
