# Gloop v2 PRD

> 状态：Draft / 待评审
> 版本：v2.0
> 核心理念：系统做机制，agent 做决策。两职业模型：剑士（生产者）+ 法师（评审者）。

## 1. 产品定位

Gloop 是一个运行在用户本地的 **Agent Loop 编排平台**。用户以「发布委托」的方式，驱动本机的 relay/traex/codex 等 Agent 按流程协作完成任务。

**核心原则：**
- 项目在用户文件系统里，平台不托管代码、不维护项目状态
- **系统做机制，agent 做决策** — 平台不做语义质量判断，不装 AI，不做关键词理解；但系统做**确定性环境判断**（如 git 状态、文件存在性）、**协议完整性判断**（如是否调用了 phase done / review）、**安全策略判断**（如权限边界、预算上限）
- 生产者和评审者分离 — 做的人不评，评的人不做
- 用户输入极简 — 一段文本描述任务，完事
- 经验等级给人看的 — 不影响 agent 能力，只影响自动派单优先级
- **预算 = 停止条件** — 不是为了省钱，是防止 loop 无限跑；预算维度包括 turn 数、返工次数、耗时、token、工具调用数

### 1.1 当前落地状态与成功指标

v2 蓝图分阶段落地，但目标不是停在 MVP。当前优先策略是：先让本地 execute 闭环真实可跑，再逐步补齐完整 Gloop 的 design / automation / learning / 多 agent 能力。

**当前已对齐的可运行范围：**
- execute 型 quest：剑士做 + 法师审 + 用户终审
- design 型 quest + 基于方案发起 execute
- git worktree / copy / readonly 工作区模式，其中 worktree 是主路径
- ACP agent + Relay / Pi / Codex CLI agent 适配 + Mock fallback
- Agent CLI syscall：`gloop phase/review/note/quest info/history/progress/ask`，以及平台工具 `phase_checkpoint` / `review_quest` / `quest_info` / `note_add`（均通过 CLI 调用）
- diff 查看 + apply / discard 基础能力；apply 走 `git apply --check` 预检冲突（拒绝时返回冲突文件列表）、备份快照，apply 事件落 events.jsonl 含 patch 引用与改动文件数
- blocked 恢复：继续执行 / 转用户终审 / 取消，支持 quest 级追加 turn / duration 预算
- 停止条件：单 session turn 上限、返工上限、阶段/quest 时长上限、连续 agent error 阻塞、连续无进展（无工具调用且无新推理产出）阻塞
- 委托强度：`quick` / `standard` / `deep` / `adversarial` 四档内部预算；Gloop 不做语义判断，只执行用户或 automation 显式选择
- Automation + Inbox 后端、CLI、调度、官方委托标记和基础内置页面
- 用户上下文（context）：跨 quest 持久化的全局背景知识，维度化存储 + 渐进式披露，支持 workspace / lark 等多源摘要
- 上下文自动化 `auto_context_refresh`：每日定时刷新用户上下文摘要
- Knowledge 视图与 `gloop context export --format okf --out <dir>`：将上下文导出为 OKF-style Markdown knowledge bundle，默认无外部副作用
- 内置 `gloop-user-context` skill + 平台工具 ABI：agent 可按需查询上下文维度
- 经验等级、职业称号、按胜率/等级自动派单
- Mage 默认只读工具下发与 Codex read-only sandbox
- Mage 专用 command allowlist 执行器：法师可通过平台白名单运行可审计验证命令；默认仅本地测试类命令，`bytedcli` 等真实 E2E 工具需显式配置
- 项目验证命令推荐：Settings 可扫描 `go.mod`、`package.json`、`pyproject.toml`、`Cargo.toml` 等项目文件，推荐测试、构建、lint、typecheck 命令；推荐项不自动启用，需用户确认写入 `mage_command_allowlist`
- ACP agent 真实 token 统计（usage_update）、结构化工具调用事件（tool_call/tool_call_update）、取消/中断（session/cancel）；超时自动中断 agent session
- `.gloopignore` 排除、workspace retention cleanup CLI/API/scheduler
- 事件流留痕 + Go server 内置 Dashboard

**仍需补齐的完整 Gloop 能力：**
- pause/resume 的前端操作体验与更细的失败恢复策略
- 更完整的移动端评审体验细节打磨（React/Vite Dashboard 工程化已落地：看板/列表双视图、移动端底部 Tab + FAB 发起委托、sticky 评审操作条、史莱姆吉祥物）

**MVP 成功指标：**
| 指标 | 目标 | 说明 |
|------|------|------|
| 任务完成率 | ≥ 70% | quest 最终 success / started |
| Apply 接纳率 | ≥ 50% | 用户选择 apply 的比例（不是 pass 比例） |
| 返工有效率 | ≥ 40% | 被打回后下一轮最终通过的比例 |
| 工作区污染率 | 0% | apply 前主工作区被意外修改的次数 |
| 平均人工介入次数 | ≤ 2 次/quest | 用户需要补充信息或决策的次数 |

### 1.2 非目标

**暂缓（完整 Gloop 路线后续阶段）：**
- 多用户 / 多公会 / 权限系统 — 本机单用户
- 多方案并行 / Best-of-N — v2 是单路径的
- 子 agent 递归 — v2 是固定的剑士+法师模式

**原则上不做（产品边界）：**
- 代码托管 / git server — 项目在用户文件系统里
- 语义判断 / 关键词分类 / 自动理解任务 — 系统不装 AI
- 六维属性 / 天赋 / 稀有度 / 抽卡 — 纯 RPG 元素没用
- MCP / Plugins — agent 自己的工具自己管，平台不插手
- Agent 内部状态管理 — agent 自己管 scratch 目录和记忆

## 2. 核心概念

### 2.1 职业

| 职业 | 角色 | 核心职责 | 工具权限 | 类比 |
|------|------|---------|---------|------|
| **剑士 Warrior** | 生产者（Maker） | 理解任务、出方案、写代码、跑命令、交付结果 | 全量（读/写/执行/搜索） | 开发工程师 |
| **法师 Mage** | 评审者（Checker） | 定义验收标准、审查结果、给出通过/打回结论 | 文件只读 + 平台 CLI + 只读 native tools | Code Reviewer / QA |

**为什么是两个？**
- 核心就是 maker/checker 分离，这是 loop engineering 的基础
- 两个职业最少、最清晰，用户一看就懂
- 以后可以扩展（游侠=探索调研、刺客=安全审查），但 v2 就两个

**Mage 权限边界：**
- 文件系统：只读，不可修改任何文件
- Native tools：默认只下发只读工具（如 `Read` / `Glob` / `Grep` / `find` / `ls`）；不下发写工具、网络工具或通用 Shell。评审结论通过平台 ABI / agent CLI syscall 提交
- Shell：不作为默认 Mage native tool 暴露。若未来要支持测试命令白名单，必须由平台提供确定性 command allowlist 执行器，而不是把通用 Bash 交给 agent 后靠 prompt 约束
- 验证命令：通过平台 `command_run` / `gloop command run <command_id>` 执行，只接受 `mage_command_allowlist` 中的命令 ID；所有 argv、退出码、耗时和 stdout/stderr 摘要写入事件流
- 验证命令推荐：平台可基于项目文件生成候选命令，但不执行、不自动启用；用户确认后才进入 `mage_command_allowlist`
- 外部副作用：白名单命令可声明 `side_effect_level`，其中 `L2` 表示外部副作用；L2 命令默认拒绝，只有 automation 显式开启 `allow_l2` 且关闭 `auto_apply` 时才允许执行
- 网络：默认关闭；仅当 agent 支持可验证的网络开关时才可配置开启
- 所有命令执行记录到 events.jsonl，可审计
- 工作区同样受 `.gloop/scratch/` 隔离：可以读 scratch，但不能写
- 设计原则：Mage 可以阅读产物并提交评审；需要运行测试的能力必须先落到平台可审计、可限制的专用机制里，不能牺牲只读边界

