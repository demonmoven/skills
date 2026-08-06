package orchestrator

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

// ==================== 自动化调度器 ====================
//
// 每分钟 tick 一次，检查所有 enabled 的 schedule 类型 automation，
// 匹配 cron 表达式的就触发执行。
//
// 设计原则：
// - 单实例运行，不考虑分布式锁（单机应用）
// - 用 automation 的 LastRunMs 防止同一分钟内重复触发
// - 触发失败不阻塞其他 automation，打日志继续
// - 每个 automation 串行触发（同一分钟内多个 automation 也串行，避免瞬时压力）

type scheduler struct {
	engine *Engine
	log    *slog.Logger

	mu      sync.Mutex
	ticker  *time.Ticker
	stopCh  chan struct{}
	done    chan struct{}
	running bool

	// parsedCron 缓存已解析的 cron 表达式，减少每次 tick 的解析开销
	cronCache map[string]*cronSchedule
	cacheMu   sync.RWMutex
}

func newScheduler(e *Engine) *scheduler {
	return &scheduler{
		engine:    e,
		log:       e.log,
		cronCache: make(map[string]*cronSchedule),
	}
}

// ==================== Engine 对外方法 ====================

// StartScheduler 启动自动化调度器。幂等。
func (e *Engine) StartScheduler(ctx context.Context) {
	e.scheduler.Start(ctx)
}

// StopScheduler 停止自动化调度器。幂等。
func (e *Engine) StopScheduler() {
	e.scheduler.Stop()
}

// SchedulerRunning 返回调度器是否在运行
func (e *Engine) SchedulerRunning() bool {
	e.scheduler.mu.Lock()
	defer e.scheduler.mu.Unlock()
	return e.scheduler.running
}

// Start 启动调度器。幂等。
func (s *scheduler) Start(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return
	}

	s.ticker = time.NewTicker(1 * time.Minute) // 每分钟 tick 一次，够 cron 粒度了
	s.stopCh = make(chan struct{})
	s.done = make(chan struct{})
	s.running = true

	go s.loop(ctx)
	s.log.Info("[Scheduler] 调度器已启动")
}

// Stop 停止调度器。幂等。
func (s *scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	s.ticker.Stop()
	close(s.stopCh)
	<-s.done
	s.running = false
	s.log.Info("[Scheduler] 调度器已停止")
}

func (s *scheduler) loop(ctx context.Context) {
	defer close(s.done)

	// 启动时先跑一次（对齐到最近的分钟）
	s.tick(ctx)

	for {
		select {
		case <-s.ticker.C:
			s.tick(ctx)
		case <-s.stopCh:
			return
		}
	}
}

