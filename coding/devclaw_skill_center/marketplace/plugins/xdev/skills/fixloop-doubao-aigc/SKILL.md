---
name: fixloop-doubao-aigc
description: Doubao/Flow 创作（creativity）业务的全链路测试修复循环。测试生成 → 部署到 PPE 泳道（bytedcli env 套件）→ pytest 执行 → 失败分析 → 跨仓修复（Go 业务 + Python 测试）→ 循环。默认测试仓库 `qa_model_effect`（`git@code.byted.org:flow/qa_model_effect.git`），生成的测试仅落在三个 AIGC 目录：`testcases/bots/aigc_case`（通用 AIGC）、`testcases/bots/aigc_fangzhou`（方舟 text2image/image2image）、`testcases/bots/aigc_video`（视频生成）。业务仓库支持单仓（有 go.mod）或 workspace（自动嗅探目标 sub-repo）。
argument-hint: "[PSM=xxx BRANCH=xxx BUSINESS_REPO=xxx TEST_REPO=xxx SPEC_DIR=xxx IDL_REPO=xxx IDL_BRANCH=xxx] [RUNTIME_ENV=online] [TEST_MARKER='l0'] [TEST_CONCURRENCY=30] [MAX_ITERATIONS=10] [GENERATE_TESTS=true] [SINGLE_TEST_RUN=false]"
---

# fixloop-doubao-aigc: Doubao AIGC 全链路测试修复循环

编排完整的 **测试生成 → 部署 → 测试 → 分析 → 修复** 循环。

> **关键原则：前置收集，自动执行。** 在循环开始前一次性向用户确认所有参数。确认后全程自动执行，不再向用户提问。

## 执行策略：Subagent 分派

> **每个大步骤必须使用 Agent 工具启动 subagent 执行。** 主 context 仅负责编排（参数收集、前置检查、流程控制、进度汇报），不直接执行具体的代码探索、生成、分析或修复工作。

**原因**：各 Stage 涉及大量代码读取和生成，直接在主 context 执行会快速耗尽上下文窗口，导致后续步骤质量下降。

**分派规则**：

| 步骤 | Subagent 类型 | 说明 |
|------|-------------|------|
| 前置检查 · workspace 嗅探 | `general-purpose` | 读取 SPEC + IDL + workspace git 状态，执行 workspace-target-detection.md 算法（仅 workspace 模式） |
| Step 0.1 用户动线提取 | `general-purpose` | 读取 SPEC + 生成 user_journeys.md |
| Step 0.2 覆盖度分析 | `general-purpose` | 分析覆盖差距 |
| Step 0.3 E2E 测试生成 | `general-purpose` | 探索测试仓库 + 生成测试代码 |
| Step 0.5 覆盖度分析 | `general-purpose` | 分析 E2E 盲区 |
| Step 0.6 单元测试生成 | `general-purpose` | 探索业务代码 + 生成单元测试 |
| Stage 1 部署 | `general-purpose` | 执行 TCE 部署流程 |
| Stage 2 集成测试 | `general-purpose` | 执行测试 + 解析结果 |
| Stage 3 失败分析 | `general-purpose` | Judge 角色分析失败根因 |
| Stage 4 代码修复 | `general-purpose` | Fixer 角色修复代码 |
| Stage 1.5 AGW IDL 更新 | 主 context 直接执行 | 简单的命令调用和 JSON 解析，无需 subagent |
| Step 0.4/0.7, Stage 5 | 主 context 直接执行 | 简单的文件同步和摘要，无需 subagent |

**Subagent prompt 模板**：

