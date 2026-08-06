package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"code.byted.org/lihuanyu.0w0/gloop/internal/events"
	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
	"code.byted.org/lihuanyu.0w0/gloop/internal/orchestrator"
)

type stringSliceFlag []string

func (s *stringSliceFlag) String() string     { return strings.Join(*s, ",") }
func (s *stringSliceFlag) Set(v string) error { *s = append(*s, v); return nil }

func runQuestCmd(ctx context.Context, log *slog.Logger, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "用法: gloop quest <list|show|run|review|diff|apply|discard|cancel>")
		return 2
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "cleanup":
		return runQuestCleanup(log, rest)
	case "ledger-audit":
		return runQuestLedgerAudit(log, rest)
	case "resolve-blocked", "unblock", "resume":
		return runQuestResolveBlocked(ctx, log, rest)
	case "info":
		return runQuestInfo(ctx, log, rest)
	case "history":
		return runQuestHistory(log, rest)
	case "progress":
		return runQuestProgress(log, rest)
	case "list":
		return runQuestList(ctx, log, rest)
	case "show":
		return runQuestShow(ctx, log, rest)
	case "run":
		return runQuestRun(ctx, log, rest)
	case "review":
		return runQuestReview(ctx, log, rest)
	case "comment":
		return runQuestComment(ctx, log, rest)
	case "answer":
		return runQuestAnswer(ctx, log, rest)
	case "backup", "backups":
		return runQuestBackup(ctx, log, rest)
	case "diff":
		return runQuestDiff(ctx, log, rest)
	case "apply":
		return runQuestApply(ctx, log, rest)
	case "discard":
		return runQuestDiscard(ctx, log, rest)
	case "cancel", "stop":
		return runQuestCancel(ctx, log, rest)
	case "spawn-execute":
		return runQuestSpawnExecute(ctx, log, rest)
	case "spawn":
		return runQuestSpawn(ctx, log, rest)
	default:
		fmt.Fprintf(os.Stderr, "未知 quest 子命令 %q\n", sub)
		return 2
	}
}

func runQuestCleanup(log *slog.Logger, args []string) int {
	fs := flag.NewFlagSet("quest cleanup", flag.ContinueOnError)
	dataDir := fs.String("data-dir", "", "数据目录")
	retentionDays := fs.Int("retention-days", -1, "工作区保留天数（默认取 config）")
	includeFailed := fs.Bool("include-failed", false, "同时清理 failed/cancelled quest 工作区")
	dryRun := fs.Bool("dry-run", true, "只列出候选项，不删除")
	jsonOut := fs.Bool("json", false, "JSON 输出")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	r := newRunner(*dataDir, log)
	if _, err := r.bootstrap(); err != nil {
		log.Error(logPrefix+" bootstrap 失败", "err", err)
		return 1
	}
	days := *retentionDays
	if days < 0 {
		days = r.cfg.WorkspaceRetentionDays
	}
	include := *includeFailed || r.cfg.AutoCleanupFailedQuests
	items, err := fsstore.NewQuestStore(r.root).CleanupExpiredWorkspaces(days, include, fsstore.NowMs(), *dryRun)
	if err != nil {
		log.Error(logPrefix+" cleanup 失败", "err", err)
		return 1
	}
	if *jsonOut {
		if encErr := json.NewEncoder(os.Stdout).Encode(items); encErr != nil {
			fmt.Fprintf(os.Stderr, "[warn] encode JSON 输出失败: %v\n", encErr)
		}
		return 0
	}
	action := "候选"
	if !*dryRun {
		action = "已清理"
	}
	fmt.Printf("workspace cleanup %s: %d 个\n", action, len(items))
	for _, item := range items {
		fmt.Printf("  %s %-10s %s\n", item.QuestID, item.Status, item.WorkspacePath)
	}
	return 0
}

func runQuestLedgerAudit(log *slog.Logger, args []string) int {
	fs := flag.NewFlagSet("quest ledger-audit", flag.ContinueOnError)
	dataDir := fs.String("data-dir", "", "数据目录")
	repair := fs.Bool("repair", false, "修复可安全补写的 thread ledger 缺口")
	jsonOut := fs.Bool("json", false, "JSON 输出")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "需要 quest id: gloop quest ledger-audit <qid> [--repair]")
		return 2
	}
	qid := fs.Arg(0)
	r := newRunner(*dataDir, log)
	if _, err := r.bootstrap(); err != nil {
		log.Error(logPrefix+" bootstrap 失败", "err", err)
		return 1
	}
	audit, err := fsstore.NewQuestStore(r.root).AuditThreadLedger(qid, *repair)
	if err != nil {
		log.Error(logPrefix+" ledger-audit 失败", "qid", qid, "err", err)
		return 1
	}
	if *jsonOut {
		return writeJSONLine(audit)
	}
	status := "PASS"
	if !audit.OK {
		status = "FAIL"
	}
	fmt.Printf("thread ledger audit %s: qid=%s posts=%d findings=%d repairs=%d\n", status, audit.QuestID, audit.PostCount, len(audit.Findings), len(audit.Repaired))
	for _, finding := range audit.Findings {
		fmt.Printf("  [%s] %s %s %s\n", finding.Severity, finding.Code, finding.PostID, finding.Message)
	}
	for _, repaired := range audit.Repaired {
		fmt.Printf("  [repaired] %s %s %s\n", repaired.Code, repaired.PostID, repaired.Detail)
	}
	if audit.OK {
		return 0
	}
	return 1
}

