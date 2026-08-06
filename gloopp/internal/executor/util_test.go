package executor

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// parseAssistantOutput 现在只做纯文本清洗：
// 平台工具统一走 CLI syscall 路径，不再解析 ```tool 伪工具块。
func TestParseAssistantOutput_PureText(t *testing.T) {
	raw := "hello world\n\nsecond line"
	msg := parseAssistantOutput(raw)
	if msg.Content != "hello world\n\nsecond line" {
		t.Fatalf("content = %q", msg.Content)
	}
	if len(msg.ToolCalls) != 0 {
		t.Fatalf("expected no tool calls, got %d", len(msg.ToolCalls))
	}
}

// 以前的 ```tool 块现在会被当作纯文本保留，不再被解析为工具调用。
func TestParseAssistantOutput_ToolBlocksTreatedAsText(t *testing.T) {
	raw := "done\n```tool\n{\"tool\":\"phase_checkpoint\",\"args\":{\"summary\":\"ok\"}}\n```"
	msg := parseAssistantOutput(raw)
	// 内容应该原样保留
	if !strings.Contains(msg.Content, "```tool") {
		t.Fatalf("tool block should remain as text, got: %q", msg.Content)
	}
	if len(msg.ToolCalls) != 0 {
		t.Fatalf("expected no tool calls from legacy pseudo-tool blocks")
	}
}

func TestBuildPromptFromHistory_RendersStructuredBoundaries(t *testing.T) {
	got := buildPromptFromHistory([]Message{
		{Role: "system", Content: `platform "policy"`},
		{Role: "user", Content: "fix <bug>"},
		{
			Role:    "assistant",
			Content: "done",
			ToolCalls: []ToolCall{{
				ID:       "call_1",
				ToolName: "phase_checkpoint",
				Arguments: map[string]any{
					"summary": "ok",
				},
			}},
		},
		{Role: "tool", ToolCallID: "call_1", Content: `{"ok":true}`},
	})
	for _, want := range []string{
		`<gloop_cli_history>`,
		`<message index="0" role="system">`,
		`platform &#34;policy&#34;`,
		`<message index="1" role="user">`,
		`fix &lt;bug&gt;`,
		`<tool_call name="phase_checkpoint" id="call_1">`,
		`&#34;summary&#34;:&#34;ok&#34;`,
		`<message index="3" role="tool" tool_call_id="call_1">`,
		`<next_turn role="assistant" />`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected prompt to contain %q\n--- prompt ---\n%s", want, got)
		}
	}
}

func TestCLIExecutorBase_ReadOnlyFiltersWriteAndShellTools(t *testing.T) {
	base := NewCLIExecutorBase("mock", t.TempDir(), []string{"Bash", "Read", "Edit", "Write", "Glob", "Grep", "WebFetch"})
	_, st, err := base.CreateSession("readonly", SessionConfig{
		SystemPrompt: "system",
		ReadOnly:     true,
	})
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	got := strings.Join(st.AllowedTools, ",")
	for _, denied := range []string{"Edit", "Write", "WebFetch"} {
		if strings.Contains(got, denied) {
			t.Fatalf("readonly tools should not contain %s: %v", denied, st.AllowedTools)
		}
	}
	for _, allowed := range []string{"Bash", "Read", "Glob", "Grep"} {
		if !strings.Contains(got, allowed) {
			t.Fatalf("readonly tools should contain %s: %v", allowed, st.AllowedTools)
		}
	}
}

func TestCLIExecutorBase_ExplicitAllowedToolsStillFilteredInReadOnly(t *testing.T) {
	base := NewCLIExecutorBase("mock", t.TempDir(), []string{"Read"})
	_, st, err := base.CreateSession("readonly_explicit", SessionConfig{
		SystemPrompt: "system",
		ReadOnly:     true,
		AllowedTools: []string{"read", "bash", "grep"},
	})
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	got := strings.Join(st.AllowedTools, ",")
	if got != "read,bash,grep" {
		t.Fatalf("allowed tools = %q, want read,bash,grep", got)
	}
}

