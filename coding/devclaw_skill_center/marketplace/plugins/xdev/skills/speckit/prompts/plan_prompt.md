**⚠️ 本指令仅针对服务端（backend）进行技术设计，包括业务逻辑代码、IDL（Contracts）、Docker 部署等。不要涉及任何前端（frontend/web/UI）相关内容。**

---

# plan 指令（技术实现方案）

> 根据所有前序文档产物，生成完整的技术实现方案

## 输入文档（documents preparation）

> **变量路径说明**：以下变量通过 system prompt 注入，值为绝对路径。

1. Spec 文档：SPEC_DOC
   - 对需求的详细描述，也是功能实现的目标，具有严格的结构化表达模式
2. 技术指导文档：TECH_GUIDANCE_DOC
   - 用户对技术方案制定的指导性说明，用于辅助整体技术方案的设计
3. Constitution：CONSTITUTION_DOC
   - 当前代码库上做需求的规约，所有的技术方案（包括后续环节的代码实现）都必须**严格遵循**它
4. 目标代码仓库路径：TARGET_SRC_DIR
5. 代码库现状分析文档：ANALYZE_DOC
   - 前序流程产物，已结合 SPEC_DOC 对当前代码库中需求相关代码进行了分析和探索
   - **重要**：充分利用其中的信息，避免盲目探索造成上下文膨胀
6. 技术方案调研文档：RESEARCH_DOC
   - 前序流程产物，基于 SPEC_DOC、TECH_GUIDANCE_DOC 充分调研后的结果
   - 请充分利用，避免重复调研
7. 隐性需求挖掘文档：MINING_DOC
   - 前序流程产物，通过分层代码分析挖掘出的隐藏技术需求
8. Contracts 文档：CONTRACTS_DOC
   - 前序流程产物，包含已生成的对外暴露的 API 接口契约文档
9. Data Model 文档：DATA_MODEL_DOC
   - 前序流程产物，包含数据模型/存储层/约束规则等的设计
10. Configuration 文档：CONFIGURATION_DOC
    - 前序流程产物，包含系统/核心领域的配置信息
11. Integration 文档：INTEGRATION_DOC
    - 前序流程产物，包含系统内部对外的集成方案（RPC 外调、外部 HTTP 服务等）

## 输出

1. Plan 文档：PLAN_DOC，文档名必须为 "plan"
   - 包含技术实现方案

---

**重要规则**：
- 充分利用 ANALYZE_DOC、MINING_DOC 中的信息，避免盲目探索代码库造成上下文膨胀
- **必须严格遵循** CONSTITUTION_DOC 中的所有条款
- RESEARCH_DOC 是前序流程中的联网调研结果，请充分利用，避免重复调研
- **忽略（不要在方案中提）**单元测试以及部署集成测试验证，只需要关注编译成功即可
  - 因为单元测试和部署集成测试有其他自动化流程来保证，不是你的责任！**只关注**技术实现即可
- **PLAN_DOC 使用 `#` 作为顶级标题**

---

## 执行流程

### 步骤 1：阅读输入文档

1. 向用户报告 MESSAGE 提示 "【当前状态】开始 >> 阅读输入文档..."

* 阅读 SPEC_DOC 理解需求范围
* 阅读 TECH_GUIDANCE_DOC 理解技术指导
* 阅读 CONSTITUTION_DOC 理解代码库规约
* 阅读 ANALYZE_DOC 理解代码库现状分析
* 阅读 RESEARCH_DOC 理解技术调研结论
* 阅读 MINING_DOC 理解隐性技术需求
* 阅读 CONTRACTS_DOC 理解 API 契约
* 阅读 DATA_MODEL_DOC 理解数据模型设计
* 阅读 CONFIGURATION_DOC 理解配置设计
* 阅读 INTEGRATION_DOC 理解系统集成设计


### 步骤 2：生成 Plan 文档（技术实现方案）

1. 向用户报告 MESSAGE 提示 "【当前状态】开始 >> 生成技术实现方案..."

从 SPEC_DOC 中提取**需求名称**（即 spec 文档的标题或核心需求描述），然后按以下结构输出到 PLAN_DOC：

```markdown
# 技术实现方案（Implementation Plan）：[需求名称]

## 概览（Overview）

[从 SPEC_DOC、RESEARCH_DOC、DATA_MODEL_DOC、CONTRACTS_DOC、CONFIGURATION_DOC、INTEGRATION_DOC、上下文中，精炼地提取出核心需求和核心技术实现思路]

## 需求相关代码结构（Spec Related Code Structure）

[实际需求相关的代码目录结构]

## 详细实现方案

[详细的技术实现方案]
```

**注意**：
* **忽略（不要在方案中提）**单元测试以及部署集成测试验证，只需要关注编译成功即可
  - 因为单元测试和部署集成测试有其他自动化流程来保证，不是你的责任！**只关注**技术实现即可

2. 向用户报告 MESSAGE 提示 "【当前状态】完成 >> 生成 Plan 文档"
