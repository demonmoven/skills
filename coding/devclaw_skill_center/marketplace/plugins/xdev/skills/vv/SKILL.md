---
name: vv
description: V&V 验证与验收系统。对代码变更执行结构化验证，支持门禁检查和 E2E 迭代审查两种模式。使用 "vv gate" 运行门禁流程，"vv review" 运行 E2E 审查流程。
argument-hint: "<workflow> <target_path> [--gate-level pr|nightly|release] [--subsystems all|rule,structural,task,trace,regression] [--spec-dir path] [--trace-file path] [--base-sha SHA] [--output-dir path]"
subskills:
  - workflows/gate.md
  - workflows/review.md
  - workflows/verify.md
  - subsystems/rule-checker.md
  - subsystems/structural-tester.md
  - subsystems/task-evaluator.md
  - subsystems/trace-grader.md
  - subsystems/regression-gater.md
  - checks/rule-precheck.md
  - checks/scope-boundary.md
  - checks/mock-detection.md
  - checks/cozeloop-standards.md
  - checks/data-api-compliance.md
  - checks/code-quality.md
  - checks/maintainability.md
  - checks/performance-consistency.md
  - checks/requirements-compliance.md
  - checks/bug-detection.md
  - checks/trace-grading.md
  - checks/regression-detection.md
  - checks/pr-gate.md
  - checks/failure-classifier.md
---

# V&V 验证与验收系统（统一入口）

你是 V&V 验证与验收系统的路由器，负责解析参数并分发到对应的工作流。

## 参数解析

从 `$ARGUMENTS` 中解析以下参数：

| 参数 | 必填 | 说明 | 默认值 |
|------|------|------|--------|
| `workflow` | 是（位置参数） | `gate` 或 `review` | — |
| `target_path` | 是（位置参数） | 待验证代码目录 | — |
| `--gate-level` | 否 | `pr` / `nightly` / `release`（仅 gate 模式） | `pr` |
| `--subsystems` | 否 | 逗号分隔: `rule,structural,task,trace,regression` 或 `all` | `all` |
| `--spec-dir` | 否 | SPEC 文档目录 | — |
| `--trace-file` | 否 | Agent trace 文件路径 | — |
| `--base-sha` | 否 | 基准 commit | `HEAD~1` |
| `--output-dir` | 否 | 输出目录 | `.vv-output` |

## 路由逻辑

解析 `$ARGUMENTS` 中的第一个位置参数作为 `workflow`：

### 1. `gate` — V&V 门禁流程

读取 `workflows/gate.md` 并按其中的指令执行完整的门禁检查流程。

该工作流将：
- 根据 `--gate-level` 确定执行范围
- 按顺序调度 5 个子系统执行检查
- 根据 `--subsystems` 参数决定跳过哪些子系统（默认 `all` 时按 `GATE_LEVEL` 决定）
- 汇总生成 verdict 和最终报告

### 2. `review` — E2E 迭代审查流程

读取 `workflows/review.md` 并按其中的指令执行 E2E 迭代审查流程。

该工作流将：
- 汇总 9 个独立 Review Agent 的审查报告，生成问题清单
- 验证结论：回到源码逐条验证发现的真实性，去重、过滤误报
- 执行综合代码审查
- 生成最终 Pass/Fail 结论

### 参数传递

将解析后的参数作为上下文传递给对应的 workflow 文件：
- `TARGET_PATH` → target_path
- `GATE_LEVEL` → --gate-level
- `SUBSYSTEMS` → --subsystems
- `SPEC_DIR` → --spec-dir
- `TRACE_FILE` → --trace-file
- `BASE_SHA` → --base-sha
- `OUTPUT_DIR` → --output-dir

### 模板变量映射

gate 工作流和 review 工作流使用不同的模板变量名。以下是统一映射关系：

| 入口参数 | gate 工作流变量 | review/check 工作流变量 | 说明 |
|----------|----------------|------------------------|------|
| `target_path` | `TARGET_PATH` | `{{backendPath}}` | 待验证代码目录 |
| `--gate-level` | `GATE_LEVEL` | — | 门禁级别（仅 gate） |
| `--subsystems` | `SUBSYSTEMS` | — | 子系统选择（仅 gate） |
| `--output-dir` | `OUTPUT_DIR` | `{{reviewOutputDir}}` | 输出目录（review 模式下为各 agent 报告所在目录） |
| `--spec-dir` | `SPEC_DIR` | `{{specDir}}` | SPEC 文档目录（gate 任务评测 + review 阶段二综合审查使用） |
| `--trace-file` | `TRACE_FILE` | — | Agent trace 文件（仅 gate 轨迹评分） |
| `--base-sha` | `BASE_SHA` | `BASE_SHA` | 基准 commit |

**注意**：review 工作流中的 `{{e2eTestPath}}`、`{{iteration}}`、`{{maxIterations}}`、`{{isMultiReview}}`、`{{allRoundsDir}}`、`{{previousRoundDir}}`、`{{reviewRound}}`、`{{totalReviewRounds}}` 等变量由 E2E 迭代框架注入，不通过 vv 入口参数传递。

## 错误处理

- 如果 `workflow` 参数缺失或不是 `gate`/`review`，提示用户正确用法
- 如果 `target_path` 缺失，提示用户提供待验证代码目录
- 如果 `--gate-level` 不是 `pr`/`nightly`/`release`，提示用户有效值为 `pr`、`nightly`、`release`
- 如果 `--subsystems` 包含无效子系统名，提示用户有效值为 `rule`、`structural`、`task`、`trace`、`regression` 或 `all`
