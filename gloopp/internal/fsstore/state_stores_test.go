package fsstore

import (
	"os"
	"path/filepath"
	"testing"

	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

func TestAutomationStateSaveAndGet(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	state := &AutomationState{
		AutomationID: "auto_ctx_refresh",
		QuestID:      "qst_001",
		LastEventID:  42,
		LastGlobalID: 100,
		LastRunTs:    1234567890,
		RunCount:     5,
	}
	if err := root.SaveAutomationState(state); err != nil {
		t.Fatalf("SaveAutomationState failed: %v", err)
	}

	got, err := root.GetAutomationState("auto_ctx_refresh", "qst_001")
	if err != nil {
		t.Fatalf("GetAutomationState failed: %v", err)
	}
	if got.AutomationID != state.AutomationID {
		t.Errorf("AutomationID = %q, want %q", got.AutomationID, state.AutomationID)
	}
	if got.LastEventID != 42 {
		t.Errorf("LastEventID = %d, want 42", got.LastEventID)
	}
	if got.LastGlobalID != 100 {
		t.Errorf("LastGlobalID = %d, want 100", got.LastGlobalID)
	}
	if got.RunCount != 5 {
		t.Errorf("RunCount = %d, want 5", got.RunCount)
	}

	// File should exist at the expected path
	path := filepath.Join(root.Sub(SubdirWorkspace, "automation_state"), "auto_ctx_refresh__qst_001.json")
	if _, err := os.Stat(path); err != nil {
		t.Errorf("automation state file should exist at %q: %v", path, err)
	}

	// Non-existent state returns error
	_, err = root.GetAutomationState("no_such", "qst_001")
	if err == nil {
		t.Error("GetAutomationState for missing state should error")
	}
	if !os.IsNotExist(err) {
		t.Logf("error type: %T (%v)", err, err) // some implementations wrap
	}

	// Save nil should be ok (no-op)
	if err := root.SaveAutomationState(nil); err != nil {
		t.Errorf("Save nil state should be no-op, got err=%v", err)
	}
	// Save with empty AutomationID should be ok (no-op)
	if err := root.SaveAutomationState(&AutomationState{}); err != nil {
		t.Errorf("Save empty state should be no-op, got err=%v", err)
	}
}

func TestRecoveryStateLifecycle(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	// Validation: empty qid
	if _, err := root.GetRecoveryState(""); err == nil {
		t.Error("empty qid GetRecoveryState should error")
	}
	if err := root.SaveRecoveryState(nil); err == nil {
		t.Error("nil SaveRecoveryState should error")
	}
	if err := root.SaveRecoveryState(&RecoveryState{}); err == nil {
		t.Error("empty-QuestID SaveRecoveryState should error")
	}

	state := &RecoveryState{
		QuestID:       "qst_blocked_01",
		PhaseIdx:      1,
		SessionID:     "sess_abc",
		PolicyName:    "session_lost_retry",
		AttemptIndex:  2,
		Status:        RecoveryStatusRunning,
		LastAction:    "launch_macro_loop",
		LastError:     "agent timeout",
		MaxAttempts:   5,
		BlockReason:   "agent hung",
		OriginalError: "agent connection lost",
	}
	if err := root.SaveRecoveryState(state); err != nil {
		t.Fatalf("SaveRecoveryState failed: %v", err)
	}
	// First save should set StartedAt and LastAttemptAt
	if state.StartedAt.IsZero() {
		t.Error("StartedAt should be set on first save")
	}
	if state.LastAttemptAt.IsZero() {
		t.Error("LastAttemptAt should be set on save")
	}
	if state.Status != RecoveryStatusRunning {
		t.Errorf("Status preserved: got %q", state.Status)
	}

	got, err := root.GetRecoveryState("qst_blocked_01")
	if err != nil {
		t.Fatalf("GetRecoveryState failed: %v", err)
	}
	if got.QuestID != "qst_blocked_01" {
		t.Errorf("QuestID = %q", got.QuestID)
	}
	if got.PolicyName != "session_lost_retry" {
		t.Errorf("PolicyName = %q", got.PolicyName)
	}
	if got.AttemptIndex != 2 {
		t.Errorf("AttemptIndex = %d", got.AttemptIndex)
	}

	// New (empty status) should default to pending
	pendingState := &RecoveryState{QuestID: "qst_pending"}
	if err := root.SaveRecoveryState(pendingState); err != nil {
		t.Fatalf("save pending failed: %v", err)
	}
	if pendingState.Status != RecoveryStatusPending {
		t.Errorf("default status = %q, want pending", pendingState.Status)
	}

	// Delete
	if err := root.DeleteRecoveryState("qst_blocked_01"); err != nil {
		t.Fatalf("DeleteRecoveryState failed: %v", err)
	}
	if _, err := root.GetRecoveryState("qst_blocked_01"); err == nil {
		t.Error("GetRecoveryState after delete should error")
	}
	// Delete again (non-existent) should be no-error
	if err := root.DeleteRecoveryState("qst_blocked_01"); err != nil {
		t.Errorf("delete non-existent should be no-op, got err=%v", err)
	}
	// Delete with empty qid should error
	if err := root.DeleteRecoveryState(""); err == nil {
		t.Error("delete empty qid should error")
	}
}

func TestAgentSessionStateLifecycle(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	exitCode := 0
	state := &AgentSessionState{
		SessionID:       "warrior_0",
		QuestID:         "qst_agent_session",
		Phase:           0,
		AgentID:         "adv_warrior",
		CapabilityTier:  "tier_b",
		HealthStatus:    "exited",
		FirstOutputAtMs: 123,
		LastOutputAtMs:  456,
		ExitCode:        &exitCode,
	}
	if err := root.SaveAgentSessionState(state); err != nil {
		t.Fatalf("SaveAgentSessionState failed: %v", err)
	}

	got, err := root.GetAgentSessionState("qst_agent_session", "warrior_0")
	if err != nil {
		t.Fatalf("GetAgentSessionState failed: %v", err)
	}
	if got.SessionID != state.SessionID || got.QuestID != state.QuestID || got.CapabilityTier != "tier_b" {
		t.Fatalf("bad agent session state: %+v", got)
	}
	if got.ExitCode == nil || *got.ExitCode != 0 {
		t.Fatalf("exit_code not persisted: %+v", got)
	}

	items, err := root.ListAgentSessionStates("qst_agent_session")
	if err != nil {
		t.Fatalf("ListAgentSessionStates failed: %v", err)
	}
	if len(items) != 1 || items[0].SessionID != "warrior_0" {
		t.Fatalf("bad list result: %+v", items)
	}
}

func TestQuestReportsLifecycle(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	if err := root.AppendMakerReport("qst_reports", &MakerReport{
		SessionID: "warrior_0",
		Phase:     0,
		Verdict:   "done",
		Summary:   "made it",
	}); err != nil {
		t.Fatalf("AppendMakerReport failed: %v", err)
	}
	if err := root.AppendReviewReport("qst_reports", &ReviewReport{
		SessionID: "mage_0",
		Phase:     1,
		Verdict:   "pass",
	}); err != nil {
		t.Fatalf("AppendReviewReport failed: %v", err)
	}
	if err := root.AppendEvidenceRef("qst_reports", &EvidenceRef{
		ID:        "ev_test",
		Kind:      "l0_command",
		TrustTier: "T0",
		Source:    "test",
		CommandID: "go-test",
	}); err != nil {
		t.Fatalf("AppendEvidenceRef failed: %v", err)
	}

	reports, err := root.LoadReports("qst_reports")
	if err != nil {
		t.Fatalf("LoadReports failed: %v", err)
	}
	if len(reports.MakerReports) != 1 || reports.MakerReports[0].SchemaVersion != "maker_report.v1" || reports.MakerReports[0].QuestID != "qst_reports" {
		t.Fatalf("bad maker reports: %+v", reports.MakerReports)
	}
	if len(reports.ReviewReports) != 1 || reports.ReviewReports[0].SchemaVersion != "review_report.v1" || reports.ReviewReports[0].Confidence != "low" {
		t.Fatalf("bad review reports: %+v", reports.ReviewReports)
	}
	refs, err := root.LoadEvidenceRefs("qst_reports")
	if err != nil {
		t.Fatalf("LoadEvidenceRefs failed: %v", err)
	}
	if len(refs) != 1 || refs[0].ID != "ev_test" || refs[0].TrustTier != "T0" {
		t.Fatalf("bad evidence refs: %+v", refs)
	}
}

func TestLoopStateSpineLifecycle(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	_, err = root.PatchLoopStateSpine("qst_loop", func(state *LoopStateSpine) {
		state.CurrentGoal = "ship v0.4"
		state.CurrentPhase = "maker_rework"
		state.NextExpectedAction = "fix review blockers"
		state.Attempts = append(state.Attempts, LoopStateAttempt{
			RunID:     "run_0",
			Summary:   "first pass",
			Result:    "request_changes",
			Reason:    "missing evidence",
			CreatedAt: NowMs(),
		})
	})
	if err != nil {
		t.Fatalf("PatchLoopStateSpine failed: %v", err)
	}
	got, err := root.LoadLoopStateSpine("qst_loop")
	if err != nil {
		t.Fatalf("LoadLoopStateSpine failed: %v", err)
	}
	if got.SchemaVersion != "loop_state_spine.v1" || got.NextExpectedAction != "fix review blockers" || len(got.Attempts) != 1 {
		t.Fatalf("bad loop state spine: %+v", got)
	}
}

func TestHumanExceptionFromQuest(t *testing.T) {
	q := &QuestMeta{
		ID:                "qst_exception",
		Status:            model.QuestStatusBlocked,
		BlockedReason:     "auth failed",
		BlockedReasonCode: "agent_error_auth",
		CreatedAtMs:       123,
	}
	item := HumanExceptionFromQuest(q)
	if item == nil || item.Reason != "auth failed" || item.RiskLevel != "high" || len(item.AvailableActions) == 0 {
		t.Fatalf("bad blocked human exception: %+v", item)
	}

	q.Status = model.QuestStatusWaitingInput
	q.WaitingInput = &WaitingInputState{QuestionText: "which branch?", Asker: "warrior"}
	item = HumanExceptionFromQuest(q)
	if item == nil || item.Reason != "which branch?" || item.RecommendedAction == "" {
		t.Fatalf("bad waiting-input human exception: %+v", item)
	}
}

func TestRecoveryStatusFromQuestStatus(t *testing.T) {
	tests := []struct {
		qs   model.QuestStatus
		want RecoveryStatus
	}{
		{model.QuestStatusBlocked, RecoveryStatusPending},
		{model.QuestStatusRunning, RecoveryStatusRunning},
		{model.QuestStatusReviewing, RecoveryStatusRunning},
		{model.QuestStatusSuccess, RecoveryStatusFailed},
		{model.QuestStatusFailed, RecoveryStatusFailed},
		{model.QuestStatusPending, RecoveryStatusFailed},
		{model.QuestStatusCancelled, RecoveryStatusFailed},
	}
	for _, tc := range tests {
		got := RecoveryStatusFromQuestStatus(tc.qs)
		if got != tc.want {
			t.Errorf("RecoveryStatusFromQuestStatus(%q) = %q, want %q", tc.qs, got, tc.want)
		}
	}
}

func TestWorkflowInstanceState(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	// Save nil is no-op
	if err := root.SaveWorkflowInstance(nil); err != nil {
		t.Errorf("save nil should be no-op, got err=%v", err)
	}
	// Save empty ID is no-op
	if err := root.SaveWorkflowInstance(&WorkflowInstanceState{}); err != nil {
		t.Errorf("save empty-ID state should be no-op, got err=%v", err)
	}

	state := &WorkflowInstanceState{
		ID:           "wf_pipeline_01",
		WorkflowID:   "design-then-execute",
		Status:       "design_done",
		LastGlobalID: 15,
		QuestIDs:     []string{"qst_design_01", "qst_exec_02"},
		Meta:         map[string]any{"phase": 2, "owner": "team-a"},
	}
	if err := root.SaveWorkflowInstance(state); err != nil {
		t.Fatalf("SaveWorkflowInstance failed: %v", err)
	}
	if state.CreatedAtMs == 0 {
		t.Error("CreatedAtMs should be set on first save")
	}
	firstCreated := state.CreatedAtMs
	if state.UpdatedAtMs == 0 {
		t.Error("UpdatedAtMs should be set on save")
	}

	// Update: CreatedAtMs preserved, UpdatedAtMs refreshed
	state.Status = "running_execute"
	state.LastGlobalID = 20
	// Simulate time advance (just verify non-zero)
	if err := root.SaveWorkflowInstance(state); err != nil {
		t.Fatalf("second SaveWorkflowInstance failed: %v", err)
	}
	if state.CreatedAtMs != firstCreated {
		t.Errorf("CreatedAtMs changed after update: %d -> %d", firstCreated, state.CreatedAtMs)
	}
	if state.UpdatedAtMs < firstCreated {
		t.Error("UpdatedAtMs should be >= CreatedAtMs")
	}

	got, err := root.GetWorkflowInstance("wf_pipeline_01")
	if err != nil {
		t.Fatalf("GetWorkflowInstance failed: %v", err)
	}
	if got.ID != "wf_pipeline_01" {
		t.Errorf("ID = %q", got.ID)
	}
	if got.WorkflowID != "design-then-execute" {
		t.Errorf("WorkflowID = %q", got.WorkflowID)
	}
	if got.Status != "running_execute" {
		t.Errorf("Status = %q", got.Status)
	}
	if len(got.QuestIDs) != 2 {
		t.Errorf("QuestIDs len = %d", len(got.QuestIDs))
	}
	if got.Meta["phase"].(float64) != 2 { // JSON numbers become float64
		t.Errorf("Meta phase = %v", got.Meta["phase"])
	}

	// Non-existent
	if _, err := root.GetWorkflowInstance("no_such"); err == nil {
		t.Error("GetWorkflowInstance for missing should error")
	}
}
