# TCC Deploy Reference — TCC 服务部署与配置操作接口参考

> **本文件定义了 TCC 类型服务的所有操作命令、参数格式与使用方式。**
> CLI 工具: `bitscli env`
> **所有 TCC 操作依赖用户权限、都需要用户手工一步步执行+确认，无自动化流水线。**

---

## 0. 核心概念与全局规则

### 0.0 🚨 高危操作强制确认规则（MANDATORY CONFIRMATION）

> **任何以下操作，执行前都必须向用户展示完整操作摘要，并等待用户明确确认（yes/确认/继续），禁止自动执行。**

| 高危级别 | 命令 | 原因 |
|:--|:--|:--|
| 🔴 **极高危** | `tcc-create-boe-prod-namespace` | 在生产级控制面（BOE prod）创建 namespace，影响所有依赖该 namespace 的 BOE 泳道 |
| 🔴 **极高危** | `tcc-sync-configs`（目标含 BOE prod） | 覆盖 BOE prod 配置内容，影响范围广 |
| 🔴 **极高危** | `tcc-batch-deploy`（目标含 BOE prod） | 批量发布 BOE prod 配置，直接影响 BOE 生产级环境 |
| 🟠 **高危** | `tcc-sync-configs`（目标为泳道） | 覆盖目标泳道已有配置内容 |
| 🟠 **高危** | `tcc-batch-deploy`（目标为泳道） | 批量发布配置到泳道环境 |
| 🟠 **高危** | `tcc-deploy-config` | 发布单个配置到目标环境 |
| 🟠 **高危** | `tcc-operate-deployment`（operation=finish） | 推进发布单到下一阶段或完成发布 |
| 🟠 **高危** | `tcc-approve-deployment` | 审批通过发布单，触发配置生效 |

**强制确认流程**：

1. 在执行任何上述命令之前，**必须**先向用户展示以下摘要：
   - 操作类型（创建 namespace / 同步配置 / 发布配置 / 推进发布单）
   - 目标 namespace
   - 目标环境 + 目标站点
   - 影响的配置数量（如已知）
   - 对于 BOE prod：明确标注"此操作将影响 BOE 生产级控制面"
2. **等待用户明确回复确认**（如"是"、"确认"、"继续"、"yes"）后才能执行
3. **禁止**将确认步骤与执行步骤合并，不得在展示计划后自动倒计时或隐式默认执行

---

### 0.1 操作分类总览

TCC 操作分为 **三大类**，均无固定工作流，由用户按需组合：

| 分类 | 说明 | 涉及命令 |
|:--|:--|:--|
| **环境部署** | 在目标环境中创建 TCC 服务或 Namespace（所有配置操作的前提） | `tcc-deploy-create`, `tcc-create-boe-prod-namespace` |
| **单个操作** | 对单个配置项进行查询/创建/更新/发布等操作 | `tcc-list-sites`, `tcc-search-namespace`, `tcc-list-envs`, `tcc-list-dir`, `tcc-list-config`, `tcc-get-config`, `tcc-create-config`, `tcc-update-config`, `tcc-deploy-config`, `tcc-publish-detail`, `tcc-operate-deployment`, `tcc-approve-deployment`, `tcc-reject-deployment` |
| **批量操作** | 跨环境批量同步与发布配置 | `tcc-sync-configs`, `tcc-batch-deploy` |

**环境部署命令区分**:

| 命令 | 适用场景 | 前置条件 |
|:--|:--|:--|
| `tcc-deploy-create` | 在 PPE/BOE 环境中创建 TCC 服务实例 | — |
| `tcc-create-boe-prod-namespace` | 在 BOE prod 创建 Namespace（从 prod site 继承 node_id、owners、operators、viewers） | 仅当 BOE prod 中该 namespace **不存在**时执行；已存在则跳过 |

### 0.2 前置依赖：TCC Deploy Create（所有配置操作的前提）

> **关键规则**：所有 TCC 配置操作（无论单个还是批量）都**依赖目标环境中已存在 TCC 服务**。
> 如果目标环境中尚未创建 TCC 服务，必须**先执行 `tcc-deploy-create`**（非 BOE prod 环境）。
> BOE prod 环境的前置准备请使用 `tcc-create-boe-prod-namespace`，**不自动执行**，需提示用户手动确认后逐步操作。

**依赖检查方法**：

TCC 服务部署有**两层前置依赖**，需按顺序检查：

> **第 1 层：TCC 控制面 prod 上 namespace 是否存在**
> PPE 泳道依赖 CN prod 上的 namespace；BOE 泳道依赖 BOE prod 上的 namespace。

```bash
# 检查 CN prod（PPE 泳道前提）
bitscli env tcc-search-namespace --keyword <namespace>

# 检查 BOE prod（BOE 泳道前提）
bitscli env tcc-search-namespace --keyword <namespace> --tcc-site BOE
```
- 返回匹配结果 → namespace 存在，可进入第 2 层
- 返回空 → namespace 不存在。BOE prod 需先执行 `tcc-create-boe-prod-namespace`

