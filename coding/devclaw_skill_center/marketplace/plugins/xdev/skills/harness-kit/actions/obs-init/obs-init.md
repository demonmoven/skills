> **Action: `obs-init`** — 由 `/xdev:harness-kit obs-init` 路由调用。
> 原 skill: `harness-obs-init` (author: guoshuai.030, version: 1.4)

# Harness Obs Init — 本地观测栈建立

## 核心理念

> **Agent 能不能不靠猜测就观察运行态？**
> — Harness Engineering 观测维度的核心追问

本技能为 Go 仓库建立"零默认开销、环境变量驱动"的本地观测栈，覆盖：
- 结构化日志（基于字节 `logs/v2` SDK 扩展）
- 分布式链路追踪（OpenTelemetry → VictoriaTraces）
- 极简基础设施（2 个 Docker 容器，直接 push，无 Collector）

**三条不变量**：
1. 未开启时必须是 noop，不影响生产行为
2. 日志必须结构化（action/result/error_kind），禁止自由文本
3. 链路可关联——任何日志都能通过 trace_id 与请求的所有阶段关联

## 激活后参考

- [references/GO-OBS-TEMPLATE.md](references/GO-OBS-TEMPLATE.md) — Go 观测包完整代码模板
- [references/INFRA-RECIPES.md](references/INFRA-RECIPES.md) — docker-compose 编排模板与替代方案
- [references/LOG-FIELD-STANDARD.md](references/LOG-FIELD-STANDARD.md) — 结构化日志字段标准
- [references/QUERY-COOKBOOK.md](references/QUERY-COOKBOOK.md) — 排障查询模板集

## 适用前提

- 仓库使用 Go
- 已引入 `code.byted.org/gopkg/logs/v2`（字节内部日志 SDK）
- 本地有 Docker 环境（用于观测后端）
- 如果仓库已有观测包，本技能在已有基础上增量追加，不覆盖

## 完整流程

### Step 0: 读参考文件

在开始前，**必须先阅读**以下参考文件：

- [references/GO-OBS-TEMPLATE.md](references/GO-OBS-TEMPLATE.md) — 理解代码模板的模块划分
- [references/LOG-FIELD-STANDARD.md](references/LOG-FIELD-STANDARD.md) — 理解日志字段规范

### Step 1: 检测现状

**1.1 检测技术栈**

扫描仓库，确认以下关键信息：

| 检测项 | 怎么查 | 影响 |
|--------|--------|------|
| Go module 路径 | `go.mod` 第一行 | 决定包导入路径 |
| logs/v2 版本 | `go.mod` 中 `code.byted.org/gopkg/logs/v2` | 确认可用接口集 |
| OTel 依赖 | `go.mod` 中 `go.opentelemetry.io/otel` | 已有则复用版本 |
| HTTP 框架 | import 中是否有 `hertz`/`gin`/`echo`/`fiber` | 决定中间件模板 |
| 已有 observability 包 | `pkg/observability/` 或类似目录 | 已有则增量，不覆盖 |
| docker-compose 文件 | 根目录 `docker-compose*.yml` | 已有则追加服务 |

**1.2 检测已有观测能力**

| 已有能力 | 信号 | 策略 |
|----------|------|------|
| 已有 VLogsWriter | 搜索 `LogWriter` 实现 | 不重复生成 writer |
| 已有 TraceMiddleware | 搜索 `logs.Middleware` 实现 | 不重复生成 middleware |
| 已有 OTel tracer | 搜索 `TracerProvider` | 复用已有初始化 |
| 已有 HTTP 中间件 | 搜索 server span 创建 | 不重复生成 |
| 已有 docker-compose.obs.yml | 文件存在 | 检查服务是否完整 |
| 散落的 logs.Infof/CtxInfo | 搜索 `logs.Ctx` + `logs.Infof` | 记录为待迁移清单 |

**1.3 向用户报告检测结果**

```
检测完成：
- Go module: xxx
- logs/v2: v2.2.2 ✓
- OTel: 未引入（需要新增）
- HTTP 框架: Hertz
- 已有观测包: 无
- 待迁移散落日志: 12 处
```

### Step 2: 基础设施选型

