package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	_ "embed"

	"code.byted.org/analyzers/go_cleaner/pkg/idl_cleaner"
	"code.byted.org/analyzers/go_cleaner/pkg/idl_cleaner/proto"
	"code.byted.org/analyzers/go_cleaner/pkg/idl_cleaner/thrift"
	"code.byted.org/analyzers/go_cleaner/pkg/out"
	"code.byted.org/analyzers/go_cleaner/pkg/util"
	"github.com/spf13/cobra"
)

const (
	thriftFileSuffix   = ".thrift"
	protobufFileSuffix = ".proto"
)

func run(cmd *cobra.Command, _ []string) {
	if SCMVersion != "" {
		out.Std.SetKV("cleaner_version", fmt.Sprintf("idl_cleaner@%s(%s)", SCMVersion, CommitHash))
	} else {
		out.Std.SetKV("cleaner_version", "idl_cleaner@"+util.GetVersion())
	}

	// 检查必须入参
	checkMustFlag()
	flow := out.Std.Procedure("clone-repo")
	// 下载idl仓库，同时从source-branch切出分支到commit-branch
	repoDir := prepareRunEnv()
	flow.Next("clean-idl")
	if repoDir == "" {
		out.Std.Fatalf("IDL仓库(%s)下载失败，请检测错误信息", params.IDLPath)
	}
	fmt.Println("repoDir:", repoDir)
	out.Std.SetOnExit(func() {
		if os.Getenv("KEEP_TMP") != "1" {
			err := os.RemoveAll(repoDir)
			if err != nil {
				fmt.Printf("remove tmp dir failed: %s\n", err.Error())
			} else {
				fmt.Println("remove tmp dir success")
			}
		} else {
			fmt.Println("keep tmp dir", repoDir)
		}
	})

	cleanFn := proto.Clean
	// 执行idl清理
	if strings.HasSuffix(params.IDLPath, thriftFileSuffix) {
		cleanFn = thrift.Clean
	}
	result := cleanFn(&idl_cleaner.Params{
		Root:    repoDir,
		File:    params.IDLPath,
		Methods: params.Methods,
		Remove:  params.Remove,
	})
	flow.Next("commit-change")
	if len(result.RemovedMethods) == 0 {
		out.Std.Fatalf("未能发现需要清理的方法")
		return
	}
	if result.Err != nil {
		out.Std.Fatalf("idl清理失败: %s", result.Err.Error())
	}
	out.Std.SetKV("removed_methods", strings.Join(result.RemovedMethods, ","))
	// 统计删除的行数
	statRemoveLines(repoDir)
	// 提交代码
	commitChange(repoDir)
	flow.Done()
}

func checkMustFlag() {
	missMustFlag := []string{}
	if params.RepoName == "" {
		missMustFlag = append(missMustFlag, "repo")
	}
	if params.SourceBranch == "" {
		missMustFlag = append(missMustFlag, "source-branch")
	}
	if params.CommitBranch == "" {
		missMustFlag = append(missMustFlag, "commit-branch")
	}
	if params.IDLPath == "" {
		missMustFlag = append(missMustFlag, "idl-path")
	}
	if len(params.Methods) == 0 {
		missMustFlag = append(missMustFlag, "methods")
	}
	if len(missMustFlag) > 0 {
		out.Std.Fatalf("缺少必须入参: %s", strings.Join(missMustFlag, ", "))
	} else if !strings.HasSuffix(params.IDLPath, thriftFileSuffix) && !strings.HasSuffix(params.IDLPath, protobufFileSuffix) {
		out.Std.Fatalf("不支持的idl(%s), 仅支持thrift和proto", params.IDLPath)
	}
}

