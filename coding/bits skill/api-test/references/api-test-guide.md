# 接口测试指引

本文档定义了 API 接口测试的详细测试流程。

## Table of Contents

- [Step 1：明确测试接口](#step-1明确测试接口)
- [Step 2：确认测试环境](#step-2确认测试环境)
  - [方式一：指定 IPport 测试](#方式一指定-ipport-测试)
  - [方式二：指定 ENV 泳道环境测试](#方式二指定-env-泳道环境测试)
- [Step 3：确认接口请求参数](#step-3确认接口请求参数)
  - [3.1 生成请求参数](#31-生成请求参数)
  - [3.2 修改请求参数](#32-修改请求参数)
- [Step 4：发送请求](#step-4发送请求)
  - [4.1 确认请求信息](#41-确认请求信息)
  - [4.2 发送请求](#42-发送请求)
- [Step 5：总结测试结果](#step-5总结测试结果)
  - [5.1 测试结果分析](#51-测试结果分析)
  - [5.2 结果总结](#52-结果总结)

## Step 1：明确测试接口

如果用户已明确指定要测试的接口，跳过此步骤
- http 服务需提供 HTTP method 和 path。
  - ⚠️ 用户仅提供了 path 但未指定 HTTP method 时，**不可跳过此步骤**，必须执行 `query-service-apis` 查询接口列表，从返回结果中匹配对应 path 的 method。**禁止假设默认 method（如 GET）。**
  - 若同一 path 匹配到多个 method（如 GET 和 POST），需将匹配结果展示给用户，请用户选择。
- rpc 服务需提供 RPC method。

否则，查询该服务的所有接口, `act` 必须为 `query-service-apis`：
```bash
bitscli api-test \
  --act query-service-apis \
  --vregion <VREGION> \
  --psm <psm> \
  --input '{"idl_source": 1, "idl_version": "<IDL_BRANCH>"}'
```

http 返回示例：
```json
{
  "error_code": 0,
  "data": [
    {
      "psm": "env.t.api",
      "method": "GET",
      "path": "/api/v1/rmq/consumer"
    }
  ]
}
```

rpc 返回示例：
```json
{
  "error_code": 0,
  "data": [
    {
      "psm": "env.t.api",
      "func_name": "GetLaneRmqConsumer"
    }
  ]
}
```

- 结合用户提供的接口信息，在接口列表中匹配查询到最符合的接口，作为被测接口。
- 如果有多个匹配度较高的接口，将接口列表展示给用户，请用户选择一个要测试的接口。

## Step 2：确认测试环境

### 方式一：指定 IPport 测试
若用户明确指定了 IPport、address 等相关信息，使用该信息进行测试。否则跳过此步骤。
- 结合用户提供的 IPport、address 信息，定义为 `<IPport>`
- 参考示例："[2605:340:cd50:2000:133c:ea0d:26a1:3e60]:9230"、"127.0.0.1:9230"。

### 方式二：指定 ENV 泳道环境测试
使用此方式测试，需要明确接口测试的泳道环境 `<ENV>`
- 若用户指定在生产环境、线上环境执行，则使用 `prod`
  - **安全提示**：首次在生产环境测试前，必须向用户确认安全风险：
    1. 该接口是否为只读操作（GET/查询类），写操作可能引发安全风险；
    2. 若仍需执行写入/修改/删除类操作，需确保使用测试账号等方式隔离或切换到测试环境，避免产生安全风险。
- 若用户需要在boe、测试环境、ppe、预览环境执行，或未作说明：
  - 需要**提示用户补充测试泳道环境信息**
  - 泳道环境有明确的规则，可按下面的表格进行校准：

| VREGION                               | 环境规则      | 示例                  |
| :------------------------------------ | :-------- | :------------------ |
| boe, boei18n                          | `boe_` 开头 | boe\_xxx, boe\_test |
| china-north, us, sg, i18n-tt, i18n-bd | `ppe_` 开头 | ppe\_xxx, ppe\_test |

## Step 3：确认接口请求参数

如果用户已提供完整的请求参数，跳过此步骤
- http 需提供 Headers、Query、Body。
- rpc 需提供 Body。

### 3.1 生成请求参数

默认必须要获取推荐的请求参数, `act` 必须为 `ai-recommend-api-test-history-traffic`：
http 获取推荐的请求参数:
```bash
bitscli api-test \
  --act ai-recommend-api-test-history-traffic \
  --vregion <VREGION> \
  --psm <psm> \
  --input '{
    "protocol": "<protocol>",
    "http_method": "<METHOD>",
    "http_path": "<PATH>",
    "api_test_ctrl_type": 1
}'
```

http 返回示例：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "api_test_traffic": {
      "http_query": [
        {
          "key": "psm",
          "value": "kiwis.llminfra.foundation"
        }
      ],
      "http_req_headers": [],
      "req_body": "{}"
    },
    "api_test_history_id": 708547682
  }
}
```
- 提取返回内容中的 `data.api_test_traffic.http_req_headers` 如果不为空，作为请求Headers参数`<HTTP_REQUEST_HEADERS>`。
- 提取返回内容中的 `data.api_test_traffic.http_query` 如果不为空，作为请求Query参数`<HTTP_QUERY>`。
- 提取返回内容中的 `data.api_test_traffic.req_body` 如果不为空，作为请求Body参数`<REQ_BODY>`。

rpc 获取推荐的请求参数:
```bash
bitscli api-test \
  --act ai-recommend-api-test-history-traffic \
  --vregion <VREGION> \
  --psm <psm> \
  --input '{
    "protocol": "<protocol>",
    "func_name": "<FUNC_NAME>",
    "api_test_ctrl_type": 1
}'
```

rpc 返回示例：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "api_test_traffic": {
      "req_body": "{}"
    },
    "api_test_history_id": 708547682
  }
}
```
- 提取返回内容中的 `data.api_test_traffic.req_body` 如果不为空，作为请求Body参数`<REQ_BODY>`。

如果推荐的请求参数获取失败或存在明显错误，则查询接口的请求 Schema, `act` 必须为 `get-service-api-schema`：
- http 查询命令：
```bash
bitscli api-test \
  --act get-service-api-schema \
  --vregion <VREGION> \
  --psm <psm> \
  --input '{
    "http_method": "<METHOD>",
    "http_path": "<PATH>",
    "idl_source": 1,
    "idl_version": "<IDL_BRANCH>",
    "test_plane": 1
}'
```

- http 返回示例：
```json
{
  "error_code": 0,
  "data": {
    "request": {
      "psm": "env.t.api",
      "http_method": "GET",
      "http_path": "/api/v1/rmq/consumer",
      "http_req_headers": [],
      "http_query": [
        {
          "key": "id",
          "value": "412"
        }
      ],
      "req_body": ""
    }
  }
}
```
- 提取返回内容中的 `data.request.http_req_headers` 如果不为空，作为请求Headers参数`<HTTP_REQUEST_HEADERS>`。
- 提取返回内容中的 `data.request.http_query` 如果不为空，作为请求Query参数`<HTTP_QUERY>`。
- 提取返回内容中的 `data.request.req_body` 如果不为空，作为请求Body参数`<REQ_BODY>`。

- rpc 查询命令：
```bash
bitscli api-test \
  --act get-service-api-schema \
  --vregion <VREGION> \
  --psm <psm> \
  --input '{
    "func_name": "<FUNC_NAME>",
    "idl_source": 1,
    "idl_version": "<IDL_BRANCH>",
    "test_plane": 1
}'
```

- rpc 返回示例：
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
- 提取返回内容中的 `data.request.req_body` 如果不为空，作为请求Body参数`<REQ_BODY>`。


### 3.2 修改请求参数

根据提取到的请求参数，分析测试意图并辅助用户进行参数修改：

- 如果上下文中**用户有明确的参数说明**，则直接进行参数修改或替换。
- 如果请求参数中涉及需要**特殊构造的** **`id`** **类参数**，与用户沟通确认，例如：user\_id、item\_id、room\_id、group\_id、order\_id 等。
- 如果请求参数中涉及一些**关键参数**，与用户沟通确认，例如：枚举值、布尔值等。

## Step 4：发送请求

根据确认好的接口请求信息，发送接口请求。

### 4.1 确认请求信息

首次发送前，必须先向用户展示完整的请求信息，提示用户确认，参考以下要求：
- 展示内容尽量精简、有效，为空的内容无需展示给用户。
- Headers、Parameters、Body等数据需要格式化展示，若内容过多，可只展示关键信息，并提示说明。
- 若用户使用 IPport 方式测试时，ENV 信息展示为固定值 “IPport 直连”。
具体可选内容包括：
- **PSM**：`<PSM>`
- **HTTP Method**：`<METHOD>`
- **HTTP Path**：`<PATH>`
- **Func Name**：`<FUNC_NAME>`
- **VRegion**：`<VREGION>`
- **ENV**：`<ENV>`
- **HTTP Host**：`<HTTP_HOST>`
- **Headers**: `<HTTP_REQUEST_HEADERS>`
- **Query Parameters**: `<HTTP_QUERY>`
- **Request Body**: `<REQ_BODY>`

然后提供给用户 3 个选择
- 确认执行
- 确认执行，后续无参数变化可自动执行
- 补充修改建议
  - 对话接收用户的参数补充信息，进一步调整并再次确认

完成确认后，执行后续步骤。

### 4.2 发送请求

#### 发送 http 请求
##### 方式一：已知 HTTP Host
当用户指定了目标 HTTP Host 时，发送请求：

```bash
bitscli api-test \
  --act do-http-request-v5 \
  --vregion <VREGION> \
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

##### 方式二： HTTP Host为空, 使用集群/IDC 或 IPPort 参数

使用以下命令发送请求：

```bash
bitscli api-test \
  --act do-http-request-v5 \
  --vregion <VREGION> \
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
#### 发送 rpc 请求：使用集群/IDC 或 IPPort 参数
执行命令发送 RPC 请求：
```bash
bitscli api-test \
  --act do-rpc-request-v5 \
  --vregion <VREGION> \
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
#### `--input` JSON 参数说明

##### 通用必填参数（HTTP & RPC）

| 参数 | 变量名 | 说明 |
| --- | --- | --- |
| `idl_version` | `<IDL_BRANCH>` | IDL 分支名 |
| `env` | `<ENV>` | 泳道环境，如 `ppe_xxx`、`prod`，⚠️默认*必传*，若指定 `address` 可不传 |
| `zone` | `<ZONE>` | 区域标识，如 `CN`，从 VRegion 映射表推导并传递，⚠️**无论使用 IPport/address 还是集群/IDC 方式，均必传** |
| `idc` | `<VDC>` | 机房标识，如 `lf`，从 VRegion 映射表推导并传递，⚠️**无论使用 IPport/address 还是集群/IDC 方式，均必传** |
| `cluster` | `<CLUSTER>` | 集群名称，默认 `default`，*使用集群/IDC方式测试时，必传⚠️* |

##### HTTP 必填参数

| 参数 | 变量名 | 说明 |
| --- | --- | --- |
| `http_method` | `<METHOD>` | HTTP 方法 |
| `http_path` | `<PATH>` | HTTP 路径 |

##### RPC 必填参数

| 参数 | 变量名 | 说明 |
| --- | --- | --- |
| `func_name` | `<FUNC_NAME>` | RPC 方法名 |

##### 通用可选参数（HTTP & RPC）

| 参数 | 变量名 | 说明 |
| --- | --- | --- |
| `address` | `<IPport>` | IPport 地址 |
| `req_body` | `<REQ_BODY>` | 请求体内容（JSON 字符串） |
| `request_timeout` | - | 请求超时时间（毫秒），默认 `60000` |

##### RPC 可选参数

| 参数 | 变量名 | 说明 |
| --- | --- | --- |
| `rpc_context` | `<RPC_CONTEXT>` | RPC 上下文，JSON 数组 |

##### HTTP 可选参数

| 参数 | 变量名 | 说明 |
| --- | --- | --- |
| `http_host` | `<HTTP_HOST>` | HttpURI，如 `https://boe-platform.bytedance.net` |
| `http_req_headers` | `<HTTP_REQUEST_HEADERS>` | 请求头 JSON 数组 |
| `http_cookies` | `<HTTP_COOKIE>` | Cookie JSON 数组 |
| `http_query` | `<HTTP_QUERY>` | 查询参数 JSON 数组 |

- 注：`--input`中，非必填的参数如果值为空，则忽略无需传递。
#### 请求示例
http 请求示例：

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
        {
            "key": "Content-Type",
            "value": "application/json"
        }
    ],
    "http_query": [
        {
            "key": "id",
            "value": "412"
        }
    ],
    "req_body": "{\"name\":\"test\",\"status\":1}",
    "request_timeout": 60000
}'
```

rpc 请求示例：

```bash
bitscli api-test \
  --act do-rpc-request-v5 \
  --vregion china-north \
  --psm env.t.rpc \
  --input '{
    "func_name": "SendLaneTce",
    "req_body": "{\"data\":\"123\",\"Base\":{\"Extra\":{\"env\":\"prod\"}}}",
    "rpc_context": [
        {
            "key": "KeyPersistent",
            "value": "1",
            "type": "persistent",
            "status": 0,
            "desc": ""
        },
        {
            "key": "KeyTransient",
            "value": "2",
            "type": "transient",
            "status": 0,
            "desc": ""
        }
    ],
    "idl_version": "master",
    "zone": "CN",
    "idc": "lf",
    "cluster": "default",
    "env": "prod"
}'
```

#### 返回示例
http 返回示例：
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
    "debug_info": {
      "tce_info": {},
      "argos_info": {},
      "bits_info": {}
    },
    "test_plane": 1
  },
  "has_permission": true
}
```

rpc 返回示例：
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
    "debug_info": {
      "tce_info": {},
      "argos_info": {},
      "bits_info": {}
    },
    "test_plane": 1
  },
  "has_permission": true
}
```
无权限返回示例：
```json
{
  "error_code": 0,
  "escape_params": { ... },
  "has_permission": false,
  "permission_link": "https://permission.bytedance.net/apply?xxx"
}
```

#### 无权限处理

请求返回后，若 `has_permission` 为 `false` 且 `permission_link` 不为空，说明当前用户无该接口的测试权限：
- 向用户展示权限审批链接 `permission_link`，提示用户点击申请权限。
- **终止后续测试流程**，不再执行 Step 5 的结果分析与总结。

## Step 5：总结测试结果

请求完成后，自动分析执行结果，给出最终测试结论。

### 5.1 测试结果分析

默认必须要执行接口测试结果分析。
- 若用户明确表示不需要分析，则直接跳过。
- 若接口请求成功，且没有明显的异常数据表现（响应为空、存在 error 信息等），则可以灵活选择跳过。

执行结果分析时，先提取响应结果中的 `logid`，然后调用以下命令：
```bash
bitscli api-test \
  --act analyze-result \
  --psm <psm> \
  --vregion <VREGION> \
  --input '{"logid": "<LOG_ID>"}'
```

- 该能力仅支持部分 <VREGION> ：i18n-tt、china-north、boe、boei18n，若不支持可直接跳过。

返回示例（SSE 流式响应）：

```
event:create_progress
data:{"content": "    a. 分析执行状态：执行成功"}

event:create_progress
data:{"content": "    b. 分析业务逻辑潜在问题"}

event:message_chunk
data:{"content": "【问题分析结论】<br>"}

event:message_chunk
data:{"content": "1. 执行状态：执行成功且无逻辑错误<br>"}

event:create_progress
data:{"content":"【问题分析】完成"}

event:create_progress
data:{"content":"【决策】思考下一步动作..."}

event:create_progress
data:{"content":"【决策】处理完成，终止"}
```

返回内容是流式响应，需要组装到一起，示例如下：

````
【决策】思考下一步动作...【决策】正在回复用户...为了分析指定LogID对应的接口测试记录，需要调用对应智能体来查询并分析该LogID的测试结果和相关日志数据。\u003cbr\u003e【决策】准备调用：问题分析【问题分析】开始【分析过程】1.获取接口调试数据：成功2.获取分析数据a.获取接口IDL数据：失败，失败原因【查询当前版本接口列表失败】b.获取代码CallGraph链路、执行程序分析：成功c.获取日志：成功d.获取接口调试覆盖的代码：失败，失败原因为【可能未接入代码覆盖率平台或代码版本未插桩】3.分析接口潜在问题a.分析执行状态：服务端报错【问题分析结论】\u003cbr\u003e1.错误类型：服务端报错\u003cbr\u003eb.分析执行日志，定位关键信息c.分析接口代码，定位关键方法d.分析服务端报错根因2.错误根因：根因：数据库中replay_case表不存在id=123的记录，GetByID返回gorm.ErrRecordNotFound，被包装成500“recordnotfound”返回。\u003cbr\u003e\n```\nSELECT*FRO\n```\n\u003cbr\u003e【问题分析】完成【决策】思考下一步动作...【决策】处理完成，终止
````

只需要提取“【问题分析结论】”后面的关键内容即可，作为测试问题分析的结果。

### 5.2 结果总结

上文的测试问题分析结果不一定完全准确，需要结合上下文、结合代码等信息综合分析，给出最终测试结果:
- 重点关注发现的测试问题，涵盖错误原因（具体到有缺陷的代码）、严重程度、处理建议等。
- 输出关键信息，包括LogID、响应状态码、响应时间、关键响应参数、关键日志等。
