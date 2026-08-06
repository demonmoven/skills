package main

import (
	"bufio"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"code.byted.org/analyzers/go_cleaner/app/cleaner/worker/v2.1/cxx"
	"code.byted.org/analyzers/go_cleaner/app/cleaner/worker/v2.1/model"
	"code.byted.org/analyzers/go_cleaner/app/cleaner/worker/v2.1/pkg/git"
	"code.byted.org/analyzers/go_cleaner/app/cleaner/worker/v2.1/pkg/log"
	"code.byted.org/analyzers/go_cleaner/app/cleaner/worker/v2.1/pkg/out"
	"code.byted.org/analyzers/go_cleaner/app/cleaner/worker/v2.1/pkg/scm"
	"code.byted.org/analyzers/go_cleaner/app/cleaner/worker/v2.1/pkg/util"
	"code.byted.org/gopkg/ctxvalues"
	"code.byted.org/gopkg/lang/v2/slicex"
	"code.byted.org/gopkg/logid"
)

const (
	ShellExitParameterError = 110
	ShellExitGoVersionError = 111
	MaxTimeout              = 120
	acquireGap              = time.Second * 5
)

var (
	//go:embed version.txt
	version string
)

var goVersionRange []string

func isWorkForCxxCodeTask() bool {
	return len(goVersionRange) == 1 && goVersionRange[0] == "c++"
}

func initGoVersionRange() {
	matchGoVersion := func(goversions []string, v string) bool {
		for _, r := range goversions {
			if strings.Contains(r, v) {
				return true
			}
		}

		return false
	}

	goVersionRange = func() (v []string) {
		for _, i := range strings.Split(os.Getenv("GoVersionRange"), ",") {
			i = strings.TrimSpace(i)
			if i != "" {
				v = append(v, i)
			}
		}

		if _, err := os.Stat("/root/tools/idl_cleaner"); err != nil {
			log.Println("stat idl_cleaner failed with err:", err)
		} else {
			log.Println("idl_cleaner is found")
			v = append(v, "idl") // 本地有idl_cleaner，就加上idl
		}

		if _, err := os.Stat("/root/tools/fixer"); err != nil {
			log.Println("stat fixer failed with err:", err)
		} else if matchGoVersion(v, "1.23") {
			log.Println("fixer is found")
			v = append(v, "panicfix") // 本地有fixer，就加上panicfix
		}

		return
	}()
}

var (
	cli AdminClient = &httpCli{}

	containerID string
	worker      string
	workerEnv   string

	buildBranch, buildCommit string
)

func runInit() {
	log.Init()

	initGoVersionRange()
	cgroupFile, err := os.ReadFile("/proc/self/cgroup")
	if err == nil {
		for _, line := range strings.Split(string(cgroupFile), "\n") {
			segs := strings.Split(line, ":")
			if len(segs) != 3 {
				continue
			}
			if strings.HasPrefix(segs[2], "/docker/") {
				containerID = strings.TrimPrefix(segs[2], "/docker/")
				break
			}
		}
	}
	if worker = os.Getenv("WORKER_NAME"); worker == "" {
		worker = fmt.Sprintf("%s@%s", containerID, os.Getenv("BYTED_HOST_IPV6"))
	}

	buildBranch = strings.TrimSpace(os.Getenv("BUILD_BRANCH"))
	buildCommit = strings.TrimSpace(os.Getenv("BUILD_COMMIT"))
	log.Println("worker name is", worker)
	log.Println("BuildBranch is", buildBranch)
	log.Println("BuildCommit is", buildCommit)
	log.Println("GoVersionRange is", goVersionRange)
	workerEnv = os.Getenv("WORKER_ENV")
	if workerEnv != "" {
		log.Println("worker env is", workerEnv)
	}
}

func main() {
	log.Println("cleaner version:", version)
	runInit()
	if len(os.Args) > 1 {
		if len(os.Args) == 3 {
			if os.Args[1] == "run" {
				runTaskById(false)
			} else if os.Args[1] == "test" {
				runTaskById(true)
			} else {
				log.Println("invalid args:", os.Args[1:])
			}
		} else if os.Args[1] != "version" {
			log.Println("invalid args:", os.Args[1:])
		}
		return
	}
	for {
		ctx := ctxvalues.SetLogID(context.Background(), logid.GenLogID())
		task, reportInterval := acquire(ctx)
		if task == nil {
			log.Println("no task")
			time.Sleep(acquireGap)
		} else {
			log.Printf("get task opt(%s) is_idl_clean(%v) id(%d)", task.GetTaskOpt(), task.GetIsIDLClean(), task.GetID())
			run(ctx, reportInterval, task)
		}
	}
}

