package quest

import (
	"strings"

	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

// Quest 是领域层的 Quest 聚合根。
//
// 注意：这是 Phase 1 的过渡设计。
// 当前阶段，fsstore.QuestMeta 仍然是持久化的主要载体，
// 领域层的 Quest 只封装核心业务规则（状态机、预算校验、失败归因等）。
// 后续 Phase 2/3 会逐步把更多字段和行为迁移到这里。

// QuestID 是 Quest 的唯一标识符类型。
type QuestID string

// Quest 领域对象（Phase 1: 轻量级，只包含状态机相关核心字段）。
//
// 设计原则：
// - 领域对象不依赖存储层，纯内存操作
// - 所有状态变更通过方法进行，保证业务规则一致性
// - 字段是值语义，修改通过返回新状态或在方法内修改
type Quest struct {
	ID        string
	Status    model.QuestStatus
	Type      model.QuestType
	Intensity model.QuestIntensity

	// 过程字段
	ReworkCount       int
	MaxRework         int
	ResumeCount       int
	ReviewHints       string
	BlockedReason     string
	BlockedReasonCode string
	BlockedCategory   string

	// 结果字段
	FinalVerdict model.QuestVerdict
	FinalComment string

	// 时间戳（毫秒）
	StartedAtMs   int64
	CompletedAtMs int64
}

// NewQuest 创建一个新的 Quest（pending 状态）。
func NewQuest(id string, questType model.QuestType, maxRework int) *Quest {
	if questType == "" {
		questType = model.QuestTypeExecute
	}
	return &Quest{
		ID:        id,
		Status:    model.QuestStatusPending,
		Type:      questType,
		MaxRework: maxRework,
	}
}

// IsTerminal 返回 quest 是否已处于终态。
func (q *Quest) IsTerminal() bool {
	return IsTerminal(q.Status)
}

// CanTransitionTo 检查能否迁移到目标状态。
func (q *Quest) CanTransitionTo(next model.QuestStatus) bool {
	return CanTransition(q.Status, next)
}

// ApplyTransitionResult 把状态迁移结果应用到 Quest 对象。
// 这是领域层和存储层之间的桥梁：领域层计算变更，调用方负责持久化。
func (q *Quest) ApplyTransitionResult(r TransitionResult) {
	q.Status = r.NewStatus
	if r.FinalVerdict != "" {
		q.FinalVerdict = r.FinalVerdict
	}
	if r.FinalComment != "" {
		q.FinalComment = r.FinalComment
	}
	if r.BlockedReason != "" {
		q.BlockedReason = r.BlockedReason
	}
	if r.BlockedReasonCode != "" {
		q.BlockedReasonCode = r.BlockedReasonCode
	}
	if r.BlockedCategory != "" {
		q.BlockedCategory = r.BlockedCategory
	}
	if r.ReviewHints != "" {
		q.ReviewHints = r.ReviewHints
	}
	if r.ReworkCount > 0 {
		q.ReworkCount = r.ReworkCount
	}
	if r.ResumeCount > 0 {
		q.ResumeCount = r.ResumeCount
	}
	if r.ClearBlocked {
		q.BlockedReason = ""
		q.BlockedReasonCode = ""
		q.BlockedCategory = ""
	}
	if r.SetCompleted && q.CompletedAtMs == 0 {
		q.CompletedAtMs = model.NowMs()
	}
}

// CanRework 检查是否还能返工（未达最大返工次数）。
func (q *Quest) CanRework() bool {
	return q.ReworkCount < q.MaxRework
}

// Source 解析 quest 来源。
// createdBy 格式: "user" 或 "automation:<id>"
func QuestSource(createdBy string) model.QuestSource {
	if strings.HasPrefix(createdBy, model.QuestSourcePrefix) {
		return model.SourceAutomation
	}
	return model.SourceUser
}

// IsAutomationQuest 检查是否为 automation 发起的 quest。
func IsAutomationQuest(createdBy string) bool {
	return QuestSource(createdBy) == model.SourceAutomation
}
