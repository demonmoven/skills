# V&V Gate 工作流（门禁检查流程）

> **适用场景**: 本工作流负责调度 5 个子系统执行完整的验证与验收门禁检查。

编排完整的 **规则检查 → 结构测试 → 任务评测 → 轨迹评分 → 回归门禁** 验证流程。

> **关键原则：按 GATE_LEVEL 确定执行范围，按顺序调度子系统，汇总生成 verdict。**

## 参数

| 参数 | 必填 | 说明 | 默认值 |
|------|------|------|--------|
| `TARGET_PATH` | 是 | 待验证代码目录 | — |
| `GATE_LEVEL` | 否 | 门禁级别：pr / nightly / release | `pr` |
| `SPEC_DIR` | 否 | SPEC 文档目录（任务评测用） | — |
| `TRACE_FILE` | 否 | Agent trace 文件（轨迹评分用） | — |
| `OUTPUT_DIR` | 否 | 输出目录 | `.vv-output` |
| `BASE_SHA` | 否 | 基准 commit | `HEAD~1` |
| `SUBSYSTEMS` | 否 | 逗号分隔的子系统列表或 `all` | `all` |

## 前置检查

```bash
mkdir -p {{OUTPUT_DIR}}/vv

# 验证 TARGET_PATH 存在
ls {{TARGET_PATH}}

# 验证 Go 环境
go version

# 写入配置
cat > {{OUTPUT_DIR}}/vv/config.json << 'CONF'
{
  "target_path": "{{TARGET_PATH}}",
  "gate_level": "{{GATE_LEVEL}}",
  "spec_dir": "{{SPEC_DIR}}",
  "trace_file": "{{TRACE_FILE}}",
  "base_sha": "{{BASE_SHA}}",
  "output_dir": "{{OUTPUT_DIR}}"
}
CONF
```

## 门禁级别与执行范围

执行范围由 `GATE_LEVEL` 决定，机器可读配置见 `references/gate-policies.json`，说明文档见 `references/gate-policies.md`。

| 门禁级别 | 执行子系统 | 阈值 |
|----------|-----------|------|
| `pr` | 规则检查 + 回归门禁 | score >= 70, 无 CRITICAL |
| `nightly` | 全部 5 个子系统 | ALL score >= 80, 无 CRITICAL/HIGH |
| `release` | 全部 + 额外检查 | ALL score >= 90, 零 CRITICAL/HIGH/MEDIUM |

### 超时处理策略

| 门禁级别 | 子系统超时 | 处理方式 |
|----------|-----------|---------|
| PR | 5 分钟 | 超时视为该子系统 FAIL，继续后续子系统 |
| Nightly | 30 分钟 | 超时视为该子系统 FAIL，继续后续子系统 |
| Release | 60 分钟 | 超时视为该子系统 FAIL，继续后续子系统 |

超时时：
1. 记录 `subsystem_timeout` finding（severity: high）
2. 该子系统 score 设为 0，pass 设为 false
3. 在 final-report.md 中标注超时的子系统

## 编排架构

本工作流作为**编排者**，按顺序读取各子系统的指令文件并内联执行。

**子系统清单**：

| 子系统 | 指令文件 | 职责 | 门禁级别 |
|--------|----------|------|----------|
| 规则检查 | `subsystems/rule-checker.md` | 规则检查子系统 | pr / nightly / release |
| 结构测试 | `subsystems/structural-tester.md` | 结构测试子系统 | nightly / release |
| 任务评测 | `subsystems/task-evaluator.md` | 任务评测子系统 | nightly / release |
| 轨迹评分 | `subsystems/trace-grader.md` | 轨迹评分子系统 | nightly / release（需 TRACE_FILE） |
| 回归门禁 | `subsystems/regression-gater.md` | 回归门禁子系统 | pr / nightly / release |

**数据传递方式**：子系统间通过 `{{OUTPUT_DIR}}/vv/` 下的文件交换数据，编排者读取 summary.json 判断流程走向。

## 子系统选择逻辑