func runQuestList(_ context.Context, log *slog.Logger, args []string) int {
	fs := flag.NewFlagSet("quest list", flag.ContinueOnError)
	dataDir := fs.String("data-dir", "", "数据目录")
	limit := fs.Int("limit", 50, "最多显示条数")
	jsonOut := fs.Bool("json", false, "以 JSON 输出")
	officialOnly := fs.Bool("official", false, "只显示官方委托")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	r := newRunner(*dataDir, log)
	if _, err := r.bootstrap(); err != nil {
		log.Error(logPrefix+" bootstrap 失败", "err", err)
		return 1
	}
	qs, err := r.engine.ListQuests()
	if err != nil {
		log.Error(logPrefix+" ListQuests 失败", "err", err)
		return 1
	}
	if *limit > 0 && len(qs) > *limit {
		qs = qs[:*limit]
	}
	if *officialOnly {
		filtered := []*fsstore.QuestMeta{}
		for _, q := range qs {
			if r.root.IsQuestOfficial(q) {
				filtered = append(filtered, q)
			}
		}
		qs = filtered
	}
	if *jsonOut {
		if encErr := json.NewEncoder(os.Stdout).Encode(qs); encErr != nil {
			fmt.Fprintf(os.Stderr, "[warn] encode JSON 输出失败: %v\n", encErr)
		}
		return 0
	}
	if len(qs) == 0 {
		fmt.Println("（暂无委托，用 `gloop quest run --query ...` 创建）")
		return 0
	}
	fmt.Printf("%-22s  %-14s  %-20s  %s\n", "ID", "STATUS", "TYPE", "QUERY")
	for _, q := range qs {
		query := q.Query
		if len(query) > 40 {
			query = query[:40] + "…"
		}
		fmt.Printf("%-22s  %-14s  %-20s  %s\n", q.ID, q.Status, q.Type, query)
	}
	return 0
}

func runQuestShow(_ context.Context, log *slog.Logger, args []string) int {
	fs := flag.NewFlagSet("quest show", flag.ContinueOnError)
	dataDir := fs.String("data-dir", "", "数据目录")
	showEvents := fs.Int("events", 0, "显示最后 N 条事件")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "需要 quest id:  gloop quest show <qid>")
		return 2
	}
	qid := fs.Arg(0)
	r := newRunner(*dataDir, log)
	if _, err := r.bootstrap(); err != nil {
		log.Error(logPrefix+" bootstrap 失败", "err", err)
		return 1
	}
	q, err := r.engine.GetQuest(qid)
	if err != nil {
		log.Error(logPrefix+" 加载委托失败", "qid", qid, "err", err)
		return 1
	}
	b, marshalErr := json.MarshalIndent(q, "", "  ")
	if marshalErr != nil {
		fmt.Fprintf(os.Stderr, "[warn] marshal quest: %v\n", marshalErr)
	}
	fmt.Println(string(b))
	if *showEvents > 0 {
		qs := fsstore.NewQuestStore(r.root)
		evs, err := qs.ReadEvents(qid, *showEvents)
		if err == nil && len(evs) > 0 {
			fmt.Printf("\n--- EVENTS (last %d) ---\n", len(evs))
			for _, e := range evs {
				p, mErr := json.Marshal(e.Payload)
				if mErr != nil {
					fmt.Fprintf(os.Stderr, "[warn] marshal event %s payload: %v\n", e.Type, mErr)
				}
				fmt.Printf("  [%s] %s %s\n", fsstore.FromMs(e.Timestamp).Format("15:04:05"), e.Type, p)
			}
		}
	}
	return 0
}

