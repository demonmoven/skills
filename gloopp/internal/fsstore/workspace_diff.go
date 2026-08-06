package fsstore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

// ==================== Diff ====================

// ComputeDiff 计算工作区相对 base 的差异。
// readonly 模式返回错误。
func (wm *WorkspaceManager) ComputeDiff(ctx context.Context, qid string, q *QuestMeta) (DiffResult, error) {
	if q.Applied {
		if diff, err := wm.computeDiffFromApplyBackup(qid); err == nil {
			return diff, nil
		}
		if q.DiffStat != "" || q.DiffChangedFiles > 0 {
			return DiffResult{
				Stat:         q.DiffStat,
				ChangedFiles: q.DiffChangedFiles,
				Additions:    q.DiffAdditions,
				Deletions:    q.DiffDeletions,
				Source:       "cached_summary",
			}, nil
		}
	}
	switch q.WorkspaceMode {
	case model.WorkspaceWorktree:
		return wm.computeDiffWorktree(ctx, qid, q)
	case model.WorkspaceCopy:
		return wm.computeDiffCopy(ctx, qid, q)
	case model.WorkspaceReadOnly:
		return DiffResult{}, errors.New("readonly 模式不支持 diff")
	default:
		return DiffResult{}, fmt.Errorf("未知 workspace 模式: %s", q.WorkspaceMode)
	}
}

func (wm *WorkspaceManager) computeDiffFromApplyBackup(qid string) (DiffResult, error) {
	backups, err := wm.ListApplyBackups(qid)
	if err != nil {
		return DiffResult{}, err
	}
	var summary *ApplyBackupMeta
	for _, backup := range backups {
		if backup == nil {
			continue
		}
		if summary == nil {
			summary = backup
		}
		if backup.PatchPath == "" {
			continue
		}
		patch, err := os.ReadFile(backup.PatchPath)
		if err != nil {
			continue
		}
		raw := string(patch)
		files, additions, deletions := parsePatchStats(raw)
		stat := fmt.Sprintf("%d files changed, %d insertions(+), %d deletions(-)", len(files), additions, deletions)
		return DiffResult{
			Stat:         stat,
			ChangedFiles: len(files),
			Additions:    additions,
			Deletions:    deletions,
			Files:        files,
			Diff:         raw,
			Source:       "apply_backup",
			BackupID:     backup.ID,
		}, nil
	}
	if summary != nil && summary.FilesChanged > 0 {
		stat := fmt.Sprintf("%d files changed", summary.FilesChanged)
		return DiffResult{
			Stat:         stat,
			ChangedFiles: summary.FilesChanged,
			Source:       "apply_backup",
			BackupID:     summary.ID,
		}, nil
	}
	return DiffResult{}, fmt.Errorf("未找到可读 apply patch 备份")
}

func (wm *WorkspaceManager) computeDiffWorktree(ctx context.Context, qid string, q *QuestMeta) (DiffResult, error) {
	qs := NewQuestStore(wm.root)
	workDir, err := qs.WorkDir(qid)
	if err != nil {
		return DiffResult{}, fmt.Errorf("创建 quest 工作目录失败: %w", err)
	}

	pathspecs := gitDiffPathspecs(q.BaseWorkingDir)

	// git diff --stat
	statArgs := append([]string{"diff", "--stat", "--no-renames", q.BaseCommit, "HEAD"}, pathspecs...)
	statOut, err := runGit(ctx, workDir, statArgs...)
	if err != nil {
		return DiffResult{}, fmt.Errorf("git diff --stat 失败: %w", err)
	}
	stat := strings.TrimSpace(statOut)

	// git diff --numstat 用于文件列表
	numstatArgs := append([]string{"diff", "--numstat", "--no-renames", q.BaseCommit, "HEAD"}, pathspecs...)
	numstatOut, err := runGit(ctx, workDir, numstatArgs...)
	if err != nil {
		return DiffResult{}, fmt.Errorf("git diff --numstat 失败: %w", err)
	}

	statusArgs := append([]string{"diff", "--name-status", "--no-renames", q.BaseCommit, "HEAD"}, pathspecs...)
	statusOut, err := runGit(ctx, workDir, statusArgs...)
	if err != nil {
		return DiffResult{}, fmt.Errorf("git diff --name-status 失败: %w", err)
	}

	files, totalAdd, totalDel := parseNumstatWithStatus(numstatOut, parseNameStatus(statusOut))

	// 完整 diff（截断保护：超过 100KB 就只给 stat）
	diffOut := ""
	diffArgs := append([]string{"diff", "--no-renames", q.BaseCommit, "HEAD"}, pathspecs...)
	diffRaw, err := runGit(ctx, workDir, diffArgs...)
	if err == nil && len(diffRaw) < 100*1024 {
		diffOut = diffRaw
	}

	return DiffResult{
		Stat:         stat,
		ChangedFiles: len(files),
		Additions:    totalAdd,
		Deletions:    totalDel,
		Files:        files,
		Diff:         diffOut,
	}, nil
}

