package fsstore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

// ApplyConflictError 表示 patch 与当前 base 冲突，apply 被拒绝。
// Files 是 git apply --check 报告的冲突文件列表。
type ApplyConflictError struct {
	Files []string
}

func (e *ApplyConflictError) Error() string {
	if len(e.Files) == 0 {
		return "patch 与当前 base 冲突，拒绝 apply"
	}
	return fmt.Sprintf("patch 与当前 base 冲突，拒绝 apply；冲突文件: %s", strings.Join(e.Files, ", "))
}

// ==================== Apply ====================

// Apply 把工作区的改动合回 base 目录。
// strategy 为空时默认 patch 策略。
// worktree 模式：根据策略选择 patch / merge / patch_then_merge。
// copy 模式：文件覆盖拷贝（策略不生效）。
// readonly 模式：报错。
// 返回本次 apply 的备份元信息（含 patch 引用与改动文件数），无改动时返回 nil。
func (wm *WorkspaceManager) Apply(ctx context.Context, qid string, q *QuestMeta, strategy string) (*ApplyBackupMeta, error) {
	if strategy == "" {
		strategy = ApplyStrategyPatch
	}
	switch q.WorkspaceMode {
	case model.WorkspaceWorktree:
		return wm.applyWorktree(ctx, qid, q, strategy)
	case model.WorkspaceCopy:
		return wm.applyCopy(ctx, qid, q)
	case model.WorkspaceReadOnly:
		return nil, errors.New("readonly 模式不支持 apply")
	default:
		return nil, fmt.Errorf("未知 workspace 模式: %s", q.WorkspaceMode)
	}
}

func (wm *WorkspaceManager) applyWorktree(ctx context.Context, qid string, q *QuestMeta, strategy string) (*ApplyBackupMeta, error) {
	qs := NewQuestStore(wm.root)
	workDir, err := qs.WorkDir(qid)
	if err != nil {
		return nil, fmt.Errorf("创建 quest 工作目录失败: %w", err)
	}
	baseDir := q.BaseWorkingDir
	if baseDir == "" {
		return nil, errors.New("base 目录为空，无法 apply")
	}

	// 获取 worktree 的 HEAD commit
	workHead, err := gitCurrentCommit(ctx, workDir)
	if err != nil {
		return nil, fmt.Errorf("获取 worktree HEAD 失败: %w", err)
	}

	// 记录 apply 前的 base HEAD。base 可能在 quest 创建后前进，只要 patch
	// 能干净应用，就不应阻塞；但失败回滚必须回到 apply 前的真实 HEAD。
	applyBaseHead, err := gitCurrentCommit(ctx, baseDir)
	if err != nil {
		return nil, fmt.Errorf("获取 base HEAD 失败: %w", err)
	}
	if dirty, err := gitDirty(ctx, baseDir, loadGloopIgnore(baseDir)); err != nil {
		return nil, fmt.Errorf("检查 base dirty 状态失败: %w", err)
	} else if dirty {
		return nil, errors.New("base 工作区有未提交改动，拒绝 apply")
	}

	// 如果 worktree 没有改动（workHead == baseCommit），直接返回
	if workHead == q.BaseCommit {
		return nil, nil // 没有改动，无需 apply
	}

	// 根据策略选择应用方式
	switch strategy {
	case ApplyStrategyMerge:
		return wm.applyWorktreeMerge(ctx, qid, q, workDir, baseDir, workHead, applyBaseHead)
	case ApplyStrategyPatchThenMerge:
		backup, err := wm.applyWorktreePatch(ctx, qid, q, workDir, baseDir, workHead, applyBaseHead)
		if err == nil {
			return backup, nil
		}
		// 只有冲突类错误才 fallback，其他错误（如 dirty base / 空 patch）直接返回
		if _, isConflict := err.(*ApplyConflictError); !isConflict {
			return nil, err
		}
		// patch 冲突，尝试 merge fallback
		return wm.applyWorktreeMerge(ctx, qid, q, workDir, baseDir, workHead, applyBaseHead)
	default: // patch
		return wm.applyWorktreePatch(ctx, qid, q, workDir, baseDir, workHead, applyBaseHead)
	}
}

