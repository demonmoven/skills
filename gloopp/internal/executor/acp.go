package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"code.byted.org/lihuanyu.0w0/gloop/internal/acp"
	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

// ==================== ACP Executor ====================
// 通过 ACP（Agent Client Protocol）接入的 Agent 执行器
//
// 每个 session 对应一个 ACP 会话（session/new 创建）
// 消息通过 session/prompt 发送，流式内容通过 session/update notification 推送

type ACPExecutor struct {
	*BaseExecutor

	command            string
	args               []string
	env                map[string]string
	workingRoot        string
	startupModelEnvVar string

	// fullHistoryFallback=true 时每轮全量重发 history（旧行为）。
	// 默认 false（增量）——ACP 有状态协议下 agent 侧维护历史，
	// 只发本轮新增即可。仅当目标 agent 实现不符合协议（无状态）
	// 导致增量模式丢上下文时开启。
	fullHistoryFallback bool

	sessions map[string]*acpSessionState
	mu       sync.RWMutex
}

type acpSessionState struct {
	SessionID    string // Gloop 的 sessionID
	ACPSessionID string // ACP agent 实际的 sessionID
	WorkingDir   string
	Model        string
	History      []Message
	CreatedAt    time.Time
	client       *acp.Client
	env          []string
	sendMu       sync.Mutex
}

type ACPOpt struct {
	ExecID             string
	OwnerID            string
	DeviceID           string
	Command            string
	Args               []string
	Env                map[string]string
	DefaultModel       string
	WorkingRoot        string
	StartupModelEnvVar string
	Name               string
	// FullHistoryFallback=true 时每轮全量重发 history（旧行为）。
	// 默认 false（增量）。仅当目标 ACP agent 实现无状态、增量模式丢上下文时开启。
	FullHistoryFallback bool
}

func NewACPExecutor(opt ACPOpt) *ACPExecutor {
	if opt.ExecID == "" {
		opt.ExecID = "acp-" + opt.Command
	}
	if opt.Name == "" {
		opt.Name = "ACP Agent (" + opt.Command + ")"
	}
	if opt.DefaultModel == "" {
		opt.DefaultModel = "auto"
	}
	if opt.WorkingRoot == "" {
		opt.WorkingRoot = "."
	}

	base := NewBase(opt.ExecID, opt.Name, model.AgentTypeACP)
	base.capabs = []Capability{CapToolUse, CapStreaming, CapCodeExec, CapContextExport}
	base.tier = CapabilityTierB
	base.models = []ModelSpec{
		{ID: "auto", Name: "Auto", Context: 200000},
	}
	base.defModel = opt.DefaultModel

	return &ACPExecutor{
		BaseExecutor:        base,
		command:             opt.Command,
		args:                opt.Args,
		env:                 opt.Env,
		workingRoot:         opt.WorkingRoot,
		startupModelEnvVar:  opt.StartupModelEnvVar,
		fullHistoryFallback: opt.FullHistoryFallback,
		sessions:            map[string]*acpSessionState{},
	}
}

// ============== 会话生命周期 ==============

