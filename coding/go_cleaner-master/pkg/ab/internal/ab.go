package internal

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"

	"code.byted.org/analyzers/go_cleaner/pkg/ab/internal/model"
)

type Opt struct {
	ExpiredKey []string
	Only       []string
	onlyRe     *regexp.Regexp
	Before     int
	test       bool
	tag        []string

	Entry []*model.ABEntry

	ModifiedFunction *ModifiedFunctionTracker
}

type Detector struct {
	module string

	prog        *ssa.Program
	pkgs        []*ssa.Package
	initialPkgs []*packages.Package

	cg *callgraph.Graph

	// 保存ssa.Value的可能值
	vs map[ssa.Value][]constant.Value
	// 保存函数参数的可能值，只用于debug
	args map[ssa.Value][]constant.Value

	vsConnected map[ssa.Value]bool

	AllFunction map[*ssa.Function]bool

	EntryFinder             *EntryFinder
	abEntryCallSite         map[ssa.CallInstruction]*ABEntryParamSig
	abEntryCallPos          map[token.Pos]*ABEntryParamSig
	abEntryCallPosEffective map[token.Pos]KeyEffectiveness

	KeyChecker KeyEffectChecker

	// ast.node 关联 ssa.Value
	ASTExprToSSAValue   map[ast.Expr]ssa.Value
	ASTExprInSSAFunc    map[ast.Expr]*ssa.Function
	ClosureBindingValue map[token.Pos]ssa.Value // 闭包捕获的自由变量 ident pos => ssa.Value
	// 保留ast.Node的所有可能值
	nodeVs map[ast.Expr][]constant.Value

	// 存储删除行，用于删除注释
	deleteLines map[string]map[int]bool

	store      map[ssa.Value][]*ssa.Store
	filedStore map[ssa.Value]map[int][]*ssa.Store
	indexStore map[ssa.Value]map[int][]*ssa.Store

	blame *Blame

	opt Opt
}

func NewDetector(o Opt) *Detector {
	module, _ := GetModuleName()
	o.onlyRe = genRe(o.Only)
	d := &Detector{
		module: module,
		prog:   nil,
		pkgs:   nil,
		cg:     nil,

		vs:   map[ssa.Value][]constant.Value{},
		args: map[ssa.Value][]constant.Value{},

		vsConnected: map[ssa.Value]bool{},

		AllFunction: map[*ssa.Function]bool{},

		EntryFinder:             NewEntryFinder(o.Entry),
		abEntryCallSite:         map[ssa.CallInstruction]*ABEntryParamSig{},
		abEntryCallPos:          map[token.Pos]*ABEntryParamSig{},
		abEntryCallPosEffective: map[token.Pos]KeyEffectiveness{},

		ASTExprToSSAValue:   map[ast.Expr]ssa.Value{},
		ASTExprInSSAFunc:    map[ast.Expr]*ssa.Function{},
		ClosureBindingValue: map[token.Pos]ssa.Value{},
		nodeVs:              map[ast.Expr][]constant.Value{},

		deleteLines: map[string]map[int]bool{},

		store:      map[ssa.Value][]*ssa.Store{},
		filedStore: map[ssa.Value]map[int][]*ssa.Store{},
		indexStore: map[ssa.Value]map[int][]*ssa.Store{},

		blame: NewBlame(),

		opt: o,
	}
	if len(o.ExpiredKey) > 0 {
		d.KeyChecker = newKeyEffectAssignChecker(o.ExpiredKey)
	} else {
		d.KeyChecker = newKeyEffectClickHouseChecker()
	}

	return d
}

func (d *Detector) Load() {
	fmt.Println("load...")
	var err error
	var opts []LoadOption
	opts = append(opts, WithFileSet())
	if d.opt.test {
		opts = append(opts, WithTest())
	}
	if len(d.opt.tag) > 0 {
		opts = append(opts, WithBuildTag(d.opt.tag))
	}
	opts = append(opts, WithStaticAlg())
	prog, initailPkgs, pkgs, cg, err := Load(opts...)
	if err != nil {
		fmt.Println("fail, err:", err)
		panic(err)
	}
	d.prog, d.pkgs, d.initialPkgs, d.cg = prog, pkgs, initailPkgs, cg

	fmt.Println("ok!")
}

