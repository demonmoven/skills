package internal

import (
	"go/types"
	"strings"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
)

type packageMatcher struct {
	module string

	ssaPkg map[*ssa.Package]bool
	typPkg map[*types.Package]bool
	pkgPkg map[*packages.Package]bool
}

func newPackageMatcher(module string) *packageMatcher {
	return &packageMatcher{
		module: module,
		ssaPkg: make(map[*ssa.Package]bool),
		typPkg: make(map[*types.Package]bool),
		pkgPkg: make(map[*packages.Package]bool),
	}
}

func (p *packageMatcher) add1(s *types.Package) {
	if p.match(s.Path()) {
		p.typPkg[s] = true
	}
}

func (p *packageMatcher) add2(s *packages.Package) {
	if p.match(s.PkgPath) {
		p.pkgPkg[s] = true
	}
}

func (p *packageMatcher) add3(s *ssa.Package) {
	if p.match(s.Pkg.Path()) {
		p.ssaPkg[s] = true
	}
}

func (p *packageMatcher) hit1(s *types.Package) bool {
	return p.typPkg[s]
}

func (p *packageMatcher) hit2(s *packages.Package) bool {
	return p.pkgPkg[s]
}

func (p *packageMatcher) hit3(s *ssa.Package) bool {
	return p.ssaPkg[s]
}

func (p *packageMatcher) match(s string) bool {
	return s == p.module || strings.HasPrefix(s, p.module+"/") || strings.HasPrefix(s, p.module+".")
}
