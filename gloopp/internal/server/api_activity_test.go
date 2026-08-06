package server

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
	"code.byted.org/lihuanyu.0w0/gloop/internal/orchestrator"
)

// saveTestQuest 写一份最小可用的 quest meta 到 root，供 activity 关联。
func saveTestQuest(t *testing.T, s *Server, q *fsstore.QuestMeta) {
	t.Helper()
	if q.ID == "" {
		t.Fatalf("quest id required")
	}
	if q.Type == "" {
		q.Type = model.QuestTypeExecute
	}
	if q.Status == "" {
		q.Status = model.QuestStatusSuccess
	}
	if q.CreatedAtMs == 0 {
		q.CreatedAtMs = 1000
	}
	if err := fsstore.NewQuestStore(s.root).SaveQuest(q); err != nil {
		t.Fatalf("SaveQuest(%s): %v", q.ID, err)
	}
}

func appendEv(t *testing.T, s *Server, ts int64, typ, qid, sid string, payload map[string]any) {
	t.Helper()
	raw, _ := json.Marshal(payload)
	ev := &fsstore.GlobalEventRow{
		Timestamp: ts,
		Type:      typ,
		QuestID:   qid,
		SessionID: sid,
		Payload:   json.RawMessage(raw),
	}
	if err := s.root.AppendGlobalEvent(ev); err != nil {
		t.Fatalf("AppendGlobalEvent(%s): %v", typ, err)
	}
}

func activityItems(t *testing.T, rr *httptest.ResponseRecorder) []ActivityItem {
	t.Helper()
	body := decodeBody(t, rr)
	raw, _ := json.Marshal(body["items"])
	var items []ActivityItem
	if err := json.Unmarshal(raw, &items); err != nil {
		t.Fatalf("unmarshal items: %v", err)
	}
	return items
}

func activityProjects(t *testing.T, rr *httptest.ResponseRecorder) []string {
	t.Helper()
	body := decodeBody(t, rr)
	raw, _ := json.Marshal(body["projects"])
	var ps []string
	if err := json.Unmarshal(raw, &ps); err != nil {
		t.Fatalf("unmarshal projects: %v", err)
	}
	return ps
}

func TestGetActivity_Empty(t *testing.T) {
	s, h := newTestServer(t)
	configureRunnableAdventurers(t, s)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq("GET", "/api/activity", ""))

	if rr.Code != 200 {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	items := activityItems(t, rr)
	if len(items) != 0 {
		t.Fatalf("empty feed should have 0 items, got %d", len(items))
	}
}

// TestGetActivity_SingleSource feed 只含 user(post) + agent.post，系统事件全过滤。
func TestGetActivity_SingleSource(t *testing.T) {
	s, h := newTestServer(t)
	configureRunnableAdventurers(t, s)

	saveTestQuest(t, s, &fsstore.QuestMeta{
		ID: "qst_a1", ShortID: "a1", Query: "给 douyin-cli 落地埋点",
		WarriorID: "adv_test_warrior", BaseWorkingDir: "/data00/douyin-cli",
		CreatedBy: "user", Status: model.QuestStatusSuccess,
	})

	// 按时间正序写入：1 条 user + 1 条 agent.post + 3 条系统事件（应被过滤）
	appendEv(t, s, 1000, "quest.created", "qst_a1", "", map[string]any{"query": "给 douyin-cli 落地埋点"})
	appendEv(t, s, 2000, "micro.token_delta", "qst_a1", "warrior_0", map[string]any{"in": 100})                                  // 过滤
	appendEv(t, s, 3000, "micro.assistant_msg", "qst_a1", "warrior_0", map[string]any{"content": "我在分析代码"})                      // 过滤
	appendEv(t, s, 4000, "micro.tool_end", "qst_a1", "warrior_0", map[string]any{"tool": "phase_checkpoint", "summary": "done"}) // 过滤
	appendEv(t, s, 5000, "agent.post", "qst_a1", "warrior_0", map[string]any{
		"post_id": "post_abc", "content": "埋点已落地，KR2 三指标可测", "kind": "milestone",
	})

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq("GET", "/api/activity", ""))

	items := activityItems(t, rr)
	if len(items) != 2 {
		t.Fatalf("expect 2 feed items (user + agent.post), got %d", len(items))
	}
	// 倒序：最新在前。post(ts=5000) → user(ts=1000)
	if items[0].Kind != "post" {
		t.Fatalf("item[0].Kind=%s want post", items[0].Kind)
	}
	if items[1].Kind != "user" {
		t.Fatalf("item[1].Kind=%s want user", items[1].Kind)
	}

	post := items[0]
	if post.Summary != "埋点已落地，KR2 三指标可测" {
		t.Fatalf("post summary=%q", post.Summary)
	}
	if post.PostID != "post_abc" {
		t.Fatalf("post_id=%q want post_abc", post.PostID)
	}
	if post.PostKind != "milestone" {
		t.Fatalf("post_kind=%q want milestone", post.PostKind)
	}
	if post.WarriorName != "test warrior" {
		t.Fatalf("warrior name=%q want 'test warrior'", post.WarriorName)
	}
	if post.Project != "douyin-cli" {
		t.Fatalf("project=%q want douyin-cli", post.Project)
	}

	user := items[1]
	if user.Summary != "给 douyin-cli 落地埋点" {
		t.Fatalf("user summary=%q", user.Summary)
	}
}

