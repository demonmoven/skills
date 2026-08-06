package passes

import (
	"fmt"
	"go/ast"

	"code.byted.org/analyzers/go_cleaner/pkg/core/repoinfo"
	"golang.org/x/tools/go/packages"
)

var RenameRedeclMains = &renameRedeclMains{
	TransformPass{
		name: "rename-redecl-mains",
		doc:  `rename redeclation main functions in the same package`,
	},
}

type renameRedeclMains struct {
	TransformPass
}

func (pass *renameRedeclMains) EnableOnlyOnTypeErrors() bool {
	return true
}

func (pass *renameRedeclMains) Run(pkg *packages.Package) (*TransformResult, error) {
	result := NewResult()
	for _, typeErr := range pkg.TypeErrors {
		for _, re := range repoinfo.RedeclaredMains {
			match := re.FindStringSubmatch(typeErr.Msg)
			if len(match) > 0 {
				pass.runForError(pkg, result)
			}
		}
	}

	return result, nil
}

func (pass *renameRedeclMains) runForError(pkg *packages.Package, result *TransformResult) {
	var count int
	for _, f := range pkg.Syntax {
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok {
				continue
			}

			if fn.Name.Name == "main" {
				fn.Name.Name = fmt.Sprintf("%s_%d", repoinfo.RenamedMainByCleanerPrefix, count)
				count += 1
				result.AddChangedFile(f)
			}
		}
	}
}
