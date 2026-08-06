package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
)

const maxConfigPackageBodySize = 16 << 20 // 16 MB

func (s *Server) exportConfigPackage(w http.ResponseWriter, r *http.Request) {
	if s.engine == nil {
		writeErr(w, http.StatusInternalServerError, "engine 未初始化")
		return
	}
	pkg, err := s.engine.ExportConfigPackage()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	filename := "gloop_config_" + time.Now().Format("20060102_150405") + ".json"
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(pkg); err != nil {
		fmt.Fprintf(os.Stderr, "[warn] encode config package failed: %v\n", err)
	}
}

func (s *Server) previewConfigPackage(w http.ResponseWriter, r *http.Request) {
	if s.engine == nil {
		writeErr(w, http.StatusInternalServerError, "engine 未初始化")
		return
	}
	var pkg fsstore.ConfigPackage
	if err := readConfigPackageBody(r, &pkg); err != nil {
		writeErr(w, http.StatusBadRequest, "请求体解析失败: "+err.Error())
		return
	}
	preview, err := s.engine.PreviewConfigPackage(&pkg)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "preview": preview})
}

func (s *Server) importConfigPackage(w http.ResponseWriter, r *http.Request) {
	if s.engine == nil {
		writeErr(w, http.StatusInternalServerError, "engine 未初始化")
		return
	}
	var pkg fsstore.ConfigPackage
	if err := readConfigPackageBody(r, &pkg); err != nil {
		writeErr(w, http.StatusBadRequest, "请求体解析失败: "+err.Error())
		return
	}
	cfg, err := s.engine.ImportConfigPackage(&pkg)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	agents, _ := s.engine.ListAgentConfigs()
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":     true,
		"config": cfg,
		"agents": agents,
	})
}

func readConfigPackageBody(r *http.Request, dst any) error {
	lr := io.LimitReader(r.Body, maxConfigPackageBodySize+1)
	raw, err := io.ReadAll(lr)
	if err != nil {
		return err
	}
	if len(raw) == 0 {
		return fmt.Errorf("empty body")
	}
	if len(raw) > maxConfigPackageBodySize {
		return fmt.Errorf("配置包超过 %d MB", maxConfigPackageBodySize>>20)
	}
	return json.Unmarshal(raw, dst)
}
