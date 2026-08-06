# pytest 结果解析规则

描述如何从 `--junitxml` + `--alluredir` 双路产物中合并生成 `failed_cases.jsonl` 和 `test_stats.json`。

## 输入

- `$TEST_REPO_PATH/output/result.xml` — pytest 的 junit XML
- `$TEST_REPO_PATH/allure_report/xml/*.json` — allure 每条用例一个 JSON 文件（`<testCaseId>-result.json`），以及 `*-attachment.*` 附件文件
- 可选：`$BUSINESS_REPO_PATH/$sub_repo/go_unit.jsonl` — Go 单元测试 `go test -json` 输出

## junit XML 解析

每个 `<testsuite>` 包含多个 `<testcase>` 节点：

```xml
<testcase classname="testcases.im.chat.test_sse_send_message"
          name="test_send[params0-Env.Online]"
          time="12.345">
  <failure message="AssertionError: ...">Traceback (most recent call last): ...</failure>
</testcase>
```

字段提取：
- `nodeid` = `<classname 替换 . 为 />.py::<name>` → 例 `testcases/im/chat/test_sse_send_message.py::test_send[params0-Env.Online]`
- `status` = `failed`（有 `<failure>`）/ `error`（有 `<error>`）/ `skipped`（有 `<skipped>`）/ `passed`（无子节点）
- `failure_message` = `<failure>` 或 `<error>` 的 `message` 属性
- `failure_traceback` = `<failure>` 或 `<error>` 的文本内容
- `duration` = `time` 属性

## allure XML 解析

对每个 `*-result.json` 文件：

```json
{
  "uuid": "...",
  "fullName": "testcases.im.chat.test_sse_send_message#test_send[params0-Env.Online]",
  "status": "failed",
  "statusDetails": {"message": "...", "trace": "..."},
  "steps": [{"name": "请求会话信息", "status": "passed", ...}, ...],
  "attachments": [{"name": "LogID", "source": "xxx.txt", "type": "text/plain"}],
  "parameters": [{"name": "params", "value": "params0"}, {"name": "env", "value": "Env.Online"}]
}
```

字段提取：
- `allure_detail.steps`：原样保留完整树
- `allure_detail.attachments`：保留 `{name, source, type}` 数组；`source` 是 `allure_report/xml/` 下相对文件名
- `allure_detail.logs`：如有 `attachment.name == "log"` 的附件，读取其内容合并进来（上限 10 KB，超长截断并标注）

## 合并规则

以 junit XML 的 `nodeid` 为主键，与 allure 的 `fullName` 对齐（替换 `#` 为 `::` 后比对）。

输出 `$OUTPUT_DIR/iteration_$N/failed_cases.jsonl`，每行（仅保留 `failed` / `error` 状态的记录）：

```json
{
  "nodeid": "testcases/im/chat/test_xxx.py::test_foo[params0-Env.Online]",
  "status": "failed",
  "failure_message": "...",
  "failure_traceback": "...",
  "duration": 12.3,
  "params": {"ENV_LABEL": "ppe_xxx", "version_id": "...", "bot_id": "..."},
  "allure_detail": {
    "steps": [...],
    "attachments": [...],
    "logs": "..."
  }
}
```

`params` 从 allure `parameters` 数组提取（先解析 `params` 字段的 JSON 字符串得到 dict，其它直接键值）。

## Go 单元测试合并

对每个 sub-repo 的 `go_unit.jsonl`，按 [go test -json 行格式](https://pkg.go.dev/cmd/test2json) 解析：
- `Action=="run"` 记录测试起点
- `Action=="fail"` / `"pass"` / `"skip"` 记录终态
- `Action=="output"` 累积输出
- `nodeid` 格式：`$PSM:$Package.$Test`（例：`flow.alice.creativity:creativity/handler/TestXxx`）
- 合并进同一份 `failed_cases.jsonl`

## test_stats.json 产出

```json
{
  "iteration": 1,
  "total": 50, "passed": 45, "failed": 3, "skipped": 2, "error": 0,
  "duration_sec": 312,
  "breakdown": {
    "pytest": {"total": 40, "passed": 38, "failed": 2, "skipped": 0, "error": 0},
    "go_unit": {"total": 10, "passed": 7, "failed": 1, "skipped": 2, "error": 0}
  }
}
```

- `total` / `passed` / `failed` / `skipped` / `error` 是两路合计
- `duration_sec` 是从 Stage 2 开始到结束的 wall-clock 时间（不是各 testcase time 累加——并发跑不等价）
- `skipped > 0 且 failed == 0 且 error == 0` 视为通过（沿用原 fixloop 规则）

## 容错

- junit XML 解析失败（如测试中断未生成文件）：直接把 `failed_cases.jsonl` 写一条 `{"status": "infra_error", "failure_message": "pytest 进程崩溃"}`，Stage 3 会按 `[INFRA]` 处理
- allure XML 未生成（如 `--alluredir` 目录为空）：退化为只用 junit 信息，`allure_detail` 字段留空
- Go 单测 jsonl 缺失：跳过 go_unit breakdown
