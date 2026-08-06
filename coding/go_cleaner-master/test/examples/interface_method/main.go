package main

import (
	"code.byted.org/analyzers/go_cleaner/test/interface_method/pkg"
)

func getT(s1 *pkg.S1) pkg.T {
	return s1
}

func main() {
	s1 := pkg.S1{}
	s1_impl, _ := getT(&s1).(*pkg.S1)
	s1_impl.UsedFunc()

	s2 := pkg.S2{}
	s2.UsedFunc()
}
