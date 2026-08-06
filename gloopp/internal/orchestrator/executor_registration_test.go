package orchestrator

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"code.byted.org/lihuanyu.0w0/gloop/internal/executor"
	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

type captureSessionExecutor struct {
	*executor.MockExecutor
	last              executor.SessionConfig
	lastSessionID     string
	workingDirMarker  string
	workingDirReadErr error
	writeSandboxFile  bool
}

func (e *captureSessionExecutor) CreateSession(ctx context.Context, sessionID string, opts executor.SessionConfig) (*executor.SessionHandle, error) {
	e.last = opts
	e.lastSessionID = sessionID
	if opts.WorkingDir != "" {
		raw, err := os.ReadFile(filepath.Join(opts.WorkingDir, "marker.txt"))
		e.workingDirMarker = string(raw)
		e.workingDirReadErr = err
		if e.writeSandboxFile {
			_ = os.WriteFile(filepath.Join(opts.WorkingDir, "mage-write.txt"), []byte("sandbox write"), 0o644)
		}
	}
	return e.MockExecutor.CreateSession(ctx, sessionID, opts)
}

func TestSetAgentEnabledRegistersAndUnregistersExecutor(t *testing.T) {
	eng, _ := setupTestEngine(t)

	if _, ok := eng.RegisteredExecutor("relay"); ok {
		t.Fatal("relay should not be registered before it is enabled")
	}

	if _, err := eng.SetAgentEnabled("relay", true); err != nil {
		t.Fatalf("enable relay: %v", err)
	}
	ex, ok := eng.RegisteredExecutor("relay")
	if !ok {
		t.Fatal("relay should be registered after enable")
	}
	if ex.Type() != model.AgentTypeCLI {
		t.Fatalf("relay type = %s, want cli", ex.Type())
	}

	if _, err := eng.SetAgentEnabled("relay", false); err != nil {
		t.Fatalf("disable relay: %v", err)
	}
	if _, ok := eng.RegisteredExecutor("relay"); ok {
		t.Fatal("relay should be unregistered after disable")
	}
}

func TestMagePhaseDefaultsToReadOnlyNativeTools(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	capEx := &captureSessionExecutor{MockExecutor: executor.NewMockExecutor("capture")}
	eng.RegisterExecutor("test_agent", capEx)

	q, err := eng.CreateQuest(ctx, "review readonly tools", model.QuestTypeExecute, "")
	if err != nil {
		t.Fatalf("CreateQuest: %v", err)
	}
	mage, err := eng.root.GetAdventurer("adv_mage_001")
	if err != nil {
		t.Fatalf("GetAdventurer: %v", err)
	}

	outcome := eng.runAgentPhase(ctx, q, mage, 1, phaseInput{PreviousOutput: "warrior output"})
	if outcome.Err != nil {
		t.Fatalf("runAgentPhase(mage): %v", outcome.Err)
	}
	if capEx.last.ReadOnly {
		t.Fatal("mage session should NOT be read-only (v0.3: maker/checker unified)")
	}
}

func TestMagePhaseUsesSandboxWorkingDir(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	capEx := &captureSessionExecutor{MockExecutor: executor.NewMockExecutor("capture_sandbox")}
	eng.RegisterExecutor("test_agent", capEx)

	baseDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(baseDir, "marker.txt"), []byte("real"), 0o644); err != nil {
		t.Fatal(err)
	}
	q, err := eng.CreateQuest(ctx, "review sandbox cwd", model.QuestTypeExecute, baseDir)
	if err != nil {
		t.Fatalf("CreateQuest: %v", err)
	}
	qs := fsstore.NewQuestStore(eng.root)
	q, err = qs.LoadQuest(q.ID)
	if err != nil {
		t.Fatalf("LoadQuest: %v", err)
	}
	workDir, err := qs.WorkDir(q.ID)
	if err != nil {
		t.Fatalf("WorkDir: %v", err)
	}
	q.WorkspacePath = workDir
	if err := qs.SaveQuest(q); err != nil {
		t.Fatalf("SaveQuest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(q.WorkspacePath, "marker.txt"), []byte("quest"), 0o644); err != nil {
		t.Fatal(err)
	}
	mage, err := eng.root.GetAdventurer("adv_mage_001")
	if err != nil {
		t.Fatalf("GetAdventurer: %v", err)
	}

	outcome := eng.runAgentPhase(ctx, q, mage, 1, phaseInput{PreviousOutput: "warrior output"})
	if outcome.Err != nil {
		t.Fatalf("runAgentPhase(mage): %v", outcome.Err)
	}
	if capEx.last.ReadOnly {
		t.Fatal("mage session should NOT be read-only (v0.3: unified)")
	}
	// v0.3: mage uses real workspace, no sandbox copy
	if capEx.last.WorkingDir != q.WorkspacePath {
		t.Fatalf("mage working dir should be real workspace, got %q want %q", capEx.last.WorkingDir, q.WorkspacePath)
	}
}

