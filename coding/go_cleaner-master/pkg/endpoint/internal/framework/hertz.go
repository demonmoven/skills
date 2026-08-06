package framework

import (
	"go/ast"
	"strings"

	"code.byted.org/analyzers/go_cleaner/pkg/endpoint/internal/types"
	"code.byted.org/analyzers/go_cleaner/pkg/endpoint/internal/util"
)

type HertzEPC struct {
}

// Determine 判断是Hertz框架，基于
// main.go 里有import code.byted.org/middleware/hertz/byted, .Init .Default .Spin
func (k *HertzEPC) Determine(p *types.Project) bool {
	file, ok := p.Tree["main.go"]
	if !ok {
		return false
	}

	if !util.Any(file.Imports, func(spec *ast.ImportSpec) bool {
		return strings.Contains(spec.Path.Value, "code.byted.org/middleware/hertz/byted")
	}) {
		return false
	}

	hasInit := false
	hasDefault := false
	hasSpin := false

	for _, expr := range p.Call["main.go"] {
		if sel, ok := expr.Fun.(*ast.SelectorExpr); ok {
			right := sel.Sel.Name
			if right == "Init" {
				hasInit = true
			}
			if right == "Default" {
				hasDefault = true
			}
			if right == "Spin" {
				hasSpin = true
			}
		}
	}

	return hasInit && hasDefault && hasSpin
}

func (k *HertzEPC) Framework() string {
	return "Hertz"
}

func (k *HertzEPC) ClearEndpoint(p *types.Project, endpoint []string, handlerPath string) error {
	for path, file := range p.Tree {
		if strings.Contains(path, "/") {
			continue
		}

		if !util.ForwardMatchAndRewrite(file, endpoint) {
			util.BackwardMatchAndRewrite(file, endpoint)
		}

		if err := util.Save(file, p.FS); err != nil {
			return err
		}
	}
	return nil
}

func (k *HertzEPC) RMEndpoint(p *types.Project, endpoint []string) error {
	return k.ClearEndpoint(p, endpoint, "")
}
