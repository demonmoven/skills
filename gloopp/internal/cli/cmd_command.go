package cli

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
	"code.byted.org/lihuanyu.0w0/gloop/internal/platformtools"
)

func runCommandCmd(ctx context.Context, log *slog.Logger, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "用法: gloop command <list|run>")
		return 2
	}
	switch args[0] {
	case "list":
		return runCommandList(log, args[1:])
	case "run":
		return runCommandRun(ctx, log, args[1:])
	default:
		fmt.Fprintf(os.Stderr, "未知 command 子命令 %q\n", args[0])
		return 2
	}
}

func runCommandList(log *slog.Logger, args []string) int {
	fs := flag.NewFlagSet("command list", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	_, _, root, _, code := loadAgentQuest(log)
	if code != 0 {
		return code
	}
	cfg, err := root.LoadConfig()
	if err != nil {
		log.Error(logPrefix+" 加载配置失败", "err", err)
		return 1
	}
	return writeJSONLine(map[string]any{"ok": true, "items": cfg.CommandAllowlist})
}

func runCommandRun(ctx context.Context, log *slog.Logger, args []string) int {
	fs := flag.NewFlagSet("command run", flag.ContinueOnError)
	extra := fs.String("extra-args", "", "额外参数，按空格拆分；仅 allow_extra_args=true 的命令可用")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "用法: gloop command run <command_id> [--extra-args \"...\"]")
		return 2
	}
	agentCtx, workDir, root, q, code := loadAgentQuest(log)
	if code != 0 {
		return code
	}
	if code := requireAgentClass(agentCtx, "mage", "command run"); code != 0 {
		return code
	}
	cfg, err := root.LoadConfig()
	if err != nil {
		log.Error(logPrefix+" 加载配置失败", "err", err)
		return 1
	}
	allowed, ok := fsstore.FindAllowedCommand(cfg.CommandAllowlist, fs.Arg(0))
	if !ok {
		return writeJSONError(4, "command_not_allowed", "命令不在 Mage 白名单: "+fs.Arg(0))
	}
	extraArgs := fieldsOrEmpty(*extra)
	if len(extraArgs) > 0 && !allowed.AllowExtraArgs {
		return writeJSONError(4, "extra_args_denied", "命令 "+allowed.ID+" 不允许 extra args")
	}
	commandWorkDir := strings.TrimSpace(os.Getenv("GLOOP_COMMAND_WORKDIR"))
	if commandWorkDir == "" {
		commandWorkDir = workDir
	}
	result, err := runAllowedCommandProcess(ctx, allowed, extraArgs, commandWorkDir)
	if err != nil {
		return writeJSONError(1, "command_run_failed", err.Error())
	}
	if err := fsstore.NewQuestStore(root).AppendEvent(q.ID, &fsstore.QuestEventRow{
		Timestamp: fsstore.NowMs(),
		Type:      "micro.command_run",
		QuestID:   q.ID,
		SessionID: agentCtx.SessionID,
		Payload:   result,
	}); err != nil {
		log.Error(logPrefix+" command run 事件写入失败", "err", err)
		return 1
	}
	return writeJSONLine(map[string]any{"ok": result.ExitCode == 0 && !result.TimedOut, "result": result})
}

func fieldsOrEmpty(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return strings.Fields(s)
}

func runAllowedCommandProcess(ctx context.Context, allowed fsstore.AllowedCommand, extraArgs []string, workDir string) (platformtools.CommandRunResult, error) {
	allowed, ok := fsstore.FindAllowedCommand([]fsstore.AllowedCommand{allowed}, allowed.ID)
	if !ok {
		return platformtools.CommandRunResult{}, fmt.Errorf("allowed command 无效")
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
	cmd.Dir = workDir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err := cmd.Run()
	durationMs := time.Since(start).Milliseconds()
	timedOut := runCtx.Err() == context.DeadlineExceeded
	exitCode := cliExitCode(err)
	if timedOut && exitCode == 0 {
		exitCode = -1
	}
	return platformtools.CommandRunResult{
		CommandID:  allowed.ID,
		Argv:       argv,
		ExitCode:   exitCode,
		DurationMs: durationMs,
		Stdout:     truncateCLICommandOutput(stdout.String()),
		Stderr:     truncateCLICommandOutput(stderr.String()),
		TimedOut:   timedOut,
	}, nil
}

func cliExitCode(err error) int {
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

func truncateCLICommandOutput(s string) string {
	const limit = 16 * 1024 // 按 rune 计，近似 16KB 量级避免刷屏
	runes := []rune(s)
	if len(runes) <= limit {
		return s
	}
	half := limit / 2
	return string(runes[:half]) + "\n...[truncated]...\n" + string(runes[len(runes)-half:])
}
