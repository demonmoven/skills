package core

import (
	"go/ast"
	"go/token"
	"strings"

	"code.byted.org/analyzers/go_cleaner/pkg/core/passes"
	"code.byted.org/analyzers/go_cleaner/pkg/core/repoinfo"
	"github.com/sirupsen/logrus"
	"golang.org/x/tools/go/packages"
)

// 使用单例模式，暂时避免在程序中不断传播PassManager
var GlobalPassManager *PassManager

type PassMgrConfig struct {
	// 直接修改源文件，而不是输出到cache中
	Inplace bool

	// 忽略测试文件的编译错误
	IgnoreTestErrs bool
	// 在测试文件存在错误的情况下，最多尝试次数
	MaxNumsOfReloadingWithTestErrs int
	// 忽略文件的编译错误
	IgnoreCompileErrs bool
	Passnames         []string
}

type PassManager struct {
	ErrMgr        *CompileErrManager
	Config        *PassMgrConfig
	ChangedSyntax map[*ast.File]*token.FileSet
	EnabledPasses []passes.Pass
}

func InitializePassManager(config *PassMgrConfig) {
	if GlobalPassManager != nil {
		return
	}

	GlobalPassManager = &PassManager{
		Config:        config,
		ChangedSyntax: map[*ast.File]*token.FileSet{},
	}

	var unknowns []string
	if GlobalPassManager.EnabledPasses, unknowns = passes.FilerKnownPasses(config.Passnames); len(unknowns) > 0 {
		logrus.Infof("Unknown passes: %v\n", strings.Join(unknowns, ","))
	}
}

func (m *PassManager) RunOnFixpoint(module *repoinfo.Module) (changed bool) {
	// 理论上，我们需要多次运行pass，直到程序不再改变为止
	// 这里我们运行程序两次来近似这个过程，从而暴露更多设计中的问题
	for i := 0; i < 2; i++ {
		changed = m.Run(module) || changed
	}

	return
}

func (m *PassManager) Rewrite(module *repoinfo.Module) {
	// 我们在所有package的分析结束后真正修改硬盘上的源文件，以便在未来加入跨package的分析（例如SSA）
	for file, fset := range m.ChangedSyntax {
		if m.Config.Inplace {
			filename := fset.Position(file.Pos()).Filename
			repoinfo.WriteToFile(fset, file, filename)
		} else {
			repoinfo.WriteToStdout(fset, file)
		}
	}

	// 在所有pass结束后，我们清空ChangedSyntax，以便下一次运行时可以重新收集所有的变更
	// 如果不刷新则可能导致同一个文件存在不同的*ast.File（每次Run都会产生*ast.File的积累）
	m.ChangedSyntax = make(map[*ast.File]*token.FileSet)
}

func (m *PassManager) Run(module *repoinfo.Module) bool {
	var passnames []string
	effectives := make(map[string]bool)
	for _, pass := range m.EnabledPasses {
		passnames = append(passnames, pass.Name())
	}

	if len(passnames) > 0 {
		logrus.Infof("Running pass %s on %d packages", strings.Join(passnames, ","), len(module.Packages))
	}

	var isChanged bool
	for _, pass := range m.EnabledPasses {
		switch p := pass.(type) {
		case passes.PackagePass:
			if p.EnableOnlyOnTypeErrors() && countErrors(module.Packages) == 0 {
				// 从硬盘上再次加载文件从而保证最新的结果能够被正确“修复式”分析
				module.Verify(false)
			}

			var isChangedDuringThisPass bool
			for _, pkg := range module.Packages {
				result, err := p.Run(pkg)
				if err != nil {
					return false
				}

				if !result.IsChanged {
					continue
				}

				isChanged = true
				isChangedDuringThisPass = true
				for file := range result.ChangedSyntax {
					m.ChangedSyntax[file] = pkg.Fset
				}
			}

			if isChangedDuringThisPass {
				effectives[pass.Name()] = true
			}
		case passes.ModulePass:
			if countErrors(module.Packages) != 0 {
				// 从硬盘上再次加载文件从而保证最新的结果能够被正确“修复式”分析
				module.Verify(false)
			}

			result, err := p.Run(module)
			if err != nil {
				return false
			}

			if !result.IsChanged {
				continue
			}

			isChanged = true
			for file := range result.ChangedSyntax {
				m.ChangedSyntax[file] = module.Config.Fset
			}

			effectives[pass.Name()] = true
		}
	}

	if len(effectives) > 0 {
		passes := make([]string, 0, len(effectives))
		for pass := range effectives {
			passes = append(passes, pass)
		}

		logrus.Infof("Pass %s changed files\n", strings.Join(passes, ","))
	}

	m.Rewrite(module)
	return isChanged
}

func countErrors(pkgs []*packages.Package) (n int) {
	packages.Visit(pkgs, nil, func(pkg *packages.Package) {
		n += len(pkg.Errors)
	})
	return
}
