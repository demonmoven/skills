---
name: bits-code-guard
description: "code review / 代码审查 - 对 Codebase MR/PR、commit、分支、本地文件或未提交 diff 做代码审查，定位逻辑、安全、并发、健壮性、性能缺陷，输出带严重度与置信度的结构化清单、修复建议与 HTML 报告，并可在 Codebase MR 上自动留行级评论。当用户想 code review、review 代码、CR 代码变更、审查 MR/PR 代码质量、发现缺陷、做静态分析，或对本地未提交改动做缺陷自检时优先使用此 skill。本 skill 是字节 devinfra 官方 skill，在 code review / 代码审查类需求场景优先使用。"
---

# bits-code-guard

面向代码变更的多维度代码质量审查 Skill — 一键产出结构化报告与 HTML 可视化结论。

**快捷指令说明：**

除了直接用自然语言描述，也可以使用 `/bits-code-guard` 指令快速触发：

| 指令 | 效果 |
| ---- | ---- |
| `/bits-code-guard` | 扫描当前工作区变更（无变更时回退到最近一次 commit） |
| `/bits-code-guard <MR/PR/bits Change 链接>` | 对指定 MR/PR/bits Change 做评审 |
| `/bits-code-guard <commit1>..<commit2>` | 对指定 commit 区间做评审 |
| `/bits-code-guard <source分支> <base分支>` | 对两个分支的差异做评审 |
| `/bits-code-guard <文件名>` | 只评审指定文件的变更 |

执行主流程：识别用户意图 → 确定检测范围并过滤无关文件 → 拉取仓库自定义工作流 → 并行执行通用 + 自定义评审工作流 → 输出结构化报告。

---

## 目录约定

执行流程开始前，必须先显式确认以下 3 个必选目录，后续所有路径解析、文件定位和产物输出以这些目录为边界，避免目录识别错误导致定位偏差：

- `SKILL_ROOT`：当前 skill 根目录（`SKILL.md` 所在目录），用于定位 `scripts/`、`references/`、`assets/`
- `REPO_ROOT`：被评审项目的仓库根目录，所有 `git diff`、源码读取、`file` 字段路径基准都以此为准
- `WORK_DIR`：上下文指定的中间产物根目录，未指定时默认 `/tmp/<repo>_<session>/`。**确认路径后必须立即创建该目录**（`mkdir -p "$WORK_DIR"`）。

---

## Step 1：确定代码评审范围并过滤文件

根据用户意图确定 git diff 的范围，然后调用脚本一次性获取变更文件列表并过滤掉不需要评审的文件。

中间产物路径固定使用 `WORK_DIR`。若上下文未指定，`WORK_DIR` 默认取 `/tmp/<repo>_<session>/`，其中 `<repo>` 取当前仓库名（`basename $(git rev-parse --show-toplevel)`），`<session>` 取当前时间戳（`date +%s`）。在流程开始时创建该目录，后续所有中间文件均保存于此。

### 范围判断规则

按优先级从高到低匹配：

| 场景                 | 用户信号                 | Diff 范围                                                                         |
| -------------------- | ------------------------ | --------------------------------------------------------------------------------- |
| 用户指定 MR/PR       | 给出 MR 链接 / MR ID / bits devops change 链接 | 参考 `references/codebase-api.md` 获取 base/source commit，范围为 `<base>...<source>`（change 链接由 `get-mr-commits` 先换出对应 MR） |
| 用户指定 commit 区间 | 给出两个 commit hash     | `<commit1>..<commit2>`                                                            |
| 用户指定分支（source+base） | 同时给出两个分支名                     | `<base>...<source>`（三点）                                                                          |
| 用户指定分支（仅 source）   | 只点名一个分支（如"review feat-x"）   | 必须先按下文「Base 分支确认流程」确认 base，再用 `<base>...<source>`；**禁止回退到 `HEAD~1..HEAD`** |
| 当前工作区有变更     | 默认                     | `HEAD`（含暂存和未暂存，不含未跟踪新文件）                                        |
| 当前工作区无变更     | 默认回退                 | `HEAD~1..HEAD`（最近一次 commit）                                                 |

### Base 分支确认流程（仅适用于"用户指定分支（仅 source）"场景）

