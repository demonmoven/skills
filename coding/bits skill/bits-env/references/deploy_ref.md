# Deploy Reference — TCE 类型服务部署执行接口参考

> 本文件定义了 **TCE 类型服务**的所有写操作（创建与升级）的完整参数规范与输出格式。
> Plan Skill 在 **EXECUTE** 阶段对 `service-type=tce`（默认）的服务调用。
>
> **适用范围**: 仅适用于 TCE 类型服务。TCC 类型服务请参考 `tcc_ref.md`。

**CLI 命令**: `bitscli env`

---

## 命令总览

| # | 命令 | 用途 | 适用场景 |
|:--|:-----|:-----|:---------|
| 1 | `deploy-create` | 创建环境 / 新建服务 / 新建集群 | CREATE_NEW / DEPLOY_TO_ENV / CLONE / ADD_CLUSTER / BATCH_DEPLOY |
| 2 | `deploy-upgrade` | 更新现有集群的代码版本 | UPGRADE / ROLLBACK / BATCH_DEPLOY |

---

## 1. Deploy Create — 创建部署

### 适用场景

- 创建新环境（env 不存在时自动创建）
- 在现有环境中首次部署某服务
- 在现有服务下添加新集群
- 服务账号批量部署（BATCH_DEPLOY）

### Command

```bash
bitscli env deploy-create \
  --env <env> \
  --psm <psm> \
  --standard-env <standard_env> \
  --cluster-config <config> \
  [--env-type <type>] \
  [--branch <branch>] \
  [--scm-version <scm_version>] \
  [--cluster-name <name>] \
  [--instance-count <count>] \
  [--base-cluster-id <id>] \
  [--cpu-mem <cpu>:<mem>] \
  [--single-idc] \
  [--service-auth]
```

### Input Parameters

| Parameter | Flag | Required | Default | Description |
|:----------|:-----|:---------|:--------|:------------|
| env | `--env` | **YES** | — | 目标环境名称。不存在时自动创建。格式: `ppe_[a-z0-9_]+` 或 `boe_[a-z0-9_]+` |
| psm | `--psm` | **YES** | — | 服务 PSM。格式: 点分隔标识符 |
| standard_env | `--standard-env` | **YES** | — | 标准环境，**必须与 env 前缀匹配**：`ppe_` → `online_*`（默认 `online_cn`），`boe_` → `boe*`（默认 `boe`）。合法值: `online_cn` / `online_i18n` / `online_i18nbd` / `online_usttp` / `online_euttp` / `boe` / `boe_i18n_sg` / `boe_i18n_va` / `boe_usttp` |
| cluster_config | `--cluster-config` | **YES** | — | 集群配置，格式见下方说明 |
| env_type | `--env-type` | NO | 自动推断 | 环境类型。`ppe_` 前缀 → `ppe`，`boe_` 前缀 → `boe_feature` |
| branch | `--branch` | ⚠️ 见规则 | — | Git 分支名。仅用户在 Git 仓库部署当前分支时使用 |
| version | `--scm-version` | ⚠️ 见规则 | — | SCM 版本号。**优先使用此参数**，非本地 Git 场景必须使用 version |
| cluster_name | `--cluster-name` | NO | `default` | 用户定义的集群标识符。冲突时递增: `default` → `default2` → `default3`（无下划线）。与 logicalCluster 是完全独立的概念 |
| instance_count | `--instance-count` | NO | `1` | 实例数量（当 cluster-config 中已含 count 时可省略） |
| base_cluster_id | `--base-cluster-id` | NO | — | Prod 线上基准集群 ID。有值时开启流量 fallback。来自 recommend-cluster 返回值 |
| cpu-mem | `--cpu-mem` | NO | CLI 默认值 | 集群实例的 CPU 与内存规格，格式 `<cpu>:<mem>`。仅用户显式指定时携带 |
| single_idc | `--single-idc` | NO | `false` | 单机房隔离模式，作用于**环境级别**（非集群级别），由首次 deploy-create 创建环境时决定，后续对同一环境的操作不再传此参数。仅新建环境 + 单 TCE 服务时使用 |
| service_auth | `--service-auth` | NO | `false` | 服务账号模式开关。BATCH_DEPLOY 场景必须携带 |

