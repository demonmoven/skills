// Package updatecheck 实现对 bnpm registry 的"最新版本"轮询，
// 给 CLI（daemon 常驻）和 dashboard 用户提示"有新版本可升级"。
//
// 设计原则（第一性原理）：
//   - 不阻塞主流程：任何网络/解析/IO 失败都要被吞掉。
//   - 不做 DDoS：最短轮询间隔 24h，结果落到 ~/.gloop/update-check.json，
//     跨进程共享（daemon 常驻 + 短命令都能复用缓存）。
//   - 可关闭：GLOOP_NO_UPDATE_NOTIFIER=1 / NO_UPDATE_NOTIFIER 时完全跳过。
//   - 不自动升级：升级动作由用户手动执行 gloop update，避免权限/签名/回滚坑。
package updatecheck

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"code.byted.org/lihuanyu.0w0/gloop/internal/version"
)

// ============================================================
// 常量
// ============================================================

const (
	// PackageName npm 包名；保持和 package.json 同步。
	PackageName = "@bytedance-dev/gloop"
	// Registry 内部 bnpm 地址；publishConfig.registry 也是它。
	Registry = "https://bnpm.byted.org"
	// DefaultInterval 两次检查的最小间隔。
	DefaultInterval = 24 * time.Hour
	// DefaultRequestTimeout 单次 HTTP 请求超时，卡网络不能卡 daemon。
	DefaultRequestTimeout = 5 * time.Second
	// cacheFile 相对数据根目录的路径。
	cacheFile = "update-check.json"
	// defaultUpgradeHint 给 CLI / Dashboard 直接展示。
	defaultUpgradeHint = "gloop update"
)

// ============================================================
// 对外结构
// ============================================================

// Status 是一次检查的结果，给 API 直接序列化出去。
type Status struct {
	// 当前运行的版本（来自 internal/version.Version）。
	Current string `json:"current"`
	// 最新版本；如果没查到/尚未检查则为空字符串。
	Latest string `json:"latest,omitempty"`
	// 比较结果："outdated" / "up-to-date" / "unknown"
	State string `json:"state"`
	// 最近一次实际发起网络检查的时间（unix ms，0 表示从未查过）。
	LastCheckedAt int64 `json:"last_checked_at,omitempty"`
	// 下次允许检查的时间（unix ms）—— 前端可选展示，防止乱点刷新。
	NextCheckAt int64 `json:"next_check_at,omitempty"`
	// 推荐用户执行的升级命令，原样展示在 banner/CLI。
	UpgradeHint string `json:"upgrade_hint,omitempty"`
	// 最近一次失败的错误信息，给排障用；API 暴露但 UI 不默认展示。
	LastError string `json:"last_error,omitempty"`
}

// Checker 封装轮询 + 缓存 + 并发访问。典型用法：
//
//	chk := updatecheck.New(dataDir)
//	chk.Start(ctx)
//	http.HandleFunc("/api/update-status", chk.HandleHTTP)
type Checker struct {
	dataDir  string
	interval time.Duration

	mu     sync.RWMutex
	status Status // 内存中的最新状态（和磁盘一致）

	// 防止并发发出多次 HTTP 请求。
	checking atomic.Bool
}

// cacheEntry 是磁盘 JSON 的结构，比 Status 多了 NextCheckAt 用于节流。
type cacheEntry struct {
	Current       string `json:"current"`
	Latest        string `json:"latest,omitempty"`
	State         string `json:"state"`
	LastCheckedAt int64  `json:"last_checked_at,omitempty"`
	NextCheckAt   int64  `json:"next_check_at,omitempty"`
	LastError     string `json:"last_error,omitempty"`
}

// ============================================================
// 构造
// ============================================================

// New 新建一个 Checker。dataDir 是 ~/.gloop 的绝对路径（由 fsstore 给）。
// 如果 dataDir 为空，缓存会写到 os.UserHomeDir() 兜底。
func New(dataDir string) *Checker {
	c := &Checker{
		dataDir:  dataDir,
		interval: DefaultInterval,
		status: Status{
			Current:     version.Version,
			State:       "unknown",
			UpgradeHint: defaultUpgradeHint,
		},
	}
	c.loadCache()
	return c
}