### 2.2 冒险者

冒险者 = 职业 + 名字 + 绑定的 Agent + 自定义 prompt + 历史数据。

- **职业**：决定角色分工和默认工具权限
- **Agent**：底层 AI agent（通过 Agent 接入，ACP 或 CLI 方式）+ 模型配置
- **自定义 prompt**：用户可以加人设/风格
- **经验/等级/胜率**：历史表现，给用户看 + 影响自动派单优先级

### 2.2.5 Skills（技能包）

Skill 是分发给 agent 的 **agent-side instruction package**：一个包含 `SKILL.md`、元数据、示例和可选资源的目录。gloop 只负责索引、分发和按需读取，不解释 skill 语义，不根据 skill 替 agent 选择工具，也不参与业务决策。

**为什么要有 Skills？**
- 平台工具是 syscall/ABI 层能力（phase checkpoint / review submit / note append 等），只定义确定性协议
- Skill 把这些协议的适用边界、触发条件、反例和调用示例暴露给 agent，让 agent 自己判断是否加载和如何使用
- 采用 progressive disclosure：session 启动只注入 `name + description` manifest，正文通过 `gloop skill show <name>` 按需读取，避免把平台手册长期塞进上下文

**Skill 组成：**
- 名称 + 描述
- 触发条件与反例
- CLI / platform tool ABI 契约（统一 CLI Contract 格式：命令一览 / 详细说明 / 通用约定）
- 纪律约束：哪些判断必须由 agent 自己做，哪些状态必须显式提交给平台
- 元数据：`kind`（capability / orchestration）、`category`、`class`、`related_skills`（结构化引用关系：depends_on / related / see_also）
- 可选脚本或资源（后续扩展时仍由 agent 主动读取和执行）

**Skill 分类：**
- **`capability`（能力型）**：提供具体工具能力或方法论，不直接驱动循环状态
- **`orchestration`（编排型）**：直接参与 Quest 生命周期，驱动阶段推进或评审

`gloop skill list` 支持 `--kind` 和 `--category` 筛选，快速定位目标技能。

**内置 Skills（共 9 个，全部以 `gloop-` 前缀命名）：**

| Skill | Kind | 职业 | 说明 |
|-------|------|------|------|
| `gloop-quest-execution` | orchestration | 剑士 | 执行阶段协议：如何查询 quest、提交 phase checkpoint、在阻塞时显式上报 |
| `gloop-quest-review` | orchestration | 法师 | 评审阶段协议：如何提交 pass/request_changes/reject，以及 hints/score 的结构 |
| `gloop-quest-fanout` | orchestration | 剑士 | 委托扇出：将大任务拆分为多个独立并行委托 |
| `gloop-note-keeping` | capability | 通用 | 笔记协议：如何追加决策、风险、TODO 等跨阶段上下文 |
| `gloop-self-awareness` | capability | 通用 | 上下文查询协议：如何读取 quest、phase、history 等平台事实 |
| `gloop-user-context` | capability | 通用 | 用户全局工作上下文查询：从本地工作区、飞书等多源提炼的跨 quest 背景知识 |
| `gloop-user-notification` | capability | 通用 | 用户通知：重要里程碑、阻塞节点主动推送飞书消息 |
| `gloop-code-exploration` | capability | 通用 | 代码探索方法论：分层代码理解模型，减少重复扫代码 |
| `gloop-inbox-triage` | capability | 通用 | 收件箱智能分类：按优先级整理待处理委托 |

**设计原则：**
- Skill 是**agent 侧知识包**，不是平台调度单元
- 平台只做 manifest 暴露、正文读取和版本分发；平台不做 skill 路由、不根据自然语言猜测应该调用哪个工具
- Skill 可以描述平台能力，但不能增加平台语义判断；所有内容判断、质量判断、工具组合策略都留给 agent
- v2 先做官方只读内置 skills；用户自定义/业务域 skills 应优先进入 agent harness，而不是让 gloop 平台解释
- 创建 session 时只注入 skill manifest；完整文档必须按需读取
- Skill 间通过 `related_skills` 元数据建立结构化引用关系，支持发现与联动，但平台不做自动推荐

### 2.2.6 ContextPack 与 Prompt Boundary

ContextPack 是 Gloop 的 **prompt-side IR**。它不替 agent 做语义判断，而是在平台向 agent 交付上下文前，把不同来源的内容拆成稳定的结构化块，并显式标注边界。

**ContextPack block 元数据：**
- `name`：上下文块名称，如 `quest_user_intent`、`rework_hints`、`warrior_artifact`、`review_protocol`
- `source`：内容来源，如 `user`、`gloop`、`mage_review`、`warrior`
- `trust`：信任层级，如 `platform`、`user`、`agent_output`
- `phase`：所属阶段，如 `warrior`、`mage_review`
- `user_controlled`：内容是否由用户直接控制

**当前落地状态：**
- 执行阶段的用户 query、返工 hints、平台执行协议由 ContextPack 渲染为 `<gloop_context kind="quest_execution">`
- 评审阶段的原始 query、剑士产物、评审要求、评审协议由 ContextPack 渲染为 `<gloop_context kind="quest_review">`
- 评审阶段额外注入 `review_evidence_pack`：quest 类型、强度、返工计数、预算覆盖、上轮 hints、缓存 diff 摘要和证据规则；剑士交付是 claim，不是 proof
- 每轮发送前，orchestrator 会把 ContextPack 摘要写入 session trace（`kind=context_pack`）
- Debug API 已支持查看 session trace 和 ContextPack 摘要：
  - `GET /api/quests/{id}/sessions/{sid}`
  - `GET /api/quests/{id}/sessions/{sid}/context`

**安全边界说明：**
- XML-ish 标签是模型可读边界，不是安全边界
- Prompt injection 的平台边界仍然由工具 ABI、orchestrator 状态机、权限检查、预算检查和显式 phase/review signal 保证
- `trust=user` 或 `trust=agent_output` 的内容不能提升为 `trust=platform`
- 用户自定义 prompt 只属于风格/偏好层，不能覆盖平台权限边界、工具 ABI 或阶段协议

**裁剪策略：**
- ContextPack 支持基础字符预算裁剪：默认单 block 16k chars，单 pack 64k chars
- 被裁剪的 block 会带 truncated marker，并在摘要中标记 `truncated=true`
- 当前不是 token-aware 裁剪；token-aware budget 和按重要性裁剪属于后续增强

### 2.2.7 用户上下文（User Context）

用户上下文是**跨 quest 持久化的全局背景知识**，从本地工作区、飞书群聊/文档/日程等多源提炼而来，以摘要形式存储。agent 在执行任务时可以按需查询，快速了解用户的工作环境、技术偏好、近期关注等背景信息。

**为什么需要全局上下文：**
- 单个 quest 的上下文是局部的，agent 不知道用户整体在做什么、团队约定是什么
- 重复任务之间有共享知识（如代码风格、常用工具、关键联系人），每次重新收集浪费 token
- 渐进式披露：agent 先看到维度索引，按需加载详情，不把全量背景塞进每轮 prompt

**维度化设计：**

上下文按「维度」组织，每个维度是一个 Markdown 摘要文件：

| 维度 | 来源 | 说明 |
|------|------|------|
| `workspace` | 本地 Git 仓库 + 项目文件 | 当前工作区的项目结构、技术栈、关键模块 |
| `lark_im` | 飞书群聊 | 近期讨论主题、关键决策、团队约定 |
| `lark_doc` | 飞书云文档 | 相关文档摘要、设计方案、技术规范 |
| `lark_calendar` | 飞书日历 | 近期重要会议、排期、待办事项 |
| `gloop_history` | Gloop 历史委托 | 用户工作模式、偏好、常见任务类型 |

