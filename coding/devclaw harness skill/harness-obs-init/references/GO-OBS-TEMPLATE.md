# Go 观测包代码模板

> 本文件是 `harness-obs-init` 技能的参考资料。
> 生成代码时以本模板为蓝本，根据目标仓库实际情况调整。

## 目录结构

```
pkg/observability/
├── config.go            # 环境变量 + 默认配置
├── init.go              # 入口点：编排 writer/middleware/tracer
├── vlogs_writer.go      # logs/v2 Writer → VictoriaLogs (NDJSON push)
├── trace_middleware.go   # logs/v2 Middleware → 注入 trace_id/span_id
├── tracer.go            # OTel TracerProvider → VictoriaTraces (OTLP/HTTP)
├── logging.go           # 业务日志 helper（action-result + 脱敏）
├── span.go              # Span 创建 helper（noop 守卫）
├── middleware.go         # [可选] HTTP 框架中间件
└── context.go           # [可选] ctx-based 结构化字段透传
```

---

## 1. config.go

```go
package observability

import "os"

// 环境变量 key — 生成时替换 {PREFIX} 为项目前缀（如 MYAPP）
const (
	EnvLocalObs       = "{PREFIX}_LOCAL_OBS"
	EnvVLogsEndpoint  = "{PREFIX}_VLOGS_ENDPOINT"
	EnvVTracesEndpoint = "{PREFIX}_VTRACES_ENDPOINT"
)

// Config 观测栈总配置
type Config struct {
	VictoriaLogs   VictoriaLogsConfig
	VictoriaTraces VictoriaTracesConfig
}

type VictoriaLogsConfig struct {
	Endpoint      string // HTTP push 地址
	BatchSize     int    // 缓冲区满阈值
	FlushInterval int    // 定时 flush 间隔（秒）
}

type VictoriaTracesConfig struct {
	Endpoint string // OTLP/HTTP 地址（不含 scheme）
}

// Enabled 检查观测栈总开关
func Enabled() bool {
	return os.Getenv(EnvLocalObs) == "1"
}

// DefaultConfig 返回默认配置，环境变量可覆盖
func DefaultConfig() Config {
	cfg := Config{
		VictoriaLogs: VictoriaLogsConfig{
			Endpoint:      "http://localhost:9428",
			BatchSize:     50,
			FlushInterval: 2,
		},
		VictoriaTraces: VictoriaTracesConfig{
			Endpoint: "localhost:10428",
		},
	}
	if ep := os.Getenv(EnvVLogsEndpoint); ep != "" {
		cfg.VictoriaLogs.Endpoint = ep
	}
	if ep := os.Getenv(EnvVTracesEndpoint); ep != "" {
		cfg.VictoriaTraces.Endpoint = ep
	}
	return cfg
}
```

---

## 2. vlogs_writer.go

实现 `code.byted.org/gopkg/logs/v2/writer` 的 `LogWriter` 接口。

```go
package observability

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"code.byted.org/gopkg/logs/v2/writer"
)

// VLogsWriter 批量推送结构化日志到 VictoriaLogs
type VLogsWriter struct {
	cfg         VictoriaLogsConfig
	serviceName string          // PSM 为空时的回填值
	client      *http.Client
	buf         []map[string]any
	mu          sync.Mutex
	done        chan struct{}
}

func NewVLogsWriter(cfg VictoriaLogsConfig, serviceName string) *VLogsWriter {
	w := &VLogsWriter{
		cfg:         cfg,
		serviceName: serviceName,
		client:      &http.Client{Timeout: 5 * time.Second},
		buf:         make([]map[string]any, 0, cfg.BatchSize),
		done:        make(chan struct{}),
	}
	go w.ticker()
	return w
}

// Write 实现 writer.LogWriter — 将 RecyclableLog 转为 JSON map 放入缓冲
func (w *VLogsWriter) Write(log writer.RecyclableLog) error {
	entry := map[string]any{
		"_msg":    string(log.GetBody()),
		"level":   log.GetLevel(),
		"time":    log.GetTime().Format(time.RFC3339Nano),
		"location": string(log.GetLocation()),
	}
	// PSM 回填
	service := log.GetPSM()
	if service == "" {
		service = w.serviceName
	}
	entry["service"] = service

	// 展平 KV list
	kvs := log.GetKVListStr()
	for i := 0; i+1 < len(kvs); i += 2 {
		entry[kvs[i]] = kvs[i+1]
	}

	w.mu.Lock()
	w.buf = append(w.buf, entry)
	shouldFlush := len(w.buf) >= w.cfg.BatchSize
	w.mu.Unlock()

	if shouldFlush {
		w.flush()
	}
	return nil
}

func (w *VLogsWriter) Flush() error { w.flush(); return nil }

func (w *VLogsWriter) Close() error {
	close(w.done)
	w.flush()
	return nil
}

func (w *VLogsWriter) ticker() {
	t := time.NewTicker(time.Duration(w.cfg.FlushInterval) * time.Second)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			w.flush()
		case <-w.done:
			return
		}
	}
}

func (w *VLogsWriter) flush() {
	w.mu.Lock()
	if len(w.buf) == 0 {
		w.mu.Unlock()
		return
	}
	batch := w.buf
	w.buf = make([]map[string]any, 0, w.cfg.BatchSize)
	w.mu.Unlock()

	var body bytes.Buffer
	for _, entry := range batch {
		_ = json.NewEncoder(&body).Encode(entry) // NDJSON: 每行一条
	}

	url := w.cfg.Endpoint + "/insert/jsonline?_stream_fields=service,level&_msg_field=_msg&_time_field=time"
	req, err := http.NewRequest("POST", url, &body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[obs] vlogs request error: %v\n", err)
		return
	}
	req.Header.Set("Content-Type", "application/x-ndjson")
	resp, err := w.client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[obs] vlogs push error: %v\n", err)
		return
	}
	resp.Body.Close()
}
```

