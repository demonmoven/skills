# 文生图 — 主 bot Action Bar 入口

> ⚪ **骨架占位**（命中本期 PRD 改造但未完整文档化）
>
> **业务 PM**：欧桐桐
> **入口特点**：用户点击 Action Bar 按钮触发；意图前端结构化（无需 LLM 意图识别）；典型场景：从某图开始文生图模板触发

---

## 状态

- [ ] 用户动线 mermaid 图（TODO）
- [ ] PSM 链路（与 [闲聊入口](./主bot%20闲聊入口.md) 类似，差异：意图来自 Action Bar 协议字段，**不依赖 LLM 意图**）
- [ ] 命中的 Tool（与闲聊版相同 image_gen）
- [ ] 命中的 SP（**Action Bar 模板可能用专门 SP**，需查代码）
- [ ] Libra 切流点（同闲聊 + Action Bar 模板专属切流）
- [ ] 历史演进备注
- [ ] 本期改造影响点

## 与闲聊入口的关键差异

| 差异 | 闲聊 | Action Bar |
|---|---|---|
| 意图识别 | LLM | 前端结构化 |
| SP 是否需要"识别意图"段 | 需要 | 可省略 |
| 协议入口字段 | `Query.ContentForModel` | `Query.ActionBar*` 系列字段 |

## TODO

需求触达本动线时，从 [闲聊入口](./主bot%20闲聊入口.md) 复用大部分内容，重点细化 Action Bar 协议解析部分。
