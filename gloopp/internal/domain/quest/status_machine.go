package quest

import (
	"fmt"

	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

// ==================== Quest 状态机（领域层纯逻辑，无存储依赖） ====================
//
// 主线: pending → running → reviewing → user_review → success/failed/cancelled
// 异常: * → blocked → running/user_review/cancelled
// 返工: reviewing/user_review → running (rework++)

// ValidTransitions 定义合法的状态迁移。
// 这是领域层的单一真相源，所有状态变更都必须经过这里校验。
var ValidTransitions = map[model.QuestStatus][]model.QuestStatus{
	"": {
		model.QuestStatusPending,
	},
	model.QuestStatusPending: {
		model.QuestStatusRunning,
		model.QuestStatusCancelled,
	},
	model.QuestStatusRunning: {
		model.QuestStatusReviewing,    // 剑士完成 → 法师评审
		model.QuestStatusWaitingInput, // 等待用户回答具体问题
		model.QuestStatusUserReview,   // 剑士完成 → 直接用户终审（quick 模式、无评审）
		model.QuestStatusFailed,       // 致命错误
		model.QuestStatusBlocked,      // 阻塞
		model.QuestStatusCancelled,    // 用户取消
	},
	model.QuestStatusWaitingInput: {
		model.QuestStatusRunning,    // 收到回复 / 超时继续
		model.QuestStatusBlocked,    // 超时且策略要求阻塞
		model.QuestStatusUserReview, // 超时提交用户终审
		model.QuestStatusCancelled,  // 用户取消
	},
	model.QuestStatusReviewing: {
		model.QuestStatusRunning,    // rework → 剑士返工
		model.QuestStatusUserReview, // 法师通过 → 用户终审
		model.QuestStatusSuccess,    // 平台托管副作用已生效的 automation → 审计通过后直接完成
		model.QuestStatusFailed,     // 评审拒绝
		model.QuestStatusBlocked,    // 法师执行错误 / 无响应
		model.QuestStatusCancelled,  // 用户取消
	},
	model.QuestStatusUserReview: {
		model.QuestStatusRunning,   // 用户要求返工
		model.QuestStatusSuccess,   // 用户通过
		model.QuestStatusFailed,    // 用户拒绝
		model.QuestStatusBlocked,   // 用户确认超时 / 等待外部输入
		model.QuestStatusCancelled, // 用户取消
	},
	model.QuestStatusBlocked: {
		model.QuestStatusRunning,    // 解除阻塞
		model.QuestStatusUserReview, // 直接提交用户终审
		model.QuestStatusCancelled,  // 取消
	},
}

// terminalStatuses 终态集合：进入后不能再迁移出去。
var terminalStatuses = map[model.QuestStatus]bool{
	model.QuestStatusSuccess:   true,
	model.QuestStatusFailed:    true,
	model.QuestStatusCancelled: true,
}

// IsTerminal 返回该状态是否为终态。
func IsTerminal(status model.QuestStatus) bool {
	return terminalStatuses[status]
}

// CanTransition 校验从 cur 能否迁移到 next。
// 纯函数，无副作用。
func CanTransition(cur, next model.QuestStatus) bool {
	// 自环一律不合法
	if cur == next {
		return false
	}
	// 终态不允许迁移出去
	if terminalStatuses[cur] {
		return false
	}
	allowed, ok := ValidTransitions[cur]
	if !ok {
		// 未知当前状态：只要 next 不是终态就允许（兼容老数据）
		return !terminalStatuses[next]
	}
	for _, s := range allowed {
		if s == next {
			return true
		}
	}
	return false
}

// ==================== 状态迁移结果类型 ====================

// TransitionResult 封装状态迁移的结果，包含状态变更和联动字段更新。
// 领域层只计算"应该变成什么样"，不负责持久化。
type TransitionResult struct {
	NewStatus model.QuestStatus

	// 联动字段（由具体迁移方法填充）
	FinalVerdict      model.QuestVerdict
	FinalComment      string
	BlockedReason     string
	BlockedReasonCode string
	BlockedCategory   string
	ReviewHints       string
	ReworkCount       int  // 返工次数增量（通常是 0 或 1）
	ResumeCount       int  // 恢复次数增量
	ClearBlocked      bool // 是否清空阻塞原因
	SetCompleted      bool // 是否设置完成时间
	SetStarted        bool // 是否设置开始时间
}

// ==================== 具体迁移操作（纯函数，返回变更结果） ====================
//
// 每个迁移函数接收当前状态和必要参数，返回 TransitionResult 或 error。
// 调用方（通常是应用服务层）负责把 result 应用到 quest 对象并持久化。
// 这样设计的好处是：领域逻辑纯净化，容易测试；持久化和领域决策解耦。

// ResumeFromBlocked 从阻塞状态恢复执行。
// 只能从 blocked 状态调用。
func ResumeFromBlocked(curStatus model.QuestStatus, comment string, currentResumeCount int) (TransitionResult, error) {
	if curStatus != model.QuestStatusBlocked {
		return TransitionResult{}, fmt.Errorf("非法状态: 只能从 blocked 状态恢复，当前状态: %s", curStatus)
	}
	if err := checkTransition(curStatus, model.QuestStatusRunning); err != nil {
		return TransitionResult{}, err
	}
	return TransitionResult{
		NewStatus:    model.QuestStatusRunning,
		ReviewHints:  comment,
		ResumeCount:  currentResumeCount + 1,
		ClearBlocked: true,
	}, nil
}

// MoveBlockedToUserReview 从阻塞状态直接转入用户终审。
// 只能从 blocked 状态调用。
func MoveBlockedToUserReview(curStatus model.QuestStatus, comment string) (TransitionResult, error) {
	if curStatus != model.QuestStatusBlocked {
		return TransitionResult{}, fmt.Errorf("非法状态: 只能从 blocked 状态转入用户终审，当前状态: %s", curStatus)
	}
	if err := checkTransition(curStatus, model.QuestStatusUserReview); err != nil {
		return TransitionResult{}, err
	}
	if comment == "" {
		comment = "blocked 后由用户转入终审"
	}
	return TransitionResult{
		NewStatus:    model.QuestStatusUserReview,
		FinalComment: comment,
		ClearBlocked: true,
	}, nil
}

// CancelFromBlocked 从阻塞状态取消。
// 只能从 blocked 状态调用。
func CancelFromBlocked(curStatus model.QuestStatus, comment string) (TransitionResult, error) {
	if curStatus != model.QuestStatusBlocked {
		return TransitionResult{}, fmt.Errorf("非法状态: 只能从 blocked 状态取消，当前状态: %s", curStatus)
	}
	if err := checkTransition(curStatus, model.QuestStatusCancelled); err != nil {
		return TransitionResult{}, err
	}
	return TransitionResult{
		NewStatus:    model.QuestStatusCancelled,
		FinalComment: comment,
		SetCompleted: true,
	}, nil
}

// RequestUserReviewRework 用户终审要求返工。
// 只能从 user_review 状态调用。
func RequestUserReviewRework(curStatus model.QuestStatus, comment string, currentReworkCount int) (TransitionResult, error) {
	if curStatus != model.QuestStatusUserReview {
		return TransitionResult{}, fmt.Errorf("非法状态: 只能从 user_review 状态请求返工，当前状态: %s", curStatus)
	}
	if err := checkTransition(curStatus, model.QuestStatusRunning); err != nil {
		return TransitionResult{}, err
	}
	return TransitionResult{
		NewStatus:   model.QuestStatusRunning,
		ReviewHints: comment,
		ReworkCount: currentReworkCount + 1,
	}, nil
}

// CompleteUserReview 用户终审完成（出结果）。
// verdict: pass → success, reject → failed
// 只能从 user_review 状态调用。
func CompleteUserReview(curStatus model.QuestStatus, verdict model.QuestVerdict, comment string) (TransitionResult, error) {
	if curStatus != model.QuestStatusUserReview {
		return TransitionResult{}, fmt.Errorf("非法状态: 只能从 user_review 状态完成终审，当前状态: %s", curStatus)
	}
	var next model.QuestStatus
	switch verdict {
	case model.VerdictPass:
		next = model.QuestStatusSuccess
	case model.VerdictReject:
		next = model.QuestStatusFailed
	default:
		return TransitionResult{}, fmt.Errorf("用户终审不支持 verdict: %s", verdict)
	}
	if err := checkTransition(curStatus, next); err != nil {
		return TransitionResult{}, err
	}
	return TransitionResult{
		NewStatus:    next,
		FinalVerdict: verdict,
		FinalComment: comment,
		SetCompleted: true,
	}, nil
}

// RequestReviewRework 法师评审要求返工（reviewing → running）。
// 只能从 reviewing 状态调用。
func RequestReviewRework(curStatus model.QuestStatus, comment string, currentReworkCount int) (TransitionResult, error) {
	if curStatus != model.QuestStatusReviewing {
		return TransitionResult{}, fmt.Errorf("非法状态: 只能从 reviewing 状态请求返工，当前状态: %s", curStatus)
	}
	if err := checkTransition(curStatus, model.QuestStatusRunning); err != nil {
		return TransitionResult{}, err
	}
	return TransitionResult{
		NewStatus:   model.QuestStatusRunning,
		ReviewHints: comment,
		ReworkCount: currentReworkCount + 1,
	}, nil
}

// CompleteReview 法师评审完成（reviewing → success/failed）。
// 只能从 reviewing 状态调用。
func CompleteReview(curStatus model.QuestStatus, verdict model.QuestVerdict, comment string) (TransitionResult, error) {
	if curStatus != model.QuestStatusReviewing {
		return TransitionResult{}, fmt.Errorf("非法状态: 只能从 reviewing 状态完成评审，当前状态: %s", curStatus)
	}
	var next model.QuestStatus
	switch verdict {
	case model.VerdictPass:
		next = model.QuestStatusSuccess
	case model.VerdictReject:
		next = model.QuestStatusFailed
	default:
		return TransitionResult{}, fmt.Errorf("评审不支持 verdict: %s", verdict)
	}
	if err := checkTransition(curStatus, next); err != nil {
		return TransitionResult{}, err
	}
	return TransitionResult{
		NewStatus:    next,
		FinalVerdict: verdict,
		FinalComment: comment,
		SetCompleted: true,
	}, nil
}

// CompleteExecute 执行阶段直接完成（running → success）。
// 用于跳过评审、且无需用户终审的场景（如 quick 模式无副作用 auto-complete）。
// 只能从 running 状态调用。
func CompleteExecute(curStatus model.QuestStatus, comment string) (TransitionResult, error) {
	if curStatus != model.QuestStatusRunning {
		return TransitionResult{}, fmt.Errorf("非法状态: 只能从 running 状态直接完成，当前状态: %s", curStatus)
	}
	return TransitionResult{
		NewStatus:    model.QuestStatusSuccess,
		FinalVerdict: model.VerdictPass,
		FinalComment: comment,
		SetCompleted: true,
	}, nil
}

// CancelQuest 取消委托（通用取消，调用方需确保当前状态允许取消）。
func CancelQuest(curStatus model.QuestStatus, reason string) (TransitionResult, error) {
	if err := checkTransition(curStatus, model.QuestStatusCancelled); err != nil {
		return TransitionResult{}, err
	}
	return TransitionResult{
		NewStatus:    model.QuestStatusCancelled,
		FinalComment: reason,
		SetCompleted: true,
	}, nil
}

// FailQuest 标记委托失败（通用失败，从任何非终态都可以失败）。
func FailQuest(curStatus model.QuestStatus, reason string) (TransitionResult, error) {
	if err := checkTransition(curStatus, model.QuestStatusFailed); err != nil {
		return TransitionResult{}, err
	}
	return TransitionResult{
		NewStatus:    model.QuestStatusFailed,
		FinalComment: reason,
		SetCompleted: true,
	}, nil
}

// BlockQuest 进入阻塞状态。
func BlockQuest(curStatus model.QuestStatus, reason string) (TransitionResult, error) {
	if err := checkTransition(curStatus, model.QuestStatusBlocked); err != nil {
		return TransitionResult{}, err
	}
	return TransitionResult{
		NewStatus:     model.QuestStatusBlocked,
		BlockedReason: reason,
	}, nil
}

// StartQuest 启动 quest（pending → running）。
// 只能从 pending 状态调用。
func StartQuest(curStatus model.QuestStatus) (TransitionResult, error) {
	if curStatus != model.QuestStatusPending {
		return TransitionResult{}, fmt.Errorf("非法状态: 只能从 pending 状态启动，当前状态: %s", curStatus)
	}
	if err := checkTransition(curStatus, model.QuestStatusRunning); err != nil {
		return TransitionResult{}, err
	}
	return TransitionResult{
		NewStatus:  model.QuestStatusRunning,
		SetStarted: true,
	}, nil
}

// StartReviewPhase 剑士阶段结束，进入评审阶段（running → reviewing）。
// 只能从 running 状态调用。
func StartReviewPhase(curStatus model.QuestStatus) (TransitionResult, error) {
	if curStatus != model.QuestStatusRunning {
		return TransitionResult{}, fmt.Errorf("非法状态: 只能从 running 状态进入评审，当前状态: %s", curStatus)
	}
	if err := checkTransition(curStatus, model.QuestStatusReviewing); err != nil {
		return TransitionResult{}, err
	}
	return TransitionResult{
		NewStatus: model.QuestStatusReviewing,
	}, nil
}

// MoveToUserReview 转入用户终审。
// 支持从 reviewing（正常评审路径）或 running（quick 模式跳过评审）转入。
func MoveToUserReview(curStatus model.QuestStatus) (TransitionResult, error) {
	if curStatus != model.QuestStatusReviewing && curStatus != model.QuestStatusRunning {
		return TransitionResult{}, fmt.Errorf("非法状态: 只能从 reviewing 或 running 状态转入用户终审，当前状态: %s", curStatus)
	}
	if err := checkTransition(curStatus, model.QuestStatusUserReview); err != nil {
		return TransitionResult{}, err
	}
	return TransitionResult{
		NewStatus: model.QuestStatusUserReview,
	}, nil
}

// ==================== 内部辅助 ====================

func checkTransition(cur, next model.QuestStatus) error {
	if !CanTransition(cur, next) {
		return fmt.Errorf("非法状态迁移: %s → %s", cur, next)
	}
	return nil
}
