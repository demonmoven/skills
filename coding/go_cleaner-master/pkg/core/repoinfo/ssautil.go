package repoinfo

import (
	"fmt"

	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/callgraph/rta"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func AllFunctions(prog *ssa.Program) (result []*ssa.Function) {
	for fn := range ssautil.AllFunctions(prog) {
		if fn.Synthetic == "" {
			result = append(result, fn)
		}
	}

	return
}

func AllInstructions(prog *ssa.Program) (result []ssa.Instruction) {
	for _, fn := range AllFunctions(prog) {
		for _, block := range fn.Blocks {
			result = append(result, block.Instrs...)
		}
	}
	return
}

func AllReferedFunctions(prog *ssa.Program) (map[*ssa.Function]bool) {
	allinsts := AllInstructions(prog)

	result := make(map[*ssa.Function]bool)

	for _, inst := range allinsts {
		mkclosure, ok := inst.(*ssa.MakeClosure)
		if !ok {
			continue
		}

		wrapper, ok := mkclosure.Fn.(*ssa.Function)
		if !ok {
			continue
		}

		fn := wrapper
		if wrapper.Synthetic != "" {
			if call, ok := wrapper.Blocks[0].Instrs[0].(*ssa.Call); ok {
				fn = call.Common().StaticCallee()
			}
		}

		if fn != nil {
			result[fn] = true
		}
	}

	return result
}

func Members(prog *ssa.Program, name string) (members []ssa.Member, pkgs []*ssa.Package) {
	for _, pkg := range prog.AllPackages() {
		member, ok := pkg.Members[name]
		if !ok {
			continue
		}
		members = append(members, member)
		pkgs = append(pkgs, pkg)
	}
	return
}

func AllFnRefs(prog *ssa.Program) {

}

func CallGraph(prog *ssa.Program) (cg *callgraph.Graph, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("call graph internal error: %v", r)
		}
	}()

	cg = rta.Analyze(AllFunctions(prog), true).CallGraph
	return
}
