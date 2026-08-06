package storage

import (
	"bufio"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"os"
	"strings"

	"code.byted.org/analyzers/go_cleaner/pkg/core"
	"code.byted.org/analyzers/go_cleaner/pkg/core/repoinfo"
	"code.byted.org/lang/gg/gmap"
	"code.byted.org/lang/gg/gslice"
	"github.com/sirupsen/logrus"
	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

type StorageMatcher interface {
	Match(psm string) bool
	GetInitializers(psm string) []ssa.CallInstruction
	ListUsers(psm string) []UserInfo
	GetEdits(psm string) []core.TextEdit
}

type UserInfo struct {
	Position string `json:"position"`
	Code     string `json:"code"`
	ModName  string `json:"mod,omitempty"`
	Verison  string `json:"version,omitempty"`
}

type scenario struct {
	pkgPath      string
	initFun      string
	initArgIndex int
	userFuns     []string
}

func (s *scenario) HintInit(fn *ssa.Function) bool {
	if fn.Pkg == nil || fn.Pkg.Pkg == nil {
		return false
	}
	return fn.Pkg.Pkg.Path() == s.pkgPath && fn.Name() == s.initFun
}

func (s *scenario) HintUser(fn *ssa.Function) bool {
	if fn.Pkg == nil || fn.Pkg.Pkg == nil {
		return false
	}
	return fn.Pkg.Pkg.Path() == s.pkgPath && gslice.Contains(s.userFuns, fn.Name())
}

func (s *scenario) MaybeHintInit(fs []*ssa.Function) bool {
	for _, fn := range fs {
		if s.HintInit(fn) {
			return true
		}
	}
	return false
}

func (s *scenario) MaybeHintUser(fs []*ssa.Function) bool {
	for _, fn := range fs {
		if s.HintUser(fn) {
			return true
		}
	}
	return false
}

type DefaultMatcher struct {
	prog         *ssa.Program
	pkgmap       map[*types.Package]*packages.Package
	ssapkgs      []*ssa.Package
	cg           *callgraph.Graph
	initializers map[string][]ssa.CallInstruction
	scenarios    []*scenario
	focus        map[*types.Package]bool
}

func (m *DefaultMatcher) Callees(call ssa.CallInstruction) (result []*ssa.Function) {
	if call == nil || call.Parent() == nil {
		return
	}

	for _, edge := range m.cg.Nodes[call.Parent()].Out {
		if edge.Site != call {
			continue
		}

		if callee := edge.Callee.Func; callee != nil {
			result = append(result, callee)
		}
	}

	return result
}

func (m *DefaultMatcher) matchCall(psm string, call ssa.CallInstruction) bool {
	callees := m.Callees(call)

	for _, s := range m.scenarios {
		if !s.MaybeHintInit(callees) || len(call.Common().Args) <= s.initArgIndex {
			continue
		}

		value := call.Common().Args[s.initArgIndex]
		c, ok := value.(*ssa.Const)
		if !ok || c.Value == nil {
			continue
		}

		if c.Value.Kind() != constant.String || c.Value.String() != "\""+psm+"\"" {
			continue
		}

		m.initializers[psm] = append(m.initializers[psm], call)
	}

	return len(m.initializers) > 0
}

func (m *DefaultMatcher) IsFocused(fn *ssa.Function) bool {
	return fn.Pkg != nil && fn.Pkg.Pkg != nil && m.focus[fn.Pkg.Pkg]
}

func (m *DefaultMatcher) Match(psm string) bool {
	logrus.Infof("Start matching %s ...", psm)
	for fn := range ssautil.AllFunctions(m.prog) {
		if !m.IsFocused(fn) {
			continue
		}

		for _, block := range fn.Blocks {
			for _, ins := range block.Instrs {
				if call, ok := ins.(ssa.CallInstruction); ok {
					m.matchCall(psm, call)
				}
			}
		}
	}

	return len(m.initializers) > 0
}

func (m *DefaultMatcher) GetInitializers(psm string) []ssa.CallInstruction {
	inits, _ := m.initializers[psm]
	return inits
}

func (m *DefaultMatcher) refers(member ssa.Value) (result []ssa.Instruction) {
	for fn := range ssautil.AllFunctions(m.prog) {
		if !m.IsFocused(fn) {
			continue
		}

		for _, bb := range fn.Blocks {
			for _, inst := range bb.Instrs {
				if v, ok := inst.(*ssa.Store); ok && v.Addr == member && v.Referrers() != nil {
					result = append(result, *v.Referrers()...)
				}

				v, ok := inst.(*ssa.UnOp)
				if !ok {
					continue
				}

				if g, ok := v.X.(*ssa.Global); ok && g == member && v.Referrers() != nil {
					result = append(result, *v.Referrers()...)
				}
			}
		}
	}

	return
}

func (m *DefaultMatcher) listUserMemberRef(vals []ssa.Value) (result []ssa.Instruction) {
	users := make(map[ssa.Instruction]bool)
	for _, v := range vals {
		var insts []ssa.Instruction

		if _, ok := v.(*ssa.Global); ok {
			insts = m.refers(v)
		}

		if _, ok := v.(*ssa.Const); ok {
			insts = m.refers(v)
		}

		for _, i := range insts {
			users[i] = true
		}
	}

	for u := range users {
		result = append(result, u)
	}

	return
}

func (m *DefaultMatcher) listUserCallInsts(vals []ssa.Value) (result []ssa.CallInstruction) {
	users := make(map[ssa.CallInstruction]bool)
	for _, v := range vals {
		var insts []ssa.Instruction

		if v.Referrers() == nil {
			continue
		}

		insts = *v.Referrers()

		for _, user := range insts {
			call, ok := user.(ssa.CallInstruction)
			if !ok {
				continue
			}

			callees := m.Callees(call)

			for _, s := range m.scenarios {
				if s.MaybeHintUser(callees) {
					users[call] = true
				}
			}
		}
	}

	for _, call := range result {
		result = append(result, call)
	}

	return
}

func (m *DefaultMatcher) listUserGlobals(psm string) (users []*ssa.Global) {
	for _, initcall := range m.initializers[psm] {
		// TODO: support *ssa.Go and *ssa.Defer
		retval := initcall.Value()
		for _, global := range getExtractedGlobalValues(retval) {
			users = append(users, global)
		}
	}

	return
}

func globalsToValues(globals []*ssa.Global) (values []ssa.Value) {
	for _, g := range globals {
		values = append(values, g)
	}
	return values
}

func (m *DefaultMatcher) ListUsers(psm string) (result []UserInfo) {
	getPkgInfo := func(p *types.Package) (pkginfo PackageInfo) {
		if p == nil {
			return
		}

		pkg := m.pkgmap[p]
		pkginfo.Name = pkg.Name
		if pkg.Module != nil {
			pkginfo.Module = pkg.Module.Path
			pkginfo.Version = pkg.Module.Version
		}
		return
	}

	positions := make(map[token.Position]PackageInfo)

	globals := m.listUserGlobals(psm)
	for _, user := range gslice.Uniq(globals) {
		positions[m.prog.Fset.Position(user.Pos())] = getPkgInfo(user.Pkg.Pkg)
	}

	for _, user := range m.listUserMemberRef(globalsToValues(globals)) {
		positions[m.prog.Fset.Position(user.Pos())] = getPkgInfo(user.Parent().Pkg.Pkg)
	}

	for _, user := range m.initializers[psm] {
		positions[m.prog.Fset.Position(user.Pos())] = getPkgInfo(user.Parent().Pkg.Pkg)
	}

	for pos, pkginfo := range positions {
		result = append(result, UserInfo{
			Position: pos.String(),
			Code:     extractCode(pos.Filename, pos.Line),
			ModName:  pkginfo.Module,
			Verison:  pkginfo.Version,
		})
	}

	return gslice.Uniq(result)
}

func (m *DefaultMatcher) GetEdits(psm string) (edits []core.TextEdit) {
	if _, ok := m.initializers[psm]; !ok {
		return
	}

	// 全局变量的定义直接可删除
	globals := m.listUserGlobals(psm)
	for _, user := range gslice.Uniq(globals) {
		scope := user.Object().Parent()
		if scope != nil && scope.Len() == 1 {
			edits = append(edits, core.TextEdit{
				Beg: scope.Pos(),
				End: scope.End(),
			})
		}

		edits = append(edits, core.TextEdit{
			Beg: user.Pos(),
			End: token.NoPos,
		})
	}

	// TODO: 删除初始化代码
	for _, initfn := range m.initializers[psm] {
		ssapkg := initfn.Parent().Pkg

		repopkg := &repoinfo.Package{
			Pkg:      m.pkgmap[ssapkg.Pkg],
			TypeInfo: ssapkg.Pkg,
			PkgSSA:   ssapkg,
		}

		for _, n := range repopkg.Path(initfn.Pos()) {
			edits = append(edits, core.TextEdit{
				Beg: n.Pos(),
				End: token.NoPos,
			})
			// 只需要删除语法树中最靠近的那段源码
			break
		}
	}

	// TODO: 删除使用的位置，如果出现在被函数入口支配的代码块，则函数体进行置空
	decls := make(map[*ast.FuncDecl]*types.Signature)
	for _, inst := range m.listUserMemberRef(globalsToValues(globals)) {
		fn := inst.Parent()
		bb := inst.Block()
		decl, ok := fn.Syntax().(*ast.FuncDecl)
		if ok && fn.Blocks[0].Dominates(bb) {
			decls[decl] = fn.Signature
		}
	}

	for decl, sig := range decls {
		tup := sig.Results()

		var text []string
		for i := 0; i < tup.Len(); i++ {
			text = append(text, typeDefaultValue(tup.At(i).Type()))
		}

		minpos := decl.Body.List[0].Pos()
		maxpos := token.NoPos
		for _, stmt := range decl.Body.List {
			if maxpos < stmt.End() {
				maxpos = stmt.End()
			}
		}

		var newtext []byte
		if len(text) > 0 {
			newtext = append(newtext, []byte("\treturn ")...)
			newtext = append(newtext, []byte(strings.Join(text, ", "))...)
		} else {
			newtext = append(newtext, []byte("\treturn")...)
		}

		edits = append(edits, core.TextEdit{
			Beg:     minpos,
			End:     maxpos,
			NewText: newtext,
		})
	}

	return edits
}

func extractCode(filename string, line int) string {
	file, err := os.Open(filename)
	if err != nil {
		logrus.Infof("open file %s failed: %s", filename, err.Error())
		return ""
	}

	defer file.Close()
	scanner := bufio.NewScanner(file)
	currentLine := 1
	for scanner.Scan() {
		if currentLine == line {
			return strings.TrimSpace(scanner.Text())
		}
		currentLine++
	}

	if err := scanner.Err(); err != nil {
		logrus.Infof("scan file %s failed: %s", filename, err.Error())
	}

	return ""
}

func getExtractedGlobalValues(call *ssa.Call) []*ssa.Global {
	if call == nil || call.Referrers() == nil {
		return nil
	}

	globals := make(map[*ssa.Global]bool)
	for _, inst := range *call.Referrers() {
		extract, ok := inst.(*ssa.Extract)
		if !ok || extract.Referrers() == nil {
			return nil
		}

		for _, user := range *extract.Referrers() {
			store, ok := user.(*ssa.Store)
			if !ok {
				continue
			}

			if g, ok := store.Addr.(*ssa.Global); ok {
				globals[g] = true
			}
		}
	}

	return gmap.Keys(globals)
}

func typeDefaultValue(t types.Type) string {
	switch e := t.(type) {
	case *types.Pointer, *types.Array, *types.Interface:
		return "nil"
	case *types.Basic:
		if e.Kind() == types.String {
			return ""
		}
		if e.Kind() == types.Bool {
			return "false"
		}
		if e.Kind() == types.Int {
			return "0"
		}
	case *types.Struct:
		parts := strings.Split(e.String(), "/")
		return parts[len(parts)-1] + "{}"
	default:
	}

	if t.Underlying() != nil {
		return typeDefaultValue(t.Underlying())
	}

	return t.String()
}
