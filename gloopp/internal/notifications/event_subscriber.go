package notifications

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"sync"
	"time"

	"code.byted.org/lihuanyu.0w0/gloop/internal/events"
)

// QuestStore 是通知模块需要的 quest 查询接口（DIP：依赖抽象，不依赖具体实现）。
type QuestStore interface {
	LoadQuest(qid string) (QuestInfo, error)
}

type RecoveryInfo struct {
	Status        string
	PolicyName    string
	LastAction    string
	BlockReason   string
	OriginalError string
	LastError     string
}

type RecoveryStore interface {
	LoadRecovery(qid string) (RecoveryInfo, error)
}

const notificationDedupWindow = 5 * time.Minute

// EventSubscriber 订阅事件总线，自动发送系统级飞书通知。
// 只对用户关心的终态/需要用户介入的状态发通知，避免刷屏。
type EventSubscriber struct {
	bus      events.EventBus
	notifier Notifier
	store    QuestStore
	log      *slog.Logger

	unsubscribe func()

	mu        sync.Mutex
	now       func() time.Time
	dedupSeen map[string]time.Time
}

// NewEventSubscriber 创建事件订阅器。
// 不会自动启动，需调用 Start()。
func NewEventSubscriber(bus events.EventBus, notifier Notifier, store QuestStore, log *slog.Logger) *EventSubscriber {
	if log == nil {
		log = slog.Default()
	}
	return &EventSubscriber{
		bus:       bus,
		notifier:  notifier,
		store:     store,
		log:       log,
		now:       time.Now,
		dedupSeen: map[string]time.Time{},
	}
}

// Start 开始订阅事件总线。
// 非阻塞：后台 goroutine 消费事件。
func (s *EventSubscriber) Start() {
	if s.bus == nil || s.notifier == nil {
		return
	}
	ch, unsub := s.bus.Subscribe()
	s.unsubscribe = unsub

	go s.loop(ch)
}

// Stop 停止订阅。
func (s *EventSubscriber) Stop() {
	if s.unsubscribe != nil {
		s.unsubscribe()
		s.unsubscribe = nil
	}
}

// loop 事件消费循环。
func (s *EventSubscriber) loop(ch <-chan events.Event) {
	for ev := range ch {
		s.handleEvent(ev)
	}
}

// handleEvent 处理单个事件，决定是否发通知。
func (s *EventSubscriber) handleEvent(ev events.Event) {
	if s.notifier == nil || !s.notifier.Available() {
		return
	}
	decision := (NotificationPolicy{}).Decide(ev)
	if decision.Action != NotificationSend {
		s.log.Debug("通知策略跳过事件", "type", ev.Type, "policy", decision.PolicyName, "reason", decision.Reason)
		return
	}
	if s.shouldDedup(ev) {
		s.log.Debug("通知策略去重事件", "type", ev.Type, "qid", ev.QuestID)
		return
	}

	// 只处理 quest 级别的关键事件
	switch ev.Type {
	case events.EvtQuestBlocked:
		s.sendQuestBlocked(ev)
	case events.EvtUserReview:
		s.sendQuestReview(ev)
	case events.EvtQuestWaitingInput:
		s.sendQuestReview(ev)
	case events.EvtQuestApplied:
		s.sendQuestImpact(ev)
	case events.EvtQuestSuccess:
		// success 走影响通知（环 4：自主闭环感知）
		s.sendQuestImpact(ev)
	case events.EvtQuestFailed:
		s.sendQuestFailed(ev)
	}
}

func (s *EventSubscriber) shouldDedup(ev events.Event) bool {
	if ev.QuestID == "" {
		return false
	}
	key := ev.QuestID + ":" + string(ev.Type)
	now := s.now()
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, seenAt := range s.dedupSeen {
		if now.Sub(seenAt) > notificationDedupWindow {
			delete(s.dedupSeen, k)
		}
	}
	if seenAt, ok := s.dedupSeen[key]; ok && now.Sub(seenAt) <= notificationDedupWindow {
		return true
	}
	s.dedupSeen[key] = now
	return false
}

