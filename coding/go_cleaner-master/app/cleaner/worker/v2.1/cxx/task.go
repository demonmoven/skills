package cxx

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"code.byted.org/analyzers/go_cleaner/app/cleaner/worker/v2.1/model"
)

type CodeTaskForCXX struct {
	RepoName        string
	Branch          string
	CommitBranch    string
	CleanerFlags    []string
	Timeout         int64
	CompilerVersion string
}

func NewCodeTaskForCXX(task *model.CodeCleanTask, timeout int64, buildScript string) *CodeTaskForCXX {
	getOrDefaultValue := func(value *string, defval string) string {
		if value == nil {
			return defval
		}
		return *value
	}

	var flags []string
	for _, f := range strings.Split(getOrDefaultValue(task.Extra.CleanerFlag, ""), " ") {
		flags = append(flags, f)
	}
	flags = append(flags, "--build-command")
	flags = append(flags, buildScript)

	return &CodeTaskForCXX{
		RepoName:     task.GetRepoName(),
		Branch:       getOrDefaultValue(task.Extra.Branch, "master"),
		CommitBranch: getOrDefaultValue(task.Extra.CommitBranch, "chore/unused_code_auto_clean"),
		CleanerFlags: flags,
		Timeout:      timeout,
	}
}

func (c *CodeTaskForCXX) SetCompilerVersion(version string) {
	if version == "llvm11" {
		c.CompilerVersion = "11"
		return
	}

	c.CompilerVersion = "16"
}

func (c *CodeTaskForCXX) GetCommand() (cmd *exec.Cmd) {
	script, _ := filepath.Abs("script/clean_cxx_script.sh")

	cmd = exec.Command("timeout",
		"--foreground",
		fmt.Sprintf("%dm", c.Timeout),
		script,
		"--repo", c.RepoName,
		"--branch", c.Branch,
		"--commit_branch", c.CommitBranch,
		"--cleaner_flag", strings.Join(c.CleanerFlags, " "),
	)

	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Env = append(cmd.Env, "LLVM_VERSION="+c.CompilerVersion)

	return cmd
}
