package repoinfo

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"code.byted.org/analyzers/go_cleaner/pkg/core/cerr"
	"github.com/sirupsen/logrus"
	"golang.org/x/tools/go/packages"
)

const RenamedMainByCleanerPrefix = "renamed_main_by_cleaner"

type Project struct {
	IsMultiModules bool
	IsMonoRepo     bool
	GoModDirs      []string
	GoModToDirs    map[string]string
	AbsPath        string
	Modules        []*Module
	ValidMods      []*Module

	EnableDeleteEmptyGoFiles bool
}

func NewDefaultProject() *Project {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	return NewProject(dir)
}

func NewProject(dir string) *Project {
	goModDirs := getGoModDirs(dir)
	abspath, err := filepath.Abs(dir)
	if err != nil {
		panic(err)
	}

	p := &Project{
		IsMultiModules: len(goModDirs) > 1,
		IsMonoRepo:     len(goModDirs) > 1, // 多go.mod必然是monorepo, 单go.mod的情况在加载module时最终确定
		GoModDirs:      goModDirs,
		AbsPath:        abspath,
		GoModToDirs:    make(map[string]string),

		EnableDeleteEmptyGoFiles: true,
	}

	for _, d := range goModDirs {
		p.addGoModules(d)
	}

	return p
}

func (p *Project) Load(configs []*ModuleConfig) (anySucc bool) {
	total := len(configs)
	numOfMainFuncs := 0
	logrus.Infof("Current project directory is %s", p.AbsPath)
	for i, c := range configs {
		prefix := fmt.Sprintf("[%d/%d]", i+1, total)

		m := NewModule(p, c)

		logrus.Infof("%s Loading module '%s' in directory %s", prefix, m.Name, m.Dir)

		if p.EnableDeleteEmptyGoFiles {
			p.removeEmptyGoFiles(m.Dir)
		}

		if m.Verify(true) || m.CanRecover() {
			p.ValidMods = append(p.ValidMods, m)
			anySucc = true
		} else {
			logrus.Infof("%s Module %s is broken", prefix, m.Name)
		}
		p.Modules = append(p.Modules, m)
		numOfMainFuncs += m.NumOfMainFuncs
	}

	if numOfMainFuncs > 1 {
		p.IsMonoRepo = true
	}

	return
}

func (p *Project) VerifyAllValidModules() cerr.Error {
	total := len(p.ValidMods)

	for i, m := range p.ValidMods {
		logrus.Infof("[%d/%d] Verifying module %s", i+1, total, m.Name)
		m.Verify(true)

		if m.LoadError != nil {
			return m.LoadError
		}
	}

	return nil
}

func (p *Project) ResetRenamedMains() {
	if p == nil {
		// 避免任务失败时引发panic
		return
	}

	rename := func(file *ast.File) (changed bool) {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}

			if strings.HasPrefix(fn.Name.Name, RenamedMainByCleanerPrefix) {
				fn.Name.Name = "main"
				changed = true
			}
		}
		return
	}

	for _, m := range p.ValidMods {
		for _, p := range m.AllPackages {
			for _, f := range p.Syntax {
				if !rename(f) {
					continue
				}

				fset := p.Fset
				filename := fset.Position(f.Pos()).Filename
				WriteToFile(fset, f, filename)
			}
		}
	}
}

func (p *Project) IsMain(name string) bool {
	return name == "main" || strings.HasPrefix(name, RenamedMainByCleanerPrefix)
}

func (p *Project) FilterOutModDirs(module string) (result []string) {
	isExist := func(path string) bool {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return false
		}
		return true
	}

	for name, dir := range p.GoModToDirs {
		if name != module && isExist(dir) {
			result = append(result, dir)
		}
	}

	if !p.IsMultiModules {
		return
	}

	// 考虑多go.mod带来的跨module依赖场景：
	//
	//		monorepo/repo1 <- /go/pkg/mod/repo2 <- monorepo/repo3
	//
	// 如果不提供跨仓依赖信息/go/pkg/mod/repo2，则在清理完monorepo/repo1后，再清理monorepo/repo3时会报错（因为加载repo2时会失败）,
	// 清理工具会在编译验证阶段报错：undefined xxx等
	//
	// 解决办法：
	// 1. 如果已提供跨仓依赖信息，importBy.txt被提供给cleaner，避免了清理monorepo/repo1中的相关符号，目前清理平台应该已经支持
	// 2. 如果未提供跨仓依赖信息，需要建立monorepo的packages依赖关系，并找出monorepo/repo3依赖的repo2，将相关符号视为已被引用

	allDepPkgs := make(map[string]*packages.Package)

	for _, m := range p.Modules {
		for _, p := range m.Packages {
			for name, dep := range p.Imports {
				allDepPkgs[name] = dep
			}
		}
	}

	logrus.Infof("Totally %d dependency packages", len(allDepPkgs))
	dirs := make(map[string]bool)
	for _, p := range allDepPkgs {
		if p.Module != nil && isExist(p.Module.Dir) {
			dirs[p.Module.Dir] = true
		}
	}

	logrus.Infof("Totally %d dependency modules", len(dirs))
	for dir := range dirs {
		result = append(result, dir)
	}

	return
}