func parsePatchStats(raw string) ([]DiffFileStat, int, int) {
	var files []DiffFileStat
	additions := 0
	deletions := 0
	var current *DiffFileStat
	for _, line := range strings.Split(raw, "\n") {
		if strings.HasPrefix(line, "diff --git ") {
			if current != nil {
				files = append(files, *current)
			}
			current = &DiffFileStat{Path: parseDiffGitPath(line), Status: "modified"}
			continue
		}
		if current == nil {
			continue
		}
		switch {
		case strings.HasPrefix(line, "new file mode"):
			current.Status = "added"
		case strings.HasPrefix(line, "deleted file mode"):
			current.Status = "deleted"
		case strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++"):
			current.Additions++
			additions++
		case strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---"):
			current.Deletions++
			deletions++
		}
	}
	if current != nil {
		files = append(files, *current)
	}
	return files, additions, deletions
}

func parseDiffGitPath(line string) string {
	parts := strings.Split(line, " ")
	if len(parts) >= 4 {
		return strings.TrimPrefix(parts[3], "b/")
	}
	return ""
}

func (wm *WorkspaceManager) computeDiffCopy(ctx context.Context, qid string, q *QuestMeta) (DiffResult, error) {
	qs := NewQuestStore(wm.root)
	workDir, err := qs.WorkDir(qid)
	if err != nil {
		return DiffResult{}, fmt.Errorf("创建 quest 工作目录失败: %w", err)
	}

	// git-init 模式：work 目录有 git，用 git diff 对比基线和当前工作区。
	// 回退到 diffDirs 仅在 git 不可用时（老 quest 未 init 基线）。
	if isGitRepo(ctx, workDir) {
		return wm.computeDiffCopyGit(ctx, qid, q, workDir)
	}

	// 回退：老 quest 没有 git 基线，用文件级 diff
	if q.BaseWorkingDir == "" {
		return DiffResult{}, errors.New("base 目录为空，无法计算 diff")
	}
	files, add, del, err := diffDirs(q.BaseWorkingDir, workDir)
	if err != nil {
		return DiffResult{}, err
	}
	stat := fmt.Sprintf("%d files changed, %d insertions(+), %d deletions(-)", len(files), add, del)
	return DiffResult{
		Stat:         stat,
		ChangedFiles: len(files),
		Additions:    add,
		Deletions:    del,
		Files:        files,
		Source:       "file_diff_fallback",
	}, nil
}