// TestGetActivity_ReplyTo 普通 agent.post 旧事件不再顶层展示，避免同 quest 过程帖刷屏。
func TestGetActivity_ReplyTo(t *testing.T) {
	s, h := newTestServer(t)
	configureRunnableAdventurers(t, s)

	saveTestQuest(t, s, &fsstore.QuestMeta{
		ID: "qst_r1", ShortID: "r1", Query: "reply test",
		WarriorID: "adv_test_warrior", BaseWorkingDir: "/data00/gloop",
		CreatedBy: "user", Status: model.QuestStatusRunning,
	})

	appendEv(t, s, 1000, "quest.created", "qst_r1", "", map[string]any{"query": "reply test"})
	appendEv(t, s, 2000, "agent.post", "qst_r1", "warrior_0", map[string]any{
		"post_id": "post_001", "content": "原始帖子", "kind": "post",
	})
	appendEv(t, s, 3000, "agent.post", "qst_r1", "mage_0", map[string]any{
		"post_id": "post_002", "content": "回复帖子", "reply_to": "post_001", "kind": "post",
	})

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq("GET", "/api/activity", ""))

	items := activityItems(t, rr)
	if len(items) != 1 {
		t.Fatalf("ordinary legacy agent posts should be filtered to reduce feed noise, got %d items: %+v", len(items), items)
	}
	if items[0].Kind != "user" || items[0].PostID != fsstore.RootPostID("qst_r1") {
		t.Fatalf("remaining item should be the root quest post: %+v", items[0])
	}
}

func TestGetActivity_ProjectFilter(t *testing.T) {
	s, h := newTestServer(t)
	configureRunnableAdventurers(t, s)

	saveTestQuest(t, s, &fsstore.QuestMeta{ID: "qst_p1", ShortID: "p1", Query: "gloop task", WarriorID: "adv_test_warrior", BaseWorkingDir: "/data00/gloop", CreatedBy: "user"})
	saveTestQuest(t, s, &fsstore.QuestMeta{ID: "qst_p2", ShortID: "p2", Query: "douyin task", WarriorID: "adv_test_warrior", BaseWorkingDir: "/data00/douyin-cli", CreatedBy: "user"})

	appendEv(t, s, 1000, "quest.created", "qst_p1", "", map[string]any{"query": "gloop task"})
	appendEv(t, s, 2000, "quest.created", "qst_p2", "", map[string]any{"query": "douyin task"})

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq("GET", "/api/activity?project=gloop", ""))

	items := activityItems(t, rr)
	if len(items) != 1 {
		t.Fatalf("project filter should leave 1 item, got %d", len(items))
	}
	if items[0].Project != "gloop" {
		t.Fatalf("project=%q want gloop", items[0].Project)
	}
	if items[0].QuestID != "qst_p1" {
		t.Fatalf("quest_id=%q want qst_p1", items[0].QuestID)
	}
}

func TestGetActivity_Limit(t *testing.T) {
	s, h := newTestServer(t)
	configureRunnableAdventurers(t, s)

	saveTestQuest(t, s, &fsstore.QuestMeta{ID: "qst_l1", ShortID: "l1", Query: "q1", WarriorID: "adv_test_warrior", BaseWorkingDir: "/data00/gloop", CreatedBy: "user"})
	for i, ts := range []int64{1000, 2000, 3000, 4000, 5000} {
		appendEv(t, s, ts, "quest.created", "qst_l1", "", map[string]any{"query": "q1", "n": i})
	}

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq("GET", "/api/activity?limit=2", ""))

	items := activityItems(t, rr)
	if len(items) != 2 {
		t.Fatalf("limit=2 should return 2 items, got %d", len(items))
	}
	if items[0].Ts != 5000 || items[1].Ts != 4000 {
		t.Fatalf("limit should return newest 2: got ts %d %d", items[0].Ts, items[1].Ts)
	}
}

// TestGetActivity_NonDynamicFiltered 非 user/post 事件全过滤。
func TestGetActivity_NonDynamicFiltered(t *testing.T) {
	s, h := newTestServer(t)
	configureRunnableAdventurers(t, s)

	saveTestQuest(t, s, &fsstore.QuestMeta{ID: "qst_n1", ShortID: "n1", Query: "q", WarriorID: "adv_test_warrior", BaseWorkingDir: "/data00/gloop", CreatedBy: "user"})
	appendEv(t, s, 1000, "micro.token_delta", "qst_n1", "warrior_0", map[string]any{"in": 10})
	appendEv(t, s, 2000, "micro.command_run", "qst_n1", "warrior_0", map[string]any{"cmd": "ls"})
	appendEv(t, s, 3000, "quest.note", "qst_n1", "", map[string]any{"content": "note"})
	appendEv(t, s, 4000, "micro.tool_end", "qst_n1", "warrior_0", map[string]any{"tool": "bash"})
	appendEv(t, s, 5000, "micro.assistant_msg", "qst_n1", "warrior_0", map[string]any{"content": "thinking"})
	appendEv(t, s, 6000, "quest.review_submitted", "qst_n1", "mage_0", map[string]any{"verdict": "pass"})

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq("GET", "/api/activity", ""))

	items := activityItems(t, rr)
	if len(items) != 0 {
		t.Fatalf("non-dynamic events should be filtered, got %d items", len(items))
	}
}

