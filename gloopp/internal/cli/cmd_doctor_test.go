package cli

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

func TestRunDoctorAgentsJSON(t *testing.T) {
	dataDir := t.TempDir()
	root, err := fsstore.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	binDir := t.TempDir()
	relay := writeDoctorFixtureBin(t, binDir, "relay", "relay 1.2.3")
	missing := filepath.Join(binDir, "missing-agent")
	agents := []fsstore.AgentConfig{
		{Name: "relay", Type: model.AgentTypeCLI, Command: relay, Args: []string{"-p"}, Enabled: true},
		{Name: "missing", Type: model.AgentTypeCLI, Command: missing, Enabled: true},
		{Name: "off", Type: model.AgentTypeACP, Command: relay, Enabled: false},
	}
	for i := range agents {
		if err := root.SaveAgent(&agents[i]); err != nil {
			t.Fatal(err)
		}
	}

	stdout, stderr, code := captureStdIO(t, func() int {
		return runDoctorAgents(context.Background(), testLogger(), []string{"--data-dir", dataDir, "--json"})
	})
	if code != 1 {
		t.Fatalf("exit code = %d, want 1 for missing enabled agent; stderr=%s stdout=%s", code, stderr, stdout)
	}
	var report doctorAgentsReport
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("decode report: %v\n%s", err, stdout)
	}
	if report.Summary.Total != 3 || report.Summary.OK != 1 || report.Summary.Error != 1 || report.Summary.Disabled != 1 {
		t.Fatalf("summary = %+v", report.Summary)
	}
	relayResult := findDoctorAgent(t, report, "relay")
	if relayResult.Status != "ok" || relayResult.Adapter != "relay-cli" || !strings.Contains(relayResult.Version, "relay 1.2.3") {
		t.Fatalf("relay result = %+v", relayResult)
	}
	missingResult := findDoctorAgent(t, report, "missing")
	if missingResult.Status != "error" || len(missingResult.Issues) == 0 {
		t.Fatalf("missing result = %+v", missingResult)
	}
}

func TestRunDoctorAgentsTextExitsZeroForDisabledUnsupported(t *testing.T) {
	dataDir := t.TempDir()
	root, err := fsstore.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	agent := &fsstore.AgentConfig{
		Name:    "traex_cli_disabled",
		Type:    model.AgentTypeCLI,
		Command: "traex",
		Enabled: false,
	}
	if err := root.SaveAgent(agent); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, code := captureStdIO(t, func() int {
		return runDoctorAgents(context.Background(), testLogger(), []string{"--data-dir", dataDir})
	})
	if code != 0 {
		t.Fatalf("exit code = %d, stderr=%s stdout=%s", code, stderr, stdout)
	}
	if !strings.Contains(stdout, "traex_cli_disabled") || !strings.Contains(stdout, "disabled") {
		t.Fatalf("stdout missing disabled agent:\n%s", stdout)
	}
}

func TestDoctorTraeXCLIAdapterAndSmokeArgs(t *testing.T) {
	agent := &fsstore.AgentConfig{
		Name:         "traex",
		Type:         model.AgentTypeCLI,
		Command:      "traex",
		DefaultModel: "auto",
		Enabled:      true,
	}
	if got := detectAgentAdapter(agent); got != "traex-cli" {
		t.Fatalf("adapter = %q, want traex-cli", got)
	}
	if issue := adapterIssue(agent); issue != "" {
		t.Fatalf("adapter issue = %q, want empty", issue)
	}
	caps := staticAgentCapabilities(agent)
	if !containsDoctorString(caps, "stateful-resume") || !containsDoctorString(caps, "experimental") {
		t.Fatalf("capabilities = %#v, want stateful-resume and experimental", caps)
	}

	args, stdin, ok := smokeCommand("traex-cli", agent)
	if !ok {
		t.Fatal("traex-cli smoke should be supported")
	}
	joined := strings.Join(args, "\x00")
	for _, forbidden := range []string{"--model\x00auto", "--permission-mode", "bypass_permissions"} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("traex smoke args contain forbidden %q: %#v", forbidden, args)
		}
	}
	for _, want := range []string{"exec", "--ephemeral", "--json", "--sandbox", "read-only"} {
		if !containsDoctorString(args, want) {
			t.Fatalf("traex smoke args missing %q: %#v", want, args)
		}
	}
	if !strings.Contains(stdin, "GLOOP_SMOKE_OK") {
		t.Fatalf("stdin = %q, want smoke prompt", stdin)
	}
}

