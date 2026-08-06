package unused

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/token"
	"path/filepath"
	"runtime"

	"code.byted.org/analyzers/go_cleaner/pkg/core"
	"code.byted.org/analyzers/go_cleaner/pkg/core/cerr"
	"code.byted.org/analyzers/go_cleaner/pkg/core/repoinfo"
	"code.byted.org/analyzers/go_cleaner/pkg/out"
	"code.byted.org/analyzers/go_cleaner/pkg/unused/internal"
	"code.byted.org/analyzers/go_cleaner/pkg/util"
	"github.com/sirupsen/logrus"
)

type Opt struct {
	*internal.Opt
	clean  bool
	output bool
}

type Output struct {
	UnusedLine int
	TotalLine  int
	FSet       *token.FileSet
	Unused     map[ast.Node]string
}

type Option func(*Opt)

// WithTestMode 设置如何处理单测的方式
// auto 自动删除无效代码的单测， 默认
// none 排除单测
// full 所有单测都保留
func WithTestMode(mode string) Option {
	return func(o *Opt) {
		o.TestMode = mode
	}
}

// WithTag 设置加载单测文件时可能需要的go build tag
func WithTag(tag []string) Option {
	return func(o *Opt) {
		o.Tag = tag
	}
}

// WithExclude 指定清理时排除的目录，支持正则
func WithExclude(path []string) Option {
	return func(o *Opt) {
		o.ExcludePath = path
	}
}

// WithOnly 指定清理目录，默认当前路径下的所有go文件
func WithOnly(path []string) Option {
	return func(o *Opt) {
		o.OnlyPath = path
	}
}

// WithForce 自动清理时直接覆盖本地未提交的修改
func WithForce() Option {
	return func(o *Opt) {
		o.Force = true
	}
}

// WithBefore 只清理month个月前的代码
func WithBefore(month int) Option {
	return func(o *Opt) {
		o.Before = month
	}
}

// WithClean 执行自动清理
func WithClean() Option {
	return func(opt *Opt) {
		opt.clean = true
	}
}

// WithOutput 输出可清理内容
func WithOutput() Option {
	return func(opt *Opt) {
		opt.output = true
	}
}

// WithSimilarConstKeep 保留相似且靠近的常量
func WithSimilarConstKeep() Option {
	return func(opt *Opt) {
		opt.SimilarConstKeep = true
	}
}

// WithAdjacentConstKeep 保留相邻的常量
func WithAdjacentConstKeep() Option {
	return func(opt *Opt) {
		opt.AdjacentConstKeep = true
	}
}

// WithSimilarConstComment 注释相似且靠近的常量
func WithSimilarConstComment() Option {
	return func(opt *Opt) {
		opt.SimilarConstComment = true
	}
}

// WithSimilarFuncKeep 保留相似且靠近的函数
func WithSimilarFuncKeep() Option {
	return func(opt *Opt) {
		opt.SimilarFuncKeep = true
	}
}

// WithDeleteUnusedGlobalAnonymousV 删除无用的全局匿名函数
// var _ I = (*X)(nil) // 检查X是否实现I接口
// 这行代码仅做编译期检查，之际运行时不会创建这个全局变量，
// 也就是说这行永远是不可达的，但是本工具不会删除这行
// 如果I或者X被删除，这行就会报错，此时会特殊处理，删除这个报错的行
func WithDeleteUnusedGlobalAnonymousV() Option {
	return func(opt *Opt) {
		opt.DeleteUnusedGlobalAnonymousV = true
	}
}

// WithUnsafeDeleteMethod 删除无用方法，暂时不支持
func WithUnsafeDeleteMethod() Option {
	return func(opt *Opt) {
		opt.UnsafeDeleteMethod = true
	}
}

// WithIgnoreTestErr 忽略有语法错误的测试文件
func WithIgnoreTestErr() Option {
	return func(opt *Opt) {
		opt.IgnoreTestErr = true
	}
}

// WithIgnoreCompileErrs 忽略编译错误
func WithIgnoreCompileErrs() Option {
	return func(opt *Opt) {
		opt.IgnoreCompileErrs = true
	}
}

// WithBuildFlags 添加编译参数
func WithBuildFlags(buildFlags []string) Option {
	return func(opt *Opt) {
		opt.BuildFlags = append(opt.BuildFlags, buildFlags...)
	}
}

// WithCalCallGraph 计算调用链
func WithCalCallGraph() Option {
	return func(opt *Opt) {
		opt.CalCallGraph = true
	}
}

// WithImportBy 声明其它仓库的依赖项
func WithImportBy(path string) Option {
	return func(opt *Opt) {
		opt.ImportBySourceFile = path
	}
}

func WithOutputChanges(disable bool) Option {
	return func(opt *Opt) {
		opt.OutputChanges = disable
	}
}

func WithEnabledPassNames(passnames []string) Option {
	return func(opt *Opt) {
		opt.EnabledPassNames = passnames
	}
}

func WithMainDir(dir string) Option {
	return func(opt *Opt) {
		opt.MainDir = dir
	}
}

