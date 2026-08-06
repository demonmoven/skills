---
name: create-workspace-multiple
description: 'DevClaw 多仓工作空间初始化 - 创建多仓 worktree 工作空间、部署 Constitution、生成 target-src-path.json。当用户表达多仓初始化工作空间的意图时使用，如"多仓初始化"、"init multiple"。'
argument-hint: "[TARGET_SRC_PATH] [BASE_BRANCH]"
---

# DevClaw 工作空间初始化

当用户表达初始化工作空间的意图时触发，包括但不限于：
"初始化"、"初始化工作空间"、"init"、"devclaw init"、"开始新项目"等。

> 兼容旧口径：如果用户说的是 `costudio init`，也按本流程执行。

## 执行流程

### 1. 信息收集

依次向用户询问以下变量的值（使用 AskUserQuestion 工具逐一询问）：

1. **REQ_NAME**：本次需求名称简述
2. **BASE_BRANCH**：所有目标仓库共用的基准分支名称（如 `main`、`master`、`develop` 等）

随后进入多轮仓库收集流程。设当前轮次编号为 `{repo_index}`，初始值为 `1`。

每一轮都按以下顺序询问：

1. **目标代码仓库{repo_index}的本地路径**：仓库本地绝对路径
2. **目标代码仓库{repo_index}的描述**：该仓库在本次需求中的职责说明
3. **还有其他的吗？Y/N**

每轮收集完成后：

- 从仓库本地路径中提取目录名，记为该仓库的 `{repo_name}`。
- 校验该路径存在且是一个 git 仓库。
- 校验 `{repo_name}` 在本次多仓列表中唯一；如果重复，则要求用户重新输入，直到唯一为止。
- 将该仓库信息加入列表 `TARGET_REPOS`，每个元素包含：
  - `repo_name`
  - `source_path`
  - `desc`
  - `worktree_path`，值为该仓库对应的 worktree 目录路径

循环规则：

- 如果用户回答 `Y`，则 `{repo_index} = {repo_index} + 1`，继续下一轮。
- 如果用户回答 `N`，结束收集。
- 至少必须收集 1 个仓库。

### 2. 生成需求标识和工作空间

#### 2.1 生成需求英文标识

将 `REQ_NAME` 翻译并整理为英文极简标识，记为 `{req_slug}`，要求：

- 全部小写
- 使用下划线连接
- 仅保留适合作为目录名 / 分支名的字符

示例：

```text
REQ_NAME="用户中心登录改造"
req_slug="user_center_login_refactor"
```

#### 2.2 生成工作空间名称

工作空间名称格式：

```text
devclaw_{req_slug}_{base_branch}_{YYYYMMDDHHmmss}
```

其中时间戳为**当前时间**，精确到秒。

例如：

```text
devclaw_user_center_login_refactor_main_20260317153000
```

将此名称记为 `{workspace_name}`，它同时也是所有目标仓库共用的 worktree 分支名。

#### 2.3 在 `~/.devclaw/` 下创建工作空间目录

```bash
mkdir -p ~/.devclaw/{workspace_name}
```

将该目录的绝对路径记为 `{work_dir}`。

### 3. 约定 `target-src-path.json` 文件路径

在工作空间目录下约定多仓清单文件路径：

```text
{work_dir}/target-src-path.json
```

将该文件的绝对路径赋值给变量：

```text
TARGET_SRC_PATH={work_dir}/target-src-path.json
```

> 此时只定义文件位置，不立即写入最终内容。
>
> `target-src-path.json` 需要在所有仓库 worktree 创建成功后再生成，因为其中的 `path` 字段必须写为实际的 worktree 目录路径。

### 4. 创建多仓 worktree

对 `TARGET_REPOS` 中的每一个仓库逐一执行以下操作：

#### 4.1 获取目标仓库信息

设当前仓库信息为：

```text
repo_name={repo_name}
repo_source_path={source_path}
repo_desc={desc}
repo_worktree_path={worktree_path}
```

当前仓库的 worktree 目标目录为：

```text
{repo_worktree_path}
```

#### 4.2 确保基准分支存在

在执行后续 worktree 逻辑前，先对当前仓库做一次基准分支兜底检查。

检查顺序如下：

1. 先检查本地是否存在 `{BASE_BRANCH}`
2. 如果本地不存在，再检查远端 `origin/{BASE_BRANCH}` 是否存在
3. 如果本地和远端都不存在，则基于当前仓库的当前 `HEAD` 新建 `{BASE_BRANCH}` 并推送到远端

推荐检查 / 处理逻辑如下：

```bash
cd {repo_source_path}

if git show-ref --verify --quiet refs/heads/{BASE_BRANCH}; then
  echo "local branch exists"
else
  if git ls-remote --exit-code --heads origin {BASE_BRANCH} >/dev/null 2>&1; then
    git fetch origin {BASE_BRANCH}:{BASE_BRANCH}
  else
    CURRENT_BRANCH="$(git branch --show-current)"
    git checkout -b {BASE_BRANCH}
    git push -u origin {BASE_BRANCH}
    if [ -n "$CURRENT_BRANCH" ] && [ "$CURRENT_BRANCH" != "{BASE_BRANCH}" ]; then
      git checkout "$CURRENT_BRANCH"
    fi
  fi
fi
```

额外保险：

- 如果本地存在 `{BASE_BRANCH}`，但远端不存在 `origin/{BASE_BRANCH}`，则先执行一次：

```bash
cd {repo_source_path} && git push -u origin {BASE_BRANCH}
```

