package policy

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"
)

const (
	ActionRequireUser  = "require_user"
	ActionAutoPass     = "auto_pass"
	ActionAutoComplete = "auto_complete"
	ActionRetry        = "retry"
	ActionEscalate     = "escalate_to_user_review"

	EffectNone               = "none"
	EffectWorkspaceDiff      = "workspace_diff"
	EffectContextStore       = "context_store"
	EffectExternalSideEffect = "external_side_effect"

	RecoveryDefaultAddTurns           = 5
	RecoveryDefaultAddDurationMinutes = 15

	// RecoveryBaseCooldownSeconds / RecoveryMaxCooldownSeconds：恢复重试之间
	// 的指数退避（base * 2^recovery_count，封顶 max）。
	RecoveryBaseCooldownSeconds = 30
	RecoveryMaxCooldownSeconds  = 300
)

type InputFacts struct {
	QuestID              string   `json:"quest_id"`
	QuestType            string   `json:"quest_type,omitempty"`
	Intensity            string   `json:"intensity,omitempty"`
	Source               string   `json:"source,omitempty"`
	WorkspacePath        string   `json:"workspace_path,omitempty"`
	EffectType           string   `json:"effect_type,omitempty"`
	HasWorkspaceDiff     bool     `json:"has_workspace_diff"`
	SideEffectLevel      string   `json:"side_effect_level,omitempty"`
	HasExternalOutput    bool     `json:"has_external_output"`
	AllowL2              bool     `json:"allow_l2"`
	CurrentPhaseIdx      int      `json:"current_phase_idx,omitempty"`
	CurrentPhaseType     string   `json:"current_phase_type,omitempty"`
	PhaseReadOnly        bool     `json:"phase_read_only"`
	PhaseEndSignal       string   `json:"phase_end_signal,omitempty"`
	AutomationID         string   `json:"automation_id,omitempty"`
	AutomationTier       string   `json:"automation_tier,omitempty"`
	AutomationTags       []string `json:"automation_tags,omitempty"`
	IsOfficialAutomation bool     `json:"is_official_automation"`
	ReviewVerdict        string   `json:"review_verdict,omitempty"`
	ReviewScore          int      `json:"review_score,omitempty"`
	HasAutoEvidence      bool     `json:"has_auto_evidence"`
	EvidencePassed       bool     `json:"evidence_passed"`
	ReworkCount          int      `json:"rework_count,omitempty"`
	RecoveryCount        int      `json:"recovery_count,omitempty"`
	RecoveryElapsedSeconds int      `json:"recovery_elapsed_seconds,omitempty"`
	BlockedReason        string   `json:"blocked_reason,omitempty"`
	BlockedReasonCode    string   `json:"blocked_reason_code,omitempty"`
	BlockedCategory      string   `json:"blocked_category,omitempty"`
}

func (f InputFacts) Hash() string {
	raw, _ := json.Marshal(f)
	sum := sha256.Sum256(raw)
	return "sha256:" + fmt.Sprintf("%x", sum[:])
}

func (f InputFacts) Map() map[string]any {
	raw, _ := json.Marshal(f)
	out := map[string]any{}
	_ = json.Unmarshal(raw, &out)
	return out
}

type Decision struct {
	PolicyName    string         `json:"policy_name"`
	Action        string         `json:"action"`
	Reason        string         `json:"reason,omitempty"`
	Conditions    map[string]any `json:"conditions,omitempty"`
	DecidedAt     time.Time      `json:"decided_at"`
	DecisionID    string         `json:"decision_id"`
	InputHash     string         `json:"input_hash"`
	SafeByDefault bool           `json:"safe_by_default"`
}

type AuditEvent struct {
	QuestID    string         `json:"quest_id"`
	PhaseIdx   int            `json:"phase_idx,omitempty"`
	Decision   Decision       `json:"decision"`
	InputFacts map[string]any `json:"input_facts"`
}

type ReviewPolicy struct{}

