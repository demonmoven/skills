---
name: gloop-note-keeping
version: 1.0.0
description: "笔记系统：剑士和法师在执行/评审过程中给未来的自己和对方留下结构化记忆。典型触发：做了一个重要决定要记下原因、发现潜在风险不想忘了、有一个 TODO 留到下轮处理、返工前把当前没做完的事写清楚。不负责：正式的阶段结论（走 phase done / review）、公开问用户问题（走 quest ask）。"
metadata:
  class: both
  category: note
  kind: capability
  related_skills:
    - name: gloop-self-awareness
      type: related
      description: 笔记是事件流的一部分，可通过 quest history 查询
    - name: gloop-user-notification
      type: related
      description: 通知不可用时降级用 note 记录；一般信息直接用 note 不必发通知
    - name: gloop-posting
      type: related
      description: note 是 quest 内部记忆，post 是 Feed 可见进展——需要让用户看到用 post
  requires:
    bins: ["gloop"]
    cliHelp: "gloop note --help"
---

# gloop-note-keeping

冒险者「记忆系统」的 CLI syscall 协议。

## Trigger Examples

| 类型 | 场景 | 说明 |
|---|---|---|
| 适用 | 想留下跨阶段能看到的持久化备注 | 笔记的职责域 |
| 不适用 | 想提交阶段结论 | 阶段结论走 phase_* / review_*，平台只认它们 |
| 不适用 | 想让平台用户回答问题 | 走 quest_ask，笔记不会通知用户 |

## CLI Contract

通过 `gloop note` 命令管理 quest 内的结构化笔记，支持追加和查看，笔记跨阶段持久化。

### 命令一览

| 命令 | 用途 |
|---|---|
| `gloop note add` | 追加一条笔记 |
| `gloop note list` | 查看笔记列表 |

### 详细说明

#### gloop note add

向当前 quest 追加一条笔记。

**用法：**
```bash
gloop note add --tag risk "<内容>"
gloop note add --tag todo "<内容>"
gloop note add "<内容>"
```

**参数：**

| 参数 | 必填 | 类型 | 说明 |
|---|---|---|---|
| `<内容>` | 是 | string | 笔记内容（位置参数） |
| `--tag` | 否 | string | 自由字符串标签，不传则为空 |

**返回：**

无（纯副作用命令）

**注意：**
- `tag` 是自由字符串，平台不做枚举校验；以下为历史使用示例，不是规范：

| tag 示例 | 含义（作者可自定义） |
|---|---|
| `risk` | 风险 / 隐患线索 |
| `todo` | 未完成项线索 |
| `idea` | 后续可跟进的想法 |
| `decision` | 决策与权衡记录 |
| `debug` | 排查中间线索 |

#### gloop note list

查看当前 quest 下的所有笔记列表。

**用法：**
```bash
gloop note list
```

**参数：**

无参数

**返回：**

返回笔记列表。

### 通用约定

- CLI 是本地 agent 调用 gloop 平台能力的主路径
- 平台注入 `GLOOP_*` 环境变量，通常不需要手填 qid/sid

## Discipline

- 笔记是**永久追加**的：写入后无法修改或删除。
- 笔记对同一 quest 下后续所有阶段、所有返工轮次、对方职业均可见。
- 笔记不会触发任何状态流转；阶段结论只能通过对应的 CLI 命令或 native tool 提交。
