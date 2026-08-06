package server

import (
	"fmt"
	"net/http"
	"os"
)

// ==================== workspace: diff / apply / discard ====================

func (s *Server) getQuestDiff(w http.ResponseWriter, r *http.Request) {
	qid := r.PathValue("id")
	if qid == "" {
		writeErr(w, http.StatusBadRequest, "missing quest id")
		return
	}
	diff, err := s.engine.ComputeDiff(r.Context(), qid)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":   true,
		"diff": diff,
	})
}

func (s *Server) listApplyBackups(w http.ResponseWriter, r *http.Request) {
	qid := r.PathValue("id")
	if qid == "" {
		writeErr(w, http.StatusBadRequest, "missing quest id")
		return
	}
	items, err := s.engine.ListApplyBackups(qid)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "items": items})
}

func (s *Server) getApplyBackup(w http.ResponseWriter, r *http.Request) {
	qid := r.PathValue("id")
	bid := r.PathValue("backup_id")
	if qid == "" || bid == "" {
		writeErr(w, http.StatusBadRequest, "missing quest id or backup id")
		return
	}
	item, err := s.engine.GetApplyBackup(qid, bid)
	if err != nil {
		writeErr(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "item": item})
}

type applyDiscardReq struct {
	Reason string `json:"reason"`
	Force  bool   `json:"force"`
}

func (s *Server) applyQuest(w http.ResponseWriter, r *http.Request) {
	qid := r.PathValue("id")
	if qid == "" {
		writeErr(w, http.StatusBadRequest, "missing quest id")
		return
	}
	var req applyDiscardReq
	force := false
	if err := readBodyLimited(r, &req); err == nil {
		force = req.Force
	}
	warnings, err := s.engine.ApplyQuest(r.Context(), qid, force)
	if err != nil {
		writeErrWith(w, http.StatusBadRequest, err.Error(), map[string]any{"warnings": warnings})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":       true,
		"qid":      qid,
		"warnings": warnings,
	})
}

func (s *Server) discardQuest(w http.ResponseWriter, r *http.Request) {
	qid := r.PathValue("id")
	if qid == "" {
		writeErr(w, http.StatusBadRequest, "missing quest id")
		return
	}
	var req applyDiscardReq
	if bodyErr := readBodyLimited(r, &req); bodyErr != nil {
		// body 解析失败，使用零值（可选字段），仅 warn
		fmt.Fprintf(os.Stderr, "[warn] 解析请求体失败: %v\n", bodyErr)
	}
	if err := s.engine.DiscardQuest(r.Context(), qid, req.Reason); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "qid": qid})
}

type spawnExecuteReq struct {
	AutoStart bool `json:"auto_start"`
}

func (s *Server) spawnExecuteQuest(w http.ResponseWriter, r *http.Request) {
	qid := r.PathValue("id")
	if qid == "" {
		writeErr(w, http.StatusBadRequest, "missing quest id")
		return
	}
	var req spawnExecuteReq
	if bodyErr := readBodyLimited(r, &req); bodyErr != nil {
		fmt.Fprintf(os.Stderr, "[warn] 解析请求体失败: %v\n", bodyErr)
	}
	q, err := s.engine.SpawnExecuteFromDesign(r.Context(), qid, req.AutoStart)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":       true,
		"qid":      q.ID,
		"short_id": q.ShortID,
		"quest":    q,
	})
}
