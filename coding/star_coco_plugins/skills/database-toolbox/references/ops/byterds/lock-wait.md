# ByteRDS 锁等待排查

## 概述

锁等待是指一个事务因等待另一个事务释放锁而被阻塞。与死锁不同，锁等待不会自动解除，可能导致大量连接堆积。

## 典型症状

- 查询长时间无响应
- 连接数快速增长
- 应用出现超时错误
- `list_active_sessions` 显示大量 `Locked` 状态连接

## 排查步骤

> 函数参数详见 [api/byterds/ops.md](../../api/byterds/ops.md)。

### 步骤 1：查看事务和锁列表

```python
from toolbox import create_client, list_transactions

client = create_client(vregion="China-North")
result = list_transactions(client, database="mydb")
# 找到持锁时间最长的事务
```

### 步骤 2：获取事务快照（锁等待链）

```python
from toolbox import transaction_snapshots

result = transaction_snapshots(
    client,
    start_time="2026-03-08 00:00:00",
    end_time="2026-03-09 00:00:00",
    database="mydb"
)
# 查看锁等待链：哪个事务阻塞了哪个事务
```

### 步骤 3：定位阻塞源

```python
from toolbox import list_active_sessions

result = list_active_sessions(client, database="mydb")
# 按 time 降序，找到执行时间最长的连接
# 关注 blocking_pid 字段（被谁阻塞）
```

### 步骤 4：导出事务数据做离线分析

```python
from toolbox import export_transactions

result = export_transactions(
    client,
    task_name="lock_wait_analysis_20260308",
    start_time=1741363200,
    end_time=1741449600,
    database="mydb"
)
```

## 常见根因

| 根因 | 说明 |
|:---|:---|
| 长事务未提交 | 事务开启后长时间未 commit/rollback，持有行锁 |
| 大批量 DML | 单条 SQL 锁定大量行 |
| DDL 等待 MDL 锁 | ALTER TABLE 等 DDL 需要 MDL 写锁，被长查询阻塞 |
| 热点行并发 | 多个事务同时更新同一行 |

## 应急处置

锁等待问题需要人工介入处理（kill 阻塞源会话、调整参数等），建议联系 DBA 协助。

## 预防建议

1. 缩短事务执行时间，及时提交
2. 避免单条 SQL 影响过多行
3. DDL 变更选择低峰期执行
4. 设置合理的 `innodb_lock_wait_timeout`
