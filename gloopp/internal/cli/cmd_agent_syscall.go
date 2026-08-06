package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"code.byted.org/lihuanyu.0w0/gloop/internal/events"
	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
)

func runPhaseCmd(_ context.Context, log *slog.Logger, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "用法: gloop phase <info|done|fail>")
		return 2
	}
	switch args[0] {
	case "info":
		return runPhaseInfo(log, args[1:])
	case "done":
		return runPhaseDone(log, args[1:])
	case "fail":
		return runPhaseFail(log, args[1:])
	default:
		fmt.Fprintf(os.Stderr, "未知 phase 子命令 %q\n", args[0])
		return 2
	}
}

func runReviewCmd(_ context.Context, log *slog.Logger, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "用法: gloop review <pass|request-changes|reject>")
		return 2
	}
	switch args[0] {
	case "pass":
		return runReviewSignal(log, "pass", args[1:])
	case "request-changes", "request_changes", "changes", "rework":
		return runReviewSignal(log, "request_changes", args[1:])
	case "reject":
		return runReviewSignal(log, "reject", args[1:])
	default:
		fmt.Fprintf(os.Stderr, "未知 review 子命令 %q\n", args[0])
		return 2
	}
}

func runPhaseInfo(log *slog.Logger, args []string) int {
	fs := flag.NewFlagSet("phase info", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	ctx, _, _, q, code := loadAgentQuest(log)
	if code != 0 {
		return code
	}
	return writeJSONLine(map[string]any{
		"ok":               true,
		"quest_id":         q.ID,
		"session_id":       ctx.SessionID,
		"phase":            ctx.Phase,
		"phase_name":       ctx.PhaseName,
		"adventurer_id":    ctx.AdventurerID,
		"adventurer_class": ctx.AdventurerClass,
		"quest_status":     q.Status,
		"quest_intensity":  q.Intensity,
		"rework_count":     q.ReworkCount,
		"review_hints":     q.ReviewHints,
		"workspace_path":   q.WorkspacePath,
	})
}

func runReviewSignal(log *slog.Logger, verdict string, args []string) int {
	fs := flag.NewFlagSet("review "+verdict, flag.ContinueOnError)
	comment := fs.String("comment", "", "评审总评")
	hints := fs.String("hints", "", "返工建议（request_changes 必填）")
	score := fs.Int("score", 0, "质量评分 1-10（pass 使用）")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *comment == "" {
		fmt.Fprintln(os.Stderr, "缺失 --comment")
		return 2
	}
	if verdict == "request_changes" && *hints == "" {
		fmt.Fprintln(os.Stderr, "request_changes 缺失 --hints")
		return 2
	}
	if *score < 0 || *score > 10 {
		fmt.Fprintln(os.Stderr, "--score 必须在 0-10 之间")
		return 2
	}

	ctx, workDir, _, q, code := loadAgentQuest(log)
	if code != 0 {
		return code
	}
	if proxy := tryServerProxy(ctx.DataDir); proxy != nil {
		if err := proxy.reviewQuest(q.ID, ctx.SessionID, verdict, *comment, *hints, *score); err != nil {
			return writeJSONError(1, "review_failed", err.Error())
		}
		return writeJSONLine(map[string]any{
			"ok":      true,
			"verdict": verdict,
			"comment": *comment,
			"hints":   *hints,
			"score":   *score,
			"message": "评审结论已提交",
		})
	}

	return writeSignal(log, signalWorkDir(q, workDir), ctx.SessionID, fsstore.PhaseSignal{
		OK:           true,
		Message:      "评审结论已提交",
		PhaseEnded:   true,
		PhaseVerdict: verdict,
		PhaseComment: *comment,
		PhaseHints:   *hints,
		PhaseScore:   *score,
		Data: map[string]any{
			"verdict": verdict,
			"comment": *comment,
			"hints":   *hints,
			"score":   *score,
			"source":  "cli",
		},
	})
}

// deliverableFlag 解析可重复的 --deliverable 参数。
// 格式：name=<显示名>,kind=<file|document|link|...>,path=<相对quest目录的本地路径|可选>,url=<外部链接|可选>,description=<描述|可选>
// path 与 url 至少传一个：path 指向工作区内文件（后端按相对路径读取并 inline 展示），url 指向外部链接。
type deliverableFlag struct {
	items []map[string]any
}

func (d *deliverableFlag) String() string { return fmt.Sprintf("%v", d.items) }

func (d *deliverableFlag) Set(raw string) error {
	m := map[string]any{}
	for _, pair := range strings.Split(raw, ",") {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) != 2 {
			return fmt.Errorf("deliverable 格式错误 %q，应为 key=value 以逗号分隔", raw)
		}
		k := strings.TrimSpace(kv[0])
		v := strings.TrimSpace(kv[1])
		if k == "" {
			return fmt.Errorf("deliverable key 为空: %q", pair)
		}
		m[k] = v
	}
	name, _ := m["name"].(string)
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("deliverable 缺少 name: %q", raw)
	}
	if _, ok := m["path"]; !ok {
		if _, ok := m["url"]; !ok {
			return fmt.Errorf("deliverable %q 缺少 path 或 url", name)
		}
	}
	d.items = append(d.items, m)
	return nil
}

