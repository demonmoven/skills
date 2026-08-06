package cli

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
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
	"strings"
	"time"

	"code.byted.org/lihuanyu.0w0/gloop/internal/updatecheck"
	"code.byted.org/lihuanyu.0w0/gloop/internal/version"
)

// =========================================================================
// gloop update
// =========================================================================

// updateFlags 集中管理 update 命令的所有 flag。
type updateFlags struct {
	check     bool   // 只检查不升级
	version   string // 指定版本（空 = latest）
	noRestart bool   // 升级后不自动重启 daemon
	dataDir   string // 数据目录（给 restart 用）
}

func parseUpdateFlags(args []string) (*updateFlags, error) {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	f := &updateFlags{}
	fs.BoolVar(&f.check, "check", false, "只检查更新，不实际升级")
	fs.StringVar(&f.version, "version", "", "升级到指定版本（默认 latest）")
	fs.BoolVar(&f.noRestart, "no-restart", false, "升级后不自动重启 daemon")
	fs.StringVar(&f.dataDir, "data-dir", "", "数据目录（默认 ~/."+version.RepoName+"）")
	// 自定义 usage，让 -h 正常退出（flag 默认 ContinueOnError + -h 会返回 ErrHelp）
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "用法: gloop update [flags]\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil, errHelpSentinel
		}
		return nil, err
	}
	return f, nil
}

// errHelpSentinel 用于区分"用户请求了 help"和"真正的参数错误"。
var errHelpSentinel = errors.New("help requested")

