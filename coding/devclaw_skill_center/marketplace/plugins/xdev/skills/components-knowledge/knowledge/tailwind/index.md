# Tailwind 颜色类名转换规则

Figma 导出的 `high-fidelity.jsx` 中的颜色类名使用 Figma token 命名，不能直接用于项目代码。必须按以下规则转换为项目预设的 `coz-` 语义化工具类。

## 转换核心逻辑

从 Figma 类名中找到 `COZ-` 开头的 token，转为小写 `coz-` 开头，去掉其他所有前缀、中文描述和 `/opacity` 后缀。

```
Figma 类名格式:  {tw-prefix}-{可选中文描述}-COZ-{token}/{opacity}
转换目标:        coz-{token}
```

## 前景色（文字颜色）

Figma 中以 `text-Fg-` 开头（有时也出现 `bg-Fg-` 用于 SVG 填充）。转换后的语义类直接设置 `color` 属性，不需要 `text-` 前缀。

| Figma 导出 | 转换结果 |
|---|---|
| `text-Fg-COZ-fg-primary/80` | `coz-fg-primary` |
| `text-Fg-COZ-fg-secondary/60` | `coz-fg-secondary` |
| `text-Fg-COZ-fg-plus/90` | `coz-fg-plus` |
| `text-Fg-COZ-fg-dim/40` | `coz-fg-dim` |
| `text-Fg-COZ-fg-white` | `coz-fg-white` |
| `text-Fg-高亮-COZ-fg-hglt` | `coz-fg-hglt` |
| `text-Fg-COZ-fg-hglt-plus` | `coz-fg-hglt-plus` |
| `text-Fg-COZ-fg-hglt-ai` | `coz-fg-hglt-ai` |
| `text-Fg-COZ-fg-hglt-red` | `coz-fg-hglt-red` |
| `text-Fg-COZ-fg-hglt-yellow` | `coz-fg-hglt-yellow` |
| `text-Fg-COZ-fg-hglt-green` | `coz-fg-hglt-green` |
| `bg-Fg-COZ-fg-secondary/60` | `coz-fg-secondary`（SVG 填充场景） |
| `bg-Fg-COZ-fg-dim/40` | `coz-fg-dim`（SVG 填充场景） |

## 背景色（页面/容器背景）

Figma 中以 `bg-Bg-` 开头。转换后的语义类直接设置 `background-color`，不需要 `bg-` 前缀。

| Figma 导出 | 转换结果 |
|---|---|
| `bg-Bg-COZ-bg-plus` | `coz-bg-plus` |
| `bg-Bg-COZ-bg` | `coz-bg` |
| `bg-Bg-COZ-bg-primary` | `coz-bg-primary` |
| `bg-Bg-COZ-bg-secondary` | `coz-bg-secondary` |
| `bg-Bg-COZ-bg-max` | `coz-bg-max` |

## 中间层色（按钮/卡片/标签背景）

Figma 中以 `bg-Mg-` 开头，通常还包含中文描述（如「高亮」「默认」「强」「次」）。转换后的语义类直接设置 `background-color`，不需要 `bg-` 前缀。

| Figma 导出 | 转换结果 |
|---|---|
| `bg-Mg-高亮-COZ-mg-hglt/30` | `coz-mg-hglt` |
| `bg-Mg-默认-COZ-mg-primary/10` | `coz-mg-primary` |
| `bg-Mg-强-COZ-mg-plus/10` | `coz-mg-plus` |
| `bg-Mg-次-COZ-mg-secondary/10` | `coz-mg-secondary` |
| `bg-Mg-COZ-mg` | `coz-mg` |
| `bg-Mg-COZ-mg-card` | `coz-mg-card` |
| `bg-Mg-高亮-COZ-mg-hglt-secondary` | `coz-mg-hglt-secondary` |
| `bg-Mg-高亮-COZ-mg-hglt-plus` | `coz-mg-hglt-plus` |

## 描边色（边框颜色）

