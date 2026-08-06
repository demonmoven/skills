package internal

import (
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"code.byted.org/analyzers/go_cleaner/pkg/core"
	"code.byted.org/analyzers/go_cleaner/pkg/core/cerr"
	"code.byted.org/analyzers/go_cleaner/pkg/core/repoinfo"
	"github.com/masatana/go-textdistance"
	"github.com/sirupsen/logrus"
	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
)

// 测试文件之间可能会有依赖，因此排除其中一部分测试错误之后，另一部分测试
// 可能因为缺少依赖而继续报错。为了简化处理，我们在此处多次尝试加载、直到
// 所有测试不再报错为止，或者达到最大尝试次数为止
// TODO: 此处仍然需要更系统的修复
var MaxNumsOfReloadingWithTestErrs = 10

type Detector struct {
	Module *repoinfo.Module

	prog       *ssa.Program   //ssa程序
	pkgs       []*ssa.Package // 当前package下的所有包，不包括第三方包
	initialPkg []*packages.Package

	cg *callgraph.Graph

	AllObjects map[types.Object][]*ssa.Function // 这个类型的所有方法，如果有的话
	UsedObject map[types.Object]bool

	// alias, const ssa 后会消失，需要单独处理
	NamedConstObject map[types.Object]bool
	AliasTypeObject  map[types.Object]types.Object

	// ast.Sepc 用于定位源码位置，自动删除时需要
	AllObjectsToSpec map[types.Object]ast.Spec

	// ast comment记录，自动删除时需要
	CommentPos CommentPos

	// ast.Sepc 用于定位源码位置，自动删除时需要
	ObjectsRetainedByMark map[types.Object]bool

	AllFunction      map[*ssa.Function]bool
	AllInnerFunction map[*ssa.Function]bool
	UsedFunction     map[*ssa.Function]bool

	// 非测试包的函数
	NonTestFunction map[string]*ssa.Function
	// 测试函数
	TestTBFFunction map[*ssa.Function]bool
	TestMFunction   map[*ssa.Function]bool
	// 非测试包的函数对应的测试函数
	PossibleUT map[*ssa.Function][]*ssa.Function

	// 相同名字的ssa.Package, 在test模式下, 每个包会对应两个ssa.Package,它们的path是相同的
	SamePathSSAPackage map[string][]*ssa.Package

	// 维护基于推断的一组常量的关系，如iota等
	RelatedConst map[types.Object]map[types.Object]bool
	// 相似常量不删除
	ConstSimilarMgr *constSimilarMgr
	MaySimilarConst map[types.Object]bool // 保存相似度不是很高的常量，这部分后续可能注释但不删除
	// 相似函数不删除
	FuncSimilarMgr *FuncSimilarMgr

	opt *Opt

	Cleaner *Cleaner

	// 如果一个package的init函数不可达，整个package删除
	UnusedPackageDir map[string]bool

	// 输出可删除行数时需要
	Unused     map[ast.Node]string
	UnusedLine int // 排除测试文件，测试文件不会清除
	TotalLine  int // 排除测试文件

	Blame *Blame

	keptCache map[string]bool

	OtherRepos *ImportBy
}

func NewDetector(opt *Opt, m *repoinfo.Module) *Detector {
	otherRepos, err := NewImportBy(opt.ImportBySourceFile, m)
	if err != nil {
		panic(err)
	}

	logrus.Infof("Module: %s", m.Name)

	return &Detector{
		Module:                m,
		prog:                  nil,
		pkgs:                  nil,
		cg:                    nil,
		AliasTypeObject:       map[types.Object]types.Object{},
		NamedConstObject:      map[types.Object]bool{},
		AllObjects:            map[types.Object][]*ssa.Function{},
		AllObjectsToSpec:      map[types.Object]ast.Spec{},
		UsedObject:            map[types.Object]bool{},
		ObjectsRetainedByMark: map[types.Object]bool{},
		AllFunction:           map[*ssa.Function]bool{},
		AllInnerFunction:      map[*ssa.Function]bool{},
		UsedFunction:          map[*ssa.Function]bool{},
		NonTestFunction:       map[string]*ssa.Function{},
		SamePathSSAPackage:    map[string][]*ssa.Package{},
		TestMFunction:         map[*ssa.Function]bool{},
		TestTBFFunction:       map[*ssa.Function]bool{},
		PossibleUT:            map[*ssa.Function][]*ssa.Function{},
		CommentPos:            NewCommentPos(),
		Cleaner:               NewCleaner(),
		UnusedPackageDir:      map[string]bool{},
		Unused:                map[ast.Node]string{},
		Blame:                 MustNewBlame(opt.Before),
		RelatedConst:          map[types.Object]map[types.Object]bool{},
		ConstSimilarMgr:       newConstSimilarMgr(),
		MaySimilarConst:       map[types.Object]bool{},
		FuncSimilarMgr:        NewFuncSimilarMgr(),
		opt:                   initExcludeOnlyPath(opt),
		keptCache:             map[string]bool{},
		OtherRepos:            otherRepos,
	}
}

func CreateModuleConfig(o *Opt, module string, directory string) *repoinfo.ModuleConfig {
	opts := createLoadOptions(directory, o)

	var opt option
	for _, o := range opts {
		o(&opt)
	}

	if opt.loadDir == "" {
		opt.loadDir = "."
	}

	pkgcfg := &packages.Config{
		Mode:       packages.LoadAllSyntax | packages.NeedModule,
		Dir:        opt.loadDir,
		Tests:      opt.test,
		BuildFlags: opt.buildFlag,
		Fset:       opt.fset,
		Overlay:    opt.overlay,
	}

	modcfg := &repoinfo.ModuleConfig{
		Name:             module,
		Dir:              directory,
		Config:           pkgcfg,
		IgnoreTestErrs:   o.IgnoreTestErr,
		MaxNumRetry:      MaxNumsOfReloadingWithTestErrs,
		LoadMainPkgsOnly: o.LoadMainPkgsOnly,
	}

	return modcfg
}

