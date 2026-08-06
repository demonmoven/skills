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

type KiteXEPC struct {
}

// Determine 判断是kitex框架，基于
// main.go 里有import kitex-gen, .NewServer
func (k *KiteXEPC) Determine(p *types.Project) bool {
	file, ok := p.Tree[p.AppRootFile("main.go")]
	if !ok {
		return false
	}

	if !util.Any(file.Imports, func(spec *ast.ImportSpec) bool {
		return strings.Contains(spec.Path.Value, "kitex_gen")
	}) {
		return false
	}

	hasNewServer := false
	hasRun := false

	for _, expr := range p.Call[p.AppRootFile("main.go")] {
		if sel, ok := expr.Fun.(*ast.SelectorExpr); ok {
			right := sel.Sel.Name
			if right == "NewServer" || right == "NewServerWithBytedConfig" {
				hasNewServer = true
			}
			if right == "Run" {
				hasRun = true
			}
		}
	}

	return hasNewServer && hasRun
}

func (k *KiteXEPC) Framework() string {
	return "kitex"
}

func (k *KiteXEPC) ClearEndpoint(p *types.Project, endpoint []string, handlerPath string) error {
	var files []*ast.File
	handlerFile := p.AppRootFile("handler.go")
	handlerDir := p.AppRootFile("handler")
	for path, file := range p.Tree {
		if path == handlerFile {
			files = append(files, file)
		} else if suc, _ := regexp.MatchString(handlerDir+"/([^/]+).go", path); suc {
			files = append(files, file)
		} else if suc, _ := regexp.MatchString(handlerPath+"/([^/]+).go", path); suc && handlerPath != "" {
			files = append(files, file)
		}
	}
	if len(files) == 0 {
		return errors.New("没有找到接口实现的入口文件，请确认 handler.go 或 handler/*.go 是否存在")
	}

	for _, file := range files {
		if err := clearEndpointInFile(p.FS, file, func(v *ast.FuncDecl) bool {
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

func clearEndpointInFile(fs *token.FileSet, file *ast.File, judgeIsDelTarget func(node *ast.FuncDecl) bool) error {
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

func (k *KiteXEPC) RMEndpoint(p *types.Project, endpoint []string) error {
	var files []*ast.File
	handlerFile := p.AppRootFile("handler.go")
	handlerDir := p.AppRootFile("handler")
	for path, file := range p.Tree {
		if path == handlerFile {
			files = append(files, file)
		} else if suc, _ := regexp.MatchString(handlerDir+"/([^/]+).go", path); suc {
			files = append(files, file)
		}
	}
	if len(files) == 0 {
		return errors.New("没有找到接口实现的入口文件，请确认 handler.go 或 handler/*.go 是否存在")
	}

	for _, file := range files {
		if err := rmEndpointInFile(p.FS, file, func(v *ast.FuncDecl) bool {
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

func rmEndpointInFile(fs *token.FileSet, file *ast.File, judgeIsDelTarget func(node *ast.FuncDecl) bool) error {
	hit := false
	delComment := map[*ast.CommentGroup]bool{}

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

func (k *KiteXEPC) isEndpoint(f *ast.FuncDecl) bool {
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
