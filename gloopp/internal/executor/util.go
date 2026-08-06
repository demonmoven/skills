package executor

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// ==================== 通用 helper（跨 mock/CLI executor 共用） ====================

const (
	RuntimeEventSessionCreated     = "runtime.session.created"
	RuntimeEventProcessStarted     = "runtime.process.started"
	RuntimeEventOutputFirst        = "runtime.output.first_seen"
	RuntimeEventOutputHeartbeat    = "runtime.output.heartbeat"
	RuntimeEventProtocolSignalSeen = "runtime.protocol.signal_seen"
	RuntimeEventIdleWarning        = "runtime.idle_warning"
	RuntimeEventProtocolWarning    = "runtime.protocol_warning"
	RuntimeEventProcessExited      = "runtime.process.exited"
)

// approxTokens 非常粗的 token 估算：中文按字符算，英文按空格/标点分词。
// 只用于 CLI 不可用时的兜底统计，不追求精度。
func approxTokens(s string) int {
	if s == "" {
		return 0
	}
	// 汉字/日文等多字节字符：每个 rune 约 1 token
	cjk := 0
	// 英文/数字等：按空白与标点切词
	words := 0
	inWord := false
	for _, r := range s {
		if r > 127 {
			cjk++
			inWord = false
			continue
		}
		isWord := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_'
		if isWord {
			if !inWord {
				words++
				inWord = true
			}
		} else {
			inWord = false
		}
	}
	// 约 1.3 倍（英文常见 4 字符 ~= 3 token，这里粗估）
	return cjk + int(float64(words)*1.3)
}

func newJSONLineScanner(raw string) *bufio.Scanner {
	scanner := bufio.NewScanner(strings.NewReader(raw))
	scanner.Buffer(make([]byte, 64*1024), 8*1024*1024)
	return scanner
}

func streamOutputLines(sessionID string, raw string, stream chan<- StreamChunk) {
	if stream == nil {
		return
	}
	for _, line := range strings.SplitAfter(raw, "\n") {
		SafeStream(stream, StreamChunk{SessionID: sessionID, Delta: line})
	}
}

func streamRuntimeEvent(sessionID string, stream chan<- StreamChunk, event string, data map[string]any) {
	if stream == nil || event == "" {
		return
	}
	defer func() { _ = recover() }()
	chunk := StreamChunk{SessionID: sessionID, Event: event, Data: data}
	if event == RuntimeEventOutputHeartbeat {
		select {
		case stream <- chunk:
		default:
		}
		return
	}
	select {
	case stream <- chunk:
	case <-time.After(200 * time.Millisecond):
	}
}

type cliRuntimeResult struct {
	Stdout   string
	Stderr   string
	WaitErr  error
	ExitCode int
}

func runCommandWithRuntimeEvents(
	sessionID string,
	cmd *exec.Cmd,
	stream chan<- StreamChunk,
	started time.Time,
	processData map[string]any,
	onStarted func(),
	onFinished func(),
) (cliRuntimeResult, error) {
	var stdout, stderr bytes.Buffer
	var firstOutput sync.Once
	startedOutput := make(chan struct{})
	cmd.Stdout = &runtimeOutputWriter{
		dst:         &stdout,
		sessionID:   sessionID,
		stream:      stream,
		firstOutput: &firstOutput,
		streamName:  "stdout",
		publish:     true,
		started:     startedOutput,
	}
	cmd.Stderr = &runtimeOutputWriter{
		dst:         &stderr,
		sessionID:   sessionID,
		stream:      stream,
		firstOutput: &firstOutput,
		streamName:  "stderr",
		started:     startedOutput,
	}
	if err := cmd.Start(); err != nil {
		return cliRuntimeResult{}, err
	}

	payload := map[string]any{}
	for k, v := range processData {
		payload[k] = v
	}
	if cmd.Process != nil {
		payload["pid"] = cmd.Process.Pid
	}
	streamRuntimeEvent(sessionID, stream, RuntimeEventProcessStarted, payload)
	close(startedOutput)
	if onStarted != nil {
		onStarted()
	}
	if onFinished != nil {
		defer onFinished()
	}

	waitErr := cmd.Wait()

	exitCode := 0
	if waitErr != nil {
		exitCode = -1
		if cmd.ProcessState != nil {
			exitCode = cmd.ProcessState.ExitCode()
		}
	}
	streamRuntimeEvent(sessionID, stream, RuntimeEventProcessExited, map[string]any{
		"exit_code":    exitCode,
		"duration_ms":  time.Since(started).Milliseconds(),
		"stdout_bytes": stdout.Len(),
		"stderr_bytes": stderr.Len(),
	})
	return cliRuntimeResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		WaitErr:  waitErr,
		ExitCode: exitCode,
	}, nil
}

