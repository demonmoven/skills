# 基础设施编排模板

> 本文件是 `harness-obs-init` 技能的参考资料。
> 提供观测后端的 docker-compose 模板和替代方案对比。

---

## Table of Contents

- [推荐方案：VictoriaLogs + VictoriaTraces](#推荐方案victorialogs--victoriatraces)
- [替代方案 A：Grafana LGTM Stack](#替代方案-agrafana-lgtm-stack)
- [替代方案 B：Jaeger（仅 trace）](#替代方案-bjaeger仅-trace)
- [替代方案 C：云服务](#替代方案-c云服务)
- [团队共享实例](#团队共享实例)
- [可选附加：Jaeger UI 对接 VictoriaTraces](#可选附加jaeger-ui-对接-victoriatraces)

---

## 推荐方案：VictoriaLogs + VictoriaTraces

### 为什么推荐

| 优势 | 说明 |
|------|------|
| 极简 | 2 个容器，无 Collector/Grafana/Agent |
| 直接 push | 应用直推日志和 trace，无中间层 |
| 内置查询 | LogsQL（日志）+ Jaeger API（trace），curl 直接用 |
| 低资源 | 单容器内存 < 100MB，适合本地开发 |
| 免费 | Apache-2.0 / AGPL，无商业限制 |
| VictoriaTraces 兼容 Jaeger | 查询 API 与 Jaeger 完全兼容，可直接用 Jaeger UI |

### docker-compose.obs.yml

```yaml
# 本地观测栈：VictoriaLogs + VictoriaTraces
# 启动：docker compose -f docker-compose.obs.yml up -d
# 停止：docker compose -f docker-compose.obs.yml down

services:
  victoria-logs:
    image: victoriametrics/victoria-logs:latest
    container_name: ${COMPOSE_PROJECT_NAME:-obs}-victoria-logs
    ports:
      - "9428:9428"
    command:
      - "-httpListenAddr=:9428"
    restart: unless-stopped
    # 数据持久化（可选，取消注释后重启不丢数据）
    # volumes:
    #   - vlogs-data:/victoria-logs-data

  victoria-traces:
    image: victoriametrics/victoria-traces:latest
    container_name: ${COMPOSE_PROJECT_NAME:-obs}-victoria-traces
    ports:
      - "10428:10428"
    command:
      - "-httpListenAddr=:10428"
    restart: unless-stopped
    # volumes:
    #   - vtraces-data:/victoria-traces-data

# volumes:
#   vlogs-data:
#   vtraces-data:
```

### 端口说明

| 服务 | 端口 | 协议 | 用途 |
|------|------|------|------|
| VictoriaLogs | 9428 | HTTP | jsonline push + LogsQL query |
| VictoriaTraces | 10428 | HTTP | OTLP push + Jaeger query API |

### 快速验证

```bash
# 启动
docker compose -f docker-compose.obs.yml up -d

# 验证 VictoriaLogs
curl http://localhost:9428/select/logsql/query?query=*&limit=5

# 验证 VictoriaTraces（Jaeger API）
curl http://localhost:10428/select/jaeger/api/services

# 手动推送测试日志
curl -X POST 'http://localhost:9428/insert/jsonline?_stream_fields=service,level&_msg_field=_msg&_time_field=time' \
  -H 'Content-Type: application/x-ndjson' \
  -d '{"_msg":"test log","level":"info","service":"test","time":"2025-01-01T00:00:00Z","action":"test.ping","result":"success"}'

# 查询测试日志
curl 'http://localhost:9428/select/logsql/query?query=action:test.ping&limit=5'
```

---

## 替代方案 A：Grafana LGTM Stack

适合需要可视化 UI 的团队。

```yaml
# Loki (日志) + Tempo (trace) + Grafana (UI)
services:
  loki:
    image: grafana/loki:latest
    ports: ["3100:3100"]
    command: -config.file=/etc/loki/local-config.yaml
    restart: unless-stopped

  tempo:
    image: grafana/tempo:latest
    ports:
      - "3200:3200"    # HTTP
      - "4318:4318"    # OTLP HTTP
    command: -config.file=/etc/tempo/tempo.yaml
    restart: unless-stopped

  grafana:
    image: grafana/grafana:latest
    ports: ["3000:3000"]
    environment:
      - GF_AUTH_ANONYMOUS_ENABLED=true
      - GF_AUTH_ANONYMOUS_ORG_ROLE=Admin
    restart: unless-stopped
```

**对代码的影响**：
- VLogsWriter 需替换为 Loki push API（`/loki/api/v1/push`，protobuf 或 JSON）
- Tracer endpoint 改为 Tempo 的 `localhost:4318`
- 查询走 Grafana UI 或 LogQL/TraceQL API

**优缺点**：

| 维度 | LGTM | Victoria |
|------|------|---------|
| 容器数 | 3（+Grafana） | 2 |
| 可视化 | Grafana 内置 | 无 UI（可选接 Jaeger UI） |
| 资源占用 | 较高 | 低 |
| Push 协议 | Loki protobuf / OTLP | jsonline（简单） |
| 查询 | LogQL + TraceQL（功能强） | LogsQL + Jaeger API |
| 复杂度 | 需配置 datasource | 开箱即用 |

---

## 替代方案 B：Jaeger（仅 trace）

只需要链路追踪，不需要日志推送时。

```yaml
services:
  jaeger:
    image: jaegertracing/jaeger:latest
    ports:
      - "16686:16686"  # Jaeger UI
      - "4318:4318"    # OTLP HTTP
    environment:
      - COLLECTOR_OTLP_ENABLED=true
    restart: unless-stopped
```

**对代码的影响**：
- Tracer endpoint 改为 `localhost:4318`
- 日志继续留在 console/file，不做远程推送
- Jaeger UI 在 `http://localhost:16686`

---

## 替代方案 C：云服务

生产环境或团队共享场景。

| 云服务 | 日志 | Trace | 适用 |
|--------|------|-------|------|
| Datadog | ✅ | ✅ | 已有 Datadog 账号的团队 |
| Grafana Cloud | ✅ (Loki) | ✅ (Tempo) | 免费套餐可用 |
| AWS CloudWatch + X-Ray | ✅ | ✅ | AWS 技术栈 |
| 字节内部 LogAgent + APM | ✅ | ✅ | 字节内网，走 PSM 自动接入 |

**对代码的影响**：
- `VLogsWriter` 替换为对应 SDK 的 writer
- Tracer exporter 替换为对应 SDK 的 exporter
- `Init()` 的 Option 模式使得替换只需改 init.go 一个文件

---

## 团队共享实例

当团队有共享开发机（如 devgpu）时，可在上面部署一套长期运行的 Victoria 实例，
本地开发通过环境变量指向共享实例：

```bash
{PREFIX}_LOCAL_OBS=1 \
{PREFIX}_VLOGS_ENDPOINT=http://<shared-ip>:9428 \
{PREFIX}_VTRACES_ENDPOINT=<shared-ip>:10428 \
go run .
```

好处：
- 多人的日志和 trace 汇聚到同一处，便于联调排障
- 不需要每台开发机跑 Docker
- 数据天然持久化

---

## 可选附加：Jaeger UI 对接 VictoriaTraces

VictoriaTraces 原生兼容 Jaeger query API，可以用 Jaeger UI 做可视化：

```yaml
services:
  # ... victoria-traces 配置同上 ...

  jaeger-ui:
    image: jaegertracing/jaeger-query:latest
    ports: ["16686:16686"]
    environment:
      - GRPC_STORAGE_SERVER=victoria-traces:10428
      - SPAN_STORAGE_TYPE=grpc
    depends_on:
      - victoria-traces
    restart: unless-stopped
```

或者直接用 VictoriaTraces 内置的 Jaeger API：
```
http://localhost:10428/select/jaeger/api/traces/<traceID>
http://localhost:10428/select/jaeger/api/services
```
