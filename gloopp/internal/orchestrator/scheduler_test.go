package orchestrator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"code.byted.org/lihuanyu.0w0/gloop/internal/executor"
	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

type blockingExecutor struct {
	*executor.MockExecutor
	block <-chan struct{}
}

func (e *blockingExecutor) SendMessage(ctx context.Context, sessionID string, msg executor.Message, model string, stream chan<- executor.StreamChunk) (*executor.ChatResponse, error) {
	select {
	case <-e.block:
		return e.MockExecutor.SendMessage(ctx, sessionID, msg, model, stream)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// ==================== cron 解析测试 ====================

func TestParseCron_AllStars(t *testing.T) {
	c, err := ParseCron("* * * * *")
	if err != nil {
		t.Fatalf("ParseCron failed: %v", err)
	}
	// 每分钟都匹配
	now := time.Now()
	if !c.Matches(now) {
		t.Error("* * * * * should match any time")
	}
}

func TestParseCron_SpecificMinute(t *testing.T) {
	c, err := ParseCron("30 * * * *")
	if err != nil {
		t.Fatalf("ParseCron failed: %v", err)
	}

	// 第 30 分钟应该匹配
	t1 := time.Date(2024, 1, 1, 10, 30, 0, 0, time.UTC)
	if !c.Matches(t1) {
		t.Error("should match minute 30")
	}

	// 第 31 分钟不匹配
	t2 := time.Date(2024, 1, 1, 10, 31, 0, 0, time.UTC)
	if c.Matches(t2) {
		t.Error("should not match minute 31")
	}
}

func TestParseCron_Step(t *testing.T) {
	c, err := ParseCron("*/15 * * * *")
	if err != nil {
		t.Fatalf("ParseCron failed: %v", err)
	}

	// 手动验证几个
	t0 := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	if !c.Matches(t0) {
		t.Error("*/15 should match minute 0")
	}
	t15 := time.Date(2024, 1, 1, 10, 15, 0, 0, time.UTC)
	if !c.Matches(t15) {
		t.Error("*/15 should match minute 15")
	}
	t30 := time.Date(2024, 1, 1, 10, 30, 0, 0, time.UTC)
	if !c.Matches(t30) {
		t.Error("*/15 should match minute 30")
	}
	t7 := time.Date(2024, 1, 1, 10, 7, 0, 0, time.UTC)
	if c.Matches(t7) {
		t.Error("*/15 should not match minute 7")
	}
}

func TestParseCron_Range(t *testing.T) {
	c, err := ParseCron("10-20 * * * *")
	if err != nil {
		t.Fatalf("ParseCron failed: %v", err)
	}

	t10 := time.Date(2024, 1, 1, 10, 10, 0, 0, time.UTC)
	if !c.Matches(t10) {
		t.Error("10-20 should match minute 10")
	}
	t15 := time.Date(2024, 1, 1, 10, 15, 0, 0, time.UTC)
	if !c.Matches(t15) {
		t.Error("10-20 should match minute 15")
	}
	t20 := time.Date(2024, 1, 1, 10, 20, 0, 0, time.UTC)
	if !c.Matches(t20) {
		t.Error("10-20 should match minute 20")
	}
	t9 := time.Date(2024, 1, 1, 10, 9, 0, 0, time.UTC)
	if c.Matches(t9) {
		t.Error("10-20 should not match minute 9")
	}
	t21 := time.Date(2024, 1, 1, 10, 21, 0, 0, time.UTC)
	if c.Matches(t21) {
		t.Error("10-20 should not match minute 21")
	}
}

func TestParseCron_Enum(t *testing.T) {
	c, err := ParseCron("0,15,30,45 * * * *")
	if err != nil {
		t.Fatalf("ParseCron failed: %v", err)
	}

	t0 := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	if !c.Matches(t0) {
		t.Error("enum should match minute 0")
	}
	t15 := time.Date(2024, 1, 1, 10, 15, 0, 0, time.UTC)
	if !c.Matches(t15) {
		t.Error("enum should match minute 15")
	}
	t10 := time.Date(2024, 1, 1, 10, 10, 0, 0, time.UTC)
	if c.Matches(t10) {
		t.Error("enum should not match minute 10")
	}
}

func TestParseCron_HourAndMinute(t *testing.T) {
	c, err := ParseCron("0 9 * * *") // 每天早上 9 点
	if err != nil {
		t.Fatalf("ParseCron failed: %v", err)
	}

	t9am := time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC)
	if !c.Matches(t9am) {
		t.Error("should match 9:00")
	}
	t9_30 := time.Date(2024, 1, 1, 9, 30, 0, 0, time.UTC)
	if c.Matches(t9_30) {
		t.Error("should not match 9:30")
	}
	t10am := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	if c.Matches(t10am) {
		t.Error("should not match 10:00")
	}
}

func TestParseCron_Aliases(t *testing.T) {
	hourly, err := ParseCron("hourly")
	if err != nil {
		t.Fatalf("hourly alias failed: %v", err)
	}
	if !hourly.Matches(time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)) {
		t.Fatal("hourly should match the top of each hour")
	}
	if hourly.Matches(time.Date(2024, 1, 1, 10, 1, 0, 0, time.UTC)) {
		t.Fatal("hourly should not match non-zero minute")
	}
	daily, err := ParseCron("daily")
	if err != nil {
		t.Fatalf("daily alias failed: %v", err)
	}
	if !daily.Matches(time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC)) {
		t.Fatal("daily should match 09:00")
	}
	weekly, err := ParseCron("weekly")
	if err != nil {
		t.Fatalf("weekly alias failed: %v", err)
	}
	if !weekly.Matches(time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC)) {
		t.Fatal("weekly should match Monday 09:00")
	}
}

