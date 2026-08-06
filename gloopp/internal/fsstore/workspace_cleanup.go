package fsstore

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

// ==================== Discard ====================

// Discard 丢弃工作区改动（不影响 base）。
// 只清理工作目录内容，保留 quest 元信息。
func (wm *WorkspaceManager) Discard(ctx context.Context, qid string, q *QuestMeta) error {
	qs := NewQuestStore(wm.root)
	workDir, err := qs.WorkDir(qid)
	if err != nil {
		return fmt.Errorf("创建 quest 工作目录失败: %w", err)
	}

	switch q.WorkspaceMode {
	case model.WorkspaceWorktree:
		// 先 git worktree remove
		baseDir := q.BaseWorkingDir
		if baseDir != "" && isGitRepo(ctx, baseDir) {
			cmd := exec.CommandContext(ctx, "git", "worktree", "remove", "--force", workDir)
			cmd.Dir = baseDir
			_ = cmd.Run() // 失败也没关系，后面直接删目录
		}
		// 然后删除分支
		if baseDir != "" && isGitRepo(ctx, baseDir) {
			branch := fmt.Sprintf("gloop/%s", qid)
			cmd := exec.CommandContext(ctx, "git", "branch", "-D", branch)
			cmd.Dir = baseDir
			_ = cmd.Run()
		}
		fallthrough // 最后统一删目录

	default:
		// 清空 workDir
		if err := os.RemoveAll(workDir); err != nil {
			return err
		}
		_ = os.MkdirAll(workDir, 0o755) // 留空目录
	}

	return nil
}

// ==================== Cleanup ====================

// Cleanup 彻底清理工作区（quest 删除时调用）。
func (wm *WorkspaceManager) Cleanup(ctx context.Context, qid string, q *QuestMeta) error {
	return wm.Discard(ctx, qid, q)
}

// ==================== Orphan Worktree 清理 ====================

// OrphanWorktreeInfo 描述一个被识别为孤儿的 worktree。
type OrphanWorktreeInfo struct {
	Path    string `json:"path"`
	Branch  string `json:"branch"`
	BaseDir string `json:"base_dir"`
	Reason  string `json:"reason"`
}

// CleanupOrphanWorktrees 扫描所有已知 baseDir，清理孤儿 worktree 和分支。
// 返回被清理的 worktree 列表。
//
// 判定孤儿的条件（保守策略，宁可漏删不可误删）：
//   - worktree 分支名匹配 gloop/<qid> 格式
//   - 对应的 qid 在 gloop quest 列表中不存在，或 quest 已终态
//
// 同时清理孤儿分支：gloop/<qid> 分支存在但 qid 对应的 quest 不存在或已终态，
// 即使 worktree 实体已不存在（手动删过目录但分支残留）。
func (wm *WorkspaceManager) CleanupOrphanWorktrees(ctx context.Context) ([]OrphanWorktreeInfo, error) {
	// 收集所有已知的 baseDir（去重）
	baseDirs, err := wm.collectBaseDirs()
	if err != nil {
		return nil, fmt.Errorf("收集 baseDir 失败: %w", err)
	}

	var cleaned []OrphanWorktreeInfo
	for _, baseDir := range baseDirs {
		orphans, err := wm.findOrphanWorktreesInBase(ctx, baseDir)
		if err != nil {
			// 单个 baseDir 失败不影响整体
			continue
		}
		for _, o := range orphans {
			if err := removeWorktreeAndBranch(ctx, baseDir, o.Path, o.Branch); err != nil {
				// 清理失败跳过，继续下一个
				continue
			}
			cleaned = append(cleaned, o)
		}
		// 清理孤儿分支（worktree 实体可能已不存在，但分支残留）
		wm.cleanOrphanBranches(ctx, baseDir, cleaned)
	}
	return cleaned, nil
}

// cleanOrphanBranches 清理 baseDir 下所有 gloop/<qid> 分支中 qid 不存在或已终态的。
// 独立于 worktree 实体扫描：即使 worktree 目录已删，分支也能清掉。
func (wm *WorkspaceManager) cleanOrphanBranches(ctx context.Context, baseDir string, cleaned []OrphanWorktreeInfo) {
	if !isGitRepo(ctx, baseDir) {
		return
	}
	out, err := runGit(ctx, baseDir, "branch", "--list", "gloop/*")
	if err != nil {
		return
	}
	qs := NewQuestStore(wm.root)
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "* ")
		if !strings.HasPrefix(line, "gloop/") {
			continue
		}
		branch := line
		qid := strings.TrimPrefix(branch, "gloop/")
		if qid == "" {
			continue
		}
		q, err := qs.LoadQuest(qid)
		isOrphan := false
		reason := ""
		if err != nil || q == nil {
			isOrphan = true
			reason = "对应的 quest 不存在"
		} else if q.Status.IsTerminal() {
			isOrphan = true
			reason = fmt.Sprintf("quest 已终态 (%s)", q.Status)
		}
		if !isOrphan {
			continue
		}
		// 删分支（worktree 实体可能已不存在，用 -D 强删）
		if _, err := runGit(ctx, baseDir, "branch", "-D", branch); err != nil {
			continue
		}
		// 如果没在 worktree 清理列表里，补一条记录
		alreadyCleaned := false
		for _, o := range cleaned {
			if o.Branch == branch {
				alreadyCleaned = true
				break
			}
		}
		if !alreadyCleaned {
			cleaned = append(cleaned, OrphanWorktreeInfo{
				Path:    "",
				Branch:  branch,
				BaseDir: baseDir,
				Reason:  reason,
			})
		}
	}
}

