# Log Explore — 完整参考

日志明细查询的完整参数说明。通用参数（bid、env、时间范围）和 ev_type 速查表见父级 SKILL.md。

## Contents

- log query — 日志明细查询（参数表、columns、filter、场景示例）
- log session — Session 分析
- log detail — 单条日志详情
- log columns — 获取可用列
- log filters — 获取可筛选字段
- 注意事项

---

## log query — 日志明细查询

### 完整参数表

| 参数           | 必须 | 默认值    | 说明                                                                                                               |
| -------------- | ---- | --------- | ------------------------------------------------------------------------------------------------------------------ |
| `--bid`        | 是   | -         | 业务 ID                                                                                                            |
| `--ev-type`    | 是   | -         | 事件类型（js_error/http/view/session/action/resource/resource_error/custom/log/performance_longtask/blank_screen） |
| `--start-time` | 否   | 最近1小时 | 开始时间戳（秒）                                                                                                   |
| `--end-time`   | 否   | 当前时间  | 结束时间戳（秒）                                                                                                   |
| `--columns`    | 否   | 自动获取  | 查询列名，逗号分隔                                                                                                 |
| `--filter`     | 否   | -         | 筛选条件，JSON 字符串                                                                                              |
| `--page-num`   | 否   | 1         | 页码                                                                                                               |
| `--page-size`  | 否   | 10        | 每页条数（最大 100）                                                                                               |
| `--order-by`   | 否   | timestamp | 排序字段                                                                                                           |
| `--order`      | 否   | desc      | 排序方向（asc/desc）                                                                                               |
| `--show-all`   | 否   | false     | 显示全部字段（忽略 --columns，返回所有列）                                                                         |
| `--env`        | 否   | 自动检测  | 指定环境                                                                                                           |
| `--all-env`    | 否   | false     | 查询所有环境                                                                                                       |
| `--site-type`  | 否   | web       | 站点类型（web/app/hybrid）                                                                                         |
| `--region`     | 否   | cn        | 区域（cn/i18n）                                                                                                    |
| `--raw`        | 否   | false     | 输出原始 JSON                                                                                                      |

### --columns 说明

- 不指定时，CLI 自动调用 `log columns` 获取该 ev_type 的默认列
- 指定时，以逗号分隔列名：`--columns "url,message,stack"`
- 使用 `--show-all` 可返回全部字段，无需手动指定列名
- 不同 ev_type 的可用列完全不同，不确定时先用 `log columns` 查询

### --filter 筛选条件格式

JSON 数组，多个条件之间是 **AND** 关系。每个条件包含三个字段：

```json
[
  { "filter_name": "<字段名>", "op": "<操作符>", "values": ["<值1>", "<值2>"] },
  { "filter_name": "<字段名>", "op": "<操作符>", "values": ["<值>"] }
]
```

- `filter_name`：筛选字段名（必须与 API 实际字段名一致）
- `op`：操作符（详见 `references/filter-operators.md`）
- `values`：值数组（字符串类型，即使是数值也需用字符串）

**注意**：使用错误的 filter_name 不会报错，但筛选会被静默忽略，导致结果不符合预期。不确定时先运行 `log filters` 确认可用字段名。

#### filter_name 字段命名规则

| ev_type                               | 普通字段                                    | 自定义维度            | 自定义指标         |
| ------------------------------------- | ------------------------------------------- | --------------------- | ------------------ |
| http/js_error/view/session 等标准类型 | 直接用字段名：`res_status`、`browser_brand` | —                     | —                  |
| custom（自定义事件）                  | 通用字段直接用：`event_name`、`user_id`     | 带 `categories.` 前缀 | 带 `metrics.` 前缀 |

**标准类型示例**：

```bash
# HTTP 状态码 >= 400（字段名是 res_status，不是 status_code）
--filter '[{"filter_name":"res_status","op":"gte","values":["400"]}]'

# 浏览器为 Chrome
--filter '[{"filter_name":"browser_brand","op":"eq","values":["Chrome"]}]'
```

**自定义事件示例**：

