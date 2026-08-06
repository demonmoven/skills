package passes

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/types"

	"code.byted.org/analyzers/go_cleaner/pkg/core/repoinfo"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/packages"
)

var CleanUnusedImportPass = &cleanUnusedImport{
	TransformPass{
		name: "clean-unused-import",
		doc:  `check for unused imports and rewrite fixes`,
	},
}

type cleanUnusedImport struct {
	TransformPass
}

func (pass *cleanUnusedImport) EnableOnlyOnTypeErrors() bool {
	return true
}

func (pass *cleanUnusedImport) Run(pkg *packages.Package) (*TransformResult, error) {
	result := NewResult()

	for _, typeErr := range pkg.TypeErrors {
		for _, re := range repoinfo.UnusedImportRegexp {
			match := re.FindStringSubmatch(typeErr.Msg)
			if len(match) > 0 {
				err := pass.runForError(pkg, typeErr, match, result)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	return result, nil
}

func (pass *cleanUnusedImport) runForError(pkg *packages.Package, err types.Error, names []string, result *TransformResult) error {
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

	isThisImport := func() bool {
		lit, ok := path[0].(*ast.BasicLit)
		return ok && lit.Value == names[1]
	}

	// import as x
	isThisImportAsAlias := func() bool {
		spec, ok := path[1].(*ast.ImportSpec)
		return ok && spec.Name != nil && spec.Name.Name == names[2]
	}

	if !isThisImport() && !isThisImportAsAlias() {
		return nil
	}

	for i := range path {
		switch imp := path[i].(type) {
		case *ast.ImportSpec:
			// Find GenDecl to which offending ValueSpec belongs.
			if decl, ok := path[i+1].(*ast.GenDecl); ok {
				fixes := pass.removeImportFromSpec(pkg, path, imp, decl)
				// fixes may be nil
				if len(fixes) > 0 {
					result.AddChangedFile(file)
				}
			}
		}
	}

	return nil
}

func (pass *cleanUnusedImport) removeImportFromSpec(pkg *packages.Package, path []ast.Node, imp *ast.ImportSpec, decl *ast.GenDecl) []analysis.SuggestedFix {
	newDecl := new(ast.GenDecl)
	*newDecl = *decl
	newDecl.Specs = nil

	for _, spec := range decl.Specs {
		if spec != imp {
			newDecl.Specs = append(newDecl.Specs, spec)
			continue
		}
	}

	// decl.End() does not include any comments, so if a comment is present we
	// need to account for it when we delete the package
	end := decl.End()
	if imp.Comment != nil && imp.Comment.End() > end {
		end = imp.Comment.End()
	}

	// There are no other specs left in the import, the whole import can
	// be deleted
	if len(newDecl.Specs) == 0 {
		for _, node := range path {
			if file, ok := node.(*ast.File); ok {
				fixes := deleteImportFromFile(file, decl)
				if len(fixes) == 0 {
					return nil // can this happen?
				}
				return fixes
			}
		}
	}

	var b bytes.Buffer
	if err := format.Node(&b, pkg.Fset, newDecl); err != nil {
		return nil
	}

	// 真正的重写
	var newspecs []ast.Spec
	for _, spec := range decl.Specs {
		if spec != imp {
			newspecs = append(newspecs, spec)
		}
	}
	decl.Specs = newspecs

	return []analysis.SuggestedFix{
		{
			Message: "Remove unused import " + imp.Path.Value,
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

func deleteImportFromFile(file *ast.File, decl *ast.GenDecl) (fixes []analysis.SuggestedFix) {
	var newdecls []ast.Decl
	for _, d := range file.Decls {
		if d != decl {
			newdecls = append(newdecls, d)
		}
	}

	fixes = append(fixes, analysis.SuggestedFix{
		Message: "Remove import",
		TextEdits: []analysis.TextEdit{
			{
				Pos:     decl.Pos(),
				End:     decl.End(),
				NewText: []byte(""),
			},
		},
	})
	return
}
