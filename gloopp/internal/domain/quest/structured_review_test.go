// Package quest — StructuredReview 校验测试（Phase 1.5）。
// 见 mage-review-spec v0.2.2 §8.4。

package quest

import (
	"encoding/json"
	"testing"

	"code.byted.org/lihuanyu.0w0/gloop/internal/events"
)

// fakePublisher 收集事件用于断言
type fakePublisher struct {
	events []struct {
		qid     string
		typ     events.EventType
		payload map[string]any
	}
}

func (f *fakePublisher) Publish(qid, sessionID string, typ events.EventType, payload map[string]any) {
	f.events = append(f.events, struct {
		qid     string
		typ     events.EventType
		payload map[string]any
	}{qid, typ, payload})
}

func TestValidateStructuredReview_Pass(t *testing.T) {
	sr := &StructuredReview{
		SchemaVersion:           "v1",
		ReviewedWarriorPhaseIdx: 0,
		Disagreements: []MageDisagreement{
			{
				ID: "d1", Dimension: DimensionCorrectness, Severity: SeverityHigh,
				TargetKind: EvidenceTargetArtifact,
				Claim:      "test claim",
				Evidence:   []EvidenceRef{{ID: "e1", Kind: EvidenceTargetArtifact}},
			},
		},
	}
	errs := ValidateStructuredReview(sr)
	if len(errs) != 0 {
		t.Errorf("expected 0 errors, got %d: %+v", len(errs), errs)
	}
}

func TestValidateStructuredReview_BadSchemaVersion(t *testing.T) {
	sr := &StructuredReview{SchemaVersion: "v999"}
	errs := ValidateStructuredReview(sr)
	if len(errs) == 0 {
		t.Fatal("expected errors for bad schema_version")
	}
	if errs[0].Code != ErrSchemaVersionUnknown {
		t.Errorf("expected SCHEMA_VERSION_UNKNOWN, got %s", errs[0].Code)
	}
}

func TestValidateStructuredReview_DuplicateID(t *testing.T) {
	sr := &StructuredReview{
		SchemaVersion: "v1",
		Disagreements: []MageDisagreement{
			{ID: "dup", Dimension: DimensionCorrectness, Severity: SeverityHigh, TargetKind: EvidenceTargetArtifact},
			{ID: "dup", Dimension: DimensionCorrectness, Severity: SeverityHigh, TargetKind: EvidenceTargetArtifact},
		},
	}
	errs := ValidateStructuredReview(sr)
	found := false
	for _, e := range errs {
		if e.Code == ErrIDDuplicate {
			found = true
		}
	}
	if !found {
		t.Errorf("expected ID_DUPLICATE error, got: %+v", errs)
	}
}

func TestValidateStructuredReview_ListTooLong(t *testing.T) {
	sr := &StructuredReview{SchemaVersion: "v1"}
	sr.Disagreements = make([]MageDisagreement, MaxDisagreements+1)
	for i := range sr.Disagreements {
		sr.Disagreements[i] = MageDisagreement{
			ID: "d" + string(rune('a'+i)), Dimension: DimensionCorrectness, Severity: SeverityLow, TargetKind: EvidenceTargetOther,
		}
	}
	errs := ValidateStructuredReview(sr)
	found := false
	for _, e := range errs {
		if e.Code == ErrListLengthExceeded {
			found = true
		}
	}
	if !found {
		t.Errorf("expected LIST_LENGTH_EXCEEDED, got: %+v", errs)
	}
}

func TestApplyStructuredValidation_StripsOnFailure(t *testing.T) {
	sr := &StructuredReview{
		SchemaVersion: "vBAD",
		Disagreements: []MageDisagreement{
			{ID: "", Dimension: DimensionCorrectness, Severity: SeverityHigh, TargetKind: EvidenceTargetArtifact},
		},
	}
	rec := &ReviewRecord{
		Ts:               1000,
		Verdict:          "pass",
		Comment:          "base comment",
		ReviewedBy:       "mage",
		StructuredReview: sr,
	}
	fp := &fakePublisher{}

	ApplyStructuredValidation(rec, nil, fp, "q-test")

	if rec.StructuredReview != nil {
		t.Error("StructuredReview should be nil after validation failure (stripped)")
	}
	if len(fp.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(fp.events))
	}
	if fp.events[0].typ != StructuredInvalidEventType {
		t.Errorf("expected %s, got %s", StructuredInvalidEventType, fp.events[0].typ)
	}
	// payload should have raw_sha256
	sha, ok := fp.events[0].payload["raw_sha256"].(string)
	if !ok || sha == "" {
		t.Error("payload missing raw_sha256")
	}
}

