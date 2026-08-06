package cxx

import (
	"os"
	"testing"
)

var DefaultTimeout = 120

func TestCXXCodeTaskCreation(t *testing.T) {
	codeTask := &CodeTaskForCXX{
		RepoName:     "ad/docking_i18n",
		Branch:       "master",
		CommitBranch: "",
		CleanerFlags: []string{
			"--build-command",
			"build.sh",
		},
	}

	cmd := codeTask.GetCommand()
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	if err := cmd.Run(); err != nil {
		t.Errorf("code task run failed, err: %v", err)
	}
}
