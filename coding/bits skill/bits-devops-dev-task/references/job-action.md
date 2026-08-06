# 流水线原子相关能力

> **入口判定**：用户说"处理待交互的 job"、"原子通过/拒绝"、"atom approve/reject"、"处理流水线卡住的原子"、"跳过原子"等。

## 概念澄清

流水线常用四类标识：

- **atomID**：原子**类型**的字符串唯一标识（`unique_id`），例如 `bits_env`、`create_upgrade_deployment`、`user_confirm`；不是数值型 ID。
- **pipelineID**：流水线**定义**在开发任务或发布单中的配置标识，表示「可运行的是哪条流水线」，区别于某一次具体运行。
- **runID**（接口字段常见 `pipeline_run_id`, `--pipeline-run-id`）：流水线**单次运行**的实例 ID。
- **jobID**（接口字段常见 `job_run_id`, `--job-run-id`）。

原子状态枚举：UNSPECIFIED(0) / IGNORED(1) / QUEUING(2) / RUNNING(3) / WAITING(4) / SKIPPING(5) / SKIPPED(6) / SKIP_FAILED(7) / ROLLBACKING(8) / ROLLBACKED(9) / ROLLBACK_FAILED(10) / CANCELLING(11) / CANCELED(12) / CANCEL_FAILED(13) / SUCCEEDED(14) / FAILED(15) / CREATED(100)

---

## 原子交互

部分流水线原子会进入**待交互**状态（例如人工确认、审批），需要人在「通过 / 不通过」等按钮上做出选择后，流水线才能继续。CLI 侧按 **查待交互 job → 拉可取操作 → 提交所选操作** 三步完成

### Step 1：查询待交互原子

通常是从 `pipeline_run.job_runs[]` 中筛选 `job_status == 4`（待交互）的 job。

以下原子无需提示待交互：
| atom_unique_id | 说明 | 
|------|------|
| `onesite_pipeline_driver`  | 子流水线驱动器原子，流水线自动驱动运行 |

本步骤必须收集到目标 runID 和 jobID。
- 可根据上下文通过「开发任务->阶段/任务->运行的流水线（主子流水线均支持）」或「发布单->阶段->运行的流水线（主子流水线均支持）」的思路收集
- 如收集不到，引导用户提供更多信息。

没收集到 runID 和 jobID 前，不允许进入下一步。

为每条待交互 job 提取以下字段，供后续步骤使用：

| 字段 | 来源 | 用途 |
|------|------|------|
| `job_name` | `job_runs[].job_name` | 向用户展示原子名称 |
| `job_run_id` | `job_runs[].job_run_id` | Step 2/3 的 `--job-run-id` |
| `pipeline_run_id` | `pipeline_run_detail.run_id` | Step 2/3 的 `--pipeline-run-id` |

**有待交互原子时** — 向用户展示（Markdown 表格，格式规则见 SKILL.md 公共输出纪律）：

```
当前流水线以下待交互原子：

| # | 原子名称 | 所属流水线 | 控制面 |
|---|---------|-----------|--------|
| 1 | 人工确认  | [CN]主流水线 | CN |
| 2 | 自定义审批  | [CN]项目流水线(项目名) | CN |
```

- 若只有 **1 个**待交互原子，自动选中，直接进入 Step 2。
- 若有 **多个**，请用户选择序号后再进入 Step 2。

**无待交互原子时** — 提示用户"当前没有待交互的原子"并结束。


### Step 2：查询可交互按钮

```bash
bitscli devops pipeline atom interactive-actions \
  --job-run-id <job_run_id> \
  --pipeline-run-id <pipeline_run_id>
```

输出结构：`{ jobId, jobName, actions: [] }`

#### 情况 A：actions 非空

向用户展示该原子的可操作按钮（Markdown 表格，格式规则见 SKILL.md 公共输出纪律）：

```
原子「<原子名称>」的可操作按钮：

| 序号 | 按钮 | 英文 |
|---|------|------|
| 1 | 通过 | Approve |
| 2 | 不通过 | Reject |
```

渲染成选项是，**必须**使用`<序号>.<按钮>`展示

若用户在初始请求中已明确指定操作（如"帮我点通过"），Agent 可跳过展示直接进入 Step 3。

#### 情况 B：actions 为空数组

该原子虽处于待交互状态，但其交互入口在页面详情面板中，CLI 无法直接执行。此时引导用户前往界面操作：

```
原子「<原子名称>」当前处于待交互状态，该原子的交互操作需要在 原子-详情Tab 完成。
请前往流程页操作：
https://bits.bytedance.net/devops/{space_id}/develop/detail/{dev_basic_id}/flow
```

**不得**提示"无可交互按钮"或"该 job 不可交互"等误导性信息，这不是错误，只是该原子的交互入口在页面上。


### Step 3：执行交互

```bash
bitscli devops pipeline atom post-action \
  --job-run-id <job_run_id> \
  --pipeline-run-id <pipeline_run_id> \
  --btn-title "<按钮标题>"
```

`--btn-title` 支持中/英文，值来自 Step 2 返回的 action 按钮标题（如 `通过` / `Approve` / `不通过` / `Reject`），**必须与 Step 2 返回的按钮标题一致**。

执行前需确认（除非用户已明确表达意图），确认文案：

```
即将对原子「<原子名称>」执行【<按钮标题>】操作，确认继续？
```

执行后根据返回的 `status` 向用户报告结果（status = 3 是成功操作）：

```
✅ 已对原子「<原子名称>」执行【<按钮标题>】操作。
```

---

## 重试原子

> **入口判定**：用户说"重试原子"、"rerun job"、"重跑这个原子"、"失败的原子重试一下"等。

当原子执行失败（`job_status=15` 表示失败）或用户希望重新运行某个原子时，使用 `rerun` 命令对指定 job 触发重跑。

```bash
bitscli devops pipeline atom rerun \
  --job-run-id <job_run_id> \
  --pipeline-run-id <pipeline_run_id>
```


执行前需向用户确认（除非用户已明确表达意图）：

```
即将对原子「<原子名称>」执行【重试】操作，确认继续？
```

执行后根据返回结果报告：

```
✅ 已对原子「<原子名称>」触发重试。
```

---

## 原子跳过

> **入口判定**：用户说"跳过原子"、"skip job"等。

当原子执行失败（`job_status=15` 表示失败）或用户希望重新运行某个原子时，使用 `skip` 命令对指定 job 触发重跑。

```bash
bitscli devops pipeline atom skip \
  --job-run-id <job_run_id> \
  --pipeline-run-id <pipeline_run_id>
```


执行前需向用户确认（除非用户已明确表达意图）：

```
即将对原子「<原子名称>」执行【跳过】操作，确认继续？
```

执行后根据返回结果报告：

```
✅ 已对原子「<原子名称>」触发跳过。
```

---