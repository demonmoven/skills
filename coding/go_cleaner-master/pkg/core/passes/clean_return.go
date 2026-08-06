package passes

import (
	"go/ast"
	"go/token"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

var CleanAfterReturnPass = &cleanAfterReturn{
	TransformPass{
		name: "clean-after-return",
		doc:  `Remove code and comments after return`,
	},
}

type cleanAfterReturn struct {
	TransformPass
}

func (pass *cleanAfterReturn) EnableOnlyOnTypeErrors() bool {
	return true
}

func (pass *cleanAfterReturn) Run(pkg *packages.Package) (*TransformResult, error) {
	result := NewResult()
	
	for _, file := range pkg.Syntax {
		if fileFilter(pkg.Fset, file) {
			continue
		}

		if changed := pass.removeCodeAfterUnreachable(pkg.Fset, file); changed {
			result.AddChangedFile(file)
		}
	}

	return result, nil
}

func fileFilter(fset *token.FileSet, file *ast.File) bool {
	filename := fset.Position(file.Pos()).Filename
	return filepath.Ext(filename) == ".go" && strings.Contains(filename, "_gen/")
}

func removeNotUsedCommentBetween(fset *token.FileSet, file *ast.File, start token.Pos, end token.Pos) {
	oldEnd := fset.Position(end).Line
	newEnd := fset.Position(start).Line
	for _, group := range file.Comments {
		var tempComments []*ast.Comment
		for _, comment := range group.List {
			cLine := fset.Position(comment.Slash).Line
			if cLine > newEnd && cLine <= oldEnd {
				continue
			}
			tempComments = append(tempComments, comment)
		}
		group.List = tempComments
	}
}

func (pass *cleanAfterReturn) removeCodeAfterUnreachable(fset *token.FileSet, file *ast.File) (changed bool) {
	ast.Inspect(file, func(n ast.Node) bool {
		var oldlist []ast.Stmt
		switch blk := n.(type) {
		case *ast.BlockStmt:
			if len(blk.List) > 1 {
				oldlist = blk.List
			}
		case *ast.CaseClause:
			if len(blk.Body) > 1 {
				oldlist = blk.Body
			}
		case *ast.CommClause:
			if len(blk.Body) > 1 {
				oldlist = blk.Body
			}
		}

		if len(oldlist) == 0 {
			return true
		}

		newlist, retIndex, keepcomments := removeCodeFromStmtList(oldlist)

		// 新的代码块至少需要包含一个return语句
		if len(newlist) == 0 || len(newlist) == len(oldlist) {
			return true
		}

		changed = true
		// 使用新的代码块替换旧的代码块
		var endPosOfRemoving token.Pos
		begPosOfRemoving := oldlist[retIndex].End()
		switch blk := n.(type) {
		case *ast.BlockStmt:
			blk.List = newlist
			endPosOfRemoving = blk.End()
		case *ast.CaseClause:
			blk.Body = newlist
			endPosOfRemoving = oldlist[len(oldlist)-1].End()
		case *ast.CommClause:
			blk.Body = newlist
			endPosOfRemoving = oldlist[len(oldlist)-1].End()
		}

		// 我们假设return和label之间的注释都是有效的，否则均为无效注释
		if !keepcomments {
			removeNotUsedCommentBetween(fset, file, begPosOfRemoving, endPosOfRemoving)
		}

		return true
	})
	return
}

func removeCodeFromStmtList(stmts []ast.Stmt) (newlist []ast.Stmt, begIndex int, keepcomments bool) {
	shouldrm := false
	endIndex := len(stmts)

	// 只删除代码块中第一个return和第一个label之间的代码
	for i := 0; i < endIndex; i++ {
		if _, ok := stmts[i].(*ast.ReturnStmt); ok && !shouldrm {
			begIndex = i
			shouldrm = true
		}

		if expr, ok := stmts[i].(*ast.ExprStmt); ok && !shouldrm {
			call, ok := expr.X.(*ast.CallExpr)
			if !ok {
				continue
			}

			ident, ok := call.Fun.(*ast.Ident)
			if !ok || len(call.Args) != 1 {
				continue
			}

			if ident.Name == "panic" {
				begIndex = i
				shouldrm = true
			}
		}

		if _, ok := stmts[i].(*ast.LabeledStmt); shouldrm && ok {
			endIndex = i
			keepcomments = true
			break
		}
	}

	if !shouldrm {
		return
	}

	for i := 0; i < begIndex+1; i++ {
		newlist = append(newlist, stmts[i])
	}
	for i := endIndex; i < len(stmts); i++ {
		newlist = append(newlist, stmts[i])
	}

	return
}