func runTaskById(test bool) {
	taskID, err := strconv.ParseInt(os.Args[2], 10, 64)
	if err != nil {
		log.Println("invalid task id:", os.Args[2])
		return
	}
	log.Println("start run task by id:", taskID)
	ctx := ctxvalues.SetLogID(context.Background(), logid.GenLogID())
	resp, err := cli.GetCodeCleanTaskByID(ctx, taskID)
	if err != nil {
		log.Error(ctx, "GetCodeCleanTaskByID error", log.KVPair("err", err), log.KVPair("resp", resp))
		return
	}
	if test {
		ctx = setIsTest(ctx)
	}
	run(ctx, 0, resp.Task)
}

func acquire(ctx context.Context) (*model.CodeCleanTask, time.Duration) {
	resp, err := cli.AcquireCodeCleanTask(ctx, &model.AcquireCodeCleanTaskRequest{
		GoVersion: goVersionRange,
		Worker:    &worker,
	})

	if resp != nil && resp.BaseResp != nil && len(resp.BaseResp.Extra) > 0 && resp.BaseResp.Extra["machine_env"] != "" {
		log.Info(ctx, "acquire machine info:",
			log.KVPair("machine_ip_v6", resp.BaseResp.Extra["machine_ip_v6"]),
			log.KVPair("machine_ip_v4", resp.BaseResp.Extra["machine_ip_v4"]),
			log.KVPair("machine_env", resp.BaseResp.Extra["machine_env"]),
			log.KVPair("machine_dc", resp.BaseResp.Extra["machine_dc"]),
			log.KVPair("machine_cluster", resp.BaseResp.Extra["machine_cluster"]),
			log.KVPair("machine_pod_name", resp.BaseResp.Extra["machine_pod_name"]),
		)
	} else {
		log.Info(ctx, "acquire machine info is empty")
	}

	if err != nil || resp == nil {
		log.Error(ctx, "AcquireCodeCleanTask error", log.KVPair("err", err), log.KVPair("resp", resp))
		return nil, 0
	} else if resp.BaseResp != nil && resp.BaseResp.StatusCode != 0 {
		log.Error(ctx, "AcquireCodeCleanTask error", log.KVPair("resp-code", resp.BaseResp.StatusCode),
			log.KVPair("resp-msg", resp.BaseResp.StatusMessage))
		return nil, 0
	} else if resp.Task == nil {
		log.Info(ctx, "acquire task is nil")
		return nil, 0
	}

	return resp.Task, time.Duration(resp.GetReportInterval()) * time.Second
}

