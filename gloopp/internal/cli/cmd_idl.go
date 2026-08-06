package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"code.byted.org/lihuanyu.0w0/gloop/internal/idl"
)

func runIDLCmd(_ context.Context, log *slog.Logger, args []string) int {
	_ = log
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "用法: gloop idl <export>")
		return 2
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "export":
		return runIDLExport(rest)
	default:
		fmt.Fprintf(os.Stderr, "未知子命令: idl %s\n", sub)
		fmt.Fprintln(os.Stderr, "用法: gloop idl <export>")
		return 2
	}
}

func runIDLExport(args []string) int {
	fs := flag.NewFlagSet("idl export", flag.ContinueOnError)
	only := fs.String("only", "all", "导出范围：all | skills | classes")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	bundle := idl.BuildBundle()
	var out any
	switch *only {
	case "all", "":
		out = bundle
	case "skills":
		out = bundle.Skills
	case "classes":
		out = bundle.Classes
	default:
		fmt.Fprintf(os.Stderr, "未知导出范围: %s（可选 all / skills / classes）\n", *only)
		return 2
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		fmt.Fprintf(os.Stderr, "导出 IDL 失败: %v\n", err)
		return 1
	}
	return 0
}