启动 subagent 时，prompt 必须包含以下内容：
1. **角色和任务描述** — 明确说明要执行的步骤
2. **参考文档路径** — 告知 subagent 读取哪个 references/*.md 文件作为执行指南
3. **所有相关参数** — 包括路径、变量等（使用绝对路径）。对于需要执行 bytedcli 命令的阶段（Stage 1/1.5/3/4），必须传入 `BYTEDCLI_SKILLS_DIR=$BYTEDCLI_SKILLS_DIR`，并指导 subagent：先读取对应的 bytedcli skill 文件获取调用方式和命令格式，执行命令前先 `--help` 确认当前版本参数
4. **上游产出路径** — 前序步骤产出的文件路径，供 subagent 读取
5. **预期产出** — 明确说明 subagent 需要产出什么文件，写到什么路径

**示例**（Step 0.1）：
```
启动 subagent，prompt 内容：
"你的任务是执行 fixloop Stage 0 Step 0.1：用户动线提取。

请先读取指南文件：/path/to/skills/fixloop/references/craft-stage1-user-journeys.md
然后按照其中的流程执行。

参数：
- spec_dir: /home/user/specs
- idl_repo_path: /home/user/idl
- output_dir: /path/to/.costudio/craft

产出：将完整的用户动线文档写入 /path/to/.costudio/craft/user_journeys.md"
```

**无依赖的步骤可以并行启动 subagent**。例如：Step 0.1 完成后，Step 0.2（覆盖度分析）和 Step 0.3（E2E 测试生成）可以并行启动，因为二者都只依赖 Step 0.1 的产出。

**Subagent 完成后**，主 context 应：
1. 简要汇报该步骤的结果（如"Step 0.1 完成，生成了 12 条用户动线"）
2. 检查产出文件是否存在
3. 继续下一步骤

---

## 参数解析

从 `$ARGUMENTS` 解析 KEY=VALUE 格式参数。完整参数定义见 `references/parameter-defaults.md`。

| 参数 | 必填 | 默认值 | 说明 |
|------|------|--------|------|
| `PSM` | 条件* | — | TCE 服务标识 |
| `BRANCH` | 是* | — | 业务仓库的部署 & 修复分支，也用于测试仓库推送 |
| `BUSINESS_REPO` | 是* | — | 业务代码仓库（本地路径或 Git 地址）。**支持单仓（有 go.mod）或 workspace（多 sub-repo 自动嗅探）** |
| `TEST_REPO` | 是* | — | 测试代码仓库（本地路径或 Git 地址） |
| `SPEC_DIR` | 是* | — | SPEC 需求文档目录 |
| `IDL_REPO` | 是* | — | IDL 仓库（本地路径或 Git 地址），包含已变更的接口定义 |
| `IDL_BRANCH` | 是* | — | IDL 仓库中已变更的分支名 |
| `MAX_ITERATIONS` | 否 | `10` | 最大迭代次数 |
| `GENERATE_TESTS` | 否 | `true` | 是否运行 Stage 0 测试生成 |
| `SINGLE_TEST_RUN` | 否 | `false` | 仅运行一次部署+测试，不进入修复循环 |
| `TCE_LANE` | 否 | 自动生成 | 复用已有泳道名；**同时作为 ENV_LABEL 注入 pytest，染色 x-tt-env header** |
| `SKIP_DEPLOY` | 否 | `false` | 跳过部署（需已有 TCE_LANE） |
| `BYTEDCLI_SITE` | 否 | `ppe` | bytedcli 站点（Doubao creativity 业务走 PPE；BOE 仅用于特殊调试） |
| `RUNTIME_ENV` | 否 | `online` | `online` / `i18nalisg` / `i18nmaliva`；注入 pytest 环境变量 |
| `TEST_CONCURRENCY` | 否 | `30` | `pytest -n N` 并发数 |
| `TEST_RERUNS` | 否 | `1` | `pytest --reruns N` 失败重试次数 |
| `TEST_MARKER` | 否 | 空 | `pytest -m '<expr>'` 的表达式（例：`l0`、`not no_devops`） |
| `OUTPUT_DIR` | 否 | `.costudio` | 输出基目录 |

> **\* 上下文感知**：标记为"是\*"的参数在用户未显式传入时，应**优先从当前对话上下文中推断**，而非立即询问用户。Fixloop 经常作为 devclaw-sdd-be 等上游 skill 的后续步骤运行，此时上下文中通常已经包含了相关信息。

### 上下文推断规则

在向用户交互式询问之前，按以下顺序尝试自动推断缺失的必填参数：

| 参数 | 推断策略 |
|------|---------|
| `BUSINESS_REPO` | 检查对话上下文中是否提及过业务代码仓库路径（如 sdd-be 的 CWD / TARGET_SRC_DIR）；如果当前工作目录是一个 git 仓库且上下文表明这就是业务仓库，可直接使用 |
| `BRANCH` | 检查上下文中提及的分支名；或者在 BUSINESS_REPO 已确定时，通过 `git branch --show-current` 获取当前分支 |
| `SPEC_DIR` | 检查上下文中是否有 spec/需求文档的路径（如 sdd-be 产出的 `specs/{feature_name}/` 目录） |
| `PSM` | 检查上下文中是否提及过 PSM 标识符（如 sdd-be 的部署配置、TCE 相关讨论） |
| `TEST_REPO` | 检查上下文中是否提及过测试仓库路径 |
| `IDL_REPO` | 检查上下文中是否提及过 IDL 仓库路径（如 sdd-be 使用的 IDL 仓库） |
| `IDL_BRANCH` | 检查上下文中提及的 IDL 分支名；或者在 IDL_REPO 已确定时，通过 `git branch --show-current` 获取当前分支 |

**推断流程**：
1. 先解析 `$ARGUMENTS` 中用户显式传入的参数
2. 对仍缺失的必填参数，扫描当前对话上下文尝试推断
3. 将推断结果汇总展示给用户确认（如："检测到上下文中的以下信息，将用作 fixloop 参数：BUSINESS_REPO=/home/user/backend, BRANCH=feat/xxx, SPEC_DIR=/home/user/specs/feature-a。请确认或修改。"）
4. 用户确认后，仅对仍缺失的参数交互式询问补充

### 仓库路径解析

```
BUSINESS_REPO 解析：
  如果值以 "/" 开头（本地路径）：
    BUSINESS_REPO_PATH = BUSINESS_REPO
    GIT_REPO = cd $BUSINESS_REPO_PATH && git remote get-url origin
    # 检查当前分支是否为 $BRANCH，若不是则切换
    CURRENT_BRANCH=$(cd $BUSINESS_REPO_PATH && git branch --show-current)
    if [ "$CURRENT_BRANCH" != "$BRANCH" ]; then
      cd $BUSINESS_REPO_PATH && git checkout $BRANCH
    fi
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

> **Workspace 模式**：如果 `BUSINESS_REPO` 指向一个没有 `go.mod` 但包含多个子仓库（每个子仓库含 `go.mod`）的目录，skill 自动进入 workspace 模式，使用 [workspace-target-detection.md](./references/workspace-target-detection.md) 的三路信号嗅探算法（git 改动 / IDL 变更 / SPEC 推断）识别本次要修改/部署的 sub-repo。嗅探结果写入 `$OUTPUT_DIR/detected_targets.jsonl`，后续 Stage 1/1.5/3/4 共享。

---

## 前置检查

```bash
mkdir -p $OUTPUT_DIR

# 清理上一次 fix-loop 的残留产出（仅在前置检查时执行一次）
if [ -d "$OUTPUT_DIR/craft" ]; then
  echo "清理上次 craft 产出: $OUTPUT_DIR/craft/"
  rm -rf "$OUTPUT_DIR/craft"
fi
for dir in "$OUTPUT_DIR"/iteration_[0-9]*; do
  if [ -d "$dir" ]; then
    echo "清理上次迭代数据: $dir/"
    rm -rf "$dir"
  fi
done

# 解析 BUSINESS_REPO → BUSINESS_REPO_PATH + GIT_REPO
# 解析 TEST_REPO → TEST_REPO_PATH（详见 references/parameter-defaults.md）

# 发现 bytedcli skills 目录（兼容 claude/codex/trae 等所有 agent）
# 部署走 env 套件，需要 bytedance-env + bytedance-scm + bytedance-tce + bytedance-auth 同时在位
BYTEDCLI_SKILLS_DIR=""
for dir in .*/ "$HOME"/.*/ ; do
  if [ -f "${dir}skills/bytedance-env/SKILL.md" ] \
     && [ -f "${dir}skills/bytedance-scm/SKILL.md" ] \
     && [ -f "${dir}skills/bytedance-tce/SKILL.md" ]; then
    BYTEDCLI_SKILLS_DIR="${dir}skills"
    break
  fi