### --cpu-mem 使用规则

**格式**: `--cpu-mem <cpu>:<mem>`，冒号分隔，左边 CPU 右边 Memory。

| 条件 | 行为 |
|------|------|
| 用户显式指定了 CPU 和 Memory | 追加 `--cpu-mem <cpu>:<mem>` |
| 用户未提及资源规格 | **不携带**，由 CLI 使用默认值 |

> **禁止**在用户未指定时自行填写 `--cpu-mem` 值。此参数完全由用户驱动。

**示例**:
- 用户说 "8 核 16G" → `--cpu-mem 8:16`
- 用户说 "4C8G" → `--cpu-mem 4:8`
- 用户未提及 → 不加此参数

### branch vs version 使用规则

| 场景 | 使用参数 | 原因 |
|------|---------|------|
| 用户在 Git 仓库中部署当前分支 | `--branch` | 明确意图部署本地开发分支 |
| **其他所有场景** | **`--scm-version`** | version 精确锁定构建产物，消除不确定性 |
| 两者都提供 | `--scm-version` | version 优先级更高 |

> **禁止**在非本地 Git 场景使用 `--branch master` 兜底。必须通过 recommend-version 流程获取精确 version。

### 同服务多集群说明

同一 PSM 下创建多个集群时，每个集群独立执行一次 deploy-create。
多个集群**可以**使用完全相同的 `--cluster-config`——它们通过 `--cluster-name` 区分，不会冲突。

示例：aweme.namek.merchant_api 需要 default 和 poi_task 两个集群，均在 HL/LF/LQ 三个机房：
- 第 1 次: `--cluster-name default --cluster-config "China-North-LF|PPE|ipv6|HL:1,LF:1,LQ:1"`
- 第 2 次: `--cluster-name poi_task --cluster-config "China-North-LF|PPE|ipv6|HL:1,LF:1,LQ:1"`

### cluster-config 格式说明

**格式**: `Zone|Physical|Logical|IDC1:count[,IDC2:count]`

- **分隔符**: `|`（竖线）
- **IDC 部分**: `机房代码:实例数`，多个机房用 `,` 分隔
- **来源**: 必须从 `recommend-cluster` 返回值拼接，禁止编造

#### 示例

| 场景 | cluster-config | 说明 |
|------|---------------|------|
| 单机房单实例 | `"China-North-LF|PPE|default|HL:1"` | 1 个集群，HL 机房 1 个实例 |
| 单集群多机房 | `"China-North-LF|PPE|ipv6|HL:1,LQ:1"` | 1 个集群，实例分布在 HL 和 LQ |
| 单机房多实例 | `"China-North-LF|PPE|default|LF:3"` | 1 个集群，LF 机房 3 个实例 |

#### 从 recommend-cluster 返回值转换

```json
{
  "zone": "China-North-LF",
  "physicalCluster": "PPE",
  "logicalCluster": "ipv6",
  "instanceList": [
    {"idc": "HL", "instanceCount": 1},
    {"idc": "LQ", "instanceCount": 1}
  ],
  "base_cluster_id": 558427
}
```

→ `--cluster-config "China-North-LF|PPE|ipv6|HL:1,LQ:1" --base-cluster-id 558427`

**转换规则**:
1. 前三段: `zone|physicalCluster|logicalCluster`（精确值，禁止修改）
2. 第四段: 遍历 `instanceList`，每项拼为 `idc:instanceCount`，用 `,` 连接
3. `base_cluster_id` 有值 → 追加 `--base-cluster-id`

### --service-auth 说明

| 条件 | 是否携带 |
|------|---------|
| 意图为 `BATCH_DEPLOY`（服务账号模式） | **必须**携带 `--service-auth` |
| 其他所有意图 | **不携带** |