func (a *ACPExecutor) CreateSession(ctx context.Context, sessionID string, opts SessionConfig) (*SessionHandle, error) {
	wd := opts.WorkingDir
	if wd == "" {
		wd = a.resolveWorkingDir()
	}
	env := mergeEnvMap(a.env, opts.Env)

	// 确定使用的模型：优先用 opts.Model，否则用 executor 默认模型
	sessModel := opts.Model
	if sessModel == "" {
		sessModel = a.DefaultModel()
	}
	if sessModel == "" {
		sessModel = "auto"
	}
	env = withStartupModelEnv(env, a.startupModelEnvVar, sessModel)

	// 启动 ACP client + Initialize 握手（带重试）
	// npx 冷启动首次可能超时（进程启动 + 模块加载），第二次通常快很多。
	const maxInitAttempts = 3
	var client *acp.Client
	var lastErr error
	for attempt := 0; attempt < maxInitAttempts; attempt++ {
		if attempt > 0 {
			select {
			case <-time.After(time.Duration(attempt) * time.Second):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		client, lastErr = acp.NewClient(ctx, a.command, a.args, env)
		if lastErr != nil {
			continue
		}
		_, lastErr = client.Initialize(ctx, "gloop", "v2")
		if lastErr == nil {
			break
		}
		client.Close()
		client = nil
	}
	if client == nil {
		if lastErr != nil {
			return nil, fmt.Errorf("ACP 启动失败（重试 %d 次）: %w", maxInitAttempts, lastErr)
		}
		return nil, fmt.Errorf("ACP 启动失败（重试 %d 次）", maxInitAttempts)
	}

	// 创建会话
	mode := ""
	// 可以通过 opts 传 mode，MVP 先用默认
	nsr, err := client.NewSession(ctx, wd, []any{}, mode, sessModel)
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("ACP session/new 失败: %w", err)
	}
	if len(nsr.Models.AvailableModels) > 0 {
		models := make([]ModelSpec, 0, len(nsr.Models.AvailableModels))
		for _, m := range nsr.Models.AvailableModels {
			if strings.TrimSpace(m.ID) == "" {
				continue
			}
			name := m.Name
			if name == "" {
				name = m.ID
			}
			models = append(models, ModelSpec{ID: m.ID, Name: name, Context: 200000})
		}
		a.SetAvailableModels(models)
	}

	st := &acpSessionState{
		SessionID:    sessionKey(opts.QuestID, sessionID),
		ACPSessionID: nsr.SessionID,
		WorkingDir:   wd,
		Model:        sessModel,
		History:      []Message{{Role: "system", Content: opts.SystemPrompt}},
		CreatedAt:    time.Now(),
		client:       client,
		env:          append([]string(nil), opts.Env...),
	}

	a.mu.Lock()
	a.sessions[st.SessionID] = st
	a.mu.Unlock()
	a.Heartbeat(len(a.sessions))

	// 实际 session 用的是 acp 返回的 sessionId，但我们用自己的 sessionID 映射
	// 这里把 acp 的 sessionId 存到 state 里
	_ = nsr.SessionID // 暂时不用，我们的 sessionID 就是 Gloop 的

	return &SessionHandle{SessionID: st.SessionID, CreatedAt: st.CreatedAt}, nil
}

func (a *ACPExecutor) CloseSession(ctx context.Context, sessionID string) error {
	a.mu.Lock()
	st, ok := a.sessions[sessionID]
	if ok {
		delete(a.sessions, sessionID)
	}
	a.mu.Unlock()

	if ok && st.client != nil {
		st.client.Close()
	}

	a.Heartbeat(len(a.sessions))
	return nil
}

func (a *ACPExecutor) GetSessionStatus(ctx context.Context, sessionID string) (interface{}, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	st, ok := a.sessions[sessionID]
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

// ============== SendMessage ==============

func (a *ACPExecutor) SendMessage(ctx context.Context, sessionID string, msg Message, model string, stream chan<- StreamChunk) (*ChatResponse, error) {
	a.mu.Lock()
	st, ok := a.sessions[sessionID]
	a.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("session 不存在: %s", sessionID)
	}
	st.sendMu.Lock()
	defer st.sendMu.Unlock()

	start := time.Now()

	a.mu.Lock()
	st.History = append(st.History, msg)
	a.mu.Unlock()

	// 确定本次调用使用的模型：优先使用传入的 model，否则使用会话默认模型
	useModel := model
	if useModel == "" {
		useModel = st.Model
	}
	if useModel == "" {
		useModel = a.DefaultModel()
	}

	promptParts := a.buildPromptParts(st, msg)
	fullPrompt := contentPartsText(promptParts)

	// 设置流式回调
	var fullText string
	var toolCalls []ToolCall
	var usageIn, usageOut int64
	var hasUsage bool
	var firstOutput sync.Once
	var mu sync.Mutex

	// 不管要不要 stream 都要监听 notification 收集结果
	st.client.SetSessionUpdateHandler(st.ACPSessionID, func(update *acp.SessionUpdate) {
		mu.Lock()
		defer mu.Unlock()

		upd := update.Update
		switch upd.SessionUpdate {
		case "agent_message_chunk":
			if upd.Content != nil && upd.Content.Type == "text" {
				fullText += upd.Content.Text
				if upd.Content.Text != "" {
					firstOutput.Do(func() {
						streamRuntimeEvent(sessionID, stream, RuntimeEventOutputFirst, map[string]any{
							"bytes": len(upd.Content.Text),
						})
					})
					if stream != nil {
						SafeStream(stream, StreamChunk{SessionID: sessionID, Delta: upd.Content.Text})
					}
				}
			}
		case "tool_call", "tool_call_update":
			if upd.ToolCallID != "" {
				idx := -1
				for i := range toolCalls {
					if toolCalls[i].ID == upd.ToolCallID {
						idx = i
						break
					}
				}
				name := upd.Title
				if name == "" {
					name = upd.Kind
				}
				status := upd.Status
				result := acpToolResultSummary(upd)
				rawEvent := truncate(rawJSONSummary(upd.Raw), 4000)
				if idx < 0 {
					tc := ToolCall{ID: upd.ToolCallID, ToolName: name, Origin: ToolOriginACPNative}
					tc.Status = status
					tc.Result = result
					tc.RawEvent = rawEvent
					if len(upd.RawInput) > 0 {
						_ = json.Unmarshal(upd.RawInput, &tc.Arguments)
					}
					toolCalls = append(toolCalls, tc)
				} else {
					if name != "" {
						toolCalls[idx].ToolName = name
					}
					if status != "" {
						toolCalls[idx].Status = status
					}
					if result != "" {
						toolCalls[idx].Result = result
					}
					if rawEvent != "" {
						toolCalls[idx].RawEvent = rawEvent
					}
					if len(upd.RawInput) > 0 {
						_ = json.Unmarshal(upd.RawInput, &toolCalls[idx].Arguments)
					}
				}
			}
		}

		if upd.Usage != nil {
			hasUsage = true
			usageIn = upd.Usage.InputTokens
			usageOut = upd.Usage.OutputTokens
		}
	})

	// 发送 prompt
	result, err := st.client.Prompt(ctx, st.ACPSessionID, promptParts, useModel)

	// 清理回调
	st.client.ClearSessionUpdateHandler(st.ACPSessionID)

	if err != nil {
		return nil, err
	}

	mu.Lock()
	finalText := fullText
	finalToolCalls := toolCalls
	finalHasUsage := hasUsage
	finalUsageIn := usageIn
	finalUsageOut := usageOut
	mu.Unlock()

	// 如果 text 为空且没有工具调用：agent 没有任何有效输出，返回错误让上层感知
	if finalText == "" && len(finalToolCalls) == 0 {
		stderrTail := st.client.StderrSnapshot()
		errMsg := "agent returned empty response (no text, no tool calls)"
		if stderrTail != "" {
			errMsg = fmt.Sprintf("%s; stderr: %s", errMsg, strings.TrimSpace(stderrTail))
		}
		if result != nil && result.StopReason != "" {
			errMsg = fmt.Sprintf("%s; stopReason=%s", errMsg, result.StopReason)
		}
		return nil, fmt.Errorf("%s", errMsg)
	}

	// 如果有工具调用但文本为空：agent 只做工具调用没说话，给个占位符
	if finalText == "" {
		finalText = "(agent invoked tools without text output)"
	}

	respMsg := parseAssistantOutput(finalText)
	respMsg.Role = "assistant"
	if len(finalToolCalls) > 0 {
		respMsg.ToolCalls = append(respMsg.ToolCalls, finalToolCalls...)
	}

	a.mu.Lock()
	st.History = append(st.History, respMsg)
	a.mu.Unlock()
	a.Heartbeat(len(a.sessions))

	// 优先用 usage_update 上报的真实 token；缺失时回退到近似估算
	tokenIn := int64(approxTokens(fullPrompt))
	tokenOut := int64(approxTokens(finalText))
	if finalHasUsage {
		tokenIn = finalUsageIn
		tokenOut = finalUsageOut
	}

	resp := &ChatResponse{
		SessionID:    sessionID,
		Message:      respMsg,
		TokenInput:   tokenIn,
		TokenOutput:  tokenOut,
		DurationMs:   time.Since(start).Milliseconds(),
		FinishReason: stopReasonToFinishReason(result.StopReason),
	}

	if len(finalToolCalls) > 0 {
		resp.FinishReason = "tool_calls"
	}

	return resp, nil
}

func acpToolResultSummary(upd acp.UpdatePayload) string {
	for _, raw := range []json.RawMessage{upd.RawOutput, upd.Result, upd.PartialResult} {
		if len(raw) == 0 {
			continue
		}
		return truncate(rawJSONSummary(raw), 2000)
	}
	if upd.Status != "" {
		return "status=" + upd.Status + "; result_unavailable"
	}
	return ""
}

func rawJSONSummary(raw json.RawMessage) string {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return string(raw)
	}
	switch x := v.(type) {
	case string:
		return x
	default:
		b, err := json.Marshal(x)
		if err != nil {
			return fmt.Sprintf("%+v", x)
		}
		return string(b)
	}
}

