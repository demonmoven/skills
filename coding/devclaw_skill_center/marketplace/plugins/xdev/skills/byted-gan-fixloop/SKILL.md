---
name: byted-gan-fixloop
description: 全链路测试修复循环（字节内场版，phase 化目录组织）：测试生成 → 部署 → 测试 → 分析 → 修复。支持参数控制各环节开关和迭代次数。
argument-hint: "[PSM=xxx BRANCH=xxx BUSINESS_REPO=xxx TEST_REPO=xxx SPEC_DIR=xxx IDL_REPO=xxx IDL_BRANCH=xxx] [MAX_ITERATIONS=10] [GENERATE_TESTS=true] [SINGLE_TEST_RUN=false]"
---

# byted-gan-fixloop: 全链路测试修复循环

编排完整的 **测试生成 → 部署 → 测试 → 分析 → 修复** 循环。

> **关键原则：前置收集，自动执行。** 在循环开始前一次性向用户确认所有参数。确认后全程自动执行，不再向用户提问。

## 执行策略：Subagent 分派

> **每个大步骤必须使用 Agent 工具启动 subagent 执行。** 主 context 仅负责编排（参数收集、前置检查、流程控制、进度汇报），不直接执行具体的代码探索、生成、分析或修复工作。

**原因**：各 Stage 涉及大量代码读取和生成，直接在主 context 执行会快速耗尽上下文窗口，导致后续步骤质量下降。

**分派规则**：

| 步骤 | Subagent 类型 | 说明 |
|------|-------------|------|
| Step 0.1 用户动线提取 | `general-purpose` | 读取 SPEC + 生成 user_journeys.md |
| Step 0.2 覆盖度分析 | `general-purpose` | 分析覆盖差距 |
| Step 0.3 E2E 测试生成 | `general-purpose` | 探索测试仓库 + 生成测试代码 |
| Step 0.5 覆盖度分析 | `general-purpose` | 分析 E2E 盲区 |
| Step 0.6 单元测试生成 | `general-purpose` | 探索业务代码 + 生成单元测试 |
| Stage 1 部署 | `general-purpose` | 执行 TCE 部署流程 |
| Stage 2 集成测试 | `general-purpose` | 执行测试 + 解析结果 |
| Stage 3 失败分析 | `general-purpose` | Judge 角色分析失败根因 |
| Stage 4 代码修复 | `general-purpose` | Fixer 角色修复代码 |
| Stage 1.5 AGW IDL 同步 | 主 context 直接执行 | 变更分析 + 命令选择 + JSON 解析，无需 subagent |
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
"你的任务是执行 byted-gan-fixloop Phase 1 Step 1：用户动线提取。

请先读取指南文件：/path/to/skills/byted-gan-fixloop/references/phase-1-craft-test-case/prompts/step1-user-journeys.md
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

从 `$ARGUMENTS` 解析 KEY=VALUE 格式参数。完整参数定义见 `references/_shared/parameter-defaults.md`。

| 参数 | 必填 | 默认值 | 说明 |
|------|------|--------|------|
| `PSM` | 是* | — | TCE 服务标识 |
| `BRANCH` | 是* | — | 业务仓库的部署 & 修复分支，也用于测试仓库推送 |
| `BUSINESS_REPO` | 是* | — | 业务代码仓库（本地路径或 Git 地址） |
| `TEST_REPO` | 是* | — | 测试代码仓库（本地路径或 Git 地址） |
| `SPEC_DIR` | 是* | — | SPEC 需求文档目录 |
| `IDL_REPO` | 是* | — | IDL 仓库（本地路径或 Git 地址），包含已变更的接口定义 |
| `IDL_BRANCH` | 是* | — | IDL 仓库中已变更的分支名 |
| `MAX_ITERATIONS` | 否 | `10` | 最大迭代次数 |
| `GENERATE_TESTS` | 否 | `true` | 是否运行 Stage 0 测试生成 |
| `SINGLE_TEST_RUN` | 否 | `false` | 仅运行一次部署+测试，不进入修复循环 |
| `TCE_LANE` | 否 | 自动生成 | 复用已有泳道名 |
| `SKIP_DEPLOY` | 否 | `false` | 跳过部署（需已有 TCE_LANE） |
| `BYTEDCLI_SITE` | 否 | `boe` | bytedcli 站点 |
| `TEST_ENV` | 否 | `fornax_boe` | 测试环境名。遇到 404 时自动回退为 `boe` 重试，后续迭代沿用成功值 |
| `OUTPUT_DIR` | 否 | `.costudio` | 输出基目录 |

