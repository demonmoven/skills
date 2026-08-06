# BOE Base Deploy Reference — BOE 基准环境部署参考

> 本文件定义了部署服务到 **BOE 基准环境（prod）** 的完整参数规范与执行流程。
> 当用户意图为向 BOE 的 prod 环境部署服务时，Agent **MUST** 参考本文件执行。
>
> **适用范围**: 仅适用于 BOE 基准环境部署（`--env prod --standard-env boe --env-type boe_base`）。
> 其他 PPE/BOE feature 环境部署请参考 `deploy_ref.md`。

**CLI 命令**: `bitscli env`

> 🚨 **CRITICAL — 服务账号强制要求**
>
> BOE 基准环境（prod）部署**必须使用服务账号**（`--service-auth`）。
> 缺少服务账号时**立即 ABORT**，提示用户提供 `secret`。
> **此规则仅针对 BOE 基准环境（`env=prod, env-type=boe_base`）**，普通 BOE feature 环境（`boe_xxx`）不受此限制。

---

## 核心概念

BOE 基准环境是 BOE 的 **prod 环境**，用于部署服务的稳定基准版本。与 PPE/BOE feature 环境的关键区别：

| 维度 | PPE / BOE Feature | BOE 基准环境 |
|:-----|:-------------------|:-------------|
| `--env` | `ppe_xxx` / `boe_xxx` | `prod` |
| `--env-type` | `ppe` / `boe_feature`（自动推断） | `boe_base`（**必须显式指定**） |
| `--standard-env` | `online_cn` / `boe` 等 | `boe` |
| 集群推荐 | 单次 recommend-cluster 即可 | **需要两次调用**（BOE + PPE），由 Skill 编排 |
| 版本来源 | recommend-version 流程 / 本地 Git 分支 | **线上 prod 版本**（`env_type=prod` 的 SCM 记录） |
| 服务账号 | 可选（仅 BATCH_DEPLOY 必需） | **强制必须**（缺失 → ABORT） |
| Validation SV-00 | `env=prod` → ABORT | `env=prod` + `env-type=boe_base` → **允许** |

---

## 触发条件

当以下任一条件满足时，识别为 BOE 基准环境部署意图：

| 用户表达模式 | 识别结果 |
|:------------|:---------|
| "部署到 boe 的 prod 环境" | BOE_BASE_DEPLOY |
| "部署到 boe 基准环境" | BOE_BASE_DEPLOY |
| "在 boe prod 上部署 xxx" | BOE_BASE_DEPLOY |
| "向 boe 基准环境添加服务" | BOE_BASE_DEPLOY |

**识别后固定参数**：
- `--env prod`
- `--standard-env boe`
- `--env-type boe_base`
- `--service-auth`（**强制**，必须使用服务账号）

---

## 执行流程

### Step 0: 服务账号认证（INIT 阶段）

> 🚨 BOE 基准环境部署**强制使用服务账号**。用户未提供 `secret` 时**必须反问**，禁止跳过。

1. 若用户未提供 `secret`，**立即反问**："请提供服务账号的密钥（secret），BOE 基准环境部署必须使用服务账号。"
2. 调用 `bitscli env service-auth --secret <secret>`
3. 确认输出包含"认证成功" → 继续；认证失败 → **ABORT** 并报告错误

### Step 1: 集群推荐（RESOLVE 阶段）

BOE Base 需要**两次** `recommend-cluster` 调用：

#### Step 1a: 获取 BOE Base 集群推荐

```bash
bitscli env recommend-cluster \
  --psm <psm> \
  --standard-env boe \
  --env-type boe_base
```

返回示例：
```json
{
  "env_type": "boe_base",
  "standard_env": "boe",
  "clusters": [
    {
      "zone": "BOE-Arbutus",
      "physicalCluster": "Arbutus",
      "logicalCluster": "ipv6",
      "instanceList": [{ "idc": "arbutus", "instanceCount": 1 }]
    },
    {
      "zone": "boe-i18n",
      "physicalCluster": "SomeCluster",
      "logicalCluster": "default",
      "instanceList": [{ "idc": "sg", "instanceCount": 1 }]
    }
  ],
  "backups": [
    {
      "zone": "BOE-Arbutus",
      "physicalCluster": "Arbutus2",
      "logicalCluster": "default",
      "instanceList": [{ "idc": "BOE", "instanceCount": 1 }]
    }
  ],
  "base_cluster_id": 41901,
  "cpu_suggest": 1,
  "mem_suggest": 1
}
```