func runPhaseDone(log *slog.Logger, args []string) int {
	fs := flag.NewFlagSet("phase done", flag.ContinueOnError)
	summary := fs.String("summary", "", "阶段完成总结")
	status := fs.String("status", "done", "阶段状态：done | blocked")
	impact := fs.String("impact", "", "影响声明 JSON：{what_changed,affected,not_touched,caveats}。HOTL 语义：声明波及什么，让人 on-the-loop 感知")
	var delvs deliverableFlag
	fs.Var(&delvs, "deliverable", "声明本次产出物，可重复。格式: name=<名>,kind=<file|document|link>,path=<相对quest目录的本地路径>或url=<外部链接>,description=<描述>")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *summary == "" {
		fmt.Fprintln(os.Stderr, "缺失 --summary")
		return 2
	}
	if *status != "done" && *status != "blocked" {
		fmt.Fprintln(os.Stderr, "--status 只能是 done 或 blocked")
		return 2
	}
	ctx, workDir, root, q, code := loadAgentQuest(log)
	if code != 0 {
		return code
	}
	// v0.3: maker/checker 统一，不再限制 phase done 只能 warrior 调用

	// 归一化 deliverable.path：warrior 传相对 cwd 或绝对路径，优先转相对 quest 目录的路径（可移植），
	// worktree 模式（workspace 在 quest 目录外）fallback 绝对路径。后端两种路径都支持。
	deliverables := delvs.items
	if len(deliverables) > 0 && workDir != "" {
		questRoot := root.Sub(fsstore.SubdirQuests, q.ID)
		for _, d := range deliverables {
			p, _ := d["path"].(string)
			if p == "" {
				continue
			}
			abs := p
			if !filepath.IsAbs(p) {
				abs = filepath.Join(workDir, p)
			}
			// 优先存相对 quest 目录的路径（可移植）；worktree 模式 workspace 在 quest 目录外时 fallback 绝对路径。
			// 后端 OpenArtifact/ArtifactPath 两种路径都支持（IsAbs 判断）。
			if rel, err := filepath.Rel(questRoot, abs); err == nil && !strings.HasPrefix(rel, "..") {
				d["path"] = rel
			} else {
				d["path"] = abs
			}
		}
	}

	// 优先走 server API（事件流经 event bus）
	if proxy := tryServerProxy(ctx.DataDir); proxy != nil {
		if err := proxy.phaseCheckpoint(q.ID, ctx.SessionID, *summary, *status, deliverables, *impact); err != nil {
			return writeJSONError(1, "phase_checkpoint_failed", err.Error())
		}
		return writeJSONLine(map[string]any{
			"ok":           true,
			"status":       *status,
			"summary":      *summary,
			"deliverables": deliverables,
			"message":      "阶段信号已提交",
		})
	}

	// fallback：直接写信号文件（deliverables 写入 Data 供 extractWarriorDeliverables 读取）
	signalData := map[string]any{
		"status":  *status,
		"summary": *summary,
		"source":  "cli",
	}
	if len(deliverables) > 0 {
		signalData["deliverables"] = deliverables
	}
	if strings.TrimSpace(*impact) != "" {
		signalData["impact"] = *impact
	}
	return writeSignal(log, signalWorkDir(q, workDir), ctx.SessionID, fsstore.PhaseSignal{
		OK:           *status == "done",
		Message:      "阶段结论已提交",
		PhaseEnded:   true,
		PhaseVerdict: *status,
		PhaseComment: *summary,
		Data:         signalData,
	})
}

