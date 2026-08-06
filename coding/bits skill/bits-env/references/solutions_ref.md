# Solutions Reference — 工单失败诊断与修复指南

> 本文件定义了 MONITOR / FAILED 阶段的错误诊断流程。
> 当 `ticket-detail` 返回 `Status=Failed` 时，LLM **必须**查阅本文件，匹配错误模式并执行对应的修复策略。

---

## 使用时机

```
MONITOR 轮询 ticket-detail
    │
    ▼
  Status = Failed ?
    │
    是 → 提取 Message 字段 → 按本文件匹配错误模式 → 执行修复策略
    │
    否 → 继续轮询
```

---

## 错误模式速查表

| 错误关键词 | 错误 ID | 类别 | 可自动修复 | 处理策略 |
|:----------|:--------|:-----|:---------|:---------|
| `get cluster info failed` / `cpu too small` / `clusters can not be found` | standard_env_mismatch | Agent 逻辑错误 | ✅ | 修正 standard-env 后重试 |
| `a same named TCE cluster already exists` | cluster_name_duplicate | Agent 逻辑错误 | ⚠️ 需用户确认 | 询问用户意图，是想要递增 cluster-name 后重试创建，还是转向进行升级 |
| `unfinished tickets in current cluster` | upgrade_unfinished_ticket | Agent 逻辑错误 | ⚠️ 需用户确认 | 引导用户取消前置工单 |
| `get service failed from any scope` | invalid_psm_name | Agent 逻辑错误 | ⚠️ 需用户确认 | 验证 PSM 后重试 |
| `SCM COMMON ERR` | scm_common_err | 上游/配置问题 | ❌ | 引导用户检查 SCM 配置 |
| `branch .* no commits found` | branch_no_commits | Agent 逻辑错误 | ⚠️ 需用户确认 | 引导用户确认分支名 |
| `Bytebuild 发版出错` | bytebuild_failure | 上游构建问题 | ❌ | 引导用户查看构建日志 |
| `please confirm the version` | version_confirm_required | Agent 逻辑错误 | ❌ | 引导用户补全版本信息 |
| `UserCancelDeployErrMsg` | user_cancel_deploy | 用户操作 | ❌ | 告知用户已取消 |
| `UpgradeInstanceMetaV3-DataNotExist` / `UpgradeInstanceMetaV4` | upgrade_not_exist | 平台问题 | ❌ | 忽略，暂不处理 |

---

## 诊断流程

```
ticket-detail 返回 Status=Failed
    │
    ▼
  提取 Message 字段
    │
    ▼
  遍历速查表，正则匹配错误关键词
    │
    ├─ 匹配到"可自动修复 ✅" → 执行自动修复（见下方详细条目）→ 重试部署
    │
    ├─ 匹配到"需用户确认 ⚠️" → 输出诊断结果 + 修复建议 → 等待用户指示
    │
    ├─ 匹配到"不可修复 ❌" → 输出诊断结果 + 平台建议 → 进入 FAILED 报告
    │
    └─ 未匹配任何模式 → 输出原始 Message → 进入 FAILED 报告
```

> **自动修复上限**: 同一工单最多自动重试 **1 次**。重试仍失败则进入 FAILED 报告。

---

## 详细错误条目

### 1. standard_env_mismatch — standard-env 与环境前缀不匹配

**匹配模式**: `get cluster info failed` / `cpu too small` / `clusters can not be found`

**根因**: 部署命令中 `--standard-env` 与 `--env` 前缀不匹配。例如 `boe_xxx` 环境使用了 `--standard-env online_cn`，导致 TCE 在错误的控制面查找集群。

**自动修复**:
1. 检查失败命令的 `--env` 前缀
2. 按映射规则修正 `--standard-env`：`boe_` → `boe`，`ppe_` → `online_cn`
3. 用修正后的参数重新执行部署命令

> 此错误已在 VALIDATE SV-15 规则中预防。若仍出现，说明 SV-15 未生效，修正后重试。

---

### 2. cluster_name_duplicate — TCE 集群同名冲突

**匹配模式**: `a same named TCE cluster already exists`

**根因**: 目标物理/逻辑集群下已存在同名 TCE 集群。常见于并发部署或环境中已有同名集群。

**修复流程**（需用户确认）:
1. 告知用户：目标集群下已存在同名 TCE 集群 `{cluster-name}`
2. 询问用户意图：
   - **升级已有集群**: 将操作从 `deploy-create` 切换为 `deploy-upgrade`，对已存在的同名集群执行升级
   - **用新名字创建**: 询问用户是否有指定名称；若无，默认递增（`default` → `default2`，`defaultN` → `default{N+1}`），用新 cluster-name 重新执行 `deploy-create`
3. 按用户选择执行对应操作

**平台建议**: 在同一物理逻辑集群中，只能存在一个同名的集群。可回退到部署配置界面，修改集群名称或更改物理逻辑集群。集群同名也可能是其他并发部署工单导致，可查看 TCE 部署时间附近的工单排查。

> 对应 VALIDATE SV-04 规则：create 前应检查 cluster-name 是否已存在。

---

### 3. upgrade_unfinished_ticket — 目标集群有未完成工单

**匹配模式**: `unfinished tickets in current cluster`

**根因**: 目标集群上存在前一次未完成的部署工单（running/pending 状态），TCE 拒绝创建新工单。

