package main

import (
	"fmt"
	"os"
	"path/filepath"

	analysis "code.byted.org/codebase/analysis-golang-engine"
	"code.byted.org/codebase/analysis-model/model"
)

type GoUnusedCodeAnalyzer struct {
	name string

	checkers map[string]analysis.AnalyzerImpl
}

func NewGoUnusedCodeAnalyzer() GoUnusedCodeAnalyzer {
	return GoUnusedCodeAnalyzer{
		name: "go-unused-code",
		checkers: map[string]analysis.AnalyzerImpl{
			"byted_unused_code":    NewUnusedStaticCodeAnalyzer(),
			"byted_expired_ab_key": NewUnusedABCodeAnalyzer(),
		}}
}

func (e GoUnusedCodeAnalyzer) Run(input analysis.Analyzer, repositorySourceDir string) (resIssues model.ResIssues, err error) {
	if os.Getenv("SCAN_TYPE") != "full" {
		return
	}

	cwd, _ := filepath.Abs(".")
	_ = os.Chdir(repositorySourceDir)
	defer func() {
		_ = os.Chdir(cwd)
	}()

	if _, err = os.Stat("./go.mod"); err != nil {
		fmt.Println("no go.mod, skip")
		return resIssues, nil
	}

	rules := input.Rules
	fmt.Printf("all rules: %v\n", rules)

	for _, rule := range rules {
		if checker, ok := e.checkers[rule.Name]; ok && checker != nil && !bool(rule.Disabled) {
			issues, err := checker.Run(input, repositorySourceDir)
			if err != nil {
				return resIssues, err
			}
			resIssues = append(resIssues, issues...)
		}
	}
	return
}

func (e GoUnusedCodeAnalyzer) ParseRule(analyzer analysis.Analyzer, sourceDir string) ([]analysis.Rule, error) {
	return nil, nil
}
