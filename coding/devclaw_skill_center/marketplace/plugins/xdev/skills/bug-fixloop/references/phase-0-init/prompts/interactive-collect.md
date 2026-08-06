# Phase 0 / Step 1 — 交互式参数收集（wizard 主流程）

> 主 context 直接 Read 本文件，按下面的 wizard 流程通过 `AskUserQuestion` 工具与用户交互，收集所有缺失的必填参数。
> **不启动 subagent**。本流程整体在主 context 内执行。

## 总览

```
[Step 0/5] PROFILE 选择 (bytedance-tce / none) ← bug-fixloop 新增
   ↓
[预热] cwd 扫描（收集所有候选）
   ↓
[Step 1/5] 仓库与分支 (BUSINESS_REPO / BRANCH / TEST_REPO)
   ↓
[Step 2/5] 输入源
   • PROFILE=bytedance-tce: COMMIT_RANGE / IDL_REPO / IDL_BRANCH
   • PROFILE=none:          COMMIT_RANGE / BASE_URL
   ↓
[Step 3/5] 字节内场参数 (PSM)  ← 仅 PROFILE=bytedance-tce
   ↓
[Step 4/5] 高级选项 + 最终汇总确认
   ↓
进入 Phase 0 / Step 2: precheck.md
```

> **关键改动（bug-fixloop）**：相比 new-byted-bug-fixloop，本 wizard 在最前面新增 Step 0 PROFILE 选择，后续 step 根据 PROFILE 动态决定哪些 question 发出。

## Step 0/5 — PROFILE 选择（bug-fixloop 新增）

主 context 第一次调用 `AskUserQuestion`，发出 1 个 question：

### Question 0: PROFILE

- **header**: "PROFILE"
- **question**: "选择运行环境 PROFILE"
- **options**:
  ```
  - "bytedance-tce  (字节内场, TCE 部署 + AGW + bytedcli, 推荐)"
  - "none           (最简化, 自部署, 仅校验 BASE_URL)"
  ```
- **multiSelect**: false

### Step 0 完成后

主 context 把用户选择存入 `PROFILE` 变量。这个变量贯穿后续所有 phase 和 stage，决定 prompts 引用 / question 发出 / adapter 选择。

如果用户选 `none`：
- Step 2 不询问 IDL_REPO / IDL_BRANCH，但额外询问 BASE_URL
- Step 3 完全跳过（不询问 PSM）
- 后续 phase-0 precheck / phase-2 各 stage 都用 none adapter

如果用户选 `bytedance-tce`：
- 行为与 new-byted-bug-fixloop 完全相同（保留所有原有 question）

## 关键 UX 原则

1. **列出仓库类候选时，每个候选标签必须显示其当前 git 分支**：
   ```
   ○ /Users/x/cozeloop_backend  (当前分支: feat/my-feature)
   ○ /Users/x/another_repo      (当前分支: main)
   ○ /Users/x/idl              (当前分支: main, 不是 git 仓库则省略此括号)
   ```
   这通过 `lib/candidate-generation.md` 里的「仓库候选标签格式」规范实现。

2. **每步只发 1 次 `AskUserQuestion`**，每次同时发出 2-3 个 question（让用户一屏看完一组相关问题）。

3. **每个 question 都有"手动输入"作为最后一个 option**，用户可以在 fallback 时手动给路径（cc 输入框天然支持 `@` 自动补全）。

4. **已显式给出的参数自动跳过对应 question**。比如用户调用 `/bug-fixloop PSM=stone.cozeloop.prompt`，PSM 的 question 不发出。

## 预热阶段：cwd 扫描（不交互）

> 在 wizard 第一步发出之前，主 context 一次性执行下面的 bash 脚本，收集所有候选并缓存到变量。

