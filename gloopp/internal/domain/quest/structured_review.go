// Package quest — StructuredReview 类型定义与两层校验（Phase 1.5）。
//
// 见 mage-review-spec v0.2.2 §4.1 + §8.4。
// 设计原则：
//   - 所有字段 nullable；缺省 = 未提供，前端降级展示原始 comment
//   - 平台只做协议完整性校验（类型 / id 唯一 / 非负整数 / 列表长度），不做语义判断
//   - 两层校验：base 字段失败 = 整条 review 拒绝；StructuredReview 失败 = 丢弃 structured 部分，保留 base
//   - 校验失败时写 review.structured_invalid event（payload 严格 8 字段，≤20KB）

package quest

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"code.byted.org/lihuanyu.0w0/gloop/internal/events"
)

// ---------------------------------------------------------------------------
// 类型定义（对齐 spec §4.1 TS contract）
// ---------------------------------------------------------------------------

type EvidenceTargetKind string

const (
	EvidenceTargetWarriorMessage  EvidenceTargetKind = "warrior_message"
	EvidenceTargetArtifact        EvidenceTargetKind = "artifact"
	EvidenceTargetCheck           EvidenceTargetKind = "check"
	EvidenceTargetDiffFile        EvidenceTargetKind = "diff_file"
	EvidenceTargetWorkspaceEffect EvidenceTargetKind = "workspace_effect"
	EvidenceTargetPolicyTarget    EvidenceTargetKind = "policy_target"
	EvidenceTargetQuest           EvidenceTargetKind = "quest"
	EvidenceTargetDesignDoc       EvidenceTargetKind = "design_doc"
	EvidenceTargetOther           EvidenceTargetKind = "other"
)

var validEvidenceTargetKinds = map[EvidenceTargetKind]bool{
	EvidenceTargetWarriorMessage: true, EvidenceTargetArtifact: true, EvidenceTargetCheck: true,
	EvidenceTargetDiffFile: true, EvidenceTargetWorkspaceEffect: true, EvidenceTargetPolicyTarget: true,
	EvidenceTargetQuest: true, EvidenceTargetDesignDoc: true, EvidenceTargetOther: true,
}

type MageDimension string

const (
	DimensionCorrectness  MageDimension = "correctness"
	DimensionCompleteness MageDimension = "completeness"
	DimensionBoundary     MageDimension = "boundary"
	DimensionRisk         MageDimension = "risk"
	DimensionValidation   MageDimension = "validation"
	DimensionCoherence    MageDimension = "coherence"
	DimensionScope        MageDimension = "scope"
	DimensionStyle        MageDimension = "style"
	DimensionOther        MageDimension = "other"
)

var validDimensions = map[MageDimension]bool{
	DimensionCorrectness: true, DimensionCompleteness: true, DimensionBoundary: true,
	DimensionRisk: true, DimensionValidation: true, DimensionCoherence: true,
	DimensionScope: true, DimensionStyle: true, DimensionOther: true,
}

type MageSeverity string

const (
	SeverityHigh MageSeverity = "high"
	SeverityMed  MageSeverity = "med"
	SeverityLow  MageSeverity = "low"
	SeverityInfo MageSeverity = "info"
)

var validSeverities = map[MageSeverity]bool{
	SeverityHigh: true, SeverityMed: true, SeverityLow: true, SeverityInfo: true,
}

type EvidenceRef struct {
	ID      string             `json:"id"`
	Kind    EvidenceTargetKind `json:"kind"`
	Anchor  string             `json:"anchor,omitempty"`
	Field   string             `json:"field,omitempty"`
	Snippet string             `json:"snippet,omitempty"`
}

type MageDisagreement struct {
	ID         string             `json:"id"`
	Dimension  MageDimension      `json:"dimension"`
	Severity   MageSeverity       `json:"severity"`
	TargetKind EvidenceTargetKind `json:"target_kind"`
	Claim      string             `json:"claim"`
	Evidence   []EvidenceRef      `json:"evidence"`
	Suggestion string             `json:"suggestion,omitempty"`
}

type MageExtraRisk struct {
	ID       string        `json:"id"`
	Kind     string        `json:"kind"`
	Severity MageSeverity  `json:"severity"`
	Detail   string        `json:"detail"`
	Evidence []EvidenceRef `json:"evidence"`
}

type MageEndorsement struct {
	ID        string        `json:"id"`
	Dimension MageDimension `json:"dimension"`
	Reason    string        `json:"reason"`
	Evidence  []EvidenceRef `json:"evidence"`
}

type MageCheckReference struct {
	CheckID  string `json:"check_id"`
	Expected string `json:"expected,omitempty"`
	Comment  string `json:"comment,omitempty"`
}

