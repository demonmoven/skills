# gstack run — Sprint 全流程

> 灵感来自 gstack 上游各 sprint skill 的串联建议（office-hours → autoplan → review → cso → qa → ship）。
> 本 action 是 devclaw 版 gstack 的"主入口"，与 speckit/run、openspec/run 同位。
> 中间产物统一落到 `{PRIMARY_REPO}/docs/xdev/gstack/{FEATURE_NAME}/`。

## 参数

- `ARG`（可选）：需求描述文本或文档链接（新建场景），或不带参数（续接场景，自动识别已有 feature）

---

## 概述

本 action 把 gstack 的核心 sprint 角色串联成一条流水线：

```
Step 0: FEATURE_NAME 解析 + 切 feature 分支 + 创建 docs/xdev/gstack/{FEATURE_NAME}/
  ↓
Step 1: office-hours        — 产品构思（HITL）
  ↓
Step 2: autoplan            — CEO + Eng + Design 三审（HITL）
  ↓
Step 3: 暂停，等待用户进入实现阶段（HITL hard stop）
  ↓
Step 4: review              — 7 路并行 code review（自动）
  ↓
Step 5: cso                 — OWASP + STRIDE 安全审计（自动）
  ↓
Step 6: qa                  — 真实浏览器 QA（自动，可选；需 browse 二进制）
  ↓
Step 7: ship                — test + push + PR 创建（自动）
  ↓
Step 8: 输出完整 sprint 摘要 + Outcomes
```

---

## 哲学约束

执行本 action 前，读取 `<skill_dir>/resources/ethos.md` 并将其内容作为隐含约束。串联流程的关键原则：

- **Boil the Lake**：每个 step 都要充分跑完，不要为了"快"跳过 cso / qa / review
- **User Sovereignty**：HITL step（Step 1/2/3）必须等待用户确认，不可擅自推进
- **Investigation Before Fix**：Step 4-7 中遇到失败时，先读 `docs/xdev/gstack/{FEATURE_NAME}/{action}.md` 的历史输出，再决定是修复还是回到调查模式
- **Atomic Commits**：Step 4-7 自动产生的代码改动每一处独立 commit

---

## 执行流程

> **`<skill_dir>` 约定**：以下路径中 `<skill_dir>` 指代 gstack skill 目录（即 actions/run.md 所在目录的父目录）。

### Step 0：FEATURE_NAME 解析 + 环境准备

#### 0a. FEATURE_NAME 解析

⚠️ **MUST 强制流程**：本步骤必须读取并按 `<skill_dir>/prompts/resolve_feature_name.md` 的指令**完整执行**，该文件内部会调用 `prompts/resolve_workspace.md`（包含 HARD-GATE 约束）。**严禁** inline 跳过、优化或自行决策任何步骤，特别是不弹 `AskUserQuestion`、自行决定主仓库、跳过 push -u origin 等行为均视为流程失败。

读取并执行 `<skill_dir>/prompts/resolve_feature_name.md`，传入：

```text
ACTION_TYPE = "gstack-run"
ARG = {用户传入的参数}
```

执行完成后获得：
- `FEATURE_NAME`
- `SPECS_DIR`（即 `{PRIMARY_REPO}/docs/xdev/gstack/{FEATURE_NAME}`）
- `IS_NEW_FEATURE`
- `PRIMARY_REPO`

> resolve_feature_name 内部已经调用了 resolve_workspace，自动完成：仓库扫描 / 脏检查（排除 .claude/.trae/.coco/.codex/） / list 让用户单次确认所有仓库的起始基准分支（用户选确认或终止）/ 主仓库选择 / 切工作分支 `FEATURE_NAME` + push -u origin / 创建 `docs/xdev/gstack/{FEATURE_NAME}/feature.md`。

#### 0b. 推导环境变量

```text
CWD = PRIMARY_REPO
FEATURE_DOC = {SPECS_DIR}/feature.md
```

后续所有子 action 在派发时都注入这套环境变量（pipeline 模式），子 action 检测到 `SPECS_DIR` 已存在则跳过自身的 `resolve_action_target` 解析。

#### 0c. 断点续传检测

检测 `{SPECS_DIR}/` 下已有产物，判断从哪一步开始：

| 检查条件 | 从哪一步开始 |
|----------|-------------|
| `IS_NEW_FEATURE = true` | Step 1 |
| `office-hours.md` 不存在 | Step 1 |
| `office-hours.md` 存在但 `autoplan.md` 不存在 | Step 2 |
| `autoplan.md` 存在但用户尚未进入实现阶段（无 commit / 无 `.implementation-started` 标记） | Step 3 暂停 |
| 用户已完成实现且 `review.md` 不存在 | Step 4 |
| `review.md` 存在但 `cso.md` 不存在 | Step 5 |
| `cso.md` 存在且未声明 skip qa 但 `qa.md` 不存在 | Step 6 |
| 上述齐全但 `ship.md` 不存在 | Step 7 |
| 全部齐全 | Step 8 |

