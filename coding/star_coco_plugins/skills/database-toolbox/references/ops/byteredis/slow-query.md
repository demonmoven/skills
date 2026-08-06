# ByteRedis 慢查询排查

## 概述

Redis 慢查询是指执行时间超过 `slowlog-log-slower-than` 阈值的命令。ByteRedis 无聚合接口，直接查看慢日志明细。

## 典型症状

- Redis 响应时间变长
- 慢日志增加
- 客户端超时

## 排查步骤

> 函数参数详见 [api/byteredis.md](../../api/byteredis.md)。

### 步骤 1：查看慢日志

```python
from toolbox import create_client, describe_slow_logs

client = create_client(vregion="China-North")
result = describe_slow_logs(
    client,
    start_time="2026-03-09 00:00:00",
    end_time="2026-03-10 00:00:00",
    psm="toutiao.redis.explorer",
    db_type="ByteRedis"
)
# 自动获取全部慢日志，无需手动翻页
# 关注：命令类型、Key 名称、耗时
```

## 常见根因

| 根因 | 说明 |
|:---|:---|
| 大 Key 操作 | HGETALL/SMEMBERS 等命令操作大 Key，单次返回数据量大 |
| 复杂命令 | SORT、KEYS *、SUNION 等 O(N) 命令 |
| 热点 Key | 单个 Key 高并发访问 |
| 大批量操作 | MGET/MSET 传入过多 Key |

## 应急处置

```python
from toolbox import get_ticket_url

result = get_ticket_url(client, psm="toutiao.redis.explorer", db_type="ByteRedis")
# 提交工单：删大 Key / 扩容 / 执行管理命令
```

## 后续排查

如果慢查询根因是大 Key，可用 `redis_list_big_keys()` 进一步分析，详见 [big-key.md](./big-key.md)。

## 预防建议

1. 避免使用 O(N) 复杂度命令（KEYS、SORT 等）
2. 大 Key 拆分为多个小 Key
3. 批量操作控制单次数量（如 MGET 不超过 100 个 Key）
4. 热点 Key 考虑本地缓存或读写分离
