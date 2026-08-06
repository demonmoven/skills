package repoinfo

import (
	"go/ast"
	"go/token"
	"go/types"
	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
)

type Package struct {
	Pkg      *packages.Package
	TypeInfo *types.Package
	PkgSSA   *ssa.Package
}

func (p *Package) Position(pos token.Pos) token.Position {
	return p.Pkg.Fset.Position(pos)
}

func (p *Package) Contains(pos token.Pos) bool {
	for _, f := range p.Pkg.Syntax {
		if f.Pos() <= pos && pos < f.End() {
			return true
		}
	}

	return false
}

func (p *Package) Find(pos token.Pos) ([]ast.Node, bool) {
	var file *ast.File
	for _, f := range p.Pkg.Syntax {
		if f.Pos() <= pos && pos < f.End() {
			file = f
			break
		}
	}

	if file == nil {
		return nil, false
	}

	return astutil.PathEnclosingInterval(file, pos, pos)
}

func (p *Package) Path(pos token.Pos) (path []ast.Node) {
	var file *ast.File
	for _, f := range p.Pkg.Syntax {
		if f.Pos() <= pos && pos < f.End() {
			file = f
			break
		}
	}

	if file == nil {
		return
	}

	path, _ = astutil.PathEnclosingInterval(file, pos, pos)
	return
}

func GetPkgsUsedByMain(initial []*packages.Package) (result []*packages.Package) {
	var mainpkgs []*packages.Package
	focused := make(map[*packages.Package]bool)
	for _, p := range initial {
		focused[p] = true
		if p.Name == "main" {
			mainpkgs = append(mainpkgs, p)
		}
	}

	// 如果找不到main，则返回所有的包，避免遗漏
	if len(mainpkgs) == 0 {
		return initial
	}

	packages.Visit(mainpkgs, nil, func(p *packages.Package) {
		if focused[p] {
			result = append(result, p)
		}
	})

	return
}