func TestGetActivity_IncludesAttentionFromQuestMeta(t *testing.T) {
	s, h := newTestServer(t)
	configureRunnableAdventurers(t, s)

	saveTestQuest(t, s, &fsstore.QuestMeta{
		ID:                "qst_attention_blocked",
		ShortID:           "attn",
		Query:             "blocked attention",
		Type:              model.QuestTypeExecute,
		Status:            model.QuestStatusBlocked,
		BaseWorkingDir:    "/data00/gloop",
		CreatedBy:         "user",
		CreatedAtMs:       1000,
		UpdatedAtMs:       9000,
		BlockedReason:     "agent auth expired",
		BlockedReasonCode: "agent_error_auth",
		BlockedCategory:   "configuration",
	})

	// No global event is written for this quest. Attention must come from
	// current quest meta, otherwise old blocked quests disappear from Feed once
	// they fall outside the event window.
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq("GET", "/api/activity", ""))

	items := activityItems(t, rr)
	if len(items) != 1 {
		t.Fatalf("attention-only feed should have 1 item, got %d: %+v", len(items), items)
	}
	item := items[0]
	if item.Kind != "attention" || item.QuestID != "qst_attention_blocked" {
		t.Fatalf("bad attention item identity: %+v", item)
	}
	if item.HumanExceptionID != "hex_qst_attention_blocked" || item.RiskLevel != "high" || item.AttentionPriority != 10 {
		t.Fatalf("attention fields missing: %+v", item)
	}
	if item.RecommendedAction == "" || item.ActionEndpointHint != "resolve-blocked" {
		t.Fatalf("attention action metadata missing: %+v", item)
	}
	if len(item.AvailableActions) != 3 || item.AvailableActions[0] != "continue" {
		t.Fatalf("available actions = %#v, want continue/user-review/cancel", item.AvailableActions)
	}
	if item.BlockedReasonCode != "agent_error_auth" || item.BlockedCategory != "configuration" {
		t.Fatalf("blocked fields missing: %+v", item)
	}

	// Once the quest leaves the human-exception state, the attention item should
	// disappear without mutating old event rows.
	qs := fsstore.NewQuestStore(s.root)
	q, err := qs.LoadQuest("qst_attention_blocked")
	if err != nil {
		t.Fatalf("LoadQuest: %v", err)
	}
	q.Status = model.QuestStatusSuccess
	q.UpdatedAtMs = 10000
	if err := qs.SaveQuest(q); err != nil {
		t.Fatalf("SaveQuest resolved: %v", err)
	}

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq("GET", "/api/activity", ""))
	items = activityItems(t, rr)
	for _, item := range items {
		if item.Kind == "attention" && item.QuestID == "qst_attention_blocked" {
			t.Fatalf("resolved attention should disappear, got %+v", items)
		}
	}
}

func TestGetActivity_ProjectsList(t *testing.T) {
	s, h := newTestServer(t)
	configureRunnableAdventurers(t, s)

	saveTestQuest(t, s, &fsstore.QuestMeta{ID: "qst_pr1", ShortID: "pr1", Query: "a", WarriorID: "adv_test_warrior", BaseWorkingDir: "/data00/gloop", CreatedBy: "user"})
	saveTestQuest(t, s, &fsstore.QuestMeta{ID: "qst_pr2", ShortID: "pr2", Query: "b", WarriorID: "adv_test_warrior", BaseWorkingDir: "/data00/douyin-cli", CreatedBy: "user"})

	appendEv(t, s, 1000, "quest.created", "qst_pr1", "", nil)
	appendEv(t, s, 2000, "quest.created", "qst_pr2", "", nil)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq("GET", "/api/activity", ""))

	ps := activityProjects(t, rr)
	want := map[string]bool{"gloop": true, "douyin-cli": true}
	got := map[string]bool{}
	for _, p := range ps {
		got[p] = true
	}
	for w := range want {
		if !got[w] {
			t.Fatalf("projects missing %q, got %v", w, ps)
		}
	}
}

func TestGetActivity_MergesThreadPostsAndLegacyEvents(t *testing.T) {
	s, h := newTestServer(t)
	configureRunnableAdventurers(t, s)

	saveTestQuest(t, s, &fsstore.QuestMeta{
		ID: "qst_new", ShortID: "new", Query: "new thread quest",
		WarriorID: "adv_test_warrior", MageID: "adv_test_mage",
		BaseWorkingDir: "/data00/gloop", CreatedBy: "user", CreatedAtMs: 1000,
	})
	saveTestQuest(t, s, &fsstore.QuestMeta{
		ID: "qst_old", ShortID: "old", Query: "old event quest",
		WarriorID: "adv_test_warrior", BaseWorkingDir: "/data00/douyin-cli",
		CreatedBy: "user", CreatedAtMs: 2000,
	})
	qs := fsstore.NewQuestStore(s.root)
	newQuest, err := qs.LoadQuest("qst_new")
	if err != nil {
		t.Fatalf("LoadQuest new: %v", err)
	}
	if _, err := qs.EnsureRootThreadPost(newQuest); err != nil {
		t.Fatalf("EnsureRootThreadPost failed: %v", err)
	}
	if _, err := qs.AppendThreadPost("qst_new", fsstore.AppendThreadPostOptions{
		PostID:         "post_new_reply",
		ThreadID:       "qst_new",
		RootPostID:     fsstore.RootPostID("qst_new"),
		ParentReplyID:  fsstore.RootPostID("qst_new"),
		CausalRefs:     []string{fsstore.RootPostID("qst_new")},
		AuthorIdentity: "adv_test_mage",
		AuthorRole:     model.PostRoleChecker,
		Kind:           "post",
		Content:        "checker reply",
		CreatedAtMs:    3000,
	}); err != nil {
		t.Fatalf("AppendThreadPost failed: %v", err)
	}
	appendEv(t, s, 1000, "quest.created", "qst_new", "", map[string]any{"query": "new thread quest"})
	appendEv(t, s, 2000, "quest.created", "qst_old", "", map[string]any{"query": "old event quest"})

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq("GET", "/api/activity", ""))

	items := activityItems(t, rr)
	if len(items) != 2 {
		t.Fatalf("items = %d, want new root with reply preview + old root: %+v", len(items), items)
	}
	ids := map[string]int{}
	for _, item := range items {
		ids[item.QuestID]++
	}
	if ids["qst_new"] != 1 || ids["qst_old"] != 1 {
		t.Fatalf("bad merged feed counts: %+v items=%+v", ids, items)
	}
	var newItem ActivityItem
	for _, item := range items {
		if item.QuestID == "qst_new" {
			newItem = item
		}
	}
	if newItem.PostID != fsstore.RootPostID("qst_new") || newItem.ReplyCount != 1 || len(newItem.ReplyPreviews) != 1 {
		t.Fatalf("new quest should use root item with reply preview: %+v", newItem)
	}
	if newItem.ReplyPreviews[0].PostID != "post_new_reply" || newItem.ReplyPreviews[0].AuthorRole != string(model.PostRoleChecker) {
		t.Fatalf("reply preview should use thread ledger metadata: %+v", newItem.ReplyPreviews)
	}
}

