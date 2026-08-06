package fsstore

import (
	"strings"
	"testing"
)

func TestInitDefaultAutomations_ContextRefreshPolicy(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := root.InitDefaultAutomations(); err != nil {
		t.Fatalf("InitDefaultAutomations failed: %v", err)
	}

	cfg, err := root.GetAutomation("auto_context_refresh")
	if err != nil {
		t.Fatalf("auto_context_refresh missing: %v", err)
	}
	if !cfg.Enabled {
		t.Fatal("context refresh should be enabled by default")
	}
	if cfg.Priority != 100 {
		t.Fatalf("context refresh priority = %d, want 100", cfg.Priority)
	}
	if !cfg.AutoStart {
		t.Fatal("context refresh should auto start and skip inbox")
	}
	if cfg.AutoApply {
		t.Fatal("context refresh should not auto apply by default")
	}
	if cfg.Query != defaultContextRefreshQuery {
		t.Fatal("context refresh should use structured context extraction prompt")
	}
	if !strings.Contains(cfg.Query, "正文默认使用中文撰写") {
		t.Fatal("context refresh prompt should require Chinese body text")
	}
	if !strings.Contains(cfg.Query, "覆盖写入，不是无限追加日志") {
		t.Fatal("context refresh prompt should define overwrite vs accumulation semantics")
	}
	if !strings.Contains(cfg.Query, "Candidate Commitment Signals (Experimental)") {
		t.Fatal("context refresh prompt should include the Phase 0 commitment signal experiment")
	}
	if !strings.Contains(cfg.Query, "lark-cli task +get-my-tasks --as user --complete=false --page-all") {
		t.Fatal("context refresh prompt should use the verified Lark task command")
	}
	if strings.Contains(cfg.Query, "写入 dim_commitment_signals") ||
		strings.Contains(cfg.Query, "新增 dim_commitment_signals") {
		t.Fatal("Phase 0 must not create an independent commitment_signals dimension")
	}

	items, err := root.ListAutomations()
	if err != nil {
		t.Fatalf("ListAutomations failed: %v", err)
	}
	if len(items) == 0 || items[0].ID != "auto_context_refresh" {
		t.Fatalf("highest priority automation should be first, got %+v", items)
	}
}

func TestAutomationTrustStateAppendOutcomeDeduplicatesTerminalOutcome(t *testing.T) {
	state := &AutomationTrustState{AutomationID: "auto_test", Tier: TrustTier1}
	first := AutomationTrustOutcome{
		QuestID:          "qst_1",
		OutcomeType:      TrustOutcomeTypeTerminal,
		Outcome:          "failure",
		VerificationType: "apply_failed",
		Independent:      true,
	}
	if !state.AppendOutcome(first) {
		t.Fatal("first terminal outcome should be recorded")
	}
	duplicate := first
	duplicate.VerificationType = "human_rejected"
	if state.AppendOutcome(duplicate) {
		t.Fatal("duplicate terminal outcome for same quest should be ignored")
	}
	if state.TotalRuns != 1 || state.TotalFailures != 1 || state.ConsecutiveFailures != 1 {
		t.Fatalf("terminal outcome counters should count once: %+v", state)
	}
	if len(state.Recent) != 1 || len(state.OutcomeKeys) != 1 {
		t.Fatalf("dedup state should contain one outcome: %+v", state)
	}
}

func TestAutomationTrustStateAppendOutcomeKeepsStageOutcomesOutOfCounters(t *testing.T) {
	state := &AutomationTrustState{AutomationID: "auto_test", Tier: TrustTier1}
	stage := AutomationTrustOutcome{
		QuestID:          "qst_1",
		OutcomeType:      TrustOutcomeTypeStage,
		Outcome:          "failure",
		VerificationType: "apply_failed",
		Independent:      true,
	}
	if !state.AppendOutcome(stage) {
		t.Fatal("stage outcome should be recorded")
	}
	if state.TotalRuns != 0 || state.TotalFailures != 0 || state.ConsecutiveFailures != 0 {
		t.Fatalf("stage outcome should not affect terminal counters: %+v", state)
	}
	if state.AppendOutcome(stage) {
		t.Fatal("duplicate stage outcome should be ignored")
	}
	if len(state.Recent) != 1 || len(state.OutcomeKeys) != 1 {
		t.Fatalf("dedup state should contain one stage outcome: %+v", state)
	}
}

