package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"code.byted.org/lihuanyu.0w0/gloop/internal/auth"
	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
	"code.byted.org/lihuanyu.0w0/gloop/internal/version"
)

// =========================================================================
// gloop start / stop / restart / status / dashboard / logs
// =========================================================================

// pidfile + logfile 放到数据目录下，避免污染全局路径。
func pidFilePath(dataDir string) string { return filepath.Join(dataDir, "server.pid") }
func logFilePath(dataDir string) string { return filepath.Join(dataDir, "server.log") }

// writePidFile 原子地写入 pidfile（先写 tmp 再 rename，避免读到半截 pid）。
func writePidFile(dataDir string, pid int) error {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return err
	}
	tmp := pidFilePath(dataDir) + ".tmp"
	if err := os.WriteFile(tmp, []byte(strconv.Itoa(pid)), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, pidFilePath(dataDir))
}

// stableShim 返回 ~/.gloop/bin/gloop（跨版本稳定入口）。
// 如果存在就优先用它注册开机自启，避免版本升级后自启指向的二进制被清理。
// 如果不存在就回退到 self，保证行为不变。
func stableShim(self string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return self
	}
	shim := filepath.Join(home, "."+version.RepoName, "bin", version.RepoName)
	if runtime.GOOS == "windows" {
		shim += ".cmd"
	}
	if _, err := os.Stat(shim); err == nil {
		return shim
	}
	return self
}

// ---- start ----

// startFlags 把 start / restart 共用的所有参数集中在一起，
// 并显式处理 --no-detach / --no-open / --no-autostart（Go flag 不自动支持 --no-*）。
type startFlags struct {
	host         string
	externalHost string
	port         int
	dataDir      string

	detach      bool
	openBrowser bool
	autostart   bool
}

func parseStartFlags(name string, args []string) (*startFlags, error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	f := &startFlags{}
	fs.StringVar(&f.host, "host", version.DefaultHost, "监听地址")
	fs.StringVar(&f.externalHost, "external-host", "", "Dashboard 对外展示地址")
	fs.IntVar(&f.port, "port", version.DefaultPort, "端口；0 = 自动探测")
	fs.StringVar(&f.dataDir, "data-dir", "", "数据目录（默认 ~/."+version.RepoName+"）")

	fs.BoolVar(&f.detach, "detach", true, "后台运行（默认开启；用 --no-detach 保持前台）")
	fs.BoolVar(&f.openBrowser, "open", true, "启动成功后自动打开 Dashboard（用 --no-open 关闭）")
	fs.BoolVar(&f.autostart, "autostart", true, "注册开机自启（用 --no-autostart 关闭；用户级，无需 sudo）")

	// 显式支持 --no-X 形式（每个默认值反向）
	noDetach := fs.Bool("no-detach", false, "保持前台运行（等价于 --detach=false）")
	noOpen := fs.Bool("no-open", false, "不自动打开浏览器（等价于 --open=false）")
	noAutostart := fs.Bool("no-autostart", false, "不注册开机自启（等价于 --autostart=false）")

	// 短写
	fs.BoolVar(&f.detach, "d", true, "后台运行（--detach 短写）")
	fs.BoolVar(&f.openBrowser, "o", true, "打开浏览器（--open 短写）")
	fs.BoolVar(noDetach, "nd", false, "保持前台（--no-detach 短写）")
	fs.BoolVar(noOpen, "no", false, "不自动打开浏览器（--no-open 短写）")
	fs.BoolVar(noAutostart, "na", false, "不注册开机自启（--no-autostart 短写）")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if *noDetach {
		f.detach = false
	}
	if *noOpen {
		f.openBrowser = false
	}
	if *noAutostart {
		f.autostart = false
	}
	return f, nil
}