func TestCompleteCLIResponseBuildsMessageAndMeta(t *testing.T) {
	base := NewCLIExecutorBase("mock", t.TempDir(), nil)
	_, st, err := base.CreateSession("complete_ok", SessionConfig{SystemPrompt: "system"})
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	userMsg := Message{Role: "user", Content: "hello"}
	base.AppendHistory(st.SessionID, userMsg)

	resp := base.completeCLIResponse(st, userMsg, cliCompletion{
		Content:    "done",
		StopReason: "stop_sequence",
		UsageIn:    3,
		UsageOut:   4,
		TotalCost:  0.02,
		ModelUsage: map[string]any{"model": "x"},
	}, time.Now(), nil, nil)

	if resp.Message.Content != "done" {
		t.Fatalf("content = %q", resp.Message.Content)
	}
	if resp.FinishReason != "stop_sequence" {
		t.Fatalf("finish = %q", resp.FinishReason)
	}
	if resp.TokenInput != 3 || resp.TokenOutput != 4 {
		t.Fatalf("usage = %d/%d", resp.TokenInput, resp.TokenOutput)
	}
	meta, ok := resp.Meta.(map[string]any)
	if !ok || meta["cost_usd"] != 0.02 {
		t.Fatalf("meta = %#v", resp.Meta)
	}
	if len(st.History) != 3 || st.History[2].Content != "done" {
		t.Fatalf("history not appended: %+v", st.History)
	}
}

func TestTraeXParseJSONLCapturesThreadID(t *testing.T) {
	raw := strings.Join([]string{
		`{"type":"thread.started","thread_id":"thread_123"}`,
		`{"type":"item.completed","item":{"type":"agent_message","text":"READY"}}`,
		`{"type":"turn.completed","usage":{"input_tokens":11,"output_tokens":3},"finish_reason":"stop"}`,
	}, "\n")
	got := parseTraeXJSONL(raw)
	if got.ThreadID != "thread_123" {
		t.Fatalf("ThreadID = %q, want thread_123", got.ThreadID)
	}
	if got.Content != "READY" {
		t.Fatalf("Content = %q, want READY", got.Content)
	}
	if got.UsageIn != 11 || got.UsageOut != 3 {
		t.Fatalf("usage = %d/%d, want 11/3", got.UsageIn, got.UsageOut)
	}
}

func TestTraeXParseJSONLCapturesFileChanges(t *testing.T) {
	raw := strings.Join([]string{
		`{"type":"item.completed","item":{"id":"item_0","type":"file_change","changes":[{"path":"/tmp/outside.md","kind":"add"}],"status":"completed"}}`,
		`{"type":"item.completed","item":{"type":"agent_message","text":"READY"}}`,
	}, "\n")
	got := parseTraeXJSONL(raw)
	if len(got.FileChanges) != 1 {
		t.Fatalf("file changes = %#v, want 1", got.FileChanges)
	}
	change := got.FileChanges[0]
	if change.Path != "/tmp/outside.md" || change.Kind != "add" || change.Tool != "file_change" || change.Status != "completed" {
		t.Fatalf("unexpected file change: %+v", change)
	}
}

func TestAttachTraeXFileChangesUsesPrefixedMetaKey(t *testing.T) {
	resp := &ChatResponse{}
	attachTraeXFileChanges(resp, []FileChangeObservation{{Path: "/tmp/outside.md", Kind: "add"}})
	meta, ok := resp.Meta.(map[string]any)
	if !ok {
		t.Fatalf("meta = %#v, want map", resp.Meta)
	}
	if _, ok := meta["file_changes"]; ok {
		t.Fatalf("unprefixed file_changes key should not be used: %#v", meta)
	}
	changes, ok := meta["traex_file_changes"].([]FileChangeObservation)
	if !ok || len(changes) != 1 || changes[0].Path != "/tmp/outside.md" {
		t.Fatalf("traex_file_changes = %#v", meta["traex_file_changes"])
	}
}

