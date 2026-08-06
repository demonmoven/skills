# 迭代数据目录结构和文件格式

## 目录结构

new-byted-gan-fixloop 在 `$OUTPUT_DIR` 下组织所有产出数据。Stage 0（craft）的产出放在 `craft/` 子目录，每轮迭代在独立的 `iteration_N/` 子目录中，确保数据隔离不被覆盖。

```
$OUTPUT_DIR/                            # 默认 .costudio/ 或 /tmp/workspace/.costudio/
├── craft/                              # Stage 0: craft 产出
│   ├── user_journeys.md                # Step 0.1: 用户动线
│   ├── stage1_to_stage2_coverage.md    # Step 0.2: Coverage analysis: 动线→E2E
│   ├── e2e_work_copy/                  # Step 0.3: 生成的 E2E 测试代码
│   ├── generated_test_cases.jsonl      # Step 0.3: E2E 测试清单
│   ├── test_dirs.txt                   # Step 0.4: 自动生成的测试列表（每行 TestFuncName ./dir）
│   ├── stage2_to_stage3_coverage.md    # Step 0.5: Coverage analysis: E2E→单元测试
│   ├── unit_tests/                     # Step 0.6: 生成的单元测试代码
│   ├── generated_unit_test_cases.jsonl # Step 0.6: 单元测试清单
│   └── unit_test_dirs.txt             # Step 0.7: 自动生成的单元测试列表（每行 TestFuncName ./dir）
├── iteration_1/
│   ├── failed_cases.jsonl              # 失败用例（Stage 2 集成测试写入）
│   ├── test_stats.json                 # 测试统计（Stage 2 集成测试写入）
│   ├── analysis_report.md              # Judge 分析报告（Stage 3 失败分析写入）
│   ├── fix_summary.md                  # 修复摘要（Stage 4 代码修复写入）
│   └── iteration_summary.md            # 迭代经验总结（Stage 5 迭代总结写入）
├── iteration_2/
│   └── ...
└── iteration_N/
    └── ...
```

## 文件格式

### craft/generated_test_cases.jsonl

JSONL 格式，每行一个 JSON 对象，表示一个生成的 E2E 测试用例。

```json
{"Package":"github.com/org/repo/test_cases/PE/CreateLabel/P0","Test":"Test_CreateLabel_ValidInput"}
```

### craft/test_dirs.txt

纯文本文件，每行一个测试条目，格式为 `TestFuncName ./relative/dir`。由 new-byted-gan-fixloop Stage 0 从 `generated_test_cases.jsonl` 自动提取生成。

> **注意**：当 `GENERATE_TESTS=false`（跳过 Stage 0）时，此文件不会生成。Stage 2 会自动降级为 `./...`（执行测试仓库下所有测试）。如需精确控制测试范围，可手动创建此文件，格式如下（每行一条，空行和 `#` 开头的注释行会被忽略）：

```
Test_CreateLabel_ValidInput ./test_cases/PE/CreateLabel/P0
Test_UpdateLabel_ValidInput ./test_cases/PE/UpdateLabel/P0
Test_DeleteLabel_ValidInput ./test_cases/PE/DeleteLabel/P0
```

### failed_cases.jsonl

JSONL 格式（JSON Lines），每行一个 JSON 对象，表示一个失败的测试用例。

