---
name: create-workspace
description: 'DevClaw 单仓工作空间初始化 - 在单仓同级目录创建 worktree、部署 Constitution、初始化 specs 目录。当用户表达初始化工作空间的意图时使用，如"初始化"、"初始化工作空间"、"init"、"devclaw init"、"开始新项目"。'
argument-hint: "<需求名称> [BASE_BRANCH]"
---

# DevClaw 单仓工作空间初始化

当用户表达初始化工作空间的意图时触发，包括但不限于：
"初始化"、"初始化工作空间"、"init"、"devclaw init"、"开始新项目"等。

> 兼容旧口径：如果用户说的是 `costudio init`，也按本流程执行。

## 执行流程

### 1. 信息收集

依次向用户询问以下变量的值（使用 AskUserQuestion 工具逐一询问）：

1. **REQ_NAME**：本次需求名称简述
2. **BASE_BRANCH**：基准分支名称（如 `main`、`master`、`develop` 等）
3. **目标代码仓库的本地路径**：仓库本地绝对路径

收集完成后：

- 从仓库本地路径中提取目录名，记为 `{repo_name}`。
- 校验该路径存在且是一个 git 仓库。

### 2. 生成 FEATURE_NAME

#### 2.1 翻译需求名称

将 `REQ_NAME` 翻译并整理为英文极简标识，要求：

- **最多 4 个单词**
- 全部小写
- 使用 `-` 连接（kebab-case）
- 仅保留适合作为目录名 / 分支名的字符

记为 `{req_slug}`。

示例：

```text
REQ_NAME="用户中心登录改造"
req_slug="user-login-refactor"
```

#### 2.2 生成 FEATURE_NAME

格式：

```text
{YYYYMMDDHHmmss}-{req_slug}
```

其中时间戳为**当前时间**，精确到秒。

例如：

```text
FEATURE_NAME=20260319183000-user-login-refactor
```

将此名称同时作为 worktree 分支名。

### 3. 确保基准分支存在

在执行 worktree 逻辑前，先对仓库做基准分支兜底检查。

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

### 4. 拉取基准分支并创建 worktree

#### 4.1 拉取基准分支

```bash
cd {repo_source_path} && git fetch origin {BASE_BRANCH}
```

#### 4.2 创建 worktree 和工作分支

在仓库的**同级目录**下创建 worktree：

```bash
cd {repo_source_path} && git worktree add ../{FEATURE_NAME} -b {FEATURE_NAME} origin/{BASE_BRANCH}
```

将 worktree 目录的绝对路径记为 `{worktree_path}`（即 `{repo_source_path}/../{FEATURE_NAME}` 解析后的绝对路径）。

#### 4.3 推送到远端

```bash
cd {worktree_path}
git push -u origin {FEATURE_NAME}
```

> 执行失败时，立即停止并向用户明确播报失败命令。

### 5. 创建 `specs/{FEATURE_NAME}/` 和 `logs/` 目录

在 worktree 目录下创建目录：

```bash
mkdir -p {worktree_path}/specs/{FEATURE_NAME}/logs
```

将目录的绝对路径赋值给对应变量：

```text
SPECS_DIR={worktree_path}/specs/{FEATURE_NAME}
LOG_DIR={worktree_path}/specs/{FEATURE_NAME}/logs
```

### 6. 部署 Constitution 到 `specs/{FEATURE_NAME}/`

#### 6.1 获取仓库的 remote URL

```bash
cd {worktree_path} && git remote get-url origin
```

将输出记为 `{repo_url}`。

#### 6.2 读取 `conf.json` 匹配 constitution

读取 `<skill_dir>/resources/conf.json` 文件，根据 `{repo_url}` 匹配对应的配置条目，提取 `constitution` 字段的值，记为 `{constitution_name}`。

> 如果 `{repo_url}` 在 conf.json 中没有匹配到任何条目，则**跳过本步骤**，向用户提示："未找到该仓库对应的 Constitution 配置，跳过 Constitution 部署。"

#### 6.3 复制 constitution 文件到 `specs/{FEATURE_NAME}/`

将 `<skill_dir>/resources/constitution/{constitution_name}/` 目录中的**所有 `.md` 文件**平铺复制到 `{SPECS_DIR}/` 目录下：

```bash
cp <skill_dir>/resources/constitution/{constitution_name}/*.md {SPECS_DIR}/
```

#### 6.4 设置 `CONSTITUTION_DOC` 变量

```text
CONSTITUTION_DOC={SPECS_DIR}/constitution.md
```

向用户播报：已基于仓库 `{repo_name}` 部署 Constitution（{constitution_name}）到 `specs/{FEATURE_NAME}/` 目录。

### 7. 提示用户创建 PRD 文档

提醒用户在 `specs/{FEATURE_NAME}/` 目录下创建 PRD 文档：

```text
请在 specs/{FEATURE_NAME}/ 目录下创建 PRD 文档：{SPECS_DIR}/prd.md
```

### 8. 完成

向用户输出：

```text
工作空间初始化完毕！

关键变量汇总：
- FEATURE_NAME={FEATURE_NAME}
- WORK_DIR={worktree_path}
- SPECS_DIR={SPECS_DIR 的值}
- LOG_DIR={LOG_DIR 的值}
- CONSTITUTION_DOC={CONSTITUTION_DOC 的值}（如已部署）

Worktree 已创建：
- {worktree_path} (分支: {FEATURE_NAME})
```

到此结束，不要再执行任何额外操作。
