# bug-fixloop 用户使用手册

本文档面向 bug-fixloop skill 的使用者。无论你使用的是 Claude Code、Cursor、Windsurf 还是其他支持 Agent Skills 的 AI 工具，本手册都适用。

---

## 一句话理解 bug-fixloop

bug-fixloop 是一个**全自动的测试修复循环**：它从你的需求文档出发，自动生成测试 → 部署服务 → 执行测试 → 分析失败原因 → 修复代码，循环往复直到所有测试通过。

---

## 快速开始

### 最简调用（上下文感知模式）

```
/bug-fixloop
```

如果你刚用 speckit 完成了开发，对话上下文中已经包含了业务仓库、分支、spec 目录等信息。此时可以不传任何参数，bug-fixloop 会自动从上下文推断，并向你确认后执行。

```
/bug-fixloop PSM=stone.cozeloop.prompt
```

只提供部分参数也可以。bug-fixloop 会先从上下文推断缺失参数，推断不到的再交互式询问。

### 标准调用（一步到位）

```
/bug-fixloop PSM=stone.cozeloop.prompt BRANCH=main BUSINESS_REPO=/home/user/backend TEST_REPO=/home/user/api_test
```

提供全部 7 个必填参数，bug-fixloop 不会再提问，直接开始自动执行全部流程。这是**最推荐的方式**（尤其在全新对话中没有上下文时）。

---

## 参数完整说明

### 必填参数（7 个）

| 参数 | 说明 | 填什么 |
|------|------|--------|
| `PSM` | 你的 TCE 服务标识符 | 例如 `stone.cozeloop.prompt`，到 TCE 控制台查看 |
| `BRANCH` | 业务仓库的代码分支名 | 部署用这个分支，修复后也 push 到这个分支。**用于业务仓库和测试仓库**（测试仓库在 Stage 0 会基于默认分支创建此分支并推送） |
| `BUSINESS_REPO` | 业务代码仓库 | 本地绝对路径（如 `/home/user/backend`）或 Git 地址（如 `git@code.byted.org:stone/backend.git`） |
| `TEST_REPO` | 测试代码仓库 | 格式同上。存放 E2E 集成测试的仓库 |
| `COMMIT_RANGE` | **git diff 范围（bug-fixloop 核心参数）** | 指定要分析的 commit 范围。默认 `HEAD~1..HEAD`（最近 1 个 commit）。支持 `HEAD~3..HEAD`、`origin/main..HEAD` 等 git log 范围语法 |
| `IDL_REPO` | IDL 仓库（仅 PROFILE=bytedance-tce） | 包含已变更接口定义的 IDL 仓库。bug-fixloop 的 phase-1 不直接用 IDL,但 phase-2 的 AGW 同步需要（仅 bytedance-tce 时） |
| `IDL_BRANCH` | IDL 仓库的变更分支名（仅 PROFILE=bytedance-tce） | IDL 变更所在的分支 |

> **上下文感知**：bug-fixloop 通常在项目目录中直接启动。zero-arg 模式下,`BUSINESS_REPO` 默认为 `$PWD`,`BRANCH` 默认为 `git branch --show-current`,`COMMIT_RANGE` 默认为 `HEAD~1..HEAD`。用户只需要在 wizard 中确认即可。

> **仓库路径说明**：本地路径以 `/` 开头，bug-fixloop 直接使用；Git 地址以 `git@` 或 `https://` 开头，bug-fixloop 会自动 clone 到 OUTPUT_DIR/repos/ 下。IDL_REPO 同理。

### 可选参数（7 个）— 行为定制开关

以下参数均有默认值，**不传则走默认行为**。根据你的场景按需覆盖。