func runPhaseFail(log *slog.Logger, args []string) int {
	fs := flag.NewFlagSet("phase fail", flag.ContinueOnError)
	reason := fs.String("reason", "", "失败原因")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *reason == "" {
		fmt.Fprintln(os.Stderr, "缺失 --reason")
		return 2
	}
	ctx, workDir, _, q, code := loadAgentQuest(log)
	if code != 0 {
		return code
	}
	// v0.3: maker/checker 统一，不再限制 phase fail 只能 warrior 调用

	// 优先走 server API
	if proxy := tryServerProxy(ctx.DataDir); proxy != nil {
		if err := proxy.phaseCheckpoint(q.ID, ctx.SessionID, *reason, "blocked", nil, ""); err != nil {
			return writeJSONError(1, "phase_checkpoint_failed", err.Error())
		}
		return writeJSONLine(map[string]any{
			"ok":      true,
			"status":  "blocked",
			"reason":  *reason,
			"message": "阶段失败信号已提交",
		})
	}

	// fallback：直接写信号文件
	return writeSignal(log, signalWorkDir(q, workDir), ctx.SessionID, fsstore.PhaseSignal{
		OK:           false,
		Message:      "阶段失败",
		PhaseEnded:   true,
		PhaseVerdict: "blocked",
		PhaseComment: *reason,
		Data: map[string]any{
			"status": "blocked",
			"reason": *reason,
			"source": "cli",
		},
	})
}

func runNoteCmd(_ context.Context, log *slog.Logger, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "用法: gloop note <add|list>")
		return 2
	}
	switch args[0] {
	case "add":
		return runNoteAdd(log, args[1:])
	case "list":
		return runNoteList(log, args[1:])
	default:
		fmt.Fprintf(os.Stderr, "未知 note 子命令 %q\n", args[0])
		return 2
	}
}

func runPostCmd(_ context.Context, log *slog.Logger, args []string) int {
	fs := flag.NewFlagSet("post", flag.ContinueOnError)
	content := fs.String("content", "", "帖子正文（Markdown）")
	replyTo := fs.String("reply-to", "", "回复的目标 post_id 或 quest_id（可选）")
	kind := fs.String("kind", "post", "帖子类型：post | milestone | blocker")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *content == "" && fs.NArg() > 0 {
		*content = strings.Join(fs.Args(), " ")
	}
	if *content == "" {
		fmt.Fprintln(os.Stderr, "缺失 --content")
		return 2
	}
	if *kind != "post" && *kind != "milestone" && *kind != "blocker" {
		fmt.Fprintln(os.Stderr, "--kind 只能是 post / milestone / blocker")
		return 2
	}
	ctx, _, _, q, code := loadAgentQuest(log)
	if code != 0 {
		return code
	}

	// 优先走 server API
	if proxy := tryServerProxy(ctx.DataDir); proxy != nil {
		postID, err := proxy.agentPost(q.ID, ctx.SessionID, *content, *replyTo, *kind)
		if err != nil {
			return writeJSONError(1, "post_failed", err.Error())
		}
		return writeJSONLine(map[string]any{
			"ok":      true,
			"post_id": postID,
			"message": "帖子已发布到 Feed",
		})
	}

	// fallback：直接写事件
	root, err := fsstore.Open(ctx.DataDir)
	if err != nil {
		return writeJSONError(1, "data_dir_open_failed", err.Error())
	}
	_ = root // fallback 路径：event 不经过 engine，post_id 本地生成
	postID := "post_" + strconv.FormatInt(fsstore.NowMs(), 36)
	payload := map[string]any{
		"post_id":  postID,
		"content":  *content,
		"reply_to": *replyTo,
		"kind":     *kind,
	}
	if err := fsstore.NewQuestStore(root).AppendEvent(q.ID, &fsstore.QuestEventRow{
		Timestamp: fsstore.NowMs(),
		Type:      string(events.EvtAgentPost),
		QuestID:   q.ID,
		SessionID: ctx.SessionID,
		Payload:   payload,
	}); err != nil {
		return writeJSONError(1, "post_failed", err.Error())
	}
	return writeJSONLine(map[string]any{"ok": true, "post_id": postID, "message": "帖子已发布到 Feed"})
}

