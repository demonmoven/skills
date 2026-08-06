# Tasks 文档：[需求名称:PhaseN]

> 前端需求的代码实现任务列表。
> 每个 Phase 对应 plan.md 中的一个实现阶段，有独立的 Tasks 文档。
> 实际生成时，只包含以下模板中某一种 Phase 类型的内容。

## 前置依赖（Prerequisites）

1. Spec 文档：SPEC_DOC ，文件名为 "spec.md" 或 "feature_spec.md"
   * 包含 User Stories、验收场景等信息

2. Plan 文档：PLAN_DOC ，文件名为 "plan.md"
   * 包含完整的技术实现方案（API 类型定义、组件拆分、User Story 集成等阶段划分）
   * **本文档的 Phase 划分直接来源于 plan.md 的实现阶段**

3. Constitution：CONSTITUTION_DOC ，文件名为 "constitution.md"
   * 当前代码库上做需求的规约，所有的代码实现都必须严格遵循它

4. Mining-Result 文档：MINING_RESULT_DOC ，文件名为 "mining-result.md"
   * 前端隐性技术需求分析报告，包含需求汇总表
   * 相关隐性需求已融合到本 Phase 的 Task 中

5. API-Doc 文档（如有）：API_DOC ，文件名为 "api-doc.md"
   * 后端 API 接口契约，包含接口定义、请求/响应格式、错误码等
   * 涉及 API 调用的 Task 需参考此文档

***

## Phase 类型 A: API 类型定义（对应 plan.md 阶段 1）

**目标**: [从 plan.md 阶段 1 提取的目标描述]

**来源追溯**: plan.md 阶段 1；mining-result 编号 [MR-xxx, MR-yyy]（如有相关）

**独立验证方式**: API Schema 更新成功，TypeScript 编译通过，所有 API 类型定义与 api-doc（如有）一致

* [ ] T001 执行 API Schema 更新命令，路径：frontend/packages/cozeloop/api-schema
    - Leverage:
      1. plan.md:阶段1-主要工作项1
      2. api-doc.md:全部接口定义（如有）
    - Restrictions: 使用 `npm run update` 自动生成，不手动修改 api-schema 文件

* [ ] T002 验证生成的 API 类型定义与 api-doc 对齐，路径：frontend/packages/cozeloop/api-schema/src
    - Leverage:
      1. plan.md:阶段1-验证点列表
      2. api-doc.md:请求/响应数据结构（如有）
    - Restrictions: 验证枚举值、必填/选填标记、嵌套结构完整性

* [ ] T003 确保 API Schema 包编译通过，路径：frontend/packages/cozeloop/api-schema
    - Leverage:
      1. plan.md:阶段1-编译验证
    - Restrictions: 运行 `rush ts-check -o @cozeloop/api-schema`，零 TypeScript 错误

***

## Phase 类型 B: 组件实现（对应 plan.md 阶段 2+，每个组件一个 Phase）

**目标**: [从 plan.md 对应组件阶段提取的目标描述]

**来源追溯**: plan.md 阶段 N；mining-result 编号 [MR-xxx]（如有相关）

**独立验证方式**: [组件渲染正常，Props 类型检查通过，预览页面可展示所有状态]

* [ ] T00x 创建 [ComponentName] 组件骨架与 Props 接口定义，路径：frontend/packages/cozeloop/[package]/src/[component-name]/index.tsx
    - Leverage:
      1. plan.md:阶段N-Props定义
      2. plan.md:阶段N-组件层级
    - Restrictions: 遵循三层组件库架构，单组件 < 400 行

* [ ] T00x 实现 [ComponentName] 核心交互逻辑，路径：frontend/packages/cozeloop/[package]/src/[component-name]/index.tsx
    - Leverage:
      1. plan.md:阶段N-主要工作项
      2. mining-result.md:编号MR-xxx（组件状态流转相关隐性需求）
    - Restrictions: 复用 @coze-arch/coze-design 基础组件，具名导入 React

* [ ] T00x 实现 [ComponentName] 样式，路径：frontend/packages/cozeloop/[package]/src/[component-name]/style.less
    - Leverage:
      1. plan.md:阶段N-组件状态（默认/加载/错误/禁用）
    - Restrictions: 使用 Tailwind 或 Less module，避免行内 style