func createLoadOptions(moddir string, o *Opt) (opts []LoadOption) {
	opts = append(opts, WithFileSet())
	opts = append(opts, WithBuildFlags(o.BuildFlags, o.Tag))

	if o.TestMode != TestModeNone {
		opts = append(opts, WithTest())
	}
	if o.IgnoreTestErr {
		opts = append(opts, WithIgnoreTestErr())
		opts = append(opts, WithCachedOverlay())
		opts = append(opts, WithMaxNumsOfReloadingWithTestErrs(MaxNumsOfReloadingWithTestErrs))
	}
	if o.CalCallGraph {
		opts = append(opts, WithCallGraph())
	}

	if o.LoadMainPkgsOnly {
		opts = append(opts, WithLoadMainPkgsOnly())
	}

	opts = append(opts, WithLoadDir(moddir))
	return
}

func (d *Detector) Load() cerr.Error {
	logrus.Info("Load start ...")
	defer func() {
		logrus.Info("Load finished ...")
		logrus.Infof("Load done! pkg (all) count:%d, initialPkg (self) count:%d", len(d.pkgs), len(d.initialPkg))
	}()

	var err error
	d.prog, d.initialPkg, d.pkgs, d.cg, err = Load(createLoadOptions(d.Module.Dir, d.opt)...)

	if err != nil {
		return cerr.NewErrorWithDetail(err, cerr.ToolAnalysisPackageContainsErr)
	}

	return nil
}

// CollectDef 收集定义的包级别类型，常量，变量
// 函数，方法在Load时通过ssa收集过了
func (d *Detector) CollectDef() {
	logrus.Info("CollectDef start...")
	defer func() {
		logrus.Info("CollectDef done!")
	}()

	for _, pkg := range core.FilterSSAPkgsNotNil(d.pkgs) { // 不包括依赖包
		pkgTyp := d.GetPackageType(pkg)

		// 保存 map[pkgPath][]*ssa.Package, test模式下，一个有单测的package会产生两个同名同path的ssa.Package，一个包含单测，一个不包含单测
		// test文件的package可以加一个_test后缀
		if pkg.Pkg != nil {
			pkgPath := strings.TrimSuffix(pkg.Pkg.Path(), "_test")
			d.SamePathSSAPackage[pkgPath] = append(d.SamePathSSAPackage[pkgPath], pkg)
			pkgPath = pkgPath + "_test"
			d.SamePathSSAPackage[pkgPath] = append(d.SamePathSSAPackage[pkgPath], pkg)
		}

		for _, member := range pkg.Members {
			switch mem := member.(type) {
			case *ssa.Type:
				if mem.Object() != nil {
					obj := mem.Object()
					typ := mem.Type()
					d.AllObjects[obj] = d.getFuncByRecv(typ) // d.AllObject不包含闭包函数
					for _, fn := range d.AllObjects[obj] {
						d.AllFunction[fn] = true
						for _, f := range getInnerFunction(fn) {
							d.RecursiveAddInnerFunc(f)
						}
						if pkgTyp == NormalPackage {
							d.NonTestFunction[fn.String()] = fn
						}
						if pkgTyp == TestPackage {
							d.CollectUT(fn)
						}
					}
				}
			case *ssa.Global:
				if mem.Object() != nil {
					d.AllObjects[mem.Object()] = []*ssa.Function{}
				}
			case *ssa.NamedConst:
				if mem.Object() != nil {
					d.AllObjects[mem.Object()] = []*ssa.Function{}
					d.NamedConstObject[mem.Object()] = true
					if d.opt.SimilarConstKeep {
						d.ConstSimilarMgr.Register(mem.Object(), d.Position(mem.Object().Pos()))
					}
				}
			case *ssa.Function:
				d.AllFunction[mem] = true
				for _, f := range getInnerFunction(mem) {
					d.RecursiveAddInnerFunc(f)
				}
				if pkgTyp == NormalPackage {
					d.NonTestFunction[mem.String()] = mem
					if d.opt.SimilarFuncKeep {
						d.FuncSimilarMgr.Register(mem) // 只注册普通函数
					}
				}
				if pkgTyp == TestPackage {
					d.CollectUT(mem)
				}
			}
		}
	}

	// 收集相互关联的一组常量
	d.CollectRelatedConsts()

	// alias 在 ssa中没有，单独处理
	for _, pkg := range d.initialPkg {
		for _, v := range pkg.TypesInfo.Defs {
			switch vv := v.(type) {
			case *types.TypeName:
				if vv.IsAlias() {
					vv.IsAlias()
					d.AllObjects[v] = []*ssa.Function{}
					d.AliasTypeObject[v] = getTypeObject(vv.Type().Underlying())
				}
			}
		}
	}

	// object to spec, 定位到源码，用于删除
	d.collectObjectToSpec()

	// 收集标注保留的object
	d.collectMarkObject()

	// collect comment, 用于删除
	d.collectDeclareComment()

	// 相似函数不删除
	if d.opt.SimilarFuncKeep {
		d.FuncSimilarMgr.Build()
	}

	// 把d.CollectUT收集单测挂到它想测试的函数上
	d.MountUT()
}

func (d *Detector) RecursiveAddInnerFunc(innerF *ssa.Function) {
	if _, ok := d.AllInnerFunction[innerF]; ok {
		return
	}
	d.AllInnerFunction[innerF] = true
	for _, f := range getInnerFunction(innerF) {
		d.RecursiveAddInnerFunc(f)
	}
}

