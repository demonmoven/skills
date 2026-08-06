# ByteDoc 慢查询排查

## 概述

ByteDoc 慢查询排查流程。ByteDoc 无聚合慢查询接口（`describe_aggregate_slow_logs`）、无慢查询趋势接口（`slow_query_trend`）、无优化建议接口（`list_slow_query_advice`），直接查看慢查询明细。

## 典型症状

- MongoDB 查询响应变慢
- 慢查询日志增加
- 应用超时

## 排查步骤

> 函数参数详见 [api/bytedoc.md](../../api/bytedoc.md)。

### 步骤 1：查看慢查询明细

```python
from toolbox import create_client, describe_slow_logs

client = create_client(vregion="China-North")
result = describe_slow_logs(
    client,
    start_time="2026-03-08 00:00:00",
    end_time="2026-03-08 01:00:00",
    database="my_doc_db",
    db_type="ByteDoc"
)
# 按 query_time 排序，定位最慢的查询
# 关注 rows_scanned vs rows_sent（全集合扫描特征：rows_scanned >> rows_sent）
# 关注 index_scan 是否为 0（0 表示未使用索引）
```

### 步骤 2：检查当前活跃连接

```python
from toolbox import list_active_sessions

result = list_active_sessions(client, database="my_doc_db", db_type="ByteDoc")
# 查看是否有长时间运行的操作
```

### 步骤 3：确认节点状态

```python
from toolbox import describe_instance_nodes

result = describe_instance_nodes(client, database="my_doc_db", db_type="ByteDoc")
# 确认 Primary/Secondary 节点状态
```

### 步骤 4：分析查询计划

```python
from toolbox import execute_sql

result = execute_sql(
    client,
    sql="db.my_collection.find({status: 'active'}).explain()",
    database="my_doc_db",
    db_type="ByteDoc"
)
# 关注 queryPlanner.winningPlan 是否使用了索引
```

## 常见根因

| 根因 | 说明 |
|:---|:---|
| 缺少索引 | 查询未命中索引，导致全集合扫描 |
| 查询条件不优 | 正则查询、$nin 等低效操作 |
| 大文档 | 返回文档过大，网络传输慢 |
| 集合数据量大 | 数据增长未及时建索引 |

## 应急处置

ByteDoc 建索引需通过 MongoDB 工单处理，不可直接在线执行 DDL。

## 预防建议

1. 对高频查询的过滤字段建立索引
2. 使用 `explain()` 验证查询是否命中索引
3. 避免全集合扫描（`find({})` 不带条件）
4. 大文档考虑使用投影（projection）只返回需要的字段
