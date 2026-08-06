package quest

import (
	"testing"

	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

// ==================== 测试用 Attributable 实现 ====================

type testAttributable struct {
	status        model.QuestStatus
	blockedReason string
	finalVerdict  model.QuestVerdict
	finalComment  string
	applyStatus   model.ApplyStatus
	applyError    string
}

func (t *testAttributable) GetStatus() model.QuestStatus        { return t.status }
func (t *testAttributable) GetBlockedReason() string            { return t.blockedReason }
func (t *testAttributable) GetFinalVerdict() model.QuestVerdict { return t.finalVerdict }
func (t *testAttributable) GetFinalComment() string             { return t.finalComment }
func (t *testAttributable) GetApplyStatus() model.ApplyStatus   { return t.applyStatus }
func (t *testAttributable) GetApplyError() string               { return t.applyError }

// ==================== AttributeFailure 测试 ====================

func TestAttributeFailure_NilInput(t *testing.T) {
	result := AttributeFailure(nil)
	if result != nil {
		t.Error("nil 输入应该返回 nil")
	}
}

func TestAttributeFailure_SuccessStatus(t *testing.T) {
	q := &testAttributable{status: model.QuestStatusSuccess}
	result := AttributeFailure(q)
	if result != nil {
		t.Error("success 状态且无 apply 失败应该返回 nil")
	}
}

func TestAttributeFailure_SuccessWithApplyFailed(t *testing.T) {
	q := &testAttributable{
		status:      model.QuestStatusSuccess,
		applyStatus: model.ApplyStatusFailed,
		applyError:  "权限不足",
	}
	result := AttributeFailure(q)
	if result == nil {
		t.Fatal("应该返回失败归因")
	}
	if result.Reason != ReasonApplyFailed {
		t.Errorf("Reason = %q, want %q", result.Reason, ReasonApplyFailed)
	}
	if result.Stage != StageApply {
		t.Errorf("Stage = %q, want %q", result.Stage, StageApply)
	}
	if !result.Recoverable {
		t.Error("apply 失败应该是可恢复的")
	}
	if result.Message != "权限不足" {
		t.Errorf("Message = %q, want %q", result.Message, "权限不足")
	}
	if len(result.Actions) == 0 {
		t.Error("应该有建议的恢复动作")
	}
}

func TestAttributeFailure_CancelledStatus(t *testing.T) {
	q := &testAttributable{status: model.QuestStatusCancelled}
	result := AttributeFailure(q)
	if result == nil {
		t.Fatal("应该返回失败归因")
	}
	if result.Reason != ReasonCancelled {
		t.Errorf("Reason = %q, want %q", result.Reason, ReasonCancelled)
	}
	if result.Recoverable {
		t.Error("取消应该是不可恢复的")
	}
}

func TestAttributeFailure_RunningStatus(t *testing.T) {
	q := &testAttributable{status: model.QuestStatusRunning}
	result := AttributeFailure(q)
	if result != nil {
		t.Error("非终态且非 blocked 应该返回 nil")
	}
}

// ==================== blocked 状态归因测试 ====================

func TestAttributeBlocked_BudgetExceeded(t *testing.T) {
	tests := []struct {
		name   string
		reason string
	}{
		{"中文预算", "预算超支，已达最大回合数"},
		{"英文 budget", "budget exceeded: max turns reached"},
		{"中文回合", "回合数用尽"},
		{"turns", "max turns reached"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &testAttributable{
				status:        model.QuestStatusBlocked,
				blockedReason: tt.reason,
			}
			result := AttributeFailure(q)
			if result == nil {
				t.Fatal("应该返回失败归因")
			}
			if result.Reason != ReasonBudgetExceeded {
				t.Errorf("Reason = %q, want %q", result.Reason, ReasonBudgetExceeded)
			}
			if result.Stage != StageWarrior {
				t.Errorf("Stage = %q, want %q", result.Stage, StageWarrior)
			}
			if !result.Recoverable {
				t.Error("预算超支应该是可恢复的")
			}
		})
	}
}

func TestAttributeBlocked_DurationExceeded(t *testing.T) {
	tests := []struct {
		name   string
		reason string
	}{
		{"中文超时", "执行超时"},
		{"英文 timeout", "execution timeout"},
		{"超过时长", "已超过最大时长"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &testAttributable{
				status:        model.QuestStatusBlocked,
				blockedReason: tt.reason,
			}
			result := AttributeFailure(q)
			if result == nil {
				t.Fatal("应该返回失败归因")
			}
			if result.Reason != ReasonDurationExceeded {
				t.Errorf("Reason = %q, want %q", result.Reason, ReasonDurationExceeded)
			}
		})
	}
}

func TestAttributeBlocked_UserConfirmTimeout(t *testing.T) {
	q := &testAttributable{
		status:        model.QuestStatusBlocked,
		blockedReason: "等待用户终审超时",
	}
	result := AttributeFailure(q)
	if result == nil {
		t.Fatal("应该返回失败归因")
	}
	if result.Reason != ReasonDurationExceeded {
		t.Errorf("Reason = %q, want %q", result.Reason, ReasonDurationExceeded)
	}
	if result.Stage != StageUser {
		t.Errorf("Stage = %q, want %q", result.Stage, StageUser)
	}
}