> **第 2 层：TCC 服务是否已部署到目标泳道**
> ⚠️ 如果目标是 BOE prod（boe_base），则**跳过此检查**，直接提示用户进行进行配置同步/发布操作。

```bash
# 查询该 namespace 部署在哪些泳道中（仅非 boe_base 场景需要）
bitscli env tcc-list-envs --namespace <namespace>
```
- 返回结果包含目标泳道 → 已部署，可直接执行配置操作
- 返回空或不包含目标泳道 → 未部署，需先执行 `tcc-deploy-create`

### 0.3 namespace 与 PSM 的关系

> TCC 平台操作命令中的 `--namespace` **等价于** `psm`，自动令 `namespace = psm`，**无需额外询问用户**。

### 0.4 TCC 与 TCE 的核心差异

- TCC 服务**无** `tcc-deploy-upgrade` 命令。配置更新通过 TCC 平台命令完成，不走环境部署流程。
- TCC 部署**不需要** `--namespace`、`--cluster-config`、`--scm-version`/`--branch`、`--cpu-mem`、`--single-idc`、`--base-cluster-id` 等 TCE 专有参数。
- TCE 类型服务的部署命令请参考 `deploy_ref.md`。

---

## 1. TCC Deploy Create — 在环境中创建 TCC 服务

> **定位**：这是所有 TCC 操作的**入口命令**，属于 FSM EXECUTE 阶段。
> 所有后续的 TCC 配置操作都**依赖此命令已成功执行**。

### 适用场景
- 在环境中创建新的 TCC 类型服务（CREATE_NEW / DEPLOY_TO_ENV / CLONE）
- 批量部署 TCC 服务（BATCH_DEPLOY）

### Command

```bash
bitscli env tcc-deploy-create \
  --env <env> \
  --psm <psm> \
  [--service-id <id>] \
  [--region <region>] \
  [--standard-env <standard_env>] \
  [--env-type <env_type>] \
  [--service-auth]
```

### Input Parameters

| Parameter | Flag | Required | Default | Description |
|:--|:--|:--:|:--|:--|
| env | `--env` | **YES** | — | 目标环境名称。不存在时自动创建。格式: `ppe_[a-z0-9_]+` 或 `boe_[a-z0-9_]+` |
| psm | `--psm` | **YES** | — | 服务 PSM。格式: 点分隔标识符 |
| service_id | `--service-id` | NO | 自动解析 | TCC service_id，不传则通过 psm 搜索接口解析 |
| region | `--region` | NO | 自动推断 | region，不传则通过 conf_space 解析并优先选择 `CN` |
| standard_env | `--standard-env` | NO | 自动推断 | `online_cn`（ppe）/ `boe`（boe） |
| env_type | `--env-type` | NO | 自动推断 | 环境类型。`ppe_` 前缀 → `ppe`，`boe_` 前缀 → `boe_feature` |
| service_auth | `--service-auth` | NO | `false` | 服务账号模式开关。**BATCH_DEPLOY 场景必须携带** |

### Output Format

```text
### TCC Deployment Initiated
| Item | Value | Description |
| :--- | :--- | :--- |
| Env Name | {env_name} | 目标环境名称 |
| Service | {psm} | 服务 PSM |
| Service Type | TCC | 服务类型 |
| Namespace | {namespace} | TCC 命名空间 |
| Action | Create | 操作类型 |
| Status | Success | 提交状态 |
| Ticket ID | {ticket_id} | 关联工单 ID |
```

### 命令示例

```bash
# 基础创建
bitscli env tcc-deploy-create --env ppe_test --psm config.center.service --standard-env online_cn

# 服务账号批量部署（BATCH_DEPLOY 场景必须 --service-auth）
bitscli env tcc-deploy-create --env ppe_batch --psm config.center.service --standard-env online_cn --service-auth
```

### Validation Checklist

| # | 检查项 | 违反动作 |
|:--|:--|:--|
| 1 | `env` 格式合法 (`ppe_[a-z0-9_]+` 或 `boe_[a-z0-9_]+`) | ABORT |
| 2 | `psm` 格式合法（含 `.` 分隔） | ABORT |
| 3 | **禁止**携带 `--namespace` 参数 | REJECT: 移除参数 |
| 4 | `standard-env` 已填写且为合法枚举值 | 补充默认值 |
| 5 | BATCH_DEPLOY 时必须携带 `--service-auth` | REJECT: 补充 flag |
| 6 | **禁止**携带 TCE 专有参数（`--cluster-config`、`--scm-version`、`--branch` 等） | REJECT: 移除 |

---

## 2. 单个操作 — TCC 平台配置命令（按需组合）

> **使用前提**：目标环境中 TCC 服务**已通过 `tcc-deploy-create` 创建**。
> 如未创建，请先回到第 1 节执行创建。
>
> 以下命令已集成到 `bitscli env` 中，LLM 可直接调用。
> 这些命令**不纳入 FSM 流程**，按用户实际需求自由组合使用。

### 2.0 公共参数

以下参数在多个单个操作命令中复用：

