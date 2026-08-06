# Gloop v0.2.16

## 修复：base_working_dir 不展开 ~ 导致委托启动失败

**现象**：用户填 `~/agent-workspace/douyin-cli` 创建委托，报错找不到目录，但 quest 以 pending 状态残留（前端显示"发起失败"但看板显示"排队中"）。

**根因**：gloop 完全不展开 `~`。`~/agent-workspace/douyin-cli` 作为字面路径传给 `isGitRepo` / `git worktree add` / `cp`，全部失败（`~` 不是真实目录）。实际目录 `/home/.../agent-workspace/douyin-cli` 存在且是 git 仓库，但代码不认 `~`。

**修复**：
- 新增 `fsstore.ExpandDir(dir)`：展开 `~`（home 目录）和 `$VAR`（环境变量）
- `CreateQuest` 入口：`wd = ExpandDir(wd)`，meta 存展开后的绝对路径
- `Prepare` 入口：`baseDir = ExpandDir(baseDir)` 兜底（防御已有 meta 里的 `~` 路径）

现在用户填 `~/xxx` 或 `$HOME/xxx` 都能正确展开。

**清理**：#2606252562 僵尸 quest 已手动标记 cancelled（pending 残留 + workspace 空）。
