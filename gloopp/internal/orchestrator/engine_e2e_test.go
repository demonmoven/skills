package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"code.byted.org/lihuanyu.0w0/gloop/internal/events"
	"code.byted.org/lihuanyu.0w0/gloop/internal/executor"
	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
	"code.byted.org/lihuanyu.0w0/gloop/internal/notifications"
	"code.byted.org/lihuanyu.0w0/gloop/internal/policy"
)

type alwaysErrorExecutor struct {
	*executor.MockExecutor
}

func (e *alwaysErrorExecutor) SendMessage(ctx context.Context, sessionID string, msg executor.Message, model string, stream chan<- executor.StreamChunk) (*executor.ChatResponse, error) {
	return nil, fmt.Errorf("forced executor error")
}

type recoverAfterErrorsExecutor struct {
	*executor.MockExecutor
	failures int
	n        int
}

func (e *recoverAfterErrorsExecutor) SendMessage(ctx context.Context, sessionID string, msg executor.Message, model string, stream chan<- executor.StreamChunk) (*executor.ChatResponse, error) {
	if e.n < e.failures {
		e.n++
		return nil, fmt.Errorf("forced transient error")
	}
	return phaseDoneResp(sessionID, "recovered", "recovered"), nil
}

type sequenceErrorExecutor struct {
	*executor.MockExecutor
	errs []error
	n    int
}

func (e *sequenceErrorExecutor) SendMessage(ctx context.Context, sessionID string, msg executor.Message, model string, stream chan<- executor.StreamChunk) (*executor.ChatResponse, error) {
	if e.n < len(e.errs) {
		err := e.errs[e.n]
		e.n++
		if err != nil {
			return nil, err
		}
	}
	return phaseDoneResp(sessionID, "agent-smoke-ok", "ok"), nil
}

type mageErrorExecutor struct {
	*executor.MockExecutor
}

func (e *mageErrorExecutor) SendMessage(ctx context.Context, sessionID string, msg executor.Message, model string, stream chan<- executor.StreamChunk) (*executor.ChatResponse, error) {
	if strings.Contains(sessionID, "mage") {
		return nil, fmt.Errorf("mage agent did not respond")
	}
	return phaseDoneResp(sessionID, "warrior ok", "warrior ok"), nil
}

// waitingInputWarriorExecutor returns a waiting_input phase signal for warrior
// sessions, delegates to default MockExecutor for mage/review sessions.
type waitingInputWarriorExecutor struct {
	*executor.MockExecutor
}

func (e *waitingInputWarriorExecutor) SendMessage(ctx context.Context, sessionID string, msg executor.Message, model string, stream chan<- executor.StreamChunk) (*executor.ChatResponse, error) {
	cfg, ok := e.SessionConfig(sessionID)
	if ok && !strings.Contains(strings.ToLower(cfg.SystemPrompt), "mage") && !strings.Contains(strings.ToLower(cfg.SystemPrompt), "checker") {
		return &executor.ChatResponse{
			SessionID:    sessionID,
			Message:      executor.Message{Role: "assistant", Content: "我需要更多信息来完成这个任务"},
			FinishReason: "stop",
			PhaseSignal: &executor.PhaseSignalData{
				OK:           false,
				Message:      "需要用户输入",
				PhaseEnded:   true,
				PhaseVerdict: "waiting_input",
				PhaseComment: "请提供更多关于任务的细节和约束条件",
				Data: map[string]any{
					"status":      "waiting_input",
					"question_id": "q_macro_waiting_test",
					"question":    "请提供更多关于任务的细节和约束条件",
				},
			},
		}, nil
	}
	return e.MockExecutor.SendMessage(ctx, sessionID, msg, model, stream)
}

// stalledExecutor 每轮都返回相同的纯文本、无工具调用，用于触发无进展检测。
type stalledExecutor struct {
	*executor.MockExecutor
}

func (e *stalledExecutor) SendMessage(ctx context.Context, sessionID string, msg executor.Message, model string, stream chan<- executor.StreamChunk) (*executor.ChatResponse, error) {
	return &executor.ChatResponse{
		SessionID:    sessionID,
		Message:      executor.Message{Role: "assistant", Content: "still thinking..."},
		FinishReason: "stop",
	}, nil
}

type captureModelSessionExecutor struct {
	*executor.MockExecutor
	lastModel string
}

func (e *captureModelSessionExecutor) CreateSession(ctx context.Context, sid string, opts executor.SessionConfig) (*executor.SessionHandle, error) {
	e.lastModel = opts.Model
	return e.MockExecutor.CreateSession(ctx, sid, opts)
}

type runtimeEventExecutor struct {
	*executor.MockExecutor
}

func (e *runtimeEventExecutor) SendMessage(ctx context.Context, sessionID string, msg executor.Message, model string, stream chan<- executor.StreamChunk) (*executor.ChatResponse, error) {
	executor.SafeStream(stream, executor.StreamChunk{SessionID: sessionID, Event: executor.RuntimeEventProcessStarted, Data: map[string]any{
		"adapter":    "test",
		"pid":        123,
		"workingDir": "/tmp/runtime-event",
	}})
	executor.SafeStream(stream, executor.StreamChunk{SessionID: sessionID, Event: executor.RuntimeEventOutputFirst, Data: map[string]any{
		"bytes": 42,
	}})
	executor.SafeStream(stream, executor.StreamChunk{SessionID: sessionID, Event: executor.RuntimeEventProcessExited, Data: map[string]any{
		"exit_code":    0,
		"duration_ms":  7,
		"stdout_bytes": 42,
	}})
	return phaseDoneResp(sessionID, "runtime healthy", "runtime healthy"), nil
}

type deltaOnlyRuntimeExecutor struct {
	*executor.MockExecutor
}

func (e *deltaOnlyRuntimeExecutor) SendMessage(ctx context.Context, sessionID string, msg executor.Message, model string, stream chan<- executor.StreamChunk) (*executor.ChatResponse, error) {
	executor.SafeStream(stream, executor.StreamChunk{SessionID: sessionID, Delta: "real output before final response"})
	return phaseDoneResp(sessionID, "delta healthy", "delta healthy"), nil
}

type externalStateWriteExecutor struct {
	*executor.MockExecutor
	outsidePath string
	insidePath  string
}

func (e *externalStateWriteExecutor) SendMessage(ctx context.Context, sessionID string, msg executor.Message, model string, stream chan<- executor.StreamChunk) (*executor.ChatResponse, error) {
	resp := phaseDoneResp(sessionID, "external state observed", "external state observed")
	resp.Meta = map[string]any{
		"traex_file_changes": []executor.FileChangeObservation{
			{Path: e.insidePath, Kind: "modify", Tool: "file_change", Status: "completed"},
			{Path: e.outsidePath, Kind: "add", Tool: "file_change", Status: "completed"},
		},
	}
	return resp, nil
}

type idleRuntimeExecutor struct {
	*executor.MockExecutor
}

func (e *idleRuntimeExecutor) SendMessage(ctx context.Context, sessionID string, msg executor.Message, model string, stream chan<- executor.StreamChunk) (*executor.ChatResponse, error) {
	select {
	case <-time.After(30 * time.Millisecond):
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return phaseDoneResp(sessionID, "idle done", "idle done"), nil
}

func setupTestEngine(t *testing.T) (*Engine, string) {
	t.Helper()

	// 临时数据目录
	dataDir := t.TempDir()
	root, err := fsstore.Open(dataDir)
	if err != nil {
		t.Fatalf("Open root failed: %v", err)
	}

	// 初始化默认文件
	_, err = root.InitDefaultFiles(fsstore.AgentProbe{}, dataDir)
	if err != nil {
		t.Fatalf("InitDefaultFiles failed: %v", err)
	}
	if err := root.SaveAgent(&fsstore.AgentConfig{
		Name:         "test_agent",
		Type:         model.AgentTypeCLI,
		Command:      "test-agent",
		DefaultModel: "mock-fast",
		Enabled:      true,
	}); err != nil {
		t.Fatalf("SaveAgent failed: %v", err)
	}

	// 测试显式创建冒险者，避免依赖产品初始化时的预创建行为。
	ts := model.NowMs()
	testAdvs := []*fsstore.AdventurerFile{
		{
			ID:          "adv_warrior_001",
			Name:        "测试剑士",
			Class:       model.ClassWarrior,
			Description: "测试执行者",
			Status:      model.AdventurerActive,
			Agent:       "test_agent",
			Level:       1,
			CreatedAtMs: ts,
		},
		{
			ID:          "adv_mage_001",
			Name:        "测试法师",
			Class:       model.ClassMage,
			Description: "测试评审者",
			Status:      model.AdventurerActive,
			Agent:       "test_agent",
			Level:       1,
			CreatedAtMs: ts,
		},
	}
	for _, a := range testAdvs {
		if err := root.SaveAdventurer(a); err != nil {
			t.Fatalf("SaveAdventurer failed: %v", err)
		}
	}

	cfg, err := root.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	// 加快测试：降低 max turns 和 max rework
	cfg.MaxTurnsPerPhase = 3
	cfg.MaxReworkPerQuest = 1
	// HOTL v0.2 默认自主闭环；测试套件沿用 v0.1 user_review 流程，显式关闭。

	bus := events.NewBus()
	workDir := t.TempDir()
	// 写点文件进去让 workspace 有内容
	os.WriteFile(filepath.Join(workDir, "README.md"), []byte("# Test Project\n\nHello world.\n"), 0o644)

	eng, err := NewEngine(root, cfg, bus, workDir, testLogger(), WithNotifier(notifications.NewNoopNotifier()))
	if err != nil {
		t.Fatalf("NewEngine failed: %v", err)
	}
	t.Cleanup(func() { eng.Shutdown(2 * time.Second) })

	// 注册 mock executor
	mockEx := executor.NewMockExecutor("exe_mock")
	eng.RegisterExecutor("test_agent", mockEx)

	return eng, dataDir
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestEngine_AdventurerModelOverridesAgentDefault(t *testing.T) {
	eng, _ := setupTestEngine(t)
	capture := &captureModelSessionExecutor{MockExecutor: executor.NewMockExecutor("capture_model")}
	eng.RegisterExecutor("test_agent", capture)

	adv, err := eng.GetAdventurer("adv_warrior_001")
	if err != nil {
		t.Fatalf("GetAdventurer: %v", err)
	}
	adv.Model = "adventurer-model"
	if _, err := eng.SaveAdventurer(adv); err != nil {
		t.Fatalf("SaveAdventurer: %v", err)
	}
	mage, err := eng.GetAdventurer("adv_mage_001")
	if err != nil {
		t.Fatalf("GetAdventurer mage: %v", err)
	}
	mage.Model = "adventurer-model"
	if _, err := eng.SaveAdventurer(mage); err != nil {
		t.Fatalf("SaveAdventurer mage: %v", err)
	}

	q, err := eng.CreateQuest(context.Background(), "模型覆盖冒烟", model.QuestTypeExecute, "", CreateQuestOptions{
		WarriorID:     "adv_warrior_001",
		MageID:        "adv_mage_001",
		WorkspaceMode: model.WorkspaceReadOnly,
	})
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	if err := eng.StartQuest(context.Background(), q.ID); err != nil {
		t.Fatalf("StartQuest failed: %v", err)
	}
	waitForStatus(t, eng, q.ID, model.QuestStatusSuccess, 10*time.Second)
	if capture.lastModel != "adventurer-model" {
		t.Fatalf("session model = %q, want adventurer-model", capture.lastModel)
	}
}

func TestEngine_PersistsAgentRuntimeHealthEvents(t *testing.T) {
	eng, _ := setupTestEngine(t)
	eng.RegisterExecutor("test_agent", &runtimeEventExecutor{MockExecutor: executor.NewMockExecutor("runtime_event")})
	ctx := context.Background()

	q, err := eng.CreateQuest(ctx, "runtime health", model.QuestTypeExecute, "", CreateQuestOptions{
		WarriorID:     "adv_warrior_001",
		MageID:        "",
		WorkspaceMode: model.WorkspaceReadOnly,
		Intensity:     model.QuestIntensityQuick,
	})
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	if err := eng.StartQuest(ctx, q.ID); err != nil {
		t.Fatalf("StartQuest failed: %v", err)
	}
	waitForStatus(t, eng, q.ID, model.QuestStatusSuccess, 10*time.Second)

	rows, err := fsstore.NewQuestStore(eng.root).ReadEvents(q.ID, 0)
	if err != nil {
		t.Fatalf("ReadEvents failed: %v", err)
	}
	seen := map[string]bool{}
	for _, row := range rows {
		seen[row.Type] = true
	}
	for _, typ := range []events.EventType{
		events.EvtAgentSessionCreated,
		events.EvtAgentProcessStarted,
		events.EvtAgentOutputFirst,
		events.EvtAgentProcessExited,
	} {
		if !seen[string(typ)] {
			t.Fatalf("missing runtime health event %s in rows: %+v", typ, rows)
		}
	}
	states, err := eng.root.ListAgentSessionStates(q.ID)
	if err != nil {
		t.Fatalf("ListAgentSessionStates failed: %v", err)
	}
	if len(states) == 0 {
		t.Fatal("agent session state should be persisted")
	}
	state := states[0]
	if state.QuestID != q.ID || state.SessionID == "" || state.CapabilityTier == "" {
		t.Fatalf("bad agent session state identity: %+v", state)
	}
	if state.FirstOutputAtMs == 0 || state.LastOutputAtMs == 0 || state.ExitCode == nil || *state.ExitCode != 0 {
		t.Fatalf("runtime fields not persisted: %+v", state)
	}
}

func TestEngine_PersistsAgentTransportSelectedEvent(t *testing.T) {
	eng, _ := setupTestEngine(t)
	eng.RegisterExecutor("test_agent", &runtimeEventExecutor{MockExecutor: executor.NewMockExecutor("runtime_event")})
	ctx := context.Background()

	q, err := eng.CreateQuest(ctx, "transport selected", model.QuestTypeExecute, "", CreateQuestOptions{
		WarriorID:     "adv_warrior_001",
		MageID:        "",
		WorkspaceMode: model.WorkspaceReadOnly,
		Intensity:     model.QuestIntensityQuick,
	})
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	if err := eng.StartQuest(ctx, q.ID); err != nil {
		t.Fatalf("StartQuest failed: %v", err)
	}
	waitForStatus(t, eng, q.ID, model.QuestStatusSuccess, 10*time.Second)

	rows, err := fsstore.NewQuestStore(eng.root).ReadEvents(q.ID, 0)
	if err != nil {
		t.Fatalf("ReadEvents failed: %v", err)
	}
	for _, row := range rows {
		if row.Type != string(events.EvtAgentTransportSelected) {
			continue
		}
		if row.SessionID == "" {
			t.Fatalf("transport event missing session id: %+v", row)
		}
		payload, ok := row.Payload.(map[string]any)
		if !ok {
			t.Fatalf("payload = %#v, want map", row.Payload)
		}
		if payload["agent"] != "test_agent" {
			t.Fatalf("agent payload = %#v, want test_agent", payload["agent"])
		}
		if payload["transport"] != string(model.AgentTypeMock) {
			t.Fatalf("transport payload = %#v, want mock", payload["transport"])
		}
		if payload["resume_capable"] != false {
			t.Fatalf("resume_capable payload = %#v, want false", payload["resume_capable"])
		}
		if payload["executor_session_id"] == "" {
			t.Fatalf("executor_session_id missing: %+v", payload)
		}
		return
	}
	t.Fatalf("missing %s event in rows: %+v", events.EvtAgentTransportSelected, rows)
}

func TestEngine_PersistsExternalStateWriteObservedEvent(t *testing.T) {
	eng, _ := setupTestEngine(t)
	workspace := t.TempDir()
	outsidePath := filepath.Join(t.TempDir(), "outside-memory.md")
	insidePath := filepath.Join(workspace, "inside.txt")
	eng.RegisterExecutor("test_agent", &externalStateWriteExecutor{
		MockExecutor: executor.NewMockExecutor("external_state"),
		outsidePath:  outsidePath,
		insidePath:   insidePath,
	})
	ctx := context.Background()

	q, err := eng.CreateQuest(ctx, "external state write", model.QuestTypeExecute, workspace, CreateQuestOptions{
		WarriorID:     "adv_warrior_001",
		MageID:        "",
		WorkspaceMode: model.WorkspaceReadOnly,
		Intensity:     model.QuestIntensityQuick,
	})
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	if err := eng.StartQuest(ctx, q.ID); err != nil {
		t.Fatalf("StartQuest failed: %v", err)
	}
	waitForStatus(t, eng, q.ID, model.QuestStatusSuccess, 10*time.Second)

	rows, err := fsstore.NewQuestStore(eng.root).ReadEvents(q.ID, 0)
	if err != nil {
		t.Fatalf("ReadEvents failed: %v", err)
	}
	seen := 0
	for _, row := range rows {
		if row.Type != string(events.EvtAgentExternalStateWriteObserved) {
			continue
		}
		seen++
		payload, ok := row.Payload.(map[string]any)
		if !ok {
			t.Fatalf("payload = %#v, want map", row.Payload)
		}
		if payload["path"] != outsidePath {
			t.Fatalf("path payload = %#v, want %s", payload["path"], outsidePath)
		}
		if payload["raw_path"] != outsidePath || payload["kind"] != "add" || payload["transport"] != "traex" {
			t.Fatalf("unexpected payload: %#v", payload)
		}
	}
	if seen != 1 {
		t.Fatalf("external state events = %d, want 1; rows=%+v", seen, rows)
	}
}

func TestEngine_PromotesFirstDeltaToRuntimeOutputEvent(t *testing.T) {
	eng, _ := setupTestEngine(t)
	eng.RegisterExecutor("test_agent", &deltaOnlyRuntimeExecutor{MockExecutor: executor.NewMockExecutor("delta_runtime")})
	ctx := context.Background()

	q, err := eng.CreateQuest(ctx, "runtime output from delta", model.QuestTypeExecute, "", CreateQuestOptions{
		WarriorID:     "adv_warrior_001",
		MageID:        "",
		WorkspaceMode: model.WorkspaceReadOnly,
		Intensity:     model.QuestIntensityQuick,
	})
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	if err := eng.StartQuest(ctx, q.ID); err != nil {
		t.Fatalf("StartQuest failed: %v", err)
	}
	waitForStatus(t, eng, q.ID, model.QuestStatusSuccess, 10*time.Second)

	rows, err := fsstore.NewQuestStore(eng.root).ReadEvents(q.ID, 0)
	if err != nil {
		t.Fatalf("ReadEvents failed: %v", err)
	}
	for _, row := range rows {
		if row.Type == string(events.EvtAgentOutputFirst) {
			return
		}
	}
	t.Fatalf("missing runtime output event promoted from first delta: %+v", rows)
}

func TestEngine_PublishesRuntimeIdleWarning(t *testing.T) {
	eng, _ := setupTestEngine(t)
	eng.cfg.RuntimeIdleWarningMs = 5
	eng.RegisterExecutor("test_agent", &idleRuntimeExecutor{MockExecutor: executor.NewMockExecutor("idle_runtime")})
	ctx := context.Background()

	q, err := eng.CreateQuest(ctx, "runtime idle warning", model.QuestTypeExecute, "", CreateQuestOptions{
		WarriorID:     "adv_warrior_001",
		MageID:        "",
		WorkspaceMode: model.WorkspaceReadOnly,
		Intensity:     model.QuestIntensityQuick,
	})
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	if err := eng.StartQuest(ctx, q.ID); err != nil {
		t.Fatalf("StartQuest failed: %v", err)
	}
	waitForStatus(t, eng, q.ID, model.QuestStatusSuccess, 10*time.Second)

	rows, err := fsstore.NewQuestStore(eng.root).ReadEvents(q.ID, 0)
	if err != nil {
		t.Fatalf("ReadEvents failed: %v", err)
	}
	for _, row := range rows {
		if row.Type == string(events.EvtAgentIdleWarning) {
			return
		}
	}
	t.Fatalf("missing runtime idle warning: %+v", rows)
}

func waitForStatus(t *testing.T, eng *Engine, qid string, target model.QuestStatus, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		q, err := eng.GetQuest(qid)
		if err != nil {
			time.Sleep(50 * time.Millisecond)
			continue
		}
		if q.Status == target {
			return
		}
		if q.Status == model.QuestStatusFailed || q.Status == model.QuestStatusCancelled {
			dumpQuestEventsOnFailure(t, eng, qid)
			t.Fatalf("quest reached unexpected terminal state: %s comment=%q", q.Status, q.FinalComment)
		}
		time.Sleep(100 * time.Millisecond)
	}
	dumpQuestEventsOnFailure(t, eng, qid)
	t.Fatalf("timeout waiting for quest status %s", target)
}

func waitForNotStatus(t *testing.T, eng *Engine, qid string, avoid model.QuestStatus, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		q, err := eng.GetQuest(qid)
		if err != nil {
			time.Sleep(50 * time.Millisecond)
			continue
		}
		if q.Status != avoid {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	dumpQuestEventsOnFailure(t, eng, qid)
	t.Fatalf("timeout waiting for quest to leave status %s", avoid)
}

func dumpQuestEventsOnFailure(t *testing.T, eng *Engine, qid string) {
	t.Helper()
	if eng == nil || eng.root == nil || qid == "" {
		return
	}
	qs := fsstore.NewQuestStore(eng.root)
	q, err := qs.LoadQuest(qid)
	if err == nil && q != nil {
		t.Logf("quest snapshot: status=%s current_phase=%d rework=%d final=%s blocked=%s/%s",
			q.Status, q.CurrentPhaseIdx(), q.ReworkCount, q.FinalVerdict, q.BlockedReasonCode, truncate(q.BlockedReason, 160))
	}
	rows, err := qs.ReadEvents(qid, 20)
	if err != nil {
		t.Logf("quest events unavailable: %v", err)
		return
	}
	t.Logf("last %d quest events:", len(rows))
	for _, row := range rows {
		t.Logf("  #%d %s sid=%s payload=%v", row.ID, row.Type, row.SessionID, row.Payload)
	}
}

func TestEngine_BlocksAfterConsecutiveAgentErrors(t *testing.T) {
	eng, _ := setupTestEngine(t)
	eng.cfg.MaxConsecutiveAgentErrors = 2
	eng.RegisterExecutor("test_agent", &alwaysErrorExecutor{MockExecutor: executor.NewMockExecutor("err_mock")})

	ctx := context.Background()
	q, err := eng.CreateQuest(ctx, "触发 executor 错误", model.QuestTypeExecute, "")
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	if err := eng.StartQuest(ctx, q.ID); err != nil {
		t.Fatalf("StartQuest failed: %v", err)
	}
	waitForStatus(t, eng, q.ID, model.QuestStatusBlocked, 10*time.Second)
	got, err := eng.GetQuest(q.ID)
	if err != nil {
		t.Fatalf("GetQuest failed: %v", err)
	}
	if !strings.Contains(got.BlockedReason, "连续 2 次") {
		t.Fatalf("blocked reason should mention consecutive errors, got %q", got.BlockedReason)
	}
	ev := lastFailureAttributionEvent(t, eng, q.ID)
	if ev["reason"] != "agent_consecutive_errors" || ev["recoverable"] != true {
		t.Fatalf("unexpected failure attribution: %+v", ev)
	}
	got, err = eng.GetQuest(q.ID)
	if err != nil {
		t.Fatalf("GetQuest failed: %v", err)
	}
	if got.FailureAttribution == nil || got.FailureAttribution.Reason != "agent_consecutive_errors" {
		t.Fatalf("failure attribution summary not cached: %+v", got.FailureAttribution)
	}
	if got.BlockedReasonCode != "agent_consecutive_errors" {
		t.Fatalf("blocked reason code = %q, want agent_consecutive_errors", got.BlockedReasonCode)
	}
	state, err := eng.root.GetRecoveryState(q.ID)
	if err != nil {
		t.Fatalf("GetRecoveryState failed: %v", err)
	}
	if state.PolicyName == "" || state.LastAction != policy.ActionRetry || state.InputHash == "" || state.BlockReason != "agent_consecutive_errors" {
		t.Fatalf("bad recovery state: %+v", state)
	}
	if state.AddTurns != policy.RecoveryDefaultAddTurns ||
		state.AddDurationMinutes != policy.RecoveryDefaultAddDurationMinutes ||
		state.MaxAttempts == 0 {
		t.Fatalf("recovery state should snapshot policy params: %+v", state)
	}
	eventsRows, err := fsstore.NewQuestStore(eng.root).ReadEvents(q.ID, 0)
	if err != nil {
		t.Fatalf("ReadEvents failed: %v", err)
	}
	foundRecoveryDecision := false
	for _, row := range eventsRows {
		if row.Type != "policy.decision" {
			continue
		}
		payload, _ := row.Payload.(map[string]any)
		if payload["policy_kind"] == "recovery" {
			foundRecoveryDecision = true
			break
		}
	}
	if !foundRecoveryDecision {
		t.Fatalf("expected recovery policy.decision event, rows=%+v", eventsRows)
	}
}

func TestEngine_BlocksImmediatelyOnAuthError(t *testing.T) {
	eng, _ := setupTestEngine(t)
	eng.RegisterExecutor("test_agent", &sequenceErrorExecutor{
		MockExecutor: executor.NewMockExecutor("auth_mock"),
		errs:         []error{fmt.Errorf("Not logged in · Please run /login")},
	})

	ctx := context.Background()
	q, err := eng.CreateQuest(ctx, "触发登录错误", model.QuestTypeExecute, "")
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	if err := eng.StartQuest(ctx, q.ID); err != nil {
		t.Fatalf("StartQuest failed: %v", err)
	}
	waitForStatus(t, eng, q.ID, model.QuestStatusBlocked, 10*time.Second)
	got, err := eng.GetQuest(q.ID)
	if err != nil {
		t.Fatalf("GetQuest failed: %v", err)
	}
	if !strings.Contains(got.BlockedReason, "auth") || !strings.Contains(got.BlockedReason, "Not logged in") {
		t.Fatalf("blocked reason should mention auth error, got %q", got.BlockedReason)
	}
	ev := lastFailureAttributionEvent(t, eng, q.ID)
	if ev["reason"] != "agent_error_auth" || ev["category"] != "auth" || ev["recoverable"] != true {
		t.Fatalf("unexpected auth attribution: %+v", ev)
	}
}

func TestEngine_BlocksWhenMageAgentErrorsDuringReviewing(t *testing.T) {
	eng, _ := setupTestEngine(t)
	eng.RegisterExecutor("test_agent", &mageErrorExecutor{MockExecutor: executor.NewMockExecutor("mage_error_mock")})

	ctx := context.Background()
	q, err := eng.CreateQuest(ctx, "法师阶段无响应", model.QuestTypeExecute, "", CreateQuestOptions{
		WarriorID: "adv_warrior_001",
		MageID:    "adv_mage_001",
	})
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	if err := eng.StartQuest(ctx, q.ID); err != nil {
		t.Fatalf("StartQuest failed: %v", err)
	}

	waitForStatus(t, eng, q.ID, model.QuestStatusBlocked, 10*time.Second)
	got, err := eng.GetQuest(q.ID)
	if err != nil {
		t.Fatalf("GetQuest failed: %v", err)
	}
	if got.BlockedReasonCode != "agent_consecutive_errors" {
		t.Fatalf("blocked reason code = %q, want agent_consecutive_errors; reason=%q", got.BlockedReasonCode, got.BlockedReason)
	}
	if !strings.Contains(got.BlockedReason, "mage agent did not respond") {
		t.Fatalf("blocked reason should mention mage error, got %q", got.BlockedReason)
	}
}

func TestEngine_RetriesRateLimitThenContinues(t *testing.T) {
	eng, _ := setupTestEngine(t)
	eng.cfg.MaxTurnsPerPhase = 4
	eng.RegisterExecutor("test_agent", &sequenceErrorExecutor{
		MockExecutor: executor.NewMockExecutor("retry_mock"),
		errs: []error{
			fmt.Errorf("429 rate limit"),
			fmt.Errorf("429 rate limit"),
			nil,
		},
	})

	ctx := context.Background()
	q, err := eng.CreateQuest(ctx, "触发限流重试", model.QuestTypeExecute, "")
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	if err := eng.StartQuest(ctx, q.ID); err != nil {
		t.Fatalf("StartQuest failed: %v", err)
	}
	waitForStatus(t, eng, q.ID, model.QuestStatusBlocked, 20*time.Second)
}

func TestEngine_RunAllowedCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture uses sh")
	}
	eng, _ := setupTestEngine(t)
	eng.cfg.CommandAllowlist = []fsstore.AllowedCommand{{
		ID:          "fixture-pass",
		Command:     "sh",
		Args:        []string{"-c", "printf ok"},
		TimeoutMs:   30_000,
		Description: "test fixture",
	}}
	q, err := eng.CreateQuest(context.Background(), "command runner", model.QuestTypeExecute, "")
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	q.WorkspacePath = eng.workDir
	if err := fsstore.NewQuestStore(eng.root).SaveQuest(q); err != nil {
		t.Fatalf("SaveQuest failed: %v", err)
	}

	result, err := eng.RunAllowedCommand(context.Background(), q.ID, "mage_0", "fixture-pass", nil)
	if err != nil {
		t.Fatalf("RunAllowedCommand failed: %v", err)
	}
	if result.ExitCode != 0 || result.Stdout != "ok" {
		t.Fatalf("unexpected result: %+v", result)
	}
	eventsRows, err := fsstore.NewQuestStore(eng.root).ReadEvents(q.ID, 0)
	if err != nil {
		t.Fatalf("ReadEvents failed: %v", err)
	}
	found := false
	for _, ev := range eventsRows {
		if ev.Type == string(events.EvtCommandRun) {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("command run event not written")
	}
}

func TestEngine_BlocksL2CommandWithoutPolicy(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture uses sh")
	}
	eng, _ := setupTestEngine(t)
	eng.cfg.CommandAllowlist = []fsstore.AllowedCommand{{
		ID:              "external-write",
		Command:         "sh",
		Args:            []string{"-c", "printf no"},
		Description:     "test L2 fixture",
		SideEffectLevel: "L2",
	}}
	q, err := eng.CreateQuest(context.Background(), "l2 command", model.QuestTypeExecute, "")
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	q.WorkspacePath = eng.workDir
	if err := fsstore.NewQuestStore(eng.root).SaveQuest(q); err != nil {
		t.Fatalf("SaveQuest failed: %v", err)
	}

	_, err = eng.RunAllowedCommand(context.Background(), q.ID, "mage_0", "external-write", nil)
	if err == nil || !strings.Contains(err.Error(), "L2") {
		t.Fatalf("expected L2 policy error, got %v", err)
	}
}

