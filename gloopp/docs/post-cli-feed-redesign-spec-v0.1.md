# Post CLI + Feed 单源重设计 spec v0.1

> 日期：2026-06-26
> 状态：设计中

## 背景

gloop v0.3 的 Feed 当前是双源模型：

1. `micro.assistant_msg` — agent 会话里的自然语言输出，靠 `isSystemNoise` 从噪声里硬捞
2. `micro.tool_end(phase_checkpoint)` — 阶段闭环的 summary

这导致三个问题：
- Feed 乱：agent 的自然语言输出混了对话/思考/系统噪声，需要一堆去重规则（碎碎念去重、前缀相似去重、病态循环抑制）在噪声里捞发言
- gloop 干预 agent 内部循环：platform tools 让 gloop 在 micro_loop 里拦截 tool call → 代执行 → 回灌结果，违反 HOTL「gloop 拥有循环状态机，不干预 agent ReAct」的原则
- 交互通道冗余：phase checkpoint / review / note / post 都有 platform tool 和 CLI 两条路径做同一件事

## 第一性原理

HOTL（Human-on-the-Loop）下 gloop 的职责边界：
- **拥有**：循环状态机（quest/phase/budget/rework/闭环）
- **不拥有**：agent 内部 ReAct 循环

因此 gloop-agent 交互应该是**单通道**：agent 通过 `gloop` CLI 向平台发信号，gloop 不感知 agent 的 tool call。agent 的 ReAct 完全黑盒，gloop 只看 phase 边界。

## 设计

### 1. 交互通道：CLI only

砍掉 platform tools 执行路径 + native tool call 感知。

**保留的 agent → gloop 通道（全部走 CLI）：**
- `gloop phase done` — 剑士声明阶段完成
- `gloop phase fail` — 剑士声明阶段阻塞
- `gloop review pass|request-changes|reject` — 法师提交评审
- `gloop post` — agent 发帖（新增，合并 note_add + post_update）
- `gloop quest info` — 查委托信息
- `gloop quest ask` — 剑士向用户提问
- `gloop context show|list` — 查全局上下文
- `gloop skill show` — 加载 skill 正文
- `gloop command run` — 白名单命令（法师验证用）

**删除的：**
- `internal/platformtools` 的 14 个工具注册 + handler（phase_checkpoint/review_quest/note_add/post_update/quest_info/context_list/context_show/command_run/code_search/notify_user/quest_spawn 等）
- micro_loop 里的 tool call 处理分支（`isNativeAgentToolCall` 路径 + platform tool 代执行路径 + tool result 回灌）
- ACP executor 的 native tool call 上报（`ToolOriginACPNative` 感知）
- 法师评审证据里的 `warrior_native_tool_calls` block（剑士自己在 phase done 的 summary/impact 里声明做了什么）

**不删的：**
- `gloop command run`（法师白名单验证命令是安全边界，不是 agent 能力重复）
- evidence_injection（评审前自动跑 L0 验证命令，纯平台机制，不涉及 agent 交互）

### 2. Post 工具契约

合并 `note_add` + `post_update` 为单一 `gloop post` CLI。

**`gloop post` 参数：**
```
gloop post --content "正文内容" [--reply-to <post_id>] [--kind post|milestone|blocker]
```

- `--content`（必填）：正文，Markdown。X 式发帖，没标题。
- `--reply-to`（可选）：回复的 post_id。显式声明回复谁，替代前端 replyTarget 推断。
- `--kind`（可选，默认 `post`）：`post` / `milestone` / `blocker`。用于前端样式区分，不影响数据模型。

**事件模型：**
- 事件类型：`agent.post`（新事件类型，替代 `agent.update`）
- payload：`{content, reply_to, kind, post_id}`
- `post_id` 由平台生成（`post_<timestamp>_<random>`），用于 reply_to 引用

**reply_to 语义：**
- 空值：顶层发言（回复委托发起者「你」）
- post_id：回复指定 post
- quest_id：回复整个委托（用于扇出委托回复父委托）

