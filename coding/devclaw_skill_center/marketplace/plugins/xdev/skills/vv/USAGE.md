# VV Skill 使用手册

> 本手册为 AI Agent 和人类用户共同编写。所有调用示例均可直接复制使用。

## 概述

`vv` 是 V&V（Verification & Validation）验证与验收系统的**唯一入口**。它通过第一个位置参数 `workflow` 选择运行模式，通过可选参数精确控制执行范围。

**两种工作流**：

| 工作流 | 命令 | 用途 | 典型场景 |
|--------|------|------|----------|
| `gate` | `vv gate <path>` | 门禁检查：调度子系统执行结构化验证，输出 verdict | PR 提交前检查、Nightly 构建、Release 发布 |
| `review` | `vv review <path>` | E2E 审查：汇总 9 个 Review Agent 报告 + 综合 CR | 迭代修复后的审查汇总 |

---

## 快速上手

### 最简调用

```
vv gate ./src
```

等价于：`vv gate ./src --gate-level pr --subsystems all --base-sha HEAD~1 --output-dir .vv-output`

### 常用场景

```
# PR 门禁（最快，2 个子系统，约 5 分钟）
vv gate ./backend

# Nightly 全量检查（5 个子系统，含需求合规和轨迹评分）
vv gate ./backend --gate-level nightly --spec-dir ./specs --trace-file ./trace.jsonl

# Release 最严格门禁
vv gate ./backend --gate-level release --spec-dir ./specs --trace-file ./trace.jsonl

# 仅执行规则检查子系统
vv gate ./backend --subsystems rule

# 仅执行规则检查 + 回归门禁（跳过结构测试、任务评测、轨迹评分）
vv gate ./backend --subsystems rule,regression

# E2E 迭代审查
vv review ./backend
```

---

## 参数详解

### 位置参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `workflow` | 是 | 工作流类型：`gate` 或 `review` |
| `target_path` | 是 | 待验证代码目录的路径 |

### 可选参数

| 参数 | 适用工作流 | 说明 | 默认值 | 可选值 |
|------|-----------|------|--------|--------|
| `--gate-level` | gate | 门禁级别，决定执行范围和通过阈值 | `pr` | `pr` / `nightly` / `release` |
| `--subsystems` | gate | 指定执行哪些子系统（覆盖 gate-level 的默认范围） | `all` | `all` / 逗号分隔列表 |
| `--spec-dir` | gate, review | SPEC 规格文档目录（gate: 任务评测子系统; review: 阶段二综合审查） | — | 目录路径 |
| `--trace-file` | gate | Agent 执行轨迹文件（轨迹评分子系统需要） | — | `.jsonl` 文件路径 |
| `--base-sha` | gate, review | 基准 commit，用于计算 diff | `HEAD~1` | Git SHA |
| `--output-dir` | gate, review | 输出目录，所有报告写入此目录下 | `.vv-output` | 目录路径 |

---

## `gate` 工作流详解

### 架构

```
vv gate
  │
  ├── Step 0: 读取门禁策略，确定执行范围
  │
  ├── Step 1: 规则检查子系统 (rule)
  │   ├── checks/rule-precheck     ← go vet / go build / secrets scan / 禁用依赖 / 禁止目录
  │   ├── checks/scope-boundary    ← 范围边界审查
  │   ├── checks/mock-detection    ← Mock/取巧代码检测
  │   ├── checks/cozeloop-standards ← Go 规范 + IDL/REST API 规范 (nightly+)
  │   ├── checks/data-api-compliance ← 数据模型与 API 合约 (nightly+)
  │   └── checks/code-quality      ← 冗余代码与 Go 惯用模式 (nightly+)
  │
  ├── Step 2: 结构测试子系统 (structural) [nightly+]
  │   ├── checks/maintainability   ← 可维护性与可测试性
  │   └── checks/performance-consistency ← 性能与分布式一致性
  │
  ├── Step 3: 任务评测子系统 (task) [nightly+, 需 --spec-dir]
  │   ├── checks/requirements-compliance ← 需求合规性
  │   ├── checks/data-api-compliance     ← 数据模型与 API
  │   └── checks/bug-detection           ← Bug 检测与 Go 陷阱
  │
  ├── Step 4: 轨迹评分子系统 (trace) [nightly+, 需 --trace-file]
  │   └── checks/trace-grading    ← Agent 执行轨迹质量评估
  │
  ├── Step 5: 回归门禁子系统 (regression)
  │   ├── checks/regression-detection ← 回归检测
  │   └── checks/pr-gate             ← PR 门禁评估
  │
  ├── Step 6: 失败分类
  │   └── checks/failure-classifier ← 将 findings 分为 6 类
  │
  └── Step 7: 报告生成
      ├── verdict.json              ← 机器可读裁决
      └── final-report.md           ← 人可读完整报告
```