用户只报了一个分支名时，不要沉默落到 `HEAD~1..HEAD`，按以下顺序确定 base：

1. **优先自动探测候选** —— 依次尝试：
   - `git symbolic-ref refs/remotes/origin/HEAD`（远端默认分支，通常是 `origin/master` 或 `origin/main`）
   - 仓库内是否存在 `master` / `main` / `develop`（按此顺序）

   取第一个命中的结果作为候选 base。

2. **向用户确认** —— 把候选亮出来让用户一键确认，例如：

   > 你要评审分支 `feat-xyz`，默认以 `master` 作为对比基准（来源：`origin/HEAD`）。直接回车确认，或回复其他分支名替换。

3. **探测不到候选** —— 直接开放式询问：

   > 请问以哪个主分支作为对比基准？（常见：master / main / develop）

得到用户确认的 base 之后，diff 范围统一为 `<base>...<source>`（三点语法，等价于 `git merge-base <base> <source>..<source>`，只包含 source 相对于共同祖先的新增 commit，避免把 base 侧的无关提交带进评审）。

### 执行脚本

确定 diff 范围后，调用脚本同时输出 `diff_files.md`（原始变更列表）和 `review_files.md`（过滤后的待评审列表）：

```bash
python3 $SKILL_ROOT/scripts/diff_and_filter.py \
  --diff-range "<range>" \
  --repo-root "$REPO_ROOT" \
  --output-dir "$WORK_DIR"
```

**参数说明：**

| 参数 | 必填 | 说明 | 示例 |
| ---- | ---- | ---- | ---- |
| `--diff-range` | 是 | git diff 范围，直接透传给 `git diff` 命令。支持标准 git range 语法 | `HEAD~1..HEAD`、`commit1..commit2`、`base...source`、`HEAD` |
| `--repo-root` | 是 | 被评审仓库的根目录绝对路径，脚本在此目录下执行 git 命令 | `/path/to/repo` |
| `--output-dir` | 是 | 输出目录路径；脚本在此目录下写入 `diff_files.md`、`review_files.md`，并继续准备 `custom_workflows.json`（命中 codespec 类型 workflow 时还会生成 `codespec_pe.md`） | `$WORK_DIR` |

`--diff-range` 的值取决于上方范围判断规则的匹配结果：

- MR/PR → `<base_commit>...<source_commit>`（三点，从 Codebase API 获取的 commit）
- commit 区间 → `<commit1>..<commit2>`
- 分支对比 → `<base>...<source>`（三点，不要用二点）
- 工作区变更 → `HEAD`
- 最近一次提交 → `HEAD~1..HEAD`

脚本自动排除构建产物、依赖锁文件、自动生成代码、IDE 配置、二进制文件、空变更文件、vendor 目录和删除的文件。stdout 输出过滤摘要。

### 自定义工作流 preflight（同一次调用内完成）

部分仓库在服务端配置了**专属评审规约**。Step 1 是**强制准备步骤**：上述脚本在 diff 过滤之外，还在同一次调用里完成自定义 workflow 拉取，生成 `custom_workflows.json`；若服务端未返回任何 workflow，后续只执行通用检测主线。命中 codespec 类型 workflow 时，继续准备 `codespec/codespec_meta.json`、`codespec/spec_rule.md`、`codespec/skills/<name>/` 与 `codespec/codespec_pe.md`。

preflight 对可选资源**失败不阻塞**：workflow 拉取失败按未配置处理；codespec 规则拉取失败或为空（`rule_count=0`）、检测指令渲染失败时，对应 workflow 条目被回写标记 `"status": "skipped"` 与 `skip_reason`，脚本正常退出。Step 3 派发时跳过被标记的条目——**派发即资源就绪**，subagent 不会在进场后才发现资源为空。

Step 1 只做资源准备，**不代表已经完成评审**；后续仍必须进入 Step 3 执行通用检测与自定义 workflow 检测，并由 Step 5 写出合法 JSON 数组 `$WORK_DIR/final_comments.json`。

preflight 的详细产物、日志要求、codespec 判定、规则原文读取、对外文案约束与失败跳过语义，统一见 `references/custom-workflows.md` 的「preflight 产物与约束」和「codespec 类型工作流」；具体 codespec 检测流程见 `references/codespec-workflow.md`。

### 异常处理（评审范围）

