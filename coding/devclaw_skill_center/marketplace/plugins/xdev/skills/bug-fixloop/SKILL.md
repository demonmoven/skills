---
name: bug-fixloop
description: 轻量级代码修复循环（diff 驱动，通用版）：基于 git commit diff 自动生成回归测试并执行 fix-loop，支持 PROFILE 选择运行环境（bytedance-tce / none）；用户可直接 /bug-fixloop 不带参数在项目根目录启动，默认分析 HEAD~1..HEAD 范围的改动。
argument-hint: "（可选，zero-arg 即可启动；显式参数会跳过对应 wizard step）"
---

# bug-fixloop: 全链路测试修复循环

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

**示例**（Phase 1 Step 2）：
```
启动 subagent，prompt 内容：
"你的任务是执行 bug-fixloop Phase 1 Step 2：改动点抽取。

请先读取指南文件：/path/to/skills/bug-fixloop/references/phase-1-bug-craft/prompts/step2-change-points.md
然后按照其中的流程执行。

参数：
- business_repo_path: /home/user/backend
- commit_range: HEAD~1..HEAD
- output_dir: /path/to/.costudio

产出：将完整的 change points 文档写入 /path/to/.costudio/craft/change_points.md"
```

**无依赖的步骤可以并行启动 subagent**。例如：Step 0.1 完成后，Step 0.2（覆盖度分析）和 Step 0.3（E2E 测试生成）可以并行启动，因为二者都只依赖 Step 0.1 的产出。

**Subagent 完成后**，主 context 应：
1. 简要汇报该步骤的结果（如"Step 0.1 完成，生成了 12 条用户动线"）
2. 检查产出文件是否存在
3. 继续下一步骤

---

## 参数解析

bug-fixloop 支持 **zero-arg 启动**：用户可以直接 `/bug-fixloop` 不带任何参数运行，所有必填参数会通过 **Phase 0 init** 阶段的交互式 wizard 自动收集。

### 调用姿势

| 姿势 | 命令 | wizard 行为 |
|------|------|----------|
| **zero-arg** | `/bug-fixloop` | 全部参数通过 wizard 交互收集 |
| **半自动** | `/bug-fixloop PSM=xxx BRANCH=main` | 已显式给出的参数跳过对应 question，缺失的进入 wizard |
| **完全显式（兼容老用法）** | `/bug-fixloop PSM=... BRANCH=... BUSINESS_REPO=... COMMIT_RANGE=HEAD~1..HEAD [IDL_REPO=... IDL_BRANCH=...]` | wizard 跳过所有 question，仅最终汇总确认 |

无论哪种姿势，wizard 都会在最后显示**汇总确认页**让用户最终过一遍参数。

### 参数清单

| 参数 | 必填 | 默认值 | wizard 推断来源 | 说明 |
|------|------|--------|------|------|
| `PSM` | 是* | — | git remote / .tcerc / bytedcli search | TCE 服务标识（仅 PROFILE=bytedance-tce） |
| `BRANCH` | 是 | — | git --show-current | 业务仓库的部署 & 修复分支 |
| `BUSINESS_REPO` | 是 | — | $PWD（cwd） | 业务代码仓库（本地路径或 Git 地址） |
| `COMMIT_RANGE` | 是 | `HEAD~1..HEAD` | git 默认最近 1 个 commit | **git diff 范围（bug-fixloop 核心参数）** |
| `TEST_REPO` | 否* | — | 同级 *_test / *_api_test | 测试代码仓库（MVP 不必填） |
| `IDL_REPO` | 否* | — | cwd 内 idl/ → submodule → 兄弟 *-idl | IDL 仓库（仅 PROFILE=bytedance-tce,用于 AGW 同步） |
| `IDL_BRANCH` | 否* | $BRANCH | 与业务同名 / IDL repo --show-current | IDL 仓库分支（仅 PROFILE=bytedance-tce） |
| `BASE_URL` | 否* | — | — | 自部署服务地址（仅 PROFILE=none） |
| `MAX_ITERATIONS` | 否 | `10` | — | 最大迭代次数 |
| `GENERATE_TESTS` | 否 | `true` | — | 是否运行 Phase 1 bug craft |
| `SINGLE_TEST_RUN` | 否 | `false` | — | 仅运行一次部署+测试，不进入修复循环 |
| `TCE_LANE` | 否 | 自动生成 | — | 复用已有泳道名 |
| `SKIP_DEPLOY` | 否 | `false` | — | 跳过部署（需已有 TCE_LANE） |
| `BYTEDCLI_SITE` | 否 | `boe` | — | bytedcli 站点 |
| `TEST_ENV` | 否 | `fornax_boe` / `local` | — | 测试环境名 |
| `OUTPUT_DIR` | 否 | `.costudio` | — | 输出基目录 |

