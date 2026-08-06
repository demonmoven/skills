package internal

import "golang.org/x/tools/go/ssa"

func (d *Detector) collectASTSSARelation(fn *ssa.Function) {
	for _, bb := range fn.Blocks {
		for _, instr := range bb.Instrs {
			if debug, ok := instr.(*ssa.DebugRef); ok {
				d.ASTExprToSSAValue[debug.Expr] = debug.X
				d.ASTExprInSSAFunc[debug.Expr] = fn
			}

			if makeClosure, ok := instr.(*ssa.MakeClosure); ok {
				for _, v := range makeClosure.Bindings {
					d.ClosureBindingValue[v.Pos()] = v
				}
			}
		}
	}
	for _, anonFunc := range fn.AnonFuncs {
		d.collectASTSSARelation(anonFunc)
	}
}
