package fsstore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"code.byted.org/lihuanyu.0w0/gloop/internal/domain/quest"
	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

func TestQuestStatus_ValidTransitions(t *testing.T) {
	// pending → running → reviewing → user_review → success
	q := &QuestMeta{Status: model.QuestStatusPending}
	if err := q.TransitionTo(model.QuestStatusRunning); err != nil {
		t.Errorf("pending→running should be ok, got %v", err)
	}
	if err := q.TransitionTo(model.QuestStatusReviewing); err != nil {
		t.Errorf("running→reviewing should be ok, got %v", err)
	}
	if err := q.TransitionTo(model.QuestStatusUserReview); err != nil {
		t.Errorf("reviewing→user_review should be ok, got %v", err)
	}
	if err := q.TransitionTo(model.QuestStatusSuccess); err != nil {
		t.Errorf("user_review→success should be ok, got %v", err)
	}
	// 终态不能再迁移
	if err := q.TransitionTo(model.QuestStatusRunning); err == nil {
		t.Error("success→running should fail")
	}
	if !q.IsTerminal() {
		t.Error("success should be terminal")
	}
}

func TestCleanupExpiredWorkspaces(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	qs := NewQuestStore(root)
	now := NowMs()

	// 受管隔离工作目录：位于 <dataDir>/workspace/quests/<qid>/work 之下，
	// 是 gloop 自己创建、可安全自动删除的 —— 应被清理。
	managedWork, err := qs.WorkDir("qst_cleanup_success")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(managedWork, "artifact.txt"), []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	q := &QuestMeta{
		ID:            "qst_cleanup_success",
		ShortID:       "cleanup",
		Query:         "cleanup",
		Type:          model.QuestTypeExecute,
		Status:        model.QuestStatusSuccess,
		WorkspacePath: managedWork,
		CreatedAtMs:   now - 10*24*60*60*1000,
		CompletedAtMs: now - 8*24*60*60*1000,
	}
	if err := qs.CreateQuest(q); err != nil {
		t.Fatal(err)
	}
	q.WorkspacePath = managedWork
	if err := qs.SaveQuest(q); err != nil {
		t.Fatal(err)
	}

	// 外部用户工作区：在 gloop 数据目录之外（模拟用户 cwd 被误记为 workspace_path），
	// 绝不应被自动清理 —— 应被护栏跳过。
	externalWork := filepath.Join(t.TempDir(), "work")
	if err := os.MkdirAll(externalWork, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(externalWork, "keep.txt"), []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	qExt := &QuestMeta{
		ID:            "qst_cleanup_external",
		ShortID:       "external",
		Query:         "external",
		Type:          model.QuestTypeExecute,
		Status:        model.QuestStatusSuccess,
		WorkspacePath: externalWork,
		CreatedAtMs:   now - 10*24*60*60*1000,
		CompletedAtMs: now - 8*24*60*60*1000,
	}
	if err := qs.CreateQuest(qExt); err != nil {
		t.Fatal(err)
	}
	qExt.WorkspacePath = externalWork
	if err := qs.SaveQuest(qExt); err != nil {
		t.Fatal(err)
	}

	items, err := qs.CleanupExpiredWorkspaces(7, false, now, true)
	if err != nil {
		t.Fatalf("dry-run cleanup failed: %v", err)
	}
	// dry-run 只列出受管候选，外部目录被护栏跳过
	if len(items) != 1 || items[0].QuestID != "qst_cleanup_success" {
		t.Fatalf("dry-run items = %+v, want only qst_cleanup_success", items)
	}
	if _, err := os.Stat(managedWork); err != nil {
		t.Fatalf("dry-run should keep managed workspace: %v", err)
	}
	if _, err := os.Stat(externalWork); err != nil {
		t.Fatalf("dry-run should keep external workspace: %v", err)
	}

	items, err = qs.CleanupExpiredWorkspaces(7, false, now, false)
	if err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}
	if len(items) != 1 || items[0].QuestID != "qst_cleanup_success" {
		t.Fatalf("items = %+v, want only qst_cleanup_success", items)
	}
	if _, err := os.Stat(managedWork); !os.IsNotExist(err) {
		t.Fatalf("managed workspace should be removed, err=%v", err)
	}
	// 外部用户工作区必须完好无损
	if _, err := os.Stat(externalWork); err != nil {
		t.Fatalf("external workspace must NOT be removed: %v", err)
	}
	got, err := qs.LoadQuest(q.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.WorkspaceCleaned || got.WorkspaceCleanedAtMs == 0 {
		t.Fatalf("quest cleanup markers not set: %+v", got)
	}
	// 外部 quest 不应被标记为已清理
	gotExt, err := qs.LoadQuest(qExt.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotExt.WorkspaceCleaned {
		t.Fatalf("external quest must not be marked cleaned")
	}
}

func TestPhaseArtifactForReviewIncludesToolEvidence(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	qs := NewQuestStore(root)
	q := &QuestMeta{
		ID:          "qst_artifact",
		ShortID:     "artifact",
		Query:       "review artifact",
		Type:        model.QuestTypeExecute,
		Status:      model.QuestStatusRunning,
		CreatedBy:   "user",
		CreatedAtMs: model.NowMs(),
	}
	if err := qs.CreateQuest(q); err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	if err := qs.AppendSessionRow(q.ID, "warrior_0", &QuestSessionRow{
		Kind:    "message",
		Role:    "assistant",
		Content: "done",
	}); err != nil {
		t.Fatalf("AppendSessionRow message failed: %v", err)
	}
	if err := qs.AppendSessionRow(q.ID, "warrior_0", &QuestSessionRow{
		Kind:     "tool_call",
		Role:     "assistant",
		ToolName: "phase_checkpoint",
		ToolArgs: `{"status":"done","summary":"GLOOP_REAL_AGENT_CONTRACT_OK"}`,
	}); err != nil {
		t.Fatalf("AppendSessionRow tool_call failed: %v", err)
	}
	if err := qs.AppendSessionRow(q.ID, "warrior_0", &QuestSessionRow{
		Kind:     "tool_result",
		Role:     "tool",
		ToolName: "phase_checkpoint",
		Content:  `{"ok":true,"phase_ended":true}`,
	}); err != nil {
		t.Fatalf("AppendSessionRow tool_result failed: %v", err)
	}
	if err := qs.AppendSessionRow(q.ID, "warrior_0", &QuestSessionRow{
		Kind:     "native_tool_call_observed",
		Role:     "assistant",
		ToolName: "wc -c marker.txt",
		ToolArgs: `{"command":"wc -c marker.txt"}`,
		Meta: map[string]any{
			"status": "completed",
			"result": "13 marker.txt",
		},
	}); err != nil {
		t.Fatalf("AppendSessionRow native tool evidence failed: %v", err)
	}

	got, err := qs.PhaseArtifactForReview(q.ID, "warrior_0")
	if err != nil {
		t.Fatalf("PhaseArtifactForReview failed: %v", err)
	}
	if got.Assistant != "done" {
		t.Fatalf("assistant = %q, want done", got.Assistant)
	}
	for _, want := range []string{
		"tool_call phase_checkpoint",
		"GLOOP_REAL_AGENT_CONTRACT_OK",
		"tool_result phase_checkpoint",
		"native_tool_observed wc -c marker.txt status=completed",
		"13 marker.txt",
	} {
		if !strings.Contains(got.PlatformToolEvidence, want) {
			t.Fatalf("platform evidence missing %q:\n%s", want, got.PlatformToolEvidence)
		}
	}
	if !strings.Contains(got.Evidence.PlatformMechanisms, "tool_call phase_checkpoint") {
		t.Fatalf("platform mechanisms missing phase checkpoint: %+v", got.Evidence)
	}
	if !strings.Contains(got.Evidence.NativeToolCalls, "native_tool_observed wc -c marker.txt") {
		t.Fatalf("native tool evidence missing: %+v", got.Evidence)
	}
}

func TestFindPhaseArtifactByKindDesignPlan(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	qs := NewQuestStore(root)
	q := &QuestMeta{
		ID:           "qst_design_plan",
		ShortID:      "designplan",
		Query:        "design first",
		Type:         model.QuestTypeExecute,
		Status:       model.QuestStatusRunning,
		PipelineName: "design",
		PipelineDef:  PhaseDefsFromDomain(quest.DesignPipeline()),
		PhaseCount:   4,
		CreatedBy:    "user",
		CreatedAtMs:  model.NowMs(),
	}
	if err := qs.CreateQuest(q); err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	if err := qs.AppendSessionRow(q.ID, "warrior_design_0", &QuestSessionRow{
		Kind:    "message",
		Role:    "assistant",
		Content: "设计方案正文",
	}); err != nil {
		t.Fatalf("AppendSessionRow failed: %v", err)
	}
	got, ok, err := qs.FindPhaseArtifactByKind(q.ID, "design_plan")
	if err != nil {
		t.Fatalf("FindPhaseArtifactByKind failed: %v", err)
	}
	if !ok || got.Kind != "design_plan" || got.PhaseName != "warrior_design" || got.Assistant != "设计方案正文" {
		t.Fatalf("bad design artifact: ok=%v artifact=%+v", ok, got)
	}
}

func TestPhaseArtifactForReviewIncludesWarriorOutputs(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	qs := NewQuestStore(root)
	q := &QuestMeta{
		ID:          "qst_outputs",
		ShortID:     "outputs",
		Query:       "review outputs",
		Type:        model.QuestTypeExecute,
		Status:      model.QuestStatusRunning,
		CreatedBy:   "user",
		CreatedAtMs: model.NowMs(),
	}
	if err := qs.CreateQuest(q); err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	if err := qs.AppendSessionRow(q.ID, "warrior_0", &QuestSessionRow{
		Kind:    "message",
		Role:    "assistant",
		Content: "done with outputs",
	}); err != nil {
		t.Fatalf("AppendSessionRow message failed: %v", err)
	}
	art, err := qs.AddDeclaredOutput(q.ID, DeclaredOutput{
		Name:   "Design Doc",
		Kind:   "document",
		URL:    "https://example.com/doc",
		Source: "warrior_phase",
	})
	if err != nil {
		t.Fatalf("AddDeclaredOutput failed: %v", err)
	}
	got, err := qs.PhaseArtifactForReview(q.ID, "warrior_0")
	if err != nil {
		t.Fatalf("PhaseArtifactForReview failed: %v", err)
	}
	if len(got.Outputs) != 1 || got.Outputs[0].ID != art.ID {
		t.Fatalf("outputs = %+v, want one %s", got.Outputs, art.ID)
	}
	check := got.OutputChecks[art.ID]
	if check.Status != "external" || check.Verified {
		t.Fatalf("check = %+v, want external unverified", check)
	}
}

func TestThreadPostsRootAndAuthority(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	qs := NewQuestStore(root)
	q := &QuestMeta{
		ID:              "qst_thread",
		ShortID:         "thread",
		Query:           "summary fallback",
		OriginalRequest: "原始委托正文\n必须保留",
		Type:            model.QuestTypeExecute,
		Status:          model.QuestStatusPending,
		WarriorID:       "adv_warrior",
		MageID:          "adv_mage",
		CreatedBy:       "user",
		CreatedAtMs:     1234,
	}
	if err := qs.CreateQuest(q); err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}

	rootPost, err := qs.EnsureRootThreadPost(q)
	if err != nil {
		t.Fatalf("EnsureRootThreadPost failed: %v", err)
	}
	if rootPost.PostID != RootPostID(q.ID) || rootPost.Content != q.OriginalRequest || rootPost.AuthorRole != model.PostRoleHuman {
		t.Fatalf("bad root post: %+v", rootPost)
	}

	if _, err := qs.AppendThreadPost(q.ID, AppendThreadPostOptions{
		Content:    "checker pretending delivery",
		AuthorRole: model.PostRoleChecker,
		ArtifactRefs: []ThreadArtifactRef{{
			ID:           "delivery_1",
			ArtifactType: model.ArtifactTypeDelivery,
		}},
	}); err == nil {
		t.Fatal("checker must not attach DeliveryArtifact")
	}
	if _, err := qs.AppendThreadPost(q.ID, AppendThreadPostOptions{
		Content:    "maker pretending review",
		AuthorRole: model.PostRoleMaker,
		ArtifactRefs: []ThreadArtifactRef{{
			ID:           "review_1",
			ArtifactType: model.ArtifactTypeReviewReport,
		}},
	}); err == nil {
		t.Fatal("maker must not attach ReviewReport")
	}

	if err := root.AppendMakerReport(q.ID, &MakerReport{
		ReportID:  "maker_report_test",
		SessionID: "warrior_0",
		Phase:     0,
		Verdict:   "done",
		Summary:   "delivered",
		Artifacts: []string{"artifact_delivery"},
	}); err != nil {
		t.Fatalf("AppendMakerReport failed: %v", err)
	}
	if err := root.AppendReviewReport(q.ID, &ReviewReport{
		ReportID:       "review_report_test",
		SessionID:      "mage_0",
		Phase:          1,
		Verdict:        model.VerdictPass,
		Comment:        "checked",
		CheckedAgainst: []string{"artifact_delivery"},
		EvidenceRefs: []EvidenceRef{{
			ID:        "ev_t0",
			TrustTier: "T0",
			Source:    "go test",
		}},
	}); err != nil {
		t.Fatalf("AppendReviewReport failed: %v", err)
	}

	posts, err := qs.LoadThreadPosts(q.ID)
	if err != nil {
		t.Fatalf("LoadThreadPosts failed: %v", err)
	}
	if len(posts) != 3 {
		t.Fatalf("thread posts = %d, want root + maker + checker; posts=%+v", len(posts), posts)
	}
	if posts[1].AuthorRole != model.PostRoleMaker || len(posts[1].ArtifactRefs) != 2 {
		t.Fatalf("bad maker post: %+v", posts[1])
	}
	if posts[2].AuthorRole != model.PostRoleChecker || len(posts[2].ArtifactRefs) != 2 {
		t.Fatalf("bad checker post: %+v", posts[2])
	}
}

func TestSaveQuestProjectsReworkAndTerminalOutcome(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	qs := NewQuestStore(root)
	q := &QuestMeta{
		ID:              "qst_projection",
		ShortID:         "projection",
		Query:           "projection quest",
		OriginalRequest: "projection quest",
		Type:            model.QuestTypeExecute,
		Status:          model.QuestStatusReviewing,
		WarriorID:       "adv_warrior",
		MageID:          "adv_mage",
		MaxRework:       2,
		CreatedBy:       "user",
	}
	if err := qs.CreateQuest(q); err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	posts, err := qs.LoadThreadPosts(q.ID)
	if err != nil {
		t.Fatalf("LoadThreadPosts failed: %v", err)
	}
	if len(posts) != 0 {
		t.Fatalf("plain CreateQuest should not project posts directly, got %+v", posts)
	}

	q.Status = model.QuestStatusRunning
	q.ReworkCount = 1
	q.ReviewHints = "fix it"
	if err := qs.SaveQuest(q); err != nil {
		t.Fatalf("SaveQuest rework failed: %v", err)
	}
	posts, err = qs.LoadThreadPosts(q.ID)
	if err != nil {
		t.Fatalf("LoadThreadPosts after rework failed: %v", err)
	}
	if len(posts) != 2 {
		t.Fatalf("posts after rework = %d, want root + rework: %+v", len(posts), posts)
	}
	if posts[1].Kind != "system_rework" || posts[1].AuthorRole != model.PostRoleSystem || !strings.Contains(posts[1].Content, "Round 1/2") {
		t.Fatalf("bad rework projection: %+v", posts[1])
	}

	q.Status = model.QuestStatusSuccess
	q.FinalVerdict = model.VerdictPass
	q.FinalComment = "done"
	q.FinalizedBy = "run_mode_auto"
	q.CompletedAtMs = 12345
	if err := qs.SaveQuest(q); err != nil {
		t.Fatalf("SaveQuest terminal failed: %v", err)
	}
	got, err := qs.LoadQuest(q.ID)
	if err != nil {
		t.Fatalf("LoadQuest failed: %v", err)
	}
	if got.PinnedOutcomeSummary == "" || !strings.Contains(got.PinnedOutcomeSummary, "Outcome: success") {
		t.Fatalf("pinned outcome missing: %+v", got)
	}
	posts, err = qs.LoadThreadPosts(q.ID)
	if err != nil {
		t.Fatalf("LoadThreadPosts after terminal failed: %v", err)
	}
	if len(posts) != 3 {
		t.Fatalf("posts after terminal = %d, want root + rework + decision: %+v", len(posts), posts)
	}
	decision := posts[2]
	if decision.Kind != "decision_note" || decision.AuthorRole != model.PostRoleSystem || len(decision.ArtifactRefs) != 1 {
		t.Fatalf("bad decision projection: %+v", decision)
	}
	if decision.ArtifactRefs[0].ArtifactType != model.ArtifactTypeDecisionNote {
		t.Fatalf("decision artifact type = %+v", decision.ArtifactRefs[0])
	}
}

func TestSaveQuestDoesNotProjectOrdinaryMetadataSave(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	qs := NewQuestStore(root)
	q := &QuestMeta{
		ID:          "qst_no_projection",
		ShortID:     "noproj",
		Query:       "",
		Type:        model.QuestTypeExecute,
		Status:      model.QuestStatusRunning,
		CreatedBy:   "user",
		CreatedAtMs: model.NowMs(),
	}
	if err := qs.CreateQuest(q); err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	q.Outputs = []QuestArtifact{{ID: "out_1", Name: "artifact"}}
	if err := qs.SaveQuest(q); err != nil {
		t.Fatalf("ordinary SaveQuest should not require root post content: %v", err)
	}
	posts, err := qs.LoadThreadPosts(q.ID)
	if err != nil {
		t.Fatalf("LoadThreadPosts failed: %v", err)
	}
	if len(posts) != 0 {
		t.Fatalf("ordinary metadata save should not project posts: %+v", posts)
	}
}

func TestSaveQuestProjectsBlockedEscalationForLegacyQuest(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	qs := NewQuestStore(root)
	q := &QuestMeta{
		ID:        "qst_blocked_projection",
		ShortID:   "blocked",
		Query:     "",
		Type:      model.QuestTypeExecute,
		Status:    model.QuestStatusRunning,
		CreatedBy: "user",
	}
	if err := qs.CreateQuest(q); err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	q.Status = model.QuestStatusBlocked
	q.BlockedReason = "连续返工无进展"
	q.BlockedReasonCode = "no_progress"
	if err := qs.SaveQuest(q); err != nil {
		t.Fatalf("SaveQuest blocked failed: %v", err)
	}
	posts, err := qs.LoadThreadPosts(q.ID)
	if err != nil {
		t.Fatalf("LoadThreadPosts failed: %v", err)
	}
	if len(posts) != 2 {
		t.Fatalf("posts after blocked = %d, want root + escalation: %+v", len(posts), posts)
	}
	if posts[0].Content == "" || !strings.Contains(posts[0].Content, q.ID) {
		t.Fatalf("legacy root fallback missing: %+v", posts[0])
	}
	if posts[1].Kind != "system_escalation" || !strings.Contains(posts[1].Content, "blocked") || !strings.Contains(posts[1].Content, "no_progress") {
		t.Fatalf("bad blocked projection: %+v", posts[1])
	}
}

func TestSaveQuest_SystemPost_SourceEventID_Rework(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	qs := NewQuestStore(root)
	q := &QuestMeta{
		ID:        "qst_rework_evt",
		ShortID:   "rwk",
		Query:     "rework with event",
		Type:      model.QuestTypeExecute,
		Status:    model.QuestStatusReviewing,
		WarriorID: "adv_warrior",
		MageID:    "adv_mage",
		MaxRework: 2,
		CreatedBy: "user",
	}
	if err := qs.CreateQuest(q); err != nil {
		t.Fatalf("CreateQuest: %v", err)
	}

	// Append a triggering event before the state transition.
	eventRow := &QuestEventRow{
		Timestamp: model.NowMs(),
		Type:      "quest.phase_changed",
		QuestID:   q.ID,
		SessionID: "mage_0",
		Payload:   map[string]any{"phase": "review", "verdict": "request_changes"},
	}
	if err := qs.AppendEvent(q.ID, eventRow); err != nil {
		t.Fatalf("AppendEvent: %v", err)
	}
	if eventRow.ID == 0 {
		t.Fatal("appended event has zero ID")
	}

	q.Status = model.QuestStatusRunning
	q.ReworkCount = 1
	q.ReviewHints = "fix the bug"
	if err := qs.SaveQuest(q); err != nil {
		t.Fatalf("SaveQuest rework: %v", err)
	}

	posts, err := qs.LoadThreadPosts(q.ID)
	if err != nil {
		t.Fatalf("LoadThreadPosts: %v", err)
	}
	var reworkPost *ThreadPost
	for i := range posts {
		if posts[i].Kind == "system_rework" {
			reworkPost = &posts[i]
			break
		}
	}
	if reworkPost == nil {
		t.Fatal("no system_rework post found")
	}
	if reworkPost.SourceEventID != eventRow.ID {
		t.Fatalf("rework SourceEventID = %d, want %d (appended event ID)", reworkPost.SourceEventID, eventRow.ID)
	}

	// Verify the event actually exists in the quest's event log.
	events, err := qs.ReadEvents(q.ID, 0)
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}
	foundEvent := false
	for _, ev := range events {
		if ev.ID == eventRow.ID {
			foundEvent = true
			break
		}
	}
	if !foundEvent {
		t.Fatalf("event %d not found in quest event log", eventRow.ID)
	}
}

