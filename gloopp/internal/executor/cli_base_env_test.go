package executor

import (
	"strings"
	"testing"
)

// envPairValue returns the LAST value for key in a KEY=VALUE slice, matching Go
// exec semantics where a duplicate env key resolves to its final occurrence.
func envPairValue(env []string, key string) string {
	prefix := key + "="
	val := ""
	for _, kv := range env {
		if strings.HasPrefix(kv, prefix) {
			val = kv[len(prefix):]
		}
	}
	return val
}

// TestCLIExecutorBase_AgentEnvReachesSession is the regression guard for the
// qst_2606291778 failure: a CLI agent's static env (e.g. CLAUDE_CONFIG_DIR) was
// computed at registration but never reached the spawned subprocess, so claude
// fell back to the wrong config dir and returned "Not logged in".
func TestCLIExecutorBase_AgentEnvReachesSession(t *testing.T) {
	base := NewCLIExecutorBase("claude", t.TempDir(), nil)
	base.SetAgentEnv([]string{"CLAUDE_CONFIG_DIR=/home/u/.claude"})

	_, st, err := base.CreateSession("s1", SessionConfig{SystemPrompt: "system"})
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	if got := envPairValue(st.Env, "CLAUDE_CONFIG_DIR"); got != "/home/u/.claude" {
		t.Fatalf("agent env not propagated to session: CLAUDE_CONFIG_DIR=%q env=%v", got, st.Env)
	}
}

// TestCLIExecutorBase_SessionEnvOverridesAgentEnv locks the precedence contract:
// agent static env is the base layer, session runtime env (GLOOP_*) overrides
// it. Relies on Go exec's last-key-wins semantics, so session env must come
// AFTER agent env in the merged slice.
func TestCLIExecutorBase_SessionEnvOverridesAgentEnv(t *testing.T) {
	base := NewCLIExecutorBase("claude", t.TempDir(), nil)
	base.SetAgentEnv([]string{
		"CLAUDE_CONFIG_DIR=/agent/dir",
		"SHARED_KEY=agent_value",
	})

	_, st, err := base.CreateSession("s2", SessionConfig{
		SystemPrompt: "system",
		Env:          []string{"SHARED_KEY=session_value", "GLOOP_QUEST_ID=qst_x"},
	})
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	// agent-only key survives
	if got := envPairValue(st.Env, "CLAUDE_CONFIG_DIR"); got != "/agent/dir" {
		t.Fatalf("agent-only key lost: CLAUDE_CONFIG_DIR=%q", got)
	}
	// session-only key present
	if got := envPairValue(st.Env, "GLOOP_QUEST_ID"); got != "qst_x" {
		t.Fatalf("session-only key lost: GLOOP_QUEST_ID=%q", got)
	}
	// conflicting key: session wins (last occurrence)
	if got := envPairValue(st.Env, "SHARED_KEY"); got != "session_value" {
		t.Fatalf("session env should override agent env: SHARED_KEY=%q, want session_value", got)
	}
}

// TestCLIExecutorBase_NoAgentEnvKeepsSessionEnv ensures the no-agent-env path is
// a clean passthrough (no nil-deref, no accidental mutation of caller slice).
func TestCLIExecutorBase_NoAgentEnvKeepsSessionEnv(t *testing.T) {
	base := NewCLIExecutorBase("relay", t.TempDir(), nil)

	_, st, err := base.CreateSession("s3", SessionConfig{
		SystemPrompt: "system",
		Env:          []string{"GLOOP_PHASE=warrior"},
	})
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	if got := envPairValue(st.Env, "GLOOP_PHASE"); got != "warrior" {
		t.Fatalf("session env lost when no agent env: GLOOP_PHASE=%q", got)
	}
}
