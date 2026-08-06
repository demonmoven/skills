package fsstore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

type MakerReport struct {
	SchemaVersion string              `json:"schema_version"`
	ReportID      string              `json:"report_id"`
	QuestID       string              `json:"quest_id"`
	SessionID     string              `json:"session_id,omitempty"`
	Phase         int                 `json:"phase"`
	Verdict       string              `json:"verdict"`
	Summary       string              `json:"summary"`
	Artifacts     []string            `json:"artifacts,omitempty"`
	Changeset     string              `json:"changeset,omitempty"`
	Evidence      []string            `json:"evidence,omitempty"`
	Impact        model.ImpactSummary `json:"impact,omitempty"`
	OpenQuestions []string            `json:"open_questions,omitempty"`
	Transition    bool                `json:"transition,omitempty"`
	SourceEventID int64               `json:"source_event_id,omitempty"`
	CreatedAtMs   int64               `json:"created_at_ms"`
}

type EvidenceRef struct {
	ID          string `json:"id"`
	Kind        string `json:"kind"`
	TrustTier   string `json:"trust_tier"`
	Source      string `json:"source"`
	CommandID   string `json:"command_id,omitempty"`
	ExitCode    int    `json:"exit_code,omitempty"`
	DurationMs  int64  `json:"duration_ms,omitempty"`
	TimedOut    bool   `json:"timed_out,omitempty"`
	OutputPath  string `json:"output_path,omitempty"`
	Snippet     string `json:"snippet,omitempty"`
	CreatedAtMs int64  `json:"created_at_ms"`
}

type ReviewReport struct {
	SchemaVersion    string             `json:"schema_version"`
	ReportID         string             `json:"report_id"`
	QuestID          string             `json:"quest_id"`
	SessionID        string             `json:"session_id,omitempty"`
	Phase            int                `json:"phase"`
	Verdict          model.QuestVerdict `json:"verdict"`
	Score            int                `json:"score,omitempty"`
	CheckedAgainst   []string           `json:"checked_against"`
	EvidenceRefs     []EvidenceRef      `json:"evidence_refs,omitempty"`
	Findings         ReviewFindings     `json:"findings"`
	RequiredChanges  []string           `json:"required_changes,omitempty"`
	ResidualRisks    []string           `json:"residual_risks,omitempty"`
	Confidence       string             `json:"confidence"`
	TransitionNote   string             `json:"transition_note,omitempty"`
	Comment          string             `json:"comment,omitempty"`
	RewriteHints     string             `json:"rewrite_hints,omitempty"`
	StructuredReview json.RawMessage    `json:"structured_review,omitempty"`
	SourceEventID    int64              `json:"source_event_id,omitempty"`
	CreatedAtMs      int64              `json:"created_at_ms"`
}

type ReviewFindings struct {
	Disagreements    []any `json:"disagreements"`
	RisksExtra       []any `json:"risks_extra"`
	Endorsements     []any `json:"endorsements"`
	ReferencedChecks []any `json:"referenced_checks"`
}

type QuestReports struct {
	MakerReports  []MakerReport  `json:"maker_reports"`
	ReviewReports []ReviewReport `json:"review_reports"`
}

func (r *Root) AppendMakerReport(qid string, report *MakerReport) error {
	if qid == "" {
		return fmt.Errorf("quest id 为空")
	}
	if report == nil {
		return fmt.Errorf("maker report 为空")
	}
	now := NowMs()
	if report.SchemaVersion == "" {
		report.SchemaVersion = "maker_report.v1"
	}
	if report.ReportID == "" {
		report.ReportID = "maker_report_" + NewIDShort()
	}
	if report.QuestID == "" {
		report.QuestID = qid
	}
	if report.CreatedAtMs == 0 {
		report.CreatedAtMs = now
	}
	if err := AppendJSONL(r.reportPath(qid, "maker_reports.jsonl"), report); err != nil {
		return err
	}
	return r.appendMakerReportThreadPost(qid, report)
}

func (r *Root) AppendReviewReport(qid string, report *ReviewReport) error {
	if qid == "" {
		return fmt.Errorf("quest id 为空")
	}
	if report == nil {
		return fmt.Errorf("review report 为空")
	}
	now := NowMs()
	if report.SchemaVersion == "" {
		report.SchemaVersion = "review_report.v1"
	}
	if report.ReportID == "" {
		report.ReportID = "review_report_" + NewIDShort()
	}
	if report.QuestID == "" {
		report.QuestID = qid
	}
	if report.Confidence == "" {
		report.Confidence = "low"
	}
	if report.CreatedAtMs == 0 {
		report.CreatedAtMs = now
	}
	if err := AppendJSONL(r.reportPath(qid, "review_reports.jsonl"), report); err != nil {
		return err
	}
	return r.appendReviewReportThreadPost(qid, report)
}

