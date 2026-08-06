package fsstore

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

// RecoveryStatus 表示 quest 恢复策略的当前状态。
type RecoveryStatus string

const (
	// RecoveryStatusPending 表示尚未开始执行恢复动作。
	RecoveryStatusPending RecoveryStatus = "pending"
	// RecoveryStatusRunning 表示恢复动作正在进行中。
	RecoveryStatusRunning RecoveryStatus = "running"
	// RecoveryStatusSucceeded 表示恢复动作已成功，quest 可继续推进。
	RecoveryStatusSucceeded RecoveryStatus = "succeeded"
	// RecoveryStatusFailed 表示恢复动作失败，需要人工介入。
	RecoveryStatusFailed RecoveryStatus = "failed"
)

// RecoveryState 持久化单个 quest 的恢复执行上下文，供跨会话断点续跑。
type RecoveryState struct {
	QuestID   string `json:"quest_id"`
	PhaseIdx  int    `json:"phase_idx,omitempty"`
	SessionID string `json:"session_id,omitempty"`

	PolicyName   string `json:"policy_name,omitempty"`
	AttemptIndex int    `json:"attempt_index,omitempty"`
	ActionIndex  int    `json:"action_index,omitempty"`
	InputHash    string `json:"input_hash,omitempty"`

	AddTurns           int `json:"add_turns,omitempty"`
	AddDurationMinutes int `json:"add_duration_minutes,omitempty"`
	CooldownSeconds    int `json:"cooldown_seconds,omitempty"`
	MaxAttempts        int `json:"max_attempts,omitempty"`
	WindowSeconds      int `json:"window_seconds,omitempty"`

	Status        RecoveryStatus `json:"status"`
	LastAction    string         `json:"last_action,omitempty"`
	LastError     string         `json:"last_error,omitempty"`
	NextRetryAt   time.Time      `json:"next_retry_at,omitempty"`
	StartedAt     time.Time      `json:"started_at,omitempty"`
	LastAttemptAt time.Time      `json:"last_attempt_at,omitempty"`

	BlockReason   string `json:"block_reason,omitempty"`
	OriginalError string `json:"original_error,omitempty"`
}

// GetRecoveryState 读取 quest 的恢复状态。qid 为空或文件不存在返回错误。
func (r *Root) GetRecoveryState(qid string) (*RecoveryState, error) {
	if qid == "" {
		return nil, fmt.Errorf("quest id 为空")
	}
	return ReadJSON[RecoveryState](r.recoveryStatePath(qid))
}

// SaveRecoveryState 保存 quest 的恢复状态。首次保存会补齐默认 status 和 StartedAt。
func (r *Root) SaveRecoveryState(state *RecoveryState) error {
	if state == nil {
		return fmt.Errorf("recovery state 为空")
	}
	if state.QuestID == "" {
		return fmt.Errorf("quest id 为空")
	}
	if state.Status == "" {
		state.Status = RecoveryStatusPending
	}
	now := time.Now().UTC()
	if state.StartedAt.IsZero() {
		state.StartedAt = now
	}
	state.LastAttemptAt = now
	return WriteJSON(r.recoveryStatePath(state.QuestID), state)
}

func (r *Root) recoveryStatePath(qid string) string {
	return filepath.Join(r.Sub(SubdirWorkspace, "recovery_state"), qid+".json")
}

// DeleteRecoveryState 删除指定 quest 的 recovery state。
// quest 进入终态（success/failed/cancelled）后调用，避免残留孤儿。
func (r *Root) DeleteRecoveryState(qid string) error {
	if qid == "" {
		return fmt.Errorf("quest id 为空")
	}
	err := os.Remove(r.recoveryStatePath(qid))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// RecoveryStatusFromQuestStatus 将 quest 运行状态映射到恢复状态的粗略分类。
// blocked → pending；running/reviewing → running；其余 → failed。
func RecoveryStatusFromQuestStatus(status model.QuestStatus) RecoveryStatus {
	switch status {
	case model.QuestStatusBlocked:
		return RecoveryStatusPending
	case model.QuestStatusRunning, model.QuestStatusReviewing:
		return RecoveryStatusRunning
	default:
		return RecoveryStatusFailed
	}
}
