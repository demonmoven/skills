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
	"sort"
	"strings"
	"time"

	"code.byted.org/lihuanyu.0w0/gloop/internal/acp"
	"code.byted.org/lihuanyu.0w0/gloop/internal/events"
	"code.byted.org/lihuanyu.0w0/gloop/internal/executor"
	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
	"code.byted.org/lihuanyu.0w0/gloop/internal/notifications"
	"code.byted.org/lihuanyu.0w0/gloop/internal/orchestrator"
)

const doctorSmokePrompt = "Reply exactly: GLOOP_SMOKE_OK. Do not edit files."

type doctorAgentsReport struct {
	DataDir string              `json:"data_dir"`
	Summary doctorAgentSummary  `json:"summary"`
	Agents  []doctorAgentResult `json:"agents"`
}

type doctorAgentSummary struct {
	Total    int `json:"total"`
	OK       int `json:"ok"`
	Warning  int `json:"warning"`
	Error    int `json:"error"`
	Disabled int `json:"disabled"`
}

type doctorAgentResult struct {
	Name            string             `json:"name"`
	Type            string             `json:"type"`
	Enabled         bool               `json:"enabled"`
	Command         string             `json:"command"`
	Args            []string           `json:"args,omitempty"`
	ResolvedCommand string             `json:"resolved_command,omitempty"`
	Adapter         string             `json:"adapter,omitempty"`
	Status          string             `json:"status"`
	Version         string             `json:"version,omitempty"`
	Capabilities    []string           `json:"capabilities,omitempty"`
	CapabilityTier  string             `json:"capability_tier,omitempty"`
	Issues          []string           `json:"issues,omitempty"`
	Smoke           *doctorSmokeResult `json:"smoke,omitempty"`
}

type doctorSmokeResult struct {
	Status      string   `json:"status"`
	DurationMs  int64    `json:"duration_ms,omitempty"`
	Args        []string `json:"args,omitempty"`
	Output      string   `json:"output,omitempty"`
	Error       string   `json:"error,omitempty"`
	FailureKind string   `json:"failure_kind,omitempty"`
	Hint        string   `json:"hint,omitempty"`
}

func (r *doctorSmokeResult) classify() {
	if r == nil || r.Status == "ok" || r.Status == "skipped" {
		return
	}
	kind, hint := classifyDoctorSmokeError(strings.Join([]string{r.Error, r.Output}, " "))
	r.FailureKind = kind
	r.Hint = hint
}

func classifyDoctorSmokeError(msg string) (kind, hint string) {
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "anthropic authentication is not supported") ||
		strings.Contains(lower, "unset anthropic_auth_token"):
		return "auth_env_conflict", "当前环境变量 ANTHROPIC_AUTH_TOKEN 与该 agent/网关不兼容；先 unset ANTHROPIC_AUTH_TOKEN 后重试，或重新执行对应 auth login。"
	case strings.Contains(lower, "authentication required") ||
		strings.Contains(lower, "not authenticated") ||
		strings.Contains(lower, "login required") ||
		strings.Contains(lower, "please login"):
		return "auth_required", "agent 认证未完成；请先执行对应 CLI 登录命令，例如 relay auth login 或 claude 登录/认证流程。"
	case strings.Contains(lower, "access denied for model") ||
		strings.Contains(lower, "model access denied") ||
		strings.Contains(lower, "rejected this resolved model") ||
		strings.Contains(lower, "permission") && strings.Contains(lower, "model"):
		return "model_access_denied", "当前账号或网关没有该模型权限；请检查 agent default_model、adventurer model 覆盖和模型 entitlement。"
	case strings.Contains(lower, "timeout") || strings.Contains(lower, "超时") || strings.Contains(lower, "deadline exceeded"):
		return "timeout", "smoke 超时；请检查 agent 是否能启动、网络是否可用，必要时增大 --smoke-timeout。"
	case strings.Contains(lower, "enoent") || strings.Contains(lower, "no such file or directory"):
		return "runtime_filesystem_error", "agent 启动后访问的本地文件或目录不存在；请检查该 CLI 的本地配置目录、日志目录和工作目录权限。"
	case strings.Contains(lower, "command 不存在") || strings.Contains(lower, "executable file not found"):
		return "command_not_found", "agent 命令不存在或不可执行；请安装对应 CLI 或修正 agents/*.json 的 command。"
	default:
		return "unknown", "未能自动归类；请查看 error/output 中的原始错误。"
	}
}

func runDoctorCmd(ctx context.Context, log *slog.Logger, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "用法: gloop doctor <agents|notifications|e2e>")
		return 2
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "agents":
		return runDoctorAgents(ctx, log, rest)
	case "notifications":
		return runDoctorNotifications(ctx, log, rest)
	case "e2e":
		return runDoctorE2E(ctx, log, rest)
	default:
		fmt.Fprintf(os.Stderr, "未知 doctor 子命令 %q\n", sub)
		return 2
	}
}

func runDoctorAgents(ctx context.Context, log *slog.Logger, args []string) int {
	fs := flag.NewFlagSet("doctor agents", flag.ContinueOnError)
	dataDir := fs.String("data-dir", "", "数据目录")
	jsonOut := fs.Bool("json", false, "输出 JSON")
	all := fs.Bool("all", true, "包含禁用 agent")
	smoke := fs.Bool("smoke", false, "对启用且支持的 agent 运行最小真实调用")
	smokeTimeout := fs.Duration("smoke-timeout", 30*time.Second, "smoke 调用超时时间")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
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
	agents, err := root.ListAgents()
	if err != nil {
		log.Error(logPrefix+" 读取 agents 失败", "err", err)
		return 1
	}
	sort.Slice(agents, func(i, j int) bool { return agents[i].Name < agents[j].Name })

	report := doctorAgentsReport{DataDir: root.Path(), Agents: []doctorAgentResult{}}
	for _, agent := range agents {
		if agent == nil {
			continue
		}
		if !*all && !agent.Enabled {
			continue
		}
		result := diagnoseAgent(ctx, agent, *smoke, *smokeTimeout)
		report.Agents = append(report.Agents, result)
		switch result.Status {
		case "ok":
			report.Summary.OK++
		case "warning":
			report.Summary.Warning++
		case "error":
			report.Summary.Error++
		case "disabled":
			report.Summary.Disabled++
		}
	}
	report.Summary.Total = len(report.Agents)

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "写入 JSON 失败: %v\n", err)
			return 1
		}
	} else {
		printDoctorAgentsReport(report)
	}
	if report.Summary.Error > 0 {
		return 1
	}
	return 0
}

