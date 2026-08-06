# ByteRedis 大 Key 分析

## 概述

大 Key 是指 Value 占用内存过大的 Key，可能导致慢查询、内存倾斜、主从同步延迟等问题。

> ⚠️ `redis_list_big_keys()` 在 China-BOE 不支持。

## 典型症状

- Redis 内存使用率高
- 部分节点内存不均匀（数据倾斜）
- 特定命令执行缓慢（HGETALL、DEL 大 Key）
- 主从同步延迟

## 排查步骤

> 函数参数详见 [api/byteredis.md](../../api/byteredis.md)。

### 步骤 1：扫描大 Key

```python
from toolbox import create_client, redis_list_big_keys

client = create_client(vregion="China-North")

# 查询今天的大 Key
result = redis_list_big_keys(
    client,
    psm="toutiao.redis.explorer",
    date="2026-03-10"
)

# 按 Key 类型过滤
result = redis_list_big_keys(
    client,
    psm="toutiao.redis.explorer",
    date="2026-03-10",
    key_type="hash",
    page_size=20
)
```

## 常见根因

| 根因 | 说明 |
|:---|:---|
| Hash 字段过多 | 单个 Hash Key 包含数万个 field |
| List/Set 元素过多 | 单个 List/Set 包含大量元素 |
| String Value 过大 | 单个 String 存储大 JSON 或二进制数据 |
| 未设过期时间 | 数据持续累积不清理 |

## 应急处置

```python
from toolbox import get_ticket_url

result = get_ticket_url(client, psm="toutiao.redis.explorer", db_type="ByteRedis")
# 提交工单：删 Key（注意大 Key 需分批删除）/ 扩容
```

## 预防建议

1. Hash Key 控制 field 数量（建议 < 5000）
2. List/Set 控制元素数量，超量时拆分
3. 大 JSON 考虑压缩后存储
4. 设置合理的过期时间
5. 定期扫描大 Key 并及时治理
