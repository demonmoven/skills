package connectors

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"log/slog"
)

// GitConfig 控制 GitConnector 的行为。
//
// 安全约束：
//   - Enabled=false（默认）时，GitConnector 完全 no-op
//   - 即使启用，也只提交到 quest 专属分支（gloop/<shortid>），永远不推 BaseBranch
//   - OpenMR 目前是占位（只记录日志），不真的调用任何 MR API
type GitConfig struct {
	Enabled     bool   // 必须显式设 true 才启用
	Remote      string // 默认 origin
	BaseBranch  string // 默认 main；只读参考，永不推送
	AuthorName  string // commit 作者名（空=用仓库默认）
	AuthorEmail string // commit 作者邮箱
	Push        bool   // 是否 push 到 remote（false=只本地 commit）
}

// GitConnector 在 quest apply 后把工作区改动提交到 quest 专属分支。
type GitConnector struct {
	cfg GitConfig
	log *slog.Logger
}

// NewGitConnector 创建 GitConnector。未启用时 Enabled() 返回 false。
func NewGitConnector(cfg GitConfig, log *slog.Logger) *GitConnector {
	if log == nil {
		log = slog.Default()
	}
	if cfg.Remote == "" {
		cfg.Remote = "origin"
	}
	if cfg.BaseBranch == "" {
		cfg.BaseBranch = "main"
	}
	return &GitConnector{cfg: cfg, log: log}
}

func (g *GitConnector) Name() string { return "git" }

func (g *GitConnector) Enabled() bool { return g.cfg.Enabled }

// HandleApplied 把 quest 工作区改动提交到 gloop/<shortid> 分支。
func (g *GitConnector) HandleApplied(ctx context.Context, ev AppliedEvent) error {
	if ev.WorkspacePath == "" {
		return fmt.Errorf("quest 无工作区路径，git connector 跳过")
	}
	if !isGitRepo(ev.WorkspacePath) {
		g.log.Info("git connector: 工作区非 git 仓库，跳过", "qid", ev.QuestID, "path", ev.WorkspacePath)
		return nil
	}

	branch := questBranch(ev.ShortID, ev.QuestID)
	g.log.Info("git connector: 提交到 quest 分支", "qid", ev.QuestID, "branch", branch, "path", ev.WorkspacePath)

	steps := [][]string{
		{"git", "add", "-A"},
		{"git", "commit", "-m", commitMessage(ev)},
	}
	// 切到 quest 专属分支（不存在则创建）。放在 commit 前会丢已暂存内容，
	// 所以先 commit 到当前分支再 cherry-pick 风险高——这里用最稳的：
	// 创建/切换分支（带改动一起带过去），再 commit。
	// 修正顺序：
	preSteps := [][]string{
		{"git", "checkout", "-b", branch},
	}
	// checkout -b 失败（分支已存在）则直接 checkout
	if err := runGit(ctx, ev.WorkspacePath, preSteps[0], g.log); err != nil {
		if err := runGit(ctx, ev.WorkspacePath, []string{"git", "checkout", branch}, g.log); err != nil {
			return fmt.Errorf("切到 quest 分支失败: %w", err)
		}
	}

	for _, step := range steps {
		if err := runGit(ctx, ev.WorkspacePath, step, g.log); err != nil {
			// commit 失败可能是"无改动"，不视为错误
			if strings.Contains(err.Error(), "nothing to commit") || strings.Contains(err.Error(), "no changes") {
				g.log.Info("git connector: 无改动可提交", "qid", ev.QuestID)
				return nil
			}
			return fmt.Errorf("git %s 失败: %w", step[1], err)
		}
	}

	if g.cfg.Push {
		if err := runGit(ctx, ev.WorkspacePath, []string{"git", "push", "-u", g.cfg.Remote, branch}, g.log); err != nil {
			return fmt.Errorf("git push 失败: %w", err)
		}
		g.log.Info("git connector: 已推送", "qid", ev.QuestID, "remote", g.cfg.Remote, "branch", branch)
	} else {
		g.log.Debug("git connector: Push=false，仅本地提交", "qid", ev.QuestID)
	}

	return nil
}

func questBranch(shortID, qid string) string {
	if shortID != "" {
		return "gloop/" + shortID
	}
	// 兜底用 qid 前 8 字符
	if len(qid) > 8 {
		return "gloop/" + qid[:8]
	}
	return "gloop/" + qid
}

func commitMessage(ev AppliedEvent) string {
	title := ev.Title
	if len([]rune(title)) > 60 {
		title = string([]rune(title)[:60])
	}
	return fmt.Sprintf("gloop: %s\n\nquest: %s\nfiles_changed: %d", title, ev.QuestID, ev.FilesChanged)
}

func isGitRepo(dir string) bool {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--is-inside-work-tree")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "true"
}

func runGit(ctx context.Context, dir string, args []string, log *slog.Logger) error {
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Debug("git 命令", "args", args, "dir", dir, "out", string(out), "err", err)
		return fmt.Errorf("%s: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