// runStartCmd 实现 `gloop start`：等价于 init + server start，但默认
// （1）后台运行（--no-detach 可关闭），（2）启动成功后打开浏览器（--no-open 可关闭），
// （3）注册开机自启（--no-autostart 可关闭）。
func runStartCmd(ctx context.Context, log *slog.Logger, args []string) int {
	f, err := parseStartFlags("start", args)
	if err != nil {
		return 2
	}
	dd, err := resolveDataDir(f.dataDir)
	if err != nil {
		log.Error(logPrefix+" 解析数据目录失败", "err", err)
		return 1
	}
	if err := os.MkdirAll(dd, 0o755); err != nil {
		log.Error(logPrefix+" 创建数据目录失败", "dir", dd, "err", err)
		return 1
	}

	// ---- 防止重复启动 ----
	if pid, running := checkRunning(dd); running {
		// 已经在跑：打开浏览器（如果要）+ 打印 URL，然后退出
		fmt.Fprintf(os.Stderr, "ℹ️  Gloop 已在运行 (pid=%d)\n", pid)
		if f.openBrowser {
			if u, ok := readDashboardURL(dd); ok {
				fmt.Printf("   Dashboard: %s\n", u)
				_ = openURL(u)
			}
		}
		return 0
	}

	// ---- 注册开机自启（先做，失败不影响启动，只 warn） ----
	if f.autostart {
		self, err := os.Executable()
		if err != nil {
			log.Warn(logPrefix+" 定位自身可执行文件失败，跳过开机自启注册", "err", err)
		} else if err := installAutostart(stableShim(self), dd, f.host, f.port); err != nil {
			log.Warn(logPrefix+" 注册开机自启失败（非致命，继续启动）", "err", err)
		} else {
			log.Info(logPrefix + " 已注册开机自启")
		}
	} else {
		// 用户显式 --no-autostart：移除已存在的自启项（幂等）
		_ = uninstallAutostart()
	}

	// ---- 启动 ----
	if !f.detach {
		// 前台：沿用旧 runServer 逻辑，启动成功后再开浏览器
		return runStartForeground(ctx, log, dd, f.host, f.externalHost, f.port, f.openBrowser)
	}

	// 后台：re-exec 自己 + --no-detach + 重定向日志
	return runStartDetached(ctx, log, dd, f.host, f.externalHost, f.port, f.openBrowser)
}

func runStartForeground(ctx context.Context, log *slog.Logger, dataDir, host, externalHost string, port int, openBrowser bool) int {
	// 复用 runServer，启动成功（打印了 URL）后开浏览器
	// 做法：临时把 tok.DashboardURL() 记住 —— 但 runServer 内部生成 token，
	// 我们在其打印后从 token.json 读。
	urlCh := make(chan string, 1)
	go watchDashboardURL(dataDir, urlCh, 30*time.Second)

	exitCh := make(chan int, 1)
	go func() { exitCh <- runServer(ctx, log, dataDir, host, externalHost, port) }()

	select {
	case u := <-urlCh:
		if u != "" && openBrowser {
			_ = openURL(u)
		}
	case <-time.After(30 * time.Second):
		// 没拿到，不阻塞退出
	}
	return <-exitCh
}

// watchDashboardURL 轮询 token.json，读到 DashboardURL 就通过 ch 返回。
func watchDashboardURL(dataDir string, ch chan<- string, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	tokenPath := filepath.Join(dataDir, version.TokenFileName)
	for time.Now().Before(deadline) {
		time.Sleep(200 * time.Millisecond)
		if u, ok := readDashboardURL(dataDir); ok {
			ch <- u
			return
		}
		_ = tokenPath
	}
	ch <- ""
}

// readDashboardURL 从 token.json 解析出完整 Dashboard URL。
func readDashboardURL(dataDir string) (string, bool) {
	raw, err := os.ReadFile(filepath.Join(dataDir, version.TokenFileName))
	if err != nil {
		return "", false
	}
	var t auth.TokenStore
	if err := json.Unmarshal(raw, &t); err != nil {
		return "", false
	}
	if t.Port <= 0 || t.Token == "" {
		return "", false
	}
	return t.DashboardURL(), true
}

