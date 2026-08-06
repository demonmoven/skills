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

// ==================== TraeX Executor ====================
// 通过本机 traex CLI（traecli 的同名二进制）真接入。
//
// 单回合用法：
//
//	echo "prompt" | traex exec --skip-git-repo-check --json \
//	     -C <workingDir> --allowed-tool Bash --allowed-tool Read ...
//
// TraeX exec 输出的是 JSONL：thread.started → turn.started → item.completed → turn.completed
// 我们抽最后一条 agent_message 当结果，用 turn.completed 里的 usage 计数。
//
// 平台工具通过 CLI syscall 调用：agent 执行 `gloop <command>`，通过信号文件与 orchestrator 交互。

type TraeXExecutor struct {
	*BaseExecutor
	*CLIExecutorBase
}

type TraeXOpt struct {
	ExecID       string
	OwnerID      string
	DeviceID     string
	BinPath      string
	DefaultModel string
	WorkingRoot  string
	AllowedTools []string
}

func NewTraeXExecutor(opt TraeXOpt) *TraeXExecutor {
	if opt.ExecID == "" {
		opt.ExecID = "traex-cli"
	}
	if opt.DefaultModel == "" {
		opt.DefaultModel = "auto"
	}
	bin := opt.BinPath
	if bin == "" {
		bin = "traex"
	}
	base := NewBase(opt.ExecID, "TraeX CLI", model.AgentTypeCLI)
	base.capabs = []Capability{CapToolUse, CapStreaming, CapCodeExec, CapContextExport}
	base.tier = CapabilityTierB
	base.models = []ModelSpec{
		{ID: "auto", Name: "TraeX Auto", Context: 200000},
		{ID: "o3", Name: "OpenAI o3", Context: 200000},
		{ID: "gpt-5.5", Name: "GPT 5.5", Context: 200000},
	}
	base.defModel = opt.DefaultModel
	if len(opt.AllowedTools) == 0 {
		opt.AllowedTools = []string{"Bash", "Read", "Edit", "Write", "Glob", "Grep", "WebFetch", "WebSearch"}
	}
	cliBase := NewCLIExecutorBase(bin, opt.WorkingRoot, opt.AllowedTools)
	return &TraeXExecutor{
		BaseExecutor:    base,
		CLIExecutorBase: cliBase,
	}
}

// ============== 会话生命周期（委托给 CLIExecutorBase） ==============

func (t *TraeXExecutor) CreateSession(ctx context.Context, sessionID string, opts SessionConfig) (*SessionHandle, error) {
	h, _, err := t.CLIExecutorBase.CreateSession(sessionID, opts)
	if err != nil {
		return nil, err
	}
	t.Heartbeat(t.activeCountLocked())
	return h, nil
}

func (t *TraeXExecutor) CloseSession(ctx context.Context, sessionID string) error {
	t.CLIExecutorBase.CloseSession(sessionID)
	t.Heartbeat(t.activeCountLocked())
	return nil
}

func (t *TraeXExecutor) GetSessionStatus(ctx context.Context, sessionID string) (interface{}, error) {
	return t.CLIExecutorBase.GetSessionStatus(sessionID)
}

func (t *TraeXExecutor) activeCountLocked() int {
	t.Mu.RLock()
	defer t.Mu.RUnlock()
	return len(t.Sessions)
}

// ============== SendMessage ==============

