package orchestrator

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"code.byted.org/lihuanyu.0w0/gloop/internal/executor"
	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

type scriptedLoopExecutor struct {
	*executor.MockExecutor
	mu             sync.Mutex
	reviewVerdicts []model.QuestVerdict
	reviewScores   []int
	reviewIdx      int
}

type structuredTextExecutor struct {
	*executor.MockExecutor
}

func (e *structuredTextExecutor) SendMessage(ctx context.Context, sessionID string, msg executor.Message, modelName string, stream chan<- executor.StreamChunk) (*executor.ChatResponse, error) {
	return &executor.ChatResponse{
		SessionID: sessionID,
		Message: executor.Message{
			Role:    "assistant",
			Content: "```json\n{\"status\":\"done\",\"summary\":\"text only\"}\n```",
		},
		FinishReason: "stop",
	}, nil
}

func newScriptedLoopExecutor(verdicts ...model.QuestVerdict) *scriptedLoopExecutor {
	return &scriptedLoopExecutor{
		MockExecutor:   executor.NewMockExecutor("scripted_loop"),
		reviewVerdicts: verdicts,
	}
}

func newScriptedLoopExecutorWithScores(verdicts []model.QuestVerdict, scores []int) *scriptedLoopExecutor {
	return &scriptedLoopExecutor{
		MockExecutor:   executor.NewMockExecutor("scripted_loop"),
		reviewVerdicts: verdicts,
		reviewScores:   scores,
	}
}

func (e *scriptedLoopExecutor) SendMessage(ctx context.Context, sessionID string, msg executor.Message, modelName string, stream chan<- executor.StreamChunk) (*executor.ChatResponse, error) {
	// execSID 格式为 "qst_xxx/warrior_design_0"，取最后一段做 phase 判断
	shortSID := sessionID
	if idx := strings.LastIndex(shortSID, "/"); idx >= 0 {
		shortSID = shortSID[idx+1:]
	}
	if strings.HasPrefix(shortSID, "warrior_") || strings.HasPrefix(shortSID, "warrior_design_") {
		return phaseDoneResp(sessionID, "warrior done", "warrior delivered"), nil
	}

	e.mu.Lock()
	verdict := model.VerdictPass
	if e.reviewIdx < len(e.reviewVerdicts) {
		verdict = e.reviewVerdicts[e.reviewIdx]
	}
	score := 8
	if e.reviewIdx < len(e.reviewScores) {
		score = e.reviewScores[e.reviewIdx]
	}
	e.reviewIdx++
	e.mu.Unlock()

	comment := fmt.Sprintf("review %s", verdict)
	hints := ""
	if verdict == model.VerdictRequestChange {
		hints = "fix requested"
		comment = fmt.Sprintf("review %s\n\n修复进度: 0/2", verdict)
	}
	return reviewDoneResp(sessionID, "mage reviewed", string(verdict), comment, hints, score), nil
}

func useScriptedExecutor(t *testing.T, eng *Engine, verdicts ...model.QuestVerdict) {
	t.Helper()
	eng.RegisterExecutor("test_agent", newScriptedLoopExecutor(verdicts...))
}

func useScriptedExecutorWithScores(t *testing.T, eng *Engine, verdicts []model.QuestVerdict, scores []int) {
	t.Helper()
	eng.RegisterExecutor("test_agent", newScriptedLoopExecutorWithScores(verdicts, scores))
}

func TestMacroLoopGolden_StandardPassReachesUserReview(t *testing.T) {
	eng, _ := setupTestEngine(t)
	useScriptedExecutor(t, eng, model.VerdictPass)
	ctx := context.Background()

	q, err := eng.CreateQuest(ctx, "standard pass", model.QuestTypeExecute, "", CreateQuestOptions{
		WorkspaceMode: model.WorkspaceReadOnly,
	})
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	if err := eng.StartQuest(ctx, q.ID); err != nil {
		t.Fatalf("StartQuest failed: %v", err)
	}
	waitForStatus(t, eng, q.ID, model.QuestStatusSuccess, 10*time.Second)
	got, _ := eng.GetQuest(q.ID)
	if got.FinalVerdict != model.VerdictPass || got.ReworkCount != 0 {
		t.Fatalf("bad pass result: %+v", got)
	}
}

