package orchestrator

import (
	"testing"

	"code.byted.org/lihuanyu.0w0/gloop/internal/events"
	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
	"code.byted.org/lihuanyu.0w0/gloop/internal/policy"
)

func TestSettleQuestEventCompletesRunningRecoveryFromQuestState(t *testing.T) {
	eng, _ := setupTestEngine(t)
	qs := fsstore.NewQuestStore(eng.root)
	q := &fsstore.QuestMeta{
		ID:          "q_recovery_success",
		ShortID:     "recok",
		Query:       "recovery success",
		Type:        model.QuestTypeExecute,
		Status:      model.QuestStatusUserReview,
		CreatedBy:   string(model.SourceUser),
		CreatedAtMs: fsstore.NowMs(),
	}
	if err := qs.SaveQuest(q); err != nil {
		t.Fatalf("SaveQuest failed: %v", err)
	}
	if err := eng.root.SaveRecoveryState(&fsstore.RecoveryState{
		QuestID:      q.ID,
		Status:       fsstore.RecoveryStatusRunning,
		LastAction:   policy.ActionRetry,
		BlockReason:  "no_progress",
		MaxAttempts:  1,
		AttemptIndex: 0,
	}); err != nil {
		t.Fatalf("SaveRecoveryState failed: %v", err)
	}

	eng.settleQuestEvent(events.Event{QuestID: q.ID, Type: events.EvtQuestBlocked})

	state, err := eng.root.GetRecoveryState(q.ID)
	if err != nil {
		t.Fatalf("GetRecoveryState failed: %v", err)
	}
	if state.Status != fsstore.RecoveryStatusSucceeded {
		t.Fatalf("recovery should settle from persisted quest state, got %+v", state)
	}
}

func TestSettleQuestEventDoesNotRollbackTerminalRecovery(t *testing.T) {
	eng, _ := setupTestEngine(t)
	qs := fsstore.NewQuestStore(eng.root)
	q := &fsstore.QuestMeta{
		ID:                "q_recovery_terminal",
		ShortID:           "recterm",
		Query:             "recovery terminal",
		Type:              model.QuestTypeExecute,
		Status:            model.QuestStatusBlocked,
		BlockedReasonCode: "no_progress",
		CreatedBy:         string(model.SourceUser),
		CreatedAtMs:       fsstore.NowMs(),
	}
	if err := qs.SaveQuest(q); err != nil {
		t.Fatalf("SaveQuest failed: %v", err)
	}
	if err := eng.root.SaveRecoveryState(&fsstore.RecoveryState{
		QuestID:     q.ID,
		Status:      fsstore.RecoveryStatusSucceeded,
		LastAction:  policy.ActionRetry,
		BlockReason: "no_progress",
	}); err != nil {
		t.Fatalf("SaveRecoveryState failed: %v", err)
	}

	eng.settleQuestEvent(events.Event{QuestID: q.ID, Type: events.EvtQuestBlocked})

	state, err := eng.root.GetRecoveryState(q.ID)
	if err != nil {
		t.Fatalf("GetRecoveryState failed: %v", err)
	}
	if state.Status != fsstore.RecoveryStatusSucceeded {
		t.Fatalf("terminal recovery state should be idempotent, got %+v", state)
	}
}

func TestSettleQuestEventMarksSameReasonRecoveryBlockedFailed(t *testing.T) {
	eng, _ := setupTestEngine(t)
	qs := fsstore.NewQuestStore(eng.root)
	q := &fsstore.QuestMeta{
		ID:                "q_recovery_blocked",
		ShortID:           "recblk",
		Query:             "recovery blocked",
		Type:              model.QuestTypeExecute,
		Status:            model.QuestStatusBlocked,
		BlockedReasonCode: "no_progress",
		CreatedBy:         string(model.SourceUser),
		CreatedAtMs:       fsstore.NowMs(),
	}
	if err := qs.SaveQuest(q); err != nil {
		t.Fatalf("SaveQuest failed: %v", err)
	}
	if err := eng.root.SaveRecoveryState(&fsstore.RecoveryState{
		QuestID:      q.ID,
		Status:       fsstore.RecoveryStatusRunning,
		LastAction:   policy.ActionRetry,
		BlockReason:  "no_progress",
		MaxAttempts:  2,
		AttemptIndex: 0,
	}); err != nil {
		t.Fatalf("SaveRecoveryState failed: %v", err)
	}

	eng.settleQuestEvent(events.Event{QuestID: q.ID, Type: events.EvtQuestBlocked})

	state, err := eng.root.GetRecoveryState(q.ID)
	if err != nil {
		t.Fatalf("GetRecoveryState failed: %v", err)
	}
	if state.Status != fsstore.RecoveryStatusFailed || state.LastError != "no_progress" {
		t.Fatalf("same blocked reason should fail recovery, got %+v", state)
	}
}

func TestSettleQuestEventRecordsAutomationTerminalTrustOnce(t *testing.T) {
	eng, _ := setupTestEngine(t)
	qs := fsstore.NewQuestStore(eng.root)
	q := &fsstore.QuestMeta{
		ID:          "q_auto_success",
		ShortID:     "autook",
		Query:       "automation success",
		Type:        model.QuestTypeExecute,
		Status:      model.QuestStatusSuccess,
		CreatedBy:   model.QuestSourcePrefix + "auto_test",
		CreatedAtMs: fsstore.NowMs(),
	}
	if err := qs.SaveQuest(q); err != nil {
		t.Fatalf("SaveQuest failed: %v", err)
	}

	eng.settleQuestEvent(events.Event{QuestID: q.ID, Type: events.EvtQuestSuccess})
	eng.settleQuestEvent(events.Event{QuestID: q.ID, Type: events.EvtQuestSuccess})

	state, err := eng.root.GetAutomationTrustState("auto_test")
	if err != nil {
		t.Fatalf("GetAutomationTrustState failed: %v", err)
	}
	if state.TotalRuns != 1 || state.TotalSuccess != 1 || len(state.Recent) != 1 {
		t.Fatalf("automation trust terminal outcome should be counted once: %+v", state)
	}
}

func TestSettleQuestEventProjectsNeedsHuman(t *testing.T) {
	eng, _ := setupTestEngine(t)
	qs := fsstore.NewQuestStore(eng.root)
	q := &fsstore.QuestMeta{
		ID:                "q_needs_human",
		ShortID:           "needhuman",
		Query:             "needs human",
		Type:              model.QuestTypeExecute,
		Status:            model.QuestStatusBlocked,
		BlockedReason:     "agent auth failed",
		BlockedReasonCode: "agent_error_auth",
		CreatedBy:         string(model.SourceUser),
		CreatedAtMs:       fsstore.NowMs(),
	}
	if err := qs.SaveQuest(q); err != nil {
		t.Fatalf("SaveQuest failed: %v", err)
	}

	eng.settleQuestEvent(events.Event{QuestID: q.ID, Type: events.EvtQuestBlocked})

	rows, err := qs.ReadEvents(q.ID, 0)
	if err != nil {
		t.Fatalf("ReadEvents failed: %v", err)
	}
	for _, row := range rows {
		if row.Type == string(events.EvtNeedsHuman) {
			payload, ok := row.Payload.(map[string]any)
			if !ok || payload["recommended_action"] == "" || payload["risk_level"] != "high" {
				t.Fatalf("bad needs_human payload: %+v", row.Payload)
			}
			return
		}
	}
	t.Fatalf("missing needs_human event: %+v", rows)
}