func (t *TraeXExecutor) SendMessage(ctx context.Context, sessionID string, msg Message, model string, stream chan<- StreamChunk) (*ChatResponse, error) {
	st, ok := t.CLIExecutorBase.Get(sessionID)
	if !ok {
		return nil, fmt.Errorf("session 不存在: %s", sessionID)
	}
	start := time.Now()

	t.CLIExecutorBase.AppendHistory(sessionID, msg)

	resuming := strings.TrimSpace(st.TransportSessionID) != ""
	prompt := traexPromptForTurn(st, msg)
	modelChoice := pick(model, t.DefaultModel())

	// 构建 args
	args := traexCommandArgs(st)
	if !resuming {
		// 模型：空串 / "auto" / "default" 都不传 --model，让 traex 用默认模型
		// resume 子命令延续已有 thread，不再重复传初始 model。
		if modelChoice != "" && modelChoice != "auto" && modelChoice != "default" {
			args = append(args, "--model", modelChoice)
		}
	}
	for _, toolName := range st.AllowedTools {
		args = append(args, "--allowed-tool", toolName)
	}

	cmd := exec.CommandContext(ctx, t.BinPath, args...)
	cmd.Dir = st.WorkingDir
	cmd.Stdin = strings.NewReader(prompt)
	if len(st.Env) > 0 {
		cmd.Env = append(cmd.Environ(), st.Env...)
	}

	result, runErr := runCommandWithRuntimeEvents(sessionID, cmd, stream, start, map[string]any{
		"command":    t.BinPath,
		"adapter":    "traex",
		"model":      modelChoice,
		"workingDir": st.WorkingDir,
	}, func() {
		t.CLIExecutorBase.SetActiveCmd(sessionID, cmd)
	}, func() {
		t.CLIExecutorBase.ClearActiveCmd(sessionID)
	})
	if runErr != nil {
		return nil, runErr
	}
	waitErr := result.WaitErr
	stdoutStr := result.Stdout

	// 解析 JSONL，取最后一条 agent/assistant 相关内容
	parsed := parseTraeXJSONL(stdoutStr)
	if parsed.ThreadID != "" {
		t.CLIExecutorBase.SetTransportSessionID(st.SessionID, parsed.ThreadID)
	}
	var fallbackErr error
	if waitErr != nil {
		fallbackErr = traexFallbackError(resuming, st.TransportSessionID, parsed.Content, waitErr)
	}
	resp := t.CLIExecutorBase.completeCLIResponse(st, msg, cliCompletion{
		Content:    parsed.Content,
		StopReason: parsed.StopReason,
		UsageIn:    parsed.UsageIn,
		UsageOut:   parsed.UsageOut,
		TotalCost:  parsed.TotalCost,
		ModelUsage: parsed.ModelUsage,
	}, start, fallbackErr, stream)
	if len(parsed.FileChanges) > 0 {
		attachTraeXFileChanges(resp, parsed.FileChanges)
	}
	t.Heartbeat(t.activeCountLocked())
	if fallbackErr != nil {
		return resp, fallbackErr
	}
	return resp, nil
}

func attachTraeXFileChanges(resp *ChatResponse, changes []FileChangeObservation) {
	if resp == nil || len(changes) == 0 {
		return
	}
	meta, _ := resp.Meta.(map[string]any)
	if meta == nil {
		meta = map[string]any{}
	}
	meta["traex_file_changes"] = append([]FileChangeObservation(nil), changes...)
	resp.Meta = meta
}

func traexFallbackError(resuming bool, threadID string, content string, err error) error {
	if err == nil {
		return nil
	}
	if resuming {
		return traexResumeError(threadID, err)
	}
	if strings.TrimSpace(content) == "" {
		return err
	}
	return nil
}

func traexResumeError(threadID string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("resume_failed=true transport=traex thread_id=%s: %w", threadID, err)
}

func traexCommandArgs(st *cliSessionState) []string {
	if st != nil && strings.TrimSpace(st.TransportSessionID) != "" {
		return []string{
			"exec",
			"resume",
			st.TransportSessionID,
			"--skip-git-repo-check",
			"--json",
		}
	}
	workingDir := ""
	if st != nil {
		workingDir = st.WorkingDir
	}
	args := []string{
		"exec",
		"--skip-git-repo-check",
		"--json",
	}
	if workingDir != "" {
		args = append(args, "--cd", workingDir)
	}
	return args
}

func traexPromptForTurn(st *cliSessionState, msg Message) string {
	if st != nil && strings.TrimSpace(st.TransportSessionID) != "" {
		return buildPromptFromHistory([]Message{msg})
	}
	if st == nil {
		return buildPromptFromHistory([]Message{msg})
	}
	return buildPromptFromHistory(st.History)
}

