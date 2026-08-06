# Phase 1: 仓库深度理解（三轮分析）

> **本 phase 在 `/harness init` 流程中的位置**：第 1 步，必须主 Agent 执行（含用户交互）。产出 `harness-init-analysis.md` 作为后续所有 phase 的输入。

**目标**：不仅检测技术栈，更要深入理解仓库的代码模式、运行时行为、历史痛点和外部集成，为后续 Phase 提供丰富的仓库特定信息。

**执行本 Phase 时，必须参考 `<skill_dir>/actions/init/phase-1-analyze/prompts/deep-analysis.md` 了解每一步的详细方法和产出格式。**

## 1.1 自动深度分析

**1. 技术栈检测**：运行 `<skill_dir>/actions/init/phase-1-analyze/scripts/detect_stack.sh` 获取基础信息（JSON 输出）。

**2. 仓库浏览**（保留原有，不能跳过）：
- 阅读 README、现有文档，理解项目定位和业务场景
- 浏览目录结构，理解模块划分方式
- 检查 monorepo 结构、CI 配置、已有质量工具
- 查看 git log 了解提交惯例
- 已有 harness 组件检查：pre-commit、AGENTS.md、ARCHITECTURE.md、docs/plans/ 等
- 如果存在 `harness-init.progress`，读取并跳过已完成的 Phase

**3. 代码模式采样**（新增）：
- 从每个主要模块选取 2-3 个典型文件（入口、核心业务、测试）
- 分析并记录：命名惯例、错误处理模式、DI/注入方式、状态管理方式、异步模式
- 寻找重复出现的结构模式

**4. 模块依赖图**（新增）：
- 分析模块间的 import/引用关系
- 绘制依赖方向图（ASCII 或 Mermaid）
- 标注每个模块的角色：入口/核心业务/基础设施/工具

**5. Git 考古**（新增）：
- `git log --oneline -200`：近期开发方向
- 高频修改文件 top 10
- revert 记录和大型 merge
- 贡献者分布

**6. 外部集成点**（新增）：
- 搜索 HTTP client 初始化、环境变量引用、配置文件中的外部 URL/host、SDK 初始化

**7. 测试模式**（新增）：
- 分析测试目录结构和命名规范
- 识别 mock 策略和覆盖率配置

将所有分析结果汇总写入临时文件 `harness-init-analysis.md`（不入库），用于后续 Phase 参考。

## 1.2 结构化用户访谈

**基于第一轮分析结果，向用户提出 5-8 个关键问题。问题库和触发条件详见 `<skill_dir>/actions/init/phase-1-analyze/prompts/interview.md`。**

**必问问题**（3 个）：
1. 核心业务场景（用 1-2 段话描述）
2. 最常遇到的 bug 类型或开发痛点
3. 希望 AI 一定要知道的关键设计决策

**条件触发问题**（根据分析结果选择 2-5 个）：根据第一轮检测到的特征（DI 容器、多区域部署、SSE/WebSocket、高频 revert、代码生成、复杂状态管理、monorepo、外部服务集成），从问题库中选择对应的问题。

采用**一问一答**方式，每个问题提供多选项 + 自由输入。

## 1.3 验证与补问

生成"仓库理解摘要"（2-3 页），包含：
- 项目定位与核心业务场景
- 技术架构概述（引用依赖图）
- 关键设计决策清单
- 已识别的代码模式清单
- 已识别的风险区域和痛点
- 已识别的知识缺口（需要标记为后续 Knowledge Gap）

向用户展示摘要并逐节确认。用户纠正后，形成最终分析报告。

## 汇报内容

Phase 1 的汇报必须包含以上所有分析结果，以及基于分析的**适配方案**：
- 哪些 hook 需要安装、哪些不需要、为什么
- lint 规则需要做哪些调整
- 文件长度上限建议
- Phase 3 将生成哪些深度文档（基于分析结果确定）

**创建 `harness-init.progress` 文件**，记录检测结果和适配方案（格式见 `<skill_dir>/references/progress-spec.md`）。
