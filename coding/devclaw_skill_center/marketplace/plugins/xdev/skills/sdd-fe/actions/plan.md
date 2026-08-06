---
name: plan
description: 根据 feature_spec 和 mining-result 生成完整技术实现方案 plan.md，涵盖技术分析、组件拆分、User Story 集成。Use when user asks to plan, 生成计划, 技术方案, 生成plan, 实现方案, technical plan.
---

# 用户 args 解析

优先从 args 里解析相应的参数，如果不存在，则按以下规则处理：

1. **SpecsDir**：feature_spec.md 和 mining-result.md 所在目录。用户未提供时**必须询问**。
2. **RepoDir**：前端代码仓库路径。用户未提供时，**默认使用当前工作目录（`pwd`）**，无需询问。
3. **InputsDir**（选填）：额外输入目录（requirement.md 等文档目录）
4. **ConstitutionFile**：constitution 规约文档文件路径。用户未提供时**必须询问**。

> SpecsDir 和 ConstitutionFile 每个变量单独询问，确保用户明确输入后再继续下一步。

# Plan

根据功能规格（feature_spec.md）和技术挖掘结论（mining-result.md），生成完整的技术实现方案（plan.md），涵盖从技术分析到 User Story 集成的全部阶段。

## 参数

| 参数 | 必填 | 说明 | 默认值 | 示例 |
|------|------|------|--------|------|
| `SpecsDir` | ✅ 必填 | feature_spec.md 和 mining-result.md 所在目录 | — | `/path/to/specs` |
| `RepoDir` | ❌ 选填 | 前端代码仓库路径，用于分析现有代码结构生成技术方案，默认为当前工作目录（`pwd`） | 当前工作目录（`pwd`） | `/path/to/frontend-repo` |
| `InputsDir` | ❌ 选填 | 额外输入目录（requirement.md 等） | — | `/path/to/inputs` |
| `ConstitutionFile` | ✅ 必填 | constitution 规约文档文件路径 | — | `/path/to/constitution.md` |

### 参数校验

- `SpecsDir` 和 `ConstitutionFile` 为**必填**，用户未提供时**必须询问获取**。
- `RepoDir` 为**选填**，用户未提供时默认使用当前工作目录（`pwd`）。
- `{SpecsDir}/feature_spec.md` 必须已存在。
- `{SpecsDir}/mining-result.md` 应已存在（mining 步骤产出）。
- `RepoDir` 必须是有效的前端代码仓库路径。
- 读取 `{ConstitutionFile}` 作为规约。

### 询问模板

如果用户未提供必填参数，询问：

```
需要以下信息来生成技术方案：
1. 📂 Spec 文档目录 (SpecsDir) — feature_spec.md 和 mining-result.md 所在目录？
2. 📂 额外输入目录 (InputsDir, 选填) — requirement.md 等文档目录？
3. 📜 规约文档路径 (ConstitutionFile) — constitution 规约文档文件路径？

💡 RepoDir 未提供，将默认使用当前工作目录：{pwd}
```

## 生成流程

### 1. 加载上下文

读取以下文档：
- `{SpecsDir}/feature_spec.md`（功能规格）
- `{SpecsDir}/mining-result.md`（隐性需求挖掘结论）
- `{SpecsDir}/api-doc.md`（**如存在**，后端 API 文档 — 由 generation-feature-spec 阶段用户提供。包含 API 接口定义、请求/响应格式、字段说明等。**用于指导技术方案中的 API 类型定义、请求层封装、数据模型设计等决策，确保前端实现与后端接口精确对齐**）
- `{InputsDir}/requirement.md`（如有，UI 设计和 API 设计信息）
- `references/plan_template.md`（作为 Plan 模板）
- `{ConstitutionFile}`（作为规约）
- 探索 `{RepoDir}` 下的代码结构，分析现有组件、目录布局、技术栈和工程约定，基于实际代码结构来规划技术方案

> **api-doc.md 对技术方案的影响**：如果存在 api-doc.md，在以下阶段必须参考其内容：
> - **阶段 1（API 类型定义）**：直接基于 api-doc 中的接口定义生成 TypeScript 类型，而非凭 feature_spec 推测
> - **组件阶段**：组件 Props 和数据流设计需与 API 返回结构对齐
> - **User Story 集成阶段**：API 调用方式、错误处理、请求参数需严格遵循 api-doc 定义

### 2. 执行计划工作流

#### 阶段 0：技术分析与决策（不写入实现阶段）

- 提取技术背景中的未知项，研究并做出决策
- 所有技术决策记录在 plan.md 的"技术分析"部分
- **不在"实现阶段"部分添加"阶段 0"**

#### 阶段 1：基础设计与架构

按固定顺序生成实现阶段：
- **阶段 1**：API 类型定义

#### 阶段 2：组件分析与拆分

- 分析 UI 参考资料，提取组件清单
- 明确页面和组件入口
- 定义页面间动线关系
- 扩展组件预览页面
- 设计组件验收场景（Given-When-Then 格式）
- 结果写入"组件分析"部分和**阶段 2+**（每个组件一个阶段）

#### 阶段 3：User Story 集成

- 从 feature_spec.md 提取所有 User Story
- 为每个 User Story 单独创建实现阶段（从组件阶段之后递增编号）
- 判断必填内容（组件集成、状态管理、交互流程、API 调用）
- **引用前序阶段定义，禁止重复定义**
- 验收场景：先完整复制 spec 中的场景，再扩充新场景

### 3. 分批写入

plan.md 可能非常长，必须分批写入：
- 先写文档头部（技术背景、章程检查、技术分析等）
- 再按阶段分批追加
- 每批写入后确认内容正确再继续

### 4. 报告完成

报告 plan.md 路径和阶段数量。

## 关键规则

| 规则 | 说明 |
|------|------|
| 阶段编号 | 从阶段 1 开始，连续递增纯数字，禁止阶段 0/字母后缀/小数点 |
| 一组件一阶段 | 每个组件都有独立的实现阶段 |
| 一故事一阶段 | 每个 User Story 都有独立的集成阶段 |
| 引用不重复 | User Story 阶段引用前序定义（API 类型、组件 Props） |
| 验收场景优先 | 先复制 spec 验收场景，再扩充 |
| 组件规模控制 | 单个组件 < 400 行，超过则拆分 |

## 输出说明

| 输出文件 | 说明 |
|----------|------|
| `{SpecsDir}/plan.md` | 完整的技术实现方案 |

## 完成总结（必须）

生成完毕后，**必须输出一份简短总结**，包含：
1. **输出文件**：plan.md 的完整路径
2. **阶段统计**：总阶段数、组件阶段数、User Story 阶段数
3. **技术决策**：关键技术决策摘要

## 衔接下一步：split-tasks（⚠️ 强制，不可跳过）

总结输出之后，**必须在最后追加一段引导语**，询问用户是否开始下一个 skill，并预填好参数值。

**⚠️ 这是你的最终输出的最后一段内容，缺少此引导语视为任务未完成。**

引导语模板（必须原样输出，仅替换花括号变量）：

```
🚀 plan 完成！是否继续执行下一步 split-tasks？

参数已就绪：
  SpecsDir: {SpecsDir}
  RepoDir: {RepoDir}
  ConstitutionFile: {ConstitutionFile}

回复「继续」即可启动。
```