### `--gate-level` 参数行为

| 门禁级别 | 执行的子系统 | 通过阈值 | 超时 |
|----------|-------------|---------|------|
| `pr` | rule + regression | 每个子系统 score >= 70, 零 CRITICAL | 5 分钟 |
| `nightly` | rule + structural + task + trace + regression | 每个子系统 score >= 80, 零 CRITICAL/HIGH | 30 分钟 |
| `release` | 同 nightly + 额外检查 | 每个子系统 score >= 90, 零 CRITICAL/HIGH/MEDIUM | 60 分钟 |

**注意**：
- `task` 子系统仅在 `--spec-dir` 提供时执行
- `trace` 子系统仅在 `--trace-file` 提供时执行
- 即使 gate-level 为 nightly/release，缺少上述参数的子系统将被静默跳过

### `--subsystems` 参数行为

当 `--subsystems` 不为 `all` 时，**覆盖** gate-level 的默认子系统范围。

**可选值**（逗号分隔，不含空格）：

| 值 | 对应子系统 | 产出目录 |
|----|-----------|---------|
| `rule` | 规则检查 | `subsystem_1_rule_checks/` |
| `structural` | 结构测试 | `subsystem_2_structural_tests/` |
| `task` | 任务评测 | `subsystem_3_task_evals/` |
| `trace` | 轨迹评分 | `subsystem_4_trace_grading/` |
| `regression` | 回归门禁 | `subsystem_5_regression_gates/` |

**示例**：

```
# 仅规则检查（最快，适合快速预检）
vv gate ./src --subsystems rule

# 规则检查 + 结构测试（跳过任务评测、轨迹评分、回归门禁）
vv gate ./src --subsystems rule,structural

# 仅任务评测（需要 spec）
vv gate ./src --subsystems task --spec-dir ./specs

# 全部子系统（等价于 --subsystems all）
vv gate ./src --gate-level nightly
```

**组合规则**：
- `--subsystems all`（默认值）→ 按 `--gate-level` 决定执行哪些子系统。例如 `--subsystems all --gate-level pr` 仅执行 rule + regression
- `--subsystems rule,structural` → 显式指定子系统，**覆盖** gate-level 的默认范围。例如 `--subsystems rule,structural --gate-level pr` 强制执行 rule + structural，即使 PR 级默认不含 structural
- 显式指定子系统时，阈值仍由 `--gate-level` 决定（如 PR 级阈值 70 分）

### 裁决结果（verdict）

| 裁决 | 含义 | 触发条件 |
|------|------|----------|
| `PASS` | 所有检查通过 | 所有子系统 pass 且满足阈值 |
| `FAIL` | 存在阻塞性问题 | 存在 CRITICAL finding，或部分子系统不 pass 且分数远低于阈值（差距 > 10） |
| `SOFT_FAIL` | 部分不通过但无阻塞性问题 | 部分子系统不 pass 但无 CRITICAL，且分数接近阈值（差距 <= 10） |
| `NEEDS_APPROVAL` | 需要人工审批 | 涉及安全类 finding |

### 评分规则

**门禁模式评分**（rule-precheck、pr-gate、各子系统 summary）：

| Finding 级别 | 扣分 |
|-------------|------|
| CRITICAL | -25 分/个 |
| HIGH | -10 分/个 |
| MEDIUM | -5 分/个 |
| LOW | 不扣分 |

基础分 100，最低 0 分。

### 输出目录结构

