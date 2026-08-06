package fsstore

import (
	"fmt"
	"os"
	"path/filepath"
)

type AgentSessionState struct {
	SessionID       string `json:"session_id"`
	QuestID         string `json:"quest_id"`
	Phase           int    `json:"phase"`
	AgentID         string `json:"agent_id,omitempty"`
	CapabilityTier  string `json:"capability_tier,omitempty"`
	HealthStatus    string `json:"health_status,omitempty"`
	StartedAtMs     int64  `json:"started_at_ms,omitempty"`
	FirstOutputAtMs int64  `json:"first_output_at_ms,omitempty"`
	LastOutputAtMs  int64  `json:"last_output_at_ms,omitempty"`
	ExitCode        *int   `json:"exit_code,omitempty"`
	UpdatedAtMs     int64  `json:"updated_at_ms,omitempty"`
}

func (r *Root) SaveAgentSessionState(state *AgentSessionState) error {
	if state == nil {
		return fmt.Errorf("agent session state 为空")
	}
	if state.QuestID == "" {
		return fmt.Errorf("quest id 为空")
	}
	if state.SessionID == "" {
		return fmt.Errorf("session id 为空")
	}
	if state.HealthStatus == "" {
		state.HealthStatus = "created"
	}
	now := NowMs()
	if state.StartedAtMs == 0 {
		state.StartedAtMs = now
	}
	state.UpdatedAtMs = now
	return WriteJSON(r.agentSessionStatePath(state.QuestID, state.SessionID), state)
}

func (r *Root) GetAgentSessionState(qid, sid string) (*AgentSessionState, error) {
	if qid == "" {
		return nil, fmt.Errorf("quest id 为空")
	}
	if sid == "" {
		return nil, fmt.Errorf("session id 为空")
	}
	return ReadJSON[AgentSessionState](r.agentSessionStatePath(qid, sid))
}

func (r *Root) ListAgentSessionStates(qid string) ([]*AgentSessionState, error) {
	if qid == "" {
		return nil, fmt.Errorf("quest id 为空")
	}
	dir := filepath.Join(r.Sub(SubdirWorkspace, "agent_sessions"), qid)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*AgentSessionState{}, nil
		}
		return nil, err
	}
	out := []*AgentSessionState{}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		state, err := ReadJSON[AgentSessionState](filepath.Join(dir, entry.Name()))
		if err == nil && state != nil {
			out = append(out, state)
		}
	}
	return out, nil
}

func (r *Root) PatchAgentSessionState(qid, sid string, patch func(*AgentSessionState)) (*AgentSessionState, error) {
	if qid == "" {
		return nil, fmt.Errorf("quest id 为空")
	}
	if sid == "" {
		return nil, fmt.Errorf("session id 为空")
	}
	state, err := r.GetAgentSessionState(qid, sid)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		state = &AgentSessionState{QuestID: qid, SessionID: sid}
	}
	if patch != nil {
		patch(state)
	}
	if err := r.SaveAgentSessionState(state); err != nil {
		return nil, err
	}
	return state, nil
}

func (r *Root) agentSessionStatePath(qid, sid string) string {
	return filepath.Join(r.Sub(SubdirWorkspace, "agent_sessions"), qid, sid+".json")
}
