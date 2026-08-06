package fsstore

import (
	"reflect"
	"testing"
)

func TestIsValidTrustTier(t *testing.T) {
	valid := []string{TrustTier0, TrustTier1, TrustTier2, TrustTier3}
	for _, v := range valid {
		if !IsValidTrustTier(v) {
			t.Errorf("%s should be valid", v)
		}
	}
	invalid := []string{"", "tier_4", "Tier1", "unknown"}
	for _, v := range invalid {
		if IsValidTrustTier(v) {
			t.Errorf("%s should be invalid", v)
		}
	}
}

func TestAutomationTrustOutcomeDedupKey(t *testing.T) {
	terminal := AutomationTrustOutcome{QuestID: "q1"}
	if k := AutomationTrustOutcomeDedupKey(terminal); k != "q1:terminal" {
		t.Errorf("default dedup key = %q, want q1:terminal", k)
	}
	terminalExplicit := AutomationTrustOutcome{QuestID: "q2", OutcomeType: TrustOutcomeTypeTerminal}
	if k := AutomationTrustOutcomeDedupKey(terminalExplicit); k != "q2:terminal" {
		t.Errorf("terminal dedup key = %q", k)
	}
	stage := AutomationTrustOutcome{QuestID: "q3", OutcomeType: TrustOutcomeTypeStage, VerificationType: "build"}
	if k := AutomationTrustOutcomeDedupKey(stage); k != "q3:stage:build" {
		t.Errorf("stage dedup key = %q, want q3:stage:build", k)
	}
}

func TestNormalizeAutomationTrustOutcome(t *testing.T) {
	// nil 安全
	normalizeAutomationTrustOutcome(nil)

	// 空 OutcomeType 归一
	o := AutomationTrustOutcome{QuestID: "q1", Outcome: "success"}
	normalizeAutomationTrustOutcome(&o)
	if o.OutcomeType != TrustOutcomeTypeTerminal {
		t.Errorf("empty OutcomeType should become terminal, got %s", o.OutcomeType)
	}
	if o.DedupKey != "q1:terminal" {
		t.Errorf("missing DedupKey should be generated, got %s", o.DedupKey)
	}

	// 已有 DedupKey 不覆盖
	o2 := AutomationTrustOutcome{QuestID: "q2", DedupKey: "custom-key"}
	normalizeAutomationTrustOutcome(&o2)
	if o2.DedupKey != "custom-key" {
		t.Errorf("existing DedupKey should not be overwritten, got %s", o2.DedupKey)
	}
}

func TestNormalizeAutomationTrustState(t *testing.T) {
	// 空 Tier 归一
	s := &AutomationTrustState{AutomationID: "a1"}
	normalizeAutomationTrustState(s)
	if s.Tier != TrustTier1 {
		t.Errorf("empty Tier should default to %s, got %s", TrustTier1, s.Tier)
	}

	// Recent 溢出裁剪（>10 条）
	var recent []AutomationTrustOutcome
	for i := 0; i < 15; i++ {
		recent = append(recent, AutomationTrustOutcome{QuestID: "q" + itoa(i), Outcome: "success"})
	}
	s2 := &AutomationTrustState{AutomationID: "a2", Tier: TrustTier1, Recent: recent}
	normalizeAutomationTrustState(s2)
	if len(s2.Recent) != 10 {
		t.Fatalf("Recent should be trimmed to 10, got %d", len(s2.Recent))
	}
	if s2.Recent[0].QuestID != "q5" {
		t.Errorf("oldest kept should be q5, got %s", s2.Recent[0].QuestID)
	}

	// OutcomeKeys 溢出裁剪（>200 条）
	var keys []string
	for i := 0; i < 210; i++ {
		keys = append(keys, "k"+itoa(i))
	}
	s3 := &AutomationTrustState{AutomationID: "a3", Tier: TrustTier1, OutcomeKeys: keys}
	normalizeAutomationTrustState(s3)
	if len(s3.OutcomeKeys) != 200 {
		t.Fatalf("OutcomeKeys should be trimmed to 200, got %d", len(s3.OutcomeKeys))
	}
	if s3.OutcomeKeys[0] != "k10" {
		t.Errorf("oldest kept OutcomeKey should be k10, got %s", s3.OutcomeKeys[0])
	}

	// 空 OutcomeKey 去重
	s4 := &AutomationTrustState{
		AutomationID: "a4",
		Tier:         TrustTier1,
		OutcomeKeys:  []string{"", "k1", "", "k1"}, // 空值应被过滤，但原逻辑只从 Recent 新增时过滤；已有保留
		Recent: []AutomationTrustOutcome{
			{QuestID: "q1", Outcome: "success"},
			{QuestID: "q1", Outcome: "success"}, // 相同 dedup，不应被追加到 OutcomeKeys
		},
	}
	normalizeAutomationTrustState(s4)
	// q1:terminal 应只出现一次
	count := 0
	for _, k := range s4.OutcomeKeys {
		if k == "q1:terminal" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("dedup key in OutcomeKeys should appear exactly once, got %d; keys=%+v", count, s4.OutcomeKeys)
	}
}