func TestPhaseConfigUsesCustomPhaseDef(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	capEx := &captureSessionExecutor{MockExecutor: executor.NewMockExecutor("capture_custom")}
	eng.RegisterExecutor("test_agent", capEx)

	q, err := eng.CreateQuest(ctx, "custom phase config", model.QuestTypeExecute, "")
	if err != nil {
		t.Fatalf("CreateQuest: %v", err)
	}
	qs := fsstore.NewQuestStore(eng.root)
	q, err = qs.LoadQuest(q.ID)
	if err != nil {
		t.Fatalf("LoadQuest: %v", err)
	}
	q.PipelineName = "custom"
	q.PhaseCount = 2
	q.WorkspacePath = t.TempDir()
	q.PipelineDef = []fsstore.PhaseTask{
		{PhaseIdx: 0, Name: "warrior", Role: "warrior", Class: model.ClassWarrior, Goal: "execute", Status: model.PhasePending},
		{PhaseIdx: 1, Name: "mage_design_review", Role: "mage", Class: model.ClassMage, Goal: "design review", ReadOnly: true, EndSignal: "review_quest", AllowedTools: []string{"Read"}, Status: model.PhasePending},
	}
	q.PipelineDefHash = ""
	q.Phases = nil
	if err := qs.SaveQuest(q); err != nil {
		t.Fatalf("SaveQuest: %v", err)
	}
	mage, err := eng.root.GetAdventurer("adv_mage_001")
	if err != nil {
		t.Fatalf("GetAdventurer: %v", err)
	}

	outcome := eng.runAgentPhase(ctx, q, mage, 1, phaseInput{PreviousOutput: "design"})
	if outcome.Err != nil {
		t.Fatalf("runAgentPhase(custom review): %v", outcome.Err)
	}
	if !capEx.last.ReadOnly {
		t.Fatal("custom review phase should be read-only")
	}
	if got := joinStrings(capEx.last.AllowedTools); got != "Read" {
		t.Fatalf("custom review tools = %q, want Read", got)
	}
	if !strings.HasSuffix(capEx.lastSessionID, "mage_design_review_0") {
		t.Fatalf("custom review session id = %q, want suffix mage_design_review_0", capEx.lastSessionID)
	}
}

