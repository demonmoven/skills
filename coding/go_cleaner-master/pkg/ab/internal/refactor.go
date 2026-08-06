package internal

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"reflect"

	"golang.org/x/tools/go/ssa"
)

func (d *Detector) Refactor() {
	for _, pkg := range d.initialPkgs {
		for _, file := range pkg.Syntax {
			// 删除无用分支
			if d.opt.onlyRe.MatchString(d.position(file.Pos()).Filename) {
				d.RefactorFile(file)
			}
		}
	}
	// 删除定义但没使用的局部变量导致的报错
	for i := 0; i < 2; i++ {
		newUnusedDeclCleaner().clean()
	}
}

// RefactorFile 可以指refactor调用了ab函数和上下游1-2层
// TODO
func (d *Detector) RefactorFile(file *ast.File) {
	d.refactor(file)
	d.deleteUnusedComment(file)
	save(file, d.prog.Fset)
}

func (d *Detector) DebugNodeVal() {
	for _, pkg := range d.initialPkgs {
		for _, file := range pkg.Syntax {
			ast.Inspect(file, func(node ast.Node) bool {
				switch v := node.(type) {
				case ast.Expr:
					vs, connected := d.NodeValue00(v)
					ssav := d.ASTExprToSSAValue[v]
					fmt.Printf("%v ast: %v => %v, ssa %v => %v, connected: %v\n", d.position(v.Pos()), reflect.TypeOf(v), vs, reflect.TypeOf(ssav), d.Value(ssav), connected)
				}
				return true
			})
		}
	}
}

func (d *Detector) refactor(file *ast.File) {
	ast.Inspect(file, func(node ast.Node) bool {
		switch v := node.(type) {
		case *ast.FuncDecl:
			v.Body = d.refactorFuncBody(v.Body)
		case *ast.FuncLit:
			v.Body = d.refactorFuncBody(v.Body)
		}
		return true
	})
}

func (d *Detector) refactorFuncBody(block *ast.BlockStmt) *ast.BlockStmt {
	block = d.refactorBlock(block)
	newList := cutReturn(block.List)

	if len(block.List) > 0 && len(newList) > 0 && len(newList) < len(block.List) {
		startPos := d.position(newList[len(newList)-1].End())
		endPos := d.position(block.List[len(block.List)-1].End())
		file := startPos.Filename
		if _, ok := d.deleteLines[file]; !ok {
			d.deleteLines[file] = map[int]bool{}
		}
		for i := startPos.Line + 1; i <= endPos.Line; i++ {
			d.deleteLines[file][i] = true
		}
	}

	block.List = newList
	return block
}

func (d *Detector) refactorBlock(block *ast.BlockStmt) *ast.BlockStmt {
	var stats []ast.Stmt
	for _, stat := range block.List {
		switch v := stat.(type) {
		case *ast.IfStmt:
			if mustTrue(d.NodeValue(v.Cond)) {
				stats = append(stats, v.Body.List...)
				break
			}
			if mustFalse(d.NodeValue(v.Cond)) {
				if elseStat, ok := v.Else.(*ast.BlockStmt); ok {
					stats = append(stats, elseStat.List...)
				}
				break
			}
			v.Cond = d.refactorExpr(v.Cond)
			stats = append(stats, v)

		case *ast.SwitchStmt:
			stats = append(stats, d.refactorSwitch(v)...)

		case *ast.ReturnStmt:
			var results []ast.Expr
			for _, v := range v.Results {
				results = append(results, d.refactorExpr(v))
			}
			v.Results = results
			stats = append(stats, v)

		case *ast.AssignStmt:
			stats = append(stats, d.refactorAssign(v))

		default:
			stats = append(stats, v)
		}
	}
	block.List = stats
	return block
}

func (d *Detector) refactorSwitch(v *ast.SwitchStmt) (stats []ast.Stmt) {
	tv := d.NodeValue(v.Tag)
	tvMap := map[constant.Value]bool{}
	for _, val := range tv {
		if val == VariantValue {
			// 如果tag的值存在不确定的，整个switch不修改
			stats = append(stats, v)
			return
		}
		tvMap[val] = true
	}
	var cases []ast.Stmt
clause:
	for _, vv := range v.Body.List {
		clause := vv.(*ast.CaseClause)
		if clause.List == nil {
			// default case
			cases = append(cases, clause)
			continue clause
		}
		for _, v := range clause.List {
			for _, val := range d.NodeValue(v) {
				// 这个值可能取到，这个case就保留
				if val == VariantValue || tvMap[val] {
					cases = append(cases, clause)
					continue clause
				}
			}
		}
	}
	// TODO: 有风险
	if len(cases) == 0 {
		return nil // 删除整个switch
	}
	// 只有一个default时，删除switch
	if len(cases) == 1 && cases[0].(*ast.CaseClause).List == nil {
		stats = append(stats, cases[0].(*ast.CaseClause).Body...)
		return
	}
	// 裁剪case
	v.Body.List = cases
	stats = append(stats, v)
	return
}

func toConstExpr(val constant.Value, pos token.Pos) ast.Expr {
	switch val.Kind() {
	case constant.String:
		return &ast.BasicLit{
			ValuePos: pos,
			Kind:     token.STRING,
			Value:    val.ExactString(),
		}
	case constant.Int:
		return &ast.BasicLit{
			ValuePos: pos,
			Kind:     token.INT,
			Value:    val.ExactString(),
		}
	case constant.Bool:
		return &ast.Ident{
			NamePos: pos,
			Name:    val.ExactString(),
		}
	}
	return nil
}

