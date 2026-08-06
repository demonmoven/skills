---
name: gloop-user-notification
version: 1.0.0
description: "用户通知：剑士和法师在重要里程碑、阻塞节点或需要用户介入时，主动通过飞书消息触达用户。典型触发：阶段完成、遇到阻塞需要用户解决、有重大决策需要用户确认、发现意料之外的重要信息。不负责：一般进展更新（用 post_update）、日志式信息（用 note_add）、每回合刷屏。"
metadata:
  class: both
  category: utility
  kind: capability
  related_skills:
    - name: gloop-note-keeping
      type: related
      description: 通知不可用时降级用 note 记录；一般信息直接用 note 不必发通知
    - name: gloop-inbox-triage
      type: see_also
      description: 收件箱分类是通知的一个重要消费场景
  requires:
    bins: ["gloop"]
    cliHelp: "gloop notify --help"
---

# gloop-user-notification

通过 `gloop notify` 命令，主动给用户发送飞书通知。

## Trigger Examples

| 类型 | 场景 | 说明 |
|---|---|---|
| 适用 | 任务完成 / 阶段结束 | 终态达成或重要里程碑时告知用户进展 |
| 适用 | 遇到阻塞，需要用户介入 | 任务 blocked，缺权限、缺信息、缺依赖等 |
| 适用 | 重要决策点需要用户确认 | 方案选择、风险确认、需求澄清等 |
| 适用 | 执行中发现重大意外信息 | 可能影响整体方向的发现 |
| 不适用 | 一般过程更新 | 用 `post_update` 或 `note_add` 即可 |
| 不适用 | 日志式 / 调试信息 | 用 `note_add` 记笔记就够了 |
| 不适用 | 每回合都发 | 绝对不要刷屏，通知要有信息量 |

## CLI Contract

通过 `gloop notify` 命令发送飞书通知，覆盖重要里程碑、阻塞节点和需要用户介入的场景。

### 命令一览

| 命令 | 用途 |
|---|---|
| `gloop notify` | 主动给用户发送一条飞书通知 |

### 详细说明

#### gloop notify

发送一条飞书通知给当前用户，用于重要里程碑、阻塞节点或需要用户介入的场景。

**用法：**
```bash
gloop notify --title "<简短标题>" --body "<markdown 正文>" --priority normal
```

**参数：**

| 参数 | 必填 | 类型 | 说明 |
|---|---|---|---|
| `--title` / `-t` | 是 | string | 通知标题，简短清晰（15字以内最佳） |
| `--body` / `-b` | 否 | string | Markdown 格式的正文，说明情况和建议动作 |
| `--priority` / `-p` | 否 | string | `normal`（默认）或 `high`（高优，用于阻塞、需用户立即关注） |
| `--json` | 否 | flag | 以 JSON 格式输出结果（agent syscall 场景自动输出） |

**返回：**

```json
{
  "ok": true,
  "message": "已发送",
  "data": {
    "available": true,
    "channel": "lark"
  }
}
```

**注意：**
- 如果飞书通知不可用，`ok` 为 `false`，`message` 会说明原因。
- 通知不可用时**不要反复重试**，改用 `gloop note add` 或 `gloop quest post-update` 记录信息即可。

### 通用约定

- CLI 是本地 agent 调用 gloop 平台能力的主路径
- 平台注入 `GLOOP_*` 环境变量，通常不需要手填 qid/sid

## Discipline

- 通知是 **best-effort** 的：发送失败不影响任务执行，也不要反复重试。
- **宁可少发，不要滥发**：每次通知都要有信息量，同一 quest 内同类通知不要超过 2-3 条。
- 终态（success/failed）可以发一条总结通知。
- 不要用通知代替笔记或平台更新：一般进展用 `post_update`，需要持久化的信息用 `note_add`。
- 如果通知不可用（`ok=false`），重要信息请用 `note_add` 记下来，不要在通知上浪费 token。
- 通知接收人是当前 lark-cli 登录用户，平台不会泄露给第三方。

## 写作建议

### 好的通知标题

- 「任务已完成：实现用户登录功能」
- 「遇到阻塞：缺少数据库访问权限」
- 「需要确认：两种方案选择」

### 好的通知正文

简要说明：
1. 发生了什么
2. 为什么重要
3. 用户需要做什么（如果需要）

示例：

```
执行阶段已完成，实现了以下内容：

- 用户注册接口
- 密码加密存储
- 单元测试覆盖

下一步：进入评审阶段，预计 2 分钟后完成。
```
