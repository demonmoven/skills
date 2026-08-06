package passes

import (
	"fmt"
	"go/ast"
	"go/types"

	"code.byted.org/analyzers/go_cleaner/pkg/core/repoinfo"
	"golang.org/x/tools/go/packages"
)

var CleanUnusedFieldPass = &fieldCleaner{
	TransformPass: TransformPass{
		name: "clean-unused-fields",
		doc:  `check for unused fields and rewrite`,
	},
	defs: map[types.Object]*Field{},
	uses: map[types.Object]bool{},
}

type Field struct {
	field  *ast.Field
	obj    types.Object
	tyspec *ast.TypeSpec
	file   *ast.File
}

func newField(path []ast.Node, obj types.Object) *Field {
	f := &Field{
		obj: obj,
	}

	for _, n := range path {
		if file, ok := n.(*ast.File); ok {
			f.file = file
		}
	}

	for _, n := range path {
		if field, ok := n.(*ast.Field); ok && len(field.Names) > 0 {
			f.field = field

			// 避免导出字段在反射中被使用导致的误删
			for _, ident := range f.field.Names {
				if ident.IsExported() {
					return nil
				}
			}
		}

		if tyspec, ok := n.(*ast.TypeSpec); ok {
			f.tyspec = tyspec
			break
		}
	}

	if f.field == nil || f.tyspec == nil || f.file == nil {
		return nil
	}
	return f
}

func (f *Field) eraseFromParent() bool {
	if f.tyspec == nil {
		return false
	}

	var list []*ast.Field
	fmt.Printf("CleanUnusedFieldPass erase field %s", f.field.Names[0].Name)
	st, ok := f.tyspec.Type.(*ast.StructType)
	if !ok {
		return false
	}

	var oldsize int
	var newsize int
	oldsize = len(st.Fields.List)
	for _, spec := range st.Fields.List {
		if spec != f.field {
			list = append(list, spec)
		}
	}
	newsize = len(list)
	st.Fields.List = list
	return oldsize > newsize
}

type fieldCleaner struct {
	TransformPass

	defs map[types.Object]*Field
	uses map[types.Object]bool
}

func (pass *fieldCleaner) Run(module *repoinfo.Module) (*TransformResult, error) {
	result := NewResult()

	pass.defs = make(map[types.Object]*Field)
	pass.uses = make(map[types.Object]bool)

	_ = func(path []ast.Node) bool {
		for _, n := range path {
			if _, ok := n.(*ast.FuncDecl); ok {
				return true
			}
		}
		return false
	}

	for _, p := range module.Packages {
		pkg := &repoinfo.Package{
			Pkg:      p,
			TypeInfo: p.Types,
		}

		for ident, o := range p.TypesInfo.Defs {
			path, _ := pkg.Find(ident.Pos())
			if field := newField(path, o); field != nil {
				pass.defs[o] = field
			}
		}
	}

	pass.collectUses(module.Packages)

	unused := make(map[*Field]bool)
	for def, field := range pass.defs {
		if !pass.uses[def] {
			unused[field] = true
		}
	}

	for field := range unused {
		if field.eraseFromParent() {
			result.AddChangedFile(field.file)
		}
	}

	return result, nil
}

func (pass *fieldCleaner) collectUses(pkgs []*packages.Package) {
	collect := func(p *types.Info, f *ast.File) {
		ast.Inspect(f, func(n ast.Node) bool {
			if _, ok := n.(*ast.TypeSpec); ok {
				return false
			}

			if ident, ok := n.(*ast.Ident); ok {
				if obj := p.ObjectOf(ident); obj != nil {
					pass.uses[obj] = true
				}
			}

			return true
		})
	}

	for _, pkg := range pkgs {
		for _, f := range pkg.Syntax {
			collect(pkg.TypesInfo, f)
		}
	}
}
