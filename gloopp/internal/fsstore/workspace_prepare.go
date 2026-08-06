package fsstore

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

// ==================== 模式探测 ====================

// DetectMode 根据 baseDir 自动选择工作区模式：
//   - 是 git repo → worktree
//   - 不是 git repo + 目录存在 → copy
//   - 目录不存在或为空 → copy（空目录起步）
func (wm *WorkspaceManager) DetectMode(ctx context.Context, baseDir string) model.WorkspaceMode {
	if isGitRepo(ctx, baseDir) {
		return model.WorkspaceWorktree
	}
	if baseDir == "" {
		return model.WorkspaceCopy
	}
	return model.WorkspaceCopy
}

// ==================== 准备工作区 ====================

// Prepare 为指定 quest 准备工作区。
// mode 为空时自动探测。返回工作区信息。
func (wm *WorkspaceManager) Prepare(ctx context.Context, qid string, baseDir string, mode model.WorkspaceMode) (WorkspaceInfo, error) {
	qs := NewQuestStore(wm.root)
	workDir, err := qs.WorkDir(qid)
	if err != nil {
		return WorkspaceInfo{}, fmt.Errorf("创建 quest 工作目录失败: %w", err)
	} // 自动创建目录

	// 模式自动探测
	if mode == "" {
		mode = wm.DetectMode(ctx, baseDir)
	}

	info := WorkspaceInfo{
		Mode:    mode,
		Path:    workDir,
		BaseDir: baseDir,
	}

	switch mode {
	case model.WorkspaceWorktree:
		if baseDir == "" {
			return info, errors.New("worktree 模式需要 baseDir")
		}
		if !isGitRepo(ctx, baseDir) {
			// 降级为 copy
			info.DowngradedFrom = model.WorkspaceWorktree
			info.DowngradeReason = "base 目录不是 git 仓库，已自动降级为 copy 模式"
			info.Mode = model.WorkspaceCopy
			return wm.prepareCopy(ctx, qid, baseDir, info)
		}
		return wm.prepareWorktree(ctx, qid, baseDir, info)

	case model.WorkspaceCopy:
		return wm.prepareCopy(ctx, qid, baseDir, info)

	case model.WorkspaceReadOnly:
		// readonly 直接用 baseDir
		info.Path = baseDir
		return info, nil

	default:
		return info, fmt.Errorf("未知 workspace 模式: %s", mode)
	}
}

// --- worktree 模式 ---

func (wm *WorkspaceManager) prepareWorktree(ctx context.Context, qid, baseDir string, info WorkspaceInfo) (WorkspaceInfo, error) {
	// 前置检查：含 submodule 的仓库降级到 copy 模式
	// git worktree 对 submodule 的支持不稳定（旧版 git 共享 .git/modules，
	// 多 worktree 会互相干扰），保守起见降级到 copy。
	if hasSub, err := gitHasSubmodules(ctx, baseDir); err == nil && hasSub {
		info.DowngradedFrom = model.WorkspaceWorktree
		info.DowngradeReason = "仓库包含 git submodule，worktree 模式兼容性不佳，已自动降级为 copy 模式"
		info.Mode = model.WorkspaceCopy
		return wm.prepareCopy(ctx, qid, baseDir, info)
	}

	qs := NewQuestStore(wm.root)
	workDir, err := qs.WorkDir(qid)
	if err != nil {
		return info, fmt.Errorf("创建 quest 工作目录失败: %w", err)
	}

	// 先获取 base 分支和 commit
	branch, err := gitCurrentBranch(ctx, baseDir)
	if err != nil {
		return info, fmt.Errorf("获取当前分支失败: %w", err)
	}
	commit, err := gitCurrentCommit(ctx, baseDir)
	if err != nil {
		return info, fmt.Errorf("获取当前 commit 失败: %w", err)
	}
	info.BaseBranch = branch
	info.BaseCommit = commit

	// worktree add 之前要确保 workDir 不存在（git worktree add 要求目标目录不存在）
	// QuestStore.WorkDir 已经创建了目录，先删掉
	if err := os.RemoveAll(workDir); err != nil {
		return info, fmt.Errorf("清理工作目录失败: %w", err)
	}

	// 创建 worktree（基于当前 commit，新建一个分支避免污染）
	worktreeBranch := fmt.Sprintf("gloop/%s", qid)
	cmd := exec.CommandContext(ctx, "git", "worktree", "add", "-b", worktreeBranch, workDir, commit)
	cmd.Dir = baseDir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return info, fmt.Errorf("git worktree add 失败: %w: %s", err, stderr.String())
	}

	// 校验 worktree 干净：从 commit 创建的 worktree 应无未提交改动。
	// 若不干净（git 版本 quirk 或共享索引泄漏），fail-fast 让人排查，
	// 而不是让 agent 在脏环境里误改别人的改动。
	if dirty, err := gitDirty(ctx, workDir, loadGloopIgnore(workDir)); err != nil {
		return info, fmt.Errorf("校验 worktree 干净状态失败: %w", err)
	} else if dirty {
		_ = os.RemoveAll(workDir)
		return info, fmt.Errorf("worktree 创建后不干净（非预期，可能 git 索引泄漏），已清理，请排查 base 仓库状态")
	}

	// 设置 .gloop 目录（在 worktree 创建后）
	if err := setupGloopDir(workDir); err != nil {
		return info, fmt.Errorf("设置 .gloop 目录失败: %w", err)
	}

	// 把 .gloop 加入 git exclude，避免出现在 diff 里
	_ = gitAddToExclude(ctx, workDir, GloopDir+"/")

	return info, nil
}