func TestSaveQuest_SystemPost_SourceEventID_Terminal(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	qs := NewQuestStore(root)
	q := &QuestMeta{
		ID:          "qst_terminal_evt",
		ShortID:     "trm",
		Query:       "terminal with event",
		Type:        model.QuestTypeExecute,
		Status:      model.QuestStatusRunning,
		WarriorID:   "adv_warrior",
		MageID:      "adv_mage",
		CreatedBy:   "user",
		CreatedAtMs: model.NowMs(),
	}
	if err := qs.CreateQuest(q); err != nil {
		t.Fatalf("CreateQuest: %v", err)
	}

	termEvent := &QuestEventRow{
		Timestamp: model.NowMs(),
		Type:      "quest.success",
		QuestID:   q.ID,
		SessionID: "warrior_0",
		Payload:   map[string]any{"summary": "all done"},
	}
	if err := qs.AppendEvent(q.ID, termEvent); err != nil {
		t.Fatalf("AppendEvent: %v", err)
	}

	q.Status = model.QuestStatusSuccess
	q.FinalVerdict = model.VerdictPass
	q.FinalComment = "ship it"
	q.FinalizedBy = "run_mode_auto"
	q.CompletedAtMs = model.NowMs()
	if err := qs.SaveQuest(q); err != nil {
		t.Fatalf("SaveQuest terminal: %v", err)
	}

	posts, err := qs.LoadThreadPosts(q.ID)
	if err != nil {
		t.Fatalf("LoadThreadPosts: %v", err)
	}
	var decisionPost *ThreadPost
	for i := range posts {
		if posts[i].Kind == "decision_note" {
			decisionPost = &posts[i]
			break
		}
	}
	if decisionPost == nil {
		t.Fatal("no decision_note post found")
	}
	if decisionPost.SourceEventID != termEvent.ID {
		t.Fatalf("decision_note SourceEventID = %d, want %d", decisionPost.SourceEventID, termEvent.ID)
	}
}

