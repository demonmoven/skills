# unfreeze

> 移植自 [gstack/unfreeze](https://github.com/garrytan/gstack/blob/main/unfreeze/SKILL.md) v0.15.1.0
> 上游 frontmatter / preamble / telemetry / proactive 配置已删除，原工作流逻辑保留。
> 中间产物统一落到 `{PRIMARY_REPO}/docs/xdev/gstack/{FEATURE_NAME}/unfreeze.md`。
>
> ⚠️ **本仓库特殊说明**：本 skill 用于解除 freeze 的目录锁定。如果 freeze 在本仓库无效(devclaw 不提供 hook 机制)，本 skill 也无效，仅作为内容参考。

## 参数

- `ARG`(可选)：通常无需参数。可为空。

## 入参校验

读取并执行 `<skill_dir>/prompts/resolve_action_target.md`，传入：
- `ACTION_NAME = "unfreeze"`
- `ARG = {用户传入的参数}`

执行完成后获得 `PRIMARY_REPO`、`PRIMARY_BRANCH`、`FEATURE_NAME`、`SPECS_DIR`、`IS_FEATURE_BRANCH`。

> 若本 action 是被 `/gstack run` 派发(pipeline 模式)，调用方已经预先设置 `SPECS_DIR` 等环境变量，本步骤检测到则跳过解析直接复用。

## 哲学约束

执行本 action 前，读取 `<skill_dir>/resources/ethos.md` 并将其内容作为隐含约束。关键原则(针对本 action)：

- **User Sovereignty**：解除编辑限制等于把方向盘交还给用户。永远不要在用户没有明确请求的情况下擅自 unfreeze。

---

## 上游工作流(已清理 gstack 私有基础设施)

# /unfreeze — Clear Freeze Boundary

Remove the edit restriction set by `/freeze`, allowing edits to all directories.

## Clear the boundary

```bash
STATE_DIR="${CLAUDE_PLUGIN_DATA:-$HOME/.gstack}"
if [ -f "$STATE_DIR/freeze-dir.txt" ]; then
  PREV=$(cat "$STATE_DIR/freeze-dir.txt")
  rm -f "$STATE_DIR/freeze-dir.txt"
  echo "Freeze boundary cleared (was: $PREV). Edits are now allowed everywhere."
else
  echo "No freeze boundary was set."
fi
```

Tell the user the result. Note that `/freeze` hooks are still registered for the
session — they will just allow everything since no state file exists. To re-freeze,
run `/freeze` again.

---

## 产物落地

本 action 的关键输出必须写入 `{SPECS_DIR}/unfreeze.md`：

- 若文件已存在则**在文件末尾追加**新一轮的输出(用 `## 第 N 次执行 ({YYYY-MM-DD HH:MM:SS+TZ})` 作为分隔标题)
- 若文件不存在则创建并写入完整结构

执行完成后，输出一行简短摘要给用户：

```
✓ unfreeze 完成。报告已写入 {SPECS_DIR}/unfreeze.md
```
