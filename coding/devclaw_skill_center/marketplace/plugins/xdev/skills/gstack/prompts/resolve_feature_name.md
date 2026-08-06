# FEATURE_NAME 解析指令（gstack 副本）

> 通用的 FEATURE_NAME 解析逻辑，被 gstack 各 action 在入参校验阶段引用。
>
> 本文件是 speckit/prompts/resolve_feature_name.md 的物理副本，差异：
> - `SPECS_DIR_NAME` 默认为 `"gstack"`（speckit 用 `"specs"`）
> - `ACTION_TYPE` 取值改为 `"gstack-run"` / `"other"`（speckit 用 `"specify"` / `"run"` / `"other"`）

## 输入参数

1. `ACTION_TYPE`：当前 action 类型
   - `"gstack-run"`：支持新建 feature（带参数=新需求描述，不带参数=续接）
   - `"other"`：仅支持选择已有 feature
2. `ARG`：用户传入的参数（可能为空字符串）

## 输出

1. `FEATURE_NAME`：解析后的 feature 名称
2. `SPECS_DIR`：`{PRIMARY_REPO}/docs/xdev/gstack/{FEATURE_NAME}` 的绝对路径
3. `IS_NEW_FEATURE`：布尔值，是否为新建 feature（仅 gstack-run 模式可能为 true）
4. `PRIMARY_REPO`：主仓库的绝对路径（新建场景由 resolve_workspace 决定；续接场景为当前 git 仓库根目录）

---

## 解析流程

### 步骤 1：判断是否有参数

检查 `ARG` 是否为非空字符串。

- **有参数** → 进入步骤 2
- **无参数** → 跳到步骤 5（续接已有 feature）

---

### 步骤 2：有参数 — 区分 action 类型

#### ACTION_TYPE = "gstack-run"

参数视为**新需求描述**，进入步骤 3（无意义检测）。

#### ACTION_TYPE = "other"

参数视为 **FEATURE_NAME**，校验 `gstack/{ARG}/` 目录是否存在：
- 存在 → `FEATURE_NAME = {ARG}`，`IS_NEW_FEATURE = false`，**解析完成**
- 不存在 → 报错提示用户："`gstack/{ARG}/` 目录不存在，请确认 FEATURE_NAME 是否正确"

---

### 步骤 3：无意义检测（仅 gstack-run）

阅读用户输入 `ARG`，判断其是否能理解为一个**有意义的软件需求描述或文档链接**。

**视为无意义的情况**：
- 随机字符 / 乱码（如 `sdklfjwlef`、`aaa123bbb`）
- 完全无关的词组拼凑（如"塑料袋咖啡机蓝山咖啡"、"今天天气真好"）
- 过于笼统以至于无法提取任何需求意图（如"做个东西"、"帮我写代码"）

**视为有意义的判断（宽松）**：
- 只要能从中提取出某种软件需求意图即可
- 文档链接（URL 格式）一律视为有意义

**判断结果**：
- 无意义 → 报错反问用户："输入的内容似乎不是有意义的需求描述，请给出有意义的需求描述或文档链接"
- 有意义 → 进入步骤 4

---

### 步骤 4：新建 Feature 流程（仅 gstack-run）

#### 4a. 记录用户输入

将 `ARG` 原文记为 `{USER_INPUT}`（不做任何 URL 读取或展开，链接内容的读取由上层 action 负责）。

#### 4b. 生成 FEATURE_NAME

1. 从 `{USER_INPUT}` 中提取需求简要描述（如果输入中包含 URL 链接，则从非链接的文本部分提取；若输入仅为链接，则使用链接中的路径关键词）
2. 翻译并整理为 `{req_slug}`：
   - 最多 4 个单词
   - 全部小写
   - 使用 `-` 连接（kebab-case）
   - 仅保留适合作为目录名的字符
3. 获取当前时间戳：`{YYYYMMDDHHmmss}`
4. `FEATURE_NAME = {YYYYMMDDHHmmss}-{req_slug}`

示例：
```
USER_INPUT = "用户登录流程优化"
req_slug = "user-login-optimization"
FEATURE_NAME = 20260325143000-user-login-optimization
```

#### 4c. 解析工作区 + 创建目录 + 写 feature.md