func run(ctx context.Context, reportInterval time.Duration, task *model.CodeCleanTask) {
	if task.GetStorageCleanTask() != nil {
		log.Println("run storage clean")
		runStorageClean(ctx, reportInterval, task)
		return
	}
	if task.GetIsIDLClean() {
		log.Println("run idl clean")
		runIDLClean(ctx, reportInterval, task)
		return
	}
	if task.GetTaskOpt() == "panicfix" {
		log.Println("run panic fixer")
		runFixer(ctx, reportInterval, task)
		return
	}
	if task.GetTaskOpt() == "c++" {
		runCppCleanTask(ctx, reportInterval, task)
		return
	} else if strings.HasPrefix(task.GetTaskOpt(), "go") {
		log.Println("run go clean")
		runGoCleanTask(ctx, reportInterval, task)
		return
	}
	log.Println("run old clean impl")
	var (
		err              error
		stdErr, stdOut   []byte
		errLock, outLock sync.Mutex

		repoName        = task.GetRepoName()
		branch          = "master"
		commitBranch    = "chore/unused_code_auto_clean"
		subdir          = "."
		cleanerFlag     = "-f"
		meegoID         = "M-3000538119"
		review          []string
		unusedEndpoints []string
		timeout         = int64(120)
		setGoVersion    = ""
		handlerPath     = ""
		preScript       = ""
		mainDir         = ""
		repoAppMains    = ""

		deletedEndpoints        []string
		deletedLoC              int64
		totalLoC                int64
		mrLink                  = ""
		downstreamRepoChecked   = map[string]bool{}
		downstreamRepoUnChecked = map[string]bool{}

		goVersion             = ""
		waitingForOtherWorker = false

		done int32
	)

	// 保存结果
	defer func() {
		resp, err := cli.SaveCodeCleanTask(ctx, &model.SaveCodeCleanTaskRequest{
			Task:   task,
			Worker: &worker,
		})
		if err != nil || resp == nil || resp.BaseResp == nil || resp.BaseResp.StatusCode != 0 {
			log.Error(ctx, "SaveCodeCleanTask error", log.KVPair("err", err), log.KVPair("resp", resp))
		}
	}()

	// panic 兜底
	defer func() {
		if r := recover(); r != nil {
			task.TaskStatus = int64Ptr(3)
			task.TaskResultErrMsg = strPtr(fmt.Errorf("panic: %v", r).Error())
		}
	}()

	// task结果收集
	defer func() {
		atomic.StoreInt32(&done, 1)

		task.TaskResultStdout = strPtr(string(stdOut))
		task.TaskOpt = strPtr(goVersion)

		if waitingForOtherWorker {
			task.TaskStatus = int64Ptr(-2)
			return
		}

		if err != nil {
			var existErr *exec.ExitError
			if errors.As(err, &existErr) {
				if status, ok := existErr.Sys().(syscall.WaitStatus); ok && status.ExitStatus() == ShellExitGoVersionError {
					// 退出码111代表当前worker的版本与改仓库的版本不匹配，此时把状态置为-2，并标记上仓库的Go版本，等待之后被执行
					task.TaskStatus = int64Ptr(-2)
					return
				}
			}

			task.TaskStatus = int64Ptr(3)
			task.TaskResultErrMsg = strPtr(err.Error())
			task.TaskResultStdErr = strPtr(string(stdErr))
		} else {
			task.TaskStatus = int64Ptr(2)
			if len(unusedEndpoints) != len(deletedEndpoints) {
				task.TaskStatus = int64Ptr(4) // 废弃接口识别不全也算失败，考虑到项目结构如果不规范工具本身就无法正确识别，单独定义一个状态
			}

			if commitBranch != "" && mrLink == "" {
				// 如果没有正确发起MR, 用分支链接代替
				mrLink = fmt.Sprintf("https://code.byted.org/%s/tree/%s", repoName, commitBranch)
			}

			task.TaskResultMrLink = strPtr(mrLink)
			task.TaskResultUnusedLine = int64Ptr(int(deletedLoC))
			task.TaskResultTotalLine = int64Ptr(int(totalLoC))
			task.Extra.DeletedEndpoint = deletedEndpoints
			for repo := range downstreamRepoChecked {
				task.Extra.DownstreamRepoChecked = append(task.Extra.DownstreamRepoChecked, repo)
			}
		}
	}()

	go func() {
		if reportInterval <= time.Second {
			log.Println("report interval is too small, skip reporting")
			return
		}
		t := time.NewTicker(reportInterval)
		for atomic.LoadInt32(&done) == 0 {
			<-t.C

			outLock.Lock()
			task.TaskResultStdout = strPtr(string(stdOut))
			outLock.Unlock()
			errLock.Lock()
			task.TaskResultStdErr = strPtr(string(stdErr))
			errLock.Unlock()
			processing := true
			log.Println("reporting from worker for task:", task.GetID())
			resp, err := cli.SaveCodeCleanTask(ctx, &model.SaveCodeCleanTaskRequest{
				Task:       task,
				Processing: &processing,
				Worker:     &worker,
			})
			if err != nil || resp == nil || resp.BaseResp == nil || resp.BaseResp.StatusCode != 0 {
				log.Error(ctx, "SaveCodeCleanTask error", log.KVPair("err", err), log.KVPair("resp", resp))
			}
		}
	}()

	if repoName == "" {
		err = fmt.Errorf("old clean worker: repo name is empty")
		return
	}

	// 获取参数
	if e := task.Extra; e != nil {
		if v := e.Branch; v != nil && *v != "" {
			branch = *v
		}
		if v := e.SubDir; v != nil && *v != "" {
			subdir = *v
		}
		if v := e.CommitBranch; v != nil && *v != "" {
			commitBranch = *v
		}
		if v := e.CleanerFlag; v != nil && *v != "" {
			cleanerFlag = *v
		}
		if v := e.Endpoint; v != nil {
			unusedEndpoints = v
		}
		if v := e.Reviewer; v != nil {
			review = v
		}
		if v := e.Timeout; v != nil && *v != 0 {
			timeout = *v
		}
		preScript = stringPtrIndirect(e.PreScript)
		mainDir = stringPtrIndirect(e.MainDir)
		repoAppMains = stringPtrIndirect(e.RepoAppMains)
	}

	//从cleanerFlag中获取timeout、meego_id
	if cleanerFlag != "" {
		timeoutCommand := util.ParseCommandLineInt(cleanerFlag, "--timeout")
		if timeoutCommand != nil {
			timeout = int64(*timeoutCommand)
		}

		flagFields := strings.Fields(cleanerFlag)
		if ind := slicex.IndexOf(flagFields, "--meego_id"); ind != -1 && ind != len(flagFields)-1 {
			meegoID = flagFields[ind+1]
			flagFields = append(flagFields[:ind], flagFields[ind+2:]...)
		} else if pa, ind := slicex.Find(flagFields, func(v string) bool {
			return strings.HasPrefix(v, "--meego_id=")
		}); ind > -1 {
			meegoID = strings.Replace(pa, "--meego_id=", "", 1)
			flagFields = append(flagFields[:ind], flagFields[ind+1:]...)
		}

		if ind := slicex.IndexOf(flagFields, "--set_go_version"); ind != -1 && ind != len(flagFields)-1 {
			setGoVersion = flagFields[ind+1]
			flagFields = append(flagFields[:ind], flagFields[ind+2:]...)
		} else if pa, ind := slicex.Find(flagFields, func(v string) bool {
			return strings.HasPrefix(v, "--set_go_version=")
		}); ind > -1 {
			setGoVersion = strings.Replace(pa, "--set_go_version=", "", 1)
			flagFields = append(flagFields[:ind], flagFields[ind+1:]...)
		}

		if ind := slicex.IndexOf(flagFields, "--handler_path"); ind != -1 && ind != len(flagFields)-1 {
			handlerPath = flagFields[ind+1]
			flagFields = append(flagFields[:ind], flagFields[ind+2:]...)
		} else if pa, ind := slicex.Find(flagFields, func(v string) bool {
			return strings.HasPrefix(v, "--handler_path=")
		}); ind > -1 {
			handlerPath = strings.Replace(pa, "--handler_path=", "", 1)
			flagFields = append(flagFields[:ind], flagFields[ind+1:]...)
		}

		cleanerFlag = strings.Join(flagFields, " ")
	}

	if timeout > MaxTimeout {
		timeout = MaxTimeout
	}
	var preLog strings.Builder
	scmGoVersion, err := getCompilerVersionFromSCM(task, &preLog)
	if err != nil {
		log.Info(ctx, "getCompilerVersionFromSCM fail. ", log.KVPair("err", err))
	}

	// 如果非C/C++ Worker发现该仓库是C/C++代码仓，那么上报后继续等待下发
	if !isWorkForCxxCodeTask() && strings.HasPrefix(scmGoVersion, "llvm") {
		log.Info(ctx, "waiting for c/c++ workers.")
		waitingForOtherWorker = true
		goVersion = "c++"
		return
	}

	// 执行任务
	var script string
	if len(unusedEndpoints) == 0 {
		script, _ = filepath.Abs("script/clean_script.sh")
	} else {
		script, _ = filepath.Abs("script/clean_endpoint_script.sh")
	}

	var cmd *exec.Cmd
	if isWorkForCxxCodeTask() {
		// C/C++ Worker只处理C/C++代码清理任务：
		// 从SCM获取仓库的编译脚本，并且根据获取的编译器版本、设置LLVM_VERSION选项
		script, err = scm.CxxBuildScript(task.Extra.GetRepoName())
		if err != nil {
			log.Info(ctx, "c/c++ build script not found.", log.KVPair("err", err))
		}

		cxxtask := cxx.NewCodeTaskForCXX(task, 2*timeout, script)
		cxxtask.SetCompilerVersion(scmGoVersion)
		cmd = cxxtask.GetCommand()
	} else {
		//timeout --foreground MaxTimeout 保证程序在MaxTimeout直接结束
		cmd = exec.Command("timeout", "--foreground", fmt.Sprintf("%dm", timeout),
			script,
			"--repo", repoName, "--branch", branch, "--subdir", subdir, "--commit_branch", commitBranch,
			"--meego_id", meegoID,
			"--set_go_version", setGoVersion,
			"--scm_go_version", scmGoVersion,
			"--handler_path", handlerPath,
			"--cleaner_flag", cleanerFlag,
			"--endpoint", strings.Join(unusedEndpoints, ","),
			"--downstream_repos", strings.Join(task.Extra.DownstreamRepo, ","),
			"--pre-script", preScript,
			"--annotate_retained_code", strings.Join(task.Extra.AnnotateRetainedCode, "||"),
			"--assigned_base_commit", task.Extra.GetAssignedBaseCommit(),
			"--main-dir", mainDir,
			"--repo-app-mains", repoAppMains,
		)
		cmd.Dir = "./codebase"
	}

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	commandString := fmt.Sprintf("full command: %s,args: %v", cmd.Path, cmd.Args)
	stdOut = append(stdOut, []byte(preLog.String())...)
	stdOut = append(stdOut, []byte(commandString+"\n")...)
	if buildBranch != "" {
		stdOut = append(stdOut, []byte(fmt.Sprintf("worker build branch: %s, build commit: %s\n", buildBranch, buildCommit))...)
	}

	log.Info(ctx, commandString)

	if err = cmd.Start(); err != nil {
		err = fmt.Errorf("cmd.Start err: %w", err)
		return
	}

	scannerOut := bufio.NewScanner(stdout)
	scannerErr := bufio.NewScanner(stderr)

	// 并行读取 stdout 和 stderr
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Error(ctx, "read stdout panic")
			}
		}()
		for scannerOut.Scan() {
			out := scannerOut.Text()

			if strings.HasPrefix(out, "_removed_lines:") {
				line := strings.TrimSpace(out[15:])
				if v, err := strconv.ParseInt(line, 10, 64); err == nil {
					deletedLoC += v
				}
			} else if strings.HasPrefix(out, "_total_lines:") {
				line := strings.TrimSpace(out[13:])
				if v, err := strconv.ParseInt(line, 10, 64); err == nil {
					totalLoC = v
				}
			} else if strings.HasPrefix(out, "_delete_endpoint:") {
				line := strings.TrimSpace(out[17:])
				deletedEndpoints = append(deletedEndpoints, line)
			} else if strings.HasPrefix(out, "_go_version:") {
				line := strings.TrimSpace(out[12:])
				goVersion = strings.TrimSpace(line)
				versionSplit := strings.Split(goVersion, ".")
				if len(versionSplit) > 2 {
					goVersion = fmt.Sprintf("%s.%s", versionSplit[0], versionSplit[1])
				}
			} else if strings.HasPrefix(out, "_downstream_repo_checked:") {
				checkedRepo := strings.TrimSpace(out[25:])
				downstreamRepoChecked[checkedRepo] = true
			}

			outLock.Lock()
			stdOut = append(stdOut, []byte(out+"\n")...)
			outLock.Unlock()
			log.Info(ctx, out) // 打印到日志文件中
		}
	}()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Error(ctx, "read stderr panic")
			}
		}()
		for scannerErr.Scan() {
			text := scannerErr.Text()
			errLock.Lock()
			stdErr = append(stdErr, []byte(text+"\n")...)
			errLock.Unlock()
			log.Error(ctx, text) // 打印到日志文件中
		}
	}()

	// 等待命令执行完毕
	if err = cmd.Wait(); err != nil {
		err = fmt.Errorf("cmd.Wait err: %w", err)
		return
	}
	errLogs := out.Extract(stdErr)
	for _, e := range errLogs.Logs {
		if e.Level == "fatal" {
			err = errors.New(e.Message)
			return
		}
	}
	if deletedLoC == 0 {
		return
	}

	for _, i := range task.Extra.DownstreamRepo {
		if !downstreamRepoChecked[i] {
			downstreamRepoUnChecked[i] = true
		}
	}
	createMRLimit := func() int64 {
		if task.Extra == nil {
			return 0
		}
		return int64(task.Extra.GetCreateMRIfLineChangeGreaterThan())
	}()
	if deletedLoC >= createMRLimit {
		// 提交 MR
		mrLink, err = git.CreateCleanerMR(git.CleanerMR{
			Repo:                          repoName,
			SourceBranch:                  commitBranch,
			TargetBranch:                  branch,
			Reviewer:                      review,
			ReviewerID:                    nil,
			DeletedEndpoint:               deletedEndpoints,
			VerifiedButNotLocatedEndpoint: util.DiffEndpoint(unusedEndpoints, deletedEndpoints),
			UnusedLoC:                     int(deletedLoC),
			MeegoID:                       meegoID,
			DownstreamRepoChecked:         downstreamRepoChecked,
			DownstreamRepoUnChecked:       downstreamRepoUnChecked,
			TaskURL:                       task.GetTaskURL(),
		})
		if err != nil {
			err = fmt.Errorf("create MR err: %w", err)
		}
	} else {
		o := out.NewBytesOutput(&stdOut, &stdErr)
		o.SetWarnf("清理代码行数(%d)未达到创建MR要求最小变更行数(%d)", deletedLoC, task.Extra.GetCreateMRIfLineChangeGreaterThan())
	}
}

