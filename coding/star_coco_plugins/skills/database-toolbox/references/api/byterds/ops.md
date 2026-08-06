---
name: "database-skill-byterds-ops"
description: "ByteRDS 运维诊断：慢查询诊断、全量SQL分析、实时会话、事务锁诊断、监控与空间分析。"
---

# ByteRDS 运维诊断

分析 ByteRDS 实例的慢查询、事务锁、空间占用等运维问题，获取优化建议。

> **可用范围**：`全部` = 所有 VRegion 可用；其他情况直接列出支持的 VRegion。

> **vdc 参数**：所有函数均支持可选 `vdc` 参数。同名数据库在多个机房（VDC）中存在时，函数会返回 `fuzzy_matches` 错误（含各机房的 `vdc` 值），将用户选择的 vdc 传入即可消歧。返回值 `context.vdc` 可直接透传给后续调用。

> **大数据量截断**：返回列表较多时，`data` 中会包含 `total`（原始总数）、`returned_count`（本次返回数量）、`truncated: true` 和 `artifact_path`（完整数据的临时文件路径，JSON 格式）。这些元数据字段在 JSON 中位于列表字段之前。当 `truncated=true` 时，根据任务判断是否需要完整数据：定位 Top 问题（"有没有慢查询"、"最慢的 SQL"）inline 数据已足够；全量统计、分布分析、遍历所有条目时，基于 `artifact_path` 文件进行读取或分析（如用 python/jq 做聚合统计）。

## 方法一览

### 慢查询与全量 SQL

| 方法 | 说明 | 所需参数 | 可用范围 |
|------|------|---------|------|
| `describe_aggregate_slow_logs()` | 慢查询聚合统计（按 SQL 模板聚合），**推荐首选** | database | 全部 |
| `describe_slow_logs()` | 查询慢查询明细日志 | database | China-North, China-East, China-BOE, China-North5, China-Fintech |
| `slow_query_trend()` | 慢查询时间序列趋势 | database | China-North, China-East, China-BOE, China-North5, China-Fintech |
| `slow_query_advice_task_history()` | 查看诊断任务历史，获取 summary_id | database | China-North, China-East, China-North5, China-Fintech |
| `list_slow_query_advice()` | 获取索引/SQL 改写建议 | database | China-North, China-East, China-BOE, China-North5, China-Fintech |
| `describe_full_sql_detail()` | 查询全量 SQL 历史详情 | database | China-North, China-BOE, China-North5, China-Fintech |

### 实时会话

| 方法 | 说明 | 所需参数 | 可用范围 |
|------|------|---------|------|
| `list_active_sessions()` | 查询实时连接/进程（类似 SHOW PROCESSLIST） | database | 全部 |

### 事务锁诊断

| 方法 | 说明 | 所需参数 | 可用范围 |
|------|------|---------|------|
| `describe_deadlock()` | 查询死锁信息 | database | 全部 |
| `list_transactions()` | 查询事务和锁列表 | database | 全部 |
| `transaction_snapshots()` | 查询事务快照 | database | China-North, China-East, China-BOE, China-North5, China-Fintech |
| `export_transactions()` | 创建事务导出任务 | database | China-North, China-East, China-BOE, China-North5, China-Fintech |

### 监控与空间分析

| 方法 | 说明 | 所需参数 | 可用范围 |
|------|------|---------|------|
| `table_write_analysis()` | 表级写入分析（按表聚合 DML/DDL 统计，定位写入最频繁的表） | database | China-North, China-BOE, China-North5, China-Fintech |
| `describe_table_metric()` | 获取表级监控 DML/DDL | database | China-North, China-BOE, China-North5, China-Fintech |
| `get_metric_data_predict()` | 监控数据预测 | database | China-BOE |
| `describe_instance_nodes()` | 查询实例节点列表（IP:Port、角色） | database | 全部 |
| `describe_health_summary()` | 实例健康概览（CPU、内存等） | database | 全部 |
| `describe_table_space()` | 查询表空间详情 | database | 全部 |

---

## describe_aggregate_slow_logs()

按 SQL 模板聚合慢查询统计，查看哪些 SQL 模式出现最频繁、耗时最多。**推荐首选**。

