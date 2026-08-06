package repoinfo

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"code.byted.org/analyzers/go_cleaner/pkg/core/cerr"
	"github.com/sirupsen/logrus"
	"golang.org/x/tools/go/packages"
)

type ModuleConfig struct {
	Name   string
	Dir    string
	Config *packages.Config

	IgnoreTestErrs   bool
	MaxNumRetry      int
	LoadMainPkgsOnly bool
}

type Module struct {
	ModuleConfig

	LoadError      cerr.Error
	Packages       []*packages.Package
	AllPackages    []*packages.Package
	ReferedIdents  map[string]struct{}
	IsSyntaxOnly   bool
	Project        *Project
	NumOfMainFuncs int
}

func NewModule(proj *Project, config *ModuleConfig) *Module {
	m := &Module{
		ModuleConfig:  *config,
		ReferedIdents: make(map[string]struct{}),
		Project:       proj,
	}

	return m
}

func (m *Module) IsReferedSyntaxOnly(name string) bool {
	_, ok := m.ReferedIdents[name]
	return ok
}

func (m *Module) Verify(verbose bool) bool {
	m.NumOfMainFuncs = 0

	if info, err := os.Stat(m.Config.Dir); os.IsNotExist(err) || !info.IsDir() {
		cerr.ExitWithDetail(err, cerr.GoModDirNotFoundErr)
	}

	if err := runGoModTidy(m.Config.Dir); err != nil {
		if verbose {
			logrus.Infof("%s\n", err.Error())
		}

		m.parseSyntaxOnly()
		m.LoadError = cerr.GoModTidyErr
		return false
	}

	var err error

	if m.Packages, err = m.load(); err != nil {
		if verbose {
			logrus.Infof("%s\n", err.Error())
		}
		m.parseSyntaxOnly()
		m.LoadError = cerr.ToolPreloadInternalErr
		return false
	}

	if m.IgnoreTestErrs {
		if err := m.tryBestToRepairTestErrs(); err != nil {
			if verbose {
				logrus.Infof("%s\n", err.Error())
			}
			m.parseSyntaxOnly()
			m.LoadError = cerr.ToolPreloadIgnoreTestErr
			return false
		}
	}

	for _, pkg := range m.Packages {
		for _, file := range pkg.Syntax {
			ast.Inspect(file, func(node ast.Node) bool {
				if fn, ok := node.(*ast.FuncDecl); ok {
					if fn.Name.Name == "main" {
						m.NumOfMainFuncs++
					}
				}
				return true
			})
		}
	}

	printErrors := func() (n int) {
		packages.Visit(m.Packages, nil, func(pkg *packages.Package) {
			for _, err := range pkg.Errors {
				if verbose && !m.IsRecoverableError(err.Msg) {
					fmt.Fprintln(os.Stderr, err)
				}
				n++
			}
		})
		return
	}

	if n := printErrors(); n > 0 {
		m.LoadError = cerr.ToolPreloadPackageContainsErr
		return false
	}

	m.LoadError = nil
	return true
}

func (m *Module) IsRecoverableError(msg string) bool {
	var fixableBugsRegexp []*regexp.Regexp
	fixableBugsRegexp = append(fixableBugsRegexp, UnusedImportRegexp...)
	fixableBugsRegexp = append(fixableBugsRegexp, UnusedVariableRegexp...)
	fixableBugsRegexp = append(fixableBugsRegexp, RedeclaredMains...)

	for _, re := range fixableBugsRegexp {
		match := re.FindStringSubmatch(msg)
		if len(match) > 0 {
			return true
		}
	}

	return false
}

func (m *Module) CanRecover() bool {
	fixable := true
	packages.Visit(m.Packages, nil, func(pkg *packages.Package) {
		if !fixable {
			return
		}

		for _, err := range pkg.TypeErrors {
			if !m.IsRecoverableError(err.Msg) {
				fixable = false
				return
			}
		}
	})

	return fixable
}

func (m *Module) load() ([]*packages.Package, error) {
	initial, err := packages.Load(m.Config, GetQueriesIgnoredByGoList(m.Dir)...)
	m.AllPackages = initial
	if err == nil && m.LoadMainPkgsOnly {
		return GetPkgsUsedByMain(initial), nil
	}
	return initial, err
}

