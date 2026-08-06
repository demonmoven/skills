package quest

import (
	"errors"
	"testing"

	"code.byted.org/lihuanyu.0w0/gloop/internal/events"
	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

// ==================== Phase 管道集成测试 ====================
//
// 验证 QuestService 中所有状态迁移操作时，
// phase 管道（Phases 列表、CurrentPhaseIdx、各阶段状态）与 quest 状态保持一致。
//
// 这是 Phase 2 的核心回归保障：每次状态变更都正确同步 phase 管道。

// ===== 测试辅助：内存 Repository =====

type memRepo struct {
	quests  map[string]*QuestMeta
	reviews map[string][]*ReviewRecord
}

func newMemRepo() *memRepo {
	return &memRepo{
		quests:  make(map[string]*QuestMeta),
		reviews: make(map[string][]*ReviewRecord),
	}
}

func (r *memRepo) Load(id string) (*QuestMeta, error) {
	q, ok := r.quests[id]
	if !ok {
		return nil, errors.New("quest not found")
	}
	// 返回副本，避免外部修改影响存储
	cp := *q
	return &cp, nil
}

func (r *memRepo) Save(q *QuestMeta) error {
	cp := *q
	r.quests[q.ID] = &cp
	return nil
}

func (r *memRepo) Delete(id string) error {
	delete(r.quests, id)
	return nil
}

func (r *memRepo) List() ([]*QuestMeta, error) {
	out := make([]*QuestMeta, 0, len(r.quests))
	for _, q := range r.quests {
		cp := *q
		out = append(out, &cp)
	}
	return out, nil
}

func (r *memRepo) ListByStatus(status model.QuestStatus) ([]*QuestMeta, error) {
	var out []*QuestMeta
	for _, q := range r.quests {
		if q.Status == status {
			cp := *q
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (r *memRepo) AppendEvent(qid string, eventType model.EventType, payload any) error {
	return nil
}

func (r *memRepo) AppendReview(qid string, review *ReviewRecord) error {
	r.reviews[qid] = append(r.reviews[qid], review)
	return nil
}

func (r *memRepo) LoadReviews(qid string) ([]*ReviewRecord, error) {
	return r.reviews[qid], nil
}

func (r *memRepo) WorkspacePath(qid string) string {
	return "/tmp/ws/" + qid
}

func (r *memRepo) CountByStatus() (map[model.QuestStatus]int, error) {
	m := make(map[model.QuestStatus]int)
	for _, q := range r.quests {
		m[q.Status]++
	}
	return m, nil
}

// ===== 测试工具 =====

func newTestService() (*Service, *memRepo) {
	repo := newMemRepo()
	svc := NewService(repo, nil)
	svc.nowMs = func() int64 { return 1000 }
	return svc, repo
}

func makePendingQuest(id string) *QuestMeta {
	return &QuestMeta{
		ID:          id,
		ShortID:     id,
		Query:       "test quest",
		Type:        model.QuestTypeExecute,
		Status:      model.QuestStatusPending,
		WarriorID:   "warrior_001",
		MageID:      "mage_001",
		MaxRework:   3,
		CreatedBy:   "user",
		CreatedAtMs: 500,
	}
}

// 断言 phase 数量
func assertPhaseCount(t *testing.T, q *QuestMeta, want int) {
	t.Helper()
	if len(q.Phases) != want {
		t.Fatalf("phase 数量 = %d, want %d", len(q.Phases), want)
	}
}

// 断言当前阶段索引
func assertCurrentPhase(t *testing.T, q *QuestMeta, wantIdx int) {
	t.Helper()
	if q.CurrentPhaseIdx != wantIdx {
		t.Fatalf("CurrentPhaseIdx = %d, want %d", q.CurrentPhaseIdx, wantIdx)
	}
}

// 断言指定阶段状态
func assertPhaseStatus(t *testing.T, q *QuestMeta, idx int, want PhaseStatus) {
	t.Helper()
	if idx >= len(q.Phases) {
		t.Fatalf("phase %d 不存在（共 %d 个）", idx, len(q.Phases))
	}
	if q.Phases[idx].Status != want {
		t.Fatalf("phase[%d] 状态 = %q, want %q", idx, q.Phases[idx].Status, want)
	}
}

// 断言指定阶段冒险者
func assertPhaseAdventurer(t *testing.T, q *QuestMeta, idx int, want string) {
	t.Helper()
	if idx >= len(q.Phases) {
		t.Fatalf("phase %d 不存在", idx)
	}
	if q.Phases[idx].AdventurerID != want {
		t.Fatalf("phase[%d] 冒险者 = %q, want %q", idx, q.Phases[idx].AdventurerID, want)
	}
}

// ===== 测试用例 =====

func TestPhasePipeline_StartQuest(t *testing.T) {
	svc, repo := newTestService()
	q := makePendingQuest("q1")
	repo.Save(q)

	result, err := svc.StartQuest("q1", 3)
	if err != nil {
		t.Fatalf("StartQuest 失败: %v", err)
	}

	// 验证状态
	if result.Status != model.QuestStatusRunning {
		t.Errorf("status = %q, want running", result.Status)
	}

	// 验证 phase 管道
	assertPhaseCount(t, result, 2)
	assertCurrentPhase(t, result, 0)
	assertPhaseStatus(t, result, 0, PhaseRunning)
	assertPhaseStatus(t, result, 1, PhasePending)
	assertPhaseAdventurer(t, result, 0, "warrior_001")
	assertPhaseAdventurer(t, result, 1, "mage_001")

	// 验证 phase 0 已启动
	if result.Phases[0].StartedAtMs == 0 {
		t.Error("phase 0 StartedAtMs 不应为 0")
	}
}

func TestPhasePipeline_StartReviewPhase(t *testing.T) {
	svc, repo := newTestService()
	q := makePendingQuest("q1")
	repo.Save(q)
	svc.StartQuest("q1", 3)

	// 进入评审阶段
	result, err := svc.StartReviewPhase("q1")
	if err != nil {
		t.Fatalf("StartReviewPhase 失败: %v", err)
	}

	if result.Status != model.QuestStatusReviewing {
		t.Errorf("status = %q, want reviewing", result.Status)
	}

	// phase 0 完成，phase 1 运行中
	assertPhaseCount(t, result, 2)
	assertCurrentPhase(t, result, 1)
	assertPhaseStatus(t, result, 0, PhaseDone)
	assertPhaseStatus(t, result, 1, PhaseRunning)

	// 验证时间戳
	if result.Phases[0].EndedAtMs == 0 {
		t.Error("phase 0 EndedAtMs 不应为 0")
	}
	if result.Phases[1].StartedAtMs == 0 {
		t.Error("phase 1 StartedAtMs 不应为 0")
	}
}

func TestPhasePipeline_CompletePhase(t *testing.T) {
	svc, repo := newTestService()
	q := makePendingQuest("q_complete_phase")
	repo.Save(q)
	svc.StartQuest(q.ID, 3)

	result, err := svc.CompletePhase(q.ID, CompletePhaseOptions{PhaseIdx: 0})
	if err != nil {
		t.Fatalf("CompletePhase failed: %v", err)
	}
	if result.Status != model.QuestStatusReviewing {
		t.Fatalf("status = %s, want reviewing", result.Status)
	}
	assertCurrentPhase(t, result, 1)
	assertPhaseStatus(t, result, 0, PhaseDone)
	assertPhaseStatus(t, result, 1, PhaseRunning)
}

func TestPhasePipeline_CompletePhaseAdvancesAdditionalReviewPhase(t *testing.T) {
	svc, repo := newTestService()
	q := makePendingQuest("q_complete_phase_2")
	q.Pipeline = Pipeline{
		DefaultPipeline()[0],
		DefaultPipeline()[1],
		{
			Index:       2,
			Name:        "mage_review_2",
			DisplayName: "法师复审",
			Role:        PhaseRoleReview,
			Class:       model.ClassMage,
			ReadOnly:    true,
			EndSignal:   "review_quest",
			ReworkTo:    0,
		},
	}
	q.Phases = []PhaseRun{
		{PhaseIdx: 0, Status: PhaseDone},
		{PhaseIdx: 1, Status: PhaseRunning},
		{PhaseIdx: 2, Status: PhasePending},
	}
	q.Status = model.QuestStatusReviewing
	q.CurrentPhaseIdx = 1
	repo.Save(q)

	result, err := svc.CompletePhase(q.ID, CompletePhaseOptions{PhaseIdx: 1})
	if err != nil {
		t.Fatalf("CompletePhase phase 1 failed: %v", err)
	}
	if result.Status != model.QuestStatusReviewing {
		t.Fatalf("status = %s, want reviewing", result.Status)
	}
	assertCurrentPhase(t, result, 2)
	assertPhaseStatus(t, result, 1, PhaseDone)
	assertPhaseStatus(t, result, 2, PhaseRunning)
}

func TestPhasePipeline_RequestReviewRework(t *testing.T) {
	svc, repo := newTestService()
	q := makePendingQuest("q1")
	repo.Save(q)
	svc.StartQuest("q1", 3)
	svc.StartReviewPhase("q1")

	// 法师要求返工
	result, err := svc.RequestReviewRework("q1", "请修复 bug")
	if err != nil {
		t.Fatalf("RequestReviewRework 失败: %v", err)
	}

	if result.Status != model.QuestStatusRunning {
		t.Errorf("status = %q, want running", result.Status)
	}
	if result.ReworkCount != 1 {
		t.Errorf("rework_count = %d, want 1", result.ReworkCount)
	}

	// 回到 phase 0，phase 0 重新运行，phase 1 被重置
	assertCurrentPhase(t, result, 0)
	assertPhaseStatus(t, result, 0, PhaseRunning)
	assertPhaseStatus(t, result, 1, PhasePending)

	// phase 0 的 ReworkCount 应该增加（返工了一次）
	if result.Phases[0].ReworkCount < 1 {
		t.Errorf("phase 0 rework_count = %d, 应 >= 1", result.Phases[0].ReworkCount)
	}

	// phase 1 应该被重置（session 清空）
	if result.Phases[1].SessionID != "" {
		t.Error("phase 1 重置后 SessionID 应为空")
	}
}

func TestPhasePipeline_RequestReworkToPhase(t *testing.T) {
	svc, repo := newTestService()
	q := makePendingQuest("q_rework_to_phase")
	repo.Save(q)
	svc.StartQuest(q.ID, 3)
	svc.StartReviewPhase(q.ID)

	result, err := svc.RequestReworkToPhase(q.ID, RequestReworkToPhaseOptions{TargetPhaseIdx: 0, Hints: "fix"})
	if err != nil {
		t.Fatalf("RequestReworkToPhase failed: %v", err)
	}
	if result.Status != model.QuestStatusRunning || result.ReworkCount != 1 {
		t.Fatalf("bad rework result: %+v", result)
	}
	assertCurrentPhase(t, result, 0)
}

func TestPhasePipeline_RequestReworkToPhase_TargetsExecutePhase(t *testing.T) {
	svc, repo := newTestService()
	q := makePendingQuest("q_rework_to_execute_phase")
	q.Pipeline = DesignPipeline()
	q.Phases = []PhaseRun{
		{PhaseIdx: 0, Status: PhaseDone, SessionID: "warrior_design_0"},
		{PhaseIdx: 1, Status: PhaseDone, SessionID: "mage_design_review_0"},
		{PhaseIdx: 2, Status: PhaseDone, SessionID: "warrior_execute_0"},
		{PhaseIdx: 3, Status: PhaseRunning, SessionID: "mage_implementation_review_0"},
	}
	q.Status = model.QuestStatusReviewing
	q.CurrentPhaseIdx = 3
	repo.Save(q)

	result, err := svc.RequestReworkToPhase(q.ID, RequestReworkToPhaseOptions{TargetPhaseIdx: 2, Hints: "fix implementation"})
	if err != nil {
		t.Fatalf("RequestReworkToPhase failed: %v", err)
	}
	if result.Status != model.QuestStatusRunning || result.ReworkCount != 1 {
		t.Fatalf("bad rework result: %+v", result)
	}
	assertCurrentPhase(t, result, 2)
	assertPhaseStatus(t, result, 0, PhaseDone)
	assertPhaseStatus(t, result, 1, PhaseDone)
	assertPhaseStatus(t, result, 2, PhaseRunning)
	assertPhaseStatus(t, result, 3, PhasePending)
	if result.Phases[0].SessionID == "" || result.Phases[1].SessionID == "" {
		t.Fatalf("earlier design phases should be preserved: %+v", result.Phases)
	}
	if result.Phases[2].SessionID != "" || result.Phases[3].SessionID != "" {
		t.Fatalf("target and later phases should be reset: %+v", result.Phases)
	}
}

func TestPhasePipeline_RequestReworkToPhase_RejectsReviewTarget(t *testing.T) {
	svc, repo := newTestService()
	q := makePendingQuest("q_rework_to_review_phase")
	q.Pipeline = DesignPipeline()
	q.Phases = []PhaseRun{
		{PhaseIdx: 0, Status: PhaseDone},
		{PhaseIdx: 1, Status: PhaseRunning},
		{PhaseIdx: 2, Status: PhasePending},
		{PhaseIdx: 3, Status: PhasePending},
	}
	q.Status = model.QuestStatusReviewing
	q.CurrentPhaseIdx = 1
	repo.Save(q)

	if _, err := svc.RequestReworkToPhase(q.ID, RequestReworkToPhaseOptions{TargetPhaseIdx: 1, Hints: "bad target"}); err == nil {
		t.Fatal("RequestReworkToPhase should reject review target")
	}
}

func TestPhasePipeline_CompleteQuest(t *testing.T) {
	svc, repo := newTestService()
	q := makePendingQuest("q_complete_quest")
	repo.Save(q)
	svc.StartQuest(q.ID, 3)
	svc.StartReviewPhase(q.ID)

	result, err := svc.CompleteQuest(q.ID, CompleteQuestOptions{Verdict: model.VerdictPass, Comment: "done"})
	if err != nil {
		t.Fatalf("CompleteQuest failed: %v", err)
	}
	if result.Status != model.QuestStatusSuccess || result.FinalVerdict != model.VerdictPass || result.FinalComment != "done" {
		t.Fatalf("bad complete quest result: %+v", result)
	}
}

func TestPhasePipeline_CompleteQuestPersistsPolicyTrace(t *testing.T) {
	svc, repo := newTestService()
	q := makePendingQuest("q_policy_complete")
	repo.Save(q)
	svc.StartQuest(q.ID, 3)
	svc.StartReviewPhase(q.ID)

	result, err := svc.CompleteQuest(q.ID, CompleteQuestOptions{
		Verdict:            model.VerdictPass,
		Comment:            "done",
		FinalizedBy:        "policy",
		AutoPassedByPolicy: "context_store_automation_auto_pass",
		PolicyDecisionID:   "pol_dec_test",
	})
	if err != nil {
		t.Fatalf("CompleteQuest failed: %v", err)
	}
	if result.FinalizedBy != "policy" || result.AutoPassedByPolicy != "context_store_automation_auto_pass" || result.PolicyDecisionID != "pol_dec_test" {
		t.Fatalf("policy trace fields not persisted: %+v", result)
	}
}

func TestPhasePipeline_CompleteExecutePersistsAutoCompleteTrace(t *testing.T) {
	svc, repo := newTestService()
	q := makePendingQuest("q_auto_complete")
	repo.Save(q)
	svc.StartQuest(q.ID, 3)

	result, err := svc.CompleteExecute(q.ID, CompleteExecuteOptions{
		Comment:               "quick report done",
		FinalizedBy:           "policy",
		AutoCompletedByPolicy: "quick_no_effect_auto_complete",
		PolicyDecisionID:      "pol_dec_quick",
	})
	if err != nil {
		t.Fatalf("CompleteExecute failed: %v", err)
	}
	if result.Status != model.QuestStatusSuccess || result.FinalVerdict != model.VerdictPass || result.FinalComment != "quick report done" {
		t.Fatalf("bad complete execute result: %+v", result)
	}
	if result.FinalizedBy != "policy" || result.AutoCompletedByPolicy != "quick_no_effect_auto_complete" || result.AutoPassedByPolicy != "" || result.PolicyDecisionID != "pol_dec_quick" {
		t.Fatalf("auto-complete trace fields not persisted: %+v", result)
	}
	assertPhaseStatus(t, result, 0, PhaseDone)
}

func TestPhasePipeline_MoveToUserReview(t *testing.T) {
	svc, repo := newTestService()
	q := makePendingQuest("q1")
	repo.Save(q)
	svc.StartQuest("q1", 3)
	svc.StartReviewPhase("q1")

	// 转入用户终审
	result, err := svc.MoveToUserReview("q1", MoveToUserReviewOptions{
		Verdict:   model.VerdictPass,
		Comment:   "做得不错，用户确认下",
		FromPhase: "mage_review",
	})
	if err != nil {
		t.Fatalf("MoveToUserReview 失败: %v", err)
	}

	if result.Status != model.QuestStatusUserReview {
		t.Errorf("status = %q, want user_review", result.Status)
	}

	// phase 1（法师评审）已完成
	assertPhaseStatus(t, result, 0, PhaseDone)
	assertPhaseStatus(t, result, 1, PhaseDone)
}

func TestPhasePipeline_ResolveUserReview_Pass(t *testing.T) {
	svc, repo := newTestService()
	q := makePendingQuest("q1")
	repo.Save(q)
	svc.StartQuest("q1", 3)
	svc.StartReviewPhase("q1")
	svc.MoveToUserReview("q1", MoveToUserReviewOptions{Verdict: model.VerdictPass})

	// 用户通过
	result, err := svc.ResolveUserReview("q1", ResolveUserReviewOptions{
		Verdict: model.VerdictPass,
		Comment: "通过",
	})
	if err != nil {
		t.Fatalf("ResolveUserReview(pass) 失败: %v", err)
	}

	if result.Status != model.QuestStatusSuccess {
		t.Errorf("status = %q, want success", result.Status)
	}
	if result.FinalizedBy != "user" {
		t.Errorf("finalized_by = %q, want user", result.FinalizedBy)
	}

	// 所有 phase 都应为 done
	assertPhaseStatus(t, result, 0, PhaseDone)
	assertPhaseStatus(t, result, 1, PhaseDone)
}

func TestPhasePipeline_ResolveUserReviewRecordsSource(t *testing.T) {
	svc, repo := newTestService()
	q := makePendingQuest("q1")
	repo.Save(q)
	svc.StartQuest("q1", 3)
	svc.StartReviewPhase("q1")
	svc.MoveToUserReview("q1", MoveToUserReviewOptions{Verdict: model.VerdictPass})

	_, err := svc.ResolveUserReview("q1", ResolveUserReviewOptions{
		Verdict:     model.VerdictPass,
		Comment:     "通过",
		ActorUserID: "user-123",
	})
	if err != nil {
		t.Fatalf("ResolveUserReview(pass) 失败: %v", err)
	}
	reviews, err := repo.LoadReviews("q1")
	if err != nil {
		t.Fatalf("LoadReviews failed: %v", err)
	}
	if len(reviews) != 1 {
		t.Fatalf("reviews = %d, want 1", len(reviews))
	}
	src := reviews[0].Source
	if src == nil || src.SourceRole != "user" || src.SourceAdventurerID != "user-123" || src.SourcePhaseIdx != -1 {
		t.Fatalf("bad review source: %+v", src)
	}
}

func TestPhasePipeline_CompleteReviewRecordsMageSource(t *testing.T) {
	svc, repo := newTestService()
	q := makePendingQuest("q1")
	repo.Save(q)
	svc.StartQuest("q1", 3)
	svc.StartReviewPhase("q1")

	phaseIdx := 1
	_, err := svc.CompleteReview("q1", CompleteReviewOptions{
		Verdict:      model.VerdictPass,
		Comment:      "ok",
		AdventurerID: "mage-123",
		PhaseIdx:     &phaseIdx,
		SessionID:    "mage_0",
	})
	if err != nil {
		t.Fatalf("CompleteReview(pass) failed: %v", err)
	}
	reviews, err := repo.LoadReviews("q1")
	if err != nil {
		t.Fatalf("LoadReviews failed: %v", err)
	}
	if len(reviews) != 1 {
		t.Fatalf("reviews = %d, want 1", len(reviews))
	}
	src := reviews[0].Source
	if src == nil || src.SourceRole != "mage" || src.SourceClass != "mage" ||
		src.SourceAdventurerID != "mage-123" || src.SourcePhaseIdx != 1 ||
		src.SourceSessionID != "mage_0" || src.SourceIncomplete {
		t.Fatalf("bad mage review source: %+v", src)
	}
}

func TestPhasePipeline_ResolveUserReview_Reject(t *testing.T) {
	svc, repo := newTestService()
	q := makePendingQuest("q1")
	repo.Save(q)
	svc.StartQuest("q1", 3)
	svc.StartReviewPhase("q1")
	svc.MoveToUserReview("q1", MoveToUserReviewOptions{Verdict: model.VerdictPass})

	// 用户拒绝
	result, err := svc.ResolveUserReview("q1", ResolveUserReviewOptions{
		Verdict: model.VerdictReject,
		Comment: "质量差",
	})
	if err != nil {
		t.Fatalf("ResolveUserReview(reject) 失败: %v", err)
	}

	if result.Status != model.QuestStatusFailed {
		t.Errorf("status = %q, want failed", result.Status)
	}

	// 所有非终态 phase 都应标记为 done（终态处理）
	assertPhaseStatus(t, result, 0, PhaseDone)
	assertPhaseStatus(t, result, 1, PhaseDone)
}

func TestPhasePipeline_ResolveUserReview_RequestChanges(t *testing.T) {
	svc, repo := newTestService()
	q := makePendingQuest("q1")
	repo.Save(q)
	svc.StartQuest("q1", 3)
	svc.StartReviewPhase("q1")
	svc.MoveToUserReview("q1", MoveToUserReviewOptions{Verdict: model.VerdictPass})

	// 用户要求返工
	result, err := svc.ResolveUserReview("q1", ResolveUserReviewOptions{
		Verdict: model.VerdictRequestChange,
		Comment: "需要调整",
	})
	if err != nil {
		t.Fatalf("ResolveUserReview(request_changes) 失败: %v", err)
	}

	if result.Status != model.QuestStatusRunning {
		t.Errorf("status = %q, want running", result.Status)
	}
	if result.ReworkCount != 1 {
		t.Errorf("rework_count = %d, want 1", result.ReworkCount)
	}

	// 回到 phase 0，重新运行
	assertCurrentPhase(t, result, 0)
	assertPhaseStatus(t, result, 0, PhaseRunning)
	assertPhaseStatus(t, result, 1, PhasePending)
}

func TestPhasePipeline_CancelQuest_FromRunning(t *testing.T) {
	svc, repo := newTestService()
	q := makePendingQuest("q1")
	repo.Save(q)
	svc.StartQuest("q1", 3)

	result, err := svc.CancelQuest("q1", "不需要了")
	if err != nil {
		t.Fatalf("CancelQuest 失败: %v", err)
	}

	if result.Status != model.QuestStatusCancelled {
		t.Errorf("status = %q, want cancelled", result.Status)
	}

	// 所有未完成 phase 标记为 failed
	assertPhaseStatus(t, result, 0, PhaseFailed)
	assertPhaseStatus(t, result, 1, PhaseFailed)
}

func TestPhasePipeline_CancelQuest_FromReviewing(t *testing.T) {
	svc, repo := newTestService()
	q := makePendingQuest("q1")
	repo.Save(q)
	svc.StartQuest("q1", 3)
	svc.StartReviewPhase("q1")

	result, err := svc.CancelQuest("q1", "放弃了")
	if err != nil {
		t.Fatalf("CancelQuest 失败: %v", err)
	}

	if result.Status != model.QuestStatusCancelled {
		t.Errorf("status = %q, want cancelled", result.Status)
	}

	// phase 0 已经 done（保留），phase 1 失败
	assertPhaseStatus(t, result, 0, PhaseDone)
	assertPhaseStatus(t, result, 1, PhaseFailed)
}

func TestPhasePipeline_FailQuest(t *testing.T) {
	svc, repo := newTestService()
	q := makePendingQuest("q1")
	repo.Save(q)
	svc.StartQuest("q1", 3)

	result, err := svc.FailQuest("q1", "系统错误")
	if err != nil {
		t.Fatalf("FailQuest 失败: %v", err)
	}

	if result.Status != model.QuestStatusFailed {
		t.Errorf("status = %q, want failed", result.Status)
	}

	// 所有未完成 phase 标记为 failed
	assertPhaseStatus(t, result, 0, PhaseFailed)
	assertPhaseStatus(t, result, 1, PhaseFailed)
}

func TestPhasePipeline_MultipleReworks(t *testing.T) {
	svc, repo := newTestService()
	q := makePendingQuest("q1")
	repo.Save(q)
	svc.StartQuest("q1", 3)

	// 第一次返工（法师要求）
	svc.StartReviewPhase("q1")
	result, err := svc.RequestReviewRework("q1", "第一次返工")
	if err != nil {
		t.Fatalf("第一次返工失败: %v", err)
	}
	if result.ReworkCount != 1 {
		t.Errorf("rework_count = %d, want 1", result.ReworkCount)
	}

	// 第二次返工（用户终审要求）
	svc.StartReviewPhase("q1")
	svc.MoveToUserReview("q1", MoveToUserReviewOptions{Verdict: model.VerdictPass})
	result, err = svc.ResolveUserReview("q1", ResolveUserReviewOptions{
		Verdict: model.VerdictRequestChange,
		Comment: "第二次返工",
	})
	if err != nil {
		t.Fatalf("第二次返工失败: %v", err)
	}
	if result.ReworkCount != 2 {
		t.Errorf("rework_count = %d, want 2", result.ReworkCount)
	}

	assertCurrentPhase(t, result, 0)
	assertPhaseStatus(t, result, 0, PhaseRunning)
	assertPhaseStatus(t, result, 1, PhasePending)
}

func TestPhasePipeline_PhasesConsistentAfterEachTransition(t *testing.T) {
	// 完整生命周期走一遍，每一步都验证 phase 状态
	svc, repo := newTestService()
	q := makePendingQuest("q_full")
	repo.Save(q)

	// Step 1: StartQuest
	q, _ = svc.StartQuest("q_full", 3)
	assertPhaseCount(t, q, 2)
	assertCurrentPhase(t, q, 0)
	assertPhaseStatus(t, q, 0, PhaseRunning)
	assertPhaseStatus(t, q, 1, PhasePending)

	// Step 2: StartReviewPhase
	q, _ = svc.StartReviewPhase("q_full")
	assertCurrentPhase(t, q, 1)
	assertPhaseStatus(t, q, 0, PhaseDone)
	assertPhaseStatus(t, q, 1, PhaseRunning)

	// Step 3: RequestReviewRework
	q, _ = svc.RequestReviewRework("q_full", "返工")
	assertCurrentPhase(t, q, 0)
	assertPhaseStatus(t, q, 0, PhaseRunning)
	assertPhaseStatus(t, q, 1, PhasePending)

	// Step 4: StartReviewPhase  again
	q, _ = svc.StartReviewPhase("q_full")
	assertCurrentPhase(t, q, 1)
	assertPhaseStatus(t, q, 0, PhaseDone)
	assertPhaseStatus(t, q, 1, PhaseRunning)

	// Step 5: MoveToUserReview
	q, _ = svc.MoveToUserReview("q_full", MoveToUserReviewOptions{Verdict: model.VerdictPass})
	assertPhaseStatus(t, q, 0, PhaseDone)
	assertPhaseStatus(t, q, 1, PhaseDone)

	// Step 6: ResolveUserReview pass
	q, _ = svc.ResolveUserReview("q_full", ResolveUserReviewOptions{Verdict: model.VerdictPass})
	if q.Status != model.QuestStatusSuccess {
		t.Fatalf("最终状态 = %q, want success", q.Status)
	}
	assertPhaseStatus(t, q, 0, PhaseDone)
	assertPhaseStatus(t, q, 1, PhaseDone)
}

// ===== PhaseRun 单元测试 =====

func TestPhaseRun_Start(t *testing.T) {
	p := &PhaseRun{PhaseIdx: 0}
	p.Start(1000)
	if p.Status != PhaseRunning {
		t.Errorf("status = %q, want running", p.Status)
	}
	if p.StartedAtMs != 1000 {
		t.Errorf("started_at = %d, want 1000", p.StartedAtMs)
	}

	// 再次调用不改变 StartedAtMs（幂等启动时间）
	p.Start(2000)
	if p.StartedAtMs != 1000 {
		t.Errorf("started_at 被覆盖了: %d, 应保持 1000", p.StartedAtMs)
	}
}

func TestPhaseRun_Complete(t *testing.T) {
	p := &PhaseRun{PhaseIdx: 0}
	p.Start(1000)
	p.Complete(2000)
	if p.Status != PhaseDone {
		t.Errorf("status = %q, want done", p.Status)
	}
	if p.EndedAtMs != 2000 {
		t.Errorf("ended_at = %d, want 2000", p.EndedAtMs)
	}
}

func TestPhaseRun_Fail(t *testing.T) {
	p := &PhaseRun{PhaseIdx: 0}
	p.Start(1000)
	p.Fail(2000)
	if p.Status != PhaseFailed {
		t.Errorf("status = %q, want failed", p.Status)
	}
	if p.EndedAtMs != 2000 {
		t.Errorf("ended_at = %d, want 2000", p.EndedAtMs)
	}
}

func TestPhaseRun_IsTerminal(t *testing.T) {
	p := &PhaseRun{}
	if p.IsTerminal() {
		t.Error("pending 不应为终态")
	}

	p.Status = PhaseRunning
	if p.IsTerminal() {
		t.Error("running 不应为终态")
	}

	p.Status = PhaseDone
	if !p.IsTerminal() {
		t.Error("done 应为终态")
	}

	p.Status = PhaseFailed
	if !p.IsTerminal() {
		t.Error("failed 应为终态")
	}
}

func TestPhaseRun_Reset(t *testing.T) {
	p := &PhaseRun{
		PhaseIdx:     0,
		AdventurerID: "adv_001",
		SessionID:    "sess_123",
		Status:       PhaseDone,
		Turns:        10,
		ReworkCount:  0,
		StartedAtMs:  1000,
		EndedAtMs:    2000,
	}

	p.Reset()

	if p.Status != PhasePending {
		t.Errorf("status = %q, want pending", p.Status)
	}
	if p.SessionID != "" {
		t.Errorf("session_id 应为空, got %q", p.SessionID)
	}
	if p.Turns != 0 {
		t.Errorf("turns = %d, want 0", p.Turns)
	}
	if p.StartedAtMs != 0 {
		t.Errorf("started_at = %d, want 0", p.StartedAtMs)
	}
	if p.EndedAtMs != 0 {
		t.Errorf("ended_at = %d, want 0", p.EndedAtMs)
	}
	if p.ReworkCount != 1 {
		t.Errorf("rework_count = %d, want 1", p.ReworkCount)
	}
	// AdventurerID 应该保留（角色不变）
	if p.AdventurerID != "adv_001" {
		t.Errorf("adventurer_id 应该保留, got %q", p.AdventurerID)
	}
}

// ===== 空事件发布器（验证事件发布不会 panic） =====

type captureEvents struct {
	events []capturedEvent
}

type capturedEvent struct {
	qid       string
	sessionID string
	typ       events.EventType
	payload   map[string]any
}

func (c *captureEvents) Publish(qid, sessionID string, typ events.EventType, payload map[string]any) {
	c.events = append(c.events, capturedEvent{
		qid:       qid,
		sessionID: sessionID,
		typ:       typ,
		payload:   payload,
	})
}

func TestPhasePipeline_EventsPublished(t *testing.T) {
	repo := newMemRepo()
	evts := &captureEvents{}
	svc := NewService(repo, evts)
	svc.nowMs = func() int64 { return 1000 }

	q := makePendingQuest("q_evt")
	repo.Save(q)

	svc.StartQuest("q_evt", 3)
	svc.StartReviewPhase("q_evt")

	// 验证至少发布了 quest_started 和 phase_changed 事件
	eventTypes := make(map[events.EventType]bool)
	for _, e := range evts.events {
		eventTypes[e.typ] = true
	}

	if !eventTypes[events.EvtQuestStarted] {
		t.Error("应该发布 quest_started 事件")
	}
	if !eventTypes[events.EvtPhaseChanged] {
		t.Error("应该发布 phase_changed 事件")
	}
}
