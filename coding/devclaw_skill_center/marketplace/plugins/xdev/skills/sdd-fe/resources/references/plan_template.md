# Plan Document Template

````markdown
## 概要

<!-- 从功能规格文档中提取：主要需求 + 技术研究的技术方案，指定前端相关内容 -->

[从需求中提取的前端功能高层描述及其技术方案]

## 技术背景

<!--
  需要操作: 用项目的技术细节替换本节内容。
  此处的结构以咨询形式呈现，用于指导迭代过程。
-->

**语言/版本**: React 18.2.0 + TypeScript 5.8.2\
**主要依赖**: @coze-arch/coze-design, @cozeloop/components, @cozeloop/biz-components-adapter, Zustand ^4.4.7, ahooks ^3.7.8\
**构建工具**: rsbuild (仅 App 应用入口项目需要构建)\
**测试框架**: Vitest (组件库/页面包通常不需要单测)\
**目标平台**: Web 浏览器 (Rush.js Monorepo)\
**项目类型**: 前端 Web 应用\
**性能目标**: 符合 Core Web Vitals，优化打包体积\
**约束条件**: 单个组件 <400 行, 遵循三层组件库架构, 禁止直接 fetch URL\
**规模/范围**: [特定领域，例如：1万用户，50个页面 或 需要澄清]

## 规范检查

*门禁: 必须在第0阶段研究之前通过。第1阶段设计之后需重新检查。*

<!-- 说明设计如何遵循已记录的技术模式和标准 -->

**前端规范合规性** (基于 `frontend-constitution.md`):

* [ ] **组件库层次结构**: 遵循三层架构 (@coze-arch/coze-design → @cozeloop/components → @cozeloop/biz-components-adapter)
* [ ] **API 规范**: 使用 @cozeloop/api-schema，禁止直接 fetch URL
* [ ] **React 导入规范**: 使用具名导入，禁止 `import React from 'react'`
* [ ] **组件规模控制**: 单个组件控制在 400 行以内
* [ ] **命名规范**: 包名使用 @cozeloop 前缀，文件名使用小写英文+中划线
* [ ] **技术栈合规**: React 18.2.0, TypeScript 5.8.2, Zustand, ahooks
* [ ] **样式规范**: 使用 Tailwind 或 Less module，避免行内 style

**以下情况需要复杂性说明**:

* 添加三层组件库架构之外的组件库
* 直接 fetch URL 而不使用 @cozeloop/api-schema
* 创建新包而不是扩展现有包
* 组件超过 400 行而不进行拆分
* 使用 `import React from 'react'` 默认导入
* 绕过既定的测试要求

## 技术分析

<!--
  阶段0填充区域：记录所有技术决策和研究发现
  每个决策应包含：决策内容、理由、考虑过的替代方案
-->

[详细的技术分析，包括架构设计、技术选型和实现方案]

### 相关 API 和 IDL

<!-- 相关的 API 接口和数据结构 -->

* **[API 名称]**: [用途和接口描述]
* **[数据格式]**: [请求/响应数据结构]

```typescript
// 来自 @cozeloop/api-schema 的 API 类型定义示例
interface FeatureAPI {
  // 定义相关 API 接口类型
}
```

### 需要修改的包

<!-- 需要修改的包和模块 -->

* **@cozeloop/components**: [添加功能相关的业务基础组件]
* **@cozeloop/biz-components-adapter**: [添加功能特定的业务组件]
* **@cozeloop/api-schema**: [通过 `npm run update` 更新 API schema - 不要直接修改文件]
* **frontend/apps/cozeloop**: [在主应用中添加新页面和路由]

### 需要修改的组件

<!-- 要添加/修改的业务组件。必填：列出组件的 props，需要复用哪些组件 -->

#### 基础组件层 (@coze-arch/coze-design)

* **复用组件**: [列出要复用的现有基础组件]

#### 业务基础组件层 (@cozeloop/components)