func TestRunAgentPhaseSupportsAgentDirectBinding(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	capEx := &captureSessionExecutor{MockExecutor: executor.NewMockExecutor("direct_agent")}
	eng.RegisterExecutor("direct_agent", capEx)
	if err := eng.root.SaveAgent(&fsstore.AgentConfig{
		Name:         "direct_agent",
		Type:         model.AgentTypeCLI,
		Command:      "direct",
		DefaultModel: "direct-model",
		Enabled:      true,
	}); err != nil {
		t.Fatalf("SaveAgent: %v", err)
	}

	q := &fsstore.QuestMeta{
		ID:             "qst_agent_direct",
		Query:          "agent direct",
		Type:           model.QuestTypeExecute,
		Status:         model.QuestStatusRunning,
		WorkspacePath:  t.TempDir(),
		ExecuteAgentID: "direct_agent",
		Phases: []fsstore.PhaseTask{{
			PhaseIdx: 0,
			Role:     "warrior",
			Class:    model.ClassWarrior,
			AgentID:  "direct_agent",
			Status:   model.PhasePending,
		}},
	}

	outcome := eng.runAgentPhase(ctx, q, nil, 0, phaseInput{})
	if outcome.Err != nil {
		t.Fatalf("runAgentPhase(agent-direct): %v", outcome.Err)
	}
	if capEx.last.Model != "mock-fast" {
		t.Fatalf("session model = %q, want mock-fast", capEx.last.Model)
	}
	if got := envValue(capEx.last.Env, "GLOOP_ADVENTURER_ID"); got != "" {
		t.Fatalf("GLOOP_ADVENTURER_ID = %q, want empty", got)
	}
	if got := envValue(capEx.last.Env, "GLOOP_ADVENTURER_CLASS"); got != "" {
		t.Fatalf("GLOOP_ADVENTURER_CLASS = %q, want empty", got)
	}
	if got := envValue(capEx.last.Env, "GLOOP_AGENT_ID"); got != "direct_agent" {
		t.Fatalf("GLOOP_AGENT_ID = %q, want direct_agent", got)
	}
	if got := envValue(capEx.last.Env, "GLOOP_PHASE_ROLE"); got != "execute" {
		t.Fatalf("GLOOP_PHASE_ROLE = %q, want execute", got)
	}
}

func TestResolvePhaseActorAutoSelectsRegisteredAgentBeforePickedAdventurer(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	capEx := &captureSessionExecutor{MockExecutor: executor.NewMockExecutor("auto_agent")}
	eng.RegisterExecutor("auto_agent", capEx)
	if err := eng.root.SaveAgent(&fsstore.AgentConfig{
		Name:         "auto_agent",
		Type:         model.AgentTypeCLI,
		Command:      "auto-agent",
		DefaultModel: "auto",
		Enabled:      true,
		Official:     true,
	}); err != nil {
		t.Fatalf("SaveAgent: %v", err)
	}

	q := &fsstore.QuestMeta{
		ID:            "qst_auto_agent",
		Query:         "auto select",
		Type:          model.QuestTypeExecute,
		Status:        model.QuestStatusRunning,
		WorkspacePath: t.TempDir(),
		Phases: []fsstore.PhaseTask{{
			PhaseIdx: 0,
			Role:     "warrior",
			Class:    model.ClassWarrior,
			Status:   model.PhasePending,
		}},
	}

	outcome := eng.runAgentPhase(ctx, q, nil, 0, phaseInput{})
	if outcome.Err != nil {
		t.Fatalf("runAgentPhase(auto-agent): %v", outcome.Err)
	}
	if got := envValue(capEx.last.Env, "GLOOP_AGENT_ID"); got != "auto_agent" {
		t.Fatalf("GLOOP_AGENT_ID = %q, want auto_agent", got)
	}
	if got := envValue(capEx.last.Env, "GLOOP_ADVENTURER_ID"); got != "" {
		t.Fatalf("auto-selected agent should not use picked adventurer, got %q", got)
	}
}

func TestPickRegisteredAgentPrefersOfficialThenName(t *testing.T) {
	eng, _ := setupTestEngine(t)
	eng.RegisterExecutor("z_unofficial", executor.NewMockExecutor("z_unofficial"))
	eng.RegisterExecutor("b_official", executor.NewMockExecutor("b_official"))
	eng.RegisterExecutor("a_official", executor.NewMockExecutor("a_official"))

	agents := []*fsstore.AgentConfig{
		{Name: "z_unofficial", Type: model.AgentTypeCLI, Enabled: true, Official: false},
		{Name: "b_official", Type: model.AgentTypeCLI, Enabled: true, Official: true},
		{Name: "a_official", Type: model.AgentTypeCLI, Enabled: true, Official: true},
		{Name: "disabled_official", Type: model.AgentTypeCLI, Enabled: false, Official: true},
		{Name: "missing_executor", Type: model.AgentTypeCLI, Enabled: true, Official: true},
	}
	got := eng.pickRegisteredAgent(agents)
	if got == nil || got.Name != "a_official" {
		t.Fatalf("pickRegisteredAgent = %+v, want a_official", got)
	}
}

