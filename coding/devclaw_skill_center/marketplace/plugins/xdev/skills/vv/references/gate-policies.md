# 门禁策略（Gate Policies）

`gate-policies.json` 是唯一的机器可读配置源；本文档负责解释策略含义，内容需与 `gate-policies.json` 保持一致。

## 三级门禁定义

### PR 级门禁（pr）

**定位**：快速、硬性、最小范围。适用于每次 PR 提交。

**执行子系统**：
1. 规则检查（subsystem_1）— 仅 rule-precheck + vv-scope-boundary + vv-mock-detection
2. 回归门禁（subsystem_5）— vv-regression-detection + gate-result

**阈值**：
- 每个子系统 score >= 70
- 零 CRITICAL 级 finding
- 允许 HIGH/MEDIUM/LOW 级 finding

**超时**：5 分钟

---

### 每夜全量门禁（nightly）

**定位**：全量、深度、覆盖所有子系统。适用于每夜定时执行。

**执行子系统**：
1. 规则检查（subsystem_1）— 全部 rule-check skills
2. 结构测试（subsystem_2）— 全部 structural-test skills
3. 任务评测（subsystem_3）— 全部 task-eval skills（需 SPEC_DIR）
4. 轨迹评分（subsystem_4）— 全部 trace-grading skills（需 TRACE_FILE）
5. 回归门禁（subsystem_5）— 全部 regression-gate skills

**阈值**：
- 每个子系统 score >= 80
- 零 CRITICAL 级 finding
- 零 HIGH 级 finding
- 允许 MEDIUM/LOW 级 finding

**超时**：30 分钟

---

### 发布级门禁（release）

**定位**：最严格，发布前最后一道防线。

**执行子系统**：
1-5 同 nightly，额外检查：
- 版本号一致性（go.mod version / tag / CHANGELOG）
- CHANGELOG 更新检查
- TODO/FIXME 扫描（不允许遗留）

**阈值**：
- 每个子系统 score >= 90
- 零 CRITICAL 级 finding
- 零 HIGH 级 finding
- 零 MEDIUM 级 finding
- 仅允许 LOW 级 finding

**超时**：60 分钟

---

## 裁决规则（Verdict Rules）

| 条件 | 裁决 |
|------|------|
| 所有子系统 pass 且满足阈值 | `PASS` |
| 存在 CRITICAL finding，或部分子系统不 pass 且分数远低于阈值（差距 > 10） | `FAIL` |
| 部分子系统不 pass 但无 CRITICAL，且分数接近阈值（差距 <= 10） | `SOFT_FAIL` |
| 涉及安全类 finding 需人工确认 | `NEEDS_APPROVAL` |

## 阈值配置覆盖

可通过 `$OUTPUT_DIR/vv/config.json` 中的 `threshold_overrides` 字段覆盖默认阈值：

```json
{
  "threshold_overrides": {
    "rule_checks": { "min_score": 60 },
    "structural_tests": { "min_score": 75 }
  }
}
```

## 一致性要求

- 编排器执行前应先校验 `gate-policies.json`
- 当 `gate-policies.json` 与本文档不一致时，以 `gate-policies.json` 为准，并应立即修正文档
- 新增门禁级别、子系统、超时或阈值时，必须同步更新 `gate-policies.json`、本文档和相关 agent 文档