> **bug-fixloop 与 gan-fixloop 的参数差异**：bug-fixloop 删除了 `SPEC_DIR`（不需要 spec），新增了 `COMMIT_RANGE`（git diff 范围）。其它参数保持一致。

完整参数定义见 `references/_shared/parameter-defaults.md`。

### 交互式收集流程（wizard）

详见 `references/phase-0-init/prompts/interactive-collect.md`。该文件定义了 5 步 wizard：

1. **Step 0/5**：PROFILE 选择（bytedance-tce / none）
2. **Step 1/5**：仓库与分支（BUSINESS_REPO / BRANCH）
3. **Step 2/5**：输入源
   - PROFILE=bytedance-tce: COMMIT_RANGE / IDL_REPO / IDL_BRANCH
   - PROFILE=none: COMMIT_RANGE / BASE_URL
4. **Step 3/5**：字节内场参数 PSM（仅 PROFILE=bytedance-tce）
5. **Step 4/5**：高级选项 + 汇总确认

每一步都通过 `AskUserQuestion` 工具与用户交互，候选项基于 cwd / git / 文件系统扫描自动生成。**列出仓库类候选时，每个候选标签都会显示其 `当前分支` 信息**（例如 `/Users/x/cozeloop_backend  (当前分支: feat/my-feature)`），方便用户辨识。

### 路径解析

仓库参数解析逻辑（本地路径 vs Git 地址、克隆策略、分支切换等）见 `references/phase-0-init/prompts/precheck.md` 中的「仓库路径解析」段。

---

## Phase 0: 初始化（init）

Phase 0 分两步执行，**主 context 直接执行（不启动 subagent）**：

### Step 1：交互式参数收集

主 context 直接 Read `references/phase-0-init/prompts/interactive-collect.md`，按其中的 wizard 流程通过 `AskUserQuestion` 工具与用户交互，收集所有缺失的必填参数。

- **zero-arg 启动**：用户直接 `/bug-fixloop` 不带参数，wizard 自动从 cwd / git / 文件系统扫描候选并让用户选择
- **半自动启动**：用户传部分参数（如 `PSM=xxx`），wizard 跳过对应 question，仅询问缺失项
- **完全显式启动**：用户传全部 7 个必填参数，wizard 跳过所有 question，仅最终汇总确认

候选生成依赖：
- `references/phase-0-init/prompts/lib/path-discovery.md` — 路径自动发现算法（cwd / 父目录 / 祖父目录逐层 fallback）
- `references/phase-0-init/prompts/lib/psm-discovery.md` — PSM 自动推断（git remote / .tcerc / bytedcli search）
- `references/phase-0-init/prompts/lib/candidate-generation.md` — 候选标签格式（含 `(当前分支: xxx)` 显示）

> **关键 UX 要求**：列出仓库类候选时，每个候选的标签必须显示其当前 git 分支，例如 `/Users/x/cozeloop_backend  (当前分支: feat/my-feature)`，方便用户在多个仓库间快速辨识。

### Step 2：前置检查（PROFILE 适配）

参数收集完成后，主 context Read `references/phase-0-init/prompts/precheck.md`，按其中流程执行通用前置检查（清理输出、克隆/校验仓库等）。

precheck.md 内部根据 `PROFILE` 变量动态加载对应的 auth adapter：

| PROFILE | adapter 文件 | 行为 |
|---------|------|------|
| `bytedance-tce` | `prompts/adapters/auth-bytedance.md` | bytedcli 二进制检查 + sub-skills 检查 + 双站点认证 + AGW service-id 查找 |
| `none` | `prompts/adapters/auth-none.md` | 跳过所有 byted 相关检查 |

### Phase 0 完成后

主 context 应当持有以下变量供后续 phase 使用：

**通用变量**：
- `PROFILE`（来自 wizard Step 0）
- `BUSINESS_REPO_PATH`、`TEST_REPO_PATH`
- `BRANCH`

**PROFILE=bytedance-tce 专用**：
- `IDL_REPO_PATH`、`IDL_BRANCH`、`PSM`
- `BYTEDCLI_SKILLS_DIR`
- `AGW_SERVICE_ID`（可能为空，phase-2 stage-1.5 据此跳过）
- `TEST_ENV` = `fornax_boe`
- bytedcli 双站点认证已通过

**PROFILE=none 专用**：
- `BASE_URL`
- `TEST_ENV` = `local`

---

## Phase 1: Bug Craft（diff 驱动的测试生成）[条件: GENERATE_TESTS=true]

当 `GENERATE_TESTS=true`（默认）时执行以下步骤。否则跳到迭代循环。

