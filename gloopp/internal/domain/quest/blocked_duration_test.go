package quest

import (
	"testing"

	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

// TestApplyTransitionResult_StampsBlockedAt verifies entering the blocked state
// records a BlockedAtMs marker so paused time can later be excluded from the
// quest duration budget.
func TestApplyTransitionResult_StampsBlockedAt(t *testing.T) {
	q := &QuestMeta{Status: model.QuestStatusRunning}
	q.ApplyTransitionResult(TransitionResult{NewStatus: model.QuestStatusBlocked})

	if q.Status != model.QuestStatusBlocked {
		t.Fatalf("status = %s, want blocked", q.Status)
	}
	if q.BlockedAtMs <= 0 {
		t.Fatalf("BlockedAtMs should be stamped on entering blocked, got %d", q.BlockedAtMs)
	}
	if q.AccumulatedBlockedMs != 0 {
		t.Fatalf("AccumulatedBlockedMs should stay 0 until unblocked, got %d", q.AccumulatedBlockedMs)
	}
}

// TestApplyTransitionResult_FlushesBlockedDuration verifies leaving the blocked
// state accumulates the elapsed blocked interval and clears the marker. This is
// the core of the duration fix: a quest that sits blocked (e.g. awaiting a
// human) must not have that idle wall-clock time count against its budget.
func TestApplyTransitionResult_FlushesBlockedDuration(t *testing.T) {
	q := &QuestMeta{Status: model.QuestStatusRunning}
	q.ApplyTransitionResult(TransitionResult{NewStatus: model.QuestStatusBlocked})
	// Simulate the quest having been blocked for ~2 hours.
	q.BlockedAtMs = model.NowMs() - 2*60*60*1000

	q.ApplyTransitionResult(TransitionResult{NewStatus: model.QuestStatusRunning, ClearBlocked: true})

	if q.Status != model.QuestStatusRunning {
		t.Fatalf("status = %s, want running", q.Status)
	}
	if q.BlockedAtMs != 0 {
		t.Fatalf("BlockedAtMs should be cleared after unblock, got %d", q.BlockedAtMs)
	}
	// ~2h accumulated (allow generous slack for test execution time).
	if q.AccumulatedBlockedMs < 2*60*60*1000-5000 {
		t.Fatalf("AccumulatedBlockedMs should capture the blocked interval, got %d", q.AccumulatedBlockedMs)
	}
}

// TestApplyTransitionResult_AccumulatesAcrossMultipleBlocks verifies repeated
// block/unblock cycles sum their durations rather than overwriting.
func TestApplyTransitionResult_AccumulatesAcrossMultipleBlocks(t *testing.T) {
	q := &QuestMeta{Status: model.QuestStatusRunning}

	// First block cycle: ~1h.
	q.ApplyTransitionResult(TransitionResult{NewStatus: model.QuestStatusBlocked})
	q.BlockedAtMs = model.NowMs() - 60*60*1000
	q.ApplyTransitionResult(TransitionResult{NewStatus: model.QuestStatusRunning, ClearBlocked: true})
	firstAccum := q.AccumulatedBlockedMs

	// Second block cycle: another ~1h.
	q.ApplyTransitionResult(TransitionResult{NewStatus: model.QuestStatusBlocked})
	q.BlockedAtMs = model.NowMs() - 60*60*1000
	q.ApplyTransitionResult(TransitionResult{NewStatus: model.QuestStatusRunning, ClearBlocked: true})

	if q.AccumulatedBlockedMs <= firstAccum {
		t.Fatalf("AccumulatedBlockedMs should grow across cycles: first=%d total=%d", firstAccum, q.AccumulatedBlockedMs)
	}
	if q.AccumulatedBlockedMs < 2*60*60*1000-5000 {
		t.Fatalf("two ~1h blocks should sum to ~2h, got %d", q.AccumulatedBlockedMs)
	}
}