func runQuestRun(ctx context.Context, log *slog.Logger, args []string) int {
	fs := flag.NewFlagSet("quest run", flag.ContinueOnError)
	dataDir := fs.String("data-dir", "", "数据目录")
	query := fs.String("query", "", "（必填）委托内容")
	questType := fs.String("type", "execute", "委托类型：execute | design")
	intensity := fs.String("intensity", "standard", "强度：quick | standard | deep | adversarial（用户侧可理解为快速/标准/深入/严审）")
	workDir := fs.String("work-dir", "", "工作目录（默认取自 config）")
	workspaceMode := fs.String("workspace-mode", "", "工作区模式：auto | worktree | copy | readonly")
	warriorID := fs.String("warrior", "", "指定剑士 adventurer id")
	mageID := fs.String("mage", "", "指定法师 adventurer id")
	executeAgentID := fs.String("execute-agent", "", "指定执行阶段 agent id（agent-direct 模式，跳过 adventurer）")
	reviewAgentID := fs.String("review-agent", "", "指定评审阶段 agent id（agent-direct 模式，跳过 adventurer）")
	watch := fs.Bool("watch", true, "是否挂住等结果（默认 true）")
	withDesignPhase := fs.Bool("with-design-phase", false, "启用可选设计阶段（design → design review → execute → review）")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *query == "" {
		// 支持从位置参数取 query
		if fs.NArg() > 0 {
			*query = strings.Join(fs.Args(), " ")
		}
	}
	if *query == "" {
		fmt.Fprintln(os.Stderr, "缺失 --query")
		return 2
	}
	r := newRunner(*dataDir, log)
	tok, err := r.bootstrap()
	if err != nil {
		log.Error(logPrefix+" bootstrap 失败", "err", err)
		return 1
	}
	qt := model.QuestType(*questType)
	if qt == "" {
		qt = model.QuestTypeExecute
	}
	wm, wmErr := parseCLIWorkspaceMode(*workspaceMode)
	if wmErr != nil {
		fmt.Fprintln(os.Stderr, wmErr.Error())
		return 2
	}
	qi, qiErr := parseCLIQuestIntensity(*intensity)
	if qiErr != nil {
		fmt.Fprintln(os.Stderr, qiErr.Error())
		return 2
	}
	q, err := r.engine.CreateQuest(ctx, *query, qt, *workDir, orchestrator.CreateQuestOptions{
		WarriorID:       *warriorID,
		MageID:          *mageID,
		ExecuteAgentID:  *executeAgentID,
		ReviewAgentID:   *reviewAgentID,
		WorkspaceMode:   wm,
		Intensity:       qi,
		WithDesignPhase: *withDesignPhase,
	})
	if err != nil {
		log.Error(logPrefix+" 创建委托失败", "err", err)
		return 1
	}
	fmt.Printf("📜 委托已创建: qid=%s  short_id=%s\n", q.ID, q.ShortID)
	fmt.Printf("   类型: %s\n", q.Type)
	fmt.Printf("   强度: %s\n", q.Intensity)
	fmt.Printf("   内容: %s\n", truncateStr(q.Query, 80))
	fmt.Printf("   Dashboard: %s\n", tok.DashboardURL())

	if !*watch {
		fmt.Printf("   状态: pending（未启动；使用 `gloop quest run` 默认 watch，或在 Dashboard/API 中启动）\n")
		return 0
	}
	if err := r.engine.StartQuest(ctx, q.ID); err != nil {
		log.Error(logPrefix+" 启动委托失败", "qid", q.ID, "err", err)
		return 1
	}

	// 用 bus 订阅事件，到 user_review / success / failed / cancelled 就退出
	return sseWatchLocal(ctx, r, q.ID)
}

// sseWatchLocal 不走 HTTP（没必要启 server），直接读 bus 订阅。
// 这样不要求 server 端也开着，纯 CLI 就能 watch。
func sseWatchLocal(ctx context.Context, r *Runner, qid string) int {
	ch, unsub := r.bus.Subscribe()
	defer unsub()

	done := make(chan int, 1)
	go func() {
		for ev := range ch {
			if ev.QuestID != "" && ev.QuestID != qid {
				continue
			}
			// 简单彩色摘要输出
			ts := fsstore.FromMs(ev.Timestamp).Format("15:04:05")
			payloadStr := string(ev.Payload)
			if len(payloadStr) > 160 {
				payloadStr = payloadStr[:160] + "…"
			}
			fmt.Printf("  [%s] %s  %s\n", ts, ev.Type, payloadStr)
			switch ev.Type {
			case events.EvtUserReview, events.EvtQuestSuccess, events.EvtQuestBlocked,
				events.EvtQuestFailed, events.EvtQuestCancelled:
				fmt.Printf("✅ 委托达到终态/评审态: %s\n", ev.Type)
				// 刷新一次 meta
				q, err := r.engine.GetQuest(qid)
				if err == nil {
					fmt.Printf("   状态: %s\n", q.Status)
					if q.FinalComment != "" {
						fmt.Printf("   评审: %s\n", q.FinalComment)
					}
				}
				done <- 0
				return
			}
		}
	}()

	// 轮询兜底（防止 bus 端消息丢失）
	go func() {
		t := time.NewTicker(5 * time.Second)
		defer t.Stop()
		for range t.C {
			q, err := r.engine.GetQuest(qid)
			if err != nil {
				continue
			}
			switch q.Status {
			case model.QuestStatusUserReview, model.QuestStatusSuccess, model.QuestStatusBlocked,
				model.QuestStatusFailed, model.QuestStatusCancelled:
				// 给 SSE 协程打印的机会
				time.Sleep(500 * time.Millisecond)
				select {
				case done <- 0:
				default:
				}
				return
			}
		}
	}()

	select {
	case <-ctx.Done():
		fmt.Println("\n（用户中断）")
		return 130
	case code := <-done:
		return code
	}
}

