package policy

import "testing"

func TestReviewPolicy_CheckerPassAutoPass(t *testing.T) {
	facts := InputFacts{
		QuestID:              "q1",
		Source:               "automation",
		AutomationID:         "auto_context_refresh",
		EffectType:           EffectContextStore,
		ReviewVerdict:        "pass",
		ReviewScore:          8,
	}
	got := (ReviewPolicy{}).Decide(facts)
	if got.Action != ActionAutoPass || got.PolicyName != "checker_pass_auto_pass" || got.SafeByDefault {
		t.Fatalf("decision = %+v, want checker_pass_auto_pass", got)
	}
	if got.InputHash == "" || got.DecisionID == "" {
		t.Fatalf("decision should carry audit ids: %+v", got)
	}
}

func TestReviewPolicy_WorkspaceDiffPassAutoPass(t *testing.T) {
	facts := InputFacts{
		QuestID:              "q1",
		Source:               "user",
		EffectType:           EffectWorkspaceDiff,
		HasWorkspaceDiff:     true,
		ReviewVerdict:        "pass",
		ReviewScore:          7,
	}
	got := (ReviewPolicy{}).Decide(facts)
	if got.Action != ActionAutoPass {
		t.Fatalf("workspace_diff + pass should auto_pass per HOTL v0.2, got %+v", got)
	}
	if got.Conditions["has_workspace_diff"] != true {
		t.Fatalf("decision should record has_workspace_diff: %+v", got.Conditions)
	}
}

func TestReviewPolicy_HardGuardrailsRequireUser(t *testing.T) {
	cases := []struct {
		name string
		f    InputFacts
	}{
		{
			name: "workspace diff without pass",
		},
		{
			name: "allow l2 without pass",
		},
		{
			name: "external side effect without pass",
		},
		{
			name: "not allowlisted without pass",
			f:    InputFacts{QuestID: "q1", Source: "automation", EffectType: EffectContextStore},
		},
		{
			name: "review did not pass",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := (ReviewPolicy{}).Decide(tc.f)
			if got.Action != ActionRequireUser || !got.SafeByDefault {
				t.Fatalf("decision = %+v, want safe require_user", got)
			}
		})
	}
}

func TestRecoveryPolicy_ConservativeDecisions(t *testing.T) {
	// consecutive_errors: window-based retry with budget
	retryFacts := InputFacts{QuestID: "q1", BlockedReason: "agent 连续错误达到阈值：2", BlockedReasonCode: "agent_consecutive_errors", EffectType: EffectNone}
	got := (RecoveryPolicy{}).Decide(retryFacts)
	if got.Action != ActionRetry || got.SafeByDefault {
		t.Fatalf("decision = %+v, want conservative retry", got)
	}
	budget := recoveryBudget("agent_consecutive_errors", "")
	if got.Conditions["add_turns"] != RecoveryDefaultAddTurns ||
		got.Conditions["add_duration_minutes"] != RecoveryDefaultAddDurationMinutes ||
		got.Conditions["max_attempts"] != budget.MaxAttempts {
		t.Fatalf("retry decision should carry execution params: %+v", got.Conditions)
	}
	if got.Conditions["window_seconds"] != budget.WindowSeconds {
		t.Fatalf("retry decision should carry window_seconds: %+v", got.Conditions)
	}
	if cd, ok := got.Conditions["cooldown_seconds"]; !ok || cd.(int) < budget.BackoffBase {
		t.Fatalf("retry decision should carry cooldown_seconds: %+v", got.Conditions)
	}

	// Window expired: elapsed >= window → require_user
	windowExpiredFacts := InputFacts{
		QuestID: "q1", BlockedReasonCode: "agent_consecutive_errors",
		RecoveryCount: 3, RecoveryElapsedSeconds: budget.WindowSeconds,
		EffectType: EffectNone,
	}
	got = (RecoveryPolicy{}).Decide(windowExpiredFacts)
	if got.Action != ActionRequireUser || !got.SafeByDefault {
		t.Fatalf("window expired should require_user: %+v", got)
	}
	if got.PolicyName != "recovery_window_expired" {
		t.Fatalf("window expired policy = %s, want recovery_window_expired", got.PolicyName)
	}

	// External side effect → always require user
	riskyFacts := InputFacts{QuestID: "q1", BlockedReasonCode: "agent_consecutive_errors", EffectType: EffectExternalSideEffect}
	got = (RecoveryPolicy{}).Decide(riskyFacts)
	if got.Action != ActionRequireUser || !got.SafeByDefault {
		t.Fatalf("decision = %+v, want require_user for external side effect", got)
	}
}

func TestRecoveryPolicy_NoProgressStaysTwoAttempts(t *testing.T) {
	f := InputFacts{QuestID: "q1", BlockedReasonCode: "no_progress", EffectType: EffectNone}
	got := (RecoveryPolicy{}).Decide(f)
	if got.Action != ActionRetry {
		t.Fatalf("no_progress should retry: %+v", got)
	}
	budget := recoveryBudget("no_progress", "")
	if got.Conditions["max_attempts"] != budget.MaxAttempts {
		t.Fatalf("no_progress max_attempts = %v, want %d", got.Conditions["max_attempts"], budget.MaxAttempts)
	}
	if got.Conditions["window_seconds"] != budget.WindowSeconds {
		t.Fatalf("no_progress window_seconds = %v, want %d", got.Conditions["window_seconds"], budget.WindowSeconds)
	}
}

