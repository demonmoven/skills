package fsstore

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

const DefaultPersonalStatsLimit = 20
const lateFailureThresholdMs = int64(30 * 60 * 1000)

type PersonalStats struct {
	GeneratedAtMs     int64                    `json:"generated_at_ms"`
	SinceMs           int64                    `json:"since_ms,omitempty"`
	TopNLimit         int                      `json:"topn_limit"`
	TotalQuests       int                      `json:"total_quests"`
	StatusCounts      map[string]int           `json:"status_counts"`
	TypeCounts        map[string]int           `json:"type_counts"`
	ActiveQuests      int                      `json:"active_quests"`
	TerminalQuests    int                      `json:"terminal_quests"`
	SuccessQuests     int                      `json:"success_quests"`
	AutoClosedQuests  int                      `json:"auto_closed_quests,omitempty"` // HOTL: finalized_by=policy 的 success quest（自主闭环）
	FailedQuests      int                      `json:"failed_quests"`
	CancelledQuests   int                      `json:"cancelled_quests"`
	BlockedQuests     int                      `json:"blocked_quests"`
	SuccessRate       float64                  `json:"success_rate"`
	TotalDurationMs   int64                    `json:"total_duration_ms"`
	P50DurationMs     int64                    `json:"p50_duration_ms"`
	P95DurationMs     int64                    `json:"p95_duration_ms"`
	TotalTurns        int64                    `json:"total_turns"`
	TotalTokensIn     int64                    `json:"total_tokens_in"`
	TotalTokensOut    int64                    `json:"total_tokens_out"`
	TotalTokens       int64                    `json:"total_tokens"`
	AvgTokensPerRun   float64                  `json:"avg_tokens_per_run"`
	ProjectStats      []PersonalProjectStat    `json:"project_stats"`
	PhaseStats        []PersonalPhaseStat      `json:"phase_stats"`
	FailureReasons    []PersonalCount          `json:"failure_reasons"`
	FailureCategories []PersonalCount          `json:"failure_categories,omitempty"`
	AnomalySignals    []PersonalAnomalySignal  `json:"anomaly_signals"`
	SlowOperations    []PersonalSlowOperation  `json:"slow_operations"`
	RecentRuns        []PersonalRecentRun      `json:"recent_runs"`
	LoopHealth        PersonalLoopHealth       `json:"loop_health"`
	DataCompleteness  PersonalDataCompleteness `json:"data_completeness"`
}

type PersonalLoopHealth struct {
	StartedSessions              int     `json:"started_sessions"`
	FirstOutputSessions          int     `json:"first_output_sessions"`
	FirstOutputCoverage          float64 `json:"first_output_coverage"`
	LateFailureCount             int     `json:"late_failure_count"`
	HumanExceptionOpenCount      int     `json:"human_exception_open_count"`
	HumanExceptionOpenAgeMs      int64   `json:"human_exception_open_age_ms"`
	MakerPhaseDoneCount          int     `json:"maker_phase_done_count"`
	MakerReportCount             int     `json:"maker_report_count"`
	MakerReportCoverage          float64 `json:"maker_report_coverage"`
	ReviewPhaseDoneCount         int     `json:"review_phase_done_count"`
	ReviewReportCount            int     `json:"review_report_count"`
	ReviewReportCoverage         float64 `json:"review_report_coverage"`
	EvidenceBackedReviewCount    int     `json:"evidence_backed_review_count"`
	EvidenceBackedReviewCoverage float64 `json:"evidence_backed_review_coverage"`
	AutomationNoFindingCount     int     `json:"automation_no_finding_count"`
	AutomationCandidateCount     int     `json:"automation_candidate_count"`
	AutomationNoFindingRate      float64 `json:"automation_no_finding_rate"`
	TierCSessionCount            int     `json:"tier_c_session_count"`
	TierCUsage                   float64 `json:"tier_c_usage"`
}

type PersonalProjectStat struct {
	Project   string            `json:"project"`
	Count     int               `json:"count"`
	Drilldown PersonalDrilldown `json:"drilldown,omitempty"`
}