```
{OUTPUT_DIR}/vv/
  config.json                           ← 运行配置
  subsystem_1_rule_checks/
    rule-precheck.md                    ← 确定性检查报告
    rule-precheck-findings.json         ← 确定性检查结构化数据
    vv-scope-boundary.md                ← LLM 审查报告
    vv-mock-detection.md
    vv-cozeloop-standards.md            ← (nightly+)
    vv-data-api-compliance.md           ← (nightly+)
    vv-code-quality.md                  ← (nightly+)
    summary.json                        ← 子系统汇总
  subsystem_2_structural_tests/         ← (nightly+)
    vv-maintainability.md
    vv-performance-consistency.md
    summary.json
  subsystem_3_task_evals/               ← (nightly+, 需 --spec-dir)
    vv-requirements-compliance.md
    vv-data-api-compliance.md
    vv-bug-detection.md
    summary.json
  subsystem_4_trace_grading/            ← (nightly+, 需 --trace-file)
    summary.json
  subsystem_5_regression_gates/
    vv-regression-detection.md
    gate-result.md
    summary.json
  failure-taxonomy.json                 ← 失败分类（6 大类型）
  verdict.json                          ← 最终裁决
  final-report.md                       ← 人可读完整报告
```

---

## `review` 工作流详解

### 架构

```
vv review
  │
  ├── 阶段一：审查结论汇总
  │   ├── 读取 9 个 Review Agent 报告
  │   ├── 提取所有问题（严重/中等/轻微/提示）
  │   ├── 生成完整版 review-conclusion.md
  │   └── 生成精简版 review-conclusion-for-agent.md（不含提示级别）
  │
  └── 阶段二：综合代码审查
      ├── Go 代码规范审查
      ├── CozeLoop IDL/REST API 审查
      ├── 通用代码质量检查
      └── 输出 overall.md
```

### 输入要求

review 工作流期望以下报告已存在于 `{{reviewOutputDir}}/` 下：

| 文件名 | 来源 |
|--------|------|
| `01-requirements-compliance.md` | 需求合规性 Review Agent |
| `02-data-api-compliance.md` | 数据模型与 API Review Agent |
| `03-scope-boundary.md` | 范围边界 Review Agent |
| `04-code-quality.md` | 代码质量 Review Agent |
| `05-maintainability.md` | 可维护性 Review Agent |
| `06-bug-detection.md` | Bug 检测 Review Agent |
| `07-regression-detection.md` | 回归检测 Review Agent |
| `08-cozeloop-standards.md` | CozeLoop 规范 Review Agent |
| `09-mock-detection.md` | Mock 检测 Review Agent |

### 评分规则

**LLM 审查模式评分**（review 工作流和各 LLM 审查 check）：

| 问题级别 | 扣分 |
|----------|------|
| 严重 (CRITICAL) | -25 分/个 |
| 中等 (HIGH) | -10 分/个 |
| 轻微 (MEDIUM) | -3 分/个 |
| 提示 (LOW) | 不扣分 |

**Pass/Fail 判定**：存在任何严重、中等或轻微问题 → Fail。仅有提示级别问题 → Pass。

---

## 失败分类体系

gate 工作流在 Step 6 将所有 findings 归类为 6 大失败类型：

| 类型 | 说明 | 主要来源 |
|------|------|----------|
| `POLICY_VIOLATION` | 违反编码策略或规范 | rule 子系统 |
| `STRUCTURE_DRIFT` | 代码结构偏离项目约定 | structural 子系统 |
| `OUTCOME_FAIL` | 产出未达成任务目标 | task / regression 子系统 |
| `TOOL_MISUSE` | Agent 工具使用错误 | trace 子系统 |
| `THRASHING` | 无效重复或来回修改 | trace 子系统 |
| `UNNECESSARY_COMPLEXITY` | 不必要的复杂性 | rule / task 子系统 |

---

## 子系统与检查项速查

### 规则检查子系统 (rule)

| 检查项 | 文件 | 类型 | PR 级 | Nightly+ |
|--------|------|------|-------|----------|
| go vet + go build | `checks/rule-precheck.md` | 确定性 | ✅ | ✅ |
| Secrets Scan | `checks/rule-precheck.md` | 确定性 | ✅ | ✅ |
| 禁用依赖 + 禁止目录 | `checks/rule-precheck.md` | 确定性 | ✅ | ✅ |
| 范围边界审查 | `checks/scope-boundary.md` | LLM | ✅ | ✅ |
| Mock/取巧检测 | `checks/mock-detection.md` | LLM | ✅ | ✅ |
| CozeLoop 规范 | `checks/cozeloop-standards.md` | LLM | — | ✅ |
| 数据/API 合规 | `checks/data-api-compliance.md` | LLM | — | ✅ |
| 代码质量 | `checks/code-quality.md` | LLM | — | ✅ |

