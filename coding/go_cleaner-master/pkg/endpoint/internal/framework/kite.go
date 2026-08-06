package framework

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"regexp"
	"strings"

	"code.byted.org/analyzers/go_cleaner/pkg/endpoint/internal/types"
	"code.byted.org/analyzers/go_cleaner/pkg/endpoint/internal/util"
)

type KiteEPC struct {
}

// Determine 判断是kite框架，基于
// server.go 里有kite.Init
func (k *KiteEPC) Determine(p *types.Project) bool {
	file, ok := p.Tree["server.go"]
	if !ok {
		return false
	}

	_, ok = p.Tree["kite.go"]
	if !ok {
		return false
	}

	_, ok = p.Tree["idls.go"]
	if !ok {
		return false
	}

	_, ok = p.Tree["handler.go"]
	if !ok {
		return false
	}

	if !util.Any(file.Imports, func(spec *ast.ImportSpec) bool {
		return strings.Contains(spec.Path.Value, "code.byted.org/kite/kite")
	}) {
		return false
	}

	hasInit := false

	for _, expr := range p.Call["server.go"] {
		if sel, ok := expr.Fun.(*ast.SelectorExpr); ok {
			right := sel.Sel.Name
			if right == "Init" {
				hasInit = true
			}
		}
	}

	return hasInit
}

func (k *KiteEPC) Framework() string {
	return "kite"
}

func (k *KiteEPC) ClearEndpoint(p *types.Project, endpoint []string, handlerPath string) error {
	var files []*ast.File
	for path, file := range p.Tree {
		if path == "handler.go" {
			files = append(files, file)
		} else if suc, _ := regexp.MatchString("handler/([^/]+).go", path); suc {
			files = append(files, file)
		} else if suc, _ := regexp.MatchString(handlerPath+"/([^/]+).go", path); suc && handlerPath != "" {
			files = append(files, file)
		}
	}
	if len(files) == 0 {
		return errors.New("没有找到接口实现的入口文件，请确认 handler.go 或 handler/*.go 是否存在")
	}

	for _, file := range files {
		if err := clearKiteEndpointInFile(p.FS, file, func(v *ast.FuncDecl) bool {
			if util.EndpointContain(v.Name.Name, endpoint) && k.isEndpoint(v) {
				return true
			}
			return false
		}); err != nil {
			return err
		}
	}
	return nil
}

func clearKiteEndpointInFile(fs *token.FileSet, file *ast.File, judgeIsDelTarget func(node *ast.FuncDecl) bool) error {
	var (
		hit = false

		name, hasErrors = util.HasErrors(file.Imports)
		conflict        = util.MayErrorsConflict(file.Imports)
	)

	if !hasErrors {
		name = "errors"
		if conflict {
			name = "stderrors"
		}
	}

	delCommentLine := map[int]bool{}

	ast.Inspect(file, func(node ast.Node) bool {
		if v, ok := node.(*ast.FuncDecl); ok && judgeIsDelTarget(v) {
			fmt.Printf("_delete_endpoint: %s\n", v.Name.Name)

			start := fs.Position(v.Pos()).Line
			end := fs.Position(v.End()).Line
			for i := start; i <= end; i++ {
				delCommentLine[i] = true
			}

			v.Body = &ast.BlockStmt{
				List: []ast.Stmt{
					&ast.ReturnStmt{
						Return: token.Pos(1), // 通常设置为 1，除非你要处理 token 的位置
						Results: []ast.Expr{
							ast.NewIdent("nil"), // return 的第一个值是 nil
							// return 的第二个值是 errors.New("interface offline") 调用
							&ast.CallExpr{
								Fun: &ast.SelectorExpr{
									X:   ast.NewIdent(name),  // errors 包
									Sel: ast.NewIdent("New"), // New 函数
								},
								Args: []ast.Expr{
									&ast.BasicLit{
										Kind:  token.STRING,          // 参数是一个字符串类型
										Value: `"interface offline"`, // 字符串的值
									},
								},
							},
						},
					},
				},
			}

			hit = true
		}
		return true
	})

	if !hit {
		return nil
	}

	var cs []*ast.CommentGroup
	for _, c := range file.Comments {
		if !delCommentLine[fs.Position(c.Pos()).Line] {
			cs = append(cs, c)
		}
	}
	file.Comments = cs

	if hit && !hasErrors {
		imp := &ast.ImportSpec{
			Path: &ast.BasicLit{
				Kind:  token.STRING,
				Value: `"errors"`,
			},
		}
		if name != "errors" {
			imp.Name = ast.NewIdent(name)
		}

		util.AddImport(file, imp)
	}

	if err := util.Save(file, fs); err != nil {
		return err
	}

	return nil
}

func (k *KiteEPC) RMEndpoint(p *types.Project, endpoint []string) error {
	var files []*ast.File
	for path, file := range p.Tree {
		if path == "handler.go" {
			files = append(files, file)
		} else if suc, _ := regexp.MatchString("handler/([^/]+).go", path); suc {
			files = append(files, file)
		}
	}
	if len(files) == 0 {
		return errors.New("没有找到接口实现的入口文件，请确认 handler.go 或 handler/*.go 是否存在")
	}

	for _, file := range files {
		if err := rmKiteEndpointInFile(p.FS, file, func(v *ast.FuncDecl) bool {
			if util.EndpointContain(v.Name.Name, endpoint) && k.isEndpoint(v) {
				return true
			}
			return false
		}); err != nil {
			return err
		}
	}
	return nil
}

func rmKiteEndpointInFile(fs *token.FileSet, file *ast.File, judgeIsDelTarget func(node *ast.FuncDecl) bool) error {
	delComment := map[*ast.CommentGroup]bool{}
	hit := false

	var ds []ast.Decl
	for _, d := range file.Decls {
		if v, ok := d.(*ast.FuncDecl); ok && judgeIsDelTarget(v) {
			fmt.Printf("_delete_endpoint: %s\n", v.Name.Name)

			if v.Doc != nil {
				delComment[v.Doc] = true
			}
			hit = true
			continue
		} else {
			ds = append(ds, d)
		}
	}

	if !hit {
		return nil
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

	if err := util.Save(file, fs); err != nil {
		return err
	}

	return nil
}

func (k *KiteEPC) isEndpoint(f *ast.FuncDecl) bool {
	if f.Recv != nil {
		return false
	}

	typ := f.Type

	//ep := f.Name.Name

	if typ.Params == nil || typ.Results == nil || len(typ.Params.List) != 2 || len(typ.Results.List) != 2 {
		return false
	}

	ctx := typ.Params.List[0]
	req := typ.Params.List[1]

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

	if vv, ok := e.Type.(*ast.Ident); !ok || vv.Name != "error" {
		return false
	}

	return true
}