func getCompilerVersionFromSCM(task *model.CodeCleanTask, preLog *strings.Builder) (string, error) {
	scmVersion, err := func() (string, error) {
		errNotFound := fmt.Errorf("not found PSM and RepoName from task")
		if task.Extra == nil {
			return "", errNotFound
		}

		psm := task.Extra.GetPSM()
		repo := task.Extra.GetRepoName()
		if psm == "" && repo == "" {
			return "", errNotFound
		}

		version, err := scm.GoVersion(psm, repo)
		if err == nil {
			return version, err
		}

		// Go worker可能会拿到一个c/c++任务
		if err == scm.ErrCompilerVersionNotFoundInImage {
			return scm.CxxVersion(psm, repo)
		}

		return version, err
	}()
	if err != nil {
		if task.Extra != nil && task.Extra.GetRecentBuildGoVersion() != "" {
			// scm仓库信息获取失败，但是有近期编译版本，使用近期编译版本
			// c++不会下发近期编译版本，不会进这个分支
			preLog.WriteString(fmt.Sprintf("get scm version failed, use recent build go version: %s\n", task.Extra.GetRecentBuildGoVersion()))
			return task.Extra.GetRecentBuildGoVersion(), nil
		}
		return "", err
	} else if task.Extra != nil && task.Extra.GetRecentBuildGoVersion() != "" {
		// scm仓库信息获取成功，但是有近期编译版本，并且近期编译版本大于scm仓库版本，使用近期编译版本
		use := theGreaterGoVersion(task.Extra.GetRecentBuildGoVersion(), scmVersion)
		preLog.WriteString(fmt.Sprintf("scm version: %s, recent build go version: %s, use %s\n", scmVersion, task.Extra.GetRecentBuildGoVersion(), use))
		return use, nil
	}
	return scmVersion, nil
}

