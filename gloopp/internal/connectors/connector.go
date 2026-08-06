// Package connectors 实现 HOTL v0.2 环 4 的 Connectors。
//
// 设计意图：quest 自主闭环（checker pass + apply）后，人不再审批内容，
// 但某些副作用（如代码改动入库）需要 connector 把"影响"落到外部系统。
// ConnectorSubscriber 订阅 EvtQuestApplied，按 quest 继承的 automation
// connectors 配置派发给具体 Connector（如 GitConnector）。
//
// 安全约束（重要）：
//   - 所有 connector 默认 OFF，必须在引擎配置里显式启用
//   - GitConnector 永远不直接推 base 分支，只推 quest 专属分支
//   - 所有操作 best-effort，失败只记录日志，不影响 quest 已闭环的状态
package connectors

import (
	"context"
	"log/slog"
	"sync"

	"code.byted.org/lihuanyu.0w0/gloop/internal/events"
)

// AppliedEvent 是 connector 处理"quest 已 apply"事件需要的全部上下文。
// 由 QuestLookup 从 quest 存储里组装后传入。
type AppliedEvent struct {
	QuestID       string
	ShortID       string
	Title         string
	WorkspacePath string   // quest 工作区路径（git 在这里跑）
	WorkingDir    string   // base 工作目录
	Connectors    []string // quest 继承自 automation 的 connector 名单
	FilesChanged  int
	Payload       map[string]any // 原始事件 payload
}

// QuestLookup 让 connector 反查 quest 信息（DIP：不直接依赖 fsstore）。
type QuestLookup interface {
	LookupForConnector(qid string) (AppliedEvent, error)
}

// Connector 是具体 connector 的接口。Name() 对应 automation config 里
// connectors 字段的值（如 "git"）。
type Connector interface {
	Name() string
	// Enabled 是否启用。未启用的 connector 不会被派发。
	Enabled() bool
	// HandleApplied 处理一个已 apply 的 quest。必须 best-effort：
	// 返回 error 只用于记录，不会回滚 quest 状态。
	HandleApplied(ctx context.Context, ev AppliedEvent) error
}

// ConnectorSubscriber 订阅 EventBus，把 EvtQuestApplied 派发给已注册的 connector。
type ConnectorSubscriber struct {
	bus        events.EventBus
	lookup     QuestLookup
	connectors map[string]Connector
	log        *slog.Logger

	unsubscribe func()
	wg          sync.WaitGroup
}

// NewConnectorSubscriber 创建订阅器。不会自动启动，需调用 Start()。
func NewConnectorSubscriber(bus events.EventBus, lookup QuestLookup, log *slog.Logger) *ConnectorSubscriber {
	if log == nil {
		log = slog.Default()
	}
	return &ConnectorSubscriber{
		bus:        bus,
		lookup:     lookup,
		connectors: map[string]Connector{},
		log:        log,
	}
}

// Register 注册一个 connector。同名后者覆盖前者。
func (s *ConnectorSubscriber) Register(c Connector) {
	if c == nil {
		return
	}
	s.connectors[c.Name()] = c
}

// Start 开始订阅。非阻塞。
func (s *ConnectorSubscriber) Start() {
	if s.bus == nil || s.lookup == nil || len(s.connectors) == 0 {
		s.log.Debug("connector subscriber 未启动（bus/lookup/connectors 缺失）")
		return
	}
	ch, unsub := s.bus.Subscribe()
	s.unsubscribe = unsub
	s.wg.Add(1)
	go s.loop(ch)
}

// Stop 停止订阅，等待 in-flight 派发完成。
func (s *ConnectorSubscriber) Stop() {
	if s.unsubscribe != nil {
		s.unsubscribe()
		s.unsubscribe = nil
	}
	s.wg.Wait()
}

func (s *ConnectorSubscriber) loop(ch <-chan events.Event) {
	defer s.wg.Done()
	for ev := range ch {
		if ev.Type != events.EvtQuestApplied {
			continue
		}
		s.dispatch(ev)
	}
}

// dispatch 把单个 EvtQuestApplied 派发给 quest 声明的 connector。
func (s *ConnectorSubscriber) dispatch(ev events.Event) {
	ae, err := s.lookup.LookupForConnector(ev.QuestID)
	if err != nil {
		s.log.Warn("connector: 反查 quest 失败", "qid", ev.QuestID, "err", err)
		return
	}
	if len(ae.Connectors) == 0 {
		return // quest 未声明任何 connector
	}
	ctx := context.Background()
	for _, name := range ae.Connectors {
		c, ok := s.connectors[name]
		if !ok {
			s.log.Debug("connector: 未注册，跳过", "qid", ev.QuestID, "connector", name)
			s.publishAudit(events.EvtConnectorSkipped, ev.QuestID, name, map[string]any{"reason": "not_registered"})
			continue
		}
		if !c.Enabled() {
			s.log.Debug("connector: 未启用，跳过", "qid", ev.QuestID, "connector", name)
			s.publishAudit(events.EvtConnectorSkipped, ev.QuestID, name, map[string]any{"reason": "disabled"})
			continue
		}
		s.log.Info("connector: 派发", "qid", ev.QuestID, "connector", name)
		s.publishAudit(events.EvtConnectorStarted, ev.QuestID, name, map[string]any{"files_changed": ae.FilesChanged})
		if err := c.HandleApplied(ctx, ae); err != nil {
			s.log.Warn("connector: 处理失败", "qid", ev.QuestID, "connector", name, "err", err)
			s.publishAudit(events.EvtConnectorFailed, ev.QuestID, name, map[string]any{"error": err.Error()})
			continue
		}
		s.publishAudit(events.EvtConnectorDone, ev.QuestID, name, map[string]any{"files_changed": ae.FilesChanged})
	}
}

func (s *ConnectorSubscriber) publishAudit(typ events.EventType, qid, connector string, payload map[string]any) {
	if s == nil || s.bus == nil {
		return
	}
	if payload == nil {
		payload = map[string]any{}
	}
	payload["connector"] = connector
	s.bus.Publish(events.Event{Type: typ, QuestID: qid, Payload: events.MarshalPayload(payload)})
}