| 参数 | Flag | 通用默认值 | Description |
|:--|:--|:--|:--|
| env | `--env` | 按命令不同 | TCC 环境，如 `prod` / `ppe` / `ppe_*` / `boe` / `boe_*` |
| standard_env | `--standard-env` | 按命令不同 | 集群环境，如 `boe` / `online_cn` |
| region | `--region` | `CN` | 区域，如 `CN`、`China-East` |
| namespace | `--namespace` | — | 命名空间（**= PSM**，无需额外询问） |
| dir | `--dir` | `/default` | 目录路径 |
| dir_id | `--dir-id` | 自动解析 | 目录 ID。自动解析失败时需通过 `tcc-list-dir` 获取后手动提供 |
| tcc_site | `--tcc-site` | 自动推断 | TCC 站点 |

### 2.1 查询类命令

#### 2.1.1 tcc-list-sites — 列出站点

> **意图触发词**: 查看站点、列出站点、TCC 有哪些站点

```bash
bitscli env tcc-list-sites
```

**输出**: JSON 数组，每项含 `key / origin / apiRoot / jwtHost / needTenantHeader`。

---

#### 2.1.2 tcc-search-namespace — 搜索命名空间

> **意图触发词**: 搜索 namespace、查找命名空间、namespace 是否存在

```bash
bitscli env tcc-search-namespace \
  --keyword <keyword> \
  [--env <env>] \
  [--tcc-site <site>]
```

| Parameter | Flag | Required | Default | Description |
|:--|:--|:--:|:--|:--|
| keyword | `--keyword` | YES | — | 搜索关键字 |
| env | `--env` | NO | `prod` | web API 环境 |
| tcc_site | `--tcc-site` | NO | 自动推断 | 站点 |

---

#### 2.1.3 tcc-list-envs — 查询 Namespace 部署的环境列表

> **意图触发词**: TCC 服务部署在哪些环境、查看 TCC 泳道、namespace 环境列表

查询某个 TCC namespace 已部署到哪些泳道（通过 TCC Platform OpenAPI）。

```bash
bitscli env tcc-list-envs --namespace <namespace>
```

| Parameter | Flag | Required | Default | Description |
|:--|:--|:--:|:--|:--|
| namespace | `--namespace` | YES | — | 命名空间（= PSM） |

**输出关键字段**:
- `count` → 部署的环境数量
- `items[].env` → 环境名称（如 `ppe_cm_test_2`）
- `items[].region` → 区域（如 `CN`）
- `items[].status` → 状态（如 `normal`）

---

#### 2.1.4 tcc-list-dir — 列出目录（获取 dir-id）

> **意图触发词**: 列出目录、查看目录、获取 dir-id

```bash
bitscli env tcc-list-dir \
  --namespace <namespace> \
  [--env <env>] \
  [--tcc-site <site>] \
  [--region <region>] \
  [--no-return-empty]
```

| Parameter | Flag | Required | Default | Description |
|:--|:--|:--:|:--|:--|
| namespace | `--namespace` | YES | — | 命名空间（= PSM） |
| env | `--env` | NO | `prod` | web API 环境 |
| tcc_site | `--tcc-site` | NO | 自动推断 | 站点 |
| region | `--region` | NO | — | 可选区域过滤 |
| no_return_empty | `--no-return-empty` | NO | `false` | 不返回空目录 |

**输出关键字段**:
- `dirs[].id` → 目录 ID（用于 `--dir-id`）
- `dirs[].path` → 目录路径（如 `/default`）
- `dirs[].description` → 目录描述
- `dirs[].owners` → owner 列表

---

#### 2.1.5 tcc-list-config — 列出配置列表

> **意图触发词**: 列出配置、查看配置列表、有哪些配置

```bash
bitscli env tcc-list-config \
  --namespace <namespace> \
  [--region <region>] \
  [--env <env>] \
  [--tcc-site <site>] \
  [--service-auth]
```

> **注意**: BOE 站点读 `CN` 时可能映射为 `all_region`（兼容 TCC 读取规则）。

---

#### 2.1.6 tcc-get-config — 查询单个配置详情

> **意图触发词**: 查看配置详情、获取配置内容、查询某个配置

```bash
bitscli env tcc-get-config \
  --namespace <namespace> \
  --config-name <config_name> \
  [--region <region>] \
  [--dir <dir>] \
  [--env <env>] \
  [--tcc-site <site>] \
  [--service-auth]
```

---

#### 2.1.7 tcc-publish-detail — 查询发布单详情

> **意图触发词**: 查看发布进度、发布单详情、发布状态

```bash
bitscli env tcc-publish-detail \
  --deployment-ref <id_or_publish_details_url> \
  [--env <env>] \
  [--tcc-site <site>]
```

**deployment-ref 支持两种格式**:
- 数字 ID：`2829686318612368`
- URL：`https://cloud.bytedance.net/tcc/namespace/<ns>/publish-details/<id>`

---

### 2.2 写入类命令

#### 2.2.1 tcc-create-config — 创建配置

> **意图触发词**: 创建配置、新建配置、添加配置项