// buildACPMessageFromHistory 把历史消息拼成一个完整 prompt。
// 保留此函数给测试和兼容路径；真实 ACP 调用走 buildACPContentPartsFromHistory。
func buildACPMessageFromHistory(history []Message) string {
	return contentPartsText(buildACPContentPartsFromHistory(history))
}

// buildPromptParts 构建本轮 prompt 的 content parts。
//
// ACP 是有状态会话协议：session/new 后 agent 侧维护完整对话历史，
// session/prompt 的 prompt 参数语义是"本轮新增输入"。因此默认只发
// 本轮 msg（增量），让 agent 侧会话状态生效，token 成本 O(n) 线性。
//
// 此前每轮 buildACPContentPartsFromHistory(st.History) 全量重发，
// 与有状态协议语义矛盾——agent 侧存一份历史，gloop 又全量重发一份，
// token 随 turn 数平方增长（实测 qst_2606199899 mage review 16 turns
// 烧 205K input token，97.5% 是重复历史）。
//
// fallback 保留：若 agent 实现有状态不符协议（返回空响应疑似丢上下文），
// 调用方可设 FullHistoryFallback=true 回退全量重发。
func (a *ACPExecutor) buildPromptParts(st *acpSessionState, msg Message) []acp.ContentPart {
	if a.fullHistoryFallback {
		return buildACPContentPartsFromHistory(st.History)
	}
	return buildACPContentPartsForMessage(msg)
}

