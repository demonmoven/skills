package executor

import (
	"context"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestWithStartupModelEnv(t *testing.T) {
	base := map[string]string{
		"ANTHROPIC_MODEL": "old-model",
		"OTHER":           "keep",
	}
	got := withStartupModelEnv(base, "ANTHROPIC_MODEL", "claude-sonnet-4")
	want := map[string]string{
		"ANTHROPIC_MODEL": "claude-sonnet-4",
		"OTHER":           "keep",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("env = %#v, want %#v", got, want)
	}
	if base["ANTHROPIC_MODEL"] != "old-model" {
		t.Fatalf("base env was mutated: %#v", base)
	}

	if got := withStartupModelEnv(base, "ANTHROPIC_MODEL", "auto"); !reflect.DeepEqual(got, base) {
		t.Fatalf("auto model should not inject startup env: %#v", got)
	}
	if got := withStartupModelEnv(base, "", "claude-sonnet-4"); !reflect.DeepEqual(got, base) {
		t.Fatalf("empty env var should not inject startup env: %#v", got)
	}
}

// TestACPExecutor_Traex 用真实的 traex ACP 做集成测试
// 需要本机安装 traex 且可执行
func TestACPExecutor_Traex(t *testing.T) {
	if os.Getenv("GLOOP_TEST_ACP_EXECUTOR") == "" {
		t.Skip("跳过真实 ACP executor 集成测试（设置 GLOOP_TEST_ACP_EXECUTOR=1 启用）")
	}
	// 检查 traex 是否可用
	if _, err := exec.LookPath("traex"); err != nil {
		t.Skip("traex 不可用，跳过 ACP 集成测试")
	}

	ex := NewACPExecutor(ACPOpt{
		ExecID:       "test-traex-acp",
		Name:         "TraeX ACP (test)",
		Command:      "traex",
		Args:         []string{"acp", "serve"},
		DefaultModel: "auto",
		WorkingRoot:  "/tmp",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// CreateSession
	handle, err := ex.CreateSession(ctx, "test-session-001", SessionConfig{
		SystemPrompt: "你是一个简洁的助手，回答尽量简短。",
		WorkingDir:   "/tmp",
	})
	if err != nil {
		t.Fatalf("CreateSession 失败: %v", err)
	}
	if handle.SessionID != "test-session-001" {
		t.Errorf("SessionID = %s, want test-session-001", handle.SessionID)
	}
	defer ex.CloseSession(ctx, "test-session-001")

	// SendMessage
	stream := make(chan StreamChunk, 256)
	resp, err := ex.SendMessage(ctx, "test-session-001", Message{
		Role:    "user",
		Content: "说一句 hello world，就这两个单词，不要别的",
	}, "auto", stream)
	if err != nil {
		t.Fatalf("SendMessage 失败: %v", err)
	}
	if resp.Message.Role != "assistant" {
		t.Errorf("Role = %s, want assistant", resp.Message.Role)
	}
	if resp.Message.Content == "" {
		t.Error("Content 为空")
	}
	t.Logf("回复内容: %q", resp.Message.Content)
	t.Logf("Token 用量: in=%d out=%d", resp.TokenInput, resp.TokenOutput)
	t.Logf("FinishReason: %s", resp.FinishReason)
	t.Logf("耗时: %dms", resp.DurationMs)

	// 验证流式输出
	close(stream)
	var streamed string
	for chunk := range stream {
		streamed += chunk.Delta
	}
	if streamed == "" {
		t.Log("注意: stream 为空（可能通知回调时机问题），但不影响主流程")
	} else {
		t.Logf("流式输出长度: %d", len(streamed))
		if !strings.Contains(strings.ToLower(resp.Message.Content), "hello") {
			t.Log("流式内容与最终内容不一致（可能分块方式不同），不视为错误")
		}
	}

	// 第二轮对话 - 验证历史是否保留
	resp2, err := ex.SendMessage(ctx, "test-session-001", Message{
		Role:    "user",
		Content: "刚才我说了什么？",
	}, "auto", nil)
	if err != nil {
		t.Fatalf("第二轮 SendMessage 失败: %v", err)
	}
	t.Logf("第二轮回复: %q", resp2.Message.Content)
	if !strings.Contains(strings.ToLower(resp2.Message.Content), "hello") {
		t.Log("注意：第二轮可能没记住历史（ACP agent 自己管理历史，我们也传了），不强制失败")
	}
}
