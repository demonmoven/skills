package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"code.byted.org/lihuanyu.0w0/gloop/internal/version"
)

func TestStableShim_NoShimFallsBack(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	self := "/usr/local/bin/some-random-gloop"
	got := stableShim(self)
	if got != self {
		t.Fatalf("expected fallback to self %q when shim missing, got %q", self, got)
	}
}

func TestStableShim_PrefersShimWhenPresent(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shim path differs on windows")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	shimDir := filepath.Join(home, "."+version.RepoName, "bin")
	if err := os.MkdirAll(shimDir, 0o755); err != nil {
		t.Fatalf("mkdir shim dir: %v", err)
	}
	shim := filepath.Join(shimDir, version.RepoName)
	if err := os.WriteFile(shim, []byte("#!/bin/sh\necho stub\n"), 0o755); err != nil {
		t.Fatalf("write shim: %v", err)
	}

	got := stableShim("/elsewhere/gloop")
	if got != shim {
		t.Fatalf("expected stable shim %q, got %q", shim, got)
	}
}

func TestBuildLaunchdPlist_UsesStablePath(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("launchd is darwin only")
	}
	home, _ := os.UserHomeDir()
	stable := filepath.Join(home, "."+version.RepoName, "bin", "gloop")
	plist := buildLaunchdPlist("com.gloop.start", []string{stable, "start", "--no-detach", "--no-open", "--no-autostart"}, "/tmp/dd")
	if !contains(plist, stable) {
		t.Fatalf("plist does not embed stable shim path %q\n%s", stable, plist)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && indexOf(s, substr) >= 0
}

func indexOf(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
