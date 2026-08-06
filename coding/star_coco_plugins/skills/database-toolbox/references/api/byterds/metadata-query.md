---
name: "database-skill-byterds-metadata-query"
description: "ByteRDS 元数据与数据查询：实例/表/字段探查、nl2sql、execute_sql、query_sql、缓存搜索、全局配置、工单。"
---

# ByteRDS 元数据与数据查询

探查 ByteRDS 及多云 MySQL（VeDBMySQL、MySQL、MySQLSharding）数据库的实例、表结构，并执行只读 SQL 查询和数据分析。

> **i18n 限制**：`list_instances()` 必须传 `database` 参数搜索，不支持列出全部实例。`nl2sql()` 仅 China-North, China-East, China-BOE, China-North5, China-Fintech 可用。

> **平台仅允许执行：SELECT、SHOW TABLES、SHOW CREATE TABLE、EXPLAIN。不支持 SHOW VARIABLES 等其他 SHOW 命令，也不支持写入操作。**

> **vdc 参数**：所有函数均支持可选 `vdc` 参数。同名数据库在多个机房（VDC）中存在时，函数会返回 `fuzzy_matches` 错误（含各机房的 `vdc` 值），将用户选择的 vdc 传入即可消歧。返回值 `context.vdc` 可直接透传给后续调用。

> **大数据量截断**：返回列表较多时，`data` 中会包含 `total`（原始总数）、`returned_count`（本次返回数量）、`truncated: true` 和 `artifact_path`（完整数据的临时文件路径，JSON 格式）。这些元数据字段在 JSON 中位于列表字段之前。当 `truncated=true` 时，根据任务判断是否需要完整数据：定位 Top 问题（"有没有慢查询"、"最慢的 SQL"）inline 数据已足够；全量统计、分布分析、遍历所有条目时，基于 `artifact_path` 文件进行读取或分析（如用 python/jq 做聚合统计）。

## 方法一览

| 方法 | 分类 | 说明 |
|------|------|------|
| `list_instances()` | 元数据 | 查询实例列表（须传过滤项） |
| `list_tables()` | 元数据 | 列出表（`fetch_all=True` 获取全部） |
| `get_table_info()` | 元数据 | 获取表结构（字段、类型、注释） |
| `search_cached_instances()` | 元数据 | 搜索本地历史记录（不需要 client） |
| `set_global_config()` | 元数据 | 设置全局配置（不需要 client） |
| `get_ticket_url()` | 元数据 | 获取工单链接（写操作/加索引/改表/清表/调参时主动调用） |
| `nl2sql()` | 查询 | 自然语言转 SQL（cn/boe only） |
| `execute_sql()` | 查询 | 执行只读 SQL，返回 dict |
| `query_sql()` | 查询 | 执行查询，返回 DataFrame |

---

## 元数据探查

### list_instances()

查询 ByteRDS 实例列表。须传过滤项（`database`/`psm`/`favor`/`owned` 至少一个）。

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `client` | DBWClient | **是** | — | 数据库客户端 |
| `db_type` | str | 否 | `"ByteRDS"` | 数据库类型 |
| `database` | str | 否 | None | 按数据库名称过滤 |
| `psm` | str | 否 | None | 按 PSM 过滤 |
| `instance_status` | str | 否 | None | 按状态过滤（如 Running） |
| `page_number` | int | 否 | 1 | 页码 |
| `page_size` | int | 否 | 10 | 每页数量 |
| `favor` | bool | 否 | False | 仅查收藏的实例 |
| `owned` | bool | 否 | False | 仅查自己创建的实例 |

```python
from toolbox import create_client, list_instances

client = create_client(vregion="China-North")
result = list_instances(client, database="my_database")
result = list_instances(client, favor=True)
result = list_instances(client, owned=True)
```

返回 `data.instances[]`，关键字段：
- `id` / `name` — 实例标识和名称
- `status` / `type` / `version` — 状态、类型、版本
- `region` / `zone` — 地域和可用区
- `endpoint` / `port` — 连接地址
- `cpu` / `memory` / `storage` — 规格
- `dept` / `owners` / `dbas` / `psm_list` — 归属信息

---

### list_tables()

列出指定数据库下的所有表。

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `client` | DBWClient | **是** | — | 数据库客户端 |
| `database` | str | **是** | — | 数据库名称 |
| `db_type` | str | 否 | None | 数据库类型 |
| `page_number` | int | 否 | 1 | 页码 |
| `page_size` | int | 否 | 50 | 每页数量 |
| `fetch_all` | bool | 否 | False | 自动翻页获取全部表 |