func TestInitDefaultAutomations_BackfillsMissingOfficialAutomations(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := root.SaveAutomation(&AutomationConfig{
		ID:      "user_custom",
		Name:    "user custom",
		Enabled: true,
		Trigger: TriggerManual,
		Query:   "user defined automation",
	}); err != nil {
		t.Fatalf("SaveAutomation failed: %v", err)
	}

	created, err := root.InitDefaultAutomations()
	if err != nil {
		t.Fatalf("InitDefaultAutomations failed: %v", err)
	}
	if len(created) != 13 {
		t.Fatalf("created defaults = %v, want 13 official automations", created)
	}
	if _, err := root.GetAutomation("auto_inbox_triage"); err != nil {
		t.Fatalf("auto_inbox_triage should be backfilled: %v", err)
	}
	if _, err := root.GetAutomation("auto_context_refresh"); err != nil {
		t.Fatalf("auto_context_refresh should be backfilled: %v", err)
	}
	knowledge, err := root.GetAutomation("auto_knowledge_pack_refresh")
	if err != nil {
		t.Fatalf("auto_knowledge_pack_refresh should be backfilled: %v", err)
	}
	if knowledge.Enabled || knowledge.AutoStart || knowledge.AutoApply || knowledge.AllowL2 {
		t.Fatalf("knowledge pack automation should be disabled and non-side-effecting by default: %+v", knowledge)
	}
	codebase, err := root.GetAutomation("auto_codebase_map_refresh")
	if err != nil {
		t.Fatalf("auto_codebase_map_refresh should be backfilled: %v", err)
	}
	if codebase.Enabled || codebase.AutoStart || codebase.AutoApply || codebase.AllowL2 {
		t.Fatalf("codebase map automation should be disabled and non-side-effecting by default: %+v", codebase)
	}
	if codebase.Trigger != TriggerManual {
		t.Fatalf("codebase map automation should be manual trigger by default: %+v", codebase)
	}
	if codebase.Query != defaultCodebaseMapRefreshQuery {
		t.Fatalf("codebase map automation should use default query")
	}
}

func TestInitDefaultAutomations_UpgradesLegacyContextRefreshPrompt(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	existing := &AutomationConfig{
		ID:          "auto_context_refresh",
		Name:        "用户工作上下文刷新",
		Description: "legacy",
		Enabled:     true,
		Trigger:     TriggerSchedule,
		Cron:        "0 9 * * *",
		Priority:    100,
		Query:       legacyContextRefreshQuery,
		QuestType:   "execute",
		AutoStart:   true,
		AutoApply:   true,
		Tags:        []string{"official", "template", "context"},
	}
	if err := root.SaveAutomation(existing); err != nil {
		t.Fatalf("SaveAutomation failed: %v", err)
	}

	created, err := root.InitDefaultAutomations()
	if err != nil {
		t.Fatalf("InitDefaultAutomations failed: %v", err)
	}
	found := false
	for _, path := range created {
		if path == "automation/auto_context_refresh.json" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected context refresh upgrade marker, got %v", created)
	}
	got, err := root.GetAutomation("auto_context_refresh")
	if err != nil {
		t.Fatalf("GetAutomation failed: %v", err)
	}
	if got.Query != defaultContextRefreshQuery {
		t.Fatal("legacy context refresh query was not upgraded")
	}
	if got.AutoApply {
		t.Fatal("context refresh upgrade should force auto_apply=false")
	}
	if !got.AutoStart {
		t.Fatal("context refresh upgrade should force auto_start=true")
	}
}