// runDoctorNotifications 检查通知渠道可用性。
func runDoctorNotifications(ctx context.Context, log *slog.Logger, args []string) int {
	fs := flag.NewFlagSet("doctor notifications", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "输出 JSON")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}

	notifier := notifications.NewLarkCliNotifier()
	available := notifier.Available()
	openID, name := notifier.Recipient()

	type notifyReport struct {
		Available bool   `json:"available"`
		Channel   string `json:"channel"`
		Recipient string `json:"recipient,omitempty"`
		OpenID    string `json:"open_id,omitempty"`
		Reason    string `json:"reason,omitempty"`
	}

	report := notifyReport{
		Available: available,
		Channel:   "lark-cli",
		Recipient: name,
		OpenID:    openID,
	}
	if !available {
		report.Reason = notifier.StatusError()
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "写入 JSON 失败: %v\n", err)
			return 1
		}
	} else {
		fmt.Println("Gloop notifications doctor")
		fmt.Printf("Channel:   %s\n", report.Channel)
		if available {
			fmt.Printf("Status:    ok\n")
			fmt.Printf("Recipient: %s (%s)\n", name, openID)
		} else {
			fmt.Printf("Status:    unavailable\n")
			fmt.Printf("Reason:    %s\n", notifier.StatusError())
			fmt.Println()
			fmt.Println("启用飞书通知：")
			fmt.Println("  1. 安装 lark-cli: npm install -g @larksuite/cli")
			fmt.Println("  2. 登录飞书:   lark-cli auth login")
		}
	}

	if !available {
		return 1
	}
	return 0
}

func diagnoseAgent(ctx context.Context, agent *fsstore.AgentConfig, smoke bool, smokeTimeout time.Duration) doctorAgentResult {
	result := doctorAgentResult{
		Name:    agent.Name,
		Type:    string(agent.Type),
		Enabled: agent.Enabled,
		Command: agent.Command,
		Args:    append([]string(nil), agent.Args...),
		Status:  "ok",
		Adapter: detectAgentAdapter(agent),
		Issues:  []string{},
		Version: "",
	}
	result.CapabilityTier = doctorCapabilityTier(agent)
	if !agent.Enabled {
		result.Status = "disabled"
		result.Capabilities = staticAgentCapabilities(agent)
		return result
	}
	if strings.TrimSpace(agent.Command) == "" {
		result.Status = "error"
		result.Issues = append(result.Issues, "command 为空")
		return result
	}
	resolved, ok, err := resolveDoctorCommand(agent.Command)
	if err != nil {
		result.Status = "error"
		result.Issues = append(result.Issues, err.Error())
		return result
	}
	result.ResolvedCommand = resolved
	if !ok {
		result.Status = "error"
		result.Issues = append(result.Issues, "command 不存在或不可执行")
		return result
	}

	if issue := adapterIssue(agent); issue != "" {
		result.Status = "error"
		result.Issues = append(result.Issues, issue)
		return result
	}
	result.Capabilities = staticAgentCapabilities(agent)
	if version, err := readAgentVersion(ctx, resolved); err == nil {
		result.Version = version
	} else {
		result.Status = "warning"
		result.Issues = append(result.Issues, err.Error())
	}
	if smoke {
		smokeResult := runAgentSmoke(ctx, agent, result.Adapter, resolved, smokeTimeout)
		smokeResult.classify()
		result.Smoke = &smokeResult
		switch smokeResult.Status {
		case "ok":
			// Keep a version warning visible but do not hide a passing smoke check.
		case "skipped":
			if result.Status == "ok" {
				result.Status = "warning"
			}
			if smokeResult.Error != "" {
				result.Issues = append(result.Issues, smokeResult.Error)
			}
		default:
			result.Status = "error"
			if smokeResult.Error != "" {
				result.Issues = append(result.Issues, smokeResult.Error)
			}
		}
	}
	return result
}

func resolveDoctorCommand(command string) (string, bool, error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return "", false, nil
	}
	if filepath.IsAbs(command) || strings.Contains(command, "/") {
		expanded, err := expandTilde(command)
		if err != nil {
			return "", false, err
		}
		st, err := os.Stat(expanded)
		if err != nil {
			return expanded, false, nil
		}
		if st.IsDir() {
			return expanded, false, fmt.Errorf("command 指向目录")
		}
		if runtime.GOOS != "windows" && st.Mode()&0o111 == 0 {
			return expanded, false, nil
		}
		return expanded, true, nil
	}
	resolved, err := exec.LookPath(command)
	if err != nil {
		return command, false, nil
	}
	return resolved, true, nil
}

func detectAgentAdapter(agent *fsstore.AgentConfig) string {
	if agent == nil {
		return ""
	}
	if agent.Type == model.AgentTypeACP {
		return "acp"
	}
	if agent.Type != model.AgentTypeCLI {
		return "unknown"
	}
	base := strings.TrimSuffix(filepath.Base(agent.Command), filepath.Ext(agent.Command))
	switch base {
	case "relay", "claude":
		return "relay-cli"
	case "codex":
		return "codex-cli"
	case "pi":
		return "pi-cli"
	case "aiden":
		switch detectDoctorAidenVariant(agent.Name, agent.Args) {
		case "x_claude":
			return "aiden-x-claude-cli"
		case "x_codex":
			return "aiden-x-codex-cli"
		default:
			return "aiden-cli"
		}
	case "traex":
		return "traex-cli"
	default:
		return "unsupported-cli"
	}
}

func detectDoctorAidenVariant(name string, args []string) string {
	if strings.Contains(name, "x_claude") || strings.Contains(name, "xclaude") {
		return "x_claude"
	}
	if strings.Contains(name, "x_codex") || strings.Contains(name, "xcodex") {
		return "x_codex"
	}
	for i := 0; i+1 < len(args); i++ {
		if args[i] != "x" {
			continue
		}
		switch args[i+1] {
		case "claude":
			return "x_claude"
		case "codex":
			return "x_codex"
		}
	}
	return "base"
}

