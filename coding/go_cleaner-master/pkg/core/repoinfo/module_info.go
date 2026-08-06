package repoinfo

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func GetWorkingRepositoryRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed get repository root failed: %v, %s", err, string(output))
	}
	return strings.TrimSpace(string(output)), nil
}

func ModuleDirAndNameOf(path string) (dir string, module string, err error) {
	repoRoot, err := GetWorkingRepositoryRoot()
	if err != nil {
		return "", "", fmt.Errorf("failed get repository root: %v", err)
	}
	modDir, modName := findGoModDir(repoRoot, filepath.Join(repoRoot, path))
	if modDir == "" {
		return "", "", fmt.Errorf("failed find go.mod from dir(%s) to path: %s", repoRoot, path)
	}
	return modDir, modName, nil
}

func findGoModDir(repoRoot, dir string) (string, string) {
	if dir == "/" {
		return "", ""
	}
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
		modFile, err := os.Open(filepath.Join(dir, "go.mod"))
		if err != nil {
			return "", ""
		}
		s := bufio.NewScanner(modFile)
		for s.Scan() {
			line := s.Text()
			if strings.HasPrefix(line, "module ") {
				return dir, strings.TrimSpace(line[7:])
			}
		}
		return "", ""
	}
	if dir == repoRoot {
		return "", "" // 如果在仓库根目录都没找到，直接退出查找，因为仓库根目录下的go
	}
	return findGoModDir(repoRoot, filepath.Dir(dir))
}
