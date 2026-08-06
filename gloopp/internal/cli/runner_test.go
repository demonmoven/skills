package cli

import "testing"

func TestResolveDataDirUsesEnvWhenFlagEmpty(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("GLOOP_DATA_DIR", dataDir)

	got, err := resolveDataDir("")
	if err != nil {
		t.Fatalf("resolveDataDir: %v", err)
	}
	if got != dataDir {
		t.Fatalf("data dir = %q, want %q", got, dataDir)
	}
}

func TestResolveDataDirFlagOverridesEnv(t *testing.T) {
	envDir := t.TempDir()
	flagDir := t.TempDir()
	t.Setenv("GLOOP_DATA_DIR", envDir)

	got, err := resolveDataDir(flagDir)
	if err != nil {
		t.Fatalf("resolveDataDir: %v", err)
	}
	if got != flagDir {
		t.Fatalf("data dir = %q, want flag dir %q", got, flagDir)
	}
}

func TestNewRunnerUsesEnvDataDir(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("GLOOP_DATA_DIR", dataDir)

	r := newRunner("", testLogger())
	if r.dataDir != dataDir {
		t.Fatalf("runner data dir = %q, want %q", r.dataDir, dataDir)
	}
}
