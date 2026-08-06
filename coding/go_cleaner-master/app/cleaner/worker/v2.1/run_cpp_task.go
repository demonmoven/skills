package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os/exec"
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
	"code.byted.org/gopkg/logid"
)

func runCppCleanTask(ctx context.Context, reportInterval time.Duration, task *model.CodeCleanTask) {
	var (
		err              error
		stdErr, stdOut   []byte
		errLock, outLock sync.Mutex

		repoName        = task.GetRepoName()
		branch          = "master"
		commitBranch    = "chore/unused_code_auto_clean"
		meegoID         = "M-3000538119"
		review          []string
		unusedEndpoints []string
		timeout         = int64(120)

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
		log.Println("task save status:", task.GetTaskStatus())
		log.Println("task save mr link:", task.GetTaskResultMrLink())
		if isTest(ctx) {
			return
		}
		logID := logid.GenLogID()
		ctx = ctxvalues.SetLogID(context.Background(), logID)
		log.Println("task save with new log id:", logID)
		resp, err := cli.SaveCodeCleanTask(ctx, &model.SaveCodeCleanTaskRequest{
			Task:   task,
			Worker: &worker,
		})
		if err != nil || resp == nil || (resp.BaseResp != nil && resp.BaseResp.StatusCode != 0) {
			log.Error(ctx, "SaveCodeCleanTask error", log.KVPair("err", err), log.KVPair("resp", resp))
		} else {
			log.Info(ctx, "SaveCodeCleanTask succeed...")
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
		err = fmt.Errorf("cpp worker: repo name is empty")
		return
	}

	// 获取参数
	if e := task.Extra; e != nil {
		if v := e.Branch; v != nil && *v != "" {
			branch = *v
		}
		if v := e.CommitBranch; v != nil && *v != "" {
			commitBranch = *v
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
	}

	if timeout > MaxTimeout {
		timeout = MaxTimeout
	}
	var preLog strings.Builder
	scmGoVersion, err := getCompilerVersionFromSCM(task, &preLog)
	if err != nil {
		log.Info(ctx, "getCompilerVersionFromSCM fail. ", log.KVPair("err", err))
	}

	// C/C++ Worker只处理C/C++代码清理任务：
	// 从SCM获取仓库的编译脚本，并且根据获取的编译器版本、设置LLVM_VERSION选项
	script, err := scm.CxxBuildScript(task.Extra.GetRepoName())
	if err != nil {
		log.Info(ctx, "c/c++ build script not found.", log.KVPair("err", err))
		return
	}

	cxxtask := cxx.NewCodeTaskForCXX(task, 2*timeout, script)
	cxxtask.SetCompilerVersion(scmGoVersion)
	cmd := cxxtask.GetCommand()

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
	if task.Extra.SkipCreateMR {
		o := out.NewBytesOutput(&stdOut, &stdErr)
		o.SetInfo("已跳过MR创建")
	} else if deletedLoC >= createMRLimit {
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