### 参数

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `client` | DBWClient | **是** | — | 数据库客户端 |
| `start_time` | str | **是** | — | 开始时间（ISO 8601，不带时区默认北京时间） |
| `end_time` | str | **是** | — | 结束时间 |
| `database` | str | **是** | — | 数据库名称 |
| `page_number` | int | 否 | 1 | 页码 |
| `page_size` | int | 否 | 10 | 每页数量 |
| `order_by` | str | 否 | "TotalQueryTime" | 排序字段：`TotalQueryTime` / `ExecuteCount` / `MaxQueryTime` / `AverageQueryTime`（China-North 等 DBW 后端额外支持 `MaxRowsExamined` / `AverageRowsExamined` / `MaxLockTime` / `AverageLockTime`） |
| `sort_by` | str | 否 | "DESC" | 排序方向：`ASC` / `DESC` |
| `users` | list[str] | 否 | None | 按用户名过滤 |
| `source_ips` | list[str] | 否 | None | 按来源 IP 过滤 |
| `keywords` | list[str] | 否 | None | 按 SQL 关键词过滤 |
| `tables` | list[str] | 否 | None | 按表名过滤 |
| `sql_methods` | list[str] | 否 | None | 按 SQL 类型过滤：SELECT, INSERT, UPDATE, DELETE 等 |
| `min_query_time` | float | 否 | None | 最小查询耗时（秒） |
| `max_query_time` | float | 否 | None | 最大查询耗时（秒） |
| `group_ignored` | list[str] | 否 | ["User","SourceIP","PSM"] | 聚合忽略维度：User, SourceIP, PSM, Table, SqlMethod, DB, SQLTemplateID |

### 用法

```python
from toolbox import create_client, describe_aggregate_slow_logs

client = create_client(vregion="China-North")

result = describe_aggregate_slow_logs(
    client,
    start_time="2026-03-08 00:00:00",
    end_time="2026-03-08 01:00:00",
    database="mydb"
)

# 只看 SELECT 类型、耗时超过 1 秒
result = describe_aggregate_slow_logs(
    client,
    start_time="2026-03-08 00:00:00",
    end_time="2026-03-08 01:00:00",
    database="mydb",
    sql_methods=["SELECT"],
    min_query_time=1.0
)
```

### 返回

`data.logs[]`，关键字段：
- `sql_template_id` — SQL 模板唯一标识，传给 `describe_slow_logs(sql_template_id=...)` 可查该模板的具体慢查询明细（仅 China-North, China-East, China-BOE, China-North5, China-Fintech 返回）
- `sql_template` / `sql_method` / `table` / `db` — SQL 模板文本和归属
- `example_sql` — 实际 SQL 样本（仅 i18n VRegion 返回；有 `sql_template_id` 时可用 `describe_slow_logs` 查看具体 SQL）
- `execute_count` / `execute_count_ratio` — 执行次数和占比
- `query_time_stats` — 查询耗时统计（Average/Max/Total）。DBW 后端额外返回 `lock_time_stats` / `rows_examined_stats`（含 Min），ByteRDS 直连后端（ChinaSinf-North + i18n）仅返回 `query_time_stats`
- `first_appear_time` / `last_appear_time` — 首末出现时间
- `truncated` / `artifact_path`（可选）— 结果超过 50 条时 `truncated=true`，完整数据写入 `artifact_path` 临时文件（JSON），`logs` 仅含 Top 50。用 Read/Grep 按需读取文件

---

## describe_slow_logs()

查询慢查询明细日志。⚠️ **仅 China-North, China-East, China-BOE, China-North5, China-Fintech 可用**，调用前先检查 `describe_aggregate_slow_logs` 返回是否包含 `sql_template_id`。有则可用本接口按模板过滤，无则跳过。

### 参数

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `client` | DBWClient | **是** | — | 数据库客户端 |
| `start_time` | str | **是** | — | 开始时间（ISO 8601，不带时区默认北京时间） |
| `end_time` | str | **是** | — | 结束时间 |
| `database` | str | **是** | — | 数据库名称 |
| `sql_template_id` | str | 否 | None | SQL 模板 ID（从 `describe_aggregate_slow_logs` 获取） |
| `page_number` | int | 否 | 1 | 页码 |
| `page_size` | int | 否 | 10 | 每页数量 |
| `order_by` | str | 否 | "QueryTime" | 排序字段：`QueryTime` / `RowsExamined` / `LockTime` |
| `sort_by` | str | 否 | "DESC" | 排序方向 |

### 用法

