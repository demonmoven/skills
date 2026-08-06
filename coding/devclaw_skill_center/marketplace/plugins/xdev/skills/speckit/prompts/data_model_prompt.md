**⚠️ 本指令仅针对服务端（backend）进行技术设计，包括业务逻辑代码、IDL（Contracts）、Docker 部署等。不要涉及任何前端（frontend/web/UI）相关内容。**

---

# data_model 指令（数据模型设计）

> 根据 Spec 文档及前序分析产物，生成数据模型设计文档

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

## 输出

1. Data Model 文档：DATA_MODEL_DOC，文档名必须为 "data-model"

---

**重要规则**：
- 充分利用 ANALYZE_DOC、MINING_DOC 中的信息，避免盲目探索代码库造成上下文膨胀
- **必须严格遵循** CONSTITUTION_DOC 中的所有条款
- RESEARCH_DOC 是前序流程中的联网调研结果，请充分利用，避免重复调研
- **输出文档的最大标题级别为 `##`**（因为后续会与其他文档合并为完整的技术方案文档，`#` 级标题留给合并后的总标题）

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


### 步骤 2：生成 Data Model 文档

1. 向用户报告 MESSAGE 提示 "【当前状态】开始 >> 生成 Data Model 文档..."

结合当前上下文生成 DATA_MODEL_DOC，内容必须涉及**此次需求相关**的：

* **实体（Entity）**
  - 包含名称（Name）、属性（Attribute）等关键信息
* **关系（Relationship）**
  - 用 ER 图来表示
  - **注意**：**一定要**用 **Mermaid** 来画
* **状态转移图**（如果有）
  - **注意**：**一定要**用 **Mermaid** 来画
* **数据校验规则**（如果有）
* **重要**：一定要**着重标记出**现存(不改)/增/删/改
* 同理，也需要包含相应数据表的变更（SQL/GraphQL 等等）
* 文档末尾加一个变更总结的表格
* **重要**：除了上述内容外，**不要添加**其他任何章节和内容（如模型转化等）

**输出格式要求**：文档内容以 `##` 作为最大标题级别，例如：

```markdown
## 数据模型（Data Model）

### 实体定义
...

### ER 图
...

### 变更总结
...
```

将文档输出到 DATA_MODEL_DOC。

2. 向用户报告 MESSAGE 提示 "【当前状态】完成 >> 生成 Data Model 文档"