func doctorCapabilityTier(agent *fsstore.AgentConfig) string {
	switch detectAgentAdapter(agent) {
	case "acp", "relay-cli", "codex-cli", "traex-cli", "aiden-x-claude-cli", "aiden-x-codex-cli":
		return "tier_b"
	case "pi-cli", "aiden-cli":
		return "tier_c"
	default:
		if agent != nil && agent.Type == model.AgentTypeMock {
			return "test_only"
		}
		return "tier_c"
	}
}

func adapterIssue(agent *fsstore.AgentConfig) string {
	if agent == nil {
		return "agent config 为空"
	}
	switch agent.Type {
	case model.AgentTypeACP:
		return ""
	case model.AgentTypeCLI:
		base := strings.TrimSuffix(filepath.Base(agent.Command), filepath.Ext(agent.Command))
		switch base {
		case "relay", "claude", "codex", "traex", "pi", "aiden":
			return ""
		default:
			return "CLI agent 暂无专用适配器"
		}
	default:
		return fmt.Sprintf("未知 agent 类型 %q", agent.Type)
	}
}

func staticAgentCapabilities(agent *fsstore.AgentConfig) []string {
	if agent == nil {
		return nil
	}
	switch detectAgentAdapter(agent) {
	case "acp":
		return []string{"acp", "streaming", "tool-use", "context-export"}
	case "traex-cli":
		return []string{"cli", "non-interactive", "jsonl", "stateful-resume", "experimental"}
	case "relay-cli", "codex-cli", "pi-cli", "aiden-cli", "aiden-x-claude-cli", "aiden-x-codex-cli":
		return []string{"cli", "non-interactive", "streaming", "working-dir"}
	default:
		return nil
	}
}

func readAgentVersion(ctx context.Context, command string) (string, error) {
	vctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(vctx, command, "--version")
	out, err := cmd.CombinedOutput()
	if vctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("--version 超时")
	}
	if err != nil {
		return "", fmt.Errorf("--version 失败: %v", err)
	}
	version := strings.TrimSpace(string(out))
	version = strings.Join(strings.Fields(version), " ")
	if version == "" {
		return "", fmt.Errorf("--version 输出为空")
	}
	if len(version) > 160 {
		version = version[:160]
	}
	return version, nil
}

func runAgentSmoke(ctx context.Context, agent *fsstore.AgentConfig, adapter, command string, timeout time.Duration) doctorSmokeResult {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	if adapter == "acp" {
		return runACPSmoke(ctx, agent, command, timeout)
	}
	args, stdin, ok := smokeCommand(adapter, agent)
	if !ok {
		return doctorSmokeResult{Status: "skipped", Error: "smoke 暂不支持该 agent 适配器"}
	}
	workDir, err := os.MkdirTemp("", "gloop-doctor-smoke-*")
	if err != nil {
		return doctorSmokeResult{Status: "error", Error: "创建 smoke 临时目录失败: " + err.Error()}
	}
	defer os.RemoveAll(workDir)

	smokeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	start := time.Now()
	cmd := exec.CommandContext(smokeCtx, command, args...)
	cmd.Dir = workDir
	if agentEnv, _ := agent.EffectiveEnv(); len(agentEnv) > 0 {
		cmd.Env = append(cmd.Environ(), envToPairsDoctor(agentEnv)...)
	}
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin + "\n")
	}
	out, err := cmd.CombinedOutput()
	duration := time.Since(start).Milliseconds()
	output := truncateDoctorText(strings.Join(strings.Fields(string(out)), " "), 300)
	result := doctorSmokeResult{
		Status:     "ok",
		DurationMs: duration,
		Args:       redactSmokeArgs(args),
		Output:     output,
	}
	if smokeCtx.Err() == context.DeadlineExceeded {
		result.Status = "error"
		result.Error = "smoke 超时"
		result.classify()
		return result
	}
	if err != nil {
		// relay-cli / claude: stream-json 协议中 exit code 1 可能是 hook 副作用，
		// 真实结果看 stdout 里的 {"type":"result","subtype":"success"} 判断。
		if adapter == "relay-cli" && isStreamJSONSuccess(string(out)) {
			// 忽略 exit code，stream-json 报告 success
		} else {
			result.Status = "error"
			result.Error = "smoke 失败: " + err.Error()
			if strings.TrimSpace(string(out)) != "" {
				result.Error = appendDoctorSmokeDetail(result.Error, string(out))
			}
			result.classify()
			return result
		}
	}
	if strings.TrimSpace(output) == "" {
		result.Status = "warning"
		result.Error = "smoke 输出为空"
		result.classify()
	}
	return result
}

