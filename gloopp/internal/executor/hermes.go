package executor

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

// HermesExecutor 通过本机 hermes CLI（Nous Research Hermes Agent）接入。
// 使用 oneshot 模式（-z）：传入 prompt，stdout 返回纯文本回复。
//
// 参考：https://github.com/NousResearch/hermes-agent
//
//	hermes -z "prompt" -m <model> --provider <provider>
//
// oneshot 模式自动启用 YOLO（自动审批工具），输出只有最终文本，无 JSON 无流式。
type HermesExecutor struct {
	*BaseExecutor
	*CLIExecutorBase
}

type HermesOpt struct {
	ExecID       string
	BinPath      string
	DefaultModel string
	WorkingRoot  string
	AllowedTools []string
}

func NewHermesExecutor(opt HermesOpt) *HermesExecutor {
	if opt.ExecID == "" {
		opt.ExecID = "hermes-cli"
	}
	if opt.DefaultModel == "" {
		opt.DefaultModel = "auto"
	}
	bin := opt.BinPath
	if bin == "" {
		bin = "hermes"
	}
	base := NewBase(opt.ExecID, "Hermes CLI", model.AgentTypeCLI)
	base.capabs = []Capability{CapToolUse, CapCodeExec, CapContextExport}
	base.tier = CapabilityTierB
	base.models = []ModelSpec{
		{ID: opt.DefaultModel, Name: "Hermes 默认模型", Context: 200000},
		{ID: "auto", Name: "auto", Context: 200000},
	}
	base.defModel = opt.DefaultModel
	if len(opt.AllowedTools) == 0 {
		opt.AllowedTools = []string{"Bash", "Read", "Edit", "Write", "Glob", "Grep"}
	}
	cliBase := NewCLIExecutorBase(bin, opt.WorkingRoot, opt.AllowedTools)
	return &HermesExecutor{
		BaseExecutor:    base,
		CLIExecutorBase: cliBase,
	}
}

func (h *HermesExecutor) CreateSession(ctx context.Context, sessionID string, opts SessionConfig) (*SessionHandle, error) {
	h_, _, err := h.CLIExecutorBase.CreateSession(sessionID, opts)
	if err != nil {
		return nil, err
	}
	h.Heartbeat(h.activeCountLocked())
	return h_, nil
}

func (h *HermesExecutor) CloseSession(ctx context.Context, sessionID string) error {
	h.CLIExecutorBase.CloseSession(sessionID)
	h.Heartbeat(h.activeCountLocked())
	return nil
}

func (h *HermesExecutor) GetSessionStatus(ctx context.Context, sessionID string) (interface{}, error) {
	return h.CLIExecutorBase.GetSessionStatus(sessionID)
}

func (h *HermesExecutor) activeCountLocked() int {
	h.Mu.RLock()
	defer h.Mu.RUnlock()
	return len(h.Sessions)
}

func (h *HermesExecutor) SendMessage(ctx context.Context, sessionID string, msg Message, modelName string, stream chan<- StreamChunk) (*ChatResponse, error) {
	st, ok := h.CLIExecutorBase.Get(sessionID)
	if !ok {
		return nil, fmt.Errorf("session 不存在: %s", sessionID)
	}
	start := time.Now()

	h.CLIExecutorBase.AppendHistory(sessionID, msg)

	prompt := buildPromptFromHistory(st.History)
	modelChoice := pick(modelName, h.DefaultModel())

	// hermes oneshot 模式：-z <prompt> 直接输出最终文本到 stdout
	args := []string{"-z", prompt}
	if modelChoice != "" && modelChoice != "auto" && modelChoice != "default" {
		args = append(args, "-m", modelChoice)
	}

	cmd := exec.CommandContext(ctx, h.BinPath, args...)
	cmd.Dir = st.WorkingDir
	if len(st.Env) > 0 {
		cmd.Env = append(cmd.Environ(), st.Env...)
	}

	result, runErr := runCommandWithRuntimeEvents(sessionID, cmd, stream, start, map[string]any{
		"command":    h.BinPath,
		"adapter":    "hermes",
		"model":      modelChoice,
		"workingDir": st.WorkingDir,
	}, func() {
		h.CLIExecutorBase.SetActiveCmd(sessionID, cmd)
	}, func() {
		h.CLIExecutorBase.ClearActiveCmd(sessionID)
	})
	if runErr != nil {
		return nil, runErr
	}
	waitErr := result.WaitErr
	stdoutStr := result.Stdout

	// hermes oneshot 输出纯文本，直接当 content 用
	content := stdoutStr
	var fallbackErr error
	if waitErr != nil {
		extra := waitErr.Error()
		if result.Stderr != "" {
			extra += " | stderr: " + truncate(result.Stderr, 200)
		}
		fallbackErr = fmt.Errorf("%s", extra)
	}
	resp := h.CLIExecutorBase.completeCLIResponse(st, msg, cliCompletion{
		Content:    content,
		StopReason: "end_turn",
	}, start, fallbackErr, stream)
	h.Heartbeat(h.activeCountLocked())
	if fallbackErr != nil {
		return resp, fallbackErr
	}
	return resp, nil
}

func (h *HermesExecutor) ExportContext(ctx context.Context, sessionID string) ([]Message, error) {
	return h.CLIExecutorBase.ExportContext(sessionID)
}
func (h *HermesExecutor) ImportContext(ctx context.Context, sessionID string, history []Message) error {
	return h.CLIExecutorBase.ImportContext(sessionID, history)
}

func (h *HermesExecutor) Interrupt(ctx context.Context, sessionID, reason string) error {
	return h.CLIExecutorBase.WrappedInterrupt(ctx, sessionID, reason)
}
func (h *HermesExecutor) Resume(ctx context.Context, sessionID string) error { return nil }
func (h *HermesExecutor) GetAvailableTools(ctx context.Context, sessionID string) ([]ToolDef, error) {
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
