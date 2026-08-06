package core

import (
	"fmt"
	"go/ast"
	"os"

	"github.com/sirupsen/logrus"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
)

func FilterSSAPkgsNotNil(pkgs []*ssa.Package) (pkgsNoErr []*ssa.Package) {
	if GlobalPassManager == nil || GlobalPassManager.ErrMgr == nil {
		return pkgs
	}

	for _, pkg := range pkgs {
		if pkg != nil {
			pkgsNoErr = append(pkgsNoErr, pkg)
		}
	}
	logrus.Infof("filter %d error SSA packages from %d", len(pkgs)-len(pkgsNoErr), len(pkgs))
	return
}

func PrintErrAfterClean(pkgs []*packages.Package) int {
	if GlobalPassManager == nil || GlobalPassManager.ErrMgr == nil {
		return packages.PrintErrors(pkgs)
	}

	var n int
	packages.Visit(pkgs, nil, func(pkg *packages.Package) {
		if GlobalPassManager.ErrMgr.IsPkgErrBeforeClean(pkg) {
			return
		}

		if len(pkg.Errors) > 0 {
			logrus.Infof("package %s has %d errors\n", pkg.PkgPath, len(pkg.Errors))
		}

		for _, err := range pkg.Errors {
			fmt.Fprint(os.Stderr, fmt.Sprintf("%s\n", err))
			n++
		}
	})
	return n
}

func FilterPkgsNoErrBeforeClean(pkgs []*packages.Package) []*packages.Package {
	if GlobalPassManager != nil && GlobalPassManager.ErrMgr != nil {
		return GlobalPassManager.ErrMgr.FilterPkgsNoErrBeforeClean(pkgs)
	}
	return pkgs
}

func MaybeNameReferenced(pattern string) bool {
	if GlobalPassManager != nil && GlobalPassManager.ErrMgr != nil {
		return GlobalPassManager.ErrMgr.MaybeErrPartReferenced(pattern)
	}
	return false
}

func MaybeSpecReferenced(n ast.Spec) bool {
	if GlobalPassManager == nil || GlobalPassManager.ErrMgr == nil {
		return false
	}

	errMgr := GlobalPassManager.ErrMgr

	switch spec := n.(type) {
	case *ast.ValueSpec:
		var patterns []string
		for _, name := range spec.Names {
			patterns = append(patterns, name.Name)
		}
		return errMgr.MaybeErrPartReferenced(patterns...)
	case *ast.TypeSpec:
		return errMgr.MaybeErrPartReferenced(spec.Name.Name)
	case *ast.ImportSpec:
		return errMgr.MaybeErrPartReferenced(spec.Name.Name)
	}

	return false
}
