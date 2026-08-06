package internal

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"unsafe"

	"github.com/masatana/go-textdistance"
	"github.com/sirupsen/logrus"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
)

var is18, _ = isGoVersionLessThan("1.19")

func getTypeObject(t types.Type) types.Object {
	switch typ := t.(type) {
	case *types.Pointer:
		return getTypeObject(typ.Elem())
	case *types.Named:
		return typ.Obj()
	}
	return nil
}

// dependentObjectGetter GetDependentObject 输入是需要保留的Object, 输出是该Object依赖的Object
// 每个需要保留的Object遍历一次就行了
var dependentObjectGetter = DependentObjectGetter{trace: map[types.Type]bool{}}

type DependentObjectGetter struct {
	trace map[types.Type]bool
}

func (d *DependentObjectGetter) GetDependentObject(t types.Type) map[types.Object]bool {
	collect := map[types.Object]bool{}
	getTypeObjectRecursive0(t, d.trace, collect)
	return collect
}

func getTypeObjectRecursive0(t types.Type, trace map[types.Type]bool, collect map[types.Object]bool) {
	if trace == nil {
		trace = map[types.Type]bool{}
	}

	if trace[t] {
		return
	}
	trace[t] = true

	// go1.18及以下，相同的typ返回的对象可能不是同一个(泛形时，尽管类型参数相同)，可能死循环
	if is18 {
		switch typ := t.(type) {
		case *types.Named:
			if typ.Obj() != nil && collect[typ.Obj()] {
				return
			}
		}
	}

	switch typ := t.(type) {
	case *types.Basic:
	case *types.Array:
		getTypeObjectRecursive0(typ.Elem(), trace, collect)
	case *types.Chan:
		getTypeObjectRecursive0(typ.Elem(), trace, collect)
	case *types.Interface:
		for i := 0; i < typ.NumEmbeddeds(); i++ {
			getTypeObjectRecursive0(typ.EmbeddedType(i), trace, collect)
		}
		for i := 0; i < typ.NumMethods(); i++ {
			getTypeObjectRecursive0(typ.Method(i).Type(), trace, collect)
		}
	case *types.Map:
		getTypeObjectRecursive0(typ.Key(), trace, collect)
		getTypeObjectRecursive0(typ.Elem(), trace, collect)
	case *types.Named:
		if typ.Obj() != nil {
			collect[typ.Obj()] = true
		}
		getTypeObjectRecursive0(typ.Underlying(), trace, collect)
		if v, ok := getUnexportedField(typ, "fromRHS").(types.Type); ok {
			getTypeObjectRecursive0(v, trace, collect)
		}
		for i := 0; i < typ.TypeArgs().Len(); i++ {
			getTypeObjectRecursive0(typ.TypeArgs().At(i), trace, collect)
		}
	case *types.Pointer:
		getTypeObjectRecursive0(typ.Elem(), trace, collect)
	case *types.Signature:
		if rev := typ.Recv(); rev != nil {
			getTypeObjectRecursive0(rev.Type(), trace, collect)
		}
		if param := typ.Params(); param != nil {
			for i := 0; i < param.Len(); i++ {
				getTypeObjectRecursive0(param.At(i).Type(), trace, collect)
			}
		}
		if param := typ.Results(); param != nil {
			for i := 0; i < param.Len(); i++ {
				getTypeObjectRecursive0(param.At(i).Type(), trace, collect)
			}
		}
		if param := typ.RecvTypeParams(); param != nil {
			for i := 0; i < param.Len(); i++ {
				getTypeObjectRecursive0(param.At(i), trace, collect)
			}
		}
		if param := typ.TypeParams(); param != nil {
			for i := 0; i < param.Len(); i++ {
				getTypeObjectRecursive0(param.At(i), trace, collect)
			}
		}
	case *types.Slice:
		getTypeObjectRecursive0(typ.Elem(), trace, collect)
	case *types.Struct:
		for i := 0; i < typ.NumFields(); i++ {
			field := typ.Field(i)
			getTypeObjectRecursive0(field.Type(), trace, collect)
		}
	case *types.Tuple:
		for i := 0; i < typ.Len(); i++ {
			field := typ.At(i)
			getTypeObjectRecursive0(field.Type(), trace, collect)
		}
	case *types.TypeParam:
		if typ.Obj() != nil {
			collect[typ.Obj()] = true
		}
		getTypeObjectRecursive0(typ.Underlying(), trace, collect)
		getTypeObjectRecursive0(typ.Constraint(), trace, collect)

	case *types.Union:
		for i := 0; i < typ.Len(); i++ {
			field := typ.Term(i)
			getTypeObjectRecursive0(field.Type(), trace, collect)
		}

	}
}