**与 gan-fixloop 的核心差异**：bug-fixloop 的 Phase 1 基于 `git diff $COMMIT_RANGE` 驱动测试生成，不需要 SPEC 或 IDL。MVP 只生成 Go 单元测试（不生成 E2E）。

### Step 1: diff 解析 [主 context bash]

> 主 context 直接 Read `references/phase-1-bug-craft/prompts/step1-diff-parse.md` 并按其中流程执行。

**参数**：
- `BUSINESS_REPO_PATH`: $BUSINESS_REPO_PATH
- `COMMIT_RANGE`: $COMMIT_RANGE（默认 `HEAD~1..HEAD`）
- `OUTPUT_DIR`: $OUTPUT_DIR

**产出**：
- `$OUTPUT_DIR/craft/diff_stat.txt`
- `$OUTPUT_DIR/craft/full_diff.txt`
- `$OUTPUT_DIR/craft/changed_files.txt`
- `$OUTPUT_DIR/craft/go_changed_files.txt`（过滤后的 Go 源文件）

**失败条件**：COMMIT_RANGE 无效 / diff 为空 / 无 Go 源文件改动时阻断主流程。

### Step 2: 改动点抽取 [Subagent]

> **使用 Agent 工具启动 subagent 执行。** Prompt 中指定读取 `references/phase-1-bug-craft/prompts/step2-change-points.md` 作为执行指南。

**参数**：
- `business_repo_path`: $BUSINESS_REPO_PATH
- `output_dir`: $OUTPUT_DIR
- `commit_range`: $COMMIT_RANGE

**产出**：`$OUTPUT_DIR/craft/change_points.md`（结构化列出每个改动函数的 signature / 类别 / 影响）

### Step 3: 测试需求生成 [Subagent]

> **使用 Agent 工具启动 subagent 执行。** Prompt 中指定读取 `references/phase-1-bug-craft/prompts/step3-test-requirements.md` 作为执行指南。

**参数**：
- `business_repo_path`: $BUSINESS_REPO_PATH
- `output_dir`: $OUTPUT_DIR

**产出**：`$OUTPUT_DIR/craft/test_requirements.md`（每个 change point 对应的 happy path / 边界 / 回归测试需求）

### Step 4: 测试代码生成 [Subagent]

> **使用 Agent 工具启动 subagent 执行。** Prompt 中指定读取 `references/phase-1-bug-craft/prompts/step4-test-generation.md` 作为执行指南。此步骤涉及大量业务代码探索和测试代码生成，必须使用 subagent。

**参数**：
- `business_repo_path`: $BUSINESS_REPO_PATH
- `output_dir`: $OUTPUT_DIR

**产出**：
- `$OUTPUT_DIR/craft/unit_tests/`（生成的测试代码副本）
- `$OUTPUT_DIR/craft/generated_unit_test_cases.jsonl`
- BUSINESS_REPO 中对应位置已写入测试文件（`go build ./...` 已校验通过）

**约束**：每个改动点最多 5 个测试，总共最多 30 个测试。编译失败最多重试 5 次。

### Step 5: 同步推送 [主 context bash]

> 主 context 直接 Read `references/phase-1-bug-craft/prompts/step5-sync-push.md` 并按其中流程执行。

**参数**：
- `BUSINESS_REPO_PATH`: $BUSINESS_REPO_PATH
- `BRANCH`: $BRANCH
- `COMMIT_RANGE`: $COMMIT_RANGE
- `OUTPUT_DIR`: $OUTPUT_DIR

**产出**：
- BUSINESS_REPO 上的新 commit（含所有生成的 `_test.go` 文件）
- `$OUTPUT_DIR/craft/unit_test_dirs.txt`（供 phase-2-stage-2 使用）
- 已 push 到 `$BRANCH` 分支

> **注意**：bug-fixloop **不生成** `test_dirs.txt`（E2E 测试列表），phase-2-stage-2 会自动跳过 E2E 测试。

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

> **使用 Agent 工具启动 subagent 执行。** Prompt 中指定读取 `references/phase-2-gan-fix-loop/stage-1-deploy/prompts/adapters/${PROFILE}.md` 作为执行指南，并传入下方所有参数。Subagent 完成后需返回 TCE_LANE 泳道名。

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

**部署失败**：参考 `references/phase-2-gan-fix-loop/stage-1-deploy/prompts/adapters/${PROFILE}.md` 中的"部署失败修复流程"章节，修复后重新部署。

---

## Phase 2 / Stage 1.5: AGW IDL 同步 [PROFILE 条件性，已合并]