这样可以确保后续始终可以基于 `origin/{BASE_BRANCH}` 创建 worktree。

#### 4.3 拉取基准分支

```bash
cd {repo_source_path} && git fetch origin {BASE_BRANCH}
```

#### 4.4 创建 worktree 和工作分支

```bash
cd {repo_source_path} && git worktree add {repo_worktree_path} -b {workspace_name} origin/{BASE_BRANCH}
```

#### 4.5 推送到远端

```bash
cd {repo_worktree_path}
git push -u origin {workspace_name}
```

> 任一仓库执行失败时，立即停止后续仓库处理，并向用户明确播报失败仓库名与失败命令。

### 5. 生成 `target-src-path.json`

在所有仓库 worktree 均创建成功后，生成 `{TARGET_SRC_PATH}` 文件。

文件内容必须是 JSON 数组，每个元素都是一个对象：

- key：仓库名 `{repo_name}`
- value：对象，包含以下字段
  - `path`：该仓库对应的 worktree 目录路径，即 `{repo_worktree_path}`
  - `desc`：用户输入的仓库描述

示例：

```json
[
  {
    "user-service": {
      "path": "/Users/bytedance/.devclaw/devclaw_user_center_login_refactor_main_20260317183000/user-service",
      "desc": "用户中心后端服务"
    }
  },
  {
    "gateway-web": {
      "path": "/Users/bytedance/.devclaw/devclaw_user_center_login_refactor_main_20260317183000/gateway-web",
      "desc": "网关前端和 BFF"
    }
  }
]
```

写入规则：

- 按用户输入顺序遍历 `TARGET_REPOS`
- 将每个仓库记录中的 `worktree_path` 和 `desc` 写入 JSON
- 不要把用户最初输入的源仓路径写入最终文件

### 6. 设置多仓变量

#### 6.1 设置 `TARGET_SRC_DIR`

为了兼容后续仍要求目录型输入的 action，将多仓模式下的 `TARGET_SRC_DIR` 设置为**整个多仓工作区根目录**：

```text
TARGET_SRC_DIR={work_dir}
```

说明：

- `TARGET_SRC_DIR` 在多仓模式下不再指向某一个具体仓库，而是指向包含所有仓库 worktree 的根目录。
- 后续 action 如需获取具体仓库列表，应读取 `TARGET_SRC_PATH` 指向的 `target-src-path.json` 文件。

向用户播报该路径。

#### 6.2 确定主仓

将用户录入的**第一个仓库**视为主仓，供 Constitution 匹配使用。

定义中间变量：

```text
PRIMARY_REPO_NAME={第一个仓库的 repo_name}
PRIMARY_TARGET_SRC_DIR={work_dir}/{PRIMARY_REPO_NAME}
```

### 7. 创建 `specs/` 和 `logs/` 目录

在工作空间目录下创建 `specs/` 和 `logs/` 目录：

```bash
mkdir -p {work_dir}/specs
mkdir -p {work_dir}/logs
```

将目录的绝对路径赋值给对应变量：

```text
SPECS_DIR={work_dir}/specs
LOG_DIR={work_dir}/logs
```

### 8. 部署 Constitution 到 `specs/`

#### 8.1 获取主仓的 remote URL

```bash
cd {PRIMARY_TARGET_SRC_DIR} && git remote get-url origin
```

将输出记为 `{repo_url}`。

#### 8.2 读取 `conf.json` 匹配 constitution

读取 `<skill_dir>/resources/conf.json` 文件，根据 `{repo_url}` 匹配对应的配置条目，提取 `constitution` 字段的值，记为 `{constitution_name}`。

> 如果 `{repo_url}` 在 conf.json 中没有匹配到任何条目，则**跳过本步骤**，向用户提示："未找到该主仓对应的 Constitution 配置，跳过 Constitution 部署。"

#### 8.3 复制 constitution 文件到 `specs/`

将 `<skill_dir>/resources/constitution/{constitution_name}/` 目录中的**所有 `.md` 文件**平铺复制到 `{SPECS_DIR}/` 目录下：

```bash
cp <skill_dir>/resources/constitution/{constitution_name}/*.md {SPECS_DIR}/
```

#### 8.4 设置 `CONSTITUTION_DOC` 变量

将主 constitution 文件的绝对路径赋值给变量：

```text
CONSTITUTION_DOC={SPECS_DIR}/constitution.md
```

向用户播报：已基于主仓 `{PRIMARY_REPO_NAME}` 部署 Constitution（{constitution_name}）到 `specs/` 目录。

### 9. 提示用户创建 PRD 文档

提醒用户在 `specs/` 目录下创建 PRD 文档，文件名为 `prd.md`。

向用户输出提示信息：

```text
请在 specs/ 目录下创建 PRD 文档：{SPECS_DIR}/prd.md
```

### 10. 完成

向用户输出：

```text
工作空间初始化完毕！

关键变量汇总：
- WORK_DIR={work_dir}
- TARGET_SRC_DIR={TARGET_SRC_DIR 的值}
- TARGET_SRC_PATH={TARGET_SRC_PATH 的值}
- SPECS_DIR={SPECS_DIR 的值}
- LOG_DIR={LOG_DIR 的值}
- CONSTITUTION_DOC={CONSTITUTION_DOC 的值}（如已部署）

已创建的仓库 worktree：
- {repo_name_1} -> {work_dir}/{repo_name_1}
- {repo_name_2} -> {work_dir}/{repo_name_2}
- ...
```

到此结束，不要再执行任何额外操作。
