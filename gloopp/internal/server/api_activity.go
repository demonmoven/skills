package server

import (
	"fmt"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

// ==================== GET /api/activity — 首页 Feed ====================
//
// Feed 收三种事件：
//   - quest.created  用户发委托 = 用户发帖，query 作正文
//   - agent.post     agent 主动发帖（gloop post CLI），content 作正文
//   - quest.success  委托完成 = agent 的 phase done summary 作为兜底
//
// 不进 feed 的：assistant_msg（agent 内部推理，非主动发言）、phase_checkpoint /
// review 信号（阶段状态机变更，沉到 quest 详情页）、其他系统事件。

// ActivityItem 一条 Feed 动态，对应前端 FeedCard。
type ActivityItem struct {
	Ts            int64                `json:"ts"`
	Kind          string               `json:"kind"` // user / post / summary / attention
	QuestID       string               `json:"quest_id"`
	ThreadID      string               `json:"thread_id,omitempty"`
	RootPostID    string               `json:"root_post_id,omitempty"`
	ShortID       string               `json:"short_id"`
	Query         string               `json:"query"`
	Project       string               `json:"project"`
	WarriorID     string               `json:"warrior_id,omitempty"`
	WarriorName   string               `json:"warrior_name,omitempty"`
	WarriorLevel  int                  `json:"warrior_level,omitempty"`
	WarriorTitle  string               `json:"warrior_title,omitempty"`
	WarriorClass  string               `json:"warrior_class,omitempty"`
	WarriorExp    int64                `json:"warrior_exp,omitempty"`
	Status        string               `json:"status"`
	Summary       string               `json:"summary,omitempty"`
	Source        string               `json:"source,omitempty"`          // user / automation:xxx / parent spawn
	PostID        string               `json:"post_id,omitempty"`         // agent.post 唯一 ID
	ReplyTo       string               `json:"reply_to,omitempty"`        // 回复的目标 post_id
	AuthorRole    string               `json:"author_role,omitempty"`     // maker / checker / human / system / automation
	CausalRefs    []string             `json:"causal_refs,omitempty"`     // 为什么触发该 reply
	PostKind      string               `json:"post_kind,omitempty"`       // post / milestone / blocker
	SourceEventID int64                `json:"source_event_id,omitempty"` // v0.5: 事件来源标识（先用 existing event row id 过渡）
	Impact        *model.ImpactSummary `json:"impact,omitempty"`
	// 委托帖上下文（kind=user）
	QuestType      string `json:"quest_type,omitempty"`
	WarriorAdvID   string `json:"warrior_adv_id,omitempty"`
	MageAdvID      string `json:"mage_adv_id,omitempty"`
	WarriorAdvName string `json:"warrior_adv_name,omitempty"`
	MageAdvName    string `json:"mage_adv_name,omitempty"`
	ParentQuestID  string `json:"parent_quest_id,omitempty"`
	ParentShortID  string `json:"parent_short_id,omitempty"`
	ChildQuestID   string `json:"child_quest_id,omitempty"`
	ChildShortID   string `json:"child_short_id,omitempty"`
	// fanout 折叠入口（P1）：只用于同 group 的 root posts，完整树仍在 Detail。
	GroupID          string                 `json:"group_id,omitempty"`
	FanoutFold       bool                   `json:"fanout_fold,omitempty"`
	FanoutLeafCount  int                    `json:"fanout_leaf_count,omitempty"`
	FanoutLeafIDs    []string               `json:"fanout_leaf_ids,omitempty"`
	FanoutQuestIDs   []string               `json:"fanout_quest_ids,omitempty"`
	MergeOwnerLeafID string                 `json:"merge_owner_leaf_id,omitempty"`
	ReplyCount       int                    `json:"reply_count,omitempty"`
	ReplyPreviews    []ActivityReplyPreview `json:"reply_previews,omitempty"`
	TriageMode       string                 `json:"triage_mode,omitempty"` // v0.5.4: candidate quest 在 Feed 暴露 triage_mode，供前端确认入口
	// attention items project HumanExceptionFromQuest into Feed.
	HumanExceptionID   string   `json:"human_exception_id,omitempty"`
	AvailableActions   []string `json:"available_actions,omitempty"`
	RecommendedAction  string   `json:"recommended_action,omitempty"`
	RiskLevel          string   `json:"risk_level,omitempty"`
	AttentionPriority  int      `json:"attention_priority,omitempty"`
	BlockedReasonCode  string   `json:"blocked_reason_code,omitempty"`
	BlockedCategory    string   `json:"blocked_category,omitempty"`
	ActionEndpointHint string   `json:"action_endpoint_hint,omitempty"`
	EvidencePostID     string   `json:"evidence_post_id,omitempty"`
}

type ActivityReplyPreview struct {
	Ts            int64    `json:"ts"`
	PostID        string   `json:"post_id"`
	ReplyTo       string   `json:"reply_to,omitempty"`
	AuthorRole    string   `json:"author_role,omitempty"`
	PostKind      string   `json:"post_kind,omitempty"`
	SourceEventID int64    `json:"source_event_id,omitempty"`
	Summary       string   `json:"summary,omitempty"`
	CausalRefs    []string `json:"causal_refs,omitempty"`
}

func (s *Server) getActivity(w http.ResponseWriter, r *http.Request) {
	if s.root == nil {
		writeErr(w, http.StatusInternalServerError, "root 未初始化")
		return
	}
	limit := parseActivityLimit(r, 50)
	projectFilter := strings.TrimSpace(r.URL.Query().Get("project"))

	questCache := map[string]*fsstore.QuestMeta{}
	questMiss := map[string]bool{}
	advCache := map[string]*fsstore.AdventurerFile{}

	loadQuest := func(qid string) *fsstore.QuestMeta {
		if qid == "" || questMiss[qid] {
			return nil
		}
		if q, ok := questCache[qid]; ok {
			return q
		}
		q, err := s.engine.GetQuest(qid)
		if err != nil || q == nil {
			questMiss[qid] = true
			return nil
		}
		questCache[qid] = q
		return q
	}
	loadAdv := func(id string) *fsstore.AdventurerFile {
		if id == "" {
			return nil
		}
		if a, ok := advCache[id]; ok {
			return a
		}
		a, err := s.engine.GetAdventurer(id)
		if err != nil || a == nil {
			advCache[id] = nil
			return nil
		}
		advCache[id] = a
		return a
	}

	projectsSet := map[string]bool{}
	items, err := s.buildActivityItemsFromThreadPosts(projectFilter, 0, loadAdv, projectsSet)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	seenItems := map[string]bool{}
	for _, item := range items {
		seenItems[activityDedupeKey(item)] = true
		for _, preview := range item.ReplyPreviews {
			if preview.PostID != "" {
				seenItems[item.QuestID+":post:"+preview.PostID] = true
			}
		}
	}
	items = foldFanoutRootActivityItems(items)

	rows, err := s.root.ReadGlobalEvents(2000)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	for _, row := range rows {
		item, ok := s.buildActivityItem(row, projectFilter, loadQuest, loadAdv, projectsSet)
		if !ok {
			continue
		}
		key := activityDedupeKey(item)
		if seenItems[key] {
			continue
		}
		items = append(items, item)
	}
	items = foldFanoutRootActivityItems(items)

	if quests, err := s.engine.ListQuests(); err == nil {
		qs := fsstore.NewQuestStore(s.root)
		for _, q := range quests {
			item, ok := buildAttentionActivityItem(q, projectFilter, projectsSet, qs)
			if !ok {
				continue
			}
			key := activityDedupeKey(item)
			if seenItems[key] {
				continue
			}
			items = append(items, item)
			seenItems[key] = true
		}
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Kind == "attention" || items[j].Kind == "attention" {
			if items[i].Kind != items[j].Kind {
				return items[i].Kind == "attention"
			}
			if items[i].AttentionPriority != items[j].AttentionPriority {
				return items[i].AttentionPriority < items[j].AttentionPriority
			}
		}
		if items[i].Ts == items[j].Ts {
			return items[i].PostID > items[j].PostID
		}
		return items[i].Ts > items[j].Ts
	})
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}

	// projects 列表：始终从全部 quest 取，不受 projectFilter 影响。
	if qs, err := s.engine.ListQuests(); err == nil {
		for _, q := range qs {
			projectsSet[projectName(q.BaseWorkingDir)] = true
		}
	}
	projects := make([]string, 0, len(projectsSet))
	for p := range projectsSet {
		projects = append(projects, p)
	}
	sort.Strings(projects)

	if items == nil {
		items = []ActivityItem{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":       true,
		"items":    items,
		"projects": projects,
	})
}

