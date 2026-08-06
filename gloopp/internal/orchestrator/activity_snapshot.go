package orchestrator

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"code.byted.org/lihuanyu.0w0/gloop/internal/events"
	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

// activitySnapshotUpdater 订阅 quest 终态事件，被动维护 dim_activity_snapshot 上下文维度。
// 这是全局活动快照：记录当前开放任务、近期完成、阻塞项等，
// 让 morning_briefing / context_refresh 等自动化无需重新推断。
type activitySnapshotUpdater struct {
	engine      *Engine
	unsub       func()
	stopCh      chan struct{}
	done        chan struct{}
	mu          sync.Mutex
	running     bool
	lastWriteMs int64
}

const activitySnapshotDimName = "activity_snapshot"
const activitySnapshotDebounceMs = 5000 // 5 秒内多次事件只写一次

// StartActivitySnapshotUpdater 启动 activity snapshot 后台更新器。幂等。
func (e *Engine) StartActivitySnapshotUpdater(ctx context.Context) {
	if e.activitySnapshot == nil {
		e.activitySnapshot = &activitySnapshotUpdater{engine: e}
	}
	e.activitySnapshot.Start(ctx)
}

// StopActivitySnapshotUpdater 停止 activity snapshot 后台更新器。幂等。
func (e *Engine) StopActivitySnapshotUpdater() {
	if e.activitySnapshot != nil {
		e.activitySnapshot.Stop()
	}
}

func (u *activitySnapshotUpdater) Start(ctx context.Context) {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.running {
		return
	}

	ch, unsub := u.engine.bus.Subscribe()
	u.unsub = unsub
	u.stopCh = make(chan struct{})
	u.done = make(chan struct{})
	u.running = true

	go u.loop(ctx, ch)
	u.engine.log.Info("[ActivitySnapshot] updater 已启动")
}

func (u *activitySnapshotUpdater) Stop() {
	u.mu.Lock()
	defer u.mu.Unlock()
	if !u.running {
		return
	}
	close(u.stopCh)
	<-u.done
	u.unsub()
	u.running = false
}

func (u *activitySnapshotUpdater) loop(ctx context.Context, ch chan events.Event) {
	defer close(u.done)

	terminalEvents := map[events.EventType]bool{
		events.EvtQuestSuccess:   true,
		events.EvtQuestFailed:    true,
		events.EvtQuestCancelled: true,
		events.EvtQuestBlocked:   true,
		events.EvtUserReview:     true,
		events.EvtQuestCreated:   true,
		events.EvtQuestStarted:   true,
		events.EvtQuestSpawned:   true,
	}

	var pending bool
	debounce := time.NewTimer(time.Hour)
	debounce.Stop()

	for {
		select {
		case <-ctx.Done():
			if pending {
				u.refresh()
			}
			return
		case <-u.stopCh:
			if pending {
				u.refresh()
			}
			return
		case ev, ok := <-ch:
			if !ok {
				return
			}
			if terminalEvents[ev.Type] {
				now := fsstore.NowMs()
				if now-u.lastWriteMs < activitySnapshotDebounceMs {
					if !pending {
						pending = true
						debounce.Reset(time.Duration(activitySnapshotDebounceMs) * time.Millisecond)
					}
				} else {
					pending = false
					debounce.Stop()
					u.refresh()
				}
			}
		case <-debounce.C:
			if pending {
				pending = false
				u.refresh()
			}
		}
	}
}

func (u *activitySnapshotUpdater) refresh() {
	u.lastWriteMs = fsstore.NowMs()

	quests, err := u.engine.ListQuests()
	if err != nil {
		u.engine.log.Error("[ActivitySnapshot] ListQuests 失败", "err", err)
		return
	}

	var running, pending, blocked, recentSuccess, recentFailed int
	var openQuests []string
	var blockedQuests []string
	var recentCompletions []string

	cutoff24h := fsstore.NowMs() - 24*60*60*1000

	for _, q := range quests {
		switch q.Status {
		case model.QuestStatusRunning:
			running++
			openQuests = append(openQuests, fmt.Sprintf("- [%s] %s (running since %s)", q.ShortID, truncate(q.Query, 50), msToTime(q.StartedAtMs)))
		case model.QuestStatusPending:
			pending++
			openQuests = append(openQuests, fmt.Sprintf("- [%s] %s (pending)", q.ShortID, truncate(q.Query, 50)))
		case model.QuestStatusBlocked:
			blocked++
			blockedQuests = append(blockedQuests, fmt.Sprintf("- [%s] %s — %s", q.ShortID, truncate(q.Query, 40), q.BlockedReason))
		case model.QuestStatusUserReview:
			pending++
			openQuests = append(openQuests, fmt.Sprintf("- [%s] %s (awaiting review)", q.ShortID, truncate(q.Query, 50)))
		case model.QuestStatusSuccess:
			if q.CompletedAtMs > cutoff24h {
				recentSuccess++
				recentCompletions = append(recentCompletions, fmt.Sprintf("- [%s] %s (pass)", q.ShortID, truncate(q.Query, 50)))
			}
		case model.QuestStatusFailed:
			if q.CompletedAtMs > cutoff24h {
				recentFailed++
				recentCompletions = append(recentCompletions, fmt.Sprintf("- [%s] %s (failed)", q.ShortID, truncate(q.Query, 50)))
			}
		}
	}

	var buf strings.Builder
	buf.WriteString("# Activity Snapshot\n\n")
	buf.WriteString("## Scope\n\n")
	buf.WriteString(fmt.Sprintf("- 自动生成，每次 quest 状态变化时平台被动更新\n- 最后更新: %s\n\n", time.Now().Format("2006-01-02 15:04")))

	buf.WriteString("## Current Activity\n\n")
	buf.WriteString(fmt.Sprintf("| 状态 | 数量 |\n|---|---|\n| Running | %d |\n| Pending | %d |\n| Blocked | %d |\n| 24h Success | %d |\n| 24h Failed | %d |\n\n", running, pending, blocked, recentSuccess, recentFailed))

	if len(openQuests) > 0 {
		buf.WriteString("## Open Quests\n\n")
		for _, line := range openQuests {
			buf.WriteString(line + "\n")
		}
		buf.WriteString("\n")
	}

	if len(blockedQuests) > 0 {
		buf.WriteString("## Blocked\n\n")
		for _, line := range blockedQuests {
			buf.WriteString(line + "\n")
		}
		buf.WriteString("\n")
	}

	if len(recentCompletions) > 0 {
		buf.WriteString("## Recent Completions (24h)\n\n")
		for _, line := range recentCompletions {
			buf.WriteString(line + "\n")
		}
		buf.WriteString("\n")
	}

	if len(quests) == 0 {
		buf.WriteString("当前无任何委托。\n")
	}

	if err := u.engine.root.RenameContextDim("loop_state", activitySnapshotDimName); err != nil {
		u.engine.log.Warn("[ActivitySnapshot] 迁移 legacy loop_state dim 失败", "err", err)
	}
	if _, err := u.engine.root.WriteContextDim(activitySnapshotDimName, buf.String()); err != nil {
		u.engine.log.Error("[ActivitySnapshot] 写入 dim 失败", "err", err)
	}
}

func msToTime(ms int64) string {
	if ms == 0 {
		return "unknown"
	}
	return time.UnixMilli(ms).Format("01-02 15:04")
}
