# Gloop v0.2.20

## 优化：首页 HOTL 信息密度 + 交付产物预览放大

### 背景

HOTL 模式下首页常空——「需要介入」区（user_review + blocked）常态无内容，首页以"待介入"为中心，但 HOTL 常态是"无需介入"，导致信息密度低。用户反馈"大部分不需要处理后首页大多数时间比较空"。同时交付产物预览窗口太小（max-height 360px），长文档查看不便。

### 修复

**A. 影响日志从「今日」扩展到「近 7 天」**：
- HOTL 常态下今日可能无闭环（agent 自主推进中），"今日"太窄，首页经常空
- "近 7 天"更符合 HOTL「感知影响」语义——用户感知近期影响，不限当日
- 默认展示 5 条（原 3 条），展开看全部
- pipeline 保留「今日已闭环」计数（快速感知今日量），影响日志区用近期

**B. 需介入为空时折叠为紧凑单行**：
- 原空状态 `cc-empty-line` min-height 74px，占大块空间
- 新 `cc-attention-empty`：单行紧凑提示（6px padding，虚线边框，绿色淡底），视觉轻量
- 空间让给执行循环和影响日志，首页信息密度提升

**C. 交付产物预览放大**：
- `max-height` 360px → 70vh（视口高度 70%，大屏看全文舒适）
- `font-size` 12.5px → 13px，`padding` 10/12 → 12/14，长文档阅读体验提升

### 为什么这是全局最优

- **A 不改后端**：纯前端时间窗口调整，completed_at_ms 数据已有
- **B 不删信息**：空状态仍提示"暂无待介入委托，冒险者自主推进中"，用户知道状态，只是不占空间
- **C 用 vh 而非固定 px**：70vh 随视口自适应，小屏不会溢出，大屏利用空间

### 变更

- `web/src/pages/FocusView.tsx`：`todayImpact` → `recentImpact`（7 天窗口），默认展示 5 条；空状态用 `cc-attention-empty`
- `web/src/styles.css`：新增 `.cc-attention-empty` 紧凑样式；`.quest-output-preview .markdown-renderer` max-height 70vh、font-size 13px、padding 加大