func containsDoctorString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func TestRunDoctorAgentsSmokeJSON(t *testing.T) {
	dataDir := t.TempDir()
	root, err := fsstore.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	binDir := t.TempDir()
	relay := writeDoctorFixtureBin(t, binDir, "relay", "relay 1.2.3")
	if err := root.SaveAgent(&fsstore.AgentConfig{
		Name:    "relay",
		Type:    model.AgentTypeCLI,
		Command: relay,
		Enabled: true,
	}); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, code := captureStdIO(t, func() int {
		return runDoctorAgents(context.Background(), testLogger(), []string{
			"--data-dir", dataDir,
			"--json",
			"--smoke",
			"--smoke-timeout", time.Second.String(),
		})
	})
	if code != 0 {
		t.Fatalf("exit code = %d, stderr=%s stdout=%s", code, stderr, stdout)
	}
	var report doctorAgentsReport
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("decode report: %v\n%s", err, stdout)
	}
	relayResult := findDoctorAgent(t, report, "relay")
	if relayResult.Smoke == nil || relayResult.Smoke.Status != "ok" {
		t.Fatalf("relay smoke = %+v", relayResult.Smoke)
	}
	if !strings.Contains(relayResult.Smoke.Output, "GLOOP_SMOKE_OK") {
		t.Fatalf("smoke output = %q", relayResult.Smoke.Output)
	}
}

func TestRunDoctorAgentsSmokeACPJSON(t *testing.T) {
	dataDir := t.TempDir()
	root, err := fsstore.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	binDir := t.TempDir()
	acpBin := writeDoctorACPFixtureBin(t, binDir, "mock-acp", false)
	if err := root.SaveAgent(&fsstore.AgentConfig{
		Name:    "codex",
		Type:    model.AgentTypeACP,
		Command: acpBin,
		Args:    []string{"serve"},
		Enabled: true,
	}); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, code := captureStdIO(t, func() int {
		return runDoctorAgents(context.Background(), testLogger(), []string{
			"--data-dir", dataDir,
			"--json",
			"--smoke",
			"--smoke-timeout", time.Second.String(),
		})
	})
	if code != 0 {
		t.Fatalf("exit code = %d, stderr=%s stdout=%s", code, stderr, stdout)
	}
	var report doctorAgentsReport
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("decode report: %v\n%s", err, stdout)
	}
	codexResult := findDoctorAgent(t, report, "codex")
	if codexResult.Smoke == nil || codexResult.Smoke.Status != "ok" {
		t.Fatalf("codex smoke = %+v", codexResult.Smoke)
	}
	if !strings.Contains(codexResult.Smoke.Output, "GLOOP_SMOKE_OK") {
		t.Fatalf("smoke output = %q", codexResult.Smoke.Output)
	}
}

func TestRunDoctorAgentsSmokeFailure(t *testing.T) {
	dataDir := t.TempDir()
	root, err := fsstore.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	binDir := t.TempDir()
	relay := writeDoctorFixtureBin(t, binDir, "relay", "relay 1.2.3")
	if err := root.SaveAgent(&fsstore.AgentConfig{
		Name:         "relay",
		Type:         model.AgentTypeCLI,
		Command:      relay,
		DefaultModel: "--fail-smoke",
		Enabled:      true,
	}); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, code := captureStdIO(t, func() int {
		return runDoctorAgents(context.Background(), testLogger(), []string{
			"--data-dir", dataDir,
			"--json",
			"--smoke",
			"--smoke-timeout", time.Second.String(),
		})
	})
	if code != 1 {
		t.Fatalf("exit code = %d, want 1; stderr=%s stdout=%s", code, stderr, stdout)
	}
	var report doctorAgentsReport
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("decode report: %v\n%s", err, stdout)
	}
	relayResult := findDoctorAgent(t, report, "relay")
	if relayResult.Smoke == nil || relayResult.Smoke.Status != "error" {
		t.Fatalf("relay smoke = %+v", relayResult.Smoke)
	}
	if relayResult.Smoke.FailureKind == "" || relayResult.Smoke.Hint == "" {
		t.Fatalf("relay smoke classification missing: %+v", relayResult.Smoke)
	}
}

