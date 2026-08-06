package internal

import (
	"bufio"
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"code.byted.org/analyzers/go_cleaner/pkg/core/imports"
	"code.byted.org/analyzers/go_cleaner/pkg/util"
	"github.com/sirupsen/logrus"
)

type blockPair struct {
	start int
	end   int
}

type CommentPos map[string]map[blockPair]blockPair

func NewCommentPos() CommentPos {
	return map[string]map[blockPair]blockPair{}
}

func (c CommentPos) Add(bs, be, cs, ce token.Position) {
	filename := bs.Filename
	if filename != be.Filename || filename != cs.Filename || filename != ce.Filename {
		return
	}
	if _, ok := c[filename]; !ok {
		c[filename] = map[blockPair]blockPair{}
	}
	c[filename][blockPair{bs.Line, be.Line}] = blockPair{cs.Line, ce.Line}
}

func (c CommentPos) GetComment(bs, be token.Position) (s, e *token.Position, ok bool) {
	filename := bs.Filename
	if filename != be.Filename {
		return nil, nil, false
	}
	if _, has := c[filename]; !has {
		return nil, nil, false
	}
	if b, has := c[filename][blockPair{bs.Line, be.Line}]; has {
		s = &token.Position{Filename: filename, Line: b.start}
		e = &token.Position{Filename: filename, Line: b.end}
		return s, e, true
	}
	return nil, nil, false
}

func rmBlankGenDecl(raw []byte) (out []byte, err error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", raw, parser.AllErrors|parser.ParseComments)
	if err != nil {
		return nil, err
	}
	// 删除空的var() const() type(), 以及对应的comment
	var decl []ast.Decl
	var comment []*ast.CommentGroup
	delComment := map[*ast.CommentGroup]bool{}
	for _, dec := range f.Decls {
		if genDecl, ok := dec.(*ast.GenDecl); ok && len(genDecl.Specs) == 0 {
			if genDecl.Doc != nil {
				delComment[genDecl.Doc] = true
			}
			continue
		}
		decl = append(decl, dec)
	}
	f.Decls = decl
	for _, c := range f.Comments {
		if !delComment[c] {
			comment = append(comment, c)
		}
	}
	f.Comments = comment

	bf := &bytes.Buffer{}
	if err = printer.Fprint(bf, fset, f); err != nil {
		return nil, err
	}
	return bf.Bytes(), nil
}

func RmBlackDirRecursive(path string) error {
	items, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	for _, item := range items {
		if item.IsDir() {
			err := RmBlackDirRecursive(filepath.Join(path, item.Name()))
			if err != nil {
				return err
			}
		}
	}

	items, err = os.ReadDir(path)
	if err != nil {
		return err
	}

	if len(items) == 0 {
		if err := os.Remove(path); err != nil {
			return err
		}
	}
	return nil
}

func canDelSafely(raw []byte) bool {
	var err error

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "", raw, parser.AllErrors|parser.ParseComments)
	if err != nil {
		return false
	}

	del := true

	ast.Inspect(node, func(n ast.Node) bool {
		switch v := n.(type) {
		case *ast.TypeSpec, *ast.ValueSpec:
			del = false
			return false
		case *ast.FuncDecl:
			del = false
			return false
		case *ast.ImportSpec:
			if v.Name != nil && v.Name.Name == "_" {
				del = false
				return false
			}
		default:
		}
		return true
	})

	return del
}

type fixCleaner struct {
	test             bool
	tags             []string
	buildFlags       []string
	unusedImportData map[string][]int

	undeclaredData         map[string][]int // fix: var _ undeclaredI  = (*undeclaredX)(nil)
	importPathNotExistData map[string][]int // fix: import _ code.byted.org/x/self/not/exist/path
}

func newFixCleaner(test bool, tags []string, buildFlags []string) *fixCleaner {
	u := &fixCleaner{
		unusedImportData:       map[string][]int{},
		undeclaredData:         map[string][]int{},
		importPathNotExistData: map[string][]int{},
		tags:                   tags,
		buildFlags:             buildFlags,
		test:                   test,
	}
	u.init()
	return u
}

func (u *fixCleaner) init() {
	logrus.Info("fixCleaner.init start...")
	defer func() {
		logrus.Info("fixCleaner.init done!")
	}()

	var opts []LoadOption
	if u.test {
		opts = append(opts, WithTest())
	}
	opts = append(opts, WithBuildFlags(u.buildFlags, u.tags))
	opts = append(opts, WithCachedOverlay())

	pkgs, _ := loadPackage(opts...)
	for _, p := range pkgs {
		for _, e := range p.TypeErrors {
			pos := e.Fset.Position(e.Pos)
			filename := pos.Filename
			line := pos.Line
			logrus.Info("TypeErrors: %s", e.Msg)
			if strings.HasSuffix(e.Msg, "imported but not used") || strings.HasSuffix(e.Msg, "imported and not used") {
				u.unusedImportData[filename] = append(u.unusedImportData[filename], line)
			} else if strings.HasPrefix(e.Msg, "undeclared name:") || strings.HasPrefix(e.Msg, "undefined:") {
				u.undeclaredData[filename] = append(u.undeclaredData[filename], line)
			} else if strings.HasPrefix(e.Msg, "could not import") {
				u.importPathNotExistData[filename] = append(u.importPathNotExistData[filename], line)
			}
		}
	}
}

func (u *fixCleaner) clean() {
	logrus.Info("fixCleaner.clean start...")
	defer func() {
		logrus.Info("fixCleaner.clean done!")
	}()

	del := func(s string) bool {
		return true
	}
	delVarUnderscore := func(s string) bool {
		//删除前缀空格以及\t
		s = strings.TrimLeft(s, " \t")
		return strings.HasPrefix(s, "var _ ") || strings.HasPrefix(s, "_ ")
	}

	file := map[string]bool{}
	for f := range u.unusedImportData {
		file[f] = true
	}
	for f := range u.undeclaredData {
		file[f] = true
	}
	for f := range u.importPathNotExistData {
		file[f] = true
	}
	for f := range file {
		lineMap := map[int]filter{}
		for _, i := range u.unusedImportData[f] {
			lineMap[i] = del
		}
		for _, i := range u.undeclaredData[f] {
			lineMap[i] = delVarUnderscore
		}
		for _, i := range u.importPathNotExistData[f] {
			lineMap[i] = del
		}
		if err := deleteLine(f, lineMap); err != nil {
			fmt.Println("fix cleaner err: ", err)
		}
	}
}

type filter func(string) bool

func deleteLine(path string, line map[int]filter) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}

	idx := 1
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		text := scanner.Text()
		if del, ok := line[idx]; !ok || !del(text) {
			lines = append(lines, text)
		}
		idx++
	}
	_ = file.Close()

	keptBytes := []byte(strings.Join(lines, "\n"))
	keptBytes, err = imports.Process(path, keptBytes, nil)
	if err != nil {
		return err
	}

	return os.WriteFile(path, keptBytes, 0644)
}

func deleteGoInDir(dir string) error {
	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, file := range files {
		path := filepath.Join(dir, file.Name())
		if file.IsDir() {
			if err = deleteGoInDir(path); err != nil {
				return err
			}
		}
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".go") {
			if util.IsCleanerIntegrationTestEnv() {
				logrus.Infof("Remove directory %s", path)
			}

			if err = os.Remove(path); err != nil {
				return err
			}
		}
	}

	return nil
}
