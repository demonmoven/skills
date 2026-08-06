package executor

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

// ==================== Codex Executor ====================
// 通过本机 codex CLI（OpenAI Codex）真接入。参数格式同 traex exec，但权限控制方式不同。
//
// 单回合用法实测通过：
//
//	echo "prompt" | codex exec --ephemeral --skip-git-repo-check --json \
//	     -C <workingDir> -s danger-full-access --dangerously-bypass-approvals-and-sandbox \
//	     --model auto_model/alwaysday1
//
// 输出 JSONL 与 traex 同构：thread.started → turn.started → item.completed → turn.completed
// item.completed 用 item.text 承载文本，turn.completed 用 usage 计数。
//
// 平台工具通过 CLI syscall 调用：agent 执行 `gloop <command>`，通过信号文件与 orchestrator 交互。

type CodexExecutor struct {
	*BaseExecutor
	*CLIExecutorBase
}

type CodexOpt struct {
	ExecID       string
	OwnerID      string
	DeviceID     string
	BinPath      string
	DefaultModel string
	WorkingRoot  string
	AllowedTools []string
}

func NewCodexExecutor(opt CodexOpt) *CodexExecutor {
	if opt.ExecID == "" {
		opt.ExecID = "codex-cli"
	}
	if opt.DefaultModel == "" {
		opt.DefaultModel = "auto_model/alwaysday1"
	}
	bin := opt.BinPath
	if bin == "" {
		bin = "codex"
	}
	base := NewBase(opt.ExecID, "Codex CLI", model.AgentTypeCLI)
	base.capabs = []Capability{CapToolUse, CapStreaming, CapCodeExec, CapContextExport}
	base.tier = CapabilityTierB
	base.models = []ModelSpec{
		{ID: opt.DefaultModel, Name: "Codex 默认模型", Context: 200000},
		{ID: "auto_model/alwaysday1", Name: "alwaysday1", Context: 200000},
		{ID: "ark/seed-code-0611", Name: "seed-code-0611", Context: 200000},
	}
	base.defModel = opt.DefaultModel
	if len(opt.AllowedTools) == 0 {
		opt.AllowedTools = []string{"Bash", "Read", "Edit", "Write", "Glob", "Grep", "WebFetch", "WebSearch"}
	}
	cliBase := NewCLIExecutorBase(bin, opt.WorkingRoot, opt.AllowedTools)
	return &CodexExecutor{
		BaseExecutor:    base,
		CLIExecutorBase: cliBase,
	}
}

// ============== 会话生命周期（委托给 CLIExecutorBase） ==============

func (c *CodexExecutor) CreateSession(ctx context.Context, sessionID string, opts SessionConfig) (*SessionHandle, error) {
	h, _, err := c.CLIExecutorBase.CreateSession(sessionID, opts)
	if err != nil {
		return nil, err
	}
	c.Heartbeat(c.activeCountLocked())
	return h, nil
}

func (c *CodexExecutor) CloseSession(ctx context.Context, sessionID string) error {
	c.CLIExecutorBase.CloseSession(sessionID)
	c.Heartbeat(c.activeCountLocked())
	return nil
}

func (c *CodexExecutor) GetSessionStatus(ctx context.Context, sessionID string) (interface{}, error) {
	return c.CLIExecutorBase.GetSessionStatus(sessionID)
}

func (c *CodexExecutor) activeCountLocked() int {
	c.Mu.RLock()
	defer c.Mu.RUnlock()
	return len(c.Sessions)
}

// ============== SendMessage ==============

func (c *CodexExecutor) SendMessage(ctx context.Context, sessionID string, msg Message, model string, stream chan<- StreamChunk) (*ChatResponse, error) {
	st, ok := c.CLIExecutorBase.Get(sessionID)
	if !ok {
		return nil, fmt.Errorf("session 不存在: %s", sessionID)
	}
	start := time.Now()

	c.CLIExecutorBase.AppendHistory(sessionID, msg)

	prompt := buildPromptFromHistory(st.History)
	modelChoice := pick(model, c.DefaultModel())

	args := []string{
		"exec",
		"--ephemeral",
		"--skip-git-repo-check",
		"--json",
		"-C", st.WorkingDir,
		"--model", modelChoice,
	}
	args = append(args, "-s", "danger-full-access", "--dangerously-bypass-approvals-and-sandbox")
	_ = st.AllowedTools // Codex 主要通过 sandbox 控制权限，不按工具名级下发

	cmd := exec.CommandContext(ctx, c.BinPath, args...)
	cmd.Dir = st.WorkingDir
	cmd.Stdin = strings.NewReader(prompt)
	if len(st.Env) > 0 {
		cmd.Env = append(cmd.Environ(), st.Env...)
	}

	result, runErr := runCommandWithRuntimeEvents(sessionID, cmd, stream, start, map[string]any{
		"command":    c.BinPath,
		"adapter":    "codex",
		"model":      modelChoice,
		"workingDir": st.WorkingDir,
	}, func() {
		c.CLIExecutorBase.SetActiveCmd(sessionID, cmd)
	}, func() {
		c.CLIExecutorBase.ClearActiveCmd(sessionID)
	})
	if runErr != nil {
		return nil, runErr
	}
	waitErr := result.WaitErr
	stdoutStr := result.Stdout

	// 复用 traex 的 parser（item.completed / turn.completed 格式完全相同）
	parsed := parseTraeXJSONL(stdoutStr)
	// stderr 里常有 login 过期 / token invalid 等 ERROR，但不影响最终文本（有 fallback 会兜底）
	var fallbackErr error
	if waitErr != nil && parsed.Content == "" {
		// 把 stderr 也合并进 originErr，便于排障
		extra := waitErr.Error()
		if result.Stderr != "" {
			extra += " | stderr: " + truncate(result.Stderr, 200)
		}
		fallbackErr = errors.New(extra)
	}
	resp := c.CLIExecutorBase.completeCLIResponse(st, msg, cliCompletion{
		Content:    parsed.Content,
		StopReason: parsed.StopReason,
		UsageIn:    parsed.UsageIn,
		UsageOut:   parsed.UsageOut,
		TotalCost:  parsed.TotalCost,
		ModelUsage: parsed.ModelUsage,
	}, start, fallbackErr, stream)
	c.Heartbeat(c.activeCountLocked())
	if fallbackErr != nil {
		return resp, fallbackErr
	}
	return resp, nil
}

// ============== 上下文导入导出 / Interrupt / Resume / Tools ==============

func (c *CodexExecutor) ExportContext(ctx context.Context, sessionID string) ([]Message, error) {
	return c.CLIExecutorBase.ExportContext(sessionID)
}
func (c *CodexExecutor) ImportContext(ctx context.Context, sessionID string, history []Message) error {
	return c.CLIExecutorBase.ImportContext(sessionID, history)
}

func (c *CodexExecutor) Interrupt(ctx context.Context, sessionID, reason string) error {
	return c.CLIExecutorBase.WrappedInterrupt(ctx, sessionID, reason)
}
func (c *CodexExecutor) Resume(ctx context.Context, sessionID string) error { return nil }
func (c *CodexExecutor) GetAvailableTools(ctx context.Context, sessionID string) ([]ToolDef, error) {
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
