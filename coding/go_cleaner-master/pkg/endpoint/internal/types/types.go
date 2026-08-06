package types

import (
	"go/ast"
	"go/token"
	"path/filepath"
)

type Project struct {
	Tree map[string]*ast.File
	FS   *token.FileSet

	AppRoot string
	Call    map[string][]*ast.CallExpr
}

func NewProject() *Project {
	return &Project{
		Tree: make(map[string]*ast.File),
		FS:   token.NewFileSet(),
		Call: make(map[string][]*ast.CallExpr),
	}
}

func (p *Project) AppRootFile(name string) string {
	return filepath.Join(p.AppRoot, name)
}

type FrameworkProcessor interface {
	FrameworkDeterminer
	EPCleaner

	Framework() string
}

type FrameworkDeterminer interface {
	Determine(p *Project) bool
}

type EPCleaner interface {
	RMEndpoint(p *Project, endpoint []string) error
	ClearEndpoint(p *Project, endpoint []string, path string) error
}
