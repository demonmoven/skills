# 结构化日志字段标准

> 本文件是 `harness-obs-init` 技能的参考资料。
> 定义基于 `logs/v2` SDK 的结构化日志字段规范。

---

## Table of Contents

- [核心范式：action-result](#核心范式action-result)
- [字段分类](#字段分类)
- [action 命名规范](#action-命名规范)
- [result 枚举](#result-枚举)
- [error_kind 枚举](#error_kind-枚举)
- [敏感字段安全规范](#敏感字段安全规范)
- [使用模式](#使用模式)

---

## 核心范式：action-result

每条业务日志必须包含 **action**（做了什么）和 **result**（结果如何），
这是结构化日志可检索、可聚合、可告警的基础。

```go
observability.Info(ctx, "workspace.create", "success", "workspace_id", wid)
observability.Error(ctx, "auth.poll_token", "failed", "auth_error", err, "device_code", code)
```

---

## 字段分类

### 必选字段

| 字段 | 类型 | 说明 | 示例 |
|------|------|------|------|
| `action` | string | 动作名，格式 `domain.operation` | `auth.poll_token`、`workspace.create` |
| `result` | string | 结果枚举 | `success` / `failed` / `pending` |

### 条件必选字段

| 字段 | 类型 | 条件 | 说明 |
|------|------|------|------|
| `error_kind` | string | result=failed 时必填 | 错误分类枚举 |
| `error_message` | string | result=failed 时可填 | `err.Error()`，长度上限 256 |

### 自动注入字段

| 字段 | 注入方式 | 说明 |
|------|----------|------|
| `trace_id` | TraceMiddleware 或 buildKVs | OTel trace ID（32 位十六进制） |
| `otel_span_id` | TraceMiddleware 或 buildKVs | OTel span ID（16 位十六进制） |
| `_msg` | buildKVs fallback | VictoriaLogs 的 message 字段；缺失时默认取 action |
| `time` | VLogsWriter | RFC3339Nano 格式时间戳 |
| `level` | logs/v2 SDK | `debug` / `info` / `warn` / `error` |
| `service` | VLogsWriter (PSM 或 serviceName) | 服务标识 |
| `location` | logs/v2 SDK | 源码位置（文件名:行号） |

### 业务上下文字段

按场景补充，字段名使用 **英文 snake_case**：

| 字段 | 适用场景 |
|------|----------|
| `user_id` | 有用户身份的请求 |
| `workspace_id` | 工作区相关操作 |
| `session_id` / `session_key` | 会话相关操作 |
| `request_id` | 请求级别唯一标识（如果不用 trace_id） |
| `method`、`path`、`status_code`、`latency_ms` | HTTP 请求日志 |
| `component` | 多组件共享进程时区分来源 |

---

## action 命名规范

### 格式

```
domain.operation
domain.sub_domain.operation
```

### 规则

1. **英文小写** + **点号分层** + **下划线连词**
2. **稳定不变** — action 名一旦上线就不随意改，需改时提供双写迁移期
3. **可检索** — 禁止自然语言（`"user logged in successfully"` ❌）
4. **可聚合** — 同类操作使用相同 action（不要把参数编进 action 名）

### 示例

```
http.request              # HTTP 访问日志（中间件自动）

auth.begin_login          # 开始登录流程
auth.poll_token           # 轮询设备码换 token
auth.logout               # 登出
auth.get_status           # 查询认证状态
auth.middleware.check      # 中间件会话检查
auth.middleware.refresh    # 中间件会话刷新

workspace.list            # 列出工作区
workspace.create          # 创建工作区
workspace.delete          # 删除工作区

sso.request_device_code   # 请求设备码
sso.exchange_device_code  # 设备码换 token
sso.get_userinfo          # 获取用户信息
sso.refresh_token         # 刷新 token

skill.upload              # 上传 Skill
skill.run_eval            # 执行 Skill 评测
resource.scan             # 扫描资源

chat.send                 # 发送聊天消息
chat.stream.consume       # 消费 SSE 流
chat.tool.exec            # 执行工具调用
```

### 新增模块 Checklist

新增模块时，先定义 action 列表并记录在模块文档中：

```
module: payment
actions:
  - payment.create_order     # 创建订单
  - payment.process          # 处理支付
  - payment.refund           # 退款
  - payment.webhook.receive  # 接收支付回调
```

---

## result 枚举

| 值 | 语义 | 说明 |
|----|------|------|
| `success` | 操作成功 | 默认成功态 |
| `failed` | 操作失败 | 必须附带 `error_kind` |
| `pending` | 操作进行中 | 用于异步/长时间操作的起始日志 |

如业务需要更多细分语义，可扩展（如 `authenticated`、`anonymous`、`skipped`），
但**不得与上述三个冲突**。

---

## error_kind 枚举

推荐枚举集（可按仓库扩展，但优先复用已有枚举）：

| error_kind | 语义 |
|------------|------|
| `validation_error` | 入参校验失败 |
| `auth_error` | 认证失败（token 无效/过期） |
| `permission_denied` | 鉴权失败（无权限） |
| `not_found` | 资源不存在 |
| `db_error` | 数据库操作失败 |
| `external_dependency_error` | 外部服务调用失败 |
| `network_error` | 网络层异常 |
| `decode_error` | 数据解析/反序列化失败 |
| `internal_error` | 内部逻辑错误（兜底） |
| `timeout_error` | 超时 |
| `rate_limit_error` | 限流触发 |
| `conflict_error` | 资源冲突（并发写入等） |

### 演进规则

- 新增枚举前先检查是否能复用已有枚举
- 同义词必须收敛（`db_error` 和 `database_error` 不能同时存在）
- 禁止用 HTTP status code 当 error_kind（`404` ❌ → `not_found` ✅）

---

## 敏感字段安全规范

### 自动脱敏

`logging.go` 的 `buildKVs` 对以下关键词做大小写不敏感匹配脱敏：

```
token / password / secret / authorization / cookie / credential
```

匹配到的 KV 值替换为 `[REDACTED]`。

### 严格禁止写入日志

即使代码层做了脱敏，以下字段**从源头就不应该传入**日志函数：

- access token / refresh token / auth token 全文
- password / secret / api key 全文
- cookie 完整值
- 用户手机号 / 身份证号（如有 PII 要求）

### 值长度截断

string 类型的值超过 **256 字符**自动截断并追加 `...`。
这是防止大 JSON/堆栈意外灌入单条日志。

---

## 使用模式

### 模式 A：logs/v2 Middleware 注入（推荐 HTTP 服务）

适合有中心化日志框架的服务端。TraceMiddleware 在每条日志写入前自动注入 trace_id/span_id。

```go
// init.go
logs.SetDefaultLogger(
    logs.AppendWriter(logs.DebugLevel, vlogsWriter),
    logs.SetMiddleware(NewTraceMiddleware()),
)

// 业务代码 — 不需要手动传 trace_id
observability.Info(ctx, "workspace.create", "success", "workspace_id", wid)
// VLogsWriter 收到的日志自动带 trace_id + otel_span_id
```

### 模式 B：buildKVs 手动注入（适合无中间件场景）

适合桌面端/CLI/worker 等没有中心化中间件的进程。在 `buildKVs` 中手动从 ctx 提取 trace/span。

```go
func buildKVs(ctx context.Context, ...) []any {
    // ... 业务字段 ...

    // 手动注入 trace 字段
    if traceID := TraceID(ctx); traceID != "" {
        kvs = append(kvs, "trace_id", traceID)
    }
    if spanID := SpanID(ctx); spanID != "" {
        kvs = append(kvs, "otel_span_id", spanID)
    }

    return kvs
}
```

### 模式 C：WithFields ctx 透传（多组件共享进程）

多组件在同一进程运行，最外层 `Init()` 一次，内层组件用 `WithFields` 打 tag。

```go
// 桌面端入口
obsShutdown := observability.Init(ctx, "my-app")
defer obsShutdown()

// 内嵌 gateway 组件
ctx = observability.WithFields(ctx, "component", "gateway")
observability.Info(ctx, "gateway.start", "success")
// 日志自动带 component=gateway + trace_id + service=my-app
```
