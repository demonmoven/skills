# 关联发布单流程

当用户输入"关联发布单"，或准出检测中 `releaseTicketNotAssociated == true` 时，执行以下流程。

## Step 1: 获取开发任务基础信息

调用开发任务详情获取基础信息：

```bash
bitscli devops dev-task get <dev_task_id>
```

从返回的 `data.basic_info` 中提取：

| 响应字段 | CLI 参数 | 说明 |
|---|---|---|
| `basic_info.space_id` | `--workspace-id` | 空间 ID |
| `basic_info.team_flow_id` | `--team-flow-id` | 研发流程 ID |
| `basic_info.dev_task_template_id` | `--dev-task-workflow-id` | 开发任务模版 ID |

> 若该信息已在上下文中获取过（如阶段完成流程的 Step 1 已调用 `dev-task get`），可复用，无需重复调用。

## Step 2: 查询可关联发布单

```bash
bitscli devops dev-task process associate-release-tickets --workspace-id <space_id> --team-flow-id <team_flow_id> --dev-task-workflow-id <dev_task_template_id> --page 1 --page-size 1000
```

其中 `teamFlowId` 直接来自 `data.basic_info.team_flow_id`。请求接口：
`GET /api/v1/cd/team_flow_config/{teamFlowId}/release_tickets?devTaskWorkflowId=...&page=1&pageSize=1000&workspaceId=...`

返回 `statusToRtList` 中状态 `3`（集成中）的前 5 条发布单。

## Step 3: 展示可关联发布单列表

按以下格式逐条展示（带序号，每行一条）：

```text
可关联发布单（集成中）：
1. [集成中]<发布单名称>  <创建时间> <创建人> <研发流程名称>
2. [集成中]<发布单名称>  <创建时间> <创建人> <研发流程名称>
...
```

- 若列表为空，输出：`未查询到可关联的集成中发布单，请前往 Bits 页面手动操作。` 并结束流程。

## Step 4: 用户选择并绑定发布单

列表展示后，向用户确认：

```text
请选择要关联的发布单序号（输入序号），或输入 0 跳过：
```

- 若用户输入 `0` 或明确表示跳过：输出 `已跳过关联发布单。` 并结束。
- 若用户选择了某条发布单：从该发布单的返回数据中提取 `integrationId`，执行绑定：

```bash
bitscli devops dev-task process bind-integration --dev-task-id <dev_task_id> --integration-id <integrationId>
```

- 绑定成功后输出：

```text
已成功将开发任务 <dev_task_id> 关联到发布单「<发布单名称>」。
```

- 绑定失败时输出错误信息，并附操作链接：
  `关联失败：<错误信息>。请前往开发任务页手动操作：https://bits.bytedance.net/devops/{space_id}/develop/detail/{dev_task_id}?devops_space_type=server_fe`

## 示例输出

```text
❌ 发布单可合入性检查  [阻塞] 当前开发任务未关联发布单，请关联一个处于"集成中"的发布单
可关联发布单（集成中）：
1. [集成中]发布单-A  2026-03-30 10:00:00  alice  研发流程-主干
2. [集成中]发布单-B  2026-03-29 18:23:11  bob  研发流程-主干
请选择要关联的发布单序号，或输入 0 跳过：
```

用户输入 `1` 后：

```text
已成功将开发任务 12345 关联到发布单「发布单-A」。
```
