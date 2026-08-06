package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"code.byted.org/lihuanyu.0w0/gloop/internal/connectors"
	"code.byted.org/lihuanyu.0w0/gloop/internal/domain/quest"
	"code.byted.org/lihuanyu.0w0/gloop/internal/events"
	"code.byted.org/lihuanyu.0w0/gloop/internal/executor"
	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
	"code.byted.org/lihuanyu.0w0/gloop/internal/notifications"
	"code.byted.org/lihuanyu.0w0/gloop/internal/prompt"
	"code.byted.org/lihuanyu.0w0/gloop/internal/skills"
	builtinprompts "code.byted.org/lihuanyu.0w0/gloop/prompts"
)

// ==================== v2 核心引擎 ====================
//
// 双职业循环：剑士（Warrior）执行 → 法师（Mage）评审 → 返工或用户终审
//
// 状态机：
//   pending → running → reviewing → user_review → success / failed
//                ↑           ↓             ↓
//                └── rework ─┘── rework ───┘
//
// 系统做机制（流程、隔离、预算、事件），Agent 做决策（内容、质量、是否完成）。

// Engine 是 v2 主引擎：fsstore 持久化、events 广播、executor 跑模型。
type Engine struct {
	root      *fsstore.Root
	cfg       *fsstore.GlobalConfig
	bus       events.EventBus
	execs     map[string]executor.Executor    // key: agent name
	skills    *skills.Registry                // 技能注册表（显式依赖，替代全局 skills.Default()）
	templates *prompt.TemplateStore           // Prompt 模板存储（内置 + 用户目录覆盖）
	notifier  notifications.Notifier          // 通知器（best-effort，可为 nil/noop）
	notifSub  *notifications.EventSubscriber  // 飞书通知事件订阅器
	connSub   *connectors.ConnectorSubscriber // HOTL v0.2 环 4: connector 订阅器（可为 nil）
	log       *slog.Logger
	workDir   string

	mu       sync.Mutex
	running  map[string]*questRuntime
	queue    []string // FIFO 队列，存 qid
	stopping bool

	scheduler        *scheduler               // 自动化调度器
	activitySnapshot *activitySnapshotUpdater // activity snapshot 被动更新器
	bgWG             sync.WaitGroup           // Engine-owned background tasks that must finish before shutdown

	// quest 应用服务：所有 quest 状态变更的统一入口
	questService *quest.Service
}

type questRuntime struct {
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}

	// commentCursor 已发送的用户评论数（持久化态，运行结束时写回 quest meta）
	commentCursor int
	// commentEnqueued 已入队的用户评论数（内存态，防止重复入队）
	commentEnqueued int
}

// ==================== 构造 ====================

// NewEngineOption 是 NewEngine 的可选参数（Functional Options 模式）。
// 留一个扩展口：测试里能塞假注册表，生产环境零配置会自动装载内置注册表。
type NewEngineOption func(*engineOptions)

type engineOptions struct {
	skills    *skills.Registry
	templates *prompt.TemplateStore
	notifier  notifications.Notifier
}

// WithSkillRegistry 显式传入技能注册表。
func WithSkillRegistry(r *skills.Registry) NewEngineOption {
	return func(o *engineOptions) { o.skills = r }
}

// WithTemplateStore 显式传入 prompt 模板存储。
// 不传则自动构造：内置模板 + 用户 prompts/ 目录 overlay。
func WithTemplateStore(s *prompt.TemplateStore) NewEngineOption {
	return func(o *engineOptions) { o.templates = s }
}

// WithNotifier 显式传入通知器（测试里注入 NoopNotifier 禁用真实通知）。
func WithNotifier(n notifications.Notifier) NewEngineOption {
	return func(o *engineOptions) { o.notifier = n }
}

func NewEngine(root *fsstore.Root, cfg *fsstore.GlobalConfig, bus events.EventBus, workDir string, log *slog.Logger, opts ...NewEngineOption) (*Engine, error) {
	if root == nil {
		return nil, errors.New("root is nil")
	}
	if cfg == nil {
		return nil, errors.New("cfg is nil")
	}
	if bus == nil {
		return nil, errors.New("bus is nil")
	}
	if log == nil {
		log = slog.Default()
	}
	if workDir == "" {
		workDir = "."
	}
	// Work on a local copy so defaults don't mutate caller's struct
	c := *cfg
	if c.MaxTurnsPerPhase <= 0 {
		c.MaxTurnsPerPhase = fsstore.DefaultGlobalConfig().MaxTurnsPerPhase
	}
	if c.MaxReworkPerQuest <= 0 {
		c.MaxReworkPerQuest = fsstore.DefaultGlobalConfig().MaxReworkPerQuest
	}
	if err := fsstore.ValidateAllowedCommands(c.CommandAllowlist); err != nil {
		return nil, fmt.Errorf("command allowlist invalid: %w", err)
	}

	// 解析可选依赖（DIP：未显式传入 → 装载内置默认）
	var o engineOptions
	for _, opt := range opts {
		opt(&o)
	}
	if o.skills == nil {
		o.skills = skills.LoadBuiltin()
	}
	if o.templates == nil {
		// 默认模板存储：内置 embed 为 base，用户目录 prompts/ 为 overlay
		overlayFS := os.DirFS(root.Sub(fsstore.SubdirPrompts))
		o.templates = prompt.NewTemplateStore(builtinprompts.FS(), overlayFS)
	}

	e := &Engine{
		root:    root,
		cfg:     &c,
		bus:     bus,
		execs:   map[string]executor.Executor{},
		skills:  o.skills,
		log:     log,
		workDir: workDir,
		running: map[string]*questRuntime{},
	}

	// 初始化 quest 应用服务（Phase 1: 领域层入口）
	repo := fsstore.NewQuestRepository(root)
	e.questService = quest.NewService(repo, &engineEventPublisher{e: e})

	e.scheduler = newScheduler(e)

	// 初始化通知器（best-effort，不可用也不影响启动）
	// 优先级：显式传入 > 环境变量禁用 > 自动检测 lark-cli
	if o.notifier != nil {
		e.notifier = o.notifier
	} else if notifications.DisabledByEnv() {
		e.notifier = notifications.NewNoopNotifier()
		e.log.Info("飞书通知已禁用（环境变量）")
	} else {
		e.notifier = notifications.NewLarkCliNotifier()
	}

	// 先用配置里的 host/port 拼默认 baseURL，server 启动后会用实际地址覆盖
	defaultBaseURL := fmt.Sprintf("http://%s:%d", advertiseNotifyHost(cfg.Host), cfg.Port)
	e.notifier.SetBaseURL(defaultBaseURL)

	if e.notifier.Available() {
		_, name := e.notifier.Recipient()
		e.log.Info("飞书通知已启用", "recipient", name)
	} else {
		e.log.Info("飞书通知未启用", "reason", e.notifier.StatusError())
	}

	// 启动飞书通知事件订阅器（自动监听 quest 状态变更并发系统通知）
	e.notifSub = notifications.NewEventSubscriber(bus, e.notifier, &questNotifyAdapter{root: root}, e.log)
	e.notifSub.Start()

	// HOTL v0.2 环 4：Connectors 订阅器。监听 EvtQuestApplied，按 automation 配置
	// 派发给具体 connector（如 GitConnector）。默认全 OFF，需在 GlobalConfig 里启用。
	e.connSub = connectors.NewConnectorSubscriber(bus, &questConnectorAdapter{root: root}, e.log)
	if gc := cfg.Connectors.Git; gc.Enabled {
		e.connSub.Register(connectors.NewGitConnector(connectors.GitConfig{
			Enabled:     true,
			Remote:      gc.Remote,
			BaseBranch:  gc.BaseBranch,
			AuthorName:  gc.AuthorName,
			AuthorEmail: gc.AuthorEmail,
			Push:        gc.Push,
		}, e.log))
		e.log.Info("connectors 已启用: git", "remote", gc.Remote, "push", gc.Push)
	}
	e.connSub.Start()

	return e, nil
}

