# Tasks 文档：[需求名称:PhaseN]

> 需求的代码实现的任务列表。
> 实际生成时，每个 Phase 都有一个独立的 Tasks 文档，每个文档只包含该 Phase 相关的任务，即只会存在以下模板中的Phase1-4的一个。

## 前置依赖（Prerequisites）

1. Spec 文档：SPEC_DOC ，文件名为 "spec.md"
   * 包含 User Stories 等信息

2. Constitution：CONSTITUTION_DOC ，文件名为 "constitution.md"
   * 当前代码库上做需求的规约，所有的代码实现都必须严格遵循它

3. Data Model 文档：DATA_MODEL_DOC ，文件名为 "data-model.md"
   * 数据模型/存储层/约束规则等的设计

4. Contracts 文档：CONTRACTS_DOC ，文件名为 "contracts.md"
   * 对外暴露的 API 接口契约文档

5. Configuration 文档：CONFIGURATION_DOC ，文件名为 "configuration.md"
   * 系统/核心领域的配置设计

6. Integration 文档：INTEGRATION_DOC ，文件名为 "integration.md"
   * 系统内部对外的集成方案

7. Plan 文档：PLAN_DOC ，文件名为 "plan.md"
   * 技术实现方案

8. Research 文档：RESEARCH_DOC ，文件名为 "research.md"
   * 技术方案调研文档

9. Mining 文档：MINING_DOC ，文件名为 "mining.md"
   * 隐性需求挖掘文档

***

## Phase 1: Setup (Shared Infrastructure)

**目标**: [项目初始化与基础结构搭建]

* [ ] T001 按实现计划创建项目结构

* [ ] T002 使用 [language] 与 [framework] 初始化项目及其依赖

* [ ] T003 配置代码检查（lint）与格式化工具

***

## Phase 2: Foundational (Blocking Prerequisites) - [Title]

**目标**: [在任何 User Story Phases 开始之前，必须完成的核心基础设施建设]

* [ ] T004 搭建数据库 Schema 及迁移（Migration）框架

* [ ] T005 实现认证 / 授权框架

* [ ] T006 搭建 API 路由与中间件结构

* [ ] T007 创建所有 User Stories 依赖的 Models/Entities

* [ ] T008 配置统一的错误处理与日志基础设施

* [ ] T009 搭建环境配置管理机制

***

## Phase 3: User Story 1 - [Title] (Priority: P1) 🎯 MVP

**目标**: [该 User Story 所交付的核心能力的简要描述]

**独立验证方式**: [独立地验证该 User Story 的方式的简要描述]

* [ ] T012 [US1] 创建 [Entity1] 模型，路径：src/models/[entity1].py

* [ ] T013 [US1] 创建 [Entity2] 模型，路径：src/models/[entity2].py

* [ ] T014 [US1] 实现 [Service] Service，路径：src/services/[service].py (依赖 T012, T013)

* [ ] T015 [US1] 实现 [endpoint/feature] Endpoint/Feature，路径：src/[location]/[file].py

* [ ] T016 [US1] 添加校验逻辑与错误处理

* [ ] T017 [US1] 为相关操作添加日志打印

***

## Phase 4: User Story 2 - [Title] (Priority: P2)

**目标**: [该 User Story 所交付的核心能力的简要描述]

**独立验证方式**: [独立地验证该 User Story 的方式的简要描述]

* [ ] T020 [US2] 创建 [Entity1] 模型，路径：src/models/[entity1].py

* [ ] T021 [US2] 实现 [Service] Service，路径：src/services/[service].py (依赖 T012, T013)

* [ ] T022 [US2] 实现 [endpoint/feature] Endpoint/Feature，路径：src/[location]/[file].py

* [ ] T023 [US2] 集成 User Story 1 的相关组件（如果涉及）

***

## Phase 的实现策略（Implementation Strategy For Each Phase）

### 上下文管理（Context Management）

**重要**：为避免上下文快速膨胀，在对代码库进行实际搜索/探索/调研前，**务必先**

* 第一步：从以下内容中阅读、分析、提取当前 Phase 中任务相关(User Story、文件路径、关键描述等)的信息

  * 上下文以及这些文档 SPEC_DOC、CONSTITUTION_DOC、DATA_MODEL_DOC、CONTRACTS_DOC、CONFIGURATION_DOC、INTEGRATION_DOC、PLAN_DOC、RESEARCH_DOC、MINING_DOC

* 第二步：基于这些信息发掘执行任务所需的 Insights

* 第三步：如果以上 Insights 仍不足以支撑后续任务的实现，再进行自由探索