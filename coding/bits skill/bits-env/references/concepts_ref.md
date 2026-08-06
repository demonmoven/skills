# Concepts Reference — 核心概念与术语定义

> ⚠️ **必读文件**: 执行任何操作前，LLM **必须**先读本文件，建立统一的术语认知，避免概念混淆。

---

## 1. 环境（Env / 泳道 / Swimlane）

- **env** 和 **泳道（swimlane）** 是同一概念，可互换使用
- 命名规范：只能小写字母、数字、下划线
  - `ppe_` 前缀 → PPE 测试环境
  - `boe_` 前缀 → BOE 测试环境
- 示例：`ppe_xiaoxi6`、`boe_test01`

> 🚨 **严禁操作生产环境**
> `prod`以及任何不含 `ppe_` / `boe_` 前缀的 env 值均为**生产环境或无效环境**，AI Agent **严禁**对其执行任何部署操作。检测到此类 env 时必须立即 ABORT。

---

## 2. 服务（Service / PSM）

- **PSM**（Product.Service.Module）是服务的唯一标识符
- 格式：点分隔，如 `user.auth.service`、`ecom.order.api`
- 一个 env 可以包含多个 PSM（服务）

---

## 3. 服务类型：TCE vs TCC

| 类型 | 特征 | 部署命令 |
|------|------|---------|
| **TCE**（默认） | 通用容器服务，需要 cluster-config、version/branch | `deploy-create` / `deploy-upgrade` |
| **TCC** | 配置中心服务，只需 env + psm，无集群/版本概念 | `tcc-deploy-create` |

- 默认为 **TCE**，仅当用户明确说明时设为 TCC
- **禁止**主动询问服务类型

---

## 4. 集群（Cluster）vs 实例（Instance）vs 机房（IDC）

> **核心区分：机房（IDC）是实例层面的概念，不是集群层面的概念。**

```
集群 (Cluster)
  └─ 由 Zone / Physical / Logical 三段唯一标识
  └─ 一个集群可以包含分布在不同机房的多个实例
       ├─ 实例 @ IDC=HL (1个)
       ├─ 实例 @ IDC=LQ (1个)
       └─ 实例 @ IDC=LF (2个)
```

### 用户表达 → 语义解析

| 用户表达 | 语义 | 操作 |
|---------|------|------|
| "部署**两个集群**，一个在 LF，一个在 LQ" | 2 个集群，各含 1 个机房实例 | 2 次 recommend-cluster + 2 次 deploy-create |
| "部署 svc.a 的 default 和 poi_task 集群，都在 LF LQ HL" | 同服务 2 个集群，相同机房 | 2 次 recommend-cluster + 2 次 deploy-create |
| "部署**一个集群**，实例分布在 LF 和 LQ" | 1 个集群，含 2 个机房实例 | 1 次 recommend-cluster（`--specify-dcs LF,LQ`）+ 1 次 deploy-create |
| "部署到 HL 和 LQ"（未明确集群数） | 默认：1 个集群，多机房实例 | 1 次 recommend-cluster（`--specify-dcs HL,LQ`） |

**无法判断时** → 必须反问："您是需要创建多个独立集群，还是一个集群内包含多机房实例？"

**判断规则**：
- 用户为同一服务指定了**多个集群名称**（如 default、poi_task）→ 每个集群独立调用一次 recommend-cluster + 一次 deploy-create，无论机房是否相同
- 用户提到"多个/两个/N个 **集群**" + 不同机房 → 多次调用，每次指定一个机房
- 用户仅说"多个 **机房/IDC/实例**" 或 仅列举机房名 → 一次调用，`--specify-dcs X,Y`

### recommend-cluster 调用次数规则

> 每次调用返回**恰好一个**集群配置。**N 个集群 = N 次调用**，即使机房分布完全相同。

| 场景 | 调用次数 |
|------|---------|
| 1 个服务 1 个集群 | 1 次 |
| 1 个服务 N 个集群（不论机房是否相同） | N 次 |
| M 个服务各 1 个集群 | M 次（可并行） |
| M 个服务，其中某服务有 K 个集群 | M + (K-1) 次 |

### cluster-name vs logicalCluster — 完全不同的两个概念

> ⚠️ 混淆这两个概念会导致死循环。

| 概念 | 来源 | 示例值 | 作用 |
|------|------|--------|------|
| `cluster-name`（`--cluster-name`） | 用户定义 | `default`、`poi_task`、`ugc` | 区分同一服务下的不同集群 |
| `logicalCluster` | recommend-cluster 返回值 | `ipv6`、`default`、`amd` | 基础设施层面的逻辑集群类型 |

- 多个 `cluster-name` **可以**使用相同的 `logicalCluster`，不冲突
- `logicalCluster=default` 与 `--cluster-name default` **同名不同义**

---

## 5. standard-env（标准环境 / 控制面）

> **必须在 INIT 阶段确定，整个流程使用同一个值。**

> **⚠️ CRITICAL — env 前缀与 standard-env 必须匹配：**
> - `ppe_` 环境 → **只能**用 `online_*`（默认 `online_cn`）
> - `boe_` 环境 → **只能**用 `boe*`（默认 `boe`）
> - **违反此规则会导致 `recommend-cluster` 返回错误控制面的 `base_cluster_id`，TCE 创建集群时报 `get cluster info fail`。**
> - **推断逻辑**：读取 env 前缀自动确定，`ppe_` → `online_cn`，`boe_` → `boe`。

**PPE 控制面（`ppe_` 环境专用）**：

| standard-env | 隔离区 | 适用场景 |
|:-------------|:------|:---------|
| `online_cn` | CN | 国内线上，**默认值** |
| `online_i18n` | I18N | 海外国际化（TikTok 等） |
| `online_usttp` | US-TTP | 美国 TTP |
| `online_euttp` | EU-TTP | 欧洲 TTP |
| `online_i18nbd` | I18N-BD | 海外 BD 业务线 |

**BOE 控制面（`boe_` 环境专用）**：

| standard-env | 隔离区 | 适用场景 |
|:-------------|:------|:---------|
| `boe` | CN,I18N | **BOE 默认值** |
| `boe_i18n_sg` | I18N | 海外 SG |
| `boe_i18n_va` | I18N | 海外 VA |
| `boe_usttp` | US-TTP | 美国 TTP |

---

## 6. single-idc（单机房隔离）

> **`--single-idc` 是环境级别属性**，由首次 `deploy-create`（即创建新环境时）传入决定，**不是集群级别属性**。
> 一旦环境创建完成，该属性固定，后续向同一环境追加服务或集群时**不再传此参数**。
> **仅在新建环境时推断；环境已存在时永远不加。TCC 服务不计入服务数量统计。**

| 条件 | single-idc |
|:-----|:-----------|
| 用户显式指定 | 直接使用 |
| 新建环境 + 只部署 **1 个 TCE 服务** | ✅ 加 `--single-idc` |
| 新建环境 + 部署 **≥ 2 个 TCE 服务** | ❌ 不加 |
| 环境已存在 | ❌ 不加 |
| 仅部署 TCC 服务（无 TCE） | ❌ 不加 |