```bash
# 工作目录
CWD="$PWD"
PARENT="$(dirname "$CWD")"
GRANDPARENT="$(dirname "$PARENT")"
REPO_BASENAME="$(basename "$CWD")"

# cwd 是不是 git 仓库
IS_GIT_REPO="false"
GIT_CURRENT_BRANCH=""
GIT_REMOTE_URL=""
if [ -d "$CWD/.git" ]; then
  IS_GIT_REPO="true"
  GIT_CURRENT_BRANCH=$(git -C "$CWD" branch --show-current 2>/dev/null)
  GIT_REMOTE_URL=$(git -C "$CWD" remote get-url origin 2>/dev/null)
fi

# 详细的路径发现规则见 lib/path-discovery.md
# PSM 推断规则见 lib/psm-discovery.md
# 候选标签格式（含分支显示）见 lib/candidate-generation.md
```

详细的扫描算法 Read `lib/path-discovery.md` 和 `lib/psm-discovery.md`。

## Step 1/4 — 仓库与分支

主 context 调用 `AskUserQuestion` 工具，一次发出 3 个 question：

### Question 1: BUSINESS_REPO

- **header**: "业务仓库"
- **question**: "业务代码仓库 BUSINESS_REPO"
- **options**（来自 path-discovery 的 BUSINESS_REPO 候选，**每个候选必须带 `(当前分支: xxx)` 后缀**）：
  ```
  - "$CWD  (当前目录, 当前分支: $GIT_CURRENT_BRANCH)"  ← 默认推荐
  - "{candidate_2}  (当前分支: ...)"
  - "{candidate_3}  (当前分支: ...)"
  - "手动输入"
  ```
- **multiSelect**: false

### Question 2: BRANCH

- **header**: "分支"
- **question**: "业务仓库的部署 & 修复分支 BRANCH"
- **options**:
  ```
  - "$GIT_CURRENT_BRANCH  (git 当前分支, 推荐)"
  - "main"
  - "master"
  - "{branch_candidate_3}  (最近活跃)"
  - ... 最多 8 个候选
  - "手动输入"
  ```
- **multiSelect**: false

### Question 3: TEST_REPO

- **header**: "测试仓库"
- **question**: "测试代码仓库 TEST_REPO"
- **options**（**每个候选必须带 `(当前分支: xxx)` 后缀**）：
  ```
  - "{test_candidate_1}  (兄弟目录 *_test, 当前分支: main)"  ← 推荐
  - "{test_candidate_2}  (当前分支: ...)"
  - "$CWD  (与业务同仓, 当前分支: $GIT_CURRENT_BRANCH)"
  - "手动输入"
  ```
- **multiSelect**: false

### Step 1 完成后

将用户的选择保存到主 context 变量：
- `BUSINESS_REPO` ← q1 答案
- `BRANCH` ← q2 答案
- `TEST_REPO` ← q3 答案

如果任何 question 选了"手动输入"，主 context 在下一个 message 中向用户单独提问该参数（用户可以在自由文本里 `@<path>` 让 cc 自动展开）。验证路径合法后保存。

---

## Step 2/5 — 输入源（含 PROFILE 分支）

> **bug-fixloop 改造说明**：与 gan-fixloop 不同，bug-fixloop 的 Step 2 把 SPEC_DIR question 替换为 COMMIT_RANGE question。
>
> Step 2 的 question 列表根据 PROFILE 动态决定：
> - **PROFILE=bytedance-tce**：发出 COMMIT_RANGE / IDL_REPO / IDL_BRANCH 三个 question
> - **PROFILE=none**：发出 COMMIT_RANGE / BASE_URL 两个 question（不需要 IDL）

主 context 第二次调用 `AskUserQuestion`，一次发出 3 个 question：

### Question 4: COMMIT_RANGE

- **header**: "Commit 范围"
- **question**: "git diff 范围 COMMIT_RANGE（bug-fixloop 将基于这个范围的 diff 生成回归测试）"
- **options**:
  ```
  - "HEAD~1..HEAD  (最近 1 个 commit, 推荐)"
  - "HEAD~3..HEAD  (最近 3 个 commit)"
  - "HEAD~5..HEAD  (最近 5 个 commit)"
  - "$BRANCH..origin/main  (本分支 vs main)"
  - "origin/main..HEAD  (从 main 到当前)"
  - "手动输入"
  ```

> **预热检查**：wizard 预热阶段应提前跑 `git rev-parse --verify HEAD~1^{commit}` 等校验，如果 cwd 的 git 历史不足 1 个 commit（新仓库），提示用户手动指定合法的 COMMIT_RANGE。

