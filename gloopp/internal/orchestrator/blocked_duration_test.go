package orchestrator

import (
	"testing"

	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

// TestQuestBlockedDurationMs_InProgress verifies an actively-blocked quest's
// in-progress blocked interval is counted (accumulated + current).
func TestQuestBlockedDurationMs_InProgress(t *testing.T) {
	now := fsstore.NowMs()
	q := &fsstore.QuestMeta{
		Status:               model.QuestStatusBlocked,
		AccumulatedBlockedMs: 1000,
		BlockedAtMs:          now - 3000,
	}
	got := questBlockedDurationMs(q, now)
	// 1000 accumulated + ~3000 in-progress
	if got < 3900 || got > 4100 {
		t.Fatalf("blocked duration = %d, want ~4000", got)
	}
}

// TestQuestBlockedDurationMs_NotBlocked verifies a running quest only reports
// previously-accumulated blocked time, not any in-progress interval.
func TestQuestBlockedDurationMs_NotBlocked(t *testing.T) {
	now := fsstore.NowMs()
	q := &fsstore.QuestMeta{
		Status:               model.QuestStatusRunning,
		AccumulatedBlockedMs: 5000,
		BlockedAtMs:          0,
	}
	if got := questBlockedDurationMs(q, now); got != 5000 {
		t.Fatalf("blocked duration = %d, want 5000", got)
	}
}

// TestQuestDurationElapsedMs_ExcludesBlocked is the regression guard for the
// reported bug: a quest that started 3h ago but spent 2h blocked should report
// ~1h of actual elapsed time, not 3h — so it is not wrongly duration_exceeded.
func TestQuestDurationElapsedMs_ExcludesBlocked(t *testing.T) {
	now := fsstore.NowMs()
	q := &fsstore.QuestMeta{
		Status:               model.QuestStatusRunning,
		StartedAtMs:          now - 3*60*60*1000, // started 3h ago
		AccumulatedBlockedMs: 2 * 60 * 60 * 1000, // 2h was spent blocked
	}
	got := questDurationElapsedMs(nil, q, now)
	// ~1h of real work time
	wantApprox := int64(60 * 60 * 1000)
	if got < wantApprox-5000 || got > wantApprox+5000 {
		t.Fatalf("elapsed = %d, want ~%d (3h wall - 2h blocked)", got, wantApprox)
	}
}