func runACPSmoke(ctx context.Context, agent *fsstore.AgentConfig, command string, timeout time.Duration) doctorSmokeResult {
	workDir, err := os.MkdirTemp("", "gloop-doctor-acp-smoke-*")
	if err != nil {
		return doctorSmokeResult{Status: "error", Error: "创建 smoke 临时目录失败: " + err.Error()}
	}
	defer os.RemoveAll(workDir)

	smokeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()
	env, envErr := agent.EffectiveEnv()
	if envErr != nil {
		return doctorSmokeResult{
			Status:     "error",
			DurationMs: time.Since(start).Milliseconds(),
			Args:       append([]string(nil), agent.Args...),
			Error:      "读取 agent env 失败: " + envErr.Error(),
		}
	}
	client, err := acp.NewClient(smokeCtx, command, append([]string(nil), agent.Args...), env)
	if err != nil {
		return doctorSmokeResult{
			Status:     "error",
			DurationMs: time.Since(start).Milliseconds(),
			Args:       append([]string(nil), agent.Args...),
			Error:      "ACP 启动失败: " + err.Error(),
		}
	}
	defer client.Close()

	output := strings.Builder{}
	var firstErr error
	modelChoice := doctorModelChoice(agent)
	if _, err = client.Initialize(smokeCtx, "gloop-doctor", "0.1"); err != nil {
		firstErr = fmt.Errorf("ACP initialize 失败: %w", err)
	} else {
		sess, sessErr := client.NewSession(smokeCtx, workDir, []any{}, "", modelChoice)
		if sessErr != nil {
			firstErr = fmt.Errorf("ACP session/new 失败: %w", sessErr)
		} else {
			client.SetSessionUpdateHandler(sess.SessionID, func(update *acp.SessionUpdate) {
				if update == nil || update.Update.Content == nil {
					return
				}
				switch update.Update.SessionUpdate {
				case "agent_message_chunk", "agent_thought_chunk":
					if update.Update.Content.Type == "text" {
						output.WriteString(update.Update.Content.Text)
					}
				}
			})
			_, firstErr = client.Prompt(smokeCtx, sess.SessionID, []acp.ContentPart{{Type: "text", Text: doctorSmokePrompt}}, modelChoice)
			client.ClearSessionUpdateHandler(sess.SessionID)
			if firstErr != nil {
				firstErr = fmt.Errorf("ACP session/prompt 失败: %w", firstErr)
			}
		}
	}

	duration := time.Since(start).Milliseconds()
	outputText := truncateDoctorText(strings.Join(strings.Fields(output.String()), " "), 300)
	result := doctorSmokeResult{
		Status:     "ok",
		DurationMs: duration,
		Args:       append([]string(nil), agent.Args...),
		Output:     outputText,
	}
	if smokeCtx.Err() == context.DeadlineExceeded {
		result.Status = "error"
		result.Error = appendDoctorSmokeDetail("ACP smoke 超时", client.StderrSnapshot())
		result.classify()
		return result
	}
	if firstErr != nil {
		result.Status = "error"
		result.Error = appendDoctorSmokeDetail(firstErr.Error(), client.StderrSnapshot())
		result.classify()
		return result
	}
	if strings.TrimSpace(outputText) == "" {
		result.Status = "warning"
		result.Error = appendDoctorSmokeDetail("ACP smoke 输出为空", client.StderrSnapshot())
		result.classify()
		return result
	}
	if !strings.Contains(output.String(), "GLOOP_SMOKE_OK") {
		result.Status = "warning"
		result.Error = appendDoctorSmokeDetail("ACP smoke 未看到 GLOOP_SMOKE_OK", client.StderrSnapshot())
		result.classify()
	}
	return result
}

func smokeCommand(adapter string, agent *fsstore.AgentConfig) (args []string, stdin string, ok bool) {
	modelChoice := doctorModelChoice(agent)
	switch adapter {
	case "relay-cli":
		args = []string{
			"-p",
			"--verbose",
			"--output-format=stream-json",
			"--permission-mode", "bypassPermissions",
			"--no-session-persistence",
			"--session-id", "00000000-0000-4000-8000-000000000001",
			"--system-prompt", "You are running a Gloop agent smoke check.",
		}
		if modelChoice != "" {
			args = append(args, "--model", modelChoice)
		}
		return args, doctorSmokePrompt, true
	case "codex-cli":
		args = []string{
			"exec",
			"--ephemeral",
			"--skip-git-repo-check",
			"--json",
			"-s", "read-only",
		}
		if modelChoice != "" {
			args = append(args, "--model", modelChoice)
		}
		args = append(args, doctorSmokePrompt)
		return args, "", true
	case "traex-cli":
		args = []string{
			"exec",
			"--ephemeral",
			"--skip-git-repo-check",
			"--json",
			"--sandbox", "read-only",
		}
		if modelChoice != "" {
			args = append(args, "--model", modelChoice)
		}
		return args, doctorSmokePrompt, true
	case "pi-cli":
		args = []string{
			"--mode", "json",
			"--print",
			"--no-session",
			"--no-tools",
			"--system-prompt", "You are running a Gloop agent smoke check.",
		}
		if modelChoice != "" {
			args = append(args, "--model", modelChoice)
		}
		args = append(args, doctorSmokePrompt)
		return args, "", true
	case "aiden-cli":
		args = []string{
			"--one-shot",
			"--stream-json",
			"--allowedTools", "Read,Glob,Grep",
			"--system-prompt", "You are running a Gloop agent smoke check.",
		}
		if modelChoice != "" {
			args = append(args, "--model", modelChoice)
		}
		args = append(args, doctorSmokePrompt)
		return args, "", true
	case "aiden-x-claude-cli":
		args = []string{"x", "claude", "--stream-json"}
		if modelChoice != "" {
			args = append(args, "--model", modelChoice)
		}
		args = append(args, "--print", doctorSmokePrompt)
		return args, "", true
	case "aiden-x-codex-cli":
		args = []string{"x", "codex", "--stream-json"}
		if modelChoice != "" {
			args = append(args, "--model", modelChoice)
		}
		args = append(args, "exec", doctorSmokePrompt)
		return args, "", true
	default:
		return nil, "", false
	}
}

func doctorModelChoice(agent *fsstore.AgentConfig) string {
	if agent == nil {
		return ""
	}
	if agent.DefaultModel != "" && agent.DefaultModel != "auto" && agent.DefaultModel != "default" {
		return agent.DefaultModel
	}
	return ""
}

func envToPairsDoctor(env map[string]string) []string {
	out := make([]string, 0, len(env))
	for k, v := range env {
		out = append(out, k+"="+v)
	}
	return out
}

func isStreamJSONSuccess(output string) bool {
	return strings.Contains(output, `"subtype":"success"`) && strings.Contains(output, `"is_error":false`)
}

func appendDoctorSmokeDetail(msg, stderr string) string {
	stderr = strings.Join(strings.Fields(stderr), " ")
	if stderr == "" {
		return msg
	}
	return msg + " | stderr: " + truncateDoctorText(stderr, 500)
}

func redactSmokeArgs(args []string) []string {
	out := append([]string(nil), args...)
	for i := 0; i < len(out); i++ {
		if out[i] == "--session-id" && i+1 < len(out) {
			out[i+1] = "<generated>"
		}
	}
	return out
}

func truncateDoctorText(s string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "..."
}

