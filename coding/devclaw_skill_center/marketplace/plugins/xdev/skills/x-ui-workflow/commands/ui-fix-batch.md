# UI Fix Batch Command

对 `ui-coding` 产物中的所有业务组件和页面**逐个**执行还原度校验与样式优化。

每个组件独立从 `high-fidelity.jsx` 中定位对应的 JSX 片段，对比现有实现代码，评估组件映射、代码完整度和 Tailwind 规范性，并修正问题。

## 调用方式

```bash
/ui-workflow ui-fix-batch <page_name>
```

参数说明：
- `page_name`: 设计稿名称（与 analyze-design 输出目录名一致）

## 输入与输出

输入：
| 文件 | 用途 |
|------|------|
| `.metis/.design_analysis/pages/<page_name>/ui.implement.xml` | 获取组件列表和 `data-node-ids` |
| `.metis/.design_analysis/pages/<page_name>/high-fidelity.jsx` | 完整页面的高保真 JSX 代码 |
| `apps/components-preview/src/design/<page_name>/` | 已实现的业务组件和页面代码 |

输出：对已有代码进行还原度修正（原地修改）。

## 执行流程

## Step 1: 校验输入与收集组件列表

1. 校验 `.metis/.design_analysis/pages/<page_name>/ui.implement.xml` 和 `high-fidelity.jsx` 是否存在；缺失则提示先执行 `/ui-workflow ui-analyze` 和 `/ui-workflow ui-structure`。
2. 校验 `apps/components-preview/src/design/<page_name>/` 目录是否存在且有实现产物；缺失则提示先执行 `/ui-workflow ui-coding`。
3. 从 `ui.implement.xml` 解析所有 `<component>` 和 `<page>` 节点，提取：
   - `id`（组件目录名）
   - `component-name`
   - `data-node-ids`
4. 读取完整的 `high-fidelity.jsx` 内容。

构建组件映射表：

| 组件目录名 | component-name | data-node-ids | 目标目录 |
|-----------|---------------|---------------|---------|
| model-card | ModelCard | 20:9417 | `apps/components-preview/src/design/<page_name>/components/model-card` |
| page | — | 20:9300 | `apps/components-preview/src/design/<page_name>/page` |

## Step 2: 获取组件库知识

调用 `components-knowledge` skill **一次性**获取可用组件列表及常用组件的 Props/示例。此步骤在组件处理前完成，结果在后续所有组件的处理中共享复用。

## Step 3: 逐个处理每个组件

按组件映射表的顺序，**逐个**对每个组件执行修正。完成当前组件的全部处理流程后，再处理下一个组件。

### 单个组件的处理流程

**3a. 定位高保真 JSX 片段**

在 `high-fidelity.jsx` 中，通过 `data-node-ids` 属性查找该组件对应的 JSX 代码片段。搜索包含目标 node-id 的根元素及其所有子元素即为该组件的高保真 JSX。

**3b. 读取组件现有实现代码**

读取组件目标目录下的所有 `.tsx`/`.ts` 文件，了解现有实现。

**3c. 评估现有实现**

将现有代码与高保真 JSX 片段逐项对比，评估以下三个维度：

1. **组件映射问题**：检查现有代码中应该使用 `@coze-arch/coze-design` 组件库的地方，是否用了原生 HTML 标签 + Tailwind 手写近似效果（如该用 `Table` 组件却用 `<div>` 手写表格、该用 `Tabs` 却用 `<div>` + 手动状态切换等）。结合 Step 2 获取的组件库知识判断。

2. **代码完整度**：对比高保真 JSX，检查现有实现是否存在遗漏：
   - 缺失的 UI 元素（如图标、badge、tooltip 等装饰性元素）
   - 缺失的布局结构（如 flex 容器嵌套层级、间距、对齐方式）
   - 缺失的交互状态样式（如 hover、active、disabled 状态）

