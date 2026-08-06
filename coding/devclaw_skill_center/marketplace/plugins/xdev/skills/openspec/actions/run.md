# OpenSpec 全流程(支持断点续传)

把 OpenSpec 的 propose / apply / archive 串联成一条龙:从一句需求描述到代码归档。

`run` 有两种模式:

| 模式 | 触发 | 行为 |
|------|------|------|
| **新需求模式** | `/openspec run <需求描述>` | propose 一把梭生成 → 等 review → apply → archive |
| **断点续传模式** | `/openspec run`(不带参数) | 列出所有活跃 change → 用户选一个续接 → 自动从中断点跑到归档 |

断点续传基于:
- propose 的 L3 检测(已存在的 artifact 跳过、缺失的生成)
- apply 的 tasks.md 扫描(只跑未勾选的 task)
- archive 的存在性检查(不重复归档)

## 参数

- `ARG`(可选):需求描述(自由文本)。**带参数 = 新需求**;**不带参数 = 进入断点续传模式**。

---

## 概述

```
Step 0: 解析工作区
   ↓
Step 1: 决定新需求 vs 续接(断点续传核心)
   ↓
Step 2: propose(L1 自动 init + L2 解析/创建 change + L3 续接 artifact)
   ↓
Step 3: 等用户 review(续接且无新生成 → 自动跳过)
   ↓
Step 4: apply(扫 tasks.md 接着没勾选的跑)
   ↓
Step 5: archive(自动 sync delta specs)
   ↓
Step 6: 完成
```

---

## 执行流程

> **`<skill_dir>` 约定**:以下路径中 `<skill_dir>` 指代本 action 所属的 openspec skill 目录的绝对路径。

### Step 0:解析工作区

⚠️ **MUST 强制流程**：本步骤必须读取并按 `<skill_dir>/prompts/resolve_workspace.md` 的指令**完整执行**(包括其中的 HARD-GATE 约束)。**严禁** inline 跳过任何步骤、严禁自行决定主仓库、严禁用 inline `git rev-parse` 假装走完流程。即使 IS_NEW_FEATURE=false 跳过 list 和工作分支创建，**步骤 1 / 1.5 / 4 仍然必须严格执行**。

读取并执行 `<skill_dir>/prompts/resolve_workspace.md`,传入:

```text
SKILL_KIND     = "openspec"
SPECS_DIR_NAME = "docs/xdev/openspec"
FEATURE_NAME   = "openspec-run"
IS_NEW_FEATURE = false
```

获得 `PRIMARY_REPO`。

> 注:这里不再做"确保已 init"的检查 — propose Step 2 会自动 L1 检测并 mkdir 仓库骨架。

### Step 1:决定新需求 vs 续接(断点续传核心)

#### 1a. ARG 非空 — 新需求模式

```text
IS_RESUME    = false
PROPOSE_ARG  = ARG
```

直接进入 Step 2,把 `PROPOSE_ARG` 作为需求描述传给 propose。

#### 1b. ARG 为空 — 进入断点续传模式

扫描所有活跃 change(排除 archive 目录):

```bash
ACTIVE_CHANGES=$(ls -d "{PRIMARY_REPO}/docs/xdev/openspec/changes/"*/ 2>/dev/null \
  | xargs -n1 basename 2>/dev/null \
  | grep -v '^archive$' \
  | sort)
```

按结果数分流:

##### Case A:0 个活跃 change

仓库内没有任何活跃 change。可能是全新仓库,或者所有 change 都已归档。

用 **AskUserQuestion 工具**反问:

> "没有找到活跃 change。请输入新需求描述(我会创建一个新 change 并跑完整流程):"

收到回复 → 设置:

```text
IS_RESUME   = false
PROPOSE_ARG = {用户输入的需求描述}
```

进入 Step 2。

##### Case B:1 个活跃 change

自动选中作为续接目标:

```text
IS_RESUME   = true
CHANGE_NAME = {唯一的活跃 change 名}
PROPOSE_ARG = {CHANGE_NAME}
```

输出:`✨ 续接已有 change: {CHANGE_NAME}`,进入 Step 2。

##### Case C:多个活跃 change

用 **AskUserQuestion 工具**列出所有 change 名 + 一个额外选项「+ 新需求(输入描述)」,让用户选:

