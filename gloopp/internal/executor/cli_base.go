package executor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// cliSessionState 三个 CLI executor 共用的 session 状态。
type cliSessionState struct {
	SessionID          string
	WorkingDir         string
	History            []Message
	CreatedAt          time.Time
	Env                []string
	AllowedTools       []string
	ReadOnly           bool
	TransportSessionID string
	// activeCmd 是当前正在运行的子进程（SendMessage 期间存在），
	// 由 CLIExecutorBase.Mu 保护访问。Interrupt 通过它实现真正的进程中断。
	activeCmd *exec.Cmd
}

// CLIExecutorBase 是各类 CLI agent 的公共骨架，
// 提供会话生命周期 CRUD + 互斥锁 + 共享 bin/root/allowedTools 字段，
// 消除 3*≈70 行的复制粘贴。
type CLIExecutorBase struct {
	BinPath      string
	WorkingRoot  string
	AllowedTools []string
	Sessions     map[string]*cliSessionState
	Mu           sync.RWMutex // 暴露给 embedding struct（因为调用点在 embedder 方法里已经用 mu.xxx）

	// agentEnv 是 agent 静态身份环境（来自 AgentConfig.Env / env_files /
	// DiscoverClaudeEnv，如 CLAUDE_CONFIG_DIR）。所有内嵌本基类的 CLI
	// executor（relay/codex/pi/aiden/traex）共用同一注入点：CreateSession
	// 把 agentEnv 垫在 session 运行时 env 前面，子进程 spawn 时由各 executor
	// 统一的 `cmd.Env = append(cmd.Environ(), st.Env...)` 落地。
	// 用 RWMutex 保护：注册时写一次，每次 CreateSession 读。
	agentEnv []string
}

func NewCLIExecutorBase(binPath, workingRoot string, defaultTools []string) *CLIExecutorBase {
	if workingRoot == "" {
		workingRoot, _ = os.Getwd()
	}
	if len(defaultTools) == 0 {
		defaultTools = []string{"Bash", "Read", "Edit", "Write", "Glob", "Grep", "WebFetch", "WebSearch"}
	}
	return &CLIExecutorBase{
		BinPath:      binPath,
		WorkingRoot:  workingRoot,
		AllowedTools: defaultTools,
		Sessions:     map[string]*cliSessionState{},
	}
}

// CreateSession 共享实现。systemPrompt 是第一个 system message。
// 返回 (*SessionHandle, *cliSessionState, error)——embedder 拿到 state 后如果要扩展可以存在自己的 map；
// 但为了简化迁移，embedder 直接用这个 Sessions map。
// 注意：embedder 对外的 CreateSession 仍需保持 Executor 接口签名（只返回 *SessionHandle, error）。
func (b *CLIExecutorBase) CreateSession(sessionID string, opts SessionConfig) (*SessionHandle, *cliSessionState, error) {
	wd := opts.WorkingDir
	if wd == "" {
		wd = b.WorkingRoot
	}
	if err := os.MkdirAll(wd, 0o755); err != nil {
		return nil, nil, fmt.Errorf("创建工作目录失败: %w", err)
	}
	// 用 questID/sessionID 做内存 map key，保证跨 quest 全局唯一。
	// 文件系统仍用原始 sessionID（quest 内唯一），两者职责分离。
	execKey := sessionKey(opts.QuestID, sessionID)
	b.Mu.Lock()
	defer b.Mu.Unlock()
	// 幂等：若 session 已存在（recovery retry 复用同一 sessionID 时上次
	// 的 defer CloseSession 尚未执行，或上次执行异常残留），先清理旧状态
	// 再重建。避免 "session 已存在" 报错阻塞 retry，也避免 retry 引用已
	// 失效的旧 session 状态。
	if old, ok := b.Sessions[execKey]; ok {
		if old.activeCmd != nil && old.activeCmd.Process != nil {
			_ = old.activeCmd.Process.Kill()
		}
		delete(b.Sessions, execKey)
	}
	st := &cliSessionState{
		SessionID:    execKey,
		WorkingDir:   wd,
		History:      []Message{{Role: "system", Content: opts.SystemPrompt}},
		CreatedAt:    time.Now(),
		Env:          b.mergeSessionEnv(opts.Env),
		AllowedTools: b.effectiveAllowedTools(opts),
		ReadOnly:     opts.ReadOnly,
	}
	b.Sessions[execKey] = st
	return &SessionHandle{SessionID: execKey, CreatedAt: st.CreatedAt}, st, nil
}

