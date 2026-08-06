package main

import (
	"github.com/spf13/cobra"
)

var (
	params Params

	SCMVersion string
	BuildUser  string
	Branch     string
	DetailURL  string
	CommitHash string
)

type Params struct {
	RepoName     string
	Methods      []string
	IDLPath      string
	SourceBranch string
	CommitBranch string
	Remove       bool
}

func main() {
	var rootCmd = &cobra.Command{
		Use:   "idl_cleaner",
		Short: "idl_cleaner is tool to clean psm idl methods",
		Long:  `Complete documentation is available at https://bytedance.feishu.cn/docx/IuZbdZ1MioLAiVxrbzRcmCGDn6e`,
		Run:   run,
	}

	rootCmd.PersistentFlags().StringVarP(&params.RepoName, "repo", "", "", "the psm to clean")
	rootCmd.PersistentFlags().StringVarP(&params.SourceBranch, "source-branch", "s", "", "the source branch")
	rootCmd.PersistentFlags().StringVarP(&params.CommitBranch, "commit-branch", "c", "", "the commit branch")
	rootCmd.PersistentFlags().BoolVarP(&params.Remove, "remove", "", false, "remove the methods")
	rootCmd.PersistentFlags().StringVarP(&params.IDLPath, "idl-path", "", "", "the idl file path relative to the repository root")
	rootCmd.PersistentFlags().StringSliceVar(&params.Methods, "methods", nil, "the methods to clean")
	_ = rootCmd.Execute()
}
