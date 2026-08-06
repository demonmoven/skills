package server

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"code.byted.org/lihuanyu.0w0/gloop/internal/auth"
	"code.byted.org/lihuanyu.0w0/gloop/internal/events"
	"code.byted.org/lihuanyu.0w0/gloop/internal/executor"
	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
	"code.byted.org/lihuanyu.0w0/gloop/internal/notifications"
	"code.byted.org/lihuanyu.0w0/gloop/internal/orchestrator"
	"code.byted.org/lihuanyu.0w0/gloop/internal/policy"
)

const testToken = "test-token"

type testBlockingExecutor struct {
	*executor.MockExecutor
	block chan struct{}
}

func (e *testBlockingExecutor) SendMessage(ctx context.Context, sessionID string, msg executor.Message, modelName string, stream chan<- executor.StreamChunk) (*executor.ChatResponse, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-e.block:
		return e.MockExecutor.SendMessage(ctx, sessionID, msg, modelName, stream)
	}
}

func newTestServer(t *testing.T) (*Server, http.Handler) {
	t.Helper()
	return newTestServerWithConfig(t, nil)
}

func newTestServerWithConfig(t *testing.T, mutate func(*fsstore.GlobalConfig)) (*Server, http.Handler) {
	t.Helper()

	root, err := fsstore.Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open root: %v", err)
	}
	cfg := fsstore.DefaultGlobalConfig()
	cfg.DefaultWorkingDir = t.TempDir()
	if mutate != nil {
		mutate(cfg)
	}
	bus := events.NewBus()
	eng, err := orchestrator.NewEngine(root, cfg, bus, cfg.DefaultWorkingDir, slog.Default(), orchestrator.WithNotifier(notifications.NewNoopNotifier()))
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	t.Cleanup(func() { eng.Shutdown(2 * time.Second) })

	streams, stopSSE := context.WithCancel(context.Background())
	s := &Server{
		cfg:     cfg,
		engine:  eng,
		bus:     bus,
		root:    root,
		Token:   &auth.TokenStore{Token: testToken},
		streams: streams,
		stopSSE: stopSSE,
	}
	return s, s.buildRouter()
}

func configureRunnableAdventurers(t *testing.T, s *Server) {
	t.Helper()
	if err := s.root.SaveAgent(&fsstore.AgentConfig{
		Name:         "test_agent",
		Type:         model.AgentTypeCLI,
		Command:      "test-agent",
		DefaultModel: "mock-fast",
		Enabled:      true,
	}); err != nil {
		t.Fatalf("SaveAgent failed: %v", err)
	}
	s.engine.RegisterExecutor("test_agent", executor.NewMockExecutor("exe_test_agent"))
	items := []*fsstore.AdventurerFile{
		{
			ID:     "adv_test_warrior",
			Name:   "test warrior",
			Class:  model.ClassWarrior,
			Status: model.AdventurerActive,
			Agent:  "test_agent",
			Level:  1,
		},
		{
			ID:     "adv_test_mage",
			Name:   "test mage",
			Class:  model.ClassMage,
			Status: model.AdventurerActive,
			Agent:  "test_agent",
			Level:  1,
		},
	}
	for _, item := range items {
		if _, err := s.engine.SaveAdventurer(item); err != nil {
			t.Fatalf("SaveAdventurer(%s) failed: %v", item.ID, err)
		}
	}
}

func authedReq(method, target, body string) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+testToken)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}

func decodeBody(t *testing.T, rr *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response body %q: %v", rr.Body.String(), err)
	}
	return body
}

func TestHealthzBypassesAuth(t *testing.T) {
	_, h := newTestServer(t)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	body := decodeBody(t, rr)
	if body["ok"] != true || body["service"] != "Gloop" {
		t.Fatalf("unexpected health body: %#v", body)
	}
}

func TestAPIRoutesRequireAuth(t *testing.T) {
	_, h := newTestServer(t)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/quests", nil))

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusUnauthorized, rr.Body.String())
	}
	body := decodeBody(t, rr)
	if body["ok"] != false || body["error"] != "missing local token" {
		t.Fatalf("unexpected 401 body: %#v", body)
	}
}

func TestDashboardAssetsBypassAuth(t *testing.T) {
	_, h := newTestServer(t)

	assetPath := findAnyAssetPath(t, h)
	if assetPath == "" {
		t.Skip("no embedded dashboard assets found in this build; skipping auth-bypass test")
	}

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, assetPath, nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("GET %s: status = %d, want %d; body=%s", assetPath, rr.Code, http.StatusOK, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "text/javascript") {
		t.Fatalf("GET %s: content-type = %q, want javascript", assetPath, ct)
	}
	if strings.Contains(rr.Body.String(), "missing local token") {
		t.Fatalf("GET %s: asset response should not be auth error: %s", assetPath, rr.Body.String())
	}
}

// findAnyAssetPath 从嵌入的 Dashboard 资源里挑一个存在的 *.js 路径，避免写死 vite content hash。
func findAnyAssetPath(t *testing.T, h http.Handler) string {
	// 先直接探测常见名字
	for _, p := range []string{
		"/assets/index-CH26mUv2.js",
		"/assets/index-DdQdzK9P.js",
		"/assets/index-5-gd3pkJ.js",
		"/assets/MarkdownRenderer-CEJea1je.js",
	} {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, p, nil))
		if rr.Code == http.StatusOK {
			return p
		}
	}
	return ""
}

func TestEngineRoutesRejectMissingEngine(t *testing.T) {
	root, err := fsstore.Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open root: %v", err)
	}
	s := &Server{
		root:  root,
		bus:   events.NewBus(),
		Token: &auth.TokenStore{Token: testToken},
	}

	rr := httptest.NewRecorder()
	s.buildRouter().ServeHTTP(rr, authedReq(http.MethodGet, "/api/quests", ""))

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusInternalServerError, rr.Body.String())
	}
	body := decodeBody(t, rr)
	if body["ok"] != false || body["error"] != "engine 未初始化" {
		t.Fatalf("unexpected missing-engine body: %#v", body)
	}
}

func TestUnknownAPIRouteDoesNotServeDashboardHTML(t *testing.T) {
	_, h := newTestServer(t)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodGet, "/api/not-a-real-route", ""))

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusNotFound, rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), "<!doctype html>") {
		t.Fatalf("unknown api route served dashboard html: %s", rr.Body.String())
	}
	body := decodeBody(t, rr)
	if body["ok"] != false {
		t.Fatalf("unexpected api fallback body: %#v", body)
	}
}

func TestExecutorsDoesNotRequireEngine(t *testing.T) {
	root, err := fsstore.Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open root: %v", err)
	}
	s := &Server{
		root:  root,
		bus:   events.NewBus(),
		Token: &auth.TokenStore{Token: testToken},
	}

	rr := httptest.NewRecorder()
	s.buildRouter().ServeHTTP(rr, authedReq(http.MethodGet, "/api/executors", ""))

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	body := decodeBody(t, rr)
	if body["ok"] != true {
		t.Fatalf("unexpected executors body: %#v", body)
	}
}

func TestSkillsDoesNotRequireEngine(t *testing.T) {
	root, err := fsstore.Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open root: %v", err)
	}
	s := &Server{
		root:  root,
		bus:   events.NewBus(),
		Token: &auth.TokenStore{Token: testToken},
	}

	rr := httptest.NewRecorder()
	s.buildRouter().ServeHTTP(rr, authedReq(http.MethodGet, "/api/skills", ""))

	if rr.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	body := decodeBody(t, rr)
	items, ok := body["items"].([]any)
	if body["ok"] != true || !ok || len(items) == 0 {
		t.Fatalf("unexpected skills body: %#v", body)
	}

	first, ok := items[0].(map[string]any)
	if !ok || first["name"] == "" {
		t.Fatalf("unexpected first skill: %#v", items[0])
	}

	rr = httptest.NewRecorder()
	s.buildRouter().ServeHTTP(rr, authedReq(http.MethodGet, "/api/skills/"+first["name"].(string), ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("get status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	detail := decodeBody(t, rr)
	item, ok := detail["item"].(map[string]any)
	if detail["ok"] != true || !ok || item["body"] == "" || item["formatted"] == "" {
		t.Fatalf("unexpected skill detail: %#v", detail)
	}
}

func TestConfigPackageExportPreviewImport(t *testing.T) {
	_, h := newTestServer(t)

	exportRR := httptest.NewRecorder()
	h.ServeHTTP(exportRR, authedReq(http.MethodGet, "/api/settings/package/export", ""))
	if exportRR.Code != http.StatusOK {
		t.Fatalf("export status = %d, body=%s", exportRR.Code, exportRR.Body.String())
	}
	var pkg fsstore.ConfigPackage
	if err := json.Unmarshal(exportRR.Body.Bytes(), &pkg); err != nil {
		t.Fatalf("decode package: %v", err)
	}
	if pkg.Kind != fsstore.ConfigPackageKind || pkg.Sections.Config == nil {
		t.Fatalf("unexpected package: %#v", pkg)
	}
	pkg.Sections.Config.MaxTurnsPerPhase = 9
	raw, err := json.Marshal(pkg)
	if err != nil {
		t.Fatalf("marshal package: %v", err)
	}

	previewRR := httptest.NewRecorder()
	h.ServeHTTP(previewRR, authedReq(http.MethodPost, "/api/settings/package/preview", string(raw)))
	if previewRR.Code != http.StatusOK {
		t.Fatalf("preview status = %d, body=%s", previewRR.Code, previewRR.Body.String())
	}
	previewBody := decodeBody(t, previewRR)
	if previewBody["ok"] != true || previewBody["preview"] == nil {
		t.Fatalf("unexpected preview body: %#v", previewBody)
	}

	importRR := httptest.NewRecorder()
	h.ServeHTTP(importRR, authedReq(http.MethodPost, "/api/settings/package/import", string(raw)))
	if importRR.Code != http.StatusOK {
		t.Fatalf("import status = %d, body=%s", importRR.Code, importRR.Body.String())
	}
	importBody := decodeBody(t, importRR)
	cfg := importBody["config"].(map[string]any)
	if int(cfg["max_turns_per_phase"].(float64)) != 9 {
		t.Fatalf("max_turns_per_phase = %#v", cfg["max_turns_per_phase"])
	}
}

func TestQuestListCreateAndGet(t *testing.T) {
	s, h := newTestServer(t)
	configureRunnableAdventurers(t, s)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodGet, "/api/quests", ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	listBody := decodeBody(t, rr)
	if listBody["ok"] != true {
		t.Fatalf("unexpected list body: %#v", listBody)
	}
	if items, ok := listBody["items"].([]any); !ok || len(items) != 0 {
		t.Fatalf("items = %#v, want empty array", listBody["items"])
	}

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodPost, "/api/quests", `{"query":"写一个 handler 测试","type":"design","workspace_mode":"readonly"}`))
	if rr.Code != http.StatusOK {
		t.Fatalf("create status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	createBody := decodeBody(t, rr)
	if createBody["ok"] != true || createBody["qid"] == "" || createBody["short_id"] == "" {
		t.Fatalf("unexpected create body: %#v", createBody)
	}
	qid, _ := createBody["qid"].(string)

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodGet, "/api/quests/"+qid, ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("get status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	getBody := decodeBody(t, rr)
	quest, ok := getBody["quest"].(map[string]any)
	if !ok {
		t.Fatalf("quest body missing: %#v", getBody)
	}
	if quest["id"] != qid || quest["type"] != string(model.QuestTypeDesign) || quest["workspace_mode"] != string(model.WorkspaceReadOnly) {
		t.Fatalf("unexpected quest: %#v", quest)
	}
	if quest["effect_type"] != "none" || quest["workspace_diff_pending"] != false {
		t.Fatalf("unexpected quest effect contract: effect_type=%#v workspace_diff_pending=%#v", quest["effect_type"], quest["workspace_diff_pending"])
	}
}

func TestQuestResponseEffectContract(t *testing.T) {
	s, _ := newTestServer(t)

	contextQuest := &fsstore.QuestMeta{
		ID:        "qst_context",
		Status:    model.QuestStatusSuccess,
		CreatedBy: model.QuestSourcePrefix + "auto_context_refresh",
	}
	got := s.questResponse(contextQuest)
	if got["effect_type"] != "context_store" || got["workspace_diff_pending"] != false {
		t.Fatalf("context quest effect contract = %#v", got)
	}

	diffQuest := &fsstore.QuestMeta{
		ID:               "qst_diff",
		Status:           model.QuestStatusSuccess,
		CreatedBy:        string(model.SourceUser),
		DiffChangedFiles: 2,
	}
	got = s.questResponse(diffQuest)
	if got["effect_type"] != "workspace_diff" || got["workspace_diff_pending"] != true {
		t.Fatalf("diff quest effect contract = %#v", got)
	}

	appliedQuest := &fsstore.QuestMeta{
		ID:               "qst_applied",
		Status:           model.QuestStatusSuccess,
		CreatedBy:        string(model.SourceUser),
		Applied:          true,
		DiffChangedFiles: 2,
	}
	got = s.questResponse(appliedQuest)
	if got["workspace_diff_pending"] != false {
		t.Fatalf("applied quest should not have pending workspace diff: %#v", got)
	}

	applyFailedQuest := &fsstore.QuestMeta{
		ID:               "qst_apply_failed",
		Status:           model.QuestStatusSuccess,
		CreatedBy:        string(model.SourceUser),
		ApplyStatus:      model.ApplyStatusFailed,
		ApplyError:       "apply failed",
		DiffChangedFiles: 2,
	}
	got = s.questResponse(applyFailedQuest)
	if got["apply_failed"] != true || got["workspace_diff_pending"] != false {
		t.Fatalf("apply failed quest should expose failed state without pending diff: %#v", got)
	}

	applyPendingQuest := &fsstore.QuestMeta{
		ID:               "qst_apply_pending",
		Status:           model.QuestStatusSuccess,
		CreatedBy:        string(model.SourceUser),
		ApplyStatus:      model.ApplyStatusPending,
		DiffChangedFiles: 2,
	}
	got = s.questResponse(applyPendingQuest)
	if got["workspace_diff_pending"] != false {
		t.Fatalf("apply pending quest should not expose pending workspace diff: %#v", got)
	}

	if err := s.root.SaveAutomation(&fsstore.AutomationConfig{
		ID:      "external_auto",
		Name:    "external",
		Enabled: true,
		Query:   "sync external system",
		AllowL2: true,
		Tags:    []string{"official", "template"},
	}); err != nil {
		t.Fatalf("SaveAutomation failed: %v", err)
	}
	externalQuest := &fsstore.QuestMeta{
		ID:        "qst_external",
		Status:    model.QuestStatusSuccess,
		CreatedBy: model.QuestSourcePrefix + "external_auto",
	}
	got = s.questResponse(externalQuest)
	if got["effect_type"] != "external_side_effect" || got["workspace_diff_pending"] != false {
		t.Fatalf("external side-effect quest contract = %#v", got)
	}
}

func TestQuestResponsePolicyView(t *testing.T) {
	s, _ := newTestServer(t)
	if err := s.root.SaveAutomation(&fsstore.AutomationConfig{
		ID:        "auto_policy_view",
		Name:      "policy view",
		Enabled:   true,
		Query:     "policy view",
		TrustTier: fsstore.TrustTier2,
	}); err != nil {
		t.Fatalf("SaveAutomation failed: %v", err)
	}
	if _, err := s.root.EnsureAutomationTrustState("auto_policy_view"); err != nil {
		t.Fatalf("EnsureAutomationTrustState failed: %v", err)
	}
	trust, err := s.root.LockAutomationTrustTier("auto_policy_view", fsstore.TrustTier3, "test")
	if err != nil || trust.Tier != fsstore.TrustTier3 {
		t.Fatalf("LockAutomationTrustTier failed: trust=%+v err=%v", trust, err)
	}
	q := &fsstore.QuestMeta{
		ID:                 "qst_policy_view",
		Status:             model.QuestStatusBlocked,
		CreatedBy:          model.QuestSourcePrefix + "auto_policy_view",
		DiffChangedFiles:   2,
		BlockedReasonCode:  "no_progress",
		BlockedReason:      "no progress",
		PolicyDecisionID:   "pol_dec_test",
		AutoPassedByPolicy: "context_store_automation_auto_pass",
	}
	if err := fsstore.NewQuestStore(s.root).CreateQuest(q); err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	if err := s.root.SaveRecoveryState(&fsstore.RecoveryState{
		QuestID:      q.ID,
		Status:       fsstore.RecoveryStatusPending,
		LastAction:   policy.ActionRetry,
		PolicyName:   "no_progress_retry",
		AttemptIndex: 1,
		MaxAttempts:  2,
		LastError:    "previous",
	}); err != nil {
		t.Fatalf("SaveRecoveryState failed: %v", err)
	}

	got := s.questResponse(q)
	view, ok := got["policy_view"].(map[string]any)
	if !ok {
		t.Fatalf("policy_view missing: %#v", got)
	}
	if view["effective_trust_tier"] != fsstore.TrustTier3 {
		t.Fatalf("effective trust tier = %#v", view)
	}
	if view["safety_floor_reason"] == "" {
		t.Fatalf("safety floor should be present: %#v", view)
	}
	decision, ok := view["last_policy_decision"].(map[string]any)
	if !ok || decision["id"] != "pol_dec_test" || decision["action"] != "auto_pass" {
		t.Fatalf("bad policy decision view: %#v", view)
	}
	recovery, ok := view["recovery_state_summary"].(map[string]any)
	if !ok || recovery["strategy"] != policy.ActionRetry || recovery["status"] != string(fsstore.RecoveryStatusPending) {
		t.Fatalf("bad recovery view: %#v", view)
	}
}

func TestCreateQuestMultipartArtifacts(t *testing.T) {
	s, h := newTestServer(t)
	configureRunnableAdventurers(t, s)

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	_ = mw.WriteField("query", "看一下截图里的问题")
	_ = mw.WriteField("type", "design")
	_ = mw.WriteField("auto_start", "false")
	fw, err := mw.CreateFormFile("artifacts", "screen.png")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := fw.Write([]byte("fake png bytes")); err != nil {
		t.Fatalf("write multipart file: %v", err)
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("close multipart: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/quests", &body)
	req.Header.Set("Authorization", "Bearer "+testToken)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("create status = %d, body=%s", rr.Code, rr.Body.String())
	}
	createBody := decodeBody(t, rr)
	qid, _ := createBody["qid"].(string)
	quest := createBody["quest"].(map[string]any)
	inputs, ok := quest["inputs"].([]any)
	if !ok || len(inputs) != 1 {
		t.Fatalf("quest inputs = %#v", quest["inputs"])
	}
	input := inputs[0].(map[string]any)
	if input["kind"] != "image" || input["name"] != "screen.png" || input["storage_path"] == "" {
		t.Fatalf("unexpected artifact meta: %#v", input)
	}
	artifactID := input["id"].(string)
	path := fsstore.NewQuestStore(s.root).ArtifactPath(qid, fsstore.QuestArtifact{
		StoragePath: input["storage_path"].(string),
	})
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read stored artifact: %v", err)
	}
	if string(raw) != "fake png bytes" {
		t.Fatalf("stored artifact = %q", string(raw))
	}

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodGet, "/api/quests/"+qid+"/artifacts/"+artifactID, ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("artifact get status = %d, body=%s", rr.Code, rr.Body.String())
	}
	if rr.Body.String() != "fake png bytes" {
		t.Fatalf("artifact response = %q", rr.Body.String())
	}
}

func TestCreateQuestPersistsPolicyOverrides(t *testing.T) {
	s, h := newTestServer(t)
	configureRunnableAdventurers(t, s)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodPost, "/api/quests", `{"query":"x","intensity":"quick","allow_quick_auto_complete":true,"auto_spawn_execute":true,"auto_start":false}`))
	if rr.Code != http.StatusOK {
		t.Fatalf("create status = %d, body=%s", rr.Code, rr.Body.String())
	}
	body := decodeBody(t, rr)
	qid, _ := body["qid"].(string)
	_, err := fsstore.NewQuestStore(s.root).LoadQuest(qid)
	if err != nil {
		t.Fatalf("LoadQuest: %v", err)
	}
}