func TestMacroLoopGolden_LowScorePassDoesNotRewriteMageVerdict(t *testing.T) {
	eng, _ := setupTestEngine(t)
	useScriptedExecutorWithScores(t, eng, []model.QuestVerdict{model.VerdictPass}, []int{3})
	ctx := context.Background()

	q, err := eng.CreateQuest(ctx, "low score pass", model.QuestTypeExecute, "", CreateQuestOptions{
		WorkspaceMode: model.WorkspaceReadOnly,
	})
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	if err := eng.StartQuest(ctx, q.ID); err != nil {
		t.Fatalf("StartQuest failed: %v", err)
	}
	waitForStatus(t, eng, q.ID, model.QuestStatusSuccess, 10*time.Second)
	got, _ := eng.GetQuest(q.ID)
	if got.FinalVerdict != model.VerdictPass || got.ReworkCount != 0 {
		t.Fatalf("low-score pass should stay pass and not trigger rework: %+v", got)
	}
	rows, err := fsstore.NewQuestStore(eng.root).ReadEvents(q.ID, 0)
	if err != nil {
		t.Fatalf("ReadEvents failed: %v", err)
	}
	foundSignal := false
	for _, row := range rows {
		if row.Type == "review.signal" {
			if payload, ok := row.Payload.(map[string]any); ok {
				if signal, _ := payload["signal"].(string); signal == "low_score_pass" {
					foundSignal = true
				}
			}
		}
	}
	if !foundSignal {
		t.Fatalf("low-score pass should emit review.signal, rows=%+v", rows)
	}
}

func TestMacroLoopGolden_ReworkStalemateIsSignalOnly(t *testing.T) {
	eng, _ := setupTestEngine(t)
	eng.cfg.MaxReworkPerQuest = 4
	useScriptedExecutor(t, eng,
		model.VerdictRequestChange,
		model.VerdictRequestChange,
		model.VerdictRequestChange,
		model.VerdictPass,
	)
	ctx := context.Background()

	q, err := eng.CreateQuest(ctx, "stuck rework signal", model.QuestTypeExecute, "", CreateQuestOptions{
		WorkspaceMode: model.WorkspaceReadOnly,
		Intensity:     model.QuestIntensityStandard,
	})
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	if err := eng.StartQuest(ctx, q.ID); err != nil {
		t.Fatalf("StartQuest failed: %v", err)
	}
	waitForStatus(t, eng, q.ID, model.QuestStatusSuccess, 10*time.Second)
	got, _ := eng.GetQuest(q.ID)
	if got.Intensity != model.QuestIntensityStandard || got.ReworkCount != 3 || got.FinalVerdict != model.VerdictPass {
		t.Fatalf("rework stalemate should not change intensity or override verdict: %+v", got)
	}
	rows, err := fsstore.NewQuestStore(eng.root).ReadEvents(q.ID, 0)
	if err != nil {
		t.Fatalf("ReadEvents failed: %v", err)
	}
	foundSignal := false
	for _, row := range rows {
		if row.Type == "review.signal" {
			if payload, ok := row.Payload.(map[string]any); ok {
				if signal, _ := payload["signal"].(string); signal == "rework_stalemate" {
					foundSignal = true
				}
			}
		}
	}
	if !foundSignal {
		t.Fatalf("stuck rework should emit review.signal, rows=%+v", rows)
	}
}

func TestMacroLoopGolden_QuickSkipsMage(t *testing.T) {
	eng, _ := setupTestEngine(t)
	useScriptedExecutor(t, eng, model.VerdictReject)
	ctx := context.Background()

	q, err := eng.CreateQuest(ctx, "quick pass", model.QuestTypeExecute, "", CreateQuestOptions{
		Intensity:     model.QuestIntensityQuick,
		WorkspaceMode: model.WorkspaceReadOnly,
	})
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	if err := eng.StartQuest(ctx, q.ID); err != nil {
		t.Fatalf("StartQuest failed: %v", err)
	}
	waitForStatus(t, eng, q.ID, model.QuestStatusSuccess, 10*time.Second)
	got, _ := eng.GetQuest(q.ID)
	if got.FinalVerdict != model.VerdictPass || got.MageID != "" {
		t.Fatalf("quick should skip mage and enter user review as pass: %+v", got)
	}
	if rows, err := fsstore.NewQuestStore(eng.root).ReadSessionRows(q.ID, "mage_0", 0); err == nil && len(rows) > 0 {
		t.Fatalf("quick quest should not create mage session rows: %+v", rows)
	}
}

