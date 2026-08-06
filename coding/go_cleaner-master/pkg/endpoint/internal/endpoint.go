package internal

import (
	"fmt"
	"go/ast"
	"go/parser"
	"os"
	"path/filepath"
	"strings"

	"code.byted.org/analyzers/go_cleaner/pkg/core/cerr"
	"code.byted.org/analyzers/go_cleaner/pkg/endpoint/internal/framework"
	"code.byted.org/analyzers/go_cleaner/pkg/endpoint/internal/types"
	"code.byted.org/analyzers/go_cleaner/pkg/endpoint/internal/util"
	"github.com/sirupsen/logrus"
)

type App struct {
	Project *types.Project

	Ps []types.FrameworkProcessor
}

func MustNewApp() *App {
	app := App{
		Ps: []types.FrameworkProcessor{
			&framework.KiteXEPC{},
			&framework.KiteEPC{},
			&framework.GDPRPCEPC{},
			&framework.HertzEPC{},
			&framework.GinexEPC{},
			framework.NewGDPAPIEPC(),
			framework.NewGDPAPIViewEPC(),
		},
	}

	if err := app.parse(); err != nil {
		panic(err)
	}

	return &app
}

func (a *App) Clear(eps []string, handlerPath string) error {
	p, err := a.selectFramework()
	if err != nil {
		return err
	}
	if a.Project.AppRoot != "" && p.Framework() != (&framework.KiteXEPC{}).Framework() {
		return cerr.AppRootNotSupportErr
	}
	logrus.Infof("framework: %s", p.Framework())

	return p.ClearEndpoint(a.Project, eps, handlerPath)
}

func (a *App) Remove(eps []string) error {
	p, err := a.selectFramework()
	if err != nil {
		return err
	}
	if a.Project.AppRoot != "" && p.Framework() != (&framework.KiteXEPC{}).Framework() {
		return cerr.AppRootNotSupportErr
	}
	logrus.Infof("framework: %s", p.Framework())

	return p.RMEndpoint(a.Project, eps)
}

func (a *App) selectFramework() (p types.FrameworkProcessor, err error) {
	for _, i := range a.Ps {
		if i.Determine(a.Project) {
			if p != nil {
				return nil, fmt.Errorf("match multiple framework, %s, %s", p.Framework(), i.Framework())
			}
			p = i
		}
	}

	if p == nil {
		return nil, fmt.Errorf("not match any framework")
	}

	return p, nil
}

func (a *App) parse() error {
	root := "."

	a.Project = types.NewProject()
	astFileMap := make(map[string]*ast.File)

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if strings.HasSuffix(path, ".go") {
			node, err := parser.ParseFile(a.Project.FS, path, nil, parser.ParseComments)
			if err != nil {
				return fmt.Errorf("endpoint cleaner ast parse err: %w", err)
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			astFileMap[rel] = node
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("endpoint cleaner ast parse err: %w", err)
	}

	a.Project.Tree = astFileMap

	for name, f := range a.Project.Tree {
		a.Project.Call[name] = util.AllCall(f)
	}

	return nil
}
