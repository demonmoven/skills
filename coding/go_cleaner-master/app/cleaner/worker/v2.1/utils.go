package main

import (
	"bufio"
	"bytes"
	"context"
	"os/exec"
	"runtime/debug"
	"sync"
	"time"

	"code.byted.org/analyzers/go_cleaner/app/cleaner/worker/v2.1/model"
	"code.byted.org/analyzers/go_cleaner/app/cleaner/worker/v2.1/pkg/log"
)

func int64Ptr(i int) *int64 {
	var v = int64(i)
	return &v
}

func strPtr(s string) *string {
	var v = s
	return &v
}

func stringPtrIndirect(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

type cmdWatcher struct {
	reportLock     sync.Mutex
	ctx            context.Context
	closed         int32
	task           *model.CodeCleanTask
	reportInterval time.Duration
	startTime      time.Time

	stdErr, stdOut         bytes.Buffer
	errScanner, outScanner *bufio.Scanner
	errLock, outLock       sync.Mutex
	errHandler, outHandler func(string)

	errPanic      bool
	errPanicObj   any
	errPanicStack string

	outPanic      bool
	outPanicObj   any
	outPanicStack string
}

func newWatcher(ctx context.Context, task *model.CodeCleanTask, reportInterval time.Duration, cmd *exec.Cmd) *cmdWatcher {
	p := &cmdWatcher{ctx: ctx, task: task, reportInterval: reportInterval}
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()
	p.errScanner = bufio.NewScanner(stderr)
	p.outScanner = bufio.NewScanner(stdout)
	p.startTime = time.Now()
	return p
}

func (s *cmdWatcher) Close() {
	s.reportLock.Lock()
	defer s.reportLock.Unlock()

	s.closed = 1
}

func (s *cmdWatcher) setErrHandler(fn func(string)) {
	s.errHandler = fn
}

func (s *cmdWatcher) setOutHandler(fn func(string)) {
	s.outHandler = fn
}

func (s *cmdWatcher) StdErr() string {
	s.errLock.Lock()
	defer s.errLock.Unlock()
	return s.stdErr.String()
}
func (s *cmdWatcher) StdOut() string {
	s.outLock.Lock()
	defer s.outLock.Unlock()
	return s.stdOut.String()
}
func (s *cmdWatcher) AppendOut(out string) {
	s.outLock.Lock()
	defer s.outLock.Unlock()
	s.stdOut.Write([]byte(out + "\n"))
}

func (s *cmdWatcher) Start() {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.errPanic = true
				s.errPanicObj = r
				s.errPanicStack = string(debug.Stack())
			}
		}()
		for s.errScanner.Scan() {
			out := s.errScanner.Text()
			s.errLock.Lock()
			s.stdErr.Write([]byte(out + "\n"))
			s.errLock.Unlock()
			if s.errHandler != nil {
				s.errHandler(out)
			}
		}
	}()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.outPanic = true
				s.outPanicObj = r
				s.outPanicStack = string(debug.Stack())
			}
		}()
		for s.outScanner.Scan() {
			out := s.outScanner.Text()
			s.outLock.Lock()
			s.stdOut.Write([]byte(out + "\n"))
			s.outLock.Unlock()
			if s.outHandler != nil {
				s.outHandler(out)
			}
		}
	}()

	go s.report()
}

func (s *cmdWatcher) report() {
	if s.reportInterval <= time.Second {
		log.Println("reportInterval too small, skip report")
		return
	}
	closed := false
	startTimeUnix := s.startTime.Unix()
	for !closed {
		time.Sleep(s.reportInterval)
		s.reportLock.Lock()
		func() {
			defer s.reportLock.Unlock()
			if s.closed == 1 {
				closed = true
				return
			}
			processing := true
			s.task.TaskResultStdErr = strPtr(s.StdErr())
			s.task.TaskResultStdout = strPtr(s.StdOut())
			log.Println("reporting from worker for task:", s.task.GetID())
			resp, err := cli.SaveCodeCleanTask(s.ctx, &model.SaveCodeCleanTaskRequest{
				Task:       s.task,
				Processing: &processing,
				Worker:     &worker,
				StartTime:  &startTimeUnix,
			})
			if err != nil || resp == nil || resp.BaseResp == nil || resp.BaseResp.StatusCode != 0 {
				log.Error(s.ctx, "SaveCodeCleanTask error", log.KVPair("err", err), log.KVPair("resp", resp))
			}
		}()
	}
}

type testCtxKey struct{}

func setIsTest(ctx context.Context) context.Context {
	return context.WithValue(ctx, testCtxKey{}, true)
}
func isTest(ctx context.Context) bool {
	v, _ := ctx.Value(testCtxKey{}).(bool)
	return v
}
