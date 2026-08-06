# 任务拆分指令

> 根据 Spec、Constitution、技术方案文件（5 个独立文件）和 Tasks 模板，生成需求的 Implementation Tasks 文档

## 输入文档

> **变量路径说明**：以下变量通过 system prompt 注入，值为绝对路径。

1. Spec 文档：SPEC_DOC
   - 对需求的详细描述，也是功能实现的目标，具有严格的结构化表达模式
2. Constitution：CONSTITUTION_DOC
   - 当前代码库上做需求的规约，所有的代码实现都必须**严格遵循**它
3. Data Model 文档：DATA_MODEL_DOC
   - 数据模型/存储层/约束规则等的设计
4. Contracts 文档：CONTRACTS_DOC
   - 对外暴露的 API 接口契约文档
5. Configuration 文档：CONFIGURATION_DOC
   - 系统/核心领域的配置设计
6. Integration 文档：INTEGRATION_DOC
   - 系统内部对外的集成方案（RPC 外调、外部 HTTP 服务等）
7. Plan 文档：PLAN_DOC
   - 技术实现方案
8. Research 文档：RESEARCH_DOC
   - 技术方案调研文档
9. Mining 文档：MINING_DOC
   - 隐性需求挖掘文档（差异化版，仅独特发现）
10. Phase Tasks 模板：TASKS_TEMPLATE_DOC
   - Phase Tasks 文档的模板
9. Specs 目录：SPECS_DIR
   - 产出文件的目标目录

## 输出

1. 多个 Phase Tasks 文档：`{SPECS_DIR}/subtasks_N.md`，N 为 Phase 编号

---

## 执行流程

### 1. 提取关键信息

1. 向用户报告 MESSAGE提示 "【当前状态】开始 >> [步骤1] 提取关键信息..."

2. 读取、理解、分析以下文档：
   - SPEC_DOC：需求内容以及 User Stories
   - CONSTITUTION_DOC：代码库规约，任务拆分和实现都必须严格遵循
   - DATA_MODEL_DOC：数据模型设计
   - CONTRACTS_DOC：API 接口契约
   - CONFIGURATION_DOC：配置设计
   - INTEGRATION_DOC：系统集成设计
   - PLAN_DOC：技术实现方案
   - RESEARCH_DOC：技术方案调研
   - MINING_DOC：隐性需求挖掘

3. 向用户报告 MESSAGE提示 "【当前状态】完成 >> [步骤1] 提取关键信息"

### 2. 读取并分析 Phase Tasks 模板

1. 向用户报告 MESSAGE提示 "【当前状态】开始 >> [步骤2] 读取并分析 Phase Tasks 模板..."

2. 读取、理解、分析 TASKS_TEMPLATE_DOC 模板，提前思考一下需求填写的内容和目标

3. 向用户报告 MESSAGE提示 "【当前状态】完成 >> [步骤2] 读取并分析 Phase Tasks 模板"

### 3. 生成 Phase Tasks 文档

1. 向用户报告 MESSAGE提示 "【当前状态】开始 >> [步骤3] 生成 Phase Tasks 文档..."