// engineEventPublisher 适配 Engine.publish 到 quest.EventPublisher 接口
type engineEventPublisher struct {
	e *Engine
}

func (p *engineEventPublisher) Publish(qid, sessionID string, typ events.EventType, payload map[string]any) {
	p.e.publish(qid, sessionID, typ, payload)
}

func isTerminalEventType(typ events.EventType) bool {
	return typ == events.EvtQuestSuccess || typ == events.EvtQuestFailed || typ == events.EvtQuestCancelled
}

// publishFanoutLeafTerminalEvent publishes a root-quest semantic event when a fanout
// leaf transitions to terminal status. This ensures root ThreadView SSE subscribers
// learn about the change and can refetch to see the updated fanout_leaf_done and
// fanout_summary posts that SaveQuest already wrote on the root thread.
//
// Ordering guarantee: by the time this is called, SaveQuest has already returned,
// so root thread posts (fanout_leaf_done, fanout_summary) are durably written.
func (e *Engine) publishFanoutLeafTerminalEvent(leafQID string, typ events.EventType, evPayload json.RawMessage) {
	qs := fsstore.NewQuestStore(e.root)
	leaf, err := qs.LoadQuest(leafQID)
	if err != nil || leaf == nil || leaf.GroupID == "" {
		return
	}
	root, found := qs.FindFanoutRoot(leaf.GroupID)
	if !found {
		return
	}
	rootPayload := map[string]any{
		"kind":     "fanout_leaf_done",
		"group_id": leaf.GroupID,
		"leaf_qid": leafQID,
		"status":   string(leaf.Status),
		"event":    string(typ),
	}
	if len(evPayload) > 0 {
		var parsed map[string]any
		if json.Unmarshal(evPayload, &parsed) == nil {
			if comment, ok := parsed["comment"]; ok {
				rootPayload["comment"] = comment
			}
		}
	}
	e.publishRecorded(root.ID, "", events.EvtQuestNote, rootPayload)
}

// RegisterExecutor 注册一个执行器（agent name 作为 key）。
func (e *Engine) RegisterExecutor(agentName string, ex executor.Executor) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.execs[agentName] = ex
	e.log.Info("注册执行器", "agent", agentName, "type", ex.Type(), "name", ex.Name())
}

// HasExecutor reports whether a agent has a registered executor.
func (e *Engine) HasExecutor(agentName string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	_, ok := e.execs[agentName]
	return ok
}

// Executors returns a copy of the registered executor map keyed by agent name.
func (e *Engine) Executors() map[string]executor.Executor {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make(map[string]executor.Executor, len(e.execs))
	for name, ex := range e.execs {
		out[name] = ex
	}
	return out
}

// Shutdown 取消所有在跑委托并等待它们全部退出（最长 timeout）。
// 测试里用于避免 TempDir 清理时被后台写入打到。
func (e *Engine) Shutdown(timeout time.Duration) {
	// 先停通知订阅器，避免 shutdown 过程中还发通知
	if e.notifSub != nil {
		e.notifSub.Stop()
	}
	// 停 connector 订阅器
	if e.connSub != nil {
		e.connSub.Stop()
	}

	e.mu.Lock()
	e.stopping = true
	e.queue = nil
	runtimes := make([]*questRuntime, 0, len(e.running))
	for _, rt := range e.running {
		runtimes = append(runtimes, rt)
	}
	e.mu.Unlock()

	for _, rt := range runtimes {
		rt.cancel()
	}
	done := make(chan struct{})
	go func() {
		for _, rt := range runtimes {
			<-rt.done
		}
		close(done)
	}()
	if timeout <= 0 {
		<-done
		return
	}
	select {
	case <-done:
	case <-time.After(timeout):
		e.log.Warn("Shutdown 等待超时，仍有委托未退出", "timeout", timeout)
	}

	bgDone := make(chan struct{})
	go func() {
		e.bgWG.Wait()
		close(bgDone)
	}()
	if timeout <= 0 {
		<-bgDone
		return
	}
	select {
	case <-bgDone:
	case <-time.After(timeout):
		e.log.Warn("Shutdown 等待后台任务超时", "timeout", timeout)
	}
}

// ==================== 对外查询 API ====================

func (e *Engine) ListQuests() ([]*fsstore.QuestMeta, error) {
	return fsstore.NewQuestStore(e.root).ListQuests()
}

func (e *Engine) GetQuest(qid string) (*fsstore.QuestMeta, error) {
	return fsstore.NewQuestStore(e.root).LoadQuest(qid)
}

func (e *Engine) ListAdventurers() ([]*fsstore.AdventurerFile, error) {
	return e.root.ListAdventurers()
}

// PromptTemplates 返回引擎使用的 prompt 模板存储。
func (e *Engine) PromptTemplates() *prompt.TemplateStore {
	return e.templates
}

func (e *Engine) GetAdventurer(id string) (*fsstore.AdventurerFile, error) {
	return e.root.GetAdventurer(id)
}

func (e *Engine) ActivateAdventurer(id string, updates *fsstore.AdventurerUpdate) (*fsstore.AdventurerFile, error) {
	return e.root.ActivateAdventurer(id, updates)
}

// SaveAdventurer 新增或更新任意冒险者字段（不触发状态机）。
// ID 为空时按 name + class 生成一个稳定 ID；已存在则覆盖写入。
func (e *Engine) SaveAdventurer(a *fsstore.AdventurerFile) (*fsstore.AdventurerFile, error) {
	if a == nil {
		return nil, errors.New("adventurer is nil")
	}
	if a.ID == "" {
		suffix := fsstore.NewIDShort()
		if a.Name != "" {
			a.ID = fmt.Sprintf("%s_%s", a.Name, suffix)
		} else {
			a.ID = fmt.Sprintf("adv_%s", suffix)
		}
	}
	if a.CreatedAtMs == 0 {
		a.CreatedAtMs = fsstore.NowMs()
	}
	if a.Status == "" {
		if a.Agent == "" {
			a.Status = model.AdventurerPendingSetup
		} else {
			a.Status = model.AdventurerActive
		}
	}
	if err := e.root.SaveAdventurer(a); err != nil {
		return nil, err
	}
	return e.root.GetAdventurer(a.ID)
}

