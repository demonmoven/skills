package internal

import (
	"go/ast"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
)

type NodeValueMgr struct {
	pkg  []*packages.Package
	ssa  []*ssa.Package
	data map[ast.Node]ssa.Value
	fun  map[ast.Node]*ssa.Function
}

// NewNodeValueMgr Deprecated
func NewNodeValueMgr(pkg []*packages.Package, ssap []*ssa.Package) *NodeValueMgr {
	m := &NodeValueMgr{
		pkg:  pkg,
		ssa:  ssap,
		data: map[ast.Node]ssa.Value{},
		fun:  map[ast.Node]*ssa.Function{},
	}
	m.CollectASTSSARelation()
	return m
}

func (m *NodeValueMgr) CollectASTSSARelation() {
	for i, pkg := range m.pkg {
		ssaPkg := m.ssa[i]
		for _, f := range pkg.Syntax {
			path := []ast.Node{&ast.Package{}}
			pathF := []ast.Node{&ast.Package{}}

			ast.Inspect(f, func(node ast.Node) bool {
				if node == nil {
					if path[len(path)-1] == pathF[len(pathF)-1] {
						pathF = pathF[:len(pathF)-1]
					}
					path = path[:len(path)-1]
					return false
				}
				switch node.(type) {
				case *ast.FuncDecl, *ast.GenDecl, *ast.FuncLit, *ast.File:
					pathF = append(pathF, node)
				}
				path = append(path, node)

				if e, ok := node.(ast.Expr); ok {
					f := ssa.EnclosingFunction(ssaPkg, append(pathF, node))
					if f != nil {
						if v, _ := f.ValueForExpr(e); v != nil {
							m.data[e] = v
							m.fun[e] = f
						}
					}
				}
				return true
			})
		}
	}
}

func (m *NodeValueMgr) Value(node ast.Node) ssa.Value {
	return m.data[node]
}
