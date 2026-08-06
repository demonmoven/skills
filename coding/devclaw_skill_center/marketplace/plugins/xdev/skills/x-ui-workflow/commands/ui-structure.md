# UI Structure Analyzer Command

根据设计信息，对页面 UI 层进行布局和组件结构分析。

你是一个专业的前端工程师，根据可以得到的需求文档、设计稿、设计稿转码产物等输入，分析设计稿实现方案，给出布局划分、组件选择等关键决策，辅助技术方案的生成。

**注意**: 本 command 需要通过 `components-knowledge` skill 获取可用组件列表，以确保候选组件名称的准确性。

# 参数
- `page_name`: 设计稿名称
- `figma_url`: 设计稿的 figma 链接
- `design_image`: 设计稿 figma 截图文件路径 (**必需**, 由主 Command 准备)
- `jsx_code`: 设计稿 jsx 高保真实现文件路径 (**必需**, 由主 Command 准备)
- `description`: (可选参数) 需求规格里的相关说明，和对分割的特殊要求

## 参数验证
- `design_image` 必须是有效的本地文件路径
- `jsx_code` 必须是有效的本地文件路径
- 如果参数缺失，返回错误信息，由主 Command 处理


# 注意事项
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

**在 XML 中的使用**：
```xml
<!-- element-type="icon" 的 icon-name 属性必须使用转换后的代码组件名 -->
<element id="back-icon" element-type="icon" icon-name="IconCozLongArrowUp" data-node-ids="..." />
```

# 执行流程

**重要：必须严格按照以下步骤顺序执行。**

## Step 1: 获取可用组件列表

**首先**调用 `components-knowledge` skill 获取组件库中所有可用的基础组件。

返回结果包含每个组件的：
- **组件名称 (component name)**: 如 `Button`, `Input`, `SideSheet` 等，用于 `candidates` 属性
- **组件描述 (description)**: 描述组件的用途和适用场景
- **状态描述**: 组件的不同使用场景和状态变体

**重要**:
- **优先使用转码产物中 `data-component-name` 标识的组件名**
- 如果转码产物未标识，则从 `components-knowledge` skill 返回的组件列表中选择
- 根据组件描述判断是否适合当前场景
- 常用布局组件: `Dialog`, `Sheet`, `Card`, `Form`, `Collapsible`, `Tabs`
- 常用表单组件: `Input`, `Select`, `Checkbox`, `RadioGroup`, `Switch`
- 常用展示组件: `Button`, `Badge`, `Tooltip`, `Avatar`

---

## Step 2: 生成页面完整结构 (ui.xml)

1. 理解整体布局
2. 明确每处基础组件候选
3. 明确页面中需要按照功能或者渲染，设计的独立组件，明确他们的功能和使用位置
4. 将整体布局、自定义组件、基础组件结合起来，补充布局类型组件等内容

### XML Schema 说明
#### 节点类型说明
- `<page>`: 页面根节点
- `<layout>`: 布局区域节点
- `<component>`: 自定义组件节点
- `<base-component>`: 基础组件节点（来自组件库）
- `<element>`: 原子元素节点（文本、图标、图片等）

#### 属性说明
- `id`: 节点唯一标识（英文）
- `name`: 节点语义化名称（中文）
- `data-node-ids`: 对应设计稿转码中的 data-node-id 属性值，多个 data-node-id 使用 "," 分隔
- `element-type`: 元素类型（仅 `<element>` 节点使用），可选值：text/icon/image/svg-resource
- `candidates`: 候选组件列表（仅 `<base-component>` 使用），**优先使用转码产物中 `data-component-name` 的值**，多个组件用逗号分隔，按优先级排序
- `figma-component`: 转码产物中 `data-component-name` 的原始值（如有）
- `figma-props`: 转码产物中识别的组件 props（如有），格式为 `prop1=value1,prop2=value2`
- `component-name`: 作为代码 ComponentName 使用，大驼峰格式（仅 `<component>` 使用）
- `purpose`: 组件用途说明（仅 `<component>` 使用）
- `is-layout`: 是否为布局组件（可选，默认 false）
- `icon-name`: 图标名称（仅 `element-type="icon"` 时使用），来自 `data-component-name`
- `desc`: 节点补充说明（可选）

### 关键说明
1. **节点命名**: 使用语义化的中文名称，清晰描述节点用途
2. **层级关系**: 通过 XML 嵌套表达父子关系，支持多级嵌套
3. **布局组件处理**: 当使用 Dialog、Sheet、Card、Form 等自带布局的组件时，将其标记为 `is-layout="true"`，其内容作为子节点
4. **data-node-id 映射**: 每个节点都必须提供对应的 `data-node-id` 属性值，用于与设计稿转码产物关联
5. **优先使用转码识别的组件**: 如果节点有 `data-component-name` 属性，优先使用该值作为 `candidates` 的首选，并记录在 `figma-component` 属性中
6. **记录 Figma Props**: 如果节点有 `data-variant`、`data-size` 等属性，记录在 `figma-props` 属性中，供后续实现参考
7. **图标处理**:
   - 如果 SVG 外层有 `data-svg-wrapper` 且 `data-component-name` 包含图标名，使用 `element-type="icon"` 并记录 `icon-name`
   - 如果是普通 SVG 资源，使用 `element-type="svg-resource"`
