# 服务部署流程

部署内场服务到 PPE 泳道用于 fixloop E2E 测试。走 `bytedcli env` 套件（环境平台 API），这是唯一正确的路径 —— 它把泳道注册到环境平台，BFF（`flow.im.gateway` 等）才能识别 `x-tt-env` 染色 header 并路由流量。**不要**用 `bytedcli tce deploy-lane`，它绕过环境平台会导致 BFF 对染色请求返回 400。

## 依赖的 bytedcli skill

执行任何命令前先读对应 skill 的 SKILL.md 并 `--help` 确认当前版本参数：

| 场景 | bytedcli skill |
|------|---------------|
| 环境创建 / 服务部署 / 升级 / 工单 | `$BYTEDCLI_SKILLS_DIR/bytedance-env/SKILL.md` |
| SCM 构建（iteration ≥ 2 升级前触发） | `$BYTEDCLI_SKILLS_DIR/bytedance-scm/SKILL.md` |
| Pod 级观察 / webshell / env-cascader | `$BYTEDCLI_SKILLS_DIR/bytedance-tce/SKILL.md` |
| 认证和站点切换 | `$BYTEDCLI_SKILLS_DIR/bytedance-auth/SKILL.md` |
| 调用前缀、全局参数 | `$BYTEDCLI_SKILLS_DIR/bytedance-tools/references/invocation.md` |

## 参数映射

fixloop 的参数名 → env 命令 flag：

| fixloop 参数 | env flag | 说明 |
|-------------|----------|------|
| `TCE_LANE` | `--env` | 泳道/环境名（CN PPE 必须 `ppe_` 前缀，正则 `^ppe_[a-zA-Z_0-9]{1,26}$`） |
| `standard-env` | `--standard-env online_cn` | `ppe` 是 env_type，**不是** standard-env |
| `flow-base` | `--flow-base prod` | SCM 基线 |
| `branch` | `--branch`（仅 deploy-tce 支持；upgrade-tce 只能用 `--scm-repo-version`） | |
| `psm` | `--psm` | |
| `--specify-dcs HL:1,LF:1` | 同名 | **PPE 必填**，否则按 prod 规模（90~460 pod）部署 |

## 首轮部署（ITERATION == 1）

### Step 1：校验泳道名

```bash
bytedcli --json env check-name --name <lane> --standard-env online_cn
```

看 `data.result.valid`。`true` 继续；否则换名。

### Step 2：创建环境（= 把泳道注册进环境平台）

```bash
bytedcli --json env create \
  --name <lane> \
  --standard-env online_cn \
  --single-idc false \
  --visibility private \
  --flow-base prod
```

- `--single-idc false`：跨 IDC 部署，具体 IDC 由下一步 `--specify-dcs` 决定
- `--single-idc true` + `--idc LF`：单 IDC 单 pod（更省资源，但只有一个 IDC 覆盖）

看 `data.create_result.status == "Succeed"` 为成功。失败看 `data.create_result.err_code / message`。

> 此步完成后 lane 已注册到环境平台，即使没有任何服务部署进来，BFF 也会识别 `x-tt-env: <lane>` 并 fallback 到 baseline。需要快速确认可用 `references/lane-route-probe.md`。

### Step 3：部署服务

```bash
bytedcli --json env service deploy-tce \
  --env <lane> \
  --standard-env online_cn \
  --psm <psm> \
  --flow-base prod \
  --branch <branch> \
  --specify-dcs HL:1,LF:1
```

**IDC 选择**：`flow.agent.creation` / `flow.alice.creativity*` 用 `HL / LF / LQ`。不确定时查 `bytedcli --json env site baseline-zones --standard-env online_cn` 里的 `lane_zones[].lane_idc_list`。

记录返回的 `data.deploy.id_str`（工单 id，用于 Step 4）和工单里的 `cluster_id`（后续 upgrade 会用，从 `data.deploy.deployment_info.<psm>.meta.clusters[0].base_cluster_id` 获取；也可事后 Step D 查）。

### Step 4：轮询工单完成

```bash
bytedcli --json env ticket get --ticket-id <id> --standard-env online_cn
```

**字段**（全小写）：
- `data.status`：`running` → `success` / `failed`
- `data.deploy_summary.deploy_progress.suc / total`：已完成集群数 / 总数
- `data.update_at`：上次状态变化时间
- `data.message` / `data.err_code`：失败原因

**轮询方式**：每次查询**原样打印完整 JSON**（至少 `status / update_at / deploy_progress / message / err_code` 五字段），**不要**用 shell `case` 或 regex 匹配"是否终态" —— 字段名/大小写换一下就会静默失败。主 context/agent 读原值做语义判断。首轮预期 3～5 分钟，每次查询间隔 30～60 秒。

### Step 5：等待 pod Running

env 套件没有 pod 列表命令，沿用 tce：

```bash
bytedcli --json tce list-instance --psm <psm> --env <lane> --tce-site prod --page-size 20
```

`data.pods[].status` 有 `Running / Pending / NotReady / Creating / Failed`（大写）。读原值，语义判断所有 pod `Running` 即 ok。

## 后续轮部署（ITERATION ≥ 2，修复后重新部署）

`env service upgrade-tce` 不接受 `--branch`，只接受 `--scm-repo-version`。所以比首轮多一步 SCM 构建。

