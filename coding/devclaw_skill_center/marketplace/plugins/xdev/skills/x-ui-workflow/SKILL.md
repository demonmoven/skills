---
name: x-ui-workflow
description: UI 还原与组件设计工作流，整合从 Figma 设计稿到代码实现、组件描述文档和 UI 修复的完整链路。包含九个子命令：x-ui-workflow /ui-analyze 获取截图和高保真 JSX；/ui-structure 进行 UI 结构分析；/ui-coding 生成 UI 代码；/ui-structure-and-coding 合并结构分析与代码生成（高效模式）；/ui-fix 按设计稿修复现有实现；/ui-fix-batch 批量并行校验所有组件还原度；/design-from-figma 从 Figma 串联分析、实现并生成 component-design.md；/design-from-directory 从已有目录补齐 component-design.md；/fix-ui 兼容旧版 UI 修复入口。当用户提到 Figma 转组件、组件描述文档、UI 还原、设计稿对齐、分析设计稿、修复 UI 偏差等场景时，优先使用此 skill。
argument-hint: <subcommand> [args...]
allowed-tools: Read, Write, Edit, Bash, Glob, Grep, Task, TodoWrite, Skill
---

## Command Routing Reference

**STOP. Check this table first before any response.**

| Trigger Pattern | Action |
|-----------------|--------|
| `x-ui-workflow /ui-analyze` OR "分析设计稿" OR "analyze figma" OR "从零分析" | **STOP. Read [commands/ui-analyze.md](./commands/ui-analyze.md) and follow it.** |
| `x-ui-workflow /ui-structure-and-coding` OR "分析并实现" OR "结构分析+编码" OR "一步到位" | **STOP. Read [commands/ui-structure-and-coding.md](./commands/ui-structure-and-coding.md) and follow it.** |
| `x-ui-workflow /ui-structure` OR "UI 结构分析" OR "structure analyze" OR "组件结构分析" | **STOP. Read [commands/ui-structure.md](./commands/ui-structure.md) and follow it.** |
| `x-ui-workflow /ui-coding` OR "生成 UI 代码" OR "ui coding" OR "实现组件" | **STOP. Read [commands/ui-coding.md](./commands/ui-coding.md) and follow it.** |
| `x-ui-workflow /ui-fix` OR "修复 UI" OR "fix ui" OR "UI 对齐" OR "还原设计稿" OR "UI 偏差" | **STOP. Read [commands/ui-fix.md](./commands/ui-fix.md) and follow it.** |
| `x-ui-workflow /ui-fix-batch` OR "批量修复 UI" OR "batch fix ui" OR "并行校验还原度" | **STOP. Read [commands/ui-fix-batch.md](./commands/ui-fix-batch.md) and follow it.** |
| `x-ui-workflow /design-from-figma` OR `/design-from-figma` OR "从 Figma 生成组件" OR "生成 component-design" OR "figma 转组件" | **STOP. Read [commands/design-from-figma.md](./commands/design-from-figma.md) and follow it.** |
| `x-ui-workflow /design-from-directory` OR `/design-from-directory` OR "补齐描述文档" OR "生成组件文档" OR "已有目录补文档" | **STOP. Read [commands/design-from-directory.md](./commands/design-from-directory.md) and follow it.** |

If no command matched, use the auto-detection rules below to determine the best command.

---

## Auto-Detection Rules

若用户未明确使用子命令，根据以下特征自动判定并路由：

| 用户输入特征 | 分发到 |
|------------|--------|
| 有 `figma_url` + `page_name`，强调"截图 / JSX / 原始分析输入" | `/ui-analyze` |
| 已有截图路径 + JSX 路径，需要"分析并实现"或"一步到位" | `/ui-structure-and-coding` |
| 已有截图路径 + JSX 路径，仅需要做 UI 结构分析（不需要编码） | `/ui-structure` |
| 有 `page_name`，无 `figma_url`，强调"生成代码"或"实现" | `/ui-coding` |
| 有 `figma_url` + `page_name`，强调"组件文档 / component-design / 从零生成组件" | `/design-from-figma` |
| 有现成目录路径，无 `figma_url`，强调"补文档 / 描述 / component-design" | `/design-from-directory` |
| 有 `figma_url` + 目标目录 + 修改描述，强调"修复 / 对齐 / 还原偏差" | `/ui-fix` |
| 有 `page_name`，强调"批量修复 / 批量校验还原度 / 并行 ui-fix" | `/ui-fix-batch` |

---

## Commands Overview

| 命令 | 签名 | 核心产出 |
|------|------|---------|
| `/ui-analyze` | `<page_name> <figma_url> [description]` | 设计稿截图 + 高保真 JSX |
| `/ui-structure-and-coding` | `<page_name> <figma_url> <design_image> <jsx_code> [description]` | 业务组件代码 + 页面代码（跳过中间文件） |
| `/ui-structure` | `<page_name> <figma_url> <design_image> <jsx_code> [description]` | `ui.xml` + `ui.implement.xml` + `ui.md` + `nodes/` |
| `/ui-coding` | `<page_name> [--components-only\|--page-only]` | 业务组件代码 + 页面代码 |
| `/ui-fix` | `"<需求描述>" <figma_url> <target_directory>` | 修改后的代码文件 |
| `/ui-fix-batch` | `<page_name>` | 批量并行校验并修正所有组件还原度 |
| `/design-from-figma` | `<page_name> <figma_url> [description]` | 组件代码 + `component-design.md` |
| `/design-from-directory` | `<reference_dir> [output_path]` | 基于现有代码生成 `component-design.md` |

### 命令关系说明

```text
高效模式（design-from-figma 默认）:
ui-analyze              → ui-structure-and-coding          → component-design.md
(截图+JSX)                (结构分析+代码实现，无中间文件)      (文档生成)

分步模式（可单独调用）:
ui-analyze         → ui-structure       → ui-coding        → ui-fix-batch
(截图+JSX)           (结构分析+节点)       (代码实现)           (批量还原度校验)

design-from-figma = ui-analyze → ui-structure-and-coding → component-design.md
design-from-directory = existing code → component-design.md
fix-ui = ui-fix
```

- `/ui-analyze`：只负责从 Figma 获取原始数据（截图、高保真 JSX），不做 UI 分析
- `/ui-structure-and-coding`：**合并命令**，在同一上下文中完成结构分析和代码生成，不生成中间文件，效率最高
- `/ui-structure`：读取 `/ui-analyze` 产物，完成 UI 结构分析和节点预处理（分步模式，需要中间文件时使用）
- `/ui-coding`：读取 `/ui-structure` 产物，生成业务组件和页面代码（分步模式，依赖 ui-structure 产物）
- `/ui-fix`：独立命令，对已有代码按 Figma 设计稿进行 UI 对齐修复
- `/ui-fix-batch`：批量命令，读取 ui-coding 产物后并行对所有组件调用 `/ui-fix` 校验还原度
- `/design-from-figma`：高级编排命令，使用高效模式串联分析与实现，并基于真实产物生成 `component-design.md`
- `/design-from-directory`：不走 Figma MCP，直接从已有实现目录提取组件设计文档

组件描述文档的结构规范统一见 [references/component-design-guidelines.md](./references/component-design-guidelines.md)。

## Global Rules

所有命令执行前必须遵守：

1. **读取技术规范**：执行前必须读取 `.claude/docs/constitution.md` 了解本工程技术规范
2. **严格按步骤执行**：命令文档中的步骤不可跳过或合并，必须等前一步完成再执行下一步
