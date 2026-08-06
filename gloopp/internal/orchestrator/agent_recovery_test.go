package orchestrator

import (
	"errors"
	"testing"

	"code.byted.org/lihuanyu.0w0/gloop/internal/executor"
)

func TestRecoveryActionsForRelay(t *testing.T) {
	ex := executor.NewMockExecutor("relay_cli")
	actions := recoveryActionsForExecutor(ex)
	if len(actions) != 2 || actions[0] != "relay auth status --json" || actions[1] != "relay auth login --sso" {
		t.Fatalf("relay actions = %#v", actions)
	}
}

func TestRecoveryActionsForTraeXAreNonInteractiveOnly(t *testing.T) {
	ex := executor.NewMockExecutor("traex")
	actions := recoveryActionsForExecutor(ex)
	if len(actions) != 1 || actions[0] != "traex login status" {
		t.Fatalf("traex actions = %#v", actions)
	}
}

func TestRecoveryActionsUnknownExecutor(t *testing.T) {
	ex := executor.NewMockExecutor("codex_cli")
	actions := recoveryActionsForExecutor(ex)
	if len(actions) != 0 {
		t.Fatalf("unknown executor actions = %#v", actions)
	}
}

// TestClassifyAgentError_BuildLevelAuth 覆盖构建级鉴权缺失错误。
// 这类错误不可重试——CLI 二进制按构建形态裁剪了鉴权后端，
// 报 "authentication is not supported in this build" 之类。
// 必须归到 "auth" 走 fail-fast，而不是落到 consecutiveErrors 循环空耗预算。
// 回归：qst_2606232210 的 mage 因 relay 构建无 Anthropic 鉴权报错，
// 未命中 auth pattern，被当 unknown 反复重试。
func TestClassifyAgentError_BuildLevelAuth(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{name: "not supported in this build", err: errors.New("Anthropic authentication is not supported in this build"), want: "auth"},
		{name: "authentication is not supported", err: errors.New("authentication is not supported"), want: "auth"},
		{name: "not logged in", err: errors.New("not logged in, please run /login"), want: "auth"},
		{name: "unauthorized", err: errors.New("401 Unauthorized"), want: "auth"},
		{name: "no api key", err: errors.New("no api key configured"), want: "configuration"},
		{name: "rate limit", err: errors.New("429 Too Many Requests"), want: "rate_limit"},
		{name: "timeout", err: errors.New("context deadline exceeded"), want: "transient"},
		{name: "timed out", err: errors.New("exit status 1 | stdout: Request timed out"), want: "transient"},
		{name: "unknown", err: errors.New("something weird happened"), want: "unknown"},
		{name: "session 不存在", err: errors.New("session 不存在: warrior_0"), want: "session_lost"},
		{name: "session not found", err: errors.New("session not found"), want: "session_lost"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := classifyAgentError(tc.err); got != tc.want {
				t.Fatalf("classifyAgentError(%v) = %q, want %q", tc.err, got, tc.want)
			}
		})
	}
}
