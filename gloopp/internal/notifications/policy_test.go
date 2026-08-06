package notifications

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"code.byted.org/lihuanyu.0w0/gloop/internal/events"
)

func TestNotificationPolicy_DefaultActionRequiredOnly(t *testing.T) {
	p := NotificationPolicy{}
	sendEvents := []events.EventType{
		events.EvtQuestBlocked,
		events.EvtUserReview,
		events.EvtQuestWaitingInput,
		events.EvtQuestFailed,
	}
	for _, typ := range sendEvents {
		got := p.Decide(events.Event{Type: typ})
		if got.Action != NotificationSend || got.Severity != "action_required" {
			t.Fatalf("%s decision = %+v, want send/action_required", typ, got)
		}
	}

	// HOTL v0.2：自主闭环事件发"影响通知"（send, severity=impact），不再 drop
	impactEvents := []events.EventType{
		events.EvtQuestSuccess,
		events.EvtQuestApplied,
	}
	for _, typ := range impactEvents {
		got := p.Decide(events.Event{Type: typ})
		if got.Action != NotificationSend || got.Severity != "impact" {
			t.Fatalf("%s decision = %+v, want send/impact", typ, got)
		}
	}

	dropEvents := []events.EventType{
		events.EvtQuestStarted,
		events.EvtMicroTurn,
	}
	for _, typ := range dropEvents {
		got := p.Decide(events.Event{Type: typ})
		if got.Action != NotificationDrop || !got.SafeByDefault {
			t.Fatalf("%s decision = %+v, want safe drop", typ, got)
		}
	}
}

type policyTestNotifier struct {
	cards []map[string]any
	now   time.Time
}

func (n *policyTestNotifier) Available() bool                                     { return true }
func (n *policyTestNotifier) StatusError() string                                 { return "" }
func (n *policyTestNotifier) SetBaseURL(string)                                   {}
func (n *policyTestNotifier) BaseURL() string                                     { return "http://127.0.0.1:37317" }
func (n *policyTestNotifier) Recipient() (string, string)                         { return "user", "user" }
func (n *policyTestNotifier) Send(context.Context, string, string) (bool, string) { return true, "ok" }
func (n *policyTestNotifier) SendCard(_ context.Context, card map[string]any) (bool, string) {
	n.cards = append(n.cards, card)
	return true, "ok"
}

type policyTestStore struct{}

func (policyTestStore) LoadQuest(qid string) (QuestInfo, error) {
	return QuestInfo{ID: qid, ShortID: "short", Title: "title", Status: "status"}, nil
}

func (policyTestStore) LoadRecovery(qid string) (RecoveryInfo, error) {
	return RecoveryInfo{
		Status:        "succeeded",
		PolicyName:    "no_progress_retry",
		LastAction:    "retry",
		BlockReason:   "no_progress",
		OriginalError: "连续无进展",
	}, nil
}

func TestEventSubscriberUsesNotificationPolicy(t *testing.T) {
	notifier := &policyTestNotifier{}
	sub := NewEventSubscriber(nil, notifier, policyTestStore{}, nil)
	// HOTL v0.2：success 发影响通知（不再 drop）
	sub.handleEvent(events.Event{Type: events.EvtQuestSuccess, QuestID: "q1", Payload: events.MarshalPayload(map[string]any{"comment": "done"})})
	if len(notifier.cards) != 1 {
		t.Fatalf("success should send impact card, sent cards=%d", len(notifier.cards))
	}
	sub.handleEvent(events.Event{Type: events.EvtQuestBlocked, QuestID: "q1", Payload: events.MarshalPayload(map[string]any{"reason": "duration_exceeded"})})
	if len(notifier.cards) != 2 {
		t.Fatalf("blocked should be sent, sent cards=%d", len(notifier.cards))
	}
}

func TestEventSubscriberAutoApplySuccessAndAppliedSendOneImpactCard(t *testing.T) {
	notifier := &policyTestNotifier{}
	sub := NewEventSubscriber(nil, notifier, policyTestStore{}, nil)

	sub.handleEvent(events.Event{
		Type:    events.EvtQuestSuccess,
		QuestID: "q1",
		Payload: events.MarshalPayload(map[string]any{
			"verdict":    "pass",
			"comment":    "done",
			"auto_apply": true,
		}),
	})
	sub.handleEvent(events.Event{
		Type:    events.EvtQuestApplied,
		QuestID: "q1",
		Payload: events.MarshalPayload(map[string]any{
			"mode":     "worktree",
			"warnings": 0,
		}),
	})

	if len(notifier.cards) != 1 {
		t.Fatalf("auto-apply success followed by applied should send one impact card, sent cards=%d", len(notifier.cards))
	}
}

func TestEventSubscriberDeduplicatesSameQuestAndType(t *testing.T) {
	notifier := &policyTestNotifier{}
	sub := NewEventSubscriber(nil, notifier, policyTestStore{}, nil)
	now := time.Date(2026, 6, 21, 10, 0, 0, 0, time.UTC)
	sub.now = func() time.Time { return now }

	sub.handleEvent(events.Event{Type: events.EvtQuestBlocked, QuestID: "q1", Payload: events.MarshalPayload(map[string]any{"reason": "no_progress"})})
	sub.handleEvent(events.Event{Type: events.EvtQuestBlocked, QuestID: "q1", Payload: events.MarshalPayload(map[string]any{"reason": "no_progress"})})
	if len(notifier.cards) != 1 {
		t.Fatalf("duplicate blocked notification should be dropped, sent cards=%d", len(notifier.cards))
	}
	sub.handleEvent(events.Event{Type: events.EvtQuestWaitingInput, QuestID: "q1", Payload: events.MarshalPayload(map[string]any{"question": "补充信息?"})})
	if len(notifier.cards) != 2 {
		t.Fatalf("waiting_input should not be deduped by blocked event, sent cards=%d", len(notifier.cards))
	}
	now = now.Add(notificationDedupWindow + time.Second)
	sub.handleEvent(events.Event{Type: events.EvtQuestBlocked, QuestID: "q1", Payload: events.MarshalPayload(map[string]any{"reason": "no_progress"})})
	if len(notifier.cards) != 3 {
		t.Fatalf("blocked notification should be sent after dedup window, sent cards=%d", len(notifier.cards))
	}
}

func TestEventSubscriberAddsRecoverySummaryToUserReview(t *testing.T) {
	notifier := &policyTestNotifier{}
	sub := NewEventSubscriber(nil, notifier, policyTestStore{}, nil)
	sub.handleEvent(events.Event{Type: events.EvtUserReview, QuestID: "q1", Payload: events.MarshalPayload(map[string]any{"comment": "请审核"})})
	if len(notifier.cards) != 1 {
		t.Fatalf("user review should send one card, sent cards=%d", len(notifier.cards))
	}
	raw, _ := json.Marshal(notifier.cards[0])
	body := string(raw)
	if !strings.Contains(body, "自动恢复摘要") || !strings.Contains(body, "no_progress_retry") || !strings.Contains(body, "请审核") {
		t.Fatalf("card should include recovery summary and review comment: %s", body)
	}
}