// runStartDetached 真正的后台启动：fork/exec 自己，前台进程等 URL 就绪后打印并退出。
func runStartDetached(ctx context.Context, log *slog.Logger, dataDir, host, externalHost string, port int, openBrowser bool) int {
	self, err := os.Executable()
	if err != nil {
		log.Error(logPrefix+" 定位自身可执行文件失败", "err", err)
		return 1
	}

	argv := []string{
		version.RepoName, "start",
		"--host", host,
		"--port", strconv.Itoa(port),
		"--data-dir", dataDir,
		"--no-detach",
		"--no-open",
		"--no-autostart", // 自启只由父进程注册一次
	}
	if externalHost != "" {
		argv = append(argv, "--external-host", externalHost)
	}

	logFile, err := os.OpenFile(logFilePath(dataDir), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		log.Error(logPrefix+" 打开日志文件失败", "err", err)
		return 1
	}
	defer logFile.Close()

	// 子进程环境继承父环境
	cmd := exec.Command(self, argv[1:]...)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Stdin = nil
	cmd.Env = os.Environ()
	if externalHost != "" {
		cmd.Env = append(cmd.Env, "GLOOP_DASHBOARD_EXTERNAL_HOST="+externalHost)
	}
	setDetachAttr(cmd) // 平台相关：setsid / CREATE_NEW_PROCESS_GROUP 等

	if err := cmd.Start(); err != nil {
		log.Error(logPrefix+" 启动后台子进程失败", "err", err)
		return 1
	}
	pid := cmd.Process.Pid

	// 前台进程等服务 ready：/healthz 可通 + token.json 可读
	fmt.Fprintf(os.Stderr, "🚀 正在启动 Gloop (pid=%d)…\n", pid)
	url, err := waitReady(ctx, dataDir, host, 45*time.Second)
	if err != nil {
		// 超时：子进程可能已经崩了，读一段日志给用户
		fmt.Fprintf(os.Stderr, "⚠️  启动超时: %v\n", err)
		if tail, e := tailLines(logFilePath(dataDir), 15); e == nil {
			fmt.Fprintln(os.Stderr, "--- server.log 末尾 ---")
			fmt.Fprint(os.Stderr, tail)
		}
		return 1
	}

	// 更新 pidfile（以防端口被扫到变了，但 pid 已经是对的）
	fmt.Printf("🟢 Gloop 已在后台启动\n")
	fmt.Printf("   Listen:      http://%s\n", listenHostPort(host, extractPortFromURL(url)))
	fmt.Printf("   Data Dir:    %s\n", dataDir)
	fmt.Printf("   Dashboard:   %s\n", url)
	fmt.Printf("   Log:         %s\n", logFilePath(dataDir))
	fmt.Printf("   Pid:         %d  (用 `gloop stop` 关闭)\n", pid)
	if openBrowser {
		if err := openURL(url); err != nil {
			fmt.Fprintf(os.Stderr, "（自动打开浏览器失败: %v，手动打开上面的 URL 即可）\n", err)
		}
	}
	fmt.Println()
	return 0
}

func listenHostPort(host string, port int) string {
	return fmt.Sprintf("%s:%d", host, port)
}
func extractPortFromURL(u string) int {
	// http://host:port/dashboard?t=xxx
	// 去掉前缀后找 : 与 下一个 /
	if i := strings.Index(u, "://"); i >= 0 {
		u = u[i+3:]
	}
	if i := strings.Index(u, "/"); i >= 0 {
		u = u[:i]
	}
	if i := strings.LastIndex(u, ":"); i >= 0 {
		if p, err := strconv.Atoi(u[i+1:]); err == nil {
			return p
		}
	}
	return version.DefaultPort
}

// waitReady 等健康检查可通，同时拿 Dashboard URL。
func waitReady(ctx context.Context, dataDir, host string, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	var lastURL string
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}
		if u, ok := readDashboardURL(dataDir); ok {
			lastURL = u
			// token 有了，再等 /healthz
			if probeHealthz(dataDir, host, extractPortFromURL(u)) {
				return u, nil
			}
		}
		time.Sleep(300 * time.Millisecond)
	}
	if lastURL != "" {
		return lastURL, nil // token 拿到了，不强求 healthz
	}
	return "", errors.New("等待服务就绪超时")
}

