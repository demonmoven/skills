package v2

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"strings"

	"code.byted.org/analyzers/go_cleaner/pkg/searchimport/internal"
)

type Opt = internal.Opt

type Detector struct {
	searcher  *astImportSearcher
	opt       *Opt
}

func NewDetector(opt *Opt, dir string) *Detector {
	d := &Detector{
		opt: opt,
	}

	s := mustNewAstImportSearcher(dir, opt.TargetModule)

	d.searcher = s

	return d
}

func (d *Detector) Print() error {
	w := os.Stdout

	if f := d.opt.OutFile; f != "" {
		var err error
		w, err = os.OpenFile(f, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)
		if err != nil {
			return err
		}
	}

	d.searcher.saveUsage(w)

	return nil
}

type astImportSearcher struct {
	dir    string
	target string

	usage map[string]map[string]struct{}
}

func mustNewAstImportSearcher(dir, targetModule string) *astImportSearcher {
	s := &astImportSearcher{
		dir:    dir,
		target: targetModule,
		usage:  map[string]map[string]struct{}{},
	}

	if err := s.parse(); err != nil {
		panic(err)
	}

	return s
}

func (s *astImportSearcher) addUsage(pkg, obj string) {
	if _, ok := s.usage[pkg]; !ok {
		s.usage[pkg] = map[string]struct{}{}
	}

	s.usage[pkg][obj] = struct{}{}
}

func (s *astImportSearcher) saveUsage(w io.Writer) {
	for pkg, objs := range s.usage {
		for obj := range objs {
			_, _ = fmt.Fprintf(w, "%s,%s\n", pkg, obj)
		}
	}
}

func (s *astImportSearcher) targetPkg(f *ast.File) map[string]string {
	pkgFullName := map[string]string{}

	ast.Inspect(f, func(node ast.Node) bool {
		if imp, ok := node.(*ast.ImportSpec); ok {
			full := strings.ReplaceAll(imp.Path.Value, "\"", "")
			if !strings.HasPrefix(full, s.target) {
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

func (s *astImportSearcher) collectUsage(f *ast.File, pkg map[string]string) {
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

func (s *astImportSearcher) parse() error {
	fset := token.NewFileSet()

	err := filepath.Walk(s.dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !strings.HasSuffix(info.Name(), ".go") {
			return nil
		}

		f, parseErr := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if parseErr != nil {
			fmt.Println(parseErr)
			return parseErr
		}

		s.collectUsage(f, s.targetPkg(f))

		return nil
	})

	return err
}
