package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"code.byted.org/analyzers/go_cleaner/cmd/cleaner/sub/comment_retained"
	"code.byted.org/analyzers/go_cleaner/cmd/cleaner/sub/endpoint"
	"code.byted.org/analyzers/go_cleaner/cmd/cleaner/sub/scope"
	"code.byted.org/analyzers/go_cleaner/cmd/cleaner/sub/searchimport"
	"code.byted.org/analyzers/go_cleaner/cmd/cleaner/sub/storage"
	"code.byted.org/analyzers/go_cleaner/pkg/core/imports"
	"code.byted.org/analyzers/go_cleaner/pkg/core/logger"
	"code.byted.org/analyzers/go_cleaner/pkg/core/passes"
	"code.byted.org/analyzers/go_cleaner/pkg/unused"
)

var (
	testMode string
	tag      []string

	excludePath []string
	onlyPath    []string
	scopeFile   string
	mainDir     string

	force bool

	before int

	similarConstKeep    bool
	adjacentConstKeep   bool
	similarConstComment bool
	similarFuncKeep     bool

	deleteUnusedGlobalAnonymousV bool

	unsafeDeleteMethod bool

	ignoreTestErr     bool
	ignoreCompileErrs bool

	callCallGraph bool

	extraImportSource string
	buildFlags        []string

	timeout       int
	outputChanges bool
	passNames     []string
)

