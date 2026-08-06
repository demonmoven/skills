package fsstore

import (
	"fmt"
	"strings"
	"time"
)

// DefaultMageCommandTimeoutMs 是法师白名单命令的默认超时（2 分钟）。
const DefaultMageCommandTimeoutMs int64 = 2 * 60 * 1000

// AllowedCommand describes one deterministic command entry that a read-only
// reviewer may ask the platform to run. It is identified by ID so the agent
// cannot smuggle an arbitrary shell string through the platform ABI.
type AllowedCommand struct {
	ID              string   `json:"id"`
	Description     string   `json:"description"`
	Command         string   `json:"command"`
	Args            []string `json:"args,omitempty"`
	AllowExtraArgs  bool     `json:"allow_extra_args,omitempty"`
	TimeoutMs       int64    `json:"timeout_ms,omitempty"`
	SideEffectLevel string   `json:"side_effect_level,omitempty"` // L0/L1/L2；空=L1 for backward compatibility
}

// DefaultCommandAllowlist 返回内置的法师评审阶段可执行命令白名单。
// 覆盖 Go/JS/通用工具三大类，全部为纯读 L0/L1 命令，无外部副作用。
func DefaultCommandAllowlist() []AllowedCommand {
	return []AllowedCommand{
		// ---------- Go 生态 ----------
		{
			ID:              "go-test",
			Description:     "Run Go tests in the current workspace package tree.",
			Command:         "go",
			Args:            []string{"test", "./..."},
			AllowExtraArgs:  true,
			TimeoutMs:       DefaultMageCommandTimeoutMs,
			SideEffectLevel: "L0",
		},
		{
			ID:              "go-vet",
			Description:     "Run go vet to report suspicious code constructs.",
			Command:         "go",
			Args:            []string{"vet", "./..."},
			AllowExtraArgs:  false,
			TimeoutMs:       DefaultMageCommandTimeoutMs,
			SideEffectLevel: "L0",
		},
		{
			ID:              "go-build",
			Description:     "Verify the code compiles (build all packages, no output).",
			Command:         "go",
			Args:            []string{"build", "./..."},
			AllowExtraArgs:  false,
			TimeoutMs:       DefaultMageCommandTimeoutMs,
			SideEffectLevel: "L0",
		},
		{
			ID:              "go-fmt-check",
			Description:     "Check if Go code is properly formatted (gofmt -l).",
			Command:         "gofmt",
			Args:            []string{"-l", "."},
			AllowExtraArgs:  false,
			TimeoutMs:       DefaultMageCommandTimeoutMs,
			SideEffectLevel: "L0",
		},
		{
			ID:              "go-staticcheck",
			Description:     "Run staticcheck for advanced Go static analysis.",
			Command:         "staticcheck",
			Args:            []string{"./..."},
			AllowExtraArgs:  false,
			TimeoutMs:       DefaultMageCommandTimeoutMs,
			SideEffectLevel: "L0",
		},
		{
			ID:              "go-sec",
			Description:     "Run gosec security scanner for Go code.",
			Command:         "gosec",
			Args:            []string{"./..."},
			AllowExtraArgs:  false,
			TimeoutMs:       DefaultMageCommandTimeoutMs,
			SideEffectLevel: "L0",
		},
		{
			ID:              "go-mod-verify",
			Description:     "Verify go.mod dependencies are clean and consistent.",
			Command:         "go",
			Args:            []string{"mod", "verify"},
			AllowExtraArgs:  false,
			TimeoutMs:       DefaultMageCommandTimeoutMs,
			SideEffectLevel: "L0",
		},

		// ---------- JavaScript / Node 生态 ----------
		{
			ID:              "npm-test",
			Description:     "Run npm test in the current workspace.",
			Command:         "npm",
			Args:            []string{"test"},
			AllowExtraArgs:  false,
			TimeoutMs:       DefaultMageCommandTimeoutMs,
			SideEffectLevel: "L0",
		},
		{
			ID:              "npm-run-test",
			Description:     "Run npm run test in the current workspace.",
			Command:         "npm",
			Args:            []string{"run", "test"},
			AllowExtraArgs:  false,
			TimeoutMs:       DefaultMageCommandTimeoutMs,
			SideEffectLevel: "L0",
		},
		{
			ID:              "npm-lint",
			Description:     "Run npm lint for code style and quality checks.",
			Command:         "npm",
			Args:            []string{"run", "lint"},
			AllowExtraArgs:  false,
			TimeoutMs:       DefaultMageCommandTimeoutMs,
			SideEffectLevel: "L0",
		},
		{
			ID:              "npm-build-check",
			Description:     "Run npm build to verify the project builds successfully.",
			Command:         "npm",
			Args:            []string{"run", "build"},
			AllowExtraArgs:  false,
			TimeoutMs:       DefaultMageCommandTimeoutMs,
			SideEffectLevel: "L0",
		},
		{
			ID:              "npm-audit",
			Description:     "Run npm audit to check for dependency vulnerabilities.",
			Command:         "npm",
			Args:            []string{"audit", "--audit-level=moderate"},
			AllowExtraArgs:  false,
			TimeoutMs:       DefaultMageCommandTimeoutMs,
			SideEffectLevel: "L0",
		},

		// ---------- 通用工具 ----------
		{
			ID:              "git-diff-stat",
			Description:     "Show diff stats (changed files, insertions, deletions).",
			Command:         "git",
			Args:            []string{"diff", "--stat", "HEAD"},
			AllowExtraArgs:  false,
			TimeoutMs:       DefaultMageCommandTimeoutMs,
			SideEffectLevel: "L0",
		},
		{
			ID:              "git-diff-name-only",
			Description:     "List changed files only (no patch content).",
			Command:         "git",
			Args:            []string{"diff", "--name-only", "HEAD"},
			AllowExtraArgs:  false,
			TimeoutMs:       DefaultMageCommandTimeoutMs,
			SideEffectLevel: "L0",
		},
		{
			ID:              "git-log-summary",
			Description:     "Show recent commit history (last 20 commits, oneline).",
			Command:         "git",
			Args:            []string{"log", "--oneline", "-20"},
			AllowExtraArgs:  false,
			TimeoutMs:       DefaultMageCommandTimeoutMs,
			SideEffectLevel: "L0",
		},
	}
}