func TestInitDefaultAutomations_UpgradesContextRefreshRuntimePolicy(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	existing := &AutomationConfig{
		ID:          "auto_context_refresh",
		Name:        "用户工作上下文刷新",
		Description: "current prompt with stale runtime policy",
		Enabled:     true,
		Trigger:     TriggerSchedule,
		Cron:        "0 9 * * *",
		Priority:    100,
		Query:       defaultContextRefreshQuery,
		QuestType:   "execute",
		AutoStart:   false,
		AutoApply:   true,
		Tags:        []string{"official", "template", "context"},
	}
	if err := root.SaveAutomation(existing); err != nil {
		t.Fatalf("SaveAutomation failed: %v", err)
	}

	created, err := root.InitDefaultAutomations()
	if err != nil {
		t.Fatalf("InitDefaultAutomations failed: %v", err)
	}
	found := false
	for _, path := range created {
		if path == "automation/auto_context_refresh.json" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected context refresh runtime policy upgrade marker, got %v", created)
	}
	got, err := root.GetAutomation("auto_context_refresh")
	if err != nil {
		t.Fatalf("GetAutomation failed: %v", err)
	}
	if !got.AutoStart {
		t.Fatal("context refresh should auto start and skip inbox after upgrade")
	}
	if got.AutoApply {
		t.Fatal("context refresh should not auto apply after upgrade")
	}
}