> **\* 上下文感知**：标记为"是\*"的参数在用户未显式传入时，应**优先从当前对话上下文中推断**，而非立即询问用户。byted-gan-fixloop 经常作为 speckit 等上游 skill 的后续步骤运行，此时上下文中通常已经包含了相关信息。

### 上下文推断规则

在向用户交互式询问之前，按以下顺序尝试自动推断缺失的必填参数：

| 参数 | 推断策略 |
|------|---------|
| `BUSINESS_REPO` | 检查对话上下文中是否提及过业务代码仓库路径（如 speckit 的 CWD / TARGET_SRC_DIR）；如果当前工作目录是一个 git 仓库且上下文表明这就是业务仓库，可直接使用 |
| `BRANCH` | 检查上下文中提及的分支名；或者在 BUSINESS_REPO 已确定时，通过 `git branch --show-current` 获取当前分支 |
| `SPEC_DIR` | 检查上下文中是否有 spec/需求文档的路径（如 speckit 产出的 `specs/{feature_name}/` 目录） |
| `PSM` | 检查上下文中是否提及过 PSM 标识符（如 speckit 的部署配置、TCE 相关讨论） |
| `TEST_REPO` | 检查上下文中是否提及过测试仓库路径 |
| `IDL_REPO` | 检查上下文中是否提及过 IDL 仓库路径（如 speckit 使用的 IDL 仓库） |
| `IDL_BRANCH` | 检查上下文中提及的 IDL 分支名；或者在 IDL_REPO 已确定时，通过 `git branch --show-current` 获取当前分支 |

**推断流程**：
1. 先解析 `$ARGUMENTS` 中用户显式传入的参数
2. 对仍缺失的必填参数，扫描当前对话上下文尝试推断
3. 将推断结果汇总展示给用户确认（如："检测到上下文中的以下信息，将用作 byted-gan-fixloop 参数：BUSINESS_REPO=/home/user/backend, BRANCH=feat/xxx, SPEC_DIR=/home/user/specs/feature-a。请确认或修改。"）
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

---

## Phase 0: 前置检查

> 主 context 直接 Read `references/phase-0-precheck/prompts/precheck.md`，按其中流程执行。
>
> 完成后主 context 应当持有以下变量供后续 phase 使用：
> - `BUSINESS_REPO_PATH`、`TEST_REPO_PATH`、`IDL_REPO_PATH`
> - `BYTEDCLI_SKILLS_DIR`（可能为空，下游 stage 容错处理）
> - `AGW_SERVICE_ID`（可能为空，phase-2 stage-1.5 据此跳过）
> - `TEST_ENV` = `fornax_boe`
> - bytedcli 双站点认证已通过

---

## Phase 1: 测试用例构造（craft-test-case）[条件: GENERATE_TESTS=true]

当 `GENERATE_TESTS=true`（默认）时执行以下步骤。否则跳到迭代循环。

### Step 0.1: 用户动线提取 [Subagent]

> **使用 Agent 工具启动 subagent 执行。** Prompt 中指定读取 `references/phase-1-craft-test-case/prompts/step1-user-journeys.md` 作为执行指南。

**参数**：
- `spec_dir`: $SPEC_DIR
- `idl_repo_path`: $IDL_REPO_PATH
- `output_dir`: $OUTPUT_DIR/craft

**产出**：`$OUTPUT_DIR/craft/user_journeys.md`

### Step 0.2: 覆盖度分析（Stage1 → Stage2） [Subagent]

> **使用 Agent 工具启动 subagent 执行。** Prompt 中指定读取 `references/phase-1-craft-test-case/prompts/coverage-analysis.md`，执行模式 A。

