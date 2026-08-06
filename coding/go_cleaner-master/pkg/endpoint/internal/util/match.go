package util

import (
	"fmt"
	"go/ast"
	"go/token"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
)

// stmt: root := r.Group("/api/admin")
// MatchGroup(stmt, "root", []string{"api, admin"}) should be ("r", 0)
func MatchGroup(stmt ast.Stmt, lhsName string, names []string) (rhsName string, index int) {
	// match x [:=] y.Group("a")
	index = -1
	assign, ok := stmt.(*ast.AssignStmt)
	if !ok || len(assign.Lhs) != 1 || len(assign.Rhs) != 1 {
		return
	}

	lhs, rhs := assign.Lhs[0], assign.Rhs[0]

	// match [x] := y.Group("a")
	lhsIdent, ok := lhs.(*ast.Ident)
	if !ok || lhsIdent.Name != lhsName {
		return
	}

	// match x := [y.Group("a")]
	call, ok := rhs.(*ast.CallExpr)
	if !ok {
		return
	}

	// match x := y.[Group("a")]
	sle, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sle.Sel.Name != "Group" || len(call.Args) != 1 {
		return
	}

	// match x := [y].Group("a")
	rhsIdent, ok := sle.X.(*ast.Ident)
	if !ok {
		return
	}

	// match x := y.Group(["a"])
	lit, ok := call.Args[0].(*ast.BasicLit)
	if !ok {
		return
	}

	if index = eatSuffix(names, SplitRestful(lit.Value)); index == -1 {
		return
	}

	rhsName = rhsIdent.Name
	return
}

// stmt: user.POST("/list", ...)
// MatchHTTPRegister(stmt, "/list") should be ("user", 0)
func MatchMethod(stmt ast.Stmt, names []string) (lhsName string, index int) {
	index = -1
	// match [y.POST("/list", ...)]
	expr, ok := stmt.(*ast.ExprStmt)
	if !ok {
		return
	}

	call, ok := expr.X.(*ast.CallExpr)
	if !ok || len(call.Args) == 0 {
		return
	}

	// match y.[POST]("/list", ...)
	sle, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || !IsHTTPRegister(sle.Sel.Name) {
		return
	}

	// match y.POST(["/list"],...)
	lit, ok := call.Args[0].(*ast.BasicLit)
	if !ok {
		return
	}

	if index = eatSuffix(names, SplitRestful(lit.Value)); index == -1 {
		return
	}

	if receiver, ok := sle.X.(*ast.Ident); ok {
		lhsName = receiver.Name
	} else {
		index = -1
	}

	return
}

func SplitRestful(litv string) (splits []string) {
	for _, k := range strings.Split(strings.ReplaceAll(litv, "\"", ""), "/") {
		if k != "" {
			splits = append(splits, k)
		}
	}
	return
}

func eatSuffix(s []string, suffix []string) int {
	if len(suffix) > len(s) {
		return -1
	}

	for i := 0; i < len(suffix); i++ {
		if s[len(s)-1-i] != suffix[len(suffix)-1-i] {
			return -1
		}
	}

	return len(s) - len(suffix)
}

func IsGroup(stat ast.Stmt) (variable, path string, isDefine bool, ok bool) {
	if assign, ok := stat.(*ast.AssignStmt); ok && len(assign.Lhs) == 1 && len(assign.Rhs) == 1 {
		g, gr := assign.Lhs[0], assign.Rhs[0]
		if v, ok := g.(*ast.Ident); ok {
			if call, ok := gr.(*ast.CallExpr); ok {
				if sle, ok := call.Fun.(*ast.SelectorExpr); ok && sle.Sel.Name == "Group" {
					if len(call.Args) > 0 {
						if lit, ok := call.Args[0].(*ast.BasicLit); ok {
							variable = v.Name
							path = strings.ReplaceAll(lit.Value, "\"", "")
							isDefine = assign.Tok == token.DEFINE
							return variable, path, isDefine, true
						}
					}
				}
			}
		}
	}
	return "", "", false, false
}

