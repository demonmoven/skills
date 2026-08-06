package cli

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"code.byted.org/lihuanyu.0w0/gloop/internal/auth"
	"code.byted.org/lihuanyu.0w0/gloop/internal/server"
	"code.byted.org/lihuanyu.0w0/gloop/internal/version"
)

func runServerCmd(ctx context.Context, log *slog.Logger, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "用法: gloop server start [flags]")
		return 2
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "start":
		fs := flag.NewFlagSet("server start", flag.ContinueOnError)
		host := fs.String("host", version.DefaultHost, "监听地址")
		externalHost := fs.String("external-host", "", "Dashboard 对外展示地址（默认自动探测非 loopback IPv4）")
		port := fs.Int("port", version.DefaultPort, "端口；0 表示自动扫描可用端口")
		dataDir := fs.String("data-dir", "", "数据目录（默认 ~/.gloop）")
		if err := fs.Parse(rest); err != nil {
			return 2
		}
		return runServer(ctx, log, *dataDir, *host, *externalHost, *port)
	default:
		fmt.Fprintf(os.Stderr, "未知 server 子命令 %q\n", sub)
		return 2
	}
}

// runServer 是 gloop start --no-detach 和 gloop server start 的共享实现。
// 普通用户应优先使用 gloop start（后台 + 自启 + 自动打开浏览器）。
func runServer(parentCtx context.Context, log *slog.Logger, dataDir, host, externalHost string, port int) int {
	r := newRunner(dataDir, log)
	// 先 bootstrap（此时 cfg 里的 host/port 可能被 CLI 覆盖）
	_, err := r.bootstrap()
	if err != nil {
		log.Error(logPrefix+" bootstrap 失败", "err", err)
		return 1
	}
	// 存储维护：清理旧版本二进制 + 过期 backup（best-effort，不阻断启动）
	PruneOldVersions(dataDir, log)
	CleanupOldBackups(dataDir, log)
	// CLI 覆盖 cfg 的 host / port（cfg 会被 server.New 读取）
	if host != "" && host != r.cfg.Host {
		r.cfg.Host = host
	}
	if port > 0 && port != r.cfg.Port {
		r.cfg.Port = port
	}
	if externalHost != "" {
		if err := os.Setenv("GLOOP_DASHBOARD_EXTERNAL_HOST", externalHost); err != nil {
			log.Error(logPrefix+" 设置 external host 失败", "err", err)
			return 1
		}
	}
	// token：按（可能被 CLI 覆盖后的）host/port 生成；server.New 会再覆盖实际绑定 port
	tok, err := auth.Generate(r.root.Path(), r.cfg.Host, r.cfg.Port, false)
	if err != nil {
		log.Error(logPrefix+" 生成 token 失败", "err", err)
		return 1
	}

	srv, err := server.New(r.cfg, r.engine, r.bus, r.root, tok)
	if err != nil {
		log.Error(logPrefix+" 创建 HTTP server 失败", "err", err)
		return 1
	}

	// 写入 pidfile：放在 server.New 之后，保证端口确定；由真正的 server 进程写，
	// 避免 detach 场景下父进程先写父 pid 导致子进程误判"已在运行"。
	if err := writePidFile(r.root.Path(), os.Getpid()); err != nil {
		log.Warn(logPrefix+" 写入 pidfile 失败", "err", err)
	}
	// 最佳努力：进程退出时清理 pidfile（SIGKILL 下清理不到，但 checkRunning 会清）
	defer func() { _ = os.Remove(pidFilePath(r.root.Path())) }()

	// 捕获 SIGINT / SIGTERM，优雅关闭
	ctx, stop := signal.NotifyContext(parentCtx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// 打印 dashboard URL（带 token）
	fmt.Printf("\n🟢 %s Server 启动成功\n", version.AppName)
	fmt.Printf("   Listen:      http://%s\n", srv.Addr)
	fmt.Printf("   Data Dir:    %s\n", r.root.Path())
	fmt.Printf("   Dashboard:   %s\n", tok.DashboardURL())
	fmt.Printf("   (Ctrl+C 关闭)\n\n")

	errCh := make(chan error, 1)
	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		fmt.Fprintln(os.Stderr, "\n收到退出信号，优雅关闭…")
		// 30s：给 engine.Shutdown drain 在跑 quest 留足时间（agent turn 收尾），
		// httpSrv.Shutdown 用剩余预算收尾。
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Warn(logPrefix+" Shutdown 不优雅", "err", err)
		}
		<-errCh
	case err := <-errCh:
		if err != nil {
			log.Error(logPrefix+" HTTP server 异常退出", "err", err)
			return 1
		}
	}
	log.Info(logPrefix + " 已退出")
	return 0
}

// =========================================================================
// quest
// =========================================================================