type PersonalPhaseStat struct {
	Phase      string  `json:"phase"`
	Count      int     `json:"count"`
	TotalMs    int64   `json:"total_ms"`
	AvgMs      float64 `json:"avg_ms"`
	P95Ms      int64   `json:"p95_ms"`
	TokensIn   int64   `json:"tokens_in"`
	TokensOut  int64   `json:"tokens_out"`
	Turns      int64   `json:"turns"`
	ErrorCount int     `json:"error_count"`
	Timeouts   int     `json:"timeouts"`
	NoProgress int     `json:"no_progress"`
}

type PersonalCount struct {
	Name      string            `json:"name"`
	Count     int               `json:"count"`
	Drilldown PersonalDrilldown `json:"drilldown,omitempty"`
}

type PersonalSlowOperation struct {
	QuestID    string            `json:"quest_id"`
	ShortID    string            `json:"short_id,omitempty"`
	SessionID  string            `json:"session_id,omitempty"`
	Kind       string            `json:"kind"`
	Name       string            `json:"name"`
	DurationMs int64             `json:"duration_ms"`
	Status     string            `json:"status,omitempty"`
	Timestamp  int64             `json:"timestamp"`
	Drilldown  PersonalDrilldown `json:"drilldown,omitempty"`
}

type PersonalDrilldown struct {
	Query map[string][]string `json:"query,omitempty"`
}

type PersonalAnomalySignal struct {
	Key       string            `json:"key"`
	Label     string            `json:"label"`
	Severity  string            `json:"severity"`
	Count     int               `json:"count"`
	Drilldown PersonalDrilldown `json:"drilldown,omitempty"`
}

type PersonalRecentRun struct {
	QuestID        string            `json:"quest_id"`
	ShortID        string            `json:"short_id,omitempty"`
	Query          string            `json:"query"`
	Type           model.QuestType   `json:"type"`
	Status         model.QuestStatus `json:"status"`
	Project        string            `json:"project,omitempty"`
	CreatedAtMs    int64             `json:"created_at_ms"`
	StartedAtMs    int64             `json:"started_at_ms,omitempty"`
	CompletedAtMs  int64             `json:"completed_at_ms,omitempty"`
	DurationMs     int64             `json:"duration_ms,omitempty"`
	Turns          int64             `json:"turns"`
	TokensIn       int64             `json:"tokens_in"`
	TokensOut      int64             `json:"tokens_out"`
	FailureReason  string            `json:"failure_reason,omitempty"`
	WorkspaceClean bool              `json:"workspace_cleaned,omitempty"`
}

type PersonalDataCompleteness struct {
	QuestMeta        bool `json:"quest_meta"`
	SessionRows      bool `json:"session_rows"`
	PlanPhaseTimings bool `json:"plan_phase_timings"`
	TokenUsage       bool `json:"token_usage"`
	CommandEvents    bool `json:"command_events"`
}

type personalPhaseAccum struct {
	stat      PersonalPhaseStat
	durations []int64
}

