package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"code.byted.org/lihuanyu.0w0/gloop/internal/prompt"
)

// runPromptCmd 是 `gloop prompt` 子命令入口。
//
// 设计原则：和 skill 子命令风格对齐。
// - `gloop prompt list` 列出所有模板 + 来源 + 预览
// - `gloop prompt show <name>` 查看单个模板完整内容
// - 用户可以在 ~/.gloop/prompts/ 下覆盖模板，list/show 看到的是实际生效版本
func runPromptCmd(_ context.Context, log *slog.Logger, args []string) int {
	_ = log
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "用法: gloop prompt <list|show>")
		return 2
	}
	sub := args[0]
	rest := args[1:]

	switch sub {
	case "list":
		return runPromptList(rest)
	case "show":
		return runPromptShow(rest)
	default:
		fmt.Fprintf(os.Stderr, "未知子命令: prompt %s\n", sub)
		fmt.Fprintln(os.Stderr, "用法: gloop prompt <list|show>")
		return 2
	}
}

func runPromptList(args []string) int {
	fs := flag.NewFlagSet("prompt list", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "JSON 输出")
	sourceFilter := fs.String("source", "", "只看指定来源（builtin / user），默认全部")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	store := prompt.DefaultTemplateStore()
	list, err := store.ListTemplates()
	if err != nil {
		fmt.Fprintf(os.Stderr, "列出模板失败: %v\n", err)
		return 1
	}

	// 来源过滤
	if *sourceFilter != "" {
		filtered := list[:0]
		for _, t := range list {
			if t.Source == *sourceFilter {
				filtered = append(filtered, t)
			}
		}
		list = filtered
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		for _, t := range list {
			_ = enc.Encode(t)
		}
		return 0
	}

	if len(list) == 0 {
		fmt.Println("（无可用 prompt 模板）")
		return 0
	}

	fmt.Printf("Prompt 模板（%d 个）：\n\n", len(list))
	fmt.Printf("  %-35s  %-8s  %8s  %s\n", "NAME", "SOURCE", "SIZE", "PREVIEW")
	for _, t := range list {
		src := t.Source
		preview := truncateUTF8(t.Preview, 50)
		fmt.Printf("  %-35s  %-8s  %7d  %s\n", t.Name, src, t.SizeBytes, preview)
	}
	fmt.Println()
	fmt.Println("运行 `gloop prompt show <name>` 查看完整模板内容。")
	fmt.Println("用户目录覆盖模板放在 ~/.gloop/prompts/ 下，重启生效。")
	return 0
}

func runPromptShow(args []string) int {
	fs := flag.NewFlagSet("prompt show", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "用法: gloop prompt show <name>")
		return 2
	}
	name := fs.Arg(0)

	store := prompt.DefaultTemplateStore()
	content, info, err := store.GetTemplate(name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "模板不存在: %s\n", name)
		// 列出可用模板名
		list, lerr := store.ListTemplates()
		if lerr == nil && len(list) > 0 {
			names := make([]string, 0, len(list))
			for _, t := range list {
				names = append(names, t.Name)
			}
			fmt.Fprintf(os.Stderr, "可用模板: %s\n", joinStr(names, ", "))
		}
		return 1
	}

	fmt.Printf("# %s\n", name)
	fmt.Printf("> 来源: %s · %d 字节\n\n", info.Source, info.SizeBytes)
	fmt.Print(content)
	return 0
}
