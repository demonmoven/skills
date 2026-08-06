package fsstore

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

func TestInitDefaultFiles_Agents(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	probe := AgentProbe{
		TraeX:  "/usr/bin/traex",
		Relay:  "/usr/bin/relay",
		Pi:     "/usr/bin/pi",
		Codex:  "/usr/bin/codex",
		Aiden:  "/usr/bin/aiden",
		Hermes: "/usr/bin/hermes",
	}
	if _, err := root.InitDefaultFiles(probe, t.TempDir()); err != nil {
		t.Fatal(err)
	}
	cases := map[string]struct {
		ptype   model.AgentType
		command string
		enabled bool
	}{
		"traex":          {model.AgentTypeCLI, "/usr/bin/traex", false},
		"relay":          {model.AgentTypeCLI, "/usr/bin/relay", false},
		"pi":             {model.AgentTypeCLI, "/usr/bin/pi", false},
		"codex":          {model.AgentTypeCLI, "/usr/bin/codex", false},
		"aiden_x_claude": {model.AgentTypeCLI, "/usr/bin/aiden", false},
		"aiden_x_codex":  {model.AgentTypeCLI, "/usr/bin/aiden", false},
		"hermes":         {model.AgentTypeCLI, "/usr/bin/hermes", false},
	}
	for name, want := range cases {
		got, err := root.GetAgent(name)
		if err != nil {
			t.Fatalf("agent %s missing: %v", name, err)
		}
		if got.Type != want.ptype || got.Command != want.command || got.Enabled != want.enabled {
			t.Fatalf("agent %s = %+v, want type=%s command=%s enabled=%v", name, got, want.ptype, want.command, want.enabled)
		}
		if !got.Official {
			t.Fatalf("agent %s should be marked official: %+v", name, got)
		}
		if name == "relay" {
			if got.DefaultModel != relayDefaultModel {
				t.Fatalf("relay default_model = %q, want %s", got.DefaultModel, relayDefaultModel)
			}
		}
	}

	adventurers, err := root.ListAdventurers()
	if err != nil {
		t.Fatalf("list adventurers: %v", err)
	}
	if len(adventurers) != 0 {
		t.Fatalf("init should not create default adventurers, got %+v", adventurers)
	}
}

func TestAgentConfigEffectiveEnv(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, "agent.env")
	if err := os.WriteFile(envPath, []byte("export FOO=from_file\nBAR=\"quoted\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	agent := &AgentConfig{
		Name:     "custom",
		EnvFiles: []string{envPath},
		Env:      map[string]string{"FOO": "override"},
	}
	got, err := agent.EffectiveEnv()
	if err != nil {
		t.Fatal(err)
	}
	if got["FOO"] != "override" || got["BAR"] != "quoted" {
		t.Fatalf("EffectiveEnv = %#v", got)
	}
}

func TestAgentPackagePreservesExecutionFields(t *testing.T) {
	supportsReadOnly := true
	pkg := sanitizeAgents([]*AgentConfig{{
		Name:                "traex",
		Type:                model.AgentTypeCLI,
		Command:             "traex",
		DefaultModel:        "auto",
		DefaultAllowedTools: []string{"Read", "Bash"},
		SupportsReadOnly:    &supportsReadOnly,
		ExecutionTraits:     []string{"jsonl", "stateful_resume"},
		Enabled:             true,
	}})
	if len(pkg) != 1 {
		t.Fatalf("package agents = %#v", pkg)
	}
	item := pkg[0]
	if !reflect.DeepEqual(item.DefaultAllowedTools, []string{"Read", "Bash"}) {
		t.Fatalf("default_allowed_tools = %#v", item.DefaultAllowedTools)
	}
	if item.SupportsReadOnly == nil || *item.SupportsReadOnly != true {
		t.Fatalf("supports_readonly = %#v", item.SupportsReadOnly)
	}
	if !reflect.DeepEqual(item.ExecutionTraits, []string{"jsonl", "stateful_resume"}) {
		t.Fatalf("execution_traits = %#v", item.ExecutionTraits)
	}
}