func TestParseCron_RangeWithStep(t *testing.T) {
	c, err := ParseCron("10-30/5 * * * *")
	if err != nil {
		t.Fatalf("ParseCron failed: %v", err)
	}

	for _, m := range []int{10, 15, 20, 25, 30} {
		tm := time.Date(2024, 1, 1, 10, m, 0, 0, time.UTC)
		if !c.Matches(tm) {
			t.Errorf("10-30/5 should match minute %d", m)
		}
	}
	for _, m := range []int{5, 8, 12, 35} {
		tm := time.Date(2024, 1, 1, 10, m, 0, 0, time.UTC)
		if c.Matches(tm) {
			t.Errorf("10-30/5 should not match minute %d", m)
		}
	}
}

func TestParseCron_Invalid(t *testing.T) {
	invalid := []string{
		"",
		"* * * *",     // 4 段
		"* * * * * *", // 6 段
		"60 * * * *",  // 分钟越界
		"* 24 * * *",  // 小时越界
		"* * 32 * *",  // 日越界
		"* * * 13 *",  // 月越界
		"* * * * 8",   // 周越界
		"abc * * * *", // 非数字
		"*/0 * * * *", // step 为 0
	}
	for _, expr := range invalid {
		_, err := ParseCron(expr)
		if err == nil {
			t.Errorf("expected error for invalid cron %q", expr)
		}
	}
}

// ==================== 调度器集成测试 ====================

func TestScheduler_TickTriggersAutomation(t *testing.T) {
	eng, _ := setupTestEngine(t)

	// 创建一个 schedule 类型的 automation，cron 每分钟都匹配
	autoCfg := &fsstore.AutomationConfig{
		ID:        "sched_test",
		Name:      "调度测试",
		Enabled:   true,
		Trigger:   fsstore.TriggerSchedule,
		Cron:      "*/1 * * * *", // 每分钟都匹配
		Query:     "调度器测试",
		QuestType: string(model.QuestTypeExecute),
		AutoStart: false,
	}
	if err := eng.SaveAutomation(autoCfg); err != nil {
		t.Fatalf("SaveAutomation failed: %v", err)
	}

	// 直接调 tick 方法（不启动后台 goroutine）
	eng.scheduler.tick(context.Background())

	// 验证：应该创建了一个 quest
	inbox, err := eng.ListInbox()
	if err != nil {
		t.Fatalf("ListInbox failed: %v", err)
	}
	found := false
	for _, q := range inbox {
		if q.CreatedBy == "automation:sched_test" {
			found = true
			break
		}
	}
	if !found {
		t.Error("scheduler tick should have created an inbox item")
	}
	eventsPath := eng.root.Sub(fsstore.SubdirWorkspace, "scheduler", "events.jsonl")
	raw, err := os.ReadFile(eventsPath)
	if err != nil {
		t.Fatalf("scheduler events should be written: %v", err)
	}
	if !strings.Contains(string(raw), "triggered") || !strings.Contains(string(raw), "sched_test") {
		t.Fatalf("scheduler events missing trigger record:\n%s", string(raw))
	}
	t.Logf("Inbox items: %d", len(inbox))
}