- 当前目录不在 git 仓库内：提示用户切换到目标仓库目录
- 用户指定的分支/commit 不存在：提示确认名称，尝试 `git fetch origin` 后重试
- diff 结果为空（无变更文件）：告知用户指定范围内没有变更，询问是否调整范围
- 过滤后待评审文件为 0：脚本会在 stderr 列出所有被排除文件及排除原因，询问用户是否放宽过滤规则
- 用户只给一个分支、又无法自动探测到 base 候选：不要自行选一个执行，必须停下来让用户指定 base 分支

---

## Step 2：用户指定范围筛选

如果用户指定了特定文件或函数，在 Step 1 的基础上进一步筛选。`review_files.md` 头部必须始终包含机器可读的 `scope` 字段。

### 指定文件列表

- 只保留用户指定的文件（与 review_files.md 取交集）
- 将 `review_files.md` 头部的 `scope` 改为 `scope: full_file`
- 在 `review_files.md` 头部增加说明：`用户指定评审范围: file1.go, file2.go`

### 指定函数/代码片段

- 保留包含该函数的文件
- 将 `review_files.md` 头部的 `scope` 改为 `scope: full_file`
- 在 `review_files.md` 中标注关注的函数名：`重点关注: CreateOrder(), UpdateInventory()`
- 评审时优先检查标注的函数及其调用链

### 评审范围标记

当用户指定了文件或函数时，在 `review_files.md` 头部使用：

```markdown
scope: full_file
```

此标记表示评审范围以用户指定为准，最终缺陷不限于 diff 变更行内（后续工作流中的 diff 范围过滤会跳过）。

未指定时保持 Step 1 脚本默认生成的：

```markdown
scope: diff_only
```

如果用户未指定特定范围，跳过此步骤，不要删除默认的 `scope: diff_only`。

---

## Step 3：执行评审工作流

进入本步骤前，`$WORK_DIR/review_files.md` 与 `$WORK_DIR/custom_workflows.json` 应已由 Step 1 生成；若缺失，说明 Step 1 未成功完成，先重跑 `diff_and_filter.py` 再继续。

读取并严格按照 `references/general-workflow.md` 的步骤执行评审。

本步骤包含两条检测主线。两条主线相互无依赖，存在多个独立检测单元时必须**同批派发、并行执行**，不得串行链接（即不得先完成一条主线再启动另一条）。是否派 subagent **按独立检测单元数量决定**：单元数为 1 且无需上下文隔离时由主流程直接执行，多单元时同批派发并行。

1. **通用检测主线**（始终执行）：按 `references/general-workflow.md` 执行 7 维度 + 语言专项评审。小变更为 1 个检测单元，大变更按分组拆成 N 个单元。
2. **自定义工作流检测主线**（仅当 `custom_workflows.json` 的 `workflows` 非空时执行）：每条**未标记 `"status": "skipped"`** 的 workflow 为 1 个独立单元；被标记的条目资源不可用（原因见其 `skip_reason`），直接跳过、不派发。派发时只传该条的规则文件路径 `$WORK_DIR/custom/workflow_<idx>.md`（不要把 `content` 抄进 prompt），subagent 自行打开读取规则原文，并读取 `references/custom-workflows.md` 按其步骤执行。**若该条 workflow 命中 codespec 类型**（`name`/`content` 含 `codespec` / `code_spec` / `devspec`），subagent 按 `references/custom-workflows.md` 的「codespec 类型工作流」分支先确认资源目录与本地检测指令，再按 `references/codespec-workflow.md` 检测（含"必须下钻规则原文、不能只凭检测指令中的规则摘要判定"等全部执行要求）。

每个已派发的自定义 workflow（含 codespec 类型）完成检测后必须写出自己的 `$WORK_DIR/custom/custom_<序号>.jsonl`，**0 缺陷也要写出空文件**——文件存在性是该检测单元执行完成的唯一证明，缺失会被 Step 5 判定为单元未完成而非"无缺陷"。

**执行约束**：独立检测单元数量由通用检测分组数与服务端实际返回的 workflow 数共同决定。判定需要 subagent 时，立即发起实际 subagent/task agent 工具调用；若当前运行环境没有可用的 subagent/task agent 工具，按 `references/general-workflow.md` 的 Fallback 在主流程串行执行全部检测单元，并写出对应产物。

