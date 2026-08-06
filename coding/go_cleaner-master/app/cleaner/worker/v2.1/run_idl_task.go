package main

import (
	"context"
	"fmt"
	"os/exec"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"code.byted.org/analyzers/go_cleaner/app/cleaner/worker/v2.1/model"
	"code.byted.org/analyzers/go_cleaner/app/cleaner/worker/v2.1/pkg/git"
	"code.byted.org/analyzers/go_cleaner/app/cleaner/worker/v2.1/pkg/log"
	"code.byted.org/analyzers/go_cleaner/app/cleaner/worker/v2.1/pkg/out"
	"code.byted.org/analyzers/go_cleaner/app/cleaner/worker/v2.1/pkg/util"
	"code.byted.org/gopkg/ctxvalues"
	"code.byted.org/gopkg/logid"
)

func runIDLClean(ctx context.Context, reportInterval time.Duration, task *model.CodeCleanTask) {
	var (
		err    error
		mrLink string

		baseCommitID     string
		removedLineCount int
		deletedEndpoints []string
		cleanerVersion   string
	)
	startTime := time.Now()
	cmd := exec.Command("timeout", "--foreground", "10m", "/root/tools/idl_cleaner",
		"--repo", task.IDLCleanTask.RepoName,
		"--source-branch", task.IDLCleanTask.SourceBranch,
		"--commit-branch", task.IDLCleanTask.CommitBranch,
		"--idl-path", task.IDLCleanTask.IDLPath,
		"--methods", strings.Join(task.IDLCleanTask.Methods, ","),
		"--remove",
	)

	watcher := newWatcher(ctx, task, reportInterval, cmd)
	if buildBranch != "" {
		watcher.AppendOut(fmt.Sprintf("worker build branch: %s, build commit: %s", buildBranch, buildCommit))
	}
	watcher.AppendOut(fmt.Sprintf("full command: %s,args: %v", cmd.Path, cmd.Args))
	watcher.Start()
	defer func() {
		log.Println("task save status:", task.GetTaskStatus())
		log.Println("task save mr link:", task.GetTaskResultMrLink())
		if isTest(ctx) {
			return
		}
		logID := logid.GenLogID()
		ctx = ctxvalues.SetLogID(context.Background(), logID)
		log.Println("task save with new log id:", logID)
		startTimeUnix := startTime.Unix()
		resp, err := cli.SaveCodeCleanTask(ctx, &model.SaveCodeCleanTaskRequest{
			Task:      task,
			Worker:    &worker,
			StartTime: &startTimeUnix,
		})
		if err != nil || resp == nil || (resp.BaseResp != nil && resp.BaseResp.StatusCode != 0) {
			log.Error(ctx, "SaveCodeCleanTask error", log.KVPair("err", err), log.KVPair("resp", resp))
		} else {
			log.Info(ctx, "SaveCodeCleanTask succeed...")
		}
	}()

	defer func() {
		watcher.Close()
		task.TaskResultStdErr = strPtr(watcher.StdErr())
		task.TaskResultStdout = strPtr(watcher.StdOut())

		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v, stack: %s", r, string(debug.Stack()))
		}
		if task.Extra == nil {
			task.Extra = &model.CodeCleanTaskExtra{}
		}
		if err != nil {
			task.TaskStatus = int64Ptr(3)
			task.TaskResultErrMsg = strPtr(err.Error())
			log.Println("runIDLClean err: ", err.Error())
		} else {
			task.TaskStatus = int64Ptr(2)
			task.TaskResultMrLink = strPtr(mrLink)
			if task.IDLCleanTask.CommitBranch != "" && mrLink == "" {
				// 如果没有正确发起MR, 用分支链接代替
				mrLink = fmt.Sprintf("https://code.byted.org/%s/tree/%s", task.IDLCleanTask.RepoName, task.IDLCleanTask.CommitBranch)
			}
			task.TaskResultMrLink = strPtr(mrLink)
			task.BaseCommitID = strPtr(baseCommitID)
			task.TaskResultUnusedLine = int64Ptr(removedLineCount)
			task.Extra.DeletedEndpoint = deletedEndpoints
			task.CleanerVersion = strPtr(cleanerVersion)
			log.Println("runIDLClean success, mrlink: ", mrLink)
		}
	}()

	if err = cmd.Start(); err != nil {
		err = fmt.Errorf("cmd.Start err: %w", err)
		return
	}

	if err = cmd.Wait(); err != nil {
		task.TaskResultStdErr = strPtr(watcher.StdErr())
		err = fmt.Errorf("cleaner执行失败：%s", task.GetTaskResultStdErr())
	} else {
		task.TaskResultStdErr = strPtr(watcher.StdErr())
	}
	task.TaskResultStdout = strPtr(watcher.StdOut())
	output := out.Extract([]byte(task.GetTaskResultStdout()))
	baseCommitID = output.KVs["base_commit_id"]
	removedLineCount, _ = strconv.Atoi(output.KVs["removed_line_count"])
	cleanerVersion = output.KVs["cleaner_version"]
	if output.KVs["removed_methods"] != "" {
		deletedEndpoints = strings.Split(output.KVs["removed_methods"], ",")
	}
	if err != nil {
		return
	}

	// 提交 MR
	mrLink, err = git.CreateCleanerMR(git.CleanerMR{
		Repo:                          task.IDLCleanTask.RepoName,
		SourceBranch:                  task.IDLCleanTask.CommitBranch,
		TargetBranch:                  task.IDLCleanTask.SourceBranch,
		DeletedEndpoint:               deletedEndpoints,
		VerifiedButNotLocatedEndpoint: util.DiffEndpoint(task.IDLCleanTask.Methods, deletedEndpoints),
		UnusedLoC:                     removedLineCount,
		TaskURL:                       task.GetTaskURL(),
	})

	if err != nil {
		err = fmt.Errorf("新建MR失败: %w", err)
	}
}
