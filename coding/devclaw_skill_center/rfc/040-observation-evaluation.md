# 4. 观测 & 评测

> **本章目标**：理解 Agent 时代的可观测性和评测体系——不仅观测 Agent Coding 过程本身，更要观测 Pre/Post Task Completion 各阶段的 Skill 质量和中间产物对最终代码质量的影响，并通过评测集系统化地度量和改进。

---

## 核心理念

### 观测不只是"看 Agent 在干什么"

传统的 Agent 观测只关注 Coding 过程本身（工具调用、token 消耗、错误率）。但在 SDD/TDD 体系下，**观测的核心目标是度量整个研发范式链路的质量**：

```text
传统观测（单点）：
  Agent 编码过程 → 记录 trace → 人看

全链路观测（本文提倡）：
  Pre Task Completion                    Post Task Completion
  ┌─────────────────────┐               ┌────────────────────┐
  │ SDD 产物质量         │               │ E2E 验证质量        │
  │ (spec/plan/tasks)    │ → 编码过程 →  │ UI 还原度           │
  │ TDD 测试设计质量      │               │ 代码门禁通过率       │
  └─────────────────────┘               └────────────────────┘
           ↓                    ↓                ↓
        ──────────── 全部汇入评测集 ────────────
                         ↓
              量化各阶段对最终质量的因果关系
```

### 两个维度

| 维度 | 关注点 |
|------|--------|
| **观测（Observation）** | 采集 Agent 行为数据——trace、工具调用、产物 |
| **评测（Evaluation）** | 度量质量——打分、门禁、基准对比（LLM-as-Judge、vv gate/review、评测集） |

---

## 观测体系

### Trace 采集

> **核心思想**：通过 Hooks 自动采集 Agent 全过程 trace，归一化为统一数据模型，提供可视化分析。

#### 采集机制

| Agent | 采集方式 | 触发点 |
|-------|----------|--------|
| Claude Code | `settings.json` Hooks | `SessionStart`（记录 Git baseline）+ `Stop`（上传 JSONL） |
| OpenCode | 内置 Plugin | `session.created` + `session.idle` |
| Trae (Coco) | 远程拉取 | 滑动窗口遍历 thread spans |

### Trace 数据模型

每条 trace 记录的核心字段（`NormalizedMessage`）：

| 字段 | 说明 |
|------|------|
| `role` | `user` / `assistant` / `system` |
| `timestamp` | ISO 时间戳 |
| `content` | 消息内容 |
| `toolCalls[]` | 工具调用：`{ name, input, toolUseId }` |
| `toolResults[]` | 工具结果：`{ toolName, isError, content, agentId }` |
| `thinking` | 扩展思考/推理内容 |
| `model` | 模型标识 |
| `usage` | Token 消耗：`{ inputTokens, outputTokens, cacheWriteTokens, cacheReadTokens }` |
| `subagentId` | SubAgent 标识（如有） |

### Session 级分析指标

| 指标类别 | 具体指标 |
|----------|----------|
| 时间 | `duration`、`startTime`、`endTime` |
| 消息量 | `userMsgCount`、`assistantMsgCount`、`systemMsgCount` |
| 工具使用 | `totalToolCalls`、`toolCounter`（按工具统计）、`toolErrorMap`（按工具错误率） |
| 文件操作 | `filesEdited`（文件路径 → 编辑次数） |
| Token/Cost | `totalInput`、`totalOutput`、`totalCacheWrite`、`totalCacheRead`、`totalCost`、`cacheEfficiency` |
| 行为 | `thinkingBlocks`、`retries`、`latencies` |
| 派生分数 | `errorRate`、`retryRate`、`thinkingRatio`、`toolDiversity`、`efficiencyScore` |

### 三层分析管线

| 层级 | 触发时机 | 内容 | 缓存策略 |
|------|----------|------|----------|
| **L1 程序化分析** | 每次加载 | 解析 JSONL → 归一化 → 提取量化指标 | 不缓存（始终实时计算） |
| **L2 工程文档指标** | 评测时 | 扫描 trace 中对工程文档的引用，交叉比对仓库实际文件 | 不缓存 |
| **L3 LLM-as-Judge** | 用户触发 | 将 JSONL 发给 LLM 做结构化评估 | 缓存（计算昂贵，结果稳定） |

### Pre/Post 阶段观测

| 阶段 | 观测点 | 数据 |
|------|--------|------|
| **SessionStart**（Pre） | Git baseline 记录 | 当前 HEAD commit |
| **编码过程** | 全量 trace | JSONL 消息流（工具调用、思考、结果） |
| **SessionStop**（Post） | Git diff 计算 | baseline → 当前 HEAD 的完整 diff |
| **工程文档行为** | 文档阅读时序 | `docReadBeforeEdit`：是否在首次代码编辑前读了文档<br>`firstReadTurn` / `firstCodeEditTurn`：精确到 turn |