func runQuestReview(ctx context.Context, log *slog.Logger, args []string) int {
	fs := flag.NewFlagSet("quest review", flag.ContinueOnError)
	dataDir := fs.String("data-dir", "", "数据目录")
	comment := fs.String("comment", "", "评审意见")
	apply := fs.Bool("apply", false, "通过后立即应用工作区改动（仅 pass 生效）")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() < 2 {
		fmt.Fprintln(os.Stderr, "用法: gloop quest review <qid> <pass|changes|reject> [--apply] [--comment ...]")
		return 2
	}
	qid := fs.Arg(0)
	verdictArg := fs.Arg(1)
	var verdict model.QuestVerdict
	switch verdictArg {
	case "pass":
		verdict = model.VerdictPass
	case "changes", "request_changes", "rework":
		verdict = model.VerdictRequestChange
	case "reject":
		verdict = model.VerdictReject
	default:
		fmt.Fprintf(os.Stderr, "未知 verdict %q，支持: pass / changes / reject\n", verdictArg)
		return 2
	}
	if *apply && verdict != model.VerdictPass {
		fmt.Fprintln(os.Stderr, "--apply 只能和 pass 一起使用")
		return 2
	}
	dd := resolveDataDirOrDefault(*dataDir)
	if proxy := tryServerProxy(dd); proxy != nil {
		if err := proxy.resolveUserReview(qid, string(verdict), *comment, *apply); err != nil {
			log.Error(logPrefix+" 评审失败 (via server)", "err", err)
			return 1
		}
	} else {
		r := newRunner(*dataDir, log)
		if _, err := r.bootstrap(); err != nil {
			log.Error(logPrefix+" bootstrap 失败", "err", err)
			return 1
		}
		if err := r.engine.ResolveUserReview(ctx, qid, verdict, *comment); err != nil {
			log.Error(logPrefix+" 评审失败", "err", err)
			return 1
		}
		if *apply {
			warnings, err := r.engine.ApplyQuest(ctx, qid, false)
			if err != nil {
				log.Error(logPrefix+" apply 失败", "qid", qid, "warnings", len(warnings), "err", err)
				return 1
			}
			fmt.Printf("✅ 已应用改动: qid=%s warnings=%d\n", qid, len(warnings))
		}
	}
	fmt.Printf("✅ 评审已提交: qid=%s verdict=%s\n", qid, verdict)
	if *comment != "" {
		fmt.Printf("   comment: %s\n", *comment)
	}
	return 0
}

func runQuestComment(_ context.Context, log *slog.Logger, args []string) int {
	fs := flag.NewFlagSet("quest comment", flag.ContinueOnError)
	dataDir := fs.String("data-dir", "", "数据目录")
	comment := fs.String("comment", "", "评论内容")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "用法: gloop quest comment <qid> --comment <text>")
		return 2
	}
	if *comment == "" && fs.NArg() > 1 {
		*comment = strings.Join(fs.Args()[1:], " ")
	}
	qid := fs.Arg(0)
	dd := resolveDataDirOrDefault(*dataDir)
	if proxy := tryServerProxy(dd); proxy != nil {
		if err := proxy.addComment(qid, *comment); err != nil {
			log.Error(logPrefix+" 评论失败 (via server)", "qid", qid, "err", err)
			return 1
		}
	} else {
		r := newRunner(*dataDir, log)
		if _, err := r.bootstrap(); err != nil {
			log.Error(logPrefix+" bootstrap 失败", "err", err)
			return 1
		}
		if err := r.engine.AppendUserComment(qid, *comment, ""); err != nil {
			log.Error(logPrefix+" 评论失败", "qid", qid, "err", err)
			return 1
		}
	}
	fmt.Printf("✅ 评论已追加: qid=%s\n", qid)
	return 0
}