// runUpdateCmd 实现 `gloop update`：
//  1. 检查最新版本
//  2. 有更新则调用 npm install -g 升级
//  3. 如果 daemon 在运行，自动重启
func runUpdateCmd(ctx context.Context, log *slog.Logger, args []string) int {
	f, err := parseUpdateFlags(args)
	if err != nil {
		if errors.Is(err, errHelpSentinel) {
			return 0
		}
		return 2
	}
	dd, err := resolveDataDir(f.dataDir)
	if err != nil {
		log.Error(logPrefix+" 解析数据目录失败", "err", err)
		return 1
	}

	// ---- 步骤 1：解析 npm 实际可见的目标版本 ----
	fmt.Fprintf(os.Stderr, "🔍 正在检查更新…\n")

	current := version.Version
	requestedVersion := "latest"
	if f.version != "" {
		requestedVersion = f.version
	}

	npmPath, err := findNPM()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ 找不到 npm: %v\n", err)
		fmt.Fprintf(os.Stderr, "   请先安装 Node.js / npm，或手动执行: %s\n", manualInstallCommand(updatecheck.PackageName))
		return 1
	}

	target, err := resolveNPMTarget(ctx, npmPath, requestedVersion)
	if err != nil {
		fmt.Fprintf(os.Stderr, "   当前版本: %s\n", current)
		if f.version != "" {
			fmt.Fprintf(os.Stderr, "   指定版本: %s\n", f.version)
		} else {
			fmt.Fprintf(os.Stderr, "   最新版本: （查询失败）\n")
		}
		fmt.Fprintf(os.Stderr, "   错误: %v\n", err)
		return 1
	}
	targetVersion := target.Version

	fmt.Fprintf(os.Stderr, "   当前版本: %s\n", current)
	if f.version != "" {
		fmt.Fprintf(os.Stderr, "   指定版本: %s\n", f.version)
		if targetVersion != f.version {
			fmt.Fprintf(os.Stderr, "   解析版本: %s\n", targetVersion)
		}
	} else {
		fmt.Fprintf(os.Stderr, "   最新版本: %s (%s)\n", targetVersion, mapStateToLabel(updatecheck.DeriveState(current, targetVersion)))
	}

	// ---- 只检查模式 ----
	if f.check {
		switch updateCheckHintFor(current, targetVersion, f.version) {
		case updateCheckHintSwitchVersion:
			fmt.Fprintf(os.Stderr, "\n💡 当前版本与目标版本不同，执行 `gloop update --version %s` 切换\n", f.version)
		case updateCheckHintUpgradeLatest:
			fmt.Fprintf(os.Stderr, "\n💡 有新版本可用，执行 `gloop update` 升级\n")
		case updateCheckHintUnknown:
			fmt.Fprintf(os.Stderr, "\n⚠️  无法可靠比较版本\n")
		default:
			fmt.Fprintf(os.Stderr, "\n✅ 已是最新版本\n")
		}
		return 0
	}

	// ---- 步骤 2：判断是否需要升级 ----
	needUpdate := false
	if f.version != "" {
		// 指定版本：和当前不一样就升级
		needUpdate = targetVersion != current
	} else {
		// 最新版本模式：outdated 才升级
		needUpdate = updatecheck.DeriveState(current, targetVersion) == "outdated"
	}

	if !needUpdate {
		if f.version != "" || updatecheck.DeriveState(current, targetVersion) == "up-to-date" {
			fmt.Fprintf(os.Stderr, "\n✅ 已是目标版本，无需升级\n")
			return 0
		}
		fmt.Fprintf(os.Stderr, "\n⚠️  无法可靠比较版本，跳过升级\n")
		return 1
	}

	// ---- 步骤 3：安装前校验当前平台可安装性 ----
	fmt.Fprintf(os.Stderr, "\n🧪 正在验证安装包…\n")
	prepared, err := prepareNPMInstallPackage(ctx, npmPath, targetVersion)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ 安装包验证失败: %v\n", err)
		fmt.Fprintf(os.Stderr, "   发布产物可能尚未同步完成，请稍后重试\n")
		return 1
	}
	defer prepared.Cleanup()

	// ---- 步骤 4：执行 npm 升级 ----
	fmt.Fprintf(os.Stderr, "\n⬆️  正在升级到 %s…\n", targetVersion)

	pkgSpec := prepared.PackageSpec
	installArgs := buildNPMInstallArgs(pkgSpec)

	cmd := exec.CommandContext(ctx, npmPath, installArgs...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(),
		// 确保 npm 使用 bnpm registry，即使全局配置了其他源
		"npm_config_registry="+updatecheck.Registry,
	)

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "\n❌ 升级失败: %v\n", err)
		fmt.Fprintf(os.Stderr, "   你可以手动执行: %s\n", manualInstallCommand(updatecheck.PackageName+"@"+targetVersion))
		return 1
	}

	fmt.Fprintf(os.Stderr, "\n✅ 升级完成\n")

	// ---- 步骤 5：如果 daemon 在运行，自动重启 ----
	if f.noRestart {
		fmt.Fprintf(os.Stderr, "   （--no-restart，跳过 daemon 重启）\n")
		return 0
	}

	pid, running := checkRunning(dd)
	if !running {
		fmt.Fprintf(os.Stderr, "   daemon 未运行，跳过重启\n")
		fmt.Fprintf(os.Stderr, "   执行 `gloop start` 启动服务\n")
		return 0
	}

	fmt.Fprintf(os.Stderr, "🔄 检测到 daemon (pid=%d) 在运行，正在重启…\n", pid)

	// 通过稳定 shim 启动重启命令。当前 update 命令仍运行在升级前的 native binary 里；
	// 如果直接调用 runRestartCmd，会用旧 os.Executable() fork 出旧版本 daemon。
	self, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  定位当前可执行文件失败，回退到当前进程重启: %v\n", err)
		return runRestartCmd(ctx, log, []string{"--data-dir", dd, "--no-open"})
	}
	restartPath, restartArgs := buildPostUpdateRestartCommand(self, dd)
	cmd = exec.CommandContext(ctx, restartPath, restartArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = os.Environ()
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ daemon 重启失败: %v\n", err)
		fmt.Fprintf(os.Stderr, "   请手动执行: %s restart --data-dir %s --no-open\n", restartPath, dd)
		return 1
	}
	return 0
}

