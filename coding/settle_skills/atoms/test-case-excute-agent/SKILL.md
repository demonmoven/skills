---
name: "test-case-excute-agent"
description: "通过异步接口批量发起测试用例，轮询任务状态直至完成，然后生成详细的 Markdown 格式测试报告，并支持将报告复制到飞书文档。"
---

## 概述

本 Skill 用于实现批量测试用例的自动化执行与报告生成流程。它围绕两个核心 API（任务发起与状态查询）构建，提供了一个命令行的 Python 脚本入口，能够：

1.  **异步发起**：通过 `POST /api/v4/bots/chat/completions` 接口提交批量测试用例，获取一个唯一的 `task_id`。
2.  **状态轮询**：使用 `GET /api/v3/bots/task/status` 接口，根据 `task_id` 定期查询任务状态，直至任务成功、失败或超时。
3.  **报告生成**：任务完成后，将返回的 JSON 结果格式化为一份结构清晰、风格统一的 Markdown 报告。
4.  **归档分享**：支持将生成的 Markdown 报告内容一键复制到新的飞书文档中，便于团队归档和分享。

## 参数说明

本 Skill 的核心是 `scripts/async_case_runner.py` 脚本。其参数严格区分为两类：**接口请求参数**（最终会进入提交任务的 POST 请求体）和**脚本运行参数**（仅用于控制脚本本身的行为，不会进入请求体）。

### 接口请求参数

这些参数直接映射到 `POST /api/v4/bots/chat/completions` 接口的请求体 `NewChatCompletionsRequest` 结构。

| 字段名 | 类型 | 是否必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `messages` | `string[]` | 否 | 测试用例列表。当 `bits_url` 未提供时，此字段为必填项之一。 |
| `user` | `string` | **是** | 用户邮箱前缀（例如 `yangchen.shine`）。此字段为必填项，用于后端服务换取用户 token 以执行操作，并记录任务触发人。 |
| `bits_url` | `string` | 否 | Bits 用例树的长链接。若提供，服务端将基于此链接自动转化生成 `messages`。<br> **强校验规则**：脚本会校验 URL 中是否包含 `projectId` 查询参数及路径中能否解析出 `caseDetail/{id}`。 |
| `env` | `string` | 否 | 执行环境（如 `prod` 或 `boe_xxx`）。若不传，则由服务端按其内部逻辑（如从 `messages` 解析 `boe_*` 或默认 `prod`）确定。 |

### 脚本运行参数

这些参数用于配置脚本的本地行为，例如指定输入源、轮询策略等，它们 **不会** 被包含在发送到服务端的请求体中。服务地址已在脚本中固定为 `https://e9r97mvz.fn.bytedance.net`，无需也不支持通过参数修改。

| 参数名 | 类型 | 默认值 | 说明 |
| :--- | :--- | :--- | :--- |
| `--user` | `string` | (无) | **(必填)** 用户邮箱前缀。脚本启动时会检查此参数，若缺失则报错退出。此值会填充到请求体的 `user` 字段。 |
| `--messages-file` | `string` | (无) | 包含用例消息列表的 JSON 文件路径。文件的内容必须是一个合法的 JSON 字符串数组，数组中的每一行为一个测试用例|
| `--bits-url` | `string` | (无) | Bits 用例的长链接（必须包含project_id）。提供此参数后，脚本会将其填入请求体的 `bits_url` 字段，并由服务端处理。 |
| `--env` | `string` | (无) | 指定执行环境。若提供，脚本会将其填入请求体的 `env` 字段。 |
| `--poll-interval` | `int` | `5` | 轮询任务状态接口的时间间隔（秒）。 |
| `--max-wait` | `int` | `1200` | 轮询任务的最大等待时间（秒）。超过此时长后，任务将被标记为“超时”。 |
| `--report-dir` | `string` | `reports/` | 生成的报告文件和执行摘要 JSON 的输出目录。 |

**输入源优先级规则：**

脚本通过以下优先级顺序确定最终的用例输入：
1.  **`--bits-url` (最高)**：如果提供了合法的 `--bits-url`，则忽略所有本地 `messages` 输入，由服务端处理。
2.  **`--messages-file` **：如果 `--bits-url` 不合法或未提供，则尝试从 `--messages-file` 指定的文件中读取。

