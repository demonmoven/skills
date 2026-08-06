package fsstore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAgentContextAndPhaseSignal(t *testing.T) {
	workDir := t.TempDir()
	ctx := AgentContext{
		DataDir:         t.TempDir(),
		QuestID:         "qst_test",
		SessionID:       "warrior_0",
		Phase:           0,
		PhaseName:       "warrior",
		AdventurerID:    "adv_warrior_001",
		AdventurerClass: "warrior",
	}
	if err := WriteAgentContext(workDir, ctx); err != nil {
		t.Fatalf("WriteAgentContext failed: %v", err)
	}
	nested := filepath.Join(workDir, "sub", "dir")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	got, gotWorkDir, err := DiscoverAgentContext(nested)
	if err != nil {
		t.Fatalf("DiscoverAgentContext failed: %v", err)
	}
	if got.QuestID != ctx.QuestID || got.SessionID != ctx.SessionID || gotWorkDir != workDir {
		t.Fatalf("bad context: got=%+v workDir=%s", got, gotWorkDir)
	}

	sig := PhaseSignal{
		OK:           true,
		Message:      "done",
		PhaseEnded:   true,
		PhaseVerdict: "done",
		PhaseComment: "implemented",
	}
	if err := WritePhaseSignal(workDir, ctx.SessionID, sig); err != nil {
		t.Fatalf("WritePhaseSignal failed: %v", err)
	}
	gotSig, err := ReadPhaseSignal(workDir, ctx.SessionID)
	if err != nil {
		t.Fatalf("ReadPhaseSignal failed: %v", err)
	}
	if gotSig.PhaseVerdict != "done" || gotSig.PhaseComment != "implemented" {
		t.Fatalf("bad signal: %+v", gotSig)
	}
	if err := RemovePhaseSignal(workDir, ctx.SessionID); err != nil {
		t.Fatalf("RemovePhaseSignal failed: %v", err)
	}
	if _, err := ReadPhaseSignal(workDir, ctx.SessionID); err == nil {
		t.Fatal("signal should have been removed")
	}
}

func TestDiscoverAgentContext_FromEnvVars(t *testing.T) {
	dataDir := t.TempDir()
	workDir := t.TempDir()
	t.Setenv("GLOOP_DATA_DIR", dataDir)
	t.Setenv("GLOOP_QUEST_ID", "qst_env")
	t.Setenv("GLOOP_SESSION_ID", "mage_0")
	t.Setenv("GLOOP_PHASE", "mage_review")
	t.Setenv("GLOOP_PHASE_INDEX", "1")
	t.Setenv("GLOOP_ADVENTURER_ID", "adv_mage_001")
	t.Setenv("GLOOP_ADVENTURER_CLASS", "mage")
	t.Setenv("GLOOP_WORKSPACE_PATH", workDir)
	t.Setenv("GLOOP_SIGNAL_WORKSPACE_PATH", dataDir)

	got, gotWorkDir, err := DiscoverAgentContext(t.TempDir())
	if err != nil {
		t.Fatalf("DiscoverAgentContext failed: %v", err)
	}
	if gotWorkDir != workDir {
		t.Fatalf("workDir = %s, want %s", gotWorkDir, workDir)
	}
	if got.DataDir != dataDir || got.QuestID != "qst_env" || got.SessionID != "mage_0" {
		t.Fatalf("bad env context: %+v", got)
	}
	if got.Phase != 1 || got.PhaseName != "mage_review" || got.AdventurerClass != "mage" {
		t.Fatalf("bad phase/adventurer context: %+v", got)
	}
	if got.SignalWorkspacePath != dataDir {
		t.Fatalf("signal workspace path = %q, want %q", got.SignalWorkspacePath, dataDir)
	}
}
