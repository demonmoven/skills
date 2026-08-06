package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
	"code.byted.org/lihuanyu.0w0/gloop/internal/version"
)

// ==================== SSE Stream ====================

func (s *Server) sseStream(w http.ResponseWriter, r *http.Request) {
	if s.bus == nil {
		writeErr(w, http.StatusInternalServerError, "event bus 未初始化")
		return
	}
	qid := r.URL.Query().Get("qid")
	if pathID := r.PathValue("id"); pathID != "" {
		qid = pathID
	}
	level := r.URL.Query().Get("level") // "semantic" filters out micro.* events

	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeErr(w, http.StatusInternalServerError, "server does not support streaming")
		return
	}

	ch, unsubscribe := s.bus.Subscribe()
	defer unsubscribe()

	// 先发一条 hello 事件
	hello := map[string]any{
		"type":      "hello",
		"app":       version.AppName,
		"qid":       qid,
		"mascot":    version.Mascot,
		"timestamp": fsstore.NowMs(),
	}
	if bs, err := json.Marshal(hello); err == nil {
		fmt.Fprintf(w, "data: %s\n\n", bs)
		flusher.Flush()
	}

	ctx := r.Context()
	streamDone := s.streamDone()
	for {
		select {
		case <-streamDone:
			return
		case <-ctx.Done():
			return
		case ev, ok := <-ch:
			if !ok {
				return
			}
			// 如果客户端指定了 qid，就只发匹配该 quest 的事件。
			// 空 QuestID 是系统级事件，不应混入详情页专属流。
			if qid != "" && ev.QuestID != qid {
				continue
			}
			if level == "semantic" && strings.HasPrefix(string(ev.Type), "micro.") {
				continue
			}
			bs, err := json.Marshal(ev)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", bs)
			flusher.Flush()
		}
	}
}

func (s *Server) streamDone() <-chan struct{} {
	if s.streams == nil {
		return nil
	}
	return s.streams.Done()
}
