package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"code.byted.org/analyzers/go_cleaner/pkg/util"
)

type EpCase struct {
	Name         string           `json:"name,omitempty"`
	GoVersion    string           `json:"go_version,omitempty"`
	Repo         string           `json:"repo,omitempty"`
	RepoCommitID string           `json:"repo_commit_id,omitempty"`
	Args         util.CleanParams `json:"args,omitempty"`
}

func diffEndpoint() error {
	epCases := loadEpCases()
	return runEndpointDiff(epCases)
}

func cleanEndpointItems() error {
	epCases := loadEpCases()
	for _, c := range epCases {
		if err := util.DefaultEnv().DeleteRemoteBranches(util.StdOut(), c.Repo); err != nil {
			fmt.Printf("delete remote branch failed: %v\n", err)
		}
	}
	return nil
}

func loadEpCases() []EpCase {
	cnt, err := os.ReadFile(casesPath)
	if err != nil {
		fmt.Printf("read cases file failed: %v\n", err)
		os.Exit(1)
	}
	testCases := []EpCase{}
	err = json.Unmarshal(cnt, &testCases)
	if err != nil {
		fmt.Printf("unmarshal cases file failed: %v\n", err)
		os.Exit(1)
	}
	return testCases
}

func runEndpointDiff(epCases []EpCase) error {
	defer func() { util.DefaultEnv().Distroy() }()
	var succeed, failed, diff int32
	var startCount, doneCount int32
	diffResults := sync.Map{}

	limitChan := make(chan struct{}, 1)
	if parallel > 0 {
		limitChan = make(chan struct{}, parallel)
	}

	wg := sync.WaitGroup{}
	for i, c := range epCases {
		i, c := i, c
		limitChan <- struct{}{}
		wg.Add(1)
		startTime := time.Now()
		logFile, err := util.DefaultEnv().NewLogFile(fmt.Sprintf("case-%d.log", i), parallel <= 1)
		if err != nil {
			wg.Done()
			fmt.Printf("create log file failed: %v\n", err)
			continue
		}
		go func() {
			fmt.Printf("(%d/%d)task-%d started\n", atomic.AddInt32(&startCount, 1), len(epCases), i)
			defer func() {
				fmt.Printf("(%d/%d)task-%d finished(%v)\n", atomic.AddInt32(&doneCount, 1), len(epCases), i, time.Since(startTime))
				<-limitChan
				wg.Done()
				if err := recover(); err != nil {
					atomic.AddInt32(&failed, 1)
					diffResults.Store(i, &diffResult{Err: fmt.Sprintf("panic: %v", err)})
				}
			}()
			if !util.IsValidGoVersion(c.GoVersion) {
				atomic.AddInt32(&failed, 1)
				diffResults.Store(i, &diffResult{Err: fmt.Sprintf("invalid go version: %s", c.GoVersion)})
				return
			}

			baseBranch := fmt.Sprintf("eptest_diff_%s_%d_base", util.DefaultEnv().JobID(), i)
			targetBranch := fmt.Sprintf("eptest_diff_%s_%d_target", util.DefaultEnv().JobID(), i)
			if err := util.DefaultEnv().CreateRemoteWorkBranch(logFile, c.Repo, c.RepoCommitID, baseBranch, targetBranch); err != nil {
				atomic.AddInt32(&failed, 1)
				diffResults.Store(i, &diffResult{Err: fmt.Sprintf("create branch failed:%v", err)})
				return
			}

			baseDocker, err := util.DefaultEnv().GetDocker("base", c.GoVersion, baseCleanerRoot, cleanerBase, parallel <= 1)
			if err != nil {
				atomic.AddInt32(&failed, 1)
				diffResults.Store(i, &diffResult{Err: fmt.Sprintf("create base docker failed:%v", err)})
				return
			}
			targetDocker, err := util.DefaultEnv().GetDocker("target", c.GoVersion, targetCleanerRoot, cleanerTarget, parallel <= 1)
			if err != nil {
				atomic.AddInt32(&failed, 1)
				diffResults.Store(i, &diffResult{Err: fmt.Sprintf("create base docker failed:%v", err)})
				return
			}

			baseParams := c.Args
			baseParams.Repo = c.Repo
			baseParams.Branch = baseBranch
			baseParams.CommitBranch = baseBranch
			startTime := time.Now()
			if err := baseDocker.CleanEndpoint(logFile, baseParams); err != nil {
				atomic.AddInt32(&failed, 1)
				diffResults.Store(i, &diffResult{Err: fmt.Sprintf("clean endpoint base failed:%v", err)})
				return
			}
			endTime := time.Now()
			baseCost := endTime.Sub(startTime)
			startTime = endTime

			targetParams := c.Args
			targetParams.Repo = c.Repo
			targetParams.Branch = targetBranch
			targetParams.CommitBranch = targetBranch
			if err := targetDocker.CleanEndpoint(logFile, targetParams); err != nil {
				atomic.AddInt32(&failed, 1)
				diffResults.Store(i, &diffResult{Err: fmt.Sprintf("clean endpoint target failed:%v", err)})
				return
			}
			endTime = time.Now()
			targetCost := endTime.Sub(startTime)

			repoURL, err := util.GetRepoOriginURL(c.Repo)
			if err != nil {
				atomic.AddInt32(&failed, 1)
				diffResults.Store(i, &diffResult{Err: fmt.Sprintf("get repo url failed:%v", err)})
				return
			}
			baseResultDir, err := util.DefaultEnv().CloneSingleBranch(logFile, repoURL, baseBranch, fmt.Sprintf("case_%d_base_result", i))
			if err != nil {
				atomic.AddInt32(&failed, 1)
				diffResults.Store(i, &diffResult{Err: fmt.Sprintf("clone base branch failed:%v", err)})
				return
			}
			targetResultDir, err := util.DefaultEnv().CloneSingleBranch(logFile, repoURL, targetBranch, fmt.Sprintf("case_%d_target_result", i))
			if err != nil {
				atomic.AddInt32(&failed, 1)
				diffResults.Store(i, &diffResult{Err: fmt.Sprintf("clone target branch failed:%v", err)})
				return
			}
			if result := diffDirGoFiles(baseResultDir, targetResultDir); !result.OK {
				atomic.AddInt32(&diff, 1)
				diffResults.Store(i, result)
			} else {
				result.BaseCost, result.TargetCost = baseCost, targetCost
				atomic.AddInt32(&succeed, 1)
			}
		}()
	}
	wg.Wait()
	util.DefaultEnv().Logf("[Done]succeed:%d failed:%d diff:%d", succeed, failed, diff)
	diffResults.Range(func(key, value any) bool {
		i := key.(int)
		r := value.(*diffResult)
		if r.Err != "" {
			util.DefaultEnv().Logf("\t[%d#%s]failed: %s", i, epCases[i].Name, r.Err)
		} else if len(r.DiffFiles) > 0 {
			util.DefaultEnv().Logf("\t[%d#%s]diff: %d files", i, epCases[i].Name, len(r.DiffFiles))
			for _, f := range r.DiffFiles {
				util.DefaultEnv().Logf("\t\t\t%s", f)
			}
		} else {
			util.DefaultEnv().Logf("\t[%d#%s]cost: base:%v vs target:%v", i, epCases[i].Name, r.BaseCost, r.TargetCost)
		}
		return true
	})
	if failed+diff > 0 {
		return fmt.Errorf("failed:%d diff:%d", failed, diff)
	}
	return nil
}