### 关键设计决策

- **追加不替换**：通过 `logs.AppendWriter` 注册，保留 console/file 输出
- **批量推送**：buffer 达 `BatchSize`(50) 或 ticker 到 `FlushInterval`(2s) 触发
- **静默失败**：push 失败只打 stderr，不影响业务——本地观测的容忍度
- **PSM 回填**：`log.GetPSM()` 为空时用构造时传入的 `serviceName`

---

## 3. trace_middleware.go

实现 `logs.Middleware`，在每条日志写入前注入 OTel 的 trace_id / span_id。

```go
package observability

import (
	"code.byted.org/gopkg/logs/v2"
	"code.byted.org/log_market/ttlogagent_gosdk/v4/codec"
	"go.opentelemetry.io/otel/trace"
)

// NewTraceMiddleware 返回 logs/v2 Middleware，自动注入 trace_id 和 otel_span_id
func NewTraceMiddleware() logs.Middleware {
	return func(log logs.RewritableLog) logs.RewritableLog {
		ctx := log.GetContext()
		if ctx == nil {
			return log
		}
		span := trace.SpanFromContext(ctx)
		sc := span.SpanContext()
		if !sc.HasTraceID() {
			return log
		}

		kvList := log.GetKVList()
		traceKV, _ := codec.NewKeyValue("trace_id", sc.TraceID().String())
		spanKV, _ := codec.NewKeyValue("otel_span_id", sc.SpanID().String())
		log.SetKVList(append(kvList, traceKV, spanKV))
		return log
	}
}
```

### 工作原理

1. `logs.SetDefaultLogger(logs.SetMiddleware(NewTraceMiddleware()))` 注册后，
   每条日志在写入任何 Writer 之前都会经过此 middleware
2. 从 `log.GetContext()` 拿到 ctx → 从 ctx 拿到 OTel span → 提取 traceID/spanID
3. 用 `codec.NewKeyValue` 创建 KV 对 → 追加到 `log.GetKVList()`
4. VLogsWriter 的 `Write` 遍历 `GetKVListStr()` 时会自动展平出 `trace_id` / `otel_span_id`

### codec 包说明

`code.byted.org/log_market/ttlogagent_gosdk/v4/codec` 是 logs/v2 内部使用的 KV 编码包。
`codec.NewKeyValue(key, value)` 返回 `(*codec.KeyValue, error)`，这是往日志 KV list 注入自定义字段的唯一正确方式。

---

## 4. tracer.go

```go
package observability

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func initTracer(ctx context.Context, serviceName string, cfg VictoriaTracesConfig) func(context.Context) error {
	exp, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(cfg.Endpoint),
		otlptracehttp.WithInsecure(), // 本地开发不需要 TLS
	)
	if err != nil {
		return func(context.Context) error { return nil }
	}

	res, _ := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
		),
	)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{}) // W3C Traceparent

	return tp.Shutdown
}
```

### 要点

- 导出到 VictoriaTraces 的 OTLP/HTTP 端口（默认 `:10428`）
- `WithInsecure()` — 本地开发无 TLS
- `propagation.TraceContext{}` — 启用 W3C Traceparent 跨进程传播
- 返回 `Shutdown` 函数供 `Init` 的 cleanup 闭包调用

---

## 5. logging.go