// AggregatePersonalStats builds a local, read-only observability snapshot from
// files Gloop already writes under workspace/quests. It intentionally avoids a
// second metrics store so the personal edition can inspect existing data
// without changing the write path.
func AggregatePersonalStats(root *Root, sinceMs int64, limit int) (*PersonalStats, error) {
	if root == nil {
		return nil, fmt.Errorf("root is nil")
	}
	if limit <= 0 {
		limit = DefaultPersonalStatsLimit
	}
	now := NowMs()
	qs := NewQuestStore(root)
	quests, err := qs.ListQuests()
	if err != nil {
		return nil, err
	}
	out := &PersonalStats{
		GeneratedAtMs:     now,
		SinceMs:           sinceMs,
		TopNLimit:         limit,
		StatusCounts:      map[string]int{},
		TypeCounts:        map[string]int{},
		ProjectStats:      []PersonalProjectStat{},
		PhaseStats:        []PersonalPhaseStat{},
		FailureReasons:    []PersonalCount{},
		FailureCategories: []PersonalCount{},
		AnomalySignals:    []PersonalAnomalySignal{},
		SlowOperations:    []PersonalSlowOperation{},
		RecentRuns:        []PersonalRecentRun{},
		DataCompleteness: PersonalDataCompleteness{
			QuestMeta: true,
		},
	}
	projectCounts := map[string]int{}
	failureCounts := map[string]int{}
	failureCategoryCounts := map[string]int{}
	phaseAccums := map[int]*personalPhaseAccum{}
	var terminalDurations []int64
	var recent []PersonalRecentRun
	var slow []PersonalSlowOperation
	discoveryArchive, _ := root.LoadAutomationDiscoveryArchive()
	for _, item := range discoveryArchive {
		if sinceMs > 0 && item.CreatedAtMs > 0 && item.CreatedAtMs < sinceMs {
			continue
		}
		if item.Outcome == "no_finding" {
			out.LoopHealth.AutomationNoFindingCount++
		}
	}

	for _, q := range quests {
		if q == nil || (sinceMs > 0 && q.CreatedAtMs < sinceMs) {
			continue
		}
		out.TotalQuests++
		out.StatusCounts[string(q.Status)]++
		out.TypeCounts[string(q.Type)]++
		project := questProject(q)
		if project != "" {
			projectCounts[project]++
		}
		if q.IsTerminal() {
			out.TerminalQuests++
		} else {
			out.ActiveQuests++
		}
		if HumanExceptionFromQuest(q) != nil {
			out.LoopHealth.HumanExceptionOpenCount++
			since := q.UpdatedAtMs
			if since == 0 {
				since = q.CreatedAtMs
			}
			if since > 0 && now > since {
				age := now - since
				if age > out.LoopHealth.HumanExceptionOpenAgeMs {
					out.LoopHealth.HumanExceptionOpenAgeMs = age
				}
			}
		}
		if strings.HasPrefix(q.CreatedBy, "automation:") && q.Status == model.QuestStatusPending && AutomationTriageMode(&AutomationConfig{AutoStart: false, TriageMode: q.TriageMode}) == "candidate" {
			out.LoopHealth.AutomationCandidateCount++
		}
		switch q.Status {
		case model.QuestStatusSuccess:
			out.SuccessQuests++
			if q.FinalizedBy == "policy" {
				out.AutoClosedQuests++
			}
		case model.QuestStatusFailed:
			out.FailedQuests++
		case model.QuestStatusCancelled:
			out.CancelledQuests++
		case model.QuestStatusBlocked:
			out.BlockedQuests++
		}

		run := PersonalRecentRun{
			QuestID:        q.ID,
			ShortID:        q.ShortID,
			Query:          q.Query,
			Type:           q.Type,
			Status:         q.Status,
			Project:        project,
			CreatedAtMs:    q.CreatedAtMs,
			StartedAtMs:    q.StartedAtMs,
			CompletedAtMs:  q.CompletedAtMs,
			WorkspaceClean: q.WorkspaceCleaned,
		}
		if d := questDurationMs(q, now); d > 0 {
			run.DurationMs = d
			if q.IsTerminal() {
				out.TotalDurationMs += d
				terminalDurations = append(terminalDurations, d)
			}
			if (q.Status == model.QuestStatusFailed || q.Status == model.QuestStatusBlocked) && d >= lateFailureThresholdMs {
				out.LoopHealth.LateFailureCount++
			}
		}
		hasQuestFailureAttribution := q.FailureAttribution != nil && q.FailureAttribution.Reason != ""
		if hasQuestFailureAttribution {
			run.FailureReason = q.FailureAttribution.Reason
			failureCounts[q.FailureAttribution.Reason]++
			if q.FailureAttribution.Category != "" {
				failureCategoryCounts[q.FailureAttribution.Category]++
			}
		}

		plan, _ := qs.LoadPlan(q.ID)
		if len(plan) > 0 {
			out.DataCompleteness.PlanPhaseTimings = true
		}
		for _, phase := range plan {
			if phase.Status == model.PhaseDone {
				if phase.Role == "warrior" || phase.Class == model.ClassWarrior {
					out.LoopHealth.MakerPhaseDoneCount++
				}
				if phase.Role == "mage" || phase.Role == "review" || phase.Class == model.ClassMage {
					out.LoopHealth.ReviewPhaseDoneCount++
				}
			}
			if phase.StartedAtMs <= 0 || phase.EndedAtMs <= phase.StartedAtMs {
				continue
			}
			acc := getPhaseAccum(phaseAccums, phase.PhaseIdx, phaseLabel(phase.PhaseIdx, phase.Role))
			d := phase.EndedAtMs - phase.StartedAtMs
			acc.stat.Count++
			acc.stat.TotalMs += d
			acc.durations = append(acc.durations, d)
		}

		sessionIDs, _ := qs.ListSessionIDs(q.ID)
		sessionStates, _ := root.ListAgentSessionStates(q.ID)
		for _, state := range sessionStates {
			if state == nil {
				continue
			}
			if state.StartedAtMs > 0 {
				out.LoopHealth.StartedSessions++
			}
			if state.FirstOutputAtMs > 0 {
				out.LoopHealth.FirstOutputSessions++
			}
			if state.CapabilityTier == "tier_c" {
				out.LoopHealth.TierCSessionCount++
			}
		}
		for _, sid := range sessionIDs {
			rows, rErr := qs.ReadSessionRows(q.ID, sid, 0)
			if rErr != nil || len(rows) == 0 {
				continue
			}
			out.DataCompleteness.SessionRows = true
			for _, row := range rows {
				phase := int(row.Phase)
				acc := getPhaseAccum(phaseAccums, phase, phaseLabel(phase, ""))
				switch row.Kind {
				case "message":
					if row.Role == "assistant" {
						run.Turns++
						out.TotalTurns++
						tin := int64FromMeta(row.Meta, "token_input")
						tout := int64FromMeta(row.Meta, "token_output")
						dur := int64FromMeta(row.Meta, "duration_ms")
						if tin != 0 || tout != 0 {
							out.DataCompleteness.TokenUsage = true
						}
						run.TokensIn += tin
						run.TokensOut += tout
						out.TotalTokensIn += tin
						out.TotalTokensOut += tout
						acc.stat.Turns++
						acc.stat.TokensIn += tin
						acc.stat.TokensOut += tout
						if dur > 0 {
							slow = append(slow, PersonalSlowOperation{
								QuestID: q.ID, ShortID: q.ShortID, SessionID: sid,
								Kind: "model_turn", Name: phaseLabel(phase, ""),
								DurationMs: dur, Status: stringValueFromMeta(row.Meta, "finish_reason"),
								Timestamp: row.Timestamp,
							})
						}
					}
				case "error":
					acc.stat.ErrorCount++
					if hasQuestFailureAttribution {
						continue
					}
					reason := normalizeReason(row.Error)
					if reason == "" {
						reason = normalizeReason(row.Content)
					}
					if reason != "" {
						failureCounts[reason]++
						if run.FailureReason == "" {
							run.FailureReason = reason
						}
					}
				case "budget_exceeded":
					acc.stat.Timeouts++
					if !hasQuestFailureAttribution {
						failureCounts["duration_exceeded"]++
						run.FailureReason = "duration_exceeded"
					}
				case "no_progress":
					acc.stat.NoProgress++
					if !hasQuestFailureAttribution {
						failureCounts["no_progress"]++
						run.FailureReason = "no_progress"
					}
				}
			}
		}

		events, _ := qs.ReadEvents(q.ID, 0)
		for _, ev := range events {
			if ev.Type != "micro.command_run" {
				continue
			}
			out.DataCompleteness.CommandEvents = true
			payload, ok := ev.Payload.(map[string]any)
			if !ok {
				continue
			}
			dur := int64FromAny(payload["duration_ms"])
			name := stringFromAny(payload["command_id"])
			status := "ok"
			if timedOut, _ := payload["timed_out"].(bool); timedOut {
				status = "timeout"
				if !hasQuestFailureAttribution {
					failureCounts["command_timeout"]++
				}
			} else if exit := int64FromAny(payload["exit_code"]); exit != 0 {
				status = fmt.Sprintf("exit_%d", exit)
				if !hasQuestFailureAttribution {
					failureCounts["command_failed"]++
				}
			}
			if dur > 0 {
				slow = append(slow, PersonalSlowOperation{
					QuestID: q.ID, ShortID: q.ShortID, Kind: "command",
					Name: name, DurationMs: dur, Status: status, Timestamp: ev.Timestamp,
				})
			}
		}

		if run.FailureReason == "" {
			run.FailureReason = questFailureReason(q)
			if run.FailureReason != "" {
				failureCounts[run.FailureReason]++
			}
		}
		if reports, repErr := root.LoadReports(q.ID); repErr == nil && reports != nil {
			out.LoopHealth.MakerReportCount += len(reports.MakerReports)
			out.LoopHealth.ReviewReportCount += len(reports.ReviewReports)
			for _, report := range reports.ReviewReports {
				if len(report.CheckedAgainst) > 0 || len(report.EvidenceRefs) > 0 {
					out.LoopHealth.EvidenceBackedReviewCount++
				}
			}
		}
		recent = append(recent, run)
	}

	if out.TerminalQuests > 0 {
		out.SuccessRate = float64(out.SuccessQuests) / float64(out.TerminalQuests)
	}
	out.TotalTokens = out.TotalTokensIn + out.TotalTokensOut
	if out.TotalQuests > 0 {
		out.AvgTokensPerRun = float64(out.TotalTokens) / float64(out.TotalQuests)
	}
	if out.LoopHealth.StartedSessions > 0 {
		out.LoopHealth.FirstOutputCoverage = safeRatio(out.LoopHealth.FirstOutputSessions, out.LoopHealth.StartedSessions)
		out.LoopHealth.TierCUsage = safeRatio(out.LoopHealth.TierCSessionCount, out.LoopHealth.StartedSessions)
	}
	if out.LoopHealth.MakerPhaseDoneCount > 0 {
		out.LoopHealth.MakerReportCoverage = safeRatio(out.LoopHealth.MakerReportCount, out.LoopHealth.MakerPhaseDoneCount)
	}
	if out.LoopHealth.ReviewPhaseDoneCount > 0 {
		out.LoopHealth.ReviewReportCoverage = safeRatio(out.LoopHealth.ReviewReportCount, out.LoopHealth.ReviewPhaseDoneCount)
	}
	if out.LoopHealth.ReviewReportCount > 0 {
		out.LoopHealth.EvidenceBackedReviewCoverage = safeRatio(out.LoopHealth.EvidenceBackedReviewCount, out.LoopHealth.ReviewReportCount)
	}
	automationDiscoveryTotal := out.LoopHealth.AutomationNoFindingCount + out.LoopHealth.AutomationCandidateCount
	if automationDiscoveryTotal > 0 {
		out.LoopHealth.AutomationNoFindingRate = safeRatio(out.LoopHealth.AutomationNoFindingCount, automationDiscoveryTotal)
	}
	out.P50DurationMs = percentile(terminalDurations, 50)
	out.P95DurationMs = percentile(terminalDurations, 95)
	topN := limit
	if topN <= 0 {
		topN = DefaultPersonalStatsLimit
	}
	out.ProjectStats = topProjectStats(projectCounts, topN)
	out.FailureReasons = topCounts(failureCounts, topN, "failure_reason")
	out.FailureCategories = topCounts(failureCategoryCounts, topN, "failure_category")
	out.AnomalySignals = buildAnomalySignals(out)
	out.PhaseStats = finalizePhaseStats(phaseAccums)
	sort.Slice(slow, func(i, j int) bool { return slow[i].DurationMs > slow[j].DurationMs })
	if len(slow) > limit {
		slow = slow[:limit]
	}
	if slow == nil {
		slow = []PersonalSlowOperation{}
	}
	for i := range slow {
		if slow[i].QuestID != "" {
			slow[i].Drilldown = PersonalDrilldown{Query: map[string][]string{"quest_id": {slow[i].QuestID}}}
		}
	}
	out.SlowOperations = slow
	sort.Slice(recent, func(i, j int) bool { return recent[i].CreatedAtMs > recent[j].CreatedAtMs })
	if len(recent) > limit {
		recent = recent[:limit]
	}
	if recent == nil {
		recent = []PersonalRecentRun{}
	}
	out.RecentRuns = recent
	return out, nil
}