#### 0d. 创建 TODO 列表

使用 `TodoWrite` 工具创建 TODO 列表，覆盖全流程 8 步。执行过程中实时更新状态（pending → in_progress → completed）。续接执行时，将已完成的步骤直接标记为 completed。

---

### Step 1：office-hours — 产品构思（HITL）

读取并执行 `<skill_dir>/actions/office-hours.md`，传入：

```text
ACTION_NAME = "office-hours"
ARG = {USER_INPUT}                  # 来自 feature.md
PIPELINE_MODE = true
PRIMARY_REPO = {PRIMARY_REPO}
SPECS_DIR = {SPECS_DIR}
FEATURE_NAME = {FEATURE_NAME}
```

子 action 完成后必须确认产物 `{SPECS_DIR}/office-hours.md` 已存在，否则报错暂停。

这是 HITL 步骤，office-hours 内部会与用户对话 reframe 需求。等用户确认 design doc 后才继续。

---

### Step 2：autoplan — CEO + Eng + Design 三审（HITL）

读取并执行 `<skill_dir>/actions/autoplan.md`，传入：

```text
ACTION_NAME = "autoplan"
ARG = {FEATURE_NAME}
PIPELINE_MODE = true
PRIMARY_REPO = {PRIMARY_REPO}
SPECS_DIR = {SPECS_DIR}
FEATURE_NAME = {FEATURE_NAME}
```

autoplan 内部会顺序读取并执行：
1. `<skill_dir>/actions/plan-ceo-review.md`
2. `<skill_dir>/actions/plan-design-review.md`
3. `<skill_dir>/actions/plan-eng-review.md`

每一步都是 HITL，等待用户在每个审查 round 拍板。

子 action 完成后必须确认产物 `{SPECS_DIR}/autoplan.md`（以及可能的 `plan-ceo-review.md`、`plan-eng-review.md`、`plan-design-review.md`）已存在。

---

### Step 3：暂停，等待用户进入实现阶段

输出以下消息并**硬停止**：

```
✓ 计划阶段完成（office-hours + autoplan）

接下来请进入实现阶段：
- 你可以在当前分支 {FEATURE_NAME} 上自由编码，gstack 不会自动写代码
- 实现完成后，重新运行 /gstack run（不带参数），自动从 Step 4 开始 review/cso/qa/ship 流程

参考产物：
- {SPECS_DIR}/feature.md
- {SPECS_DIR}/office-hours.md
- {SPECS_DIR}/autoplan.md
```

**禁止在用户重新触发前继续 Step 4。** 等用户回到本流程并重新触发 `/gstack run` 时，断点续传检测会从 Step 4 开始。

> 如何标记"用户已完成实现"？检测 `{FEATURE_NAME}` 分支自 autoplan 完成时间戳之后是否有新 commit。有则视为已开始/完成实现，可进入 Step 4；没有则继续暂停。

---

### Step 4：review — 7 路并行 code review（自动化）

读取并执行 `<skill_dir>/actions/review.md`，传入：

```text
ACTION_NAME = "review"
ARG = {base_branch}                 # 自动从 git 配置读取
PIPELINE_MODE = true
PRIMARY_REPO = {PRIMARY_REPO}
SPECS_DIR = {SPECS_DIR}
FEATURE_NAME = {FEATURE_NAME}
```

review 内部会派发 7 路并行专家子代理（测试 / 可维护性 / 安全 / 性能 / 数据迁移 / API 契约 / 红队）。

完成后确认 `{SPECS_DIR}/review.md` 已存在。如果 review 发现 P0/P1 问题且自动修复不可行，**暂停反问用户**是修复还是 skip。

---

### Step 5：cso — OWASP + STRIDE 安全审计（自动化）

读取并执行 `<skill_dir>/actions/cso.md`，传入：

```text
ACTION_NAME = "cso"
ARG = {scope}                       # 默认为 diff
PIPELINE_MODE = true
PRIMARY_REPO = {PRIMARY_REPO}
SPECS_DIR = {SPECS_DIR}
FEATURE_NAME = {FEATURE_NAME}
```

完成后确认 `{SPECS_DIR}/cso.md` 已存在。如果发现 CRITICAL 安全问题，**暂停反问用户**是修复还是接受风险。

---

### Step 6：qa — 真实浏览器 QA（自动化，可选）

#### 6a. 检查 qa 是否需要执行

如果 `docs/xdev/gstack/{FEATURE_NAME}/.skip-qa` 文件存在（用户在 Step 3 或之前明确声明跳过），跳过 Step 6 进入 Step 7。