func TestHasOutcomeDedupKey(t *testing.T) {
	// nil 安全
	var ns *AutomationTrustState
	if ns.HasOutcomeDedupKey("x") {
		t.Error("nil state HasOutcomeDedupKey should be false")
	}
	if ns.HasOutcomeDedupKey("") {
		t.Error("nil state HasOutcomeDedupKey('') should be false")
	}
	// 空 key 永远 false
	s := &AutomationTrustState{AutomationID: "a"}
	if s.HasOutcomeDedupKey("") {
		t.Error("empty key should return false")
	}
	// 在 OutcomeKeys 中
	s2 := &AutomationTrustState{AutomationID: "a", OutcomeKeys: []string{"k1"}}
	if !s2.HasOutcomeDedupKey("k1") {
		t.Error("k1 should be found in OutcomeKeys")
	}
	if s2.HasOutcomeDedupKey("k2") {
		t.Error("k2 should not be found")
	}
	// 在 Recent 中（且 normalize 会生成 DedupKey）
	s3 := &AutomationTrustState{
		AutomationID: "a",
		Recent:       []AutomationTrustOutcome{{QuestID: "q1", Outcome: "success"}},
	}
	if !s3.HasOutcomeDedupKey("q1:terminal") {
		t.Error("should find via Recent normalized dedup key")
	}
}

func TestAppendOutcome(t *testing.T) {
	s := &AutomationTrustState{AutomationID: "a1", Tier: TrustTier1}

	o1 := AutomationTrustOutcome{QuestID: "q1", Outcome: "success", OutcomeType: TrustOutcomeTypeTerminal, Independent: true, TsMs: 100}
	if !s.AppendOutcome(o1) {
		t.Fatal("first append should succeed")
	}
	if s.TotalRuns != 1 || s.TotalSuccess != 1 || s.ConsecutiveIndependentSuccess != 1 {
		t.Errorf("after success: TotalRuns=%d TotalSuccess=%d CIS=%d", s.TotalRuns, s.TotalSuccess, s.ConsecutiveIndependentSuccess)
	}

	// 重复 key 应被拒绝
	if s.AppendOutcome(o1) {
		t.Error("duplicate dedup key should be rejected")
	}

	oFail := AutomationTrustOutcome{QuestID: "q2", Outcome: "failure", OutcomeType: TrustOutcomeTypeTerminal, TsMs: 200}
	if !s.AppendOutcome(oFail) {
		t.Fatal("failure append should succeed")
	}
	if s.TotalFailures != 1 || s.ConsecutiveFailures != 1 {
		t.Errorf("after failure: TotalFailures=%d CF=%d", s.TotalFailures, s.ConsecutiveFailures)
	}
	if s.ConsecutiveIndependentSuccess != 0 {
		t.Errorf("CIS should reset to 0 after failure, got %d", s.ConsecutiveIndependentSuccess)
	}

	// stage 类型不计数 TotalRuns
	oStage := AutomationTrustOutcome{QuestID: "q3", Outcome: "success", OutcomeType: TrustOutcomeTypeStage, VerificationType: "build", TsMs: 300}
	if !s.AppendOutcome(oStage) {
		t.Fatal("stage append should succeed")
	}
	if s.TotalRuns != 2 {
		t.Errorf("stage outcome should not affect TotalRuns, got %d", s.TotalRuns)
	}

	// TsMs=0 时自动填充
	oNoTs := AutomationTrustOutcome{QuestID: "q10", Outcome: "success", OutcomeType: TrustOutcomeTypeTerminal}
	if !s.AppendOutcome(oNoTs) {
		t.Fatal("no-ts append should succeed")
	}
	// 找到这条 recent
	found := false
	for _, r := range s.Recent {
		if r.QuestID == "q10" && r.TsMs > 0 {
			found = true
			break
		}
	}
	if !found {
		t.Error("AppendOutcome should auto-fill TsMs when 0")
	}

	// nil 安全
	var ns *AutomationTrustState
	if ns.AppendOutcome(o1) {
		t.Error("nil state AppendOutcome should be false")
	}
}

// --- 小工具 ---

func itoa(i int) string {
	// 避免引入 strconv 依赖
	buf := make([]byte, 0, 10)
	neg := i < 0
	if neg {
		i = -i
	}
	if i == 0 {
		buf = append(buf, '0')
	}
	for i > 0 {
		buf = append([]byte{byte('0' + i%10)}, buf...)
		i /= 10
	}
	if neg {
		buf = append([]byte{'-'}, buf...)
	}
	return string(buf)
}

// 仅用于验证 withDefaults 行为——测试 AllowedCommand 默认值不被破坏
func TestWithDefaults_PreservesCustomValues(t *testing.T) {
	c := AllowedCommand{
		ID:              "custom",
		Description:     "my desc",
		Command:         "mycmd",
		Args:            []string{"a", "b"},
		TimeoutMs:       42,
		SideEffectLevel: "L2",
	}
	def := c.withDefaults()
	if def.TimeoutMs != 42 {
		t.Errorf("custom TimeoutMs should be preserved, got %d", def.TimeoutMs)
	}
	if def.SideEffectLevel != "L2" {
		t.Errorf("custom SideEffectLevel should be preserved, got %s", def.SideEffectLevel)
	}
	if !reflect.DeepEqual(def.Args, []string{"a", "b"}) {
		t.Errorf("custom Args should be preserved, got %v", def.Args)
	}
}

func TestWithDefaults_FillsDefaults(t *testing.T) {
	c := AllowedCommand{ID: "x", Command: "go"}
	def := c.withDefaults()
	if def.TimeoutMs == 0 {
		t.Errorf("TimeoutMs should get a default value, got 0")
	}
	if def.SideEffectLevel == "" {
		t.Errorf("SideEffectLevel should get a default value")
	}
}