// tick 执行一次调度检查
func (s *scheduler) tick(ctx context.Context) {
	now := time.Now()
	s.appendSchedulerEvent("tick", "", map[string]any{"ts": now.UnixMilli()})
	// scheduler 审计日志轮转：超 10M 轮转，保留最近 1 份
	schedulerLog := s.engine.root.Sub(fsstore.SubdirWorkspace, "scheduler", "events.jsonl")
	if err := fsstore.RotateJSONLIfBig(schedulerLog, 10*1024*1024, 1); err != nil {
		s.log.Warn("[Scheduler] 审计日志轮转失败", "err", err)
	}
	s.blockExpiredUserReviews(now)
	// running/reviewing 超时兜底：覆盖 rework 路径等 goroutine 失踪场景
	// （launchRunningLoop 静默失败导致 quest 状态 running 但无人推进，
	// in-loop 超时检查无从触发）。实测 qst_2606199899 卡死 127h。
	s.blockExpiredRunningQuests(now)
	// 恢复 waiting_input 超时的 quest。此前仅在 server 启动时跑一次
	// (server.go startBackgroundRecoveries)，导致 1h 超时实际可能等数小时
	// 才被恢复。接入每分钟 tick 后，超时恢复延迟 ≤1min。
	s.engine.RecoverWaitingInputTimeouts(ctx)
	s.cleanupExpiredWorkspaces(now)
	s.cleanupStaleRecoveryStates()
	// 每小时清理一次孤儿 worktree + 分支（整点附近 5 分钟内）
	if now.Minute() < 5 {
		s.engine.CleanupOrphanWorkspaces(ctx)
	}
	if err := s.engine.ProcessEventAutomations(ctx); err != nil {
		s.log.Warn("[Scheduler] 处理事件自动化失败", "err", err)
		s.appendSchedulerEvent("event_automation_error", "", map[string]any{"error": err.Error()})
	}

	autos, err := s.engine.ListAutomations()
	if err != nil {
		s.log.Error("[Scheduler] 列 automation 失败", "err", err)
		s.appendSchedulerEvent("list_error", "", map[string]any{"error": err.Error()})
		return
	}

	// 精确到分钟的时间戳（用于判断是否已在同一分钟内跑过）
	minuteTs := now.Truncate(time.Minute).UnixMilli()

	for _, auto := range autos {
		if !auto.Enabled {
			s.appendSchedulerEvent("skip_disabled", auto.ID, nil)
			continue
		}
		if auto.Trigger != fsstore.TriggerSchedule {
			continue
		}
		if auto.Cron == "" {
			s.appendSchedulerEvent("skip_missing_cron", auto.ID, nil)
			continue
		}
		// 检查同一分钟内是否已触发过
		if auto.LastRunMs >= minuteTs {
			s.appendSchedulerEvent("skip_already_ran", auto.ID, map[string]any{"minute_ts": minuteTs})
			continue
		}

		// 解析 cron（带缓存）
		cron, err := s.getCron(auto.Cron)
		if err != nil {
			s.log.Warn("[Scheduler] cron 表达式解析失败，跳过",
				"id", auto.ID, "cron", auto.Cron, "err", err)
			s.appendSchedulerEvent("cron_error", auto.ID, map[string]any{"cron": auto.Cron, "error": err.Error()})
			continue
		}

		if !cron.Matches(now) {
			s.appendSchedulerEvent("skip_not_due", auto.ID, map[string]any{"cron": auto.Cron})
			continue
		}

		// 重叠防护：如果该 automation 的上一个 quest 仍处于活跃状态，跳过本次触发。
		if s.hasActiveQuestForAutomation(auto.ID) {
			s.log.Warn("[Scheduler] automation 已有活跃 quest，跳过", "id", auto.ID)
			s.appendSchedulerEvent("skip_overlap", auto.ID, nil)
			continue
		}

		// 触发执行
		s.log.Info("[Scheduler] 触发 automation", "id", auto.ID, "name", auto.Name)
		s.appendSchedulerEvent("trigger", auto.ID, map[string]any{"name": auto.Name})
		qid, runErr := s.engine.RunAutomation(ctx, auto.ID)
		if runErr != nil {
			s.log.Error("[Scheduler] 触发 automation 失败",
				"id", auto.ID, "err", runErr)
			s.appendSchedulerEvent("trigger_error", auto.ID, map[string]any{"error": runErr.Error()})
			continue
		}
		s.log.Info("[Scheduler] automation 触发成功", "id", auto.ID, "qid", qid)
		s.appendSchedulerEvent("triggered", auto.ID, map[string]any{"qid": qid})
	}
}

func (s *scheduler) cleanupExpiredWorkspaces(now time.Time) {
	cfg := s.engine.cfg
	if cfg.WorkspaceRetentionDays <= 0 {
		return
	}
	items, err := fsstore.NewQuestStore(s.engine.root).CleanupExpiredWorkspaces(
		cfg.WorkspaceRetentionDays,
		cfg.AutoCleanupFailedQuests,
		now.UnixMilli(),
		false,
	)
	if err != nil {
		s.log.Warn("[Scheduler] workspace cleanup 失败", "err", err)
		s.appendSchedulerEvent("cleanup_error", "", map[string]any{"error": err.Error()})
		return
	}
	if len(items) > 0 {
		s.appendSchedulerEvent("cleanup", "", map[string]any{"count": len(items), "items": items})
	}
}

// cleanupStaleRecoveryStates 清理终态 quest 残留的 recovery_state 文件。
// recovery_state 只对 running/blocked quest 有意义；quest 进终态后是垃圾。
// 也清理 meta 已不存在的孤儿文件。
func (s *scheduler) cleanupStaleRecoveryStates() {
	rdir := s.engine.root.Sub(fsstore.SubdirWorkspace, "recovery_state")
	entries, err := os.ReadDir(rdir)
	if err != nil {
		return
	}
	qs := fsstore.NewQuestStore(s.engine.root)
	removed := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		qid := strings.TrimSuffix(e.Name(), ".json")
		q, err := qs.LoadQuest(qid)
		if err != nil || q == nil {
			// meta 不存在或读取失败 → 孤儿，删
			_ = os.Remove(filepath.Join(rdir, e.Name()))
			removed++
			continue
		}
		switch q.Status {
		case model.QuestStatusSuccess, model.QuestStatusFailed, model.QuestStatusCancelled:
			_ = s.engine.root.DeleteRecoveryState(qid)
			removed++
		}
	}
	if removed > 0 {
		s.appendSchedulerEvent("recovery_state_cleanup", "", map[string]any{"count": removed})
	}
}

func (s *scheduler) appendSchedulerEvent(kind, automationID string, payload map[string]any) {
	if payload == nil {
		payload = map[string]any{}
	}
	payload["kind"] = kind
	payload["automation_id"] = automationID
	payload["ts"] = time.Now().UnixMilli()
	if err := fsstore.AppendJSONL(s.engine.root.Sub(fsstore.SubdirWorkspace, "scheduler", "events.jsonl"), payload); err != nil {
		s.log.Warn("[Scheduler] 写调度事件失败", "kind", kind, "automation_id", automationID, "err", err)
	}
}