func main() {
	logger.Init(logrus.InfoLevel)

	var rootCmd = &cobra.Command{
		Use:   "cleaner",
		Short: "cleaner is a dead code detection and auto-clean tool",
		Long:  `Complete documentation is available at https://bytedance.feishu.cn/docx/AJ6ldZ4nzo0LNQxzzwIcybxfnGc`,
		Run: func(cmd *cobra.Command, args []string) {
			opts := []unused.Option{unused.WithClean()}
			if scopeFile != "" && len(onlyPath) == 0 {
				cnt, err := os.ReadFile(scopeFile)
				if err == nil {
					onlyPath = strings.Split(string(cnt), "\n")
				} else if !os.IsNotExist(err) {
					fmt.Printf("failed read scope file(%s): %v\n", scopeFile, err)
				}
			}
			if testMode != "" {
				opts = append(opts, unused.WithTestMode(testMode))
			}
			if len(tag) > 0 {
				opts = append(opts, unused.WithTag(tag))
			}
			if len(excludePath) > 0 {
				opts = append(opts, unused.WithExclude(excludePath))
			}
			if len(onlyPath) > 0 {
				opts = append(opts, unused.WithOnly(onlyPath))
			}
			if len(buildFlags) > 0 {
				opts = append(opts, unused.WithBuildFlags(buildFlags))
			}
			if force {
				opts = append(opts, unused.WithForce())
			}
			if before >= 0 {
				opts = append(opts, unused.WithBefore(before))
			}
			if similarConstKeep {
				opts = append(opts, unused.WithSimilarConstKeep())
			}
			if adjacentConstKeep {
				opts = append(opts, unused.WithAdjacentConstKeep())
			}
			if similarConstComment {
				opts = append(opts, unused.WithSimilarConstComment())
			}
			if similarFuncKeep {
				opts = append(opts, unused.WithSimilarFuncKeep())
			}
			if unsafeDeleteMethod {
				opts = append(opts, unused.WithUnsafeDeleteMethod())
			}
			if callCallGraph {
				opts = append(opts, unused.WithCalCallGraph())
			}
			if deleteUnusedGlobalAnonymousV {
				opts = append(opts, unused.WithDeleteUnusedGlobalAnonymousV())
			}
			if ignoreTestErr {
				opts = append(opts, unused.WithIgnoreTestErr())
			}
			if ignoreCompileErrs {
				opts = append(opts, unused.WithIgnoreCompileErrs())
			}
			if extraImportSource != "" {
				opts = append(opts, unused.WithImportBy(extraImportSource))
			}
			if outputChanges {
				opts = append(opts, unused.WithOutputChanges(outputChanges))
			}
			if len(passNames) > 0 {
				opts = append(opts, unused.WithEnabledPassNames(passNames))
			}

			logrus.Infof("got main dir: %s", mainDir)
			if mainDir != "" {

				opts = append(opts, unused.WithMainDir(mainDir))
			}

			unused.Run(opts...)
		},
	}

	rootCmd.PersistentFlags().StringSliceVar(&tag, "tag", []string{"unittest"}, "tags for test-only files if has")
	rootCmd.PersistentFlags().StringVar(&testMode, "test_mode", "auto", "if set auto, unused code including UTs will be cleaned; if set full, UTs and dependent codes are all kept; if set none, UTs are neglected")
	rootCmd.PersistentFlags().StringSliceVar(&excludePath, "exclude", []string{"kitex_gen/", "model_gen/", "thrift_gen/", "mock_gen/", "rpc_gen/", "wcc/"}, "dont clean codes in the path matching this regex")
	rootCmd.PersistentFlags().StringSliceVar(&onlyPath, "only", []string{}, "only clean codes in the path matching this regex")
	rootCmd.PersistentFlags().BoolVarP(&force, "force", "f", false, "if set true, cleaner will not check uncommitted files that may be override")
	rootCmd.PersistentFlags().IntVar(&before, "before", 0, "only functions or objects latest updated before this month will be cleaned")
	rootCmd.PersistentFlags().BoolVar(&similarConstKeep, "similar_const_keep", true, "if set true, similar and close const declaration will not be cleaned singly")
	rootCmd.PersistentFlags().BoolVar(&adjacentConstKeep, "adjacent_const_keep", false, "if set true, adjacent const declaration will not be cleaned singly")
	rootCmd.PersistentFlags().BoolVar(&similarConstComment, "similar_const_comment", true, "if set true, similar and close const declaration will be commented instead of deleted, the similar determining method is less strict than similar_const_keep")
	rootCmd.PersistentFlags().BoolVar(&similarFuncKeep, "similar_func", false, "if set true, similar and close func declaration will not be cleaned singly")
	rootCmd.PersistentFlags().BoolVar(&deleteUnusedGlobalAnonymousV, "delete_unused_anon_v", true, "if set true, `var _ I = (*X)(nil)` will be automatically deleted if I or X is deleted")
	rootCmd.PersistentFlags().BoolVar(&unsafeDeleteMethod, "unsafe_delete_method", false, "if set true, uncalled methods will be cleaned, which may cause error")
	rootCmd.PersistentFlags().BoolVar(&ignoreTestErr, "ignore_test_err", true, "if set true, test files with syntax error will be ignored")
	rootCmd.PersistentFlags().BoolVar(&callCallGraph, "call_graph", false, "if set true, call graph will be build using vta")
	rootCmd.PersistentFlags().StringVar(&extraImportSource, "extra_import", "", "everything declared in this file will be kept, the file is always generated by cleaner search --module=x --out=path")
	rootCmd.PersistentFlags().BoolVar(&outputChanges, "output_changes", false, "if set true, logger will be disabled and source changes would be printed to stdout")
	rootCmd.PersistentFlags().StringSliceVar(&passNames, "passes", passes.GetDefaultPassNames(), "developer option: enabled passes for running. The default setting is to run all passes.")
	rootCmd.PersistentFlags().StringSliceVar(&buildFlags, "build_flags", []string{}, "build flags for go build, default is none")
	rootCmd.PersistentFlags().BoolVar(&ignoreCompileErrs, "ignore_errors", false, "if set true, compile errors will be ignored")
	rootCmd.PersistentFlags().StringVar(&scopeFile, "scope_file", "", "scope file for scope cleaning")
	rootCmd.PersistentFlags().StringVar(&mainDir, "main-dir", "", "main dir of the project")
	rootCmd.PersistentFlags().BoolVar(&unused.Config.DisableFallback, "disable_fallback", false, "disable fallback")
	rootCmd.PersistentFlags().StringSliceVar(&imports.PkgsNoSideEffets, "pkgs_no_side_effects", []string{}, "cleaner will assume those packages have no side effects")

	//只做解析，实际不使用
	rootCmd.PersistentFlags().IntVar(&timeout, "timeout", 0, "timeout in minutes to run go list, default is 30")

	os.Setenv("GOARCH", "amd64")
	os.Setenv("CGO_ENABLED", "1")

	rootCmd.AddCommand(
		endpoint.Cmd(),
		searchimport.Cmd(),
		comment_retained.Cmd(),
		scope.Cmd(),
		storage.Cmd(),
	)

	err := rootCmd.Execute()
	if err != nil {
		fmt.Println("fail, err:", err)
		os.Exit(1)
	}
}