func TestScheduler_DisabledAutomationNotTriggered(t *testing.T) {
	eng, _ := setupTestEngine(t)

	autoCfg := &fsstore.AutomationConfig{
		ID:        "sched_disabled",
		Name:      "已禁用调度",
		Enabled:   false,
		Trigger:   fsstore.TriggerSchedule,
		Cron:      "* * * * *",
		Query:     "禁用测试",
		QuestType: string(model.QuestTypeExecute),
		AutoStart: false,
	}
	if err := eng.SaveAutomation(autoCfg); err != nil {
		t.Fatalf("SaveAutomation failed: %v", err)
	}

	eng.scheduler.tick(context.Background())

	inbox, _ := eng.ListInbox()
	for _, q := range inbox {
		if q.CreatedBy == "automation:sched_disabled" {
			t.Error("disabled automation should not be triggered")
		}
	}
}

func TestScheduler_ManualTriggerNotTriggered(t *testing.T) {
	eng, _ := setupTestEngine(t)

	autoCfg := &fsstore.AutomationConfig{
		ID:        "sched_manual",
		Name:      "手动触发",
		Enabled:   true,
		Trigger:   fsstore.TriggerManual,
		Cron:      "* * * * *",
		Query:     "手动测试",
		QuestType: string(model.QuestTypeExecute),
		AutoStart: false,
	}
	if err := eng.SaveAutomation(autoCfg); err != nil {
		t.Fatalf("SaveAutomation failed: %v", err)
	}

	eng.scheduler.tick(context.Background())

	inbox, _ := eng.ListInbox()
	for _, q := range inbox {
		if q.CreatedBy == "automation:sched_manual" {
			t.Error("manual trigger automation should not be triggered by scheduler")
		}
	}
}

func TestScheduler_StartStop(t *testing.T) {
	eng, _ := setupTestEngine(t)

	if eng.SchedulerRunning() {
		t.Error("scheduler should not be running initially")
	}

	eng.StartScheduler(context.Background())
	if !eng.SchedulerRunning() {
		t.Error("scheduler should be running after Start")
	}

	// 幂等：多次启动不报错
	eng.StartScheduler(context.Background())
	if !eng.SchedulerRunning() {
		t.Error("scheduler should still be running")
	}

	eng.StopScheduler()
	if eng.SchedulerRunning() {
		t.Error("scheduler should not be running after Stop")
	}

	// 幂等：多次停止不报错
	eng.StopScheduler()
	if eng.SchedulerRunning() {
		t.Error("scheduler should still not be running")
	}
}

func TestScheduler_UserReviewTimeoutBlocksQuest(t *testing.T) {
	eng, _ := setupTestEngine(t)
	eng.cfg.UserConfirmTimeoutMs = 60 * 60 * 1000

	qs := fsstore.NewQuestStore(eng.root)
	q := &fsstore.QuestMeta{
		ID:          "qst_timeout_user_review",
		ShortID:     "timeout",
		Query:       "timeout",
		Type:        model.QuestTypeExecute,
		Status:      model.QuestStatusUserReview,
		CreatedAtMs: time.Now().Add(-2 * time.Hour).UnixMilli(),
		StartedAtMs: time.Now().Add(-2 * time.Hour).UnixMilli(),
		MaxRework:   1,
	}
	if err := qs.CreateQuest(q); err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	q.CreatedAtMs = time.Now().Add(-2 * time.Hour).UnixMilli()
	q.StartedAtMs = q.CreatedAtMs
	if err := qs.SaveQuest(q); err != nil {
		t.Fatalf("SaveQuest failed: %v", err)
	}
	eng.scheduler.tick(context.Background())
	got, err := eng.GetQuest(q.ID)
	if err != nil {
		t.Fatalf("GetQuest failed: %v", err)
	}
	if got.Status != model.QuestStatusBlocked {
		t.Fatalf("status = %s, want blocked", got.Status)
	}
	if got.BlockedReason == "" {
		t.Fatal("blocked reason should be set")
	}
}

