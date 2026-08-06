package fsstore

import (
	"fmt"
	"strings"

	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

type HumanExceptionItem struct {
	ID                string   `json:"id"`
	QuestID           string   `json:"quest_id"`
	Reason            string   `json:"reason"`
	RecommendedAction string   `json:"recommended_action"`
	EvidenceSummary   string   `json:"evidence_summary,omitempty"`
	RiskLevel         string   `json:"risk_level"`
	Priority          int      `json:"priority"`
	AvailableActions  []string `json:"available_actions"`
	AuditRef          string   `json:"audit_ref"`
	SourceStatus      string   `json:"source_status"`
	CreatedAtMs       int64    `json:"created_at_ms,omitempty"`
}

func HumanExceptionFromQuest(q *QuestMeta) *HumanExceptionItem {
	if q == nil {
		return nil
	}
	base := &HumanExceptionItem{
		ID:           "hex_" + q.ID,
		QuestID:      q.ID,
		AuditRef:     "/quests/" + q.ID,
		SourceStatus: string(q.Status),
		CreatedAtMs:  q.UpdatedAtMs,
		RiskLevel:    "medium",
		Priority:     50,
	}
	if base.CreatedAtMs == 0 {
		base.CreatedAtMs = q.CreatedAtMs
	}
	switch q.Status {
	case model.QuestStatusBlocked:
		base.Reason = nonEmpty(q.BlockedReason, "任务已阻塞")
		base.RecommendedAction = blockedRecommendedAction(q)
		base.EvidenceSummary = exceptionEvidenceSummary(q)
		base.RiskLevel = "high"
		base.Priority = blockedPriority(q)
		base.AvailableActions = []string{"continue", "user-review", "cancel"}
		return base
	case model.QuestStatusWaitingInput:
		question := "Agent 等待你补充信息"
		if q.WaitingInput != nil && q.WaitingInput.QuestionText != "" {
			question = q.WaitingInput.QuestionText
		}
		base.Reason = question
		base.RecommendedAction = "回答 Agent 提问后继续执行。"
		if q.WaitingInput != nil {
			base.EvidenceSummary = nonEmpty(q.WaitingInput.Asker, q.WaitingInput.PhaseType)
		}
		base.RiskLevel = "medium"
		base.Priority = 70
		base.AvailableActions = []string{"answer", "cancel"}
		return base
	case model.QuestStatusUserReview:
		base.Reason = nonEmpty(q.FinalComment, "等待人工审核最终结果")
		base.RecommendedAction = "审核交付后选择通过、返工或拒绝。"
		if q.FinalVerdict != "" {
			base.EvidenceSummary = string(q.FinalVerdict)
		}
		base.RiskLevel = "medium"
		base.Priority = 60
		base.AvailableActions = []string{"pass", "request_changes", "reject"}
		return base
	default:
		if q.ApplyStatus == model.ApplyStatusFailed {
			base.SourceStatus = "apply_failed"
			base.Reason = nonEmpty(q.ApplyError, "应用失败")
			base.RecommendedAction = "查看 apply 错误并重试或丢弃变更。"
			base.EvidenceSummary = truncateExceptionText(q.ApplyError, 180)
			base.RiskLevel = "high"
			base.Priority = 40
			base.AvailableActions = []string{"apply", "discard"}
			return base
		}
	}
	return nil
}

func nonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func blockedRecommendedAction(q *QuestMeta) string {
	if q == nil {
		return "处理阻塞后恢复执行，或转入人工终审。"
	}
	switch q.BlockedReasonCode {
	case "agent_error_auth":
		return "完成 Agent 登录或授权后恢复执行。"
	case "agent_error_configuration":
		return "修正 Agent 配置、命令或模型后恢复执行。"
	case "duration_exceeded":
		return "检查是否卡住；确认可继续时追加预算并恢复。"
	case "max_rework_exceeded":
		return "返工已达上限，建议人工裁决通过、返工或拒绝。"
	case "waiting_input_timeout":
		return "补充缺失信息，或取消该委托。"
	default:
		return "处理阻塞后恢复执行，或转入人工终审。"
	}
}

func blockedPriority(q *QuestMeta) int {
	if q == nil {
		return 50
	}
	switch q.BlockedReasonCode {
	case "agent_error_auth", "agent_error_configuration":
		return 10
	case "duration_exceeded":
		return 20
	case "max_rework_exceeded":
		return 30
	case "waiting_input_timeout":
		return 70
	default:
		return 50
	}
}

func exceptionEvidenceSummary(q *QuestMeta) string {
	if q == nil {
		return ""
	}
	parts := []string{}
	if q.BlockedReasonCode != "" {
		parts = append(parts, "code="+q.BlockedReasonCode)
	}
	if q.BlockedCategory != "" {
		parts = append(parts, "category="+q.BlockedCategory)
	}
	if q.FailureAttribution != nil {
		if q.FailureAttribution.Reason != "" {
			parts = append(parts, "reason="+q.FailureAttribution.Reason)
		}
		if len(q.FailureAttribution.Actions) > 0 {
			parts = append(parts, "actions="+strings.Join(q.FailureAttribution.Actions, ","))
		}
	}
	if len(parts) == 0 {
		return truncateExceptionText(q.BlockedReason, 180)
	}
	return truncateExceptionText(strings.Join(parts, "; "), 220)
}

func truncateExceptionText(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 || len([]rune(s)) <= max {
		return s
	}
	r := []rune(s)
	return fmt.Sprintf("%s...", string(r[:max]))
}