type diffResult struct {
	CaseName  string
	OK        bool
	Err       string
	DiffFiles []string

	BaseCost, TargetCost time.Duration
}

func diffDirGoFiles(src, dst string) *diffResult {
	srcGoFiles, err := loadGoFiles(src)
	if err != nil {
		return &diffResult{OK: false, Err: fmt.Sprintf("load go files from %s failed: %v", src, err)}
	}
	dstGoFiles, err := loadGoFiles(dst)
	if err != nil {
		return &diffResult{OK: false, Err: fmt.Sprintf("load go files from %s failed: %v", dst, err)}
	}
	if len(srcGoFiles) != len(dstGoFiles) {
		return &diffResult{OK: false, Err: fmt.Sprintf("go files count not equal: %d vs %d", len(srcGoFiles), len(dstGoFiles))}
	}
	result := diffResult{OK: true, DiffFiles: []string{}}
	for path, srcContent := range srcGoFiles {
		dstContent, ok := dstGoFiles[path]
		if !ok {
			result.OK = false
			result.DiffFiles = append(result.DiffFiles, path)
			continue
		}
		if srcContent != dstContent {
			result.OK = false
			result.DiffFiles = append(result.DiffFiles, path)
		}
	}
	for path := range dstGoFiles {
		if _, ok := srcGoFiles[path]; !ok {
			result.OK = false
			result.DiffFiles = append(result.DiffFiles, path)
		}
	}
	return &result
}

func loadGoFiles(dir string) (map[string]string, error) {
	goFiles := map[string]string{}
	err := filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		} else if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		cnt, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		goFiles[strings.TrimPrefix(path, dir)] = string(cnt)
		return nil
	})
	return goFiles, err
}
