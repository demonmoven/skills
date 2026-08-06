package orchestrator

import (
	"context"
	"fmt"

	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
)

// ProcessQuestEventAutomations consumes one quest's persisted event log for
// event-triggered automations. This is deliberately per-quest; cross-quest
// workflow automation needs a global event log and belongs to a later phase.
func (e *Engine) ProcessQuestEventAutomations(ctx context.Context, qid string) error {
	autos, err := e.root.ListAutomations()
	if err != nil {
		return err
	}
	eventsRows, err := fsstore.NewQuestStore(e.root).ReadEvents(qid, 0)
	if err != nil {
		return err
	}
	for _, auto := range autos {
		if !auto.Enabled || auto.Trigger != fsstore.TriggerEvent || auto.Event == "" {
			continue
		}
		state, _ := e.root.GetAutomationState(auto.ID, qid)
		if state == nil {
			state = &fsstore.AutomationState{AutomationID: auto.ID, QuestID: qid}
		}
		for _, row := range eventsRows {
			if row.ID <= state.LastEventID || row.Type != auto.Event {
				continue
			}
			if err := e.runEventAutomationAction(ctx, auto, qid, row); err != nil {
				return err
			}
			state.LastEventID = row.ID
			state.LastRunTs = fsstore.NowMs()
			state.RunCount++
			if err := e.root.SaveAutomationState(state); err != nil {
				return err
			}
		}
	}
	return nil
}

func (e *Engine) ProcessEventAutomations(ctx context.Context) error {
	if err := e.ProcessGlobalEventAutomations(ctx); err != nil {
		return err
	}
	items, err := fsstore.NewQuestStore(e.root).ListQuests()
	if err != nil {
		return err
	}
	for _, q := range items {
		if q == nil {
			continue
		}
		if err := e.ProcessQuestEventAutomations(ctx, q.ID); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) ProcessGlobalEventAutomations(ctx context.Context) error {
	autos, err := e.root.ListAutomations()
	if err != nil {
		return err
	}
	rows, err := e.root.ReadGlobalEvents(0)
	if err != nil {
		return err
	}
	for _, auto := range autos {
		if !auto.Enabled || auto.Trigger != fsstore.TriggerEvent || auto.Event == "" || auto.EventScope != "global" {
			continue
		}
		state, _ := e.root.GetAutomationState(auto.ID, "__global__")
		if state == nil {
			state = &fsstore.AutomationState{AutomationID: auto.ID, QuestID: "__global__"}
		}
		for _, row := range rows {
			if row.GlobalID <= state.LastGlobalID || row.Type != auto.Event {
				continue
			}
			if err := e.runGlobalEventAutomationAction(ctx, auto, row); err != nil {
				return err
			}
			state.LastGlobalID = row.GlobalID
			state.LastRunTs = fsstore.NowMs()
			state.RunCount++
			if err := e.root.SaveAutomationState(state); err != nil {
				return err
			}
		}
	}
	return nil
}

func (e *Engine) runEventAutomationAction(ctx context.Context, auto *fsstore.AutomationConfig, qid string, row fsstore.QuestEventRow) error {
	// HOTL v0.2: event_action=spawn_quest → 创建跨 quest 工作流
	if auto.EventAction == "spawn_quest" && auto.SpawnAutomation != "" {
		return e.spawnQuestFromEvent(ctx, auto, qid, row.Type)
	}
	switch {
	case auto.HasTag("triage"):
		e.publish(qid, "", "quest.note", map[string]any{
			"content": fmt.Sprintf("event automation %s handled %s#%d", auto.ID, row.Type, row.ID),
			"tag":     "event_triage",
		})
		return nil
	case auto.HasTag("notify"):
		e.publish(qid, "", "automation.event", map[string]any{
			"automation": auto.ID,
			"event":      row.Type,
			"event_id":   row.ID,
		})
		return nil
	default:
		e.publish(qid, "", "quest.note", map[string]any{
			"content": fmt.Sprintf("event automation %s observed %s#%d", auto.ID, row.Type, row.ID),
			"tag":     "event",
		})
		return nil
	}
}

func (e *Engine) runGlobalEventAutomationAction(ctx context.Context, auto *fsstore.AutomationConfig, row fsstore.GlobalEventRow) error {
	// HOTL v0.2: global event spawn_quest
	if auto.EventAction == "spawn_quest" && auto.SpawnAutomation != "" {
		return e.spawnQuestFromEvent(ctx, auto, row.QuestID, row.Type)
	}
	if row.QuestID == "" {
		e.publishSystem("automation.event", map[string]any{
			"automation": auto.ID,
			"global_id":  row.GlobalID,
			"event":      row.Type,
		})
		return nil
	}
	e.publish(row.QuestID, "", "automation.event", map[string]any{
		"automation": auto.ID,
		"global_id":  row.GlobalID,
		"event":      row.Type,
	})
	return nil
}

// spawnQuestFromEvent 触发 spawn_automation 指向的 automation 创建新 quest。
// 链式深度限制：通过 RunCount 间接控制（每个 event automation 的 RunCount
// 有 scheduler 层面的频率限制），且 spawn 的 automation 自身如果也是 event
// 触发，其 RunCount 会独立计数，不会无限递归。
func (e *Engine) spawnQuestFromEvent(ctx context.Context, auto *fsstore.AutomationConfig, sourceQid string, eventType string) error {
	spawnID := auto.SpawnAutomation
	spawnCfg, err := e.root.GetAutomation(spawnID)
	if err != nil {
		e.log.Error("event automation spawn 失败: 目标 automation 不存在", "source", auto.ID, "spawn", spawnID, "err", err)
		return nil
	}
	if !spawnCfg.Enabled {
		e.log.Warn("event automation spawn 跳过: 目标 automation 已禁用", "source", auto.ID, "spawn", spawnID)
		return nil
	}
	e.log.Info("event automation spawn_quest", "source", auto.ID, "spawn", spawnID, "event", eventType, "source_quest", sourceQid)
	if _, err := e.RunAutomation(ctx, spawnID); err != nil {
		e.log.Error("event automation spawn RunAutomation 失败", "source", auto.ID, "spawn", spawnID, "err", err)
	}
	return nil
}
