package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
	"code.byted.org/lihuanyu.0w0/gloop/internal/version"
)

func runConfigCmd(_ context.Context, log *slog.Logger, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "用法: gloop config <show|edit>")
		return 2
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "show":
		fs := flag.NewFlagSet("config show", flag.ContinueOnError)
		dataDir := fs.String("data-dir", "", "数据目录")
		if err := fs.Parse(rest); err != nil {
			return 2
		}
		dd, err := resolveDataDir(*dataDir)
		if err != nil {
			log.Error(logPrefix+" 解析数据目录失败", "err", err)
			return 1
		}
		root, err := fsstore.Open(dd)
		if err != nil {
			log.Error(logPrefix+" Open 失败", "err", err)
			return 1
		}
		cfg, err := root.LoadConfig()
		if err != nil {
			log.Error(logPrefix+" LoadConfig 失败", "err", err)
			return 1
		}
		agentsPath := root.Sub(fsstore.SubdirAgents)
		advPath := root.Sub(fsstore.SubdirAdventurers)
		tokenPath := filepath.Join(root.Path(), version.TokenFileName)
		fmt.Printf("PATH:   %s\n", root.Sub(fsstore.FileConfig))
		fmt.Printf("PROV:   %s\n", agentsPath)
		fmt.Printf("ADV:    %s\n", advPath)
		fmt.Printf("TOKEN:  %s\n", tokenPath)
		fmt.Println("--- JSON ---")
		b, marshalErr := json.MarshalIndent(cfg, "", "  ")
		if marshalErr != nil {
			fmt.Fprintf(os.Stderr, "[warn] write config: %v\n", marshalErr)
		}
		fmt.Println(string(b))
		return 0

	case "edit":
		fs := flag.NewFlagSet("config edit", flag.ContinueOnError)
		dataDir := fs.String("data-dir", "", "数据目录")
		if err := fs.Parse(rest); err != nil {
			return 2
		}
		dd, err := resolveDataDir(*dataDir)
		if err != nil {
			log.Error(logPrefix+" 解析数据目录失败", "err", err)
			return 1
		}
		root, err := fsstore.Open(dd)
		if err != nil {
			log.Error(logPrefix+" Open 失败", "err", err)
			return 1
		}
		// 先保证 config 存在
		if _, err := root.LoadConfig(); err != nil {
			log.Error(logPrefix+" LoadConfig 失败", "err", err)
			return 1
		}
		cfgPath := root.Sub(fsstore.FileConfig)
		editor := os.Getenv("EDITOR")
		if editor == "" {
			switch runtime.GOOS {
			case "windows":
				editor = "notepad"
			default:
				if _, err := exec.LookPath("vim"); err == nil {
					editor = "vim"
				} else if _, err := exec.LookPath("nano"); err == nil {
					editor = "nano"
				} else {
					fmt.Printf("（未找到编辑器，直接打开文件位置）\n%s\n", cfgPath)
					return 0
				}
			}
		}
		fmt.Printf("打开 %s  (%s)\n", cfgPath, editor)
		cmd := exec.Command(editor, cfgPath)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			log.Error(logPrefix+" 打开编辑器失败", "err", err)
			return 1
		}
		return 0

	default:
		fmt.Fprintf(os.Stderr, "未知 config 子命令 %q\n", sub)
		return 2
	}
}

// =========================================================================
// misc
// =========================================================================
