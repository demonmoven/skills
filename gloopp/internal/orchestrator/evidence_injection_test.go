package orchestrator

import (
	"os"
	"path/filepath"
	"testing"
)

// TestAutoEvidenceCommands_GoPriority 互斥项目类型判定：
// go.mod 存在即 Go 项目，即使同目录有 package.json 也不跑 npm 命令。
// 回归：Go 仓库常带 eslint/prettier 的 package.json，旧逻辑会同时触发
// npm-build-check（npm run build），在 30s L0 预算内超时/失败并淹没 Go 证据。
func TestAutoEvidenceCommands_GoPriority(t *testing.T) {
	dir := t.TempDir()
	mustTouch(t, filepath.Join(dir, "go.mod"))
	mustTouch(t, filepath.Join(dir, "package.json")) // Go 仓库常带

	var e Engine
	cmds := e.autoEvidenceCommands(dir)
	if got := contains(cmds, "go-build"); !got {
		t.Fatalf("go project should run go-build, got %v", cmds)
	}
	if got := contains(cmds, "npm-build-check"); got {
		t.Fatalf("go project should NOT run npm-build-check, got %v", cmds)
	}
	if got := contains(cmds, "npm-audit"); got {
		t.Fatalf("go project should NOT run npm-audit, got %v", cmds)
	}
}

func TestAutoEvidenceCommands_PureJS(t *testing.T) {
	dir := t.TempDir()
	mustTouch(t, filepath.Join(dir, "package.json"))

	var e Engine
	cmds := e.autoEvidenceCommands(dir)
	if got := contains(cmds, "npm-build-check"); !got {
		t.Fatalf("pure JS project should run npm-build-check, got %v", cmds)
	}
	if got := contains(cmds, "npm-audit"); !got {
		t.Fatalf("pure JS project should run npm-audit, got %v", cmds)
	}
	if got := contains(cmds, "go-build"); got {
		t.Fatalf("pure JS project should NOT run go-build, got %v", cmds)
	}
}

func TestAutoEvidenceCommands_GitAlwaysAndEmpty(t *testing.T) {
	// 空目录：仍跑 git 证据（失败也没关系），无语言命令
	dir := t.TempDir()
	var e Engine
	cmds := e.autoEvidenceCommands(dir)
	if got := contains(cmds, "git-diff-stat"); !got {
		t.Fatalf("git-diff-stat should always run, got %v", cmds)
	}
	if got := contains(cmds, "git-diff-name-only"); !got {
		t.Fatalf("git-diff-name-only should always run, got %v", cmds)
	}
	if len(cmds) != 2 {
		t.Fatalf("empty dir should only have 2 git cmds, got %v", cmds)
	}

	// 空路径
	if cmds := e.autoEvidenceCommands(""); cmds != nil {
		t.Fatalf("empty path should return nil, got %v", cmds)
	}
}

func mustTouch(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}