func TestApplyStructuredValidation_KeepsOnSuccess(t *testing.T) {
	sr := &StructuredReview{
		SchemaVersion:           "v1",
		ReviewedWarriorPhaseIdx: 0,
		Disagreements: []MageDisagreement{
			{ID: "d1", Dimension: DimensionCorrectness, Severity: SeverityHigh, TargetKind: EvidenceTargetArtifact, Claim: "ok"},
		},
	}
	rec := &ReviewRecord{
		Ts:               1000,
		Verdict:          "pass",
		Comment:          "base",
		ReviewedBy:       "mage",
		StructuredReview: sr,
	}
	fp := &fakePublisher{}

	ApplyStructuredValidation(rec, nil, fp, "q-test")

	if rec.StructuredReview == nil {
		t.Error("StructuredReview should be preserved on validation success")
	}
	if len(fp.events) != 0 {
		t.Errorf("expected 0 events, got %d", len(fp.events))
	}
}

func TestApplyStructuredValidation_NilNoOp(t *testing.T) {
	rec := &ReviewRecord{Ts: 1, Verdict: "pass", Comment: "c", ReviewedBy: "mage"}
	fp := &fakePublisher{}
	ApplyStructuredValidation(rec, nil, fp, "q")
	if rec.StructuredReview != nil {
		t.Error("nil StructuredReview should stay nil")
	}
	if len(fp.events) != 0 {
		t.Error("nil StructuredReview should not emit event")
	}
}

func TestBuildStructuredInvalidPayload_SnippetTruncation(t *testing.T) {
	// >16KB → should truncate
	big := make([]byte, MaxSnippetBytes+1000)
	for i := range big {
		big[i] = 'x'
	}
	sr := &StructuredReview{SchemaVersion: "v1"}
	p := BuildStructuredInvalidPayload(big, sr, []StructuredValidationError{{Code: "X", Path: "", Message: "test"}}, 999, "mage")
	if p.SnippetTruncatedBytes != 1000 {
		t.Errorf("expected 1000 truncated bytes, got %d", p.SnippetTruncatedBytes)
	}
	if len(p.TruncatedSnippet) == 0 {
		t.Error("truncated snippet should not be empty")
	}
	// ≤16KB → full snippet
	small := []byte(`{"hello":"world"}`)
	p2 := BuildStructuredInvalidPayload(small, sr, nil, 999, "mage")
	if p2.SnippetTruncatedBytes != 0 {
		t.Errorf("expected 0 truncated bytes for small payload, got %d", p2.SnippetTruncatedBytes)
	}
}

func TestParseStructuredReview(t *testing.T) {
	// empty → nil, nil
	sr, err := ParseStructuredReview(nil)
	if err != nil || sr != nil {
		t.Error("empty JSON should return nil, nil")
	}
	// valid
	raw := []byte(`{"schema_version":"v1","reviewed_warrior_phase_idx":0,"disagreements":[],"risks_extra":[],"endorsements":[]}`)
	sr, err = ParseStructuredReview(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sr == nil || sr.SchemaVersion != "v1" {
		t.Error("failed to parse valid structured review")
	}
	// invalid JSON
	sr, err = ParseStructuredReview([]byte(`{bad json`))
	if err == nil {
		t.Error("expected error for bad JSON")
	}
}

func TestStructuredReviewJSONRoundtrip(t *testing.T) {
	sr := StructuredReview{
		SchemaVersion:           "v1",
		ReviewedWarriorPhaseIdx: 0,
		Disagreements: []MageDisagreement{
			{ID: "d1", Dimension: DimensionRisk, Severity: SeverityMed, TargetKind: EvidenceTargetDiffFile, Claim: "risk", Evidence: []EvidenceRef{{ID: "e1", Kind: EvidenceTargetDiffFile, Snippet: "line"}}},
		},
	}
	raw, err := json.Marshal(sr)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var back StructuredReview
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if len(back.Disagreements) != 1 || back.Disagreements[0].ID != "d1" {
		t.Error("roundtrip lost data")
	}
}
