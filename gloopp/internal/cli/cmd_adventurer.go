package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
)

func runAdventurerCmd(_ context.Context, log *slog.Logger, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "用法: gloop adventurer list|activate")
		return 2
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "list":
		fs := flag.NewFlagSet("adventurer list", flag.ContinueOnError)
		dataDir := fs.String("data-dir", "", "数据目录")
		jsonOut := fs.Bool("json", false, "JSON 输出")
		if err := fs.Parse(rest); err != nil {
			return 2
		}
		r := newRunner(*dataDir, log)
		if _, err := r.bootstrap(); err != nil {
			log.Error(logPrefix+" bootstrap 失败", "err", err)
			return 1
		}
		as, err := r.engine.ListAdventurers()
		if err != nil {
			log.Error(logPrefix+" 列冒险者失败", "err", err)
			return 1
		}
		if *jsonOut {
			if encErr := json.NewEncoder(os.Stdout).Encode(as); encErr != nil {
				fmt.Fprintf(os.Stderr, "[warn] encode JSON 输出失败: %v\n", encErr)
			}
			return 0
		}
		if len(as) == 0 {
			fmt.Println("（无冒险者，请先 `gloop init`）")
			return 0
		}
		fmt.Printf("%-24s  %-20s  %-10s  %-14s  %-6s  %s\n", "ID", "NAME", "CLASS", "STATUS", "Lv.", "TITLE")
		for _, a := range as {
			fmt.Printf("%-24s  %-20s  %-10s  %-14s  %-6d  %s\n",
				a.ID, a.Name, a.Class, a.Status, a.Level, a.GetTitle())
		}
		return 0

	case "show":
		fs := flag.NewFlagSet("adventurer show", flag.ContinueOnError)
		dataDir := fs.String("data-dir", "", "数据目录")
		if err := fs.Parse(rest); err != nil {
			return 2
		}
		if fs.NArg() == 0 {
			fmt.Fprintln(os.Stderr, "用法: gloop adventurer show <adv_id>")
			return 2
		}
		r := newRunner(*dataDir, log)
		if _, err := r.bootstrap(); err != nil {
			log.Error(logPrefix+" bootstrap 失败", "err", err)
			return 1
		}
		a, err := r.engine.GetAdventurer(fs.Arg(0))
		if err != nil {
			log.Error(logPrefix+" 获取冒险者失败", "err", err)
			return 1
		}
		b, marshalErr := json.MarshalIndent(a, "", "  ")
		if marshalErr != nil {
			fmt.Fprintf(os.Stderr, "[warn] marshal adventurer: %v\n", marshalErr)
		}
		fmt.Println(string(b))
		return 0

	case "activate":
		fs := flag.NewFlagSet("adventurer activate", flag.ContinueOnError)
		dataDir := fs.String("data-dir", "", "数据目录")
		name := fs.String("name", "", "修改名字")
		agent := fs.String("agent", "", "修改绑定的 agent agent")
		jsonOut := fs.Bool("json", false, "JSON 输出")
		if err := fs.Parse(rest); err != nil {
			return 2
		}
		args := fs.Args()
		if len(args) == 0 {
			fmt.Fprintln(os.Stderr, "用法: gloop adventurer activate <adv_id>")
			return 2
		}
		advID := args[0]

		r := newRunner(*dataDir, log)
		if _, err := r.bootstrap(); err != nil {
			log.Error(logPrefix+" bootstrap 失败", "err", err)
			return 1
		}

		updates := &fsstore.AdventurerUpdate{}
		if *name != "" {
			updates.Name = *name
		}
		if *agent != "" {
			updates.Agent = *agent
		}

		a, err := r.engine.ActivateAdventurer(advID, updates)
		if err != nil {
			log.Error(logPrefix+" 激活失败", "err", err)
			return 1
		}
		if *jsonOut {
			if encErr := json.NewEncoder(os.Stdout).Encode(a); encErr != nil {
				fmt.Fprintf(os.Stderr, "[warn] encode JSON 输出失败: %v\n", encErr)
			}
			return 0
		}
		fmt.Printf("冒险者已激活: %s (%s) Lv.%d %s\n", a.Name, a.Class, a.Level, a.GetTitle())
		return 0

	default:
		fmt.Fprintf(os.Stderr, "未知 adventurer 子命令 %q\n", sub)
		return 2
	}
}
