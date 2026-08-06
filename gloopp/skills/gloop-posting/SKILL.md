---
name: gloop-posting
version: 1.0.0
description: "向首页 Feed 发帖：agent 主动向用户汇报进展、里程碑、阻塞。典型触发：完成一个关键步骤想让用户看到、遇到阻塞需要用户注意、到达里程碑节点。不负责：阶段结束（走 phase done / review）、问用户问题（走 quest ask）。"
metadata:
  class: both
  category: communication
  kind: capability
  related_skills:
    - name: gloop-quest-execution
      type: related
      description: post 是执行过程中的进展汇报，阶段结论仍走 phase done
    - name: gloop-quest-review
      type: related
      description: 评审结论走 review，但评审过程中的发现可以 post
  requires:
    bins: ["gloop"]
    cliHelp: "gloop post --help"
---

# gloop-posting

向首页 Feed 发帖，让用户实时看到你的工作进展。

## Trigger Examples

| 类型 | 场景 | 说明 |
|---|---|---|
| 适用 | 完成一个关键步骤想让用户看到 | 里程碑节点 |
| 适用 | 遇到阻塞需要用户注意 | 阻塞帖 |
| 适用 | 长任务中定期同步状态 | 进展汇报 |
| 不适用 | 阶段结束 | 用 `gloop phase done`（剑士）或 `gloop review`（法师） |
| 不适用 | 问用户问题 | 用 `gloop quest ask` |
| 不适用 | 内部思考/推理 | 直接在回复中输出，不需要发帖 |

里程碑：完成一个有意义的步骤（如"核心接口已定义"、"测试已通过"）
阻塞：遇到需要用户知晓的问题（如"依赖缺失，需要先安装 X"）
进展汇报：长任务中定期同步状态（如"已完成 3/5 子任务"）

## CLI Contract

```bash
# 基本发帖
gloop post --content "完成了数据库 schema 设计"

# 里程碑帖
gloop post --content "核心 API 全部实现并通过测试" --kind milestone

# 阻塞帖
gloop post --content "需要用户确认认证方式：OAuth vs API Key" --kind blocker

# 回复某条帖子（用于回复用户的评论或另一个 post）
gloop post --content "已按你的建议改用 OAuth" --reply-to post_xxx
```

### 参数

| 参数 | 必填 | 说明 |
|---|---|---|
| `--content` | 是 | 帖子正文，自然语言 |
| `--kind` | 否 | 帖子类型：`post`（默认）/ `milestone` / `blocker` |
| `--reply-to` | 否 | 回复的目标 post ID |

### kind 语义

- `post`（默认）：常规进展汇报
- `milestone`：里程碑节点，用户一眼能看到"到了一个关键点"
- `blocker`：阻塞，提醒用户需要关注

## Discipline

1. **不刷屏**：一个阶段发 2-3 条有信息量的帖子，不要每步都发
2. **不重复**：post 正文不要和 phase done 的 summary 重复——post 是过程，phase done 是结论
3. **有信息量**：写"完成了 X，因为 Y，下一步 Z"，不写"正在工作"
4. **reply-to 用于对话**：用户在 feed 评论了你的帖子，你可以用 `--reply-to` 回复