// buildACPContentPartsForMessage 把单条消息转成 ACP content parts（增量模式）。
func buildACPContentPartsForMessage(msg Message) []acp.ContentPart {
	parts := make([]acp.ContentPart, 0, 4)
	if msg.Content != "" {
		parts = append(parts, acp.ContentPart{Type: "text", Text: msg.Content})
	}
	for _, p := range msg.Parts {
		parts = append(parts, acp.ContentPart{Type: "text", Text: renderContentBlockForPrompt(p)})
	}
	// tool_result：ACP agent 侧记得 tool_call，这里只发结果
	for _, tc := range msg.ToolCalls {
		argsJSON, _ := json.Marshal(tc.Arguments)
		parts = append(parts, acp.ContentPart{Type: "text", Text: string(argsJSON)})
	}
	if len(parts) == 0 {
		parts = append(parts, acp.ContentPart{Type: "text", Text: "(empty)"})
	}
	return parts
}

func buildACPContentPartsFromHistory(history []Message) []acp.ContentPart {
	parts := make([]acp.ContentPart, 0, len(history)*2+1)
	parts = append(parts, acp.ContentPart{Type: "text", Text: "<gloop_acp_history>\n"})
	for i, msg := range history {
		header := fmt.Sprintf("  <message index=\"%s\" role=\"%s\"", xmlPromptAttr(fmt.Sprintf("%d", i)), xmlPromptAttr(msg.Role))
		if msg.ToolCallID != "" {
			header += fmt.Sprintf(" tool_call_id=\"%s\"", xmlPromptAttr(msg.ToolCallID))
		}
		header += ">\n"
		parts = append(parts, acp.ContentPart{Type: "text", Text: header})
		if msg.Content != "" {
			parts = append(parts, acp.ContentPart{Type: "text", Text: "    <content>\n" + indentLines(xmlPromptText(msg.Content), "      ") + "\n    </content>\n"})
		}
		for _, part := range msg.Parts {
			parts = append(parts, acp.ContentPart{Type: "text", Text: renderContentBlockForPrompt(part)})
		}
		for _, tc := range msg.ToolCalls {
			argsJSON, _ := json.Marshal(tc.Arguments)
			toolHeader := fmt.Sprintf("    <tool_call name=\"%s\"", xmlPromptAttr(tc.ToolName))
			if tc.ID != "" {
				toolHeader += fmt.Sprintf(" id=\"%s\"", xmlPromptAttr(tc.ID))
			}
			toolHeader += ">\n"
			parts = append(parts, acp.ContentPart{Type: "text", Text: toolHeader + indentLines(xmlPromptText(string(argsJSON)), "      ") + "\n    </tool_call>\n"})
		}
		parts = append(parts, acp.ContentPart{Type: "text", Text: "  </message>\n"})
	}
	parts = append(parts, acp.ContentPart{Type: "text", Text: "  <next_turn role=\"assistant\" />\n</gloop_acp_history>\n"})
	return parts
}

