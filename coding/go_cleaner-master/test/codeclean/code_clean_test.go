package codeclean

import (
	"path/filepath"
	"testing"
)

// func TestNestModuleClean(t *testing.T) {
// 	runCase(t, CleanParams{
// 		CleanerFlag: "-f --before=0",
// 		MeegoID:     testMeegoID,
// 		RepoDir:     filepath.Join(codeCleanTestDir, "nest_module"),
// 		Branch:      "feat/autoTest2",
// 	})
// }

func TestMonorepo1(t *testing.T) {
	checkers := []func(t *testing.T, param CleanParams){
		checkFileContent,
	}

	runCase(t, CleanParams{
		CleanerFlag:  "-f --before=0",
		MeegoID:      testMeegoID,
		RepoDir:      filepath.Join(codeCleanTestDir, "monorepo1"),
		Branch:       "feat/autoTest2",
		MainDir:      "test/codeclean/monorepo1/cmds/cmd1",
		RepoAppMains: "test/codeclean/monorepo1/cmds/cmd1,test/codeclean/monorepo1/cmds/cmd2",
	}, checkers...)
}

func TestMonorepo2(t *testing.T) {
	checkers := []func(t *testing.T, param CleanParams){
		checkFileContent,
		checkFileNotExist("test/codeclean/monorepo2/a.go"),
		checkFileNotExist("test/codeclean/monorepo2/repo2/pkg2/b.go"),
	}

	runCase(t, CleanParams{
		CleanerFlag:  "-f --before=0",
		MeegoID:      testMeegoID,
		RepoDir:      filepath.Join(codeCleanTestDir, "monorepo2"),
		Branch:       "feat/autoTest2",
		MainDir:      "test/codeclean/monorepo2/repo2",
		RepoAppMains: "test/codeclean/monorepo2/repo1,test/codeclean/monorepo2/repo2",
	}, checkers...)
}

func TestCommentRetained(t *testing.T) {
	runCase(t, CleanParams{
		CleanerFlag:          "-f --before=0",
		MeegoID:              testMeegoID,
		RepoDir:              filepath.Join(codeCleanTestDir, "comment_retained"),
		Branch:               "fix/retained",
		AnnotateRetainedCode: "test/codeclean/comment_retained/pkg1/pkg1.go:4,4;8,8;12,12;14,14;22,22",
	})
}
