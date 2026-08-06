# UI Coding Command

读取设计分析结果，完成 UI 层代码实现，包括基础组件、业务组件和页面。

读取 `.metis/.design_analysis/pages/<page_name>/` 的分析结果，使用 `/components-knowledge` 组件库生成代码到 preview 目录。

## 重要规则

1. `data-node-ids` 仅用于实现阶段定位与样式对照；**最终输出的 JSX/TSX 不得包含 `data-node-ids` 属性**。

## 调用方式

```bash
/ui-workflow ui-coding <page_name>
/ui-workflow ui-coding <page_name> --components-only
/ui-workflow ui-coding <page_name> --page-only
```

参数说明：
- `page_name`: 设计稿名称（与 analyze-design 输出目录名一致）
- `--components-only`: 仅实现业务组件
- `--page-only`: 仅实现页面整合
- 两个选项不可同时使用

## 输入与输出

输入目录：从 `.metis/.design_analysis/pages/<page_name>/` 读取：
| 文件 | 用途 |
|------|------|
| `ui.implement.xml` | 精简结构，确定组件列表和层级关系 |
| `ui.md` | 分析结论，了解组件用途、设计图 URL 和实现建议 |
| `high-fidelity.jsx` | 高保真代码，作为样式和布局参考 |

输出到 `apps/components-preview/src/design/<page_name>/`：

```
<page_name>/
├── components/               # 业务组件
│   └── <component-id>/
│       ├── implement.tsx     # 组件实现代码
│       ├── preview.tsx       # 预览入口
│       └── implement.md      # 实现记录
├── page/                     # 页面
│   ├── implement.tsx         # 页面实现
│   └── preview.tsx           # 页面预览入口
└── summary.md                # 实现总结
```
## 执行流程

## Step 1: 校验与解析

1. 校验输入文件是否存在；缺失则提示先执行 `/ui-workflow ui-analyze`。
2. 从 `ui.implement.xml` 解析：
   - 业务组件：`id`、`component-name`、`purpose`、`data-node-ids`
   - 基础组件：`id`、`candidates`、`figma-props`
   - 页面：`id`、`name`

## Step 2: 查询组件库

1. 调用 `/components-knowledge` skill 获取可用组件列表。
2. 对 `candidates` 调用 `/components-knowledge` skill 获取组件 Props/示例。
3. 根据 `ui.implement.xml` 中每个 `<component>` 的 `component-name` 属性，按近似效果映射到组件库组件（参见下方映射规则）。

### 近似效果映射规则

实现业务组件时，需根据 `ui.implement.xml` 中 `<component>` 节点的 **`component-name`** 名称识别是否应使用组件库中的对应组件作为核心实现。

判断依据——当 `component-name` 包含以下关键词时，优先使用对应的组件库组件：
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

示例：

```xml
<component id="model-table" component-name="ModelTable" ...>
```

→ `ModelTable` 包含关键词 `Table`，应调用 `/components-knowledge` skill 查询 `Table` 组件用法，使用 `Table` 组件实现而非 `<div>` + Tailwind 手写表格。

映射时应调用 `/components-knowledge` skill 确认组件库中是否存在对应组件，若存在则优先使用组件库实现，而非用原生 HTML 标签 + Tailwind 手写近似效果。

## Step 3: 生成业务组件

跳过条件：`--page-only`

1. 按依赖顺序实现业务组件（先叶子组件，再父组件）。
2. 参考：
   - 结构：`ui.implement.xml`
   - 样式：`high-fidelity.jsx`（可通过 `data-node-ids` 定位），**必须应用「样式方案（Tailwind CSS 规则）」中的颜色类名转换和无效类名清除**
   - 基础组件：`@coze-arch/coze-design`（按 Step 2 近似效果映射规则，当 `component-name` 包含组件库关键词时优先使用对应组件库组件）
   - 图标：`@coze-arch/coze-design/icons`
3. 生成 `implement.tsx`、`preview.tsx`、`implement.md`。
4. `implement.tsx` 中**不要输出** `data-node-ids`。

## Step 4: 生成页面（可跳过）

跳过条件：`--components-only`

1. 整合业务组件生成 `page/implement.tsx` 与 `page/preview.tsx`，样式同样**必须应用「样式方案（Tailwind CSS 规则）」**。
2. `page/implement.tsx` 中**不要输出** `data-node-ids`。

## Step 5: 注册预览路由

为保证产物可以直接在 `components-preview` 中预览，实现完成后**必须同步更新**路由配置文件：

- 路由文件：`apps/components-preview/src/page/config.tsx`
- 需要为当前 `page_name` 新增对应的 `preview.tsx` import
- 需要将新增的业务组件预览与页面预览注册到 `previewConfig`
- 分组结构需遵循现有约定：`UI 实现` → `<page_name>` → `Page` / `Components`

约束：
- `--components-only` 时，只注册业务组件预览，不新增页面预览项
- `--page-only` 时，只注册页面预览；若对应页面分组不存在，可补齐最小必要分组
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

如果 `ui.implement.xml` 中的 `icon-name` 属性已经是转换后的名称（如 `IconCozLongArrowUp`），直接使用即可。

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

在实现组件时，**必须**根据 `ui.implement.xml` 中节点的 `figma-props` 属性正确配置组件的 props。

### figma-props 解析规则

`figma-props` 属性格式为 `key=value,key2=value2`，需要解析并映射到组件 props：

```xml
<!-- 示例：ui.implement.xml 中的节点 -->
<base-component id="back-btn" candidates="Button" figma-props="size=icon,variant=Ghost" />
```

解析后应用到组件：
```tsx
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

如果不确定某个组件支持哪些 props，调用 `/components-knowledge` skill 查询，返回的结果包含完整的 Props 定义和使用示例。

## 检查项

- [ ] 输入分析结果已加载
- [ ] 组件库信息已查询
- [ ] 业务组件已实现或按参数跳过
- [ ] 页面已整合或按参数跳过
- [ ] 已更新 `apps/components-preview/src/page/config.tsx` 中的预览路由
- [ ] 产物中无 `data-node-ids` JSX 属性
- [ ] Figma 导出的颜色类名已转换为语义化工具类（无 `text-Fg-`、`bg-Bg-`、`bg-Mg-`、`border-Stroke-` 前缀残留）
- [ ] 已移除无效的默认 Tailwind 类名（`justify-start`、`font-['PingFang_SC']` 等）
- [ ] 输出 `summary.md`
- [ ] 已自动执行 `cd apps/components-preview && pnpm dev`
- [ ] 代码符合 `constitution.md`

## 返回结果格式

1. 实现统计（业务组件 X/Y，页面完成状态）
2. 产物路径
3. 路由注册结果（需说明已更新 `apps/components-preview/src/page/config.tsx`）
4. 预览进展（需说明已自动执行）：

```bash
cd apps/components-preview && pnpm dev
```
