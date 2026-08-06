package util

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	"code.byted.org/gopkg/env"
)

type CleanParams struct {
	Repo                 string `flag:"repo"`
	SetGoVersion         string `flag:"set_go_version"`
	ScmGoVerison         string `flag:"scm_go_version"`
	Branch               string `flag:"branch"`
	SubDir               string `flag:"subdir"`
	CommitBranch         string `flag:"commit_branch"`
	MeegoID              string `flag:"meego_id"`
	CleanerFlag          string `flag:"cleaner_flag"`
	Endpoint             string `flag:"endpoint"`
	HandlerPath          string `flag:"handler_path"`
	Downstream           string `flag:"downstream_repos"`
	AnnotateRetainedCode string `flag:"annotate_retained_code"`
	AssignedBaseCommit   string `flag:"assigned_base_commit"`
	MainDir              string `flag:"main-dir"`
	RepoDir              string `flag:"repo-dir"`
	PreScript            string `flag:"pre-script"`
}

func (p CleanParams) BuildFlags() []string {
	rv := reflect.ValueOf(p)
	flags := make([]string, 0, rv.NumField()*2)
	for i := 0; i < rv.NumField(); i++ {
		f := rv.Type().Field(i)
		if rv.Field(i).String() == "" {
			continue
		}
		flags = append(flags, "--"+f.Tag.Get("flag"), rv.Field(i).String())
	}
	return flags
}

type TestEnv struct {
	workDir string
	jobID   string

	envLog Out

	createDockerLock sync.Mutex
	dockerInstances  map[string]*DockerInstance

	rwdLock        sync.Mutex
	rwds           map[string]string
	remoteBranches map[string][]string
}

var (
	defaultEnv *TestEnv
)

func init() {
	if IsCleanerIntegrationTestEnv() {
		defaultEnv = NewTestEnv().MustInit() 
	}
}

func DefaultEnv() *TestEnv { return defaultEnv }

func NewTestEnv() *TestEnv {
	jobID := "cleaner-test-" + RandString(10)
	return &TestEnv{
		workDir:         filepath.Join(os.TempDir(), jobID),
		jobID:           jobID,
		rwds:            map[string]string{},
		dockerInstances: map[string]*DockerInstance{},
		remoteBranches:  map[string][]string{},
	}
}
func (e *TestEnv) JobID() string { return e.jobID }
func (e *TestEnv) MustInit() *TestEnv {
	e.dockerInstances = map[string]*DockerInstance{}
	Must(os.MkdirAll(filepath.Join(e.workDir, "logs"), 0777))
	Must(os.MkdirAll(filepath.Join(e.workDir, "codes"), 0777))
	var err error
	e.envLog, err = e.NewLogFile("env.log", true)
	if err != nil {
		panic(err)
	}
	return e
}
func (e *TestEnv) GetDocker(nameSuffix, goVersion, root, cleanerVersion string, writeStd bool) (*DockerInstance, error) {
	istName := fmt.Sprintf("%s.%s_go%s_container", testUname, nameSuffix, goVersion)

	if createDockerScript == "" {
		return nil, fmt.Errorf("create docker script not set")
	}
	e.createDockerLock.Lock()
	defer e.createDockerLock.Unlock()
	if e.dockerInstances[istName] != nil {
		if e.dockerInstances[istName].err != nil {
			return nil, e.dockerInstances[istName].err
		}
		return e.dockerInstances[istName], nil
	}

	if validGoVersions[goVersion] == nil {
		return nil, fmt.Errorf("invalid go version: (%s)", goVersion)
	}

	goToolDir := filepath.Join(workDir, "go_bins", goVersion)
	err := os.MkdirAll(goToolDir, 0777)
	if err != nil {
		return nil, fmt.Errorf("failed create go tool dir: %v", err)
	}
	file, err := e.NewLogFile("docker-"+goVersion+".log", writeStd)
	if err != nil {
		return nil, fmt.Errorf("failed create docker log file: %v", err)
	}
	if err := RunCmdAt(file, filepath.Join(root, "app/cleaner/worker/v2.1"), "bash", createDockerScript,
		testUname+"."+nameSuffix, goVersion, env.IDC(), goToolDir); err != nil {
		return nil, fmt.Errorf("failed run docker: %v", err)
	}
	ist := &DockerInstance{
		name:      istName,
		goVersion: goVersion,
		out:       file,
	}
	e.dockerInstances[istName] = ist
	if err := ist.InstallGoImports(); err != nil {
		ist.err = err
		return nil, ist.err
	}
	if err := ist.InstallCleaner(cleanerVersion); err != nil {
		ist.err = fmt.Errorf("failed install cleaner@%s", cleanerVersion)
		return nil, ist.err
	}
	return ist, nil
}
func (e *TestEnv) CloneSingleBranch(out Out, repo, branch, outputDir string) (string, error) {
	repoDir := filepath.Join(e.workDir, "codes", strings.ReplaceAll(repo, "/", "_"))
	if outputDir != "" {
		repoDir = filepath.Join(e.workDir, "codes", outputDir)
	}
	err := RunCmdAt(out, "", "git", "clone", "--single-branch", "-b", branch, repo, repoDir)
	if err != nil {
		return "", err
	}
	return repoDir, nil
}
func (e *TestEnv) Dir(dir string, elems ...string) string {
	if len(elems) == 0 {
		return filepath.Join(e.workDir, dir)
	}
	return filepath.Join(append([]string{e.workDir, dir}, elems...)...)
}
func (e *TestEnv) CreateRemoteWorkBranch(out Out, repoName, c1, b1, b2 string) error {
	if repoName == "" {
		return nil
	}
	e.rwdLock.Lock()
	defer e.rwdLock.Unlock()

	targetDir := e.rwds[repoName]
	if targetDir == "" {
		var err error
		if targetDir, err = e.initRepo(out, repoName); err != nil {
			return err
		}
		e.rwds[repoName] = targetDir
	}
	rp := strings.NewReplacer("{{c1}}", c1, "{{b1}}", b1, "{{b2}}", b2)
	cmdStr := rp.Replace("git fetch origin {{c1}}:{{b1}} && git push origin {{b1}} && git fetch origin {{c1}}:{{b2}} && git push origin {{b2}}")
	e.remoteBranches[repoName] = append(e.remoteBranches[repoName], b1, b2)
	return RunCmdAt(out, targetDir, "bash", "-c", cmdStr)
}
func (e *TestEnv) DeleteRemoteBranches(out Out, repoName string) error {
	if repoName == "" {
		return nil
	}
	e.rwdLock.Lock()
	defer e.rwdLock.Unlock()
	targetDir := e.rwds[repoName]
	if targetDir == "" {
		var err error
		if targetDir, err = e.initRepo(out, repoName); err != nil {
			return fmt.Errorf("failed init repo: %v", err)
		}
		e.rwds[repoName] = targetDir
	}
	if err := RunCmdAt(out, targetDir, "git", "fetch", "origin", "refs/heads/eptest_diff*:refs/remotes/origin/eptest_diff*"); err != nil {
		return fmt.Errorf("failed fetch remote branches: %v", err)
	}
	cmd := exec.Command("git", "branch", "-r", "--list", "origin/eptest_diff*")
	cmd.Dir = targetDir
	byteCnt, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed list remote branches: \n%v", string(byteCnt))
	}
	cnt := strings.TrimSpace(string(byteCnt))
	if cnt == "" {
		fmt.Println("repo remote branch(", targetDir, ") count: 0")
		return nil
	}
	branches := strings.Split(cnt, "\n")
	delBranches := []string{}
	for _, branch := range branches {
		branch = strings.TrimSpace(branch)
		if strings.HasPrefix(branch, "origin/eptest_diff") {
			branch = strings.TrimPrefix(branch, "origin/")
			delBranches = append(delBranches, ":"+branch)
		}
	}
	if len(delBranches) == 0 {
		return nil
	}
	return RunCmdAt(out, targetDir, "git", append([]string{"push", "origin"}, delBranches...)...)
}

