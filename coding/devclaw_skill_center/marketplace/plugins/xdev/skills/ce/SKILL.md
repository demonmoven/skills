---
name: ce
description: "Compound Engineering 复利工程引擎 - 移植自 EveryInc/compound-engineering-plugin v2.66.1（MIT）。6 核心 + 2 辅助 action 覆盖 ideate → brainstorm → plan → work → review → compound 全循环，每次产出沉淀为可检索知识让下一次更容易。当用户提到 'compound'、'复利工程'、'知识沉淀'、'brainstorm'、'ideate'、'ce review' 时触发。"
argument-hint: "<action> [参数]"
---

# ce

Compound Engineering — 复利工程引擎，统一入口。

> 移植自 [EveryInc/compound-engineering-plugin](https://github.com/EveryInc/compound-engineering-plugin) v2.66.1（MIT）。
> 去除了 config 系统（`.compound-engineering/`）、外部 CLI 依赖（agent-browser / vhs / silicon / ffmpeg）、model 指定、telemetry、自动触发机制，原核心工作流逻辑保留。
> 产物目录对齐 xdev 规范：`docs/xdev/ce/`。

## 用户输入

```text
$ARGUMENTS
```

## 参数格式

`$ARGUMENTS` 格式：`<action> [参数]`

第一个词为 `action`，决定执行哪个 CE 步骤。

| action | 参数 | 说明 | 类型 |
|--------|------|------|------|
| `brainstorm` | `[需求描述]` | 交互式 Q&A 探索需求，产出 requirements.md | 混合 |
| `plan` | `[需求描述或 requirements 路径]` | 将需求转化为结构化实施计划 | 混合 |
| `work` | `[plan 路径或需求描述]` | 按计划执行开发（支持 inline/serial/parallel subagent） | 自动化 |
| `review` | `[base_branch]` | 多 persona 并行 Code review + 自动修复 | 混合 |
| `compound` | `[主题描述]` | 沉淀本次解决方案为可检索知识 | 混合 |
| `ideate` | `[聚焦方向]` | 主动发现代码库中高价值改进点 | 混合 |
| `debug` | `[问题描述]` | 系统化根因定位 + 修复 | 自动化 |
| `optimize` | `[优化目标]` | 迭代优化（实验 + 度量门控） | 混合 |

---

## 执行方式

> **`<skill_dir>` 约定**：以下路径中 `<skill_dir>` 指代本 SKILL.md 所在目录的绝对路径。

### 1. 解析 action

从 `$ARGUMENTS` 中提取第一个词作为 `action`，剩余部分作为该 action 的参数。

### 2. 路由分派

| action 值 | 入口文件 |
|-----------|---------|
| `brainstorm` | `<skill_dir>/actions/brainstorm.md` |
| `plan` | `<skill_dir>/actions/plan.md` |
| `work` | `<skill_dir>/actions/work.md` |
| `review` | `<skill_dir>/actions/review.md` |
| `compound` | `<skill_dir>/actions/compound.md` |
| `ideate` | `<skill_dir>/actions/ideate.md` |
| `debug` | `<skill_dir>/actions/debug.md` |
| `optimize` | `<skill_dir>/actions/optimize.md` |
| 其他 | 输出下方的帮助信息 |

### 3. 执行 action

读取对应的入口 md 文件后：

1. 将文件内容作为当前任务的执行指令
2. 文件中所有 `<skill_dir>` 均指向本 SKILL.md 所在目录
3. 将剩余参数（去掉 action 后的部分）传递给该 action 的执行逻辑

### 4. 帮助信息（action 未匹配时输出）

```
ce — Compound Engineering 复利工程引擎

用法：/ce <action> [参数]

核心循环（每次产出让下一次更容易）：
  brainstorm [需求描述]          探索需求，产出 requirements.md
  plan [需求或 requirements]     结构化实施计划，含并行调研 + 置信度检查
  work [plan 路径或需求]          按计划执行（inline / serial / parallel subagent）
  review [base_branch]          多 persona 并行 Code review（16 审阅者 + 置信度门控）
  compound [主题]               沉淀解决方案为可检索知识（复利工程的灵魂）
  ideate [聚焦方向]              主动发现代码库中高价值改进点

辅助：
  debug [问题描述]               系统化根因定位 + 修复
  optimize [优化目标]            迭代优化（实验 + 度量门控）

典型使用场景：
  /ce brainstorm 用户反馈列表加载太慢
  /ce plan                              # 从已有 requirements.md 生成计划
  /ce work                              # 从已有 plan 开始执行
  /ce review main                       # 对比 main 分支做 code review
  /ce compound                          # 刚修完 bug？沉淀一下
  /ce ideate                            # 给我一些改进建议

产物目录：docs/xdev/ce/{brainstorms,plans,solutions,ideation,reviews}/

搭配使用：
  /exec-plan                            # 更轻量的单文档敏捷开发
  /harness init                         # 仓库工程实践初始化
```