向用户确认观测后端选型。**默认推荐 VictoriaLogs + VictoriaTraces**。

推荐理由：
- 极简：2 个容器，无 Collector/Grafana
- 免费：无商业许可限制
- 直接 push：不需要 OTel Collector 做中转
- 内置查询 API：LogsQL + Jaeger-compatible API，curl 直接用
- 低资源占用：适合本地开发

替代方案说明见 [references/INFRA-RECIPES.md](references/INFRA-RECIPES.md)。

用户确认后进入 Step 3。

### Step 3: 生成基础设施编排

根据选型生成 `docker-compose.obs.yml`。

**默认模板**（VictoriaLogs + VictoriaTraces）：

```yaml
services:
  victoria-logs:
    image: victoriametrics/victoria-logs:latest
    container_name: ${PROJECT}-victoria-logs
    ports: ["9428:9428"]
    command: ["-httpListenAddr=:9428"]
    restart: unless-stopped
  victoria-traces:
    image: victoriametrics/victoria-traces:latest
    container_name: ${PROJECT}-victoria-traces
    ports: ["10428:10428"]
    command: ["-httpListenAddr=:10428"]
    restart: unless-stopped
```

生成后告知用户启动命令：
```bash
docker compose -f docker-compose.obs.yml up -d
```

### Step 4: 生成观测包代码

参照 [references/GO-OBS-TEMPLATE.md](references/GO-OBS-TEMPLATE.md)，在仓库中生成观测包。

**目标目录**：`pkg/observability/`（或用户指定路径）

**必须生成的文件**（按依赖顺序）：

| 文件 | 职责 | 核心接口 |
|------|------|----------|
| `config.go` | 环境变量读取 + 默认配置 | `Enabled() bool`、`DefaultConfig()` |
| `vlogs_writer.go` | 实现 `writer.LogWriter`，批量推送 NDJSON 到 VictoriaLogs | `Write()`、`Flush()`、`Close()` |
| `trace_middleware.go` | 实现 `logs.Middleware`，从 OTel span 注入 trace_id/span_id | `NewTraceMiddleware()` |
| `tracer.go` | OTel TracerProvider 初始化，导出到 VictoriaTraces | `initTracer()` |
| `logging.go` | 业务日志 helper（action-result 范式 + 脱敏） | `Info()`、`Warn()`、`Error()` |
| `span.go` | Span 创建 helper（带 noop 守卫） | `StartSpan()`、`StartSpanWithStatus()` |
| `init.go` | 入口点，编排 writer/middleware/tracer | `Init(ctx, serviceName) func()` |

**可选文件**：

| 文件 | 职责 | 何时需要 |
|------|------|----------|
| `middleware.go` | HTTP 框架中间件（server span + access log + X-Trace-Id） | 有 HTTP 服务时 |
| `context.go` | ctx-based 结构化字段透传 `WithFields()` | 多组件共享进程时 |

**关键实现约束**：

1. **VLogsWriter 必须实现 `writer.LogWriter` 接口**（`logs/v2` 的 `writer` 子包）：
   ```go
   type LogWriter interface {
       io.Closer
       Write(log RecyclableLog) error
       Flush() error
   }
   ```

2. **TraceMiddleware 必须返回 `logs.Middleware` 类型**：
   ```go
   type Middleware func(log RewritableLog) RewritableLog
   ```
   从 `log.GetContext()` 获取 OTel span，用 `codec.NewKeyValue` 创建 KV 对，通过 `log.SetKVList()` 注入。

3. **Init 通过 `logs.SetDefaultLogger` 的 Option 模式注册**：
   ```go
   logs.SetDefaultLogger(
       logs.AppendWriter(logs.DebugLevel, vlogsWriter),  // 追加，不替换
       logs.SetMiddleware(NewTraceMiddleware()),
   )
   ```
   `AppendWriter` 是追加到已有 writer 列表，保留 console/file 输出不受影响。

4. **所有 helper 函数内置 noop 守卫**：
   ```go
   func Info(ctx context.Context, action, result string, kvs ...any) {
       if !Enabled() { return }
       logs.CtxInfoKVs(ctx, buildKVs(action, result, "", nil, kvs)...)
   }
   ```

