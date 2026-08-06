package auth

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"code.byted.org/lihuanyu.0w0/gloop/internal/version"
)

// 本地访问控制：启动时一次性生成随机密钥，写入数据目录的 token.json（权限 0600）。
// 打开 Dashboard / 访问时通过三种方式携带密钥：URL ?t= / Authorization: Bearer / Cookie。
// 通过后写入 30 天 Cookie，避免之后每次都在地址栏暴露密钥。
//
// 说明：gloop 是本地工具，不做用户登录/鉴权。这里的"密钥"本质是一次性绑定密钥，
// 防止同一开发机上其它进程误读写 ~/.gloop。

const (
	TokenBytes   = 24
	CookieName   = "ag_token"
	CookieMaxAge = 30 * 24 * 3600 // 浏览器 Cookie 有效 30 天
)

type TokenStore struct {
	Token     string    `json:"token"`
	CreatedAt time.Time `json:"created_at"`
	Port      int       `json:"port,omitempty"`
	BindHost  string    `json:"bind_host,omitempty"`
}

func (t *TokenStore) DashboardURL() string {
	host := advertiseHost(t.BindHost)
	return fmt.Sprintf("http://%s:%d/dashboard?t=%s", host, t.Port, t.Token)
}

func (t *TokenStore) Save(dataDir string) error {
	if err := secureTokenDir(dataDir); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化 token 失败: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, version.TokenFileName), append(raw, '\n'), 0600); err != nil {
		return err
	}
	return nil
}

func advertiseHost(bindHost string) string {
	if host := os.Getenv("GLOOP_DASHBOARD_EXTERNAL_HOST"); host != "" {
		return host
	}
	if host := os.Getenv("WEB_EXTERNAL_HOST"); host != "" {
		return host
	}
	if bindHost != "" && bindHost != "0.0.0.0" && bindHost != "::" {
		return bindHost
	}
	if ip := firstNonLoopbackIPv4(); ip != "" {
		return ip
	}
	return "127.0.0.1"
}

func firstNonLoopbackIPv4() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
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
			if ip4 := ip.To4(); ip4 != nil && !ip4.IsLoopback() {
				return ip4.String()
			}
		}
	}
	return ""
}

// Generate 生成本地绑定密钥并落盘到 dataDir/token.json。若已存在则复用（除非 force=true）。
// 文件路径 = filepath.Join(dataDir, version.TokenFileName)。
func Generate(dataDir string, bindHost string, port int, force bool) (*TokenStore, error) {
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, err
	}
	if err := secureTokenDir(dataDir); err != nil {
		return nil, err
	}
	// 防御：父目录可能已经是 world-readable，检查并 warn
	if info, stErr := os.Stat(dataDir); stErr == nil && info.Mode()&0o077 != 0 {
		fmt.Fprintf(os.Stderr, "[warn] token dir %s is world-accessible (mode=%o)\n", dataDir, info.Mode())
	}
	p := filepath.Join(dataDir, version.TokenFileName)
	if !force {
		if raw, err := os.ReadFile(p); err == nil {
			var s TokenStore
			if json.Unmarshal(raw, &s) == nil && s.Token != "" {
				if s.Port == 0 {
					s.Port = port
				}
				if s.BindHost == "" {
					s.BindHost = bindHost
				}
				return &s, nil
			}
		}
	}
	buf := make([]byte, TokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return nil, err
	}
	s := &TokenStore{
		Token:     hex.EncodeToString(buf),
		CreatedAt: time.Now(),
		Port:      port,
		BindHost:  bindHost,
	}
	if err := s.Save(dataDir); err != nil {
		return nil, err
	}
	return s, nil
}

func secureTokenDir(dataDir string) error {
	info, err := os.Stat(dataDir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("token dir %s 不是目录", dataDir)
	}
	if info.Mode().Perm()&0o077 != 0 {
		if err := os.Chmod(dataDir, 0o700); err != nil {
			return fmt.Errorf("收紧 token dir 权限失败: %w", err)
		}
	}
	return nil
}

// Load 只读取不生成，用于 CLI 侧读取当前 token。
func Load(dataDir string) (*TokenStore, error) {
	raw, err := os.ReadFile(filepath.Join(dataDir, version.TokenFileName))
	if err != nil {
		return nil, err
	}
	var s TokenStore
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// Middleware 包装 http.Handler：先看 cookie，再看 query ?t=，再看 Authorization。
// 未通过返回 401（HTML 友好 + JSON 友好）。
//
// 说明：HTTP 状态码沿用 401/WWW-Authenticate: Bearer 是为了让标准客户端（curl / fetch / browser）
// 走默认提示路径，语义上是"缺少本地绑定密钥"，不是真正的用户登录。
func Middleware(store *TokenStore, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if store == nil || store.Token == "" {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusInternalServerError)
			if encErr := json.NewEncoder(w).Encode(map[string]interface{}{
				"ok": false, "error": "local token not generated (restart gloop server)",
			}); encErr != nil {
				fmt.Fprintf(os.Stderr, "[warn] token middleware write 500 response failed: %v\n", encErr)
			}
			return
		}
		// 1. cookie
		if c, err := r.Cookie(CookieName); err == nil && c != nil && c.Value == store.Token {
			next.ServeHTTP(w, r)
			return
		}
		// 2. query ?t=
		if t := r.URL.Query().Get("t"); t == store.Token && t != "" {
			http.SetCookie(w, &http.Cookie{
				Name:     CookieName,
				Value:    t,
				Path:     "/",
				MaxAge:   CookieMaxAge,
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
			})
			// strip t param 后重定向（避免 token 留在地址栏）
			q := r.URL.Query()
			q.Del("t")
			r.URL.RawQuery = q.Encode()
			next.ServeHTTP(w, r)
			return
		}
		// 3. Authorization: Bearer
		if ah := r.Header.Get("Authorization"); len(ah) > 7 && ah[:7] == "Bearer " {
			if ah[7:] == store.Token {
				next.ServeHTTP(w, r)
				return
			}
		}
		w.Header().Set("WWW-Authenticate", "Bearer")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusUnauthorized)
		if encErr := json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": false, "error": "missing local token",
			"hint": "open the dashboard URL printed by `gloop server start` (it carries the ?t= binding token)",
		}); encErr != nil {
			fmt.Fprintf(os.Stderr, "[warn] token middleware write 401 response failed: %v\n", encErr)
		}
	})
}
