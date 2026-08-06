package util

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"code.byted.org/analyzers/go_cleaner/pkg/core/imports"
)

func IterCall(f ast.Node, fn func(expr *ast.CallExpr)) {
	ast.Inspect(f, func(node ast.Node) bool {
		if call, ok := node.(*ast.CallExpr); ok {
			fn(call)
		}
		return true
	})
}

func AllCall(f ast.Node) (call []*ast.CallExpr) {
	IterCall(f, func(expr *ast.CallExpr) {
		call = append(call, expr)
	})
	return
}

func RightestIdent(expr ast.Expr) string {
	switch v := expr.(type) {
	case *ast.SelectorExpr:
		return v.Sel.Name
	case *ast.Ident:
		return v.Name
	default:
		return ""
	}
}

func Save(file *ast.File, fset *token.FileSet) error {
	bf := &bytes.Buffer{}
	if err := printer.Fprint(bf, fset, file); err != nil {
		return err
	}

	data, err := imports.Process(fset.Position(file.Pos()).Filename, bf.Bytes(), nil)
	if err != nil {
		return err
	}

	if canDelSafely(data) {
		return delFile(fset.Position(file.Pos()).Filename)
	}

	err = os.WriteFile(fset.Position(file.Pos()).Filename, data, 0644)
	if err != nil {
		return err
	}

	return nil
}

func delFile(path string) error {
	if err := os.Remove(path); err != nil {
		return err
	}
	path = filepath.Dir(path)
	fs, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	if len(fs) == 0 {
		return delFile(path)
	}

	return nil
}

func canDelSafely(raw []byte) bool {
	var err error

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "", raw, parser.AllErrors|parser.ParseComments)
	if err != nil {
		return false
	}

	del := true

	ast.Inspect(node, func(n ast.Node) bool {
		switch v := n.(type) {
		case *ast.TypeSpec, *ast.ValueSpec:
			del = false
			return false
		case *ast.FuncDecl:
			del = false
			return false
		case *ast.ImportSpec:
			if v.Name != nil && v.Name.Name == "_" {
				del = false
				return false
			}
		default:
		}
		return true
	})

	return del
}

func HasErrors(imports []*ast.ImportSpec) (string, bool) {
	for _, imp := range imports {
		if strings.Trim(imp.Path.Value, `"`) == "errors" {
			name := "errors"
			if imp.Name != nil && imp.Name.Name != "" {
				name = imp.Name.Name
			}
			return name, true
		}
	}
	return "", false
}

func MayErrorsConflict(imports []*ast.ImportSpec) bool {
	for _, imp := range imports {
		_, last := filepath.Split(strings.Trim(imp.Path.Value, `"`))
		if last == "errors" {
			return true
		}
		if imp.Name != nil && imp.Name.Name == "errors" {
			return true
		}
	}

	return false
}

func AddImport(file *ast.File, imp *ast.ImportSpec) {
	importDecl := &ast.GenDecl{
		Tok:   token.IMPORT,
		Specs: []ast.Spec{imp},
	}

	if len(file.Decls) > 0 {
		if decl, ok := file.Decls[0].(*ast.GenDecl); ok && decl.Tok == token.IMPORT {
			decl.Specs = append([]ast.Spec{imp}, decl.Specs...)
		} else {
			file.Decls = append([]ast.Decl{importDecl}, file.Decls...)
		}
	} else {
		file.Decls = append(file.Decls, importDecl)
	}
}

func StmtsInProgramOrder(fn *ast.FuncDecl) (stmts []ast.Stmt) {
	var hasControlFlow bool
	ast.Inspect(fn, func(n ast.Node) bool {
		switch v := n.(type) {
		case *ast.IfStmt, *ast.BranchStmt, *ast.ForStmt:
			hasControlFlow = true
			return false
		case *ast.RangeStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt:
			hasControlFlow = true
			return false
		case *ast.GoStmt, *ast.DeferStmt, *ast.SelectStmt:
			hasControlFlow = true
			return false
		case ast.Stmt:
			stmts = append(stmts, v)
		}
		return true
	})

	if hasControlFlow {
		return []ast.Stmt{}
	}

	return
}

func IsVariableStrongUpdatedBetween(stmts []ast.Stmt, variable string, start int, end int) bool {
	for i := start + 1; i <= end; i++ {
		assign, ok := stmts[i].(*ast.AssignStmt)
		if !ok {
			continue
		}

		for _, lhs := range assign.Lhs {
			if lhsIdent, ok := lhs.(*ast.Ident); ok && lhsIdent.Name == variable {
				return true
			}
		}
	}

	return false
}