```bash
# 查询指定 event_name 的自定义事件
--filter '[{"filter_name":"event_name","op":"in","values":["my_event"]}]'

# 按自定义维度筛选（注意 categories. 前缀）
--filter '[{"filter_name":"categories.page_type","op":"eq","values":["home"]}]'

# 按自定义指标筛选（注意 metrics. 前缀）
--filter '[{"filter_name":"metrics.load_time","op":"gte","values":["3000"]}]'
```

### 常用查询场景示例

```bash
# 1. 基础查询 — JS 错误明细
slardar-web-cli log query --bid <BID> --ev-type js_error

# 2. 指定时间范围（先通过 time parse 获取时间戳）
slardar-web-cli log query --bid <BID> --ev-type http --start-time 1773244800 --end-time 1773676800

# 3. 带筛选条件 — 查 HTTP 4xx/5xx 错误
slardar-web-cli log query --bid <BID> --ev-type http \
  --filter '[{"filter_name":"res_status","op":"gte","values":["400"]}]'

# 4. 自定义列 + 分页
slardar-web-cli log query --bid <BID> --ev-type js_error \
  --columns "url,message,stack" --page-size 20

# 5. 查询自定义事件（必须通过 filter 指定 event_name）
slardar-web-cli log query --bid <BID> --ev-type custom \
  --filter '[{"filter_name":"event_name","op":"in","values":["my_event"]}]'

# 6. 查询资源加载错误
slardar-web-cli log query --bid <BID> --ev-type resource_error

# 7. 查询白屏检测记录
slardar-web-cli log query --bid <BID> --ev-type blank_screen

# 8. 查询长任务
slardar-web-cli log query --bid <BID> --ev-type performance_longtask

# 9. 多条件组合筛选
slardar-web-cli log query --bid <BID> --ev-type http \
  --filter '[{"filter_name":"res_status","op":"gte","values":["400"]},{"filter_name":"req_method","op":"eq","values":["POST"]}]'

# 10. 全量字段查询
slardar-web-cli log query --bid <BID> --ev-type js_error --show-all --page-size 5
```

---

## log session — Session 分析

查询指定 session 的全部日志，自动翻页获取所有记录。

### 完整参数表

| 参数           | 必须 | 默认值   | 说明                       |
| -------------- | ---- | -------- | -------------------------- |
| `--bid`        | 是   | -        | 业务 ID                    |
| `--ev-type`    | 是   | -        | 事件类型（通常为 session） |
| `--session-id` | 是   | -        | Session ID                 |
| `--start-time` | 否   | 最近1天  | 开始时间戳（秒）           |
| `--end-time`   | 否   | 当前时间 | 结束时间戳（秒）           |
| `--env`        | 否   | 自动检测 | 指定环境                   |
| `--all-env`    | 否   | false    | 查询所有环境               |
| `--site-type`  | 否   | web      | 站点类型（web/app/hybrid） |
| `--region`     | 否   | cn       | 区域（cn/i18n）            |
| `--raw`        | 否   | false    | 输出原始 JSON              |

### 输出说明

输出包含以下信息：

- **用户基本信息**：session_id、user_id、device_id、浏览器、操作系统等
- **时间范围**：session 的起止时间
- **事件分布统计**：各 ev_type 的日志条数
- **按 ev_type 分组的日志明细**：每组最多 20 条记录

```bash
# 典型用法
slardar-web-cli log session --bid <BID> --ev-type session --session-id <SESSION_ID>

# 指定时间范围（session 默认查最近 1 天，比 query 的 1 小时更长）
slardar-web-cli log session --bid <BID> --ev-type session --session-id <SESSION_ID> \
  --start-time 1773244800 --end-time 1773676800
```

---

## log detail — 单条日志详情

查看一条日志的完整信息（包含 query 结果中未显示的字段）。

### 完整参数表