Figma 中以 `border-Stroke-` 开头。转换后的语义类直接设置 `border-color`，不需要 `border-` 前缀。

| Figma 导出 | 转换结果 |
|---|---|
| `border-Stroke-COZ-stroke-primary/10` | `coz-stroke-primary` |
| `border-Stroke-COZ-stroke-hglt` | `coz-stroke-hglt` |
| `border-Stroke-COZ-stroke-plus` | `coz-stroke-plus` |
| `border-Stroke-COZ-stroke-opaque` | `coz-stroke-opaque` |
| `border-Stroke-COZ-stroke-max` | `coz-stroke-max` |

> 注意：`border`（边框宽度）仍然需要保留，只是颜色部分用 `coz-stroke-*` 替换。

## 完整转换示例

Figma 导出的原始代码：

```html
<div class="text-Fg-COZ-fg-secondary/60 text-sm bg-Mg-默认-COZ-mg-primary/10 border border-Stroke-COZ-stroke-primary/10 font-['PingFang_SC'] justify-start relative">
  <span class="text-Fg-高亮-COZ-fg-hglt text-xs">高亮文字</span>
</div>
```

转换后的代码：

```tsx
<div className="coz-fg-secondary text-sm coz-mg-primary border coz-stroke-primary">
  <span className="coz-fg-hglt text-xs">高亮文字</span>
</div>
```

转换要点：
1. `text-Fg-COZ-fg-secondary/60` → `coz-fg-secondary`（去前缀 + 去透明度）
2. `bg-Mg-默认-COZ-mg-primary/10` → `coz-mg-primary`（去前缀 + 去中文 + 去透明度）
3. `border-Stroke-COZ-stroke-primary/10` → `coz-stroke-primary`（去前缀 + 去透明度，保留 `border` 宽度类）
4. `text-Fg-高亮-COZ-fg-hglt` → `coz-fg-hglt`（去前缀 + 去中文）
5. `font-['PingFang_SC']` → 删除（全局字体已配置）
6. `justify-start` → 删除（flex 默认值）
7. `relative` → 删除（无 absolute 子元素时不需要）

## 所有可用的语义类速查

### 前景色 coz-fg-*（设置 color）

`coz-fg` · `coz-fg-primary` · `coz-fg-secondary` · `coz-fg-dim` · `coz-fg-plus` · `coz-fg-white` · `coz-fg-white-dim` · `coz-fg-hglt` · `coz-fg-hglt-dim` · `coz-fg-hglt-plus` · `coz-fg-hglt-plus-dim` · `coz-fg-hglt-ai` · `coz-fg-hglt-ai-dim` · `coz-fg-hglt-red` · `coz-fg-hglt-red-dim` · `coz-fg-hglt-yellow` · `coz-fg-hglt-yellow-dim` · `coz-fg-hglt-green` · `coz-fg-hglt-green-dim`

### 背景色 coz-bg-*（设置 background-color）

`coz-bg` · `coz-bg-primary` · `coz-bg-secondary` · `coz-bg-plus` · `coz-bg-max`

### 中间层色 coz-mg-*（设置 background-color）

`coz-mg` · `coz-mg-primary` · `coz-mg-secondary` · `coz-mg-plus` · `coz-mg-card` · `coz-mg-mask` · `coz-mg-hglt` · `coz-mg-hglt-plus` · `coz-mg-hglt-secondary` · `coz-mg-hglt-ai` · `coz-mg-hglt-red` · `coz-mg-hglt-yellow` · `coz-mg-hglt-green` · `coz-mg-color-plus-brand`

含交互态后缀：`-pressed` · `-hovered` · `-dim`

### 描边色 coz-stroke-*（设置 border-color）

`coz-stroke-primary` · `coz-stroke-plus` · `coz-stroke-hglt` · `coz-stroke-opaque` · `coz-stroke-max` · `coz-stroke-hglt-red` · `coz-stroke-hglt-yellow` · `coz-stroke-hglt-green`
