package main

import (
	"github.com/spf13/cobra"

	"code.byted.org/analyzers/go_cleaner/pkg/ab"
)

var (
	clean      = false
	expiredKey []string
	only       []string
	before     = 0
	force      bool

	debug bool

	printInstr bool
)

func main() {

	var rootCmd = &cobra.Command{
		Use:   "ab_cleaner",
		Short: "ab_cleaner is a unused ab branch refactor tool",
		Long:  `Complete documentation is available at https://bytedance.feishu.cn/docx/IuZbdZ1MioLAiVxrbzRcmCGDn6e`,
		Run: func(cmd *cobra.Command, args []string) {
			opts := []ab.Option{ab.WithOutput(), ab.WithBefore(before)}
			if len(expiredKey) > 0 {
				opts = append(opts, ab.WithExpiredKey(expiredKey))
			}
			if clean {
				opts = append(opts, ab.WithClean())
			}
			if len(only) > 0 {
				opts = append(opts, ab.WithOnly(only))
			}
			if force {
				opts = append(opts, ab.WithForce())
			}
			if debug {
				opts = append(opts, ab.WithDebug())
			}
			if printInstr {
				opts = append(opts, ab.WithPrintInstruction())
			}
			ab.Run(opts...)
		},
	}

	rootCmd.PersistentFlags().BoolVarP(&clean, "clean", "c", false, "if set true, will clean the unused AB branch.")
	rootCmd.PersistentFlags().StringSliceVar(&expiredKey, "key", []string{}, "if set, the passed key is regarded as expired key; otherwise, it will get expired key from ab warehouse")
	rootCmd.PersistentFlags().StringSliceVar(&only, "only", []string{}, "only clean codes in the path matching this regex")
	rootCmd.PersistentFlags().IntVar(&before, "before", 6, "only unused ab codes added before the given month would be cleaned")
	rootCmd.PersistentFlags().BoolVarP(&force, "force", "f", false, "if set true, cleaner will not check uncommitted files that may be override")

	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "if set true, all ast expr value will be calculated and printed")
	rootCmd.PersistentFlags().BoolVarP(&printInstr, "instr", "i", false, "if set true, all instructions of functions will be printed")

	//before = 0
	//force = true
	//clean = true
	//os.Chdir("/Users/bytedance/go/src/code.byted.org/xiaoxing.sn/abtest_clean_demo")

	_ = rootCmd.Execute()
}