func TestMacroLoopGolden_RequestChangesReworksThenUserReview(t *testing.T) {
	eng, _ := setupTestEngine(t)
	useScriptedExecutor(t, eng, model.VerdictRequestChange, model.VerdictPass)
	ctx := context.Background()

	q, err := eng.CreateQuest(ctx, "request changes", model.QuestTypeExecute, "", CreateQuestOptions{
		WorkspaceMode: model.WorkspaceReadOnly,
	})
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	if err := eng.StartQuest(ctx, q.ID); err != nil {
		t.Fatalf("StartQuest failed: %v", err)
	}
	waitForStatus(t, eng, q.ID, model.QuestStatusSuccess, 10*time.Second)
	got, _ := eng.GetQuest(q.ID)
	if got.ReworkCount != 1 || got.FinalVerdict != model.VerdictPass || got.ReviewHints != "fix requested" {
		t.Fatalf("bad rework result: %+v", got)
	}
	loopState, err := eng.root.LoadLoopStateSpine(q.ID)
	if err != nil {
		t.Fatalf("LoadLoopStateSpine failed: %v", err)
	}
	if loopState.NextExpectedAction != "fix requested" || len(loopState.Attempts) == 0 || loopState.Attempts[0].Result != string(model.VerdictRequestChange) {
		t.Fatalf("bad loop state spine: %+v", loopState)
	}
}

func TestMacroLoopGolden_RejectFailsQuest(t *testing.T) {
	eng, _ := setupTestEngine(t)
	useScriptedExecutor(t, eng, model.VerdictReject)
	ctx := context.Background()

	q, err := eng.CreateQuest(ctx, "reject", model.QuestTypeExecute, "", CreateQuestOptions{
		WorkspaceMode: model.WorkspaceReadOnly,
	})
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	if err := eng.StartQuest(ctx, q.ID); err != nil {
		t.Fatalf("StartQuest failed: %v", err)
	}
	waitForStatus(t, eng, q.ID, model.QuestStatusFailed, 10*time.Second)
	got, _ := eng.GetQuest(q.ID)
	if got.CompletedAtMs == 0 || got.FinalComment == "" {
		t.Fatalf("reject should fail and record final comment: %+v", got)
	}
}

func TestMacroLoopGolden_ThreePhasePipelineRunsSecondReview(t *testing.T) {
	eng, _ := setupTestEngine(t)
	useScriptedExecutor(t, eng, model.VerdictPass, model.VerdictPass)
	ctx := context.Background()

	q, err := eng.CreateQuest(ctx, "three phase", model.QuestTypeExecute, "", CreateQuestOptions{
		WorkspaceMode: model.WorkspaceReadOnly,
	})
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	qs := fsstore.NewQuestStore(eng.root)
	q, err = qs.LoadQuest(q.ID)
	if err != nil {
		t.Fatalf("LoadQuest failed: %v", err)
	}
	q.PipelineName = "custom"
	q.PhaseCount = 3
	q.PipelineDef = []fsstore.PhaseTask{
		{PhaseIdx: 0, Role: "warrior", Class: model.ClassWarrior, Goal: "execute", Status: model.PhasePending},
		{PhaseIdx: 1, Role: "mage", Class: model.ClassMage, Goal: "review", Status: model.PhasePending},
		{PhaseIdx: 2, Role: "mage", Class: model.ClassMage, Goal: "second review", Status: model.PhasePending},
	}
	q.PipelineDefHash = ""
	q.Phases = nil
	if err := qs.SaveQuest(q); err != nil {
		t.Fatalf("SaveQuest failed: %v", err)
	}

	if err := eng.StartQuest(ctx, q.ID); err != nil {
		t.Fatalf("StartQuest failed: %v", err)
	}
	waitForStatus(t, eng, q.ID, model.QuestStatusSuccess, 10*time.Second)
	got, _ := eng.GetQuest(q.ID)
	if got.PhaseCount != 3 || got.CurrentPhaseIdx() != 2 || got.FinalVerdict != model.VerdictPass {
		t.Fatalf("bad three phase result: %+v", got)
	}
	rows, err := qs.ReadSessionRows(q.ID, "mage_2_0", 0)
	if err != nil || len(rows) == 0 {
		t.Fatalf("expected second review session rows, rows=%d err=%v", len(rows), err)
	}
}