func TestCreateQuestPersistsWorkflowMode(t *testing.T) {
	s, h := newTestServer(t)
	configureRunnableAdventurers(t, s)

	cases := []struct {
		name         string
		body         string
		wantMode     model.WorkflowMode
		wantPipeline string
	}{
		{
			name:         "run mode becomes direct",
			body:         `{"query":"direct task","mode":"run","auto_start":false}`,
			wantMode:     model.WorkflowModeDirect,
			wantPipeline: "run",
		},
		{
			name:         "check mode becomes checked",
			body:         `{"query":"checked task","mode":"check","auto_start":false}`,
			wantMode:     model.WorkflowModeChecked,
			wantPipeline: "default",
		},
		{
			name:         "design mode becomes goal",
			body:         `{"query":"goal task","mode":"design","auto_start":false}`,
			wantMode:     model.WorkflowModeGoal,
			wantPipeline: "design",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, authedReq(http.MethodPost, "/api/quests", tc.body))
			if rr.Code != http.StatusOK {
				t.Fatalf("create status = %d, body=%s", rr.Code, rr.Body.String())
			}
			body := decodeBody(t, rr)
			qid, _ := body["qid"].(string)
			q, err := fsstore.NewQuestStore(s.root).LoadQuest(qid)
			if err != nil {
				t.Fatalf("LoadQuest: %v", err)
			}
			if q.WorkflowMode != tc.wantMode {
				t.Fatalf("workflow_mode = %q, want %q", q.WorkflowMode, tc.wantMode)
			}
			if q.OriginalRequest != q.Query || q.OriginalRequest == "" {
				t.Fatalf("original request should preserve query: query=%q original=%q", q.Query, q.OriginalRequest)
			}
			if q.PipelineName != tc.wantPipeline {
				t.Fatalf("pipeline_name = %q, want %q", q.PipelineName, tc.wantPipeline)
			}
			posts, err := fsstore.NewQuestStore(s.root).LoadThreadPosts(qid)
			if err != nil {
				t.Fatalf("LoadThreadPosts: %v", err)
			}
			if len(posts) != 1 || posts[0].PostID != fsstore.RootPostID(qid) {
				t.Fatalf("root post missing: %+v", posts)
			}
		})
	}
}

func TestUpgradeQuestWorkflowModeWritesSystemPost(t *testing.T) {
	s, h := newTestServer(t)
	configureRunnableAdventurers(t, s)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodPost, "/api/quests", `{"query":"upgrade task","mode":"run","auto_start":false}`))
	if rr.Code != http.StatusOK {
		t.Fatalf("create status = %d, body=%s", rr.Code, rr.Body.String())
	}
	body := decodeBody(t, rr)
	qid, _ := body["qid"].(string)

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodPost, "/api/quests/"+qid+"/workflow-mode", `{"workflow_mode":"checked","reason":"external write requires review"}`))
	if rr.Code != http.StatusOK {
		t.Fatalf("upgrade status = %d, body=%s", rr.Code, rr.Body.String())
	}
	q, err := fsstore.NewQuestStore(s.root).LoadQuest(qid)
	if err != nil {
		t.Fatalf("LoadQuest: %v", err)
	}
	if q.WorkflowMode != model.WorkflowModeChecked {
		t.Fatalf("workflow_mode = %q, want checked", q.WorkflowMode)
	}
	posts, err := fsstore.NewQuestStore(s.root).LoadThreadPosts(qid)
	if err != nil {
		t.Fatalf("LoadThreadPosts: %v", err)
	}
	if len(posts) != 2 {
		t.Fatalf("posts = %d, want root + upgrade system reply: %+v", len(posts), posts)
	}
	upgrade := posts[1]
	if upgrade.Kind != "system_workflow_upgrade" || upgrade.AuthorRole != model.PostRoleSystem {
		t.Fatalf("bad upgrade post: %+v", upgrade)
	}
	if !strings.Contains(upgrade.Content, "direct → checked") || !strings.Contains(upgrade.Content, "external write requires review") {
		t.Fatalf("upgrade post should explain mode change and reason: %+v", upgrade)
	}

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodPost, "/api/quests/"+qid+"/workflow-mode", `{"workflow_mode":"checked","reason":"duplicate"}`))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("duplicate upgrade status = %d, want 400; body=%s", rr.Code, rr.Body.String())
	}
	posts, err = fsstore.NewQuestStore(s.root).LoadThreadPosts(qid)
	if err != nil {
		t.Fatalf("LoadThreadPosts after duplicate: %v", err)
	}
	if len(posts) != 2 {
		t.Fatalf("duplicate upgrade must not append another post: %+v", posts)
	}
}

func TestUpgradeQuestWorkflowModeRejectsDowngrade(t *testing.T) {
	s, h := newTestServer(t)
	configureRunnableAdventurers(t, s)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodPost, "/api/quests", `{"query":"goal task","mode":"design","auto_start":false}`))
	if rr.Code != http.StatusOK {
		t.Fatalf("create status = %d, body=%s", rr.Code, rr.Body.String())
	}
	body := decodeBody(t, rr)
	qid, _ := body["qid"].(string)

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodPost, "/api/quests/"+qid+"/workflow-mode", `{"workflow_mode":"checked","reason":"try downgrade"}`))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("downgrade status = %d, want 400; body=%s", rr.Code, rr.Body.String())
	}
	q, err := fsstore.NewQuestStore(s.root).LoadQuest(qid)
	if err != nil {
		t.Fatalf("LoadQuest: %v", err)
	}
	if q.WorkflowMode != model.WorkflowModeGoal {
		t.Fatalf("workflow_mode changed on downgrade: %q", q.WorkflowMode)
	}
	posts, err := fsstore.NewQuestStore(s.root).LoadThreadPosts(qid)
	if err != nil {
		t.Fatalf("LoadThreadPosts: %v", err)
	}
	if len(posts) != 1 {
		t.Fatalf("downgrade must not append system post: %+v", posts)
	}

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodPost, "/api/quests/"+qid+"/workflow-mode", `{"workflow_mode":"sideways","reason":"bad mode"}`))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("invalid mode status = %d, want 400; body=%s", rr.Code, rr.Body.String())
	}
}

func TestUpgradeQuestWorkflowModeRejectsActiveRuntime(t *testing.T) {
	tests := []struct {
		name       string
		configure  func(*fsstore.GlobalConfig)
		setup      func(t *testing.T, s *Server, h http.Handler) (qid string, wantMode model.WorkflowMode)
		targetMode model.WorkflowMode
	}{
		{
			name: "running",
			setup: func(t *testing.T, s *Server, h http.Handler) (string, model.WorkflowMode) {
				qid := createWorkflowModeQuest(t, h, `{"query":"active running upgrade","mode":"run","auto_start":false}`)
				qs := fsstore.NewQuestStore(s.root)
				q, err := qs.LoadQuest(qid)
				if err != nil {
					t.Fatalf("LoadQuest: %v", err)
				}
				q.Status = model.QuestStatusRunning
				if err := qs.SaveQuest(q); err != nil {
					t.Fatalf("SaveQuest: %v", err)
				}
				return qid, model.WorkflowModeDirect
			},
			targetMode: model.WorkflowModeChecked,
		},
		{
			name: "reviewing",
			setup: func(t *testing.T, s *Server, h http.Handler) (string, model.WorkflowMode) {
				qid := createWorkflowModeQuest(t, h, `{"query":"active reviewing upgrade","mode":"check","auto_start":false}`)
				qs := fsstore.NewQuestStore(s.root)
				q, err := qs.LoadQuest(qid)
				if err != nil {
					t.Fatalf("LoadQuest: %v", err)
				}
				q.Status = model.QuestStatusReviewing
				q.CurrentPhase = 1
				if len(q.Phases) > 1 {
					q.Phases[0].Status = model.PhaseDone
					q.Phases[1].Status = model.PhaseRunning
				}
				if err := qs.SaveQuest(q); err != nil {
					t.Fatalf("SaveQuest: %v", err)
				}
				return qid, model.WorkflowModeChecked
			},
			targetMode: model.WorkflowModeGoal,
		},
		{
			name: "queued",
			configure: func(cfg *fsstore.GlobalConfig) {
				cfg.MaxConcurrent = 1
			},
			setup: func(t *testing.T, s *Server, h http.Handler) (string, model.WorkflowMode) {
				blockRunningSlot(t, s)
				qid := createWorkflowModeQuest(t, h, `{"query":"queued upgrade","mode":"run","auto_start":false}`)
				if err := s.engine.StartQuest(t.Context(), qid); err != nil {
					t.Fatalf("StartQuest queued: %v", err)
				}
				if s.engine.QueuePosition(qid) == 0 {
					t.Fatalf("quest should be queued")
				}
				return qid, model.WorkflowModeDirect
			},
			targetMode: model.WorkflowModeChecked,
		},
		{
			name: "starting",
			configure: func(cfg *fsstore.GlobalConfig) {
				cfg.MaxConcurrent = 1
			},
			setup: func(t *testing.T, s *Server, h http.Handler) (string, model.WorkflowMode) {
				qid := createWorkflowModeQuest(t, h, `{"query":"starting upgrade","mode":"run","auto_start":false}`)
				if err := s.engine.StartQuest(t.Context(), qid); err != nil {
					t.Fatalf("StartQuest starting: %v", err)
				}
				qs := fsstore.NewQuestStore(s.root)
				q, err := qs.LoadQuest(qid)
				if err != nil {
					t.Fatalf("LoadQuest: %v", err)
				}
				// Keep the runtime slot reserved while preserving the pending
				// persisted view used by IsQuestStarting.
				q.Status = model.QuestStatusPending
				q.UpdatedAtMs = fsstore.NowMs()
				if err := qs.SaveQuest(q); err != nil {
					t.Fatalf("SaveQuest starting: %v", err)
				}
				if !s.engine.IsQuestStarting(qid) {
					t.Fatalf("quest should be starting")
				}
				return qid, model.WorkflowModeDirect
			},
			targetMode: model.WorkflowModeChecked,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s, h := newTestServerWithConfig(t, tc.configure)
			configureRunnableAdventurers(t, s)
			qid, wantMode := tc.setup(t, s, h)
			qs := fsstore.NewQuestStore(s.root)

			rr := httptest.NewRecorder()
			body := `{"workflow_mode":"` + string(tc.targetMode) + `","reason":"too late"}`
			h.ServeHTTP(rr, authedReq(http.MethodPost, "/api/quests/"+qid+"/workflow-mode", body))
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("active %s upgrade status = %d, want 400; body=%s", tc.name, rr.Code, rr.Body.String())
			}
			got, err := qs.LoadQuest(qid)
			if err != nil {
				t.Fatalf("LoadQuest after upgrade: %v", err)
			}
			if got.WorkflowMode != wantMode {
				t.Fatalf("workflow_mode changed for active quest: %q", got.WorkflowMode)
			}
			posts, err := qs.LoadThreadPosts(qid)
			if err != nil {
				t.Fatalf("LoadThreadPosts: %v", err)
			}
			if len(posts) != 1 {
				t.Fatalf("active upgrade must not append system post: %+v", posts)
			}
		})
	}
}

