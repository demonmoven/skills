**⚠️ 本指令仅针对服务端（backend）进行技术设计，包括业务逻辑代码、IDL（Contracts）、Docker 部署等。不要涉及任何前端（frontend/web/UI）相关内容。**

---

# analyze 指令（代码库现状分析）

> 根据 Spec 文档、技术指导文档、Constitution，对代码库进行现状分析并生成 Analyze 文档

## 输入文档（documents preparation）

> **变量路径说明**：以下变量通过 system prompt 注入，值为绝对路径。

1. Spec 文档：SPEC_DOC
   - 对需求的详细描述，也是功能实现的目标，具有严格的结构化表达模式
2. 技术指导文档：TECH_GUIDANCE_DOC
   - 用户对技术方案制定的指导性说明，用于辅助整体技术方案的设计
3. Constitution：CONSTITUTION_DOC
   - 当前代码库上做需求的规约，所有的技术方案都必须**严格遵循**它
4. 目标代码仓库路径：TARGET_SRC_DIR
5. SubAgent 文档：
   - CODEBASE_LOCATOR_AGENT_DOC
   - CODEBASE_ANALYZER_AGENT_DOC
   - CODEBASE_PATTERN_FINDER_AGENT_DOC

## 输出

1. Analyze 文档：ANALYZE_DOC，文档名必须为 "analyze"

---

## 关键原则：在此阶段你唯一的任务是分析和解释代码库涉及当前需求部分的现状

- 除非用户明确要求，否则**不要**提出任何改进或变更建议
- 除非用户明确要求，否则**不要**进行任何根因分析
- 除非用户明确要求，否则**不要**规划任何未来增强功能
- **不要**批评实现方案或指出问题
- **不要**建议重构、优化或架构调整
- **仅需**描述现有内容、所在位置、运作方式以及组件间的交互关系
- 你的工作是为现有系统绘制技术地图/编写技术文档

---

## 执行流程

### 步骤 1：阅读输入文档

1. 向用户报告 MESSAGE 提示 "【当前状态】开始 >> 阅读输入文档..."

* 阅读 SPEC_DOC 理解需求范围
* 阅读 TECH_GUIDANCE_DOC 理解技术指导
* 阅读 CONSTITUTION_DOC 理解代码库规约

* **关键**：在生成任何子任务并委托给 SubAgent 之前，先在主上下文中亲自阅读这些文件
* 这确保你在分解分析任务之前拥有完整的上下文

### 步骤 2：分析并拆解问题

1. 向用户报告 MESSAGE 提示 "【当前状态】开始 >> 分析并拆解问题..."

* 将技术上下文中的未知部分拆解为可组合的分析领域
* 花时间深入思考底层模式、关联关系和架构含义
* 识别需要调查的具体组件、模式或概念
* 使用 `TodoWrite` 创建分析计划，跟踪所有子任务
* 考虑哪些目录、文件或架构模式与分析相关

### 步骤 3：生成并行 SubAgents 任务进行全面分析

1. 向用户报告 MESSAGE 提示 "【当前状态】开始 >> 分派 SubAgents 进行代码库分析..."

* 创建多个 Task SubAgents，并行分析不同方面

我们有 3 种专门的 SubAgent，分别读取对应的 Agent DOC 文件内容作为该 SubAgent 独立的 system prompt（即以该 DOC 为模板创建一个 Claude Code SubAgent）：

**代码库调研 SubAgents：**

* 使用 **CODEBASE_LOCATOR_AGENT_DOC** 创建 SubAgent 来定位文件和组件的**位置**
  - 先读取该文件内容，将其作为 SubAgent 的 system prompt
  - 告诉 SubAgent 你要定位什么内容，SubAgent 会自行搜索

* 使用 **CODEBASE_ANALYZER_AGENT_DOC** 创建 SubAgent 来理解特定代码的**运作方式**
  - 先读取该文件内容，将其作为 SubAgent 的 system prompt
  - 告诉 SubAgent 你要分析什么组件，SubAgent 会深入阅读代码并记录实现细节

* 使用 **CODEBASE_PATTERN_FINDER_AGENT_DOC** 创建 SubAgent 来查找现有模式的**示例**
  - 先读取该文件内容，将其作为 SubAgent 的 system prompt
  - 告诉 SubAgent 你要查找什么模式，SubAgent 会搜索并展示代码示例

**重要**：所有 SubAgents 都是文档记录者，而非评论家。它们只用客观地描述现有内容，不提出改进建议或指出问题。

使用这些 SubAgents 的要点：

* 首先使用 codebase-locator 找出存在哪些内容
* 然后在最有价值的发现上使用 codebase-analyzer，记录其运作方式
* 当搜索不同内容时，并行运行多个 SubAgent
* 每个 SubAgent 都清楚自己的职责——只需告诉它你在寻找什么
* 不要编写关于**如何**搜索的详细提示——SubAgents 已经知道方法
* 提醒 SubAgents 它们是在记录文档，而非评估或改进

### 步骤 4：等待所有 SubAgent 完成并综合分析结果

1. 向用户报告 MESSAGE 提示 "【当前状态】开始 >> 综合 SubAgent 分析结果..."

* 重要：在继续之前，**等待所有** SubAgent 任务完成
* 汇总所有 SubAgent 的结果
* 将实际代码库的发现作为主要事实来源
* 关联不同组件间的发现
* 包含具体的文件路径和行号作为参考
* 突出模式、关联和架构决策
* 用具体证据回答用户的特定问题

### 步骤 5：生成现状分析报告

1. 向用户报告 MESSAGE 提示 "【当前状态】开始 >> 生成现状分析报告..."

按照以下格式生成分析报告，输出到 ANALYZE_DOC：

```markdown
# Analyze 文档（需求相关代码库现状分析）：[需求名]

**Git 提交**：[分析时采用的 commit id]
**分支**：[分析时采用分支名称]
**仓库**：[仓库名称]

## 摘要

[对分析过程中发现的高层概述]

## 详细发现

### [组件/领域 1]

* 现有内容描述（[file.ext:行号](链接)）
* 与其他组件的关联方式
* 当前实现细节（不作评判）

### [组件/领域 2]

...

## 代码引用

* `path/to/file.go:123` - 该处代码的描述
* `another/file.go:45-67` - 该代码块的描述

## 架构文档

[代码库中发现的当前模式、约定和设计实现]

## 待解决问题

[需要进一步调研的领域]
```

2. 向用户报告 MESSAGE 提示 "【当前状态】完成 >> 代码库现状分析，并生成 Analyze 文档"
