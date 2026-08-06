package repoinfo

import (
	"fmt"
	"go/ast"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type SyntaxUsage struct {
	Dirs    []string
	Pattern string
	Usage   map[string]map[string]struct{}
}

func NewSyntaxUsage(dirs []string, pattern string) *SyntaxUsage {
	return &SyntaxUsage{
		Dirs:    dirs,
		Pattern: pattern,
		Usage:   map[string]map[string]struct{}{},
	}
}

func (s *SyntaxUsage) addUsage(pkg, obj string) {
	if _, ok := s.Usage[pkg]; !ok {
		s.Usage[pkg] = map[string]struct{}{}
	}

	s.Usage[pkg][obj] = struct{}{}
}

func (s *SyntaxUsage) saveUsage(w io.Writer) {
	for pkg, objs := range s.Usage {
		for obj := range objs {
			_, _ = fmt.Fprintf(w, "%s,%s\n", pkg, obj)
		}
	}
}

func (s *SyntaxUsage) targetPkg(f *ast.File) map[string]string {
	pkgFullName := map[string]string{}

	ast.Inspect(f, func(node ast.Node) bool {
		if imp, ok := node.(*ast.ImportSpec); ok {
			full := strings.ReplaceAll(imp.Path.Value, "\"", "")
			if !strings.HasPrefix(full, s.Pattern) {
				return true
			}
			name := filepath.Base(full)
			if imp.Name != nil {
				name = imp.Name.Name
			}
			if name == "." {
				return true
			}

			pkgFullName[name] = full

			return true
		}

		return true
	})

	return pkgFullName
}

func (s *SyntaxUsage) Parse() error {
	fset := token.NewFileSet()

	for _, dir := range s.Dirs {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() || !strings.HasSuffix(info.Name(), ".go") {
				return nil
			}

			f, parseErr := ParseFileWithoutAllErrs(fset, path, nil)
			if parseErr != nil {
				return parseErr
			}

			if f != nil {
				s.collectUsage(f, s.targetPkg(f))
			}

			return nil
		})

		if err != nil {
			return err
		}
	}

	return nil
}

func (s *SyntaxUsage) collectUsage(f *ast.File, pkg map[string]string) {
	ast.Inspect(f, func(node ast.Node) bool {
		if selExpr, ok := node.(*ast.SelectorExpr); ok {
			// 一般pkg.X 中Obj都是nil, 局部变量var.X中Obj不是nil，判断Obj==nil可以减少一些误判
			if xIdent, ok := selExpr.X.(*ast.Ident); ok && xIdent.Obj == nil {
				if fullPkg, ok := pkg[xIdent.Name]; ok {
					if v := selExpr.Sel; v != nil {
						s.addUsage(fullPkg, v.Name)
					}
				}
			}
		}

		return true
	})
}