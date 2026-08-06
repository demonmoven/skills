package orchestrator

import (
	"strings"
	"testing"

	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

func TestResolvePhaseBindingPriority(t *testing.T) {
	agents := []*fsstore.AgentConfig{{Name: "phase_agent"}, {Name: "quest_agent"}, {Name: "default_agent"}}
	adventurers := []*fsstore.AdventurerFile{{
		ID:     "picked_warrior",
		Class:  model.ClassWarrior,
		Status: model.AdventurerActive,
	}}
	cfg := &fsstore.GlobalConfig{DefaultExecuteAgentID: "default_agent"}
	q := &fsstore.QuestMeta{ExecuteAgentID: "quest_agent", WarriorID: "legacy_warrior"}
	phase := fsstore.PhaseTask{
		Role:         "warrior",
		Class:        model.ClassWarrior,
		AdventurerID: "phase_adventurer",
		AgentID:      "phase_agent",
	}

	got := resolvePhaseBinding(phase, q, cfg, adventurers, agents)
	if got.Source != PhaseBindingSourcePhaseAdventurer || got.AdventurerID != "phase_adventurer" || got.AgentID != "" {
		t.Fatalf("phase adventurer should win, got %+v", got)
	}

	phase.AdventurerID = ""
	got = resolvePhaseBinding(phase, q, cfg, adventurers, agents)
	if got.Source != PhaseBindingSourcePhaseAgent || got.AgentID != "phase_agent" {
		t.Fatalf("phase agent should win, got %+v", got)
	}

	phase.AgentID = ""
	got = resolvePhaseBinding(phase, q, cfg, adventurers, agents)
	if got.Source != PhaseBindingSourceQuestAgent || got.AgentID != "quest_agent" {
		t.Fatalf("quest agent should win, got %+v", got)
	}

	q.ExecuteAgentID = ""
	got = resolvePhaseBinding(phase, q, cfg, adventurers, agents)
	if got.Source != PhaseBindingSourceConfigDefaultAgent || got.AgentID != "default_agent" {
		t.Fatalf("config default agent should win, got %+v", got)
	}

	cfg.DefaultExecuteAgentID = ""
	got = resolvePhaseBinding(phase, q, cfg, adventurers, agents)
	if got.Source != PhaseBindingSourceLegacyQuestAdventurer || got.AdventurerID != "legacy_warrior" {
		t.Fatalf("legacy adventurer should win, got %+v", got)
	}

	q.WarriorID = ""
	got = resolvePhaseBinding(phase, q, cfg, adventurers, agents)
	if got.Source != PhaseBindingSourcePickedAdventurer || got.AdventurerID != "picked_warrior" {
		t.Fatalf("picked adventurer should win, got %+v", got)
	}
}

func TestResolvePhaseBindingReviewDefaults(t *testing.T) {
	agents := []*fsstore.AgentConfig{{Name: "review_agent"}}
	cfg := &fsstore.GlobalConfig{DefaultExecuteAgentID: "execute_agent", DefaultReviewAgentID: "review_agent"}
	q := &fsstore.QuestMeta{ExecuteAgentID: "quest_execute", ReviewAgentID: "quest_review"}

	got := resolvePhaseBinding(fsstore.PhaseTask{Role: "mage", Class: model.ClassMage}, q, cfg, nil, agents)
	if got.Source != PhaseBindingSourceQuestAgent || got.AgentID != "quest_review" {
		t.Fatalf("review phase should use review agent, got %+v", got)
	}

	q.ReviewAgentID = ""
	got = resolvePhaseBinding(fsstore.PhaseTask{Name: "security_review"}, q, cfg, nil, agents)
	if got.Source != PhaseBindingSourceConfigDefaultAgent || got.AgentID != "review_agent" {
		t.Fatalf("review phase should use default review agent, got %+v", got)
	}
}

func TestResolvePhaseBindingWarnsForMissingAgent(t *testing.T) {
	got := resolvePhaseBinding(
		fsstore.PhaseTask{Role: "warrior", Class: model.ClassWarrior, AgentID: "missing_agent"},
		nil,
		nil,
		nil,
		[]*fsstore.AgentConfig{{Name: "other_agent"}},
	)
	if got.Source != PhaseBindingSourcePhaseAgent || got.AgentID != "missing_agent" {
		t.Fatalf("explicit missing agent should still resolve identity, got %+v", got)
	}
	if len(got.Warnings) != 1 || got.Warnings[0] != "agent_not_found:missing_agent" {
		t.Fatalf("warnings = %#v", got.Warnings)
	}
}

func TestBuildSystemForPhaseActorUsesAgentPersona(t *testing.T) {
	actor := &phaseActor{
		Agent:     &fsstore.AgentConfig{Name: "direct_agent"},
		Role:      "execute",
		PhaseName: "warrior",
	}
	got, err := buildSystemForPhaseActor(actor, nil, nil)
	if err != nil {
		t.Fatalf("BuildSystemForPhaseActor: %v", err)
	}
	for _, want := range []string{"# 冒险者：direct_agent", "剑士", "gloop phase done"} {
		if !strings.Contains(got, want) {
			t.Fatalf("system prompt missing %q:\n%s", want, got)
		}
	}
}

func TestBuildSystemForPhaseActorUsesPhaseRoleForBehaviorAndAdventurerForPersona(t *testing.T) {
	actor := &phaseActor{
		Agent: &fsstore.AgentConfig{Name: "review_agent"},
		Adventurer: &fsstore.AdventurerFile{
			Name:         "warrior-persona",
			Class:        model.ClassWarrior,
			Level:        7,
			CustomPrompt: "说话像战士",
		},
		Role:      "review",
		PhaseName: "mage_review",
	}
	got, err := buildSystemForPhaseActor(actor, nil, nil)
	if err != nil {
		t.Fatalf("BuildSystemForPhaseActor: %v", err)
	}
	for _, want := range []string{"# 冒险者：warrior-persona", "等级 7", "可以自由验证", "说话像战士"} {
		if !strings.Contains(got, want) {
			t.Fatalf("system prompt missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "阶段完成时通过 `gloop phase done` 提交交付") {
		t.Fatalf("review phase must not use warrior execute permissions:\n%s", got)
	}
}

func TestReviewAgentIDForPhaseUsesPhaseBindingFallback(t *testing.T) {
	q := &fsstore.QuestMeta{
		Phases: []fsstore.PhaseTask{
			{PhaseIdx: 0, Role: "warrior", Class: model.ClassWarrior},
			{PhaseIdx: 1, Role: "mage", Class: model.ClassMage, AgentID: "phase_review_agent"},
		},
	}
	if got := reviewAgentIDForPhase(q, 1); got != "phase_review_agent" {
		t.Fatalf("reviewAgentIDForPhase = %q, want phase_review_agent", got)
	}
	q.ReviewAgentID = "quest_review_agent"
	if got := reviewAgentIDForPhase(q, 1); got != "quest_review_agent" {
		t.Fatalf("quest review agent should win, got %q", got)
	}
}