type runtimeOutputWriter struct {
	mu          sync.Mutex
	dst         *bytes.Buffer
	sessionID   string
	stream      chan<- StreamChunk
	firstOutput *sync.Once
	streamName  string
	publish     bool
	started     <-chan struct{}
}

func (w *runtimeOutputWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if w.started != nil {
		<-w.started
	}
	chunk := append([]byte(nil), p...)
	w.mu.Lock()
	_, _ = w.dst.Write(chunk)
	w.mu.Unlock()
	if w.firstOutput != nil {
		w.firstOutput.Do(func() {
			streamRuntimeEvent(w.sessionID, w.stream, RuntimeEventOutputFirst, map[string]any{
				"bytes":  len(chunk),
				"stream": w.streamName,
			})
		})
	}
	streamRuntimeEvent(w.sessionID, w.stream, RuntimeEventOutputHeartbeat, map[string]any{
		"bytes":  len(chunk),
		"stream": w.streamName,
	})
	if w.publish {
		SafeStream(w.stream, StreamChunk{SessionID: w.sessionID, Delta: string(bytes.ToValidUTF8(chunk, nil))})
	}
	return len(p), nil
}

type cliCompletion struct {
	Content    string
	StopReason string
	UsageIn    int64
	UsageOut   int64
	TotalCost  float64
	ModelUsage map[string]any
}

func (b *CLIExecutorBase) completeCLIResponse(st *cliSessionState, userMsg Message, parsed cliCompletion, started time.Time, fallbackErr error, stream chan<- StreamChunk) *ChatResponse {
	resp := &ChatResponse{
		SessionID:   st.SessionID,
		DurationMs:  time.Since(started).Milliseconds(),
		TokenInput:  parsed.UsageIn,
		TokenOutput: parsed.UsageOut,
	}
	content := parsed.Content
	if fallbackErr != nil && strings.TrimSpace(content) == "" {
		content = fallbackErr.Error()
	}
	resp.Message = parseAssistantOutput(content)
	if resp.Message.Content == "" {
		resp.Message.Content = content
	}
	if fallbackErr != nil {
		resp.Error = fallbackErr.Error()
		resp.FinishReason = "error"
		resp.Meta = map[string]any{"origin_error": fallbackErr.Error()}
	} else if parsed.StopReason != "" {
		resp.FinishReason = parsed.StopReason
	} else {
		resp.FinishReason = detectFinishReason(content)
	}
	if len(resp.Message.ToolCalls) > 0 && resp.FinishReason == "stop" {
		resp.FinishReason = "tool_calls"
	}
	if parsed.TotalCost != 0 {
		meta, _ := resp.Meta.(map[string]any)
		if meta == nil {
			meta = map[string]any{}
		}
		meta["cost_usd"] = parsed.TotalCost
		meta["model_usage"] = parsed.ModelUsage
		resp.Meta = meta
	}
	b.AppendHistory(st.SessionID, resp.Message)
	return resp
}

func int64FromAny(v any) int64 {
	switch x := v.(type) {
	case float64:
		return int64(x)
	case int64:
		return x
	case int:
		return int64(x)
	case json.Number:
		i, _ := x.Int64()
		return i
	default:
		return 0
	}
}

func int64FromAnyPath(root map[string]any, path ...string) int64 {
	var cur any = root
	for _, key := range path {
		m, ok := cur.(map[string]any)
		if !ok {
			return 0
		}
		cur = m[key]
	}
	return int64FromAny(cur)
}

