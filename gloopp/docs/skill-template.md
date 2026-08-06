---
name: skill-name
version: 0.1.0
description: "一句话描述这个 skill 做什么、典型触发场景、不负责什么。建议包含：核心能力 + 典型触发 + 边界（不负责什么）。"
metadata:
  class: both          # warrior | mage | both — 哪些职业可用
  category: utility    # 分类：execution / review / utility / info 等
  kind: capability     # capability | orchestration — 单能力 vs 编排型
  related_skills:
    - name: gloop-note-keeping
      type: related
      description: 简要说明关系
  requires:
    bins: ["gloop"]
    cliHelp: "gloop <cmd> --help"
---

# skill-name

用一段话简要介绍这个 skill 是做什么的、解决什么问题、agent 什么时候应该加载它。

## Trigger Examples

用表格或列表说明什么时候该用 / 不该用这个 skill。让 agent 一眼判断是否需要加载。

| 类型 | 场景 | 说明 |
|---|---|---|
| 适用 | <场景 1> | 什么时候应该调用这个 skill |
| 适用 | <场景 2> | 另一个典型触发场景 |
| 不适用 | <反例 1> | 什么时候不该用，应该走别的 skill |
| 不适用 | <反例 2> | 另一个边界情况 |

## CLI Contract

概述这个 skill 对应的 CLI 命令有哪些，整体用途是什么。

### 命令一览

| 命令 | 用途 |
|---|---|
| `gloop <cmd>` | 命令一句话描述 |
| `gloop <cmd2> <sub>` | 另一个命令 |

### 详细说明

#### gloop <cmd>

命令的详细说明：做什么、什么时候用。

**用法：**
```bash
gloop <cmd> --param "<value>"
```

**参数：**

| 参数 | 必填 | 类型 | 说明 |
|---|---|---|---|
| `--param` / `-p` | 是 | string | 参数说明 |
| `--flag` | 否 | flag | 开关参数说明 |
| `--json` | 否 | flag | 以 JSON 格式输出结果（agent syscall 场景自动输出） |

**返回：**

```json
{
  "ok": true,
  "message": "操作成功",
  "data": {
    "key": "value"
  }
}
```

**注意：**
- 注意事项 1
- 注意事项 2

### 通用约定

- CLI 是本地 agent 调用 gloop 平台能力的主路径
- 平台注入 `GLOOP_*` 环境变量，通常不需要手填 qid/sid
- 环境变量包括：`GLOOP_DATA_DIR`、`GLOOP_QUEST_ID`、`GLOOP_SESSION_ID`、`GLOOP_PHASE`、`GLOOP_ADVENTURER_CLASS`、`GLOOP_WORKSPACE_PATH`、`GLOOP_CONTEXT` 等
- **Syscall 机制**：CLI syscall 优先通过 `GLOOP_*` 环境变量发现当前 quest/sid；也会回退读取工作区 `.gloop/context.json`

## Discipline

使用这个 skill 时必须遵守的纪律和最佳实践。用要点列出来。

- 原则一：...
- 原则二：...
- 原则三：...

### 常见错误（可选）

- 错误 1：描述 + 正确做法
- 错误 2：描述 + 正确做法

### 最佳实践（可选）

- 实践 1
- 实践 2

## 写作建议（可选）

如果这个 skill 涉及输出内容（如通知、笔记、评审等），提供写作规范和示例。

### 好的例子

- 示例 1
- 示例 2

### 不好的例子

- 反例 1
- 反例 2

## 注意事项（可选）

其他需要提醒 agent 或开发者的事项。