func TestSaveQuest_FanoutLeafDone_NotifiesRootThread(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	qs := NewQuestStore(root)

	// Create root quest (GroupID set, no FanoutLeafID = canonical root).
	rootQ := &QuestMeta{
		ID:          "qst_root_leaf_done",
		ShortID:     "rootld",
		Query:       "root quest for fanout",
		Type:        model.QuestTypeExecute,
		Status:      model.QuestStatusRunning,
		CreatedBy:   "user",
		GroupID:     "grp_test_leaf_done",
		CreatedAtMs: model.NowMs(),
	}
	if err := qs.CreateQuest(rootQ); err != nil {
		t.Fatalf("CreateQuest root: %v", err)
	}

	// Create leaf quest.
	leafQ := &QuestMeta{
		ID:           "qst_leaf_done_test",
		ShortID:      "leaft",
		Query:        "leaf quest",
		Type:         model.QuestTypeExecute,
		Status:       model.QuestStatusRunning,
		CreatedBy:    "adv_maker",
		GroupID:      "grp_test_leaf_done",
		FanoutLeafID: "leaf-1",
		ParentQuestID: rootQ.ID,
		CreatedAtMs:  model.NowMs(),
	}
	if err := qs.CreateQuest(leafQ); err != nil {
		t.Fatalf("CreateQuest leaf: %v", err)
	}

	// Transition leaf to terminal (success).
	leafQ.Status = model.QuestStatusSuccess
	leafQ.FinalVerdict = model.VerdictPass
	leafQ.CompletedAtMs = model.NowMs()
	if err := qs.SaveQuest(leafQ); err != nil {
		t.Fatalf("SaveQuest leaf terminal: %v", err)
	}

	// Verify root thread has a fanout_leaf_done post.
	rootPosts, err := qs.LoadThreadPosts(rootQ.ID)
	if err != nil {
		t.Fatalf("LoadThreadPosts root: %v", err)
	}
	var leafDonePost *ThreadPost
	for i := range rootPosts {
		if rootPosts[i].Kind == "fanout_leaf_done" {
			leafDonePost = &rootPosts[i]
			break
		}
	}
	if leafDonePost == nil {
		t.Fatal("root thread should have fanout_leaf_done post after leaf terminal")
	}
	if leafDonePost.PostID != "fanout_leaf_done_qst_leaf_done_test" {
		t.Fatalf("fanout_leaf_done post_id = %s, want fanout_leaf_done_qst_leaf_done_test", leafDonePost.PostID)
	}
	if leafDonePost.AuthorRole != model.PostRoleSystem {
		t.Fatalf("fanout_leaf_done author_role = %s, want system", leafDonePost.AuthorRole)
	}
	if leafDonePost.Content == "" || !strings.Contains(leafDonePost.Content, "success") {
		t.Fatalf("fanout_leaf_done content should mention success, got: %s", leafDonePost.Content)
	}
	// Should reference leaf's decision_note as causal ref.
	hasDecisionRef := false
	for _, ref := range leafDonePost.CausalRefs {
		if strings.HasPrefix(ref, "sys_outcome_") {
			hasDecisionRef = true
			break
		}
	}
	if !hasDecisionRef {
		t.Logf("warning: fanout_leaf_done causal_refs = %v, expected decision_note ref", leafDonePost.CausalRefs)
	}
}