// Disabled 返回 true 时，Checker 不做任何网络请求，只返回 current 版本。
func Disabled() bool {
	return os.Getenv("GLOOP_NO_UPDATE_NOTIFIER") == "1" ||
		os.Getenv("NO_UPDATE_NOTIFIER") != ""
}

// ============================================================
// 生命周期（给 server 用）
// ============================================================

// Start 启动后台周期性检查。ctx 取消时退出。
// 首次调用会立即触发一次检查（如果超过节流窗口）。
func (c *Checker) Start(ctx context.Context) {
	if Disabled() {
		return
	}
	// 启动即先查一次（走节流保护；24h 内查过就直接用缓存）。
	go c.Check(ctx)
	go c.loop(ctx)
}

func (c *Checker) loop(ctx context.Context) {
	// interval 做 2% 左右的抖动，避免一堆 daemon 同时打 registry。
	jitter := time.Duration(float64(c.interval) * (0.01 + randFrac()))
	t := time.NewTimer(c.interval + jitter)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			c.Check(ctx)
			jitter = time.Duration(float64(c.interval) * (0.01 + randFrac()))
			t.Reset(c.interval + jitter)
		}
	}
}

// ============================================================
// 核心检查逻辑
// ============================================================

// Check 主动触发一次 registry 查询（受节流保护）。
// 并发调用只会真正发一次请求；失败会写到 Status.LastError，但不返回 error。
func (c *Checker) Check(ctx context.Context) {
	c.check(ctx, false)
}

// ForceCheck 和 Check 类似，但忽略节流，强制发起一次网络请求。
// 给 `gloop update` / Dashboard 的"手动刷新"按钮用。
func (c *Checker) ForceCheck(ctx context.Context) {
	c.check(ctx, true)
}

func (c *Checker) check(ctx context.Context, force bool) {
	if Disabled() {
		return
	}
	now := time.Now()
	if !force {
		c.mu.RLock()
		next := c.status.NextCheckAt
		c.mu.RUnlock()
		if next > 0 && now.UnixMilli() < next {
			return
		}
	}
	if !c.checking.CompareAndSwap(false, true) {
		return
	}
	defer c.checking.Store(false)

	latest, err := fetchLatest(ctx)
	checkedAt := time.Now().UnixMilli()

	c.recordCheckResult(latest, err, checkedAt)
}

func (c *Checker) recordCheckResult(latest string, err error, checkedAt int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.status.Current = version.Version
	c.status.LastCheckedAt = checkedAt
	c.status.NextCheckAt = checkedAt + int64(c.interval/time.Millisecond)
	if err != nil {
		c.status.LastError = err.Error()
		c.status.Latest = ""
		c.status.State = "unknown"
	} else {
		c.status.LastError = ""
		c.status.Latest = latest
		c.status.State = deriveState(version.Version, latest)
	}
	c.status.UpgradeHint = defaultUpgradeHint
	c.saveCacheLocked()
}

// ============================================================
// 状态读取 + HTTP Handler
// ============================================================

// Status 返回当前状态的拷贝（线程安全）。
func (c *Checker) Get() Status {
	c.mu.RLock()
	defer c.mu.RUnlock()
	s := c.status
	return s
}

// HandleHTTP 给 dashboard 用，返回 JSON 状态。
//   - ?refresh=1：触发一次检查（仍受节流保护，防止狂点刷爆 registry）
//   - ?force=1：强制检查，忽略节流（给"立即检查"按钮用，要慎重）
func (c *Checker) HandleHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("force") == "1" {
		c.ForceCheck(r.Context())
	} else if r.URL.Query().Get("refresh") == "1" {
		c.Check(r.Context())
	}
	s := c.Get()
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(s)
}

// ============================================================
// registry 查询
// ============================================================

