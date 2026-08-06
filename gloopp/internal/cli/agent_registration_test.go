package cli

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"code.byted.org/lihuanyu.0w0/gloop/internal/events"
	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
	"code.byted.org/lihuanyu.0w0/gloop/internal/notifications"
	"code.byted.org/lihuanyu.0w0/gloop/internal/orchestrator"
)

func TestRegisterExecutors_CLIAdapters(t *testing.T) {
	dataDir := t.TempDir()
	root, err := fsstore.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	cfg := fsstore.DefaultGlobalConfig()
	eng, err := orchestrator.NewEngine(root, cfg, events.NewBus(), t.TempDir(), slog.New(slog.NewTextHandler(os.Stderr, nil)), orchestrator.WithNotifier(notifications.NewNoopNotifier()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { eng.Shutdown(2 * time.Second) })
	r := &Runner{
		root:    root,
		cfg:     cfg,
		engine:  eng,
		bus:     events.NewBus(),
		log:     slog.New(slog.NewTextHandler(os.Stderr, nil)),
		dataDir: dataDir,
	}

	binDir := t.TempDir()
	for _, name := range []string{"relay", "codex", "traex", "pi", "omp", "aiden"} {
		path := filepath.Join(binDir, name)
		if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	agents := []fsstore.AgentConfig{
		{Name: "relay_cli", Type: model.AgentTypeCLI, Command: filepath.Join(binDir, "relay"), DefaultModel: "sonnet", Enabled: true},
		{Name: "codex_cli", Type: model.AgentTypeCLI, Command: filepath.Join(binDir, "codex"), DefaultModel: "auto", Enabled: true},
		{Name: "pi_cli", Type: model.AgentTypeCLI, Command: filepath.Join(binDir, "pi"), DefaultModel: "auto", Enabled: true},
		{Name: "traex_cli", Type: model.AgentTypeCLI, Command: filepath.Join(binDir, "traex"), DefaultModel: "auto", Enabled: true},
		{Name: "omp_acp", Type: model.AgentTypeACP, Command: filepath.Join(binDir, "omp"), Args: []string{"acp"}, DefaultModel: "auto", Enabled: true},
		{Name: "aiden_cli", Type: model.AgentTypeCLI, Command: filepath.Join(binDir, "aiden"), DefaultModel: "auto", Enabled: true},
		{Name: "aiden_x_claude_cli", Type: model.AgentTypeCLI, Command: filepath.Join(binDir, "aiden"), Args: []string{"x", "claude", "--stream-json"}, DefaultModel: "auto", Enabled: true},
		{Name: "aiden_x_codex_cli", Type: model.AgentTypeCLI, Command: filepath.Join(binDir, "aiden"), Args: []string{"x", "codex", "--stream-json"}, DefaultModel: "auto", Enabled: true},
	}
	for i := range agents {
		if err := root.SaveAgent(&agents[i]); err != nil {
			t.Fatal(err)
		}
	}

	if err := r.registerExecutors(t.TempDir()); err != nil {
		t.Fatalf("registerExecutors returned error: %v", err)
	}

	if !eng.HasExecutor("relay_cli") {
		t.Fatal("relay_cli should be registered")
	}
	if !eng.HasExecutor("codex_cli") {
		t.Fatal("codex_cli should be registered")
	}
	if !eng.HasExecutor("pi_cli") {
		t.Fatal("pi_cli should be registered")
	}
	if !eng.HasExecutor("omp_acp") {
		t.Fatal("omp_acp should be registered")
	}
	if !eng.HasExecutor("aiden_cli") {
		t.Fatal("aiden_cli should be registered")
	}
	if !eng.HasExecutor("aiden_x_claude_cli") {
		t.Fatal("aiden_x_claude_cli should be registered")
	}
	if !eng.HasExecutor("aiden_x_codex_cli") {
		t.Fatal("aiden_x_codex_cli should be registered")
	}
	if !eng.HasExecutor("traex_cli") {
		t.Fatal("traex_cli should be registered")
	}
}
