package internal

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"strings"

	"code.byted.org/analyzers/go_cleaner/pkg/core/imports"
	"code.byted.org/analyzers/go_cleaner/pkg/util"
	"github.com/sirupsen/logrus"
	"golang.org/x/tools/go/ssa"
)

type Cleaner struct {
	all    *stat
	unused *stat
	used   *stat

	comment *stat // 没有用但不删除，注释掉。只支持常量
}

func NewCleaner() *Cleaner {
	return &Cleaner{
		all:     newStat(),
		unused:  newStat(),
		used:    newStat(),
		comment: newStat(),
	}
}

func (c *Cleaner) AddFunc(f *ssa.Function) {
	if f.Syntax() == nil {
		return
	}
	c.all.addFun(f)
}

func (c *Cleaner) AddUnusedFunc(f *ssa.Function) {
	if f.Syntax() == nil {
		return
	}
	c.unused.addFun(f)
}

func (c *Cleaner) AddUsedFunc(f *ssa.Function) {
	if f.Syntax() == nil {
		return
	}
	c.used.addFun(f)
}

func (c *Cleaner) AddObject(o ast.Spec, fset *token.FileSet) {
	c.all.addObject(o, fset)
}

func (c *Cleaner) AddUnusedObject(o ast.Spec, fset *token.FileSet) {
	c.unused.addObject(o, fset)
}

func (c *Cleaner) AddUsedObject(o ast.Spec, fset *token.FileSet) {
	c.used.addObject(o, fset)
}

func (c *Cleaner) AddCommentObject(o ast.Spec, fset *token.FileSet) {
	c.comment.addObject(o, fset)
}

func (c *Cleaner) Clean(fset *token.FileSet, commentPos CommentPos, test bool, opt *Opt, deleteDirs map[string]bool) (int, int) {
	clean := c.unused.Sub(c.used) //在考虑单测场景下不sub好像会有问题，之前遇到了才加上的
	clean.addComment(fset, commentPos)
	cleaned, err := clean.clean(c.comment)
	if err != nil {
		fmt.Println("error: ", err)
		return 0, 0
	}
	// 删除没有使用的package
	logrus.Info("deleteDir (unused go package) start...")
	for dir := range deleteDirs {
		// package没有被使用，但是内部的某些成员因为最近修改过，在新修改不删除的保护策略下，需要排除
		if c.used.HasPath(dir) {
			continue
		}
		_ = deleteGoInDir(dir)
	}
	logrus.Info("deleteDir (unused go package) done!")

	// 删除空目录
	logrus.Info("RmBlackDirRecursive start...")
	_ = RmBlackDirRecursive(".")
	logrus.Info("RmBlackDirRecursive done!")

	// 删除无用import, 针对`import . "path"`这种场景
	// 删除报错的var _  I = (*X)(nil)
	newFixCleaner(test, opt.Tag, opt.BuildFlags).clean()

	total := c.all.notTestSum()
	if total == 0 {
		total = 1
	}
	logrus.Infof("Cleaned/Total: %d/%d, Rate %.2f%%\n", cleaned, total, float64(cleaned)/float64(total)*100)
	return cleaned, total
}

type statPair struct {
	start     token.Pos
	end       token.Pos
	fileName  string
	startLine int
	endLint   int
}

type stat struct {
	data map[string]map[int]struct{}
	pair map[statPair]bool
}

func newStat() *stat {
	return &stat{
		data: map[string]map[int]struct{}{},
		pair: map[statPair]bool{},
	}
}

func (t *stat) addFun(f *ssa.Function) {
	if f == nil || f.Prog == nil || f.Prog.Fset == nil || f.Syntax() == nil {
		return
	}
	from := f.Prog.Fset.Position(f.Syntax().Pos())
	to := f.Prog.Fset.Position(f.Syntax().End())
	t.addSpan(&from, &to, f.Syntax().Pos(), f.Syntax().End())
}

func (t *stat) addObject(o ast.Spec, fset *token.FileSet) {
	from := fset.Position(o.Pos())
	to := fset.Position(o.End())
	t.addSpan(&from, &to, o.Pos(), o.End())
}

func (t *stat) addComment(fset *token.FileSet, pos CommentPos) {
	for p := range t.pair {
		start := fset.Position(p.start)
		end := fset.Position(p.end)
		if s, e, ok := pos.GetComment(start, end); ok {
			t.addSpan(s, e, 0, 0)
		}
	}
}

var blankConst = []byte("const ()")
var blankVar = []byte("var ()")
var blankType = []byte("type ()")

func same(s1, s2 []byte) bool {
	if len(s1) != len(s2) {
		return false
	}
	for i := 0; i < len(s1); i++ {
		if s1[i] != s2[i] {
			return false
		}
	}
	return true
}

