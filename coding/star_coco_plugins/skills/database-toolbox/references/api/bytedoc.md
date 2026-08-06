---
name: "database-skill-bytedoc"
description: "ByteDoc 全部 API：元数据探查（实例/集合/字段）、数据查询（execute_sql / nl2sql）、运维诊断（慢查询/会话/节点/磁盘）。支持 Cloud Native Mongo 和 Classic ByteDoc 两种子类型。"
---

# ByteDoc API 参考

ByteDoc 文档数据库的全部 Toolbox 方法。传 `db_type="ByteDoc"` 使用。

bytedoc.py 统一接管所有 ByteDoc 函数，内部按 `instance_type` 自动分流：
- **Mongo**（Cloud Native）→ 走 DBW API 或 ByteDoc Cloud API
- **Classic**（ByteDoc）→ 走 ByteDoc Classic 原生 API

> **vdc 参数**：所有函数均支持可选 `vdc` 参数。同名数据库在多个机房（VDC）中存在时，函数会返回 `fuzzy_matches` 错误（含各机房的 `vdc` 值），将用户选择的 vdc 传入即可消歧。返回值 `context.vdc` 可直接透传给后续调用。

> **大数据量截断**：返回列表较多时，`data` 中会包含 `total`（原始总数）、`returned_count`（本次返回数量）、`truncated: true` 和 `artifact_path`（完整数据的临时文件路径，JSON 格式）。这些元数据字段在 JSON 中位于列表字段之前。当 `truncated=true` 时，根据任务判断是否需要完整数据：定位 Top 问题（"有没有慢查询"、"最慢的 SQL"）inline 数据已足够；全量统计、分布分析、遍历所有条目时，基于 `artifact_path` 文件进行读取或分析（如用 python/jq 做聚合统计）。

> **重要：`sql` 参数使用 MongoDB 语法**（`show collections`、`db.<collection>.find()`、`db.<collection>.aggregate()`、`db.runCommand()`），不支持 MySQL 语法，不支持 shell helper（如 `getCollectionNames()`、`countDocuments()`）。

## 方法一览

| 方法 | 分类 | 可用范围 | Mongo | Classic | 说明 |
|------|------|------|-------|---------|------|
| `list_instances()` | 元数据 | China-North, China-East, China-BOE, China-North5, China-Fintech, ChinaSinf-North | ✅ | ✅ | 查询实例列表（须传 `database` 或 `favor`） |
| `list_tables()` | 元数据 | China-North, China-East, China-BOE, China-North5, China-Fintech, ChinaSinf-North | ✅ | ✅ | 列出集合 |
| `get_table_info()` | 元数据 | China-North, China-East, China-BOE, China-North5, China-Fintech, ChinaSinf-North | ✅ | ✅ | 获取集合结构（Mongo: 字段定义；Classic: 索引信息） |
| `nl2sql()` | 数据查询 | China-North, China-East, China-BOE, China-North5, China-Fintech, ChinaSinf-North | ✅ | ❌ | 自然语言转查询语句（**必须指定 tables**） |
| `execute_sql()` | 数据查询 | China-North, China-East, China-BOE, China-North5, China-Fintech, ChinaSinf-North | ✅ | ✅ | 执行只读查询（MongoDB 语法） |
| `describe_slow_logs()` | 运维诊断 | China-North, China-East, China-BOE, China-North5, China-Fintech, ChinaSinf-North | ✅ | ✅ | 慢查询（Mongo: 明细日志；Classic: 聚合统计） |
| `list_active_sessions()` | 运维诊断 | China-North, China-East, China-BOE, China-North5, China-Fintech, ChinaSinf-North | ✅ | ❌ | 实时连接/进程 |
| `describe_instance_nodes()` | 运维诊断 | China-North, China-East, China-BOE, China-North5, China-Fintech, ChinaSinf-North | ✅ | ✅ | 实例节点（Mongo: 节点列表；Classic: 静态信息） |
| `describe_table_space()` | 监控空间 | China-North, China-East, China-BOE, China-North5, China-Fintech, ChinaSinf-North | ✅ | ✅ | collection 磁盘详情 |
| `get_ticket_url()` | 元数据 | China-North, China-East, China-BOE, China-North5, China-Fintech, ChinaSinf-North | ✅ | ✅ | 获取工单链接（区分 Mongo / Classic） |

