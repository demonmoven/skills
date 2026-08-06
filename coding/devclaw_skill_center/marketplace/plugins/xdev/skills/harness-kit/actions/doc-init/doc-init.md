> **Action: `doc-init`** — 由 `/xdev:harness-kit doc-init` 路由调用。
> 原 skill: `harness-doc-init` (author: guoshuai.030, version: 1.4)

# Harness Doc Init — 仓库基线文档体系建立

## 核心理念

> **给 Agent 一张地图，别给它一本千页大辞典。**
> — OpenAI Harness Engineering

本技能聚焦 Harness Engineering 的**上下文工程**支柱——为仓库建立分层文档体系：

| 产出 | 说明 |
|------|------|
| **分层 AGENTS.md** | 根入口 + 模块级，Agent 任务路由与约束 |
| **ARCHITECTURE.md** | 代码地图、模块边界、不变量 |
| **docs/ 索引** | 中层导航 + 专题文档分类 |
| **hook 建议**（附带） | 基于技术栈的 pre-commit 配置建议，仅建议不安装 |

**不做的事**：不写业务代码、不修改已有测试、不安装 hooks、不覆盖用户手动维护的文档内容。

## 激活后必须先读

- [references/DOC-STANDARD.md](references/DOC-STANDARD.md) — 文档命名、分层原则、写作规范
- [references/TEMPLATES.md](references/TEMPLATES.md) — 各类文档的完整模板
- 遇到 hook 设置时，读 [references/HOOK-CATALOG.md](references/HOOK-CATALOG.md)

## 适用场景

- 新建仓库，需要从零建立文档体系
- 已有仓库，但缺少系统化的 Agent 导航文档
- 已有仓库，需要补齐质量护栏（hooks、lint、文件长度检查）
- 想把现有零散文档收敛为 harness 风格的分层体系

## 完整流程

### Phase 1: 仓库侦察

在写任何文档前，必须先建立对仓库的结构认知。

**1.1 检测技术栈**

扫描根目录特征文件：

| 特征文件 | 技术栈 |
|---------|-------|
| `go.mod` / `go.work` | Go |
| `package.json` / `pnpm-workspace.yaml` | Node/TypeScript |
| `pyproject.toml` / `setup.py` / `requirements.txt` | Python |
| `Cargo.toml` / `Cargo.lock` | Rust |
| `pom.xml` / `build.gradle` | Java/Kotlin |
| 以上多种共存 | 混合技术栈 |

**1.2 检测项目结构**

| 结构类型 | 判断依据 |
|---------|---------|
| 单模块 | 只有一个 go.mod / package.json，无子项目 |
| 多模块 | 2-4 个独立模块目录，各有独立构建 |
| Monorepo | go.work / pnpm-workspace / rush.json / nx.json，5+ 包 |

**1.3 盘点已有 harness 资产**

逐项检查并记录：
- [ ] 已有 AGENTS.md？在哪些目录？内容是否过时？
- [ ] 已有 ARCHITECTURE.md？
- [ ] 已有 README.md？
- [ ] 已有 docs/ 目录？子结构？
- [ ] 已有 .pre-commit-config.yaml / .husky / lefthook.yml？
- [ ] 已有 lint 配置？（.golangci.yml / biome.json / .eslintrc / ruff.toml / clippy.toml）
- [ ] 已有 CI 配置？（.github/workflows / .gitlab-ci.yml / Jenkinsfile / Buildkite / CircleCI 等）
- [ ] 已有测试？框架是什么？有没有统一入口？
- [ ] 已有 Makefile / Taskfile.yml / justfile？
- [ ] 已有设计文档？（ADR / RFC / design-docs / specs / exec-plans 等知识沉淀）

**1.4 识别模块边界**

列出所有主要模块/包，记录各自的：
- 目录路径
- 职责（一句话）
- 入口文件
- 构建/测试命令
- 模块间依赖方向

### Phase 2: Harness 成熟度速评

基于 Phase 1 的证据，对六个维度做快速评估（1-5 分）：

| 维度 | 检查要点 |
|------|---------|
| 文档体系 | 有没有导航入口？是否分层？是否过时？ |
| 架构约束 | 有没有分层规则？依赖方向是否被管控？ |
| 测试体系 | 有没有统一测试入口？覆盖率？ |
| 静态检查 | lint 是否进入了自动化流程？ |
| Hook 护栏 | 本地提交前拦截了什么？ |
| 可观测性 | 日志/trace 是否结构化？Agent 能否查询？ |

输出一个速评表格，向用户展示当前状态和目标状态的差距。

### Phase 3: 文档规划

基于分析结果，决定文档层级和生成清单。

**自适应分层原则**（详见 [references/DOC-STANDARD.md](references/DOC-STANDARD.md)）：