```bash
bitscli env tcc-create-config \
  --namespace <namespace> \
  --config-name <config_name> \
  --description <desc> \
  [--env <env>] \
  [--tcc-site <site>] \
  [--region <region>] \
  [--dir <dir>] \
  [--dir-id <id>] \
  [--data-type <yaml|json|string>] \
  [--config-type <static|...>] \
  (--value <value> | --file <path>) \
  [--tags <a,b>] \
  [--note <note>]
```

| Parameter | Flag | Required | Default | Description |
|:--|:--|:--:|:--|:--|
| namespace | `--namespace` | YES | — | 命名空间（= PSM） |
| config_name | `--config-name` | YES | — | 配置名 |
| description | `--description` | YES | — | 配置描述（TCC Web 创建要求非空） |
| env | `--env` | NO | `ppe` | 目标环境 |
| tcc_site | `--tcc-site` | NO | 自动推断 | 站点 |
| region | `--region` | NO | `CN` | 区域 |
| dir | `--dir` | NO | `/default` | 目录路径 |
| dir_id | `--dir-id` | NO | 自动解析 | 目录 ID。若目录为空且解析失败，需手动提供 |
| data_type | `--data-type` | NO | `yaml` | 数据类型 |
| config_type | `--config-type` | NO | `static` | 配置类型 |
| value | `--value` | 二选一 | — | 直接传值 |
| file | `--file` | 二选一 | — | 从文件读取值 |
| tags | `--tags` | NO | — | 逗号分隔标签 |
| note | `--note` | NO | — | 备注 |

---

#### 2.2.2 tcc-update-config — 更新配置

> **意图触发词**: 更新配置、修改配置、编辑配置内容

```bash
bitscli env tcc-update-config \
  --namespace <namespace> \
  --config-name <config_name> \
  [--env <env>] \
  [--tcc-site <site>] \
  [--region <region>] \
  [--dir <dir>] \
  (--value <value> | --file <path>) \
  [--description <desc>] \
  [--data-type <yaml|json|string>] \
  [--config-type <static|...>] \
  [--tags <a,b>] \
  [--note <note>]
```

> **同步 region 组**: 若配置属于同步 region 组（如 `CN` + `China-East`），更新时会自动扩展到该组内已存在副本。

---

### 2.3 发布与审批类命令

#### 2.3.1 tcc-deploy-config — 发布单个配置

> 🟠 **高危操作 — 执行前必须向用户确认**
> **执行前必须展示以下摘要并等待用户明确确认：**
> - Namespace、配置名、目标环境 + 站点
> - 发布版本（from_version → to_version，如已知）
> - publish-mode（`auto`/`force-auto` 会自动推进发布单，需特别注明）

> **意图触发词**: 发布配置、上线配置、推送配置

```bash
bitscli env tcc-deploy-config \
  --namespace <namespace> \
  --config-name <config_name> \
  [--env <env>] \
  [--tcc-site <site>] \
  [--region <region>] \
  [--from-version <n>] \
  [--to-version <n>] \
  [--strategy-id <id>] \
  [--remark <remark>] \
  [--publish-mode <manual|auto|force-auto>]
```

**publish-mode 说明**:

| Mode | 行为 |
|:--|:--|
| `manual` | 仅创建发布单，不自动 start/finish（用于手动滚动） |
| `auto`（默认） | 不需要 review 则自动 start/finish；需要 review 则返回 review 信息 |
| `force-auto` | 强制自动推进，忽略 review 要求（尽力推进） |

**输出关键字段**: `deployment_id`、`console_url`、`need_review`

---

#### 2.3.2 tcc-operate-deployment — 操作发布单步骤

> 🟠 **高危操作 — 执行前必须向用户确认**（仅 `finish` 操作）
> **执行 `--operation finish` 前必须展示以下摘要并等待用户明确确认：**
> - 发布单 ID、当前步骤名称
> - 完成此步骤后的下一个阶段
> - 对于 BOE prod 的发布单：明确标注"此操作将推进 BOE 生产级发布"

> **意图触发词**: 推进发布、手动 start/finish、操作发布单

```bash
bitscli env tcc-operate-deployment \
  --deployment-ref <id_or_publish_details_url> \
  --operation <start|finish|review_pass|review_reject> \
  [--current-step-index <n>] \
  [--env <env>] \
  [--tcc-site <site>]
```

> 未传 `--current-step-index` 时，CLI 尝试自动推断当前步骤；推断失败时需手动指定。

---

#### 2.3.3 tcc-approve-deployment — 审批通过

> 🟠 **高危操作 — 执行前必须向用户确认**
> **执行前必须展示以下摘要并等待用户明确确认：**
> - 发布单 ID、审批通过后将触发配置生效
> - 对于 BOE prod 的发布单：明确标注"此审批将使配置在 BOE 生产级控制面生效"

> **意图触发词**: 通过审批、approve、审批通过

```bash
bitscli env tcc-approve-deployment \
  --deployment-ref <id_or_publish_details_url> \
  [--current-step-index <n>] \
  [--env <env>] \
  [--tcc-site <site>]
```

---

#### 2.3.4 tcc-reject-deployment — 审批驳回