// applyWorktreePatch 用 git diff + git apply 的方式合入，快且精确。
func (wm *WorkspaceManager) applyWorktreePatch(ctx context.Context, qid string, q *QuestMeta, workDir, baseDir, workHead, applyBaseHead string) (*ApplyBackupMeta, error) {
	patchArgs := append([]string{"diff", "--binary", q.BaseCommit, "HEAD"}, gitDiffPathspecs(q.BaseWorkingDir)...)
	patch, err := runGit(ctx, workDir, patchArgs...)
	if err != nil {
		return nil, fmt.Errorf("生成 patch 失败: %w", err)
	}
	if strings.TrimSpace(patch) == "" {
		return nil, nil
	}

	// 先 dry-run 探测冲突：git apply --check 不修改任何文件
	if _, checkErr := runGitWithInput(ctx, baseDir, patch, "apply", "--index", "--check", "-"); checkErr != nil {
		return nil, &ApplyConflictError{Files: parseApplyConflictFiles(checkErr.Error())}
	}

	backup, err := wm.createApplyBackup(ctx, qid, q, applyBaseHead, workHead, patch)
	if err != nil {
		return nil, fmt.Errorf("创建 apply 备份失败: %w", err)
	}
	if _, err := runGitWithInput(ctx, baseDir, patch, "apply", "--index", "-"); err != nil {
		_, _ = runGit(ctx, baseDir, "reset", "--hard", applyBaseHead)
		return nil, fmt.Errorf("git apply 失败: %w", err)
	}
	if _, err := runGit(ctx, baseDir, "commit", "-m", fmt.Sprintf("Apply Gloop quest %s", qid)); err != nil {
		_, _ = runGit(ctx, baseDir, "reset", "--hard", applyBaseHead)
		return nil, fmt.Errorf("提交 apply 结果失败: %w", err)
	}

	return backup, nil
}

// applyWorktreeMerge 用 git merge 的方式合入，能处理 base 前进后的三路合并。
// 缺点是会把 worktree 分支上的所有提交（包括中间提交）都带进来，历史可能不干净。
// workDir 参数暂未使用（共享对象库可直接用 commit hash merge），保留以保持与 patch 版本一致的签名。
func (wm *WorkspaceManager) applyWorktreeMerge(ctx context.Context, qid string, q *QuestMeta, _ /* workDir */, baseDir, workHead, applyBaseHead string) (*ApplyBackupMeta, error) {
	// merge 之前先确认 base 还是干净的（前面已经检查过，但防御性再查一次）
	if dirty, _ := gitDirty(ctx, baseDir, loadGloopIgnore(baseDir)); dirty {
		return nil, errors.New("base 工作区有未提交改动，拒绝 merge apply")
	}

	// 用 merge --no-ff 把 worktree 分支的改动合入 base
	// 注意：我们直接 merge worktree 分支的 HEAD commit，不需要先 fetch 因为共享对象库
	mergeMsg := fmt.Sprintf("Apply Gloop quest %s (merge)", qid)
	if _, err := runGit(ctx, baseDir, "merge", "--no-ff", "--no-edit", "-m", mergeMsg, workHead); err != nil {
		// merge 冲突或失败，回滚
		_, _ = runGit(ctx, baseDir, "merge", "--abort")
		_, _ = runGit(ctx, baseDir, "reset", "--hard", applyBaseHead)

		// 尝试解析冲突文件列表（从 git 输出里抓）
		statusOut, _ := runGit(ctx, baseDir, "diff", "--name-only", "--diff-filter=U")
		files := []string{}
		for _, line := range strings.Split(strings.TrimSpace(statusOut), "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				files = append(files, line)
			}
		}
		return nil, &ApplyConflictError{Files: files}
	}

	// merge 成功，生成 patch 用于备份和审计
	patchArgs := append([]string{"diff", "--binary", applyBaseHead, "HEAD"}, gitDiffPathspecs(q.BaseWorkingDir)...)
	patch, _ := runGit(ctx, baseDir, patchArgs...)

	backup, err := wm.createApplyBackup(ctx, qid, q, applyBaseHead, workHead, patch)
	if err != nil {
		// 备份失败不影响 merge 结果，因为 merge 已经成功了
		// 但还是要返回错误让调用方知道
		return backup, fmt.Errorf("merge 成功但创建备份失败: %w", err)
	}

	backup.ApplyMethod = "merge"
	return backup, nil
}

// parseApplyConflictFiles 从 git apply --check 的 stderr 中提取冲突文件路径。
// git 输出形如 "error: patch failed: path/to/file:12" 或 "error: path/to/file: ...".
func parseApplyConflictFiles(stderr string) []string {
	seen := map[string]bool{}
	var files []string
	for _, line := range strings.Split(stderr, "\n") {
		line = strings.TrimSpace(line)
		const prefix = "error: patch failed: "
		var path string
		if strings.HasPrefix(line, prefix) {
			path = strings.TrimPrefix(line, prefix)
			if i := strings.LastIndex(path, ":"); i > 0 {
				path = path[:i]
			}
		} else if strings.HasPrefix(line, "error: ") && strings.Contains(line, ": does not exist") {
			path = strings.TrimSuffix(strings.TrimPrefix(line, "error: "), ": does not exist in index")
		}
		path = strings.TrimSpace(path)
		if path != "" && !seen[path] {
			seen[path] = true
			files = append(files, path)
		}
	}
	return files
}