func createWorkflowModeQuest(t *testing.T, h http.Handler, body string) string {
	t.Helper()
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodPost, "/api/quests", body))
	if rr.Code != http.StatusOK {
		t.Fatalf("create status = %d, body=%s", rr.Code, rr.Body.String())
	}
	decoded := decodeBody(t, rr)
	qid, _ := decoded["qid"].(string)
	if qid == "" {
		t.Fatalf("create response missing qid: %#v", decoded)
	}
	return qid
}

func blockRunningSlot(t *testing.T, s *Server) {
	t.Helper()
	blocking := &testBlockingExecutor{
		MockExecutor: executor.NewMockExecutor("blocking"),
		block:        make(chan struct{}),
	}
	s.engine.RegisterExecutor("test_agent", blocking)
	t.Cleanup(func() { close(blocking.block) })

	q, err := s.engine.CreateQuest(t.Context(), "occupy workflow upgrade slot", model.QuestTypeExecute, "")
	if err != nil {
		t.Fatalf("CreateQuest running: %v", err)
	}
	if err := s.engine.StartQuest(t.Context(), q.ID); err != nil {
		t.Fatalf("StartQuest running: %v", err)
	}
}

func TestGetQuestReturnsThreadPosts(t *testing.T) {
	s, h := newTestServer(t)
	configureRunnableAdventurers(t, s)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodPost, "/api/quests", `{"query":"thread detail","mode":"check","auto_start":false}`))
	if rr.Code != http.StatusOK {
		t.Fatalf("create status = %d, body=%s", rr.Code, rr.Body.String())
	}
	body := decodeBody(t, rr)
	qid, _ := body["qid"].(string)
	if _, err := fsstore.NewQuestStore(s.root).AppendThreadPost(qid, fsstore.AppendThreadPostOptions{
		PostID:         "post_detail_test",
		ThreadID:       qid,
		ParentReplyID:  fsstore.RootPostID(qid),
		RootPostID:     fsstore.RootPostID(qid),
		AuthorIdentity: "system",
		AuthorRole:     model.PostRoleSystem,
		Kind:           "system_rework",
		Content:        "rework projected",
	}); err != nil {
		t.Fatalf("AppendThreadPost failed: %v", err)
	}

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodGet, "/api/quests/"+qid, ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("get status = %d, body=%s", rr.Code, rr.Body.String())
	}
	got := decodeBody(t, rr)
	posts, ok := got["thread_posts"].([]any)
	if !ok || len(posts) != 2 {
		t.Fatalf("thread_posts = %#v, want root + system post", got["thread_posts"])
	}
}

func TestGetQuestLedgerAuditCanRepair(t *testing.T) {
	s, h := newTestServer(t)
	configureRunnableAdventurers(t, s)

	q := &fsstore.QuestMeta{
		ID:            "qst_api_ledger_repair",
		ShortID:       "apiledger",
		Query:         "api ledger repair",
		Type:          model.QuestTypeExecute,
		Status:        model.QuestStatusSuccess,
		FinalVerdict:  model.VerdictPass,
		FinalComment:  "done",
		FinalizedBy:   "user",
		CreatedBy:     "user",
		CreatedAtMs:   1000,
		CompletedAtMs: 2000,
	}
	if err := fsstore.NewQuestStore(s.root).CreateQuest(q); err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodGet, "/api/quests/"+q.ID+"/ledger-audit", ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("audit status = %d, body=%s", rr.Code, rr.Body.String())
	}
	got := decodeBody(t, rr)
	if got["ok"] != false {
		t.Fatalf("audit should report unhealthy ledger: %#v", got)
	}

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodGet, "/api/quests/"+q.ID+"/ledger-audit?repair=true", ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("repair status = %d, body=%s", rr.Code, rr.Body.String())
	}
	got = decodeBody(t, rr)
	if got["ok"] != true {
		t.Fatalf("repair should return clean audit: %#v", got)
	}
	audit, ok := got["audit"].(map[string]any)
	if !ok {
		t.Fatalf("audit body malformed: %#v", got["audit"])
	}
	repaired, _ := audit["repaired"].([]any)
	if len(repaired) != 3 {
		t.Fatalf("repair should record 3 actions, got %#v", repaired)
	}
}

func TestGetQuestReturnsFanoutGroupProjection(t *testing.T) {
	s, h := newTestServer(t)
	configureRunnableAdventurers(t, s)

	first := `{"query":"leaf a","group_id":"grp_detail_tree","leaf_id":"leaf-a","ownership_scopes":["internal/server"],"merge_strategy":"single_leaf","merge_owner_leaf_id":"leaf-a"}`
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodPost, "/api/quests/spawn", first))
	if rr.Code != http.StatusOK {
		t.Fatalf("first spawn status = %d, body=%s", rr.Code, rr.Body.String())
	}
	firstBody := decodeBody(t, rr)
	firstQID, _ := firstBody["qid"].(string)

	second := `{"query":"leaf b","group_id":"grp_detail_tree","leaf_id":"leaf-b","ownership_scopes":["internal/server"],"merge_strategy":"single_leaf","merge_owner_leaf_id":"leaf-a"}`
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodPost, "/api/quests/spawn", second))
	if rr.Code != http.StatusOK {
		t.Fatalf("second spawn status = %d, body=%s", rr.Code, rr.Body.String())
	}
	secondBody := decodeBody(t, rr)
	secondQID, _ := secondBody["qid"].(string)

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodGet, "/api/quests/"+secondQID, ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("get status = %d, body=%s", rr.Code, rr.Body.String())
	}
	got := decodeBody(t, rr)
	group, ok := got["fanout_group"].(map[string]any)
	if !ok {
		t.Fatalf("fanout_group missing or malformed: %#v", got["fanout_group"])
	}
	if group["group_id"] != "grp_detail_tree" {
		t.Fatalf("fanout_group group_id = %#v", group["group_id"])
	}
	leaves, ok := group["leaves"].([]any)
	if !ok || len(leaves) != 2 {
		t.Fatalf("fanout leaves = %#v, want 2", group["leaves"])
	}

	byLeaf := map[string]map[string]any{}
	for _, raw := range leaves {
		leaf, ok := raw.(map[string]any)
		if !ok {
			t.Fatalf("leaf malformed: %#v", raw)
		}
		byLeaf[leaf["fanout_leaf_id"].(string)] = leaf
	}
	if byLeaf["leaf-a"]["quest_id"] != firstQID || byLeaf["leaf-b"]["quest_id"] != secondQID {
		t.Fatalf("leaf quest ids not projected from stored quests: %#v", byLeaf)
	}
	if byLeaf["leaf-b"]["merge_owner_leaf_id"] != "leaf-a" || byLeaf["leaf-b"]["merge_strategy"] != fsstore.MergeStrategySingleLeaf {
		t.Fatalf("merge contract not projected: %#v", byLeaf["leaf-b"])
	}
	scopes, ok := byLeaf["leaf-b"]["ownership_scopes"].([]any)
	if !ok || len(scopes) != 1 || scopes[0] != "internal/server" {
		t.Fatalf("ownership scopes not projected: %#v", byLeaf["leaf-b"]["ownership_scopes"])
	}
	if byLeaf["leaf-b"]["is_current"] != true || byLeaf["leaf-a"]["is_current"] == true {
		t.Fatalf("current leaf marker wrong: %#v", byLeaf)
	}
}

func TestSpawnQuestPersistsFanoutContract(t *testing.T) {
	s, h := newTestServer(t)
	configureRunnableAdventurers(t, s)

	body := `{"query":"leaf api","group_id":"grp_api","leaf_id":"leaf-api","ownership_scopes":["internal/server"],"merge_strategy":"no_merge"}`
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodPost, "/api/quests/spawn", body))
	if rr.Code != http.StatusOK {
		t.Fatalf("spawn status = %d, body=%s", rr.Code, rr.Body.String())
	}
	got := decodeBody(t, rr)
	qid, _ := got["qid"].(string)
	if got["leaf_id"] != "leaf-api" {
		t.Fatalf("response leaf_id = %#v", got["leaf_id"])
	}
	q, err := fsstore.NewQuestStore(s.root).LoadQuest(qid)
	if err != nil {
		t.Fatalf("LoadQuest failed: %v", err)
	}
	if q.FanoutLeafID != "leaf-api" || q.MergeStrategy != fsstore.MergeStrategyNoMerge || len(q.OwnershipScopes) != 1 {
		t.Fatalf("fanout contract not persisted: %+v", q)
	}
}

func TestSpawnQuestRejectsOwnershipConflict(t *testing.T) {
	s, h := newTestServer(t)
	configureRunnableAdventurers(t, s)

	first := `{"query":"leaf a","group_id":"grp_api_conflict","leaf_id":"leaf-a","ownership_scopes":["internal/server"],"merge_strategy":"no_merge"}`
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodPost, "/api/quests/spawn", first))
	if rr.Code != http.StatusOK {
		t.Fatalf("first spawn status = %d, body=%s", rr.Code, rr.Body.String())
	}
	second := `{"query":"leaf b","group_id":"grp_api_conflict","leaf_id":"leaf-b","ownership_scopes":["internal/server"],"merge_strategy":"no_merge"}`
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodPost, "/api/quests/spawn", second))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("conflicting spawn status = %d, want 400; body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "ownership scope conflict") {
		t.Fatalf("response should explain ownership conflict: %s", rr.Body.String())
	}
}

func TestSpawnQuestInfrastructureErrorIsServerError(t *testing.T) {
	_, h := newTestServer(t)

	body := `{"query":"no adventurer","group_id":"grp_no_agent"}`
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodPost, "/api/quests/spawn", body))
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("spawn without runnable adventurer status = %d, want 500; body=%s", rr.Code, rr.Body.String())
	}
}

func TestCreateQuestMultipartPersistsPolicyOverrides(t *testing.T) {
	s, h := newTestServer(t)
	configureRunnableAdventurers(t, s)

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	_ = mw.WriteField("query", "带附件的 quick 委托")
	_ = mw.WriteField("intensity", "quick")
	_ = mw.WriteField("allow_quick_auto_complete", "true")
	_ = mw.WriteField("auto_spawn_execute", "true")
	_ = mw.WriteField("auto_start", "false")
	fw, err := mw.CreateFormFile("artifacts", "note.txt")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := fw.Write([]byte("hello")); err != nil {
		t.Fatalf("write multipart file: %v", err)
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("close multipart: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/quests", &body)
	req.Header.Set("Authorization", "Bearer "+testToken)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("create status = %d, body=%s", rr.Code, rr.Body.String())
	}
	createBody := decodeBody(t, rr)
	qid, _ := createBody["qid"].(string)
	q, err := fsstore.NewQuestStore(s.root).LoadQuest(qid)
	if err != nil {
		t.Fatalf("LoadQuest: %v", err)
	}
	if len(q.Inputs) != 1 {
		t.Fatalf("multipart input should still persist: %+v", q.Inputs)
	}
}

func TestQuestListSupportsTimeFilters(t *testing.T) {
	s, h := newTestServer(t)

	old := &fsstore.QuestMeta{
		ID:          "qst_old",
		ShortID:     "old",
		Query:       "old",
		Type:        model.QuestTypeExecute,
		Status:      model.QuestStatusSuccess,
		CreatedBy:   "user",
		CreatedAtMs: 1000,
		UpdatedAtMs: 2000,
	}
	recent := &fsstore.QuestMeta{
		ID:          "qst_recent",
		ShortID:     "recent",
		Query:       "recent",
		Type:        model.QuestTypeExecute,
		Status:      model.QuestStatusPending,
		CreatedBy:   "user",
		CreatedAtMs: 3000,
		UpdatedAtMs: 9000,
	}
	legacy := &fsstore.QuestMeta{
		ID:            "qst_legacy",
		ShortID:       "legacy",
		Query:         "legacy",
		Type:          model.QuestTypeExecute,
		Status:        model.QuestStatusSuccess,
		CreatedBy:     "user",
		CreatedAtMs:   4000,
		CompletedAtMs: 8000,
	}
	for _, q := range []*fsstore.QuestMeta{old, recent, legacy} {
		if err := fsstore.NewQuestStore(s.root).CreateQuest(q); err != nil {
			t.Fatalf("CreateQuest(%s): %v", q.ID, err)
		}
		switch q.ID {
		case "qst_old":
			q.CreatedAtMs = 1000
			q.UpdatedAtMs = 2000
		case "qst_recent":
			q.CreatedAtMs = 3000
			q.UpdatedAtMs = 9000
		case "qst_legacy":
			q.CreatedAtMs = 4000
			q.UpdatedAtMs = 0
			q.CompletedAtMs = 8000
		}
		if err := fsstore.WriteJSON(s.root.Sub(fsstore.SubdirQuests, q.ID, "meta.json"), q); err != nil {
			t.Fatalf("rewrite quest %s: %v", q.ID, err)
		}
	}

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodGet, "/api/quests?updated_since_ms=8000", ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("list status = %d, body=%s", rr.Code, rr.Body.String())
	}
	body := decodeBody(t, rr)
	items, ok := body["items"].([]any)
	if !ok {
		t.Fatalf("items missing: %#v", body)
	}
	got := map[string]bool{}
	for _, item := range items {
		q := item.(map[string]any)
		got[q["id"].(string)] = true
	}
	if got["qst_old"] || !got["qst_recent"] || !got["qst_legacy"] {
		t.Fatalf("updated_since filter result = %#v", got)
	}

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodGet, "/api/quests?created_since_ms=3500", ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("list status = %d, body=%s", rr.Code, rr.Body.String())
	}
	body = decodeBody(t, rr)
	items, ok = body["items"].([]any)
	if !ok {
		t.Fatalf("items missing: %#v", body)
	}
	got = map[string]bool{}
	for _, item := range items {
		q := item.(map[string]any)
		got[q["id"].(string)] = true
	}
	if got["qst_old"] || got["qst_recent"] || !got["qst_legacy"] {
		t.Fatalf("created_since filter result = %#v", got)
	}
}

