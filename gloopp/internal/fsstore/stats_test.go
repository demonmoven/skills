package fsstore

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

func TestLevelFromExp(t *testing.T) {
	cases := []struct {
		exp  int64
		want int
	}{
		// Lv.1: 0-99
		{0, 1},
		{50, 1},
		{99, 1},
		// Lv.2: 100-299
		{100, 2},
		{150, 2},
		{299, 2},
		// Lv.3: 300-599
		{300, 3},
		{450, 3},
		{599, 3},
		// Lv.4: 600-999
		{600, 4},
		{800, 4},
		{999, 4},
		// Lv.5: 1000-1499
		{1000, 5},
		{1200, 5},
		{1499, 5},
		// Lv.6: 1500-2099
		{1500, 6},
		{2099, 6},
		// Lv.7: 2100-2799
		{2100, 7},
		{2799, 7},
		// Lv.8: 2800-3599
		{2800, 8},
		{3599, 8},
		// Lv.9: 3600-4499
		{3600, 9},
		{4499, 9},
		// Lv.10: 4500+ (满级)
		{4500, 10},
		{5000, 10},
		{10000, 10},
	}
	for _, c := range cases {
		got := LevelFromExp(c.exp)
		if got != c.want {
			t.Errorf("LevelFromExp(%d) = %d, want %d", c.exp, got, c.want)
		}
		if got < 1 || got > 10 {
			t.Errorf("LevelFromExp(%d) = %d, out of range [1,10]", c.exp, got)
		}
	}
}

func TestLevelFromExp_Monotonic(t *testing.T) {
	prevLv := 0
	for exp := int64(0); exp < 10000; exp += 100 {
		lv := LevelFromExp(exp)
		if lv < prevLv {
			t.Errorf("LevelFromExp not monotonic: exp=%d -> lv=%d (prev=%d)", exp, lv, prevLv)
		}
		prevLv = lv
	}
}

func TestLevelFromExp_CapAt10(t *testing.T) {
	if got := LevelFromExp(1e12); got != 10 {
		t.Errorf("LevelFromExp(1e12) = %d, want 10", got)
	}
}

func TestLevelTitle(t *testing.T) {
	cases := []struct {
		class model.AdventurerClass
		level int
		want  string
	}{
		{model.ClassWarrior, 1, "新兵"},
		{model.ClassWarrior, 2, "剑士"},
		{model.ClassWarrior, 3, "大剑士"},
		{model.ClassWarrior, 5, "剑圣"},
		{model.ClassWarrior, 10, "神话剑士"},
		{model.ClassMage, 1, "学徒"},
		{model.ClassMage, 3, "大法师"},
		{model.ClassMage, 5, "魔导师"},
		{model.ClassMage, 7, "贤者"},
		{model.ClassMage, 10, "神话法师"},
		// 边界
		{model.ClassWarrior, 0, ""},
		{model.ClassWarrior, 11, ""},
		{model.AdventurerClass("unknown"), 5, ""},
	}
	for _, c := range cases {
		got := LevelTitle(c.class, c.level)
		if got != c.want {
			t.Errorf("LevelTitle(%s, %d) = %q, want %q", c.class, c.level, got, c.want)
		}
	}
}

func TestAdventurerFile_Title(t *testing.T) {
	a := &AdventurerFile{
		ID:    "adv_test",
		Name:  "测试",
		Class: model.ClassWarrior,
		Level: 5,
	}
	a.populateComputed()
	if a.Title != "剑圣" {
		t.Errorf("warrior Lv.5 Title = %q, want 剑圣", a.Title)
	}
	if a.GetTitle() != "剑圣" {
		t.Errorf("warrior Lv.5 GetTitle() = %q, want 剑圣", a.GetTitle())
	}

	a2 := &AdventurerFile{
		ID:    "adv_mage",
		Name:  "法师",
		Class: model.ClassMage,
		Level: 7,
	}
	a2.populateComputed()
	if a2.Title != "贤者" {
		t.Errorf("mage Lv.7 Title = %q, want 贤者", a2.Title)
	}
}

