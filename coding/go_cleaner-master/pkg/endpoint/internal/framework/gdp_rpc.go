package framework

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"code.byted.org/analyzers/go_cleaner/pkg/endpoint/internal/types"
	util2 "code.byted.org/analyzers/go_cleaner/pkg/endpoint/internal/util"
)

type GDPRPCEPC struct {
}

// Determine 判断是kitex框架，基于
// main.go 里有import gdp/gdp, gdp.Init, .Run()
func (k *GDPRPCEPC) Determine(p *types.Project) bool {
	file, ok := p.Tree["main.go"]
	if !ok {
		return false
	}

	if !util2.Any(file.Imports, func(spec *ast.ImportSpec) bool {
		return strings.Contains(spec.Path.Value, "code.byted.org/gdp/gdp")
	}) {
		return false
	}

	if !util2.Any(file.Imports, func(spec *ast.ImportSpec) bool {
		return strings.Contains(spec.Path.Value, "code.byted.org/gdp/raf/adapter/kitex")
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

func (k *GDPRPCEPC) Framework() string {
	return "gdp-rpc"
}

func (k *GDPRPCEPC) ClearEndpoint(p *types.Project, endpoint []string, handlerPath string) error {
	for path, v := range p.Tree {
		if strings.HasPrefix(path, "action/") {
			if err := k.clearEndpointInFile(v, p.FS, endpoint); err != nil {
				return err
			}
		}
	}
	return nil
}

func (k *GDPRPCEPC) clearEndpointInFile(file *ast.File, fs *token.FileSet, endpoint []string) error {
	ast.Inspect(file, func(node ast.Node) bool {
		if v, ok := node.(*ast.FuncDecl); ok && util2.EndpointContain(v.Name.Name, endpoint) && k.isActionEndpoint(v) {
			fmt.Printf("_delete_endpoint: %s\n", v.Name.Name)

			v.Body = &ast.BlockStmt{
				List: []ast.Stmt{
					&ast.ReturnStmt{
						Results: []ast.Expr{
							ast.NewIdent("nil"),
							ast.NewIdent("nil"),
						},
					},
				},
			}
		}
		return true
	})

	if err := util2.Save(file, fs); err != nil {
		return err
	}

	return nil
}

func (k *GDPRPCEPC) RMEndpoint(p *types.Project, endpoint []string) error {
	for path, v := range p.Tree {
		if strings.HasPrefix(path, "handler/") {
			if err := k.rmEndpointInFile(v, p.FS, endpoint); err != nil {
				return err
			}
		}
	}
	return nil
}

func (k *GDPRPCEPC) rmEndpointInFile(file *ast.File, fs *token.FileSet, endpoint []string) error {
	delComment := map[*ast.CommentGroup]bool{}

	var ds []ast.Decl
	for _, d := range file.Decls {
		if v, ok := d.(*ast.FuncDecl); ok && util2.EndpointContain(v.Name.Name, endpoint) && k.isHandlerEndpoint(v) {
			fmt.Printf("_delete_endpoint: %s\n", v.Name.Name)

			if v.Doc != nil {
				delComment[v.Doc] = true
			}
			continue
		} else {
			ds = append(ds, d)
		}
	}
	file.Decls = ds

	var cs []*ast.CommentGroup
	for _, c := range file.Comments {
		if delComment[c] {
			continue
		}
		cs = append(cs, c)
	}
	file.Comments = cs

	if err := util2.Save(file, fs); err != nil {
		return err
	}

	return nil
}

func (k *GDPRPCEPC) isActionEndpoint(f *ast.FuncDecl) bool {
	if f.Recv != nil && len(f.Recv.List) != 0 {
		return false
	}

	typ := f.Type

	//ep := f.Name.Name

	if len(typ.Params.List) != 2 || len(typ.Results.List) != 2 {
		return false
	}

	ctx := typ.Params.List[0]
	req := typ.Params.List[1]

	if typ.Results == nil {
		return false
	}
	resp := typ.Results.List[0]
	e := typ.Results.List[1]

	if v, ok := ctx.Type.(*ast.SelectorExpr); !ok {
		return false
	} else {
		if x, ok := v.X.(*ast.Ident); !ok || x.Name != "context" {
			return false
		}
		if x := v.Sel; x.Name != "Context" {
			return false
		}
	}

	if vv, ok := req.Type.(*ast.StarExpr); !ok {
		return false
	} else {
		if v, ok := vv.X.(*ast.SelectorExpr); !ok {
			return false
		} else {
			if x, ok := v.X.(*ast.Ident); !ok || x.Name == "" {
				return false
			}
			//if x := v.Sel; x.Name != fmt.Sprintf("%sRequest", ep) {
			//	return false
			//}
		}
	}

	if vv, ok := resp.Type.(*ast.StarExpr); !ok {
		return false
	} else {
		if v, ok := vv.X.(*ast.SelectorExpr); !ok {
			return false
		} else {
			if x, ok := v.X.(*ast.Ident); !ok || x.Name == "" {
				return false
			}
			//if x := v.Sel; x.Name != fmt.Sprintf("%sResponse", ep) {
			//	return false
			//}
		}
	}

	if v, ok := e.Type.(*ast.SelectorExpr); !ok {
		return false
	} else {
		if x, ok := v.X.(*ast.Ident); !ok || x.Name != "raf" {
			return false
		}
		if x := v.Sel; x.Name != "RespError" {
			return false
		}
	}

	return true
}

func (k *GDPRPCEPC) isHandlerEndpoint(f *ast.FuncDecl) bool {
	if f.Recv == nil || len(f.Recv.List) != 1 {
		return false
	}

	if v, ok := f.Recv.List[0].Type.(*ast.StarExpr); !ok {
		return false
	} else {
		if vv, ok := v.X.(*ast.Ident); !ok || !strings.HasSuffix(vv.Name, "Impl") {
			return false
		}
	}

	typ := f.Type

	//ep := f.Name.Name

	if len(typ.Params.List) != 2 || len(typ.Results.List) != 2 {
		return false
	}

	ctx := typ.Params.List[0]
	req := typ.Params.List[1]

	if typ.Results == nil {
		return false
	}
	resp := typ.Results.List[0]
	e := typ.Results.List[1]

	if v, ok := ctx.Type.(*ast.SelectorExpr); !ok {
		return false
	} else {
		if x, ok := v.X.(*ast.Ident); !ok || x.Name != "context" {
			return false
		}
		if x := v.Sel; x.Name != "Context" {
			return false
		}
	}

	if vv, ok := req.Type.(*ast.StarExpr); !ok {
		return false
	} else {
		if v, ok := vv.X.(*ast.SelectorExpr); !ok {
			return false
		} else {
			if x, ok := v.X.(*ast.Ident); !ok || x.Name == "" {
				return false
			}
			//if x := v.Sel; x.Name != fmt.Sprintf("%sRequest", ep) {
			//	return false
			//}
		}
	}

	if vv, ok := resp.Type.(*ast.StarExpr); !ok {
		return false
	} else {
		if v, ok := vv.X.(*ast.SelectorExpr); !ok {
			return false
		} else {
			if x, ok := v.X.(*ast.Ident); !ok || x.Name == "" {
				return false
			}
			//if x := v.Sel; x.Name != fmt.Sprintf("%sResponse", ep) {
			//	return false
			//}
		}
	}

	if v, ok := e.Type.(*ast.SelectorExpr); !ok {
		return false
	} else {
		if x, ok := v.X.(*ast.Ident); !ok || x.Name != "raf" {
			return false
		}
		if x := v.Sel; x.Name != "RespError" {
			return false
		}
	}

	return true
}
