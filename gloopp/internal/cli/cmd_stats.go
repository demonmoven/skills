package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
)

func runStatsCmd(_ *slog.Logger, args []string) int {
	fs := flag.NewFlagSet("stats", flag.ContinueOnError)
	dataDir := fs.String("data-dir", "", "数据目录")
	rangeName := fs.String("range", "all", "统计范围：all | today | week | 7d | 30d")
	limit := fs.Int("limit", fsstore.DefaultPersonalStatsLimit, "明细条数")
	jsonOut := fs.Bool("json", false, "JSON 输出")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	root, err := fsstore.Open(*dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "打开数据目录失败: %v\n", err)
		return 1
	}
	since := statsSinceMs(*rangeName)
	stats, err := fsstore.AggregatePersonalStats(root, since, *limit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "统计失败: %v\n", err)
		return 1
	}
	if *jsonOut {
		if encErr := json.NewEncoder(os.Stdout).Encode(stats); encErr != nil {
			fmt.Fprintf(os.Stderr, "[warn] encode JSON 输出失败: %v\n", encErr)
		}
		return 0
	}
	printPersonalStats(stats, *rangeName)
	return 0
}

func statsSinceMs(rangeName string) int64 {
	now := fsstore.NowMs()
	switch rangeName {
	case "today":
		t := fsstore.FromMs(now)
		y, m, d := t.Date()
		return time.Date(y, m, d, 0, 0, 0, 0, t.Location()).UnixMilli()
	case "week", "7d":
		return now - int64(7*24*time.Hour/time.Millisecond)
	case "30d", "month":
		return now - int64(30*24*time.Hour/time.Millisecond)
	default:
		return 0
	}
}

func printPersonalStats(st *fsstore.PersonalStats, rangeName string) {
	fmt.Printf("Gloop stats (%s)\n", rangeName)
	fmt.Printf("  runs: %d  active: %d  terminal: %d  success_rate: %.1f%%\n",
		st.TotalQuests, st.ActiveQuests, st.TerminalQuests, st.SuccessRate*100)
	fmt.Printf("  duration: p50=%s  p95=%s  total=%s\n",
		formatMs(st.P50DurationMs), formatMs(st.P95DurationMs), formatMs(st.TotalDurationMs))
	fmt.Printf("  turns: %d  tokens: %d in / %d out / %d total\n",
		st.TotalTurns, st.TotalTokensIn, st.TotalTokensOut, st.TotalTokens)

	if len(st.ProjectStats) > 0 {
		fmt.Println("\nProjects")
		for _, item := range st.ProjectStats {
			fmt.Printf("  %-24s %d\n", item.Project, item.Count)
		}
	}
	if len(st.PhaseStats) > 0 {
		fmt.Println("\nPhases")
		for _, ph := range st.PhaseStats {
			fmt.Printf("  %-10s turns=%d p95=%s tokens=%d errors=%d\n",
				ph.Phase, ph.Turns, formatMs(ph.P95Ms), ph.TokensIn+ph.TokensOut, ph.ErrorCount+ph.Timeouts+ph.NoProgress)
		}
	}
	if len(st.FailureReasons) > 0 {
		fmt.Println("\nFailures")
		for _, item := range st.FailureReasons {
			fmt.Printf("  %-28s %d\n", item.Name, item.Count)
		}
	}
	if len(st.SlowOperations) > 0 {
		fmt.Println("\nSlow operations")
		for _, item := range st.SlowOperations {
			fmt.Printf("  %-14s %-18s %-12s %s\n", item.Kind, item.Name, item.QuestID, formatMs(item.DurationMs))
		}
	}
	if len(st.RecentRuns) > 0 {
		fmt.Println("\nRecent runs")
		for _, run := range st.RecentRuns {
			fmt.Printf("  %-18s %-11s %-8s %s\n",
				statsRunID(run), run.Status, formatMs(run.DurationMs), truncateStr(run.Query, 56))
		}
	}
}

func formatMs(ms int64) string {
	if ms <= 0 {
		return "-"
	}
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	d := time.Duration(ms) * time.Millisecond
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	return fmt.Sprintf("%.1fh", d.Hours())
}

func statsRunID(r fsstore.PersonalRecentRun) string {
	if r.ShortID != "" {
		return r.ShortID
	}
	return r.QuestID
}