type FreeCommentRange struct {
	StartChar int `json:"start_char"`
	EndChar   int `json:"end_char"`
}

type StructuredReview struct {
	SchemaVersion             string               `json:"schema_version"`
	ReviewedWarriorPhaseIdx   int                  `json:"reviewed_warrior_phase_idx"`
	ReviewedWarriorSessionID  string               `json:"reviewed_warrior_session_id,omitempty"`
	ReviewedWarriorLatestTurn int                  `json:"reviewed_warrior_latest_turn,omitempty"`
	CoverageRatio             [2]int               `json:"coverage_ratio,omitempty"`
	Confidence                string               `json:"confidence,omitempty"`
	Disagreements             []MageDisagreement   `json:"disagreements"`
	RisksExtra                []MageExtraRisk      `json:"risks_extra"`
	Endorsements              []MageEndorsement    `json:"endorsements"`
	ReferencedChecks          []MageCheckReference `json:"referenced_checks,omitempty"`
	FreeCommentRange          *FreeCommentRange    `json:"free_comment_range,omitempty"`
}

// ---------------------------------------------------------------------------
// 列表长度上限（spec §4.1 原则 4）
// ---------------------------------------------------------------------------

const (
	MaxDisagreements   = 20
	MaxEndorsements    = 3
	MaxEvidencePerItem = 5
	MaxErrorsInEvent   = 20
	MaxSnippetBytes    = 16384 // 16KB
)

// ---------------------------------------------------------------------------
// 校验错误
// ---------------------------------------------------------------------------

type StructuredValidationError struct {
	Code    string      `json:"code"`
	Path    string      `json:"path"`
	Message string      `json:"message"`
	Got     interface{} `json:"got,omitempty"`
}

const (
	ErrSchemaVersionUnknown = "SCHEMA_VERSION_UNKNOWN"
	ErrListLengthExceeded   = "LIST_LENGTH_EXCEEDED"
	ErrIDDuplicate          = "ID_DUPLICATE"
	ErrIDEmpty              = "ID_EMPTY"
	ErrEnumInvalid          = "ENUM_INVALID"
	ErrTypeMismatch         = "TYPE_MISMATCH"
	ErrEvidenceKindUnknown  = "EVIDENCE_KIND_UNKNOWN"
)

// ---------------------------------------------------------------------------
// 两层校验
// ---------------------------------------------------------------------------

