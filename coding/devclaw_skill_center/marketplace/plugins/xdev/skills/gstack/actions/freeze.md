# freeze

> 移植自 [gstack/freeze](https://github.com/garrytan/gstack/blob/main/freeze/SKILL.md) v0.15.1.0
> 上游 frontmatter / preamble / telemetry / proactive 配置已删除，原工作流逻辑保留。
> 中间产物统一落到 `{PRIMARY_REPO}/docs/xdev/gstack/{FEATURE_NAME}/freeze.md`。
>
> ⚠️ **本仓库特殊说明**：本 skill 的核心是"限制文件编辑范围到指定目录"。上游通过 hook 实现硬拦截。**devclaw 不提供 hook 机制**，本 action 迁移后**仅作为内容参考** —— 实际的目录锁定可改为"软提醒"模式：约定 Claude 在写入前先比对路径，超出边界则停下来征询用户。

## 参数

- `ARG`(可选)：用户希望锁定编辑的目录路径。可为空——若为空则进入 Setup 阶段通过 AskUserQuestion 询问目标目录。

## 入参校验

读取并执行 `<skill_dir>/prompts/resolve_action_target.md`，传入：
- `ACTION_NAME = "freeze"`
- `ARG = {用户传入的参数}`

执行完成后获得 `PRIMARY_REPO`、`PRIMARY_BRANCH`、`FEATURE_NAME`、`SPECS_DIR`、`IS_FEATURE_BRANCH`。

> 若本 action 是被 `/gstack run` 派发(pipeline 模式)，调用方已经预先设置 `SPECS_DIR` 等环境变量，本步骤检测到则跳过解析直接复用。

## 哲学约束

执行本 action 前，读取 `<skill_dir>/resources/ethos.md` 并将其内容作为隐含约束。关键原则(针对本 action)：

- **User Sovereignty**：限制范围由用户决定，永远不要替用户猜目录。即使是显而易见的子目录，也要让用户拍板。

---

## 上游工作流(已清理 gstack 私有基础设施)

# /freeze — Restrict Edits to a Directory

Lock file edits to a specific directory. Any Edit or Write operation targeting
a file outside the allowed path will be **blocked** (not just warned).

## Setup

Ask the user which directory to restrict edits to. Use AskUserQuestion:

- Question: "Which directory should I restrict edits to? Files outside this path will be blocked from editing."
- Text input (not multiple choice) — the user types a path.

Once the user provides a directory path:

1. Resolve it to an absolute path:
```bash
FREEZE_DIR=$(cd "<user-provided-path>" 2>/dev/null && pwd)
echo "$FREEZE_DIR"
```

2. Ensure trailing slash and save to the freeze state file:
```bash
FREEZE_DIR="${FREEZE_DIR%/}/"
STATE_DIR="${CLAUDE_PLUGIN_DATA:-$HOME/.gstack}"
mkdir -p "$STATE_DIR"
echo "$FREEZE_DIR" > "$STATE_DIR/freeze-dir.txt"
echo "Freeze boundary set: $FREEZE_DIR"
```

Tell the user: "Edits are now restricted to `<path>/`. Any Edit or Write
outside this directory will be blocked. To change the boundary, run `/freeze`
again. To remove it, run `/unfreeze` or end the session."

## How it works

The hook reads `file_path` from the Edit/Write tool input JSON, then checks
whether the path starts with the freeze directory. If not, it returns
`permissionDecision: "deny"` to block the operation.

The freeze boundary persists for the session via the state file. The hook
script reads it on every Edit/Write invocation.

## Notes

- The trailing `/` on the freeze directory prevents `/src` from matching `/src-old`
- Freeze applies to Edit and Write tools only — Read, Bash, Glob, Grep are unaffected
- This prevents accidental edits, not a security boundary — Bash commands like `sed` can still modify files outside the boundary
- To deactivate, run `/unfreeze` or end the conversation

---

## 产物落地

本 action 的关键输出必须写入 `{SPECS_DIR}/freeze.md`：

- 若文件已存在则**在文件末尾追加**新一轮的输出(用 `## 第 N 次执行 ({YYYY-MM-DD HH:MM:SS+TZ})` 作为分隔标题)
- 若文件不存在则创建并写入完整结构

执行完成后，输出一行简短摘要给用户：

```
✓ freeze 完成。报告已写入 {SPECS_DIR}/freeze.md
```
