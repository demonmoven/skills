package framework

import (
	"fmt"
	"go/ast"
	"path/filepath"
	"strings"

	"code.byted.org/analyzers/go_cleaner/pkg/endpoint/internal/types"
	"code.byted.org/analyzers/go_cleaner/pkg/endpoint/internal/util"
)

type GDPAPIViewEPC struct {
	unusedInitRouter map[string]bool
}

func NewGDPAPIViewEPC() *GDPAPIViewEPC {
	return &GDPAPIViewEPC{make(map[string]bool)}
}

// Determine 判断是gdp api框架 views格式，基于
// main.go 里有import gdp/gdp, gdp.Init, .Run()
func (k *GDPAPIViewEPC) Determine(p *types.Project) bool {
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
		return strings.HasPrefix(s, "views/")
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

func (k *GDPAPIViewEPC) Framework() string {
	return "gdp-api(view)"
}

func (k *GDPAPIViewEPC) ClearEndpoint(p *types.Project, endpoint []string, handlerPath string) error {
	// views目录下的一级go文件
	for path, file := range p.Tree {
		if filepath.Dir(path) == "views" {
			if err := k.clearEndpointInFile(file, p, endpoint); err != nil {
				return err
			}
		}
	}

	for path, file := range p.Tree {
		if filepath.Dir(path) == "views" {
			if err := k.clearRouterRegister(file, p, endpoint); err != nil {
				return err
			}
		}
	}
	return nil
}

func (k *GDPAPIViewEPC) clearEndpointInFile(file *ast.File, p *types.Project, endpoint []string) error {
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

func (k *GDPAPIViewEPC) clearRouterRegister(file *ast.File, p *types.Project, endpoint []string) error {
	ast.Inspect(file, func(node ast.Node) bool {
		if v, ok := node.(*ast.FuncDecl); ok && v.Body != nil && len(v.Body.List) > 0 {
			var stats []ast.Stmt
			for _, stat := range v.Body.List {
				if expr, ok := stat.(*ast.ReturnStmt); ok && len(expr.Results) == 1 {
					if composite, ok := expr.Results[0].(*ast.CompositeLit); ok {
						if array, ok := composite.Type.(*ast.ArrayType); ok {
							if star, ok := array.Elt.(*ast.StarExpr); ok {
								if sle, ok := star.X.(*ast.SelectorExpr); ok {
									if x, ok := sle.X.(*ast.Ident); ok && x.Name == "af" && sle.Sel.Name == "Router" {
										elts := []ast.Expr{}
										for _, e := range composite.Elts {
											if call, ok := e.(*ast.CallExpr); ok {
												if v, ok := call.Fun.(*ast.Ident); ok && k.unusedInitRouter[v.Name] {
													continue
												}
											}
											elts = append(elts, e)
										}
										composite.Elts = elts
									}
								}
							}
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

func (k *GDPAPIViewEPC) RMEndpoint(p *types.Project, endpoint []string) error {
	return k.ClearEndpoint(p, endpoint, "")
}