done
if [ -z "$BYTEDCLI_SKILLS_DIR" ]; then
  echo "警告：未找到包含 bytedance-env / bytedance-scm / bytedance-tce 的 bytedcli skills 目录，bytedcli 命令需手动查阅 --help"
fi

# bytedcli 认证检查
# 参考 $BYTEDCLI_SKILLS_DIR/bytedance-auth/SKILL.md 获取认证命令用法
# 先 --help 确认参数，然后检查 prod 和 BOE 两个站点的认证状态
# 如果任一站点未认证，执行登录；登录失败则输出错误信息并终止流程

# AGW service-id 查找（仅执行一次，缓存结果）
# 参考 $BYTEDCLI_SKILLS_DIR/bytedance-agw/SKILL.md 获取 service 搜索命令用法
# 用 PSM 作为关键字搜索，提取第一个非空的 service_id
AGW_SERVICE_ID=""
# ... 执行搜索命令，解析 JSON 结果 ...

if [ -z "$AGW_SERVICE_ID" ]; then
  echo "提示：未找到 PSM=$PSM 对应的 AGW service-id，将跳过 AGW IDL 更新步骤"
fi

# 校验参数组合
if [ "$SKIP_DEPLOY" = "true" ] && [ -z "$TCE_LANE" ]; then
  echo "错误：SKIP_DEPLOY=true 必须同时指定 TCE_LANE 参数，否则测试请求无法路由到目标泳道"
  # 终止流程