| 参数 | 默认值 | 说明 | 什么时候需要改 |
|------|--------|------|--------------|
| `MAX_ITERATIONS` | `10` | 最大迭代轮次 | 想快速试跑设 `3`；想充分修复设 `15` |
| `GENERATE_TESTS` | `true` | 是否自动生成测试用例 | 已有测试代码时设 `false` 跳过生成 |
| `SINGLE_TEST_RUN` | `false` | 只跑一次测试就停 | 只想看测试结果、不要自动修复时设 `true` |
| `TCE_LANE` | 自动生成 | 指定复用的泳道名 | 已有泳道时传入避免重复创建，如 `boe_wtj_0306` |
| `SKIP_DEPLOY` | `false` | 跳过部署阶段 | 服务已在泳道中运行时设 `true`（必须同时指定 `TCE_LANE`） |
| `BYTEDCLI_SITE` | `boe` | bytedcli 目标站点 | 非 BOE 环境时修改，通常不需要动 |
| `OUTPUT_DIR` | `.costudio` | 输出数据存放目录 | 想自定义输出位置时修改 |

---

## 场景化使用指南

下面按**从最常见到最特殊**的顺序列出典型场景。找到最接近你需求的场景，复制命令修改参数即可。

### 场景 1：全流程（最常用）

**你有**：需求文档 + 业务代码仓库 + 测试仓库 + IDL 仓库
**你想**：从零开始，自动生成测试并反复修复直到通过

```
/bug-fixloop PSM=stone.cozeloop.prompt BRANCH=feat/my-feature BUSINESS_REPO=/home/user/backend TEST_REPO=/home/user/api_test
```

**实际执行流程**：
```
Stage 0: 从 SPEC 文档生成 E2E 测试 + 单元测试
    ↓
Stage 1: 部署分支到 TCE 泳道（自动创建泳道）
    ↓
Stage 1.5: 更新 AGW IDL 配置到泳道（首轮自动执行）
    ↓
Stage 2: 执行测试 → 全部通过? → 结束
    ↓ 有失败
Stage 3: 分析失败根因（测试代码问题 or 业务代码问题）
    ↓
Stage 4: 自动修复代码 + push
    ↓
Stage 5: 迭代总结
    ↓
回到 Stage 1（最多 10 轮）
```

---

### 场景 2：已有测试，只做修复循环

**你有**：已经写好了测试用例
**你想**：跳过测试生成，直接跑 部署→测试→修复 循环

```
/bug-fixloop PSM=stone.cozeloop.prompt BRANCH=main BUSINESS_REPO=/home/user/backend TEST_REPO=/home/user/api_test GENERATE_TESTS=false
```

**与全流程的区别**：跳过 Stage 0（不生成测试用例），直接从 Stage 1 部署开始。

---

### 场景 3：只跑一次测试看结果（不修复）

**你有**：代码已部署或准备部署
**你想**：只看测试通过率，不要自动修复代码

```
/bug-fixloop PSM=stone.cozeloop.prompt BRANCH=main BUSINESS_REPO=/home/user/backend TEST_REPO=/home/user/api_test GENERATE_TESTS=false SINGLE_TEST_RUN=true
```

**实际执行流程**：
```
Stage 1: 部署
    ↓
Stage 1.5: 更新 AGW IDL 配置到泳道（首轮自动执行）
    ↓
Stage 2: 执行测试 → 输出结果 → 结束（无论通过或失败）
```

不会进入 Stage 3/4/5，不会修改任何代码。

---

### 场景 4：复用已有泳道

**你有**：之前 bug-fixloop 创建的泳道还在运行
**你想**：复用泳道，不重新创建

```
/bug-fixloop PSM=stone.cozeloop.prompt BRANCH=main BUSINESS_REPO=/home/user/backend TEST_REPO=/home/user/api_test GENERATE_TESTS=false TCE_LANE=boe_costudio_a1b2
```

泳道名可以在上一次 bug-fixloop 的输出摘要中找到。

---

### 场景 5：跳过部署，只跑测试+修复

**你有**：服务已在泳道中运行，代码没变化不需要重新部署
**你想**：直接执行测试和修复

```
/bug-fixloop PSM=stone.cozeloop.prompt BRANCH=main BUSINESS_REPO=/home/user/backend TEST_REPO=/home/user/api_test GENERATE_TESTS=false TCE_LANE=boe_costudio_a1b2 SKIP_DEPLOY=true
```

> **注意**：`SKIP_DEPLOY=true` 必须配合 `TCE_LANE` 一起使用。没有指定泳道就跳过部署没有意义。
>
> `SKIP_DEPLOY` 仅影响首轮部署。如果进入修复循环，后续迭代会自动重新部署修复后的代码（使用 upgrade 动作），确保测试验证的是最新代码。