func TestQuestListSupportsCombinedFilters(t *testing.T) {
	s, h := newTestServer(t)

	fixtures := []*fsstore.QuestMeta{
		{
			ID:          "qst_match",
			ShortID:     "match",
			Query:       "match",
			Type:        model.QuestTypeDesign,
			Status:      model.QuestStatusBlocked,
			CreatedBy:   "automation:auto_daily",
			Official:    true,
			CreatedAtMs: 1000,
			UpdatedAtMs: 9000,
		},
		{
			ID:          "qst_wrong_type",
			ShortID:     "wrongtype",
			Query:       "wrong type",
			Type:        model.QuestTypeExecute,
			Status:      model.QuestStatusBlocked,
			CreatedBy:   "automation:auto_daily",
			Official:    true,
			CreatedAtMs: 1000,
			UpdatedAtMs: 9000,
		},
		{
			ID:          "qst_wrong_source",
			ShortID:     "wrongsource",
			Query:       "wrong source",
			Type:        model.QuestTypeDesign,
			Status:      model.QuestStatusBlocked,
			CreatedBy:   "user",
			Official:    true,
			CreatedAtMs: 1000,
			UpdatedAtMs: 9000,
		},
		{
			ID:          "qst_too_old",
			ShortID:     "tooold",
			Query:       "too old",
			Type:        model.QuestTypeDesign,
			Status:      model.QuestStatusBlocked,
			CreatedBy:   "automation:auto_daily",
			Official:    true,
			CreatedAtMs: 1000,
			UpdatedAtMs: 3000,
		},
	}
	for _, q := range fixtures {
		createdAt := q.CreatedAtMs
		updatedAt := q.UpdatedAtMs
		if err := fsstore.NewQuestStore(s.root).CreateQuest(q); err != nil {
			t.Fatalf("CreateQuest(%s): %v", q.ID, err)
		}
		q.CreatedAtMs = createdAt
		q.UpdatedAtMs = updatedAt
		if err := fsstore.WriteJSON(s.root.Sub(fsstore.SubdirQuests, q.ID, "meta.json"), q); err != nil {
			t.Fatalf("rewrite quest %s: %v", q.ID, err)
		}
	}

	path := "/api/quests?official=true&status=blocked&type=design&created_by=automation:auto_daily&updated_since_ms=8000"
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodGet, path, ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("list status = %d, body=%s", rr.Code, rr.Body.String())
	}
	body := decodeBody(t, rr)
	items, ok := body["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("items = %#v, want one match", body["items"])
	}
	got := items[0].(map[string]any)
	if got["id"] != "qst_match" {
		t.Fatalf("filtered quest = %#v, want qst_match", got)
	}
}

func TestQuestListSupportsStatsDrilldownFilters(t *testing.T) {
	s, h := newTestServer(t)
	projectDir := filepath.Join(t.TempDir(), "stats-project")
	fixtures := []*fsstore.QuestMeta{
		{
			ID:             "qst_attr",
			ShortID:        "attr",
			Query:          "attr",
			Type:           model.QuestTypeExecute,
			Status:         model.QuestStatusBlocked,
			BaseWorkingDir: projectDir,
			CreatedBy:      "user",
			CreatedAtMs:    1000,
			UpdatedAtMs:    9000,
			FailureAttribution: &fsstore.FailureAttribution{
				Reason:   "agent_error_auth",
				Category: "auth",
			},
		},
		{
			ID:                "qst_block_code",
			ShortID:           "blockcode",
			Query:             "block code",
			Type:              model.QuestTypeExecute,
			Status:            model.QuestStatusBlocked,
			BlockedReason:     "中文原文不参与筛选",
			BlockedReasonCode: "agent_consecutive_errors",
			CreatedBy:         "user",
			CreatedAtMs:       1000,
			UpdatedAtMs:       9000,
		},
		{
			ID:          "qst_session",
			ShortID:     "session",
			Query:       "session",
			Type:        model.QuestTypeExecute,
			Status:      model.QuestStatusFailed,
			CreatedBy:   "user",
			CreatedAtMs: 1000,
			UpdatedAtMs: 9000,
		},
		{
			ID:               "qst_apply_failed",
			ShortID:          "apply",
			Query:            "apply",
			Type:             model.QuestTypeExecute,
			Status:           model.QuestStatusSuccess,
			ApplyStatus:      model.ApplyStatusFailed,
			ApplyError:       "apply 失败: patch 与当前 base 冲突",
			DiffChangedFiles: 1,
			CreatedBy:        "user",
			CreatedAtMs:      1000,
			UpdatedAtMs:      9000,
		},
	}
	qs := fsstore.NewQuestStore(s.root)
	for _, q := range fixtures {
		if err := qs.CreateQuest(q); err != nil {
			t.Fatalf("CreateQuest(%s): %v", q.ID, err)
		}
	}
	if err := qs.AppendSessionRow("qst_session", "sid_0", &fsstore.QuestSessionRow{
		Seq:       1,
		Timestamp: 2000,
		Kind:      "error",
		SessionID: "sid_0",
		Error:     "context deadline exceeded",
	}); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		path string
		want string
	}{
		{"/api/quests?project=stats-project", "qst_attr"},
		{"/api/quests?failure_reason=agent_error_auth", "qst_attr"},
		{"/api/quests?failure_reason=agent_consecutive_errors", "qst_block_code"},
		{"/api/quests?failure_category=auth", "qst_attr"},
		{"/api/quests?failure_reason=timeout", "qst_session"},
		{"/api/quests?failure_reason=apply_failed", "qst_apply_failed"},
		{"/api/quests?status=apply_failed", "qst_apply_failed"},
		{"/api/quests?quest_id=qst_session", "qst_session"},
	}
	for _, tc := range cases {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, authedReq(http.MethodGet, tc.path, ""))
		if rr.Code != http.StatusOK {
			t.Fatalf("%s status = %d, body=%s", tc.path, rr.Code, rr.Body.String())
		}
		body := decodeBody(t, rr)
		items, ok := body["items"].([]any)
		if !ok || len(items) != 1 {
			t.Fatalf("%s items = %#v, want one", tc.path, body["items"])
		}
		got := items[0].(map[string]any)
		if got["id"] != tc.want {
			t.Fatalf("%s filtered quest = %#v, want %s", tc.path, got, tc.want)
		}
	}
}

func TestRecoverAgentRejectsNonBlockedQuest(t *testing.T) {
	s, h := newTestServer(t)
	configureRunnableAdventurers(t, s)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodPost, "/api/quests", `{"query":"x","auto_start":false}`))
	if rr.Code != http.StatusOK {
		t.Fatalf("create status = %d, body=%s", rr.Code, rr.Body.String())
	}
	body := decodeBody(t, rr)
	qid, _ := body["qid"].(string)

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodPost, "/api/quests/"+qid+"/recover-agent", `{"action":"relay_auth_login"}`))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("recover status = %d, want %d; body=%s", rr.Code, http.StatusBadRequest, rr.Body.String())
	}
}

func TestInboxUpdateAcceptsFullEditableContract(t *testing.T) {
	s, h := newTestServer(t)
	configureRunnableAdventurers(t, s)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodPost, "/api/automations", `{"id":"auto_inbox_api","name":"inbox api","query":"old","auto_start":false,"enabled":true}`))
	if rr.Code != http.StatusCreated {
		t.Fatalf("create automation status = %d, body=%s", rr.Code, rr.Body.String())
	}

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodPost, "/api/automations/auto_inbox_api/run", ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("run automation status = %d, body=%s", rr.Code, rr.Body.String())
	}
	body := decodeBody(t, rr)
	qid, _ := body["qid"].(string)

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodPost, "/api/inbox/"+qid, `{"query":"new","type":"design","work_dir":"/tmp/gloop-api-inbox","workspace_mode":"copy"}`))
	if rr.Code != http.StatusOK {
		t.Fatalf("update inbox status = %d, body=%s", rr.Code, rr.Body.String())
	}
	body = decodeBody(t, rr)
	quest, ok := body["quest"].(map[string]any)
	if !ok {
		t.Fatalf("quest body missing: %#v", body)
	}
	if quest["query"] != "new" || quest["type"] != "design" || quest["base_working_dir"] != "/tmp/gloop-api-inbox" || quest["workspace_mode"] != "copy" {
		t.Fatalf("unexpected updated inbox quest: %#v", quest)
	}
}

func TestInboxAcceptReturnsQueuedQuestAndListHidesQueuedItem(t *testing.T) {
	s, h := newTestServerWithConfig(t, func(cfg *fsstore.GlobalConfig) {
		cfg.MaxConcurrent = 1
	})
	configureRunnableAdventurers(t, s)

	blocking := &testBlockingExecutor{
		MockExecutor: executor.NewMockExecutor("blocking"),
		block:        make(chan struct{}),
	}
	s.engine.RegisterExecutor("test_agent", blocking)
	defer close(blocking.block)

	ctx := context.Background()
	running, err := s.engine.CreateQuest(ctx, "occupy slot", model.QuestTypeExecute, "")
	if err != nil {
		t.Fatalf("CreateQuest running: %v", err)
	}
	if err := s.engine.StartQuest(ctx, running.ID); err != nil {
		t.Fatalf("StartQuest running: %v", err)
	}

	qid := "qst_queued_inbox_api"
	if err := fsstore.NewQuestStore(s.root).CreateQuest(&fsstore.QuestMeta{
		ID:             qid,
		ShortID:        "queued",
		Query:          "queued",
		Type:           model.QuestTypeExecute,
		Status:         model.QuestStatusPending,
		WorkspaceMode:  model.WorkspaceCopy,
		BaseWorkingDir: s.cfg.DefaultWorkingDir,
		CreatedBy:      model.QuestSourcePrefix + "auto_queue_api",
	}); err != nil {
		t.Fatalf("CreateQuest queued: %v", err)
	}

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodPost, "/api/inbox/"+qid+"/accept", ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("accept status = %d, body=%s", rr.Code, rr.Body.String())
	}
	body := decodeBody(t, rr)
	quest, ok := body["quest"].(map[string]any)
	if !ok {
		t.Fatalf("accept response should include quest: %#v", body)
	}
	if quest["queued"] != true || quest["queue_position"] != float64(1) {
		t.Fatalf("accepted quest should be visibly queued: %#v", quest)
	}

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodGet, "/api/inbox", ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("list inbox status = %d, body=%s", rr.Code, rr.Body.String())
	}
	body = decodeBody(t, rr)
	items, _ := body["items"].([]any)
	for _, raw := range items {
		item, _ := raw.(map[string]any)
		if item["id"] == qid {
			t.Fatalf("queued accepted quest should leave inbox list: %#v", item)
		}
	}
}

func TestStartQuestReturnsQueuedQuestWhenConcurrencyFull(t *testing.T) {
	s, h := newTestServerWithConfig(t, func(cfg *fsstore.GlobalConfig) {
		cfg.MaxConcurrent = 1
	})
	configureRunnableAdventurers(t, s)

	blocking := &testBlockingExecutor{
		MockExecutor: executor.NewMockExecutor("blocking"),
		block:        make(chan struct{}),
	}
	s.engine.RegisterExecutor("test_agent", blocking)
	defer close(blocking.block)

	ctx := context.Background()
	running, err := s.engine.CreateQuest(ctx, "occupy slot", model.QuestTypeExecute, "")
	if err != nil {
		t.Fatalf("CreateQuest running: %v", err)
	}
	if err := s.engine.StartQuest(ctx, running.ID); err != nil {
		t.Fatalf("StartQuest running: %v", err)
	}
	queued, err := s.engine.CreateQuest(ctx, "queued from detail", model.QuestTypeExecute, "")
	if err != nil {
		t.Fatalf("CreateQuest queued: %v", err)
	}

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodPost, "/api/quests/"+queued.ID+"/start", ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("start status = %d, body=%s", rr.Code, rr.Body.String())
	}
	body := decodeBody(t, rr)
	quest, ok := body["quest"].(map[string]any)
	if !ok {
		t.Fatalf("start response should include quest: %#v", body)
	}
	if quest["id"] != queued.ID || quest["queued"] != true || quest["queue_position"] != float64(1) {
		t.Fatalf("start response should show queued quest: %#v", quest)
	}
}

func TestHumanExceptionsEndpointProjectsActionableItems(t *testing.T) {
	s, h := newTestServer(t)
	qs := fsstore.NewQuestStore(s.root)
	fixtures := []*fsstore.QuestMeta{
		{
			ID:                "qst_blocked_action",
			ShortID:           "blocked",
			Query:             "blocked quest",
			Type:              model.QuestTypeExecute,
			Status:            model.QuestStatusBlocked,
			BlockedReason:     "agent auth expired",
			BlockedReasonCode: "agent_error_auth",
			CreatedBy:         "user",
			CreatedAtMs:       1000,
			UpdatedAtMs:       3000,
		},
		{
			ID:          "qst_waiting_action",
			ShortID:     "waiting",
			Query:       "waiting quest",
			Type:        model.QuestTypeExecute,
			Status:      model.QuestStatusWaitingInput,
			CreatedBy:   "user",
			CreatedAtMs: 1000,
			UpdatedAtMs: 2000,
			WaitingInput: &fsstore.WaitingInputState{
				QuestionID:   "ask_1",
				QuestionText: "需要哪个分支？",
			},
		},
	}
	for _, q := range fixtures {
		if err := qs.CreateQuest(q); err != nil {
			t.Fatalf("CreateQuest(%s): %v", q.ID, err)
		}
	}

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodGet, "/api/human-exceptions", ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rr.Code, rr.Body.String())
	}
	body := decodeBody(t, rr)
	items, ok := body["items"].([]any)
	if !ok || len(items) != 2 {
		t.Fatalf("items = %#v, want two exceptions", body["items"])
	}
	first, _ := items[0].(map[string]any)
	if first["quest_id"] != "qst_blocked_action" || first["risk_level"] != "high" {
		t.Fatalf("high risk blocked item should sort first: %#v", first)
	}
	if first["recommended_action"] == "" || first["evidence_summary"] == "" {
		t.Fatalf("blocked item should be actionable: %#v", first)
	}
	actions, _ := first["available_actions"].([]any)
	if len(actions) == 0 || actions[0] != "continue" {
		t.Fatalf("blocked item actions should align with resolve-blocked API: %#v", first)
	}
}