func (r *Root) AppendEvidenceRef(qid string, ref *EvidenceRef) error {
	if qid == "" {
		return fmt.Errorf("quest id 为空")
	}
	if ref == nil {
		return fmt.Errorf("evidence ref 为空")
	}
	if ref.ID == "" {
		ref.ID = "ev_" + NewIDShort()
	}
	if ref.CreatedAtMs == 0 {
		ref.CreatedAtMs = NowMs()
	}
	return AppendJSONL(r.reportPath(qid, "evidence_refs.jsonl"), ref)
}

func (r *Root) LoadEvidenceRefs(qid string) ([]EvidenceRef, error) {
	if qid == "" {
		return nil, fmt.Errorf("quest id 为空")
	}
	refs, err := ReadJSONL[EvidenceRef](r.reportPath(qid, "evidence_refs.jsonl"), 0)
	if err != nil {
		if os.IsNotExist(err) {
			return []EvidenceRef{}, nil
		}
		return nil, err
	}
	return refs, nil
}

func (r *Root) LoadReports(qid string) (*QuestReports, error) {
	if qid == "" {
		return nil, fmt.Errorf("quest id 为空")
	}
	maker, mErr := ReadJSONL[MakerReport](r.reportPath(qid, "maker_reports.jsonl"), 0)
	if mErr != nil && !os.IsNotExist(mErr) {
		return nil, mErr
	}
	review, rErr := ReadJSONL[ReviewReport](r.reportPath(qid, "review_reports.jsonl"), 0)
	if rErr != nil && !os.IsNotExist(rErr) {
		return nil, rErr
	}
	return &QuestReports{MakerReports: maker, ReviewReports: review}, nil
}

func (r *Root) reportPath(qid, name string) string {
	return filepath.Join(r.Sub(SubdirQuests), qid, "reports", name)
}

func (r *Root) appendMakerReportThreadPost(qid string, report *MakerReport) error {
	if r == nil || report == nil {
		return nil
	}
	qs := NewQuestStore(r)
	q, _ := qs.LoadQuest(qid)
	refs := []ThreadArtifactRef{{
		ID:           report.ReportID,
		ArtifactType: model.ArtifactTypeMakerReport,
		Ref:          report.ReportID,
		Role:         model.PostRoleMaker,
	}}
	for _, id := range report.Artifacts {
		if id == "" {
			continue
		}
		refs = append(refs, ThreadArtifactRef{
			ID:           id,
			ArtifactType: model.ArtifactTypeDelivery,
			Ref:          id,
			Role:         model.PostRoleMaker,
		})
	}
	return qs.AppendReportThreadPost(qid, q, AppendThreadPostOptions{
		PostID:         "post_" + report.ReportID,
		ThreadID:       qid,
		ParentReplyID:  RootPostID(qid),
		RootPostID:     RootPostID(qid),
		CausalRefs:     []string{RootPostID(qid)},
		AuthorIdentity: authorIdentityForRole(q, model.PostRoleMaker),
		AuthorRole:     model.PostRoleMaker,
		ArtifactRefs:   refs,
		Kind:           "maker_report",
		Content:        report.Summary,
		SourceEventID:  report.SourceEventID,
		CreatedAtMs:    report.CreatedAtMs,
	})
}

func (r *Root) appendReviewReportThreadPost(qid string, report *ReviewReport) error {
	if r == nil || report == nil {
		return nil
	}
	qs := NewQuestStore(r)
	q, _ := qs.LoadQuest(qid)
	refs := []ThreadArtifactRef{{
		ID:           report.ReportID,
		ArtifactType: model.ArtifactTypeReviewReport,
		Ref:          report.ReportID,
		Role:         model.PostRoleChecker,
	}}
	for _, evidence := range report.EvidenceRefs {
		if evidence.ID == "" {
			continue
		}
		refs = append(refs, ThreadArtifactRef{
			ID:           evidence.ID,
			ArtifactType: model.ArtifactTypeEvidence,
			Ref:          evidence.ID,
			Role:         model.PostRoleChecker,
		})
	}
	causalRefs := append([]string(nil), report.CheckedAgainst...)
	if len(causalRefs) == 0 {
		causalRefs = []string{RootPostID(qid)}
	}
	return qs.AppendReportThreadPost(qid, q, AppendThreadPostOptions{
		PostID:         "post_" + report.ReportID,
		ThreadID:       qid,
		ParentReplyID:  RootPostID(qid),
		RootPostID:     RootPostID(qid),
		CausalRefs:     causalRefs,
		AuthorIdentity: authorIdentityForRole(q, model.PostRoleChecker),
		AuthorRole:     model.PostRoleChecker,
		ArtifactRefs:   refs,
		Kind:           "review_report",
		Content:        report.Comment,
		SourceEventID:  report.SourceEventID,
		CreatedAtMs:    report.CreatedAtMs,
	})
}
