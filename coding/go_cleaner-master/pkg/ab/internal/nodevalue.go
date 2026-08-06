package internal

import (
	"go/ast"
	"go/constant"
	"go/token"
	"strconv"
)

func (d *Detector) NodeValue(exp ast.Expr) []constant.Value {
	if exp == nil {
		return []constant.Value{VariantValue}
	}
	if vs, ok := d.nodeVs[exp]; ok {
		return vs
	}
	vs := d.NodeValue0(exp)
	d.nodeVs[exp] = vs
	return vs
}

func (d *Detector) NodeValue0(exp ast.Expr) []constant.Value {
	vs, connected := d.NodeValue00(exp)
	if connected {
		return vs
	}
	return []constant.Value{VariantValue}
}

func (d *Detector) NodeValue00(exp ast.Expr) ([]constant.Value, bool) {
	switch v := exp.(type) {
	case *ast.BasicLit:
		switch v.Kind {
		case token.INT:
			if v, err := strconv.ParseInt(v.Value, 0, 64); err == nil {
				return []constant.Value{constant.MakeInt64(v)}, true
			}
		case token.FLOAT:
			if v, err := strconv.ParseFloat(v.Value, 64); err == nil {
				return []constant.Value{constant.MakeFloat64(v)}, true
			}
		case token.STRING:
			return []constant.Value{constant.MakeString(v.Value)}, true
		}

	case *ast.Ident:
		if v.Obj == nil {
			if v.Name == "true" {
				return []constant.Value{constant.MakeBool(true)}, true
			} else if v.Name == "false" {
				return []constant.Value{constant.MakeBool(false)}, true
			}
		} else if v.Obj.Kind == ast.Con {
			if valueSpec, ok := v.Obj.Decl.(*ast.ValueSpec); ok {
				for idx, vv := range valueSpec.Names {
					if vv == v && idx < len(valueSpec.Values) {
						return d.NodeValue00(valueSpec.Values[idx])
					}
				}
			}
		}

	case *ast.BinaryExpr:
		switch v.Op {
		case token.LAND:
			lv, lconnected := d.NodeValue00(v.X)
			rv, rconnected := d.NodeValue00(v.Y)
			if mustTrue(lv) && mustTrue(rv) {
				return []constant.Value{constant.MakeBool(true)}, lconnected || rconnected
			}
			if mustFalse(lv) || mustFalse(rv) {
				return []constant.Value{constant.MakeBool(false)}, lconnected || rconnected
			}
			return []constant.Value{constant.MakeBool(true), constant.MakeBool(false)}, lconnected || rconnected
		case token.LOR:
			lv, lconnected := d.NodeValue00(v.X)
			rv, rconnected := d.NodeValue00(v.Y)
			if mustTrue(lv) || mustTrue(rv) {
				return []constant.Value{constant.MakeBool(true)}, lconnected || rconnected
			}
			if mustFalse(lv) && mustFalse(rv) {
				return []constant.Value{constant.MakeBool(false)}, lconnected || rconnected
			}
			return []constant.Value{constant.MakeBool(true), constant.MakeBool(false)}, lconnected || rconnected
		}
	case *ast.UnaryExpr:
		if v.Op == token.NOT {
			vs, connected := d.NodeValue00(v.X)
			if mustTrue(vs) {
				return []constant.Value{constant.MakeBool(false)}, connected
			}
			if mustFalse(vs) {
				return []constant.Value{constant.MakeBool(true)}, connected
			}
			return []constant.Value{constant.MakeBool(true), constant.MakeBool(false)}, connected
		}

	case *ast.ParenExpr:
		return d.NodeValue00(v.X)

	}
	ssaValue := d.ASTExprToSSAValue[exp]
	if ssaValue == nil {
		return []constant.Value{VariantValue}, false
	}
	vs := d.Value(ssaValue)
	return vs, d.vsConnected[ssaValue] || d.opt.ModifiedFunction.Has(d.ASTExprInSSAFunc[exp]) // 这个exp在需要被refactor的函数之中，那么认为connect为true
}