func theGreaterGoVersion(v1, v2 string) string {
	ver1, ok := parseGoVersion(v1)
	if !ok {
		return v2
	}
	ver2, ok := parseGoVersion(v2)
	if !ok {
		return v1
	}
	if ver1.Major > ver2.Major {
		return v1
	} else if ver1.Major < ver2.Major {
		return v2
	} else if ver1.Minor > ver2.Minor {
		return v1
	} else if ver1.Minor < ver2.Minor {
		return v2
	} else if ver1.Patch > ver2.Patch {
		return v1
	} else if ver1.Patch < ver2.Patch {
		return v2
	}
	return v1
}

type goVersion struct {
	Major int64
	Minor int64
	Patch int64
}

func parseGoVersion(v string) (goVersion, bool) {
	ver := goVersion{}
	segs := strings.Split(v, ".")
	var err error
	if len(segs) == 2 {
		ver.Major, err = strconv.ParseInt(segs[0], 10, 64)
		if err != nil {
			return ver, false
		}
		ver.Minor, err = strconv.ParseInt(segs[1], 10, 64)
		if err != nil {
			return ver, false
		}
		ver.Patch = -1
	} else if len(segs) == 3 {
		ver.Major, err = strconv.ParseInt(segs[0], 10, 64)
		if err != nil {
			return ver, false
		}
		ver.Minor, err = strconv.ParseInt(segs[1], 10, 64)
		if err != nil {
			return ver, false
		}
		ver.Patch, err = strconv.ParseInt(segs[2], 10, 64)
		if err != nil {
			ver.Patch = -1
		}
	} else {
		return ver, false
	}
	return ver, true
}