func (qs *QuestStore) ListSessionIDs(qid string) ([]string, error) {
	entries, err := os.ReadDir(qs.sessionsDir(qid))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".jsonl" {
			continue
		}
		out = append(out, strings.TrimSuffix(e.Name(), ".jsonl"))
	}
	sort.Strings(out)
	return out, nil
}

func questDurationMs(q *QuestMeta, now int64) int64 {
	if q.StartedAtMs <= 0 {
		return 0
	}
	if q.CompletedAtMs > q.StartedAtMs {
		return q.CompletedAtMs - q.StartedAtMs
	}
	if !q.IsTerminal() && now > q.StartedAtMs {
		return now - q.StartedAtMs
	}
	return 0
}

func questProject(q *QuestMeta) string {
	if q.BaseWorkingDir != "" {
		return filepath.Base(q.BaseWorkingDir)
	}
	if q.WorkspacePath != "" {
		return filepath.Base(q.WorkspacePath)
	}
	return ""
}

func questFailureReason(q *QuestMeta) string {
	if q.Status == model.QuestStatusBlocked && q.BlockedReason != "" {
		return normalizeReason(q.BlockedReason)
	}
	if q.Status == model.QuestStatusCancelled {
		return "cancelled"
	}
	if q.Status == model.QuestStatusFailed {
		if q.FinalVerdict != "" {
			return string(q.FinalVerdict)
		}
		return normalizeReason(q.FinalComment)
	}
	return ""
}