// mapStateToLabel 把 state 字符串转成人类可读的中文标签。
func mapStateToLabel(state string) string {
	switch state {
	case "outdated":
		return "有更新"
	case "up-to-date":
		return "最新"
	case "unknown":
		return "未知"
	default:
		return state
	}
}

func buildPostUpdateRestartCommand(self, dataDir string) (string, []string) {
	return stableShim(self), []string{"restart", "--data-dir", dataDir, "--no-open"}
}

func buildNPMInstallArgs(pkgSpec string) []string {
	// 使用 --registry 指定 bnpm，和 package.json 的 publishConfig 保持一致。
	// --prefer-online 让 npm 重新校验 registry 元数据，减少刚发布版本被本地缓存误判为 ETARGET。
	return []string{
		"install",
		"-g",
		"--registry", updatecheck.Registry,
		"--prefer-online",
		pkgSpec,
	}
}

type updateCheckHint string

const (
	updateCheckHintCurrent       updateCheckHint = "current"
	updateCheckHintSwitchVersion updateCheckHint = "switch-version"
	updateCheckHintUpgradeLatest updateCheckHint = "upgrade-latest"
	updateCheckHintUnknown       updateCheckHint = "unknown"
)

func updateCheckHintFor(current, targetVersion, requestedVersion string) updateCheckHint {
	if requestedVersion != "" {
		if targetVersion != current {
			return updateCheckHintSwitchVersion
		}
		return updateCheckHintCurrent
	}
	switch updatecheck.DeriveState(current, targetVersion) {
	case "outdated":
		return updateCheckHintUpgradeLatest
	case "up-to-date":
		return updateCheckHintCurrent
	default:
		return updateCheckHintUnknown
	}
}

func manualInstallCommand(pkgSpec string) string {
	return fmt.Sprintf("npm i -g --registry %s --prefer-online %s", updatecheck.Registry, pkgSpec)
}

type npmViewInfo struct {
	Version string
	Tarball string
}

func buildNPMViewArgs(pkgSpec string) []string {
	return []string{
		"view",
		pkgSpec,
		"version",
		"dist.tarball",
		"--registry", updatecheck.Registry,
		"--prefer-online",
		"--json",
	}
}

