package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
	"code.byted.org/lihuanyu.0w0/gloop/internal/orchestrator"
)

// ==================== automation 命令 ====================

func runAutomationCmd(ctx context.Context, log *slog.Logger, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "用法: gloop automation <list|run|show>")
		return 2
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "list":
		return runAutomationList(ctx, log, rest)
	case "templates":
		return runAutomationTemplates(ctx, log, rest)
	case "run":
		return runAutomationRun(ctx, log, rest)
	case "show":
		return runAutomationShow(ctx, log, rest)
	case "edit", "update":
		return runAutomationEdit(ctx, log, rest)
	case "enable":
		return runAutomationSetEnabled(ctx, log, rest, true)
	case "disable":
		return runAutomationSetEnabled(ctx, log, rest, false)
	default:
		fmt.Fprintf(os.Stderr, "未知 automation 子命令 %q\n", sub)
		return 2
	}
}

func runAutomationList(_ context.Context, log *slog.Logger, args []string) int {
	fs := flag.NewFlagSet("automation list", flag.ContinueOnError)
	dataDir := fs.String("data-dir", "", "数据目录")
	jsonOut := fs.Bool("json", false, "JSON 输出")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	r := newRunner(*dataDir, log)
	if _, err := r.bootstrap(); err != nil {
		log.Error(logPrefix+" bootstrap 失败", "err", err)
		return 1
	}
	items, err := r.engine.ListAutomations()
	if err != nil {
		log.Error(logPrefix+" 列 automation 失败", "err", err)
		return 1
	}
	if *jsonOut {
		if encErr := json.NewEncoder(os.Stdout).Encode(items); encErr != nil {
			fmt.Fprintf(os.Stderr, "[warn] encode JSON 输出失败: %v\n", encErr)
		}
		return 0
	}
	if len(items) == 0 {
		fmt.Println("（暂无 automation，请先 `gloop init`）")
		return 0
	}
	fmt.Printf("%-22s  %-8s  %-30s  %s\n", "ID", "ENABLED", "NAME", "TRIGGER")
	for _, a := range items {
		enabled := "  off"
		if a.Enabled {
			enabled = "  on"
		}
		fmt.Printf("%-22s  %-8s  %-30s  %s\n", a.ID, enabled, a.Name, a.Trigger)
	}
	return 0
}

func runAutomationTemplates(_ context.Context, log *slog.Logger, args []string) int {
	fs := flag.NewFlagSet("automation templates", flag.ContinueOnError)
	dataDir := fs.String("data-dir", "", "数据目录")
	jsonOut := fs.Bool("json", false, "JSON 输出")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	r := newRunner(*dataDir, log)
	if _, err := r.bootstrap(); err != nil {
		log.Error(logPrefix+" bootstrap 失败", "err", err)
		return 1
	}
	items, err := r.engine.ListAutomationTemplates()
	if err != nil {
		log.Error(logPrefix+" 列 automation templates 失败", "err", err)
		return 1
	}
	if *jsonOut {
		if encErr := json.NewEncoder(os.Stdout).Encode(items); encErr != nil {
			fmt.Fprintf(os.Stderr, "[warn] encode JSON 输出失败: %v\n", encErr)
		}
		return 0
	}
	if len(items) == 0 {
		fmt.Println("（暂无 automation 模板）")
		return 0
	}
	fmt.Printf("%-22s  %-8s  %-30s  %s\n", "ID", "ENABLED", "NAME", "TYPE")
	for _, a := range items {
		enabled := "  off"
		if a.Enabled {
			enabled = "  on"
		}
		fmt.Printf("%-22s  %-8s  %-30s  %s\n", a.ID, enabled, a.Name, a.QuestType)
	}
	return 0
}

