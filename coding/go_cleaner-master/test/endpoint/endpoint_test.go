package endpoint

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"code.byted.org/analyzers/go_cleaner/pkg/util"
)

var (
	workDir         string
	newWorkEnv      bool
	endpointTestDir string
	testBranPrefix  string
	currentBranch   string
)

const (
	testMeegoID = "M-3000538119"
)

func runCase(t *testing.T, params CleanEndpointParams, posts ...func(*testing.T, CleanEndpointParams)) {
	if params.RepoDir == "" {
		t.Errorf("repo dir is empty")
		return
	}
	if err := os.Chdir(params.RepoDir); err != nil {
		t.Errorf("chdir to %s failed: %v", params.RepoDir, err)
		return
	}
	runOnRandBranch(t, func(branch string) {
		params.Branch = branch
		params.CommitBranch = branch
		runCleanEndpointScript(params)
		for _, post := range posts {
			post(t, params)
		}
	})
}

func runOnRandBranch(t *testing.T, fn func(string)) {
	branch := testBranPrefix + "/" + t.Name()
	err := exec.Command("git", "checkout", "-b", branch).Run()
	if err != nil {
		t.Errorf("failed checkout to test branch %s: %v", branch, err)
		return
	}
	if err := runWithStdIO("git", "push", "origin", branch); err != nil {
		t.Errorf("failed push test branch %s: %v", branch, err)
		return
	}
	defer func() {
		if err = exec.Command("git", "checkout", currentBranch).Run(); err != nil {
			t.Logf("failed checkout to %s: %v", currentBranch, err)
		}
		if !t.Failed() {
			if err = exec.Command("git", "branch", "-D", branch).Run(); err != nil {
				t.Logf("failed delete test branch %s: %v", branch, err)
			}
			if err = exec.Command("git", "push", "origin", ":"+branch).Run(); err != nil {
				t.Logf("failed delete test branch %s: %v", branch, err)
			}
		}
	}()
	fn(branch)
}

func TestMain(m *testing.M) {
	prepareRunEnv()
	code := m.Run()
	cleanup(code)
	os.Exit(code)
}

type CleanEndpointParams struct {
	SetGoVersion string `flag:"set_go_version"`
	ScmGoVerison string `flag:"scm_go_version"`
	Branch       string `flag:"branch"`
	SubDir       string `flag:"subdir"`
	CommitBranch string `flag:"commit_branch"`
	MeegoID      string `flag:"meego_id"`
	CleanerFlag  string `flag:"cleaner_flag"`
	Endpoint     string `flag:"endpoint"`
	HandlerPath  string `flag:"handler_path"`
	Downstream   string `flag:"downstream_repos"`
	MainDir      string `flag:"main_dir"`
	RepoDir      string `flag:"repo-dir"`
}

func (p CleanEndpointParams) buildFlags() []string {
	rv := reflect.ValueOf(p)
	flags := make([]string, 0, rv.NumField()*2+1)
	flags = append(flags, cleanEndpointScript())
	for i := 0; i < rv.NumField(); i++ {
		f := rv.Type().Field(i)
		if rv.Field(i).String() == "" {
			continue
		}
		flags = append(flags, "--"+f.Tag.Get("flag"), rv.Field(i).String())
	}
	return flags
}

func cleanEndpointScript() string {
	return filepath.Join(workDir, "worker", "script", "clean_endpoint_script.sh")
}

func checkExpectedFiles(t *testing.T, params CleanEndpointParams) {
	root := params.RepoDir
	unexpected := make([]string, 0)
	anyExp := false
	if err := filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		for _, s := range []string{
			".exp.txt",
			".expected.txt",
		} {
			if strings.HasSuffix(info.Name(), ".go"+s) {
				anyExp = true
				expected, _ := os.ReadFile(path)
				actual, _ := os.ReadFile(strings.TrimSuffix(path, s))
				if strings.TrimSpace(string(expected)) != strings.TrimSpace(string(actual)) {
					unexpected = append(unexpected, path)
				}
			}
		}
		return nil
	}); err != nil {
		t.Errorf("checkExpectedFiles walk %s failed: %v", root, err)
	}
	if !anyExp {
		t.Errorf("no expected files found")
	} else if len(unexpected) > 0 {
		t.Errorf("unexpected files: %v", unexpected)
	}
}

