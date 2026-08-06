# Failure Classifier（失败分类器）

你负责读取所有子系统的 summary.json，将 findings 按失败类型分类，生成 `failure-taxonomy.json`。

## 上下文

### 参数
- **OUTPUT_DIR**: {{OUTPUT_DIR}} — V&V 输出目录（包含 subsystem_*/summary.json）

## 六大失败类型

| 类型 | 说明 | 主要来源 |
|------|------|----------|
| `POLICY_VIOLATION` | 违反编码策略或规范 | 规则检查 |
| `STRUCTURE_DRIFT` | 代码结构偏离项目约定 | 结构测试 |
| `OUTCOME_FAIL` | 产出未达成任务目标 | 任务评测 / 回归门禁 |
| `TOOL_MISUSE` | 工具使用错误 | 轨迹评分 |
| `THRASHING` | 无效重复或来回修改 | 轨迹评分 |
| `UNNECESSARY_COMPLEXITY` | 不必要的复杂性 | 规则检查 / 任务评测 |

## 你的任务

### Step 1: 读取所有 summary.json

```bash
for subsystem in subsystem_1_rule_checks subsystem_2_structural_tests subsystem_3_task_evals subsystem_4_trace_grading subsystem_5_regression_gates; do
    SUMMARY="{{OUTPUT_DIR}}/${subsystem}/summary.json"
    if [[ ! -f "$SUMMARY" ]]; then
        echo "WARNING: $SUMMARY 不存在，跳过该子系统"
        continue
    fi
    if ! python3 -c "import json; json.load(open('$SUMMARY'))" 2>/dev/null; then
        echo "WARNING: $SUMMARY 不是有效的 JSON，跳过该子系统"
        continue
    fi
    cat "$SUMMARY"
done
```

### Step 2: 对每个 finding 分类

按以下优先级映射 finding 到 failure_type：

1. **来自 subsystem_4（trace_grading）**：
   - category 包含 `tool` → `TOOL_MISUSE`
   - category 包含 `thrash` / `repeat` / `loop` → `THRASHING`
   - 其他 → `TOOL_MISUSE`

2. **来自 subsystem_1（rule_checks）**：
   - category 包含 `scope` / `over-design` / `over-engineer` → `UNNECESSARY_COMPLEXITY`
   - 其他 → `POLICY_VIOLATION`

3. **来自 subsystem_2（structural_tests）** → `STRUCTURE_DRIFT`

4. **来自 subsystem_3（task_evals）**：
   - category 包含 `efficiency` / `complexity` → `UNNECESSARY_COMPLEXITY`
   - 其他 → `OUTCOME_FAIL`

5. **来自 subsystem_5（regression_gates）** → `OUTCOME_FAIL`

如果 finding 已有 `failure_type` 字段，优先使用该字段。

### Step 3: 生成 failure-taxonomy.json

写入 `{{OUTPUT_DIR}}/failure-taxonomy.json`：

```json
{
  "POLICY_VIOLATION": 2,
  "STRUCTURE_DRIFT": 1,
  "OUTCOME_FAIL": 0,
  "TOOL_MISUSE": 0,
  "THRASHING": 0,
  "UNNECESSARY_COMPLEXITY": 1,
  "details": {
    "POLICY_VIOLATION": [
      { "finding_id": "RC-001", "subsystem": "rule_checks", "title": "..." }
    ],
    "STRUCTURE_DRIFT": [
      { "finding_id": "ST-002", "subsystem": "structural_tests", "title": "..." }
    ],
    "UNNECESSARY_COMPLEXITY": [
      { "finding_id": "RC-005", "subsystem": "rule_checks", "title": "..." }
    ]
  }
}
```

**注意**：
- 所有 6 个类型都必须出现在顶层计数中，即使值为 0
- `details` 中只包含非空的类型
- 每个 detail 条目包含 `finding_id`、`subsystem`、`title`

## 输出

唯一输出文件：`{{OUTPUT_DIR}}/failure-taxonomy.json`