**参数**：
- `spec_dir`: $SPEC_DIR
- `idl_repo_path`: $IDL_REPO_PATH
- `output_dir`: $OUTPUT_DIR/craft
- `mode`: stage1_to_stage2

**产出**：`$OUTPUT_DIR/craft/stage1_to_stage2_coverage.md`

### Step 0.3: E2E 测试生成 [Subagent]

> **使用 Agent 工具启动 subagent 执行。** Prompt 中指定读取 `references/phase-1-craft-test-case/prompts/step3-e2e-generation.md` 作为执行指南。此步骤涉及大量代码探索和生成，必须使用 subagent。

**参数**：
- `spec_dir`: $SPEC_DIR
- `e2e_test_repo`: $TEST_REPO_PATH
- `idl_repo_path`: $IDL_REPO_PATH
- `output_dir`: $OUTPUT_DIR/craft

**产出**：`$OUTPUT_DIR/craft/e2e_work_copy/` + `$OUTPUT_DIR/craft/generated_test_cases.jsonl`

### Step 0.4: 同步 E2E 测试 + 生成 test_dirs.txt

```bash
# 同步测试代码到本地测试仓库
cp -r $OUTPUT_DIR/craft/e2e_work_copy/* $TEST_REPO_PATH/
cd $TEST_REPO_PATH && git checkout -b $BRANCH && git add -A && git commit -m "feat: auto-generated E2E tests" && git push origin $BRANCH

# 从 JSONL 提取精确测试用例列表
python3 -c "
import json
seen = set()
with open('$OUTPUT_DIR/craft/generated_test_cases.jsonl') as f:
    for line in f:
        line = line.strip()
        if not line: continue
        case = json.loads(line)
        test = case.get('Test', '') or case.get('test', '')
        pkg = case.get('Package', '') or case.get('package', '')
        if test and pkg:
            parts = pkg.split('/')
            idx = next((i for i, p in enumerate(parts) if p == 'test_cases'), -1)
            rel_dir = './' + '/'.join(parts[idx:]) if idx >= 0 else ('./' + '/'.join(parts[-2:]) if len(parts) >= 2 else './')
            key = f'{test} {rel_dir}'
            if key not in seen:
                seen.add(key)
                print(key)
" > $OUTPUT_DIR/craft/test_dirs.txt
```

### Step 0.5: 覆盖度分析（Stage2 → Stage3） [Subagent]

> **使用 Agent 工具启动 subagent 执行。** Prompt 中指定读取 `references/phase-1-craft-test-case/prompts/coverage-analysis.md`，执行模式 B。

**参数**：
- `spec_dir`: $SPEC_DIR
- `idl_repo_path`: $IDL_REPO_PATH
- `output_dir`: $OUTPUT_DIR/craft
- `mode`: stage2_to_stage3
- `business_repo_path`: $BUSINESS_REPO_PATH

**产出**：`$OUTPUT_DIR/craft/stage2_to_stage3_coverage.md`

### Step 0.6: 单元测试生成 [Subagent]

> **使用 Agent 工具启动 subagent 执行。** Prompt 中指定读取 `references/phase-1-craft-test-case/prompts/step6-unit-tests.md` 作为执行指南。此步骤涉及大量代码探索和生成，必须使用 subagent。

**参数**：
- `spec_dir`: $SPEC_DIR
- `business_repo_path`: $BUSINESS_REPO_PATH
- `idl_repo_path`: $IDL_REPO_PATH
- `output_dir`: $OUTPUT_DIR/craft

**产出**：`$OUTPUT_DIR/craft/unit_tests/` + `$OUTPUT_DIR/craft/generated_unit_test_cases.jsonl`

### Step 0.7: 同步单元测试 + 生成 unit_test_dirs.txt

