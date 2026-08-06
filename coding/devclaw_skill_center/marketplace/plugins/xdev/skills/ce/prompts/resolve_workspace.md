# 工作区解析指令（resolve_workspace）

> 通用的「git 仓库扫描 + 起始基准分支确认 + 主仓库选择 + 工作分支创建」逻辑，被 speckit / exec-plan / openspec / gstack 四个 skill 各自的 prompts/ 目录下分别保存一份**物理副本**。
>
> 物理副本之间应当保持同步，任何修改请同步其他 skill 内的同名文件。

---

<HARD-GATE>

## ⚠️ 强制流程约束（HARD-GATE）

**这是 `${SKILL_KIND}` 的工作区解析强制流程。调用方（agent）必须严格按本文件执行，不得 inline 优化、跳过或自行决策任何步骤。**

### Iron Law（铁律，不可违反）

1. **必须真正调用 `AskUserQuestion` 工具**弹出步骤 3 的 list（即使只有 1 个仓库），不得自行假设用户答案
2. **必须真正执行**步骤 1.5 的脏检查命令（含 pathspec 排除），不得跳过
3. **必须真正执行**步骤 5 的 `git checkout -b` 和 `git push -u origin`，不得遗漏 push
4. **必须真正调用 `AskUserQuestion`** 完成步骤 4 的多仓库主仓库选择，不得自行决定

### 禁止行为（任何一项发生即视为流程失败）

- ❌ 跳过步骤 3 的 list 直接进步骤 4
- ❌ 用 inline 的 `git rev-parse` + `git branch --show-current` 假装走完了流程
- ❌ 自己决定主仓库而不弹 `AskUserQuestion`
- ❌ 跳过 `push -u origin`
- ❌ 把"resolve_workspace 内部行为"当成可选的注释，自己改写流程

### Red Flags（出现以下念头时立即 STOP，回到对应步骤重做）

| 念头 | 反驳 |
|------|------|
| "只有 1 个仓库，list 没意义，可以省略" | ❌ 必须弹。即使 1 个仓库也要弹，这是用户体验的一致性 |
| "当前分支看起来就是正确的，省略确认" | ❌ 必须弹。"看起来正确"是 agent 的主观判断，不是用户的明确确认 |
| "用户没明确要求 list" | ❌ 必须弹。用户没要求是因为流程已经规定了，不是因为不需要 |
| "我已经在前面用 git rev-parse 看过了，可以直接进下一步" | ❌ 看过 ≠ 让用户确认 |
| "为了快速完成任务，可以省略一些步骤" | ❌ "快速"不是绕过流程的理由，断裂的用户体验代价更高 |
| "用户应该能理解我的判断，不需要再问" | ❌ 不需要"理解"，需要"确认"。这两件事不一样 |

### 例外

**没有例外**。`IS_NEW_FEATURE = false` 的场景（如 openspec）只是会跳过步骤 3 的 list 和步骤 5 的工作分支创建，但**步骤 1 / 1.5 / 4 / 6 仍然必须严格执行**。

</HARD-GATE>

---

## 输入参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `SKILL_KIND` | 是 | `"speckit"` / `"exec-plan"` / `"openspec"` / `"gstack"`，仅用于交互文案区分 |
| `SPECS_DIR_NAME` | 是 | 主仓库内放置规格的目录名。speckit = `"speckit"`，exec-plan = `"exec-plan"`，openspec = `"openspec"`，gstack = `"gstack"` |
| `FEATURE_NAME` | 是 | 调用方已经决定的本次需求标识（kebab-case）。新建工作分支时统一以此为分支名。 |
| `IS_NEW_FEATURE` | 是 | 布尔值。`true` = 本次是新建 feature 流程（**走 list 让用户单次确认所有仓库的起始基准分支 + execute 阶段创建工作分支 FEATURE_NAME 并 push**）；`false` = 续接已有 feature 流程或不参与分支管理（**跳过 list、跳过工作分支创建**，仅做仓库扫描和主仓库选择） |

---

## 输出参数

| 参数 | 类型 | 说明 |
|------|------|------|
| `WORKSPACE_REPOS` | List[Repo] | 用户确认过的所有参与仓库（每项含 `repo_path` / `base_branch` / `is_new_branch`） |
| `PRIMARY_REPO` | string | 主仓库的绝对路径，规格目录将创建在该仓库根目录下 |
| `PRIMARY_BRANCH` | string | `IS_NEW_FEATURE=true` 时为 FEATURE_NAME；`IS_NEW_FEATURE=false` 时为主仓库的 base_branch |