* **新增组件**: [列出要创建的新业务基础组件]
* **修改组件**: [列出要修改的现有组件]

#### 业务特性组件层 (@cozeloop/biz-components-adapter)

* **新增组件**: [列出要创建的新特性组件]
* **组件规模**: [确保每个组件 < 400 行]

#### 组件预览页面

<!-- 将功能相关的业务组件添加到预览页面 -->

* **预览页面**: 将组件集成到 `@cozeloop/components-preview-page` 包中，扩展 preview page 组件列表
* **规则**: 在实现前先阅读一下 `@cozeloop/components-preview-page` 包的 README，了解如何集成组件
* **[组件名称]**: [组件描述]
  * **Props**: [组件 props 及其类型列表]
  * **用法**: [在预览页面中使用组件的示例]
  * **预览**: [需要在预览页面中展示该组件，列举所有可能的组件状态]

## 组件分析

<!--
  阶段2填充区域：记录所有组件分析和拆分决策
  每个决策应包含：组件职责、层级划分、拆分理由
-->

### UI 参考资料（如果存在）

* **设计稿来源**: [Figma 链接 / 设计截图路径]
* **UI 分析结论**: [从 ui_analysis 中提取的组件列表]
* **UI 参考实现**: [从 plan 中提取的页面实现位置]

### 页面和组件入口

**页面维度**:

* [页面1路径]: [页面描述]
* [页面2路径]: [页面描述]

**组件维度**:

```text
[页面1]: /path/to/page1
  - [ComponentA] (业务基础组件)
  - [ComponentB] (业务特性组件)
    - [ComponentC] (业务基础组件，被 B 复用)
```

### 页面间动线关系

```text
[起始页面]
  → [触发操作]
  → [目标页面] (传递参数: [参数列表])
```

### 组件预览页面规划

**需求维度目录**: `@cozeloop/components-preview-page/src/[feature-name]/`

**组件状态展示要求**:

* 默认状态
* 加载状态
* 错误状态
* 禁用状态
* 不同数据场景

## User Story 概述

<!--
  阶段3填充区域：记录 User Story 分析结果
-->

- **总 User Story 数**: [N]
- **阶段编号范围**: 阶段 [M] ~ 阶段 [M+N-1]
- **前置依赖**:
  - 阶段 1: API 类型定义
  - 阶段 2+: 组件实现

### User Story 列表

[从 spec.md 提取的 User Story 清单，每个 User Story 一行]

## 项目结构

### 文档 (本功能)

```text
specs/[###-feature]/
├── plan.md              # 本文件 (/speckit.plan 命令输出)
└── tasks.md             # 计划任务 (/speckit.tasks 命令 - 不是由 /speckit.plan 创建)
```

### 源代码 (Rush.js Monorepo 结构)

```text
frontend/
├── apps/
│   └── cozeloop/            # 主应用入口
│       ├── src/
│       │   ├── pages/       # 新功能页面
│       │   └── routes/      # 路由配置
│       └── package.json
├── packages/
│   ├── cozeloop/
│   │   ├── components/      # 业务基础组件
│   │   │   └── [feature-name]/
│   │   ├── biz-components-adapter/  # 功能特定组件
│   │   │   └── [feature-name]/
│   │   └── api-schema/      # API 类型定义 (自动生成)
│   └── arch/
│       └── coze-design/     # 基础 UI 组件 (仅复用)
└── config/                  # 共享配置
    ├── eslint-config/
    ├── ts-config/
    └── vitest-config/
```

**结构决策**: 采用 Rush.js Monorepo 和三层组件库架构

## 实现阶段

<!--
  ⚠️ 关键规则：阶段编号必须从 1 开始 ⚠️

  每个阶段都可测试，应包含：
  1. 阶段目标：简短描述本阶段要完成什么
  2. 主要工作：详细的任务清单
  3. 验收场景：Given-When-Then 格式的测试场景

  阶段编号规则（严格遵守）：
  ✅ 正确：从"阶段 1"开始，使用连续递增的纯数字（1, 2, 3, 4, 5...）
  ❌ 错误：使用"阶段 0"、"阶段 3A"、"阶段 4.1"、"阶段 6a"等格式

  阶段顺序（固定）：
  - 阶段 1: API 类型定义
  - 阶段 2+: 组件实现（每个组件一个阶段）
  - 阶段 N+: User Story 集成（每个 User Story 一个阶段）
