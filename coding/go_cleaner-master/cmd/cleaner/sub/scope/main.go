package scope

import (
	"os"
	"path/filepath"
	"strings"

	"code.byted.org/analyzers/go_cleaner/pkg/out"
	"code.byted.org/analyzers/go_cleaner/pkg/util"
	"github.com/spf13/cobra"
)

var (
	mainDir      string   // 程序入口目录
	repoAppMains []string // 代码仓库下包括的PSM程序入口
	output       string   // 输出文件
	cleanerFlag  string   // 清理器参数
)

func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scope",
		Short: "scope",
		Long:  "scope",
		Run: func(cmd *cobra.Command, args []string) {
			if output == "" {
				out.Fatalf("scope|output-empty", "output file is empty")
			}
			if mainDir == "" {
				return // 正常至少有个 . ，什么都没有的认为是不用处理
			} else if mainDir == "." || len(repoAppMains) == 1 {
				// main.go在仓库根目录下的，全仓可清理
				// 全仓关联psm的main只有一个的话，认为费monorepo，跳过清理
				return
			}
			repoRoot, err := util.GetWorkingRepositoryRoot()
			if err != nil {
				out.Fatal("scope|not-in-git-repo", err.Error())
			}
			if _, err := os.Stat(filepath.Join(repoRoot, mainDir)); err != nil && os.IsNotExist(err) {
				out.Warnf("当前仓库的线上产物main目录(%s)不存在，跳过项目清理范围分析", mainDir)
				return
			} else if err != nil {
				out.Fatal("scope|stat-main-dir-failed", err.Error())
			}
			repoAppMains, ok := filterMainDirs(repoRoot, repoAppMains)
			if !ok { // 处理出现异常，不圈定范围，全仓可清理
				return
			}
			if len(repoAppMains) == 1 {
				out.Warnf("当前仓库下有效的main目录只有一个(%s)，判定为非monorepo仓库，跳过项目清理范围分析", repoAppMains[0])
				return // 仓库下的psm就一个，判定为非monorepo仓库，不进行分析
			}
			s := scoper{
				repoAppMains: repoAppMains,
				mainDir:      mainDir,
				repoRoot:     repoRoot,
			}
			scopeDirs, err := s.Eval()
			if err != nil {
				out.Fatal("scope|eval-failed", err.Error())
			}
			if err := os.WriteFile(output, []byte(strings.Join(scopeDirs, "\n")), 0644); err != nil {
				out.Fatal("scope|write-file-failed", err.Error())
			}
		},
	}
	cmd.PersistentFlags().StringVar(&mainDir, "main-dir", "", "the **directory** of main.go")
	cmd.PersistentFlags().StringSliceVar(&repoAppMains, "repo-app-mains", []string{}, "the main pkgs of psm apps in current repo")
	cmd.PersistentFlags().StringVar(&output, "output", "", "the output file")
	cmd.PersistentFlags().StringVar(&cleanerFlag, "cleaner-flag", "", "the cleaner flag")
	return cmd
}

func filterMainDirs(repoRoot string, mainDirs []string) ([]string, bool) {
	filtered := []string{}
	for _, mainDir := range mainDirs {
		if mainDir == "." {
			continue
		}
		absDir := filepath.Join(repoRoot, mainDir)
		if _, err := os.Stat(absDir); err == nil {
			filtered = append(filtered, mainDir)
		} else if os.IsNotExist(err) {
			continue
		} else {
			return nil, false
		}
	}
	return filtered, true
}