---

### 场景 6：快速验证（少量迭代）

**你想**：快速试跑 3 轮看看效果

```
/bug-fixloop PSM=stone.cozeloop.prompt BRANCH=main BUSINESS_REPO=/home/user/backend TEST_REPO=/home/user/api_test MAX_ITERATIONS=3
```

---

### 场景 7：使用远程 Git 仓库

**你没有**本地 clone 的仓库
**你想**：让 bug-fixloop 自动 clone

```
/bug-fixloop PSM=stone.cozeloop.prompt BRANCH=feat/new-api BUSINESS_REPO=git@code.byted.org:stone/backend.git TEST_REPO=git@code.byted.org:stone/api_test.git
```

bug-fixloop 会自动 clone 到 `.costudio/repos/backend/` 和 `.costudio/repos/api_test/`。

---

### 场景 8：仅生成测试用例（不部署不执行）

**你想**：只生成测试代码，手动 review 后再决定是否跑

```
/bug-fixloop PSM=stone.cozeloop.prompt BRANCH=main BUSINESS_REPO=/home/user/backend TEST_REPO=/home/user/api_test MAX_ITERATIONS=0
```

`MAX_ITERATIONS=0` 使迭代循环不执行（循环条件 `ITERATION(1) <= 0` 为假，直接跳过）。Stage 0 生成测试后流程输出最终摘要并结束，测试代码在 `$OUTPUT_DIR/craft/` 中。

---

## 参数组合速查表

| 我想... | 关键参数设置 |
|---------|------------|
| 全自动端到端 | （全部默认） |
| 跳过测试生成 | `GENERATE_TESTS=false` |
| 只看测试结果不修复 | `SINGLE_TEST_RUN=true` |
| 只生成测试不执行 | `MAX_ITERATIONS=0` |
| 复用已有泳道 | `TCE_LANE=<泳道名>` |
| 跳过部署 | `SKIP_DEPLOY=true TCE_LANE=<泳道名>` |
| 限制迭代次数 | `MAX_ITERATIONS=<N>` |
| 自定义输出目录 | `OUTPUT_DIR=<路径>` |

**组合使用**——参数可自由叠加，例如：
```
# 跳过测试生成 + 复用泳道 + 最多 5 轮
/bug-fixloop ... GENERATE_TESTS=false TCE_LANE=boe_xxx MAX_ITERATIONS=5
```

---

## 产出物说明

bug-fixloop 执行完毕后，所有数据存放在 `OUTPUT_DIR`（默认 `.costudio/`）下：

```
.costudio/
├── craft/                              # Stage 0 测试生成产出
│   ├── user_journeys.md                # 用户动线文档（从 SPEC 提取的测试场景）
│   ├── stage1_to_stage2_coverage.md    # 覆盖度分析：动线→E2E
│   ├── e2e_work_copy/                  # 生成的 E2E 集成测试代码
│   ├── generated_test_cases.jsonl      # E2E 测试清单
│   ├── test_dirs.txt                   # 测试目录列表（供后续迭代使用）
│   ├── stage2_to_stage3_coverage.md    # 覆盖度分析：E2E→单元测试
│   ├── unit_tests/                     # 生成的单元测试代码
│   ├── generated_unit_test_cases.jsonl # 单元测试清单
│   └── unit_test_dirs.txt              # 单元测试目录列表（供后续迭代使用）
│
├── iteration_1/                        # 第 1 轮迭代数据
│   ├── test_stats.json                 # 测试统计 {"passed_cases":7,"failed_cases":2,...}
│   ├── failed_cases.jsonl              # 失败用例详情（每行一个 JSON）
│   ├── analysis_report.md              # Judge 根因分析报告
│   ├── fix_summary.md                  # 修复摘要
│   └── iteration_summary.md            # 迭代经验总结
│
├── iteration_2/                        # 第 2 轮...
│   └── ...
└── iteration_N/
    └── ...
```

### 关键文件解读