```bash
# 同步单元测试代码到业务仓库
cp -r $OUTPUT_DIR/craft/unit_tests/* $BUSINESS_REPO_PATH/
cd $BUSINESS_REPO_PATH
CURRENT_BRANCH=$(git branch --show-current)
if [ "$CURRENT_BRANCH" != "$BRANCH" ]; then
  git checkout $BRANCH
fi
git add -A && git commit -m "feat: auto-generated unit tests" && git push origin $BRANCH

# 从 JSONL 提取单元测试列表（相对于业务仓库的路径）
cd $BUSINESS_REPO_PATH
MODULE_NAME=$(head -1 go.mod | awk '{print $2}')
python3 -c "
import json, sys
module = sys.argv[1]
seen = set()
with open(sys.argv[2]) as f:
    for line in f:
        line = line.strip()
        if not line: continue
        case = json.loads(line)
        test = case.get('Test', '') or case.get('test', '')
        pkg = case.get('Package', '') or case.get('package', '')
        if test and pkg:
            # 去掉 module 前缀得到相对路径
            if pkg.startswith(module):
                rel = pkg[len(module):]
                rel = rel.lstrip('/')
                rel_dir = './' + rel if rel else '.'
            else:
                rel_dir = './' + pkg
            key = f'{test} {rel_dir}'
            if key not in seen:
                seen.add(key)
                print(key)
" "$MODULE_NAME" "$OUTPUT_DIR/craft/generated_unit_test_cases.jsonl" > $OUTPUT_DIR/craft/unit_test_dirs.txt
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
    Stage 1.5: AGW IDL 同步 [条件: ITERATION==1 AND AGW_SERVICE_ID 非空]
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

## Phase 2: Gan Fix Loop（核心循环开始）

> 以下 5 个 stage 是 fix-loop 的核心循环主体，按 stage-1 → stage-2 → stage-3 → stage-4 → stage-5 顺序执行，直到测试全部通过或达到 MAX_ITERATIONS。

## Phase 2 / Stage 1: 部署 [Subagent]

> **使用 Agent 工具启动 subagent 执行。** Prompt 中指定读取 `references/phase-2-gan-fix-loop/stage-1-deploy/prompts/tce-deploy.md` 作为执行指南，并传入下方所有参数。Subagent 完成后需返回 TCE_LANE 泳道名。

**跳过条件**：仅当以下条件**全部满足**时跳过本阶段：
- `SKIP_DEPLOY=true`
- `TCE_LANE` 已指定
- `ITERATION == 1`（首轮）

> **重要**：`SKIP_DEPLOY` 仅影响首轮部署。从第 2 轮开始，Stage 4 修复了代码并 push 后必须重新部署才能验证修复效果，因此 **ITERATION >= 2 时始终执行部署**（使用 upgrade 动作）。

**参数**：
- `psm`: $PSM
- `env`: $TCE_LANE（首轮未指定则自动生成 `boe_costudio_<随机4位>`）
- `branch`: $BRANCH
- `flow-base`: prod
- `standard-env`: boe
- `BUSINESS_REPO_PATH`: $BUSINESS_REPO_PATH（部署失败修复时使用）

**部署成功后**：
- 记录 `TCE_LANE`（后续轮复用）
- 报告泳道名和实例状态

**部署失败**：参考 `references/phase-2-gan-fix-loop/stage-1-deploy/prompts/tce-deploy.md` 中的"部署失败修复流程"章节，修复后重新部署。

---

## Phase 2 / Stage 1.5: AGW IDL 同步 [主 context 执行]

> 在主 context 直接执行（无需 subagent）。包含变更分析、命令选择和 JSON 解析，逻辑简单。

**执行条件**（全部满足才执行）：
- `AGW_SERVICE_ID` 非空（前置检查中成功获取）
- `ITERATION == 1`（首轮必须执行；后续轮次跳过——当前 Stage 4 不修改 IDL 文件）

**步骤**：

1. 读取 `$BYTEDCLI_SKILLS_DIR/bytedance-agw/SKILL.md`，了解所有可用的 AGW IDL 相关命令及其用途
2. 执行前先 `--help` 确认当前版本参数
3. **分析 IDL 变更内容，选择合适的命令**：
   - 对比 `IDL_BRANCH` 与基线分支的 diff，判断本次变更的性质
   - **涉及新增接口、修改路由路径等路由变更** → 选择同时更新 IDL 和路由的命令（确保网关路由规则与接口定义同步生效）
   - **仅修改已有接口的字段定义（入参/出参结构变更）** → 选择仅更新 IDL 的命令即可
   - 具体命令名以 `$BYTEDCLI_SKILLS_DIR/bytedance-agw/SKILL.md` 中的文档为准，不硬编码
4. 调用选定的命令，传入 `AGW_SERVICE_ID`、`TCE_LANE`、`IDL_BRANCH`、发布模式
5. 解析 JSON 返回结果中的 `changed` 字段判断是否有变更
6. 如果更新失败，检查错误输出是否为泳道环境未注册（匹配 `泳道未注册`、`env not found`、`not registered` 等关键字）：
   - **是泳道未注册**：读取 `$BYTEDCLI_SKILLS_DIR/bytedance-agw/SKILL.md` 获取泳道注册命令，执行注册（传入 `AGW_SERVICE_ID`、`TCE_LANE`），然后重试步骤 4
   - **非泳道注册错误 或 重试仍失败**：阻断主流程，停止执行并报告错误

**错误处理**：AGW IDL 同步失败时**阻断主流程**，停止执行并向用户报告错误，不继续进入 Stage 2。唯一的自动恢复路径是泳道未注册时的自动注册重试。

**参考文档**：详见 `references/phase-2-gan-fix-loop/stage-1-deploy/prompts/agw-idl-sync.md`。

---

## Phase 2 / Stage 2: 集成测试 [Subagent]

> **使用 Agent 工具启动 subagent 执行。** Prompt 中指定读取 `references/phase-2-gan-fix-loop/stage-2-test/prompts/test-execution.md` 作为执行指南，并传入下方所有参数。Subagent 完成后需返回测试通过/失败统计。

**参数**：
- `TEST_REPO_PATH`: $TEST_REPO_PATH
- `TEST_SCOPE`（每行 TestFuncName ./dir，精确匹配执行）：
  - 如果 `$OUTPUT_DIR/craft/test_dirs.txt` 存在 → 读取其内容作为测试范围
  - 如果不存在（如 `GENERATE_TESTS=false` 跳过了 Stage 0）→ 使用 `./...` 执行测试仓库下所有测试
- `TCE_LANE`: $TCE_LANE
- `PSM`: $PSM
- `ITERATION`: $ITERATION
- `OUTPUT_DIR`: $OUTPUT_DIR
- `BUSINESS_REPO_PATH`: $BUSINESS_REPO_PATH（单元测试执行目录）
- `TEST_ENV`: $TEST_ENV
- `UNIT_TEST_SCOPE`：
  - 如果 `$OUTPUT_DIR/craft/unit_test_dirs.txt` 存在 → 读取其内容作为单元测试范围
  - 如果不存在 → 跳过单元测试执行

**404 回退处理**：Subagent 返回后，检查是否发生了 TEST_ENV 切换（subagent 会在结果中标注）。如果 TEST_ENV 被切换为 `boe`，主 context 更新 `TEST_ENV = "boe"` 供后续迭代沿用。

**判断测试结果**：读取 `$OUTPUT_DIR/iteration_$ITERATION/test_stats.json`：
- `failed_cases == 0` → 设置 `TESTS_PASSED = true`，退出循环（注意：`skipped_cases > 0` 但 `failed_cases == 0` 仍视为通过，跳过的测试信息保留在最终摘要中供用户参考）
- **[条件: SINGLE_TEST_RUN=true]** → 输出测试结果，直接退出循环（不进入修复）

---

## Phase 2 / Stage 3: 失败分析 [Subagent]

> **使用 Agent 工具启动 subagent 执行。** Prompt 中指定读取 `references/phase-2-gan-fix-loop/stage-3-analyze/prompts/failure-analysis.md` 作为执行指南。此步骤涉及大量代码阅读和日志分析，必须使用 subagent。

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

## Phase 2 / Stage 4: 代码修复 [Subagent]

> **使用 Agent 工具启动 subagent 执行。** Prompt 中指定读取 `references/phase-2-gan-fix-loop/stage-4-fix/prompts/fix-procedure.md` 作为执行指南。此步骤涉及大量代码阅读和修改，必须使用 subagent。

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
cd $BUSINESS_REPO_PATH && git add -A && git commit -m "fix: iteration $ITERATION fixes" && git push origin $BRANCH
cd $TEST_REPO_PATH && git add -A && { git diff --cached --quiet || (git commit -m "fix: iteration $ITERATION test fixes" && git push origin HEAD:$BRANCH); }
```

