package comment_retained

import (
	"github.com/spf13/cobra"

	"code.byted.org/analyzers/go_cleaner/pkg/comment_retained"
)

var (
	annotateRetainedCode string
)

func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "comment_retained",
		Short: "cleaner comment_retained",
		Long:  "",
		Run: func(cmd *cobra.Command, args []string) {
			comment_retained.Run(annotateRetainedCode)
		},
	}

	cmd.PersistentFlags().StringVar(&annotateRetainedCode, "annotate_retained_code", "", "")

	return cmd
}
