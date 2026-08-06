# 用户上下文：承诺信号维度 Spec v0.3

> 版本：v0.3（v0.1 已废弃，v0.2 → v0.3 为 schema 与执行细节修正）
> 状态：Draft
> 日期：2026-06-22
> 负责人：lihuanyu.0w0
> 前置依赖：`auto_context_refresh` 自动化（已落地）、`dim_lark_im` 维度（已落地）
> 关联讨论：飞书话题群 #gloop-design
> 历史决策参考：`dim_lark_im.md` 中 "Context 维度只存环境信息，不存进展、结论、TODO"（本 spec 遵守该约束，不突破）

---

## 一、背景与动机

### 1.1 现状

Gloop 用户上下文目前有 8 个维度，设计原则是：

- 存**背景知识**，不存**进展流水账**
- 存**稳定模式**，不存**瞬时状态**
- 存**可复用偏好**，不存**单次任务细节**

`dim_lark_im.md` 也明确写着：
> Context 维度只存环境信息，不存进展、结论、TODO。

### 1.2 问题

当前上下文缺少一类**高价值信号**——用户对外承诺的存在性线索。

典型场景：
- 飞书任务里有若干未完成任务，最近 deadline 在几天后
- 群聊里有几句听起来像承诺的话，但需要确认算不算数
- 日历上近期有重要会议，用户可能有相应的交付物

这些信息有三个特点：
1. **有行动价值**：agent 知道后可以辅助 Inbox/Quest 层判断任务排布、建议创建 Quest、提醒用户
2. **是信号不是事实**：聊天里的"我看看"不一定是承诺，飞书任务可能已经转手了，需要二次核验
3. **数量少、变化快**：活跃信号通常不超过 5-8 条，但状态随时在变

现在这些信号散落在各处，agent 每次都要重新检索，效率低且容易漏掉。

### 1.3 核心约束（不能突破）

> **Context 层 = 环境事实 + 检索入口。状态流转、完成/延期、优先级，属于 Inbox/Quest/外部 Task adapter，不靠 Markdown 表格维护。**

因此，本 spec **不建任务系统**、**不做状态机**、**不存事务型数据**。Context 里只有信号（signals）和策略（policy）。

---

## 二、三层模型

为了避免 Context 滑向半吊子任务系统，承诺相关的能力拆成三层，各司其职：

```
┌─────────────────────────────────────────────────────┐
│  Layer 3: Quest 层                                   │
│  （已存在）用户确认后转 Quest，有完整生命周期        │
│  Quest 完成可反向更新 candidate 状态                 │
├─────────────────────────────────────────────────────┤
│  Layer 2: Runtime / Inbox 层                        │
│  ObligationCandidate：有 ID、source、freshness、    │
│  confirmation_status；可过期、可 dismiss、可转 Quest │
│  （本 spec 只定义接口，不实现，留给未来 Inbox 扩建） │
├─────────────────────────────────────────────────────┤
│  Layer 1: Context 层  ← 本 spec 的范围              │
│  commitment_signals 维度：只存信号、策略、检索入口  │
│  不维护状态、不做事务、不保证准确性                   │
└─────────────────────────────────────────────────────┘
```

### Layer 1：Context 层（本 spec 范围）

**定位**：信号层。告诉 agent "这里可能有承诺，你可以去查查"。

**内容**：
- 承诺识别策略（什么样的表达算/不算承诺）
- 候选承诺信号列表（action 摘要、时间暗示、置信度、为什么不确定、核验入口）
- 检索入口：去哪里查更多信息

**不做**：
- 不维护"完成/未完成/延期"状态
- 不设优先级（Context 不排序、不打标优先级，只给 raw signal）
- 不做增删改 API（只有 automation 能写，agent 不能改）
- 不保证准确性

### Layer 2：Runtime / Inbox 层（远期规划，本 spec 不实现）

**定位**：候选对象层。经过初步筛选、有明确来源和新鲜度的承诺候选。

**特征**：
- 有唯一 ID
- 有 confirmation_status（`unconfirmed` / `user_confirmed` / `dismissed` / `converted_to_quest`）
- 有 freshness / ttl，过期自动失效
- 可以 dismiss、可以转 Quest
- 存储在 runtime state（如 inbox 目录），不在 context 目录