---

## 设计原则

### 单仓与多仓走完全相同的交互流程

唯一的差别是「待处理仓库列表」的来源不同：

- 当前目录本身是 git 仓库 → repos 列表 = `[当前 git 仓库]`
- 当前目录不是 git 仓库 → repos 列表 = 扫描一级子目录得到的所有 git 仓库

得到 repos 列表后，无论 1 个还是 N 个，都进入**统一的单次确认流程**。

### plan / execute 两阶段模型

整个 resolve_workspace 严格分为两个阶段：

- **plan 阶段**（步骤 1 ~ 4）：仅询问与决策，**不动 git**（除了只读的 `git status` / `git rev-parse` / `git branch --show-current`）
- **execute 阶段**（步骤 5）：根据 plan 阶段产生的 selections 一次性执行所有写操作（`git checkout` / `git push`）

### 用户对起始基准分支的修改：交给用户在流程外手动处理

不在 list 内部支持用户修改某个仓库的起始基准分支。原因：修改基准这一动作本身就是 `git checkout`，而 `git checkout` 必须基于已有分支 —— 这就形成了「为了切到正确的基准而需要先有一个基准」的递归矛盾。

因此 list 设计成**单次确认**：
- 用户只能选「✅ 全部正确，继续」或「❌ 有错误，终止流程」
- 选终止后，由用户在流程外手动 `git checkout` 到正确的起始基准分支，再重新运行 skill

---

## 总体决策树

```
[plan 阶段：仅询问与决策，不动 git]

步骤 1：判定 repos 列表来源
        │
   ┌────┴────┐
   当前是      当前不是
   git 仓库    git 仓库
        │           │
        ▼           ▼
   repos = [当前]  扫描一级子目录
        │           │
        └─────┬─────┘
              ▼
      步骤 1.5：脏检查（pathspec 排除 .claude/.trae/.coco/.codex/）
              │
              ▼
      步骤 2：初始化 selections（仅设置 base_branch = 当前分支，不动 git）
              │
              ▼
      步骤 3：list 单次确认（仅 IS_NEW_FEATURE=true）
              │
              ▼
      步骤 4：主仓库选择（仅多仓时交互，单仓自动选定）

[execute 阶段：根据 selections 一次性执行 git]

      步骤 5：创建工作分支 FEATURE_NAME 并 push（仅 IS_NEW_FEATURE=true）
              │
              ▼
      步骤 6：输出 WORKSPACE_REPOS / PRIMARY_REPO / PRIMARY_BRANCH
```

---

## 步骤 1：判定 repos 列表来源

```bash
git rev-parse --is-inside-work-tree 2>/dev/null
```

### Case 1：当前目录本身是 git 仓库

```bash
PRIMARY_GIT_ROOT=$(git rev-parse --show-toplevel)
CURRENT_BRANCH=$(git branch --show-current)
```

构造 repos 列表（只有 1 个元素）：

```text
repos = [
  {path: PRIMARY_GIT_ROOT, current_branch: CURRENT_BRANCH}
]
```

继续步骤 1.5。

### Case 2：当前目录不是 git 仓库 — 扫描一级子目录

```bash
CWD=$(pwd)
repos=()
for dir in "$CWD"/*/; do
  [ -d "$dir" ] || continue
  if git -C "$dir" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    repo_path=$(git -C "$dir" rev-parse --show-toplevel)
    current_branch=$(git -C "$dir" branch --show-current)
    repos+=("${repo_path}|${current_branch}")
  fi
done
```

#### Case 2 异常：列表为空

如果扫描结果为空，**报错并终止**：

> 当前目录及其一级子目录都不是 git 仓库。
> 请 `cd` 到一个 git 仓库（或者一个含多个 git 子仓库的工作区目录）后再运行 `/{SKILL_KIND} ...`。

---

## 步骤 1.5：脏检查（所有 repos 一次性，排除工具/IDE 目录）

对 repos 中**每个**仓库执行：

```bash
git -C "${repo.path}" status --porcelain -- \
  ':!.claude/' \
  ':!.trae/' \
  ':!.coco/' \
  ':!.codex/'
```

`--` 之后的 `:!<path>` 是 git 的 magic pathspec exclude 短形式，把 4 个 agent / IDE 工具目录排除在脏检查之外。这些目录通常由 xdev plugin / agent 管理，不应该影响业务脏检查。

收集所有「输出非空」的脏仓库列表 `dirty_repos`。

