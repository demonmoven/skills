package events

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// 简单 in-memory fan-out EventBus：单 goroutine Publish，多个订阅方 channel。
// MVP 单机单用户足够，后续可换成 NATS/Redis pubsub。

type EventType string

const (
	// Quest 生命周期
	EvtQuestCreated   EventType = "quest.created"
	EvtQuestStarted   EventType = "quest.started"
	EvtQuestSuccess   EventType = "quest.success"
	EvtQuestFailed    EventType = "quest.failed"
	EvtQuestCancelled EventType = "quest.cancelled"
	EvtQuestBlocked   EventType = "quest.blocked"
	EvtQuestSpawned   EventType = "quest.spawned"

	// 阶段与评审
	EvtPhaseChanged      EventType = "quest.phase_changed"
	EvtReviewSubmitted   EventType = "quest.review_submitted"
	EvtUserReview        EventType = "quest.user_review"
	EvtQuestWaitingInput EventType = "quest.waiting_input"
	EvtRework            EventType = "quest.rework"
	EvtQuestApplied      EventType = "quest.applied"
	EvtQuestApplyFailed  EventType = "quest.apply_failed"
	EvtQuestDiscarded    EventType = "quest.discarded"
	EvtQuestNote         EventType = "quest.note"
	EvtWorkflowUpgraded  EventType = "quest.workflow_upgraded"
	EvtNeedsHuman        EventType = "needs_human.created"

	// Connector audit
	EvtConnectorStarted EventType = "connector.started"
	EvtConnectorSkipped EventType = "connector.skipped"
	EvtConnectorDone    EventType = "connector.done"
	EvtConnectorFailed  EventType = "connector.failed"

	// 工作区
	EvtWorkspaceCreated   EventType = "workspace.created"
	EvtWorkspaceDiffReady EventType = "workspace.diff_ready"

	// 失败归因
	EvtFailureAttributed EventType = "failure.attributed"

	// Micro 级别（实时流）
	EvtMicroTurn       EventType = "micro.turn"
	EvtTokenDelta      EventType = "micro.token_delta"
	EvtAssistantMsg    EventType = "micro.assistant_msg"
	EvtToolStart       EventType = "micro.tool_start"
	EvtToolEnd         EventType = "micro.tool_end"
	EvtCommandRun      EventType = "micro.command_run"
	EvtProgressUpdated EventType = "micro.progress"

	// Agent runtime health（v0.4）：用户可见的运行健康信号，不等同于语义进度。
	EvtAgentSessionCreated  EventType = "runtime.session.created"
	EvtAgentProcessStarted  EventType = "runtime.process.started"
	EvtAgentOutputFirst     EventType = "runtime.output.first_seen"
	EvtAgentOutputHeartbeat EventType = "runtime.output.heartbeat"
	EvtAgentProtocolSignal  EventType = "runtime.protocol.signal_seen"
	EvtAgentIdleWarning     EventType = "runtime.idle_warning"
	EvtAgentProtocolWarning EventType = "runtime.protocol_warning"
	EvtAgentProcessExited   EventType = "runtime.process.exited"
	EvtAgentErrorClassified EventType = "runtime.error.classified"

	// Agent transport selection audit.
	EvtAgentTransportSelected          EventType = "agent.transport_selected"
	EvtAgentExternalStateWriteObserved EventType = "agent.external_state_write_observed"

	// Agent 发帖（Feed 单源：agent 显式声明要发的帖）
	EvtAgentPost EventType = "agent.post"

	// 用户上下文
	EvtContextUpdated EventType = "context.updated"
)

type Event struct {
	ID        int64           `json:"id,omitempty"`
	GlobalID  int64           `json:"global_id,omitempty"`
	Type      EventType       `json:"type"`
	QuestID   string          `json:"quest_id,omitempty"`
	SessionID string          `json:"session_id,omitempty"`
	Payload   json.RawMessage `json:"payload"`
	Timestamp int64           `json:"ts"`
}

// EventBus 抽象事件总线能力。
// MVP 用 in-memory fan-out；未来可替换为 NATS/Redis PubSub。
type EventBus interface {
	Subscribe() (ch chan Event, unsubscribe func())
	Publish(e Event)
}

type Bus struct {
	mu   sync.RWMutex
	subs map[chan Event]struct{}
}

func NewBus() *Bus {
	return &Bus{subs: map[chan Event]struct{}{}}
}

func (b *Bus) Subscribe() (ch chan Event, unsubscribe func()) {
	ch = make(chan Event, 512)
	b.mu.Lock()
	b.subs[ch] = struct{}{}
	b.mu.Unlock()
	return ch, func() {
		b.mu.Lock()
		delete(b.subs, ch)
		b.mu.Unlock()
		close(ch)
	}
}

func (b *Bus) Publish(e Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.subs {
		// 非阻塞写：订阅方慢了就丢（MVP 前端只看最新状态）
		select {
		case ch <- e:
		default:
			fmt.Fprintf(os.Stderr, "[warn] event bus: subscriber slow, dropped event type=%s qid=%s\n", e.Type, e.QuestID)
		}
	}
}

func MarshalPayload(v any) json.RawMessage {
	if v == nil {
		return json.RawMessage("{}")
	}
	b, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage(`{"_error":"marshal"}`)
	}
	return b
}