func TestGetActivity_RootItemIncludesReplyPreviews(t *testing.T) {
	s, h := newTestServer(t)
	configureRunnableAdventurers(t, s)

	saveTestQuest(t, s, &fsstore.QuestMeta{
		ID: "qst_thread_preview", ShortID: "preview", Query: "thread preview quest",
		WarriorID: "adv_test_warrior", MageID: "adv_test_mage",
		BaseWorkingDir: "/data00/gloop", CreatedBy: "user", CreatedAtMs: 1000,
	})
	qs := fsstore.NewQuestStore(s.root)
	q, err := qs.LoadQuest("qst_thread_preview")
	if err != nil {
		t.Fatalf("LoadQuest: %v", err)
	}
	if _, err := qs.EnsureRootThreadPost(q); err != nil {
		t.Fatalf("EnsureRootThreadPost failed: %v", err)
	}
	for _, spec := range []struct {
		id   string
		role model.PostAuthorRole
		kind string
		text string
		ts   int64
	}{
		{"post_maker", model.PostRoleMaker, "maker_report", "maker delivered", 2000},
		{"post_checker", model.PostRoleChecker, "review_report", "checker passed", 3000},
		{"post_system", model.PostRoleSystem, "decision_note", "Outcome: success", 4000},
	} {
		if _, err := qs.AppendThreadPost(q.ID, fsstore.AppendThreadPostOptions{
			PostID:         spec.id,
			ThreadID:       q.ID,
			RootPostID:     fsstore.RootPostID(q.ID),
			ParentReplyID:  fsstore.RootPostID(q.ID),
			CausalRefs:     []string{fsstore.RootPostID(q.ID)},
			AuthorIdentity: string(spec.role),
			AuthorRole:     spec.role,
			Kind:           spec.kind,
			Content:        spec.text,
			CreatedAtMs:    spec.ts,
		}); err != nil {
			t.Fatalf("AppendThreadPost(%s) failed: %v", spec.id, err)
		}
	}

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq("GET", "/api/activity", ""))

	items := activityItems(t, rr)
	if len(items) != 1 {
		t.Fatalf("thread ledger activity should return one root item, got %d: %+v", len(items), items)
	}
	item := items[0]
	if item.Kind != "user" || item.PostID != fsstore.RootPostID(q.ID) {
		t.Fatalf("activity item should be root post: %+v", item)
	}
	if item.ReplyCount != 3 || len(item.ReplyPreviews) != 3 {
		t.Fatalf("reply preview count mismatch: %+v", item)
	}
	if item.ReplyPreviews[0].PostID != "post_system" || item.ReplyPreviews[0].AuthorRole != string(model.PostRoleSystem) {
		t.Fatalf("newest reply should be first preview: %+v", item.ReplyPreviews)
	}
	if item.ReplyPreviews[2].PostID != "post_maker" {
		t.Fatalf("preview should preserve latest-three order: %+v", item.ReplyPreviews)
	}
}

func TestGetActivity_DedupesPublishPostReplyEvent(t *testing.T) {
	s, h := newTestServer(t)
	configureRunnableAdventurers(t, s)

	q, err := s.engine.CreateQuest(t.Context(), "publish post dedupe", model.QuestTypeExecute, "", orchestrator.CreateQuestOptions{
		WarriorID:    "adv_test_warrior",
		MageID:       "adv_test_mage",
		WorkflowMode: model.WorkflowModeChecked,
	})
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	postID, err := s.engine.PublishPost(q.ID, "mage_0", "published checker reply", fsstore.RootPostID(q.ID), "review_report")
	if err != nil {
		t.Fatalf("PublishPost failed: %v", err)
	}

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq("GET", "/api/activity", ""))

	items := activityItems(t, rr)
	if len(items) != 1 {
		t.Fatalf("published reply should not appear as duplicate activity item: %+v", items)
	}
	root := items[0]
	if root.PostID != fsstore.RootPostID(q.ID) || root.ReplyCount != 1 || len(root.ReplyPreviews) != 1 {
		t.Fatalf("root should carry published reply preview: %+v", root)
	}
	if root.ReplyPreviews[0].PostID != postID {
		t.Fatalf("reply preview post id = %q, want %q", root.ReplyPreviews[0].PostID, postID)
	}
}

func TestGetActivity_FoldsOrdinaryTopLevelPostIntoRootPreview(t *testing.T) {
	s, h := newTestServer(t)
	configureRunnableAdventurers(t, s)

	q, err := s.engine.CreateQuest(t.Context(), "ordinary post fold", model.QuestTypeExecute, "", orchestrator.CreateQuestOptions{
		WarriorID:    "adv_test_warrior",
		MageID:       "adv_test_mage",
		WorkflowMode: model.WorkflowModeChecked,
	})
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	qs := fsstore.NewQuestStore(s.root)
	if _, err := qs.AppendThreadPost(q.ID, fsstore.AppendThreadPostOptions{
		PostID:      "post_ordinary_top",
		ThreadID:    q.ID,
		RootPostID:  fsstore.RootPostID(q.ID),
		AuthorRole:  model.PostRoleMaker,
		Kind:        "post",
		Content:     "ordinary process update",
		CreatedAtMs: 3000,
	}); err != nil {
		t.Fatalf("AppendThreadPost failed: %v", err)
	}

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq("GET", "/api/activity", ""))

	items := activityItems(t, rr)
	if len(items) != 1 {
		t.Fatalf("ordinary top-level post should fold into root preview, got %+v", items)
	}
	root := items[0]
	if root.PostID != fsstore.RootPostID(q.ID) || root.ReplyCount != 1 || len(root.ReplyPreviews) != 1 {
		t.Fatalf("root should carry ordinary post preview: %+v", root)
	}
	if root.ReplyPreviews[0].PostID != "post_ordinary_top" || root.ReplyPreviews[0].ReplyTo != fsstore.RootPostID(q.ID) {
		t.Fatalf("ordinary post preview mismatch: %+v", root.ReplyPreviews[0])
	}
}

