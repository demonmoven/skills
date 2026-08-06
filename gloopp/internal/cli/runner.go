// Package cli 是 Gloop 的命令行核心：
//   - 所有子命令（init/start/stop/status/logs/server/quest/adventurer/config 等）的 flag 解析与 dispatch
//   - 所有入口（cmd/gloop / cmd/gloopd）共用同一份 Run() 逻辑
//     （cmd/gloopd 已合并为 gloop start --no-detach 的兼容别名，会打印警告）
//
// 内部按「先 bootstrap 再执行业务」的顺序：
//  1. open fsstore.Root（必要时 init）
//  2. LoadConfig + NewBus + NewEngine
//  3. 根据 agents/*.json 构造 executors 并注册到 engine
//  4. 执行业务（start / server / quest run / ...）
package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"code.byted.org/lihuanyu.0w0/gloop/internal/auth"
	"code.byted.org/lihuanyu.0w0/gloop/internal/events"
	"code.byted.org/lihuanyu.0w0/gloop/internal/executor"
	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
	"code.byted.org/lihuanyu.0w0/gloop/internal/orchestrator"
	"code.byted.org/lihuanyu.0w0/gloop/internal/version"
)

const logPrefix = "[Gloop]"

// Runner 封装一次命令执行所需的所有依赖。
type Runner struct {
	root     *fsstore.Root
	cfg      *fsstore.GlobalConfig
	engine   *orchestrator.Engine
	bus      events.EventBus
	log      *slog.Logger
	dataDir  string
	extraEnv []string // 注入到 engine / server 的额外环境变量
}

// -------------------- 构造 --------------------

func newRunner(dataDir string, log *slog.Logger) *Runner {
	if d, err := resolveDataDir(dataDir); err == nil {
		dataDir = d
	} else if dataDir == "" {
		dataDir, _ = fsstore.DefaultPath()
	}
	if log == nil {
		log = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelInfo,
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				return a
			},
		}))
	}
	return &Runner{dataDir: dataDir, log: log}
}

// -------------------- 公共小工具 --------------------

// expandTilde 把 ~/foo 展开为绝对路径；空串返回 "".
func expandTilde(p string) (string, error) {
	if p == "" {
		return "", nil
	}
	if strings.HasPrefix(p, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		p = filepath.Join(home, strings.TrimPrefix(p, "~"))
		// TrimPrefix 对 "~" 会变 ""，Join 会返回 home，OK
	}
	return filepath.Clean(p), nil
}

// detectBin 先 exec.LookPath；失败再 fallback；都失败返回 ""。
func detectBin(name, fallback string) string {
	if p, err := exec.LookPath(name); err == nil && p != "" {
		return p
	}
	if fallback != "" {
		if abs, err := expandTilde(fallback); err == nil {
			if st, err := os.Stat(abs); err == nil && !st.IsDir() {
				return abs
			}
		}
	}
	return ""
}

// defaultWorkingDir 选一个合适的默认工作目录（pwd 或者 ~/GloopProjects）。
func defaultWorkingDir() string {
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(home, "GloopProjects")
}

// InitDefaultFilesIfEmpty 仅当数据目录看起来未初始化时，调用 root.InitDefaultFiles。
// 幂等：已经初始化过就 no-op。
func (r *Runner) InitDefaultFilesIfEmpty() error {
	probe := detectAgents()
	wd := defaultWorkingDir()
	created, err := r.root.InitDefaultFiles(probe, wd)
	if err != nil {
		return err
	}
	if len(created) > 0 {
		r.log.Info(logPrefix+" 初始化默认文件", "count", len(created), "files", created)
	}
	return nil
}

func detectAgents() fsstore.AgentProbe {
	return fsstore.AgentProbe{
		TraeX:  detectBin("traex", "~/.local/bin/traex"),
		Relay:  detectBin("relay", ""),
		Pi:     detectBin("pi", ""),
		Codex:  detectBin("codex", ""),
		Aiden:  detectBin("aiden", "~/.npm-global/bin/aiden"),
		Hermes: detectBin("hermes", "~/.local/bin/hermes"),
	}
}

