# 字节内部依赖 → 本地替代方案速查表

> 本文件是 `harness-local-twin` 技能的参考资料。
> 分析依赖时对照本表归类；未覆盖的依赖按 SKILL.md 中的启发规则判断。

---

## Table of Contents

- [bytedcli — 内部基础设施统一操作工具](#bytedcli--内部基础设施统一操作工具)
- [字节内部基础设施全景](#字节内部基础设施全景)
- [日志与观测](#日志与观测)
- [配置中心](#配置中心)
- [数据存储](#数据存储)
- [对象存储](#对象存储)
- [消息队列](#消息队列)
- [RPC 框架与 Overpass](#rpc-框架与-overpass)
- [服务发现与网关（常见启动拦路虎 #1）](#服务发现与网关常见启动拦路虎-1)
- [凭证管理（常见启动拦路虎 #2）](#凭证管理常见启动拦路虎-2)
- [认证与安全](#认证与安全)
- [部署与运行时](#部署与运行时)
- [内部工具 SDK](#内部工具-sdk)
- [docker-compose 参考模板](#docker-compose-参考模板)

---

## bytedcli — 内部基础设施统一操作工具

分析和调试内部依赖时，推荐使用 `bytedcli` 查看配置、查询实例、操作资源：

```bash
curl -o install-bytedcli.sh https://bytedcli.gf-preview.bytedance.net && bash install-bytedcli.sh

# 查看 TCC 配置（确认服务依赖哪些配置项）
bytedcli tcc config get --psm <PSM> --key <key>

# 查看 TCE 实例（确认 BOE 实例地址）
bytedcli tce --help

# 操作 TOS
bytedcli tos --help

# 查看服务树
bytedcli bytetree --help

# 查看 RDS / Redis / BMQ
bytedcli rds --help
bytedcli cache --help
bytedcli bmq --help
```

分析时可用 `bytedcli tcc config get` 导出服务的全部 TCC 配置，
作为本地 JSON 配置文件的数据源。

---

## 字节内部基础设施全景

> 分析服务依赖时，对照此表理解每个内部平台/服务的作用，判断本地化策略。

### 配置与部署

| 平台 | bytedcli 域 | 作用 | 本地化相关性 |
|------|------------|------|-------------|
| **TCC** | `tcc` | 配置中心——namespace/配置项/发布/审批，几乎所有业务配置从此拉取 | **高**：启动依赖，需本地 JSON 替代或连 BOE |
| **TCE** | `tce` | 容器部署平台——服务/集群/实例/发布工单管理 | 中：本地不需要部署，但 `TCE_*` 环境变量需手动设置 |
| **Bytetree** | `bytetree` | 服务树——PSM 节点搜索/详情/父子层级，管理服务归属和权限 | 低：分析时用于查 PSM 归属，运行时不直接依赖 |
| **Env** | `env` | 多环境管理——BOE/PPE 环境创建/设备绑定/TCE 服务升级 | 低：了解泳道，运行时不直接依赖 |
| **Settings** | `settings` | 运营配置平台——item/draft/review/deploy/白名单/AB 变量 | 中：类似 TCC，需查明配置项并本地替代 |

### 数据存储

| 平台 | bytedcli 域 | 作用 | 本地化相关性 |
|------|------------|------|-------------|
| **RDS** | `rds` | MySQL 数据库管理——库详情/表/schema/SQL 查询/慢查询/BPM 工单 | **高**：核心存储，本地 MySQL 容器或连 BOE |
| **Redis/Cache** | `cache` | Redis 缓存——服务搜索/慢日志/大 key/命令执行 | **高**：核心缓存，miniredis 或本地容器或连 BOE |
| **ByteDoc** | `bytedoc` | MongoDB 文档数据库——搜索/关注列表/集合/慢查询 | 中：mongodb-in-memory 或本地容器 |
| **TOS** | `tos` | 对象存储——bucket/用户信息/站点与 vregion | 中：MinIO 容器替代或本地文件 stub |
| **ES** | `es` | Elasticsearch——DSL 查询/mapping 查询与更新 | 中：本地 ES 容器或 mock |
| **Hive** | `hive` | 数仓——DataLeap 资产搜索/schema/lineage/partition/rows | 低：离线分析，本地开发通常不需要 |

### 消息队列

| 平台 | bytedcli 域 | 作用 | 本地化相关性 |
|------|------------|------|-------------|
| **BMQ** | `bmq` | Kafka 消息队列——topic/cluster/consumer/mirror 管理 | 中：本地 Kafka 容器替代，或 channel mock |

### 网络与网关

| 平台 | bytedcli 域 | 作用 | 本地化相关性 |
|------|------------|------|-------------|
| **AGW** | `agw` | API 网关——产品/服务/环境注册/IDL 更新与发布 | 中：本地直连后端跳过网关 |
| **BAM** | `bam` | PSM/方法/版本/IDL 管理——服务接口元信息 | 低：分析用，运行时不直接依赖 |
| **Netlink** | `netlink` | 域名治理——域名/路径/topology/servername 配置 | 低：本地用 localhost 替代 |
| **Neptune** | `neptune` | 流量治理——dispatch/stability/rate limit/security 配置 | 低：本地开发通常不需要 |

### 认证与安全

| 平台 | bytedcli 域 | 作用 | 本地化相关性 |
|------|------------|------|-------------|
| **DKMS** | `dkms` | 密钥管理——data key 查询/权限检查/授权 | 中：本地用环境变量替代密钥 |
| **KMS v2** | `kmsv2` | 新版密钥管理——keyring/customer key/ACL | 中：同上 |
| **IAM** | `iam` | 身份管理——员工信息查询 | 低：本地 bypass 或 mock |

### 代码与发布

| 平台 | bytedcli 域 | 作用 | 本地化相关性 |
|------|------------|------|-------------|
| **Codebase** | `codebase` | 代码仓库——MR/Review/Issue/CI/文件 | 低：开发工具，不影响服务运行 |
| **BITS** | `bits` | 研发流水线——develop 任务/lane/流水线/release | 低：CI/CD 工具 |
| **SCM** | `scm` | 软件配置管理——仓库/构建/版本 | 低：构建工具 |
| **Devflow** | `devflow` | 发布任务——创建发布任务/绑定 TCE/TCC | 低：发布流程 |
| **Overpass** | `overpass` | RPC 代码自动生成——`code.byted.org/overpass/*` 包的来源，消灭手动 `kitex_gen` | 低：生成的包是 Kitex 包装，本地化策略同 Kitex |

### 观测与监控

| 平台 | bytedcli 域 | 作用 | 本地化相关性 |
|------|------------|------|-------------|
| **Log** | `log` | 日志平台——PSM 日志/LogID 查询/实例日志/聚类 | 低：本地用 console 或 VictoriaLogs 替代 |
| **APM** | `apm` | 性能监控——service preview/QPS/下游/Redis 监控 | 低：本地用 OTel + VictoriaTraces 替代 |
| **Slardar** | `slardar` | 客户端监控——Web/App/OS 告警/JS Error/SOP | 低：客户端监控，服务端不直接依赖 |
| **Archer** | `archer` | 链路覆盖率——按 PSM+traceId 查询函数调用链路 | 低：调试工具 |
| **Cronjob** | `cronjob` | 定时任务——挂载/任务/执行记录/重跑/debug | 中：本地用 cron 或手动触发替代 |

### 数据分析与 AI

| 平台 | bytedcli 域 | 作用 | 本地化相关性 |
|------|------------|------|-------------|
| **Dorado** | `dorado` | 数据开发——project/task/instance/ad-hoc SQL | 低：离线数据 |
| **Aeolus** | `aeolus` | 报表——dashboard/dataset/SQL 查询 | 低：离线数据 |
| **Merlin** | `merlin` | ML 训练——job/trial/tracking/quota | 低：AI 训练平台 |
| **DataQ** | `dataq` | 海外 RDS 查询（i18n-tt 站点） | 低：特定场景 |
| **TQS** | `tqs` | Hive SQL 执行 | 低：离线查询 |

### 协作与文档

| 平台 | bytedcli 域 | 作用 | 本地化相关性 |
|------|------------|------|-------------|
| **Feishu** | `feishu` | 飞书——文档/Wiki/日历/任务/消息/Sheet/Bitable | 低：协作工具 |
| **Meego** | `meego` | 项目管理——工作项/视图/评论/图表 | 低：项目管理 |
| **Starling** | `starling` | 文案平台——业务线/项目/空间/文案搜索 | 低：国际化文案 |

---

## 日志与观测

| 内部包 | 功能 | 层级 | 本地替代方案 |
|--------|------|------|-------------|
| `code.byted.org/gopkg/logs/v2` | 结构化日志 SDK | **L1** | 直接可用——本地默认输出到 console/file；可选 `AppendWriter` 推送到 VictoriaLogs |
| `code.byted.org/gopkg/logs` (v1) | 旧版日志 SDK | **L1** | 同上，兼容 |
| `code.byted.org/log_market/ttlogagent_gosdk/v4` | LogAgent 推送 SDK（codec 包） | **L1** | 本地不推送，仅用作 logs/v2 KV 编码工具 |
| `code.byted.org/gopkg/metrics` | Metrics 打点 | **L1** | 本地 noop（不上报），或推到本地 VictoriaMetrics |
| `code.byted.org/trace/trace-sdk-go` | 内部 Trace SDK | **L1** | 替换为 OpenTelemetry 直接导出到 VictoriaTraces |
| `code.byted.org/microservice/slardar` | APM 接入 | **L1** | 移除或 noop，不影响业务逻辑 |

## 配置中心

| 内部包 | 功能 | 层级 | 本地替代方案 |
|--------|------|------|-------------|
| `code.byted.org/gopkg/tccclient` | TCC 配置中心 V2 | **L3** | **Mock Client**：用 `tccmock.GetMockTccClient` + `MustSetTccData` 预设 KV；**本地 JSON**：启动时从本地 JSON 文件读取配置，实现 `Get(ctx, key)` 接口 |
| `code.byted.org/gopkg/tccclient/v2` | TCC V2 别名 | **L3** | 同上 |
| `code.byted.org/middlewares/confx` | TCC 包装层 | **L3** | 同上，confx 底层走 tccclient |
| `code.byted.org/gopkg/env` | 环境变量读取（PSM、Region 等） | **L1** | 设置本地环境变量即可：`export PSM=local.dev.service` |

## 数据存储

### MySQL / RDS

| 内部包 | 功能 | 层级 | 本地替代方案 |
|--------|------|------|-------------|
| `code.byted.org/gorm/bytedgorm` | 字节 GORM 封装（自动连接 RDS） | **L2** | **本地 MySQL**：docker-compose 起 MySQL 5.7/8.0 容器，用标准 `gorm.Open(mysql.Open(dsn))` 替换 `bytedgorm.MySQL(psm, db)` |
| `code.byted.org/gorm/bytedgorm/v2` | bytedgorm V2 | **L2** | 同上 |
| `gorm.io/gorm` | 标准 GORM | **L0** | 直接可用 |

**SQLite 快速方案**（适合开发调试，不适合复杂 SQL）：
```go
import "gorm.io/driver/sqlite"
db, _ := gorm.Open(sqlite.Open("local.db"), &gorm.Config{})
```

**SmartUnit SDK 沙盒方案**：
```go
import "code.byted.org/smart-qa/smart_unit_help/smartunitbuilder/sqlmock"
handler := sqlmock.NewMysqlMockHandler(sqlmock.SQLITE3DriverName, "schema.sql", &dbName)
db := handler.GetConnection(ctx)
handler.PrepareDBWithData([]string{"fixtures/users.yml"})
```

### Redis

| 内部包 | 功能 | 层级 | 本地替代方案 |
|--------|------|------|-------------|
| `code.byted.org/kv/goredis` | 字节 Redis 客户端 | **L2** | **miniredis**（内存）：`github.com/alicebob/miniredis`，零外部依赖；**本地 Redis 容器**：docker-compose 起 Redis |
| `github.com/go-redis/redis` | 开源 Redis 客户端 | **L0** | 直接可用，连本地 Redis 容器 |

**miniredis 方案**：
```go
import "github.com/alicebob/miniredis"
srv, _ := miniredis.Run()
client := goredis.NewClientWithServers("", []string{srv.Addr()}, goredis.NewOption())
```

**SmartUnit SDK 方案**：
```go
import "code.byted.org/smart-qa/smart_unit_help/smartunitbuilder/storagemock"
client := storagemock.MustNewRedisClient("test_cluster")
```

### MongoDB / ByteDoc

| 内部包 | 功能 | 层级 | 本地替代方案 |
|--------|------|------|-------------|
| `code.byted.org/bytedoc/mongo-go-driver` | 字节 MongoDB 客户端 | **L2** | **mongodb-in-memory**：`code.byted.org/inf/mongodb-in-memory`（需 Go ≥ 1.17）；**本地 MongoDB 容器**：docker-compose 起 MongoDB |

**mongodb-in-memory 方案**：
```go
import mim "code.byted.org/inf/mongodb-in-memory"
server, _ := mim.Start(ctx, "5.0.2")
client, _ := mongo.Connect(ctx, options.Client().ApplyURI(server.URI()))
defer server.Stop(ctx)
```

## 对象存储

| 内部包 | 功能 | 层级 | 本地替代方案 |
|--------|------|------|-------------|
| `code.byted.org/gopkg/tos` | TOS 对象存储 | **L3** | **本地文件系统 Stub**：实现 `PutObject`/`GetObject` interface，读写本地目录；**MinIO 容器**：S3 兼容的开源对象存储 |

**MinIO docker-compose**：
```yaml
minio:
  image: minio/minio:latest
  ports: ["9000:9000", "9001:9001"]
  command: server /data --console-address ":9001"
  environment:
    MINIO_ROOT_USER: minioadmin
    MINIO_ROOT_PASSWORD: minioadmin
```

需要写 adapter 将 TOS API 映射到 S3 API，或直接实现 TOS interface stub。

## 消息队列

| 内部包 | 功能 | 层级 | 本地替代方案 |
|--------|------|------|-------------|
| `code.byted.org/mq/bmq-sdk-go` | BMQ 消息队列 | **L3** | **本地 Kafka 容器** + adapter；或 **channel mock**（如果只需要同步消费） |
| `code.byted.org/mq/rocketmq-go-client` | RocketMQ 客户端 | **L3** | 本地 RocketMQ 容器 |

**Kafka docker-compose**（替代 BMQ）：
```yaml
kafka:
  image: confluentinc/cp-kafka:latest
  ports: ["9092:9092"]
  environment:
    KAFKA_NODE_ID: 1
    KAFKA_PROCESS_ROLES: broker,controller
    KAFKA_CONTROLLER_QUORUM_VOTERS: 1@kafka:9093
    KAFKA_LISTENERS: PLAINTEXT://0.0.0.0:9092,CONTROLLER://0.0.0.0:9093
    KAFKA_CONTROLLER_LISTENER_NAMES: CONTROLLER
    KAFKA_INTER_BROKER_LISTENER_NAME: PLAINTEXT
    CLUSTER_ID: local-twin
```

## RPC 框架与 Overpass

| 内部包 | 功能 | 层级 | 本地替代方案 |
|--------|------|------|-------------|
| `github.com/cloudwego/hertz` | HTTP 框架（开源） | **L0** | 直接可用 |
| `github.com/cloudwego/kitex` | RPC 框架（开源） | **L1** | 直接可用，本地需配置直连模式（跳过服务发现） |
| `code.byted.org/kite/kitex` | Kitex 内部版本 | **L1** | 用开源 `github.com/cloudwego/kitex` 替代，或配置 `WithHostPorts` 直连 |
| `code.byted.org/microservice/hertz` | Hertz 内部版本 | **L1** | 用开源 `github.com/cloudwego/hertz` 替代 |
| `code.byted.org/overpass/*` | Overpass 自动生成的 RPC 客户端包 | **L1** | 底层走 Kitex，直连策略同 Kitex |

### Overpass 说明

[Overpass](https://overpass.bytedance.net/) 是 RPC 调用代码自动生成服务。传统方式下每个仓库各自维护 `kitex_gen`（数十万行生成代码），
Overpass 将其统一收敛到 `code.byted.org/overpass/{P_S_M}` 仓库，业务侧只需 1 行 import + 1 行调用。

**分析时注意**：go.mod 中大量 `code.byted.org/overpass/*` 依赖不需要恐慌——
它们只是 Kitex client 的自动生成包装，底层 RPC 机制与直接使用 Kitex 完全一致。
本地化策略和原生 Kitex 相同：配置 `WithHostPorts` 直连或通过 Consul 连 BOE。

### Kitex 本地直连（跳过 Consul 服务发现）

```go
client, _ := xxxservice.NewClient("service-name",
    client.WithHostPorts("127.0.0.1:8888"),
)
```

## 服务发现与网关（常见启动拦路虎 #1）

> **Consul 是内场服务本地化最常见的启动阻塞点之一。**
> Kitex/Hertz client 默认通过 Consul resolver 查找下游，本地无 Consul 时直接 panic。

| 内部包/服务 | 功能 | 层级 | 本地替代方案 |
|------------|------|------|-------------|
| Consul / service mesh | 服务发现 | **L1**（有 devbox 时） | 设环境变量指向 devbox Consul（见下方） |
| AGW | API 网关 | **L4** | 本地直连后端服务，跳过网关层 |
| `code.byted.org/gopkg/naming` | 服务命名 SDK | **L4** | Mock 注册，返回本地地址 |

### macOS 本地连 BOE Consul（推荐方案）

只需设置环境变量。**优先使用公共 Consul 代理**（无需 devbox）：

```bash
export CONSUL_HTTP_HOST=common-consul-boe.bytedance.net
export CONSUL_HTTP_PORT=2280
```

如果公共代理不可用，也可以指向自己的 devbox IP（没有 devbox 到 [字节云](http://cloud.bytedance.net/devbox) 申请）：

```bash
export CONSUL_HTTP_HOST=<devbox-ip>
export CONSUL_HTTP_PORT=2280
```

如果出现 `idc=-` 错误（`no available translator for {PSM}@-#0`），追加：
```bash
export RUNTIME_IDC_NAME=boe
```

IPv6 only 实例需要额外设置：
```bash
export BYTED_HOST_IPV6=1   # 非空即可，让 consul SDK 走 v6 逻辑
```

**IDE 注意**：在终端设置的环境变量不会自动传到 GoLand/VSCode 的运行环境。
需要在 IDE 的 `Run/Debug Configurations` → `Environment variables` 中手动添加，
或者关闭 IDE 重启以继承系统环境变量。

### 完全本地化方案（无 devbox 时）

Kitex/Hertz client 改为 `WithHostPorts` 直连，跳过 Consul：

```go
client, _ := xxxservice.NewClient("service-name",
    client.WithHostPorts("127.0.0.1:8888"),
)
```

或本地起 Consul dev agent：`consul agent -dev`

**检测方法**：搜索 `consul`、`naming`、`resolver`、`discovery` 关键词，
以及 Kitex client 创建处是否缺少 `WithHostPorts` 选项。

## 凭证管理（常见启动拦路虎 #2）

> **doas 是很多内部 SDK 初始化的隐式依赖。**
> TCC、TOS、ByteDoc、DKMS、Redis、Kafka、RDS 等 SDK 启动时需要 ZTI Token（`SEC_TOKEN_STRING`），
> 本地未获取 token 时这些 SDK 初始化直接失败。

| 工具/服务 | 功能 | 层级 | 本地替代方案 |
|----------|------|------|-------------|
| `doas` | ZTI Token 获取 | **L1**（已安装时） | 用 doas 包裹启动命令（见下方） |

### doas 快速上手

**前置条件**：
1. 账号在目标 PSM 下拥有 `owner`、`zti.doas.token` 或 `zti.authorized_user.{partition}` 角色之一
2. 完成 Kerberos 认证：
   ```bash
   kinit your_email_prefix@BYTEDANCE.COM
   # 输入 SSO 密码（不是飞书密码）
   ```

**方式一：doas 包裹命令（推荐）**

直接在原有启动命令前加 `doas`，自动注入 `SEC_TOKEN_STRING` 环境变量到子进程：

```bash
# 启动 Go 服务
doas -t zti -e boe -p <你的PSM> go run .

# 跑测试
doas -t zti -e boe -p <你的PSM> go test ./...

# 启动 Node.js
doas -t zti -e boe -p <你的PSM> npm run dev
```

核心参数：
- `-t zti` — Token 类型（推荐 zti）
- `-e boe` — 环境（本地开发用 boe）
- `-p <PSM>` — 服务 PSM（必需）

**方式二：打印 Token 后配到 IDE**

```bash
doas -t zti -e boe -p <你的PSM> -V 0 --print-env
# 输出: export SEC_TOKEN_STRING="ey..."
```

然后在 IDE 中配置：
- **GoLand**：`Run/Debug Configurations` → `Environment variables` → 添加 `SEC_TOKEN_STRING=ey...`
- **VSCode**：`launch.json` 的 `env` 字段添加 `"SEC_TOKEN_STRING": "ey..."`
- 也可使用 GoLand 的 ByteTool 插件或 Doas IDE 插件自动注入

### 分析时检测 doas 依赖

**检测方法**：搜索 `SEC_TOKEN_STRING`、`doas`、`token`、`credential`、`WithCredentials` 关键词；
检查哪些 SDK 初始化依赖这个环境变量。

**常见隐式依赖 doas 的 SDK**：
- `tccclient`（TCC 配置中心）
- `code.byted.org/gopkg/tos`（对象存储）
- `code.byted.org/bytedoc/mongo-go-driver`（ByteDoc/MongoDB）
- `code.byted.org/kv/goredis`（Redis，走 ACL 时）
- `code.byted.org/security/dkms-sdk-go`（密钥管理）

## 认证与安全

> **服务间 PSM ACL 鉴权是线上专属机制，BOE 大概率没有，不是本地化障碍。**

| 内部包/服务 | 功能 | 层级 | 本地替代方案 |
|------------|------|------|-------------|
| SSO / IAM | 统一登录认证 | **L4** | **本地 bypass**：开发模式跳过 SSO，注入固定 user_id/session；或实现 AuthMiddleware interface 的 noop 版本 |
| `code.byted.org/security/dkms-sdk-go` | 密钥管理 | **L3** | 从环境变量 / 本地文件读取 secret，不走 DKMS API |
| `code.byted.org/security/iam-sdk-go` | IAM 鉴权 | **L4** | 本地 bypass 或 mock 固定权限 |

## 部署与运行时

| 内部包/服务 | 功能 | 层级 | 本地替代方案 |
|------------|------|------|-------------|
| TCE 环境变量 (`TCE_*`) | 部署元信息 | **L1** | 手动设置：`export TCE_PSM=local.dev.service` |
| FaaS 运行时 | 函数计算 | **L5** | 暂不可本地化（需要完整运行时） |
| Cron / 定时任务 | 内部调度平台 | **L3** | 本地 cron 或手动触发 |

## 内部工具 SDK

| 内部包 | 功能 | 层级 | 本地替代方案 |
|--------|------|------|-------------|
| `code.byted.org/gopkg/mockito` | Mock 框架 | **L0** | 直接可用（测试时） |
| `code.byted.org/smart-qa/smart_unit_help` | SmartUnit 测试辅助 | **L0** | 直接可用（提供 TCC/Redis/MySQL/ByteDoc mock 工具） |
| `code.byted.org/gopkg/tos` 的 `WithCredentials` | TOS 凭证 | **L3** | 本地 MinIO + 写 adapter |

---

## docker-compose 参考模板

L2 层依赖的本地容器组合（按需裁剪）：

```yaml
# docker-compose.local-twin.yml
services:
  mysql:
    image: mysql:8.0
    ports: ["3306:3306"]
    environment:
      MYSQL_ROOT_PASSWORD: root123
      MYSQL_DATABASE: local_dev
    restart: unless-stopped

  redis:
    image: redis:7-alpine
    ports: ["6379:6379"]
    restart: unless-stopped

  mongo:
    image: mongo:5.0
    ports: ["27017:27017"]
    restart: unless-stopped

  minio:
    image: minio/minio:latest
    ports: ["9000:9000", "9001:9001"]
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER: minioadmin
      MINIO_ROOT_PASSWORD: minioadmin
    restart: unless-stopped

  # 观测栈（可选，参见 harness-obs-init）
  victoria-logs:
    image: victoriametrics/victoria-logs:latest
    ports: ["9428:9428"]
    command: ["-httpListenAddr=:9428"]
    restart: unless-stopped

  victoria-traces:
    image: victoriametrics/victoria-traces:latest
    ports: ["10428:10428"]
    command: ["-httpListenAddr=:10428"]
    restart: unless-stopped
```