**存储结构（`~/.gloop/context/`）：**
- `summary.md`：总览摘要（一句话版本）
- `meta.json`：元信息（更新时间、各维度大小）
- `dim_<name>.md`：各维度详情（Markdown 格式）

**维度正文结构：**

每个 `dim_<name>.md` 优先采用稳定段落模板，降低 agent 消费成本：

章节标题使用英文作为稳定 schema anchor；正文默认中文，技术名词、命令、文件路径、API、状态枚举和产品名可保留英文。

每日刷新采用覆盖更新语义，而不是无限追加日志。短期事实按刷新窗口替换；稳定偏好、项目契约、反模式和检索线索可以跨天保留，但需要根据新证据修正、合并或删除过时项。

| 段落 | 用途 |
|------|------|
| `Scope` | 数据来源范围、时间新鲜度和主要覆盖对象 |
| `Current Focus` | 用户近期正在推进的项目、战役、风险或阻塞点 |
| `Stable Preferences` | 长期稳定的用户偏好、技术判断原则、沟通风格和验收口径 |
| `Project Contracts` | 项目/仓库/服务事实、边界、常用验证命令、API/前后端契约 |
| `Recent Decisions` | 近期明确拍板的结论，尽量带来源类型、新鲜度和置信度 |
| `Anti-Patterns` | 已被用户纠正或明确不希望重复发生的行为 |
| `Retrieval Index` | 需要细查时应该去哪里查的可行动线索 |
| `Gaps` | 无法确认、可能过时、存在冲突或需要二次确认的信息 |

`summary.md` 只作为路由入口，覆盖 Current Focus、Stable Preferences、Decision Rules、Freshness/Gaps，不复制所有维度正文。

**访问方式：**
- **平台工具 ABI**：`context_list` / `context_show` / `context_summary`（agent tool call）
- **CLI 命令**：`gloop context list/show/summary/refresh`（agent shell + 用户手动）
- **API**：REST 接口（前端 Dashboard 用）
- **Skill**：`gloop-user-context` 内置 skill，告诉 agent 何时以及如何查询上下文

**刷新机制：**
- 通过 `auto_context_refresh` 自动化定时刷新（默认每天 9:00）
- 用户可手动 `gloop context refresh` 触发
- 每个维度独立更新，不影响其他维度

**安全与隐私：**
- 只存摘要，不存原始数据（减少隐私泄露面 + 控制体积）
- 只保留主题级信息，不保存原始聊天、群名、链接、精确日程、个人安排、可识别参会信息或凭证
- 文件全部是 Markdown + JSON，用户可随时手改或删除
- 上下文是「参考」不是「事实」，关键信息需要 agent 二次确认
- 写权限区分：读权限两职业都有，写权限仅剑士（执行型任务可能需要补充上下文）

**设计原则：**
- 渐进式加载：先 list 看维度，再按需 show 详情，禁止一次性全量加载
- 仅供参考：上下文是摘要不是权威事实，关键信息需确认
- 用户可控：所有维度用户可见、可编辑、可删除
- 时效性意识：每个维度带更新时间，过时信息谨慎使用

### 2.3 Quest（委托）

Quest = 一次任务，分两种类型：

| 类型 | 说明 | 产出 |
|------|------|------|
| **execute** | 执行型：直接动手做 | 代码 / 文档 / 产物 |
| **design** | 方案型：讨论方案、出设计 | 方案文档 / 设计稿 |

**共同点**：都是剑士做 + 法师审 + 用户终审的两阶段 loop。
**区别**：产出物不同。方案型 quest 完成后，用户可以「基于此方案一键发起执行 quest」。

**状态机**：`pending → running → reviewing → user_review → success / failed / cancelled`

### 2.4 Agent 接入层

Gloop 通过 **Agent Agent** 抽象层接入所有 AI Agent。Agent 有两种类型：**ACP**（标准协议）和 **CLI**（命令行调用）。

#### ACP 方式（推荐）

ACP（Agent Client Protocol）是 Agent 通信的标准协议，类似于 LSP 之于编辑器：一套协议，对接所有 agent。

- **传输层**：JSON-RPC 2.0 over stdio
- **流式更新**：`session/update` notification 推送增量内容
- **优势**：接入零成本，一行配置就能接任何 ACP 兼容的 agent
- **当前 prompt renderer**：Gloop 使用 `<gloop_acp_history>` 分段 text content parts 保留 role/tool 边界；这不是完整的 ACP 原生 role/message 协议，后续可在 agent 能力允许时升级

#### CLI 方式（兼容）

不支持 ACP 的 agent，通过 CLI 命令行调用的方式接入。平台管理会话历史，每轮通过 stdin/stdout 交互。

- **适用**：relay 等暂不支持 ACP 的 agent
- **特点**：每个 agent 需要适配输出格式，接入成本略高
- **定位**：过渡期方案，等 agent 原生支持 ACP 后可无缝切换
- **当前 prompt renderer**：Gloop 使用 `<gloop_cli_history>` 展平 history，避免 `-p` 文本模式下丢失 role/tool 边界

#### v2 支持的 agent

| Agent | 接入方式 | 命令 | 说明 |
|-------|---------|------|------|
| **TraeX** | ACP | `traex acp serve` | 字节内部，强工具集成；不保留 TraeX CLI 专用适配 |
| **OMP** | ACP | `omp acp` | Oh My Pi；支持 ACP，按通用 ACP agent 配置接入 |
| **Relay** | CLI | `relay -p --verbose --output-format=stream-json` | ByteDance Relay，主力备选 |
| **Pi** | CLI | `pi --mode json --print` | Pi coding-agent；CLI JSON 事件适配 |
| **Codex** | ACP | `npx @zed-industries/codex-acp` | OpenAI 出品，强推理 |
| **Claude Code** | ACP | `npx --yes --package @agentclientprotocol/claude-agent-acp claude-agent-acp` | Anthropic 出品，长上下文 |

> 任何 ACP 兼容的 agent 都可以通过配置接入，不需要改平台代码。CLI 方式的 agent 也只需要一个 Agent 配置。

**Agent 能力矩阵：**

| 能力 | ACP 型 | CLI 型 | 说明 |
|------|--------|--------|------|
| 流式输出（SSE） | ✅ 原生支持 | ⚠️ 需适配输出格式 | 事件流推送体验不同 |
| 工具调用事件 | ✅ 结构化 | ⚠️ 需解析输出 | 影响 micro loop 事件粒度 |
| Session 复用 | ✅ 原生 | ❌ 每次新建 | CLI 方式每轮都是独立进程 |
| 取消 / 中断 | ✅ 原生 | ⚠️ 进程 kill | CLI 方式可能丢最后状态 |
| Token 统计 | ✅ 协议级 | ⚠️ 需 agent 自行上报 | 影响预算管控精度 |
| 多模型切换 | ✅ 原生 | ⚠️ 靠命令行参数 | 取决于具体 agent 实现 |
| 接入成本 | 低（配置化） | 中（需适配输出） | ACP 是方向，CLI 是过渡 |

> ACP 落地状态：ACP client 已接入 `usage_update`（真实 token 统计）、`tool_call` / `tool_call_update`（结构化工具调用事件），并通过 `session/cancel` 支持取消/中断。`Resume` 因 ACP 协议无对应语义保留为 no-op。

**MVP 策略：** 先跑通 1-2 个 agent（建议 TraeX ACP + Relay CLI），验证闭环后再扩充。不追求一开始就支持 4 个。

**启用策略：** `gloop init` 只生成 agent 模板，不自动启用真实 agent。即使探测到本机二进制，
也只作为候选记录；用户需要在验证账号/模型可用后显式启用 agent，并在冒险者配置中绑定。
Mock executor 始终作为兜底可用。

