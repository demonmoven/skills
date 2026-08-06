# careful

> 移植自 [gstack/careful](https://github.com/garrytan/gstack/blob/main/careful/SKILL.md) v0.15.1.0
> 上游 frontmatter / preamble / telemetry / proactive 配置已删除，原工作流逻辑保留。
> 中间产物统一落到 `{PRIMARY_REPO}/docs/xdev/gstack/{FEATURE_NAME}/careful.md`。
>
> ⚠️ **本仓库特殊说明**：本 skill 的核心机制是"在执行破坏性命令前警告"。上游 gstack 通过 hook 拦截 `rm -rf` / `DROP TABLE` / `git push --force` 等命令实现。**devclaw 不提供同等的 hook 机制**，本 action 迁移后**仅作为内容参考** —— 实际的破坏性命令拦截由 Claude Code 主进程的安全机制提供，本 action 起到"提醒模式"的作用。

## 参数

- `ARG`(可选)：用户希望进入安全模式的语义触发，例如 "be careful"、"safety mode"、"prod mode"、"careful mode"。可为空。

## 入参校验

读取并执行 `<skill_dir>/prompts/resolve_action_target.md`，传入：
- `ACTION_NAME = "careful"`
- `ARG = {用户传入的参数}`

执行完成后获得 `PRIMARY_REPO`、`PRIMARY_BRANCH`、`FEATURE_NAME`、`SPECS_DIR`、`IS_FEATURE_BRANCH`。

> 若本 action 是被 `/gstack run` 派发(pipeline 模式)，调用方已经预先设置 `SPECS_DIR` 等环境变量，本步骤检测到则跳过解析直接复用。

## 哲学约束

执行本 action 前，读取 `<skill_dir>/resources/ethos.md` 并将其内容作为隐含约束。关键原则(针对本 action)：

- **User Sovereignty**：所有破坏性操作必须征询用户。即使模式判定一个命令"安全"，最终决定权也属于用户；careful 只负责挑明风险，不替用户拍板。

---

## 上游工作流(已清理 gstack 私有基础设施)

# /careful — Destructive Command Guardrails

Safety mode is now **active**. Every bash command will be checked for destructive
patterns before running. If a destructive command is detected, you'll be warned
and can choose to proceed or cancel.

## What's protected

| Pattern | Example | Risk |
|---------|---------|------|
| `rm -rf` / `rm -r` / `rm --recursive` | `rm -rf /var/data` | Recursive delete |
| `DROP TABLE` / `DROP DATABASE` | `DROP TABLE users;` | Data loss |
| `TRUNCATE` | `TRUNCATE orders;` | Data loss |
| `git push --force` / `-f` | `git push -f origin main` | History rewrite |
| `git reset --hard` | `git reset --hard HEAD~3` | Uncommitted work loss |
| `git checkout .` / `git restore .` | `git checkout .` | Uncommitted work loss |
| `kubectl delete` | `kubectl delete pod` | Production impact |
| `docker rm -f` / `docker system prune` | `docker system prune -a` | Container/image loss |

## Safe exceptions

These patterns are allowed without warning:
- `rm -rf node_modules` / `.next` / `dist` / `__pycache__` / `.cache` / `build` / `.turbo` / `coverage`

## How it works

The hook reads the command from the tool input JSON, checks it against the
patterns above, and returns `permissionDecision: "ask"` with a warning message
if a match is found. You can always override the warning and proceed.

To deactivate, end the conversation or start a new one. Hooks are session-scoped.

---

## 产物落地

本 action 的关键输出必须写入 `{SPECS_DIR}/careful.md`：

- 若文件已存在则**在文件末尾追加**新一轮的输出(用 `## 第 N 次执行 ({YYYY-MM-DD HH:MM:SS+TZ})` 作为分隔标题)
- 若文件不存在则创建并写入完整结构

执行完成后，输出一行简短摘要给用户：

```
✓ careful 完成。报告已写入 {SPECS_DIR}/careful.md
```
