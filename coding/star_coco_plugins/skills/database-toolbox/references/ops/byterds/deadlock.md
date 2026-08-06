# ByteRDS 死锁排查

## 概述

死锁是两个或多个事务互相等待对方持有的锁，导致所有事务都无法继续执行。MySQL 会自动检测并回滚其中一个事务。

## 典型症状

- 应用日志出现 `Deadlock found when trying to get lock` 错误
- 部分事务被自动回滚
- 业务偶发性失败

## 排查步骤

> 函数参数详见 [api/byterds/ops.md](../../api/byterds/ops.md)。

### 步骤 1：查看死锁详情

```python
from toolbox import create_client, describe_deadlock

client = create_client(vregion="China-North")
result = describe_deadlock(client, database="mydb")
# 查看等待链、涉及的 SQL、锁类型
```

### 步骤 2：查看当前事务和锁关系

```python
from toolbox import list_transactions

result = list_transactions(client, database="mydb")
# 关注长事务、持锁事务
```

### 步骤 3：定位持锁会话

```python
from toolbox import list_active_sessions

result = list_active_sessions(client, database="mydb")
# 按 time 降序排列，关注执行时间最长的连接
```

## 常见根因

| 根因 | 说明 |
|:---|:---|
| 事务顺序不一致 | 不同事务以不同顺序访问相同资源 |
| 长事务 | 事务持锁时间过长，增大死锁概率 |
| 大范围锁定 | UPDATE/DELETE 涉及大量行，锁定范围过大 |
| 缺少索引 | 无合适索引导致锁升级（行锁→表锁） |

## 应急处置

```python
from toolbox import get_ticket_url

result = get_ticket_url(client, database="mydb")
# 提交工单让 DBA 处理（kill 问题会话 / 调整参数）
```

## 预防建议

1. 统一事务中的表访问顺序
2. 缩短事务执行时间，尽快提交
3. 为 WHERE 条件字段建索引，减少锁范围
4. 避免在事务中执行用户交互等耗时操作
