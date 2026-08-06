# CLI 命令参考

本文档是 api-test skill 的**命令单一源**，集中维护所有 `bitscli api-test` 命令模板、入参说明与返回示例。`SKILL.md` 与 `flows/test-flow.md` 只引用本文件，不再各自内联命令块，避免多处维护导致漂移。

> 工具链安装/更新、`bitscli` 安装、JWT 获取与登录鉴权统一见 [runtime-support.md](../support/runtime-support.md)，本文件不再重复。

## 目录
- [CLI 命令参考](#cli-命令参考)
  - [目录](#目录)
  - [服务与 IDL 相关命令](#服务与-idl-相关命令)
    - [获取服务 IDL 设置](#获取服务-idl-设置)
    - [获取分支列表](#获取分支列表)
    - [查询服务接口列表](#查询服务接口列表)
    - [获取接口 Schema](#获取接口-schema)
  - [请求参数相关命令](#请求参数相关命令)
    - [获取推荐请求参数（历史流量）](#获取推荐请求参数历史流量)
  - [发送请求相关命令](#发送请求相关命令)
    - [发送 HTTP 请求](#发送-http-请求)
    - [发送 RPC 请求](#发送-rpc-请求)
    - [`--input` JSON 参数说明](#--input-json-参数说明)
    - [请求示例](#请求示例)
    - [返回示例](#返回示例)


## 服务与 IDL 相关命令

### 获取服务 IDL 设置

```bash
bitscli api-test \
  --act get-service-idl-setting \
  --vregion <VRegion> \
  --psm <psm>
```

返回示例：

```json
{
  "error_code": 0,
  "error_message": "",
  "data": {
    "idl_repo": "env/lane",
    "main_idl": "idl/lane_rpc.thrift",
    "idl_repo_branch": "master"
  }
}
```

返回字段说明：

| 字段 | 说明 |
|------|------|
| `error_code` | 错误码：0 成功，其他表示失败 |
| `error_message` | 错误信息 |
| `data.idl_repo` | IDL 仓库路径 |
| `data.main_idl` | 主 IDL 文件路径 |
| `data.idl_repo_branch` | IDL 仓库默认分支（兜底值） |

### 获取分支列表

```bash
bitscli api-test \
  --act get-branch-list \
  --vregion <VRegion> \
  --psm <psm> \
  --input '{
    "repo": "<idl_repo>",
    "page": 0,
    "count": 20,
    "query": "<IDL_BRANCH>"
  }'
```

返回示例：

```json
{
  "error_code": 0,
  "error_message": "",
  "data": {
    "branches": [
      {
        "name": "master",
        "commit": {
          "sha": "43a419996f8e5d287fcd8dd07447778bfc0c5651",
          "message": "commit message",
          "committer": { "name": "commiter name", "email": "commiter email" },
          "author": { "name": "author name", "email": "author email" }
        }
      }
    ],
    "total": 1,
    "message": ""
  }
}
```

返回字段说明：

| 字段 | 说明 |
|------|------|
| `error_code` | 错误码：0 成功，其他表示失败 |
| `error_message` | 错误信息 |
| `data.branches[].name` | IDL 仓库分支名称 |
| `data.branches[].commit` | 分支最新提交信息 |

### 查询服务接口列表

```bash
bitscli api-test \
  --act query-service-apis \
  --vregion <VRegion> \
  --psm <psm> \
  --input '{"idl_source": 1, "idl_version": "<IDL_BRANCH>"}'
```

HTTP 返回示例：

```json
{
  "error_code": 0,
  "data": [
    { "psm": "env.t.api", "method": "GET", "path": "/api/v1/rmq/consumer" }
  ]
}
```

RPC 返回示例：

```json
{
  "error_code": 0,
  "data": [
    { "psm": "env.t.api", "func_name": "GetLaneRmqConsumer" }
  ]
}
```

### 获取接口 Schema

#### HTTP

```bash
bitscli api-test \
  --act get-service-api-schema \
  --vregion <VRegion> \
  --psm <psm> \
  --input '{
    "http_method": "<METHOD>",
    "http_path": "<PATH>",
    "idl_version": "<IDL_BRANCH>",
    "test_plane": 1
  }'
```

HTTP 返回示例：

```json
{
  "error_code": 0,
  "data": {
    "request": {
      "psm": "env.t.api",
      "http_method": "GET",
      "http_path": "/api/v1/rmq/consumer",
      "http_req_headers": [],
      "http_query": [],
      "req_body": ""
    }
  }
}
```

#### RPC

```bash
bitscli api-test \
  --act get-service-api-schema \
  --vregion <VRegion> \
  --psm <psm> \
  --input '{
    "func_name": "<FUNC_NAME>",
    "idl_version": "<IDL_BRANCH>",
    "test_plane": 1
  }'
```

RPC 返回示例：

```json
{
  "error_code": 0,
  "data": {
    "request": {
      "psm": "env.t.api",
      "func_name": "GetLaneRmqConsumer",
      "req_body": ""
    }
  }
}
```

## 请求参数相关命令

### 获取推荐请求参数（历史流量）

#### HTTP

```bash
bitscli api-test \
  --act ai-recommend-api-test-history-traffic \
  --vregion <VRegion> \
  --psm <psm> \
  --input '{
    "protocol": "http",
    "http_method": "<METHOD>",
    "http_path": "<PATH>",
    "api_test_ctrl_type": 1
  }'
```

HTTP 返回示例：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "api_test_traffic": {
      "http_query": [],
      "http_req_headers": [],
      "req_body": "{}"
    },
    "api_test_history_id": 708547682
  }
}
```

#### RPC

> 注意：input 参数中必须是 `function_name`，不能是 `func_name`。

```bash
bitscli api-test \
  --act ai-recommend-api-test-history-traffic \
  --vregion <VRegion> \
  --psm <psm> \
  --input '{
    "protocol": "thrift",
    "function_name": "<FUNC_NAME>",
    "api_test_ctrl_type": 1
  }'
```

RPC 返回示例：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "api_test_traffic": { "req_body": "{}" },
    "api_test_history_id": 708547682
  }
}
```

## 发送请求相关命令

### 发送 HTTP 请求

#### 方式一：已知 HTTP Host

```bash
bitscli api-test \
  --act do-http-request-v5 \
  --vregion <VRegion> \
  --psm <psm> \
  --input '{
    "idl_version": "<IDL_BRANCH>",
    "env": "<ENV>",
    "http_method": "<METHOD>",
    "http_path": "<PATH>",
    "http_host": "<HTTP_HOST>",
    "zone": "<ZONE>",
    "idc": "<VDC>",
    "http_req_headers": <HTTP_REQUEST_HEADERS>,
    "http_query": <HTTP_QUERY>,
    "http_cookies": <HTTP_COOKIE>,
    "req_body": "<REQ_BODY>",
    "request_timeout": 60000
  }'
```

#### 方式二：HTTP Host 为空，使用集群/IDC 或 IPport 参数

```bash
bitscli api-test \
  --act do-http-request-v5 \
  --vregion <VRegion> \
  --psm <psm> \
  --input '{
    "http_method": "<METHOD>",
    "http_path": "<PATH>",
    "idl_version": "<IDL_BRANCH>",
    "address": "<IPport>",
    "env": "<ENV>",
    "zone": "<ZONE>",
    "idc": "<VDC>",
    "cluster": "<CLUSTER>",
    "http_req_headers": <HTTP_REQUEST_HEADERS>,
    "http_query": <HTTP_QUERY>,
    "http_cookies": <HTTP_COOKIE>,
    "req_body": "<REQ_BODY>",
    "request_timeout": 60000
  }'
```

### 发送 RPC 请求

```bash
bitscli api-test \
  --act do-rpc-request-v5 \
  --vregion <VRegion> \
  --psm <psm> \
  --input '{
    "func_name": "<FUNC_NAME>",
    "req_body": "<REQ_BODY>",
    "idl_version": "<IDL_BRANCH>",
    "address": "<IPport>",
    "zone": "<ZONE>",
    "idc": "<VDC>",
    "cluster": "<CLUSTER>",
    "env": "<ENV>",
    "rpc_context": "<RPC_CONTEXT>",
    "request_timeout": 60000,
    "connect_timeout": 60000
  }'
```

### `--input` JSON 参数说明

#### 通用必填参数（HTTP & RPC）

| 参数 | 变量名 | 说明 |
|------|--------|------|
| `idl_version` | `<IDL_BRANCH>` | IDL 分支名 |
| `env` | `<ENV>` | 泳道环境，如 `ppe_xxx`、`prod`，默认*必传*，若指定 `address` 可不传 |
| `zone` | `<ZONE>` | 区域标识，如 `CN`，从 VRegion 映射表推导并传递，**无论使用 IPport/address 还是集群/IDC 方式，均必传** |
| `idc` | `<VDC>` | 机房标识，如 `lf`，从 VRegion 映射表推导并传递，**无论使用 IPport/address 还是集群/IDC 方式，均必传** |
| `cluster` | `<CLUSTER>` | 集群名称，默认 `default`，*使用集群/IDC 方式测试时，必传* |

#### HTTP 必填参数

| 参数 | 变量名 | 说明 |
|------|--------|------|
| `http_method` | `<METHOD>` | HTTP 方法 |
| `http_path` | `<PATH>` | HTTP 路径 |

#### RPC 必填参数

| 参数 | 变量名 | 说明 |
|------|--------|------|
| `func_name` | `<FUNC_NAME>` | RPC 方法名 |

#### 通用可选参数（HTTP & RPC）

| 参数 | 变量名 | 说明 |
|------|--------|------|
| `address` | `<IPport>` | IPport 地址 |
| `req_body` | `<REQ_BODY>` | 请求体内容（JSON 字符串） |
| `request_timeout` | - | 请求超时时间（毫秒），默认 `60000` |

#### RPC 可选参数

| 参数 | 变量名 | 说明 |
|------|--------|------|
| `rpc_context` | `<RPC_CONTEXT>` | RPC 上下文，JSON 数组 |

#### HTTP 可选参数

| 参数 | 变量名 | 说明 |
|------|--------|------|
| `http_host` | `<HTTP_HOST>` | HttpURI，如 `https://boe-platform.bytedance.net` |
| `http_req_headers` | `<HTTP_REQUEST_HEADERS>` | 请求头 JSON 数组 |
| `http_cookies` | `<HTTP_COOKIE>` | Cookie JSON 数组 |
| `http_query` | `<HTTP_QUERY>` | 查询参数 JSON 数组 |

> 注：`--input` 中，非必填的参数如果值为空，则忽略无需传递。

### 请求示例

HTTP 请求示例：

```bash
bitscli api-test \
  --act do-http-request-v5 \
  --vregion boe \
  --psm env.t.api \
  --input '{
    "http_method": "POST",
    "http_path": "/api/v1/rmq/consumer",
    "http_host": "https://boe-platform.bytedance.net",
    "idl_version": "master",
    "env": "boe_test",
    "zone": "CN",
    "idc": "lf",
    "http_req_headers": [
      { "key": "Content-Type", "value": "application/json" }
    ],
    "http_query": [
      { "key": "id", "value": "412" }
    ],
    "req_body": "{\"name\":\"test\",\"status\":1}",
    "request_timeout": 60000
  }'
```

RPC 请求示例：

```bash
bitscli api-test \
  --act do-rpc-request-v5 \
  --vregion china-north \
  --psm env.t.rpc \
  --input '{
    "func_name": "SendLaneTce",
    "req_body": "{\"data\":\"123\",\"Base\":{\"Extra\":{\"env\":\"prod\"}}}",
    "rpc_context": [
      { "key": "KeyPersistent", "value": "1", "type": "persistent", "status": 0, "desc": "" },
      { "key": "KeyTransient", "value": "2", "type": "transient", "status": 0, "desc": "" }
    ],
    "idl_version": "master",
    "zone": "CN",
    "idc": "lf",
    "cluster": "default",
    "env": "prod"
  }'
```

### 返回示例

HTTP 返回示例：

```json
{
  "error_code": 0,
  "data": {
    "http_status_code": 200,
    "resp_headers": {
      "Content-Type": "application/json; charset=utf-8",
      "X-Tt-Logid": "202603231530481186BEFDDA8B983DD36A"
    },
    "resp_body": "{\"code\":\"SUCCESS\",\"message\":\"\",\"data\":{\"people_list\":[\"user1\",\"user2\"],\"department_list\":[\"dept1\",\"dept2\"]}}",
    "request_address": "boe-platform.bytedance.net",
    "log_id": "202603231530481186BEFDDA8B983DD36A",
    "req_latency": "8.072488ms",
    "protocol": "http",
    "history_id": 728497859,
    "psm": "inf.hae.boe",
    "func_name": "PostApiV4ToolTest",
    "online": true,
    "http_method": "POST",
    "http_path": "/api/v4/tool/test",
    "argos_link": "https://cloud.bytedance.net/argos/streamlog/info_overview/log_id_search?psm=inf.hae.boe&region=cn&logId=202603231530481186BEFDDA8B983DD36A",
    "biz_status_code": 0,
    "debug_info": { "tce_info": {}, "argos_info": {}, "bits_info": {} },
    "test_plane": 1
  },
  "has_permission": true
}
```

RPC 返回示例：

```json
{
  "error_code": 0,
  "data": {
    "http_status_code": 200,
    "resp_headers": {
      "Content-Type": "application/json; charset=utf-8",
      "X-Tt-Logid": "202603231530481186BEFDDA8B983DD36A"
    },
    "resp_body": "{\"code\":\"SUCCESS\",\"message\":\"\",\"data\":{\"people_list\":[\"user1\",\"user2\"],\"department_list\":[\"dept1\",\"dept2\"]}}",
    "request_address": "boe-platform.bytedance.net",
    "log_id": "202603231530481186BEFDDA8B983DD36A",
    "req_latency": "8.072488ms",
    "protocol": "rpc",
    "history_id": 728497859,
    "psm": "inf.hae.boe",
    "func_name": "PostApiV4ToolTest",
    "online": true,
    "argos_link": "https://cloud.bytedance.net/argos/streamlog/info_overview/log_id_search?psm=inf.hae.boe&region=cn&logId=202603231530481186BEFDDA8B983DD36A",
    "biz_status_code": 0,
    "debug_info": { "tce_info": {}, "argos_info": {}, "bits_info": {} },
    "test_plane": 1
  },
  "has_permission": true
}
```

无权限返回示例：

```json
{
  "error_code": 0,
  "escape_params": { },
  "has_permission": false,
  "permission_link": "https://permission.bytedance.net/apply?xxx"
}
```