func TestInitDefaultAutomations_UpgradesScheduledOfficialRuntimePolicy(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{
		"auto_morning_briefing",
		"auto_stale_quest_cleanup",
		"auto_context_decay_alert",
		"auto_inbox_triage",
	} {
		if err := root.SaveAutomation(&AutomationConfig{
			ID:        id,
			Name:      "stale " + id,
			Enabled:   true,
			Source:    SourceOfficial,
			Trigger:   TriggerSchedule,
			Cron:      "0 9 * * *",
			Query:     "custom query",
			QuestType: "design",
			AutoStart: false,
			AutoApply: true,
			Tags:      []string{"official", "template"},
		}); err != nil {
			t.Fatalf("SaveAutomation(%s) failed: %v", id, err)
		}
	}

	created, err := root.InitDefaultAutomations()
	if err != nil {
		t.Fatalf("InitDefaultAutomations failed: %v", err)
	}
	for _, id := range []string{
		"auto_morning_briefing",
		"auto_stale_quest_cleanup",
		"auto_context_decay_alert",
		"auto_inbox_triage",
	} {
		relPath := "automation/" + id + ".json"
		found := false
		for _, path := range created {
			if path == relPath {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected %s runtime policy upgrade marker, got %v", id, created)
		}
		got, err := root.GetAutomation(id)
		if err != nil {
			t.Fatalf("GetAutomation(%s) failed: %v", id, err)
		}
		if !got.AutoStart {
			t.Fatalf("%s should auto start after upgrade", id)
		}
		if got.AutoApply {
			t.Fatalf("%s should not auto apply after upgrade", id)
		}
		if got.OfficialPolicyVersion != CurrentOfficialAutomationPolicyVersion {
			t.Fatalf("%s policy version = %d, want %d", id, got.OfficialPolicyVersion, CurrentOfficialAutomationPolicyVersion)
		}
		if got.Query != "custom query" {
			t.Fatalf("%s query should not be overwritten: %q", id, got.Query)
		}
	}
}

func TestInitDefaultAutomations_RespectsRuntimePolicyUserOverride(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := root.SaveAutomation(&AutomationConfig{
		ID:                        "auto_morning_briefing",
		Name:                      "morning with explicit inbox gate",
		Enabled:                   true,
		Source:                    SourceOfficial,
		Trigger:                   TriggerSchedule,
		Cron:                      "30 9 * * 1-5",
		Query:                     "custom query",
		QuestType:                 "design",
		AutoStart:                 false,
		AutoApply:                 false,
		RuntimePolicyUserOverride: true,
		Tags:                      []string{"official", "template", "briefing"},
	}); err != nil {
		t.Fatalf("SaveAutomation failed: %v", err)
	}

	created, err := root.InitDefaultAutomations()
	if err != nil {
		t.Fatalf("InitDefaultAutomations failed: %v", err)
	}
	found := false
	for _, path := range created {
		if path == "automation/auto_morning_briefing.json" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected policy version upgrade marker, got %v", created)
	}
	got, err := root.GetAutomation("auto_morning_briefing")
	if err != nil {
		t.Fatalf("GetAutomation failed: %v", err)
	}
	if got.AutoStart {
		t.Fatal("user-overridden runtime policy should remain inbox-confirmed")
	}
	if got.OfficialPolicyVersion != CurrentOfficialAutomationPolicyVersion {
		t.Fatalf("policy version = %d, want %d", got.OfficialPolicyVersion, CurrentOfficialAutomationPolicyVersion)
	}
	if !got.RuntimePolicyUserOverride {
		t.Fatal("runtime policy override marker should be preserved")
	}
}

func TestInitDefaultAutomations_DoesNotUpgradeManualOfficialRuntimePolicy(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{
		"auto_daily_lint",
		"auto_code_review",
		"auto_dependency_check",
		"auto_workspace_drift_detect",
		"auto_audit_completed",
		"auto_quest_pattern_report",
		"auto_knowledge_pack_refresh",
	} {
		if err := root.SaveAutomation(&AutomationConfig{
			ID:        id,
			Name:      "manual " + id,
			Enabled:   false,
			Source:    SourceOfficial,
			Trigger:   TriggerManual,
			Query:     "manual query",
			QuestType: "design",
			AutoStart: false,
			AutoApply: false,
			Tags:      []string{"official", "template"},
		}); err != nil {
			t.Fatalf("SaveAutomation(%s) failed: %v", id, err)
		}
	}

	if _, err := root.InitDefaultAutomations(); err != nil {
		t.Fatalf("InitDefaultAutomations failed: %v", err)
	}
	for _, id := range []string{
		"auto_daily_lint",
		"auto_code_review",
		"auto_dependency_check",
		"auto_workspace_drift_detect",
		"auto_audit_completed",
		"auto_quest_pattern_report",
		"auto_knowledge_pack_refresh",
	} {
		got, err := root.GetAutomation(id)
		if err != nil {
			t.Fatalf("GetAutomation(%s) failed: %v", id, err)
		}
		if got.AutoStart {
			t.Fatalf("%s should remain inbox-confirmed", id)
		}
		if got.OfficialPolicyVersion != CurrentOfficialAutomationPolicyVersion {
			t.Fatalf("%s policy version = %d, want %d", id, got.OfficialPolicyVersion, CurrentOfficialAutomationPolicyVersion)
		}
	}
}

func TestInitDefaultAutomations_UpgradesStructuredContextPromptWithoutLanguagePolicy(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	structuredWithoutLanguage := strings.Replace(defaultContextRefreshQuery, `语言要求：
- 章节标题必须保持上述英文名称，作为稳定 schema anchor。
- 正文默认使用中文撰写，方便中文用户和飞书场景直接消费。
- 技术名词、命令、文件路径、API、状态枚举、产品名可以保留英文。

刷新/沉淀语义：
- context_write_dim 和 context_write_summary 是覆盖写入，不是无限追加日志。
- Current Focus / Recent Decisions / lark_calendar 等近期事实按本轮时间窗口刷新，旧的短期事实不应长期保留。
- Stable Preferences / Project Contracts / Anti-Patterns / Retrieval Index 可以跨天保留，但必须根据本轮证据修正、合并或删除过时项。
- 需要长期沉淀的稳定规则应写成可复用原则；不要把每日聊天、日程和临时任务堆成历史流水账。

`, "", 1)
	if err := root.SaveAutomation(&AutomationConfig{
		ID:        "auto_context_refresh",
		Name:      "用户工作上下文刷新",
		Enabled:   true,
		Trigger:   TriggerSchedule,
		Query:     structuredWithoutLanguage,
		AutoStart: true,
		Tags:      []string{"official", "template", "context"},
	}); err != nil {
		t.Fatalf("SaveAutomation failed: %v", err)
	}

	if _, err := root.InitDefaultAutomations(); err != nil {
		t.Fatalf("InitDefaultAutomations failed: %v", err)
	}
	got, err := root.GetAutomation("auto_context_refresh")
	if err != nil {
		t.Fatalf("GetAutomation failed: %v", err)
	}
	if got.Query != defaultContextRefreshQuery {
		t.Fatal("structured context refresh query without language policy was not upgraded")
	}
}

func TestInitDefaultAutomations_UpgradesContextPromptWithoutCommitmentSignalsExperiment(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	withoutExperiment := strings.Replace(defaultContextRefreshQuery, `  - Phase 0 实验：如 user 身份和飞书任务权限可用，可在 lark_im 的 Current Focus 下新增三级标题 "### Candidate Commitment Signals (Experimental)"，只从飞书任务提取最多 3 条候选承诺信号。
  - 命令：lark-cli task +get-my-tasks --as user --complete=false --page-all。若命令失败、无授权或任务数为 0，写入 Gaps 说明，不要编造信号。
  - 字段：Action 摘要、时间暗示、来源类型、置信度、不确定原因、核验关键词。来源类型固定为 lark_task，置信度可为 高/中高。
  - 边界：候选信号仅用于二次核验，不作为行动事实；不得创建 dim_commitment_signals 独立维度；不得写状态、优先级、task_guid、URL、open_id、群名、消息链接或原始标题全文。
  - 展示顺序仅用于可读性：时间暗示更近、置信度更高的信号靠前；不得写入 priority 字段或暗示已确认承诺。
`, "", 1)
	if err := root.SaveAutomation(&AutomationConfig{
		ID:        "auto_context_refresh",
		Name:      "用户工作上下文刷新",
		Enabled:   true,
		Trigger:   TriggerSchedule,
		Query:     withoutExperiment,
		AutoStart: true,
		Tags:      []string{"official", "template", "context"},
	}); err != nil {
		t.Fatalf("SaveAutomation failed: %v", err)
	}

	if _, err := root.InitDefaultAutomations(); err != nil {
		t.Fatalf("InitDefaultAutomations failed: %v", err)
	}
	got, err := root.GetAutomation("auto_context_refresh")
	if err != nil {
		t.Fatalf("GetAutomation failed: %v", err)
	}
	if got.Query != defaultContextRefreshQuery {
		t.Fatal("context refresh query without commitment signal experiment was not upgraded")
	}
}

func TestInitDefaultAutomations_DoesNotOverwriteExistingOfficialAutomation(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	existing := &AutomationConfig{
		ID:        "auto_daily_lint",
		Name:      "custom daily lint",
		Enabled:   false,
		Trigger:   TriggerManual,
		Cron:      "30 8 * * *",
		Query:     "custom query",
		AutoStart: false,
		Tags:      []string{"official", "template"},
	}
	if err := root.SaveAutomation(existing); err != nil {
		t.Fatalf("SaveAutomation failed: %v", err)
	}

	if _, err := root.InitDefaultAutomations(); err != nil {
		t.Fatalf("InitDefaultAutomations failed: %v", err)
	}
	got, err := root.GetAutomation("auto_daily_lint")
	if err != nil {
		t.Fatalf("GetAutomation failed: %v", err)
	}
	if got.Name != existing.Name || got.Enabled != existing.Enabled || got.Cron != existing.Cron || got.Query != existing.Query || got.AutoStart != existing.AutoStart {
		t.Fatalf("existing official automation was overwritten: got %+v", got)
	}
}

func TestSaveAutomation_NormalizesCronPreset(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cfg := &AutomationConfig{
		ID:      "preset_daily9",
		Name:    "preset",
		Enabled: true,
		Trigger: TriggerSchedule,
		Cron:    "daily9",
		Query:   "test",
	}
	if err := root.SaveAutomation(cfg); err != nil {
		t.Fatalf("SaveAutomation failed: %v", err)
	}
	got, err := root.GetAutomation("preset_daily9")
	if err != nil {
		t.Fatalf("GetAutomation failed: %v", err)
	}
	if got.Cron != "0 9 * * *" {
		t.Fatalf("cron = %q, want %q", got.Cron, "0 9 * * *")
	}
}
