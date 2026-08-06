package fsstore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPhaseSignalJSONString(t *testing.T) {
	sig := &PhaseSignal{
		OK:           true,
		Message:      "done",
		PhaseEnded:   true,
		PhaseVerdict: "pass",
		PhaseComment: "great work",
	}
	js := sig.JSONString()
	var parsed PhaseSignal
	if err := json.Unmarshal([]byte(js), &parsed); err != nil {
		t.Fatalf("JSONString did not produce valid JSON: %q err=%v", js, err)
	}
	if !parsed.OK || parsed.PhaseVerdict != "pass" || parsed.PhaseComment != "great work" {
		t.Errorf("roundtrip mismatch: %+v", parsed)
	}
}

func TestWriteReadRemovePhaseSignal(t *testing.T) {
	workDir := t.TempDir()
	sid := "sess_123"
	sig := PhaseSignal{
		OK:           true,
		PhaseEnded:   true,
		PhaseVerdict: "done",
		PhaseComment: "done",
	}
	if err := WritePhaseSignal(workDir, sid, sig); err != nil {
		t.Fatalf("WritePhaseSignal: %v", err)
	}
	got, err := ReadPhaseSignal(workDir, sid)
	if err != nil {
		t.Fatalf("ReadPhaseSignal: %v", err)
	}
	if !got.PhaseEnded || got.PhaseVerdict != "done" {
		t.Errorf("read back mismatch: %+v", got)
	}
	if got.CreatedAtMs == 0 {
		t.Error("WritePhaseSignal should fill CreatedAtMs when 0")
	}
	if err := RemovePhaseSignal(workDir, sid); err != nil {
		t.Errorf("RemovePhaseSignal: %v", err)
	}
	if _, err := ReadPhaseSignal(workDir, sid); err == nil {
		t.Error("after remove, read should fail")
	}
	// 再删一次应成功（文件不存在不报错）
	if err := RemovePhaseSignal(workDir, sid); err != nil {
		t.Errorf("RemovePhaseSignal on missing file should be noop: %v", err)
	}
}

func TestWriteReadPhaseSignal_EmptyArgs(t *testing.T) {
	// 空 workDir
	if err := WritePhaseSignal("", "sid", PhaseSignal{}); err == nil {
		t.Error("empty workDir should fail")
	}
	// 空 sid
	if err := WritePhaseSignal(t.TempDir(), "", PhaseSignal{}); err == nil {
		t.Error("empty sid should fail")
	}
	// 空参数读取直接返回 ErrNotExist
	if _, err := ReadPhaseSignal("", "sid"); !os.IsNotExist(err) {
		t.Errorf("empty workDir read should return NotExist, got %v", err)
	}
	if _, err := ReadPhaseSignal(t.TempDir(), ""); !os.IsNotExist(err) {
		t.Errorf("empty sid read should return NotExist, got %v", err)
	}
	// 空参数删除是 noop
	if err := RemovePhaseSignal("", "sid"); err != nil {
		t.Errorf("empty workDir remove should be noop, got %v", err)
	}
	if err := RemovePhaseSignal(t.TempDir(), ""); err != nil {
		t.Errorf("empty sid remove should be noop, got %v", err)
	}
}

func TestWriteAgentContext_And_DiscoverByWalk(t *testing.T) {
	base := t.TempDir()
	nested := filepath.Join(base, "a", "b", "c")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	// 将 context 写入 base 作为 quest 根
	ctx := AgentContext{
		DataDir:         base,
		QuestID:         "q_discover",
		SessionID:       "s1",
		Phase:           0,
		PhaseName:       "warrior",
		AdventurerID:    "adv",
		AdventurerClass: "warrior",
	}
	if err := WriteAgentContext(base, ctx); err != nil {
		t.Fatalf("WriteAgentContext: %v", err)
	}
	// 读回 context.json 验证 WriteAgentContext 回填了 WorkspacePath 和 CreatedAtMs
	written, err := ReadJSON[AgentContext](filepath.Join(base, AgentContextRel))
	if err != nil {
		t.Fatalf("read back context: %v", err)
	}
	if written.WorkspacePath != base {
		t.Errorf("WriteAgentContext should set WorkspacePath = base, got %s", written.WorkspacePath)
	}
	if written.CreatedAtMs == 0 {
		t.Error("WriteAgentContext should fill CreatedAtMs")
	}
	// 从 nested 目录向上发现
	got, workDir, err := DiscoverAgentContext(nested)
	if err != nil {
		t.Fatalf("DiscoverAgentContext: %v", err)
	}
	if workDir != base {
		t.Errorf("discovered workDir = %s, want %s", workDir, base)
	}
	if got.QuestID != "q_discover" || got.SessionID != "s1" {
		t.Errorf("discovered context wrong: %+v", got)
	}
}

