package proto

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"code.byted.org/analyzers/go_cleaner/pkg/idl_cleaner"
)

func TestProto(t *testing.T) {
	repoRoot := getRepoRoot()
	t.Log(repoRoot)
	t.Run("Remove", func(t *testing.T) {
		result := Clean(&idl_cleaner.Params{
			Root:    repoRoot,
			File:    "test/idls/service_idl.proto",
			Write:   writeFileWithSuffix(".expt.remove"),
			Methods: []string{"/cleaner/test/api2/", "/cleaner/test/api4_post/", "/cleaner/test/api5/"},
			Remove:  true,
		})
		if result.Err != nil {
			t.Fatal(result.Err)
		}
	})
	t.Run("Comment", func(t *testing.T) {
		result := Clean(&idl_cleaner.Params{
			Root:    repoRoot,
			File:    "test/idls/service_idl.proto",
			Write:   writeFileWithSuffix(".expt.comment"),
			Methods: []string{"/cleaner/test/api2/", "/cleaner/test/api4_post/", "/cleaner/test/api5/"},
			Remove:  false,
		})
		if result.Err != nil {
			t.Fatal(result.Err)
		}
	})
}

func getRepoRoot() string {
	wd, _ := os.Getwd()
	abs, _ := filepath.Abs(filepath.Join(wd, "../../.."))
	return abs
}

func writeFileWithSuffix(suffix string) func(file string, cnt []byte) error {
	return func(file string, cnt []byte) error {
		if !strings.HasSuffix(suffix, ".proto") {
			suffix += ".proto"
		}
		return os.WriteFile(strings.TrimSuffix(file, ".proto")+suffix, cnt, 0644)
	}
}