func TestSaveQuest_FanoutLeafDone_Idempotent(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	qs := NewQuestStore(root)

	rootQ := &QuestMeta{
		ID:          "qst_root_idem",
		ShortID:     "ridem",
		Query:       "root for idempotent test",
		Type:        model.QuestTypeExecute,
		Status:      model.QuestStatusRunning,
		CreatedBy:   "user",
		GroupID:     "grp_idem",
		CreatedAtMs: model.NowMs(),
	}
	if err := qs.CreateQuest(rootQ); err != nil {
		t.Fatalf("CreateQuest root: %v", err)
	}

	leafQ := &QuestMeta{
		ID:           "qst_leaf_idem",
		ShortID:      "lidem",
		Query:        "leaf idem",
		Type:         model.QuestTypeExecute,
		Status:       model.QuestStatusSuccess,
		CreatedBy:    "adv_maker",
		GroupID:      "grp_idem",
		FanoutLeafID: "leaf-1",
		ParentQuestID: rootQ.ID,
		CreatedAtMs:  model.NowMs(),
		FinalVerdict: model.VerdictPass,
		CompletedAtMs: model.NowMs(),
	}
	if err := qs.CreateQuest(leafQ); err != nil {
		t.Fatalf("CreateQuest leaf: %v", err)
	}

	// Save twice — should not duplicate the fanout_leaf_done post.
	if err := qs.SaveQuest(leafQ); err != nil {
		t.Fatalf("SaveQuest 1: %v", err)
	}
	if err := qs.SaveQuest(leafQ); err != nil {
		t.Fatalf("SaveQuest 2: %v", err)
	}

	rootPosts, _ := qs.LoadThreadPosts(rootQ.ID)
	count := 0
	for _, p := range rootPosts {
		if p.Kind == "fanout_leaf_done" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected 1 fanout_leaf_done post after two saves, got %d", count)
	}
}

func TestSaveQuest_FanoutLeafDone_NonLeafNoNotification(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	qs := NewQuestStore(root)

	// Normal quest (no GroupID, not a leaf) reaching terminal.
	q := &QuestMeta{
		ID:          "qst_normal_terminal",
		ShortID:     "nterm",
		Query:       "normal quest",
		Type:        model.QuestTypeExecute,
		Status:      model.QuestStatusSuccess,
		CreatedBy:   "user",
		CreatedAtMs: model.NowMs(),
	}
	if err := qs.CreateQuest(q); err != nil {
		t.Fatalf("CreateQuest: %v", err)
	}
	if err := qs.SaveQuest(q); err != nil {
		t.Fatalf("SaveQuest: %v", err)
	}

	posts, _ := qs.LoadThreadPosts(q.ID)
	for _, p := range posts {
		if p.Kind == "fanout_leaf_done" {
			t.Fatal("non-leaf quest should NOT produce fanout_leaf_done post")
		}
	}
}

func TestSaveQuest_SystemPost_SourceEventID_Blocked(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	qs := NewQuestStore(root)
	q := &QuestMeta{
		ID:          "qst_blocked_evt",
		ShortID:     "blk",
		Query:       "blocked with event",
		Type:        model.QuestTypeExecute,
		Status:      model.QuestStatusRunning,
		CreatedBy:   "user",
		CreatedAtMs: model.NowMs(),
	}
	if err := qs.CreateQuest(q); err != nil {
		t.Fatalf("CreateQuest: %v", err)
	}

	blockedEvent := &QuestEventRow{
		Timestamp: model.NowMs(),
		Type:      "quest.blocked",
		QuestID:   q.ID,
		Payload:   map[string]any{"reason_code": "agent_error_auth"},
	}
	if err := qs.AppendEvent(q.ID, blockedEvent); err != nil {
		t.Fatalf("AppendEvent: %v", err)
	}

	q.Status = model.QuestStatusBlocked
	q.BlockedReason = "auth failed"
	q.BlockedReasonCode = "agent_error_auth"
	if err := qs.SaveQuest(q); err != nil {
		t.Fatalf("SaveQuest blocked: %v", err)
	}

	posts, err := qs.LoadThreadPosts(q.ID)
	if err != nil {
		t.Fatalf("LoadThreadPosts: %v", err)
	}
	var escalationPost *ThreadPost
	for i := range posts {
		if posts[i].Kind == "system_escalation" {
			escalationPost = &posts[i]
			break
		}
	}
	if escalationPost == nil {
		t.Fatal("no system_escalation post found")
	}
	if escalationPost.SourceEventID != blockedEvent.ID {
		t.Fatalf("system_escalation SourceEventID = %d, want %d", escalationPost.SourceEventID, blockedEvent.ID)
	}
}

func TestSaveQuest_SystemPost_SourceEventID_NoEventFallback(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	qs := NewQuestStore(root)
	q := &QuestMeta{
		ID:          "qst_no_evt",
		ShortID:     "noevt",
		Query:       "no events fallback",
		Type:        model.QuestTypeExecute,
		Status:      model.QuestStatusRunning,
		CreatedBy:   "user",
		CreatedAtMs: model.NowMs(),
	}
	if err := qs.CreateQuest(q); err != nil {
		t.Fatalf("CreateQuest: %v", err)
	}

	// No events appended — this simulates a legacy quest with empty event log.
	q.Status = model.QuestStatusBlocked
	q.BlockedReason = "legacy blocked"
	q.BlockedReasonCode = "legacy"
	if err := qs.SaveQuest(q); err != nil {
		t.Fatalf("SaveQuest blocked: %v", err)
	}

	posts, err := qs.LoadThreadPosts(q.ID)
	if err != nil {
		t.Fatalf("LoadThreadPosts: %v", err)
	}
	var escalationPost *ThreadPost
	for i := range posts {
		if posts[i].Kind == "system_escalation" {
			escalationPost = &posts[i]
			break
		}
	}
	if escalationPost == nil {
		t.Fatal("no system_escalation post found")
	}
	// No events → SourceEventID is allowed to be 0 as best-effort fallback.
	if escalationPost.SourceEventID != 0 {
		t.Fatalf("no-event fallback SourceEventID = %d, want 0", escalationPost.SourceEventID)
	}
}

func TestAuditThreadLedgerRepairsRootAndTerminalDecision(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	qs := NewQuestStore(root)
	q := &QuestMeta{
		ID:            "qst_ledger_repair",
		ShortID:       "ledger",
		Query:         "ledger repair",
		Type:          model.QuestTypeExecute,
		Status:        model.QuestStatusSuccess,
		FinalVerdict:  model.VerdictPass,
		FinalComment:  "done",
		FinalizedBy:   "user",
		CreatedBy:     "user",
		CreatedAtMs:   1000,
		CompletedAtMs: 2000,
	}
	if err := qs.CreateQuest(q); err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}

	audit, err := qs.AuditThreadLedger(q.ID, false)
	if err != nil {
		t.Fatalf("AuditThreadLedger failed: %v", err)
	}
	if audit.OK || len(audit.Findings) != 3 {
		t.Fatalf("audit should find missing root/outcome/decision: %+v", audit)
	}

	repaired, err := qs.AuditThreadLedger(q.ID, true)
	if err != nil {
		t.Fatalf("repair audit failed: %v", err)
	}
	if !repaired.OK || len(repaired.Findings) != 0 || len(repaired.Repaired) != 3 {
		t.Fatalf("repair should leave clean ledger and record repairs: %+v", repaired)
	}
	got, err := qs.LoadQuest(q.ID)
	if err != nil {
		t.Fatalf("LoadQuest failed: %v", err)
	}
	if got.PinnedOutcomeSummary == "" || !strings.Contains(got.PinnedOutcomeSummary, "Outcome: success") {
		t.Fatalf("pinned outcome not repaired: %+v", got)
	}
	posts, err := qs.LoadThreadPosts(q.ID)
	if err != nil {
		t.Fatalf("LoadThreadPosts failed: %v", err)
	}
	if len(posts) != 2 || posts[0].PostID != RootPostID(q.ID) || posts[1].Kind != "decision_note" {
		t.Fatalf("repair should create root + decision note: %+v", posts)
	}
}

