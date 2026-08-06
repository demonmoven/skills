# fixloop-pytest 参数定义

## 必填参数（标 `*` 表示 workspace 模式下可从嗅探填）

| 参数 | 必填 | 默认 | 说明 |
|---|---|---|---|
| `PSM` | 条件* | — | TCE 服务标识;workspace 模式可由嗅探自动填 |
| `BRANCH` | 是 | — | 业务分支，同时是测试仓库推送分支 |
| `BUSINESS_REPO` | 是 | — | 单仓路径（有 `go.mod`）或 workspace 路径 |
| `TEST_REPO` | 是 | `git@code.byted.org:flow/qa_model_effect.git` | 测试仓库路径或 Git 地址。**默认仓库 `qa_model_effect`**：本 skill 面向 Doubao/Flow 创作（creativity）业务，测试用例统一落在该仓库的 `testcases/bots/aigc_case`、`testcases/bots/aigc_fangzhou`、`testcases/bots/aigc_video` 三个目录下（详见 `pytest-test-discovery.md` Step A）。上下文推断优先：若 CWD（或其祖先）目录基名为 `qa_model_effect` 且是 git 仓库，直接取该路径为 `TEST_REPO_PATH`，跳过 clone |
| `SPEC_DIR` | 是 | — | 需求文档目录（markdown） |
| `IDL_REPO` | 是 | `git@code.byted.org:flow/alice_idl.git` | IDL 仓库路径或 Git 地址 |
| `IDL_BRANCH` | 是 | — | IDL 仓库中变更分支 |

## 可选参数

| 参数 | 默认 | 说明 |
|---|---|---|
| `RUNTIME_ENV` | `online` | `online` / `i18nalisg` / `i18nmaliva`；注入 pytest 环境变量 |
| `TEST_CONCURRENCY` | `30` | `pytest -n N` |
| `TEST_RERUNS` | `1` | `pytest --reruns N` |
| `TEST_MARKER` | 空 | `pytest -m '<expr>'` 的表达式；例：`l0`、`not no_devops` |
| `MAX_ITERATIONS` | `10` | 最大迭代次数；`0` 表示只跑 Stage 0，不进循环 |
| `GENERATE_TESTS` | `true` | 是否跑 Stage 0 测试生成 |
| `SINGLE_TEST_RUN` | `false` | 只跑一轮部署+测试，不进修复循环 |
| `TCE_LANE` | 自动（`ppe_flow_<rand4>`） | PPE 泳道名；**同时作为 `ENV_LABEL` 注入 pytest** |
| `SKIP_DEPLOY` | `false` | 跳过首轮部署（需已有 `TCE_LANE`） |
| `BYTEDCLI_SITE` | `ppe` | bytedcli 站点（Doubao creativity 业务默认 PPE） |
| `OUTPUT_DIR` | `.costudio` | 输出基目录 |

## 关键语义

**`TCE_LANE` 一变量三含义：**
1. TCE 部署目标泳道名
2. 测试运行时注入给 pytest 的 `ENV_LABEL` 环境变量
3. 请求染色 header `x-tt-env` 的取值

所以 `TCE_LANE` 必须遵循泳道命名约定：字母 + 数字 + 下划线，不超过 32 字符。PPE 环境以 `ppe_` 开头（默认），BOE 环境以 `boe_` 开头（特殊场景）。

## 仓库路径解析

### `BUSINESS_REPO`

```
if BUSINESS_REPO 以 "/" 开头（本地路径）:
  BUSINESS_REPO_PATH = BUSINESS_REPO
  判断单仓 vs workspace：看是否存在 $BUSINESS_REPO/go.mod
  对单仓：
    CURRENT_BRANCH=$(cd $BUSINESS_REPO_PATH && git branch --show-current)
    if [ "$CURRENT_BRANCH" != "$BRANCH" ]:
      git checkout $BRANCH
  对 workspace：
    不自动切分支（每个 sub-repo 独立）；workspace-target-detection.md 负责嗅探
if 以 "git@" / "https://" 开头:
  仅对单仓模式有效：git clone --branch $BRANCH <地址> $OUTPUT_DIR/repos/<仓库名>
  workspace 模式不支持 Git URL clone（需要用户本地准备好 workspace）
```

### `TEST_REPO`

```
if TEST_REPO 以 "/" 开头:
  TEST_REPO_PATH = TEST_REPO
if 以 Git URL 开头:
  DEFAULT_BRANCH=$(git ls-remote --symref <地址> HEAD | grep 'ref:' | awk '{print $2}' | sed 's|refs/heads/||')
  git clone --branch $DEFAULT_BRANCH <地址> $OUTPUT_DIR/repos/<仓库名>
  TEST_REPO_PATH = $OUTPUT_DIR/repos/<仓库名>
# 测试仓库的 $BRANCH 分支由 Stage 0 Step 0.4 创建并推送
```

### `IDL_REPO`

```
if IDL_REPO 以 "/" 开头:
  IDL_REPO_PATH = IDL_REPO
if 以 Git URL 开头:
  git clone --branch $IDL_BRANCH <地址> $OUTPUT_DIR/repos/<仓库名>
  IDL_REPO_PATH = $OUTPUT_DIR/repos/<仓库名>
```

## 上下文推断规则（降低必填摩擦）

在向用户交互式询问之前，按以下顺序自动推断缺失参数：

| 参数 | 推断策略 |
|---|---|
| `BUSINESS_REPO` | 对话上下文中提及的业务仓库路径；或 CWD 是 Go 仓库时取 CWD |
| `BRANCH` | `cd $BUSINESS_REPO_PATH && git branch --show-current`（仅当 BUSINESS_REPO 已知） |
| `SPEC_DIR` | 对话上下文中提及的 spec/需求文档路径（如 sdd-be 产出的 `specs/<feature>/`） |
| `PSM` | 单仓模式下从 `$BUSINESS_REPO/conf/app.yaml` / `conf/*.toml` 的 PSM 字段解析 |
| `TEST_REPO` | ①若 CWD 或其祖先目录基名为 `qa_model_effect` 且是 git 仓库 → 取该绝对路径；②否则默认 Git 地址 `git@code.byted.org:flow/qa_model_effect.git`（按"仓库路径解析"小节 clone 到 `$OUTPUT_DIR/repos/`） |
| `IDL_REPO` | 默认 `git@code.byted.org:flow/alice_idl.git`；如 clone 失败则问用户 |
| `IDL_BRANCH` | `cd $IDL_REPO_PATH && git branch --show-current`（仅当已 checkout 过） |

**推断结果必须展示给用户确认**（类似 sdd-be 的 onboarding 风格），用户确认 / 修改后再进入前置检查。