**--messages-file中单条用例json字符串样例**：
"{\"case_id\":\"caseID_1778769343570_0008\",\"title\":\"（P1）微信风控换单：DAO 使用 nil DB 句柄仍可完成查询与换单约束校验\",\"preconditions\":\"存在可触发换单流程的指令记录（status=init 且满足允许换单的业务前置）；相关表：split_order_record、settle_detail、change_order_no_record 中有可查询数据；\",\"step\":\"1. 触发 ChangeInstructionNoService.CheckInstructionStatus/CheckInterval/QueryDetailsAndSplitSin 全链路；2. 观察查询是否成功、是否出现 db nil 引发的异常；\",\"expected_results\":[\"1. 所有查询正常返回（不因 db 入参为 nil 导致 panic 或 DB 为空不可用）；2. 换单间隔校验仍生效（24 小时限制等）；3. 子单/明细数据能被正确加载用于后续判断；\"]}"

**注意**：
1. 必须至少提供 `--bits-url`、`--messages-file` 中的一种。
2. 如果bits-url中不存在project_id则禁止使用bits-url的方式执行，必须使用messages-file的传参方式。
3. 对于已有的json结构化用例对象数组文件，则需要将其内容逐条转义为符合条件的json字符串，写入到一个新的JSON字符串数组文件中（适配数字人），除了转换以外，禁止对原始信息进行任何修改。

## 执行流程

1.  **参数解析与校验**：脚本启动，解析所有命令行参数。
    -   `--user` 必须提供。
    -   `--bits-url` (如果提供) 必须通过 `projectId` 和 `caseDetailId` 的校验。
    -   至少需要一个有效的输入源。
2.  **构造请求体**：根据输入参数，严格按照 **接口请求参数** 的定义构建 JSON 请求体。
3.  **发起任务**：向 `POST https://e9r97mvz.fn.bytedance.net/api/v4/bots/chat/completions` 发送请求。
    -   成功后，从响应中提取 `task_id`。
    -   若请求失败，将进行最多 3 次内部重试。若最终失败，则生成错误报告并退出。
4.  **轮询状态**：进入轮询循环，每隔 `--poll-interval` 秒向 `GET https://e9r97mvz.fn.bytedance.net/api/v3/bots/task/status?task_id={task_id}` 发送请求。
    -   循环将持续，直到任务状态变为 `success`, `failed`, `timeout`，或等待时间超过 `--max-wait`。
5.  **结果解析与报告生成**：
    -   任务成功后，脚本会稳健地解析响应 `data` 字段，提取 `batchResult`。
    -   调用 `scripts/report_md_formatter.py` 将结果转换为 Markdown 报告。
    -   报告将保存在 `--report-dir` 指定的目录中。
6.  **输出总结**：
    -   向标准输出打印一个 JSON 对象，包含 `task_id`、最终状态 `status` 以及报告相对路径 `report_md`。
    -   该 JSON 对象也会被保存到 `--report-dir` 目录下的 `summary_*.json` 文件中。
7.  **归档至飞书**：产出的md文档直接copy到飞书文档，适当调整一下涉及的表格的列宽使得超长的工具输入输出看起来更美观后即为最终报告，严禁对报告进行二次加工和总结。
8.  **适配数字人**：将原始返回信息进一步按照execution_report_demo.md的格式进行标准化

## 标准化执行记录 JSON 产出

在完成批量任务执行并成功解析出 `batchResult` 后，脚本会默认追加生成一份标准化执行记录 JSON 文件，用于后续结构化归档与自动化处理。

- **生成开关**：
  - 通过 `--emit-standard-json` 控制是否生成标准化执行记录 JSON，默认开启。
  - 如需关闭，可传入 `--no-emit-standard-json` 显式关闭，不影响 Markdown 报告生成。
- **默认路径与命名**：
  - 当未指定 `--standard-json-path` 时，文件将输出到 `--report-dir` 目录下：`execution_records_{task_id}.json`；
  - 若任务 ID 不可用，则退化为 `execution_records_{timestamp}.json`。
- **字段结构**：产物为数组，每个元素对应一条用例完整执行记录，结构如下：

  ```json
  [
    {
      "case_id": "...",
      "module": "...",
      "title": "...",
      "steps": [
        { "index": 1, "tool_name": "...", "description": "..." }
      ],
      "tool_exec_result": [
        {
          "tool_name": "...",
          "tool_url": "",
          "tool_input": "...",
          "tool_output": "...",
          "description": "...",
          "exec_status": "...",
          "log_id": "..."
        }
      ]
    }
  ]
  ```

