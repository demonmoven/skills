# gloop Impact Summary Spec v0.1 — agent 主动声明影响

> 版本：v0.1.0
> 状态：已落地
> 前置：HOTL v0.2（自主闭环 + 影响通知）

## 第一性原理

HOTL 的人感知单位是"工作区自驱做了什么、什么影响、要不要我介入"。三层语义：变更事实（diff，已有）/ 变更影响（波及什么，本次补齐）/ 决策信号（attention，已有）。中间层原本是空的——旧"影响通知"是伪影响，用 final_comment + files_changed 冒充。

## 方案

- 数据：QuestMeta.ImpactSummary（WhatChanged/Affected/NotTouched/Caveats），与 FinalComment/DiffStat 正交，omitempty 向后兼容。
- 采集：gloop phase done --impact → CLI → server → engine.PhaseCheckpoint → PhaseSignal.Data → macro_loop 提取 → 持久化。Native tool 路径同步。解析失败回落旧逻辑。
- 消费：通知侧 sendQuestImpact 优先用 ImpactSummary；视图侧 FocusView 补 blocked/failed + impact 文本；详情侧 ImpactSummaryCard。
- prompt：warrior_execution_instruction.md 要求 agent 用 --impact 声明。

## 不做
不做 git log 流水账、不做独立 LLM 调用、不拦 user_review。
