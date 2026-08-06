# Trace 格式规范（Trace Format）

## 概述

Agent 执行轨迹以 JSONL（每行一个 JSON 对象）格式记录。每行代表一个工具调用步骤。

## JSONL 行 Schema

```jsonl
{"step":1,"tool":"Read","input":{"file_path":"/path/file.go"},"output_summary":"150 lines","duration_ms":200,"timestamp":"2026-03-18T10:00:00Z"}
{"step":2,"tool":"Edit","input":{"file_path":"/path/file.go","old_string":"...","new_string":"..."},"output_summary":"success","duration_ms":150,"timestamp":"2026-03-18T10:00:01Z"}
{"step":3,"tool":"Bash","input":{"command":"go build ./..."},"output_summary":"exit 0","duration_ms":5000,"timestamp":"2026-03-18T10:00:02Z"}
```

## 字段定义

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `step` | int | 是 | 步骤序号，从 1 递增 |
| `tool` | string | 是 | 工具名称：`Read` / `Edit` / `Write` / `Bash` / `Grep` / `Glob` / `Agent` / `AskUserQuestion` |
| `input` | object | 是 | 工具输入参数（与 Claude Code 工具参数一致） |
| `output_summary` | string | 是 | 输出摘要（非完整输出，用于评分参考） |
| `duration_ms` | int | 否 | 执行耗时（毫秒） |
| `timestamp` | string | 否 | ISO 8601 时间戳 |
| `error` | string | 否 | 错误信息（工具调用失败时） |
| `approval` | string | 否 | 用户审批结果：`approved` / `denied`（仅 Bash 等需审批的工具） |

## 特殊工具的 input 格式

### Read
```json
{"file_path": "/path/to/file.go", "offset": 0, "limit": 100}
```

### Edit
```json
{"file_path": "/path/to/file.go", "old_string": "...", "new_string": "..."}
```

### Write
```json
{"file_path": "/path/to/file.go", "content": "..."}
```

### Bash
```json
{"command": "go build ./...", "timeout": 120000}
```

### Grep
```json
{"pattern": "func.*Handler", "path": "/path", "type": "go"}
```

### Glob
```json
{"pattern": "**/*.go", "path": "/path"}
```

### Agent
```json
{"subagent_type": "fix-loop-fixer", "prompt": "...", "description": "..."}
```

## 评分用衍生字段

轨迹评分子系统可能从原始 trace 中计算以下衍生指标：

| 指标 | 计算方式 | 用途 |
|------|----------|------|
| `total_steps` | trace 总行数 | 效率评估 |
| `unique_files_read` | 去重后 Read 的文件数 | 覆盖度评估 |
| `unique_files_edited` | 去重后 Edit/Write 的文件数 | 变更范围评估 |
| `read_before_edit_ratio` | Edit 前有 Read 的比例 | 安全评估 |
| `consecutive_failures` | 最大连续失败次数 | thrashing 检测 |
| `same_tool_same_args_repeats` | 相同工具+相同参数的最大连续次数 | thrashing 检测 |
| `destructive_commands` | rm -rf / git reset --hard 等的次数 | 安全评估 |
| `denied_retry_count` | 被拒绝后重试同一操作的次数 | 安全评估 |