func (d *Detector) keepInterfaceChecks() {
	logrus.Info("keptAsUsed-InterfaceChecks start")
	defer logrus.Info("keptAsUsed-InterfaceChecks end")
	isInterfaceCheck := func(spec ast.Spec) bool {
		valspec, ok := spec.(*ast.ValueSpec)
		if !ok {
			return false
		}

		if len(valspec.Names) != 1 || len(valspec.Values) != 1 {
			return false
		}

		if valspec.Names[0].Name != "_" {
			return false
		}

		switch valspec.Type.(type) {
		case *ast.Ident, *ast.SelectorExpr:
			if unary, ok := valspec.Values[0].(*ast.UnaryExpr); ok {
				return unary.Op == token.AND
			}

			if call, ok := valspec.Values[0].(*ast.CallExpr); ok {
				if paren, ok := call.Fun.(*ast.ParenExpr); ok {
					_, ok = paren.X.(*ast.StarExpr)
					return ok
				}

				if ident, ok := call.Fun.(*ast.Ident); ok && ident.Name == "new" {
					return true
				}
			}
		default:
		}

		return false
	}

	candidates := make(map[*ast.ValueSpec]bool)
	for _, pkg := range d.initialPkg {
		for _, file := range pkg.Syntax {
			for _, decl := range file.Decls {
				gendecl, ok := decl.(*ast.GenDecl)
				if !ok {
					continue
				}

				for _, spec := range gendecl.Specs {
					if isInterfaceCheck(spec) {
						candidates[spec.(*ast.ValueSpec)] = true
					}
				}
			}
		}
	}

	idents := make(map[*ast.Ident]struct{})
	for c := range candidates {
		ast.Inspect(c, func(n ast.Node) bool {
			if ident, ok := n.(*ast.Ident); ok {
				idents[ident] = struct{}{}
			}
			return true
		})
	}

	for ident := range idents {
		for _, pkg := range d.initialPkg {
			if obj := pkg.TypesInfo.ObjectOf(ident); obj != nil {
				d.addUsedObject(obj)
			}
		}
	}
}

func (d *Detector) keepSyntaxReferences() {
	logrus.Info("keptAsUsed-SyntaxReferences start")
	defer logrus.Info("keptAsUsed-SyntaxReferences end")
	for name := range d.Module.ReferedIdents {
		for _, pkg := range d.prog.AllPackages() {
			member, ok := pkg.Members[name]
			if !ok {
				continue
			}

			switch m := member.(type) {
			case *ssa.Type, *ssa.NamedConst, *ssa.Global:
				d.addUsedObject(m.Object())
			case *ssa.Function:
				d.addUsedFunc(m)
			default:
			}
		}
	}
}

// CollectUsed 收集可达的包级别类型，常量，变量，函数，方法
// 对于类型，变量，函数，方法，以main.main, main.init为入口递归搜索
// 对于常量，alias，没法通过ssa分析得到，这里做简化处理，全局范围被使用了的const, alias(且底层类型被使用)被保留
func (d *Detector) CollectUsed() {
	logrus.Info("CollectUsed start...")
	defer func() {
		logrus.Info("CollectUsed done!")
	}()

	isEntry := func(f *ssa.Function) bool {
		if f.Pkg == nil {
			return false
		}

		proj := d.Module.Project

		// 在monorepo中可能存在多个main函数，因此这里保守地估计所有main函数都被访问到了
		if proj.IsMonoRepo && proj.IsMain(f.Name()) {
			return true
		}

		if !d.IsProjectPkg(f.Pkg) {
			return false
		}
		if f.Signature != nil && f.Signature.Recv() != nil {
			return false
		}
		pkgTyp := d.GetPackageType(f.Pkg)
		pkgName := f.Pkg.Pkg.Name()
		fnName := f.Name()

		// 1. 在多module的monorepo中可能不存在main函数，因此这里保守地认为所有init都被访问到了
		// 2. 隐式依赖场景：https://bytedance.larkoffice.com/wiki/X55jwaVjaiAAPzkSUfrcM6T1nze
		if d.Module.Project.IsMonoRepo && pkgTyp == NormalPackage && fnName == "init" {
			return true
		}

		if pkgTyp == NormalPackage && pkgName == "main" && (proj.IsMain(fnName) || fnName == "init") {
			return true
		}

		// core.Logger.At("LegacyDetector").Infof("[Detector] Not entrypoint %s pkg (type %d) %s mod %s", fnName, pkgTyp,f.Pkg.Pkg.Path(), d.Module.Name)
		return false
	}

	logrus.Info("CollectUsed-addUsedFunc start...")
	for f := range d.AllFunction {
		// main.main, main.init (main.init 是自动生成的函数，会调用其它包的init,再调用本包手写的init函数) 为入口
		if isEntry(f) {
			d.addUsedFunc(f)
		}
	}
	logrus.Info("CollectUsed-addUsedFunc done!")

	if d.opt.UnsafeDeleteMethod {
		// 因为一个结构体可能会实现 some interface，如果未经检查就删除了方法，在该接口做断言检查的
		// 时候可能会报告一个impossible implementation的错误； strict来说，我们应当搜索整个程序找出
		// 那些可能做断言检查的地方，这里为了实现简单做更保守的处理
		logrus.Info("CollectReferedFunc start")
		for fn := range repoinfo.AllReferedFunctions(d.prog) {
			d.addUsedFunc(fn)
		}
		logrus.Info("CollectReferedFunc done")
	}

	// 如果指定只删除特定目录下的文件，其它目录下的都认为是可到达的
	d.keptAsUsed()

	logrus.Info("CollectUsed-aliasUnused start")
	// 对于 alias, ssa 分析时会自动转化为底层类型
	// 如果底层类型没被使用，alias一定没有用
	aliasUnused := map[types.Object]bool{}
	for a, o := range d.AliasTypeObject {
		if o == nil {
			continue
		}
		if _, ok := d.AllObjects[o]; ok {
			if _, ok := d.UsedObject[o]; !ok {
				aliasUnused[a] = true
			}
		}
	}
	// const, alias 全局范围被使用了的const, alias(且底层类型被使用)被保留
	for _, pkg := range d.initialPkg {
		for _, v := range pkg.TypesInfo.Uses { //这里可以做优化。pkg.TypesInfo.Uses是整个包依赖的，删除无用方法后，这个依赖的类型不会被删除。导致部分代码没有被删除
			if _, ok := d.NamedConstObject[v]; ok {
				d.addUsedObject(v)
			}
			if _, ok := d.AliasTypeObject[v]; ok && !aliasUnused[v] {
				d.addUsedObject(v)
			}
		}
	}

	logrus.Info("CollectUsed-aliasUnused done")

	//标注保留的代码
	logrus.Info("CollectUsed-ObjectsRetainedByMark start")
	for o := range d.ObjectsRetainedByMark {
		logrus.Infof("IsRetainedByMark, object:%s", o.String())
		d.addUsedObject(o)
	}
	logrus.Info("CollectUsed-ObjectsRetainedByMark end")

	logrus.Info("CollectUsed-FunctionRetainedByMark start")
	for f := range d.AllFunction {
		if !d.IsProjectPkg(f.Pkg) {
			continue // 项目外忽略
		}
		// 被标注保留的代码
		if funcDecl, ok := f.Syntax().(*ast.FuncDecl); ok {
			if IsRetainedByMark(funcDecl.Doc) {
				logrus.Infof("IsRetainedByMark, func:%s", funcDecl.Name.String())
				d.addUsedFunc(f)
			}
		}
	}
	logrus.Info("CollectUsed-FunctionRetainedByMark end")

}

