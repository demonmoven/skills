---
name: "database-skill-byteredis"
description: "ByteRedis 全部 API：服务搜索与工单、Redis 只读查询、慢日志与大 Key 分析。"
---

# ByteRedis API 参考

ByteRedis（Redis 缓存服务）的全部 Toolbox 方法。传 `db_type="ByteRedis"` 使用。

> **参数说明**：`psm` 传 PSM（如 `toutiao.redis.explorer`）。
>
> **不适用的函数**：`list_tables`、`get_table_info`、`nl2sql` 不适用于 Redis。Redis 是 Key-Value 存储，没有表和列的概念。需要查看数据时直接使用 `execute_sql()` 执行 Redis 只读命令。

## 方法一览

| 方法 | 分类 | 说明 |
|------|------|------|
| `list_instances()` | 元数据 | 搜索 Redis 服务列表（按 PSM 或收藏） |
| `get_ticket_url()` | 元数据 | Redis 工单页面链接（扩容/缩容/执行命令/删Key） |
| `execute_sql()` | 数据查询 | Redis 只读查询命令 |
| `describe_slow_logs()` | 运维诊断 | Redis 慢日志 |
| `redis_list_big_keys()` | 运维诊断 | 大 Key 分析（⚠️ China-BOE 不支持） |

---

## 元数据

### list_instances()

搜索 Redis 服务列表。

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `client` | ToolboxClient | **是** | — | 客户端 |
| `db_type` | str | **是** | — | 必须传 `"ByteRedis"` |
| `psm` | str | 否 | None | PSM 关键字搜索 |
| `favor` | bool | 否 | False | 仅查收藏 |
| `page_number` | int | 否 | 1 | 页码 |
| `page_size` | int | 否 | 10 | 每页数量 |

```python
from toolbox import create_client, list_instances

client = create_client(vregion="China-North")
result = list_instances(client, db_type="ByteRedis", psm="toutiao.redis.explorer")
result = list_instances(client, db_type="ByteRedis", favor=True)
```

> `psm` 在 ByteRedis 场景下是 PSM（如 `toutiao.redis.explorer`），不是数据库实例名。

---

### get_ticket_url()

获取 Redis 工单页面链接。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `client` | ToolboxClient | **是** | 客户端 |
| `psm` | str | **是** | PSM |
| `db_type` | str | **是** | 必须传 `"ByteRedis"` |

| 触发场景 | 推荐工单类型 |
|------|-------------|
| Redis 容量不足 | 扩容 |
| Redis 资源闲置 | 缩容 |
| 需要执行管理命令 | 执行命令 |
| 需要清理大 Key | 删Key |

```python
result = get_ticket_url(client, psm="toutiao.redis.explorer", db_type="ByteRedis")
# result["data"]["url"] → 工单页面链接
# result["data"]["ticket_types"] → ["扩容", "缩容", "执行命令", "删Key"]
```

---

## 数据查询

### execute_sql()

执行 Redis 只读查询命令。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `client` | ToolboxClient | **是** | 客户端 |
| `sql` | str | **是** | Redis 命令字符串（如 `GET mykey`），自动拆分为 command + args |
| `psm` | str | **是** | PSM |
| `db_type` | str | **是** | 必须传 `"ByteRedis"` |

#### sql 参数解析

| sql 值 | command | args |
|--------|---------|------|
| `"GET mykey"` | `GET` | `mykey` |
| `"HGETALL myhash"` | `HGETALL` | `myhash` |
| `"ZRANGEBYSCORE myzset 0 100"` | `ZRANGEBYSCORE` | `myzset 0 100` |
| `"LRANGE mylist 0 -1"` | `LRANGE` | `mylist 0 -1` |

#### 支持的命令

**仅支持只读查询命令**：

| 类别 | 常用命令 |
|------|---------|
| Key | `EXISTS`, `TTL`, `PTTL`, `TYPE` |
| String | `GET`, `GETRANGE`, `MGET`, `STRLEN`, `CGET` |
| Hash | `HGET`, `HGETALL`, `HMGET`, `HLEN`, `HEXISTS`, `HSTRLEN` |
| List | `LRANGE`, `LLEN`, `LINDEX` |
| Set | `SMEMBERS`, `SCARD`, `SRANDMEMBER` |
| Sorted Set | `ZRANGE`, `ZRANGEBYSCORE`, `ZREVRANGE`, `ZREVRANGEBYSCORE`, `ZSCORE`, `ZCARD`, `ZCOUNT`, `ZRANK`, `ZREVRANK` |
| Bloom Filter | `BF.EXISTS`, `BF.INFO`, `BF.MEXISTS` |
| TairHash | `EXHGET`, `EXHGETALL`, `EXHMGET`, `EXHLEN`, `EXHKEYS`, `EXHVALS`, `EXHSCAN` |
| TairZset | `EXZRANGE`, `EXZRANGEBYSCORE`, `EXZREVRANGE`, `EXZSCORE`, `EXZCARD` |

**不支持的命令**：所有写入（SET/DEL/EXPIRE 等）、扫描（KEYS/SCAN 等）、管理（INFO/PING/CONFIG 等）、阻塞/事务/Pub/Sub。

```python
from toolbox import create_client, execute_sql

client = create_client(vregion="China-North")

result = execute_sql(client, sql="GET mykey",
                     psm="toutiao.redis.dbw_test", db_type="ByteRedis")

result = execute_sql(client, sql="HGETALL myhash",
                     psm="toutiao.redis.dbw_test", db_type="ByteRedis")

result = execute_sql(client, sql="ZRANGEBYSCORE myzset -inf +inf",
                     psm="toutiao.redis.dbw_test", db_type="ByteRedis")
```

> 返回格式与 MySQL 不同，直接返回 JSON 响应。仅支持只读查询命令，写入/删除/管理命令会在发送前拦截并返回明确错误。

---

## 运维诊断

### describe_slow_logs()

查询 Redis 慢日志。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `client` | ToolboxClient | **是** | 客户端 |
| `start_time` | str | **是** | 开始时间 |
| `end_time` | str | **是** | 结束时间 |
| `psm` | str | **是** | PSM |
| `db_type` | str | **是** | 必须传 `"ByteRedis"` |

```python
from toolbox import create_client, describe_slow_logs

client = create_client(vregion="China-North")
result = describe_slow_logs(client, start_time="2026-03-09 00:00:00",
    end_time="2026-03-10 00:00:00", psm="toutiao.redis.explorer", db_type="ByteRedis")
```

> 自动获取全部慢日志，无需手动翻页。

---

### redis_list_big_keys()

Redis 大 Key 分析。⚠️ China-BOE 不支持。

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `client` | ToolboxClient | **是** | — | 客户端 |
| `psm` | str | **是** | — | PSM |
| `date` | str | **是** | — | 日期（`YYYY-MM-DD`） |
| `begin` | str | 否 | `"00:00:00"` | 开始时间 |
| `end` | str | 否 | `"23:59:59"` | 结束时间 |
| `page` | int | 否 | 1 | 页码 |
| `page_size` | int | 否 | 10 | 每页数量 |
| `key_type` | str | 否 | `"string"` | Key 类型过滤 |
| `db_type` | str | 否 | `"ByteRedis"` | 数据库类型 |

```python
from toolbox import create_client, redis_list_big_keys

client = create_client(vregion="China-North")
result = redis_list_big_keys(client, psm="toutiao.redis.explorer", date="2026-03-10")
result = redis_list_big_keys(client, psm="toutiao.redis.explorer", date="2026-03-10",
                              key_type="hash", page_size=20)
```