```python
from toolbox import create_client, describe_aggregate_slow_logs, describe_slow_logs

client = create_client(vregion="China-North")

# 推荐：先聚合 → 再按 sql_template_id 查明细
agg = describe_aggregate_slow_logs(
    client, start_time="2026-03-08 00:00:00",
    end_time="2026-03-08 01:00:00", database="mydb"
)
tid = agg["data"]["logs"][0]["sql_template_id"]

result = describe_slow_logs(
    client, start_time="2026-03-08 00:00:00",
    end_time="2026-03-08 01:00:00", database="mydb",
    sql_template_id=tid, page_size=20
)
```

### 返回

`data.logs[]`，关键字段：
- `sql` — 完整 SQL 文本
- `template` — SQL 模板
- `query_time` — 查询耗时（秒）
- `lock_time` — 锁等待时间（秒）
- `rows_scanned` / `rows_sent` — 扫描行数 / 返回行数
- `time` / `timestamp` — 执行时间
- `user` / `ip` / `db` — 来源信息

---

## slow_query_trend()

查询慢查询数量的时间序列趋势，用于定位高峰时段。

### 参数

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `client` | DBWClient | **是** | — | 数据库客户端 |
| `start_time` | str | **是** | — | 开始时间 |
| `end_time` | str | **是** | — | 结束时间 |
| `database` | str | **是** | — | 数据库名称 |
| `interval` | int | 否 | 300 | 统计间隔（秒） |
| `users` | list[str] | 否 | None | 按用户名过滤 |
| `source_ips` | list[str] | 否 | None | 按来源 IP 过滤 |
| `min_query_time` | float | 否 | None | 最小查询耗时（秒） |
| `max_query_time` | float | 否 | None | 最大查询耗时（秒） |

```python
from toolbox import create_client, slow_query_trend

client = create_client(vregion="China-North")
result = slow_query_trend(
    client, start_time="2026-03-08 00:00:00",
    end_time="2026-03-08 01:00:00", database="mydb", interval=60
)
```

---

## slow_query_advice_task_history()

查看已完成的慢查询诊断任务历史。返回的 `id`（即 summary_id）用于 `list_slow_query_advice()`。

> ⚠️ China-BOE 不支持。

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `client` | DBWClient | **是** | — | 数据库客户端 |
| `database` | str | **是** | — | 数据库名称 |
| `page_number` | int | 否 | 1 | 页码 |
| `page_size` | int | 否 | 10 | 每页数量 |

```python
from toolbox import create_client, slow_query_advice_task_history

client = create_client(vregion="China-North")
result = slow_query_advice_task_history(client, database="mydb", page_size=5)

tasks = result["data"]["tasks"]
latest = next(t for t in tasks if t["status"] == "SUCCESS")
summary_id = latest["id"]
```

### 返回

`data.tasks[]`，关键字段：
- `id` — 诊断任务 ID（即 summary_id），传给 `list_slow_query_advice(summary_id=...)` 获取建议
- `status` — 任务状态：`INIT`（诊断中）/ `SUCCESS`（已完成）/ `EXCEPTION`（异常）。仅 SUCCESS 有建议可查
- `slow_query_count` / `index_advice_count` / `rewrite_advice_count` — 统计数量

---

## list_slow_query_advice()

基于诊断任务结果，获取索引优化或 SQL 改写建议。

> ⚠️ China-BOE 不支持。

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `client` | DBWClient | **是** | — | 数据库客户端 |
| `database` | str | **是** | — | 数据库名称 |
| `summary_id` | str | **是** | — | 诊断任务 ID（从 `slow_query_advice_task_history()` 获取） |
| `advice_type` | str | **是** | — | 建议类型：`index_advice` / `rewrite_sql_advice` |
| `group_by` | str | **是** | — | 分组方式：`Module`（按 SQL 模板）/ `Advice`（按建议） |
| `order_by` | str | 否 | "Benefit" | 排序字段 |
| `page_number` | int | 否 | 1 | 页码 |
| `page_size` | int | 否 | 10 | 每页数量 |

```python
from toolbox import create_client, list_slow_query_advice

client = create_client(vregion="China-North")

# 索引建议
result = list_slow_query_advice(
    client, database="mydb", summary_id="summary_xxx",
    advice_type="index_advice", group_by="Advice"
)

# SQL 改写建议
result = list_slow_query_advice(
    client, database="mydb", summary_id="summary_xxx",
    advice_type="rewrite_sql_advice", group_by="Module"
)
```

### 返回

