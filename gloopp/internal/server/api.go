package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

const maxBodySize = 1 << 20 // 1 MB

// ==================== 小工具 ====================

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if encErr := json.NewEncoder(w).Encode(v); encErr != nil {
		fmt.Fprintf(os.Stderr, "[warn] encode JSON 输出失败: %v\n", encErr)
	}
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{
		"ok":    false,
		"error": msg,
	})
}

func writeErrWith(w http.ResponseWriter, status int, msg string, fields map[string]any) {
	resp := map[string]any{
		"ok":    false,
		"error": msg,
	}
	for k, v := range fields {
		resp[k] = v
	}
	writeJSON(w, status, resp)
}

func readBodyLimited(r *http.Request, dst any) error {
	lr := io.LimitReader(r.Body, maxBodySize)
	raw, err := io.ReadAll(lr)
	if err != nil {
		return err
	}
	if len(raw) == 0 {
		return fmt.Errorf("empty body")
	}
	return json.Unmarshal(raw, dst)
}
