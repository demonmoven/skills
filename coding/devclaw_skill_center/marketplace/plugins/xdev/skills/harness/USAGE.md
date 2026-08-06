# harness 使用说明

> Harness Engineering 工程实践一键落地引擎。原项目：`git@code.byted.org:stone/harness-bootstrap.git`

## 核心理念

Agent 的性能瓶颈不在模型本身，而在它工作的环境。通过给仓库装上"约束"（文档体系 + lint + hooks），构建高效安全的 AI Agent 工作环境。三大支柱：

- **Context Engineering**：分层文档体系（AGENTS.md → ARCHITECTURE.md → 模块文档 → reference/guidance/rules/quality 子目录）
- **Architecture Constraints**：显式不变量 + 多级强制（文档 → lint → hooks → CI）
- **Garbage Collection**：AI Guard hooks 自动拦截占位符、幻觉标记、超长文件等典型 AI 质量问题

## 5 个 action 全览

| action | 用户场景 | 是否需用户交互 |
|--------|---------|---------------|
| `init` | 第一次给新仓库做 harness 化 | ✅（Phase 1 访谈 + 各 phase 间确认） |
| `debt-fix` | 周期性技术债盘点和修复 | 部分（确认修复目标） |
| `doc-fix` | 文档与代码对齐修复 | 部分 |
| `evolve` | 文档健康检查 + 演进路由建议 | 仅生成报告，不直接修改 |
| `lint-promote` | 把反复违反的文档约束升级为 lint 规则 | 部分 |

---

## /harness init —— 全流程初始化

为任意代码仓库一键落地 harness。语言无关、框架无关。

### 流程（6 个 Phase）

| Phase | 名称 | 是否需用户交互 | 是否可委托 subagent |
|-------|------|---------------|--------------------|
| 1 | 仓库深度理解（`detect_stack.sh` + 代码采样 + Git 考古 + 用户访谈 + 验证） | ✅ | ❌（必须主 Agent） |
| 2 | Pre-commit Hooks 安装（路径 A：pre-commit / 路径 B：husky 等） | ❌ | ✅ |
| 3 | 文档骨架与知识库（`docs/{reference,guidance,plans,rules,quality}/` + 模板填充） | ❌ | ✅ |
| 4 | AGENTS.md 生成（按仓库规模分层） | ❌ | ✅ |
| 5 | ARCHITECTURE.md 生成（matklad 方法论） | ❌ | ✅ |
| 6 | 18 点验证（已删除原 Phase 8 中与 `.skills/` / 多平台 symlink 相关的 6 项） | ❌ | ❌（必须主 Agent） |

> **注**：原 harness-bootstrap 的 Phase 3.5（演进 skill 拷贝）、Phase 6（Skill 安装）、Phase 7（多平台 symlink）均已废弃 —— 演进能力现已是 harness 自身的 action（debt-fix/doc-fix/evolve/lint-promote），多平台兼容由 xdev CLI 统一处理。

### 适配优先原则（重要）

`/harness init` **不是把一套标准模板贴到所有仓库上**。它的核心原则是 **适配优先，绝不照搬**：

- **hook 选择**：基于 `detect_stack.sh` 的检测结果，**选择性安装**对应语言的 hook，不混装；已有 husky/lefthook 时**不引入** pre-commit
- **lint 规则**：参考 `lint-strategy-guide` 适配每条规则的目的，而非粘贴示例配置
- **文件长度上限**：基于实际仓库的文件长度分布选择合理值，而非强制 600
- **Skill 选择**：只装对当前项目有实际价值的 Skill
- **已有配置**：尊重已有 ESLint/Ruff/Clippy/Makefile/CI 等配置，**绝不默认覆盖**

### 反模式警示

以下行为是错误的：
- 对 100 行代码的小工具安装 5 个 Skill 和 18 个 hook
- 对已有完善 ESLint 配置的项目强行安装 Biome
- 对不使用 AI 编码工具的团队安装 ai-guard hooks
- 对函数普遍 200+ 行的数据管道项目强制 `funlen=120`
- 对 Python 项目安装 golangci-lint 配置
- 对 Rust 项目安装 go-fmt hook

---

## /harness debt-fix —— 技术债修复