func (p *Project) addGoModules(directory string) {
	if _, err := os.Stat(directory); os.IsNotExist(err) {
		cerr.ExitWithDetail(fmt.Errorf("directory %s is not found", directory), cerr.GoModDirNotFoundErr)
	}

	args := []string{
		"go",
		"list",
		"-f",
		"{{.Path}} {{.Dir}}",
		"-m",
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stderr = os.Stderr
	cmd.Dir = directory

	output, err := cmd.Output()

	if err != nil {
		command := strings.Join(args[1:], " ")
		cerr.ExitWithDetail(fmt.Errorf("failed to run '%s' in directory %s: %s", command, directory, err), cerr.GoModFileParsingErr)
	}

	lines := strings.TrimSpace(string(output))

	for _, line := range strings.Split(lines, "\n") {
		name, dir := strings.Split(line, " ")[0], strings.Split(line, " ")[1]
		p.GoModToDirs[name] = dir
	}
}

func (p *Project) removeEmptyGoFiles(directory string) {
	emptyfiles := make(map[string]bool)
	visit := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		if !IsValidGoSourceFile(path, nil) {
			emptyfiles[path] = true
		}

		return nil
	}

	filepath.Walk(directory, visit)

	for file := range emptyfiles {
		logrus.Infof("Removing empty go file %s", file)
		os.Remove(file)
	}
}

func (p *Project) GetModuleNames(maindir string) (nameToDirs map[string]string) {
	if maindir == "" && p.IsMultiModules {
		return p.GoModToDirs
	}

	nameToDirs = make(map[string]string)

	if maindir != "" {
		logrus.Infof("Main Dir: %s", maindir)
		dir, mod, err := ModuleDirAndNameOf(maindir)
		if err != nil {
			cerr.ExitWithDetail(err, cerr.GoModDirNotFoundErr)
		}
		nameToDirs[mod] = dir
	} else {
		mod, err := getModuleName(p.AbsPath)
		if err != nil {
			cerr.ExitWithDetail(err, cerr.GoModDirNotFoundErr)
		}

		nameToDirs[mod] = p.AbsPath
	}

	return
}

func getGoModDirs(root string) (result []string) {
	dirs := make(map[string]bool)
	visit := func(path string, _ os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if strings.Contains(path, "/vendor/") {
			return nil
		}

		if filepath.Base(path) == "go.mod" {
			dirs[filepath.Dir(path)] = true
		}

		return nil
	}

	filepath.Walk(root, visit)
	for d := range dirs {
		result = append(result, d)
	}

	return
}

func IsValidGoSourceFile(path string, src any) bool {
	isValid := func(code []byte) bool {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "", code, parser.AllErrors|parser.ParseComments)
		if err != nil {
			return false
		}

		return file != nil
	}

	if src != nil {
		bs, err := sourceToBytes(src)
		if err != nil {
			return false
		}
		return isValid(bs)
	}

	if bytes, err := os.ReadFile(path); err == nil {
		return isValid(bytes)
	}

	return false
}

func ParseFile(fset *token.FileSet, filename string, src any) (*ast.File, error) {
	if !IsValidGoSourceFile(filename, src) {
		return nil, nil
	}

	const mode = parser.AllErrors | parser.ParseComments
	return parser.ParseFile(fset, filename, src, mode)
}

func ParseFileWithoutAllErrs(fset *token.FileSet, filename string, src any) (*ast.File, error) {
	if !IsValidGoSourceFile(filename, src) {
		return nil, nil
	}

	return parser.ParseFile(fset, filename, src, parser.ParseComments)
}

func sourceToBytes(src any) ([]byte, error) {
	switch s := src.(type) {
	case string:
		return []byte(s), nil
	case []byte:
		return s, nil
	case *bytes.Buffer:
		// is io.Reader, but src is already available in []byte form
		if s != nil {
			return s.Bytes(), nil
		}
	case io.Reader:
		return io.ReadAll(s)
	}
	return nil, errors.New("invalid source")
}
