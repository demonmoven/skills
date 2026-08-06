package server

import (
	"encoding/json"
	"net/http"
)

// ==================== Notify API ====================

type notifyRequest struct {
	Title    string `json:"title"`
	Body     string `json:"body"`
	Priority string `json:"priority"` // normal / high
}

type notifyResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
	Channel string `json:"channel"`
	// Priority 暂不区分，后端 best-effort
}

// notifyUser 给当前用户发送飞书通知。
// POST /api/notify
func (s *Server) notifyUser(w http.ResponseWriter, r *http.Request) {
	var req notifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "请求体格式错误: "+err.Error())
		return
	}
	if req.Title == "" {
		writeErr(w, http.StatusBadRequest, "title 必填")
		return
	}

	ok, msg := s.engine.NotifyUser(r.Context(), req.Title, req.Body)
	writeJSON(w, http.StatusOK, notifyResponse{
		OK:      ok,
		Message: msg,
		Channel: "lark",
	})
}
