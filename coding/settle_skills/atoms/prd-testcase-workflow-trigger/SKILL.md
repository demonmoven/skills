---
name: prd-testcase-workflow-trigger
description: 基于一个 PRD 文档链接和 PRD 标题，触发 qagents_next 平台用例生成 workflow（默认 workflow ID 10670）的 Skill。支持同步（默认）和异步两种调用模式。同步模式直接调用 RunWorkflowSync 等待返回；异步模式调用 RunWorkflow + 轮询 GetWorkflowTaskDetail 直到 status==3。无论何种模式，最终统一输出 {"bits_case_url":"..."} 格式。适用于：质量保障同学基于 PRD 文档批量触发用例生成、回归测试用例自动产出、把 qagents_next 的"PRD → 测试用例"流水线固化到 Aime 工作流的场景。
---

# PRD 测试用例生成 Workflow 触发器（同步 / 异步双模式）

## 关于 Skill

本 Skill 封装了 qagents_next 平台的"基于 PRD 自动生成测试用例" workflow 触发流程，支持**同步**（默认）和**异步**两种模式：

### 同步模式（默认）
1. **仅调用一次** `RunWorkflowSync` 接口，等待同步返回
2. 不调用 `GetWorkflowTaskDetail`
3. 从返回中递归查找 `bits_case_url` 并按 `{"bits_case_url":"<url>"}` 格式输出

### 异步模式
1. **仅调用一次** `RunWorkflow` 异步接口，从返回中取出 `workflowTaskID`
2. 按固定间隔（默认 30s）轮询 `GetWorkflowTaskDetail`，直到 `workflowTask.status == 3`
3. 从 `end_0` 节点取出 `output`，递归查找 `bits_case_url`，按 `{"bits_case_url":"<url>"}` 格式输出

接口规范：

- **同步触发**：`POST https://qagents-next-api.bytedance.net/apin/qagents_next/workflow/RunWorkflowSync`
- **异步触发**：`POST https://qagents-next-api.bytedance.net/apin/qagents_next/workflow/RunWorkflow`
- **任务详情**：`POST https://qagents-next-api.bytedance.net/apin/qagents_next/workflow/GetWorkflowTaskDetail`
- Header（所有接口一致）：
  - `Content-Type: application/json`
  - `Authorization: Bearer AHhnnlOprYywGXXY`（Access Key）
- 触发接口 Body（sync/async 相同）：
  ```json
  {
    "workflowID": "10670",
    "input": "{\"prd_link\":\"<PRD 文档链接>\",\"prd_name\":\"<PRD 文档标题>\"}"
  }
  ```
- 详情接口 Body（仅 async 模式使用）：
  ```json
  {
    "workflowTaskID": "<workflowTaskID>"
  }
  ```

> 注意：`input` 字段本身是字符串，内部承载再次序列化后的 JSON。脚本已自动处理这层嵌套。

## 何时使用

- 用户给出一个新的 PRD 文档链接（飞书 docx / wiki / 其他可访问 URL），希望走 qagents_next 自动生成测试用例
- 需要把"输入 PRD → 调起用例生成 workflow → 拿到 bits_case_url"的步骤标准化、可复用
- 需要在不同模式（sync/async）、不同 workflow ID、不同 Access Key 之间切换

## 输入说明

- **PRD 链接（必需）**：一个可被 qagents_next workflow 访问的 PRD 文档 URL，作为 `input.prd_link` 字段
- **PRD 标题（必需）**：`prd_link` 对应文档的标题名称，作为 `input.prd_name` 字段
  - 如用户未直接给出标题，需要先打开/读取该 PRD 链接拿到标题
  - 不要凭空编造，也不要把链接本身当标题
- **模式（可选）**：`--mode sync`（默认）或 `--mode async`
- **workflow ID（可选）**：默认 `10670`
- **Bearer Access Key（可选）**：默认 `AHhnnlOprYywGXXY`
- **轮询参数（可选，仅 async）**：默认每 30 秒轮询一次，最多 120 次（约 1 小时）

## 如何使用

