package executor

import (
	"context"
	"sync"
	"time"

	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

// ==================== Tool 定义 ====================

type ToolParam struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"` // string | int | bool | object | array
	Description string   `json:"description"`
	Required    bool     `json:"required"`
	Enum        []string `json:"enum,omitempty"`
}

type ToolDef struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Params      []ToolParam `json:"params"`
}

type ToolCall struct {
	ID        string                 `json:"id"`
	ToolName  string                 `json:"tool_name"`
	Arguments map[string]interface{} `json:"arguments"`
	// CLIInvocation holds the parsed argv slice for CLI-style tool calls
	// (Bash / Gloop) when the executor prefers argv over Arguments.
	CLIInvocation []string `json:"-"`
	// InvokedToolName records the concrete tool name used (Bash / Gloop / …)
	// so downstream routing can dispatch to the right runner.
	InvokedToolName string `json:"-"`
	Origin          string `json:"origin,omitempty"`
	Status          string `json:"status,omitempty"`
	Result          string `json:"result,omitempty"`
	RawEvent        string `json:"raw_event,omitempty"`
}

const (
	ToolOriginACPNative = "acp_native"
)

type ToolResult struct {
	CallID string      `json:"call_id"`
	Result interface{} `json:"result"`
	Error  string      `json:"error,omitempty"`
}

// ==================== Chat / Streaming 定义 ====================