**与 Context 层的关系**：Context 层的 signals 是 Inbox 层 candidates 的输入源之一。Context 负责"发现"，Inbox 负责"管理"。

### Layer 3：Quest 层（已存在）

**定位**：执行层。用户确认要做、且适合 agent 做的事，转成 Quest。

**与 Layer 2 的关系**：ObligationCandidate → 用户确认 → 创建 Quest → Quest 完成 → 反向标记 candidate 已完成。

---

## 三、Context 层设计

### 3.1 维度定义

**维度名**：`commitment_signals`
**文件**：`dim_commitment_signals.md`（Phase 2 才独立成文件，Phase 0-1 寄生在 `lark_im`）
**定位**：用户对外承诺的**候选信号**和**识别策略**。仅供 agent 参考，不作为行动事实。

**核心原则**：
- 所有条目都是**信号**，不是**事实**
- 强调**不确定性**：每条都要说明"为什么可能是承诺"和"为什么不一定是"
- 提供**核验路径**：告诉 agent 去哪里确认
- 数量克制：活跃信号不超过 5 条，宁少勿滥

### 3.2 维度结构

**严格使用标准 8 个二级标题**（与其他维度一致，英文标题作为 schema anchor），不新增同级标题。

候选信号表放在 **Current Focus** 下作为三级标题（`### Candidate Signals`），不升格为二级 schema anchor。

```markdown
# Commitment Signals

## Scope
- 数据来源：飞书任务（高置信）、飞书聊天记录（中置信）、飞书日历（低置信，仅作时间压力参考）
- 刷新频率：每天 2 次（9:00 / 15:00）
- 定位：候选承诺信号，不是已确认的待办清单；使用前请核验
- 隐私处理：不存原始聊天、不存消息链接、不存对方真实身份、不存精确 ID

## Current Focus
（2-3 句总览：当前候选承诺数量级、最近的时间压力、主要集中在哪些领域）

### Candidate Signals
（候选承诺信号列表，最多 5 条；展示顺序仅用于可读性：时间暗示更近、置信度更高的信号靠前；不得写入 priority 字段）
| # | Action 摘要 | 时间暗示 | 来源类型 | 置信度 | 不确定原因 | 核验关键词 |
|---|---|---|---|---|---|---|
| 1 | 输出某项目 context signals 维度设计文档 | 约 2026-06-22 | group_chat | 中 | 群聊中"我来写一版"是口语表述，未确认 deadline 刚性 | 关键词：signals spec；时间窗：近 7 天 |
| 2 | 审核 v2 方案相关交付物 | 2026-06-24（明确截止） | lark_task | 高 | 任务可能已转手或延期，需核验当前状态 | 关键词：v2方案；任务列表：+get-my-tasks |
| 3 | 与合作方对齐接口定义 | 约下周初 | p2p_chat | 低 | 仅提到"下周聊聊"，不确定是否有交付物承诺 | 关键词：接口对齐；需用户确认 |

> 说明：置信度分高/中/低三档。高=飞书任务或明确 @ 分配并确认；中=群聊中有明确动作和时间；低=模糊提及或仅日历线索。
> 来源类型枚举：lark_task / group_chat / p2p_chat / calendar_hint。

## Stable Preferences
（用户对承诺的态度和处理习惯，从历史模式提炼；也作为识别策略/Commitment Policy）
- 例："非正式群聊中的口头答应不算数，需要用户明确确认才算承诺"
- 例："'我看看''我 follow 一下'通常只是表态关注，不是交付承诺"
- 例："用户习惯在 deadline 前 1 天集中交付"
- 例："周五下午不接新承诺"
- 例："高置信度承诺用户会主动同步进展，没消息就是在推进中"

## Project Contracts
（承诺相关的工具、命令、检索入口；以实测 lark-cli --help 为准）
- 飞书任务列表：`lark-cli task +get-my-tasks --as user --page-all --complete=false`
- 飞书任务搜索：`lark-cli task +search --as user --query <关键词>`（assignee 需传 open_id，不支持 "me"）
- 聊天全文搜索：`lark-cli im +messages-search --as user --query <关键词>`
- 日历日程查询：`lark-cli calendar +agenda --as user --start <date> --end <date>`

## Recent Decisions
（近期明确的承诺处理决策和误判案例，用于更新识别策略；不存履约评价）
- 格式：「事项 - 处理结果（确认/驳回/误判）- 确认来源」
- 只存模式参考，不存"按时/延期"这种评价性历史
- 例："'我来 follow 一下'被判为误判，用户说只是关注不是交付承诺 - 误判案例 - 用户明确纠正"

## Anti-Patterns
（已被验证为误判的"伪承诺"模式）
- "'我看看''我跟进一下'通常不是交付承诺，只是表态关注"
- "会议邀请不等于交付承诺，除非会议前有明确的产出物约定"
- "用户创建的飞书任务（自己是创建人不是负责人）不算承诺"
- "群里 @ 某人但对方未明确回复接受，不算承诺"

## Retrieval Index
- 飞书任务列表入口：`lark-cli task +get-my-tasks --as user --complete=false`
- 飞书任务搜索：`lark-cli task +search --as user --completed=false --due <start>,<end>`
- 聊天承诺关键词：承诺、约定、deadline、截止、我来、交给我、周三前、下周、交付、输出
- 相关维度：`lark_im`（聊天上下文）、`lark_calendar`（时间分布）、`activity_snapshot`（内部任务负载）

## Gaps
- 私聊中的承诺可能因权限原因无法捕获
- 飞书任务的负责人变更可能有延迟
- 所有信号都需要二次核验，不能直接当作事实使用
- 当前缺少用户手动确认/驳回信号的机制（待 Layer 2 建设）
- 聊天消息搜索范围和权限受限，可能有漏检
```

