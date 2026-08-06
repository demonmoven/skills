package orchestrator

import (
	"fmt"
	"sort"
	"strings"

	"code.byted.org/lihuanyu.0w0/gloop/internal/domain/quest"
	"code.byted.org/lihuanyu.0w0/gloop/internal/executor"
	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
	"code.byted.org/lihuanyu.0w0/gloop/internal/prompt"
)

type phaseActor struct {
	Agent      *fsstore.AgentConfig
	Adventurer *fsstore.AdventurerFile
	Binding    PhaseBindingResolution
	Role       quest.PhaseRole
	PhaseName  string
}

func (e *Engine) resolvePhaseActorForQuestPhase(q *fsstore.QuestMeta, phaseIdx int, fallbackAdv *fsstore.AdventurerFile) (*phaseActor, error) {
	if q == nil {
		return nil, fmt.Errorf("quest is nil")
	}
	phase := phaseTaskForActor(q, phaseIdx)
	adventurers, err := e.root.ListAdventurers()
	if err != nil {
		return nil, err
	}
	agents, err := e.root.ListAgents()
	if err != nil {
		return nil, err
	}
	binding := resolvePhaseBinding(phase, q, e.cfg, adventurers, agents)
	if binding.Source == PhaseBindingSourcePickedAdventurer || binding.Source == PhaseBindingSourceNone {
		if agent := e.pickRegisteredAgent(agents); agent != nil {
			binding = PhaseBindingResolution{
				AgentID: agent.Name,
				Source:  PhaseBindingSourceAutoSelected,
			}
		}
	}
	actor := &phaseActor{
		Binding:   binding,
		Role:      phaseRoleForActor(phase, phaseIdx),
		PhaseName: phaseNameForActor(phase, phaseIdx),
	}
	if binding.AgentID != "" {
		actor.Agent = findAgentConfig(agents, binding.AgentID)
		if actor.Agent == nil {
			return nil, fmt.Errorf("agent %s 不存在", binding.AgentID)
		}
	}
	if binding.AdventurerID != "" {
		adv, err := e.root.GetAdventurer(binding.AdventurerID)
		if err != nil {
			return nil, err
		}
		actor.Adventurer = adv
	}
	if actor.Adventurer == nil && fallbackAdv != nil {
		actor.Adventurer = fallbackAdv
	}
	if actor.Agent == nil && actor.Adventurer != nil {
		actor.Agent = findAgentConfig(agents, actor.Adventurer.Agent)
	}
	return actor, nil
}

func (e *Engine) pickRegisteredAgent(agents []*fsstore.AgentConfig) *fsstore.AgentConfig {
	registered := make([]*fsstore.AgentConfig, 0, len(agents))
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, agent := range agents {
		if agent == nil || !agent.Enabled {
			continue
		}
		if _, ok := e.execs[agent.Name]; !ok {
			continue
		}
		registered = append(registered, agent)
	}
	sort.SliceStable(registered, func(i, j int) bool {
		if registered[i].Official != registered[j].Official {
			return registered[i].Official
		}
		return registered[i].Name < registered[j].Name
	})
	if len(registered) == 0 {
		return nil
	}
	return registered[0]
}

func (e *Engine) executorForPhaseActor(actor *phaseActor) (executor.Executor, string, error) {
	if actor == nil {
		return nil, "", fmt.Errorf("phase actor is nil")
	}
	if actor.Agent != nil && actor.Binding.AgentID != "" {
		ex, modelName, err := e.getExecutorForAgent(actor.Agent.Name)
		if err != nil {
			return nil, "", err
		}
		if actor.Adventurer != nil && strings.TrimSpace(actor.Adventurer.Model) != "" {
			modelName = actor.Adventurer.Model
		}
		return ex, modelName, nil
	}
	if actor.Adventurer != nil {
		return e.getExecutor(actor.Adventurer)
	}
	return nil, "", fmt.Errorf("phase actor 没有 agent 或 adventurer")
}

func phaseTaskForActor(q *fsstore.QuestMeta, phaseIdx int) fsstore.PhaseTask {
	if q != nil {
		q.EnsurePhases()
		if phaseIdx >= 0 && phaseIdx < len(q.Phases) {
			return q.Phases[phaseIdx]
		}
	}
	def := phaseDefForQuest(q, phaseIdx)
	return fsstore.PhaseTask{
		PhaseIdx: phaseIdx,
		Role:     phaseRoleFromDomainForActor(def.Role),
		Class:    def.Class,
		Name:     def.Name,
		AgentID:  def.AgentID,
	}
}

func findAgentConfig(agents []*fsstore.AgentConfig, id string) *fsstore.AgentConfig {
	for _, agent := range agents {
		if agent != nil && agent.Name == id {
			return agent
		}
	}
	return nil
}

func phaseRoleForActor(phase fsstore.PhaseTask, phaseIdx int) quest.PhaseRole {
	role := strings.ToLower(strings.TrimSpace(phase.Role))
	switch role {
	case string(quest.PhaseRoleExecute), "warrior":
		return quest.PhaseRoleExecute
	case string(quest.PhaseRoleReview), "mage":
		return quest.PhaseRoleReview
	case string(quest.PhaseRoleHuman):
		return quest.PhaseRoleHuman
	}
	if phase.Class == model.ClassMage || strings.Contains(strings.ToLower(phase.Name), "review") {
		return quest.PhaseRoleReview
	}
	if phaseIdx == 0 {
		return quest.PhaseRoleExecute
	}
	return quest.PhaseRoleReview
}

func phaseRoleFromDomainForActor(role quest.PhaseRole) string {
	switch role {
	case quest.PhaseRoleExecute:
		return "warrior"
	case quest.PhaseRoleReview:
		return "mage"
	default:
		return string(role)
	}
}

func phaseNameForActor(phase fsstore.PhaseTask, phaseIdx int) string {
	if strings.TrimSpace(phase.Name) != "" {
		return phase.Name
	}
	return phaseNameForIdx(phaseIdx)
}

func buildSystemForPhaseActor(actor *phaseActor, skills prompt.SkillRegistry, templates *prompt.TemplateStore) (string, error) {
	if actor == nil {
		return "", fmt.Errorf("phase actor is nil")
	}
	return prompt.BuildSystemForPersona(personaForPhaseActor(actor), skills, templates)
}

func personaForPhaseActor(actor *phaseActor) prompt.SystemPromptPersona {
	persona := prompt.SystemPromptPersona{Class: phaseClassForRole(quest.PhaseRoleExecute)}
	if actor == nil {
		return persona
	}
	persona.Class = phaseClassForRole(actor.Role)
	name := "agent-direct"
	if actor.Agent != nil {
		name = actor.Agent.Name
	} else if actor.PhaseName != "" {
		name = actor.PhaseName
	}
	persona.Name = name
	if actor.Adventurer != nil {
		persona.Name = actor.Adventurer.Name
		persona.Level = actor.Adventurer.Level
		persona.CustomPrompt = actor.Adventurer.CustomPrompt
	}
	return persona
}

func phaseClassForRole(role quest.PhaseRole) model.AdventurerClass {
	if role == quest.PhaseRoleReview {
		return model.ClassMage
	}
	return model.ClassWarrior
}