// ============== 上下文导入导出 / Interrupt / Resume / Tools ==============

func (t *TraeXExecutor) ExportContext(ctx context.Context, sessionID string) ([]Message, error) {
	return t.CLIExecutorBase.ExportContext(sessionID)
}
func (t *TraeXExecutor) ImportContext(ctx context.Context, sessionID string, history []Message) error {
	return t.CLIExecutorBase.ImportContext(sessionID, history)
}

func (t *TraeXExecutor) Interrupt(ctx context.Context, sessionID, reason string) error {
	return t.CLIExecutorBase.WrappedInterrupt(ctx, sessionID, reason)
}
func (t *TraeXExecutor) Resume(ctx context.Context, sessionID string) error { return nil }
func (t *TraeXExecutor) GetAvailableTools(ctx context.Context, sessionID string) ([]ToolDef, error) {
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

// ==================== 辅助：TraeX JSONL 解析 ====================

type traexParseResult struct {
	Content     string
	StopReason  string
	ThreadID    string
	UsageIn     int64
	UsageOut    int64
	TotalCost   float64
	ModelUsage  map[string]any
	FileChanges []FileChangeObservation
}

// parseTraeXJSONL 解析 traex exec --json 输出的 JSONL；
// 抽最后一条 assistant / item.completed 的内容，以及 turn.completed 的 usage
func parseTraeXJSONL(raw string) traexParseResult {
	var out traexParseResult
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
		// 不同 TraeX 版本字段名不同，兼容多种格式
		typ, _ := obj["type"].(string)
		switch typ {
		case "thread.started":
			if id, ok := obj["thread_id"].(string); ok && id != "" {
				out.ThreadID = id
			}
		case "item.started":
			out.FileChanges = append(out.FileChanges, traexFileChangeObservations(obj)...)
		case "item.completed", "assistant_message", "message":
			out.FileChanges = append(out.FileChanges, traexFileChangeObservations(obj)...)
			if c, ok := obj["content"].(string); ok && c != "" {
				out.Content = c
			} else if item, ok := obj["item"].(map[string]any); ok {
				// traex 最新版本格式: item={id,type,text,...}
				if c, ok := item["text"].(string); ok && c != "" {
					out.Content = c
				} else if c, ok := item["content"].(string); ok && c != "" {
					out.Content = c
				}
			} else if data, ok := obj["data"].(map[string]any); ok {
				if c, ok := data["content"].(string); ok {
					out.Content = c
				}
			}
		case "turn.completed", "thread.completed":
			if usage, ok := obj["usage"].(map[string]any); ok {
				out.UsageIn = int64FromAny(usage["input_tokens"])
				out.UsageOut = int64FromAny(usage["output_tokens"])
			}
			if mu, ok := obj["modelUsage"]; ok {
				out.ModelUsage, _ = mu.(map[string]any)
			}
			if cost, ok := obj["cost_usd"].(float64); ok {
				out.TotalCost = cost
			}
			if sr, ok := obj["finish_reason"].(string); ok {
				out.StopReason = sr
			}
		}
	}
	// 兜底：如果没解析到 content，把整个 raw 当 content
	if out.Content == "" {
		out.Content = strings.TrimSpace(raw)
	}
	return out
}

func traexFileChangeObservations(obj map[string]any) []FileChangeObservation {
	item, ok := obj["item"].(map[string]any)
	if !ok {
		return nil
	}
	itemType, _ := item["type"].(string)
	if itemType != "file_change" {
		return nil
	}
	status, _ := item["status"].(string)
	changes, _ := item["changes"].([]any)
	out := make([]FileChangeObservation, 0, len(changes))
	for _, raw := range changes {
		change, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		path, _ := change["path"].(string)
		if strings.TrimSpace(path) == "" {
			continue
		}
		kind, _ := change["kind"].(string)
		out = append(out, FileChangeObservation{
			Path:   path,
			Kind:   kind,
			Tool:   "file_change",
			Status: status,
		})
	}
	return out
}