### 3.3 数据来源与置信度

| 来源 | 来源类型值 | 置信度 | 说明 | 核验方式 |
|---|---|---|---|---|
| 飞书任务（我是负责人 + 未完成 + 有 deadline） | `lark_task` | 高 | 正式系统分配，有明确状态 | `+get-my-tasks` 或 `+search` |
| 飞书任务（我是负责人 + 无 deadline） | `lark_task` | 中高 | 有正式分配，但时间不明确 | 同上 |
| 群聊中 @ 用户分配 + 用户明确接受 | `group_chat` | 中 | 有公开确认，但非正式系统 | 消息搜索回溯上下文 |
| 群聊中用户主动说"我来做 X"+ 有时间 | `group_chat` | 中 | 有动作有时间，但可能是随口一说 | 消息搜索回溯上下文 |
| 私聊约定 | `p2p_chat` | 中低 | 场景不明确，容易误判 | 需更多上下文或用户确认 |
| 日历事件（有会议但无产出物约定） | `calendar_hint` | 低 | 只能说明有时间压力，不是承诺本身 | 日历 + 聊天交叉验证 |
| 别人在群里说"XX 你做一下"用户没回应 | — | 极低 | 用户没确认，不算数 | 不收录 |

---

## 四、隐私设计

### 4.1 反识别原则

commitment_signals 涉及"对谁承诺"的关系信息，比普通背景信息更敏感。采取以下脱敏措施：

1. **对方身份抽象化**：不存真实姓名、open_id、群名。使用 `counterparty_label` 抽象标签：
   - 格式：`<角色>/<项目或场景>`
   - 例：`方案评审人/项目A`、`接口对齐对象/中台团队`、`需求方/运营线`
   - 同一个对方在同一维度内使用相同标签，保持一致性

2. **来源场景抽象化**：
   - 不存群名，存 `source_label`（如 `项目讨论/接口对齐场景`）
   - 不存 chat_id、task_guid、消息链接
   - 不存原始聊天原文

3. **核验入口只给关键词级**：
   - 核验命令只给关键词和时间窗，不给精确 ID
   - agent 需要精确查询时走检索命令，自行定位
   - 如果未来需要精确回溯能力，原始 ID 存在 runtime state 或敏感存储中，不在 context markdown 里

### 4.2 保留策略

- Candidate Signals 最多保留 5 条
- 超过 30 天无新佐证的信号自动移除
- Recent Decisions 最多保留 5 条、不超过 2 周，且只存决策模式不存评价
- Anti-Patterns 可以长期保留，但必须有明确的误判证据

---

## 五、分阶段实施路径

### Phase -1：Source Probe（前置验证，先做这个）

**目标**：在不动 context schema 的前提下，验证数据源可用性、数据形态、脱敏可行性。

