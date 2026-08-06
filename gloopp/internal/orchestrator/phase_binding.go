package orchestrator

import (
	"strings"

	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

type PhaseBindingSource string

const (
	PhaseBindingSourcePhaseAdventurer       PhaseBindingSource = "phase_adventurer"
	PhaseBindingSourcePhaseAgent            PhaseBindingSource = "phase_agent"
	PhaseBindingSourceQuestAgent            PhaseBindingSource = "quest_agent"
	PhaseBindingSourceConfigDefaultAgent    PhaseBindingSource = "config_default_agent"
	PhaseBindingSourceLegacyQuestAdventurer PhaseBindingSource = "legacy_quest_adventurer"
	PhaseBindingSourceAutoSelected          PhaseBindingSource = "auto_selected_agent"
	PhaseBindingSourcePickedAdventurer      PhaseBindingSource = "picked_adventurer"
	PhaseBindingSourceNone                  PhaseBindingSource = "none"
)

type PhaseBindingResolution struct {
	AgentID      string
	AdventurerID string
	Source       PhaseBindingSource
	Warnings     []string
}

func resolvePhaseBinding(
	phase fsstore.PhaseTask,
	q *fsstore.QuestMeta,
	cfg *fsstore.GlobalConfig,
	adventurers []*fsstore.AdventurerFile,
	agents []*fsstore.AgentConfig,
) PhaseBindingResolution {
	if strings.TrimSpace(phase.AdventurerID) != "" {
		return PhaseBindingResolution{
			AdventurerID: phase.AdventurerID,
			Source:       PhaseBindingSourcePhaseAdventurer,
		}
	}
	if strings.TrimSpace(phase.AgentID) != "" {
		return phaseAgentResolution(phase.AgentID, PhaseBindingSourcePhaseAgent, agents)
	}
	if agentID := questAgentIDForPhase(q, phase); strings.TrimSpace(agentID) != "" {
		return phaseAgentResolution(agentID, PhaseBindingSourceQuestAgent, agents)
	}
	if agentID := configDefaultAgentIDForPhase(cfg, phase); strings.TrimSpace(agentID) != "" {
		return phaseAgentResolution(agentID, PhaseBindingSourceConfigDefaultAgent, agents)
	}
	if advID := legacyAdventurerIDForPhase(q, phase); strings.TrimSpace(advID) != "" {
		return PhaseBindingResolution{
			AdventurerID: advID,
			Source:       PhaseBindingSourceLegacyQuestAdventurer,
		}
	}
	if picked := pickActiveAdventurerForPhase(adventurers, phase); picked != nil {
		return PhaseBindingResolution{
			AdventurerID: picked.ID,
			Source:       PhaseBindingSourcePickedAdventurer,
		}
	}
	return PhaseBindingResolution{Source: PhaseBindingSourceNone}
}

func phaseAgentResolution(agentID string, source PhaseBindingSource, agents []*fsstore.AgentConfig) PhaseBindingResolution {
	out := PhaseBindingResolution{AgentID: agentID, Source: source}
	if !agentExists(agentID, agents) {
		out.Warnings = append(out.Warnings, "agent_not_found:"+agentID)
	}
	return out
}

func agentExists(agentID string, agents []*fsstore.AgentConfig) bool {
	if strings.TrimSpace(agentID) == "" {
		return false
	}
	for _, agent := range agents {
		if agent != nil && agent.Name == agentID {
			return true
		}
	}
	return false
}

func questAgentIDForPhase(q *fsstore.QuestMeta, phase fsstore.PhaseTask) string {
	if q == nil {
		return ""
	}
	if phaseIsReview(phase) {
		return q.ReviewAgentID
	}
	return q.ExecuteAgentID
}

func configDefaultAgentIDForPhase(cfg *fsstore.GlobalConfig, phase fsstore.PhaseTask) string {
	if cfg == nil {
		return ""
	}
	if phaseIsReview(phase) {
		return cfg.DefaultReviewAgentID
	}
	return cfg.DefaultExecuteAgentID
}

func legacyAdventurerIDForPhase(q *fsstore.QuestMeta, phase fsstore.PhaseTask) string {
	if q == nil {
		return ""
	}
	if phaseIsReview(phase) {
		return q.MageID
	}
	return q.WarriorID
}

func pickActiveAdventurerForPhase(adventurers []*fsstore.AdventurerFile, phase fsstore.PhaseTask) *fsstore.AdventurerFile {
	wantClass := phaseClassForBinding(phase)
	var pick *fsstore.AdventurerFile
	for _, adventurer := range adventurers {
		if adventurer == nil || adventurer.Status != model.AdventurerActive || adventurer.Class != wantClass {
			continue
		}
		if pick == nil || fsstore.BetterAdventurer(adventurer, pick) {
			pick = adventurer
		}
	}
	return pick
}

func phaseIsReview(phase fsstore.PhaseTask) bool {
	role := strings.ToLower(strings.TrimSpace(phase.Role))
	name := strings.ToLower(strings.TrimSpace(phase.Name))
	return role == "mage" || role == "review" || phase.Class == model.ClassMage || strings.Contains(name, "review")
}

func phaseClassForBinding(phase fsstore.PhaseTask) model.AdventurerClass {
	if phase.Class != "" {
		return phase.Class
	}
	if phaseIsReview(phase) {
		return model.ClassMage
	}
	return model.ClassWarrior
}