func (d *Detector) refactorAssign(assign *ast.AssignStmt) *ast.AssignStmt {
	if len(assign.Rhs) == 1 {
		if expr, ok := assign.Rhs[0].(*ast.TypeAssertExpr); ok {
			if call, ok := expr.X.(*ast.CallExpr); ok && d.abEntryCallPos[call.Lparen] != nil {
				if vals := d.NodeValue(call); len(vals) == 1 && vals[0] != VariantValue {
					if constExpr := toConstExpr(vals[0], expr.Pos()); constExpr != nil {
						if ident, ok := expr.Type.(*ast.Ident); ok && (ident.Name == "int32" || ident.Name == "int64") {
							call := &ast.CallExpr{
								Fun: &ast.Ident{
									NamePos: constExpr.Pos() - 6,
									Name:    ident.Name,
									Obj:     nil,
								},
								Lparen:   constExpr.Pos() - 1,
								Args:     []ast.Expr{constExpr},
								Ellipsis: 0,
								Rparen:   constExpr.Pos() + 1,
							}
							assign.Rhs = []ast.Expr{call}
						} else {
							assign.Rhs = []ast.Expr{constExpr}
						}
						assign.Lhs = assign.Lhs[:1]
						return assign
					}
				}
			}
		}
	}

	var rhs []ast.Expr
	for _, e := range assign.Rhs {
		rhs = append(rhs, d.refactorExpr(e))
	}
	if len(assign.Lhs) == len(rhs) {
		assign.Rhs = rhs
	}
	return assign
}

func (d *Detector) refactorExpr(expr ast.Expr) ast.Expr {
	if expr == nil {
		return expr
	}
	switch v := expr.(type) {
	case *ast.BinaryExpr:
		switch v.Op {
		case token.LAND:
			if mustTrue(d.NodeValue(v.X)) && mustTrue(d.NodeValue(v.Y)) {
				return &ast.Ident{
					NamePos: v.Pos(),
					Name:    "true",
				}
			}
			if mustTrue(d.NodeValue(v.X)) {
				return d.refactorExpr(v.Y)
			}
			if mustTrue(d.NodeValue(v.Y)) {
				return d.refactorExpr(v.X)
			}
			if mustFalse(d.NodeValue(v.X)) || mustFalse(d.NodeValue(v.Y)) {
				return &ast.Ident{
					NamePos: v.Pos(),
					Name:    "false",
				}
			}
		case token.LOR:
			if mustFalse(d.NodeValue(v.X)) && mustFalse(d.NodeValue(v.Y)) {
				return &ast.Ident{
					NamePos: v.Pos(),
					Name:    "false",
				}
			}
			if mustFalse(d.NodeValue(v.X)) {
				return d.refactorExpr(v.Y)
			}
			if mustFalse(d.NodeValue(v.Y)) {
				return d.refactorExpr(v.X)
			}
			if mustTrue(d.NodeValue(v.X)) || mustTrue(d.NodeValue(v.Y)) {
				return &ast.Ident{
					NamePos: v.Pos(),
					Name:    "true",
				}
			}
		}
		v.X = d.refactorExpr(v.X)
		v.Y = d.refactorExpr(v.Y)

	case *ast.UnaryExpr:
		switch v.Op {
		case token.NOT:
			v.X = d.refactorExpr(v.X)
		}

	case *ast.ParenExpr:
		v.X = d.refactorExpr(v.X)

	case *ast.CallExpr:
		if _, ok := d.ASTExprToSSAValue[v].(*ssa.Call); ok {
			// 确实是调用了函数，且非 build in func
			vals := d.NodeValue(v)
			if len(vals) == 1 && vals[0] != VariantValue {
				val := vals[0]
				switch val.Kind() {
				case constant.String:
					return &ast.BasicLit{
						ValuePos: v.Pos(),
						Kind:     token.STRING,
						Value:    val.ExactString(),
					}
				case constant.Int:
					return &ast.BasicLit{
						ValuePos: v.Pos(),
						Kind:     token.INT,
						Value:    val.ExactString(),
					}
				case constant.Bool:
					return &ast.Ident{
						NamePos: v.Pos(),
						Name:    val.ExactString(),
					}
				}
			}
		}

		var args []ast.Expr
		for _, exp := range v.Args {
			args = append(args, d.refactorExpr(exp))
		}
		v.Args = args
	}

	return expr
}

func cutReturn(stats []ast.Stmt) []ast.Stmt {
	for i, v := range stats {
		if _, ok := v.(*ast.ReturnStmt); ok {
			return stats[:i+1]
		}
	}
	return stats
}

func (d *Detector) deleteUnusedComment(file *ast.File) {
	lines := d.deleteLines[d.position(file.Pos()).Filename]
	if lines == nil {
		return
	}
	var commentGroup []*ast.CommentGroup
c:
	for _, cg := range file.Comments {
		start := d.position(cg.Pos()).Line
		end := d.position(cg.Pos()).Line
		for i := start; i <= end; i++ {
			if lines[i] {
				continue c
			}
		}
		commentGroup = append(commentGroup, cg)
	}

	file.Comments = commentGroup
}