// TestScheduler_RunningTimeoutBlocksQuest 覆盖 rework 路径 goroutine 失踪：
// quest 状态 running 但无 goroutine 推进时，scheduler 的总时长兜底应将其
// block。回归：qst_2606199899 因 launchRunningLoop 静默失败卡死 127h。
func TestScheduler_RunningTimeoutBlocksQuest(t *testing.T) {
	eng, _ := setupTestEngine(t)
	eng.cfg.MaxDurationPerQuestMs = 60 * 60 * 1000 // 1h

	qs := fsstore.NewQuestStore(eng.root)
	q := &fsstore.QuestMeta{
		ID:          "qst_timeout_running",
		ShortID:     "runtimeout",
		Query:       "running timeout",
		Type:        model.QuestTypeExecute,
		Status:      model.QuestStatusRunning,
		CreatedAtMs: time.Now().Add(-2 * time.Hour).UnixMilli(),
		StartedAtMs: time.Now().Add(-2 * time.Hour).UnixMilli(),
		MaxRework:   1,
	}
	if err := qs.CreateQuest(q); err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	q.StartedAtMs = time.Now().Add(-2 * time.Hour).UnixMilli()
	if err := qs.SaveQuest(q); err != nil {
		t.Fatalf("SaveQuest failed: %v", err)
	}
	eng.scheduler.tick(context.Background())
	got, err := eng.GetQuest(q.ID)
	if err != nil {
		t.Fatalf("GetQuest failed: %v", err)
	}
	if got.Status != model.QuestStatusBlocked {
		t.Fatalf("status = %s, want blocked", got.Status)
	}
	if got.BlockedReasonCode != "duration_exceeded" {
		t.Fatalf("blocked_reason_code = %q, want duration_exceeded", got.BlockedReasonCode)
	}
}

func TestScheduler_CleanupExpiredWorkspaces(t *testing.T) {
	eng, _ := setupTestEngine(t)
	eng.cfg.WorkspaceRetentionDays = 7

	// 在 gloop 管理的隔离工作目录内创建 workspace，cleanup 才会安全删除。
	// 历史上 workspace_path 曾被记录为用户 cwd，过期清理误删用户代码，
	// 因此 isManagedWorkDir 只允许 <dataDir>/workspace/quests/ 之下的路径。
	qs := fsstore.NewQuestStore(eng.root)
	workDir := filepath.Join(eng.root.Sub(fsstore.SubdirQuests), "qst_scheduler_cleanup", "workspace")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workDir, "artifact.txt"), []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	q := &fsstore.QuestMeta{
		ID:            "qst_scheduler_cleanup",
		ShortID:       "cleanup",
		Query:         "cleanup",
		Type:          model.QuestTypeExecute,
		Status:        model.QuestStatusSuccess,
		WorkspacePath: workDir,
		CreatedAtMs:   time.Now().Add(-10 * 24 * time.Hour).UnixMilli(),
		CompletedAtMs: time.Now().Add(-8 * 24 * time.Hour).UnixMilli(),
	}
	if err := qs.CreateQuest(q); err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	q.WorkspacePath = workDir
	if err := qs.SaveQuest(q); err != nil {
		t.Fatalf("SaveQuest failed: %v", err)
	}

	eng.scheduler.tick(context.Background())
	if _, err := os.Stat(workDir); !os.IsNotExist(err) {
		t.Fatalf("workspace should be cleaned by scheduler, err=%v", err)
	}
	got, err := eng.GetQuest(q.ID)
	if err != nil {
		t.Fatalf("GetQuest failed: %v", err)
	}
	if !got.WorkspaceCleaned {
		t.Fatalf("workspace cleaned marker should be set: %+v", got)
	}
}