func TestStartQuestReturnsStartingQuestWhenRuntimeReserved(t *testing.T) {
	s, h := newTestServerWithConfig(t, func(cfg *fsstore.GlobalConfig) {
		cfg.MaxConcurrent = 1
	})
	configureRunnableAdventurers(t, s)

	neverCalled := &testBlockingExecutor{
		MockExecutor: executor.NewMockExecutor("blocking"),
		block:        make(chan struct{}),
	}
	s.engine.RegisterExecutor("test_agent", neverCalled)
	defer close(neverCalled.block)

	ctx := context.Background()
	q, err := s.engine.CreateQuest(ctx, "slow workspace prepare", model.QuestTypeExecute, "")
	if err != nil {
		t.Fatalf("CreateQuest: %v", err)
	}
	if err := s.engine.StartQuest(ctx, q.ID); err != nil {
		t.Fatalf("initial StartQuest: %v", err)
	}
	// Force the persisted view back to pending while the runtime slot remains
	// reserved. This reproduces the user-visible gap between reserveQuestRuntime
	// and questService.StartQuest during slow workspace preparation.
	q.Status = model.QuestStatusPending
	q.WorkspacePath = ""
	q.UpdatedAtMs = fsstore.NowMs()
	if err := fsstore.NewQuestStore(s.root).SaveQuest(q); err != nil {
		t.Fatalf("SaveQuest pending while runtime reserved: %v", err)
	}

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodPost, "/api/quests/"+q.ID+"/start", ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("start status = %d, body=%s", rr.Code, rr.Body.String())
	}
	body := decodeBody(t, rr)
	quest, ok := body["quest"].(map[string]any)
	if !ok {
		t.Fatalf("start response should include quest: %#v", body)
	}
	if quest["id"] != q.ID || (quest["starting"] != true && quest["status"] != string(model.QuestStatusRunning)) {
		t.Fatalf("start response should expose starting or running state: %#v", quest)
	}
}

func TestResolveUserReviewPassApplyCombinesReviewAndApply(t *testing.T) {
	s, h := newTestServer(t)
	baseDir := t.TempDir()
	qs := fsstore.NewQuestStore(s.root)
	q := &fsstore.QuestMeta{
		ID:               "qst_review_apply_api",
		ShortID:          "revapply",
		Query:            "review apply",
		Type:             model.QuestTypeExecute,
		Status:           model.QuestStatusUserReview,
		WorkspaceMode:    model.WorkspaceCopy,
		BaseWorkingDir:   baseDir,
		CreatedBy:        "user",
		FinalVerdict:     model.VerdictPass,
		DiffChangedFiles: 1,
		Outputs: []fsstore.QuestArtifact{
			{ID: "out_1", Kind: "doc", StoragePath: "https://example.test/notes"},
		},
	}
	if err := qs.CreateQuest(q); err != nil {
		t.Fatalf("CreateQuest: %v", err)
	}

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodPost, "/api/quests/qst_review_apply_api/resolve", `{"verdict":"pass","comment":"ship","apply":true}`))
	if rr.Code != http.StatusOK {
		t.Fatalf("resolve status = %d, body=%s", rr.Code, rr.Body.String())
	}
	body := decodeBody(t, rr)
	if body["ok"] != true || body["applied"] != true {
		t.Fatalf("resolve should apply in one request: %#v", body)
	}
	got, err := qs.LoadQuest(q.ID)
	if err != nil {
		t.Fatalf("LoadQuest: %v", err)
	}
	if got.Status != model.QuestStatusSuccess || got.FinalizedBy != "user" || !got.Applied || got.ApplyStatus != model.ApplyStatusApplied {
		t.Fatalf("bad review+apply quest state: %+v", got)
	}
}

func TestResolveUserReviewAcceptsShortID(t *testing.T) {
	s, h := newTestServer(t)
	qs := fsstore.NewQuestStore(s.root)
	q := &fsstore.QuestMeta{
		ID:            "qst_2606221806",
		ShortID:       "2606221806",
		Query:         "short id review",
		Type:          model.QuestTypeExecute,
		Status:        model.QuestStatusUserReview,
		WorkspaceMode: model.WorkspaceReadOnly,
		CreatedBy:     "user",
		FinalVerdict:  model.VerdictPass,
	}
	if err := qs.CreateQuest(q); err != nil {
		t.Fatalf("CreateQuest: %v", err)
	}

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodPost, "/api/quests/2606221806/resolve", `{"verdict":"pass","comment":"ship","apply":false}`))
	if rr.Code != http.StatusOK {
		t.Fatalf("resolve by short id status = %d, body=%s", rr.Code, rr.Body.String())
	}
	body := decodeBody(t, rr)
	quest, ok := body["quest"].(map[string]any)
	if !ok {
		t.Fatalf("resolve response should include updated quest: %#v", body)
	}
	if quest["id"] != "qst_2606221806" || quest["status"] != string(model.QuestStatusSuccess) {
		t.Fatalf("unexpected resolved quest response: %#v", quest)
	}
	got, err := qs.LoadQuest("2606221806")
	if err != nil {
		t.Fatalf("LoadQuest by short id: %v", err)
	}
	if got.ID != q.ID || got.Status != model.QuestStatusSuccess {
		t.Fatalf("bad short id load result: %+v", got)
	}
}

func TestCreateQuestValidation(t *testing.T) {
	_, h := newTestServer(t)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodPost, "/api/quests", `{"query":"x","workspace_mode":"bad"}`))

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusBadRequest, rr.Body.String())
	}
	body := decodeBody(t, rr)
	if body["ok"] != false || body["error"] != "workspace_mode 必须是 auto | worktree | copy | readonly" {
		t.Fatalf("unexpected validation body: %#v", body)
	}
}

func TestCreateQuestRequiresRunnableAdventurers(t *testing.T) {
	s, h := newTestServer(t)
	if err := s.root.SaveAgent(&fsstore.AgentConfig{
		Name:         "mock_custom",
		Type:         model.AgentTypeMock,
		Command:      "mock",
		DefaultModel: "mock-fast",
		Enabled:      true,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.engine.SaveAdventurer(&fsstore.AdventurerFile{
		ID:     "adv_mock_warrior",
		Name:   "mock warrior",
		Class:  model.ClassWarrior,
		Status: model.AdventurerActive,
		Agent:  "mock_custom",
		Level:  1,
	}); err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodPost, "/api/quests", `{"query":"x","warrior_id":"adv_mock_warrior"}`))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusBadRequest, rr.Body.String())
	}
	body := decodeBody(t, rr)
	if !strings.Contains(body["error"].(string), "Mock") {
		t.Fatalf("unexpected validation body: %#v", body)
	}
}

func TestCreateAutomationValidation(t *testing.T) {
	_, h := newTestServer(t)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodPost, "/api/automations", `{"name":"bad","query":"x","trigger":"bad"}`))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("trigger status = %d, want %d; body=%s", rr.Code, http.StatusBadRequest, rr.Body.String())
	}
	body := decodeBody(t, rr)
	if body["error"] != "trigger 必须是 manual | schedule | event" {
		t.Fatalf("unexpected trigger validation body: %#v", body)
	}

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodPost, "/api/automations", `{"name":"bad","query":"x","workspace_mode":"bad"}`))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("workspace status = %d, want %d; body=%s", rr.Code, http.StatusBadRequest, rr.Body.String())
	}
	body = decodeBody(t, rr)
	if body["error"] != "workspace_mode 必须是 auto | worktree | copy | readonly" {
		t.Fatalf("unexpected workspace validation body: %#v", body)
	}

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodPost, "/api/automations", `{"name":"bad","query":"x","trigger":"schedule","cron":"bad cron"}`))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("cron status = %d, want %d; body=%s", rr.Code, http.StatusBadRequest, rr.Body.String())
	}
	body = decodeBody(t, rr)
	if !strings.Contains(body["error"].(string), "cron 表达式无效") {
		t.Fatalf("unexpected cron validation body: %#v", body)
	}
}

func TestAutomationDiscoveryArchiveAPI(t *testing.T) {
	_, h := newTestServer(t)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodPost, "/api/automations/auto_none/archive-no-finding", `{"reason":"empty scan","summary":"nothing to do"}`))
	if rr.Code != http.StatusOK {
		t.Fatalf("archive status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodGet, "/api/automations/discovery-archive", ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("list archive status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	body := decodeBody(t, rr)
	items, ok := body["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("bad discovery archive body: %#v", body)
	}
	item, _ := items[0].(map[string]any)
	if item["automation_id"] != "auto_none" || item["outcome"] != "no_finding" {
		t.Fatalf("bad discovery archive item: %#v", item)
	}
}
func TestRecommendMageCommandsAPI(t *testing.T) {
	_, h := newTestServer(t)
	projectDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte("module example.com/app\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodGet, "/api/settings/mage-command-recommendations?project_dir="+projectDir, ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	body := decodeBody(t, rr)
	items, ok := body["items"].([]any)
	if body["ok"] != true || !ok || len(items) == 0 {
		t.Fatalf("unexpected recommendation body: %#v", body)
	}
	foundGoTest := false
	for _, raw := range items {
		item, _ := raw.(map[string]any)
		if item["id"] == "go-test" {
			foundGoTest = true
			if item["present"] != true {
				t.Fatalf("go-test should be marked present: %#v", item)
			}
		}
	}
	if !foundGoTest {
		t.Fatalf("go-test recommendation missing: %#v", items)
	}
}

func TestAutomationDelete(t *testing.T) {
	_, h := newTestServer(t)

	raw := `{"id":"auto_delete_me","name":"delete me","query":"x","trigger":"manual","enabled":true}`
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodPost, "/api/automations", raw))
	if rr.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body=%s", rr.Code, rr.Body.String())
	}

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodDelete, "/api/automations/auto_delete_me", ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("delete status = %d, body=%s", rr.Code, rr.Body.String())
	}
	body := decodeBody(t, rr)
	if body["ok"] != true || body["id"] != "auto_delete_me" {
		t.Fatalf("unexpected delete body: %#v", body)
	}

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodPost, "/api/automations/auto_delete_me/run", ""))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("run deleted status = %d, want %d; body=%s", rr.Code, http.StatusBadRequest, rr.Body.String())
	}
}

func TestContextDimDetailIncludesBody(t *testing.T) {
	s, h := newTestServer(t)
	if _, err := s.engine.WriteContextDim("workspace", "hello context body"); err != nil {
		t.Fatalf("WriteContextDim failed: %v", err)
	}

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodGet, "/api/context/dims/workspace", ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	body := decodeBody(t, rr)
	dim, ok := body["dim"].(map[string]any)
	if body["ok"] != true || !ok || dim["body"] != "hello context body" {
		t.Fatalf("context dim body missing: %#v", body)
	}
}

func TestContextExportOKFZip(t *testing.T) {
	s, h := newTestServer(t)
	if err := s.engine.WriteContextSummary("hello summary"); err != nil {
		t.Fatalf("WriteContextSummary failed: %v", err)
	}
	if _, err := s.engine.WriteContextDim("workspace", "hello context body"); err != nil {
		t.Fatalf("WriteContextDim failed: %v", err)
	}

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodGet, "/api/context/export?format=okf", ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/zip" {
		t.Fatalf("content-type = %q, want application/zip", ct)
	}
	zr, err := zip.NewReader(bytes.NewReader(rr.Body.Bytes()), int64(rr.Body.Len()))
	if err != nil {
		t.Fatalf("read zip: %v", err)
	}
	files := map[string]string{}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open zip file %s: %v", f.Name, err)
		}
		raw, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			t.Fatalf("read zip file %s: %v", f.Name, err)
		}
		files[f.Name] = string(raw)
	}
	if !strings.Contains(files["index.md"], "gloop_context_bundle") {
		t.Fatalf("index.md missing OKF frontmatter: %q", files["index.md"])
	}
	if !strings.Contains(files["dimensions/workspace.md"], "hello context body") {
		t.Fatalf("workspace dimension missing body: %q", files["dimensions/workspace.md"])
	}

	previewRR := httptest.NewRecorder()
	h.ServeHTTP(previewRR, authedReq(http.MethodGet, "/api/context/export/preview", ""))
	if previewRR.Code != http.StatusOK {
		t.Fatalf("preview status = %d, body=%s", previewRR.Code, previewRR.Body.String())
	}
	previewBody := decodeBody(t, previewRR)
	last, ok := previewBody["last"].(map[string]any)
	if !ok || last["filename"] == "" || last["file_count"].(float64) < 2 {
		t.Fatalf("last export metadata missing: %#v", previewBody)
	}
}

func TestContextExportPreviewOmitsFileContent(t *testing.T) {
	s, h := newTestServer(t)
	if err := s.engine.WriteContextSummary("hello summary"); err != nil {
		t.Fatalf("WriteContextSummary failed: %v", err)
	}
	if _, err := s.engine.WriteContextDim("workspace", "hello context body"); err != nil {
		t.Fatalf("WriteContextDim failed: %v", err)
	}

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodGet, "/api/context/export/preview", ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rr.Code, rr.Body.String())
	}
	body := decodeBody(t, rr)
	preview, ok := body["preview"].(map[string]any)
	if body["ok"] != true || !ok {
		t.Fatalf("unexpected preview body: %#v", body)
	}
	if preview["file_count"].(float64) < 3 || preview["total_size_bytes"].(float64) == 0 {
		t.Fatalf("preview summary missing: %#v", preview)
	}
	files, ok := preview["files"].([]any)
	if !ok || len(files) == 0 {
		t.Fatalf("preview files missing: %#v", preview)
	}
	for _, raw := range files {
		item := raw.(map[string]any)
		if _, exists := item["content"]; exists {
			t.Fatalf("preview must not include file content: %#v", item)
		}
		if item["path"] == "" || item["size_bytes"].(float64) == 0 {
			t.Fatalf("preview file missing path/size: %#v", item)
		}
	}
}

func TestContextExportPreviewDiffsAgainstLastExport(t *testing.T) {
	s, h := newTestServer(t)
	if err := s.engine.WriteContextSummary("hello summary"); err != nil {
		t.Fatalf("WriteContextSummary failed: %v", err)
	}
	if _, err := s.engine.WriteContextDim("workspace", "hello context body"); err != nil {
		t.Fatalf("WriteContextDim failed: %v", err)
	}

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodGet, "/api/context/export?format=okf", ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("export status = %d, body=%s", rr.Code, rr.Body.String())
	}
	if _, err := s.engine.WriteContextDim("custom_notes", "new notes"); err != nil {
		t.Fatalf("WriteContextDim custom_notes failed: %v", err)
	}

	previewRR := httptest.NewRecorder()
	h.ServeHTTP(previewRR, authedReq(http.MethodGet, "/api/context/export/preview", ""))
	if previewRR.Code != http.StatusOK {
		t.Fatalf("preview status = %d, body=%s", previewRR.Code, previewRR.Body.String())
	}
	body := decodeBody(t, previewRR)
	preview := body["preview"].(map[string]any)
	diff, ok := preview["diff"].(map[string]any)
	if !ok {
		t.Fatalf("preview diff missing: %#v", preview)
	}
	added := diff["added"].([]any)
	if len(added) != 1 || added[0].(map[string]any)["path"] != "dimensions/custom_notes.md" {
		t.Fatalf("added diff mismatch: %#v", diff)
	}
	if diff["unchanged"].(float64) == 0 {
		t.Fatalf("expected unchanged files in diff: %#v", diff)
	}
}

