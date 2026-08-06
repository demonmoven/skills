package fsstore

import (
	"fmt"
	"path/filepath"
)

// TrustTier0 允许 automation 启动（默认不自动执行任何委托）。
// TrustTier1 允许 automation 自动发起 execute 类执行型委托，但所有产出需人类终审。
// TrustTier2 允许 automation 对简单委托自动闭环（apply），但高风险改动仍需终审。
// TrustTier3 允许 automation 完全自主运行，含高风险外部副作用。
const (
	TrustTier0 = "tier_0"
	TrustTier1 = "tier_1"
	TrustTier2 = "tier_2"
	TrustTier3 = "tier_3"

	TrustOutcomeTypeTerminal = "terminal"
	TrustOutcomeTypeStage    = "stage"
)

// AutomationTrustState 是单个 automation 的信任分层运行态数据。
// 包含累计运行统计、最近结果、升级/降级历史和锁定状态。
type AutomationTrustState struct {
	AutomationID                  string                        `json:"automation_id"`
	Tier                          string                        `json:"tier"`
	Locked                        bool                          `json:"locked,omitempty"`
	ConsecutiveIndependentSuccess int                           `json:"consecutive_independent_success,omitempty"`
	ConsecutiveFailures           int                           `json:"consecutive_failures,omitempty"`
	TotalRuns                     int                           `json:"total_runs,omitempty"`
	TotalSuccess                  int                           `json:"total_success,omitempty"`
	TotalFailures                 int                           `json:"total_failures,omitempty"`
	OutcomeKeys                   []string                      `json:"outcome_keys,omitempty"`
	Recent                        []AutomationTrustOutcome      `json:"recent,omitempty"`
	History                       []AutomationTrustHistoryEntry `json:"history,omitempty"`
	UpdatedAtMs                   int64                         `json:"updated_at_ms,omitempty"`
}

// AutomationTrustOutcome 是 automation 单次委托执行的结果记录，用于去重和信任升级统计。
type AutomationTrustOutcome struct {
	QuestID          string `json:"quest_id"`
	OutcomeType      string `json:"outcome_type,omitempty"`
	Outcome          string `json:"outcome"`
	VerificationType string `json:"verification_type,omitempty"`
	Independent      bool   `json:"independent"`
	DedupKey         string `json:"dedup_key,omitempty"`
	TsMs             int64  `json:"ts_ms"`
}

// AutomationTrustHistoryEntry 记录 trust tier 变更历史（升级/降级/锁定），便于审计。
type AutomationTrustHistoryEntry struct {
	TsMs   int64  `json:"ts_ms"`
	From   string `json:"from,omitempty"`
	To     string `json:"to"`
	Reason string `json:"reason,omitempty"`
	Locked bool   `json:"locked,omitempty"`
}

// GetAutomationTrustState 读取 automation 的信任状态。文件不存在或 id 为空返回错误。
func (r *Root) GetAutomationTrustState(automationID string) (*AutomationTrustState, error) {
	if automationID == "" {
		return nil, fmt.Errorf("automation id 为空")
	}
	state, err := ReadJSON[AutomationTrustState](r.automationTrustPath(automationID))
	if err != nil {
		return nil, err
	}
	normalizeAutomationTrustState(state)
	return state, nil
}

// SaveAutomationTrustState 持久化信任状态。保存前自动执行 normalize（补默认 tier、截短队列、刷新时间戳）。
func (r *Root) SaveAutomationTrustState(state *AutomationTrustState) error {
	if state == nil {
		return fmt.Errorf("automation trust state 为空")
	}
	if state.AutomationID == "" {
		return fmt.Errorf("automation id 为空")
	}
	normalizeAutomationTrustState(state)
	state.UpdatedAtMs = NowMs()
	return WriteJSON(r.automationTrustPath(state.AutomationID), state)
}

// EnsureAutomationTrustState 保证指定 automation 有信任状态记录。
// 已存在则直接返回；不存在则以 tier_1 创建新记录并保存。
func (r *Root) EnsureAutomationTrustState(automationID string) (*AutomationTrustState, error) {
	state, err := r.GetAutomationTrustState(automationID)
	if err == nil {
		return state, nil
	}
	state = &AutomationTrustState{AutomationID: automationID, Tier: TrustTier1}
	if err := r.SaveAutomationTrustState(state); err != nil {
		return nil, err
	}
	return state, nil
}

// LockAutomationTrustTier 将 automation 的 trust tier 锁定到指定值，
// 写入 history 并同步更新 automation 配置。锁定后自动升级/降级不再生效。
func (r *Root) LockAutomationTrustTier(automationID, tier, reason string) (*AutomationTrustState, error) {
	if !IsValidTrustTier(tier) {
		return nil, fmt.Errorf("invalid trust tier: %s", tier)
	}
	state, err := r.EnsureAutomationTrustState(automationID)
	if err != nil {
		return nil, err
	}
	from := state.Tier
	state.Tier = tier
	state.Locked = true
	state.History = append(state.History, AutomationTrustHistoryEntry{
		TsMs:   NowMs(),
		From:   from,
		To:     tier,
		Reason: reason,
		Locked: true,
	})
	if err := r.SaveAutomationTrustState(state); err != nil {
		return nil, err
	}
	if cfg, err := r.GetAutomation(automationID); err == nil && cfg != nil {
		cfg.TrustTier = tier
		cfg.TrustTierLocked = true
		_ = r.SaveAutomation(cfg)
	}
	return state, nil
}