func (t *stat) clean(comment *stat) (cleaned int, err error) {
	logrus.Info("stat.clean start...")
	defer func() {
		logrus.Info("stat.clean done!")
	}()

	cwd, _ := filepath.Abs(".")

	paths := map[string]struct{}{}
	for p := range t.data {
		paths[p] = struct{}{}
	}
	for p := range comment.data {
		paths[p] = struct{}{}
	}
	for path := range paths {
		// 当开启测试模式，可能会生成一些由go test产生的文件
		if !strings.HasPrefix(path, cwd) {
			continue
		}

		shouldRMLines := t.data[path]
		shouldCommentLine := comment.data[path]

		raw, err := os.ReadFile(path)
		if err != nil {
			return 0, fmt.Errorf("auto clean err: %w", err)
		}
		var keptBytes []byte
		for idx, line := range bytes.Split(raw, []byte{'\n'}) {
			if shouldRMLines != nil {
				if _, ok := shouldRMLines[idx+1]; ok {
					cleaned++
					if util.IsCleanerIntegrationTestEnv() {
						logrus.Infof("remove line: %s", string(line))
					}
					continue
				}
			}
			if shouldCommentLine != nil {
				if _, ok := shouldCommentLine[idx+1]; ok {
					line = append([]byte{'/', '/', ' '}, bytes.TrimSpace(line)...)
				}
			}
			keptBytes = append(keptBytes, line...)
			keptBytes = append(keptBytes, '\n')

		}
		keptBytes, err = rmBlankGenDecl(keptBytes)
		if err != nil {
			fmt.Println(fmt.Errorf("rm blank declr err: %w", err))
			continue
		}

		keptBytes, err = imports.Process(path, keptBytes, nil)
		if err != nil {
			fmt.Println(fmt.Errorf("format err: %w", err))
			continue
		}

		var keptBytes0 = keptBytes
		keptBytes = make([]byte, 0, len(keptBytes0))
		for _, line := range bytes.Split(keptBytes0, []byte{'\n'}) {
			if same(line, blankConst) || same(line, blankVar) || same(line, blankType) {
				continue
			}
			keptBytes = append(keptBytes, line...)
			keptBytes = append(keptBytes, '\n')
		}

		// remove redundant tailing '\n'
		i := len(keptBytes) - 1
		for i-1 >= 0 && keptBytes[i-1] == '\n' {
			i--
		}
		keptBytes = keptBytes[:i+1]

		if canDelSafely(keptBytes) {
			err = os.Remove(path)
			if err != nil {
				return 0, fmt.Errorf("auto clean err: %w", err)
			}

			// _ = os.Remove(path[:len(path)-3] + "_test.go") // 如果a.go删除，也删除a_test.go
			continue
		}

		if OutputChangesToStdout {
			_, err = io.WriteString(os.Stdout, string(keptBytes))
		} else {
			err = os.WriteFile(path, keptBytes, 0644)
		}
		if err != nil {
			return 0, fmt.Errorf("auto clean err: %w", err)
		}
	}
	return cleaned, nil
}

func (t *stat) add(pos *token.Position) {
	if _, ok := t.data[pos.Filename]; !ok {
		t.data[pos.Filename] = map[int]struct{}{}
	}
	t.data[pos.Filename][pos.Line] = struct{}{}
}

func (t *stat) addSpan(from, to *token.Position, fromPos, toPos token.Pos) {
	if (from.Filename != to.Filename) || from.Line > to.Line {
		return
	}
	pos := *from
	for pos.Line <= to.Line {
		t.add(&pos)
		pos.Line++
	}
	t.pair[statPair{fromPos, toPos, from.Filename, from.Line, to.Line}] = true
}

func (t *stat) debugSpan() {
	for p := range t.pair {
		logrus.Infof("%s:%d-%d", p.fileName, p.startLine, p.endLint)
	}
}

func (t *stat) sum() int {
	counter := 0
	for _, f := range t.data {
		counter += len(f)
	}
	return counter
}

func (t *stat) Sub(t1 *stat) (o *stat) {
	o = newStat()
	for p := range t.pair {
		if t1.pair[p] {
			continue
		}
		o.pair[p] = true
		for i := p.startLine; i <= p.endLint; i++ {
			if _, ok := o.data[p.fileName]; !ok {
				o.data[p.fileName] = map[int]struct{}{}
			}
			o.data[p.fileName][i] = struct{}{}
		}
	}
	return
}

func (t *stat) Has(pos token.Position) bool {
	if lineInfo, ok := t.data[pos.Filename]; !ok {
		return false
	} else {
		if _, ok := lineInfo[pos.Line]; !ok {
			return false
		}
	}
	return true
}

func (t *stat) HasPath(dir string) bool {
	for file := range t.data {
		if strings.HasPrefix(file, dir) {
			return true
		}
	}
	return false
}

func (t *stat) notTestSum() int {
	cwd, _ := filepath.Abs(".")
	counter := 0
	for path, f := range t.data {
		// 当开启测试模式，可能会生成一些由go test产生的文件
		if !strings.HasPrefix(path, cwd) {
			continue
		}
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		if strings.Contains(path, "_gen/") {
			continue
		}
		counter += len(f)
	}
	return counter
}
