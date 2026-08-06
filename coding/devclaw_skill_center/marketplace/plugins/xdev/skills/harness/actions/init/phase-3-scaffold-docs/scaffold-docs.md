# Phase 3: 文档骨架与知识库体系（三类文档生成）

> **本 phase 在 `/harness init` 流程中的位置**：第 3 步，可委托 subagent 执行。依赖 Phase 1 产出的 `harness-init-analysis.md`。

**目标**：创建完整的 `docs/` 知识库体系，基于 Phase 1 的深度分析结果，生成三类文档——标准骨架、深度文档、知识缺口清单。

> 不变量提取的详细标准参见 `<skill_dir>/actions/init/phase-3-scaffold-docs/prompts/invariants-extraction.md`。

`docs/` 是仓库专题文档的统一收纳目录。它采用分类子目录的方式组织，每个子目录有明确的定位：

| 子目录 | 定位 | 内容类型 | 典型文件 |
|---|---|---|---|
| `reference/` | **稳定事实说明**——"是什么" | 代码模式、运行时行为、外部集成、架构决策等不经常变化的事实 | `code-patterns.md`、`runtime-behavior.md`、`integrations.md`、`adr/` |
| `guidance/` | **操作手册/SOP**——"怎么做" | 本地启动指南、测试指南、调试排查、常见坑点等可执行的操作步骤 | `local-dev-setup.md`、`debugging-playbook.md`、`common-pitfalls.md` |
| `plans/` | **执行计划** | ExecPlan 工作流的生命周期目录 | `proposal/`、`active/`、`completed/`、`artifacts/` |
| `rules/` | **规则与不变量** | 架构不变量、工作原则等约束性规则 | `invariants.md`、`golden-principles.md` |
| `quality/` | **质量追踪** | 技术债日志、知识缺口清单 | `debt-log.md`、`knowledge-gaps.md` |

**关键区分**：
- `reference/` vs `guidance/`：reference 描述**事实**（"SSO 登录流程是这样的"），guidance 描述**操作**（"本地启动服务需要这几步"）。一份文档如果同时包含两者，按主要用途归类。
- `reference/` 的内容通常由深度调研或设计决策产生，相对稳定；`guidance/` 的内容随工具链和流程变化而更新。

**操作**：

## 步骤 1：创建目录结构

```
docs/
├── AGENTS.md          # 中层索引（本 Phase 创建）
├── reference/         # 稳定事实说明
│   └── adr/           # 架构决策记录（如需要）
├── guidance/          # 操作手册
├── plans/             # 执行计划
│   ├── proposal/
│   ├── active/
│   ├── completed/
│   └── artifacts/
├── rules/             # 规则与不变量
└── quality/           # 质量追踪
```

## 步骤 2：创建 `docs/AGENTS.md`（中层索引）

这是 `docs/` 目录的入口文档，说明每个子目录的定义、维护规则和导航。参考 `<skill_dir>/actions/init/phase-3-scaffold-docs/assets/templates/docs-agents.md` 的结构，但内容必须根据目标仓库的实际文档填充。它应包含：
- 阅读顺序（Agent / 人类）
- 文档分层表（从根 AGENTS.md 到 docs/ 各子目录的关系）
- 按任务导航表（"我想做 X → 去看 Y"）
- 当前目录结构的 tree 输出
- 维护规则（新增文档时放在哪里、怎么分类）

## 步骤 3：第一类文档——标准骨架（改进版）

Agent 应检查仓库的 README、package.json scripts、Makefile 等，提取素材并创建以下文档。

**guidance/ 操作手册**：
- **本地开发启动指南** (`guidance/local-dev-setup.md`)：如何从零开始运行项目。这是最重要的 guidance 文档。
- **测试指南** (`guidance/testing-guide.md`)：如何运行测试、覆盖率要求
- **发布指南** (`guidance/release-guide.md`)：版本管理和发布流程（如有 changesets/lerna 等）

**rules/ 规则模板**：
- 参考 `<skill_dir>/actions/init/phase-3-scaffold-docs/assets/templates/rules/invariants.md` 创建 `docs/rules/invariants.md`（invariants 生成标准见下方）
- 参考 `<skill_dir>/actions/init/phase-3-scaffold-docs/assets/templates/rules/golden-principles.md` 创建 `docs/rules/golden-principles.md`

**quality/ 质量追踪**：
- 参考 `<skill_dir>/actions/init/phase-3-scaffold-docs/assets/templates/quality/debt-log.md` 创建 `docs/quality/debt-log.md`

**模板改进要求**：

1. **生成指令注释替代 `_TODO_`**：骨架文档中信息不足的区域，使用 `<!-- HARNESS-GEN: ... -->` 注释代替 `_TODO_`，说明应从 Phase 1 的哪项分析产出中提取内容。例如：

   ```markdown
   <!-- HARNESS-GEN: 从 Phase 1 代码模式采样中提取本仓库的命名惯例，
        列出文件命名、函数命名、变量命名的实际规律。
        如果 Phase 1 未能确定，标记为 Knowledge Gap。 -->
   ```