func TestEngine_AllowsL2CommandWithAutomationPolicy(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture uses sh")
	}
	eng, _ := setupTestEngine(t)
	eng.cfg.CommandAllowlist = []fsstore.AllowedCommand{{
		ID:              "external-write",
		Command:         "sh",
		Args:            []string{"-c", "printf ok"},
		Description:     "test L2 fixture",
		SideEffectLevel: "L2",
	}}
	autoCfg := &fsstore.AutomationConfig{
		ID:        "l2_policy",
		Name:      "L2 policy",
		Enabled:   true,
		Query:     "l2 command",
		QuestType: string(model.QuestTypeExecute),
		AllowL2:   true,
	}
	if err := eng.SaveAutomation(autoCfg); err != nil {
		t.Fatalf("SaveAutomation failed: %v", err)
	}
	q, err := eng.CreateQuest(context.Background(), "l2 command", model.QuestTypeExecute, "")
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	q.WorkspacePath = eng.workDir
	q.CreatedBy = model.QuestSourcePrefix + autoCfg.ID
	if err := fsstore.NewQuestStore(eng.root).SaveQuest(q); err != nil {
		t.Fatalf("SaveQuest failed: %v", err)
	}

	result, err := eng.RunAllowedCommand(context.Background(), q.ID, "mage_0", "external-write", nil)
	if err != nil {
		t.Fatalf("RunAllowedCommand failed: %v", err)
	}
	if result.Stdout != "ok" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestEngine_BlocksAfterNoProgress(t *testing.T) {
	eng, _ := setupTestEngine(t)
	eng.cfg.MaxNoProgressTurns = 3
	eng.cfg.MaxTurnsPerPhase = 10 // 确保无进展先于 turn 上限触发
	eng.RegisterExecutor("test_agent", &stalledExecutor{MockExecutor: executor.NewMockExecutor("stall_mock")})

	ctx := context.Background()
	q, err := eng.CreateQuest(ctx, "触发无进展", model.QuestTypeExecute, "")
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	if err := eng.StartQuest(ctx, q.ID); err != nil {
		t.Fatalf("StartQuest failed: %v", err)
	}
	waitForStatus(t, eng, q.ID, model.QuestStatusBlocked, 10*time.Second)
	got, err := eng.GetQuest(q.ID)
	if err != nil {
		t.Fatalf("GetQuest failed: %v", err)
	}
	if !strings.Contains(got.BlockedReason, "无进展") && !strings.Contains(got.BlockedReason, "无新") {
		t.Fatalf("blocked reason should mention no progress, got %q", got.BlockedReason)
	}
	ev := lastFailureAttributionEvent(t, eng, q.ID)
	if ev["reason"] != "no_progress" || ev["recoverable"] != true {
		t.Fatalf("unexpected no-progress attribution: %+v", ev)
	}
}

func lastFailureAttributionEvent(t *testing.T, eng *Engine, qid string) map[string]any {
	t.Helper()
	rows, err := eng.GetQuestEvents(qid, 0)
	if err != nil {
		t.Fatalf("GetQuestEvents failed: %v", err)
	}
	for i := len(rows) - 1; i >= 0; i-- {
		if rows[i].Type == string(events.EvtFailureAttributed) {
			payload, ok := rows[i].Payload.(map[string]any)
			if !ok {
				t.Fatalf("failure attribution payload type = %T, want map[string]any", rows[i].Payload)
			}
			return payload
		}
	}
	t.Fatalf("missing %s event in %+v", events.EvtFailureAttributed, rows)
	return nil
}

func TestEngine_FullLoop_Mock(t *testing.T) {
	eng, _ := setupTestEngine(t)
	hotlAutoClose := true
	eng.cfg.HotlAutoClose = &hotlAutoClose
	ctx := context.Background()

	// 1. 创建 quest
	q, err := eng.CreateQuest(ctx, "实现一个 hello world 函数并加上单元测试", model.QuestTypeExecute, "")
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	if q.Status != model.QuestStatusPending {
		t.Errorf("expected pending status, got %s", q.Status)
	}
	t.Logf("Quest created: id=%s, mode=%s", q.ID, q.WorkspaceMode)

	// 2. 启动 quest
	if err := eng.StartQuest(ctx, q.ID); err != nil {
		t.Fatalf("StartQuest failed: %v", err)
	}
	if !eng.IsQuestRunning(q.ID) {
		t.Error("quest should be running")
	}

	// 3. HOTL: checker pass → auto-close → success（不再经过 user_review）
	waitForStatus(t, eng, q.ID, model.QuestStatusSuccess, 10*time.Second)

	q, _ = eng.GetQuest(q.ID)
	t.Logf("Quest reached success. rework_count=%d, final_verdict=%s", q.ReworkCount, q.FinalVerdict)

	if q.FinalVerdict != model.VerdictPass {
		t.Errorf("expected pass verdict, got %s", q.FinalVerdict)
	}
	if q.CompletedAtMs == 0 {
		t.Fatalf("completed_at should be set: %+v", q)
	}
	// HOTL: auto-closed by policy, not user

	// 5. 计算 diff
	diff, err := eng.ComputeDiff(ctx, q.ID)
	if err != nil {
		t.Fatalf("ComputeDiff failed: %v", err)
	}
	t.Logf("Diff: %d files changed, +%d -%d", diff.ChangedFiles, diff.Additions, diff.Deletions)
	t.Logf("Stat: %s", diff.Stat)

	// 6. HOTL auto-close 已自动 apply；验证状态
	q, _ = eng.GetQuest(q.ID)
	if !q.Applied {
		t.Error("quest should be marked as applied by HOTL auto-close")
	}
	reports, err := eng.root.LoadReports(q.ID)
	if err != nil {
		t.Fatalf("LoadReports failed: %v", err)
	}
	if len(reports.MakerReports) == 0 {
		t.Fatal("expected phase done to create MakerReport")
	}
	if reports.MakerReports[len(reports.MakerReports)-1].SchemaVersion != "maker_report.v1" {
		t.Fatalf("bad MakerReport: %+v", reports.MakerReports[len(reports.MakerReports)-1])
	}
	if len(reports.ReviewReports) == 0 {
		t.Fatal("expected review verdict to create ReviewReport")
	}
	reviewReport := reports.ReviewReports[len(reports.ReviewReports)-1]
	if reviewReport.SchemaVersion != "review_report.v1" || reviewReport.Confidence != "high" || len(reviewReport.CheckedAgainst) == 0 {
		t.Fatalf("bad evidence-backed ReviewReport: %+v", reviewReport)
	}
	if len(reviewReport.EvidenceRefs) == 0 || reviewReport.EvidenceRefs[0].TrustTier != "T0" || reviewReport.EvidenceRefs[0].CommandID == "" {
		t.Fatalf("ReviewReport should include T0 command evidence refs: %+v", reviewReport.EvidenceRefs)
	}
}

func TestEngine_HotlAutoCloseDisabledReachesUserReview(t *testing.T) {
	eng, _ := setupTestEngine(t)
	hotlAutoClose := false
	eng.cfg.HotlAutoClose = &hotlAutoClose

	q, err := eng.CreateQuest(context.Background(), "verify manual review path", model.QuestTypeExecute, "")
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	if err := eng.StartQuest(context.Background(), q.ID); err != nil {
		t.Fatalf("StartQuest failed: %v", err)
	}

	waitForStatus(t, eng, q.ID, model.QuestStatusUserReview, 10*time.Second)
	got, err := eng.GetQuest(q.ID)
	if err != nil {
		t.Fatalf("GetQuest failed: %v", err)
	}
	if got.Status != model.QuestStatusUserReview || got.Applied {
		t.Fatalf("HOTL disabled should stop at user_review without apply: %+v", got)
	}
	if got.FinalVerdict != model.VerdictPass {
		t.Fatalf("review verdict should still be recorded before user_review, got %q", got.FinalVerdict)
	}
}

