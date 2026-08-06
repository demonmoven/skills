package orchestrator

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"code.byted.org/lihuanyu.0w0/gloop/internal/executor"
	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

func (e *Engine) RegisterAgentExecutor(p *fsstore.AgentConfig) (executor.Executor, error) {
	ex, err := e.buildExecutorForAgent(p)
	if err != nil {
		return nil, err
	}
	e.RegisterExecutor(p.Name, ex)
	return ex, nil
}

func (e *Engine) UnregisterExecutor(agentName string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.execs, agentName)
}

func (e *Engine) RegisteredExecutor(agentName string) (executor.Executor, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	ex, ok := e.execs[agentName]
	return ex, ok
}

func (e *Engine) buildExecutorForAgent(p *fsstore.AgentConfig) (executor.Executor, error) {
	if p == nil {
		return nil, fmt.Errorf("agent config is nil")
	}
	env, err := p.EffectiveEnv()
	if err != nil {
		return nil, err
	}
	cmd := resolveAgentCommand(p.Command)
	if cmd == "" {
		return nil, fmt.Errorf("agent command 为空")
	}
	defModel := p.DefaultModel
	if defModel == "" {
		defModel = "auto"
	}
	switch p.Type {
	case model.AgentTypeACP:
		return executor.NewACPExecutor(executor.ACPOpt{
			ExecID:             "exe_" + p.Name,
			Command:            cmd,
			Args:               p.Args,
			Env:                env,
			DefaultModel:       defModel,
			WorkingRoot:        e.workDir,
			StartupModelEnvVar: acpStartupModelEnvVar(p, cmd),
			Name:               p.Name + " executor",
		}), nil
	case model.AgentTypeCLI:
		cliEx, err := buildCLIExecutor(cmd, defModel, e.workDir, p)
		if err != nil {
			return nil, err
		}
		// 把 agent 静态身份 env（CLAUDE_CONFIG_DIR / env_files / DiscoverClaudeEnv）
		// 注入 CLI executor 的共同基类。单一注入点：5 个 CLI executor 都内嵌
		// *CLIExecutorBase，由它在 CreateSession 时与 session 运行时 env 合并。
		// 这是修复 CLI 分支历史性丢弃 agent env 的根因（此前仅 ACP 分支透传 env）。
		if base := executor.ResolveCLIExecutorBase(cliEx); base != nil {
			base.SetAgentEnv(envMapToPairs(env))
		}
		return cliEx, nil
	default:
		return nil, fmt.Errorf("未知 agent 类型 %q", p.Type)
	}
}

// buildCLIExecutor 按命令 basename 选择具体的 CLI executor 适配器。
// env 注入由调用方统一处理（见 buildExecutorForAgent），这里只负责构造。
func buildCLIExecutor(cmd, defModel, workDir string, p *fsstore.AgentConfig) (executor.Executor, error) {
	switch strings.TrimSuffix(filepath.Base(cmd), filepath.Ext(cmd)) {
	case "relay", "claude":
		return executor.NewClaudeCodeExecutor(executor.ClaudeCodeOpt{
			ExecID:       "exe_" + p.Name,
			BinPath:      cmd,
			DefaultModel: defModel,
			WorkingRoot:  workDir,
		}), nil
	case "traex":
		return executor.NewTraeXExecutor(executor.TraeXOpt{
			ExecID:       "exe_" + p.Name,
			BinPath:      cmd,
			DefaultModel: defModel,
			WorkingRoot:  workDir,
		}), nil
	case "codex":
		return executor.NewCodexExecutor(executor.CodexOpt{
			ExecID:       "exe_" + p.Name,
			BinPath:      cmd,
			DefaultModel: defModel,
			WorkingRoot:  workDir,
		}), nil
	case "pi":
		return executor.NewPiExecutor(executor.PiOpt{
			ExecID:       "exe_" + p.Name,
			BinPath:      cmd,
			DefaultModel: defModel,
			WorkingRoot:  workDir,
		}), nil
	case "aiden":
		return executor.NewAidenExecutor(executor.AidenOpt{
			ExecID:       "exe_" + p.Name,
			BinPath:      cmd,
			DefaultModel: defModel,
			WorkingRoot:  workDir,
			Variant:      detectAidenVariant(p.Name, p.Args),
		}), nil
	case "hermes":
		return executor.NewHermesExecutor(executor.HermesOpt{
			ExecID:       "exe_" + p.Name,
			BinPath:      cmd,
			DefaultModel: defModel,
			WorkingRoot:  workDir,
		}), nil
	default:
		return nil, fmt.Errorf("CLI agent 暂无专用适配器: %s", cmd)
	}
}

// envMapToPairs 把 EffectiveEnv 的 map 转成 KEY=VALUE 切片，供 CLI executor
// 基类注入。顺序不重要——同名 key 在 map 里本就唯一，子进程 spawn 时再与
// session env 合并。
func envMapToPairs(env map[string]string) []string {
	if len(env) == 0 {
		return nil
	}
	pairs := make([]string, 0, len(env))
	for k, v := range env {
		pairs = append(pairs, k+"="+v)
	}
	return pairs
}

func resolveAgentCommand(cmd string) string {
	if cmd == "" {
		return ""
	}
	if filepath.IsAbs(cmd) || strings.Contains(cmd, "/") {
		return cmd
	}
	if p, err := exec.LookPath(cmd); err == nil && p != "" {
		return p
	}
	return cmd
}

func acpStartupModelEnvVar(p *fsstore.AgentConfig, resolvedCmd string) string {
	if isClaudeACPAgent(p, resolvedCmd) {
		return "ANTHROPIC_MODEL"
	}
	return ""
}

func isClaudeACPAgent(p *fsstore.AgentConfig, resolvedCmd string) bool {
	if p == nil {
		return false
	}
	if strings.EqualFold(p.Name, "claude") {
		return true
	}
	for _, arg := range p.Args {
		if strings.Contains(strings.ToLower(arg), "claude-agent-acp") {
			return true
		}
	}
	base := strings.TrimSuffix(filepath.Base(resolvedCmd), filepath.Ext(resolvedCmd))
	return strings.Contains(strings.ToLower(base), "claude")
}

func DetectAidenVariantForAgent(name string, args []string) executor.AidenVariant {
	if strings.Contains(name, "x_claude") || strings.Contains(name, "xclaude") {
		return executor.AidenVariantXClaude
	}
	if strings.Contains(name, "x_codex") || strings.Contains(name, "xcodex") {
		return executor.AidenVariantXCodex
	}
	for i := 0; i+1 < len(args); i++ {
		if args[i] == "x" {
			switch args[i+1] {
			case "claude":
				return executor.AidenVariantXClaude
			case "codex":
				return executor.AidenVariantXCodex
			}
		}
	}
	return executor.AidenVariantBase
}

func detectAidenVariant(name string, args []string) executor.AidenVariant {
	return DetectAidenVariantForAgent(name, args)
}
