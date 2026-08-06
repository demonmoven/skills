# 接口测试流程

本文档定义 API 接口测试的执行流程。所有 `bitscli api-test` 命令模板、入参说明与返回示例统一查阅 [cli-reference.md](../refs/cli-reference.md)，本文档只描述「何时调哪个命令、如何提参与决策」，不再内联命令块。

Step 5 下的子步骤指标上报是强制项。子步骤统一使用 [data-reference.md](../refs/data-reference.md#step-id-对照表) 中的 `S5_1_SELECT_API`、`S5_2_BUILD_REQUEST`、`S5_3_SEND_REQUEST`、`S5_4_SUMMARY`；相关字段仅允许写入 `report_info.customs`。

- 进入每个子步骤前，必须先执行对应的 `step-start` 记录本地开始时间；该命令本身不发起上报。
- 子步骤完成后，必须执行对应的 `step-finish --result success`；真正的步骤上报只在 `step-finish` 发生。
- 子步骤若被跳过，必须执行 `step-finish --result skipped --skip-reason <REASON>`。
- 子步骤执行失败时，必须执行 `step-finish --result failed --error-type <TYPE>`。

## 目录
- [接口测试流程](#接口测试流程)
  - [目录](#目录)
  - [Step 1：明确测试接口](#step-1明确测试接口)
  - [Step 2：确认接口请求参数](#step-2确认接口请求参数)
    - [2.1 生成请求参数](#21-生成请求参数)
    - [2.2 修改请求参数](#22-修改请求参数)
  - [Step 3：发送请求](#step-3发送请求)
    - [3.1 确认请求信息](#31-确认请求信息)
    - [3.2 发送请求](#32-发送请求)
    - [3.3 无权限处理](#33-无权限处理)
  - [Step 4：总结测试结果](#step-4总结测试结果)
    - [4.1 测试结果分析诊断](#41-测试结果分析诊断)
    - [4.2 结果总结](#42-结果总结)

## Step 1：明确测试接口

如果用户已明确指定要测试的接口，跳过此步骤。

- HTTP 服务需提供 HTTP method 和 path。
  - ⚠️ 用户仅提供了 path 但未指定 HTTP method 时，**不可跳过此步骤**，必须执行 `query-service-apis` 查询接口列表，从返回结果中匹配对应 path 的 method。**禁止假设默认 method（如 GET）。**
  - 若同一 path 匹配到多个 method（如 GET 和 POST），需将匹配结果展示给用户，请用户选择。
- RPC 服务需提供 RPC method。

否则，查询该服务的所有接口（命令参考 [cli-reference.md](../refs/cli-reference.md#查询服务接口列表)）：

- 结合用户提供的接口信息，在接口列表中匹配查询到最符合的接口，作为被测接口。
- 如果有多个匹配度较高的接口，将接口列表展示给用户，请用户选择要测试的接口。

## Step 2：确认接口请求参数

如果用户已提供完整的请求参数，跳过此步骤。

- HTTP 需提供 Headers、Query、Body。
- RPC 需提供 Body。

### 2.1 生成请求参数

默认必须要获取推荐的请求参数（命令参考 [cli-reference.md](../refs/cli-reference.md#获取推荐请求参数历史流量)）：

- HTTP：提取返回内容中的 `data.api_test_traffic.http_req_headers` 如果不为空，作为请求 Headers 参数 `<HTTP_REQUEST_HEADERS>`；提取 `data.api_test_traffic.http_query` 作为 Query 参数 `<HTTP_QUERY>`；提取 `data.api_test_traffic.req_body` 作为 Body 参数 `<REQ_BODY>`。
- RPC：提取返回内容中的 `data.api_test_traffic.req_body` 如果不为空，作为请求 Body 参数 `<REQ_BODY>`。

如果推荐的请求参数获取失败或存在明显错误，则查询接口的请求 Schema（命令参考 [cli-reference.md](../refs/cli-reference.md#获取接口-schema)）：

- HTTP：提取返回内容中的 `data.request.http_req_headers` 作为 `<HTTP_REQUEST_HEADERS>`，`data.request.http_query` 作为 `<HTTP_QUERY>`，`data.request.req_body` 作为 `<REQ_BODY>`。
- RPC：提取返回内容中的 `data.request.req_body` 作为 `<REQ_BODY>`。

### 2.2 修改请求参数

根据提取到的请求参数，分析测试意图，辅助用户修改参数：

- 如果上下文中**用户有明确的参数说明**，则直接进行参数修改或替换。
- 如果请求参数中涉及需要**特殊构造的 `id` 类参数**，与用户沟通确认，例如：user_id、item_id、room_id、group_id、order_id 等。
- 如果请求参数中涉及一些**关键参数**，与用户沟通确认，例如：枚举值、布尔值等。

## Step 3：发送请求

根据确认好的接口请求信息，发送接口请求。

### 3.1 确认请求信息

首次发送前，必须先向用户展示完整的测试环境信息、接口信息、请求参数信息，提示用户确认，参考以下要求：

- 表格形式、格式化展示内容，精简；可选信息为空时不展示。
- 若用户使用 IPport 方式测试时，ENV 信息展示为固定值 "IPport 直连"。

必展示信息包括：

- **PSM**：`<psm>`
- **Protocol**：`<protocol>`
- **IDL Branch**：`<IDL_BRANCH>`
- **VRegion**：`<VRegion>`
- **ENV 或 IPport**：`<ENV>` 或 `<IPport>`

可选展示信息包括：

- **HTTP Method**：`<METHOD>`
- **HTTP Path**：`<PATH>`
- **Func Name**：`<FUNC_NAME>`
- **HTTP Host**：`<HTTP_HOST>`
- **Headers**：`<HTTP_REQUEST_HEADERS>`
- **Query Parameters**：`<HTTP_QUERY>`
- **Request Body**：`<REQ_BODY>`

然后提供给用户 3 个选择：

- 确认执行
- 确认执行，后续无参数变化可自动执行
- 补充修改建议
  - 对话接收用户对参数的修改和补充，进一步调整并再次确认

完成确认后，执行后续步骤。

### 3.2 发送请求

命令模板与参数说明见 [cli-reference.md](../refs/cli-reference.md#发送请求相关命令)：

- HTTP 方式一（已知 HTTP Host）：`do-http-request-v5`，带 `http_host`。
- HTTP 方式二（Host 为空，使用集群/IDC 或 IPport）：`do-http-request-v5`，带 `address`/`cluster`。
- RPC：`do-rpc-request-v5`。

### 3.3 无权限处理

详细无权限处理请参考 [runtime-support.md](../support/runtime-support.md#无权限处理)。

## Step 4：总结测试结果

本步骤用于在请求完成后，基于本次测试结果输出摘要，必要时执行测试结果分析与诊断。

### 4.1 测试结果分析诊断

执行规则如下：

- 完成本 Skill 的基础测试后：
  - 如果测试结果都成功，则默认不执行测试结果分析诊断
  - 如果测试结果有失败情况，则对失败情况执行测试结果分析诊断
- 若缺少 `logid`、环境、IDL、日志等关键证据，应明确说明数据缺口；但只要用户明确需要，仍可基于现有结果继续做低置信度诊断。
- 执行结果分析诊断时，必须按 [doctor-flow.md](./doctor-flow.md) 的 `标准流程` 执行：
  - 先整理 `interface_context`、`test_context`、`env_context`、`log_context`、`code_context`
  - 先读取原始测试结果中的现象证据，再补充结构证据、环境证据和变化证据
  - 若已有 `logid`、关键日志或 Trace，可作为补充证据使用；没有则必须如实标记缺失
  - 按 `根因分类框架` 完成归类，并按 `推荐报告结构` 输出结果

### 4.2 结果总结

无论是否执行 4.1，**必须基于测试请求和测试结果给出总结**。

- 总结内容包括 3 部分：测试服务和测试环境信息，接口 + 请求 + 响应信息，测试结果诊断（如果执行了 4.1 测试结果分析）信息。
- 表格形式、格式化展示必要但精简的内容；可选信息为空时不展示。
- 测试服务相关信息合并展示一行
- 测试环境相关信息合并展示一行
- 多个接口测试时，每个接口以单独的列表展示 被测接口 + 请求 + 响应信息，和测试结果诊断信息

测试服务信息：

- **PSM**：`<psm>`
- **Protocol**：`<protocol>`
- **IDL Branch**：`<IDL_BRANCH>`

测试环境信息：

- **VRegion**：`<VRegion>`
- **ENV 或 IPport**：`<ENV>` 或 `<IPport>`

被测接口：

- **接口**：http 接口则拼接 `<METHOD>` `<PATH>`，rpc 接口则为 `<FUNC_NAME>`

请求信息：

- **Query 参数**：`<HTTP_QUERY>` 当 http 接口且不为空时展示
- **Request Body**：`<REQ_BODY>`

响应信息：

- **响应状态码**：`<HTTP_STATUS_CODE>` 当 http 接口时展示
- **业务码**：`<biz_status_code>` 需要从响应体中提取
- **LogId**：`logid` 如果返回 `logid` 则展示，否则为空
- **响应体**：`<RESP_BODY>` 如果内容长度超过 200 个字符，则折叠隐藏超出长度的内容
- **错误信息**：如果响应中包含错误信息，则展示，否则为空

测试结果诊断信息：

- 展示 4.1 测试结果分析诊断 输出的内容
