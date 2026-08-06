package repoinfo

import (
	"path/filepath"

	"code.byted.org/lang/gg/gmap"
	"golang.org/x/tools/go/packages"
)

func GetSyntaxUsageOfModule(p *Project, module string) *SyntaxUsage {
	s := &SyntaxUsage{
		Dirs:    p.FilterOutModDirs(module),
		Pattern: module,
		Usage:   map[string]map[string]struct{}{},
	}

	if err := s.Parse(); err != nil {
		panic(err)
	}

	return s
}

func GetSyntaxUsageOfPackage(pkg string, otherpkgs []*packages.Package) *SyntaxUsage {
	dirs := make(map[string]struct{})

	for _, p := range otherpkgs {
		for _, f := range p.GoFiles {
			dirs[filepath.Dir(f)] = struct{}{}
		}
	}

	s := &SyntaxUsage{
		Dirs:    gmap.Keys(dirs),
		Pattern: pkg,
		Usage:   map[string]map[string]struct{}{},
	}

	if err := s.Parse(); err != nil {
		panic(err)
	}

	return s
}
