package framework

import (
	"fmt"
	"go/ast"
	"path/filepath"
	"strings"

	"code.byted.org/analyzers/go_cleaner/pkg/endpoint/internal/types"
	"code.byted.org/analyzers/go_cleaner/pkg/endpoint/internal/util"
)

type GDPAPIEPC struct {
	unusedInitRouter map[string]bool
}

func NewGDPAPIEPC() *GDPAPIEPC {
	return &GDPAPIEPC{make(map[string]bool)}
}

// Determine 判断是gdp api框架，基于
// main.go 里有import gdp/gdp, gdp.Init, .Run()
func (k *GDPAPIEPC) Determine(p *types.Project) bool {
	file, ok := p.Tree["main.go"]
	if !ok {
		return false
	}

	if !util.Any(file.Imports, func(spec *ast.ImportSpec) bool {
		return strings.Contains(spec.Path.Value, "code.byted.org/gdp/gdp")
	}) {
		return false
	}

	if !util.Any(file.Imports, func(spec *ast.ImportSpec) bool {
		return strings.Contains(spec.Path.Value, "code.byted.org/gdp/af/adapter/hertz")
	}) {
		return false
	}

	if !util.AnyKey(p.Tree, func(s string) bool {
		return strings.HasPrefix(s, "router/")
	}) {
		return false
	}

	hasInit := false
	hasRun := false

	for _, expr := range p.Call["main.go"] {
		if sel, ok := expr.Fun.(*ast.SelectorExpr); ok {
			left := sel.X
			right := sel.Sel.Name
			if v, ok := left.(*ast.Ident); ok && v.Name == "gdp" && right == "Init" {
				hasInit = true
			}

			if right == "Run" {
				hasRun = true
			}
		}
	}

	return hasInit && hasRun
}

func (k *GDPAPIEPC) Framework() string {
	return "gdp-api"
}

func (k *GDPAPIEPC) ClearEndpoint(p *types.Project, endpoint []string, handlerPath string) error {
	for path, file := range p.Tree {
		if strings.HasPrefix(path, "router/") {
			if err := k.clearEndpointInFile(file, p, endpoint); err != nil {
				return err
			}
		}
	}

	if file, ok := p.Tree["router/init.go"]; ok {
		if err := k.clearRouterRegister(file, p, endpoint); err != nil {
			return err
		}
	}
	return nil
}

func (k *GDPAPIEPC) clearEndpointInFile(file *ast.File, p *types.Project, endpoint []string) error {
	ast.Inspect(file, func(node ast.Node) bool {
		if v, ok := node.(*ast.FuncDecl); ok && v.Body != nil && len(v.Body.List) > 0 {
			group := map[string]string{}
			delGroup := map[string]bool{}
			keepGroup := map[string]bool{}
			groupStatIndex := map[string]int{}
			var stats []ast.Stmt
			for _, stat := range v.Body.List {
				// g := af.NewRouter("a")
				if v, p, define, ok := util.IsGroupGDPAF(stat); ok {
					if define {
						groupStatIndex[v] = len(stats)
					}
					group[v] = p
					stats = append(stats, stat)
					continue
				}

				// g.Get("b", h)
				if expr, ok := stat.(*ast.ExprStmt); ok {
					if call, ok := expr.X.(*ast.CallExpr); ok {
						if sle, ok := call.Fun.(*ast.SelectorExpr); ok && util.IsHTTPRegister(sle.Sel.Name) {
							if g, ok := sle.X.(*ast.Ident); ok {
								if len(call.Args) > 0 {
									if lit, ok := call.Args[0].(*ast.BasicLit); ok {
										if util.HTTPEndpointContains(group[g.Name], strings.ReplaceAll(lit.Value, "\"", ""), endpoint) {
											fmt.Printf("_delete_endpoint: %s\n", filepath.Join(group[g.Name], strings.ReplaceAll(lit.Value, "\"", "")))

											delGroup[g.Name] = true
											continue
										}
									}
								}
							}
						}
					}
				}

				if expr, ok := stat.(*ast.ExprStmt); ok {
					if call, ok := expr.X.(*ast.CallExpr); ok {
						if sle, ok := call.Fun.(*ast.SelectorExpr); ok {
							if g, ok := sle.X.(*ast.Ident); ok {
								keepGroup[g.Name] = true
							}
						}
					}
				}

				stats = append(stats, stat)
			}

			delIndex := map[int]bool{}

			for g, i := range groupStatIndex {
				if delGroup[g] && !keepGroup[g] {
					delIndex[i] = true
				}
			}

			if len(delIndex) == len(group) {
				k.unusedInitRouter[v.Name.Name] = true
			}

			v.Body.List = stats
		}
		return true
	})

	if err := util.Save(file, p.FS); err != nil {
		return err
	}

	return nil
}

func (k *GDPAPIEPC) clearRouterRegister(file *ast.File, p *types.Project, endpoint []string) error {
	ast.Inspect(file, func(node ast.Node) bool {
		if v, ok := node.(*ast.FuncDecl); ok && v.Body != nil && len(v.Body.List) > 0 {
			var stats []ast.Stmt
			for _, stat := range v.Body.List {
				if expr, ok := stat.(*ast.AssignStmt); ok && len(expr.Rhs) == 1 {
					if call, ok := expr.Rhs[0].(*ast.CallExpr); ok {
						if fi, ok := call.Fun.(*ast.Ident); ok && fi.Name == "append" {
							args := []ast.Expr{}
							for _, e := range call.Args {
								if ce, ok := e.(*ast.CallExpr); ok {
									if cef, ok := ce.Fun.(*ast.Ident); ok && k.unusedInitRouter[cef.Name] {
										continue
									}
								}
								args = append(args, e)
							}
							if len(args) < 2 {
								// 删除 rs = append(rs, initAudioTrack())
								continue
							}
							call.Args = args
						}
					}
				}
				stats = append(stats, stat)
			}

			v.Body.List = stats
		}
		return true
	})

	if err := util.Save(file, p.FS); err != nil {
		return err
	}

	return nil
}

func (k *GDPAPIEPC) RMEndpoint(p *types.Project, endpoint []string) error {
	return k.ClearEndpoint(p, endpoint, "")
}