func getUsedFunc(fn *ssa.Function) (d map[*ssa.Function]bool) {
	d = map[*ssa.Function]bool{} //解析代码块的ssa指令，获取调用函数，标记为使用
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			ops := instr.Operands(nil)
			for _, op := range ops {
				if op == nil || *op == nil {
					continue
				}
				if used, ok := (*op).(*ssa.Function); ok && used != nil {
					d[used] = true
				}
			}
		}
	}

	for _, f := range getInnerFunction(fn) {
		d[f] = true
	}

	return
}

func getInnerFunction(fn *ssa.Function) (inner []*ssa.Function) {
	for _, f := range fn.AnonFuncs {
		inner = append(inner, f)
	}

	// 泛型函数模版
	orig := reflect.ValueOf(fn).Elem().FieldByName("topLevelOrigin").UnsafePointer()
	if orig != nil {
		o := (*ssa.Function)(orig)
		inner = append(inner, o)
	}

	return inner
}

func getUsedFuncRecursive(fn *ssa.Function, d map[*ssa.Function]bool) {
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			ops := instr.Operands(nil)
			for _, op := range ops {
				if op == nil || *op == nil {
					continue
				}
				if used, ok := (*op).(*ssa.Function); ok && used != nil {
					d[used] = true
				}
			}
		}
	}
	// 闭包
	for _, f := range fn.AnonFuncs {
		d[f] = true
		getUsedFuncRecursive(f, d)
	}

	// 泛型函数模版
	orig := reflect.ValueOf(fn).Elem().FieldByName("topLevelOrigin").UnsafePointer()
	if orig != nil {
		o := (*ssa.Function)(orig)
		d[o] = true
		getUsedFuncRecursive(o, d)
	}
	return
}

func getFuncUsedObject(fn *ssa.Function) (d map[types.Object]bool) {
	d = map[types.Object]bool{}
	recv := fn.Signature.Recv()
	if recv != nil && getTypeObject(recv.Type()) != nil {
		d[getTypeObject(recv.Type())] = true
	}

	if param := fn.Signature.Params(); param != nil {
		for i := 0; i < param.Len(); i++ {
			for o := range dependentObjectGetter.GetDependentObject(param.At(i).Type()) {
				d[o] = true
			}
		}
	}
	if param := fn.Signature.Results(); param != nil {
		for i := 0; i < param.Len(); i++ {
			for o := range dependentObjectGetter.GetDependentObject(param.At(i).Type()) {
				d[o] = true
			}
		}
	}

	if param := fn.Signature.TypeParams(); param != nil {
		for i := 0; i < param.Len(); i++ {
			for o := range dependentObjectGetter.GetDependentObject(param.At(i)) {
				d[o] = true
			}
		}
	}

	if param := fn.Signature.RecvTypeParams(); param != nil {
		for i := 0; i < param.Len(); i++ {
			for o := range dependentObjectGetter.GetDependentObject(param.At(i)) {
				d[o] = true
			}
		}
	}

	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			ops := instr.Operands(nil)
			for _, op := range ops {
				if op == nil || *op == nil {
					continue
				}
				switch o := (*op).(type) {
				case *ssa.Global:
					if o != nil && o.Object() != nil {
						d[o.Object()] = true
					}
				}
				for o := range dependentObjectGetter.GetDependentObject((*op).Type()) {
					d[o] = true
				}
			}

			switch ins := instr.(type) {
			case *ssa.ChangeInterface:
				for o := range dependentObjectGetter.GetDependentObject(ins.Type()) {
					d[o] = true
				}
			case *ssa.ChangeType:
				for o := range dependentObjectGetter.GetDependentObject(ins.Type()) {
					d[o] = true
				}
			case *ssa.Convert:
				for o := range dependentObjectGetter.GetDependentObject(ins.Type()) {
					d[o] = true
				}

			}
		}
	}
	return
}