| 文件 | 什么时候看 | 内容 |
|------|----------|------|
| `test_stats.json` | 想快速了解某轮通过率 | `{"total_tests":10, "passed_cases":7, "failed_cases":2, "skipped_cases":1}` |
| `failed_cases.jsonl` | 想看具体哪些测试挂了 | 每行一个 JSON，含测试名、错误信息、LogID |
| `analysis_report.md` | 想理解失败根因 | Judge 对每个失败用例的分类（业务代码问题 / 测试代码问题）和修复建议 |
| `fix_summary.md` | 想看某轮修了什么 | 修改的文件、修改原因、根因类别 |
| `iteration_summary.md` | 想看迭代趋势 | 通过率趋势、已解决/未解决问题、反复出现的模式 |

---

## 前置条件

运行 bug-fixloop 前请确认：

1. **bytedcli 已安装且可用**
   - bug-fixloop 会自动发现已安装的 bytedcli skills（兼容所有 agent）
   - 未登录时 bug-fixloop 会尝试自动登录

2. **bytedcli skills 已安装**
   bug-fixloop 依赖以下 bytedcli skills，请提前通过 `ai-skills add` 安装：
   - `bytedance-auth` — 认证管理
   - `bytedance-tce` — TCE 部署和实例管理
   - `bytedance-agw` — AGW 网关 IDL 更新
   - `bytedance-log` — 日志查询
   - `bytedance-tools` — 通用调用方式参考

3. **仓库可访问**（业务仓库、测试仓库、IDL 仓库）
   - 本地路径：目录存在且包含 `.git`
   - Git 地址：当前环境有 clone 权限

4. **SPEC 文档目录非空**
   - 目录下应包含 API 规格文档、PRD、接口契约等
   - 格式不限（Markdown、JSON、YAML、OpenAPI 均可）

5. **内网环境**
   - bytedcli 需要内网/VPN 访问 TCE API

---

## 执行流程详解

```
┌─────────────────────────────────────────────────────────────┐
│ 参数解析：从命令行解析 KEY=VALUE                            │
│ 上下文推断：从对话上下文补全缺失参数 → 向用户确认          │
│ 交互补充：仍缺失的参数交互式询问                            │
│ 前置检查：创建输出目录，清理上次残留，认证 bytedcli（npx）  │
└─────────────┬───────────────────────────────────────────────┘
              │
              ▼ [GENERATE_TESTS=true]
┌─────────────────────────────────────────────────────────────┐
│ Stage 0: 测试生成                                           │
│  0.1 从 SPEC 提取用户动线                                    │
│  0.2 覆盖度分析（动线 → E2E）                                │
│  0.3 生成 E2E 集成测试代码                                   │
│  0.4 同步 E2E 到测试仓库 + 生成测试目录清单                  │
│  0.5 覆盖度分析（E2E → 单元测试）                            │
│  0.6 生成单元测试代码                                        │
│  0.7 同步单元测试到业务仓库 + push                           │
└─────────────┬───────────────────────────────────────────────┘
              │ [GENERATE_TESTS=false 时直接到这里]
              ▼
         ITERATION = 1
              │
              ▼
┌─────── 迭代循环 (WHILE ITERATION <= MAX_ITERATIONS) ────────┐
│                                                              │
│  Stage 1: 部署 [SKIP_DEPLOY=true 时跳过]                    │
│    → 部署 BRANCH 到 TCE 泳道（首轮创建，后续复用）          │
│    → 部署失败 → 自动修复编译错误 → 重新部署                 │
│                                                              │
│  Stage 1.5: AGW IDL 同步 [首轮自动执行]                     │
│    → 查找 PSM 对应的 AGW service-id                         │
│    → 根据变更类型选择 AGW 命令，同步 IDL（及路由）到泳道    │
│    → 失败不阻断，记录警告继续测试                           │
│                                                              │
│  Stage 2: 集成测试                                           │
│    → 执行 go test，解析结果                                  │
│    → 全部通过 → 退出循环                                    │
│    → [SINGLE_TEST_RUN=true] → 输出结果，退出循环            │
│                                                              │
│  Stage 3: 失败分析                                           │
│    → Judge 角色分析每个失败用例的根因                        │
│    → 分类：业务代码问题 / 测试代码问题                      │
│    → 查询 LogID 获取服务端日志辅助诊断                      │
│                                                              │
│  Stage 4: 代码修复                                           │
│    → 按优先级修复前 5 个问题                                │
│    → 迭代越深修复策略越激进（标准→策略切换→深层追踪）       │
│    → commit + push                                           │
│                                                              │
│  Stage 5: 迭代总结                                           │
│    → 合成经验总结，供下一轮修复参考                         │
│    → ITERATION++                                             │
│                                                              │
└──────────────────────────────────────────────────────────────┘
              │
              ▼
┌─────────────────────────────────────────────────────────────┐
│ 最终摘要：总迭代次数、最终通过率、泳道名、各轮统计          │
└─────────────────────────────────────────────────────────────┘
```

