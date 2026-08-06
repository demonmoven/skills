package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"code.byted.org/lihuanyu.0w0/gloop/internal/auth"
	"code.byted.org/lihuanyu.0w0/gloop/internal/events"
	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
	"code.byted.org/lihuanyu.0w0/gloop/internal/orchestrator"
	"code.byted.org/lihuanyu.0w0/gloop/internal/updatecheck"
	"code.byted.org/lihuanyu.0w0/gloop/internal/version"
)

const (
	MaxPortScan = 20
)

type Server struct {
	Host  string
	Port  int
	Token *auth.TokenStore
	Addr  string

	cfg     *fsstore.GlobalConfig
	engine  *orchestrator.Engine
	bus     events.EventBus
	root    *fsstore.Root
	stopSSE context.CancelFunc
	streams context.Context

	httpSrv *http.Server
	ln      net.Listener

	// updater 后台轮询 npm 最新版本，给 dashboard 渲染升级横幅。
	// 永远非 nil（即便 opt-out 也会构造一个不发起请求的 Checker），
	// 这样 handler 里不用判空。
	updater *updatecheck.Checker

	// stopUpdater 在 Shutdown 时 cancel 后台 updatecheck goroutine。
	stopUpdater context.CancelFunc
}

type serverRoute struct {
	Pattern        string
	Handler        http.HandlerFunc
	RequiresEngine bool
}

// New 构造 Server；host/port 优先从 cfg 取，缺省用 version.DefaultHost / version.DefaultPort。
// port<=0 时会从 DefaultPort 开始扫描空闲端口。
func New(
	cfg *fsstore.GlobalConfig,
	engine *orchestrator.Engine,
	bus events.EventBus,
	root *fsstore.Root,
	tok *auth.TokenStore,
) (*Server, error) {
	host := version.DefaultHost
	port := version.DefaultPort
	if cfg != nil {
		if cfg.Host != "" {
			host = cfg.Host
		}
		if cfg.Port > 0 {
			port = cfg.Port
		} else if cfg.Port == 0 {
			port = 0
		}
	}
	ln, actual, err := bindWithFallback(host, port)
	if err != nil {
		return nil, err
	}
	if tok != nil {
		tok.Port = actual
		tok.BindHost = host
		if root != nil {
			if err := tok.Save(root.Path()); err != nil {
				return nil, fmt.Errorf("保存 token 端口信息失败: %w", err)
			}
		}
	}
	streams, stopSSE := context.WithCancel(context.Background())
	s := &Server{
		Host:    host,
		Port:    actual,
		Token:   tok,
		Addr:    fmt.Sprintf("%s:%d", host, actual),
		cfg:     cfg,
		engine:  engine,
		bus:     bus,
		root:    root,
		stopSSE: stopSSE,
		streams: streams,
	}
	// 版本更新检查：数据目录优先走 root.Path()，fallback 到默认值
	// （Check 内部会再读一次 os.UserHomeDir 兜底）。
	dataDir := ""
	if root != nil {
		dataDir = root.Path()
	}
	s.updater = updatecheck.New(dataDir)
	s.httpSrv = &http.Server{Handler: s.buildRouter()}
	s.ln = ln

	// 把实际监听的地址同步给 engine，用于飞书通知里的跳转链接
	if s.engine != nil {
		baseURL := fmt.Sprintf("http://%s:%d", s.advertiseHost(), actual)
		s.engine.SetDashboardBaseURL(baseURL)
	}

	return s, nil
}

func bindWithFallback(host string, port int) (net.Listener, int, error) {
	// port<=0：未指定端口，自动扫描空闲端口（仅开发场景，生产不可达）
	if port <= 0 {
		start := version.DefaultPort
		end := start + MaxPortScan
		for p := start; p < end; p++ {
			ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, p))
			if err == nil {
				return ln, p, nil
			}
		}
		return nil, 0, fmt.Errorf("无空闲端口可绑定，扫描范围 %d-%d 均失败", start, end-1)
	}
	// port>0：指定端口（含默认端口），冲突即报错。
	// 避免 fallback 静默换端口导致多实例并立、pidfile 互相覆盖、用户不知实际端口。
	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return nil, 0, fmt.Errorf("端口 %d 已被占用：%w；可能已有 Gloop 实例在运行，用 `gloop status` 查看，或 `gloop stop` 后重启", port, err)
	}
	return ln, port, nil
}