func (e *TestEnv) initRepo(out Out, repoName string) (string, error) {
	targetDir := filepath.Join(e.workDir, "codes", "rwd_"+strings.ReplaceAll(repoName, "/", "_"))
	if err := os.MkdirAll(targetDir, 0777); err != nil {
		return "", err
	} else if err := RunCmdAt(out, targetDir, "bash", "-c",
		fmt.Sprintf("git init && git remote add origin https://oauth2:%s@code.byted.org/%s.git", GetIesOpsOauth(), repoName)); err != nil {
		return "", err
	}
	return targetDir, nil
}
func (e *TestEnv) NewLogFile(fileName string, writeStd bool) (Out, error) {
	if _, err := os.Stat(filepath.Join(e.workDir, "logs")); os.IsNotExist(err) {
		_ = os.MkdirAll(filepath.Join(e.workDir, "logs"), 0777)
	} else if err != nil {
		return nil, fmt.Errorf("failed stat log dir: %v", err)
	}
	filePath := filepath.Join(e.workDir, "logs", fileName)
	file, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed create file: %v", err)
	}
	return &fileOut{file: file, writeStd: writeStd}, nil
}
func (e *TestEnv) Distroy() {
	if os.Getenv("NOCLEAN") == "1" {
		return
	}
	e.rmDockers()
	e.delRemoteBranches()
}

func (e *TestEnv) rmDockers() {
	wg := sync.WaitGroup{}
	wg.Add(len(e.dockerInstances))
	for _, ist := range e.dockerInstances {
		go func(ist *DockerInstance) {
			defer wg.Done()
			_ = RunCmd(ist.out, "docker", "rm", "-f", ist.name)
		}(ist)
	}
	wg.Wait()
}
func (e *TestEnv) delRemoteBranches() {
	for repo, branches := range e.remoteBranches {
		delBranches := []string{}
		for _, branch := range branches {
			delBranches = append(delBranches, ":"+branch)
		}
		RunCmdAt(e.envLog, e.rwds[repo], "git", append([]string{"push", "origin"}, delBranches...)...)
	}
}

func (e *TestEnv) Logf(format string, args ...any) {
	fmt.Fprintf(e.envLog.Out(), format+"\n", args...)
}

// 判断是否为集成测试情况
func IsCleanerIntegrationTestEnv() bool {
	val, ok := os.LookupEnv("FOR_CLEANER_INTEGRATION_TEST")
	return ok && val == "1"
}

func SetCleanerIntegrationTestEnv() {
	os.Setenv("FOR_CLEANER_INTEGRATION_TEST", "1")
}

// 判断是否为线上测试环境
func IsCleanerOnlineTestEnv() bool {
	val, ok := os.LookupEnv("FOR_CLEANER_TEST")
	return ok && val == "1"
}

// 判断是否为线上测试环境 - debug模式
func IsCleanerOnlineDebugEnv() bool {
	val, ok := os.LookupEnv("FOR_CLEANER_DEBUG")
	return ok && val == "1"
}

func IsNotOnlineEnv() bool {
	return IsCleanerIntegrationTestEnv() || IsCleanerOnlineDebugEnv() || IsCleanerOnlineTestEnv()
}