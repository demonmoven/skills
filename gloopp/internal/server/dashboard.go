package server

import (
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"strings"

	webdist "code.byted.org/lihuanyu.0w0/gloop/web"
)

// dashboardFS 返回前端资源：默认用编译期 go:embed 的 dist；
// 设置 GLOOP_WEB_DIST 时改读磁盘目录，便于前端联调免重新编译。
func dashboardFS() fs.FS {
	if dir := os.Getenv("GLOOP_WEB_DIST"); dir != "" {
		return os.DirFS(dir)
	}
	return webdist.FS()
}

// serveDashboardOrRedirect 服务 SPA 入口。根级真实文件（如 favicon.svg）直接返回；
// 其余未知路径回 index.html，交给前端路由处理（token 仍走 query ?t=）。
func (s *Server) serveDashboardOrRedirect(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		writeErr(w, http.StatusNotFound, "unknown api route: "+r.URL.Path)
		return
	}
	clean := path.Clean(strings.TrimPrefix(r.URL.Path, "/"))
	if clean != "." && clean != "" && !strings.Contains(clean, "..") {
		if f, err := dashboardFS().Open(clean); err == nil {
			defer func() { _ = f.Close() }()
			if st, sErr := f.Stat(); sErr == nil && !st.IsDir() {
				if ct := contentTypeByExt(clean); ct != "" {
					w.Header().Set("Content-Type", ct)
				}
				setDashboardNoCache(w)
				_, _ = io.Copy(w, f)
				return
			}
		}
	}
	serveIndex(w, r)
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
	f, err := dashboardFS().Open("index.html")
	if err != nil {
		http.Error(w, "dashboard 未构建：请先执行 cd web && npm run build", http.StatusInternalServerError)
		return
	}
	defer func() { _ = f.Close() }()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	setDashboardNoCache(w)
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, f)
}

// serveDashboardAsset 服务 /assets/* 下的静态构建产物。
func (s *Server) serveDashboardAsset(w http.ResponseWriter, r *http.Request) {
	clean := path.Clean(strings.TrimPrefix(r.URL.Path, "/"))
	if clean == "." || strings.Contains(clean, "..") {
		http.NotFound(w, r)
		return
	}
	f, err := dashboardFS().Open(clean)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer func() { _ = f.Close() }()

	if ct := contentTypeByExt(clean); ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	setDashboardNoCache(w)
	_, _ = io.Copy(w, f)
}

func setDashboardNoCache(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
}

func contentTypeByExt(name string) string {
	switch {
	case strings.HasSuffix(name, ".js"):
		return "text/javascript; charset=utf-8"
	case strings.HasSuffix(name, ".css"):
		return "text/css; charset=utf-8"
	case strings.HasSuffix(name, ".svg"):
		return "image/svg+xml"
	case strings.HasSuffix(name, ".json"):
		return "application/json"
	case strings.HasSuffix(name, ".woff2"):
		return "font/woff2"
	default:
		return ""
	}
}
