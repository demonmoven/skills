package server

import (
	"net/http"

	"code.byted.org/lihuanyu.0w0/gloop/internal/orchestrator"
	"code.byted.org/lihuanyu.0w0/gloop/internal/skills"
)

type skillDetail struct {
	skills.Manifest
	Source    string `json:"source"`
	Body      string `json:"body"`
	Formatted string `json:"formatted"`
}

func (s *Server) listSkills(w http.ResponseWriter, r *http.Request) {
	if s.engine == nil {
		// engine 未初始化时退回到内置技能表（保持历史测试兼容）
		items := skills.Default().ListManifests()
		withSrc := make([]map[string]any, 0, len(items))
		for _, m := range items {
			withSrc = append(withSrc, map[string]any{
				"name":              m.Name,
				"version":           m.Version,
				"description":       m.Description,
				"category":          m.Category,
				"requires_bins":     m.RequiresBins,
				"requires_cli_help": m.RequiresCliHelp,
				"warrior_available": m.WarriorAvailable,
				"mage_available":    m.MageAvailable,
				"source":            "builtin",
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "items": withSrc})
		return
	}
	items, err := s.engine.ListSkillsManifests()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if items == nil {
		items = []orchestrator.SkillManifestExt{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "items": items})
}

func (s *Server) getSkill(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeErr(w, http.StatusBadRequest, "missing skill name")
		return
	}
	if s.engine == nil {
		sk, err := skills.Default().Get(name)
		if err != nil {
			writeErr(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"ok": true,
			"item": skillDetail{
				Manifest:  sk.Manifest(),
				Source:    "builtin",
				Body:      sk.Body,
				Formatted: sk.FormatMarkdown(),
			},
		})
		return
	}
	ext, body, formatted, err := s.engine.GetSkillDetail(name)
	if err != nil {
		writeErr(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok": true,
		"item": skillDetail{
			Manifest:  ext.Manifest,
			Source:    ext.Source,
			Body:      body,
			Formatted: formatted,
		},
	})
}