// keptAsUsed 将参数指定的不删除类型，方法认为是可达的
// exclude 匹配的目录是不删除的，匹配认为可达
// only 匹配的目录是需要删除的，不匹配认为可达
func (d *Detector) keptAsUsed() {
	logrus.Info("keptAsUsed start...AllFunction count:%d,AllObjects count:%d", len(d.AllFunction), len(d.AllObjects))
	defer func() {
		logrus.Info("keptAsUsed done!")
	}()

	logrus.Info("keptAsUsed-AllFunction start")
	skiped := make(map[*types.Package]bool)
	for f := range d.AllFunction {
		if !d.IsProjectPkg(f.Pkg) {
			if f.Pkg != nil && f.Pkg.Pkg != nil && !skiped[f.Pkg.Pkg] {
				skiped[f.Pkg.Pkg] = true
			}

			continue // 项目外的可以忽略，因为删除时也不会包含项目外的
		}
		// 按路径保留
		if f.Pkg != nil && f.Pkg.Pkg != nil && f.Pos() != token.NoPos && d.kept(d.Position(f.Pos()).Filename) {
			if _, ok := d.UsedFunction[f]; !ok {
				d.addUsedFunc(f)
			}
		}
		// 按时间保留
		if f.Syntax() != nil && d.Blame.RecentlyModifiedWithin(d.Position(f.Syntax().Pos()), d.Position(f.Syntax().End())) {
			d.addUsedFunc(f)
		}

		// 其它仓库引用则保留
		if others := d.OtherRepos; others != nil && others.ImportF(f) {
			d.addUsedFunc(f)
		}
	}
	logrus.Info("keptAsUsed-AllFunction end")

	logrus.Info("keptAsUsed-AllObjects start")
	for o := range d.AllObjects {
		if !d.IsProjectPkg3(o.Pkg()) {
			continue
		}
		// 按路径保留
		if o.Pkg() != nil && o.Pos() != token.NoPos && (d.kept(d.Position(o.Pos()).Filename)) {
			d.addUsedObject(o)
		}

		// 按时间保留
		if spec, ok := d.AllObjectsToSpec[o]; ok && spec != nil && d.Blame.RecentlyModifiedWithin(d.Position(spec.Pos()), d.Position(spec.End())) {
			d.addUsedObject(o)
		}

		// 其它仓库引用则保留
		if others := d.OtherRepos; others != nil && others.Import(o) {
			d.addUsedObject(o)
		}
	}

	d.keepInterfaceChecks()
	d.keepSyntaxReferences()

	logrus.Info("keptAsUsed-AllObjects end")
}

const MarkRetainCommentText = "// Marker Retained to Prevent Clearing"

func IsRetainedByMark(doc *ast.CommentGroup) bool {
	if doc == nil {
		return false
	}
	for _, c := range doc.List {
		if c.Text == MarkRetainCommentText {
			return true
		}
	}
	return false
}

// CollectRelatedConsts 收集基于推断的一组常量，并维护其关系, 如以下场景
/**
const (
	A = 1
	B
)

const (
	A1 = iota
	B1
)
*/
func (d *Detector) CollectRelatedConsts() {
	extractFuncs := []func(pkg *packages.Package, constDecl *ast.GenDecl) map[types.Object]map[types.Object]bool{
		ExtractConstsRelationIota,
		ExtractConstRelationInheritLast,
	}
	if d.opt.AdjacentConstKeep {
		extractFuncs = append(extractFuncs, ExtractConstsRelationAdjacent)
	}
	for _, fn := range extractFuncs {
		for _, pkg := range d.initialPkg {
			for _, file := range pkg.Syntax {
				for _, decl := range file.Decls {
					if constDecl, ok := decl.(*ast.GenDecl); ok && constDecl.Tok == token.CONST {
						for v, os := range fn(pkg, constDecl) {
							if _, ok := d.RelatedConst[v]; !ok {
								d.RelatedConst[v] = map[types.Object]bool{}
							}
							for o := range os {
								d.RelatedConst[v][o] = true
							}
						}
					}
				}
			}
		}
	}
}

// kept determines based on file full path.
func (d *Detector) kept(path string) bool {
	if f, ok := d.keptCache[path]; ok {
		return f
	}
	var f bool
	if d.opt.TestMode == TestModeFull && strings.HasSuffix(path, "_test.go") {
		f = true
	} else {
		f = (len(d.opt.OnlyPath) > 0 && !d.opt.onlyPathRe.MatchString(path)) || (len(d.opt.ExcludePath) > 0 && d.opt.excludePathRe.MatchString(path))
	}

	if KeepAutoGen(path) {
		f = true
	}

	d.keptCache[path] = f
	return f 
}

