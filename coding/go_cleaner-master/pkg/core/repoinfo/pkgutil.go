package repoinfo

import (
	"go/ast"

	"golang.org/x/tools/go/packages"
)

func AllDecls(pkgs []*packages.Package) (result []ast.Decl) {
	for _, p := range pkgs {
		for _, f := range p.Syntax {
			result = append(result, f.Decls...)
		}
	}
	return
}


func AllGenDecls(pkgs []*packages.Package) (result []*ast.GenDecl) {
	for _, decl := range AllDecls(pkgs) {
		if gen, ok := decl.(*ast.GenDecl); ok {
			result = append(result, gen)
		}
	}
	return
}