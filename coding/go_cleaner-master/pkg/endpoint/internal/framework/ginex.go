package framework

import (
	"go/ast"
	"strings"

	"code.byted.org/analyzers/go_cleaner/pkg/endpoint/internal/types"
	"code.byted.org/analyzers/go_cleaner/pkg/endpoint/internal/util"
)

type GinexEPC struct {
}

// Determine 判断是Ginex框架，基于
// main.go 里有import code.byted.org/gin/ginex, .Init .Default .Spin
func (k *GinexEPC) Determine(p *types.Project) bool {
	file, ok := p.Tree["main.go"]
	if !ok {
		return false
	}

	if !util.Any(file.Imports, func(spec *ast.ImportSpec) bool {
		return strings.Contains(spec.Path.Value, "code.byted.org/gin/ginex")
	}) {
		return false
	}

	hasInit := false
	hasDefault := false
	hasRun := false

	for _, expr := range p.Call["main.go"] {
		if sel, ok := expr.Fun.(*ast.SelectorExpr); ok {
			right := sel.Sel.Name
			if right == "Init" {
				hasInit = true
			}
			if right == "Default" {
				hasDefault = true
			}
			if right == "Run" {
				hasRun = true
			}
		}
	}

	return hasInit && hasDefault && hasRun
}

func (k *GinexEPC) Framework() string {
	return "Ginex"
}

func (k *GinexEPC) ClearEndpoint(p *types.Project, endpoint []string, handlerPath string) error {
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

func (k *GinexEPC) RMEndpoint(p *types.Project, endpoint []string) error {
	return k.ClearEndpoint(p, endpoint, "")
}