func (d *Detector) CleanPrepare() {
	logrus.Info("CleanPrepare start...")
	defer func() {
		logrus.Info("CleanPrepare done!")
	}()
	fset := d.prog.Fset
	for o := range d.AllObjects {
		if !d.IsProjectPkg3(o.Pkg()) {
			continue
		}
		if d.AllObjectsToSpec[o] == nil {
			continue
		}
		d.Cleaner.AddObject(d.AllObjectsToSpec[o], fset)
		if _, ok := d.UsedObject[o]; !ok {
			//fmt.Println(d.prog.Fset.Position(o.Pos()), o.Name(), "unused.")
			if d.MaySimilarConst[o] {
				// 相似度不高的常量暂时注释而不删除
				d.Cleaner.AddCommentObject(d.AllObjectsToSpec[o], fset)
			} else if !core.MaybeSpecReferenced(d.AllObjectsToSpec[o]) {
				d.Cleaner.AddUnusedObject(d.AllObjectsToSpec[o], fset)
			}
		} else {
			d.Cleaner.AddUsedObject(d.AllObjectsToSpec[o], fset)
		}
	}
	for f := range d.AllFunction {
		if !d.IsProjectPkg(f.Pkg) {
			continue
		}
		d.Cleaner.AddFunc(f)
		if _, ok := d.UsedFunction[f]; !ok && !core.MaybeNameReferenced(f.Name()) {
			d.Cleaner.AddUnusedFunc(f)
			if f.Name() == "init" && f.Signature != nil && f.Signature.Recv() == nil {
				if dir := d.PackageDir(f.Pkg); dir != "" && !d.kept(dir) && !containsSubdirectory(dir) {
					d.UnusedPackageDir[dir] = true
				}
			}
		} else {
			d.Cleaner.AddUsedFunc(f)
		}
	}
}

func (d *Detector) Output() {
	logrus.Info("Output start...")
	defer func() {
		logrus.Info("Output done!")
	}()
	cwd, _ := filepath.Abs(".")
	used := d.Cleaner.used
	clean := d.Cleaner.unused.Sub(d.Cleaner.used)
	d.UnusedLine = clean.notTestSum()
	d.TotalLine = d.Cleaner.all.notTestSum()
	for o := range d.AllObjects {
		if !d.IsProjectPkg3(o.Pkg()) {
			continue
		}
		spec := d.AllObjectsToSpec[o]
		pos := d.Position(spec.Pos())
		if !strings.HasPrefix(pos.Filename, cwd) {
			continue
		}
		if _, ok := d.UsedObject[o]; !ok && !used.Has(pos) {
			d.Unused[d.AllObjectsToSpec[o]] = o.Name()
		}
	}
	for f := range d.AllFunction {
		if !d.IsProjectPkg(f.Pkg) || f.Syntax() == nil {
			continue
		}
		syn := f.Syntax()
		pos := d.Position(syn.Pos())
		if !strings.HasPrefix(pos.Filename, cwd) {
			continue
		}
		if _, ok := d.UsedFunction[f]; !ok && !used.Has(pos) {
			d.Unused[syn] = f.Name()
		}
	}
}

func (d *Detector) Clean() (int, int) {
	logrus.Info("Clean start...")
	defer func() {
		logrus.Info("Clean done!")
	}()
	test := true
	if d.opt.TestMode == TestModeNone {
		test = false
	}
	return d.Cleaner.Clean(d.prog.Fset, d.CommentPos, test, d.opt, d.UnusedPackageDir)
}

func (d *Detector) addUsedObjOrFunc(o types.Object) {
	if o == nil {
		return
	}

	if fn, ok := o.(*types.Func); ok {
		if ssafn := d.prog.FuncValue(fn); ssafn != nil {
			d.addUsedFunc(ssafn)
			return
		}
	}

	d.addUsedObject(o)
}

// addUsedObject  DFS递归添加可达的类型，包括泛形类型的原类型和方法
func (d *Detector) addUsedObject(o types.Object) {
	if o == nil {
		return
	}
	if _, ok := d.AllObjects[o]; !ok {
		return
	}
	if _, ok := d.UsedObject[o]; ok {
		return
	}
	d.UsedObject[o] = true

	if !d.opt.UnsafeDeleteMethod {
		// 如果类型保留，它所有的方法保留
		for _, f := range d.AllObjects[o] {
			d.addUsedFunc(f)
		}
	} else {
		// 删除方法时，保留所有导出方法
		for _, f := range d.AllObjects[o] {
			syntax := f.Syntax()
			if syntax == nil {
				continue
			}

			if fn, ok := syntax.(*ast.FuncDecl); ok && fn.Name.IsExported() {
				d.addUsedFunc(f)
			}
		}
	}

	// 如果类型保留，它关联的其它类型保留
	for i := range getObjectUsedObject(o) {
		d.addUsedObject(i)
	}

	// 相互关联的常量保留
	for v := range d.RelatedConst[o] {
		d.addUsedObject(v)
	}

	// 相似常量保留
	if d.opt.SimilarConstKeep && d.NamedConstObject[o] {
		for _, similarO := range d.ConstSimilarMgr.Similar(o, d.Position(o.Pos()), 0.8) {
			d.addUsedObject(similarO)
		}

	}
	// 相似常量注释
	if d.opt.SimilarConstComment && d.NamedConstObject[o] {
		for _, similarO := range d.ConstSimilarMgr.Similar(o, d.Position(o.Pos()), 0.6) {
			d.MaySimilarConst[similarO] = true
		}
	}
}

