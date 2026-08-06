package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
	"code.byted.org/lihuanyu.0w0/gloop/internal/orchestrator"
	"code.byted.org/lihuanyu.0w0/gloop/internal/policy"
)

func (s *Server) listQuests(w http.ResponseWriter, r *http.Request) {
	qs, err := s.engine.ListQuests()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if qs == nil {
		qs = []*fsstore.QuestMeta{}
	}
	qs = filterQuestList(s.root, qs, parseQuestListFilter(r))
	items := make([]map[string]any, 0, len(qs))
	for _, q := range qs {
		items = append(items, s.questResponse(q))
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "items": items})
}

type questListFilter struct {
	official          *bool
	statuses          map[string]bool
	types             map[string]bool
	questIDs          map[string]bool
	projects          map[string]bool
	failureReasons    map[string]bool
	failureCategories map[string]bool
	createdBy         string
	adventurerID      string
	createdSince      int64
	updatedSince      int64
}

func parseQuestListFilter(r *http.Request) questListFilter {
	q := r.URL.Query()
	filter := questListFilter{
		statuses:          stringSet(q["status"]),
		types:             stringSet(q["type"]),
		questIDs:          stringSet(q["quest_id"]),
		projects:          stringSet(q["project"]),
		failureReasons:    stringSet(q["failure_reason"]),
		failureCategories: stringSet(q["failure_category"]),
		createdBy:         q.Get("created_by"),
		adventurerID:      q.Get("adventurer_id"),
		createdSince:      parseQuestSince(r, "created_since_ms", "created_after_ms"),
	}
	if raw := q.Get("official"); raw != "" {
		want, _ := strconv.ParseBool(raw)
		filter.official = &want
	}
	if since := parseQuestSince(r, "updated_since_ms", "updated_after_ms", "since_ms"); since > 0 {
		filter.updatedSince = since
	} else {
		filter.updatedSince = parseQuestRangeSince(r)
	}
	return filter
}

