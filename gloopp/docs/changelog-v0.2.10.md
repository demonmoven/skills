# Gloop v0.2.10

## 修复：首页"需介入"统计口径不一致

**现象**：首页 primaryStatus 显示"7 个需要介入"，pipeline "需介入=7"，panel head "需要介入 7"，但 panel body 却显示空态"没有等待输入、阻塞或应用失败的委托"。用户被虚假的介入信号打扰。

**根因**：FocusView 的 `allActionable` 把 `failedItems`（终态 failed quest）算进"需介入"计数，但 `attentionSections`（panel body 实际渲染的分组）不含 failed。head 用 allActionable 计数、body 用 attentionSections 渲染，口径不一致。failed 是终态历史（v0.2.9 已在 list view 折叠），不该占首页 attention。

**修复**（两口一起对齐）：
- **allActionable 去掉 failedItems**：终态 failed 不再算"需介入"。failed 留在 list view 折叠状态，用户想看可展开。
- **buildAttentionSections 加"执行超时"section**：`isRunningTooLong`（running/reviewing 超过 maxTurns*2 分钟）的 quest 单独成组。这才是真正需要介入的信号——agent 可能卡死或循环。此前 stuckItems 只在 allActionable 计数里，body 不显示，现在 body 也能看到具体哪几个卡住了。

效果：head 计数和 body 渲染口径完全一致。当前无 active quest 时首页显示"风平浪静"，不再被 7 个 failed 终态污染。