**内容**：
1. 验证 lark user 授权状态和 scope（`lark-cli auth status --json`）
2. 跑通 `lark-cli task +get-my-tasks --as user --complete=false --page-all`，确认返回字段、数据量、脱敏难度
3. 跑通 `lark-cli im +messages-search --as user --query <关键词>`，确认搜索范围、权限、返回格式
4. 确认日历命令：`lark-cli calendar +agenda --as user` 的可用性
5. 产出一份 Source Probe 报告：各数据源的字段、噪音水平、脱敏成本、建议优先级
6. 产出 2-3 条**脱敏后的**样例信号，验证格式合理性

**验收标准**：
- 至少 1 个主数据源（飞书任务）能跑通
- 样例信号符合隐私要求（无真名、无精确 ID、无原始内容）
- 明确知道每个 CLI 命令的准确 flag 和返回结构（不拍脑袋）

**工作量**：~0.5-1 天

### Phase 0：飞书任务信号最小实验

**目标**：用最高置信、最低噪音的数据源（飞书任务），验证 commitment_signals 的产品价值。

**做法**：
- 不新增独立维度
- 修改 `auto_context_refresh` 的 prompt，在 `dim_lark_im` 维度的 **Current Focus** 下新增三级标题 `### Candidate Commitment Signals (Experimental)`
- 这是复用现有 context refresh 载体的临时挂载；语义上来源是 Lark Task，不归属 IM 主题。Phase 2 独立维度后必须从 `lark_im` 移除。
- 只从飞书任务提取，**不做聊天信号**
- 最多 3 条，宁缺毋滥
- 字段：action 摘要、时间暗示、来源类型、置信度、不确定原因、核验关键词
- 明确声明："以下为候选承诺信号，仅用于二次核验，不作为行动事实。"
- 所有样例严格脱敏

**验收标准**：
- 连续跑 3-5 天后 review
- **看 precision，不看数量**：高置信信号准确率 ≥ 95%（飞书任务本身是事实系统，应该接近 100%）
- 至少有 1-2 条信号是"agent 如果知道了确实能帮上忙"的
- 如果飞书任务数据量为 0 或价值不明显，直接砍掉，不进入下一阶段

**工作量**：~1 天（主要是改 prompt + 飞书任务接入 + 脱敏逻辑）

### Phase 1：聊天信号补充（Optional）

**目标**：在飞书任务信号验证通过后，评估聊天信号的补充价值和噪音水平。

**做法**：
- 在 Phase 0 的基础上，增加聊天来源的信号
- 聊天信号置信度统一标为"中"或"低"
- 聊天信号最多 2 条（占总 5 条中的少数派）
- 严格收录标准：必须同时有"明确动作" + "时间暗示" + "用户是责任人"
- 继续寄生在 `lark_im` 的实验段中

**验收标准**：
- 聊天信号的 precision ≥ 70%（低于这个就不值得做）
- 聊天信号补充了飞书任务覆盖不到的场景（比如临时约定、非正式任务）
- 如果噪音太大或价值不高，可以砍掉，不影响主线

**工作量**：~1 天

### Phase 2：独立 `commitment_signals` 维度

**目标**：信号数量和价值都验证通过后，升级为独立维度，建立完整结构。

**做法**：
- 新增 `dim_commitment_signals.md` 独立维度
- 严格使用标准 8 段式结构（Scope / Current Focus / Stable Preferences / Project Contracts / Recent Decisions / Anti-Patterns / Retrieval Index / Gaps）
- Candidate Signals 作为 **Current Focus 下的三级标题**
- 合并各来源信号：飞书任务（高置信主干）+ 聊天（补充）+ 日历（参考）
- 在 `auto_context_refresh` 中增加独立的刷新逻辑
- 刷新频率：每天 2 次（9:00 / 15:00）
- 更新 `gloop-user-context` skill，指导 agent 何时查询该维度、如何核验信号
- `summary.md` 只增加**一句压力摘要**：如"当前有 N 个候选承诺信号，最近 deadline 约 X 天后"，不列具体内容。summary 是路由入口，不是 dashboard

**验收标准**：
- 维度结构完整、格式稳定，schema anchor 与标准一致
- 各来源信号去重正确
- 脱敏规则执行到位
- agent 能通过 `context_show dim=commitment_signals` 获取信息
- 有至少 1 个消费场景参考（如 morning briefing、inbox triage 做信息补充）

**工作量**：~2-3 天