func TestScheduler_CleanupSkipsExternalWorkspaces(t *testing.T) {
	eng, _ := setupTestEngine(t)
	eng.cfg.WorkspaceRetentionDays = 7

	// workspace_path 指向 gloop 管理目录之外（如 /tmp 或用户仓库），
	// cleanup 安全护栏必须跳过，绝不自动删除。
	qs := fsstore.NewQuestStore(eng.root)
	externalDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(externalDir, "user_code.go"), []byte("package main"), 0o644); err != nil {
		t.Fatal(err)
	}
	q := &fsstore.QuestMeta{
		ID:            "qst_scheduler_external",
		ShortID:       "external",
		Query:         "external workspace",
		Type:          model.QuestTypeExecute,
		Status:        model.QuestStatusSuccess,
		WorkspacePath: externalDir,
		CreatedAtMs:   time.Now().Add(-10 * 24 * time.Hour).UnixMilli(),
		CompletedAtMs: time.Now().Add(-8 * 24 * time.Hour).UnixMilli(),
	}
	if err := qs.CreateQuest(q); err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	q.WorkspacePath = externalDir
	if err := qs.SaveQuest(q); err != nil {
		t.Fatalf("SaveQuest failed: %v", err)
	}

	eng.scheduler.tick(context.Background())
	// 外部目录必须仍然存在——cleanup 安全护栏保护了它
	if _, err := os.Stat(externalDir); err != nil {
		t.Fatalf("external workspace must NOT be cleaned by scheduler, err=%v", err)
	}
	got, err := eng.GetQuest(q.ID)
	if err != nil {
		t.Fatalf("GetQuest failed: %v", err)
	}
	if got.WorkspaceCleaned {
		t.Fatalf("external workspace must NOT be marked cleaned: %+v", got)
	}
}

// ==================== 并发控制 + 队列测试 ====================

func TestConcurrency_ConcurrentStartHonorsMaxConcurrent(t *testing.T) {
	eng, _ := setupTestEngine(t)
	defer eng.Shutdown(10 * time.Second)
	eng.cfg.MaxConcurrent = 1
	release := make(chan struct{})
	eng.RegisterExecutor("test_agent", &blockingExecutor{
		MockExecutor: executor.NewMockExecutor("blocking_mock"),
		block:        release,
	})
	defer close(release)

	ctx := context.Background()
	const n = 8
	quests := make([]*fsstore.QuestMeta, 0, n)
	for i := 0; i < n; i++ {
		q, err := eng.CreateQuest(ctx, fmt.Sprintf("并发启动任务 %d", i), model.QuestTypeExecute, "")
		if err != nil {
			t.Fatalf("CreateQuest %d failed: %v", i, err)
		}
		quests = append(quests, q)
	}

	var wg sync.WaitGroup
	errCh := make(chan error, n)
	for _, q := range quests {
		q := q
		wg.Add(1)
		go func() {
			defer wg.Done()
			errCh <- eng.StartQuest(ctx, q.ID)
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("StartQuest failed: %v", err)
		}
	}

	if got := eng.RunningCount(); got != 1 {
		t.Fatalf("running count = %d, want 1", got)
	}
	if total := eng.RunningCount() + eng.QueueSize(); total != n {
		t.Fatalf("running+queued = %d, want %d (running=%d queued=%d)", total, n, eng.RunningCount(), eng.QueueSize())
	}
}

func TestConcurrency_ConcurrentDuplicateStartIsIdempotent(t *testing.T) {
	eng, _ := setupTestEngine(t)
	defer eng.Shutdown(10 * time.Second)
	eng.cfg.MaxConcurrent = 1
	release := make(chan struct{})
	eng.RegisterExecutor("test_agent", &blockingExecutor{
		MockExecutor: executor.NewMockExecutor("blocking_mock"),
		block:        release,
	})
	defer close(release)

	ctx := context.Background()
	q, err := eng.CreateQuest(ctx, "同一任务并发启动", model.QuestTypeExecute, "")
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}

	const n = 8
	var wg sync.WaitGroup
	errCh := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errCh <- eng.StartQuest(ctx, q.ID)
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("StartQuest should be idempotent while runtime is reserved: %v", err)
		}
	}

	if got := eng.RunningCount(); got != 1 {
		t.Fatalf("running count = %d, want 1", got)
	}
	if got := eng.QueueSize(); got != 0 {
		t.Fatalf("queue size = %d, want 0", got)
	}
}