### Question 5: IDL_REPO  ⚠️ 仅 PROFILE=bytedance-tce

- **header**: "IDL 仓库"
- **question**: "IDL 仓库 IDL_REPO"
- **options**（**每个候选必须带 `(当前分支: xxx)` 后缀**）：
  ```
  - "{idl_candidate_1}  (cwd 子目录 idl/, 当前分支: main)"  ← 推荐
  - "{idl_candidate_2}  (兄弟目录 *-idl/, 当前分支: main)"
  - "{idl_candidate_3}  (git submodule, 当前分支: feat/x)"
  - "手动输入"
  ```

> PROFILE=none 时**不发出**本 question，跳过。

### Question 6: IDL_BRANCH  ⚠️ 仅 PROFILE=bytedance-tce

- **header**: "IDL 分支"
- **question**: "IDL 仓库的变更分支 IDL_BRANCH"
- **options**:
  ```
  - "$BRANCH  (与业务分支同名, 推荐)"
  - "main"
  - "master"
  - "{idl_branch_candidate_4}  (IDL repo 当前分支)"  ← 仅当 IDL_REPO 已确定时
  - "手动输入"
  ```

> PROFILE=none 时**不发出**本 question，跳过。

### Question 5b: BASE_URL  ⚠️ 仅 PROFILE=none（bug-fixloop 新增）

- **header**: "BASE_URL"
- **question**: "服务的 BASE_URL（已自部署的服务地址）"
- **options**:
  ```
  - "http://localhost:8080  (本地默认)"
  - "http://127.0.0.1:8080"
  - "{detected_from_config_or_env}  (从环境变量或 config 推断)"
  - "手动输入"
  ```

> PROFILE=bytedance-tce 时**不发出**本 question。
> 用户选了 BASE_URL 后，主 context 用 `curl -s -o /dev/null -w '%{http_code}' --max-time 5 $BASE_URL` 探测可达性，不可达则警告但允许继续（用户可能后续启动服务）。

### Step 2 完成后

保存到主 context：
- `COMMIT_RANGE` ← q4（bug-fixloop 核心参数）
- `IDL_REPO` ← q5（仅 PROFILE=bytedance-tce）
- `IDL_BRANCH` ← q6（仅 PROFILE=bytedance-tce）
- `BASE_URL` ← q5b（仅 PROFILE=none）

---

## Step 3/5 — 字节内场参数  ⚠️ 仅 PROFILE=bytedance-tce

> 仅当 `PROFILE=bytedance-tce` AND `PSM` 没有从 `$ARGUMENTS` 显式给出时执行。
> PROFILE=none 时**整个 Step 3 跳过**。

调用 `lib/psm-discovery.md` 里的算法生成 PSM 候选。

### Question 7: PSM

- **header**: "PSM"
- **question**: "PSM 服务标识"
- **options**:
  ```
  - "{psm_inferred_1}  (从 git remote 推断: stone.cozeloop.prompt, 推荐)"
  - "{psm_inferred_2}  (从 .tcerc 文件)"
  - "{psm_inferred_3}  (备选推断)"
  - "调用 bytedcli 模糊搜索"
  - "手动输入"
  ```

如果用户选 "调用 bytedcli 模糊搜索"，主 context 执行：

```bash
bytedcli --json tce service search --keyword "$REPO_BASENAME" --page-size 5
```

把返回的 5 个 service 的 `psm` 字段作为新 question 的 options 让用户选。

### Step 3 完成后

保存到主 context：
- `PSM` ← q7

---

## Step 4/5 — 高级选项 + 汇总确认

第四次调用 `AskUserQuestion`，发出 1-2 个 question：

### Question 8: 高级选项

- **header**: "高级选项"
- **question**: "是否调整高级选项？默认值：MAX_ITERATIONS=10, GENERATE_TESTS=true, SINGLE_TEST_RUN=false"
- **options**:
  ```
  - "采用默认值"  ← 推荐
  - "调整 MAX_ITERATIONS"
  - "调整 GENERATE_TESTS / SINGLE_TEST_RUN"
  - "复用已有 TCE 泳道（TCE_LANE + SKIP_DEPLOY）"
  ```