// IsValidTrustTier 判断给定 tier 字符串是否为合法的 4 档信任等级之一。
func IsValidTrustTier(tier string) bool {
	switch tier {
	case TrustTier0, TrustTier1, TrustTier2, TrustTier3:
		return true
	default:
		return false
	}
}

func normalizeAutomationTrustState(state *AutomationTrustState) {
	if state.Tier == "" {
		state.Tier = TrustTier1
	}
	known := map[string]bool{}
	for _, key := range state.OutcomeKeys {
		if key != "" {
			known[key] = true
		}
	}
	for i := range state.Recent {
		normalizeAutomationTrustOutcome(&state.Recent[i])
		if state.Recent[i].DedupKey != "" && !known[state.Recent[i].DedupKey] {
			state.OutcomeKeys = append(state.OutcomeKeys, state.Recent[i].DedupKey)
			known[state.Recent[i].DedupKey] = true
		}
	}
	if len(state.Recent) > 10 {
		state.Recent = append([]AutomationTrustOutcome(nil), state.Recent[len(state.Recent)-10:]...)
	}
	if len(state.OutcomeKeys) > 200 {
		state.OutcomeKeys = append([]string(nil), state.OutcomeKeys[len(state.OutcomeKeys)-200:]...)
	}
}

func normalizeAutomationTrustOutcome(outcome *AutomationTrustOutcome) {
	if outcome == nil {
		return
	}
	if outcome.OutcomeType == "" {
		outcome.OutcomeType = TrustOutcomeTypeTerminal
	}
	if outcome.DedupKey == "" {
		outcome.DedupKey = AutomationTrustOutcomeDedupKey(*outcome)
	}
}

// AutomationTrustOutcomeDedupKey 生成 outcome 的去重键。
// terminal 型按 quest 去重（每个委托只记一次终态）；stage 型按 quest+verification 去重。
func AutomationTrustOutcomeDedupKey(outcome AutomationTrustOutcome) string {
	outcomeType := outcome.OutcomeType
	if outcomeType == "" {
		outcomeType = TrustOutcomeTypeTerminal
	}
	switch outcomeType {
	case TrustOutcomeTypeStage:
		return outcome.QuestID + ":stage:" + outcome.VerificationType
	default:
		return outcome.QuestID + ":terminal"
	}
}

// HasOutcomeDedupKey 判断该信任状态中是否已记录过指定去重键的结果（查 recent + OutcomeKeys）。
func (state *AutomationTrustState) HasOutcomeDedupKey(key string) bool {
	if state == nil || key == "" {
		return false
	}
	for _, item := range state.Recent {
		normalizeAutomationTrustOutcome(&item)
		if item.DedupKey == key {
			return true
		}
	}
	for _, item := range state.OutcomeKeys {
		if item == key {
			return true
		}
	}
	return false
}

// AppendOutcome 追加一条执行结果到统计中。若结果已去重则返回 false 不做改动。
// 终态结果会累计 total/success/failure，影响连续成功/失败计数。
func (state *AutomationTrustState) AppendOutcome(outcome AutomationTrustOutcome) bool {
	if state == nil {
		return false
	}
	normalizeAutomationTrustOutcome(&outcome)
	if state.HasOutcomeDedupKey(outcome.DedupKey) {
		return false
	}
	if outcome.TsMs == 0 {
		outcome.TsMs = NowMs()
	}
	state.OutcomeKeys = append(state.OutcomeKeys, outcome.DedupKey)
	if len(state.OutcomeKeys) > 200 {
		state.OutcomeKeys = append([]string(nil), state.OutcomeKeys[len(state.OutcomeKeys)-200:]...)
	}
	state.Recent = append(state.Recent, outcome)
	if len(state.Recent) > 10 {
		state.Recent = append([]AutomationTrustOutcome(nil), state.Recent[len(state.Recent)-10:]...)
	}
	if outcome.OutcomeType == TrustOutcomeTypeTerminal {
		state.TotalRuns++
		switch outcome.Outcome {
		case "success":
			state.TotalSuccess++
			state.ConsecutiveFailures = 0
			if outcome.Independent {
				state.ConsecutiveIndependentSuccess++
			}
		case "failure":
			state.TotalFailures++
			state.ConsecutiveFailures++
			state.ConsecutiveIndependentSuccess = 0
		}
	}
	return true
}

func (r *Root) automationTrustPath(automationID string) string {
	return filepath.Join(r.Sub(SubdirWorkspace, "automation_trust"), automationID+".json")
}
