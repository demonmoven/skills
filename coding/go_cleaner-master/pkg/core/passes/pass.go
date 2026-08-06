package passes

import (
	"go/ast"

	"code.byted.org/analyzers/go_cleaner/pkg/core/repoinfo"
	"golang.org/x/tools/go/packages"
)

type Pass interface {
	Name() string
	Doc() string
}

type ModulePass interface {
	Pass
	Run(mod *repoinfo.Module) (*TransformResult, error)
}

type PackagePass interface {
	Pass
	Run(pkg *packages.Package) (*TransformResult, error)
	EnableOnlyOnTypeErrors() bool
}

type TransformPass struct {
	name string
	doc  string
}

func (pass *TransformPass) Name() string {
	return pass.name
}

func (pass *TransformPass) Doc() string {
	return pass.doc
}

type TransformResult struct {
	IsChanged     bool
	ChangedSyntax map[*ast.File]bool
}

func NewResult() *TransformResult {
	return &TransformResult{
		IsChanged:     false,
		ChangedSyntax: make(map[*ast.File]bool),
	}
}

func (r *TransformResult) AddChangedFile(f *ast.File) {
	r.IsChanged = true
	r.ChangedSyntax[f] = true
}

var orderedDefaultPasses = []Pass{
	CleanAfterReturnPass,
	CleanUnusedDeclPass,
	CleanUnusedImportPass,
	RenameRedeclMains,
}

func GetAllPass() (allpasses []Pass) {
	allpasses = append(allpasses, orderedDefaultPasses...)
	allpasses = append(allpasses, CleanUnusedFieldPass)
	return
}

func GetDefaultPassNames() (names []string) {
	for _, p := range orderedDefaultPasses {
		names = append(names, p.Name())
	}
	return
}

func FilerKnownPasses(passnames []string) (enabledPasses []Pass, unknowns []string) {
	if len(passnames) == 1 {
		switch passnames[0] {
		case "all":
			enabledPasses = orderedDefaultPasses
			return
		case "deep":
			enabledPasses = GetAllPass()
			return
		}
	}

	for _, name := range passnames {
		var found Pass
		for _, p := range GetAllPass() {
			if p.Name() == name {
				found = p
			}
		}

		if found != nil {
			enabledPasses = append(enabledPasses, found)
		} else {
			unknowns = append(unknowns, name)
		}
	}
	return
}
