package cli

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
	"code.byted.org/lihuanyu.0w0/gloop/internal/platformtools"
)

// ==================== code 命令（用户 + agent syscall 通用） ====================
//
// 代码搜索工具，在工作区内做符号级搜索，返回结构化结果。
// agent 在 quest 里也可以调用，自动从 quest 上下文取工作区路径。

func runCodeCmd(ctx context.Context, log *slog.Logger, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "用法: gloop code <search>")
		return 2
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "search", "s":
		return runCodeSearchCmd(ctx, log, rest)
	default:
		fmt.Fprintf(os.Stderr, "未知 code 子命令 %q\n", sub)
		return 2
	}
}

func runCodeSearchCmd(_ context.Context, _ *slog.Logger, args []string) int {
	fs := flag.NewFlagSet("code search", flag.ContinueOnError)
	mode := fs.String("mode", "definition", "搜索模式：definition（只搜符号定义）/ reference（只搜引用）/ all（全部）")
	language := fs.String("language", "", "语言过滤：go / ts / js / py / java / rs / c / cpp / sh / md")
	includeTests := fs.Bool("include-tests", false, "是否包含测试文件")
	limit := fs.Int("limit", 50, "返回结果上限，最大 200")
	workDir := fs.String("work-dir", "", "工作目录（默认取当前目录；在 quest 内自动取 quest 工作区）")
	jsonOut := fs.Bool("json", false, "JSON 结构化输出")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	query := strings.Join(fs.Args(), " ")
	if strings.TrimSpace(query) == "" {
		fmt.Fprintln(os.Stderr, "缺失搜索关键词")
		return 2
	}

	// 工作目录优先级：--work-dir > quest 上下文工作区 > 当前目录
	wd := *workDir
	if wd == "" {
		if _, _, _, q, code := loadAgentQuestQuiet(); code == 0 && q != nil && q.WorkspacePath != "" {
			wd = q.WorkspacePath
		} else {
			var err error
			wd, err = os.Getwd()
			if err != nil {
				return writeJSONError(1, "internal_error", "获取当前目录失败: "+err.Error())
			}
		}
	}
	wd, err := filepath.Abs(wd)
	if err != nil {
		return writeJSONError(1, "internal_error", "解析工作目录失败: "+err.Error())
	}

	results, backend, err := platformtools.RunCodeSearch(wd, query, *mode, *language, *includeTests, *limit)
	if err != nil {
		return writeJSONError(1, "search_failed", err.Error())
	}

	if *jsonOut {
		return writeJSONLine(map[string]any{
			"ok":      true,
			"query":   query,
			"mode":    *mode,
			"count":   len(results),
			"backend": backend,
			"results": results,
		})
	}

	// 人类可读输出
	fmt.Printf("找到 %d 条结果", len(results))
	if backend == "pure_go" {
		fmt.Print("（pure_go 模式，速度较慢；安装 ripgrep 可提升性能）")
	}
	fmt.Println()
	if len(results) == 0 {
		return 0
	}
	fmt.Println()

	for _, r := range results {
		symType := r.SymbolType
		if symType == "" || symType == "other" {
			symType = "—"
		}
		symName := r.SymbolName
		if symName == "" {
			symName = "—"
		}
		fmt.Printf("  %s:%d  [%s] %s\n", r.File, r.Line, symType, symName)
		fmt.Printf("    %s\n", r.Snippet)
		fmt.Println()
	}

	return 0
}

// loadAgentQuestQuiet 尝试加载 agent quest 上下文，失败时不打日志。
// 用于可选地使用 quest 上下文的命令（如 code search）。
func loadAgentQuestQuiet() (*fsstore.AgentContext, string, *fsstore.Root, *fsstore.QuestMeta, int) {
	discardLog := slog.New(slog.NewTextHandler(newNopWriter(), nil))
	return loadAgentQuest(discardLog)
}

type nopWriter struct{}

func (w *nopWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func newNopWriter() *nopWriter {
	return &nopWriter{}
}