func runNoteAdd(log *slog.Logger, args []string) int {
	fs := flag.NewFlagSet("note add", flag.ContinueOnError)
	tag := fs.String("tag", "", "标签")
	content := fs.String("content", "", "笔记内容")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *content == "" && fs.NArg() > 0 {
		*content = strings.Join(fs.Args(), " ")
	}
	if *content == "" {
		fmt.Fprintln(os.Stderr, "缺失笔记内容")
		return 2
	}
	ctx, _, _, q, code := loadAgentQuest(log)
	if code != 0 {
		return code
	}

	// 优先走 server API（事件流经 event bus）
	if proxy := tryServerProxy(ctx.DataDir); proxy != nil {
		if err := proxy.addNote(q.ID, ctx.SessionID, *content, *tag); err != nil {
			return writeJSONError(1, "note_add_failed", err.Error())
		}
		return writeJSONLine(map[string]any{
			"ok": true,
			"note": map[string]any{
				"content": *content,
				"tag":     *tag,
			},
		})
	}

	// fallback：直接写 fsstore
	root, err := fsstore.Open(ctx.DataDir)
	if err != nil {
		return writeJSONError(1, "data_dir_open_failed", err.Error())
	}
	qs := fsstore.NewQuestStore(root)
	payload := map[string]any{"content": *content, "tag": *tag}
	if err := qs.AppendEvent(q.ID, &fsstore.QuestEventRow{
		Timestamp: fsstore.NowMs(),
		Type:      string(events.EvtQuestNote),
		QuestID:   q.ID,
		SessionID: ctx.SessionID,
		Payload:   payload,
	}); err != nil {
		return writeJSONError(1, "note_add_failed", err.Error())
	}
	return writeJSONLine(map[string]any{"ok": true, "note": payload})
}

func runNoteList(log *slog.Logger, args []string) int {
	fs := flag.NewFlagSet("note list", flag.ContinueOnError)
	limit := fs.Int("limit", 50, "最多显示条数")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	_, _, root, q, code := loadAgentQuest(log)
	if code != 0 {
		return code
	}
	eventsRows, err := fsstore.NewQuestStore(root).ReadEvents(q.ID, 0)
	if err != nil {
		log.Error(logPrefix+" note list 失败", "err", err)
		return 1
	}
	notes := []fsstore.QuestEventRow{}
	for _, ev := range eventsRows {
		if ev.Type == string(events.EvtQuestNote) {
			notes = append(notes, ev)
		}
	}
	if *limit > 0 && len(notes) > *limit {
		notes = notes[len(notes)-*limit:]
	}
	return writeJSONLine(map[string]any{"ok": true, "items": notes})
}

func runQuestInfo(_ context.Context, log *slog.Logger, args []string) int {
	fs := flag.NewFlagSet("quest info", flag.ContinueOnError)
	dataDir := fs.String("data-dir", "", "数据目录")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	var root *fsstore.Root
	var qid string
	if fs.NArg() > 0 {
		qid = fs.Arg(0)
		dd, err := resolveDataDir(*dataDir)
		if err != nil {
			log.Error(logPrefix+" 解析数据目录失败", "err", err)
			return 1
		}
		var openErr error
		root, openErr = fsstore.Open(dd)
		if openErr != nil {
			log.Error(logPrefix+" 打开数据目录失败", "err", openErr)
			return 1
		}
	} else {
		ctx, _, r, _, code := loadAgentQuest(log)
		if code != 0 {
			return code
		}
		root = r
		qid = ctx.QuestID
	}
	q, err := fsstore.NewQuestStore(root).LoadQuest(qid)
	if err != nil {
		log.Error(logPrefix+" 加载委托失败", "qid", qid, "err", err)
		return 1
	}
	cfg, _ := root.LoadConfig()
	hotlAutoClose := false
	if cfg != nil {
		hotlAutoClose = true
	}
	return writeJSONLine(map[string]any{
		"ok":                                 true,
		"id":                                 q.ID,
		"short_id":                           q.ShortID,
		"type":                               q.Type,
		"intensity":                          q.Intensity,
		"status":                             q.Status,
		"query":                              q.Query,
		"rework_count":                       q.ReworkCount,
		"max_rework":                         q.MaxRework,
		"max_turns_per_phase_override":       q.MaxTurnsPerPhaseOverride,
		"max_duration_per_quest_ms_override": q.MaxDurationPerQuestMsOverride,
		"review_hints":                       q.ReviewHints,
		"workspace_mode":                     q.WorkspaceMode,
		"workspace_path":                     q.WorkspacePath,
		"warrior_id":                         q.WarriorID,
		"mage_id":                            q.MageID,
		"created_by":                         q.CreatedBy,
		"hotl_auto_close":                    hotlAutoClose,
	})
}

