> **Action: `local-twin`** — 由 `/xdev:harness-kit local-twin` 路由调用。
> 原 skill: `harness-local-twin` (author: guoshuai.030, version: 1.5)

# Harness Local Twin — 内场服务本地化分析

## 核心理念

> **每个 git worktree 独立启动一个应用实例，Agent 每做一个改动都能拉起一份独立的应用来测。**
> — OpenAI Harness Engineering

本地孪生的目标**不是服务的完全本地化**，而是：

**让 Agent 能在本地快速 `go run`，改完代码后秒级重启验证，无需走 BOE 编译部署流程。**

因此分析的判断标准只有一个：**是否阻塞本地 `go run`？**

- **不阻塞启动的** → 不需要本地化（埋点、观测、上报、metrics 等，启动时静默跳过即可）
- **本地能直连的** → 不需要本地化（BOE 的 Consul、Redis、MySQL 等基础设施，配个环境变量就能连）
- **阻塞启动的** → 必须处理（`MustInit` panic、连不上就 Fatal 的 SDK）
- **需要本地数据控制的** → 建议本地化（数据库——Agent 需要自由读写测试数据而不污染 BOE）
- **不能随意调试的上下游** → 建议本地化（调用有副作用、或 BOE 实例不稳定的下游服务）

**本技能只做分析和建议，不执行改造。**
改造实施建议使用 `harness-exec-plan` 创建执行计划。

## 激活后参考

- [references/BYTED-DEPENDENCY-CATALOG.md](references/BYTED-DEPENDENCY-CATALOG.md) — 字节内部依赖 → 本地替代方案速查表

## 关键背景知识

### 内场服务的典型技术栈

- **RPC/HTTP 框架**：绝大多数服务使用 Kitex（RPC）或 Hertz（HTTP），两者均为 CloudWeGo 开源框架，本地可直接启动
- **服务发现**：通过 Consul 做服务注册与发现。Kitex/Hertz client 默认走 Consul resolver，本地需改为 `WithHostPorts` 直连或搭本地 Consul agent
- **配置中心**：TCC（通过 `tccclient`）是标配，几乎所有业务配置从 TCC 拉取

### 推荐工具：bytedcli

对于 TCC 配置查看/修改、TCE 部署查询、TOS 操作、RDS/Redis/BMQ 等内部基础设施交互，
推荐使用 `bytedcli`（字节内部 CLI 工具集）。

安装：
```bash
curl -o install-bytedcli.sh https://bytedcli.gf-preview.bytedance.net && bash install-bytedcli.sh
```

或：

```bash
NPM_CONFIG_REGISTRY=http://bnpm.byted.org npm install -g @bytedance-dev/bytedcli@latest
```

常用命令：
```bash
bytedcli tcc config get --psm <PSM> --key <key>       # 查 TCC 配置
bytedcli tce --help                                     # TCE 部署相关
bytedcli tos --help                                     # TOS 对象存储
bytedcli rds --help                                     # RDS 数据库
bytedcli bytetree --help                                # 服务树查询
```

分析依赖时可通过 `bytedcli` 快速确认服务的 TCC 配置内容、TCE 实例状态等，
辅助判断哪些配置项需要本地化。

### 服务间鉴权——不是本地化障碍

字节内场的服务间鉴权基于 **PSM ACL 机制**（按调用方 PSM 授权）。
关键事实：**这套鉴权只在线上（Prod）生效，BOE 环境大概率没有。**
因此服务间鉴权**不构成本地化的阻塞点**，分析时可标记为"线上专属，本地无需处理"。

### 环境分层与网络可达性

| 环境 | 标识 | 本地可达 | 隔离情况 | 说明 |
|------|------|---------|---------|------|
| **BOE 基准** | `prod`（BOE 内） | ✅ | 与线上完全隔离（流量+存储） | 代码与线上一致，功能环境缺失时的兜底 |
| **BOE 功能** | `boe_xxx` | ✅ | 与线上完全隔离 | 按 Feature 创建的泳道环境 |
| **PPE** | `ppe_xxx` | ❌ | 流量隔离，**但 DB 与 Prod 共享** | 预发布环境，不应从本地随意连接 |
| **Prod** | `prod` | ❌ | — | 线上环境 |

