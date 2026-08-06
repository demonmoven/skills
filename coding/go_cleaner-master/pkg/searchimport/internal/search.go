package internal

import (
	"fmt"
	"go/types"
	"os"
	"regexp"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
)

type Detector struct {
	matcher *packageMatcher

	targetModule string

	prog       *ssa.Program
	pkgs       []*ssa.Package
	initialPkg []*packages.Package

	matchedO map[types.Object]bool

	opt *Opt
}

func NewDetector(opt *Opt) *Detector {
	return &Detector{
		matcher: newPackageMatcher(opt.TargetModule),

		prog: nil,
		pkgs: nil,

		matchedO: make(map[types.Object]bool),
		opt:      opt,
	}
}

func (d *Detector) Load() {
	var err error
	var opts []LoadOption
	opts = append(opts, WithFileSet())
	if d.opt.TestMode != "none" {
		opts = append(opts, WithTest())
	}
	if len(d.opt.Tag) > 0 {
		opts = append(opts, WithBuildTag(d.opt.Tag))
	}
	if d.opt.IgnoreTestErr {
		opts = append(opts, WithIgnoreTestErr())
	}
	if d.opt.CalCallGraph {
		opts = append(opts, WithCallGraph())
	}

	prog, initialPkg, pkgs, _, err := Load(opts...)
	if err != nil {
		fmt.Println("fail, err:", err)
		panic(err)
	}
	d.prog, d.pkgs, d.initialPkg = prog, pkgs, initialPkg
}

// SearchImport 收集所有使用了targetModule的types.Object
func (d *Detector) SearchImport() {
	for _, pkg := range d.prog.AllPackages() {
		d.matcher.add3(pkg)
		d.matcher.add1(pkg.Pkg)
	}

	for _, pkg := range d.initialPkg {
		if pkg.TypesInfo == nil {
			continue
		}
		for _, o := range pkg.TypesInfo.Uses {
			if d.matcher.hit1(o.Pkg()) {
				d.matchedO[o] = true
			}
		}
	}
}

func (d *Detector) Print() error {
	w := os.Stdout

	if f := d.opt.OutFile; f != "" {
		wf, err := os.OpenFile(f, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			return err
		}

		w = wf
	}

	v := map[string]struct{}{}

	for o := range d.matchedO {
		v[fmt.Sprintf("%s,%s\n", o.Pkg().Path(), o.Name())] = struct{}{}
	}

	for o := range v {
		if _, err := w.WriteString(o); err != nil {
			return err
		}

	}

	return nil
}

type Opt struct {
	TestMode string
	Tag      []string

	ExcludePath   []string
	excludePathRe *regexp.Regexp

	OnlyPath   []string
	onlyPathRe *regexp.Regexp

	Force bool

	IgnoreMock bool

	Before int

	SimilarConstKeep    bool
	SimilarConstComment bool
	SimilarFuncKeep     bool

	DeleteUnusedGlobalAnonymousV bool

	UnsafeDeleteMethod bool

	CalCallGraph bool

	IgnoreTestErr bool

	TargetBranch string

	TargetModule string
	OutFile      string
}