`data.advices[]`，关键字段：
- `table` / `sql` — 涉及的表和 SQL
- `advice` — 建议内容（如 `CREATE INDEX idx_status ON orders(status)`）
- `level` — 优先级：`High` / `Medium` / `Low`
- `benefit` — 收益分（0~1，越高越好）
- `speed_up` — 预估加速倍数
- `estimated_time_after` — 优化后预估耗时（秒）

---

## describe_full_sql_detail()

查询全量 SQL 历史详情。实例未开启全量 SQL 分析时返回错误。

> ⚠️ China-East 不支持。

### 参数

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `client` | DBWClient | **是** | — | 数据库客户端 |
| `start_time` | str | **是** | — | 开始时间 |
| `end_time` | str | **是** | — | 结束时间 |
| `database` | str | **是** | — | 数据库名称 |
| `page_size` | int | 否 | 10 | 每页数量 |
| `users` | list[str] | 否 | None | 按用户名过滤 |
| `source_ips` | list[str] | 否 | None | 按来源 IP 过滤 |
| `keywords` | list[str] | 否 | None | 按 SQL 关键词过滤 |
| `tables` | list[str] | 否 | None | 按表名过滤 |
| `sql_methods` | list[str] | 否 | None | 按 SQL 类型过滤 |
| `min_exec_time` | int | 否 | None | 最小执行耗时（微秒） |
| `max_exec_time` | int | 否 | None | 最大执行耗时（微秒） |
| `context` | str | 否 | None | 翻页游标（从上次返回的 `data.context` 获取） |

> **翻页方式**：不使用 page_number，使用游标翻页。将上次返回的 `data.context` 传入下次调用。`data.list_over=true` 表示已无更多数据。

### 用法

```python
from toolbox import create_client, describe_full_sql_detail

client = create_client(vregion="China-North")

result = describe_full_sql_detail(
    client, start_time="2026-03-08 00:10:00",
    end_time="2026-03-08 00:18:00", database="mydb"
)

# 翻页
next_page = describe_full_sql_detail(
    client, start_time="2026-03-08 00:00:00",
    end_time="2026-03-08 01:00:00", database="mydb",
    context=result["data"]["context"]
)
```

### 返回

`data.sql_list[]`，关键字段：
- `query_string` / `sql_fingerprint` / `sql_type` — SQL 文本、指纹和类型
- `exec_time` / `cpu_time` — 执行耗时和 CPU 耗时（微秒）
- `rows_examined` / `rows_sent` — 扫描行数 / 返回行数
- `user_name` / `client_ip` / `node_id` — 来源信息

翻页控制：
- `data.context` — 翻页游标，传给下次调用的 `context` 参数
- `data.list_over` — `true` 表示已无更多数据

---

## list_active_sessions()

查询数据库实时连接/进程列表（类似 `SHOW PROCESSLIST`），按执行时间降序排列。

> **注意区分**：查询的是**数据库引擎层的实时连接**（process），不是 DBW 工作台的 SQL 窗口会话。

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `client` | DBWClient | **是** | — | 数据库客户端 |
| `database` | str | **是** | — | 数据库名称 |
| `show_sleep` | bool | 否 | False | 是否包含 Sleep 状态连接 |

```python
from toolbox import create_client, list_active_sessions

client = create_client(vregion="China-North")
result = list_active_sessions(client, database="mydb")
result = list_active_sessions(client, database="mydb", show_sleep=True)
```

### 返回

`data.sessions[]`（按 `time` 降序），关键字段：
- `process_id` — 进程 ID
- `user` / `host` / `db` — 连接信息
- `command` — 命令类型（Query/Sleep 等）
- `time` — 执行时间（秒）
- `state` — 当前状态
- `info` — 正在执行的 SQL
- `blocking_pid` — 阻塞源进程 ID

---

## describe_deadlock()

查询实例的死锁信息。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `client` | DBWClient | **是** | 数据库客户端 |
| `database` | str | **是** | 数据库名称 |

```python
from toolbox import create_client, describe_deadlock

client = create_client(vregion="China-North")
result = describe_deadlock(client, database="mydb")
```

---

## list_transactions()

查询当前事务和锁列表。

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `client` | DBWClient | **是** | — | 数据库客户端 |
| `database` | str | **是** | — | 数据库名称 |
| `page_number` | int | 否 | 1 | 页码 |
| `page_size` | int | 否 | 10 | 每页数量 |

```python
from toolbox import create_client, list_transactions

client = create_client(vregion="China-North")
result = list_transactions(client, database="mydb")
```