#### Step 1b: 获取 PPE 参考数据

```bash
bitscli env recommend-cluster \
  --psm <psm> \
  --standard-env online_cn \
  --env-type ppe
```

返回示例：
```json
{
  "env_type": "ppe",
  "standard_env": "online_cn",
  "clusters": [ ... ],
  "backups": [ ... ],
  "base_cluster_id": 8192815,
  "cpu_suggest": 4,
  "mem_suggest": 8
}
```

此调用的目的是获取 PPE 的 `base_cluster_id` 和 `cpu_suggest` / `mem_suggest` 作为参考。

#### Step 1c: 组合结果

按以下规则从两次调用的返回值中提取最终参数：

##### 集群选择

从 **Step 1a**（BOE）的返回值中选取集群，按优先级：

1. 从 `clusters` 中筛选（见下方 Zone 筛选规则）
2. `clusters` 筛选后为空 → 从 `backups` 中筛选
3. 均为空 → 报错，终止流程

**Zone 筛选规则**：

| Zone | 处理方式 | 原因 |
|:-----|:---------|:-----|
| `BOE-Arbutus` | ✅ **优先选择** | BOE 主力可用区 |
| `boe-i18n` | ❌ **排除** | 国际化专用，不适合 BOE Base |
| `huabei2` | ❌ **排除** | 旧区域，不推荐 |
| 其他 zone | ✅ 可选，但优先级低于 `BOE-Arbutus` | — |

**筛选后排序**: `BOE-Arbutus` 排在最前，其他 zone 在后。选取排序后的第一个集群。

##### base_cluster_id

| 条件 | 取值 |
|------|------|
| Step 1b（PPE）的 `base_cluster_id > 0` | 使用 PPE 的值 |
| PPE 的为 0，Step 1a（BOE）的 `base_cluster_id > 0` | 使用 BOE 的值 |
| 两者均为 0 | 不追加 `--base-cluster-id` |

> PPE 的 `base_cluster_id` 优先，因为它对应线上 prod 集群，流量 fallback 更准确。

##### cpu-mem 规格

| 条件 | 取值策略 |
|------|---------|
| 用户显式指定了规格 | 使用用户指定的值 |
| Step 1b（PPE）有 `cpu_suggest` / `mem_suggest` | PPE 值**减半**（最小为 1） |
| PPE 无值，Step 1a（BOE）有值 | 直接使用 BOE 的值 |

**减半公式**: `cpu = max(1, PPE_cpu / 2)`, `mem = max(1, PPE_mem / 2)`

> 减半原因：BOE 环境的流量和负载通常是 PPE 的一半，无需全量资源。

---

### Step 2: 版本获取（RESOLVE 阶段）

BOE 基准环境使用 **线上 prod 版本**，获取方式（`recommend-version` 是决策流程，非 CLI 命令，详见 `recommend_version_ref.md`）：

1. 调用 `bitscli env scm-repo --psm <psm>` 获取仓库信息
2. 从输出的 `Env SCM Dependencies` 表中选取 `env_type=prod` 对应的版本记录（Version 字段）
3. 若 prod 行的 Version 为空，调用 `bitscli env scm-latest-version --repo-id <repo_id> --branch <prod_branch>` 获取最新构建版本
4. **禁止**使用 `boe_base` 或 `boe_feature` 的版本

> **关键区别**: 普通 BOE feature 环境可能使用 BOE 版本，但 BOE 基准环境**必须**使用 prod（线上）版本。

---

### Step 3: 部署执行（EXECUTE 阶段）

```bash
bitscli env deploy-create \
  --env prod \
  --psm <psm> \
  --scm-version <prod_version> \
  --cluster-config "<Zone>|<Physical>|<Logical>|<IDC>:<count>" \
  --standard-env boe \
  --base-cluster-id <base_cluster_id> \
  --service-auth
```