func TestResolvePhaseActorFallsBackToPickedAdventurerWhenNoRegisteredAgent(t *testing.T) {
	eng, _ := setupTestEngine(t)
	eng.UnregisterExecutor("test_agent")
	q := &fsstore.QuestMeta{
		ID:            "qst_picked_adv",
		Query:         "fallback",
		Type:          model.QuestTypeExecute,
		Status:        model.QuestStatusRunning,
		WorkspacePath: t.TempDir(),
		Phases: []fsstore.PhaseTask{{
			PhaseIdx: 0,
			Role:     "warrior",
			Class:    model.ClassWarrior,
			Status:   model.PhasePending,
		}},
	}
	actor, err := eng.resolvePhaseActorForQuestPhase(q, 0, nil)
	if err != nil {
		t.Fatalf("resolvePhaseActorForQuestPhase: %v", err)
	}
	if actor.Adventurer == nil || actor.Adventurer.ID != "adv_warrior_001" {
		t.Fatalf("expected picked warrior adventurer, got %+v", actor.Adventurer)
	}
	if actor.Binding.Source != PhaseBindingSourcePickedAdventurer {
		t.Fatalf("binding source = %s, want picked_adventurer", actor.Binding.Source)
	}
}

func TestGetExecutorForAgentDoesNotFallback(t *testing.T) {
	eng, _ := setupTestEngine(t)
	if _, _, err := eng.getExecutorForAgent("missing_agent"); err == nil {
		t.Fatal("getExecutorForAgent should fail for missing agent and not fallback")
	}
}

func joinStrings(in []string) string {
	out := ""
	for i, s := range in {
		if i > 0 {
			out += ","
		}
		out += s
	}
	return out
}

// TestCLIAgentEnvInjectedIntoExecutor is the orchestrator-level regression guard
// for qst_2606291778: a CLI agent's configured env (e.g. CLAUDE_CONFIG_DIR) must
// reach the spawned subprocess. Before the fix, buildExecutorForAgent computed
// EffectiveEnv() for CLI agents but dropped it (only the ACP branch forwarded
// env), so claude read the wrong config dir and returned "Not logged in".
func TestCLIAgentEnvInjectedIntoExecutor(t *testing.T) {
	eng, _ := setupTestEngine(t)

	// A claude-style CLI agent (routed to ClaudeCodeExecutor by basename) with a
	// static CLAUDE_CONFIG_DIR — exactly the failing real-world config shape.
	agent := &fsstore.AgentConfig{
		Name:         "claude_envtest",
		Type:         model.AgentTypeCLI,
		Command:      "/usr/bin/claude",
		DefaultModel: "auto",
		Enabled:      true,
		Env:          map[string]string{"CLAUDE_CONFIG_DIR": "/home/u/.claude"},
	}

	ex, err := eng.buildExecutorForAgent(agent)
	if err != nil {
		t.Fatalf("buildExecutorForAgent: %v", err)
	}
	base := executor.ResolveCLIExecutorBase(ex)
	if base == nil {
		t.Fatal("expected a CLI executor base, got nil")
	}

	// CreateSession should merge the agent env into the session env that will be
	// handed to the subprocess.
	_, st, err := base.CreateSession("envtest_session", executor.SessionConfig{
		SystemPrompt: "system",
		Env:          []string{"GLOOP_QUEST_ID=qst_envtest"},
	})
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if got := envValue(st.Env, "CLAUDE_CONFIG_DIR"); got != "/home/u/.claude" {
		t.Fatalf("agent env not injected: CLAUDE_CONFIG_DIR=%q env=%v", got, st.Env)
	}
	if got := envValue(st.Env, "GLOOP_QUEST_ID"); got != "qst_envtest" {
		t.Fatalf("session env lost: GLOOP_QUEST_ID=%q", got)
	}
}

