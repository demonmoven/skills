package fsstore

import "path/filepath"

// WorkflowInstanceState 描述一条跨 quest 的工作流实例的运行快照，
// 用于串联多个 quest 形成更大的交付单元（pipeline / design-then-execute 等）。
type WorkflowInstanceState struct {
	ID           string         `json:"id"`
	WorkflowID   string         `json:"workflow_id"`
	Status       string         `json:"status"`
	LastGlobalID int64          `json:"last_global_id"`
	QuestIDs     []string       `json:"quest_ids,omitempty"`
	CreatedAtMs  int64          `json:"created_at_ms"`
	UpdatedAtMs  int64          `json:"updated_at_ms"`
	Meta         map[string]any `json:"meta,omitempty"`
}

// GetWorkflowInstance 按 ID 读取工作流实例。文件不存在返回错误。
func (r *Root) GetWorkflowInstance(id string) (*WorkflowInstanceState, error) {
	return ReadJSON[WorkflowInstanceState](r.workflowInstancePath(id))
}

// SaveWorkflowInstance 持久化工作流实例。首次保存会设置 CreatedAtMs，
// 每次保存都会刷新 UpdatedAtMs。nil 或空 ID 静默跳过。
func (r *Root) SaveWorkflowInstance(state *WorkflowInstanceState) error {
	if state == nil || state.ID == "" {
		return nil
	}
	now := NowMs()
	if state.CreatedAtMs == 0 {
		state.CreatedAtMs = now
	}
	state.UpdatedAtMs = now
	return WriteJSON(r.workflowInstancePath(state.ID), state)
}

func (r *Root) workflowInstancePath(id string) string {
	return filepath.Join(r.Sub(SubdirWorkspace, "workflows"), id+".json")
}
