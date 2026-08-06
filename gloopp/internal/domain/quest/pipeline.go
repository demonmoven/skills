package quest

import (
	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

// ==================== Pipeline 阶段管道 ====================
//
// Pipeline 是有序的 PhaseDef 列表，定义了一个 Quest 的完整执行流程。
// 管道是线性的，但支持返工循环（回到 phase 0）。
//
// 设计原则：
// - 线性管道 + 简单返工循环，不做全功能 DAG
// - 每个 quest 有自己的 pipeline 实例（可以有不同的模板）
// - 默认两阶段管道（执行 + 评审）保持与现有系统兼容

// Pipeline 阶段管道——有序的 PhaseDef 列表。
type Pipeline []PhaseDef

// DefaultPipeline 返回标准两阶段管道（执行 + 评审）。
//
// 这是 gloop 的经典模式：
//
//	Phase 0: Execute (warrior) — 剑士执行任务
//	Phase 1: Review  (mage)    — 法师评审结果
//
// 用户终审（user_review）目前作为 quest 级状态存在，
// 未来可以升级为管道中的第三个阶段。
func DefaultPipeline() Pipeline {
	return Pipeline{
		{
			Index:                   0,
			Name:                    "warrior",
			DisplayName:             "剑士执行",
			Role:                    PhaseRoleExecute,
			Class:                   model.ClassWarrior,
			Goal:                    "按照用户需求完成任务，交付可验证的产出",
			ReadOnly:                false,
			EndSignal:               "phase_checkpoint",
			EndTool:                 "phase_checkpoint",
			ReworkTo:                0,
			DefaultHints:            2,
			DefaultMaxTurns:         0, // 0 表示跟随全局配置
			PreCompletionProcessors: []string{"workspace_commit"},
			AfterCommitSubscribers:  []string{"diff_summary"},
		},
		{
			Index:                   1,
			Name:                    "mage_review",
			DisplayName:             "法师评审",
			Role:                    PhaseRoleReview,
			Class:                   model.ClassMage,
			Goal:                    "评审剑士的产出，给出通过/返工/拒绝结论",
			ReadOnly:                false,
			EndSignal:               "phase_checkpoint",
			EndTool:                 "phase_checkpoint",
			ReworkTo:                0,
			DefaultHints:            2,
			DefaultMaxTurns:         0, // 0 表示跟随全局配置
			PreCompletionProcessors: []string{"auto_apply"},
			AfterCommitSubscribers:  []string{"review_notify"},
		},
	}
}

// DesignPipeline 返回带设计阶段的四阶段管道。
func DesignPipeline() Pipeline {
	return Pipeline{
		{
			Index:                   0,
			Name:                    "warrior_design",
			DisplayName:             "剑士设计",
			Role:                    PhaseRoleExecute,
			Class:                   model.ClassWarrior,
			Goal:                    "理解需求并产出结构化设计方案",
			ReadOnly:                false,
			EndSignal:               "phase_checkpoint",
			EndTool:                 "phase_checkpoint",
			ReworkTo:                0,
			DefaultHints:            2,
			PreCompletionProcessors: []string{"workspace_commit"},
			AfterCommitSubscribers:  []string{"diff_summary"},
		},
		{
			Index:                   1,
			Name:                    "mage_design_review",
			DisplayName:             "法师方案评审",
			Role:                    PhaseRoleReview,
			Class:                   model.ClassMage,
			Goal:                    "评审设计方案的方向、边界、风险和可落地性",
			ReadOnly:                false,
			EndSignal:               "phase_checkpoint",
			EndTool:                 "phase_checkpoint",
			ReworkTo:                0,
			DefaultHints:            2,
			PreCompletionProcessors: []string{},
			AfterCommitSubscribers:  []string{"review_notify"},
		},
		{
			Index:                   2,
			Name:                    "warrior_execute",
			DisplayName:             "剑士执行",
			Role:                    PhaseRoleExecute,
			Class:                   model.ClassWarrior,
			Goal:                    "按设计方案实现并交付可验证产出",
			ReadOnly:                false,
			EndSignal:               "phase_checkpoint",
			EndTool:                 "phase_checkpoint",
			ReworkTo:                2,
			DefaultHints:            2,
			PreCompletionProcessors: []string{"workspace_commit"},
			AfterCommitSubscribers:  []string{"diff_summary"},
		},
		{
			Index:                   3,
			Name:                    "mage_implementation_review",
			DisplayName:             "法师实现评审",
			Role:                    PhaseRoleReview,
			Class:                   model.ClassMage,
			Goal:                    "评审实现质量，并检查实现是否符合设计方案",
			ReadOnly:                false,
			EndSignal:               "phase_checkpoint",
			EndTool:                 "phase_checkpoint",
			ReworkTo:                2,
			DefaultHints:            2,
			PreCompletionProcessors: []string{"auto_apply"},
			AfterCommitSubscribers:  []string{"review_notify"},
		},
	}
}

// Count 返回管道中的阶段数量。
func (p Pipeline) Count() int {
	return len(p)
}

// At 返回指定索引的阶段定义。
// 如果索引越界，返回 nil。
func (p Pipeline) At(idx int) *PhaseDef {
	if idx < 0 || idx >= len(p) {
		return nil
	}
	return &p[idx]
}

// FindByName 按名称查找阶段定义。
// 找不到返回 -1 和 nil。
func (p Pipeline) FindByName(name string) (int, *PhaseDef) {
	for i := range p {
		if p[i].Name == name {
			return i, &p[i]
		}
	}
	return -1, nil
}

// NextPhaseIdx 返回下一阶段的索引。
// 如果已经是最后一个阶段，返回 -1。
func (p Pipeline) NextPhaseIdx(currentIdx int) int {
	next := currentIdx + 1
	if next >= len(p) {
		return -1
	}
	return next
}

// PrevPhaseIdx 返回上一阶段的索引。
// 如果已经是第一个阶段，返回 -1。
func (p Pipeline) PrevPhaseIdx(currentIdx int) int {
	prev := currentIdx - 1
	if prev < 0 {
		return -1
	}
	return prev
}

// IsFirstPhase 返回是否是第一个阶段。
func (p Pipeline) IsFirstPhase(idx int) bool {
	return idx == 0
}

// IsLastPhase 返回是否是最后一个阶段。
func (p Pipeline) IsLastPhase(idx int) bool {
	return idx == len(p)-1
}

// ReworkTargetIdx 返回指定阶段的返工目标阶段索引。
// 旧调用不传 idx 时保持默认两阶段行为：返工回到 phase 0。
func (p Pipeline) ReworkTargetIdx(idx ...int) int {
	targetPhase := len(p) - 1
	if len(idx) > 0 {
		targetPhase = idx[0]
	}
	if targetPhase >= 0 && targetPhase < len(p) {
		return p[targetPhase].ReworkTo
	}
	return 0
}

// PhaseIdxForStatus 根据 quest 状态推断当前阶段索引。
//
// 这是为了兼容现有状态机设计的映射：
//
//	running    → phase 0 (执行)
//	reviewing  → phase 1 (评审)
//	其他状态   → -1 (不在任何 agent 阶段)
func (p Pipeline) PhaseIdxForStatus(status model.QuestStatus) int {
	switch status {
	case model.QuestStatusRunning:
		if len(p) > 0 {
			return 0
		}
	case model.QuestStatusReviewing:
		if len(p) > 1 {
			return 1
		}
	}
	return -1
}