func getObjectUsedObject(obj types.Object) map[types.Object]bool {
	if obj == nil {
		return nil
	}
	return dependentObjectGetter.GetDependentObject(obj.Type())
}

// MustHasGitAndAllCommitted checks if there are any uncommitted changes in the git repository at the given directory.
func MustHasGitAndAllCommitted(dir string) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = dir

	output, err := cmd.Output()
	if err != nil {
		panic(err)
	}

	if len(strings.TrimSpace(string(output))) != 0 {
		panic("please commit modified files first")
	}
}

func collectGeneratePath(opt *Opt) {
	_ = filepath.Walk(".", func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		absPath, err := filepath.Abs(path)
		if err != nil {
			return nil
		}
		bytes := make([]byte, 200)
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		_, err = f.Read(bytes)
		if err != nil {
			return err
		}
		head := strings.ToLower(string(bytes[:200]))
		if strings.Contains(head, "do not edit") && strings.Contains(head, "generate") {
			opt.ExcludePath = append(opt.ExcludePath, absPath)
		}
		return nil
	})
	if opt.IgnoreMock {
		opt.ExcludePath = append(opt.ExcludePath, "mock")
	}
}

func genRe(path []string) *regexp.Regexp {
	var re []string
	for i := range path {
		re = append(re, fmt.Sprintf("(%s)", path[i]))
	}
	return regexp.MustCompile(strings.Join(re, "|"))
}

type constSimilarObject struct {
	obj types.Object
	pos token.Position
}
type constSimilarMgr struct {
	data map[string]map[string]constSimilarObject
}

func newConstSimilarMgr() *constSimilarMgr {
	return &constSimilarMgr{map[string]map[string]constSimilarObject{}}
}

func (c *constSimilarMgr) Register(constObj types.Object, pos token.Position) {
	file := pos.Filename
	if _, ok := c.data[file]; !ok {
		c.data[file] = map[string]constSimilarObject{}
	}
	name := constObj.Name()
	if _, ok := c.data[file][name]; !ok {
		c.data[file][name] = constSimilarObject{constObj, pos}
	}
}

// Similar regards two const definition is similar when
// JaroWinklerDistance > 0.8
// two const is in same file, and they declaration positions are within 5 lines.
func (c *constSimilarMgr) Similar(constObj types.Object, pos token.Position, factor float64) (similar []types.Object) {
	line1 := pos.Line
	for name, o := range c.data[pos.Filename] {
		if o.obj == constObj {
			continue
		}
		if textdistance.JaroWinklerDistance(constObj.Name(), name) < factor {
			continue
		}
		gap := line1 - o.pos.Line
		if gap <= 5 && gap >= -5 {
			similar = append(similar, o.obj)
		}
	}
	return
}

type FuncSimilarMgr struct {
	function map[string][]*ssa.Function
	idx      map[*ssa.Function]int
	source   map[string][]string
	fset     *token.FileSet
}

func NewFuncSimilarMgr() *FuncSimilarMgr {
	return &FuncSimilarMgr{function: map[string][]*ssa.Function{}, idx: map[*ssa.Function]int{}, source: map[string][]string{}}
}

func (m *FuncSimilarMgr) Register(fn *ssa.Function) {
	if fn == nil || fn.Pos() == token.NoPos {
		return
	}
	if fn.Prog != nil && fn.Prog.Fset != nil {
		m.fset = fn.Prog.Fset
	}
	p := m.fset.Position(fn.Pos())
	if _, ok := m.source[p.Filename]; !ok {
		raw, err := os.ReadFile(p.Filename)
		if err != nil {
			fmt.Println(err)
			return
		}
		m.source[p.Filename] = strings.Split(string(raw), "\n")
	}
	m.function[p.Filename] = append(m.function[p.Filename], fn)
}