func TestEngine_CreateQuestCreatesRootThreadPost(t *testing.T) {
	eng, _ := setupTestEngine(t)

	q, err := eng.CreateQuest(context.Background(), "保留原始委托\n第二行", model.QuestTypeExecute, "", CreateQuestOptions{
		WarriorID:    "adv_warrior_001",
		MageID:       "adv_mage_001",
		WorkflowMode: model.WorkflowModeDirect,
	})
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	if q.OriginalRequest != "保留原始委托\n第二行" {
		t.Fatalf("original_request = %q", q.OriginalRequest)
	}
	if q.WorkflowMode != model.WorkflowModeDirect {
		t.Fatalf("workflow_mode = %q, want direct", q.WorkflowMode)
	}
	if q.IntentSummary != "保留原始委托" {
		t.Fatalf("intent_summary = %q", q.IntentSummary)
	}

	posts, err := fsstore.NewQuestStore(eng.root).LoadThreadPosts(q.ID)
	if err != nil {
		t.Fatalf("LoadThreadPosts failed: %v", err)
	}
	if len(posts) != 1 {
		t.Fatalf("thread posts = %d, want root only: %+v", len(posts), posts)
	}
	root := posts[0]
	if root.PostID != fsstore.RootPostID(q.ID) || root.ThreadID != q.ID || root.RootPostID != root.PostID {
		t.Fatalf("bad root identity: %+v", root)
	}
	if root.AuthorRole != model.PostRoleHuman || root.Content != q.OriginalRequest {
		t.Fatalf("bad root post: %+v", root)
	}
}

func TestEngine_PublishPostDedupesActivityReplyPreview(t *testing.T) {
	eng, _ := setupTestEngine(t)

	q, err := eng.CreateQuest(context.Background(), "activity publish post preview", model.QuestTypeExecute, "", CreateQuestOptions{
		WarriorID:    "adv_warrior_001",
		MageID:       "adv_mage_001",
		WorkflowMode: model.WorkflowModeChecked,
	})
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	postID, err := eng.PublishPost(q.ID, "mage_0", "published checker reply", fsstore.RootPostID(q.ID), "review_report")
	if err != nil {
		t.Fatalf("PublishPost failed: %v", err)
	}
	posts, err := fsstore.NewQuestStore(eng.root).LoadThreadPosts(q.ID)
	if err != nil {
		t.Fatalf("LoadThreadPosts failed: %v", err)
	}
	if len(posts) != 2 {
		t.Fatalf("thread posts = %d, want root + published reply: %+v", len(posts), posts)
	}
	found := false
	for _, post := range posts {
		if post.PostID == postID {
			found = true
		}
	}
	if !found {
		t.Fatalf("published post %s not written to thread_posts: %+v", postID, posts)
	}
}

func TestEngine_SpawnQuestPersistsFanoutContract(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	qid, err := eng.SpawnQuest(ctx, "fanout leaf", "grp_contract", "", "", fsstore.FanoutContract{
		LeafID:          "leaf-a",
		OwnershipScopes: []string{"internal/server", " internal/server "},
		MergeStrategy:   fsstore.MergeStrategyNoMerge,
	})
	if err != nil {
		t.Fatalf("SpawnQuest failed: %v", err)
	}
	q, err := eng.GetQuest(qid)
	if err != nil {
		t.Fatalf("GetQuest failed: %v", err)
	}
	if q.GroupID != "grp_contract" || q.FanoutLeafID != "leaf-a" || q.MergeStrategy != fsstore.MergeStrategyNoMerge {
		t.Fatalf("fanout contract not persisted: %+v", q)
	}
	if len(q.OwnershipScopes) != 1 || q.OwnershipScopes[0] != "internal/server" {
		t.Fatalf("ownership scopes not normalized: %+v", q.OwnershipScopes)
	}
}

func TestEngine_SpawnQuestRejectsOwnershipConflictWithoutMergeOwner(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	if _, err := eng.SpawnQuest(ctx, "leaf a", "grp_conflict", "", "", fsstore.FanoutContract{
		LeafID:          "leaf-a",
		OwnershipScopes: []string{"internal/server"},
		MergeStrategy:   fsstore.MergeStrategyNoMerge,
	}); err != nil {
		t.Fatalf("SpawnQuest leaf-a failed: %v", err)
	}
	_, err := eng.SpawnQuest(ctx, "leaf b", "grp_conflict", "", "", fsstore.FanoutContract{
		LeafID:          "leaf-b",
		OwnershipScopes: []string{"internal/server"},
		MergeStrategy:   fsstore.MergeStrategyNoMerge,
	})
	if err == nil || !strings.Contains(err.Error(), "ownership scope conflict") {
		t.Fatalf("expected ownership conflict, got %v", err)
	}
}

func TestFirstSpawn_PromotesParentToRoot(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	// Create a normal quest (no GroupID) as parent.
	parent, err := eng.CreateQuest(ctx, "parent quest for fanout", model.QuestTypeExecute, "")
	if err != nil {
		t.Fatalf("CreateQuest parent failed: %v", err)
	}
	if parent.GroupID != "" {
		t.Fatalf("new quest should have empty GroupID, got %q", parent.GroupID)
	}

	// Spawn a leaf under this parent — should promote parent to root.
	leafQID, err := eng.SpawnQuest(ctx, "leaf task", "", "", parent.ID, fsstore.FanoutContract{
		LeafID: "leaf-1",
	})
	if err != nil {
		t.Fatalf("SpawnQuest failed: %v", err)
	}

	// Verify parent was promoted to root.
	parentAfter, err := eng.GetQuest(parent.ID)
	if err != nil {
		t.Fatalf("GetQuest parent failed: %v", err)
	}
	if parentAfter.GroupID == "" {
		t.Fatal("parent should have been promoted to root (GroupID set)")
	}
	if parentAfter.FanoutLeafID != "" {
		t.Fatalf("root should have empty FanoutLeafID, got %q", parentAfter.FanoutLeafID)
	}

	// Verify leaf has ParentQuestID pointing to root.
	leaf, err := eng.GetQuest(leafQID)
	if err != nil {
		t.Fatalf("GetQuest leaf failed: %v", err)
	}
	if leaf.ParentQuestID != parent.ID {
		t.Fatalf("leaf ParentQuestID = %q, want %q", leaf.ParentQuestID, parent.ID)
	}
	if leaf.GroupID != parentAfter.GroupID {
		t.Fatalf("leaf GroupID = %q, want %q (same as root)", leaf.GroupID, parentAfter.GroupID)
	}
}

func TestSecondSpawn_ReusesSameRoot(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	parent, err := eng.CreateQuest(ctx, "parent quest", model.QuestTypeExecute, "")
	if err != nil {
		t.Fatalf("CreateQuest parent failed: %v", err)
	}

	// First spawn promotes parent.
	leaf1QID, err := eng.SpawnQuest(ctx, "leaf 1", "", "", parent.ID, fsstore.FanoutContract{
		LeafID: "leaf-1",
	})
	if err != nil {
		t.Fatalf("first SpawnQuest failed: %v", err)
	}

	parentAfter, _ := eng.GetQuest(parent.ID)
	groupID := parentAfter.GroupID

	// Second spawn should reuse same root, not promote again.
	leaf2QID, err := eng.SpawnQuest(ctx, "leaf 2", "", "", parent.ID, fsstore.FanoutContract{
		LeafID: "leaf-2",
	})
	if err != nil {
		t.Fatalf("second SpawnQuest failed: %v", err)
	}

	parentFinal, _ := eng.GetQuest(parent.ID)
	if parentFinal.GroupID != groupID {
		t.Fatalf("root GroupID changed: was %q, now %q", groupID, parentFinal.GroupID)
	}

	leaf1, _ := eng.GetQuest(leaf1QID)
	leaf2, _ := eng.GetQuest(leaf2QID)
	if leaf1.ParentQuestID != parent.ID || leaf2.ParentQuestID != parent.ID {
		t.Fatalf("both leaves should point to same root, got %q and %q", leaf1.ParentQuestID, leaf2.ParentQuestID)
	}
	if leaf1.GroupID != groupID || leaf2.GroupID != groupID {
		t.Fatalf("both leaves should share same group, got %q and %q", leaf1.GroupID, leaf2.GroupID)
	}
}

func TestFindFanoutRoot_FindsRootByGroupID(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	parent, err := eng.CreateQuest(ctx, "parent quest", model.QuestTypeExecute, "")
	if err != nil {
		t.Fatalf("CreateQuest parent failed: %v", err)
	}

	// Spawn to trigger promotion.
	_, err = eng.SpawnQuest(ctx, "leaf task", "", "", parent.ID, fsstore.FanoutContract{
		LeafID: "leaf-1",
	})
	if err != nil {
		t.Fatalf("SpawnQuest failed: %v", err)
	}

	parentAfter, _ := eng.GetQuest(parent.ID)
	qs := fsstore.NewQuestStore(eng.root)

	// FindFanoutRoot should find the promoted root.
	root, found := qs.FindFanoutRoot(parentAfter.GroupID)
	if !found {
		t.Fatal("FindFanoutRoot should find promoted root")
	}
	if root.ID != parent.ID {
		t.Fatalf("FindFanoutRoot returned %q, want %q", root.ID, parent.ID)
	}
	if root.FanoutLeafID != "" {
		t.Fatalf("found root should have empty FanoutLeafID, got %q", root.FanoutLeafID)
	}

	// Non-existent group should not be found.
	_, found = qs.FindFanoutRoot("nonexistent_group")
	if found {
		t.Fatal("FindFanoutRoot should not find non-existent group")
	}
}

func TestFirstSpawn_PromotesParentAndWritesGroupOpened(t *testing.T) {
	eng, _ := setupTestEngine(t)
	// Disable auto-close so quest doesn't complete during test.
	hotlAutoClose := false
	eng.cfg.HotlAutoClose = &hotlAutoClose
	ctx := context.Background()

	parent, err := eng.CreateQuest(ctx, "parent for group opened test", model.QuestTypeExecute, "")
	if err != nil {
		t.Fatalf("CreateQuest parent failed: %v", err)
	}

	_, err = eng.SpawnQuest(ctx, "leaf task", "", "", parent.ID, fsstore.FanoutContract{
		LeafID: "leaf-1",
	})
	if err != nil {
		t.Fatalf("SpawnQuest failed: %v", err)
	}

	// Verify fanout_group_opened post exists on root thread.
	qs := fsstore.NewQuestStore(eng.root)
	posts, err := qs.LoadThreadPosts(parent.ID)
	if err != nil {
		t.Fatalf("LoadThreadPosts failed: %v", err)
	}
	var groupOpenedPost *fsstore.ThreadPost
	for i := range posts {
		if posts[i].Kind == "fanout_group_opened" {
			groupOpenedPost = &posts[i]
			break
		}
	}
	if groupOpenedPost == nil {
		t.Fatal("fanout_group_opened post should exist on root thread after first spawn")
	}
	if groupOpenedPost.AuthorRole != model.PostRoleSystem {
		t.Errorf("group_opened author_role = %s, want %s", groupOpenedPost.AuthorRole, model.PostRoleSystem)
	}
	if groupOpenedPost.SourceEventID == 0 {
		t.Error("group_opened SourceEventID should not be 0")
	}
	if !strings.HasPrefix(groupOpenedPost.PostID, "fanout_group_opened_") {
		t.Errorf("group_opened post_id = %s, want prefix fanout_group_opened_", groupOpenedPost.PostID)
	}
}