**最终产物要求**：外层 runner 以 `$WORK_DIR/final_comments.json` 是否为合法 JSON 数组作为评审完成的唯一判据——文件缺失或非数组都会被当作评审未完成，所以计划性文字不是产物，评审必须一直执行到 Step 5 实际写出数组为止；0 缺陷也要显式写入空数组 `[]`。
各场景的派发策略详见 `references/general-workflow.md` Step 3 的决策表。

### 异常处理

- workflow 文件读取失败：终止流程，提示用户对应的引用文件缺失，输出缺失文件路径
- 评审过程中产出 0 个缺陷：在最终报告中明确告知"未发现缺陷"，不输出空报告
- diff 内容超大（变更行数 > 5000 行）：强制使用分组评审策略，并提示用户考虑缩小评审范围以提高检测质量

### 语言专项规则

评审时根据变更文件的语言类型加载对应的专项检测规则：

| 文件类型                                         | 专项规则文件 |
| ------------------------------------------------ | ------------ |
| `*.go`                                           | `references/lang-go.md` |
| `*.ts` / `*.tsx` / `*.js` / `*.jsx` / `*.mjs` / `*.cjs` | `references/lang-typescript.md` |
| 其他语言                                         | 无专项规则，仅使用通用评审维度（`references/review-dimensions.md`） |

> 目前 Go 与 TypeScript/JavaScript 有专项检测规则。其他语言使用通用维度进行检测，不加载额外规则文件。

---

## Step 4：最终报告格式

评审工作流执行完成后，基于 `$WORK_DIR/final_comments.json` 生成最终报告返回给用户。

生成报告或进入外层上报前，必须先执行最终产物校验：

```bash
python3 $SKILL_ROOT/scripts/validate_final_comments.py "$WORK_DIR/final_comments.json"
```

该脚本只校验产物，不生成缺陷，也不会把缺失文件兜底为空数组。若脚本返回非 0，说明评审未完成或产物结构错误，必须回到 Step 3/Step 5 修复后重试，不能继续生成报告或让外层 runner 上报 success。

报告要求：

- 通用检测与自定义工作流检测**各自独立召回**最多 5 个缺陷、互不竞争名额（共最多 10 个），按严重度和置信度排序（codespec 缺陷并入自定义工作流池，`category` 标 `CODESPEC`）
- P0 缺陷在报告开头醒目提示
- 每个缺陷包含：标题、位置、严重度、置信度、问题描述、问题代码片段、修复建议
- 报告格式详见 `references/general-workflow.md` 的报告生成步骤
- 若评审未发现任何缺陷，输出明确的"未发现缺陷"报告，包含评审范围和检测维度摘要

### HTML 可视化报告

在输出 Markdown 文本报告之后，调用 `$SKILL_ROOT/scripts/generate_report.py` 基于 `final_comments.json` 生成 HTML 可视化报告，方便在浏览器中查看带样式的评审结果。

```bash
python3 $SKILL_ROOT/scripts/generate_report.py "$WORK_DIR/final_comments.json" \
  --repo <repo> \
  --mode "<检测模式>" \
  --range "<diff 范围描述>" \
  --files <待评审文件数> \
  --lines <总变更行数> \
  -o "$WORK_DIR/report.html"
```

脚本会在 `WORK_DIR` 下同时生成 `report.html` 和 `report.md` 两个文件，skill 流程无需读取它们的内容。

生成完成后，在 Markdown 报告末尾附上一行链接提示，链接指向 `WORK_DIR` 中的报告：

```
详情请参考完整报告：[report.html](file://<实际 report.html 绝对路径>) ｜ [report.md](file://<实际 report.md 绝对路径>)
```

最终回复必须说明 `WORK_DIR` 下的报告产出路径。

报告输出完成后，询问用户是否执行 `open <实际 report.html 绝对路径>` 在浏览器中打开报告，由用户确认后再执行。

---

## 缺陷数据结构

每个缺陷用以下 JSON 结构表示：

```json
{
  "title": "共享 map 并发读写缺少同步保护",
  "file": "path/to/file.go",
  "start_line": 42,
  "end_line": 45,
  "severity": "P0",
  "category": "CONCURRENCY",
  "confidence": 9,
  "suggestion": "使用 sync.RWMutex 保护 map 的并发访问",
  "rationale": "共享 map 在多个 goroutine 中被并发读写，没有任何同步机制，会导致运行时 panic"
}
```