// addUsedFunc DFS递归添加可达的函数，包括泛形函数的原函数，以及它所在包的init方法。
func (d *Detector) addUsedFunc(f *ssa.Function) {
	if f == nil {
		return
	}
	if _, ok := d.UsedFunction[f]; ok {
		return
	}
	d.UsedFunction[f] = true

	// TODO: 待确认
	// AllFunction， AllInnerFunction里不存在的函数，可以不递归处理，以加快速度
	if !d.AllFunction[f] && !d.AllInnerFunction[f] {
		// 但是，泛形函数的原函数如果在项目内的话需要保留
		orig := reflect.ValueOf(f).Elem().FieldByName("topLevelOrigin").UnsafePointer()
		if orig == nil {
			return
		}
		o := (*ssa.Function)(orig)
		if !d.AllFunction[o] && !d.AllInnerFunction[o] {
			return
		}
	}

	// TODO: 这里应该可以删掉
	// 它所在包的init方法都保留，包括test模式下生成package对应的init
	// 主要目的是如果包init可达，其在test模式下对应的包的init也可达
	if f.Name() == "init" && f.Pkg != nil && f.Pkg.Pkg != nil {
		for _, pkg := range d.SamePathSSAPackage[f.Pkg.Pkg.Path()] {
			if initFn, ok := pkg.Members["init"]; ok {
				if v, ok := initFn.(*ssa.Function); ok {
					d.addUsedFunc(v)
				}
			}
		}
	}

	// 如果某个单测函数需要保留，其所在包的init函数也需要保留
	if (d.TestTBFFunction[f] || d.TestMFunction[f]) && f.Pkg != nil && f.Pkg.Pkg != nil {
		if initFn, ok := f.Pkg.Members["init"]; ok {
			if v, ok := initFn.(*ssa.Function); ok {
				d.addUsedFunc(v)
			}
		}
	}

	// 它调用的方法都保留，包括动态调用
	if d.opt.CalCallGraph && d.cg != nil {
		if node, ok := d.cg.Nodes[f]; ok {
			for _, edge := range node.Out {
				fn := edge.Callee.Func
				// testMode=auto模式下，避免所有的单测都可达
				// TODO: 这里应该可以去掉，原意是 f -> fn, f 是外部函数， fn 是测试函数 ( 比如m.Run()最终会调用所有的单测函数 )
				if d.opt.TestMode == TestModeAuto && d.TestTBFFunction[fn] && (!d.IsProjectPkg(f.Pkg)) {
					continue
				}
				d.addUsedFunc(fn)
			}
		}

		// Exported函数均保留
		if f.Syntax() != nil {
			if decl, ok := f.Syntax().(*ast.FuncDecl); ok && decl.Name.IsExported() {
				d.addUsedFunc(f)
			}
		}
	}

	// 它用到的函数都保留，包括闭包，泛形函数
	for fn := range getUsedFunc(f) {
		// testMode=auto模式下，避免所有的单测都可达
		// TODO: 这里应该可以去掉，原意是 f -> fn, f 是外部函数， fn 是测试函数 ( 比如m.Run()最终会调用所有的单测函数 )
		if d.opt.TestMode == TestModeAuto && d.TestTBFFunction[fn] && !d.IsProjectPkg(f.Pkg) {
			continue
		}
		d.addUsedFunc(fn)
	}

	// 开启方法清理
	if d.opt.UnsafeDeleteMethod {
		// 1. 结构体转接口，相关方法无论是否被调用都需要保留
		for fn := range d.keepImplements(f) {
			d.addUsedFunc(fn)
		}

		// 2. 接口断言
		// TODO

		// 3. 反射调用
		// TODO
	}

	// 相似函数保留
	if d.opt.SimilarFuncKeep {
		for _, fn := range d.FuncSimilarMgr.Similar(f) {
			d.addUsedFunc(fn)
		}
	}

	// 它的UT都保留
	for _, ut := range d.PossibleUT[f] {
		d.addUsedFunc(ut)
	}

	// 它用到的类型都保留
	for o := range getFuncUsedObject(f) {
		if _, ok := d.AllObjects[o]; ok {
			d.addUsedObject(o)
		}
	}
}

func (d *Detector) keepImplements(fn *ssa.Function) map[*ssa.Function]bool {
	m := map[*ssa.Function]bool{}

	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if makeInterface, ok := instr.(*ssa.MakeInterface); ok {
				interfaceT := interfaceType(makeInterface.Type())
				instanceT := namedType(makeInterface.X.Type())
				if interfaceT == nil || instanceT == nil {
					continue
				}
				for _, v := range d.AllObjects[instanceT.Obj()] {
					for i := 0; i < interfaceT.NumMethods(); i++ {
						// Go 只需要比较方法名就行了
						if v.Name() == interfaceT.Method(i).Name() {
							m[v] = true
						}
					}
				}
			}
		}
	}

	return m
}

func (d *Detector) IsProjectPkg(pkg *ssa.Package) bool {
	if pkg == nil || pkg.Pkg == nil {
		return false
	}
	return d.IsProjectPkg3(pkg.Pkg)
}

func (d *Detector) IsProjectPkg2(pkg *packages.Package) bool {
	if pkg == nil {
		return false
	}
	return d.IsProjectPkg0(pkg.PkgPath)
}

func (d *Detector) IsProjectPkg3(pkg *types.Package) bool {
	if pkg == nil {
		return false
	}
	return d.IsProjectPkg0(pkg.Path())
}

func (d *Detector) IsProjectPkg0(pkgPath string) bool {
	return pkgPath == d.Module.Name || strings.HasPrefix(pkgPath, d.Module.Name+"/") || strings.HasPrefix(pkgPath, d.Module.Name+".")
}

func (d *Detector) collectObjectToSpec() {
	for _, pkg := range d.initialPkg {
		for _, file := range pkg.Syntax {
			for _, decl := range file.Decls {
				if _, ok := decl.(*ast.GenDecl); !ok {
					continue
				}
				genDecl := decl.(*ast.GenDecl)
				for _, spec := range genDecl.Specs {
					switch x := spec.(type) {
					case *ast.TypeSpec:
						if o := pkg.TypesInfo.ObjectOf(x.Name); o != nil {
							d.AllObjectsToSpec[o] = x
						}
					case *ast.ValueSpec:
						for _, name := range x.Names {
							if o := pkg.TypesInfo.ObjectOf(name); o != nil {
								d.AllObjectsToSpec[o] = x
							}
						}

					}
				}
			}
		}
	}
}

