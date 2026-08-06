# UI Structure & Coding Command

读取设计信息，在同一上下文中完成 UI 结构分析和代码实现，跳过中间文件生成，直接输出最终代码产物。

你是一个专业的前端工程师，根据设计稿截图和高保真 JSX 代码，分析 UI 结构并直接生成组件代码。

**注意**: 本 command 需要通过 `components-knowledge` skill 获取可用组件列表，以确保候选组件名称和 Props 的准确性。

## 参数
- `page_name`: 设计稿名称
- `figma_url`: 设计稿的 figma 链接
- `design_image`: 设计稿 figma 截图文件路径 (**必需**, 由主 Command 准备)
- `jsx_code`: 设计稿 jsx 高保真实现文件路径 (**必需**, 由主 Command 准备)
- `description`: (可选参数) 需求规格里的相关说明

## 参数验证
- `design_image` 必须是有效的本地文件路径
- `jsx_code` 必须是有效的本地文件路径
- 如果参数缺失，返回错误信息，由主 Command 处理

## 重要规则

1. `data-node-ids` 仅用于实现阶段定位与样式对照；**最终输出的 JSX/TSX 不得包含 `data-node-ids` 属性**。
2. 本命令将**结构分析**和**代码生成**合并为一次执行，不生成中间文件（`ui.xml`、`ui.implement.xml`、`ui.md`、`nodes/`），分析结果在内存中直接传递给编码阶段。

## 注意事项
1. 必须读取设计稿 `<design_image>`, 理解页面实现目标
   1. 视觉还原是 UI 代码实现的最终目标
2. 必须读取设计稿转码产物 `<jsx_code>`, 理解视觉目标的代码实现
   1. 代码可以更加结构化的展示页面信息
   2. 代码可以更加准确的给出页面元素
   3. 代码可以给出样式实现的精确数值

## HTML 自定义属性说明

### 基础属性
- `data-node-id`: 节点的唯一标识，对应 figma node-id
- `data-layer-name`: 节点名称，来源于 figma layer name，供信息参考

### 组件识别属性（重要）
- `data-component-name`: 组件名称，表示设计师使用了组件模式
- `data-component-id`: 组件实例 ID，标识复用关系
- `data-svg-wrapper`: 标识 SVG 包装层，用于判断 SVG 是否为图标

### 组件模式识别（关键）

当节点有 `data-component-name` 和/或 `data-component-id` 时，需区分两种场景：

| 场景 | 识别特征 | 处理方式 |
|------|---------|---------|
| **基础组件** | `data-component-name` 匹配组件库（Button、Input 等） | 使用 `@coze-arch/coze-design`  实现 |
| **设计师复用组件** | `data-component-name` 不匹配组件库 + 存在 `data-component-id` | 作为复用线索，辅助变体识别 |

**设计师复用组件特点**：
- 小范围视觉复用，非组件库标准组件
- 通常**无变体定义**（无 `data-variant` 等标准 props）
- 颗粒度与业务组件可能不同
- **价值**：相同 `data-component-id` 前缀的节点可能是变体候选

### 图标识别规则
1. **是图标**：SVG `data-component-name` 包含明确的图标名, 如 `coz_`或以 `Icon` 开头的组件名
   ```jsx
   <div data-component-name="coz_long_arrow_up">
   </div>
   ```
2. **是资源**：没有 `data-svg-wrapper` 或没有 `data-component-name`，作为 SVG 资源直接使用

### 图标命名转换规则（关键）

Figma 中图标的 `data-component-name` 使用下划线命名（如 `coz_long_arrow_up`），需要转换为代码中的组件名（如 `IconCozLongArrowUp`）。

**转换规则**：`coz_{snake_case}` → `IconCoz{PascalCase}`

| Figma `data-component-name` | 代码 `icon-name` | 转换说明 |
|------------------------------|-------------------|---------|
| `coz_long_arrow_up` | `IconCozLongArrowUp` | `coz_` → `IconCoz` + 下划线分词转帕斯卡 |
| `coz_arrow_down` | `IconCozArrowDown` | 同上 |
| `coz_ellipse` | `IconCozEllipse` | 同上 |
| `coz_people_fill` | `IconCozPeopleFill` | 同上 |

**转换步骤**：
1. 去掉 `coz_` 前缀
2. 按 `_` 拆分为单词
3. 每个单词首字母大写
4. 拼接为 `IconCoz` + PascalCase

**校验**：转换后的图标名称必须存在于 `@coze-arch/coze-design/icons` 的导出列表中。完整的可用图标列表见 `.trae/skills/components-knowledge/knowledge/coze-design/icons/icons-index.md`。

---

# 执行流程

**重要：必须严格按照以下步骤顺序执行。**

## Step 1: 获取可用组件列表与 Props

**一次性**调用 `components-knowledge` skill 获取组件库中所有可用的基础组件及其 Props/示例。

返回结果包含每个组件的：
- **组件名称 (component name)**: 如 `Button`, `Input`, `SideSheet` 等
- **组件描述 (description)**: 描述组件的用途和适用场景
- **状态描述**: 组件的不同使用场景和状态变体
- **Props 定义和示例**: 组件接受的属性和使用方式