func runAutomationRun(ctx context.Context, log *slog.Logger, args []string) int {
	fs := flag.NewFlagSet("automation run", flag.ContinueOnError)
	dataDir := fs.String("data-dir", "", "数据目录")
	watch := fs.Bool("watch", true, "挂住等结果，默认开启以保持本地 automation 执行进程存活")
	noWatch := fs.Bool("no-watch", false, "触发后立即返回（仅适合 auto_start=false 的 automation）")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "需要 automation id:  gloop automation run <id>")
		return 2
	}
	id := fs.Arg(0)
	r := newRunner(*dataDir, log)
	tok, err := r.bootstrap()
	if err != nil {
		log.Error(logPrefix+" bootstrap 失败", "err", err)
		return 1
	}
	qid, err := r.engine.RunAutomation(ctx, id)
	if err != nil {
		log.Error(logPrefix+" 运行 automation 失败", "id", id, "err", err)
		return 1
	}
	fmt.Printf("✅ automation 已触发: %s\n", id)
	fmt.Printf("   委托 ID: %s\n", qid)

	cfg, errGA := r.engine.GetAutomation(id)
	if errGA != nil {
		log.Error(logPrefix+" 获取 automation 配置失败", "id", id, "err", errGA)
		return 1
	}
	if cfg != nil && !cfg.AutoStart {
		fmt.Printf("   状态: 已加入收件箱（inbox），待确认后启动\n")
		fmt.Printf("   确认启动: gloop inbox accept %s\n", qid)
		return 0
	}

	if *noWatch || !*watch {
		fmt.Printf("   提示: auto-start automation 需要保持 CLI 进程存活；已忽略立即返回选项，避免委托停留在 orphan running 状态。\n")
	}
	fmt.Printf("   Dashboard: %s\n", tok.DashboardURL())
	return sseWatchLocal(ctx, r, qid)
}

func runAutomationShow(_ context.Context, log *slog.Logger, args []string) int {
	fs := flag.NewFlagSet("automation show", flag.ContinueOnError)
	dataDir := fs.String("data-dir", "", "数据目录")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "需要 automation id:  gloop automation show <id>")
		return 2
	}
	id := fs.Arg(0)
	r := newRunner(*dataDir, log)
	if _, err := r.bootstrap(); err != nil {
		log.Error(logPrefix+" bootstrap 失败", "err", err)
		return 1
	}
	a, err := r.engine.GetAutomation(id)
	if err != nil {
		log.Error(logPrefix+" 获取 automation 失败", "id", id, "err", err)
		return 1
	}
	b, marshalErr := json.MarshalIndent(a, "", "  ")
	if marshalErr != nil {
		fmt.Fprintf(os.Stderr, "[warn] marshal automation: %v\n", marshalErr)
	}
	fmt.Println(string(b))
	return 0
}

func runAutomationSetEnabled(_ context.Context, log *slog.Logger, args []string, enabled bool) int {
	name := "disable"
	if enabled {
		name = "enable"
	}
	fs := flag.NewFlagSet("automation "+name, flag.ContinueOnError)
	dataDir := fs.String("data-dir", "", "数据目录")
	jsonOut := fs.Bool("json", false, "JSON 输出")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() == 0 {
		fmt.Fprintf(os.Stderr, "需要 automation id:  gloop automation %s <id>\n", name)
		return 2
	}
	id := fs.Arg(0)
	r := newRunner(*dataDir, log)
	if _, err := r.bootstrap(); err != nil {
		log.Error(logPrefix+" bootstrap 失败", "err", err)
		return 1
	}
	item, err := r.engine.SetAutomationEnabled(id, enabled)
	if err != nil {
		log.Error(logPrefix+" 设置 automation 状态失败", "id", id, "err", err)
		return 1
	}
	if *jsonOut {
		if encErr := json.NewEncoder(os.Stdout).Encode(item); encErr != nil {
			fmt.Fprintf(os.Stderr, "[warn] encode JSON 输出失败: %v\n", encErr)
		}
		return 0
	}
	state := "禁用"
	if enabled {
		state = "启用"
	}
	fmt.Printf("✅ automation 已%s: %s\n", state, id)
	return 0
}

