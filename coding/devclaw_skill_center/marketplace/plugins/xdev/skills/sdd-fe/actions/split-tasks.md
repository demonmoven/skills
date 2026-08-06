---
name: split-tasks
description: 从 plan.md 中提取已划分的实现阶段，结合 mining-result 隐性需求和 api-doc（如有），为每个 Phase 生成独立的前端 Tasks 文档（subtasks_N.md）。Use when user asks to split tasks, 拆分任务, 任务拆分, 生成subtasks, generate implementation tasks.
---

# 用户 args 解析

优先从 args 里解析相应的参数，如果不存在，则按以下规则处理：

1. **SpecsDir**：所有前序产出文档（spec, plan, mining-result 等）所在目录。用户未提供时**必须询问**。
2. **RepoDir**：前端代码仓库路径。用户未提供时，**默认使用当前工作目录（`pwd`）**，无需询问。
3. **ConstitutionFile**：constitution 规约文档文件路径。用户未提供时**必须询问**。

> SpecsDir 和 ConstitutionFile 每个变量单独询问，确保用户明确输入后再继续下一步。

# Split Tasks

从 plan.md 中提取已划分好的实现阶段（Phase），结合 mining-result.md 中的隐性技术需求和 api-doc.md（如有）中的接口契约，为每个 Phase 生成独立的前端 Tasks 文档（subtasks_1.md, subtasks_2.md, ...）。

**核心原则：Phase 划分由 plan.md 决定，split-tasks 不重新划分 Phase，只在每个 Phase 内细化 Task。**

## 参数

| 参数 | 必填 | 说明 | 默认值 | 示例 |
|------|------|------|--------|------|
| `SpecsDir` | ✅ 必填 | 所有前序产出文档目录（spec.md, plan.md, mining-result.md 等） | — | `/path/to/specs` |
| `RepoDir` | ❌ 选填 | 前端代码仓库路径，用于生成精确文件路径的 Task 和传递给后续 development，默认为当前工作目录（`pwd`） | 当前工作目录（`pwd`） | `/path/to/frontend-repo` |
| `ConstitutionFile` | ✅ 必填 | constitution 规约文档文件路径 | — | `/path/to/constitution.md` |

### 参数校验

- `SpecsDir` 和 `ConstitutionFile` 为**必填**，用户未提供时**必须询问获取**。
- `RepoDir` 为**选填**，用户未提供时默认使用当前工作目录（`pwd`）。
- `{SpecsDir}/plan.md` **必须已存在**（plan 步骤产出，是 Phase 划分的唯一来源）。
- `{SpecsDir}/mining-result.md` **必须已存在**（mining 步骤产出）。
- `RepoDir` 必须是有效的前端代码仓库路径，用于生成 Task 中的精确文件路径。
- 读取 `{ConstitutionFile}` 作为规约。

### 询问模板

如果用户未提供必填参数，询问：

```
需要以下信息来拆分任务：
1. 📂 Spec 文档目录 (SpecsDir) — 所有前序产出文档（spec, plan, mining-result 等）所在目录？
2. 📜 规约文档路径 (ConstitutionFile) — constitution 规约文档文件路径？

💡 RepoDir 未提供，将默认使用当前工作目录：{pwd}
```

## 拆分流程

### 1. 加载并分析前序文档

读取并分析以下文档：

**必读文档**：
- `{SpecsDir}/plan.md` — **Phase 划分的唯一来源**，提取其"实现阶段"部分中的所有阶段定义（阶段目标、主要工作、验收场景等）
- `{SpecsDir}/mining-result.md` — 提取隐性技术需求汇总表，将相关隐性需求注入对应 Phase 的 Task 中
- `{SpecsDir}/spec.md` 或 `{SpecsDir}/feature_spec.md` — 需求内容及 User Stories

**可选文档**（如存在则必须读取）：
- `{SpecsDir}/api-doc.md` — API 接口契约，用于细化涉及 API 调用的 Task（参数、返回值、错误处理等）
- `{ConstitutionFile}` — 作为规约