### 3. Feed 单源模型

Feed 只收两种事件：
- `quest.created` → kind=user（委托帖，用户发起委托本身就是帖）
- `agent.post` → kind=post（agent 主动发帖）

**删除的 Feed 数据源：**
- `micro.assistant_msg`（agent 自然语言输出不进 feed）
- `micro.tool_end(phase_checkpoint)`（phase summary 不进 feed）

**删除的去重规则：**
- `isSystemNoise` 过滤
- 碎碎念去重（<30 字只留最新）
- 前缀相似去重（前 40 字相同只留最新）
- 病态循环抑制（连续 3 条砍最早）
- 自动化去重（同自动化只留最新）

post 是 agent 显式声明「这是我要发的帖」，不需要去重——agent 要发就发，不发就没有。

**phase checkpoint 的去向：**
- summary / verdict / impact / deliverables 全部沉到 quest 详情页
- Feed 里不再有 kind=phase 的帖子
- 用户想看阶段闭环数据，去详情页

**agent 不发 post 怎么办：**
- 不兜底。HOTL 的诚实性：agent 不主动汇报就是没汇报。
- 极端情况 quest 详情页有完整 phase checkpoint 数据 + Execution Trace，用户随时能看。
- 这是行为约束：不调 post = feed 里不存在，倒逼 agent 学会用 post。

### 4. Prompt + Skill

**System prompt 变更：**
- 删除 platform tool manifest 注入（`prompt.go` 里的 tool 列表）
- 新增 post 协议：「你要对用户说话时，用 `gloop post --content "..."` 发帖。不要直接输出自然语言指望平台捞——feed 里只有你主动 post 的内容。」
- phase checkpoint 协议不变（已走 CLI）

**新增 skill `gloop-posting`：**
- 正文：2-3 个 post 示例（汇报进展、回复评审、声明阻塞）
- 渐进式披露：system prompt 注入 name+description，agent 按需 `gloop skill show gloop-posting`

## 迁移清单

### 删除
- `internal/platformtools/registry.go` — 工具注册 + handler（保留 command_run 相关）
- `internal/orchestrator/micro_loop.go` — tool call 处理分支（`isNativeAgentToolCall` + platform tool 代执行 + tool result 回灌）
- `internal/executor/acp.go` — native tool call 上报（`ToolOriginACPNative` 感知路径）
- `internal/server/api_activity.go` — `isSystemNoise` + assistant_msg 捞取 + phase summary + 全部去重规则
- `internal/prompt/prompt.go` — `warrior_native_tool_calls` block + platform tool manifest 注入
- `web/src/components/FeedCard.tsx` — kind=phase 相关渲染（保留 post/user）

### 新增
- `internal/cli/cmd_agent_syscall.go` — `runPostCmd`（gloop post 子命令）
- `internal/cli/server_proxy.go` — `postAgentPost` API 调用
- `internal/server/api_post.go` — POST /api/quests/{id}/post 端点
- `internal/orchestrator/engine.go` — `PublishPost` 方法（发 `agent.post` 事件）
- `internal/events/` — `EvtAgentPost` 事件常量
- `web/src/api/types.ts` — ActivityKind 简化为 `user | post`
- `web/src/components/FeedCard.tsx` — reply_to 渲染

### 修改
- `internal/prompt/prompt.go` — 删 tool manifest，加 post 协议
- `internal/prompt/templates.go` / skill 目录 — 新增 gloop-posting skill
- `internal/server/api_activity.go` — buildActivityItem 只认 quest.created + agent.post
- `web/src/pages/FeedView.tsx` — 适配新数据源

## 不做的事

- 不碰 ACP client（JSON-RPC over stdio 本来就复杂，砍的是 tool call 感知不是协议本身）
- 不碰 evidence_injection（评审带证据是核心）
- 不碰 command_run（安全边界）
- 不做 phase checkpoint 自动 post（会导致 post + phase summary 重复，比之前更吵）