8. **组件目的**: 对于自定义组件，在 `purpose` 属性中说明其封装的功能和交互能力
9. **紧凑性**: 对于无子节点的元素，使用自闭合标签 `<element ... />`；可适当添加 XML 注释增强可读性

### 完成度自检
1. [ ] 检查是否已包含全部元素
2. [ ] 检查基础组件候选是否完整
3. [ ] 检查布局功能基础组件是否考虑到
4. [ ] 检查需要独立实现的组件是否完整
5. [ ] 检查布局是否合理，检查整体是否由布局功能基础组件实现

### 保存结果
- UI 结构描述（XML 格式）`/.design_analysis/pages/<page_name>/ui.xml`

---

## Step 3: 生成页面精简结构 (ui.implement.xml)

完整结构分解粒度较细，需要生成一个**精简组件结构**，让结构更凸显：
- **整体布局划分**
- **组件划分**（component）
- **基础组件引入点**（base-component）

### 精简方向

#### 方向 1: 合并功能一致的布局层级

将多个纯布局功能的层级合并，减少嵌套深度：
- 连续的 `layout` 节点，如果只是容器嵌套，可以合并为一个
- 保留有明确语义的布局节点（如 header、content、footer 等区域划分）
- 保留包含 `component` 或 `base-component` 的关键层级

#### 方向 2: 合并实现相同的重复节点

**⚠️ 警告：必须特别谨慎，避免不必要的精简影响最终效果。**

只有**完全相同实现**的节点才能合并，需要满足：
- **样式完全一致**：背景色、边框、尺寸、间距等样式属性完全相同
- **功能完全一致**：交互行为、状态处理完全相同
- **仅内容元素有差异**：如文本内容、图片资源等

**不能合并的情况**（即使结构相似）：
- 背景色不同（可能需要不同的 `variant` 或 `type` 属性）
- 尺寸不同（可能需要不同的 `size` 属性）
- 状态不同（如 active/inactive、selected/unselected）
- 任何需要通过组件属性差异来控制的样式差异

合并规则：
- 可以合并为一个代表节点
- **保留一个 `data-node-ids` 作为示例**，不需要合并所有的 node-ids
- 在节点上添加 `repeat-count` 属性标注重复次数
- 在节点上添加 `repeat-note` 属性说明重复内容的差异（仅限内容差异）
- 实现时通过**组件复用**或**代码重复**来还原完整页面

**重要**：
- 合并重复节点时必须给出说明，避免完整页面生成时内容有缺失
- 如果有任何样式/属性差异，**不要合并**，保留独立节点
- 属性使用的精准性需要独立校验，不确定时宁可不合并

### 精简规则

1. **必须保留**所有的 `component` 自定义组件类型节点
2. **必须保留**所有需要引入的 `base-component` 节点
3. 对 `base-component` 通用基础组件做适当的合并
   1. 如果节点内只有简单的 `layout` 和 `element` 节点，且节点数量不大时，可以直接合并实现
   2. 如果节点内部有 `component`，或者其他复杂组合时，可以分层分别实现
   3. 根据组件实际情况合理判断
      - 例如 `Radio` 和 `RadioItem` 组件，会被分解为两层实现，但是实际他们可以一起实现
      - 例如 `FormItem` 通常整体作为 form 处理，可以独立处理内容节点
      - 例如 `Card` 分为 title、content 等多个区域，在实现时需要调整结构
4. **不要创建新的节点**，只使用原来的节点，节点 id 和属性保持不变，否则会无法获取节点相关的分析数据
5. **谨慎合并重复节点**：
   - 只有样式和功能**完全相同**的节点才能合并
   - 任何样式差异（背景色、尺寸、状态等）都**不能合并**
   - 合并时选择其中一个作为代表节点
   - **保留该代表节点的 `data-node-ids`**，不需要合并所有重复节点的 node-ids
   - 添加 `repeat-count` 和 `repeat-note` 属性
   - **不确定时宁可不合并**，保留独立节点更安全

### 精简示例

