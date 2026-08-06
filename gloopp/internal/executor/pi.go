package executor

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

type PiExecutor struct {
	*BaseExecutor
	*CLIExecutorBase
}

type PiOpt struct {
	ExecID       string
	BinPath      string
	DefaultModel string
	WorkingRoot  string
	AllowedTools []string
}

func NewPiExecutor(opt PiOpt) *PiExecutor {
	if opt.ExecID == "" {
		opt.ExecID = "pi-cli"
	}
	if opt.DefaultModel == "" {
		opt.DefaultModel = "auto"
	}
	bin := opt.BinPath
	if bin == "" {
		bin = "pi"
	}
	base := NewBase(opt.ExecID, "Pi CLI", model.AgentTypeCLI)
	base.capabs = []Capability{CapToolUse, CapStreaming, CapCodeExec, CapContextExport}
	base.tier = CapabilityTierC
	base.models = []ModelSpec{
		{ID: opt.DefaultModel, Name: "Pi default", Context: 200000},
		{ID: "auto", Name: "Pi auto", Context: 200000},
	}
	base.defModel = opt.DefaultModel
	if len(opt.AllowedTools) == 0 {
		opt.AllowedTools = []string{"read", "bash", "edit", "write", "grep", "find", "ls"}
	}
	return &PiExecutor{
		BaseExecutor:    base,
		CLIExecutorBase: NewCLIExecutorBase(bin, opt.WorkingRoot, opt.AllowedTools),
	}
}

func (p *PiExecutor) CreateSession(ctx context.Context, sessionID string, opts SessionConfig) (*SessionHandle, error) {
	h, _, err := p.CLIExecutorBase.CreateSession(sessionID, opts)
	if err != nil {
		return nil, err
	}
	p.Heartbeat(p.activeCountLocked())
	return h, nil
}

func (p *PiExecutor) CloseSession(ctx context.Context, sessionID string) error {
	p.CLIExecutorBase.CloseSession(sessionID)
	p.Heartbeat(p.activeCountLocked())
	return nil
}

func (p *PiExecutor) GetSessionStatus(ctx context.Context, sessionID string) (interface{}, error) {
	return p.CLIExecutorBase.GetSessionStatus(sessionID)
}

func (p *PiExecutor) activeCountLocked() int {
	p.Mu.RLock()
	defer p.Mu.RUnlock()
	return len(p.Sessions)
}

func (p *PiExecutor) SendMessage(ctx context.Context, sessionID string, msg Message, modelName string, stream chan<- StreamChunk) (*ChatResponse, error) {
	st, ok := p.CLIExecutorBase.Get(sessionID)
	if !ok {
		return nil, fmt.Errorf("session 不存在: %s", sessionID)
	}
	start := time.Now()
	p.CLIExecutorBase.AppendHistory(sessionID, msg)

	promptText := buildPromptFromHistory(st.History)
	modelChoice := pick(modelName, p.DefaultModel())
	args := []string{
		"--mode", "json",
		"--print",
		"--no-session",
		"--system-prompt", st.History[0].Content,
	}
	if modelChoice != "" && modelChoice != "auto" && modelChoice != "default" {
		args = append(args, "--model", modelChoice)
	}
	if len(st.AllowedTools) == 0 {
		args = append(args, "--no-tools")
	} else {
		args = append(args, "--tools", strings.Join(st.AllowedTools, ","))
	}
	args = append(args, promptText)

	cmd := exec.CommandContext(ctx, p.BinPath, args...)
	cmd.Dir = st.WorkingDir
	if len(st.Env) > 0 {
		cmd.Env = append(cmd.Environ(), st.Env...)
	}
	result, runErr := runCommandWithRuntimeEvents(sessionID, cmd, stream, start, map[string]any{
		"command":    p.BinPath,
		"adapter":    "pi",
		"model":      modelChoice,
		"workingDir": st.WorkingDir,
	}, func() {
		p.CLIExecutorBase.SetActiveCmd(sessionID, cmd)
	}, func() {
		p.CLIExecutorBase.ClearActiveCmd(sessionID)
	})
	if runErr != nil {
		return nil, runErr
	}
	waitErr := result.WaitErr
	stdoutStr := result.Stdout

	parsed := parsePiJSON(stdoutStr)
	var fallbackErr error
	if waitErr != nil || parsed.IsError {
		extra := ""
		if waitErr != nil {
			extra = waitErr.Error()
		}
		if extra == "" {
			extra = "pi returned error"
		}
		if result.Stderr != "" {
			extra += " | stderr: " + truncate(result.Stderr, 200)
		}
		if stdoutStr != "" && parsed.Content == "" {
			extra += " | stdout: " + truncate(stdoutStr, 200)
		}
		fallbackErr = fmt.Errorf("%s", extra)
	}
	resp := p.CLIExecutorBase.completeCLIResponse(st, msg, cliCompletion{
		Content:    parsed.Content,
		StopReason: parsed.StopReason,
		UsageIn:    parsed.UsageIn,
		UsageOut:   parsed.UsageOut,
	}, start, fallbackErr, stream)
	p.Heartbeat(p.activeCountLocked())
	if fallbackErr != nil {
		return resp, fallbackErr
	}
	return resp, nil
}

func (p *PiExecutor) ExportContext(ctx context.Context, sessionID string) ([]Message, error) {
	return p.CLIExecutorBase.ExportContext(sessionID)
}

func (p *PiExecutor) ImportContext(ctx context.Context, sessionID string, history []Message) error {
	return p.CLIExecutorBase.ImportContext(sessionID, history)
}

func (p *PiExecutor) Interrupt(ctx context.Context, sessionID, reason string) error {
	return p.CLIExecutorBase.WrappedInterrupt(ctx, sessionID, reason)
}
func (p *PiExecutor) Resume(ctx context.Context, sessionID string) error { return nil }
func (p *PiExecutor) GetAvailableTools(ctx context.Context, sessionID string) ([]ToolDef, error) {
	return []ToolDef{
		{Name: "bash", Description: "执行 shell 命令", Params: []ToolParam{{Name: "command", Type: "string", Required: true}}},
		{Name: "read", Description: "读取文件", Params: []ToolParam{{Name: "path", Type: "string", Required: true}}},
		{Name: "edit", Description: "编辑文件", Params: []ToolParam{{Name: "path", Type: "string", Required: true}}},
		{Name: "write", Description: "写文件", Params: []ToolParam{{Name: "path", Type: "string", Required: true}}},
	}, nil
}