| 项目复杂度 | 推荐层数 | 文档集 |
|-----------|---------|-------|
| 小型（单模块，<5 目录） | 1-2 层 | 根 AGENTS.md + 可选 ARCHITECTURE.md |
| 中型（2-4 模块） | 2-3 层 | + 模块级 AGENTS.md + 可选 docs/AGENTS.md |
| 大型（5+ 模块 + 丰富文档） | 3-4 层 | + docs/AGENTS.md + docs/reference/ + docs/guidance/ |
| 特大型（monorepo、多团队） | 4-5 层 | + 子模块 AGENTS.md + .skills/ |

**向用户展示规划表并征求确认**，包括：
- 将生成的文档列表
- 每个文档的覆盖范围
- 已有文档的处理策略（保留/更新/替换）
- 建议的 hook 配置

### Phase 4: 文档生成

严格按 [references/TEMPLATES.md](references/TEMPLATES.md) 中的模板生成。

**生成顺序（重要）**：

1. **先生成模块级 AGENTS.md** — 每个主要模块一个
2. **生成 ARCHITECTURE.md** — 从代码分析中提取结构、边界、不变量
3. **生成 docs/AGENTS.md** — 如果有 docs/ 目录，自动聚合已有文档列表
4. **最后生成根 AGENTS.md** — 汇总导航表，引用上面所有文档

这个顺序确保上层文档引用下层文档时，引用目标已经存在。

**生成规则**：

- 每个文件路径、命令、类型名都必须从 Phase 1 的真实分析中提取
- 不编造不存在的目录、文件或命令
- 模块级 AGENTS.md 不重复根级约束，只引用
- 知识导航表覆盖开发者 80% 的常见任务
- 核心约束必须具体、可执行，不是空泛的"写好代码"

**已有文档的处理**：

- 已有且内容准确的：保留，仅补充缺失的导航或约束章节
- 已有但明显过时的：标记过时部分，建议更新方案，征求用户确认后更新
- 不存在的：按模板新建

### Phase 5: 质量护栏建议

基于检测到的技术栈，建议 pre-commit hook 配置。

详细 hook 目录见 [references/HOOK-CATALOG.md](references/HOOK-CATALOG.md)，分为三个层级：

| 层级 | 内容 | 安装优先级 |
|------|------|----------|
| L0 通用卫生 | trailing-whitespace、end-of-file、check-yaml/json、大文件拦截、merge-conflict、secrets | 必装 |
| L1 语言质量 | go fmt/build/lint、tsc/biome、ruff/mypy、cargo fmt/clippy | 按技术栈装 |
| L2 AI 专项守护 | 占位符/幻觉标记检测、文件行数上限、函数长度上限 | 推荐装 |

**只建议，不自动安装**。生成配置文件内容后，向用户展示并征求确认。

### Phase 6: 验证与自检

生成完成后，执行以下检查：

**文档一致性检查**：
- [ ] 根 AGENTS.md 的知识导航表中，每个文件路径都实际存在
- [ ] 模块 AGENTS.md 中列出的目录结构与实际代码一致
- [ ] ARCHITECTURE.md 中提到的模块和文件路径都实际存在
- [ ] docs/AGENTS.md 的文件树与实际 docs/ 目录一致
- [ ] 常用命令中列出的命令可以在对应目录执行

**结构完整性检查**：
- [ ] 从根 AGENTS.md 出发，可以导航到所有重要模块
- [ ] 每份文档都有"知识导航"或"进一步阅读"章节
- [ ] 核心约束是从代码分析中提取的，不是空泛的

**不存在死链**：
- [ ] 所有交叉引用的相对路径指向真实文件

如果发现不一致，立即修正。

## Gotchas

- **不要从其他仓库复制文档内容**。每个仓库的结构不同，文档必须从本仓库的代码分析中生成
- **AGENTS.md 不是 README.md**。AGENTS.md 面向 Agent，注重任务路由和约束；README.md 面向人类，注重项目介绍和快速开始
- **模块入口统一用 AGENTS.md**。子目录默认不新增 README.md，除非该模块是独立发布的包（npm/crate/PyPI）需要 README 作为发布元数据
- **ARCHITECTURE.md 只负责代码地图和不变量**，不承担文档导航职责
- **模块级文档不重复全局约束**，只引用根 AGENTS.md
- **知识导航表是文档体系中最重要的部分**，它决定了 Agent 能否快速找到正确的信息
- **约束要具体到可机械执行**。"禁止在业务逻辑中滥用 panic" 好于 "写好错误处理"
- **不要把所有东西堆进一个文件**。根 AGENTS.md 目标 ~100 行，超过 200 行说明需要下沉
- **已有的 .pre-commit-config.yaml 不要覆盖**，应该在现有基础上补充缺失的检查项
- **docs/ 下的文档按类型分目录**：reference/（事实说明）、guidance/（操作手册）、plans/ 或 adr/（设计决策）
- **如果仓库已有文档站框架**（Docusaurus / MkDocs / mdBook），AGENTS.md 体系与其共存——AGENTS.md 面向 Agent 导航，文档站面向人类浏览，互不替代
- **小型单文件项目不需要 ARCHITECTURE.md**。只有当项目有 2+ 个需要解释边界的模块时才生成
