# Tech Debt Log

本文件追踪项目中已识别的技术债。每条记录包含问题描述、风险评估和建议修复方案。

**数据来源**：harness-debt-fix 技能扫描后自动追加，或人工在 Code Review 中发现后手动添加。

**处理流程**：定期 review 此文件，将高优先级条目转化为 ExecPlan（`docs/plans/proposal/`）。修复完成后将条目移至 Resolved Debt 表。

## Active Debt

| ID | Area | Type | Description | Risk | Suggested Fix | Status |
|---|---|---|---|---|---|---|
| TD-001 | _module_ | _duplicate-impl / scattered-state / ..._ | _description_ | _high/medium/low_ | _fix suggestion_ | open |

## Resolved Debt

| ID | Area | Resolution | Resolved Date |
|---|---|---|---|
| _TD-XXX_ | _module_ | _how it was resolved_ | _YYYY-MM-DD_ |

## Debt Types

- `duplicate-persistence`: 多处存储相同数据
- `divergent-impl`: 同一功能多处独立实现
- `model-divergence`: 跨层数据模型不一致
- `scattered-state`: 配置/状态散落多处
- `unabstracted-flow`: 相似流程未抽象
- `missing-test`: 关键路径缺少测试覆盖
- `outdated-doc`: 文档与代码不一致