```python
from toolbox import create_client, list_tables

client = create_client(vregion="China-North")
result = list_tables(client, database="my_database", fetch_all=True)
```

返回 `data.tables[]` 表名列表，`data.total` 总数。

---

### get_table_info()

获取表的字段定义、类型、注释和建表语句。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `client` | DBWClient | **是** | 数据库客户端 |
| `table` | str | **是** | 表名 |
| `database` | str | **是** | 数据库名称 |
| `db_type` | str | 否 | 数据库类型 |

```python
from toolbox import create_client, get_table_info

client = create_client(vregion="China-North")
result = get_table_info(client, table="users", database="my_database")
```

### 返回

```json
{
  "success": true,
  "data": {
    "name": "users",
    "engine": "InnoDB",
    "charset": "utf8mb4",
    "definition": "CREATE TABLE `users` (...)",
    "columns": [
      {
        "name": "id",
        "type": "bigint",
        "length": "20",
        "nullable": false,
        "primary_key": false,
        "auto_increment": true,
        "default": null,
        "comment": "用户ID"
      }
    ]
  }
}
```

> Agent 应使用 `data.columns` 的字段名和注释来编写精确 SQL。

---

### search_cached_instances()

搜索本地缓存的数据库列表。**不需要 `client`**，不调用远程 API。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `keyword` | str | 否 | 按数据库名模糊搜索（子串匹配，不区分大小写） |
| `vregion` | str | 否 | 按虚拟地域精确过滤 |

```python
from toolbox import search_cached_instances

result = search_cached_instances()                            # 列出全部缓存
result = search_cached_instances(keyword="order")             # 按关键词搜索
result = search_cached_instances(keyword="user", vregion="China-North")  # 组合过滤
```

返回 `data.instances[]`，每项含 `database`、`vregion`。

---

### set_global_config()

设置全局配置。**不需要 `client`**。

| 参数 | 类型 | 说明 |
|------|------|------|
| `default_database` | str | 默认数据库名称 |
| `default_vregion` | str | 默认虚拟地域 |
| `default_db_type` | str | 默认数据库类型 |

```python
from toolbox import set_global_config

result = set_global_config(
    default_database="my_database",
    default_vregion="China-North",
    default_db_type="ByteRDS"
)
```

---

### get_ticket_url()

获取 ByteRDS 工单创建页面链接。

| 触发场景 | 推荐工单类型 |
|------|-------------|
| 用户想执行 INSERT/UPDATE/DELETE | DML 工单 |
| 慢查询建议加索引、用户想改表结构 | DDL 工单 |
| 表空间分析发现大表需要清理 | 清表工单 |
| 诊断后需调整数据库参数 | 参数修改 |
| CPU/内存等资源不足 | 规格升降配 |

> **⚠️ 工单涉及 SQL 语句时，需先生成可用 SQL，再引导用户将 SQL 填入工单。**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `client` | ToolboxClient | **是** | 数据库客户端 |
| `database` | str | **是** | 数据库名称 |

```python
from toolbox import create_client, get_ticket_url

client = create_client(vregion="China-North")
result = get_ticket_url(client, database="mydb")
# result["data"]["url"] → 工单页面链接
# result["data"]["ticket_types"] → ["DML 工单（UPDATE/DELETE/INSERT）", "DDL（新建/修改表）", ...]
```

---

## 数据查询

两种方式可选，根据场景判断：

**nl2sql**：`list_tables() → nl2sql(query, tables=[...]) → execute_sql/query_sql`
步骤少、速度快，但生成的 SQL 可能出现字段名偏差或条件遗漏。

**查询 schema 后自写 SQL**：`list_tables() → get_table_info() → 根据 schema 自行编写 SQL → execute_sql/query_sql`
步骤多，但能看到真实字段名和注释，SQL 更精准。

### nl2sql()

用自然语言描述查询需求，自动生成 SQL。生成后需用 `execute_sql()` 或 `query_sql()` 执行。

> ⚠️ 仅 China-North, China-East, China-BOE, China-North5, China-Fintech 可用。

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `client` | DBWClient | **是** | — | 数据库客户端 |
| `query` | str | **是** | — | 自然语言查询描述 |
| `database` | str | **是** | — | 数据库名称 |
| `db_type` | str | 否 | None | 数据库类型 |
| `tables` | List[str] | 否 | None | 指定涉及的表名，生成更精准 |

```python
from toolbox import create_client, nl2sql, execute_sql

client = create_client(vregion="China-North")
result = nl2sql(client, query="查询最近一周的销售额",
                database="mydb", tables=["orders"])
if result["success"]:
    sql = result["data"]["sql"]
    data = execute_sql(client, sql=sql, database="mydb")
```