```go
package observability

import (
	"context"
	"fmt"
	"strings"

	"code.byted.org/gopkg/logs/v2"
)

// sensitiveKeys 触发脱敏的关键词（大小写不敏感匹配）
var sensitiveKeys = []string{
	"token", "password", "secret", "authorization", "cookie", "credential",
}

func Info(ctx context.Context, action, result string, kvs ...any) {
	if !Enabled() { return }
	logs.CtxInfoKVs(ctx, buildKVs(action, result, "", nil, kvs)...)
}

func Warn(ctx context.Context, action, result, errorKind string, kvs ...any) {
	if !Enabled() { return }
	logs.CtxWarnKVs(ctx, buildKVs(action, result, errorKind, nil, kvs)...)
}

func Error(ctx context.Context, action, result, errorKind string, err error, kvs ...any) {
	if !Enabled() { return }
	logs.CtxErrorKVs(ctx, buildKVs(action, result, errorKind, err, kvs)...)
}

func buildKVs(action, result, errorKind string, err error, extra []any) []any {
	kvs := make([]any, 0, 10+len(extra))
	kvs = append(kvs, "action", action, "result", result)

	if errorKind != "" {
		kvs = append(kvs, "error_kind", errorKind)
	}
	if err != nil {
		msg := err.Error()
		if len(msg) > 256 {
			msg = msg[:256] + "..."
		}
		kvs = append(kvs, "error_message", msg)
	}

	// 合并业务 KV，做脱敏和截断
	for i := 0; i+1 < len(extra); i += 2 {
		key := fmt.Sprintf("%v", extra[i])
		val := extra[i+1]

		// _msg 映射（兼容 VictoriaLogs message 字段）
		if key == "msg" {
			key = "_msg"
		}

		// 敏感字段脱敏
		if isSensitiveKey(key) {
			val = "[REDACTED]"
		} else if s, ok := val.(string); ok && len(s) > 256 {
			val = s[:256] + "..."
		}

		kvs = append(kvs, key, val)
	}

	// 默认 _msg fallback
	hasMsg := false
	for i := 0; i < len(kvs); i += 2 {
		if kvs[i] == "_msg" { hasMsg = true; break }
	}
	if !hasMsg {
		kvs = append(kvs, "_msg", action)
	}

	return kvs
}

func isSensitiveKey(key string) bool {
	lower := strings.ToLower(key)
	for _, s := range sensitiveKeys {
		if strings.Contains(lower, s) {
			return true
		}
	}
	return false
}
```

---

## 6. span.go

```go
package observability

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "observability" // 替换为你的模块名

// StartSpan 创建一个 span，返回 ctx 和 end 函数
func StartSpan(ctx context.Context, name string, kvs ...any) (context.Context, func()) {
	if !Enabled() {
		return ctx, func() {}
	}
	ctx, span := otel.Tracer(tracerName).Start(ctx, name,
		trace.WithAttributes(attributesFromKVs(kvs)...),
	)
	return ctx, func() { span.End() }
}

// StartSpanWithStatus 创建 span，end 时根据 error 自动标记状态
func StartSpanWithStatus(ctx context.Context, name string, kvs ...any) (context.Context, func(error)) {
	if !Enabled() {
		return ctx, func(error) {}
	}
	ctx, span := otel.Tracer(tracerName).Start(ctx, name,
		trace.WithAttributes(attributesFromKVs(kvs)...),
	)
	return ctx, func(err error) {
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)
		}
		span.End()
	}
}
```

### attributesFromKVs

将 `...any` 格式的 KV 对转为 OTel `attribute.KeyValue` 切片：

```go
import "go.opentelemetry.io/otel/attribute"

func attributesFromKVs(kvs []any) []attribute.KeyValue {
	attrs := make([]attribute.KeyValue, 0, len(kvs)/2)
	for i := 0; i+1 < len(kvs); i += 2 {
		key := fmt.Sprintf("%v", kvs[i])
		switch v := kvs[i+1].(type) {
		case string:
			attrs = append(attrs, attribute.String(key, v))
		case int:
			attrs = append(attrs, attribute.Int(key, v))
		case int64:
			attrs = append(attrs, attribute.Int64(key, v))
		case float64:
			attrs = append(attrs, attribute.Float64(key, v))
		case bool:
			attrs = append(attrs, attribute.Bool(key, v))
		default:
			attrs = append(attrs, attribute.String(key, fmt.Sprintf("%v", v)))
		}
	}
	return attrs
}
```

---

## 7. init.go