func probeHealthz(dataDir, host string, port int) bool {
	if port <= 0 {
		port = version.DefaultPort
	}
	// 实际 probe 用 127.0.0.1 更可靠（即使 host=0.0.0.0 也能通）
	probeHost := host
	if host == "" || host == "0.0.0.0" || host == "::" {
		probeHost = "127.0.0.1"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("http://%s:%d/healthz", probeHost, port), nil)
	if err != nil {
		return false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

// ---- stop / status / logs ----

func runStopCmd(_ context.Context, log *slog.Logger, args []string) int {
	fs := flag.NewFlagSet("stop", flag.ContinueOnError)
	dataDir := fs.String("data-dir", "", "数据目录")
	wait := fs.Bool("wait", true, "等进程真正退出")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	dd, err := resolveDataDir(*dataDir)
	if err != nil {
		log.Error(logPrefix+" 解析数据目录失败", "err", err)
		return 1
	}
	pid, running := checkRunning(dd)
	if !running {
		fmt.Fprintf(os.Stderr, "ℹ️  Gloop 未运行\n")
		_ = os.Remove(pidFilePath(dd))
		return 0
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  找不到 pid=%d 的进程（清理 pidfile）\n", pid)
		_ = os.Remove(pidFilePath(dd))
		return 0
	}
	if err := p.Signal(syscall.SIGTERM); err != nil {
		fmt.Fprintf(os.Stderr, "❌ 发送 SIGTERM 失败: %v；强制杀掉\n", err)
		_ = p.Kill()
	}
	if !*wait {
		fmt.Fprintf(os.Stderr, "📡 已发送停止信号给 pid=%d（--wait=false 不等待退出）\n", pid)
		return 0
	}
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if err := p.Signal(syscall.Signal(0)); err != nil {
			fmt.Fprintf(os.Stderr, "🛑 Gloop (pid=%d) 已停止\n", pid)
			_ = os.Remove(pidFilePath(dd))
			return 0
		}
		time.Sleep(200 * time.Millisecond)
	}
	fmt.Fprintf(os.Stderr, "⏱️  15s 未优雅退出，强制 kill\n")
	_ = p.Kill()
	_ = os.Remove(pidFilePath(dd))
	return 0
}

// ---- restart：stop(wait=true) + start ----
//
// restart 的典型场景：
//   - 改了 config.json（端口 / 预算 / 并发等 bootstrap 只读字段）
//   - 改了 agents/*.json（加/改 agent 执行器注册）
//   - `npm i -g @bytedance-dev/gloop` 升级了二进制
//
// 冒险者 / prompts 运行时读取，通常不需要重启。
func runRestartCmd(ctx context.Context, log *slog.Logger, args []string) int {
	// restart 接受和 start 完全相同的 flag。先用 parseStartFlags 解析一次，
	// 拿到 data-dir 给 stop 用，然后把原始 args 再交给 runStartCmd 执行。
	f, err := parseStartFlags("restart", args)
	if err != nil {
		return 2
	}
	dd, _ := resolveDataDir(f.dataDir)

	fmt.Fprintln(os.Stderr, "🔄 正在重启 Gloop…")
	if code := runStopCmd(ctx, log, []string{"--data-dir", dd}); code != 0 {
		return code
	}
	// stop 之后等一小会儿，避免端口还在 TIME_WAIT
	time.Sleep(500 * time.Millisecond)
	return runStartCmd(ctx, log, args)
}

func runStatusCmd(_ context.Context, _ *slog.Logger, args []string) int {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	dataDir := fs.String("data-dir", "", "数据目录")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	dd, err := resolveDataDir(*dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ 解析数据目录失败: %v\n", err)
		return 1
	}
	pid, running := checkRunning(dd)
	if running {
		fmt.Printf("🟢 运行中\n")
		fmt.Printf("   Pid:      %d\n", pid)
	} else {
		fmt.Printf("⚪ 未运行\n")
	}
	fmt.Printf("   Data Dir: %s\n", dd)
	if u, ok := readDashboardURL(dd); ok {
		fmt.Printf("   Dashboard: %s\n", u)
	} else {
		fmt.Printf("   Dashboard: （服务未启动，无可用地址）\n")
	}
	if as := autostartInstalled(); as != "" {
		fmt.Printf("   Autostart: %s\n", as)
	} else {
		fmt.Printf("   Autostart: 未注册\n")
	}
	return 0
}

func runDashboardCmd(_ context.Context, _ *slog.Logger, args []string) int {
	fs := flag.NewFlagSet("dashboard", flag.ContinueOnError)
	dataDir := fs.String("data-dir", "", "数据目录")
	printOnly := fs.Bool("print", false, "只打印 Dashboard URL，不打开浏览器")
	openBrowser := fs.Bool("open", true, "打开浏览器（默认开启；用 --no-open 关闭）")
	noOpen := fs.Bool("no-open", false, "不打开浏览器（等价于 --print）")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *noOpen {
		*openBrowser = false
		*printOnly = true
	}
	if *printOnly {
		*openBrowser = false
	}

	dd, err := resolveDataDir(*dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ 解析数据目录失败: %v\n", err)
		return 1
	}
	if _, running := checkRunning(dd); !running {
		fmt.Fprintf(os.Stderr, "⚪ Gloop 未运行。先执行 `gloop start` 启动 Dashboard。\n")
		return 1
	}
	u, ok := readDashboardURL(dd)
	if !ok {
		fmt.Fprintf(os.Stderr, "❌ 找不到可用 Dashboard URL。请执行 `gloop restart` 重新生成本地访问 token。\n")
		return 1
	}

	if *printOnly {
		fmt.Println(u)
		return 0
	}

	fmt.Printf("   Dashboard: %s\n", u)
	if *openBrowser {
		if err := openURL(u); err != nil {
			fmt.Fprintf(os.Stderr, "（自动打开浏览器失败: %v，手动打开上面的 URL 即可）\n", err)
		}
	}
	return 0
}

func runLogsCmd(_ context.Context, _ *slog.Logger, args []string) int {
	fs := flag.NewFlagSet("logs", flag.ContinueOnError)
	dataDir := fs.String("data-dir", "", "数据目录")
	follow := fs.Bool("f", false, "持续跟踪（tail -f）")
	lines := fs.Int("n", 50, "显示末尾 N 行")
	questID := fs.String("quest", "", "显示指定 quest 的事件日志")
	scheduler := fs.Bool("scheduler", false, "显示调度器事件日志")
	all := fs.Bool("all", false, "显示 server、scheduler 和最近 quest 事件")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *follow && (*questID != "" || *scheduler || *all) {
		fmt.Fprintln(os.Stderr, "❌ -f 目前只支持 server.log；quest/scheduler/all 请用非 follow 模式")
		return 2
	}
	dd, err := resolveDataDir(*dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ 解析数据目录失败: %v\n", err)
		return 1
	}
	if *questID != "" {
		return printScopedLog("quest "+*questID, questEventsLogPath(dd, *questID), *lines)
	}
	if *scheduler {
		return printScopedLog("scheduler", schedulerEventsLogPath(dd), *lines)
	}
	if *all {
		return runLogsAll(dd, *lines)
	}
	lp := logFilePath(dd)
	if _, e := os.Stat(lp); e != nil {
		fmt.Fprintf(os.Stderr, "ℹ️  日志文件不存在: %s\n", lp)
		return 0
	}
	if !*follow {
		out, e := tailLines(lp, *lines)
		if e != nil {
			fmt.Fprintf(os.Stderr, "❌ 读取日志失败: %v\n", e)
			return 1
		}
		fmt.Print(out)
		return 0
	}
	// follow 模式：先用 cat/type 等价的系统命令更可靠，但为了零依赖直接读
	f, e := os.Open(lp)
	if e != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", e)
		return 1
	}
	defer f.Close()
	// 跳到末尾倒数 N 行
	if _, err := seekLastLines(f, *lines); err != nil {
		fmt.Fprintf(os.Stderr, "❌ seek: %v\n", err)
		return 1
	}
	reader := bufio.NewReader(f)
	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			fmt.Print(line)
		}
		if err == io.EOF {
			time.Sleep(500 * time.Millisecond)
			continue
		}
		if err != nil {
			return 0
		}
	}
}

