package storage

import (
	"fmt"
	"go/token"

	"code.byted.org/analyzers/go_cleaner/pkg/core"
	"code.byted.org/analyzers/go_cleaner/pkg/core/cerr"
	"code.byted.org/analyzers/go_cleaner/pkg/core/repoinfo"
	"github.com/sirupsen/logrus"
	"golang.org/x/tools/go/packages"
)

type StorageConfig struct {
	PSMs                           []string
	LoadDir                        string
	LoadTests                      bool
	BuildFlags                     []string
	IgnoreTestErr                  bool
	MaxNumsOfReloadingWithTestErrs int
	PassNames                      []string
}

func (c *StorageConfig) CreateModuleConfig(module string, directory string) *repoinfo.ModuleConfig {
	if c.LoadDir == "" {
		c.LoadDir = "."
	}

	pkgcfg := &packages.Config{
		Mode:       packages.LoadAllSyntax | packages.NeedModule,
		Dir:        c.LoadDir,
		Tests:      c.LoadTests,
		BuildFlags: c.BuildFlags,
		Fset:       token.NewFileSet(),
		Overlay:    make(map[string][]byte),
	}

	return &repoinfo.ModuleConfig{
		Name:           module,
		Dir:            directory,
		Config:         pkgcfg,
		IgnoreTestErrs: c.IgnoreTestErr,
		MaxNumRetry:    c.MaxNumsOfReloadingWithTestErrs,
	}
}

func Run(config *StorageConfig) {
	proj := repoinfo.NewDefaultProject()

	var modcfgs []*repoinfo.ModuleConfig
	for name, dir := range proj.GetModuleNames("") {
		modcfgs = append(modcfgs, config.CreateModuleConfig(name, dir))
	}

	// 此时我们第一次加载所有module，对于多module的项目，我们跳过一些可能错误的module
	if !proj.Load(modcfgs) {
		exiterr := cerr.ToolPreloadPackageContainsErr
		if proj.IsMultiModules {
			cerr.ExitWithDetail(fmt.Errorf("Failed to load too many modules"), exiterr)
		} else {
			cerr.ExitWithDetail(proj.Modules[0].LoadError, exiterr)
		}
	}

	valids := make(map[*repoinfo.Module]bool)
	for _, m := range proj.ValidMods {
		valids[m] = true
	}

	core.InitializePassManager(&core.PassMgrConfig{
		Inplace:                        true,
		IgnoreTestErrs:                 config.IgnoreTestErr,
		IgnoreCompileErrs:              false,
		MaxNumsOfReloadingWithTestErrs: config.MaxNumsOfReloadingWithTestErrs,
		Passnames:                      config.PassNames,
	})

	total := len(proj.Modules)
	finished := make(map[*repoinfo.Module]bool)

	for i, m := range proj.Modules {
		prefix := fmt.Sprintf("[%d/%d]", i+1, total)

		if !valids[m] {
			logrus.Infof("%s Skip broken module %s", prefix, m.Name)
			continue
		}

		if finished[m] {
			logrus.Infof("%s Skip broken module %s", prefix, m.Name)
			continue
		}

		core.GlobalPassManager.RunOnFixpoint(m)
		if result := runStorageCleaner(config, m); result == nil {
			finished[m] = true
		}
	}

	// 只load，不做后续清理和分析，为了保证编译通过.编译不通过会直接panic
	if err := proj.VerifyAllValidModules(); err != nil {
		panic(err)
	}
}
