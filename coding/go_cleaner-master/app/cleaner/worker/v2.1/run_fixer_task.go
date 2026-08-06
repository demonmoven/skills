package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"time"

	"code.byted.org/analyzers/go_cleaner/app/cleaner/worker/v2.1/model"
	"code.byted.org/analyzers/go_cleaner/app/cleaner/worker/v2.1/pkg/git"
	"code.byted.org/analyzers/go_cleaner/app/cleaner/worker/v2.1/pkg/log"
	"code.byted.org/analyzers/go_cleaner/app/cleaner/worker/v2.1/pkg/out"
	"code.byted.org/gopkg/ctxvalues"
	"code.byted.org/gopkg/logid"
)

func runFixer(ctx context.Context, reportInterval time.Duration, task *model.CodeCleanTask) {
	var (
		err    error
		mrLink string

		baseCommitID     string
		removedLineCount int
		deletedEndpoints []string
		cleanerVersion   string
	)
	startTime := time.Now()

	tempFile, err := os.CreateTemp(os.TempDir(), "fixer-task-*.json")
	if err != nil {
		err = fmt.Errorf("create temp file error: %w", err)
		return
	}

	defer os.Remove(tempFile.Name())

	// 序列化FixerTask到临时文件
	if err = json.NewEncoder(tempFile).Encode(task.FixerTask); err != nil {
		err = fmt.Errorf("write FixerTask error: %w", err)
		return
	}
	tempFile.Close()

	// TODO: 根据FixerTask的参数生成json文件，并且作为第一个参数传递给/root/tools/fixer
	cmd := exec.Command("timeout", "--foreground", "10m", "fixer",
		tempFile.Name(),
	)

	log.Println("fixer cmd: ", cmd.String())

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
			log.Println("runFixer err: ", err.Error())
		} else {
			task.TaskStatus = int64Ptr(2)
			task.TaskResultMrLink = strPtr(mrLink)
			if *task.FixerTask.GitBranch != "" && mrLink == "" {
				// 如果没有正确发起MR, 用分支链接代替
				mrLink = fmt.Sprintf("https://code.byted.org/%s/tree/%s", task.FixerTask.GetRepoName(), *task.FixerTask.GitBranch)
			}
			task.TaskResultMrLink = strPtr(mrLink)
			task.BaseCommitID = strPtr(baseCommitID)
			task.TaskResultUnusedLine = int64Ptr(removedLineCount)
			task.Extra.DeletedEndpoint = deletedEndpoints
			task.CleanerVersion = strPtr(cleanerVersion)
			log.Println("runFixer success, mrlink: ", mrLink)
		}
	}()

	if err = cmd.Start(); err != nil {
		err = fmt.Errorf("cmd.Start err: %w", err)
		return
	}

	if err = cmd.Wait(); err != nil {
		task.TaskResultStdErr = strPtr(watcher.StdErr())
		err = fmt.Errorf("fixer execute failed：%s", task.GetTaskResultStdErr())
	} else {
		task.TaskResultStdErr = strPtr(watcher.StdErr())
	}
	task.TaskResultStdout = strPtr(watcher.StdOut())
	output := out.Extract([]byte(task.GetTaskResultStdout()))

	newBranch := output.KVs["fixer_new_branch"]
	workSpace := output.KVs["fixer_workspace"]
	cleanerVersion = output.KVs["fixer_version"]

	if err != nil {
		return
	}

	log.Printf("[%s@%s] fixerTask: %s", workSpace, newBranch, task.FixerTask.String())

	var originDirPath string

	originDirPath, err = os.Getwd()
	if err != nil {
		err = fmt.Errorf("getwd failed: %w", err)
		return
	}

	originDirPath, err = filepath.Abs(originDirPath)
	if err != nil {
		err = fmt.Errorf("get abspath failed: %w", err)
		return
	}

	if err = os.Chdir(workSpace); err != nil {
		err = fmt.Errorf("chdir failed: %w", err)
		return
	}

	defer func() {
		if err = os.Chdir(originDirPath); err != nil {
			err = fmt.Errorf("chdir origin directory failed: %w", err)
		}
	}()

	title := fmt.Sprintf("fix: solve the panic issue %s", *task.FixerTask.ErrorId)
	description := fmt.Sprintf("this patch fixed issue https://acha.bytedance.net/error_info?error_id=%s", *task.FixerTask.ErrorId)

	mrLink, err = git.CreateCustomMR(git.CleanerMR{
		Repo:         task.FixerTask.GetRepoName(),
		SourceBranch: newBranch,
		TargetBranch: *task.FixerTask.GitBranch,
		TaskURL:      task.GetTaskURL(),
	}, title, description)

	if err != nil {
		err = fmt.Errorf("create new MR failed: %w", err)
	}
}