---

## transaction_snapshots()

查询事务快照。

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `client` | DBWClient | **是** | — | 数据库客户端 |
| `start_time` | str | **是** | — | 开始时间 |
| `end_time` | str | **是** | — | 结束时间 |
| `database` | str | **是** | — | 数据库名称 |
| `page_number` | int | 否 | 1 | 页码 |
| `page_size` | int | 否 | 10 | 每页数量 |

```python
from toolbox import create_client, transaction_snapshots

client = create_client(vregion="China-North")
result = transaction_snapshots(
    client, start_time="2026-03-08 00:00:00",
    end_time="2026-03-09 00:00:00", database="mydb"
)
```

---

## export_transactions()

创建事务导出任务。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `client` | DBWClient | **是** | 数据库客户端 |
| `task_name` | str | **是** | 任务名称 |
| `start_time` | int | **是** | 开始时间（Unix 时间戳） |
| `end_time` | int | **是** | 结束时间（Unix 时间戳） |
| `database` | str | **是** | 数据库名称 |

```python
from toolbox import create_client, export_transactions

client = create_client(vregion="China-North")
result = export_transactions(
    client, task_name="trx_export_20260308",
    start_time=1741363200, end_time=1741449600, database="mydb"
)
```

---

## table_write_analysis()

表级写入分析：按表维度聚合 DML/DDL 统计，定位写入最频繁的表。依赖全量 SQL 功能开启。

> ⚠️ China-East 不支持。

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `client` | DBWClient | **是** | — | 数据库客户端 |
| `start_time` | str | **是** | — | 开始时间 |
| `end_time` | str | **是** | — | 结束时间 |
| `database` | str | **是** | — | 数据库名称 |
| `tables` | list[str] | 否 | [] | 按表名过滤 |
| `order_by` | str | 否 | "ExecCount" | 排序字段（见下方枚举） |
| `sort_by` | str | 否 | "ASC" | 排序方向 |

**order_by 可选值：**

| 值 | 说明 |
|------|------|
| `ExecCount` | 总执行次数 |
| `ExecTimeAvg` | 平均执行时间 |
| `DmlExecCount` | DML 执行次数 |
| `DmlExecTimeAvg` / `DmlExecTimeTotal` / `DmlExecTimeMax` | DML 执行时间统计 |
| `DmlRowExaminedAvg` / `DmlRowAffectedAvg` | DML 行数统计 |
| `DmlRowLockWaitAvg` / `DmlMdlWaitAvg` | DML 锁等待统计 |
| `DdlExecCount` | DDL 执行次数 |
| `DdlExecTimeAvg` / `DdlExecTimeTotal` / `DdlExecTimeMax` | DDL 执行时间统计 |
| `DdlRowExaminedAvg` / `DdlRowAffectedAvg` | DDL 行数统计 |
| `DdlRowLockWaitAvg` / `DdlMdlWaitAvg` | DDL 锁等待统计 |

```python
from toolbox import create_client, table_write_analysis

client = create_client(vregion="China-North")
result = table_write_analysis(
    client, start_time="2026-03-08 00:00:00",
    end_time="2026-03-09 00:00:00", database="mydb",
    order_by="DmlExecCount", sort_by="DESC"
)
```

---

## describe_table_metric()

获取表级别监控指标（DML 或 DDL 操作统计）。依赖全量 SQL 功能开启。

> ⚠️ China-East 不支持。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `client` | DBWClient | **是** | 数据库客户端 |
| `item_type` | `"DML"` \| `"DDL"` | **是** | 监控类型 |
| `db_name` | str | **是** | 数据库名 |
| `table` | str | **是** | 表名 |
| `start_time` | str | **是** | 开始时间 |
| `end_time` | str | **是** | 结束时间 |
| `database` | str | **是** | 数据库名称 |

```python
from toolbox import create_client, describe_table_metric

client = create_client(vregion="China-North")
result = describe_table_metric(
    client, item_type="DML", db_name="mydb", table="users",
    start_time="2026-03-08 00:00:00", end_time="2026-03-09 00:00:00",
    database="mydb"
)
```

---

## get_metric_data_predict()

获取监控数据预测。时间跨度不超过 7 天。

> ⚠️ 仅 BOE（`vregion="boe"` / `China-BOE`）可用。

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `client` | DBWClient | **是** | — | 数据库客户端 |
| `metric_name` | str | **是** | — | 指标名称 |
| `start_time` | str | **是** | — | 开始时间 |
| `end_time` | str | **是** | — | 结束时间 |
| `period` | int | 否 | 60 | 采样周期（秒） |
| `database` | str | **是** | — | 数据库名称 |

