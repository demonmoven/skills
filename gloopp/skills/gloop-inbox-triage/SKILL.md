---
name: gloop-inbox-triage
version: 1.0.0
description: "收件箱智能分类：v0.4 下明确区分 Human Exceptions 与 Automation Candidates。Human Exceptions 是 blocked / waiting_input / user_review / apply_failed 等需要用户介入的异常；Automation Candidates 是 automation 创建但需要确认的候选委托。典型触发：定时巡检 inbox / human-exceptions / blocked 状态、自动化批量产出后汇总、用户主动询问待办。不负责：自动审批或修改委托状态（必须由用户手动操作）、直接执行委托内容、替代 gloop-user-notification 的通知写作规范。"
metadata:
  class: both
  category: utility
  kind: capability
  related_skills:
    - name: gloop-user-notification
      type: depends_on
      description: 通知的写作规范、参数说明、失败处理
    - name: gloop-self-awareness
      type: related
      description: 使用 quest list / show 获取委托状态与详情
    - name: gloop-note-keeping
      type: related
      description: 发现重要趋势时可用 note add 记录
  requires:
    bins: ["gloop"]
    cliHelp: "gloop quest --help"
---

# gloop-inbox-triage

扫描并分类 Gloop 委托，输出结构化的待办清单和处理建议。v0.4 UI 与通知统一使用两条 lane：Human Exceptions / Automation Candidates。

**v0.4 核心原则（重要）：**
- **Human Exceptions 放需人介入的异常**：`blocked` / `waiting_input` / `user_review` / `apply_failed`。这些是人必须动手的。
- **Automation Candidates 单独成 lane**：`status=pending` 且 `created_by=automation:*` 的委托是候选，不要混入 Human Exceptions。
- **自主闭环的委托进影响日志**：`success` / `applied` 的委托已由 checker 放行自动闭环，人不再审批内容，只在"影响日志"分区感知它们做了什么。
- **只分类、建议、通知，绝不自动 accept / review / apply 任何委托。**

## Trigger Examples

| 类型 | 场景 | 说明 |
|---|---|---|
| 适用 | 定时巡检 | 每天固定时间扫描待办，发飞书摘要 |
| 适用 | 自动化批量产出后 | 多个 automation 创建了 inbox 项，需要汇总 |
| 适用 | 用户主动询问待办 | 用户问"今天有什么要处理的" |
| 适用 | 发现有 blocked 委托 | 注意到某个任务阻塞了，联动扫描全局 |
| 不适用 | 单次任务完成后顺便扫 | 不要每次 quest 结束都扫一遍，浪费 token |
| 不适用 | 自动批准委托 | 绝对不能自动 accept / apply，必须用户手动 |
| 不适用 | 刷屏通知 | 同一 automation 运行周期内最多发 1 条通知 |

## CLI Contract

本 skill 覆盖收件箱分类相关的 gloop CLI 命令，用于获取委托列表、查看详情和发送通知。

### 命令一览

| 命令 | 用途 |
|---|---|
| `gloop quest list --json` | 获取所有委托列表（JSON 格式） |
| `gloop quest show <qid> --json` | 查看单个委托详情（JSON 格式） |
| `gloop notify` | 发送飞书通知（完整规范见 gloop-user-notification skill） |

### 详细说明

#### gloop quest list

获取所有委托的列表，用于批量扫描和分类。

**用法：**
```bash
gloop quest list --json
```

**参数：**

| 参数 | 必填 | 类型 | 说明 |
|---|---|---|---|
| `--json` | 否 | flag | 以 JSON 格式输出，便于自动化解析 |

**返回：**

委托列表的 JSON 数组，每条包含委托基本信息。

**注意：**
- 也可以不传 `--json` 使用默认输出格式，但不便于解析
- 扫描要高效：先 list 再按需 show，不要每个 quest 都完整读一遍事件流，节省 token

#### gloop quest show

查看单个委托的详细信息。

**用法：**
```bash
gloop quest show <qid> --json
```

**参数：**

| 参数 | 必填 | 类型 | 说明 |
|---|---|---|---|
| `<qid>` | 否 | string | 委托 ID；省略时从 `$GLOOP_QUEST_ID` 环境变量读取当前 quest |
| `--json` | 否 | flag | 以 JSON 格式输出 |

**返回：**

单个委托的详细信息（JSON 格式）。

**注意：**
- 不传 `<qid>` 时默认获取当前 quest 的上下文

#### gloop notify

发送飞书通知，通知的写作规范、参数说明、失败处理遵循 `gloop-user-notification` skill。

**用法：**
```bash
gloop notify --title "<标题>" --body "<markdown 正文>" --priority <normal|high>
```

**参数：**

| 参数 | 必填 | 类型 | 说明 |
|---|---|---|---|
| `--title` | 是 | string | 通知标题 |
| `--body` | 是 | string | 通知正文，支持 Markdown |
| `--priority` | 否 | string | 通知优先级，默认 `normal` |

**返回：**

无（纯副作用命令）

**注意：**
- 完整规范见 `gloop-user-notification` skill
- 同一 automation 运行周期内最多发 1 条飞书通知
- 没有待办（0 项）时不发通知
- 通知正文里不要包含 token、密钥、隐私数据等敏感信息

### 扫描分类与输出格式

#### 扫描范围（按优先级从高到低）

#### Human Exceptions（需人介入）