func runQuestAnswer(_ context.Context, log *slog.Logger, args []string) int {
	fs := flag.NewFlagSet("quest answer", flag.ContinueOnError)
	dataDir := fs.String("data-dir", "", "数据目录")
	questionID := fs.String("question-id", "", "问题 ID")
	answer := fs.String("answer", "", "回答内容")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "用法: gloop quest answer <qid> [--question-id xxx] --answer <text>")
		return 2
	}
	if *answer == "" && fs.NArg() > 1 {
		*answer = strings.Join(fs.Args()[1:], " ")
	}
	qid := fs.Arg(0)
	dd := resolveDataDirOrDefault(*dataDir)
	if proxy := tryServerProxy(dd); proxy != nil {
		if err := proxy.addAnswer(qid, *questionID, *answer); err != nil {
			log.Error(logPrefix+" 回答失败 (via server)", "qid", qid, "err", err)
			return 1
		}
	} else {
		r := newRunner(*dataDir, log)
		if _, err := r.bootstrap(); err != nil {
			log.Error(logPrefix+" bootstrap 失败", "err", err)
			return 1
		}
		if _, err := r.engine.AppendUserAnswer(qid, *questionID, *answer, "cli"); err != nil {
			log.Error(logPrefix+" 回答失败", "qid", qid, "err", err)
			return 1
		}
	}
	fmt.Printf("✅ 回答已追加: qid=%s\n", qid)
	return 0
}

func runQuestBackup(_ context.Context, log *slog.Logger, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "用法: gloop quest backup <list|show>")
		return 2
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "list":
		return runQuestBackupList(log, rest)
	case "show":
		return runQuestBackupShow(log, rest)
	default:
		fmt.Fprintf(os.Stderr, "未知 backup 子命令 %q\n", sub)
		return 2
	}
}

func runQuestBackupList(log *slog.Logger, args []string) int {
	fs := flag.NewFlagSet("quest backup list", flag.ContinueOnError)
	dataDir := fs.String("data-dir", "", "数据目录")
	jsonOut := fs.Bool("json", false, "JSON 输出")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "需要 quest id:  gloop quest backup list <qid>")
		return 2
	}
	r := newRunner(*dataDir, log)
	if _, err := r.bootstrap(); err != nil {
		log.Error(logPrefix+" bootstrap 失败", "err", err)
		return 1
	}
	items, err := r.engine.ListApplyBackups(fs.Arg(0))
	if err != nil {
		log.Error(logPrefix+" backup list 失败", "qid", fs.Arg(0), "err", err)
		return 1
	}
	if *jsonOut {
		if encErr := json.NewEncoder(os.Stdout).Encode(items); encErr != nil {
			fmt.Fprintf(os.Stderr, "[warn] encode JSON 输出失败: %v\n", encErr)
		}
		return 0
	}
	if len(items) == 0 {
		fmt.Println("（暂无 apply backup）")
		return 0
	}
	fmt.Printf("%-24s  %-20s  %s\n", "BACKUP", "BASE", "PATCH")
	for _, item := range items {
		fmt.Printf("%-24s  %-20s  %s\n", item.ID, truncateStr(item.BaseCommit, 12), item.PatchPath)
	}
	return 0
}

func runQuestBackupShow(log *slog.Logger, args []string) int {
	fs := flag.NewFlagSet("quest backup show", flag.ContinueOnError)
	dataDir := fs.String("data-dir", "", "数据目录")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() < 2 {
		fmt.Fprintln(os.Stderr, "用法: gloop quest backup show <qid> <backup_id>")
		return 2
	}
	r := newRunner(*dataDir, log)
	if _, err := r.bootstrap(); err != nil {
		log.Error(logPrefix+" bootstrap 失败", "err", err)
		return 1
	}
	item, err := r.engine.GetApplyBackup(fs.Arg(0), fs.Arg(1))
	if err != nil {
		log.Error(logPrefix+" backup show 失败", "qid", fs.Arg(0), "backup", fs.Arg(1), "err", err)
		return 1
	}
	b, marshalErr := json.MarshalIndent(item, "", "  ")
	if marshalErr != nil {
		fmt.Fprintf(os.Stderr, "[warn] marshal backup: %v\n", marshalErr)
	}
	fmt.Println(string(b))
	return 0
}

// ==================== diff / apply / discard ====================

