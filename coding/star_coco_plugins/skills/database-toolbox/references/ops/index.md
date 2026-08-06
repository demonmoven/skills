# 运维排查场景索引

根据 db_type 和问题场景，路由到对应的排查 SOP。

## 通用入口

无论哪种场景，都可先用 `describe_health_summary()` 获取实例整体健康概览：

```python
from toolbox import create_client, describe_health_summary

client = create_client(vregion="China-North")
result = describe_health_summary(
    client,
    start_time="2026-03-08 10:00:00",
    end_time="2026-03-08 10:15:00",
    database="mydb"
)
```

> 函数参数详见 [api/byterds/ops.md](../api/byterds/ops.md)。

## 场景路由表

### ByteRDS

| 优先级 | 场景 | SOP 文件 | SITE 限制 |
|:---|:---|:---|:---|
| P1 | 慢查询 | [slow-query.md](./byterds/slow-query.md) | cn, boe（部分函数 cn only） |
| P1 | 死锁 | [deadlock.md](./byterds/deadlock.md) | cn, boe |
| P1 | 锁等待 | [lock-wait.md](./byterds/lock-wait.md) | cn, boe |
| P2 | 表空间问题 | [table-space.md](./byterds/table-space.md) | cn, boe（China-East 部分不支持） |
| P2 | 连接/会话问题 | [session-issue.md](./byterds/session-issue.md) | cn, boe |

### ByteDoc

| 优先级 | 场景 | SOP 文件 | 说明 |
|:---|:---|:---|:---|
| P1 | 慢查询 | [slow-query.md](./bytedoc/slow-query.md) | 无聚合/趋势接口，直接查明细 |

### ByteRedis

| 优先级 | 场景 | SOP 文件 | 说明 |
|:---|:---|:---|:---|
| P1 | 慢查询 | [slow-query.md](./byteredis/slow-query.md) | 直接查明细 |
| P2 | 大 Key | [big-key.md](./byteredis/big-key.md) | ⚠️ China-BOE 不支持 |

## 不支持的场景

以下场景当前函数不足以支撑完整排查，不创建 SOP：

- **CPU/内存/磁盘压力** — 无 `get_metric_data()` 时序监控（仅有 `describe_health_summary` 概览）
- **ByteDoc/ByteRedis 的死锁/锁等待/表空间** — 这些 db_type 无对应函数

## 应急处置

字节云无 `kill_process()` 接口，应急处置统一通过工单：

```python
from toolbox import create_client, get_ticket_url

client = create_client(vregion="China-North")
result = get_ticket_url(client, database="mydb")
# result["data"]["url"] → 工单页面链接
```