```go
package observability

import "context"

// Init 初始化观测栈，返回 cleanup 函数。
// 当 {PREFIX}_LOCAL_OBS != "1" 时为 noop。
func Init(ctx context.Context, serviceName string) func() {
	if !Enabled() {
		return func() {}
	}

	cfg := DefaultConfig()

	// 1. 创建 VLogsWriter
	vlogsWriter := NewVLogsWriter(cfg.VictoriaLogs, serviceName)

	// 2. 注册到 logs/v2（追加 writer + 设置 middleware）
	logs.SetDefaultLogger(
		logs.AppendWriter(logs.DebugLevel, vlogsWriter),
		logs.SetMiddleware(NewTraceMiddleware()),
	)

	// 3. 初始化 OTel tracer
	tracerShutdown := initTracer(ctx, serviceName, cfg.VictoriaTraces)

	return func() {
		_ = vlogsWriter.Close()
		_ = tracerShutdown(ctx)
	}
}
```

### 初始化顺序

```
Init(ctx, serviceName)
  ├─ Enabled() → false → return noop
  ├─ DefaultConfig() + env 覆盖
  ├─ NewVLogsWriter()          ← 创建 writer（启动 ticker goroutine）
  ├─ logs.SetDefaultLogger()   ← 注册 writer + middleware 到 logs/v2
  ├─ initTracer()              ← OTel TracerProvider + Propagator
  └─ return cleanup func       ← Close writer + Shutdown tracer
```

---

## 8. middleware.go（可选 — Hertz 版）

```go
package observability

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// HertzMiddleware 创建 Hertz HTTP 观测中间件
func HertzMiddleware(serviceName string) app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		if !Enabled() {
			ctx.Next(c)
			return
		}

		method := string(ctx.Method())
		path := string(ctx.Path())
		spanName := fmt.Sprintf("%s %s", method, path)

		c, span := otel.Tracer(serviceName).Start(c, spanName,
			trace.WithSpanKind(trace.SpanKindServer),
		)
		start := time.Now()

		ctx.Next(c)

		status := ctx.Response.StatusCode()
		latency := time.Since(start).Milliseconds()

		// 记录 access log
		Info(c, "http.request", resultFromStatus(status),
			"method", method,
			"path", path,
			"status_code", status,
			"latency_ms", latency,
		)

		// 5xx 标记 Error
		if status >= 500 {
			span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", status))
		}

		// 回传 X-Trace-Id
		if sc := span.SpanContext(); sc.HasTraceID() {
			ctx.Response.Header.Set("X-Trace-Id", sc.TraceID().String())
		}

		span.End()
	}
}

func resultFromStatus(code int) string {
	if code >= 200 && code < 400 {
		return "success"
	}
	return "failed"
}
```

---

## 9. context.go（可选 — 多组件共享进程时）

当多个组件在同一进程中运行（如桌面端内嵌 runtime），不应多次调用 `Init()`。
内层组件通过 `WithFields` 在 ctx 级别打 tag 区分日志来源。

```go
package observability

import "context"

type ctxFieldsKey struct{}

// WithFields 在 context 中附加结构化字段，后续 Info/Warn/Error 自动携带
func WithFields(ctx context.Context, kvs ...any) context.Context {
	existing := fieldsFromContext(ctx)
	merged := make([]any, 0, len(existing)+len(kvs))
	merged = append(merged, existing...)
	merged = append(merged, kvs...)
	return context.WithValue(ctx, ctxFieldsKey{}, merged)
}

func fieldsFromContext(ctx context.Context) []any {
	if v, ok := ctx.Value(ctxFieldsKey{}).([]any); ok {
		return v
	}
	return nil
}
```

使用 `WithFields` 后，`logging.go` 的 `buildKVs` 需合并 ctx 字段：

```go
func buildKVs(ctx context.Context, action, result, errorKind string, err error, extra []any) []any {
	kvs := make([]any, 0, 10+len(extra))
	// ... action/result/error_kind/error_message 同上 ...

	// 合并 context fields
	if ctxFields := fieldsFromContext(ctx); len(ctxFields) > 0 {
		kvs = append(kvs, ctxFields...)
	}

	// 合并业务 KV（脱敏/截断同上）
	// ...
}
```

---

## 10. Go 依赖清单

生成代码后需要确保 `go.mod` 包含：

```
require (
    code.byted.org/gopkg/logs/v2                              v2.2.2
    code.byted.org/log_market/ttlogagent_gosdk/v4             // trace_middleware 需要 codec 包
    go.opentelemetry.io/otel                                   v1.28.0
    go.opentelemetry.io/otel/exporters/otlp/otlptrace         v1.28.0
    go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.28.0
    go.opentelemetry.io/otel/sdk                               v1.28.0
    go.opentelemetry.io/otel/trace                             v1.28.0
)
```

版本号以目标仓库已有版本为准；如果仓库尚未引入 OTel，使用最新稳定版。