// bootstrap 是进入所有「需要 engine」的命令前的公共步骤：
//  1. open root
//  2. LoadConfig
//  3. new bus + engine
//  4. 枚举 agents/*.json → 构造 executors → 注册
//  5. 返回当前 token（给 server start / quest run 等用）
func (r *Runner) bootstrap(extraEnv ...string) (*auth.TokenStore, error) {
	r.extraEnv = extraEnv
	// 1) open root
	root, err := fsstore.Open(r.dataDir)
	if err != nil {
		return nil, fmt.Errorf("打开数据目录失败: %w", err)
	}
	r.root = root

	// 保证默认文件（冒险者 / agents / config）存在
	if err := r.InitDefaultFilesIfEmpty(); err != nil {
		r.log.Warn(logPrefix+" InitDefaultFiles 失败，继续", "err", err)
	}

	// 2) LoadConfig
	cfg, err := root.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("加载全局配置失败: %w", err)
	}
	r.cfg = cfg

	workingDir := cfg.DefaultWorkingDir
	if workingDir == "" {
		workingDir = defaultWorkingDir()
	}
	if wd, err := expandTilde(workingDir); err == nil {
		workingDir = wd
	}
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		r.log.Warn(logPrefix+" 创建工作目录失败", "dir", workingDir, "err", err)
	}

	// 3) new bus + engine
	r.bus = events.NewBus()
	eng, err := orchestrator.NewEngine(root, cfg, r.bus, workingDir, r.log)
	if err != nil {
		return nil, fmt.Errorf("创建引擎失败: %w", err)
	}
	r.engine = eng

	// 4) 构造 Executors
	if err := r.registerExecutors(workingDir); err != nil {
		r.log.Warn(logPrefix+" 部分 executor 注册失败", "err", err)
	}
	// 兜底：mock
	mockEx := executor.NewMockExecutor("exe_mock")
	r.engine.RegisterExecutor(string(model.AgentTypeMock), mockEx)

	// 5) 生成 / 加载 token（host/port 用 cfg，后续 server 会覆盖实际端口）
	host := cfg.Host
	port := cfg.Port
	if host == "" {
		host = version.DefaultHost
	}
	if port <= 0 {
		port = version.DefaultPort
	}
	tok, err := auth.Generate(r.root.Path(), host, port, false)
	if err != nil {
		return nil, fmt.Errorf("生成 token 失败: %w", err)
	}
	return tok, nil
}

// registerExecutors 根据 agents/*.json 注册执行器。
// 找不到二进制或不可用就 logger.Warn 且不启用（但保留 entry，后续用户可手改 json）。
func (r *Runner) registerExecutors(workingDir string) error {
	agents, err := r.root.ListAgents()
	if err != nil {
		r.log.Warn(logPrefix+" 读取 agents 失败，跳过真实执行器", "err", err)
		return err
	}
	registered := 0
	for _, p := range agents {
		if !p.Enabled {
			r.log.Debug(logPrefix+" agent 显式禁用，跳过",
				"name", p.Name, "type", p.Type)
			continue
		}

		if ex, err := r.engine.RegisterAgentExecutor(p); err != nil {
			r.log.Warn(logPrefix+" agent 执行器不可用，跳过",
				"name", p.Name, "type", p.Type, "command", p.Command, "err", err)
		} else {
			registered++
			r.log.Info(logPrefix+" 注册执行器",
				"name", p.Name, "type", p.Type, "command", p.Command, "executor", ex.ID())
		}
	}
	if registered == 0 {
		r.log.Warn(logPrefix + " 没有可用的真实执行器，全部走 mock 兜底")
	} else {
		r.log.Info(logPrefix+" 执行器注册完成", "real", registered)
	}
	return nil
}

// =========================================================================
// 命令入口
// =========================================================================

