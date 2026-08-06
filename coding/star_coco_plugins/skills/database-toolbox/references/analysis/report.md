# 报告生成

## 报告风格

5 种经实战验证的视觉风格。用户未指定时随机选择。

| 风格 | 标志元素 | 最适场景 | 核心配色 |
|------|---------|---------|----------|
| Financial Times | 三文鱼粉底 + 4px蓝色顶线 | 金融分析、叙事报告 | 背景 `#FFF1E5`，强调蓝 `#0F5499` |
| McKinsey Consulting | 深蓝Header + Exhibit编号 | 战略分析、框架评估 | Header `#003366`，强调蓝 `#4472C4`，强调橙 `#ED7D31` |
| The Economist | 6px红色顶线 + editorial标题 | 行业洞察、观点报告 | 红 `#E3120B`，白底 |
| Goldman Sachs | Rating徽章 + 金色强调 | 财务建模、估值报告 | 深蓝 `#00338D`，金 `#D4AF37`，负面红 `#C62828` |
| Swiss / NZZ | 黑白灰红 + 72px大字 | 数据展示、设计感报告 | 黑 `#000000`，红 `#FF0000`，Helvetica |

**风格要点**：
- McKinsey：结论式标题（「Cloud revenue drives 60%」而非「Revenue by segment」），无渐变/阴影/圆角
- Economist：Editorial 标题带观点（「AI的胃口」而非「资本开支趋势」）
- Swiss：极端字号对比，禁止圆角/阴影/渐变/背景色块

## 模板选型

每个模板单独一个文件在 `templates/` 下，生成报告时合并为完整 HTML。

| 模板 | 文件 | 用途 |
|------|------|------|
| KPI看板 | [template-kpi.md](./templates/template-kpi.md) | 数据概览 / 关键指标 |
| 对比表格 | [template-table.md](./templates/template-table.md) | 多维度对比 |
| 柱状图 | [template-bar.md](./templates/template-bar.md) | 趋势展示 / 类别对比 |
| 诊断卡片 | [template-diagnosis.md](./templates/template-diagnosis.md) | 问题诊断 / 优化建议 |
| 折线图 | [template-line.md](./templates/template-line.md) | 时间序列趋势 |
| 饼图/环形图 | [template-pie.md](./templates/template-pie.md) | 占比分析 |
| 漏斗图 | [template-funnel.md](./templates/template-funnel.md) | 转化率分析 |
| 进度条 | [template-progress.md](./templates/template-progress.md) | 目标达成情况 |
| 排名榜单 | [template-rank.md](./templates/template-rank.md) | TOP N 排名 |
| 多Tab报告 | [template-tabs.md](./templates/template-tabs.md) | 综合报告 |
| 地图 | [template-map.md](./templates/template-map.md) | 地区分布 |
| 热力图 | [template-heatmap.md](./templates/template-heatmap.md) | 交叉分析 |
| 词云 | [template-wordcloud.md](./templates/template-wordcloud.md) | 关键词/热词 |

**内容 → 模板对应**：

| 内容类型 | 推荐模板 |
|----------|----------|
| 数据概览/周报开头 | KPI看板 |
| 多维度对比 | 对比表格 |
| 趋势展示 | 柱状图 / 折线图 |
| 占比分析 | 饼图/环形图 |
| 转化分析 | 漏斗图 |
| 目标达成 | 进度条 |
| 排名榜单 | 排名榜单 |
| 问题诊断 | 诊断卡片 |
| 多板块报告 | 多Tab切换 |

## 生成流程

### 1. 合并模板为完整 HTML

```html
<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<style>
  body { margin: 0; padding: 40px; background: #F5E6D3; font-family: "PingFang SC", Arial, sans-serif; }
  .section { margin-bottom: 40px; background: #FFFDF7; border: 4px solid #1A1A1A; border-radius: 12px; padding: 30px; box-shadow: 6px 6px 0 #1A1A1A; }
</style>
</head>
<body>
  <div class="section"><!-- KPI HTML --></div>
  <div class="section"><!-- 表格 HTML --></div>
</body>
</html>
```

### 2. 保存 HTML

HTML 较长时（超过 50 行），用 Write 工具直接写入 `wf.get_output_path("06_report", "analysis_report.html")` 返回的路径。

> ⚠️ **禁止**把大段 HTML 塞进 `python3 -c "..."` 命令行。Shell 有长度限制。

### 3. Playwright 截图

```bash
# HTML_PATH / PNG_PATH 来自 wf.get_output_path() 的输出
npx playwright screenshot "file://${HTML_PATH}" "${PNG_PATH}" \
  --viewport-size=1200,675 --full-page --wait-for-timeout=3000
```

### 4. 交付

展示 PNG 截图 + HTML 文件链接给用户。

## 配色速查

| 用途 | 色值 |
|------|------|
| 珊瑚（主色） | `#E17055` |
| 薄荷绿 | `#45B7AA` |
| 橄榄绿 | `#5B8C5A` |
| 金色（暗金） | `#D4A017` |
| 正面/增长 | `#4CAF50` |
| 负面/警告 | `#FF3B4F` |
| 背景奶油 | `#F5E6D3` |
| 卡片白 | `#FFFDF7` |

## 设计规范

**必须做**：零CDN依赖（纯SVG/内联JS）、柱状图Y轴从0开始、条形图绝对比例、辅助文字≥10pt、表格斑马纹。

**禁止做**：CDN资源（Chart.js/ECharts）、SVG坐标超出viewBox、CSS absolute定位数据点、同一报告混用不同风格。
