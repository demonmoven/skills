package internal

import (
	"go/ast"
	"go/token"
	"strings"

	"golang.org/x/tools/go/packages"
)

// 删除定义但没有使用但局部变量
type unusedDeclCleaner struct {
	data map[*ast.File]map[token.Pos]bool
	pkg  []*packages.Package
	file map[string]*ast.File
	fSet *token.FileSet
}

func newUnusedDeclCleaner() *unusedDeclCleaner {
	u := &unusedDeclCleaner{data: map[*ast.File]map[token.Pos]bool{}, file: map[string]*ast.File{}}
	u.init()
	return u
}

func (u *unusedDeclCleaner) init() {
	pkgs, _ := LoadPkg(WithFileSet())
	u.pkg = pkgs

	for _, p := range pkgs {
		if u.fSet == nil {
			u.fSet = p.Fset
		}
		for _, f := range p.Syntax {
			path := p.Fset.Position(f.Pos()).Filename
			u.file[path] = f
		}
	}

	for _, p := range pkgs {
		for _, e := range p.TypeErrors {
			if strings.HasSuffix(e.Msg, "declared but not used") {
				pos := e.Fset.Position(e.Pos)
				file := u.file[pos.Filename]
				if file == nil {
					continue
				}
				if _, ok := u.data[file]; !ok {
					u.data[file] = map[token.Pos]bool{}
				}
				u.data[file][e.Pos] = true
			}
		}
	}
	for file, pos := range u.data {
		u.collect(file, pos)
	}
}

// collect 收集所有的需要删除局部变量的位置
//
// var a int  // pos1 is the pos of a
// a = 1      // pos2 is the pos of a
//
// pos:{pos1}, after collect, got pos: {pos1, pos2}
func (u *unusedDeclCleaner) collect(file *ast.File, pos map[token.Pos]bool) {
	obj := map[*ast.Object]bool{}
	ast.Inspect(file, func(node ast.Node) bool {
		if ident, ok := node.(*ast.Ident); ok && pos[ident.Pos()] && ident.Obj != nil {
			obj[ident.Obj] = true
		}
		return true
	})
	ast.Inspect(file, func(node ast.Node) bool {
		if ident, ok := node.(*ast.Ident); ok && ident.Obj != nil && obj[ident.Obj] {
			pos[ident.Pos()] = true
		}
		return true
	})
}

func (u *unusedDeclCleaner) clean() {
	for file, pos := range u.data {
		ast.Inspect(file, func(node ast.Node) bool {
			switch v := node.(type) {
			case *ast.FuncDecl:
				v.Body = delUnusedDecl(v.Body, pos)
			case *ast.FuncLit:
				v.Body = delUnusedDecl(v.Body, pos)
			}
			return true
		})
		save(file, u.fSet)
	}
}

func delUnusedDecl(block *ast.BlockStmt, pos map[token.Pos]bool) *ast.BlockStmt {
	var stats []ast.Stmt
	for _, i := range block.List {
		switch v := i.(type) {
		case *ast.AssignStmt:
			if vv := delDefineAssign(v, pos); vv != nil {
				stats = append(stats, vv)
			}

		case *ast.DeclStmt:
			if genDecl, ok := v.Decl.(*ast.GenDecl); ok && (genDecl.Tok == token.VAR || genDecl.Tok == token.CONST) {
				var spec []ast.Spec
				for _, i := range genDecl.Specs {
					if vv := delValueSpec(i.(*ast.ValueSpec), pos); vv != nil {
						spec = append(spec, vv)
					}
				}
				if len(spec) == 0 {
					break
				}
				genDecl.Specs = spec
			}
			stats = append(stats, v)
		case *ast.BlockStmt:
			stats = append(stats, delUnusedDecl(v, pos))
		default:
			stats = append(stats, v)
		}
	}
	block.List = stats
	return block
}

func delDefineAssign(def *ast.AssignStmt, pos map[token.Pos]bool) *ast.AssignStmt {
	var lhs []ast.Expr
	var keptLhs []ast.Expr
	var keptRhs []ast.Expr
	hasDel := false
	hasNoDel := false
	for idx, i := range def.Lhs {
		if v, ok := i.(*ast.Ident); ok && pos[v.Pos()] {
			ident := *v
			ident.Name = "_"
			lhs = append(lhs, &ident)
			hasDel = true
		} else {
			lhs = append(lhs, i)
			keptLhs = append(keptLhs, i)
			if idx < len(def.Rhs) {
				keptRhs = append(keptRhs, def.Rhs[idx])
			}
			hasNoDel = true
		}
	}
	if hasDel && !hasNoDel {
		// 全部删除
		return nil
	} else if hasDel && hasNoDel && len(def.Lhs) == len(def.Rhs) {
		// 保留部分
		def.Lhs = keptLhs
		def.Rhs = keptRhs
	} else {
		// 仅部分重命名为_
		def.Lhs = lhs
	}

	return def
}

func delValueSpec(spec *ast.ValueSpec, pos map[token.Pos]bool) *ast.ValueSpec {
	var specName []*ast.Ident
	var keptName []*ast.Ident
	var keptValue []ast.Expr
	hasDel := false
	hasNoDel := false
	for idx, i := range spec.Names {
		if pos[i.Pos()] {
			ident := *i
			ident.Name = "_"
			specName = append(specName, &ident)
			hasDel = true
		} else {
			specName = append(specName, i)
			keptName = append(keptName, i)
			if idx < len(spec.Values) {
				keptValue = append(keptValue, spec.Values[idx])
			}
			hasNoDel = true
		}
	}
	if hasDel && !hasNoDel {
		// 全部删除
		return nil
	} else if hasDel && hasNoDel && len(spec.Names) == len(spec.Values) {
		// 保留部分
		spec.Names = keptName
		spec.Values = keptValue
	} else {
		// 仅部分重命名为_
		spec.Names = specName
	}

	return spec
}