### Phase 3：Inbox / Quest 联动（Layer 2 + Layer 3，远期）

**目标**：从"信号"走向"行动"，建立完整的承诺→行动链路。

**内容**（不全部承诺，按需推进）：
- 建立 `ObligationCandidate` 概念（Inbox 层），有 ID、source、freshness、confirmation_status、ttl
- 支持用户 dismiss / 确认信号
- 支持"信号 → 建议创建 Quest"的用户确认流程
- Quest 完成后反向标记 candidate 状态
- 承诺负载感知：创建新 Quest 时参考用户当前承诺负载

**与 Context 层的关系**：
- Context 层仍然只管信号发现，不管状态管理
- ObligationCandidate 的状态变化不回写 context
- Context 刷新时可以参考已知的 candidate 状态，但不依赖它

---

## 六、与现有系统的边界

### 6.1 和 `lark_im` 维度的关系

- `lark_im` = 广泛的聊天主题 + 决策摘要
- `commitment_signals` = 从多源提炼的**可行动信号子集**
- Phase 0-1 阶段信号寄生在 `lark_im` 中，Phase 2 独立后，`lark_im` 不再维护信号列表
- 两个维度可以互相引用：`lark_im` 的 Retrieval Index 指向 `commitment_signals`，反之亦然

### 6.2 和 `activity_snapshot` 维度的关系

- `activity_snapshot` = Gloop **内部** quest 的运行状态（agent 正在做的事）
- `commitment_signals` = **外部**承诺的候选信号（用户答应别人的事）
- 两者合起来能反映用户的完整工作负载，但各自独立维护
- 未来可以在 `activity_snapshot` 的 Current Focus 中增加一句"外部承诺负载"的摘要引用

### 6.3 和 Inbox Triage 的关系

- `gloop-inbox-triage` 目前只看 Gloop 内部 quest 状态
- Phase 2 之后可以接入 `commitment_signals` 作为**参考信息**（如"用户近期有时间压力，相关 quest 可以排前面"）
- 但 signals 不直接决定优先级，inbox triage 的排序依据仍然是 quest 自身状态；优先级是 Inbox/Quest 层的事，不是 Context 层的事

### 6.4 和 `summary.md` 的关系

- `summary.md` = 路由入口，2-4 段总览
- Phase 2 之后，summary 中最多增加**一句**承诺压力摘要
- **不复制承诺正文**、不列清单、不放 deadline 表格
- agent 想知道详情，需要主动 `context_show dim=commitment_signals`

---

## 七、CLI 命令清单（实测对齐版）

> 以下命令均以 `lark-cli <cmd> --help` 实测为准写入。不确定的标注"待 Phase -1 验证"。

### 飞书任务

| 命令 | 用途 | 备注 |
|---|---|---|
| `lark-cli task +get-my-tasks --as user --complete=false --page-all` | 拉取我的未完成任务 | 已确认存在；`--complete` 是 bool flag，默认不传则返回全部 |
| `lark-cli task +search --as user --query <关键词>` | 按关键词搜索任务 | 已确认存在；`--assignee` 传 open_id，不支持 "me"；`--completed` 是 bool flag |
| `lark-cli task +get-related-tasks --as user --include-complete=false` | 拉取与我相关的任务 | 已确认存在；包含我创建的、我关注的等 |

### 飞书聊天

| 命令 | 用途 | 备注 |
|---|---|---|
| `lark-cli im +messages-search --as user --query <关键词> --page-size N` | 跨聊天搜索消息 | 已确认存在；搜索范围和权限待 Phase -1 验证 |
| `lark-cli im +chat-messages-list --chat-id <id> --as user --start <ISO> --end <ISO>` | 拉取指定聊天的历史消息 | 已确认存在；`--user-id` 可自动解析 P2P chat_id |

> 注意：本节是实现/探测命令清单，不等于可写入 Context 的核验入口。Context markdown 只能保留关键词和时间窗；`chat_id`、`message_id`、`task_guid`、URL 和消息链接只能存在 runtime/sensitive state 或一次性探测输出中。

### 飞书日历

| 命令 | 用途 | 备注 |
|---|---|---|
| `lark-cli calendar +agenda --as user --start <date> --end <date>` | 查询指定时间范围内的日程 | 待 Phase -1 验证 |

---

## 八、修订历史