func TestDiscoverAgentContext_NotFound(t *testing.T) {
	dir := t.TempDir()
	if _, _, err := DiscoverAgentContext(dir); err == nil {
		t.Error("DiscoverAgentContext on empty dir should fail")
	}
}

func TestDiscoverAgentContext_ByGLOOP_CONTEXT(t *testing.T) {
	base := t.TempDir()
	p := filepath.Join(base, AgentContextRel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	ctx := AgentContext{DataDir: base, QuestID: "q_by_env", SessionID: "s_env"}
	raw, _ := json.Marshal(ctx)
	if err := os.WriteFile(p, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GLOOP_CONTEXT", p)
	got, _, err := DiscoverAgentContext(t.TempDir()) // 任意 startDir，应直接走 env
	if err != nil {
		t.Fatalf("DiscoverAgentContext via env: %v", err)
	}
	if got.QuestID != "q_by_env" {
		t.Errorf("env ctx qid=%s, want q_by_env", got.QuestID)
	}
}

func TestAgentContextFromEnv(t *testing.T) {
	work := t.TempDir()
	t.Setenv("GLOOP_DATA_DIR", work)
	t.Setenv("GLOOP_QUEST_ID", "q_env_vars")
	t.Setenv("GLOOP_SESSION_ID", "s_env_vars")
	t.Setenv("GLOOP_WORKSPACE_PATH", work)
	t.Setenv("GLOOP_PHASE_INDEX", "3")
	t.Setenv("GLOOP_PHASE", "warrior")
	t.Setenv("GLOOP_ADVENTURER_ID", "adv1")
	t.Setenv("GLOOP_ADVENTURER_CLASS", "warrior")
	t.Setenv("GLOOP_SIGNAL_WORKSPACE_PATH", work+"/signal")

	ctx, wd, ok := agentContextFromEnv()
	if !ok {
		t.Fatalf("agentContextFromEnv should return ok=true with all env set")
	}
	if wd != work {
		t.Errorf("workDir=%s, want %s", wd, work)
	}
	if ctx.QuestID != "q_env_vars" || ctx.SessionID != "s_env_vars" {
		t.Errorf("ctx qid/sid wrong: %+v", ctx)
	}
	if ctx.Phase != 3 {
		t.Errorf("Phase=%d, want 3", ctx.Phase)
	}
	if ctx.SignalWorkspacePath != work+"/signal" {
		t.Errorf("SignalWorkspacePath = %s", ctx.SignalWorkspacePath)
	}

	// 关键 env 缺失时 ok=false
	os.Unsetenv("GLOOP_DATA_DIR")
	if _, _, ok := agentContextFromEnv(); ok {
		t.Error("missing GLOOP_DATA_DIR should result in ok=false")
	}
}

func TestAgentContextFromEnv_InvalidPhaseIndex(t *testing.T) {
	work := t.TempDir()
	t.Setenv("GLOOP_DATA_DIR", work)
	t.Setenv("GLOOP_QUEST_ID", "q")
	t.Setenv("GLOOP_SESSION_ID", "s")
	t.Setenv("GLOOP_WORKSPACE_PATH", work)
	t.Setenv("GLOOP_PHASE_INDEX", "not_a_number")
	ctx, _, ok := agentContextFromEnv()
	if !ok {
		t.Fatal("invalid phase index should still return ok (strconv.Atoi default 0), but got false")
	}
	if ctx.Phase != 0 {
		t.Errorf("Phase should default to 0 on invalid, got %d", ctx.Phase)
	}
}

// 验证 context.json 路径在 discover 时不是通过相对路径读到别的东西
func TestDiscoverAgentContext_StopsAtRoot(t *testing.T) {
	// 在一个没有 context.json 的临时目录下查询，最终应在到达根目录后停止
	dir := t.TempDir()
	_, _, err := DiscoverAgentContext(dir)
	if err == nil {
		t.Fatal("expected error when no context.json exists")
	}
	if !strings.Contains(err.Error(), "未找到") {
		t.Errorf("error message should mention '未找到': %v", err)
	}
}