- **字段提取规则（简要）**：
  - `case_id / module / title`：优先将 `test_case_str` 作为 JSON 解析读取显式键；若不存在，则按如下顺序回退：
    - `case_id`：在原始文本中匹配 `case_id` 或形如 `caseID_***` 的片段，取首个命中；
    - `module`：匹配 JSON 中的 `module` / `模块`，若缺失则尝试从 `goal` 文本中提取方括号、书名号等包裹的模块提示；
    - `title`：匹配 JSON 中的 `title` / `标题`，若缺失则优先取 `goal` 文本中的第一个枚举项（如以 `1、`、`1.`、`(1)`、`（1）` 开头的条目）。
  - `steps`：优先复制 `final_plan.steps`，若不存在则回退到 `initial_plan.steps`，均缺失时为空数组。
  - `tool_exec_result`：遍历 `step_records`，按以下规则映射：
    - `tool_name`：取 `工具名称` / `tool_name`；
    - `tool_url`：固定为空字符串；
    - `tool_input`：取 `工具执行输入` / `tool_input`；
    - `tool_output`：取 `工具执行输出` / `tool_output`；
    - `description`：优先取 `reason`，其次取 `description`；
    - `exec_status`：优先取 `expected_success`；
    - `log_id`：从 `tool_output` 中正则提取，支持：
      - 对嵌套 JSON 进行一到多次解码，读取 `logid` 字段；
      - 文本中包含 `Logid` / `logid` / `LOGID` / `重试logid` 后接中英文冒号与编号；
      - 混合字符串中只要出现 `logid` 关键字，即会捕获其后的连续字母数字编号。

- **与 Markdown 报告关系**：
  - Markdown 报告仍是人工阅读与飞书复制的主产物；
  - 标准化执行记录 JSON 作为结构化归档产物，与 Markdown 报告并存，不改变原有报告生成与使用方式；
  - `summary_*.json` 与标准输出到控制台的 JSON 对象会新增 `standard_json` 字段，值为上述 JSON 文件的相对路径，便于上游编排能力直接消费。

### 状态查询返回结构与解析约定

状态查询接口返回结构为：

- `TaskResponse { code, message, data }`
- 其中 `data` 为单条任务记录，对应后端的 `AgentTask` 结构。

`AgentTask` 中与本 Skill 解析强相关的字段如下：

| 字段名 | 类型 | 说明 | 兼容键名 |
| :--- | :--- | :--- | :--- |
| `task_id` | `string` | 任务 ID，用于唯一标识一次批量执行任务。 | `task_id` |
| `task_status` | `string` | 任务状态，例如 `success` / `failed` / `timeout` / `processing`。 | `task_status`、`status` |
| `error_msg` | `string` | 任务级错误信息，例如批量执行失败原因、内部异常等。 | `error_msg`、`error`、`error_message` |
| `result` | `object` / `string` | 任务最终执行结果，期望为批量执行结果 `batchResult`。 | `result`、`task_result`、`Result`、`taskResult` |

脚本在解析状态查询返回时的约定如下：

- 优先从 `task_status` 读取任务状态，无法获取时再回退到 `status`，并统一规范为 `success` / `failed` / `timeout` / `processing` 四类之一。
- 任务错误信息优先从 `error_msg` 读取，若不存在则回退到 `error` 与 `error_message`。
- 结果字段会按顺序尝试：`result`、`task_result`、`Result`、`taskResult`：
  - 当字段为 `dict` 且包含 `total_cases` 与 `case_results` 时，视为标准的 `batchResult` 结构。
  - 当字段为字符串时，会尝试按 JSON 解析；若解析成功且为 `batchResult`，亦视为合法结果；若解析失败，则记录 `parse_error`，并在最终报告顶部展示解析失败原因。
- 当 `data` 本体就是一个包含 `total_cases` 与 `case_results` 的对象时，会直接将其视为 `batchResult`，跳过上述 `result` 字段解析逻辑，用于兼容旧版或特殊接入方式。

整体判断规则为：

- 当任务状态最终为 `success` 且成功解析出包含 `total_cases` / `case_results` 的 `batchResult` 时，进入报告生成流程。
- 若任务状态为 `success` 但缺少可用的 `batchResult`，或结果字段 JSON 解析失败，则整体视为失败：
  - 最终状态会被归一为 `failed`。
  - 失败原因会写入报告顶部（包括解析错误 `parse_error`、后端返回的 `error_msg` 等）。

## 注意事项

-   **接口契约**：本 Skill 严格对齐 `NewChatCompletionsRequest` 请求体和 `TaskResponse` 响应结构。如果后端接口发生变更，需同步更新 `scripts/async_case_runner.py`。
-   **服务地址固定**：服务地址已在脚本中固定为 `https://e9r97mvz.fn.bytedance.net`，无需也不支持通过参数传入或修改。
-   **配置**：`--poll-interval` 和 `--max-wait` 应根据实际业务场景的平均耗时进行合理配置。
-   **报告产出**：产出的md文档直接copy到飞书文档调整下表格列宽即为最终报告，严禁对报告进行二次加工和总结。
