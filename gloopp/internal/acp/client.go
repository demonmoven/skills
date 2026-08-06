package acp

// ACP (Agent Client Protocol) 客户端
// JSON-RPC 2.0 over stdio
//
// 核心方法：
//   - initialize           握手
//   - session/new          创建会话
//   - session/prompt       发送消息（流式通过 session/update notification）
//   - session/close        关闭会话
//
// 流式更新通过 session/update notification 推送：
//   - agent_message_chunk  消息增量
//   - usage_update        用量更新
//   - available_commands_update  可用命令更新

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

const defaultControlRequestTimeout = 60 * time.Second
const maxTransportErrorDetailRunes = 2048

// resolveControlTimeout 从环境变量 GLOOP_ACP_CONTROL_TIMEOUT 读取控制面超时，
// 解析失败或未设置时退回 defaultControlRequestTimeout。
// 支持 Go duration 语法（如 "90s", "2m"）。
func resolveControlTimeout() time.Duration {
	if v := os.Getenv("GLOOP_ACP_CONTROL_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
	}
	return defaultControlRequestTimeout
}

// ==================== JSON-RPC 基础结构 ====================

type jsonrpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
	ID      any    `json:"id,omitempty"` // int 或 string；notification 无 id
}

type jsonrpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonrpcError   `json:"error,omitempty"`
	ID      any             `json:"id"`
}

type jsonrpcMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonrpcError   `json:"error,omitempty"`
	ID      any             `json:"id,omitempty"`
}

type jsonrpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func (e *jsonrpcError) Error() string {
	if e.Data != nil {
		if raw, err := json.Marshal(e.Data); err == nil && len(raw) > 0 && string(raw) != "null" {
			return fmt.Sprintf("JSON-RPC error %d: %s data=%s", e.Code, e.Message, raw)
		}
	}
	return fmt.Sprintf("JSON-RPC error %d: %s", e.Code, e.Message)
}

// ==================== Content Part ====================

// ContentPart 是 ACP prompt 的内容块
type ContentPart struct {
	Type string `json:"type"` // text / image / audio / resource_link / resource
	Text string `json:"text,omitempty"`
	// 其他类型字段按需扩展
}

// ==================== Session Update ====================

// SessionUpdate 是 session/update notification 的 payload
type SessionUpdate struct {
	SessionID string        `json:"sessionId"`
	Update    UpdatePayload `json:"update"`
}

// UpdatePayload 使用 RawMessage 延迟解析不同类型的 update
type UpdatePayload struct {
	SessionUpdate     string          `json:"sessionUpdate"`
	Content           *ContentPart    `json:"content,omitempty"` // agent_message_chunk
	AvailableCommands []CommandInfo   `json:"availableCommands,omitempty"`
	Usage             *UsageInfo      `json:"usage,omitempty"` // usage_update
	ToolCallID        string          `json:"toolCallId,omitempty"`
	Title             string          `json:"title,omitempty"`
	Kind              string          `json:"kind,omitempty"`
	Status            string          `json:"status,omitempty"`
	StopReason        string          `json:"stopReason,omitempty"`
	RawInput          json.RawMessage `json:"rawInput,omitempty"`
	RawOutput         json.RawMessage `json:"rawOutput,omitempty"`
	Result            json.RawMessage `json:"result,omitempty"`
	PartialResult     json.RawMessage `json:"partialResult,omitempty"`
	// 其他类型按需扩展
	Raw json.RawMessage `json:"-"`
}

// UsageInfo 是 usage_update notification 的 token 用量。
type UsageInfo struct {
	InputTokens  int64 `json:"inputTokens"`
	OutputTokens int64 `json:"outputTokens"`
}

// 自定义 Unmarshal：先拿 sessionUpdate 字段，再根据类型解析
func (u *UpdatePayload) UnmarshalJSON(data []byte) error {
	type alias UpdatePayload
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*u = UpdatePayload(a)
	u.Raw = data
	return nil
}

type CommandInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Input       struct {
		Hint string `json:"hint"`
	} `json:"input,omitempty"`
}

// ==================== Client ====================

