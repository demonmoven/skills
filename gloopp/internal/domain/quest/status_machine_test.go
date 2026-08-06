package quest

import (
	"testing"

	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

// ==================== 状态机基础测试 ====================

func TestIsTerminal(t *testing.T) {
	tests := []struct {
		status model.QuestStatus
		want   bool
	}{
		{model.QuestStatusSuccess, true},
		{model.QuestStatusFailed, true},
		{model.QuestStatusCancelled, true},
		{model.QuestStatusPending, false},
		{model.QuestStatusRunning, false},
		{model.QuestStatusReviewing, false},
		{model.QuestStatusWaitingInput, false},
		{model.QuestStatusUserReview, false},
		{model.QuestStatusBlocked, false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := IsTerminal(tt.status); got != tt.want {
				t.Errorf("IsTerminal(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestCanTransition_ValidTransitions(t *testing.T) {
	// 主要的合法迁移路径
	validPairs := []struct {
		from model.QuestStatus
		to   model.QuestStatus
	}{
		// 主线流程
		{"", model.QuestStatusPending},
		{model.QuestStatusPending, model.QuestStatusRunning},
		{model.QuestStatusPending, model.QuestStatusCancelled},
		{model.QuestStatusRunning, model.QuestStatusReviewing},
		{model.QuestStatusRunning, model.QuestStatusWaitingInput},
		{model.QuestStatusRunning, model.QuestStatusFailed},
		{model.QuestStatusRunning, model.QuestStatusBlocked},
		{model.QuestStatusRunning, model.QuestStatusCancelled},
		{model.QuestStatusReviewing, model.QuestStatusRunning},
		{model.QuestStatusReviewing, model.QuestStatusUserReview},
		{model.QuestStatusReviewing, model.QuestStatusSuccess},
		{model.QuestStatusReviewing, model.QuestStatusFailed},
		{model.QuestStatusReviewing, model.QuestStatusBlocked},
		{model.QuestStatusReviewing, model.QuestStatusCancelled},
		{model.QuestStatusUserReview, model.QuestStatusRunning},
		{model.QuestStatusUserReview, model.QuestStatusSuccess},
		{model.QuestStatusUserReview, model.QuestStatusFailed},
		{model.QuestStatusUserReview, model.QuestStatusBlocked},
		{model.QuestStatusUserReview, model.QuestStatusCancelled},
		{model.QuestStatusWaitingInput, model.QuestStatusRunning},
		{model.QuestStatusWaitingInput, model.QuestStatusBlocked},
		{model.QuestStatusWaitingInput, model.QuestStatusUserReview},
		{model.QuestStatusWaitingInput, model.QuestStatusCancelled},

		// blocked 恢复路径
		{model.QuestStatusBlocked, model.QuestStatusRunning},
		{model.QuestStatusBlocked, model.QuestStatusUserReview},
		{model.QuestStatusBlocked, model.QuestStatusCancelled},
	}

	for _, tt := range validPairs {
		t.Run(string(tt.from)+"_to_"+string(tt.to), func(t *testing.T) {
			if !CanTransition(tt.from, tt.to) {
				t.Errorf("CanTransition(%q, %q) = false, want true", tt.from, tt.to)
			}
		})
	}
}

func TestCanTransition_InvalidTransitions(t *testing.T) {
	invalidPairs := []struct {
		from model.QuestStatus
		to   model.QuestStatus
		desc string
	}{
		// 自环不合法
		{model.QuestStatusPending, model.QuestStatusPending, "自环 pending"},
		{model.QuestStatusRunning, model.QuestStatusRunning, "自环 running"},

		// 终态不能迁移出去
		{model.QuestStatusSuccess, model.QuestStatusRunning, "终态 success 不能转出"},
		{model.QuestStatusFailed, model.QuestStatusRunning, "终态 failed 不能转出"},
		{model.QuestStatusCancelled, model.QuestStatusRunning, "终态 cancelled 不能转出"},

		// 跳阶段不合法
		{model.QuestStatusPending, model.QuestStatusReviewing, "pending 不能直接到 reviewing"},
		{model.QuestStatusPending, model.QuestStatusSuccess, "pending 不能直接到 success"},
		{model.QuestStatusRunning, model.QuestStatusSuccess, "running 不能直接到 success"},
		{model.QuestStatusBlocked, model.QuestStatusReviewing, "blocked 不能直接到 reviewing"},
	}

	for _, tt := range invalidPairs {
		t.Run(tt.desc, func(t *testing.T) {
			if CanTransition(tt.from, tt.to) {
				t.Errorf("CanTransition(%q, %q) = true, want false", tt.from, tt.to)
			}
		})
	}
}

// ==================== 阻塞相关迁移测试 ====================

func TestResumeFromBlocked(t *testing.T) {
	result, err := ResumeFromBlocked(model.QuestStatusBlocked, "修复了配置问题", 2)
	if err != nil {
		t.Fatalf("ResumeFromBlocked 返回错误: %v", err)
	}
	if result.NewStatus != model.QuestStatusRunning {
		t.Errorf("NewStatus = %q, want %q", result.NewStatus, model.QuestStatusRunning)
	}
	if result.ReviewHints != "修复了配置问题" {
		t.Errorf("ReviewHints = %q, want %q", result.ReviewHints, "修复了配置问题")
	}
	if result.ResumeCount != 3 {
		t.Errorf("ResumeCount = %d, want 3", result.ResumeCount)
	}
	if !result.ClearBlocked {
		t.Error("ClearBlocked = false, want true")
	}
}

func TestResumeFromBlocked_WrongStatus(t *testing.T) {
	_, err := ResumeFromBlocked(model.QuestStatusRunning, "test", 0)
	if err == nil {
		t.Error("ResumeFromBlocked from running 应该返回错误")
	}
}

func TestMoveBlockedToUserReview(t *testing.T) {
	result, err := MoveBlockedToUserReview(model.QuestStatusBlocked, "需要人工确认")
	if err != nil {
		t.Fatalf("MoveBlockedToUserReview 返回错误: %v", err)
	}
	if result.NewStatus != model.QuestStatusUserReview {
		t.Errorf("NewStatus = %q, want %q", result.NewStatus, model.QuestStatusUserReview)
	}
	if result.FinalComment != "需要人工确认" {
		t.Errorf("FinalComment = %q, want %q", result.FinalComment, "需要人工确认")
	}
	if !result.ClearBlocked {
		t.Error("ClearBlocked = false, want true")
	}
}

func TestMoveBlockedToUserReview_DefaultComment(t *testing.T) {
	result, err := MoveBlockedToUserReview(model.QuestStatusBlocked, "")
	if err != nil {
		t.Fatalf("MoveBlockedToUserReview 返回错误: %v", err)
	}
	if result.FinalComment == "" {
		t.Error("空 comment 应该有默认值")
	}
}

func TestCancelFromBlocked(t *testing.T) {
	result, err := CancelFromBlocked(model.QuestStatusBlocked, "用户放弃")
	if err != nil {
		t.Fatalf("CancelFromBlocked 返回错误: %v", err)
	}
	if result.NewStatus != model.QuestStatusCancelled {
		t.Errorf("NewStatus = %q, want %q", result.NewStatus, model.QuestStatusCancelled)
	}
	if result.FinalComment != "用户放弃" {
		t.Errorf("FinalComment = %q, want %q", result.FinalComment, "用户放弃")
	}
	if !result.SetCompleted {
		t.Error("SetCompleted = false, want true")
	}
}

// ==================== 用户终审相关迁移测试 ====================

func TestCompleteUserReview_Pass(t *testing.T) {
	result, err := CompleteUserReview(model.QuestStatusUserReview, model.VerdictPass, "做得很好")
	if err != nil {
		t.Fatalf("CompleteUserReview(pass) 返回错误: %v", err)
	}
	if result.NewStatus != model.QuestStatusSuccess {
		t.Errorf("NewStatus = %q, want %q", result.NewStatus, model.QuestStatusSuccess)
	}
	if result.FinalVerdict != model.VerdictPass {
		t.Errorf("FinalVerdict = %q, want %q", result.FinalVerdict, model.VerdictPass)
	}
	if result.FinalComment != "做得很好" {
		t.Errorf("FinalComment = %q, want %q", result.FinalComment, "做得很好")
	}
	if !result.SetCompleted {
		t.Error("SetCompleted = false, want true")
	}
}

func TestCompleteUserReview_Reject(t *testing.T) {
	result, err := CompleteUserReview(model.QuestStatusUserReview, model.VerdictReject, "质量不达标")
	if err != nil {
		t.Fatalf("CompleteUserReview(reject) 返回错误: %v", err)
	}
	if result.NewStatus != model.QuestStatusFailed {
		t.Errorf("NewStatus = %q, want %q", result.NewStatus, model.QuestStatusFailed)
	}
	if result.FinalVerdict != model.VerdictReject {
		t.Errorf("FinalVerdict = %q, want %q", result.FinalVerdict, model.VerdictReject)
	}
	if !result.SetCompleted {
		t.Error("SetCompleted = false, want true")
	}
}

func TestCompleteUserReview_InvalidVerdict(t *testing.T) {
	_, err := CompleteUserReview(model.QuestStatusUserReview, "invalid", "test")
	if err == nil {
		t.Error("无效 verdict 应该返回错误")
	}
}

func TestCompleteUserReview_WrongStatus(t *testing.T) {
	_, err := CompleteUserReview(model.QuestStatusRunning, model.VerdictPass, "test")
	if err == nil {
		t.Error("从 running 状态调用 CompleteUserReview 应该返回错误")
	}
}

func TestRequestUserReviewRework(t *testing.T) {
	result, err := RequestUserReviewRework(model.QuestStatusUserReview, "请修复 bug", 1)
	if err != nil {
		t.Fatalf("RequestUserReviewRework 返回错误: %v", err)
	}
	if result.NewStatus != model.QuestStatusRunning {
		t.Errorf("NewStatus = %q, want %q", result.NewStatus, model.QuestStatusRunning)
	}
	if result.ReviewHints != "请修复 bug" {
		t.Errorf("ReviewHints = %q, want %q", result.ReviewHints, "请修复 bug")
	}
	if result.ReworkCount != 2 {
		t.Errorf("ReworkCount = %d, want 2", result.ReworkCount)
	}
}

func TestRequestUserReviewRework_WrongStatus(t *testing.T) {
	_, err := RequestUserReviewRework(model.QuestStatusBlocked, "test", 0)
	if err == nil {
		t.Error("从 blocked 状态调用 RequestUserReviewRework 应该返回错误")
	}
}

// ==================== 通用迁移测试 ====================

func TestCancelQuest(t *testing.T) {
	result, err := CancelQuest(model.QuestStatusRunning, "不需要了")
	if err != nil {
		t.Fatalf("CancelQuest 返回错误: %v", err)
	}
	if result.NewStatus != model.QuestStatusCancelled {
		t.Errorf("NewStatus = %q, want %q", result.NewStatus, model.QuestStatusCancelled)
	}
	if result.FinalComment != "不需要了" {
		t.Errorf("FinalComment = %q, want %q", result.FinalComment, "不需要了")
	}
	if !result.SetCompleted {
		t.Error("SetCompleted = false, want true")
	}
}

func TestCancelQuest_FromTerminal(t *testing.T) {
	_, err := CancelQuest(model.QuestStatusSuccess, "test")
	if err == nil {
		t.Error("从终态取消应该返回错误")
	}
}

func TestBlockQuest(t *testing.T) {
	result, err := BlockQuest(model.QuestStatusRunning, "API 调用超限")
	if err != nil {
		t.Fatalf("BlockQuest 返回错误: %v", err)
	}
	if result.NewStatus != model.QuestStatusBlocked {
		t.Errorf("NewStatus = %q, want %q", result.NewStatus, model.QuestStatusBlocked)
	}
	if result.BlockedReason != "API 调用超限" {
		t.Errorf("BlockedReason = %q, want %q", result.BlockedReason, "API 调用超限")
	}
}

func TestBlockQuest_FromReviewing(t *testing.T) {
	result, err := BlockQuest(model.QuestStatusReviewing, "法师无响应")
	if err != nil {
		t.Fatalf("BlockQuest(reviewing) 返回错误: %v", err)
	}
	if result.NewStatus != model.QuestStatusBlocked {
		t.Errorf("NewStatus = %q, want %q", result.NewStatus, model.QuestStatusBlocked)
	}
	if result.BlockedReason != "法师无响应" {
		t.Errorf("BlockedReason = %q, want %q", result.BlockedReason, "法师无响应")
	}
}

func TestBlockQuest_FromTerminal(t *testing.T) {
	_, err := BlockQuest(model.QuestStatusCancelled, "test")
	if err == nil {
		t.Error("从终态阻塞应该返回错误")
	}
}

// ==================== 法师评审相关迁移测试 ====================

func TestRequestReviewRework(t *testing.T) {
	result, err := RequestReviewRework(model.QuestStatusReviewing, "请修复bug", 1)
	if err != nil {
		t.Fatalf("RequestReviewRework 返回错误: %v", err)
	}
	if result.NewStatus != model.QuestStatusRunning {
		t.Errorf("NewStatus = %q, want %q", result.NewStatus, model.QuestStatusRunning)
	}
	if result.ReviewHints != "请修复bug" {
		t.Errorf("ReviewHints = %q, want %q", result.ReviewHints, "请修复bug")
	}
	if result.ReworkCount != 2 {
		t.Errorf("ReworkCount = %d, want 2", result.ReworkCount)
	}
}

func TestRequestReviewRework_WrongStatus(t *testing.T) {
	_, err := RequestReviewRework(model.QuestStatusBlocked, "test", 0)
	if err == nil {
		t.Error("从 blocked 状态调用 RequestReviewRework 应该返回错误")
	}
}

func TestCompleteReview_Pass(t *testing.T) {
	result, err := CompleteReview(model.QuestStatusReviewing, model.VerdictPass, "做得好")
	if err != nil {
		t.Fatalf("CompleteReview(pass) 返回错误: %v", err)
	}
	if result.NewStatus != model.QuestStatusSuccess {
		t.Errorf("NewStatus = %q, want %q", result.NewStatus, model.QuestStatusSuccess)
	}
	if result.FinalVerdict != model.VerdictPass {
		t.Errorf("FinalVerdict = %q, want %q", result.FinalVerdict, model.VerdictPass)
	}
	if result.FinalComment != "做得好" {
		t.Errorf("FinalComment = %q, want %q", result.FinalComment, "做得好")
	}
	if !result.SetCompleted {
		t.Error("SetCompleted = false, want true")
	}
}

func TestCompleteReview_Reject(t *testing.T) {
	result, err := CompleteReview(model.QuestStatusReviewing, model.VerdictReject, "质量差")
	if err != nil {
		t.Fatalf("CompleteReview(reject) 返回错误: %v", err)
	}
	if result.NewStatus != model.QuestStatusFailed {
		t.Errorf("NewStatus = %q, want %q", result.NewStatus, model.QuestStatusFailed)
	}
	if result.FinalVerdict != model.VerdictReject {
		t.Errorf("FinalVerdict = %q, want %q", result.FinalVerdict, model.VerdictReject)
	}
	if !result.SetCompleted {
		t.Error("SetCompleted = false, want true")
	}
}

func TestCompleteReview_InvalidVerdict(t *testing.T) {
	_, err := CompleteReview(model.QuestStatusReviewing, "invalid", "test")
	if err == nil {
		t.Error("无效 verdict 应该返回错误")
	}
}

func TestCompleteReview_WrongStatus(t *testing.T) {
	_, err := CompleteReview(model.QuestStatusRunning, model.VerdictPass, "test")
	if err == nil {
		t.Error("从 running 状态调用 CompleteReview 应该返回错误")
	}
}

// ==================== QuestMeta 领域方法测试 ====================

func TestQuestMeta_IsTerminal(t *testing.T) {
	q := &QuestMeta{Status: model.QuestStatusRunning}
	if q.IsTerminal() {
		t.Error("running 状态不应为终态")
	}

	q.Status = model.QuestStatusSuccess
	if !q.IsTerminal() {
		t.Error("success 状态应为终态")
	}
}

func TestQuestMeta_CanTransitionTo(t *testing.T) {
	q := &QuestMeta{Status: model.QuestStatusBlocked}
	if !q.CanTransitionTo(model.QuestStatusRunning) {
		t.Error("blocked 应能迁移到 running")
	}
	if q.CanTransitionTo(model.QuestStatusReviewing) {
		t.Error("blocked 不能迁移到 reviewing")
	}
}

func TestQuestMeta_ApplyTransitionResult(t *testing.T) {
	q := &QuestMeta{
		Status:        model.QuestStatusBlocked,
		BlockedReason: "旧的阻塞原因",
		ResumeCount:   2,
	}

	result := TransitionResult{
		NewStatus:    model.QuestStatusRunning,
		ReviewHints:  "修复完成",
		ResumeCount:  3,
		ClearBlocked: true,
	}

	q.ApplyTransitionResult(result)

	if q.Status != model.QuestStatusRunning {
		t.Errorf("Status = %q, want %q", q.Status, model.QuestStatusRunning)
	}
	if q.ReviewHints != "修复完成" {
		t.Errorf("ReviewHints = %q, want %q", q.ReviewHints, "修复完成")
	}
	if q.ResumeCount != 3 {
		t.Errorf("ResumeCount = %d, want 3", q.ResumeCount)
	}
	if q.BlockedReason != "" {
		t.Errorf("BlockedReason = %q, want 空", q.BlockedReason)
	}
}

func TestQuestMeta_ApplyTransitionResult_Completed(t *testing.T) {
	q := &QuestMeta{
		Status:       model.QuestStatusUserReview,
		FinalVerdict: "",
		FinalComment: "",
	}

	result := TransitionResult{
		NewStatus:    model.QuestStatusSuccess,
		FinalVerdict: model.VerdictPass,
		FinalComment: "完美",
		SetCompleted: true,
	}

	q.ApplyTransitionResult(result)

	if q.Status != model.QuestStatusSuccess {
		t.Errorf("Status = %q, want %q", q.Status, model.QuestStatusSuccess)
	}
	if q.FinalVerdict != model.VerdictPass {
		t.Errorf("FinalVerdict = %q, want %q", q.FinalVerdict, model.VerdictPass)
	}
	if q.FinalComment != "完美" {
		t.Errorf("FinalComment = %q, want %q", q.FinalComment, "完美")
	}
	if q.CompletedAtMs == 0 {
		t.Error("CompletedAtMs 应该被设置")
	}
}

func TestQuestMeta_CanRework(t *testing.T) {
	tests := []struct {
		name        string
		reworkCount int
		maxRework   int
		want        bool
	}{
		{"还可以返工", 0, 3, true},
		{"即将达到上限", 2, 3, true},
		{"刚达到上限", 3, 3, false},
		{"超过上限", 4, 3, false},
		{"零次返工机会", 0, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &QuestMeta{ReworkCount: tt.reworkCount, MaxRework: tt.maxRework}
			if got := q.CanRework(); got != tt.want {
				t.Errorf("CanRework() = %v, want %v (rework=%d, max=%d)", got, tt.want, tt.reworkCount, tt.maxRework)
			}
		})
	}
}

func TestQuestMeta_Source(t *testing.T) {
	tests := []struct {
		name      string
		createdBy string
		want      model.QuestSource
	}{
		{"用户创建", "user", model.SourceUser},
		{"自动化创建", "automation:auto_123", model.SourceAutomation},
		{"空", "", model.SourceUser},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &QuestMeta{CreatedBy: tt.createdBy}
			if got := q.Source(); got != tt.want {
				t.Errorf("Source() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsAutomationQuest(t *testing.T) {
	if !IsAutomationQuest("automation:auto_1") {
		t.Error("automation 创建的应该返回 true")
	}

	if IsAutomationQuest("user") {
		t.Error("user 创建的应该返回 false")
	}
}
