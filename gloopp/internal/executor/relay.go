package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

// ClaudeCodeExecutor 驱动 Claude Code 系的 CLI agent。
// 同时服务两类二进制：官方 `claude`（@anthropic-ai/claude-code）与字节
// `relay`（@bytedance-relay/claude-code 套壳）——二者 argv 兼容、stream-json
// 输出同构，故复用同一适配器。displayName 按二进制名区分（Claude CLI / Relay CLI）。
type ClaudeCodeExecutor struct {
	*BaseExecutor
	*CLIExecutorBase
}

type ClaudeCodeOpt struct {
	ExecID       string
	BinPath      string
	DefaultModel string
	WorkingRoot  string
	AllowedTools []string
}

func NewClaudeCodeExecutor(opt ClaudeCodeOpt) *ClaudeCodeExecutor {
	if opt.ExecID == "" {
		opt.ExecID = "relay-cli"
	}
	if opt.DefaultModel == "" {
		opt.DefaultModel = "auto"
	}
	bin := opt.BinPath
	if bin == "" {
		bin = "relay"
	}
	baseName := filepath.Base(bin)
	displayName := "Relay CLI"
	if strings.Contains(baseName, "claude") {
		displayName = "Claude CLI"
	}
	base := NewBase(opt.ExecID, displayName, model.AgentTypeCLI)
	base.capabs = []Capability{CapToolUse, CapStreaming, CapCodeExec, CapContextExport}
	base.tier = CapabilityTierB
	base.models = []ModelSpec{
		{ID: opt.DefaultModel, Name: displayName + " default", Context: 200000},
		{ID: "auto", Name: "auto", Context: 200000},
	}
	base.defModel = opt.DefaultModel
	if len(opt.AllowedTools) == 0 {
		opt.AllowedTools = []string{"Bash", "Read", "Edit", "Write", "Glob", "Grep", "WebFetch", "WebSearch"}
	}
	return &ClaudeCodeExecutor{
		BaseExecutor:    base,
		CLIExecutorBase: NewCLIExecutorBase(bin, opt.WorkingRoot, opt.AllowedTools),
	}
}

func (r *ClaudeCodeExecutor) CreateSession(ctx context.Context, sessionID string, opts SessionConfig) (*SessionHandle, error) {
	h, _, err := r.CLIExecutorBase.CreateSession(sessionID, opts)
	if err != nil {
		return nil, err
	}
	r.Heartbeat(r.activeCountLocked())
	return h, nil
}

func (r *ClaudeCodeExecutor) CloseSession(ctx context.Context, sessionID string) error {
	r.CLIExecutorBase.CloseSession(sessionID)
	r.Heartbeat(r.activeCountLocked())
	return nil
}

func (r *ClaudeCodeExecutor) GetSessionStatus(ctx context.Context, sessionID string) (interface{}, error) {
	return r.CLIExecutorBase.GetSessionStatus(sessionID)
}

func (r *ClaudeCodeExecutor) activeCountLocked() int {
	r.Mu.RLock()
	defer r.Mu.RUnlock()
	return len(r.Sessions)
}

func (r *ClaudeCodeExecutor) SendMessage(ctx context.Context, sessionID string, msg Message, modelName string, stream chan<- StreamChunk) (*ChatResponse, error) {
	st, ok := r.CLIExecutorBase.Get(sessionID)
	if !ok {
		return nil, fmt.Errorf("session 不存在: %s", sessionID)
	}
	start := time.Now()
	r.CLIExecutorBase.AppendHistory(sessionID, msg)

	promptText := buildPromptFromHistory(st.History)
	modelChoice := pick(modelName, r.DefaultModel())
	args := relayCommandArgs(st, modelChoice)

	cmd := exec.CommandContext(ctx, r.BinPath, args...)
	cmd.Dir = st.WorkingDir
	cmd.Stdin = strings.NewReader(promptText)
	if len(st.Env) > 0 {
		cmd.Env = append(cmd.Environ(), st.Env...)
	}
	result, runErr := runCommandWithRuntimeEvents(sessionID, cmd, stream, start, map[string]any{
		"command":    r.BinPath,
		"adapter":    "relay",
		"model":      modelChoice,
		"workingDir": st.WorkingDir,
	}, func() {
		r.CLIExecutorBase.SetActiveCmd(sessionID, cmd)
	}, func() {
		r.CLIExecutorBase.ClearActiveCmd(sessionID)
	})
	if runErr != nil {
		return nil, runErr
	}
	waitErr := result.WaitErr
	stdoutStr := result.Stdout

	parsed := parseRelayStreamJSON(stdoutStr)
	var fallbackErr error
	if waitErr != nil || parsed.IsError {
		extra := ""
		if waitErr != nil {
			extra = waitErr.Error()
		}
		if extra == "" {
			extra = "relay returned error"
		}
		// stdout 里 relay 的 result 事件 is_error=true 时携带错误描述，
		// 是定位 agent 失败真因的首要线索，必须落进 fallbackErr（进而进 session 错误行）。
		if parsed.IsError && parsed.Content != "" {
			extra += " | stdout: " + truncate(parsed.Content, 400)
		}
		if result.Stderr != "" {
			extra += " | stderr: " + truncate(result.Stderr, 200)
		}
		fallbackErr = fmt.Errorf("%s", extra)
	}
	resp := r.CLIExecutorBase.completeCLIResponse(st, msg, cliCompletion{
		Content:    parsed.Content,
		StopReason: parsed.StopReason,
		UsageIn:    parsed.UsageIn,
		UsageOut:   parsed.UsageOut,
		TotalCost:  parsed.TotalCost,
		ModelUsage: parsed.ModelUsage,
	}, start, fallbackErr, stream)
	r.Heartbeat(r.activeCountLocked())
	if fallbackErr != nil {
		return resp, fallbackErr
	}
	return resp, nil
}