// Client 是 ACP 协议客户端，管理一个 agent 子进程
type Client struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Scanner
	stderr io.ReadCloser

	reqID   atomic.Int64
	pending map[int64]chan *jsonrpcResponse
	mu      sync.Mutex
	writeMu sync.Mutex

	notifyMu              sync.RWMutex
	sessionHandlers       map[string]func(*SessionUpdate)
	defaultHandler        func(*SessionUpdate)
	stderrMu              sync.Mutex
	stderrBuf             bytes.Buffer
	controlRequestTimeout time.Duration

	// 生命周期
	closed bool
	wg     sync.WaitGroup
}

// NewClient 启动一个 ACP agent 子进程并返回客户端
func NewClient(ctx context.Context, command string, args []string, env map[string]string) (*Client, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	if len(env) > 0 {
		cmd.Env = append(cmd.Environ(), envToPairs(env)...)
	}
	// 进程组绑定：daemon 死时 agent 子进程一起死（linux 靠 Pdeathsig，其他平台靠 Setpgid+kill 组）
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	setPlatformChildAttr(cmd.SysProcAttr)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("创建 stdin pipe 失败: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return nil, fmt.Errorf("创建 stdout pipe 失败: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		stdin.Close()
		stdout.Close()
		return nil, fmt.Errorf("创建 stderr pipe 失败: %w", err)
	}

	if err := cmd.Start(); err != nil {
		stdin.Close()
		stdout.Close()
		stderr.Close()
		return nil, fmt.Errorf("启动 agent 进程失败: %w", err)
	}
	setChildMemoryLimit(cmd.Process)

	c := &Client{
		cmd:                   cmd,
		stdin:                 stdin,
		stdout:                bufio.NewScanner(stdout),
		stderr:                stderr,
		pending:               make(map[int64]chan *jsonrpcResponse),
		sessionHandlers:       make(map[string]func(*SessionUpdate)),
		controlRequestTimeout: resolveControlTimeout(),
	}
	c.stdout.Buffer(make([]byte, 0, 64*1024), 8*1024*1024) // 最大 8MB 行

	c.wg.Add(1)
	go c.readLoop()

	go c.captureStderr(stderr)

	return c, nil
}

func (c *Client) captureStderr(r io.Reader) {
	if r == nil {
		return
	}
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 16*1024), 512*1024)
	for scanner.Scan() {
		c.appendStderrLine(scanner.Text())
	}
}

func (c *Client) appendStderrLine(line string) {
	const maxStderrSnapshot = 16 * 1024
	c.stderrMu.Lock()
	defer c.stderrMu.Unlock()
	c.stderrBuf.WriteString(line)
	c.stderrBuf.WriteByte('\n')
	if c.stderrBuf.Len() <= maxStderrSnapshot {
		return
	}
	raw := c.stderrBuf.Bytes()
	keep := append([]byte(nil), raw[len(raw)-maxStderrSnapshot:]...)
	c.stderrBuf.Reset()
	c.stderrBuf.Write(keep)
}

// StderrSnapshot returns a bounded tail of the agent stderr stream.
func (c *Client) StderrSnapshot() string {
	c.stderrMu.Lock()
	defer c.stderrMu.Unlock()
	return c.stderrBuf.String()
}

func envToPairs(env map[string]string) []string {
	pairs := make([]string, 0, len(env))
	for k, v := range env {
		pairs = append(pairs, k+"="+v)
	}
	return pairs
}

// readLoop 持续读取 stdout，分发 response 和 notification
func (c *Client) readLoop() {
	defer c.wg.Done()
	defer c.failPendingRequests("agent stdout closed, process likely exited")

	for c.stdout.Scan() {
		line := c.stdout.Bytes()
		if len(line) == 0 {
			continue
		}

		var msg jsonrpcMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			continue
		}

		if msg.Method != "" {
			if msg.ID != nil {
				c.handleRequest(msg.Method, msg.Params, msg.ID)
			} else {
				c.handleNotification(msg.Method, msg.Params)
			}
			continue
		}

		if msg.ID != nil {
			var idInt int64
			switch id := msg.ID.(type) {
			case float64:
				idInt = int64(id)
			case int64:
				idInt = id
			case int:
				idInt = int64(id)
			default:
				continue
			}

			c.mu.Lock()
			ch, ok := c.pending[idInt]
			delete(c.pending, idInt)
			c.mu.Unlock()

			if ok {
				ch <- &jsonrpcResponse{
					JSONRPC: msg.JSONRPC,
					Result:  msg.Result,
					Error:   msg.Error,
					ID:      msg.ID,
				}
				close(ch)
			}
		}
	}
}

