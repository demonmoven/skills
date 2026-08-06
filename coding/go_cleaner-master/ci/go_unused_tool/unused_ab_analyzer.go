package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"code.byted.org/analyzers/go_cleaner/pkg/ab"
	analysis "code.byted.org/codebase/analysis-golang-engine"
	"code.byted.org/codebase/analysis-model/enum"
	"code.byted.org/codebase/analysis-model/model"
)

type UnusedABCodeAnalyzer struct{}

func NewUnusedABCodeAnalyzer() *UnusedABCodeAnalyzer {
	return &UnusedABCodeAnalyzer{}
}

func (e UnusedABCodeAnalyzer) Run(input analysis.Analyzer, repositorySourceDir string) (resIssues model.ResIssues, err error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("panic", r)
		}
	}()

	if e.skip() {
		return
	}

	output := ab.Run(ab.WithOutput(), ab.WithForce(), ab.WithEntry(entry), ab.WithBefore(1))
	resIssues = e.genIssue(output)
	return
}

func (e UnusedABCodeAnalyzer) ParseRule(analyzer analysis.Analyzer, sourceDir string) ([]analysis.Rule, error) {
	return nil, nil
}

func (e UnusedABCodeAnalyzer) genIssue(output []*ab.Output) (resIssues model.ResIssues) {
	for _, o := range output {
		resIssues = append(resIssues, &model.ResIssue{
			Key:        "",
			ScanStepID: 0,
			AnalyzerID: 0,
			Location: model.Location{
				Path:    o.Pos.Filename,
				Range:   model.Range{StartLine: uint(o.Pos.Line), StartOffset: uint(o.Pos.Offset), EndLine: uint(o.Pos.Line), EndOffset: uint(o.Pos.Offset)},
				Content: "",
			},
			RuleName:     "byted_expired_ab_key",
			Level:        enum.Info,
			Severity:     enum.Info,
			Category:     enum.Maintenance,
			Message:      fmt.Sprintf("ab key %s is expired", o.Key),
			Detail:       "",
			Associations: nil,
			Suggestion: &model.Suggestion{
				Description:  "https://bytedance.larkoffice.com/docx/IuZbdZ1MioLAiVxrbzRcmCGDn6e",
				Replacements: nil,
			},
			Blames: nil,
		})
	}
	return
}

func (e UnusedABCodeAnalyzer) skip() bool {
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

	walk(locWalk)

	fmt.Println("loc:", loc)

	if loc > 150000 {
		return true
	}

	return false
}

var entry = []*ab.Entry{
	{
		ID:           592,
		Package:      "code.byted.org/tiktok/post_api/pkg/ab",
		Recv:         "",
		Func:         []string{"PackFromAbParams"},
		KeyIndex:     []int{1},
		DefaultIndex: 2,
		KeyPrefix:    "tiktok",
		DefaultVal:   "",
		DefaultTyp:   "",
	},

	{
		ID:           591,
		Package:      "code.byted.org/iesarch/abtest/ab",
		Recv:         "AbTest",
		Func:         []string{"GetBoolV", "GetIntV", "GetStringV", "GetNumberV"},
		KeyIndex:     []int{1},
		DefaultIndex: 2,
		KeyPrefix:    "",
		DefaultVal:   "",
		DefaultTyp:   "",
	},

	{
		ID:           23,
		Package:      "code.byted.org/tiktok/pack_material/pack_base/base_data",
		Recv:         "AppContext",
		Func:         []string{"GetMergeAbParamStringDefault", "GetMergeAbParamBoolDefault", "GetMergeAbParamInt64Default", "GetMergeAbParamInt32Default", "GetMergeAbParamFloat64Default"},
		KeyIndex:     []int{1, 2},
		DefaultIndex: 3,
		KeyPrefix:    "",
		DefaultVal:   "",
		DefaultTyp:   "",
	},

	{
		ID:           22,
		Package:      "code.byted.org/tiktok/pack_material/pack_base/base_data",
		Recv:         "AppContext",
		Func:         []string{"GetAbClientParamDefault", "GetAbServerParamDefault", "GetMergeAbParamDefault"},
		KeyIndex:     []int{1, 2},
		DefaultIndex: 4,
		KeyPrefix:    "",
		DefaultVal:   "",
		DefaultTyp:   "",
	},

	{
		ID:           21,
		Package:      "code.byted.org/tiktok/pack_engine/helper",
		Recv:         "",
		Func:         []string{"GetOtherAbParam"},
		KeyIndex:     []int{1, 2},
		DefaultIndex: 0,
		KeyPrefix:    "",
		DefaultVal:   "0",
		DefaultTyp:   "int",
	},

	{
		ID:           20,
		Package:      "code.byted.org/tiktok/pack_engine/helper",
		Recv:         "",
		Func:         []string{"GetAbParam"},
		KeyIndex:     []int{1},
		DefaultIndex: 3,
		KeyPrefix:    "tiktok",
		DefaultVal:   "",
		DefaultTyp:   "",
	},

	{
		ID:           19,
		Package:      "code.byted.org/tiktok/pack_engine/helper",
		Recv:         "",
		Func:         []string{"GetAbClientParamDefault", "GetAbServerParamDefault", "GetMergeAbParamDefault", "GetMergeAbParamWithDefaultVal", "GetRawAbParamWithDefaultVal", "GetAbClientParamWithDefaultVal"},
		KeyIndex:     []int{1, 2},
		DefaultIndex: 4,
		KeyPrefix:    "",
		DefaultVal:   "",
		DefaultTyp:   "",
	},

	{
		ID:           18,
		Package:      "code.byted.org/tiktok/pack_engine/helper",
		Recv:         "",
		Func:         []string{"GetMergeAbParamStringDefault", "GetMergeAbParamBoolDefault", "GetMergeAbParamInt64Default", "GetMergeAbParamInt32Default", "GetMergeAbParamFloat64Default"},
		KeyIndex:     []int{1, 2},
		DefaultIndex: 3,
		KeyPrefix:    "",
		DefaultVal:   "",
		DefaultTyp:   "",
	},
}