// ValidateStructuredReview 校验 StructuredReview 的协议完整性。
// 返回 errors 列表；空列表 = 校验通过。
// 平台只做类型/枚举/id 唯一性/列表长度校验，不做语义判断。
func ValidateStructuredReview(sr *StructuredReview) []StructuredValidationError {
	var errs []StructuredValidationError

	// schema_version
	if sr.SchemaVersion != "v1" {
		errs = append(errs, StructuredValidationError{
			Code:    ErrSchemaVersionUnknown,
			Path:    "schema_version",
			Message: fmt.Sprintf("unknown schema_version %q, expected \"v1\"", sr.SchemaVersion),
			Got:     sr.SchemaVersion,
		})
	}

	// disagreements
	if len(sr.Disagreements) > MaxDisagreements {
		errs = append(errs, StructuredValidationError{
			Code:    ErrListLengthExceeded,
			Path:    "disagreements",
			Message: fmt.Sprintf("disagreements length %d exceeds max %d", len(sr.Disagreements), MaxDisagreements),
		})
	}
	seenIDs := make(map[string]bool)
	for i, d := range sr.Disagreements {
		base := fmt.Sprintf("disagreements[%d]", i)
		if d.ID == "" {
			errs = append(errs, StructuredValidationError{Code: ErrIDEmpty, Path: base + ".id", Message: "id is empty"})
		} else if seenIDs[d.ID] {
			errs = append(errs, StructuredValidationError{Code: ErrIDDuplicate, Path: base + ".id", Message: fmt.Sprintf("duplicate id %q", d.ID)})
		}
		seenIDs[d.ID] = true
		if !validDimensions[d.Dimension] {
			errs = append(errs, StructuredValidationError{Code: ErrEnumInvalid, Path: base + ".dimension", Message: fmt.Sprintf("invalid dimension %q", d.Dimension), Got: string(d.Dimension)})
		}
		if !validSeverities[d.Severity] {
			errs = append(errs, StructuredValidationError{Code: ErrEnumInvalid, Path: base + ".severity", Message: fmt.Sprintf("invalid severity %q", d.Severity), Got: string(d.Severity)})
		}
		if !validEvidenceTargetKinds[d.TargetKind] {
			errs = append(errs, StructuredValidationError{Code: ErrEvidenceKindUnknown, Path: base + ".target_kind", Message: fmt.Sprintf("invalid target_kind %q", d.TargetKind), Got: string(d.TargetKind)})
		}
		if len(d.Evidence) > MaxEvidencePerItem {
			errs = append(errs, StructuredValidationError{Code: ErrListLengthExceeded, Path: base + ".evidence", Message: fmt.Sprintf("evidence length %d exceeds max %d", len(d.Evidence), MaxEvidencePerItem)})
		}
		errs = append(errs, validateEvidenceRefs(d.Evidence, base+".evidence")...)
	}

	// endorsements
	if len(sr.Endorsements) > MaxEndorsements {
		errs = append(errs, StructuredValidationError{Code: ErrListLengthExceeded, Path: "endorsements", Message: fmt.Sprintf("endorsements length %d exceeds max %d", len(sr.Endorsements), MaxEndorsements)})
	}
	for i, e := range sr.Endorsements {
		base := fmt.Sprintf("endorsements[%d]", i)
		if e.ID == "" {
			errs = append(errs, StructuredValidationError{Code: ErrIDEmpty, Path: base + ".id", Message: "id is empty"})
		} else if seenIDs[e.ID] {
			errs = append(errs, StructuredValidationError{Code: ErrIDDuplicate, Path: base + ".id", Message: fmt.Sprintf("duplicate id %q", e.ID)})
		}
		seenIDs[e.ID] = true
		if !validDimensions[e.Dimension] {
			errs = append(errs, StructuredValidationError{Code: ErrEnumInvalid, Path: base + ".dimension", Message: fmt.Sprintf("invalid dimension %q", e.Dimension), Got: string(e.Dimension)})
		}
		if len(e.Evidence) > MaxEvidencePerItem {
			errs = append(errs, StructuredValidationError{Code: ErrListLengthExceeded, Path: base + ".evidence", Message: fmt.Sprintf("evidence length %d exceeds max %d", len(e.Evidence), MaxEvidencePerItem)})
		}
		errs = append(errs, validateEvidenceRefs(e.Evidence, base+".evidence")...)
	}

	// risks_extra
	for i, r := range sr.RisksExtra {
		base := fmt.Sprintf("risks_extra[%d]", i)
		if r.ID == "" {
			errs = append(errs, StructuredValidationError{Code: ErrIDEmpty, Path: base + ".id", Message: "id is empty"})
		} else if seenIDs[r.ID] {
			errs = append(errs, StructuredValidationError{Code: ErrIDDuplicate, Path: base + ".id", Message: fmt.Sprintf("duplicate id %q", r.ID)})
		}
		seenIDs[r.ID] = true
		if !validSeverities[r.Severity] {
			errs = append(errs, StructuredValidationError{Code: ErrEnumInvalid, Path: base + ".severity", Message: fmt.Sprintf("invalid severity %q", r.Severity), Got: string(r.Severity)})
		}
		if len(r.Evidence) > MaxEvidencePerItem {
			errs = append(errs, StructuredValidationError{Code: ErrListLengthExceeded, Path: base + ".evidence", Message: fmt.Sprintf("evidence length %d exceeds max %d", len(r.Evidence), MaxEvidencePerItem)})
		}
		errs = append(errs, validateEvidenceRefs(r.Evidence, base+".evidence")...)
	}

	// 截断错误列表（spec §8.4：超 20 条合并）
	if len(errs) > MaxErrorsInEvent {
		merged := StructuredValidationError{
			Code:    "LIST_TOO_LONG_MANY_ERRORS",
			Path:    "",
			Message: fmt.Sprintf("too many validation errors (%d), showing first %d", len(errs), MaxErrorsInEvent),
		}
		errs = append(errs[:MaxErrorsInEvent], merged)
	}

	return errs
}

func validateEvidenceRefs(refs []EvidenceRef, basePath string) []StructuredValidationError {
	var errs []StructuredValidationError
	for i, ref := range refs {
		if ref.ID == "" {
			errs = append(errs, StructuredValidationError{Code: ErrIDEmpty, Path: fmt.Sprintf("%s[%d].id", basePath, i), Message: "evidence id is empty"})
		}
		if !validEvidenceTargetKinds[ref.Kind] {
			errs = append(errs, StructuredValidationError{Code: ErrEvidenceKindUnknown, Path: fmt.Sprintf("%s[%d].kind", basePath, i), Message: fmt.Sprintf("invalid evidence kind %q", ref.Kind), Got: string(ref.Kind)})
		}
	}
	return errs
}

// ---------------------------------------------------------------------------
// structured_invalid event 构造（spec §8.4）
// ---------------------------------------------------------------------------