-->

### 阶段 1: API 类型定义

<!-- 更新 API Schema 并验证 -->

**阶段目标**: 更新 API Schema 并验证生成的类型定义符合规格要求

**主要工作**:

1. **确认 API Schema 更新**

   ```bash
   cd frontend/packages/cozeloop/api-schema
   npm run update
   ```

2. **验证生成的类型定义**

   需要验证的 API 类型（从 api-doc 中提取）:

   | API 名称 | 请求类型 | 响应类型 | 验证点 |
   |---------|---------|---------|--------|
   | [API 1] | [RequestType] | [ResponseType] | [关键字段验证] |
   | [API 2] | [RequestType] | [ResponseType] | [关键字段验证] |

   **关键验证点**:
   - [ ] 所有 API 的 Request/Response 类型已正确生成
   - [ ] 枚举类型值正确（如 `EvaluatorType.Code = 2`）
   - [ ] 必填字段标记正确（`field?` vs `field`）
   - [ ] 嵌套对象结构完整

3. **确保编译通过**
   - 运行 `rush ts-check -o @cozeloop/api-schema`
   - 验证无 TypeScript 错误

### 阶段 2+: 组件实现阶段

<!--
  重要：每个组件一个阶段，从阶段 2 开始递增编号
  编号必须是连续递增的纯数字
-->

### 阶段 N: [组件名称] 组件实现

**组件描述**: [详细描述该组件的功能和职责]

**组件层级**: [业务基础组件 / 业务特性组件]

**所属包**: [具体包路径，如 `@cozeloop/components` 或 `@cozeloop/biz-components-adapter`]

**主要工作**:

- 复用基础 UI 组件 (来自 @coze-arch/coze-design)
  - [列出具体要复用的组件，如 Button, Input, Modal 等]
- 实现组件核心功能
  - 添加到包: [具体包路径]
  - Props 定义: [列出完整的 Props 接口]
  - 状态管理: [如需要，说明状态管理方案，如 useState, useReducer 等]
  - 事件处理: [列出主要的事件处理函数]

- 集成到 `@cozeloop/components-preview-page` 包
  - 需求维度目录: [如 `evaluator/`]
  - 需要展示的状态: [列出所有组件状态变体]

**Props 定义**:

```typescript
interface [ComponentName]Props {
  // 详细的 props 定义，包含类型、是否必填、默认值、说明
  prop1: string;                    // 说明
  prop2?: number;                   // 可选属性说明
  onEvent?: (data: Type) => void;   // 事件回调说明
}
```

**复用的基础组件**:

* `Button` (来自 @coze-arch/coze-design): [使用场景]

* `Input` (来自 @coze-arch/coze-design): [使用场景]

* `Modal` (来自 @coze-arch/coze-design): [使用场景]

**组件状态**:

* 默认状态: [描述]

* 加载状态: [描述]

* 错误状态: [描述]

* 禁用状态: [描述]

**验收场景**:

1. **Given** [前置条件], **When** [触发动作], **Then** [预期结果]
2. **Given** [前置条件], **When** [触发动作], **Then** [预期结果]
3. **Given** [前置条件], **When** [触发动作], **Then** [预期结果]

### 阶段 N+: User Story 集成阶段

<!--
  重要：每个 User Story 一个阶段，从组件阶段之后开始递增编号
  编号必须是连续递增的纯数字
  必须引用前序阶段定义的 API 类型、组件 Props，不允许重复定义
-->

### 阶段 M: [User Story 标题] 集成

**User Story**: [从 spec 中复制完整的 user story 描述]

**涉及的页面/路由**:

