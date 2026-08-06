package internal

import (
	"bytes"
	"errors"
	"go/types"
	"os"

	"code.byted.org/analyzers/go_cleaner/pkg/core/repoinfo"
	"code.byted.org/lang/gg/gslice"
	"golang.org/x/tools/go/ssa"
)

type items map[string]bool

type ImportBy struct {
	raw map[string]items
	pkg map[*types.Package]items
}

func (b *ImportBy) addImportByOtherRepo(source string) error {
	raw, err := os.ReadFile(source)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	for _, line := range bytes.Split(raw, []byte{'\n'}) {
		if vs := bytes.Split(line, []byte{','}); len(vs) > 1 {
			pkg, obj := string(vs[0]), string(vs[1])
			if _, ok := b.raw[pkg]; !ok {
				b.raw[pkg] = map[string]bool{}
			}
			b.raw[pkg][obj] = true
		}
	}

	return nil
}

func (b *ImportBy) addImportByMonoRepo(project *repoinfo.Project, module string) {
	synref := repoinfo.GetSyntaxUsageOfModule(project, module)
	for pkg, refs := range synref.Usage {
		if _, ok := b.raw[pkg]; !ok {
			b.raw[pkg] = map[string]bool{}
		}

		for obj := range refs {
			b.raw[pkg][obj] = true
		}
	}
}

func (b *ImportBy) addImportByNonMainPackages(module *repoinfo.Module) {
	otherpkgs := gslice.Diff(module.AllPackages, module.Packages)

	for _, pkg := range module.Packages {
		synref := repoinfo.GetSyntaxUsageOfPackage(pkg.Name, otherpkgs)
		for pkg, refs := range synref.Usage {
			if _, ok := b.raw[pkg]; !ok {
				b.raw[pkg] = map[string]bool{}
			}
	
			for obj := range refs {
				b.raw[pkg][obj] = true
			}
		}
	}
}

func NewImportBy(source string, module *repoinfo.Module) (*ImportBy, error) {
	v := &ImportBy{
		raw: make(map[string]items),
		pkg: make(map[*types.Package]items),
	}

	if source != "" {
		v.addImportByOtherRepo(source)
	}

	if module.Project.IsMultiModules {
		v.addImportByMonoRepo(module.Project, module.Name)
	}

	if module.LoadMainPkgsOnly {
		v.addImportByNonMainPackages(module)
	}

	return v, nil
}

func (b *ImportBy) Import(obj types.Object) bool {
	if obj == nil || obj.Pkg() == nil {
		return false
	}

	if ok := b.fast(obj); ok {
		return true
	}

	return b.slow(obj)
}

func (b *ImportBy) fast(obj types.Object) bool {
	if v, ok := b.pkg[obj.Pkg()]; ok {
		return v[obj.Name()]
	}

	return false
}

func (b *ImportBy) slow(obj types.Object) bool {
	if v, ok := b.raw[obj.Pkg().Path()]; !ok || len(v) == 0 {
		b.pkg[obj.Pkg()] = map[string]bool{}
		return false
	} else {
		b.pkg[obj.Pkg()] = v
		return v[obj.Name()]
	}
}

func (b *ImportBy) ImportF(fn *ssa.Function) bool {
	if fn == nil {
		return false
	}
	obj := fn.Object()

	return b.Import(obj)
}