func TestCreateQuestWithAgentDirectIDs(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	q, err := eng.CreateQuest(ctx, "agent direct creation", model.QuestTypeExecute, "", CreateQuestOptions{
		ExecuteAgentID: "test_agent",
		ReviewAgentID:  "test_agent",
	})
	if err != nil {
		t.Fatalf("CreateQuest: %v", err)
	}
	if q.ExecuteAgentID != "test_agent" {
		t.Fatalf("q.ExecuteAgentID = %q, want test_agent", q.ExecuteAgentID)
	}
	if q.ReviewAgentID != "test_agent" {
		t.Fatalf("q.ReviewAgentID = %q, want test_agent", q.ReviewAgentID)
	}

	q.EnsurePhases()
	if len(q.Phases) < 2 {
		t.Fatalf("expected at least 2 phases, got %d", len(q.Phases))
	}
	if q.Phases[0].AgentID != "test_agent" {
		t.Fatalf("phase[0].AgentID = %q, want test_agent", q.Phases[0].AgentID)
	}
	if q.Phases[1].AgentID != "test_agent" {
		t.Fatalf("phase[1].AgentID = %q, want test_agent", q.Phases[1].AgentID)
	}
}

func TestCreateQuestAgentDirectSkipsAdventurerPickWhenRequireAgent(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	// 注册一个 executor，让 agent-direct 能通过 registered 检查
	eng.RegisterExecutor("direct_agent_req", executor.NewMockExecutor("direct_agent_req"))
	if err := eng.root.SaveAgent(&fsstore.AgentConfig{
		Name:         "direct_agent_req",
		Type:         model.AgentTypeCLI,
		Command:      "direct",
		DefaultModel: "mock-fast",
		Enabled:      true,
	}); err != nil {
		t.Fatalf("SaveAgent: %v", err)
	}

	q, err := eng.CreateQuest(ctx, "require agent with direct binding", model.QuestTypeExecute, "", CreateQuestOptions{
		ExecuteAgentID: "direct_agent_req",
		ReviewAgentID:  "direct_agent_req",
		RequireAgent:   true,
	})
	if err != nil {
		t.Fatalf("CreateQuest: %v", err)
	}
	// agent-direct 模式下不应该被分配 adventurer
	if q.WarriorID != "" {
		t.Fatalf("q.WarriorID = %q, want empty (agent-direct should skip adventurer pick)", q.WarriorID)
	}
	if q.MageID != "" {
		t.Fatalf("q.MageID = %q, want empty (agent-direct should skip adventurer pick)", q.MageID)
	}
	if q.ExecuteAgentID != "direct_agent_req" {
		t.Fatalf("q.ExecuteAgentID = %q, want direct_agent_req", q.ExecuteAgentID)
	}
}

func TestCreateQuestRequireAgentStillPicksWithoutAgentDirect(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	q, err := eng.CreateQuest(ctx, "require agent legacy", model.QuestTypeExecute, "", CreateQuestOptions{
		RequireAgent: true,
	})
	if err != nil {
		t.Fatalf("CreateQuest: %v", err)
	}
	if q.WarriorID == "" {
		t.Fatal("RequireAgent should pick a warrior adventurer when no agent-direct is specified")
	}
	if q.MageID == "" {
		t.Fatal("RequireAgent should pick a mage adventurer when no agent-direct is specified")
	}
}

func TestTransportInfoForTraeXCLI(t *testing.T) {
	ex := executor.NewTraeXExecutor(executor.TraeXOpt{ExecID: "exe_traex", BinPath: "traex"})
	transport, resumeCapable := transportInfoForExecutor(ex)
	if transport != "traex" || !resumeCapable {
		t.Fatalf("transport = %q resume=%v, want traex true", transport, resumeCapable)
	}
}

func envValue(env []string, key string) string {
	prefix := key + "="
	for _, item := range env {
		if len(item) >= len(prefix) && item[:len(prefix)] == prefix {
			return item[len(prefix):]
		}
	}
	return ""
}