func (m *Module) tryBestToRepairTestErrs() (err error) {
	// 排除存在错误的test文件, 将含有语法错误的test文件存入overlay中，
	// 后续就会自动排除这些test文件
	if m.Config.Overlay == nil {
		m.Config.Overlay = make(map[string][]byte)
	}

	haveToCheckReloading := true
	// 如果发现有错误文件被搜集到，则进行重新加载，直到没有错误文件或者达到最大尝试次数
	for i := 0; haveToCheckReloading && i < m.MaxNumRetry+1; i++ {
		// 此分支必然不会在第一次循环时命中
		if len(m.Config.Overlay) > 0 {
			if m.Packages, err = m.load(); err != nil {
				return
			}
		}

		// 检查测试文件是否存在编译错误
		overlay, err := gatherErrTestFile(m.Packages)
		if err != nil {
			return fmt.Errorf("gatherErrTestFile err: %w", err)
		}

		// 更新overlay，并避免出现重复的文件
		haveToCheckReloading = false
		for path, data := range overlay {
			abspath, err := filepath.Abs(path)
			// FIXME: 这种错误是否属于内部错误，是否该直接panic？
			if err != nil {
				cerr.ExitWithDetail(err, cerr.ToolPreloadIgnoreTestErr)
			}

			if _, ok := m.Config.Overlay[abspath]; !ok {
				m.Config.Overlay[abspath] = data
				haveToCheckReloading = true
			}
		}
	}
	return
}

// 检查是否有被go list忽略的文件
// https://pkg.go.dev/cmd/go#hdr-Package_lists_and_patterns
func GetQueriesIgnoredByGoList(directory string) (queries []string) {
	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !strings.HasSuffix(info.Name(), ".go") {
			return nil
		}

		hasIgnoredPrefix := false
		for _, splited := range strings.Split(path, string(os.PathSeparator)) {
			if strings.HasPrefix(splited, ".") || strings.HasPrefix(splited, "_") {
				hasIgnoredPrefix = true
			}
		}

		if hasIgnoredPrefix {
			queries = append(queries, "file="+path)
		}

		return nil
	})

	if err != nil {
		cerr.ExitWithDetail(err, cerr.ToolPreloadIgnoredByGoListErr)
	}

	queries = append(queries, "./...")
	return
}

func gatherErrTestFile(pkgs []*packages.Package) (overLay map[string][]byte, err error) {
	overLay = make(map[string][]byte)

	firstLine := func(path string) ([]byte, error) {
		file, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		if scanner.Scan() {
			return scanner.Bytes(), nil
		}

		if err := scanner.Err(); err != nil {
			return nil, err
		}

		return nil, nil
	}

	for _, p := range pkgs {
		for _, e := range p.Errors {
			seps := strings.Split(e.Pos, ":")
			if len(seps) > 0 && strings.HasSuffix(seps[0], "_test.go") {
				file := seps[0]
				line, err := firstLine(file)
				if err != nil {
					return nil, err
				}
				overLay[file] = line
			}
		}
	}
	return overLay, nil
}

func runGoModTidy(directory string) error {
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = directory

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()

	if err == nil {
		os.Stdout.Write(out.Bytes())
		return nil
	}

	output := out.String()
	re := regexp.MustCompile(`go: go\.mod file indicates go (\d+\.\d+), but maximum version supported by tidy is (\d+\.\d+)`)
	matches := re.FindStringSubmatch(output)

	var versionInGoModFile string
	var versionOfGoModTidy string

	if len(matches) >= 3 {
		versionInGoModFile = matches[1]
		versionOfGoModTidy = matches[2]

		os.Stderr.WriteString(fmt.Sprintf(
			"go.mod file indicates go %s, but maximum version supported by tidy is %s. skip 'go mod tidy'\n",
			versionInGoModFile, versionOfGoModTidy,
		))

		return nil
	}

	os.Stderr.Write(out.Bytes())
	return err
}

func (m *Module) addSyntaxUsageOfModule(f *ast.File) {
	ast.Inspect(f, func(node ast.Node) bool {
		if ident, ok := node.(*ast.Ident); ok {
			m.ReferedIdents[ident.Name] = struct{}{}
		}

		if fn, ok := node.(*ast.FuncDecl); ok {
			if fn.Name.Name == "main" {
				m.NumOfMainFuncs++
			}
		}

		return true
	})
}

func (m *Module) parseSyntaxOnly() {
	fset := token.NewFileSet()

	err := filepath.Walk(m.Dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		if !IsValidGoSourceFile(path, nil) {
			return nil
		}

		f, parseErr := ParseFileWithoutAllErrs(fset, path, nil)
		if parseErr != nil {
			return parseErr
		}

		m.addSyntaxUsageOfModule(f)
		return nil
	})

	if err != nil {
		cerr.ExitWithDetail(err, cerr.GoParserSyntaxErr)
	}

	m.IsSyntaxOnly = true
}

func getModuleName(path string) (string, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	goModPath := filepath.Join(path, "go.mod")
	_, err = os.Stat(goModPath)
	if err != nil {
		if path != "/" && filepath.Dir(path) != path {
			return getModuleName(filepath.Dir(path))
		}
		return "", err
	}
	content, err := os.ReadFile(goModPath)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(content), "\n") {
		if strings.HasPrefix(line, "module") {
			for _, i := range strings.Split(strings.ReplaceAll(line, "\t", " "), " ") {
				if i != "" && i != "module" {
					return i, nil
				}

			}
		}
	}
	return "", errors.New("cannot parse module name")
}
