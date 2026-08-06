# 3.1.1.2 OpenSpec

> **本节目标**：理解 OpenSpec 的 Artifact-guided 工作流、Schema 引擎机制和中间产物体系。

---

## 概念与来源

**OpenSpec**（[Fission-AI/OpenSpec](https://github.com/Fission-AI/OpenSpec)，v1.3.0）是当前社区最广泛采用的 SDD 框架，支持 20+ AI 助手（Claude Code、Codex、Cursor、Gemini CLI 等）。

核心理念：**Artifact-centric——Spec 是一等公民，代码是 Spec 的派生物**。所有中间产物（proposal、spec、design、tasks）都是 Markdown 文件，存储在仓库中，可版本管理。

### 关键概念：Delta Spec

OpenSpec 特有的 **Delta Spec** 描述"变了什么"而非"全是什么"——这使得 OpenSpec 天然适合 Brownfield（已有项目）开发，不需要从零写完整 Spec。

---

## Slash Command

OpenSpec 通过 **Profile** 分两级命令体系：

### Core Profile（默认，新装即用）

| 命令 | 说明 | 产物 |
|------|------|------|
| `/opsx:propose` | **主入口**——创建 change 并一次性生成所有规划产物 | `proposal.md` + `specs/**/*.md` + `design.md` + `tasks.md` |
| `/opsx:explore` | 思考模式——调研想法、探索代码库、比较方案 | —（对话输出，不持久化） |
| `/opsx:apply` | 按 tasks.md 逐个实现编码 | 代码变更 + 更新 `tasks.md` checkbox |
| `/opsx:archive` | 归档已完成的 change | 移动 change 目录至 archive |

**简单需求快速路径**：`/opsx:propose → /opsx:apply`

### Expanded Profile（通过 `openspec config profile` 启用）

| 命令 | 说明 | 产物 |
|------|------|------|
| `/opsx:new` | 仅创建 change 脚手架，不生成产物 | change 目录 + `.openspec.yaml` |
| `/opsx:continue` | 按依赖图逐个生成下一阶段产物 | DAG 中下一个 artifact（视阶段而定） |
| `/opsx:ff` | 快进——一次生成所有前置阶段产物 | 同 `/opsx:propose` 的全部产物 |
| `/opsx:verify` | 验证实现是否符合 Spec | —（对话输出验证报告） |
| `/opsx:sync` | 将 change 的 delta spec 合并回主 spec | 更新主 `specs/` 目录 |
| `/opsx:bulk-archive` | 批量归档多个已完成的 change | 移动多个 change 目录至 archive |
| `/opsx:onboard` | 引导式教程 | —（交互式教学） |

**复杂需求逐步路径**：`/opsx:new → /opsx:continue → /opsx:continue → /opsx:apply`

### 多平台语法差异

| 平台 | 语法 |
|------|------|
| Claude Code | `/opsx:propose`、`/opsx:apply` |
| Cursor / Windsurf / Copilot | `/opsx-propose`、`/opsx-apply`（横线替代冒号） |
| Trae | `/openspec-propose`、`/openspec-apply-change` |

---

## 工作流

### Schema 引擎（OPSX）

OpenSpec 通过 Schema 定义工作流，每个 Schema 是一个 YAML 文件，声明各阶段的 artifacts 和依赖关系（DAG）：

```yaml
# openspec/schemas/spec-driven/schema.yaml
name: spec-driven
version: 1
description: 标准 SDD 工作流
artifacts:
  - id: proposal
    generates: proposal.md
    requires: []

  - id: specs
    generates: specs/**/*.md
    requires: [proposal]

  - id: design
    generates: design.md
    requires: [proposal]

  - id: tasks
    generates: tasks.md
    requires: [specs, design]

apply:
  requires: [tasks]
  tracks: tasks.md
```

DAG 驱动执行顺序：`proposal → specs + design（并行）→ tasks → apply`

### 自定义 Schema

项目可创建自己的 Schema 适配特定工作流：

```text
.openspec/schemas/
├── spec-driven/          # OpenSpec 默认 Schema
│   └── schema.yaml
└── my-team-flow/         # 团队自定义 Schema
    ├── schema.yaml
    └── instructions/
```

支持 `openspec schema init` 创建、`openspec schema fork` 从默认 Schema 派生。

---

## 中间产物

每个 change 的产物存储在 `openspec/changes/{change-name}/` 下：

```text
openspec/changes/{change-name}/
├── proposal.md            # 需求提案
├── specs/
│   └── {capability}/
│       └── spec.md        # EARS 格式需求规格（每个能力一份）
├── design.md              # 技术方案设计
├── tasks.md               # 任务拆解清单
└── metadata.json          # change 元数据
```

### 1. proposal.md

> **核心思想**：回答"为什么做、改什么、影响什么"——change 的顶层概览。

```markdown
# {Change Name}

## Why
<!-- 概要：变更动机和背景 -->

## What Changes
<!-- 概要：具体改动内容 -->

## Capabilities
### New Capabilities
<!-- 概要：新增的能力 -->
### Modified Capabilities
<!-- 概要：修改的现有能力 -->

## Impact
<!-- 概要：对现有系统的影响面 -->
```

| 维度 | 要求 |
|------|------|
| 范围 | 必须明确 New / Modified / Removed 三类 |
| Impact | 必须评估对现有功能的影响 |
| 粒度 | 每个 Capability 对应一份独立的 spec |

### 2. spec.md（每个 Capability 一份）

> **核心思想**：用 EARS 格式（Event-Action-Response-State）描述需求的增量变化（Delta Spec）。

```markdown
## ADDED Requirements
### Requirement: {name}
#### Scenario: {name}
- WHEN {条件}
- THEN {预期结果}

## MODIFIED Requirements
### Requirement: {name}
- PREVIOUS: {原行为}
- UPDATED: {新行为}

## REMOVED Requirements
### Requirement: {name}
- RATIONALE: {移除原因}
```

| 维度 | 要求 |
|------|------|
| 格式 | 严格 EARS：WHEN/THEN 结构 |
| Delta | 使用 ADDED / MODIFIED / REMOVED / RENAMED 标记增量 |
| 场景化 | 每个 Requirement 包含具体 Scenario |

### 3. design.md

> **核心思想**：基于 spec 做技术决策——架构选择、权衡取舍、风险评估。

```markdown
## Context
<!-- 概要：技术背景和约束 -->

## Goals / Non-Goals
<!-- 概要：做什么 / 明确不做什么 -->

## Decisions
<!-- 概要：关键技术决策及理由 -->

## Risks / Trade-offs
<!-- 概要：风险识别和权衡 -->
```

### 4. tasks.md

> **核心思想**：将 design 拆解为可逐个勾选的 checkbox 任务，每个 2-4 小时。

```markdown
## 1. {Task Group Name}
- [ ] 1.1 {任务描述}
- [ ] 1.2 {任务描述}

## 2. {Task Group Name}
- [ ] 2.1 {任务描述}
```

| 维度 | 要求 |
|------|------|
| 粒度 | 每个 task 2-4 小时可完成 |
| 分组 | 按功能模块或依赖顺序分组 |
| 追踪 | `/opsx:apply` 逐个实现并勾选 |

---

## 定位：小中型项目的标准化 SDD

| 维度 | OpenSpec | 对比 Spec Kit |
|------|---------|--------------|
| Spec 文档数 | ~4 个（proposal + specs + design + tasks） | ~5 个（多了 constitution） |
| 难度 | 新手友好 | 需要熟练掌握 |
| 适合规模 | 小中型项目 | 中大型项目 |
| 核心差异 | 流程灵活，无严格 Phase Gate | Constitution 前置，强约束贯穿 |

OpenSpec 生成的文档比 Spec Kit 少，流程更灵活——适合小中型项目快速上手。需要更严格约束和更完整文档链的中大型项目，建议用 Spec Kit。

---

[上一节：exec-plan](031a1-exec-plan.md) | [下一节：Spec Kit →](031a3-speckit.md) | [返回上级：SDD](031a-sdd.md)