func stringSet(values []string) map[string]bool {
	if len(values) == 0 {
		return nil
	}
	out := map[string]bool{}
	for _, raw := range values {
		if raw != "" {
			out[raw] = true
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func filterQuestList(root *fsstore.Root, qs []*fsstore.QuestMeta, filter questListFilter) []*fsstore.QuestMeta {
	if len(qs) == 0 {
		return qs
	}
	out := make([]*fsstore.QuestMeta, 0, len(qs))
	for _, q := range qs {
		if q == nil || !questMatchesFilter(root, q, filter) {
			continue
		}
		out = append(out, q)
	}
	return out
}

func questMatchesFilter(root *fsstore.Root, q *fsstore.QuestMeta, filter questListFilter) bool {
	if filter.official != nil && root.IsQuestOfficial(q) != *filter.official {
		return false
	}
	if len(filter.statuses) > 0 && !filter.statuses[string(q.Status)] {
		if !(filter.statuses["apply_failed"] && q.ApplyStatus == model.ApplyStatusFailed) {
			return false
		}
	}
	if len(filter.types) > 0 && !filter.types[string(q.Type)] {
		return false
	}
	if len(filter.questIDs) > 0 && !filter.questIDs[q.ID] {
		return false
	}
	if len(filter.projects) > 0 && !filter.projects[questFilterProject(q)] {
		return false
	}
	if len(filter.failureReasons) > 0 && !questFilterFailureReasons(root, q, filter.failureReasons) {
		return false
	}
	if len(filter.failureCategories) > 0 && !filter.failureCategories[questFilterFailureCategory(q)] {
		return false
	}
	if filter.adventurerID != "" && q.WarriorID != filter.adventurerID && q.MageID != filter.adventurerID {
		return false
	}
	if filter.createdBy != "" && q.CreatedBy != filter.createdBy {
		return false
	}
	if filter.createdSince > 0 && q.CreatedAtMs < filter.createdSince {
		return false
	}
	if filter.updatedSince > 0 && questActivityTime(q) < filter.updatedSince {
		return false
	}
	return true
}

func questFilterProject(q *fsstore.QuestMeta) string {
	if q == nil {
		return ""
	}
	if q.BaseWorkingDir != "" {
		return filepath.Base(q.BaseWorkingDir)
	}
	if q.WorkspacePath != "" {
		return filepath.Base(q.WorkspacePath)
	}
	return ""
}

func questFilterFailureReasons(root *fsstore.Root, q *fsstore.QuestMeta, want map[string]bool) bool {
	if len(want) == 0 {
		return true
	}
	if q == nil {
		return false
	}
	if q.FailureAttribution != nil && q.FailureAttribution.Reason != "" {
		return want[q.FailureAttribution.Reason]
	}
	if q.Status == model.QuestStatusBlocked && q.BlockedReasonCode != "" {
		if want[q.BlockedReasonCode] {
			return true
		}
	}
	if q.Status == model.QuestStatusBlocked && q.BlockedReason != "" {
		if want[normalizeQuestFilterReason(q.BlockedReason)] {
			return true
		}
	}
	if q.Status == model.QuestStatusCancelled {
		if want["cancelled"] {
			return true
		}
	}
	if q.Status == model.QuestStatusFailed {
		if q.FinalVerdict != "" {
			if want[string(q.FinalVerdict)] {
				return true
			}
		}
		if want[normalizeQuestFilterReason(q.FinalComment)] {
			return true
		}
	}
	if q.ApplyStatus == model.ApplyStatusFailed && want["apply_failed"] {
		return true
	}
	if root == nil {
		return false
	}
	qs := fsstore.NewQuestStore(root)
	sessionIDs, _ := qs.ListSessionIDs(q.ID)
	for _, sid := range sessionIDs {
		rows, rErr := qs.ReadSessionRows(q.ID, sid, 0)
		if rErr != nil {
			continue
		}
		for _, row := range rows {
			switch row.Kind {
			case "error":
				reason := normalizeQuestFilterReason(row.Error)
				if reason == "" {
					reason = normalizeQuestFilterReason(row.Content)
				}
				if want[reason] {
					return true
				}
			case "budget_exceeded":
				if want["duration_exceeded"] {
					return true
				}
			case "no_progress":
				if want["no_progress"] {
					return true
				}
			}
		}
	}
	events, _ := qs.ReadEvents(q.ID, 0)
	for _, ev := range events {
		if ev.Type != "micro.command_run" {
			continue
		}
		payload, ok := ev.Payload.(map[string]any)
		if !ok {
			continue
		}
		if timedOut, _ := payload["timed_out"].(bool); timedOut && want["command_timeout"] {
			return true
		}
		if exit := int64FromQuestFilterAny(payload["exit_code"]); exit != 0 && want["command_failed"] {
			return true
		}
	}
	return false
}

func questFilterFailureCategory(q *fsstore.QuestMeta) string {
	if q == nil || q.FailureAttribution == nil {
		return ""
	}
	return q.FailureAttribution.Category
}

func normalizeQuestFilterReason(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return ""
	}
	switch {
	case strings.Contains(s, "apply"):
		return "apply_failed"
	case strings.Contains(s, "context deadline exceeded"), strings.Contains(s, "timeout"), strings.Contains(s, "超时"):
		return "timeout"
	case strings.Contains(s, "duration_exceeded"), strings.Contains(s, "时长超限"):
		return "duration_exceeded"
	case strings.Contains(s, "no_progress"), strings.Contains(s, "无进展"):
		return "no_progress"
	case strings.Contains(s, "permission"), strings.Contains(s, "权限"):
		return "permission"
	case strings.Contains(s, "command"):
		return "command_failed"
	case strings.Contains(s, "agent 连续错误"):
		return "agent_consecutive_errors"
	}
	if runeCount := len([]rune(s)); runeCount > 80 {
		s = string([]rune(s)[:80])
	}
	return s
}

func int64FromQuestFilterAny(v any) int64 {
	switch x := v.(type) {
	case int:
		return int64(x)
	case int64:
		return x
	case float64:
		return int64(x)
	case json.Number:
		n, _ := x.Int64()
		return n
	default:
		return 0
	}
}

func parseQuestSince(r *http.Request, keys ...string) int64 {
	for _, key := range keys {
		if raw := r.URL.Query().Get(key); raw != "" {
			if n, err := strconv.ParseInt(raw, 10, 64); err == nil && n > 0 {
				return n
			}
		}
	}
	return 0
}

func parseQuestRangeSince(r *http.Request) int64 {
	now := fsstore.NowMs()
	switch r.URL.Query().Get("range") {
	case "today":
		t := fsstore.FromMs(now)
		y, m, d := t.Date()
		return time.Date(y, m, d, 0, 0, 0, 0, t.Location()).UnixMilli()
	case "week", "7d":
		return now - int64(7*24*time.Hour/time.Millisecond)
	case "30d", "month":
		return now - int64(30*24*time.Hour/time.Millisecond)
	default:
		return 0
	}
}

func questActivityTime(q *fsstore.QuestMeta) int64 {
	if q == nil {
		return 0
	}
	if q.UpdatedAtMs > 0 {
		return q.UpdatedAtMs
	}
	if q.CompletedAtMs > 0 {
		return q.CompletedAtMs
	}
	if q.StartedAtMs > 0 {
		return q.StartedAtMs
	}
	return q.CreatedAtMs
}

func (s *Server) getQuest(w http.ResponseWriter, r *http.Request) {
	qid := r.PathValue("id")
	if qid == "" {
		writeErr(w, http.StatusBadRequest, "missing quest id")
		return
	}
	q, err := s.engine.GetQuest(qid)
	if err != nil {
		writeErr(w, http.StatusNotFound, err.Error())
		return
	}
	resp := map[string]any{"ok": true, "quest": s.questResponse(q)}
	// 附加评审记录、checks 结果、方案文档（如果有）
	if s.root != nil {
		qs := fsstore.NewQuestStore(s.root)
		if group := s.fanoutGroupResponse(q); group != nil {
			resp["fanout_group"] = group
		}
		if reviews, rErr := qs.LoadReviews(qid); rErr == nil && len(reviews) > 0 {
			resp["reviews"] = reviews
		}
		if reports, repErr := s.root.LoadReports(qid); repErr == nil && reports != nil {
			resp["reports"] = reports
		}
		if posts, postErr := qs.LoadThreadPosts(qid); postErr == nil && len(posts) > 0 {
			resp["thread_posts"] = posts
		}
		if loopState, lsErr := s.root.LoadLoopStateSpine(qid); lsErr == nil && loopState != nil {
			resp["loop_state_spine"] = loopState
		}
		var checks []map[string]any
		if lErr := qs.LoadChecks(qid, &checks); lErr == nil && len(checks) > 0 {
			resp["checks"] = checks
		}
		if q.Type == model.QuestTypeDesign {
			if doc, dErr := qs.LoadDesignDoc(qid); dErr == nil {
				resp["design"] = doc
			}
		}
		if sessions, sErr := s.root.ListAgentSessionStates(qid); sErr == nil && len(sessions) > 0 {
			resp["agent_sessions"] = sessions
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) fanoutGroupResponse(current *fsstore.QuestMeta) map[string]any {
	if s == nil || s.engine == nil || current == nil || strings.TrimSpace(current.GroupID) == "" {
		return nil
	}
	items, err := s.engine.ListQuests()
	if err != nil {
		return nil
	}
	leaves := make([]map[string]any, 0)
	for _, q := range items {
		if q == nil || q.GroupID != current.GroupID || strings.TrimSpace(q.FanoutLeafID) == "" {
			continue
		}
		leaves = append(leaves, map[string]any{
			"quest_id":            q.ID,
			"short_id":            q.ShortID,
			"query":               q.Query,
			"status":              q.Status,
			"fanout_leaf_id":      q.FanoutLeafID,
			"ownership_scopes":    q.OwnershipScopes,
			"merge_strategy":      q.MergeStrategy,
			"merge_owner_leaf_id": q.MergeOwnerLeafID,
			"warrior_id":          q.WarriorID,
			"mage_id":             q.MageID,
			"workflow_mode":       q.WorkflowMode,
			"pinned_outcome":      q.PinnedOutcomeSummary,
			"created_at_ms":       q.CreatedAtMs,
			"updated_at_ms":       q.UpdatedAtMs,
			"completed_at_ms":     q.CompletedAtMs,
			"is_current":          q.ID == current.ID,
		})
	}
	if len(leaves) == 0 {
		return nil
	}
	return map[string]any{
		"group_id": current.GroupID,
		"leaves":   leaves,
	}
}

func (s *Server) questResponse(q *fsstore.QuestMeta) map[string]any {
	raw, err := json.Marshal(q)
	if err != nil {
		return map[string]any{}
	}
	out := map[string]any{}
	_ = json.Unmarshal(raw, &out)
	effectType := s.questEffectType(q)
	out["effect_type"] = effectType
	out["workspace_diff_pending"] = questWorkspaceDiffPending(q)
	out["apply_failed"] = q != nil && q.ApplyStatus == model.ApplyStatusFailed
	if s != nil && s.root != nil && q != nil {
		if sessions, err := s.root.ListAgentSessionStates(q.ID); err == nil && len(sessions) > 0 {
			out["agent_sessions"] = sessions
		}
	}
	if s != nil && s.engine != nil && q != nil {
		if pos := s.engine.QueuePosition(q.ID); pos > 0 {
			out["queued"] = true
			out["queue_position"] = pos
		} else {
			out["queued"] = false
		}
		out["starting"] = s.engine.IsQuestStarting(q.ID)
	}
	if humanException := fsstore.HumanExceptionFromQuest(q); humanException != nil {
		out["human_exception"] = humanException
	}
	// Dynamic official flag: look up from automation config (with fallback to stored field for old data)
	if s != nil && s.root != nil {
		out["official"] = s.root.IsQuestOfficial(q)
	}
	out["policy_view"] = s.questPolicyView(q, effectType)
	return out
}

func (s *Server) questPolicyView(q *fsstore.QuestMeta, effectType string) map[string]any {
	if q == nil {
		return map[string]any{}
	}
	view := map[string]any{}
	facts := policy.NewFactBuilder(s.root).BuildReviewFacts(q, string(q.FinalVerdict), q.MageScore)
	facts.EffectType = effectType
	if reason, blocked := policy.HardGuardrailReason(facts); blocked {
		view["safety_floor_reason"] = reason
	}
	if autoID := q.AutomationID(); autoID != "" && s != nil && s.root != nil {
		if state, err := s.root.GetAutomationTrustState(autoID); err == nil && state != nil {
			view["effective_trust_tier"] = state.Tier
		} else if cfg, err := s.root.GetAutomation(autoID); err == nil && cfg != nil && cfg.TrustTier != "" {
			view["effective_trust_tier"] = cfg.TrustTier
		}
	}
	if q.PolicyDecisionID != "" || q.AutoPassedByPolicy != "" || q.AutoCompletedByPolicy != "" {
		action := "auto_pass"
		policyName := q.AutoPassedByPolicy
		if q.AutoCompletedByPolicy != "" {
			action = "auto_complete"
			policyName = q.AutoCompletedByPolicy
		}
		view["last_policy_decision"] = map[string]any{
			"id":          q.PolicyDecisionID,
			"policy_name": policyName,
			"action":      action,
			"reason":      "stored quest policy trace",
		}
	} else if decision := latestPolicyDecision(s.root, q.ID); decision != nil {
		view["last_policy_decision"] = decision
	}
	if summary := recoveryStateSummary(s.root, q.ID); summary != nil {
		view["recovery_state_summary"] = summary
	}
	return view
}

func latestPolicyDecision(root *fsstore.Root, qid string) map[string]any {
	if root == nil || qid == "" {
		return nil
	}
	rows, err := fsstore.NewQuestStore(root).ReadEvents(qid, 0)
	if err != nil {
		return nil
	}
	for i := len(rows) - 1; i >= 0; i-- {
		if rows[i].Type != "policy.decision" {
			continue
		}
		payload, ok := rows[i].Payload.(map[string]any)
		if !ok {
			continue
		}
		return map[string]any{
			"id":          stringValue(payload["decision_id"]),
			"policy_name": stringValue(payload["policy_name"]),
			"action":      stringValue(payload["policy_action"]),
			"reason":      stringValue(payload["reason"]),
			"at_ms":       rows[i].Timestamp,
		}
	}
	return nil
}

func recoveryStateSummary(root *fsstore.Root, qid string) map[string]any {
	if root == nil || qid == "" {
		return nil
	}
	state, err := root.GetRecoveryState(qid)
	if err != nil || state == nil {
		return nil
	}
	return map[string]any{
		"strategy":           state.LastAction,
		"attempt":            state.AttemptIndex,
		"max_attempts":       state.MaxAttempts,
		"next_attempt_at_ms": timeToMs(state.NextRetryAt),
		"status":             string(state.Status),
		"policy_name":        state.PolicyName,
		"last_error":         state.LastError,
	}
}

func timeToMs(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixMilli()
}

func stringValue(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

// questEffectType 推导 quest 的效应类型。
//
// 第一性原理：效应类型描述 quest 对外部世界产生了什么可感知的影响，
// 优先级从高到低：
//  1. workspace_diff：工作区内有实质文件变更，需要 apply 才能合并到 base
//  2. context_store：更新了全局上下文存储
//  3. external_side_effect：对外部系统产生了副作用（飞书文档、API 调用等）
//  4. none：没有可感知的外部影响
//
// 当剑士通过 phase_checkpoint.deliverables 显式声明了产出物时，以声明为准；
// 未声明时回退到 diff / automation 标签的启发式判断（兼容旧行为）。
func (s *Server) questEffectType(q *fsstore.QuestMeta) string {
	return policy.NewFactBuilder(s.root).EffectType(q)
}

// questWorkspaceDiffPending 判断是否有待应用的工作区变更。
// 只有 effect_type 为 workspace_diff 且 quest 已成功、未应用时才为 true。
// 这意味着：如果剑士声明了外部产出物（如飞书文档）且没有文件型产出，
// 即使 workspace 有 diff（临时草稿文件），也不会显示 apply 按钮。
func questWorkspaceDiffPending(q *fsstore.QuestMeta) bool {
	if q == nil {
		return false
	}
	if q.Status != model.QuestStatusSuccess {
		return false
	}
	if q.Applied {
		return false
	}
	if q.ApplyStatus == model.ApplyStatusFailed {
		return false
	}
	if q.ApplyStatus == model.ApplyStatusPending {
		return false
	}
	return q.DiffChangedFiles > 0 && !hasOnlyExternalOutputs(q)
}

// hasOnlyExternalOutputs 检查 quest 是否只有外部产出物（没有本地文件型产出）。
// 当只有外部产出物时，workspace diff 被视为临时文件，不触发 apply 流程。
func hasOnlyExternalOutputs(q *fsstore.QuestMeta) bool {
	if len(q.Outputs) == 0 {
		return false
	}
	hasExternal := false
	hasFile := false
	for _, out := range q.Outputs {
		if strings.HasPrefix(out.StoragePath, "http://") || strings.HasPrefix(out.StoragePath, "https://") {
			hasExternal = true
		} else if out.StoragePath != "" {
			hasFile = true
		}
	}
	return hasExternal && !hasFile
}

// ==================== ThreadView response types (P0b-4b) ====================

// ThreadActor 是 thread 参与者的身份解析结果。identity 是稳定键，
// 不是从帖子正文猜的人名。
type ThreadActor struct {
	Identity   string `json:"identity"`
	Role       string `json:"role"`        // maker / checker / system / human / automation
	Name       string `json:"name"`        // 解析出的冒险者名称
	Class      string `json:"class"`       // warrior / mage / system
	SourceType string `json:"source_type"` // adventurer / system / automation
}

// ThreadFanoutLeafPreview 是 fanout 分支的只读预览，不折进 root thread。
type ThreadFanoutLeafPreview struct {
	QuestID           string `json:"quest_id"`
	ShortID           string `json:"short_id"`
	Status            string `json:"status"`
	LeafID            string `json:"leaf_id"`
	LatestPostTs      int64  `json:"latest_post_ts"`
	LatestPostPreview string `json:"latest_post_preview"`
}

// ThreadFanoutSummary 是 read-time view model，只读 group/leaf QuestMeta。
type ThreadFanoutSummary struct {
	GroupID     string                    `json:"group_id"`
	IsRoot      bool                      `json:"is_root"`
	IsLeaf      bool                      `json:"is_leaf"`
	RootQuestID string                    `json:"root_quest_id,omitempty"`
	Leaves      []ThreadFanoutLeafPreview `json:"leaves,omitempty"`
}

// ThreadActionEntrypoint 是导航提示，不是授权。不返回可执行 payload。
type ThreadActionEntrypoint struct {
	ActionType     string `json:"action_type"`
	TargetSection  string `json:"target_section"`
	EvidencePostID string `json:"evidence_post_id,omitempty"`
	Priority       int    `json:"priority"`
	Hint           string `json:"hint,omitempty"`
}

func buildThreadActors(q *fsstore.QuestMeta, posts []fsstore.ThreadPost, loadAdv func(string) *fsstore.AdventurerFile) []ThreadActor {
	seen := map[string]ThreadActor{}
	add := func(identity, role string) {
		if identity == "" {
			return
		}
		if _, ok := seen[identity]; ok {
			return
		}
		actor := ThreadActor{
			Identity:   identity,
			Role:       role,
			SourceType: "system",
		}
		if adv := loadAdv(identity); adv != nil {
			actor.Name = adv.Name
			actor.Class = string(adv.Class)
			actor.SourceType = "adventurer"
		} else if identity == "system" {
			actor.Name = "System"
			actor.Class = "system"
		} else if strings.HasPrefix(identity, model.QuestSourcePrefix) {
			actor.Name = identity
			actor.SourceType = "automation"
		} else {
			actor.Name = identity
		}
		seen[identity] = actor
	}

	add(q.WarriorID, string(model.PostRoleMaker))
	add(q.MageID, string(model.PostRoleChecker))
	if q.Source() == model.SourceAutomation {
		add(q.CreatedBy, string(model.PostRoleAutomation))
	} else if q.CreatedBy != "" {
		add(q.CreatedBy, string(model.PostRoleHuman))
	}
	for _, p := range posts {
		add(p.AuthorIdentity, string(p.AuthorRole))
	}

	out := make([]ThreadActor, 0, len(seen))
	for _, a := range seen {
		out = append(out, a)
	}
	return out
}

func buildThreadFanoutSummary(q *fsstore.QuestMeta, qs *fsstore.QuestStore) *ThreadFanoutSummary {
	if q == nil || strings.TrimSpace(q.GroupID) == "" {
		return nil
	}
	summary := &ThreadFanoutSummary{
		GroupID: q.GroupID,
		IsRoot:  q.FanoutLeafID == "",
		IsLeaf:  q.FanoutLeafID != "",
	}
	if root, ok := qs.FindFanoutRoot(q.GroupID); ok {
		summary.RootQuestID = root.ID
	}
	if summary.IsRoot {
		leaves := qs.ListFanoutGroupLeaves(q.GroupID)
		for _, leaf := range leaves {
			preview := ThreadFanoutLeafPreview{
				QuestID: leaf.ID,
				ShortID: shortIDFromQuest(leaf.ID, leaf),
				Status:  string(leaf.Status),
				LeafID:  leaf.FanoutLeafID,
			}
			if leafPosts, err := qs.LoadThreadPosts(leaf.ID); err == nil && len(leafPosts) > 0 {
				for i := len(leafPosts) - 1; i >= 0; i-- {
					if leafPosts[i].PostID != fsstore.RootPostID(leaf.ID) {
						preview.LatestPostTs = leafPosts[i].CreatedAtMs
						preview.LatestPostPreview = activityPreviewSummary(leafPosts[i].Content)
						break
					}
				}
			}
			summary.Leaves = append(summary.Leaves, preview)
		}
	}
	return summary
}

func buildThreadActionEntrypoints(q *fsstore.QuestMeta, posts []fsstore.ThreadPost) []ThreadActionEntrypoint {
	if q == nil {
		return nil
	}
	var entries []ThreadActionEntrypoint
	findEvidencePost := func(kind string) string {
		for i := len(posts) - 1; i >= 0; i-- {
			if posts[i].Kind == kind {
				return posts[i].PostID
			}
		}
		return ""
	}

	switch q.Status {
	case model.QuestStatusWaitingInput:
		evidence := findEvidencePost("system_waiting_input")
		entries = append(entries, ThreadActionEntrypoint{
			ActionType:     "answer",
			TargetSection:  "composer",
			EvidencePostID: evidence,
			Priority:       0,
			Hint:           "Agent is waiting for your input",
		})
	case model.QuestStatusUserReview:
		entries = append(entries, ThreadActionEntrypoint{
			ActionType:    "resolve_review",
			TargetSection: "review_panel",
			Priority:      0,
			Hint:          "Review the deliverable",
		})
	case model.QuestStatusBlocked:
		evidence := findEvidencePost("system_escalation")
		entries = append(entries, ThreadActionEntrypoint{
			ActionType:     "resolve_blocked",
			TargetSection:  "blocked_panel",
			EvidencePostID: evidence,
			Priority:       0,
			Hint:           "Quest is blocked, needs human intervention",
		})
	}

	if !q.Status.IsTerminal() {
		entries = append(entries, ThreadActionEntrypoint{
			ActionType:    "add_comment",
			TargetSection: "composer",
			Priority:      1,
			Hint:          "Post a message to the thread",
		})
	}
	return entries
}

// getQuestThread returns the narrative thread (ThreadPosts) for a quest.
// This is the primary reading surface for v0.6: Feed → ThreadView.
// GET /api/quests/{id}/thread
func (s *Server) getQuestThread(w http.ResponseWriter, r *http.Request) {
	qid := r.PathValue("id")
	if qid == "" {
		writeErr(w, http.StatusBadRequest, "missing quest id")
		return
	}
	q, err := s.engine.GetQuest(qid)
	if err != nil {
		writeErr(w, http.StatusNotFound, err.Error())
		return
	}
	qs := fsstore.NewQuestStore(s.root)

	// Ensure root post exists for this quest.
	if _, err := qs.EnsureRootThreadPost(q); err != nil {
		writeErr(w, http.StatusInternalServerError, "ensure root thread post failed: "+err.Error())
		return
	}

	posts, err := qs.LoadThreadPosts(qid)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "load thread posts failed: "+err.Error())
		return
	}
	if posts == nil {
		posts = []fsstore.ThreadPost{}
	}

	// Build response with quest context for the ThreadView.
	loadAdv := func(id string) *fsstore.AdventurerFile {
		if id == "" {
			return nil
		}
		a, err := s.engine.GetAdventurer(id)
		if err != nil || a == nil {
			return nil
		}
		return a
	}

	resp := map[string]any{
		"ok": true,
		"quest": map[string]any{
			"id":         q.ID,
			"short_id":   q.ShortID,
			"query":      q.Query,
			"status":     q.Status,
			"created_by": q.CreatedBy,
			"warrior_id": q.WarriorID,
			"mage_id":    q.MageID,
			"created_at_ms": q.CreatedAtMs,
			"updated_at_ms": q.UpdatedAtMs,
		},
		"posts":              posts,
		"actors":             buildThreadActors(q, posts, loadAdv),
		"fanout_summary":     buildThreadFanoutSummary(q, qs),
		"action_entrypoints": buildThreadActionEntrypoints(q, posts),
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) getQuestArtifact(w http.ResponseWriter, r *http.Request) {
	qid := r.PathValue("id")
	artifactID := r.PathValue("artifact_id")
	if qid == "" || artifactID == "" {
		writeErr(w, http.StatusBadRequest, "missing quest id or artifact id")
		return
	}
	f, artifact, err := fsstore.NewQuestStore(s.root).OpenArtifact(qid, artifactID)
	if err != nil {
		writeErr(w, http.StatusNotFound, err.Error())
		return
	}
	defer f.Close()
	if artifact.MIME != "" {
		w.Header().Set("Content-Type", artifact.MIME)
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", artifact.Name))
	http.ServeContent(w, r, artifact.Name, time.UnixMilli(artifact.CreatedAtMs), f)
}

func (s *Server) getQuestSession(w http.ResponseWriter, r *http.Request) {
	qid := r.PathValue("id")
	sid := r.PathValue("sid")
	if qid == "" || sid == "" {
		writeErr(w, http.StatusBadRequest, "missing quest id or session id")
		return
	}
	limit := parseLimit(r, 200)
	rows, err := fsstore.NewQuestStore(s.root).ReadSessionRows(qid, sid, limit)
	if err != nil {
		writeErr(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "qid": qid, "sid": sid, "items": rows})
}

func (s *Server) getQuestSessionContext(w http.ResponseWriter, r *http.Request) {
	qid := r.PathValue("id")
	sid := r.PathValue("sid")
	if qid == "" || sid == "" {
		writeErr(w, http.StatusBadRequest, "missing quest id or session id")
		return
	}
	limit := parseLimit(r, 50)
	rows, err := fsstore.NewQuestStore(s.root).ReadContextPackRows(qid, sid, limit)
	if err != nil {
		writeErr(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "qid": qid, "sid": sid, "items": rows})
}

func (s *Server) getQuestLedgerAudit(w http.ResponseWriter, r *http.Request) {
	qid := r.PathValue("id")
	if qid == "" {
		writeErr(w, http.StatusBadRequest, "missing quest id")
		return
	}
	repair := false
	if raw := r.URL.Query().Get("repair"); raw != "" {
		repair, _ = strconv.ParseBool(raw)
	}
	audit, err := fsstore.NewQuestStore(s.root).AuditThreadLedger(qid, repair)
	if err != nil {
		if strings.Contains(err.Error(), "委托不存在") {
			writeErr(w, http.StatusNotFound, err.Error())
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": audit.OK, "audit": audit})
}

func parseLimit(r *http.Request, def int) int {
	limit := def
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil {
			limit = n
		}
	}
	if limit < 0 {
		limit = 0
	}
	if limit > 1000 {
		limit = 1000
	}
	return limit
}

func parseWorkspaceMode(raw string) (model.WorkspaceMode, error) {
	switch raw {
	case "", "auto":
		return "", nil
	case string(model.WorkspaceWorktree):
		return model.WorkspaceWorktree, nil
	case string(model.WorkspaceCopy):
		return model.WorkspaceCopy, nil
	case string(model.WorkspaceReadOnly):
		return model.WorkspaceReadOnly, nil
	default:
		return "", fmt.Errorf("workspace_mode 必须是 auto | worktree | copy | readonly")
	}
}

func parseQuestIntensity(raw string) (model.QuestIntensity, error) {
	switch raw {
	case "", string(model.QuestIntensityStandard):
		return model.QuestIntensityStandard, nil
	case string(model.QuestIntensityQuick):
		return model.QuestIntensityQuick, nil
	case string(model.QuestIntensityDeep):
		return model.QuestIntensityDeep, nil
	case string(model.QuestIntensityAdversarial):
		return model.QuestIntensityAdversarial, nil
	default:
		return "", fmt.Errorf("intensity 必须是 quick | standard | deep | adversarial")
	}
}

// ==================== createQuest ====================

const (
	maxQuestArtifactSize  = 16 << 20
	maxQuestArtifactCount = 8
)

type createQuestReq struct {
	Query              string                  `json:"query"`
	Mode               string                  `json:"mode"`        // run | check | design (v0.3.7+, 优先)
	Type               string                  `json:"type"`        // execute | design (legacy, mode 未设时 fallback)
	Intensity          string                  `json:"intensity"`   // quick | standard (legacy, mode 未设时 fallback)
	WorkDir            string                  `json:"work_dir"`    // 工作目录，空=用默认
	WorkingDir         string                  `json:"working_dir"` // 兼容 PRD/API 文档字段
	WorkspaceMode      string                  `json:"workspace_mode"`
	WarriorID          string                  `json:"warrior_id"`
	MageID             string                  `json:"mage_id"`
	ExecuteAgentID     string                  `json:"execute_agent_id"`
	ReviewAgentID      string                  `json:"review_agent_id"`
	AutoStart          bool                    `json:"auto_start"`
	WithDesignPhase    bool                    `json:"with_design_phase"`
	AutoSpawnExecute   bool                    `json:"auto_spawn_execute"`
	Connectors         []string                `json:"connectors"`
	Inputs             []fsstore.QuestArtifact `json:"inputs"`
	SourceQuestID      string                  `json:"source_quest_id"`
	GroupID            string                  `json:"group_id"`
	AcceptanceCriteria string                  `json:"acceptance_criteria"`
}

type createQuestUpload struct {
	Name string
	MIME string
	File multipart.File
}

func (u *createQuestUpload) Close() {
	if u != nil && u.File != nil {
		_ = u.File.Close()
	}
}

func (s *Server) createQuest(w http.ResponseWriter, r *http.Request) {
	req, uploads, err := s.readCreateQuestRequest(r)
	defer closeQuestUploads(uploads)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "body 解析失败: "+err.Error())
		return
	}
	if req.Query == "" {
		writeErr(w, http.StatusBadRequest, "query 不能为空")
		return
	}

	// Mode 优先：如果前端传了 mode，从 mode 推导 type/intensity
	questType := model.QuestType(req.Type)
	if req.Mode != "" {
		questType = model.ModeToQuestType(model.QuestMode(req.Mode))
		if req.Intensity == "" {
			if model.QuestMode(req.Mode) == model.ModeRun {
				req.Intensity = "quick"
			} else {
				req.Intensity = "standard"
			}
		}
		if model.QuestMode(req.Mode) == model.ModeDesign {
			req.WithDesignPhase = true
		}
	}
	if questType == "" {
		questType = model.QuestTypeExecute
	}

	workDir := req.WorkDir
	if workDir == "" {
		workDir = req.WorkingDir
	}
	workspaceMode, err := parseWorkspaceMode(req.WorkspaceMode)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	intensity, err := parseQuestIntensity(req.Intensity)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.ensureRunnableAdventurer(req.WarriorID, model.ClassWarrior); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.ensureRunnableAdventurer(req.MageID, model.ClassMage); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	// 默认冒险者：未指定时用全局配置的默认剑士/法师
	// agent-direct 模式下不强制选 adventurer
	warriorID := req.WarriorID
	mageID := req.MageID
	executeAgentID := req.ExecuteAgentID
	reviewAgentID := req.ReviewAgentID
	if cfg := s.engine.Config(); cfg != nil {
		if executeAgentID == "" && cfg.DefaultExecuteAgentID != "" {
			executeAgentID = cfg.DefaultExecuteAgentID
		}
		if reviewAgentID == "" && cfg.DefaultReviewAgentID != "" {
			reviewAgentID = cfg.DefaultReviewAgentID
		}
		if executeAgentID == "" {
			if warriorID == "" && cfg.DefaultWarriorID != "" {
				warriorID = cfg.DefaultWarriorID
			}
		}
		if reviewAgentID == "" {
			if mageID == "" && cfg.DefaultMageID != "" {
				mageID = cfg.DefaultMageID
			}
		}
	}

	initialInputs := req.Inputs
	if req.SourceQuestID != "" && len(initialInputs) > 0 {
		initialInputs = nil
	}
	workflowMode := model.WorkflowModeFromQuest(questType, req.Mode == "run")
	if req.Mode != "" {
		workflowMode = model.QuestModeToWorkflowMode(model.QuestMode(req.Mode))
	}
	if req.WithDesignPhase {
		workflowMode = model.WorkflowModeGoal
	}

	q, err := s.engine.CreateQuest(r.Context(), req.Query, questType, workDir, orchestrator.CreateQuestOptions{
		WarriorID:          warriorID,
		MageID:             mageID,
		ExecuteAgentID:     executeAgentID,
		ReviewAgentID:      reviewAgentID,
		WorkspaceMode:      workspaceMode,
		Intensity:          intensity,
		WorkflowMode:       workflowMode,
		RequireAgent:       true,
		Inputs:             initialInputs,
		GroupID:            req.GroupID,
		AcceptanceCriteria: req.AcceptanceCriteria,
		WithDesignPhase:    req.WithDesignPhase,
		SkipReview:         req.Mode == "run",
		AutoSpawnExecute:   req.AutoSpawnExecute,
		Connectors:         req.Connectors,
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if len(uploads) > 0 {
		qs := fsstore.NewQuestStore(s.root)
		for _, upload := range uploads {
			artifact, err := qs.SaveArtifact(q.ID, fsstore.ArtifactInput{
				Name:   upload.Name,
				MIME:   upload.MIME,
				Source: "upload",
				Reader: upload.File,
			})
			if err != nil {
				writeErr(w, http.StatusInternalServerError, "保存 artifact 失败: "+err.Error())
				return
			}
			q.Inputs = append(q.Inputs, artifact)
		}
		if err := qs.SaveQuest(q); err != nil {
			writeErr(w, http.StatusInternalServerError, "保存委托输入失败: "+err.Error())
			return
		}
	}
	if req.SourceQuestID != "" && len(req.Inputs) > 0 {
		qs := fsstore.NewQuestStore(s.root)
		for _, input := range req.Inputs {
			f, source, err := qs.OpenArtifact(req.SourceQuestID, input.ID)
			if err != nil {
				writeErr(w, http.StatusBadRequest, "克隆 artifact 失败: "+err.Error())
				return
			}
			artifact, err := qs.SaveArtifact(q.ID, fsstore.ArtifactInput{
				Name:   source.Name,
				MIME:   source.MIME,
				Source: "quest:" + req.SourceQuestID,
				Reader: f,
			})
			_ = f.Close()
			if err != nil {
				writeErr(w, http.StatusInternalServerError, "保存 artifact 失败: "+err.Error())
				return
			}
			q.Inputs = append(q.Inputs, artifact)
		}
		if err := qs.SaveQuest(q); err != nil {
			writeErr(w, http.StatusInternalServerError, "保存委托输入失败: "+err.Error())
			return
		}
	}

	// 自动启动
	if req.AutoStart {
		if err := s.engine.StartQuest(r.Context(), q.ID); err != nil {
			writeErr(w, http.StatusInternalServerError, "启动委托失败: "+err.Error())
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":       true,
		"qid":      q.ID,
		"short_id": q.ShortID,
		"quest":    q,
	})
}

func (s *Server) upgradeQuestWorkflowMode(w http.ResponseWriter, r *http.Request) {
	qid := r.PathValue("id")
	if qid == "" {
		writeErr(w, http.StatusBadRequest, "missing quest id")
		return
	}
	var req struct {
		WorkflowMode string `json:"workflow_mode"`
		Reason       string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "请求体格式错误: "+err.Error())
		return
	}
	q, err := s.engine.UpgradeWorkflowMode(r.Context(), qid, model.WorkflowMode(req.WorkflowMode), req.Reason)
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "委托不存在") {
			writeErr(w, http.StatusNotFound, msg)
			return
		}
		var upgradeErr *orchestrator.WorkflowModeUpgradeError
		if errors.As(err, &upgradeErr) {
			writeErr(w, http.StatusBadRequest, msg)
			return
		}
		writeErr(w, http.StatusInternalServerError, msg)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "qid": q.ID, "quest": s.questResponse(q)})
}

// spawnQuest 扇出一个独立委托（quest_spawn 平台工具的 HTTP 版本）。
// 比 createQuest 简单：只接受 query/group_id/work_dir，直接创建并启动 execute 委托。
// POST /api/quests/spawn
func (s *Server) spawnQuest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query            string   `json:"query"`
		GroupID          string   `json:"group_id"`
		WorkDir          string   `json:"work_dir"`
		ParentQuestID    string   `json:"parent_quest_id"`
		LeafID           string   `json:"leaf_id"`
		OwnershipScopes  []string `json:"ownership_scopes"`
		MergeStrategy    string   `json:"merge_strategy"`
		MergeOwnerLeafID string   `json:"merge_owner_leaf_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "请求体格式错误: "+err.Error())
		return
	}
	if req.Query == "" {
		writeErr(w, http.StatusBadRequest, "query 必填")
		return
	}

	contract := fsstore.FanoutContract{
		LeafID:           req.LeafID,
		OwnershipScopes:  req.OwnershipScopes,
		MergeStrategy:    req.MergeStrategy,
		MergeOwnerLeafID: req.MergeOwnerLeafID,
	}
	qid, err := s.engine.SpawnQuest(r.Context(), req.Query, req.GroupID, req.WorkDir, req.ParentQuestID, contract)
	if err != nil {
		var contractErr *fsstore.FanoutContractError
		if errors.As(err, &contractErr) {
			writeErr(w, http.StatusBadRequest, "spawn 失败: "+err.Error())
			return
		}
		writeErr(w, http.StatusInternalServerError, "spawn 失败: "+err.Error())
		return
	}
	q, _ := s.engine.GetQuest(qid)

	resp := map[string]any{
		"ok":       true,
		"qid":      qid,
		"group_id": req.GroupID,
		"message":  "委托已创建并启动",
	}
	if q != nil {
		resp["group_id"] = q.GroupID
		resp["leaf_id"] = q.FanoutLeafID
		resp["ownership_scopes"] = q.OwnershipScopes
		resp["merge_strategy"] = q.MergeStrategy
		resp["merge_owner_leaf_id"] = q.MergeOwnerLeafID
	}
	writeJSON(w, http.StatusOK, resp)
}

func formValue(values map[string][]string, key string) string {
	if values == nil || len(values[key]) == 0 {
		return ""
	}
	return values[key][0]
}

func closeQuestUploads(uploads []createQuestUpload) {
	for i := range uploads {
		uploads[i].Close()
	}
}

func (s *Server) readCreateQuestRequest(r *http.Request) (createQuestReq, []createQuestUpload, error) {
	if strings.HasPrefix(strings.ToLower(r.Header.Get("Content-Type")), "multipart/form-data") {
		return s.readMultipartCreateQuestRequest(r)
	}
	var req createQuestReq
	if err := readBodyLimited(r, &req); err != nil {
		return req, nil, err
	}
	return req, nil, nil
}

func (s *Server) readMultipartCreateQuestRequest(r *http.Request) (createQuestReq, []createQuestUpload, error) {
	var req createQuestReq
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		return req, nil, err
	}
	form := r.MultipartForm
	if form == nil {
		return req, nil, fmt.Errorf("multipart form 为空")
	}
	req.Query = formValue(form.Value, "query")
	req.Mode = formValue(form.Value, "mode")
	req.Type = formValue(form.Value, "type")
	req.Intensity = formValue(form.Value, "intensity")
	req.WorkDir = formValue(form.Value, "work_dir")
	req.WorkingDir = formValue(form.Value, "working_dir")
	req.WorkspaceMode = formValue(form.Value, "workspace_mode")
	req.WarriorID = formValue(form.Value, "warrior_id")
	req.MageID = formValue(form.Value, "mage_id")
	req.SourceQuestID = formValue(form.Value, "source_quest_id")
	if raw := formValue(form.Value, "auto_start"); raw != "" {
		req.AutoStart, _ = strconv.ParseBool(raw)
	}
	if raw := formValue(form.Value, "with_design_phase"); raw != "" {
		req.WithDesignPhase, _ = strconv.ParseBool(raw)
	}
	if raw := formValue(form.Value, "allow_quick_auto_complete"); raw != "" {
	}
	if raw := formValue(form.Value, "auto_spawn_execute"); raw != "" {
		req.AutoSpawnExecute, _ = strconv.ParseBool(raw)
	}
	if raw := formValue(form.Value, "inputs"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &req.Inputs); err != nil {
			return req, nil, fmt.Errorf("inputs 解析失败: %w", err)
		}
	}

	var uploads []createQuestUpload
	for _, key := range []string{"artifacts", "files", "images"} {
		for _, fh := range form.File[key] {
			if len(uploads) >= maxQuestArtifactCount {
				closeQuestUploads(uploads)
				return req, nil, fmt.Errorf("artifact 数量不能超过 %d", maxQuestArtifactCount)
			}
			upload, err := openCreateQuestUpload(fh)
			if err != nil {
				closeQuestUploads(uploads)
				return req, nil, err
			}
			uploads = append(uploads, upload)
		}
	}
	return req, uploads, nil
}

func openCreateQuestUpload(fh *multipart.FileHeader) (createQuestUpload, error) {
	if fh == nil {
		return createQuestUpload{}, fmt.Errorf("artifact file 为空")
	}
	if fh.Size > maxQuestArtifactSize {
		return createQuestUpload{}, fmt.Errorf("artifact %s 超过大小限制 %d bytes", fh.Filename, maxQuestArtifactSize)
	}
	file, err := fh.Open()
	if err != nil {
		return createQuestUpload{}, err
	}
	name := filepath.Base(strings.TrimSpace(fh.Filename))
	if name == "" {
		name = "artifact"
	}
	return createQuestUpload{
		Name: name,
		MIME: fh.Header.Get("Content-Type"),
		File: file,
	}, nil
}

type cleanupQuestWorkspacesReq struct {
	RetentionDays int   `json:"retention_days"`
	IncludeFailed bool  `json:"include_failed"`
	DryRun        *bool `json:"dry_run"`
}

func (s *Server) cleanupQuestWorkspaces(w http.ResponseWriter, r *http.Request) {
	var req cleanupQuestWorkspacesReq
	if bodyErr := readBodyLimited(r, &req); bodyErr != nil {
		fmt.Fprintf(os.Stderr, "[warn] 解析 cleanup 请求体失败: %v\n", bodyErr)
	}
	dryRun := true
	if req.DryRun != nil {
		dryRun = *req.DryRun
	}
	days := req.RetentionDays
	if days <= 0 {
		days = s.engine.Config().WorkspaceRetentionDays
	}
	includeFailed := req.IncludeFailed || s.engine.Config().AutoCleanupFailedQuests
	items, err := fsstore.NewQuestStore(s.root).CleanupExpiredWorkspaces(days, includeFailed, fsstore.NowMs(), dryRun)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":             true,
		"dry_run":        dryRun,
		"retention_days": days,
		"items":          items,
	})
}

// ==================== startQuest / stopQuest ====================

func (s *Server) startQuest(w http.ResponseWriter, r *http.Request) {
	qid := r.PathValue("id")
	if qid == "" {
		writeErr(w, http.StatusBadRequest, "missing quest id")
		return
	}
	if err := s.engine.StartQuest(r.Context(), qid); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	resp := map[string]any{"ok": true, "qid": qid}
	if q, err := s.engine.GetQuest(qid); err == nil {
		quest := s.questResponse(q)
		if q.Status == model.QuestStatusPending && quest["queued"] != true {
			quest["starting"] = true
		}
		resp["quest"] = quest
	}
	writeJSON(w, http.StatusOK, resp)
}

type questCommentReq struct {
	Comment       string `json:"comment"`
	Content       string `json:"content"`
	ParentReplyID string `json:"parent_reply_id"`
}

type questAnswerReq struct {
	QuestionID string `json:"question_id"`
	Answer     string `json:"answer"`
	Content    string `json:"content"`
	Source     string `json:"source"`
}

func (s *Server) addQuestComment(w http.ResponseWriter, r *http.Request) {
	qid := r.PathValue("id")
	if qid == "" {
		writeErr(w, http.StatusBadRequest, "missing quest id")
		return
	}
	var req questCommentReq
	if err := readBodyLimited(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "body 解析失败: "+err.Error())
		return
	}
	comment := req.Comment
	if comment == "" {
		comment = req.Content
	}
	if err := s.engine.AppendUserComment(qid, comment, req.ParentReplyID); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "qid": qid})
}

func (s *Server) addQuestAnswer(w http.ResponseWriter, r *http.Request) {
	qid := r.PathValue("id")
	if qid == "" {
		writeErr(w, http.StatusBadRequest, "missing quest id")
		return
	}
	var req questAnswerReq
	if err := readBodyLimited(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "body 解析失败: "+err.Error())
		return
	}
	answer := req.Answer
	if answer == "" {
		answer = req.Content
	}
	rec, err := s.engine.AppendUserAnswer(qid, req.QuestionID, answer, req.Source)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "qid": qid, "answer": rec})
}

func (s *Server) stopQuest(w http.ResponseWriter, r *http.Request) {
	qid := r.PathValue("id")
	if qid == "" {
		writeErr(w, http.StatusBadRequest, "missing quest id")
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	if bodyErr := readBodyLimited(r, &req); bodyErr != nil {
		// body 解析失败，使用零值（可选字段），仅 warn
		fmt.Fprintf(os.Stderr, "[warn] 解析请求体失败: %v\n", bodyErr)
	}
	if err := s.engine.StopQuest(qid, req.Reason); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "qid": qid})
}

type resolveBlockedReq struct {
	Action             string `json:"action"`
	Comment            string `json:"comment"`
	AddTurns           int    `json:"add_turns"`
	AddDurationMinutes int    `json:"add_duration_minutes"`
	AddDurationMs      int64  `json:"add_duration_ms"`
}

type recoverAgentReq struct {
	Action             string `json:"action"`
	Comment            string `json:"comment"`
	AddTurns           int    `json:"add_turns"`
	AddDurationMinutes int    `json:"add_duration_minutes"`
}

func (s *Server) resolveBlockedQuest(w http.ResponseWriter, r *http.Request) {
	qid := r.PathValue("id")
	if qid == "" {
		writeErr(w, http.StatusBadRequest, "missing quest id")
		return
	}
	var req resolveBlockedReq
	if err := readBodyLimited(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "body 解析失败: "+err.Error())
		return
	}
	if req.AddDurationMinutes == 0 && req.AddDurationMs > 0 {
		req.AddDurationMinutes = int((req.AddDurationMs + 60*1000 - 1) / (60 * 1000))
	}
	if err := s.engine.ResolveBlockedQuest(r.Context(), qid, orchestrator.ResumeBlockedOptions{
		Action:             req.Action,
		Comment:            req.Comment,
		AddTurns:           req.AddTurns,
		AddDurationMinutes: req.AddDurationMinutes,
	}); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":     true,
		"qid":    qid,
		"action": req.Action,
	})
}

func (s *Server) recoverAgentQuest(w http.ResponseWriter, r *http.Request) {
	qid := r.PathValue("id")
	if qid == "" {
		writeErr(w, http.StatusBadRequest, "missing quest id")
		return
	}
	var req recoverAgentReq
	if err := readBodyLimited(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "body 解析失败: "+err.Error())
		return
	}
	if err := s.engine.RecoverBlockedQuest(r.Context(), qid, req.Action, orchestrator.ResumeBlockedOptions{
		Comment:            req.Comment,
		AddTurns:           req.AddTurns,
		AddDurationMinutes: req.AddDurationMinutes,
	}); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "qid": qid, "action": req.Action})
}

// ==================== resolveUserReview ====================

type resolveUserReviewReq struct {
	Verdict string `json:"verdict"` // pass | request_changes | reject
	Comment string `json:"comment"`
	Apply   bool   `json:"apply"`
}

func (s *Server) resolveUserReview(w http.ResponseWriter, r *http.Request) {
	qid := r.PathValue("id")
	if qid == "" {
		writeErr(w, http.StatusBadRequest, "missing quest id")
		return
	}
	var req resolveUserReviewReq
	if err := readBodyLimited(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "body 解析失败: "+err.Error())
		return
	}
	var verdict model.QuestVerdict
	switch req.Verdict {
	case "pass", "Pass", "PASS":
		verdict = model.VerdictPass
	case "request_changes", "rework", "changes":
		verdict = model.VerdictRequestChange
	case "reject", "Reject", "REJECT":
		verdict = model.VerdictReject
	default:
		writeErr(w, http.StatusBadRequest, "verdict 必须是 pass | request_changes | reject")
		return
	}
	if err := s.engine.ResolveUserReview(r.Context(), qid, verdict, req.Comment); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	resp := map[string]any{
		"ok":      true,
		"qid":     qid,
		"verdict": verdict,
	}
	if verdict == model.VerdictPass && req.Apply {
		warnings, err := s.engine.ApplyQuest(r.Context(), qid, false)
		if err != nil {
			writeErrWith(w, http.StatusBadRequest, "用户终审已通过，但 apply 失败: "+err.Error(), map[string]any{"warnings": warnings})
			return
		}
		resp["applied"] = true
		resp["warnings"] = warnings
	}
	if q, err := s.engine.GetQuest(qid); err == nil {
		resp["quest"] = s.questResponse(q)
	}
	writeJSON(w, http.StatusOK, resp)
}

// getQuestTrace returns a unified timeline merging semantic events + native tool evidence.
func (s *Server) getQuestTrace(w http.ResponseWriter, r *http.Request) {
	qid := r.PathValue("id")
	if qid == "" {
		writeErr(w, http.StatusBadRequest, "missing quest id")
		return
	}

	qs := fsstore.NewQuestStore(s.root)
	meta, err := qs.LoadQuest(qid)
	if err != nil {
		writeErr(w, http.StatusNotFound, err.Error())
		return
	}

	type traceEntry struct {
		ID        int64       `json:"id,omitempty"`
		Kind      string      `json:"kind"` // "event", "tool", or "message"
		Timestamp int64       `json:"ts"`
		Type      string      `json:"type,omitempty"`
		Payload   interface{} `json:"payload,omitempty"`
		SID       string      `json:"sid,omitempty"`
		ToolName  string      `json:"tool_name,omitempty"`
		ToolArgs  string      `json:"tool_args,omitempty"`
		Status    string      `json:"status,omitempty"`
		Result    string      `json:"result,omitempty"`
		Content   string      `json:"content,omitempty"`
		Turn      int         `json:"turn,omitempty"`
		Phase     int         `json:"phase,omitempty"`
		Round     int         `json:"round"`
		Meta      any         `json:"meta,omitempty"`
	}

	var entries []traceEntry
	scope := r.URL.Query().Get("scope")
	if scope == "" {
		scope = "current"
	}
	if scope != "current" && scope != "all" {
		writeErr(w, http.StatusBadRequest, "scope 必须是 current 或 all")
		return
	}
	roundFilter := -1
	if raw := r.URL.Query().Get("round"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			writeErr(w, http.StatusBadRequest, "round 必须是非负整数")
			return
		}
		roundFilter = n
	}
	limit := parseLimit(r, 200)
	if limit == 0 {
		limit = 200
	}

	// 1. Semantic events from events.jsonl
	events, _ := qs.ReadEvents(qid, limit)
	for _, ev := range events {
		entries = append(entries, traceEntry{
			ID:        ev.ID,
			Kind:      "event",
			Timestamp: ev.Timestamp,
			Type:      ev.Type,
			Payload:   ev.Payload,
			SID:       ev.SessionID,
		})
	}

	// 2. Native tool evidence + assistant messages from session rows
	reworkCount := 0
	if meta.ReworkCount > 0 {
		reworkCount = meta.ReworkCount
	}
	sids := traceSessionIDs(qs, qid, reworkCount, scope, roundFilter)
	for _, sid := range sids {
		round := traceRoundFromSID(sid)
		rows, err := qs.ReadSessionRows(qid, sid, 0)
		if err != nil {
			continue
		}
		for _, row := range rows {
			switch row.Kind {
			case "native_tool_call_observed":
				status := ""
				result := ""
				if row.Meta != nil {
					if s, ok := row.Meta["status"]; ok {
						status = fmt.Sprintf("%v", s)
					}
					if r, ok := row.Meta["result"]; ok {
						if rs, ok := r.(string); ok {
							result = rs
						} else {
							if b, err := json.Marshal(r); err == nil {
								result = string(b)
							}
						}
					}
				}
				entries = append(entries, traceEntry{
					Kind:      "tool",
					Timestamp: row.Timestamp,
					SID:       sid,
					Round:     round,
					ToolName:  row.ToolName,
					ToolArgs:  row.ToolArgs,
					Status:    status,
					Result:    result,
				})
			case "message":
				if row.Role != "assistant" {
					continue
				}
				entries = append(entries, traceEntry{
					Kind:      "message",
					Timestamp: row.Timestamp,
					SID:       sid,
					Round:     round,
					Content:   row.Content,
					Turn:      row.Seq,
					Phase:     row.Phase,
					Meta:      row.Meta,
				})
			case "tool_result":
				entries = append(entries, traceEntry{
					Kind:      "tool_result",
					Timestamp: row.Timestamp,
					SID:       sid,
					Round:     round,
					ToolName:  row.ToolName,
					Result:    row.Content,
					Content:   row.Content,
					Turn:      row.Seq,
					Phase:     row.Phase,
					Meta:      row.Meta,
				})
			case "user_comment":
				count := 0
				if row.Meta != nil {
					if c, ok := row.Meta["count"]; ok {
						if ci, ok := c.(float64); ok {
							count = int(ci)
						}
					}
				}
				entries = append(entries, traceEntry{
					Kind:      "comment",
					Timestamp: row.Timestamp,
					SID:       sid,
					Round:     round,
					Content:   row.Content,
					Turn:      row.Seq,
					Phase:     row.Phase,
					Meta:      map[string]any{"count": count},
				})
			case "context_pack":
				// 上下文注入：包含结构化 blocks 和渲染后的原始文本
				meta := make(map[string]any)
				if row.Meta != nil {
					if summary, ok := row.Meta["context_pack"]; ok {
						meta["summary"] = summary
					}
					if blocks, ok := row.Meta["context_pack_blocks"]; ok {
						meta["blocks"] = blocks
					}
				}
				entries = append(entries, traceEntry{
					Kind:      "context_pack",
					Timestamp: row.Timestamp,
					SID:       sid,
					Round:     round,
					Content:   row.Content,
					Turn:      row.Seq,
					Phase:     row.Phase,
					Meta:      meta,
				})
			}
		}
	}

	// 3. Sort by timestamp
	for i := 1; i < len(entries); i++ {
		for j := i; j > 0 && entries[j].Timestamp < entries[j-1].Timestamp; j-- {
			entries[j], entries[j-1] = entries[j-1], entries[j]
		}
	}
	truncated := false
	if len(entries) > limit {
		truncated = true
		entries = entries[:limit]
	}

	resp := map[string]any{
		"ok":          true,
		"qid":         qid,
		"scope":       scope,
		"items":       entries,
		"truncated":   truncated,
		"next_cursor": "",
	}
	if roundFilter >= 0 {
		resp["round"] = roundFilter
	}
	writeJSON(w, http.StatusOK, resp)
}

var traceSIDPattern = regexp.MustCompile(`^(warrior|mage)(?:_[a-z]+)*(?:_review)?_([0-9]+)$`)

func traceSessionIDs(qs *fsstore.QuestStore, qid string, currentRound int, scope string, roundFilter int) []string {
	if scope == "all" {
		all, _ := qs.ListSessionIDs(qid)
		out := make([]string, 0, len(all))
		for _, sid := range all {
			round := traceRoundFromSID(sid)
			if roundFilter >= 0 && round != roundFilter {
				continue
			}
			out = append(out, sid)
		}
		if len(out) > 0 {
			return out
		}
	}
	if roundFilter >= 0 {
		currentRound = roundFilter
	}
	return []string{
		fmt.Sprintf("warrior_%d", currentRound),
		fmt.Sprintf("mage_%d", currentRound),
	}
}

func traceRoundFromSID(sid string) int {
	m := traceSIDPattern.FindStringSubmatch(sid)
	if len(m) != 3 {
		return 0
	}
	n, err := strconv.Atoi(m[2])
	if err != nil {
		return 0
	}
	return n
}