**修复流程**（需用户确认）:
1. 告知用户：目标集群存在进行中的工单，新部署被拒绝
2. 建议操作：
   - 进入 BITS 环境详情 → 找到对应服务 → 点击"跳转 Bits-TCE" → 在工单页面找到进行中的工单 → 取消
   - 或等待前置工单自然完成
3. 用户确认已处理后，重新执行部署命令

**平台建议**: 当前相同泳道相同集群下存在进行中的 TCE 工单，导致后置工单无法创建成功，可进入 TCE 取消前置进行中的工单。

---

### 4. invalid_psm_name — PSM 名称无效

**匹配模式**: `get service failed from any scope`

**根因**: 部署使用的 PSM 在 TCE 中不存在（BOE/PPE/线上均查不到）。常见于 Agent 推断 PSM 时拼写错误。

**修复流程**（需用户确认）:
1. 告知用户：PSM `{psm}` 在平台上不存在
2. 建议验证：
   - 调用 `bitscli env scm-repo --psm <psm>` 检查 PSM 是否有 SCM 仓库
   - 调用 `bitscli env env-search --psm <psm>` 检查是否有部署记录
3. 若 PSM 确实不存在，请用户提供正确的 PSM 名称
4. 用修正后的 PSM 重新执行部署

**平台建议**: 查找服务在 BOE 和 PPE 以及线上都失败，请到 TCE 确认服务至少在 BOE prod 或者线上存在。

---

### 5. scm_common_err — SCM 版本解析失败

**匹配模式**: `SCM COMMON ERR`

**根因**: SCM 编译/版本解析过程出错。可能原因：
- `standard-env` 不匹配导致 SCM 在错误控制面查找版本（先排查 standard_env_mismatch）
- PSM 的 SCM repo/branch 配置有问题
- 构建依赖缺失

**修复流程**:
1. **先排查 standard-env**：确认 `--standard-env` 与 `--env` 前缀匹配
2. 若 standard-env 正确，引导用户：
   - 点开工单页面的编译工单链接，查看编译失败原因
   - 调用 `bitscli env scm-repo --psm <psm>` 确认 repo 和 branch 是否有效

**平台建议**: 点开编译工单链接，查看编译失败原因，根据提示处理后重试。

---

### 6. branch_no_commits — 分支无有效 commit

**匹配模式**: `branch .* no commits found`

**根因**: 部署指定的 branch 在 SCM 上不存在或没有任何 commit。常见于用户提供的分支名拼写错误或分支尚未推送。

**修复流程**（需用户确认）:
1. 告知用户：分支 `{branch}` 在 SCM 上没有找到有效的 commit
2. 建议验证：先调用 `bitscli env scm-repo --psm <psm>` 获取 `repo_id`，再调用 `bitscli env scm-latest-version --repo-id <repo_id> --branch <branch>` 确认分支是否有有效构建
3. 若分支无效，请用户提供正确的分支名
4. 用修正后的参数重新执行部署

---

### 7. bytebuild_failure — 构建失败

**匹配模式**: `Bytebuild 发版出错`

**根因**: 上游构建系统（Bytebuild/SCM）编译失败，通常是服务代码或构建配置问题，非 Agent 可控。

**处理**: 告知用户构建失败，Agent 无法自动修复。

**平台建议**: 构建 SCM 出错，请点击【SCM 编译】步骤的工单链接查看构建失败详情。

---

### 8. version_confirm_required — 需要确认版本

**匹配模式**: `please confirm the version`

**根因**: 服务存在额外的 SCM 依赖（如 debug 模式引入的附加 repo），Agent 推荐的版本只覆盖了主仓库和 runtime，平台要求补全所有依赖的版本信息。

**处理**: 告知用户该服务有额外的 SCM 依赖，需要手动在工单页面确认完整版本信息后重试。

---

### 9. upgrade_not_exist — 工单详情不可用

**匹配模式**: `UpgradeInstanceMetaV3-DataNotExist` / `UpgradeInstanceMetaV4.*UpgradeServiceDeployment`

**根因**: 平台侧问题，工单详情接口未返回有效信息。可能是工单已过期、接口权限不足或环境已被删除。

**处理**: 报告查询异常，建议用户在 BITS 平台上直接查看工单状态。

---

### 10. user_cancel_deploy — 用户取消部署

**匹配模式**: `UserCancelDeployErrMsg`

**根因**: 用户在部署过程中手动取消了工单，非错误。

**处理**: 告知用户部署已被取消。如需重新部署，可从工单页点击【从错误处重试】或重新发起。

---

## FAILED 报告模板

当错误不可自动修复或重试失败时，输出以下格式：

```
═══════════════════════════════════════════════════
  ❌ Deployment Failed — 诊断报告
═══════════════════════════════════════════════════
  Ticket: {ticket_id}
  Service: {psm}
  Environment: {env}

  错误类型: {error_id} ({category})
  错误信息: {message}

  诊断: {root_cause}
  平台建议: {platform_solution}
  修复建议: {recovery_steps}

  工单链接: {ticket_url}
═══════════════════════════════════════════════════
```

> 若未匹配到任何已知模式，`错误类型` 显示 `unknown`，`诊断` 显示原始 Message，`修复建议` 显示 "请查看工单详情页面获取更多信息"。
