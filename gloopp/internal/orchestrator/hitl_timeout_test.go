package orchestrator

import (
	"testing"
	"time"

	"code.byted.org/lihuanyu.0w0/gloop/internal/events"
	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

func TestWaitingInputDefaultTimeoutIsLongEnoughForHITL(t *testing.T) {
	got := waitingInputStateFromSignal(PhaseSignal{
		PhaseComment: "需要用户确认",
	}, 0)

	want := int64(24 * time.Hour / time.Millisecond)
	if got.TimeoutMs != want {
		t.Fatalf("default waiting input timeout = %dms, want %dms", got.TimeoutMs, want)
	}
}

func TestQuestDurationElapsedExcludesWaitingInputInterval(t *testing.T) {
	eng, _ := setupTestEngine(t)
	qs := fsstore.NewQuestStore(eng.root)
	q, err := eng.CreateQuest(t.Context(), "等待输入不计入总时长", model.QuestTypeExecute, "")
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}

	startedAt := int64(1_000_000)
	waitingAt := startedAt + int64(time.Minute/time.Millisecond)
	answeredAt := waitingAt + int64(23*time.Hour/time.Millisecond)
	now := answeredAt + int64(5*time.Minute/time.Millisecond)
	q.StartedAtMs = startedAt
	q.Status = model.QuestStatusRunning
	if err := qs.SaveQuest(q); err != nil {
		t.Fatalf("SaveQuest failed: %v", err)
	}
	if err := qs.AppendEvent(q.ID, &fsstore.QuestEventRow{
		Timestamp: waitingAt,
		Type:      string(events.EvtQuestWaitingInput),
		Payload: map[string]any{
			"question_id": "question_1",
			"question":    "需要用户确认",
		},
	}); err != nil {
		t.Fatalf("Append waiting event failed: %v", err)
	}
	if err := qs.AppendEvent(q.ID, &fsstore.QuestEventRow{
		Timestamp: answeredAt,
		Type:      string(events.EvtQuestNote),
		Payload: map[string]any{
			"answer_id":   "answer_1",
			"question_id": "question_1",
			"content":     "确认继续",
			"source":      "dashboard",
		},
	}); err != nil {
		t.Fatalf("Append answer event failed: %v", err)
	}

	got := questDurationElapsedMs(qs, q, now)
	want := int64(6 * time.Minute / time.Millisecond)
	if got != want {
		t.Fatalf("questDurationElapsedMs = %dms, want active duration %dms", got, want)
	}
}

func TestQuestDurationElapsedExcludesCurrentWaitingInput(t *testing.T) {
	eng, _ := setupTestEngine(t)
	qs := fsstore.NewQuestStore(eng.root)
	q, err := eng.CreateQuest(t.Context(), "当前等待输入不计入总时长", model.QuestTypeExecute, "")
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}

	startedAt := int64(2_000_000)
	waitingAt := startedAt + int64(2*time.Minute/time.Millisecond)
	now := waitingAt + int64(12*time.Hour/time.Millisecond)
	q.StartedAtMs = startedAt
	q.Status = model.QuestStatusWaitingInput
	q.WaitingInput = &fsstore.WaitingInputState{
		QuestionID:    "question_current",
		QuestionText:  "需要用户确认",
		TimeoutMs:     int64(24 * time.Hour / time.Millisecond),
		TimeoutAction: "continue",
		AskedAt:       time.UnixMilli(waitingAt).UTC(),
	}
	if err := qs.SaveQuest(q); err != nil {
		t.Fatalf("SaveQuest failed: %v", err)
	}
	if err := qs.AppendEvent(q.ID, &fsstore.QuestEventRow{
		Timestamp: waitingAt,
		Type:      string(events.EvtQuestWaitingInput),
		Payload: map[string]any{
			"question_id": "question_current",
		},
	}); err != nil {
		t.Fatalf("Append waiting event failed: %v", err)
	}

	got := questDurationElapsedMs(qs, q, now)
	want := int64(2 * time.Minute / time.Millisecond)
	if got != want {
		t.Fatalf("questDurationElapsedMs = %dms, want active duration %dms", got, want)
	}
}