| 类别 | 状态 | 优先级 | 说明 |
|---|---|---|---|
| Blocked | `status=blocked` | P0 | 需要用户介入解阻塞 |
| Waiting Input | `status=waiting_input` | P0 | agent 在等人补充信息 |
| User Review（超时） | `status=user_review` 且停留 > 24h | P0 | 长时间未终审 |
| User Review（普通） | `status=user_review` 且停留 ≤ 24h | P1 | 等待用户终审 |
| Inbox（超时） | `status=pending` + source=automation 且停留 > 12h | P1 | inbox 里被遗忘的项 |
| Inbox（新） | `status=pending` + source=automation 且 ≤ 12h | P2 | 新的待确认项 |
| Reviewing 超时 | `status=reviewing` 且停留 > 4h | P2 | 可能卡住了需要关注 |

#### Automation Candidates（待确认自动化委托）

| 类别 | 状态 | 说明 |
|---|---|---|
| Automation Candidate | `status=pending` + `created_by=automation:*` + `triage_mode=candidate` | 自动化发现了候选工作，等待用户确认是否启动 |

#### 影响日志区（自主闭环，仅供感知，不进待办）

| 类别 | 状态 | 说明 |
|---|---|---|
| 已自动完成 | `status in (success, applied)` 且近 24h | checker 放行后自主闭环，人只感知影响不审批 |
| Goal 迭代中 | 有 `goal.*` 事件 | /goal 模式仍在迭代，记一条进展即可 |

**注意**：`success` / `applied` 的委托**不进待办区**，不要给它们标 P0/P1/P2，也不要建议人去 review。它们属于"已发生的影响"，人看一眼即可。

#### 输出格式模板

```
📋 待处理委托：共 N 项（P0: x，P1: y，P2: z）

## P0 · 阻塞中 / 紧急
- **q_abc123** · [execute] 实现用户登录功能
  - 状态：blocked · 已阻塞 3h
  - 阻塞原因：缺少数据库访问权限
  - 建议：resolve-blocked

## P1 · 待处理
- ...

## P2 · 关注
- ...

## 影响日志（近 24h 自主闭环，仅供感知）
- ✅ q_def456 · 修复登录页样式 · 改动 3 文件
- ✅ q_ghi789 · 升级依赖版本 · 改动 1 文件
- 🔁 q_jkl012 · goal 迭代 2/10 · 目标未达成，继续

## 整体建议
1. 先处理 2 个 blocked 委托，再看 user_review
2. 影响日志里 3 项已自动完成，无需操作
```

如果是 0 项待办，待办区输出 `✅ 当前没有待处理委托`；影响日志区仍可单独汇总（若近 24h 有自主闭环的委托）。**没有待办时不发通知**（除非影响日志区有需要人感知的异常，如 goal 迭代达到上限未达成）。

### 通用约定

- CLI 是本地 agent 调用 gloop 平台能力的主路径
- 平台注入 `GLOOP_*` 环境变量，通常不需要手填 qid/sid
- **只读分析**：只读取 quest 信息，不修改任何数据，不执行外部副作用命令
- **绝不自动操作**：不能 accept / reject / review / apply 任何委托，状态变更必须用户手动
- **不刷屏**：同一 automation 运行周期内最多 1 条飞书通知；没有待办时不发通知
- 依赖 `gloop-user-notification` skill：通知的写作规范、参数说明、失败处理都遵循该 skill

## Discipline

- **绝不自动操作**：不能 accept / reject / review / apply 任何委托，状态变更必须用户手动。
- **只读分析**：只读取 quest 信息，不修改任何数据，不执行外部副作用命令。
- **不替代用户判断**：建议是建议，决定权在用户；不要用"我已经帮你处理了"这种语气。
- **自主闭环不进待办**：`success` / `applied` 的委托已由 checker 放行，人不再审批内容。只在"影响日志"区汇总让人感知，不要建议人去 review 它们。
- **不刷屏**：同一 automation 运行周期内最多 1 条飞书通知；没有待办时不发通知。
- **不泄露敏感信息**：通知正文里不要包含 token、密钥、隐私数据等。
- **依赖 `gloop-user-notification` skill**：通知的写作规范、参数说明、失败处理都遵循该 skill，不要自己重写一套。
- **扫描要高效**：先 list 再按需 show，不要每个 quest 都完整读一遍事件流，节省 token。

## 写作建议

### 好的通知标题

- 「待处理委托：3 项等你处理（1 个阻塞）」
- 「收件箱更新：2 个新委托待确认」
- 「⚠️ 有 1 个委托阻塞超过 4 小时」

### 好的通知正文

简要说明：
1. 总数和各优先级数量
2. 前 3-5 条最重要的（优先 P0，再 P1）
3. 每条一行：`[P0] q_abc · 一句话摘要`
4. 末尾引导：`完整清单见 Gloop Dashboard`

示例：

```
共 5 项待处理（P0: 1，P1: 2，P2: 2）

[P0] q_a1b2 · 用户登录功能阻塞
[P1] q_c3d4 · 代码评审待终审
[P1] q_e5f6 · 依赖升级待确认

完整清单见 Gloop Dashboard → Inbox
```

## 相关 Skills

- **gloop-user-notification**：通知的写作规范、参数说明、失败处理
- **gloop-self-awareness**：`gloop quest show` 使用方法、状态查询
- **gloop-note-keeping**：如果发现重要趋势，可以用 `gloop note add` 记下来
