---
name: gloop-self-awareness
version: 1.0.0
description: "任务上下文自查：剑士或法师在不确定当前状态、返工次数、工作目录位置、历史事件流时查询全局真相。典型触发：刚接到一个任务需要先摸清情况、返工到第 N 轮忘了之前发生了什么、不知道代码在哪个目录要去读文件、想看事件流确认阶段时间点。"
metadata:
  class: both
  category: info
  kind: capability
  related_skills:
    - name: gloop-quest-execution
      type: see_also
      description: 要提交阶段结论走 execution，self-awareness 只负责读
    - name: gloop-quest-review
      type: see_also
      description: 要提交评审结论走 review，self-awareness 只负责读
    - name: gloop-note-keeping
      type: related
      description: 两者都是 quest 内信息查询，但 note 是可写的事件流
    - name: gloop-user-context
      type: related
      description: 前者是单 quest 状态，后者是跨 quest 全局背景
  requires:
    bins: ["gloop"]
    cliHelp: "gloop quest --help"
---

# gloop-self-awareness

上下文查询工具的 CLI 协议与返回字段契约。

## Trigger Examples

| 类型 | 场景 | 说明 |
|---|---|---|
| 适用 | 需要 quest 元信息 | quest_info 协议字段可获取 |
| 适用 | 需要事件流历史 | quest_history 可获取 |
| 适用 | 需要当前阶段信息 | phase_info 可获取 |
| 不适用 | 要提交阶段结论 | 走 gloop-quest-execution / gloop-quest-review 的阶段工具 |

## CLI Contract

上下文查询工具的 CLI 协议与返回字段契约，支持查询 quest 元信息、事件流历史和当前阶段信息。

### 命令一览

| 命令 | 用途 |
|---|---|
| `gloop quest info` | 查询 quest 元信息（返回结构化 JSON） |
| `gloop quest history` | 查询事件流历史 |
| `gloop phase info` | 查询当前阶段信息 |

### 详细说明

#### gloop quest info

查询当前 quest 的元信息，返回结构化 JSON。

**用法：**
```bash
gloop quest info
```

**参数：**

无参数

**返回：**

返回结构化 JSON，稳定字段包括：
`id`, `short_id`, `status`, `type`, `query`, `rework_count`, `max_rework`,
`workspace_mode`, `workspace_path`, `created_at_ms`, `started_at_ms`,
`warrior_id`, `mage_id`, `created_by`, `review_hints`

关键字段语义：

| 字段 | 类型 | 说明 |
|---|---|---|
| `rework_count` | int | 当前已返工次数，从 0 开始 |
| `max_rework` | int | 最大返工次数，达到上限后 request_changes 转用户终审 |
| `review_hints` | string | 上一轮评审 hints；仅当 `rework_count > 0` 时非空 |
| `workspace_path` | string | 当前 quest 的工作目录绝对路径 |
| `status` | string | quest 生命周期状态枚举（pending/running/reviewing/user_review/...） |

#### gloop quest history

查询当前 quest 的事件流历史。

**用法：**
```bash
gloop quest history --limit <条数>
```

**参数：**

| 参数 | 必填 | 类型 | 说明 |
|---|---|---|---|
| `--limit` | 是 | int | 返回的事件条数 |

**返回：**

返回事件流历史列表。

#### gloop phase info

查询当前阶段的信息。

**用法：**
```bash
gloop phase info
```

**参数：**

无参数

**返回：**

返回当前阶段的信息。

### 通用约定

- CLI 是本地 agent 调用 gloop 平台能力的主路径
- 平台注入 `GLOOP_*` 环境变量，通常不需要手填 qid/sid

## Discipline

- 所有查询都是**只读**的，不改变 quest 状态，不触发任何副作用。
- 返回字段平台不做语义解释；`review_hints` 的内容是上一轮法师自由文本，不做结构化规范化。
- `quest_info` 在 BuildQuestUserMessage 之外的字段（rework_count、workspace_path 等）
  只能通过本工具获取。
- `gloop quest info` 返回稳定 JSON 字段，供 agent 直接解析。
