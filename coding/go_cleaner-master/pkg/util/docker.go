package util

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	workDir   string
	testUname string

	validGoVersions = map[string]*goVersionConf{
		"1.18": {ImportsVersion: "v0.18.0"},
		"1.19": {ImportsVersion: "v0.24.0"},
		"1.20": {ImportsVersion: "v0.24.0"},
		"1.21": {ImportsVersion: "v0.24.0"},
		"1.22": {ImportsVersion: "v0.26.0"},
		"1.23": {ImportsVersion: "v0.26.0"},
	}

	stdOut Out = &stdOutImpl{}
)

type goVersionConf struct {
	ImportsVersion string
}

func init() {
	if IsCleanerIntegrationTestEnv() {
		testUname = "cleaner_test_" + LowerRandString(5)
		workDir = filepath.Join(os.TempDir(), testUname)
		Must(os.Mkdir(workDir, 0777))
	}
	
	prepareDockerGitConf()
}

func StdOut() Out { return stdOut }

func WorkDir() string { return workDir }

func prepareDockerGitConf() {
	if os.Getenv("HOME") == "/home/tiger" {
		return
	}
	homeDir := os.Getenv("HOME")
	if _, err := os.Stat(filepath.Join(homeDir, "gitconfig/tiktok/.gitconfig")); err == os.ErrNotExist {
		_ = os.MkdirAll(filepath.Join(homeDir, "gitconfig/tiktok"), 0777)
		if err = os.WriteFile(filepath.Join(homeDir, "gitconfig/tiktok/.gitconfig"), []byte(strings.NewReplacer(
			"{{oauth2}}", GetIesOpsOauth(),
			"{{user}}", os.Getenv("USER"),
		).Replace(gitConfigTpl)), 0777); err != nil {
			panic(err)
		}
	}
	if _, err := os.Stat(filepath.Join(homeDir, "gitconfig/git_ssh")); err == os.ErrNotExist {
		err := exec.Command("cp", "-r", filepath.Join(homeDir, ".ssh"), filepath.Join(homeDir, "gitconfig/git_ssh")).Run()
		if err != nil {
			panic(err)
		}
	}
}

var (
	gitConfigTpl = `[user]
email = go_cleaner
name = go_cleaner
[url "https://oauth2:{{oauth2}}@code.byted.org/"]
        insteadOf=https://code.byted.org/
[url "https://oauth2:{{oauth2}}@code.byted.org/"]
        insteadOf=git@code.byted.org:
#需要配合ssh key兼容老版本的gerrit
[url "ssh://{{user}}@git.byted.org:29418"]
     insteadOf = https://git.byted.org
[url "ssh://{{user}}@git.byted.org:29418"]
     insteadOf = https://review.byted.org
[alias]
root = !pwd`

	createDockerScript string
)

const (
	inDockerCleanEndpointScriptPath = "script/clean_endpoint_script.sh"
	inDockerCleanScriptPath         = "script/clean_script.sh"
)

type DockerInstance struct {
	name      string
	goVersion string
	out       Out

	err error
}

func (d *DockerInstance) CleanEndpoint(out Out, params CleanParams) error {
	return dockerExecBash(out, d.name, inDockerCleanEndpointScriptPath, params.BuildFlags()...)
}

func (d *DockerInstance) Clean(out Out, params CleanParams) error {
	return dockerExecBash(out, d.name, inDockerCleanScriptPath, params.BuildFlags()...)
}

func (d *DockerInstance) InstallGoImports() error {
	goImportsVersion := validGoVersions[d.goVersion].ImportsVersion
	if err := dockerExecBin(d.out, d.name,
		"go", "install", "golang.org/x/tools/cmd/goimports@"+goImportsVersion); err != nil {
		return fmt.Errorf("failed install goimports")
	}
	return nil
}
func (d *DockerInstance) InstallCleaner(cleanerVersion string) error {
	if err := dockerExecBin(d.out, d.name,
		"go", "install", "code.byted.org/analyzers/go_cleaner/cmd/cleaner@"+cleanerVersion); err != nil {
		return fmt.Errorf("failed install cleaner")
	}
	if d.err != nil && strings.HasPrefix(d.err.Error(), "failed install cleaner") {
		d.err = nil
	}
	return nil
}

func SetCreateDockerScript(script string) {
	createDockerScript = script
}

type Out interface {
	Err() io.Writer
	Out() io.Writer
}

type stdOutImpl struct{}

func (s *stdOutImpl) Err() io.Writer {
	return os.Stderr
}
func (s *stdOutImpl) Out() io.Writer {
	return os.Stdout
}

type fileOut struct {
	file     *os.File
	writeStd bool
}

func (o *fileOut) Err() io.Writer {
	if o.writeStd {
		return &withStdWriter{file: o.file, std: os.Stderr}
	}
	return o.file
}
func (o *fileOut) Out() io.Writer {
	if o.writeStd {
		return &withStdWriter{file: o.file, std: os.Stdout}
	}
	return o.file
}

type bufOut struct {
	buf bytes.Buffer
}

func (b *bufOut) Err() io.Writer { return &b.buf }
func (b *bufOut) Out() io.Writer { return &b.buf }

type withStdWriter struct {
	file *os.File
	std  io.Writer
}

func (w *withStdWriter) Write(p []byte) (n int, err error) {
	w.std.Write(p)
	return w.file.Write(p)
}

func RunCmdAt(out Out, dir, name string, arg ...string) error {
	defer func() {
		fmt.Fprintln(out.Err(), strings.Repeat("<", 50))
	}()
	fmt.Fprintln(out.Out(), strings.Repeat(">", 50))
	if dir != "" {
		fmt.Fprintf(out.Out(), ">>> @(%s)\n", dir)
	} else if wd, _ := os.Getwd(); wd != "" {
		fmt.Fprintf(out.Out(), ">>> @(%s)\n", wd)
	}
	fmt.Fprintf(out.Out(), ">>> %s %v\n", name, arg)
	cmd := exec.Command(name, arg...)
	cmd.Stdout = out.Err()
	cmd.Stderr = out.Out()
	if dir != "" {
		cmd.Dir = dir
	}
	return cmd.Run()
}
func RunCmd(out Out, name string, arg ...string) error {
	return RunCmdAt(out, "", name, arg...)
}
func dockerExecBash(out Out, docker string, script string, args ...string) error {
	execArgs := append([]string{"exec", docker, "bash", script}, args...)
	cmd := exec.Command("docker", execArgs...)
	cmd.Stderr = out.Err()
	cmd.Stdout = out.Out()
	fmt.Fprintln(out.Out(), strings.Repeat(">", 50))
	fmt.Fprintf(out.Out(), "[docker-bash(%s)]>>> %s %v\n", docker, script, args)
	defer fmt.Fprintln(out.Out(), strings.Repeat("<", 50))
	return cmd.Run()
}

func dockerExecBin(out Out, docker string, name string, args ...string) error {
	execArgs := append([]string{"exec", docker, name}, args...)
	cmd := exec.Command("docker", execArgs...)
	cmd.Stderr = out.Err()
	cmd.Stdout = out.Out()
	fmt.Fprintln(out.Out(), strings.Repeat(">", 50))
	fmt.Fprintf(out.Out(), "[docker(%s)]>>> %s %v\n", docker, name, args)
	defer fmt.Fprintf(out.Out(), "%s\n", strings.Repeat("<", 50))
	return cmd.Run()
}

func IsValidGoVersion(goVersion string) bool {
	return validGoVersions[goVersion] != nil
}
