# Search Reference — 搜索与查询接口参考

> 本文件定义了所有**只读查询**操作的完整参数规范与输出格式。
> Plan Skill 在 **SEARCH** 和 **MONITOR** 阶段调用。

**CLI 路径**: `bitscli env`

---

## 命令总览

| # | 命令 | 用途 | 必填参数 | 典型使用场景 | Plan 阶段 |
|:--|:-----|:-----|:---------|:------------|:---------|
| 1 | `env-search` | 环境搜索 | `--keyword` / `--psm` / `--manage` / `--subscribe` (至少一个) + `--standard-env` | **查服务部署在哪些环境** | SEARCH |
| 2 | `tickets` | 查询部署工单列表 | `--env` 或 `--psm` (至少一个) + `--standard-env` | 查看部署历史记录 | SEARCH |
| 3 | `ticket-detail` | 获取工单详情与进度 | `--id` + `--standard-env` | 监控部署进度 | MONITOR |
| 4 | `instance-meta` | 获取环境内服务的实例元数据 | `--env` + `--standard-env` | **查环境里有哪些服务** | SEARCH |
| 5 | `instance-detail` | 获取服务部署拓扑详情 | `--id` + `--standard-env` | 查看集群配置、实例列表 | SEARCH |
| 6 | `env-meta` | 获取环境元数据 | `--env` + `--standard-env` | 查看环境机房、隔离方式 | SEARCH (预检) |
| 7 | `scm-repo` | 获取服务 SCM 仓库信息 | `--psm` | 查看服务的代码仓库地址 | SEARCH / RESOLVE |

---

> **⚠️ CRITICAL — `--standard-env` 强制携带规则**
>
> 除 `scm-repo` 外，所有查询命令**必须始终显式携带** `--standard-env` 参数。
>
> | 命令 | 是否必须携带 `--standard-env` | 原因 |
> |:-----|:---:|:-----|
> | `env-search` | **YES** | 非默认控制面（如 `online_i18n`）不带会搜不到结果 |
> | `tickets` | **YES** | 非默认控制面不带会找不到工单 |
> | `ticket-detail` | **YES** | 非默认控制面不带会返回 404 |
> | `instance-meta` | **YES** | 任何非 `online_cn` 场景不带都会返回 0 实例 |
> | `instance-detail` | **YES** | 非默认控制面不带会返回 404 |
> | `env-meta` | **YES** | 非默认控制面不带会报 env not exists |
> | `scm-repo` | **NO** | 不支持该参数，加了会报错 |
>
> **规则**：在 INIT 阶段确定 `standard-env` 后，后续所有查询命令（除 `scm-repo`）都必须携带。
> 即使 CLI 对某些前缀（如 `boe_`）有自动推断能力，也**必须显式传入**，以确保在所有控制面下行为一致。

---

## 快速决策指南 — 该用哪个命令？

**记忆口诀**：
- 问"**哪些环境**" → 用 `env-search`（环境在结果里）
- 问"**哪些服务**" → 用 `instance-meta`（服务在结果里）
- 看命令名：`env-search` 返回 env 列表，`instance-meta` 返回 instance 列表

**根据问题选择命令：**

| 你的问题 | 使用命令 | 示例 |
|:---------|:---------|:-----|
| "哪些环境部署了服务X？" | `env-search --psm <psm>` | `bitscli env env-search --psm inf.hae.boe --standard-env boe` |
| "环境X里有哪些服务？" | `instance-meta --env <env>` | `bitscli env instance-meta --env boe_test --standard-env boe` |
| "环境X的详细信息？" | `env-meta --env <env>` | `bitscli env env-meta --env boe_test --standard-env boe` |
| "服务X的代码仓库？" | `scm-repo --psm <psm>` | `bitscli env scm-repo --psm inf.hae.boe` |
| "查找环境名包含X的环境？" | `env-search --keyword <keyword>` | `bitscli env env-search --keyword test --standard-env boe` |
| "当前仓库/项目对应的服务在哪些boe环境中运行？" | 找到psm后 `env-search --psm <psm> --standard-env <standard-env>` | `bitscli env env-search --psm inf.hae.boe --standard-env boe` |

**⚠️ 常见错误：**

| ❌ 错误用法 | ✅ 正确用法 | 原因 |
|:-----------|:-----------|:-----|
| `instance-meta --psm <psm>` | `env-search --psm <psm>` | `instance-meta` 必须提供 `--env`，不能只用 `--psm`，且 `--env` 是环境名称而不是 `standard-env` |
| `env-search --env <env>` | `env-meta --env <env>` | `env-search` 用于搜索环境列表，不是查看环境详情 |

---

---

## 1. Search Environments — 环境搜索

### Command

