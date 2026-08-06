# 文生图 — 「AI 图片生成」官方多 bot 入口

> ⚪ **骨架占位**
>
> **业务 PM**：欧桐桐
> **入口特点**：豆包平台官方多 bot（含 AI 图片生成 / AI 漫画生成）；用户进入 bot 即默认创作意图；可能用 bot_engine 映射机制接入到生图 agent

---

## 状态

- [ ] 用户动线 mermaid
- [ ] PSM 链路（**特殊**：可能涉及 bot_engine 映射 → flow.agent.creation）
- [ ] bot 映射机制
- [ ] 命中的 Tool / SP / Libra
- [ ] 历史演进备注（基于 hook 的生图 agent 时代起源）
- [ ] 本期改造影响点

## 与闲聊入口的关键差异

| 差异 | 闲聊 | 多 bot |
|---|---|---|
| 入口 | 主 bot 任意对话 | 进入 AI 图片生成 / AI 漫画生成 bot |
| 意图先验 | LLM 识别 | bot 即类型 |
| 路由机制 | 直接 AgentStream | 经过 bot 映射 → AgentStream |
| SP / 模型可能不同 | — | 不同 bot 可能用不同 SP |

## TODO

需求触达时补充。