func IsGroupGDPAF(stat ast.Stmt) (variable, path string, isDefine bool, ok bool) {
	if assign, ok := stat.(*ast.AssignStmt); ok && len(assign.Lhs) == 1 && len(assign.Rhs) == 1 {
		g, gr := assign.Lhs[0], assign.Rhs[0]
		if v, ok := g.(*ast.Ident); ok {
			if call, ok := gr.(*ast.CallExpr); ok {
				if sle, ok := call.Fun.(*ast.SelectorExpr); ok && sle.Sel.Name == "NewRouter" {
					if len(call.Args) > 0 {
						if lit, ok := call.Args[0].(*ast.BasicLit); ok {
							variable = v.Name
							path = strings.ReplaceAll(lit.Value, "\"", "")
							isDefine = assign.Tok == token.DEFINE
							return variable, path, isDefine, true
						}
					}
				}
			}
		}
	}
	return "", "", false, false
}

func BackwardMatchAndRewrite(file *ast.File, endpoints []string) (changed bool) {
	var findRootGroup func(splits []string, lhsv string, start int, previousStart int) bool

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		stmts := StmtsInProgramOrder(fn)
		if len(stmts) == 0 {
			continue
		}

		findRootGroup = func(splits []string, lhsv string, start int, previousStart int) bool {
			if len(splits) == 0 {
				return true
			}

			for i := start; i >= 0; i-- {
				rhv, index := MatchGroup(stmts[i], lhsv, splits)
				if index != -1 && !IsVariableStrongUpdatedBetween(stmts, lhsv, i, previousStart) {
					return findRootGroup(splits[0:index], rhv, i-1, start)
				}
			}

			return false
		}

		removes := make(map[ast.Stmt]bool)
		for _, ep := range endpoints {
			splits := SplitRestful(ep)

			candidates := make(map[ast.Stmt]string)
			splitindex := make(map[ast.Stmt]int)
			stmtsindex := make(map[ast.Stmt]int)
			for i, stmt := range stmts {
				if receiver, index := MatchMethod(stmt, splits); index != -1 {
					candidates[stmt] = receiver
					splitindex[stmt] = index
					stmtsindex[stmt] = i - 1
				}
			}

			for stmt, name := range candidates {
				if findRootGroup(splits[0:splitindex[stmt]], name, stmtsindex[stmt], stmtsindex[stmt]+1) {
					removes[stmt] = true
					fmt.Printf("_delete_endpoint: %s\n", ep)
				}
			}
		}

		if len(removes) > 0 {
			changed = true
		}

		for stmt := range removes {
			astutil.Apply(fn, nil, func(c *astutil.Cursor) bool {
				if c.Node() == stmt {
					c.Delete()
				}
				return true
			})
		}
	}

	return
}

func ForwardMatchAndRewrite(file *ast.File, endpoints []string) (changed bool) {
	ast.Inspect(file, func(node ast.Node) bool {
		if v, ok := node.(*ast.FuncDecl); ok && v.Body != nil && len(v.Body.List) > 0 {
			group := map[string]string{}
			delGroup := map[string]bool{}
			keepGroup := map[string]bool{}
			groupStatIndex := map[string]int{}
			var stats []ast.Stmt
			for _, stat := range v.Body.List {
				// g := r.Group("a")
				if v, p, define, ok := IsGroup(stat); ok {
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
						if sle, ok := call.Fun.(*ast.SelectorExpr); ok && IsHTTPRegister(sle.Sel.Name) {
							if g, ok := sle.X.(*ast.Ident); ok {
								if len(call.Args) > 0 {
									if lit, ok := call.Args[0].(*ast.BasicLit); ok {
										if HTTPEndpointContains(group[g.Name], strings.ReplaceAll(lit.Value, "\"", ""), endpoints) {
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

			var tmp []ast.Stmt
			for i := range stats {
				if delIndex[i] {
					changed = true
					continue
				}
				tmp = append(tmp, stats[i])
			}
			v.Body.List = tmp
		}
		return true
	})
	return
}