func TestAuditThreadLedgerReportsNonRepairableFindings(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	qs := NewQuestStore(root)
	q := &QuestMeta{
		ID:          "qst_ledger_bad",
		ShortID:     "badledger",
		Query:       "bad ledger",
		Type:        model.QuestTypeExecute,
		Status:      model.QuestStatusRunning,
		CreatedBy:   "user",
		CreatedAtMs: 1000,
	}
	if err := qs.CreateQuest(q); err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	if _, err := qs.EnsureRootThreadPost(q); err != nil {
		t.Fatalf("EnsureRootThreadPost failed: %v", err)
	}
	bad := &ThreadPost{
		PostID:      RootPostID(q.ID),
		ThreadID:    q.ID,
		RootPostID:  RootPostID(q.ID),
		AuthorRole:  model.PostRoleChecker,
		Kind:        "review_report",
		Content:     "checker attached delivery",
		CreatedAtMs: 1100,
		ArtifactRefs: []ThreadArtifactRef{{
			ID:           "delivery_1",
			ArtifactType: model.ArtifactTypeDelivery,
		}},
	}
	if err := AppendJSONL(qs.threadPostsPath(q.ID), bad); err != nil {
		t.Fatalf("append bad post failed: %v", err)
	}

	audit, err := qs.AuditThreadLedger(q.ID, true)
	if err != nil {
		t.Fatalf("AuditThreadLedger failed: %v", err)
	}
	if audit.OK {
		t.Fatalf("bad ledger should not be OK: %+v", audit)
	}
	if len(audit.Repaired) != 0 {
		t.Fatalf("non-repairable findings should not be repaired: %+v", audit)
	}
	codes := map[string]bool{}
	for _, finding := range audit.Findings {
		codes[finding.Code] = true
	}
	if !codes["duplicate_post_id"] || !codes["invalid_authority"] {
		t.Fatalf("expected duplicate_post_id and invalid_authority findings: %+v", audit.Findings)
	}
}

func TestQuestPipelineFieldsBackfilledForLegacyQuest(t *testing.T) {
	q := &QuestMeta{
		ID:        "qst_pipeline_legacy",
		ShortID:   "pipeline",
		Query:     "legacy",
		Type:      model.QuestTypeExecute,
		Status:    model.QuestStatusReviewing,
		WarriorID: "warrior",
		MageID:    "mage",
	}
	q.EnsurePhases()
	if q.PipelineName != DefaultPipelineName {
		t.Fatalf("pipeline_name = %q, want %q", q.PipelineName, DefaultPipelineName)
	}
	if q.PipelineVersion != DefaultPipelineVersion {
		t.Fatalf("pipeline_version = %d, want %d", q.PipelineVersion, DefaultPipelineVersion)
	}
	if q.PhaseCount != 2 || len(q.PipelineDef) != 2 || len(q.Phases) != 2 {
		t.Fatalf("bad pipeline fields: phase_count=%d def=%d phases=%d", q.PhaseCount, len(q.PipelineDef), len(q.Phases))
	}
	if q.CurrentPhaseIdx() != 1 {
		t.Fatalf("current_phase_idx = %d, want 1", q.CurrentPhaseIdx())
	}
	if q.PipelineDefHash == "" {
		t.Fatal("pipeline_def_hash should be populated")
	}
}

func TestQuestPipelineValidationRejectsPhaseCountMismatch(t *testing.T) {
	q := &QuestMeta{
		ID:              "qst_pipeline_bad",
		ShortID:         "pipeline_bad",
		Query:           "bad",
		Type:            model.QuestTypeExecute,
		Status:          model.QuestStatusRunning,
		PipelineName:    DefaultPipelineName,
		PipelineVersion: DefaultPipelineVersion,
		PhaseCount:      3,
		CurrentPhase:    0,
		Phases: []PhaseTask{{
			PhaseIdx: 0,
			Role:     "warrior",
			Class:    model.ClassWarrior,
			Status:   model.PhaseRunning,
		}, {
			PhaseIdx: 1,
			Role:     "mage",
			Class:    model.ClassMage,
			Status:   model.PhasePending,
		}},
		PipelineDef: defaultPhaseTasks("", ""),
	}
	q.PipelineDefHash = phaseDefHash(q.PipelineDef)
	if err := q.ValidatePipelineState(); err == nil {
		t.Fatal("ValidatePipelineState should reject phase_count mismatch")
	}
}

func TestQuestAgentBindingFieldsRoundTripJSON(t *testing.T) {
	raw := []byte(`{
		"id":"qst_agent_binding",
		"short_id":"agent_binding",
		"query":"bind agents",
		"type":"execute",
		"status":"pending",
		"execute_agent_id":"traex",
		"review_agent_id":"claude_cli",
		"phases":[
			{"phase_idx":0,"role":"warrior","class":"warrior","agent_id":"phase_execute","status":"pending"},
			{"phase_idx":1,"role":"mage","class":"mage","agent_id":"phase_review","status":"pending"}
		]
	}`)
	var q QuestMeta
	if err := json.Unmarshal(raw, &q); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if q.ExecuteAgentID != "traex" || q.ReviewAgentID != "claude_cli" {
		t.Fatalf("quest agent ids = %q/%q", q.ExecuteAgentID, q.ReviewAgentID)
	}
	if len(q.Phases) != 2 || q.Phases[0].AgentID != "phase_execute" || q.Phases[1].AgentID != "phase_review" {
		t.Fatalf("phase agent ids not preserved: %+v", q.Phases)
	}
	out, err := json.Marshal(&q)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, want := range []string{`"execute_agent_id":"traex"`, `"review_agent_id":"claude_cli"`, `"agent_id":"phase_execute"`, `"agent_id":"phase_review"`} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("marshaled quest missing %s: %s", want, string(out))
		}
	}
}

func TestQuestSyncPhaseIDsSyncsAgentBindings(t *testing.T) {
	q := &QuestMeta{Phases: []PhaseTask{
		{AdventurerID: "warrior", AgentID: "execute_agent"},
		{AdventurerID: "mage", AgentID: "review_agent"},
	}}
	q.SyncPhaseIDs()
	if q.WarriorID != "warrior" || q.MageID != "mage" {
		t.Fatalf("legacy adventurers = %q/%q", q.WarriorID, q.MageID)
	}
	if q.ExecuteAgentID != "execute_agent" || q.ReviewAgentID != "review_agent" {
		t.Fatalf("agent bindings = %q/%q", q.ExecuteAgentID, q.ReviewAgentID)
	}
}

func TestQuestAgentBindingsSurviveDomainRoundTrip(t *testing.T) {
	q := &QuestMeta{
		ID:             "qst_domain_agent_binding",
		ShortID:        "domain_agent_binding",
		Query:          "bind",
		Type:           model.QuestTypeExecute,
		Status:         model.QuestStatusPending,
		WarriorID:      "warrior",
		MageID:         "mage",
		ExecuteAgentID: "execute_agent",
		ReviewAgentID:  "review_agent",
		Phases: []PhaseTask{
			{PhaseIdx: 0, AdventurerID: "warrior", AgentID: "phase_execute", Status: model.PhasePending},
			{PhaseIdx: 1, AdventurerID: "mage", AgentID: "phase_review", Status: model.PhasePending},
		},
		PipelineDef: []PhaseTask{
			{PhaseIdx: 0, Role: "warrior", Class: model.ClassWarrior, Goal: "execute", AgentID: "def_execute"},
			{PhaseIdx: 1, Role: "mage", Class: model.ClassMage, Goal: "review", AgentID: "def_review"},
		},
	}
	roundTrip := QuestMetaFromDomain(QuestMetaToDomain(q))
	if roundTrip.ExecuteAgentID != "phase_execute" || roundTrip.ReviewAgentID != "phase_review" {
		t.Fatalf("quest agent bindings = %q/%q", roundTrip.ExecuteAgentID, roundTrip.ReviewAgentID)
	}
	if len(roundTrip.Phases) != 2 || roundTrip.Phases[0].AgentID != "phase_execute" || roundTrip.Phases[1].AgentID != "phase_review" {
		t.Fatalf("phase agent bindings = %+v", roundTrip.Phases)
	}
	if len(roundTrip.PipelineDef) != 2 || roundTrip.PipelineDef[0].AgentID != "def_execute" || roundTrip.PipelineDef[1].AgentID != "def_review" {
		t.Fatalf("pipeline agent bindings = %+v", roundTrip.PipelineDef)
	}
}