func runQuestDiff(_ context.Context, log *slog.Logger, args []string) int {
	fs := flag.NewFlagSet("quest diff", flag.ContinueOnError)
	dataDir := fs.String("data-dir", "", "数据目录")
	full := fs.Bool("full", false, "显示完整 diff（默认只显示统计）")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "需要 quest id:  gloop quest diff <qid>")
		return 2
	}
	qid := fs.Arg(0)
	r := newRunner(*dataDir, log)
	if _, err := r.bootstrap(); err != nil {
		log.Error(logPrefix+" bootstrap 失败", "err", err)
		return 1
	}
	diff, err := r.engine.ComputeDiff(context.Background(), qid)
	if err != nil {
		log.Error(logPrefix+" 计算 diff 失败", "qid", qid, "err", err)
		return 1
	}
	fmt.Printf("📊 Diff 摘要\n")
	fmt.Printf("   %s\n", diff.Stat)
	fmt.Printf("   变更文件: %d  +%d -%d\n", diff.ChangedFiles, diff.Additions, diff.Deletions)
	if len(diff.Files) > 0 {
		fmt.Printf("\n📁 变更文件列表:\n")
		for _, f := range diff.Files {
			statusIcon := "M"
			switch f.Status {
			case "added":
				statusIcon = "+"
			case "deleted":
				statusIcon = "-"
			}
			fmt.Printf("   %s %-40s  +%d -%d\n", statusIcon, f.Path, f.Additions, f.Deletions)
		}
	}
	if *full && diff.Diff != "" {
		fmt.Printf("\n📝 完整 diff:\n")
		fmt.Println(diff.Diff)
	}
	return 0
}

func runQuestApply(_ context.Context, log *slog.Logger, args []string) int {
	fs := flag.NewFlagSet("quest apply", flag.ContinueOnError)
	dataDir := fs.String("data-dir", "", "数据目录")
	force := fs.Bool("force", false, "强制 apply，跳过安全检查")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "需要 quest id:  gloop quest apply <qid>")
		return 2
	}
	qid := fs.Arg(0)
	r := newRunner(*dataDir, log)
	if _, err := r.bootstrap(); err != nil {
		log.Error(logPrefix+" bootstrap 失败", "err", err)
		return 1
	}
	warnings, err := r.engine.ApplyQuest(context.Background(), qid, *force)
	if err != nil {
		log.Error(logPrefix+" apply 失败", "qid", qid, "err", err)
		if len(warnings) > 0 {
			fmt.Fprintf(os.Stderr, "\n⚠️  安全警告:\n")
			for _, w := range warnings {
				fmt.Fprintf(os.Stderr, "   [%s] %s: %s\n", w.Severity, w.Category, w.Message)
			}
		}
		return 1
	}
	fmt.Printf("✅ 已应用改动: qid=%s\n", qid)
	if len(warnings) > 0 {
		fmt.Printf("   安全警告: %d 条\n", len(warnings))
		for _, w := range warnings {
			fmt.Printf("   [%s] %s\n", w.Severity, w.Message)
		}
	}
	fmt.Printf("   改动已合入工作目录\n")
	return 0
}

func runQuestDiscard(_ context.Context, log *slog.Logger, args []string) int {
	fs := flag.NewFlagSet("quest discard", flag.ContinueOnError)
	dataDir := fs.String("data-dir", "", "数据目录")
	reason := fs.String("reason", "", "丢弃原因")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "需要 quest id:  gloop quest discard <qid>")
		return 2
	}
	qid := fs.Arg(0)
	dd := resolveDataDirOrDefault(*dataDir)
	if proxy := tryServerProxy(dd); proxy != nil {
		if err := proxy.discardQuest(qid, *reason); err != nil {
			log.Error(logPrefix+" discard 失败 (via server)", "qid", qid, "err", err)
			return 1
		}
	} else {
		r := newRunner(*dataDir, log)
		if _, err := r.bootstrap(); err != nil {
			log.Error(logPrefix+" bootstrap 失败", "err", err)
			return 1
		}
		if err := r.engine.DiscardQuest(context.Background(), qid, *reason); err != nil {
			log.Error(logPrefix+" discard 失败", "qid", qid, "err", err)
			return 1
		}
	}
	fmt.Printf("🗑️  已丢弃改动: qid=%s\n", qid)
	fmt.Printf("   工作区已清空，不影响原目录\n")
	return 0
}

func runQuestCancel(_ context.Context, log *slog.Logger, args []string) int {
	fs := flag.NewFlagSet("quest cancel", flag.ContinueOnError)
	dataDir := fs.String("data-dir", "", "数据目录")
	reason := fs.String("reason", "", "取消原因")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "需要 quest id:  gloop quest cancel <qid>")
		return 2
	}
	qid := fs.Arg(0)
	dd := resolveDataDirOrDefault(*dataDir)
	if proxy := tryServerProxy(dd); proxy != nil {
		if err := proxy.cancelQuest(qid, *reason); err != nil {
			log.Error(logPrefix+" cancel 失败 (via server)", "qid", qid, "err", err)
			return 1
		}
	} else {
		r := newRunner(*dataDir, log)
		if _, err := r.bootstrap(); err != nil {
			log.Error(logPrefix+" bootstrap 失败", "err", err)
			return 1
		}
		if err := r.engine.StopQuest(qid, *reason); err != nil {
			log.Error(logPrefix+" cancel 失败", "qid", qid, "err", err)
			return 1
		}
	}
	fmt.Printf("🛑 已取消委托: qid=%s\n", qid)
	if *reason != "" {
		fmt.Printf("   原因: %s\n", *reason)
	}
	return 0
}

