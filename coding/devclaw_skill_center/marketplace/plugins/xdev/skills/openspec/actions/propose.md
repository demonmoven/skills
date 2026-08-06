# OpenSpec Propose

一步生成完整的 change(proposal + specs + design + tasks),随后可直接进入 apply。

`propose` 是 openspec 工作流的**主入口**,内置 3 层自动检测 + 按需创建:

1. **L1 — 仓库骨架**:`docs/xdev/openspec/` 不存在就自动创建(吸收旧 `init`)
2. **L2 — change 目录**:`changes/<name>/` 不存在就建,存在则用户选续接/换名/删重建
3. **L3 — artifact 文件**:已存在的跳过、缺失的自动生成(自动续接,吸收旧 `continue` / `ff`)

适用于「需求清晰、想直接进入实现」的快速路径。如果需求模糊,先用 `/openspec explore`。

## 参数

- `ARG`(可选):需求描述(自由文本)或 change-name(kebab-case)。不提供则交互式输入。

---

## 执行流程

> **`<skill_dir>` 约定**:以下路径中 `<skill_dir>` 指代本 action 所属的 openspec skill 目录的绝对路径。

### Step 1:解析工作区

⚠️ **MUST 强制流程**：本步骤必须读取并按 `<skill_dir>/prompts/resolve_workspace.md` 的指令**完整执行**(包括其中的 HARD-GATE 约束)。**严禁** inline 跳过任何步骤、严禁自行决定主仓库、严禁用 inline `git rev-parse` 假装走完流程。即使 IS_NEW_FEATURE=false 跳过 list 和工作分支创建，**步骤 1 / 1.5 / 4 仍然必须严格执行**。

读取并执行 `<skill_dir>/prompts/resolve_workspace.md`,传入:

```text
SKILL_KIND     = "openspec"
SPECS_DIR_NAME = "docs/xdev/openspec"
FEATURE_NAME   = "openspec-propose"
IS_NEW_FEATURE = false
```

获得 `PRIMARY_REPO`。

### Step 2:L1 检测 — 自动确保仓库骨架

```bash
if [ ! -d "{PRIMARY_REPO}/docs/xdev/openspec/" ]; then
  mkdir -p "{PRIMARY_REPO}/docs/xdev/openspec/specs"
  mkdir -p "{PRIMARY_REPO}/docs/xdev/openspec/changes"
  echo "✨ 自动完成 openspec 首次初始化(创建 docs/xdev/openspec/{specs,changes}/)"
fi
```

不存在 → 自动 mkdir,不询问用户。
存在 → 静默跳过。

> 不再询问用户是否创建 `docs/xdev/openspec/config.yaml`。该配置已废弃,改为 Step 5 的 agentic project context。

### Step 3:L2 检测 — 解析 change-name

读取并执行 `<skill_dir>/prompts/resolve_change_name.md`,传入:

```text
ARG          = {用户传入参数}
PRIMARY_REPO = {PRIMARY_REPO}
```

获得 `CHANGE_NAME` / `IS_NEW_CHANGE` / `CHANGE_DIR`(`{PRIMARY_REPO}/docs/xdev/openspec/changes/{CHANGE_NAME}`)。

> resolve_change_name 已经实现「重名时让用户选 [1] 续接 / [2] 换名 / [3] 删重建」,这正是 L2 检测的核心交互。

### Step 4:创建 change 目录(如 IS_NEW_CHANGE)

如果 `IS_NEW_CHANGE = true`:

```bash
mkdir -p "{CHANGE_DIR}"
mkdir -p "{CHANGE_DIR}/specs"
```

写入 `{CHANGE_DIR}/.openspec.yaml`:

```yaml
schema: spec-driven
created: {YYYY-MM-DD}
```

如果 `IS_NEW_CHANGE = false`(用户选择续接已有 change),跳过本步。

### Step 5:加载 agentic project context

为了让生成的 proposal/specs/design/tasks 贴合本项目实际,先从仓库根读取通用项目背景文档(替代旧的 `config.yaml`)。

#### 5a. 检测旧 config.yaml(R3 兜底)

```bash
ls "{PRIMARY_REPO}/docs/xdev/openspec/config.yaml" 2>/dev/null
```

存在 → 输出一行 warning(继续执行,不阻塞):

```
⚠️ 检测到 docs/xdev/openspec/config.yaml — 该文件已废弃,propose 不再读取。
   请把项目级 context(技术栈/测试规范等)放到 README.md / AGENTS.md / CLAUDE.md,
   或直接在对话中告知。如不再需要可手动删除该文件。
```

#### 5b. 扫描项目根通用文档

按以下顺序尝试 Read(每个文件存在才 Read,不存在静默跳过):

1. `{PRIMARY_REPO}/README.md` — 项目定位、技术栈、运行方式
2. `{PRIMARY_REPO}/AGENTS.md` — Agent 协作约定(如有)
3. `{PRIMARY_REPO}/CLAUDE.md` 或 `{PRIMARY_REPO}/.claude/CLAUDE.md` — Claude 项目级指引
4. `{PRIMARY_REPO}/package.json` — 依赖与脚本(JS/TS 项目)
5. `{PRIMARY_REPO}/pyproject.toml` / `Cargo.toml` / `go.mod` — 其他语言生态(按需)

