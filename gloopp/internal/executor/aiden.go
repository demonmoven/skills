package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

// ==================== Aiden (X 系列) Executor ====================
// 通过本机 aiden CLI 的 --stream-json 模式真接入。
//
// 支持多个变种（variant），每个变种都是独立 agent：
//   - base:    基础 aiden（`aiden --one-shot --stream-json "prompt"）
//   - x_claude: Aiden X Claude（`aiden x claude --stream-json --print "prompt"`）
//   - x_codex:  Aiden X Codex（`aiden x codex --stream-json exec "prompt"`）
//
// 所有变种共用同一套 stream-json 输出格式（NDJSON）:
//   - session: 会话信息
//   - event[name=message:create/update]: 消息内容与 usage
//   - done: 结束状态

type AidenVariant string

const (
	AidenVariantBase    AidenVariant = "base"
	AidenVariantXClaude AidenVariant = "x_claude"
	AidenVariantXCodex  AidenVariant = "x_codex"
)

type AidenExecutor struct {
	*BaseExecutor
	*CLIExecutorBase
	variant AidenVariant
}

type AidenOpt struct {
	ExecID       string
	Variant      AidenVariant
	BinPath      string
	DefaultModel string
	WorkingRoot  string
	AllowedTools []string
	DisplayName  string
}

func NewAidenExecutor(opt AidenOpt) *AidenExecutor {
	if opt.ExecID == "" {
		opt.ExecID = "aiden-cli"
	}
	if opt.Variant == "" {
		opt.Variant = AidenVariantBase
	}
	if opt.DefaultModel == "" {
		opt.DefaultModel = "auto"
	}
	bin := opt.BinPath
	if bin == "" {
		bin = "aiden"
	}

	displayName := opt.DisplayName
	if displayName == "" {
		switch opt.Variant {
		case AidenVariantXClaude:
			displayName = "Aiden X Claude"
		case AidenVariantXCodex:
			displayName = "Aiden X Codex"
		default:
			displayName = "Aiden CLI"
		}
	}

	base := NewBase(opt.ExecID, displayName, model.AgentTypeCLI)
	base.capabs = []Capability{CapToolUse, CapStreaming, CapCodeExec, CapContextExport}
	switch opt.Variant {
	case AidenVariantXClaude, AidenVariantXCodex:
		base.tier = CapabilityTierB
	default:
		base.tier = CapabilityTierC
	}
	base.models = aidenDefaultModels(opt.Variant)
	base.defModel = opt.DefaultModel

	if len(opt.AllowedTools) == 0 {
		opt.AllowedTools = []string{"Bash", "Read", "Edit", "Write", "Glob", "Grep", "WebFetch", "WebSearch"}
	}

	return &AidenExecutor{
		BaseExecutor:    base,
		CLIExecutorBase: NewCLIExecutorBase(bin, opt.WorkingRoot, opt.AllowedTools),
		variant:         opt.Variant,
	}
}

// aidenDefaultModels 返回各变种的默认模型列表
func aidenDefaultModels(v AidenVariant) []ModelSpec {
	switch v {
	case AidenVariantXClaude:
		return []ModelSpec{
			{ID: "auto", Name: "Auto", Context: 270000},
			{ID: "seedcode0611", Name: "Seed Code 0611", Context: 270000},
			{ID: "alwaysday1", Name: "AlwaysDay 1", Context: 270000},
			{ID: "deepseek-v4-pro", Name: "DeepSeek V4 Pro", Context: 270000},
			{ID: "gpt-5.4", Name: "GPT 5.4", Context: 270000},
		}
	case AidenVariantXCodex:
		return []ModelSpec{
			{ID: "auto", Name: "Auto", Context: 270000},
			{ID: "gpt-5.4", Name: "GPT 5.4", Context: 270000},
			{ID: "gpt-5.5-paygo", Name: "GPT 5.5 (PayGo)", Context: 270000},
		}
	default:
		return []ModelSpec{
			{ID: "auto", Name: "Auto", Context: 270000},
			{ID: "gpt-5.4", Name: "GPT 5.4", Context: 270000},
		}
	}
}

// ============== 会话生命周期（委托给 CLIExecutorBase） ==============

func (a *AidenExecutor) CreateSession(ctx context.Context, sessionID string, opts SessionConfig) (*SessionHandle, error) {
	h, _, err := a.CLIExecutorBase.CreateSession(sessionID, opts)
	if err != nil {
		return nil, err
	}
	a.Heartbeat(a.activeCountLocked())
	return h, nil
}