当 `SUBSYSTEMS` 参数不为 `all` 时，仅执行指定的子系统（覆盖 `GATE_LEVEL` 的默认子系统范围）。否则按 `GATE_LEVEL` 决定执行范围。

**覆盖模式的边界处理**：
- 显式指定子系统时，通过阈值仍由 `GATE_LEVEL` 决定（如 PR 级阈值 70 分）
- 如果指定了 `task` 但未提供 `SPEC_DIR`，该子系统仍会执行，但任务评测基于代码本身判断（无需求文档对照）
- 如果指定了 `trace` 但未提供 `TRACE_FILE`，该子系统产出 score=0、pass=false 的 summary（无轨迹可评）
- 当 `regression` 单独执行而 `rule` 未执行时，回归门禁基于回归检测结果独立判定（跳过规则检查依赖项）

## 完整流程

```
┌─ Step 0: 读取 gate-policies，确定执行范围 ─────┐
│ 根据 GATE_LEVEL 确定需要执行的子系统           │
└──────────────────────────────────────────────┘
                    ↓
┌─ Step 1: 规则检查 ─────────────────────────────┐
│ Read subsystems/rule-checker.md 并按指令执行    │
│ 产出: subsystem_1_rule_checks/summary.json     │
└──────────────────────────────────────────────┘
                    ↓
┌─ Step 2: 结构测试 (nightly+) ──────────────────┐
│ Read subsystems/structural-tester.md 并按指令执行│
│ 产出: subsystem_2_structural_tests/summary.json │
└──────────────────────────────────────────────┘
                    ↓
┌─ Step 3: 任务评测 (nightly+) ──────────────────┐
│ Read subsystems/task-evaluator.md 并按指令执行  │
│ 产出: subsystem_3_task_evals/summary.json      │
└──────────────────────────────────────────────┘
                    ↓
┌─ Step 4: 轨迹评分 (nightly+, 可选) ────────────┐
│ Read subsystems/trace-grader.md 并按指令执行    │
│ 产出: subsystem_4_trace_grading/summary.json   │
└──────────────────────────────────────────────┘
                    ↓
┌─ Step 5: 回归门禁 ─────────────────────────────┐
│ Read subsystems/regression-gater.md 并按指令执行│
│ 产出: subsystem_5_regression_gates/summary.json │
└──────────────────────────────────────────────┘
                    ↓
┌─ Step 6: 失败分类 ─────────────────────────────┐
│ Read checks/failure-classifier.md 并按指令执行  │
│ 产出: failure-taxonomy.json                    │
└──────────────────────────────────────────────┘
                    ↓
┌─ Step 7: 报告生成 ─────────────────────────────┐
│ 生成 verdict.json + final-report.md            │
└──────────────────────────────────────────────┘
```

---

## Step 0: 读取门禁策略

读取并校验门禁策略配置：

```bash
bash scripts/validate-gate-policies.sh references/gate-policies.json
cat references/gate-policies.json
```

根据 `GATE_LEVEL` 确定需要执行的子系统列表和阈值。

---

## Step 1: 规则检查

> 所有门禁级别均执行。

读取 `subsystems/rule-checker.md` 并按其中的指令执行，传递以下参数：
- TARGET_PATH: {{TARGET_PATH}}
- BASE_SHA: {{BASE_SHA}}
- OUTPUT_DIR: {{OUTPUT_DIR}}/vv/subsystem_1_rule_checks
- GATE_LEVEL: {{GATE_LEVEL}}

编排者读取 `{{OUTPUT_DIR}}/vv/subsystem_1_rule_checks/summary.json`。

### Step 1.5: 校验规则检查产出

```bash
bash scripts/validate-summary-json.sh {{OUTPUT_DIR}}/vv/subsystem_1_rule_checks/summary.json
```

如果校验失败或 summary.json 缺失：
- 记录一个 `subsystem_failure` finding（severity: critical）
- 该子系统 score 设为 0，pass 设为 false
- **继续执行后续子系统**（不中断流程）

---

## Step 2: 结构测试（nightly / release）

