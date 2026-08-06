package orchestrator

import (
	"errors"
	"os/exec"
	"reflect"
	"runtime"
	"strings"
	"syscall"
	"testing"
)

func TestExitCodeFromError_Nil(t *testing.T) {
	if code := exitCodeFromError(nil); code != 0 {
		t.Errorf("nil err should give 0, got %d", code)
	}
}

func TestExitCodeFromError_GenericError(t *testing.T) {
	if code := exitCodeFromError(errors.New("boom")); code != -1 {
		t.Errorf("generic err should give -1, got %d", code)
	}
}

func TestExitCodeFromError_ExitError(t *testing.T) {
	// 使用真实的子进程来构造 ExitError
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}
	cmd := exec.Command("sh", "-c", "exit 42")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected non-zero exit")
	}
	if code := exitCodeFromError(err); code != 42 {
		t.Errorf("ExitError(42) code = %d, want 42", code)
	}
	// ExitError 但 Sys() 不是 syscall.WaitStatus 的极端情况——在真实平台上基本遇不到，
	// 这里只确保分支能覆盖到：构造一个带 Sys()=非 WaitStatus 的 ExitError 太 hacky，忽略。
	_ = syscall.WaitStatus(0)
}

func TestTruncateMiddle(t *testing.T) {
	// limit <= 0 返回空
	if s := truncateMiddle("hello", 0); s != "" {
		t.Errorf("limit=0 should be empty, got %q", s)
	}
	if s := truncateMiddle("hello", -1); s != "" {
		t.Errorf("limit=-1 should be empty, got %q", s)
	}
	// 小于等于 limit：原样返回
	in := "hello"
	if s := truncateMiddle(in, 10); s != in {
		t.Errorf("short string should be kept: got %q", s)
	}
	// 超出：截断，含 marker
	long := strings.Repeat("a", 1000)
	s := truncateMiddle(long, 100)
	if !strings.Contains(s, "...[truncated]...") {
		t.Errorf("truncated string should contain marker, got %d chars", len(s))
	}
	if len([]rune(s)) > 100+len("\n...[truncated]...\n") {
		t.Errorf("output too long: got %d runes", len([]rune(s)))
	}
	// 前后保留前缀+后缀
	prefix := strings.Repeat("P", 100)
	suffix := strings.Repeat("S", 100)
	mid := strings.Repeat("x", 500)
	full := prefix + mid + suffix
	limit := 100
	out := truncateMiddle(full, limit)
	half := limit / 2
	if !strings.HasPrefix(out, prefix[:half]) {
		t.Errorf("should preserve prefix portion: got first 30=%q", out[:30])
	}
	expectedSuffix := (prefix + mid + suffix)[len(full)-half:]
	if !strings.HasSuffix(out, expectedSuffix) {
		t.Errorf("should preserve suffix portion: last 30=%q, expected last 30=%q", out[len(out)-30:], expectedSuffix[len(expectedSuffix)-30:])
	}
}

func TestFilteredEnv(t *testing.T) {
	in := []string{
		"HOME=/root",
		"PATH=/usr/bin",
		"GLOOP_QUEST_ID=q1",
		"GLOOP_PHASE=warrior",
		"USER=gloop",
		"GLOOP_SIGNAL_WORKSPACE_PATH=/tmp/x",
	}
	out := filteredEnv(in)
	want := []string{"HOME=/root", "PATH=/usr/bin", "USER=gloop"}
	if !reflect.DeepEqual(out, want) {
		t.Errorf("filteredEnv = %+v, want %+v", out, want)
	}
	// 没有 GLOOP_ 变量：原量返回
	noGloop := []string{"HOME=/", "PATH=/bin"}
	o2 := filteredEnv(noGloop)
	if !reflect.DeepEqual(o2, noGloop) {
		t.Errorf("filteredEnv with no gloop = %+v, want same", o2)
	}
	// 全部 GLOOP：返回空
	allGloop := []string{"GLOOP_A=1", "GLOOP_B=2"}
	o3 := filteredEnv(allGloop)
	if len(o3) != 0 {
		t.Errorf("all-gloop should be empty, got %+v", o3)
	}
}