// collectBaseDirs 从所有 quest 中收集去重后的 baseDir 列表。
func (wm *WorkspaceManager) collectBaseDirs() ([]string, error) {
	qs := NewQuestStore(wm.root)
	quests, err := qs.ListQuests()
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var dirs []string
	for _, q := range quests {
		if q.BaseWorkingDir == "" {
			continue
		}
		if seen[q.BaseWorkingDir] {
			continue
		}
		seen[q.BaseWorkingDir] = true
		dirs = append(dirs, q.BaseWorkingDir)
	}
	return dirs, nil
}

// findOrphanWorktreesInBase 在单个 baseDir 下找出孤儿 worktree。
func (wm *WorkspaceManager) findOrphanWorktreesInBase(ctx context.Context, baseDir string) ([]OrphanWorktreeInfo, error) {
	if !isGitRepo(ctx, baseDir) {
		return nil, nil
	}

	trees, err := listWorktrees(ctx, baseDir)
	if err != nil {
		return nil, err
	}

	qs := NewQuestStore(wm.root)
	var orphans []OrphanWorktreeInfo
	for _, tree := range trees {
		if !strings.HasPrefix(tree.Branch, "gloop/") {
			continue
		}
		qid := strings.TrimPrefix(tree.Branch, "gloop/")
		if qid == "" {
			continue
		}

		// 检查 quest 是否存在
		q, err := qs.LoadQuest(qid)
		if err != nil || q == nil {
			orphans = append(orphans, OrphanWorktreeInfo{
				Path:    tree.Path,
				Branch:  tree.Branch,
				BaseDir: baseDir,
				Reason:  "对应的 quest 不存在",
			})
			continue
		}

		// quest 存在但不是 worktree 模式 → 也是孤儿（模式切换后遗漏的）
		if q.WorkspaceMode != model.WorkspaceWorktree {
			orphans = append(orphans, OrphanWorktreeInfo{
				Path:    tree.Path,
				Branch:  tree.Branch,
				BaseDir: baseDir,
				Reason:  fmt.Sprintf("quest 模式为 %s，不再是 worktree", q.WorkspaceMode),
			})
			continue
		}

		// quest 存在且是 worktree 模式，但已终态 → 孤儿（workspace 应随终态清理）
		if q.Status.IsTerminal() {
			orphans = append(orphans, OrphanWorktreeInfo{
				Path:    tree.Path,
				Branch:  tree.Branch,
				BaseDir: baseDir,
				Reason:  fmt.Sprintf("quest 已终态 (%s)", q.Status),
			})
			continue
		}

		// quest 存在且是 worktree 模式，但路径对不上 → 可能是旧的残留
		workDir, _ := qs.WorkDir(qid)
		if workDir != "" && tree.Path != workDir {
			orphans = append(orphans, OrphanWorktreeInfo{
				Path:    tree.Path,
				Branch:  tree.Branch,
				BaseDir: baseDir,
				Reason:  "worktree 路径与 quest 记录不一致",
			})
		}
	}
	return orphans, nil
}

// gitWorktreeEntry 是 `git worktree list --porcelain` 解析出的单条记录。
type gitWorktreeEntry struct {
	Path   string
	Branch string // 形如 refs/heads/gloop/qst_xxx 或 (detached HEAD)
}

// listWorktrees 解析 `git worktree list --porcelain` 的输出。
func listWorktrees(ctx context.Context, baseDir string) ([]gitWorktreeEntry, error) {
	cmd := exec.CommandContext(ctx, "git", "worktree", "list", "--porcelain")
	cmd.Dir = baseDir
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git worktree list failed: %w", err)
	}

	var entries []gitWorktreeEntry
	var current *gitWorktreeEntry
	for _, line := range strings.Split(string(out), "\n") {
		if line == "" {
			if current != nil {
				entries = append(entries, *current)
				current = nil
			}
			continue
		}
		if strings.HasPrefix(line, "worktree ") {
			current = &gitWorktreeEntry{Path: strings.TrimPrefix(line, "worktree ")}
			continue
		}
		if current == nil {
			continue
		}
		if strings.HasPrefix(line, "branch ") {
			ref := strings.TrimPrefix(line, "branch ")
			// 去掉 refs/heads/ 前缀
			current.Branch = strings.TrimPrefix(ref, "refs/heads/")
		}
	}
	// 最后一条如果没遇到空行
	if current != nil {
		entries = append(entries, *current)
	}
	return entries, nil
}

// removeWorktreeAndBranch 清理一个 worktree 及其分支，失败不返回错误（尽力而为）。
func removeWorktreeAndBranch(ctx context.Context, baseDir, worktreePath, branch string) error {
	// 先删 worktree
	if worktreePath != "" {
		cmd := exec.CommandContext(ctx, "git", "worktree", "remove", "--force", worktreePath)
		cmd.Dir = baseDir
		_ = cmd.Run()
	}
	// 再删分支
	if branch != "" {
		cmd := exec.CommandContext(ctx, "git", "branch", "-D", branch)
		cmd.Dir = baseDir
		_ = cmd.Run()
	}
	return nil
}