### 8.1 v0.2 → v0.3 变更说明

v0.2 架构方向已过关，但执行细节有几个坑会在落地时咬人。v0.3 是 schema 与执行细节修正：

1. **Schema 严格对齐标准 8 段式**：Candidate Signals 从二级标题降级为 Current Focus 下的三级标题，不新增 schema anchor；Stable Preferences 恢复原名，Commitment Policy 作为其内容的一部分。
2. **Phase 顺序重排**：新增 Phase -1 Source Probe（先验证数据源）；Phase 0 改为飞书任务信号实验（高置信主干优先）；聊天信号降级为 Phase 1 Optional。
3. **CLI 命令全修正**：所有命令以 `--help` 实测为准写入；修正 `+search --assignee me` 错误（应传 open_id）；修正 `calendar +events` → `calendar +agenda`；增加专门的命令清单章节。
4. **样例全脱敏**：去掉真实姓名、群名、chat_id 占位符；统一用 source_type 枚举 + counterparty_label 抽象标签；核验入口只给关键词不给精确 ID。
5. **Recent Decisions 收敛**：不存"按时/延期"这种评价性历史，只存决策模式和误判案例；Context 是协作辅助，不是绩效档案。
6. **术语清理**：去掉"高优先级承诺"等旧模型残留表述；Context 不设优先级，优先级是 Inbox/Quest 层的事。

### 8.2 v0.1 → v0.2 变更说明

v0.1 版本的方向有价值，但设计上把"上下文环境层"和"任务/承诺状态层"搅在一起了。v0.2 是一次架构级修正，核心变更：

1. **概念重定位**：从 `commitments`（承诺事实）改为 `commitment_signals`（承诺信号），明确"信号不是事实"
2. **三层模型**：Context 层 / Runtime Inbox 层 / Quest 层，各司其职；本 spec 只做 Context 层
3. **数据模型简化**：去掉状态（完成/延期）、去掉优先级字段、去掉事务型字段；只留 action 摘要、时间暗示、置信度、不确定原因、核验入口
4. **数据源优先级反转**：先接飞书任务（高置信主干），再考虑聊天（补充信号）；而不是反过来
5. **Phase 0 定位调整**：从"放到 Recent Decisions 子表格"改为"独立实验段 + 明确标注不确定性 + 看 precision 不看数量"
6. **隐私策略收紧**：对方身份用抽象标签（counterparty_label），不存真名；不存可直接回溯的 ID
7. **summary 边界收紧**：只加一句压力摘要，不列具体承诺内容，保持路由入口定位
8. **技术命令修正**：`lark-cli task list` → `lark-cli task +get-my-tasks` / `+search`

---

## 九、开放问题

1. **聊天消息搜索的实际覆盖范围**：当前 user token 的 `+messages-search` 能搜到哪些聊天？私聊能不能搜到？搜索时间范围有多大？这直接决定 Phase 1 聊天信号有没有必要做。（Phase -1 会验证）
2. **飞书任务的数据量**：用户手上平均有多少个未完成任务？如果只有 0-2 个，那信号的价值就很有限。（Phase -1 会验证）
3. **命名**：`commitment_signals` vs `obligation_signals` vs `pending_commitment_signals`？目前倾向 `commitment_signals`。
4. **刷新频率**：每天 2 次够不够？飞书任务状态变化可能很快，但 context 刷新成本也不低。
5. **信号过期机制**：Context 是覆盖写入的，每次刷新自然会淘汰旧信号。要不要额外显式的 ttl 机制？

---

## 十、参考

- 用户上下文整体设计：`PRDv2.md` 第 2.2.7 节
- 现有维度实现：`internal/fsstore/context.go`
- 上下文刷新 automation：`internal/fsstore/automation.go`（`defaultContextRefreshQuery`）
- 飞书 IM 维度样例：`.gloop-dev/context/dim_lark_im.md`
- 收件箱分类 skill：`skills/gloop-inbox-triage/SKILL.md`
- 飞书任务 CLI：`lark-cli task +get-my-tasks --help`、`lark-cli task +search --help`
- Source Probe 报告：`docs/context-commitment-signals-source-probe-2026-06-22.md`
- v0.1 废弃版 spec：git 历史（本文件 v0.1 版本）
- v0.2 版 spec：git 历史（本文件 v0.2 版本）