| 参数          | 必须 | 默认值   | 说明                              |
| ------------- | ---- | -------- | --------------------------------- |
| `--bid`       | 是   | -        | 业务 ID                           |
| `--ev-type`   | 是   | -        | 事件类型                          |
| `--dh-key`    | 是   | -        | 日志的 dh_key（从列表数据中获取） |
| `--env`       | 否   | 自动检测 | 指定环境                          |
| `--site-type` | 否   | web      | 站点类型（web/app/hybrid）        |
| `--region`    | 否   | cn       | 区域（cn/i18n）                   |
| `--raw`       | 否   | false    | 输出原始 JSON                     |

### 典型工作流（query → 取 dh_key → detail）

1. 先用 `log query` 查询日志列表，定位目标记录
2. 从查询结果中取出该记录的 `dh_key` 字段值
3. 用 `log detail` 传入 `dh_key` 查看完整详情

```bash
# 第 1 步：查询 JS 错误列表
slardar-web-cli log query --bid <BID> --ev-type js_error --page-size 5

# 第 2 步：从结果中找到目标记录的 dh_key，查看完整详情
slardar-web-cli log detail --bid <BID> --ev-type js_error --dh-key "xxxx-yyyy-zzzz"
```

---

## log columns — 获取可用列

查询指定 ev_type 的所有可用列名及是否为默认列，用于确认 `--columns` 参数值。

### 完整参数表

| 参数          | 必须 | 默认值   | 说明                       |
| ------------- | ---- | -------- | -------------------------- |
| `--bid`       | 是   | -        | 业务 ID                    |
| `--ev-type`   | 是   | -        | 事件类型                   |
| `--env`       | 否   | 自动检测 | 指定环境                   |
| `--site-type` | 否   | web      | 站点类型（web/app/hybrid） |
| `--region`    | 否   | cn       | 区域（cn/i18n）            |
| `--raw`       | 否   | false    | 输出原始 JSON              |

```bash
slardar-web-cli log columns --bid <BID> --ev-type js_error
slardar-web-cli log columns --bid <BID> --ev-type http
slardar-web-cli log columns --bid <BID> --ev-type custom
```

---

## log filters — 获取可筛选字段

查询指定 ev_type 的所有可筛选字段名及其类型，用于确认 `--filter` 参数中的 `filter_name`。

### 完整参数表

| 参数          | 必须 | 默认值   | 说明                       |
| ------------- | ---- | -------- | -------------------------- |
| `--bid`       | 是   | -        | 业务 ID                    |
| `--ev-type`   | 是   | -        | 事件类型                   |
| `--env`       | 否   | 自动检测 | 指定环境                   |
| `--site-type` | 否   | web      | 站点类型（web/app/hybrid） |
| `--region`    | 否   | cn       | 区域（cn/i18n）            |
| `--raw`       | 否   | false    | 输出原始 JSON              |

```bash
slardar-web-cli log filters --bid <BID> --ev-type http
slardar-web-cli log filters --bid <BID> --ev-type custom
```

---

## 注意事项

1. **ev_type 仅接受固定值**：`--ev-type` 仅接受 11 个固定值（js_error、http、view、session、resource_error、resource、custom、action、log、performance_longtask、blank_screen）。不在此列表中的事件名视为自定义事件名，必须用 `--ev-type custom` + `--filter` 中 `event_name` 的 `in` 筛选来查询。**禁止**将自定义事件名直接传入 `--ev-type`
2. **自定义事件查询方式**：使用 `--ev-type custom`，且 `--filter` 中**必须**包含 `{"filter_name":"event_name","op":"in","values":["xxx"]}`
3. **custom 字段带前缀**：自定义事件的维度列和筛选字段用 `categories.xxx`，指标列和筛选字段用 `metrics.xxx`，通用字段（如 `event_name`、`user_id`、`session_id`）直接使用
4. **字段名以 API 为准**：API 返回的实际字段名可能与直觉不同（如 HTTP 状态码是 `res_status` 而非 `status_code`）。不确定时用 `log columns` 或 `log filters` 确认
5. **数据保留期**：明细日志通常保留 30 天
6. **数据延迟**：日志摄入有约 5 分钟延迟，查询最近几分钟的数据可能不完整
7. **分页限制**：`--page-size` 最大 100，超出会被截断
8. **session 默认时间范围**：`log session` 默认查最近 1 天，`log query` 默认查最近 1 小时