把所有 Read 到的内容汇总为 `PROJECT_CONTEXT`,作为 Step 6 生成 artifact 时的隐式背景。

> **不要**把 `PROJECT_CONTEXT` 复制到 artifact 文件里,只在生成 prompt 时携带。

#### 5c. 兜底:全部找不到(R4 兜底)

如果 5b 中**一个文件都没找到**,用 **AskUserQuestion 工具**反问:

> "找不到项目根的 README/AGENTS/CLAUDE.md/package.json。请用一句话告诉我项目的技术栈与测试规范(可跳过):"

收到回复 → 把回复作为 `PROJECT_CONTEXT`。
用户跳过 → `PROJECT_CONTEXT = ""`,继续。

#### 5d. 对话上下文优先

如果用户在当前对话中已经明确给出技术栈/约束/规范(例如"项目用 Vitest 单测"),**优先**使用对话中的信息,合并到 `PROJECT_CONTEXT`。

### Step 6:L3 检测 + 循环生成 artifacts 直到 apply-ready

使用 **TodoWrite 工具**创建 TODO 列表跟踪进度,初始任务列表 = `["proposal", "specs", "design", "tasks"]`(实际依赖关系由 status 扫描决定)。

维护一个 `NEW_ARTIFACTS = []` 列表,记录本次 propose 实际新生成(非跳过)的 artifact id —— 供 `actions/run.md` 在 review 阶段判定 IS_NEW_GENERATION。

#### 6a. 拉取当前状态

读取并执行 `<skill_dir>/prompts/parse_status_json.md`,传入:

```text
PRIMARY_REPO = {PRIMARY_REPO}
CHANGE_NAME  = {CHANGE_NAME}
```

获得 `READY_ARTIFACTS` / `BLOCKED_ARTIFACTS` / `DONE_ARTIFACTS` / `APPLY_REQUIRES` / `IS_APPLY_READY`。

#### 6b. 检查终止条件

- `IS_APPLY_READY = true` → 跳到 Step 7
- `READY_ARTIFACTS` 为空且 `IS_APPLY_READY = false` → 异常,报错并列出 `BLOCKED_ARTIFACTS` 与缺失的 deps

#### 6c. 取第一个 ready artifact 并生成

```text
ARTIFACT_ID = READY_ARTIFACTS[0]
```

读取并执行 `<skill_dir>/prompts/artifact_instructions.md`,传入:

```text
ARTIFACT_ID     = {ARTIFACT_ID}
CHANGE_DIR      = {CHANGE_DIR}
PRIMARY_REPO    = {PRIMARY_REPO}
PROJECT_CONTEXT = {Step 5 收集的 agentic context}
```

`artifact_instructions.md` 内含 4 个 artifact 类型的模板路径 / 输出路径 / dependencies / instruction。

**关键约束**:
- 生成 proposal 时把用户的需求描述 `ARG` 作为 Why / What Changes 的核心输入
- 生成 specs 时,先 Read `{PRIMARY_REPO}/docs/xdev/openspec/specs/` 现有的 spec 目录,识别哪些是 New Capability、哪些是 Modified Capability
- design 是 optional,如果是简单 change 可以在 proposal.md 中标注"不需要 design",此时把 design 视为 done

生成成功 → 把 `ARTIFACT_ID` 追加到 `NEW_ARTIFACTS`。

#### 6d. 更新 TodoWrite,回到 6a

把 `{ARTIFACT_ID}` 标记为 completed,回到 6a 拉取最新状态。

### Step 7:输出最终摘要

```
✅ OpenSpec change `{CHANGE_NAME}` 已就绪可进入 apply

位置:{CHANGE_DIR}

本次新生成 artifacts:
  {遍历 NEW_ARTIFACTS,逐个列出"✓ {id}"}
  {如果 NEW_ARTIFACTS 为空,输出"(无新生成,所有 artifact 都已存在)"}

完整 artifact 状态:
  ✓ proposal.md     — {简短描述}
  ✓ specs/<cap>/spec.md  — {简短描述}
  ✓ design.md       — {简短描述 / "skipped(简单 change)"}
  ✓ tasks.md        — {N} 个任务

NEW_ARTIFACTS = {NEW_ARTIFACTS}    # 供 run.md 判定 IS_NEW_GENERATION

下一步:
  /openspec apply {CHANGE_NAME}    立即开始实施
```

---

## Guardrails

- L1 检测自动 mkdir,不询问用户
- 不再读写 `docs/xdev/openspec/config.yaml`(已废弃);检测到旧文件输出 warning
- 项目根的 README / AGENTS / CLAUDE.md / package.json(任一存在即可)Read 作为隐式 PROJECT_CONTEXT
- 全部找不到 → 主动 AskUserQuestion 反问技术栈
- 如果 change 已存在且部分 artifact 已 done,仅生成缺失的(L3 自动续接)
- 不复制 `<context>` / `<rules>` / `<project_context>` 块到 artifact 文件
- 每生成一个 artifact 立刻 verify 文件存在再进入下一个
- 如果遇到关键歧义,用 AskUserQuestion 反问用户,但默认偏向"做出合理判断继续"
- 生成完后提示用户 review,等用户确认前不要自动 apply(由 `/openspec run` 或用户显式调用 apply 来触发)
- 输出必须包含 `NEW_ARTIFACTS` 列表,供 run.md 判定是否需要 HITL review 暂停
