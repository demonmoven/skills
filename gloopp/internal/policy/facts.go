package policy

import (
	"strings"

	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

type FactBuilder struct {
	root *fsstore.Root
}

func NewFactBuilder(root *fsstore.Root) *FactBuilder {
	return &FactBuilder{root: root}
}

func (b *FactBuilder) BuildReviewFacts(q *fsstore.QuestMeta, reviewVerdict string, reviewScore int) InputFacts {
	if q == nil {
		return InputFacts{}
	}
	effectType := b.EffectType(q)
	autoID := q.AutomationID()
	autoTags := []string(nil)
	allowL2 := false
	isOfficial := false
	if b != nil && b.root != nil && autoID != "" {
		if cfg, err := b.root.GetAutomation(autoID); err == nil && cfg != nil {
			autoTags = append(autoTags, cfg.Tags...)
			allowL2 = cfg.AllowL2
			isOfficial = cfg.IsOfficial()
		}
	}
	hasExternal, _ := outputKinds(q)
	return InputFacts{
		QuestID:              q.ID,
		QuestType:            string(q.Type),
		Intensity:            string(q.Intensity),
		Source:               string(q.Source()),
		WorkspacePath:        q.WorkspacePath,
		EffectType:           effectType,
		HasWorkspaceDiff:     effectType == EffectWorkspaceDiff || q.DiffChangedFiles > 0 && !hasOnlyExternalOutputsLocal(q),
		SideEffectLevel:      sideEffectLevel(effectType, allowL2),
		HasExternalOutput:    hasExternal,
		AllowL2:              allowL2,
		CurrentPhaseIdx:      q.CurrentPhaseIdx(),
		CurrentPhaseType:     phaseType(q.CurrentPhaseIdx()),
		PhaseReadOnly:        false,
		PhaseEndSignal:       phaseEndSignal(q.CurrentPhaseIdx()),
		AutomationID:         autoID,
		AutomationTags:       autoTags,
		IsOfficialAutomation: isOfficial,
		ReviewVerdict:        reviewVerdict,
		ReviewScore:          reviewScore,
		ReworkCount:          q.ReworkCount,
		BlockedReason:        q.BlockedReason,
		BlockedReasonCode:    q.BlockedReasonCode,
		BlockedCategory:      q.BlockedCategory,
	}
}

func (b *FactBuilder) BuildRecoveryFacts(q *fsstore.QuestMeta, recoveryCount int) InputFacts {
	facts := b.BuildReviewFacts(q, "", 0)
	facts.RecoveryCount = recoveryCount
	return facts
}

// EffectType derives the quest's effect type from outputs, diff, and automation tags.
// Priority: workspace_diff > context_store > external_side_effect > none.
func (b *FactBuilder) EffectType(q *fsstore.QuestMeta) string {
	if q == nil {
		return "unknown"
	}
	hasExternal, hasFile := outputKinds(q)
	if hasFile || q.DiffChangedFiles > 0 {
		if !hasExternal || hasFile {
			return EffectWorkspaceDiff
		}
	}
	if autoID := q.AutomationID(); autoID != "" {
		if autoID == "auto_context_refresh" {
			return EffectContextStore
		}
		if b != nil && b.root != nil {
			if cfg, err := b.root.GetAutomation(autoID); err == nil && cfg != nil {
				if cfg.HasTag("context") {
					return EffectContextStore
				}
				if cfg.AllowL2 {
					return EffectExternalSideEffect
				}
			}
		}
	}
	if hasExternal {
		return EffectExternalSideEffect
	}
	return EffectNone
}

func outputKinds(q *fsstore.QuestMeta) (hasExternal bool, hasFile bool) {
	for _, out := range q.Outputs {
		if strings.HasPrefix(out.StoragePath, "http://") || strings.HasPrefix(out.StoragePath, "https://") {
			hasExternal = true
		} else if out.StoragePath != "" {
			hasFile = true
		}
	}
	return hasExternal, hasFile
}

func hasOnlyExternalOutputsLocal(q *fsstore.QuestMeta) bool {
	if q == nil || len(q.Outputs) == 0 {
		return false
	}
	hasExternal, hasFile := outputKinds(q)
	return hasExternal && !hasFile
}

func sideEffectLevel(effectType string, allowL2 bool) string {
	if allowL2 {
		return "high"
	}
	switch effectType {
	case EffectWorkspaceDiff, EffectExternalSideEffect:
		return "medium"
	case EffectContextStore:
		return "low"
	default:
		return "none"
	}
}

func phaseType(idx int) string {
	if idx <= 0 {
		return "execute"
	}
	return "review"
}

func phaseEndSignal(_ int) string {
	return "phase_checkpoint"
}

var _ = model.SourceUser
