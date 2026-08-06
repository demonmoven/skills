package quest

import (
	"strings"

	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

// ==================== 失败归因（领域层纯逻辑） ====================
//
// 失败归因是业务规则：根据 quest 的状态、事件、session 等信息，
// 判断失败/阻塞的原因和分类。这是领域逻辑，不应该散落在 API 层。

// FailureStage 失败发生阶段。
type FailureStage string

const (
	StageWarrior FailureStage = "warrior" // 剑士执行阶段
	StageMage    FailureStage = "mage"    // 法师评审阶段
	StageUser    FailureStage = "user"    // 用户终审阶段
	StageApply   FailureStage = "apply"   // 应用阶段
	StageUnknown FailureStage = "unknown" // 未知阶段
)

// FailureReason 失败原因类别。
type FailureReason string

const (
	ReasonBudgetExceeded   FailureReason = "budget_exceeded"   // 预算超支
	ReasonDurationExceeded FailureReason = "duration_exceeded" // 超时
	ReasonAgentError       FailureReason = "agent_error"       // Agent 错误
	ReasonToolError        FailureReason = "tool_error"        // 工具调用错误
	ReasonCommandFailed    FailureReason = "command_failed"    // 命令执行失败
	ReasonCommandTimeout   FailureReason = "command_timeout"   // 命令超时
	ReasonNoProgress       FailureReason = "no_progress"       // 无进展
	ReasonCancelled        FailureReason = "cancelled"         // 被取消
	ReasonUserReject       FailureReason = "reject"            // 用户拒绝
	ReasonApplyFailed      FailureReason = "apply_failed"      // Apply 失败
	ReasonUnknown          FailureReason = "unknown"           // 未知原因
)

// FailureAttribution 结构化失败归因。
type FailureAttribution struct {
	Stage       FailureStage   `json:"stage"`
	Reason      FailureReason  `json:"reason"`
	Category    string         `json:"category,omitempty"`
	Message     string         `json:"message,omitempty"`
	Recoverable bool           `json:"recoverable"`
	Actions     []string       `json:"actions,omitempty"`
	Details     map[string]any `json:"details,omitempty"`
}

// ==================== 失败归因规则引擎 ====================

// Attributable 归因所需的最小信息集合。
// 任何能提供这些信息的对象都可以做失败归因。
type Attributable interface {
	GetStatus() model.QuestStatus
	GetBlockedReason() string
	GetFinalVerdict() model.QuestVerdict
	GetFinalComment() string
	GetApplyStatus() model.ApplyStatus
	GetApplyError() string
}

// AttributeFailure 根据 quest 元信息进行失败归因。
// 这是领域层的纯逻辑，不依赖存储。
func AttributeFailure(q Attributable) *FailureAttribution {
	if q == nil {
		return nil
	}

	status := q.GetStatus()

	switch status {
	case model.QuestStatusBlocked:
		return attributeBlocked(q)
	case model.QuestStatusFailed:
		return attributeFailed(q)
	case model.QuestStatusCancelled:
		return &FailureAttribution{
			Stage:       StageUnknown,
			Reason:      ReasonCancelled,
			Recoverable: false,
		}
	case model.QuestStatusSuccess:
		if q.GetApplyStatus() == model.ApplyStatusFailed {
			return &FailureAttribution{
				Stage:       StageApply,
				Reason:      ReasonApplyFailed,
				Message:     q.GetApplyError(),
				Recoverable: true,
				Actions:     []string{"重试 apply", "检查工作区状态"},
			}
		}
		return nil
	default:
		return nil
	}
}

func attributeBlocked(q Attributable) *FailureAttribution {
	reason := q.GetBlockedReason()
	attr := &FailureAttribution{
		Stage:  StageUnknown,
		Reason: ReasonUnknown,
	}

	// 按从具体到一般的顺序匹配，避免宽泛关键字先命中
	switch {
	// 用户确认超时（最具体，优先匹配）
	case containsAny(reason, "用户确认超时", "user_confirm_timeout", "等待用户终审超时"):
		attr.Reason = ReasonDurationExceeded
		attr.Stage = StageUser
		attr.Recoverable = true
		attr.Actions = []string{"用户确认继续", "取消委托"}

	// 预算超支
	case containsAny(reason, "预算", "budget", "turns", "回合", "超支", "no_progress"):
		attr.Reason = ReasonBudgetExceeded
		attr.Stage = StageWarrior
		attr.Recoverable = true
		attr.Actions = []string{"增加预算", "减少任务范围"}

	// 时长/超时（放在用户确认之后，避免误匹配）
	case containsAny(reason, "超时", "timeout", "duration", "超过时长", "最大时长"):
		attr.Reason = ReasonDurationExceeded
		attr.Stage = StageWarrior
		attr.Recoverable = true
		attr.Actions = []string{"延长时间", "简化任务"}

	// Agent / 执行器错误
	case containsAny(reason, "agent", "执行器", "executor", "consecutive_errors"):
		attr.Reason = ReasonAgentError
		attr.Stage = StageWarrior
		attr.Recoverable = true
		attr.Actions = []string{"检查 agent 配置", "更换冒险者"}

	// 工具/命令错误
	case containsAny(reason, "tool", "工具", "command", "命令"):
		attr.Reason = ReasonToolError
		attr.Stage = StageWarrior
		attr.Recoverable = true

	default:
		attr.Reason = ReasonUnknown
		attr.Recoverable = true
	}

	attr.Message = reason
	return attr
}

func attributeFailed(q Attributable) *FailureAttribution {
	verdict := q.GetFinalVerdict()
	comment := q.GetFinalComment()
	attr := &FailureAttribution{
		Stage:  StageUnknown,
		Reason: ReasonUnknown,
	}

	switch verdict {
	case model.VerdictReject:
		attr.Reason = ReasonUserReject
		attr.Stage = StageMage // 默认是法师拒绝，后续可根据上下文细化
		attr.Recoverable = true
		attr.Actions = []string{"根据评审意见修改后重新提交"}
	default:
		// 没有 verdict 的失败，可能是执行过程中出错
		switch {
		case containsAny(comment, "agent", "执行器", "executor"):
			attr.Reason = ReasonAgentError
			attr.Stage = StageWarrior
			attr.Recoverable = true
		default:
			attr.Reason = ReasonUnknown
			attr.Stage = StageWarrior
		}
	}

	attr.Message = comment
	return attr
}

// ==================== 辅助 ====================

func containsAny(s string, keywords ...string) bool {
	s = strings.ToLower(s)
	for _, kw := range keywords {
		if strings.Contains(s, strings.ToLower(kw)) {
			return true
		}
	}
	return false
}
