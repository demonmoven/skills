# 文生音乐 — 主 bot 指令集入口

> ⚪ **骨架占位**（**部分超出 scope**）
>
> **模型来源**：语音团队 / TT music
> **入口特点**：仅指令集触发；模型链路与生图 / 生视频不同（不走智创工程）

---

## 状态

- [ ] 用户动线
- [ ] PSM 链路（**注意**：不走 `flow.aigc.dag` / `flow.aigc.management` / `flow.aigc.tool`）
- [ ] 命中的 Tool（可能是 `text_to_music` 或独立 Tool）

## scope 提示

文生音乐的模型链路**与本 skill 重点 scope 解耦较深**：

- 生图 / 生视频走智创工程（aigc 三件套）
- 生音乐**接入 TT music**（独立链路）

如果你的需求是"在豆包闲聊中调音乐"，仍走 agent；但工具层链路完全不同。

## TODO

需求触达时确认：
1. text_to_music Tool 的 schema 在哪？
2. 调用入口是 aigc_tool 还是直接打 TT music PSM？
3. 异步上屏机制（音乐通常异步）

## 与本 skill 的关系

- 阶段一 `skill_request_brainstorm` 仍可命中本动线
- 阶段二 `skill_link_explore` 在 `tool_chain_trace` 子动作可能要"跳出"aigc 三件套去查 TT music
