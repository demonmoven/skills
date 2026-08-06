package internal

import (
	"errors"
	"fmt"
	"go/token"
	"os"
	"strings"

	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/callgraph/cha"
	"golang.org/x/tools/go/callgraph/static"
	"golang.org/x/tools/go/callgraph/vta"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

type option struct {
	test      bool
	buildFlag []string
	fset      *token.FileSet

	static bool
}
type LoadOption func(o *option)

func WithTest() LoadOption {
	return func(o *option) {
		o.test = true
	}
}

func WithBuildTag(tags []string) LoadOption {
	return func(o *option) {
		// "-tags", "unittest"
		o.buildFlag = append(o.buildFlag, "-tags", strings.Join(tags, " "))
	}
}

func WithFileSet() LoadOption {
	return func(o *option) {
		o.fset = token.NewFileSet()
	}
}

func WithStaticAlg() LoadOption {
	return func(o *option) {
		o.static = true
	}
}

// Load builds the ssa program and call graph by VTA algorithm.
func Load(opts ...LoadOption) (prog *ssa.Program, initialPkgs []*packages.Package, pkgs []*ssa.Package, cg *callgraph.Graph, err error) {
	var opt option
	for _, o := range opts {
		o(&opt)
	}

	cfg := &packages.Config{
		Mode:       packages.LoadAllSyntax,
		Dir:        ".",
		Tests:      opt.test,
		BuildFlags: opt.buildFlag,
		Fset:       opt.fset,
	}
	initialPkgs, err = packages.Load(cfg, "./...")
	if err != nil {
		return
	}
	if packages.PrintErrors(initialPkgs) > 0 {
		err = fmt.Errorf("packages contain errors")
		return
	}
	// Create and build SSA-form program representation.
	mode := ssa.InstantiateGenerics | ssa.GlobalDebug // instantiate generics by default for soundness
	prog, pkgs = ssautil.AllPackages(initialPkgs, mode)
	prog.Build()

	if opt.static {
		cg = static.CallGraph(prog)
	} else {
		cg = vta.CallGraph(ssautil.AllFunctions(prog), cha.CallGraph(prog))
	}

	return
}

// LoadPkg x
func LoadPkg(opts ...LoadOption) ([]*packages.Package, error) {
	var opt option
	for _, o := range opts {
		o(&opt)
	}

	cfg := &packages.Config{
		Mode:       packages.LoadAllSyntax,
		Dir:        ".",
		Tests:      opt.test,
		BuildFlags: opt.buildFlag,
		Fset:       opt.fset,
	}
	return packages.Load(cfg, "./...")
}

func GetModuleName() (string, error) {
	_, err := os.Stat("go.mod")
	if err != nil {
		return "", err
	}
	content, err := os.ReadFile("go.mod")
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(content), "\n") {
		if strings.HasPrefix(line, "module") {
			for _, i := range strings.Split(strings.ReplaceAll(line, "\t", " "), " ") {
				if i != "" && i != "module" {
					return i, nil
				}

			}
		}
	}
	return "", errors.New("cannot parse module name")
}
