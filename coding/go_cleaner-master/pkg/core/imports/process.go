package imports

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/tools/go/packages"
	goimports "golang.org/x/tools/imports"
)

var PkgsNoSideEffets []string

func Process(path string, src []byte, opt *goimports.Options) ([]byte, error) {
	return processInitAware(path, src, opt)
}

// ProcessInitAware 行为：当 goimports 要删除某个 import 时，若该包或其依赖包含 init()，则把该 import 改成下划线导入保留副作用。
func processInitAware(filename string, src []byte, opt *goimports.Options) ([]byte, error) {
	var err error
	if src == nil {
		src, err = os.ReadFile(filename)
		if err != nil {
			return nil, err
		}
	}
	if opt == nil {
		opt = &goimports.Options{Comments: true, TabIndent: true, TabWidth: 8}
	}

	// 1) 解析原文件的 import 集合
	fset := token.NewFileSet()
	origFile, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse original: %w", err)
	}
	origImports := collectImportPaths(origFile)

	// 2) 常规 goimports 处理一次
	formatted, err := goimports.Process(filename, src, opt)
	if err != nil {
		return nil, err
	}

	// 3) 解析“处理后”的 import 集合，找出被删除的 import
	postFile, err := parser.ParseFile(fset, filename, formatted, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse formatted: %w", err)
	}
	postImports := collectImportPaths(postFile)

	var removed []string
	for p := range origImports {
		if !postImports[p] {
			removed = append(removed, p)
		}
	}
	if len(removed) == 0 {
		// 没有 import 被删，直接返回 goimports 的结果
		return formatted, nil
	}

	// 4) 对“被删除”的 import 检查是否存在 init（在包或其依赖）
	dir := safeDir(filename)
	var needKeepBlank []string
	for _, path := range removed {
		has, err := hasInitInPkgOrDeps(path, dir)
		if err != nil {
			// 为了稳妥，不因查询失败而误删副作用包：出现错误时，保守起见当作有 init 处理
			has = true
		}
		if has {
			needKeepBlank = append(needKeepBlank, path)
		}
	}
	if len(needKeepBlank) == 0 {
		return formatted, nil
	}

	// 5) 把需要保留的 import 以 `_ "path"` 形式插回 “处理后”的 AST
	newFile, fset2, err := reinsertBlankImports(filename, formatted, needKeepBlank, "")
	if err != nil {
		return nil, err
	}

	// 6) 将 AST 打印为源码
	var buf bytes.Buffer
	cfg := &printer.Config{Mode: printer.TabIndent | printer.UseSpaces, Tabwidth: opt.TabWidth}
	if err := cfg.Fprint(&buf, fset2, newFile); err != nil {
		return nil, fmt.Errorf("print reinjected: %w", err)
	}

	// 7) 再跑一次 goimports 整理（排序分组等）
	final, err := goimports.Process(filename, buf.Bytes(), opt)
	if err != nil {
		// 即便第二次 goimports 失败，也尽量返回插入 blank import 的版本
		return buf.Bytes(), nil
	}
	return final, nil
}

// ---------- 辅助函数 ----------

func collectImportPaths(f *ast.File) map[string]bool {
	m := make(map[string]bool)
	for _, imp := range f.Imports {
		if imp.Path != nil {
			p := strings.Trim(imp.Path.Value, `"`)
			if p != "" {
				m[p] = true
			}
		}
	}
	return m
}

func safeDir(filename string) string {
	if filename == "" {
		d, _ := os.Getwd()
		return d
	}
	if info, err := os.Stat(filename); err == nil && info.IsDir() {
		return filename
	}
	return filepath.Dir(filename)
}

// 只把 PkgPath 含 ".byted.org" 的包视为“字节内部包”
func isBytedPackage(p *packages.Package) bool {
	return p != nil && strings.Contains(p.PkgPath, ".byted.org")
}

func hasNoSideEffects(p *packages.Package) bool {
	if p == nil {
		return true
	}

	for _, pkg := range PkgsNoSideEffets {
		if strings.Contains(p.PkgPath, pkg) {
			return true
		}
	}

	return false
}

// 仅检查“非字节内部库”的包及其“非字节内部库”依赖里是否存在 init()。
// 若 importPath 本身是非字节内部库，直接返回 false（不保留为下划线导入）。
func hasInitInPkgOrDeps(importPath, workDir string) (bool, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedImports | packages.NeedSyntax | packages.NeedDeps,
		Dir:  workDir,
		Env:  os.Environ(),
	}
	pkgs, err := packages.Load(cfg, importPath)
	if err != nil {
		return false, fmt.Errorf("packages.Load %s: %w", importPath, err)
	}
	if len(pkgs) == 0 {
		return false, fmt.Errorf("packages.Load %s: no packages", importPath)
	}

	// 根包必须是字节内部包，否则不保留为下划线导入
	if !isBytedPackage(pkgs[0]) {
		return false, nil
	}

	seen := map[*packages.Package]bool{}
	var q []*packages.Package
	q = append(q, pkgs...)

	for len(q) > 0 {
		p := q[0]
		q = q[1:]
		if p == nil || seen[p] {
			continue
		}
		seen[p] = true

		// 依赖遍历时也跳过非字节内部库或者无副作用的包
		if !isBytedPackage(p) || hasNoSideEffects(p) {
			continue
		}

		// 查找 init 声明
		for _, f := range p.Syntax {
			for _, decl := range f.Decls {
				fd, ok := decl.(*ast.FuncDecl)
				if !ok || fd.Body == nil {
					continue
				}

				if fd.Name != nil && fd.Name.Name == "init" && len(fd.Body.List) > 0 {
					return true, nil
				}
			}
		}
		// 扩展到依赖
		for _, dep := range p.Imports {
			if dep != nil && !seen[dep] {
				q = append(q, dep)
			}
		}
	}
	return false, nil
}

// reinsertBlankImports 把 needKeepBlank 列表以 `_ "path"` 形式插回 src（src 是 goimports 一轮后的结果）
func reinsertBlankImports(
	filename string,
	src []byte,
	needKeepBlank []string,
	eolComment string,
) (*ast.File, *token.FileSet, error) {
	if eolComment == "" {
		eolComment = "BAG cleaner: keep for side effects (init function), it can be deleted if not necessary"
	}

	sort.Strings(needKeepBlank) // 稳定性
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return nil, nil, fmt.Errorf("parse for reinject: %w", err)
	}

	// 现有 import 集合，避免重复
	have := map[string]bool{}
	for _, imp := range f.Imports {
		if imp.Path != nil {
			have[strings.Trim(imp.Path.Value, `"`)] = true
		}
	}

	// 找到或创建一个 import GenDecl
	var impDecl *ast.GenDecl
	for _, d := range f.Decls {
		if gd, ok := d.(*ast.GenDecl); ok && gd.Tok == token.IMPORT {
			impDecl = gd
			break
		}
	}
	if impDecl == nil {
		impDecl = &ast.GenDecl{Tok: token.IMPORT, TokPos: f.Package}
		// 将 import 声明插入到包声明之后
		f.Decls = append([]ast.Decl{impDecl}, f.Decls...)
	}

	for _, p := range needKeepBlank {
		if have[p] {
			continue
		}
		imp := &ast.ImportSpec{
			Name: ast.NewIdent("_"),
			Path: &ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(p)},
			Comment: &ast.CommentGroup{
				List: []*ast.Comment{
					{Text: "// " + eolComment},
				},
			},
		}
		impDecl.Specs = append(impDecl.Specs, imp)
		have[p] = true
	}
	return f, fset, nil
}
