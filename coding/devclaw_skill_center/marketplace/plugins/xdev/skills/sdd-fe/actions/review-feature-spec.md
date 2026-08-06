---
name: review-feature-spec
description: 审阅并修复 feature_spec.md，对比原始 PRD 进行双层审查（PRD覆盖度+业务逻辑完整性），就地修复问题。Use when user asks to review spec, 审阅spec, 审阅功能规格, review feature spec, 检查spec质量.
---

# 用户 args 解析

优先从 args 里解析相应的参数，如果不存在，则依次向用户询问以下 3 个变量的值（使用 AskUserQuestion 工具逐一询问）：

1. **InputsDir**：原始 PRD 文档所在目录
2. **SpecsDir**：feature_spec.md 所在目录
3. **ConstitutionFile**：constitution 规约文档文件路径

> 每个变量单独询问，确保用户明确输入后再继续下一步。

# Review Feature Spec

对比原始 PRD 文档和 feature_spec.md，从 **PRD 完整性覆盖** 和 **业务逻辑完整性** 两个层面进行全面审阅，直接在原始 feature_spec.md 上修复发现的问题，并生成变更总结报告。

## 参数

| 参数 | 必填 | 说明 | 默认值 | 示例 |
|------|------|------|--------|------|
| `InputsDir` | ✅ 必填 | 原始 PRD 文档所在目录 | — | `/path/to/inputs` |
| `SpecsDir` | ✅ 必填 | feature_spec.md 所在目录 | — | `/path/to/specs` |
| `ConstitutionFile` | ✅ 必填 | constitution 规约文档文件路径 | — | `/path/to/constitution.md` |

### 参数校验

- `InputsDir`、`SpecsDir` 和 `ConstitutionFile` 为**必填**，用户未提供时**必须询问获取**。
- `{SpecsDir}/feature_spec.md` 必须已存在（需先执行 generation-feature-spec）。

### 询问模板

如果用户未提供必填参数，询问：

```
需要以下信息来审阅 Feature Spec：
1. 📂 原始 PRD 文档目录 (InputsDir) — PRD 文档所在目录？
2. 📁 Feature Spec 所在目录 (SpecsDir) — feature_spec.md 所在目录？
3. 📜 规约文档路径 (ConstitutionFile) — constitution 规约文档文件路径？
```

## 审阅流程）

### 1. 读取文档

- 读取 `{InputsDir}` 下的原始 PRD 文档
- 读取 `{SpecsDir}/feature_spec.md`

### 2. 双层审查

#### 第一层：PRD 覆盖度审查（字面对应）

逐行对比 PRD 内容，检查 feature_spec.md 中的覆盖度：

| 维度 | 检查内容 |
|------|----------|
| 业务逻辑字面覆盖 | PRD 每个业务步骤、判断条件、状态转换是否对应体现 |
| 文案完整性 | 按钮文案、提示语、错误消息是否与 PRD 一致 |
| 数据结构与代码示例 | 完整代码块、JSON Schema、API 契约是否完整 |

#### 第二层：业务逻辑完整性审查（功能闭环）

从开发者视角推理业务场景：

| 维度 | 检查内容 |
|------|----------|
| CRUD 功能闭环 | 创建后如何编辑？版本管理？生命周期？ |
| 状态管理与数据流转 | 跨界面数据同步、状态一致性、持久化 |
| 交互流程完整性 | 操作触发方式、退出路径、确认机制 |
| 错误处理与边界 | 错误消息、失败流程、超时处理、空状态 |
| 数据验证与约束 | 输入验证、返回值约束、业务规则约束 |

### 3. 问题分类与评级

| 严重程度 | 说明 | 处理方式 |
|----------|------|----------|
| 🔴 严重 | PRD 核心功能遗漏、业务逻辑断点 | 必须直接修复 |
| 🟡 中等 | PRD 细节遗漏、业务场景覆盖不全 | 直接修复补充 |
| 🟢 轻微 | 文案格式不一致、优化性建议 | 直接修复优化 |

问题类型标签：`[PRD遗漏]` `[逻辑缺失]` `[细节不足]` `[文案差异]`

### 4. 就地修复

- 使用 Edit 工具**直接修改**原始 `{SpecsDir}/feature_spec.md`，逐章节增量编辑
- 无法确定的内容使用 `[待澄清: 具体问题]` 标记
- 保持原有模板结构和格式
- FR 编号格式：FR-001, FR-002（禁止嵌套编号）

### 5. 生成变更总结报告

将变更总结报告保存到 `{SpecsDir}/review_feature_spec.md`，包含：
- 修复汇总表（各严重程度问题数量及修复状态）
- 修复详情（每个问题的章节、描述、修复内容）
- 待澄清项清单

## 核心原则

- **上下文隔离**：只使用提供的两个文档路径，不访问其他文件
- **就地修复**：直接修改原始 feature_spec.md，不生成新 spec 文件
- **不要过度推断**：PRD 未明确的内容不算问题
- **中文输出**：所有内容必须使用中文

## 输出说明

| 输出文件 | 说明 |
|----------|------|
| `{SpecsDir}/feature_spec.md` | 直接修改后的规格文件 |
| `{SpecsDir}/review_feature_spec.md` | 变更总结报告 |

## 完成总结（必须）

审阅修复完毕后，**必须输出一份简短总结**：

```
✅ 审查修复完成
- 已直接修改: {SpecsDir}/feature_spec.md
- 总结报告: {SpecsDir}/review_feature_spec.md

📊 修复汇总:
🔴 严重问题: X 个（已修复）
🟡 中等问题: Y 个（已修复）
🟢 轻微问题: Z 个（已修复）
❓ 待澄清标记: N 个（需用户确认）
```



## 直接开始下一步：mining

直接按照下面的指令开始后续的工作！

```prompt
直接执行 devclaw-sdd-fe mining
  SpecsDir: {SpecsDir}
  InputsDir: {InputsDir}
  RepoDir: （需提供前端代码仓库路径）
  ConstitutionFile: {ConstitutionFile}
```
