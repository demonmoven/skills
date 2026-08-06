package searchimport

import (
	"github.com/spf13/cobra"

	"code.byted.org/analyzers/go_cleaner/pkg/searchimport"
)

var (
	targetModule string
	outFile      string
	method       string
)

func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search",
		Short: "search everything of a module used in this project",
		Long:  `Complete documentation is available at https://bytedance.feishu.cn/docx/AJ6ldZ4nzo0LNQxzzwIcybxfnGc`,
		Run: func(cmd *cobra.Command, args []string) {
			var opts []searchimport.Option
			if targetModule != "" {
				opts = append(opts, searchimport.WithTargetModule(targetModule))
			}
			if outFile != "" {
				opts = append(opts, searchimport.WithOutFile(outFile))
			}
			if method != "" {
				opts = append(opts, searchimport.WithMethod(method))
			}
			searchimport.Run(opts...)
		},
	}

	cmd.PersistentFlags().StringVar(&targetModule, "target", "fmt", "search all usages of this module in this project")
	cmd.PersistentFlags().StringVar(&outFile, "out", "", "output file")
	cmd.PersistentFlags().StringVar(&method, "method", "", "if set to `ast`, search usages based on ast without parsing the project")

	return cmd
}
