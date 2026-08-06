package thrift

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"code.byted.org/analyzers/go_cleaner/pkg/idl_cleaner"
	"code.byted.org/lang/gg/gslice"
	"code.byted.org/wujianhui.0218/thriftgo/parser"
)

func Clean(p *idl_cleaner.Params) idl_cleaner.Result {
	c := &Cleaner{
		params:        p,
		modifiedFiles: map[string]*parser.Thrift{},
		rmSpans:       map[string][]span{},
	}
	return c.Clean()
}

type Cleaner struct {
	params        *idl_cleaner.Params
	ast           *parser.Thrift
	modifiedFiles map[string]*parser.Thrift
	rmFns         []string
	rmSpans       map[string][]span
}

type span struct {
	Start, End uint32
}

func (s span) String() string {
	return fmt.Sprintf("%d->%d", s.Start, s.End)
}

func (c *Cleaner) Clean() (result idl_cleaner.Result) {
	err := os.Chdir(c.params.Root)
	if err != nil {
		return result.SetErrf("切换工作目录失败: %s", err.Error())
	}
	p := c.params
	filePath := filepath.Join(p.Root, p.File)
	ast, err := parser.ParseFile(filePath, []string{p.Root}, true)
	if err != nil {
		return result.SetErr(err)
	}
	c.ast = ast

	for _, srv := range ast.Services {
		if err := c.removeServiceMethods(c.ast, srv, p.Methods); err != nil {
			return result.SetErr(err)
		}
	}
	result = result.SetRemoved(c.rmFns).SetMiss(gslice.Diff(p.Methods, c.rmFns))
	if len(c.rmFns) == 0 {
		return result.SetErrf("no method found in idl")
	}
	if err = c.flushChange2(); err != nil {
		return result.SetErr(err)
	}
	return result
}

func (c *Cleaner) removeServiceMethods(ast *parser.Thrift, srv *parser.Service, methods []string) error {
	fns := make([]*parser.Function, 0)
	rmFns := []string{}
	for _, fn := range srv.Functions {
		if gslice.Contains(methods, fn.Name) {
			rmFns = append(rmFns, fn.Name)
			c.rmSpans[ast.Filename] = append(c.rmSpans[ast.Filename], span{
				Start: fn.Start,
				End:   fn.End - 1, // 看起来end是开区间
			})
			fmt.Printf("find need remove method(%s) in service(%s#%s)\n", fn.Name, ast.Filename, srv.Name)
		} else if apiPaths := c.getMethodAPIPath(fn); len(apiPaths) > 0 {
			if matches := gslice.Intersect(apiPaths, methods); len(matches) > 0 {
				rmFns = append(rmFns, matches...)
				c.rmSpans[ast.Filename] = append(c.rmSpans[ast.Filename], span{
					Start: fn.Start,
					End:   fn.End - 1, // 看起来end是开区间
				})
				fmt.Printf("find need remove method(%s) with matched api path(%s) in service(%s#%s)\n",
					fn.Name, strings.Join(matches, ","), ast.Filename, srv.Name)
			}
		} else {
			fns = append(fns, fn)
		}
	}
	if len(srv.Functions) != len(fns) {
		fmt.Printf("service(%s#%s) fn count %d -> %d\n", ast.Filename, srv.Name, len(srv.Functions), len(fns))
		srv.Functions = fns
	}
	if len(rmFns) > 0 {
		c.modifiedFiles[ast.Filename] = ast
		c.rmFns = append(c.rmFns, rmFns...)
	}
	if diff := gslice.Diff(methods, rmFns); len(diff) > 0 {
		if srv.Extends != "" {
			extAst, extSrv, err := c.getIncludeExtendsAst(ast, srv.Extends)
			if err != nil {
				return err
			}
			return c.removeServiceMethods(extAst, extSrv, diff)
		}
	}
	return nil
}
func (c *Cleaner) getMethodAPIPath(fn *parser.Function) []string {
	if fn.Annotations == nil {
		return nil
	}
	// 参照【ByteAPI Thrift IDL 定义规范】 https://bytedance.larkoffice.com/wiki/wikcnBevBcZqVc0bbuFr0JLr92d ，【Method 规范】小节
	// 2. 每个 URI 对应一个 Method，通过注解关联，注解不可为空。一个 Method 支持对应多个 URI，通过 , 隔开，例如 api.get='/path1,/path2,/path3'。
	for _, ann := range fn.Annotations {
		if ann.Key == "api.get" || ann.Key == "api.post" || ann.Key == "api.put" || ann.Key == "api.delete" || ann.Key == "api.patch" {
			// 目前看来不支持同时有多个api注解，这里先按第一个生效处理，后面研究下实际应该是最后一个还是第一个
			return ann.Values
		}
	}
	return nil
}
func (c *Cleaner) getIncludeExtendsAst(ast *parser.Thrift, extendService string) (*parser.Thrift, *parser.Service, error) {
	if strings.Contains(extendService, ".") {
		segs := strings.Split(extendService, ".")
		for _, include := range ast.Includes {
			_, includeName := filepath.Split(include.Path)
			if strings.TrimSuffix(includeName, ".thrift") == segs[0] {
				for _, srv := range include.Reference.Services {
					if srv.Name == segs[1] {
						return include.Reference, srv, nil
					}
				}
				return nil, nil, fmt.Errorf("service(%s) not found in file(%s)", segs[1], include.Path)
			}
		}
		return nil, nil, fmt.Errorf("file(%s) extend(%s) include not found", ast.Filename, extendService)
	} else {
		for _, srv := range ast.Services {
			if srv.Name == extendService {
				return ast, srv, nil
			}
		}
		return nil, nil, fmt.Errorf("service(%s) not found in file(%s)", extendService, ast.Filename)
	}
}

