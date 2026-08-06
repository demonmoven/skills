package executor

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

// ==================== Mock Executor（无需真实 agent 环境也能跑通闭环） ====================

type MockExecutor struct {
	*BaseExecutor
	mu         sync.RWMutex
	sessions   map[string][]Message
	sessionCfg map[string]SessionConfig
}

func NewMockExecutor(execID string) *MockExecutor {
	base := NewBase(execID, "模拟引擎（调试用）", model.AgentTypeMock)
	base.capabs = []Capability{CapToolUse, CapStreaming, CapContextExport}
	base.tier = CapabilityTierTest
	base.models = []ModelSpec{
		{ID: "mock-slow", Name: "Mock 慢速", Context: 64000},
		{ID: "mock-fast", Name: "Mock 快速", Context: 32000},
	}
	base.defModel = "mock-fast"
	return &MockExecutor{
		BaseExecutor: base,
		sessions:     map[string][]Message{},
		sessionCfg:   map[string]SessionConfig{},
	}
}

func (m *MockExecutor) CreateSession(ctx context.Context, sid string, opts SessionConfig) (*SessionHandle, error) {
	execKey := sessionKey(opts.QuestID, sid)
	m.mu.Lock()
	m.sessions[execKey] = []Message{{Role: "system", Content: opts.SystemPrompt}}
	m.sessionCfg[execKey] = opts
	activeCount := len(m.sessions)
	m.mu.Unlock()
	m.Heartbeat(activeCount)
	return &SessionHandle{SessionID: execKey, CreatedAt: time.Now()}, nil
}

func (m *MockExecutor) CloseSession(ctx context.Context, sid string) error {
	m.mu.Lock()
	delete(m.sessions, sid)
	delete(m.sessionCfg, sid)
	activeCount := len(m.sessions)
	m.mu.Unlock()
	m.Heartbeat(activeCount)
	return nil
}

// SessionConfig returns the config for a session (for testing).
func (m *MockExecutor) SessionConfig(sid string) (SessionConfig, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cfg, ok := m.sessionCfg[sid]
	return cfg, ok
}

func (m *MockExecutor) GetSessionStatus(ctx context.Context, sid string) (interface{}, error) {
	m.mu.RLock()
	h, ok := m.sessions[sid]
	m.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("session 不存在: %s", sid)
	}
	return map[string]interface{}{"turns": len(h)}, nil
}

func (m *MockExecutor) SendMessage(ctx context.Context, sid string, msg Message, model string, stream chan<- StreamChunk) (*ChatResponse, error) {
	m.mu.Lock()
	h, ok := m.sessions[sid]
	if !ok {
		m.mu.Unlock()
		return nil, fmt.Errorf("session 不存在: %s", sid)
	}
	h = append(h, msg)
	m.sessions[sid] = h
	cfg := m.sessionCfg[sid]
	m.mu.Unlock()

	turn := len(h) / 2
	assistantMsg, phaseSig := buildMockResponse(msg.Content, turn, h, cfg)

	if stream != nil {
		for _, line := range strings.SplitAfter(assistantMsg.Content, "\n") {
			time.Sleep(time.Duration(rand.Intn(20)) * time.Millisecond)
			SafeStream(stream, StreamChunk{SessionID: sid, Delta: line})
		}
	}

	m.mu.Lock()
	h, ok = m.sessions[sid]
	if !ok {
		m.mu.Unlock()
		return nil, fmt.Errorf("session 不存在: %s", sid)
	}
	h = append(h, assistantMsg)
	m.sessions[sid] = h
	activeCount := len(m.sessions)
	m.mu.Unlock()
	m.Heartbeat(activeCount)

	resp := &ChatResponse{
		SessionID:    sid,
		Message:      assistantMsg,
		TokenInput:   int64(approxTokens(msg.Content)),
		TokenOutput:  int64(approxTokens(assistantMsg.Content)),
		DurationMs:   int64(rand.Intn(1200) + 300),
		FinishReason: detectFinishReason(assistantMsg.Content),
	}
	if phaseSig != nil {
		resp.PhaseSignal = phaseSig
	}
	return resp, nil
}