### Dashboard 可视化

| Tab | 内容 |
|-----|------|
| **Summary** | 指标卡片（Duration/Cost/Turns/Tool Calls/Error Rate/Efficiency/Cache Hit/Models/LLM Score）+ 工程文档效果分析 |
| **Conversation** | 完整消息时间线（user/assistant/tool/thinking） |
| **Cost Analysis** | 累计成本曲线、逐 turn token 堆叠图、逐 turn 明细表 |
| **Trajectory** | 效率指标进度条、工具调用频率、响应延迟、错误率、重试序列、文件触达 |

---

## 评测体系

### 评测维度

两套互补的评测系统：

#### System A：LLM-as-Judge（Session 级质量评估）

> **核心思想**：8 维度加权评分，不仅评"做完了没"，还评"做的过程好不好"——效率、沟通、工程实践都在评分范围内。

| 维度 | 权重 | 评测内容 |
|------|------|----------|
| `taskCompletion` | 20% | 是否完整完成了用户请求 |
| `userAlignment` | 20% | 是否遵循了用户真实意图（检测人工纠正、手动接管、方向偏移） |
| `efficiency` | 15% | 工具调用/工作流是否高效（检测无效调用、冗余循环、过度工程） |
| `errorHandling` | 10% | 出错时的恢复能力 |
| `decisionQuality` | 10% | 工具选择、方案选择、范围判断 |
| `communication` | 10% | 清晰度、简洁度、主动进展汇报 |
| `codeQuality` | 10% | 代码变更的恰当性、风格遵从 |
| `engineeringPractice` | 5% | 是否在编辑前阅读工程文档、是否为复杂任务做规划、是否跑测试 |

此外还提取两类结构化诊断：
- **userCorrections**：用户纠正/拒绝事件，含严重等级（critical/high/medium）
- **inefficiencies**：低效模式，含浪费的 turn 数和改进建议

#### System B：vv Trace Grader（门禁级评测）

> **核心思想**：CI/CD 集成的门禁评测，base 100 扣分制，按 PR/Nightly/Release 三级设不同通过阈值。

| 评测维度 | 检查内容 |
|----------|----------|
| 工具选择正确性 | Read vs cat、Grep vs grep、edit 前是否 read |
| 步骤质量 | 检测 thrashing（3+ 次相同调用）、来回编辑（A→B→A）、空步骤 |
| 控制流合理性 | 执行前是否规划、出错后是否调整策略、步骤数是否合理 |
| 安全行为 | 破坏性 git 命令、未授权删除、跳过 hook |

| 门禁级别 | 通过阈值 |
|----------|----------|
| PR | ≥ 70 |
| Nightly | ≥ 80 |
| Release | ≥ 90 |

#### 工程文档效果评测

> **核心思想**：不只是"工程文档存在不存在"，而是"被用了没有、怎么被用的、没被用的话是否应该被用"——形成工程文档改进的反馈闭环。

| 机制类型 | 说明 | 评测方式 |
|----------|------|----------|
| auto-injected | 系统自动注入的上下文（CLAUDE.md 等） | 是否被引用 |
| hard constraints | 硬约束（hooks、lint） | 是否被触发、是否被绕过 |
| explicit invocation | 显式调用的 Skill/Slash Command | 调用频次、调用时机 |
| passive documentation | 被动文档（ARCHITECTURE.md 等） | 是否在编辑前被阅读 + LLM 评估未读文档是否本应有用 |

---

## 评测集

### 概念

**评测集（Eval Dataset）**是系统化度量 Agent Coding 能力的基础设施——一组标准化的输入/输出对，配合自动化评分器，实现可重复、可对比的质量评测。

### 业界评测集全景

| 评测集 | 来源 | 维度 | 评测内容 |
|--------|------|------|----------|
| **SWE-bench Verified** | Princeton/OpenAI | 代码修复 | 给定 GitHub issue + 代码库，生成 patch，运行测试验证 |
| **FeatureBench** | ICLR 2026 | 功能开发 | 复杂 feature 开发（非 patch 级），200 实例，24 个真实仓库 |
| **ABC-Bench** | OpenMOSS | 全链路 | 代码探索 → 容器化部署 → E2E API 测试 |
| **SWE-EVO** | 2025 | 长周期演进 | 软件长期演进场景，引入 Fix Rate（部分通过的软指标） |
| **Terminal-Bench** | Stanford | 终端操作 | 沙箱命令行环境中的多步操作 |
| **RE-Bench** | METR | 研发能力 | 7 个开放式 ML 研究工程环境，与 71 位人类专家对比 |
| **Plan-RewardBench** | 2026 | 规划质量 | 轨迹级偏好评测：安全拒绝、工具不可用、复杂规划、错误恢复 |
| **AgentBoard** | NeurIPS 2024 | 过程质量 | Progress Rate：对比 Agent 轨迹与预期轨迹，给中间子目标部分得分 |

