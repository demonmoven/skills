package main

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"syscall"
	"time"

	"code.byted.org/analyzers/go_cleaner/app/cleaner/worker/v2.1/model"
	"code.byted.org/analyzers/go_cleaner/app/cleaner/worker/v2.1/pkg/git"
	"code.byted.org/analyzers/go_cleaner/app/cleaner/worker/v2.1/pkg/log"
	"code.byted.org/analyzers/go_cleaner/app/cleaner/worker/v2.1/pkg/out"
)

func runStorageClean(ctx context.Context, reportInterval time.Duration, task *model.CodeCleanTask) {
	var (
		err    error
		mrLink string

		baseCommitID   string
		cleanerVersion string
		psmList        []string
		timeout        = int64(45)
		branch         string
		commitBranch   string
		repoName       string
		scmGoVersion   string

		goVersion string
	)
	startTime := time.Now()
	script, _ := filepath.Abs("script/clean_storage_script.sh")
	var preLog strings.Builder
	scmGoVersion, err = getCompilerVersionFromSCM(task, &preLog)
	if err != nil {
		log.Info(ctx, "getCompilerVersionFromSCM fail. ", log.KVPair("err", err))
	}
	commitBranch = task.Extra.GetCommitBranch()
	branch = task.Extra.GetBranch()
	repoName = task.Extra.GetRepoName()
	psmList = strings.Split(task.StorageCleanTask.GetStoragePSM(), ",")

	cmd := exec.Command("timeout", "--foreground", strconv.FormatInt(timeout, 10)+"m", script,
		"--repo", repoName,
		"--set_go_version", "",
		"--scm_go_version", scmGoVersion,
		"--branch", branch,
		"--commit-branch", commitBranch,
		"--storage-psms", strings.Join(psmList, ","),
	)

	watcher := newWatcher(ctx, task, reportInterval, cmd)
	watcher.setOutHandler(func(s string) {
		if strings.HasPrefix(s, "_go_version:") {
			line := strings.TrimSpace(s[12:])
			goVersion = strings.TrimSpace(line)
			versionSplit := strings.Split(goVersion, ".")
			if len(versionSplit) > 2 {
				goVersion = fmt.Sprintf("%s.%s", versionSplit[0], versionSplit[1])
			}
		}
	})
	watcher.AppendOut(preLog.String())
	watcher.AppendOut(fmt.Sprintf("full command: %s,args: %v", cmd.Path, cmd.Args))
	watcher.Start()
	defer func() {
		startTimeUnix := startTime.Unix()
		resp, err := cli.SaveCodeCleanTask(ctx, &model.SaveCodeCleanTaskRequest{
			Task:      task,
			Worker:    &worker,
			StartTime: &startTimeUnix,
		})
		if err != nil || resp == nil || resp.BaseResp == nil || resp.BaseResp.StatusCode != 0 {
			log.Println("SaveCodeCleanTask error", err, resp)
		} else {
			log.Info(ctx, "SaveCodeCleanTask success", log.KVPair("resp", resp))
		}
	}()

	defer func() {
		watcher.Close()
		task.TaskResultStdErr = strPtr(watcher.StdErr())
		task.TaskResultStdout = strPtr(watcher.StdOut())
		task.TaskOpt = strPtr(goVersion)

		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v, stack: %s", r, string(debug.Stack()))
		}
		if task.Extra == nil {
			task.Extra = &model.CodeCleanTaskExtra{}
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
			log.Println("runStorageClean err: ", err.Error())
		} else {
			task.TaskStatus = int64Ptr(2)
			task.TaskResultMrLink = strPtr(mrLink)
			if commitBranch != "" && mrLink == "" {
				// 如果没有正确发起MR, 用分支链接代替
				mrLink = fmt.Sprintf("https://code.byted.org/%s/tree/%s", repoName, commitBranch)
			}
			task.TaskResultMrLink = strPtr(mrLink)
			task.BaseCommitID = strPtr(baseCommitID)
			task.CleanerVersion = strPtr(cleanerVersion)
			log.Println("runStorageClean success, mrlink: ", mrLink)
		}
	}()

	if err = cmd.Start(); err != nil {
		log.Printf("cmd.Start err: %v", err)
		err = fmt.Errorf("cmd.Start err: %w", err)
		return
	}

	if err = cmd.Wait(); err != nil {
		log.Printf("cleaner执行失败：%s", err.Error())
		task.TaskResultStdErr = strPtr(watcher.StdErr())
		return
	} else {
		task.TaskResultStdErr = strPtr(watcher.StdErr())
	}
	task.TaskResultStdout = strPtr(watcher.StdOut())
	if err != nil {
		return
	}
	if task.StorageCleanTask.OnlyCheck {
		return
	}
	o := out.Extract([]byte(task.GetTaskResultStdout()))
	if len(o.KVs) == 0 || o.KVs[""] == "" {
		task.TaskResultStdout = strPtr(fmt.Sprintf("未检测到疑似引用存储(%s)代码\n%s", task.StorageCleanTask.GetStoragePSM(), task.GetTaskResultStdout()))
		return
	}
	appendStdOut(task, fmt.Sprintf("创建MR中...Repo: %s, SourceBranch: %s, TargetBranch: %s", repoName, commitBranch, branch))
	// 提交 MR
	mrLink, err = git.CreateCleanerMR(git.CleanerMR{
		Repo:         repoName,
		SourceBranch: commitBranch,
		TargetBranch: branch,
		TaskURL:      task.GetTaskURL(),
	})

	if err != nil {
		err = fmt.Errorf("新建MR失败: %w", err)
	}
}

func appendStdOut(task *model.CodeCleanTask, s string) {
	if task.TaskResultStdout == nil {
		task.TaskResultStdout = strPtr("")
	}
	task.TaskResultStdout = strPtr(*task.TaskResultStdout + "\n" + s)
}