func TestGetActivity_KeepsPublishPostTopLevelEvent(t *testing.T) {
	s, h := newTestServer(t)
	configureRunnableAdventurers(t, s)

	q, err := s.engine.CreateQuest(t.Context(), "top level post", model.QuestTypeExecute, "", orchestrator.CreateQuestOptions{
		WarriorID:    "adv_test_warrior",
		MageID:       "adv_test_mage",
		WorkflowMode: model.WorkflowModeChecked,
	})
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	postID, err := s.engine.PublishPost(q.ID, "warrior_0", "top level explicit post", "", "milestone")
	if err != nil {
		t.Fatalf("PublishPost failed: %v", err)
	}

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq("GET", "/api/activity", ""))

	items := activityItems(t, rr)
	if len(items) != 2 {
		t.Fatalf("top-level post should remain independent alongside root: %+v", items)
	}
	var top ActivityItem
	var root ActivityItem
	for _, item := range items {
		if item.PostID == postID {
			top = item
		}
		if item.PostID == fsstore.RootPostID(q.ID) {
			root = item
		}
	}
	if top.PostID != postID || top.Kind != "post" || top.ReplyTo != "" {
		t.Fatalf("top-level post item missing: %+v", items)
	}
	if root.PostID != fsstore.RootPostID(q.ID) || root.ReplyCount != 0 {
		t.Fatalf("root should not preview top-level post: %+v", root)
	}
}

func TestGetActivity_KeepsDecisionNoteTopLevel(t *testing.T) {
	s, h := newTestServer(t)
	configureRunnableAdventurers(t, s)

	q, err := s.engine.CreateQuest(t.Context(), "decision note top level", model.QuestTypeExecute, "", orchestrator.CreateQuestOptions{
		WarriorID:    "adv_test_warrior",
		MageID:       "adv_test_mage",
		WorkflowMode: model.WorkflowModeChecked,
	})
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	qs := fsstore.NewQuestStore(s.root)
	if _, err := qs.AppendThreadPost(q.ID, fsstore.AppendThreadPostOptions{
		PostID:      "post_decision_note",
		ThreadID:    q.ID,
		RootPostID:  fsstore.RootPostID(q.ID),
		AuthorRole:  model.PostRoleSystem,
		Kind:        "decision_note",
		Content:     "Outcome: pass",
		CreatedAtMs: 4000,
	}); err != nil {
		t.Fatalf("AppendThreadPost failed: %v", err)
	}

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq("GET", "/api/activity", ""))

	items := activityItems(t, rr)
	if len(items) != 2 {
		t.Fatalf("decision_note should remain top-level with root: %+v", items)
	}
	var decision ActivityItem
	for _, item := range items {
		if item.PostID == "post_decision_note" {
			decision = item
		}
	}
	if decision.PostID != "post_decision_note" || decision.PostKind != "decision_note" {
		t.Fatalf("decision_note top-level item missing: %+v", items)
	}
}

func TestGetActivity_AutomationRunAppearsAsReplyPreview(t *testing.T) {
	s, h := newTestServer(t)
	configureRunnableAdventurers(t, s)

	cfg := &fsstore.AutomationConfig{
		ID:         "auto_feed_actor",
		Name:       "Feed Actor",
		Query:      "automation feed actor",
		Trigger:    fsstore.TriggerManual,
		Enabled:    true,
		AutoStart:  false,
		TriageMode: "candidate",
	}
	if err := s.engine.SaveAutomation(cfg); err != nil {
		t.Fatalf("SaveAutomation failed: %v", err)
	}
	qid, err := s.engine.RunAutomation(t.Context(), cfg.ID)
	if err != nil {
		t.Fatalf("RunAutomation failed: %v", err)
	}

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq("GET", "/api/activity", ""))

	items := activityItems(t, rr)
	if len(items) != 1 {
		t.Fatalf("automation run should render as root item with preview, got %+v", items)
	}
	root := items[0]
	if root.PostID != fsstore.RootPostID(qid) || root.AuthorRole != string(model.PostRoleAutomation) {
		t.Fatalf("automation root should come from real quest source: %+v", root)
	}
	if root.ReplyCount != 1 || len(root.ReplyPreviews) != 1 {
		t.Fatalf("automation run should be a single reply preview: %+v", root)
	}
	preview := root.ReplyPreviews[0]
	if preview.AuthorRole != string(model.PostRoleAutomation) || preview.PostKind != "automation_run" {
		t.Fatalf("automation run preview should preserve automation role/kind: %+v", preview)
	}
	if !strings.Contains(preview.Summary, "Feed Actor") || !strings.Contains(preview.Summary, "candidate") {
		t.Fatalf("automation run preview summary should expose real run context: %+v", preview)
	}
}