func TestInitDefaultFiles_RepairsLegacyClaudeACPArgs(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := root.SaveAgent(&AgentConfig{
		Name:         "claude",
		Type:         model.AgentTypeACP,
		Command:      "npx",
		Args:         append([]string(nil), legacyClaudeACPArgs...),
		DefaultModel: "claude-sonnet-4-20250514",
		Enabled:      true,
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := root.InitDefaultFiles(AgentProbe{}, t.TempDir()); err != nil {
		t.Fatal(err)
	}
	got, err := root.GetAgent("claude")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Args, claudeACPArgs) {
		t.Fatalf("claude args = %#v, want %#v", got.Args, claudeACPArgs)
	}
	if got.DefaultModel != "claude-sonnet-4-20250514" || !got.Enabled {
		t.Fatalf("repair should preserve user fields, got %+v", got)
	}
}

func TestInitDefaultFiles_RepairsLegacyRelayDefaultModel(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := root.SaveAgent(&AgentConfig{
		Name:         "relay",
		Type:         model.AgentTypeCLI,
		Command:      "/usr/bin/relay",
		Args:         []string{"-p"},
		DefaultModel: "auto",
		Enabled:      true,
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := root.InitDefaultFiles(AgentProbe{}, t.TempDir()); err != nil {
		t.Fatal(err)
	}
	got, err := root.GetAgent("relay")
	if err != nil {
		t.Fatal(err)
	}
	if got.DefaultModel != relayDefaultModel {
		t.Fatalf("relay default_model = %q, want %s", got.DefaultModel, relayDefaultModel)
	}
	if !got.Enabled || got.Command != "/usr/bin/relay" {
		t.Fatalf("repair should preserve user fields, got %+v", got)
	}
}

func TestDiscoverClaudeEnvParsesShellAssignments(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	zshrc := filepath.Join(os.Getenv("HOME"), ".zshrc")
	raw := []byte(strings.Join([]string{
		"alias ll='ls -la'",
		"export ANTHROPIC_BASE_URL=\"https://example.test\"",
		"export ANTHROPIC_AUTH_TOKEN=secret",
		"export ANTHROPIC_MODEL=\"ark/seed-code-0611\"",
		"export ANTHROPIC_DEFAULT_OPUS_MODEL=\"ark/seed-code-0611\"",
		"export ANTHROPIC_DEFAULT_SONNET_MODEL=\"ark/seed-code-0611\"",
		"export ANTHROPIC_DEFAULT_HAIKU_MODEL=\"ark/seed-code-0611\"",
		"export CLAUDE_CODE_SUBAGENT_MODEL=\"ark/seed-code-0611\"",
		"export CLAUDE_CODE_AUTO_COMPACT_WINDOW=\"140000\"",
		"export CLAUDE_CODE_ATTRIBUTION_HEADER=0",
		"export PATH=/tmp/bin",
	}, "\n"))
	if err := os.WriteFile(zshrc, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := DiscoverClaudeEnv()
	if err != nil {
		t.Fatal(err)
	}
	if got["ANTHROPIC_BASE_URL"] != "https://example.test" || got["ANTHROPIC_AUTH_TOKEN"] != "secret" {
		t.Fatalf("DiscoverClaudeEnv = %#v", got)
	}
	for _, key := range []string{
		"ANTHROPIC_MODEL",
		"ANTHROPIC_DEFAULT_OPUS_MODEL",
		"ANTHROPIC_DEFAULT_SONNET_MODEL",
		"ANTHROPIC_DEFAULT_HAIKU_MODEL",
		"CLAUDE_CODE_SUBAGENT_MODEL",
		"CLAUDE_CODE_AUTO_COMPACT_WINDOW",
		"CLAUDE_CODE_ATTRIBUTION_HEADER",
	} {
		if _, ok := got[key]; !ok {
			t.Fatalf("DiscoverClaudeEnv missing %s: %#v", key, got)
		}
	}
	if _, ok := got["PATH"]; ok {
		t.Fatalf("DiscoverClaudeEnv should not collect PATH: %#v", got)
	}
}