func (a *AidenExecutor) CloseSession(ctx context.Context, sessionID string) error {
	a.CLIExecutorBase.CloseSession(sessionID)
	a.Heartbeat(a.activeCountLocked())
	return nil
}

func (a *AidenExecutor) GetSessionStatus(ctx context.Context, sessionID string) (interface{}, error) {
	return a.CLIExecutorBase.GetSessionStatus(sessionID)
}

func (a *AidenExecutor) DisplayName() string { return a.Name() }

func (a *AidenExecutor) activeCountLocked() int {
	a.Mu.RLock()
	defer a.Mu.RUnlock()
	return len(a.Sessions)
}

// ============== SendMessage ==============

func (a *AidenExecutor) SendMessage(ctx context.Context, sessionID string, msg Message, modelName string, stream chan<- StreamChunk) (*ChatResponse, error) {
	st, ok := a.CLIExecutorBase.Get(sessionID)
	if !ok {
		return nil, fmt.Errorf("session 不存在: %s", sessionID)
	}
	start := time.Now()
	a.CLIExecutorBase.AppendHistory(sessionID, msg)

	promptText := buildPromptFromHistory(st.History)
	userPromptText := buildPromptFromHistory(nonSystemHistory(st.History))
	modelChoice := pick(modelName, a.DefaultModel())

	// 根据变种构建 args
	args := a.buildArgs(modelChoice, st, promptText, userPromptText)

	cmd := exec.CommandContext(ctx, a.BinPath, args...)
	cmd.Dir = st.WorkingDir
	if len(st.Env) > 0 {
		cmd.Env = append(cmd.Environ(), st.Env...)
	}

	result, runErr := runCommandWithRuntimeEvents(sessionID, cmd, stream, start, map[string]any{
		"command":    a.BinPath,
		"adapter":    "aiden",
		"model":      modelChoice,
		"workingDir": st.WorkingDir,
	}, func() {
		a.CLIExecutorBase.SetActiveCmd(sessionID, cmd)
	}, func() {
		a.CLIExecutorBase.ClearActiveCmd(sessionID)
	})
	if runErr != nil {
		return nil, runErr
	}
	waitErr := result.WaitErr
	stdoutStr := result.Stdout

	// 解析 stream-json
	parsed := parseAidenStreamJSON(stdoutStr)
	var fallbackErr error
	if waitErr != nil || parsed.IsError {
		extra := ""
		if waitErr != nil {
			extra = waitErr.Error()
		}
		if extra == "" {
			extra = "aiden returned error"
		}
		if result.Stderr != "" {
			extra += " | stderr: " + truncate(result.Stderr, 200)
		}
		if stdoutStr != "" && parsed.Content == "" {
			extra += " | stdout: " + truncate(stdoutStr, 200)
		}
		fallbackErr = fmt.Errorf("%s", extra)
	}

	resp := a.CLIExecutorBase.completeCLIResponse(st, msg, cliCompletion{
		Content:    parsed.Content,
		StopReason: parsed.StopReason,
		UsageIn:    parsed.UsageIn,
		UsageOut:   parsed.UsageOut,
		TotalCost:  parsed.TotalCost,
		ModelUsage: parsed.ModelUsage,
	}, start, fallbackErr, stream)
	a.Heartbeat(a.activeCountLocked())
	if fallbackErr != nil {
		return resp, fallbackErr
	}
	return resp, nil
}

// buildArgs 根据变种构建命令行参数
func (a *AidenExecutor) buildArgs(modelChoice string, st *cliSessionState, promptText, userPromptText string) []string {
	switch a.variant {
	case AidenVariantXClaude:
		return a.buildXClaudeArgs(modelChoice, st, userPromptText)
	case AidenVariantXCodex:
		return a.buildXCodexArgs(modelChoice, st, userPromptText)
	default:
		return a.buildBaseArgs(modelChoice, st, promptText)
	}
}

func nonSystemHistory(history []Message) []Message {
	out := make([]Message, 0, len(history))
	for _, msg := range history {
		if msg.Role == "system" {
			continue
		}
		out = append(out, msg)
	}
	return out
}

// buildBaseArgs 基础 aiden 模式
func (a *AidenExecutor) buildBaseArgs(modelChoice string, st *cliSessionState, promptText string) []string {
	args := []string{
		"--one-shot",
		"--stream-json",
		"--permission-mode", "agentFull",
	}
	if modelChoice != "" && modelChoice != "auto" && modelChoice != "default" {
		args = append(args, "--model", modelChoice)
	}
	if len(st.AllowedTools) > 0 {
		args = append(args, "--allowedTools", strings.Join(st.AllowedTools, ","))
	}
	if len(st.History) > 0 && st.History[0].Role == "system" {
		args = append(args, "--system-prompt", st.History[0].Content)
	}
	args = append(args, promptText)
	return args
}