5. **PSM（服务名）处理**：
   - 字节内部服务有 `env.PSM` 环境变量，`logs/v2` 默认读取
   - 本地开发/桌面端场景 PSM 常为空，VLogsWriter 需要 `serviceName` 回填逻辑
   - 在 `Write()` 中：`service := log.GetPSM(); if service == "" { service = w.serviceName }`

### Step 5: 接入 HTTP 中间件

如果仓库有 HTTP 服务，生成框架对应的观测中间件。

**中间件职责**（框架无关）：
1. 从请求 context 创建 server span（`otel.Tracer(serviceName).Start(ctx, "http.request")`）
2. 记录请求级结构化日志（action=`http.request`，含 method/path/status/latency_ms）
3. 回传 `X-Trace-Id` 响应头
4. 5xx 自动标记 OTel Error 状态

**Hertz 中间件**（字节内部最常用）：

```go
func HertzMiddleware(serviceName string) app.HandlerFunc {
    return func(c context.Context, ctx *app.RequestContext) {
        // 1. 创建 server span
        // 2. ctx.Next(c)
        // 3. 记录 access log
        // 4. 设置 X-Trace-Id
    }
}
```

**其他框架对照**（生成时按实际框架替换）：

| 框架 | 中间件签名 | 注册方式 |
|------|-----------|----------|
| Hertz | `app.HandlerFunc` | `h.Use(middleware)` |
| Gin | `gin.HandlerFunc` | `r.Use(middleware)` |
| Echo | `echo.MiddlewareFunc` | `e.Use(middleware)` |
| 标准库 | `func(http.Handler) http.Handler` | `http.Handle("/", middleware(handler))` |

### Step 6: 配置 main.go 接入

在仓库入口文件中注入观测栈初始化：

```go
func main() {
    ctx := context.Background()

    // 观测栈初始化（未开启时 noop）
    obsShutdown := observability.Init(ctx, "your-service-name")
    defer obsShutdown()

    // HTTP 中间件注册（如有）
    h := server.Default()
    h.Use(observability.HertzMiddleware("your-service-name"))

    // ... 业务代码
}
```

**环境变量说明**（写入 README 或 AGENTS.md）：

| 变量 | 作用 | 默认值 |
|------|------|--------|
| `{PREFIX}_LOCAL_OBS` | 总开关，`1` 启用 | 未设置（noop） |
| `{PREFIX}_VLOGS_ENDPOINT` | VictoriaLogs 地址 | `http://localhost:9428` |
| `{PREFIX}_VTRACES_ENDPOINT` | VictoriaTraces OTLP 地址 | `localhost:10428` |

`{PREFIX}` 由用户指定（如 `MYAPP`），避免与其他项目冲突。

### Step 7: 迁移散落日志（可选）

如果 Step 1 检测到散落的 `logs.Infof` / `logs.CtxInfo` 调用，建议分批迁移到结构化范式：

**迁移规则**：

| 原始调用 | 迁移为 | 说明 |
|----------|--------|------|
| `logs.Infof("xxx success")` | `observability.Info(ctx, "domain.op", "success")` | 添加 action + result |
| `logs.CtxErrorf(ctx, "failed: %v", err)` | `observability.Error(ctx, "domain.op", "failed", "error_kind", err)` | 添加 error_kind |
| `logs.CtxInfoKVs(ctx, "k1", v1)` | `observability.Info(ctx, "domain.op", "success", "k1", v1)` | 添加 action 骨架 |

**迁移优先级**：
1. HTTP handler 入口日志（已被中间件覆盖的可跳过）
2. 错误路径日志（必须有 error_kind）
3. 关键业务节点日志
4. 基础设施日志（DB/RDS/缓存初始化）— 最低优先级，可保留原样

### Step 8: 验证

**8.1 启动验证**

1\. 启动观测后端：

```bash
docker compose -f docker-compose.obs.yml up -d
```

2\. 启动服务（开启观测）：

```bash
{PREFIX}_LOCAL_OBS=1 go run .
```

3\. 验证服务注册（应包含 your-service-name）：

