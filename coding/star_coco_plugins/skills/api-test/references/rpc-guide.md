# RPC 接口测试指引

本文档定义 RPC 服务的接口测试流程。所有 byte-cli 命令均使用 `<base-dir>/assets/config.json` 作为配置文件。

**配置路径变量**（后续命令中统一使用）：

**重要**：`<base-dir>` 是系统提供的 Base directory for this skill，需要在实际使用时替换为真实路径。

```
CONFIG=<base-dir>/assets/config.json
```

---

## Step 1：确认 PSM 与 IDL 版本

从项目的 AGENTS.md 或 CLAUDE.md 中查找 `Project Meta` 章节，获取以下信息：

- **PSM**：服务的 Process Service Module 名称（必须）
- **IDL Branch**：IDL 仓库分支名（通常为 `master`）

如果上下文中无法确定，**询问用户**提供 PSM 和 IDL Branch。

### 可选：使用 BAM IDL Version 替代 IDL Branch

如果用户指定使用 BAM IDL Version（如 `1.0.206`）而非 IDL Branch，需要先查询该版本对应的 IDL Branch：

```bash
byte-cli --config $CONFIG ApiTest CN GetApiVersions \
  --psm <PSM> \
  --output-filter '.response_body.data[] | select(.version == "<VERSION>") | .idl_branch'
```

| 参数 | 说明 |
|------|------|
| `--psm` | 服务 PSM 名称（必填） |
| `--output-filter` | jq 表达式，按 version 过滤并提取 idl_branch |

该命令返回 IDL 版本列表（内容较多），**必须使用 `--output-filter` 提取所需字段，避免信息过载**。

将获取到的 `idl_branch` 值作为后续步骤中的 `IDL_BRANCH`。

---

## Step 2：获取接口列表

如果用户已明确指定要测试的 RPC 方法（提供了 `func_name`），跳过此步骤。

否则，查询该服务的所有接口：

```bash
byte-cli --config $CONFIG ApiTest CN GetApiList \
  --psm <PSM> \
  --idl-version <IDL_BRANCH>
```

| 参数 | 说明 |
|------|------|
| `--psm` | 服务 PSM 名称（必填） |
| `--idl-source` | IDL 来源，默认 `1` |
| `--idl-version` | IDL 分支名，默认 `master` |

返回示例：

```json
[
  {"func_name": "QueryClusters", "method": "", "path": ""},
  {"func_name": "QueryEnvs", "method": "", "path": ""}
]
```

RPC 服务的接口列表中 `method` 和 `path` 通常为空，**以 `func_name` 为准**。

**将接口列表展示给用户，让用户选择要测试的 RPC 方法。**

---

## Step 3：获取请求模板

如果用户已提供完整的请求体内容，跳过此步骤。

### 3.1 获取接口 Schema

查询目标 RPC 方法的请求/响应 Schema：

```bash
byte-cli --config $CONFIG ApiTest CN GetRpcApiSchema \
  --psm <PSM> \
  --func-name <FUNC_NAME> \
  --idl-version <IDL_BRANCH>
```

| 参数 | 说明 |
|------|------|
| `--psm` | 服务 PSM 名称（必填） |
| `--func-name` | RPC 方法名称，如 `QueryClusters`（必填） |
| `--idl-source` | IDL 来源，默认 `1` |
| `--idl-version` | IDL 分支名，默认 `master` |
| `--test-plane` | 测试面（1=线上，2=BOE），默认 `1` |
| `--source` | 来源标识，默认 `1` |

返回内容包含 `request` 字段，其中：

- `func_name`：RPC 方法名称
- `body` / `req_body`：请求体 Schema（JSON 字符串，包含所有字段和默认值）

### 3.2 获取 AI 推荐请求体（可选）

尝试获取基于历史流量的推荐请求体，作为填写参考：

```bash
byte-cli --config $CONFIG ApiTest CN GetRecommendRequest \
  --psm <PSM> \
  --function-name <FUNC_NAME>
```

| 参数 | 说明 |
|------|------|
| `--psm` | 服务 PSM 名称（必填） |
| `--function-name` | RPC 方法名称（必填） |
| `--protocol` | 协议类型，默认 `thrift` |
| `--api-test-ctrl-type` | 推荐类型（1=首次推荐，2=基于历史），默认 `1` |

返回的 `data.api_test_traffic.req_body` 即为推荐的请求体 JSON 字符串。

如果调用失败或无推荐数据，回退使用 Step 3.1 的 Schema 默认值。

### 3.3 确认请求内容

**根据 Schema 和推荐请求体，与用户沟通确认最终的请求体内容**，包括：

- 哪些字段需要用户填写实际值（如业务 ID、查询条件等）
- `Base` 字段中的 `TrafficEnv`、`Extra` 等是否需要调整
- 可选字段是否需要传递

---

## Step 4：确认目标环境与集群

RPC 请求需要指定目标环境（env）和集群（zone / idc / cluster）。

**如果用户已明确指定环境和集群信息，直接使用，跳过查询步骤。**

### 4.1 获取可用环境

```bash
byte-cli --config $CONFIG ApiTest CN GetEnvs \
  --psm <PSM> \
  --test-plane <TEST_PLANE>
```

| 参数 | 说明 |
|------|------|
| `--psm` | 服务 PSM 名称（必填） |
| `--test-plane` | 测试面（1=线上，2=BOE），默认 `2` |