// fetchLatest 向 bnpm registry 查询 @bytedance-dev/gloop 的 latest 版本。
// 返回纯版本字符串，如 "0.1.2"。
func fetchLatest(ctx context.Context) (string, error) {
	// URL: https://bnpm.byted.org/@bytedance-dev%2Fgloop/latest
	// scoped package 的 slash 必须编码，避免 registry 路由按多级路径解释。
	url := Registry + "/" + registryPackagePath(PackageName) + "/latest"
	reqCtx, cancel := context.WithTimeout(ctx, DefaultRequestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	// 走 UA 让 registry 能简单区分 gloop daemon 流量和普通 npm 流量。
	req.Header.Set("User-Agent", fmt.Sprintf(
		"gloop/%s (%s; %s) update-check",
		version.Version, runtime.GOOS, runtime.GOARCH,
	))
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("registry request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("registry http %d", resp.StatusCode)
	}
	var payload registryLatestPayload
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("parse registry response: %w", err)
	}
	latest, err := latestVersionFromRegistryPayload(payload)
	if err != nil {
		return "", err
	}
	return latest, nil
}

type registryLatestPayload struct {
	Version  string            `json:"version"`
	DistTags map[string]string `json:"dist-tags"`
	Versions json.RawMessage   `json:"versions"`
}

func registryPackagePath(name string) string {
	return strings.ReplaceAll(name, "/", "%2F")
}

func latestVersionFromRegistryPayload(payload registryLatestPayload) (string, error) {
	latest := strings.TrimSpace(payload.Version)
	if tagLatest := strings.TrimSpace(payload.DistTags["latest"]); tagLatest != "" {
		latest = tagLatest
	}
	if latest == "" {
		return "", errors.New("registry returned empty version")
	}
	if len(payload.Versions) > 0 {
		contains, err := registryVersionsContain(payload.Versions, latest)
		if err != nil {
			return "", err
		}
		if !contains {
			return "", fmt.Errorf("registry latest %q is not present in package versions", latest)
		}
	}
	return latest, nil
}

func registryVersionsContain(raw json.RawMessage, version string) (bool, error) {
	var list []string
	if err := json.Unmarshal(raw, &list); err == nil {
		for _, item := range list {
			if item == version {
				return true, nil
			}
		}
		return false, nil
	}

	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err == nil {
		_, ok := obj[version]
		return ok, nil
	}

	return false, errors.New("registry returned invalid versions shape")
}

// ============================================================
// 缓存读写
// ============================================================

func (c *Checker) cachePath() string {
	root := c.dataDir
	if root == "" {
		if h, err := os.UserHomeDir(); err == nil {
			root = filepath.Join(h, "."+version.RepoName)
		}
	}
	return filepath.Join(root, cacheFile)
}

func (c *Checker) loadCache() {
	p := c.cachePath()
	raw, err := os.ReadFile(p)
	if err != nil {
		return
	}
	var e cacheEntry
	if err := json.Unmarshal(raw, &e); err != nil {
		return
	}
	// 数据目录/版本有变动就不认旧 Latest，强制下次重查。
	nowMs := time.Now().UnixMilli()
	c.mu.Lock()
	defer c.mu.Unlock()
	c.status.Current = version.Version
	c.status.LastCheckedAt = e.LastCheckedAt
	c.status.NextCheckAt = e.NextCheckAt
	c.status.LastError = e.LastError
	if e.Current == version.Version && e.NextCheckAt > nowMs {
		c.status.Latest = e.Latest
		c.status.State = deriveState(version.Version, e.Latest)
	} else {
		// 缓存过期或当前版本不一致：把 state 打回 unknown，下次 Check 会覆盖。
		// 版本不一致时尤其不能沿用旧 NextCheckAt，否则启动检查会被旧节流挡住。
		c.status.Latest = ""
		c.status.State = "unknown"
		c.status.NextCheckAt = 0
	}
	c.status.UpgradeHint = defaultUpgradeHint
}

// saveCacheLocked 把内存状态写入磁盘。调用方必须持有 c.mu 写锁。
func (c *Checker) saveCacheLocked() {
	p := c.cachePath()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return
	}
	entry := cacheEntry{
		Current:       c.status.Current,
		Latest:        c.status.Latest,
		State:         c.status.State,
		LastCheckedAt: c.status.LastCheckedAt,
		NextCheckAt:   c.status.NextCheckAt,
		LastError:     c.status.LastError,
	}
	buf, err := json.Marshal(entry)
	if err != nil {
		return
	}
	// 原子写：tmp + rename，防止 daemon 崩溃时写半页 JSON。
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, buf, 0o644); err != nil {
		return
	}
	_ = os.Rename(tmp, p)
}