**BOE 回落是最省力的本地化策略**：BOE 与线上完全隔离（流量和存储都独立），本地开发连 BOE 是安全的。
对于 L2-L4 层依赖，如果能直连 BOE 环境的对应服务，就不需要在本地搭 MySQL/Redis/Consul 等基础设施。

**泳道（Swimlane）**：多人同时开发同一服务时，各自创建 BOE 功能环境（`boe_xxx`），
流量通过标识各行其道。功能环境中缺失的下游服务会自动兜底到 BOE 基准环境。
本地开发时一般连 BOE 基准即可，如需连指定泳道需配置 `x-tt-env` header。

**环境相关环境变量**：
- `TCE_ENV` / `SERVICE_ENV` — 服务运行环境标识，本地可设为对应 BOE 环境名
- `RUNTIME_IDC_NAME=boe` — 配合 Consul 使用，标识当前 IDC

### 两大常见启动拦路虎

**1. Consul 服务发现**

Kitex/Hertz client 默认通过 Consul resolver 查找下游服务地址。
本地没有 Consul agent 时，client 初始化直接报错或 panic。

解法（按优先级）：
- **连公共 Consul 代理**（最快，无需 devbox）：设置 `CONSUL_HTTP_HOST=common-consul-boe.bytedance.net` + `CONSUL_HTTP_PORT=2280`，
  出现 `idc=-` 错误时追加 `RUNTIME_IDC_NAME=boe`
- 配置 `WithHostPorts("ip:port")` 直连（指向 BOE 实例或本地实例）
- 本地起 Consul dev agent：`consul agent -dev`

**2. doas 鉴权（ZTI Token）**

`doas` 是字节内部的凭证管理工具。TCC、TOS、ByteDoc、Redis、DKMS 等 SDK
初始化时需要 `SEC_TOKEN_STRING` 环境变量，该变量由 doas 注入。

解法（按优先级）：
- **doas 包裹启动命令**（最快）：`doas -t zti -e boe -p <PSM> go run .`
  前提：先 `kinit` 完成 Kerberos 认证，且账号有目标 PSM 的 owner/doas 权限
- **打印 Token 配到 IDE**：`doas -t zti -e boe -p <PSM> -V 0 --print-env`，
  拿到 `SEC_TOKEN_STRING` 配到 GoLand/VSCode 环境变量
- 对不需要的 SDK，在初始化链路中加 noop 分支绕过

分析时应检查每个内部 SDK 初始化是否隐式依赖 `SEC_TOKEN_STRING`。
详见 [BYTED-DEPENDENCY-CATALOG.md](references/BYTED-DEPENDENCY-CATALOG.md) 中的具体操作步骤。

**auth 失败排障流程**：
当本地启动服务因 auth/token 问题失败时，按以下顺序排查：
1. 检查环境变量 `SEC_TOKEN_STRING` 是否已设置：`echo $SEC_TOKEN_STRING`
2. 如果为空或已过期，用 doas 重新生成：`doas -t zti -e boe -p <PSM> -V 0 --print-env`
3. 将新 token 注入环境后重试启动
4. 如果仍然失败，提醒用户：可能是账号本身缺少目标 PSM 的权限（需要 `owner`、`zti.doas.token` 或 `zti.authorized_user.{partition}` 角色之一），请联系 PSM 负责人或在 ByteTree 服务树上确认权限

## 本地化决策树

分析每个依赖时，按以下顺序判断：

```
该依赖是否阻塞 go run 启动？
├─ 否 → 跳过，不需要本地化（metrics/埋点/上报/APM 等）
└─ 是 → 本地能否直连 BOE 实例？
    ├─ 能 → 配环境变量直连 BOE，不需要改代码
    │   （Consul/Redis/MySQL/RPC 下游等）
    └─ 不能，或者需要本地数据控制 →
        该依赖是否需要 Agent 自由读写数据？
        ├─ 是 → 本地化（数据库、缓存——Agent 需要自由操控测试数据）
        └─ 否 → 该下游调用是否有副作用或不稳定？
            ├─ 是 → 本地 mock/stub
            └─ 否 → 直连 BOE
```