---

## 元数据探查

### list_instances()

查询 ByteDoc 实例列表。**不传 `database` 或 `favor=True` 时返回空列表。**

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `client` | DBWClient | **是** | — | 数据库客户端 |
| `db_type` | str | **是** | — | 必须传 `"ByteDoc"` |
| `database` | str | 条件必填 | None | 按名称查找（与 `favor` 至少传一个） |
| `favor` | bool | 条件必填 | False | 仅查收藏（与 `database` 至少传一个） |
| `page_number` | int | 否 | 1 | 页码 |
| `page_size` | int | 否 | 10 | 每页数量 |

```python
from toolbox import create_client, list_instances

client = create_client(vregion="China-North")
result = list_instances(client, db_type="ByteDoc", database="my_doc_db")
result = list_instances(client, db_type="ByteDoc", favor=True)
```

返回 `data.instances[]`，关键字段：`id`、`name`、`status`、`type`（Mongo / ByteDoc）。

---

### list_tables()

列出 ByteDoc 实例下的集合。

- **Mongo**: 通过 DBW API 获取，返回集合名列表
- **Classic**: 通过 ByteDoc 原生 `get_collections` 获取

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `client` | DBWClient | **是** | — | 数据库客户端 |
| `database` | str | **是** | — | 数据库名称 |
| `db_type` | str | 否 | — | 传 `"ByteDoc"` |
| `page_number` | int | 否 | 1 | 页码（仅 Mongo） |
| `page_size` | int | 否 | 50 | 每页数量（仅 Mongo） |
| `fetch_all` | bool | 否 | False | 自动翻页获取全部（仅 Mongo） |

```python
result = list_tables(client, database="my_doc_db", db_type="ByteDoc")
```

返回 `data.tables[]` 集合名列表，`data.total` 总数。

---

### get_table_info()

获取集合结构信息。

- **Mongo**: 返回字段定义（`data.columns[]`）
- **Classic**: 返回索引信息（`data.indexes[]`）

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `client` | DBWClient | **是** | 数据库客户端 |
| `table` | str | **是** | 集合名 |
| `database` | str | **是** | 数据库名称 |
| `db_type` | str | 否 | 传 `"ByteDoc"` |

```python
result = get_table_info(client, table="my_collection", database="my_doc_db", db_type="ByteDoc")
```

---

## 数据查询

按子类型选择工作流：

**Mongo（Cloud Native）**：
- **nl2sql 方式**：`list_tables() → nl2sql(query, tables=[...], db_type="ByteDoc") → execute_sql()`
- **自写方式**：`list_tables() → get_table_info()` 获取字段定义 → 自行编写 MongoDB 语句 → `execute_sql()`

**Classic ByteDoc**（无 nl2sql，`get_table_info` 仅返回索引不含字段结构）：
- `list_tables()` 获取 collection 列表 → `execute_sql(sql="db.<col>.find().limit(1)")` 查看 sample document 推断 schema → 自行编写 MongoDB 语句 → `execute_sql()`

### nl2sql()

> **仅 Mongo 支持。Classic ByteDoc 暂不支持 nl2sql。**
> **ByteDoc 必须指定 `tables` 参数**。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `client` | DBWClient | **是** | 数据库客户端 |
| `query` | str | **是** | 自然语言查询描述 |
| `database` | str | **是** | 数据库名称 |
| `tables` | List[str] | **是** | 涉及的集合名（ByteDoc 必填） |
| `db_type` | str | **是** | 传 `"ByteDoc"` |

```python
from toolbox import create_client, nl2sql, execute_sql

client = create_client(vregion="China-North")
result = nl2sql(client, query="查询 category 为 tech 的文档数量",
                database="my_doc_db", tables=["articles"], db_type="ByteDoc")
if result["success"]:
    data = execute_sql(client, sql=result["data"]["sql"],
                       database="my_doc_db", db_type="ByteDoc")
```

返回 `data.sql` — 生成的 MongoDB 查询语句。

---

### execute_sql()

执行只读查询，一次一条。

