package cli

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

func setupAgentCLITest(t *testing.T) (dataDir, workDir string, q *fsstore.QuestMeta) {
	t.Helper()
	dataDir = t.TempDir()
	root, err := fsstore.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	qs := fsstore.NewQuestStore(root)
	q = &fsstore.QuestMeta{
		ID:             "qst_agent_cli",
		ShortID:        "agent_cli",
		Query:          "test query",
		Type:           model.QuestTypeExecute,
		Status:         model.QuestStatusRunning,
		WorkspaceMode:  model.WorkspaceCopy,
		BaseWorkingDir: t.TempDir(),
		WarriorID:      "adv_warrior_001",
		MageID:         "adv_mage_001",
		MaxRework:      2,
		CreatedBy:      "user",
	}
	if err := qs.CreateQuest(q); err != nil {
		t.Fatal(err)
	}
	workDir, err = qs.WorkDir(q.ID)
	if err != nil {
		t.Fatal(err)
	}
	q.WorkspacePath = workDir
	if err := qs.SaveQuest(q); err != nil {
		t.Fatal(err)
	}
	if err := fsstore.WriteAgentContext(workDir, fsstore.AgentContext{
		DataDir:         dataDir,
		QuestID:         q.ID,
		SessionID:       "warrior_0",
		Phase:           0,
		PhaseName:       "warrior",
		AdventurerID:    q.WarriorID,
		AdventurerClass: "warrior",
	}); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GLOOP_CONTEXT", filepath.Join(workDir, fsstore.AgentContextRel))
	return dataDir, workDir, q
}

func TestAgentCLIQuestInfoAndSignals(t *testing.T) {
	dataDir, workDir, q := setupAgentCLITest(t)
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))

	stdout, stderr, code := captureStdIO(t, func() int {
		return runQuestInfo(t.Context(), log, nil)
	})
	if code != 0 {
		t.Fatalf("quest info failed: code=%d stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, `"id":"qst_agent_cli"`) && !strings.Contains(stdout, `"id": "qst_agent_cli"`) {
		t.Fatalf("quest info missing id: %s", stdout)
	}

	stdout, stderr, code = captureStdIO(t, func() int {
		return runPhaseDone(log, []string{"--summary", "done summary"})
	})
	if code != 0 {
		t.Fatalf("phase done failed: code=%d stdout=%s stderr=%s", code, stdout, stderr)
	}
	sig, err := fsstore.ReadPhaseSignal(workDir, "warrior_0")
	if err != nil {
		t.Fatalf("ReadPhaseSignal failed: %v", err)
	}
	if sig.PhaseVerdict != "done" || sig.PhaseComment != "done summary" {
		t.Fatalf("bad phase signal: %+v", sig)
	}

	if err := fsstore.WriteAgentContext(workDir, fsstore.AgentContext{
		DataDir:         dataDir,
		QuestID:         q.ID,
		SessionID:       "mage_0",
		Phase:           1,
		PhaseName:       "mage_review",
		AdventurerID:    q.MageID,
		AdventurerClass: "mage",
	}); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GLOOP_CONTEXT", filepath.Join(workDir, fsstore.AgentContextRel))

	stdout, stderr, code = captureStdIO(t, func() int {
		return Run(t.Context(), []string{"gloop", "review", "pass", "--comment", "contract ok", "--score", "9"}, nil)
	})
	if code != 0 {
		t.Fatalf("review pass failed: code=%d stdout=%s stderr=%s", code, stdout, stderr)
	}
	sig, err = fsstore.ReadPhaseSignal(workDir, "mage_0")
	if err != nil {
		t.Fatalf("ReadPhaseSignal review failed: %v", err)
	}
	if sig.PhaseVerdict != "pass" || sig.PhaseComment != "contract ok" || sig.PhaseScore != 9 {
		t.Fatalf("bad review signal: %+v", sig)
	}
}

func TestAgentCLIPermissionDenied(t *testing.T) {
	dataDir, workDir, q := setupAgentCLITest(t)
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// v0.3: class restrictions removed; test phase done works for any class
	stdout, _, code := captureStdIO(t, func() int {
		return runPhaseDone(log, []string{"--summary", "testing unified access"})
	})
	if code != 0 {
		t.Fatalf("phase done should succeed for any class: stdout=%s code=%d", stdout, code)
	}

	if err := fsstore.WriteAgentContext(workDir, fsstore.AgentContext{
		DataDir:         dataDir,
		QuestID:         q.ID,
		SessionID:       "mage_0",
		Phase:           1,
		PhaseName:       "mage_review",
		AdventurerID:    q.MageID,
		AdventurerClass: "mage",
	}); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GLOOP_CONTEXT", filepath.Join(workDir, fsstore.AgentContextRel))

	stdout, _, code = captureStdIO(t, func() int {
		return runPhaseDone(log, []string{"--summary", "also allowed now"})
	})
	// v0.3: phase done is no longer restricted by class; mage can also use it
	if code != 0 {
		t.Fatalf("phase done should succeed for any class (v0.3 unified): stdout=%s code=%d", stdout, code)
	}
}