### Step A：push 代码

```bash
cd $BUSINESS_REPO_PATH
git add -A
git commit -m "fix(iter $ITERATION): ..."
git push origin $BRANCH
```

### Step B：触发 SCM 构建

```bash
bytedcli --json scm repo build <repo> \
  --branch <branch> \
  --type online \
  -m "fixloop iter $ITERATION"
```

`<repo>` 从 PSM 反推：`flow.agent.creation` → `flow/agent/creation`。

返回 `data.result.version_version` 或 `data.version`（形如 `1.0.0.9756`），记录为 `$NEW_SCM_VERSION`。

### Step C：等 SCM 构建完成

```bash
bytedcli --json scm repo version list <repo> --page-size 5
```

在结果里找 `version == $NEW_SCM_VERSION` 的那条，看 `status`：
- `build_ok`（`status_display=编译成功`）→ 可继续
- `building` → 继续等（build 通常 2～4 分钟）
- `build_failed / fail / dropped / timeout` → 失败，查 `bytedcli scm repo build-log <repo> <version>`

同样原样打印完整 JSON 语义判断，别匹配字符串。

### Step D：确认 cluster-id

首轮部署工单里已经有（`data.deploy.deployment_info.<psm>.meta.clusters[0].base_cluster_id`），也可按需查：

```bash
# 拿 service-id
bytedcli --json env service list --env <lane> --standard-env online_cn
# data.result.items[].meta.id

# 拿 cluster-id
bytedcli --json tce list-service-clusters --service-id <service-id> --tce-site prod
# data.clusters[0].meta.id
```

### Step E：升级

```bash
bytedcli --json env service upgrade-tce \
  --env <lane> \
  --standard-env online_cn \
  --psm <psm> \
  --cluster-id <cluster-id> \
  --flow-base prod \
  --scm-env-type prod \
  --scm-repo-version $NEW_SCM_VERSION
```

返回 `data.deploy.id_str`（新工单 id），`data.deploy.deployment_info.<psm>.action == "upgrade"`。

### Step F：轮询工单 + pod 滚动

同首轮 Step 4+5。升级工单通常 2～4 分钟完成，pod 滚动替换时 `name` 中间的 replica-set hash 会变（新 pod 在起来）。

## 部署失败修复

### 定位错误

- CLI 报错：响应 `status == "error"`，看 `error.message`
- 工单失败：`env ticket get` 看 `status=failed / message / err_code / deploy_summary`
- 构建失败：`scm repo build-log <repo> <version>` 或 web 端 `job_url`
- Pod 崩溃：`tce list-instance` 拿 pod name → `bytedcli tce webshell exec` 看运行日志

### 分类处理

| 错误 | 症状 | 修复 |
|------|------|------|
| 编译错误 | build-log 里 go build 报错 | 修语法/类型/引用，push 后重跑 build |
| 启动崩溃 | pod `CrashLoopBackOff`，日志里有 panic | 修 nil pointer / 配置缺失，重跑 build + upgrade |
| 环境名冲突 | `env create` err_code 含 `exist` | 换一个名字，或去环境平台 web UI 删掉冲突 lane |
| IDC 不可用 | deploy-tce 报 no schedulable node | 换 `--specify-dcs` 的 IDC（用 `baseline-zones` 查可用集合） |
| 额度不足 | pre_check `validate_error=3` | 减 `--specify-dcs` 数量，PPE 通常 `<IDC>:1` 就够测 |

### 修复原则

- 最小改动：只修导致失败的问题，不顺手重构、不改测试代码、不加功能
- 编译自检：`cd $BUSINESS_REPO_PATH && go build ./...`
- 推代码 → 重跑 build → upgrade-tce，按 Step A–F 走一遍
- **重试上限 3 次**。仍失败 → 详情写到 `$OUTPUT_DIR/iteration_$ITERATION/fix_summary.md`，终止循环交人工

## Workspace 模式下的多 sub-repo 部署

当 `$BUSINESS_REPO` 是 workspace 目录（无 `go.mod`，下辖多个子仓库）时：

1. 主 context 遍历 `$OUTPUT_DIR/detected_targets.jsonl`，串行对每个 target 调一次 Stage 1 subagent
2. **env 只创建一次**（同名 lane 多服务共享）：第一个 target 做 `env check-name` + `env create`，后续 target 跳过创建，只调 `env service deploy-tce` / `upgrade-tce`
3. 通过写入 `$OUTPUT_DIR/lane_state.json`（`{lane_name, standard_env, created: true, clusters: {<psm>: <cluster_id>}}`）在 subagent 之间共享状态
4. 任一 sub-repo 部署失败 → 整个 Stage 1 失败，停止后续 target

## 其他注意

- 首轮（env create + deploy-tce）约 5～7 分钟；后续轮（scm build + upgrade-tce）约 5 分钟
- 临时泳道**不会自动清理**，去环境平台 web UI 删（bytedcli env 当前没 delete 子命令）
- bytedcli 需要内网访问权限（VPN 或办公网络）
- **各层 status 字段大小写不统一**：env 工单 `success/running/failed`（小写），scm `build_ok/building/build_failed`（小写带下划线），tce pod `Running/Pending/Failed`（大写）。**轮询必须原样打 JSON 让 agent 判断**