#### 6b. 检查 browse 二进制可用性

```bash
if [ -x "$HOME/.claude/skills/gstack/bin/browse" ]; then
  BROWSE_AVAILABLE=true
else
  BROWSE_AVAILABLE=false
fi
```

如果 `BROWSE_AVAILABLE=false`，使用 AskUserQuestion 反问：

> Step 6 (qa) 需要 gstack 的 browse 浏览器二进制。当前未检测到。请选择：
>
> A) 安装上游 gstack：`git clone --depth 1 https://github.com/garrytan/gstack ~/.claude/skills/gstack && cd ~/.claude/skills/gstack && bun install && bun run build && cd -`
> B) 跳过 qa，直接进入 Step 7 (ship)
> C) 中止全流程

选 A 后重新检测；选 B 创建 `.skip-qa` 文件并跳过；选 C 终止 run。

#### 6c. 执行 qa

读取并执行 `<skill_dir>/actions/qa.md`，传入：

```text
ACTION_NAME = "qa"
ARG = {url}                         # 需要用户在 Step 3 提供本地 / staging URL，或 AskUserQuestion 询问
PIPELINE_MODE = true
PRIMARY_REPO = {PRIMARY_REPO}
SPECS_DIR = {SPECS_DIR}
FEATURE_NAME = {FEATURE_NAME}
```

完成后确认 `{SPECS_DIR}/qa.md` 已存在。

---

### Step 7：ship — 发布全流程（自动化）

读取并执行 `<skill_dir>/actions/ship.md`，传入：

```text
ACTION_NAME = "ship"
ARG = ""
PIPELINE_MODE = true
PRIMARY_REPO = {PRIMARY_REPO}
SPECS_DIR = {SPECS_DIR}
FEATURE_NAME = {FEATURE_NAME}
```

ship 内部会执行：
- 同步 main / base branch
- 跑测试
- 跑 review（如果 Step 4 已经跑过则跳过）
- bump VERSION
- 更新 CHANGELOG
- commit + push
- 创建 PR

完成后确认 `{SPECS_DIR}/ship.md` 已存在，PR URL 已记录。

---

### Step 8：输出 Sprint 摘要

```
✓ gstack sprint 全流程完成

主仓库：{PRIMARY_REPO}
分支：{FEATURE_NAME}
规格目录：docs/xdev/gstack/{FEATURE_NAME}/

产出文件：
  - feature.md             — 原始需求描述
  - office-hours.md        — 产品构思设计文档
  - autoplan.md            — 三审汇总
  - plan-ceo-review.md     — CEO 审查报告
  - plan-eng-review.md     — 工程审查报告
  - plan-design-review.md  — 设计审查报告
  - review.md              — 7 路 code review 报告
  - cso.md                 — 安全审计报告（OWASP + STRIDE）
  - qa.md                  — 浏览器 QA 报告（若执行）
  - ship.md                — 发布摘要（VERSION、CHANGELOG diff、PR URL）

代码变更：
  - 已 commit + push 到 origin/{FEATURE_NAME}
  - PR：{PR_URL}
```

同时把所有 step 的 Decision Log 汇总到 `{SPECS_DIR}/run.md`，记录：
- 每个 step 的开始 / 完成时间戳（使用用户本地时区）
- 每个 step 的决策点与用户响应
- 跳过的 step 与原因（如 skip qa）
- 阻塞与恢复点

---

## 与单 action 模式的差异

| 维度 | `/gstack run` | `/gstack {single-action}` |
|------|--------------|-------------------------|
| 切分支 | ✅ 强制（resolve_feature_name 创建 `{14位时间戳}-{slug}` 分支） | ❌ 不切，使用当前分支名 |
| 创建 specs 目录 | ✅ 强制 `{PRIMARY_REPO}/docs/xdev/gstack/{FEATURE_NAME}/` | ✅ 同样，但 FEATURE_NAME 是当前分支名 |
| 多 step 串联 | ✅ 8 步流水线 | ❌ 单步执行 |
| HITL 暂停点 | ✅ Step 1/2/3/4(高危)/5(高危)/6(BROWSE) | 由各 action 自行决定 |
| 断点续传 | ✅ 8 步内任意位置 | ❌ 单步无续传概念 |

---

## 产物落地

run action 自身的产物 = `{SPECS_DIR}/run.md`，记录全流程的：
- 时间线（每个 step 的开始/结束时间戳）
- 决策日志（HITL 用户响应、自动决策理由）
- 跳过的 step 与原因
- 阻塞与恢复点
- 最终摘要

每个子 action 的产物由其自身负责写入 `{SPECS_DIR}/{action}.md`。

执行完成后输出：

```
✓ run 完成。Sprint 摘要已写入 {SPECS_DIR}/run.md
```