```
IF GATE_LEVEL in (nightly, release):
  Read subsystems/structural-tester.md 并按指令执行，传递参数：
  - TARGET_PATH: {{TARGET_PATH}}
  - BASE_SHA: {{BASE_SHA}}
  - OUTPUT_DIR: {{OUTPUT_DIR}}/vv/subsystem_2_structural_tests
  - GATE_LEVEL: {{GATE_LEVEL}}
```

### Step 2.5: 校验结构测试产出

```bash
bash scripts/validate-summary-json.sh {{OUTPUT_DIR}}/vv/subsystem_2_structural_tests/summary.json
```

如果校验失败或 summary.json 缺失：
- 记录一个 `subsystem_failure` finding（severity: critical）
- 该子系统 score 设为 0，pass 设为 false
- **继续执行后续子系统**（不中断流程）

---

## Step 3: 任务评测（nightly / release）

```
IF GATE_LEVEL in (nightly, release) AND SPEC_DIR 存在:
  Read subsystems/task-evaluator.md 并按指令执行，传递参数：
  - TARGET_PATH: {{TARGET_PATH}}
  - SPEC_DIR: {{SPEC_DIR}}
  - BASE_SHA: {{BASE_SHA}}
  - OUTPUT_DIR: {{OUTPUT_DIR}}/vv/subsystem_3_task_evals
  - GATE_LEVEL: {{GATE_LEVEL}}
```

### Step 3.5: 校验任务评测产出

```bash
bash scripts/validate-summary-json.sh {{OUTPUT_DIR}}/vv/subsystem_3_task_evals/summary.json
```

如果校验失败或 summary.json 缺失：
- 记录一个 `subsystem_failure` finding（severity: critical）
- 该子系统 score 设为 0，pass 设为 false
- **继续执行后续子系统**（不中断流程）

---

## Step 4: 轨迹评分（nightly / release，可选）

```
IF GATE_LEVEL in (nightly, release) AND TRACE_FILE 存在:
  Read subsystems/trace-grader.md 并按指令执行，传递参数：
  - TRACE_FILE: {{TRACE_FILE}}
  - OUTPUT_DIR: {{OUTPUT_DIR}}/vv/subsystem_4_trace_grading
  - GATE_LEVEL: {{GATE_LEVEL}}
```

### Step 4.5: 校验轨迹评分产出

```bash
bash scripts/validate-summary-json.sh {{OUTPUT_DIR}}/vv/subsystem_4_trace_grading/summary.json
```

如果校验失败或 summary.json 缺失：
- 记录一个 `subsystem_failure` finding（severity: critical）
- 该子系统 score 设为 0，pass 设为 false
- **继续执行后续子系统**（不中断流程）

---

## Step 5: 回归门禁

> 所有门禁级别均执行。

读取 `subsystems/regression-gater.md` 并按指令执行，传递参数：
- TARGET_PATH: {{TARGET_PATH}}
- BASE_SHA: {{BASE_SHA}}
- GATE_LEVEL: {{GATE_LEVEL}}
- OUTPUT_DIR: {{OUTPUT_DIR}}/vv/subsystem_5_regression_gates
- RULE_CHECKS_SUMMARY: {{OUTPUT_DIR}}/vv/subsystem_1_rule_checks/summary.json

### Step 5.5: 校验回归门禁产出

```bash
bash scripts/validate-summary-json.sh {{OUTPUT_DIR}}/vv/subsystem_5_regression_gates/summary.json
```

如果校验失败或 summary.json 缺失：
- 记录一个 `subsystem_failure` finding（severity: critical）
- 该子系统 score 设为 0，pass 设为 false
- **继续执行后续子系统**（不中断流程）

---

## Step 6: 失败分类

读取 `checks/failure-classifier.md` 并按指令执行，传递参数：
- OUTPUT_DIR: {{OUTPUT_DIR}}/vv

分类定义详见 `references/failure-taxonomy.md`。

将分类结果写入 `{{OUTPUT_DIR}}/vv/failure-taxonomy.json`。

