package orchestrator

import (
	"reflect"
	"testing"

	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

func TestParseImpactSummary(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want model.ImpactSummary
	}{
		{"空字符串", "", model.ImpactSummary{}},
		{"空白", "  ", model.ImpactSummary{}},
		{"烂JSON", "{bad", model.ImpactSummary{}},
		{
			"全字段",
			`{"what_changed":"改了X","affected":["A","B"],"not_touched":["C"],"caveats":["注意D"]}`,
			model.ImpactSummary{WhatChanged: "改了X", Affected: []string{"A", "B"}, NotTouched: []string{"C"}, Caveats: []string{"注意D"}},
		},
		{
			"部分字段",
			`{"what_changed":"只改了X"}`,
			model.ImpactSummary{WhatChanged: "只改了X"},
		},
		{
			"空数组字段",
			`{"what_changed":"","affected":[]}`,
			// JSON [] 解析成非 nil 空切片；IsEmpty() 仍返回 true（len==0）
			model.ImpactSummary{Affected: []string{}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseImpactSummary(tt.raw)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseImpactSummary(%q) = %+v, want %+v", tt.raw, got, tt.want)
			}
		})
	}
}

func TestExtractImpactFromSignal(t *testing.T) {
	// Data 为 nil
	if got := extractImpactFromSignal(PhaseSignal{Data: nil}); !got.IsEmpty() {
		t.Errorf("nil Data should return empty, got %+v", got)
	}

	// Data 不是 map
	if got := extractImpactFromSignal(PhaseSignal{Data: "string"}); !got.IsEmpty() {
		t.Errorf("non-map Data should return empty, got %+v", got)
	}

	// Data 是 map 但无 impact key
	if got := extractImpactFromSignal(PhaseSignal{Data: map[string]any{"other": "x"}}); !got.IsEmpty() {
		t.Errorf("missing impact key should return empty, got %+v", got)
	}

	// impact 是 string
	got := extractImpactFromSignal(PhaseSignal{Data: map[string]any{
		"impact": `{"what_changed":"从string解析"}`,
	}})
	if got.WhatChanged != "从string解析" {
		t.Errorf("string impact: WhatChanged = %q, want %q", got.WhatChanged, "从string解析")
	}

	// impact 是 map[string]any
	got = extractImpactFromSignal(PhaseSignal{Data: map[string]any{
		"impact": map[string]any{"what_changed": "从map解析"},
	}})
	if got.WhatChanged != "从map解析" {
		t.Errorf("map impact: WhatChanged = %q, want %q", got.WhatChanged, "从map解析")
	}

	// impact 是 model.ImpactSummary
	got = extractImpactFromSignal(PhaseSignal{Data: map[string]any{
		"impact": model.ImpactSummary{WhatChanged: "直接结构体"},
	}})
	if got.WhatChanged != "直接结构体" {
		t.Errorf("struct impact: WhatChanged = %q, want %q", got.WhatChanged, "直接结构体")
	}
}
