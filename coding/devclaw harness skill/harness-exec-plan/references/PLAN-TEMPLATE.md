# ExecPlan 骨架模板

创建新 ExecPlan 时，从以下模板开始填充。`{{variable}}` 由代码分析填入。

## Table of Contents

- [时间戳格式](#时间戳格式)
- [修改记录格式](#修改记录格式)
- [Review 摘要模板（Phase 3）](#review-摘要模板phase-3)

---

```markdown
# {{action_oriented_title}}

本 ExecPlan 是一份活文档。进度追踪、意外发现、决策日志 和 成果与复盘 章节必须随工作推进持续更新。

**创建时代码基线：**
- 分支：{{git_branch}}
- Commit SHA：{{git_sha}}
- 时区：{{timezone}}
- 提交策略：按里程碑提交 / 完成后 squash / 混合（选一）

## 目标与全局视角

用几句话解释改动完成后用户能获得什么，以及如何看到它在工作。

**需求对齐记录**（Phase 0 产出）：
- 用户原始需求：{{user_request}}
- Agent 理解：{{agent_understanding}}
- 已确认的边界：
  - 做：{{in_scope}}
  - 不做：{{out_of_scope}}
- 关键澄清问答：
  - Q: {{question_1}} → A: {{answer_1}}
  - Q: {{question_2}} → A: {{answer_2}}

## 进度追踪

- [x] ({{timestamp}}) Phase 0: 需求对齐完成
- [x] ({{timestamp}}) Phase 2: 方案撰写完成
- [ ] ({{timestamp}}) Phase 3: 用户 Review 通过
- [ ] ({{timestamp}}) Milestone 1: {{description}}
- [ ] ({{timestamp}}) Milestone 2: {{description}}
- [ ] ({{timestamp}}) Milestone N: 文档更新
- [ ] ({{timestamp}}) Phase 5: 结果汇报
- [ ] ({{timestamp}}) Phase 7: 代码提交/PR 合并

## 意外发现

- 观察：...
  证据：...

## 决策日志

- 决策：...
  理由：...
  日期/作者：...

## 成果与复盘

在主要里程碑或全部完成时填写。

### 完成汇报（Phase 5 产出）

**目标达成**：✅ / ⚠️ / ❌

**变更概览**：
- 新增/修改/删除文件数
- 关键变更点

**验收结果**：
- 测试：
- 编译：
- Lint：

**已知风险/遗留**：

**建议后续**：

## 上下文与方向

描述与任务相关的仓库当前状态。列出关键文件完整路径。定义所有非显而易见的术语。
假设读者什么都不知道——因为恢复时的 Agent 确实什么都不知道。

## 工作计划

用散文描述编辑和新增的顺序。对每个编辑，指明文件、位置和要修改的内容。

### Milestone 1: {{milestone_title}}

**范围**：{{scope}}

**成果**：{{deliverable}}

**命令**：
    {{verification_command}}

**验收**（刚性量化指标）：
- {{quantitative_criterion_1}}
- {{quantitative_criterion_2}}

### Milestone 2: {{milestone_title}}

...

### Milestone N: 文档更新

**范围**：更新受本次改动影响的仓库文档

**成果**：
- {{doc_file_1}} 更新：{{update_description}}
- {{doc_file_2}} 更新：{{update_description}}

**验收**：
- 更新后的文档与代码实际行为一致
- 新增内容在知识导航表中可达
- 无过时描述

## 具体步骤

列出精确命令、工作目录和预期输出。

    # 在仓库根目录执行
    {{command}}
    # 预期输出：
    # {{expected_output}}

## 验证与验收

描述如何验证和观察系统行为。每个验收条件必须可自动化判定：

- [ ] {{test_criterion}}：`{{test_command}}` 全部 PASS
- [ ] {{build_criterion}}：`{{build_command}}` 零错误
- [ ] {{lint_criterion}}：`{{lint_command}}` 零告警
- [ ] {{behavior_criterion}}：{{observable_behavior}}

## 文档更新

列出需要更新的文档文件及更新内容：

- `{{doc_path}}` — {{update_description}}

如经评估无需更新，写明"已评估，无需文档更新"及理由。

## 幂等性与恢复

说明步骤的可重复性和失败恢复路径。
- 步骤 X 可安全重复执行
- 步骤 Y 如果失败，回滚方式为：{{rollback}}

## 产物与备注

关键的终端输出、diff 或代码片段。
前端任务需包含"实操步骤清单 + 每步截图"。

截图存放在计划目录附近的 `artifacts/` 下，使用相对路径引用：

    ![步骤描述](../../artifacts/{{screenshot_name}}.png)

## 接口与依赖

使用的库、接口签名、类型定义。

- 库：{{library_name}} {{version}}
- 接口：{{interface_signature}}

## 后续修复记录（Phase 6）

如有 Review 反馈或 CI 修复，在此追加：

### Fix 1: {{description}}
- 触发原因：{{trigger}}（Review 反馈 / CI 失败 / 回归）
- 修复内容：{{fix_description}}
- 验证：{{verification}}
```

---

## 时间戳格式

所有时间戳统一格式：`YYYY-MM-DD HH:MM:SS±HH:MM`

**禁止混用时区**。整个 ExecPlan 内所有时间戳使用同一时区。

## 修改记录格式

ExecPlan 修订时，在文件末尾追加：

```markdown
---

[2026-04-20 14:30:00+08:00] 修改说明：{{description}}
```

## Review 摘要模板（Phase 3）

方案撰写完成后，向用户展示精简摘要而非全文：

```markdown
## ExecPlan Review 摘要

**目标**：{{one_sentence_goal}}

**关键技术决策**（需要您拍板）：
1. {{decision_point_1}} — 我的建议：{{recommendation}}
2. {{decision_point_2}} — 我的建议：{{recommendation}}

**里程碑概览**：{{count}} 个里程碑，预估 {{hours}} 小时
{{milestone_list_brief}}

**风险点**：
- {{risk_1}}

**核心验收指标**：
- {{criterion_1}}
- {{criterion_2}}

请 review 后告知是否可以开始执行，或需要修改哪些部分。
```
