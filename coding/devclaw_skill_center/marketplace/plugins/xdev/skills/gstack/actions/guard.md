# guard

> 移植自 [gstack/guard](https://github.com/garrytan/gstack/blob/main/guard/SKILL.md) v0.15.1.0
> 上游 frontmatter / preamble / telemetry / proactive 配置已删除，原工作流逻辑保留。
> 中间产物统一落到 `{PRIMARY_REPO}/docs/xdev/gstack/{FEATURE_NAME}/guard.md`。
>
> ⚠️ **本仓库特殊说明**：本 skill 等于 careful + freeze 的组合，依赖上游 hook 实现破坏性命令拦截 + 编辑目录硬锁定。**devclaw 不提供 hook 机制**，本 action 迁移后**仅作为内容参考** —— 同样的限制：破坏性命令由 Claude Code 主进程的安全机制提供，编辑边界改为软提醒模式。

## 参数

- `ARG`(可选)：用户希望在 guard 模式下锁定编辑的目录路径。可为空——若为空则进入 Setup 阶段询问。

## 入参校验

读取并执行 `<skill_dir>/prompts/resolve_action_target.md`，传入：
- `ACTION_NAME = "guard"`
- `ARG = {用户传入的参数}`

执行完成后获得 `PRIMARY_REPO`、`PRIMARY_BRANCH`、`FEATURE_NAME`、`SPECS_DIR`、`IS_FEATURE_BRANCH`。

> 若本 action 是被 `/gstack run` 派发(pipeline 模式)，调用方已经预先设置 `SPECS_DIR` 等环境变量，本步骤检测到则跳过解析直接复用。

## 哲学约束

执行本 action 前，读取 `<skill_dir>/resources/ethos.md` 并将其内容作为隐含约束。关键原则(针对本 action)：

- **User Sovereignty**：双重防护(破坏命令 + 编辑边界)的范围与触发标准都由用户拍板，guard 不能擅自扩大或收缩。
- **Investigation Before Fix**：进入 guard 模式说明现场已经"踩雷可能成本高"，任何修改前都先把上下文捋清楚再下手，禁止边查边改。

---

## 上游工作流(已清理 gstack 私有基础设施)

# /guard — Full Safety Mode

Activates both destructive command warnings and directory-scoped edit restrictions.
This is the combination of `/careful` + `/freeze` in a single command.

**Dependency note:** This skill references hook scripts from the sibling `/careful`
and `/freeze` skill directories. Both must be installed (they are installed together
by the gstack setup script).

## Setup

Ask the user which directory to restrict edits to. Use AskUserQuestion:

- Question: "Guard mode: which directory should edits be restricted to? Destructive command warnings are always on. Files outside the chosen path will be blocked from editing."
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

Tell the user:
- "**Guard mode active.** Two protections are now running:"
- "1. **Destructive command warnings** — rm -rf, DROP TABLE, force-push, etc. will warn before executing (you can override)"
- "2. **Edit boundary** — file edits restricted to `<path>/`. Edits outside this directory are blocked."
- "To remove the edit boundary, run `/unfreeze`. To deactivate everything, end the session."

## What's protected

See `<skill_dir>/actions/careful.md` for the full list of destructive command patterns and safe exceptions.
See `<skill_dir>/actions/freeze.md` for how edit boundary enforcement works.

---

## 产物落地

本 action 的关键输出必须写入 `{SPECS_DIR}/guard.md`：

- 若文件已存在则**在文件末尾追加**新一轮的输出(用 `## 第 N 次执行 ({YYYY-MM-DD HH:MM:SS+TZ})` 作为分隔标题)
- 若文件不存在则创建并写入完整结构

执行完成后，输出一行简短摘要给用户：

```
✓ guard 完成。报告已写入 {SPECS_DIR}/guard.md
```