// SetAgentEnv 注入 agent 静态身份环境（来自 AgentConfig）。
// 由 executor 注册时调用一次（见 orchestrator.buildExecutorForAgent →
// ResolveCLIExecutorBase）。空切片表示清空，便于动态重配 agent。
func (b *CLIExecutorBase) SetAgentEnv(env []string) {
	b.Mu.Lock()
	defer b.Mu.Unlock()
	b.agentEnv = append([]string(nil), env...)
}

// mergeSessionEnv 把 agent 静态 env 垫在 session 运行时 env 前面，返回最终
// 注入子进程的 env 切片。顺序即优先级：后者覆盖前者（Go exec 对同名 key
// 取最后一个），所以 session 运行时 env（GLOOP_*）可覆盖 agent 静态 env。
// 调用方（CreateSession）已持有 b.Mu 写锁，这里直接读 b.agentEnv，不再加锁。
func (b *CLIExecutorBase) mergeSessionEnv(sessionEnv []string) []string {
	if len(b.agentEnv) == 0 {
		return append([]string(nil), sessionEnv...)
	}
	merged := make([]string, 0, len(b.agentEnv)+len(sessionEnv))
	merged = append(merged, b.agentEnv...)
	merged = append(merged, sessionEnv...)
	return merged
}

// containsString reports whether xs contains s.
func containsString(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

// IsReadOnlySafeTool reports whether a user-facing tool name is safe when the
// phase is marked read-only. Bash remains available because supported Gloop
// agents use it to invoke the local gloop CLI; mutation safety is enforced by
// the gloop subcommands themselves and the mage command allowlist.
func IsReadOnlySafeTool(tool string) bool {
	switch strings.ToLower(strings.TrimSpace(tool)) {
	case "bash", "read", "glob", "grep", "find", "ls", "list":
		return true
	default:
		return false
	}
}

func (b *CLIExecutorBase) effectiveAllowedTools(opts SessionConfig) []string {
	tools := b.AllowedTools
	if len(opts.AllowedTools) > 0 {
		tools = opts.AllowedTools
	}
	tools = append([]string(nil), tools...)
	if !opts.ReadOnly {
		return tools
	}
	out := make([]string, 0, len(tools))
	for _, tool := range tools {
		if IsReadOnlySafeTool(tool) {
			out = append(out, tool)
		}
	}
	return out
}

func (b *CLIExecutorBase) CloseSession(sessionID string) bool {
	b.Mu.Lock()
	_, ok := b.Sessions[sessionID]
	if ok {
		delete(b.Sessions, sessionID)
	}
	b.Mu.Unlock()
	return ok
}

func (b *CLIExecutorBase) Get(sessionID string) (*cliSessionState, bool) {
	b.Mu.RLock()
	defer b.Mu.RUnlock()
	st, ok := b.Sessions[sessionID]
	return st, ok
}

func (b *CLIExecutorBase) SetTransportSessionID(sessionID, transportSessionID string) {
	transportSessionID = strings.TrimSpace(transportSessionID)
	if transportSessionID == "" {
		return
	}
	b.Mu.Lock()
	defer b.Mu.Unlock()
	if st, ok := b.Sessions[sessionID]; ok {
		st.TransportSessionID = transportSessionID
	}
}

func (b *CLIExecutorBase) AppendHistory(sessionID string, msg Message) {
	b.Mu.Lock()
	defer b.Mu.Unlock()
	if st, ok := b.Sessions[sessionID]; ok {
		st.History = append(st.History, msg)
	}
}

func (b *CLIExecutorBase) GetSessionStatus(sessionID string) (interface{}, error) {
	st, ok := b.Get(sessionID)
	if !ok {
		return nil, fmt.Errorf("session 不存在: %s", sessionID)
	}
	return map[string]interface{}{
		"turns":   len(st.History),
		"working": st.WorkingDir,
		"created": st.CreatedAt,
		"session": sessionID,
	}, nil
}

func (b *CLIExecutorBase) ExportContext(sessionID string) ([]Message, error) {
	st, ok := b.Get(sessionID)
	if !ok {
		return nil, fmt.Errorf("session 不存在: %s", sessionID)
	}
	out := make([]Message, len(st.History))
	copy(out, st.History)
	return out, nil
}

func (b *CLIExecutorBase) ImportContext(sessionID string, history []Message) error {
	b.Mu.Lock()
	defer b.Mu.Unlock()
	st, ok := b.Sessions[sessionID]
	if !ok {
		return errors.New("session 不存在")
	}
	st.History = append([]Message{}, history...)
	return nil
}

// FallbackToMock is retained only for explicit debug/test adapters. Production
// executors should return their original error so the orchestrator can block
// with clear recovery actions instead of pretending a real agent completed work.
func (b *CLIExecutorBase) FallbackToMock(st *cliSessionState, userMsg Message, originErr error, stream chan<- StreamChunk) *ChatResponse {
	content := MockDraftResponse(userMsg.Content, originErr)
	// 按行拆，流式推送（视觉效果 + 安全发送）
	if stream != nil {
		for _, line := range strings.SplitAfter(content, "\n") {
			if line == "" {
				continue
			}
			SafeStream(stream, StreamChunk{SessionID: st.SessionID, Delta: line})
		}
	}
	resp := &ChatResponse{
		SessionID:    st.SessionID,
		Message:      Message{Role: "assistant", Content: content},
		TokenInput:   int64(approxTokens(userMsg.Content)),
		TokenOutput:  int64(approxTokens(content)),
		FinishReason: detectFinishReason(content),
	}
	b.AppendHistory(st.SessionID, resp.Message)
	if originErr != nil {
		resp.Error = originErr.Error()
		resp.Meta = map[string]any{"fallback": true, "origin_error": originErr.Error()}
	}
	return resp
}

// ==================== 给 embedder 包一层接口兼容的 method ====================
// （为了让 embedder 无需逐个写 boilerplate）

// WrappedCreateSession 保持 Executor 接口签名：丢弃 state，返回 (*SessionHandle, error)。
func (b *CLIExecutorBase) WrappedCreateSession(_ context.Context, sessionID string, opts SessionConfig) (*SessionHandle, error) {
	h, _, err := b.CreateSession(sessionID, opts)
	return h, err
}

// WrappedCloseSession 保持 Executor 接口签名。
func (b *CLIExecutorBase) WrappedCloseSession(_ context.Context, sessionID string) error {
	b.CloseSession(sessionID)
	return nil
}

// WrappedGetSessionStatus 保持 Executor 接口签名。
func (b *CLIExecutorBase) WrappedGetSessionStatus(_ context.Context, sessionID string) (interface{}, error) {
	return b.GetSessionStatus(sessionID)
}

// WrappedExportContext 保持 Executor 接口签名。
func (b *CLIExecutorBase) WrappedExportContext(_ context.Context, sessionID string) ([]Message, error) {
	return b.ExportContext(sessionID)
}

// WrappedImportContext 保持 Executor 接口签名。
func (b *CLIExecutorBase) WrappedImportContext(_ context.Context, sessionID string, history []Message) error {
	return b.ImportContext(sessionID, history)
}

// ==================== 活动进程追踪 + 真实中断 ====================

// SetActiveCmd 注册当前 session 正在运行的子进程。
// SendMessage 开始时调用，结束时调用 ClearActiveCmd。
func (b *CLIExecutorBase) SetActiveCmd(sessionID string, cmd *exec.Cmd) {
	b.Mu.Lock()
	defer b.Mu.Unlock()
	if st, ok := b.Sessions[sessionID]; ok {
		st.activeCmd = cmd
	}
}

// ClearActiveCmd 清除活动进程记录。SendMessage 结束时调用。
func (b *CLIExecutorBase) ClearActiveCmd(sessionID string) {
	b.Mu.Lock()
	defer b.Mu.Unlock()
	if st, ok := b.Sessions[sessionID]; ok {
		st.activeCmd = nil
	}
}

// WrappedInterrupt 实现 Executor 接口的 Interrupt 方法。
// 找到当前 session 的活动进程并 kill，使 SendMessage 提前返回。
func (b *CLIExecutorBase) WrappedInterrupt(_ context.Context, sessionID string, reason string) error {
	b.Mu.RLock()
	var cmd *exec.Cmd
	if st, ok := b.Sessions[sessionID]; ok {
		cmd = st.activeCmd
	}
	b.Mu.RUnlock()

	if cmd == nil || cmd.Process == nil {
		// 没有在运行的进程，不算错误
		return nil
	}
	// 直接 kill 进程。exec.CommandContext 本身也会通过 context 取消触发 kill，
	// 这里提供一条独立的中断路径，方便从外部 goroutine 触发。
	return cmd.Process.Kill()
}