2. 按照以下要求对 Phase Tasks 模板进行填写

   **注意**：模板中以下章节要保持原封不动，不许改/删，也不许移动位置
       - 前置依赖（Prerequisites）
       - Phase 的实现策略（Implementation Strategy For Each Phase）

   Phase 章节按照下面的指引进行填写：

   a. 按照 User Stories 组织任务
      - 按照 User Story 的优先级分 Phase，顺序组织
      - 确保每个 User Story 内的任务的完整性，并确保 User Story 可独立测试验证

   b. Phase-Task 拆解规则
      - 将 Tasks 拆解到各个 Phase 中
      - 根据模板提示，识别出 Setup 和 Foundational 类型的 Phases，确保遵循
        - 优先级：Setup > Foundational > User Stories
        - Setup Phase：**最多只有** 1 个，负责项目的初始化、框架/骨架搭建等
          - 有些需求是在已经初始化过的项目上进行时可以没有这一步 Phase
          - 内容必须为 `项目的初始化、框架/骨架搭建相关`，如不相关，则可没有这个 Phase
            - 该 Phase 不是必须的
        - Foundational Phases: 可以有多个，确保遵循
          - 依赖 Setup Phase(如有) 先完成
          - 按照优先级拆分 Phases：系统对外集成变更(Integration等)、数据库变更、配置变更、其他...
          - **注意**：像data-model中核心领域模型变更、应用和服务层的变更 **不属于** Foundational Phases，应该在相应的 User Stories Phases中实现
          - 拆分后的 Foundational Phase 标题例如为 `Phase 2: Foundational (Blocking Prerequisites) - [数据库变更]`
          - **每个Phase内的Task数量不大于10个，若真超过10个则考虑进一步拆分Phase或合并Task**
            - 注意每个Phase的内聚性
          - Foundational Phases根据情况判断，可能没有
            - 比如，某个需求直接改 User Story 相关的某个业务模块的相关业务逻辑，都不涉及 Foundational 功能
        - User Story Phases：确保遵循
          - 依赖 Foundational Phases(如果有) 先完成
          - 每个 User Story 一个 Phase (按优先级排序，从上到下P1/P2/P3/...的顺序排列)
            - **重要**：**绝对不能**合并多个 User Stories 到一个 Phase，**必须1个User Story对应1个Phase**
          - 全部都依赖 Foundational 的完成
          - User stories 按从上到下(P1 → P2 → P3)顺序排列
          - **每个Phase内的Task数量不大于10个，若真超过10个则考虑进一步拆分Phase或合并Task**
          - 注意每个 Phase 的内聚性，确保其 Tasks 和 Phases 都严格属于 SPEC_DOC 中定义的 User Story

   c. 对 Phase 的要求
      - 每个 Phase 都要包含目标描述
      - **注意** + **重要**：最后一个 User Story Phase 后 **不要添加** 编译验证/服务部署/集成测试的收尾Phase
        - 因为：之前每个Phase都会保证编译通过，另外编译验证/服务部署/集成测试有其他工程自动化流程来保证，**而不是你的职责**

   d. 对 Task 格式的要求
      - 严格 follow 后续的 **Task 格式要求** 章节

3. 生成 TASKS_DOC
   - 每个 Phase 对应一个独立的 Tasks 文档，文档路径为 `{SPECS_DIR}/subtasks_N.md`，N 为 Phase 序号
   - 不要将多个 Phase 的 Tasks 合并到一个文档中
   - **重要**：每个 `subtasks_N.md` 文档中，除了该 Phase 自身的任务外，还必须包含以下内容（按顺序）：
     1. **前置依赖（Prerequisites）** 章节完整内容（含标题）—— 从 TASKS_TEMPLATE_DOC 中原封不动复制
     2. 该 Phase 对应的 **目标** 和 **独立验证方式**
     3. 该 Phase 的 Tasks 列表
     4. **Phase 的实现策略（Implementation Strategy For Each Phase）** 章节完整内容（含标题）—— 从 TASKS_TEMPLATE_DOC 中原封不动复制

4. 向用户报告 MESSAGE提示 "【当前状态】完成 >> [步骤3] 生成 Phase Tasks 文档，{TASK_REPORT}"
   - 其中 {TASK_REPORT} 包含以下信息：
     1. Phase 数：总数，User Stories 数
     2. Task 数：总数，各 Phase 内 Task 数

---

## **Task 格式要求**：`[ID] [Story] 任务内容描述`

1. **复选框（Checkbox）**: 必须始终以 `- [ ]` 打头（Markdown 复选框）
2. **Task ID**: 按执行顺序使用连续编号：T001, T002, T003, ...
3. **[Story]标签**: **仅用于** User Story Phase 的任务，且是 **必需** 的
    - 格式: [US1], [US2], [US3], etc. (对应 SPEC_DOC 中的 User Stories)
    - Setup phase: **不允许** 有 [Story]标签
    - Foundational phase: **不允许** 有 [Story]标签
    - User Story phases: **必须** 有 [Story]标签