func TestContextExportPreviewDiffDetectsSameSizeContentChange(t *testing.T) {
	s, h := newTestServer(t)
	if _, err := s.engine.WriteContextDim("workspace", "abc"); err != nil {
		t.Fatalf("WriteContextDim failed: %v", err)
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodGet, "/api/context/export?format=okf", ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("export status = %d, body=%s", rr.Code, rr.Body.String())
	}
	if _, err := s.engine.WriteContextDim("workspace", "xyz"); err != nil {
		t.Fatalf("WriteContextDim update failed: %v", err)
	}

	previewRR := httptest.NewRecorder()
	h.ServeHTTP(previewRR, authedReq(http.MethodGet, "/api/context/export/preview", ""))
	if previewRR.Code != http.StatusOK {
		t.Fatalf("preview status = %d, body=%s", previewRR.Code, previewRR.Body.String())
	}
	body := decodeBody(t, previewRR)
	diff := body["preview"].(map[string]any)["diff"].(map[string]any)
	changed := diff["changed"].([]any)
	if len(changed) != 1 || changed[0].(map[string]any)["path"] != "dimensions/workspace.md" {
		t.Fatalf("changed diff should include same-size content update: %#v", diff)
	}
}

func TestWriteContextDimPublishesContextUpdated(t *testing.T) {
	s, h := newTestServer(t)
	ch, unsubscribe := s.bus.Subscribe()
	defer unsubscribe()

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodPut, "/api/context/dims/workspace", `{"content":"fresh context body"}`))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rr.Code, rr.Body.String())
	}
	body := decodeBody(t, rr)
	if body["ok"] != true {
		t.Fatalf("unexpected body: %#v", body)
	}

	select {
	case ev := <-ch:
		if ev.Type != events.EvtContextUpdated {
			t.Fatalf("event type = %s, want %s", ev.Type, events.EvtContextUpdated)
		}
		if ev.QuestID != "" {
			t.Fatalf("context event should be system-level, got quest_id=%q", ev.QuestID)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for context.updated")
	}
}

func TestShutdownClosesSSEStreams(t *testing.T) {
	s, h := newTestServer(t)
	req := authedReq(http.MethodGet, "/api/stream", "")
	rr := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		h.ServeHTTP(rr, req)
		close(done)
	}()

	deadline := time.After(time.Second)
	for !strings.Contains(rr.Body.String(), `"type":"hello"`) {
		select {
		case <-done:
			t.Fatalf("stream returned before hello: %s", rr.Body.String())
		case <-deadline:
			t.Fatal("timed out waiting for stream hello")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	start := time.Now()
	err := s.Shutdown(context.Background())
	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("SSE stream did not close after shutdown")
	}
	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Fatalf("Shutdown with SSE stream took %s, want under 500ms", elapsed)
	}
}

func TestShutdownWithOpenSSEConnectionReturnsQuickly(t *testing.T) {
	t.Setenv("GLOOP_NO_UPDATE_NOTIFIER", "1")
	root, err := fsstore.Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open root: %v", err)
	}
	cfg := fsstore.DefaultGlobalConfig()
	cfg.Host = "127.0.0.1"
	cfg.Port = 0
	cfg.DefaultWorkingDir = t.TempDir()
	tok := &auth.TokenStore{Token: testToken}
	s, err := New(cfg, nil, events.NewBus(), root, tok)
	if err != nil {
		t.Fatalf("New server: %v", err)
	}

	errCh := make(chan error, 1)
	go func() {
		err := s.Start()
		if err == http.ErrServerClosed {
			err = nil
		}
		errCh <- err
	}()

	client := &http.Client{Timeout: 2 * time.Second}
	req, err := http.NewRequest(http.MethodGet, "http://"+s.Addr+"/api/stream", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+testToken)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("open SSE stream: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("stream status = %d", resp.StatusCode)
	}
	reader := bufio.NewReader(resp.Body)
	line, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read stream hello: %v", err)
	}
	if !strings.Contains(line, `"type":"hello"`) {
		t.Fatalf("first stream line = %q, want hello", line)
	}
	if _, err := reader.ReadString('\n'); err != nil {
		t.Fatalf("read stream hello separator: %v", err)
	}

	start := time.Now()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := s.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Fatalf("Shutdown with open SSE connection took %s, want under 500ms", elapsed)
	}

	if _, readErr := reader.ReadString('\n'); readErr == nil {
		t.Fatal("SSE response body remained readable after shutdown")
	}
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("server returned error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("server did not exit after shutdown")
	}
}

func TestOfficialAutomationAPIsRejectMutation(t *testing.T) {
	s, h := newTestServer(t)
	// 官方自动化（通过 source 字段标记）不可修改删除
	if err := s.engine.SaveAutomation(&fsstore.AutomationConfig{
		ID:      "auto_official_template",
		Name:    "official template",
		Query:   "x",
		Trigger: fsstore.TriggerManual,
		Enabled: true,
		Source:  fsstore.SourceOfficial,
		Tags:    []string{"template"},
	}); err != nil {
		t.Fatalf("save official template: %v", err)
	}
	// 官方上下文自动化同样受保护
	if err := s.engine.SaveAutomation(&fsstore.AutomationConfig{
		ID:        "auto_official_context",
		Name:      "official context",
		Query:     "x",
		Trigger:   fsstore.TriggerSchedule,
		Cron:      "0 9 * * *",
		Enabled:   true,
		Source:    fsstore.SourceOfficial,
		Tags:      []string{"context"},
		AutoStart: true,
	}); err != nil {
		t.Fatalf("save official context: %v", err)
	}
	// 用户自建的 context 自动化不受保护，可以正常修改
	if err := s.engine.SaveAutomation(&fsstore.AutomationConfig{
		ID:        "auto_user_context",
		Name:      "user context",
		Query:     "x",
		Trigger:   fsstore.TriggerManual,
		Enabled:   true,
		Source:    fsstore.SourceUser,
		Tags:      []string{"context"},
		AutoStart: false,
	}); err != nil {
		t.Fatalf("save user context: %v", err)
	}

	// 官方自动化不能修改删除
	protectedCases := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "official update", method: http.MethodPatch, path: "/api/automations/auto_official_template", body: `{"name":"mutated"}`},
		{name: "official context update", method: http.MethodPatch, path: "/api/automations/auto_official_context", body: `{"name":"mutated"}`},
		{name: "official delete", method: http.MethodDelete, path: "/api/automations/auto_official_template"},
		{name: "official context delete", method: http.MethodDelete, path: "/api/automations/auto_official_context"},
	}
	for _, tc := range protectedCases {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, authedReq(tc.method, tc.path, tc.body))
			if rr.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusForbidden, rr.Body.String())
			}
			body := decodeBody(t, rr)
			if body["ok"] != false || !strings.Contains(body["error"].(string), "不能") {
				t.Fatalf("unexpected official automation body: %#v", body)
			}
		})
	}

	// 用户自建的 context 自动化可以正常修改
	t.Run("user context can update", func(t *testing.T) {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, authedReq(http.MethodPatch, "/api/automations/auto_user_context", `{"name":"updated"}`))
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
		}
		body := decodeBody(t, rr)
		if body["ok"] != true {
			t.Fatalf("unexpected update body: %#v", body)
		}
	})

	// 用户自建的 context 自动化可以正常删除
	t.Run("user context can delete", func(t *testing.T) {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, authedReq(http.MethodDelete, "/api/automations/auto_user_context", ""))
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
		}
		body := decodeBody(t, rr)
		if body["ok"] != true {
			t.Fatalf("unexpected delete body: %#v", body)
		}
	})
}

func TestContextAutomationCanToggleEnabledFromAutomationAPI(t *testing.T) {
	s, h := newTestServer(t)
	if err := s.engine.SaveAutomation(&fsstore.AutomationConfig{
		ID:        "auto_protected_context",
		Name:      "protected context",
		Query:     "refresh context",
		Trigger:   fsstore.TriggerSchedule,
		Cron:      "0 9 * * *",
		Enabled:   true,
		Tags:      []string{"context"},
		AutoStart: true,
	}); err != nil {
		t.Fatalf("save protected context: %v", err)
	}

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodPost, "/api/automations/auto_protected_context/disable", ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("disable context status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	body := decodeBody(t, rr)
	if body["ok"] != true || body["enabled"] != false {
		t.Fatalf("unexpected disable body: %#v", body)
	}

	got, err := s.engine.GetAutomation("auto_protected_context")
	if err != nil {
		t.Fatalf("GetAutomation failed: %v", err)
	}
	if got.Enabled {
		t.Fatalf("context automation should be disabled: %+v", got)
	}
}

func TestContextAutomationCanRunFromAutomationAPI(t *testing.T) {
	s, h := newTestServer(t)
	configureRunnableAdventurers(t, s)
	if err := s.engine.SaveAutomation(&fsstore.AutomationConfig{
		ID:        "auto_protected_context",
		Name:      "protected context",
		Query:     "refresh context",
		Trigger:   fsstore.TriggerManual,
		Enabled:   true,
		Tags:      []string{"context"},
		AutoStart: false,
	}); err != nil {
		t.Fatalf("save protected context: %v", err)
	}

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodPost, "/api/automations/auto_protected_context/run", ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("run context status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	body := decodeBody(t, rr)
	if body["ok"] != true || body["qid"] == "" {
		t.Fatalf("unexpected context run body: %#v", body)
	}
}

func TestExecutorsReturnsMockFallback(t *testing.T) {
	_, h := newTestServer(t)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodGet, "/api/executors", ""))

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	body := decodeBody(t, rr)
	items, ok := body["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("items = %#v, want one mock executor", body["items"])
	}
	item, ok := items[0].(map[string]any)
	if !ok || item["type"] != "mock" || item["enabled"] != true {
		t.Fatalf("unexpected executor item: %#v", items[0])
	}
}

