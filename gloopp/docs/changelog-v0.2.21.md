# Gloop v0.2.21

## 修复：legacy review 盖掉真实评审 + 首页需介入空时下方空白

### Bug 1：4386 法师评审显示"未评分 通过 Legacy 溯源不完整"

**现象**：qst_2606254386 法师评审区显示"未评分 ● 通过 Legacy 溯源信息不完整，建议升级到最新 agent"。

**根因**：4386 有两条 review 记录：
- `reviewed_by=adv_mage_001, verdict=pass, score=8`（exact_adventurer，真实法师评审）
- `reviewed_by=mage, verdict=pass, score=None, source_incomplete=True`（legacy 旧写法）

`selectLatestMageReview` 按 `ts` 取最大值，相同 ts 才走 `exact > legacy` 优先级。legacy 那条 ts 更大，直接盖掉真实评审，导致显示"未评分 + Legacy + 溯源不完整"，8 分评审被隐藏。

**修复**：有 `exact_adventurer` 候选时，过滤掉所有 `legacy` 候选。legacy 是旧写法噪音，有真实评审时不该参与竞争。

### Bug 2：首页需介入空时下方空白

**现象**：需介入区折叠成紧凑单行后，下方仍空一大片（v0.2.20 的 B 优化没完全解决）。

**根因**：`cc-panel-primary` 在 grid 里被默认 `align-self: stretch` 撑到与右侧 `cc-panel`（执行循环+影响日志）等高。内容紧凑了但容器还撑着，下方空白。

**修复**：`cc-panel-primary` 加 `align-self: start`，高度自适应内容，不撑满 grid 行。

### 变更

- `web/src/domain/mageReview.ts`：`selectLatestMageReview` 有 exact_adventurer 时过滤 legacy 候选
- `web/src/styles.css`：`.cc-panel-primary` 加 `align-self: start`