> "找到多个活跃 change,请选择要续接的:"
> [1] {change-1}
> [2] {change-2}
> ...
> [N] {change-N}
> [N+1] + 新需求(输入描述)

- 选中某个 change → `IS_RESUME = true`,`CHANGE_NAME = 选中项`,`PROPOSE_ARG = CHANGE_NAME`
- 选「+ 新需求」→ 二次 AskUserQuestion 收集需求描述 → `IS_RESUME = false`,`PROPOSE_ARG = 描述`

进入 Step 2。

### Step 2:propose

调用 `<skill_dir>/actions/propose.md`,传入 `PROPOSE_ARG`。

执行完成后从 propose 输出中获得:
- `CHANGE_NAME`:最终的 change 名
- `NEW_ARTIFACTS`:本次实际新生成的 artifact id 列表

判定:

```text
IS_NEW_GENERATION = (NEW_ARTIFACTS 非空)
```

### Step 3:等用户 review

#### 3a. IS_NEW_GENERATION = true(有新生成)

输出:

```
📋 Change `{CHANGE_NAME}` 的 planning artifacts 已生成

本次新生成:
  {遍历 NEW_ARTIFACTS}

请审阅以下文件:
  - {PRIMARY_REPO}/docs/xdev/openspec/changes/{CHANGE_NAME}/proposal.md
  - {PRIMARY_REPO}/docs/xdev/openspec/changes/{CHANGE_NAME}/specs/<cap>/spec.md
  - {PRIMARY_REPO}/docs/xdev/openspec/changes/{CHANGE_NAME}/design.md (如有)
  - {PRIMARY_REPO}/docs/xdev/openspec/changes/{CHANGE_NAME}/tasks.md

确认无误后回复 "ok" / "继续" 进入 apply 阶段。
有修改建议直接说,我会更新对应 artifact 后再次暂停。
```

**暂停等用户回复**。
- 用户回复肯定词 → 进入 Step 4
- 用户提修改建议 → 在对话中按建议更新对应文件 → 再次暂停

#### 3b. IS_NEW_GENERATION = false(全部已存在,断点续传)

说明本次 propose 没有产生新内容(所有 4 个 artifact 之前已经生成完毕),用户上次已经 review 过 → **自动跳过 review 暂停**,直接进入 Step 4。

输出一行提示:

```
✨ 续接模式:所有 planning artifacts 已就绪(无新生成),自动进入 apply。
```

### Step 4:apply

调用 `<skill_dir>/actions/apply.md`,传入 `CHANGE_NAME`。

apply 自身就支持断点(扫 `tasks.md` 中未勾选的 task 接着跑),无需 run.md 额外处理。

apply 完成后(所有 task 都 `[x]`)→ 进入 Step 5。

### Step 5:archive

调用 `<skill_dir>/actions/archive.md`,传入 `CHANGE_NAME`。

如果有 delta specs,在 archive 的 Step 5 delta sync 询问中**自动选择 sync**(因为 run 是一条龙模式,默认走完整流程)。

如果归档目标目录已存在(通常是上次 archive 失败后重跑),由 archive.md 自身报错并提示用户处理。

### Step 6:完成

```
🎉 OpenSpec 全流程完成

Change: {CHANGE_NAME}
模式:    {新需求 / 断点续传}
归档位置:{PRIMARY_REPO}/docs/xdev/openspec/changes/archive/YYYY-MM-DD-{CHANGE_NAME}/
Delta sync:{已合并 / 无 delta}
```

---

## Guardrails

- Step 3a 是硬性 HITL 暂停点,**禁止跳过**(除非 Step 3b 的 IS_NEW_GENERATION=false 触发自动跳过)
- 中途任何步骤报错都暂停反问用户,不要硬上
- run 模式下 archive 阶段默认 sync delta(覆盖 archive.md 的 AskUserQuestion 默认值)
- 断点续传模式下,Case B(单个 change)直接续接,不需要再次确认 — 但首次输出明确告知用户「正在续接 X」,给用户一次 Ctrl+C 的机会
- Case A(0 个 change)不能默默退出,必须 AskUserQuestion 引导用户给需求描述
- 如果用户在 Step 1 之后中断,下次重跑 `/openspec run` 会从 Step 1 重新开始,通过 L3 检测 / tasks.md 扫描自动定位中断点