---

## Step 7: 报告生成

### 7.1 生成 verdict.json

根据门禁策略和各子系统评分，生成最终裁决。

verdict.json schema 详见 `references/verdict.schema.json` 与 `references/output-format.md`。

裁决逻辑：
1. 收集所有子系统的 score 和 pass 状态
2. 根据 GATE_LEVEL 的阈值判断
3. 存在 CRITICAL finding → `FAIL`
4. 所有子系统 pass 且满足阈值 → `PASS`
5. 部分子系统不 pass 但无 CRITICAL，且分数接近阈值（差距 <= 10） → `SOFT_FAIL`
6. 部分子系统不 pass 但无 CRITICAL，且分数远低于阈值（差距 > 10） → `FAIL`
7. 需要人工审批的情况 → `NEEDS_APPROVAL`

写入 `{{OUTPUT_DIR}}/vv/verdict.json`。

### 7.2 生成 final-report.md

综合所有子系统报告，生成人可读的完整报告。

```markdown
# V&V 验证报告

## 基本信息
- 门禁级别: {{GATE_LEVEL}}
- 目标路径: {{TARGET_PATH}}
- 基准 SHA: {{BASE_SHA}}
- 最终裁决: {verdict}

## 子系统评分
| 子系统 | 分数 | 通过 | 关键发现 |
|--------|------|------|----------|
| 规则检查 | {score} | {pass} | {count} 个问题 |
| ... | ... | ... | ... |

## 失败分类
{failure-taxonomy 内容}

## 详细发现
{各子系统的 findings 汇总}

## 裁决说明
{verdict.reason}
```

写入 `{{OUTPUT_DIR}}/vv/final-report.md`。

### 7.3 执行输出校验

在生成 `failure-taxonomy.json`、`verdict.json` 和 `final-report.md` 后，执行统一输出校验：

```bash
bash scripts/validate-vv-outputs.sh {{OUTPUT_DIR}}/vv
```

如果校验失败，必须优先修复结构化输出，再结束编排流程。

---

## 飞书进度通知

当 `CHAT_ID` 和 `FEISHU_TASK_CARD_SCRIPT` 环境变量存在时，在每个 Step 的开始和结束时发送/更新飞书 task_card。

**Step 开始时 — 发送卡片**：
```bash
STEP_START_TIME=$(date "+%Y-%m-%d %H:%M:%S.%3N")
STEP_MSG_ID=$(python3 $FEISHU_TASK_CARD_SCRIPT send \
  --chat-id $CHAT_ID \
  --task-name "{子系统名}" \
  --agent-type "Worker" \
  --agent-id "vv" \
  --session "vv" \
  --message "{开始消息}" \
  --start-time "$STEP_START_TIME" \
  --status "运行中" \
  --role $FEISHU_ROLE \
  2>/dev/null | grep -oP 'TASK_CARD_MESSAGE_ID=\K.*')
```

**Step 完成时 — 更新卡片**：
```bash
STEP_END_TIME=$(date "+%Y-%m-%d %H:%M:%S.%3N")
python3 $FEISHU_TASK_CARD_SCRIPT update \
  --message-id "$STEP_MSG_ID" \
  --task-name "{子系统名}" \
  --agent-type "Worker" \
  --agent-id "vv" \
  --session "vv" \
  --status "成功" \
  --message "{完成消息}" \
  --start-time "$STEP_START_TIME" \
  --end-time "$STEP_END_TIME" \
  --duration "{耗时}" \
  --role $FEISHU_ROLE
```

## 输出目录结构

详见 `references/output-format.md`。

## References

- 门禁策略：`references/gate-policies.json`
- 门禁说明：`references/gate-policies.md`
- Summary Schema：`references/summary.schema.json`
- 输出格式：`references/output-format.md`
- Verdict Schema：`references/verdict.schema.json`
- 失败分类：`references/failure-taxonomy.md`
- Failure Taxonomy Schema：`references/failure-taxonomy.schema.json`
- Trace 格式：`references/trace-format.md`
