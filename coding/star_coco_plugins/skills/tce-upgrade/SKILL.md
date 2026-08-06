---
name: tce-upgrade
description: This skill should be used when the user asks to "upgrade TCE service", "升级 TCE 服务", "更新服务版本", "deploy new version", "升级到新版本", "升级集群", "upgrade cluster", or needs to perform a TCE service upgrade in BOE or PPE environments. This skill guides through the complete upgrade workflow including version selection, cluster selection, confirmation, and deployment monitoring.
allowed-tools: Bash(byte-cli:*)
version: 0.1.0
tags:
  - byte-skill
---

# ByteDance TCE Upgrade (BOE / PPE Only)

## ⛔ 强安全约束（必须遵守）

### 环境限制

- ✅ 允许：
  - `BOE`（所有 env）
  - `CN` 且 env 名称以 `ppe_` 开头（例如 `ppe_xxx`）
- ❌ 禁止：
  - `CN` + `prod`
  - `CN` 下非 `ppe_*` 的任何环境

### 交互确认

- 建议在真正创建升级工单前要求二次确认

### 禁止操作（本 skill 不执行）

- 不执行任何 **生产环境** 变更
- 不执行 Delete/Scale/Rollback/Cancel 等危险操作
- 只做：查询（Search/Get/List）→ 创建升级工单（CreateUpgradeTicket）→ 监控工单（GetDeploymentTicket）


## Prerequisites
0. **uv installed**
   - `curl -LsSf https://astral.sh/uv/install.sh | sh`
1. **byte-cli installed**
   - `uv tool install --index https://bytedpypi.byted.org/simple --force "git+https://code.byted.org/bytedance/byte-skill.git#subdirectory=byte-cli"`
2. **Config path（本 skill 专用）**

   当调用本 skill 时，系统会返回 "Base directory for this skill"。

   需要将此 base directory 与相对路径 `assets/config.json` 拼接成完整路径传给 `--config` 参数：

   ```bash
   # <base-dir> 为系统返回的 Base directory for this skill
   CONFIG="<base-dir>/assets/config.json"
   ```

## 最小升级流程（Happy Path）

下面流程以 `BOE` 为例；`CN` 同理，把 `BOE` 改成 `CN`，并确保 env 满足 `ppe_*` 约束。

### Step 1: SearchService → 获取 service_id

```bash
byte-cli --config "$CONFIG" TCE BOE SearchService \
  --search "${PSM}" \
  --output-filter '.response_body.data'
```

从返回中提取：
- `service_id`（一般在 `data[].meta.id`）

### Step 2: GetService → 确认服务信息与 main repo

```bash
byte-cli --config "$CONFIG" TCE BOE GetService \
  --service-id "${SERVICE_ID}" \
  --output-filter '.response_body.data'
```

建议关注：
- `meta.psm / meta.env / meta.status`
- `build.scm_repo_info[].main_repo == true` 的 repo（用于主版本选择）

### Step 3: ListClusters → 获取可升级集群

```bash
byte-cli --config "$CONFIG" TCE BOE ListClusters \
  --service-id "${SERVICE_ID}" \
  --output-filter '.response_body.data'
```

你需要从每个 cluster 提取：
- cluster id：`meta.id`
- name：`meta.name`
- zone / physical_cluster：`resource.zone` / `resource.physical_cluster`
- rollout_strategy（通常来自 `runtime.traits.rollout_strategy` 或页面默认）

### Step 4: GetRepoInfoList → 获取可选版本

```bash
byte-cli --config "$CONFIG" TCE BOE GetRepoInfoList \
  --service-id "${SERVICE_ID}" \
  --output-filter '.response_body.data'
```

建议处理策略（保持简单即可）：
- 主 repo：给用户展示最近 5 个版本，让用户选一个
- 非主 repo：默认取各自最新版本（页面默认行为）

### Step 5: 组装 CreateUpgradeTicket payload

`CreateUpgradeTicket` 的关键字段：
- `--service`：service_id（integer）
- `--cluster-list`：JSON array 字符串，例如：`[{"id":201473163,"rollout_strategy":"eager"}]`
- `--cluster-info`：JSON object 字符串，至少包含 `runtime.repo_info`（即页面里的 repo_info 列表）

示例（注意 JSON 字符串的引号）：

```bash
CLUSTER_LIST='[{"id":201473163,"rollout_strategy":"eager"}]'
CLUSTER_INFO='{ "runtime": { "repo_info": [
  {"name":"code_forge/pipeline/api","version":"1.0.0.676","description":"...","scm_repo_id":"452040"}
]}}'
```

### Step 6: 用户确认后创建升级工单

```bash
byte-cli --config "$CONFIG" TCE BOE CreateUpgradeTicket \
  --service "${SERVICE_ID}" \
  --pipeline-template 1 \
  --cluster-list "$CLUSTER_LIST" \
  --cluster-info "$CLUSTER_INFO" \
  --output-filter '.response_body.data'
```

返回中提取：
- `ticket_id`（常见在 `data.id` 或 `data.pipeline_id`）

### Step 7: 监控工单

```bash
byte-cli --config "$CONFIG" TCE BOE GetDeploymentTicket \
  --ticket-id "${TICKET_ID}" \
  --output-filter '.response_body.data.meta'
```

- 轮询频率：60s
- 超时建议：10min
- 终态：`success/finished/failed/cancelled`（以实际字段为准）

完整轮询脚本与更友好的输出格式见：`references/monitoring.md`。

## 常见失败场景（精简版）

- `service not found`：PSM/环境不对；回到 Step 1 校验
- `No valid cookie`：确保用户已在 Chrome 浏览器中登录 `cloud.bytedance.net`
- `InvalidParameter`：多见于 `CreateUpgradeTicket` payload 字段缺失/格式不对；对照 `references/api.md` 的 payload 要求
- `Permission denied`：需要 service owner 授权（通常在 `GetService` 的 `auth` 信息能看到 owner/iam）
