# Write AGENTS.md — 分层文档体系生成

> **本文件是 prompt 指令**，由 `<skill_dir>/actions/init/phase-4-write-agents/write-agents.md` 加载并传给 subagent 执行。原属于 `harness-bootstrap` 项目的 `write-agents-md` 子 skill。

为代码仓库分析目录结构和技术栈，自适应生成分层的 AGENTS.md 文档体系。

## 核心理念

> 给 Agent 一张地图，别给它一本千页大辞典。

AGENTS.md 不是百科全书，而是知识导航。通过分层结构实现渐进式披露（Progressive Disclosure），Agent 从总入口出发，按需深入到具体模块。

## 分层原则

层数由项目实际复杂度决定，不强制固定：

- **小型项目**（单模块，<5 个主要目录）：只需根 AGENTS.md
- **中型项目**（2-4 个独立模块）：根 AGENTS.md + 模块级 AGENTS.md
- **大型项目**（5+ 个模块 + 丰富文档）：根 AGENTS.md + docs/AGENTS.md + 模块级 AGENTS.md

## 执行流程

### Phase 1: 深度分析仓库

在写任何文字之前，必须彻底分析仓库：

1. 列出顶层目录结构
2. 递归探索每个主要目录，理解其职责
3. 检测技术栈（Go/Node/Python/Rust/Java 等）
4. 阅读已有文档（README.md、现有 AGENTS.md 等）
5. 识别模块边界和依赖关系
6. 识别代码生成目录（DO NOT EDIT 标注）
7. 识别测试目录和测试策略
8. 识别配置文件和环境变量

目标是回答：这个项目解决什么问题？有哪些独立模块？模块间的依赖方向？有哪些 Agent 必须遵守的约束？

### Phase 2: 规划文档层级

基于分析结果，决定需要几层文档，向用户展示规划并征求确认。

### Phase 3: 生成根 AGENTS.md

包含以下章节：

1. **项目简介**：一句话 + 技术栈概要
2. **文档体系表**：所有层级文档的位置和作用
3. **知识导航表**（Progressive Disclosure）："我要做什么 → 去哪里看"
4. **核心约束**：从代码分析中提取的具体规则（参见 `<skill_dir>/actions/init/phase-3-scaffold-docs/prompts/invariants-extraction.md`）
   - 文件/函数长度限制
   - 代码生成规则
   - 测试策略
   - 依赖方向
   - 安全约束
5. **代码质量回压**：pre-commit hooks 的安装和使用方式
6. **执行计划**：ExecPlan 使用约定（如已安装）

约束应当具体、可执行，从代码分析中提取，而非空泛的"写好代码"。

**文档感知规则**：在生成导航表之前，Agent 必须扫描 `docs/reference/`、`docs/guidance/`、`docs/quality/` 目录，将所有已存在的文档纳入导航表。不要硬编码文档列表。

具体来说，知识导航表必须包含以下条目（如果对应文档存在）：
- "理解代码惯例和模式" → `docs/reference/code-patterns.md`
- "理解运行时数据流和状态变化" → `docs/reference/runtime-behavior.md`
- "了解外部服务集成方式" → `docs/reference/integrations.md`
- "查看架构决策历史" → `docs/reference/adr/`
- "排查构建/运行时/测试问题" → `docs/guidance/debugging-playbook.md`
- "避开已知坑点" → `docs/guidance/common-pitfalls.md`
- "查看待补充的知识缺口" → `docs/quality/knowledge-gaps.md`

### Phase 4: 生成模块级 AGENTS.md

每个模块只管本模块的内容：模块定位、目录结构、本地约束、常用命令、关键文件。不重复根级约束。

### Phase 5: 生成 docs/AGENTS.md（如需要）

仅当项目有 `docs/` 且包含多个文档时生成，作为中层索引。

docs/AGENTS.md 的"按任务导航"和"目录树"必须覆盖 `docs/reference/`、`docs/guidance/`、`docs/quality/` 下的所有文件。

### Phase 6: 自检

- 每份文档都有导航章节
- 从根 AGENTS.md 可导航到所有重要目录
- 核心约束是具体的、从代码分析中提取的
- 模块级不重复根级内容

## 更新模式

对已有 AGENTS.md 执行更新时：先阅读现有内容 → 重新分析仓库 → 增量更新 → 保留用户自定义内容。

## 语言选择

- 项目已有中文文档 → 中文
- 项目已有英文文档 → 英文
- 默认跟随用户语言偏好