* 入口页面: [页面路径，如 `/workspace/evaluator`]

* 相关页面: [列出所有相关页面]

**主要工作**:

#### 1. 组件集成 (如有则必填)

> **关键依赖**: 必须引用阶段 2+ 中已定义的组件，包括：
> * 组件名称（完全一致，不允许修改）
> * Props 定义（必须使用已定义的 Props 接口）
> * 事件处理（必须使用已定义的回调函数）

**集成位置**: [具体页面组件路径，如 `frontend/apps/cozeloop/src/pages/WorkspacePage.tsx`]

**使用的组件**:

* `[ComponentA]` (来自阶段 [N]): [在页面中的位置] - [作用]
  * Props: [引用阶段 N 中定义的 Props，如 `{ prop1, prop2, onEvent }`]

* `[ComponentB]` (来自阶段 [M]): [在页面中的位置] - [作用]
  * Props: [引用阶段 M 中定义的 Props]

#### 2. 状态管理 (如有则必填)

**状态定义**: 使用 [Zustand/useState/useReducer]

```typescript
// 页面级状态（非组件内部状态）
const [state1, setState1] = useState<Type1>([初始值])
const [state2, setState2] = useState<Type2>([初始值])
```

**状态更新时机**:

* `state1`: 当 [具体操作/事件] 时更新
* `state2`: 当 [具体操作/事件] 时更新

**组件间状态传递**:

* `state1` → 通过 [props/context/store] 传递给 `[ComponentA]`, `[ComponentB]`

#### 3. 用户交互流程 (必填)

**步骤 1**: 用户 [具体操作]

* 触发: `[handleXxx]` 事件处理函数
* 状态更新: `[setState(...)]`
* UI 反馈: [显示什么]

**步骤 2**: 用户在 [组件名] 中 [具体操作]

* 触发: `[组件的 onEvent]` 回调方法
* 数据传递: `[data]` 传递给父组件
* 状态更新: [更新哪些状态]

**步骤 3**: [后续处理]

* API 调用: `[apiMethod(params)]`
* 等待响应
* UI 更新: [根据结果更新什么]

#### 4. API 调用处理 (如有则必填)

> **关键依赖**:
> 1. **必须使用阶段 1 中已定义的 API 类型**，从 `@cozeloop/api-schema` 导入
> 2. 必须参考 {{api-doc}} 文档中的 API 定义
> 3. 确保参数、返回值、错误码等与 api-doc 一致

**API 方法**: `[apiMethod]` (来自 `@cozeloop/api-schema`，已在阶段 1 中定义)

**API 文档参考**: {{api-doc}} 中的 [具体章节名称]

**调用时机**: 在 `[PageComponent]` 的 `[handleXxx]` 中调用

**请求参数** (参考 api-doc):

```typescript
{
  param1: [来源],    // api-doc: [类型], [required/optional]
  param2: [来源],    // api-doc: [类型], [required/optional]
}
```

**返回值处理** (参考 api-doc):

```typescript
interface [ApiResponse] {
  field1: [类型]    // api-doc: [说明]
  field2: [类型]    // api-doc: [说明]
}
```

**成功场景**:

* 状态更新: [更新哪些状态]
* UI 反馈: [显示什么提示]
* 后续操作: [执行什么操作]

**失败场景** (参考 api-doc 错误码):

* 错误码映射:
  * `[ERROR_CODE_1]`: "[友好提示]"
  * `[ERROR_CODE_2]`: "[友好提示]"
* 用户提示: [如何显示错误]
* 降级方案: [如有，说明降级策略]

#### 5. 组件集成代码示例

```tsx
function [PageComponent]() {
  // 状态管理
  const [state, setState] = useState(...)

  // 事件处理
  const handleXxx = () => {
    // 具体实现
  }

  return (
    <div>
      <ComponentA prop1={value1} onEvent={handleXxx} />
      {condition && <ComponentB prop2={value2} />}
    </div>
  )
}
```

#### 6. 数据流示意