func TestSpawnQuest_WritesLeafSpawnedPostInRootThread(t *testing.T) {
	eng, _ := setupTestEngine(t)
	hotlAutoClose := false
	eng.cfg.HotlAutoClose = &hotlAutoClose
	ctx := context.Background()

	parent, err := eng.CreateQuest(ctx, "parent for leaf spawned test", model.QuestTypeExecute, "")
	if err != nil {
		t.Fatalf("CreateQuest parent failed: %v", err)
	}

	// First spawn: should have both group_opened and leaf_spawned.
	_, err = eng.SpawnQuest(ctx, "leaf task 1", "", "", parent.ID, fsstore.FanoutContract{
		LeafID: "leaf-1",
	})
	if err != nil {
		t.Fatalf("first SpawnQuest failed: %v", err)
	}

	// Second spawn: should have another leaf_spawned but no new group_opened.
	_, err = eng.SpawnQuest(ctx, "leaf task 2", "", "", parent.ID, fsstore.FanoutContract{
		LeafID: "leaf-2",
	})
	if err != nil {
		t.Fatalf("second SpawnQuest failed: %v", err)
	}

	qs := fsstore.NewQuestStore(eng.root)
	posts, err := qs.LoadThreadPosts(parent.ID)
	if err != nil {
		t.Fatalf("LoadThreadPosts failed: %v", err)
	}

	var groupOpenedCount, leafSpawnedCount int
	for _, p := range posts {
		switch p.Kind {
		case "fanout_group_opened":
			groupOpenedCount++
		case "fanout_leaf_spawned":
			leafSpawnedCount++
		}
	}
	if groupOpenedCount != 1 {
		t.Fatalf("should have exactly 1 fanout_group_opened post, got %d", groupOpenedCount)
	}
	if leafSpawnedCount != 2 {
		t.Fatalf("should have exactly 2 fanout_leaf_spawned posts (one per leaf), got %d", leafSpawnedCount)
	}

	// Verify leaf_spawned post properties.
	for _, p := range posts {
		if p.Kind == "fanout_leaf_spawned" {
			if p.AuthorRole != model.PostRoleSystem {
				t.Errorf("leaf_spawned author_role = %s, want %s", p.AuthorRole, model.PostRoleSystem)
			}
			if p.SourceEventID == 0 {
				t.Error("leaf_spawned SourceEventID should not be 0")
			}
			if !strings.HasPrefix(p.PostID, "fanout_leaf_spawned_") {
				t.Errorf("leaf_spawned post_id = %s, want prefix fanout_leaf_spawned_", p.PostID)
			}
		}
	}

	// Verify each leaf_spawned post's SourceEventID corresponds to an actual root event.
	events, evErr := qs.ReadEvents(parent.ID, 0)
	if evErr != nil {
		t.Fatalf("ReadEvents failed: %v", evErr)
	}
	leafSpawnedPosts := make(map[int64]bool)
	for _, p := range posts {
		if p.Kind == "fanout_leaf_spawned" {
			leafSpawnedPosts[p.SourceEventID] = true
		}
	}
	for evID := range leafSpawnedPosts {
		found := false
		for _, ev := range events {
			if ev.ID == evID {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("leaf_spawned post SourceEventID %d not found in root events", evID)
		}
	}
	if len(leafSpawnedPosts) != 2 {
		t.Errorf("expected 2 distinct leaf_spawned SourceEventIDs, got %d", len(leafSpawnedPosts))
	}
}

func TestLeafTerminal_PublishesRootQuestEvent(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()
	qs := fsstore.NewQuestStore(eng.root)

	// Create parent and spawn a leaf (establishes fanout group).
	parent, err := eng.CreateQuest(ctx, "parent for leaf terminal event test", model.QuestTypeExecute, "")
	if err != nil {
		t.Fatalf("CreateQuest parent: %v", err)
	}
	leaf, err := eng.SpawnQuest(ctx, "leaf task for terminal test", "", "", parent.ID, fsstore.FanoutContract{
		LeafID: "term-leaf-1",
	})
	if err != nil {
		t.Fatalf("SpawnQuest: %v", err)
	}

	// Drain any events from spawn phase before subscribing, to avoid false positives.
	time.Sleep(200 * time.Millisecond)

	// Subscribe AFTER spawn has settled.
	ch, unsubscribe := eng.bus.Subscribe()
	defer unsubscribe()

	// Make leaf terminal via SaveQuest (writes fanout_leaf_done + fanout_summary on root thread).
	leafQ, err := qs.LoadQuest(leaf)
	if err != nil {
		t.Fatalf("LoadQuest leaf: %v", err)
	}
	leafQ.Status = model.QuestStatusSuccess
	leafQ.FinalComment = "unique-marker-terminal-test-xyz"
	if err := qs.SaveQuest(leafQ); err != nil {
		t.Fatalf("SaveQuest leaf terminal: %v", err)
	}

	// Verify root thread posts exist immediately after SaveQuest.
	rootPosts, err := qs.LoadThreadPosts(parent.ID)
	if err != nil {
		t.Fatalf("LoadThreadPosts root: %v", err)
	}
	var hasLeafDone, hasSummary bool
	for _, p := range rootPosts {
		if p.Kind == "fanout_leaf_done" {
			hasLeafDone = true
		}
		if p.Kind == "fanout_summary" {
			hasSummary = true
		}
	}
	if !hasLeafDone {
		t.Fatal("root thread must have fanout_leaf_done post after SaveQuest")
	}
	if !hasSummary {
		t.Fatal("root thread must have fanout_summary post after SaveQuest")
	}

	// Publish the terminal event through publishRecorded — this is the central path
	// that all terminal events go through (via Engine.publish → publishRecorded →
	// publishRecordedEvent). The hook in publishRecordedEvent should now fire.
	leafPayload := map[string]any{"comment": "unique-marker-terminal-test-xyz", "status": "success"}
	eng.publishRecorded(leaf, "", events.EvtQuestSuccess, leafPayload)

	// Wait for root quest.note event with matching leaf_qid on the bus.
	deadline := time.After(3 * time.Second)
	var rootEventReceived bool
	for {
		select {
		case ev := <-ch:
			if ev.QuestID != parent.ID || ev.Type != events.EvtQuestNote {
				continue
			}
			if len(ev.Payload) == 0 {
				continue
			}
			var payload map[string]any
			if err := json.Unmarshal(ev.Payload, &payload); err != nil {
				continue
			}
			// Must match our specific leaf, not some other root note.
			leafQIDRaw, ok := payload["leaf_qid"]
			if !ok {
				continue
			}
			leafQIDStr, ok := leafQIDRaw.(string)
			if !ok || leafQIDStr != leaf {
				continue
			}
			// Must be kind=fanout_leaf_done.
			kindRaw, ok := payload["kind"]
			if !ok {
				t.Fatal("root event payload missing 'kind'")
			}
			if kindStr, ok := kindRaw.(string); !ok || kindStr != "fanout_leaf_done" {
				t.Fatalf("root event payload kind = %v, want fanout_leaf_done", kindRaw)
			}
			// Verify comment propagated from original event payload.
			if commentRaw, ok := payload["comment"]; ok {
				if commentStr, ok := commentRaw.(string); !ok || commentStr != "unique-marker-terminal-test-xyz" {
					t.Fatalf("root event payload comment = %v, want unique-marker-terminal-test-xyz", commentRaw)
				}
			}
			rootEventReceived = true

			// Ordering proof: at the moment the root event arrives,
			// the root thread posts must already exist.
			postsNow, readErr := qs.LoadThreadPosts(parent.ID)
			if readErr != nil {
				t.Fatalf("LoadThreadPosts at event arrival: %v", readErr)
			}
			var leafDoneNow, summaryNow bool
			for _, p := range postsNow {
				if p.Kind == "fanout_leaf_done" {
					leafDoneNow = true
				}
				if p.Kind == "fanout_summary" {
					summaryNow = true
				}
			}
			if !leafDoneNow || !summaryNow {
				t.Fatalf("ordering violation: root event arrived but posts missing (leaf_done=%v summary=%v)", leafDoneNow, summaryNow)
			}
			goto done
		case <-deadline:
			t.Fatal("timeout waiting for root quest.note event with matching leaf_qid")
		}
	}
done:
	if !rootEventReceived {
		t.Fatal("root quest.note event was not received")
	}

	// Verify the root event was also persisted to root quest's event log.
	rootEvents, err := qs.ReadEvents(parent.ID, 0)
	if err != nil {
		t.Fatalf("ReadEvents root: %v", err)
	}
	var foundRootNote bool
	for _, ev := range rootEvents {
		if ev.Type == string(events.EvtQuestNote) && ev.QuestID == parent.ID {
			foundRootNote = true
			break
		}
	}
	if !foundRootNote {
		t.Fatal("root quest event log must contain quest.note after leaf terminal")
	}
}

func TestLeafTerminal_NonFanout_NoRootEvent(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()
	qs := fsstore.NewQuestStore(eng.root)

	// Create a standalone quest (no fanout group).
	q, err := eng.CreateQuest(ctx, "standalone quest for terminal test", model.QuestTypeExecute, "")
	if err != nil {
		t.Fatalf("CreateQuest: %v", err)
	}

	// Let initial events settle.
	time.Sleep(200 * time.Millisecond)

	ch, unsubscribe := eng.bus.Subscribe()
	defer unsubscribe()

	// Make it terminal.
	qMeta, err := qs.LoadQuest(q.ID)
	if err != nil {
		t.Fatalf("LoadQuest: %v", err)
	}
	qMeta.Status = model.QuestStatusSuccess
	if err := qs.SaveQuest(qMeta); err != nil {
		t.Fatalf("SaveQuest: %v", err)
	}

	// Publish through central path.
	eng.publishRecorded(q.ID, "", events.EvtQuestSuccess, map[string]any{"comment": "done"})

	// Collect events for 500ms — should only see the quest's own success event,
	// no extra quest.note events for this quest.
	deadline := time.After(500 * time.Millisecond)
	var extraNoteEvents int
	for {
		select {
		case ev := <-ch:
			if ev.Type == events.EvtQuestNote && ev.QuestID == q.ID {
				extraNoteEvents++
			}
		case <-deadline:
			if extraNoteEvents != 0 {
				t.Fatalf("non-fanout terminal quest should NOT produce extra quest.note events, got %d", extraNoteEvents)
			}
			return
		}
	}
}

func TestFirstLeafWithMergeContract_WritesMergePost(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()
	qs := fsstore.NewQuestStore(eng.root)

	parent, err := eng.CreateQuest(ctx, "parent with merge contract", model.QuestTypeExecute, "")
	if err != nil {
		t.Fatalf("CreateQuest parent: %v", err)
	}

	// Spawn leaf with merge contract.
	leafID, err := eng.SpawnQuest(ctx, "leaf with merge strategy", "", "", parent.ID, fsstore.FanoutContract{
		LeafID:           "leaf-merge-1",
		MergeStrategy:    "single_leaf",
		MergeOwnerLeafID: "leaf-merge-1",
	})
	if err != nil {
		t.Fatalf("SpawnQuest: %v", err)
	}

	// Verify merge post exists on root thread.
	parentAfter, err := eng.GetQuest(parent.ID)
	if err != nil {
		t.Fatalf("GetQuest parent: %v", err)
	}
	posts, err := qs.LoadThreadPosts(parentAfter.ID)
	if err != nil {
		t.Fatalf("LoadThreadPosts: %v", err)
	}
	var mergePost *fsstore.ThreadPost
	for i := range posts {
		if posts[i].Kind == "fanout_merge_decision" {
			mergePost = &posts[i]
			break
		}
	}
	if mergePost == nil {
		t.Fatal("root thread should have fanout_merge_decision post after spawn with merge contract")
	}
	expectedID := "sys_fanout_merge_" + parentAfter.GroupID
	if mergePost.PostID != expectedID {
		t.Errorf("merge post_id = %s, want %s", mergePost.PostID, expectedID)
	}
	if !strings.Contains(mergePost.Content, "single_leaf") {
		t.Errorf("merge post content should mention strategy, got: %s", mergePost.Content)
	}

	// Idempotent: calling the helper again should not create duplicate.
	leaf, err := eng.GetQuest(leafID)
	if err != nil {
		t.Fatalf("GetQuest leaf: %v", err)
	}
	if err := eng.appendFanoutMergeDecisionPost(parentAfter, leaf, parentAfter.GroupID); err != nil {
		t.Fatalf("second appendFanoutMergeDecisionPost: %v", err)
	}
	posts2, _ := qs.LoadThreadPosts(parentAfter.ID)
	count := 0
	for _, p := range posts2 {
		if p.Kind == "fanout_merge_decision" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected 1 merge_decision post after second call, got %d", count)
	}
}

func TestSpawnQuest_NoMergeContract_NoMergePost(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()
	qs := fsstore.NewQuestStore(eng.root)

	parent, err := eng.CreateQuest(ctx, "parent no merge", model.QuestTypeExecute, "")
	if err != nil {
		t.Fatalf("CreateQuest parent: %v", err)
	}

	// Spawn leaf WITHOUT merge contract.
	_, err = eng.SpawnQuest(ctx, "leaf without merge", "", "", parent.ID, fsstore.FanoutContract{
		LeafID: "leaf-nomerge-1",
	})
	if err != nil {
		t.Fatalf("SpawnQuest: %v", err)
	}

	parentAfter, err := eng.GetQuest(parent.ID)
	if err != nil {
		t.Fatalf("GetQuest parent: %v", err)
	}
	posts, _ := qs.LoadThreadPosts(parentAfter.ID)
	for _, p := range posts {
		if p.Kind == "fanout_merge_decision" {
			t.Fatal("leaf without merge contract should NOT produce fanout_merge_decision post")
		}
	}
}

func TestEngine_ApplyQuestReadOnlyNoOpSuccess(t *testing.T) {
	// v0.3.4 合约：readonly workspace 没有隔离工作区，改动直接落在基准目录，
	// ApplyQuest 是 no-op 成功（Applied=true），不报错——避免 readonly 调研委托卡在 apply_failed。
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	q, err := eng.CreateQuest(ctx, "readonly apply no-op", model.QuestTypeExecute, "")
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	q.Status = model.QuestStatusSuccess
	q.WorkspaceMode = model.WorkspaceReadOnly
	q.DiffChangedFiles = 1
	if err := fsstore.NewQuestStore(eng.root).SaveQuest(q); err != nil {
		t.Fatalf("SaveQuest failed: %v", err)
	}

	warnings, err := eng.ApplyQuest(ctx, q.ID, false)
	if err != nil {
		t.Fatalf("ApplyQuest readonly should be no-op success, got err: %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("readonly apply no-op should not produce warnings, got: %+v", warnings)
	}
	got, err := eng.GetQuest(q.ID)
	if err != nil {
		t.Fatalf("GetQuest failed: %v", err)
	}
	if !got.Applied {
		t.Errorf("readonly apply should mark Applied=true, got %+v", got)
	}
	if got.ApplyStatus != model.ApplyStatusApplied {
		t.Errorf("ApplyStatus = %s, want %s", got.ApplyStatus, model.ApplyStatusApplied)
	}
	if got.ApplyError != "" || got.ApplyFailedAtMs != 0 {
		t.Errorf("readonly apply should clear failure state, got ApplyError=%q ApplyFailedAtMs=%d", got.ApplyError, got.ApplyFailedAtMs)
	}
}

func TestEngine_RecoverPendingApplies(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()
	q, err := eng.CreateQuest(ctx, "pending apply recovery", model.QuestTypeExecute, "")
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	q.Status = model.QuestStatusSuccess
	q.ApplyStatus = model.ApplyStatusPending
	q.Outputs = []fsstore.QuestArtifact{{ID: "out_1", Kind: "doc", StoragePath: "https://example.test/doc"}}
	if err := fsstore.NewQuestStore(eng.root).SaveQuest(q); err != nil {
		t.Fatalf("SaveQuest failed: %v", err)
	}

	recovered, failed, total := eng.RecoverPendingApplies(ctx)
	if total != 1 || recovered != 1 || failed != 0 {
		t.Fatalf("RecoverPendingApplies = recovered:%d failed:%d total:%d", recovered, failed, total)
	}
	got, err := eng.GetQuest(q.ID)
	if err != nil {
		t.Fatalf("GetQuest failed: %v", err)
	}
	if !got.Applied || got.ApplyStatus != model.ApplyStatusApplied || got.ApplyError != "" {
		t.Fatalf("pending apply was not recovered: %+v", got)
	}
}

func TestEngine_CreateQuestWorkspaceModeOption(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	q, err := eng.CreateQuest(ctx, "只读调研", model.QuestTypeDesign, "", CreateQuestOptions{
		WorkspaceMode: model.WorkspaceReadOnly,
	})
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	if q.WorkspaceMode != model.WorkspaceReadOnly {
		t.Fatalf("workspace mode = %s, want readonly", q.WorkspaceMode)
	}
	if q.CreatedBy != "user" {
		t.Fatalf("created_by = %q, want user", q.CreatedBy)
	}
}

func TestEngine_DefaultWorkspaceModeAutoIsStoredEmptyUntilStart(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	q, err := eng.CreateQuest(ctx, "自动工作区", model.QuestTypeExecute, "")
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	if q.WorkspaceMode != "" {
		t.Fatalf("workspace mode before start = %s, want empty auto sentinel", q.WorkspaceMode)
	}
}

func TestEngine_ResolveBlockedUserReviewAndCancel(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()
	qs := fsstore.NewQuestStore(eng.root)

	q, err := eng.CreateQuest(ctx, "blocked review", model.QuestTypeExecute, "")
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	q.WarriorID = "adv_warrior_001"
	q.MageID = "adv_mage_001"
	if err := q.TransitionTo(model.QuestStatusRunning); err != nil {
		t.Fatalf("TransitionTo running failed: %v", err)
	}
	if err := q.TransitionTo(model.QuestStatusBlocked); err != nil {
		t.Fatalf("TransitionTo blocked failed: %v", err)
	}
	if err := qs.SaveQuest(q); err != nil {
		t.Fatalf("SaveQuest failed: %v", err)
	}
	if err := eng.ResolveBlockedQuest(ctx, q.ID, ResumeBlockedOptions{Action: "user_review", Comment: "manual review"}); err != nil {
		t.Fatalf("ResolveBlockedQuest user_review alias failed: %v", err)
	}
	got, _ := eng.GetQuest(q.ID)
	if got.Status != model.QuestStatusUserReview || got.FinalComment != "manual review" {
		t.Fatalf("bad user-review result: %+v", got)
	}
	if got.BlockedReason != "" {
		t.Fatalf("blocked reason should be cleared after user-review resolution: %+v", got)
	}

	cancelQ, err := eng.CreateQuest(ctx, "blocked cancel", model.QuestTypeExecute, "")
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	cancelQ.WarriorID = "adv_warrior_001"
	cancelQ.MageID = "adv_mage_001"
	if err := cancelQ.TransitionTo(model.QuestStatusRunning); err != nil {
		t.Fatalf("TransitionTo running failed: %v", err)
	}
	if err := cancelQ.TransitionTo(model.QuestStatusBlocked); err != nil {
		t.Fatalf("TransitionTo blocked failed: %v", err)
	}
	if err := qs.SaveQuest(cancelQ); err != nil {
		t.Fatalf("SaveQuest failed: %v", err)
	}
	if err := eng.ResolveBlockedQuest(ctx, cancelQ.ID, ResumeBlockedOptions{Action: "cancel", Comment: "stop"}); err != nil {
		t.Fatalf("ResolveBlockedQuest cancel failed: %v", err)
	}
	got, _ = eng.GetQuest(cancelQ.ID)
	if got.Status != model.QuestStatusCancelled || got.CompletedAtMs == 0 {
		t.Fatalf("bad cancel result: %+v", got)
	}
	if got.FinalComment != "stop" {
		t.Fatalf("cancel final comment = %q, want stop", got.FinalComment)
	}
}

func TestAutomation_AutoApplyPassesWithoutUserReview(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	autoCfg := &fsstore.AutomationConfig{
		ID:        "test_auto_apply",
		Name:      "测试自动应用",
		Enabled:   true,
		Trigger:   fsstore.TriggerManual,
		Query:     "自动化测试：实现一个 hello world 函数并加上单元测试，完成后申请评审",
		QuestType: string(model.QuestTypeExecute),
		AutoStart: true,
		AutoApply: true,
	}
	if err := eng.SaveAutomation(autoCfg); err != nil {
		t.Fatalf("SaveAutomation failed: %v", err)
	}

	qid, err := eng.RunAutomation(ctx, "test_auto_apply")
	if err != nil {
		t.Fatalf("RunAutomation failed: %v", err)
	}

	waitForStatus(t, eng, qid, model.QuestStatusSuccess, 10*time.Second)
	q, err := eng.GetQuest(qid)
	if err != nil {
		t.Fatalf("GetQuest failed: %v", err)
	}
	if !q.Applied {
		t.Fatal("auto-applied quest should be marked applied")
	}
	if q.FinalVerdict != model.VerdictPass {
		t.Fatalf("expected final verdict pass, got %s", q.FinalVerdict)
	}
}

func TestEngine_DiscardQuest(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	q, err := eng.CreateQuest(ctx, "写点测试内容", model.QuestTypeExecute, "")
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}

	if err := eng.StartQuest(ctx, q.ID); err != nil {
		t.Fatalf("StartQuest failed: %v", err)
	}

	waitForStatus(t, eng, q.ID, model.QuestStatusSuccess, 10*time.Second)

	// discard 应该能工作
	if err := eng.DiscardQuest(ctx, q.ID, "测试 discard"); err != nil {
		t.Fatalf("DiscardQuest failed: %v", err)
	}
}

func TestAutomation_ListDefaults(t *testing.T) {
	eng, _ := setupTestEngine(t)

	items, err := eng.ListAutomations()
	if err != nil {
		t.Fatalf("ListAutomations failed: %v", err)
	}
	if len(items) < 4 {
		t.Errorf("expected at least 4 default automations, got %d", len(items))
	}
	t.Logf("Found %d automations", len(items))
	for _, a := range items {
		t.Logf("  - %s (%s, enabled=%v)", a.ID, a.Name, a.Enabled)
		if !a.HasTag("official") || !a.HasTag("template") {
			t.Errorf("default automation should be official template: %+v", a)
		}
		if a.ID == "auto_context_refresh" {
			if !a.Enabled || !a.AutoStart || a.AutoApply || a.Priority != 100 {
				t.Errorf("context refresh default policy mismatch: %+v", a)
			}
			continue
		}
		if a.ID == "auto_morning_briefing" || a.ID == "auto_stale_quest_cleanup" || a.ID == "auto_context_decay_alert" || a.ID == "auto_inbox_triage" {
			if !a.Enabled || !a.AutoStart || a.AutoApply {
				t.Errorf("scheduled official automation default policy mismatch: %+v", a)
			}
			continue
		}
		if a.Enabled {
			t.Errorf("default automation template should be disabled by default: %s", a.ID)
		}
	}
	templates, err := eng.ListAutomationTemplates()
	if err != nil {
		t.Fatalf("ListAutomationTemplates failed: %v", err)
	}
	if len(templates) != len(items) {
		t.Fatalf("templates = %d, want %d", len(templates), len(items))
	}
}

func TestAutomation_AuditCompletedQuest(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	done, err := eng.CreateQuest(ctx, "已完成任务", model.QuestTypeExecute, "")
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	done.Status = model.QuestStatusSuccess
	done.FinalVerdict = model.VerdictPass
	done.FinalComment = "已通过"
	if err := fsstore.NewQuestStore(eng.root).SaveQuest(done); err != nil {
		t.Fatalf("SaveQuest failed: %v", err)
	}
	if _, err := eng.SetAutomationEnabled("auto_audit_completed", true); err != nil {
		t.Fatalf("enable audit automation failed: %v", err)
	}
	qid, err := eng.RunAutomation(ctx, "auto_audit_completed")
	if err != nil {
		t.Fatalf("RunAutomation failed: %v", err)
	}
	q, err := eng.GetQuest(qid)
	if err != nil {
		t.Fatalf("GetQuest failed: %v", err)
	}
	if !strings.Contains(q.Query, done.ID) || !strings.Contains(q.Query, "已完成任务") {
		t.Fatalf("audit query should include completed quest context:\n%s", q.Query)
	}
}

func TestEventAutomation_PerQuestCursorTriageNote(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	cfg := &fsstore.AutomationConfig{
		ID:      "event_triage_test",
		Name:    "事件分类测试",
		Enabled: true,
		Trigger: fsstore.TriggerEvent,
		Event:   string(events.EvtQuestSuccess),
		Tags:    []string{"triage"},
	}
	if err := eng.SaveAutomation(cfg); err != nil {
		t.Fatalf("SaveAutomation failed: %v", err)
	}
	q, err := eng.CreateQuest(ctx, "event automation", model.QuestTypeExecute, "")
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	qs := fsstore.NewQuestStore(eng.root)
	if err := qs.AppendEvent(q.ID, &fsstore.QuestEventRow{Type: string(events.EvtQuestSuccess)}); err != nil {
		t.Fatalf("AppendEvent failed: %v", err)
	}

	if err := eng.ProcessQuestEventAutomations(ctx, q.ID); err != nil {
		t.Fatalf("ProcessQuestEventAutomations failed: %v", err)
	}
	if err := eng.ProcessQuestEventAutomations(ctx, q.ID); err != nil {
		t.Fatalf("ProcessQuestEventAutomations second run failed: %v", err)
	}
	rows, err := qs.ReadEvents(q.ID, 0)
	if err != nil {
		t.Fatalf("ReadEvents failed: %v", err)
	}
	notes := 0
	for _, row := range rows {
		if row.Type == string(events.EvtQuestNote) {
			notes++
		}
	}
	if notes != 1 {
		t.Fatalf("event automation should write one note, got %d rows=%+v", notes, rows)
	}
	state, err := eng.root.GetAutomationState(cfg.ID, q.ID)
	if err != nil {
		t.Fatalf("GetAutomationState failed: %v", err)
	}
	if state.LastEventID != 2 || state.RunCount != 1 {
		t.Fatalf("bad automation state: %+v", state)
	}
}

func TestEnginePublishCarriesPersistedEventID(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ch, unsubscribe := eng.bus.Subscribe()
	defer unsubscribe()

	q, err := eng.CreateQuest(context.Background(), "event id", model.QuestTypeExecute, "")
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	deadline := time.After(2 * time.Second)
	for {
		select {
		case ev := <-ch:
			if ev.Type != events.EvtQuestCreated || ev.QuestID != q.ID {
				continue
			}
			if ev.ID == 0 {
				t.Fatalf("published event should carry persisted id: %+v", ev)
			}
			if ev.GlobalID == 0 {
				t.Fatalf("published event should carry global id: %+v", ev)
			}
			rows, err := fsstore.NewQuestStore(eng.root).ReadEvents(q.ID, 0)
			if err != nil {
				t.Fatalf("ReadEvents failed: %v", err)
			}
			if len(rows) == 0 || rows[0].ID != ev.ID {
				t.Fatalf("persisted event id mismatch rows=%+v ev=%+v", rows, ev)
			}
			globalRows, err := eng.root.ReadGlobalEvents(0)
			if err != nil {
				t.Fatalf("ReadGlobalEvents failed: %v", err)
			}
			if len(globalRows) == 0 || globalRows[0].GlobalID != ev.GlobalID || globalRows[0].QuestID != q.ID {
				t.Fatalf("global event mismatch rows=%+v ev=%+v", globalRows, ev)
			}
			return
		case <-deadline:
			t.Fatal("timeout waiting for quest.created event")
		}
	}
}

func TestGlobalEventAutomationUsesGlobalCursor(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	cfg := &fsstore.AutomationConfig{
		ID:         "global_event_test",
		Name:       "全局事件测试",
		Enabled:    true,
		Trigger:    fsstore.TriggerEvent,
		Event:      string(events.EvtQuestSuccess),
		EventScope: "global",
		Tags:       []string{"notify"},
	}
	if err := eng.SaveAutomation(cfg); err != nil {
		t.Fatalf("SaveAutomation failed: %v", err)
	}
	q, err := eng.CreateQuest(ctx, "global event automation", model.QuestTypeExecute, "")
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	eng.publish(q.ID, "", events.EvtQuestSuccess, map[string]any{"comment": "done"})
	if err := eng.ProcessGlobalEventAutomations(ctx); err != nil {
		t.Fatalf("ProcessGlobalEventAutomations failed: %v", err)
	}
	if err := eng.ProcessGlobalEventAutomations(ctx); err != nil {
		t.Fatalf("ProcessGlobalEventAutomations second run failed: %v", err)
	}
	state, err := eng.root.GetAutomationState(cfg.ID, "__global__")
	if err != nil {
		t.Fatalf("GetAutomationState failed: %v", err)
	}
	if state.LastGlobalID == 0 || state.RunCount != 1 {
		t.Fatalf("bad global state: %+v", state)
	}
	rows, err := fsstore.NewQuestStore(eng.root).ReadEvents(q.ID, 0)
	if err != nil {
		t.Fatalf("ReadEvents failed: %v", err)
	}
	automationEvents := 0
	for _, row := range rows {
		if row.Type == "automation.event" {
			automationEvents++
		}
	}
	if automationEvents != 1 {
		t.Fatalf("global event automation should publish once, got %d rows=%+v", automationEvents, rows)
	}
}

func TestAutomation_RunNoAutoStart_Inbox(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	// 创建一个 auto_start=false 的 automation
	autoCfg := &fsstore.AutomationConfig{
		ID:        "test_inbox",
		Name:      "测试收件箱",
		Enabled:   true,
		Trigger:   fsstore.TriggerManual,
		Query:     "自动化测试：进收件箱",
		QuestType: string(model.QuestTypeExecute),
		AutoStart: false,
	}
	if err := eng.SaveAutomation(autoCfg); err != nil {
		t.Fatalf("SaveAutomation failed: %v", err)
	}

	// 运行 automation
	qid, err := eng.RunAutomation(ctx, "test_inbox")
	if err != nil {
		t.Fatalf("RunAutomation failed: %v", err)
	}
	t.Logf("Automation created quest: qid=%s", qid)

	// 验证 quest 在 pending 状态（不自动启动）
	q, _ := eng.GetQuest(qid)
	if q.Status != model.QuestStatusPending {
		t.Errorf("expected pending status for inbox quest, got %s", q.Status)
	}
	if eng.IsQuestRunning(qid) {
		t.Error("quest should NOT be running when auto_start=false")
	}

	// 验证在收件箱中
	inbox, err := eng.ListInbox()
	if err != nil {
		t.Fatalf("ListInbox failed: %v", err)
	}
	found := false
	for _, item := range inbox {
		if item.ID == qid {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("quest %s should be in inbox", qid)
	}
	if q.TriageMode != "candidate" {
		t.Errorf("automation candidate quest should expose triage mode candidate, got %q", q.TriageMode)
	}
	t.Logf("Inbox has %d items", len(inbox))
}

func TestAutomationTriageNoFindingArchivesWithoutQuest(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()
	autoCfg := &fsstore.AutomationConfig{
		ID:        "test_triage_empty",
		Name:      "empty triage",
		Enabled:   true,
		Trigger:   fsstore.TriggerManual,
		Query:     "triage",
		QuestType: string(model.QuestTypeDesign),
		AutoStart: true,
		Tags:      []string{"triage"},
	}
	if err := eng.SaveAutomation(autoCfg); err != nil {
		t.Fatalf("SaveAutomation failed: %v", err)
	}

	qid, err := eng.RunAutomation(ctx, autoCfg.ID)
	if err != nil {
		t.Fatalf("RunAutomation failed: %v", err)
	}
	// v0.5.4: no_finding 现在创建轻量 quest 进 Feed（RequireAgent=false，不启动执行），
	// 让 automation 像账号发"无发现"动态。归档仍记录。
	if qid == "" {
		t.Fatalf("no-finding triage should create a lightweight quest for Feed, got empty qid")
	}
	q, err := eng.GetQuest(qid)
	if err != nil {
		t.Fatalf("GetQuest failed: %v", err)
	}
	if q.Status != model.QuestStatusPending && q.Status != model.QuestStatusSuccess {
		t.Fatalf("no-finding quest should not be started, status=%s", q.Status)
	}
	// 验证 no_finding thread post 投影
	posts, err := fsstore.NewQuestStore(eng.root).LoadThreadPosts(qid)
	if err != nil {
		t.Fatalf("LoadThreadPosts failed: %v", err)
	}
	var foundNoFinding bool
	for _, p := range posts {
		if p.Kind == "automation_no_finding" && p.AuthorRole == model.PostRoleAutomation {
			foundNoFinding = true
			break
		}
	}
	if !foundNoFinding {
		t.Fatalf("no_finding thread post not projected, posts: %+v", posts)
	}
	items, err := eng.root.LoadAutomationDiscoveryArchive()
	if err != nil {
		t.Fatalf("LoadAutomationDiscoveryArchive failed: %v", err)
	}
	if len(items) == 0 || items[len(items)-1].AutomationID != autoCfg.ID || items[len(items)-1].Outcome != "no_finding" {
		t.Fatalf("bad discovery archive: %+v", items)
	}
}

func TestPickRunnableAdventurerAvoidsTierCWhenTierBAvailable(t *testing.T) {
	eng, _ := setupTestEngine(t)
	if err := eng.root.SaveAgent(&fsstore.AgentConfig{
		Name:         "tier_c_agent",
		Type:         model.AgentTypeCLI,
		Command:      "pi",
		DefaultModel: "auto",
		Enabled:      true,
	}); err != nil {
		t.Fatalf("SaveAgent tier_c failed: %v", err)
	}
	if err := eng.root.SaveAgent(&fsstore.AgentConfig{
		Name:         "tier_b_agent",
		Type:         model.AgentTypeACP,
		Command:      "agent",
		DefaultModel: "auto",
		Enabled:      true,
	}); err != nil {
		t.Fatalf("SaveAgent tier_b failed: %v", err)
	}
	if err := eng.root.SaveAdventurer(&fsstore.AdventurerFile{
		ID:          "adv_tier_c",
		Name:        "Tier C",
		Class:       model.ClassWarrior,
		Status:      model.AdventurerActive,
		Agent:       "tier_c_agent",
		Level:       10,
		CreatedAtMs: 1,
	}); err != nil {
		t.Fatalf("SaveAdventurer tier_c failed: %v", err)
	}
	if err := eng.root.SaveAdventurer(&fsstore.AdventurerFile{
		ID:          "adv_tier_b",
		Name:        "Tier B",
		Class:       model.ClassWarrior,
		Status:      model.AdventurerActive,
		Agent:       "tier_b_agent",
		Level:       1,
		CreatedAtMs: 2,
	}); err != nil {
		t.Fatalf("SaveAdventurer tier_b failed: %v", err)
	}
	eng.RegisterExecutor("tier_c_agent", executor.NewPiExecutor(executor.PiOpt{BinPath: "pi"}))
	eng.RegisterExecutor("tier_b_agent", executor.NewACPExecutor(executor.ACPOpt{Command: "agent"}))

	adv, err := eng.pickRunnableAdventurer("", model.ClassWarrior)
	if err != nil {
		t.Fatalf("pickRunnableAdventurer failed: %v", err)
	}
	if adv.ID != "adv_tier_b" {
		t.Fatalf("default pick = %s, want adv_tier_b", adv.ID)
	}
}

func TestInbox_AcceptLegacyContextRefreshStartsQuest(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()
	autoStart := false
	autoApply := false
	if _, err := eng.UpdateAutomation("auto_context_refresh", AutomationUpdate{
		AutoStart: &autoStart,
		AutoApply: &autoApply,
	}); err != nil {
		t.Fatalf("UpdateAutomation failed: %v", err)
	}

	qid, err := eng.RunAutomation(ctx, "auto_context_refresh")
	if err != nil {
		t.Fatalf("RunAutomation failed: %v", err)
	}
	q, err := eng.GetQuest(qid)
	if err != nil {
		t.Fatalf("GetQuest failed: %v", err)
	}
	if q.Status != model.QuestStatusPending {
		t.Fatalf("legacy context refresh should first be pending inbox item, got %s", q.Status)
	}
	if q.WorkspaceMode != model.WorkspaceReadOnly {
		t.Fatalf("context refresh automation should default to readonly workspace, got %s", q.WorkspaceMode)
	}

	if err := eng.AcceptInboxItem(ctx, qid); err != nil {
		t.Fatalf("AcceptInboxItem failed: %v", err)
	}
	if !eng.IsQuestRunning(qid) {
		t.Fatal("accepted context refresh quest should be running")
	}
	// workspace 准备异步化后，status 从 pending → running/success 有短暂延迟。
	// 等 status 离开 pending 再验证 inbox。
	waitForNotStatus(t, eng, qid, model.QuestStatusPending, 10*time.Second)
	inbox, err := eng.ListInbox()
	if err != nil {
		t.Fatalf("ListInbox failed: %v", err)
	}
	for _, item := range inbox {
		if item.ID == qid {
			t.Fatalf("accepted context refresh quest %s should leave inbox", qid)
		}
	}
	waitForStatus(t, eng, qid, model.QuestStatusSuccess, 10*time.Second)
}

func TestRecoverAutoStartInboxItems_StartsOfficialPolicyPendingQuest(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	cfg, err := eng.GetAutomation("auto_context_refresh")
	if err != nil {
		t.Fatalf("GetAutomation failed: %v", err)
	}
	cfg.AutoStart = false
	cfg.AutoApply = false
	cfg.RuntimePolicyUserOverride = true
	if err := eng.SaveAutomation(cfg); err != nil {
		t.Fatalf("SaveAutomation stale policy failed: %v", err)
	}
	qid, err := eng.RunAutomation(ctx, "auto_context_refresh")
	if err != nil {
		t.Fatalf("RunAutomation failed: %v", err)
	}
	q, err := eng.GetQuest(qid)
	if err != nil {
		t.Fatalf("GetQuest failed: %v", err)
	}
	if q.Status != model.QuestStatusPending {
		t.Fatalf("precondition: quest should be pending, got %s", q.Status)
	}

	cfg.AutoStart = true
	cfg.AutoApply = false
	cfg.RuntimePolicyUserOverride = false
	cfg.OfficialPolicyVersion = fsstore.CurrentOfficialAutomationPolicyVersion
	if err := eng.SaveAutomation(cfg); err != nil {
		t.Fatalf("SaveAutomation migrated policy failed: %v", err)
	}

	recovered, failed, total := eng.RecoverAutoStartInboxItems(ctx)
	if recovered != 1 || failed != 0 || total != 1 {
		t.Fatalf("RecoverAutoStartInboxItems = recovered:%d failed:%d total:%d, want 1/0/1", recovered, failed, total)
	}
	if !eng.IsQuestRunning(qid) {
		t.Fatal("recovered auto-start inbox quest should be running")
	}
	waitForStatus(t, eng, qid, model.QuestStatusSuccess, 10*time.Second)
}

func TestRecoverAutoStartInboxItems_LeavesUserOverridePending(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	cfg, err := eng.GetAutomation("auto_context_refresh")
	if err != nil {
		t.Fatalf("GetAutomation failed: %v", err)
	}
	cfg.AutoStart = false
	cfg.AutoApply = false
	cfg.RuntimePolicyUserOverride = true
	cfg.OfficialPolicyVersion = fsstore.CurrentOfficialAutomationPolicyVersion
	if err := eng.SaveAutomation(cfg); err != nil {
		t.Fatalf("SaveAutomation failed: %v", err)
	}
	qid, err := eng.RunAutomation(ctx, "auto_context_refresh")
	if err != nil {
		t.Fatalf("RunAutomation failed: %v", err)
	}

	recovered, failed, total := eng.RecoverAutoStartInboxItems(ctx)
	if recovered != 0 || failed != 0 || total != 0 {
		t.Fatalf("RecoverAutoStartInboxItems = recovered:%d failed:%d total:%d, want 0/0/0", recovered, failed, total)
	}
	q, err := eng.GetQuest(qid)
	if err != nil {
		t.Fatalf("GetQuest failed: %v", err)
	}
	if q.Status != model.QuestStatusPending {
		t.Fatalf("user override pending quest should remain pending, got %s", q.Status)
	}
}

func TestInbox_Edit(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	autoCfg := &fsstore.AutomationConfig{
		ID:        "test_inbox_edit",
		Name:      "测试编辑",
		Enabled:   true,
		Query:     "旧描述",
		QuestType: string(model.QuestTypeExecute),
		AutoStart: false,
	}
	if err := eng.SaveAutomation(autoCfg); err != nil {
		t.Fatalf("SaveAutomation failed: %v", err)
	}
	qid, err := eng.RunAutomation(ctx, "test_inbox_edit")
	if err != nil {
		t.Fatalf("RunAutomation failed: %v", err)
	}
	query := "新描述"
	questType := string(model.QuestTypeDesign)
	workDir := t.TempDir()
	workspaceMode := model.WorkspaceCopy
	q, err := eng.UpdateInboxItem(qid, InboxUpdate{
		Query:         &query,
		QuestType:     &questType,
		WorkDir:       &workDir,
		WorkspaceMode: &workspaceMode,
	})
	if err != nil {
		t.Fatalf("UpdateInboxItem failed: %v", err)
	}
	if q.Query != query || q.Type != model.QuestTypeDesign || q.BaseWorkingDir != workDir || q.WorkspaceMode != workspaceMode {
		t.Fatalf("inbox item not updated: %+v", q)
	}
}

func TestInbox_Reject(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	// 创建 automation 并运行 → quest 进 inbox
	autoCfg := &fsstore.AutomationConfig{
		ID:        "test_reject",
		Name:      "测试拒绝",
		Enabled:   true,
		Query:     "收件箱拒绝测试",
		QuestType: string(model.QuestTypeExecute),
		AutoStart: false,
	}
	if err := eng.SaveAutomation(autoCfg); err != nil {
		t.Fatalf("SaveAutomation failed: %v", err)
	}

	qid, err := eng.RunAutomation(ctx, "test_reject")
	if err != nil {
		t.Fatalf("RunAutomation failed: %v", err)
	}

	// 拒绝收件箱项
	if err := eng.RejectInboxItem(qid, "不需要这个"); err != nil {
		t.Fatalf("RejectInboxItem failed: %v", err)
	}

	// 验证 quest 已取消
	q, _ := eng.GetQuest(qid)
	if q.Status != model.QuestStatusCancelled {
		t.Errorf("expected cancelled status, got %s", q.Status)
	}
	if q.FinalComment != "收件箱拒绝: 不需要这个" {
		t.Errorf("unexpected final comment: %q", q.FinalComment)
	}

	// 验证不在收件箱中
	inbox, _ := eng.ListInbox()
	for _, item := range inbox {
		if item.ID == qid {
			t.Errorf("quest %s should NOT be in inbox after reject", qid)
			break
		}
	}
	t.Logf("Inbox has %d items after reject", len(inbox))
}

func TestStopQuestCancelsPendingAutomationInboxItem(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	autoCfg := &fsstore.AutomationConfig{
		ID:        "test_cancel_pending",
		Name:      "测试取消 pending",
		Enabled:   true,
		Query:     "收件箱取消测试",
		QuestType: string(model.QuestTypeExecute),
		AutoStart: false,
	}
	if err := eng.SaveAutomation(autoCfg); err != nil {
		t.Fatalf("SaveAutomation failed: %v", err)
	}
	qid, err := eng.RunAutomation(ctx, autoCfg.ID)
	if err != nil {
		t.Fatalf("RunAutomation failed: %v", err)
	}

	if err := eng.StopQuest(qid, "old context automation used copy workspace"); err != nil {
		t.Fatalf("StopQuest pending automation failed: %v", err)
	}
	q, err := eng.GetQuest(qid)
	if err != nil {
		t.Fatalf("GetQuest failed: %v", err)
	}
	if q.Status != model.QuestStatusCancelled {
		t.Fatalf("expected cancelled status, got %s", q.Status)
	}
	if q.FinalComment != "收件箱拒绝: old context automation used copy workspace" {
		t.Fatalf("unexpected final comment: %q", q.FinalComment)
	}
}

func TestAutomation_Disabled(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	autoCfg := &fsstore.AutomationConfig{
		ID:        "test_disabled",
		Name:      "已禁用的 automation",
		Enabled:   false,
		Query:     "不应该运行",
		QuestType: string(model.QuestTypeExecute),
	}
	if err := eng.SaveAutomation(autoCfg); err != nil {
		t.Fatalf("SaveAutomation failed: %v", err)
	}

	_, err := eng.RunAutomation(ctx, "test_disabled")
	if err == nil {
		t.Error("expected error when running disabled automation")
	}
	t.Logf("Disabled automation correctly rejected: %v", err)
}

func TestAutomation_EnableDisable(t *testing.T) {
	eng, _ := setupTestEngine(t)

	autoCfg := &fsstore.AutomationConfig{
		ID:        "test_toggle",
		Name:      "开关测试",
		Enabled:   false,
		Query:     "toggle",
		QuestType: string(model.QuestTypeExecute),
	}
	if err := eng.SaveAutomation(autoCfg); err != nil {
		t.Fatalf("SaveAutomation failed: %v", err)
	}
	item, err := eng.SetAutomationEnabled("test_toggle", true)
	if err != nil {
		t.Fatalf("enable failed: %v", err)
	}
	if !item.Enabled {
		t.Fatal("automation should be enabled")
	}
	item, err = eng.SetAutomationEnabled("test_toggle", false)
	if err != nil {
		t.Fatalf("disable failed: %v", err)
	}
	if item.Enabled {
		t.Fatal("automation should be disabled")
	}
}

func TestAutomation_OfficialQuestMarker(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	if _, err := eng.SetAutomationEnabled("auto_daily_lint", true); err != nil {
		t.Fatalf("enable official template failed: %v", err)
	}
	qid, err := eng.RunAutomation(ctx, "auto_daily_lint")
	if err != nil {
		t.Fatalf("RunAutomation failed: %v", err)
	}
	q, err := eng.GetQuest(qid)
	if err != nil {
		t.Fatalf("GetQuest failed: %v", err)
	}
	if !eng.root.IsQuestOfficial(q) {
		t.Fatalf("official template should create official quest: %+v", q)
	}
}

func TestAutomation_Update(t *testing.T) {
	eng, _ := setupTestEngine(t)

	autoCfg := &fsstore.AutomationConfig{
		ID:        "test_update",
		Name:      "旧名字",
		Enabled:   true,
		Query:     "old",
		QuestType: string(model.QuestTypeExecute),
		AutoStart: false,
	}
	if err := eng.SaveAutomation(autoCfg); err != nil {
		t.Fatalf("SaveAutomation failed: %v", err)
	}
	name := "新名字"
	query := "new query"
	questType := string(model.QuestTypeDesign)
	intensity := string(model.QuestIntensityDeep)
	autoStart := true
	allowHOTL := true
	warriorID := "warrior_test"
	mageID := "mage_test"
	cron := "daily9"
	priority := 42
	trustTier := fsstore.TrustTier2
	trustLocked := true
	item, err := eng.UpdateAutomation("test_update", AutomationUpdate{
		Name:            &name,
		Query:           &query,
		QuestType:       &questType,
		Intensity:       &intensity,
		WarriorID:       &warriorID,
		MageID:          &mageID,
		Cron:            &cron,
		Priority:        &priority,
		AutoStart:       &autoStart,
		AllowHOTL:       &allowHOTL,
		TrustTier:       &trustTier,
		TrustTierLocked: &trustLocked,
	})
	if err != nil {
		t.Fatalf("UpdateAutomation failed: %v", err)
	}
	if item.WarriorID != warriorID || item.MageID != mageID || item.Cron != "0 9 * * *" || item.Priority != priority {
		t.Fatalf("automation not updated: %+v", item)
	}
	if item.TrustTier != fsstore.TrustTier2 || !item.TrustTierLocked {
		t.Fatalf("automation trust tier not updated: %+v", item)
	}
	state, err := eng.root.GetAutomationTrustState("test_update")
	if err != nil {
		t.Fatalf("GetAutomationTrustState failed: %v", err)
	}
	if state.Tier != fsstore.TrustTier2 || !state.Locked {
		t.Fatalf("automation trust state not locked: %+v", state)
	}
}

func TestAutomation_RunUsesIntensity(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	autoCfg := &fsstore.AutomationConfig{
		ID:        "test_intensity",
		Name:      "强度测试",
		Enabled:   true,
		Query:     "强度测试",
		QuestType: string(model.QuestTypeExecute),
		Intensity: string(model.QuestIntensityAdversarial),
		AutoStart: false,
	}
	if err := eng.SaveAutomation(autoCfg); err != nil {
		t.Fatalf("SaveAutomation failed: %v", err)
	}
	qid, err := eng.RunAutomation(ctx, "test_intensity")
	if err != nil {
		t.Fatalf("RunAutomation failed: %v", err)
	}
	q, err := eng.GetQuest(qid)
	if err != nil {
		t.Fatalf("GetQuest failed: %v", err)
	}
	if q.Intensity != model.QuestIntensityAdversarial || q.MaxRework != 5 || q.MaxTurnsPerPhaseOverride != 100 {
		t.Fatalf("automation intensity budget mismatch: %+v", q)
	}
}

func TestAutomation_RunCountAndLastRun(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	autoCfg := &fsstore.AutomationConfig{
		ID:        "test_run_count",
		Name:      "运行计数测试",
		Enabled:   true,
		Query:     "计数测试",
		QuestType: string(model.QuestTypeExecute),
		AutoStart: false,
	}
	if err := eng.SaveAutomation(autoCfg); err != nil {
		t.Fatalf("SaveAutomation failed: %v", err)
	}

	// 运行两次
	qid1, err := eng.RunAutomation(ctx, "test_run_count")
	if err != nil {
		t.Fatalf("first RunAutomation failed: %v", err)
	}
	qid2, err := eng.RunAutomation(ctx, "test_run_count")
	if err != nil {
		t.Fatalf("second RunAutomation failed: %v", err)
	}
	if qid1 == qid2 {
		t.Error("each run should create a new quest")
	}

	// 验证运行计数
	cfg, err := eng.GetAutomation("test_run_count")
	if err != nil {
		t.Fatalf("GetAutomation failed: %v", err)
	}
	if cfg.RunCount != 2 {
		t.Errorf("expected run_count=2, got %d", cfg.RunCount)
	}
	if cfg.LastRunMs == 0 {
		t.Error("last_run_ms should be set")
	}
	t.Logf("RunCount=%d, LastRunMs=%d", cfg.RunCount, cfg.LastRunMs)

	// 收件箱里应该有 2 个
	inbox, _ := eng.ListInbox()
	count := 0
	for _, item := range inbox {
		if item.CreatedBy == "automation:test_run_count" {
			count++
		}
	}
	if count != 2 {
		t.Errorf("expected 2 inbox items from this automation, got %d", count)
	}
}

func TestAutomation_RunPublishesAutomationEvent(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()
	ch, unsubscribe := eng.bus.Subscribe()
	defer unsubscribe()

	autoCfg := &fsstore.AutomationConfig{
		ID:        "test_publish_event",
		Name:      "publish event",
		Query:     "x",
		Trigger:   fsstore.TriggerManual,
		Enabled:   true,
		AutoStart: false,
	}
	if err := eng.SaveAutomation(autoCfg); err != nil {
		t.Fatalf("SaveAutomation failed: %v", err)
	}
	qid, err := eng.RunAutomation(ctx, autoCfg.ID)
	if err != nil {
		t.Fatalf("RunAutomation failed: %v", err)
	}

	deadline := time.After(2 * time.Second)
	for {
		select {
		case ev := <-ch:
			if ev.Type == events.EventType("automation.run") && ev.QuestID == qid {
				return
			}
		case <-deadline:
			t.Fatal("automation.run event was not published")
		}
	}
}

func TestEngine_PublishPostStoresSourceEventID(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()
	q, err := eng.CreateQuest(ctx, "source event post", model.QuestTypeExecute, "", CreateQuestOptions{
		WarriorID: "adv_warrior_001",
		MageID:    "adv_mage_001",
	})
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	postID, err := eng.PublishPost(q.ID, "warrior_0", "source event linked post", fsstore.RootPostID(q.ID), "maker_report")
	if err != nil {
		t.Fatalf("PublishPost failed: %v", err)
	}
	posts, err := fsstore.NewQuestStore(eng.root).LoadThreadPosts(q.ID)
	if err != nil {
		t.Fatalf("LoadThreadPosts failed: %v", err)
	}
	for _, post := range posts {
		if post.PostID != postID {
			continue
		}
		if post.SourceEventID == 0 {
			t.Fatalf("published post should link to source event id: %+v", post)
		}
		return
	}
	t.Fatalf("published post %s not found in thread posts: %+v", postID, posts)
}

func TestAutomation_RunProjectsAutomationReply(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	autoCfg := &fsstore.AutomationConfig{
		ID:         "test_feed_actor",
		Name:       "Feed Actor",
		Query:      "automation feed actor",
		Trigger:    fsstore.TriggerManual,
		Enabled:    true,
		AutoStart:  false,
		TriageMode: "candidate",
	}
	if err := eng.SaveAutomation(autoCfg); err != nil {
		t.Fatalf("SaveAutomation failed: %v", err)
	}
	qid, err := eng.RunAutomation(ctx, autoCfg.ID)
	if err != nil {
		t.Fatalf("RunAutomation failed: %v", err)
	}
	posts, err := fsstore.NewQuestStore(eng.root).LoadThreadPosts(qid)
	if err != nil {
		t.Fatalf("LoadThreadPosts failed: %v", err)
	}
	var automationReply *fsstore.ThreadPost
	for i := range posts {
		if posts[i].AuthorRole == model.PostRoleAutomation && posts[i].Kind == "automation_run" {
			automationReply = &posts[i]
			break
		}
	}
	if automationReply == nil {
		t.Fatalf("automation.run should project an automation reply, posts=%+v", posts)
	}
	if automationReply.ParentReplyID != fsstore.RootPostID(qid) {
		t.Fatalf("automation reply should target root post: %+v", automationReply)
	}
	if automationReply.SourceEventID == 0 {
		t.Fatalf("automation reply should link source event id: %+v", automationReply)
	}
	if !strings.Contains(automationReply.Content, "Feed Actor") || !strings.Contains(automationReply.Content, "candidate") {
		t.Fatalf("automation reply content should summarize run context: %+v", automationReply)
	}
}

func TestAutomation_FanoutRootPostsUseAutomationAuthor(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	autoCfg := &fsstore.AutomationConfig{
		ID:        "test_fanout_actor",
		Name:      "Fanout Actor",
		Query:     "fanout parent",
		Trigger:   fsstore.TriggerManual,
		Enabled:   true,
		AutoStart: false,
		QuestType: string(model.QuestTypeExecute),
		FanOut: []fsstore.FanOutSpec{
			{LeafID: "leaf-a", Query: "leaf a", OwnershipScopes: []string{"a"}, MergeStrategy: fsstore.MergeStrategySingleLeaf, MergeOwnerLeafID: "leaf-a"},
			{LeafID: "leaf-b", Query: "leaf b", OwnershipScopes: []string{"b"}, MergeStrategy: fsstore.MergeStrategySingleLeaf, MergeOwnerLeafID: "leaf-a"},
		},
	}
	if err := eng.SaveAutomation(autoCfg); err != nil {
		t.Fatalf("SaveAutomation failed: %v", err)
	}
	if _, err := eng.RunAutomation(ctx, autoCfg.ID); err != nil {
		t.Fatalf("RunAutomation failed: %v", err)
	}
	items, err := eng.ListQuests()
	if err != nil {
		t.Fatalf("ListQuests failed: %v", err)
	}
	rootsChecked := 0
	qs := fsstore.NewQuestStore(eng.root)
	for _, q := range items {
		if q == nil || q.GroupID == "" {
			continue
		}
		posts, err := qs.LoadThreadPosts(q.ID)
		if err != nil {
			t.Fatalf("LoadThreadPosts(%s): %v", q.ID, err)
		}
		if len(posts) == 0 || posts[0].AuthorRole != model.PostRoleAutomation {
			t.Fatalf("fanout root post should use automation author, q=%+v posts=%+v", q, posts)
		}
		rootsChecked++
	}
	if rootsChecked != 2 {
		t.Fatalf("checked %d fanout roots, want 2", rootsChecked)
	}
}

func TestAutomationFanOutRejectsBatchOwnershipConflictWithoutPartialCreate(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	autoCfg := &fsstore.AutomationConfig{
		ID:        "test_fanout_contract",
		Name:      "fanout contract",
		Query:     "parent",
		Trigger:   fsstore.TriggerManual,
		Enabled:   true,
		AutoStart: false,
		QuestType: string(model.QuestTypeExecute),
		FanOut: []fsstore.FanOutSpec{
			{
				Query:           "leaf a",
				LeafID:          "leaf-a",
				OwnershipScopes: []string{"internal/server"},
				MergeStrategy:   fsstore.MergeStrategyNoMerge,
			},
			{
				Query:           "leaf b",
				LeafID:          "leaf-b",
				OwnershipScopes: []string{"internal/server"},
				MergeStrategy:   fsstore.MergeStrategyNoMerge,
			},
		},
	}
	if err := eng.SaveAutomation(autoCfg); err != nil {
		t.Fatalf("SaveAutomation failed: %v", err)
	}
	_, err := eng.RunAutomation(ctx, autoCfg.ID)
	if err == nil || !strings.Contains(err.Error(), "ownership scope conflict") {
		t.Fatalf("expected ownership conflict, got %v", err)
	}
	quests, err := eng.ListQuests()
	if err != nil {
		t.Fatalf("ListQuests failed: %v", err)
	}
	for _, q := range quests {
		if q.GroupID != "" && strings.Contains(q.GroupID, autoCfg.ID) {
			t.Fatalf("fanout should not partially create quests on invalid batch: %+v", q)
		}
	}
}

func TestSpawnExecuteFromDesign(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	design, err := eng.CreateQuest(ctx, "设计权限系统", model.QuestTypeDesign, "", CreateQuestOptions{
		WarriorID:     "adv_warrior_001",
		MageID:        "adv_mage_001",
		WorkspaceMode: model.WorkspaceReadOnly,
	})
	if err != nil {
		t.Fatalf("CreateQuest(design) failed: %v", err)
	}
	qs := fsstore.NewQuestStore(eng.root)
	design.Status = model.QuestStatusSuccess
	design.FinalVerdict = model.VerdictPass
	design.FinalComment = "采用 RBAC + policy guard"
	design.DesignSummary = "设计方案：RBAC 负责角色，policy guard 负责资源级校验。"
	if err := qs.SaveQuest(design); err != nil {
		t.Fatalf("SaveQuest(design) failed: %v", err)
	}
	if err := qs.SaveDesignDoc(design.ID, design.BuildDesignDoc(design.DesignSummary, design.FinalComment)); err != nil {
		t.Fatalf("SaveDesignDoc failed: %v", err)
	}

	child, err := eng.SpawnExecuteFromDesign(ctx, design.ID, false)
	if err != nil {
		t.Fatalf("SpawnExecuteFromDesign failed: %v", err)
	}
	if child.Type != model.QuestTypeExecute {
		t.Fatalf("child type = %s, want execute", child.Type)
	}
	if child.ParentQuestID != design.ID {
		t.Fatalf("parent id = %s, want %s", child.ParentQuestID, design.ID)
	}
	if child.Status != model.QuestStatusPending {
		t.Fatalf("child status = %s, want pending", child.Status)
	}
	if child.WarriorID != design.WarriorID || child.MageID != design.MageID {
		t.Fatalf("child adventurers = (%s,%s), want (%s,%s)", child.WarriorID, child.MageID, design.WarriorID, design.MageID)
	}
	if child.WorkspaceMode != design.WorkspaceMode {
		t.Fatalf("child workspace mode = %s, want %s", child.WorkspaceMode, design.WorkspaceMode)
	}
	if !strings.Contains(child.Query, design.DesignSummary) || !strings.Contains(child.Query, design.Query) {
		t.Fatalf("child query should include original query and design summary:\n%s", child.Query)
	}
	doc, err := qs.LoadDesignDoc(design.ID)
	if err != nil {
		t.Fatalf("LoadDesignDoc failed: %v", err)
	}
	if doc.ExecutePrompt != child.Query {
		t.Fatalf("child query should come from design doc execute prompt")
	}
	if doc.SchemaVersion != "gloop.design.v1" || len(doc.ImplementationSteps) == 0 || len(doc.Risks) == 0 {
		t.Fatalf("design doc should include schema and structured sections: %+v", doc)
	}
	loadedParent, err := eng.GetQuest(design.ID)
	if err != nil {
		t.Fatalf("GetQuest(parent) failed: %v", err)
	}
	if loadedParent.ChildExecuteQuestID != child.ID || loadedParent.SpawnError != "" {
		t.Fatalf("parent should persist child link and clear spawn error: %+v", loadedParent)
	}
	again, err := eng.SpawnExecuteFromDesign(ctx, design.ID, false)
	if err != nil {
		t.Fatalf("SpawnExecuteFromDesign second call should be idempotent: %v", err)
	}
	if again.ID != child.ID {
		t.Fatalf("second spawn id = %s, want existing child %s", again.ID, child.ID)
	}
}

func TestAutoSpawnExecuteFromDesign(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	design, err := eng.CreateQuest(ctx, "自动执行方案", model.QuestTypeDesign, "", CreateQuestOptions{
		WarriorID:        "adv_warrior_001",
		MageID:           "adv_mage_001",
		WorkspaceMode:    model.WorkspaceReadOnly,
		AutoSpawnExecute: true,
	})
	if err != nil {
		t.Fatalf("CreateQuest(design) failed: %v", err)
	}
	qs := fsstore.NewQuestStore(eng.root)
	design.Status = model.QuestStatusSuccess
	design.FinalVerdict = model.VerdictPass
	design.FinalComment = "采用策略中心"
	design.DesignSummary = "设计方案：用策略中心统一处理权限。"
	if err := qs.SaveQuest(design); err != nil {
		t.Fatalf("SaveQuest(design) failed: %v", err)
	}
	if err := qs.SaveDesignDoc(design.ID, design.BuildDesignDoc(design.DesignSummary, design.FinalComment)); err != nil {
		t.Fatalf("SaveDesignDoc failed: %v", err)
	}

	eng.maybeAutoSpawnExecuteFromDesign(ctx, design.ID)
	parent, err := eng.GetQuest(design.ID)
	if err != nil {
		t.Fatalf("GetQuest(parent) failed: %v", err)
	}
	if parent.ChildExecuteQuestID == "" || parent.SpawnError != "" {
		t.Fatalf("auto spawn should persist child and clear error: %+v", parent)
	}
	child, err := eng.GetQuest(parent.ChildExecuteQuestID)
	if err != nil {
		t.Fatalf("GetQuest(child) failed: %v", err)
	}
	if child.ParentQuestID != design.ID || child.Type != model.QuestTypeExecute {
		t.Fatalf("bad child quest: %+v", child)
	}
}

func TestAutoSpawnExecuteRejectsQuickDesign(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	design, err := eng.CreateQuest(ctx, "quick 自动执行方案", model.QuestTypeDesign, "", CreateQuestOptions{
		WarriorID:        "adv_warrior_001",
		MageID:           "",
		WorkspaceMode:    model.WorkspaceReadOnly,
		Intensity:        model.QuestIntensityQuick,
		AutoSpawnExecute: true,
	})
	if err != nil {
		t.Fatalf("CreateQuest(design) failed: %v", err)
	}
	qs := fsstore.NewQuestStore(eng.root)
	design.Status = model.QuestStatusSuccess
	design.FinalVerdict = model.VerdictPass
	design.DesignSummary = "quick 方案"
	if err := qs.SaveQuest(design); err != nil {
		t.Fatalf("SaveQuest(design) failed: %v", err)
	}

	eng.maybeAutoSpawnExecuteFromDesign(ctx, design.ID)
	parent, err := eng.GetQuest(design.ID)
	if err != nil {
		t.Fatalf("GetQuest(parent) failed: %v", err)
	}
	if parent.ChildExecuteQuestID != "" || !strings.Contains(parent.SpawnError, "quick design") {
		t.Fatalf("quick design should not auto spawn: %+v", parent)
	}
}

func TestSpawnExecuteFromDesignRejectsInvalidParent(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	execQuest, err := eng.CreateQuest(ctx, "直接执行", model.QuestTypeExecute, "")
	if err != nil {
		t.Fatalf("CreateQuest(execute) failed: %v", err)
	}
	if _, err := eng.SpawnExecuteFromDesign(ctx, execQuest.ID, false); err == nil {
		t.Fatal("expected execute parent to be rejected")
	}

	design, err := eng.CreateQuest(ctx, "未完成方案", model.QuestTypeDesign, "")
	if err != nil {
		t.Fatalf("CreateQuest(design) failed: %v", err)
	}
	if _, err := eng.SpawnExecuteFromDesign(ctx, design.ID, false); err == nil {
		t.Fatal("expected non-success design parent to be rejected")
	}
}

func TestCreateQuestSpecifiedAdventurers(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	q, err := eng.CreateQuest(ctx, "指定冒险者", model.QuestTypeExecute, "", CreateQuestOptions{
		WarriorID: "adv_warrior_001",
		MageID:    "adv_mage_001",
	})
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	if q.WarriorID != "adv_warrior_001" || q.MageID != "adv_mage_001" {
		t.Fatalf("quest should store specified adventurers: %+v", q)
	}
	if err := eng.StartQuest(ctx, q.ID); err != nil {
		t.Fatalf("StartQuest should accept specified active adventurers: %v", err)
	}

	bad, err := eng.CreateQuest(ctx, "职业错误", model.QuestTypeExecute, "", CreateQuestOptions{
		WarriorID: "adv_mage_001",
		MageID:    "adv_mage_001",
	})
	if err != nil {
		t.Fatalf("CreateQuest(bad) failed: %v", err)
	}
	if err := eng.StartQuest(ctx, bad.ID); err == nil {
		t.Fatal("StartQuest should reject mismatched warrior class")
	}
}

func TestCreateQuestIntensityBudgets(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	standard, err := eng.CreateQuest(ctx, "标准强度", model.QuestTypeExecute, "")
	if err != nil {
		t.Fatalf("CreateQuest standard failed: %v", err)
	}
	if standard.Intensity != model.QuestIntensityStandard {
		t.Fatalf("standard intensity = %s", standard.Intensity)
	}
	if standard.MaxRework != eng.cfg.MaxReworkPerQuest || standard.MaxTurnsPerPhaseOverride != 0 || standard.MaxDurationPerQuestMsOverride != 0 {
		t.Fatalf("standard budget should follow global config: %+v", standard)
	}

	quick, err := eng.CreateQuest(ctx, "快速强度", model.QuestTypeExecute, "", CreateQuestOptions{Intensity: model.QuestIntensityQuick})
	if err != nil {
		t.Fatalf("CreateQuest quick failed: %v", err)
	}
	if quick.MaxRework != 1 || quick.MaxTurnsPerPhaseOverride != 10 || quick.MaxDurationPerQuestMsOverride != 30*60*1000 {
		t.Fatalf("quick budget mismatch: %+v", quick)
	}

	deep, err := eng.CreateQuest(ctx, "深入强度", model.QuestTypeExecute, "", CreateQuestOptions{Intensity: model.QuestIntensityDeep})
	if err != nil {
		t.Fatalf("CreateQuest deep failed: %v", err)
	}
	if deep.MaxRework != 4 || deep.MaxTurnsPerPhaseOverride != 80 || deep.MaxDurationPerQuestMsOverride != 8*60*60*1000 {
		t.Fatalf("deep budget mismatch: %+v", deep)
	}

	strict, err := eng.CreateQuest(ctx, "严审强度", model.QuestTypeExecute, "", CreateQuestOptions{Intensity: model.QuestIntensityAdversarial})
	if err != nil {
		t.Fatalf("CreateQuest adversarial failed: %v", err)
	}
	if strict.MaxRework != 5 || strict.MaxTurnsPerPhaseOverride != 100 || strict.MaxDurationPerQuestMsOverride != 12*60*60*1000 {
		t.Fatalf("adversarial budget mismatch: %+v", strict)
	}

	if _, err := eng.CreateQuest(ctx, "非法强度", model.QuestTypeExecute, "", CreateQuestOptions{Intensity: model.QuestIntensity("huge")}); err == nil {
		t.Fatal("invalid intensity should be rejected")
	}
}

func TestAppendUserComment(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()
	q, err := eng.CreateQuest(ctx, "评论测试", model.QuestTypeExecute, "")
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	if err := eng.AppendUserComment(q.ID, "用户补充上下文", ""); err != nil {
		t.Fatalf("AppendUserComment failed: %v", err)
	}
	evs, err := eng.GetQuestEvents(q.ID, 10)
	if err != nil {
		t.Fatalf("GetQuestEvents failed: %v", err)
	}
	found := false
	var noteEventID int64
	for _, ev := range evs {
		if ev.Type == string(events.EvtQuestNote) {
			found = true
			noteEventID = ev.ID
			break
		}
	}
	if !found {
		t.Fatal("comment should be stored as quest.note event")
	}

	// Verify durable ThreadPost projection (P0b-1)
	qs := fsstore.NewQuestStore(eng.root)
	posts, err := qs.LoadThreadPosts(q.ID)
	if err != nil {
		t.Fatalf("LoadThreadPosts failed: %v", err)
	}
	var commentPost *fsstore.ThreadPost
	for i := range posts {
		if posts[i].Kind == "human_comment" {
			commentPost = &posts[i]
			break
		}
	}
	if commentPost == nil {
		t.Fatal("human_comment thread post should exist after AppendUserComment")
	}
	if commentPost.AuthorRole != model.PostRoleHuman {
		t.Errorf("comment author_role = %s, want %s", commentPost.AuthorRole, model.PostRoleHuman)
	}
	expectedPostID := fmt.Sprintf("comment_%d", noteEventID)
	if commentPost.PostID != expectedPostID {
		t.Errorf("comment post_id = %s, want %s", commentPost.PostID, expectedPostID)
	}
	if commentPost.SourceEventID != noteEventID {
		t.Errorf("comment SourceEventID = %d, want %d", commentPost.SourceEventID, noteEventID)
	}
	if commentPost.SourceEventID == 0 {
		t.Error("comment SourceEventID should not be 0")
	}
	rootID := fsstore.RootPostID(q.ID)
	hasRootRef := false
	for _, ref := range commentPost.CausalRefs {
		if ref == rootID {
			hasRootRef = true
			break
		}
	}
	if !hasRootRef {
		t.Errorf("comment causal_refs should include root post %s, got %v", rootID, commentPost.CausalRefs)
	}
	if commentPost.ParentReplyID != "" {
		t.Errorf("comment parent_reply_id should be empty when replying to root, got %q", commentPost.ParentReplyID)
	}
	if commentPost.Content != "用户补充上下文" {
		t.Errorf("comment content = %q, want %q", commentPost.Content, "用户补充上下文")
	}
}

func TestAppendUserComment_ReplyToPost(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()
	q, err := eng.CreateQuest(ctx, "回复评论测试", model.QuestTypeExecute, "")
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}

	// First, write a top-level comment to reply to.
	if err := eng.AppendUserComment(q.ID, "一楼评论", ""); err != nil {
		t.Fatalf("first AppendUserComment failed: %v", err)
	}

	qs := fsstore.NewQuestStore(eng.root)
	posts, err := qs.LoadThreadPosts(q.ID)
	if err != nil {
		t.Fatalf("LoadThreadPosts failed: %v", err)
	}
	var firstComment *fsstore.ThreadPost
	for i := range posts {
		if posts[i].Kind == "human_comment" {
			firstComment = &posts[i]
			break
		}
	}
	if firstComment == nil {
		t.Fatal("first comment should exist")
	}

	// Reply to the first comment.
	if err := eng.AppendUserComment(q.ID, "回复一楼", firstComment.PostID); err != nil {
		t.Fatalf("reply AppendUserComment failed: %v", err)
	}

	posts, err = qs.LoadThreadPosts(q.ID)
	if err != nil {
		t.Fatalf("LoadThreadPosts after reply failed: %v", err)
	}
	var replyPost *fsstore.ThreadPost
	for i := range posts {
		if posts[i].Content == "回复一楼" {
			replyPost = &posts[i]
			break
		}
	}
	if replyPost == nil {
		t.Fatal("reply post should exist")
	}
	if replyPost.ParentReplyID != firstComment.PostID {
		t.Errorf("reply parent_reply_id = %q, want %q", replyPost.ParentReplyID, firstComment.PostID)
	}
	hasCausalRef := false
	for _, ref := range replyPost.CausalRefs {
		if ref == firstComment.PostID {
			hasCausalRef = true
			break
		}
	}
	if !hasCausalRef {
		t.Errorf("reply causal_refs should include replied-to post %s, got %v", firstComment.PostID, replyPost.CausalRefs)
	}
	if replyPost.SourceEventID == 0 {
		t.Error("reply SourceEventID should not be 0")
	}
}

func TestAppendUserComment_InvalidParentReplyID(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()
	q, err := eng.CreateQuest(ctx, "无效 parent 测试", model.QuestTypeExecute, "")
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}

	// Attempting to reply to a non-existent post should fail.
	err = eng.AppendUserComment(q.ID, "test reply", "post_does_not_exist_12345")
	if err == nil {
		t.Fatal("AppendUserComment with invalid parent_reply_id should fail")
	}
	if !strings.Contains(err.Error(), "parent_reply_id") {
		t.Errorf("error should mention parent_reply_id, got: %v", err)
	}

	// Verify no post was written.
	qs := fsstore.NewQuestStore(eng.root)
	posts, _ := qs.LoadThreadPosts(q.ID)
	for _, p := range posts {
		if p.Kind == "human_comment" {
			t.Fatalf("no human_comment should be written on invalid parent, found %+v", p)
		}
	}
}

