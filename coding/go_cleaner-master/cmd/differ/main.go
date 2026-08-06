package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"code.byted.org/analyzers/go_cleaner/pkg/util"
)

var (
	cleanerBase   string
	cleanerTarget string
	casesPath     string
	cleanType     string
	cleanerRoot   string
	parallel      int

	baseCleanerRoot   string
	targetCleanerRoot string

	clean bool
)

const (
	cleanTypeEndpoint    = "endpoint"
	cleanTypeCode        = "code"
	relPathRunTestDocker = "app/cleaner/worker/v2.1/run_test_docker.sh"
)

func init() {
	baseCleanerRoot = util.DefaultEnv().Dir("base")
	targetCleanerRoot = util.DefaultEnv().Dir("target")
}

func main() {
	defer func() {
		// _ = os.RemoveAll(cleanerTestsDir)
	}()
	preCheck()
	loadArgs()
	downloadCleanerVersions()

	switch cleanType {
	case cleanTypeEndpoint:
		if clean {
			if err := cleanEndpointItems(); err != nil {
				os.Exit(1)
			}
			return
		}
		if err := diffEndpoint(); err != nil {
			os.Exit(1)
		}
	}
}

func preCheck() {
	path, err := exec.LookPath("docker")
	if err != nil || path == "" {
		log.Fatalln("command `docker` not found in current session!")
	}
}

func loadArgs() {
	flag.StringVar(&cleanerBase, "base", "", "cleaner base version, a commit id or a branch name")
	flag.StringVar(&cleanerTarget, "target", "", "cleaner target version, a commit id or a branch name")
	flag.StringVar(&casesPath, "case-file", "", "the case file path")
	flag.StringVar(&cleanType, "type", "", "clean type: endpoint / code(not support now)")
	flag.StringVar(&cleanerRoot, "root", "", "root of cleaner project")
	flag.IntVar(&parallel, "parallel", 0, "parallel run cases")
	flag.BoolVar(&clean, "clean", false, "clean docker images and remote branch")
	flag.Parse()

	if cleanerBase == "" || cleanerTarget == "" {
		panic(fmt.Errorf("cleaner base/target(%s/%s) version not specified", cleanerBase, cleanerTarget))
	} else if casesPath == "" {
		panic(fmt.Errorf("cases file path not provided"))
	}
	util.SetCreateDockerScript(filepath.Join(cleanerRoot, relPathRunTestDocker))
	if cleanType == cleanTypeEndpoint {
		return
	} else if cleanType == cleanTypeCode {
		panic(fmt.Errorf("code clean diff not support now"))
	}
}

func downloadCleanerVersions() {
	downloadCleanerVersion(targetCleanerRoot, cleanerTarget)
	downloadCleanerVersion(baseCleanerRoot, cleanerBase)
}

func downloadCleanerVersion(root, revision string) {
	if err := util.RunCmdWithStdIO("git", "clone", "git@code.byted.org:analyzers/go_cleaner.git", root); err != nil {
		log.Fatalf("failed download go_cleaner base version(%s)\n", revision)
	} else if err := util.RunCmdWithStdIOAt(root, "git", "fetch", "origin", revision+":differing"); err != nil {
		log.Fatalf("failed fetch cleaner base version(%v)\n", revision)
	} else if err := util.RunCmdWithStdIOAt(root, "git", "checkout", "differing"); err != nil {
		log.Fatalf("failed checkout cleaner base version(%v)\n", revision)
	}
}
