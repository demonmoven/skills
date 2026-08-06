# Action 目标解析指令（resolve_action_target）

> gstack 专属的"中间产物落地目标"解析逻辑，被 gstack 的所有单 action（非 run）在入参校验阶段引用。
>
> 与 `resolve_feature_name.md` 的差异：
> - `resolve_feature_name.md` 适用于 `/gstack run`，会切新分支；
> - 本文件适用于单 action（如 `/gstack qa`、`/gstack review`、`/gstack ship`），**从不切分支**，只解析"产物应该写到哪"。

---

## 输入参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `ACTION_NAME` | 是 | 当前 action 名（如 `qa`、`review`、`ship`、`office-hours` 等） |
| `ARG` | 是 | 用户传入的参数（可能为空字符串） |

---

## 输出参数

| 参数 | 说明 |
|------|------|
| `PRIMARY_REPO` | 当前 git 仓库根目录的绝对路径 |
| `PRIMARY_BRANCH` | 当前 git 分支名 |
| `FEATURE_NAME` | 当前 feature 标识。若分支名匹配 `{14位时间戳}-{slug}` 模式则用作 FEATURE_NAME；否则用裸分支名（main / develop 等）作为 FEATURE_NAME |
| `SPECS_DIR` | `{PRIMARY_REPO}/docs/xdev/gstack/{FEATURE_NAME}/` 的绝对路径，已自动 mkdir |
| `IS_FEATURE_BRANCH` | 布尔值。`true` = 当前分支由 gstack/speckit/exec-plan 创建（匹配时间戳模式）；`false` = 普通分支 |

---

## 解析流程

### 步骤 1：校验 git 仓库

```bash
git rev-parse --is-inside-work-tree 2>/dev/null
```

- 输出 `true` → 继续步骤 2
- 否则 → 报错："`/gstack {ACTION_NAME}` 必须在 git 仓库内运行。请 `cd` 到目标仓库后重试。"

### 步骤 2：获取仓库元信息

```bash
PRIMARY_REPO=$(git rev-parse --show-toplevel)
PRIMARY_BRANCH=$(git -C "$PRIMARY_REPO" branch --show-current)
```

如果 `PRIMARY_BRANCH` 为空（detached HEAD 状态），报错："当前处于 detached HEAD 状态，无法解析 FEATURE_NAME。请先 checkout 一个分支后重试。"

### 步骤 3：识别 FEATURE_NAME

检测 `PRIMARY_BRANCH` 是否匹配正则 `^[0-9]{14}-[a-z0-9-]+$`（即由 gstack run / speckit / exec-plan 创建的"时间戳-slug"分支）：

- 匹配 → `FEATURE_NAME = PRIMARY_BRANCH`，`IS_FEATURE_BRANCH = true`
- 不匹配 → `FEATURE_NAME = PRIMARY_BRANCH`（裸分支名），`IS_FEATURE_BRANCH = false`
  - 此时 SPECS_DIR 仍然写到 `{PRIMARY_REPO}/docs/xdev/gstack/{裸分支名}/`，便于"在 main 分支临时跑一次 qa"等场景的产物落地与续接

> **注**：分支名中的 `/` 字符需要替换为 `-`，避免在 `gstack/` 下产生子目录嵌套。例如 `feat/login` → `feat-login`。

### 步骤 4：创建 SPECS_DIR

```bash
SPECS_DIR="$PRIMARY_REPO/docs/xdev/gstack/$FEATURE_NAME"
mkdir -p "$SPECS_DIR"
```

如果 `gstack/` 还未在 `.gitignore` 中且本目录是首次创建，**不主动**修改 `.gitignore`（由用户自己决定是否提交 gstack 产物）。

### 步骤 5：返回值

```text
PRIMARY_REPO       = <绝对路径>
PRIMARY_BRANCH     = <当前分支名>
FEATURE_NAME       = <FEATURE_NAME>
SPECS_DIR          = <PRIMARY_REPO>/gstack/<FEATURE_NAME>
IS_FEATURE_BRANCH  = true / false
```

**解析完成**。

---

## 设计说明

### 为什么单 action 也写产物到 docs/xdev/gstack/{FEATURE_NAME}/？

用户的核心要求是"中间产物都统一维护到项目根目录的 docs/xdev/gstack/{FEATURE_NAME} 下"。这意味着：

1. **产物可追溯**：每次 review / qa / ship 的报告都有持久化文件，便于跨 session 续接
2. **路径可预测**：无论从 run 全流程进来，还是单独跑 qa，产物落在同一处
3. **多人协作友好**：把 gstack 目录加进 git，PR 评审者能直接看到 AI sprint 的所有中间产物

### 为什么不强制切 feature 分支？

1. 单 action 的典型用法是"在已有分支上跑一次 review/qa"，强制切分支会破坏工作流
2. 在 main 分支跑 single action 也能有合理产物路径（`gstack/main/qa.md`），不会失败
3. `/gstack run` 模式才会强制切分支（由 `resolve_feature_name.md` 处理）

### 与 resolve_feature_name 的关系

| 场景 | 调用 |
|------|------|
| `/gstack run "需求描述"` | `resolve_feature_name`（切新分支 + 创建 docs/xdev/gstack/{FEATURE_NAME}/） |
| `/gstack run`（无参数，续接） | `resolve_feature_name`（识别现有 feature） |
| `/gstack {single-action}` | `resolve_action_target`（不切分支，只解析路径） |

`run` action 内部串联各个 single action 时，会**预先**通过 `resolve_feature_name` 把 `PRIMARY_REPO` / `FEATURE_NAME` / `SPECS_DIR` 计算好并注入到子 action 的执行环境，子 action 检测到这些环境变量已存在后**跳过** `resolve_action_target` 的步骤 1-4，直接复用。