func runAutomationEdit(_ context.Context, log *slog.Logger, args []string) int {
	fs := flag.NewFlagSet("automation edit", flag.ContinueOnError)
	dataDir := fs.String("data-dir", "", "数据目录")
	name := fs.String("name", "", "名称")
	description := fs.String("description", "", "描述")
	query := fs.String("query", "", "委托内容")
	questType := fs.String("type", "", "委托类型：execute | design")
	workDir := fs.String("work-dir", "", "工作目录")
	workspaceModeFlag := fs.String("workspace-mode", "", "工作区模式：auto | worktree | copy | readonly")
	triggerFlag := fs.String("trigger", "", "触发类型：manual | schedule")
	cron := fs.String("cron", "", "cron 表达式")
	autoStart := fs.Bool("auto-start", false, "创建后自动开始")
	noAutoStart := fs.Bool("no-auto-start", false, "创建后进入 inbox")
	autoApply := fs.Bool("auto-apply", false, "通过后自动 apply")
	noAutoApply := fs.Bool("no-auto-apply", false, "关闭自动 apply")
	noAutoSpawnExecute := fs.Bool("no-auto-spawn-execute", false, "关闭 design 自动创建执行委托")
	autoSpawnExecute := fs.Bool("auto-spawn-execute", false, "design 完成后自动创建执行委托")
	jsonOut := fs.Bool("json", false, "JSON 输出")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "需要 automation id:  gloop automation edit <id> [flags]")
		return 2
	}
	id := fs.Arg(0)
	upd := orchestrator.AutomationUpdate{}
	var workspaceModeErr error
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "name":
			upd.Name = name
		case "description":
			upd.Description = description
		case "query":
			upd.Query = query
		case "type":
			upd.QuestType = questType
		case "work-dir":
			upd.WorkingDir = workDir
		case "workspace-mode":
			wm, err := parseAutomationWorkspaceMode(*workspaceModeFlag)
			if err != nil {
				workspaceModeErr = err
				return
			}
			upd.WorkspaceMode = &wm
		case "trigger":
			t := fsstore.AutomationTrigger(*triggerFlag)
			upd.Trigger = &t
		case "cron":
			upd.Cron = cron
		case "auto-start":
			v := true
			upd.AutoStart = &v
		case "no-auto-start":
			v := false
			upd.AutoStart = &v
		case "auto-apply":
			v := true
			upd.AutoApply = &v
		case "no-auto-apply":
			v := false
			upd.AutoApply = &v
		case "auto-spawn-execute":
			v := true
			upd.AutoSpawnExecute = &v
		case "no-auto-spawn-execute":
			v := false
			upd.AutoSpawnExecute = &v
		}
	})
	if workspaceModeErr != nil {
		fmt.Fprintln(os.Stderr, workspaceModeErr.Error())
		return 2
	}
	if *autoStart && *noAutoStart {
		fmt.Fprintln(os.Stderr, "--auto-start 和 --no-auto-start 不能同时使用")
		return 2
	}
	if *autoApply && *noAutoApply {
		fmt.Fprintln(os.Stderr, "--auto-apply 和 --no-auto-apply 不能同时使用")
		return 2
	}
	if *autoSpawnExecute && *noAutoSpawnExecute {
		fmt.Fprintln(os.Stderr, "--auto-spawn-execute 和 --no-auto-spawn-execute 不能同时使用")
		return 2
	}
	r := newRunner(*dataDir, log)
	if _, err := r.bootstrap(); err != nil {
		log.Error(logPrefix+" bootstrap 失败", "err", err)
		return 1
	}
	item, err := r.engine.UpdateAutomation(id, upd)
	if err != nil {
		log.Error(logPrefix+" 更新 automation 失败", "id", id, "err", err)
		return 1
	}
	if *jsonOut {
		if encErr := json.NewEncoder(os.Stdout).Encode(item); encErr != nil {
			fmt.Fprintf(os.Stderr, "[warn] encode JSON 输出失败: %v\n", encErr)
		}
		return 0
	}
	fmt.Printf("✅ automation 已更新: %s\n", id)
	return 0
}

// ==================== inbox 命令 ====================

func runInboxCmd(ctx context.Context, log *slog.Logger, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "用法: gloop inbox <list|accept|reject>")
		return 2
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "list", "ls":
		return runInboxList(ctx, log, rest)
	case "edit", "update":
		return runInboxEdit(ctx, log, rest)
	case "accept":
		return runInboxAccept(ctx, log, rest)
	case "reject":
		return runInboxReject(ctx, log, rest)
	default:
		fmt.Fprintf(os.Stderr, "未知 inbox 子命令 %q\n", sub)
		return 2
	}
}

func runInboxList(_ context.Context, log *slog.Logger, args []string) int {
	fs := flag.NewFlagSet("inbox list", flag.ContinueOnError)
	dataDir := fs.String("data-dir", "", "数据目录")
	jsonOut := fs.Bool("json", false, "JSON 输出")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	r := newRunner(*dataDir, log)
	if _, err := r.bootstrap(); err != nil {
		log.Error(logPrefix+" bootstrap 失败", "err", err)
		return 1
	}
	items, err := r.engine.ListInbox()
	if err != nil {
		log.Error(logPrefix+" 列 inbox 失败", "err", err)
		return 1
	}
	if *jsonOut {
		if encErr := json.NewEncoder(os.Stdout).Encode(items); encErr != nil {
			fmt.Fprintf(os.Stderr, "[warn] encode JSON 输出失败: %v\n", encErr)
		}
		return 0
	}
	if len(items) == 0 {
		fmt.Println("（收件箱空）")
		return 0
	}
	fmt.Printf("📥 收件箱 (%d)\n\n", len(items))
	fmt.Printf("%-22s  %-14s  %-20s  %s\n", "QID", "TYPE", "CREATED", "QUERY")
	for _, q := range items {
		query := q.Query
		if len(query) > 30 {
			query = query[:30] + "…"
		}
		created := fsstore.FromMs(q.CreatedAtMs).Format(time.DateTime)
		fmt.Printf("%-22s  %-14s  %-20s  %s\n", q.ID, q.Type, created, query)
	}
	return 0
}