func TestClassifyDoctorSmokeError(t *testing.T) {
	cases := []struct {
		name string
		msg  string
		kind string
		hint string
	}{
		{
			name: "auth",
			msg:  "JSON-RPC error -32000: Authentication required",
			kind: "auth_required",
			hint: "登录",
		},
		{
			name: "model permission",
			msg:  "API Error: Access denied for model (ark/seed-code-0611). The gateway rejected this resolved model",
			kind: "model_access_denied",
			hint: "模型",
		},
		{
			name: "anthropic token",
			msg:  "Anthropic authentication is not supported in this build. Unset ANTHROPIC_AUTH_TOKEN",
			kind: "auth_env_conflict",
			hint: "ANTHROPIC_AUTH_TOKEN",
		},
		{
			name: "timeout",
			msg:  "ACP smoke 超时",
			kind: "timeout",
			hint: "超时",
		},
		{
			name: "runtime fs",
			msg:  "Error: ENOENT: no such file or directory, mkdir '/home/user/.aiden/log'",
			kind: "runtime_filesystem_error",
			hint: "目录",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			kind, hint := classifyDoctorSmokeError(tc.msg)
			if kind != tc.kind {
				t.Fatalf("kind = %q, want %q", kind, tc.kind)
			}
			if !strings.Contains(hint, tc.hint) {
				t.Fatalf("hint = %q, want contains %q", hint, tc.hint)
			}
		})
	}
}

func TestDoctorSmokeResultClassify(t *testing.T) {
	res := doctorSmokeResult{Status: "error", Error: "API Error: Access denied for model alwaysday1"}
	res.classify()
	if res.FailureKind != "model_access_denied" || !strings.Contains(res.Hint, "模型") {
		t.Fatalf("classified result = %+v", res)
	}
}

func TestRunDoctorHelpExitsZero(t *testing.T) {
	_, stderr, code := captureStdIO(t, func() int {
		return runDoctorAgents(context.Background(), testLogger(), []string{"--help"})
	})
	if code != 0 {
		t.Fatalf("doctor agents --help exit code = %d, stderr=%s", code, stderr)
	}
	_, stderr, code = captureStdIO(t, func() int {
		return runDoctorE2E(context.Background(), testLogger(), []string{"--help"})
	})
	if code != 0 {
		t.Fatalf("doctor e2e --help exit code = %d, stderr=%s", code, stderr)
	}
}