// Run 是整个 CLI 的总入口。argv = os.Args（长度至少 1）。
// 返回值：进程退出码。
func Run(ctx context.Context, argv []string, envEnviron []string) (exitCode int) {
	// 根 logger，所有子命令共享
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	_ = envEnviron // 预留未来通过环境变量传额外配置

	if len(argv) == 0 {
		argv = []string{version.RepoName}
	}
	binName := filepath.Base(argv[0])

	// 兼容：`gloopd` 可执行文件（旧用法）等价于 `gloop start --no-detach`
	if binName == version.Daemon || strings.TrimSuffix(binName, filepath.Ext(binName)) == version.Daemon {
		fmt.Fprintf(os.Stderr, "⚠️  警告: `%s` 已合并到 `gloop start`，请改用 `gloop start`（推荐后台）或 `gloop start --no-detach`\n", binName)
		argv = append([]string{version.RepoName, "start", "--no-detach", "--no-autostart"}, argv[1:]...)
		binName = version.RepoName
	}

	// 如果用户不带任何参数 → help
	if len(argv) == 1 {
		printTopLevelHelp(binName)
		return 0
	}

	// 顶层 subcommand
	sub := argv[1]
	rest := argv[2:]

	switch sub {
	case "help", "-h", "--help":
		printTopLevelHelp(binName)
		return 0

	case "version", "--version", "-V":
		fmt.Printf("%s %s · %s · %s\n", version.AppName, version.Version, version.Slogan, version.Mascot)
		return 0

	// ===== start / stop / restart / status / dashboard / logs：合并了原 gloopd 守护进程入口 =====
	case "start":
		return runStartCmd(ctx, log, rest)
	case "stop":
		return runStopCmd(ctx, log, rest)
	case "restart":
		return runRestartCmd(ctx, log, rest)
	case "status":
		return runStatusCmd(ctx, log, rest)
	case "dashboard":
		return runDashboardCmd(ctx, log, rest)
	case "logs":
		return runLogsCmd(ctx, log, rest)
	case "doctor":
		return runDoctorCmd(ctx, log, rest)

	case "update":
		return runUpdateCmd(ctx, log, rest)

	case "init":
		return runInit(ctx, log, rest)

	case "server":
		return runServerCmd(ctx, log, rest)

	case "quest":
		return runQuestCmd(ctx, log, rest)

	case "phase":
		return runPhaseCmd(ctx, log, rest)

	case "review":
		return runReviewCmd(ctx, log, rest)

	case "note":
		return runNoteCmd(ctx, log, rest)

	case "post":
		return runPostCmd(ctx, log, rest)

	case "notify":
		return runNotifyCmd(ctx, log, rest)

	case "command":
		return runCommandCmd(ctx, log, rest)

	case "run":
		return runQuestRun(ctx, log, rest)

	case "design":
		return runQuestRun(ctx, log, append([]string{"--type", "design"}, rest...))

	case "adventurer":
		return runAdventurerCmd(ctx, log, rest)

	case "config":
		return runConfigCmd(ctx, log, rest)

	case "automation":
		return runAutomationCmd(ctx, log, rest)

	case "context":
		return runContextCmd(ctx, log, rest)

	case "skill":
		return runSkillCmd(ctx, log, rest)

	case "prompt":
		return runPromptCmd(ctx, log, rest)

	case "stats":
		return runStatsCmd(log, rest)

	case "idl":
		return runIDLCmd(ctx, log, rest)

	case "code":
		return runCodeCmd(ctx, log, rest)

	case "inbox":
		return runInboxCmd(ctx, log, rest)

	default:
		fmt.Fprintf(os.Stderr, "未知子命令 %q\n\n", sub)
		printTopLevelHelp(binName)
		return 2
	}
}

// =========================================================================
// help
// =========================================================================

