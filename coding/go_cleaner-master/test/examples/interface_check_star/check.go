// RUN: %cleaner -f
// RUN: cat %s | %filecheck %s
package main

import (
	"fmt"

	"code.byted.org/analyzers/go_cleaner/test/interface_check_ref/pkg"
)

type K struct {
	V int
}

func (k *K) Method() {}

var _ pkg.T = (*K)(nil)

func main() {
	k := &K{}
	fmt.Println(k.V)
}

// CHECK: "code.byted.org/analyzers/go_cleaner/test/interface_check_ref/pkg"
