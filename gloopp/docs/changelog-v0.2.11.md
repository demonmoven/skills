# Gloop v0.2.11

## 修复：iOS 移动端点开输入框自动放大

**现象**：iOS Safari 聚焦输入框时页面自动放大。

**根因**：iOS Safari 对 `font-size < 16px` 的输入框聚焦时会自动 zoom 到 16px 等效，以便用户看清正在编辑的文字。Gloop 大量输入框用了 11-15px 小字体，触发此特性。

**修复**：viewport meta 加 `maximum-scale=1.0, user-scalable=no`。Safari 对表单聚焦缩放（input focus auto-zoom）与双指缩放是两套机制——`maximum-scale=1` 对前者仍然有效（iOS 10+ 的无障碍策略只解除双指缩放限制，不影响表单聚焦缩放抑制）。一行改动，零视觉回归。