// failPendingRequests 在 agent 进程退出/连接断开时，让所有等待中的请求立即返回错误，避免永久阻塞。
func (c *Client) failPendingRequests(reason string) {
	c.mu.Lock()
	pending := c.pending
	c.pending = make(map[int64]chan *jsonrpcResponse)
	c.mu.Unlock()

	stderrTail := c.StderrSnapshot()
	errMsg := reason
	if stderrTail != "" {
		errMsg = fmt.Sprintf("%s; stderr tail: %s", reason, strings.TrimSpace(stderrTail))
	}
	procErr := &jsonrpcError{
		Code:    -32000,
		Message: errMsg,
	}

	for _, ch := range pending {
		ch <- &jsonrpcResponse{
			JSONRPC: "2.0",
			Error:   procErr,
		}
		close(ch)
	}
}

func (c *Client) handleRequest(method string, params json.RawMessage, id any) {
	switch method {
	case "session/request_permission":
		result := map[string]any{
			"outcome": map[string]any{
				"outcome":  "selected",
				"optionId": selectPermissionOption(params),
			},
		}
		_ = c.respond(id, result, nil)
	default:
		_ = c.respond(id, nil, &jsonrpcError{Code: -32601, Message: "method not found"})
	}
}

func selectPermissionOption(params json.RawMessage) string {
	var req struct {
		Options []struct {
			OptionID string `json:"optionId"`
			Kind     string `json:"kind"`
		} `json:"options"`
	}
	if err := json.Unmarshal(params, &req); err == nil {
		for _, opt := range req.Options {
			if opt.Kind == "allow_once" && opt.OptionID != "" {
				return opt.OptionID
			}
		}
		for _, opt := range req.Options {
			if opt.OptionID != "" {
				return opt.OptionID
			}
		}
	}
	return "approved"
}

func (c *Client) handleNotification(method string, params json.RawMessage) {
	switch method {
	case "session/update":
		var update SessionUpdate
		if err := json.Unmarshal(params, &update); err != nil {
			return
		}
		if handler := c.sessionUpdateHandler(update.SessionID); handler != nil {
			handler(&update)
		}
		// 其他 notification 类型按需扩展
	}
}

func (c *Client) SetSessionUpdateHandler(sessionID string, handler func(*SessionUpdate)) {
	c.notifyMu.Lock()
	defer c.notifyMu.Unlock()
	if sessionID == "" {
		c.defaultHandler = handler
		return
	}
	if handler == nil {
		delete(c.sessionHandlers, sessionID)
		return
	}
	c.sessionHandlers[sessionID] = handler
}

func (c *Client) ClearSessionUpdateHandler(sessionID string) {
	c.SetSessionUpdateHandler(sessionID, nil)
}

func (c *Client) sessionUpdateHandler(sessionID string) func(*SessionUpdate) {
	c.notifyMu.RLock()
	defer c.notifyMu.RUnlock()
	if sessionID != "" {
		if handler := c.sessionHandlers[sessionID]; handler != nil {
			return handler
		}
	}
	return c.defaultHandler
}

func (c *Client) respond(id any, result any, rpcErr *jsonrpcError) error {
	resp := jsonrpcRequest{
		JSONRPC: "2.0",
		ID:      id,
	}
	if rpcErr != nil {
		rawErr := map[string]any{
			"code":    rpcErr.Code,
			"message": rpcErr.Message,
		}
		if rpcErr.Data != nil {
			rawErr["data"] = rpcErr.Data
		}
		resp.Params = nil
		b, err := json.Marshal(map[string]any{
			"jsonrpc": "2.0",
			"id":      id,
			"error":   rawErr,
		})
		if err != nil {
			return err
		}
		return c.writeLine(b)
	}
	b, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  result,
	})
	if err != nil {
		return err
	}
	return c.writeLine(b)
}

func (c *Client) writeLine(b []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	b = append(b, '\n')
	_, err := c.stdin.Write(b)
	if err != nil {
		return c.withTransportErrorDetail(err)
	}
	return nil
}

func (c *Client) withTransportErrorDetail(err error) error {
	stderrTail := c.compactStderrSnapshotForError()
	if stderrTail == "" {
		return err
	}
	return fmt.Errorf("%w; stderr tail: %s", err, stderrTail)
}

