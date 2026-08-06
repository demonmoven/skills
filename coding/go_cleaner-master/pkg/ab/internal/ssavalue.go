package internal

import (
	"fmt"
	"go/constant"
	"go/token"
	"strconv"
	"strings"

	"golang.org/x/tools/go/ssa"
)

func (d *Detector) Value(v ssa.Value) (vs []constant.Value) {
	if v == nil {
		return []constant.Value{VariantValue}
	}
	v.Pos()
	if vs, ok := d.vs[v]; ok && len(vs) > 0 {
		return vs
	}

	d.vs[v] = []constant.Value{VariantValue} // 防止死循环
out:
	switch vv := v.(type) {
	case *ssa.Const:
		vs = append(vs, vv.Value)

	case *ssa.BinOp:
		for _, x := range d.Value(vv.X) {
			if x == VariantValue {
				vs = nil
				break out
			}
			for _, y := range d.Value(vv.Y) {
				if y == VariantValue {
					vs = nil
					break out
				}
				func() {
					defer func() {
						if r := recover(); r != nil {
							fmt.Println("const value cal err: ", r)
							vs = append(vs, VariantValue)
						}
						d.vsConnected[v] = d.vsConnected[v] || d.vsConnected[vv.X] || d.vsConnected[vv.Y]
					}()
					switch vv.Op {
					case token.ADD, token.MUL, token.SUB, token.QUO, token.REM:
						vs = append(vs, constant.BinaryOp(x, vv.Op, y))
					case token.EQL, token.NEQ, token.LSS, token.LEQ, token.GTR, token.GEQ:
						vs = append(vs, constant.MakeBool(constant.Compare(x, vv.Op, y)))
					case token.SHL, token.SHR:
						if yv, ok := constant.Uint64Val(y); ok {
							vs = append(vs, constant.Shift(x, vv.Op, uint(yv)))
						} else {
							vs = append(vs, VariantValue)
						}
					default:
						vs = append(vs, constant.BinaryOp(x, vv.Op, y))
					}
				}()
			}
		}

	case *ssa.Phi:
		for _, ele := range vv.Edges {
			vs = append(vs, d.Value(ele)...)
			d.vsConnected[v] = d.vsConnected[v] || d.vsConnected[ele]
		}

	case *ssa.Alloc, *ssa.Global:
		for _, store := range d.store[vv] {
			for _, x := range d.Value(store.Val) {
				vs = append(vs, x)
			}
			d.vsConnected[v] = d.vsConnected[v] || d.vsConnected[store.Val]
		}

	case *ssa.Slice:
		vs = append(vs, tran(d.SliceValue(vv))...)

	case *ssa.FieldAddr:
		stru := []ssa.Value{vv.X}

		if arg, ok := vv.X.(*ssa.Parameter); ok {
			for _, o := range d.argValue(arg) {
				stru = append(stru, o)
			}
		}
		if c, ok := vv.X.(*ssa.Call); ok {
			for _, o := range d.callValue(c, 0) {
				stru = append(stru, o)
			}
		}

		if ss, ok := d.store[vv.X]; ok {
			for _, s := range ss {
				stru = append(stru, s.Val)
				if arg, ok := s.Val.(*ssa.Parameter); ok {
					for _, o := range d.argValue(arg) {
						stru = append(stru, o)
					}
				}
				if c, ok := s.Val.(*ssa.Call); ok {
					for _, o := range d.callValue(c, 0) {
						stru = append(stru, o)
					}
				}
			}
		}
		for _, s := range stru {
			if _, ok := d.filedStore[s]; !ok {
				continue
			}
			if _, ok := d.filedStore[s][vv.Field]; !ok {
				continue
			}
			for _, store := range d.filedStore[s][vv.Field] {
				for _, o := range d.Value(store.Val) {
					vs = append(vs, o)
				}
				d.vsConnected[v] = d.vsConnected[v] || d.vsConnected[store.Val]
			}
		}

	case *ssa.IndexAddr:
		x := vv.X
		if vvx, ok := x.(*ssa.Slice); ok {
			x = vvx.X
		}
		if _, ok := d.filedStore[x]; !ok {
			break
		}
		if c, ok := vv.Index.(*ssa.Const); ok && c.Value.Kind() == constant.Int {
			idx, _ := strconv.ParseInt(c.Value.String(), 10, 64)
			if _, ok := d.filedStore[x][int(idx)]; !ok {
				break
			}
			for _, store := range d.filedStore[x][int(idx)] {
				for _, o := range d.Value(store.Val) {
					vs = append(vs, o)
				}
				d.vsConnected[v] = d.vsConnected[v] || d.vsConnected[store.Val]
			}
		}

	case *ssa.Field:
		if _, ok := d.filedStore[vv.X]; !ok {
			break
		}
		if _, ok := d.filedStore[vv.X][vv.Field]; !ok {
			break
		}
		for _, store := range d.filedStore[vv.X][vv.Field] {
			for _, o := range d.Value(store.Val) {
				vs = append(vs, o)
			}
			d.vsConnected[v] = d.vsConnected[v] || d.vsConnected[store.Val]
		}

	case *ssa.UnOp:
		switch vv.Op {
		case token.MUL:
			for _, o := range d.Value(vv.X) {
				vs = append(vs, o)
			}
			d.vsConnected[v] = d.vsConnected[v] || d.vsConnected[vv.X]
		case token.NOT:
			func() {
				defer func() {
					if r := recover(); r != nil {
						vs = nil
					}
				}()

				for _, o := range d.Value(vv.X) {
					vs = append(vs, constant.UnaryOp(vv.Op, o, 1))
				}
				d.vsConnected[v] = d.vsConnected[v] || d.vsConnected[vv.X]
			}()

		default:
			vs = append(vs, VariantValue)
		}

	case *ssa.Parameter:
		for _, arg := range d.argValue(vv) {
			for _, x := range d.Value(arg) {
				vs = append(vs, x)
			}
			d.vsConnected[v] = d.vsConnected[v] || d.vsConnected[arg]
		}

	case *ssa.FreeVar:
		if bindingV, ok := d.ClosureBindingValue[vv.Pos()]; ok {
			for _, x := range d.Value(bindingV) {
				vs = append(vs, x)
			}
			d.vsConnected[v] = d.vsConnected[v] || d.vsConnected[bindingV]
		}
		for _, store := range d.store[vv] {
			for _, x := range d.Value(store.Val) {
				vs = append(vs, x)
			}
			d.vsConnected[v] = d.vsConnected[v] || d.vsConnected[store.Val]
		}

	case *ssa.Call:
		// 单一返回值
		if val, ok := d.determineABCall(vv); ok {
			vs = append(vs, val...)
			if d.abEntryCallPosEffective[v.Pos()].Expired() {
				d.vsConnected[v] = true
				d.opt.ModifiedFunction.Add(vv.Parent()) // 这个函数调用的废弃的实验参数，那么这个函数就可能需要被refactor
			}
		} else {
			for _, ret := range d.callValue(vv, 0) {
				for _, o := range d.Value(ret) {
					vs = append(vs, o)
				}
				d.vsConnected[v] = d.vsConnected[v] || d.vsConnected[ret]
				if fn, ok := vv.Call.Value.(*ssa.Function); ok && d.opt.ModifiedFunction.Has(fn) {
					d.vsConnected[v] = true
					d.opt.ModifiedFunction.Add(vv.Parent()) // 这个函数调用了需要refactor的函数，那这个函数也可能需要refactor
				}
			}
		}

	case *ssa.Extract:
		// 多返回值
		if c, ok := vv.Tuple.(*ssa.Call); ok {
			idx := vv.Index
			for _, ret := range d.callValue(c, idx) {
				for _, o := range d.Value(ret) {
					vs = append(vs, o)
				}
				d.vsConnected[v] = d.vsConnected[v] || d.vsConnected[ret]
			}
		}

		if c, ok := vv.Tuple.(*ssa.TypeAssert); ok {
			if vv.Index == 0 {
				for _, o := range d.Value(c) {
					vs = append(vs, o)
				}
				d.vsConnected[v] = d.vsConnected[v] || d.vsConnected[vv.Tuple]
			} else if vv.Index == 1 {
				if call, ok := c.X.(*ssa.Call); ok && d.abEntryCallPosEffective[call.Pos()].Expired() {
					vs = append(vs, constant.MakeBool(true)) // TODO: 认为类型断言总是成功的
					d.vsConnected[v] = true
				}
			}
		}

	case *ssa.MakeInterface:
		for _, o := range d.Value(vv.X) {
			vs = append(vs, o)
		}
		d.vsConnected[v] = d.vsConnected[v] || d.vsConnected[vv.X]

	case *ssa.ChangeInterface:
		for _, o := range d.Value(vv.X) {
			vs = append(vs, o)
		}
		d.vsConnected[v] = d.vsConnected[v] || d.vsConnected[vv.X]

	case *ssa.ChangeType:
		for _, o := range d.Value(vv.X) {
			vs = append(vs, o)
		}
		d.vsConnected[v] = d.vsConnected[v] || d.vsConnected[vv.X]

	case *ssa.Convert:
		for _, o := range d.Value(vv.X) {
			vs = append(vs, o)
		}
		d.vsConnected[v] = d.vsConnected[v] || d.vsConnected[vv.X]

	case *ssa.TypeAssert:
		for _, o := range d.Value(vv.X) {
			vs = append(vs, o)
		}
		d.vsConnected[v] = d.vsConnected[v] || d.vsConnected[vv.X]

	default:
		vs = append(vs, VariantValue)

	}

	if len(vs) == 0 {
		vs = append(vs, VariantValue)
	}

	visited := map[string]bool{}
	var vsUniq []constant.Value
	for _, v := range vs {
		if v == nil {
			continue
		}
		if _, ok := visited[v.ExactString()]; !ok {
			visited[v.ExactString()] = true
			vsUniq = append(vsUniq, v)
		}
	}
	vs = vsUniq

	d.vs[v] = vs

	return
}

var VariantValue = constant.MakeString("_unknown_")

func (d *Detector) SliceValue(v *ssa.Slice) (vs map[int][]constant.Value) {
	vs = map[int][]constant.Value{}
	for _, v := range d.parseSetStore(v.X) {
		for idx, stores := range d.indexStore[v] {
			for _, store := range stores {
				for _, o := range d.Value(store.Val) {
					vs[idx] = append(vs[idx], o)
				}
				d.vsConnected[v] = d.vsConnected[v] || d.vsConnected[store.Val]
			}
		}
	}
	return
}

func tran(vs map[int][]constant.Value) []constant.Value {
	var res []constant.Value
	var tmp []string
	n := -1
	for k := range vs {
		if n < k {
			n = k
		}
	}
	var recur func(int)
	recur = func(i int) {
		if i > n {
			res = append(res, constant.MakeString(strings.Join(tmp, ".")))
			return
		}
		s := vs[i]
		if len(s) == 0 {
			s = []constant.Value{constant.MakeString("")}
		}
		for _, val := range s {
			tmp = append(tmp, strings.ReplaceAll(val.ExactString(), "\"", ""))
			recur(i + 1)
			tmp = tmp[:len(tmp)-1]
		}
	}

	recur(0)
	return res
}