### 结构测试子系统 (structural)

| 检查项 | 文件 | 类型 |
|--------|------|------|
| 可维护性与可测试性 | `checks/maintainability.md` | LLM |
| 性能与分布式一致性 | `checks/performance-consistency.md` | LLM |

### 任务评测子系统 (task)

| 检查项 | 文件 | 类型 | 前提 |
|--------|------|------|------|
| 需求合规性 | `checks/requirements-compliance.md` | LLM | --spec-dir |
| 数据模型与 API | `checks/data-api-compliance.md` | LLM | --spec-dir |
| Bug 检测 | `checks/bug-detection.md` | LLM | — |

### 轨迹评分子系统 (trace)

| 检查项 | 文件 | 类型 | 前提 |
|--------|------|------|------|
| Agent 轨迹评分 | `checks/trace-grading.md` | LLM | --trace-file |

### 回归门禁子系统 (regression)

| 检查项 | 文件 | 类型 |
|--------|------|------|
| 回归检测 | `checks/regression-detection.md` | LLM |
| PR 门禁评估 | `checks/pr-gate.md` | 汇总 |

---

## 高级用法

### 自定义阈值

在 `--output-dir` 指定的目录下预先创建 `vv/config.json`，添加 `threshold_overrides` 字段：

```json
{
  "threshold_overrides": {
    "rule_checks": { "min_score": 60 },
    "structural_tests": { "min_score": 75 }
  }
}
```

gate 工作流会在前置检查中写入 config.json，但如果文件已存在，手动设置的 overrides 将被保留。

### 自定义禁用依赖和禁止目录

在 config.json 中添加：

```json
{
  "deny_deps": [
    "github.com/astaxie/beego",
    "github.com/your-org/deprecated-pkg"
  ],
  "forbidden_dirs": [
    "vendor/",
    "kitex_gen/",
    "your-generated-dir/"
  ]
}
```

### 飞书通知集成

当以下环境变量存在时，gate 工作流会在每个 Step 开始/完成时发送飞书 task_card：

| 环境变量 | 说明 |
|----------|------|
| `CHAT_ID` | 飞书群聊 ID |
| `FEISHU_TASK_CARD_SCRIPT` | task_card 脚本路径 |
| `FEISHU_ROLE` | 发送者角色标识 |

---

## 常见问题

### Q: gate 和 review 有什么区别？

**gate** 是自动化门禁流程，调度 5 个子系统按顺序执行检查，产出结构化的 verdict.json 和 final-report.md。适用于 CI/CD 集成。

**review** 是 E2E 迭代审查流程，汇总 9 个独立 Review Agent 的报告并生成综合结论。适用于迭代修复后的审查汇总。

两个工作流独立运行，互不干涉。

### Q: 为什么 nightly 级别没有执行任务评测？

任务评测子系统需要 `--spec-dir` 参数提供规格文档目录。如果未提供，该子系统将被静默跳过。请确保传入了 `--spec-dir` 参数。

### Q: 如何只运行特定的检查？

使用 `--subsystems` 参数。例如只想检查代码质量和可维护性：

```
vv gate ./src --subsystems rule,structural --gate-level nightly
```

注意：单个 check 不能独立选择，只能按子系统粒度控制。

### Q: summary.json 校验失败怎么办？

每个子系统执行完后会自动调用 `scripts/validate-summary-json.sh` 校验产出。如果失败：
- 该子系统 score 设为 0，pass 设为 false
- 流程**不会中断**，继续执行后续子系统
- 在 final-report.md 中会标注该子系统失败

### Q: Finding ID 有什么命名规则？

| 子系统 | ID 前缀 | 示例 |
|--------|---------|------|
| 规则检查 | `RC-` | `RC-001` |
| 结构测试 | `ST-` | `ST-003` |
| 任务评测 | `TE-` | `TE-002` |
| 轨迹评分 | `TG-` | `TG-001` |
| 回归门禁 | `RG-` | `RG-004` |