func (d *Detector) collectMarkObject() {
	for _, pkg := range d.initialPkg {
		for _, file := range pkg.Syntax {
			for _, decl := range file.Decls {
				if _, ok := decl.(*ast.GenDecl); !ok {
					continue
				}
				genDecl := decl.(*ast.GenDecl)
				declRetained := IsRetainedByMark(genDecl.Doc)
				for _, spec := range genDecl.Specs {
					switch x := spec.(type) {
					case *ast.TypeSpec:
						if declRetained || IsRetainedByMark(x.Doc) {
							if o := pkg.TypesInfo.ObjectOf(x.Name); o != nil {
								d.ObjectsRetainedByMark[o] = true
							}
						}
					case *ast.ValueSpec:
						if declRetained || IsRetainedByMark(x.Doc) {
							for _, name := range x.Names {
								if o := pkg.TypesInfo.ObjectOf(name); o != nil {
									d.ObjectsRetainedByMark[o] = true
								}
							}
						}
					}
				}
			}
		}
	}
}

// getFuncByRecv0 返回 typ 关联的所有 ssa.Function
func (d *Detector) getFuncByRecv0(typ types.Type) []*ssa.Function {
	var fns []*ssa.Function

	methods := d.prog.MethodSets.MethodSet(typ)
	for i := 0; i < methods.Len(); i++ {
		sel := methods.At(i)
		fn := d.prog.MethodValue(sel)
		if fn != nil {
			fns = append(fns, fn)
		}
		// 泛形结构体的方法上面那种方式不能正确获取，这种方式可以
		fn = d.prog.FuncValue(sel.Obj().(*types.Func))
		if fn != nil {
			fns = append(fns, fn)
		}
	}

	return fns
}

func (d *Detector) getFuncByRecv(typ types.Type) []*ssa.Function {
	var fns []*ssa.Function

	fns = append(fns, d.getFuncByRecv0(typ)...)

	if t, ok := typ.(*types.Pointer); ok {
		fns = append(fns, d.getFuncByRecv0(t.Elem())...)
	} else {
		fns = append(fns, d.getFuncByRecv0(types.NewPointer(typ))...)
	}

	return fns
}

func (d *Detector) collectDeclareComment() {
	for _, pkg := range d.initialPkg {
		if !d.IsProjectPkg2(pkg) {
			continue
		}
		for _, f := range pkg.Syntax {
			for _, decl := range f.Decls {
				switch v := decl.(type) {
				case *ast.GenDecl:
					if v.Doc != nil {
						d.CommentPos.Add(d.Position(decl.Pos()), d.Position(decl.End()), d.Position(v.Doc.Pos()), d.Position(v.Doc.End()))
					}
					for _, spec := range v.Specs {
						switch s := spec.(type) {
						case *ast.ValueSpec:
							if s.Doc != nil {
								d.CommentPos.Add(d.Position(s.Pos()), d.Position(s.End()), d.Position(s.Doc.Pos()), d.Position(s.Doc.End()))
							}
						case *ast.TypeSpec:
							if s.Doc != nil {
								d.CommentPos.Add(d.Position(s.Pos()), d.Position(s.End()), d.Position(s.Doc.Pos()), d.Position(s.Doc.End()))
							}
						}
					}
				case *ast.FuncDecl:
					if v.Doc != nil {
						d.CommentPos.Add(d.Position(decl.Pos()), d.Position(decl.End()), d.Position(v.Doc.Pos()), d.Position(v.Doc.End()))
					}
				}
			}
		}
	}
}

func (d *Detector) Position(pos token.Pos) token.Position {
	return d.prog.Fset.Position(pos)
}

func (d *Detector) PackageDir(p *ssa.Package) string {
	if p.Pkg == nil {
		return ""
	}

	rel, err := filepath.Rel(d.Module.Name, p.Pkg.Path())
	if err != nil {
		return ""
	}
	wd, err := filepath.Abs(".")
	if err != nil {
		return ""
	}

	return filepath.Join(wd, rel)
}

func (d *Detector) FSet() *token.FileSet {
	return d.prog.Fset
}

func initExcludeOnlyPath(opt *Opt) *Opt {
	collectGeneratePath(opt)
	opt.excludePathRe = genRe(opt.ExcludePath)
	opt.onlyPathRe = genRe(opt.OnlyPath)
	return opt
}

type PackageType int

const (
	NormalPackage      PackageType = 1 // 不包含.test.go
	TestPackage                    = 2 // 包含.go, .test.go
	GenTestMainPackage             = 3 // 自动生成main包，用于执行test case
)

func (d *Detector) GetPackageType(p *ssa.Package) PackageType {
	if p.Pkg == nil {
		return NormalPackage
	}

	if strings.HasSuffix(p.Pkg.Path(), ".test") {
		return GenTestMainPackage
	}

	var testGoScope func(scope *types.Scope) bool

	testGoScope = func(scope *types.Scope) bool {
		for i := 0; i < scope.NumChildren(); i++ {
			if testGoScope(scope.Child(i)) {
				return true
			}
		}
		if scope == nil || !scope.Pos().IsValid() {
			return false
		}
		if strings.HasSuffix(d.Position(scope.Pos()).Filename, "_test.go") {
			return true
		}
		return false
	}

	if testGoScope(p.Pkg.Scope()) {
		return TestPackage
	}

	return NormalPackage
}

var utNamePattern = regexp.MustCompile("^(Test|Benchmark|Fuzz)")

func (d *Detector) CollectUT(f *ssa.Function) {
	if f == nil {
		return
	}
	if !utNamePattern.MatchString(f.Name()) {
		return
	}
	if !strings.HasSuffix(d.Position(f.Pos()).Filename, "_test.go") {
		return
	}
	if f.Signature == nil {
		return
	}
	if f.Signature.Params().Len() != 1 || f.Signature.Recv() != nil || f.Signature.Results().Len() != 0 {
		return
	}
	switch f.Signature.Params().At(0).Type().String() {
	case "*testing.T", "*testing.B", "*testing.F":
		d.TestTBFFunction[f] = true
	case "*testing.M":
		d.TestMFunction[f] = true
	default:
	}
}

