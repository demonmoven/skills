package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
)

// ==================== context 命令（用户 + agent syscall 通用） ====================
//
// context 命令是全局的，不依赖 quest 上下文。
// agent 在 quest 里也可以调用，用来查询用户工作上下文。

func runContextCmd(ctx context.Context, log *slog.Logger, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "用法: gloop context <list|show|summary>")
		return 2
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "list", "ls":
		return runContextList(ctx, log, rest)
	case "show":
		return runContextShow(ctx, log, rest)
	case "get":
		return runContextGet(ctx, log, rest)
	case "summary":
		return runContextSummary(ctx, log, rest)
	case "refresh":
		return runContextRefresh(ctx, log, rest)
	case "export":
		return runContextExport(ctx, log, rest)
	case "write-dim":
		return runContextWriteDim(ctx, log, rest)
	case "write-summary":
		return runContextWriteSummary(ctx, log, rest)
	default:
		fmt.Fprintf(os.Stderr, "未知 context 子命令 %q\n", sub)
		return 2
	}
}

func runContextGet(_ context.Context, log *slog.Logger, args []string) int {
	fs := flag.NewFlagSet("context get", flag.ContinueOnError)
	workDir := fs.String("work-dir", "", "quest 工作目录（默认自动发现）")
	jsonOut := fs.Bool("json", false, "JSON 输出")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "需要 block 名称: gloop context get <block-name>")
		return 2
	}
	blockName := fs.Arg(0)
	wd := *workDir
	if wd == "" {
		if _, discovered, _, _, code := loadAgentQuestQuiet(); code == 0 {
			wd = discovered
		}
	}
	if wd == "" {
		cwd, err := os.Getwd()
		if err != nil {
			log.Error(logPrefix+" 获取当前目录失败", "err", err)
			return 1
		}
		wd = cwd
	}
	path := filepath.Join(wd, fsstore.GloopDir, "context", contextBlockFilename(blockName))
	raw, err := os.ReadFile(path)
	if err != nil {
		log.Error(logPrefix+" 读取文件式上下文失败", "block", blockName, "path", path, "err", err)
		return 1
	}
	content := string(raw)
	if *jsonOut {
		if encErr := json.NewEncoder(os.Stdout).Encode(map[string]any{
			"ok":      true,
			"block":   blockName,
			"path":    path,
			"content": content,
		}); encErr != nil {
			fmt.Fprintf(os.Stderr, "[warn] encode JSON 输出失败: %v\n", encErr)
		}
		return 0
	}
	fmt.Print(content)
	return 0
}

func contextBlockFilename(name string) string {
	safe := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= 'A' && r <= 'Z':
			return r
		case r >= '0' && r <= '9':
			return r
		case r == '_' || r == '-':
			return r
		default:
			return '_'
		}
	}, name)
	return safe + ".md"
}

func runContextList(_ context.Context, log *slog.Logger, args []string) int {
	fs := flag.NewFlagSet("context list", flag.ContinueOnError)
	dataDir := fs.String("data-dir", "", "数据目录")
	jsonOut := fs.Bool("json", false, "JSON 输出")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	root, err := fsstore.Open(*dataDir)
	if err != nil {
		log.Error(logPrefix+" 打开数据目录失败", "err", err)
		return 1
	}
	dims, err := root.ListContextDims()
	if err != nil {
		log.Error(logPrefix+" 列上下文维度失败", "err", err)
		return 1
	}
	if *jsonOut {
		if encErr := json.NewEncoder(os.Stdout).Encode(map[string]any{
			"ok":    true,
			"items": dims,
			"total": len(dims),
		}); encErr != nil {
			fmt.Fprintf(os.Stderr, "[warn] encode JSON 输出失败: %v\n", encErr)
		}
		return 0
	}
	if len(dims) == 0 {
		fmt.Println("（暂无上下文，运行 gloop automation run auto_context_refresh 刷新）")
		return 0
	}
	meta, _ := root.GetContextMeta()
	if meta != nil && meta.UpdatedAtMs > 0 {
		fmt.Printf("更新时间: %s\n\n", fsstore.FromMs(meta.UpdatedAtMs).Format("2006-01-02 15:04:05"))
	}
	fmt.Printf("%-20s  %-20s  %-10s  %s\n", "DIM", "TITLE", "SIZE", "UPDATED")
	for _, d := range dims {
		updated := fsstore.FromMs(d.UpdatedAtMs).Format("01-02 15:04")
		fmt.Printf("%-20s  %-20s  %-10d  %s\n", d.Name, d.Title, d.SizeBytes, updated)
	}
	return 0
}

