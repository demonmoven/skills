package searchimport

import (
	"fmt"
	"go/ast"
	"go/token"
	"path/filepath"

	"code.byted.org/analyzers/go_cleaner/pkg/searchimport/internal"
	v2 "code.byted.org/analyzers/go_cleaner/pkg/searchimport/internal/v2"
	"github.com/sirupsen/logrus"
)

type Opt struct {
	*internal.Opt

	method string
}

type Option func(*Opt)

// WithTargetModule 设置target模块名
func WithTargetModule(target string) Option {
	return func(opt *Opt) {
		opt.TargetModule = target
	}
}

// WithOutFile 设置target模块名
func WithOutFile(file string) Option {
	return func(opt *Opt) {
		opt.OutFile = file
	}
}

func WithMethod(m string) Option {
	return func(opt *Opt) {
		opt.method = m
	}
}

func Run(option ...Option) *Output {
	o := &Opt{
		Opt: &internal.Opt{
			TestMode:                     "auto",
			Tag:                          nil,
			ExcludePath:                  []string{"kitex_gen/", "model_gen/", "thrift_gen/", "mock_gen/"},
			OnlyPath:                     nil,
			Force:                        false,
			IgnoreMock:                   false,
			Before:                       0,
			SimilarConstKeep:             false,
			SimilarFuncKeep:              false,
			DeleteUnusedGlobalAnonymousV: true, // 全局默认开启
			IgnoreTestErr:                true,
			CalCallGraph:                 false,
		},
	}
	for _, v := range option {
		v(o)
	}
	cwd, err := filepath.Abs(".")
	if err != nil {
		o.OnlyPath = append(o.OnlyPath, cwd) // 强制只删除当前目录下的文件
	}

	logrus.Infof("option: %+v", o.Opt)

	if o.method == "ast" {
		// FIXME: 在引入monorepo之后，这里"."是否需要更改？
		d := v2.NewDetector(o.Opt, ".")
		if err := d.Print(); err != nil {
			fmt.Printf("d.Print err: %v", err)
			panic(err)
		}
	} else {
		d := internal.NewDetector(o.Opt)
		d.Load()
		d.SearchImport()
		if err := d.Print(); err != nil {
			fmt.Printf("d.Print err: %v", err)
			panic(err)
		}
	}

	return nil
}

type Output struct {
	UnusedLine int
	TotalLine  int
	FSet       *token.FileSet
	Unused     map[ast.Node]string
}
