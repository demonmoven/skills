package internal

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io/ioutil"
	"os"
	"sort"

	"code.byted.org/gopkg/logs"
)

type LineRange struct {
	LineStart int
	LineEnd   int
}

const MarkRetainCommentText = "// Marker Retained to Prevent Clearing"

func InsertComments(filePath string, lineRanges []*LineRange) error {
	// 读取文件内容
	src, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %v", err)
	}

	// 创建一个新的文件集
	fset := token.NewFileSet()

	// 解析源码到AST
	fileNode, err := parser.ParseFile(fset, filePath, src, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse source code: %v", err)
	}

	for ind, decl := range fileNode.Decls {
		// fileNode的-1层的pre节点
		var preLv1 ast.Node
		if ind > 0 {
			preLv1 = fileNode.Decls[ind-1]
		}
		// 命中标注判断 & 插入注释
		if fd, ok := decl.(*ast.FuncDecl); ok {
			if checkIsMarkTarget(fset, decl, lineRanges) {
				fd.Doc, fileNode.Comments = insertComment(fset, decl, preLv1, fd.Doc, fileNode.Comments)
			}
		} else if gd, ok := decl.(*ast.GenDecl); ok {
			if !checkIsMarkTarget(fset, gd, lineRanges) {
				continue
			}
			switch gd.Tok {
			case token.VAR, token.CONST, token.TYPE:
				if len(gd.Specs) > 1 { // 分组声明
					for specsInd, sp := range gd.Specs {
						// fileNode的-2层的pre节点
						var preLv2 ast.Node
						if specsInd == 0 {
							preLv2 = preLv1
						} else {
							preLv2 = gd.Specs[specsInd-1]
						}
						if checkIsMarkTarget(fset, sp, lineRanges) {
							if valueSp, ok := sp.(*ast.ValueSpec); ok {
								valueSp.Doc, fileNode.Comments = insertComment(fset, valueSp, preLv2, valueSp.Doc, fileNode.Comments)
							} else if typeSpec, ok := sp.(*ast.TypeSpec); ok {
								typeSpec.Doc, fileNode.Comments = insertComment(fset, typeSpec, preLv2, typeSpec.Doc, fileNode.Comments)
							} else {
								logs.Error("find unexpected spec type in var/const decl")
							}
						}
					}
				} else {
					// specsInd=1表示“未分组”或“分组中只1个变量”；如果为前者，对单个spec插入注释的处理就会导致异常
					// 因此无论是否为分组声明，都对整个decl插入作用注释
					gd.Doc, fileNode.Comments = insertComment(fset, gd, preLv1, gd.Doc, fileNode.Comments)
				}
			default:
			}
		}
	}

	// 将修改后的AST格式化为源码
	var buf bytes.Buffer
	err = printer.Fprint(&buf, fset, fileNode)
	if err != nil {
		return err
	}

	err = os.WriteFile(filePath, buf.Bytes(), 0644)
	if err != nil {
		return err
	}
	return nil
}

func checkIsMarkTarget(fset *token.FileSet, node ast.Node, lineRanges []*LineRange) bool {
	for _, lineRange := range lineRanges {
		if fset.Position(node.Pos()).Line <= lineRange.LineEnd && fset.Position(node.End()).Line >= lineRange.LineStart {
			return true
		}
	}
	return false
}

func insertComment(fset *token.FileSet, cur, pre ast.Node, nodeSelfComment *ast.CommentGroup, fileComments []*ast.CommentGroup) (*ast.CommentGroup, []*ast.CommentGroup) {
	if nodeSelfComment == nil {
		nodeSelfComment = &ast.CommentGroup{List: []*ast.Comment{
			{
				Slash: cur.Pos() - 1,
				Text:  MarkRetainCommentText,
			},
		}}
		if pre != nil && fset.Position(pre.End()).Line == fset.Position(cur.Pos()-1).Line {
			nodeSelfComment = &ast.CommentGroup{List: []*ast.Comment{
				{
					Slash: cur.Pos() - 1,
					Text:  "\n" + MarkRetainCommentText,
				},
			}}
		}
		fileComments = append(fileComments, nodeSelfComment)
		sort.Slice(fileComments, func(i, j int) bool {
			return fileComments[i].Pos() < fileComments[j].Pos()
		})
	} else {
		for _, c := range nodeSelfComment.List {
			if c.Text == MarkRetainCommentText { // 已存在注释，无需重复插入
				return nodeSelfComment, fileComments
			}
		}
		nodeSelfComment.List = append(nodeSelfComment.List, &ast.Comment{
			Slash: cur.Pos() - 1,
			Text:  MarkRetainCommentText,
		})
	}
	return nodeSelfComment, fileComments
}