> **意图触发词**: 驳回审批、reject、审批拒绝

```bash
bitscli env tcc-reject-deployment \
  --deployment-ref <id_or_publish_details_url> \
  [--current-step-index <n>] \
  [--env <env>] \
  [--tcc-site <site>]
```

---

### 2.4 常见操作组合参考

> 以下仅为常见使用模式参考，**不构成强制流程**。根据用户实际意图灵活组合。

**模式 A: 创建并发布新配置**
```
tcc-create-config → tcc-deploy-config → (可选) tcc-publish-detail → (如需审批) tcc-approve-deployment
```

**模式 B: 更新已有配置并发布**
```
tcc-list-config → tcc-get-config → tcc-update-config → tcc-deploy-config → (可选) tcc-publish-detail
```

**模式 C: 排查命名空间和目录**
```
tcc-list-sites → tcc-search-namespace → tcc-list-dir → tcc-list-config
```

---

### 2.5 Namespace 管理与 ACL 命令

#### 2.5.1 tcc-create-boe-prod-namespace — 在 BOE prod 创建 Namespace

> 🔴 **极高危操作 — 执行前必须向用户确认**
> 此命令在 BOE 生产级控制面上创建 namespace，影响所有依赖该 namespace 的 BOE 泳道。
> **执行前必须展示以下摘要并等待用户明确确认：**
> - Namespace 名称
> - 将在 BOE prod 控制面创建（不可撤销）
> - 继承来源：CN prod 上同名 namespace 的 owners/operators/viewers

> **用途**: 在 BOE 控制面的 prod 环境创建 namespace，自动从 prod site 查询同名 namespace 的 `node_id`、`owners`、`operators`、`viewers`。
> **支持个人账号和服务账号两种认证方式。**

```bash
bitscli env tcc-create-boe-prod-namespace \
  --name <namespace_name> \
  --description <description> \
  [--owners <owner1,owner2>] \
  [--operators <op1,op2>] \
  [--viewers <viewer1,viewer2>] \
  [--acl-status <on|off>] \
  [--service-auth]
```

| 参数 | 必填 | 默认值 | 说明 |
|---|---|---|---|
| `--name` | ✅ | | namespace 名称（必须在 prod site 上已存在同名 namespace） |
| `--description` | ✅ | | 描述信息 |
| `--owners` | ❌ | 继承 prod | owner 列表，逗号分隔；不传时自动从 prod site 同名 namespace 继承 |
| `--operators` | ❌ | 继承 prod | operator 列表，逗号分隔；不传时自动从 prod site 同名 namespace 继承 |
| `--viewers` | ❌ | 继承 prod | viewer 列表，逗号分隔；不传时自动从 prod site 同名 namespace 继承 |
| `--acl-status` | ❌ | `on` | 访问控制开关 |
| `--service-auth` | ❌ | `false` | 使用服务账号认证 |

> **执行流程**:
> 1. 在 prod site（`cloud.bytedance.net`）获取同名 namespace 详情，提取 `node_info.id`、`owners`、`operators`、`viewers`
> 2. 用该 `node_id` 和继承的角色信息在 boe site（`cloud-boe.bytedance.net`）调用 `/namespace/create`

## 2. 批量操作 — 跨环境同步与发布命令（按需组合）

> **使用前提**：目标环境中 TCC 服务**已通过 `tcc-deploy-create` 创建**。
>
> 批量操作命令和单个操作一样，**没有固定的工作流**，由用户根据实际需求选择执行。
> 典型使用场景的参考示例见第 4 节。

### 2.1 tcc-sync-configs — 批量同步配置

> **意图触发词**: 批量同步配置、同步 TCC 配置到环境、复制 prod 配置、跨环境配置同步

将源环境中某个 namespace 下的配置同步到目标环境。**仅复制内容，不会自动发布**。

```bash
bitscli env tcc-sync-configs \
  --namespace <namespace> \
  --to-env <to_env> \
  --to-site <to_site> \
  --to-regions <region1,region2> \
  [--from-env <from_env>] \
  [--from-region <from_region>] \
  [--dir <dir>] \
  [--tcc-site <site>] \
  [--sync-type <online_version|latest_version>] \
  [--update-exist] \
  [--operator <username>] \
  [--service-auth]
```

| Parameter | Flag | Required | Default | Description |
|:--|:--|:--:|:--|:--|
| namespace | `--namespace` | YES | — | TCC 命名空间（= PSM）。源和目标使用同一 namespace |
| to_env | `--to-env` | YES | — | 目标环境，如 `ppe_xiaoxi11` |
| to_site | `--to-site` | **YES** | — | 目标站点。PPE 环境填 `CN`，BOE 环境填 `BOE` |
| to_regions | `--to-regions` | **YES** | — | 目标 regions（逗号分隔）。PPE 填 `CN`/`China-East` 等，BOE 填 `China-BOE` |
| from_env | `--from-env` | NO | `prod` | 源环境 |
| from_region | `--from-region` | NO | 自动发现 | 源 region。不指定时自动查询 namespace 的所有 region |
| dir | `--dir` | NO | 全部目录 | 只同步指定目录下的配置（如 `/default`）。不指定时同步所有目录 |
| tcc_site | `--tcc-site` | NO | 自动推断 | TCC 站点 |
| sync_type | `--sync-type` | NO | `online_version` | 同步类型：`online_version`（线上版本）/ `latest_version`（最新版本） |
| update_exist | `--update-exist` | NO | `true` | 是否更新已存在的配置 |
| operator | `--operator` | NO | — | 操作人用户名（用于同步请求中标记 operator） |
| service_auth | `--service-auth` | NO | `false` | 使用服务账号 JWT 执行同步。需先执行 `service-auth --secret <secret>` |