func (ReviewPolicy) Decide(f InputFacts) Decision {


	// HOTL v0.2: checker pass = 最优解。人审不动 agent 产出（comprehension debt），
	// user_review 作为"人审质量"是伪命题。checker 给 pass 就自主闭环。
	requiresImpactNotification := f.AllowL2 || f.EffectType == EffectExternalSideEffect

	if f.ReviewVerdict == "pass" {
		return Decision{
			PolicyName: "checker_pass_auto_pass",
			Action:     ActionAutoPass,
			Reason:     "checker verdict=pass, auto-close loop per HOTL v0.2",
			Conditions: map[string]any{
				"effect_type":                  f.EffectType,
				"has_workspace_diff":           f.HasWorkspaceDiff,
				"requires_impact_notification": requiresImpactNotification,
				"intensity":                    f.Intensity,
				"rework_count":                 f.ReworkCount,
			},
			DecidedAt:     time.Now().UTC(),
			DecisionID:    decisionID("checker_pass_auto_pass", f),
			InputHash:     f.Hash(),
			SafeByDefault: false,
		}
	}
	return requireUser("default_require_user", "checker verdict is not pass", f, nil)
}

type RecoveryPolicy struct{}

func (RecoveryPolicy) Decide(f InputFacts) Decision {
	reasonCode := f.BlockedReasonCode
	if reasonCode == "" {
		reasonCode = f.BlockedReason
	}
	if reasonCode == "" {
		return requireUser("recovery_no_block_reason", "missing blocked reason", f, nil)
	}
	if f.AllowL2 || f.EffectType == EffectExternalSideEffect {
		return requireUser("recovery_global_safety_floor", "high-risk side effects require user recovery", f, map[string]any{"guardrail": true})
	}

	// 时间窗口预算模型：按 (reason_code, category) 分配恢复窗口。
	// 窗口内用指数退避自动重试，窗口耗尽才升级用户。
	// 不再有全局 max_attempts 硬熔断——时间窗口本身就是熔断器。
	budget := recoveryBudget(reasonCode, f.BlockedCategory)
	if budget.WindowSeconds == 0 {
		return requireUser(budget.PolicyName, budget.Reason, f, nil)
	}
	if f.RecoveryElapsedSeconds >= budget.WindowSeconds {
		return requireUser("recovery_window_expired", "recovery time window exhausted, escalate to user", f, map[string]any{
			"window_seconds":  budget.WindowSeconds,
			"elapsed_seconds": f.RecoveryElapsedSeconds,
			"attempts":        f.RecoveryCount,
		})
	}
	return retryWithBudget(budget, f)
}

func HardGuardrailReason(f InputFacts) (string, bool) {
	switch {
	case f.AllowL2:
		return "L2 side-effect permission requires user review", true
	case f.EffectType == EffectExternalSideEffect:
		return "external side effect requires user review", true

	default:
		return "", false
	}
}

// RecoveryBudget 定义单个 category 的恢复时间窗口预算。
type RecoveryBudget struct {
	PolicyName    string
	Reason        string
	WindowSeconds int // 0 = 不可恢复，立即升级用户
	BackoffBase   int // 首次退避秒数
	MaxAttempts   int // 窗口内最大尝试次数（兜底防空转）
}

// recoveryBudget 按 (reasonCode, category) 返回恢复预算。
func recoveryBudget(reasonCode, category string) RecoveryBudget {
	switch reasonCode {
	case "agent_consecutive_errors", "consecutive_errors":
		return RecoveryBudget{"consecutive_errors_retry", "transient consecutive errors", 600, 30, 5}
	case "agent_error_session_lost":
		return RecoveryBudget{"session_lost_retry", "session loss is transient", 60, 5, 3}
	case "agent_error_transient":
		return RecoveryBudget{"transient_retry", "transient error, auto-retry within window", 600, 30, 5}
	case "no_progress":
		return RecoveryBudget{"no_progress_retry", "no progress, retry with extra budget", 120, 30, 2}
	case "agent_phase_error":
		return budgetByCategory(category)
	default:
		return RecoveryBudget{"recovery_default_require_user", "no recovery policy matched", 0, 0, 0}
	}
}