// StructuredInvalidEventPayload review.structured_invalid event 的 payload。
// 严格 8 字段，不含完整 structured JSON。硬上限 ≈20KB（典型 <4KB）。
type StructuredInvalidEventPayload struct {
	SchemaVersion         string                      `json:"schema_version"`
	RawSizeBytes          int                         `json:"raw_size_bytes"`
	RawSha256             string                      `json:"raw_sha256"`
	Errors                []StructuredValidationError `json:"errors"`
	TruncatedSnippet      string                      `json:"truncated_snippet,omitempty"`
	SnippetTruncatedBytes int                         `json:"snippet_truncated_bytes,omitempty"`
	AttachedReviewTsMs    int64                       `json:"attached_review_ts_ms"`
	ReviewedBy            string                      `json:"reviewed_by"`
}

// BuildStructuredInvalidPayload 从原始 structured JSON 字节构造 event payload。
// rawJSON 是提交方原始 JSON（可能无法反序列化到 StructuredReview）。
func BuildStructuredInvalidPayload(rawJSON []byte, sr *StructuredReview, errs []StructuredValidationError, reviewTs int64, reviewedBy string) StructuredInvalidEventPayload {
	hash := sha256.Sum256(rawJSON)
	payload := StructuredInvalidEventPayload{
		RawSizeBytes:       len(rawJSON),
		RawSha256:          hex.EncodeToString(hash[:]),
		Errors:             errs,
		AttachedReviewTsMs: reviewTs,
		ReviewedBy:         reviewedBy,
	}
	if sr != nil {
		payload.SchemaVersion = sr.SchemaVersion
	}
	// ≤16KB middle snippet
	if len(rawJSON) > 0 && len(rawJSON) <= MaxSnippetBytes {
		payload.TruncatedSnippet = string(rawJSON)
	} else if len(rawJSON) > MaxSnippetBytes {
		half := MaxSnippetBytes / 2
		truncated := len(rawJSON) - MaxSnippetBytes
		payload.TruncatedSnippet = fmt.Sprintf("...%s...[TRUNCATED by %d bytes]...%s...",
			rawJSON[:half], truncated, rawJSON[len(rawJSON)-half:])
		payload.SnippetTruncatedBytes = truncated
	}
	return payload
}

// StructuredInvalidEventType review.structured_invalid 事件类型。
const StructuredInvalidEventType events.EventType = "review.structured_invalid"

// ParseStructuredReview 尝试从 rawJSON 解析 StructuredReview。
// 解析失败时返回 nil + 错误（调用方应据此发 structured_invalid event）。
func ParseStructuredReview(rawJSON []byte) (*StructuredReview, error) {
	if len(rawJSON) == 0 {
		return nil, nil
	}
	var sr StructuredReview
	if err := json.Unmarshal(rawJSON, &sr); err != nil {
		return nil, fmt.Errorf("structured review JSON parse failed: %w", err)
	}
	return &sr, nil
}

// ApplyStructuredValidation 对 ReviewRecord 执行 StructuredReview 两层校验（spec §8.4）。
//
// 行为：
//   - StructuredReview 为 nil → 直接返回（无 structured 部分，无需校验）
//   - 校验通过 → 保留 StructuredReview，不发声
//   - 校验失败 → 把 rec.StructuredReview 置 nil（丢弃 structured 部分，保留 base），
//     并通过 publisher 发 review.structured_invalid event（payload 严格 8 字段）
//
// rawJSON 用于计算 SHA-256 和 snippet；如果 rawJSON 为 nil，则从 StructuredReview 重新 marshal。
func ApplyStructuredValidation(rec *ReviewRecord, rawJSON []byte, publisher EventPublisher, qid string) {
	if rec.StructuredReview == nil {
		return
	}

	if rawJSON == nil {
		var err error
		rawJSON, err = json.Marshal(rec.StructuredReview)
		if err != nil {
			return
		}
	}

	errs := ValidateStructuredReview(rec.StructuredReview)
	if len(errs) == 0 {
		return
	}

	// 校验失败：丢弃 structured 部分，保留 base（"tail must not wag dog"）
	payload := BuildStructuredInvalidPayload(rawJSON, rec.StructuredReview, errs, rec.Ts, rec.ReviewedBy)
	rec.StructuredReview = nil

	if publisher != nil {
		publisher.Publish(qid, "", StructuredInvalidEventType, PayloadToMap(payload))
	}
}

// PayloadToMap 把 StructuredInvalidEventPayload 转成 map[string]any 供 EventBus.Publish 使用。
func PayloadToMap(p StructuredInvalidEventPayload) map[string]any {
	// 通过 round-trip JSON 保证 payload 结构稳定
	b, err := json.Marshal(p)
	if err != nil {
		return map[string]any{"error": "failed to marshal structured_invalid payload"}
	}
	m := map[string]any{}
	_ = json.Unmarshal(b, &m)
	return m
}