func (c *Client) compactStderrSnapshotForError() string {
	stderrTail := compactTransportErrorDetail(c.StderrSnapshot())
	if stderrTail != "" {
		return stderrTail
	}
	// 子进程启动即退时，stdin 写入可能先于 stderr goroutine 捕获尾部日志失败。
	// 给 stderr reader 一个很短的调度窗口，避免把真实启动错误压成裸 broken pipe。
	time.Sleep(20 * time.Millisecond)
	return compactTransportErrorDetail(c.StderrSnapshot())
}

func compactTransportErrorDetail(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	runes := []rune(raw)
	if len(runes) <= maxTransportErrorDetailRunes {
		return raw
	}
	return "..." + string(runes[len(runes)-maxTransportErrorDetailRunes:])
}

// call 发起一个 JSON-RPC 请求，等待响应。
//
// 控制面调用（initialize / session/new）应传 applyControlTimeout=true，
// 让 client 在 controlRequestTimeout 内强制返回，避免 agent 启动期挂起；
// 数据面调用（session/prompt）应传 false，执行时长完全交给调用方 ctx 控制。
func (c *Client) call(ctx context.Context, method string, params any, result any, applyControlTimeout bool) error {
	if c.closed {
		return errors.New("client closed")
	}
	callCtx := ctx
	var cancel context.CancelFunc
	var timeout time.Duration
	if applyControlTimeout {
		timeout = c.effectiveControlTimeout()
		if timeout > 0 {
			callCtx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
		}
	}

	id := c.reqID.Add(1)
	ch := make(chan *jsonrpcResponse, 1)

	c.mu.Lock()
	c.pending[id] = ch
	c.mu.Unlock()

	req := jsonrpcRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      id,
	}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return fmt.Errorf("序列化请求失败: %w", err)
	}
	reqBytes = append(reqBytes, '\n')

	if err := c.writeLine(reqBytes[:len(reqBytes)-1]); err != nil {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return fmt.Errorf("写入请求失败: %w", err)
	}

	select {
	case <-callCtx.Done():
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("%s timed out after %s: %w", method, timeout, callCtx.Err())
	case resp := <-ch:
		if resp.Error != nil {
			return resp.Error
		}
		if result != nil && len(resp.Result) > 0 {
			return json.Unmarshal(resp.Result, result)
		}
		return nil
	}
}

func (c *Client) effectiveControlTimeout() time.Duration {
	if c.controlRequestTimeout == 0 {
		return defaultControlRequestTimeout
	}
	return c.controlRequestTimeout
}

// ==================== ACP 方法封装 ====================

// InitializeResult 是 initialize 的返回
type InitializeResult struct {
	ProtocolVersion   int            `json:"protocolVersion"`
	AgentCapabilities map[string]any `json:"agentCapabilities"`
	AuthMethods       []AuthMethod   `json:"authMethods,omitempty"`
	AgentInfo         AgentInfo      `json:"agentInfo"`
}

type AuthMethod struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Type        string    `json:"type,omitempty"`
	Vars        []AuthVar `json:"vars,omitempty"`
}

type AuthVar struct {
	Name string `json:"name"`
}

type AgentInfo struct {
	Name    string `json:"name"`
	Title   string `json:"title,omitempty"`
	Version string `json:"version,omitempty"`
}

// Initialize 握手
func (c *Client) Initialize(ctx context.Context, clientName, clientVersion string) (*InitializeResult, error) {
	var result InitializeResult
	err := c.call(ctx, "initialize", map[string]any{
		"protocolVersion": 1,
		"capabilities":    map[string]any{},
		"clientInfo": map[string]string{
			"name":    clientName,
			"version": clientVersion,
		},
	}, &result, true)
	return &result, err
}

// NewSessionResult 是 session/new 的返回
type NewSessionResult struct {
	SessionID string        `json:"sessionId"`
	Modes     SessionModes  `json:"modes"`
	Models    SessionModels `json:"models"`
	// 其他字段按需添加
}

type SessionModes struct {
	CurrentModeID  string          `json:"currentModeId"`
	AvailableModes []AvailableMode `json:"availableModes"`
}

type AvailableMode struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type SessionModels struct {
	CurrentModelID  string           `json:"currentModelId"`
	AvailableModels []AvailableModel `json:"availableModels,omitempty"`
}