func (d *Detector) MountUT() {
	for f := range d.TestTBFFunction {
		v := d.determineUT(f)

		// core.Logger.At("LegacyDetector").Infof("UT Relation, TestFunc: %v, BizFunc: %v", f, v)

		// 没有找到该单测想测的函数，该单测不会挂在任何函数上，之后可能被删除，这样可能会导致问题，比如
		// test函数调用其它一个测试函数，通过传入不同的参数测试不同的场景
		// 没有找到该单测想测的函数，该单测挂到这个pkg（包含test文件的pkg）的init函数上
		// TODO: 是否有更好的办法
		if len(v) == 0 {
			if initFn, ok := f.Pkg.Members["init"]; ok {
				if v, ok := initFn.(*ssa.Function); ok {
					d.PossibleUT[v] = append(d.PossibleUT[v], f)
				}
			}
		}

		for vv := range v {
			d.PossibleUT[vv] = append(d.PossibleUT[vv], f)
		}
	}

	// testMain挂到这个pkg（包含test文件的pkg）的init函数上
	for f := range d.TestMFunction {
		if initFn, ok := f.Pkg.Members["init"]; ok {
			if v, ok := initFn.(*ssa.Function); ok {
				d.PossibleUT[v] = append(d.PossibleUT[v], f)
			}
		}
	}

}

// determineUT determines the function that the passed f wants to test.
func (d *Detector) determineUT(f *ssa.Function) map[*ssa.Function]bool {
	fs := map[*ssa.Function]bool{}

	getUsedFuncRecursive(f, fs)

	if d.opt.CalCallGraph && d.cg != nil {
		if node, ok := d.cg.Nodes[f]; ok {
			for _, edge := range node.Out {
				fs[edge.Callee.Func] = true
			}
		}
	}

	noTest := func(o []*ssa.Function) (v []*ssa.Function) {
		for _, i := range o {
			name := i.String()
			if vv, ok := d.NonTestFunction[name]; ok {
				v = append(v, vv)
			} else if vv, ok := d.NonTestFunction[strings.Split(name, "$")[0]]; ok {
				v = append(v, vv)
			}

		}
		return
	}

	targetFuncName := ""
	testFuncName := f.Name()
	if strings.HasPrefix(testFuncName, "Test") {
		targetFuncName = testFuncName[4:]
	}
	if strings.HasPrefix(testFuncName, "Test_") {
		targetFuncName = testFuncName[5:]
	}
	if strings.HasPrefix(testFuncName, "Benchmark") {
		targetFuncName = testFuncName[9:]
	}
	if strings.HasPrefix(testFuncName, "Benchmark_") {
		targetFuncName = testFuncName[10:]
	}
	if strings.HasPrefix(testFuncName, "Fuzz") {
		targetFuncName = testFuncName[4:]
	}
	if strings.HasPrefix(testFuncName, "Fuzz_") {
		targetFuncName = testFuncName[5:]
	}
	targetFuncName = strings.ToLower(strings.ReplaceAll(targetFuncName, "_", ""))

	sameName := func(o []*ssa.Function) (v []*ssa.Function) {
		similar := 0.8
		var matchFunc *ssa.Function
		for _, vv := range o {
			vName := strings.ToLower(strings.ReplaceAll(vv.Name(), "_", ""))
			if vName == targetFuncName {
				v = append(v, vv)
			}
			if len(v) == 0 {
				if score := textdistance.JaroWinklerDistance(vName, targetFuncName); score > similar {
					similar = score
					matchFunc = vv
				}
			}
		}
		if len(v) == 0 && matchFunc != nil {
			v = append(v, matchFunc)
		}
		return v
	}

	allowPkg := d.SamePathSSAPackage[f.Pkg.Pkg.Path()]

	samePkg := func(o []*ssa.Function) (v []*ssa.Function) {
		for _, vv := range o {
			for _, p := range allowPkg {
				if vv.Pkg == p {
					v = append(v, vv)
				}
			}
		}
		return v
	}

	sameFile := func(o []*ssa.Function) (v []*ssa.Function) {
		target := strings.ReplaceAll(d.Position(f.Pos()).Filename, "_test.go", ".go")
		for _, vv := range o {
			if d.Position(vv.Pos()).Filename == target {
				v = append(v, vv)
			}
		}
		return v
	}

	var fl []*ssa.Function
	for f := range fs {
		fl = append(fl, f)
	}

	for _, filter := range []func(o []*ssa.Function) (v []*ssa.Function){noTest, samePkg, sameFile, sameName} {
		if v := filter(fl); len(v) > 0 {
			fl = v
		}
	}

	v := map[*ssa.Function]bool{}
	for _, o := range fl {
		if vv, ok := d.NonTestFunction[o.String()]; ok {
			v[vv] = true
		}
	}

	return v
}

const (
	TestModeFull = "full"
	TestModeAuto = "auto"
	TestModeNone = "none"
)

type Opt struct {
	TestMode string   `json:"test_mode"`
	Tag      []string `json:"tag"`

	ExcludePath   []string       `json:"exclude_path"`
	excludePathRe *regexp.Regexp `json:"-"`

	OnlyPath   []string       `json:"only_path"`
	onlyPathRe *regexp.Regexp `json:"-"`

	Force bool `json:"force"`

	IgnoreMock bool `json:"ignore_mock"`

	Before int `json:"before"`

	SimilarConstKeep    bool `json:"similar_const_keep"`
	AdjacentConstKeep   bool `json:"adjacent_const_keep"`
	SimilarConstComment bool `json:"similar_const_comment"`
	SimilarFuncKeep     bool `json:"similar_func_keep"`

	DeleteUnusedGlobalAnonymousV bool `json:"delete_unused_global_anonymous_v"`

	UnsafeDeleteMethod bool `json:"unsafe_delete_method"`

	CalCallGraph bool `json:"cal_call_graph"`

	IgnoreTestErr     bool     `json:"ignore_test_err"`
	IgnoreCompileErrs bool     `json:"ignore_compile_errs"`
	BuildFlags        []string `json:"build_flags"`

	ImportBySourceFile string   `json:"import_by_source_file"`
	OutputChanges      bool     `json:"output_changes"`
	EnabledPassNames   []string `json:"enabled_pass_names"`
	MainDir            string   `json:"main_dir"`
	LoadMainPkgsOnly   bool     `json:"load_main_pkgs_only"`
}
