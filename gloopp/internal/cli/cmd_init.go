package cli

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
	"code.byted.org/lihuanyu.0w0/gloop/internal/version"
)

func runInit(_ context.Context, log *slog.Logger, args []string) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	dataDir := fs.String("data-dir", "", "数据目录（默认 ~/.gloop）")
	force := fs.Bool("force", false, "保留兼容参数；init 仅刷新缺失模板，不自动启用 agent")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	dd, err := resolveDataDir(*dataDir)
	if err != nil {
		log.Error(logPrefix+" 解析数据目录失败", "err", err)
		return 1
	}
	root, err := fsstore.Open(dd)
	if err != nil {
		log.Error(logPrefix+" Open 数据目录失败", "err", err)
		return 1
	}
	probe := detectAgents()
	log.Info(logPrefix+" 二进制探测结果",
		"traex", probe.TraeX, "relay", probe.Relay,
		"pi", probe.Pi, "codex", probe.Codex, "aiden", probe.Aiden, "hermes", probe.Hermes)

	wd := defaultWorkingDir()
	created, err := root.InitDefaultFiles(probe, wd)
	if err != nil {
		log.Error(logPrefix+" InitDefaultFiles 失败", "err", err)
		return 1
	}

	if *force {
		log.Info(logPrefix + " init --force 不再自动启用 agent；请显式启用已验证的 agent")
	}

	fmt.Printf("✅ %s 初始化完成\n", version.AppName)
	fmt.Printf("   数据目录: %s\n", root.Path())
	fmt.Printf("   工作目录: %s\n", wd)
	if len(created) > 0 {
		fmt.Printf("   新建/更新文件 (%d):\n", len(created))
		for _, c := range created {
			fmt.Printf("     - %s\n", c)
		}
	} else {
		fmt.Printf("   （所有默认文件均已存在，未修改）\n")
	}

	// Agent 探测汇总
	fmt.Println()
	fmt.Println("📋 Agent 探测汇总:")
	agentStatuses := getAgentStatusList(probe)
	for _, s := range agentStatuses {
		statusIcon := "✅"
		statusText := "已探测"
		extra := ""
		if !s.Detected {
			statusIcon = "💤"
			statusText = "未探测"
			extra = fmt.Sprintf("（可安装: %s）", s.InstallHint)
		}
		fmt.Printf("   %s %-10s %s %s\n", statusIcon, s.Name, statusText, extra)
		if s.Detected && s.Path != "" {
			fmt.Printf("      路径: %s\n", s.Path)
		}
	}

	// 统计
	detectedCount := 0
	for _, s := range agentStatuses {
		if s.Detected {
			detectedCount++
		}
	}
	fmt.Println()
	fmt.Printf("   已探测 %d / %d 个 agent\n", detectedCount, len(agentStatuses))
	if detectedCount == 0 {
		fmt.Println("   ⚠️  未探测到任何真实 agent；将仅保留 mock 兜底")
		fmt.Println("      可安装以下任一 agent 以获得最佳体验:")
		for _, s := range agentStatuses {
			if !s.Detected {
				fmt.Printf("        - %s\n", s.InstallHint)
			}
		}
	}

	fmt.Println()
	fmt.Println("💡 提示:")
	fmt.Println("   • 所有 agent 默认 disabled，请按需启用（编辑 agents/*.json 或用 config 命令）")
	fmt.Println("   • 本命令不会自动修改 PATH、不会自动 npm install")
	fmt.Println("   • 启用前建议自行验证 agent 可正常运行")

	return 0
}

// agentStatus 用于 init 输出展示
type agentStatus struct {
	Name        string
	Detected    bool
	Path        string
	InstallHint string
}

func getAgentStatusList(probe fsstore.AgentProbe) []agentStatus {
	return []agentStatus{
		{Name: "traex", Detected: probe.TraeX != "", Path: probe.TraeX, InstallHint: "TraeX CLI - npm i -g @bytedance-dev/traecli"},
		{Name: "relay", Detected: probe.Relay != "", Path: probe.Relay, InstallHint: "Relay CLI - npm i -g @bytedance-seed/claude-code"},
		{Name: "pi", Detected: probe.Pi != "", Path: probe.Pi, InstallHint: "Pi CLI - 字节内部 AI 编程助手"},
		{Name: "codex", Detected: probe.Codex != "", Path: probe.Codex, InstallHint: "Codex CLI - npm i -g @openai/codex"},
		{Name: "aiden_x_claude", Detected: probe.Aiden != "", Path: probe.Aiden, InstallHint: "Aiden X Claude - npm i -g @bytedance-dev/aiden"},
		{Name: "aiden_x_codex", Detected: probe.Aiden != "", Path: probe.Aiden, InstallHint: "Aiden X Codex - npm i -g @bytedance-dev/aiden"},
		{Name: "hermes", Detected: probe.Hermes != "", Path: probe.Hermes, InstallHint: "Hermes CLI - 待确认安装方式"},
	}
}

// binBaseName 返回二进制文件名（不带路径和扩展名）
func binBaseName(path string) string {
	if path == "" {
		return ""
	}
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// =========================================================================
// server
// =========================================================================