> `--service-auth` 是无值 flag。CLI 检测到后从 `~/.local/share/bits-env/.bits_env_cli_token.json` 读取服务账号 token。
> 若 `~/.local/share/bits-env/.bits_env_cli_token.json` 不存在，CLI 报错终止，不回退个人账号。

### Output Format

```text
### Deployment Initiated
| Item | Value | Description |
| :--- | :--- | :--- |
| Env Name | {env_name} | 目标环境名称 |
| Service | {psm} | 服务 PSM |
| Action | Create | 操作类型 |
| Status | Success | 提交状态 |
| Ticket ID | {ticket_id} | 关联工单 ID |

### Cluster Configuration
| Cluster Name | Zone | Physical/Logical | IDC Distribution | Base Cluster | CPU:MEM |
| :--- | :--- | :--- | :--- | :--- | :--- |
| {name} | {zone} | {physical}/{logical} | {idc1}:{count}, {idc2}:{count} | {base_cluster_id 或 —} | {cpu}:{mem} 或 — |
```

### 完整命令示例

**基础创建（单机房）**:
```bash
bitscli env deploy-create \
  --env ppe_test --psm bits.env.api \
  --scm-version 1.0.1.810 \
  --cluster-config "China-North-LF|PPE|default|LF:1" \
  --standard-env online_cn
```

**指定 CPU 和 Memory**:
```bash
bitscli env deploy-create \
  --env ppe_skills_multi_idc --psm aweme.api.social_basic \
  --scm-version 1.0.0.3821 \
  --cluster-config "China-North-LF|PPE|ipv6|HL:1,LF:1,LQ:1" \
  --cluster-name ugc --base-cluster-id 300294 \
  --cpu-mem 8:16 \
  --standard-env online_cn --service-auth
```

**多机房实例 + 流量 fallback**:
```bash
bitscli env deploy-create \
  --env ppe_test --psm ecom.fulfillment.promise \
  --scm-version 1.0.1.810 \
  --cluster-config "China-North-LF|PPE|ipv6|HL:1,LQ:1" \
  --base-cluster-id 558427 \
  --standard-env online_cn
```

**服务账号批量部署**:
```bash
bitscli env deploy-create \
  --env ppe_batch --psm svc.a \
  --scm-version 1.0.1.810 \
  --cluster-config "China-North-LF|PPE|default|HL:1" \
  --standard-env online_cn \
  --service-auth
```

**本地 Git 分支部署（唯一使用 --branch 的场景）**:
```bash
bitscli env deploy-create \
  --env ppe_feat_x --psm my.service.psm \
  --branch feat/new-api \
  --cluster-config "China-North-LF|PPE|default|LF:1" \
  --standard-env online_cn
```

---

## 2. Deploy Upgrade — 升级部署

### 适用场景

- 更新现有集群的代码版本
- 回滚到指定版本
- 服务账号批量升级（BATCH_DEPLOY）

### Command

```bash
bitscli env deploy-upgrade \
  --env <env> \
  --psm <psm> \
  --standard-env <standard_env> \
  --cluster-id <cluster_id> \
  [--branch <branch>] \
  [--scm-version <scm_version>] \
  [--env-type <type>] \
  [--service-auth]
```

### Input Parameters

| Parameter | Flag | Required | Default | Description |
|:----------|:-----|:---------|:--------|:------------|
| env | `--env` | **YES** | — | 目标环境名称 |
| psm | `--psm` | **YES** | — | 服务 PSM |
| standard_env | `--standard-env` | **YES** | — | 标准环境 |
| cluster_id | `--cluster-id` | **YES** | — | 目标升级的集群 ID（从 `instance-detail` 获取） |
| branch | `--branch` | ⚠️ 见规则 | — | Git 分支名。仅本地 Git 场景使用 |
| version | `--scm-version` | ⚠️ 见规则 | — | SCM 版本号。**优先使用** |
| env_type | `--env-type` | NO | 自动推断 | 环境类型 |
| service_auth | `--service-auth` | NO | `false` | 服务账号模式开关。BATCH_DEPLOY 场景必须携带 |