// CollectAll 收集
// 所有的函数
// ssa.Value, ssa.Args debug使用
// collectStore
// collectABCall
func (d *Detector) CollectAll() {
	for _, pkg := range d.prog.AllPackages() {
		for _, member := range pkg.Members {
			switch mem := member.(type) {
			case *ssa.Type:
				if mem.Object() != nil {
					obj := mem.Object()
					typ := obj.Type()
					for _, fn := range d.getFuncByRecv(typ) {
						d.AllFunction[fn] = true
					}
				}
			case *ssa.Global:
				if mem.Object() != nil {
					d.vs[mem] = []constant.Value{}
				}
			case *ssa.NamedConst:
				if mem.Object() != nil {
					d.vs[mem.Value] = []constant.Value{}
				}
			case *ssa.Function:
				d.AllFunction[mem] = true
			}
		}
	}

	var closures []*ssa.Function
	for fn := range d.AllFunction {
		for _, f := range getAnonFun(fn) {
			closures = append(closures, f)
		}
	}
	for _, fn := range closures {
		d.AllFunction[fn] = true
	}

	for fn := range d.AllFunction {
		d.collectASTSSARelation(fn)
		if !d.IsProjectFunc(fn) {
			continue
		}
		for _, v := range fn.Params {
			d.vs[v] = []constant.Value{}
		}
		for _, v := range fn.Locals {
			d.vs[v] = []constant.Value{}
		}

		for _, bb := range fn.Blocks {
			for _, instr := range bb.Instrs {
				if ifInstr, ok := instr.(*ssa.If); ok {
					d.vs[ifInstr.Cond] = []constant.Value{}
				}
				if callInstr, ok := instr.(*ssa.Call); ok {
					for _, arg := range callInstr.Call.Args {
						d.vs[arg] = []constant.Value{}
						d.args[arg] = []constant.Value{}
					}
				}
			}
		}
	}

	d.collectStore()
	d.collectABCall()
	d.collectExpiredABCall()
}

func (d *Detector) collectStore() {
	for fn := range d.AllFunction {
		if !d.IsProjectFunc(fn) {
			continue
		}
		for _, bb := range fn.Blocks {
			for _, instr := range bb.Instrs {
				if store, ok := instr.(*ssa.Store); ok {
					d.store[store.Addr] = append(d.store[store.Addr], store)
					if _, ok := store.Addr.(*ssa.FreeVar); ok && d.ClosureBindingValue[store.Addr.Pos()] != nil {
						d.store[d.ClosureBindingValue[store.Addr.Pos()]] = append(d.store[d.ClosureBindingValue[store.Addr.Pos()]], store)
					}
					switch addr := store.Addr.(type) {
					case *ssa.FieldAddr:
						v := addr.X
						idx := addr.Field
						for _, vv := range d.parseSetStore(v) {
							if _, ok := d.filedStore[vv]; !ok {
								d.filedStore[vv] = map[int][]*ssa.Store{}
							}
							d.filedStore[vv][idx] = append(d.filedStore[vv][idx], store)
						}

					case *ssa.IndexAddr:
						v := addr.X
						if _, ok := v.(*ssa.Slice); ok {
							v = v.(*ssa.Slice).X
						}
						idx := addr.Index
						if c, ok := idx.(*ssa.Const); ok && c.Value.Kind() == constant.Int {
							for _, vv := range d.parseSetStore(v) {
								if _, ok := d.indexStore[vv]; !ok {
									d.indexStore[vv] = map[int][]*ssa.Store{}
								}
								idx, _ := strconv.ParseInt(c.Value.String(), 10, 64)
								d.indexStore[vv][int(idx)] = append(d.indexStore[vv][int(idx)], store)
							}

						}

					}
				}
			}
		}
	}
}

func (d *Detector) DebugArgValue() {
	for v := range d.args {
		if len(d.Value(v)) > 0 && strings.Contains(d.Value(v)[0].String(), "default") {
			fmt.Println(v)
		}
		fmt.Println(d.position(v.Pos()), d.Value(v))
	}

	for v := range d.vs {
		if strings.Contains(d.position(v.Pos()).Filename, "abtest_clean_demo") {
			if len(d.Value(v)) > 0 && strings.Contains(d.Value(v)[0].String(), "default") {
				fmt.Println(v)
			}
			fmt.Println(d.position(v.Pos()), d.Value(v))
		}
	}
}
