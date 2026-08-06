package pkg2

import (
	"code.byted.org/analyzers/go_cleaner/test/mono_direct_call/repo1/pkg1"
)

func UsedFunc() int {
	return pkg1.UsedByRepo2()
}