func TestAppendUserAnswerResumesWaitingInput(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()
	q, err := eng.CreateQuest(ctx, "回答测试", model.QuestTypeExecute, "")
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	q.Status = model.QuestStatusWaitingInput
	if err := fsstore.NewQuestStore(eng.root).SaveQuest(q); err != nil {
		t.Fatalf("SaveQuest failed: %v", err)
	}
	rec, err := eng.AppendUserAnswer(q.ID, "question_1", "答案", "test")
	if err != nil {
		t.Fatalf("AppendUserAnswer failed: %v", err)
	}
	if rec.AnswerID == "" || rec.QuestionID != "question_1" || rec.Content != "答案" {
		t.Fatalf("bad answer record: %+v", rec)
	}
	got, err := eng.GetQuest(q.ID)
	if err != nil {
		t.Fatalf("GetQuest failed: %v", err)
	}
	if got.Status != model.QuestStatusRunning {
		t.Fatalf("waiting_input answer should resume running, got %+v", got)
	}
	if got.WaitingInput == nil || got.WaitingInput.LastAnswerID != rec.AnswerID {
		t.Fatalf("waiting_input should retain answer cursor until injected: %+v", got.WaitingInput)
	}
	answers, err := fsstore.NewQuestStore(eng.root).LoadAnswers(q.ID)
	if err != nil {
		t.Fatalf("LoadAnswers failed: %v", err)
	}
	if len(answers) != 1 || answers[0].AnswerID != rec.AnswerID {
		t.Fatalf("answer should be persisted: %+v", answers)
	}

	// Verify human_answer ThreadPost projection (P0b-2).
	qs := fsstore.NewQuestStore(eng.root)
	posts, err := qs.LoadThreadPosts(q.ID)
	if err != nil {
		t.Fatalf("LoadThreadPosts failed: %v", err)
	}
	var answerPost *fsstore.ThreadPost
	for i := range posts {
		if posts[i].Kind == "human_answer" {
			answerPost = &posts[i]
			break
		}
	}
	if answerPost == nil {
		t.Fatal("human_answer thread post should exist after AppendUserAnswer")
	}
	if answerPost.AuthorRole != model.PostRoleHuman {
		t.Errorf("answer author_role = %s, want %s", answerPost.AuthorRole, model.PostRoleHuman)
	}
	if answerPost.SourceEventID == 0 {
		t.Error("answer SourceEventID should not be 0")
	}
	if !strings.HasPrefix(answerPost.PostID, "human_answer_") {
		t.Errorf("answer post_id = %s, want prefix human_answer_", answerPost.PostID)
	}
	if answerPost.Content != "答案" {
		t.Errorf("answer content = %q, want %q", answerPost.Content, "答案")
	}
	hasRootRef := false
	for _, ref := range answerPost.CausalRefs {
		if ref == fsstore.RootPostID(q.ID) {
			hasRootRef = true
			break
		}
	}
	if !hasRootRef {
		t.Errorf("answer causal_refs should include root, got %v", answerPost.CausalRefs)
	}

	injected := eng.collectWaitingInputAnswers(q.ID)
	if len(injected) != 1 || injected[0].AnswerID != rec.AnswerID {
		t.Fatalf("waiting_input answer should be injectable: %+v", injected)
	}
	eng.clearWaitingInputAfterAnswerInject(q.ID)
	got, err = eng.GetQuest(q.ID)
	if err != nil {
		t.Fatalf("GetQuest failed: %v", err)
	}
	if got.WaitingInput != nil {
		t.Fatalf("waiting_input should be cleared after injection: %+v", got.WaitingInput)
	}
}