// buildMockResponse 根据回合数和用户输入生成更真实的 mock 回复。
// Mock 不暴露 Gloop 平台工具；阶段结束用结构化文本表达，由 orchestrator
// 的 structured fallback 解析。真实 agent 主路径是通过 Bash 调用 gloop CLI。
func buildMockResponse(userPrompt string, turn int, history []Message, cfg SessionConfig) (Message, *PhaseSignalData) {
	lower := strings.ToLower(userPrompt)
	allContext := strings.ToLower(historyText(history) + "\n" + userPrompt)

	// 识别任务类型
	isCode := strings.Contains(lower, "代码") || strings.Contains(lower, "函数") ||
		strings.Contains(lower, "实现") || strings.Contains(lower, "build") ||
		strings.Contains(lower, "编译") || strings.Contains(lower, "go ") ||
		strings.Contains(lower, "bug") || strings.Contains(lower, "修复") ||
		strings.Contains(lower, "重构")

	isResearch := strings.Contains(lower, "调研") || strings.Contains(lower, "分析") ||
		strings.Contains(lower, "对比") || strings.Contains(lower, "方案") ||
		strings.Contains(lower, "评估")

	// v0.3: ReadOnly no longer indicates review; detect from system prompt and context
	isReviewPhase := strings.Contains(strings.ToLower(cfg.SystemPrompt), "mage") || strings.Contains(strings.ToLower(cfg.SystemPrompt), "checker") || strings.Contains(strings.ToLower(cfg.SystemPrompt), "评审")
	isReview := isReviewPhase && (strings.Contains(allContext, "评审") || strings.Contains(allContext, "review") ||
		strings.Contains(allContext, "验收") || strings.Contains(allContext, "review_quest"))

	// 识别"应该用哪条 gloop CLI syscall 完成阶段"的信号：
	//  - 兼容旧协议名（review_quest / phase_checkpoint）
	//  - 识别 gloop CLI 文案
	wantPhaseDone := strings.Contains(lower, "phase_checkpoint") ||
		(strings.Contains(lower, "`gloop`") && strings.Contains(lower, "phase")) ||
		strings.Contains(lower, `"phase","done"`) ||
		strings.Contains(lower, `"phase","fail"`) ||
		strings.Contains(lower, "gloop phase done")

	wantReviewVerdict := isReviewPhase && (strings.Contains(lower, "review_quest") ||
		(strings.Contains(lower, "`gloop`") && strings.Contains(lower, "review")) ||
		strings.Contains(lower, `"review","pass"`) ||
		strings.Contains(lower, `"review","request_changes"`) ||
		strings.Contains(lower, `"review","reject"`) ||
		strings.Contains(lower, "gloop review pass"))

	// 检测是否是工具结果回灌
	isToolResult := strings.Contains(userPrompt, "tool_result") ||
		strings.Contains(userPrompt, "工具执行结果") ||
		len(history) > 0 && history[len(history)-1].Role == "tool"

	content := ""
	var toolCalls []ToolCall
	var phaseSig *PhaseSignalData

	if wantReviewVerdict {
		content = `评审完成。

## 评审摘要
- 交付物满足原始委托
- 未发现阻塞性问题
- 后续可继续由用户终审
`
		phaseSig = &PhaseSignalData{
			OK:           true,
			Message:      "评审结论已提交",
			PhaseEnded:   true,
			PhaseVerdict: "pass",
			PhaseComment: "Mock 评审通过：交付物满足原始委托，未发现阻塞性问题。",
			PhaseScore:   8,
		}
	} else if wantPhaseDone {
		content = `任务已完成。

## 交付摘要
- 核心功能已实现
- 基础验证通过
- 代码风格与现有代码一致
`
		phaseSig = &PhaseSignalData{
			OK:           true,
			Message:      "阶段结论已提交",
			PhaseEnded:   true,
			PhaseVerdict: "done",
			PhaseComment: fmt.Sprintf("第 %d 阶段完成：%s", turn, truncate(userPrompt, 50)),
		}
	} else if isToolResult {
		// 工具结果回来后，总结一下然后继续
		switch turn % 3 {
		case 0:
			content = "工具执行完成，结果符合预期。继续推进下一阶段任务。"
		case 1:
			content = "已完成当前步骤，下面进行验证。"
		default:
			content = "步骤已完成，产出已确认。"
		}
	} else if turn == 0 {
		// 第一轮：分析任务；真实 executor 会在内部用 native tools 探索，mock 只写文本。
		if isCode {
			content = fmt.Sprintf(`好的，我来处理这个任务。

## 任务分析
- 目标：%s
- 类型：代码实现
- 预估回合：3-4 回合

先模拟探索项目结构，找到相关文件。
`, truncate(userPrompt, 120))
		} else if isResearch {
			content = fmt.Sprintf(`收到任务，开始调研。

## 调研方向
- 主题：%s
- 方法：对比主流方案 + 适用性分析

先模拟收集相关信息。
`, truncate(userPrompt, 100))
		} else if isReview {
			content = "好的，我来进行评审。先模拟查看相关代码和文档。"
		} else {
			content = fmt.Sprintf(`收到任务：%s

让我先了解一下当前项目的状态。
`, truncate(userPrompt, 100))
		}
	} else if turn <= 2 {
		// 中间回合：执行核心工作
		if isCode {
			content = `找到了相关文件。下面开始实现核心逻辑。

## 实现方案
- 保持接口兼容
- 新增 fallback 机制
- 保留原有行为作为默认路径

具体修改见工具调用。`
		} else if isResearch {
			content = `收集到一些信息，整理如下：

## 调研结果摘要

### 方案 A
- 优点：成熟稳定，生态丰富
- 缺点：较重，定制化成本高

### 方案 B
- 优点：轻量灵活
- 缺点：需要更多自研

### 建议
根据当前 MVP 阶段需求，推荐方案 B。`
		} else if isReview {
			content = `## 评审结论

### 优点
- 架构清晰，分层合理
- 接口设计简洁

### 待改进
- 错误处理可以更细致
- 部分函数缺少注释

总体质量：B+，建议通过（有小改动）。`
		} else {
			content = "继续执行中。当前进度正常，预计下回合可以完成。"
		}
	} else {
		// 后面的回合：收尾 + 模拟 CLI 阶段信号
		if isReview {
			content = `评审完成。

## 评审摘要
- 交付物满足原始委托
- 未发现阻塞性问题
- 后续可继续由用户终审
`
			phaseSig = &PhaseSignalData{
				OK:           true,
				Message:      "评审结论已提交",
				PhaseEnded:   true,
				PhaseVerdict: "pass",
				PhaseComment: "Mock 评审通过：交付物满足原始委托，未发现阻塞性问题。",
				PhaseScore:   8,
			}
		} else if strings.Contains(lower, "完成") || turn >= 4 {
			content = `任务已完成。

## 交付摘要
- 核心功能已实现
- 基础验证通过
- 代码风格与现有代码一致

申请评审。
`
			phaseSig = &PhaseSignalData{
				OK:           true,
				Message:      "阶段结论已提交",
				PhaseEnded:   true,
				PhaseVerdict: "done",
				PhaseComment: fmt.Sprintf("第 %d 阶段完成：%s", turn, truncate(userPrompt, 50)),
			}
		} else {
			content = "当前阶段任务完成，进入检查点。"
			phaseSig = &PhaseSignalData{
				OK:           true,
				Message:      "阶段结论已提交",
				PhaseEnded:   true,
				PhaseVerdict: "done",
				PhaseComment: fmt.Sprintf("第 %d 阶段完成：%s", turn, truncate(userPrompt, 50)),
			}
		}
	}

	return Message{
		Role:      "assistant",
		Content:   content,
		ToolCalls: toolCalls,
	}, phaseSig
}

