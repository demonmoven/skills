package passes

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"go/types"
	"strings"

	"code.byted.org/analyzers/go_cleaner/pkg/core/repoinfo"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/packages"
)

var CleanUnusedDeclPass = &cleanUnusedDeclarations{
	TransformPass{
		name: "clean-unused-decl",
		doc:  `check for unused variables and rewrite fixes`,
	},
}

type cleanUnusedDeclarations struct {
	TransformPass
}

func (pass *cleanUnusedDeclarations) EnableOnlyOnTypeErrors() bool {
	return true
}

func (pass *cleanUnusedDeclarations) Run(pkg *packages.Package) (*TransformResult, error) {
	result := NewResult()

	for _, typeErr := range pkg.TypeErrors {
		for _, re := range repoinfo.UnusedVariableRegexp {
			match := re.FindStringSubmatch(typeErr.Msg)
			if len(match) > 0 {
				varName := match[1]
				// Beginning in Go 1.23, go/types began quoting vars as `v'.
				varName = strings.Trim(varName, "'`'")

				err := pass.runForError(pkg, typeErr, varName, result)
				if err != nil {
					return nil, err
				}
				pass.removeUnderscoreIdentIfNeeded(pkg)
			}
		}
	}
	return result, nil
}

func (pass *cleanUnusedDeclarations) runForError(pkg *packages.Package, err types.Error, name string, result *TransformResult) error {
	var file *ast.File
	for _, f := range pkg.Syntax {
		if f.Pos() <= err.Pos && err.Pos < f.End() {
			file = f
			break
		}
	}
	if file == nil {
		return nil
	}

	path, _ := astutil.PathEnclosingInterval(file, err.Pos, err.Pos)
	if len(path) < 2 {
		return nil
	}

	ident, ok := path[0].(*ast.Ident)
	if !ok || ident.Name != name {
		return nil
	}

	for i := range path {
		switch stmt := path[i].(type) {
		case *ast.ValueSpec:
			// Find GenDecl to which offending ValueSpec belongs.
			if decl, ok := path[i+1].(*ast.GenDecl); ok {
				fixes := pass.removeVariableFromSpec(pkg, path, stmt, decl, ident)
				// fixes may be nil
				if len(fixes) > 0 {
					result.AddChangedFile(file)
				}
			}

		case *ast.AssignStmt:
			if stmt.Tok != token.DEFINE {
				continue
			}

			containsIdent := false
			for _, expr := range stmt.Lhs {
				if expr == ident {
					containsIdent = true
				}
			}
			if !containsIdent {
				continue
			}

			fixes := removeVariableFromAssignment(path, stmt, ident)
			// fixes may be nil
			if len(fixes) > 0 {
				result.AddChangedFile(file)
			}
		}
	}

	return nil
}

func (pass *cleanUnusedDeclarations) removeVariableFromSpec(pkg *packages.Package, path []ast.Node, stmt *ast.ValueSpec, decl *ast.GenDecl, ident *ast.Ident) []analysis.SuggestedFix {
	newDecl := new(ast.GenDecl)
	*newDecl = *decl
	newDecl.Specs = nil

	for _, spec := range decl.Specs {
		if spec != stmt {
			newDecl.Specs = append(newDecl.Specs, spec)
			continue
		}

		newSpec := new(ast.ValueSpec)
		*newSpec = *stmt
		newSpec.Names = nil

		for _, n := range stmt.Names {
			if n != ident {
				newSpec.Names = append(newSpec.Names, n)
			}
		}

		if len(newSpec.Names) > 0 {
			newDecl.Specs = append(newDecl.Specs, newSpec)
		}
	}

	// decl.End() does not include any comments, so if a comment is present we
	// need to account for it when we delete the statement
	end := decl.End()
	if stmt.Comment != nil && stmt.Comment.End() > end {
		end = stmt.Comment.End()
	}

	// There are no other specs left in the declaration, the whole statement can
	// be deleted
	if len(newDecl.Specs) == 0 {
		// Find parent DeclStmt and delete it
		for _, node := range path {
			if declStmt, ok := node.(*ast.DeclStmt); ok {
				edits := deleteStmtFromBlock(path, declStmt)
				if len(edits) == 0 {
					return nil // can this happen?
				}
				return []analysis.SuggestedFix{
					{
						Message:   suggestedFixMessage(ident.Name),
						TextEdits: edits,
					},
				}
			}
		}
	}

	var b bytes.Buffer
	if err := format.Node(&b, pkg.Fset, newDecl); err != nil {
		return nil
	}

	for _, node := range path {
		if declStmt, ok := node.(*ast.DeclStmt); ok {
			declStmt.Decl = newDecl
			break
		}
	}

	return []analysis.SuggestedFix{
		{
			Message: suggestedFixMessage(ident.Name),
			TextEdits: []analysis.TextEdit{
				{
					Pos: decl.Pos(),
					// Avoid adding a new empty line
					End:     end + 1,
					NewText: b.Bytes(),
				},
			},
		},
	}
}

