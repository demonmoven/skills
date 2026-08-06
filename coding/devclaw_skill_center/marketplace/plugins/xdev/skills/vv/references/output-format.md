# 输出格式规范（Output Format）

## 输出目录结构

```
$OUTPUT_DIR/vv/
  config.json                          # 编排配置
  subsystem_1_rule_checks/
    rule-precheck.md                   # 确定性检查结果
    rule-precheck-findings.json        # 确定性检查结构化数据
    vv-scope-boundary.md               # LLM 审查报告
    vv-mock-detection.md
    vv-cozeloop-standards.md
    vv-data-api-compliance.md
    vv-code-quality.md
    summary.json
  subsystem_2_structural_tests/
    vv-maintainability.md
    vv-performance-consistency.md
    summary.json
  subsystem_3_task_evals/
    vv-requirements-compliance.md
    vv-data-api-compliance.md
    vv-bug-detection.md
    summary.json
  subsystem_4_trace_grading/
    summary.json
  subsystem_5_regression_gates/
    vv-regression-detection.md
    gate-result.md
    summary.json
  failure-taxonomy.json                # 失败分类
  verdict.json                         # 最终裁决
  final-report.md                      # 人可读完整报告
```

## summary.json Schema

正式 schema 见 `summary.schema.json`。

每个子系统输出一个 `summary.json`，统一 schema：

```json
{
  "subsystem": "rule_checks",
  "score": 55,
  "pass": false,
  "findings_count": {
    "critical": 1,
    "high": 2,
    "medium": 0,
    "low": 3
  },
  "findings": [
    {
      "id": "RC-001",
      "severity": "critical",
      "category": "secrets_detected",
      "file": "config/db.go",
      "line": 42,
      "title": "硬编码数据库密码",
      "description": "在源码中发现明文数据库密码",
      "fix_hint": "使用环境变量替代",
      "failure_type": "POLICY_VIOLATION"
    }
  ]
}
```

### 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `subsystem` | string | 子系统标识：`rule_checks` / `structural_tests` / `task_evals` / `trace_grading` / `regression_gates` |
| `score` | int | 0-100 分 |
| `pass` | bool | 是否通过 |
| `findings_count` | object | 各严重级别的 finding 数量 |
| `findings` | array | finding 详细列表 |

### Finding 字段

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | 唯一标识，格式为 `{子系统缩写}-{序号}`，如 `RC-001`、`ST-003` |
| `severity` | string | `critical` / `high` / `medium` / `low` |
| `category` | string | 分类标签（自由文本） |
| `file` | string | 相关文件路径 |
| `line` | int | 行号（可选） |
| `title` | string | 简短标题 |
| `description` | string | 详细描述 |
| `fix_hint` | string | 修复建议 |
| `failure_type` | string | 失败类型，见 failure-taxonomy.md |

### Finding ID 前缀约定

| 子系统 | ID 前缀 |
|--------|---------|
| 规则检查 | `RC-` |
| 结构测试 | `ST-` |
| 任务评测 | `TE-` |
| 轨迹评分 | `TG-` |
| 回归门禁 | `RG-` |

---

## verdict.json Schema

正式 schema 见 `verdict.schema.json`。

```json
{
  "verdict": "FAIL",
  "reason": "存在 1 个 CRITICAL 级规则违规",
  "gate_level": "nightly",
  "subsystem_scores": {
    "rule_checks": { "score": 55, "pass": false },
    "structural_tests": { "score": 95, "pass": true },
    "task_evals": { "score": 90, "pass": true },
    "trace_grading": { "score": 78, "pass": false },
    "regression_gates": { "score": 100, "pass": true }
  },
  "failure_taxonomy": {
    "POLICY_VIOLATION": 1,
    "STRUCTURE_DRIFT": 0,
    "OUTCOME_FAIL": 0,
    "TOOL_MISUSE": 0,
    "THRASHING": 0,
    "UNNECESSARY_COMPLEXITY": 0
  },
  "blocking_findings": ["RC-001"]
}
```

### verdict 取值

| 值 | 含义 |
|----|------|
| `PASS` | 所有检查通过 |
| `FAIL` | 存在阻塞性问题 |
| `SOFT_FAIL` | 部分不通过但无阻塞性问题，建议修复 |
| `NEEDS_APPROVAL` | 需要人工审批（安全类问题） |

---

## final-report.md 模板

```markdown
# V&V 验证报告

## 基本信息

| 项目 | 值 |
|------|-----|
| 门禁级别 | {gate_level} |
| 目标路径 | {target_path} |
| 基准 SHA | {base_sha} |
| 执行时间 | {timestamp} |
| **最终裁决** | **{verdict}** |

## 子系统评分

| 子系统 | 分数 | 通过 | Critical | High | Medium | Low |
|--------|------|------|----------|------|--------|-----|
| 规则检查 | {score} | {pass} | {n} | {n} | {n} | {n} |
| 结构测试 | {score} | {pass} | {n} | {n} | {n} | {n} |
| 任务评测 | {score} | {pass} | {n} | {n} | {n} | {n} |
| 轨迹评分 | {score} | {pass} | {n} | {n} | {n} | {n} |
| 回归门禁 | {score} | {pass} | {n} | {n} | {n} | {n} |

## 失败分类

{按 failure_type 分组的 findings}

## 阻塞性发现

{blocking_findings 的详细描述}

## 详细发现

### 规则检查
{subsystem_1 的所有 findings}

### 结构测试
{subsystem_2 的所有 findings}

...

## 裁决说明

{verdict.reason}
```