- 如果 `dirty_repos` **非空** → **报错并终止**：

  ```text
  以下仓库有未提交改动（已排除 .claude/.trae/.coco/.codex/），请先 commit / stash / discard 后重新运行 /{SKILL_KIND}：
    - /path/to/dirty_repo_1
    - /path/to/dirty_repo_2
  ```

- 如果 `dirty_repos` **为空** → 继续步骤 2。

> **设计意图**：脏检查放在 list 之前一次性兜底，因为后续 execute 阶段会做 `git checkout`，要求工作区干净。提前一次性验证。

---

## 步骤 2：初始化 selections（不动 git）

为每个 repo 初始化一份「本次基准记录」状态：

```text
selections = {
  "/path/to/repo_1": {
    base_branch: "<repo_1.current_branch>"
  },
  "/path/to/repo_2": {
    base_branch: "<repo_2.current_branch>"
  },
  ...
}
```

> **本步骤完全不动 git**。selections 只是把当前各 repo 的"当前分支"记录下来，作为后续步骤 3 list 展示和步骤 6 输出的源数据。

---

## 步骤 3：list 单次确认（仅 IS_NEW_FEATURE=true）

### 3.0 IS_NEW_FEATURE=false 时整步跳过

当 `IS_NEW_FEATURE = false` 时，**直接跳过本步骤**，进入步骤 4。原因：续接场景或不参与分支管理的 skill（如 openspec）不需要确认基准分支，selections 维持步骤 2 初始化的当前分支。

### 3.1 弹出 AskUserQuestion

**核心约束**：本步骤必须使用 AskUserQuestion 工具。**无论 repos 列表是 1 个仓库还是 N 个仓库，都要弹出列表让用户单次确认。**

**仓库列表写在 question 文本里**（不是写在 options 里），这样用户能一次看到所有仓库的当前分支。**options 只有 2 个固定选项**。

**question 文本**示例：

```text
以下是当前工作区中的 git 仓库及其当前分支（将作为本次需求的起始基准分支）：

  1. devclaw_skills_center        当前分支（起始基准分支）：feat/xxx
  2. another-repo                 当前分支（起始基准分支）：main

请确认所有仓库的起始基准分支是否正确。
```

每行的格式：`{编号}. {basename(repo.path)}        当前分支（起始基准分支）：{selections[repo.path].base_branch}`

**options**（只有 2 个固定选项）：

```
[1] ✅ 全部正确，继续
[2] ❌ 有仓库起始基准分支有误，终止流程
```

### 3.2 用户选 [1]：全部正确

退出本步骤，进入步骤 4。

### 3.3 用户选 [2]：终止流程

**报错并终止**：

```text
本次流程已终止。

请按以下步骤处理后重新运行 /{SKILL_KIND}：
  1. cd 到对应仓库的根目录
  2. 执行 git checkout <正确的起始基准分支>
  3. 全部仓库都切换到正确分支后，重新运行 /{SKILL_KIND} ...
```

### 3.4 Self-Check Gate（进步骤 4 之前必须自检）

在进入步骤 4 之前，agent **必须** verify 以下事实，**任何一项 false 都要回到步骤 3.1 重做**：

- [ ] 我**真的**调用了 `AskUserQuestion` 工具（不是 inline 假设用户会同意）
- [ ] `AskUserQuestion` 的 question 文本里**真的**列出了所有仓库 + 当前分支
- [ ] options **真的**只有 2 个：「✅ 全部正确，继续」和「❌ 终止流程」
- [ ] 用户**真的**回复了选项（不是我自己脑补的）

**通过自检后才能进步骤 4。**

---

## 步骤 4：主仓库选择

如果 selections 中只有 1 个仓库，直接 `PRIMARY_REPO = 该仓库`，跳到步骤 5。

否则使用 **AskUserQuestion 工具**：

> "请选择主仓库。`{SPECS_DIR_NAME}/` 目录将创建在主仓库的根目录下。"

选项：

```
[1] devclaw_skills_center   (基准分支: main)
[2] another-repo            (基准分支: develop)
[3] third-repo              (基准分支: feat/foo)
```

label 用 `basename "${repo.path}"`，description 用 `基准分支: ${selections[repo.path].base_branch}`。

用户选定后，`PRIMARY_REPO = 选中的仓库`。

> ⚠️ **多仓库场景下严禁 inline 决策**：当 repos 列表多于 1 个时，**必须**调用 `AskUserQuestion` 让用户选主仓库；**严禁**自己根据 "看起来主要的那个" / "字母顺序第一个" / "上次用户用的那个" 等理由 inline 决定。这是 HARD-GATE 第 4 条铁律。