其中：
- `cluster-config` 从 Step 1c 选定的集群构造（格式见 `recommend_cluster_ref.md` 规则 2）
- `base-cluster-id` 从 Step 1c 确定的值
- `--service-auth` **必须携带**（BOE 基准环境强制要求）
- 如果 Step 1c 确定了 cpu-mem，追加 `--cpu-mem <cpu>:<mem>`

#### 完整示例

```bash
# Step 0: 服务账号认证（必须）
bitscli env service-auth --secret <secret>

# Step 1a: BOE Base 推荐
bitscli env recommend-cluster --psm chenmiao.ppedebug.euler --standard-env boe --env-type boe_base

# Step 1b: PPE 参考数据
bitscli env recommend-cluster --psm chenmiao.ppedebug.euler --standard-env online_cn --env-type ppe

# Step 3: 部署（假设 BOE 返回 BOE-Arbutus 集群，PPE 返回 base_cluster_id=8192815，cpu=4/mem=8 → 减半为 2:4）
bitscli env deploy-create \
  --env prod \
  --psm chenmiao.ppedebug.euler \
  --scm-version 1.0.0.1 \
  --cluster-config "BOE-Arbutus|Arbutus|ipv6|arbutus:1" \
  --standard-env boe \
  --base-cluster-id 8192815 \
  --cpu-mem 2:4 \
  --service-auth

# Step 4: 清理认证
bitscli env service-auth --clear
```

---

## Validation 特殊规则

BOE 基准环境部署对标准校验规则有以下例外：

| 标准规则 | BOE 基准环境行为 | 说明 |
|:---------|:----------------|:-----|
| SV-00: `env=prod` → ABORT | **豁免**: `env=prod` + `env-type=boe_base` → 允许 | BOE 基准环境的 env 名称固定为 `prod` |
| SV-15: `env` 前缀与 `standard-env` 匹配 | **豁免**: `env=prod` 无 `ppe_`/`boe_` 前缀但 `standard-env=boe` 合法 | 特殊场景 |
| env 格式: `^(ppe\|boe)_[a-z0-9_]+$` | **豁免**: `prod` 不符合此正则但合法 | 特殊场景 |
| env-type 自动推断 | **禁止自动推断**: 必须显式传入 `--env-type boe_base` | `DefaultEnvType("boe")` 返回 `boe_feature`，不适用于此场景 |
| `--service-auth` 可选 | **强制必须**: BOE 基准环境部署的所有 deploy-create 命令**必须**携带 `--service-auth` | 缺失 → **ABORT**，反问用户提供 secret |

> **⚠️ CRITICAL**: 上述豁免**仅在 `--env-type boe_base` 显式传入时生效**。
> 缺少 `--env-type boe_base` 时，`--env prod` 仍然触发 SV-00 ABORT。

---

## 与现有流程的差异汇总

| FSM 阶段 | 普通部署 | BOE 基准环境部署 |
|:---------|:--------|:---------------------|
| INIT | 自动推断 env-type | **必须**设定 `env-type=boe_base`；**必须**收集 `secret`（服务账号） |
| SEARCH | env-search 查环境 | 可跳过（prod 环境固定存在） |
| RESOLVE (cluster) | 1 次 recommend-cluster | **2 次**：BOE(`boe_base`) + PPE(`ppe`)，Skill 编排组合 |
| RESOLVE (version) | recommend-version 流程 | 使用 prod 版本（`env_type=prod` 的 SCM 记录） |
| VALIDATE | SV-00 拦截 `env=prod` | SV-00 豁免（需 `env-type=boe_base`）；SV-16 强制 `--service-auth` |
| EXECUTE | `deploy-create` 标准参数 | 追加 `--cpu-mem`（PPE 减半）+ `--base-cluster-id`（PPE 优先）+ `--service-auth`（**强制**） |
| MONITOR | 标准轮询 | 标准轮询（无差异） |
| CLEANUP | 仅 BATCH_DEPLOY 触发 | **始终触发** `service-auth --clear` |
