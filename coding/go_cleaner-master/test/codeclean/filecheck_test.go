package codeclean

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func checkFileContent(t *testing.T, param CleanParams) {
	filecheck := func(filename string) {
		file, err := os.Open(filename)
		if err != nil {
			t.Error(err)
		}
		defer file.Close()

		cmd := exec.Command("filecheck", filename)
		cmd.Stdin = file
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			t.Error(err)
		}
	}

	filepath.Walk(param.RepoDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			t.Error("Error accessing path:", err)
		}

		basename := filepath.Base(path)
		if strings.HasPrefix(basename, "check") && strings.HasSuffix(basename, ".go") {
			t.Logf("Run filecheck on %s\n", path)
			filecheck(path)
		}

		return nil
	})
}

func checkFileNotExist(filename string) func(t *testing.T, param CleanParams) {
	return func(t *testing.T, param CleanParams) {
		t.Logf("Check file should not exist %s\n", filename)
		if _, err := os.Stat(filename); err == nil {
			t.Errorf("File %s should not exist", filename)
		}
	}
}