---

## 常见问题

### Q: bug-fixloop 中途失败了，怎么从断点继续？

不支持断点续跑。但你可以利用参数跳过已完成的阶段：

```
# 测试已生成，泳道已存在 → 跳过生成和部署
/bug-fixloop ... GENERATE_TESTS=false TCE_LANE=boe_xxx SKIP_DEPLOY=true
```

### Q: 修复循环一直不收敛（通过率不提升）怎么办？

1. 查看最后一轮的 `iteration_summary.md`，了解反复失败的模式
2. 设小 `MAX_ITERATIONS=3` 快速跑几轮，看趋势
3. 手动检查 `analysis_report.md` 中标为 UNCERTAIN 的问题，可能需要人工介入

### Q: 只想重跑测试（不重新生成、不重新部署）？

```
/bug-fixloop ... GENERATE_TESTS=false TCE_LANE=boe_xxx SKIP_DEPLOY=true SINGLE_TEST_RUN=true
```

### Q: 仓库用本地路径还是 Git 地址？

- **本地路径**（推荐）：速度快，修复后代码直接在本地可见。路径必须是**绝对路径**，以 `/` 开头。
- **Git 地址**：bug-fixloop 自动 clone。适合 CI/CD 或远程环境没有预先 clone 的场景。
- 以上规则同样适用于 `IDL_REPO`。

### Q: COMMIT_RANGE 怎么写？

bug-fixloop 用 COMMIT_RANGE 指定要分析的 git commit 范围，格式与 `git log` 一致：

| 写法 | 含义 | 适用场景 |
|------|------|---------|
| `HEAD~1..HEAD` | 最近 1 个 commit（默认） | 刚改完一个 commit 想加测试 |
| `HEAD~3..HEAD` | 最近 3 个 commit | 合并前对连续几个 commit 补测试 |
| `origin/main..HEAD` | 当前分支与 main 的全部差异 | 整个 feature 分支的 diff |
| `<sha>..HEAD` | 从某个 commit 到当前 | 针对某个问题起点 |

**注意**：bug-fixloop 不支持 staged / unstaged 改动（必须已 commit）。如果你有未提交改动,先 `git commit -m "wip"` 再跑 bug-fixloop。

### Q: OUTPUT_DIR 里的旧数据会怎样？

每次启动 bug-fixloop 会**清理** OUTPUT_DIR 下的 `craft/` 和 `iteration_*/` 子目录。`repos/`（clone 的仓库）和其他文件不受影响。

---

## 参数决策流程图

不确定该传哪些参数？按这个流程走：

```
你有现成的测试代码吗？
├── 有 → GENERATE_TESTS=false
│   │
│   ├── 服务已在泳道中运行？
│   │   ├── 是 → SKIP_DEPLOY=true TCE_LANE=<泳道名>
│   │   └── 否 → （默认部署）
│   │
│   ├── 只想看测试结果？
│   │   ├── 是 → SINGLE_TEST_RUN=true
│   │   └── 否 → （默认进入修复循环）
│   │
│   └── 想限制迭代轮数？
│       ├── 是 → MAX_ITERATIONS=<N>
│       └── 否 → （默认 10 轮）
│
└── 没有 → GENERATE_TESTS=true（默认）
    │
    └── 只想生成测试，不跑？
        ├── 是 → MAX_ITERATIONS=0
        └── 否 → （全流程执行）
```