func runContextShow(_ context.Context, log *slog.Logger, args []string) int {
	fs := flag.NewFlagSet("context show", flag.ContinueOnError)
	dataDir := fs.String("data-dir", "", "数据目录")
	jsonOut := fs.Bool("json", false, "JSON 输出")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "需要维度名:  gloop context show <dim>")
		return 2
	}
	name := fs.Arg(0)
	root, err := fsstore.Open(*dataDir)
	if err != nil {
		log.Error(logPrefix+" 打开数据目录失败", "err", err)
		return 1
	}
	dim, err := root.ReadContextDim(name)
	if err != nil {
		log.Error(logPrefix+" 读取上下文维度失败", "dim", name, "err", err)
		return 1
	}
	if *jsonOut {
		if encErr := json.NewEncoder(os.Stdout).Encode(map[string]any{
			"ok": true,
			"dim": map[string]any{
				"name":          dim.Name,
				"title":         dim.Title,
				"description":   dim.Description,
				"updated_at_ms": dim.UpdatedAtMs,
				"size_bytes":    dim.SizeBytes,
				"body":          dim.Body,
			},
		}); encErr != nil {
			fmt.Fprintf(os.Stderr, "[warn] encode JSON 输出失败: %v\n", encErr)
		}
		return 0
	}
	fmt.Printf("# %s\n\n", dim.Title)
	fmt.Printf("_更新: %s | 大小: %d 字节_\n\n",
		fsstore.FromMs(dim.UpdatedAtMs).Format("2006-01-02 15:04:05"),
		dim.SizeBytes)
	fmt.Println(dim.Body)
	return 0
}

func runContextSummary(_ context.Context, log *slog.Logger, args []string) int {
	fs := flag.NewFlagSet("context summary", flag.ContinueOnError)
	dataDir := fs.String("data-dir", "", "数据目录")
	jsonOut := fs.Bool("json", false, "JSON 输出")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	root, err := fsstore.Open(*dataDir)
	if err != nil {
		log.Error(logPrefix+" 打开数据目录失败", "err", err)
		return 1
	}
	summary, err := root.ReadContextSummary()
	if err != nil {
		log.Error(logPrefix+" 读取上下文摘要失败", "err", err)
		return 1
	}
	if *jsonOut {
		if encErr := json.NewEncoder(os.Stdout).Encode(map[string]any{
			"ok":      true,
			"summary": summary,
		}); encErr != nil {
			fmt.Fprintf(os.Stderr, "[warn] encode JSON 输出失败: %v\n", encErr)
		}
		return 0
	}
	if summary == "" {
		fmt.Println("（暂无上下文摘要）")
		return 0
	}
	fmt.Println(summary)
	return 0
}

