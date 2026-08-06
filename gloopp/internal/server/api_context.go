package server

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ==================== context API ====================

func (s *Server) getContextMeta(w http.ResponseWriter, r *http.Request) {
	meta, err := s.engine.GetContextMeta()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "meta": meta})
}

func (s *Server) listContextDims(w http.ResponseWriter, r *http.Request) {
	dims, err := s.engine.ListContextDims()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":    true,
		"items": dims,
		"total": len(dims),
	})
}

func (s *Server) getContextDim(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeErr(w, http.StatusBadRequest, "缺少维度名")
		return
	}
	dim, err := s.engine.ReadContextDim(name)
	if err != nil {
		writeErr(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":  true,
		"dim": dim,
	})
}

func (s *Server) getContextSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := s.engine.ReadContextSummary()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"summary": summary,
	})
}

func (s *Server) refreshContext(w http.ResponseWriter, r *http.Request) {
	qid, err := s.engine.RunAutomation(r.Context(), "auto_context_refresh")
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "触发上下文刷新失败: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":  true,
		"qid": qid,
	})
}

// writeContextSummary 写入总览摘要。
// PUT /api/context/summary
func (s *Server) writeContextSummary(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Summary string `json:"summary"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "请求体解析失败: "+err.Error())
		return
	}
	if strings.TrimSpace(body.Summary) == "" {
		writeErr(w, http.StatusBadRequest, "summary 不能为空")
		return
	}

	if err := s.engine.WriteContextSummary(body.Summary); err != nil {
		writeErr(w, http.StatusInternalServerError, "写入摘要失败: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"message": "摘要已写入",
	})
}

func (s *Server) previewKnowledgeExport(w http.ResponseWriter, r *http.Request) {
	preview, last, err := s.engine.KnowledgeExportPreview()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "生成 knowledge 预览失败: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"preview": preview,
		"last":    last,
	})
}

// ==================== context import / write ====================

func (s *Server) writeContextDim(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeErr(w, http.StatusBadRequest, "缺少维度名")
		return
	}

	var body struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "请求体解析失败: "+err.Error())
		return
	}
	if strings.TrimSpace(body.Content) == "" {
		writeErr(w, http.StatusBadRequest, "内容不能为空")
		return
	}

	dim, err := s.engine.WriteContextDim(name, body.Content)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "写入维度失败: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":  true,
		"dim": dim,
	})
}

// ==================== context export ====================

func (s *Server) exportContextDim(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeErr(w, http.StatusBadRequest, "缺少维度名")
		return
	}
	dim, err := s.engine.ReadContextDim(name)
	if err != nil {
		writeErr(w, http.StatusNotFound, err.Error())
		return
	}

	filename := "context_" + name + ".md"
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(dim.Body)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(dim.Body))
}

func (s *Server) exportAllContext(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("format") == "okf" {
		s.exportKnowledgeBundle(w, r)
		return
	}
	dims, err := s.engine.ListContextDims()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	summary, _ := s.engine.ReadContextSummary()

	var sb strings.Builder
	sb.WriteString("# Gloop 用户上下文导出\n\n")
	sb.WriteString("导出时间：" + time.Now().Format("2006-01-02 15:04:05") + "\n\n")
	sb.WriteString("---\n\n")

	// 总览摘要
	sb.WriteString("## 总览摘要\n\n")
	if summary != "" {
		sb.WriteString(summary + "\n\n")
	} else {
		sb.WriteString("_暂无总览摘要_\n\n")
	}
	sb.WriteString("---\n\n")

	// 各维度
	for i, d := range dims {
		dim, readErr := s.engine.ReadContextDim(d.Name)
		if readErr != nil {
			continue
		}
		if i > 0 {
			sb.WriteString("---\n\n")
		}
		sb.WriteString("## 维度：" + dim.Title + " (" + dim.Name + ")\n\n")
		if dim.Description != "" {
			sb.WriteString("_" + dim.Description + "_\n\n")
		}
		updated := time.UnixMilli(dim.UpdatedAtMs).Format("2006-01-02 15:04:05")
		sb.WriteString("更新时间：" + updated + " | 大小：" + fmt.Sprintf("%d", dim.SizeBytes) + " 字节\n\n")
		sb.WriteString(dim.Body + "\n\n")
	}

	filename := "gloop_context_export_" + time.Now().Format("20060102_150405") + ".md"
	content := sb.String()
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(content))
}

func (s *Server) exportKnowledgeBundle(w http.ResponseWriter, _ *http.Request) {
	bundle, err := s.engine.BuildKnowledgeExport()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, f := range bundle.Files {
		h := &zip.FileHeader{
			Name:   f.Path,
			Method: zip.Deflate,
		}
		h.SetModTime(time.Now())
		wr, createErr := zw.CreateHeader(h)
		if createErr != nil {
			_ = zw.Close()
			writeErr(w, http.StatusInternalServerError, "创建 knowledge zip 失败: "+createErr.Error())
			return
		}
		if _, writeFileErr := wr.Write([]byte(f.Content)); writeFileErr != nil {
			_ = zw.Close()
			writeErr(w, http.StatusInternalServerError, "写入 knowledge zip 失败: "+writeFileErr.Error())
			return
		}
	}
	if err := zw.Close(); err != nil {
		writeErr(w, http.StatusInternalServerError, "关闭 knowledge zip 失败: "+err.Error())
		return
	}
	filename := "gloop_knowledge_bundle_" + time.Now().Format("20060102_150405") + ".zip"
	_ = s.engine.WriteKnowledgeExportMeta(filename, bundle)
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", buf.Len()))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf.Bytes())
}