3. **Tailwind 规范性**：检查现有代码中的 Tailwind 类名是否符合项目规范：
   - 是否存在 Figma 原始颜色 token（如 `text-Fg-COZ-fg-secondary/60`）未转换为 `coz-` 语义类
   - 是否存在值等于 CSS 默认值的冗余类名（如 `justify-start`、`items-stretch`、`flex-row` 等）
   - 是否存在 Figma 导出特有的冗余模式（如 `font-['PingFang_SC']`、多余的 `relative`）

**3d. 修正问题**

根据 3c 的评估结果修改代码：
- 将手写近似实现替换为 `@coze-arch/coze-design` 组件
- 补全遗漏的 UI 元素和布局结构
- 将 Figma 颜色 token 转换为 `coz-` 语义类，移除冗余类名
- 图标替换为 `@coze-arch/coze-design/icons`
- **保留业务逻辑**，只修改 UI 层

### 单个组件的 query 模板

```text
校验并修正组件还原度。

组件信息：
- 组件名：<component_name>
- 目标目录：apps/components-preview/src/design/<page_name>/components/<component_id>

可用组件库知识：
<Step 2 获取的组件库信息摘要>

高保真 JSX 片段（从 high-fidelity.jsx 中按 data-node-ids="<node_ids>" 提取）：
<对应的高保真 JSX 代码片段>

请执行以下操作：
1. 读取目标目录现有代码
2. 对比高保真 JSX 评估现有实现：
   - 组件映射问题：是否有应使用组件库却手写的地方
   - 代码完整度：是否有相比高保真 JSX 遗漏的元素或布局
   - Tailwind 规范性：颜色 token 是否已转换、是否存在冗余类名
3. 修正发现的问题，保留业务逻辑
4. 输出评估报告和修改总结
```

## Tailwind CSS 清洗规则

评估和修正 Tailwind 类名时**必须**依据以下两项规则。

### 颜色类名转换

Figma 导出的颜色类名（如 `text-Fg-COZ-fg-secondary/60`）**不能直接使用**，需转换为 `coz-` 开头的语义类。规则：去除 `text-`/`bg-`/`border-` 前缀及中文描述，提取 `COZ-` token 转小写，丢弃 `/opacity` 后缀。

颜色变量参考文件：`.trae/skills/components-knowledge/knowledge/tailwind/index.md`

示例：`text-Fg-COZ-fg-secondary/60` → `coz-fg-secondary`，`bg-Mg-默认-COZ-mg-primary/10` → `coz-mg-primary`

> 语义类已绑定对应 CSS 属性（`color`/`background-color`/`border-color`），无需再加 `text-`/`bg-`/`border-` 前缀。

### 去除无效类名

移除 Figma 导出中值等于 CSS 默认值的冗余类名：`justify-start`、`items-stretch`、`flex-row`、`flex-nowrap`、`self-auto`、`order-none`、`flex-grow-0`、`flex-shrink`、`basis-auto`、`static`、`visible`、`border-solid`、`bg-transparent`、`text-left`、`font-normal`、`leading-normal`、`tracking-normal`、`opacity-100`、`decoration-none`。

同时移除 Figma 导出特有的冗余模式：`font-['PingFang_SC']`（全局字体已配置）、`inline-flex`（多数应改为 `flex`）、`relative`（仅在有 `absolute` 子元素时保留）。

## 检查项

- [ ] `ui.implement.xml` 和 `high-fidelity.jsx` 已加载
- [ ] 实现产物目录存在
- [ ] 组件映射表已构建（id、data-node-ids、目标目录）
- [ ] 组件库知识已获取（一次性）
- [ ] 每个组件的高保真 JSX 片段已定位
- [ ] 已逐个对所有组件执行评估与修正
- [ ] 组件映射问题已修正
- [ ] 代码完整度已补全
- [ ] Tailwind CSS 规范已应用（颜色类名转换 + 无效类名移除）

## 返回结果格式

1. 组件映射表（组件名、node-ids、目标目录）
2. 各组件评估报告（组件映射问题 / 代码缺漏 / Tailwind 问题）
3. 各组件修正结果摘要
4. 修正统计（修改文件数、调整项）
