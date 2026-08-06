package endpoint

import (
	"github.com/spf13/cobra"

	"code.byted.org/analyzers/go_cleaner/pkg/endpoint"
	"code.byted.org/analyzers/go_cleaner/pkg/out"
)

var (
	eps         []string
	rm          bool
	handlerPath string
	appRoot     string
)

func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "endpoint",
		Short: "cleaner endpoint delete unused endpoints",
		Long:  `Complete documentation is available at https://bytedance.feishu.cn/docx/AJ6ldZ4nzo0LNQxzzwIcybxfnGc`,
		Run: func(cmd *cobra.Command, args []string) {
			var opts []endpoint.Option
			if rm {
				opts = append(opts, endpoint.WithRM())
			}
			if handlerPath != "" {
				opts = append(opts, endpoint.WithHandlerPath(handlerPath))
			}
			if appRoot != "" {
				opts = append(opts, endpoint.WithAppRoot(appRoot))
			}
			err := endpoint.Run(eps, opts...)
			if err != nil {
				out.Std.Fatal(err)
			}
		},
	}

	cmd.PersistentFlags().StringSliceVar(&eps, "ep", []string{}, "unused endpoint name or url list")
	cmd.PersistentFlags().BoolVarP(&rm, "delete", "d", false, "if set true, passed unused endpoint codes will be removed, otherwise, they will be substituted by empty implementation")
	cmd.PersistentFlags().StringVar(&handlerPath, "handler_path", "", "")
	cmd.PersistentFlags().StringVar(&appRoot, "main-dir", "", "the **directory** of main.go")

	return cmd
}
