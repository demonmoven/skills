# bug-fixloop 参数定义与默认值

## 参数表

| 参数 | 必填 | 默认值 | 说明 | 示例 |
|------|------|--------|------|------|
| `PSM` | 是* | — | TCE 服务标识 | `stone.cozeloop.prompt` |
| `BRANCH` | 是* | — | 业务仓库的部署 & 修复分支，也用于测试仓库推送 | `main` |
| `BUSINESS_REPO` | 是* | — | 业务代码仓库（本地路径或 Git 地址） | `/Users/xxx/backend` 或 `git@code.byted.org:xxx/repo.git` |
| `TEST_REPO` | 是* | — | 测试代码仓库（本地路径或 Git 地址） | `/Users/xxx/api_test` 或 `git@code.byted.org:xxx/test.git` |
| `COMMIT_RANGE` | 是* | `HEAD~1..HEAD` | **git diff 范围（bug-fixloop 核心参数）** | `HEAD~1..HEAD`、`origin/main..HEAD`、`HEAD~3..HEAD` |
| `IDL_REPO` | 否* | — | IDL 仓库（本地路径或 Git 地址）。仅 PROFILE=bytedance-tce 时用于 AGW 同步 | `/Users/xxx/idl` |
| `IDL_BRANCH` | 否* | — | IDL 仓库中已变更的分支名。仅 PROFILE=bytedance-tce 时使用 | `feat/my-api` |
| `BASE_URL` | 否* | — | 自部署服务的 BASE URL。仅 PROFILE=none 时必填 | `http://localhost:8080` |
| `MAX_ITERATIONS` | 否 | `10` | 最大迭代次数 | `10` |
| `GENERATE_TESTS` | 否 | `true` | 是否运行 Stage 0 测试生成 | `true` |
| `SINGLE_TEST_RUN` | 否 | `false` | 仅运行一次部署+测试，不进入修复循环 | `false` |
| `TCE_LANE` | 否 | 自动生成 `boe_costudio_<随机4位>` | 复用已有泳道名 | `boe_wtj_0306` |
| `SKIP_DEPLOY` | 否 | `false` | 跳过部署（需已有 TCE_LANE） | `false` |
| `BYTEDCLI_SITE` | 否 | `boe` | bytedcli 站点 | `boe` |
| `TEST_ENV` | 否 | `fornax_boe` | 测试环境名。默认 `fornax_boe`；测试遇到 404 时自动回退为 `boe` 重试，后续迭代沿用成功值。影响配置文件路径 (`test_env/${TEST_ENV}.config.yaml`) 和 `go test` 环境变量 | `fornax_boe`, `boe` |
| `OUTPUT_DIR` | 否 | `.costudio` | 输出基目录 | `.costudio` |

> **\* 上下文感知**：标记为"是\*"的参数在用户未显式传入时，应**优先从当前对话上下文中推断**，而非立即询问用户。详见 SKILL.md 的上下文推断规则。

## $ARGUMENTS 解析规则

从 `$ARGUMENTS` 中解析 KEY=VALUE 格式的参数：

```
# 解析逻辑伪代码
for each token in $ARGUMENTS:
    if token matches "KEY=VALUE":
        set parameter KEY to VALUE
    else:
        ignore (or treat as positional)
```

**缺失必填参数处理**：

1. **上下文推断优先**：bug-fixloop 通常在项目目录中直接启动。对话上下文中可能已包含 BUSINESS_REPO（CWD）、BRANCH（当前分支）、COMMIT_RANGE（默认 HEAD~1..HEAD）等信息。对缺失的必填参数，先扫描对话上下文尝试推断。
2. **确认推断结果**：将推断到的参数汇总展示给用户确认（"检测到以下上下文信息：…，请确认或修改"）。
3. **交互式补充**：仅对推断不到的参数，向用户交互式询问补充。

## 仓库路径解析

`BUSINESS_REPO` 和 `TEST_REPO` 支持本地路径或 Git 地址：

```
BUSINESS_REPO 解析：
  如果值以 "/" 开头（本地路径）：
    BUSINESS_REPO_PATH = BUSINESS_REPO
    GIT_REPO = cd $BUSINESS_REPO_PATH && git remote get-url origin
  如果值以 "git@" 或 "https://" 开头：
    git clone --branch $BRANCH <地址> $OUTPUT_DIR/repos/<仓库名>
    BUSINESS_REPO_PATH = $OUTPUT_DIR/repos/<仓库名>
    GIT_REPO = <原始 Git 地址>

TEST_REPO 解析（注意：不使用 $BRANCH，自动检测远程默认分支）：
  如果值以 "/" 开头（本地路径）：
    TEST_REPO_PATH = TEST_REPO
    # 获取默认分支：git remote show origin | grep 'HEAD branch' | awk '{print $NF}'
  如果值以 "git@" 或 "https://" 开头：
    # 先查询远程默认分支
    DEFAULT_BRANCH=$(git ls-remote --symref <地址> HEAD | grep 'ref:' | awk '{print $2}' | sed 's|refs/heads/||')
    git clone --branch $DEFAULT_BRANCH <地址> $OUTPUT_DIR/repos/<仓库名>
    TEST_REPO_PATH = $OUTPUT_DIR/repos/<仓库名>

IDL_REPO 解析（使用 $IDL_BRANCH）：
  如果值以 "/" 开头（本地路径）：
    IDL_REPO_PATH = IDL_REPO
  如果值以 "git@" 或 "https://" 开头：
    git clone --branch $IDL_BRANCH <地址> $OUTPUT_DIR/repos/<仓库名>
    IDL_REPO_PATH = $OUTPUT_DIR/repos/<仓库名>
```

> **重要**：`BRANCH` 用于业务仓库（部署 & 修复分支）以及测试仓库（Stage 0 生成测试后推送的分支名）。`IDL_BRANCH` 仅用于 IDL 仓库。测试仓库自动检测远程默认分支（可能是 main、master、release 等），测试生成阶段（Stage 0）会在测试仓库中基于默认分支创建 `$BRANCH` 新分支并推送。

## 调用示例

```bash
# 最简（zero-arg，全部通过 wizard 收集，默认 COMMIT_RANGE=HEAD~1..HEAD）
/bug-fixloop

# 指定 commit 范围
/bug-fixloop COMMIT_RANGE=HEAD~3..HEAD

# PROFILE=bytedance-tce + 完整参数（跳过 wizard）
/bug-fixloop PSM=stone.cozeloop.prompt BRANCH=main BUSINESS_REPO=/path/backend COMMIT_RANGE=HEAD~1..HEAD IDL_REPO=/path/idl IDL_BRANCH=main

# PROFILE=none + 完整参数
/bug-fixloop BRANCH=main BUSINESS_REPO=/path/backend COMMIT_RANGE=HEAD~1..HEAD BASE_URL=http://localhost:8080

# 跳过测试生成，直接进入修复循环（使用已有的 _test.go）
/bug-fixloop PSM=xxx BRANCH=main BUSINESS_REPO=/path/backend GENERATE_TESTS=false

# 仅跑一次测试，不进入修复循环
/bug-fixloop PSM=xxx BRANCH=main BUSINESS_REPO=/path/backend COMMIT_RANGE=HEAD~1..HEAD SINGLE_TEST_RUN=true
```