func runContextRefresh(ctx context.Context, log *slog.Logger, args []string) int {
	fs := flag.NewFlagSet("context refresh", flag.ContinueOnError)
	dataDir := fs.String("data-dir", "", "数据目录")
	watch := fs.Bool("watch", true, "挂住等结果，默认开启以保持本地上下文刷新进程存活")
	noWatch := fs.Bool("no-watch", false, "触发后立即返回（会被忽略，避免刷新委托孤儿化）")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	r := newRunner(*dataDir, log)
	tok, err := r.bootstrap()
	if err != nil {
		log.Error(logPrefix+" bootstrap 失败", "err", err)
		return 1
	}
	qid, err := r.engine.RunAutomation(ctx, "auto_context_refresh")
	if err != nil {
		log.Error(logPrefix+" 触发上下文刷新失败", "err", err)
		return 1
	}
	fmt.Printf("上下文刷新已启动: %s\n", qid)
	if *noWatch || !*watch {
		fmt.Printf("提示: 上下文刷新需要保持 CLI 进程存活；已忽略立即返回选项，避免委托停留在 orphan running 状态。\n")
	}
	fmt.Printf("Dashboard: %s\n", tok.DashboardURL())
	return sseWatchLocal(ctx, r, qid)
}

func runContextExport(_ context.Context, log *slog.Logger, args []string) int {
	fs := flag.NewFlagSet("context export", flag.ContinueOnError)
	dataDir := fs.String("data-dir", "", "数据目录")
	format := fs.String("format", "okf", "导出格式，目前支持 okf")
	outDir := fs.String("out", "", "导出目录")
	jsonOut := fs.Bool("json", false, "JSON 输出")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *format != "okf" {
		fmt.Fprintf(os.Stderr, "不支持的导出格式 %q，目前仅支持 okf\n", *format)
		return 2
	}
	if *outDir == "" {
		fmt.Fprintln(os.Stderr, "需要导出目录: gloop context export --format okf --out <dir>")
		return 2
	}
	root, err := fsstore.Open(*dataDir)
	if err != nil {
		log.Error(logPrefix+" 打开数据目录失败", "err", err)
		return 1
	}
	absOut, err := filepath.Abs(*outDir)
	if err != nil {
		log.Error(logPrefix+" 解析导出目录失败", "err", err)
		return 1
	}
	bundle, err := root.WriteKnowledgeExport(absOut)
	if err != nil {
		log.Error(logPrefix+" 导出 knowledge bundle 失败", "err", err)
		return 1
	}
	if *jsonOut {
		if encErr := json.NewEncoder(os.Stdout).Encode(map[string]any{
			"ok":         true,
			"format":     bundle.Format,
			"out":        absOut,
			"file_count": len(bundle.Files),
			"files":      bundle.Files,
		}); encErr != nil {
			fmt.Fprintf(os.Stderr, "[warn] encode JSON 输出失败: %v\n", encErr)
		}
		return 0
	}
	fmt.Printf("Knowledge bundle 已导出: %s\n", absOut)
	fmt.Printf("格式: %s\n", bundle.Format)
	fmt.Printf("文件数: %d\n", len(bundle.Files))
	for _, f := range bundle.Files {
		fmt.Printf("  %s\n", f.Path)
	}
	return 0
}