func TestAgentCLINoteAddAndList(t *testing.T) {
	_, _, _ = setupAgentCLITest(t)
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))

	stdout, stderr, code := captureStdIO(t, func() int {
		return runNoteAdd(log, []string{"--tag", "risk", "remember", "this"})
	})
	if code != 0 {
		t.Fatalf("note add failed: code=%d stdout=%s stderr=%s", code, stdout, stderr)
	}
	var addResp map[string]any
	if err := json.Unmarshal([]byte(stdout), &addResp); err != nil || addResp["ok"] != true {
		t.Fatalf("bad note add response: %s err=%v", stdout, err)
	}

	stdout, stderr, code = captureStdIO(t, func() int {
		return runNoteList(log, []string{"--limit", "10"})
	})
	if code != 0 {
		t.Fatalf("note list failed: code=%d stdout=%s stderr=%s", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "remember this") || !strings.Contains(stdout, "risk") {
		t.Fatalf("note list missing note: %s", stdout)
	}
}

func TestAgentCLICommandRunMageOnly(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture uses sh")
	}
	dataDir, workDir, q := setupAgentCLITest(t)
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))

	root, err := fsstore.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := root.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	cfg.CommandAllowlist = []fsstore.AllowedCommand{{
		ID:        "fixture-pass",
		Command:   "sh",
		Args:      []string{"-c", "printf ok"},
		TimeoutMs: 30_000,
	}}
	if err := root.SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}

	stdout, _, code := captureStdIO(t, func() int {
		return runCommandRun(t.Context(), log, []string{"fixture-pass"})
	})
	if code == 0 || !strings.Contains(stdout, "permission_denied") {
		t.Fatalf("warrior command run should be denied: code=%d stdout=%s", code, stdout)
	}

	if err := fsstore.WriteAgentContext(workDir, fsstore.AgentContext{
		DataDir:         dataDir,
		QuestID:         q.ID,
		SessionID:       "mage_0",
		Phase:           1,
		PhaseName:       "mage_review",
		AdventurerID:    q.MageID,
		AdventurerClass: "mage",
	}); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GLOOP_CONTEXT", filepath.Join(workDir, fsstore.AgentContextRel))

	stdout, stderr, code := captureStdIO(t, func() int {
		return runCommandRun(t.Context(), log, []string{"fixture-pass"})
	})
	if code != 0 {
		t.Fatalf("mage command run failed: code=%d stdout=%s stderr=%s", code, stdout, stderr)
	}
	if !strings.Contains(stdout, `"ok":true`) || !strings.Contains(stdout, `"stdout":"ok"`) {
		t.Fatalf("unexpected command response: %s", stdout)
	}
}

func TestAgentCLICommandRunUsesCommandWorkdirOverride(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture uses sh")
	}
	dataDir, realWorkDir, q := setupAgentCLITest(t)
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))

	sandboxDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(realWorkDir, "marker.txt"), []byte("real"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sandboxDir, "marker.txt"), []byte("sandbox"), 0o644); err != nil {
		t.Fatal(err)
	}

	root, err := fsstore.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := root.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	cfg.CommandAllowlist = []fsstore.AllowedCommand{{
		ID:        "pwd-marker",
		Command:   "sh",
		Args:      []string{"-c", "pwd; cat marker.txt"},
		TimeoutMs: 30_000,
	}}
	if err := root.SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}

	if err := fsstore.WriteAgentContext(realWorkDir, fsstore.AgentContext{
		DataDir:         dataDir,
		QuestID:         q.ID,
		SessionID:       "mage_0",
		Phase:           1,
		PhaseName:       "mage_review",
		AdventurerID:    q.MageID,
		AdventurerClass: "mage",
	}); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GLOOP_CONTEXT", filepath.Join(realWorkDir, fsstore.AgentContextRel))
	t.Setenv("GLOOP_COMMAND_WORKDIR", sandboxDir)

	stdout, stderr, code := captureStdIO(t, func() int {
		return runCommandRun(t.Context(), log, []string{"pwd-marker"})
	})
	if code != 0 {
		t.Fatalf("mage command run failed: code=%d stdout=%s stderr=%s", code, stdout, stderr)
	}
	if !strings.Contains(stdout, sandboxDir) || !strings.Contains(stdout, "sandbox") {
		t.Fatalf("command should run in sandbox dir, got stdout=%s stderr=%s", stdout, stderr)
	}
	if strings.Contains(stdout, realWorkDir) || strings.Contains(stdout, "real") {
		t.Fatalf("command leaked real workdir execution, got stdout=%s", stdout)
	}
}