func boolFromAny(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	case string:
		return x == "true"
	default:
		return false
	}
}

// detectFinishReason 根据内容粗略判断结束原因
func detectFinishReason(content string) string {
	if strings.TrimSpace(content) == "" {
		return "length"
	}
	// 常见的中断信号
	low := strings.ToLower(content)
	markers := []string{
		"[truncated]", "output cut", "max_tokens", "context length exceeded",
		"token limit reached", "上下文长度超限", "被截断",
	}
	for _, m := range markers {
		if strings.Contains(low, m) {
			return "length"
		}
	}
	return "stop"
}

// truncate 按 rune 截断字符串
func truncate(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	r := []rune(s)
	return string(r[:n]) + "…"
}

// toJSON 把任意值格式化成 JSON 字符串，失败则回退到 %+v
func toJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%+v", v)
	}
	return string(b)
}

// buildPromptFromHistory 把会话历史拼成一段单 prompt 文本，供 CLI agent
// 的 -p 模式使用。CLI agent 会丢失原生 role/message 边界，因此这里渲染为
// XML-ish 的稳定边界，避免 system/user/tool 内容在文本展平时互相污染。
func buildPromptFromHistory(history []Message) string {
	var sb strings.Builder
	sb.WriteString("<gloop_cli_history>\n")
	for i, m := range history {
		sb.WriteString(fmt.Sprintf("  <message index=\"%s\" role=\"%s\"", xmlPromptAttr(fmt.Sprintf("%d", i)), xmlPromptAttr(m.Role)))
		if m.ToolCallID != "" {
			sb.WriteString(fmt.Sprintf(" tool_call_id=\"%s\"", xmlPromptAttr(m.ToolCallID)))
		}
		sb.WriteString(">\n")
		if m.Content != "" {
			sb.WriteString("    <content>\n")
			sb.WriteString(indentLines(xmlPromptText(m.Content), "      "))
			sb.WriteString("\n    </content>\n")
		}
		for _, part := range m.Parts {
			sb.WriteString(renderContentBlockForPrompt(part))
		}
		for _, tc := range m.ToolCalls {
			sb.WriteString(fmt.Sprintf("    <tool_call name=\"%s\"", xmlPromptAttr(tc.ToolName)))
			if tc.ID != "" {
				sb.WriteString(fmt.Sprintf(" id=\"%s\"", xmlPromptAttr(tc.ID)))
			}
			sb.WriteString(">\n")
			sb.WriteString(indentLines(xmlPromptText(toJSON(tc.Arguments)), "      "))
			sb.WriteString("\n    </tool_call>\n")
		}
		sb.WriteString("  </message>\n")
	}
	sb.WriteString("  <next_turn role=\"assistant\" />\n")
	sb.WriteString("</gloop_cli_history>\n")
	return sb.String()
}

func renderContentBlockForPrompt(part ContentBlock) string {
	var sb strings.Builder
	switch part.Type {
	case "text":
		if part.Text != "" {
			sb.WriteString("    <content_part type=\"text\">\n")
			sb.WriteString(indentLines(xmlPromptText(part.Text), "      "))
			sb.WriteString("\n    </content_part>\n")
		}
	case "artifact":
		sb.WriteString(fmt.Sprintf("    <artifact id=\"%s\" kind=\"%s\" mime=\"%s\" name=\"%s\" size=\"%d\" sha256=\"%s\">\n",
			xmlPromptAttr(part.ArtifactID),
			xmlPromptAttr(part.Kind),
			xmlPromptAttr(part.MIME),
			xmlPromptAttr(part.Name),
			part.Size,
			xmlPromptAttr(part.SHA256),
		))
		if part.AbsolutePath != "" {
			sb.WriteString("      <path>")
			sb.WriteString(xmlPromptText(part.AbsolutePath))
			sb.WriteString("</path>\n")
		} else if part.StoragePath != "" {
			sb.WriteString("      <storage_path>")
			sb.WriteString(xmlPromptText(part.StoragePath))
			sb.WriteString("</storage_path>\n")
		}
		sb.WriteString("    </artifact>\n")
	}
	return sb.String()
}

