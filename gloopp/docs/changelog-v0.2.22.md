# Gloop v0.2.22

## 修复：首页需介入为空时左列空白

### 现象

v0.2.21 的 `align-self: start` 只让左列不撑满，但 grid 空间仍空着——需介入提示下方一大片空白。

### 根因

grid 两列布局（左：需介入，右：执行循环+影响日志），需介入为空时左列只有紧凑提示，下方 grid 区域空闲。`align-self: start` 缩了 panel 高度，但 grid 单元格仍占位，空白没消除。

### 修复

需介入为空时（`attentionSections.length === 0`）**隐藏整个左列**，grid 变单列（`grid-template-columns: 1fr`），右列执行循环+影响日志全宽展示。首页不再有空白区域。

### 变更

- `web/src/pages/FocusView.tsx`：`attentionSections.length === 0` 时不渲染左列 cc-panel-primary，grid 加 `cc-grid-single` class
- `web/src/styles.css`：新增 `.cc-grid.cc-grid-single { grid-template-columns: 1fr; }`