func runQuestHistory(log *slog.Logger, args []string) int {
	fs := flag.NewFlagSet("quest history", flag.ContinueOnError)
	limit := fs.Int("limit", 5, "最多显示事件条数")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	_, _, root, q, code := loadAgentQuest(log)
	if code != 0 {
		return code
	}
	items, err := fsstore.NewQuestStore(root).ReadEvents(q.ID, *limit)
	if err != nil {
		log.Error(logPrefix+" quest history 失败", "err", err)
		return 1
	}
	return writeJSONLine(map[string]any{"ok": true, "items": items})
}

func runQuestProgress(log *slog.Logger, args []string) int {
	fs := flag.NewFlagSet("quest progress", flag.ContinueOnError)
	percent := fs.Int("percent", -1, "进度百分比 0-100")
	note := fs.String("note", "", "进度说明")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *percent < 0 || *percent > 100 {
		fmt.Fprintln(os.Stderr, "--percent 必须在 0-100 之间")
		return 2
	}
	if *note == "" && fs.NArg() > 0 {
		*note = strings.Join(fs.Args(), " ")
	}
	ctx, _, root, q, code := loadAgentQuest(log)
	if code != 0 {
		return code
	}
	payload := map[string]any{
		"percent": *percent,
		"note":    *note,
		"phase":   ctx.PhaseName,
	}
	if err := fsstore.NewQuestStore(root).AppendEvent(q.ID, &fsstore.QuestEventRow{
		Timestamp: fsstore.NowMs(),
		Type:      string(events.EvtProgressUpdated),
		QuestID:   q.ID,
		SessionID: ctx.SessionID,
		Payload:   payload,
	}); err != nil {
		log.Error(logPrefix+" quest progress 失败", "err", err)
		return 1
	}
	return writeJSONLine(map[string]any{"ok": true, "progress": payload})
}
func writeSignalFromContext(log *slog.Logger, sig fsstore.PhaseSignal) int {
	ctx, workDir, _, q, code := loadAgentQuest(log)
	if code != 0 {
		return code
	}
	return writeSignal(log, signalWorkDir(q, workDir), ctx.SessionID, sig)
}

func writeSignal(log *slog.Logger, workDir string, sid string, sig fsstore.PhaseSignal) int {
	if err := fsstore.WritePhaseSignal(workDir, sid, sig); err != nil {
		log.Error(logPrefix+" 写阶段信号失败", "err", err)
		return 1
	}
	return writeJSONLine(map[string]any{"ok": true, "signal": sig})
}

func requireAgentClass(ctx *fsstore.AgentContext, want string, action string) int {
	if ctx.AdventurerClass == want {
		return 0
	}
	return writeJSONError(3, "permission_denied", fmt.Sprintf("%s 只能由 %s 阶段调用，当前是 %s", action, want, ctx.AdventurerClass))
}

func loadAgentQuest(log *slog.Logger) (*fsstore.AgentContext, string, *fsstore.Root, *fsstore.QuestMeta, int) {
	ctx, workDir, err := fsstore.DiscoverAgentContext("")
	if err != nil {
		log.Error(logPrefix+" 未找到 agent 上下文", "err", err)
		return nil, "", nil, nil, 1
	}
	root, err := fsstore.Open(ctx.DataDir)
	if err != nil {
		log.Error(logPrefix+" 打开数据目录失败", "err", err)
		return nil, "", nil, nil, 1
	}
	q, err := fsstore.NewQuestStore(root).LoadQuest(ctx.QuestID)
	if err != nil {
		log.Error(logPrefix+" 加载委托失败", "qid", ctx.QuestID, "err", err)
		return nil, "", nil, nil, 1
	}
	return ctx, workDir, root, q, 0
}

func signalWorkDir(q *fsstore.QuestMeta, discoveredWorkDir string) string {
	if env := strings.TrimSpace(os.Getenv("GLOOP_SIGNAL_WORKSPACE_PATH")); env != "" {
		return env
	}
	if q != nil && q.WorkspacePath != "" {
		return q.WorkspacePath
	}
	return discoveredWorkDir
}

func writeJSONLine(v any) int {
	enc := json.NewEncoder(os.Stdout)
	if err := enc.Encode(v); err != nil {
		fmt.Fprintf(os.Stderr, "JSON 输出失败: %v\n", err)
		return 1
	}
	return 0
}

func writeJSONError(code int, errCode string, message string) int {
	_ = json.NewEncoder(os.Stdout).Encode(map[string]any{
		"ok":      false,
		"error":   errCode,
		"message": message,
	})
	return code
}
