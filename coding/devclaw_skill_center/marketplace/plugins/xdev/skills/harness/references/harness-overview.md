# Harness Engineering 概览

本文档是 harness-bootstrap 技能的背景参考，解释 Harness Engineering 的理念、三大支柱，以及为什么要对代码仓库进行 harness 化。

## 什么是 Harness Engineering

Harness Engineering（驾驭工程）是一套让 AI Agent 在代码仓库中高效、安全工作的工程实践体系。核心思想：与其让 Agent 自由发挥，不如通过精心设计的"缰绳"（harness）来引导它——给予足够的上下文、设定清晰的边界、在输出端安装质量门禁。

该理念源自 OpenAI 2025 年发表的工程实践总结，核心发现是：**Agent 的表现上限不取决于模型本身，而取决于它工作的环境有多好。** 一个配置了完善文档、明确约束和自动化检查的仓库，能让任何 AI 编码工具发挥出更高水平。

## 三大支柱

### 支柱一：Context Engineering（上下文工程）

**问题**：Agent 不知道仓库的架构、约束和惯例，容易产出不符合项目风格的代码。

**解决方案**：通过分层文档体系，让 Agent 从总入口出发，按需获取上下文：

- **AGENTS.md**（总入口）：项目简介、文档导航、核心约束
- **ARCHITECTURE.md**（架构文档）：代码地图、模块边界、架构不变量
- **模块级 AGENTS.md**（模块入口）：本模块的目录结构、本地约束、常用命令
- **docs/ 专题文档**：参考资料、操作手册、设计方案

关键原则是"渐进式披露"（Progressive Disclosure）：Agent 不需要一次性读完所有文档，而是按任务需要逐层深入。文档层数由项目复杂度决定，小项目一层就够，大项目可能需要三四层。

### 支柱二：Architecture Constraints（架构约束）

**问题**：Agent 可能写出违反项目架构的代码（如跨层调用、引入循环依赖、忽略代码生成流程）。

**解决方案**：在文档中显式声明"不做什么"的约束，并通过自动化工具强制执行：

- **文档层约束**：在 AGENTS.md 中写明不变量（如"生成代码不可手动修改"、"model 层不依赖 view 层"）
- **Lint 层约束**：通过语言对应的 lint 工具在编码时检查
- **Hook 层约束**：通过 pre-commit hooks 在提交时拦截违规
- **CI 层约束**：通过 CI pipeline 在合并时做最终检查

约束的价值在于描述"不做什么"而非"做什么"。因为"某事物的缺席"光看代码很难发现，但一旦被 Agent 打破，可能造成严重的架构腐化。

详细的不变量编写方法参见 `invariants-guide.md`。

### 支柱三：Garbage Collection（垃圾回收）

**问题**：Agent 产出的代码可能包含占位符实现、幻觉标记、未完成的 TODO、过大的文件等质量问题。

**解决方案**：通过 AI Guard hooks 在提交前自动拦截：

- **占位符检测**：检查 `PLACEHOLDER`、`FIXME(agent)`、`TODO(implement)` 等标记
- **幻觉标记检测**：检查 `<PLACEHOLDER>`、`<INSERT_HERE>`、`<FILL_IN>` 等标记
- **文件长度检查**：防止 Agent 生成超长文件（上限由项目实际情况决定）
- **活跃计划检查**：pre-push 时检查是否有未完成的 ExecPlan

## 为什么要 Harness 化

不做 harness 化的仓库，Agent 的表现类似于"第一天入职的新人没有任何入职文档"——它会猜测项目结构、忽略已有约定、重复实现已有功能。

做了 harness 化的仓库，Agent 的表现类似于"有完善入职手册和 CI 门禁的成熟团队新成员"——它知道去哪里找信息、知道什么不能做、提交前会被自动检查。

## harness-bootstrap 的定位

harness-bootstrap 将上述三大支柱的最佳实践打包为可复用的参考基线和子技能，通过 8 个 Phase 引导用户为 **任意语言、任意框架** 的代码仓库落地 harness engineering：

1. 环境检测 → 深度理解仓库技术栈和现有实践
2. 基础设施 → 安装 hooks、lint 配置、Makefile
3. 文档骨架 → 创建 docs/plans/ 目录和规则/工作流模板
4. 文档生成 → 分析仓库并生成 AGENTS.md 体系
5. 架构文档 → 生成 ARCHITECTURE.md
6. Skill 安装 → 安装 ExecPlan、技术债检查等 Skill
7. 多平台兼容 → 创建 .claude/skills/ 等符号链接
8. 验证 → 运行完整性检查

每个阶段都会向用户报告进展并征求确认，确保用户全程掌控。