// ============================================================
// 工具
// ============================================================

// deriveState 比较 current vs latest，返回 "outdated" / "up-to-date" / "unknown"。
// 只在能证明 latest 语义化版本严格高于 current 时提示，避免 registry 还没同步或
// dist-tag 回退时把 0.1.24 -> 0.1.23 误报成升级。
func deriveState(current, latest string) string {
	if latest == "" {
		return "unknown"
	}
	if current == latest {
		return "up-to-date"
	}
	currentSemver, currentOK := parseSemver(current)
	latestSemver, latestOK := parseSemver(latest)
	if currentOK && latestOK {
		// current 是正式版、latest 是 rc/canary —— 不催更。
		if latestSemver.pre != "" && currentSemver.pre == "" {
			return "up-to-date"
		}
		if compareSemver(latestSemver, currentSemver) > 0 {
			return "outdated"
		}
		return "up-to-date"
	}
	// 无法可靠比较时保持静默；版本提示是附加值，不能误报降级。
	return "unknown"
}

// DeriveState exposes the same conservative version comparison used by
// background update checks for foreground update flows.
func DeriveState(current, latest string) string {
	return deriveState(current, latest)
}

type semverVersion struct {
	major int
	minor int
	patch int
	pre   string
}

func parseSemver(raw string) (semverVersion, bool) {
	v := strings.TrimSpace(raw)
	v = strings.TrimPrefix(v, "v")
	if v == "" {
		return semverVersion{}, false
	}
	if buildIdx := strings.IndexByte(v, '+'); buildIdx >= 0 {
		v = v[:buildIdx]
	}
	pre := ""
	if preIdx := strings.IndexByte(v, '-'); preIdx >= 0 {
		pre = v[preIdx+1:]
		v = v[:preIdx]
		if pre == "" {
			return semverVersion{}, false
		}
	}
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return semverVersion{}, false
	}
	nums := [3]int{}
	for i, p := range parts {
		n, ok := parseNumericIdentifier(p)
		if !ok {
			return semverVersion{}, false
		}
		nums[i] = n
	}
	return semverVersion{major: nums[0], minor: nums[1], patch: nums[2], pre: pre}, true
}

func compareSemver(a, b semverVersion) int {
	if a.major != b.major {
		return compareInt(a.major, b.major)
	}
	if a.minor != b.minor {
		return compareInt(a.minor, b.minor)
	}
	if a.patch != b.patch {
		return compareInt(a.patch, b.patch)
	}
	if a.pre == b.pre {
		return 0
	}
	if a.pre == "" {
		return 1
	}
	if b.pre == "" {
		return -1
	}
	return comparePrerelease(a.pre, b.pre)
}

func comparePrerelease(a, b string) int {
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")
	limit := len(aParts)
	if len(bParts) < limit {
		limit = len(bParts)
	}
	for i := 0; i < limit; i++ {
		if aParts[i] == bParts[i] {
			continue
		}
		aNum, aOK := parseNumericIdentifier(aParts[i])
		bNum, bOK := parseNumericIdentifier(bParts[i])
		if aOK && bOK {
			return compareInt(aNum, bNum)
		}
		if aOK {
			return -1
		}
		if bOK {
			return 1
		}
		if aParts[i] < bParts[i] {
			return -1
		}
		return 1
	}
	return compareInt(len(aParts), len(bParts))
}

func parseNumericIdentifier(s string) (int, bool) {
	if s == "" {
		return 0, false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, false
		}
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return n, true
}

func compareInt(a, b int) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

// randFrac 返回一个伪随机 [0, 1) 小数，不依赖 math/rand 全局 seed。
func randFrac() float64 {
	// 用当前纳秒末位做简单扰动，2% 抖动足够了。
	n := uint32(time.Now().UnixNano() & 0xffff)
	return float64(n) / float64(0x10000)
}
