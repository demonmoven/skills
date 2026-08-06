# go test -json 输出格式和解析规则

## 输出格式

`go test -json` 将测试结果以 JSON Lines（每行一个 JSON 对象）格式输出到 stdout。

### 事件结构

```json
{
  "Time": "2026-03-09T10:00:00.123456Z",
  "Action": "run",
  "Package": "github.com/org/repo/test_cases/PE/CreateLabel/P0",
  "Test": "Test_CreateLabel_ValidWorkspaceIdAndKey",
  "Output": "=== RUN   Test_CreateLabel_ValidWorkspaceIdAndKey\n",
  "Elapsed": 1.234,
  "FailedBuild": ""
}
```

### 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `Time` | string | 事件发生时间（RFC3339） |
| `Action` | string | 事件类型，见下方 |
| `Package` | string | Go 包路径 |
| `Test` | string | 测试函数名（可为空，包级别事件时为空） |
| `Output` | string | 输出内容（仅 action=output 时有值） |
| `Elapsed` | float | 耗时秒数（仅 action=pass/fail/skip 时有值） |
| `FailedBuild` | string | 编译失败的包路径（仅编译失败时有值） |

### Action 类型

| Action | 说明 | 后续处理 |
|--------|------|---------|
| `run` | 测试开始运行 | 开始跟踪该测试 |
| `output` | 测试产生输出 | 累积到该测试的 outputs |
| `pass` | 测试通过 | 计数 +1，清理缓存 |
| `fail` | 测试失败 | 计数 +1，写入 JSONL，清理缓存 |
| `skip` | 测试跳过 | 计数 +1，写入 JSONL，清理缓存 |

## 解析规则

### 1. 编译失败检测

当 `Action == "fail"` 且 `FailedBuild` 字段非空时，表示**包编译失败**（不是测试失败）：

```json
{"Action":"fail","FailedBuild":"github.com/org/repo/pkg","Package":"github.com/org/repo/pkg"}
```

处理：记录为 `status: "build_fail"`，写入 JSONL。

### 2. 忽略包级别事件

当 `Test` 字段为空时，是包级别事件（如包整体 pass/fail），**应跳过**不计入测试用例统计。

### 3. 测试用例跟踪

按 `Package + "/" + Test` 为唯一键跟踪每个测试用例的生命周期：

```
run (开始) → output* (多次输出) → pass/fail/skip (结束)
```

### 4. 失败用例立即写入

检测到 `Action == "fail"` 的测试用例时，**立即**以 JSONL 格式追加写入 failed_cases.jsonl。不要等所有测试执行完毕。

## JSONL 输出 Schema（failed_cases.jsonl）

每行一个 JSON 对象：

```json
{
  "package": "github.com/org/repo/test_cases/PE/CreateLabel/P0",
  "test": "Test_CreateLabel_ValidWorkspaceIdAndKey",
  "status": "fail",
  "message": "=== RUN   Test_CreateLabel...\n--- FAIL: Test_CreateLabel (1.23s)\n    test.go:42: expected 200, got 500\n",
  "elapsed": 1.234,
  "start_time": "2026-03-09T10:00:00Z",
  "end_time": "2026-03-09T10:00:01Z",
  "log_ids": ["20260309100000C91A145A63CB5F0B9D80"]
}
```

### 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `package` | string | Go 包路径 |
| `test` | string | 测试函数名（build_fail 时为空） |
| `status` | string | `fail`、`build_fail` 或 `skip` |
| `message` | string | 错误输出（包含完整的测试输出） |
| `elapsed` | float | 耗时秒数 |
| `start_time` | string | 开始时间（RFC3339） |
| `end_time` | string | 结束时间（RFC3339） |
| `log_ids` | string[] | 从测试输出中提取的 LogID 列表（可为空数组） |

### LogID 提取

解析 `Action == "output"` 事件时，从 `Output` 内容中用正则提取 LogID：

```python
LOGID_PATTERN = re.compile(r'(?:X-Tt-Logid|x-tt-logid|logid|LogID)[=:\s]+([a-fA-F0-9]{20,})')
```

**提取时机**：每次累积 output 时扫描当前行，发现 LogID 则添加到该用例的 `log_ids` 列表。

**去重**：写入 JSONL 前对 `log_ids` 去重（`list(set(...))`）。

**空数组**：如果测试输出中没有 LogID 信息，`log_ids` 为空数组 `[]`。这是正常的——不是所有测试都会产生 HTTP 请求，或测试框架可能未打印响应头。

### status 取值

| status | 说明 |
|--------|------|
| `fail` | 测试用例执行失败（断言错误、panic 等） |
| `build_fail` | 包编译失败（此时 test 字段为空） |
| `skip` | 测试用例被跳过（`t.Skip()`、条件跳过等） |

## 边界情况

1. **非 JSON 行**：`go test -json` 偶尔输出非 JSON 内容（如 panic 堆栈），解析时跳过无法 JSON parse 的行
2. **超长输出行**：单行可能超过 64KB（大量 output），解析器需要支持大 buffer
3. **无 Test 字段的 fail**：包级别 fail 事件，可能伴随 `FailedBuild`，需要单独处理
4. **go test 退出码非零**：有测试失败时 `go test` 返回退出码 1，这是正常的，不应中断执行

## test_stats.json 格式

```json
{
  "total_tests": 10,
  "passed_cases": 7,
  "failed_cases": 2,
  "skipped_cases": 1
}
```

`total_tests = passed_cases + failed_cases + skipped_cases`