func (wm *WorkspaceManager) createApplyBackup(ctx context.Context, qid string, q *QuestMeta, applyBaseHead, workHead, patch string) (*ApplyBackupMeta, error) {
	qs := NewQuestStore(wm.root)
	backupID := fmt.Sprintf("apply_%d", NowMs())
	backupDir := filepath.Join(qs.dir(qid), "backups", backupID)
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return nil, err
	}

	patchPath := filepath.Join(backupDir, "apply.patch")
	if err := os.WriteFile(patchPath, []byte(patch), 0o644); err != nil {
		return nil, err
	}

	bundlePath := filepath.Join(backupDir, "base.bundle")
	if _, err := runGit(ctx, q.BaseWorkingDir, "bundle", "create", bundlePath, "HEAD"); err != nil {
		return nil, err
	}

	meta := &ApplyBackupMeta{
		ID:           backupID,
		QuestID:      qid,
		Mode:         string(q.WorkspaceMode),
		BaseDir:      q.BaseWorkingDir,
		BaseBranch:   q.BaseBranch,
		BaseCommit:   q.BaseCommit,
		ApplyCommit:  applyBaseHead,
		WorkCommit:   workHead,
		FilesChanged: strings.Count(patch, "diff --git "),
		CreatedAtMs:  NowMs(),
		BundlePath:   bundlePath,
		PatchPath:    patchPath,
	}
	if err := WriteJSON(filepath.Join(backupDir, "meta.json"), meta); err != nil {
		return nil, err
	}
	return meta, nil
}

func (wm *WorkspaceManager) applyCopy(ctx context.Context, qid string, q *QuestMeta) (*ApplyBackupMeta, error) {
	qs := NewQuestStore(wm.root)
	workDir, err := qs.WorkDir(qid)
	if err != nil {
		return nil, fmt.Errorf("创建 quest 工作目录失败: %w", err)
	}
	baseDir := q.BaseWorkingDir

	if baseDir == "" {
		return nil, errors.New("base 目录为空，无法 apply")
	}

	// git-init 模式：work 目录有 git，生成 patch 并保存备份。
	// apply 到 base 仍用文件覆盖（base 可能不是 git repo），
	// 但 patch 备份让审计和回滚能力对齐 worktree。
	var patch string
	useGit := isGitRepo(ctx, workDir)
	if useGit {
		// 先暂存所有改动（包括 untracked），让 git diff HEAD 能看到全部
		_, _ = runGit(ctx, workDir, "add", "--all", "--force", "--", ".", ":(exclude).gloop")
		patchArgs := append([]string{"diff", "--binary", "HEAD"}, gitDiffPathspecs(q.BaseWorkingDir)...)
		patch, _ = runGit(ctx, workDir, patchArgs...)
		// reset 暂存区（恢复到 HEAD，不影响工作区文件）
		_, _ = runGit(ctx, workDir, "reset", "--quiet", "--mixed", "HEAD")
	}
	if strings.TrimSpace(patch) == "" {
		return nil, nil // 没有改动
	}

	// 文件级 apply：从 git patch 解析变更文件列表（精确反映 agent 改动）。
	// 不再用 diffDirs(baseDir, workDir)——base 可能已经 drift，导致虚假 diff。
	var files []DiffFileStat
	if useGit {
		files, _, _ = parsePatchStats(patch)
	}
	if len(files) == 0 {
		// fallback: git patch 解析失败或为空，用 diffDirs
		var diffErr error
		files, _, _, diffErr = diffDirs(baseDir, workDir)
		if diffErr != nil {
			return nil, diffErr
		}
	}

	for _, f := range files {
		srcPath := filepath.Join(workDir, f.Path)
		dstPath := filepath.Join(baseDir, f.Path)

		switch f.Status {
		case "added", "modified":
			if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
				return nil, fmt.Errorf("创建目标目录失败: %w", err)
			}
			if err := copyFile(srcPath, dstPath); err != nil {
				return nil, fmt.Errorf("拷贝文件失败 %s: %w", f.Path, err)
			}
		case "deleted":
			if err := os.Remove(dstPath); err != nil && !os.IsNotExist(err) {
				return nil, fmt.Errorf("删除文件失败 %s: %w", f.Path, err)
			}
		}
	}

	// FilesChanged 用 git patch 的文件数（和 diff 一致），fallback 到 diffDirs
	filesChanged := len(files)
	if useGit {
		if parsed, _, _ := parsePatchStats(patch); len(parsed) > 0 {
			filesChanged = len(parsed)
		}
	}

	backupID := fmt.Sprintf("apply_%d", NowMs())
	backupDir := filepath.Join(qs.dir(qid), "backups", backupID)
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return nil, fmt.Errorf("创建 apply 备份失败: %w", err)
	}
	// 保存 patch 备份（和 worktree 模式一致，支持审计和回滚）
	patchPath := filepath.Join(backupDir, "apply.patch")
	if err := os.WriteFile(patchPath, []byte(patch), 0o644); err != nil {
		return nil, fmt.Errorf("保存 patch 备份失败: %w", err)
	}
	meta := &ApplyBackupMeta{
		ID:           backupID,
		QuestID:      qid,
		Mode:         string(q.WorkspaceMode),
		BaseDir:      baseDir,
		FilesChanged: filesChanged,
		CreatedAtMs:  NowMs(),
		PatchPath:    patchPath,
	}
	if err := WriteJSON(filepath.Join(backupDir, "meta.json"), meta); err != nil {
		return nil, err
	}
	return meta, nil
}