// RetireAdventurer 把冒险者置为 retired（软删除）。
func (e *Engine) RetireAdventurer(id string) error {
	a, err := e.root.GetAdventurer(id)
	if err != nil {
		return err
	}
	a.Status = model.AdventurerRetired
	return e.root.SaveAdventurer(a)
}

// SaveConfig 部分更新全局配置（budget / workspace / 命令白名单 等）。
// 传入的字段为 nil 表示保留原值；SaveConfig 会原子地替换 Engine.cfg 并落盘。
func (e *Engine) SaveConfig(patch *fsstore.GlobalConfigPatch) (*fsstore.GlobalConfig, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if patch == nil {
		return e.Config(), nil
	}
	merged := patch.Apply(e.cfg)
	if err := merged.Validate(); err != nil {
		return nil, fmt.Errorf("配置校验失败: %w", err)
	}
	if err := e.root.SaveConfig(merged); err != nil {
		return nil, err
	}
	e.cfg = merged
	return e.Config(), nil
}

func (e *Engine) ExportConfigPackage() (*fsstore.ConfigPackage, error) {
	return e.root.ExportConfigPackage()
}

func (e *Engine) PreviewConfigPackage(pkg *fsstore.ConfigPackage) (*fsstore.ConfigPackagePreview, error) {
	return e.root.PreviewConfigPackage(pkg)
}

func (e *Engine) ImportConfigPackage(pkg *fsstore.ConfigPackage) (*fsstore.GlobalConfig, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	cfg, err := e.root.ImportConfigPackage(pkg)
	if err != nil {
		return nil, err
	}
	e.cfg = cfg
	return e.Config(), nil
}

// ListAgentConfigs 列出所有 Agent 配置（含 enabled 状态）。
func (e *Engine) ListAgentConfigs() ([]*fsstore.AgentConfig, error) {
	return e.root.ListAgents()
}

// SetAgentEnabled 开关某个 Agent。
func (e *Engine) SetAgentEnabled(name string, enabled bool) (*fsstore.AgentConfig, error) {
	p, err := e.root.GetAgent(name)
	if err != nil {
		return nil, err
	}
	var ex executor.Executor
	if enabled {
		ex, err = e.buildExecutorForAgent(p)
		if err != nil {
			return nil, err
		}
	}
	p.Enabled = enabled
	if err := e.root.SaveAgent(p); err != nil {
		return nil, err
	}
	if enabled {
		e.RegisterExecutor(p.Name, ex)
	} else {
		e.UnregisterExecutor(p.Name)
	}
	return e.root.GetAgent(name)
}

// UpsertAgentConfig 新建或全量更新一个 Agent。
func (e *Engine) UpsertAgentConfig(p *fsstore.AgentConfig) (*fsstore.AgentConfig, error) {
	if p == nil || p.Name == "" {
		return nil, errors.New("agent name 不能为空")
	}
	var ex executor.Executor
	var err error
	if p.Enabled {
		ex, err = e.buildExecutorForAgent(p)
		if err != nil {
			return nil, err
		}
	}
	if err := e.root.SaveAgent(p); err != nil {
		return nil, err
	}
	if p.Enabled {
		e.RegisterExecutor(p.Name, ex)
	} else {
		e.UnregisterExecutor(p.Name)
	}
	return e.root.GetAgent(p.Name)
}

func (e *Engine) DeleteAgentConfig(name string) error {
	if name == "" {
		return errors.New("agent name 不能为空")
	}
	adventurers, err := e.root.ListAdventurers()
	if err != nil {
		return err
	}
	for _, adv := range adventurers {
		if adv.Agent == name && adv.Status == model.AdventurerActive {
			return fmt.Errorf("agent %s 正被冒险者 %s 使用，先迁移或退役该冒险者", name, adv.Name)
		}
	}
	if err := e.root.DeleteAgent(name); err != nil {
		return err
	}
	e.UnregisterExecutor(name)
	return nil
}

func (e *Engine) GetQuestEvents(qid string, n int) ([]fsstore.QuestEventRow, error) {
	return fsstore.NewQuestStore(e.root).ReadEvents(qid, n)
}

func (e *Engine) Config() *fsstore.GlobalConfig {
	c := *e.cfg
	return &c
}

func (e *Engine) WorkDir() string {
	return e.workDir
}

// ==================== 内部辅助 ====================

// pickWarrior 选择一个剑士冒险者（按等级倒序）。
func (e *Engine) pickWarrior() (*fsstore.AdventurerFile, error) {
	return e.root.FindByClass(model.ClassWarrior)
}

// pickMage 选择一个法师冒险者（按等级倒序）。
func (e *Engine) pickMage() (*fsstore.AdventurerFile, error) {
	return e.root.FindByClass(model.ClassMage)
}

func (e *Engine) pickRunnableAdventurer(id string, class model.AdventurerClass) (*fsstore.AdventurerFile, error) {
	if id != "" {
		adv, err := e.pickAdventurer(id, class)
		if err != nil {
			return nil, err
		}
		if err := e.ensureAdventurerAgentRunnable(adv); err != nil {
			return nil, err
		}
		return adv, nil
	}
	all, err := e.root.ListAdventurers()
	if err != nil {
		return nil, err
	}
	var pick *fsstore.AdventurerFile
	for _, adv := range all {
		if adv.Status != model.AdventurerActive || adv.Class != class {
			continue
		}
		if err := e.ensureAdventurerAgentRunnable(adv); err != nil {
			continue
		}
		if ex, _, err := e.getExecutor(adv); err == nil && ex.CapabilityTier() == executor.CapabilityTierC && hasNonTierCAdventurer(e, all, class) {
			continue
		}
		if pick == nil || fsstore.BetterAdventurer(adv, pick) {
			pick = adv
		}
	}
	if pick == nil {
		return nil, fmt.Errorf("没找到职业=%s 且绑定可用 agent 的冒险者", class)
	}
	return pick, nil
}

func hasNonTierCAdventurer(e *Engine, all []*fsstore.AdventurerFile, class model.AdventurerClass) bool {
	for _, adv := range all {
		if adv == nil || adv.Status != model.AdventurerActive || adv.Class != class {
			continue
		}
		if err := e.ensureAdventurerAgentRunnable(adv); err != nil {
			continue
		}
		if ex, _, err := e.getExecutor(adv); err == nil && ex.CapabilityTier() != executor.CapabilityTierC {
			return true
		}
	}
	return false
}

