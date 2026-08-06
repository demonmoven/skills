package fsstore

import "path/filepath"

// AutomationState 记录单个 automation 在某个 quest 维度下的推进游标，
// 避免事件驱动型 automation 重复处理已消费的事件。
type AutomationState struct {
	AutomationID string `json:"automation_id"`
	QuestID      string `json:"quest_id"`
	LastEventID  int64  `json:"last_event_id"`
	LastGlobalID int64  `json:"last_global_id,omitempty"`
	LastRunTs    int64  `json:"last_run_ts"`
	RunCount     int    `json:"run_count"`
}

// GetAutomationState 读取指定 automation+quest 的推进状态；文件不存在返回错误。
func (r *Root) GetAutomationState(automationID, qid string) (*AutomationState, error) {
	path := r.automationStatePath(automationID, qid)
	state, err := ReadJSON[AutomationState](path)
	if err != nil {
		return nil, err
	}
	return state, nil
}

// SaveAutomationState 持久化 automation 推进状态。state 为空或关键字段缺失时静默跳过。
func (r *Root) SaveAutomationState(state *AutomationState) error {
	if state == nil {
		return nil
	}
	if state.AutomationID == "" || state.QuestID == "" {
		return nil
	}
	return WriteJSON(r.automationStatePath(state.AutomationID, state.QuestID), state)
}

func (r *Root) automationStatePath(automationID, qid string) string {
	return filepath.Join(r.Sub(SubdirWorkspace, "automation_state"), automationID+"__"+qid+".json")
}