### 2.5 工作区（Workspace）

每个 quest 有自己的独立工作区：
- **git 项目 + 写操作** → git worktree 隔离
- **非 git 项目 + 写操作** → 目录拷贝隔离
- **只读任务** → 直接用原目录

工作区内有一个系统保留目录 `.gloop/scratch/`，是 agent 的自留地：
- agent 可以在里面存放中间状态、临时文件、草稿
- 同 quest 内所有 session（剑士/法师、返工轮次）共享
- apply / diff 时系统自动排除，不会污染用户项目
- 系统不管理目录内容，纯 agent 自治

用户终审通过后，可以选择 **apply**（合入主目录）或 **discard**（丢弃）。

### 2.6 系统与 Agent 的职责边界

**核心原则：系统做机制，agent 做决策；系统管流转，agent 管内容。**

判断一件事该谁做，用两个标准：
1. **能不能不靠 AI 做？** 能 → 系统做。不能 → agent 做。
2. **是不是机制性的？** 是 → 系统做。否 → agent 做。

具体边界：

| 职责 | 系统 | Agent |
|------|------|-------|
| 阶段流转、状态机 | ✅ | ❌ |
| 工作区隔离、apply/diff | ✅ | ❌ |
| 权限控制（工具白名单） | ✅ | ❌ |
| 预算/停止条件（turn 数、返工次数） | ✅ | ❌ |
| 消息/产物/notes 的传递 | ✅（搬运） | ❌ |
| 任务理解、方案设计 | ❌ | ✅ |
| 自己的推理过程、记忆组织 | ❌ | ✅ |
| 产物内容的质量 | ❌ | ✅（做的人）/ ✅（评的人） |
| 结构化状态（如 todo 列表、中间结果）怎么存、怎么用 | ❌ | ✅ |

**关于 Memory：**
- 系统不做内容级的记忆提取、摘要、检索
- 系统提供传递机制：notes、scratch 目录、历史查询接口
- 记忆怎么组织、怎么用，是 agent 自己的事
- 冒险者的经验/等级/胜率只做展示和派单排序，不构成"记忆"

**关于用户上下文（User Context）：**
- 系统提供**跨 quest 的全局上下文存储**（workspace / lark 等维度摘要），作为 agent 启动前可获取的背景知识
- 上下文是**摘要**，不是原始数据；由自动化定期刷新，用户可手动刷新
- 系统只管存储和刷新机制，**不做语义检索、不替 agent 选择加载哪个维度**（agent 通过 skill 自主判断何时查询、查哪个维度）
- 上下文是参考信息，不构成系统状态或任务状态；agent 仍需自己判断信息相关性和时效性

## 3. 核心流程

### 3.1 执行型 Quest（execute）

```
用户输入一段 query + 选择"直接执行"
  ↓
[系统] 创建 quest，分配剑士
  ↓
[剑士 session 启动]
  剑士理解任务 → 执行
  → 调用 gloop phase done 标记完成
  ↓
[系统自动转交法师]
  ↓
[法师 session 启动]
  法师理解任务 → 自己定验收标准 → 审查结果
  → 通过：调用 gloop review pass
  → 打回：调用 gloop review request-changes --hints "..."
  → 拒绝：调用 gloop review reject
  ↓
通过？→ 是 → 进入用户终审
    → 否 → 打回剑士返工（新开 session，产物+意见传回去）
    → 达返工上限 → 进入用户终审
      ↓
用户终审
  → 通过 + apply → 改动合入主目录，quest success
  → 通过 + discard → 只记录结果，不应用改动，quest success
  → 返工 → 继续循环
  → 拒绝 → quest failed，工作区可清理
```

### 3.2 方案型 Quest（design）

```
用户输入一段 query + 选择"方案讨论"
  ↓
[系统] 创建 quest，分配剑士
  ↓
[剑士 session 启动]
  剑士理解任务 → 出方案
  → 调用 gloop phase done 标记方案完成
  ↓
[系统自动转交法师]
  ↓
[法师 session 启动]
  法师审查方案 → 给出评审意见
  → 通过：调用 gloop review pass
  → 打回：调用 gloop review request-changes
  → 拒绝：调用 gloop review reject
  ↓
通过？→ 是 → 进入用户终审
    → 否 → 返工循环
    → 达上限 → 进入用户终审
      ↓
用户终审
  → 通过 → quest success
    ↓ 用户可以点「基于此方案发起执行」→ 自动创建 execute 型 quest
  → 返工 → 继续改方案
  → 拒绝 → quest failed
```

### 3.3 关键设计决策

**方案是独立 quest，不是 phase。**
- 职责清晰：方案型就出方案，执行型就干活
- 可复用：一个方案可以发起多次执行
- 用户体验好：方案满意了再点开始执行

**默认哪种类型？**
- 用户创建时可以选：「直接执行」 / 「先出方案」
- 默认：直接执行（execute）
- 系统不做关键词判断，用户自己选

**法师"忘记"调用评审工具怎么办？**
- 系统 prompt 明确要求：必须调用 `gloop review` 给出结论
- 系统检测：session 结束（达到 max turns / finish_reason=stop）时如果没调过 review → 补一条 hint
- 补 hint 次数：默认 2 次，用户可配置
- 补完还没调 → 标记"评审超时"，直接进入用户终审
- 这是系统机制，不是智能判断

**返工新开 session 还是续跑？**
- **新开 session**
- 上下文隔离，避免越改越乱
- 之前的产物 + 修改意见作为初始输入传进去

**推进状态必须显式调 CLI。**
- 系统不猜"agent 是不是干完了"
- 剑士必须调 `gloop phase done` 才算阶段完成
- 法师必须调 `gloop review xxx` 才算评审完成
- 系统只做兜底检测和补 hint

**discard 之后还能重做吗？**
- discard 是用户终审的结果之一（通过但不应用改动）
- quest 一旦到了 success/failed 就是终态，不能再返工
- 想重做 → 基于原 quest 一键创建新 quest（复制 query 和配置）

### 3.4 预算与停止条件

预算 = 停止条件。不是为了省钱，是防止 loop 无限跑。

| 预算维度 | 默认值 | 说明 |
|---------|--------|------|
| `max_turns_per_session` | 10 | 单 session 最大 turn 数 |
| `max_rework_per_quest` | 3 | 最大返工次数 |
| `max_duration_minutes` | 180 | 单 quest 总耗时上限（wall clock） |
| `review_hint_retries` | 2 | 法师忘调 review 时补 hint 次数 |
| `user_confirm_timeout_hours` | 72 | 等待用户确认的超时时间（超时自动挂起） |

委托强度是预算档，不是语义分类器；Gloop 本身没有智能，不判断任务难度，只执行显式选择。

| 强度 | 用户含义 | 预算行为 |
|------|----------|----------|
| `quick` | 快速 | 4 turn / 1 次返工 / 30 分钟 quest |
| `standard` | 标准 | 跟随全局配置，默认 10 turn / 3 次返工 / 3 小时 quest |
| `deep` | 深入 | 20 turn / 4 次返工 / 8 小时 quest |
| `adversarial` | 严审 | 20 turn / 5 次返工 / 12 小时 quest，法师需要更强证据链 |

Token、底层 native tool 调用次数和模型内部 budget 属于 agent/agent 内部控制面。Gloop 可以记录 agent 上报的 token/usage 作为观测数据，但不把 token 当作平台停止条件；否则平台会越界干涉 agent 自己的上下文和推理策略。

ContextPack 另有 prompt-side 字符预算裁剪，用于限制单个上下文块和整包上下文的输入规模；它是 prompt 组装层的保护，不替代 agent/agent 的 token 预算。