func printScopedLog(title, path string, lines int) int {
	if _, e := os.Stat(path); e != nil {
		fmt.Fprintf(os.Stderr, "ℹ️  日志文件不存在: %s\n", path)
		return 0
	}
	out, e := tailLines(path, lines)
	if e != nil {
		fmt.Fprintf(os.Stderr, "❌ 读取日志失败: %v\n", e)
		return 1
	}
	fmt.Printf("== %s ==\n", title)
	fmt.Print(out)
	return 0
}

func runLogsAll(dataDir string, lines int) int {
	code := printScopedLog("server", logFilePath(dataDir), lines)
	if code != 0 {
		return code
	}
	_ = printScopedLog("scheduler", schedulerEventsLogPath(dataDir), lines)
	for _, path := range recentQuestEventLogs(dataDir, 5) {
		qid := filepath.Base(filepath.Dir(path))
		_ = printScopedLog("quest "+qid, path, lines)
	}
	return 0
}

func schedulerEventsLogPath(dataDir string) string {
	return filepath.Join(dataDir, fsstore.SubdirWorkspace, "scheduler", "events.jsonl")
}

func questEventsLogPath(dataDir, qid string) string {
	qid = strings.TrimSpace(qid)
	if qid != "" && !strings.HasPrefix(qid, "qst_") {
		qid = "qst_" + qid
	}
	return filepath.Join(dataDir, fsstore.SubdirQuests, qid, "events.jsonl")
}