### 2. 提取 Plan 中的 Phase 列表

从 `plan.md` 的"实现阶段"部分，按顺序提取所有阶段，识别每个阶段的：
- **阶段编号**（阶段 1, 阶段 2, ...）
- **阶段类型**（API 类型定义 / 组件实现 / User Story 集成）
- **阶段目标**
- **主要工作项**
- **验收场景**（如有）
- **涉及的包/组件/页面路径**

> ⚠️ **严格保持 plan.md 中的 Phase 顺序和数量，不增不减不合并。**

### 3. 匹配 Mining-result 隐性需求

将 mining-result.md 中需求汇总表的每条隐性需求，按以下规则分配到对应 Phase：
- 根据隐性需求的**所属层级**和**影响范围**判断它属于哪个 Phase
- 页面生命周期相关 → 对应页面所在的 User Story Phase
- 组件状态流转相关 → 对应组件所在的组件实现 Phase
- 用户交互动线相关 → 对应交互场景所在的 User Story Phase
- 数据获取与缓存相关 → 可能涉及 API 类型定义 Phase 或 User Story Phase
- 组件复用与影响相关 → 对应被复用组件的组件实现 Phase
- 若某条隐性需求跨多个 Phase，在最相关的 Phase 中创建 Task，并在 Leverage 中注明影响范围

### 4. 融合 API-Doc（如存在）

如果 `{SpecsDir}/api-doc.md` 存在，则：
- **API 类型定义 Phase**：对照 api-doc 中的接口定义，细化类型验证 Task（字段类型、必填/选填、枚举值等）
- **User Story 集成 Phase**：对照 api-doc 细化 API 调用相关 Task（请求参数来源、响应数据处理、错误码映射等）
- 在相关 Task 的 Leverage 中引用 api-doc 的具体章节

### 5. 读取 Phase Tasks 模板

读取 `references/subtasks_template.md`，理解模板结构和格式要求。

### 6. 生成 Phase Tasks 文档

为 plan.md 中的每个阶段生成对应的 `subtasks_N.md`：

#### Phase 与 Plan 阶段的映射关系

| Plan 阶段类型 | subtasks 文件 | 说明 |
|--------------|---------------|------|
| 阶段 1: API 类型定义 | `subtasks_1.md` | API Schema 更新与类型验证 |
| 阶段 2+: 组件实现 | `subtasks_2.md` ~ `subtasks_M.md` | 每个组件一个 Phase |
| 阶段 N+: User Story 集成 | `subtasks_{M+1}.md` ~ `subtasks_{M+K}.md` | 每个 User Story 一个 Phase |

#### Task 格式要求

```markdown
- [ ] T001 [USx] 任务描述，路径：src/path/to/file.ts
    - Leverage:
      1. src/path/to/reference.ts:100行-120行
      2. plan.md:阶段N-主要工作项X
      3. mining-result.md:编号MR-xxx
      4. api-doc.md:接口名称（如有）
    - Restrictions: 约束条件说明
```

- 必须以 `- [ ]` 复选框开头
- Task ID 连续编号：T001, T002, T003...（跨 Phase 连续）
- User Story 集成 Phase 的 Task **必须**有 `[USx]` 标签
- API 类型定义 / 组件实现 Phase 的 Task **不允许**有 `[USx]` 标签
- 包含精确的文件路径（以 `RepoDir` 为基准）
- Leverage 中**必须注明信息来源**：plan.md 的哪个阶段、mining-result.md 的哪条需求、api-doc.md 的哪个接口

#### Task 拆分粒度指导

