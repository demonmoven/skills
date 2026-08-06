// RUN: pushd $(dirname $(dirname $(pwd)))
// RUN: %cleaner -f
// RUN: cat %s | filecheck %s
package pkg1

import (
	"code.byted.org/analyzers/go_cleaner/test/mono_direct_call/repo1/pkg2"
	"code.byted.org/analyzers/go_cleaner/test/mono_direct_call/repo1/pkg3"
)

var x int = pkg2.JustDoIt(&T{})

type T struct{}

func (t *T) Do() int {
	return -1
}

func UsedFunc() int {
	return x
}

func UnusedFunc() int {
	pkg3.X = 2
	return x
}

func UsedByRepo2() int {
	return x
}

// CHECK: package pkg1
// CHECK: _ "code.byted.org/analyzers/go_cleaner/test/mono_direct_call/repo1/pkg3"
// CHECK: type T struct{}
// CHECK: func UsedFunc() int {
// CHECK-NOT: func UnusedFunc() int {
// CHECK: func UsedByRepo2() int {
