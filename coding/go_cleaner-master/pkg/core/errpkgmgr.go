package core

import (
	"bufio"
	"go/ast"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"golang.org/x/tools/go/packages"
)

type CompileErrManager struct {
	pkgNameToPkgs map[string]*packages.Package
	errFilePaths  map[string][]string
}

func NewCompileErrManager() *CompileErrManager {
	return &CompileErrManager{
		pkgNameToPkgs: map[string]*packages.Package{},
		errFilePaths:  map[string][]string{},
	}
}

func (m *CompileErrManager) NumErrPkgs() int {
	return len(m.pkgNameToPkgs)
}

func (m *CompileErrManager) AddPkg(pkg *packages.Package) {
	m.pkgNameToPkgs[pkg.PkgPath] = pkg
	var pathsAssumeAsErr []string
	for _, file := range pkg.Syntax {
		pathsAssumeAsErr = append(pathsAssumeAsErr, pkg.Fset.Position(file.Pos()).Filename)
	}

	for _, path := range pathsAssumeAsErr {
		if _, ok := m.errFilePaths[path]; ok {
			continue
		}

		m.errFilePaths[path] = []string{}
		logrus.Infof("add error file: %s", path)

		f, err := os.Open(path)
		if err != nil {
			logrus.Errorf("open error file err: %v", err)
			f.Close()
			continue
		}

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			m.errFilePaths[path] = append(m.errFilePaths[path], scanner.Text())
		}

		if err := scanner.Err(); err != nil {
			logrus.Errorf("scan error file err: %v", err)
		}

		f.Close()
	}
}

// 不能直接使用指针查询，因为对项目的不同加载可能会导致同一个package的指针不同
func (m *CompileErrManager) IsPkgErrBeforeClean(pkg *packages.Package) bool {
	return m.pkgNameToPkgs[pkg.PkgPath] != nil
}

func (m *CompileErrManager) FilterPkgsNoErrBeforeClean(pkgs []*packages.Package) (pkgsNoErr []*packages.Package) {
	for _, pkg := range pkgs {
		if m.IsPkgErrBeforeClean(pkg) {
			logrus.Infof("package %s has %d errors\n", pkg.PkgPath, len(pkg.Errors))
		} else {
			pkgsNoErr = append(pkgsNoErr, pkg)
		}
	}
	logrus.Infof("filter %d error packages from %d", len(pkgs)-len(pkgsNoErr), len(pkgs))
	return
}

func (m *CompileErrManager) MaybeErrPartReferenced(patterns ...string) bool {
	for _, path := range m.errFilePaths {
		for _, line := range path {
			for _, p := range patterns {
				if strings.Contains(line, p) {
					return true
				}
			}
		}
	}
	return false
}

func getFilePathsFromTypeErr(pkg *packages.Package) (filepaths []string) {
	for _, err := range pkg.TypeErrors {
		var file *ast.File
		for _, f := range pkg.Syntax {
			if f.Pos() <= err.Pos && err.Pos < f.End() {
				file = f
				break
			}
		}
		if file == nil {
			continue
		}

		filepaths = append(filepaths, pkg.Fset.Position(file.Pos()).Filename)
	}
	return
}

func getFilePathsFromErr(pkg *packages.Package) (filepaths []string) {
	for _, err := range pkg.Errors {
		if seps := strings.Split(err.Pos, ":"); len(seps) > 0 {
			filepaths = append(filepaths, seps[0])
		}
	}
	return
}