**核心原则**：能直连 BOE 的就直连，能 noop 的就 noop，**只本地化那些真正挡路的**。

## 本地化难度分层（L0–L5）

| 层级 | 类别 | 判断标准 | 处理方式 | 典型工作量 |
|------|------|---------|----------|-----------|
| **L0** | 无需处理 | 不阻塞启动，或纯开源库 | 原样保留 | 0 |
| **L1** | noop/跳过 | 阻塞启动但可通过环境变量/配置关闭 | 设环境变量或配置项跳过 | < 1h |
| **L2** | 直连 BOE | 阻塞启动，但本地能直连 BOE 实例 | 配 Consul/endpoint 环境变量 | < 1h |
| **L3** | 本地容器 | 阻塞启动，且需要本地数据控制（DB/缓存） | docker-compose 起本地实例 | 1-4h |
| **L4** | Mock/Stub | 阻塞启动，无法直连也无开源等价，需写 stub | 实现 interface stub 或 bypass | 4h-3d |
| **L5** | 功能裁剪 | 阻塞启动且无合理替代 | 在 init 链路中加开关跳过 | 视情况 |

## 完整流程

### Step 1: 确定分析范围

向用户确认：

- 目标仓库路径
- 服务入口（`main.go` 位置或 `cmd/` 目录）
- 本地化目标（完整运行 vs 核心链路可调试 vs 仅单测可跑）
- **网络条件**：本地是否能访问 BOE 环境？（直接影响方案选择）
- **doas 状态**：本地 doas 是否可用？（`doas whoami` 检查）
- 是否有已知的本地化尝试（已有 docker-compose、已有 mock 包等）

**多入口 monorepo 处理**：如果仓库有多个 `cmd/*/main.go` 或 `scripts/*/main.go`，
先列出所有入口及其 PSM，再按以下规则确定分析优先级：
1. 先分析主服务（通常代码量最大、RPC 注册最多的入口）
2. 其他入口通常共享 `modules/` 或 `internal/` 下的 infra 代码——主服务的改造成果可直接复用
3. FaaS 类入口（`bytefaas`/`faasx`）单独标注：FaaS 运行时无法本地启动，需提取 handler 为普通 HTTP server 才能本地运行

### Step 2: 静态依赖扫描

#### 2.0 已有本地化资产检测（优先执行）

在开始依赖扫描前，**先搜索仓库中已有的本地化基础设施**——这些是最大的捷径：

| 搜索模式 | 含义 |
|----------|------|
| `in_mem_*`、`inmem_*`、`memory_*` 文件名 | 已有内存替代实现（如内存 MySQL、内存 Redis） |
| `mock_*` 目录或 `*_mock.go` 文件 | 已有 mock 实现（mockgen 产物） |
| `miniredis` import | 已集成 miniredis |
| `go-mysql-server` 或 `dolthub` import | 已集成内存 MySQL |
| `sqlite` import | 已集成 SQLite 替代 |
| `docker-compose*.yml` | 已有容器编排 |
| `conf/config.local.*` 或 `config.dev.*` | 已有本地配置文件 |
| `conf/config.boe.*` | 已有 BOE 配置文件（可直接用于 BOE 路径） |
| `_test.go` 中的 `TestMain` | 可能包含 mock 初始化代码可以复用 |

**关键判断**：这些资产如果存在但只在 `_test.go` 中使用，
P0 的核心工作就是**把它们从 test-only 升级到可注入生产启动路径**，
而不是从零开始写 mock。在报告中单独列出这些已有资产。

#### 2.1 go.mod 分析

读取所有 `go.mod`（monorepo 可能有多个），将依赖分为三类：

