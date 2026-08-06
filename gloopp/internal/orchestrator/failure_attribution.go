package orchestrator

import (
	"fmt"

	"code.byted.org/lihuanyu.0w0/gloop/internal/domain/quest"
	"code.byted.org/lihuanyu.0w0/gloop/internal/events"
)

type failureAttribution struct {
	Stage       string         `json:"stage"`
	Reason      string         `json:"reason"`
	Category    string         `json:"category,omitempty"`
	Message     string         `json:"message,omitempty"`
	Recoverable bool           `json:"recoverable"`
	Actions     []string       `json:"actions,omitempty"`
	Details     map[string]any `json:"details,omitempty"`
}

func (e *Engine) publishFailureAttribution(qid string, result PhaseSignal, stage string) {
	attr := attributionFromToolResult(result, stage)
	if attr.Reason == "" {
		return
	}
	e.cacheFailureAttribution(qid, attr)
	e.publish(qid, "", events.EvtFailureAttributed, map[string]any{
		"stage":       attr.Stage,
		"reason":      attr.Reason,
		"category":    attr.Category,
		"message":     attr.Message,
		"recoverable": attr.Recoverable,
		"actions":     attr.Actions,
		"details":     attr.Details,
	})
}

func (e *Engine) cacheFailureAttribution(qid string, attr failureAttribution) {
	// 通过 QuestService 更新失败归因（统一入口）
	domainAttr := &quest.FailureAttribution{
		Stage:       quest.FailureStage(attr.Stage),
		Reason:      quest.FailureReason(attr.Reason),
		Category:    attr.Category,
		Message:     attr.Message,
		Recoverable: attr.Recoverable,
		Actions:     append([]string(nil), attr.Actions...),
		Details:     cloneAnyMap(attr.Details),
	}
	if err := e.questService.UpdateFailureAttribution(qid, domainAttr); err != nil {
		e.log.Warn("保存失败归因摘要失败", "qid", qid, "err", err)
	}
}

func attributionFromToolResult(result PhaseSignal, stage string) failureAttribution {
	data, _ := result.Data.(map[string]any)
	if data == nil {
		data = map[string]any{}
	}
	reason := stringFromAny(data["reason"])
	if reason == "" {
		reason = result.Message
	}
	category := stringFromAny(data["category"])
	actions := stringSliceFromAny(data["actions"])
	details := map[string]any{}
	for key, value := range data {
		switch key {
		case "reason", "category", "actions":
			continue
		default:
			details[key] = value
		}
	}
	return failureAttribution{
		Stage:       stage,
		Reason:      reason,
		Category:    category,
		Message:     result.PhaseComment,
		Recoverable: isRecoverableFailure(reason, category, actions),
		Actions:     actions,
		Details:     details,
	}
}

func isRecoverableFailure(reason, category string, actions []string) bool {
	if len(actions) > 0 {
		return true
	}
	switch category {
	case "auth", "configuration", "rate_limit", "transient":
		return true
	}
	switch reason {
	case "duration_exceeded", "turn_limit", "no_progress", "agent_consecutive_errors":
		return true
	default:
		return false
	}
}

func stringFromAny(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case fmt.Stringer:
		return x.String()
	default:
		return ""
	}
}

func stringSliceFromAny(v any) []string {
	switch x := v.(type) {
	case []string:
		return append([]string(nil), x...)
	case []any:
		out := make([]string, 0, len(x))
		for _, item := range x {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func cloneAnyMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