### 评测数据字段设计

一条完整的评测数据应包含以下字段：

#### 输入字段（Task 定义）

| 字段 | 说明 | 示例 |
|------|------|------|
| `task_id` | 评测任务唯一标识 | `repo__feature-name-001` |
| `repo` | 目标代码仓库 | `github.com/org/repo` |
| `base_commit` | 基线 commit | `abc123` |
| `problem_statement` | 需求/问题描述 | issue 内容或 feature 描述 |
| `difficulty` | 难度分级 | `easy` / `medium` / `hard` |
| `expected_skills` | 预期使用的 Skill 链路 | `[speckit.specify, speckit.plan, fixloop]` |

#### 产出字段（Agent 输出）

| 字段 | 说明 | 示例 |
|------|------|------|
| `trace_jsonl` | 完整 session trace | JSONL 消息流 |
| `patch` | 代码 diff | `git diff` 输出 |
| `intermediate_artifacts` | 中间产物 | `{ spec.md, plan.md, tasks.md, ... }` |
| `tool_calls` | 工具调用序列 | `[{name, input, result, isError}]` |
| `total_tokens` | Token 总消耗 | `125000` |
| `total_cost` | 费用 | `$0.85` |
| `duration` | 耗时 | `320s` |

#### 评分字段（Gold Reference + 自动评分）

| 字段 | 说明 | 示例 |
|------|------|------|
| `gold_patch` | 标准答案 patch | 人工编写的正确 diff |
| `test_patch` | 验证测试 | 用于自动判定 pass/fail 的测试 |
| `FAIL_TO_PASS` | 原本失败 → 应该通过的测试 | `["test_feature_x"]` |
| `PASS_TO_PASS` | 原本通过 → 应该继续通过的测试 | `["test_existing_y"]` |
| `resolved` | 是否通过（二值） | `true` / `false` |
| `fix_rate` | 部分通过率（软指标） | `0.75`（3/4 测试通过） |
| `spec_quality_score` | 中间产物质量评分 | `0.82`（LLM-as-Judge） |
| `plan_compliance` | 规划合规度 | `{ phase: 0.9, order: 1.0, fidelity: 0.85 }` |
| `process_score` | 过程质量评分（8 维度） | `0.78` |

### 评测集设计原则

| 原则 | 说明 |
|------|------|
| **全链路覆盖** | 不只评最终代码，还评中间产物（spec/plan/tasks）和过程（trace） |
| **软硬结合** | 硬指标（测试通过率）+ 软指标（LLM-as-Judge 质量评分） |
| **因果可追溯** | 评测数据中保留完整 trace，可追溯"哪个阶段出了问题" |
| **成本归一化** | 相同质量下，token 消耗少的方案更优（Cost-Normalized Accuracy） |
| **抗数据污染** | 持续更新评测集（参考 SWE-bench-Live），避免模型过拟合 |

### 规划质量评测（From Plan to Action, 2026）

> **核心思想**：评测 Agent 是否忠实执行了规划——区分"碰巧做对了"和"按计划做对了"。

三个合规度指标：

| 指标 | 说明 |
|------|------|
| **Plan Phase Compliance** | Agent 是否执行了计划中规定的各阶段 |
| **Plan Order Compliance** | 各阶段是否按正确顺序执行 |
| **Plan Phase Fidelity** | 每个阶段的执行与计划的一致度 |

`PC = 1` 表示完全合规。使用几何均值聚合。

### 行业缺口（机会领域）

当前评测体系的一个显著缺口：

> **没有评测集专门评估 SDD 流程中生成的中间文档（spec/design/plan）的质量。**

- 现有评测集只评最终代码（SWE-bench）或功能完成度（FeatureBench）
- Plan compliance 研究关注"是否按计划执行"，但不评"计划本身好不好"
- 只有 ~21.6% 的学术研究评估了中间管线和产物

这意味着：**构建一个评测 spec 质量 + test 质量 + implementation 质量的统一评测集，是一个有价值的方向。**

---

[上一章：研发范式](030-dev-paradigm.md) | [下一章：Harness Engineering →](050-harness-engineering.md) | [返回目录](000-index.md)