---

## Phase 2 / Stage 5: 迭代总结

读取 `references/phase-2-gan-fix-loop/stage-5-summarize/prompts/iteration-summary.md`，按照其中的流程合成迭代经验总结。

- **首轮**：`cp $OUTPUT_DIR/iteration_1/fix_summary.md $OUTPUT_DIR/iteration_1/iteration_summary.md`
- **后续轮**：读取所有历史 `fix_summary.md` + `test_stats.json`，合成经验总结写入 `$OUTPUT_DIR/iteration_$ITERATION/iteration_summary.md`

`ITERATION++`，回到迭代循环顶部。

---

## Phase 2: Gan Fix Loop（核心循环结束）

---

## Phase 3: 最终摘要

> 主 context 直接 Read `references/phase-3-finalize/prompts/final-summary.md`，按其中格式打印最终摘要给用户。
>
> 数据来源：主 context 持有的 `ITERATION` / `TESTS_PASSED` / `TCE_LANE` 变量 + `$OUTPUT_DIR/iteration_*/test_stats.json` 文件 + `$OUTPUT_DIR/craft/` 目录（如有）。

---

## 迭代数据目录

详见 `references/_shared/iteration-data-format.md`。

```
$OUTPUT_DIR/
  craft/                         # Stage 0 产出
    ├── user_journeys.md
    ├── e2e_work_copy/
    ├── generated_test_cases.jsonl
    ├── test_dirs.txt             # 仅 GENERATE_TESTS=true 时生成（E2E 测试列表）
    ├── unit_test_dirs.txt         # 仅 GENERATE_TESTS=true 时生成（单元测试列表）
    ├── unit_tests/
    └── ...
  iteration_N/
    ├── failed_cases.jsonl
    ├── test_stats.json
    ├── analysis_report.md
    ├── fix_summary.md
    └── iteration_summary.md
```