扫描架构级技术债（5 条规则：duplicate-persistence、divergent-impl、model-divergence、scattered-state、unabstracted-flow），**每次最多修复 3 个**，逐 commit 验证。

### 安全约束

1. 每次最多修复 3 条
2. 每修复一条独立 commit（可独立回滚）
3. 每个 commit 前必须通过测试和 lint 检查
4. 验证失败立即回滚该条所有改动
5. 低置信度条目需用户**显式批准**

### 触发场景

- 周期性盘点（用户手动触发）
- ExecPlan 完成后顺手修复
- 用户指定特定文件/模块

---

## /harness doc-fix —— 文档治理

以代码为唯一真相来源，修改文档中与代码不符的部分。**只改文档，绝不改代码**。

### 检查 6 个维度

1. ARCHITECTURE.md vs 实际目录结构（幽灵模块 / 未记录模块）
2. code-patterns.md vs 实际代码模式
3. invariants.md vs 实际代码约束
4. AGENTS.md 导航完整性
5. knowledge-gaps.md 中的 open 条目
6. ADR 决策与当前实现是否一致

---

## /harness evolve —— 知识演进编排

**自身不修改任何代码或文档**，只做三件事：检测、报告、路由建议。

### 两种使用场景

| 场景 | 触发方式 | 输出 |
|------|---------|------|
| 事件驱动 | ExecPlan 完成或检测到结构变更 | 检测 7 项命中清单 + 建议调用哪些 action（doc-fix / debt-fix / lint-promote） |
| 定期健康检查 | 用户手动调用 | 5 维度健康度报告 + 高/中/低优先级建议清单 |

### 路由对象

evolve 的"路由建议"指向 harness skill 自身的 3 个 action：

- 文档/代码对齐问题 → 建议 `/harness doc-fix`
- 不变量违反 / 重复实现 → 建议 `/harness debt-fix`
- 反复违反的规则 → 建议 `/harness lint-promote`

---

## /harness lint-promote —— 规范 lint 化

把反复违反的文档级规范升级为自动化 lint 规则。**核心洞察：lint > docs**——文档可被忽略，lint 会阻断提交。

### 混合策略

| 规范类型 | 检测方式 | 实现形式 |
|---------|---------|---------|
| 文本级 | grep / 正则 | Shell pre-commit hook |
| 语义级 | AST | 原生 linter 规则 |
| 设计级 | AI 理解 | 文档 + prompt 强制引用 |

---

## 与其他 skill 的搭配

### `/exec-plan`（独立 skill）

复杂多步任务（数小时以上的开发、跨模块重构）开始前，先用 `/exec-plan` 写设计文档，再开始动手。harness skill 不内置 exec-plan 能力。

### 不迁移的能力（来自原 harness-bootstrap）

| 能力 | 原位置 | 不迁移的原因 |
|------|--------|------------|
| `exec-plan` | `skills/exec-plan/` | Skills Center 已有独立 `/exec-plan` skill |
| `codex-audit` | `skills/codex-audit/` | 依赖 OpenAI Codex CLI（小众），如需手动安装原 skill |
| `install.sh` | 根目录 | xdev CLI 统一分发，无需保留 |
| Phase 3.5（演进 skill 拷贝） | 主 SKILL.md | 演进能力已是 harness 自身的 action |
| Phase 6（Skill 安装到目标仓库） | 主 SKILL.md | xdev CLI 统一分发 |
| Phase 7（多平台 symlink） | 主 SKILL.md | xdev CLI 统一处理 |

---

## 关键文件位置

```
harness/
├── SKILL.md                            # 入口路由
├── USAGE.md                            # 本文件
├── references/                         # 全局方法论文档
│   ├── harness-overview.md
│   ├── harness-engineering.md
│   └── progress-spec.md
└── actions/
    ├── init/                           # 全流程初始化（6 phase）
    │   ├── init.md                     # 编排器
    │   ├── phase-1-analyze/
    │   ├── phase-2-install-hooks/
    │   ├── phase-3-scaffold-docs/
    │   ├── phase-4-write-agents/
    │   ├── phase-5-write-architecture/
    │   └── phase-6-validate/
    ├── debt-fix/
    ├── doc-fix/
    ├── evolve/
    └── lint-promote/
```