func TestConcurrency_ShutdownClearsQueueWithoutStartingNext(t *testing.T) {
	eng, _ := setupTestEngine(t)
	eng.cfg.MaxConcurrent = 1
	release := make(chan struct{})
	eng.RegisterExecutor("test_agent", &blockingExecutor{
		MockExecutor: executor.NewMockExecutor("blocking_mock"),
		block:        release,
	})
	defer close(release)

	ctx := context.Background()
	q1, err := eng.CreateQuest(ctx, "运行中任务", model.QuestTypeExecute, "")
	if err != nil {
		t.Fatalf("CreateQuest q1 failed: %v", err)
	}
	q2, err := eng.CreateQuest(ctx, "排队任务", model.QuestTypeExecute, "")
	if err != nil {
		t.Fatalf("CreateQuest q2 failed: %v", err)
	}
	if err := eng.StartQuest(ctx, q1.ID); err != nil {
		t.Fatalf("StartQuest q1 failed: %v", err)
	}
	if err := eng.StartQuest(ctx, q2.ID); err != nil {
		t.Fatalf("StartQuest q2 failed: %v", err)
	}
	if eng.RunningCount() != 1 || eng.QueueSize() != 1 {
		t.Fatalf("precondition running=%d queue=%d, want 1/1", eng.RunningCount(), eng.QueueSize())
	}

	eng.Shutdown(10 * time.Second)

	if eng.RunningCount() != 0 || eng.QueueSize() != 0 {
		t.Fatalf("after shutdown running=%d queue=%d, want 0/0", eng.RunningCount(), eng.QueueSize())
	}
	q2meta, err := eng.GetQuest(q2.ID)
	if err != nil {
		t.Fatal(err)
	}
	if q2meta.Status != model.QuestStatusPending {
		t.Fatalf("queued quest should remain pending and not auto-start during shutdown, got %s", q2meta.Status)
	}
}