func runCleanEndpointScript(params CleanEndpointParams) {
	defer func() {
		fmt.Println(strings.Repeat("<", 30))
	}()
	fmt.Println(strings.Repeat(">", 30))
	flags := params.buildFlags()
	wd, _ := os.Getwd()
	if wd != "" {
		fmt.Printf(">>> (%s) bash %v\n", wd, flags)
	} else {
		fmt.Printf(">>> bash %v\n", flags)
	}
	cmd := exec.Command("bash", flags...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stderr = os.Stderr
	cmd.Env = make([]string, 0, len(os.Environ()))
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "PATH=") {
			e = e + ":" + filepath.Join(workDir, "bin") + ":" + filepath.Join(workDir, "worker", "bin")
		}
		cmd.Env = append(cmd.Env, e)
	}
	must(cmd.Run())
}

func prepareRunEnv() {
	defer printRunEnv()

	endpointTestDir = getEndpointTestDir()
	currentBranch = getCurrentBranch()
	testBranPrefix = "eptest_" + strconv.FormatInt(time.Now().Unix(), 10) + "_" + util.RandString(5)
	if os.Getenv("CLEANER_ROOT") != "" {
		workDir = os.Getenv("CLEANER_ROOT")
		setEnvPath()
		return
	}
	newWorkEnv = true
	workDir = filepath.Join(os.TempDir(), "cleaner-ep-test-"+util.RandString(10))
	must(os.MkdirAll(filepath.Join(workDir, "bin"), 0755))
	mustBuildWorker()
	mustBuildCleaner()
	setEnvPath()
}

func printRunEnv() {
	fmt.Println(strings.Repeat(">", 50))
	fmt.Println("- TestWorkDir:", workDir)
	fmt.Println("- NewWorkEnv:", newWorkEnv)
	fmt.Println("- EndpointTestDir:", endpointTestDir)
	fmt.Println("- TestBranchPrefix:", testBranPrefix)
	fmt.Println("- CurrentBranch:", currentBranch)
	fmt.Println(strings.Repeat(">", 50))
}

func cleanup(_ int) {
	if os.Getenv("NO_CLEANUP") != "1" && newWorkEnv {
		_ = os.RemoveAll(workDir)
	}
	os.Chdir(endpointTestDir)
}

func setEnvPath() {
	path := os.Getenv("PATH")
	os.Setenv("PATH", filepath.Join(workDir, "worker", "bin")+":"+filepath.Join(workDir, "bin")+":"+path)
}

func getEndpointTestDir() string {
	_, fileName, _, _ := runtime.Caller(0)
	dir, _ := filepath.Split(fileName)
	return dir
}

func getCurrentBranch() string {
	cnt, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		panic(err)
	}
	return strings.TrimSpace(string(cnt))
}

func mustBuildWorker() {
	fmt.Println("build worker start...")
	must(os.Chdir(filepath.Join(endpointTestDir, "../../app/cleaner/worker/v2.1")))
	mustRunWithStdIO("go", "version")
	mustRunWithStdIO("bash", "build.sh")
	must(exec.Command("mv", "output", filepath.Join(workDir, "worker")).Run())
	fmt.Println("build worker success")
}

func mustBuildCleaner() {
	fmt.Println("build cleaner start...")
	must(os.Chdir(filepath.Join(endpointTestDir, "../../cmd/cleaner")))
	mustRunWithStdIO("go", "build", "-o", "cleaner")
	must(exec.Command("mv", "cleaner", filepath.Join(workDir, "bin")).Run())
	fmt.Println("build cleaner success")
}

func mustRunWithStdIO(name string, arg ...string) {
	must(runWithStdIO(name, arg...))
}

func runWithStdIO(name string, arg ...string) error {
	defer func() {
		fmt.Println(strings.Repeat("<", 30))
	}()
	fmt.Println(strings.Repeat(">", 30))
	wd, _ := os.Getwd()
	if wd != "" {
		fmt.Printf(">>> (%s) %s %v\n", wd, name, arg)
	} else {
		fmt.Printf(">>> %s %v\n", name, arg)
	}
	cmd := exec.Command(name, arg...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