```bash
bitscli env env-search [--keyword <keyword>] [--psm <psm>] [--manage] [--subscribe] --standard-env <env>
```

### Input Parameters

| Parameter | Flag | Required | Description |
|:----------|:-----|:---------|:------------|
| keyword | `--keyword` | ⚠️ 四选一 | 按环境名称模糊搜索 |
| psm | `--psm` | ⚠️ 四选一 | 按服务精确匹配，返回包含该服务的环境 |
| manage | `--manage` | ⚠️ 四选一 | 查询当前用户创建的环境 |
| subscribe | `--subscribe` | ⚠️ 四选一 | 查询当前用户收藏的环境 |
| standard_env | `--standard-env` | **YES** | 标准环境，见下表 |

> **约束**：`keyword` / `psm` / `manage` / `subscribe` 至少提供一个。

### standard_env 取值

| 值 | 场景 |
|:---|:-----|
| `online_cn` | 国内线上环境（默认） |
| `online_i18n` | TikTok 国际化环境 |
| `online_i18nbd` | 国际化 BD 环境 |
| `online_usttp` | 美国 TTP 环境 |
| `online_euttp` | 欧洲 TTP 环境 |
| `boe` | BOE 测试环境 |

> **⚠️ 重要**：BOE 环境的 `standard_env` **仅使用 `boe`**，不要传其他值。`standard_env` 必须与目标环境类型匹配，否则搜索不到结果。

### Examples

```bash
# 按服务搜索
bitscli env env-search --psm inf.hae.boe --standard-env online_cn

# 按环境名模糊搜索
bitscli env env-search --keyword ppe_xiao --standard-env online_cn

# 查询我名下的环境
bitscli env env-search --manage --standard-env online_cn

# 查询收藏的环境
bitscli env env-search --subscribe --standard-env online_cn

# BOE 环境查询（standard_env 必须为 boe）
bitscli env env-search --psm inf.hae.boe --standard-env boe

# 海外环境查询
bitscli env env-search --psm tiktok.recommend.service --standard-env online_i18n
```

### Output Format

```text
Found {N} environments:
| Item       | Value       | Description |
| :---       | :---        | :--- |
| Env Name   | {env_name}  | 环境名称 (API: `name`) |
| Status     | {status}    | 环境状态: Running / Free (API: `status_name`) |
| Creator    | {creator}   | 环境创建者 (API: `creator`) |
| Updated At | {date}      | 最后更新时间 (API: `updated_at`) |
```

### 搜索结果判定逻辑

| 搜索方式 | 结果为空 | 结果非空 |
|:---------|:--------|:--------|
| `--psm` | 服务未部署，可 `deploy-create` | 服务已部署，询问用户是否需要升级，升级用 `deploy-upgrade` |
| `--keyword` | 环境不存在，可创建 | 环境已存在 |
| `--manage` | 当前用户未创建环境 | 返回用户名下环境列表 |

### 典型工作流：判断服务是否已部署

```bash
# 1. 搜索服务部署情况
bitscli env env-search --psm user.auth.service --standard-env online_cn

# 2. 结果包含目标环境 → 询问用户是否需要升级（deploy-upgrade）；不包含 → deploy-create

# 3. 查看实例详情（如需）
bitscli env instance-meta --env <env_name> --psm user.auth.service --standard-env online_cn
bitscli env instance-detail --id <instance_id> --standard-env online_cn
```

### Plan 使用要点

- **环境存在性验证**（VALIDATE SV-05）：返回空 → 可安全 CREATE；非空 → 需进一步查询服务列表
- **服务存在性验证**：`--psm` 搜索确认是否已部署，结合 `instance-meta` 获取实例详情
- **搜索策略**：优先 `--psm`（精确），其次 `--keyword`（模糊），`--manage` 查个人环境

---

## 2. Query Tickets — 查询部署工单列表

### Command

```bash
bitscli env tickets [--env <env>] [--psm <psm>] --standard-env <env>
```

### Input Parameters

| Parameter | Flag | Required | Default | Description |
|:----------|:-----|:---------|:--------|:------------|
| env | `--env` | ⚠️ 至少一个 | — | 环境名称，筛选特定环境的部署单 |
| psm | `--psm` | ⚠️ 至少一个 | — | 服务 PSM，筛选特定服务的部署单 |
| standard_env | `--standard-env` | **YES** | — | 标准环境，必须显式传入 |

> **约束**: `env` 和 `psm` 至少需提供一个。

### Output Format

```text
| Item       | Value       | Description |
| :---       | :---        | :--- |
| Ticket ID  | {ticket_id} | 工单 ID (API: `id`) |
| Action     | {action}    | 变更动作: Deploy / Restart (API: `action`) |
| Status     | {status}    | 工单状态: Success / Running / Failed (API: `status`) |
| Env        | {env_name}  | 关联环境名称 (API: `env`) |
| Creator    | {creator}   | 工单创建者 (API: `create_user`) |
| Created At | {date}      | 创建时间 (API: `create_at`) |
```

