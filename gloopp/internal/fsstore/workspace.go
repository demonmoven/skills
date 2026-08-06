package fsstore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

// ==================== Workspace 管理器 ====================
//
// 三种工作区模式：
//   - worktree: git worktree 隔离，支持 diff / apply / discard
//   - copy:     完整目录拷贝，支持 diff / apply / discard（文件级对比）
//   - readonly: 直接使用源目录（只读约定），不支持 diff / apply
//
// 工作区路径固定为：workspace/quests/<qid>/work/
// 由 QuestStore.WorkDir() 统一返回。
//
// .gloop/ 目录：
//   - 工作区内的系统目录，Agent 可以使用但不会被 diff / apply 追踪
//   - .gloop/scratch/：临时文件目录，Agent 可自由读写
//   - 所有 .gloop/ 下的文件都排除在 diff 和 apply 之外

const (
	// GloopDir 是工作区内的系统目录名（不会被 diff/apply 追踪）。
	GloopDir = ".gloop"
	// ScratchDir 是 Agent 可自由读写的临时目录。
	ScratchDir = ".gloop/scratch"
)

// Apply 策略常量（仅 worktree 模式生效）。
const (
	ApplyStrategyPatch          = "patch"            // 默认：git diff + git apply，快且精确
	ApplyStrategyMerge          = "merge"            // 直接 git merge，适合 base 前进较多的场景
	ApplyStrategyPatchThenMerge = "patch_then_merge" // patch 冲突时 fallback 到 merge
)

type WorkspaceInfo struct {
	Mode            model.WorkspaceMode `json:"mode"`
	Path            string              `json:"path"`
	BaseDir         string              `json:"base_dir"`
	BaseBranch      string              `json:"base_branch,omitempty"`
	BaseCommit      string              `json:"base_commit,omitempty"`
	DowngradedFrom  model.WorkspaceMode `json:"downgraded_from,omitempty"`  // 从哪个模式降级来的
	DowngradeReason string              `json:"downgrade_reason,omitempty"` // 降级原因
}

// WorkspaceDowngradeInfo 是 QuestMeta 里引用的降级信息结构。
// 放在这里是为了保持 fsstore 作为 workspace 事实源。
type WorkspaceDowngradeInfo struct {
	From   model.WorkspaceMode `json:"from"`
	Reason string              `json:"reason"`
}

type DiffFileStat struct {
	Path      string `json:"path"`
	Status    string `json:"status"` // added / modified / deleted
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
}

type DiffResult struct {
	Stat         string         `json:"stat"`
	ChangedFiles int            `json:"changed_files"`
	Additions    int            `json:"additions"`
	Deletions    int            `json:"deletions"`
	Files        []DiffFileStat `json:"files,omitempty"`
	Diff         string         `json:"diff,omitempty"`
	Source       string         `json:"source,omitempty"`
	BackupID     string         `json:"backup_id,omitempty"`
}

type ApplyBackupMeta struct {
	ID           string `json:"id"`
	QuestID      string `json:"quest_id"`
	Mode         string `json:"mode"`
	BaseDir      string `json:"base_dir"`
	BaseBranch   string `json:"base_branch,omitempty"`
	BaseCommit   string `json:"base_commit"`
	ApplyCommit  string `json:"apply_commit,omitempty"`
	WorkCommit   string `json:"work_commit,omitempty"`
	FilesChanged int    `json:"files_changed,omitempty"`
	CreatedAtMs  int64  `json:"created_at_ms"`
	BundlePath   string `json:"bundle_path,omitempty"`
	PatchPath    string `json:"patch_path,omitempty"`
	ApplyMethod  string `json:"apply_method,omitempty"` // patch / merge
}

func (wm *WorkspaceManager) ListApplyBackups(qid string) ([]*ApplyBackupMeta, error) {
	qs := NewQuestStore(wm.root)
	dir := filepath.Join(qs.dir(qid), "backups")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []*ApplyBackupMeta{}, nil
		}
		return nil, err
	}
	out := []*ApplyBackupMeta{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		meta, err := ReadJSON[ApplyBackupMeta](filepath.Join(dir, entry.Name(), "meta.json"))
		if err != nil {
			continue
		}
		out = append(out, meta)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAtMs > out[j].CreatedAtMs })
	return out, nil
}

func (wm *WorkspaceManager) GetApplyBackup(qid string, backupID string) (*ApplyBackupMeta, error) {
	if backupID == "" {
		return nil, fmt.Errorf("backup id 为空")
	}
	qs := NewQuestStore(wm.root)
	return ReadJSON[ApplyBackupMeta](filepath.Join(qs.dir(qid), "backups", backupID, "meta.json"))
}

type WorkspaceManager struct {
	root *Root
}

func NewWorkspaceManager(root *Root) *WorkspaceManager {
	return &WorkspaceManager{root: root}
}

// CommitWorkspaceChanges 把工作区所有未提交改动（含未跟踪文件）提交成一个新 commit。
// 只对 worktree 模式有效；copy / readonly 模式返回空字符串且无错误（空操作）。
// 如果没有任何改动，返回 ("", nil)。
// 成功返回新 commit 的 hash。
func (wm *WorkspaceManager) CommitWorkspaceChanges(ctx context.Context, qid string, q *QuestMeta, message string) (string, error) {
	if q.WorkspaceMode != model.WorkspaceWorktree {
		return "", nil
	}
	qs := NewQuestStore(wm.root)
	workDir, err := qs.WorkDir(qid)
	if err != nil {
		return "", fmt.Errorf("获取工作目录失败: %w", err)
	}
	return gitCommitAll(ctx, workDir, message)
}