func (e *Engine) ensureAdventurerAgentRunnable(adv *fsstore.AdventurerFile) error {
	if adv == nil {
		return fmt.Errorf("冒险者为空")
	}
	if adv.Agent == "" {
		return fmt.Errorf("冒险者 %s 未绑定 agent", adv.ID)
	}
	agent, err := e.root.GetAgent(adv.Agent)
	if err != nil {
		return fmt.Errorf("冒险者 %s 绑定的 agent %s 不存在: %w", adv.ID, adv.Agent, err)
	}
	if !agent.Enabled {
		return fmt.Errorf("冒险者 %s 绑定的 agent %s 未启用", adv.ID, adv.Agent)
	}
	if agent.Type == model.AgentTypeMock || agent.Name == "mock" {
		return fmt.Errorf("冒险者 %s 绑定的是 Mock agent %s", adv.ID, adv.Agent)
	}
	if !e.HasExecutor(adv.Agent) {
		if _, err := e.RegisterAgentExecutor(agent); err != nil {
			return fmt.Errorf("冒险者 %s 绑定的 agent %s 无法注册执行器: %w", adv.ID, adv.Agent, err)
		}
	}
	return nil
}

func (e *Engine) pickAdventurer(id string, class model.AdventurerClass) (*fsstore.AdventurerFile, error) {
	if id == "" {
		return e.root.FindByClass(class)
	}
	adv, err := e.root.GetAdventurer(id)
	if err != nil {
		return nil, err
	}
	if adv.Class != class {
		return nil, fmt.Errorf("冒险者 %s 职业是 %s，不是 %s", id, adv.Class, class)
	}
	if adv.Status != model.AdventurerActive {
		return nil, fmt.Errorf("冒险者 %s 状态是 %s，未激活", id, adv.Status)
	}
	return adv, nil
}

// getExecutor 按冒险者配置找执行器。
func (e *Engine) getExecutor(adv *fsstore.AdventurerFile) (executor.Executor, string, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	modelFor := func(ex executor.Executor) string {
		if model := strings.TrimSpace(adv.Model); model != "" {
			return model
		}
		return ex.DefaultModel()
	}

	// 优先用冒险者绑定的 agent/agent
	if adv.Agent != "" {
		if ex, ok := e.execs[adv.Agent]; ok {
			return ex, modelFor(ex), nil
		}
		if adv.Class == model.ClassMage && adv.Agent == "relay" {
			if ex, ok := e.execs["traex"]; ok {
				return ex, modelFor(ex), nil
			}
		}
	}

	// 兜底：找第一个已注册的
	for _, ex := range e.execs {
		return ex, modelFor(ex), nil
	}

	return nil, "", fmt.Errorf("没有可用的执行器，请先配置 agent")
}

func (e *Engine) getExecutorForAgent(agentID string) (executor.Executor, string, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if strings.TrimSpace(agentID) == "" {
		return nil, "", fmt.Errorf("agent id 为空")
	}
	ex, ok := e.execs[agentID]
	if !ok {
		return nil, "", fmt.Errorf("agent %s 没有已注册执行器", agentID)
	}
	return ex, ex.DefaultModel(), nil
}

// publish 发布事件到总线 + 写事件日志。micro.* 事件只推送不持久化。
func (e *Engine) publish(qid, sid string, typ events.EventType, payload map[string]any) {
	e.publishRecorded(qid, sid, typ, payload)
}

func (e *Engine) publishRecorded(qid, sid string, typ events.EventType, payload map[string]any) events.Event {
	ev := e.recordEvent(qid, sid, typ, payload)
	e.publishRecordedEvent(ev)
	return ev
}

func (e *Engine) recordEvent(qid, sid string, typ events.EventType, payload map[string]any) events.Event {
	ev, _ := e.recordEventWithDurability(qid, sid, typ, payload, false)
	return ev
}

func (e *Engine) recordQuestEventRequired(qid, sid string, typ events.EventType, payload map[string]any) (events.Event, error) {
	return e.recordEventWithDurability(qid, sid, typ, payload, true)
}

func (e *Engine) recordEventWithDurability(qid, sid string, typ events.EventType, payload map[string]any, requireQuestEvent bool) (events.Event, error) {
	ev := events.Event{
		Type:      typ,
		QuestID:   qid,
		SessionID: sid,
		Payload:   events.MarshalPayload(payload),
		Timestamp: fsstore.NowMs(),
	}

	if qid == "" || typ == events.EvtTokenDelta || typ == events.EvtProgressUpdated {
		return ev, nil
	}

	qs := fsstore.NewQuestStore(e.root)
	row := &fsstore.QuestEventRow{
		Timestamp: ev.Timestamp,
		Type:      string(ev.Type),
		QuestID:   qid,
		SessionID: sid,
		Payload:   payload,
	}
	if err := qs.AppendEvent(qid, row); err != nil {
		e.log.Error("append event failed", "type", typ, "qid", qid, "err", err)
		if requireQuestEvent {
			return ev, err
		}
	} else {
		ev.ID = row.ID
		global := &fsstore.GlobalEventRow{
			EventID:   row.ID,
			Timestamp: row.Timestamp,
			Type:      row.Type,
			QuestID:   row.QuestID,
			SessionID: row.SessionID,
			Payload:   payload,
		}
		if err := e.root.AppendGlobalEvent(global); err != nil {
			e.log.Error("append global event failed", "type", typ, "qid", qid, "err", err)
		} else {
			ev.GlobalID = global.GlobalID
		}
	}
	return ev, nil
}

func (e *Engine) publishRecordedEvent(ev events.Event) {
	e.settleQuestEvent(ev)
	e.bus.Publish(ev)
	if isTerminalEventType(ev.Type) && ev.QuestID != "" {
		e.publishFanoutLeafTerminalEvent(ev.QuestID, ev.Type, ev.Payload)
	}
}

func (e *Engine) settleQuestEvent(ev events.Event) {
	if ev.QuestID == "" {
		return
	}
	e.projectNeedsHumanForEvent(ev.QuestID, ev.Type)
	e.settleRecoveryForEvent(ev.QuestID, ev.Type)
	e.settleAutomationTrustForEvent(ev.QuestID, ev.Type)
}

func (e *Engine) projectNeedsHumanForEvent(qid string, typ events.EventType) {
	switch typ {
	case events.EvtQuestBlocked, events.EvtQuestWaitingInput, events.EvtUserReview, events.EvtQuestApplyFailed:
	default:
		return
	}
	q, err := fsstore.NewQuestStore(e.root).LoadQuest(qid)
	if err != nil || q == nil {
		return
	}
	item := fsstore.HumanExceptionFromQuest(q)
	if item == nil {
		return
	}
	e.publish(qid, "", events.EvtNeedsHuman, map[string]any{
		"id":                 item.ID,
		"reason":             item.Reason,
		"recommended_action": item.RecommendedAction,
		"risk_level":         item.RiskLevel,
		"source_status":      item.SourceStatus,
		"available_actions":  item.AvailableActions,
	})
}