2. **golden-principles 按技术栈适配**：
   - Go 项目：强调错误处理惯例（`if err != nil` 模式、error wrapping 策略）、interface 设计（小 interface、accept interfaces return structs）
   - 前端项目：强调组件隔离（单向数据流、props 边界）、状态管理（local vs global state 划分）
   - 全栈项目：覆盖前后端边界（API contract、共享类型、BFF 层职责）
   - Rust 项目：强调 ownership 惯例、error enum 设计、trait 边界
   - Python 项目：强调类型注解覆盖策略、import 组织、测试隔离

## 步骤 4：第二类文档——深度文档（基于 Phase 1 分析）

基于 Phase 1 的深度分析产出，**选择性**创建以下文档。并非全部必须生成——Agent 根据仓库特征决定。

| 文档 | 位置 | 生成条件 | 模板参考 |
|---|---|---|---|
| **代码模式指南** | `docs/reference/code-patterns.md` | **所有仓库必须生成**（任何仓库都有代码惯例） | `<skill_dir>/actions/init/phase-3-scaffold-docs/assets/templates/reference/code-patterns.md` |
| **运行时行为文档** | `docs/reference/runtime-behavior.md` | Phase 1 检测到异步/流式/事件驱动逻辑时生成 | `<skill_dir>/actions/init/phase-3-scaffold-docs/assets/templates/reference/runtime-behavior.md` |
| **外部集成关系** | `docs/reference/integrations.md` | Phase 1 检测到外部服务/API 集成时生成 | `<skill_dir>/actions/init/phase-3-scaffold-docs/assets/templates/reference/integrations.md` |
| **架构决策记录** | `docs/reference/adr/` | 用户在 Phase 1 访谈中提供了设计决策时生成 | `<skill_dir>/actions/init/phase-3-scaffold-docs/assets/templates/reference/adr/adr-template.md` |
| **调试排查手册** | `docs/guidance/debugging-playbook.md` | **所有仓库必须生成**（至少包含构建/测试排查） | `<skill_dir>/actions/init/phase-3-scaffold-docs/assets/templates/guidance/debugging-playbook.md` |
| **常见坑点** | `docs/guidance/common-pitfalls.md` | Phase 1 git 考古或用户访谈中发现坑点时生成 | `<skill_dir>/actions/init/phase-3-scaffold-docs/assets/templates/guidance/common-pitfalls.md` |

**内容要求**：深度文档的内容**必须来自 Phase 1 的分析产出**，而非模板填充。模板文件只提供结构参考，Agent 必须用仓库特定的信息填充每个章节。对于无法从分析中确定的内容，标记为 Knowledge Gap（在步骤 5 中记录）。

## 步骤 5：第三类文档——知识缺口清单

**必须创建** `docs/quality/knowledge-gaps.md`，参考 `<skill_dir>/actions/init/phase-3-scaffold-docs/assets/templates/quality/knowledge-gaps.md` 模板。

此文档记录 Phase 1 分析过程中识别到但无法完全确定的知识缺口。**必须包含至少 3 个具体缺口**，来源包括：

- Phase 1 代码模式采样中无法确定的惯例（如"错误处理存在两种模式，不确定哪种是团队规范"）
- Phase 1 外部集成点中无法确定的故障降级策略
- Phase 1 git 考古中发现的高频修改文件但无法确定原因
- 用户访谈中未能回答的问题
- 深度文档中标记为 `<!-- HARNESS-GEN: ... -->` 但无法从分析中填充的区域

每个缺口必须包含：ID、所属领域、关联文档、具体描述、建议填补方式、状态。

## 步骤 6：Invariants 生成标准

`docs/rules/invariants.md` 的生成必须满足以下标准（Agent 从 Phase 1 代码模式采样中提取）：

1. **每条 invariant 必须包含具体的代码路径或模块边界示例**：
   - 差："`core` 不应直接依赖 UI 框架"
   - 好："`packages/core/` 不应 import 来自 `react-dom` 或 `apps/web/` 的任何模块。所有 UI 交互通过 `src/view/` 层的 Provider 暴露，外部通过依赖注入获取具体实现"

2. **从 Phase 1 分析中提取 invariants**，寻找以下信号：
   - 模块间的 import 方向约束
   - 代码生成目录的边界（生成代码不应手动修改）
   - 状态管理的分层规则
   - 命名/文件组织的强制惯例
   - 错误处理的一致性要求

3. **Enforcement 列必须反映当前实际的强制方式**，而非理想状态。如果目前只靠 code review 保证，就写 `code review`，不要写 `lint rule` 或 `CI check`。如果完全没有强制手段，写 `none (convention only)`。

**汇报**：分三类说明——创建了哪些标准骨架文档、哪些深度文档（及为什么选择/跳过）、知识缺口清单中记录了多少个缺口。
