package server

import (
	"net/http"
	"strconv"
	"time"

	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
)

func (s *Server) getStats(w http.ResponseWriter, r *http.Request) {
	if s.root == nil {
		writeErr(w, http.StatusInternalServerError, "root 未初始化")
		return
	}
	limit := parseLimit(r, fsstore.DefaultPersonalStatsLimit)
	sinceMs := parseStatsSince(r)
	stats, err := fsstore.AggregatePersonalStats(s.root, sinceMs, limit)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "stats": stats})
}

func parseStatsSince(r *http.Request) int64 {
	if raw := r.URL.Query().Get("since_ms"); raw != "" {
		if n, err := strconv.ParseInt(raw, 10, 64); err == nil && n > 0 {
			return n
		}
	}
	now := fsstore.NowMs()
	switch r.URL.Query().Get("range") {
	case "today":
		t := fsstore.FromMs(now)
		y, m, d := t.Date()
		return time.Date(y, m, d, 0, 0, 0, 0, t.Location()).UnixMilli()
	case "week", "7d":
		return now - int64(7*24*time.Hour/time.Millisecond)
	case "30d", "month":
		return now - int64(30*24*time.Hour/time.Millisecond)
	default:
		return 0
	}
}