func TestRecoveryPolicy_SessionLostRetries(t *testing.T) {
	cases := []struct {
		code       string
		wantPolicy string
	}{
		{"agent_error_session_lost", "session_lost_retry"},
		{"agent_error_transient", "transient_retry"},
	}
	for _, tc := range cases {
		f := InputFacts{QuestID: "q1", BlockedReasonCode: tc.code, EffectType: EffectNone}
		got := (RecoveryPolicy{}).Decide(f)
		if got.Action != ActionRetry {
			t.Fatalf("%s should retry, got %s", tc.code, got.Action)
		}
		if got.PolicyName != tc.wantPolicy {
			t.Fatalf("%s policy = %s, want %s", tc.code, got.PolicyName, tc.wantPolicy)
		}
	}
}

func TestRecoveryPolicy_AgentPhaseErrorByCategory(t *testing.T) {
	// transient → retry
	transientFacts := InputFacts{
		QuestID:           "q1",
		BlockedReasonCode: "agent_phase_error",
		BlockedCategory:   "transient",
		EffectType:        EffectNone,
	}
	got := (RecoveryPolicy{}).Decide(transientFacts)
	if got.Action != ActionRetry {
		t.Fatalf("agent_phase_error+transient should retry, got %s", got.Action)
	}
	if got.PolicyName != "agent_phase_transient_retry" {
		t.Fatalf("policy = %s, want agent_phase_transient_retry", got.PolicyName)
	}

	// rate_limit → retry
	rateFacts := InputFacts{
		QuestID:           "q1",
		BlockedReasonCode: "agent_phase_error",
		BlockedCategory:   "rate_limit",
		EffectType:        EffectNone,
	}
	got = (RecoveryPolicy{}).Decide(rateFacts)
	if got.Action != ActionRetry {
		t.Fatalf("agent_phase_error+rate_limit should retry, got %s", got.Action)
	}

	// auth → require user
	authFacts := InputFacts{
		QuestID:           "q1",
		BlockedReasonCode: "agent_phase_error",
		BlockedCategory:   "auth",
		EffectType:        EffectNone,
	}
	got = (RecoveryPolicy{}).Decide(authFacts)
	if got.Action != ActionRequireUser {
		t.Fatalf("agent_phase_error+auth should require_user, got %s", got.Action)
	}

	// empty category → require user
	unknownFacts := InputFacts{
		QuestID:           "q1",
		BlockedReasonCode: "agent_phase_error",
		BlockedCategory:   "",
		EffectType:        EffectNone,
	}
	got = (RecoveryPolicy{}).Decide(unknownFacts)
	if got.Action != ActionRequireUser {
		t.Fatalf("agent_phase_error+empty should require_user, got %s", got.Action)
	}
}

func TestRecoveryCooldownExponentialBackoff(t *testing.T) {
	cases := []struct {
		count int
		base  int
		want  int
	}{
		{0, RecoveryBaseCooldownSeconds, RecoveryBaseCooldownSeconds},
		{1, RecoveryBaseCooldownSeconds, RecoveryBaseCooldownSeconds * 2},
		{2, RecoveryBaseCooldownSeconds, RecoveryBaseCooldownSeconds * 4},
		{10, RecoveryBaseCooldownSeconds, RecoveryMaxCooldownSeconds},
		{0, 5, 5},
		{1, 5, 10},
		{2, 5, 20},
	}
	for _, c := range cases {
		if got := recoveryCooldownSeconds(c.count, c.base); got != c.want {
			t.Fatalf("recoveryCooldownSeconds(%d, %d) = %d, want %d", c.count, c.base, got, c.want)
		}
	}
}

func TestRecoveryAttemptLimitDecision(t *testing.T) {
	f := InputFacts{QuestID: "q1", BlockedReasonCode: "agent_consecutive_errors", RecoveryCount: 5, EffectType: EffectNone}
	got := RecoveryAttemptLimitDecision(f, 5)
	if got.Action != ActionRequireUser || !got.SafeByDefault || got.Conditions["max_attempts"] != 5 {
		t.Fatalf("limit decision = %+v, want require_user with max_attempts=5", got)
	}
}

func TestRecoveryWindowExpiredDecision(t *testing.T) {
	f := InputFacts{QuestID: "q1", BlockedReasonCode: "no_progress", RecoveryCount: 2, RecoveryElapsedSeconds: 130, EffectType: EffectNone}
	got := RecoveryWindowExpiredDecision(f, 120)
	if got.Action != ActionRequireUser || !got.SafeByDefault {
		t.Fatalf("window expired decision = %+v, want require_user", got)
	}
	if got.Conditions["window_seconds"] != 120 || got.Conditions["elapsed_seconds"] != 130 {
		t.Fatalf("window expired should carry timing: %+v", got.Conditions)
	}
}

func TestInputFactsHashStable(t *testing.T) {
	f := InputFacts{QuestID: "q1", EffectType: EffectNone}
	if f.Hash() != f.Hash() {
		t.Fatal("hash should be stable")
	}
}