func printDoctorAgentsReport(report doctorAgentsReport) {
	fmt.Printf("Gloop agent doctor\n")
	fmt.Printf("Data dir: %s\n", report.DataDir)
	fmt.Printf("Summary: total=%d ok=%d warning=%d error=%d disabled=%d\n\n",
		report.Summary.Total, report.Summary.OK, report.Summary.Warning, report.Summary.Error, report.Summary.Disabled)
	if len(report.Agents) == 0 {
		fmt.Println("No agents configured.")
		return
	}
	fmt.Printf("%-22s %-8s %-9s %-8s %-22s %s\n", "NAME", "TYPE", "STATUS", "TIER", "ADAPTER", "COMMAND")
	for _, agent := range report.Agents {
		command := agent.ResolvedCommand
		if command == "" {
			command = agent.Command
		}
		fmt.Printf("%-22s %-8s %-9s %-8s %-22s %s\n", agent.Name, agent.Type, agent.Status, agent.CapabilityTier, agent.Adapter, command)
		if agent.Version != "" {
			fmt.Printf("  version: %s\n", agent.Version)
		}
		if agent.Smoke != nil {
			fmt.Printf("  smoke: %s", agent.Smoke.Status)
			if agent.Smoke.DurationMs > 0 {
				fmt.Printf(" (%dms)", agent.Smoke.DurationMs)
			}
			if agent.Smoke.Error != "" {
				fmt.Printf(" - %s", agent.Smoke.Error)
			}
			fmt.Println()
			if agent.Smoke.FailureKind != "" {
				fmt.Printf("  smoke kind: %s\n", agent.Smoke.FailureKind)
			}
			if agent.Smoke.Hint != "" {
				fmt.Printf("  smoke hint: %s\n", agent.Smoke.Hint)
			}
		}
		for _, issue := range agent.Issues {
			fmt.Printf("  issue: %s\n", issue)
		}
	}
}

type doctorE2EReport struct {
	Status     string                 `json:"status"`
	DataDir    string                 `json:"data_dir"`
	WorkDir    string                 `json:"work_dir"`
	QuestID    string                 `json:"quest_id,omitempty"`
	DurationMs int64                  `json:"duration_ms"`
	Steps      []doctorE2EStep        `json:"steps"`
	Diff       *doctorE2EDiff         `json:"diff,omitempty"`
	Contracts  map[string]bool        `json:"contracts,omitempty"`
	Warnings   []string               `json:"warnings,omitempty"`
	Error      string                 `json:"error,omitempty"`
	Meta       map[string]string      `json:"meta,omitempty"`
	Details    map[string]interface{} `json:"details,omitempty"`
}

type doctorE2EStep struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	DurationMs int64  `json:"duration_ms,omitempty"`
	Detail     string `json:"detail,omitempty"`
}

type doctorE2EDiff struct {
	ChangedFiles int    `json:"changed_files"`
	Additions    int    `json:"additions"`
	Deletions    int    `json:"deletions"`
	Stat         string `json:"stat,omitempty"`
}

func runDoctorE2E(ctx context.Context, log *slog.Logger, args []string) int {
	fs := flag.NewFlagSet("doctor e2e", flag.ContinueOnError)
	dataDir := fs.String("data-dir", "", "数据目录；为空时创建临时目录")
	workDir := fs.String("work-dir", "", "工作目录；为空时创建临时项目")
	jsonOut := fs.Bool("json", false, "输出 JSON")
	timeout := fs.Duration("timeout", 20*time.Second, "E2E 超时时间")
	keep := fs.Bool("keep", false, "保留临时数据目录和工作目录")
	agentName := fs.String("agent", "", "指定真实 agent 跑 quest 合同诊断；为空时使用 mock E2E")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	var report doctorE2EReport
	var err error
	if strings.TrimSpace(*agentName) == "" {
		report, err = runMockE2EBenchmark(ctx, log, *dataDir, *workDir, *timeout, *keep)
	} else {
		report, err = runRealAgentE2EContract(ctx, log, *dataDir, *workDir, strings.TrimSpace(*agentName), *timeout, *keep)
	}
	if err != nil && report.Error == "" {
		report.Status = "error"
		report.Error = err.Error()
	}
	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if encErr := enc.Encode(report); encErr != nil {
			fmt.Fprintf(os.Stderr, "写入 JSON 失败: %v\n", encErr)
			return 1
		}
	} else {
		printDoctorE2EReport(report)
	}
	if err != nil || report.Status != "ok" {
		return 1
	}
	return 0
}

