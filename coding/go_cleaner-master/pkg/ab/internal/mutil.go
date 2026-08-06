package internal

import (
	"go/constant"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
)

// argValue 获取所有实参value
func (d *Detector) argValue(param *ssa.Parameter) (vs []ssa.Value) {
	pID := 0
	for i, p := range param.Parent().Params {
		if p == param {
			pID = i
			break
		}
	}
	if n, ok := d.cg.Nodes[param.Parent()]; ok {
		for _, caller := range n.In {
			args := caller.Site.Common().Args
			if len(args) > pID {
				vs = append(vs, args[pID])
				if o, ok := args[pID].(*ssa.UnOp); ok {
					vs = append(vs, o.X)
				}
			}

		}
	}
	return
}

// callValue 获取call的返回值
func (d *Detector) callValue(c *ssa.Call, i int) (vs []ssa.Value) {
	if fn, ok := c.Call.Value.(*ssa.Function); ok {
		hasExpired := false
		for _, bb := range fn.Blocks {
			for _, instr := range bb.Instrs {
				if ret, ok := instr.(*ssa.Return); ok {
					if len(ret.Results) > i {
						vs = append(vs, ret.Results[i])
					}
				}

				if call, ok := instr.(*ssa.Call); ok {
					if _, ok := d.determineABCall(call); ok {
						if d.abEntryCallPosEffective[call.Pos()].Expired() {
							hasExpired = true
						}
					}
				}
			}
		}

		if hasExpired {
			d.opt.ModifiedFunction.Add(fn) // 这个函数获取了废弃实验参数，这个函数可能需要被refactor
		}
	}
	return
}

// parseSetStore 如果参数是指针,slice,map,参数会修改原来的值
// collectStore 时需要使用
func (d *Detector) parseSetStore(v ssa.Value) (vs []ssa.Value) {
	vs = append(vs, v)

	q := []ssa.Value{v}
	visited := map[ssa.Value]bool{v: true}

	for len(q) > 0 {
		v = q[0]
		if len(q) > 1 {
			q = q[1:]
		} else {
			q = []ssa.Value{}
		}
		if param, ok := v.(*ssa.Parameter); ok {
			switch param.Type().(type) {
			case *types.Pointer, *types.Slice, *types.Map:
				for _, vv := range d.argValue(param) {
					if !visited[vv] {
						visited[vv] = true
						q = append(q, vv)
						vs = append(vs, vv)
					}
				}
			}
		}
	}
	return
}

// mustTrue returns true if value `v` can and only can be `true`.
func (d *Detector) mustTrue(v ssa.Value) bool {
	if v == nil {
		return false
	}
	vs := d.Value(v)
	if len(vs) == 0 {
		return false
	}
	for _, v := range vs {
		if v.Kind() != constant.Bool || constant.BoolVal(v) != true {
			return false
		}
	}
	return true
}

// mustTrue returns true if value `v` can and only can be `false`.
func (d *Detector) mustFalse(v ssa.Value) bool {
	if v == nil {
		return false
	}
	vs := d.Value(v)
	if len(vs) == 0 {
		return false
	}
	for _, v := range vs {
		if v.Kind() != constant.Bool || constant.BoolVal(v) != false {
			return false
		}
	}
	return true
}

func (d *Detector) getFuncByRecv(typ types.Type) []*ssa.Function {
	var fns []*ssa.Function

	methods := d.prog.MethodSets.MethodSet(typ)
	for i := 0; i < methods.Len(); i++ {
		sel := methods.At(i)
		fn := d.prog.MethodValue(sel)
		if fn == nil {
			continue
		}
		fns = append(fns, fn)
	}

	methods = d.prog.MethodSets.MethodSet(types.NewPointer(typ))
	for i := 0; i < methods.Len(); i++ {
		sel := methods.At(i)
		fn := d.prog.MethodValue(sel)
		if fn == nil {
			continue
		}
		fns = append(fns, fn)
	}

	if t, ok := typ.(*types.Pointer); ok {
		methods = d.prog.MethodSets.MethodSet(t.Elem())
		for i := 0; i < methods.Len(); i++ {
			sel := methods.At(i)
			fn := d.prog.MethodValue(sel)
			if fn == nil {
				continue
			}
			fns = append(fns, fn)
		}
	}

	return fns
}

func (d *Detector) IsProjectFunc(fn *ssa.Function) bool {
	if fn == nil {
		return false
	}
	if d.IsProjectPkg(fn.Pkg) {
		return true
	}
	if fn.Parent() != nil && d.IsProjectFunc(fn.Parent()) {
		return true
	}
	return false
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
	return d.IsProjectPkg0(pkg.PkgPath, d.module)
}

func (d *Detector) IsProjectPkg3(pkg *types.Package) bool {
	if pkg == nil {
		return false
	}
	return d.IsProjectPkg0(pkg.Path(), d.module)
}

func (d *Detector) IsProjectPkg0(pkgPath, module string) bool {
	return strings.HasPrefix(pkgPath, module)
}

func (d *Detector) position(pos token.Pos) token.Position {
	return d.prog.Fset.Position(pos)
}