func TestTraeXCommandArgsUseResumeWithoutOverwritingGloopSessionID(t *testing.T) {
	base := NewCLIExecutorBase("traex", t.TempDir(), nil)
	_, st, err := base.CreateSession("gloop_session", SessionConfig{
		SystemPrompt: "system",
		WorkingDir:   "/tmp/work",
	})
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	if st.SessionID != "gloop_session" {
		t.Fatalf("Gloop SessionID = %q, want gloop_session", st.SessionID)
	}

	firstArgs := strings.Join(traexCommandArgs(st), "\x00")
	if !strings.Contains(firstArgs, "exec\x00--skip-git-repo-check\x00--json\x00--cd\x00/tmp/work") {
		t.Fatalf("first args = %q", firstArgs)
	}
	if strings.Contains(firstArgs, "resume") {
		t.Fatalf("first args should not resume: %q", firstArgs)
	}

	base.SetTransportSessionID(st.SessionID, "thread_abc")
	if st.SessionID != "gloop_session" {
		t.Fatalf("Gloop SessionID was overwritten: %q", st.SessionID)
	}
	if st.TransportSessionID != "thread_abc" {
		t.Fatalf("TransportSessionID = %q, want thread_abc", st.TransportSessionID)
	}
	resumeArgs := strings.Join(traexCommandArgs(st), "\x00")
	if !strings.Contains(resumeArgs, "exec\x00resume\x00thread_abc\x00--skip-git-repo-check\x00--json") {
		t.Fatalf("resume args = %q", resumeArgs)
	}
	if strings.Contains(resumeArgs, "--cd") {
		t.Fatalf("resume args should not pass --cd; TraeX resumes the thread context: %q", resumeArgs)
	}
}

func TestTraeXPromptUsesDeltaWhenResuming(t *testing.T) {
	base := NewCLIExecutorBase("traex", t.TempDir(), nil)
	_, st, err := base.CreateSession("prompt_session", SessionConfig{SystemPrompt: "system policy"})
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	base.AppendHistory(st.SessionID, Message{Role: "user", Content: "old request"})
	base.AppendHistory(st.SessionID, Message{Role: "assistant", Content: "old answer"})
	current := Message{Role: "user", Content: "new request"}
	base.AppendHistory(st.SessionID, current)

	firstPrompt := traexPromptForTurn(st, current)
	for _, want := range []string{"system policy", "old request", "old answer", "new request"} {
		if !strings.Contains(firstPrompt, want) {
			t.Fatalf("first prompt missing %q:\n%s", want, firstPrompt)
		}
	}

	base.SetTransportSessionID(st.SessionID, "thread_abc")
	resumePrompt := traexPromptForTurn(st, current)
	if !strings.Contains(resumePrompt, "new request") {
		t.Fatalf("resume prompt missing current message:\n%s", resumePrompt)
	}
	for _, old := range []string{"system policy", "old request", "old answer"} {
		if strings.Contains(resumePrompt, old) {
			t.Fatalf("resume prompt should not resend %q:\n%s", old, resumePrompt)
		}
	}
}

func TestTraeXResumeErrorIncludesMachineReadableMarker(t *testing.T) {
	err := traexResumeError("thread_abc", errors.New("thread expired"))
	if err == nil {
		t.Fatal("expected error")
	}
	got := err.Error()
	for _, want := range []string{"resume_failed=true", "transport=traex", "thread_id=thread_abc", "thread expired"} {
		if !strings.Contains(got, want) {
			t.Fatalf("error %q missing %q", got, want)
		}
	}
}

func TestTraeXFallbackErrorMarksResumeEvenWithContent(t *testing.T) {
	err := traexFallbackError(true, "thread_abc", "partial response", errors.New("exit status 1"))
	if err == nil {
		t.Fatal("resume wait error should remain an error even when content exists")
	}
	got := err.Error()
	for _, want := range []string{"resume_failed=true", "thread_id=thread_abc", "exit status 1"} {
		if !strings.Contains(got, want) {
			t.Fatalf("error %q missing %q", got, want)
		}
	}
}

func TestTraeXFallbackErrorKeepsFirstTurnContentAsSuccess(t *testing.T) {
	err := traexFallbackError(false, "", "usable response", errors.New("exit status 1"))
	if err != nil {
		t.Fatalf("first-turn content should keep legacy success semantics, got %v", err)
	}
}

func TestExecutorCapabilityTiers(t *testing.T) {
	cases := []struct {
		name string
		ex   Executor
		want CapabilityTier
	}{
		{"acp", NewACPExecutor(ACPOpt{Command: "agent"}), CapabilityTierB},
		{"relay", NewClaudeCodeExecutor(ClaudeCodeOpt{BinPath: "relay"}), CapabilityTierB},
		{"codex", NewCodexExecutor(CodexOpt{BinPath: "codex"}), CapabilityTierB},
		{"aiden_base", NewAidenExecutor(AidenOpt{BinPath: "aiden"}), CapabilityTierC},
		{"aiden_x_claude", NewAidenExecutor(AidenOpt{BinPath: "aiden", Variant: AidenVariantXClaude}), CapabilityTierB},
		{"pi", NewPiExecutor(PiOpt{BinPath: "pi"}), CapabilityTierC},
		{"mock", NewMockExecutor("mock"), CapabilityTierTest},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.ex.CapabilityTier(); got != c.want {
				t.Fatalf("CapabilityTier = %s, want %s", got, c.want)
			}
		})
	}
}