func (e *Engine) publishSystem(typ events.EventType, payload map[string]any) {
	ev := events.Event{
		Type:      typ,
		Payload:   events.MarshalPayload(payload),
		Timestamp: fsstore.NowMs(),
	}
	e.bus.Publish(ev)
}

// AppendNote 往 quest 追加一条笔记。
func (e *Engine) AppendNote(qid, sid, content, tag string) error {
	qs := fsstore.NewQuestStore(e.root)
	// 笔记写到 events.jsonl 里（type=quest.note）
	payload := map[string]any{
		"content": content,
		"tag":     tag,
	}
	if err := qs.AppendEvent(qid, &fsstore.QuestEventRow{
		Timestamp: fsstore.NowMs(),
		Type:      string(events.EvtQuestNote),
		QuestID:   qid,
		SessionID: sid,
		Payload:   payload,
	}); err != nil {
		e.log.Error("append note event failed", "qid", qid, "err", err)
	}
	// 也发布到总线
	e.publish(qid, sid, events.EvtQuestNote, payload)
	return nil
}

// PhaseCheckpoint 提交执行阶段的阶段结束信号（HTTP API / CLI 调用入口）。
// 写信号文件 + 发布事件，micro loop 下一轮检测到信号就结束当前阶段。
func (e *Engine) PhaseCheckpoint(qid, sid, summary, status string, deliverables []map[string]any, impactRaw string) error {
	if summary == "" {
		return fmt.Errorf("summary 必填")
	}
	if status == "" {
		status = "done"
	}
	if status != "done" && status != "blocked" {
		return fmt.Errorf("status 只能是 done 或 blocked")
	}

	q, err := e.GetQuest(qid)
	if err != nil {
		return fmt.Errorf("获取 quest 失败: %w", err)
	}

	data := map[string]any{
		"status":  status,
		"summary": summary,
		"source":  "http_api",
	}
	if deliverables != nil {
		data["deliverables"] = deliverables
	}
	if impact := parseImpactSummary(impactRaw); !impact.IsEmpty() {
		data["impact"] = impact
	} else if strings.TrimSpace(impactRaw) != "" {
		e.log.Warn("impact JSON 解析失败，已忽略", "qid", qid, "raw_len", len(impactRaw))
	}

	sig := fsstore.PhaseSignal{
		OK:           status == "done",
		Message:      "阶段结论已提交",
		PhaseEnded:   true,
		PhaseVerdict: status,
		PhaseComment: summary,
		Data:         data,
	}
	if err := fsstore.WritePhaseSignal(q.WorkspacePath, sid, sig); err != nil {
		return fmt.Errorf("写阶段信号失败: %w", err)
	}

	e.publish(qid, sid, events.EvtToolEnd, map[string]any{
		"tool":    "phase_checkpoint",
		"verdict": status,
		"summary": summary,
	})
	return nil
}

// ReviewQuest 提交评审阶段的阶段结束信号（HTTP API / CLI 调用入口）。
func (e *Engine) ReviewQuest(qid, sid, verdict, comment, hints string, score int) error {
	if verdict == "" {
		return fmt.Errorf("verdict 必填")
	}
	if !isValidReviewVerdict(PhaseSignal{PhaseEnded: true, PhaseVerdict: verdict}) {
		return fmt.Errorf("verdict 必须是 pass / request_changes / reject")
	}
	if comment == "" {
		return fmt.Errorf("comment 必填")
	}
	if verdict == "request_changes" && hints == "" {
		return fmt.Errorf("verdict=request_changes 时 hints 必填")
	}
	if score < 0 || score > 10 {
		score = 0
	}

	q, err := e.GetQuest(qid)
	if err != nil {
		return fmt.Errorf("获取 quest 失败: %w", err)
	}

	sig := fsstore.PhaseSignal{
		OK:           true,
		Message:      "评审结论已提交",
		PhaseEnded:   true,
		PhaseVerdict: verdict,
		PhaseComment: comment,
		PhaseHints:   hints,
		PhaseScore:   score,
		Data: map[string]any{
			"verdict": verdict,
			"comment": comment,
			"hints":   hints,
			"score":   score,
			"source":  "http_api",
		},
	}
	if err := fsstore.WritePhaseSignal(q.WorkspacePath, sid, sig); err != nil {
		return fmt.Errorf("写阶段信号失败: %w", err)
	}

	e.publish(qid, sid, events.EvtToolEnd, map[string]any{
		"tool":    "review_quest",
		"verdict": verdict,
		"comment": comment,
		"score":   score,
	})
	return nil
}

// QuestAsk 剑士向用户提问（等待用户输入）。
// projectWaitingInputRaised projects a waiting_input event into a durable ThreadPost.
// Only writes if evID > 0 (guardrail: no empty provenance).
// P0b: parent_reply_id always empty (reply to root). triggerPostID optional for causal_refs.
func (e *Engine) projectWaitingInputRaised(qid, questionText string, evID int64, triggerPostID string) {
	if evID == 0 {
		return
	}
	q, err := e.GetQuest(qid)
	if err != nil {
		e.log.Warn("projectWaitingInputRaised: get quest failed", "qid", qid, "err", err)
		return
	}
	qs := fsstore.NewQuestStore(e.root)
	if _, err := qs.EnsureRootThreadPost(q); err != nil {
		e.log.Error("projectWaitingInputRaised: ensure root failed, aborting post", "qid", qid, "err", err)
		return
	}
	causalRefs := []string{fsstore.RootPostID(qid)}
	if triggerPostID != "" {
		causalRefs = append(causalRefs, triggerPostID)
	}
	postID := fmt.Sprintf("sys_waiting_input_%d", evID)
	content := questionText
	if content == "" {
		content = "等待人工输入"
	}
	_, err = qs.AppendThreadPost(qid, fsstore.AppendThreadPostOptions{
		PostID:        postID,
		ThreadID:      qid,
		ParentReplyID: "",
		RootPostID:    fsstore.RootPostID(qid),
		CausalRefs:    causalRefs,
		AuthorRole:    model.PostRoleSystem,
		SourceEventID: evID,
		Kind:          "system_waiting_input",
		Content:       content,
	})
	if err != nil {
		e.log.Warn("projectWaitingInputRaised: append post failed", "qid", qid, "ev_id", evID, "err", err)
	}
}

