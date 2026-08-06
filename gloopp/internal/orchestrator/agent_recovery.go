package orchestrator

import (
	"context"
	"os/exec"
	"strings"
	"time"

	"code.byted.org/lihuanyu.0w0/gloop/internal/executor"
)

type agentRecoverySpec struct {
	matchNames              []string
	nonInteractiveCommand   []string
	nonInteractiveTimeout   time.Duration
	nonInteractiveSuccess   string
	userRecoveryActionHints []string
	userActions             map[string][]string
	userActionTimeout       time.Duration
}

var agentRecoveryRegistry = []agentRecoverySpec{
	{
		matchNames:              []string{"relay"},
		nonInteractiveCommand:   []string{"relay", "auth", "status", "--json"},
		nonInteractiveTimeout:   5 * time.Second,
		nonInteractiveSuccess:   "relay auth status succeeded; retrying agent turn",
		userRecoveryActionHints: []string{"relay auth status --json", "relay auth login --sso"},
		userActionTimeout:       60 * time.Second,
		userActions: map[string][]string{
			"relay_auth_status": {"relay", "auth", "status", "--json"},
			"relay_auth_login":  {"relay", "auth", "login", "--sso"},
		},
	},
	{
		matchNames:              []string{"traex", "trae"},
		nonInteractiveCommand:   []string{"traex", "login", "status"},
		nonInteractiveTimeout:   5 * time.Second,
		nonInteractiveSuccess:   "traex login status succeeded; retrying agent turn",
		userRecoveryActionHints: []string{"traex login status"},
		userActionTimeout:       15 * time.Second,
		userActions: map[string][]string{
			"traex_login_status": {"traex", "login", "status"},
		},
	},
}

func recoverySpecForExecutor(ex executor.Executor) *agentRecoverySpec {
	name := strings.ToLower(ex.Name() + " " + ex.ID())
	for i := range agentRecoveryRegistry {
		spec := &agentRecoveryRegistry[i]
		for _, needle := range spec.matchNames {
			if strings.Contains(name, needle) {
				return spec
			}
		}
	}
	return nil
}

func tryNonInteractiveAuthRecovery(ctx context.Context, ex executor.Executor) (string, bool) {
	spec := recoverySpecForExecutor(ex)
	if spec == nil || len(spec.nonInteractiveCommand) == 0 {
		return "", false
	}
	if runNonInteractiveAuthCommand(ctx, spec.nonInteractiveTimeout, spec.nonInteractiveCommand[0], spec.nonInteractiveCommand[1:]...) {
		return spec.nonInteractiveSuccess, true
	}
	return "", false
}

func runNonInteractiveAuthCommand(parent context.Context, timeout time.Duration, name string, args ...string) bool {
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.Run() == nil
}

func recoveryActionsForExecutor(ex executor.Executor) []string {
	spec := recoverySpecForExecutor(ex)
	if spec == nil {
		return nil
	}
	return append([]string(nil), spec.userRecoveryActionHints...)
}

func runUserRecoveryAction(ctx context.Context, action string) (string, error) {
	for _, spec := range agentRecoveryRegistry {
		cmdline, ok := spec.userActions[action]
		if !ok {
			continue
		}
		timeout := spec.userActionTimeout
		if timeout <= 0 {
			timeout = 30 * time.Second
		}
		if len(cmdline) == 0 {
			return "", nil
		}
		runCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		cmd := exec.CommandContext(runCtx, cmdline[0], cmdline[1:]...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return strings.TrimSpace(string(out)), err
		}
		return strings.TrimSpace(string(out)), nil
	}
	return "", errUnknownRecoveryAction(action)
}

type unknownRecoveryActionError string

func (e unknownRecoveryActionError) Error() string {
	return "unknown recovery action: " + string(e)
}

func errUnknownRecoveryAction(action string) error {
	return unknownRecoveryActionError(action)
}
