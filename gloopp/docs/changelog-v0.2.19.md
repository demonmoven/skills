# Gloop v0.2.19

## 修复：「冒险者结论」优先展示产物 md 而非 turn summary

### 现象

v0.2.18 修复了产物声明链路（outputs 不再为空），但「冒险者结论」区块仍显示 warrior 的 turn summary 文本（`docs/xxx.md` 被当 inline code），产物 md 全文没展示。

### 根因

`selectQuestProducerArtifact` 优先级：`warrior_message` > `warrior_context_artifact` > `warrior_tool_summary` > **`summary_artifact`**。产物 md 排最后。design quest 普遍有 `warrior_context_artifact`（mage review 注入的 warrior turn summary），抢在产物 md 前面，所以「冒险者结论」一直显示 turn summary 文本。

`pickSummaryArtifact` 也偏窄：只认 `kind=summary` 或 name 含 `summary/review/conclusion`。spec/design 等产物名不匹配，导致 `summary_artifact` 为 null，直接回退到 warrior turn summary。

### 修复

**`selectQuestProducerArtifact`**：`summary_artifact` 提到最高优先级。产物 md 是 warrior 的实际交付物，优于 turn summary（口头描述）。turn summary 常含 `docs/xxx.md` 引用，渲染成 inline code 体验差；产物 md 全文才是「冒险者结论」该展示的。

**`pickSummaryArtifact`**：扩展回退逻辑——`kind=summary` > name 含 `summary/review/conclusion` > 第一个文本产物（`isTextualArtifact` 命中 `.md/.txt/.log/.json`）。design quest 的 spec/design/review md 都能被选中。

### 为什么这是全局最优

- **语义对齐**：「冒险者结论」= 冒险者的交付结论，产物 md 是结论本身，turn summary 只是结论的描述
- **回退完备**：有产物 md 展示产物；无产物回退 warrior 发言；再无回退 design_summary。各级都有兜底
- **不改后端**：纯前端选取逻辑调整，outputs 数据结构不变

### 变更

- `web/src/pages/questDetailHelpers.ts`：
  - `selectQuestProducerArtifact`：`summary_artifact` 分支提到函数最前
  - `pickSummaryArtifact`：增加 `first_textual` 回退（取第一个文本产物）
  - `SummaryArtifactMatch.reason` 类型加 `'first_textual'`

### 关联

v0.2.18 打通了产物声明链路（outputs 有值），本版让前端真正优先展示产物 md。两版配合彻底解决「产物 md 看不到 / 冒险者结论 inlinecode」。
