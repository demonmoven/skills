package server

import (
	"net/http"

	"code.byted.org/lihuanyu.0w0/gloop/internal/updatecheck"
)

// handleUpdateStatus GET /api/update-status —— 返回 gloop 当前版本 + npm registry 最新版本 + 状态。
//
// Query 参数：
//   - refresh=1：同步触发一次检查（仍受 24h 节流保护）。
//
// 返回 JSON（见 updatecheck.Status），免鉴权：这是纯信息接口，不泄漏任何用户数据。
func (s *Server) handleUpdateStatus(w http.ResponseWriter, r *http.Request) {
	if s.updater == nil {
		// 兜底：构造一个最小 Checker，只返回 current 版本，不发起任何请求。
		writeJSON(w, http.StatusOK, updatecheck.Status{Current: "unknown", State: "unknown"})
		return
	}
	s.updater.HandleHTTP(w, r)
}
