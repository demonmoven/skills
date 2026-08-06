package fsstore

import (
	"fmt"
	"path/filepath"
)

type LoopStateSpine struct {
	SchemaVersion      string             `json:"schema_version"`
	QuestID            string             `json:"quest_id"`
	CurrentGoal        string             `json:"current_goal,omitempty"`
	CurrentPhase       string             `json:"current_phase,omitempty"`
	Attempts           []LoopStateAttempt `json:"attempts,omitempty"`
	ConfirmedFacts     []string           `json:"confirmed_facts,omitempty"`
	FailedPaths        []string           `json:"failed_paths,omitempty"`
	OpenBlockers       []string           `json:"open_blockers,omitempty"`
	NextExpectedAction string             `json:"next_expected_action,omitempty"`
	HumanExceptions    []string           `json:"human_exceptions,omitempty"`
	UpdatedAtMs        int64              `json:"updated_at_ms"`
}

type LoopStateAttempt struct {
	RunID     string `json:"run_id"`
	Summary   string `json:"summary,omitempty"`
	Result    string `json:"result,omitempty"`
	Reason    string `json:"reason,omitempty"`
	CreatedAt int64  `json:"created_at_ms"`
}

func (r *Root) SaveLoopStateSpine(state *LoopStateSpine) error {
	if state == nil {
		return fmt.Errorf("loop state 为空")
	}
	if state.QuestID == "" {
		return fmt.Errorf("quest id 为空")
	}
	if state.SchemaVersion == "" {
		state.SchemaVersion = "loop_state_spine.v1"
	}
	state.UpdatedAtMs = NowMs()
	return WriteJSON(r.loopStateSpinePath(state.QuestID), state)
}

func (r *Root) LoadLoopStateSpine(qid string) (*LoopStateSpine, error) {
	if qid == "" {
		return nil, fmt.Errorf("quest id 为空")
	}
	return ReadJSON[LoopStateSpine](r.loopStateSpinePath(qid))
}

func (r *Root) PatchLoopStateSpine(qid string, patch func(*LoopStateSpine)) (*LoopStateSpine, error) {
	if qid == "" {
		return nil, fmt.Errorf("quest id 为空")
	}
	state, err := r.LoadLoopStateSpine(qid)
	if err != nil {
		state = &LoopStateSpine{QuestID: qid}
	}
	if patch != nil {
		patch(state)
	}
	if err := r.SaveLoopStateSpine(state); err != nil {
		return nil, err
	}
	return state, nil
}

func (r *Root) loopStateSpinePath(qid string) string {
	return filepath.Join(r.Sub(SubdirQuests), qid, "loop_state_spine.json")
}