func recentQuestEventLogs(dataDir string, limit int) []string {
	dir := filepath.Join(dataDir, fsstore.SubdirQuests)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	type item struct {
		path string
		mod  time.Time
	}
	items := make([]item, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(dir, entry.Name(), "events.jsonl")
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		items = append(items, item{path: path, mod: info.ModTime()})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].mod.After(items[j].mod) })
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.path)
	}
	return out
}

// seekLastLines 把文件指针移到倒数 n 行之前的位置。
func seekLastLines(f *os.File, n int) (int64, error) {
	fi, err := f.Stat()
	if err != nil {
		return 0, err
	}
	size := fi.Size()
	if size == 0 {
		return 0, nil
	}
	// 从末尾 4KB 起逐步往前扫
	const chunk = 4096
	linesFound := 0
	offset := size
	buf := make([]byte, chunk)
	for offset > 0 {
		readSize := int64(chunk)
		if offset < readSize {
			readSize = offset
		}
		offset -= readSize
		if _, err := f.ReadAt(buf[:readSize], offset); err != nil && !errors.Is(err, io.EOF) {
			return 0, err
		}
		for i := readSize - 1; i >= 0; i-- {
			if buf[i] == '\n' {
				linesFound++
				if linesFound > n {
					return f.Seek(offset+i+1, io.SeekStart)
				}
			}
		}
	}
	return f.Seek(0, io.SeekStart)
}

func tailLines(path string, n int) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := seekLastLines(f, n); err != nil {
		return "", err
	}
	b, err := io.ReadAll(f)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// checkRunning 读取 pidfile 并检查进程是否真的在跑。返回 (pid, isRunning)。
func checkRunning(dataDir string) (int, bool) {
	raw, err := os.ReadFile(pidFilePath(dataDir))
	if err != nil {
		return 0, false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil {
		_ = os.Remove(pidFilePath(dataDir))
		return 0, false
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		_ = os.Remove(pidFilePath(dataDir))
		return pid, false
	}
	// Go 里 FindProcess 在 Unix 永远不返回 nil；用 Signal(0) 探测
	if err := p.Signal(syscall.Signal(0)); err != nil {
		_ = os.Remove(pidFilePath(dataDir))
		return pid, false
	}
	return pid, true
}

// =========================================================================
// 自动打开浏览器
// =========================================================================

func openURL(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default: // linux / bsd / ...
		if xdg, err := exec.LookPath("xdg-open"); err == nil {
			cmd = exec.Command(xdg, url)
		} else if sensible, err := exec.LookPath("sensible-browser"); err == nil {
			cmd = exec.Command(sensible, url)
		} else {
			return fmt.Errorf("未找到 xdg-open/sensible-browser，无法自动打开浏览器")
		}
	}
	cmd.Stdin = nil
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Start()
}

// =========================================================================
// 后台化（平台相关 syscall 属性）
// =========================================================================