1. **调用工作区解析逻辑**（强制流程，必须严格按 resolve_workspace.md 执行）：
   
   ⚠️ **MUST**：本步骤必须读取并按 `<skill_dir>/prompts/resolve_workspace.md` 的指令**完整执行**（包括其中的 HARD-GATE 约束），**严禁 inline 跳过、优化或自行决策任何步骤**。
   
   特别地，**严禁**以下行为：
   - ❌ 不弹 `AskUserQuestion` 而自己假定用户的确认
   - ❌ 自行决定主仓库而不让用户选择
   - ❌ 跳过 push -u origin
   - ❌ 用 inline `git rev-parse` + `git branch --show-current` 假装走完了流程
   
   读取并执行 `<skill_dir>/prompts/resolve_workspace.md`，传入：
   ```text
   SKILL_KIND     = "gstack"
   SPECS_DIR_NAME = "gstack"
   FEATURE_NAME   = {FEATURE_NAME}
   IS_NEW_FEATURE = true
   ```
   获得 `PRIMARY_REPO` / `PRIMARY_BRANCH` / `WORKSPACE_REPOS`。

   resolve_workspace 内部行为（plan/execute 两阶段模型，单仓多仓走相同的交互流程）：
   - **plan 阶段**（不动 git）：
     - 当前目录是 git 仓库 → repos 列表 = `[当前仓库]`；非 git 仓库 → 扫描一级子目录得到 repos 列表
     - 一次性脏检查所有 repos（pathspec 排除 `.claude/`、`.trae/`、`.coco/`、`.codex/`），任意脏即报错终止
     - IS_NEW_FEATURE=true 时弹出 list 让用户**单次确认**所有仓库的当前分支即"起始基准分支"
     - 用户只能选「✅ 全部正确」或「❌ 终止流程」；选终止则报错并提示用户手动 `git checkout` 后重新运行
     - 多仓库时再交互选择主仓库；单仓库时自动选定
   - **execute 阶段**（统一执行 git 写操作）：
     - 在每个 repo 上 `git checkout -b FEATURE_NAME` 并 `push -u origin FEATURE_NAME`

2. 创建 gstack 目录（在主仓库根目录下）：
   ```bash
   mkdir -p {PRIMARY_REPO}/docs/xdev/gstack/{FEATURE_NAME}
   ```
3. 将 `{USER_INPUT}`（原始参数原文）写入 `{PRIMARY_REPO}/docs/xdev/gstack/{FEATURE_NAME}/feature.md`

设置返回值：
- `SPECS_DIR = {PRIMARY_REPO}/docs/xdev/gstack/{FEATURE_NAME}` 的绝对路径
- `IS_NEW_FEATURE = true`
- `PRIMARY_REPO`（同时返回，供调用方作为 CWD 使用）

**解析完成**。

---

### 步骤 5：续接已有 Feature（无参数）

> 续接流程默认要求当前目录就是一个 git 仓库（即业务主仓库）。如果当前目录不是 git 仓库，报错提示用户 `cd` 进入对应仓库。

#### 5a. 校验当前目录是 git 仓库

```bash
git rev-parse --is-inside-work-tree 2>/dev/null
```

- 输出 `true` → 继续 5b
- 否则 → 报错："当前目录不是 git 仓库。续接已有 feature 需要 `cd` 到对应业务仓库后再运行。"

设置：

```text
PRIMARY_REPO = $(git rev-parse --show-toplevel)
```

#### 5b. 检查当前 git 分支名

```bash
git -C "$PRIMARY_REPO" branch --show-current
```

获取当前分支名 `{branch}`，检查 `{PRIMARY_REPO}/docs/xdev/gstack/{branch}/` 目录是否存在：

- 存在 → `FEATURE_NAME = {branch}`，`SPECS_DIR = {PRIMARY_REPO}/docs/xdev/gstack/{branch}`，`IS_NEW_FEATURE = false`，**解析完成**
- 不存在 → 进入 5c

#### 5c. 扫描 gstack/ 目录

```bash
ls -d "$PRIMARY_REPO"/gstack/*/ 2>/dev/null
```

- **有子目录** → 列出所有 feature 目录名，使用交互式选择让用户选择：

  ```
  检测到以下 gstack feature，请选择：

  1. 20260319183000-user-login-optimization
  2. 20260320100000-payment-refactor
  3. 20260325090000-notification-service
  {仅 gstack-run 模式展示} N. [输入新需求描述]

  请输入序号：
  ```

  - 用户选择已有 feature → `FEATURE_NAME = 选中值`，`SPECS_DIR = {PRIMARY_REPO}/docs/xdev/gstack/{FEATURE_NAME}`，`IS_NEW_FEATURE = false`，**解析完成**
  - 用户选择"输入新需求描述"（仅 gstack-run）→ 提示用户输入需求描述 → 将输入作为 `ARG` 回到步骤 3

- **无子目录**（gstack/ 为空或不存在）：
  - ACTION_TYPE = "gstack-run" → 提示用户："当前没有已创建的 gstack feature，请输入需求描述或文档链接" → 将输入作为 `ARG` 回到步骤 3
  - ACTION_TYPE = "other" → 报错："当前没有已创建的 gstack feature，请先运行 `/gstack run <需求描述>` 创建 feature"
