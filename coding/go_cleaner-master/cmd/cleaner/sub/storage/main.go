package storage

import (
	"github.com/spf13/cobra"

	"code.byted.org/analyzers/go_cleaner/pkg/storage"
)

func Cmd() *cobra.Command {
	config := &storage.StorageConfig{
		IgnoreTestErr:                  true,
		MaxNumsOfReloadingWithTestErrs: 3,
	}

	cmd := &cobra.Command{
		Use:   "storage",
		Short: "storage cleaner",
		Long:  "",
		Run: func(cmd *cobra.Command, args []string) {
			storage.Run(config)
		},
	}

	cmd.PersistentFlags().StringSliceVar(&config.PSMs, "psm-list", []string{}, "the list of storage PSM to clean")

	return cmd
}