func runInboxEdit(_ context.Context, log *slog.Logger, args []string) int {
	fs := flag.NewFlagSet("inbox edit", flag.ContinueOnError)
	dataDir := fs.String("data-dir", "", "数据目录")
	query := fs.String("query", "", "委托内容")
	questType := fs.String("type", "", "委托类型：execute | design")
	workDir := fs.String("work-dir", "", "工作目录")
	workspaceModeFlag := fs.String("workspace-mode", "", "工作区模式：auto | worktree | copy | readonly")
	jsonOut := fs.Bool("json", false, "JSON 输出")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "需要 quest id:  gloop inbox edit <qid> [flags]")
		return 2
	}
	upd := orchestrator.InboxUpdate{}
	var workspaceModeErr error
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "query":
			upd.Query = query
		case "type":
			upd.QuestType = questType
		case "work-dir":
			upd.WorkDir = workDir
		case "workspace-mode":
			wm, err := parseAutomationWorkspaceMode(*workspaceModeFlag)
			if err != nil {
				workspaceModeErr = err
				return
			}
			upd.WorkspaceMode = &wm
		}
	})
	if workspaceModeErr != nil {
		fmt.Fprintln(os.Stderr, workspaceModeErr.Error())
		return 2
	}
	r := newRunner(*dataDir, log)
	if _, err := r.bootstrap(); err != nil {
		log.Error(logPrefix+" bootstrap 失败", "err", err)
		return 1
	}
	q, err := r.engine.UpdateInboxItem(fs.Arg(0), upd)
	if err != nil {
		log.Error(logPrefix+" inbox edit 失败", "qid", fs.Arg(0), "err", err)
		return 1
	}
	if *jsonOut {
		if encErr := json.NewEncoder(os.Stdout).Encode(q); encErr != nil {
			fmt.Fprintf(os.Stderr, "[warn] encode JSON 输出失败: %v\n", encErr)
		}
		return 0
	}
	fmt.Printf("✅ inbox 委托已更新: qid=%s\n", q.ID)
	return 0
}

func parseAutomationWorkspaceMode(raw string) (model.WorkspaceMode, error) {
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

func runInboxAccept(ctx context.Context, log *slog.Logger, args []string) int {
	fs := flag.NewFlagSet("inbox accept", flag.ContinueOnError)
	dataDir := fs.String("data-dir", "", "数据目录")
	watch := fs.Bool("watch", false, "挂住等结果")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "需要 quest id:  gloop inbox accept <qid>")
		return 2
	}
	qid := fs.Arg(0)
	r := newRunner(*dataDir, log)
	tok, err := r.bootstrap()
	if err != nil {
		log.Error(logPrefix+" bootstrap 失败", "err", err)
		return 1
	}
	if err := r.engine.AcceptInboxItem(ctx, qid); err != nil {
		log.Error(logPrefix+" accept 失败", "qid", qid, "err", err)
		return 1
	}
	fmt.Printf("✅ 已接受委托: qid=%s\n", qid)
	if !*watch {
		return 0
	}
	fmt.Printf("   Dashboard: %s\n", tok.DashboardURL())
	return sseWatchLocal(ctx, r, qid)
}

func runInboxReject(_ context.Context, log *slog.Logger, args []string) int {
	fs := flag.NewFlagSet("inbox reject", flag.ContinueOnError)
	dataDir := fs.String("data-dir", "", "数据目录")
	reason := fs.String("reason", "", "拒绝原因")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "需要 quest id:  gloop inbox reject <qid>")
		return 2
	}
	qid := fs.Arg(0)
	r := newRunner(*dataDir, log)
	if _, err := r.bootstrap(); err != nil {
		log.Error(logPrefix+" bootstrap 失败", "err", err)
		return 1
	}
	if err := r.engine.RejectInboxItem(qid, *reason); err != nil {
		log.Error(logPrefix+" reject 失败", "qid", qid, "err", err)
		return 1
	}
	fmt.Printf("🗑️  已拒绝委托: qid=%s\n", qid)
	if *reason != "" {
		fmt.Printf("   原因: %s\n", *reason)
	}
	return 0
}