// computeDiffCopyGit 用 git diff 对比 copy 模式的基线（初始 commit）和当前工作区。
// 精度和 worktree 模式一致：真实行级 diff，不是 diffFileLines 的行数近似。
// 注意：copy 模式 agent 的改动是未跟踪的，git diff HEAD 默认不显示新增文件。
// 这里先 git add -A 暂存（不 commit），让 diff 能看到所有改动，然后 reset 暂存区。
func (wm *WorkspaceManager) computeDiffCopyGit(ctx context.Context, qid string, q *QuestMeta, workDir string) (DiffResult, error) {
	// 暂存所有改动（包括 untracked），让 git diff HEAD 能看到所有改动。
	// 排除 .gloop（scratch 目录的改动不应出现在 diff 里）。
	if _, err := runGit(ctx, workDir, "add", "--all", "--force", "--", ".", ":(exclude).gloop"); err != nil {
		return DiffResult{}, fmt.Errorf("git add 失败: %w", err)
	}
	// 确保暂存的改动回到工作区（不改变工作区文件内容）
	// reset --mixed 只把暂存区重置到 HEAD，但我们要 diff 的是工作区 vs HEAD，
	// 所以这里不 reset——直接用 git diff HEAD --cached 看暂存的改动
	// 但 --cached 只看暂存区 vs HEAD，而工作区可能还有未暂存的改动。
	// 最准确：git add --all 后 git diff HEAD（工作区已全部暂存，diff HEAD 显示全部）
	// pathspec 用 base 目录的 .gloopignore（work 目录的 .gloopignore 在 copyDir 时被跳过）
	pathspecs := gitDiffPathspecs(q.BaseWorkingDir)

	// git diff --stat
	statArgs := append([]string{"diff", "--stat", "--no-renames", "HEAD"}, pathspecs...)
	statOut, err := runGit(ctx, workDir, statArgs...)
	if err != nil {
		return DiffResult{}, fmt.Errorf("git diff --stat 失败: %w", err)
	}
	stat := strings.TrimSpace(statOut)

	// git diff --numstat 用于文件列表
	numstatArgs := append([]string{"diff", "--numstat", "--no-renames", "HEAD"}, pathspecs...)
	numstatOut, err := runGit(ctx, workDir, numstatArgs...)
	if err != nil {
		return DiffResult{}, fmt.Errorf("git diff --numstat 失败: %w", err)
	}

	statusArgs := append([]string{"diff", "--name-status", "--no-renames", "HEAD"}, pathspecs...)
	statusOut, err := runGit(ctx, workDir, statusArgs...)
	if err != nil {
		return DiffResult{}, fmt.Errorf("git diff --name-status 失败: %w", err)
	}

	files, totalAdd, totalDel := parseNumstatWithStatus(numstatOut, parseNameStatus(statusOut))

	// 完整 diff（截断保护：超过 100KB 就只给 stat）
	diffOut := ""
	diffArgs := append([]string{"diff", "--no-renames", "HEAD"}, pathspecs...)
	diffRaw, err := runGit(ctx, workDir, diffArgs...)
	if err == nil && len(diffRaw) < 100*1024 {
		diffOut = diffRaw
	}

	// reset 暂存区（恢复到 HEAD，不影响工作区文件）
	_, _ = runGit(ctx, workDir, "reset", "--quiet", "--mixed", "HEAD")

	return DiffResult{
		Stat:         stat,
		ChangedFiles: len(files),
		Additions:    totalAdd,
		Deletions:    totalDel,
		Files:        files,
		Diff:         diffOut,
		Source:       "copy_git",
	}, nil
}

// parseNumstat 解析 git diff --numstat 输出
func parseNumstat(raw string) ([]DiffFileStat, int, int) {
	return parseNumstatWithStatus(raw, nil)
}

func parseNumstatWithStatus(raw string, statusByPath map[string]string) ([]DiffFileStat, int, int) {
	var files []DiffFileStat
	totalAdd := 0
	totalDel := 0
	seen := map[string]bool{}
	for _, line := range strings.Split(strings.TrimSpace(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 3 {
			continue
		}
		add := parseCount(parts[0])
		del := parseCount(parts[1])
		path := parts[2]

		status := "modified"
		if statusByPath != nil && statusByPath[path] != "" {
			status = statusByPath[path]
		} else if add > 0 && del == 0 {
			status = "added"
		} else if add == 0 && del > 0 {
			status = "deleted"
		}

		seen[path] = true
		files = append(files, DiffFileStat{
			Path:      path,
			Status:    status,
			Additions: add,
			Deletions: del,
		})
		totalAdd += add
		totalDel += del
	}
	for path, status := range statusByPath {
		if seen[path] {
			continue
		}
		files = append(files, DiffFileStat{Path: path, Status: status})
	}
	return files, totalAdd, totalDel
}

func parseNameStatus(raw string) map[string]string {
	out := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 2 {
			continue
		}
		status := "modified"
		switch parts[0][0] {
		case 'A':
			status = "added"
		case 'D':
			status = "deleted"
		}
		out[parts[len(parts)-1]] = status
	}
	return out
}

func parseCount(s string) int {
	if s == "-" {
		return 0 // 二进制文件
	}
	n := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		} else {
			return 0
		}
	}
	return n
}