func writeDoctorFixtureBin(t *testing.T, dir, name, version string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	content := "#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then echo '" + version + "'; exit 0; fi\ncase \" $* \" in *\" --fail-smoke \"*) echo smoke failed >&2; exit 42;; esac\necho GLOOP_SMOKE_OK\nexit 0\n"
	mode := os.FileMode(0o755)
	if runtime.GOOS == "windows" {
		content = "@echo off\necho " + version + "\n"
		mode = 0o755
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeDoctorACPFixtureBin(t *testing.T, dir, name string, failPrompt bool) string {
	t.Helper()
	path := filepath.Join(dir, name)
	promptCase := `echo '{"jsonrpc":"2.0","method":"session/update","params":{"sessionId":"sess_doctor","update":{"sessionUpdate":"agent_message_chunk","content":{"type":"text","text":"GLOOP_SMOKE_OK"}}}}'
echo '{"jsonrpc":"2.0","result":{"stopReason":"end_turn"},"id":3}'`
	if failPrompt {
		promptCase = `echo 'fixture stderr prompt failure' >&2
echo '{"jsonrpc":"2.0","error":{"code":-32603,"message":"Internal error","data":{"reason":"boom"}},"id":3}'`
	}
	content := `#!/bin/sh
if [ "$1" = "--version" ]; then echo 'mock-acp 1.0.0'; exit 0; fi
while IFS= read -r line; do
  case "$line" in
    *'"method":"initialize"'*)
      echo '{"jsonrpc":"2.0","result":{"protocolVersion":1,"agentCapabilities":{"promptCapabilities":{"image":false}},"agentInfo":{"name":"mock-acp","version":"1.0.0"}},"id":1}'
      ;;
    *'"method":"session/new"'*)
      echo '{"jsonrpc":"2.0","result":{"sessionId":"sess_doctor","modes":{"currentModeId":"default","availableModes":[]},"models":{"currentModelId":"test-model","availableModels":[]}},"id":2}'
      ;;
    *'"method":"session/prompt"'*)
      ` + promptCase + `
      ;;
  esac
done
`
	mode := os.FileMode(0o755)
	if runtime.GOOS == "windows" {
		t.Skip("ACP shell fixture is Unix-only")
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeDoctorACPContractFixtureBin(t *testing.T, dir, name string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("ACP shell fixture is Unix-only")
	}
	path := filepath.Join(dir, name)
	content := `#!/bin/sh
if [ "$1" = "--version" ]; then echo 'contract-acp 1.0.0'; exit 0; fi
write_signal() {
  meta="$GLOOP_DATA_DIR/workspace/quests/$GLOOP_QUEST_ID/meta.json"
  real_workdir="$(sed -n 's/.*"workspace_path"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$meta" | head -1)"
  if [ -z "$real_workdir" ]; then real_workdir="$GLOOP_WORKSPACE_PATH"; fi
  mkdir -p "$real_workdir/.gloop/signals"
  cat > "$real_workdir/.gloop/signals/$GLOOP_SESSION_ID.json" <<EOF
$1
EOF
}
while IFS= read -r line; do
  case "$line" in
    *'"method":"initialize"'*)
      echo '{"jsonrpc":"2.0","result":{"protocolVersion":1,"agentCapabilities":{"promptCapabilities":{"image":false}},"agentInfo":{"name":"contract-acp","version":"1.0.0"}},"id":1}'
      ;;
    *'"method":"session/new"'*)
      echo '{"jsonrpc":"2.0","result":{"sessionId":"sess_contract","modes":{"currentModeId":"default","availableModes":[]},"models":{"currentModelId":"test-model","availableModels":[]}},"id":2}'
      ;;
    *'"method":"session/prompt"'*)
      case "$GLOOP_ADVENTURER_CLASS" in
        mage)
          write_signal '{"ok":true,"message":"评审结论已提交","phase_ended":true,"phase_verdict":"pass","phase_comment":"contract ok","phase_score":9,"data":{"verdict":"pass","comment":"contract ok","score":9,"source":"contract_fixture"}}'
          ;;
        *)
          write_signal '{"ok":true,"message":"阶段结论已提交","phase_ended":true,"phase_verdict":"done","phase_comment":"GLOOP_REAL_AGENT_CONTRACT_OK","data":{"status":"done","summary":"GLOOP_REAL_AGENT_CONTRACT_OK","source":"contract_fixture"}}'
          ;;
      esac
      echo '{"jsonrpc":"2.0","method":"session/update","params":{"sessionId":"sess_contract","update":{"sessionUpdate":"agent_message_chunk","content":{"type":"text","text":"GLOOP_REAL_AGENT_CONTRACT_OK contract signal attempted"}}}}'
      echo '{"jsonrpc":"2.0","result":{"stopReason":"end_turn","usage":{"inputTokens":5,"outputTokens":3}},"id":3}'
      ;;
  esac
done
`
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func findDoctorAgent(t *testing.T, report doctorAgentsReport, name string) doctorAgentResult {
	t.Helper()
	for _, agent := range report.Agents {
		if agent.Name == name {
			return agent
		}
	}
	t.Fatalf("agent %q not found in %+v", name, report.Agents)
	return doctorAgentResult{}
}
