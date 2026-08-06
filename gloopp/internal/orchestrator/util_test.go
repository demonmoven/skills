package orchestrator

import (
	"strings"
	"testing"

	"code.byted.org/lihuanyu.0w0/gloop/internal/executor"
	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

// phaseDoneResp builds a ChatResponse that simulates `gloop phase done` via PhaseSignal.
// Used by test executors that need to end a warrior (execution) phase.
func phaseDoneResp(sid, content, comment string) *executor.ChatResponse {
	return &executor.ChatResponse{
		SessionID:    sid,
		Message:      executor.Message{Role: "assistant", Content: content},
		FinishReason: "stop",
		PhaseSignal: &executor.PhaseSignalData{
			OK:           true,
			Message:      "阶段结论已提交",
			PhaseEnded:   true,
			PhaseVerdict: "done",
			PhaseComment: comment,
		},
	}
}

// reviewDoneResp builds a ChatResponse that simulates `gloop review` via PhaseSignal.
// Used by test executors that need to end a mage (review) phase.
func reviewDoneResp(sid, content, verdict, comment, hints string, score int) *executor.ChatResponse {
	return &executor.ChatResponse{
		SessionID:    sid,
		Message:      executor.Message{Role: "assistant", Content: content},
		FinishReason: "stop",
		PhaseSignal: &executor.PhaseSignalData{
			OK:           true,
			Message:      "评审结论已提交",
			PhaseEnded:   true,
			PhaseVerdict: verdict,
			PhaseComment: comment,
			PhaseHints:   hints,
			PhaseScore:   score,
		},
	}
}

func TestTruncate(t *testing.T) {
	cases := []struct {
		in     string
		n      int
		want   string
		suffix bool
	}{
		{"hello", 10, "hello", false},
		{"hello", 3, "hel…", true},
		{"你好世界", 2, "你好…", true},
		{"你好", 5, "你好", false},
		{"", 5, "", false},
	}
	for i, c := range cases {
		got := truncate(c.in, c.n)
		if c.suffix && !strings.HasSuffix(got, "…") {
			t.Errorf("case %d: truncate(%q, %d) = %q, want suffix …", i, c.in, c.n, got)
		}
		if !c.suffix && got != c.want {
			t.Errorf("case %d: truncate(%q, %d) = %q, want %q", i, c.in, c.n, got, c.want)
		}
	}
}

func TestToJSON(t *testing.T) {
	v := map[string]any{"a": 1, "b": "x"}
	s := toJSON(v)
	if !strings.Contains(s, "a") {
		t.Errorf("toJSON = %q, want contain a", s)
	}
	ch := make(chan int)
	s2 := toJSON(ch)
	if s2 == "" {
		t.Error("toJSON(chan) should return fallback string")
	}
}

func TestPhaseShouldStopHonorsPhaseContractAndBlocked(t *testing.T) {
	cfg := &phaseConfig{IsDone: isValidReviewVerdict}
	if !phaseShouldStop(cfg, PhaseSignal{PhaseEnded: true, PhaseVerdict: "pass"}) {
		t.Fatal("valid review verdict should stop the phase")
	}
	if !phaseShouldStop(cfg, PhaseSignal{PhaseEnded: true, PhaseVerdict: "blocked"}) {
		t.Fatal("blocked verdict should stop the phase")
	}
	if phaseShouldStop(cfg, PhaseSignal{PhaseEnded: true, PhaseVerdict: "invalid_review_verdict"}) {
		t.Fatal("invalid review verdict should not stop the phase")
	}
}

func TestMarshalMap(t *testing.T) {
	if s := marshalMap(nil); s != "" {
		t.Errorf("marshalMap(nil) = %q, want empty", s)
	}
	if s := marshalMap(map[string]any{}); s != "" {
		t.Errorf("marshalMap({}) = %q, want empty", s)
	}
	s := marshalMap(map[string]any{"k": "v"})
	if !strings.Contains(s, "k") {
		t.Errorf("marshalMap({k:v}) = %q, want contain k", s)
	}
}

func TestExtractJSON(t *testing.T) {
	tests := []struct {
		input string
		found bool
	}{
		{"hello {\"a\":1} world", true},
		{"no json here", false},
		{"{\"nested\":{\"deep\":1}}", true},
		{"", false},
	}
	for i, tt := range tests {
		_, ok := extractJSON(tt.input)
		if ok != tt.found {
			t.Errorf("case %d: extractJSON(%q) found=%v, want %v", i, tt.input, ok, tt.found)
		}
	}
}

func TestCalcExpGain(t *testing.T) {
	// 验证基础值（无返工、无评分）
	w, m := calcExpGain(model.VerdictPass, 0, 0)
	if w != 100 || m != 60 {
		t.Errorf("pass 0 rework score=0: warrior=%d mage=%d, want 100/60", w, m)
	}

	w, m = calcExpGain(model.VerdictReject, 0, 0)
	if w != 5 || m != 40 {
		t.Errorf("reject 0 rework score=0: warrior=%d mage=%d, want 5/40", w, m)
	}

	w, m = calcExpGain(model.VerdictRequestChange, 0, 0)
	if w != 20 || m != 25 {
		t.Errorf("request_change 0 rework score=0: warrior=%d mage=%d, want 20/25", w, m)
	}
}

func TestCalcExpGain_ReworkPenalty(t *testing.T) {
	// 每次返工 -20%，最低 0.5 倍
	cases := []struct {
		rework int
		wantW  int64 // pass verdict 基础 100
		wantM  int64 // pass verdict 基础 60
	}{
		{0, 100, 60},
		{1, 80, 48},  // ×0.8
		{2, 60, 36},  // ×0.6
		{3, 50, 30},  // ×0.4 → 被地板 0.5 限住 → ×0.5
		{5, 50, 30},  // 0 rework 后一直是 0.5 倍
		{10, 50, 30}, // 超多次返工，仍有 0.5 倍保底
	}
	for _, c := range cases {
		w, m := calcExpGain(model.VerdictPass, c.rework, 0)
		if w != c.wantW || m != c.wantM {
			t.Errorf("pass rework=%d score=0: warrior=%d mage=%d, want %d/%d",
				c.rework, w, m, c.wantW, c.wantM)
		}
	}
}

func TestCalcExpGain_QualityCoefficient(t *testing.T) {
	// 质量系数仅在 pass 且有评分时作用于剑士
	cases := []struct {
		verdict model.QuestVerdict
		score   int
		wantW   int64
		wantM   int64
		desc    string
	}{
		{model.VerdictPass, 0, 100, 60, "pass score=0（无评分，不乘质量系数）"},
		{model.VerdictPass, 10, 150, 60, "pass score=10（×1.5）"},
		{model.VerdictPass, 9, 150, 60, "pass score=9（×1.5）"},
		{model.VerdictPass, 8, 150, 60, "pass score=8（×1.5）"},
		{model.VerdictPass, 7, 100, 60, "pass score=7（×1.0）"},
		{model.VerdictPass, 6, 100, 60, "pass score=6（×1.0）"},
		{model.VerdictPass, 5, 100, 60, "pass score=5（×1.0）"},
		{model.VerdictPass, 4, 50, 60, "pass score=4（×0.5）"},
		{model.VerdictPass, 1, 50, 60, "pass score=1（×0.5）"},
		{model.VerdictReject, 8, 5, 40, "reject score=8（质量系数不生效）"},
		{model.VerdictRequestChange, 8, 20, 25, "request_change score=8（质量系数不生效）"},
	}
	for _, c := range cases {
		w, m := calcExpGain(c.verdict, 0, c.score)
		if w != c.wantW || m != c.wantM {
			t.Errorf("%s: warrior=%d mage=%d, want %d/%d",
				c.desc, w, m, c.wantW, c.wantM)
		}
	}
}

func TestCalcExpGain_ReworkAndQualityCombined(t *testing.T) {
	// 返工惩罚 × 质量系数 组合效果
	// pass verdict, 1 次返工(×0.8), 评分 9(×1.5) → 剑士: 100 × 0.8 × 1.5 = 120
	// pass verdict, 2 次返工(×0.6), 评分 9(×1.5) → 剑士: 100 × 0.6 × 1.5 = 90
	// pass verdict, 3 次返工(×0.5地板), 评分 3(×0.5) → 剑士: 100 × 0.5 × 0.5 = 25
	cases := []struct {
		rework int
		score  int
		wantW  int64
		wantM  int64
	}{
		{1, 9, 120, 48},
		{2, 9, 90, 36},
		{3, 3, 25, 30},
		{0, 10, 150, 60},
	}
	for _, c := range cases {
		w, m := calcExpGain(model.VerdictPass, c.rework, c.score)
		if w != c.wantW || m != c.wantM {
			t.Errorf("pass rework=%d score=%d: warrior=%d mage=%d, want %d/%d",
				c.rework, c.score, w, m, c.wantW, c.wantM)
		}
	}
}