func activityDedupeKey(item ActivityItem) string {
	if item.PostID != "" {
		return item.QuestID + ":post:" + item.PostID
	}
	return item.QuestID + ":" + item.Kind + ":" + strconv.FormatInt(item.Ts, 10)
}

func (s *Server) buildActivityItemsFromThreadPosts(
	projectFilter string,
	limit int,
	loadAdv func(string) *fsstore.AdventurerFile,
	projectsSet map[string]bool,
) ([]ActivityItem, error) {
	if s == nil || s.engine == nil {
		return nil, nil
	}
	quests, err := s.engine.ListQuests()
	if err != nil {
		return nil, err
	}
	var items []ActivityItem
	for _, q := range quests {
		if q == nil {
			continue
		}
		posts, err := fsstore.NewQuestStore(s.root).LoadThreadPosts(q.ID)
		if err != nil {
			return nil, err
		}
		if questItems, ok := s.buildActivityItemFromThreadPosts(posts, q, projectFilter, loadAdv, projectsSet); ok {
			items = append(items, questItems...)
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Ts == items[j].Ts {
			return items[i].PostID > items[j].PostID
		}
		return items[i].Ts > items[j].Ts
	})
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func (s *Server) buildActivityItemFromThreadPosts(
	posts []fsstore.ThreadPost,
	q *fsstore.QuestMeta,
	projectFilter string,
	loadAdv func(string) *fsstore.AdventurerFile,
	projectsSet map[string]bool,
) ([]ActivityItem, bool) {
	if q == nil || len(posts) == 0 {
		return nil, false
	}
	rootIdx := -1
	for i := range posts {
		if posts[i].PostID == fsstore.RootPostID(q.ID) || posts[i].Kind == "root" {
			rootIdx = i
			break
		}
	}
	if rootIdx < 0 {
		return nil, false
	}
	item, ok := s.buildActivityItemFromThreadPost(posts[rootIdx], q, projectFilter, loadAdv, projectsSet)
	if !ok {
		return nil, false
	}
	var out []ActivityItem
	previews := make([]ActivityReplyPreview, 0, len(posts)-1)
	for i := range posts {
		if i == rootIdx {
			continue
		}
		post := posts[i]
		if post.PostID == "" {
			continue
		}
		if post.ParentReplyID == "" {
			if isTopLevelActivityPostKind(post.Kind) {
				if top, ok := s.buildActivityItemFromThreadPost(post, q, projectFilter, loadAdv, projectsSet); ok {
					out = append(out, top)
				}
				continue
			}
			previews = append(previews, ActivityReplyPreview{
				Ts:            post.CreatedAtMs,
				PostID:        post.PostID,
				ReplyTo:       fsstore.RootPostID(q.ID),
				AuthorRole:    string(post.AuthorRole),
				PostKind:      post.Kind,
				SourceEventID: post.SourceEventID,
				Summary:       activityPreviewSummary(post.Content),
				CausalRefs:    append([]string(nil), post.CausalRefs...),
			})
			continue
		}
		previews = append(previews, ActivityReplyPreview{
			Ts:            post.CreatedAtMs,
			PostID:        post.PostID,
			ReplyTo:       post.ParentReplyID,
			AuthorRole:    string(post.AuthorRole),
			PostKind:      post.Kind,
			SourceEventID: post.SourceEventID,
			Summary:       activityPreviewSummary(post.Content),
			CausalRefs:    append([]string(nil), post.CausalRefs...),
		})
	}
	sort.SliceStable(previews, func(i, j int) bool {
		if previews[i].Ts == previews[j].Ts {
			return previews[i].PostID > previews[j].PostID
		}
		return previews[i].Ts > previews[j].Ts
	})
	item.ReplyCount = len(previews)
	if len(previews) > 3 {
		item.ReplyPreviews = previews[:3]
	} else {
		item.ReplyPreviews = previews
	}
	out = append(out, item)
	return out, true
}

func isTopLevelActivityPostKind(kind string) bool {
	switch kind {
	case "milestone", "blocker", "system_escalation", "decision_note":
		return true
	default:
		return false
	}
}

func activityPreviewSummary(content string) string {
	const maxRunes = 200
	runes := []rune(content)
	if len(runes) <= maxRunes {
		return content
	}
	return string(runes[:maxRunes]) + "…"
}

func (s *Server) buildActivityItemFromThreadPost(
	post fsstore.ThreadPost,
	q *fsstore.QuestMeta,
	projectFilter string,
	loadAdv func(string) *fsstore.AdventurerFile,
	projectsSet map[string]bool,
) (ActivityItem, bool) {
	if q == nil {
		return ActivityItem{}, false
	}
	project := projectName(q.BaseWorkingDir)
	if projectFilter != "" && project != projectFilter {
		return ActivityItem{}, false
	}
	projectsSet[project] = true

	kind := "post"
	if post.PostID == fsstore.RootPostID(q.ID) || post.Kind == "root" {
		kind = "user"
	}
	speakerID := post.AuthorIdentity
	if kind == "user" {
		speakerID = ""
	}
	item := ActivityItem{
		Ts:            post.CreatedAtMs,
		Kind:          kind,
		QuestID:       q.ID,
		ThreadID:      post.ThreadID,
		RootPostID:    post.RootPostID,
		ShortID:       shortIDFromQuest(q.ID, q),
		Query:         q.Query,
		Project:       project,
		WarriorID:     speakerID,
		Status:        string(q.Status),
		Summary:       post.Content,
		Source:        activitySource(q),
		PostID:        post.PostID,
		ReplyTo:       post.ParentReplyID,
		AuthorRole:    string(post.AuthorRole),
		CausalRefs:    append([]string(nil), post.CausalRefs...),
		PostKind:      post.Kind,
		SourceEventID: post.SourceEventID,
		QuestType:     string(q.Type),
		WarriorAdvID:  q.WarriorID,
		MageAdvID:     q.MageID,
		ParentQuestID: q.ParentQuestID,
		ChildQuestID:  q.ChildExecuteQuestID,
	}
	if kind == "user" {
		item.PostKind = ""
		applyFanoutRootFields(&item, q)
	}
	if q.WarriorID != "" {
		if a := loadAdv(q.WarriorID); a != nil {
			item.WarriorAdvName = a.Name
		}
	}
	if q.MageID != "" {
		if a := loadAdv(q.MageID); a != nil {
			item.MageAdvName = a.Name
		}
	}
	if speakerID != "" {
		if a := loadAdv(speakerID); a != nil {
			item.WarriorName = a.Name
			item.WarriorLevel = a.Level
			item.WarriorTitle = a.GetTitle()
			item.WarriorClass = string(a.Class)
			item.WarriorExp = a.Exp
		}
	}
	return item, true
}

func foldFanoutRootActivityItems(items []ActivityItem) []ActivityItem {
	if len(items) == 0 {
		return items
	}
	type group struct {
		items []ActivityItem
	}
	groups := map[string]*group{}
	for _, item := range items {
		if !shouldFoldFanoutRoot(item) {
			continue
		}
		g := groups[item.GroupID]
		if g == nil {
			g = &group{}
			groups[item.GroupID] = g
		}
		g.items = append(g.items, item)
	}
	if len(groups) == 0 {
		return items
	}

	replacements := map[string]ActivityItem{}
	skip := map[string]bool{}
	for groupID, g := range groups {
		if len(g.items) < 2 {
			continue
		}
		sort.SliceStable(g.items, func(i, j int) bool {
			left := ""
			right := ""
			if len(g.items[i].FanoutLeafIDs) > 0 {
				left = g.items[i].FanoutLeafIDs[0]
			}
			if len(g.items[j].FanoutLeafIDs) > 0 {
				right = g.items[j].FanoutLeafIDs[0]
			}
			if left == right {
				return g.items[i].QuestID < g.items[j].QuestID
			}
			return left < right
		})
		fold := g.items[0]
		fold.GroupID = groupID
		fold.FanoutFold = true
		fold.FanoutLeafCount = len(g.items)
		fold.FanoutLeafIDs = make([]string, 0, len(g.items))
		fold.FanoutQuestIDs = make([]string, 0, len(g.items))
		fold.Summary = fmt.Sprintf("展开 %d 个分支", len(g.items))
		fold.ReplyCount = 0
		fold.ReplyPreviews = nil
		var previews []ActivityReplyPreview
		for _, item := range g.items {
			if len(item.FanoutLeafIDs) > 0 {
				fold.FanoutLeafIDs = append(fold.FanoutLeafIDs, item.FanoutLeafIDs[0])
			}
			fold.FanoutQuestIDs = append(fold.FanoutQuestIDs, item.QuestID)
			fold.ReplyCount += item.ReplyCount
			previews = append(previews, item.ReplyPreviews...)
			if item.Ts > fold.Ts {
				fold.Ts = item.Ts
				fold.QuestID = item.QuestID
				fold.ThreadID = item.ThreadID
				fold.RootPostID = item.RootPostID
				fold.ShortID = item.ShortID
				fold.Query = item.Query
				fold.Status = item.Status
				fold.PostID = item.PostID
			}
			skip[itemKey(item)] = true
		}
		sort.SliceStable(previews, func(i, j int) bool {
			if previews[i].Ts == previews[j].Ts {
				return previews[i].PostID > previews[j].PostID
			}
			return previews[i].Ts > previews[j].Ts
		})
		if len(previews) > 3 {
			fold.ReplyPreviews = previews[:3]
		} else {
			fold.ReplyPreviews = previews
		}
		replacements[itemKey(fold)] = fold
	}
	if len(replacements) == 0 {
		return items
	}
	out := make([]ActivityItem, 0, len(items))
	added := map[string]bool{}
	for _, item := range items {
		key := itemKey(item)
		if repl, ok := replacements[key]; ok && !added[repl.GroupID] {
			out = append(out, repl)
			added[repl.GroupID] = true
			continue
		}
		if skip[key] {
			continue
		}
		out = append(out, item)
	}
	return out
}

func shouldFoldFanoutRoot(item ActivityItem) bool {
	return item.Kind == "user" &&
		item.GroupID != "" &&
		len(item.FanoutLeafIDs) == 1 &&
		item.PostID == item.RootPostID
}

func itemKey(item ActivityItem) string {
	return item.QuestID + ":" + item.PostID + ":" + strconv.FormatInt(item.Ts, 10)
}

// buildActivityItem 把一条全局事件转成 Feed 动态；非动态事件返回 ok=false。
func (s *Server) buildActivityItem(
	row fsstore.GlobalEventRow,
	projectFilter string,
	loadQuest func(string) *fsstore.QuestMeta,
	loadAdv func(string) *fsstore.AdventurerFile,
	projectsSet map[string]bool,
) (ActivityItem, bool) {
	var kind string
	switch row.Type {
	case "quest.created":
		kind = "user"
	case "agent.post":
		kind = "post"
	case "quest.success":
		kind = "summary"
	default:
		return ActivityItem{}, false
	}

	q := loadQuest(row.QuestID)
	query := ""
	project := ""
	var warriorID string
	status := ""
	var impact *model.ImpactSummary
	source := ""
	if q != nil {
		query = q.Query
		project = projectName(q.BaseWorkingDir)
		warriorID = q.WarriorID
		status = string(q.Status)
		if kind == "summary" {
			impact = impactPtr(q.ImpactSummary)
		}
		source = activitySource(q)
	}
	if query == "" {
		query = payloadStr(row.Payload, "query")
	}
	if project == "" {
		project = "(未分类)"
	}

	// post 的发言者由 sid 推断：mage_X → 法师，warrior_X → 剑士。
	speakerID := warriorID
	if kind == "post" {
		if sid := row.SessionID; strings.HasPrefix(sid, "mage_") && q != nil && q.MageID != "" {
			speakerID = q.MageID
		}
	} else if kind == "user" {
		speakerID = ""
	}

	if projectFilter != "" && project != projectFilter {
		return ActivityItem{}, false
	}
	projectsSet[project] = true
	if kind == "post" && !isTopLevelActivityPostKind(payloadStr(row.Payload, "kind")) {
		return ActivityItem{}, false
	}

	item := ActivityItem{
		Ts:            row.Timestamp,
		Kind:          kind,
		QuestID:       row.QuestID,
		ThreadID:      row.QuestID,
		RootPostID:    fsstore.RootPostID(row.QuestID),
		ShortID:       shortIDFromQuest(row.QuestID, q),
		Query:         query,
		Project:       project,
		WarriorID:     speakerID,
		Status:        status,
		Impact:        impact,
		Source:        source,
		SourceEventID: row.EventID,
	}

	// 委托帖上下文
	if kind == "user" && q != nil {
		item.QuestType = string(q.Type)
		item.WarriorAdvID = q.WarriorID
		item.MageAdvID = q.MageID
		item.ParentQuestID = q.ParentQuestID
		item.ChildQuestID = q.ChildExecuteQuestID
		item.TriageMode = q.TriageMode
		applyFanoutRootFields(&item, q)
		if q.WarriorID != "" {
			if a := loadAdv(q.WarriorID); a != nil {
				item.WarriorAdvName = a.Name
			}
		}
		if q.MageID != "" {
			if a := loadAdv(q.MageID); a != nil {
				item.MageAdvName = a.Name
			}
		}
		if q.ParentQuestID != "" {
			if pq := loadQuest(q.ParentQuestID); pq != nil {
				item.ParentShortID = shortIDFromQuest(q.ParentQuestID, pq)
			} else {
				item.ParentShortID = strings.TrimPrefix(q.ParentQuestID, "qst_")
			}
		}
		if q.ChildExecuteQuestID != "" {
			if cq := loadQuest(q.ChildExecuteQuestID); cq != nil {
				item.ChildShortID = shortIDFromQuest(q.ChildExecuteQuestID, cq)
			} else {
				item.ChildShortID = strings.TrimPrefix(q.ChildExecuteQuestID, "qst_")
			}
		}
	}

	// user 发帖：query 作正文
	if kind == "user" {
		item.Summary = query
		item.PostID = fsstore.RootPostID(row.QuestID)
		if q != nil && q.Source() == model.SourceAutomation {
			item.AuthorRole = string(model.PostRoleAutomation)
		} else {
			item.AuthorRole = string(model.PostRoleHuman)
		}
	}

	// agent post：content 作正文
	if kind == "post" {
		item.Summary = payloadStr(row.Payload, "content")
		item.PostID = payloadStr(row.Payload, "post_id")
		if item.PostID == "" && row.EventID > 0 {
			item.PostID = fmt.Sprintf("post_%d", row.EventID)
		}
		item.ReplyTo = payloadStr(row.Payload, "reply_to")
		item.AuthorRole = payloadStr(row.Payload, "author_role")
		item.CausalRefs = payloadStringSlice(row.Payload, "causal_refs")
		item.PostKind = payloadStr(row.Payload, "kind")
	}

	// quest 完成：agent 自己写的交付摘要作为 Feed 兜底
	if kind == "summary" && q != nil {
		switch {
		case q.WarriorSummary != "":
			item.Summary = q.WarriorSummary
		case q.ImpactSummary.WhatChanged != "":
			item.Summary = q.ImpactSummary.WhatChanged
		case q.FinalComment != "":
			item.Summary = q.FinalComment
		default:
			item.Summary = "委托已完成"
		}
	}

	// 冒险者信息
	if speakerID != "" {
		if a := loadAdv(speakerID); a != nil {
			item.WarriorName = a.Name
			item.WarriorLevel = a.Level
			item.WarriorTitle = a.GetTitle()
			item.WarriorClass = string(a.Class)
			item.WarriorExp = a.Exp
		}
	}

	return item, true
}

func buildAttentionActivityItem(q *fsstore.QuestMeta, projectFilter string, projectsSet map[string]bool, qs *fsstore.QuestStore) (ActivityItem, bool) {
	item := fsstore.HumanExceptionFromQuest(q)
	if item == nil || q == nil {
		return ActivityItem{}, false
	}
	project := projectName(q.BaseWorkingDir)
	if projectFilter != "" && project != projectFilter {
		return ActivityItem{}, false
	}
	projectsSet[project] = true

	// Find durable narrative evidence post for this attention item.
	evidencePostID := findAttentionEvidencePostID(qs, q)

	return ActivityItem{
		Ts:                 item.CreatedAtMs,
		Kind:               "attention",
		QuestID:            q.ID,
		ThreadID:           q.ID,
		RootPostID:         fsstore.RootPostID(q.ID),
		ShortID:            shortIDFromQuest(q.ID, q),
		Query:              q.Query,
		Project:            project,
		Status:             string(q.Status),
		Summary:            item.Reason,
		Source:             activitySource(q),
		PostID:             "attention_" + q.ID,
		AuthorRole:         string(model.PostRoleSystem),
		PostKind:           "attention",
		QuestType:          string(q.Type),
		WarriorAdvID:       q.WarriorID,
		MageAdvID:          q.MageID,
		ParentQuestID:      q.ParentQuestID,
		ChildQuestID:       q.ChildExecuteQuestID,
		HumanExceptionID:   item.ID,
		AvailableActions:   append([]string(nil), item.AvailableActions...),
		RecommendedAction:  item.RecommendedAction,
		RiskLevel:          item.RiskLevel,
		AttentionPriority:  item.Priority,
		BlockedReasonCode:  q.BlockedReasonCode,
		BlockedCategory:    q.BlockedCategory,
		ActionEndpointHint: actionEndpointHintForQuest(q),
		EvidencePostID:     evidencePostID,
	}, true
}

// findAttentionEvidencePostID looks up the durable ThreadPost that serves as
// narrative evidence for an attention item. Returns empty string if no matching
// post is found (e.g. legacy quests without projections).
func findAttentionEvidencePostID(qs *fsstore.QuestStore, q *fsstore.QuestMeta) string {
	if qs == nil || q == nil {
		return ""
	}
	posts, err := qs.LoadThreadPosts(q.ID)
	if err != nil || len(posts) == 0 {
		return ""
	}

	// Determine which post kind to look for based on the attention trigger.
	var targetKind string
	switch {
	case q.Status == model.QuestStatusBlocked:
		targetKind = "system_escalation"
	case q.Status == model.QuestStatusWaitingInput:
		targetKind = "system_waiting_input"
	case q.Status == model.QuestStatusUserReview:
		targetKind = "review_report"
	case q.ApplyStatus == model.ApplyStatusFailed:
		targetKind = "decision_note"
	default:
		return ""
	}

	// Find the latest matching post (highest created_at_ms).
	var latest *fsstore.ThreadPost
	for i := range posts {
		p := &posts[i]
		if p.Kind != targetKind {
			continue
		}
		if latest == nil || p.CreatedAtMs > latest.CreatedAtMs {
			latest = p
		}
	}
	if latest != nil {
		return latest.PostID
	}
	return ""
}

func actionEndpointHintForQuest(q *fsstore.QuestMeta) string {
	if q == nil {
		return ""
	}
	switch {
	case q.Status == model.QuestStatusBlocked:
		return "resolve-blocked"
	case q.Status == model.QuestStatusWaitingInput:
		return "answer"
	case q.Status == model.QuestStatusUserReview:
		return "resolve-user-review"
	case q.ApplyStatus == model.ApplyStatusFailed:
		return "apply-discard"
	default:
		return ""
	}
}

func applyFanoutRootFields(item *ActivityItem, q *fsstore.QuestMeta) {
	if item == nil || q == nil || item.Kind != "user" || strings.TrimSpace(q.GroupID) == "" || strings.TrimSpace(q.FanoutLeafID) == "" {
		return
	}
	item.GroupID = q.GroupID
	item.FanoutLeafIDs = []string{q.FanoutLeafID}
	item.FanoutQuestIDs = []string{q.ID}
	item.MergeOwnerLeafID = q.MergeOwnerLeafID
}

// ==================== helpers ====================

func parseActivityLimit(r *http.Request, def int) int {
	limit := def
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil {
			limit = n
		}
	}
	if limit < 1 {
		limit = def
	}
	if limit > 200 {
		limit = 200
	}
	return limit
}