func TestQuestStatus_InvalidTransitions(t *testing.T) {
	cases := []struct {
		name string
		from model.QuestStatus
		to   model.QuestStatus
	}{
		{"pending→success", model.QuestStatusPending, model.QuestStatusSuccess},
		{"success→failed", model.QuestStatusSuccess, model.QuestStatusFailed},
		{"pending→user_review", model.QuestStatusPending, model.QuestStatusUserReview},
		{"failed→running", model.QuestStatusFailed, model.QuestStatusRunning},
		{"blocked→reviewing", model.QuestStatusBlocked, model.QuestStatusReviewing},
	}
	for _, c := range cases {
		q := &QuestMeta{Status: c.from}
		if q.CanTransitionTo(c.to) {
			t.Errorf("%s: CanTransitionTo should be false", c.name)
		}
		if err := q.TransitionTo(c.to); err == nil {
			t.Errorf("%s: TransitionTo should fail", c.name)
		}
		// 状态不变
		if q.Status != c.from {
			t.Errorf("%s: status should remain %s, got %s", c.name, c.from, q.Status)
		}
	}
}

func TestQuestStatus_ReworkTransitions(t *testing.T) {
	// reviewing → running (rework)
	q := &QuestMeta{Status: model.QuestStatusReviewing}
	if err := q.TransitionTo(model.QuestStatusRunning); err != nil {
		t.Errorf("reviewing→running (rework) should be ok, got %v", err)
	}
	// user_review → running (rework)
	q.Status = model.QuestStatusUserReview
	if err := q.TransitionTo(model.QuestStatusRunning); err != nil {
		t.Errorf("user_review→running (rework) should be ok, got %v", err)
	}
	q.Status = model.QuestStatusUserReview
	if err := q.TransitionTo(model.QuestStatusBlocked); err != nil {
		t.Errorf("user_review→blocked should be ok, got %v", err)
	}
}

func TestQuestStatus_BlockedTransitions(t *testing.T) {
	// running → blocked
	q := &QuestMeta{Status: model.QuestStatusRunning}
	if err := q.TransitionTo(model.QuestStatusBlocked); err != nil {
		t.Errorf("running→blocked should be ok, got %v", err)
	}
	// blocked → running (unblock)
	if err := q.TransitionTo(model.QuestStatusRunning); err != nil {
		t.Errorf("blocked→running should be ok, got %v", err)
	}
	// blocked → cancelled
	q.Status = model.QuestStatusBlocked
	if err := q.TransitionTo(model.QuestStatusCancelled); err != nil {
		t.Errorf("blocked→cancelled should be ok, got %v", err)
	}
}

func TestQuestMeta_ResumeFromBlocked(t *testing.T) {
	q := &QuestMeta{
		Status:        model.QuestStatusBlocked,
		ResumeCount:   2,
		BlockedReason: "timeout",
	}
	if err := q.ResumeFromBlocked("继续执行"); err != nil {
		t.Fatalf("ResumeFromBlocked failed: %v", err)
	}
	if q.Status != model.QuestStatusRunning {
		t.Fatalf("status = %s, want running", q.Status)
	}
	if q.ResumeCount != 3 || q.BlockedReason != "" || q.ReviewHints != "继续执行" {
		t.Fatalf("unexpected quest fields: %+v", q)
	}
}

func TestQuestMeta_MoveBlockedToUserReview(t *testing.T) {
	q := &QuestMeta{Status: model.QuestStatusBlocked, BlockedReason: "need user"}
	if err := q.MoveBlockedToUserReview(""); err != nil {
		t.Fatalf("MoveBlockedToUserReview failed: %v", err)
	}
	if q.Status != model.QuestStatusUserReview {
		t.Fatalf("status = %s, want user_review", q.Status)
	}
	if q.FinalComment != "blocked 后由用户转入终审" || q.BlockedReason != "" {
		t.Fatalf("unexpected quest fields: %+v", q)
	}
}

func TestQuestMeta_CancelFromBlocked(t *testing.T) {
	q := &QuestMeta{Status: model.QuestStatusBlocked}
	if err := q.CancelFromBlocked("stop"); err != nil {
		t.Fatalf("CancelFromBlocked failed: %v", err)
	}
	if q.Status != model.QuestStatusCancelled || q.FinalComment != "stop" || q.CompletedAtMs == 0 {
		t.Fatalf("unexpected quest fields: %+v", q)
	}
}

func TestQuestMeta_UserReviewDecisions(t *testing.T) {
	t.Run("rework", func(t *testing.T) {
		q := &QuestMeta{Status: model.QuestStatusUserReview, ReworkCount: 1}
		if err := q.RequestUserReviewRework("fix it"); err != nil {
			t.Fatalf("RequestUserReviewRework failed: %v", err)
		}
		if q.Status != model.QuestStatusRunning || q.ReworkCount != 2 || q.ReviewHints != "fix it" {
			t.Fatalf("unexpected quest fields: %+v", q)
		}
	})

	t.Run("complete", func(t *testing.T) {
		q := &QuestMeta{Status: model.QuestStatusUserReview}
		if err := q.CompleteUserReview(model.VerdictPass, "ship it"); err != nil {
			t.Fatalf("CompleteUserReview failed: %v", err)
		}
		if q.Status != model.QuestStatusSuccess || q.FinalVerdict != model.VerdictPass || q.FinalComment != "ship it" || q.CompletedAtMs == 0 {
			t.Fatalf("unexpected quest fields: %+v", q)
		}
	})

	t.Run("reject", func(t *testing.T) {
		q := &QuestMeta{Status: model.QuestStatusUserReview}
		if err := q.CompleteUserReview(model.VerdictReject, "no"); err != nil {
			t.Fatalf("CompleteUserReview failed: %v", err)
		}
		if q.Status != model.QuestStatusFailed || q.FinalVerdict != model.VerdictReject || q.FinalComment != "no" || q.CompletedAtMs == 0 {
			t.Fatalf("unexpected quest fields: %+v", q)
		}
	})

	t.Run("request changes is not completion", func(t *testing.T) {
		q := &QuestMeta{Status: model.QuestStatusUserReview}
		if err := q.CompleteUserReview(model.VerdictRequestChange, "more work"); err == nil {
			t.Fatal("CompleteUserReview should reject request_changes")
		}
		if q.Status != model.QuestStatusUserReview || q.FinalVerdict != "" || q.CompletedAtMs != 0 {
			t.Fatalf("rejected completion should not mutate quest: %+v", q)
		}
	})
}

