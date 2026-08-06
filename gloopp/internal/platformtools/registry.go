package platformtools

// CommandRunResult 是平台白名单命令的执行结果。
// 由 orchestrator.command_runner 产出、CLI cmd_command 消费，跨层共享因此留在这个包。
type CommandRunResult struct {
	CommandID  string   `json:"command_id"`
	Argv       []string `json:"argv"`
	ExitCode   int      `json:"exit_code"`
	DurationMs int64    `json:"duration_ms"`
	Stdout     string   `json:"stdout,omitempty"`
	Stderr     string   `json:"stderr,omitempty"`
	TimedOut   bool     `json:"timed_out,omitempty"`
}