| Phase 类型 | Task 拆分粒度 | 示例 |
|-----------|--------------|------|
| API 类型定义 | 按 API Schema 更新、类型验证、编译检查拆分 | T001 更新 API Schema、T002 验证类型定义 |
| 组件实现 | 按组件骨架、Props 定义、核心逻辑、样式、预览页面集成拆分 | T003 创建组件骨架与 Props、T004 实现核心逻辑 |
| User Story 集成 | 按页面创建、组件集成、状态管理、API 调用、交互流程、错误处理拆分 | T010 创建页面路由、T011 集成组件 |

#### 每个 subtasks_N.md 必须包含

1. **前置依赖（Prerequisites）** 章节（从模板原封不动复制）
2. 该 Phase 的**目标**（直接引用 plan.md 中对应阶段的目标）
3. 该 Phase 的**独立验证方式**（基于 plan.md 中的验收场景转化）
4. **来源追溯** — 标注该 Phase 对应 plan.md 的阶段编号、融合了 mining-result 的哪些隐性需求编号
5. 该 Phase 的 **Tasks 列表**
6. **Phase 的实现策略（Implementation Strategy For Each Phase）** 章节（从模板原封不动复制）

## 🚫 禁止事项

| # | 禁止事项 | 正确做法 |
|---|----------|----------|
| 1 | 自行创造 plan.md 中不存在的 Phase | 严格按 plan.md 中的阶段划分，一一对应 |
| 2 | 合并 plan.md 中的多个阶段到一个 subtasks 文件 | 1 个 plan 阶段 = 1 个 subtasks 文件 |
| 3 | 遗漏 plan.md 中已定义的阶段 | 每个阶段都必须有对应的 subtasks 文件 |
| 4 | 忽略 mining-result 中的隐性需求 | 每条隐性需求必须分配到某个 Phase 的 Task 中 |
| 5 | 包含 api-doc/IDL 变更 Task | api-doc 仅作为参考来源，不变更 IDL 本身 |
| 6 | 单 Phase 超过 10 个 Task | 合并细碎 Task |
| 7 | 最后添加编译验证/部署收尾 Phase | 每个 Phase 自保编译通过 |
| 8 | Task 缺少复选框/ID/路径/来源追溯 | 严格遵循 Task 格式 |

## 输出说明

| 输出文件 | 说明 |
|----------|------|
| `{SpecsDir}/subtasks_1.md` | Phase 1 任务文档（对应 plan.md 阶段 1） |
| `{SpecsDir}/subtasks_2.md` | Phase 2 任务文档（对应 plan.md 阶段 2） |
| ... | ... |
| `{SpecsDir}/subtasks_N.md` | Phase N 任务文档（对应 plan.md 阶段 N） |

## 完成总结（必须）

拆分完毕后，**必须输出一份简短总结**：

```
✅ 任务拆分完成

📂 输出目录：{SpecsDir}
📊 Phase 统计（与 plan.md 阶段一一对应）：
  - 总 Phase 数：X（与 plan.md 阶段数一致）
  - API 类型定义 Phase：Y 个
  - 组件实现 Phase：Z 个
  - User Story 集成 Phase：W 个

📝 Task 统计：
  - 总 Task 数：N
  - Phase 1 ({名称}): M 个 Task
  - Phase 2 ({名称}): M 个 Task
  - ...

🔍 隐性需求融合：
  - mining-result 总需求数：X
  - 已分配到 Task 的需求数：Y
  - 未分配需求：Z（附说明）

📄 API-Doc 融合：{已融合 / 无 api-doc}
```

## 衔接下一步：development（⚠️ 强制，不可跳过）

总结输出之后，**必须在最后追加一段引导语**，询问用户是否开始开发，并预填好参数值。

**⚠️ 这是你的最终输出的最后一段内容，缺少此引导语视为任务未完成。**

引导语模板（必须原样输出，仅替换花括号变量）：

```
🚀 split-tasks 完成！是否继续执行下一步 development（Phase 1）？

参数已就绪：
  SpecsDir: {SpecsDir}
  PhaseNumber: 1
  RepoDir: {RepoDir}
  ConstitutionFile: {ConstitutionFile}

回复「继续」即可启动。
```