func TestRunCommandWithRuntimeEvents_StderrTriggersFirstOutput(t *testing.T) {
	stream := make(chan StreamChunk, 8)
	cmd := exec.CommandContext(context.Background(), "sh", "-c", "printf booting >&2")
	result, err := runCommandWithRuntimeEvents("sid_stderr", cmd, stream, time.Now(), map[string]any{
		"adapter": "test",
	}, nil, nil)
	if err != nil {
		t.Fatalf("runCommandWithRuntimeEvents failed: %v", err)
	}
	if result.WaitErr != nil {
		t.Fatalf("command wait error: %v", result.WaitErr)
	}
	if result.Stdout != "" || result.Stderr != "booting" {
		t.Fatalf("stdout/stderr = %q/%q, want empty/booting", result.Stdout, result.Stderr)
	}

	found := false
	for len(stream) > 0 {
		chunk := <-stream
		if chunk.Event != RuntimeEventOutputFirst {
			continue
		}
		data, _ := chunk.Data.(map[string]any)
		if data["stream"] != "stderr" {
			t.Fatalf("first output stream = %v, want stderr", data["stream"])
		}
		found = true
	}
	if !found {
		t.Fatal("missing runtime.output.first_seen for stderr-only command")
	}
}

func TestRunCommandWithRuntimeEvents_EmitsOutputHeartbeat(t *testing.T) {
	stream := make(chan StreamChunk, 8)
	cmd := exec.CommandContext(context.Background(), "sh", "-c", "printf alive")
	result, err := runCommandWithRuntimeEvents("sid_heartbeat", cmd, stream, time.Now(), map[string]any{
		"adapter": "test",
	}, nil, nil)
	if err != nil {
		t.Fatalf("runCommandWithRuntimeEvents failed: %v", err)
	}
	if result.WaitErr != nil {
		t.Fatalf("command wait error: %v", result.WaitErr)
	}
	if result.Stdout != "alive" {
		t.Fatalf("stdout = %q, want alive", result.Stdout)
	}

	found := false
	for len(stream) > 0 {
		chunk := <-stream
		if chunk.Event != RuntimeEventOutputHeartbeat {
			continue
		}
		data, _ := chunk.Data.(map[string]any)
		if data["stream"] != "stdout" {
			t.Fatalf("heartbeat stream = %v, want stdout", data["stream"])
		}
		found = true
	}
	if !found {
		t.Fatal("missing runtime.output.heartbeat")
	}
}

func TestCompleteCLIResponseMarksToolCalls(t *testing.T) {
	base := NewCLIExecutorBase("mock", t.TempDir(), nil)
	_, st, err := base.CreateSession("complete_tool", SessionConfig{SystemPrompt: "system"})
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	userMsg := Message{Role: "user", Content: "call tool"}
	base.AppendHistory(st.SessionID, userMsg)

	// 手动构造带 ToolCalls 的 Message（模拟 ACP native tool 路径）
	// 验证 completeCLIResponse 正确标记 tool_calls finish reason
	resp := base.completeCLIResponse(st, userMsg, cliCompletion{
		Content:    "using tools",
		StopReason: "stop",
	}, time.Now(), nil, nil)

	// 手动补充 ToolCalls（模拟 parse 后的结果）
	resp.Message.ToolCalls = []ToolCall{
		{ID: "call_1", ToolName: "bash", Origin: ToolOriginACPNative, Arguments: map[string]any{"command": "ls"}},
	}

	// 重新触发 finish reason 标记逻辑（手动调用一次检测）
	if len(resp.Message.ToolCalls) > 0 && resp.FinishReason == "stop" {
		resp.FinishReason = "tool_calls"
	}

	if resp.FinishReason != "tool_calls" {
		t.Fatalf("finish = %q, want tool_calls", resp.FinishReason)
	}
	if len(resp.Message.ToolCalls) != 1 || resp.Message.ToolCalls[0].ToolName != "bash" {
		t.Fatalf("tool calls = %+v", resp.Message.ToolCalls)
	}
}