---

## References

按 phase 组织。phase-2 是核心循环，内部嵌套 stage 子层。

- **共享（_shared）**：
  - `references/_shared/parameter-defaults.md` — 参数定义与默认值
  - `references/_shared/iteration-data-format.md` — 迭代数据格式
  - `references/_shared/go-test-json-format.md` — go test -json 解析规则

- **Phase 0 — 前置检查**：`references/phase-0-precheck/`
  - `prompts/precheck.md` — 前置检查流程

- **Phase 1 — 测试用例构造**：`references/phase-1-craft-test-case/`
  - `prompts/step1-user-journeys.md` — 用户动线提取
  - `prompts/coverage-analysis.md` — 覆盖度差距分析（被 step2 / step5 共用）
  - `prompts/step3-e2e-generation.md` — E2E 测试生成
  - `prompts/step6-unit-tests.md` — 单元测试生成

- **Phase 2 — Gan Fix Loop（核心循环）**：`references/phase-2-gan-fix-loop/`
  - **`stage-1-deploy/`** — 部署 + AGW IDL 同步
    - `prompts/tce-deploy.md` — TCE 部署流程
    - `prompts/agw-idl-sync.md` — AGW IDL/路由同步
    - `lib/tce-commands.md` — TCE 命令参考
    - `lib/tce-troubleshooting.md` — TCE 常见问题排查
  - **`stage-2-test/`** — 集成测试 + 单元测试
    - `prompts/test-execution.md` — 测试执行流程
  - **`stage-3-analyze/`** — Judge 失败根因分析
    - `prompts/failure-analysis.md`
  - **`stage-4-fix/`** — Fixer 代码修复
    - `prompts/fix-procedure.md`
  - **`stage-5-summarize/`** — 迭代总结
    - `prompts/iteration-summary.md`

- **Phase 3 — 最终摘要**：`references/phase-3-finalize/`
  - `prompts/final-summary.md` — 最终摘要输出