func relayCommandArgs(st *cliSessionState, modelChoice string) []string {
	args := []string{
		"-p",
		"--verbose",
		"--output-format=stream-json",
		"--permission-mode", "bypassPermissions",
		"--no-session-persistence",
		"--session-id", UUID(),
		"--add-dir", st.WorkingDir,
		"--system-prompt", st.History[0].Content,
	}
	if modelChoice != "" && modelChoice != "auto" && modelChoice != "default" {
		args = append(args, "--model", modelChoice)
	}
	if len(st.AllowedTools) > 0 {
		toolList := strings.Join(st.AllowedTools, ",")
		args = append(args, "--allowedTools", toolList)
	}
	return args
}

type relayParseResult struct {
	Content    string
	StopReason string
	UsageIn    int64
	UsageOut   int64
	TotalCost  float64
	ModelUsage map[string]any
	IsError    bool
}

func parseRelayStreamJSON(raw string) relayParseResult {
	var out relayParseResult
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
		case "assistant":
			if msg, ok := obj["message"].(map[string]any); ok {
				if text := relayMessageText(msg); text != "" {
					out.Content = text
				}
				if usage, ok := msg["usage"].(map[string]any); ok {
					out.UsageIn += int64FromAny(usage["input_tokens"])
					out.UsageOut += int64FromAny(usage["output_tokens"])
				}
				if sr, ok := msg["stop_reason"].(string); ok && sr != "" {
					out.StopReason = sr
				}
			}
			if errText, ok := obj["error"].(string); ok && errText != "" {
				out.IsError = true
			}
		case "result":
			if res, ok := obj["result"].(string); ok && res != "" {
				out.Content = res
			}
			if sr, ok := obj["stop_reason"].(string); ok && sr != "" {
				out.StopReason = sr
			}
			out.IsError = boolFromAny(obj["is_error"])
			out.UsageIn += int64FromAnyPath(obj, "usage", "input_tokens")
			out.UsageOut += int64FromAnyPath(obj, "usage", "output_tokens")
			if cost, ok := obj["total_cost_usd"].(float64); ok {
				out.TotalCost = cost
			}
			if mu, ok := obj["modelUsage"].(map[string]any); ok {
				out.ModelUsage = mu
			}
		}
	}
	if out.Content == "" {
		out.Content = strings.TrimSpace(raw)
	}
	return out
}

func relayMessageText(msg map[string]any) string {
	content, ok := msg["content"].([]any)
	if !ok {
		return ""
	}
	var sb strings.Builder
	for _, item := range content {
		part, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if typ, _ := part["type"].(string); typ != "" && typ != "text" {
			continue
		}
		if text, ok := part["text"].(string); ok {
			sb.WriteString(text)
		}
	}
	return sb.String()
}

func (r *ClaudeCodeExecutor) ExportContext(ctx context.Context, sessionID string) ([]Message, error) {
	return r.CLIExecutorBase.ExportContext(sessionID)
}

func (r *ClaudeCodeExecutor) ImportContext(ctx context.Context, sessionID string, history []Message) error {
	return r.CLIExecutorBase.ImportContext(sessionID, history)
}

func (r *ClaudeCodeExecutor) Interrupt(ctx context.Context, sessionID, reason string) error {
	return r.CLIExecutorBase.WrappedInterrupt(ctx, sessionID, reason)
}
func (r *ClaudeCodeExecutor) Resume(ctx context.Context, sessionID string) error { return nil }
func (r *ClaudeCodeExecutor) GetAvailableTools(ctx context.Context, sessionID string) ([]ToolDef, error) {
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
