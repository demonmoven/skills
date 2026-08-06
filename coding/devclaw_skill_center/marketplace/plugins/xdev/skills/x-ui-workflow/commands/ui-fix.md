# X-UI-Fix Command

根据 Figma 设计稿，将指定目录中的 UI 代码修改为与设计稿视觉一致的实现。

## 重要规则

1. 开始前必须读取 `.claude/docs/constitution.md`。了解本工程的技术规范，后续代码实现需要遵循规范

## 调用方式

```bash
/ui-workflow ui-fix "<需求描述>" <figma_url> <target_directory>
```

### 参数说明

- `需求描述`：本次修改的目标说明，例如 "将顶部导航栏改为设计稿样式"
- `figma_url`：Figma 设计稿链接，支持完整页面或带 node-id 的局部区域
- `target_directory`：需要修改的代码文件目录路径（相对于仓库根目录）


### 示例

```bash
/ui-workflow ui-fix "将侧边栏改为新版设计稿样式" https://www.figma.com/design/xxx?node-id=123:456 apps/coze-studio/src/components/sidebar

/ui-workflow ui-fix "对齐 Header 组件与设计稿" https://www.figma.com/design/xxx?node-id=789:012 packages/ui/src/header
```

---

## 执行流程

## Step 1: 解析参数，读取目标目录现有代码

1. 解析三个参数：需求描述、Figma URL、目标目录
2. 用 Glob 列出目标目录中所有 `.tsx`/`.ts`/`.css`/`.scss` 文件
3. 读取核心文件，了解现有实现结构和组件职责

## Step 2: 获取 Figma 高保真 JSX

调用 `figma-to-code` 获取高保真 JSX 代码：

```bash
npx -p @byted/x-figma-to-code figma-to-code "<figma_url>" \
  --framework HTML \
  --html-mode jsx \
  --embed-vectors \
  --show-node-ids \
  --optimize-layout
```

从高保真 JSX 中提取：
- 整体布局结构（flex/grid 方向、间距、对齐）
- 颜色、字体、圆角、阴影等视觉样式
- 图标使用情况
- 组件层级关系

## Step 3: 获取仓库 Components，映射设计元素

**3a. 获取组件列表**

调用 `components-knowledge` skill 获取所有可用组件。

**3b. 映射 Figma 元素到组件**

分析高保真 JSX 和目标目录中已有的业务组件，将 UI 元素映射到 Components：

| Figma 元素特征 | 对应组件 |
|---------------|---------|
| 按钮类元素 | Button |
| 文本输入框 | Input / Textarea |
| 下拉选择 | Select / Dropdown |
| 开关/切换 | Switch / Toggle |
| 对话框/弹窗 | Dialog / Modal |
| 标签页 / tab 切换结构 | Tabs / TabPane |
| 行列式数据展示（表头+数据行） | Table |
| 可折叠/展开的内容区域 | Collapse / Accordion |
| 垂直项目列表且每项可点击 | List / Menu |
| 分页控件 | Pagination |
| 面包屑导航 | Breadcrumb |
| 步骤流程指示 | Steps |
| 树形层级结构 | Tree / TreeSelect |
| 标签/筛选条 | Tag / TagGroup |
| 进度指示 | Progress |
| 徽标/角标 | Badge |
| 提示文本 | Tooltip |
| 加载态 | Spinner / Skeleton |
| 图标 | @coze-coding/icons |

**近似效果映射原则**：当目标目录中业务组件的名称包含组件库已有组件的关键词（如组件名含 `Table`、`Tabs`、`Modal` 等），**必须优先使用组件库对应组件实现**，而非用原生 HTML 标签 + Tailwind 手写近似效果。例如：
- 组件名为 `ModelTable` → 包含 `Table`，使用组件库 `Table` 组件
- 组件名为 `ConfigTabs` → 包含 `Tabs`，使用组件库 `Tabs` / `TabPane` 组件
- 组件名为 `DetailCollapse` → 包含 `Collapse`，使用组件库 `Collapse` 组件

**3c. 查询目标组件详情**

对需要使用的组件调用 `components-knowledge` skill 获取详情。

**3d. 查询图标（如有需要）**

调用 `components-knowledge` skill 查询图标。

## Step 4: 分析组件拆分（新增 UI 内容时必做）

**在动手写代码之前**，先分析 Figma 设计稿中新增的 UI 区域，判断是否需要拆分子组件。

#### 拆分判断标准

**同时满足以下所有条件**，才应将该区域提取为独立组件文件：

- 视觉上是一个可复用的独立单元（如卡片、列表项、头部区域）
- 提取后的子组件代码量 **≥ 30 行**（不足 30 行说明拆分粒度过细，直接内联即可）
- 父组件因此变得更简洁，而不是增加了复杂度

**不要拆分**的情况：

- 提取后的子组件不足 30 行——直接内联在父组件里
- 仅仅是几个 div + 文字的简单展示区域
- 只在一处使用且无内部逻辑的纯样式片段

#### 拆分原则

- **每个子组件单独成文件**：放在同目录的 `components/` 子目录下，或与目标文件同级
- **父组件只做组合**：父文件中只保留子组件的引用和数据传递，不内联大量 JSX
- **Props 设计精简**：子组件只接收渲染所需的最小数据，业务回调向上传递

#### 拆分示例