// advertiseHost 返回对外展示的 host。
// 优先级：GLOOP_DASHBOARD_EXTERNAL_HOST > WEB_EXTERNAL_HOST > 绑定 host（非通配时）> 第一个非回环 IPv4 > 127.0.0.1
// 和 auth 包 advertiseHost 逻辑一致，确保通知里的链接能被用户点击访问。
func (s *Server) advertiseHost() string {
	if host := os.Getenv("GLOOP_DASHBOARD_EXTERNAL_HOST"); host != "" {
		return host
	}
	if host := os.Getenv("WEB_EXTERNAL_HOST"); host != "" {
		return host
	}
	if s.Host != "" && s.Host != "0.0.0.0" && s.Host != "::" {
		return s.Host
	}
	if ip := firstNonLoopbackIPv4(); ip != "" {
		return ip
	}
	return "127.0.0.1"
}

// firstNonLoopbackIPv4 返回第一个非回环的 IPv4 地址。
// 找不到时返回空字符串。
func firstNonLoopbackIPv4() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		// 跳过常见的虚拟网卡名（docker、veth 等），但不要太激进
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

func (s *Server) Start() error {
	if s.engine != nil {
		s.startBackgroundRecoveries()
		s.engine.StartScheduler(context.Background())
		s.engine.StartActivitySnapshotUpdater(context.Background())
	}
	// 后台版本检查：独立 ctx，在 Shutdown 里 cancel。
	// updater.Start 内部会 respect GLOOP_NO_UPDATE_NOTIFIER，所以无条件启动。
	if s.updater != nil {
		updCtx, cancel := context.WithCancel(context.Background())
		s.stopUpdater = cancel
		s.updater.Start(updCtx)
	}
	return s.httpSrv.Serve(s.ln)
}

func (s *Server) startBackgroundRecoveries() {
	if s.engine == nil {
		return
	}
	go func() {
		// These recoveries may start agent work. Keep them out of the HTTP
		// startup path so /healthz and the dashboard do not hang behind a
		// long-running auto-start quest.
		s.engine.RecoverPendingApplies(context.Background())
		s.engine.RecoverWaitingInputTimeouts(context.Background())
		s.engine.RecoverAutoStartInboxItems(context.Background())
		s.engine.RecoverOrphanedQuests(context.Background())
		s.engine.CleanupOrphanWorkspaces(context.Background())
	}()
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.stopUpdater != nil {
		s.stopUpdater()
	}
	if s.stopSSE != nil {
		s.stopSSE()
	}
	if s.engine != nil {
		s.engine.StopScheduler()
		// drain 在跑 quest：取消并等待退出，避免重启时 agent turn 被硬切。
		// 用 ctx 剩余超时，至少留 2s 给 httpSrv.Shutdown 收尾。
		drainDeadline, ok := ctx.Deadline()
		if ok {
			remaining := time.Until(drainDeadline)
			if remaining > 2*time.Second {
				s.engine.Shutdown(remaining - 2*time.Second)
			} else {
				s.engine.Shutdown(remaining)
			}
		} else {
			s.engine.Shutdown(0)
		}
	}
	if s.httpSrv == nil {
		return nil
	}
	return s.httpSrv.Shutdown(ctx)
}

