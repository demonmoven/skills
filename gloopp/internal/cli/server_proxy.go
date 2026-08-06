package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"code.byted.org/lihuanyu.0w0/gloop/internal/auth"
	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
	"code.byted.org/lihuanyu.0w0/gloop/internal/version"
)

// serverProxy 封装对运行中 server 的 HTTP 调用。
// CLI mutation 命令优先通过 server API 执行，保证事件流经 server 的 event bus。
type serverProxy struct {
	baseURL string
	token   string
	client  *http.Client
}

// tryServerProxy 尝试连接运行中的 server。
// 返回 nil 表示 server 不可达，应 fallback 到直接引擎操作。
func tryServerProxy(dataDir string) *serverProxy {
	raw, err := os.ReadFile(filepath.Join(dataDir, version.TokenFileName))
	if err != nil {
		return nil
	}
	var t auth.TokenStore
	if err := json.Unmarshal(raw, &t); err != nil || t.Token == "" || t.Port <= 0 {
		return nil
	}

	host := t.BindHost
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	baseURL := fmt.Sprintf("http://%s:%d", host, t.Port)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(baseURL + "/healthz")
	if err != nil {
		return nil
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}

	return &serverProxy{
		baseURL: baseURL,
		token:   t.Token,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

// post 发起 POST 请求，返回响应 body 和错误。
func (p *serverProxy) post(path string, body any) (map[string]any, error) {
	return p.doJSON("POST", path, body)
}

// put 发起 PUT 请求，返回响应 body 和错误。
func (p *serverProxy) put(path string, body any) (map[string]any, error) {
	return p.doJSON("PUT", path, body)
}

// doJSON 发起 JSON 请求。
func (p *serverProxy) doJSON(method, path string, body any) (map[string]any, error) {
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, p.baseURL+path, &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]any
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}
	if resp.StatusCode >= 400 {
		errMsg := ""
		if e, ok := result["error"].(string); ok {
			errMsg = e
		}
		return result, fmt.Errorf("server: %s", errMsg)
	}
	return result, nil
}

// resolveUserReview 通过 server API 提交评审。
func (p *serverProxy) resolveUserReview(qid, verdict, comment string, apply bool) error {
	_, err := p.post(fmt.Sprintf("/api/quests/%s/resolve-user-review", qid), map[string]any{
		"verdict": verdict,
		"comment": comment,
		"apply":   apply,
	})
	return err
}

// cancelQuest 通过 server API 取消委托。
func (p *serverProxy) cancelQuest(qid, reason string) error {
	_, err := p.post(fmt.Sprintf("/api/quests/%s/cancel", qid), map[string]string{
		"reason": reason,
	})
	return err
}

// discardQuest 通过 server API 丢弃改动。
func (p *serverProxy) discardQuest(qid, reason string) error {
	_, err := p.post(fmt.Sprintf("/api/quests/%s/discard", qid), map[string]string{
		"reason": reason,
	})
	return err
}

// addComment 通过 server API 追加评论。
func (p *serverProxy) addComment(qid, comment string) error {
	_, err := p.post(fmt.Sprintf("/api/quests/%s/comment", qid), map[string]string{
		"comment": comment,
	})
	return err
}

func (p *serverProxy) addAnswer(qid, questionID, answer string) error {
	_, err := p.post(fmt.Sprintf("/api/quests/%s/answer", qid), map[string]string{
		"question_id": questionID,
		"answer":      answer,
		"source":      "cli",
	})
	return err
}

// resolveBlocked 通过 server API 处理 blocked 委托。
func (p *serverProxy) resolveBlocked(qid, action, comment string) error {
	_, err := p.post(fmt.Sprintf("/api/quests/%s/resolve-blocked", qid), map[string]string{
		"action":  action,
		"comment": comment,
	})
	return err
}

// spawnQuest 通过 server API 扇出独立委托。
func (p *serverProxy) spawnQuest(query, groupID, workDir, parentQuestID string, contract fsstore.FanoutContract) (string, string, error) {
	result, err := p.post("/api/quests/spawn", map[string]any{
		"query":               query,
		"group_id":            groupID,
		"work_dir":            workDir,
		"parent_quest_id":     parentQuestID,
		"leaf_id":             contract.LeafID,
		"ownership_scopes":    contract.OwnershipScopes,
		"merge_strategy":      contract.MergeStrategy,
		"merge_owner_leaf_id": contract.MergeOwnerLeafID,
	})
	if err != nil {
		return "", "", err
	}
	qid, _ := result["qid"].(string)
	gid, _ := result["group_id"].(string)
	return qid, gid, nil
}

// writeContextDim 通过 server API 写入上下文维度。
func (p *serverProxy) writeContextDim(dim, content string) error {
	_, err := p.put("/api/context/dims/"+dim, map[string]string{
		"content": content,
	})
	return err
}

// writeContextSummary 通过 server API 写入总览摘要。
func (p *serverProxy) writeContextSummary(summary string) error {
	_, err := p.put("/api/context/summary", map[string]string{
		"summary": summary,
	})
	return err
}

// phaseCheckpoint 通过 server API 提交阶段结束信号。
func (p *serverProxy) phaseCheckpoint(qid, sid, summary, status string, deliverables []map[string]any, impact string) error {
	body := map[string]any{
		"summary":      summary,
		"status":       status,
		"deliverables": deliverables,
	}
	if strings.TrimSpace(impact) != "" {
		body["impact"] = impact
	}
	_, err := p.post(fmt.Sprintf("/api/quests/%s/phases/%s/checkpoint", qid, sid), body)
	return err
}

// reviewQuest 通过 server API 提交评审结论。
func (p *serverProxy) reviewQuest(qid, sid, verdict, comment, hints string, score int) error {
	_, err := p.post(fmt.Sprintf("/api/quests/%s/reviews/%s", qid, sid), map[string]any{
		"verdict": verdict,
		"comment": comment,
		"hints":   hints,
		"score":   score,
	})
	return err
}

func (p *serverProxy) addNote(qid, sid, content, tag string) error {
	_, err := p.post(fmt.Sprintf("/api/quests/%s/notes", qid), map[string]string{
		"session_id": sid,
		"content":    content,
		"tag":        tag,
	})
	return err
}

// agentPost 通过 server API 发帖到 Feed。
func (p *serverProxy) agentPost(qid, sid, content, replyTo, kind string) (string, error) {
	body := map[string]any{
		"session_id": sid,
		"content":    content,
	}
	if strings.TrimSpace(replyTo) != "" {
		body["reply_to"] = replyTo
	}
	if strings.TrimSpace(kind) != "" {
		body["kind"] = kind
	}
	res, err := p.post(fmt.Sprintf("/api/quests/%s/posts", qid), body)
	if err != nil {
		return "", err
	}
	if postID, ok := res["post_id"].(string); ok {
		return postID, nil
	}
	return "", nil
}

// questAsk 通过 server API 提交用户提问（阻塞等待）。
func (p *serverProxy) questAsk(qid, sid, question string) error {
	_, err := p.post(fmt.Sprintf("/api/quests/%s/ask/%s", qid, sid), map[string]string{
		"question": question,
	})
	return err
}

// resolveDataDirOrDefault 解析 --data-dir 参数，为空时返回默认路径。
func resolveDataDirOrDefault(flagDir string) string {
	dir := flagDir
	if d, err := resolveDataDir(dir); err == nil {
		return d
	}
	if dir == "" {
		d, _ := fsstore.DefaultPath()
		return d
	}
	return dir
}