```python
from toolbox import create_client, get_metric_data_predict

client = create_client(vregion="boe")
result = get_metric_data_predict(
    client, metric_name="CpuUtil",
    start_time="2026-03-08 00:00:00", end_time="2026-03-09 00:00:00",
    database="mydb"
)
```

---

## describe_instance_nodes()

查询实例节点列表，返回每个节点的 IP:Port 和角色。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `client` | DBWClient | **是** | 数据库客户端 |
| `database` | str | **是** | 数据库名称 |

```python
from toolbox import create_client, describe_instance_nodes

client = create_client(vregion="China-North")
result = describe_instance_nodes(client, database="mydb")
```

### 返回

`data.nodes[]`，关键字段：
- `node_id` — 节点 ID（格式 IP:Port），可传给 `describe_health_summary(node_ids=[...])`
- `node_type` — 角色：`Primary` / `Secondary`

---

## describe_health_summary()

查询实例健康概览，包含 CPU、内存使用率等监控指标。**全部 VRegion 可用。**

> 不传 `node_ids` 时自动获取主节点。也可先调 `describe_instance_nodes()` 获取节点列表后指定。

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `client` | DBWClient | **是** | — | 数据库客户端 |
| `start_time` | str | **是** | — | 开始时间 |
| `end_time` | str | **是** | — | 结束时间 |
| `database` | str | **是** | — | 数据库名称 |
| `node_ids` | list[str] | 否 | 自动取主节点 | 节点 ID 列表（如 `["10.203.8.199:3306"]`） |
| `diag_type` | str | 否 | `"ALL"` | 诊断类型 |

```python
from toolbox import create_client, describe_health_summary

client = create_client(vregion="China-North")

# 自动获取主节点（推荐）
result = describe_health_summary(
    client, start_time="2026-03-08 10:00:00",
    end_time="2026-03-08 10:15:00", database="mydb"
)

# 手动指定节点
from toolbox import describe_instance_nodes
nodes = describe_instance_nodes(client, database="mydb")
node_id = nodes["data"]["nodes"][0]["node_id"]
result = describe_health_summary(
    client, start_time="2026-03-08 10:00:00",
    end_time="2026-03-08 10:15:00", database="mydb",
    node_ids=[node_id]
)
```

### 返回

`data.metrics[]`，关键字段：
- `name` — 指标名称（如"CPU利用率"）
- `avg` / `max` / `min` — 统计值
- `unit` — 单位
- `mom` — 环比（正值=上升，负值=下降）
- `yoy` — 同比（正值=上升，负值=下降）

> 部分 vregion（ByteRDS 直连通道）额外返回 `data.diagnostic`：
> - `suggests` — 优化建议列表
> - `slave_status` — 主从复制状态
> - `slow_logs` — 诊断期间的慢查询

---

## describe_table_space()

查询表空间详情，可按数据库或表名过滤。

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `client` | DBWClient | **是** | — | 数据库客户端 |
| `database` | str | **是** | — | 数据库名称 |
| `filter_database` | str | 否 | None | 按数据库名过滤 |
| `table_name` | str | 否 | None | 按表名过滤 |
| `page_number` | int | 否 | 1 | 页码 |
| `page_size` | int | 否 | 10 | 每页数量 |

```python
from toolbox import create_client, describe_table_space

client = create_client(vregion="China-North")
result = describe_table_space(client, database="mydb", filter_database="mydb")
```

---

## 常用辅助查询

> 平台仅允许：SELECT、SHOW TABLES、SHOW CREATE TABLE、EXPLAIN。

```python
from toolbox import create_client, execute_sql

client = create_client(vregion="China-North")

# 查看表结构
result = execute_sql(client, sql="SHOW CREATE TABLE my_table", database="mydb")

# 查看执行计划
result = execute_sql(client, sql="EXPLAIN SELECT * FROM my_table WHERE id=1", database="mydb")

# 查看表统计信息
result = execute_sql(client, sql="""
    SELECT TABLE_NAME, TABLE_ROWS, DATA_LENGTH
    FROM information_schema.TABLES
    WHERE TABLE_SCHEMA='mydb'
    ORDER BY DATA_LENGTH DESC LIMIT 10
""", database="mydb")
```