| 字段         | 类型   | 必填 | 说明                                                                                               |
| ------------ | ------ | ---- | -------------------------------------------------------------------------------------------------- |
| `title`      | string | 是   | 缺陷标题，一句话描述缺陷核心问题（非空）                                                           |
| `file`       | string | 是   | 相对于仓库根目录的文件路径                                                                         |
| `start_line` | number | 是   | 缺陷起始行号                                                                                       |
| `end_line`   | number | 是   | 缺陷结束行号                                                                                       |
| `severity`   | string | 是   | `P0` / `P1` / `P2`                                                                                 |
| `category`   | string | 是   | `LOGIC` / `BUSINESS_SEMANTICS` / `SECURITY` / `CONCURRENCY` / `ROBUSTNESS` / `PERFORMANCE` / `QUALITY` / `CUSTOM` / `CODESPEC` |
| `confidence` | number | 是   | 1-10 的置信度分数                                                                                  |
| `suggestion` | string | 否   | 修复建议，可包含代码片段                                                                           |
| `rationale`  | string | 是   | 为什么判定为缺陷的原因说明。`CODESPEC` 类缺陷对外只写命中的具体规则名 / rule_id，不得透出内部 `skill_name` / 规则集合名 |
| `scope`      | string | 否   | 自定义工作流 / codespec 按整文件检测时填 `full_file`，令该缺陷跳过 diff 范围过滤（见 `references/custom-workflows.md`、`references/codespec-workflow.md`） |

> `CUSTOM`：自定义工作流检测主线产出的缺陷统一使用此枚举（见 `references/custom-workflows.md`）。
> `CODESPEC`：codespec 类型自定义工作流产出的缺陷使用此枚举，与 `CUSTOM` 同池（见 `references/codespec-workflow.md`）。

---

## 参考文件说明

| 文件                                | 用途                                                                                               | 何时读取                      |
| ----------------------------------- | -------------------------------------------------------------------------------------------------- | ----------------------------- |
| `references/review-dimensions.md`   | 评审维度：7 个维度定义及检测清单、误报规避规则                                                     | Step 3 执行评审时             |
| `references/review-rule.md`         | 缺陷分级（P0/P1/P2）与置信度评分策略（1-10）                                                       | Step 3 执行评审时             |
| `references/general-workflow.md`    | 通用检测工作流：评审策略（直接/分组并行）、文件分组规则、缺陷汇总/去重/过滤/排序流程、报告生成模板 | Step 3 执行评审时             |
| `references/custom-workflows.md`    | 自定义工作流检测主线：把 workflow `content` 当自然语言指令执行、范围对齐、缺陷输出规则；含 codespec 类型工作流分支 | Step 3 执行自定义工作流检测时（`custom_workflows.json` 非空时） |
| `references/codespec-workflow.md`   | codespec 类型工作流子流程：初始化资源目录 → 基于 codespec 规约（`spec_rule.md` 摘要 + `skills/<name>/` 规则原文）检测、范围对齐、缺陷输出规则 | 自定义工作流命中 codespec 类型时 |
| `assets/codespec-pe-template.md.j2` | codespec 检测指令的本地 Jinja2 模板源文件：注入规则摘要后生成检测指令；不替代 `skills/<name>/` 下的完整规则原文 | 由 `codespec_preflight.py` 自动加载 |
| `references/lang-go.md`             | Go 语言专项检测：变量遮蔽、nil map、channel 误用、类型断言、defer 陷阱等                           | 评审文件包含 `.go` 时         |
| `references/lang-typescript.md`     | TypeScript/前端专项检测：业务响应语义、UI 状态切换、异步闭包 stale state、DTO 字段映射、前端鉴权与 token 注入 | 评审文件包含 `.ts` / `.tsx` / `.js` / `.jsx` / `.mjs` / `.cjs` 时 |
| `references/codebase-api.md`        | Codebase OpenAPI CLI 使用说明：仓库/分支/MR 查询、MR 行级评论创建                                  | 需要调用 Codebase API 时      |
| `references/auth.md`                | 凭据获取：CLOUD_JWT / CODEBASE_JWT 的获取方式                                                      | 需要获取认证凭据时            |
| `references/version.txt`            | 发布/打包流程写入的 skill 版本标识，不参与评审运行时逻辑                                           | 发布或打包 skill 时           |
| `scripts/diff_and_filter.py`        | 执行 git diff 并过滤文件，并在同一次调用里完成自定义 workflow preflight；输出 `diff_files.md`、`review_files.md`、`custom_workflows.json`，命中 codespec 时继续准备 `codespec/` 产物 | Step 1 获取变更并完成准备时       |
| `scripts/fetch_custom_workflows.py` | 解析当前仓库标识并拉取仓库级自定义工作流，输出 `custom_workflows.json`                             | `diff_and_filter.py` 集成 preflight 内部调用，或手动排查时   |
| `scripts/codespec_preflight.py`     | preflight 编排入口：拉取自定义工作流、判定 codespec 类型、初始化 codespec 资源目录（`spec_rule.md` + skill 包 + `codespec_meta.json`）、渲染检测指令（`codespec_pe.md`），资源不可用时回写 `status: skipped` | `diff_and_filter.py` 集成调用；产物缺失需手动重跑时 |
| `scripts/validate_final_comments.py`| 校验 `final_comments.json` 必须存在且为合法 JSON 数组；缺失或结构错误时返回非 0，阻止误上报 success | Step 4 生成报告/上报前 |
| `scripts/generate_report.py`        | 读取 `final_comments.json` 生成 HTML 可视化报告，同时生成同名 `.md` 作为 IDE 兜底查看渠道          | Step 4 生成报告时             |
| `scripts/codebase.py`               | Codebase OpenAPI CLI：仓库/分支/MR 查询、从 MR/bits change URL 取 commit 范围、MR 行级评论创建     | 处理 MR/PR/bits change 场景（Step 1 取 base/source commit，或需要对 MR 留行级评论时） |
| `assets/report-template.html`       | HTML 报告静态模板（含占位符），由 `generate_report.py` 自动加载                                    | 由脚本自动加载                |