func (s *scheduler) blockExpiredUserReviews(now time.Time) {
	timeout := s.engine.cfg.UserConfirmTimeoutMs
	if timeout <= 0 {
		return
	}
	deadline := now.UnixMilli() - timeout

	// 使用 QuestService 进行状态变更，确保经过统一校验和事件发布
	quests, err := s.engine.questService.ListByStatus(model.QuestStatusUserReview)
	if err != nil {
		s.log.Warn("[Scheduler] 扫描 user_review 超时失败", "err", err)
		return
	}

	for _, q := range quests {
		start := q.StartedAtMs
		if start == 0 {
			start = q.CreatedAtMs
		}
		if start == 0 || start > deadline {
			continue
		}
		// 通过服务层执行阻塞操作，统一入口
		ok, err := s.engine.questService.BlockUserReviewTimeout(q.ID, timeout)
		if err != nil {
			s.log.Warn("[Scheduler] user_review 超时阻塞失败", "qid", q.ID, "err", err)
			continue
		}
		if ok {
			s.log.Debug("[Scheduler] user_review 超时已阻塞", "qid", q.ID)
		}
	}
}

// blockExpiredRunningQuests 扫描 running/reviewing 状态的 quest，
// 扣除 waiting_input 暂停时长后若超过 max_duration_per_quest_ms 则 block。
//
// 兜底 rework 路径等 goroutine 失踪场景：launchRunningLoop 因
// reserveQuestRuntime 返回 nil（旧 goroutine 占着 e.running map）静默失败，
// quest 状态 running 但无 goroutine 推进，in-loop 的 max_duration 检查
// 无从触发。scheduler 层独立扫描确保这类 quest 不会无限卡死。
func (s *scheduler) blockExpiredRunningQuests(now time.Time) {
	limitMs := s.engine.cfg.MaxDurationPerQuestMs
	if limitMs <= 0 {
		return
	}
	nowMs := now.UnixMilli()
	qs := fsstore.NewQuestStore(s.engine.root)

	for _, status := range []model.QuestStatus{model.QuestStatusRunning, model.QuestStatusReviewing, model.QuestStatusWaitingInput} {
		quests, err := s.engine.questService.ListByStatus(status)
		if err != nil {
			s.log.Warn("[Scheduler] 扫描 running 超时失败", "status", status, "err", err)
			continue
		}
		for _, q := range quests {
			qMeta := fsstore.QuestMetaFromDomain(q)
			if qMeta == nil {
				continue
			}
			elapsed := questDurationElapsedMs(qs, qMeta, nowMs)
			// Wall-clock hard limit: 即使等待时间被扣除，wall-clock 超过
			// 2 倍预算也必须 block（防止 waiting_input 循环逃逸预算）。
			wallElapsed := nowMs - qMeta.StartedAtMs
			exceeded := elapsed >= limitMs || (qMeta.StartedAtMs > 0 && wallElapsed >= 2*limitMs)
			if !exceeded {
				continue
			}
			ok, err := s.engine.questService.BlockRunningTimeout(q.ID, elapsed, limitMs)
			if err != nil {
				s.log.Warn("[Scheduler] running 超时阻塞失败", "qid", q.ID, "err", err)
				continue
			}
			if ok {
				s.log.Warn("[Scheduler] running 超时已阻塞", "qid", q.ID, "elapsed_ms", elapsed, "limit_ms", limitMs)
				s.appendSchedulerEvent("running_timeout_blocked", q.ID, map[string]any{
					"elapsed_ms":      elapsed,
					"wall_elapsed_ms": wallElapsed,
					"limit_ms":        limitMs,
				})
			}
		}
	}
}

// getCron 获取已解析的 cron schedule（带缓存）
// hasActiveQuestForAutomation 检查某 automation 是否已有活跃（非终态）quest。
func (s *scheduler) hasActiveQuestForAutomation(automationID string) bool {
	activeStatuses := []model.QuestStatus{
		model.QuestStatusRunning,
		model.QuestStatusWaitingInput,
		model.QuestStatusReviewing,
		model.QuestStatusUserReview,
	}
	createdBy := model.QuestSourcePrefix + automationID
	for _, status := range activeStatuses {
		quests, err := s.engine.questService.ListByStatus(status)
		if err != nil {
			continue
		}
		for _, q := range quests {
			if q.CreatedBy == createdBy {
				return true
			}
		}
	}
	return false
}

func (s *scheduler) getCron(expr string) (*cronSchedule, error) {
	s.cacheMu.RLock()
	if c, ok := s.cronCache[expr]; ok {
		s.cacheMu.RUnlock()
		return c, nil
	}
	s.cacheMu.RUnlock()

	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	// double check
	if c, ok := s.cronCache[expr]; ok {
		return c, nil
	}

	c, err := ParseCron(expr)
	if err != nil {
		return nil, err
	}
	s.cronCache[expr] = c
	return c, nil
}
