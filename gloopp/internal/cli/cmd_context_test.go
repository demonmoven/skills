package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
)

func TestRunContextExportOKF(t *testing.T) {
	dataDir := t.TempDir()
	root, err := fsstore.Open(dataDir)
	if err != nil {
		t.Fatalf("open root: %v", err)
	}
	if err := root.WriteContextSummary("hello summary"); err != nil {
		t.Fatalf("write summary: %v", err)
	}
	if _, err := root.WriteContextDim("workspace", "hello context body"); err != nil {
		t.Fatalf("write dim: %v", err)
	}
	outDir := filepath.Join(t.TempDir(), "bundle")

	stdout, stderr, code := captureCLIOutput(t, func() int {
		return runContextExport(context.Background(), testLogger(), []string{
			"--data-dir", dataDir,
			"--format", "okf",
			"--out", outDir,
		})
	})
	if code != 0 {
		t.Fatalf("exit code = %d, stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, "Knowledge bundle 已导出") {
		t.Fatalf("stdout missing export message: %s", stdout)
	}
	raw, err := os.ReadFile(filepath.Join(outDir, "dimensions", "workspace.md"))
	if err != nil {
		t.Fatalf("read exported dim: %v", err)
	}
	text := string(raw)
	if !strings.Contains(text, "gloop_context_dimension") || !strings.Contains(text, "hello context body") {
		t.Fatalf("exported dim missing OKF content:\n%s", text)
	}
}

func TestRunContextGetReadsFileModeBlock(t *testing.T) {
	workDir := t.TempDir()
	ctxDir := filepath.Join(workDir, fsstore.GloopDir, "context")
	if err := os.MkdirAll(ctxDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ctxDir, "review_history.md"), []byte("---\nname: review_history\n---\n\nbody"), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, code := captureCLIOutput(t, func() int {
		return runContextGet(context.Background(), testLogger(), []string{
			"--work-dir", workDir,
			"review_history",
		})
	})
	if code != 0 {
		t.Fatalf("exit code = %d, stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, "name: review_history") || !strings.Contains(stdout, "body") {
		t.Fatalf("stdout missing context block: %s", stdout)
	}
}

func TestRunContextGetJSON(t *testing.T) {
	workDir := t.TempDir()
	ctxDir := filepath.Join(workDir, fsstore.GloopDir, "context")
	if err := os.MkdirAll(ctxDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ctxDir, "odd_name.md"), []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, code := captureCLIOutput(t, func() int {
		return runContextGet(context.Background(), testLogger(), []string{
			"--work-dir", workDir,
			"--json",
			"odd/name",
		})
	})
	if code != 0 {
		t.Fatalf("exit code = %d, stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, `"ok":true`) || !strings.Contains(stdout, `"block":"odd/name"`) || !strings.Contains(stdout, `"content":"content"`) {
		t.Fatalf("bad json stdout: %s", stdout)
	}
}