func TestEngine_RecoverWaitingInputTimeouts(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()
	q, err := eng.CreateQuest(ctx, "等待输入超时", model.QuestTypeExecute, "")
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	q.Status = model.QuestStatusWaitingInput
	q.WaitingInput = &fsstore.WaitingInputState{
		QuestionID:     "question_timeout",
		QuestionText:   "需要补充信息",
		PhaseIdx:       0,
		TimeoutMs:      1,
		TimeoutAction:  "continue",
		ResumePhaseIdx: 0,
		AskedAt:        time.Now().Add(-time.Minute),
	}
	if err := fsstore.NewQuestStore(eng.root).SaveQuest(q); err != nil {
		t.Fatalf("SaveQuest failed: %v", err)
	}
	continued, blocked, escalated, total := eng.RecoverWaitingInputTimeouts(ctx)
	if total != 1 || continued != 1 || blocked != 0 || escalated != 0 {
		t.Fatalf("RecoverWaitingInputTimeouts = continued:%d blocked:%d escalated:%d total:%d", continued, blocked, escalated, total)
	}
	got, err := eng.GetQuest(q.ID)
	if err != nil {
		t.Fatalf("GetQuest failed: %v", err)
	}
	if got.Status != model.QuestStatusRunning || got.WaitingInput == nil {
		t.Fatalf("timeout continue should resume running and retain injection cursor: %+v", got)
	}
	answers, err := fsstore.NewQuestStore(eng.root).LoadAnswers(q.ID)
	if err != nil {
		t.Fatalf("LoadAnswers failed: %v", err)
	}
	if len(answers) != 1 || answers[0].Source != "timeout" {
		t.Fatalf("timeout should append synthetic answer: %+v", answers)
	}
}

