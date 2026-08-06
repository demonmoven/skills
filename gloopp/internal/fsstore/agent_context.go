package fsstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

const (
	AgentContextRel = ".gloop/context.json"
	SignalDirRel    = ".gloop/signals"
)

// AgentContext 是 agent 进程从工作区读取的运行时上下文。
//
// 编排层在每次触发 agent 子进程前，把当前委托/阶段/冒险者信息写入工作区的
// .gloop/context.json，agent 通过 `gloop phase*` / `gloop review` 等 CLI
// 命令再从中回读，从而把「agent 沙箱进程」与「gloop 编排主线」串联起来。
// 也可以通过 GLOOP_* 环境变量等价注入（见 agentContextFromEnv）。
type AgentContext struct {
	DataDir         string `json:"data_dir"`
	QuestID         string `json:"quest_id"`
	SessionID       string `json:"session_id"`
	Phase           int    `json:"phase"`
	PhaseName       string `json:"phase_name"`
	AdventurerID    string `json:"adventurer_id"`
	AdventurerClass string `json:"adventurer_class"`
	AgentID         string `json:"agent_id,omitempty"`
	PhaseRole       string `json:"phase_role,omitempty"`
	WorkspacePath   string `json:"workspace_path"`
	// SignalWorkspacePath is where phase-end signal files are written/read.
	// It can differ from WorkspacePath for read-only sandbox phases: commands
	// execute in the sandbox, but the orchestrator watches the real quest workdir.
	SignalWorkspacePath string `json:"signal_workspace_path,omitempty"`
	CreatedAtMs         int64  `json:"created_at_ms"`
}

// PhaseSignal 是 agent→gloop 的阶段信号文件。
//
// v0.3 起剑士/法师通过 `gloop phase done` / `gloop review` 把阶段结论写成 JSON
// 信号文件，编排线程在 micro loop 下一轮读入后推进 quest 状态机。
// 与 orchestrator.PhaseSignal 同语义：fsstore 版本作为磁盘持久化结构，
// 多了 CreatedAtMs 便于读取端按 sid 去重。
type PhaseSignal struct {
	OK           bool           `json:"ok"`
	Message      string         `json:"message,omitempty"`
	Data         map[string]any `json:"data,omitempty"`
	PhaseEnded   bool           `json:"phase_ended"`
	PhaseVerdict string         `json:"phase_verdict,omitempty"`
	PhaseComment string         `json:"phase_comment,omitempty"`
	PhaseHints   string         `json:"phase_hints,omitempty"`
	PhaseScore   int            `json:"phase_score,omitempty"`
	CreatedAtMs  int64          `json:"created_at_ms"`
}

// WriteAgentContext 向 workDir/.gloop/context.json 写入 agent 运行时上下文。
// 若 ctx.CreatedAtMs 为 0 则自动填充当前时间；WorkspacePath 统一为传入 workDir。
func WriteAgentContext(workDir string, ctx AgentContext) error {
	if workDir == "" {
		return errors.New("workDir is empty")
	}
	ctx.WorkspacePath = workDir
	if ctx.CreatedAtMs == 0 {
		ctx.CreatedAtMs = NowMs()
	}
	return WriteJSON(filepath.Join(workDir, AgentContextRel), &ctx)
}

// DiscoverAgentContext 定位当前 quest 的 agent 上下文。
//
// 优先级：
//  1. GLOOP_CONTEXT 环境变量（直接指向 context.json 绝对路径）
//  2. GLOOP_DATA_DIR / GLOOP_QUEST_ID 等一组环境变量（agentContextFromEnv）
//  3. 从 startDir 开始逐级向上查找 .gloop/context.json；startDir 为空时用当前工作目录
//
// 返回值：上下文、该 quest 的工作目录根、错误（找不到或读取失败）。
func DiscoverAgentContext(startDir string) (*AgentContext, string, error) {
	if env := os.Getenv("GLOOP_CONTEXT"); env != "" {
		ctx, err := ReadJSON[AgentContext](env)
		if err != nil {
			return nil, "", err
		}
		return ctx, filepath.Dir(filepath.Dir(env)), nil
	}
	if ctx, workDir, ok := agentContextFromEnv(); ok {
		return ctx, workDir, nil
	}
	if startDir == "" {
		var err error
		startDir, err = os.Getwd()
		if err != nil {
			return nil, "", err
		}
	}
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return nil, "", err
	}
	for {
		p := filepath.Join(dir, AgentContextRel)
		ctx, err := ReadJSON[AgentContext](p)
		if err == nil {
			return ctx, dir, nil
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, "", err
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return nil, "", fmt.Errorf("未找到 %s；请在 quest 工作区内运行，或设置 GLOOP_CONTEXT", AgentContextRel)
}

func agentContextFromEnv() (*AgentContext, string, bool) {
	dataDir := os.Getenv("GLOOP_DATA_DIR")
	qid := os.Getenv("GLOOP_QUEST_ID")
	sid := os.Getenv("GLOOP_SESSION_ID")
	workDir := os.Getenv("GLOOP_WORKSPACE_PATH")
	if dataDir == "" || qid == "" || sid == "" || workDir == "" {
		return nil, "", false
	}
	phaseIdx, _ := strconv.Atoi(os.Getenv("GLOOP_PHASE_INDEX"))
	return &AgentContext{
		DataDir:             dataDir,
		QuestID:             qid,
		SessionID:           sid,
		Phase:               phaseIdx,
		PhaseName:           os.Getenv("GLOOP_PHASE"),
		AdventurerID:        os.Getenv("GLOOP_ADVENTURER_ID"),
		AdventurerClass:     os.Getenv("GLOOP_ADVENTURER_CLASS"),
		AgentID:             os.Getenv("GLOOP_AGENT_ID"),
		PhaseRole:           os.Getenv("GLOOP_PHASE_ROLE"),
		WorkspacePath:       workDir,
		SignalWorkspacePath: os.Getenv("GLOOP_SIGNAL_WORKSPACE_PATH"),
	}, workDir, true
}

// WritePhaseSignal 把阶段信号写到 workDir/.gloop/signals/<sid>.json，
// 编排侧循环按 SessionID 消费。CreatedAtMs 未填时自动补当前时间。
func WritePhaseSignal(workDir string, sid string, sig PhaseSignal) error {
	if workDir == "" {
		return errors.New("workDir is empty")
	}
	if sid == "" {
		return errors.New("session id is empty")
	}
	if sig.CreatedAtMs == 0 {
		sig.CreatedAtMs = NowMs()
	}
	dir := filepath.Join(workDir, SignalDirRel)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return WriteJSON(filepath.Join(dir, sid+".json"), &sig)
}

// ReadPhaseSignal 按 session id 读取阶段信号。workDir 或 sid 为空时直接返回 os.ErrNotExist。
func ReadPhaseSignal(workDir string, sid string) (*PhaseSignal, error) {
	if workDir == "" || sid == "" {
		return nil, os.ErrNotExist
	}
	return ReadJSON[PhaseSignal](filepath.Join(workDir, SignalDirRel, sid+".json"))
}

// RemovePhaseSignal 删除阶段信号文件（消费完成后清理）。文件不存在视为成功。
func RemovePhaseSignal(workDir string, sid string) error {
	if workDir == "" || sid == "" {
		return nil
	}
	err := os.Remove(filepath.Join(workDir, SignalDirRel, sid+".json"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

// JSONString 把 PhaseSignal 序列化成紧凑 JSON，用于日志/事件 payload 打印。
func (s *PhaseSignal) JSONString() string {
	raw, err := json.Marshal(s)
	if err != nil {
		return "{}"
	}
	return string(raw)
}
