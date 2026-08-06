# 3.1.1.3 Spec Kit

> **本节目标**：理解 GitHub Spec Kit 的 Constitution-driven 工作流和全流程交付机制。

---

## 概念与来源

**Spec Kit**（[github/spec-kit](https://github.com/github/spec-kit)）是 GitHub 官方开源的 SDD 工具包，支持 22+ AI Agent（Claude Code、GitHub Copilot、Gemini CLI、Cursor、Windsurf 等）。

核心理念：**Constitution-driven——先定义项目"宪法"（编码规范/架构原则），再以此驱动全流程**。Constitution 确保 Agent 在整个交付过程中始终遵守项目级别的约束。

---

## Slash Command

### 核心命令（5 个阶段）

| 命令 | 说明 | 产物 |
|------|------|------|
| `/speckit.constitution` | 定义或更新项目宪法 | `.specify/memory/constitution.md` |
| `/speckit.specify` | 基于 constitution 约束生成需求规格 | `docs/spec-kit/{feature}/spec.md` |
| `/speckit.plan` | 技术方案设计（含 Constitution Check 门禁） | `docs/spec-kit/{feature}/plan.md` |
| `/speckit.tasks` | 任务拆解（DAG 依赖排序） | `docs/spec-kit/{feature}/tasks.md` |
| `/speckit.implement` | 逐 task 编码实现 | 代码变更 + 更新 `tasks.md` checkbox |

### 辅助命令

| 命令 | 说明 | 产物 |
|------|------|------|
| `/speckit.clarify` | 识别 spec 中的模糊地带，提出最多 5 个定向澄清问题 | —（对话输出） |
| `/speckit.analyze` | 非破坏性跨产物一致性与质量分析 | —（对话输出分析报告） |
| `/speckit.checklist` | 为当前 feature 生成自定义检查清单 | `docs/spec-kit/{feature}/checklist.md` |
| `/speckit.taskstoissues` | 将 tasks.md 转换为 GitHub Issues | GitHub Issues |

### Git 扩展命令

| 命令 | 说明 | 产物 |
|------|------|------|
| `/speckit.git.commit` | 基于 Spec 上下文生成 commit | git commit |
| `/speckit.git.feature` | 创建 feature 分支 | git branch |
| `/speckit.git.initialize` | 初始化 Git 仓库 | `.git/` |
| `/speckit.git.validate` | 验证 Git 状态与 Spec 一致性 | —（对话输出验证结果） |

**标准工作流**：`/speckit.constitution → /speckit.specify → /speckit.plan → /speckit.tasks → /speckit.implement`

---

## 工作流

### 关键特性

- **Constitution 前置**：在写任何 Spec 之前，先确立项目的基本法——这些规则会贯穿后续所有阶段
- **Constitution Check 门禁**：`/speckit.plan` 在 Phase 0 研究前和 Phase 1 设计后各执行一次 Constitution 合规检查
- **DAG 任务排序**：tasks 阶段自动分析任务间依赖，生成有向无环图排序
- **Constitution 语义版本化**：MAJOR（破坏性原则变更）/ MINOR（新增原则）/ PATCH（措辞优化）
- **VS Code 扩展**：Spec Kit Assistant 提供可视化面板，展示阶段状态、任务清单、DAG 图

---

## 中间产物

```text
.specify/memory/
├── constitution.md          # 项目宪法
docs/spec-kit/{feature-name}/
├── spec.md                  # 需求规格文档
├── plan.md                  # 技术方案
└── tasks.md                 # 任务清单（checkbox + DAG）
```

### 1. constitution.md

> **核心思想**：项目的最高权威文档——所有 Spec、Plan、Task 都必须遵守宪法原则，`/speckit.analyze` 自动将违反 constitution 的问题标为 CRITICAL。

```markdown
# {PROJECT_NAME} Constitution

## Core Principles
### {Principle 1 Name}
<!-- 概要：声明式、可测试的原则，使用 MUST/SHOULD 而非模糊的"应该" -->

### {Principle 2 Name}
<!-- 概要：每条原则独立可验证 -->

## {Section: 技术栈约束 / 安全要求 / 性能标准}
<!-- 概要：项目特定的硬性约束 -->

## {Section: 开发工作流 / 代码审查流程}
<!-- 概要：团队协作约束 -->

## Governance
<!-- 概要：宪法修订规则、合规要求 -->
<!-- Version: x.y.z | Ratified: YYYY-MM-DD -->
```

| 维度 | 要求 |
|------|------|
| 原则表述 | 声明式、可测试，禁止模糊语言（"should" → MUST/SHOULD + rationale） |
| 版本管理 | 语义版本化（MAJOR/MINOR/PATCH） |
| 权威性 | 所有后续阶段产物必须符合 constitution，冲突即 CRITICAL |
| NON-NEGOTIABLE | 关键原则标记为 NON-NEGOTIABLE，不可在后续阶段被绕过 |

### 2. spec.md

> **核心思想**：基于 constitution 约束的用户故事和功能需求——每条需求可追溯到 constitution 原则。

```markdown
# {Feature Name} Specification

## User Stories
### As a {role}, I want to {action} so that {benefit}

## Functional Requirements
### FR-1: {Requirement Name}
- Description: {详细描述}
- Acceptance Criteria: {量化验收标准}
- Constitution Ref: {关联的 constitution 原则}

## Assumptions & Constraints
<!-- 概要：前提假设和限制条件 -->

## Success Criteria
<!-- 概要：feature 级别的成功标准 -->
```

### 3. plan.md

> **核心思想**：技术方案设计，带 Constitution Check 门禁——设计前后各检查一次合规性。

```markdown
# {Feature Name} Implementation Plan

## Technical Context
<!-- 概要：现有架构、相关模块、依赖关系 -->

## Constitution Check
<!-- 概要：本方案遵守 / 涉及的 constitution 原则 -->

## Project Structure
<!-- 概要：文件变更清单、新增模块 -->

## Design Decisions
<!-- 概要：关键技术选择及理由 -->
```

### 4. tasks.md

> **核心思想**：按 DAG 依赖排序的 checkbox 任务清单——每个 task 有独立的验证标准。

```markdown
# {Feature Name} Tasks

## Phase 1: {阶段名}
- [ ] T1: {任务描述}
  - Depends on: —
  - Verify: {验证标准}
- [ ] T2: {任务描述}
  - Depends on: T1
  - Verify: {验证标准}

## Phase 2: {阶段名}
- [ ] T3: {任务描述}
  - Depends on: T1, T2
  - Verify: {验证标准}
```

---

## Constitution 最佳实践

> 以下最佳实践来自 [stone/speckitx](https://code.byted.org/stone/speckitx) 的 `constitution-dev/` 目录中的真实项目宪法。

### 最佳实践 1：代码变更绑定 Skill 自动生成（最重要）

Constitution 最关键的实践是：**将特定文件变更与必须执行的 Skill 绑定**——Agent 在修改这些文件后，宪法约束它必须调用对应的 Skill 完成代码生成，否则为违宪。

```markdown
## Code Generation Toolchain (NON-NEGOTIABLE)

### Principle: 文件变更必须触发对应 Skill

| 变更类型 | 必须执行的 Skill | 说明 |
|----------|-----------------|------|
| IDL 文件（Thrift）修改 | `/upgrade-idl` | 执行后如有新接口，必须同步添加 apis 接口 |
| Schema 文件（SQL）修改 | `/upgrade-sql` | 触发 GORM 代码生成，须同步更新 4 个位置（MySQL/ClickHouse x Docker/K8s） |
| wire.go 修改 | `/upgrade-wire` | 触发依赖注入代码生成 |
| Error Code YAML 修改 | `/upgrade-bizcode` | 触发 Go 错误码代码生成 |
| Config YAML 修改 | `/upgrade-config` | 须同步更新 2 个位置（Docker/K8s） |

### 只读文件保护
`kitex_gen/`、`loop_gen/`、`wire_gen.go` 等生成文件 **严禁手动修改**。
```

### 最佳实践 2：NON-NEGOTIABLE 原则标记

关键原则用 `NON-NEGOTIABLE` 显式标记，Agent 在任何阶段都不可绕过：

```markdown
### I. DDD Architecture (NON-NEGOTIABLE)
<!-- 领域驱动设计：domain 层不依赖 infra -->

### II. Single Module Principle (NON-NEGOTIABLE)
<!-- 一个 phase = 一个 module -->

### VIII. Test-First Principle (NON-NEGOTIABLE)
<!-- 先写测试，再写实现 -->
```

### 最佳实践 3：多环境一致性强制

数据库和配置变更必须同步到所有环境——不是建议而是宪法强制：

```markdown
### Multi-Environment Sync Rule (NON-NEGOTIABLE)
- SQL 变更：必须同步 MySQL Docker + MySQL K8s + ClickHouse Docker + ClickHouse K8s（4 处）
- Config 变更：必须同步 Docker config + K8s config（2 处）
```

### 最佳实践 4：层级宪法体系

主宪法引用领域特定子宪法，避免主文档膨胀：

```markdown
### IX. Domain-Specific Constitutions
- Prompt Domain: 参见 `prompt-domain-specific-constitution.md`
- Evaluation Domain: 参见 `evaluation-domain-specific-constitution.md`
```

### 最佳实践 5：闭环测试强制

开发流程最后一步必须执行端到端测试 Skill，失败则进入"记录 → 分析 → 修复 → 重测"循环：

```markdown
### E2E Testing (NON-NEGOTIABLE)
Phase 7 完成后，**必须执行** `/api-test` Skill：
- 失败 → 记录到 `specs/{branch}/api-test/fail-records.md`
- 分析 → 生成 `fix.md` 修复计划
- 修复 → 重新测试
- 循环直到全部通过
```

---

## 定位：中大型项目的完整 SDD

| 维度 | Spec Kit | 对比 OpenSpec |
|------|---------|--------------|
| Spec 文档数 | ~5 个（constitution + spec + plan + tasks + impl） | ~4 个（无 constitution） |
| 难度 | 熟练——需要理解 Constitution 机制 | 新手友好 |
| 适合规模 | 中大型项目 | 小中型项目 |
| 核心差异 | Constitution 前置，强约束贯穿全流程 | 流程灵活，无严格 Phase Gate |

Spec Kit 比 OpenSpec 多了 Constitution 阶段，生成的文档更完整——适合需要强约束和完整文档链的中大型项目。小中型项目用 OpenSpec 更轻快。

---

[上一节：OpenSpec](031a2-openspec.md) | [下一节：superpowers（SDD 视角）→](031a4-superpowers.md) | [返回上级：SDD](031a-sdd.md)