func (c *Cleaner) flushChange2() error {
	if len(c.modifiedFiles) == 0 {
		fmt.Println("no modified file")
		return nil
	}
	fmt.Printf("modified files(%d):\n", len(c.modifiedFiles))
	entryFileDir := filepath.Dir(filepath.Join(c.params.Root, c.params.File))
	fmt.Println("entry file dir:", entryFileDir)
	for path, ast := range c.modifiedFiles {
		fmt.Println("\t", path)
		writePath, err := filepath.Abs(path)
		if err != nil {
			fmt.Printf("get abs path(%s) failed:%v\n", path, err)
			return fmt.Errorf("get abs path(%s) failed: %w", path, err)
		}
		fmt.Printf("\t\twrite-path: %s\n", writePath)
		fmt.Printf("\t\trm spans: %v\n", c.rmSpans[path])
		newCnt, err := c.removeSpans(string(ast.Content), c.rmSpans[path])
		if err != nil {
			fmt.Printf("remove spans(%s) failed:%v\n", path, err)
			return fmt.Errorf("remove spans(%s) failed: %w", path, err)
		}
		if c.params.Write != nil {
			if err := c.params.Write(writePath, []byte(newCnt)); err != nil {
				fmt.Printf("write idl(%s) with custom writer failed:%v\n", path, err)
				return fmt.Errorf("write idl(%s) with custom writer failed: %w", path, err)
			}
			fmt.Printf("write to %s with custom writer success\n", path)
		} else {
			if err := os.WriteFile(writePath, []byte(newCnt), 0777); err != nil {
				fmt.Printf("write idl(%s) failed:%v\n", path, err)
				return fmt.Errorf("write idl(%s) failed: %w", path, err)
			}
			fmt.Printf("write to %s success\n", path)
		}
	}
	return nil
}

func (c *Cleaner) removeSpans(cnt string, spans []span) (string, error) {
	gslice.SortBy(spans, func(s1, s2 span) bool { return s1.Start < s2.Start })
	startIndex := uint32(0)
	oldCnt := []rune(cnt)
	newCnt := make([]rune, 0, len(oldCnt))
	for _, rmSpan := range spans {
		if rmSpan.Start < startIndex {
			return "", fmt.Errorf("invalid span: %d -> %d, start index: %d", rmSpan.Start, rmSpan.End, startIndex)
		}
		fmt.Printf("span[%d->%d]%s\n", rmSpan.Start, rmSpan.End, string(oldCnt[rmSpan.Start:rmSpan.End+1]))
		if oldCnt[startIndex] == '\n' && oldCnt[rmSpan.Start] == '\n' {
			newCnt = append(newCnt, '\n')
		}
		newCnt = append(newCnt, oldCnt[startIndex:rmSpan.Start]...)
		startIndex = rmSpan.End + 1
	}
	newCnt = append(newCnt, oldCnt[startIndex:]...)
	return string(newCnt), nil
}
