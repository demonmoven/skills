# ByteRDS 表空间问题排查

## 概述

表空间问题通常表现为磁盘使用率上升、碎片率高或单表数据量过大，影响性能和可用空间。

## 典型症状

- 磁盘使用率持续上升
- 单表数据量异常增长
- 查询性能随数据量增大而下降
- `describe_health_summary` 显示磁盘相关指标异常

## 排查步骤

> **必须按顺序尝试所有步骤**，不要只执行步骤 1 就直接给结论。步骤 2、3 提供写入来源和 DML 趋势等关键诊断信息。仅当某步返回不支持时才跳过。
>
> 函数参数详见 [api/byterds/ops.md](../../api/byterds/ops.md)。

### 步骤 1：查看表空间详情

```python
from toolbox import create_client, describe_table_space

client = create_client(vregion="China-North")
result = describe_table_space(client, database="mydb", filter_database="mydb")
# 查看数据量、碎片率、表空间大小
```

### 步骤 2：分析写入来源（必须尝试）

> **步骤 1 完成后必须执行此步**，定位哪些表写入最频繁，这是判断空间增长根因的关键。

```python
from toolbox import table_write_analysis

result = table_write_analysis(
    client,
    start_time="2026-03-08 00:00:00",
    end_time="2026-03-09 00:00:00",
    database="mydb",
    order_by="DmlExecCount",
    sort_by="DESC"
)
# 定位写入最频繁的表
```

> ⚠️ 依赖全量 SQL 开启，China-East 不支持。调用失败时跳过即可。

### 步骤 3：查看表级监控

```python
from toolbox import describe_table_metric

result = describe_table_metric(
    client,
    item_type="DML",
    db_name="mydb",
    table="large_table",
    start_time="2026-03-08 00:00:00",
    end_time="2026-03-09 00:00:00",
    database="mydb"
)
# 查看 DML 执行频率、影响行数
```

> ⚠️ 依赖全量 SQL 开启，China-East 不支持。

### 辅助：确认整体状况

```python
from toolbox import describe_health_summary

result = describe_health_summary(
    client,
    start_time="2026-03-08 10:00:00",
    end_time="2026-03-08 10:15:00",
    database="mydb"
)
```

## 常见根因

| 根因 | 说明 |
|:---|:---|
| 日志表未清理 | 业务日志表持续写入，未做归档/清理 |
| 碎片率高 | 频繁 DELETE 后未 OPTIMIZE TABLE |
| 大字段存储 | TEXT/BLOB 字段存储大量数据 |
| 分区缺失 | 大表未分区，单表过大 |

## 应急处置

```python
from toolbox import get_ticket_url

result = get_ticket_url(client, database="mydb")
# 提交清表工单（归档旧数据）/ DDL 工单（OPTIMIZE TABLE 回收碎片）
```

## 预防建议

1. 对日志类表设置定期清理策略
2. 大表使用分区
3. 定期监控表空间增长趋势
4. 碎片率超过 30% 时考虑 OPTIMIZE TABLE