func runMockE2EBenchmark(ctx context.Context, log *slog.Logger, dataDir, workDir string, timeout time.Duration, keep bool) (doctorE2EReport, error) {
	start := time.Now()
	report := doctorE2EReport{
		Status: "ok",
		Steps:  []doctorE2EStep{},
		Meta: map[string]string{
			"executor": "mock",
			"scope":    "engine-workspace-review-apply",
		},
	}
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var cleanup []string
	defer func() {
		report.DurationMs = time.Since(start).Milliseconds()
		if keep {
			return
		}
		for _, p := range cleanup {
			_ = os.RemoveAll(p)
		}
	}()

	if dataDir == "" {
		dir, err := os.MkdirTemp("", "gloop-doctor-e2e-data-*")
		if err != nil {
			return failE2E(report, "prepare_data_dir", err)
		}
		dataDir = dir
		cleanup = append(cleanup, dir)
	}
	if workDir == "" {
		dir, err := os.MkdirTemp("", "gloop-doctor-e2e-work-*")
		if err != nil {
			return failE2E(report, "prepare_work_dir", err)
		}
		workDir = dir
		cleanup = append(cleanup, dir)
	}
	report.DataDir = dataDir
	report.WorkDir = workDir

	stepStart := time.Now()
	root, err := fsstore.Open(dataDir)
	if err != nil {
		return failE2E(report, "open_root", err)
	}
	appendE2EStep(&report, "open_root", "ok", stepStart, dataDir)

	stepStart = time.Now()
	if err := os.WriteFile(filepath.Join(workDir, "README.md"), []byte("# Gloop E2E\n\nbaseline\n"), 0o644); err != nil {
		return failE2E(report, "prepare_project", err)
	}
	_, err = root.InitDefaultFiles(fsstore.AgentProbe{}, workDir)
	if err != nil {
		return failE2E(report, "init_defaults", err)
	}
	if err := configureMockAdventurers(root); err != nil {
		return failE2E(report, "configure_mock_adventurers", err)
	}
	appendE2EStep(&report, "prepare_project", "ok", stepStart, workDir)

	stepStart = time.Now()
	cfg, err := root.LoadConfig()
	if err != nil {
		return failE2E(report, "load_config", err)
	}
	cfg.MaxTurnsPerPhase = 3
	cfg.MaxReworkPerQuest = 1
	cfg.DefaultWorkingDir = workDir
	cfg.DefaultWorkspaceMode = string(model.WorkspaceCopy)
	// doctor 验证显式 user_review→approve 全链路，关闭 HOTL 自主闭环
	hotlAutoClose := false
	cfg.HotlAutoClose = &hotlAutoClose
	if err := root.SaveConfig(cfg); err != nil {
		return failE2E(report, "save_config", err)
	}
	bus := events.NewBus()
	eng, err := orchestrator.NewEngine(root, cfg, bus, workDir, log)
	if err != nil {
		return failE2E(report, "new_engine", err)
	}
	eng.RegisterExecutor(string(model.AgentTypeMock), executor.NewMockExecutor("doctor_e2e_mock"))
	defer eng.Shutdown(3 * time.Second)
	appendE2EStep(&report, "new_engine", "ok", stepStart, "")

	stepStart = time.Now()
	q, err := eng.CreateQuest(ctx, "实现一个 hello world 函数并加上单元测试", model.QuestTypeExecute, workDir, orchestrator.CreateQuestOptions{
		WorkspaceMode: model.WorkspaceCopy,
	})
	if err != nil {
		return failE2E(report, "create_quest", err)
	}
	report.QuestID = q.ID
	appendE2EStep(&report, "create_quest", "ok", stepStart, q.ID)

	stepStart = time.Now()
	if err := eng.StartQuest(ctx, q.ID); err != nil {
		return failE2E(report, "start_quest", err)
	}
	if _, err := waitForQuestStatus(ctx, eng, q.ID, model.QuestStatusUserReview); err != nil {
		return failE2E(report, "wait_user_review", err)
	}
	appendE2EStep(&report, "wait_user_review", "ok", stepStart, string(model.QuestStatusUserReview))

	stepStart = time.Now()
	if err := writeDoctorE2EWorkspaceMutation(eng, q.ID); err != nil {
		return failE2E(report, "workspace_mutation", err)
	}
	appendE2EStep(&report, "workspace_mutation", "ok", stepStart, "doctor-e2e.txt")

	stepStart = time.Now()
	if err := eng.ResolveUserReview(ctx, q.ID, model.VerdictPass, "doctor e2e pass"); err != nil {
		return failE2E(report, "user_review_pass", err)
	}
	if _, err := waitForQuestStatus(ctx, eng, q.ID, model.QuestStatusSuccess); err != nil {
		return failE2E(report, "wait_success", err)
	}
	appendE2EStep(&report, "user_review_pass", "ok", stepStart, string(model.QuestStatusSuccess))

	stepStart = time.Now()
	diff, err := eng.ComputeDiff(ctx, q.ID)
	if err != nil {
		return failE2E(report, "compute_diff", err)
	}
	report.Diff = &doctorE2EDiff{
		ChangedFiles: diff.ChangedFiles,
		Additions:    diff.Additions,
		Deletions:    diff.Deletions,
		Stat:         diff.Stat,
	}
	appendE2EStep(&report, "compute_diff", "ok", stepStart, diff.Stat)

	stepStart = time.Now()
	warnings, err := eng.ApplyQuest(ctx, q.ID, true)
	if err != nil {
		return failE2E(report, "apply", err)
	}
	for _, warning := range warnings {
		report.Warnings = append(report.Warnings, fmt.Sprintf("[%s] %s: %s", warning.Severity, warning.Category, warning.Message))
	}
	appendE2EStep(&report, "apply", "ok", stepStart, fmt.Sprintf("warnings=%d", len(warnings)))
	report.DurationMs = time.Since(start).Milliseconds()
	return report, nil
}

func runRealAgentE2EContract(ctx context.Context, log *slog.Logger, dataDir, workDir, agentName string, timeout time.Duration, keep bool) (doctorE2EReport, error) {
	start := time.Now()
	report := doctorE2EReport{
		Status:    "ok",
		Steps:     []doctorE2EStep{},
		Contracts: map[string]bool{},
		Details:   map[string]interface{}{},
		Meta: map[string]string{
			"executor":       "real",
			"agent":          agentName,
			"scope":          "agent-quest-contract",
			"workspace_mode": string(model.WorkspaceReadOnly),
		},
	}
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var cleanup []string
	defer func() {
		report.DurationMs = time.Since(start).Milliseconds()
		if keep {
			return
		}
		for _, p := range cleanup {
			_ = os.RemoveAll(p)
		}
	}()

	if dataDir == "" {
		dir, err := os.MkdirTemp("", "gloop-doctor-real-e2e-data-*")
		if err != nil {
			return failE2E(report, "prepare_data_dir", err)
		}
		dataDir = dir
		cleanup = append(cleanup, dir)
	}
	if workDir == "" {
		dir, err := os.MkdirTemp("", "gloop-doctor-real-e2e-work-*")
		if err != nil {
			return failE2E(report, "prepare_work_dir", err)
		}
		workDir = dir
		cleanup = append(cleanup, dir)
	}
	report.DataDir = dataDir
	report.WorkDir = workDir

	stepStart := time.Now()
	root, err := fsstore.Open(dataDir)
	if err != nil {
		return failE2E(report, "open_root", err)
	}
	appendE2EStep(&report, "open_root", "ok", stepStart, dataDir)

	stepStart = time.Now()
	if err := os.WriteFile(filepath.Join(workDir, "README.md"), []byte("# Gloop Real Agent E2E\n\nbaseline\n"), 0o644); err != nil {
		return failE2E(report, "prepare_project", err)
	}
	_, err = root.InitDefaultFiles(detectAgents(), workDir)
	if err != nil {
		return failE2E(report, "init_defaults", err)
	}
	agentCfg, err := root.GetAgent(agentName)
	if err != nil {
		return failE2E(report, "load_agent", err)
	}
	agentCfg.Enabled = true
	agentCfg.Env = mergeDoctorAgentEnv(agentCfg.Env, doctorRealAgentEnv())
	if err := root.SaveAgent(agentCfg); err != nil {
		return failE2E(report, "enable_agent", err)
	}
	if err := configureRealAgentContractAdventurers(root, agentName); err != nil {
		return failE2E(report, "configure_adventurers", err)
	}
	appendE2EStep(&report, "prepare_project", "ok", stepStart, workDir)

	stepStart = time.Now()
	cfg, err := root.LoadConfig()
	if err != nil {
		return failE2E(report, "load_config", err)
	}
	cfg.MaxTurnsPerPhase = 4
	cfg.MaxReworkPerQuest = 0
	cfg.MaxConsecutiveAgentErrors = 2
	cfg.MaxNoProgressTurns = 2
	cfg.DefaultWorkingDir = workDir
	cfg.DefaultWorkspaceMode = string(model.WorkspaceReadOnly)
	cfg.MaxDurationPerQuestMs = int64(timeout / time.Millisecond)
	// doctor 验证显式 user_review→approve 全链路，关闭 HOTL 自主闭环
	hotlAutoClose := false
	cfg.HotlAutoClose = &hotlAutoClose
	if err := root.SaveConfig(cfg); err != nil {
		return failE2E(report, "save_config", err)
	}
	bus := events.NewBus()
	eng, err := orchestrator.NewEngine(root, cfg, bus, workDir, log)
	if err != nil {
		return failE2E(report, "new_engine", err)
	}
	if _, err := eng.RegisterAgentExecutor(agentCfg); err != nil {
		return failE2E(report, "register_agent", err)
	}
	defer eng.Shutdown(3 * time.Second)
	appendE2EStep(&report, "new_engine", "ok", stepStart, agentName)

	stepStart = time.Now()
	q, err := eng.CreateQuest(ctx, realAgentE2EQuery(), model.QuestTypeExecute, workDir, orchestrator.CreateQuestOptions{
		WarriorID:     "adv_doctor_real_warrior",
		MageID:        "adv_doctor_real_mage",
		WorkspaceMode: model.WorkspaceReadOnly,
		Intensity:     model.QuestIntensityStandard,
	})
	if err != nil {
		return failE2E(report, "create_quest", err)
	}
	report.QuestID = q.ID
	appendE2EStep(&report, "create_quest", "ok", stepStart, q.ID)

	stepStart = time.Now()
	if err := eng.StartQuest(ctx, q.ID); err != nil {
		return failE2E(report, "start_quest", err)
	}
	if _, err := waitForQuestStatus(ctx, eng, q.ID, model.QuestStatusUserReview); err != nil {
		enrichRealAgentE2EReport(root, &report, q.ID)
		return failE2E(report, "wait_user_review", err)
	}
	appendE2EStep(&report, "wait_user_review", "ok", stepStart, string(model.QuestStatusUserReview))

	stepStart = time.Now()
	if err := validateRealAgentContracts(root, &report, q.ID); err != nil {
		return failE2E(report, "validate_contracts", err)
	}
	appendE2EStep(&report, "validate_contracts", "ok", stepStart, fmt.Sprintf("contracts=%v", report.Contracts))
	report.DurationMs = time.Since(start).Milliseconds()
	return report, nil
}