| 前缀 | 分类 | 处理 |
|------|------|------|
| `code.byted.org/` | 字节内部包 | 逐个分析，按 L0-L5 分层 |
| `github.com/` / `golang.org/` 等 | 开源包 | 标记为 L0 |
| 其他 `*.byted.org` / 内部域名 | 内部基础设施 | 重点分析 |

**关键指标**：统计内部包占比，比例越高本地化难度越大。

#### 2.2 import 路径分析

扫描所有 `.go` 文件的 import 块，提取 `code.byted.org/` 开头的 import，
按**包前缀**聚合并统计引用次数，识别出高频依赖：

```
code.byted.org/gopkg/logs/v2          → 42 files  (日志 SDK)
code.byted.org/gopkg/tccclient        → 18 files  (配置中心)
code.byted.org/kv/goredis             → 12 files  (Redis)
code.byted.org/gorm/bytedgorm         → 8 files   (MySQL ORM)
code.byted.org/gopkg/tos              → 5 files   (对象存储)
...
```

#### 2.3 配置与环境变量扫描

搜索以下模式，识别运行时依赖：

| 搜索模式 | 识别为 |
|----------|--------|
| `os.Getenv("TCE_*")` | TCE 部署环境依赖 |
| `os.Getenv("PSM")` / `env.PSM` | 服务注册依赖 |
| `tccclient.NewClientV2` / `confx` | TCC 配置中心 |
| `consul` / `service_discovery` | 服务发现 |
| `dkms` / `kms` | 密钥管理 |

#### 2.4 网络调用扫描

搜索硬编码的内部域名和 IP：

| 搜索模式 | 识别为 |
|----------|--------|
| `*.byted.org` / `*.bytedance.net` | 内部 HTTP 依赖 |
| `10.*.*.*` / `100.*.*.*` | 内部网络 IP |
| 显式 gRPC/Kitex client 创建 | 内部 RPC 依赖 |

#### 2.5 接口抽象检测

检查关键依赖是否已通过 interface 抽象：

- 有 interface → 本地替换容易（只需实现 interface）
- 直接使用具体类型 → 需要先重构出 interface 再替换

### Step 3: 依赖分层归类

将 Step 2 发现的每个依赖，对照 [BYTED-DEPENDENCY-CATALOG.md](references/BYTED-DEPENDENCY-CATALOG.md) 归入 L0-L5。

对于目录中未覆盖的依赖，按以下启发规则判断：

| 信号 | 可能层级 |
|------|---------|
| 纯工具/编解码类（`codec`、`env`、`util`） | L0-L1 |
| 有 `Mock` / `Fake` / `InMemory` 实现的 | L2-L3 |
| 需要网络连接才能初始化的 | L3-L5 |
| `init()` 中 panic 或 `log.Fatal` 的 | L4-L5（阻塞启动） |

### Step 4: 启动链路分析

#### 4.1 启动阻塞点搜索

**内部依赖数量不等于启动阻塞点数量**。一个仓库可能有 300+ 个内部依赖，
但真正阻塞 `go run` 的往往只有 10-20 个 init 调用。
分析的重点是找到**这些硬阻塞点**，而不是逐个分析所有依赖。

在 `main.go` 及其直接调用的 init 文件中，搜索以下**硬阻塞模式**：

| 搜索模式 | 严重度 | 说明 |
|----------|--------|------|
| `MustInit(` / `MustNew(` | 硬阻塞 | 内部 SDK 常见的"失败即 panic"初始化函数 |
| `panic(` | 硬阻塞 | 显式 panic |
| `log.Fatal(` / `logs.Fatal(` | 硬阻塞 | 致命日志 + os.Exit |
| `os.Exit(` | 硬阻塞 | 直接退出 |

对每个找到的阻塞点，追溯其依赖链：
```
MustInit() → 调用了什么 SDK → 需要什么外部资源（Consul? doas? 网络?）
```

#### 4.2 init 调用链速写

从 `main.go` 出发，画出启动顺序的依赖骨架：

