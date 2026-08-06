package main

import (
	"fmt"

	"code.byted.org/analyzers/codeclean/monorepo1/cmds/cmd1/ipkg1"
	"code.byted.org/analyzers/codeclean/monorepo1/pkg/pkg1"
	"code.byted.org/analyzers/codeclean/monorepo1/pkg/pkg2"
)

func main() {
	fmt.Println(ipkg1.IPkg1Func2())
	fmt.Println(pkg1.Pkg1Func())
	fmt.Println(pkg2.Pkg2Func())
}