func xmlPromptText(s string) string {
	var b bytes.Buffer
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

func xmlPromptAttr(s string) string {
	return strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&apos;",
	).Replace(s)
}

func indentLines(s, prefix string) string {
	if s == "" {
		return prefix
	}
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}

// parseAssistantOutput 解析模型输出为 Message。
//
// 平台工具调用统一走 CLI syscall 路径（agent 调用 gloop CLI → daemon 处理 → 事件总线发布）。
// 这里只做最基础的文本清洗。
func parseAssistantOutput(raw string) Message {
	return Message{
		Role:    "assistant",
		Content: strings.TrimSpace(raw),
	}
}

// pick 返回 a 如果非空且非 auto/default，否则返回 b。
func pick(a, b string) string {
	if a == "" || a == "auto" || a == "default" {
		return b
	}
	return a
}

// UUID 生成 RFC4122 v4 UUID（给 CLI agent 的 --session-id 用）
func UUID() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		// 兜底：纳秒+随机的兜底
		return fmt.Sprintf("00000000-0000-4000-8000-%012d", time.Now().UnixNano()%1000000000000)
	}
	buf[6] = (buf[6] & 0x0f) | 0x40
	buf[8] = (buf[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", buf[0:4], buf[4:6], buf[6:8], buf[8:10], buf[10:])
}

// ShortID 生成带前缀的短 ID，给 orchestrator 层 session/quest 用
func ShortID(prefix string) string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}
	return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(b)[:12])
}

// SafeStream 往 stream 里安全发送；如果 channel 已关或消费者慢，都不 panic/不阻塞。
// 使用非阻塞 select 避免阻塞核心路径；用 recover 防御 send to closed channel。
func SafeStream(stream chan<- StreamChunk, chunk StreamChunk) {
	if stream == nil {
		return
	}
	defer func() { _ = recover() }()
	select {
	case stream <- chunk:
	default:
	}
}

// ResolveCLIExecutorBase extracts the embedded *CLIExecutorBase from any of
// the concrete CLI executor implementations. Non-CLI executors (ACP, Mock)
// yield nil, meaning the interception point in micro_loop becomes a no-op.
func ResolveCLIExecutorBase(ex Executor) *CLIExecutorBase {
	switch v := ex.(type) {
	case *TraeXExecutor:
		return v.CLIExecutorBase
	case *CodexExecutor:
		return v.CLIExecutorBase
	case *ClaudeCodeExecutor:
		return v.CLIExecutorBase
	case *AidenExecutor:
		return v.CLIExecutorBase
	case *PiExecutor:
		return v.CLIExecutorBase
	}
	return nil
}

// ResolveSessionWorkingDir returns the configured working directory for a
// session on any CLI executor. Empty string fallback is handled by callers.
func ResolveSessionWorkingDir(ex Executor, sessionID string) string {
	switch v := ex.(type) {
	case *TraeXExecutor:
		if st, ok := v.Get(sessionID); ok {
			return st.WorkingDir
		}
	case *CodexExecutor:
		if st, ok := v.Get(sessionID); ok {
			return st.WorkingDir
		}
	case *ClaudeCodeExecutor:
		if st, ok := v.Get(sessionID); ok {
			return st.WorkingDir
		}
	case *AidenExecutor:
		if st, ok := v.Get(sessionID); ok {
			return st.WorkingDir
		}
	case *PiExecutor:
		if st, ok := v.Get(sessionID); ok {
			return st.WorkingDir
		}
	}
	return ""
}

// AdventurerClassFromEnv scans an env slice (os.Environ form) for the
// GLOOP_ADVENTURER_CLASS variable. Empty/unset yields "" so callers can pick
// their own safe default.
func AdventurerClassFromEnv(env []string) string {
	const prefix = "GLOOP_ADVENTURER_CLASS="
	for _, kv := range env {
		if strings.HasPrefix(kv, prefix) {
			return strings.ToLower(strings.TrimSpace(strings.TrimPrefix(kv, prefix)))
		}
	}
	return ""
}
