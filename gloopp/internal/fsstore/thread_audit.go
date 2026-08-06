package fsstore

import (
	"fmt"

	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

type ThreadLedgerAudit struct {
	QuestID   string                 `json:"quest_id"`
	OK        bool                   `json:"ok"`
	Findings  []ThreadLedgerFinding  `json:"findings,omitempty"`
	Repaired  []ThreadLedgerRepair   `json:"repaired,omitempty"`
	PostCount int                    `json:"post_count"`
	Meta      map[string]interface{} `json:"meta,omitempty"`
}

type ThreadLedgerFinding struct {
	Code     string `json:"code"`
	PostID   string `json:"post_id,omitempty"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

type ThreadLedgerRepair struct {
	Code   string `json:"code"`
	PostID string `json:"post_id,omitempty"`
	Detail string `json:"detail,omitempty"`
}

func (qs *QuestStore) AuditThreadLedger(qid string, repair bool) (*ThreadLedgerAudit, error) {
	q, err := qs.LoadQuest(qid)
	if err != nil {
		return nil, err
	}
	report, err := qs.auditThreadLedgerOnce(q)
	if err != nil {
		return nil, err
	}
	if repair {
		repaired, err := qs.repairThreadLedger(q, report)
		if err != nil {
			return nil, err
		}
		if len(repaired) > 0 {
			report, err = qs.auditThreadLedgerOnce(q)
			if err != nil {
				return nil, err
			}
			report.Repaired = repaired
		}
	}
	report.OK = len(report.Findings) == 0
	return report, nil
}

func (qs *QuestStore) auditThreadLedgerOnce(q *QuestMeta) (*ThreadLedgerAudit, error) {
	posts, err := qs.LoadThreadPosts(q.ID)
	if err != nil {
		return nil, err
	}
	report := &ThreadLedgerAudit{
		QuestID:   q.ID,
		PostCount: len(posts),
		Meta: map[string]interface{}{
			"status":        q.Status,
			"workflow_mode": q.WorkflowMode,
		},
	}
	seen := map[string]int{}
	hasRoot := false
	hasDecision := false
	for _, post := range posts {
		if post.PostID == "" {
			report.Findings = append(report.Findings, ThreadLedgerFinding{
				Code:     "missing_post_id",
				Severity: "required",
				Message:  "thread post is missing post_id",
			})
		}
		seen[post.PostID]++
		if post.PostID == RootPostID(q.ID) {
			hasRoot = true
		}
		if post.Kind == "decision_note" {
			hasDecision = true
		}
		if post.ThreadID != "" && post.ThreadID != q.ID {
			report.Findings = append(report.Findings, ThreadLedgerFinding{
				Code:     "wrong_thread_id",
				PostID:   post.PostID,
				Severity: "required",
				Message:  fmt.Sprintf("thread_id %s does not match quest %s", post.ThreadID, q.ID),
			})
		}
		if post.RootPostID != "" && post.RootPostID != RootPostID(q.ID) {
			report.Findings = append(report.Findings, ThreadLedgerFinding{
				Code:     "wrong_root_post_id",
				PostID:   post.PostID,
				Severity: "required",
				Message:  fmt.Sprintf("root_post_id %s does not match expected %s", post.RootPostID, RootPostID(q.ID)),
			})
		}
		if err := validateThreadPostAuthority(&post); err != nil {
			report.Findings = append(report.Findings, ThreadLedgerFinding{
				Code:     "invalid_authority",
				PostID:   post.PostID,
				Severity: "required",
				Message:  err.Error(),
			})
		}
	}
	for postID, count := range seen {
		if postID == "" || count < 2 {
			continue
		}
		report.Findings = append(report.Findings, ThreadLedgerFinding{
			Code:     "duplicate_post_id",
			PostID:   postID,
			Severity: "required",
			Message:  fmt.Sprintf("post_id %s appears %d times", postID, count),
		})
	}
	if !hasRoot {
		report.Findings = append(report.Findings, ThreadLedgerFinding{
			Code:     "missing_root_post",
			PostID:   RootPostID(q.ID),
			Severity: "required",
			Message:  "thread ledger is missing root post",
		})
	}
	if isTerminalStatus(q.Status) && q.PinnedOutcomeSummary == "" {
		report.Findings = append(report.Findings, ThreadLedgerFinding{
			Code:     "missing_pinned_outcome",
			Severity: "required",
			Message:  "terminal quest is missing pinned_outcome_summary",
		})
	}
	if isTerminalStatus(q.Status) && !hasDecision {
		report.Findings = append(report.Findings, ThreadLedgerFinding{
			Code:     "missing_decision_note",
			Severity: "required",
			Message:  "terminal quest is missing decision_note thread post",
		})
	}
	report.OK = len(report.Findings) == 0
	return report, nil
}

func (qs *QuestStore) repairThreadLedger(q *QuestMeta, report *ThreadLedgerAudit) ([]ThreadLedgerRepair, error) {
	var repaired []ThreadLedgerRepair
	needsRoot := false
	needsPinned := false
	needsDecision := false
	for _, finding := range report.Findings {
		switch finding.Code {
		case "missing_root_post":
			needsRoot = true
		case "missing_pinned_outcome":
			needsPinned = true
		case "missing_decision_note":
			needsDecision = true
		}
	}
	if needsRoot {
		post, err := qs.EnsureRootThreadPost(q)
		if err != nil {
			return repaired, err
		}
		repaired = append(repaired, ThreadLedgerRepair{Code: "missing_root_post", PostID: post.PostID, Detail: "created root post"})
	}
	if needsPinned {
		q.PinnedOutcomeSummary = pinnedOutcomeSummary(q)
		if err := qs.SaveQuest(q); err != nil {
			return repaired, err
		}
		repaired = append(repaired, ThreadLedgerRepair{Code: "missing_pinned_outcome", Detail: "updated quest pinned_outcome_summary"})
	}
	if needsDecision {
		content := q.PinnedOutcomeSummary
		if content == "" {
			content = pinnedOutcomeSummary(q)
		}
		projection := systemProjection{
			PostID:     terminalProjectionPostID(q),
			Kind:       "decision_note",
			Content:    content,
			CausalRefs: []string{RootPostID(q.ID)},
			ArtifactRefs: []ThreadArtifactRef{{
				ID:           "decision_" + q.ID,
				ArtifactType: model.ArtifactTypeDecisionNote,
				Ref:          "pinned_outcome_summary",
				Role:         model.PostRoleSystem,
			}},
			CreatedAt: terminalProjectionTime(q),
		}
		if err := qs.appendSystemThreadPost(q, projection); err != nil {
			return repaired, err
		}
		repaired = append(repaired, ThreadLedgerRepair{Code: "missing_decision_note", PostID: projection.PostID, Detail: "created decision_note post"})
	}
	return repaired, nil
}

func isTerminalStatus(status model.QuestStatus) bool {
	return status == model.QuestStatusSuccess ||
		status == model.QuestStatusFailed ||
		status == model.QuestStatusCancelled
}