func runQuestResolveBlocked(ctx context.Context, log *slog.Logger, args []string) int {
	fs := flag.NewFlagSet("quest resolve-blocked", flag.ContinueOnError)
	dataDir := fs.String("data-dir", "", "数据目录")
	action := fs.String("action", "continue", "处理动作：continue | user-review | cancel")
	comment := fs.String("comment", "", "补充说明 / 返工提示")
	addTurns := fs.Int("add-turns", 0, "为该 quest 追加单阶段 turn 预算")
	addDurationMinutes := fs.Int("add-duration-minutes", 0, "为该 quest 追加总时长预算（分钟）")
	watch := fs.Bool("watch", false, "continue 后挂住等结果")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "需要 quest id:  gloop quest resolve-blocked <qid> [--action continue|user-review|cancel]")
		return 2
	}
	qid := fs.Arg(0)
	dd := resolveDataDirOrDefault(*dataDir)
	needLocalEngine := *watch || *addTurns > 0 || *addDurationMinutes > 0
	if proxy := tryServerProxy(dd); proxy != nil && !needLocalEngine {
		if err := proxy.resolveBlocked(qid, *action, *comment); err != nil {
			log.Error(logPrefix+" resolve-blocked 失败 (via server)", "qid", qid, "err", err)
			return 1
		}
	} else {
		r := newRunner(*dataDir, log)
		tok, err := r.bootstrap()
		if err != nil {
			log.Error(logPrefix+" bootstrap 失败", "err", err)
			return 1
		}
		if err := r.engine.ResolveBlockedQuest(ctx, qid, orchestrator.ResumeBlockedOptions{
			Action:             *action,
			Comment:            *comment,
			AddTurns:           *addTurns,
			AddDurationMinutes: *addDurationMinutes,
		}); err != nil {
			log.Error(logPrefix+" resolve-blocked 失败", "qid", qid, "err", err)
			return 1
		}
		if *watch && (*action == "" || *action == "continue") {
			fmt.Printf("   Dashboard: %s\n", tok.DashboardURL())
			return sseWatchLocal(ctx, r, qid)
		}
	}
	fmt.Printf("✅ blocked 委托已处理: qid=%s action=%s\n", qid, *action)
	return 0
}