func (s *Server) buildRouter() http.Handler {
	mux := http.NewServeMux()

	// 免鉴权端点：健康检查 + 版本信息
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"service":"` + version.AppName + `"}`))
	})
	mux.HandleFunc("GET /api/update-status", s.handleUpdateStatus)
	mux.HandleFunc("GET /assets/", s.serveDashboardAsset)
	mux.HandleFunc("GET /favicon.svg", s.serveDashboardOrRedirect)
	mux.HandleFunc("GET /favicon-knight.svg", s.serveDashboardOrRedirect)
	mux.HandleFunc("GET /favicon-mage.svg", s.serveDashboardOrRedirect)
	mux.HandleFunc("GET /icon-128.svg", s.serveDashboardOrRedirect)
	mux.HandleFunc("GET /icon-guild-badge.svg", s.serveDashboardOrRedirect)
	mux.HandleFunc("GET /icon-mono.svg", s.serveDashboardOrRedirect)
	mux.HandleFunc("GET /icon-mono-inverted.svg", s.serveDashboardOrRedirect)
	mux.HandleFunc("GET /icon-wordmark.svg", s.serveDashboardOrRedirect)
	mux.HandleFunc("GET /avatar-lark.png", s.serveDashboardOrRedirect)
	mux.HandleFunc("GET /avatar-lark.svg", s.serveDashboardOrRedirect)
	mux.HandleFunc("GET /avatar-lark-mage.png", s.serveDashboardOrRedirect)
	mux.HandleFunc("GET /avatar-lark-mage.svg", s.serveDashboardOrRedirect)

	// 其余都过 Token 中间件
	root := http.NewServeMux()
	for _, route := range s.routes() {
		root.HandleFunc(route.Pattern, route.Handler)
	}

	mux.Handle("/", auth.Middleware(s.Token, s.requireEngine(root)))
	return mux
}

func (s *Server) routes() []serverRoute {
	engine := true
	return []serverRoute{
		{Pattern: "GET /api/quests", Handler: s.listQuests, RequiresEngine: engine},
		{Pattern: "POST /api/quests", Handler: s.createQuest, RequiresEngine: engine},
		{Pattern: "POST /api/quests/spawn", Handler: s.spawnQuest, RequiresEngine: engine},
		{Pattern: "POST /api/quests/cleanup", Handler: s.cleanupQuestWorkspaces, RequiresEngine: engine},
		{Pattern: "GET /api/quests/{id}", Handler: s.getQuest, RequiresEngine: engine},
		{Pattern: "GET /api/quests/{id}/thread", Handler: s.getQuestThread, RequiresEngine: engine},
		{Pattern: "GET /api/quests/{id}/artifacts/{artifact_id}", Handler: s.getQuestArtifact, RequiresEngine: engine},
		{Pattern: "GET /api/quests/{id}/sessions/{sid}", Handler: s.getQuestSession, RequiresEngine: engine},
		{Pattern: "GET /api/quests/{id}/sessions/{sid}/context", Handler: s.getQuestSessionContext, RequiresEngine: engine},
		{Pattern: "GET /api/quests/{id}/ledger-audit", Handler: s.getQuestLedgerAudit, RequiresEngine: engine},
		{Pattern: "POST /api/quests/{id}/start", Handler: s.startQuest, RequiresEngine: engine},
		{Pattern: "POST /api/quests/{id}/comment", Handler: s.addQuestComment, RequiresEngine: engine},
		{Pattern: "POST /api/quests/{id}/answer", Handler: s.addQuestAnswer, RequiresEngine: engine},
		{Pattern: "POST /api/quests/{id}/stop", Handler: s.stopQuest, RequiresEngine: engine},
		{Pattern: "POST /api/quests/{id}/cancel", Handler: s.stopQuest, RequiresEngine: engine},
		{Pattern: "POST /api/quests/{id}/resolve-blocked", Handler: s.resolveBlockedQuest, RequiresEngine: engine},
		{Pattern: "POST /api/quests/{id}/recover-agent", Handler: s.recoverAgentQuest, RequiresEngine: engine},
		{Pattern: "POST /api/quests/{id}/workflow-mode", Handler: s.upgradeQuestWorkflowMode, RequiresEngine: engine},
		{Pattern: "POST /api/quests/{id}/resolve", Handler: s.resolveUserReview, RequiresEngine: engine},
		{Pattern: "POST /api/quests/{id}/resolve-user-review", Handler: s.resolveUserReview, RequiresEngine: engine},
		{Pattern: "GET /api/quests/{id}/diff", Handler: s.getQuestDiff, RequiresEngine: engine},
		{Pattern: "GET /api/quests/{id}/backups", Handler: s.listApplyBackups, RequiresEngine: engine},
		{Pattern: "GET /api/quests/{id}/backups/{backup_id}", Handler: s.getApplyBackup, RequiresEngine: engine},
		{Pattern: "POST /api/quests/{id}/apply", Handler: s.applyQuest, RequiresEngine: engine},
		{Pattern: "POST /api/quests/{id}/discard", Handler: s.discardQuest, RequiresEngine: engine},
		{Pattern: "GET /api/quests/{id}/stream", Handler: s.sseStream, RequiresEngine: engine},
		{Pattern: "POST /api/quests/{id}/spawn-execute", Handler: s.spawnExecuteQuest, RequiresEngine: engine},
		{Pattern: "GET /api/quests/{id}/trace", Handler: s.getQuestTrace, RequiresEngine: engine},
		{Pattern: "GET /api/human-exceptions", Handler: s.listHumanExceptions, RequiresEngine: engine},

		// phase / review / note / post（agent syscall HTTP 入口）
		{Pattern: "POST /api/quests/{id}/phases/{sid}/checkpoint", Handler: s.phaseCheckpoint, RequiresEngine: engine},
		{Pattern: "POST /api/quests/{id}/reviews/{sid}", Handler: s.reviewQuest, RequiresEngine: engine},
		{Pattern: "POST /api/quests/{id}/notes", Handler: s.addNote, RequiresEngine: engine},
		{Pattern: "POST /api/quests/{id}/posts", Handler: s.agentPost, RequiresEngine: engine},
		{Pattern: "POST /api/quests/{id}/ask/{sid}", Handler: s.questAsk, RequiresEngine: engine},
		{Pattern: "GET /api/adventurers", Handler: s.listAdventurers, RequiresEngine: engine},
		{Pattern: "GET /api/adventurers/{id}", Handler: s.getAdventurer, RequiresEngine: engine},
		{Pattern: "POST /api/adventurers", Handler: s.createAdventurer, RequiresEngine: engine},
		{Pattern: "POST /api/adventurers/{id}", Handler: s.updateAdventurer, RequiresEngine: engine},
		{Pattern: "PATCH /api/adventurers/{id}", Handler: s.updateAdventurer, RequiresEngine: engine},
		{Pattern: "POST /api/adventurers/{id}/activate", Handler: s.activateAdventurer, RequiresEngine: engine},
		{Pattern: "POST /api/adventurers/{id}/retire", Handler: s.retireAdventurer, RequiresEngine: engine},
		{Pattern: "GET /api/executors", Handler: s.listExecutors},
		{Pattern: "POST /api/executors", Handler: s.upsertExecutor, RequiresEngine: engine},
		{Pattern: "DELETE /api/executors/{id}", Handler: s.deleteExecutor, RequiresEngine: engine},
		{Pattern: "POST /api/executors/{id}/enable", Handler: s.enableExecutor, RequiresEngine: engine},
		{Pattern: "POST /api/executors/{id}/disable", Handler: s.disableExecutor, RequiresEngine: engine},
		{Pattern: "GET /api/settings", Handler: s.getSettings, RequiresEngine: engine},
		{Pattern: "POST /api/settings", Handler: s.saveSettings, RequiresEngine: engine},
		{Pattern: "PATCH /api/settings", Handler: s.saveSettings, RequiresEngine: engine},
		{Pattern: "GET /api/settings/mage-command-recommendations", Handler: s.recommendMageCommands, RequiresEngine: engine},
		{Pattern: "GET /api/settings/package/export", Handler: s.exportConfigPackage, RequiresEngine: engine},
		{Pattern: "POST /api/settings/package/preview", Handler: s.previewConfigPackage, RequiresEngine: engine},
		{Pattern: "POST /api/settings/package/import", Handler: s.importConfigPackage, RequiresEngine: engine},
		{Pattern: "GET /api/skills", Handler: s.listSkills},
		{Pattern: "GET /api/skills/{name}", Handler: s.getSkill},
		{Pattern: "GET /api/prompts", Handler: s.listPrompts},
		{Pattern: "GET /api/prompts/{name...}", Handler: s.getPrompt},
		{Pattern: "GET /api/stats", Handler: s.getStats},
		{Pattern: "GET /api/activity", Handler: s.getActivity, RequiresEngine: engine},

		{Pattern: "POST /api/notify", Handler: s.notifyUser, RequiresEngine: engine},

		{Pattern: "GET /api/automations", Handler: s.listAutomations, RequiresEngine: engine},
		{Pattern: "POST /api/automations", Handler: s.createAutomation, RequiresEngine: engine},
		{Pattern: "POST /api/automations/{id}", Handler: s.updateAutomation, RequiresEngine: engine},
		{Pattern: "PATCH /api/automations/{id}", Handler: s.updateAutomation, RequiresEngine: engine},
		{Pattern: "DELETE /api/automations/{id}", Handler: s.deleteAutomation, RequiresEngine: engine},
		{Pattern: "POST /api/automations/{id}/run", Handler: s.runAutomation, RequiresEngine: engine},
		{Pattern: "POST /api/automations/{id}/archive-no-finding", Handler: s.archiveAutomationNoFinding, RequiresEngine: engine},
		{Pattern: "GET /api/automations/discovery-archive", Handler: s.listAutomationDiscoveryArchive, RequiresEngine: engine},
		{Pattern: "POST /api/automations/{id}/enable", Handler: s.enableAutomation, RequiresEngine: engine},
		{Pattern: "POST /api/automations/{id}/disable", Handler: s.disableAutomation, RequiresEngine: engine},

		{Pattern: "GET /api/inbox", Handler: s.listInbox, RequiresEngine: engine},
		{Pattern: "POST /api/inbox/{id}", Handler: s.updateInboxItem, RequiresEngine: engine},
		{Pattern: "PATCH /api/inbox/{id}", Handler: s.updateInboxItem, RequiresEngine: engine},
		{Pattern: "POST /api/inbox/{id}/accept", Handler: s.acceptInboxItem, RequiresEngine: engine},
		{Pattern: "POST /api/inbox/{id}/reject", Handler: s.rejectInboxItem, RequiresEngine: engine},

		{Pattern: "GET /api/context", Handler: s.getContextMeta, RequiresEngine: engine},
		{Pattern: "GET /api/context/dims", Handler: s.listContextDims, RequiresEngine: engine},
		{Pattern: "GET /api/context/dims/{name}", Handler: s.getContextDim, RequiresEngine: engine},
		{Pattern: "PUT /api/context/dims/{name}", Handler: s.writeContextDim, RequiresEngine: engine},
		{Pattern: "GET /api/context/dims/{name}/export", Handler: s.exportContextDim, RequiresEngine: engine},
		{Pattern: "GET /api/context/summary", Handler: s.getContextSummary, RequiresEngine: engine},
		{Pattern: "PUT /api/context/summary", Handler: s.writeContextSummary, RequiresEngine: engine},
		{Pattern: "POST /api/context/refresh", Handler: s.refreshContext, RequiresEngine: engine},
		{Pattern: "GET /api/context/export/preview", Handler: s.previewKnowledgeExport, RequiresEngine: engine},
		{Pattern: "GET /api/context/export", Handler: s.exportAllContext, RequiresEngine: engine},

		{Pattern: "GET /api/stream", Handler: s.sseStream},
		{Pattern: "GET /", Handler: s.serveDashboardOrRedirect},
		{Pattern: "GET /dashboard", Handler: s.serveDashboardOrRedirect},
	}
}

// requireEngine 包装 mux：所有需要 engine 的路由在 engine 为 nil 时统一返回 500，
// 避免在每个 handler 里重复写 `if s.engine == nil { writeErr(...) }`。
func (s *Server) requireEngine(h http.Handler) http.Handler {
	engineRoutes := http.NewServeMux()
	for _, route := range s.routes() {
		if route.RequiresEngine {
			engineRoutes.HandleFunc(route.Pattern, func(http.ResponseWriter, *http.Request) {})
		}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, pattern := engineRoutes.Handler(r)
		if pattern != "" && s.engine == nil {
			writeErr(w, http.StatusInternalServerError, "engine 未初始化")
			return
		}
		h.ServeHTTP(w, r)
	})
}