**重要**:
- **优先使用转码产物中 `data-component-name` 标识的组件名**
- 如果转码产物未标识，则从 `components-knowledge` skill 返回的组件列表中选择
- 根据组件描述判断是否适合当前场景
- 常用布局组件: `Dialog`, `Sheet`, `Card`, `Form`, `Collapsible`, `Tabs`
- 常用表单组件: `Input`, `Select`, `Checkbox`, `RadioGroup`, `Switch`
- 常用展示组件: `Button`, `Badge`, `Tooltip`, `Avatar`

此步骤的结果在后续所有步骤中共享复用，不再重复查询。

---

## Step 2: 分析 UI 结构（内存中，不输出文件）

读取设计稿截图和高保真 JSX，在内存中完成以下分析：

### 2a. 理解整体布局
1. 理解整体布局划分
2. 明确每处基础组件候选
3. 明确页面中需要按照功能或渲染设计的独立组件（component），明确他们的功能和使用位置
4. 将整体布局、自定义组件、基础组件结合起来

### 2b. 确定组件划分

在分析过程中，对每个组件确定以下信息（无需写入文件）：
- `id`: 组件标识（英文）
- `component-name`: 大驼峰格式的代码 ComponentName
- `purpose`: 组件用途说明
- `data-node-ids`: 对应设计稿中的节点 ID
- `candidates`: 候选基础组件列表
- `figma-props`: 识别的组件 props

### 2c. 精简合并判断

在分析过程中直接完成精简：
- 合并功能一致的布局层级
- 识别可复用的重复节点（**仅**样式和功能**完全相同**的节点）
- 不确定时宁可不合并，保留独立节点

### 2d. 近似效果映射

当 `component-name` 包含以下关键词时，优先使用对应的组件库组件：

| component-name 关键词 | 对应组件库组件 |
|----------------------|--------------|
| `Table` | `Table` |
| `Tabs` / `Tab` | `Tabs` / `TabPane` |
| `Collapse` / `Accordion` | `Collapse` |
| `List` / `Menu` | `List` / `Menu` |
| `Pagination` / `Pager` | `Pagination` |
| `Breadcrumb` | `Breadcrumb` |
| `Steps` / `Step` | `Steps` |
| `Tree` | `Tree` / `TreeSelect` |
| `Tag` | `Tag` / `TagGroup` |
| `Progress` | `Progress` |
| `Modal` / `Dialog` | `Dialog` / `Modal` |

---

## Step 3: 生成业务组件代码

基于 Step 2 的内存分析结果，按依赖顺序实现业务组件（先叶子组件，再父组件）。

参考：
- 结构：Step 2 的分析结果
- 样式：`high-fidelity.jsx`（通过 `data-node-ids` 定位），**必须应用「样式方案（Tailwind CSS 规则）」中的颜色类名转换和无效类名清除**
- 基础组件：`@coze-arch/coze-design`（按 Step 2 近似效果映射规则，当 `component-name` 包含组件库关键词时优先使用对应组件库组件）
- 图标：`@coze-arch/coze-design/icons`

每个业务组件生成以下文件：
- `implement.tsx`: 组件实现代码
- `preview.tsx`: 预览入口

输出到 `apps/components-preview/src/design/<page_name>/components/<component-id>/`

## Step 4: 生成页面代码

整合业务组件生成页面代码，样式同样**必须应用「样式方案（Tailwind CSS 规则）」**。

生成以下文件：
- `page/implement.tsx`: 页面实现
- `page/preview.tsx`: 页面预览入口

输出到 `apps/components-preview/src/design/<page_name>/page/`

## Step 5: 注册预览路由

为保证产物可以直接在 `components-preview` 中预览，实现完成后**必须同步更新**路由配置文件：

- 路由文件：`apps/components-preview/src/page/config.tsx`
- 需要为当前 `page_name` 新增对应的 `preview.tsx` import
- 需要将新增的业务组件预览与页面预览注册到 `previewConfig`
- 分组结构需遵循现有约定：`UI 实现` → `<page_name>` → `Page` / `Components`

约束：
- 已存在的 import / route 不要重复添加
- 最终应保证新实现的组件可以从侧边栏进入并预览

## Step 6: 启动预览

实现完成后，**必须自动执行**以下命令启动预览环境：

```bash
cd apps/components-preview && pnpm dev
```

如果命令执行失败，需要在最终返回中明确说明失败原因。

## Step 7: 生成总结

在 `summary.md` 记录：
- 实现时间
- 组件完成统计
- 业务组件列表与设计图链接
- 使用的基础组件与图标
- 未完成项（如有）

输出到 `apps/components-preview/src/design/<page_name>/summary.md`

---

# 代码规范

## 组件引入

```tsx
// 基础组件 - 从主入口引入
import { Button } from '@coze-arch/coze-design';
import { TextArea } from '@coze-arch/coze-design';
import { Input } from '@coze-arch/coze-design';

// 图标 - 从 icons 子路径引入，所有图标均以 IconCoz 开头
import { IconCozArrowLeft, IconCozSetting, IconCozPlus } from '@coze-arch/coze-design/icons';

```