返回示例：`["prod"]`

- 如果只有**一个环境**，直接使用，无需询问用户。
- 如果有**多个环境**，展示列表让用户选择。

将选定的环境记为 `ENV`。

### 4.2 获取可用集群

```bash
byte-cli --config $CONFIG ApiTest CN GetClusters \
  --psm <PSM> \
  --env <ENV> \
  --test-plane <TEST_PLANE>
```

| 参数 | 说明 |
|------|------|
| `--psm` | 服务 PSM 名称（必填） |
| `--env` | 环境名称，如 `prod`（必填） |
| `--test-plane` | 测试面（1=线上，2=BOE），默认 `2` |

返回示例：

```json
[
  {"cluster": "default", "zone": "BOE", "idc": "boe", "online": false},
  {"cluster": "test", "zone": "BOE", "idc": "boe", "online": false}
]
```

- 如果只有**一个集群**，直接使用，无需询问用户。
- 如果有**多个集群**，展示列表让用户选择。

从选定的集群中提取 `ZONE`、`IDC`、`CLUSTER` 三个值，以及 `ONLINE` 标识。

---

## Step 5：发送请求

根据前面步骤获取的所有信息，通过接口测试平台代理向目标服务发送 RPC 请求。

### 5.1 确认请求信息

**发送前，必须先向用户展示完整的请求信息，获得用户确认后再发送。** 展示内容包括：

- **PSM**：`<PSM>`
- **RPC Method**：`<FUNC_NAME>`
- **IDL Version**：`<IDL_BRANCH>`
- **Environment**：`<ENV>`
- **Cluster**：`<CLUSTER>` (zone: `<ZONE>`, idc: `<IDC>`)
- **Online**：`<ONLINE>`
- **Request Body**（格式化展示）

等待用户确认（或调整）后，再执行后续步骤。

### 5.2 准备输出目录与文件名

请求的响应可能很大，**必须将输出重定向到文件**。如果用户未指定输出路径，使用以下默认规则：

```bash
LOG_DIR=./api-test-log
mkdir -p "$LOG_DIR"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
# FUNC_NAME: RPC 方法名称，如 QueryClusters
REQ_FILE="$LOG_DIR/${TIMESTAMP}_RPC_${FUNC_NAME}_req.md"
RESP_FILE="$LOG_DIR/${TIMESTAMP}_RPC_${FUNC_NAME}_resp.json"
```

### 5.3 记录请求信息

将本次请求的关键信息写入 `_req.md` 文件，便于回溯：

```bash
cat > "$REQ_FILE" << 'REQEOF'
# RPC Test Request

- **PSM**: <PSM>
- **IDL Version**: <IDL_BRANCH>
- **RPC Method**: <FUNC_NAME>
- **Environment**: <ENV>
- **Cluster**: <CLUSTER>
- **Zone**: <ZONE>
- **IDC**: <IDC>
- **Online**: <ONLINE>

## Request Body

```json
<REQ_BODY>
```
REQEOF
```

### 5.4 发送请求并保存响应

```bash
byte-cli --config $CONFIG ApiTest CN SendRpcRequest \
  --psm <PSM> \
  --func-name <FUNC_NAME> \
  --req-body '<REQ_BODY>' \
  --idl-version <IDL_BRANCH> \
  --zone <ZONE> \
  --idc <IDC> \
  --cluster <CLUSTER> \
  --env <ENV> \
  | jq '.' > "$RESP_FILE"
```

如需发送到线上环境，追加 `--online` 参数。

| 参数 | 说明 |
|------|------|
| `--psm` | 服务 PSM 名称（必填） |
| `--func-name` | RPC 方法名称（必填） |
| `--req-body` | 请求体内容，JSON 字符串 |
| `--idl-version` | IDL 分支名，默认 `master` |
| `--zone` | 区域标识，如 `CN`、`BOE`（必填） |
| `--idc` | 机房标识，如 `hl`、`boe`（必填） |
| `--cluster` | 集群名称，如 `default`，默认 `default` |
| `--env` | 环境名称，如 `prod`（必填） |
| `--online` | 发送到线上环境（Flag，传此参数表示线上） |
| `--request-timeout` | 请求超时时间（毫秒），默认 `60000` |
| `--connect-timeout` | 连接超时时间（毫秒），默认 `60000` |

### 5.5 输出结果

请求完成后：

1. **告知用户文件路径**：说明响应已保存到 `$RESP_FILE`，请求信息已保存到 `$REQ_FILE`。
2. **摘要响应关键信息**：从响应文件中提取并展示：
   - `data.req_latency`：请求耗时
   - `data.log_id`：日志 ID（可用于 Argos 追踪）
   - `data.error`：错误信息（如有）
   - `data.biz_status_code`：业务状态码
   - `data.argos_link`：Argos 日志链接
   - `data.resp_body`：响应体摘要（仅展示前几行或关键字段，避免信息过载）

```bash
jq '{req_latency: .data.req_latency, log_id: .data.log_id, error: .data.error, biz_status_code: .data.biz_status_code, argos_link: .data.argos_link}' "$RESP_FILE"
```

如果用户需要查看完整响应体，可以执行：

```bash
jq -r '.data.resp_body' "$RESP_FILE" | jq '.'
```