返回 `data.sql` — 生成的 SQL 语句。

---

### execute_sql()

执行只读 SQL，返回结构化 dict 结果。一次只能执行一条 SQL。

> ⚠️ **行数上限**：平台最多返回 **3000 行**。查询大表时务必通过以下方式控制数据量：
> - 聚合分析用 `GROUP BY` + `COUNT/SUM/AVG`（不受行数限制影响）
> - 取样查询加 `LIMIT n`（建议 ≤ 500）
> - 范围筛选用 `WHERE id > x LIMIT n`（翻页扫描）
> - 避免 `SELECT * FROM large_table`，超出行数时返回截断数据

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `client` | DBWClient | **是** | 数据库客户端 |
| `sql` | str | **是** | SQL 语句（一次一条） |
| `database` | str | **是** | 数据库名称 |
| `db_type` | str | 否 | 数据库类型 |

```python
from toolbox import create_client, execute_sql

client = create_client(vregion="China-North")
result = execute_sql(client, sql="SELECT tenant_id, COUNT(*) as cnt FROM my_table GROUP BY tenant_id", database="mydb")
result = execute_sql(client, sql="SHOW TABLES", database="mydb")
result = execute_sql(client, sql="EXPLAIN SELECT * FROM users WHERE id=1", database="mydb")
```

### 返回（成功）

```json
{
  "success": true,
  "data": {
    "command_str": "SELECT tenant_id, COUNT(*) ...",
    "state": "Success",
    "row_count": 5,
    "columns": ["tenant_id", "cnt"],
    "rows": [
      {"Cells": ["tenant_a", "120"]},
      {"Cells": ["tenant_b", "85"]}
    ]
  }
}
```

> **注意**：`rows` 中每行是 `{"Cells": [...]}` 格式，值都是字符串。

### 返回（失败）

```json
{
  "success": false,
  "message": "SQL被安全拦截：仅允许SELECT/SHOW TABLES/SHOW CREATE TABLE/EXPLAIN",
  "error": {
    "state": "failed",
    "reason_detail": "具体错误原因"
  }
}
```

---

### query_sql()

执行 SELECT/SHOW 查询并返回 pandas DataFrame，适合数据分析和统计计算。

> ⚠️ **行数上限**：同 `execute_sql`，最多返回 **3000 行**。大表分析建议先用聚合 SQL 统计，或加 `LIMIT` 取样后用 pandas 本地计算。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `client` | DBWClient | **是** | 数据库客户端 |
| `sql` | str | **是** | SQL 语句（SELECT/SHOW） |
| `database` | str | **是** | 数据库名称 |
| `db_type` | str | 否 | 数据库类型 |

```python
from toolbox import create_client, query_sql

client = create_client(vregion="China-North")
df = query_sql(client, sql="SELECT * FROM users LIMIT 100", database="mydb")
df = query_sql(client, sql="SELECT status, COUNT(*) as cnt FROM orders GROUP BY status", database="mydb")
```

### execute_sql vs query_sql

| | execute_sql | query_sql |
|---|---|---|
| 返回类型 | dict（`rows` 为 `[{"Cells": [...]}, ...]`） | DataFrame |
| 支持的 SQL | SELECT/SHOW/EXPLAIN | SELECT/SHOW |
| 依赖 | 无 | 需要 pandas |
| 适用场景 | 通用查询、EXPLAIN、SHOW CREATE TABLE | 数据分析、统计计算 |

---

## 可视化与报告

分析完数据后，可以生成 HTML 报告。详细风格规范参考：
- [`report-style.md`](../../analysis/report-style.md) — 报告风格（Financial Times / McKinsey / Economist 等）
- [`html-templates.md`](../../analysis/html-templates.md) — HTML 模板和组件

### 截图命令

```bash
npx playwright screenshot "file:///path/to/report.html" output.png \
  --viewport-size=1200,675 --wait-for-timeout=2000
```

---

## 常见场景

| 用户说法 | 调用 |
|---------|------|
| "有哪些数据库/实例" | `list_instances(client)` |
| "我收藏的数据库" | `list_instances(client, favor=True)` — 远程 API |
| "之前用过哪些数据库" | `search_cached_instances()` — 本地历史 |
| "这个库里有哪些表" | `list_tables(client, database="x", fetch_all=True)` |
| "表结构是什么" | `get_table_info(client, table="xxx", database="x")` |
| "我想改数据" / "帮我加索引" | `get_ticket_url(client, database="x")` → 工单 |