// messageCapturingExecutor 捕获每轮收到的用户消息，用于验证评论注入。
type messageCapturingExecutor struct {
	*executor.MockExecutor
	mu           sync.Mutex
	receivedMsgs []executor.Message
	callCount    int
	// 第几轮结束（返回 phase_checkpoint）
	endOnTurn int
}

func newMessageCapturingExecutor(endOnTurn int) *messageCapturingExecutor {
	return &messageCapturingExecutor{
		MockExecutor: executor.NewMockExecutor("exe_capture"),
		endOnTurn:    endOnTurn,
	}
}

func (e *messageCapturingExecutor) SendMessage(
	ctx context.Context, sessionID string, msg executor.Message, model string, stream chan<- executor.StreamChunk,
) (*executor.ChatResponse, error) {
	e.mu.Lock()
	e.receivedMsgs = append(e.receivedMsgs, msg)
	turn := e.callCount
	e.callCount++
	e.mu.Unlock()

	// 工具结果回灌或普通用户消息，都根据轮次决定是否结束
	if e.endOnTurn > 0 && turn >= e.endOnTurn {
		// 仅剑士 session 返回阶段结束信号；法师 session 不返回（让 mage 阶段
		// 跑到 max turns 后走 BuildFallback）。这样法师阶段的多轮运行能让评论
		// 注入有机会被发送给 executor——评论注入测试需要 mage 跑满多轮才有意义。
		// （历史：v0.2 时这模拟的是 phase_checkpoint 对法师不可用；v0.3.4 起 native
		// tool 分发层已删，但"mage 不主动结束、走 fallback"的测试时序仍需要。）
		shortSID := sessionID
		if idx := strings.LastIndex(shortSID, "/"); idx >= 0 {
			shortSID = shortSID[idx+1:]
		}
		if !strings.HasPrefix(shortSID, "mage_") {
			return phaseDoneResp(sessionID, "任务完成，提交阶段总结", "done"), nil
		}
	}

	return &executor.ChatResponse{
		SessionID:    sessionID,
		Message:      executor.Message{Role: "assistant", Content: "我正在分析任务..."},
		FinishReason: "stop",
	}, nil
}

func (e *messageCapturingExecutor) getReceivedMsgs() []executor.Message {
	e.mu.Lock()
	defer e.mu.Unlock()
	result := make([]executor.Message, len(e.receivedMsgs))
	copy(result, e.receivedMsgs)
	return result
}

func TestQuestAsk_WritesWaitingInputThreadPost(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()
	q, err := eng.CreateQuest(ctx, "等待输入提问测试", model.QuestTypeExecute, "")
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}

	// Simulate agent asking a question via QuestAsk API.
	// QuestAsk requires WorkspacePath for phase signal writing.
	q.WorkspacePath = eng.workDir
	if err := fsstore.NewQuestStore(eng.root).SaveQuest(q); err != nil {
		t.Fatalf("SaveQuest failed: %v", err)
	}
	question := "目标框架版本是多少？"
	if err := eng.QuestAsk(q.ID, "session_1", question); err != nil {
		t.Fatalf("QuestAsk failed: %v", err)
	}

	// Verify system_waiting_input ThreadPost projection.
	qs := fsstore.NewQuestStore(eng.root)
	posts, err := qs.LoadThreadPosts(q.ID)
	if err != nil {
		t.Fatalf("LoadThreadPosts failed: %v", err)
	}
	var waitingPost *fsstore.ThreadPost
	for i := range posts {
		if posts[i].Kind == "system_waiting_input" {
			waitingPost = &posts[i]
			break
		}
	}
	if waitingPost == nil {
		t.Fatal("system_waiting_input thread post should exist after QuestAsk")
	}
	if waitingPost.AuthorRole != model.PostRoleSystem {
		t.Errorf("waiting_input author_role = %s, want %s", waitingPost.AuthorRole, model.PostRoleSystem)
	}
	if waitingPost.SourceEventID == 0 {
		t.Error("waiting_input SourceEventID should not be 0")
	}
	if !strings.HasPrefix(waitingPost.PostID, "sys_waiting_input_") {
		t.Errorf("waiting_input post_id = %s, want prefix sys_waiting_input_", waitingPost.PostID)
	}
	if waitingPost.Content != question {
		t.Errorf("waiting_input content = %q, want %q", waitingPost.Content, question)
	}
	hasRootRef := false
	for _, ref := range waitingPost.CausalRefs {
		if ref == fsstore.RootPostID(q.ID) {
			hasRootRef = true
			break
		}
	}
	if !hasRootRef {
		t.Errorf("waiting_input causal_refs should include root, got %v", waitingPost.CausalRefs)
	}
}

func TestMacroLoopWaitingSignal_WritesWaitingInputThreadPost(t *testing.T) {
	eng, _ := setupTestEngine(t)
	// Register executor that returns waiting_input for warrior phase.
	waitingEx := &waitingInputWarriorExecutor{MockExecutor: executor.NewMockExecutor("waiting_mock")}
	eng.RegisterExecutor("test_agent", waitingEx)

	// Disable HOTL auto-close so quest doesn't auto-complete.
	hotlAutoClose := false
	eng.cfg.HotlAutoClose = &hotlAutoClose

	ctx := context.Background()
	q, err := eng.CreateQuest(ctx, "macro loop waiting input test", model.QuestTypeExecute, "")
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}

	if err := eng.StartQuest(ctx, q.ID); err != nil {
		t.Fatalf("StartQuest failed: %v", err)
	}

	// Wait for quest to reach waiting_input status via macro loop.
	waitForStatus(t, eng, q.ID, model.QuestStatusWaitingInput, 15*time.Second)

	// Verify system_waiting_input ThreadPost was written by macro loop path.
	qs := fsstore.NewQuestStore(eng.root)
	posts, err := qs.LoadThreadPosts(q.ID)
	if err != nil {
		t.Fatalf("LoadThreadPosts failed: %v", err)
	}
	var waitingPost *fsstore.ThreadPost
	for i := range posts {
		if posts[i].Kind == "system_waiting_input" {
			waitingPost = &posts[i]
			break
		}
	}
	if waitingPost == nil {
		t.Fatal("system_waiting_input thread post should exist after macro loop waiting signal")
	}
	if waitingPost.AuthorRole != model.PostRoleSystem {
		t.Errorf("waiting_input author_role = %s, want %s", waitingPost.AuthorRole, model.PostRoleSystem)
	}
	if waitingPost.SourceEventID == 0 {
		t.Error("waiting_input SourceEventID should not be 0")
	}
	if !strings.HasPrefix(waitingPost.PostID, "sys_waiting_input_") {
		t.Errorf("waiting_input post_id = %s, want prefix sys_waiting_input_", waitingPost.PostID)
	}
	if waitingPost.Content == "" {
		t.Error("waiting_input content should not be empty")
	}
	hasRootRef := false
	for _, ref := range waitingPost.CausalRefs {
		if ref == fsstore.RootPostID(q.ID) {
			hasRootRef = true
			break
		}
	}
	if !hasRootRef {
		t.Errorf("waiting_input causal_refs should include root, got %v", waitingPost.CausalRefs)
	}
}

func TestEngine_UserCommentDeduplicated(t *testing.T) {
	// 验证：同一条评论不会重复注入
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	mockEx := newMessageCapturingExecutor(5)
	eng.RegisterExecutor("test_agent", mockEx)

	q, err := eng.CreateQuest(ctx, "评论去重测试", model.QuestTypeExecute, "")
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}

	// 先发一条评论，再启动
	if err := eng.AppendUserComment(q.ID, "初始评论", ""); err != nil {
		t.Fatalf("AppendUserComment failed: %v", err)
	}

	if err := eng.StartQuest(ctx, q.ID); err != nil {
		t.Fatalf("StartQuest failed: %v", err)
	}

	// 等 quest 跑完
	waitForStatus(t, eng, q.ID, model.QuestStatusBlocked, 10*time.Second)

	// 检查 user_comments 消息出现的次数
	msgs := mockEx.getReceivedMsgs()
	commentCount := 0
	for _, m := range msgs {
		if m.Role == "user" && strings.Contains(m.Content, "user_comments") {
			commentCount++
		}
	}

	// 初始评论应该只在第一轮注入一次，后续轮次不重复注入
	if commentCount != 1 {
		t.Errorf("期望用户评论注入 1 次，实际注入了 %d 次", commentCount)
		t.Logf("收到的 user 消息:")
		for i, m := range msgs {
			if m.Role == "user" {
				t.Logf("  [%d] %s", i, truncate(m.Content, 100))
			}
		}
	}
}