// setDetachAttr 给 cmd 设上"脱离终端 / 新建进程组"属性。
// 注意：跨平台字段放在 *_platform.go 里避免在 Linux/macOS 上读到 Windows 专属字段。
func setDetachAttr(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	setPlatformDetachAttr(cmd.SysProcAttr)
}

// =========================================================================
// 跨平台开机自启
// =========================================================================
//
// 统一走 **用户级** 自启，不需要 sudo / 管理员权限：
//   - macOS:    ~/Library/LaunchAgents/com.gloop.start.plist
//   - Linux:    ~/.config/systemd/user/gloop.service  + systemctl --user enable gloop
//   - Windows:  %APPDATA%\Microsoft\Windows\Start Menu\Programs\Startup\gloop.lnk
//               或回退到 HKCU\Software\Microsoft\Windows\CurrentVersion\Run
//
// 自启项启动命令：<self> start --no-autostart
// （避免启动时再走一遍 installAutostart 做无意义的覆盖写）

const (
	autostartLabel = "com.gloop.start"
)

func autostartTargetPath() (string, string) {
	// 返回 (路径, 文件格式 plist/service/lnk/reg)
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "LaunchAgents", autostartLabel+".plist"), "plist"
	case "windows":
		// Start Menu Startup 比 registry 更直观；两者二选一
		if appdata := os.Getenv("APPDATA"); appdata != "" {
			return filepath.Join(appdata, "Microsoft", "Windows", "Start Menu", "Programs", "Startup", "gloop.vbs"), "vbs"
		}
		return "", "registry"
	default: // linux / bsd
		return filepath.Join(home, ".config", "systemd", "user", "gloop.service"), "systemd"
	}
}

// installAutostart 注册用户级开机自启。幂等：已存在且内容相同就跳过。
func installAutostart(self, dataDir, host string, port int) error {
	if self == "" {
		return errors.New("可执行文件路径为空")
	}
	path, kind := autostartTargetPath()
	args := []string{self, "start", "--no-detach", "--no-open", "--no-autostart",
		"--data-dir", dataDir, "--host", host, "--port", strconv.Itoa(port)}

	switch kind {
	case "plist":
		content := buildLaunchdPlist(autostartLabel, args, dataDir)
		return writeIfChanged(path, []byte(content))

	case "systemd":
		content := buildSystemdUserUnit(args, dataDir)
		if err := writeIfChanged(path, []byte(content)); err != nil {
			return err
		}
		// systemctl --user daemon-reload && enable
		if _, err := exec.LookPath("systemctl"); err == nil {
			_ = exec.Command("systemctl", "--user", "daemon-reload").Run()
			_ = exec.Command("systemctl", "--user", "enable", "gloop.service").Run()
		}
		return nil

	case "vbs":
		// 用 VBS 做无窗口启动（避免一直挂个 cmd 黑框）
		content := buildWindowsStartupVBS(self, dataDir, host, port)
		return writeIfChanged(path, []byte(content))

	case "registry":
		return installAutostartRegistry(args)

	default:
		return fmt.Errorf("未知平台自启类型: %s", kind)
	}
}

// uninstallAutostart 移除已注册的自启项（幂等）。
func uninstallAutostart() error {
	path, kind := autostartTargetPath()
	switch kind {
	case "plist", "systemd", "vbs":
		if path != "" {
			if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}
	case "registry":
		return uninstallAutostartRegistry()
	}
	if kind == "systemd" {
		if _, err := exec.LookPath("systemctl"); err == nil {
			_ = exec.Command("systemctl", "--user", "disable", "gloop.service").Run()
			_ = exec.Command("systemctl", "--user", "daemon-reload").Run()
		}
	}
	if kind == "plist" {
		_ = exec.Command("launchctl", "unload", path).Run()
	}
	return nil
}

// autostartInstalled 返回自启项当前的安装状态（给 status 展示）。
func autostartInstalled() string {
	path, kind := autostartTargetPath()
	switch kind {
	case "plist", "systemd", "vbs":
		if path != "" {
			if _, err := os.Stat(path); err == nil {
				return fmt.Sprintf("已注册 (%s)", path)
			}
		}
	case "registry":
		if registryAutostartExists() {
			return "已注册 (HKCU Run)"
		}
	}
	return ""
}

// ---- 具体内容模板 ----