func (e *Engine) QuestAsk(qid, sid, question string) error {
	if err := model.ValidateQuestionText(question); err != nil {
		// 拒绝劣质提问：把可读错误透回给 agent，逼它重写。
		return err
	}

	q, err := e.GetQuest(qid)
	if err != nil {
		return fmt.Errorf("获取 quest 失败: %w", err)
	}

	questionID := "question_" + fsstore.NewIDShort()
	phaseIdx := q.CurrentPhaseIdx()
	if phaseIdx < 0 {
		phaseIdx = 0
	}

	// 写信号
	sig := fsstore.PhaseSignal{
		OK:           false,
		Message:      "需要用户输入",
		PhaseEnded:   true,
		PhaseVerdict: string(model.QuestStatusWaitingInput),
		PhaseComment: question,
		Data: map[string]any{
			"status":      string(model.QuestStatusWaitingInput),
			"question_id": questionID,
			"question":    question,
			"source":      "http_api",
		},
	}
	if err := fsstore.WritePhaseSignal(q.WorkspacePath, sid, sig); err != nil {
		return fmt.Errorf("写阶段信号失败: %w", err)
	}

	// Record event durably, then write ThreadPost, then publish (avoid SSE refetch race).
	payload := map[string]any{
		"question_id": questionID,
		"question":    question,
		"status":      string(model.QuestStatusWaitingInput),
	}
	ev, err := e.recordQuestEventRequired(qid, sid, events.EvtQuestWaitingInput, payload)
	if err != nil {
		e.log.Error("record quest_ask event failed", "qid", qid, "err", err)
		// Non-fatal: phase signal was already written; macro loop will handle the state.
	} else {
		e.projectWaitingInputRaised(qid, question, ev.ID, "")
		e.publishRecordedEvent(ev)
	}
	return nil
}

// PublishPost 让 agent 显式发帖到 Feed。
// Feed 单源模型：agent 的自然语言输出不进 feed，只有显式 post 的内容进 feed。
// replyTo 为空=顶层发言（回复委托发起者）；非空=回复指定 post/quest。
func (e *Engine) PublishPost(qid, sid, content, replyTo, kind string) (string, error) {
	if strings.TrimSpace(qid) == "" {
		return "", fmt.Errorf("qid 不能为空")
	}
	if strings.TrimSpace(content) == "" {
		return "", fmt.Errorf("content 不能为空")
	}
	if kind == "" {
		kind = "post"
	}
	q, err := e.GetQuest(qid)
	if err != nil {
		return "", fmt.Errorf("获取 quest 失败: %w", err)
	}
	role, identity := postAuthorFromSession(q, sid)
	// Record event first, then derive stable post ID from event ID.
	eventPayload := map[string]any{
		"thread_id":    qid,
		"root_post_id": fsstore.RootPostID(qid),
		"author_role":  role,
		"author_id":    identity,
		"content":      content,
		"reply_to":     replyTo,
		"causal_refs":  causalRefsForReply(replyTo),
		"kind":         kind,
	}
	ev, err := e.recordQuestEventRequired(qid, sid, events.EvtAgentPost, eventPayload)
	if err != nil {
		return "", fmt.Errorf("写入 source event 失败: %w", err)
	}
	if ev.ID == 0 {
		return "", fmt.Errorf("agent post event has zero ID")
	}
	postID := fmt.Sprintf("post_%d", ev.ID)
	if _, err := fsstore.NewQuestStore(e.root).AppendThreadPost(qid, fsstore.AppendThreadPostOptions{
		PostID:         postID,
		ThreadID:       qid,
		ParentReplyID:  replyTo,
		RootPostID:     fsstore.RootPostID(qid),
		CausalRefs:     causalRefsForReply(replyTo),
		AuthorIdentity: identity,
		AuthorRole:     role,
		SourceEventID:  ev.ID,
		Kind:           kind,
		Content:        content,
	}); err != nil {
		return "", fmt.Errorf("写入 thread post 失败: %w", err)
	}
	e.publishRecordedEvent(ev)
	return postID, nil
}

func postAuthorFromSession(q *fsstore.QuestMeta, sid string) (model.PostAuthorRole, string) {
	if q != nil && strings.HasPrefix(sid, "mage") {
		return model.PostRoleChecker, q.MageID
	}
	if q != nil && strings.HasPrefix(sid, "warrior") {
		return model.PostRoleMaker, q.WarriorID
	}
	if q != nil && q.Source() == model.SourceAutomation {
		return model.PostRoleAutomation, q.CreatedBy
	}
	return model.PostRoleSystem, ""
}

func causalRefsForReply(replyTo string) []string {
	if strings.TrimSpace(replyTo) == "" {
		return nil
	}
	return []string{strings.TrimSpace(replyTo)}
}

// NotifyUser 给当前用户发送飞书通知。
// best-effort：通知不可用或发送失败时返回 ok=false + 原因描述，不会返回 error。
// 由平台工具 notify_user 调用，也可在引擎内部直接调用。
func (e *Engine) NotifyUser(ctx context.Context, title, body string) (ok bool, message string) {
	if e.notifier == nil {
		return false, "通知未配置"
	}
	return e.notifier.Send(ctx, title, body)
}

