package quest

import (
	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

// ==================== Phase 领域抽象 ====================
//
// Phase 是 Quest 管道中的一个阶段。
// 每个阶段有明确的角色（执行/评审/人工）、目标、预算和产出。
//
// 设计原则：
// - 数据驱动，非接口多态：所有 Agent 阶段共享同一执行引擎
// - 差异通过 PhaseDef 配置注入，而不是通过多态方法
// - Pipeline 是有序阶段列表，返工回到 phase 0
//
// 两种核心概念：
// - PhaseDef: 阶段定义（静态配置，管道的一部分）
// - PhaseRun: 阶段运行时状态（动态，每个 quest 有自己的 phase 运行记录）

// PhaseRole 阶段角色分类。
type PhaseRole string

const (
	PhaseRoleExecute PhaseRole = "execute" // 执行类（生产者，如剑士）
	PhaseRoleReview  PhaseRole = "review"  // 评审类（审核者，如法师）
	PhaseRoleHuman   PhaseRole = "human"   // 人工介入类（如用户终审）
)

// PhaseDef 阶段定义——管道中的一个节点。
//
// 这是静态配置，描述"这个阶段是什么"，不包含运行时状态。
// 同一个 PhaseDef 可以被多个 quest 复用。
type PhaseDef struct {
	Index        int                   // 在管道中的索引
	Name         string                // 阶段标识：warrior / mage_review / user_review
	DisplayName  string                // 展示名：剑士执行 / 法师评审 / 用户终审
	Role         PhaseRole             // 角色分类
	Class        model.AdventurerClass // 对应冒险者职业（human 阶段为空）
	Goal         string                // 阶段目标描述
	ReadOnly     bool                  // 是否只读阶段
	EndSignal    string                // 结束阶段的信号标识（如 phase_checkpoint / review_quest）
	EndTool      string                // Deprecated: 旧字段名，兼容 EndSignal
	ReworkTo     int                   // 返工目标阶段索引，-1 表示不支持返工
	DefaultHints int                   // 默认 hint 次数
	AllowedTools []string              // 允许工具列表；空表示跟随 adventurer 默认工具
	AgentID      string                // 可选：直接绑定的 agent profile

	ExitCriteria            PhaseExitCriteria
	PreCompletionProcessors []string
	AfterCommitSubscribers  []string

	// 预算默认值（可被 quest 级别覆盖）
	DefaultMaxTurns      int
	DefaultMaxDurationMs int64
}

type PhaseExitCriteria struct {
	MinScore int
}

func (p PhaseDef) EffectiveEndSignal() string {
	if p.EndSignal != "" {
		return p.EndSignal
	}
	return p.EndTool
}

// IsAgentPhase 返回该阶段是否由 Agent 执行（非人工）。
func (p PhaseDef) IsAgentPhase() bool {
	return p.Role == PhaseRoleExecute || p.Role == PhaseRoleReview
}

// PhaseStatus 阶段执行状态（复用 model 层定义）。
type PhaseStatus = model.PhaseStatus

const (
	PhasePending = model.PhasePending
	PhaseRunning = model.PhaseRunning
	PhaseDone    = model.PhaseDone
	PhaseFailed  = model.PhaseFailed
)

// PhaseRun 阶段运行时状态。
//
// 对应存储层的 PhaseTask，但领域层不依赖存储实现。
// 每个 quest 实例有一组 PhaseRun，记录各阶段的执行进度。
type PhaseRun struct {
	PhaseIdx     int
	AdventurerID string
	AgentID      string
	SessionID    string
	Status       PhaseStatus
	Turns        int
	ReworkCount  int
	StartedAtMs  int64
	EndedAtMs    int64
}

// IsTerminal 返回阶段是否已结束（成功或失败）。
func (p *PhaseRun) IsTerminal() bool {
	return p.Status == PhaseDone || p.Status == PhaseFailed
}

// Start 标记阶段开始。
func (p *PhaseRun) Start(nowMs int64) {
	p.Status = PhaseRunning
	if p.StartedAtMs == 0 {
		p.StartedAtMs = nowMs
	}
}

// Complete 标记阶段成功完成。
func (p *PhaseRun) Complete(nowMs int64) {
	p.Status = PhaseDone
	p.EndedAtMs = nowMs
}

// Fail 标记阶段失败。
func (p *PhaseRun) Fail(nowMs int64) {
	p.Status = PhaseFailed
	p.EndedAtMs = nowMs
}

// Reset 重置阶段（用于返工）。
func (p *PhaseRun) Reset() {
	p.Status = PhasePending
	p.SessionID = ""
	p.Turns = 0
	p.StartedAtMs = 0
	p.EndedAtMs = 0
	p.ReworkCount++
}