func TestQuestStore_AnswersAppendOnly(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	qs := NewQuestStore(root)
	q := &QuestMeta{
		ID:        "q_answers",
		Query:     "answer me",
		Type:      model.QuestTypeExecute,
		Status:    model.QuestStatusWaitingInput,
		CreatedBy: "user",
	}
	if err := qs.CreateQuest(q); err != nil {
		t.Fatal(err)
	}
	if err := qs.AppendAnswer(q.ID, &AnswerRecord{QuestionID: "question_1", Content: "first", Source: "test"}); err != nil {
		t.Fatal(err)
	}
	if err := qs.AppendAnswer(q.ID, &AnswerRecord{QuestionID: "question_1", Content: "second", Source: "test"}); err != nil {
		t.Fatal(err)
	}
	got, err := qs.LoadAnswers(q.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Content != "first" || got[1].Content != "second" || got[0].AnswerID == "" || got[1].AnswerID == "" {
		t.Fatalf("bad answers: %+v", got)
	}
}

func TestRecoveryStatePersists(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	state := &RecoveryState{
		QuestID:     "q_recovery",
		PolicyName:  "recovery_default_require_user",
		InputHash:   "sha256:test",
		LastAction:  "require_user",
		BlockReason: "blocked",
	}
	if err := root.SaveRecoveryState(state); err != nil {
		t.Fatal(err)
	}
	got, err := root.GetRecoveryState("q_recovery")
	if err != nil {
		t.Fatal(err)
	}
	if got.QuestID != state.QuestID || got.PolicyName != state.PolicyName || got.Status != RecoveryStatusPending || got.StartedAt.IsZero() {
		t.Fatalf("bad recovery state: %+v", got)
	}
}

func TestQuestMeta_BuildDesignDoc(t *testing.T) {
	q := &QuestMeta{
		ID:           "qst_design_doc",
		Query:        "设计权限系统",
		Type:         model.QuestTypeDesign,
		FinalComment: "采用 RBAC + policy guard",
	}
	doc := q.BuildDesignDoc("", q.FinalComment)

	if doc.QuestID != q.ID || doc.SchemaVersion != "gloop.design.v1" {
		t.Fatalf("unexpected design doc identity: %+v", doc)
	}
	if doc.Summary != q.FinalComment {
		t.Fatalf("summary = %q, want review comment fallback", doc.Summary)
	}
	if doc.OriginalQuery != q.Query || doc.ReviewComment != q.FinalComment {
		t.Fatalf("unexpected query/comment fields: %+v", doc)
	}
	if len(doc.Goals) == 0 || len(doc.ImplementationSteps) == 0 || len(doc.Risks) == 0 || len(doc.AcceptanceCriteria) == 0 {
		t.Fatalf("design doc should include structured sections: %+v", doc)
	}
	if doc.ExecutePrompt == "" || !strings.Contains(doc.ExecutePrompt, q.Query) || !strings.Contains(doc.ExecutePrompt, doc.Summary) {
		t.Fatalf("execute prompt should include original query and summary:\n%s", doc.ExecutePrompt)
	}
	if doc.CreatedAtMs == 0 {
		t.Fatalf("created_at_ms should be set: %+v", doc)
	}
}

func TestQuestStatus_SelfLoopNotAllowed(t *testing.T) {
	// v2 没有自环概念
	states := []model.QuestStatus{
		model.QuestStatusPending,
		model.QuestStatusRunning,
		model.QuestStatusReviewing,
		model.QuestStatusWaitingInput,
		model.QuestStatusUserReview,
		model.QuestStatusBlocked,
	}
	for _, s := range states {
		q := &QuestMeta{Status: s}
		if q.CanTransitionTo(s) {
			t.Errorf("%s self-loop should be false", s)
		}
	}
}

func TestQuestStatus_CancelledFromAnyNonTerminal(t *testing.T) {
	states := []model.QuestStatus{
		model.QuestStatusPending,
		model.QuestStatusRunning,
		model.QuestStatusReviewing,
		model.QuestStatusWaitingInput,
		model.QuestStatusUserReview,
		model.QuestStatusBlocked,
	}
	for _, s := range states {
		q := &QuestMeta{Status: s}
		if err := q.TransitionTo(model.QuestStatusCancelled); err != nil {
			t.Errorf("%s→cancelled should be ok, got %v", s, err)
		}
	}
}

func TestQuestStatus_TerminalStates(t *testing.T) {
	terminals := []model.QuestStatus{
		model.QuestStatusSuccess,
		model.QuestStatusFailed,
		model.QuestStatusCancelled,
	}
	for _, s := range terminals {
		q := &QuestMeta{Status: s}
		if !q.IsTerminal() {
			t.Errorf("%s should be terminal", s)
		}
	}
	nonTerminals := []model.QuestStatus{
		model.QuestStatusPending,
		model.QuestStatusRunning,
		model.QuestStatusReviewing,
		model.QuestStatusWaitingInput,
		model.QuestStatusUserReview,
		model.QuestStatusBlocked,
	}
	for _, s := range nonTerminals {
		q := &QuestMeta{Status: s}
		if q.IsTerminal() {
			t.Errorf("%s should not be terminal", s)
		}
	}
}

func TestReadContextPackRows(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	qs := NewQuestStore(root)
	q := &QuestMeta{ID: "qst_context_trace", ShortID: "ctx", Query: "demo", Type: model.QuestTypeExecute, Status: model.QuestStatusPending}
	if err := qs.CreateQuest(q); err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	if err := qs.AppendSessionRow(q.ID, "warrior_0", &QuestSessionRow{Kind: "message", Role: "user", Content: "hello"}); err != nil {
		t.Fatalf("AppendSessionRow message failed: %v", err)
	}
	meta := map[string]any{"context_pack": map[string]any{"kind": "quest_execution"}}
	if err := qs.AppendSessionRow(q.ID, "warrior_0", &QuestSessionRow{Kind: "context_pack", Role: "user", Meta: meta}); err != nil {
		t.Fatalf("AppendSessionRow context failed: %v", err)
	}
	rows, err := qs.ReadContextPackRows(q.ID, "warrior_0", 0)
	if err != nil {
		t.Fatalf("ReadContextPackRows failed: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
	if rows[0].Kind != "context_pack" {
		t.Fatalf("kind = %s, want context_pack", rows[0].Kind)
	}
}

func TestQuestEventsAssignMonotonicIDs(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	qs := NewQuestStore(root)
	q := &QuestMeta{ID: "qst_events", ShortID: "events", Query: "events", Type: model.QuestTypeExecute, Status: model.QuestStatusPending, CreatedBy: "user"}
	if err := qs.CreateQuest(q); err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	if err := qs.AppendEvent(q.ID, &QuestEventRow{Type: "quest.created"}); err != nil {
		t.Fatalf("AppendEvent 1 failed: %v", err)
	}
	if err := qs.AppendEvent(q.ID, &QuestEventRow{Type: "quest.started"}); err != nil {
		t.Fatalf("AppendEvent 2 failed: %v", err)
	}
	rows, err := qs.ReadEvents(q.ID, 0)
	if err != nil {
		t.Fatalf("ReadEvents failed: %v", err)
	}
	if len(rows) != 2 || rows[0].ID != 1 || rows[1].ID != 2 {
		t.Fatalf("event IDs = %+v, want 1/2", rows)
	}
}

func TestGlobalEventsAssignMonotonicIDs(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	if err := root.AppendGlobalEvent(&GlobalEventRow{Type: "quest.created", QuestID: "q1", EventID: 1}); err != nil {
		t.Fatalf("AppendGlobalEvent 1 failed: %v", err)
	}
	if err := root.AppendGlobalEvent(&GlobalEventRow{Type: "quest.created", QuestID: "q2", EventID: 1}); err != nil {
		t.Fatalf("AppendGlobalEvent 2 failed: %v", err)
	}
	rows, err := root.ReadGlobalEvents(0)
	if err != nil {
		t.Fatalf("ReadGlobalEvents failed: %v", err)
	}
	if len(rows) != 2 || rows[0].GlobalID != 1 || rows[1].GlobalID != 2 {
		t.Fatalf("global IDs = %+v, want 1/2", rows)
	}
}

func TestWorkflowInstanceStatePersistsGlobalCursor(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	state := &WorkflowInstanceState{
		ID:           "wf_1",
		WorkflowID:   "design_execute_test",
		Status:       "running",
		LastGlobalID: 42,
		QuestIDs:     []string{"q1", "q2"},
		Meta:         map[string]any{"stage": "execute"},
	}
	if err := root.SaveWorkflowInstance(state); err != nil {
		t.Fatalf("SaveWorkflowInstance failed: %v", err)
	}
	got, err := root.GetWorkflowInstance("wf_1")
	if err != nil {
		t.Fatalf("GetWorkflowInstance failed: %v", err)
	}
	if got.LastGlobalID != 42 || got.WorkflowID != "design_execute_test" || len(got.QuestIDs) != 2 || got.CreatedAtMs == 0 || got.UpdatedAtMs == 0 {
		t.Fatalf("bad workflow state: %+v", got)
	}
}

// P1-1：修复进度解析测试
func TestParseRepairProgress(t *testing.T) {
	tests := []struct {
		name         string
		comment      string
		wantRepaired int
		wantTotal    int
		wantHasData  bool
	}{
		{"标准格式", "修复进度: 3/5\n其他内容", 3, 5, true},
		{"中文冒号", "修复进度：2/4", 2, 4, true},
		{"带空格", "修复进度: 1 / 3", 1, 3, true},
		{"多行格式带未修复", "修复进度: 4/6\n未修复: 问题1\n新增: 问题2", 4, 6, true},
		{"空字符串", "", 0, 0, false},
		{"无修复进度", "这是一个普通评论", 0, 0, false},
		{"格式错误", "修复进度: 三/五", 0, 0, false},
		{"分母为0", "修复进度: 0/0", 0, 0, false},
		{"负数", "修复进度: -1/5", 0, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := ParseRepairProgress(tt.comment)
			if p.HasData != tt.wantHasData {
				t.Fatalf("HasData = %v, want %v", p.HasData, tt.wantHasData)
			}
			if !tt.wantHasData {
				return
			}
			if p.Repaired != tt.wantRepaired || p.Total != tt.wantTotal {
				t.Fatalf("Repaired=%d Total=%d, want %d/%d", p.Repaired, p.Total, tt.wantRepaired, tt.wantTotal)
			}
		})
	}
}

func TestParseRepairProgress_Rate(t *testing.T) {
	p := ParseRepairProgress("修复进度: 2/5")
	if !p.HasData {
		t.Fatal("expected HasData=true")
	}
	if p.Rate() != 0.4 {
		t.Fatalf("rate = %f, want 0.4", p.Rate())
	}

	// 无数据时返回 -1
	p2 := ParseRepairProgress("")
	if p2.Rate() != -1 {
		t.Fatalf("empty rate = %f, want -1", p2.Rate())
	}
}

func TestQuestRepositoryAdapter_ReviewSourceAndStructuredReviewRoundTrip(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	qs := NewQuestStore(root)
	q := &QuestMeta{
		ID:          "qst_review_roundtrip",
		ShortID:     "review_roundtrip",
		Query:       "review roundtrip",
		Type:        model.QuestTypeExecute,
		Status:      model.QuestStatusUserReview,
		CreatedBy:   "user",
		CreatedAtMs: model.NowMs(),
	}
	if err := qs.CreateQuest(q); err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	repo := NewQuestRepository(root)
	err = repo.AppendReview(q.ID, &quest.ReviewRecord{
		Ts:         123,
		Verdict:    model.VerdictPass,
		Comment:    "structured",
		ReviewedBy: "mage-1",
		Score:      9,
		Source: &quest.ReviewSource{
			SourceRole:         "mage",
			SourceClass:        "mage",
			SourcePhaseIdx:     1,
			SourceAdventurerID: "mage-1",
			SourceSessionID:    "mage_0",
		},
		StructuredReview: &quest.StructuredReview{
			SchemaVersion:           "v1",
			ReviewedWarriorPhaseIdx: 0,
			Disagreements: []quest.MageDisagreement{{
				ID:         "d1",
				Dimension:  quest.DimensionCorrectness,
				Severity:   quest.SeverityHigh,
				TargetKind: quest.EvidenceTargetArtifact,
				Claim:      "claim",
			}},
		},
	})
	if err != nil {
		t.Fatalf("AppendReview failed: %v", err)
	}
	reviews, err := repo.LoadReviews(q.ID)
	if err != nil {
		t.Fatalf("LoadReviews failed: %v", err)
	}
	if len(reviews) != 1 {
		t.Fatalf("reviews = %d, want 1", len(reviews))
	}
	got := reviews[0]
	if got.Source == nil || got.Source.SourceRole != "mage" || got.Source.SourceSessionID != "mage_0" {
		t.Fatalf("bad review source roundtrip: %+v", got.Source)
	}
	if got.StructuredReview == nil || len(got.StructuredReview.Disagreements) != 1 || got.StructuredReview.Disagreements[0].ID != "d1" {
		t.Fatalf("bad structured review roundtrip: %+v", got.StructuredReview)
	}
}

func TestAppendFanoutSummarySnapshot_WritesOnLeafTerminal(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	qs := NewQuestStore(root)
	groupID := "grp_test_summary"

	// Create root quest.
	rootQ := &QuestMeta{
		ID:          "qst_root_summary",
		ShortID:     "root",
		Query:       "fanout root",
		Type:        model.QuestTypeExecute,
		Status:      model.QuestStatusRunning,
		GroupID:     groupID,
		CreatedBy:   "user",
		CreatedAtMs: 1000,
	}
	if err := qs.CreateQuest(rootQ); err != nil {
		t.Fatalf("CreateQuest root: %v", err)
	}

	// Create a leaf quest.
	leaf := &QuestMeta{
		ID:            "qst_leaf_summary",
		ShortID:       "leaf",
		Query:         "fanout leaf",
		Type:          model.QuestTypeExecute,
		Status:        model.QuestStatusRunning,
		GroupID:       groupID,
		FanoutLeafID:  "leaf_1",
		ParentQuestID: "qst_root_summary",
		CreatedBy:     "user",
		CreatedAtMs:   2000,
	}
	if err := qs.CreateQuest(leaf); err != nil {
		t.Fatalf("CreateQuest leaf: %v", err)
	}

	// Ensure root has a thread post.
	qs.EnsureRootThreadPost(rootQ)

	// Transition leaf to success — should trigger fanout_leaf_done + fanout_summary.
	leaf.Status = model.QuestStatusSuccess
	leaf.UpdatedAtMs = 3000
	if err := qs.SaveQuest(leaf); err != nil {
		t.Fatalf("SaveQuest leaf terminal: %v", err)
	}

	// Verify fanout_summary post exists on root thread.
	posts, err := qs.LoadThreadPosts("qst_root_summary")
	if err != nil {
		t.Fatalf("LoadThreadPosts: %v", err)
	}
	var summaryPost *ThreadPost
	for i := range posts {
		if posts[i].Kind == "fanout_summary" {
			summaryPost = &posts[i]
			break
		}
	}
	if summaryPost == nil {
		t.Fatal("fanout_summary post should exist after leaf terminal")
	}
	if summaryPost.AuthorRole != model.PostRoleSystem {
		t.Errorf("summary author_role = %s, want %s", summaryPost.AuthorRole, model.PostRoleSystem)
	}
	if !strings.Contains(summaryPost.Content, "all terminal") {
		t.Errorf("summary content should mention all terminal, got %q", summaryPost.Content)
	}
	if !strings.HasPrefix(summaryPost.PostID, "sys_fanout_summary_"+groupID+"_") {
		t.Errorf("summary post_id = %s, want prefix sys_fanout_summary_%s_", summaryPost.PostID, groupID)
	}
	hasRootRef := false
	for _, ref := range summaryPost.CausalRefs {
		if ref == RootPostID("qst_root_summary") {
			hasRootRef = true
			break
		}
	}
	if !hasRootRef {
		t.Errorf("summary causal_refs should include root, got %v", summaryPost.CausalRefs)
	}
}

func TestAppendFanoutSummarySnapshot_SeqIncrements(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	qs := NewQuestStore(root)
	groupID := "grp_test_seq"

	rootQ := &QuestMeta{
		ID:          "qst_root_seq",
		ShortID:     "root",
		Query:       "seq test root",
		Type:        model.QuestTypeExecute,
		Status:      model.QuestStatusRunning,
		GroupID:     groupID,
		CreatedBy:   "user",
		CreatedAtMs: 1000,
	}
	qs.CreateQuest(rootQ)
	qs.EnsureRootThreadPost(rootQ)

	leaf1 := &QuestMeta{
		ID:           "qst_leaf1_seq",
		ShortID:      "l1",
		Query:        "leaf 1",
		Type:         model.QuestTypeExecute,
		Status:       model.QuestStatusRunning,
		GroupID:      groupID,
		FanoutLeafID: "leaf_1",
		CreatedBy:    "user",
		CreatedAtMs:  2000,
	}
	qs.CreateQuest(leaf1)

	leaf2 := &QuestMeta{
		ID:           "qst_leaf2_seq",
		ShortID:      "l2",
		Query:        "leaf 2",
		Type:         model.QuestTypeExecute,
		Status:       model.QuestStatusRunning,
		GroupID:      groupID,
		FanoutLeafID: "leaf_2",
		CreatedBy:    "user",
		CreatedAtMs:  3000,
	}
	qs.CreateQuest(leaf2)

	// Terminal leaf1 → summary seq 1.
	leaf1.Status = model.QuestStatusSuccess
	leaf1.UpdatedAtMs = 4000
	qs.SaveQuest(leaf1)

	// Terminal leaf2 → summary seq 2.
	leaf2.Status = model.QuestStatusFailed
	leaf2.UpdatedAtMs = 5000
	qs.SaveQuest(leaf2)

	posts, _ := qs.LoadThreadPosts("qst_root_seq")
	summaryCount := 0
	for _, p := range posts {
		if p.Kind == "fanout_summary" {
			summaryCount++
		}
	}
	if summaryCount != 2 {
		t.Fatalf("summary post count = %d, want 2 (one per terminal leaf)", summaryCount)
	}
}

func TestAppendFanoutSummarySnapshot_NoDuplicateOnRepeatedTerminalSave(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	qs := NewQuestStore(root)
	groupID := "grp_nodup"

	rootQ := &QuestMeta{
		ID:          "qst_root_nodup",
		ShortID:     "root",
		Query:       "no duplicate test",
		Type:        model.QuestTypeExecute,
		Status:      model.QuestStatusRunning,
		GroupID:     groupID,
		CreatedBy:   "user",
		CreatedAtMs: 1000,
	}
	qs.CreateQuest(rootQ)
	qs.EnsureRootThreadPost(rootQ)

	leaf := &QuestMeta{
		ID:           "qst_leaf_nodup",
		ShortID:      "leaf",
		Query:        "leaf",
		Type:         model.QuestTypeExecute,
		Status:       model.QuestStatusRunning,
		GroupID:      groupID,
		FanoutLeafID: "leaf_1",
		CreatedBy:    "user",
		CreatedAtMs:  2000,
	}
	qs.CreateQuest(leaf)

	// First terminal transition → should write summary.
	leaf.Status = model.QuestStatusSuccess
	leaf.UpdatedAtMs = 3000
	if err := qs.SaveQuest(leaf); err != nil {
		t.Fatalf("SaveQuest #1: %v", err)
	}

	// Second save while already terminal → should NOT write another summary.
	leaf.UpdatedAtMs = 4000
	if err := qs.SaveQuest(leaf); err != nil {
		t.Fatalf("SaveQuest #2: %v", err)
	}

	// Third save while still terminal → no summary.
	leaf.UpdatedAtMs = 5000
	if err := qs.SaveQuest(leaf); err != nil {
		t.Fatalf("SaveQuest #3: %v", err)
	}

	posts, _ := qs.LoadThreadPosts("qst_root_nodup")
	summaryCount := 0
	for _, p := range posts {
		if p.Kind == "fanout_summary" {
			summaryCount++
		}
	}
	if summaryCount != 1 {
		t.Fatalf("summary post count = %d, want 1 (only on first terminal transition)", summaryCount)
	}
}