// runContextWriteDim 写入（覆盖）一个上下文维度。
// 优先走 server HTTP API，server 没跑就 fallback 到本地直接写。
func runContextWriteDim(_ context.Context, log *slog.Logger, args []string) int {
	fs := flag.NewFlagSet("context write-dim", flag.ContinueOnError)
	dataDir := fs.String("data-dir", "", "数据目录")
	content := fs.String("content", "", "维度内容（Markdown 格式）")
	contentFile := fs.String("content-file", "", "从文件读取内容（与 --content 二选一）")
	jsonOut := fs.Bool("json", false, "JSON 输出")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "需要维度名:  gloop context write-dim <dim>")
		return 2
	}
	dim := fs.Arg(0)

	body := *content
	if *contentFile != "" {
		data, err := os.ReadFile(*contentFile)
		if err != nil {
			if *jsonOut {
				return writeJSONError(1, "read_failed", "读取内容文件失败: "+err.Error())
			}
			log.Error(logPrefix+" 读取内容文件失败", "err", err)
			return 1
		}
		body = string(data)
	}
	if strings.TrimSpace(body) == "" {
		fmt.Fprintln(os.Stderr, "内容不能为空（--content 或 --content-file）")
		return 2
	}

	dd := resolveDataDirOrDefault(*dataDir)

	// 优先走 server API
	if proxy := tryServerProxy(dd); proxy != nil {
		if err := proxy.writeContextDim(dim, body); err != nil {
			if *jsonOut {
				return writeJSONError(1, "write_failed", err.Error())
			}
			log.Error(logPrefix+" 写入维度失败", "err", err)
			return 1
		}
		if *jsonOut {
			return writeJSONLine(map[string]any{
				"ok":      true,
				"dim":     dim,
				"message": "维度已写入",
			})
		}
		fmt.Printf("维度 %s 已写入\n", dim)
		return 0
	}

	// fallback 到本地直接写
	root, err := fsstore.Open(dd)
	if err != nil {
		if *jsonOut {
			return writeJSONError(1, "internal_error", "打开数据目录失败: "+err.Error())
		}
		log.Error(logPrefix+" 打开数据目录失败", "err", err)
		return 1
	}
	info, err := root.WriteContextDim(dim, body)
	if err != nil {
		if *jsonOut {
			return writeJSONError(1, "write_failed", err.Error())
		}
		log.Error(logPrefix+" 写入维度失败", "err", err)
		return 1
	}
	if *jsonOut {
		return writeJSONLine(map[string]any{
			"ok":      true,
			"dim":     info.Name,
			"title":   info.Title,
			"size":    info.SizeBytes,
			"updated": info.UpdatedAtMs,
		})
	}
	fmt.Printf("维度 %s 已写入（%d 字节）\n", dim, info.SizeBytes)
	return 0
}

// runContextWriteSummary 写入总览摘要。
// 优先走 server HTTP API，server 没跑就 fallback 到本地直接写。
func runContextWriteSummary(_ context.Context, log *slog.Logger, args []string) int {
	fs := flag.NewFlagSet("context write-summary", flag.ContinueOnError)
	dataDir := fs.String("data-dir", "", "数据目录")
	summary := fs.String("summary", "", "摘要文本（1-3 段）")
	summaryFile := fs.String("summary-file", "", "从文件读取摘要（与 --summary 二选一）")
	jsonOut := fs.Bool("json", false, "JSON 输出")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	body := *summary
	if *summaryFile != "" {
		data, err := os.ReadFile(*summaryFile)
		if err != nil {
			if *jsonOut {
				return writeJSONError(1, "read_failed", "读取摘要文件失败: "+err.Error())
			}
			log.Error(logPrefix+" 读取摘要文件失败", "err", err)
			return 1
		}
		body = string(data)
	}
	if strings.TrimSpace(body) == "" {
		fmt.Fprintln(os.Stderr, "摘要不能为空（--summary 或 --summary-file）")
		return 2
	}

	dd := resolveDataDirOrDefault(*dataDir)

	// 优先走 server API
	if proxy := tryServerProxy(dd); proxy != nil {
		if err := proxy.writeContextSummary(body); err != nil {
			if *jsonOut {
				return writeJSONError(1, "write_failed", err.Error())
			}
			log.Error(logPrefix+" 写入摘要失败", "err", err)
			return 1
		}
		if *jsonOut {
			return writeJSONLine(map[string]any{
				"ok":      true,
				"message": "摘要已写入",
			})
		}
		fmt.Println("摘要已写入")
		return 0
	}

	// fallback 到本地直接写
	root, err := fsstore.Open(dd)
	if err != nil {
		if *jsonOut {
			return writeJSONError(1, "internal_error", "打开数据目录失败: "+err.Error())
		}
		log.Error(logPrefix+" 打开数据目录失败", "err", err)
		return 1
	}
	if err := root.WriteContextSummary(body); err != nil {
		if *jsonOut {
			return writeJSONError(1, "write_failed", err.Error())
		}
		log.Error(logPrefix+" 写入摘要失败", "err", err)
		return 1
	}
	if *jsonOut {
		return writeJSONLine(map[string]any{
			"ok":      true,
			"message": "摘要已写入",
		})
	}
	fmt.Println("摘要已写入")
	return 0
}