---

## 步骤 5：创建工作分支 FEATURE_NAME（仅 IS_NEW_FEATURE=true）

当 `IS_NEW_FEATURE = false` 时，**整步跳过**，直接进步骤 6。

当 `IS_NEW_FEATURE = true` 时，对 repos 中**每个**仓库执行：

```bash
git -C "${repo.path}" checkout -b "${FEATURE_NAME}" 2>/dev/null \
  || git -C "${repo.path}" checkout "${FEATURE_NAME}"
git -C "${repo.path}" push -u origin "${FEATURE_NAME}"
```

第一行：先尝试新建工作分支，失败（已存在）则降级为切换到已有同名分支。

第二行：把工作分支 push 到远端（强制 push -u 建立 tracking）。

向用户输出一行简要日志：

> 已在仓库 `<repo.path>` 创建/切换到工作分支 `<FEATURE_NAME>` 并 push 到远端。

### 5.1 push 失败处理

如果 `git push -u origin ${FEATURE_NAME}` 失败，用 **AskUserQuestion** 反问：

> "工作分支 `${FEATURE_NAME}` 已创建，但 push 到远端失败：`<error message>`。请选择："
> [1] 继续（仅本地分支，后续手动 push）
> [2] 重试 push
> [3] 终止本次需求

根据用户选择继续或退出。

---

## 步骤 6：完成

```text
WORKSPACE_REPOS = [
  {repo_path: ..., base_branch: ..., is_new_branch: ...},
  ...
]
PRIMARY_REPO = <用户选定的主仓库（或单仓库自动选定）>

# IS_NEW_FEATURE = true:
PRIMARY_BRANCH = FEATURE_NAME

# IS_NEW_FEATURE = false:
PRIMARY_BRANCH = selections[PRIMARY_REPO].base_branch
```

其中 `is_new_branch` 字段：
- 若该 repo 在步骤 5 通过 `git checkout -b` 新建了工作分支 → `true`
- 若该 repo 已经存在同名分支被切到 → `false`
- 若 `IS_NEW_FEATURE = false`（步骤 5 整步跳过）→ `false`

输出后，**resolve_workspace 解析完成**。

---

## 边界情况清单

| 场景 | 处理 |
|------|------|
| 当前目录 + 一级子目录都不是 git 仓库 | 步骤 1 Case 2 异常：报错终止 |
| 任何 repo 工作区脏（排除 .claude/.trae/.coco/.codex/ 之外仍有改动） | 步骤 1.5 报错终止 |
| 用户在 list 上选「❌ 终止」 | 步骤 3.3 报错终止，提示手动 git checkout 后重新运行 |
| git submodule（.git 是文件而非目录） | `git rev-parse --is-inside-work-tree` 仍返回 true，视为普通仓库 |
| 仓库无任何提交（空仓库） | `git checkout -b` 仍可工作；`git push -u origin` 可能失败，按 push 失败反问处理 |
| 没有 origin remote | `git push -u origin` 失败 → 走 push 失败反问 |
| 当前已经在 FEATURE_NAME 同名分支上 | 步骤 5 `git checkout -b` 失败，降级为 `git checkout`，输出提示并继续 |
| FEATURE_NAME 含 git 不允许字符 | 调用方（resolve_feature_name）已 kebab-case 化，本步骤不再校验 |
| `IS_NEW_FEATURE = false`（如 openspec） | 步骤 3 整步跳过 + 步骤 5 整步跳过，本流程退化为「扫描 + 主仓库选择」 |

---

## 与 specify / run 流程的拼接示例

```text
# 在 resolve_feature_name 决定 FEATURE_NAME 与 IS_NEW_FEATURE 后：

读取并执行 <skill_dir>/prompts/resolve_workspace.md，传入：
  SKILL_KIND     = "speckit"
  SPECS_DIR_NAME = "speckit"
  FEATURE_NAME   = {FEATURE_NAME}
  IS_NEW_FEATURE = {IS_NEW_FEATURE}

获得：
  PRIMARY_REPO
  PRIMARY_BRANCH
  WORKSPACE_REPOS

后续：
  CWD = PRIMARY_REPO   # 取代原来的 $(pwd)
  SPECS_DIR = {PRIMARY_REPO}/docs/xdev/speckit/{FEATURE_NAME}
  mkdir -p {SPECS_DIR}
  写入 feature.md 等
```
