# 排障查询模板集

> 本文件是 `harness-obs-init` 技能的参考资料。
> 提供 VictoriaLogs + VictoriaTraces 的常用查询命令模板。

---

## Table of Contents

- [约定](#约定)
- [1. VictoriaLogs 查询（LogsQL）](#1-victorialogs-查询logsql)
- [2. VictoriaTraces 查询（Jaeger API）](#2-victoriatraces-查询jaeger-api)
- [3. 排障流程模板](#3-排障流程模板)
- [4. noop 模式验证](#4-noop-模式验证)
- [5. LogsQL 速查](#5-logsql-速查)

---

## 约定

- 以下所有模板中 `{ENDPOINT}` 默认为 `http://localhost:9428`（VictoriaLogs）和 `http://localhost:10428`（VictoriaTraces）
- 如果使用团队共享实例，替换为对应 IP
- 占位符用 `<VALUE>` 表示

---

## 1. VictoriaLogs 查询（LogsQL）

### 基础查询

```bash
# 按 action 查询成功日志
curl '{ENDPOINT}/select/logsql/query?query=action:<ACTION>%20AND%20result:success&limit=20'

# 按 action 查询失败日志
curl '{ENDPOINT}/select/logsql/query?query=action:<ACTION>%20AND%20result:failed&limit=20'

# 按 error_kind 查询所有错误
curl '{ENDPOINT}/select/logsql/query?query=error_kind:<ERROR_KIND>&limit=20'

# 按 trace_id 查全链路日志
curl '{ENDPOINT}/select/logsql/query?query=trace_id:<TRACE_ID>&limit=50'
```

### 时间范围查询

```bash
# 最近 5 分钟
curl '{ENDPOINT}/select/logsql/query?query=action:<ACTION>&limit=20&start=5m'

# 最近 1 小时
curl '{ENDPOINT}/select/logsql/query?query=action:<ACTION>&limit=20&start=1h'

# 指定时间范围（ISO 8601）
curl '{ENDPOINT}/select/logsql/query?query=action:<ACTION>&limit=20&start=2025-01-01T00:00:00Z&end=2025-01-01T01:00:00Z'
```

### 组合查询

```bash
# 某服务的所有错误
curl '{ENDPOINT}/select/logsql/query?query=service:<SERVICE_NAME>%20AND%20result:failed&limit=50'

# 某用户的所有操作
curl '{ENDPOINT}/select/logsql/query?query=user_id:<USER_ID>&limit=50'

# 某工作区的操作历史
curl '{ENDPOINT}/select/logsql/query?query=workspace_id:<WS_ID>%20AND%20action:workspace.*&limit=50'

# DB 错误排查
curl '{ENDPOINT}/select/logsql/query?query=error_kind:db_error&limit=20'

# 外部依赖错误
curl '{ENDPOINT}/select/logsql/query?query=error_kind:external_dependency_error&limit=20'
```

### 文本搜索

```bash
# 在 _msg 中搜索关键词
curl '{ENDPOINT}/select/logsql/query?query=_msg:~"timeout"&limit=20'

# 在任意字段中搜索
curl '{ENDPOINT}/select/logsql/query?query=~"connection refused"&limit=20'
```

### 统计查询

```bash
# 按 action 统计日志量
curl '{ENDPOINT}/select/logsql/stats_query?query=*&stats_field=action&stats_func=count'

# 按 error_kind 统计错误分布
curl '{ENDPOINT}/select/logsql/stats_query?query=result:failed&stats_field=error_kind&stats_func=count'

# 按 service 统计日志量
curl '{ENDPOINT}/select/logsql/stats_query?query=*&stats_field=service&stats_func=count'
```

---

## 2. VictoriaTraces 查询（Jaeger API）

### 服务列表

```bash
# 查看已注册的服务
curl '{TRACES_ENDPOINT}/select/jaeger/api/services'
```

### 按 trace ID 查询

```bash
# 查询完整 span 树
curl '{TRACES_ENDPOINT}/select/jaeger/api/traces/<TRACE_ID>'

# JSON 格式化输出（便于阅读）
curl -s '{TRACES_ENDPOINT}/select/jaeger/api/traces/<TRACE_ID>' | python3 -m json.tool
```

### 按条件搜索 trace

```bash
# 按服务名搜索最近 trace
curl '{TRACES_ENDPOINT}/select/jaeger/api/traces?service=<SERVICE_NAME>&limit=10'

# 按操作名搜索
curl '{TRACES_ENDPOINT}/select/jaeger/api/traces?service=<SERVICE_NAME>&operation=<SPAN_NAME>&limit=10'

# 按时间范围搜索（微秒时间戳）
curl '{TRACES_ENDPOINT}/select/jaeger/api/traces?service=<SERVICE_NAME>&start=<START_US>&end=<END_US>&limit=10'

# 按最小耗时搜索（微秒）
curl '{TRACES_ENDPOINT}/select/jaeger/api/traces?service=<SERVICE_NAME>&minDuration=1000000&limit=10'

# 按 tag 搜索
curl '{TRACES_ENDPOINT}/select/jaeger/api/traces?service=<SERVICE_NAME>&tags={"error":"true"}&limit=10'
```

### 服务操作列表

```bash
# 列出某服务的所有 span 操作名
curl '{TRACES_ENDPOINT}/select/jaeger/api/services/<SERVICE_NAME>/operations'
```

---

## 3. 排障流程模板

### 场景 A：用户报告功能失败

```bash
# 1. 从前端/响应头获取 trace_id
TRACE_ID="<paste-here>"

# 2. 用 trace_id 查全链路日志
curl "http://localhost:9428/select/logsql/query?query=trace_id:${TRACE_ID}&limit=50"

# 3. 找到 result:failed 的条目，确认 action 和 error_kind

# 4. 查看 span 树，定位耗时瓶颈
curl "http://localhost:10428/select/jaeger/api/traces/${TRACE_ID}" | python3 -m json.tool

# 5. 根据 action 查同类历史错误
curl "http://localhost:9428/select/logsql/query?query=action:<FAILED_ACTION>%20AND%20result:failed&limit=20&start=1h"
```

### 场景 B：排查系统性错误

```bash
# 1. 查看错误分布
curl 'http://localhost:9428/select/logsql/query?query=result:failed&limit=50&start=30m'

# 2. 按 error_kind 聚合
curl 'http://localhost:9428/select/logsql/stats_query?query=result:failed&stats_field=error_kind&stats_func=count'

# 3. 深入最多的错误类型
curl 'http://localhost:9428/select/logsql/query?query=error_kind:db_error&limit=20&start=30m'

# 4. 看对应的 trace
# 从日志中取 trace_id，再查 span 树
```

### 场景 C：验证新功能观测接入

```bash
# 1. 触发新功能的 HTTP 请求
curl -i http://localhost:8080/api/v1/your-endpoint

# 2. 从响应头取 X-Trace-Id
# X-Trace-Id: abc123...

# 3. 验证日志存在
curl 'http://localhost:9428/select/logsql/query?query=trace_id:abc123&limit=50'

# 4. 验证 span 存在
curl 'http://localhost:10428/select/jaeger/api/traces/abc123'

# 5. 验证字段完整性
# - 成功路径：action + result:success + trace_id ✓
# - 失败路径：action + result:failed + error_kind + trace_id ✓
```

### 场景 D：性能排查

```bash
# 1. 找到慢请求的 trace_id（如 latency_ms > 1000）
curl 'http://localhost:9428/select/logsql/query?query=action:http.request%20AND%20latency_ms:>1000&limit=10'

# 2. 查看 span 树找瓶颈
curl "http://localhost:10428/select/jaeger/api/traces/<TRACE_ID>" | python3 -m json.tool

# 3. 搜索超过阈值的 trace
curl 'http://localhost:10428/select/jaeger/api/traces?service=<SERVICE>&minDuration=1000000&limit=10'
```

---

## 4. noop 模式验证

```bash
# 不设置 {PREFIX}_LOCAL_OBS 启动服务
go run .

# 验证：
# 1. 服务正常启动无报错
# 2. 不发任何请求到 VictoriaLogs/VictoriaTraces
# 3. 业务逻辑不受影响
```

---

## 5. LogsQL 速查

| 操作 | 语法 | 示例 |
|------|------|------|
| 精确匹配 | `field:value` | `action:auth.login` |
| AND | 空格 | `action:auth.login result:failed` |
| OR | `OR` | `error_kind:db_error OR error_kind:network_error` |
| NOT | `NOT` | `result:failed NOT error_kind:validation_error` |
| 模糊匹配 | `field:~"pattern"` | `_msg:~"timeout"` |
| 存在性 | `field:*` | `error_kind:*`（所有有 error_kind 的日志） |
| 数值比较 | `field:>N` / `field:<N` | `latency_ms:>1000` |
| 全文搜索 | `~"text"` | `~"connection refused"` |
| 通配符 | `field:prefix*` | `action:auth.*` |
