package fsstore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRecommendMageCommands(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/app\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"scripts":{"test":"vitest","lint":"eslint .","build":"vite build"}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	items, err := RecommendMageCommands(dir, []AllowedCommand{{ID: "go-test"}})
	if err != nil {
		t.Fatalf("RecommendMageCommands failed: %v", err)
	}
	byID := map[string]CommandRecommendation{}
	for _, item := range items {
		byID[item.ID] = item
	}
	for _, id := range []string{"go-test", "go-vet", "go-test-race", "npm-test", "npm-run-lint", "npm-run-build"} {
		if _, ok := byID[id]; !ok {
			t.Fatalf("missing recommendation %s in %#v", id, byID)
		}
	}
	if !byID["go-test"].Present {
		t.Fatal("existing go-test should be marked present")
	}
	if byID["npm-run-lint"].Command != "npm" || len(byID["npm-run-lint"].Args) != 2 || byID["npm-run-lint"].Args[1] != "lint" {
		t.Fatalf("bad npm lint recommendation: %+v", byID["npm-run-lint"])
	}
}

func TestRecommendMageCommandsRejectsMissingDir(t *testing.T) {
	if _, err := RecommendMageCommands(filepath.Join(t.TempDir(), "missing"), nil); err == nil {
		t.Fatal("missing directory should be rejected")
	}
}