func TestGetActivity_ReplyPreviewSummaryIsTruncated(t *testing.T) {
	s, h := newTestServer(t)
	configureRunnableAdventurers(t, s)

	saveTestQuest(t, s, &fsstore.QuestMeta{
		ID: "qst_long_preview", ShortID: "longpreview", Query: "long preview quest",
		WarriorID: "adv_test_warrior", MageID: "adv_test_mage",
		BaseWorkingDir: "/data00/gloop", CreatedBy: "user", CreatedAtMs: 1000,
	})
	qs := fsstore.NewQuestStore(s.root)
	q, err := qs.LoadQuest("qst_long_preview")
	if err != nil {
		t.Fatalf("LoadQuest: %v", err)
	}
	if _, err := qs.EnsureRootThreadPost(q); err != nil {
		t.Fatalf("EnsureRootThreadPost failed: %v", err)
	}
	long := strings.Repeat("长", 260)
	if _, err := qs.AppendThreadPost(q.ID, fsstore.AppendThreadPostOptions{
		PostID:        "post_long",
		ThreadID:      q.ID,
		RootPostID:    fsstore.RootPostID(q.ID),
		ParentReplyID: fsstore.RootPostID(q.ID),
		AuthorRole:    model.PostRoleChecker,
		Kind:          "review_report",
		Content:       long,
		CreatedAtMs:   2000,
	}); err != nil {
		t.Fatalf("AppendThreadPost failed: %v", err)
	}

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq("GET", "/api/activity", ""))

	items := activityItems(t, rr)
	if len(items) != 1 || len(items[0].ReplyPreviews) != 1 {
		t.Fatalf("bad long preview activity: %+v", items)
	}
	got := []rune(items[0].ReplyPreviews[0].Summary)
	if len(got) != 201 || got[len(got)-1] != '…' {
		t.Fatalf("preview should be 200 runes plus ellipsis, len=%d tail=%q", len(got), string(got[len(got)-1]))
	}
}

func TestGetActivity_FoldsFanoutRootPosts(t *testing.T) {
	s, h := newTestServer(t)
	configureRunnableAdventurers(t, s)

	saveTestQuest(t, s, &fsstore.QuestMeta{
		ID: "qst_leaf_a", ShortID: "leaf_a", Query: "leaf a",
		WarriorID: "adv_test_warrior", MageID: "adv_test_mage",
		BaseWorkingDir: "/data00/gloop", CreatedBy: "user", CreatedAtMs: 1000,
		GroupID: "grp_feed_fold", FanoutLeafID: "leaf-a",
		OwnershipScopes: []string{"internal/server"},
		MergeStrategy:   fsstore.MergeStrategySingleLeaf, MergeOwnerLeafID: "leaf-a",
	})
	saveTestQuest(t, s, &fsstore.QuestMeta{
		ID: "qst_leaf_b", ShortID: "leaf_b", Query: "leaf b",
		WarriorID: "adv_test_warrior", MageID: "adv_test_mage",
		BaseWorkingDir: "/data00/gloop", CreatedBy: "user", CreatedAtMs: 2000,
		GroupID: "grp_feed_fold", FanoutLeafID: "leaf-b",
		OwnershipScopes: []string{"web/src"},
		MergeStrategy:   fsstore.MergeStrategySingleLeaf, MergeOwnerLeafID: "leaf-a",
	})

	qs := fsstore.NewQuestStore(s.root)
	for _, qid := range []string{"qst_leaf_a", "qst_leaf_b"} {
		q, err := qs.LoadQuest(qid)
		if err != nil {
			t.Fatalf("LoadQuest(%s): %v", qid, err)
		}
		if _, err := qs.EnsureRootThreadPost(q); err != nil {
			t.Fatalf("EnsureRootThreadPost(%s): %v", qid, err)
		}
	}
	if _, err := qs.AppendThreadPost("qst_leaf_b", fsstore.AppendThreadPostOptions{
		PostID:         "post_leaf_b_reply",
		ThreadID:       "qst_leaf_b",
		RootPostID:     fsstore.RootPostID("qst_leaf_b"),
		ParentReplyID:  fsstore.RootPostID("qst_leaf_b"),
		CausalRefs:     []string{fsstore.RootPostID("qst_leaf_b")},
		AuthorIdentity: "adv_test_mage",
		AuthorRole:     model.PostRoleChecker,
		Kind:           "review_report",
		Content:        "leaf b checked",
		CreatedAtMs:    3000,
	}); err != nil {
		t.Fatalf("AppendThreadPost failed: %v", err)
	}

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq("GET", "/api/activity", ""))

	items := activityItems(t, rr)
	if len(items) != 1 {
		t.Fatalf("items = %d, want one fanout fold root with reply preview: %+v", len(items), items)
	}
	fold := items[0]
	if fold.Kind != "user" || !fold.FanoutFold {
		t.Fatalf("root posts should collapse into fanout fold: %+v", fold)
	}
	if fold.GroupID != "grp_feed_fold" || fold.FanoutLeafCount != 2 {
		t.Fatalf("bad fanout fold metadata: %+v", fold)
	}
	if len(fold.FanoutLeafIDs) != 2 || fold.FanoutLeafIDs[0] != "leaf-a" || fold.FanoutLeafIDs[1] != "leaf-b" {
		t.Fatalf("fanout leaf ids should be real and sorted: %+v", fold.FanoutLeafIDs)
	}
	if len(fold.FanoutQuestIDs) != 2 || fold.FanoutQuestIDs[0] != "qst_leaf_a" || fold.FanoutQuestIDs[1] != "qst_leaf_b" {
		t.Fatalf("fanout quest ids should be real and sorted by leaf id: %+v", fold.FanoutQuestIDs)
	}
	if fold.ReplyCount != 1 || len(fold.ReplyPreviews) != 1 || fold.ReplyPreviews[0].PostID != "post_leaf_b_reply" {
		t.Fatalf("fold should preserve child reply preview: %+v", fold)
	}
}

