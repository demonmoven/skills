---
name: skill_doubao_init
description: "doubao-agent-scene-brainstorm 套件的初始化技能，AGENT 无需主动使用"
---

<!-- @format -->

# skill_doubao_init（特殊技能，无需主动使用）

参考 ecom-buy `skill_init`，本 skill 是 doubao-agent-scene-brainstorm 套件的**初始化锚点**，不在三阶段流程中主动触发。

## 可能的用途（v2 扩展）

- 验证 5 个 locator 的 `last_synced_commit` 是否仍指向当前分支
- 输出当前 8 仓库 worktree 的健康状态
- 提示 RD 可能的 knowledge / locator 漂移

## 当前实现

仅占位，套件入口由顶层 [`../../SKILL.md`](../../SKILL.md) 与 [`../../guidance.md`](../../guidance.md) 承担。