4. **任务内容描述**:
   - 包含精确的文件路径
       - **注意**：路径以代码库根目录为基准绝对路径
           - 例如：`src/main/java/com/bytedance/stone/controller/StoneController.java`
   - 如有需要，可以包含 Leverage 信息：
       - 该字段用于指出该任务需要引用或复用的**既有**资源，包括但不限于：
           1. 代码
           2. 标准、规范、设计说明、信息挖掘
           3. 配置文件
           4. 特定工具
           5. 上下文中相关的DOC：SPEC_DOC, CONSTITUTION_DOC, DATA_MODEL_DOC, CONTRACTS_DOC, CONFIGURATION_DOC, INTEGRATION_DOC, PLAN_DOC
              - 如果有 Leverage 到这些 DOC 信息，则 Leverage 字段必填
       - 引用/复用的资源**必须**含**精确的文件路径**
           - 若精确路径获取不到，则可以直接使用文件名（有可能存在于上下文中）
       - 引用/复用的资源**必须**注明是文件的**第几行到第几行**
         - 例如：`src/main/java/com/bytedance/stone/controller/StoneController.java:100行-120行`
   - 如有需要，包含 Restrictions 信息：
       - 该字段用于明确该任务必须遵守的约束条件（比如使用什么SKILLS之类的）
   - 举例：
   ```markdown
   - [ ] T012 [US1] 创建 User Model，路径：src/models/user.py
       - Leverage:
         1. src/main/java/com/bytedance/stone/controller/StoneController.java:100行-120行
         2. plan.md:10行-15行
         3. mining.md:20行-25行
         4. configuration.md:35行-66行
       - Restrictions: 该 Model 必须使用 SQLAlchemy 定义，且必须包含 `id`, `username`, `email`, `password` 四个字段
   ```
5. **示例**:

- ✅ CORRECT: `- [ ] T001 根据实现计划创建项目结构`
- ✅ CORRECT: `- [ ] T005 实现身份认证中间件，路径：src/middleware/auth.py`
- ✅ CORRECT: `- [ ] T012 [US1] 创建 User Model，路径：src/models/user.py`
- ✅ CORRECT: `- [ ] T014 [US1] 实现 UserService，路径：src/services/user_service.py`
- ❌ WRONG: `- [ ] 创建 User model` (Task ID 和 Story 标签缺失)
- ❌ WRONG: `T001 [US1] 创建 model` (复选框缺失)
- ❌ WRONG: `- [ ] [US1] 创建 User model` (Task ID 缺失)
- ❌ WRONG: `- [ ] T001 [US1] 创建 model` (精确路径缺失)

---

## Phase内任务的安排（Within Each User Story）

- 任务顺序安排一般情况下 follow 代码分层 or Constitution 等原则
    - 例如：
        - Data Model 先于 Service 先于 Endpoints
        - 核心实现先于系统集成
- 如果 Story 之间有依赖关系，则必须标记清楚（但一般不会出现这种情况，还是要尽量保持 Story 的独立性）
- 对于 Data Model：
  - 将每一个实体（entity）映射到其所服务的 User Story（一个或多个）
  - 如果某个实体服务于多个 User Stories，则
  - 将其 Tasks 放入较早的 User Story Phase，或放入 Setup Phase
  - 对于实体之间的关系（relationships），放在相应 User Story Phase 的 service 层相关任务中
- 对于 Contracts：
  - 将每一个 contract / API endpoint 映射到它所服务的 User Story 中
- 仅某个 User Story 所需的初始化配置 放入 该 User Story Phase

---

## 备注以及特别强调（Notes）

- [Story] 标签用于将任务映射到具体的 User Story，确保可追溯性
- User Story 都必须可以独立完成、独立测试
- **⚠️ 再次强调 Phase-Task 拆解规则中的关键约束**：
    - **Setup Phase 最多 1 个**，且仅限项目初始化/框架骨架搭建，不是必须的
    - **Foundational Phases** 可以有多个，依赖 Setup Phase(如有) 先完成。按照优先级拆分：系统对外集成变更(Integration等) > 数据库变更 > 配置变更 > 其他...；核心领域模型变更、应用和服务层的变更**不属于** Foundational，应放入 User Story Phases
    - **User Story Phases 必须 1 个 User Story 对应 1 个 Phase**，绝对不能合并多个 User Stories
    - **每个 Phase 内 Task 数量不超过 10 个**，超过则进一步拆分 Phase 或合并 Task
    - **最后一个 User Story Phase 后不要添加**编译验证/服务部署/集成测试的收尾 Phase
- **⚠️ 再次强调每个 subtasks_N.md 的完整性要求**：
    - 每个 `subtasks_N.md` 不能只有 Task 列表，**必须**同时包含从 TASKS_TEMPLATE_DOC 中原封不动复制的**前置依赖（Prerequisites）**章节和**Phase 的实现策略（Implementation Strategy For Each Phase）**章节
    - 这两个章节是下游逐阶段开发执行任务时的关键上下文，缺失会导致执行质量下降
- **避免以下情况**：
    - 任务描述含糊不清
    - 破坏 User Story 的独立性
