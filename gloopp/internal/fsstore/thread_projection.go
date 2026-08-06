package fsstore

import (
	"fmt"
	"strings"

	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

// latestQuestEventIDBestEffort returns the highest event ID currently written for
// this quest. It is a best-effort attribution anchor for system state-transition
// posts — NOT a guarantee that the returned event triggered the state change.
// If the caller wrote an unrelated event just before SaveQuest, this helper will
// misattribute. Returns 0 when no events exist (legacy / fallback).
func (qs *QuestStore) latestQuestEventIDBestEffort(qid string) int64 {
	rows, err := qs.ReadEvents(qid, 0)
	if err != nil || len(rows) == 0 {
		return 0
	}
	maxID := int64(0)
	for _, row := range rows {
		if row.ID > maxID {
			maxID = row.ID
		}
	}
	return maxID
}

func (qs *QuestStore) prepareThreadStateTransitionProjection(prev, next *QuestMeta) ([]systemProjection, error) {
	if qs == nil || next == nil || strings.TrimSpace(next.ID) == "" {
		return nil, nil
	}
	if prev == nil {
		return nil, nil
	}
	projectRework := shouldProjectRework(prev, next)
	projectTerminal := shouldProjectTerminalOutcome(prev, next)
	projectBlocked := shouldProjectBlockedEscalation(prev, next)
	if !projectRework && !projectTerminal && !projectBlocked {
		return nil, nil
	}
	latestEventID := qs.latestQuestEventIDBestEffort(next.ID)
	var projections []systemProjection
	if projectRework {
		projections = append(projections, systemProjection{
			PostID:        fmt.Sprintf("sys_rework_%s_%d", next.ID, next.ReworkCount),
			Kind:          "system_rework",
			Content:       reworkProjectionContent(prev, next),
			CausalRefs:    reworkCausalRefs(next),
			CreatedAt:     next.UpdatedAtMs,
			SourceEventID: latestEventID,
		})
	}
	if projectTerminal {
		outcome := pinnedOutcomeSummary(next)
		next.PinnedOutcomeSummary = outcome
		projections = append(projections, systemProjection{
			PostID:        terminalProjectionPostID(next),
			Kind:          "decision_note",
			Content:       outcome,
			CausalRefs:    []string{RootPostID(next.ID)},
			CreatedAt:     terminalProjectionTime(next),
			SourceEventID: latestEventID,
			ArtifactRefs: []ThreadArtifactRef{{
				ID:           "decision_" + next.ID,
				ArtifactType: model.ArtifactTypeDecisionNote,
				Ref:          "pinned_outcome_summary",
				Role:         model.PostRoleSystem,
			}},
		})
	}
	if projectBlocked {
		projections = append(projections, systemProjection{
			PostID:        fmt.Sprintf("sys_blocked_%s_%d", next.ID, next.UpdatedAtMs),
			Kind:          "system_escalation",
			Content:       blockedProjectionContent(next),
			CausalRefs:    []string{RootPostID(next.ID)},
			CreatedAt:     next.UpdatedAtMs,
			SourceEventID: latestEventID,
		})
	}
	return projections, nil
}

func (qs *QuestStore) appendThreadStateProjections(q *QuestMeta, projections []systemProjection) error {
	if len(projections) == 0 {
		return nil
	}
	if _, err := qs.EnsureRootThreadPost(q); err != nil {
		return err
	}
	for _, projection := range projections {
		if err := qs.appendSystemThreadPost(q, projection); err != nil {
			return err
		}
	}
	return nil
}

type systemProjection struct {
	PostID        string
	Kind          string
	Content       string
	CausalRefs    []string
	ArtifactRefs  []ThreadArtifactRef
	CreatedAt     int64
	SourceEventID int64
}

func (qs *QuestStore) appendSystemThreadPost(q *QuestMeta, p systemProjection) error {
	if strings.TrimSpace(p.Content) == "" {
		return nil
	}
	_, err := qs.AppendThreadPost(q.ID, AppendThreadPostOptions{
		PostID:         p.PostID,
		ThreadID:       q.ID,
		ParentReplyID:  RootPostID(q.ID),
		RootPostID:     RootPostID(q.ID),
		CausalRefs:     p.CausalRefs,
		AuthorIdentity: "system",
		AuthorRole:     model.PostRoleSystem,
		ArtifactRefs:   p.ArtifactRefs,
		Kind:           p.Kind,
		Content:        p.Content,
		CreatedAtMs:    p.CreatedAt,
		SourceEventID:  p.SourceEventID,
	})
	return err
}

func shouldProjectRework(prev, next *QuestMeta) bool {
	return prev.Status != model.QuestStatusRunning &&
		next.Status == model.QuestStatusRunning &&
		next.ReworkCount > prev.ReworkCount
}

func reworkProjectionContent(prev, next *QuestMeta) string {
	source := "Checker"
	if prev.Status == model.QuestStatusUserReview {
		source = "Human"
	}
	hints := strings.TrimSpace(next.ReviewHints)
	if hints == "" {
		hints = "返工要求已记录。"
	}
	return fmt.Sprintf("%s requested rework. Round %d/%d. %s", source, next.ReworkCount, next.MaxRework, hints)
}

func reworkCausalRefs(q *QuestMeta) []string {
	refs := []string{RootPostID(q.ID)}
	if q.ReviewHints != "" {
		refs = append(refs, "review_hints")
	}
	return refs
}

func shouldProjectTerminalOutcome(prev, next *QuestMeta) bool {
	if prev.Status == next.Status {
		return false
	}
	return next.Status == model.QuestStatusSuccess ||
		next.Status == model.QuestStatusFailed ||
		next.Status == model.QuestStatusCancelled
}

func shouldProjectBlockedEscalation(prev, next *QuestMeta) bool {
	return prev.Status != model.QuestStatusBlocked && next.Status == model.QuestStatusBlocked
}

func blockedProjectionContent(q *QuestMeta) string {
	reason := strings.TrimSpace(q.BlockedReason)
	if reason == "" {
		reason = strings.TrimSpace(q.BlockedReasonCode)
	}
	if reason == "" {
		reason = "需要人工介入。"
	}
	if q.BlockedReasonCode != "" && !strings.Contains(reason, q.BlockedReasonCode) {
		return fmt.Sprintf("Escalation: blocked (%s). %s", q.BlockedReasonCode, reason)
	}
	return "Escalation: blocked. " + reason
}

func terminalProjectionPostID(q *QuestMeta) string {
	when := terminalProjectionTime(q)
	if when == 0 {
		when = q.UpdatedAtMs
	}
	return fmt.Sprintf("sys_outcome_%s_%s_%d", q.ID, q.Status, when)
}

func terminalProjectionTime(q *QuestMeta) int64 {
	if q.CompletedAtMs > 0 {
		return q.CompletedAtMs
	}
	if q.UpdatedAtMs > 0 {
		return q.UpdatedAtMs
	}
	return NowMs()
}

func pinnedOutcomeSummary(q *QuestMeta) string {
	status := string(q.Status)
	switch q.Status {
	case model.QuestStatusSuccess:
		status = "success"
	case model.QuestStatusFailed:
		status = "failed"
	case model.QuestStatusCancelled:
		status = "cancelled"
	}
	var parts []string
	parts = append(parts, "Outcome: "+status)
	if q.FinalVerdict != "" {
		parts = append(parts, "verdict="+string(q.FinalVerdict))
	}
	if strings.TrimSpace(q.FinalComment) != "" {
		parts = append(parts, strings.TrimSpace(q.FinalComment))
	} else if strings.TrimSpace(q.WarriorSummary) != "" {
		parts = append(parts, strings.TrimSpace(q.WarriorSummary))
	}
	if q.Applied {
		parts = append(parts, "applied=true")
	}
	if q.FinalizedBy != "" {
		parts = append(parts, "finalized_by="+q.FinalizedBy)
	}
	return strings.Join(parts, " | ")
}

// appendFanoutLeafDoneNotification writes a fanout_leaf_done post on the root
// quest's thread when a leaf quest reaches a terminal state. Idempotent via
// stable post_id = fanout_leaf_done_{leaf_qid}.
func (qs *QuestStore) appendFanoutLeafDoneNotification(leaf *QuestMeta) error {
	if leaf == nil {
		return nil
	}
	if leaf.GroupID == "" || leaf.FanoutLeafID == "" {
		return nil
	}
	if !isTerminalStatus(leaf.Status) {
		return nil
	}
	root, found := qs.FindFanoutRoot(leaf.GroupID)
	if !found {
		return nil
	}

	postID := fmt.Sprintf("fanout_leaf_done_%s", leaf.ID)

	// Check if already written (idempotent).
	existingPosts, err := qs.LoadThreadPosts(root.ID)
	if err == nil {
		for _, p := range existingPosts {
			if p.PostID == postID {
				return nil
			}
		}
	}

	// Build causal ref to leaf's decision_note if available.
	leafPosts, _ := qs.LoadThreadPosts(leaf.ID)
	var decisionNoteRef string
	for _, p := range leafPosts {
		if p.Kind == "decision_note" {
			decisionNoteRef = p.PostID
			break
		}
	}
	causalRefs := []string{RootPostID(root.ID)}
	if decisionNoteRef != "" {
		causalRefs = append(causalRefs, decisionNoteRef)
	}

	outcome := pinnedOutcomeSummary(leaf)
	content := fmt.Sprintf("Leaf #%s (%s) terminal. %s", leaf.ShortID, leaf.FanoutLeafID, outcome)

	if _, err := qs.EnsureRootThreadPost(root); err != nil {
		return err
	}
	_, err = qs.AppendThreadPost(root.ID, AppendThreadPostOptions{
		PostID:         postID,
		ThreadID:       root.ID,
		ParentReplyID:  RootPostID(root.ID),
		RootPostID:     RootPostID(root.ID),
		CausalRefs:     causalRefs,
		AuthorIdentity: "system",
		AuthorRole:     model.PostRoleSystem,
		Kind:           "fanout_leaf_done",
		Content:        content,
		CreatedAtMs:    leaf.UpdatedAtMs,
	})
	return err
}

// AppendFanoutSummarySnapshot writes a fanout_summary ThreadPost on the root thread.
// It is only called on semantic state changes (leaf spawned / leaf terminal / merge decision),
// not on every update. The seq is derived from existing summary post count.
func (qs *QuestStore) AppendFanoutSummarySnapshot(groupID string) error {
	if strings.TrimSpace(groupID) == "" {
		return nil
	}
	root, found := qs.FindFanoutRoot(groupID)
	if !found {
		return nil
	}
	leaves := qs.ListFanoutGroupLeaves(groupID)
	if len(leaves) == 0 {
		return nil
	}

	// Count existing summary posts to determine seq.
	existingPosts, err := qs.LoadThreadPosts(root.ID)
	if err != nil {
		return err
	}
	seq := int64(0)
	var causalRefs []string
	for _, p := range existingPosts {
		if p.Kind == "fanout_summary" {
			seq++
		}
		// Collect lifecycle post refs for causal chain.
		if p.Kind == "fanout_leaf_spawned" || p.Kind == "fanout_leaf_done" || p.Kind == "fanout_merge_decision" || p.Kind == "fanout_group_opened" {
			causalRefs = append(causalRefs, p.PostID)
		}
	}
	seq++

	// Build summary text from leaf states.
	var total, running, terminal, blocked, waiting int
	for _, leaf := range leaves {
		total++
		switch leaf.Status {
		case model.QuestStatusRunning, model.QuestStatusPending:
			running++
		case model.QuestStatusSuccess, model.QuestStatusFailed, model.QuestStatusCancelled:
			terminal++
		case model.QuestStatusBlocked:
			blocked++
		case model.QuestStatusWaitingInput:
			waiting++
		}
	}
	allDone := terminal == total && total > 0

	var parts []string
	parts = append(parts, fmt.Sprintf("Fanout group %s status: %d leaves", groupID, total))
	if allDone {
		parts = append(parts, "all terminal")
	} else {
		if running > 0 {
			parts = append(parts, fmt.Sprintf("%d running", running))
		}
		if terminal > 0 {
			parts = append(parts, fmt.Sprintf("%d done", terminal))
		}
		if blocked > 0 {
			parts = append(parts, fmt.Sprintf("%d blocked", blocked))
		}
		if waiting > 0 {
			parts = append(parts, fmt.Sprintf("%d waiting_input", waiting))
		}
	}
	if root.MergeStrategy != "" {
		parts = append(parts, "merge="+string(root.MergeStrategy))
	}
	if root.MergeOwnerLeafID != "" {
		parts = append(parts, "owner_leaf="+root.MergeOwnerLeafID)
	}

	// Ensure root post exists.
	if _, err := qs.EnsureRootThreadPost(root); err != nil {
		return err
	}

	causalRefs = append([]string{RootPostID(root.ID)}, causalRefs...)
	postID := fmt.Sprintf("sys_fanout_summary_%s_%d", groupID, seq)
	content := strings.Join(parts, " | ")

	_, err = qs.AppendThreadPost(root.ID, AppendThreadPostOptions{
		PostID:         postID,
		ThreadID:       root.ID,
		ParentReplyID:  RootPostID(root.ID),
		RootPostID:     RootPostID(root.ID),
		CausalRefs:     causalRefs,
		AuthorIdentity: "system",
		AuthorRole:     model.PostRoleSystem,
		Kind:           "fanout_summary",
		Content:        content,
		CreatedAtMs:    NowMs(),
	})
	return err
}