---

## 3. Get Ticket Detail — 获取工单详情（MONITOR 核心接口）

### Command

```bash
bitscli env ticket-detail --id <ticket_id> --standard-env <env>
```

### Input Parameters

| Parameter | Flag | Required | Default | Description |
|:----------|:-----|:---------|:--------|:------------|
| id | `--id` | **YES** | — | 工单 ID（从 deploy 返回或 tickets 查询获取） |
| standard_env | `--standard-env` | **YES** | — | 标准环境，必须显式传入 |

### Output Format

```text
### Ticket Details
| Item       | Value       | Description |
| :---       | :---        | :--- |
| ID         | {ticket_id} | 工单 ID |
| Status     | {status}    | 当前状态: Success / Running / Pending / Failed / Cancelled |
| Environment| {env_name}  | 目标环境 |
| Creator    | {user}      | 创建人 |
| Created At | {date}      | 创建时间 |
| Message    | {msg}       | 附加信息或错误消息 |

### Deployment Progress: {suc}/{total} ({rate}%)

### 部署摘要
| 环境/服务 | 集群 | 变更类型 | 变更内容 |
| :--- | :--- | :--- | :--- |
| {env_name} | - | {action} ({status}) | - |
| {psm} | **{cluster}**<br>{zone}/{vc} | {cluster_name} ({status}) | 实例规格变更: ...<br>SCM 依赖变更: ... |
```

### Plan 使用要点 — MONITOR 阶段

- **终态判定**: `Status` 为 Success / Failed / Cancelled 时工单结束
- **进行中**: `Status` 为 Running / Pending 时继续轮询
- **失败分析**: 当 Status=Failed 时，`Message` 字段包含错误原因，用于失败分析
- **进度指示**: `Deployment Progress` 的 `{suc}/{total}` 用于向用户播报进度

---

## 4. Get Instance Meta — 获取环境内服务实例元数据

### Command

```bash
bitscli env instance-meta --env <env> [--psm <psm>] --standard-env <env>
```

### Input Parameters

| Parameter | Flag | Required | Default | Description |
|:----------|:-----|:---------|:--------|:------------|
| env | `--env` | **YES** | — | 环境名称 |
| psm | `--psm` | NO | — | 服务 PSM，指定后只返回该服务的实例 |
| standard_env | `--standard-env` | **YES** | — | 标准环境，必须显式传入（不带会导致非 online_cn 环境返回 0 实例） |

### Output Format

```text
| Item       | Value       | Source Field | Description |
| :---       | :---        | :---         | :--- |
| ID         | {id}        | `id`         | 实例 ID（用于后续 instance-detail 查询） |
| Service    | {service}   | `service`    | 服务名称 (PSM) |
| Type       | {type}      | `service_type`| 服务类型 |
| Env        | {env_name}  | `env`        | 环境名称 |
| Status     | {status}    | `status`     | 实例状态 |
| User       | {user}      | `user`       | 所有者 |
| Language   | {language}  | `meta.debug_info.language` | 编程语言 |
| Updated    | {date}      | `updated_at` | 最后更新时间 |
```

### Plan 使用要点

- **获取实例 ID**: 输出中的 `ID` 字段是调用 `instance-detail` 的必填参数
- **判断服务是否存在**: 不指定 `--psm` 可获取环境所有服务列表，用于判断目标 PSM 是否已部署
- **CLONE 场景**: 不指定 `--psm` 获取源环境的全部服务列表，然后逐个调用 `instance-detail`

---

## 5. Get Instance Detail — 获取服务部署拓扑详情

### Command

```bash
bitscli env instance-detail --id <instance_id> --standard-env <env>
```

### Input Parameters

| Parameter | Flag | Required | Default | Description |
|:----------|:-----|:---------|:--------|:------------|
| id | `--id` | **YES** | — | 实例 ID（从 `instance-meta` 输出的 `ID` 字段获取） |
| standard_env | `--standard-env` | **YES** | — | 标准环境，必须显式传入 |

### Output Format