**安全校验**:
- `--to-site` 和 `--to-regions` 为**必填参数**，不指定时报错
- 当 `--to-env` 为 `prod` 时，`--to-site` **必须为 `BOE`**，禁止向 prod 的 CN 或其他非 BOE 站点同步（防止误操作覆盖线上配置）

**源站点默认推断规则（`--tcc-site` 未显式指定时）**：

> 该规则决定从哪个 TCC 控制面的 `prod` 环境读取配置。**用户未指定时自动应用，用户明确指定则尊重用户选择**。

| 目标环境类型 | 默认 `--from-env` | 默认 `--tcc-site`（源站点） | 原因 |
|:------------|:-----------------|:--------------------------|:-----|
| BOE prod（`boe_base`，`--to-env prod --to-site BOE`） | `prod` | `prod`（CN 站点） | BOE prod 本身从 CN prod 复制而来，配置基准在 CN 控制面 |
| BOE feature 泳道（`boe_*`，如 `boe_cm_test_2`） | `prod` | `BOE` | BOE 泳道的基准是 BOE prod，应从 BOE 控制面的 prod 同步 |
| PPE 泳道（`ppe_*`） | `prod` | `prod`（CN 站点） | PPE 泳道的基准是 CN prod |

**执行前必须展示计划并等待用户确认**：执行 `tcc-sync-configs` 之前，**必须**向用户展示操作摘要并**等待用户明确回复确认后**才能执行：

| 摘要项 | 内容 |
|:--|:--|
| Namespace | `<namespace>` |
| 同步源 | `<tcc-site>` 控制面 / `<from-env>` 环境 |
| 同步目标 | `<to-site>` 站点 / `<to-env>` 环境 / `<to-regions>` |
| 是否覆盖已有配置 | `--update-exist`（默认 true） |
| ⚠️ BOE prod 警告 | 当 `--to-env prod --to-site BOE` 时，**必须**额外显示："⚠️ 此操作将覆盖 BOE 生产级控制面（cloud-boe.bytedance.net）的 prod 配置，请谨慎确认" |

**服务账号说明**:
- `--service-auth` 使用缓存的服务账号 JWT 调用 TCC API，TCC 平台会以服务账号身份记录操作人
- 服务账号需要对目标 namespace 有 Owner 权限。权限校验基于 **`from_env` 所在站点**（源站点），而非 `des_sites`
- 使用前必须先执行 `bitscli env service-auth --secret <secret>` 完成认证

**行为**:
- 若未指定 `--from-region`，自动发现 namespace 的所有 region（优先通过 namespace detail 获取，回退到 conf_space API）
- 按 region 逐一同步：每个源 region 独立查询配置，默认同步到目标环境的同名 region（如 `CN → CN`、`China-East → China-East`）
- 若指定了 `--dir`，只同步该目录下的配置；未指定则每个 region 内按目录分组，每个目录独立发送一次同步请求
- `from_dir` 与该组配置的实际目录一致，实现目录结构一比一复刻
- 无效 config id 自动跳过（stderr Warning）
- 单个 region 或目录同步失败不影响其他 region/目录继续执行

**输出关键字段**: `region_count`、`config_count`、`success_count`、`fail_count`、`region_results[]`（每个 region 的同步结果，含 `dir_results[]`）

---

### 2.2 tcc-batch-deploy — 批量发布未上线配置

> 🟠 **高危操作 — 执行前必须向用户确认**（对 BOE prod 为 🔴 极高危）
> **执行前必须向用户展示以下摘要并等待用户明确回复确认后才能执行：**
>
> | 摘要项 | 内容 |
> |:--|:--|
> | Namespace | `<namespace>` |
> | 目标环境 | `<env>` |
> | 目标站点（自动推断） | `<site>`（boe_* → BOE，ppe_* → CN） |
> | 待发布配置数 | 自动查询 `latest_version > online_version` 的数量 |
> | ⚠️ BOE prod 警告 | 当 `--env prod --tcc-site BOE` 时，**必须**额外显示："⚠️ 此操作将在 BOE 生产级控制面批量发布配置，不可逆，请谨慎确认" |

> **意图触发词**: 批量发布配置、一键发布、发布所有未上线配置、批量上线

自动查询 namespace 下所有 `latest_version > online_version` 的配置，一次性创建发布单并执行发布。