```
main()
  ├─ config.Init()         ← TCC? 需要 doas token? 可否读本地 JSON/YAML?
  ├─ db.Init()             ← RDS? 走 Consul 解析? 连 BOE 还是本地 MySQL?
  ├─ cache.Init()          ← Redis? 走 Consul 解析? 连 BOE 还是本地?
  ├─ tos.Init()            ← TOS? 需要 doas 凭证?
  ├─ rpc.Init()            ← Kitex client? 走 Consul? 有几个下游?
  ├─ mq.Init()             ← RocketMQ/Kafka? 走 Consul?
  ├─ server.Start()        ← Hertz/Kitex server? 端口冲突?
  └─ ...
```

#### 4.3 重点关注

- **Consul 依赖**：所有 Kitex/Hertz client 创建处、`bytedgorm.MySQL(psm, db)` 调用处、
  `goredis.NewClientWithOption(psm)` 调用处——这些都隐式走 Consul resolver
- **doas 依赖**：所有内部 SDK 初始化处（TCC/TOS/ByteDoc/DKMS/Redis），
  检查是否隐式依赖 `SEC_TOKEN_STRING`
- **FaaS 入口**：`bytefaas.Start()` / `faasx.New()` 等 FaaS 运行时初始化——
  这些无法在本地运行，需要标注为"提取 handler 为独立 HTTP server"

**启动阻塞点是改造优先级最高的依赖。**

### Step 5: 产出分析报告

#### 5.0 已有本地化资产（如有）

如果 Step 2.0 发现了已有的内存实现、mock 目录、本地配置文件等，
**在报告开头单独列出**，标注当前用途（test-only / 已在生产路径 / 未使用）
和升级建议（能否直接注入生产启动路径）。

这些已有资产可大幅缩短改造工作量估算。

#### 5.1 依赖清单

按 L0-L5 分层列出所有依赖，每个依赖包含：

| 字段 | 说明 |
|------|------|
| 依赖名 | 包路径或服务名 |
| 层级 | L0-L5 |
| 引用文件数 | 影响范围 |
| 关键引用点 | 文件路径:行号（最多列 3 个） |
| 是否阻塞启动 | 是/否 |
| 是否有 interface 抽象 | 是/否 |
| 本地替代方案 | 具体建议 |
| 预估工作量 | 时间估算 |

#### 5.2 依赖统计

```
总依赖数：XX
├─ L0 直接可用：XX (XX%)
├─ L1 配置替换：XX (XX%)
├─ L2 容器替代：XX (XX%)
├─ L3 Mock/Stub：XX (XX%)
├─ L4 适配层：XX (XX%)
└─ L5 暂不可本地化：XX (XX%)

启动阻塞点：X 个
预估总工作量：X 人天
```

#### 5.3 最小可启动子集

列出让服务能在本地 `go run` 起来的最小改动。
**目标是混合模式**：直连 BOE 能连的 + noop 不重要的 + 只本地化真正挡路的。

**第 1 步：环境变量（零代码改动，解决大部分 L1-L2 依赖）**

Consul 连 BOE 公共代理：

```bash
export CONSUL_HTTP_HOST=common-consul-boe.bytedance.net
export CONSUL_HTTP_PORT=2280
export RUNTIME_IDC_NAME=boe
```

服务标识：

```bash
export PSM=<服务PSM>
export SERVICE_ENV=boe
```

doas 包裹启动（注入 SEC_TOKEN_STRING）：

```bash
kinit <email_prefix>@BYTEDANCE.COM
doas -t zti -e boe -p <PSM> go run ./cmd/<entry>
```

如果仓库有 `conf/config.boe.*`，指定使用该配置文件。

**第 2 步：处理仍然 panic 的启动阻塞点（L4-L5）**

环境变量配好后尝试 `go run`，看哪些 init 仍然 panic。
对每个 panic 点给出具体建议：
- `MustInit` 类 → 加 `if isLocal() { return }` 开关
- 需要本地数据控制的 DB → docker-compose 起本地实例
- 有 interface 抽象的 → 复用已有 mock 或写最简 stub

**第 3 步：按需本地化数据层（L3）**

Agent 需要自由操控测试数据的存储，建议本地化：

docker-compose.local-twin.yml（按需裁剪）：

