# HTTP 接口测试指引

本文档定义 HTTP 服务的接口测试流程。所有 byte-cli 命令均使用 `<base-dir>/assets/config.json` 作为配置文件。

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

如果用户已明确指定要测试的接口（提供了 HTTP method 和 path），跳过此步骤。

否则，查询该服务的所有 HTTP 接口：

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
  {"func_name": "ListTasks", "method": "GET", "path": "/api/pipeline/v1/tasks"},
  {"func_name": "CreateTasks", "method": "POST", "path": "/api/pipeline/v1/tasks"}
]
```

**将接口列表展示给用户，让用户选择要测试的接口。**

---

## Step 3：获取请求模板

如果用户已提供完整的请求内容（headers、query params、body），跳过此步骤。

### 3.1 获取接口 Schema

查询目标接口的请求/响应 Schema：

```bash
byte-cli --config $CONFIG ApiTest CN GetHttpApiSchema \
  --psm <PSM> \
  --idl-version <IDL_BRANCH> \
  --http-method <METHOD> \
  --http-path <PATH>
```

| 参数 | 说明 |
|------|------|
| `--psm` | 服务 PSM 名称（必填） |
| `--idl-source` | IDL 来源，默认 `1` |
| `--idl-version` | IDL 分支名，默认 `master` |
| `--http-method` | 目标接口的 HTTP 方法，如 `GET`、`POST`（必填） |
| `--http-path` | 目标接口的 HTTP 路径，如 `/api/pipeline/v1/tasks`（必填） |
| `--source` | 来源标识，默认 `1` |

返回内容包含 `request` 字段，其中：

- `headers`：请求头定义（key、desc）
- `query_params`：查询参数定义（key、desc、optional）
- `body` / `req_body`：请求体 Schema

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
| `--function-name` | 接口方法名称，即 Step 2 返回的 `func_name`（必填） |
| `--protocol` | 协议类型，默认 `thrift` |
| `--api-test-ctrl-type` | 推荐类型（1=首次推荐，2=基于历史），默认 `1` |

返回的 `data.api_test_traffic.req_body` 即为推荐的请求体 JSON 字符串。

如果调用失败或无推荐数据，回退使用 Step 3.1 的 Schema 默认值。

### 3.3 确认请求内容

**根据 Schema 和推荐请求体，与用户沟通确认要发起的请求内容**，包括：

- 哪些参数需要用户填写实际值
- 可选参数是否需要传递
- 请求体的具体内容

---

## Step 4：发送请求

根据前面步骤获取的信息，通过接口测试平台代理向目标服务发送 HTTP 请求。

### 4.1 确认请求信息

**发送前，必须先向用户展示完整的请求信息，获得用户确认后再发送。** 展示内容包括：

- **PSM**：`<PSM>`
- **IDL Version**：`<IDL_BRANCH>`
- **Target Host**：`<HTTP_HOST>`
- **Method**：`<METHOD>`
- **Path**：`<PATH>`
- **Online**：`false`
- **Headers**（格式化展示）
- **Query Parameters**（格式化展示）
- **Request Body**（格式化展示）

等待用户确认（或调整）后，再执行后续步骤。

### 4.3 准备输出目录与文件名

请求的响应可能很大，**必须将输出重定向到文件**。如果用户未指定输出路径，使用以下默认规则：

```bash
LOG_DIR=./api-test-log
mkdir -p "$LOG_DIR"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
# METHOD: 目标接口的 HTTP 方法（大写），如 GET、POST
# API_NAME: 接口路径的最后一段，如 tasks_v1
REQ_FILE="$LOG_DIR/${TIMESTAMP}_${METHOD}_${API_NAME}_req.md"
RESP_FILE="$LOG_DIR/${TIMESTAMP}_${METHOD}_${API_NAME}_resp.json"
```

### 4.4 记录请求信息

将本次请求的关键信息写入 `_req.md` 文件，便于回溯：

```bash
cat > "$REQ_FILE" << 'REQEOF'
# API Test Request

- **PSM**: <PSM>
- **IDL Version**: <IDL_BRANCH>
- **Target Host**: <HTTP_HOST>
- **Method**: <METHOD>
- **Path**: <PATH>
- **Online**: false

## Headers

```json
<HEADERS_JSON>
```

## Query Parameters

```json
<QUERY_JSON>
```

## Request Body

```json
<BODY_CONTENT>
```
REQEOF
```

### 4.5 发送请求并保存响应

```bash
byte-cli --config $CONFIG ApiTest CN SendHttpRequest \
  --psm <PSM> \
  --http-method <METHOD> \
  --http-path <PATH> \
  --http-host <HTTP_HOST> \
  --idl-version <IDL_BRANCH> \
  --http-req-headers '<HEADERS_JSON>' \
  --http-query '<QUERY_JSON>' \
  --req-body '<BODY_CONTENT>' \
  | jq '.' > "$RESP_FILE"
```

| 参数 | 说明 |
|------|------|
| `--psm` | 服务 PSM 名称（必填） |
| `--http-method` | 目标接口的 HTTP 方法，如 `GET`、`POST`（必填） |
| `--http-path` | 目标接口的 HTTP 路径（必填） |
| `--http-host` | 目标服务地址，如 `https://service-boe.bytedance.net`（必填） |
| `--online` | 是否发送到线上环境，默认 BOE（Flag，传此参数表示线上） |
| `--preset-env-id` | 预设环境 ID（BOE 环境可选） |
| `--idl-version` | IDL 分支名，默认 `master` |
| `--http-req-headers` | 请求头，JSON 数组字符串，每项含 `key`/`value`，可选 `optional`/`desc` |
| `--http-query` | 查询参数，JSON 数组字符串，每项含 `key`/`value` |
| `--req-body` | 请求体内容（字符串格式，用于 POST/PUT 等方法） |
| `--request-timeout` | 请求超时时间（毫秒），默认 `60000` |

**关于 `--http-req-headers` 和 `--http-query` 的格式**：

值为 JSON 数组字符串，示例：

```
--http-req-headers '[{"key":"X-Jwt-Token","value":"xxx"},{"key":"x-use-ppe","value":"1","optional":true}]'
--http-query '[{"key":"page_index","value":"0"},{"key":"page_size","value":"10"}]'
```

如不需要传递，可省略（默认为空数组）。

### 4.6 输出结果

请求完成后：

1. **告知用户文件路径**：说明响应已保存到 `$RESP_FILE`，请求信息已保存到 `$REQ_FILE`。
2. **摘要响应关键信息**：从响应文件中提取并展示：
   - `data.http_status_code`：目标服务的 HTTP 状态码
   - `data.req_latency`：请求耗时
   - `data.log_id`：日志 ID（可用于 Argos 追踪）
   - `data.error`：错误信息（如有）
   - `data.argos_link`：Argos 日志链接
   - `data.resp_body`：响应体摘要（仅展示前几行或关键字段，避免信息过载）

```bash
jq '{http_status_code: .data.http_status_code, req_latency: .data.req_latency, log_id: .data.log_id, error: .data.error, argos_link: .data.argos_link}' "$RESP_FILE"
```

如果用户需要查看完整响应体，可以执行：

```bash
jq -r '.data.resp_body' "$RESP_FILE" | jq '.'
```

