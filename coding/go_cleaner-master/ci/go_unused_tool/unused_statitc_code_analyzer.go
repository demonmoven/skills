package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"code.byted.org/analyzers/go_cleaner/pkg/unused"
	analysis "code.byted.org/codebase/analysis-golang-engine"
	"code.byted.org/codebase/analysis-model/enum"
	"code.byted.org/codebase/analysis-model/model"
)

type UnusedStaticCodeAnalyzer struct{}

func NewUnusedStaticCodeAnalyzer() *UnusedStaticCodeAnalyzer {
	return &UnusedStaticCodeAnalyzer{}
}

func (e UnusedStaticCodeAnalyzer) Run(input analysis.Analyzer, repositorySourceDir string) (resIssues model.ResIssues, err error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("panic", r)
		}
	}()

	if e.skip() {
		return
	}

	// output := unused.Run(unused.WithOutput(), unused.WithForce())
	// resIssues = e.genIssue(output)
	return
}

func (e UnusedStaticCodeAnalyzer) ParseRule(analyzer analysis.Analyzer, sourceDir string) ([]analysis.Rule, error) {
	return nil, nil
}

func (e UnusedStaticCodeAnalyzer) genIssue(output *unused.Output) (resIssues model.ResIssues) {
	cwd, _ := filepath.Abs(".")

	for f, name := range output.Unused {
		start := output.FSet.Position(f.Pos())
		end := output.FSet.Position(f.End())
		path, _ := filepath.Rel(cwd, start.Filename)
		resIssues = append(resIssues, &model.ResIssue{
			Key:        "",
			ScanStepID: 0,
			AnalyzerID: 0,
			Location: model.Location{
				Path:    path,
				Range:   model.Range{StartLine: uint(start.Line), StartOffset: uint(start.Offset), EndLine: uint(end.Line), EndOffset: uint(end.Offset)},
				Content: "",
			},
			RuleName:     "byted_unused_code",
			Level:        enum.Info,
			Severity:     enum.Info,
			Category:     enum.Maintenance,
			Message:      fmt.Sprintf("%s is unused, unused/total_unused/total loc: %d/%d/%d", name, end.Line-start.Line+1, output.UnusedLine, output.TotalLine),
			Detail:       "",
			Associations: nil,
			Suggestion: &model.Suggestion{
				Description:  "https://bytedance.feishu.cn/docx/AJ6ldZ4nzo0LNQxzzwIcybxfnGc",
				Replacements: nil,
			},
			Blames: nil,
		})
	}
	return
}

func (e UnusedStaticCodeAnalyzer) skip() bool {
	walk := func(fn ...func(raw [][]byte)) {
		_ = filepath.Walk(".", func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				if strings.Contains(path, "_gen/") {
					return filepath.SkipDir
				}
				return nil
			}
			if filepath.Ext(path) == ".go" && !strings.HasSuffix(path, "_test.go") {
				raw, _ := os.ReadFile(path)
				raws := bytes.Split(raw, []byte{'\n'})
				for _, f := range fn {
					f(raws)
				}
			}
			return nil
		})
	}
	// 大仓库skip
	loc := 0
	locWalk := func(raw [][]byte) {
		loc += len(raw)
	}
	// 公共库skip
	hasMain := false
	pkgPattern := []byte("package main")
	fnPattern := []byte("func main")
	mainWalk := func(raw [][]byte) {
		pkg := false
		fn := false
		for _, v := range raw {
			if bytes.HasPrefix(v, pkgPattern) {
				pkg = true
			}
			if bytes.HasPrefix(v, fnPattern) {
				fn = true
			}
		}
		hasMain = hasMain || (pkg && fn)
	}

	walk(locWalk, mainWalk)

	fmt.Println("loc:", loc)
	fmt.Println("has_main:", hasMain)

	if loc > 150000 {
		return true
	}
	if !hasMain {
		return true
	}

	return false
}