```bash
bitscli env tcc-batch-deploy \
  --namespace <namespace> \
  --env <env> \
  [--region <region>] \
  [--tcc-site <site>] \
  [--remark <remark>] \
  [--publish-mode <manual|auto|force-auto>] \
  [--service-auth]
```

| Parameter | Flag | Required | Default | Description |
|:--|:--|:--:|:--|:--|
| namespace | `--namespace` | YES | — | TCC 命名空间（= PSM） |
| env | `--env` | YES | — | 目标环境，如 `ppe_xiaoxi11` |
| region | `--region` | NO | `CN` | 查询配置用的 region |
| tcc_site | `--tcc-site` | NO | 自动推断 | TCC 站点 |
| remark | `--remark` | NO | `batch deploy` | 发布备注 |
| publish_mode | `--publish-mode` | NO | `auto` | 发布模式（同 `tcc-deploy-config`） |
| service_auth | `--service-auth` | NO | `false` | 使用服务账号 JWT 执行发布。需先执行 `service-auth --secret <secret>` |

**服务账号说明**:
- `--service-auth` 时，CLI 会根据目标 TCC 站点（由 `env` 前缀推断）自动选择对应的 auth endpoint 获取服务账号 JWT：
  - `ppe_*` 环境 → `cloud.bytedance.net`（prod auth）
  - `boe_*` 环境 → `cloud-boe.bytedance.net`（BOE auth）
  - `prod` 环境 + BOE 站点 → `cloud-boe.bytedance.net`（BOE auth）
- 服务账号需要对目标 namespace 在对应 TCC 站点上有 Owner 权限
- 使用前必须先执行 `bitscli env service-auth --secret <secret>` 完成认证

**publish-mode 说明**:

| Mode | 行为 |
|:--|:--|
| `manual` | 仅创建发布单，不自动推进（需手动 start/finish） |
| `auto`（默认） | 不需要 review 时自动 start → finish；需要 review 时附带审批信息等待审批 |
| `force-auto` | 强制自动推进，忽略 review 要求 |

**行为**:
- 自动筛选 `latest_version > online_version` 的配置（已发布的跳过）
- 无需发布的配置 → 输出提示并正常退出
- 自动获取发布策略 → 执行检查 → 判断审批 → `auto` 模式下自动 start + finish

**输出关键字段**: `deployment_id`、`console_url`、`config_count`、`config_changes`、`need_review`、`start`/`finish`、`strategy_id`

---

## 4. 典型流程参考示例

> 以下为常见场景的**手工操作步骤参考**，每一步都需要用户确认后再执行下一步。
> 这些不是自动化流水线，仅作为操作指引。

### 4.1 BOE prod 完整流程示例（创建 Namespace → 同步 → 发布）

> **场景**: 将某个 TCC 服务的 prod 配置同步到 BOE 的 prod 环境。
> **认证方式**: 支持个人账号（默认）和服务账号（`--service-auth`）两种方式。

**Step 1: 在 BOE prod 创建 Namespace（如尚未创建）**

自动从 prod site 获取 `node_id`、`owners`、`operators`、`viewers` 并继承到 BOE：

```bash
bitscli env tcc-create-boe-prod-namespace \
  --name chenmiao.ppedebug.test \
  --description "chenmiao.ppedebug.test namespace for BOE prod"
```

**Step 2:（可选）配置 ACL 访问控制**

```bash
bitscli env tcc-create-acl-rule \
  --namespace chenmiao.ppedebug.test \
  --psm-list "chenmiao.ppedebug.test" \
  --tcc-site boe
```

**Step 3: 同步 prod 配置到 BOE prod**

区域映射：CN → China-BOE。`--to-env prod --to-site BOE` 表示目标是 BOE 的 prod。

```bash
# 全量同步所有目录
bitscli env tcc-sync-configs \
  --namespace chenmiao.ppedebug.test \
  --from-env prod \
  --to-env prod \
  --to-site BOE \
  --to-regions China-BOE

# 或只同步指定目录
bitscli env tcc-sync-configs \
  --namespace chenmiao.ppedebug.test \
  --from-env prod \
  --to-env prod \
  --to-site BOE \
  --to-regions China-BOE \
  --dir /default
```

**Step 4: 批量发布同步后的配置**

BOE prod 需显式指定 `--region China-BOE` 和 `--tcc-site BOE`：

```bash
bitscli env tcc-batch-deploy \
  --namespace chenmiao.ppedebug.test \
  --env prod \
  --region China-BOE \
  --tcc-site BOE \
  --publish-mode auto
```

> **注意**: BOE prod 可能采用多步发布策略（如 小流量→单机房→全量），
> `auto` 模式仅完成第一步 start+finish，后续步骤需手动推进：

```bash
bitscli env tcc-operate-deployment \
  --deployment-ref <deployment_id> \
  --operation start \
  --current-step-index <next_step_index> \
  --tcc-site BOE

bitscli env tcc-operate-deployment \
  --deployment-ref <deployment_id> \
  --operation finish \
  --current-step-index <next_step_index> \
  --tcc-site BOE
```

**Step 5:（可选）查看发布详情**