func TestMacroLoopGolden_WithDesignPhaseRunsFourPhasePipeline(t *testing.T) {
	eng, _ := setupTestEngine(t)
	useScriptedExecutor(t, eng, model.VerdictPass, model.VerdictPass)
	ctx := context.Background()

	q, err := eng.CreateQuest(ctx, "design phase", model.QuestTypeExecute, "", CreateQuestOptions{
		WorkspaceMode:   model.WorkspaceReadOnly,
		WithDesignPhase: true,
	})
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	if q.PipelineName != "design" || q.PhaseCount != 4 {
		t.Fatalf("quest should use design pipeline: %+v", q)
	}

	if err := eng.StartQuest(ctx, q.ID); err != nil {
		t.Fatalf("StartQuest failed: %v", err)
	}
	waitForStatus(t, eng, q.ID, model.QuestStatusSuccess, 10*time.Second)
	got, _ := eng.GetQuest(q.ID)
	if got.PhaseCount != 4 || got.CurrentPhaseIdx() != 3 || got.FinalVerdict != model.VerdictPass {
		t.Fatalf("bad design pipeline result: %+v", got)
	}
	qs := fsstore.NewQuestStore(eng.root)
	for _, sid := range []string{"warrior_design_0", "mage_design_review_0", "warrior_execute_0", "mage_implementation_review_0"} {
		rows, err := qs.ReadSessionRows(q.ID, sid, 0)
		if err != nil || len(rows) == 0 {
			t.Fatalf("expected session rows for %s, rows=%d err=%v", sid, len(rows), err)
		}
	}
	executeRows, err := qs.ReadContextPackRows(q.ID, "warrior_execute_0", 0)
	if err != nil || len(executeRows) == 0 {
		t.Fatalf("expected warrior_execute context rows, rows=%d err=%v", len(executeRows), err)
	}
	if !strings.Contains(executeRows[0].Content, `name="design_plan"`) {
		t.Fatalf("warrior_execute should receive design_plan block:\n%s", executeRows[0].Content)
	}
	reviewRows, err := qs.ReadContextPackRows(q.ID, "mage_implementation_review_0", 0)
	if err != nil || len(reviewRows) == 0 {
		t.Fatalf("expected implementation review context rows, rows=%d err=%v", len(reviewRows), err)
	}
	if !strings.Contains(reviewRows[0].Content, `name="design_plan"`) {
		t.Fatalf("implementation review should receive design_plan block:\n%s", reviewRows[0].Content)
	}
	if !strings.Contains(reviewRows[0].Content, `name="design_alignment_review"`) {
		t.Fatalf("implementation review should receive design_alignment_review block:\n%s", reviewRows[0].Content)
	}
}

func TestMacroLoopGolden_ImplementationReviewReworksToExecutePhase(t *testing.T) {
	eng, _ := setupTestEngine(t)
	useScriptedExecutor(t, eng, model.VerdictPass, model.VerdictRequestChange, model.VerdictPass)
	ctx := context.Background()

	q, err := eng.CreateQuest(ctx, "design phase implementation rework", model.QuestTypeExecute, "", CreateQuestOptions{
		WorkspaceMode:   model.WorkspaceReadOnly,
		WithDesignPhase: true,
	})
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	if err := eng.StartQuest(ctx, q.ID); err != nil {
		t.Fatalf("StartQuest failed: %v", err)
	}
	waitForStatus(t, eng, q.ID, model.QuestStatusSuccess, 10*time.Second)
	got, _ := eng.GetQuest(q.ID)
	if got.ReworkCount != 1 || got.CurrentPhaseIdx() != 3 || got.FinalVerdict != model.VerdictPass {
		t.Fatalf("bad implementation rework result: %+v", got)
	}

	qs := fsstore.NewQuestStore(eng.root)
	if rows, err := qs.ReadSessionRows(q.ID, "warrior_design_1", 0); err == nil && len(rows) > 0 {
		t.Fatalf("implementation rework should not rerun design phase: %+v", rows)
	}
	for _, sid := range []string{"warrior_execute_1", "mage_implementation_review_1"} {
		rows, err := qs.ReadSessionRows(q.ID, sid, 0)
		if err != nil || len(rows) == 0 {
			t.Fatalf("expected rerun session rows for %s, rows=%d err=%v", sid, len(rows), err)
		}
	}
}