func prepareRunEnv() string {
	randDir := "idl_cleaner_" + util.RandString(6)
	// git@code.byted.org:webcast/qin_manager.git
	cloneRepo := exec.Command("git", "clone", "--single-branch", "-b", params.SourceBranch, fmt.Sprintf("git@code.byted.org:%s.git", params.RepoName), randDir)
	cloneRepo.Dir = os.TempDir()
	outBuf, errBuf := strings.Builder{}, strings.Builder{}
	cloneRepo.Stdout = &outBuf
	cloneRepo.Stderr = &errBuf
	if err := cloneRepo.Run(); err != nil {
		fmt.Printf("\n%s\n>>> git clone --single-branch -b %s git@code.byted.org:%s.git %s\n",
			strings.Repeat(">", 50), params.SourceBranch, params.RepoName, randDir)
		fmt.Printf("==== stdout ====\n%s\n", outBuf.String())
		fmt.Printf("==== stderr ====\n%s\n%s\n", errBuf.String(), strings.Repeat(">", 50))
		out.Std.Fatalf("代码下载失败: %s", errBuf.String())
		return ""
	}

	checkBranch := exec.Command("git", "checkout", "-b", params.CommitBranch)
	checkBranch.Dir = filepath.Join(os.TempDir(), randDir)
	outBuf, errBuf = strings.Builder{}, strings.Builder{}
	checkBranch.Stdout = &outBuf
	checkBranch.Stderr = &errBuf
	if err := checkBranch.Run(); err != nil {
		fmt.Printf("\n%s\n>>> git checkout -b %s\n",
			strings.Repeat(">", 50), params.CommitBranch)
		fmt.Printf("==== stdout ====\n%s\n", outBuf.String())
		fmt.Printf("==== stderr ====\n%s\n%s\n", errBuf.String(), strings.Repeat(">", 50))
		out.Std.Fatalf("分支切换失败: %s", errBuf.String())
		return ""
	}

	getHeadCommitID := exec.Command("git", "rev-parse", "HEAD")
	getHeadCommitID.Dir = filepath.Join(os.TempDir(), randDir)
	outBuf, errBuf = strings.Builder{}, strings.Builder{}
	getHeadCommitID.Stdout = &outBuf
	getHeadCommitID.Stderr = &errBuf
	if err := getHeadCommitID.Run(); err != nil {
		fmt.Printf("\n%s\n>>> git rev-parse HEAD\n",
			strings.Repeat(">", 50))
		fmt.Printf("==== stdout ====\n%s\n", outBuf.String())
		fmt.Printf("==== stderr ====\n%s\n%s\n", errBuf.String(), strings.Repeat(">", 50))
		out.Std.SetWarnf("获取base commit id失败: %s", errBuf.String())
	} else {
		out.Std.SetKV("base_commit_id", strings.TrimSpace(outBuf.String()))
	}
	return filepath.Join(os.TempDir(), randDir)
}

func statRemoveLines(repoDir string) {
	diffCmd := exec.Command("git", "diff", "--numstat")
	diffCmd.Dir = repoDir
	outBuf, errBuf := strings.Builder{}, strings.Builder{}
	diffCmd.Stdout = &outBuf
	diffCmd.Stderr = &errBuf
	fmt.Printf("\n%s\n>>> git diff --numstat\n",
		strings.Repeat(">", 50))
	if err := diffCmd.Run(); err != nil {
		fmt.Printf("==== stdout ====\n%s\n", outBuf.String())
		fmt.Printf("==== stderr ====\n%s\n%s\n", errBuf.String(), strings.Repeat(">", 50))
		out.Std.SetWarnf("代码变更统计失败: %s", errBuf.String())
	} else {
		fmt.Printf("==== stdout ====\n%s\n%s\n", outBuf.String(), strings.Repeat(">", 50))
	}
	output := strings.TrimSpace(outBuf.String())
	if output == "" {
		out.Std.SetKV("removed_line_count", "0")
		return
	}
	lines := strings.Split(output, "\n")
	addLines, delLines := 0, 0
	for _, line := range lines {
		if line == "" {
			continue
		}
		segs := strings.Split(line, "\t")
		if len(segs) != 3 {
			continue
		}
		add, err := strconv.Atoi(segs[0])
		if err != nil {
			out.Std.SetWarnf("代码变更统计失败: %s", line)
			return
		}
		del, err := strconv.Atoi(segs[1])
		if err != nil {
			out.Std.SetWarnf("代码变更统计失败: %s", line)
			return
		}
		addLines += add
		delLines += del
	}
	out.Std.SetKV("removed_line_count", strconv.Itoa(delLines-addLines))
}

func commitChange(repoDir string) {
	commitCmd := exec.Command("bash", "-c", fmt.Sprintf(
		"git add . && git commit -m 'chore: [冗余代码清理平台-task]clean unused code' && git push -f origin %s", params.CommitBranch,
	))
	commitCmd.Dir = repoDir
	outBuf, errBuf := strings.Builder{}, strings.Builder{}
	commitCmd.Stdout = &outBuf
	commitCmd.Stderr = &errBuf
	if err := commitCmd.Run(); err != nil {
		fmt.Printf("\n%s\n>>> git add . && git commit -m 'chore: [冗余代码清理平台]clean unused code' && git push -f origin %s\n",
			strings.Repeat(">", 50), params.CommitBranch)
		fmt.Printf("==== stdout ====\n%s\n", outBuf.String())
		fmt.Printf("==== stderr ====\n%s\n%s\n", errBuf.String(), strings.Repeat(">", 50))
		out.Std.Fatalf("变更提交失败: %s", errBuf.String())
	}
}
