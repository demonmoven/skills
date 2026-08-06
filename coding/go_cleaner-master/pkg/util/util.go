package util

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/bytedance/gopkg/lang/fastrand"
)

func GetWorkingRepositoryRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed get repository root failed: %v, %s", err, string(output))
	}
	return strings.TrimSpace(string(output)), nil
}

func ResetWorkingRepository() error {
	if IsCleanerIntegrationTestEnv() {
		return nil
	}

	cmd := exec.Command("git", "reset", "--hard", "HEAD")
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

func IsInSideWorkTree() bool {
	if IsCleanerIntegrationTestEnv() {
		return true
	}

	cmd := exec.Command("git", "-C", ".", "rev-parse", "--is-inside-work-tree")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}

	return strings.TrimSpace(string(output)) == "true"
}

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func RandString(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[fastrand.Intn(len(charset))]
	}
	return string(b)
}

func LowerRandString(length int) string {
	return strings.ToLower(RandString(length))
}

func MustRunCmdWithStdIO(name string, arg ...string) {
	Must(RunCmdWithStdIO(name, arg...))
}

func RunCmdWithStdIO(name string, arg ...string) error {
	return RunCmdWithStdIOAt("", name, arg...)
}

func RunCmdWithStdIOAt(dir string, name string, arg ...string) error {
	defer func() {
		fmt.Println(strings.Repeat("<", 50))
	}()
	fmt.Println(strings.Repeat(">", 50))
	if dir != "" {
		fmt.Printf(">>> (%s)\n", dir)
	} else if wd, _ := os.Getwd(); wd != "" {
		fmt.Printf(">>> (%s)\n", wd)
	}
	fmt.Printf(">>> %s %v\n", name, arg)
	cmd := exec.Command(name, arg...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if dir != "" {
		cmd.Dir = dir
	}
	return cmd.Run()
}

func Must(err error) {
	if err != nil {
		panic(err)
	}
}

func RepoPath(path string) (string, error) {
	repoRoot, err := GetWorkingRepositoryRoot()
	if err != nil {
		return "", fmt.Errorf("failed get repository root: %v", err)
	}
	return filepath.Join(repoRoot, path), nil
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

func GetVersion() string {
	_, file, _, ok := runtime.Caller(1)
	if !ok {
		return "unknown"
	} else if !strings.Contains(file, "@") {
		return "main"
	}
	ver := file[strings.LastIndex(file, "@")+1:]
	return strings.Split(ver, "/")[0]
}
