package passes

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/packages"
)

var CleanRepeatImportAlias = &repeatImportAlias{
	TransformPass{
		name: "clean-repeat-import-alias",
		doc:  `remove repeat import alias`,
	},
}

type repeatImportAlias struct {
	TransformPass
}

func (pass *repeatImportAlias) EnableOnlyOnTypeErrors() bool {
	return true
}

func (pass *repeatImportAlias) Run(pkg *packages.Package) (*TransformResult, error) {
	result := NewResult()

	for _, f := range pkg.Syntax {
		imports := make(map[string]map[*ast.ImportSpec]bool)

		ast.Inspect(f, func(n ast.Node) bool {
			if spec, ok := n.(*ast.ImportSpec); ok {
				if imports[spec.Path.Value] == nil {
					imports[spec.Path.Value] = make(map[*ast.ImportSpec]bool)
				}
				imports[spec.Path.Value][spec] = true
			}
			return true
		})

		removes := make(map[*ast.ImportSpec]bool)
		for _, specs := range imports {
			if len(specs) > 1 {
				removeSameImportWithSameAlias(specs, removes)
				removeSameImportWithNoAlias(specs, removes)
			}
		}

		astutil.Apply(f, func(c *astutil.Cursor) bool {
			if n, ok := c.Node().(*ast.ImportSpec); ok {
				if removes[n] {
					c.Delete()
					result.AddChangedFile(f)
				}
			}
			return true
		}, nil)
	}

	return result, nil
}

// 如果多个的import的alias都一样，那么就删除其中1个
func removeSameImportWithSameAlias(specs map[*ast.ImportSpec]bool, removes map[*ast.ImportSpec]bool) {
	alias := make(map[string][]*ast.ImportSpec)
	for spec := range specs {
		if spec.Name != nil {
			alias[spec.Name.Name] = append(alias[spec.Name.Name], spec)
		}
	}

	for _, samespecs := range alias {
		for i, spec := range samespecs {
			if i != 0 {
				removes[spec] = true
			}
		}
	}
}

// 如果多个import的alias不一样，但是suffix一样，那么就删除没有alias的那个
func removeSameImportWithNoAlias(specs map[*ast.ImportSpec]bool, removes map[*ast.ImportSpec]bool) {
	alias := make(map[string][]*ast.ImportSpec)
	for spec := range specs {
		splits := strings.Split(spec.Path.Value, "/")
		if len(splits) == 0 {
			continue
		}

		suffix := splits[len(splits)-1]
		if suffix != "" {
			alias[suffix] = append(alias[suffix], spec)
		}
	}

	atLeastOneAliasEqualToSuffix := func(spec *ast.ImportSpec, suffix string) bool {
		for _, s := range alias[suffix] {
			if s != spec && s.Name != nil && s.Name.Name == suffix {
				return true
			}
		}
		return false
	}

	var shouldremoves []*ast.ImportSpec
	for suffix, samespecs := range alias {
		for _, spec := range samespecs {
			if spec.Name == nil && atLeastOneAliasEqualToSuffix(spec, suffix) {
				shouldremoves = append(shouldremoves, spec)
			}
		}

		// 如果所有spec都没有alias，那么至少保留一个
		if len(shouldremoves) == len(samespecs) {
			shouldremoves = shouldremoves[1:]
		}

		for _, spec := range shouldremoves {
			removes[spec] = true
		}
	}
}
