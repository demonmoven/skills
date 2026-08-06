**⚠️ 本指令仅针对服务端（backend）进行技术设计，包括业务逻辑代码、IDL（Contracts）、Docker 部署等。不要涉及任何前端（frontend/web/UI）相关内容。**

---

# research 指令（技术方案调研）

> 根据 Spec 文档、技术指导文档、Constitution，进行技术方案调研并生成 Research 文档

## 输入文档（documents preparation）

> **变量路径说明**：以下变量通过 system prompt 注入，值为绝对路径。

1. Spec 文档：SPEC_DOC
   - 对需求的详细描述，也是功能实现的目标，具有严格的结构化表达模式
2. 技术指导文档：TECH_GUIDANCE_DOC
   - 用户对技术方案制定的指导性说明，用于辅助整体技术方案的设计
3. Constitution：CONSTITUTION_DOC
   - 当前代码库上做需求的规约，所有的技术方案（包括后续环节的代码实现）都必须**严格遵循**它
4. 目标代码仓库路径：TARGET_SRC_DIR

## 输出

1. Research 文档：RESEARCH_DOC，文档名必须为 "research"

---

**重要规则**：
- 调研（以及决策）**必须严格遵循** CONSTITUTION_DOC 中的所有条款

---

## 执行流程

### 步骤 1：阅读输入文档

1. 向用户报告 MESSAGE 提示 "【当前状态】开始 >> 阅读输入文档..."

* 阅读 SPEC_DOC 理解需求范围
* 阅读 TECH_GUIDANCE_DOC 理解技术指导
* 阅读 CONSTITUTION_DOC 理解代码库规约


### 步骤 2：提取关键未知项（unknowns）

1. 向用户报告 MESSAGE 提示 "【当前状态】开始 >> 提取关键未知项..."

结合当前上下文，提取出技术方案需要解决的**关键未知项（unknowns）**。

未知项类型：
- **NEEDS CLARIFICATION**：需求中不明确的地方
- **Dependency**：技术依赖
- **Integration**：系统集成方案

### 步骤 3：分派 research agent 调研每个未知项

1. 向用户报告 MESSAGE 提示 "【当前状态】开始 >> 技术方案调研..."

对于每个 unknown，分派一个独立的 **research agent**（SubAgent）来执行调研任务。

不同类型 unknown 对应的调研指令不同：
- **NEEDS CLARIFICATION** 类型：给分派的 **research agent** 的任务指令为 "请结合上下文，对这个 {unknown} 进行调研"
- **Dependency** 类型：给分派的 **research agent** 的任务指令为 "请结合上下文，调研一下这个 {unknown} 在业界相关领域的最佳实践"
- **Integration** 类型：给分派的 **research agent** 的任务指令为 "请结合上下文，调研一下这个 {unknown} 相关业界最佳的集成方案"

多个 unknown 的调研可以**并行分派**多个 SubAgent 同时执行。

### 步骤 4：综合调研结果并生成 RESEARCH_DOC

1. 向用户报告 MESSAGE 提示 "【当前状态】开始 >> 综合调研结果，生成 Research 文档..."

等待所有 research agent 完成后，将调研结果整合输出到 RESEARCH_DOC，格式要求如下：

```markdown
# Research 文档：[需求名称]

---

## 调研项-1

- 内容：
  [调研项内容：对应 unknown 的描述]
- 类型：
  [NEEDS CLARIFICATION / Dependency / Integration]
- 技术决策：
  [决策结果描述]
- 决策理由：
  [决策理由描述]
- 其他可行选择调研：
  [其他可行选择调研内容，以及为什么没有比最终决策更合适]

---

## 调研项-2

- 内容：
  [调研项内容：对应 unknown 的描述]
- 类型：
  [NEEDS CLARIFICATION / Dependency / Integration]
- 技术决策：
  [决策结果描述]
- 决策理由：
  [决策理由描述]
- 其他可行选择调研：
  [其他可行选择调研内容，以及为什么没有比最终决策更合适]

---

[更多调研项列举]
```

2. 向用户报告 MESSAGE 提示 "【当前状态】完成 >> 技术方案调研，并生成 Research 文档"