func TestCreateAdventurerRejectsInvalidActiveAgent(t *testing.T) {
	s, h := newTestServer(t)
	if err := s.root.SaveAgent(&fsstore.AgentConfig{
		Name:         "relay_disabled",
		Type:         model.AgentTypeCLI,
		Command:      "relay",
		DefaultModel: "auto",
		Enabled:      false,
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.root.SaveAgent(&fsstore.AgentConfig{
		Name:         "mock_custom",
		Type:         model.AgentTypeMock,
		Command:      "mock",
		DefaultModel: "mock-fast",
		Enabled:      true,
	}); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		raw  string
	}{
		{
			name: "empty agent",
			raw:  `{"name":"no agent","class":"warrior","status":"active"}`,
		},
		{
			name: "disabled agent",
			raw:  `{"name":"disabled","class":"warrior","agent":"relay_disabled","status":"active"}`,
		},
		{
			name: "mock agent",
			raw:  `{"name":"mock","class":"warrior","agent":"mock_custom","status":"active"}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, authedReq(http.MethodPost, "/api/adventurers", tc.raw))
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusBadRequest, rr.Body.String())
			}
		})
	}
}

func TestAdventurerModelOverridePersistsThroughCreateUpdateActivate(t *testing.T) {
	s, h := newTestServer(t)
	if err := s.root.SaveAgent(&fsstore.AgentConfig{
		Name:         "relay_enabled",
		Type:         model.AgentTypeCLI,
		Command:      "relay",
		DefaultModel: "agent-default",
		Enabled:      true,
	}); err != nil {
		t.Fatal(err)
	}
	s.engine.RegisterExecutor("relay_enabled", executor.NewMockExecutor("exe_relay_enabled"))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodPost, "/api/adventurers", `{"id":"model_warrior","name":"model warrior","class":"warrior","agent":"relay_enabled","model":"ark/custom","status":"active"}`))
	if rr.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body=%s", rr.Code, rr.Body.String())
	}
	body := decodeBody(t, rr)
	item, ok := body["item"].(map[string]any)
	if !ok || item["model"] != "ark/custom" {
		t.Fatalf("create model not persisted: %#v", body)
	}
	id := item["id"].(string)

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodPost, "/api/adventurers/"+id, `{"model":"ark/updated"}`))
	if rr.Code != http.StatusOK {
		t.Fatalf("update status = %d, body=%s", rr.Code, rr.Body.String())
	}
	body = decodeBody(t, rr)
	item, ok = body["item"].(map[string]any)
	if !ok || item["model"] != "ark/updated" {
		t.Fatalf("update model not persisted: %#v", body)
	}

	if _, err := s.engine.SaveAdventurer(&fsstore.AdventurerFile{
		ID:     "pending_model_mage",
		Name:   "pending model mage",
		Class:  model.ClassMage,
		Status: model.AdventurerPendingSetup,
		Level:  1,
	}); err != nil {
		t.Fatal(err)
	}
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodPost, "/api/adventurers/pending_model_mage/activate", `{"agent":"relay_enabled","model":"ark/activated"}`))
	if rr.Code != http.StatusOK {
		t.Fatalf("activate status = %d, body=%s", rr.Code, rr.Body.String())
	}
	body = decodeBody(t, rr)
	item, ok = body["item"].(map[string]any)
	if !ok || item["model"] != "ark/activated" || item["status"] != "active" {
		t.Fatalf("activate model not persisted: %#v", body)
	}
}

func TestExecutorUpsertPersistsAgentDefaultModel(t *testing.T) {
	_, h := newTestServer(t)

	raw := `{"name":"relay_custom","type":"cli","command":"relay","args":["-p"],"default_model":"ark/seed-code-0611","enabled":true}`
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodPost, "/api/executors", raw))
	if rr.Code != http.StatusOK {
		t.Fatalf("upsert status = %d, body=%s", rr.Code, rr.Body.String())
	}
	body := decodeBody(t, rr)
	item, ok := body["item"].(map[string]any)
	if body["ok"] != true || !ok {
		t.Fatalf("unexpected upsert body: %#v", body)
	}
	if item["agent"] != "relay_custom" || item["command"] != "relay" {
		t.Fatalf("unexpected agent item: %#v", item)
	}
	if item["default_model"] != "ark/seed-code-0611" {
		t.Fatalf("default_model = %#v", item["default_model"])
	}

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodGet, "/api/executors", ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("list status = %d, body=%s", rr.Code, rr.Body.String())
	}
	body = decodeBody(t, rr)
	items, _ := body["items"].([]any)
	var found map[string]any
	for _, rawItem := range items {
		m, _ := rawItem.(map[string]any)
		if m["agent"] == "relay_custom" {
			found = m
			break
		}
	}
	if found == nil {
		t.Fatalf("relay_custom not found in executors: %#v", items)
	}
	if found["default_model"] != "ark/seed-code-0611" {
		t.Fatalf("listed default_model = %#v", found["default_model"])
	}
	if _, ok := found["models"]; ok {
		t.Fatalf("executors should not expose model lists: %#v", found["models"])
	}
}

func TestExecutorUpsertRejectsBuiltinTypeChange(t *testing.T) {
	s, h := newTestServer(t)
	if err := s.root.SaveAgent(&fsstore.AgentConfig{
		Name:         "relay",
		Type:         model.AgentTypeCLI,
		Command:      "relay",
		Args:         []string{"-p"},
		DefaultModel: "auto",
		Enabled:      false,
		Official:     true,
	}); err != nil {
		t.Fatal(err)
	}

	raw := `{"name":"relay","type":"acp","command":"relay","args":["-p"],"default_model":"auto","enabled":false}`
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodPost, "/api/executors", raw))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("upsert status = %d, want %d; body=%s", rr.Code, http.StatusBadRequest, rr.Body.String())
	}
	agent, err := s.root.GetAgent("relay")
	if err != nil {
		t.Fatalf("GetAgent failed: %v", err)
	}
	if agent.Type != model.AgentTypeCLI {
		t.Fatalf("builtin relay type changed to %s", agent.Type)
	}
}

func TestExecutorsMarksBuiltinAgentOfficial(t *testing.T) {
	s, h := newTestServer(t)
	if err := s.root.SaveAgent(&fsstore.AgentConfig{
		Name:         "relay",
		Type:         model.AgentTypeCLI,
		Command:      "relay",
		DefaultModel: "auto",
		Enabled:      false,
	}); err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodGet, "/api/executors", ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("list status = %d, body=%s", rr.Code, rr.Body.String())
	}
	body := decodeBody(t, rr)
	items, _ := body["items"].([]any)
	var found map[string]any
	for _, rawItem := range items {
		m, _ := rawItem.(map[string]any)
		if m["agent"] == "relay" {
			found = m
			break
		}
	}
	if found == nil {
		t.Fatalf("relay not found: %#v", items)
	}
	if found["official"] != true {
		t.Fatalf("relay official = %#v, item=%#v", found["official"], found)
	}
}

func TestSettingsMarksBuiltinAgentOfficial(t *testing.T) {
	s, h := newTestServer(t)
	if err := s.root.SaveAgent(&fsstore.AgentConfig{
		Name:         "aiden_x_claude",
		Type:         model.AgentTypeCLI,
		Command:      "aiden",
		Args:         []string{"x", "claude", "--stream-json"},
		DefaultModel: "auto",
		Enabled:      false,
	}); err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodGet, "/api/settings", ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("settings status = %d, body=%s", rr.Code, rr.Body.String())
	}
	body := decodeBody(t, rr)
	items, _ := body["agents"].([]any)
	var found map[string]any
	for _, rawItem := range items {
		m, _ := rawItem.(map[string]any)
		if m["agent"] == "aiden_x_claude" {
			found = m
			break
		}
	}
	if found == nil {
		t.Fatalf("aiden_x_claude not found in settings agents: %#v", items)
	}
	if found["official"] != true || found["type"] != string(model.AgentTypeCLI) || found["name"] != "aiden_x_claude" {
		t.Fatalf("unexpected settings agent item: %#v", found)
	}
}

func TestExecutorDeleteRejectsBuiltinAgent(t *testing.T) {
	s, h := newTestServer(t)
	if err := s.root.SaveAgent(&fsstore.AgentConfig{
		Name:         "relay",
		Type:         model.AgentTypeCLI,
		Command:      "relay",
		DefaultModel: "auto",
		Enabled:      false,
		Official:     true,
	}); err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodDelete, "/api/executors/relay", ""))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("delete status = %d, want %d; body=%s", rr.Code, http.StatusBadRequest, rr.Body.String())
	}
	if _, err := s.root.GetAgent("relay"); err != nil {
		t.Fatalf("builtin relay should remain after delete rejection: %v", err)
	}
}

func TestExecutorsPreferConfiguredDefaultModel(t *testing.T) {
	s, h := newTestServer(t)
	agent := &fsstore.AgentConfig{
		Name:         "mock_custom",
		Type:         model.AgentTypeMock,
		Command:      "mock",
		DefaultModel: "configured-default",
		Enabled:      true,
	}
	if err := s.root.SaveAgent(agent); err != nil {
		t.Fatal(err)
	}
	s.engine.RegisterExecutor("mock_custom", executor.NewMockExecutor("exe_mock_custom"))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodGet, "/api/executors", ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("list status = %d, body=%s", rr.Code, rr.Body.String())
	}
	body := decodeBody(t, rr)
	items, _ := body["items"].([]any)
	var found map[string]any
	for _, rawItem := range items {
		m, _ := rawItem.(map[string]any)
		if m["agent"] == "mock_custom" {
			found = m
			break
		}
	}
	if found == nil {
		t.Fatalf("mock_custom not found: %#v", items)
	}
	if found["default_model"] != "configured-default" {
		t.Fatalf("default_model = %#v", found["default_model"])
	}
	if _, ok := found["models"]; ok {
		t.Fatalf("executors should not expose model lists: %#v", found["models"])
	}
}

func TestExecutorToggleReturnsCompleteAgentInfo(t *testing.T) {
	_, h := newTestServer(t)

	raw := `{"name":"toggle_custom","type":"cli","command":"relay","args":["-p"],"default_model":"auto","enabled":false}`
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodPost, "/api/executors", raw))
	if rr.Code != http.StatusOK {
		t.Fatalf("upsert status = %d, body=%s", rr.Code, rr.Body.String())
	}

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodPost, "/api/executors/toggle_custom/enable", ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("enable status = %d, body=%s", rr.Code, rr.Body.String())
	}
	body := decodeBody(t, rr)
	item, _ := body["item"].(map[string]any)
	if item["agent"] != "toggle_custom" || item["command"] != "relay" || item["enabled"] != true {
		t.Fatalf("incomplete toggle item: %#v", item)
	}
	if item["default_model"] != "auto" {
		t.Fatalf("toggle item default_model = %#v", item["default_model"])
	}
	if _, ok := item["models"]; ok {
		t.Fatalf("toggle item should not expose model lists: %#v", item["models"])
	}
}

func TestExecutorEnableFailureDoesNotPersistEnabled(t *testing.T) {
	s, h := newTestServer(t)

	raw := `{"name":"bad_unknown_cli","type":"cli","command":"unknown-cli-agent","enabled":false}`
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodPost, "/api/executors", raw))
	if rr.Code != http.StatusOK {
		t.Fatalf("upsert status = %d, body=%s", rr.Code, rr.Body.String())
	}

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodPost, "/api/executors/bad_unknown_cli/enable", ""))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("enable status = %d, want %d; body=%s", rr.Code, http.StatusBadRequest, rr.Body.String())
	}

	agent, err := s.root.GetAgent("bad_unknown_cli")
	if err != nil {
		t.Fatalf("GetAgent failed: %v", err)
	}
	if agent.Enabled {
		t.Fatalf("agent should remain disabled after enable failure: %+v", agent)
	}
	if s.engine.HasExecutor("bad_unknown_cli") {
		t.Fatal("failed enable should not register runtime executor")
	}
}

func TestExecutorDeleteRemovesUnusedAgent(t *testing.T) {
	s, h := newTestServer(t)

	raw := `{"name":"delete_custom","type":"cli","command":"relay","enabled":true}`
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodPost, "/api/executors", raw))
	if rr.Code != http.StatusOK {
		t.Fatalf("upsert status = %d, body=%s", rr.Code, rr.Body.String())
	}
	if _, err := s.root.GetAgent("delete_custom"); err != nil {
		t.Fatalf("agent should exist before delete: %v", err)
	}

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodDelete, "/api/executors/delete_custom", ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("delete status = %d, body=%s", rr.Code, rr.Body.String())
	}
	if _, err := s.root.GetAgent("delete_custom"); err == nil {
		t.Fatal("agent should be removed after delete")
	}
	if s.engine.HasExecutor("delete_custom") {
		t.Fatal("deleted agent should unregister runtime executor")
	}
}

func TestExecutorDeleteRejectsActiveAdventurerReference(t *testing.T) {
	s, h := newTestServer(t)

	raw := `{"name":"used_custom","type":"cli","command":"relay","enabled":true}`
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodPost, "/api/executors", raw))
	if rr.Code != http.StatusOK {
		t.Fatalf("upsert status = %d, body=%s", rr.Code, rr.Body.String())
	}
	if _, err := s.engine.SaveAdventurer(&fsstore.AdventurerFile{
		ID:     "adv_uses_custom",
		Name:   "uses custom",
		Class:  model.ClassWarrior,
		Status: model.AdventurerActive,
		Agent:  "used_custom",
		Level:  1,
	}); err != nil {
		t.Fatalf("SaveAdventurer failed: %v", err)
	}

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodDelete, "/api/executors/used_custom", ""))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("delete status = %d, want %d; body=%s", rr.Code, http.StatusBadRequest, rr.Body.String())
	}
	if _, err := s.root.GetAgent("used_custom"); err != nil {
		t.Fatalf("referenced agent should remain: %v", err)
	}
}

func TestQuestTraceIncludesContextPackBlocks(t *testing.T) {
	s, h := newTestServer(t)
	qs := fsstore.NewQuestStore(s.root)
	q := &fsstore.QuestMeta{
		ID:          "qst_trace_context",
		ShortID:     "trace",
		Query:       "trace context",
		Type:        model.QuestTypeExecute,
		Status:      model.QuestStatusRunning,
		CreatedBy:   "user",
		CreatedAtMs: 1000,
	}
	if err := qs.CreateQuest(q); err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	if err := qs.AppendSessionRow(q.ID, "warrior_0", &fsstore.QuestSessionRow{
		Seq:       1,
		Timestamp: 2000,
		Kind:      "context_pack",
		SessionID: "warrior_0",
		Role:      "user",
		Content:   "<gloop_context>rendered</gloop_context>",
		Phase:     0,
		Meta: map[string]any{
			"context_pack": map[string]any{"kind": "quest_execution"},
			"context_pack_blocks": []any{
				map[string]any{"name": "quest_user_intent", "trust": "user"},
			},
		},
	}); err != nil {
		t.Fatalf("AppendSessionRow failed: %v", err)
	}

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodGet, "/api/quests/"+q.ID+"/trace", ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("trace status = %d, body=%s", rr.Code, rr.Body.String())
	}
	body := decodeBody(t, rr)
	items, ok := body["items"].([]any)
	if !ok || len(items) == 0 {
		t.Fatalf("items = %#v, want context_pack entry", body["items"])
	}
	for _, raw := range items {
		item := raw.(map[string]any)
		if item["kind"] != "context_pack" {
			continue
		}
		if !strings.Contains(item["content"].(string), "rendered") {
			t.Fatalf("context_pack content missing rendered text: %#v", item)
		}
		meta := item["meta"].(map[string]any)
		if _, ok := meta["summary"]; !ok {
			t.Fatalf("context_pack meta missing summary: %#v", meta)
		}
		blocks, ok := meta["blocks"].([]any)
		if !ok || len(blocks) != 1 {
			t.Fatalf("context_pack blocks = %#v, want one", meta["blocks"])
		}
		return
	}
	t.Fatalf("context_pack entry not found: %#v", items)
}

func TestQuestTraceIncludesPhaseToolResult(t *testing.T) {
	s, h := newTestServer(t)
	qs := fsstore.NewQuestStore(s.root)
	q := &fsstore.QuestMeta{
		ID:          "qst_trace_tool_result",
		ShortID:     "trtool",
		Query:       "trace tool result",
		Type:        model.QuestTypeExecute,
		Status:      model.QuestStatusSuccess,
		CreatedBy:   "user",
		CreatedAtMs: 1000,
	}
	if err := qs.CreateQuest(q); err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	result := `{"ok":true,"message":"阶段结论已提交","data":{"summary":"warrior delivery summary"}}`
	if err := qs.AppendSessionRow(q.ID, "warrior_0", &fsstore.QuestSessionRow{
		Seq:       1,
		Timestamp: 2000,
		Kind:      "tool_result",
		SessionID: "warrior_0",
		Role:      "tool",
		ToolName:  "gloop_cli_signal",
		Content:   result,
		Phase:     0,
	}); err != nil {
		t.Fatalf("AppendSessionRow failed: %v", err)
	}

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodGet, "/api/quests/"+q.ID+"/trace", ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("trace status = %d, body=%s", rr.Code, rr.Body.String())
	}
	body := decodeBody(t, rr)
	items, ok := body["items"].([]any)
	if !ok || len(items) == 0 {
		t.Fatalf("items = %#v, want tool_result entry", body["items"])
	}
	for _, raw := range items {
		item := raw.(map[string]any)
		if item["kind"] != "tool_result" {
			continue
		}
		if item["tool_name"] != "gloop_cli_signal" || !strings.Contains(item["content"].(string), "warrior delivery summary") {
			t.Fatalf("bad tool_result entry: %#v", item)
		}
		return
	}
	t.Fatalf("tool_result entry not found: %#v", items)
}

func TestQuestTraceAllScopeRoundAndLimit(t *testing.T) {
	s, h := newTestServer(t)
	qs := fsstore.NewQuestStore(s.root)
	q := &fsstore.QuestMeta{
		ID:          "qst_trace_all",
		ShortID:     "trall",
		Query:       "trace all",
		Type:        model.QuestTypeExecute,
		Status:      model.QuestStatusRunning,
		ReworkCount: 1,
		CreatedBy:   "user",
		CreatedAtMs: 1000,
	}
	if err := qs.CreateQuest(q); err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	for _, tc := range []struct {
		sid     string
		content string
		ts      int64
	}{
		{sid: "warrior_0", content: "round zero", ts: 2000},
		{sid: "warrior_1", content: "round one", ts: 3000},
		{sid: "mage_1", content: "mage one", ts: 4000},
	} {
		if err := qs.AppendSessionRow(q.ID, tc.sid, &fsstore.QuestSessionRow{
			Seq:       1,
			Timestamp: tc.ts,
			Kind:      "message",
			SessionID: tc.sid,
			Role:      "assistant",
			Content:   tc.content,
			Phase:     0,
		}); err != nil {
			t.Fatalf("AppendSessionRow %s failed: %v", tc.sid, err)
		}
	}
	sids, err := qs.ListSessionIDs(q.ID)
	if err != nil {
		t.Fatalf("ListSessionIDs failed: %v", err)
	}
	if len(sids) != 3 {
		t.Fatalf("session ids = %#v, want three", sids)
	}
	filtered := traceSessionIDs(qs, q.ID, q.ReworkCount, "all", 1)
	if len(filtered) != 2 {
		t.Fatalf("filtered session ids = %#v, want two round 1 sessions", filtered)
	}

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodGet, "/api/quests/"+q.ID+"/trace?scope=all&round=1&limit=1", ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("trace status = %d, body=%s", rr.Code, rr.Body.String())
	}
	body := decodeBody(t, rr)
	if body["scope"] != "all" {
		t.Fatalf("scope = %#v, want all", body["scope"])
	}
	if round, _ := body["round"].(float64); int(round) != 1 {
		t.Fatalf("round = %#v, want 1", body["round"])
	}
	if truncated, _ := body["truncated"].(bool); !truncated {
		t.Fatalf("trace should be truncated when entries exceed limit: %#v", body)
	}
	items, ok := body["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("items = %#v, want one limited item", body["items"])
	}
	item := items[0].(map[string]any)
	if item["sid"] != "warrior_1" || item["content"] != "round one" {
		t.Fatalf("round filter should return first round 1 item, got %#v", item)
	}
	if gotRound, _ := item["round"].(float64); int(gotRound) != 1 {
		t.Fatalf("item round = %#v, want 1", item["round"])
	}
}

func TestGetQuestThread_ReturnsPosts(t *testing.T) {
	s, h := newTestServer(t)
	configureRunnableAdventurers(t, s)

	// Create quest via HTTP API.
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodPost, "/api/quests", `{"query":"thread view test quest","type":"execute"}`))
	if rr.Code != http.StatusOK {
		t.Fatalf("create quest status = %d, body=%s", rr.Code, rr.Body.String())
	}
	createBody := decodeBody(t, rr)
	qid, _ := createBody["qid"].(string)
	if qid == "" {
		t.Fatal("create quest returned empty qid")
	}

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodGet, "/api/quests/"+qid+"/thread", ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /thread status = %d, body=%s", rr.Code, rr.Body.String())
	}
	body := decodeBody(t, rr)
	if ok, _ := body["ok"].(bool); !ok {
		t.Fatal("ok should be true")
	}

	questInfo, ok := body["quest"].(map[string]any)
	if !ok {
		t.Fatal("response should include quest")
	}
	if questInfo["id"] != qid {
		t.Fatalf("quest.id = %v, want %s", questInfo["id"], qid)
	}

	posts, ok := body["posts"].([]any)
	if !ok {
		t.Fatal("response should include posts")
	}
	if len(posts) == 0 {
		t.Fatal("posts should not be empty (at least root post from repair-on-read)")
	}

	// Second GET should be idempotent (repair-on-read doesn't duplicate).
	rr2 := httptest.NewRecorder()
	h.ServeHTTP(rr2, authedReq(http.MethodGet, "/api/quests/"+qid+"/thread", ""))
	body2 := decodeBody(t, rr2)
	posts2, _ := body2["posts"].([]any)
	if len(posts2) != len(posts) {
		t.Fatalf("second GET returned %d posts, want %d (idempotent repair-on-read)", len(posts2), len(posts))
	}
}

func TestGetQuestThread_NotFound(t *testing.T) {
	_, h := newTestServer(t)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodGet, "/api/quests/nonexistent/thread", ""))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("GET /thread for nonexistent quest status = %d, want 404", rr.Code)
	}
}

func TestGetQuestThread_NormalQuest_ActorsAndEntrypoints(t *testing.T) {
	s, h := newTestServer(t)
	configureRunnableAdventurers(t, s)

	q, err := s.engine.CreateQuest(t.Context(), "normal thread quest", model.QuestTypeExecute, "", orchestrator.CreateQuestOptions{
		WarriorID: "adv_test_warrior",
		MageID:    "adv_test_mage",
	})
	if err != nil {
		t.Fatalf("CreateQuest: %v", err)
	}

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodGet, "/api/quests/"+q.ID+"/thread", ""))
	body := decodeBody(t, rr)

	actors, ok := body["actors"].([]any)
	if !ok || len(actors) == 0 {
		t.Fatal("response should include actors")
	}
	hasMaker := false
	hasChecker := false
	for _, a := range actors {
		m := a.(map[string]any)
		if m["role"] == "maker" {
			hasMaker = true
		}
		if m["role"] == "checker" {
			hasChecker = true
		}
	}
	if !hasMaker || !hasChecker {
		t.Fatalf("actors should include maker and checker: %+v", actors)
	}

	entries, ok := body["action_entrypoints"].([]any)
	if !ok || len(entries) == 0 {
		t.Fatal("response should include action_entrypoints")
	}
	hasAddComment := false
	for _, e := range entries {
		m := e.(map[string]any)
		if m["action_type"] == "add_comment" {
			hasAddComment = true
		}
	}
	if !hasAddComment {
		t.Fatalf("running quest should have add_comment entrypoint: %+v", entries)
	}

	fanout, _ := body["fanout_summary"]
	if fanout != nil {
		t.Fatal("normal quest should have nil fanout_summary")
	}
}

func TestGetQuestThread_WaitingInputEntrypoint(t *testing.T) {
	s, h := newTestServer(t)
	configureRunnableAdventurers(t, s)

	q, err := s.engine.CreateQuest(t.Context(), "waiting input quest", model.QuestTypeExecute, "", orchestrator.CreateQuestOptions{
		WarriorID: "adv_test_warrior",
		MageID:    "adv_test_mage",
	})
	if err != nil {
		t.Fatalf("CreateQuest: %v", err)
	}
	q.Status = model.QuestStatusWaitingInput
	if err := fsstore.NewQuestStore(s.root).SaveQuest(q); err != nil {
		t.Fatalf("SaveQuest: %v", err)
	}

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodGet, "/api/quests/"+q.ID+"/thread", ""))
	body := decodeBody(t, rr)

	entries, _ := body["action_entrypoints"].([]any)
	hasAnswer := false
	for _, e := range entries {
		m := e.(map[string]any)
		if m["action_type"] == "answer" && m["target_section"] == "composer" {
			hasAnswer = true
		}
	}
	if !hasAnswer {
		t.Fatalf("waiting_input quest should have answer entrypoint: %+v", entries)
	}
}

func TestGetQuestThread_BlockedEntrypoint(t *testing.T) {
	s, h := newTestServer(t)
	configureRunnableAdventurers(t, s)

	q, err := s.engine.CreateQuest(t.Context(), "blocked quest", model.QuestTypeExecute, "", orchestrator.CreateQuestOptions{
		WarriorID: "adv_test_warrior",
		MageID:    "adv_test_mage",
	})
	if err != nil {
		t.Fatalf("CreateQuest: %v", err)
	}
	q.Status = model.QuestStatusBlocked
	if err := fsstore.NewQuestStore(s.root).SaveQuest(q); err != nil {
		t.Fatalf("SaveQuest: %v", err)
	}

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodGet, "/api/quests/"+q.ID+"/thread", ""))
	body := decodeBody(t, rr)

	entries, _ := body["action_entrypoints"].([]any)
	hasResolveBlocked := false
	for _, e := range entries {
		m := e.(map[string]any)
		if m["action_type"] == "resolve_blocked" && m["target_section"] == "blocked_panel" {
			hasResolveBlocked = true
		}
	}
	if !hasResolveBlocked {
		t.Fatalf("blocked quest should have resolve_blocked entrypoint: %+v", entries)
	}
}

func TestGetQuestThread_UserReviewEntrypoint(t *testing.T) {
	s, h := newTestServer(t)
	configureRunnableAdventurers(t, s)

	q, err := s.engine.CreateQuest(t.Context(), "user review quest", model.QuestTypeExecute, "", orchestrator.CreateQuestOptions{
		WarriorID: "adv_test_warrior",
		MageID:    "adv_test_mage",
	})
	if err != nil {
		t.Fatalf("CreateQuest: %v", err)
	}
	q.Status = model.QuestStatusUserReview
	if err := fsstore.NewQuestStore(s.root).SaveQuest(q); err != nil {
		t.Fatalf("SaveQuest: %v", err)
	}

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodGet, "/api/quests/"+q.ID+"/thread", ""))
	body := decodeBody(t, rr)

	entries, _ := body["action_entrypoints"].([]any)
	hasResolveReview := false
	for _, e := range entries {
		m := e.(map[string]any)
		if m["action_type"] == "resolve_review" && m["target_section"] == "review_panel" {
			hasResolveReview = true
		}
	}
	if !hasResolveReview {
		t.Fatalf("user_review quest should have resolve_review entrypoint: %+v", entries)
	}
}

func TestGetQuestThread_AutomationActor(t *testing.T) {
	s, h := newTestServer(t)
	configureRunnableAdventurers(t, s)

	q, err := s.engine.CreateQuest(t.Context(), "automation quest", model.QuestTypeExecute, "", orchestrator.CreateQuestOptions{
		WarriorID: "adv_test_warrior",
		MageID:    "adv_test_mage",
	})
	if err != nil {
		t.Fatalf("CreateQuest: %v", err)
	}
	q.CreatedBy = model.QuestSourcePrefix + "my_automation"
	if err := fsstore.NewQuestStore(s.root).SaveQuest(q); err != nil {
		t.Fatalf("SaveQuest: %v", err)
	}

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodGet, "/api/quests/"+q.ID+"/thread", ""))
	body := decodeBody(t, rr)

	actors, _ := body["actors"].([]any)
	hasAutomation := false
	for _, a := range actors {
		m := a.(map[string]any)
		if m["role"] == "automation" && m["source_type"] == "automation" {
			hasAutomation = true
			if m["identity"] != q.CreatedBy {
				t.Fatalf("automation actor identity = %v, want %v", m["identity"], q.CreatedBy)
			}
		}
	}
	if !hasAutomation {
		t.Fatalf("automation-created quest should have automation actor: %+v", actors)
	}
}

func TestGetQuestThread_FanoutRootSummary(t *testing.T) {
	s, h := newTestServer(t)
	configureRunnableAdventurers(t, s)

	parent, err := s.engine.CreateQuest(t.Context(), "fanout root", model.QuestTypeExecute, "", orchestrator.CreateQuestOptions{
		WarriorID: "adv_test_warrior",
		MageID:    "adv_test_mage",
	})
	if err != nil {
		t.Fatalf("CreateQuest parent: %v", err)
	}

	leafID, err := s.engine.SpawnQuest(t.Context(), "leaf task", "", parent.BaseWorkingDir, parent.ID, fsstore.FanoutContract{
		LeafID: "leaf-1",
	})
	if err != nil {
		t.Fatalf("SpawnQuest: %v", err)
	}

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodGet, "/api/quests/"+parent.ID+"/thread", ""))
	body := decodeBody(t, rr)

	fanout, ok := body["fanout_summary"].(map[string]any)
	if !ok {
		t.Fatal("fanout root should have fanout_summary")
	}
	if fanout["is_root"] != true {
		t.Fatalf("is_root = %v, want true", fanout["is_root"])
	}
	if fanout["is_leaf"] != false {
		t.Fatalf("is_leaf = %v, want false", fanout["is_leaf"])
	}
	leaves, _ := fanout["leaves"].([]any)
	if len(leaves) < 1 {
		t.Fatalf("fanout root should list at least 1 leaf: %+v", leaves)
	}
	foundLeaf := false
	for _, l := range leaves {
		m := l.(map[string]any)
		if m["quest_id"] == leafID {
			foundLeaf = true
			if m["status"] == "" {
				t.Fatal("leaf status should not be empty")
			}
		}
	}
	if !foundLeaf {
		t.Fatalf("fanout root leaves should include spawned leaf %s: %+v", leafID, leaves)
	}
}

func TestGetQuestThread_FanoutLeafSummary(t *testing.T) {
	s, h := newTestServer(t)
	configureRunnableAdventurers(t, s)

	parent, err := s.engine.CreateQuest(t.Context(), "fanout root for leaf", model.QuestTypeExecute, "", orchestrator.CreateQuestOptions{
		WarriorID: "adv_test_warrior",
		MageID:    "adv_test_mage",
	})
	if err != nil {
		t.Fatalf("CreateQuest parent: %v", err)
	}

	leafID, err := s.engine.SpawnQuest(t.Context(), "leaf task 2", "", parent.BaseWorkingDir, parent.ID, fsstore.FanoutContract{
		LeafID: "leaf-2",
	})
	if err != nil {
		t.Fatalf("SpawnQuest: %v", err)
	}

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodGet, "/api/quests/"+leafID+"/thread", ""))
	body := decodeBody(t, rr)

	fanout, ok := body["fanout_summary"].(map[string]any)
	if !ok {
		t.Fatal("fanout leaf should have fanout_summary")
	}
	if fanout["is_leaf"] != true {
		t.Fatalf("is_leaf = %v, want true", fanout["is_leaf"])
	}
	if fanout["is_root"] != false {
		t.Fatalf("is_root = %v, want false", fanout["is_root"])
	}
	if fanout["root_quest_id"] != parent.ID {
		t.Fatalf("root_quest_id = %v, want %s", fanout["root_quest_id"], parent.ID)
	}
}

func TestGetQuestThread_ReturnsQuestTimeFields(t *testing.T) {
	s, h := newTestServer(t)
	configureRunnableAdventurers(t, s)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodPost, "/api/quests", `{"query":"time fields test","type":"execute"}`))
	if rr.Code != http.StatusOK {
		t.Fatalf("create quest status = %d", rr.Code)
	}
	createBody := decodeBody(t, rr)
	qid, _ := createBody["qid"].(string)
	if qid == "" {
		t.Fatal("create quest returned empty qid")
	}

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq(http.MethodGet, "/api/quests/"+qid+"/thread", ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /thread status = %d", rr.Code)
	}
	body := decodeBody(t, rr)
	questInfo, ok := body["quest"].(map[string]any)
	if !ok {
		t.Fatal("response should include quest")
	}

	createdAtMs, ok := questInfo["created_at_ms"].(float64)
	if !ok || createdAtMs <= 0 {
		t.Fatalf("quest.created_at_ms = %v (type %T), want positive number", questInfo["created_at_ms"], questInfo["created_at_ms"])
	}
	updatedAtMs, ok := questInfo["updated_at_ms"].(float64)
	if !ok || updatedAtMs <= 0 {
		t.Fatalf("quest.updated_at_ms = %v (type %T), want positive number", questInfo["updated_at_ms"], questInfo["updated_at_ms"])
	}
	if createdAtMs > updatedAtMs {
		t.Fatalf("created_at_ms (%v) > updated_at_ms (%v)", createdAtMs, updatedAtMs)
	}

	// Ensure non-_ms variants are NOT present (contract enforcement).
	if _, exists := questInfo["created_at"]; exists {
		t.Fatal("quest.created_at should NOT be present; use created_at_ms")
	}
	if _, exists := questInfo["updated_at"]; exists {
		t.Fatal("quest.updated_at should NOT be present; use updated_at_ms")
	}
}
