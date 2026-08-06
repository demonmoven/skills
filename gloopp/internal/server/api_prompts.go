package server

import (
	"net/http"

	"code.byted.org/lihuanyu.0w0/gloop/internal/prompt"
)

// promptDetail 是单条模板详情（show 接口用）。
type promptDetail struct {
	Name         string `json:"name"`
	Source       string `json:"source"`
	SizeBytes    int    `json:"size_bytes"`
	Content      string `json:"content"`
	IsOverridden bool   `json:"is_overridden"`
}

// listPrompts 返回所有 prompt 模板列表。
// 不需要 engine，直接用内建模板 + 用户目录（如果 engine 存在则用 engine 的 store）。
func (s *Server) listPrompts(w http.ResponseWriter, r *http.Request) {
	store := s.promptStore()
	list, err := store.ListTemplates()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "列出模板失败: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":    true,
		"items": list,
		"total": len(list),
	})
}

// getPrompt 返回单个模板详情。
func (s *Server) getPrompt(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeErr(w, http.StatusBadRequest, "missing template name")
		return
	}
	store := s.promptStore()
	content, info, err := store.GetTemplate(name)
	if err != nil {
		writeErr(w, http.StatusNotFound, "模板不存在: "+name)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok": true,
		"item": promptDetail{
			Name:         info.Name,
			Source:       info.Source,
			SizeBytes:    info.SizeBytes,
			Content:      content,
			IsOverridden: info.IsOverridden,
		},
	})
}

// promptStore 返回当前使用的模板存储。
// 如果 engine 存在则用 engine 的（带用户目录 overlay），否则退回内建默认。
func (s *Server) promptStore() *prompt.TemplateStore {
	if s.engine != nil {
		if ts := s.engine.PromptTemplates(); ts != nil {
			return ts
		}
	}
	return prompt.DefaultTemplateStore()
}