```bash
curl http://localhost:10428/select/jaeger/api/services
```

**8.2 最小验收（4 个场景必须通过）**

| 场景 | 验证方法 |
|------|----------|
| 成功请求 | 查询 `action:xxx AND result:success`，结果包含 trace_id |
| 失败请求 | 查询 `action:xxx AND result:failed`，结果包含 error_kind |
| trace 关联 | 从响应头取 X-Trace-Id，按 `trace_id:<ID>` 查到完整调用链 |
| noop 模式 | 不设 `{PREFIX}_LOCAL_OBS`，服务正常启动无报错 |

**8.3 查询命令**

参见 [references/QUERY-COOKBOOK.md](references/QUERY-COOKBOOK.md)。

### Step 9: 产出文档

在仓库中更新以下文档：

1. **README.md / AGENTS.md** — 新增"本地观测"段落，含启动命令和环境变量说明
2. **新功能接入 Checklist** — 新增 handler 时必须补齐的观测项：
   - 已定义 action 名称并符合 `domain.operation` 命名规范
   - 成功路径有 `result=success` 日志
   - 失败路径有 `result=failed` + `error_kind` 日志
   - 关键业务阶段有子 span
   - 不记录敏感字段
   - 本地可通过查询模板复现并定位

## logs/v2 SDK 快速参考

### 核心接口

```go
// Writer 接口 — code.byted.org/gopkg/logs/v2/writer
type LogWriter interface {
    io.Closer
    Write(log RecyclableLog) error
    Flush() error
}

// RecyclableLog 可读字段
log.GetBody()       // []byte — 日志正文
log.GetLevel()      // string — "debug"/"info"/"warn"/"error"
log.GetTime()       // time.Time
log.GetPSM()        // string — 服务名（可能为空）
log.GetKVListStr()  // []string — [k1,v1,k2,v2,...] 扁平 KV
log.GetKVList()     // []*codec.KeyValue — 结构化 KV
log.GetContext()    // context.Context

// Middleware 类型
type Middleware func(log RewritableLog) RewritableLog
// RewritableLog 额外可写
log.SetBody([]byte)
log.SetKVList([]*codec.KeyValue)
```

### 注册方式

```go
logs.SetDefaultLogger(
    logs.AppendWriter(logs.DebugLevel, myWriter),   // 追加 writer（不替换）
    logs.SetMiddleware(myMiddleware),                // 设置 middleware
)
```

### 业务侧调用

```go
// 结构化 KV 风格（推荐）
logs.CtxInfoKVs(ctx, "action", "domain.op", "result", "success", "user_id", uid)
logs.CtxWarnKVs(ctx, "action", "domain.op", "result", "failed", "error_kind", "auth_error")
logs.CtxErrorKVs(ctx, "action", "domain.op", "result", "failed", "error_kind", "db_error")

// 格式字符串风格（不推荐新代码使用，但兼容存量）
logs.CtxInfof(ctx, "operation completed: %s", detail)
```

### KV 注入（codec 包）

```go
import "code.byted.org/log_market/ttlogagent_gosdk/v4/codec"

kv, _ := codec.NewKeyValue("trace_id", traceID)
log.SetKVList(append(log.GetKVList(), kv))
```

## 跨进程 Trace 传播

| 传输层 | 协议 | 实现 |
|--------|------|------|
| HTTP | W3C `Traceparent` header | `otel.GetTextMapPropagator().Inject/Extract()` |
| WebSocket | frame metadata 中 `traceparent` 字段 | 手动注入/提取后调 `Extract()` |
| gRPC | metadata header（OTel gRPC instrumentation 自动处理） | `otelgrpc` interceptor |

## 与其他 Harness 技能的衔接

| 技能 | 衔接点 |
|------|--------|
| `harness-analysis` | 本技能产出的观测栈直接提升"观测体系"维度评分 |
| `harness-hook-init` | 可在 pre-commit hook 中加入"新 handler 必须有 action 日志"检查 |
| `harness-doc-gardener` | 巡检时检查观测文档（环境变量、查询模板）是否与代码同步 |
| `harness-debt-scan` | 扫描散落 `logs.Infof` 调用，标记为待迁移技术债 |
