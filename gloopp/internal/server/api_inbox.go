package server

import (
	"fmt"
	"net/http"
	"os"
	"sort"

	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
	"code.byted.org/lihuanyu.0w0/gloop/internal/orchestrator"
)

// ==================== inbox ====================

func (s *Server) listInbox(w http.ResponseWriter, r *http.Request) {
	items, err := s.engine.ListInbox()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if items == nil {
		items = []*fsstore.QuestMeta{}
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		if s.engine.QueuePosition(item.ID) > 0 {
			continue
		}
		if s.engine.IsQuestStarting(item.ID) {
			continue
		}
		row := s.questResponse(item)
		row["triage_mode"] = "candidate"
		row["candidate_source"] = item.CreatedBy
		out = append(out, row)
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "items": out})
}

func (s *Server) listHumanExceptions(w http.ResponseWriter, r *http.Request) {
	quests, err := s.engine.ListQuests()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	items := make([]*fsstore.HumanExceptionItem, 0)
	for _, q := range quests {
		item := fsstore.HumanExceptionFromQuest(q)
		if item != nil {
			items = append(items, item)
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Priority != items[j].Priority {
			return items[i].Priority < items[j].Priority
		}
		if items[i].RiskLevel != items[j].RiskLevel {
			return items[i].RiskLevel == "high"
		}
		return items[i].CreatedAtMs > items[j].CreatedAtMs
	})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "items": items})
}

func (s *Server) acceptInboxItem(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeErr(w, http.StatusBadRequest, "missing quest id")
		return
	}
	if err := s.engine.AcceptInboxItem(r.Context(), id); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	resp := map[string]any{"ok": true, "qid": id}
	if q, err := s.engine.GetQuest(id); err == nil {
		resp["quest"] = s.questResponse(q)
	}
	writeJSON(w, http.StatusOK, resp)
}

type inboxRejectReq struct {
	Reason string `json:"reason"`
}

type inboxUpdateReq struct {
	Query         *string `json:"query"`
	Type          *string `json:"type"`
	WorkDir       *string `json:"work_dir"`
	WorkingDir    *string `json:"working_dir"`
	WorkspaceMode *string `json:"workspace_mode"`
}

func (s *Server) updateInboxItem(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeErr(w, http.StatusBadRequest, "missing quest id")
		return
	}
	var req inboxUpdateReq
	if err := readBodyLimited(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "body 解析失败: "+err.Error())
		return
	}
	workDir := req.WorkDir
	if workDir == nil {
		workDir = req.WorkingDir
	}
	var workspaceMode *model.WorkspaceMode
	if req.WorkspaceMode != nil {
		v, err := parseWorkspaceMode(*req.WorkspaceMode)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		workspaceMode = &v
	}
	q, err := s.engine.UpdateInboxItem(id, orchestrator.InboxUpdate{
		Query:         req.Query,
		QuestType:     req.Type,
		WorkDir:       workDir,
		WorkspaceMode: workspaceMode,
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "quest": q})
}

func (s *Server) rejectInboxItem(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeErr(w, http.StatusBadRequest, "missing quest id")
		return
	}
	var req inboxRejectReq
	if bodyErr := readBodyLimited(r, &req); bodyErr != nil {
		// body 解析失败，使用零值（可选字段），仅 warn
		fmt.Fprintf(os.Stderr, "[warn] 解析请求体失败: %v\n", bodyErr)
	}
	if err := s.engine.RejectInboxItem(id, req.Reason); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "qid": id})
}