fi

ls $SPEC_DIR && ls $BUSINESS_REPO_PATH && ls $TEST_REPO_PATH && ls $IDL_REPO_PATH
```

---

## Stage 0: 测试生成 [条件: GENERATE_TESTS=true]

当 `GENERATE_TESTS=true`（默认）时执行以下步骤。否则跳到迭代循环。

### Step 0.1: 用户动线提取 [Subagent]

> **使用 Agent 工具启动 subagent 执行。** Prompt 中指定读取 `references/craft-stage1-user-journeys.md` 作为执行指南。

**参数**：
- `spec_dir`: $SPEC_DIR
- `idl_repo_path`: $IDL_REPO_PATH
- `output_dir`: $OUTPUT_DIR/craft

**产出**：`$OUTPUT_DIR/craft/user_journeys.md`

### Step 0.2: 覆盖度分析（Stage1 → Stage2） [Subagent]

> **使用 Agent 工具启动 subagent 执行。** Prompt 中指定读取 `references/craft-coverage-analysis.md`，执行模式 A。

**参数**：
- `spec_dir`: $SPEC_DIR
- `idl_repo_path`: $IDL_REPO_PATH
- `output_dir`: $OUTPUT_DIR/craft
- `mode`: stage1_to_stage2

**产出**：`$OUTPUT_DIR/craft/stage1_to_stage2_coverage.md`

### Step 0.3: E2E 测试生成 [Subagent]

> **使用 Agent 工具启动 subagent 执行。** Prompt 中指定读取三个指南：`references/craft-stage2-e2e-generation.md`（编排）+ `references/pytest-test-discovery.md`（邻居驱动协议）+ `references/qa-model-effect-json-schema.md`（JSON 字段规范）。此步骤涉及大量代码探索和生成，必须使用 subagent。

**参数**：
- `spec_dir`: $SPEC_DIR
- `e2e_test_repo`: $TEST_REPO_PATH
- `idl_repo_path`: $IDL_REPO_PATH
- `output_dir`: $OUTPUT_DIR/craft

**产出**：`$OUTPUT_DIR/craft/e2e_work_copy/` + `$OUTPUT_DIR/craft/generated_test_cases.jsonl`

### Step 0.4: 同步 E2E 测试 + 生成 pytest_selectors.txt

```bash
# 同步测试代码到本地测试仓库
cp -r $OUTPUT_DIR/craft/e2e_work_copy/* $TEST_REPO_PATH/
cd $TEST_REPO_PATH
# 基于远程默认分支创建 $BRANCH（若不存在）
DEFAULT_BRANCH=$(git remote show origin | grep 'HEAD branch' | awk '{print $NF}')
git checkout "$DEFAULT_BRANCH" && git pull
git checkout -b "$BRANCH" 2>/dev/null || git checkout "$BRANCH"
git add -A
git commit -m "feat(iter $ITERATION): auto-generated E2E pytest cases" || echo "(空提交已跳过)"
git push origin "$BRANCH"

# 从 JSONL 提取 pytest node-id 列表
python3 <<'PY' > $OUTPUT_DIR/craft/pytest_selectors.txt
import json
seen = set()
with open("$OUTPUT_DIR/craft/generated_test_cases.jsonl") as f:
    for line in f:
        line = line.strip()
        if not line:
            continue
        case = json.loads(line)
        test_file = case["test_file"]  # e.g. testcases/im/chat/test_xxx.py
        test_func = case["test_func"]
        param_index = case.get("param_index", 0)
        selector = f"{test_file}::{test_func}[params{param_index}-Env.Online]"
        if selector not in seen:
            seen.add(selector)
            print(selector)
PY
```

(Note: the `[params{N}-Env.Online]` node-id suffix is pytest's auto-generated parametrize ID. Stage 2 runs `pytest --collect-only` to normalize if environment combinations differ.)

### Step 0.5: 覆盖度分析（Stage2 → Stage3） [Subagent]

> **使用 Agent 工具启动 subagent 执行。** Prompt 中指定读取 `references/craft-coverage-analysis.md`，执行模式 B。

**参数**：
- `spec_dir`: $SPEC_DIR
- `idl_repo_path`: $IDL_REPO_PATH
- `output_dir`: $OUTPUT_DIR/craft
- `mode`: stage2_to_stage3
- `business_repo_path`: $BUSINESS_REPO_PATH

**产出**：`$OUTPUT_DIR/craft/stage2_to_stage3_coverage.md`

### Step 0.6: 单元测试生成 [Subagent]

> **使用 Agent 工具启动 subagent 执行。** Prompt 中指定读取 `references/craft-stage3-unit-tests.md` 作为执行指南。此步骤涉及大量代码探索和生成，必须使用 subagent。

**参数**：
- `spec_dir`: $SPEC_DIR
- `business_repo_path`: $BUSINESS_REPO_PATH
- `idl_repo_path`: $IDL_REPO_PATH
- `output_dir`: $OUTPUT_DIR/craft

**产出**：`$OUTPUT_DIR/craft/unit_tests/` + `$OUTPUT_DIR/craft/generated_unit_test_cases.jsonl`

### Step 0.7: 同步单元测试 + 生成 unit_test_dirs.txt

**Workspace 模式**：按 `$OUTPUT_DIR/detected_targets.jsonl` 分发到各 sub-repo；单仓模式退化到原流程（一个 target）。

```bash
for target in $(cat $OUTPUT_DIR/detected_targets.jsonl); do
  sub_repo=$(echo "$target" | jq -r .sub_repo_path)
  sub_repo_name=$(basename "$sub_repo")

  # 判断 craft/unit_tests/ 下是否有该 sub-repo 的产出（按目录名匹配）
  if [ -d "$OUTPUT_DIR/craft/unit_tests/$sub_repo_name" ]; then
    src="$OUTPUT_DIR/craft/unit_tests/$sub_repo_name"
  elif [ "$(ls $OUTPUT_DIR/craft/unit_tests 2>/dev/null | wc -l)" = "1" ]; then
    # 单仓模式：unit_tests/ 下只有一组，直接复制全部内容
    src="$OUTPUT_DIR/craft/unit_tests"
  else
    continue  # 该 sub-repo 没有生成单测，跳过
  fi

  cp -r "$src/"* "$sub_repo/"
  cd "$sub_repo"
  CURRENT_BRANCH=$(git branch --show-current)
  if [ "$CURRENT_BRANCH" != "$BRANCH" ]; then
    git checkout "$BRANCH" 2>/dev/null || git checkout -b "$BRANCH"
  fi
  git add -A
  if ! git diff --cached --quiet; then
    git commit -m "feat(iter $ITERATION): auto-generated Go unit tests"
    git push origin "$BRANCH"
  fi

  # 产出单测 node-id 列表到 sub-repo 本地，Stage 2 读取
  MODULE_NAME=$(head -1 go.mod | awk '{print $2}')
  python3 <<PY > "$sub_repo/unit_test_dirs.txt"
import json, sys
module = "$MODULE_NAME"
seen = set()
with open("$OUTPUT_DIR/craft/generated_unit_test_cases.jsonl") as f:
    for line in f:
        line = line.strip()
        if not line: continue
        case = json.loads(line)
        test = case.get('Test', '') or case.get('test', '')
        pkg = case.get('Package', '') or case.get('package', '')
        # 仅输出属于本 sub-repo 的单测
        if pkg.startswith(module):
            rel = pkg[len(module):].lstrip('/')
            rel_dir = './' + rel if rel else '.'
            key = f'{test} {rel_dir}'
            if key not in seen:
                seen.add(key)
                print(key)
PY
done
```

---

## 迭代循环

```
ITERATION = 1
TESTS_PASSED = false

# 注意：当 MAX_ITERATIONS=0 时，循环条件 (1 <= 0) 为假，直接跳过循环。
# 这意味着仅运行 Stage 0（如果 GENERATE_TESTS=true），然后输出最终摘要退出。

WHILE ITERATION <= MAX_ITERATIONS AND NOT TESTS_PASSED:
    Stage 1: 部署
    Stage 1.5: AGW IDL 更新 [条件: ITERATION==1 AND AGW_SERVICE_ID 非空]
    Stage 2: 集成测试
    判断: TESTS_PASSED? → 退出循环
    [SINGLE_TEST_RUN=true] → 输出结果，退出循环
    Stage 3: 失败分析
    Stage 4: 代码修复
    Stage 5: 迭代总结
    ITERATION++
END WHILE
```

---

## Stage 1: 部署 [Subagent]

> **使用 Agent 工具启动 subagent 执行。** Prompt 中指定读取 `references/deploy-procedure.md` 作为执行指南，并传入下方所有参数。Subagent 完成后需返回 TCE_LANE 泳道名。

**Workspace 模式下的多 sub-repo 部署**：主 context 遍历 `$OUTPUT_DIR/detected_targets.jsonl`，串行（非并行）对每个 target 调用 Stage 1 subagent。每次调用独立设置 `psm` / `BUSINESS_REPO_PATH` 参数，但复用同一个 `TCE_LANE`（多服务同泳道）。任一 sub-repo 部署失败 → 整个 Stage 1 失败，停止后续 target，报错退出循环。单仓模式退化为单次调用。详见 [deploy-procedure.md](./references/deploy-procedure.md) 的"Workspace 模式"章节。

**跳过条件**：仅当以下条件**全部满足**时跳过本阶段：
- `SKIP_DEPLOY=true`
- `TCE_LANE` 已指定
- `ITERATION == 1`（首轮）

> **重要**：`SKIP_DEPLOY` 仅影响首轮部署。从第 2 轮开始，Stage 4 修复了代码并 push 后必须重新部署才能验证修复效果，因此 **ITERATION >= 2 时始终执行部署**（使用 upgrade 动作）。

**参数**：
- `psm`: $PSM
- `env`: $TCE_LANE（首轮未指定则自动生成 `ppe_costudio_<随机4位>`）
- `branch`: $BRANCH
- `flow-base`: `prod`
- `standard-env`: `online_cn`（Doubao 默认；BOE 调试用 `boe`）
- `specify-dcs`: `HL:1,LF:1`（PPE 必填，CN agent/creativity 类默认双 IDC 各 1 pod）
- `ITERATION`: $ITERATION（首轮走 env create + deploy-tce；后续轮走 scm build + upgrade-tce）
- `BUSINESS_REPO_PATH`: $BUSINESS_REPO_PATH（部署失败修复时使用）
- `BYTEDCLI_SKILLS_DIR`: $BYTEDCLI_SKILLS_DIR

**部署成功后**：
- 记录 `TCE_LANE`（后续轮复用）
- 首轮额外记录 `cluster-id`（后续轮 upgrade-tce 需要，写入 `$OUTPUT_DIR/lane_state.json`）
- 报告泳道名和实例状态

**部署失败**：参考 `references/deploy-procedure.md` 中的"部署失败修复流程"章节，修复后重新部署。

---

## Stage 1.5: AGW IDL 更新 [主 context 执行]

> 在主 context 直接执行（无需 subagent）。仅包含 1 条 bytedcli 命令调用和 JSON 解析，逻辑简单。

**执行条件**（全部满足才执行）：
- `AGW_SERVICE_ID` 非空（前置检查中成功获取）
- `ITERATION == 1`（首轮必须执行；后续轮次跳过——当前 Stage 4 不修改 IDL 文件）

**步骤**：

1. 读取 `$BYTEDCLI_SKILLS_DIR/bytedance-agw/SKILL.md` 获取 AGW IDL 更新命令的用法
2. 执行前先 `--help` 确认当前版本参数
3. 调用 AGW IDL 更新命令，传入 `AGW_SERVICE_ID`、`TCE_LANE`、发布模式
4. 解析 JSON 返回结果中的 `changed` 字段判断是否有变更

**错误处理**：AGW IDL 更新失败不阻断主流程，记录警告后继续 Stage 2。

**参考文档**：详见 `references/agw-idl-update.md`。

**Workspace 模式**：`$OUTPUT_DIR/detected_targets.jsonl` 里每个 target 的 PSM 都要独立查询 AGW service-id（若存在）并更新 IDL；串行执行，任一失败只记警告不阻断。

---

## Stage 2: 集成测试 [Subagent]

> **使用 Agent 工具启动 subagent 执行。** Prompt 中指定读取两个指南：`references/test-execution.md`（执行编排）+ `references/pytest-result-format.md`（junit + allure 双路解析规则）。Subagent 完成后需返回测试通过/失败统计。

**参数**：
- `TEST_REPO_PATH`: $TEST_REPO_PATH
- `TEST_SELECTORS`:
  - 如果 `$OUTPUT_DIR/craft/pytest_selectors.txt` 存在 → 读取其内容作为 pytest node-id 列表（精确选择器）
  - 否则若 `$TEST_MARKER` 非空 → 按 marker 过滤：`testcases/ -m '$TEST_MARKER'`
  - 否则 → `testcases/`（跑测试仓库下全部）
- `RUNTIME_ENV`: $RUNTIME_ENV
- `TEST_CONCURRENCY`: $TEST_CONCURRENCY
- `TEST_RERUNS`: $TEST_RERUNS
- `TEST_MARKER`: $TEST_MARKER
- `TCE_LANE`: $TCE_LANE
- `PSM`: $PSM
- `ITERATION`: $ITERATION
- `OUTPUT_DIR`: $OUTPUT_DIR
- `BUSINESS_REPO_PATH`: $BUSINESS_REPO_PATH（单元测试执行目录）
- `UNIT_TEST_SCOPE`：遍历 `$OUTPUT_DIR/detected_targets.jsonl`；对每个 sub-repo，如果 `$sub_repo/unit_test_dirs.txt` 存在 → 跑 `go test -json`，否则跳过。

**判断测试结果**：读取 `$OUTPUT_DIR/iteration_$ITERATION/test_stats.json`：
- `failed == 0 && error == 0` → 设置 `TESTS_PASSED = true`，退出循环（注意：`skipped > 0` 但 `failed == 0` 仍视为通过，跳过的测试信息保留在最终摘要中供用户参考）
- **[条件: SINGLE_TEST_RUN=true]** → 输出测试结果，直接退出循环（不进入修复）

---

## Stage 3: 失败分析 [Subagent]

> **使用 Agent 工具启动 subagent 执行。** Prompt 中指定读取 `references/failure-analysis.md` 作为执行指南。此步骤涉及大量代码阅读和日志分析，必须使用 subagent。

**参数**：
- `ITERATION`: $ITERATION
- `FAILED_CASES_FILE`: $OUTPUT_DIR/iteration_$ITERATION/failed_cases.jsonl
- `BUSINESS_REPO_PATH`: $BUSINESS_REPO_PATH
- `TEST_REPO_PATH`: $TEST_REPO_PATH
- `OUTPUT_DIR`: $OUTPUT_DIR
- `SPEC_DIR`: $SPEC_DIR
- `PSM`: $PSM
- `BYTEDCLI_SITE`: $BYTEDCLI_SITE
- `BYTEDCLI_SKILLS_DIR`: $BYTEDCLI_SKILLS_DIR

**产出**：`$OUTPUT_DIR/iteration_$ITERATION/analysis_report.md`

---

## Stage 4: 代码修复 [Subagent]

> **使用 Agent 工具启动 subagent 执行。** Prompt 中指定读取 `references/fix-procedure.md` 作为执行指南。此步骤涉及大量代码阅读和修改，必须使用 subagent。

**参数**：
- `ITERATION`: $ITERATION
- `JUDGE_REPORT_PATH`: $OUTPUT_DIR/iteration_$ITERATION/analysis_report.md
- `BUSINESS_REPO_PATH`: $BUSINESS_REPO_PATH
- `TEST_REPO_PATH`: $TEST_REPO_PATH
- `FAILED_CASES_FILE`: $OUTPUT_DIR/iteration_$ITERATION/failed_cases.jsonl
- `OUTPUT_DIR`: $OUTPUT_DIR
- `BRANCH`: $BRANCH
- `HISTORY_SUMMARY_FILE`（可选）: $OUTPUT_DIR/iteration_$((ITERATION-1))/iteration_summary.md（仅 ITERATION > 1 时传入；该文件由上一轮 Stage 5 产出，若文件不存在则不传此参数）
- `BYTEDCLI_SKILLS_DIR`: $BYTEDCLI_SKILLS_DIR

**修复完成后**：commit + push

```bash
# 业务侧：workspace 模式遍历 detected_targets 串行 commit+push；单仓退化为一次
for target in $(cat $OUTPUT_DIR/detected_targets.jsonl); do
  sub_repo=$(echo "$target" | jq -r .sub_repo_path)
  cd "$sub_repo"
  git add -A
  git diff --cached --quiet && continue
  git commit -m "fix(iter $ITERATION): $(basename $sub_repo)"
  git push origin "$BRANCH"
done

# 测试侧（如有改动）
cd $TEST_REPO_PATH
git add -A
if ! git diff --cached --quiet; then
  git commit -m "fix(iter $ITERATION): test fixes"
  git push origin "HEAD:$BRANCH"
fi
```

---

## Stage 5: 迭代总结

读取 `references/iteration-summary-procedure.md`，按照其中的流程合成迭代经验总结。

- **首轮**：`cp $OUTPUT_DIR/iteration_1/fix_summary.md $OUTPUT_DIR/iteration_1/iteration_summary.md`
- **后续轮**：读取所有历史 `fix_summary.md` + `test_stats.json`，合成经验总结写入 `$OUTPUT_DIR/iteration_$ITERATION/iteration_summary.md`

`ITERATION++`，回到迭代循环顶部。

---

## 循环结束 & 最终摘要

输出最终摘要：

```
=== Fix-Loop 最终摘要 ===
总迭代次数: N
最终测试结果: PASSED / FAILED (X 个失败)
TCE 泳道: <lane_name>

Stage 0 产出:
  用户动线: $OUTPUT_DIR/craft/user_journeys.md
  E2E 测试: $OUTPUT_DIR/craft/e2e_work_copy/
  单元测试: $OUTPUT_DIR/craft/unit_tests/

各轮迭代统计:
  迭代 1: pass=X, fail=Y, skip=Z
  ...

输出目录: $OUTPUT_DIR/
```

---

## 迭代数据目录

详见 `references/iteration-data-format.md`。

```
$OUTPUT_DIR/
  craft/                         # Stage 0 产出
    ├── user_journeys.md
    ├── e2e_work_copy/
    ├── generated_test_cases.jsonl
    ├── pytest_selectors.txt         # 替代原 test_dirs.txt
    ├── stage1_to_stage2_coverage.md
    ├── stage2_to_stage3_coverage.md
    ├── unit_tests/
    └── generated_unit_test_cases.jsonl
  detected_targets.jsonl              # workspace 嗅探结果（workspace 模式）
  iteration_N/
    ├── pytest.log
    ├── failed_cases.jsonl
    ├── test_stats.json
    ├── analysis_report.md
    ├── fix_summary.md
    └── iteration_summary.md
```

---

## References

- 参数定义与默认值：`references/parameter-defaults.md`
- Workspace 目标嗅探算法：`references/workspace-target-detection.md`
- 用户动线提取流程：`references/craft-stage1-user-journeys.md`
- E2E pytest 生成流程：`references/craft-stage2-e2e-generation.md`
- 邻居驱动测试发现协议：`references/pytest-test-discovery.md`
- qa_model_effect JSON 字段规范：`references/qa-model-effect-json-schema.md`
- Go 单元测试生成流程：`references/craft-stage3-unit-tests.md`
- 覆盖度差距分析：`references/craft-coverage-analysis.md`
- AGW IDL 更新流程：`references/agw-idl-update.md`
- 部署流程：`references/deploy-procedure.md`
- 泳道路由探针：`references/lane-route-probe.md`
- 测试执行流程：`references/test-execution.md`
- pytest 结果解析规则：`references/pytest-result-format.md`
- 失败根因分析流程：`references/failure-analysis.md`
- 代码修复流程：`references/fix-procedure.md`
- TCE 查询命令：`references/tce-commands.md`
- 部署/测试常见问题：`references/tce-troubleshooting.md`
- 迭代数据目录格式：`references/iteration-data-format.md`
- 迭代总结合成流程：`references/iteration-summary-procedure.md`