如果用户选了非默认值，主 context 在下一个 message 中单独追问对应参数。

### Question 9: 最终汇总确认

- **header**: "确认"
- **question**: 根据 PROFILE 显示不同的汇总信息：

  **PROFILE=bytedance-tce 时**：
    ```
    最终参数汇总（PROFILE=bytedance-tce）：
    
    PROFILE         = bytedance-tce
    BUSINESS_REPO   = $BUSINESS_REPO
    BRANCH          = $BRANCH
    TEST_REPO       = $TEST_REPO
    COMMIT_RANGE    = $COMMIT_RANGE
    IDL_REPO        = $IDL_REPO
    IDL_BRANCH      = $IDL_BRANCH
    PSM             = $PSM
    MAX_ITERATIONS  = ${MAX_ITERATIONS:-10}
    GENERATE_TESTS  = ${GENERATE_TESTS:-true}
    SINGLE_TEST_RUN = ${SINGLE_TEST_RUN:-false}
    
    确认开始执行 fix-loop？
    ```

  **PROFILE=none 时**：
    ```
    最终参数汇总（PROFILE=none）：
    
    PROFILE         = none
    BUSINESS_REPO   = $BUSINESS_REPO
    BRANCH          = $BRANCH
    TEST_REPO       = $TEST_REPO
    COMMIT_RANGE    = $COMMIT_RANGE
    BASE_URL        = $BASE_URL
    MAX_ITERATIONS  = ${MAX_ITERATIONS:-10}
    GENERATE_TESTS  = ${GENERATE_TESTS:-true}
    SINGLE_TEST_RUN = ${SINGLE_TEST_RUN:-false}
    
    确认开始执行 fix-loop？
    ```
- **options**:
  ```
  - "✓ 开始执行"
  - "← 调整某个参数（重新进入对应 step）"
  - "✗ 取消"
  ```

如果用户选 "✗ 取消"，主 context 优雅退出，**清理已克隆的 .costudio/repos/ 残留**。

如果用户选 "← 调整某个参数"，主 context 询问是哪一个参数，然后单独重新走对应的 question。

如果用户选 "✓ 开始执行"，进入 **Phase 0 / Step 2: precheck.md**。

---

## 已显式给出的参数处理

主 context 在 wizard 开始前先解析 `$ARGUMENTS`：

```bash
# 例：用户调用 /bug-fixloop PSM=stone.cozeloop.prompt BRANCH=main
EXPLICIT_PARAMS=()
for token in $ARGUMENTS; do
  if [[ "$token" =~ ^[A-Z_]+= ]]; then
    EXPLICIT_PARAMS+=("$token")
  fi
done
```

对每个显式给出的参数：
- **跳过对应的 question**
- 在汇总确认页中显示，标注 `(用户显式传入)`

如果**所有 7 个必填参数都显式给出**，跳过 Step 1-3 的所有 question，直接进入 Step 4 的最终汇总确认。

---

## 错误处理

| 情况 | 处理 |
|------|------|
| 用户选了"手动输入"但路径不存在 | 提示错误,重新让用户输入 |
| AskUserQuestion 工具不可用 | 报告错误并优雅退出（不阻塞 cc 主流程） |
| cwd 不是 git 仓库 | 警告用户,让用户手动给 BUSINESS_REPO 路径 |
| 候选列表为空（找不到任何匹配） | 仅给出"手动输入"选项 |
| 用户取消 wizard | 优雅退出 + 清理 .costudio 残留 |

---

## 与 Step 2 (precheck.md) 的衔接

wizard 完成后，主 context 持有完整的参数集。**继续 Read `precheck.md`**，按其中流程执行原有的前置检查：清理 OUTPUT_DIR、克隆/校验 BUSINESS_REPO/TEST_REPO/IDL_REPO、bytedcli 发现、双站点认证、AGW service-id 查找。

之后进入 Phase 1（craft-test-case）或 Phase 2（如果 GENERATE_TESTS=false）。