func buildLaunchdPlist(label string, args []string, dataDir string) string {
	progArgs := ""
	for _, a := range args {
		progArgs += fmt.Sprintf("    <string>%s</string>\n", escapeXML(a))
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key><string>%s</string>
  <key>ProgramArguments</key>
  <array>
%s  </array>
  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key>
  <dict>
    <key>SuccessfulExit</key><false/>
  </dict>
  <key>WorkingDirectory</key><string>%s</string>
  <key>StandardOutPath</key><string>%s</string>
  <key>StandardErrorPath</key><string>%s</string>
</dict>
</plist>
`, label, progArgs, dataDir, logFilePath(dataDir), logFilePath(dataDir))
}

func buildSystemdUserUnit(args []string, dataDir string) string {
	execLine := strings.Join(quoteAll(args[1:]), " ")
	return fmt.Sprintf(`[Unit]
Description=Gloop Loop Engine Workbench
PartOf=graphical-session.target
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=%s
WorkingDirectory=%s
Restart=on-failure
RestartSec=5
StandardOutput=append:%s
StandardError=append:%s

[Install]
WantedBy=default.target
`, shellQuote(args[0])+" "+execLine, dataDir, logFilePath(dataDir), logFilePath(dataDir))
}

// buildWindowsStartupVBS 生成一个 VBScript，用 hidden 方式启动 gloop start。
// 这样用户登录后不会看到一闪而过的 cmd 窗口。
func buildWindowsStartupVBS(self, dataDir, host string, port int) string {
	// 双引号在 VBS 字符串里用 "" 转义
	q := func(s string) string { return "\"" + strings.ReplaceAll(s, "\"", "\"\"") + "\"" }
	args := strings.Join([]string{
		q("start"),
		q("--no-detach"),
		q("--no-open"),
		q("--no-autostart"),
		q("--data-dir"), q(dataDir),
		q("--host"), q(host),
		q("--port"), q(strconv.Itoa(port)),
	}, ", ")
	return fmt.Sprintf(`Set WshShell = CreateObject("WScript.Shell")
WshShell.CurrentDirectory = %s
WshShell.Run %s, 0, False
`, q(dataDir), q(self)+", "+args)
}

// ---- registry 回退（Windows 无 APPDATA 或 Startup 失败时用） ----

func installAutostartRegistry(args []string) error {
	if runtime.GOOS != "windows" {
		return nil
	}
	cmdLine := strings.Join(args, " ")
	return exec.Command("reg", "add",
		`HKCU\Software\Microsoft\Windows\CurrentVersion\Run`,
		"/v", "Gloop", "/t", "REG_SZ", "/d", cmdLine, "/f").Run()
}
func uninstallAutostartRegistry() error {
	if runtime.GOOS != "windows" {
		return nil
	}
	return exec.Command("reg", "delete",
		`HKCU\Software\Microsoft\Windows\CurrentVersion\Run`,
		"/v", "Gloop", "/f").Run()
}
func registryAutostartExists() bool {
	if runtime.GOOS != "windows" {
		return false
	}
	out, err := exec.Command("reg", "query",
		`HKCU\Software\Microsoft\Windows\CurrentVersion\Run`,
		"/v", "Gloop").Output()
	return err == nil && strings.Contains(string(out), "Gloop")
}

// ---- 小工具 ----

func writeIfChanged(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if old, err := os.ReadFile(path); err == nil && string(old) == string(content) {
		return nil // 内容相同，跳过
	}
	return os.WriteFile(path, content, 0o644)
}

func escapeXML(s string) string {
	r := strings.NewReplacer(
		"&", "&amp;", "<", "&lt;", ">", "&gt;",
		`"`, "&quot;", "'", "&apos;",
	)
	return r.Replace(s)
}

func shellQuote(s string) string {
	if s == "" {
		return `''`
	}
	if !strings.ContainsAny(s, " \t\n'\"`$();&|<>*?[\\") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
func quoteAll(ss []string) []string {
	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = shellQuote(s)
	}
	return out
}

// 确保 fsstore 被用到（某些命令里用到了 fsstore.Subdir* 等；这里保持 import 稳定）
var _ = fsstore.FileConfig
