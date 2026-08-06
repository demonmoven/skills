package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

// ==================== phase / review / note API ====================
// 对应 agent syscall：phase checkpoint、review quest、note add
// 都走 engine 方法，保证事件流经 event bus。

// phaseCheckpoint 提交执行阶段结束信号。
// POST /api/quests/{id}/phases/{sid}/checkpoint
func (s *Server) phaseCheckpoint(w http.ResponseWriter, r *http.Request) {
	qid := r.PathValue("id")
	sid := r.PathValue("sid")
	if qid == "" || sid == "" {
		writeErr(w, http.StatusBadRequest, "缺少 qid 或 sid")
		return
	}

	var body struct {
		Summary      string           `json:"summary"`
		Status       string           `json:"status"`
		Deliverables []map[string]any `json:"deliverables"`
		Impact       string           `json:"impact"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "请求体解析失败: "+err.Error())
		return
	}
	if strings.TrimSpace(body.Summary) == "" {
		writeErr(w, http.StatusBadRequest, "summary 不能为空")
		return
	}

	if err := s.engine.PhaseCheckpoint(qid, sid, body.Summary, body.Status, body.Deliverables, body.Impact); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"message": "阶段信号已提交",
	})
}

// reviewQuest 提交评审阶段结束信号。
// POST /api/quests/{id}/reviews/{sid}
func (s *Server) reviewQuest(w http.ResponseWriter, r *http.Request) {
	qid := r.PathValue("id")
	sid := r.PathValue("sid")
	if qid == "" || sid == "" {
		writeErr(w, http.StatusBadRequest, "缺少 qid 或 sid")
		return
	}

	var body struct {
		Verdict string `json:"verdict"`
		Comment string `json:"comment"`
		Hints   string `json:"hints"`
		Score   int    `json:"score"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "请求体解析失败: "+err.Error())
		return
	}
	if body.Verdict == "" {
		writeErr(w, http.StatusBadRequest, "verdict 必填")
		return
	}
	if strings.TrimSpace(body.Comment) == "" {
		writeErr(w, http.StatusBadRequest, "comment 不能为空")
		return
	}

	if err := s.engine.ReviewQuest(qid, sid, body.Verdict, body.Comment, body.Hints, body.Score); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"message": "评审结论已提交",
		"verdict": body.Verdict,
	})
}

// addNote 追加笔记。
// POST /api/quests/{id}/notes
func (s *Server) addNote(w http.ResponseWriter, r *http.Request) {
	qid := r.PathValue("id")
	if qid == "" {
		writeErr(w, http.StatusBadRequest, "缺少 qid")
		return
	}

	var body struct {
		SessionID string `json:"session_id"`
		Content   string `json:"content"`
		Tag       string `json:"tag"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "请求体解析失败: "+err.Error())
		return
	}
	if strings.TrimSpace(body.Content) == "" {
		writeErr(w, http.StatusBadRequest, "content 不能为空")
		return
	}

	if err := s.engine.AppendNote(qid, body.SessionID, body.Content, body.Tag); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"message": "笔记已添加",
	})
}

// agentPost agent 发帖到 Feed（Feed 单源：agent 显式声明要发的帖）。
// POST /api/quests/{id}/posts
func (s *Server) agentPost(w http.ResponseWriter, r *http.Request) {
	qid := r.PathValue("id")
	if qid == "" {
		writeErr(w, http.StatusBadRequest, "缺少 qid")
		return
	}

	var body struct {
		SessionID string `json:"session_id"`
		Content   string `json:"content"`
		ReplyTo   string `json:"reply_to"`
		Kind      string `json:"kind"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "请求体解析失败: "+err.Error())
		return
	}
	if strings.TrimSpace(body.Content) == "" {
		writeErr(w, http.StatusBadRequest, "content 不能为空")
		return
	}

	postID, err := s.engine.PublishPost(qid, body.SessionID, body.Content, body.ReplyTo, body.Kind)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"post_id": postID,
		"message": "帖子已发布到 Feed",
	})
}

// questAsk 剑士向用户提问（阻塞等待）。
// POST /api/quests/{id}/ask
func (s *Server) questAsk(w http.ResponseWriter, r *http.Request) {
	qid := r.PathValue("id")
	sid := r.PathValue("sid")
	if qid == "" || sid == "" {
		writeErr(w, http.StatusBadRequest, "缺少 qid 或 sid")
		return
	}

	var body struct {
		Question string `json:"question"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "请求体解析失败: "+err.Error())
		return
	}
	if strings.TrimSpace(body.Question) == "" {
		writeErr(w, http.StatusBadRequest, "question 不能为空")
		return
	}
	if err := model.ValidateQuestionText(body.Question); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := s.engine.QuestAsk(qid, sid, body.Question); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"message": "问题已提交，等待用户回复",
	})
}