// TestEngine_UserCommentAutoResumeBlocked 验证 blocked 状态下用户评论自动解除阻塞并继续运行
func TestEngine_AutoRetryHonorsMaxAttempts(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	q, err := eng.CreateQuest(ctx, "自动恢复次数上限测试", model.QuestTypeExecute, "", CreateQuestOptions{
		Intensity: model.QuestIntensityQuick,
	})
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	qs := fsstore.NewQuestStore(eng.root)
	q, err = qs.LoadQuest(q.ID)
	if err != nil {
		t.Fatalf("LoadQuest failed: %v", err)
	}
	q.WarriorID = "adv_warrior_001"
	q.WorkspacePath = t.TempDir()
	q.Status = model.QuestStatusBlocked
	q.BlockedReason = "no progress"
	q.BlockedReasonCode = "no_progress"
	if err := qs.SaveQuest(q); err != nil {
		t.Fatalf("SaveQuest failed: %v", err)
	}
	if err := eng.root.SaveRecoveryState(&fsstore.RecoveryState{
		QuestID:            q.ID,
		PolicyName:         "no_progress_retry",
		AttemptIndex:       1,
		AddTurns:           7,
		AddDurationMinutes: 11,
		MaxAttempts:        1,
		Status:             fsstore.RecoveryStatusPending,
		LastAction:         policy.ActionRetry,
		BlockReason:        "no_progress",
	}); err != nil {
		t.Fatalf("SaveRecoveryState failed: %v", err)
	}

	if eng.shouldAutoRetryBlockedLocked(q.ID) {
		t.Fatal("auto retry should stop when attempt index reaches max_attempts")
	}
	got, err := eng.GetQuest(q.ID)
	if err != nil {
		t.Fatalf("GetQuest failed: %v", err)
	}
	if got.Status != model.QuestStatusBlocked {
		t.Fatalf("quest should remain blocked when retry limit reached: %+v", got)
	}
}

// TestEngine_UserCommentNotDuplicatedAcrossRework 验证返工后历史评论不重复注入

func TestSourceEventID_RootPost(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	q, err := eng.CreateQuest(ctx, "验证 root post source_event_id", model.QuestTypeExecute, "")
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}

	posts, err := fsstore.NewQuestStore(eng.root).LoadThreadPosts(q.ID)
	if err != nil {
		t.Fatalf("LoadThreadPosts failed: %v", err)
	}
	if len(posts) < 1 {
		t.Fatal("no thread posts found")
	}

	rootID := fsstore.RootPostID(q.ID)
	var root *fsstore.ThreadPost
	for i := range posts {
		if posts[i].PostID == rootID {
			root = &posts[i]
			break
		}
	}
	if root == nil {
		t.Fatalf("root post %s not found in %+v", rootID, posts)
	}
	if root.SourceEventID == 0 {
		t.Fatalf("root post SourceEventID should be non-zero, got post: %+v", root)
	}
	if root.Kind != "root" {
		t.Fatalf("root post kind = %q, want %q", root.Kind, "root")
	}

	// Verify the event exists in events.jsonl with matching ID
	events, err := fsstore.NewQuestStore(eng.root).ReadEvents(q.ID, 0)
	if err != nil {
		t.Fatalf("ReadEvents failed: %v", err)
	}
	foundEvent := false
	for _, ev := range events {
		if ev.ID == root.SourceEventID && ev.Type == "quest.created" {
			foundEvent = true
			break
		}
	}
	if !foundEvent {
		t.Fatalf("quest.created event with ID %d not found in events (have %d rows)", root.SourceEventID, len(events))
	}
}

func TestSourceEventID_WorkflowUpgrade(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	q, err := eng.CreateQuest(ctx, "验证 workflow upgrade source_event_id", model.QuestTypeExecute, "", CreateQuestOptions{
		WarriorID:    "adv_warrior_001",
		MageID:       "adv_mage_001",
		WorkflowMode: model.WorkflowModeDirect,
		Intensity:    model.QuestIntensityQuick,
	})
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	if q.WorkflowMode != model.WorkflowModeDirect {
		t.Fatalf("quest workflow_mode = %s, want direct", q.WorkflowMode)
	}

	_, err = eng.UpgradeWorkflowMode(ctx, q.ID, model.WorkflowModeChecked, "test upgrade")
	if err != nil {
		t.Fatalf("UpgradeWorkflowMode failed: %v", err)
	}

	posts, err := fsstore.NewQuestStore(eng.root).LoadThreadPosts(q.ID)
	if err != nil {
		t.Fatalf("LoadThreadPosts failed: %v", err)
	}

	var upgradePost *fsstore.ThreadPost
	for i := range posts {
		if posts[i].Kind == "system_workflow_upgrade" {
			upgradePost = &posts[i]
			break
		}
	}
	if upgradePost == nil {
		t.Fatalf("system_workflow_upgrade post not found in %+v", posts)
	}
	if upgradePost.SourceEventID == 0 {
		t.Fatalf("workflow upgrade post SourceEventID should be non-zero, got: %+v", upgradePost)
	}

	// Verify the event exists
	found := false
	events, _ := fsstore.NewQuestStore(eng.root).ReadEvents(q.ID, 0)
	for _, ev := range events {
		if ev.ID == upgradePost.SourceEventID && ev.Type == "quest.workflow_upgraded" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("quest.workflow_upgraded event with ID %d not found", upgradePost.SourceEventID)
	}
}

func TestSourceEventID_OrchestratorMakerReport(t *testing.T) {
	eng, _ := setupTestEngine(t)

	q, err := eng.CreateQuest(context.Background(), "验证 maker report source_event_id", model.QuestTypeExecute, "", CreateQuestOptions{
		WarriorID: "adv_warrior_001",
		MageID:    "adv_mage_001",
		Intensity: model.QuestIntensityQuick,
	})
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}

	// Simulate warrior phase end to trigger persistMakerReport
	eng.persistMakerReport(q.ID, "warrior_0", 0, PhaseSignal{
		OK:           true,
		PhaseEnded:   true,
		PhaseVerdict: "done",
		PhaseComment: "warrior delivered summary",
	})

	posts, err := fsstore.NewQuestStore(eng.root).LoadThreadPosts(q.ID)
	if err != nil {
		t.Fatalf("LoadThreadPosts failed: %v", err)
	}

	var makerPost *fsstore.ThreadPost
	for i := range posts {
		if posts[i].Kind == "maker_report" {
			makerPost = &posts[i]
			break
		}
	}
	if makerPost == nil {
		t.Fatalf("maker_report post not found in %+v", posts)
	}
	if makerPost.SourceEventID == 0 {
		t.Fatalf("maker report post SourceEventID should be non-zero, got: %+v", makerPost)
	}

	found := false
	events, _ := fsstore.NewQuestStore(eng.root).ReadEvents(q.ID, 0)
	for _, ev := range events {
		if ev.ID == makerPost.SourceEventID && ev.Type == "report.maker_submitted" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("report.maker_submitted event with ID %d not found", makerPost.SourceEventID)
	}
}

func TestSourceEventID_OrchestratorReviewReport(t *testing.T) {
	eng, _ := setupTestEngine(t)

	q, err := eng.CreateQuest(context.Background(), "验证 review report source_event_id", model.QuestTypeExecute, "", CreateQuestOptions{
		WarriorID: "adv_warrior_001",
		MageID:    "adv_mage_001",
	})
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}

	// Simulate mage review to trigger persistReviewReport
	rv := &fsstore.ReviewRecord{
		Verdict:    model.VerdictPass,
		Comment:    "looks good",
		ReviewedBy: "adv_mage_001",
		Score:      8,
	}
	eng.persistReviewReport(q.ID, "mage_0", 1, rv)

	posts, err := fsstore.NewQuestStore(eng.root).LoadThreadPosts(q.ID)
	if err != nil {
		t.Fatalf("LoadThreadPosts failed: %v", err)
	}

	var reviewPost *fsstore.ThreadPost
	for i := range posts {
		if posts[i].Kind == "review_report" {
			reviewPost = &posts[i]
			break
		}
	}
	if reviewPost == nil {
		t.Fatalf("review_report post not found in %+v", posts)
	}
	if reviewPost.SourceEventID == 0 {
		t.Fatalf("review report post SourceEventID should be non-zero, got: %+v", reviewPost)
	}

	found := false
	events, _ := fsstore.NewQuestStore(eng.root).ReadEvents(q.ID, 0)
	for _, ev := range events {
		if ev.ID == reviewPost.SourceEventID && ev.Type == "report.review_submitted" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("report.review_submitted event with ID %d not found", reviewPost.SourceEventID)
	}
}

func TestSourceEventID_FsstoreDirectReportAllowsZero(t *testing.T) {
	// fsstore 直接 AppendMakerReport/AppendReviewReport 允许 SourceEventID=0
	eng, _ := setupTestEngine(t)

	q, err := eng.CreateQuest(context.Background(), "验证 fsstore direct report 允许 0", model.QuestTypeExecute, "")
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}

	// Direct fsstore call (no event recording)
	err = eng.root.AppendMakerReport(q.ID, &fsstore.MakerReport{
		QuestID: q.ID,
		Phase:   0,
		Verdict: "done",
		Summary: "direct fsstore maker report",
	})
	if err != nil {
		t.Fatalf("AppendMakerReport failed: %v", err)
	}

	err = eng.root.AppendReviewReport(q.ID, &fsstore.ReviewReport{
		QuestID:    q.ID,
		Phase:      1,
		Verdict:    model.VerdictPass,
		Confidence: "low",
		Comment:    "direct fsstore review report",
	})
	if err != nil {
		t.Fatalf("AppendReviewReport failed: %v", err)
	}

	posts, err := fsstore.NewQuestStore(eng.root).LoadThreadPosts(q.ID)
	if err != nil {
		t.Fatalf("LoadThreadPosts failed: %v", err)
	}

	for _, post := range posts {
		if post.Kind == "maker_report" || post.Kind == "review_report" {
			if post.SourceEventID != 0 {
				t.Fatalf("fsstore direct %s post SourceEventID = %d, want 0", post.Kind, post.SourceEventID)
			}
		}
	}
}

// TestEngine_DirectExternalWriteEscalatesToChecked 验证 Slice 3：
// 真实 Direct 入口（SkipReview=true → mode=run 单阶段 pipeline）+ 外部写 →
// 升档 Checked + 补 mage + 恢复两阶段 pipeline → Checker loop → success。
func TestEngine_DirectExternalWriteEscalatesToChecked(t *testing.T) {
	eng, _ := setupTestEngine(t)
	hotlAutoClose := true
	eng.cfg.HotlAutoClose = &hotlAutoClose

	// SkipReview=true + quick 模拟真实 API mode=run：PipelineName=run, PhaseCount=1, 单阶段
	q, err := eng.CreateQuest(context.Background(), "发布飞书文档并返回链接", model.QuestTypeExecute, "", CreateQuestOptions{
		WarriorID:    "adv_warrior_001",
		WorkflowMode: model.WorkflowModeDirect,
		SkipReview:   true,
		Intensity:    model.QuestIntensityQuick,
	})
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}

	// 预声明外部产出物，确保 effect routing 检测到 external_side_effect
	qs := fsstore.NewQuestStore(eng.root)
	if _, err := qs.AddDeclaredOutput(q.ID, fsstore.DeclaredOutput{
		Name:   "外部文档",
		Kind:   "external",
		URL:    "https://example.com/docs/external-write-result",
		Source: "warrior_phase",
	}); err != nil {
		t.Fatalf("AddDeclaredOutput failed: %v", err)
	}

	if err := eng.StartQuest(context.Background(), q.ID); err != nil {
		t.Fatalf("StartQuest failed: %v", err)
	}

	// 升档 + Checker pass → HOTL auto-close → success
	waitForStatus(t, eng, q.ID, model.QuestStatusSuccess, 15*time.Second)

	got, err := eng.GetQuest(q.ID)
	if err != nil {
		t.Fatalf("GetQuest failed: %v", err)
	}
	if got.WorkflowMode != model.WorkflowModeChecked {
		t.Fatalf("expected workflow_mode=checked, got %q", got.WorkflowMode)
	}

	// 验证经过了 Checker loop：mage review 生成了 review_report
	reports, err := eng.root.LoadReports(q.ID)
	if err != nil {
		t.Fatalf("LoadReports failed: %v", err)
	}
	if len(reports.ReviewReports) == 0 {
		t.Fatal("expected review_report from mage review (Checker loop), got none")
	}

	// 验证 system_workflow_upgrade reply 已投影到 thread
	posts, err := fsstore.NewQuestStore(eng.root).LoadThreadPosts(q.ID)
	if err != nil {
		t.Fatalf("LoadThreadPosts failed: %v", err)
	}
	foundUpgrade := false
	for _, post := range posts {
		if post.Kind == "system_workflow_upgrade" {
			foundUpgrade = true
		}
	}
	if !foundUpgrade {
		t.Fatalf("expected system_workflow_upgrade post in thread, got %d posts", len(posts))
	}
}

// TestEngine_DirectExternalWriteEscalationRestoresPipeline 验证升档后
// pipeline 字段与 Checked 语义一致：两阶段 pipeline + phase_count=2 + pipeline_name=default。
func TestEngine_DirectExternalWriteEscalationRestoresPipeline(t *testing.T) {
	eng, _ := setupTestEngine(t)
	hotlAutoClose := false // 停在 user_review 以便检查中间状态
	eng.cfg.HotlAutoClose = &hotlAutoClose

	q, err := eng.CreateQuest(context.Background(), "发布飞书文档", model.QuestTypeExecute, "", CreateQuestOptions{
		WarriorID:    "adv_warrior_001",
		WorkflowMode: model.WorkflowModeDirect,
		SkipReview:   true,
		Intensity:    model.QuestIntensityQuick,
	})
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}

	before, _ := eng.GetQuest(q.ID)
	if before.PipelineName != "run" || before.PhaseCount != 1 {
		t.Fatalf("precondition: expected run pipeline, got name=%s count=%d", before.PipelineName, before.PhaseCount)
	}

	qs := fsstore.NewQuestStore(eng.root)
	if _, err := qs.AddDeclaredOutput(q.ID, fsstore.DeclaredOutput{
		Name: "外部文档", Kind: "external",
		URL: "https://example.com/docs/result", Source: "warrior_phase",
	}); err != nil {
		t.Fatalf("AddDeclaredOutput failed: %v", err)
	}

	if err := eng.StartQuest(context.Background(), q.ID); err != nil {
		t.Fatalf("StartQuest failed: %v", err)
	}

	// HOTL off → Checker pass 后停在 user_review
	waitForStatus(t, eng, q.ID, model.QuestStatusUserReview, 15*time.Second)

	got, err := eng.GetQuest(q.ID)
	if err != nil {
		t.Fatalf("GetQuest failed: %v", err)
	}
	if got.PipelineName != fsstore.DefaultPipelineName {
		t.Errorf("expected pipeline_name=%s, got %s", fsstore.DefaultPipelineName, got.PipelineName)
	}
	if got.PhaseCount != 2 {
		t.Errorf("expected phase_count=2, got %d", got.PhaseCount)
	}
	if len(got.PipelineDef) != 2 {
		t.Errorf("expected pipeline_def len=2, got %d", len(got.PipelineDef))
	}
	if got.WorkflowMode != model.WorkflowModeChecked {
		t.Errorf("expected workflow_mode=checked, got %s", got.WorkflowMode)
	}
	if got.MageID == "" {
		t.Error("expected mage_id assigned after escalation")
	}
}

// TestEngine_DirectExternalWriteEscalationFailureBlocks 验证升档失败时
// 不回退 auto-complete，而是 block 委托（reason_code=checked_escalation_unavailable）。
// 通过删除所有 mage 冒险者来模拟 mage 分配失败。
func TestEngine_DirectExternalWriteEscalationFailureBlocks(t *testing.T) {
	eng, _ := setupTestEngine(t)
	hotlAutoClose := true
	eng.cfg.HotlAutoClose = &hotlAutoClose

	// 把测试 mage 设为 retired，让 pickAdventurer("", mage) 失败
	mage, _ := eng.root.GetAdventurer("adv_mage_001")
	if mage == nil {
		t.Fatal("precondition: adv_mage_001 not found")
	}
	mage.Status = model.AdventurerRetired
	if err := eng.root.SaveAdventurer(mage); err != nil {
		t.Fatalf("SaveAdventurer failed: %v", err)
	}

	q, err := eng.CreateQuest(context.Background(), "发布飞书文档", model.QuestTypeExecute, "", CreateQuestOptions{
		WarriorID:    "adv_warrior_001",
		WorkflowMode: model.WorkflowModeDirect,
		SkipReview:   true,
		Intensity:    model.QuestIntensityQuick, // 启动时不分配 mage；effect routing 时才补分配
	})
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}

	qs := fsstore.NewQuestStore(eng.root)
	if _, err := qs.AddDeclaredOutput(q.ID, fsstore.DeclaredOutput{
		Name: "外部文档", Kind: "external",
		URL: "https://example.com/docs/result", Source: "warrior_phase",
	}); err != nil {
		t.Fatalf("AddDeclaredOutput failed: %v", err)
	}

	if err := eng.StartQuest(context.Background(), q.ID); err != nil {
		t.Fatalf("StartQuest failed: %v", err)
	}

	// mage 分配失败 → block（不 success）
	waitForStatus(t, eng, q.ID, model.QuestStatusBlocked, 15*time.Second)

	got, err := eng.GetQuest(q.ID)
	if err != nil {
		t.Fatalf("GetQuest failed: %v", err)
	}
	if got.BlockedReasonCode != "checked_escalation_unavailable" {
		t.Fatalf("expected blocked_reason_code=checked_escalation_unavailable, got %q (reason=%s)", got.BlockedReasonCode, got.BlockedReason)
	}
}

// TestEngine_DirectLocalWriteAutoCompletes 验证 Slice 3 控制组：
// Direct 委托（SkipReview=true + quick，真实 API mode=run）没有外部写时，
// 走 auto-complete 到 success，不升档。
func TestEngine_DirectLocalWriteAutoCompletes(t *testing.T) {
	eng, _ := setupTestEngine(t)
	hotlAutoClose := true
	eng.cfg.HotlAutoClose = &hotlAutoClose

	q, err := eng.CreateQuest(context.Background(), "实现一个 hello world 函数", model.QuestTypeExecute, "", CreateQuestOptions{
		WarriorID:    "adv_warrior_001",
		WorkflowMode: model.WorkflowModeDirect,
		SkipReview:   true,
		Intensity:    model.QuestIntensityQuick,
	})
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	if err := eng.StartQuest(context.Background(), q.ID); err != nil {
		t.Fatalf("StartQuest failed: %v", err)
	}

	// 无外部写 → auto-close → success
	waitForStatus(t, eng, q.ID, model.QuestStatusSuccess, 15*time.Second)

	got, err := eng.GetQuest(q.ID)
	if err != nil {
		t.Fatalf("GetQuest failed: %v", err)
	}
	if got.WorkflowMode != model.WorkflowModeDirect {
		t.Fatalf("expected workflow_mode=direct (no escalation), got %q", got.WorkflowMode)
	}
}