func realAgentE2EQuery() string {
	return "Doctor real-agent contract check. Do not edit files. " +
		"First, complete the warrior phase by running `gloop phase done --summary GLOOP_REAL_AGENT_CONTRACT_OK`. " +
		"During review, verify that marker and run `$GLOOP_DOCTOR_CURRENT_BINARY review pass --comment \"contract ok\" --score 9`. " +
		"Do not use another gloop binary."
}

func doctorRealAgentEnv() map[string]string {
	env := map[string]string{
		"GLOOP_DOCTOR_CURRENT_BINARY": os.Args[0],
		"GLOOP_NO_UPDATE_NOTIFIER":    "1",
		"NO_UPDATE_NOTIFIER":          "1",
	}
	if dir := filepath.Dir(os.Args[0]); dir != "." && dir != "" {
		env["PATH"] = dir + string(os.PathListSeparator) + os.Getenv("PATH")
	}
	return env
}

func mergeDoctorAgentEnv(base map[string]string, extra map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range base {
		out[k] = v
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}

func configureRealAgentContractAdventurers(root *fsstore.Root, agentName string) error {
	now := fsstore.NowMs()
	items := []*fsstore.AdventurerFile{
		{
			ID:          "adv_doctor_real_warrior",
			Name:        "Doctor Real Warrior",
			Class:       model.ClassWarrior,
			Description: "doctor e2e real-agent warrior",
			Status:      model.AdventurerActive,
			Agent:       agentName,
			CustomPrompt: strings.Join([]string{
				"Doctor contract mode.",
				"Do not investigate broadly.",
				"Do not edit files.",
				"Your only required action is to run: $GLOOP_DOCTOR_CURRENT_BINARY phase done --summary GLOOP_REAL_AGENT_CONTRACT_OK",
				"Do not use any other gloop binary.",
			}, "\n"),
			Level:       1,
			CreatedAtMs: now,
		},
		{
			ID:          "adv_doctor_real_mage",
			Name:        "Doctor Real Mage",
			Class:       model.ClassMage,
			Description: "doctor e2e real-agent mage",
			Status:      model.AdventurerActive,
			Agent:       agentName,
			CustomPrompt: strings.Join([]string{
				"Doctor contract mode.",
				"Do not investigate broadly or inspect extra files.",
				"Use only the visible warrior artifact and review context.",
				"If the warrior artifact includes GLOOP_REAL_AGENT_CONTRACT_OK, immediately run: $GLOOP_DOCTOR_CURRENT_BINARY review pass --comment \"contract ok\" --score 9",
				"Do not use any other gloop binary.",
			}, "\n"),
			Level:       1,
			CreatedAtMs: now,
		},
	}
	for _, item := range items {
		if err := root.SaveAdventurer(item); err != nil {
			return err
		}
	}
	return nil
}

func validateRealAgentContracts(root *fsstore.Root, report *doctorE2EReport, qid string) error {
	enrichRealAgentE2EReport(root, report, qid)
	required := []string{"warrior_phase_checkpoint", "mage_review_quest", "final_verdict_pass"}
	for _, key := range required {
		if !report.Contracts[key] {
			return fmt.Errorf("real agent contract missing: %s", key)
		}
	}
	return nil
}

func enrichRealAgentE2EReport(root *fsstore.Root, report *doctorE2EReport, qid string) {
	qs := fsstore.NewQuestStore(root)
	q, err := qs.LoadQuest(qid)
	if err == nil {
		report.Details["quest_status"] = string(q.Status)
		report.Details["final_verdict"] = string(q.FinalVerdict)
		report.Details["final_comment"] = truncateDoctorText(q.FinalComment, 500)
		report.Contracts["final_verdict_pass"] = q.FinalVerdict == model.VerdictPass
	}
	warriorRows, _ := qs.ReadSessionRows(qid, "warrior_0", 0)
	mageRows, _ := qs.ReadSessionRows(qid, "mage_0", 0)
	report.Details["warrior_turn_rows"] = len(warriorRows)
	report.Details["mage_turn_rows"] = len(mageRows)
	for _, row := range warriorRows {
		if row.Kind == "tool_call" && row.ToolName == "phase_checkpoint" {
			report.Contracts["warrior_phase_checkpoint"] = true
			report.Details["warrior_phase_checkpoint_args"] = truncateDoctorText(row.ToolArgs, 500)
		}
		if row.Kind == "tool_result" && row.ToolName == "gloop_cli_signal" && strings.Contains(row.Content, "GLOOP_REAL_AGENT_CONTRACT_OK") {
			report.Contracts["warrior_phase_checkpoint"] = true
			report.Details["warrior_phase_checkpoint_args"] = truncateDoctorText(row.Content, 500)
		}
		if row.Kind == "message" && strings.Contains(row.Content, "GLOOP_REAL_AGENT_CONTRACT_OK") {
			report.Contracts["warrior_marker_seen"] = true
		}
	}
	for _, row := range mageRows {
		if row.Kind == "tool_call" && row.ToolName == "review_quest" {
			report.Contracts["mage_review_quest"] = true
			report.Details["mage_review_quest_args"] = truncateDoctorText(row.ToolArgs, 500)
		}
		if row.Kind == "tool_result" && row.ToolName == "gloop_cli_signal" && strings.Contains(row.Content, `"phase_verdict":"pass"`) {
			report.Contracts["mage_review_quest"] = true
			report.Details["mage_review_quest_args"] = truncateDoctorText(row.Content, 500)
		}
	}
}

func configureMockAdventurers(root *fsstore.Root) error {
	now := fsstore.NowMs()
	items := []*fsstore.AdventurerFile{
		{
			ID:          "adv_warrior_001",
			Name:        "Doctor Warrior",
			Class:       model.ClassWarrior,
			Description: "doctor e2e mock warrior",
			Status:      model.AdventurerActive,
			Agent:       string(model.AgentTypeMock),
			Level:       1,
			CreatedAtMs: now,
		},
		{
			ID:          "adv_mage_001",
			Name:        "Doctor Mage",
			Class:       model.ClassMage,
			Description: "doctor e2e mock mage",
			Status:      model.AdventurerActive,
			Agent:       string(model.AgentTypeMock),
			Level:       1,
			CreatedAtMs: now,
		},
	}
	for _, item := range items {
		if err := root.SaveAdventurer(item); err != nil {
			return err
		}
	}
	return nil
}

func writeDoctorE2EWorkspaceMutation(eng *orchestrator.Engine, qid string) error {
	q, err := eng.GetQuest(qid)
	if err != nil {
		return err
	}
	if q.WorkspacePath == "" {
		return fmt.Errorf("quest workspace path is empty")
	}
	content := []byte("GLOOP_DOCTOR_E2E_OK\n")
	return os.WriteFile(filepath.Join(q.WorkspacePath, "doctor-e2e.txt"), content, 0o644)
}

func waitForQuestStatus(ctx context.Context, eng *orchestrator.Engine, qid string, target model.QuestStatus) (*fsstore.QuestMeta, error) {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	var last model.QuestStatus
	for {
		q, err := eng.GetQuest(qid)
		if err == nil {
			last = q.Status
			if q.Status == target {
				return q, nil
			}
			if q.Status == model.QuestStatusFailed || q.Status == model.QuestStatusCancelled || q.Status == model.QuestStatusBlocked {
				return q, fmt.Errorf("quest reached %s before %s", q.Status, target)
			}
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting for %s; last status=%s", target, last)
		case <-ticker.C:
		}
	}
}

func appendE2EStep(report *doctorE2EReport, name, status string, start time.Time, detail string) {
	report.Steps = append(report.Steps, doctorE2EStep{
		Name:       name,
		Status:     status,
		DurationMs: time.Since(start).Milliseconds(),
		Detail:     detail,
	})
}

func failE2E(report doctorE2EReport, step string, err error) (doctorE2EReport, error) {
	report.Status = "error"
	report.Error = err.Error()
	report.Steps = append(report.Steps, doctorE2EStep{Name: step, Status: "error", Detail: err.Error()})
	return report, err
}

func printDoctorE2EReport(report doctorE2EReport) {
	fmt.Printf("Gloop E2E doctor: %s (%dms)\n", report.Status, report.DurationMs)
	fmt.Printf("Data dir: %s\n", report.DataDir)
	fmt.Printf("Work dir: %s\n", report.WorkDir)
	if report.QuestID != "" {
		fmt.Printf("Quest: %s\n", report.QuestID)
	}
	for _, step := range report.Steps {
		fmt.Printf("  %-18s %-6s %dms", step.Name, step.Status, step.DurationMs)
		if step.Detail != "" {
			fmt.Printf("  %s", step.Detail)
		}
		fmt.Println()
	}
	if report.Diff != nil {
		fmt.Printf("Diff: files=%d +%d -%d %s\n", report.Diff.ChangedFiles, report.Diff.Additions, report.Diff.Deletions, report.Diff.Stat)
	}
	if len(report.Contracts) > 0 {
		fmt.Println("Contracts:")
		keys := make([]string, 0, len(report.Contracts))
		for key := range report.Contracts {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			status := "missing"
			if report.Contracts[key] {
				status = "ok"
			}
			fmt.Printf("  %-28s %s\n", key, status)
		}
	}
	if len(report.Details) > 0 {
		fmt.Println("Details:")
		keys := make([]string, 0, len(report.Details))
		for key := range report.Details {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			fmt.Printf("  %-28s %s\n", key, truncateDoctorText(fmt.Sprint(report.Details[key]), 500))
		}
	}
	if len(report.Warnings) > 0 {
		fmt.Printf("Warnings: %d\n", len(report.Warnings))
		for _, warning := range report.Warnings {
			fmt.Printf("  %s\n", warning)
		}
	}
	if report.Error != "" {
		fmt.Printf("Error: %s\n", report.Error)
	}
}