func TestConcurrency_UserReviewReworkQueuesWhenConcurrencyFull(t *testing.T) {
	eng, _ := setupTestEngine(t)
	defer eng.Shutdown(10 * time.Second)
	eng.cfg.MaxConcurrent = 1
	release := make(chan struct{})
	eng.RegisterExecutor("test_agent", &blockingExecutor{
		MockExecutor: executor.NewMockExecutor("blocking_mock"),
		block:        release,
	})
	defer close(release)

	ctx := context.Background()
	runningQuest, err := eng.CreateQuest(ctx, "占用运行槽", model.QuestTypeExecute, "")
	if err != nil {
		t.Fatalf("CreateQuest running failed: %v", err)
	}
	if err := eng.StartQuest(ctx, runningQuest.ID); err != nil {
		t.Fatalf("StartQuest running failed: %v", err)
	}

	reworkQuest, err := eng.CreateQuest(ctx, "等待用户返工", model.QuestTypeExecute, "")
	if err != nil {
		t.Fatalf("CreateQuest rework failed: %v", err)
	}
	qs := fsstore.NewQuestStore(eng.root)
	reworkQuest.Status = model.QuestStatusUserReview
	reworkQuest.MaxRework = 2
	if err := qs.SaveQuest(reworkQuest); err != nil {
		t.Fatalf("SaveQuest rework failed: %v", err)
	}

	if err := eng.ResolveUserReview(ctx, reworkQuest.ID, model.VerdictRequestChange, "排队返工"); err != nil {
		t.Fatalf("ResolveUserReview request_changes failed: %v", err)
	}

	if eng.RunningCount() != 1 {
		t.Fatalf("running count = %d, want 1", eng.RunningCount())
	}
	if eng.QueueSize() != 1 {
		t.Fatalf("queue size = %d, want 1", eng.QueueSize())
	}
	got, err := eng.GetQuest(reworkQuest.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != model.QuestStatusRunning || got.ReviewHints != "排队返工" {
		t.Fatalf("rework quest should be marked running with hints while queued: %+v", got)
	}
}

func TestResolveUserReviewReworkSurvivesCancelledRequestContext(t *testing.T) {
	eng, _ := setupTestEngine(t)
	defer eng.Shutdown(10 * time.Second)

	ctx := context.Background()
	q, err := eng.CreateQuest(ctx, "请求上下文取消后的返工", model.QuestTypeExecute, "")
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}
	qs := fsstore.NewQuestStore(eng.root)
	q.Status = model.QuestStatusUserReview
	q.WarriorID = "adv_warrior_001"
	q.MageID = "adv_mage_001"
	q.MaxRework = 2
	if err := qs.SaveQuest(q); err != nil {
		t.Fatalf("SaveQuest failed: %v", err)
	}

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := eng.ResolveUserReview(cancelled, q.ID, model.VerdictRequestChange, "请求已结束也要返工"); err != nil {
		t.Fatalf("ResolveUserReview request_changes failed: %v", err)
	}

	waitForStatus(t, eng, q.ID, model.QuestStatusUserReview, 10*time.Second)
	got, err := eng.GetQuest(q.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ReworkCount != 1 {
		t.Fatalf("rework_count = %d, want 1", got.ReworkCount)
	}
}

func TestConcurrency_RemoveFromQueueOnStop(t *testing.T) {
	eng, _ := setupTestEngine(t)
	defer eng.Shutdown(10 * time.Second)
	eng.cfg.MaxConcurrent = 1

	ctx := context.Background()

	q1, _ := eng.CreateQuest(ctx, "占坑任务", model.QuestTypeExecute, "")
	q2, _ := eng.CreateQuest(ctx, "排队任务", model.QuestTypeExecute, "")

	// 启动第一个占坑
	eng.StartQuest(ctx, q1.ID)

	// 第二个入队
	eng.StartQuest(ctx, q2.ID)
	if eng.QueueSize() != 1 {
		t.Fatalf("expected queue size 1, got %d", eng.QueueSize())
	}

	// 停止第二个（应该从队列移除并标记取消）
	if err := eng.StopQuest(q2.ID, "不需要了"); err != nil {
		t.Fatalf("StopQuest (queued) failed: %v", err)
	}

	if eng.QueueSize() != 0 {
		t.Errorf("expected queue size 0 after stop, got %d", eng.QueueSize())
	}

	q2meta, _ := eng.GetQuest(q2.ID)
	if q2meta.Status != model.QuestStatusCancelled {
		t.Errorf("expected cancelled status, got %s", q2meta.Status)
	}
	if q2meta.FinalComment != "已取消（排队）：不需要了" {
		t.Errorf("unexpected final comment: %q", q2meta.FinalComment)
	}
}

func TestConcurrency_ZeroMaxUnlimited(t *testing.T) {
	eng, _ := setupTestEngine(t)
	eng.cfg.MaxConcurrent = 0 // 0 = 不限
	release := make(chan struct{})
	eng.RegisterExecutor("test_agent", &blockingExecutor{
		MockExecutor: executor.NewMockExecutor("blocking_mock"),
		block:        release,
	})

	ctx := context.Background()

	// 启动 5 个都应该立即运行
	for i := 0; i < 5; i++ {
		q, err := eng.CreateQuest(ctx, fmt.Sprintf("任务 %d", i), model.QuestTypeExecute, "")
		if err != nil {
			t.Fatalf("CreateQuest failed: %v", err)
		}
		if err := eng.StartQuest(ctx, q.ID); err != nil {
			t.Fatalf("StartQuest failed: %v", err)
		}
	}

	if eng.RunningCount() != 5 {
		t.Errorf("expected 5 running, got %d", eng.RunningCount())
	}
	if eng.QueueSize() != 0 {
		t.Errorf("expected 0 queued, got %d", eng.QueueSize())
	}
	close(release)
	eng.Shutdown(10 * time.Second)
}

func TestConcurrency_DuplicateStartIgnored(t *testing.T) {
	eng, _ := setupTestEngine(t)
	defer eng.Shutdown(10 * time.Second)
	eng.cfg.MaxConcurrent = 1

	ctx := context.Background()

	q, err := eng.CreateQuest(ctx, "重复启动测试", model.QuestTypeExecute, "")
	if err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}

	// 第一次启动
	if err := eng.StartQuest(ctx, q.ID); err != nil {
		t.Fatalf("first StartQuest failed: %v", err)
	}

	// 第二次启动同一个 quest（在运行中）——异步化后幂等返回 nil，不重复启动
	_ = eng.StartQuest(ctx, q.ID)

	// 验证只有一个运行实例
	if got := eng.RunningCount(); got != 1 {
		t.Fatalf("running count = %d, want 1 (duplicate start should be idempotent)", got)
	}
}