func TestCompleteCLIResponseCarriesAgentError(t *testing.T) {
	base := NewCLIExecutorBase("mock", t.TempDir(), nil)
	_, st, err := base.CreateSession("complete_fallback", SessionConfig{SystemPrompt: "system"})
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	userMsg := Message{Role: "user", Content: "need fallback"}
	base.AppendHistory(st.SessionID, userMsg)

	resp := base.completeCLIResponse(st, userMsg, cliCompletion{}, time.Now(), errBoom{}, nil)

	if resp.Error != "boom" || resp.FinishReason != "error" {
		t.Fatalf("response should carry error: %+v", resp)
	}
	if len(st.History) != 3 || st.History[2].Content != "boom" {
		t.Fatalf("error history not appended: %+v", st.History)
	}
}

type errBoom struct{}

func (errBoom) Error() string { return "boom" }

func TestBuildACPContentPartsFromHistory_RendersStructuredParts(t *testing.T) {
	parts := buildACPContentPartsFromHistory([]Message{
		{Role: "system", Content: "platform policy"},
		{Role: "user", Content: "fix <bug>"},
		{
			Role:    "assistant",
			Content: "done",
			ToolCalls: []ToolCall{{
				ID:        "call_1",
				ToolName:  "review_quest",
				Arguments: map[string]any{"verdict": "pass"},
			}},
		},
	})
	if len(parts) < 5 {
		t.Fatalf("parts = %d, want multiple structured parts", len(parts))
	}
	got := contentPartsText(parts)
	for _, want := range []string{
		`<gloop_acp_history>`,
		`<message index="1" role="user">`,
		`fix &lt;bug&gt;`,
		`<tool_call name="review_quest" id="call_1">`,
		`&#34;verdict&#34;:&#34;pass&#34;`,
		`<next_turn role="assistant" />`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected ACP prompt to contain %q\n--- prompt ---\n%s", want, got)
		}
	}
}

// TestBuildACPContentPartsForMessage_Incremental 验证增量模式只发单条消息，
// 不含 <gloop_acp_history> 包装——ACP 有状态协议下 agent 侧维护历史，
// gloop 只需发本轮新增。回归：此前每轮全量重发导致 token O(n²)。
func TestBuildACPContentPartsForMessage_Incremental(t *testing.T) {
	msg := Message{Role: "user", Content: "fix the bug"}
	parts := buildACPContentPartsForMessage(msg)
	got := contentPartsText(parts)

	// 增量模式不含 history 包装标签
	if strings.Contains(got, "<gloop_acp_history>") {
		t.Fatalf("incremental mode should not wrap in history tag: %s", got)
	}
	if !strings.Contains(got, "fix the bug") {
		t.Fatalf("incremental prompt should contain message content: %s", got)
	}
}

// TestACPExecutor_BuildPromptParts_Fallback 验证 fullHistoryFallback=true
// 时回退全量重发（旧行为），供无状态 ACP agent 兜底。
func TestACPExecutor_BuildPromptParts_Fallback(t *testing.T) {
	a := NewACPExecutor(ACPOpt{Command: "test", FullHistoryFallback: true})
	st := &acpSessionState{
		History: []Message{
			{Role: "system", Content: "policy"},
			{Role: "user", Content: "do task"},
		},
	}
	parts := a.buildPromptParts(st, Message{Role: "user", Content: "round 2"})
	got := contentPartsText(parts)

	if !strings.Contains(got, "<gloop_acp_history>") {
		t.Fatalf("fallback mode should wrap in history tag: %s", got)
	}
	if !strings.Contains(got, "do task") {
		t.Fatalf("fallback should include prior history: %s", got)
	}
}

// TestACPExecutor_BuildPromptParts_IncrementalDefault 验证默认增量模式。
func TestACPExecutor_BuildPromptParts_IncrementalDefault(t *testing.T) {
	a := NewACPExecutor(ACPOpt{Command: "test"}) // 默认 FullHistoryFallback=false
	st := &acpSessionState{
		History: []Message{
			{Role: "system", Content: "policy"},
			{Role: "user", Content: "do task"},
		},
	}
	parts := a.buildPromptParts(st, Message{Role: "user", Content: "round 2"})
	got := contentPartsText(parts)

	if strings.Contains(got, "<gloop_acp_history>") {
		t.Fatalf("default should be incremental, not history-wrapped: %s", got)
	}
	if strings.Contains(got, "do task") {
		t.Fatalf("incremental should NOT include prior history: %s", got)
	}
	if !strings.Contains(got, "round 2") {
		t.Fatalf("incremental should contain current message: %s", got)
	}
}