**触发规则：**
- 达到任一预算上限 → 立即停止当前阶段，进入 `blocked` 状态
- 用户可以通过 blocked 恢复接口选择继续（可追加该 quest 的 turn / duration 预算）、直接进入用户终审，或取消
- 预算超限时发送事件通知，前端展示告警
- 连续 3 轮无有效进展（无新文件改动、无新的推理产出）→ 视为卡住，自动停止

### 3.5 Agent 失败处理

系统必须处理 agent 各种异常情况，不能让单个 agent 的失败拖垮整个 quest。

| 失败场景 | 检测方式 | 处理策略 |
|---------|---------|---------|
| **进程崩溃 / 异常退出** | executor 返回错误 | 记录错误；连续错误达到 `max_consecutive_agent_errors` 后进入 `blocked` |
| **输出不可解析** | ACP 消息格式错误 / CLI 输出非预期格式 | 作为 executor 错误记录；连续达到阈值后进入 `blocked` |
| **超时** | 达到 session / 阶段时间限制 | 强制终止 agent 进程，进入 `blocked` |
| **忘记调用推进命令** | 达到 max_turns 仍未调 phase done / review | 补 hint（最多 review_hint_retries 次）；仍不调 → 标记阶段超时，直接进入下一环节或用户终审 |
| **工具调用失败** | 工具执行错误 / 权限不足 | 由 agent 自行处理；连续失败超过阈值则进入 `blocked` |
| **无意义循环** | 重复相同的工具调用或输出 | 检测到连续 N 轮无进展 → 补 hint；仍无效 → 进入 `blocked` |

**通用原则：**
- 失败可观测：所有异常都记录事件、保留现场（工作区、日志）
- 失败可恢复：用户可以选择重试、跳过、或手动接管
- 不静默重试：除非是明确可恢复的瞬时错误，否则不自动重跑（防止浪费 token / 放大破坏）

## 4. 平台 CLI 工具（Agent 用）

Agent 通过 `gloop` CLI 与平台交互。这是 agent 操作系统的唯一官方渠道。

实现约定：agent CLI syscall 通过工作区内的 `.gloop/context.json` 发现当前
`quest_id / session_id / data_dir`，再把阶段结束信号写入
`.gloop/signals/<session_id>.json`。orchestrator 每轮读取并消费该 signal，
因此 CLI 与 native tool 都是确定性的落盘协议，不是进程内魔法状态。

### 4.1 通用命令

```bash
# 查看当前 quest 信息（JSON 输出，方便 agent 解析）
gloop quest info --format json

# 查看当前阶段
gloop phase info

# 上报进度
gloop quest progress --percent 50 --note "正在做xxx"

# 查看历史（前 N 轮摘要）
gloop quest history --limit 5

# 记录备注（append-only，所有职业可用）
gloop note add "发现配置文件路径，下一步读取"
gloop note add --tag risk "此操作可能删除文件，需用户确认"

# 列出已留痕的备注（只读，供 agent 自查上下文）
gloop note list --limit 50

# 列出可用的用户上下文维度（全局，跨 quest）
gloop context list

# 读取某个上下文维度详情（全局，跨 quest）
gloop context show workspace

# 读取上下文总览摘要（一句话版本，全局）
gloop context summary
```

**Note 的定位：**
- 纯文本备注，append-only，不可修改不可删除
- 属于当前 session / 当前 phase，会出现在事件流中
- 阶段切换时，上一阶段的 notes 会作为摘要自动传给下一阶段
- 返工新开 session 时，之前所有轮次的 notes 会一并传入
- 系统提供只读的 `gloop note list`（供 agent 自查已留痕的备注），但不提供可变更或语义检索的查询接口；agent 如需持久化结构化状态，请写入工作区文件

### 4.2 剑士可用

```bash
# 标记当前阶段完成（交付结果），自动转下一环节
gloop phase done --summary "完成了xxx"

# 请求用户输入（需要用户决策时）
gloop quest ask --question "xxx"

# 标记任务无法完成
gloop phase fail --reason "原因"

# 写入（创建或覆盖）用户上下文维度（全局，跨 quest；剑士可写）
gloop context write-dim --name custom_notes --content "# 自定义笔记\n..."

# 写入上下文总览摘要（全局；剑士可写）
gloop context write-summary --content "用户近期主要在做..."
```

### 4.3 法师可用

```bash
# 评审通过
gloop review pass --comment "代码质量良好" --score 8

# 要求修改
gloop review request-changes --comment "xxx有问题" --hints "具体改什么"

# 拒绝（重做）
gloop review reject --comment "方向错了，重做"
```

### 4.4 设计原则

- CLI 是 agent 和平台交互的唯一官方渠道
- 命令要少而精，agent 容易学会
- 输出格式稳定（JSON 优先），方便 agent 解析
- 系统只认 CLI 命令的结果，不猜 agent 的意图
- 提供 agent-side SKILL.md 描述命令协议和边界；平台只分发，不解释 skill 语义

## 5. 冒险者系统

### 5.1 配置格式

每个冒险者一个 JSON 文件，存在 `~/.gloop/adventurers/` 下。

```json
{
  "id": "adv_warrior_001",
  "name": "格雷·暴风",
  "class": "warrior",
  "description": "行动力强的剑士，擅长写代码和跑流程",
  "agent": "traex",
  "model": "auto",
  "fallback_models": [],
  "custom_prompt": "",
  "tools": ["bash", "read", "edit", "write", "glob", "grep", "web_fetch"],
  "status": "pending_setup",
  "level": 1,
  "exp": 0,
  "win_count": 0,
  "lose_count": 0
}
```

`agent` 字段对应 `agents/` 下的 agent 配置名。

### 5.2 招募引导

`gloop init` 不创建默认冒险者。冒险者是用户确认过的执行身份，绑定具体 Agent、职责、人设与工具边界；未确认前不应静默落盘成可派单角色。

首次使用由 Dashboard 首页引导：
1. 先配置并启用至少一个 Agent
2. 再招募剑士和法师
3. 两个职业都有 active 冒险者后，才能发起委托

招募表单可以按职业提供推荐预填，但保存行为必须由用户显式触发。

### 5.3 自动派单规则

创建 quest 时选择冒险者的优先级：
1. 用户手动指定了 → 用指定的
2. 否则 → 选同职业下胜率最高、等级次之的 active 冒险者
3. 没有同职业的 active 冒险者 → 报错，提示用户先在 Dashboard 招募或激活对应职业冒险者

> 设计说明：剑士/法师职责与默认工具权限不同，跨职业派单会破坏 maker/checker 分离与权限边界，因此没有同职业可用时直接报错而非跨职业兜底。「继承上次同类型任务所用冒险者」属后续增强，当前按胜率→等级排序已能收敛到稳定选择。

经验等级影响的是**优先级排序**，不是权限。

### 5.4 经验与等级

**10 级上限**，每级所需经验递增（差 100/200/300/...）。

| 等级 | 累计经验 |
|------|---------|
| Lv.1 | 0 |
| Lv.2 | 100 |
| Lv.3 | 300 |
| Lv.4 | 600 |
| Lv.5 | 1000 |
| Lv.6 | 1500 |
| Lv.7 | 2100 |
| Lv.8 | 2800 |
| Lv.9 | 3600 |
| Lv.10 | 4500 |

**等级称呼（按职业区分）**

| 等级 | 剑士称呼 | 法师称呼 |
|------|---------|---------|
| Lv.1 | 新兵 | 学徒 |
| Lv.2 | 剑士 | 法师 |
| Lv.3 | 大剑士 | 大法师 |
| Lv.4 | 剑豪 | 魔导士 |
| Lv.5 | 剑圣 | 魔导师 |
| Lv.6 | 剑尊 | 大魔导师 |
| Lv.7 | 剑神 | 贤者 |
| Lv.8 | 宗师 | 宗师 |
| Lv.9 | 传奇剑客 | 传奇法师 |
| Lv.10 | 神话剑士 | 神话法师 |

