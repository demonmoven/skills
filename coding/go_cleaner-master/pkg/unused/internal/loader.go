package internal

import (
	"errors"
	"fmt"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"code.byted.org/analyzers/go_cleaner/pkg/core"
	"code.byted.org/analyzers/go_cleaner/pkg/core/logger"
	"code.byted.org/analyzers/go_cleaner/pkg/core/repoinfo"
	"github.com/sirupsen/logrus"
	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

var optCache = struct {
	overlay map[string][]byte
}{}

type option struct {
	test      bool
	buildFlag []string
	fset      *token.FileSet

	callGraph                      bool
	ignoreTestErr                  bool
	maxNumsOfReloadingWithTestErrs int
	loadMainPkgsOnly               bool

	overlay map[string][]byte
	loadDir string
}
type LoadOption func(o *option)

func WithTest() LoadOption {
	return func(o *option) {
		o.test = true
	}
}

func WithBuildFlags(buildFlags []string, tags []string) LoadOption {
	return func(o *option) {
		o.buildFlag = append(o.buildFlag, buildFlags...)
		// "-tags", "unittest"
		if len(tags) > 0 {
			o.buildFlag = append(o.buildFlag, "-tags", strings.Join(tags, " "))
		}
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

func WithMaxNumsOfReloadingWithTestErrs(n int) LoadOption {
	return func(o *option) {
		o.maxNumsOfReloadingWithTestErrs = n
	}
}

func WithLoadMainPkgsOnly() LoadOption {
	return func(o *option) {
		o.loadMainPkgsOnly = true
	}
}

func WithOverLay(overlay map[string][]byte) LoadOption {
	return func(o *option) {
		o.overlay = overlay
	}
}

func WithCachedOverlay() LoadOption {
	return func(o *option) {
		if len(optCache.overlay) > 0 {
			o.overlay = optCache.overlay
		}
	}
}

func WithLoadDir(dir string) LoadOption {
	return func(o *option) {
		o.loadDir = dir
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
		logrus.Errorf("package loading internal error: %v", err)
		return
	}

	if core.PrintErrAfterClean(initialPkgs) > 0 {
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
		cg, err = repoinfo.CallGraph(prog)
	}

	return
}

func loadPackage(opts ...LoadOption) (pkgs []*packages.Package, err error) {
	defer func() {
		pkgs = core.FilterPkgsNoErrBeforeClean(pkgs)
	}()

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
		Overlay:    opt.overlay,
	}
	if opt.loadDir != "" {
		cfg.Dir = opt.loadDir
		logrus.Infof("LoadPackage with load dir: %s", opt.loadDir)
	}

	load := func() ([]*packages.Package, error) {
		initial, err := packages.Load(cfg, repoinfo.GetQueriesIgnoredByGoList(cfg.Dir)...)
		if err == nil && opt.loadMainPkgsOnly {
			return repoinfo.GetPkgsUsedByMain(initial), nil
		}
		return initial, err
	}

	logrus.Infof("package loader options (%s):\n%s", cfg.Dir, logger.PrettyPackagesConfigJSON(cfg))

	pkgs, err = load()
	if !opt.ignoreTestErr {
		return
	}

	// 排除存在错误的test文件, 将含有语法错误的test文件存入全局变量optCache，
	// 后续就会自动排除这些test文件
	optCache.overlay = make(map[string][]byte)
	cfg.Overlay = optCache.overlay
	haveToCheckReloading := true
	// 如果发现有错误文件被搜集到，则进行重新加载，直到没有错误文件或者达到最大尝试次数
	for i := 0; haveToCheckReloading && i < opt.maxNumsOfReloadingWithTestErrs+1; i++ {
		// 此分支必然不会在第一次循环时命中
		if len(optCache.overlay) > 0 {
			if pkgs, err = load(); err != nil {
				return
			}
		}

		// 检查测试文件是否存在编译错误
		overlay, err := gatherErrTestFile(pkgs)
		if err != nil {
			return pkgs, fmt.Errorf("gatherErrTestFile err: %w", err)
		}

		// 更新optCache.overlay，并避免出现重复的文件
		haveToCheckReloading = false
		for path, data := range overlay {
			if _, ok := optCache.overlay[path]; !ok {
				optCache.overlay[path] = data
				haveToCheckReloading = true
			}
		}
	}

	return
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
