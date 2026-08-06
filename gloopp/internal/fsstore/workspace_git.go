package fsstore

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

// setupGloopDir 创建 .gloop/scratch 目录结构。
func setupGloopDir(workDir string) error {
	scratchPath := filepath.Join(workDir, ScratchDir)
	return os.MkdirAll(scratchPath, 0o755)
}

// gitAddToExclude 把路径加入 git 的 exclude 列表（类似 .gitignore 但不提交）。
// 支持 worktree 模式：通过 git rev-parse --git-path 获取正确路径。
func gitAddToExclude(ctx context.Context, repoDir, pattern string) error {
	// 获取 info/exclude 的真实路径（worktree 模式下 .git 是文件）
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--git-path", "info/exclude")
	cmd.Dir = repoDir
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("git rev-parse --git-path failed: %w", err)
	}
	excludeFile := strings.TrimSpace(string(out))
	if !filepath.IsAbs(excludeFile) {
		excludeFile = filepath.Join(repoDir, excludeFile)
	}

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(excludeFile), 0o755); err != nil {
		return err
	}

	// 检查是否已经存在
	data, err := os.ReadFile(excludeFile)
	if err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.TrimSpace(line) == pattern {
				return nil // 已经存在
			}
		}
	}

	// 追加
	f, err := os.OpenFile(excludeFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(pattern + "\n")
	return err
}
func gitDiffPathspecs(baseDir string) []string {
	ignore := loadGloopIgnore(baseDir)
	if len(ignore) == 0 {
		return nil
	}
	out := []string{"--", "."}
	for _, pattern := range ignore {
		p := strings.TrimSuffix(pattern, "/")
		if p == "" {
			continue
		}
		out = append(out, ":(exclude)"+p)
		if !strings.ContainsAny(p, "*?[") {
			out = append(out, ":(exclude)"+p+"/**")
		}
	}
	return out
}

func isGitRepo(ctx context.Context, dir string) bool {
	if dir == "" {
		return false
	}
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--is-inside-work-tree")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "true"
}

func gitCurrentBranch(ctx context.Context, dir string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func gitCurrentCommit(ctx context.Context, dir string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func runGit(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s: %w", stderr.String(), err)
	}
	return stdout.String(), nil
}

func runGitWithInput(ctx context.Context, dir string, input string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(input)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s: %w", stderr.String(), err)
	}
	return stdout.String(), nil
}

func gitDirty(ctx context.Context, dir string, ignore gloopIgnore) (bool, error) {
	out, err := runGit(ctx, dir, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if len(line) < 4 {
			return true, nil
		}
		rel := strings.TrimSpace(line[3:])
		if shouldSkipWorkspacePath(path.Base(rel), rel, false, ignore) {
			continue
		}
		return true, nil
	}
	return false, nil
}

// gitHasSubmodules 检测 git 仓库是否包含子模块。
// 优先检查 .gitmodules 文件，再验证 git 配置中是否有实际的 submodule 条目，
// 避免空的 .gitmodules 文件误触发降级。
func gitHasSubmodules(ctx context.Context, dir string) (bool, error) {
	// 快速路径：.gitmodules 文件不存在就直接返回 false
	gitmodulesPath := filepath.Join(dir, ".gitmodules")
	if _, err := os.Stat(gitmodulesPath); os.IsNotExist(err) {
		return false, nil
	}

	// 文件存在，但还要确认配置里真的有 submodule
	out, err := runGit(ctx, dir, "config", "--file", ".gitmodules", "--name-only", "--get-regexp", `^submodule\..*\.path$`)
	if err != nil {
		// .gitmodules 格式异常或无匹配，保守认为没有
		return false, nil
	}
	return strings.TrimSpace(out) != "", nil
}

// gitCommitAll 暂存工作区所有改动并创建一个提交。
// - 包含未跟踪文件（git add -A）
// - 自动排除 .gloop/ 等已在 git exclude 中的路径
// - 如果没有任何改动，返回 ("", nil) 表示空提交
// - 成功返回 commit hash
func gitCommitAll(ctx context.Context, dir string, message string) (string, error) {
	// 先检查是否有任何改动（含未跟踪），没有就直接返回
	dirty, err := gitDirty(ctx, dir, nil)
	if err != nil {
		return "", fmt.Errorf("检查工作区状态失败: %w", err)
	}
	if !dirty {
		return "", nil
	}

	// 暂存所有改动（包括未跟踪文件）
	if _, err := runGit(ctx, dir, "add", "-A"); err != nil {
		return "", fmt.Errorf("git add 失败: %w", err)
	}

	// 检查暂存区是否有内容（add 之后可能因为 ignore 规则仍然为空）
	statusOut, err := runGit(ctx, dir, "diff", "--cached", "--name-only")
	if err != nil {
		return "", fmt.Errorf("检查暂存区失败: %w", err)
	}
	if strings.TrimSpace(statusOut) == "" {
		return "", nil
	}

	// 创建提交
	if _, err := runGit(ctx, dir, "commit", "-m", message); err != nil {
		return "", fmt.Errorf("git commit 失败: %w", err)
	}

	// 返回新的 commit hash
	hash, err := gitCurrentCommit(ctx, dir)
	if err != nil {
		return "", fmt.Errorf("获取新 commit hash 失败: %w", err)
	}
	return hash, nil
}
