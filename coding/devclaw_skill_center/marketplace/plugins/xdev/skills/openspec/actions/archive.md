# OpenSpec Archive

把已完成的 change 归档到 `docs/xdev/openspec/changes/archive/YYYY-MM-DD-<name>/`,并在归档前(可选地)把 delta specs 合并回主 specs。

## 参数

- `ARG`(可选):change-name。不提供则从对话上下文推断 / 列出选择。

---

## 执行流程

> **`<skill_dir>` 约定**:以下路径中 `<skill_dir>` 指代本 action 所属的 openspec skill 目录的绝对路径。

### Step 1:解析工作区

⚠️ **MUST 强制流程**：本步骤必须读取并按 `<skill_dir>/prompts/resolve_workspace.md` 的指令**完整执行**(包括其中的 HARD-GATE 约束)。**严禁** inline 跳过任何步骤、严禁自行决定主仓库、严禁用 inline `git rev-parse` 假装走完流程。即使 IS_NEW_FEATURE=false 跳过 list 和工作分支创建，**步骤 1 / 1.5 / 4 仍然必须严格执行**。

读取并执行 `<skill_dir>/prompts/resolve_workspace.md`,传入:

```text
SKILL_KIND     = "openspec"
SPECS_DIR_NAME = "docs/xdev/openspec"
FEATURE_NAME   = "openspec-archive"
IS_NEW_FEATURE = false
```

获得 `PRIMARY_REPO`。

### Step 2:选择 change

#### 2a. 有参数

`CHANGE_NAME = ARG`,校验 `{PRIMARY_REPO}/docs/xdev/openspec/changes/{CHANGE_NAME}/` 存在。

#### 2b. 无参数

1. 从对话上下文推断
2. 推断不到 → 列出活跃 changes:`ls -d "{PRIMARY_REPO}/docs/xdev/openspec/changes/"*/ 2>/dev/null | xargs -n1 basename | grep -v '^archive$'`
3. AskUserQuestion 让用户选择
4. **不要自动选**,必须由用户明确选定

### Step 3:检查 artifact 完成状态

读取并执行 `<skill_dir>/prompts/parse_status_json.md`。

如果 `IS_ALL_DONE = false`(有 artifact 未完成):

用 **AskUserQuestion 工具** 反问:

> "Change `{CHANGE_NAME}` 有未完成的 artifact:{列出未完成的}。是否仍然归档?"
> [1] 仍然归档(带 warning)
> [2] 取消,先补齐 artifact

### Step 4:检查 task 完成状态

```bash
TASKS_FILE="{PRIMARY_REPO}/docs/xdev/openspec/changes/{CHANGE_NAME}/tasks.md"
if [ -f "$TASKS_FILE" ]; then
  remaining=$(grep -c '^- \[ \]' "$TASKS_FILE")
  complete=$(grep -c '^- \[x\]' "$TASKS_FILE")
fi
```

如果有未完成 task(`remaining > 0`),同 Step 3 反问用户。

如果 tasks.md 不存在,跳过本步。

### Step 5:评估 delta spec sync

```bash
ls -d "{PRIMARY_REPO}/docs/xdev/openspec/changes/{CHANGE_NAME}/specs/" 2>/dev/null
```

#### 5a. 不存在 delta specs

跳到 Step 6。

#### 5b. 存在 delta specs

汇总 delta 操作(读 `{CHANGE_DIR}/specs/<cap>/spec.md`,统计 ADDED/MODIFIED/REMOVED/RENAMED 数量)。

用 **AskUserQuestion 工具**:

> "Change `{CHANGE_NAME}` 包含 {N} 个 delta spec({Total} 个 delta 操作),归档前是否要 sync 到主 specs?"
> [1] 现在 sync(推荐)
> [2] 仅归档不 sync

#### 选 [1]:现在 sync

读取并执行 `<skill_dir>/prompts/delta_merge.md`,传入:

```text
CHANGE_NAME  = {CHANGE_NAME}
PRIMARY_REPO = {PRIMARY_REPO}
```

`delta_merge.md` 会按 ACID 风格把 4 类 delta 操作(ADDED / MODIFIED / REMOVED / RENAMED)合并到主 specs。完成后回到 Step 6。

如果 delta_merge 报错(同名冲突 / MODIFIED 找不到 / REMOVED 缺 Reason 等),**不要 mv 归档目录**,把错误直接报告给用户。

#### 选 [2]:仅归档不 sync

跳到 Step 6。delta specs 会随归档目录一起被 mv,但不合并到主 specs。

### Step 6:执行归档(目录搬移)

计算归档目标:

```text
DATE         = $(date +%Y-%m-%d)
ARCHIVE_DIR  = {PRIMARY_REPO}/docs/xdev/openspec/changes/archive/{DATE}-{CHANGE_NAME}
SOURCE_DIR   = {PRIMARY_REPO}/docs/xdev/openspec/changes/{CHANGE_NAME}
```

检查归档目标是否已存在:

```bash
ls -d "{ARCHIVE_DIR}/" 2>/dev/null
```

存在 → 报错并提示用户:"归档目录 `archive/{DATE}-{CHANGE_NAME}/` 已存在。请先重命名旧归档或换个日期。"

执行归档:

```bash
mkdir -p "$(dirname "{ARCHIVE_DIR}")"
mv "{SOURCE_DIR}" "{ARCHIVE_DIR}"
```

### Step 7:输出归档摘要

```
✅ OpenSpec Change 已归档

Change:       {CHANGE_NAME}
归档位置:     {ARCHIVE_DIR}
Delta sync:   {已合并 / 无 delta / 跳过}

警告(如有):
  - {未完成 artifact 数量}
  - {未完成 task 数量}
```

---

## Guardrails

- 不要自动选择 change,必须由用户明确选定
- mv 之前先 verify 归档目标不存在,避免覆盖
- 警告不阻塞归档,只是告知
- 归档后保留 `.openspec.yaml`(它会随目录一起被 mv)
- 如果用户在 Step 5 选了 sync,sync 完成后才能 mv;sync 失败则不 mv
- **delta merge 是 ACID 风格**:全部成功或全部失败,不留半成品。任何 cap 报错 → 立即停止,不 mv 归档目录
- delta merge 前先 `git status` 确认主 specs 没有未提交改动;有改动 → 提示用户先 stash 或 commit
- 处理 H4 scenarios(`####`)时严格保持 4 个 hashtags,**不能误改成 3 个**(由 prompts/delta_merge.md 内部保证,但 archive 层级也提醒一遍)