```yaml
services:
  mysql:
    image: mysql:8.0
    ports: ["3306:3306"]
    environment: { MYSQL_ROOT_PASSWORD: root123, MYSQL_DATABASE: local_dev }
  redis:
    image: redis:7-alpine
    ports: ["6379:6379"]
```

如果 Step 2.0 发现了已有的内存实现（miniredis/dolthub/sqlite），优先复用。

#### 5.4 改造路线图

按优先级排序：

| 优先级 | 目标 | 涉及依赖 | 方案 | 工作量 |
|--------|------|---------|------|--------|
| **P0** | `go run` 不 panic | 环境变量 + 仍 panic 的 `MustInit` 点 | 环境变量 + init 开关 | 预估 |
| **P1** | 数据层本地可控 | DB/缓存（Agent 需要自由读写测试数据） | 本地容器或复用已有内存实现 | 预估 |
| **P2** | 关键下游可 mock | 有副作用或不稳定的 RPC 下游 | interface stub | 预估 |
| **P3** | 本地观测闭环 | 日志/trace（可选） | 参见 `harness-obs-init` | 预估 |

#### 5.5 无需本地化的依赖

明确列出**不需要花时间处理**的依赖，避免分析报告过度膨胀：

| 类别 | 处理 | 示例 |
|------|------|------|
| 不阻塞启动的旁路 | 原样保留或 noop | metrics 上报、APM、埋点、Slardar、日志推送 |
| BOE 可直连的基础设施 | 配环境变量即可 | Consul、大部分 Redis/MySQL（通过 Consul 解析） |
| BOE 可直连的下游 RPC | 走 Consul 自动路由 | Overpass 生成的 client（连上 Consul 就能调通） |
| 线上专属机制 | 本地不存在 | PSM ACL 鉴权（BOE 大概率没有） |
| 代码生成工具 | 不在运行时 | `gorm_gen`、`handler_gen`、`kitex_gen` |

## 报告规则

- 每个依赖必须有具体文件路径作为证据，不能只说"用了 TCC"
- 启动阻塞点必须追踪到具体的 `init()` / `main()` 调用链
- **内部依赖数量 ≠ 启动阻塞点数量**——报告中要明确区分"import 引用数"和"实际阻塞启动的 init 点数"，避免夸大难度
- 工作量估算要区分"新建 interface 抽象"和"已有 interface / mock 只需复用"
- L5 不是垃圾桶——归入 L5 前必须确认确实没有合理的本地替代
- 已有本地化资产（内存实现、mock 目录、本地配置文件）必须在报告中单独列出
- 多 PSM monorepo 的改造路线图应说明共享 infra 的复用关系——改一处惠及多个入口

## 与本地观测栈结合

本地孪生 + 本地观测栈 = **Agent 自主调试闭环**：

```
Agent 改代码 → go run 秒级重启 → 触发请求 → 观测栈捕获日志+trace
→ Agent 通过 trace_id 查全链路日志 → 定位问题 → 改代码 → 重复
```

当 P0（`go run` 不 panic）完成后，建议立即用 `harness-obs-init` 接入本地观测栈：
- 结构化日志推送到 VictoriaLogs → Agent 可通过 `action:xxx AND result:failed` 快速定位错误
- OTel trace 导出到 VictoriaTraces → Agent 可通过 trace_id 查看完整调用链和耗时瓶颈
- HTTP 中间件回传 `X-Trace-Id` → Agent 从响应头即可拿到 trace 入口

**没有观测栈的本地孪生只解决了"能跑"，加上观测栈才解决"能调"。**

## 与其他 Harness 技能的衔接

| 技能 | 衔接方式 |
|------|---------|
| `harness-obs-init` | P0 完成后立即接入，实现"能跑"→"能调"的闭环 |
| `harness-exec-plan` | 为 P1-P2 改造创建执行计划 |
| `harness-analysis` | 本地化程度是"观测体系"和"测试体系"维度的关键前置条件 |
| `harness-hook-init` | 本地化后可建立 pre-commit 验证护栏 |
