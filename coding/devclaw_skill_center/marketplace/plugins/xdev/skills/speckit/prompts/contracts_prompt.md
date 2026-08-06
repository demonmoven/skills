**⚠️ 本指令仅针对服务端（backend）进行技术设计，包括业务逻辑代码、IDL（Contracts）、Docker 部署等。不要涉及任何前端（frontend/web/UI）相关内容。**

---

# contracts 指令（API 接口契约设计）

> 根据 Spec 文档及前序分析产物，生成系统对外暴露的 API 接口契约文档

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
8. Contracts 撰写规范文档：CONTRACTS_STANDARD_DOC
   - 最佳实践参考文档，定义了 Contracts 文档的标准撰写格式
   - **重要**：生成的 Contracts 文档**必须严格遵循**此文档的格式规范

## 输出

1. Contracts 文档：CONTRACTS_DOC，文档名必须为 "contracts"

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
* 阅读 CONTRACTS_STANDARD_DOC 理解 Contracts 文档的标准撰写格式

### 步骤 2：生成 Contracts 文档

1. 向用户报告 MESSAGE 提示 "【当前状态】开始 >> 生成 Contracts 文档..."

结合当前上下文生成 CONTRACTS_DOC，**必须严格参照 CONTRACTS_STANDARD_DOC 的格式规范**。

内容必须涉及**此次需求相关**的：

* **API 接口（Endpoint）**
  - 只需要包含**本需求中系统对外暴露给前端的接口**，系统内部集成外部的接口**不要包含**（如：系统内部集成沙箱、数据库、外部的 RPC 等）
  - **注意**：**不要**给出 Mock 出入参
* 根据上下文指引，使用正确的 Pattern 来定义
  - 如 Thrift IDL / REST / GraphQL 等

**撰写格式规范**（参照 CONTRACTS_STANDARD_DOC）：

每个 API Endpoint 必须按以下格式撰写：

1. **元数据头**（每项单独一行）：
   - `**api path**`: API 路径（用反引号包裹）
   - `**方法**`: HTTP 方法（GET/POST/PUT/PATCH/DELETE）
   - `**文件位置**`: IDL 文件路径
   - `**变更类型**`: 标注**增**/**改**/**不变**/**不变(间接受到影响)**
   - `**对应用户操作**`: 对应的 User Story / 用户操作说明

2. **请求**：`**请求: <变更标记>**`
   - 变更标记：`新增`/`改`/`不变`/`不变(间接受到影响)`
   - 对于间接受到影响的，需标注依赖链（如：`- 间接依赖(A -> B -> C)有变更：新增字段 X`）
   - 附 Thrift IDL 代码块，新增字段用 `// 【新增】` 注释标注
   - 现有字段可用 `// ... 省略：现有字段 ...` 折叠

3. **响应**：`**响应: <变更标记>**`，格式同请求

4. **Service 定义**：`**Service 定义: <变更标记>**`
   - 附 Thrift IDL 代码块，标注 API 路由注解
   - 新增方法用 `// 【新增】` 注释标注

5. **备注**：补充说明（字段含义、业务规则等），可选

6. 各 Endpoint 之间用 `***` 分隔

**公共结构定义**按以下格式撰写：

- `**文件位置**` + `**变更类型**`
- 附 Thrift IDL 代码块，新增结构/字段用 `// 【新增】` 注释标注

将文档输出到 CONTRACTS_DOC。

2. 向用户报告 MESSAGE 提示 "【当前状态】完成 >> 生成 Contracts 文档"
