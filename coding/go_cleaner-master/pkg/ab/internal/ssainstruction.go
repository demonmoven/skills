package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"text/tabwriter"

	"golang.org/x/tools/go/ssa"
)

func (d *Detector) PrintSSAInstruction() {
	p := NewPrinter()
	for _, pkg := range d.pkgs {
		for _, member := range pkg.Members {
			switch mem := member.(type) {
			case *ssa.Type:
				if mem.Object() != nil {
					obj := mem.Object()
					typ := obj.Type()
					for _, fn := range d.getFuncByRecv(typ) {
						d.printSSAFunction(fn, p)
					}
				}
			case *ssa.Function:
				d.printSSAFunction(mem, p)
			}
		}
	}
	p.Flush()
}

func (d *Detector) printSSAFunction(fn *ssa.Function, p *Printer) {
	wd, _ := filepath.Abs(".")
	for idx, bb := range fn.Blocks {
		p.Print(fmt.Sprintf("#%d", idx))
		for _, instr := range bb.Instrs {
			if _, ok := instr.(*ssa.DebugRef); ok {
				continue
			}
			posStr, _ := filepath.Rel(wd, d.position(instr.Pos()).String())
			p.Print(reflect.TypeOf(instr), instr.String(), posStr)
			//for _, operand := range instr.Operands(nil) {
			//	if operand != nil {
			//		fmt.Println("\t", *operand)
			//	}
			//}
		}
	}
}

type Printer struct {
	data [][]string
}

func NewPrinter() *Printer {
	return &Printer{data: make([][]string, 0)}
}

func (p *Printer) Print(v ...any) {
	var line []string
	for _, i := range v {
		item := fmt.Sprintf("%+v", i)
		if len(item) > 50 {
			item = item[:50] + "...."
		}
		line = append(line, item)
	}
	p.data = append(p.data, line)
}

func (p *Printer) Flush() {
	maxLen := 0
	for _, i := range p.data {
		if maxLen < len(i) {
			maxLen = len(i)
		}
	}
	var lines []string
	for _, i := range p.data {
		for len(i) < maxLen {
			i = append(i, "")
		}
		lines = append(lines, strings.Join(i, "\t"))
	}

	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.StripEscape)
	_, _ = fmt.Fprintln(writer, strings.Join(lines, "\n"))
	_ = writer.Flush()

}
