package notifications

import (
	"encoding/json"

	"code.byted.org/lihuanyu.0w0/gloop/internal/events"
)

type NotificationAction string

const (
	NotificationSend NotificationAction = "send"
	NotificationDrop NotificationAction = "drop"
)

type NotificationDecision struct {
	PolicyName    string             `json:"policy_name"`
	Action        NotificationAction `json:"action"`
	Reason        string             `json:"reason,omitempty"`
	Severity      string             `json:"severity,omitempty"`
	SafeByDefault bool               `json:"safe_by_default"`
}

type NotificationPolicy struct{}

func (NotificationPolicy) Decide(ev events.Event) NotificationDecision {
	switch ev.Type {
	case events.EvtQuestBlocked, events.EvtUserReview, events.EvtQuestWaitingInput, events.EvtQuestFailed:
		return NotificationDecision{
			PolicyName: "action_required_notifications",
			Action:     NotificationSend,
			Reason:     "human action is required",
			Severity:   "action_required",
		}
	case events.EvtQuestApplied:
		// HOTL v0.2：自主闭环的 quest 发"影响通知"，让人感知而非审批。
		// severity=impact 低于 action_required，前端/通知渠道可据此降噪。
		return NotificationDecision{
			PolicyName: "impact_notifications",
			Action:     NotificationSend,
			Reason:     "quest auto-closed, notify human of impact (perceive, not approve)",
			Severity:   "impact",
		}
	case events.EvtQuestSuccess:
		if isAutoAppliedSuccessEvent(ev) {
			return NotificationDecision{
				PolicyName:    "drop_auto_apply_success_wait_for_applied",
				Action:        NotificationDrop,
				Reason:        "auto-apply completion also emits quest.applied; applied is the single user-visible impact notification",
				Severity:      "debug",
				SafeByDefault: true,
			}
		}
		// 非 apply 型 success 没有后续 quest.applied 事件，仍需要一条影响通知。
		return NotificationDecision{
			PolicyName: "impact_notifications",
			Action:     NotificationSend,
			Reason:     "quest auto-closed, notify human of impact (perceive, not approve)",
			Severity:   "impact",
		}
	default:
		return NotificationDecision{
			PolicyName:    "default_drop_non_actionable",
			Action:        NotificationDrop,
			Reason:        "event does not require immediate human action",
			Severity:      "debug",
			SafeByDefault: true,
		}
	}
}

func isAutoAppliedSuccessEvent(ev events.Event) bool {
	if ev.Type != events.EvtQuestSuccess {
		return false
	}
	var payload map[string]any
	if err := json.Unmarshal(ev.Payload, &payload); err != nil {
		return false
	}
	autoApply, _ := payload["auto_apply"].(bool)
	return autoApply
}