func TestGetActivity_FanoutFoldReplyCountAcrossLeaves(t *testing.T) {
	s, h := newTestServer(t)
	configureRunnableAdventurers(t, s)

	saveTestQuest(t, s, &fsstore.QuestMeta{
		ID: "qst_fold_a", ShortID: "folda", Query: "fold leaf a",
		WarriorID: "adv_test_warrior", MageID: "adv_test_mage",
		BaseWorkingDir: "/data00/gloop", CreatedBy: "user", CreatedAtMs: 1000,
		GroupID: "grp_fold_multi", FanoutLeafID: "leaf-a",
		OwnershipScopes: []string{"a"},
		MergeStrategy:   fsstore.MergeStrategySingleLeaf, MergeOwnerLeafID: "leaf-a",
	})
	saveTestQuest(t, s, &fsstore.QuestMeta{
		ID: "qst_fold_b", ShortID: "foldb", Query: "fold leaf b",
		WarriorID: "adv_test_warrior", MageID: "adv_test_mage",
		BaseWorkingDir: "/data00/gloop", CreatedBy: "user", CreatedAtMs: 1100,
		GroupID: "grp_fold_multi", FanoutLeafID: "leaf-b",
		OwnershipScopes: []string{"b"},
		MergeStrategy:   fsstore.MergeStrategySingleLeaf, MergeOwnerLeafID: "leaf-a",
	})
	qs := fsstore.NewQuestStore(s.root)
	for _, spec := range []struct {
		qid  string
		post string
		text string
		ts   int64
	}{
		{"qst_fold_a", "post_a", "reply a", 2000},
		{"qst_fold_b", "post_b", "reply b", 3000},
	} {
		q, err := qs.LoadQuest(spec.qid)
		if err != nil {
			t.Fatalf("LoadQuest(%s): %v", spec.qid, err)
		}
		if _, err := qs.EnsureRootThreadPost(q); err != nil {
			t.Fatalf("EnsureRootThreadPost(%s): %v", spec.qid, err)
		}
		if _, err := qs.AppendThreadPost(spec.qid, fsstore.AppendThreadPostOptions{
			PostID:        spec.post,
			ThreadID:      spec.qid,
			RootPostID:    fsstore.RootPostID(spec.qid),
			ParentReplyID: fsstore.RootPostID(spec.qid),
			AuthorRole:    model.PostRoleChecker,
			Kind:          "review_report",
			Content:       spec.text,
			CreatedAtMs:   spec.ts,
		}); err != nil {
			t.Fatalf("AppendThreadPost(%s): %v", spec.post, err)
		}
	}

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq("GET", "/api/activity", ""))

	items := activityItems(t, rr)
	if len(items) != 1 || !items[0].FanoutFold {
		t.Fatalf("expected single fanout fold item: %+v", items)
	}
	if items[0].ReplyCount != 2 || len(items[0].ReplyPreviews) != 2 {
		t.Fatalf("fold reply_count should be group-wide reply total: %+v", items[0])
	}
	if items[0].ReplyPreviews[0].PostID != "post_b" || items[0].ReplyPreviews[1].PostID != "post_a" {
		t.Fatalf("fold previews should be newest replies across leaves: %+v", items[0].ReplyPreviews)
	}
}

func TestGetActivity_SourceEventID_Passthrough(t *testing.T) {
	s, h := newTestServer(t)
	configureRunnableAdventurers(t, s)

	q, err := s.engine.CreateQuest(t.Context(), "source event id feed test", model.QuestTypeExecute, "", orchestrator.CreateQuestOptions{
		WarriorID:    "adv_test_warrior",
		MageID:       "adv_test_mage",
		WorkflowMode: model.WorkflowModeChecked,
	})
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq("GET", "/api/activity", ""))

	items := activityItems(t, rr)
	if len(items) != 1 {
		t.Fatalf("expected 1 activity item, got %d: %+v", len(items), items)
	}
	root := items[0]
	if root.Kind != "user" {
		t.Fatalf("root item kind=%q, want user", root.Kind)
	}
	if root.SourceEventID == 0 {
		t.Fatalf("root item source_event_id should be non-zero (quest.created event id)")
	}
	if root.PostID != fsstore.RootPostID(q.ID) {
		t.Fatalf("root item post_id=%q, want %q", root.PostID, fsstore.RootPostID(q.ID))
	}
}

func TestGetActivity_SourceEventID_WorkflowUpgrade(t *testing.T) {
	s, h := newTestServer(t)
	configureRunnableAdventurers(t, s)

	q, err := s.engine.CreateQuest(t.Context(), "workflow upgrade feed test", model.QuestTypeExecute, "", orchestrator.CreateQuestOptions{
		WarriorID:    "adv_test_warrior",
		MageID:       "adv_test_mage",
		WorkflowMode: model.WorkflowModeDirect,
		Intensity:    model.QuestIntensityQuick,
	})
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	if _, err := s.engine.UpgradeWorkflowMode(t.Context(), q.ID, model.WorkflowModeChecked, "test escalation"); err != nil {
		t.Fatalf("UpgradeWorkflowMode failed: %v", err)
	}

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq("GET", "/api/activity", ""))

	items := activityItems(t, rr)
	if len(items) != 1 {
		t.Fatalf("expected 1 activity item, got %d: %+v", len(items), items)
	}
	root := items[0]
	if root.SourceEventID == 0 {
		t.Fatalf("root item source_event_id should be non-zero")
	}
	if root.ReplyCount != 1 || len(root.ReplyPreviews) != 1 {
		t.Fatalf("expected 1 reply preview for workflow upgrade, got reply_count=%d previews=%+v", root.ReplyCount, root.ReplyPreviews)
	}
	preview := root.ReplyPreviews[0]
	if preview.PostKind != "system_workflow_upgrade" {
		t.Fatalf("reply preview post_kind=%q, want system_workflow_upgrade", preview.PostKind)
	}
	if preview.SourceEventID == 0 {
		t.Fatalf("workflow upgrade reply preview source_event_id should be non-zero (quest.workflow_upgraded event id)")
	}
}