func TestAggregatePersonalStats(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	qs := NewQuestStore(root)
	q := &QuestMeta{
		ID:             "qst_stats_001",
		ShortID:        "stats001",
		Query:          "统计本地运行",
		Type:           model.QuestTypeExecute,
		Status:         model.QuestStatusSuccess,
		BaseWorkingDir: filepath.Join(t.TempDir(), "demo-project"),
		CreatedBy:      "user",
		StartedAtMs:    1000,
		CompletedAtMs:  5000,
	}
	if err := qs.CreateQuest(q); err != nil {
		t.Fatal(err)
	}
	q.CreatedAtMs = 900
	if err := qs.SaveQuest(q); err != nil {
		t.Fatal(err)
	}
	if err := qs.SavePlan(q.ID, []PhaseTask{{
		PhaseIdx:    0,
		Role:        "warrior",
		Status:      model.PhaseDone,
		StartedAtMs: 1100,
		EndedAtMs:   3100,
	}, {
		PhaseIdx:    1,
		Role:        "mage",
		Status:      model.PhaseDone,
		StartedAtMs: 3200,
		EndedAtMs:   4500,
	}}); err != nil {
		t.Fatal(err)
	}
	if err := root.SaveAgentSessionState(&AgentSessionState{
		SessionID:       "sid_0",
		QuestID:         q.ID,
		Phase:           0,
		CapabilityTier:  "tier_c",
		HealthStatus:    "first_output_seen",
		StartedAtMs:     1100,
		FirstOutputAtMs: 1600,
		LastOutputAtMs:  2100,
	}); err != nil {
		t.Fatal(err)
	}
	if err := qs.AppendSessionRow(q.ID, "sid_0", &QuestSessionRow{
		Seq:       1,
		Timestamp: 2000,
		Kind:      "message",
		SessionID: "sid_0",
		Role:      "assistant",
		Phase:     0,
		Meta: map[string]any{
			"token_input":   float64(120),
			"token_output":  float64(30),
			"duration_ms":   float64(1500),
			"finish_reason": "stop",
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := qs.AppendEvent(q.ID, &QuestEventRow{
		Timestamp: 2500,
		Type:      "micro.command_run",
		QuestID:   q.ID,
		Payload: map[string]any{
			"command_id":  "test",
			"exit_code":   float64(0),
			"duration_ms": float64(700),
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := root.AppendMakerReport(q.ID, &MakerReport{
		Phase:   0,
		Verdict: "done",
		Summary: "maker summary",
	}); err != nil {
		t.Fatal(err)
	}
	if err := root.AppendReviewReport(q.ID, &ReviewReport{
		Phase:          1,
		Verdict:        model.VerdictPass,
		CheckedAgainst: []string{"tests"},
		EvidenceRefs: []EvidenceRef{{
			ID:          "ev_test",
			Kind:        "command",
			TrustTier:   "tier_0",
			Source:      "go test",
			CreatedAtMs: 2600,
		}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := root.ArchiveAutomationDiscovery(&AutomationDiscoveryArchive{
		AutomationID: "auto_stats",
		Outcome:      "no_finding",
		Reason:       "no candidates",
	}); err != nil {
		t.Fatal(err)
	}

	stats, err := AggregatePersonalStats(root, 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if stats.TotalQuests != 1 || stats.SuccessQuests != 1 || stats.TotalTokens != 150 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
	if stats.P95DurationMs != 4000 {
		t.Fatalf("P95DurationMs = %d, want 4000", stats.P95DurationMs)
	}
	if len(stats.PhaseStats) != 2 || stats.PhaseStats[0].Turns != 1 || stats.PhaseStats[0].TokensIn != 120 {
		t.Fatalf("unexpected phase stats: %+v", stats.PhaseStats)
	}
	if len(stats.SlowOperations) != 2 {
		t.Fatalf("slow operations len = %d, want 2", len(stats.SlowOperations))
	}
	if stats.ProjectStats[0].Drilldown.Query["project"][0] != "demo-project" {
		t.Fatalf("project drilldown missing: %+v", stats.ProjectStats[0].Drilldown)
	}
	if len(stats.AnomalySignals) == 0 || stats.AnomalySignals[0].Drilldown.Query["status"][0] == "" {
		t.Fatalf("anomaly signals missing drilldown: %+v", stats.AnomalySignals)
	}
	if stats.SlowOperations[0].Drilldown.Query["quest_id"][0] != q.ID {
		t.Fatalf("slow operation drilldown missing: %+v", stats.SlowOperations[0].Drilldown)
	}
	if !stats.DataCompleteness.SessionRows || !stats.DataCompleteness.TokenUsage || !stats.DataCompleteness.CommandEvents {
		t.Fatalf("unexpected completeness: %+v", stats.DataCompleteness)
	}
	if stats.LoopHealth.StartedSessions != 1 || stats.LoopHealth.FirstOutputCoverage != 1 || stats.LoopHealth.TierCUsage != 1 {
		t.Fatalf("unexpected loop health runtime coverage: %+v", stats.LoopHealth)
	}
	if stats.LoopHealth.MakerReportCoverage != 1 || stats.LoopHealth.ReviewReportCoverage != 1 || stats.LoopHealth.EvidenceBackedReviewCoverage != 1 {
		t.Fatalf("unexpected loop health report coverage: %+v", stats.LoopHealth)
	}
	if stats.LoopHealth.AutomationNoFindingCount != 1 || stats.LoopHealth.AutomationNoFindingRate != 1 {
		t.Fatalf("unexpected loop health automation no-finding: %+v", stats.LoopHealth)
	}
}

func TestAggregatePersonalStatsUsesFailureAttribution(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	qs := NewQuestStore(root)
	q := &QuestMeta{
		ID:             "qst_failure_attr",
		ShortID:        "failattr",
		Query:          "失败归因统计",
		Type:           model.QuestTypeExecute,
		Status:         model.QuestStatusBlocked,
		BaseWorkingDir: filepath.Join(t.TempDir(), "demo-project"),
		CreatedBy:      "user",
		StartedAtMs:    1000,
		BlockedReason:  "agent 连续 2 次执行失败",
		FailureAttribution: &FailureAttribution{
			Stage:       "warrior",
			Reason:      "agent_error_auth",
			Category:    "auth",
			Recoverable: true,
		},
	}
	if err := qs.CreateQuest(q); err != nil {
		t.Fatal(err)
	}
	if err := qs.SaveQuest(q); err != nil {
		t.Fatal(err)
	}
	if err := qs.AppendSessionRow(q.ID, "sid_0", &QuestSessionRow{
		Seq:       1,
		Timestamp: 2000,
		Kind:      "error",
		SessionID: "sid_0",
		Error:     "some noisy raw executor error",
		Phase:     0,
	}); err != nil {
		t.Fatal(err)
	}

	stats, err := AggregatePersonalStats(root, 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(stats.FailureReasons) != 1 || stats.FailureReasons[0].Name != "agent_error_auth" || stats.FailureReasons[0].Count != 1 {
		t.Fatalf("unexpected failure reasons: %+v", stats.FailureReasons)
	}
	if stats.FailureReasons[0].Drilldown.Query["failure_reason"][0] != "agent_error_auth" {
		t.Fatalf("failure reason drilldown missing: %+v", stats.FailureReasons[0].Drilldown)
	}
	if len(stats.FailureCategories) != 1 || stats.FailureCategories[0].Name != "auth" || stats.FailureCategories[0].Count != 1 {
		t.Fatalf("unexpected failure categories: %+v", stats.FailureCategories)
	}
	if stats.FailureCategories[0].Drilldown.Query["failure_category"][0] != "auth" {
		t.Fatalf("failure category drilldown missing: %+v", stats.FailureCategories[0].Drilldown)
	}
	if len(stats.RecentRuns) != 1 || stats.RecentRuns[0].FailureReason != "agent_error_auth" {
		t.Fatalf("recent run should use structured failure reason: %+v", stats.RecentRuns)
	}
}

func TestAggregatePersonalStatsEmptySlices(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	stats, err := AggregatePersonalStats(root, 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if stats.RecentRuns == nil || stats.SlowOperations == nil || stats.ProjectStats == nil || stats.FailureReasons == nil || stats.FailureCategories == nil || stats.PhaseStats == nil {
		t.Fatalf("empty aggregate should return stable empty slices: %+v", stats)
	}
	if stats.LoopHealth.FirstOutputCoverage != 0 ||
		stats.LoopHealth.MakerReportCoverage != 0 ||
		stats.LoopHealth.ReviewReportCoverage != 0 ||
		stats.LoopHealth.EvidenceBackedReviewCoverage != 0 ||
		stats.LoopHealth.AutomationNoFindingRate != 0 ||
		stats.LoopHealth.TierCUsage != 0 {
		t.Fatalf("empty aggregate should keep loop health ratios zero-safe: %+v", stats.LoopHealth)
	}
	if _, err := json.Marshal(stats); err != nil {
		t.Fatalf("empty aggregate should marshal without NaN/Inf: %v", err)
	}
}

func TestAggregatePersonalStatsLoopHealthZeroDenominatorsMarshal(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	qs := NewQuestStore(root)
	q := &QuestMeta{
		ID:          "qst_stats_zero_denominator",
		ShortID:     "zero",
		Query:       "zero denominator",
		Type:        model.QuestTypeExecute,
		Status:      model.QuestStatusPending,
		CreatedBy:   "user",
		CreatedAtMs: 1000,
	}
	if err := qs.CreateQuest(q); err != nil {
		t.Fatal(err)
	}
	stats, err := AggregatePersonalStats(root, 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if stats.LoopHealth.StartedSessions != 0 ||
		stats.LoopHealth.MakerPhaseDoneCount != 0 ||
		stats.LoopHealth.ReviewPhaseDoneCount != 0 ||
		stats.LoopHealth.ReviewReportCount != 0 ||
		stats.LoopHealth.AutomationNoFindingCount != 0 ||
		stats.LoopHealth.AutomationCandidateCount != 0 {
		t.Fatalf("zero-denominator fixture should not create loop health denominators: %+v", stats.LoopHealth)
	}
	if stats.LoopHealth.FirstOutputCoverage != 0 ||
		stats.LoopHealth.MakerReportCoverage != 0 ||
		stats.LoopHealth.ReviewReportCoverage != 0 ||
		stats.LoopHealth.EvidenceBackedReviewCoverage != 0 ||
		stats.LoopHealth.AutomationNoFindingRate != 0 ||
		stats.LoopHealth.TierCUsage != 0 {
		t.Fatalf("zero denominators should produce zero ratios: %+v", stats.LoopHealth)
	}
	if _, err := json.Marshal(stats); err != nil {
		t.Fatalf("zero-denominator aggregate should marshal without NaN/Inf: %v", err)
	}
}
