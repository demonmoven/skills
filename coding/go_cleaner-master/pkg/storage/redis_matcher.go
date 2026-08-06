package storage

import (
	"go/types"
	"golang.org/x/tools/go/callgraph/static"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
)

type RedisMatcher struct {
	*DefaultMatcher
}

func NewRedisMatcher(prog *ssa.Program, ssapkgs []*ssa.Package, pkgs []*packages.Package) *RedisMatcher {
	focus := make(map[*types.Package]bool)
	for _, pkg := range ssapkgs {
		focus[pkg.Pkg] = true
	}

	pkgmap := make(map[*types.Package]*packages.Package)
	packages.Visit(pkgs, nil, func(p *packages.Package) {
		if p.Types != nil {
			pkgmap[p.Types] = p
		}
	})

	return &RedisMatcher{
		DefaultMatcher: &DefaultMatcher{
			prog:         prog,
			pkgmap:       pkgmap,
			ssapkgs:      ssapkgs,
			cg:           static.CallGraph(prog),
			initializers: make(map[string][]ssa.CallInstruction),
			focus:        focus,
			scenarios: []*scenario{
				{
					pkgPath:      "code.byted.org/kv/goredis",
					initFun:      "NewClientWithOption",
					initArgIndex: 0,
					userFuns:     []string{"Get", "Set", "Del", "MGet", "MSet", "MSetNX", "Incr", "Decr", "IncrBy", "DecrBy"},
				},
			},
		},
	}
}