直接通过 `bash` 工具执行脚本，**严禁**用 Aime 自带 HTTP 工具或自行编写新的 Python 脚本绕过此命令。

### 同步模式（默认）

```bash
python3 scripts/trigger_workflow.py \
  --prd-link "<PRD 文档链接>" \
  --prd-name "<PRD 文档标题>"
```

或显式声明：

```bash
python3 scripts/trigger_workflow.py --mode sync \
  --prd-link "<PRD 文档链接>" \
  --prd-name "<PRD 文档标题>"
```

### 异步模式

```bash
python3 scripts/trigger_workflow.py --mode async \
  --prd-link "<PRD 文档链接>" \
  --prd-name "<PRD 文档标题>"
```

### 完整可选参数

```bash
python3 scripts/trigger_workflow.py \
  --mode sync \
  --prd-link "<PRD 文档链接>" \
  --prd-name "<PRD 文档标题>" \
  --workflow-id 10670 \
  --api-url "https://qagents-next-api.bytedance.net/apin/qagents_next/workflow/RunWorkflowSync" \
  --auth-token "AHhnnlOprYywGXXY" \
  --timeout 600
```

```bash
python3 scripts/trigger_workflow.py \
  --mode async \
  --prd-link "<PRD 文档链接>" \
  --prd-name "<PRD 文档标题>" \
  --workflow-id 10670 \
  --api-url "https://qagents-next-api.bytedance.net/apin/qagents_next/workflow/RunWorkflow" \
  --detail-url "https://qagents-next-api.bytedance.net/apin/qagents_next/workflow/GetWorkflowTaskDetail" \
  --auth-token "AHhnnlOprYywGXXY" \
  --timeout 600 \
  --poll-interval 30 \
  --max-polls 120
```

## 输出格式

两种模式**统一**在 stdout 末尾追加一行：
```
BITS_CASE_RESULT={"bits_case_url":"https://bits.bytedance.net/devops/749690368002/quality/case/caseDetail/14503605"}
```

标准输出完整结构（JSON）：
```json
{
  "mode": "sync | async",
  "stage": "completed | completed_without_url | timeout | trigger_failed",
  "trigger_result": { /* 触发接口原始响应 */ },
  "bits_case_url": "https://...",
  // 以下仅 async 模式包含：
  "workflow_task_id": 12345678,
  "poll_attempts": 3,
  "status": 3,
  "end_node_output": { /* end_0 节点 output */ },
  "last_detail_response": { /* GetWorkflowTaskDetail 最近一次响应 */ }
}

BITS_CASE_RESULT={"bits_case_url":"..."}
```

## 退出码

| 退出码 | 含义 |
|--------|------|
| 0 | 成功，已拿到 `bits_case_url` |
| 1 | 触发接口失败（非 2xx）或解析 workflowTaskID 失败 |
| 2 | 异步轮询超时，未达到 `status == 3` |
| 3 | 流程完成但未能从返回中解析出 `bits_case_url` |

## 关键注意事项

- **触发接口仅调用一次**：无论 sync 还是 async，脚本仅在启动时调用一次触发接口
- **sync 模式不轮询**：RunWorkflowSync 是同步接口，直接等待返回即可，不需要也不会调用 GetWorkflowTaskDetail
- **async 模式根据 status==3 退出**：不再用具体业务字段判断，第一层 `workflowTask.status == 3` 即视为完成
- **输出统一为 {"bits_case_url":"..."}**：上层只需匹配 `BITS_CASE_RESULT={...}` 这一行
- **轮询间隔默认 30s**（仅 async）；最大 120 次（约 1 小时）
- **每次只处理一个 PRD 链接**：多个 PRD 请逐个调用
- **prd_name 必须与 prd_link 标题一致**：标题来自 PRD 文档本身
- **不要修改 `input` 嵌套结构**：脚本已自动处理
- **Access Key 默认值**：`AHhnnlOprYywGXXY`，只有当用户明确给了新值时才传入
- **结果回显**：必须把 `BITS_CASE_RESULT` 行中的 `bits_case_url` 完整呈现给用户