// runQuestSpawn 扇出一个独立委托（quest_spawn 平台工具的 CLI 版本）。
// 优先走 server HTTP API，server 没跑就 fallback 到本地 engine。
func runQuestSpawn(ctx context.Context, log *slog.Logger, args []string) int {
	fs := flag.NewFlagSet("quest spawn", flag.ContinueOnError)
	query := fs.String("query", "", "新委托的任务描述")
	groupID := fs.String("group-id", "", "分组标签；不填则自动生成")
	workDir := fs.String("work-dir", "", "工作目录；在 quest 内调用时自动继承当前工作区")
	leafID := fs.String("leaf-id", "", "fanout leaf 标识；同一 group 内唯一")
	ownershipScopes := fs.String("ownership-scopes", "", "ownership scope 列表，逗号分隔（文件/模块/问题域）")
	mergeStrategy := fs.String("merge-strategy", "", "合并策略：single_leaf | sequential | no_merge")
	mergeOwnerLeafID := fs.String("merge-owner-leaf-id", "", "合并责任 leaf；single_leaf/sequential 必填")
	parentQuestID := fs.String("parent", "", "父委托 ID；首次 spawn 会将父委托提升为 fanout root")
	dataDir := fs.String("data-dir", "", "数据目录")
	jsonOut := fs.Bool("json", false, "JSON 结构化输出")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *query == "" {
		// 支持位置参数：gloop quest spawn "任务描述"
		if fs.NArg() > 0 {
			*query = strings.Join(fs.Args(), " ")
		}
		if *query == "" {
			fmt.Fprintln(os.Stderr, "query 必填（--query 或位置参数）")
			return 2
		}
	}

	dd := resolveDataDirOrDefault(*dataDir)

	// 工作目录：优先 --work-dir，其次 quest 上下文工作区
	wd := *workDir
	effectiveParent := *parentQuestID
	if effectiveParent == "" {
		if _, _, _, q, code := loadAgentQuestQuiet(); code == 0 && q != nil {
			effectiveParent = q.ID
		}
	}
	if wd == "" {
		if _, _, _, q, code := loadAgentQuestQuiet(); code == 0 && q != nil && q.WorkspacePath != "" {
			wd = q.WorkspacePath
		}
	}
	contract := fsstore.FanoutContract{
		LeafID:           *leafID,
		OwnershipScopes:  parseCSVFlag(*ownershipScopes),
		MergeStrategy:    *mergeStrategy,
		MergeOwnerLeafID: *mergeOwnerLeafID,
	}

	// 优先走 server API
	if proxy := tryServerProxy(dd); proxy != nil {
		qid, gid, err := proxy.spawnQuest(*query, *groupID, wd, effectiveParent, contract)
		if err != nil {
			if *jsonOut {
				return writeJSONError(1, "spawn_failed", err.Error())
			}
			fmt.Fprintf(os.Stderr, "spawn 失败: %v\n", err)
			return 1
		}
		if *jsonOut {
			return writeJSONLine(map[string]any{
				"ok":       true,
				"qid":      qid,
				"group_id": gid,
				"message":  "委托已创建并启动",
			})
		}
		fmt.Printf("委托已创建并启动: qid=%s group_id=%s\n", qid, gid)
		return 0
	}

	// fallback 到本地 engine
	r := newRunner(*dataDir, log)
	if _, err := r.bootstrap(); err != nil {
		log.Error(logPrefix+" bootstrap 失败", "err", err)
		return 1
	}
	qid, err := r.engine.SpawnQuest(ctx, *query, *groupID, wd, effectiveParent, contract)
	if err != nil {
		if *jsonOut {
			return writeJSONError(1, "spawn_failed", err.Error())
		}
		log.Error(logPrefix+" spawn 失败", "err", err)
		return 1
	}
	gid := *groupID
	if q, qErr := r.engine.GetQuest(qid); qErr == nil && q != nil && q.GroupID != "" {
		gid = q.GroupID
	}
	if *jsonOut {
		return writeJSONLine(map[string]any{
			"ok":       true,
			"qid":      qid,
			"group_id": gid,
			"message":  "委托已创建并启动",
		})
	}
	fmt.Printf("委托已创建并启动: qid=%s group_id=%s\n", qid, gid)
	return 0
}

func parseCSVFlag(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if item := strings.TrimSpace(part); item != "" {
			out = append(out, item)
		}
	}
	return out
}

func runQuestSpawnExecute(ctx context.Context, log *slog.Logger, args []string) int {
	fs := flag.NewFlagSet("quest spawn-execute", flag.ContinueOnError)
	dataDir := fs.String("data-dir", "", "数据目录")
	watch := fs.Bool("watch", false, "创建后启动并挂住等结果")
	start := fs.Bool("start", false, "创建后立即启动")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "需要 design quest id:  gloop quest spawn-execute <qid>")
		return 2
	}
	r := newRunner(*dataDir, log)
	tok, err := r.bootstrap()
	if err != nil {
		log.Error(logPrefix+" bootstrap 失败", "err", err)
		return 1
	}
	q, err := r.engine.SpawnExecuteFromDesign(ctx, fs.Arg(0), *start || *watch)
	if err != nil {
		log.Error(logPrefix+" spawn-execute 失败", "err", err)
		return 1
	}
	fmt.Printf("⚔️  执行委托已创建: qid=%s parent=%s\n", q.ID, q.ParentQuestID)
	fmt.Printf("   Dashboard: %s\n", tok.DashboardURL())
	if *watch {
		return sseWatchLocal(ctx, r, q.ID)
	}
	return 0
}

func truncateStr(s string, n int) string {
	if n <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "…"
}

func parseCLIWorkspaceMode(raw string) (model.WorkspaceMode, error) {
	switch raw {
	case "", "auto":
		return "", nil
	case string(model.WorkspaceWorktree):
		return model.WorkspaceWorktree, nil
	case string(model.WorkspaceCopy):
		return model.WorkspaceCopy, nil
	case string(model.WorkspaceReadOnly):
		return model.WorkspaceReadOnly, nil
	default:
		return "", fmt.Errorf("workspace-mode 必须是 auto | worktree | copy | readonly")
	}
}

func parseCLIQuestIntensity(raw string) (model.QuestIntensity, error) {
	switch raw {
	case "", string(model.QuestIntensityStandard):
		return model.QuestIntensityStandard, nil
	case string(model.QuestIntensityQuick):
		return model.QuestIntensityQuick, nil
	case string(model.QuestIntensityDeep):
		return model.QuestIntensityDeep, nil
	case string(model.QuestIntensityAdversarial):
		return model.QuestIntensityAdversarial, nil
	default:
		return "", fmt.Errorf("intensity 必须是 quick | standard | deep | adversarial")
	}
}

// =========================================================================
// adventurer
// =========================================================================
