package arch

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
)

const modulePath = "code.byted.org/lihuanyu.0w0/gloop"

type packageInfo struct {
	ImportPath string
	Imports    []string
}

func TestInternalImportBoundaries(t *testing.T) {
	cmd := exec.Command("go", "list", "-json", "./internal/...")
	cmd.Dir = "../.."
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list failed: %v", err)
	}

	dec := json.NewDecoder(strings.NewReader(string(out)))
	for dec.More() {
		var pkg packageInfo
		if err := dec.Decode(&pkg); err != nil {
			t.Fatalf("decode go list package: %v", err)
		}
		checkPackageImports(t, pkg)
	}
}

func checkPackageImports(t *testing.T, pkg packageInfo) {
	from := strings.TrimPrefix(pkg.ImportPath, modulePath+"/")
	for _, imp := range pkg.Imports {
		to, ok := internalPackage(imp)
		if !ok {
			continue
		}
		if violatesBoundary(from, to) {
			t.Errorf("%s must not import %s", from, to)
		}
	}
}

func internalPackage(importPath string) (string, bool) {
	prefix := modulePath + "/internal/"
	if !strings.HasPrefix(importPath, prefix) {
		return "", false
	}
	rest := strings.TrimPrefix(importPath, modulePath+"/")
	parts := strings.Split(rest, "/")
	if len(parts) < 2 {
		return rest, true
	}
	return parts[0] + "/" + parts[1], true
}

func violatesBoundary(from, to string) bool {
	if from == to {
		return false
	}

	switch from {
	case "internal/model", "internal/version":
		return true
	case "internal/fsstore":
		return oneOf(to, "internal/orchestrator", "internal/server", "internal/cli", "internal/prompt", "internal/platformtools", "internal/skills")
	case "internal/executor":
		return oneOf(to, "internal/orchestrator", "internal/server", "internal/cli", "internal/fsstore", "internal/prompt", "internal/platformtools", "internal/skills")
	case "internal/orchestrator":
		return oneOf(to, "internal/server", "internal/cli")
	case "internal/server":
		return to == "internal/cli"
	case "internal/prompt", "internal/platformtools", "internal/skills":
		return oneOf(to, "internal/orchestrator", "internal/server", "internal/cli")
	default:
		return false
	}
}

func oneOf(s string, items ...string) bool {
	for _, item := range items {
		if s == item {
			return true
		}
	}
	return false
}
