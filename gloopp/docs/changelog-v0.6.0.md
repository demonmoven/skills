# Gloop v0.6.0 — Thread as Default Surface

## 概述

v0.6 将 ThreadView 确立为委托的默认阅读面，QuestDetail 退居管理面（overlay drawer）。
三账本分离落地：Narrative（ThreadPost 持久化）、Observation（ExecutionTrace/会话事件）、Action Authority（QuestMeta + action API）。

## 主要变更

### ThreadView 默认阅读面（P0c）

- 打开委托默认进入 ThreadView，按时间线展示所有帖子
- QuestDetail 通过 Management Drawer 叠加在 ThreadView 上，不再全页替换
- Root post 作为叙事锚点始终可见，不被过滤

### Fanout 生命周期投影（P0b-7/8/12）

- `fanout_group_opened`：首次 spawn 时在 root thread 标记分组开启
- `fanout_leaf_spawned`：每个 leaf spawn 后在 root thread 记录
- `fanout_leaf_done`：leaf 进入终态（success/failed/cancelled）时在 root thread 记录
- `fanout_summary`：语义状态变化时（spawn/terminal/merge）生成快照汇总
- `fanout_merge_decision`：leaf 携带 merge 策略时在 root thread 记录

### Reply-to-post（P1-3）

- ThreadLedgerPanel 每个非 root 帖子增加回复按钮
- 回复时在 composer 上方显示回复上下文（作者 + 取消按钮）
- 后端 `AppendUserComment` 新增 `parent_reply_id` 参数，校验目标帖子存在于 thread 中
- `commentOnQuest` API 支持可选 `parent_reply_id`

### Fanout Summary Snapshot（P1-2）

- `AppendFanoutSummarySnapshot` 在 root thread 写入 `fanout_summary` 类型帖子
- seq 基于已有 summary 数量递增，天然幂等
- 内容包含 leaf 状态统计（total/running/terminal/blocked/waiting）、merge 策略、owner leaf
- causal refs 包含所有 lifecycle 帖子

### Thread Streaming SSE（P2）

- ThreadView 集成 `useEventStream` hook，订阅 per-quest semantic SSE
- 收到 semantic 事件后自动 refetch thread，1.5s debounce 避免频繁刷新
- 实时状态指示器：连接中/已连接/重连中/已断开
- Leaf terminal 时 root quest 发布 `quest.note` 事件，确保 root ThreadView 订阅者收到刷新通知

### Root Thread SSE on Leaf Terminal

- `publishRecordedEvent` 中心路径检测 terminal 事件类型
- Fanout leaf 进入终态时自动为 root quest 发布 `quest.note`（kind=fanout_leaf_done）
- 顺序保证：SaveQuest 先写 root thread posts → 再 publish root event
- 非 fanout 委托不产生额外 root 事件

## 账本/通知对齐

| 场景 | Narrative (ThreadPost) | Observation (Event) | Action Authority |
|------|----------------------|-------------------|-----------------|
| Leaf spawn | fanout_leaf_spawned | root quest.note + leaf quest.spawned | QuestMeta.GroupID |
| Leaf terminal | fanout_leaf_done + fanout_summary | leaf quest.success/failed + root quest.note | QuestMeta.Status |
| User comment | human_comment (可选 parent_reply_id) | quest.note | commentOnQuest API |

## 兼容性与回滚

- Thread API (`/api/quests/{id}/thread`) 时间字段统一用 `_ms` 后缀
- Fanout root 身份：`GroupID != "" && FanoutLeafID == ""` = canonical root
- 回滚：v0.5 QuestDetail 页面代码保留，可通过移除 ThreadView 默认路由恢复
- 所有新增 ThreadPost 类型向后兼容，旧版本前端会忽略未知 kind

## 已知非阻塞项

- `ThreadLedgerPanel` 帖子视觉外壳后续可收敛为 `ThreadPostCard`（减少 debug 元信息）
- `AppendFanoutSummarySnapshot` seq 基于计数，后续如需更强幂等可引入状态 hash
- P2 streaming indicator 状态文案目前只有中文，英文 i18n 待补

## 测试

- Go: `go test ./...` 全量通过
- Web: 6 个测试套件全部通过（mage-review 20/20、quest-answer-api、quest-display-semantics、thread-view-display、streaming 6/6、i18n-smoke）
- Streaming behavior tests 覆盖：EventSource URL 构造、hello 帧过滤、semantic 事件分发、1.5s debounce、indicator CSS class、SSE handler + debounce 集成
- Race detector: `go test -race` 通过