// FindAllowedCommand 按 ID 在白名单中查找条目，未找到返回 false。
// 找到的条目会补齐默认值（timeout、side_effect_level）。
func FindAllowedCommand(items []AllowedCommand, id string) (AllowedCommand, bool) {
	for _, item := range items {
		if item.ID == id {
			return item.withDefaults(), true
		}
	}
	return AllowedCommand{}, false
}

// ValidateAllowedCommands 校验白名单条目：ID 非空且不重复、命令为纯文件名、超时和副作用等级合法。
func ValidateAllowedCommands(items []AllowedCommand) error {
	seen := map[string]bool{}
	for _, item := range items {
		item = item.withDefaults()
		if item.ID == "" {
			return fmt.Errorf("allowed command id 不能为空")
		}
		if seen[item.ID] {
			return fmt.Errorf("allowed command id 重复: %s", item.ID)
		}
		seen[item.ID] = true
		if item.Command == "" {
			return fmt.Errorf("allowed command %s command 不能为空", item.ID)
		}
		if strings.ContainsAny(item.Command, "\x00/\\") {
			return fmt.Errorf("allowed command %s command 必须是可执行文件名，不能包含路径分隔符", item.ID)
		}
		if item.TimeoutMs < 0 {
			return fmt.Errorf("allowed command %s timeout_ms 不能为负数", item.ID)
		}
		if item.SideEffectLevel != "" && item.SideEffectLevel != "L0" && item.SideEffectLevel != "L1" && item.SideEffectLevel != "L2" {
			return fmt.Errorf("allowed command %s side_effect_level 必须是 L0/L1/L2", item.ID)
		}
		if _, err := time.ParseDuration(fmt.Sprintf("%dms", item.TimeoutMs)); err != nil {
			return fmt.Errorf("allowed command %s timeout_ms 非法: %w", item.ID, err)
		}
	}
	return nil
}

func (c AllowedCommand) withDefaults() AllowedCommand {
	if c.TimeoutMs == 0 {
		c.TimeoutMs = DefaultMageCommandTimeoutMs
	}
	if c.SideEffectLevel == "" {
		c.SideEffectLevel = "L1"
	}
	return c
}