func normalizeReason(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return ""
	}
	switch {
	case strings.Contains(s, "context deadline exceeded"), strings.Contains(s, "timeout"), strings.Contains(s, "超时"):
		return "timeout"
	case strings.Contains(s, "duration_exceeded"), strings.Contains(s, "时长超限"):
		return "duration_exceeded"
	case strings.Contains(s, "no_progress"), strings.Contains(s, "无进展"):
		return "no_progress"
	case strings.Contains(s, "permission"), strings.Contains(s, "权限"):
		return "permission"
	case strings.Contains(s, "command"):
		return "command_failed"
	case strings.Contains(s, "agent 连续错误"):
		return "agent_consecutive_errors"
	}
	if runeCount := len([]rune(s)); runeCount > 80 {
		s = string([]rune(s)[:80])
	}
	return s
}

func phaseLabel(idx int, role string) string {
	if role == "warrior" || idx == 0 {
		return "warrior"
	}
	if role == "mage" || idx == 1 {
		return "mage"
	}
	return fmt.Sprintf("phase_%d", idx)
}

func getPhaseAccum(items map[int]*personalPhaseAccum, idx int, label string) *personalPhaseAccum {
	acc := items[idx]
	if acc == nil {
		acc = &personalPhaseAccum{stat: PersonalPhaseStat{Phase: label}}
		items[idx] = acc
	}
	if acc.stat.Phase == "" {
		acc.stat.Phase = label
	}
	return acc
}