**图标命名规则**：所有图标组件以 `IconCoz` 开头，后接 PascalCase 名称。完整的可用图标列表见 `.trae/skills/components-knowledge/knowledge/coze-design/icons/icons-index.md`。

如果分析中识别的 `icon-name` 属性已经是转换后的名称（如 `IconCozLongArrowUp`），直接使用即可。

## 样式方案（Tailwind CSS 规则）

使用 Tailwind CSS，从 `high-fidelity.jsx` 提取样式时**必须**应用以下两项清洗规则。

### 颜色类名转换

Figma 导出的颜色类名（如 `text-Fg-COZ-fg-secondary/60`）**不能直接使用**，需转换为 `coz-` 开头的语义类。规则：去除 `text-`/`bg-`/`border-` 前缀及中文描述，提取 `COZ-` token 转小写，丢弃 `/opacity` 后缀。

颜色变量参考文件：`.trae/skills/components-knowledge/knowledge/tailwind/index.md`

示例：`text-Fg-COZ-fg-secondary/60` → `coz-fg-secondary`，`bg-Mg-默认-COZ-mg-primary/10` → `coz-mg-primary`

> 语义类已绑定对应 CSS 属性（`color`/`background-color`/`border-color`），无需再加 `text-`/`bg-`/`border-` 前缀。

### 去除无效类名

移除 Figma 导出中值等于 CSS 默认值的冗余类名：`justify-start`、`items-stretch`、`flex-row`、`flex-nowrap`、`self-auto`、`order-none`、`flex-grow-0`、`flex-shrink`、`basis-auto`、`static`、`visible`、`border-solid`、`bg-transparent`、`text-left`、`font-normal`、`leading-normal`、`tracking-normal`、`opacity-100`、`decoration-none`。

同时移除 Figma 导出特有的冗余模式：`font-['PingFang_SC']`（全局字体已配置）、`inline-flex`（多数应改为 `flex`）、`relative`（仅在有 `absolute` 子元素时保留）。

## 组件 Props 配置（重要）

在实现组件时，**必须**根据分析阶段识别的 `figma-props` 正确配置组件的 props。

### figma-props 解析规则

`figma-props` 格式为 `key=value,key2=value2`，需要解析并映射到组件 props：

```tsx
// 示例：figma-props="size=icon,variant=Ghost"
<Button size="icon" variant="ghost">
  <ChevronLeft className="w-4 h-4" />
</Button>
```

### Props 映射表

| figma-props 值 | 组件 prop | 说明 |
|----------------|-----------|------|
| `size=icon` | `size="icon"` | 图标按钮尺寸 |
| `size=icon-sm` | `size="icon"` + 自定义 className | 小图标按钮 |
| `size=sm` | `size="sm"` | 小尺寸 |
| `size=default` | `size="default"` | 默认尺寸 |
| `size=lg` | `size="lg"` | 大尺寸 |
| `variant=Ghost` | `variant="ghost"` | 幽灵按钮 |
| `variant=Outline` | `variant="outline"` | 边框按钮 |
| `variant=Default` | `variant="default"` | 默认样式 |
| `variant=Secondary` | `variant="secondary"` | 次要样式 |
| `variant=Link` | `variant="link"` | 链接样式 |
| `variant=Destructive` | `variant="destructive"` | 危险操作 |
| `state=Current` | 特殊处理 | 当前激活状态 |
| `state=Pressed` | 特殊处理 | 按下状态 |
| `state=Default` | 无需特殊处理 | 默认状态 |
| `status=Active` | `checked={true}` | 激活/选中状态 |

### 查询组件 Props 详情

如果不确定某个组件支持哪些 props，利用 Step 1 中已获取的组件库知识进行查询，无需再次调用 skill。

---

# 检查项

- [ ] 输入文件已读取（设计截图 + 高保真 JSX）
- [ ] 组件库信息已查询（一次性）
- [ ] UI 结构已在内存中完成分析
- [ ] 业务组件已实现
- [ ] 页面已整合
- [ ] 已更新 `apps/components-preview/src/page/config.tsx` 中的预览路由
- [ ] 产物中无 `data-node-ids` JSX 属性
- [ ] Figma 导出的颜色类名已转换为语义化工具类（无 `text-Fg-`、`bg-Bg-`、`bg-Mg-`、`border-Stroke-` 前缀残留）
- [ ] 已移除无效的默认 Tailwind 类名（`justify-start`、`font-['PingFang_SC']` 等）
- [ ] 输出 `summary.md`
- [ ] 已自动执行 `cd apps/components-preview && pnpm dev`

---

# 返回结果格式

1. 实现统计（业务组件 X/Y，页面完成状态）
2. 产物路径
3. 路由注册结果（需说明已更新 `apps/components-preview/src/page/config.tsx`）
4. 预览进展（需说明已自动执行）：

```bash
cd apps/components-preview && pnpm dev
```