func parseNPMViewOutput(raw []byte) (npmViewInfo, error) {
	var payload struct {
		Version     string `json:"version"`
		DistTarball string `json:"dist.tarball"`
		Dist        struct {
			Tarball string `json:"tarball"`
		} `json:"dist"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(raw), &payload); err != nil {
		return npmViewInfo{}, fmt.Errorf("parse npm view output: %w", err)
	}
	info := npmViewInfo{
		Version: strings.TrimSpace(payload.Version),
		Tarball: strings.TrimSpace(payload.DistTarball),
	}
	if info.Tarball == "" {
		info.Tarball = strings.TrimSpace(payload.Dist.Tarball)
	}
	if info.Version == "" {
		return npmViewInfo{}, errors.New("npm returned empty version")
	}
	if info.Tarball == "" {
		return npmViewInfo{}, errors.New("npm returned empty tarball")
	}
	return info, nil
}

func resolveNPMTarget(ctx context.Context, npmPath, requested string) (npmViewInfo, error) {
	pkgSpec := updatecheck.PackageName + "@" + requested
	cmd := exec.CommandContext(ctx, npmPath, buildNPMViewArgs(pkgSpec)...)
	cmd.Env = append(os.Environ(), "npm_config_registry="+updatecheck.Registry)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return npmViewInfo{}, fmt.Errorf("npm view %s failed: %w: %s", pkgSpec, err, msg)
		}
		return npmViewInfo{}, fmt.Errorf("npm view %s failed: %w", pkgSpec, err)
	}
	return parseNPMViewOutput(out)
}

func currentNativeArchiveName(ver string) (string, error) {
	platform := runtime.GOOS
	ext := ".tar.gz"
	switch runtime.GOOS {
	case "darwin", "linux":
	case "windows":
		platform = "windows"
		ext = ".zip"
	default:
		return "", fmt.Errorf("unsupported platform: %s-%s", runtime.GOOS, runtime.GOARCH)
	}
	switch runtime.GOARCH {
	case "amd64", "arm64":
	default:
		return "", fmt.Errorf("unsupported platform: %s-%s", runtime.GOOS, runtime.GOARCH)
	}
	return fmt.Sprintf("%s-%s-%s-%s%s", version.RepoName, ver, platform, runtime.GOARCH, ext), nil
}

type npmPackResult struct {
	Filename string `json:"filename"`
}

func buildNPMPackArgs(pkgSpec, destDir string) []string {
	return []string{
		"pack",
		pkgSpec,
		"--registry", updatecheck.Registry,
		"--prefer-online",
		"--ignore-scripts",
		"--json",
		"--pack-destination", destDir,
	}
}

func parseNPMPackOutput(raw []byte) (npmPackResult, error) {
	var payload []npmPackResult
	if err := json.Unmarshal(bytes.TrimSpace(raw), &payload); err != nil {
		return npmPackResult{}, fmt.Errorf("parse npm pack output: %w", err)
	}
	if len(payload) == 0 || strings.TrimSpace(payload[0].Filename) == "" {
		return npmPackResult{}, errors.New("npm pack returned no filename")
	}
	return payload[0], nil
}

func readPackedPackage(tgzPath string) (map[string]bool, string, error) {
	f, err := os.Open(tgzPath)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, "", err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	paths := map[string]bool{}
	checksums := ""
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, "", err
		}
		name := strings.TrimPrefix(h.Name, "package/")
		if name == "" {
			continue
		}
		paths[name] = true
		if name == "checksums.txt" {
			buf, err := io.ReadAll(io.LimitReader(tr, 1<<20))
			if err != nil {
				return nil, "", err
			}
			checksums = string(buf)
		}
	}
	return paths, checksums, nil
}

func checkPackedPackageInstallability(ver string, paths map[string]bool, checksums string) (bool, error) {
	archive, err := currentNativeArchiveName(ver)
	if err != nil {
		return false, err
	}
	if !paths["checksums.txt"] {
		return false, errors.New("package is missing checksums.txt")
	}
	if !checksumHasEntry(checksums, archive) {
		return false, fmt.Errorf("checksums.txt is missing entry for %s", archive)
	}
	return paths[filepath.ToSlash(filepath.Join("dist", archive))], nil
}

func checksumHasEntry(content, filename string) bool {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		idx := strings.Index(line, "  ")
		if idx < 0 {
			continue
		}
		if strings.TrimSpace(line[idx+2:]) == filename {
			return true
		}
	}
	return false
}

type preparedNPMInstall struct {
	PackageSpec string
	Cleanup     func()
}

func prepareNPMInstallPackage(ctx context.Context, npmPath, ver string) (preparedNPMInstall, error) {
	tmp, err := os.MkdirTemp("", "gloop-update-pack-")
	if err != nil {
		return preparedNPMInstall{}, err
	}
	cleanup := func() { _ = os.RemoveAll(tmp) }
	keep := false
	defer func() {
		if !keep {
			cleanup()
		}
	}()

	pkgSpec := updatecheck.PackageName + "@" + ver
	cmd := exec.CommandContext(ctx, npmPath, buildNPMPackArgs(pkgSpec, tmp)...)
	cmd.Env = append(os.Environ(), "npm_config_registry="+updatecheck.Registry)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return preparedNPMInstall{}, fmt.Errorf("npm pack %s failed: %w: %s", pkgSpec, err, msg)
		}
		return preparedNPMInstall{}, fmt.Errorf("npm pack %s failed: %w", pkgSpec, err)
	}
	pack, err := parseNPMPackOutput(out)
	if err != nil {
		return preparedNPMInstall{}, err
	}
	tgzPath := filepath.Join(tmp, pack.Filename)
	paths, checksums, err := readPackedPackage(tgzPath)
	if err != nil {
		return preparedNPMInstall{}, fmt.Errorf("inspect npm package: %w", err)
	}
	bundled, err := checkPackedPackageInstallability(ver, paths, checksums)
	if err != nil {
		return preparedNPMInstall{}, err
	}
	if bundled {
		keep = true
		return preparedNPMInstall{PackageSpec: tgzPath, Cleanup: cleanup}, nil
	}

	urls, err := binaryDownloadURLs(ver)
	if err != nil {
		return preparedNPMInstall{}, err
	}
	for _, url := range urls {
		available, err := remoteBinaryExists(ctx, url)
		if err == nil && available {
			keep = true
			return preparedNPMInstall{PackageSpec: tgzPath, Cleanup: cleanup}, nil
		}
	}
	archive, _ := currentNativeArchiveName(ver)
	return preparedNPMInstall{}, fmt.Errorf("package does not bundle %s and remote binary is unreachable", archive)
}

func binaryDownloadURLs(ver string) ([]string, error) {
	archive, err := currentNativeArchiveName(ver)
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/-/binary/%s/v%s/%s", version.RepoName, ver, archive)
	urls := []string{}
	if explicit := strings.TrimSpace(os.Getenv("GLOOP_BINARY_BASE_URL")); explicit != "" {
		if !strings.HasPrefix(explicit, "https://") {
			return nil, errors.New("GLOOP_BINARY_BASE_URL must be an https URL")
		}
		urls = append(urls, strings.TrimRight(explicit, "/")+"/v"+ver+"/"+archive)
	}
	registry := strings.TrimSpace(os.Getenv("npm_config_registry"))
	if registry == "" {
		registry = updatecheck.Registry
	}
	if strings.HasPrefix(registry, "https://") {
		urls = append(urls, strings.TrimRight(registry, "/")+path)
	}
	fallback := strings.TrimRight(updatecheck.Registry, "/") + path
	for _, url := range urls {
		if url == fallback {
			return urls, nil
		}
	}
	return append(urls, fallback), nil
}

func remoteBinaryExists(ctx context.Context, url string) (bool, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	for _, method := range []string{http.MethodHead, http.MethodGet} {
		req, err := http.NewRequestWithContext(ctx, method, url, nil)
		if err != nil {
			return false, err
		}
		if method == http.MethodGet {
			req.Header.Set("Range", "bytes=0-0")
		}
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1))
		_ = resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return true, nil
		}
	}
	return false, nil
}

// findNPM 查找 npm 可执行文件路径。
// 优先用 PATH 里的 npm；找不到的话尝试常见 Node 安装路径。
func findNPM() (string, error) {
	// 1. PATH 查找
	if p, err := exec.LookPath("npm"); err == nil {
		return p, nil
	}

	// 2. 常见 Node 安装路径兜底
	candidates := []string{}
	switch runtime.GOOS {
	case "darwin":
		candidates = []string{
			"/usr/local/bin/npm",
			"/opt/homebrew/bin/npm",
			"/usr/local/bin/npm",
		}
	case "linux":
		candidates = []string{
			"/usr/bin/npm",
			"/usr/local/bin/npm",
			"/snap/bin/npm",
		}
	case "windows":
		candidates = []string{
			`C:\Program Files\nodejs\npm.cmd`,
			`C:\Program Files (x86)\nodejs\npm.cmd`,
		}
	}

	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	// 3. 尝试从环境变量推断
	if nodePath := os.Getenv("NODE_PATH"); nodePath != "" {
		// NODE_PATH 通常是模块目录，往上找 bin
		// 简单起见不做复杂推导
	}

	return "", errors.New("未在 PATH 和常见位置找到 npm")
}
