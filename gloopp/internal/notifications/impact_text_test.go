package notifications

import (
	"strings"
	"testing"

	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

func TestBuildImpactSummaryText(t *testing.T) {
	// 全字段
	info := QuestInfo{
		Impact: model.ImpactSummary{
			WhatChanged: "改了X",
			Affected:    []string{"A", "B"},
			NotTouched:  []string{"C"},
			Caveats:     []string{"注意D"},
		},
	}
	got := buildImpactSummaryText(info)
	if !strings.Contains(got, "做了什么：改了X") {
		t.Errorf("missing WhatChanged: %q", got)
	}
	if !strings.Contains(got, "波及：A、B") {
		t.Errorf("missing Affected: %q", got)
	}
	if !strings.Contains(got, "没碰：C") {
		t.Errorf("missing NotTouched: %q", got)
	}
	if !strings.Contains(got, "注意：注意D") {
		t.Errorf("missing Caveats: %q", got)
	}

	// 部分字段
	info2 := QuestInfo{
		Impact: model.ImpactSummary{WhatChanged: "只改了X"},
	}
	if got2 := buildImpactSummaryText(info2); got2 != "做了什么：只改了X" {
		t.Errorf("partial impact = %q, want %q", got2, "做了什么：只改了X")
	}

	// 空 Impact + 有 Comment → 标注 + comment
	info3 := QuestInfo{Comment: "这是评论"}
	got3 := buildImpactSummaryText(info3)
	if !strings.Contains(got3, "⚠️ agent 未声明影响声明") {
		t.Errorf("missing warning: %q", got3)
	}
	if !strings.Contains(got3, "这是评论") {
		t.Errorf("missing comment: %q", got3)
	}

	// 空 Impact + 空 Comment → 纯标注
	info4 := QuestInfo{}
	if got4 := buildImpactSummaryText(info4); got4 != "⚠️ agent 未声明影响声明" {
		t.Errorf("empty impact+comment = %q, want %q", got4, "⚠️ agent 未声明影响声明")
	}
}