func (e *Engine) AppendUserComment(qid, comment, parentReplyID string) error {
	if comment == "" {
		return fmt.Errorf("comment 不能为空")
	}
	q, err := e.GetQuest(qid)
	if err != nil {
		return err
	}
	payload := map[string]any{
		"content": comment,
		"source":  "user",
	}

	// Step 1: Record event durably first. Fail if not persisted (guardrail: no empty provenance).
	ev, err := e.recordQuestEventRequired(qid, "", events.EvtQuestNote, payload)
	if err != nil {
		return fmt.Errorf("record comment event failed: %w", err)
	}
	if ev.ID == 0 {
		return fmt.Errorf("comment event has no ID, aborting thread post")
	}

	// Step 2: Ensure root thread post exists. Fail if broken chain.
	qs := fsstore.NewQuestStore(e.root)
	if _, err := qs.EnsureRootThreadPost(q); err != nil {
		return fmt.Errorf("ensure root thread post failed: %w", err)
	}

	// Step 2b: Validate parent_reply_id belongs to this thread (if provided).
	trimmedParent := strings.TrimSpace(parentReplyID)
	if trimmedParent != "" {
		existingPosts, loadErr := qs.LoadThreadPosts(qid)
		if loadErr != nil {
			return fmt.Errorf("load thread posts for parent validation failed: %w", loadErr)
		}
		found := false
		for _, p := range existingPosts {
			if p.PostID == trimmedParent {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("parent_reply_id %q not found in thread %q", trimmedParent, qid)
		}
	}

	// Step 3: Write human_comment ThreadPost before publishing event (avoid SSE refetch race).
	causalRefs := []string{fsstore.RootPostID(qid)}
	if trimmedParent != "" {
		causalRefs = append(causalRefs, trimmedParent)
	}
	postID := fmt.Sprintf("comment_%d", ev.ID)
	_, err = qs.AppendThreadPost(qid, fsstore.AppendThreadPostOptions{
		PostID:        postID,
		ThreadID:      qid,
		ParentReplyID: trimmedParent,
		RootPostID:    fsstore.RootPostID(qid),
		CausalRefs:    causalRefs,
		AuthorRole:    model.PostRoleHuman,
		SourceEventID: ev.ID,
		Kind:          "human_comment",
		Content:       comment,
	})
	if err != nil {
		return fmt.Errorf("append human_comment thread post failed: %w", err)
	}

	// Step 4: Now publish to event bus — subscribers that refetch will see the post.
	e.publishRecordedEvent(ev)

	// 状态感知：blocked 状态下用户评论自动解除阻塞继续运行
	if q.Status == model.QuestStatusBlocked {
		e.log.Info("用户评论触发阻塞恢复", "qid", qid)
		if err := e.ResolveBlockedQuest(context.Background(), qid, ResumeBlockedOptions{
			Action:  "continue",
			Comment: comment,
		}); err != nil {
			e.log.Warn("评论自动解除阻塞失败", "qid", qid, "err", err)
			// 不返回错误，评论已经成功保存
		}
	}

	return nil
}

func (e *Engine) AppendUserAnswer(qid, questionID, answer, source string) (*fsstore.AnswerRecord, error) {
	if strings.TrimSpace(answer) == "" {
		return nil, fmt.Errorf("answer 不能为空")
	}
	q, err := e.GetQuest(qid)
	if err != nil {
		return nil, err
	}
	rec := &fsstore.AnswerRecord{
		QuestionID: strings.TrimSpace(questionID),
		Content:    strings.TrimSpace(answer),
		Source:     source,
	}
	if rec.Source == "" {
		rec.Source = "api"
	}
	qs := fsstore.NewQuestStore(e.root)
	if err := qs.AppendAnswer(qid, rec); err != nil {
		return nil, err
	}
	answerPayload := map[string]any{
		"answer_id":   rec.AnswerID,
		"question_id": rec.QuestionID,
		"content":     rec.Content,
		"source":      rec.Source,
	}
	// Record event durably, then write ThreadPost, then publish (avoid SSE refetch race).
	ev, evErr := e.recordQuestEventRequired(qid, "", events.EvtQuestNote, answerPayload)
	if evErr != nil {
		return rec, fmt.Errorf("record answer event failed: %w", evErr)
	}
	if ev.ID == 0 {
		return rec, fmt.Errorf("record answer event returned zero ID")
	}
	// Ensure root exists before writing answer post.
	if _, rootErr := qs.EnsureRootThreadPost(q); rootErr != nil {
		return rec, fmt.Errorf("ensure root thread post failed for answer projection: %w", rootErr)
	}
	// Find latest system_waiting_input post for causal_refs.
	causalRefs := []string{fsstore.RootPostID(qid)}
	if posts, loadErr := qs.LoadThreadPosts(qid); loadErr == nil {
		for i := len(posts) - 1; i >= 0; i-- {
			if posts[i].Kind == "system_waiting_input" {
				causalRefs = append(causalRefs, posts[i].PostID)
				break
			}
		}
	}
	postID := fmt.Sprintf("human_answer_%d", ev.ID)
	if _, postErr := qs.AppendThreadPost(qid, fsstore.AppendThreadPostOptions{
		PostID:        postID,
		ThreadID:      qid,
		ParentReplyID: "",
		RootPostID:    fsstore.RootPostID(qid),
		CausalRefs:    causalRefs,
		AuthorRole:    model.PostRoleHuman,
		SourceEventID: ev.ID,
		Kind:          "human_answer",
		Content:       rec.Content,
	}); postErr != nil {
		return rec, fmt.Errorf("append human_answer thread post failed: %w", postErr)
	}
	e.publishRecordedEvent(ev)

	if q.Status == model.QuestStatusWaitingInput {
		q.Status = model.QuestStatusRunning
		if q.WaitingInput == nil {
			q.WaitingInput = &fsstore.WaitingInputState{}
		}
		if q.WaitingInput != nil {
			q.WaitingInput.LastAnswerID = rec.AnswerID
			q.WaitingInput.ProcessedAnswerIDs = append(q.WaitingInput.ProcessedAnswerIDs, rec.AnswerID)
			// 显式累加本次等待时长，供 questDurationElapsedMs 直接读取，
			// 避免依赖 events.jsonl 落盘时序（竞态会导致扣减失败）。
			if !q.WaitingInput.AskedAt.IsZero() {
				paused := time.Since(q.WaitingInput.AskedAt).Milliseconds()
				if paused > 0 {
					q.WaitingInput.AccumulatedPausedMs += paused
				}
			}
		}
		q.UpdatedAtMs = fsstore.NowMs()
		if err := qs.SaveQuest(q); err != nil {
			return rec, fmt.Errorf("保存 waiting_input 恢复状态失败: %w", err)
		}
		if q.WorkspacePath != "" {
			e.launchRunningLoop(context.Background(), qid)
		}
	}
	return rec, nil
}

// appendSessionRow 往 session 追加一行记录。
func (e *Engine) appendSessionRow(qid, sid string, row *fsstore.QuestSessionRow) {
	qs := fsstore.NewQuestStore(e.root)
	if err := qs.AppendSessionRow(qid, sid, row); err != nil {
		e.log.Error("append session row failed", "qid", qid, "sid", sid, "err", err)
	}
}

// ==================== 飞书通知辅助 ====================

// SetDashboardBaseURL 设置 dashboard 基础 URL，用于通知卡片里的跳转链接。
// server 启动后调用，用实际监听的地址覆盖默认值。
// 格式如 "http://127.0.0.1:37317"。
func (e *Engine) SetDashboardBaseURL(url string) {
	if e.notifier != nil {
		e.notifier.SetBaseURL(url)
	}
}

// questNotifyAdapter 把 fsstore.Root 适配成 notifications.QuestStore 接口。
type questNotifyAdapter struct {
	root *fsstore.Root
}

func (a *questNotifyAdapter) LoadQuest(qid string) (notifications.QuestInfo, error) {
	qs := fsstore.NewQuestStore(a.root)
	meta, err := qs.LoadQuest(qid)
	if err != nil {
		return notifications.QuestInfo{}, err
	}

	// 计算耗时
	// 优先用 StartedAtMs 作为起点（与 personal_stats 的 questDurationMs 一致），
	// 终态用 CompletedAtMs 作为终点，非终态用 UpdatedAtMs 近似当前耗时；
	// 都没有时回退到 CreatedAtMs -> UpdatedAtMs。
	var durationMs int64
	start := meta.StartedAtMs
	if start <= 0 {
		start = meta.CreatedAtMs
	}
	var end int64
	if meta.IsTerminal() && meta.CompletedAtMs > start {
		end = meta.CompletedAtMs
	} else if meta.UpdatedAtMs > start {
		end = meta.UpdatedAtMs
	}
	if end > start {
		durationMs = end - start
	}

	// 取评论/原因
	comment := meta.FinalComment
	if comment == "" {
		comment = meta.BlockedReason
	}

	return notifications.QuestInfo{
		ID:          meta.ID,
		ShortID:     meta.ShortID,
		Title:       meta.Query,
		Status:      string(meta.Status),
		ReworkCount: meta.ReworkCount,
		DurationMs:  durationMs,
		Comment:     comment,
		Impact:      meta.ImpactSummary,
	}, nil
}

func (a *questNotifyAdapter) LoadRecovery(qid string) (notifications.RecoveryInfo, error) {
	state, err := a.root.GetRecoveryState(qid)
	if err != nil {
		return notifications.RecoveryInfo{}, err
	}
	return notifications.RecoveryInfo{
		Status:        string(state.Status),
		PolicyName:    state.PolicyName,
		LastAction:    state.LastAction,
		BlockReason:   state.BlockReason,
		OriginalError: state.OriginalError,
		LastError:     state.LastError,
	}, nil
}

// questConnectorAdapter 把 fsstore.Root 适配成 connectors.QuestLookup 接口。
// 反查 quest meta + 继承自 automation 的 connectors 名单。
type questConnectorAdapter struct {
	root *fsstore.Root
}

func (a *questConnectorAdapter) LookupForConnector(qid string) (connectors.AppliedEvent, error) {
	qs := fsstore.NewQuestStore(a.root)
	meta, err := qs.LoadQuest(qid)
	if err != nil {
		return connectors.AppliedEvent{}, err
	}
	ev := connectors.AppliedEvent{
		QuestID:       meta.ID,
		ShortID:       meta.ShortID,
		Title:         meta.Query,
		WorkspacePath: meta.WorkspacePath,
		WorkingDir:    meta.BaseWorkingDir,
	}
	// quest 自身声明的 connectors（创建时选的）
	if len(meta.Connectors) > 0 {
		ev.Connectors = append(ev.Connectors, meta.Connectors...)
	}
	// quest 由 automation 创建 → 继承 automation 的 connectors 配置
	if id := meta.AutomationID(); id != "" {
		if auto, err := a.root.GetAutomation(id); err == nil && auto != nil {
			ev.Connectors = append(ev.Connectors, auto.Connectors...)
		}
	}
	// 去重
	seen := make(map[string]bool)
	dedup := ev.Connectors[:0]
	for _, c := range ev.Connectors {
		if !seen[c] {
			seen[c] = true
			dedup = append(dedup, c)
		}
	}
	ev.Connectors = dedup
	return ev, nil
}

// advertiseNotifyHost 决定通知里显示的 host。
// 和 auth 包 advertiseHost / server.advertiseHost 逻辑一致：
// 优先级：GLOOP_DASHBOARD_EXTERNAL_HOST > WEB_EXTERNAL_HOST > 绑定 host（非通配时）> 第一个非回环 IPv4 > 127.0.0.1
//
// 注：server 启动后会用 SetDashboardBaseURL 覆盖默认值，这里只是兜底。
func advertiseNotifyHost(bindHost string) string {
	if host := os.Getenv("GLOOP_DASHBOARD_EXTERNAL_HOST"); host != "" {
		return host
	}
	if host := os.Getenv("WEB_EXTERNAL_HOST"); host != "" {
		return host
	}
	if bindHost != "" && bindHost != "0.0.0.0" && bindHost != "::" {
		return bindHost
	}
	if ip := firstNonLoopbackIPv4Notify(); ip != "" {
		return ip
	}
	return "127.0.0.1"
}

// firstNonLoopbackIPv4Notify 返回第一个非回环 IPv4 地址（通知模块专用副本，避免循环依赖）。
func firstNonLoopbackIPv4Notify() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if strings.HasPrefix(iface.Name, "docker") || strings.HasPrefix(iface.Name, "veth") {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip4 := ip.To4()
			if ip4 != nil {
				return ip4.String()
			}
		}
	}
	return ""
}

