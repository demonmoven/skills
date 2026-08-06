package internal

import (
	"fmt"
	"go/constant"
	"os/exec"
	"regexp"
	"strings"

	"golang.org/x/tools/go/ssa"
)

func getAnonFun(fn *ssa.Function) (closures []*ssa.Function) {
	for _, f := range fn.AnonFuncs {
		closures = append(closures, f)
		closures = append(closures, getAnonFun(f)...)
	}
	return
}

func mustTrue(vs []constant.Value) bool {
	for _, v := range vs {
		if v == VariantValue {
			return false
		}
		if v.Kind() != constant.Bool || !constant.BoolVal(v) {
			return false
		}
	}
	return true
}

func mustFalse(vs []constant.Value) bool {
	for _, v := range vs {
		if v == VariantValue {
			return false
		}
		if v.Kind() != constant.Bool || constant.BoolVal(v) {
			return false
		}
	}
	return true
}

// combine returns the cross join result from k1,k2......
// joined by '.'
// k1 = ["tiktok"], k2 = ["a", "b"], returns ["tiktok.a", "tiktok.b"]
func combine(k1 []constant.Value, k2 ...[]constant.Value) ([]constant.Value, bool) {
	var k []constant.Value
	if len(k2) == 0 {
		k = append(k, k1...)
		return k, true
	}

	for _, v := range k1 {
		if !isString(v) {
			return []constant.Value{VariantValue}, false
		}
		for _, vv := range k2[0] {
			if !isString(vv) {
				return []constant.Value{VariantValue}, false
			}
			k = append(k, constant.MakeString(constant.StringVal(v)+"."+constant.StringVal(vv)))
		}
	}

	return combine(k, k2[1:]...)
}

func isString(vv constant.Value) bool {
	if vv == VariantValue {
		return false
	}
	if vv.Kind() != constant.String {
		return false
	}
	s := constant.StringVal(vv)
	if s == "" {
		return false
	}
	return true
}

func genRe(path []string) *regexp.Regexp {
	var re []string
	for i := range path {
		re = append(re, fmt.Sprintf("(%s)", path[i]))
	}
	return regexp.MustCompile(strings.Join(re, "|"))
}

// MustHasGitAndAllCommitted checks if there are any uncommitted changes in the git repository at the given directory.
func MustHasGitAndAllCommitted(dir string) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = dir

	output, err := cmd.Output()
	if err != nil {
		panic(err)
	}

	if len(strings.TrimSpace(string(output))) != 0 {
		panic("please commit modified files first")
	}
}

const ModifiedFunctionTrackerFilePath = "~/.ab_cleaner_tracker_data"

type ModifiedFunctionTracker struct {
	funcName map[string]bool
}

func NewModifiedFunctionTracker() *ModifiedFunctionTracker {
	t := &ModifiedFunctionTracker{funcName: map[string]bool{}}
	return t
}

func (t *ModifiedFunctionTracker) Add(f *ssa.Function) {
	if f == nil {
		return
	}
	t.funcName[f.String()] = true
	for _, anno := range f.AnonFuncs {
		t.Add(anno)
	}
}

func (t *ModifiedFunctionTracker) Has(f *ssa.Function) bool {
	if f == nil {
		return false
	}
	return t.funcName[f.String()]
}