func (s *EventSubscriber) loadQuestInfo(qid string) (QuestInfo, bool) {
	if s.store == nil {
		return QuestInfo{}, false
	}
	info, err := s.store.LoadQuest(qid)
	if err != nil {
		s.log.Warn("通知：读取 quest 信息失败", "qid", qid, "error", err)
		return QuestInfo{}, false
	}
	return info, true
}

// sendQuestImpact 发送"影响通知"（HOTL v0.2 环 4）。
// 优先用 agent 主动声明的 ImpactSummary，没有才回落到 Comment。
func (s *EventSubscriber) sendQuestImpact(ev events.Event) {
	info, ok := s.loadQuestInfo(ev.QuestID)
	if !ok {
		return
	}
	summary := buildImpactSummaryText(info)
	card := BuildQuestImpactCard(
		QuestInfo{
			ID:          info.ID,
			ShortID:     info.ShortID,
			Title:       info.Title,
			Status:      info.Status,
			ReworkCount: info.ReworkCount,
			DurationMs:  info.DurationMs,
		},
		summary,
		s.notifier.BaseURL(),
	)
	ok, msg := s.notifier.SendCard(context.Background(), card)
	if !ok {
		s.log.Warn("飞书通知发送失败（quest.applied 影响通知）", "qid", ev.QuestID, "error", msg)
	} else {
		s.log.Info("飞书通知已发送（quest.applied 影响通知）", "qid", ev.QuestID)
	}
}

func buildImpactSummaryText(info QuestInfo) string {
	if !info.Impact.IsEmpty() {
		var b strings.Builder
		if info.Impact.WhatChanged != "" {
			b.WriteString("做了什么：")
			b.WriteString(info.Impact.WhatChanged)
		}
		if len(info.Impact.Affected) > 0 {
			if b.Len() > 0 {
				b.WriteString("\n")
			}
			b.WriteString("波及：")
			b.WriteString(strings.Join(info.Impact.Affected, "、"))
		}
		if len(info.Impact.NotTouched) > 0 {
			if b.Len() > 0 {
				b.WriteString("\n")
			}
			b.WriteString("没碰：")
			b.WriteString(strings.Join(info.Impact.NotTouched, "、"))
		}
		if len(info.Impact.Caveats) > 0 {
			if b.Len() > 0 {
				b.WriteString("\n")
			}
			b.WriteString("注意：")
			b.WriteString(strings.Join(info.Impact.Caveats, "；"))
		}
		return b.String()
	}
	// HOTL 要求 agent 用 --impact 声明影响；空值是异常，标注让人感知「该有却没有」
	if strings.TrimSpace(info.Comment) != "" {
		return "⚠️ agent 未声明影响声明，以下为评论：\n" + info.Comment
	}
	return "⚠️ agent 未声明影响声明"
}

func (s *EventSubscriber) sendQuestFailed(ev events.Event) {
	info, ok := s.loadQuestInfo(ev.QuestID)
	if !ok {
		return
	}

	// 从 payload 里提取原因
	reason := info.Comment
	var payload map[string]any
	if err := json.Unmarshal(ev.Payload, &payload); err == nil {
		if r, ok := payload["reason"].(string); ok && r != "" {
			reason = r
		} else if c, ok := payload["comment"].(string); ok && c != "" {
			reason = c
		}
	}

	card := BuildQuestFailedCard(
		QuestInfo{
			ID:          info.ID,
			ShortID:     info.ShortID,
			Title:       info.Title,
			Status:      info.Status,
			ReworkCount: info.ReworkCount,
			DurationMs:  info.DurationMs,
		},
		reason,
		s.notifier.BaseURL(),
	)

	ok, msg := s.notifier.SendCard(context.Background(), card)
	if !ok {
		s.log.Warn("飞书通知发送失败（quest.failed）", "qid", ev.QuestID, "error", msg)
	} else {
		s.log.Info("飞书通知已发送（quest.failed）", "qid", ev.QuestID)
	}
}