// getCommentEnqueued 返回 quest 已入队的用户评论游标（用于防止重复入队）。
func (e *Engine) getCommentEnqueued(qid string) int {
	e.mu.Lock()
	defer e.mu.Unlock()
	rt, ok := e.running[qid]
	if !ok {
		// 不在运行中，从持久化读取
		qs := fsstore.NewQuestStore(e.root)
		q, err := qs.LoadQuest(qid)
		if err != nil {
			return 0
		}
		return q.CommentCursor
	}
	return rt.commentEnqueued
}

// setCommentEnqueued 设置 quest 已入队的用户评论游标。
func (e *Engine) setCommentEnqueued(qid string, cursor int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	rt, ok := e.running[qid]
	if !ok {
		return
	}
	rt.commentEnqueued = cursor
}

// syncCommentEnqueuedToCursor 将入队游标对齐到已发送游标。
// 每次 micro loop 启动时调用，确保前一轮入队但未发送的评论会被重新检测。
func (e *Engine) syncCommentEnqueuedToCursor(qid string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	rt, ok := e.running[qid]
	if !ok {
		return
	}
	rt.commentEnqueued = rt.commentCursor
}

// setCommentCursor 设置 quest 已发送的用户评论游标（写内存，运行结束时统一持久化）。
func (e *Engine) setCommentCursor(qid string, cursor int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	rt, ok := e.running[qid]
	if !ok {
		return
	}
	rt.commentCursor = cursor
}

func (e *Engine) persistCommentCursor(qid string, cursor int) {
	qs := fsstore.NewQuestStore(e.root)
	q, err := qs.LoadQuest(qid)
	if err != nil {
		return
	}
	if q.CommentCursor >= cursor {
		return
	}
	q.CommentCursor = cursor
	if err := qs.SaveQuest(q); err != nil {
		e.log.Warn("持久化 comment cursor 失败", "qid", qid, "err", err)
	}
}

// flushCommentCursor 将内存中的 comment cursor 写回 quest meta 持久化。
// 在宏循环结束（或 quest 退出运行态）时调用。
// 直接使用传入的 rt，不依赖 running map（releaseQuestRuntime 已先 delete）。
func (e *Engine) flushCommentCursor(qid string, rt *questRuntime) {
	if rt == nil {
		return
	}
	cursor := rt.commentCursor

	qs := fsstore.NewQuestStore(e.root)
	q, err := qs.LoadQuest(qid)
	if err != nil {
		return
	}
	if q.CommentCursor == cursor {
		return
	}
	e.persistCommentCursor(qid, cursor)
}

// ==================== Impact Summary 辅助 ====================

func parseImpactSummary(raw string) model.ImpactSummary {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return model.ImpactSummary{}
	}
	var is model.ImpactSummary
	if err := json.Unmarshal([]byte(raw), &is); err != nil {
		return model.ImpactSummary{}
	}
	return is
}

func extractImpactFromSignal(tr PhaseSignal) model.ImpactSummary {
	v, ok := tr.Data.(map[string]any)
	if !ok || v == nil {
		return model.ImpactSummary{}
	}
	raw, ok := v["impact"]
	if !ok || raw == nil {
		return model.ImpactSummary{}
	}
	if is, ok := raw.(model.ImpactSummary); ok {
		return is
	}
	if s, ok := raw.(string); ok {
		return parseImpactSummary(s)
	}
	if m, ok := raw.(map[string]any); ok {
		b, err := json.Marshal(m)
		if err != nil {
			return model.ImpactSummary{}
		}
		return parseImpactSummary(string(b))
	}
	return model.ImpactSummary{}
}