func contentPartsText(parts []acp.ContentPart) string {
	var sb strings.Builder
	for _, part := range parts {
		if part.Type == "text" {
			sb.WriteString(part.Text)
		}
	}
	return sb.String()
}

func stopReasonToFinishReason(stopReason string) string {
	switch stopReason {
	case "end_turn":
		return "stop"
	case "tool_calls":
		return "tool_calls"
	case "max_tokens":
		return "length"
	case "error":
		return "error"
	default:
		return "stop"
	}
}

// ============== fallback ==============

func (a *ACPExecutor) fallback(st *acpSessionState, userMsg Message, originErr error, stream chan<- StreamChunk) (*ChatResponse, error) {
	content := MockDraftResponse(userMsg.Content, originErr)
	if stream != nil {
		for _, line := range strings.SplitAfter(content, "\n") {
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
	a.mu.Lock()
	st.History = append(st.History, resp.Message)
	a.mu.Unlock()
	a.Heartbeat(len(a.sessions))
	if originErr != nil {
		resp.Error = originErr.Error()
		resp.Meta = map[string]any{"fallback": true, "origin_error": originErr.Error()}
	}
	return resp, nil
}

// ============== 上下文导入导出 ==============

func (a *ACPExecutor) ExportContext(ctx context.Context, sessionID string) ([]Message, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	st, ok := a.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("session 不存在: %s", sessionID)
	}
	out := make([]Message, len(st.History))
	copy(out, st.History)
	return out, nil
}

func (a *ACPExecutor) ImportContext(ctx context.Context, sessionID string, history []Message) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	st, ok := a.sessions[sessionID]
	if !ok {
		return fmt.Errorf("session 不存在: %s", sessionID)
	}
	st.History = append([]Message{}, history...)
	// TODO: 同步到 ACP agent 的会话里（如果 agent 支持的话）
	return nil
}

// ============== 其他 ==============

func (a *ACPExecutor) Interrupt(ctx context.Context, sessionID, reason string) error {
	a.mu.RLock()
	st, ok := a.sessions[sessionID]
	a.mu.RUnlock()
	if !ok || st.client == nil {
		return nil
	}
	return st.client.Cancel(st.ACPSessionID)
}

func (a *ACPExecutor) Resume(ctx context.Context, sessionID string) error {
	// ACP 协议没有 resume 语义：被 cancel 的 prompt 不可续跑，需重新发起 prompt。
	// 这里保留 no-op，由 orchestrator 在需要时新开 session 或重发消息。
	return nil
}

func (a *ACPExecutor) GetAvailableTools(ctx context.Context, sessionID string) ([]ToolDef, error) {
	// ACP agent 的工具由 agent 自己管理，平台不直接控制
	// 这里返回一个空列表表示工具由 agent 自洽
	return []ToolDef{}, nil
}

func mergeEnvMap(base map[string]string, extra []string) map[string]string {
	if len(base) == 0 && len(extra) == 0 {
		return nil
	}
	out := map[string]string{}
	for k, v := range base {
		out[k] = v
	}
	for _, kv := range extra {
		k, v, ok := strings.Cut(kv, "=")
		if !ok || k == "" {
			continue
		}
		out[k] = v
	}
	return out
}

func withStartupModelEnv(env map[string]string, envVar string, modelName string) map[string]string {
	if strings.TrimSpace(envVar) == "" || !isConcreteModelName(modelName) {
		return env
	}
	out := map[string]string{}
	for k, v := range env {
		out[k] = v
	}
	out[envVar] = modelName
	return out
}

func isConcreteModelName(modelName string) bool {
	switch strings.ToLower(strings.TrimSpace(modelName)) {
	case "", "auto", "default":
		return false
	default:
		return true
	}
}

// resolveWorkingDir 返回工作根目录
func (a *ACPExecutor) resolveWorkingDir() string {
	return a.workingRoot
}