```xml
<!-- 原始结构：3 个相同的 Tab 项 -->
<layout id="tabs" name="标签栏">
  <base-component id="tab-1" name="标签1" candidates="Tab" data-node-ids="1:100">
    <element id="tab-1-text" element-type="text"/>
  </base-component>
  <base-component id="tab-2" name="标签2" candidates="Tab" data-node-ids="1:101">
    <element id="tab-2-text" element-type="text"/>
  </base-component>
  <base-component id="tab-3" name="标签3" candidates="Tab" data-node-ids="1:102">
    <element id="tab-3-text" element-type="text"/>
  </base-component>
</layout>

<!-- 精简结构：合并为 1 个代表节点，保留一个 data-node-ids 作为示例 -->
<layout id="tabs" name="标签栏">
  <base-component id="tab-1" name="标签项" candidates="Tab" data-node-ids="1:100"
                  repeat-count="3" repeat-note="3 个标签项，仅文本内容不同"/>
</layout>
```

### 保存结果
- UI 精简结构描述（XML 格式）`/.design_analysis/pages/<page_name>/ui.implement.xml`

---

## Step 4: 基于精简结构生成分析结论 (ui.md)

**重要：此步骤必须在 Step 2 完成后执行，分析结论必须基于精简 XML (`ui.implement.xml`) 生成。**

基于已生成的精简 XML 结构，生成最终的 UI 分析结论文档。

### ui.md 文档结构

```markdown
# <page_name> UI 结构分析

## 基本信息
- **页面名称**: <page_name>
- **Figma 链接**: <figma_url>
- **分析时间**: <timestamp>

## 整体布局

[描述页面的整体布局结构，主要区域划分]

## 组件列表

### 页面整体实现
| 节点 ID | 名称 | 类型 | data-node-ids | 说明 |
|---------|------|------|---------------|------|
| <page_id> | <name> | page | <node-ids> | 页面根节点 |

### 自定义组件 (component)
| 节点 ID | 组件名称 | 名称 | data-node-ids | 用途说明 |
|---------|----------|------|---------------|----------|
| <id> | <ComponentName> | <name> | <node-ids> | <purpose> |

### 基础组件 (base-component)
| 节点 ID | 名称 | 候选组件 | data-node-ids | 是否布局组件 |
|---------|------|----------|---------------|-------------|
| <id> | <name> | <candidates> | <node-ids> | <is-layout> |

## 复用组件说明

[列出需要通过组件复用实现完整页面的节点，确保页面内容不缺失]

| 节点 ID | 名称 | 重复次数 | 复用说明 |
|---------|------|----------|----------|
| <id> | <name> | <repeat-count> | <repeat-note> |

**实现方式**：
- 这些节点在精简结构中只保留了一个代表节点
- 实现时需要通过循环渲染或代码复制来还原完整页面
- 每个节点的差异主要在内容元素上（如文本、图片等）

## 实现建议

[基于分析给出的实现建议，包括：]
- 布局实现方案
- 组件复用建议
- 注意事项
```

### 关键要求
1. **组件列表必须与精简 XML 对应**：只列出 `ui.implement.xml` 中的关键节点
2. **节点分类**：
   - 页面整体实现：`<page>` 节点
   - 自定义组件：所有 `<component>` 节点
   - 基础组件：所有 `<base-component>` 节点
3. **复用组件说明必填**：如果有合并的重复节点，必须在"复用组件说明"部分列出，方便构建最终页面效果
4. **data-node-ids 保留示例**：合并的重复节点只保留一个代表节点的 data-node-ids

### 保存结果
- UI 分析结果，保存到 `/.design_analysis/pages/<page_name>/ui.md`

---


# 产物自检

1. [ ] **验证已调用 `get_components_list`** - 确保组件名称来自组件库
2. [ ] **验证 `candidates` 组件名称准确** - 所有 `base-component` 的 `candidates` 属性值必须与组件库一致
3. [ ] **验证 `ui.xml` 文件已生成** - 完整结构，包含所有节点
4. [ ] **验证 `ui.implement.xml` 文件已生成** - 精简结构
5. [ ] **验证精简结构凸显关键节点**：整体布局划分、component、base-component
6. [ ] **验证重复节点已标注**：合并的重复节点有 `repeat-count` 和 `repeat-note` 属性，只保留一个 `data-node-ids`
7. [ ] **验证 `ui.md` 文件已生成** - 基于精简 XML 的分析结论
8. [ ] **验证 `ui.md` 中的组件列表与 `ui.implement.xml` 一致**
9. [ ] **验证 `ui.md` 中有复用组件说明**：如有重复节点，必须在"复用组件说明"部分列出
10. [ ] **验证节点预处理已完成** - `nodes/nodes.md` 已生成
11. [ ] **验证 `ui.md` 中 `tos_url` 列已补充完整**（关键：确保脱离本地缓存后有效）

---

# 返回结果

所有保存的文件结果：
- UI 结构描述（XML 格式）`/.design_analysis/pages/<page_name>/ui.xml`
- UI 精简结构描述（XML 格式）`/.design_analysis/pages/<page_name>/ui.implement.xml`
- UI 分析结果 `/.design_analysis/pages/<page_name>/ui.md`
- 节点预处理结果 `/.design_analysis/pages/<page_name>/nodes/nodes.md`
