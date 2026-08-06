# charge-atom-bits-pipeline — BITS 流水线原子 SKILL

## 定位

通用 BITS 开发任务创建 / 流水线触发 / 状态查询原子。被 `general_dev` 等子流程调用,在「抖音支付计收费结算」DevOps 空间(`space-id=749690368002`)下,基于指定工作流模板(`team-flow-id=751258043394`)创建 dev task,触发自检流水线并轮询结果。

**锁定空间 + 锁定工作流模板**:空间固定为 `749690368002`,工作流模板固定为 `team-flow-id=751258043394`(创建开发任务时必须选此模板)。若调用方另有偏好,可通过 `--from-dev-id <devId>` 指定源模板任务,或用 `--team-flow-id` 覆盖工作流。

## DevOps 空间信息

- **空间名称**: 抖音支付计收费结算
- **space_id**: `749690368002`
- **team_flow_id**(工作流模板): `751258043394`(创建开发任务时固定选用此模板)
- **示例任务模板链接**: https://bits.bytedance.net/devops/749690368002/develop/detail/2261563

## 输入参数（由子流程传入）

```yaml
space_id: 749690368002                     # 固定
team_flow_id: 751258043394                 # 固定,创建开发任务时选用的工作流模板 (--team-flow-id)
task_title: "[charge] ${PRD标题}"           # 开发任务标题
service: example.bytepay.charge             # 待变更的 BITS service 名(必填)
repo_path: bytepay/bytepay_charge          # 仅用于报告输出,不直接进 create 参数
branch: feature/<slug>                     # 当前开发分支(将通过 --change 传入)
prd_title: ${PRD标题}
prd_link: ${PRD链接}
commit_id: ${commit_id}                    # 仅用于报告
mr_iid: ""                                 # 可选;若上轮已有 MR,通过 mr=<iid> 复用
meego_links: []                            # 可选,Meego 工作项 URL/ID 列表
qa_email: ""                               # 可选,QA 邮箱
extra_vars: {}                             # 可选,自定义变量 (--var key=value)
template_dev_id: ""                        # 可选,显式指定模板 dev id

# 循环复跑场景:
existing_dev_id: ""                        # 已有 dev task,直接 quick-run 复跑
```

## 输出（返回给子流程）

```yaml
dev_id: "xxxxxxx"                          # 新建/复用的开发任务 ID(devBasicId)
dev_url: "https://bits.bytedance.net/devops/749690368002/develop/detail/<dev_id>"
pipeline_id: "yyyyyyy"                     # quick-run 触发的流水线 ID
pipeline_run_seq: 3                        # 本轮 run 序号
pipeline_url: "https://..."                # 流水线详情页
pipeline_status: pending | running | success | failed | timeout
mr_status:                                 # check-mr 结果(若有)
  bound: true|false
  mr_iid: "..."
build_log_excerpt: "..."                   # 失败时附最后 200 行
```

## 执行流程

### Step 1. 鉴权检查

```bash
bytedcli auth status      # 期待 ByteCloud Auth: ready
```

未就绪 → 返回错误 `AUTH_REQUIRED`(由 `charge-dev-router` 已在入口处握手完成,本原子不主动登录)。

### Step 2. 创建 / 复用 BITS 开发任务

**首次创建**(无 `existing_dev_id`):

```bash
bytedcli --json bits develop create \
  --space-id 749690368002 \
  --team-flow-id 751258043394 \
  --title "${task_title}" \
  --change "service=${service},branch=${branch}" \
  ${template_dev_id:+--from-dev-id ${template_dev_id}} \
  ${qa_email:+--qa ${qa_email}} \
  ${meego_links:+$(printf -- '--meego %s ' "${meego_links[@]}")} \
  ${extra_vars:+$(for k in "${!extra_vars[@]}"; do printf -- '--var %s=%s ' "$k" "${extra_vars[$k]}"; done)}
```

- `--team-flow-id 751258043394` **必带**:锁定「抖音支付计收费结算」空间下的指定工作流模板,不允许省略走 BITS 默认模板
- `--change` 可重复;若涉及多 service 变更,在调用层拼接多个 `--change`
- 若已有 open MR 想复用,`--change "service=${service},branch=${branch},mr=${mr_iid}"`
- 期待返回 `data.dev_id`(或同义字段),记入 `dev_id`

**复用**(有 `existing_dev_id`):跳过 create,直接进 Step 3。

### Step 3. 触发自检流水线(quick-run)

```bash
bytedcli --json bits develop quick-run \
  --dev-id ${dev_id} \
  --space-id 749690368002 \
  --wait \
  --wait-timeout-sec 1800 \
  --poll-interval-sec 30
```

- `--wait` 让 CLI 在内部轮询直到流水线终态
- 终态:`success` | `failed` | `cancelled`
- 超时:`pipeline_status: timeout`(不阻塞,交由子流程兜底)
- 不带 `--wait` 时需自己轮询 `bytedcli bits develop get --dev-id ${dev_id} --pipeline-id ${pipeline_id}`

期待返回:`pipeline_id`、`run_seq`、`status`、`logs_url`(若有)。

### Step 4. 拉取流水线详情(失败或子流程要求时)

```bash
bytedcli --json bits pipeline ${pipeline_id} --run-seq ${pipeline_run_seq}
```

返回 jobs 列表 + 状态 + 各 step 链接。失败时取最近失败 job 的日志:

```bash
# 找到失败的 job_run_id 后:
bytedcli --json bits job-run ${job_run_id}
bytedcli --json bits step-logs ${job_run_id} <step_name>
```

把日志最后 200 行截入 `build_log_excerpt`。

### Step 5. MR 绑定检查(可选,子流程在测试通过后调用)

```bash
bytedcli --json bits develop check-mr --dev-id ${dev_id}
```

- `bound=true` → 已有 MR;返回 `mr_iid`
- `bound=false` → 子流程后续走 `bytedcli codebase mr create`,再用 `bytedcli bits develop reuse-mr --dev-id ${dev_id} --mr <iid>` 反向绑回

## 增量复跑(同一 dev task,不同 commit)

子流程在循环修复场景下应**复用同一个 `dev_id`**:

1. 把 `existing_dev_id` 透传给本原子
2. Step 2 跳过,直接 Step 3 `quick-run`,产生新的 `pipeline_run_seq`
3. dev task 的运行历史在 BITS UI 上完整可追溯

## 通用约束

- 不操作生产环境,只在 DevOps 空间触发自检流水线(`DevDevelopStage` / `DevDevelopStageSelfTestTask`)
- 不在源码或日志中暴露 token;统一走 SSO JWT
- 创建任务时把 PRD 链接放入 `--meego` 或 `--var prd_link=${prd_link}`,便于审计
- 失败不自动重跑;由子流程结合人工/自动测试结论决定下一步

## 异常处理

| 异常 | 处理 |
|---|---|
| `bytedcli auth status` 未就绪 | 返回 `AUTH_REQUIRED`,由路由层重新走 SSO 握手 |
| `bits develop create` 报权限错误 | 返回错误,提示用户检查 BITS 空间 749690368002 权限 |
| `--change` 中 service 名未在空间注册 | 返回错误,要求确认 `service` 字段 |
| `quick-run` 超时(>30min) | �pipeline_status: timeout`,附 `pipeline_url`,交由子流程兜底 |
| `quick-run` 内部 SSE 断流 | 退化到主动轮询 `bits develop get --dev-id ${dev_id} --pipeline-id ${pipeline_id}`,每 30s 一次 |
| `check-mr` 返回未绑定 | 不视为错误;`mr_status.bound=false` 透传给子流程 |
