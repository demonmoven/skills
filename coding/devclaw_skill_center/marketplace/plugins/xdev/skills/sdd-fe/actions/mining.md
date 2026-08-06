---
name: mining
description: 基于 Spec 文档和 UI 分析，通过前端分层分析方法论挖掘代码中隐藏的技术需求。Use when user asks to mining, 挖掘需求, 技术挖掘, fe mining, 挖掘隐性需求, hidden requirements analysis.
---

# 用户 args 解析

优先从 args 里解析相应的参数，如果不存在，则按以下规则处理：

1. **SpecsDir**：feature_spec.md 所在目录。用户未提供时**必须询问**。
2. **InputsDir**：requirement.md（含 UI 分析）所在目录。用户未提供时**必须询问**。
3. **RepoDir**：前端代码仓库路径。用户未提供时，**默认使用当前工作目录（`pwd`）**，无需询问。
4. **ConstitutionFile**：constitution 规约文档文件路径。用户未提供时**必须询问**。

> SpecsDir、InputsDir 和 ConstitutionFile 每个变量单独询问，确保用户明确输入后再继续下一步。

# FE Mining

基于 Spec 文档、UI 分析结果及项目规约，通过前端分层分析方法论，挖掘代码中隐藏的、未在文档中明确写出但实际存在的技术需求。

## 参数

| 参数 | 必填 | 说明 | 默认值 | 示例 |
|------|------|------|--------|------|
| `SpecsDir` | ✅ 必填 | feature_spec.md 所在目录 | — | `/path/to/specs` |
| `InputsDir` | ✅ 必填 | requirement.md 所在目录（含 UI 分析） | — | `/path/to/inputs` |
| `RepoDir` | ❌ 选填 | 前端代码仓库路径，用于探索工程代码挖掘隐性需求，默认为当前工作目录（`pwd`） | 当前工作目录（`pwd`） | `/path/to/frontend-repo` |
| `ConstitutionFile` | ✅ 必填 | constitution 规约文档文件路径 | — | `/path/to/constitution.md` |

### 参数校验

- `SpecsDir`、`InputsDir` 和 `ConstitutionFile` 为**必填**，用户未提供时**必须询问获取**。
- `RepoDir` 为**选填**，用户未提供时默认使用当前工作目录（`pwd`）。
- `{SpecsDir}/feature_spec.md` 必须已存在。
- `RepoDir` 必须是有效的前端代码仓库路径。
- 读取 `{ConstitutionFile}` 作为规约。

### 询问模板

如果用户未提供必填参数，询问：

```
需要以下信息来执行技术挖掘：
1. 📂 Spec 文档目录 (SpecsDir) — feature_spec.md 所在目录？
2. 📂 需求文档目录 (InputsDir) — requirement.md（含 UI 分析）所在目录？
3. 📜 规约文档路径 (ConstitutionFile) — constitution 规约文档文件路径？

💡 RepoDir 未提供，将默认使用当前工作目录：{pwd}
```

## 挖掘流程

### 1. 输入校验 & 加载

读取并分析以下文档：
- `{SpecsDir}/feature_spec.md`（功能规格）
- `{InputsDir}/requirement.md` 中 UI 设计相关模块（UI 分析）
- `{ConstitutionFile}`（作为规约）
- 探索 `{RepoDir}` 下的代码结构，了解现有组件、hooks、工具函数等，为挖掘隐性需求提供工程上下文

### 2. 前端五层分析挖掘

按以下五个维度逐层分析，挖掘隐性技术需求：

| 层级 | 分析维度 | 核心问题 |
|------|----------|----------|
| 第一层 | 页面生命周期分析 | 初始化、中间态、退出清理、刷新恢复 |
| 第二层 | 组件状态流转分析 | 跨组件共享、状态保留、冲突解决、副作用 |
| 第三层 | 用户交互动线分析 | 防抖节流、长耗时操作、失败重试、撤销重做 |
| 第四层 | 数据获取与缓存分析 | 避免重复请求、重新渲染、弱网降级、分页管理 |
| 第五层 | 组件复用与影响分析 ⭐ | 复用组件修改影响、Props 兼容性、迁移方案 |

### 3. 输出文档

将分析结果输出到 `{SpecsDir}/mining-result.md`。

## ⚠️ 重要约束

- **禁止输出任何具体代码片段**，只描述逻辑和流程
- **必须生成 Mermaid 状态图/时序图**，用于可视化状态流转和交互流程
- 使用文字描述代替代码引用
- 每个分析维度至少一个图表
- 使用中文标注参与者和消息

## 输出格式

```markdown
# 前端隐性技术需求分析报告

## 1. 分析概览
## 2. 分层分析结果
### 2.1 页面生命周期分析
### 2.2 组件状态流转分析
### 2.3 用户交互动线分析
### 2.4 数据获取与缓存分析
### 2.5 组件复用与影响分析 ⭐
## 3. 需求汇总表
| 编号 | 需求描述 | 所属层级 | 重要程度 | 影响范围 |
```

## 输出说明

| 输出文件 | 说明 |
|----------|------|
| `{SpecsDir}/mining-result.md` | 前端隐性技术需求分析报告 |

## 完成总结（必须）

挖掘完毕后，**必须输出一份简短总结**，包含：
1. **输出文件**：mining-result.md 的完整路径
2. **发现数量**：按重要程度（关键/重要/一般）统计隐性需求数
3. **关键发现**：列出最重要的 2-3 个隐性需求

# 直接开始下一步：plan
直接按照下面的指令开始后续的工作！

```prompt
执行下一步 devclaw-sdd-fe plan
  SpecsDir: {SpecsDir}
  InputsDir: {InputsDir}
  RepoDir: {RepoDir}
  ConstitutionFile: {ConstitutionFile}
```
