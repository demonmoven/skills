# Design From Figma Command

从 Figma 设计稿出发，串联分析与代码实现，并基于真实产物生成 `component-design.md`。

## 重要规则

1. 开始前必须读取 `.claude/docs/constitution.md`，遵守本工程技术规范。
2. 必须严格顺序执行 `/ui-analyze` → `/ui-structure-and-coding`，任何一步都不能跳过。
3. `component-design.md` 必须基于真实生成的代码产物编写，禁止凭空总结。
4. 组件文档规范统一遵循 [../references/component-design-guidelines.md](../references/component-design-guidelines.md)。

## 调用方式

```bash
x-ui-workflow /design-from-figma <page_name> <figma_url> [description]
```

参数说明：
- `page_name`：英文 kebab-case 页面名，例如 `reward-modal`
- `figma_url`：Figma 页面或节点级链接
- `description`：可选的业务背景或补充说明

## 执行流程

## Step 1: 执行 `/ui-analyze`

调用 [ui-analyze.md](./ui-analyze.md) 中的工作流，传入：

```bash
x-ui-workflow /ui-analyze <page_name> <figma_url> [description]
```

必须产出：
- `.metis/.design_analysis/pages/<page_name>/design.png`
- `.metis/.design_analysis/pages/<page_name>/high-fidelity.jsx`

即使文件已存在，也必须重新生成。

## Step 2: 执行 `/ui-structure-and-coding`

调用 [ui-structure-and-coding.md](./ui-structure-and-coding.md) 中的工作流，传入 Step 1 的真实产物路径：

```bash
x-ui-workflow /ui-structure-and-coding <page_name> <figma_url> <design_image> <jsx_code> [description]
```

此命令在同一上下文中完成 UI 结构分析和代码实现，**不生成中间文件**（`ui.xml`、`ui.implement.xml`、`ui.md`），分析结果在内存中直接传递给编码阶段。

必须产出：
- `apps/components-preview/src/design/<page_name>/components/*/implement.tsx`
- `apps/components-preview/src/design/<page_name>/page/implement.tsx`
- `apps/components-preview/src/design/<page_name>/summary.md`

如果产物缺失，说明实现步骤未完成，必须先补齐，再进入 Step 3。

## Step 3: 生成 `component-design.md`

### 3.1 读取真实代码产物

必须重新读取以下文件：
- `apps/components-preview/src/design/<page_name>/components/*/implement.tsx`
- `apps/components-preview/src/design/<page_name>/page/implement.tsx`
- `apps/components-preview/src/design/<page_name>/summary.md`

信息冲突时，以真实代码为准。

### 3.2 生成文档

将组件设计文档写入：

```text
apps/components-preview/src/design/<page_name>/component-design.md
```

> **再次强调**：此文件必须与组件代码放在同一目录下，即 `apps/components-preview/src/design/<page_name>/`。
> 不要写入 `.metis/.design_analysis/` 目录——那里只存放分析中间产物。

文档内容必须：
- 与真实组件结构、命名、路径一致
- 包含页面与业务组件的 UI 结构说明
- 使用 ASCII 框图表达主要层级和布局
- 明确哪些内容是组件职责，哪些只是页面组合

## 输出位置

| 产物 | 路径 |
|------|------|
| 设计原始数据 | `.metis/.design_analysis/pages/<page_name>/` |
| UI 实现代码 | `apps/components-preview/src/design/<page_name>/` |
| 组件描述文档 | `apps/components-preview/src/design/<page_name>/component-design.md` |

## 返回结果

1. 简述各步骤是否执行完成
2. 给出 `component-design.md` 的输出路径（必须以 `apps/components-preview/src/design/` 开头）
3. 读取最终文件内容，并将完整 Markdown 作为回复主体返回