func (pass *cleanUnusedDeclarations) removeUnderscoreIdentIfNeeded(pkg *packages.Package) {
	for _, file := range pkg.Syntax {
		ast.Inspect(file, func(node ast.Node) bool {
			assignment, ok := node.(*ast.AssignStmt)
			if !ok {
				return true
			}

			isAllIdentsTurnIntoUnderscore := true
			for _, expr := range assignment.Lhs {
				if ident, ok := expr.(*ast.Ident); ok {
					if ident.Name != "_" {
						isAllIdentsTurnIntoUnderscore = false
					}
				}
			}

			if isAllIdentsTurnIntoUnderscore {
				assignment.Tok = token.ASSIGN
			}

			return true
		})
	}
}

func removeVariableFromAssignment(path []ast.Node, stmt *ast.AssignStmt, ident *ast.Ident) []analysis.SuggestedFix {
	// The only variable in the assignment is unused
	if len(stmt.Lhs) == 1 {
		// If LHS has only one expression to be valid it has to have 1 expression
		// on RHS
		//
		// RHS may have side effects, preserve RHS
		if exprMayHaveSideEffects(stmt.Rhs[0]) {
			// 这里我们使用_也不会出错，避免将ast.AssignStmt
			// 变成其他语句带来的复杂性
			ident.Name = "_"

			// Delete until RHS
			return []analysis.SuggestedFix{
				{
					Message: suggestedFixMessage(ident.Name),
					TextEdits: []analysis.TextEdit{
						{
							Pos: ident.Pos(),
							End: stmt.Rhs[0].Pos(),
						},
					},
				},
			}
		}

		// RHS does not have any side effects, delete the whole statement
		edits := deleteStmtFromBlock(path, stmt)
		if len(edits) == 0 {
			return nil // can this happen?
		}
		return []analysis.SuggestedFix{
			{
				Message:   suggestedFixMessage(ident.Name),
				TextEdits: edits,
			},
		}
	}

	// Otherwise replace ident with `_`
	ident.Name = "_"
	return []analysis.SuggestedFix{
		{
			Message: suggestedFixMessage(ident.Name),
			TextEdits: []analysis.TextEdit{
				{
					Pos:     ident.Pos(),
					End:     ident.End(),
					NewText: []byte("_"),
				},
			},
		},
	}
}

func suggestedFixMessage(name string) string {
	return fmt.Sprintf("Remove variable %s", name)
}

func deleteStmtFromBlock(path []ast.Node, stmt ast.Stmt) []analysis.TextEdit {
	// Find innermost enclosing BlockStmt.
	var block *ast.BlockStmt
	for i := range path {
		if blockStmt, ok := path[i].(*ast.BlockStmt); ok {
			block = blockStmt
			break
		}
	}

	nodeIndex := -1
	for i, blockStmt := range block.List {
		if blockStmt == stmt {
			nodeIndex = i
			break
		}
	}

	// The statement we need to delete was not found in BlockStmt
	if nodeIndex == -1 {
		return nil
	}

	// Delete until the end of the block unless there is another statement after
	// the one we are trying to delete
	end := block.Rbrace
	if nodeIndex < len(block.List)-1 {
		end = block.List[nodeIndex+1].Pos()
	}

	if nodeIndex == len(block.List)-1 {
		block.List = block.List[:nodeIndex]
	} else {
		block.List = append(block.List[:nodeIndex], block.List[nodeIndex+1:]...)
	}

	return []analysis.TextEdit{
		{
			Pos: stmt.Pos(),
			End: end,
		},
	}
}

// exprMayHaveSideEffects reports whether the expression may have side effects
// (because it contains a function call or channel receive). We disregard
// runtime panics as well written programs should not encounter them.
func exprMayHaveSideEffects(expr ast.Expr) bool {
	var mayHaveSideEffects bool
	ast.Inspect(expr, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.CallExpr: // possible function call
			mayHaveSideEffects = true
			return false
		case *ast.UnaryExpr:
			if n.Op == token.ARROW { // channel receive
				mayHaveSideEffects = true
				return false
			}
		case *ast.FuncLit:
			return false // evaluating what's inside a FuncLit has no effect
		}
		return true
	})

	return mayHaveSideEffects
}