// --- copy 模式 ---

func (wm *WorkspaceManager) prepareCopy(ctx context.Context, qid, baseDir string, info WorkspaceInfo) (WorkspaceInfo, error) {
	qs := NewQuestStore(wm.root)
	workDir, err := qs.WorkDir(qid)
	if err != nil {
		return info, fmt.Errorf("创建 quest 工作目录失败: %w", err)
	}

	if baseDir == "" {
		// 没有源目录，就用空目录（仍 git init 以支持后续 diff）
		if err := setupGloopDir(workDir); err != nil {
			return info, fmt.Errorf("设置 .gloop 目录失败: %w", err)
		}
		if err := initGitBaseline(ctx, workDir); err != nil {
			return info, fmt.Errorf("初始化 git 基线失败: %w", err)
		}
		return info, nil
	}

	// 先清空 workDir（QuestStore.WorkDir 已创建空目录）
	entries, err := os.ReadDir(workDir)
	if err == nil && len(entries) > 0 {
		if err := os.RemoveAll(workDir); err != nil {
			return info, fmt.Errorf("清理工作目录失败: %w", err)
		}
		if err := os.MkdirAll(workDir, 0o755); err != nil {
			return info, fmt.Errorf("重建工作目录失败: %w", err)
		}
	}

	// 拷贝目录
	if err := copyDir(baseDir, workDir); err != nil {
		return info, fmt.Errorf("拷贝工作目录失败: %w", err)
	}

	// 设置 .gloop 目录
	if err := setupGloopDir(workDir); err != nil {
		return info, fmt.Errorf("设置 .gloop 目录失败: %w", err)
	}

	// git-init 模式：在 work 目录 git init + commit 成基线，
	// 后续 diff/apply 复用 git 原语，精度对齐 worktree。
	// copy 模式不含 .git（shouldSkipWorkspacePath 跳过），
	// 所以这是独立于 base 仓库的本地 git。
	if err := initGitBaseline(ctx, workDir); err != nil {
		return info, fmt.Errorf("初始化 git 基线失败: %w", err)
	}
	info.BaseCommit = copyBaselineCommit

	return info, nil
}

// copyBaselineCommit 是 copy 模式 git-init 基线 commit 的占位引用。
// 真正的 commit hash 在 initGitBaseline 里生成，diff 时用 "HEAD" 对比。
const copyBaselineCommit = "HEAD"

// initGitBaseline 在 work 目录初始化 git 并把当前所有文件 commit 成基线。
// 这样 copy 模式也能用 git diff 精确计算差异，不用 diffDirs 的行数近似。
// .gloop/ 不纳入基线（通过 pathspec 排除），后续 agent 在 scratch 写文件不会出现在 diff 里。
func initGitBaseline(ctx context.Context, workDir string) error {
	// git init
	if _, err := runGit(ctx, workDir, "init", "--quiet"); err != nil {
		return fmt.Errorf("git init 失败: %w", err)
	}
	// 配置 user（本地 commit 需要）
	if _, err := runGit(ctx, workDir, "config", "user.email", "gloop@local"); err != nil {
		return fmt.Errorf("git config user.email 失败: %w", err)
	}
	if _, err := runGit(ctx, workDir, "config", "user.name", "Gloop"); err != nil {
		return fmt.Errorf("git config user.name 失败: %w", err)
	}
	// 把 .gloop 加入 exclude（diff 时 untracked 的 .gloop 文件不显示）
	_ = gitAddToExclude(ctx, workDir, GloopDir+"/")
	// git add 所有文件，然后从暂存区移除 .gloop（避免基线 commit 包含 .gloop，
	// 否则 agent 后续在 .gloop/scratch 写文件会出现在 diff 里）
	if _, err := runGit(ctx, workDir, "add", "--all", "--force"); err != nil {
		return fmt.Errorf("git add 失败: %w", err)
	}
	if _, err := runGit(ctx, workDir, "reset", "--quiet", "--", ".gloop"); err != nil {
		// reset .gloop 失败不致命（可能没有 .gloop 文件被暂存）
	}
	if _, err := runGit(ctx, workDir, "commit", "--quiet", "-m", "gloop copy baseline"); err != nil {
		return fmt.Errorf("git commit 基线失败: %w", err)
	}
	return nil
}
