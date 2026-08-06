# Analyze Design Command

从 Figma 设计稿获取原始数据（截图 + 高保真 JSX），为后续 UI 结构分析提供输入材料。

**重要**: 在开始工作前，必须先读取 `.claude/docs/constitution.md` 了解本工程的技术规范。

## 调用方式

```
/ui-workflow ui-analyze <page_name> <figma_url> [description]
```

### 参数

- `page_name`: 设计稿名称（英文，kebab-case 格式）
- `figma_url`: 设计稿的 Figma 链接（可以是完整页面或包含 node-id 的局部区域）
- `description`: (可选) 需求规格里对设计稿理解有帮助的相关说明

### 示例

```bash
# 分析完整页面
/ui-workflow ui-analyze agent-editor-page https://www.figma.com/design/xxx?node-id=123:456

# 分析页面中的局部区域
/ui-workflow ui-analyze model-config-panel https://www.figma.com/design/xxx?node-id=123:789 "模型配置面板"
```

## 输出产物

所有产物输出到 `.design_analysis/pages/<page_name>/` 目录:

| 文件 | 说明 |
|------|------|
| `design.png` | 设计稿截图 |
| `high-fidelity.jsx` | 高保真 JSX 代码 |

> UI 结构分析产物（`ui.xml`、`ui.implement.xml`、`ui.md`、`nodes/`）由 `ui-structure-analyzer` 命令生成。

---

# 执行流程

**重要**: 本 Command 集中处理所有 MCP 调用，subagent 不直接调用 MCP 工具。

## Step 1: 参数解析与环境准备

### 解析输入参数

- `$0` → `page_name`
- `$1` → `figma_url`
- `$2` → `description` (可选)

### 创建输出目录

```bash
mkdir -p .design_analysis/pages/<page_name>
```

## Step 2: 调用 `figma-screenshot` skill 获取设计稿截图

调用 `figma-screenshot` skill:

```bash
npx -p @byted/x-figma-to-code figma-to-code "<figma_url>" \
  --screenshot \
  --format png \
  --scale 1 \
  --output .design_analysis/pages/<page_name>/design.png
```

记录返回的 `outputPath`（截图本地路径）。

## Step 3: 调用 `figma-to-code` skill 获取 高保真 JSX


```bash
npx -p @byted/x-figma-to-code figma-to-code "<figma_url>" \
  --framework Tailwind \
  --html-mode jsx \
  --no-embed-vectors \
  --no-embed-images \
  --output .design_analysis/pages/<page_name>/high-fidelity.jsx
```

记录返回的 `outputPath`（JSX 文件路径）。

---

# 检查项

- [ ] 设计稿截图已获取
- [ ] 高保真 JSX 文件已生成

---

# 返回结果

完成后返回产物清单（含本地路径和 TOS URL）:

| 产物 | 路径 |
|------|------|
| 设计截图 | `.design_analysis/pages/<page_name>/design.png` |
| 高保真 JSX | `.design_analysis/pages/<page_name>/high-fidelity.jsx` |