func payloadStr(p any, key string) string {
	m, ok := p.(map[string]any)
	if !ok {
		return ""
	}
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func payloadInt(p any, key string) int {
	m, ok := p.(map[string]any)
	if !ok {
		return 0
	}
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	return 0
}

func payloadStringSlice(p any, key string) []string {
	m, ok := p.(map[string]any)
	if !ok {
		return nil
	}
	v, ok := m[key]
	if !ok {
		return nil
	}
	switch x := v.(type) {
	case []string:
		return append([]string(nil), x...)
	case []any:
		out := make([]string, 0, len(x))
		for _, item := range x {
			if s, ok := item.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func projectName(baseDir string) string {
	if baseDir == "" {
		return "(未分类)"
	}
	name := filepath.Base(strings.TrimRight(baseDir, "/"))
	switch name {
	case "", ".", "work":
		return "(未分类)"
	}
	return name
}

func shortIDFromQuest(qid string, q *fsstore.QuestMeta) string {
	if q != nil && q.ShortID != "" {
		return q.ShortID
	}
	return strings.TrimPrefix(qid, "qst_")
}

func activitySource(q *fsstore.QuestMeta) string {
	if q == nil {
		return ""
	}
	if q.ParentQuestID != "" {
		return "parent_spawn"
	}
	return q.CreatedBy
}

func impactPtr(s model.ImpactSummary) *model.ImpactSummary {
	if s.IsEmpty() {
		return nil
	}
	return &s
}