```bash
bitscli env tcc-publish-detail --deployment-ref <deployment_id> --tcc-site BOE
```

---

### 4.2 PPE 环境同步示例

```bash
# 同步 prod 配置到 PPE 环境
bitscli env tcc-sync-configs \
  --namespace bits.env.api \
  --from-env prod \
  --to-env ppe_xiaoxi11 \
  --to-site CN \
  --to-regions CN

# 批量发布
bitscli env tcc-batch-deploy \
  --namespace bits.env.api \
  --env ppe_xiaoxi11 \
  --publish-mode auto
```

---

### 4.3 BOE Feature 环境同步示例（服务账号）

```bash
# 服务账号认证
bitscli env service-auth --secret <secret>

# 同步配置
bitscli env tcc-sync-configs \
  --namespace chenmiao.ppedebug.flask \
  --from-env prod \
  --to-env boe_cm_test_2 \
  --to-site BOE \
  --to-regions China-BOE \
  --service-auth

# 批量发布
bitscli env tcc-batch-deploy \
  --namespace chenmiao.ppedebug.flask \
  --env boe_cm_test_2 \
  --region China-BOE \
  --publish-mode auto \
  --service-auth
```

---

## 5. 错误处理参考

| 操作 | 错误场景 | 处理方式 |
|:--|:--|:--|
| tcc-deploy-create | BOE 环境 `tcc service not found` | BOE TCC 站点无此服务，尝试手动指定 `--service-id`（从 prod 获取） |
| tcc-deploy-create | BOE 环境 `record not found` | BOE TCC 站点与 prod 独立，namespace 未在 BOE 站点注册。需先用 `tcc-create-boe-prod-namespace` 创建 |
| tcc-sync-configs | 源 namespace 无配置 | 确认 namespace 是否正确 |
| tcc-sync-configs | 部分同步失败 | 输出失败列表，已成功部分不受影响 |
| tcc-sync-configs | `--to-env prod` 且 `--to-site` 非 BOE | CLI 报错：禁止向 prod 的 CN 或其他非 BOE 站点同步 |
| tcc-sync-configs | 服务账号 PERMISSION_DENIED | 服务账号未在源站点上被加为 namespace Owner |
| tcc-batch-deploy | 没有未发布的配置 | 正常结束，所有配置已是最新 |
| tcc-batch-deploy | 需要审批 (need_review=true) | 输出 console_url，用户前往审批 |
| tcc-batch-deploy | 多步发布策略 | `auto` 模式仅完成首步，后续用 `tcc-operate-deployment` 手动推进 |
| tcc-batch-deploy | 服务账号 PERMISSION_DENIED | 服务账号未在目标 TCC 站点上被加为 namespace Owner |

---

## 6. FSM 阶段说明（TCC 特殊处理）

> 以下说明 TCC 服务在 FSM 各阶段的特殊行为，与 SKILL.MD 主文件配合使用。
> 仅适用于 `tcc-deploy-create`（环境部署命令），单个操作和批量操作不纳入 FSM。

| FSM 阶段 | TCC 行为 |
|:--|:--|
| **INIT** | 识别 `service-type=tcc`。不提取 version/branch/cluster-config/cpu-mem/namespace |
| **SEARCH** | 通过 `instance-meta` 检查服务是否已存在。已存在则告知用户（无 upgrade 操作） |
| **VALIDATE** | 执行 SV-12（禁止 TCE 专有参数和 --namespace）、SV-13（仅需 env+psm+standard-env）、SV-14（service-type 合法） |
| **RESOLVE** | **跳过**。不需要 recommend-cluster 和版本推荐流程 |
| **PLAN** | 纳入部署计划，标注 `Type: TCC`。混合部署时与 TCE 服务一起展示 |
| **EXECUTE** | 调用 `tcc-deploy-create`。可与 TCE 服务并行执行。无先锋-跟随 |
| **MONITOR** | TCC 工单纳入正常监控流程，终态判断与 TCE 相同 |

### 混合部署说明

TCC 与 TCE 服务可同时出现在一次 BATCH_DEPLOY 中。两类服务使用各自的部署命令，互不影响：
- TCE 服务 → `deploy-create` / `deploy-upgrade`
- TCC 服务 → `tcc-deploy-create`

---

## 7. TCC 与 TCE 参数对比速查

| 参数 | TCE (deploy-create) | TCC (tcc-deploy-create) |
|:--|:--:|:--:|
| `--env` | ✅ | ✅ |
| `--psm` | ✅ | ✅ |
| `--standard-env` | ✅ | ✅ |
| `--service-auth` | ✅ | ✅ |
| `--namespace` | ❌ | ❌（TCC 平台命令中使用，= PSM） |
| `--cluster-config` | ✅ | ❌ 不适用 |
| `--cluster-name` | ✅ | ❌ 不适用 |
| `--scm-version` / `--branch` | ✅ | ❌ 不适用 |
| `--base-cluster-id` | ✅ | ❌ 不适用 |
| `--cpu-mem` | ✅ | ❌ 不适用 |
| `--single-idc` | ✅ | ❌ 不适用 |