// buildXClaudeArgs Aiden X Claude 模式
func (a *AidenExecutor) buildXClaudeArgs(modelChoice string, st *cliSessionState, promptText string) []string {
	args := []string{"x", "claude", "--stream-json"}
	if modelChoice != "" && modelChoice != "auto" && modelChoice != "default" {
		args = append(args, "--model", modelChoice)
	}
	if len(st.History) > 0 && st.History[0].Role == "system" {
		args = append(args, "--system-prompt", st.History[0].Content)
	}
	// X claude 通过 --print 传递 prompt
	args = append(args, "--print", promptText)
	return args
}

// buildXCodexArgs Aiden X Codex 模式
func (a *AidenExecutor) buildXCodexArgs(modelChoice string, st *cliSessionState, promptText string) []string {
	args := []string{"x", "codex", "--stream-json"}
	if modelChoice != "" && modelChoice != "auto" && modelChoice != "default" {
		args = append(args, "--model", modelChoice)
	}
	args = append(args, "exec")
	if len(st.History) > 0 && st.History[0].Role == "system" {
		args = append(args, "--system-prompt", st.History[0].Content)
	}
	// X codex 通过 exec 子命令传递 prompt
	args = append(args, promptText)
	return args
}

// ==================== 辅助：Aiden stream-json 解析 ====================

type aidenParseResult struct {
	Content    string
	StopReason string
	UsageIn    int64
	UsageOut   int64
	TotalCost  float64
	ModelUsage map[string]any
	IsError    bool
}

// parseAidenStreamJSON 解析 aiden --stream-json 输出的 NDJSON。
// 取最后一条 message:update 的 content 和 usage 作为最终结果。
func parseAidenStreamJSON(raw string) aidenParseResult {
	var out aidenParseResult
	scanner := newJSONLineScanner(raw)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var obj map[string]any
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			continue
		}
		typ, _ := obj["type"].(string)
		switch typ {
		case "event":
			ev, ok := obj["event"].(map[string]any)
			if !ok {
				continue
			}
			name, _ := ev["name"].(string)
			switch name {
			case "message:create", "message:update":
				if content, ok := ev["content"].(string); ok && content != "" {
					out.Content = content
				}
				if usage, ok := ev["usage"].(map[string]any); ok {
					if in, ok := usage["input_tokens"].(float64); ok {
						out.UsageIn = int64(in)
					}
					if outTok, ok := usage["output_tokens"].(float64); ok {
						out.UsageOut = int64(outTok)
					}
				}
			}
		case "done":
			status, _ := obj["status"].(string)
			if status == "failed" || status == "error" {
				out.IsError = true
			}
			if status == "completed" {
				out.StopReason = "stop"
			}
		}
	}
	// 兜底：出错时把整个 raw 当 content（便于排查）
	if out.Content == "" && out.IsError {
		out.Content = strings.TrimSpace(raw)
	}
	return out
}

// ============== 上下文导入导出 / Interrupt / Resume / Tools ==============

func (a *AidenExecutor) ExportContext(ctx context.Context, sessionID string) ([]Message, error) {
	return a.CLIExecutorBase.ExportContext(sessionID)
}
func (a *AidenExecutor) ImportContext(ctx context.Context, sessionID string, history []Message) error {
	return a.CLIExecutorBase.ImportContext(sessionID, history)
}

func (a *AidenExecutor) Interrupt(ctx context.Context, sessionID, reason string) error {
	return a.CLIExecutorBase.WrappedInterrupt(ctx, sessionID, reason)
}
func (a *AidenExecutor) Resume(ctx context.Context, sessionID string) error { return nil }

func (a *AidenExecutor) GetAvailableTools(ctx context.Context, sessionID string) ([]ToolDef, error) {
	return []ToolDef{
		{Name: "Bash", Description: "执行 shell 命令", Params: []ToolParam{{Name: "command", Type: "string", Required: true}}},
		{Name: "Read", Description: "读取文件", Params: []ToolParam{{Name: "path", Type: "string", Required: true}}},
		{Name: "Edit", Description: "编辑文件", Params: []ToolParam{
			{Name: "path", Type: "string", Required: true},
			{Name: "old_string", Type: "string", Required: true},
			{Name: "new_string", Type: "string", Required: true},
		}},
		{Name: "Write", Description: "写文件", Params: []ToolParam{
			{Name: "path", Type: "string", Required: true},
			{Name: "content", Type: "string", Required: true},
		}},
	}, nil
}