func historyText(history []Message) string {
	var sb strings.Builder
	for _, m := range history {
		sb.WriteString(m.Role)
		sb.WriteString("\n")
		sb.WriteString(m.Content)
		sb.WriteString("\n")
		for _, tc := range m.ToolCalls {
			sb.WriteString(tc.ToolName)
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

func (m *MockExecutor) Interrupt(ctx context.Context, sid, reason string) error { return nil }
func (m *MockExecutor) Resume(ctx context.Context, sid string) error            { return nil }
func (m *MockExecutor) ExportContext(ctx context.Context, sid string) ([]Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	h, ok := m.sessions[sid]
	if !ok {
		return nil, fmt.Errorf("session 不存在: %s", sid)
	}
	out := make([]Message, len(h))
	copy(out, h)
	return out, nil
}
func (m *MockExecutor) ImportContext(ctx context.Context, sid string, history []Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[sid] = append([]Message{}, history...)
	return nil
}
func (m *MockExecutor) GetAvailableTools(ctx context.Context, sid string) ([]ToolDef, error) {
	return []ToolDef{
		{Name: "Bash", Description: "执行 shell 命令", Params: []ToolParam{{Name: "command", Type: "string", Required: true}}},
		{Name: "Read", Description: "读取文件", Params: []ToolParam{{Name: "path", Type: "string", Required: true}}},
	}, nil
}

// ==================== 两个 Agent 共用的 Mock 回复生成器 ====================

func MockDraftResponse(userPrompt string, originErr error) string {
	header := "【模拟冒险者 · 交付稿】\n\n"
	errNote := ""
	if originErr != nil {
		errNote = fmt.Sprintf("⚠️ 真实执行器不可用（%v），以下为模拟引擎生成。请先安装或配置可用执行器以获得真实结果。\n\n", originErr)
	}
	common := fmt.Sprintf(`> 收到用户指令：
> %s

## 一、方案拆解（WBS）

1. **理解需求**：确认目标、边界、依赖
2. **最小可行实现**：按 80/20 原则先跑通核心
3. **验证**：至少 1 个可运行的验收路径
4. **交付**：给出使用说明 + 下一步建议

## 二、执行记录

- 回合 1 需求分析 —— 已完成
- 回合 2 方案设计 —— 已完成
- 回合 3 代码落地 —— 已完成（模拟）
- 回合 4 自查自测 —— 已完成（模拟）

## 三、交付物清单

| 路径 | 说明 |
|---|---|
| cmd/gloop/main.go  | CLI 入口（含 start/stop/restart/status/logs 守护命令）|
| cmd/gloopd/main.go | 兼容入口（已合并到 gloop start，保留仅防老脚本失效）|
| cmd/guildctl/main.go | CLI 入口 |
| internal/model/model.go | 5 张核心表 ORM |
| internal/prompt/prompt.go | 职业模板 + Prompt 装配管线 |
| internal/executor/*.go | Executor 统一接口 + ACP/CLI/Mock 实现 |
| internal/orchestrator/*.go | 三层 Loop 引擎（核心：Loop Engineering） |
| web/src/* | 响应式 Dashboard（含移动端） |

## 四、使用方法

`+"```"+`bash
# 初始化（首次运行可选；`+"`"+`gloop start`+"`"+` 会自动初始化缺失文件）
gloop init

# 启动服务端 + Web Dashboard（后台 + 自动弹页 + 开机自启）
gloop start
# 或前台模式：gloop start --no-detach
# 重启（改 config / 升版本 / 改 agent 后）：gloop restart
# 停止：gloop stop
# 看状态 / Dashboard 地址：gloop status

# 发委托（示例：把 README 翻译成英文）
gloop quest run --title "翻译 README" --desc "把仓库根目录 README 翻译成中文" --category build --rank B
`+"```"+`

## 五、下一步建议

- 接入真实 Relay / Codex / TraeX 等 Agent
- 接入 GitHub Issue / 飞书文档 作为委托来源
- 引入真正的多 Agent 协作（队长+队员分工）
- 接入移动端 PWA 支持（离线缓存 + 推送）
`, truncate(userPrompt, 300))
	return header + errNote + common
}