```text
用户操作 → handleXxx()
    ↓
更新 state
    ↓
传递 props → ComponentA
    ↓
ComponentA.onEvent → API 调用
    ↓
API 响应 → 更新 state → UI 更新
```

#### 7. 验收场景

> **重要**: 如果 spec.md 中已包含验收场景，必须先完整复制，再根据实际情况扩充。不允许删减或修改 spec 中的验收场景。

1. **Given** [前置条件], **When** [触发动作], **Then** [预期结果]
2. **Given** [前置条件], **When** [触发动作], **Then** [预期结果]
3. **Given** [前置条件], **When** [触发动作], **Then** [预期结果]


## 前端规范合规性总结

**规范版本**: 1.0.0\
**合规状态**: [通过/未通过，附说明]

**关键合规点**:

* ✅ 三层组件库架构遵循
* ✅ API 规范遵循 (@cozeloop/api-schema)
* ✅ React 导入规范 (具名导入)
* ✅ 组件规模控制 (<400 行)
* ✅ 命名规范 (@cozeloop 前缀, 小写+中划线)
* ✅ 技术栈合规
* ✅ 样式规范 (Tailwind/Less module)

**质量保障命令**:

```bash
# 更新依赖
rush update

# ESLint 检查
npm run lint

# TypeScript 检查
rush ts-check -o [package-name]

# API Schema 更新
cd frontend/packages/cozeloop/api-schema && npm run update
```

---

## 文档质量检查清单

生成 plan.md 后，请确保以下检查项全部通过：

### 结构检查

- [ ] 文档包含所有必需部分：概要、技术背景、规范检查、技术分析、组件分析、User Story 概述、项目结构、实现阶段、前端规范合规性总结
- [ ] "实现阶段"部分的第一个阶段是"阶段 1"（不是"阶段 0"）
- [ ] 所有阶段编号使用连续递增的纯数字（无字母、小数点等）
- [ ] 阶段 1 是 API 类型定义
- [ ] 阶段 2+ 是组件实现（每个组件一个阶段）
- [ ] 组件阶段之后是 User Story 集成阶段

### 内容检查

- [ ] "技术分析"部分包含所有核心技术决策（决策内容、理由、替代方案）
- [ ] "相关 API 和 IDL"部分列出了所有需要的 API 及其类型定义
- [ ] "需要修改的包"和"需要修改的组件"清晰列出了所有相关模块
- [ ] "组件分析"部分包含了 UI 参考资料、页面入口、动线关系、预览页面规划
- [ ] 每个组件都有完整的 Props 定义
- [ ] 每个组件都列出了复用的基础组件（来自 @coze-arch/coze-design）
- [ ] 每个组件都有至少 2-3 个验收场景
- [ ] 每个 User Story 都完整复制了 spec.md 中的描述和验收场景（如有）
- [ ] User Story 的组件集成部分引用了阶段 2+ 的组件名称和 Props（而非重复定义）
- [ ] User Story 的 API 调用部分使用了阶段 1 的 API 类型（而非重复定义）

### 格式检查

- [ ] 所有代码块使用正确的语言标记（typescript、bash、json 等）
- [ ] 验收场景使用 Given-When-Then 格式
- [ ] 文件路径使用绝对路径
- [ ] API 类型引用 @cozeloop/api-schema
- [ ] 组件层级标注正确（业务基础组件 / 业务特性组件）
- [ ] 包路径使用绝对路径

### 一致性检查

- [ ] 阶段 1（API）定义的类型在后续阶段中被正确引用
- [ ] 组件名称与 ui_analysis 中的命名保持一致
- [ ] 组件层级分配与 plan.md 中的"需要修改的包"一致
- [ ] 验收场景覆盖了 spec.md 中的 user story
- [ ] 引用的组件 Props 与阶段 2+ 定义完全一致
- [ ] 引用的 API 类型与阶段 1 定义完全一致
- [ ] 验收场景包含了 spec.md 中的所有场景（如有）

````
