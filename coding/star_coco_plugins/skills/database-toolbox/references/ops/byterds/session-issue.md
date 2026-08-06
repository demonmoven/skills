# ByteRDS 连接/会话问题排查

## 概述

连接/会话问题通常表现为连接数过高、大量空闲连接占用资源、或连接池耗尽导致新连接无法建立。

## 典型症状

- 应用报 `Too many connections` 错误
- 大量 Sleep 状态连接
- 活跃连接数异常增多
- 新连接建立超时

## 排查步骤

> 函数参数详见 [api/byterds/ops.md](../../api/byterds/ops.md)。

### 步骤 1：查看全部连接

```python
from toolbox import create_client, list_active_sessions

client = create_client(vregion="China-North")
result = list_active_sessions(client, database="mydb", show_sleep=True)
# show_sleep=True 包含 Sleep 连接
# 关注：总连接数、Sleep 连接占比、执行时间最长的连接
```

### 步骤 2：确认整体资源状况

```python
from toolbox import describe_health_summary

result = describe_health_summary(
    client,
    start_time="2026-03-08 10:00:00",
    end_time="2026-03-08 10:15:00",
    database="mydb"
)
# 关注连接数、CPU、内存等指标
```

## 常见根因

| 根因 | 说明 |
|:---|:---|
| 连接池配置不当 | 最大连接数设置过小，或空闲超时过长 |
| 连接泄漏 | 应用未正确关闭连接 |
| 慢查询堆积 | 大量慢查询占用连接不释放 |
| 突发流量 | 业务流量突增超出连接池容量 |

## 应急处置

```python
from toolbox import get_ticket_url

result = get_ticket_url(client, database="mydb")
# 提交工单：kill 空闲连接 / 调整 max_connections / 扩容
```

## 预防建议

1. 合理配置连接池参数（最大连接数、空闲超时）
2. 确保应用代码正确关闭连接（使用 try-finally 或连接池管理）
3. 监控连接数指标，设置告警
4. 设置合理的 `wait_timeout` 清理空闲连接
