---
name: gloop-user-context
version: 1.0.0
description: "用户工作上下文查询：从本地工作区、飞书群聊/文档/日程等多源提炼的全局背景知识。典型触发：接到新任务想先了解用户最近在做什么、需要知道用户的技术偏好或团队约定、不确定某个项目的背景信息。不负责：当前 quest 的阶段状态或事件流（走 gloop-self-awareness）、当前 quest 的笔记（走 gloop-note-keeping）。"
metadata:
  class: both
  category: info
  kind: capability
  related_skills:
    - name: gloop-self-awareness
      type: related
      description: 都是上下文/信息类，但 user-context 是跨 quest 全局背景
    - name: gloop-note-keeping
      type: related
      description: 都是持久化信息，但 context 是跨 quest 全局刷新
    - name: gloop-code-exploration
      type: related
      description: codebase_map 维度是代码探索的 L2 层核心输入
  requires:
    bins: ["gloop"]
    cliHelp: "gloop context --help"
---

# gloop-user-context

用户全局工作上下文的 CLI 协议。上下文是跨 quest 持久化的背景知识，
由 `auto_context_refresh` 自动化定期刷新。

v0.4 边界：全局 Context 是环境信息层，不是当前任务状态机。当前循环进展、最近活动和待处理事项属于 Activity Snapshot；单个委托的可恢复执行记忆属于 Loop State Spine。需要当前 quest 的状态、事件、Human Exceptions 或 Loop State Spine 时，走 gloop-self-awareness / quest detail，不要把全局 context 当执行源。

## Trigger Examples

| 类型 | 场景 | 说明 |
|---|---|---|
| 适用 | 新任务开始，想先了解用户最近在忙什么 | 上下文能给出全局背景 |
| 适用 | 不确定用户的技术栈或团队约定 | 上下文里可能有相关信息 |
| 适用 | 需要关联飞书里讨论过的决策 | 上下文摘要了近期群聊主题 |
| 不适用 | 查询当前 quest 的状态或事件流 | 走 gloop-self-awareness |
| 不适用 | 给当前 quest 记笔记 | 走 gloop-note-keeping |
| 不适用 | 上下文没有相关维度时 | 不要强行调用，直接按已有信息工作 |

## 使用方式：渐进式披露

1. 先 `gloop context list` 看有哪些维度可用
2. 挑相关的维度 `gloop context show <name>` 读详情
3. 只在确实需要时加载，不要全量灌进上下文

## 内容结构

上下文维度应优先按以下 Markdown 段落组织。旧数据可能缺段，读取时按已有内容处理。
章节标题使用英文作为稳定 schema anchor；正文默认中文，技术名词、命令、文件路径、API、状态枚举和产品名可保留英文。
每日刷新是覆盖更新，不是无限追加日志。短期事实按时间窗口替换；稳定偏好、项目契约、反模式和检索线索可以跨天保留，但要在刷新时按新证据修正或删除过时项。

| 段落 | 用途 |
|---|---|
| Scope | 数据来源范围、时间新鲜度和覆盖对象 |
| Current Focus | 用户近期正在推进的项目、战役、风险或阻塞点 |
| Stable Preferences | 长期稳定的用户偏好、技术判断原则、沟通风格和验收口径 |
| Project Contracts | 项目/仓库/服务事实、边界、常用验证命令、API/前后端契约 |
| Recent Decisions | 近期明确拍板的结论，尽量带来源类型、新鲜度和置信度 |
| Anti-Patterns | 已被用户纠正或明确不希望重复发生的行为 |
| Retrieval Index | 需要细查时应该去哪里查的可行动线索 |
| Gaps | 无法确认、可能过时、存在冲突或需要二次确认的信息 |

Activity Snapshot 与 Loop State Spine 的区别：
- Activity Snapshot：全局近期活动摘要，回答“最近发生了什么、还有什么在跑”。
- Loop State Spine：单个 quest 的可恢复执行记忆，回答“当前 phase 卡在哪里、下一步期望动作是什么”。
- 全局 Context 只保存环境事实和检索线索；progress/TODO 不写入普通 context 维度。

## CLI Contract

用户全局工作上下文的 CLI 协议，支持列出维度、读取维度详情和获取总览摘要。上下文是跨 quest 持久化的背景知识，由 `auto_context_refresh` 自动化定期刷新。

### 命令一览

| 命令 | 用途 |
|---|---|
| `gloop context list` | 列出所有可用的上下文维度（元信息，不含正文） |
| `gloop context show` | 读取某个维度的详细摘要 |
| `gloop context summary` | 读取总览摘要（一句话版本） |

### 详细说明

#### gloop context list

列出所有可用的上下文维度元信息，不含正文内容。

**用法：**
```bash
gloop context list
```

**参数：**

无参数

**返回：**

```json
{
  "ok": true,
  "items": [
    {
      "name": "workspace",
      "title": "本地工作区",
      "description": "从本地 Git 仓库和项目文件提炼的项目上下文",
      "updated_at_ms": 1718000000000,
      "size_bytes": 2048
    }
  ],
  "total": 2
}
```

#### gloop context show

读取指定维度的详细摘要内容。

**用法：**
```bash
gloop context show <dim_name>
```

**参数：**

| 参数 | 必填 | 类型 | 说明 |
|---|---|---|---|
| `<dim_name>` | 是 | string | 维度名称，对应 `context list` 中的 `name` 字段 |

**返回：**

```json
{
  "ok": true,
  "dim": {
    "name": "workspace",
    "title": "本地工作区",
    "body": "# 本地工作区摘要\n\n..."
  }
}
```

#### gloop context summary

读取所有上下文的一句话总览摘要。

**用法：**
```bash
gloop context summary
```

**参数：**

无参数

**返回：**

```json
{
  "ok": true,
  "summary": "用户近期主要在做 XXX 项目，使用 Go + React 技术栈..."
}
```

### 通用约定

- CLI 是本地 agent 调用 gloop 平台能力的主路径
- 平台注入 `GLOOP_*` 环境变量，通常不需要手填 qid/sid

## 注意事项

- 上下文是**摘要**，不是原始数据。需要更细的信息请自行去对应源里查
- 上下文有刷新周期，不一定包含最新的实时信息
- 如果发现上下文维度缺失或过时，可以建议用户运行 `gloop automation run auto_context_refresh` 刷新
- 不要在每一轮都重新加载上下文，只在确实需要相关背景时调用

## Discipline

1. **渐进式加载**：先 list 看有哪些维度，再按需 show 详情，禁止一次性加载所有维度
2. **仅供参考**：上下文是摘要不是事实，关键信息需要二次确认
3. **不修改全局上下文**：普通任务只能读取，不能调用 `gloop context write-*` 命令
4. **隐私意识**：上下文中可能包含敏感信息，不要在输出中原样复述给无关人员
5. **时效性**：注意上下文的更新时间，过时信息需要谨慎使用
6. **分层使用**：Stable Preferences 可作为默认工作方式；Current Focus / Recent Decisions 需要结合更新时间判断；Gaps 里的内容不能当事实使用