type Message struct {
	Role       string         `json:"role"` // system | user | assistant | tool
	Content    string         `json:"content"`
	Parts      []ContentBlock `json:"parts,omitempty"`
	ToolCalls  []ToolCall     `json:"tool_calls,omitempty"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
	Meta       any            `json:"meta,omitempty"`
}

type ContentBlock struct {
	Type         string `json:"type"` // text | artifact
	Text         string `json:"text,omitempty"`
	ArtifactID   string `json:"artifact_id,omitempty"`
	Kind         string `json:"kind,omitempty"`
	MIME         string `json:"mime,omitempty"`
	Name         string `json:"name,omitempty"`
	Size         int64  `json:"size,omitempty"`
	SHA256       string `json:"sha256,omitempty"`
	StoragePath  string `json:"storage_path,omitempty"`
	AbsolutePath string `json:"absolute_path,omitempty"`
}

type ChatResponse struct {
	SessionID    string  `json:"session_id"`
	Message      Message `json:"message"`
	TokenInput   int64   `json:"token_input"`
	TokenOutput  int64   `json:"token_output"`
	DurationMs   int64   `json:"duration_ms"`
	FinishReason string  `json:"finish_reason"` // stop | tool_calls | length | error
	Error        string  `json:"error,omitempty"`
	Meta         any     `json:"meta,omitempty"` // cost_usd、model_usage、fallback 标记
	// PhaseSignal 由 mock executor 设置，模拟 `gloop phase done` / `gloop review` 的 CLI 信号。
	// 真实 executor（ACP/CLI）不设置此字段——它们通过 gloop CLI 写信号文件，由 consumePhaseSignal 消费。
	// 非空时 micro_loop 直接当作阶段结束信号处理。
	PhaseSignal *PhaseSignalData `json:"phase_signal,omitempty"`
}

type FileChangeObservation struct {
	Path   string `json:"path"`
	Kind   string `json:"kind,omitempty"`
	Tool   string `json:"tool,omitempty"`
	Status string `json:"status,omitempty"`
}

// PhaseSignalData 是 mock executor 模拟 CLI 阶段信号的数据结构。
// 字段与 fsstore.PhaseSignal 对齐，但定义在 executor 包避免循环依赖。
type PhaseSignalData struct {
	OK           bool   `json:"ok"`
	Message      string `json:"message,omitempty"`
	PhaseEnded   bool   `json:"phase_ended,omitempty"`
	PhaseVerdict string `json:"phase_verdict,omitempty"`
	PhaseComment string `json:"phase_comment,omitempty"`
	PhaseHints   string `json:"phase_hints,omitempty"`
	PhaseScore   int    `json:"phase_score,omitempty"`
	Data         any    `json:"data,omitempty"`
}

// StreamChunk 流式增量 chunk
type StreamChunk struct {
	SessionID string `json:"sid"`
	Delta     string `json:"delta"`
	Finish    bool   `json:"finish,omitempty"`
	Error     string `json:"error,omitempty"`
	Event     string `json:"event,omitempty"`
	Data      any    `json:"data,omitempty"`
}

type ModelSpec struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Context  int     `json:"context_window"`
	CostIn   float64 `json:"cost_per_1k_input"`
	CostOut  float64 `json:"cost_per_1k_output"`
	MaxSpeed string  `json:"max_speed"` // fast / standard / slow
}

type Capability string

const (
	CapToolUse       Capability = "tool_use"
	CapStreaming     Capability = "streaming"
	CapVision        Capability = "vision"
	CapCodeExec      Capability = "code_exec"
	CapContextExport Capability = "context_export"
)

type CapabilityTier string

const (
	CapabilityTierA    CapabilityTier = "tier_a"
	CapabilityTierB    CapabilityTier = "tier_b"
	CapabilityTierC    CapabilityTier = "tier_c"
	CapabilityTierTest CapabilityTier = "test_only"
)

// ==================== Executor 接口拆分（ISP：接口隔离原则） ====================
//
// 四大能力按职责拆分，调用方可以只依赖最小需要的接口：
//   - ExecutorMeta:     只读元信息
//   - SessionLifecycle: 会话创建/销毁/中断
//   - ChatExecutor:     对话推理 + 工具
//   - ContextExporter:  上下文导入/导出（审计追溯用）
//
// 组合接口 Executor 嵌入全部 4 个小接口，保持完全向后兼容。

// ExecutorMeta 抽象执行器的只读元信息能力。
type ExecutorMeta interface {
	ID() string
	Type() model.AgentType
	Name() string
	AvailableModels() []ModelSpec
	DefaultModel() string
	Capabilities() []Capability
	CapabilityTier() CapabilityTier
}

// SessionLifecycle 抽象会话生命周期管理。
type SessionLifecycle interface {
	CreateSession(ctx context.Context, sessionID string, opts SessionConfig) (*SessionHandle, error)
	CloseSession(ctx context.Context, sessionID string) error
	GetSessionStatus(ctx context.Context, sessionID string) (interface{}, error)
	Interrupt(ctx context.Context, sessionID string, reason string) error
	Resume(ctx context.Context, sessionID string) error
}

// ChatExecutor 抽象对话推理 + 工具能力（核心路径）。
type ChatExecutor interface {
	SendMessage(ctx context.Context, sessionID string, msg Message, model string, stream chan<- StreamChunk) (*ChatResponse, error)
	GetAvailableTools(ctx context.Context, sessionID string) ([]ToolDef, error)
}

// ContextExporter 抽象上下文导入/导出（可追溯性用）。
type ContextExporter interface {
	ExportContext(ctx context.Context, sessionID string) ([]Message, error)
	ImportContext(ctx context.Context, sessionID string, history []Message) error
}

// ==================== Executor 统一接口（南向标准契约，嵌入 4 个小接口） ====================

type Executor interface {
	ExecutorMeta
	SessionLifecycle
	ChatExecutor
	ContextExporter
}

type SessionConfig struct {
	SystemPrompt       string
	Model              string
	CapabilitiesNeeded []Capability
	MaxTurns           int
	Temperature        float64
	WorkingDir         string   // 工作目录（工具执行的 CWD）
	AllowedTools       []string // 允许的工具名（空=全部）
	ReadOnly           bool     // 只读模式（mage 评审时使用，禁止写文件和破坏性命令）
	Env                []string // 额外环境变量，格式 KEY=VALUE；用于注入 GLOOP_* syscall context
	QuestID            string   // 所属 quest ID；executor 用它 + sessionID 构造全局唯一的内存 map key，实现 per-quest 隔离
}

type SessionHandle struct {
	SessionID string
	CreatedAt time.Time
}

// sessionKey 构造 executor 内存 map 用的全局唯一 key。
// 文件系统 sessionID 只保证 quest 内唯一（如 warrior_0），
// executor 跨 quest 共享，必须加 questID 前缀才能避免冲突。
func sessionKey(questID, sessionID string) string {
	if questID == "" {
		return sessionID
	}
	return questID + "/" + sessionID
}

// ==================== 通用基类（减少每个 Agent 重复代码） ====================

type BaseExecutor struct {
	mu          sync.Mutex
	execID      string
	name        string
	agent       model.AgentType
	capabs      []Capability
	tier        CapabilityTier
	models      []ModelSpec
	defModel    string
	maxParallel int
	activeCount int
	lastBeat    time.Time
}

func NewBase(execID, name string, agent model.AgentType) *BaseExecutor {
	return &BaseExecutor{
		execID:      execID,
		name:        name,
		agent:       agent,
		maxParallel: 4,
		lastBeat:    time.Now(),
	}
}

func (b *BaseExecutor) ID() string            { return b.execID }
func (b *BaseExecutor) Type() model.AgentType { return b.agent }
func (b *BaseExecutor) Name() string          { return b.name }
func (b *BaseExecutor) DefaultModel() string  { return b.defModel }
func (b *BaseExecutor) CapabilityTier() CapabilityTier {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.tier != "" {
		return b.tier
	}
	switch b.agent {
	case model.AgentTypeACP:
		return CapabilityTierB
	case model.AgentTypeCLI:
		return CapabilityTierC
	case model.AgentTypeMock:
		return CapabilityTierTest
	default:
		return CapabilityTierC
	}
}
func (b *BaseExecutor) AvailableModels() []ModelSpec {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]ModelSpec(nil), b.models...)
}
func (b *BaseExecutor) Capabilities() []Capability {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]Capability(nil), b.capabs...)
}

func (b *BaseExecutor) SetAvailableModels(models []ModelSpec) {
	if len(models) == 0 {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.models = append([]ModelSpec(nil), models...)
}

func (b *BaseExecutor) Heartbeat(activeCount int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.activeCount = activeCount
	b.lastBeat = time.Now()
}

// ActiveCount 加锁读取 activeCount。
func (b *BaseExecutor) ActiveCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.activeCount
}

// LastBeat 加锁读取 lastBeat。
func (b *BaseExecutor) LastBeat() time.Time {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.lastBeat
}