> branch vs version 规则同 deploy-create。
> **注意**: `deploy-upgrade` **不支持** `--cpu-mem`。资源规格仅在创建时指定。

### Output Format

```text
### Upgrade Initiated
| Item | Value | Description |
| :--- | :--- | :--- |
| Env Name | {env_name} | 目标环境名称 |
| Service | {psm} | 服务 PSM |
| Target Cluster ID | {cluster_id} | 升级目标集群 |
| Action | Upgrade | 操作类型 |
| Status | Success | 提交状态 |
| Ticket ID | {ticket_id} | 关联工单 ID |
```

### 完整命令示例

**标准升级**:
```bash
bitscli env deploy-upgrade \
  --env ppe_stable --psm user.auth \
  --scm-version 1.0.2 \
  --cluster-id 12345 \
  --standard-env online_cn
```

**服务账号批量升级**:
```bash
bitscli env deploy-upgrade \
  --env ppe_batch --psm svc.a \
  --scm-version 1.0.2 \
  --cluster-id 12345 \
  --standard-env online_cn \
  --service-auth
```

**回滚**:
```bash
bitscli env deploy-upgrade \
  --env ppe_stable --psm user.auth \
  --scm-version 0.9.8 \
  --cluster-id 12345 \
  --standard-env online_cn
```

---

## Validation Checklist — EXECUTE 前必检清单

> 🚨 **#0 为最高优先级规则，必须在所有其他检查项之前执行。**

| # | 检查项 | create | upgrade | 违反动作 |
|:--|:-------|:------:|:-------:|:---------|
| 0 | **`env` 严禁为 `prod` 或任何不含 `ppe_`/`boe_` 前缀的值**。**例外**: `env=prod` + `env-type=boe_base` 时允许（BOE 基准环境部署，详见 `boe_base_deploy_ref.md`） | ✅ | ✅ | **ABORT: 立即终止，提示用户走正规发布流程；boe_base 豁免** |
| 1 | `env` 格式合法 (`ppe_[a-z0-9_]+` 或 `boe_[a-z0-9_]+`)。**例外**: `env=prod` + `env-type=boe_base` 时豁免 | ✅ | ✅ | ABORT |
| 2 | `psm` 格式合法（含 `.` 分隔） | ✅ | ✅ | ABORT |
| 3 | `standard-env` 已填写且为合法枚举值 | ✅ | ✅ | 补充默认值 |
| 4 | `version` 非空（非本地 Git 场景）；本地 Git 场景 `branch` 非空 | ✅ | ✅ | ABORT, 要求用户指定 |
| 5 | `cluster-config` 格式合法且各段为精确值 | ✅ | — | REJECT, 用 recommend-cluster 结果拼接 |
| 6 | `cluster-config` 中 IDC:count 与 recommend-cluster instanceList 一致 | ✅ | — | REJECT |
| 7 | `cluster-name` 在同 PSM 下唯一 | ✅ | — | → RESOLVE 递增 |
| 8 | `cluster-id` 存在于目标环境 | — | ✅ | REJECT, 重新 search |
| 9 | `base-cluster-id` 来自 recommend-cluster 返回值（有值时） | ✅ | — | REJECT, 禁止编造 |
| 10 | BATCH_DEPLOY 时携带 `--service-auth` | ✅ | ✅ | REJECT, 补充 flag |
| 11 | 非 BATCH_DEPLOY 时**不**携带 `--service-auth` | ✅ | ✅ | REJECT, 移除 flag |
| 12 | `--cpu-mem` 仅在用户显式指定时携带；未指定时**不携带** | ✅ | — | REJECT, 移除参数 |
| 13 | `standard-env` 与 `env` 前缀匹配：`ppe_` → `online_*`，`boe_` → `boe*`。不匹配则修正。**例外**: `env=prod` + `env-type=boe_base` 时 `standard-env=boe` 合法 | ✅ | ✅ | REJECT, 按 env 前缀重新推断 |