```json
{"package":"github.com/org/repo/test_cases/PE/CreateLabel/P0","test":"Test_CreateLabel_ValidWorkspaceIdAndKey","status":"fail","message":"=== RUN   Test_CreateLabel...\n--- FAIL: (1.23s)\n    test.go:42: expected 200, got 500\n","elapsed":1.234,"start_time":"2026-03-09T10:00:00Z","end_time":"2026-03-09T10:00:01Z","log_ids":["20260309100000C91A145A63CB5F0B9D80"]}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `package` | string | Go 包路径 |
| `test` | string | 测试函数名（build_fail 时为空） |
| `status` | string | `fail`、`build_fail` 或 `skip` |
| `message` | string | 错误输出（包含完整的测试输出） |
| `elapsed` | float | 耗时秒数 |
| `start_time` | string | 开始时间（RFC3339） |
| `end_time` | string | 结束时间（RFC3339） |
| `log_ids` | string[] | 从测试输出中提取的 LogID 列表。无 LogID 时为空数组 `[]`（不得为 `null` 或省略该字段） |

### test_stats.json

单个 JSON 对象，汇总本轮测试统计。

```json
{
  "total_tests": 10,
  "passed_cases": 7,
  "failed_cases": 2,
  "skipped_cases": 1
}
```

`total_tests = passed_cases + failed_cases + skipped_cases`

### analysis_report.md

Markdown 格式的分析报告，末尾包含结构化 JSON 摘要。

JSON 摘要包裹在 HTML 注释标记中：
```
<!-- JUDGE_STRUCTURED_OUTPUT_START -->
```json
{...}
```
<!-- JUDGE_STRUCTURED_OUTPUT_END -->
```

JSON 结构：
```json
{
  "summary": {
    "total_failed": 2,
    "total_skipped": 1,
    "e2e_test_code_issues": 1,
    "business_code_issues": 1,
    "uncertain_issues": 1
  },
  "recommendation": "FIX_BOTH",
  "fix_priority": [
    {
      "priority": 1,
      "test_name": "TestLogin",
      "status": "fail",
      "classification": "BUSINESS_CODE",
      "severity": "critical",
      "root_cause": "API returns 500",
      "fix_suggestion": "Fix null pointer in handler",
      "files_to_modify": ["/abs/path/handler.go"],
      "log_ids": ["20260309100000C91A145A63CB5F0B9D80"],
      "log_evidence": "panic: runtime error: invalid memory address or nil pointer dereference at handler.go:42"
    }
  ]
}
```

`fix_priority` 最多 5 条记录。

### fix_summary.md

Markdown 格式的修复摘要，记录本轮修复了哪些问题。

```markdown
## 本轮修复摘要

### 修复的问题（按优先级）

1. **TestLogin** (priority: 1, classification: BUSINESS_CODE)
   - 修改文件: `/abs/path/handler.go`
   - 修改内容: 修复空指针异常
   - 修复原因: handler 未检查 nil

### 统计
- 修复的失败测试: 1 个
- 修复的跳过测试: 0 个
- 本轮修改的文件总数: 1 个
```

### iteration_summary.md

Markdown 格式的迭代经验总结，合成所有历史迭代的数据。

内容包含：测试通过率趋势、已解决/未解决的问题、反复出现的模式、下一轮建议。

## 跨阶段数据传递

| 数据 | 来源 | 目标 | 传递方式 |
|------|------|------|---------|
| E2E 测试代码 | Stage 0 craft-stage2 (e2e_work_copy/) | TEST_REPO_PATH | 文件同步 (cp) |
| 单元测试代码 | Stage 0 craft-stage3 (unit_tests/) | BUSINESS_REPO_PATH | 文件同步 (cp) |
| 测试列表 | Stage 0 (generated_test_cases.jsonl) | Stage 2 (test_dirs.txt → TEST_SCOPE) | 自动提取生成 test_dirs.txt（每行 TestFuncName ./dir） |
| TCE 泳道名 | Stage 1 首轮 | 后续所有轮的 Stage 1、2 | 对话上下文中记忆 |
| 测试是否通过 | Stage 2 的 test_stats.json | 循环条件判断 | 读取文件检查 `failed_cases == 0` |
| 失败用例列表 | Stage 2 | Stage 3 | 文件路径传递 |
| LogIDs（per 失败用例） | Stage 2 (failed_cases.jsonl `log_ids` 字段) | Stage 3 (failure-analysis 日志查询) | 文件路径传递 |
| PSM | new-byted-gan-fixloop 参数 | Stage 3 (failure-analysis `--psm` 过滤) | Agent prompt 参数 |
| BYTEDCLI_SITE | new-byted-gan-fixloop 参数 | Stage 3 (failure-analysis `--site` 多站点) | Agent prompt 参数 |
| 分析报告 | Stage 3 | Stage 4 | 文件路径传递 |
| 修复摘要 | Stage 4 | Stage 5 | 文件路径传递 |
| 迭代经验总结 | Stage 5 第 N 轮 | Stage 4 第 N+1 轮 | `HISTORY_SUMMARY_FILE` 参数 |