func Run(option ...Option) {
	proj, werr := run(internal.TestModeAuto, false, option...)
	defer proj.ResetRenamedMains()

	// 默认使用整仓扫描
	if werr == nil {
		err := proj.VerifyAllValidModules()
		if err == nil {
			return
		}

		if !util.IsInSideWorkTree() {
			cerr.ExitWithDetail(err, cerr.ToolAnalysisPackageContainsErr)
		}
	}

	if Config.DisableFallback || (werr != nil && !util.IsInSideWorkTree()) {
		cerr.Exit(werr)
	}

	out.SetOutput("clean-strategy-fallback", "true")
	logrus.Warn("to avoid analytical failure, fall back on conservative strategies")

	// 缓解memory压力
	proj = nil
	runtime.GC()

	if err := util.ResetWorkingRepository(); err != nil {
		panic(err)
	}

	// 如果整仓失败，则避免扫描tests以及非main packages
	proj, werr = run(internal.TestModeNone, true, option...)
	if werr != nil {
		cerr.Exit(werr)
	}

	if err := proj.VerifyAllValidModules(); err != nil {
		panic(err)
	}
}

func run(testmode string, onlymain bool, option ...Option) (*repoinfo.Project, cerr.Error) {
	o := &Opt{
		Opt: &internal.Opt{
			TestMode:                     testmode,
			Tag:                          nil,
			ExcludePath:                  []string{"kitex_gen/", "model_gen/", "thrift_gen/", "mock_gen/", "wcc/"},
			OnlyPath:                     nil,
			Force:                        false,
			IgnoreMock:                   false,
			Before:                       0,
			SimilarConstKeep:             false,
			SimilarFuncKeep:              false,
			DeleteUnusedGlobalAnonymousV: true, // 全局默认开启
			IgnoreTestErr:                false,
			CalCallGraph:                 false,
			ImportBySourceFile:           "",
			OutputChanges:                false,
			BuildFlags:                   nil,
			LoadMainPkgsOnly:             onlymain,
		},
	}
	for _, v := range option {
		v(o)
	}
	cwd, err := filepath.Abs(".")
	if err != nil {
		o.OnlyPath = append(o.OnlyPath, cwd) // 强制只删除当前目录下的文件
	}
	if o.OutputChanges {
		internal.DisableLoggingAndOutputChanges()
	}
	if !o.Force {
		internal.MustHasGitAndAllCommitted(".")
	}

	// 保证覆盖用户指定的测试选项
	if testmode == internal.TestModeNone {
		o.TestMode = internal.TestModeNone
		o.UnsafeDeleteMethod = false
	}

	optJSON, err := json.MarshalIndent(o.Opt, "", "  ")
	if err != nil {
		logrus.Errorf("marshal option failed, err: %v", err)
	} else {
		logrus.Infof("cleaner options:\n%s", optJSON)
	}

	proj := repoinfo.NewDefaultProject()

	var modcfgs []*repoinfo.ModuleConfig
	for name, dir := range proj.GetModuleNames(o.MainDir) {
		modcfgs = append(modcfgs, internal.CreateModuleConfig(o.Opt, name, dir))
	}

	// 此时我们第一次加载所有module，对于多module的项目，我们跳过一些可能错误的module
	if !proj.Load(modcfgs) {
		exiterr := cerr.ToolPreloadPackageContainsErr
		if proj.IsMultiModules {
			return nil, cerr.NewErrorWithDetail(fmt.Errorf("failed to load too many modules"), exiterr)
		} else {
			return nil, cerr.NewErrorWithDetail(proj.Modules[0].LoadError, exiterr)
		}
	}

	valids := make(map[*repoinfo.Module]bool)
	for _, m := range proj.ValidMods {
		valids[m] = true
	}

	core.InitializePassManager(&core.PassMgrConfig{
		Inplace:                        true,
		IgnoreTestErrs:                 o.IgnoreTestErr,
		IgnoreCompileErrs:              o.IgnoreCompileErrs,
		MaxNumsOfReloadingWithTestErrs: internal.MaxNumsOfReloadingWithTestErrs,
		Passnames:                      o.EnabledPassNames,
	})

	// 对于原本正确的module，我们在分析过程中反复加载它们，如果有任何错误都表明分析过程中有缺陷
	// 分析需要重复进行已保证结果更加完整，为了提高速度、改成两次
	total := len(proj.Modules)
	finished := make(map[*repoinfo.Module]bool)
	for repeat := 0; repeat < 2; repeat++ {
		for i, m := range proj.Modules {
			prefix := fmt.Sprintf("[%d/%d]", i+1, total)

			if !valids[m] {
				logrus.Infof("%s skip broken module %s", prefix, m.Name)
				continue
			}

			logrus.Infof("%s running module %s %d times, directory %s", prefix, m.Name, repeat+1, m.Dir)
			if finished[m] {
				logrus.Infof("%s skip finished module %s", prefix, m.Name)
				continue
			}

			core.GlobalPassManager.RunOnFixpoint(m)
			output, err := legacyCleanWorkflow(o, m)
			if err != nil {
				return nil, err
			}

			if output == nil || output.UnusedLine == 0 {
				finished[m] = true
			}
		}

		// 覆盖本次提交
		o.Force = true
	}

	// 只load，不做后续清理和分析，为了保证编译通过.编译不通过会直接panic
	return proj, nil
}

func legacyCleanWorkflow(o *Opt, module *repoinfo.Module) (*Output, cerr.Error) {
	d := internal.NewDetector(o.Opt, module)

	if err := d.Load(); err != nil {
		return nil, err
	}

	d.CollectDef()
	d.CollectUsed()
	d.CleanPrepare()
	if o.output {
		d.Output()
		output := &Output{
			UnusedLine: d.UnusedLine,
			TotalLine:  d.TotalLine,
			FSet:       d.FSet(),
			Unused:     d.Unused,
		}

		return output, nil
	}

	if o.clean {
		cleanedLine, totalLine := d.Clean()
		output := &Output{UnusedLine: cleanedLine, TotalLine: totalLine}
		return output, nil
	}

	return nil, nil
}
