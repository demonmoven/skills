# gloop 前端 HOTL 适配 Spec v0.2.1

> 版本：v0.2.1
> 状态：已落地
> 前置：v0.2.0 后端 HOTL 五环已落地（checker 权威化 + 自主闭环），但前端 UI 仍停留在 HITL 语义
> 参考框架：[Addy Osmani — Loop Engineering](https://addyosmani.com/blog/loop-engineering/)

---

## 一、第一性原理

v0.2.0 把后端从 HITL 演进到 HOTL：checker pass = 最优解，quest 自主 apply + 闭环，人不再 review agent 产出。但前端 UI 没跟上——文案、分区、优先级仍是 HITL 语义，把人定位成"审核者/裁决者"。

**HOTL 下人的真实角色**：感知影响 + 处理异常（blocked / waiting_input / apply 失败）。不是审核者。

前端必须对齐这个角色定位，否则 v0.2.0 的体感没落地——人打开 Dashboard 看到的还是"等你裁决""用户审核"，和 HITL 没区别。

## 二、缺口（HITL 残留）

### 2.1 文案类：把人定位成"审核者"

| 位置 | HITL 文案 | 问题 |
|------|----------|------|
| FocusView pipeline | "等你裁决" | v0.2.0 人不裁决，只处理异常 |
| FocusView primaryHint | "先处理阻塞、失败和用户审核" | "用户审核"是 HITL 概念 |
| FocusView 空态 | "没有待审核、等待输入..." | "待审核"是 HITL 概念 |
| FocusView 空态 | "没有阻塞，没有待审核项" | 同上 |
| questsBoardHelpers | focusPrimaryLabel user_review → "去审核" | 人不审核 |
| questsBoardHelpers | attention section "等待审核 / 需要你做最终审核 / apply" | 人不做最终审核 |
| QuestCard | reviewLabel 默认 "等待你审核" | 同上 |
| QuestCard | action 文案 "审核" | 同上 |
| CurrentWorkStatus | "等待你的评审裁决" | 同上 |

### 2.2 分区类：缺影响日志

spec v0.2 环4c 明确要求："自主闭环的 quest 进影响日志分区，默认折叠，显示影响摘要"。当前 FocusView 只有"最近结果"露 4 条（含 failed/cancelled），人无法感知全局影响。

**影响日志是 HOTL"人感知影响"的核心载体**，漏了等于 v0.2.0 的体感没落地。

### 2.3 不改的（保留）

| 项 | 为什么保留 |
|----|-----------|
| STATUS_LABEL `user_review: '用户审核'` | 状态机技术术语，统一词汇，改一处要改全套且影响 API/trace 一致性 |
| `FinalVerdictCard` "用户终审"（finalized_by=user） | 人工介入时人确实做了终审动作，标签准确 |
| `finalizedByLabel` "策略结案/人工结案" | 已对齐，policy=自主闭环、user=人工 |
| `CreateAutomationSheet` "跳过人工终审并直接 apply" | automation 的 auto_apply 配置语义没变，描述准确 |
| `contextLabels` "冲突裁决/裁决规则" | context 配置项，与 quest 审核无关 |
| `traceModel` verdict → "裁决" | 事件类型标签，内部术语 |
| `QuestDetail` "转人工审核"操作 | 这是 blocked → user_review 的显式人工入口，操作语义准确 |

## 三、方案

### 3.1 文案对齐（HITL → HOTL）

核心原则：把"审核/裁决"替换为"处理/查看"。人处理异常，查看结果。

| 位置 | 旧 | 新 |
|------|----|-----|
| FocusView pipeline | 等你裁决 | 需介入 |
| FocusView primaryHint | 先处理阻塞、失败和用户审核 | 先处理阻塞、失败和应用异常 |
| FocusView 空态 | 没有待审核、等待输入、阻塞或应用失败的委托 | 没有等待输入、阻塞或应用失败的委托 |
| FocusView 空态 | 没有阻塞，没有待审核项 | 没有阻塞，没有待处理项 |
| focusPrimaryLabel | user_review → "去审核" | "去处理" |
| focusPrimaryLabel | pass → "确认结果" | "查看结果" |
| attention section | "等待审核 / 需要你做最终审核 / apply" | "需人工处理 / 返工耗尽或人工转入，需你决定方向" |
| QuestCard reviewLabel | "等待你审核" | "需人工处理" |
| QuestCard action | "审核" | "处理" |
| CurrentWorkStatus | "等待你的评审裁决" | "等待你处理" |

### 3.2 影响日志分区

FocusView 的"最近结果"升级为"影响日志"：

- **数据源**：今天自主闭环的 quest（success 且非 apply_failed），按完成时间倒序
- **每条显示**：闭环徽章 + query 摘要 + 文件变更数 + effect label（非 muted）+ 相对时间
- **默认折叠**：前 3 条，超过显示"展开全部 N 条"
- **空态**："今天暂无自主闭环记录"

这样人打开 Dashboard 就能一眼看到"今天 agent 自主闭环了哪些事、改了多少文件、什么影响"——HOTL 感知影响的载体。

### 3.3 优先级（不改）

`focusPriority` 把 user_review 排优先级 0（和 blocked/apply_failed 同级）。v0.2.0 下 user_review 只在 apply 失败/返工耗尽/人工转入时出现，保持高优先级合理——这些确实需要人介入。不改。

## 四、落地清单

| 文件 | 改动 |
|------|------|
| `web/src/pages/FocusView.tsx` | pipeline/hint/空态文案；最近结果 → 影响日志分区（useState + todayImpact + 展开折叠） |
| `web/src/pages/questsBoardHelpers.ts` | focusPrimaryLabel 文案；buildAttentionSections user_review section 文案 |
| `web/src/pages/QuestCard.tsx` | reviewLabel 默认值；action 文案 |
| `web/src/pages/CurrentWorkStatus.tsx` | "等待你的评审裁决" → "等待你处理" |
| `web/src/styles.css` | cc-result-badge / cc-result-effect / cc-result-time 样式 |

## 五、验证

- `tsc` 类型检查通过
- `vite build` 生产构建通过
- `go test ./...` 不受影响（纯前端改动）
- 版本：0.2.0 → 0.2.1
