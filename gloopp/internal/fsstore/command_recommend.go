package fsstore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type CommandRecommendation struct {
	AllowedCommand
	Reason  string `json:"reason"`
	Source  string `json:"source"`
	Present bool   `json:"present"`
}

func RecommendMageCommands(projectDir string, existing []AllowedCommand) ([]CommandRecommendation, error) {
	if projectDir == "" {
		return nil, fmt.Errorf("project_dir 不能为空")
	}
	if info, err := os.Stat(projectDir); err != nil {
		return nil, err
	} else if !info.IsDir() {
		return nil, fmt.Errorf("project_dir 不是目录: %s", projectDir)
	}

	seen := map[string]bool{}
	for _, item := range existing {
		seen[item.ID] = true
	}
	var out []CommandRecommendation
	add := func(item AllowedCommand, source, reason string) {
		item = item.withDefaults()
		out = append(out, CommandRecommendation{
			AllowedCommand: item,
			Source:         source,
			Reason:         reason,
			Present:        seen[item.ID],
		})
	}

	if fileExists(projectDir, "go.mod") {
		add(AllowedCommand{ID: "go-test", Description: "Run Go tests in the current workspace package tree.", Command: "go", Args: []string{"test", "./..."}}, "go.mod", "Go module detected")
		add(AllowedCommand{ID: "go-vet", Description: "Run go vet in the current workspace package tree.", Command: "go", Args: []string{"vet", "./..."}}, "go.mod", "Go module detected")
		add(AllowedCommand{ID: "go-test-race", Description: "Run Go tests with the race detector.", Command: "go", Args: []string{"test", "-race", "./..."}, TimeoutMs: 5 * 60 * 1000}, "go.mod", "Go module detected; useful for concurrency-sensitive changes")
	}
	if fileExists(projectDir, "Cargo.toml") {
		add(AllowedCommand{ID: "cargo-test", Description: "Run Rust tests.", Command: "cargo", Args: []string{"test"}}, "Cargo.toml", "Rust crate detected")
		add(AllowedCommand{ID: "cargo-clippy", Description: "Run Rust clippy lint checks.", Command: "cargo", Args: []string{"clippy", "--all-targets", "--all-features"}}, "Cargo.toml", "Rust crate detected")
	}
	if fileExists(projectDir, "pyproject.toml") || fileExists(projectDir, "pytest.ini") {
		add(AllowedCommand{ID: "pytest", Description: "Run Python tests with pytest.", Command: "pytest"}, firstExisting(projectDir, "pyproject.toml", "pytest.ini"), "Python project detected")
		add(AllowedCommand{ID: "ruff-check", Description: "Run Ruff lint checks.", Command: "ruff", Args: []string{"check", "."}}, firstExisting(projectDir, "pyproject.toml", "ruff.toml"), "Python project detected")
	}
	if scripts, ok := readPackageScripts(filepath.Join(projectDir, "package.json")); ok {
		for _, name := range []string{"test", "lint", "typecheck", "build"} {
			if _, exists := scripts[name]; exists {
				id := "npm-run-" + strings.ReplaceAll(name, ":", "-")
				if name == "test" {
					id = "npm-test"
				}
				add(AllowedCommand{
					ID:          id,
					Description: "Run npm script: " + name,
					Command:     "npm",
					Args:        []string{"run", name},
				}, "package.json", "package.json script detected: "+name)
			}
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Present != out[j].Present {
			return !out[i].Present && out[j].Present
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}

func fileExists(dir, name string) bool {
	info, err := os.Stat(filepath.Join(dir, name))
	return err == nil && !info.IsDir()
}

func firstExisting(dir string, names ...string) string {
	for _, name := range names {
		if fileExists(dir, name) {
			return name
		}
	}
	return ""
}

func readPackageScripts(path string) (map[string]string, bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if err := json.Unmarshal(raw, &pkg); err != nil || len(pkg.Scripts) == 0 {
		return nil, false
	}
	return pkg.Scripts, true
}