func TestGetActivity_AttentionEvidencePostID(t *testing.T) {
	s, h := newTestServer(t)
	configureRunnableAdventurers(t, s)
	qs := fsstore.NewQuestStore(s.root)

	// Blocked quest with system_escalation projection post.
	saveTestQuest(t, s, &fsstore.QuestMeta{
		ID:              "qst_attn_blocked",
		ShortID:         "ablk",
		Query:           "blocked with evidence",
		Type:            model.QuestTypeExecute,
		Status:          model.QuestStatusBlocked,
		BaseWorkingDir:  "/data00/gloop",
		CreatedBy:       "user",
		CreatedAtMs:     1000,
		UpdatedAtMs:     9000,
		BlockedReason:   "token expired",
		BlockedReasonCode: "agent_error_auth",
		BlockedCategory: "configuration",
	})
	blockedQ, _ := qs.LoadQuest("qst_attn_blocked")
	qs.EnsureRootThreadPost(blockedQ)
	qs.AppendThreadPost("qst_attn_blocked", fsstore.AppendThreadPostOptions{
		PostID:         "sys_blocked_qst_attn_blocked_9000",
		ThreadID:       "qst_attn_blocked",
		RootPostID:     fsstore.RootPostID("qst_attn_blocked"),
		ParentReplyID:  fsstore.RootPostID("qst_attn_blocked"),
		AuthorIdentity: "system",
		AuthorRole:     model.PostRoleSystem,
		Kind:           "system_escalation",
		Content:        "Escalation: blocked. token expired",
		CreatedAtMs:    9000,
	})

	// User review quest with review_report post.
	saveTestQuest(t, s, &fsstore.QuestMeta{
		ID:             "qst_attn_review",
		ShortID:        "arev",
		Query:          "review with evidence",
		Type:           model.QuestTypeExecute,
		Status:         model.QuestStatusUserReview,
		BaseWorkingDir: "/data00/gloop",
		CreatedBy:      "user",
		CreatedAtMs:    2000,
		UpdatedAtMs:    8000,
	})
	reviewQ, _ := qs.LoadQuest("qst_attn_review")
	qs.EnsureRootThreadPost(reviewQ)
	qs.AppendThreadPost("qst_attn_review", fsstore.AppendThreadPostOptions{
		PostID:         "review_report_qst_attn_review",
		ThreadID:       "qst_attn_review",
		RootPostID:     fsstore.RootPostID("qst_attn_review"),
		ParentReplyID:  fsstore.RootPostID("qst_attn_review"),
		AuthorIdentity: "adv_test_checker",
		AuthorRole:     model.PostRoleChecker,
		Kind:           "review_report",
		Content:        "Review: needs fixes",
		CreatedAtMs:    8000,
	})

	// Quest in human-exception state but without any projection post (legacy).
	saveTestQuest(t, s, &fsstore.QuestMeta{
		ID:             "qst_attn_nopost",
		ShortID:        "anop",
		Query:          "blocked without post",
		Type:           model.QuestTypeExecute,
		Status:         model.QuestStatusBlocked,
		BaseWorkingDir: "/data00/gloop",
		CreatedBy:      "user",
		CreatedAtMs:    3000,
		UpdatedAtMs:    7000,
		BlockedReason:  "stuck",
	})

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authedReq("GET", "/api/activity", ""))
	items := activityItems(t, rr)

	found := map[string]ActivityItem{}
	for _, item := range items {
		if item.Kind == "attention" {
			found[item.QuestID] = item
		}
	}

	// Blocked quest should reference its system_escalation post.
	if blk, ok := found["qst_attn_blocked"]; !ok {
		t.Fatalf("missing blocked attention item")
	} else if blk.EvidencePostID != "sys_blocked_qst_attn_blocked_9000" {
		t.Fatalf("blocked evidence_post_id=%q, want sys_blocked_qst_attn_blocked_9000", blk.EvidencePostID)
	}

	// Review quest should reference its review_report post.
	if rev, ok := found["qst_attn_review"]; !ok {
		t.Fatalf("missing review attention item")
	} else if rev.EvidencePostID != "review_report_qst_attn_review" {
		t.Fatalf("review evidence_post_id=%q, want review_report_qst_attn_review", rev.EvidencePostID)
	}

	// Legacy quest without projection should have empty evidence_post_id.
	if nop, ok := found["qst_attn_nopost"]; !ok {
		t.Fatalf("missing no-post attention item")
	} else if nop.EvidencePostID != "" {
		t.Fatalf("no-post evidence_post_id=%q, want empty", nop.EvidencePostID)
	}
}

func TestProjectName(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"/data00/home/lihuanyu.0w0/agent-workspace/gloop", "gloop"},
		{"/data00/.../agent-workspace/douyin-cli", "douyin-cli"},
		{"/data00/.../agent-workspace/gloop/work", "(未分类)"},
		{"", "(未分类)"},
		{"/tmp/", "tmp"},
	}
	for _, c := range cases {
		if got := projectName(c.in); got != c.want {
			t.Fatalf("projectName(%q)=%q want %q", c.in, got, c.want)
		}
	}
}

func TestParseActivityLimit(t *testing.T) {
	cases := []struct {
		q    string
		want int
	}{
		{"", 50},
		{"limit=10", 10},
		{"limit=0", 50},
		{"limit=-1", 50},
		{"limit=999", 200},
		{"limit=abc", 50},
	}
	for _, c := range cases {
		req := authedReq("GET", "/api/activity?"+c.q, "")
		if got := parseActivityLimit(req, 50); got != c.want {
			t.Fatalf("parseActivityLimit(%q)=%d want %d", c.q, got, c.want)
		}
	}
}