func (s *EventSubscriber) sendQuestBlocked(ev events.Event) {
	info, ok := s.loadQuestInfo(ev.QuestID)
	if !ok {
		return
	}

	// 从 payload 里提取阻塞原因
	reason := info.Comment
	var payload map[string]any
	if err := json.Unmarshal(ev.Payload, &payload); err == nil {
		if r, ok := payload["reason"].(string); ok && r != "" {
			reason = formatBlockReason(r)
		}
		if d, ok := payload["detail"].(string); ok && d != "" {
			if reason != "" {
				reason += "\n" + d
			} else {
				reason = d
			}
		}
	}

	card := BuildQuestBlockedCard(
		QuestInfo{
			ID:          info.ID,
			ShortID:     info.ShortID,
			Title:       info.Title,
			Status:      info.Status,
			ReworkCount: info.ReworkCount,
			DurationMs:  info.DurationMs,
		},
		reason,
		s.notifier.BaseURL(),
	)

	ok, msg := s.notifier.SendCard(context.Background(), card)
	if !ok {
		s.log.Warn("飞书通知发送失败（quest.blocked）", "qid", ev.QuestID, "error", msg)
	} else {
		s.log.Info("飞书通知已发送（quest.blocked）", "qid", ev.QuestID)
	}
}

func (s *EventSubscriber) sendQuestReview(ev events.Event) {
	info, ok := s.loadQuestInfo(ev.QuestID)
	if !ok {
		return
	}

	// 从 payload 里提取评论
	comment := info.Comment
	var payload map[string]any
	if err := json.Unmarshal(ev.Payload, &payload); err == nil {
		if c, ok := payload["comment"].(string); ok && c != "" {
			comment = c
		}
		// 如果是 agent 问问题，把问题拼进去
		if q, ok := payload["question"].(string); ok && q != "" {
			if comment != "" {
				comment = q + "\n\n" + comment
			} else {
				comment = q
			}
		}
	}
	if summary := s.recoverySummary(ev.QuestID); summary != "" {
		if comment != "" {
			comment = summary + "\n\n" + comment
		} else {
			comment = summary
		}
	}

	card := BuildQuestReviewCard(
		QuestInfo{
			ID:          info.ID,
			ShortID:     info.ShortID,
			Title:       info.Title,
			Status:      info.Status,
			ReworkCount: info.ReworkCount,
			DurationMs:  info.DurationMs,
		},
		comment,
		s.notifier.BaseURL(),
	)

	ok, msg := s.notifier.SendCard(context.Background(), card)
	if !ok {
		s.log.Warn("飞书通知发送失败（quest.user_review）", "qid", ev.QuestID, "error", msg)
	} else {
		s.log.Info("飞书通知已发送（quest.user_review）", "qid", ev.QuestID)
	}
}

func (s *EventSubscriber) recoverySummary(qid string) string {
	store, ok := s.store.(RecoveryStore)
	if !ok {
		return ""
	}
	info, err := store.LoadRecovery(qid)
	if err != nil || info.Status != "succeeded" || info.LastAction != "retry" {
		return ""
	}
	summary := "**自动恢复摘要：**\n"
	if info.PolicyName != "" {
		summary += "- 策略：" + info.PolicyName + "\n"
	}
	if info.BlockReason != "" {
		summary += "- 原阻塞：" + info.BlockReason + "\n"
	}
	if info.OriginalError != "" {
		summary += "- 原始错误：" + info.OriginalError + "\n"
	}
	summary += "- 结果：已恢复并进入人工审核"
	return summary
}

// formatBlockReason 把机器可读的 reason code 转成人话。
func formatBlockReason(reason string) string {
	switch reason {
	case "duration_exceeded":
		return "执行超时（超出最大时长限制）"
	case "user_confirm_timeout":
		return "用户确认超时"
	case "service_interruption_cannot_recover":
		return "服务中断后无法恢复"
	case "phase_budget_exceeded":
		return "阶段预算超支"
	default:
		return reason
	}
}