**经验获取规则：**
- 用户评审通过：基础 100 exp × 质量系数
  - 法师评审给高分（≥8分）：× 1.5
  - 法师评审中等（5-7分）：× 1.0
  - 法师评审低分（<5分）但用户还是通过了：× 0.5
- 每返工一次：最终经验 -20%（最低 0.5 倍保底）
- 用户评审驳回：10 exp（参与奖）

**设计原则：** 经验等级只做展示 + 自动派单排序，不影响 agent 能力和权限。

## 6. 工作区隔离

### 6.1 三种模式

| 模式 | 说明 | 适用场景 |
|------|------|----------|
| `worktree` | git worktree 隔离，可 apply/discard | git 项目 + 写操作 |
| `copy` | 完整目录拷贝，可 apply/discard | 非 git 项目 + 写操作 |
| `readonly` | 直接用原目录，只读 | 查询/分析/调研/方案讨论 |

### 6.2 选择策略

由用户指定。系统不自动判断。

- 创建 quest 时可以选工作区模式
- 剑士也可以在执行过程中切换（仅限更严格的模式）
- 默认 `auto`：如果工作目录是 git repo 选 `worktree`；否则选 `copy`。`readonly` 必须由用户、automation 或上层 UI 显式选择。

### 6.3 Apply / Discard

用户终审通过后：
- **Apply**：把改动合回原目录
  - worktree 模式：通过 `git diff --binary` + `git apply --index` 应用，并生成一条 apply commit
  - copy 模式：按目录 diff 把文件增删改同步回原目录
- **Discard**：丢弃改动，只保留 quest 记录
- **先看 diff 再决定**：用户可以先看 diff 再选

**Apply 安全规则：**
- 创建 quest 时记录 base commit，apply 时校验主分支 HEAD 是否前进
- 如果主分支已前进 → 拒绝 apply，提示用户先 rebase 或手动处理
- 主工作区有未提交改动（dirty） → 拒绝 apply，防止污染
- patch 存在冲突 → 拒绝 apply，返回冲突文件列表
- worktree 模式基于 git patch，支持 git 原生能力（含增删改、重命名、二进制文件）；copy 模式仅按目录 diff 处理文件增删改
- apply 前自动生成备份快照，失败可回滚
- 所有 apply 操作记录到 events.jsonl，包含 patch 引用（备份路径）、改动文件数与结果

### 6.4 清理

- quest 完成后工作区保留 7 天（默认，可配置）
- 用户可以手动清理
- 失败/取消的 quest 工作区可设置为自动清理

### 6.5 Agent Scratch 目录

每个 quest 工作区内有一个系统保留目录 `.gloop/scratch/`，供 agent 存放中间状态和临时文件。

**规则：**
- 系统在创建工作区时自动建立该目录
- 同 quest 内所有 session 共享，返工轮次也可读取
- 所有职业（剑士/法师）均可读写
- apply 时自动排除，不会合入用户项目
- diff 时自动排除，不会出现在改动列表中
- 系统不管理、不解析目录内容，纯 agent 自治

**与 note 的区别：**

| 维度 | note | scratch 目录 |
|------|------|-------------|
| 用途 | 留痕、交接备注 | agent 内部状态、临时文件 |
| 用户可见 | 是（事件流中） | 否 |
| 格式 | 纯文本，单条 | 任意文件 |
| 修改性 | append-only，不可改删 | 自由读写 |
| 系统态度 | 结构化传递给下一轮 | 完全不管 |

**典型用法：**
- 剑士存 `plan.md`、`todo.json`、`discovered_issues.md`
- 法师存 `review_notes.md`
- 返工轮次读取上一轮的思考记录，避免从零开始

## 7. 自动化与 Discovery

### 7.1 Automation

用户可以配置自动化任务，定时运行，自动发现问题并创建 quest。

配置文件存 `~/.gloop/automations/`：

```json
{
  "id": "auto-lint",
  "name": "每日代码检查",
  "trigger": "schedule",
  "cron": "daily",
  "working_dir": "/path/to/project",
  "query": "扫描项目中的 lint 错误和明显的 bug，给出修复建议",
  "quest_type": "execute",
  "warrior_id": "",
  "mage_id": "",
  "auto_start": false,
  "auto_apply": false,
  "enabled": false,
  "tags": ["official"]
}
```

- `trigger`: `manual`（仅 API/CLI 手动触发）/ `schedule`（按 cron 定时触发）
- `cron`: schedule 模式的调度表达式，支持 daily / hourly / weekly alias 或标准 5 段 cron
- `auto_start`: true = 创建后自动开始执行（不进 inbox）；false = 进 inbox 等用户确认
- `auto_apply`: true = 用户终审通过后自动 apply（慎用，仅限低风险任务）；false = 等用户手动 apply
- `enabled`: 默认 false，用户手动开启
- `tags`: 标记 quest 来源，"official" 表示官方委托，"template" 表示开箱模板

> 设计说明：把"自动执行"拆成 `auto_start`（是否自动开跑）与 `auto_apply`（是否自动合入）两个正交开关，是为了把不同风险级别分离——自动跑一个只读分析远比自动 apply 代码改动安全。`allow_l2` 是第三个独立安全开关，只允许显式标记 L2 的外部副作用命令；它与 `auto_apply` 互斥。

### 7.2 开箱模板

初始化时预置几个自动化模板（默认关闭）：

| 模板名 | 说明 | 类型 |
|--------|------|------|
| 每日代码检查 | 扫描常见代码问题、潜在 bug 和可改进点，输出分级检查报告 | execute |
| 代码评审助手 | 审查当前工作区与主分支的差异，给出质量评审意见 | design |
| 依赖安全检查 | 检查第三方依赖是否有已知漏洞，给出升级建议 | execute |
| 审计已完成委托 | 二次审查最近完成的 quest，输出审计报告 | design |
| 用户上下文刷新 | 每日定时刷新用户全局工作上下文摘要（workspace / lark 等多源） | context_refresh |

用户可以直接启用，也可以复制一份改改再用。

### 7.3 Triage Inbox

Automation 创建的 quest 进入"待处理"列表（Inbox）。用户可以：
- 接受 → 开始执行
- 拒绝 → 关闭
- 编辑 → 改改描述再开始

### 7.4 官方委托

Automation 创建的 quest 标记为「官方委托」，跟用户手动创建的区分开。

## 8. 数据模型

### 8.1 目录结构

```
~/.gloop/
├── config.json
├── agents/
│   ├── traex.json        ← Agent agent 配置（ACP 型）
│   ├── omp.json          ← Agent agent 配置（ACP 型）
│   ├── relay.json        ← Agent agent 配置（CLI 型）
│   ├── pi.json           ← Agent agent 配置（CLI 型）
│   ├── codex.json
│   └── claude.json
├── adventurers/
│   └── <user-created-adventurer>.json
├── automations/
│   └── auto-lint.json
├── context/              ← 用户全局上下文（跨 quest）
│   ├── summary.md        ← 总览摘要
│   ├── meta.json         ← 元信息
│   ├── dim_workspace.md  ← 维度：本地工作区
│   ├── dim_lark_im.md    ← 维度：飞书群聊
│   └── dim_*.md          ← 其他维度
└── workspace/
    ├── quests/
    │   └── qst_<shortid>/
    │       ├── meta.json
    │       ├── sessions/
    │       │   ├── warrior_run1.jsonl
    │       │   └── mage_run1.jsonl
    │       ├── events.jsonl
    │       ├── reviews.jsonl
    │       └── work/              ← 工作目录（业务代码）
    │           └── .gloop/scratch/  ← agent 自留地（apply/diff 时排除）
    └── stats/
        └── adv_*.json
```