- **Mongo**: 通过 DBW API 执行，支持 `show collections`、`db.<col>.find()`、`db.<col>.aggregate()` 等
- **Classic**: 通过 ByteDoc 原生 `web_query` 执行，**必须使用 `db.<collection>.operation(...)` 格式**

> ⚠️ **行数上限**：平台最多返回 **3000 行**，`find()` 查询建议加 `.limit(n)`。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `client` | DBWClient | **是** | 数据库客户端 |
| `sql` | str | **是** | MongoDB 查询语句 |
| `database` | str | **是** | 数据库名称 |
| `db_type` | str | 否 | 传 `"ByteDoc"` |

```python
# 查询文档
result = execute_sql(client, sql="db.my_col.find({category: 'tech'}).limit(10)",
                     database="my_doc_db", db_type="ByteDoc")

# 聚合查询
result = execute_sql(client,
    sql="db.my_col.aggregate([{$group: {_id: '$category', count: {$sum: 1}}}])",
    database="my_doc_db", db_type="ByteDoc")

# 列出所有集合（仅 Mongo 支持）
result = execute_sql(client, sql="show collections",
                     database="my_doc_db", db_type="ByteDoc")
```

---

## 运维诊断

> ByteDoc 无 `describe_aggregate_slow_logs`、`slow_query_trend`、`list_slow_query_advice` 等聚合/趋势/建议接口。

### describe_slow_logs()

查询 ByteDoc 慢查询日志。

- **Mongo**: 返回慢查询**明细日志**（每条独立记录）
- **Classic**: 返回慢查询**聚合统计**（按 query pattern 分组）

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `client` | DBWClient | **是** | — | 数据库客户端 |
| `start_time` | str | **是** | — | 开始时间（ISO 8601，不带时区默认北京时间） |
| `end_time` | str | **是** | — | 结束时间 |
| `database` | str | **是** | — | 数据库名称 |
| `db_type` | str | **是** | — | 传 `"ByteDoc"` |
| `page_size` | int | 否 | 10 | 每页数量（仅 Mongo） |
| `sort_by` | str | 否 | "DESC" | 排序方向（仅 Mongo） |
| `millis` | int | 否 | 100 | 慢查询阈值毫秒数（仅 Classic） |

```python
from toolbox import create_client, describe_slow_logs

client = create_client(vregion="China-North")
result = describe_slow_logs(client, start_time="2026-03-08 00:00:00",
    end_time="2026-03-08 01:00:00", database="my_doc_db", db_type="ByteDoc")
```

**Mongo 返回** `data.logs[]`，关键字段：
- `sql` — 操作详情 JSON（含 op、ns 等）
- `query_time` — 执行耗时（毫秒）
- `rows_scanned`（FileScan）— 文档扫描数
- `index_scan` — 索引扫描数（0 表示未使用索引）
- `rows_sent`（Return）— 返回文档数

**Classic 返回** `data.logs[]`，关键字段：
- `sql` — 查询模式
- `collection` — 集合名
- `count` — 出现次数
- `avg_time_ms` — 平均耗时（毫秒）
- `max_time_ms` — 最大耗时（毫秒）

---

### list_active_sessions()

> **仅 Mongo 支持。Classic ByteDoc 暂不支持。**

查询实时连接/进程列表，按执行时间降序排列。

```python
result = list_active_sessions(client, database="my_doc_db", db_type="ByteDoc")
# show_sleep=True 包含 Sleep 连接
```

---

### describe_instance_nodes()

查询实例节点信息。

- **Mongo**: 返回节点列表（节点 ID、角色 Primary/Secondary）
- **Classic**: 返回数据库静态信息

```python
result = describe_instance_nodes(client, database="my_doc_db", db_type="ByteDoc")
```

---

### describe_table_space()

查询 collection 磁盘详情。

- **Mongo**: 通过 ByteDoc Cloud API 获取 collection 磁盘信息
- **Classic**: 先获取 cluster_id，再通过 capacity_manager API 获取

```python
result = describe_table_space(client, database="my_doc_db", db_type="ByteDoc")
```

**返回** `data.tables[]`，关键字段：
- `collection` — 集合名
- `size` — 磁盘大小
- `documents` — 文档数
- Mongo 额外字段：`avg_doc_size`、`index_count`、`last_day_incr_size_gb`、`last_day_incr_count`