func (m *FuncSimilarMgr) Build() {
	for _, f := range m.function {
		sort.Slice(f, func(i, j int) bool {
			return f[i].Pos() < f[j].Pos()
		})
	}
	for _, f := range m.function {
		for idx, fn := range f {
			m.idx[fn] = idx
		}
	}
}

func (m *FuncSimilarMgr) Similar(fn *ssa.Function) (similarFn []*ssa.Function) {
	idx, ok := m.idx[fn]
	if !ok {
		return
	}
	pos := m.fset.Position(fn.Pos())
	filename := pos.Filename
	file := m.source[filename]
	functions := m.function[filename]

	line := pos.Line - 1
	if line < 0 || line >= len(file) {
		return
	}
	s := file[line]

	cmp := func(i int) bool {
		dPos := m.fset.Position(functions[i].Pos())
		dLine := dPos.Line - 1
		if dLine-line > 10 || dLine-line < -10 {
			return false
		}
		if dLine < 0 || dLine >= len(file) {
			return false
		}
		d := file[dLine]
		if textdistance.JaroWinklerDistance(s, d) < 0.9 {
			return false
		}
		return true
	}

	for i := idx + 1; i < len(functions); i++ {
		if !cmp(i) {
			break
		}
		similarFn = append(similarFn, functions[i])
	}
	for i := idx - 1; i >= 0; i-- {
		if !cmp(i) {
			break
		}
		similarFn = append(similarFn, functions[i])
	}
	return
}

func containsSubdirectory(dir string) bool {
	files, err := os.ReadDir(dir)
	if err != nil {
		return false
	}

	for _, file := range files {
		if file.IsDir() {
			return true
		}
	}

	return false
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

// isGoVersionLessThan compares current Go version with a specified version.
func isGoVersionLessThan(version string) (bool, error) {
	// 获取当前Go版本，例如 "go1.18.1"
	currentVersion := runtime.Version()
	// 去掉版本字符串的 "go" 前缀
	currentVersion = strings.TrimPrefix(currentVersion, "go")

	// 比较当前版本和给定版本
	return compareGoVersions(currentVersion, version)
}

// compareGoVersions 返回true如果v1 < v2
func compareGoVersions(v1, v2 string) (bool, error) {
	v1Segments := strings.Split(v1, ".")
	v2Segments := strings.Split(v2, ".")

	// 仅取前两部分（主版本和次版本），忽略修订号
	maxSegments := len(v1Segments)
	if len(v2Segments) < maxSegments {
		maxSegments = len(v2Segments)
	}

	for i := 0; i < maxSegments; i++ {
		v1Int, err := strconv.Atoi(v1Segments[i])
		if err != nil {
			return false, err
		}
		v2Int, err := strconv.Atoi(v2Segments[i])
		if err != nil {
			return false, err
		}

		if v1Int < v2Int {
			return true, nil
		} else if v1Int > v2Int {
			return false, nil
		}
	}

	// 如果达到这里，那么到现在为止两个版本是相等的
	// 检查有没有更多的版本号信息来比较
	if len(v1Segments) < len(v2Segments) {
		// 如果v1少于v2的分段数量，继续检查
		return true, nil
	}

	return false, nil
}

// ExtractConstsRelationAdjacent 收集一组常量间的关系，若常量临近且定义类型相同，则视为同一类
func ExtractConstsRelationAdjacent(pkg *packages.Package, constDecl *ast.GenDecl) map[types.Object]map[types.Object]bool {
	relation := map[types.Object]map[types.Object]bool{}
	groups := make([]types.Object, 0)
	addGroup := func() {
		if len(groups) == 0 {
			return
		}
		for _, x := range groups {
			if relation[x] == nil {
				relation[x] = make(map[types.Object]bool)
			}
			for _, y := range groups {
				relation[x][y] = true
			}
		}
		groups = groups[:0]
	}

	allSpecs := constDecl.Specs
	for _, spec := range allSpecs {
		valueSpec, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue // will not happen
		}
		if len(valueSpec.Names) > 0 {
			var tmp []types.Object
			for _, name := range valueSpec.Names {
				obj := pkg.TypesInfo.ObjectOf(name)
				if obj == nil {
					continue
				}
				tmp = append(tmp, obj)
			}
			if len(groups) > 0 && tmp[0].Type() != groups[0].Type() {
				addGroup()
			}
			groups = append(groups, tmp...)
		}
	}
	addGroup()

	return relation
}