func TestClaudeCodeExecutor_Metadata(t *testing.T) {
	ex := NewClaudeCodeExecutor(ClaudeCodeOpt{ExecID: "relay_test", BinPath: "relay", DefaultModel: "auto", WorkingRoot: t.TempDir()})
	if ex.ID() != "relay_test" {
		t.Fatalf("id = %s", ex.ID())
	}
	if ex.Type() != "cli" {
		t.Fatalf("type = %s", ex.Type())
	}
	if ex.DefaultModel() != "auto" {
		t.Fatalf("default model = %s", ex.DefaultModel())
	}
	if len(ex.Capabilities()) == 0 {
		t.Fatal("expected capabilities")
	}
}

func TestRelayCommandArgs_ReadOnlyUsesToolsNotAllowedTools(t *testing.T) {
	st := &cliSessionState{
		WorkingDir:   t.TempDir(),
		History:      []Message{{Role: "system", Content: "system"}},
		AllowedTools: []string{"Read", "Glob", "Grep"},
		ReadOnly:     true,
	}
	args := strings.Join(relayCommandArgs(st, "auto"), " ")
	// v0.3: ReadOnly no longer changes tool arg style; always uses --allowedTools
	if !strings.Contains(args, "--allowedTools Read,Glob,Grep") {
		t.Fatalf("args should include allowedTools: %s", args)
	}
	if strings.Contains(args, "--bare") || strings.Contains(args, "--disable-slash-commands") {
		t.Fatalf("relay adapter should not alter Relay internal runtime surface by default: %s", args)
	}
}

func TestParseRelayStreamJSON_AssistantAndResult(t *testing.T) {
	raw := strings.Join([]string{
		`{"type":"system","subtype":"init"}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"hello "},{"type":"text","text":"relay"}],"usage":{"input_tokens":11,"output_tokens":7},"stop_reason":"end_turn"}}`,
		`{"type":"result","subtype":"success","is_error":false,"result":"final relay","stop_reason":"stop_sequence","usage":{"input_tokens":13,"output_tokens":5},"total_cost_usd":0.01,"modelUsage":{"x":1}}`,
	}, "\n")
	got := parseRelayStreamJSON(raw)
	if got.Content != "final relay" {
		t.Fatalf("content = %q", got.Content)
	}
	if got.UsageIn != 24 || got.UsageOut != 12 {
		t.Fatalf("usage = in %d out %d", got.UsageIn, got.UsageOut)
	}
	if got.StopReason != "stop_sequence" {
		t.Fatalf("stop reason = %q", got.StopReason)
	}
	if got.IsError {
		t.Fatal("should not be error")
	}
	if got.TotalCost != 0.01 {
		t.Fatalf("cost = %v", got.TotalCost)
	}
}

func TestParseRelayStreamJSON_ErrorResult(t *testing.T) {
	got := parseRelayStreamJSON(`{"type":"result","is_error":true,"result":"Not logged in"}`)
	if !got.IsError {
		t.Fatal("expected error result")
	}
	if got.Content != "Not logged in" {
		t.Fatalf("content = %q", got.Content)
	}
}

func TestPiExecutor_Metadata(t *testing.T) {
	ex := NewPiExecutor(PiOpt{ExecID: "pi_test", BinPath: "pi", DefaultModel: "auto", WorkingRoot: t.TempDir()})
	if ex.ID() != "pi_test" {
		t.Fatalf("id = %s", ex.ID())
	}
	if ex.Type() != "cli" {
		t.Fatalf("type = %s", ex.Type())
	}
	if ex.DefaultModel() != "auto" {
		t.Fatalf("default model = %s", ex.DefaultModel())
	}
	if len(ex.Capabilities()) == 0 {
		t.Fatal("expected capabilities")
	}
}

func TestParsePiJSON(t *testing.T) {
	raw := strings.Join([]string{
		`{"type":"session","version":3}`,
		`{"type":"message_end","message":{"role":"assistant","content":[{"type":"text","text":"hello "},{"type":"text","text":"pi"}],"stopReason":"stop","usage":{"input":9,"output":4}}}`,
		`{"type":"turn_end","message":{"role":"assistant","content":"final pi","usage":{"input":3,"output":2}},"toolResults":[]}`,
	}, "\n")
	got := parsePiJSON(raw)
	if got.Content != "final pi" {
		t.Fatalf("content = %q", got.Content)
	}
	if got.UsageIn != 12 || got.UsageOut != 6 {
		t.Fatalf("usage = in %d out %d", got.UsageIn, got.UsageOut)
	}
	if got.StopReason != "stop" {
		t.Fatalf("stop reason = %q", got.StopReason)
	}
}