func budgetByCategory(category string) RecoveryBudget {
	switch category {
	case "transient":
		return RecoveryBudget{"agent_phase_transient_retry", "transient agent phase error", 600, 30, 5}
	case "rate_limit":
		return RecoveryBudget{"agent_phase_rate_limit_retry", "rate limited, backoff and retry", 300, 60, 3}
	case "session_lost":
		return RecoveryBudget{"agent_phase_session_retry", "session lost, rebuild", 60, 5, 3}
	case "auth", "configuration":
		return RecoveryBudget{"agent_phase_" + category, category + " error requires user", 0, 0, 0}
	default:
		return RecoveryBudget{"recovery_default_require_user", "unknown category", 0, 0, 0}
	}
}

func retryWithBudget(budget RecoveryBudget, f InputFacts) Decision {
	cooldown := recoveryCooldownSeconds(f.RecoveryCount, budget.BackoffBase)
	return Decision{
		PolicyName: budget.PolicyName,
		Action:     ActionRetry,
		Reason:     budget.Reason,
		Conditions: map[string]any{
			"blocked_reason_code":  f.BlockedReasonCode,
			"blocked_category":     f.BlockedCategory,
			"recovery_count":       f.RecoveryCount,
			"add_turns":            RecoveryDefaultAddTurns,
			"add_duration_minutes": RecoveryDefaultAddDurationMinutes,
			"window_seconds":       budget.WindowSeconds,
			"max_attempts":         budget.MaxAttempts,
			"cooldown_seconds":     cooldown,
		},
		DecidedAt:     time.Now().UTC(),
		DecisionID:    decisionID(budget.PolicyName, f),
		InputHash:     f.Hash(),
		SafeByDefault: false,
	}
}

// recoveryCooldownSeconds 计算指数退避冷却秒数：base * 2^recoveryCount，封顶 max。
func recoveryCooldownSeconds(recoveryCount int, backoffBase int) int {
	if recoveryCount < 0 {
		recoveryCount = 0
	}
	if backoffBase <= 0 {
		backoffBase = RecoveryBaseCooldownSeconds
	}
	cd := backoffBase
	for i := 0; i < recoveryCount && cd < RecoveryMaxCooldownSeconds; i++ {
		cd *= 2
	}
	if cd > RecoveryMaxCooldownSeconds {
		cd = RecoveryMaxCooldownSeconds
	}
	return cd
}



func requireUser(name, reason string, f InputFacts, conditions map[string]any) Decision {
	return Decision{
		PolicyName:    name,
		Action:        ActionRequireUser,
		Reason:        reason,
		Conditions:    conditions,
		DecidedAt:     time.Now().UTC(),
		DecisionID:    decisionID(name, f),
		InputHash:     f.Hash(),
		SafeByDefault: true,
	}
}

func decisionID(policyName string, f InputFacts) string {
	sum := sha256.Sum256([]byte(policyName + ":" + f.Hash()))
	return "pol_dec_" + fmt.Sprintf("%x", sum[:8])
}

func RecoveryAttemptLimitDecision(f InputFacts, maxAttempts int) Decision {
	return requireUser("recovery_attempt_limit", "recovery attempt limit reached within window", f, map[string]any{
		"recovery_count": f.RecoveryCount,
		"max_attempts":   maxAttempts,
	})
}

func RecoveryWindowExpiredDecision(f InputFacts, windowSeconds int) Decision {
	return requireUser("recovery_window_expired", "recovery time window exhausted", f, map[string]any{
		"window_seconds":  windowSeconds,
		"elapsed_seconds": f.RecoveryElapsedSeconds,
		"attempts":        f.RecoveryCount,
	})
}