### 8.2 Agent 配置

每个 agent agent 一个 JSON 配置文件，存在 `~/.gloop/agents/` 下。Agent 有两种类型：`acp` 和 `cli`。

**ACP 型 agent：

```json
{
  "name": "traex",
  "type": "acp",
  "command": "traex",
  "args": ["acp", "serve"],
  "env": {},
  "default_model": "auto",
  "enabled": false
}
```

**CLI 型 agent：

```json
{
  "name": "relay",
  "type": "cli",
  "command": "relay",
  "args": ["-p", "--verbose", "--output-format=stream-json"],
  "env": {},
  "default_model": "auto",
  "enabled": false
}
```

- `type`: `acp` 或 `cli`
- `command` + `args`: 启动命令
- `default_model`: 默认模型名
- 新增 agent = 新增一个 JSON 文件，不改代码（ACP 型完全配置化；CLI 型如输出格式不同需适配代码，但接口层已有实现的可直接复用）

### 8.3 QuestMeta

```go
type QuestMeta struct {
    ID          string `json:"id"`
    ShortID     string `json:"short_id"`
    Query       string `json:"query"`           // 用户输入的原始文本
    Type        string `json:"type"`            // execute / design
    Status      model.QuestStatus `json:"status"`

    // 冒险者
    WarriorID   string `json:"warrior_id"`
    MageID      string `json:"mage_id"`

    // 工作区
    WorkspaceMode   string `json:"workspace_mode"` // worktree/copy/readonly
    WorkspacePath   string `json:"workspace_path"`
    BaseWorkingDir  string `json:"base_working_dir"`
    BaseBranch      string `json:"base_branch,omitempty"`
    BaseCommit      string `json:"base_commit,omitempty"`

    // 过程
    ReworkCount     int    `json:"rework_count"`
    MaxRework       int    `json:"max_rework"`
    ReviewHints     string `json:"review_hints,omitempty"`
    DesignSummary   string `json:"design_summary,omitempty"` // 方案型 quest 的方案摘要
    ParentQuestID   string `json:"parent_quest_id,omitempty"` // 从方案 quest 发起的执行 quest

    // 结果
    FinalVerdict    model.QuestVerdict `json:"final_verdict,omitempty"`
    FinalComment    string `json:"final_comment,omitempty"`
    Applied         bool   `json:"applied,omitempty"`

    // 元信息
    CreatedBy       string `json:"created_by"`   // user / automation:xxx
    Official        bool   `json:"official,omitempty"`
    CreatedAtMs     int64  `json:"created_at_ms"`
    StartedAtMs     int64  `json:"started_at_ms,omitempty"`
    CompletedAtMs   int64  `json:"completed_at_ms,omitempty"`
}
```

### 8.3 事件类型

```
quest.created
quest.started
quest.phase_changed
quest.review_submitted
quest.rework
quest.user_review
quest.success
quest.failed
quest.cancelled
quest.applied
quest.discarded
quest.note

workspace.created
workspace.diff_ready

micro.turn
micro.token_delta
micro.assistant_msg
micro.tool_start
micro.tool_end
```

### 8.4 数据留痕与隐私

**留痕原则：**
- 所有关键状态变更都落盘（事件驱动，append-only）
- agent 的完整输入输出都可追溯
- 用户可随时导出、清理、删除任何 quest 的数据

**保存的内容：**
- quest 元信息（状态、配置、时间戳）
- 事件流（events.jsonl）
- session 历史（prompt、assistant 消息、工具调用、工具结果）
- 评审记录（reviews.jsonl）
- 工作区产物（work/ 目录，含代码改动）
- 冒险者数据（等级、经验、胜率）

**不保存的内容：**
- 不在 gloop 工作区范围内的用户文件
- 用户的其他项目代码（除非 quest 工作区包含）
- Agent 方的账号凭证（只存引用，不存明文）

**隐私与安全：**
- 所有数据存在 `~/.gloop/` 下，用户完全可控
- 不上传任何用户代码到第三方服务器（除了用户配置的 agent agent 本身）
- 可配置敏感文件排除规则（`.gloopignore`）：支持空行、`#` 注释、精确路径、目录前缀和基础 glob；copy/worktree 的 diff/apply 都必须排除这些路径
- 删除 quest 时可选择是否保留工作区产物
- 日志脱敏：默认不在日志中打印完整代码内容（可配置开启 debug 模式）

## 9. API

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/quests` | 创建 quest（`{ query, type?, working_dir?, warrior_id?, mage_id?, workspace_mode? }`） |
| POST | `/api/quests/cleanup` | 按 retention 清理终态 quest 工作区（默认 dry-run） |
| GET | `/api/quests` | 列表（支持 status/type/created_by 过滤） |
| GET | `/api/quests/:id` | 详情 |
| POST | `/api/quests/:id/start` | 开始执行 |
| POST | `/api/quests/:id/comment` | 用户评论 |
| POST | `/api/quests/:id/resolve-blocked` | 处理 blocked quest（`action: continue/user-review/cancel`, `add_turns?`, `add_duration_minutes?`） |
| POST | `/api/quests/:id/resolve` | 用户终审（`verdict: pass/reject/rework`, apply?: bool） |
| POST | `/api/quests/:id/resolve-user-review` | 用户终审兼容别名（当前实现保留） |
| POST | `/api/quests/:id/apply` | 应用改动 |
| POST | `/api/quests/:id/discard` | 丢弃改动 |
| GET | `/api/quests/:id/diff` | 获取 diff |
| GET | `/api/quests/:id/backups` | 列 apply 备份 |
| GET | `/api/quests/:id/backups/:backup_id` | 查看 apply 备份元信息 |
| POST | `/api/quests/:id/cancel` | 取消 |
| POST | `/api/quests/:id/stop` | 取消兼容别名（当前实现保留） |
| GET | `/api/quests/:id/stream` | SSE 事件流 |
| GET | `/api/stream?qid=...` | 全局 SSE 事件流（当前实现保留） |
| POST | `/api/quests/:id/spawn-execute` | 基于方案 quest 发起执行 quest |
| GET | `/api/adventurers` | 冒险者列表 |
| GET | `/api/adventurers/:id` | 冒险者详情 |
| POST | `/api/adventurers/:id/activate` | 确认配置，激活冒险者 |
| GET | `/api/inbox` | 待处理列表（automation 创建的） |
| POST/PATCH | `/api/inbox/:id` | 编辑待处理委托 |
| POST | `/api/inbox/:id/accept` | 接受 |
| POST | `/api/inbox/:id/reject` | 拒绝 |
| GET | `/api/automations` | 自动化任务列表 |
| POST/PATCH | `/api/automations/:id` | 更新 automation 配置 |
| POST | `/api/automations/:id/run` | 手动触发 automation |
| POST | `/api/automations/:id/enable` | 启用 |
| POST | `/api/automations/:id/disable` | 禁用 |
| GET | `/api/context` | 上下文元信息（更新时间、总大小等） |
| GET | `/api/context/dims` | 上下文维度列表（元信息，不含正文） |
| GET | `/api/context/dims/:name` | 单个维度详情（含 Markdown 正文） |
| PUT | `/api/context/dims/:name` | 写入/覆盖维度内容 |
| GET | `/api/context/dims/:name/export` | 导出单个维度为 .md 文件 |
| GET | `/api/context/summary` | 总览摘要 |
| POST | `/api/context/refresh` | 手动触发上下文刷新 |

## 10. CLI（用户侧）

```bash
# 核心
gloop run "给项目加个登录页面"           # 直接发起执行 quest
gloop design "重新设计权限系统"          # 发起方案讨论 quest
gloop quest list                        # 列表
gloop quest show <qid>                  # 详情
gloop quest review <qid> pass|rework|reject  # 评审
gloop quest apply <qid>                 # 应用改动
gloop quest discard <qid>               # 丢弃改动
gloop quest diff <qid>                  # 看 diff
gloop quest cancel <qid>                # 取消
gloop quest resolve-blocked <qid>       # 处理 blocked：继续 / 转终审 / 取消
gloop quest cleanup [--dry-run]         # 按 retention 清理已完成工作区，默认 dry-run
gloop quest spawn-execute <qid>         # 基于方案 quest 发起执行

