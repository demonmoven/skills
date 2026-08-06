package cli

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"code.byted.org/lihuanyu.0w0/gloop/internal/auth"
	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
)

func TestRunDashboardCmdPrintsURLWhenRunning(t *testing.T) {
	t.Setenv("GLOOP_DASHBOARD_EXTERNAL_HOST", "10.4.4.239")
	dataDir := t.TempDir()
	tok := &auth.TokenStore{
		Token:     "test-token",
		CreatedAt: time.Now(),
		Port:      37317,
		BindHost:  "0.0.0.0",
	}
	if err := tok.Save(dataDir); err != nil {
		t.Fatalf("save token: %v", err)
	}
	if err := writePidFile(dataDir, os.Getpid()); err != nil {
		t.Fatalf("write pidfile: %v", err)
	}

	stdout, stderr, code := captureCLIOutput(t, func() int {
		return runDashboardCmd(context.Background(), testLogger(), []string{"--data-dir", dataDir, "--print"})
	})
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %q", code, stderr)
	}
	want := "http://10.4.4.239:37317/dashboard?t=test-token\n"
	if stdout != want {
		t.Fatalf("stdout = %q, want %q", stdout, want)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
}

func TestRunDashboardCmdFailsWhenNotRunning(t *testing.T) {
	dataDir := t.TempDir()
	tok := &auth.TokenStore{
		Token:     "test-token",
		CreatedAt: time.Now(),
		Port:      37317,
		BindHost:  "127.0.0.1",
	}
	if err := tok.Save(dataDir); err != nil {
		t.Fatalf("save token: %v", err)
	}
	if err := os.WriteFile(pidFilePath(dataDir), []byte(strconv.Itoa(os.Getpid()+1000000)), 0o644); err != nil {
		t.Fatalf("write stale pidfile: %v", err)
	}

	stdout, stderr, code := captureCLIOutput(t, func() int {
		return runDashboardCmd(context.Background(), testLogger(), []string{"--data-dir", dataDir, "--print"})
	})
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "Gloop 未运行") {
		t.Fatalf("stderr = %q, want not-running hint", stderr)
	}
}

func TestLogsQuestUsesShortIDPath(t *testing.T) {
	dataDir := t.TempDir()
	qid := "qst_2606225000"
	dir := filepath.Join(dataDir, fsstore.SubdirQuests, qid)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "events.jsonl"), []byte("{\"type\":\"quest.created\"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := captureCLIOutput(t, func() int {
		return runLogsCmd(context.Background(), testLogger(), []string{"--data-dir", dataDir, "--quest", "2606225000"})
	})
	if code != 0 || stderr != "" {
		t.Fatalf("code=%d stderr=%q", code, stderr)
	}
	if !strings.Contains(stdout, "== quest 2606225000 ==") || !strings.Contains(stdout, "quest.created") {
		t.Fatalf("unexpected stdout: %q", stdout)
	}
}

func TestLogsSchedulerScope(t *testing.T) {
	dataDir := t.TempDir()
	path := filepath.Join(dataDir, fsstore.SubdirWorkspace, "scheduler", "events.jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{\"kind\":\"tick\"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := captureCLIOutput(t, func() int {
		return runLogsCmd(context.Background(), testLogger(), []string{"--data-dir", dataDir, "--scheduler"})
	})
	if code != 0 || stderr != "" {
		t.Fatalf("code=%d stderr=%q", code, stderr)
	}
	if !strings.Contains(stdout, "== scheduler ==") || !strings.Contains(stdout, "tick") {
		t.Fatalf("unexpected stdout: %q", stdout)
	}
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func captureCLIOutput(t *testing.T, fn func() int) (string, string, int) {
	t.Helper()

	oldStdout := os.Stdout
	oldStderr := os.Stderr
	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stdout: %v", err)
	}
	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stderr: %v", err)
	}
	os.Stdout = stdoutW
	os.Stderr = stderrW
	defer func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	}()

	code := fn()
	_ = stdoutW.Close()
	_ = stderrW.Close()
	stdout, err := io.ReadAll(stdoutR)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	stderr, err := io.ReadAll(stderrR)
	if err != nil {
		t.Fatalf("read stderr: %v", err)
	}

	return string(stdout), string(stderr), code
}
