package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
)

func TestSkillListJSON_ProgressiveDisclosure(t *testing.T) {
	stdout, stderr, code := captureStdIO(t, func() int {
		return runSkillCmd(t.Context(), nil, []string{"list", "--json", "--class", "warrior"})
	})
	if code != 0 {
		t.Fatalf("skill list failed with code %d, stderr=%s", code, stderr)
	}
	if strings.Contains(stdout, "Body") || strings.Contains(stdout, "## CLI Contract") || strings.Contains(stdout, "# gloop-quest-execution") {
		t.Fatalf("skill list --json leaked skill body:\n%s", stdout)
	}
	dec := json.NewDecoder(strings.NewReader(stdout))
	count := 0
	for {
		var row map[string]any
		if err := dec.Decode(&row); err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("decode manifest failed: %v\n%s", err, stdout)
		}
		if _, ok := row["body"]; ok {
			t.Fatalf("manifest should not contain body: %+v", row)
		}
		if row["name"] == "" || row["description"] == "" {
			t.Fatalf("manifest missing name/description: %+v", row)
		}
		count++
	}
	if count == 0 {
		t.Fatal("expected at least one manifest")
	}
}

func TestSkillShow_LoadsBodyOnDemand(t *testing.T) {
	stdout, stderr, code := captureStdIO(t, func() int {
		return runSkillCmd(t.Context(), nil, []string{"show", "gloop-quest-execution"})
	})
	if code != 0 {
		t.Fatalf("skill show failed with code %d, stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, "# gloop-quest-execution") || !strings.Contains(stdout, "## CLI Contract") {
		t.Fatalf("skill show should include full body, got:\n%s", stdout)
	}
}

func TestIDLExport_All(t *testing.T) {
	// v0.3.4：bundle 只导出 skills + classes，不再导出 tools / agent_schemas
	// （native platform tool 分发层已移除，agent→gloop 走 CLI syscall）。
	stdout, stderr, code := captureStdIO(t, func() int {
		return runIDLCmd(t.Context(), nil, []string{"export"})
	})
	if code != 0 {
		t.Fatalf("idl export failed with code %d, stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, `"idl_version": "gloop.idl.v1"`) {
		t.Fatalf("idl export missing idl version:\n%s", stdout)
	}
	if !strings.Contains(stdout, `"name": "gloop-quest-execution"`) {
		t.Fatalf("idl export missing skill manifest:\n%s", stdout)
	}
	if strings.Contains(stdout, `"tools"`) {
		t.Fatalf("v0.3.4+ idl export must not carry tools field:\n%s", stdout)
	}
	if strings.Contains(stdout, `"agent_schemas"`) {
		t.Fatalf("v0.3.4+ idl export must not carry agent_schemas field:\n%s", stdout)
	}
	if strings.Contains(stdout, `"cli_usage"`) {
		t.Fatalf("v0.3.4+ idl export must not carry tool cli_usage (no tools):\n%s", stdout)
	}
	// skill body 不该泄露到 manifest
	if strings.Contains(stdout, "## CLI Contract") {
		t.Fatalf("idl export must not leak skill body:\n%s", stdout)
	}
}

func TestIDLExport_OnlySkills(t *testing.T) {
	stdout, stderr, code := captureStdIO(t, func() int {
		return runIDLCmd(t.Context(), nil, []string{"export", "--only", "skills"})
	})
	if code != 0 {
		t.Fatalf("idl export --only skills failed with code %d, stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, `"name": "gloop-quest-execution"`) {
		t.Fatalf("skills export missing gloop-quest-execution:\n%s", stdout)
	}
	if strings.Contains(stdout, `"tools"`) {
		t.Fatalf("skills-only export should not include bundle tools field:\n%s", stdout)
	}
}

func TestIDLExport_OnlyClasses(t *testing.T) {
	// v0.3.4：classes 只导出 per-class skill 可见性，不再有 tools
	stdout, stderr, code := captureStdIO(t, func() int {
		return runIDLCmd(t.Context(), nil, []string{"export", "--only", "classes"})
	})
	if code != 0 {
		t.Fatalf("idl export --only classes failed with code %d, stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, `"warrior"`) || !strings.Contains(stdout, `"mage"`) {
		t.Fatalf("classes export missing class entries:\n%s", stdout)
	}
	if !strings.Contains(stdout, `"skills"`) {
		t.Fatalf("classes export should list per-class skills:\n%s", stdout)
	}
	if strings.Contains(stdout, `"tools"`) {
		t.Fatalf("v0.3.4+ classes export must not carry tools:\n%s", stdout)
	}
}

func captureStdIO(t *testing.T, fn func() int) (stdout string, stderr string, code int) {
	t.Helper()
	oldOut := os.Stdout
	oldErr := os.Stderr
	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	errR, errW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = outW
	os.Stderr = errW
	defer func() {
		os.Stdout = oldOut
		os.Stderr = oldErr
	}()

	// 后台 goroutine 负责读 pipe，防止 fn() 输出超过 pipe buffer（默认 64KB）时写阻塞死锁。
	var outBuf bytes.Buffer
	var errBuf bytes.Buffer
	done := make(chan struct{})
	go func() {
		_, _ = outBuf.ReadFrom(outR)
		_, _ = errBuf.ReadFrom(errR)
		close(done)
	}()

	code = fn()
	_ = outW.Close()
	_ = errW.Close()
	<-done

	return outBuf.String(), errBuf.String(), code
}