type AvailableModel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// NewSession 创建新会话
func (c *Client) NewSession(ctx context.Context, cwd string, mcpServers []any, mode string, model string) (*NewSessionResult, error) {
	params := newSessionParams(cwd, mcpServers, mode, model)
	var result NewSessionResult
	err := c.call(ctx, "session/new", params, &result, true)
	return &result, err
}

func newSessionParams(cwd string, mcpServers []any, mode string, model string) map[string]any {
	params := map[string]any{
		"cwd":        cwd,
		"mcpServers": mcpServers,
	}
	if mode != "" {
		params["mode"] = mode
	}
	if isConcreteModel(model) {
		params["model"] = model
	}
	return params
}

// PromptResult 是 session/prompt 的返回
type PromptResult struct {
	StopReason string `json:"stopReason"`
}

// Prompt 发送消息，结果通过 session/update notification 流式推送。
// 调用方可通过 SetSessionUpdateHandler 订阅指定 session 的更新。
//
// 注意：prompt 是数据面调用，agent 真实执行时长可能很长（分钟级），
// 因此不施加 client 层控制面超时；执行时长完全由调用方传入的 ctx 决定，
// 进程死亡/连接断开则由 readLoop 退出时的 failPendingRequests 兜底。
func (c *Client) Prompt(ctx context.Context, sessionID string, parts []ContentPart, model string) (*PromptResult, error) {
	params := promptParams(sessionID, parts, model)
	var result PromptResult
	err := c.call(ctx, "session/prompt", params, &result, false)
	return &result, err
}

func promptParams(sessionID string, parts []ContentPart, model string) map[string]any {
	params := map[string]any{
		"sessionId": sessionID,
		"prompt":    parts,
	}
	if isConcreteModel(model) {
		params["model"] = model
	}
	return params
}

func isConcreteModel(model string) bool {
	switch strings.ToLower(strings.TrimSpace(model)) {
	case "", "auto", "default":
		return false
	default:
		return true
	}
}

// Cancel 发送 session/cancel notification，请求 agent 中断当前 prompt 处理。
// ACP 协议中 cancel 是 notification（无响应），agent 会以 stopReason=cancelled 结束当前 prompt。
func (c *Client) Cancel(sessionID string) error {
	b, err := json.Marshal(jsonrpcRequest{
		JSONRPC: "2.0",
		Method:  "session/cancel",
		Params:  map[string]any{"sessionId": sessionID},
	})
	if err != nil {
		return err
	}
	return c.writeLine(b)
}

// Close 关闭客户端和 agent 进程
func (c *Client) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	c.mu.Unlock()

	// 1) 先关闭 stdin：给 agent 发送优雅退出信号
	if c.stdin != nil {
		_ = c.stdin.Close()
	}

	// 2) 并行等待：readLoop 退出 + 子进程 Wait，整体 5 秒超时
	//    （旧实现把 c.wg.Wait() 放在超时 select 之前，agent 挂起时会永久阻塞）
	waitDone := make(chan struct{})
	var cmdWaitOnce sync.Once
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if c.cmd != nil {
			cmdWaitOnce.Do(func() { _ = c.cmd.Wait() })
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		c.wg.Wait() // readLoop
	}()
	go func() {
		wg.Wait()
		close(waitDone)
	}()

	select {
	case <-waitDone:
		// 正常关闭
	case <-time.After(5 * time.Second):
		// 超时：强制 kill 整个进程组（agent 可能 spawn 了孙进程），
		// 然后确保 Wait 被调用（只一次，避免与上面 goroutine 数据竞争）
		if c.cmd != nil && c.cmd.Process != nil {
			killProcessGroup(c.cmd.Process)
		}
		// cmd.Wait 由上面 goroutine 调用（sync.Once 保证单次），再等 readLoop wg
		cmdWaitOnce.Do(func() {
			if c.cmd != nil {
				_ = c.cmd.Wait()
			}
		})
		c.wg.Wait()
	}

	// 3) 关闭剩余 pipes（cmd.Wait() 已关 stdout pipe，但显式关 stderr 更稳妥）
	if c.stdin != nil {
		_ = c.stdin.Close()
	}
	if c.stderr != nil {
		_ = c.stderr.Close()
	}

	return nil
}

// PID 返回 agent 进程的 PID
func (c *Client) PID() int {
	if c.cmd == nil || c.cmd.Process == nil {
		return 0
	}
	return c.cmd.Process.Pid
}
