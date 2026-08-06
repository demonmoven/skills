# gloop 前端 HOTL 适配 Spec v0.2.2

> 版本：v0.2.2
> 状态：已落地（go test 全绿 + tsc + vite build 通过）
> 前置：v0.2.1 补了 FocusView 文案 + 影响日志分区，但信息架构/创建页/详情页/Inbox/Stats 仍有 HITL 残留
> 参考框架：[Addy Osmani — Loop Engineering](https://addyosmani.com/blog/loop-engineering/)

---

## 一、第一性原理

HOTL 下人的角色：**感知影响 + 处理异常**，不是审核者。v0.2.1 只改了文案表层，这次改信息架构和交互语义——让前端从"为人设计的审核台"变成"为人设计的影响感知+异常处理台"。

## 二、缺口与方案

### 2.1 FocusView pipeline 信息架构错位（HITL 视角）

**缺口**：pipeline 是「待召唤→剑士执行→法师评审→需介入→已归档」5 段。把"法师评审"单独拎给人看，暗示人得盯评审；"已归档"是 HITL 概念，自主闭环不是归档。

**方案**：改成 HOTL 人视角 3 段：
- **正在推进**（running + reviewing 合并，人不需要区分）
- **需介入**（blocked + waiting_input + user_review + apply_failed + failed）
- **今日已闭环**（today success，非"归档"）

### 2.2 影响日志缺汇总头

**缺口**：每条只有 N 文件 + effect + 时间，人要一条条数，无法一眼感知全局影响。

**方案**：影响日志上方加今日汇总头：`今日自主闭环 X 个 · 改动 Y 个文件` + effect 分布（如 `workspace_diff 5 · external 2`）。

### 2.3 QuestDetail user_review 操作栏伪装审批

**缺口**：canReview 时底部是「返工/拒绝/通过并应用/通过但不应用」——审核者视角。v0.2.0 下 user_review 只在 apply 失败/返工耗尽出现，语义是异常处理不是审批。

**方案**：操作栏改成异常处理语义。保留底层 verdict API（后端不变），但前端按钮重新组织：
- **放弃变更**（reject，丢弃工作区）
- **继续返工**（request_changes，追加预算重试）
- **通过并应用**（pass+apply，人确认接受这次产出——apply 失败重试或人工放行时用）
- 把"通过但不应用"降级到次级菜单（冷门操作）

### 2.4 Inbox 强制确认与自主闭环冲突

**缺口**：Inbox 是所有 automation quest 的强制关卡，空态写"等待你确认后再进入执行流程"。v0.2.0 环2 event automation spawn 的 quest 应自主执行。

**方案**：Inbox 文案对齐 HOTL（"需你确认的委托"而非"等待确认后执行"）。auto_accept 配置已有（AutoStart=true 直接跑不进 inbox），前端只需文案对齐 + 空态说明"自动接受的委托不经过此处"。

### 2.5 创建页 HITL 残留 + 缺 connector 选项

**缺口**：
1. subtitle "最终由你终审"——HITL 承诺
2. "跳过人工终审"措辞——暗示自主闭环是例外
3. 强度 hint 没讲自主闭环
4. "施工"措辞偏 HITL
5. **缺 connector 选项**（功能性缺失）——quest 自主闭环后影响要落到外部系统，创建时无法选

**方案**：
1. subtitle → "剑士执行，法师评审通过后自主应用；你只在异常时介入"
2. "跳过人工终审" → "自主闭环"，hint 正面陈述
3. 强度 hint 补"通过后自主应用"
4. "施工" → "自主施工"
5. **加 connector 可选区**：列出已启用的 connector（如 Git），创建时可选"闭环后 push 到 gloop 分支"。需后端支持：
   - `QuestMeta.Connectors []string`（quest 自身 connector 配置）
   - `CreateQuestOptions.Connectors`
   - `createQuestReq.Connectors`
   - `questConnectorAdapter` 补读 quest 自身 Connectors（automation 继承 + quest 自身合并）

### 2.6 list view 文案漏改

**缺口**：QuestsBoard list view 第 207 行"等待你审核"琥珀色 hint，v0.2.1 漏改。

**方案**：→ "需人工处理"。

### 2.7 Stats 缺自主闭环率

**缺口**：核心指标是 successRate，但 HOTL 下"自主闭环率"（多少 quest 全自动走完 checker pass → apply，没进 user_review）更有意义。successRate 100% 全人审 vs 全自主闭环，体感天差地别。

**方案**：Stats 加"自主闭环率"指标卡。数据源：success 且 finalized_by=policy（或 auto_completed_by_policy/auto_passed_by_policy）占比。前端用现有字段计算，后端无需改。

### 2.8 phase 命名 fallback "用户审核"

**缺口**：questSelectors PHASE_NAME_FALLBACK 和 phaseNameFallback 硬编码"用户审核"。

**方案**：→ "人工处理"（技术状态名 user_review 不动）。

## 三、落地清单

| 文件 | 改动 |
|------|------|
| `web/src/pages/FocusView.tsx` | pipeline 5→3 段；影响日志加汇总头 |
| `web/src/pages/QuestDetail.tsx` | user_review 操作栏重组为异常处理语义 |
| `web/src/pages/Inbox.tsx` | 空态文案对齐 HOTL |
| `web/src/components/CreateQuestSheet.tsx` | subtitle/checkbox/hint 文案 + connector 可选区 |
| `web/src/pages/QuestsBoard.tsx` | list view "等待你审核" → "需人工处理" |
| `web/src/pages/Stats.tsx` | 加自主闭环率指标卡 |
| `web/src/domain/questSelectors.ts` | PHASE_NAME_FALLBACK "用户审核" → "人工处理" |
| `internal/model/model.go` | QuestMeta 加 Connectors 字段 |
| `internal/fsstore/quests.go` | QuestMeta Connectors 持久化 |
| `internal/orchestrator/quest_lifecycle.go` | CreateQuestOptions 加 Connectors |
| `internal/orchestrator/engine.go` | questConnectorAdapter 补读 quest 自身 Connectors |
| `internal/server/api_quests.go` | createQuestReq 加 Connectors + 透传 |

## 四、验证

- `go test ./...` 全绿
- `go build` + `tsc` + `vite build` 通过
- 版本：0.2.1 → 0.2.2