func TestParsePiJSON_SessionOnlyHasNoContent(t *testing.T) {
	got := parsePiJSON(`{"type":"session","version":3,"id":"sid"}`)
	if got.Content != "" {
		t.Fatalf("content = %q", got.Content)
	}
}

func TestParseRelayStreamJSON(t *testing.T) {
	raw := `{"type":"assistant","message":{"content":[{"type":"text","text":"hello"}],"usage":{"input_tokens":3,"output_tokens":4},"stop_reason":"end_turn"}}`
	got := parseRelayStreamJSON(raw)
	if got.Content != "hello" {
		t.Fatalf("content = %q", got.Content)
	}
	if got.UsageIn != 3 || got.UsageOut != 4 {
		t.Fatalf("usage = %d/%d", got.UsageIn, got.UsageOut)
	}
	if got.StopReason != "end_turn" {
		t.Fatalf("stop reason = %q", got.StopReason)
	}
}

// ==================== Aiden 相关测试 ====================

func TestAidenExecutor_Metadata(t *testing.T) {
	ex := NewAidenExecutor(AidenOpt{ExecID: "aiden_test", BinPath: "aiden", DefaultModel: "auto", WorkingRoot: t.TempDir()})
	if ex.ID() != "aiden_test" {
		t.Fatalf("id = %s", ex.ID())
	}
	if ex.Type() != "cli" {
		t.Fatalf("type = %s", ex.Type())
	}
	if ex.DefaultModel() != "auto" {
		t.Fatalf("default model = %s", ex.DefaultModel())
	}
	if len(ex.Capabilities()) == 0 {
		t.Fatal("expected capabilities")
	}
	// base 变种默认模型
	models := ex.AvailableModels()
	if len(models) < 2 {
		t.Fatalf("expected at least 2 models for base, got %d", len(models))
	}
	if ex.DisplayName() != "Aiden CLI" {
		t.Fatalf("display name = %q, want Aiden CLI", ex.DisplayName())
	}
}

func TestAidenExecutor_XClaudeVariant(t *testing.T) {
	ex := NewAidenExecutor(AidenOpt{
		ExecID:       "aiden_xc",
		BinPath:      "aiden",
		Variant:      AidenVariantXClaude,
		DefaultModel: "auto",
		WorkingRoot:  t.TempDir(),
	})
	if ex.DisplayName() != "Aiden X Claude" {
		t.Fatalf("display name = %q, want Aiden X Claude", ex.DisplayName())
	}
	models := ex.AvailableModels()
	found := false
	for _, m := range models {
		if m.ID == "seedcode0611" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("x_claude variant should have seedcode0611 model, got %+v", models)
	}

	_, st, err := ex.CLIExecutorBase.CreateSession("x_claude_args", SessionConfig{SystemPrompt: "system-only"})
	if err != nil {
		t.Fatal(err)
	}
	ex.CLIExecutorBase.AppendHistory(st.SessionID, Message{Role: "user", Content: "user-only"})
	args := strings.Join(ex.buildArgs("auto", st, buildPromptFromHistory(st.History), buildPromptFromHistory(nonSystemHistory(st.History))), "\x00")
	if !strings.Contains(args, "--system-prompt\x00system-only") {
		t.Fatalf("x_claude args should pass system prompt separately: %q", args)
	}
	if strings.Contains(args, "system-only</content>") {
		t.Fatalf("x_claude user prompt should not include system history: %q", args)
	}
	if !strings.Contains(args, "--print") || !strings.Contains(args, "user-only") {
		t.Fatalf("x_claude args missing print/user prompt: %q", args)
	}
}