```text
### Environment & Service Info
| Item       | Value       | Source Field |
| :---       | :---        | :---         |
| Env Name   | {env}       | `env` |
| Type       | {env_type}  | `env_type` |
| Service    | {service}   | `service` |
| User       | {user}      | `user` |
| Updated    | {date}      | `updated_at` |

### Cluster List

#### Cluster: {name} (ID: {id})
| Item             | Value       | Source Field | Description |
| :---             | :---        | :---         | :--- |
| Cluster Name     | {name}      | `instances[].name` | 集群名称 |
| Cluster ID       | {id}        | `instances[].id` | 集群 ID（UPGRADE 必填） |
| Zone             | {zone}      | `instances[].zone` | 区域（精确字符串） |
| Physical Cluster | {physical}  | `instances[].physical_cluster` | 物理集群 |
| Logical Cluster  | {logical}   | `instances[].logical_cluster` | 逻辑集群 |
| VRegion          | {vregion}   | `instances[].vregion` | 虚拟区域 |
| SCM Repo         | {repo_name} | `instances[].repo_info[].name` | 主代码仓库名称 |
| SCM Version      | {version}   | `instances[].repo_info[].version` | 分支或 Tag |
| SCM Commit       | {commit}    | `instances[].repo_info[].commit` | Commit Hash |

**Instances:**
| IDC | Instance Name | Status | IP | Webshell & Container |
| :--- | :--- | :--- | :--- | :--- |
| {idc} | {pod_name} | {status} | {ip} | {deployment_name} |
```

### Plan 使用要点 — 关键数据提取

从此命令输出中需要精确提取以下字段到 ContextSnapshot：

| 需提取字段 | 来源 | 用途 |
|:----------|:-----|:-----|
| `Cluster ID` | `instances[].id` | UPGRADE 时作为 `--cluster-id` 参数 |
| `Cluster Name` | `instances[].name` | 冲突检测（VALIDATE SV-04） |
| `Zone` | `instances[].zone` | 精确值，不可修改，直接用于 cluster-config |
| `Physical Cluster` | `instances[].physical_cluster` | 精确值 |
| `Logical Cluster` | `instances[].logical_cluster` | 精确值 |
| `IDC` | Instances 表中的 `IDC` 列 | 精确值 |
| `SCM Version` | `instances[].repo_info[].version` | 当前部署版本（CLONE/ROLLBACK 参考） |

> **CRITICAL**: Zone / Physical / Logical / IDC 的值必须按原样使用，**逐字符精确匹配**，不可截断或添加后缀。

---

## 6. Get Environment Metadata — 获取环境元数据

### Command

```bash
bitscli env env-meta --env <env> --standard-env <env> [--type <type>]
```

### Input Parameters

| Parameter | Flag | Required | Default | Description |
|:----------|:-----|:---------|:--------|:------------|
| env | `--env` | **YES** | — | 环境名称 |
| standard_env | `--standard-env` | **YES** | — | 标准环境，必须显式传入 |

### Output Format

```text
### Environment Metadata
| Item | Value | Description |
| :--- | :--- | :--- |
| Name | {name} | 环境名称 |
| Type | {type} | 环境类型 (ppe / boe) |
| Standard Env | {standard_env} | 标准环境归属 |
| Status | {status} | 环境当前状态 |
| Manager | {user} | 环境负责人 |
| IDC | {idc} | 所在 IDC 机房 |
| Usage | {env_usage} | 环境用途描述 |
| Isolation | {isolation_way} | 隔离方式 (namespace / cluster) |
| Created At | {created_at} | 创建时间 |
| Updated At | {updated_at} | 更新时间 |
| Description | {desc} | 详细描述 |
```

### Plan 使用要点 — 预检阶段

- **单机房判断**: 检查 `IDC` 字段和 `Isolation` 字段，若环境为单机房隔离，需在 recommend 命令中追加 `--specify-dcs <idc>`

---

## 7. Get SCM Repository Info — 获取服务 SCM 仓库信息

### Command

```bash
bitscli env scm-repo --psm <psm>
```

### Input Parameters

| Parameter | Flag | Required | Default | Description |
|:----------|:-----|:---------|:--------|:------------|
| psm | `--psm` | **YES** | — | 服务 PSM |

### Output Format

```text
### Env SCM Dependencies
| Env Type | Repo Name            | Version   | Branch | Is Main |
| :------- | :------------------- | :-------- | :----- | :------ |
| {type}   | {repo_name}          | {version} | {branch} | {Yes/No}|

### Main Repository Git Info
| Repo Name            | Repo ID | Git URL                                     | Git Source | Desc                     |
| :------------------- | :------ | :------------------------------------------ | :--------- | :----------------------- |
| {repo_name}          | {id}    | {git_url}                                   | {source}   | {desc}                   |
```

### Plan 使用要点

- **版本推荐流程**（非 CLI 命令，详见 `recommend_version_ref.md`）: 用于获取 PPE/Prod 基准版本——先调用 `scm-repo --psm <psm>` 查 `Env SCM Dependencies`，优先取 `env_type=prod` 的 Version；为空时再调用 `scm-latest-version --repo-id <repo_id> --branch <branch>` 获取最新版本
- **本地 Git 验证**: 对比本地仓库地址与 `Main Repository Git Info` 中的 `Git URL`
