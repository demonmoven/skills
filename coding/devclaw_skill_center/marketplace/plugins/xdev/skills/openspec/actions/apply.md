# OpenSpec Apply

按 change 内的 `tasks.md` 实施代码。每完成一个 task 立即把 `- [ ]` 改成 `- [x]`,遇到阻塞或歧义则暂停反问用户。

## 参数

- `ARG`(可选):change-name。不提供则从对话上下文推断 / 自动选择。

---

## 执行流程

> **`<skill_dir>` 约定**:以下路径中 `<skill_dir>` 指代本 action 所属的 openspec skill 目录的绝对路径。

### Step 1:解析工作区

⚠️ **MUST 强制流程**：本步骤必须读取并按 `<skill_dir>/prompts/resolve_workspace.md` 的指令**完整执行**(包括其中的 HARD-GATE 约束)。**严禁** inline 跳过任何步骤、严禁自行决定主仓库、严禁用 inline `git rev-parse` 假装走完流程。即使 IS_NEW_FEATURE=false 跳过 list 和工作分支创建，**步骤 1 / 1.5 / 4 仍然必须严格执行**。

读取并执行 `<skill_dir>/prompts/resolve_workspace.md`,传入:

```text
SKILL_KIND     = "openspec"
SPECS_DIR_NAME = "docs/xdev/openspec"
FEATURE_NAME   = "openspec-apply"
IS_NEW_FEATURE = false
```

获得 `PRIMARY_REPO`。

### Step 2:选择 change

#### 2a. 有参数

`CHANGE_NAME = ARG`,校验 `{PRIMARY_REPO}/docs/xdev/openspec/changes/{CHANGE_NAME}/` 存在,否则报错。

#### 2b. 无参数

1. 从对话上下文推断(如果用户刚执行过 `/openspec propose <name>`,可推断出 name)
2. 推断不到 → 列出活跃 changes:`ls -d "{PRIMARY_REPO}/docs/xdev/openspec/changes/"*/ 2>/dev/null | xargs -n1 basename`
3. 如果只有 1 个活跃 change → 自动选中
4. 多个 → AskUserQuestion 让用户选择
5. 一个都没有 → 报错:"`{PRIMARY_REPO}/docs/xdev/openspec/changes/` 下没有活跃 change。请先运行 `/openspec propose <需求描述>`"

确认后输出:"Using change: `{CHANGE_NAME}`(如需切换:`/openspec apply <other-name>`)"

### Step 3:拉取状态

读取并执行 `<skill_dir>/prompts/parse_status_json.md`,传入:

```text
PRIMARY_REPO = {PRIMARY_REPO}
CHANGE_NAME  = {CHANGE_NAME}
```

获得 `IS_APPLY_READY` / `IS_ALL_DONE`。

- `IS_APPLY_READY = false` → 阻塞,提示用户先运行 `/openspec propose <change-name>` 补齐 planning artifacts(propose 的 L3 检测会跳过已存在的、生成缺失的),并终止
- `IS_ALL_DONE = true` 且 tasks 全部 `[x]` → 提示 "全部任务已完成,是否归档?" → 选 yes 则跳到 archive,no 则终止

### Step 4:加载 apply 上下文

apply 阶段的"指令"嵌入在本文件,不依赖外部 CLI。

**Apply Instruction**(来自 spec-driven schema):

> Read context files, work through pending tasks, mark complete as you go. Pause if you hit blockers or need clarification.

**contextFiles**(必读,按顺序):

1. `{PRIMARY_REPO}/docs/xdev/openspec/changes/{CHANGE_NAME}/proposal.md`
2. `{PRIMARY_REPO}/docs/xdev/openspec/changes/{CHANGE_NAME}/specs/**/*.md`(所有 spec 文件)
3. `{PRIMARY_REPO}/docs/xdev/openspec/changes/{CHANGE_NAME}/design.md`(如有)
4. `{PRIMARY_REPO}/docs/xdev/openspec/changes/{CHANGE_NAME}/tasks.md`

**项目级 agentic context**(按需 Read,与 propose Step 5 一致):

- `{PRIMARY_REPO}/README.md` / `AGENTS.md` / `CLAUDE.md` / `package.json` 等仓库根的通用项目文档
- 不再读 `docs/xdev/openspec/config.yaml`(已废弃)

### Step 5:扫描 tasks.md 计算进度

```bash
# 统计 - [ ] 与 - [x] 数量
grep -c '^- \[ \]' "{PRIMARY_REPO}/docs/xdev/openspec/changes/{CHANGE_NAME}/tasks.md"  # remaining
grep -c '^- \[x\]' "{PRIMARY_REPO}/docs/xdev/openspec/changes/{CHANGE_NAME}/tasks.md"  # complete
```

派生:

```text
total     = remaining + complete
progress  = "complete/total"
state     = "all_done" (remaining = 0) / "ready" (remaining > 0)
```

#### 5a. state = "all_done"

提示 "All tasks complete! 可以运行 `/openspec archive {CHANGE_NAME}` 归档",终止。

#### 5b. state = "ready"

继续 Step 6。

### Step 6:实施 tasks 循环

使用 **TodoWrite 工具** 跟踪进度,把 tasks.md 中所有 `- [ ]` 项目作为 TODO 列表。

对每一个 `- [ ]` 状态的 task:

1. 输出 "Working on task X/Y: <description>"
2. **实施代码改动**:按 task 描述、参考 design.md、遵守 specs/
3. 改动尽量小且聚焦,**不引入计划外的重构 / 顺手清理**
4. 把 tasks.md 中对应行的 `- [ ]` 改为 `- [x]`(用 Edit 工具)
5. 如果遇到以下情况,**暂停**:
   - Task 描述模糊 → 反问用户
   - 实施过程发现 design.md 有缺陷 → 暂停,建议用户更新 design.md 后再继续
   - 报错或阻塞 → 输出错误信息,等用户指示
   - User interrupted

### Step 7:输出实施摘要

#### 7a. 全部 task 完成

```
✅ Apply 完成

Change: {CHANGE_NAME}
Schema: spec-driven
Progress: {N}/{N} tasks complete

本次完成的 tasks:
  - [x] task 1
  - [x] task 2
  ...

下一步:
  /openspec archive {CHANGE_NAME}   归档(自动 sync delta specs)

如果想对照 spec 检查代码一致性,直接对话里说"帮我对照 spec 检查代码"即可。
```

#### 7b. 中途暂停

```
⏸ Apply 暂停

Change: {CHANGE_NAME}
Progress: {M}/{N} tasks complete

暂停原因:{原因}

待用户确认/修复后,重新运行 /openspec apply {CHANGE_NAME} 继续。
```

---

## Guardrails

- 实施时不要做计划外的"顺手清理"或"顺便重构"
- 每完成一个 task 立即更新 tasks.md(原子性,不要攒批)
- contextFiles 必须读完再开始改代码
- 遇到 design 缺陷优先暂停而非"凭判断硬上"
- 续接场景:重新进入时先扫描 tasks.md,从第一个 `- [ ]` 开始