func printTopLevelHelp(binName string) {
	cliName := version.RepoName
	fmt.Printf(`%s %s — %s

用法：
  %s <command> [flags]

常用（人类用户）：
  start [--host] [--external-host] [--port] [--data-dir]
                                    后台启动 Server + 弹 Dashboard + 注册开机自启
                                    （分别用 --no-detach / --no-open / --no-autostart 关闭）
  stop [--wait]                     停止后台服务
  restart [flags]                   stop + start（改 config / 升版本 / 改 agent 后用）
  status                            查看服务状态 / Dashboard 地址 / 自启状态
  dashboard [--print]               打开当前 Dashboard；--print 只打印 URL
  logs [-f] [-n N] [--data-dir]     查看服务日志
  doctor agents [--json] [--smoke]  检查本机 agent 配置、二进制和适配器状态
  doctor e2e [--json]               跑本地 mock E2E 基准，验证状态机/workspace/review/apply
  update [--check] [--version X]    检查/升级 gloop 版本，升级后自动重启 daemon

命令：
  init                              初始化 ~/%s、agents 和基础配置
  server start [--host] [--external-host] [--port] [--data-dir]
                                    前台启动 HTTP Server + Dashboard（被 start 合并，推荐用 start）
  run --query <text> [flags]        创建 execute 委托（quest run 别名）
  design --query <text> [flags]     创建 design 委托（quest run --type design 别名）
  quest run  --query <text> [flags] 创建委托 → 挂起直到出结果
  quest info [qid]                   Agent syscall：查看当前/指定委托
  quest list                        列委托
  quest show  <qid>                 显示委托详情
  quest review <qid> <pass|changes|reject> [--apply] [--comment ...]
                                    人工终审
  quest comment <qid> --comment <text>
                                    追加用户评论
  quest answer <qid> [--question-id xxx] --answer <text>
                                    回答等待输入的问题
  quest diff <qid> [--full]         查看改动 diff
  quest apply <qid>                 应用改动到工作目录
  quest backup list <qid>           列 apply 备份
  quest backup show <qid> <backup>  查看 apply 备份元信息
  quest cleanup [--dry-run]         按保留期清理终态委托工作区
  quest ledger-audit <qid> [--repair]
                                    审计 / 修复 thread ledger 可补写缺口
  quest discard <qid> [--reason ...]
                                    丢弃改动
  quest cancel <qid> [--reason ...] 取消委托
  quest resolve-blocked <qid> [--action continue|user-review|cancel]
                                    处理 blocked 委托
  quest spawn-execute <qid> [--start]
                                    基于成功的 design 委托创建 execute 委托
  quest spawn --query <text> [--group-id ...] [--leaf-id ...] [--ownership-scopes ...]
                                    扇出独立委托（quest_spawn 平台工具）
  phase done --summary <text>        Agent syscall：提交执行阶段结论
  phase fail --reason <text>         Agent syscall：提交执行阶段阻塞/失败
  review pass --comment <text>       Agent syscall：提交评审通过
  review request-changes --comment <text> --hints <text>
                                    Agent syscall：要求返工
  review reject --comment <text>     Agent syscall：拒绝交付
  note add [--tag tag] <text>        Agent syscall：追加笔记
  notify --title <text> [--body ...] 发送飞书通知
  command list                       Agent syscall：列 Mage 可用验证命令
  command run <id>                   Agent syscall：执行 Mage 白名单验证命令
  code search [--mode ...] <query>   在工作区内搜索代码符号，结构化输出
  context list/show/summary/refresh/export/write-dim/write-summary
                                    查询、刷新、写入和导出用户上下文
  adventurer list                   列冒险者
  adventurer show <id>              查看冒险者详情
  skill list                        列可用技能
  skill show  <name>                查看技能详情
  skill validate [<name>]           校验技能格式与内容
  prompt list                       列 prompt 模板
  prompt show <name>                查看模板详情
  stats [--range today|week|all]     查看个人版本地运行统计
  idl export [--only all|skills|classes]
                                    导出 skill manifest IDL
  automation list                   列 automation
  automation templates              列官方 automation 模板
  automation run <id> [--watch]     手动触发 automation
  automation edit <id> [flags]      编辑 automation 配置
  automation enable <id>            启用 automation
  automation disable <id>           禁用 automation
  automation show <id>              查看 automation 详情
  inbox list                        列收件箱
  inbox edit <qid> [flags]          编辑待处理委托
  inbox accept <qid> [--watch]      接受收件箱委托
  inbox reject <qid> [--reason ...] 拒绝收件箱委托
  config show                       打印 config.json 路径与内容
  config edit                       用 $EDITOR / 默认编辑器打开 config.json
  update [--check] [--version X]     升级 gloop 到最新/指定版本，自动重启 daemon
  version                           显示版本
  help                              显示本帮助
`, version.AppName, version.Version, version.Slogan,
		cliName,
		version.DirName())
}

// =========================================================================
// init
// =========================================================================

func resolveDataDir(dataDir string) (string, error) {
	if dataDir == "" {
		dataDir = os.Getenv("GLOOP_DATA_DIR")
	}
	if dataDir == "" {
		return fsstore.DefaultPath()
	}
	return expandTilde(dataDir)
}

// MainEntry 是所有 cmd/*/main.go 的公共入口。
// 负责：信号上下文构造 → 调用 Run → os.Exit。
// 这样 gloop / gloopd 两个二进制入口只需要一行。
func MainEntry() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	os.Exit(Run(ctx, os.Args, os.Environ()))
}