* [ ] T00x 将 [ComponentName] 集成到预览页面，路径：frontend/packages/cozeloop/components-preview-page/src/[feature-name]/
    - Leverage:
      1. plan.md:阶段N-验收场景
    - Restrictions: 展示所有组件状态变体（默认、加载、错误、禁用、不同数据场景）

***

## Phase 类型 C: User Story 集成（对应 plan.md 阶段 N+，每个 User Story 一个 Phase）

**目标**: [从 plan.md 对应 User Story 阶段提取的目标描述]

**来源追溯**: plan.md 阶段 M；mining-result 编号 [MR-xxx, MR-yyy]（如有相关）；api-doc 接口 [接口名称]（如有相关）

**独立验证方式**: [从 plan.md 的验收场景转化，描述独立验证方式]

* [ ] T00x [USn] 创建页面组件与路由配置，路径：frontend/apps/cozeloop/src/pages/[PageName]/index.tsx
    - Leverage:
      1. plan.md:阶段M-涉及的页面/路由
      2. spec.md:User Story N 描述
    - Restrictions: 路由配置遵循现有 routes 目录结构

* [ ] T00x [USn] 集成阶段 2+ 已实现的组件到页面，路径：frontend/apps/cozeloop/src/pages/[PageName]/index.tsx
    - Leverage:
      1. plan.md:阶段M-组件集成（引用前序阶段组件名称和 Props）
      2. mining-result.md:编号MR-xxx（组件复用影响相关隐性需求）
    - Restrictions: 必须使用前序阶段定义的 Props 接口，不重复定义

* [ ] T00x [USn] 实现页面状态管理，路径：frontend/apps/cozeloop/src/pages/[PageName]/store.ts
    - Leverage:
      1. plan.md:阶段M-状态管理
      2. mining-result.md:编号MR-xxx（页面生命周期/状态流转相关隐性需求）
    - Restrictions: 使用 Zustand/useState，状态更新时机与 plan 一致

* [ ] T00x [USn] 实现 API 调用与数据处理，路径：frontend/apps/cozeloop/src/pages/[PageName]/hooks.ts
    - Leverage:
      1. plan.md:阶段M-API调用处理
      2. api-doc.md:对应接口定义（如有）
      3. mining-result.md:编号MR-xxx（数据获取与缓存相关隐性需求）
    - Restrictions: 使用 @cozeloop/api-schema 中阶段 1 定义的类型，参数/返回值/错误码与 api-doc 一致

* [ ] T00x [USn] 实现用户交互流程与错误处理，路径：frontend/apps/cozeloop/src/pages/[PageName]/index.tsx
    - Leverage:
      1. plan.md:阶段M-用户交互流程
      2. mining-result.md:编号MR-xxx（用户交互动线相关隐性需求）
      3. api-doc.md:错误码映射（如有）
    - Restrictions: 覆盖成功/失败/加载状态，防抖节流按 mining-result 建议处理

***

## Phase 的实现策略（Implementation Strategy For Each Phase）

### 上下文管理（Context Management）

**重要**：为避免上下文快速膨胀，在对代码库进行实际搜索/探索/调研前，**务必先**

* 第一步：从以下内容中阅读、分析、提取当前 Phase 中任务相关(User Story、文件路径、关键描述等)的信息

  * 上下文以及这些文档 SPEC_DOC、PLAN_DOC、CONSTITUTION_DOC、MINING_RESULT_DOC、API_DOC（如有）

* 第二步：基于这些信息发掘执行任务所需的 Insights

* 第三步：如果以上 Insights 仍不足以支撑后续任务的实现，再进行自由探索

### 逐任务提交（Commit Per Task）

**重要**：每完成一个 Task 的代码实现后，**必须立即执行 git commit**，确保每个 Task 有独立的提交记录。

* **commit message 格式**：`feat({feature}): {TaskID} {task 简述}`
  * `{feature}`：从 Phase 标题或 plan.md 中提取的需求/功能名称（kebab-case）
  * `{TaskID}`：当前 task 编号，如 `T001`、`T002`
  * `{task 简述}`：当前 task 的核心描述，简明扼要（英文）
* **仅 `git add` + `git commit`**，不执行 `git push`
* **无代码变更的 task**（如纯验证类）跳过 commit，标记完成时注明「无代码变更，跳过 commit」