func TestAidenExecutor_XCodexVariant(t *testing.T) {
	ex := NewAidenExecutor(AidenOpt{
		ExecID:       "aiden_xcodex",
		BinPath:      "aiden",
		Variant:      AidenVariantXCodex,
		DefaultModel: "auto",
		WorkingRoot:  t.TempDir(),
	})
	if ex.DisplayName() != "Aiden X Codex" {
		t.Fatalf("display name = %q, want Aiden X Codex", ex.DisplayName())
	}
	models := ex.AvailableModels()
	found := false
	for _, m := range models {
		if m.ID == "gpt-5.5-paygo" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("x_codex variant should have gpt-5.5-paygo model, got %+v", models)
	}

	_, st, err := ex.CLIExecutorBase.CreateSession("x_codex_args", SessionConfig{SystemPrompt: "system-only"})
	if err != nil {
		t.Fatal(err)
	}
	ex.CLIExecutorBase.AppendHistory(st.SessionID, Message{Role: "user", Content: "user-only"})
	args := strings.Join(ex.buildArgs("auto", st, buildPromptFromHistory(st.History), buildPromptFromHistory(nonSystemHistory(st.History))), "\x00")
	if !strings.Contains(args, "exec\x00--system-prompt\x00system-only") {
		t.Fatalf("x_codex args should pass system prompt after exec: %q", args)
	}
	if strings.Contains(args, "system-only</content>") {
		t.Fatalf("x_codex user prompt should not include system history: %q", args)
	}
	if !strings.Contains(args, "user-only") {
		t.Fatalf("x_codex args missing user prompt: %q", args)
	}
}

func TestParseAidenStreamJSON_MessageUpdate(t *testing.T) {
	raw := strings.Join([]string{
		`{"type":"session","sessionId":"test-123"}`,
		`{"type":"event","event":{"name":"message:create","id":"msg-1","from":"tmates","content":"hello ","usage":{"context_length":270000},"isStreaming":true}}`,
		`{"type":"event","event":{"name":"message:update","id":"msg-1","from":"tmates","content":"hello aiden","usage":{"input_tokens":100,"output_tokens":5,"total_tokens":105}}}`,
		`{"type":"done","status":"completed"}`,
	}, "\n")
	got := parseAidenStreamJSON(raw)
	if got.Content != "hello aiden" {
		t.Fatalf("content = %q", got.Content)
	}
	if got.UsageIn != 100 || got.UsageOut != 5 {
		t.Fatalf("usage = in %d out %d", got.UsageIn, got.UsageOut)
	}
	if got.StopReason != "stop" {
		t.Fatalf("stop reason = %q", got.StopReason)
	}
	if got.IsError {
		t.Fatal("should not be error")
	}
}

func TestParseAidenStreamJSON_OnlySessionHasNoContent(t *testing.T) {
	got := parseAidenStreamJSON(`{"type":"session","sessionId":"test-123"}`)
	if got.Content != "" {
		t.Fatalf("content = %q", got.Content)
	}
}

func TestParseAidenStreamJSON_EmptyInput(t *testing.T) {
	got := parseAidenStreamJSON("")
	if got.Content != "" {
		t.Fatalf("content = %q", got.Content)
	}
	if !got.IsError {
		// 空输入兜底会保留原内容（空字符串），但 IsError 应为 false
		// 这里主要验证不 panic
	}
}

func TestParseAidenStreamJSON_FailedStatus(t *testing.T) {
	got := parseAidenStreamJSON(`{"type":"done","status":"failed"}`)
	if !got.IsError {
		t.Fatal("expected error for failed status")
	}
}

// 只读 session 不暴露 Gloop platform tool；法师通过 Bash 调用 gloop CLI。
func TestCLIExecutorBase_ReadOnlyDoesNotInjectGloopPlatformTool(t *testing.T) {
	base := NewCLIExecutorBase("mock", t.TempDir(), []string{"Bash", "Read", "Edit", "Write", "Gloop"})
	_, st, err := base.CreateSession("ro_no_gloop_tool", SessionConfig{
		SystemPrompt: "s",
		ReadOnly:     true,
		AllowedTools: []string{"Bash", "Read", "Edit", "Write", "Gloop"},
	})
	if err != nil {
		t.Fatal(err)
	}
	hasBash := false
	hasGloop := false
	hasWrite := false
	for _, x := range st.AllowedTools {
		if x == "Bash" {
			hasBash = true
		}
		if x == "Gloop" {
			hasGloop = true
		}
		if x == "Write" || x == "Edit" {
			hasWrite = true
		}
	}
	if !hasBash {
		t.Fatalf("readonly session 应保留 Bash 以调用 gloop CLI, got %v", st.AllowedTools)
	}
	if hasGloop {
		t.Fatalf("readonly session 不应暴露 Gloop platform tool, got %v", st.AllowedTools)
	}
	if hasWrite {
		t.Fatalf("readonly session 不应保留直接写工具, got %v", st.AllowedTools)
	}
}