func finalizePhaseStats(items map[int]*personalPhaseAccum) []PersonalPhaseStat {
	keys := make([]int, 0, len(items))
	for k := range items {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	out := make([]PersonalPhaseStat, 0, len(keys))
	for _, k := range keys {
		acc := items[k]
		st := acc.stat
		if st.Count > 0 {
			st.AvgMs = float64(st.TotalMs) / float64(st.Count)
			st.P95Ms = percentile(acc.durations, 95)
		}
		out = append(out, st)
	}
	return out
}

func topProjectStats(items map[string]int, limit int) []PersonalProjectStat {
	out := make([]PersonalProjectStat, 0, len(items))
	for k, v := range items {
		out = append(out, PersonalProjectStat{
			Project:   k,
			Count:     v,
			Drilldown: PersonalDrilldown{Query: map[string][]string{"project": {k}}},
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count == out[j].Count {
			return out[i].Project < out[j].Project
		}
		return out[i].Count > out[j].Count
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

func topCounts(items map[string]int, limit int, filterKey string) []PersonalCount {
	out := make([]PersonalCount, 0, len(items))
	for k, v := range items {
		if k == "" || v <= 0 {
			continue
		}
		out = append(out, PersonalCount{
			Name:      k,
			Count:     v,
			Drilldown: PersonalDrilldown{Query: map[string][]string{filterKey: {k}}},
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count == out[j].Count {
			return out[i].Name < out[j].Name
		}
		return out[i].Count > out[j].Count
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

func buildAnomalySignals(st *PersonalStats) []PersonalAnomalySignal {
	if st == nil {
		return []PersonalAnomalySignal{}
	}
	items := []PersonalAnomalySignal{
		{
			Key:       "blocked",
			Label:     "阻塞委托",
			Severity:  severityFromCount(st.BlockedQuests, "bad"),
			Count:     st.BlockedQuests,
			Drilldown: PersonalDrilldown{Query: map[string][]string{"status": {"blocked"}}},
		},
		{
			Key:       "failed",
			Label:     "失败委托",
			Severity:  severityFromCount(st.FailedQuests, "bad"),
			Count:     st.FailedQuests,
			Drilldown: PersonalDrilldown{Query: map[string][]string{"status": {"failed"}}},
		},
		{
			Key:       "user_review",
			Label:     "待用户审核",
			Severity:  severityFromCount(st.StatusCounts["user_review"], "warn"),
			Count:     st.StatusCounts["user_review"],
			Drilldown: PersonalDrilldown{Query: map[string][]string{"status": {"user_review"}}},
		},
	}
	for _, item := range items {
		if item.Count > 0 {
			return items
		}
	}
	return items
}

func severityFromCount(count int, nonzero string) string {
	if count > 0 {
		return nonzero
	}
	return "ok"
}

func safeRatio(numerator int, denominator int) float64 {
	if denominator <= 0 {
		return 0
	}
	return clampRatio(float64(numerator) / float64(denominator))
}

func clampRatio(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func percentile(values []int64, p int) int64 {
	if len(values) == 0 {
		return 0
	}
	cp := append([]int64(nil), values...)
	sort.Slice(cp, func(i, j int) bool { return cp[i] < cp[j] })
	if len(cp) == 1 {
		return cp[0]
	}
	rank := (float64(p) / 100) * float64(len(cp)-1)
	idx := int(math.Ceil(rank))
	if idx < 0 {
		idx = 0
	}
	if idx >= len(cp) {
		idx = len(cp) - 1
	}
	return cp[idx]
}

func int64FromMeta(meta map[string]any, key string) int64 {
	if meta == nil {
		return 0
	}
	return int64FromAny(meta[key])
}

func stringValueFromMeta(meta map[string]any, key string) string {
	if meta == nil {
		return ""
	}
	return stringFromAny(meta[key])
}

func int64FromAny(v any) int64 {
	switch x := v.(type) {
	case int:
		return int64(x)
	case int64:
		return x
	case int32:
		return int64(x)
	case float64:
		return int64(x)
	case float32:
		return int64(x)
	case jsonNumber:
		n, _ := x.Int64()
		return n
	default:
		return 0
	}
}

type jsonNumber interface {
	Int64() (int64, error)
}

func stringFromAny(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case fmt.Stringer:
		return x.String()
	default:
		return ""
	}
}