// ExtractConstsRelationIota 收集一组常量间的关系，iota定义的一组常量，要么同时保留，要么同时删除
func ExtractConstsRelationIota(pkg *packages.Package, constDecl *ast.GenDecl) map[types.Object]map[types.Object]bool {
	hasIota := false
	var shouldKeep []types.Object

	for _, spec := range constDecl.Specs {
		if _, ok := spec.(*ast.ValueSpec); !ok {
			continue // will not happen
		}
		valueSpec := spec.(*ast.ValueSpec)

		for _, v := range valueSpec.Values {
			if v == nil {
				// will not happen
				hasIota = true
			}
			// 检查是否包含iota
			ast.Inspect(v, func(node ast.Node) bool {
				if ident, ok := node.(*ast.Ident); ok && ident.Name == "iota" {
					hasIota = true
				}

				return true
			})
		}

		for _, name := range valueSpec.Names {
			if o := pkg.TypesInfo.ObjectOf(name); o != nil {
				shouldKeep = append(shouldKeep, o)
			}
		}
	}

	relation := map[types.Object]map[types.Object]bool{}

	if !hasIota {
		return relation
	}

	group := map[types.Object]bool{}
	for _, o := range shouldKeep {
		group[o] = true
	}

	for _, o := range shouldKeep {
		relation[o] = group
	}

	return relation
}

// ExtractConstRelationInheritLast 收集一组常量间的关系，常量b继承常量a时，如果b被保留，则a也被保留
func ExtractConstRelationInheritLast(pkg *packages.Package, constDecl *ast.GenDecl) map[types.Object]map[types.Object]bool {
	relation := map[types.Object]map[types.Object]bool{}

	var last []types.Object

	for _, spec := range constDecl.Specs {
		if _, ok := spec.(*ast.ValueSpec); !ok {
			continue // will not happen
		}
		valueSpec := spec.(*ast.ValueSpec)

		var cur []types.Object
		for _, name := range valueSpec.Names {
			if o := pkg.TypesInfo.ObjectOf(name); o != nil {
				cur = append(cur, o)
			}
		}

		// o = 0
		// a = 1
		// b
		// c
		// b, c 都会继承 a的值
		if len(valueSpec.Names) != len(valueSpec.Values) {
			for _, o := range cur {
				if _, ok := relation[o]; !ok {
					relation[o] = map[types.Object]bool{}
				}
				for _, v := range last {
					relation[o][v] = true
				}
			}
		} else {
			last = cur
		}
	}

	return relation
}

func getUnexportedField(named interface{}, fieldName string) interface{} {
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("getUnexportedField panic: %v", r)
		}
	}()

	val := reflect.ValueOf(named)
	if val.Kind() == reflect.Pointer {
		val = val.Elem()
	}
	field := val.FieldByName(fieldName)

	// 检查字段是否存在和可访问
	if !field.IsValid() {
		panic(fmt.Sprintf("Field %s not found", fieldName))
	}
	if !field.CanInterface() {
		field = reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem()
	}
	return field.Interface()
}

func interfaceType(t types.Type) *types.Interface {
	if v, ok := t.(*types.Interface); ok {
		return v
	}

	if p, ok := t.(*types.Pointer); ok {
		return interfaceType(p.Elem())
	}

	if u := t.Underlying(); u != nil && u != t {
		return interfaceType(u)
	}

	return nil
}

func namedType(t types.Type) *types.Named {
	if named, ok := t.(*types.Named); ok {
		return named
	}

	if p, ok := t.(*types.Pointer); ok {
		return namedType(p.Elem())
	}

	if u := t.Underlying(); u != nil && u != t {
		return namedType(u)
	}

	return nil
}