func TestAttributeBlocked_AgentError(t *testing.T) {
	tests := []struct {
		name   string
		reason string
	}{
		{"agent error", "agent 调用失败"},
		{"executor error", "executor error: connection refused"},
		{"执行器错误", "执行器初始化失败"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &testAttributable{
				status:        model.QuestStatusBlocked,
				blockedReason: tt.reason,
			}
			result := AttributeFailure(q)
			if result == nil {
				t.Fatal("应该返回失败归因")
			}
			if result.Reason != ReasonAgentError {
				t.Errorf("Reason = %q, want %q", result.Reason, ReasonAgentError)
			}
			if result.Stage != StageWarrior {
				t.Errorf("Stage = %q, want %q", result.Stage, StageWarrior)
			}
		})
	}
}

func TestAttributeBlocked_UnknownReason(t *testing.T) {
	q := &testAttributable{
		status:        model.QuestStatusBlocked,
		blockedReason: "一些奇怪的原因",
	}
	result := AttributeFailure(q)
	if result == nil {
		t.Fatal("应该返回失败归因")
	}
	if result.Reason != ReasonUnknown {
		t.Errorf("Reason = %q, want %q", result.Reason, ReasonUnknown)
	}
	if !result.Recoverable {
		t.Error("未知原因的 blocked 默认应该是可恢复的")
	}
}

// ==================== failed 状态归因测试 ====================

func TestAttributeFailed_UserReject(t *testing.T) {
	q := &testAttributable{
		status:       model.QuestStatusFailed,
		finalVerdict: model.VerdictReject,
		finalComment: "质量不符合要求",
	}
	result := AttributeFailure(q)
	if result == nil {
		t.Fatal("应该返回失败归因")
	}
	if result.Reason != ReasonUserReject {
		t.Errorf("Reason = %q, want %q", result.Reason, ReasonUserReject)
	}
	if result.Stage != StageMage {
		t.Errorf("Stage = %q, want %q", result.Stage, StageMage)
	}
	if !result.Recoverable {
		t.Error("用户拒绝应该是可恢复的（可以返工）")
	}
	if len(result.Actions) == 0 {
		t.Error("应该有建议的恢复动作")
	}
	if result.Message != "质量不符合要求" {
		t.Errorf("Message = %q, want %q", result.Message, "质量不符合要求")
	}
}

func TestAttributeFailed_NoVerdict_AgentError(t *testing.T) {
	q := &testAttributable{
		status:       model.QuestStatusFailed,
		finalComment: "agent 执行出错",
	}
	result := AttributeFailure(q)
	if result == nil {
		t.Fatal("应该返回失败归因")
	}
	if result.Reason != ReasonAgentError {
		t.Errorf("Reason = %q, want %q", result.Reason, ReasonAgentError)
	}
	if result.Stage != StageWarrior {
		t.Errorf("Stage = %q, want %q", result.Stage, StageWarrior)
	}
}

func TestAttributeFailed_NoVerdict_Unknown(t *testing.T) {
	q := &testAttributable{
		status:       model.QuestStatusFailed,
		finalComment: "莫名其妙就失败了",
	}
	result := AttributeFailure(q)
	if result == nil {
		t.Fatal("应该返回失败归因")
	}
	if result.Reason != ReasonUnknown {
		t.Errorf("Reason = %q, want %q", result.Reason, ReasonUnknown)
	}
	if result.Stage != StageWarrior {
		t.Errorf("Stage = %q, want %q (默认应为 warrior)", result.Stage, StageWarrior)
	}
}

// ==================== QuestMeta.AttributeFailure 测试 ====================

func TestQuestMeta_AttributeFailure(t *testing.T) {
	q := &QuestMeta{
		Status:        model.QuestStatusBlocked,
		BlockedReason: "预算超支",
	}
	result := q.AttributeFailure()
	if result == nil {
		t.Fatal("应该返回失败归因")
	}
	if result.Reason != ReasonBudgetExceeded {
		t.Errorf("Reason = %q, want %q", result.Reason, ReasonBudgetExceeded)
	}
}

// ==================== containsAny 辅助函数测试 ====================

func TestContainsAny(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		keywords []string
		want     bool
	}{
		{"匹配单个", "hello world", []string{"hello"}, true},
		{"匹配多个中的一个", "hello world", []string{"foo", "world", "bar"}, true},
		{"不匹配", "hello world", []string{"foo", "bar"}, false},
		{"大小写不敏感", "Hello World", []string{"hello"}, true},
		{"空字符串", "", []string{"test"}, false},
		{"空关键词", "test", []string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := containsAny(tt.s, tt.keywords...); got != tt.want {
				t.Errorf("containsAny(%q, %v) = %v, want %v", tt.s, tt.keywords, got, tt.want)
			}
		})
	}
}
