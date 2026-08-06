# Gloop v0.2.31

## 优化：copy 模式升级 git-init — diff 精度对齐 worktree

### 现象

copy 模式用文件系统原语复刻 git 能力，但精度全面打折：
- diff 是行数近似（diffFileLines = workLines - baseLines），stat 不准
- apply 是文件覆盖无冲突检测
- 无 patch 备份，审计和回滚能力残缺

### 修复

copy 模式 work 目录 git init + base 快照 commit 成基线，diff/apply 复用 git 原语。

**1. prepareCopy：git-init 基线**

work 目录复制完成后，`git init` + `git add --all` + `git commit` 成基线。
`.gloop/` 不纳入基线（通过 `git reset -- .gloop` 排除），agent 在 scratch 写文件不污染 diff。

**2. computeDiffCopy：git diff 替代 diffDirs**

用 `git diff HEAD` 对比基线和当前工作区，精度对齐 worktree：
- 真实行级 diff，不是行数近似
- pathspec 用 base 目录的 .gloopignore（work 目录的 .gloopignore 在 copyDir 时被跳过）
- 先 `git add --all` 暂存（包括 untracked），让 diff 能看到新增文件，然后 reset 暂存区

老 quest（无 git 基线）回退到 diffDirs 文件级 diff。

**3. applyCopy：patch 备份**

apply 仍用文件覆盖（base 可能不是 git repo），但生成 patch 备份：
- `git diff HEAD --binary` 生成 patch
- FilesChanged 用 patch 的文件数（和 diff 一致）
- patch 保存到 backups 目录，审计和回滚能力对齐 worktree

### 效果

copy 模式 diff 精度从"行数近似"升级到"真实行级 diff"，stat 准确，前端 diff 质量对齐 worktree。
apply 有 patch 备份，审计和回滚能力补齐。

### 变更

- `internal/fsstore/workspace_prepare.go`：
  - `prepareCopy` 复制后调 `initGitBaseline`
  - 新增 `initGitBaseline`：git init + config + add（排除 .gloop）+ commit
  - 新增 `copyBaselineCommit` 常量（"HEAD"）
- `internal/fsstore/workspace_diff.go`：
  - `computeDiffCopy` 优先用 git，回退 diffDirs
  - 新增 `computeDiffCopyGit`：git diff HEAD，精度对齐 worktree
- `internal/fsstore/workspace_apply.go`：
  - `applyCopy` 生成 patch 备份，FilesChanged 用 patch 文件数