> **bug-fixloop 改造说明**：原独立的 Stage 1.5 内容已经合并到 Stage 1 的 `adapters/bytedance-tce.md` 文件中（"AGW IDL 同步（合并自原 stage-1.5）"段）。
>
> 主 context 在执行 Stage 1 时，如果 PROFILE=bytedance-tce，adapter 会自动执行 AGW IDL 同步逻辑；如果 PROFILE=none，adapter 是 noop 桩，不涉及 AGW。
>
> 因此 SKILL.md 不再单独编排 Stage 1.5。

**执行条件**（仅 PROFILE=bytedance-tce 时通过 adapter 内部逻辑执行）：
- `AGW_SERVICE_ID` 非空（在 phase-0 auth-bytedance adapter 中获取）
- `ITERATION == 1`（仅首轮执行）

**详细步骤**：见 `references/phase-2-gan-fix-loop/stage-1-deploy/prompts/adapters/bytedance-tce.md` 文件末尾的"AGW IDL 同步"段。

---

## Phase 2 / Stage 2: 集成测试 [Subagent]

> **使用 Agent 工具启动 subagent 执行。** Prompt 中指定读取 `references/phase-2-gan-fix-loop/stage-2-test/prompts/adapters/${PROFILE}.md` 作为执行指南，并传入下方所有参数。Subagent 完成后需返回测试通过/失败统计。

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
- `TEST_REPO_PATH`: $TEST_REPO_PATH（可空,bug-fixloop MVP 不一定有）
- `OUTPUT_DIR`: $OUTPUT_DIR
- `CHANGE_POINTS_FILE`: $OUTPUT_DIR/craft/change_points.md（bug-fixloop 专用,替代原 SPEC_DIR,Judge 可参考改动点信息辅助分析）
- `PSM`: $PSM（仅 PROFILE=bytedance-tce）
- `BYTEDCLI_SITE`: $BYTEDCLI_SITE（仅 PROFILE=bytedance-tce）
- `BYTEDCLI_SKILLS_DIR`: $BYTEDCLI_SKILLS_DIR（仅 PROFILE=bytedance-tce）

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

- **Phase 0 — 初始化（init）**：`references/phase-0-init/`
  - `prompts/interactive-collect.md` — 交互式参数收集 wizard 主流程（5 步,含 Step 0 PROFILE 选择）
  - `prompts/precheck.md` — 前置检查流程（PROFILE 适配）
  - `prompts/adapters/auth-bytedance.md` — bytedcli 认证 + AGW service-id 查找（PROFILE=bytedance-tce）
  - `prompts/adapters/auth-none.md` — 跳过 byted 检查（PROFILE=none）
  - `prompts/lib/path-discovery.md` — 路径自动发现算法
  - `prompts/lib/psm-discovery.md` — PSM 自动推断算法（仅 bytedance-tce）
  - `prompts/lib/candidate-generation.md` — 候选生成与标签格式（含分支显示）

- **Phase 1 — Bug Craft（diff 驱动）**：`references/phase-1-bug-craft/`
  - `prompts/step1-diff-parse.md` — diff 解析（主 context bash）
  - `prompts/step2-change-points.md` — 改动点抽取（subagent）
  - `prompts/step3-test-requirements.md` — 测试需求生成（subagent）
  - `prompts/step4-test-generation.md` — Go 单元测试代码生成（subagent）
  - `prompts/step5-sync-push.md` — 同步推送（主 context bash）

- **Phase 2 — Gan Fix Loop（核心循环，PROFILE 适配）**：`references/phase-2-gan-fix-loop/`
  - **`stage-1-deploy/`** — 部署 + AGW IDL 同步
    - `prompts/adapters/bytedance-tce.md` — TCE 部署 + AGW IDL 同步（合并）
    - `prompts/adapters/none.md` — BASE_URL 校验（不部署）
    - `lib/tce-commands.md` — TCE 命令参考
    - `lib/tce-troubleshooting.md` — TCE 常见问题排查
  - **`stage-2-test/`** — 集成测试 + 单元测试
    - `prompts/adapters/bytedance-tce.md` — x-tt-env 注入 + fornax_boe 配置
    - `prompts/adapters/none.md` — 直接 BASE_URL 替换 test config
  - **`stage-3-analyze/`** — Judge 失败根因分析
    - `prompts/failure-analysis.md` — Judge 主流程（PROFILE 无关）
    - `prompts/adapters/log-query-bytedance.md` — bytedcli log 查询（PROFILE=bytedance-tce）
    - `prompts/adapters/log-query-none.md` — 跳过日志查询（PROFILE=none）
  - **`stage-4-fix/`** — Fixer 代码修复
    - `prompts/fix-procedure.md`（运行时日志收集仅 bytedance-tce 可用）
  - **`stage-5-summarize/`** — 迭代总结
    - `prompts/iteration-summary.md`

- **Phase 3 — 最终摘要**：`references/phase-3-finalize/`
  - `prompts/final-summary.md` — 最终摘要输出
