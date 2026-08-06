# ByteRDS 慢查询排查

## 概述

慢查询是指执行时间超过阈值的 SQL 语句，可能由缺少索引、SQL 写法不当、数据量过大等原因导致。

## 典型症状

- 查询响应时间变长
- 慢查询日志量增加
- 特定页面或接口变慢

## 排查步骤

> **必须按顺序尝试所有步骤**，不要只执行前几步就直接给结论。每一步都可能提供关键信息。仅当某步明确标注"按需"或返回不支持时才跳过。
>
> 函数参数详见 [api/byterds/ops.md](../../api/byterds/ops.md)。

### 步骤 1：确认整体状况

```python
from toolbox import create_client, describe_health_summary

client = create_client(vregion="China-North")
result = describe_health_summary(
    client,
    start_time="2026-03-08 10:00:00",
    end_time="2026-03-08 10:15:00",
    database="mydb"
)
# 关注 CPU 利用率、连接数等指标是否异常
```

### 步骤 2：定位高频/高耗时 SQL 模板

```python
from toolbox import describe_aggregate_slow_logs

result = describe_aggregate_slow_logs(
    client,
    start_time="2026-03-08 00:00:00",
    end_time="2026-03-08 01:00:00",
    database="mydb",
    order_by="TotalQueryTime"  # 按总耗时排序
)
# 返回字段因 vregion 而异：
# - 有 sql_template_id → 可继续步骤 3 查明细
# - 有 example_sql → 已包含实际 SQL 样本，跳过步骤 3
```

### 步骤 3：查看具体 SQL 文本（按需）

> **必须检查**步骤 2 的返回：若有 `sql_template_id`，用 `describe_slow_logs` 按模板过滤查看明细；若有 `example_sql` 则跳过此步。

```python
from toolbox import describe_slow_logs

logs = result["data"]["logs"]
if logs and "sql_template_id" in logs[0]:
    tid = logs[0]["sql_template_id"]
    detail = describe_slow_logs(
        client,
        start_time="2026-03-08 00:00:00",
        end_time="2026-03-08 01:00:00",
        database="mydb",
        sql_template_id=tid,
        page_size=20
    )
```

### 步骤 4：定位高峰时段（按需）

> `slow_query_trend` 部分 vregion 不支持，调用失败时跳过即可。

```python
from toolbox import slow_query_trend

trend = slow_query_trend(
    client,
    start_time="2026-03-08 00:00:00",
    end_time="2026-03-08 01:00:00",
    database="mydb",
    interval=60  # 每分钟统计
)
```

### 步骤 5：获取优化建议（⚠️ cn only，发现慢查询时必须尝试）

> **发现慢查询后必须执行此步**，不要跳过直接给手动分析。即使没有历史任务，也要调用确认。

```python
from toolbox import slow_query_advice_task_history, list_slow_query_advice

# 5a: 查看诊断任务历史
history = slow_query_advice_task_history(client, database="mydb")
tasks = history["data"]["tasks"]
success_tasks = [t for t in tasks if t["status"] == "SUCCESS"]

if success_tasks:
    sid = success_tasks[0]["id"]

    # 5b: 获取索引建议
    advice = list_slow_query_advice(
        client, database="mydb",
        summary_id=sid,
        advice_type="index_advice",
        group_by="Advice"
    )
```

> `slow_query_advice_task_history` 和 `list_slow_query_advice` 仅 cn SITE 可用，China-BOE 不支持。调用失败时跳过即可。

### 步骤 6：分析执行计划

```python
from toolbox import execute_sql

result = execute_sql(
    client,
    sql="EXPLAIN SELECT * FROM orders WHERE status=1",
    database="mydb"
)
# 关注 type（ALL=全表扫描）、key（使用的索引）、rows（扫描行数）
```

## 常见根因

| 根因 | 说明 | 判断依据 |
|:---|:---|:---|
| 缺少索引 | 全表扫描 | EXPLAIN type=ALL，rows 很大 |
| 索引选择错误 | 用了低效索引 | EXPLAIN key 不是最优索引 |
| 大表扫描 | 数据量大，扫描行数多 | rows_examined 远大于 rows_sent |
| 复杂 JOIN | 多表 JOIN | EXPLAIN 多行，嵌套循环 |
| 锁等待 | 被其他事务阻塞 | lock_time 较大 |

## 应急处置

字节云无 `kill_process()` 接口，需通过工单处理：

```python
from toolbox import get_ticket_url

result = get_ticket_url(client, database="mydb")
# 引导用户提交 DDL 工单（加索引）或参数修改工单
```

## 预防建议

1. 定期查看慢查询聚合统计，关注 Top SQL
2. 新 SQL 上线前用 EXPLAIN 验证执行计划
3. 对高频查询确保有合适的索引覆盖
4. 避免 SELECT *，只查需要的字段
5. 大表查询加 LIMIT
