package internal

import (
	"bufio"
	"errors"
	"fmt"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/callgraph/cha"
	"golang.org/x/tools/go/callgraph/vta"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

// 全局变量，存在语法错误的测试文件
var errTestOverlay map[string][]byte = nil

type option struct {
	test      bool
	buildFlag []string
	fset      *token.FileSet

	callGraph     bool
	ignoreTestErr bool
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

func WithCallGraph() LoadOption {
	return func(o *option) {
		o.callGraph = true
	}
}

func WithIgnoreTestErr() LoadOption {
	return func(o *option) {
		o.ignoreTestErr = true
	}
}

// Load builds the ssa program and call graph by VTA algorithm.
func Load(opts ...LoadOption) (prog *ssa.Program, initialPkgs []*packages.Package, pkgs []*ssa.Package, cg *callgraph.Graph, err error) {
	var opt option
	for _, o := range opts {
		o(&opt)
	}

	logrus.Info("Load-loadPackage start...")
	initialPkgs, err = loadPackage(opts...)
	logrus.Info("Load-loadPackage done!")
	if err != nil {
		return
	}
	if packages.PrintErrors(initialPkgs) > 0 {
		err = fmt.Errorf("packages contain errors")
		return
	}
	// Create and build SSA-form program representation.
	mode := ssa.InstantiateGenerics // instantiate generics by default for soundness
	logrus.Info("Load-ssautil.AllPackages start...")
	prog, pkgs = ssautil.AllPackages(initialPkgs, mode)
	logrus.Info("Load-ssautil.AllPackages done!")

	logrus.Info("Load-prog.Build start...")
	prog.Build()
	logrus.Info("Load-prog.Build done!")

	if opt.callGraph {
		cg = vta.CallGraph(ssautil.AllFunctions(prog), cha.CallGraph(prog))
	}

	return
}

func loadPackage(opts ...LoadOption) (initialPkgs []*packages.Package, err error) {
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
		Logf:       logrus.Infof,
	}
	pkgs, err := packages.Load(cfg, "./...")

	// 排除存在错误的test文件
	// 只会在第一次加载时执行一次
	if opt.ignoreTestErr && errTestOverlay == nil {
		// 保证只会执行一次
		errTestOverlay = map[string][]byte{}

		overlay, err := gatherErrTestFile(pkgs)
		if err != nil {
			return pkgs, fmt.Errorf("gatherErrTestFile err: %w", err)
		}

		if len(overlay) > 0 {
			errTestOverlay = overlay
			return loadPackage(opts...)
		}

	}

	return pkgs, err
}

func gatherErrTestFile(pkgs []*packages.Package) (overLay map[string][]byte, err error) {
	overLay = make(map[string][]byte)
	for _, p := range pkgs {
		for _, e := range p.Errors {
			seps := strings.Split(e.Pos, ":")
			if len(seps) > 0 && strings.HasSuffix(seps[0], "_test.go") {
				file := seps[0]
				line, err := firstLine(file)
				if err != nil {
					return nil, err
				}
				overLay[file] = line
			}
		}
	}
	return overLay, nil
}

func GetModuleName(path string) (string, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	goModPath := filepath.Join(path, "go.mod")
	_, err = os.Stat(goModPath)
	if err != nil {
		if path != "/" && filepath.Dir(path) != path {
			return GetModuleName(filepath.Dir(path))
		}
		return "", err
	}
	content, err := os.ReadFile(goModPath)
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

func firstLine(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		return scanner.Bytes(), nil
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return nil, nil
}