# 冒险者
gloop adventurer list
gloop adventurer show <id>
gloop adventurer activate <id>

# 自动化
gloop automation list
gloop automation run <id>
gloop automation edit <id> --query "..." --type execute --auto-start
gloop automation enable <id>
gloop automation disable <id>
gloop automation run <id>

# 上下文（全局，跨 quest）
gloop context list                    # 列出可用维度
gloop context show <dim_name>         # 读取维度详情
gloop context summary                 # 总览摘要
gloop context refresh                 # 手动触发刷新
gloop context export --format okf --out ./gloop-knowledge

# 系统
gloop init
gloop server start
gloop config show
```

## 11. 前端

### 11.1 设计原则

- 移动端优先
- 信息密度适中，不堆日志
- 关键动作好找（通过/返工/拒绝、apply/discard）
- 工会主题做皮肤，不绑架核心功能

### 11.2 主要页面

**首页（Quests）**
- 进行中的 quest 卡片
- 待评审的 quest（突出显示）
- 底部大按钮：「发布委托」
  - 点了弹底部 sheet：输入一段描述 + 选择类型（直接执行 / 先出方案）

**Quest 详情**
- 状态条 + 当前阶段
- 时间线：剑士做 → 法师审 → 返工 → 法师审 → ...
- 产物预览
- 法师验收报告
- 评审操作区（通过/返工/拒绝）
- Apply / Discard（通过后显示）
- Diff 查看
- 方案型 quest 增加：「基于此方案发起执行」按钮

**冒险者**
- 冒险者卡片列表（职业分类）
- 等级、胜率、状态
- 点进去看详情、历史、配置
- 新冒险者需要确认配置才能激活

**Inbox**
- 官方委托 / 自动化发现的候选 quest
- 接受 / 拒绝 / 编辑

**设置**
- ACP Agent 配置（列表 + 开关）
- 自动化配置（列表 + 开关）
- 默认工作目录
- 补 hint 次数配置

### 11.3 工会主题怎么体现

- 名字：委托、冒险者、剑士、法师、官方委托
- 视觉：卡片风格、徽章、等级标识
- 而不是：六维属性、天赋、稀有度、抽卡……这些跟 loop 没关系的

> 本节内容已整合进「1.2 非目标」，分为「暂不做」和「原则上不做」两类，详见前文。

## 13. 实施路线

### Phase 0: 准备
- [x] 数据模型重构（QuestMeta / Adventurer）
- [x] 平台 CLI 框架（agent 侧命令）
- [x] fsstore 调整
- [x] 状态机重写
- [x] Agent 抽象层：统一 ACP + CLI 两种接入方式
- [x] ACP 接入实现（JSON-RPC 2.0 client + ACPExecutor）
- [x] CLI 方式保留并适配（Relay / Pi / Codex；TraeX / OMP 走 ACP 配置）
- [x] 预置 agent 配置：traex (ACP) / omp (ACP) / relay (CLI) / pi (CLI) / codex (ACP) / claude (ACP)，默认均关闭

### Phase 1: 核心循环（execute 型 quest）
- [x] 剑士执行 loop
- [x] 法师验收 loop
- [x] 返工循环
- [x] 用户终审
- [x] 内置 Dashboard：quest 列表 + 详情 + 评审
- [x] 用户侧 CLI 基础命令

### Phase 2: Workspace 隔离
- [x] git worktree 支持
- [x] directory copy 支持
- [x] Apply / Discard 基础能力
- [x] Diff 计算 + 展示
- [x] 内置 Dashboard diff 面板
- [x] PRD 级 apply 安全策略收紧：dirty 拒绝、base 前进拒绝、git diff / git apply --index
- [x] apply 前备份快照与非破坏性查看

### Phase 3: 方案型 quest
- [x] design 型 quest 基础流程（方案摘要随 user review 通过落盘到 `design.json`）
- [x] 基于方案一键发起执行
- [x] 内置 Dashboard 方案类型 UI
- [x] 发起执行入口
- [x] design 专属 prompt/产物结构化增强
- [x] design 文档 schema 增强为 `gloop.design.v1`

### Phase 4: Automation + Inbox
- [x] Automation 启用/禁用控制面
- [x] Automation 配置编辑 API / CLI
- [x] Automation 调度随 server 启停，支持 hourly / daily / weekly alias
- [x] Automation 调度可观测性增强
- [x] 开箱模板（official/template 元数据 + 默认关闭 + CLI/API 列表）
- [x] Triage Inbox（列表、编辑、接受、拒绝）
- [x] 官方委托（official quest 标记 + 列表过滤）
- [x] 内置前端 Inbox 页面（列表、编辑、接受、关闭）

### Phase 5: 优化
- [x] 经验等级 & 自动派单（胜率优先、等级次之；支持显式 warrior/mage）
- [x] 职业等级称呼
- [x] 冒险者激活流程
- [x] 审计自动化（已完成 quest 二次 review）
- [x] 性能和稳定性优化（scheduler 生命周期、用户终审超时阻塞）
- [x] blocked 恢复流（continue / user-review / cancel，支持 quest 级追加预算）
- [x] workspace retention cleanup（CLI/API/scheduler）
- [x] Mage 默认只读工具过滤

### Phase 6: 用户上下文（User Context）
- [x] 全局上下文存储层（`~/.gloop/context/`，维度化 Markdown + meta.json）
- [x] 上下文 API：list / show / summary / refresh（REST + CLI + 平台工具 ABI）
- [x] 内置 `gloop-user-context` skill（渐进式披露 + 触发示例 + 纪律约束）
- [x] `auto_context_refresh` 自动化模板（每日定时刷新，默认关闭）
- [x] 权限分离：两职业可读，仅剑士可写
- [x] 维度：workspace / lark_im / lark_doc / lark_calendar / gloop_history

## 14. 配置项

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| default_working_dir | gloop 启动目录 | 默认工作目录 |
| workspace_retention_days | 7 | 工作区保留天数 |
| default_warrior_id | (自动选) | 默认剑士 |
| default_mage_id | (自动选) | 默认法师 |

**预算与停止条件：**

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| max_turns_per_session | 10 | 单 session 最大 turn 数 |
| max_rework_per_quest | 3 | 最大返工次数 |
| max_duration_minutes | 180 | 单 quest 总耗时上限（分钟） |
| review_hint_retries | 2 | 法师忘调 review 时补 hint 次数 |
| max_consecutive_agent_errors | 2 | executor 连续错误达到阈值后阻塞阶段 |
| user_confirm_timeout_hours | 72 | 等待用户确认超时（小时） |

**安全与隔离：**

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| default_workspace_mode | auto | auto / worktree / copy / readonly |
| apply_require_clean_working_tree | true | apply 时要求主工作区干净 |
| apply_backup_before_apply | true | apply 前自动备份 |
| auto_cleanup_failed_quests | false | 失败 quest 是否自动清理工作区 |
| mage_command_allowlist | go-test / npm-test / npm-run-test | 法师可通过平台执行的验证命令 ID 白名单；Settings 可推荐项目本地测试/构建/lint/typecheck 命令，但必须人工确认；命令可声明 `side_effect_level=L0/L1/L2`，L2 默认拒绝，需 automation `allow_l2` 授权且不能 `auto_apply`；`bytedcli` 等真实 E2E 命令必须显式配置，不默认开放 |