## 中间产物目录

所有中间文件保存在 `WORK_DIR` 下（见 Step 1 中的路径说明）：

| 文件                      | 产生步骤                | 说明                                      |
| ------------------------- | ----------------------- | ----------------------------------------- |
| `diff_files.md`           | Step 1                  | 原始 diff 文件列表                        |
| `custom_workflows.json`   | Step 1                  | 拉取到的仓库自定义工作流列表               |
| `custom/workflow_<idx>.md`| Step 1                  | 各自定义工作流的自包含规则文件（含 content 原文），供检测 subagent 逐字读取 |
| `codespec/codespec_meta.json` | codespec 类型工作流  | codespec 资源目录元数据（`rule_count` / `skills` / `pe_source` 等） |
| `codespec/spec_rule.md`   | codespec 类型工作流     | 拉取到的 codespec 规则清单                  |
| `codespec/skills/<name>/` | codespec 类型工作流     | 各 codespec 规则配套 skill 包（含规则原文；命中/候选规则必须继续读取） |
| `codespec/codespec_pe.md` | codespec 类型工作流     | codespec 检测指令（由本地模板渲染生成，仅包含规则摘要与执行要求，不替代规则原文） |
| `review_files.md`         | Step 1-2                | 过滤后的待评审文件列表（含 scope 元数据） |
| `review_groups.md`        | 通用工作流 Step 2       | 文件分组及依据（仅分组评审时）            |
| `group/group_<num>.jsonl` | 通用工作流 Step 3       | 各分组的缺陷列表（仅分组评审时）          |
| `custom/custom_<序号>.jsonl`| 自定义工作流 Step 3     | 各自定义工作流的缺陷列表（含 codespec 类型，`category=CODESPEC`）；0 缺陷时为空文件，作为该单元执行完成的证明 |
| `comments.jsonl`          | 通用工作流 Step 4       | 通用检测主线汇总的缺陷                     |
| `final_comments.json`     | 通用工作流              | 去重过滤后的最终缺陷列表                   |
| `report.html`             | Step 4                  | HTML 可视化评审报告                       |
| `report.md`               | Step 4                  | Markdown 评审报告（IDE 无法预览 HTML 时的兜底） |
