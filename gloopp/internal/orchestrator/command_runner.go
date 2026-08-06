package orchestrator

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"code.byted.org/lihuanyu.0w0/gloop/internal/events"
	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
	"code.byted.org/lihuanyu.0w0/gloop/internal/platformtools"
)

const commandOutputLimit = 16 * 1024

func (e *Engine) RunAllowedCommand(ctx context.Context, qid, sid, commandID string, extraArgs []string) (platformtools.CommandRunResult, error) {
	if e == nil || e.cfg == nil {
		return platformtools.CommandRunResult{}, fmt.Errorf("command runner 未初始化")
	}
	allowed, ok := fsstore.FindAllowedCommand(e.cfg.CommandAllowlist, commandID)
	if !ok {
		return platformtools.CommandRunResult{}, fmt.Errorf("命令不在 白名单: %s", commandID)
	}
	if len(extraArgs) > 0 && !allowed.AllowExtraArgs {
		return platformtools.CommandRunResult{}, fmt.Errorf("命令 %s 不允许 extra_args", commandID)
	}
	q, err := e.GetQuest(qid)
	if err != nil {
		return platformtools.CommandRunResult{}, err
	}
	if allowed.SideEffectLevel == "L2" && !e.l2Allowed(q) {
		e.publish(qid, sid, events.EvtCommandRun, map[string]any{
			"command_id":        commandID,
			"argv":              append([]string{allowed.Command}, allowed.Args...),
			"exit_code":         -1,
			"side_effect_level": allowed.SideEffectLevel,
			"blocked":           true,
			"reason":            "l2_requires_policy",
		})
		return platformtools.CommandRunResult{}, fmt.Errorf("命令 %s 是 L2 外部副作用命令，需要 automation allow_l2 policy 且不能 auto_apply", commandID)
	}
	if q.WorkspacePath == "" {
		return platformtools.CommandRunResult{}, fmt.Errorf("quest workspace_path 为空")
	}

	args := append([]string(nil), allowed.Args...)
	args = append(args, extraArgs...)
	argv := append([]string{allowed.Command}, args...)
	timeout := time.Duration(allowed.TimeoutMs) * time.Millisecond
	if timeout <= 0 {
		timeout = time.Duration(fsstore.DefaultMageCommandTimeoutMs) * time.Millisecond
	}

	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, allowed.Command, args...)
	cmd.Dir = q.WorkspacePath
	// 隔离 GLOOP_* 环境变量：这些是 gloop 给 agent 的 syscall 上下文，
	// 不该泄漏进平台验证命令（go test / go build 等）的子进程——否则
	// 仓库测试会读到 GLOOP_QUEST_ID 去写当前 quest 的 signal 文件而失败。
	cmd.Env = filteredEnv(os.Environ())
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err = cmd.Run()
	durationMs := time.Since(start).Milliseconds()
	timedOut := runCtx.Err() == context.DeadlineExceeded
	exitCode := exitCodeFromError(err)
	if timedOut && exitCode == 0 {
		exitCode = -1
	}

	result := platformtools.CommandRunResult{
		CommandID:  commandID,
		Argv:       argv,
		ExitCode:   exitCode,
		DurationMs: durationMs,
		Stdout:     truncateMiddle(stdout.String(), commandOutputLimit),
		Stderr:     truncateMiddle(stderr.String(), commandOutputLimit),
		TimedOut:   timedOut,
	}
	e.appendSessionRow(qid, sid, &fsstore.QuestSessionRow{
		Timestamp: fsstore.NowMs(),
		Kind:      "command_run",
		SessionID: sid,
		Role:      "tool",
		Content:   fmt.Sprintf("%s exit=%d", commandID, exitCode),
		Meta: map[string]any{
			"command_id":        result.CommandID,
			"argv":              result.Argv,
			"exit_code":         result.ExitCode,
			"duration_ms":       result.DurationMs,
			"timed_out":         result.TimedOut,
			"side_effect_level": allowed.SideEffectLevel,
		},
	})
	e.publish(qid, sid, events.EvtCommandRun, map[string]any{
		"command_id":        result.CommandID,
		"argv":              result.Argv,
		"exit_code":         result.ExitCode,
		"duration_ms":       result.DurationMs,
		"timed_out":         result.TimedOut,
		"side_effect_level": allowed.SideEffectLevel,
		"stdout":            result.Stdout,
		"stderr":            result.Stderr,
	})
	return result, nil
}

func exitCodeFromError(err error) int {
	if err == nil {
		return 0
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			return status.ExitStatus()
		}
	}
	return -1
}

func truncateMiddle(s string, limit int) string {
	if limit <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= limit {
		return s
	}
	half := limit / 2
	return string(runes[:half]) + "\n...[truncated]...\n" + string(runes[len(runes)-half:])
}

// filteredEnv 返回剔除 GLOOP_* 前缀变量后的环境变量切片。
// GLOOP_* 是 gloop 注入给 agent 的 syscall 上下文，不应泄漏进平台验证
// 命令（go test/go build 等）的子进程，否则仓库测试会被污染。
func filteredEnv(env []string) []string {
	out := make([]string, 0, len(env))
	for _, kv := range env {
		if strings.HasPrefix(kv, "GLOOP_") {
			continue
		}
		out = append(out, kv)
	}
	return out
}