❌ **不好的做法**：把所有 JSX 塞进父组件
```tsx
// reward-points-modal.tsx（父组件）
<motion.div key="step2">
  {/* 100 行卡片列表 JSX 直接堆在这里 */}
  <div className="grid grid-cols-2 gap-4">
    {cases.map(item => (
      <button key={item.id}>
        <div>... 大量内联 JSX ...</div>
      </button>
    ))}
  </div>
</motion.div>
```

✅ **正确的做法**：提取子组件
```tsx
// components/recommend-case-card.tsx（新文件）
interface RecommendCaseCardProps {
  item: RecommendCase;
  onClick: (item: RecommendCase) => void;
}
export function RecommendCaseCard({ item, onClick }: RecommendCaseCardProps) {
  return <button onClick={() => onClick(item)}>...</button>;
}

// reward-points-modal.tsx（父组件，保持简洁）
import { RecommendCaseCard } from './components/recommend-case-card';
<motion.div key="step2">
  <div className="grid grid-cols-2 gap-4">
    {cases.map(item => (
      <RecommendCaseCard key={item.id} item={item} onClick={handleCaseClick} />
    ))}
  </div>
</motion.div>
```

## Step 5: 实施代码修改

基于以下信息综合修改代码：
- **需求描述**：明确修改范围和目标
- **现有代码**：保持组件接口、业务逻辑、数据流不变
- **高保真 JSX**：作为布局和样式参考
- **Components Knowledge**：替换或新增使用仓库内组件
- **Step 4 的拆分计划**：先创建子组件文件，再在父组件中引用

#### 修改原则

1. **组件拆分优先**：新增的独立 UI 区域提取为子组件，不在父组件内堆砌 JSX
2. **保留业务逻辑**：只修改 UI 层，不改变 props 接口、state 管理、事件处理逻辑
3. **优先使用 Components Knowledge**：将 Figma 中的 UI 元素替换为 `@coze-arch/coze-design` 对应组件
4. **样式使用 Tailwind CSS**：使用 `cn()` 工具函数合并类名，**必须应用下方「Tailwind CSS 清洗规则」**
5. **图标使用 `@coze-coding/icons`**：替换内联 SVG 或其他图标库

#### 组件引入规范

```tsx
// Core Components - 从子路径引入
import { Button,Input,Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@coze-arch/coze-design';

// 图标 - 从主入口引入
import { ArrowLeft, Settings } from '@coze-arch/coze-design/icons';

```

#### 执行顺序

1. 先创建所有子组件文件（Write 工具）
2. 再修改父组件，引入子组件（Edit 工具）
3. 确保每个文件修改后代码语法正确

## Step 6: 输出修改总结

1. **修改文件清单**：列出所有修改过的文件及修改内容摘要
2. **组件使用情况**：新引入的 Core Components 和图标
3. **设计稿对照**：说明哪些设计元素已还原，哪些有所调整及原因

---

## Tailwind CSS 清洗规则

从高保真 JSX 提取样式时**必须**应用以下两项清洗规则。

### 颜色类名转换

Figma 导出的颜色类名（如 `text-Fg-COZ-fg-secondary/60`）**不能直接使用**，需转换为 `coz-` 开头的语义类。规则：去除 `text-`/`bg-`/`border-` 前缀及中文描述，提取 `COZ-` token 转小写，丢弃 `/opacity` 后缀。

颜色变量参考文件：`.trae/skills/components-knowledge/knowledge/tailwind/index.ts`

示例：`text-Fg-COZ-fg-secondary/60` → `coz-fg-secondary`，`bg-Mg-默认-COZ-mg-primary/10` → `coz-mg-primary`

> 语义类已绑定对应 CSS 属性（`color`/`background-color`/`border-color`），无需再加 `text-`/`bg-`/`border-` 前缀。

### 去除无效类名

移除 Figma 导出中值等于 CSS 默认值的冗余类名：`justify-start`、`items-stretch`、`flex-row`、`flex-nowrap`、`self-auto`、`order-none`、`flex-grow-0`、`flex-shrink`、`basis-auto`、`static`、`visible`、`border-solid`、`bg-transparent`、`text-left`、`font-normal`、`leading-normal`、`tracking-normal`、`opacity-100`、`decoration-none`。

同时移除 Figma 导出特有的冗余模式：`font-['PingFang_SC']`（全局字体已配置）、`inline-flex`（多数应改为 `flex`）、`relative`（仅在有 `absolute` 子元素时保留）。

---

## 关键注意事项

- **不改变组件接口**：保持 props、回调函数签名不变
- **颜色变量**：严格按「Tailwind CSS 清洗规则」转换颜色类名，使用 `coz-` 语义类，而非硬编码颜色值或 Figma 原始 token
- **保持样式方案一致**：若现有代码使用 CSS Modules / styled-components，保持相同方案

## 检查项

- [ ] 目标目录现有代码已读取并理解
- [ ] 高保真 JSX 已获取并解析布局/样式信息
- [ ] Components Knowledge 已查询，Figma 元素已完成映射
- [ ] 新增 UI 区域已完成组件拆分分析，独立单元提取为子组件文件
- [ ] 子组件文件已创建（如有拆分）
- [ ] 父组件修改已实施，引用子组件，业务逻辑保留不变
- [ ] 修改后代码无语法错误
- [ ] Figma 导出的颜色类名已转换为 `coz-` 语义类，无效默认类名已移除
- [ ] 输出修改总结
