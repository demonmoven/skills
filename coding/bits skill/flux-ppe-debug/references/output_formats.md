# Output Formats Reference

本文件定义了 `flux-ppe-debug` 技能中所有命令的标准输入参数说明及输出格式模板。
所有输出 **必须** 使用 Markdown 表格，确保清晰可读。

---

## 1. Start Watchpoint Session

**Command:** `watchpoint-start-session`

**Input Parameters:**
- `psm` (Required): 目标服务 PSM。
- `env` (Required): 环境名称（如 `ppe_demo`、`ppe_xxx`）。
- `cluster` (Required): 集群名称（如 `default`）。
- `instance-name` (Required): Pod/实例名称。
- `commit-id` (Optional): Commit ID。
- `expired-at` (Optional): 过期时间（Unix 秒）。
- `standard-env` (Optional): 标准环境，默认为 `online_cn`。

**Output Description:**
展示创建的调试会话信息，并输出可用于后续命令的 `session_id`。

**Output Format:**

```text
### Watchpoint Session
| Item        | Value           | Description |
| :---        | :---            | :--- |
| Session ID  | {session_id}    | 会话 ID (API: `session_id`) |
| PSM         | {psm}           | 服务 PSM |
| Env         | {env}           | 环境 |
| Cluster     | {cluster}       | 集群 |
| Pod         | {instance_name} | Pod/实例名 |
| Expired At  | {expired_at}    | 过期时间 |
| Status      | {status}        | 会话状态 |
```

---

## 2. Add Watchpoints (Batch)

**Command:** `watchpoint-add-batch`

**Input Parameters:**
- `session-id` (Required): 会话 ID（来自 `watchpoint-start-session` 输出）。
- `breakpoints` (Required): 断点列表，格式为 `<file_path>:<line_number>`，多条用逗号分隔。
- `action` (Optional): 默认 `CAPTURE`。
- `psm` (Optional): 目标服务 PSM。
- `env` (Optional): 环境名称。
- `cluster` (Optional): 集群名称。
- `instance-name` (Optional): Pod/实例名称。
- `standard-env` (Optional): 标准环境，默认为 `online_cn`。

**Output Description:**
展示成功创建的 watchpoint 列表，并输出 `watchpoint_id` 供后续查询使用。

**Output Format:**

```text
### Watchpoints
| Watchpoint ID | File        | Line | Action  | Description |
| :---          | :---        | :--- | :---    | :--- |
| {id}          | {file}      | {n}  | CAPTURE | 断点创建成功 |
```

---

## 3. Query Watchpoint Info

**Command:** `watchpoint-info`

**Input Parameters:**
- `id` (Required): watchpoint ID。
- `session-id` (Optional): 会话 ID。
- `psm` (Optional): 目标服务 PSM。
- `env` (Optional): 环境名称。
- `cluster` (Optional): 集群名称。
- `pod` (Optional): Pod/实例名。
- `standard-env` (Optional): 标准环境，默认为 `online_cn`。

**Output Description:**
展示 watchpoint 的快照记录与堆栈信息概要。

**Output Format:**

```text
### Watchpoint Snapshot
| Item        | Value         | Description |
| :---        | :---          | :--- |
| Watchpoint ID | {id}        | 观察点/断点 ID |
| Record ID   | {record_id}   | 快照记录 ID |
| Status      | {status}      | 状态 |
| Message     | {message}     | 附加信息 |

### Stack Frames
| # | Function | File | Line |
| :-- | :--- | :--- | :--- |
| {i} | {fn} | {file} | {line} |
```

---

## 4. Delete Session / Close

**Command:** `watchpoint-delete-session`

**Input Parameters:**
- `id` (Optional): 会话数据库 ID。
- `session-id` (Optional): 会话 ID。
- `standard-env` (Optional): 标准环境，默认为 `online_cn`。
*注：`id` 和 `session-id` 至少需提供一个。*

**Output Description:**
关闭调试会话（调试结束必须执行）。

**Output Format:**

```text
### Delete Session Result
| Item      | Value     | Description |
| :---      | :---      | :--- |
| Session ID| {session_id} | 被关闭的会话 ID |
| Result    | {result}  | 关闭结果 |
| Message   | {message} | 附加信息 |
```

---

## 5. API Doc (Reference)

Watchpoint 请求的参考说明详见 `references/watchpoint_request_doc.md`。
