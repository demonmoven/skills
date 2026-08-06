package proto

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"code.byted.org/analyzers/go_cleaner/pkg/idl_cleaner"
	"code.byted.org/analyzers/go_cleaner/pkg/idl_cleaner/proto/plugin/byteapi"
	"code.byted.org/lang/gg/gslice"
	"github.com/bufbuild/protocompile"
	"github.com/bufbuild/protocompile/linker"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/runtime/protoimpl"
)

var (
	commentPrefix = []byte("/*")
	commentSuffix = []byte("*/")
)

type Cleaner struct {
	params        *idl_cleaner.Params
	file          linker.File
	locPathToName map[string]string
	methodLocs    map[MethodSourcePath]protoreflect.SourceLocation
	removeLocs    []protoreflect.SourceLocation
	result        idl_cleaner.Result
}

type MethodSourcePath struct {
	Service int32
	Method  int32
}

func Clean(p *idl_cleaner.Params) idl_cleaner.Result {
	c := &Cleaner{
		params:        p,
		locPathToName: map[string]string{},
		methodLocs:    map[MethodSourcePath]protoreflect.SourceLocation{},
		removeLocs:    make([]protoreflect.SourceLocation, 0),
	}
	return c.Clean()
}

func (c *Cleaner) Clean() idl_cleaner.Result {
	p := c.params
	if err := c.getFile(p); err != nil {
		return c.result.SetErr(err)
	} else if err = c.arrangeMethodLocs(); err != nil { // 解析方法，用于后续获取方法的位置信息
		return c.result.SetErr(err)
	} else if err = c.findRevLocations(); err != nil { // 找到需要删除的方法信息
		return c.result.SetErr(err)
	}
	if len(c.removeLocs) == 0 {
		return c.result
	}
	if c.params.Remove {
		return c.result.SetErr(c.applyRemove(p.Root, c.file, c.removeLocs))
	}
	return c.result.SetErr(c.applyComment(p.Root, c.file, c.removeLocs))
}

func (c *Cleaner) arrangeMethodLocs() error {
	methodLocs := map[MethodSourcePath]protoreflect.SourceLocation{}
	for i := 0; i < c.file.SourceLocations().Len(); i++ {
		loc := c.file.SourceLocations().Get(i)
		sourcePath := loc.Path.String()
		if len(sourcePath) < 8 {
			continue
		}
		segs := strings.Split(sourcePath[1:], ".")
		if len(segs) != 2 {
			continue
		} else if !strings.HasPrefix(segs[0], "service[") {
			continue
		} else if !strings.HasPrefix(segs[1], "method[") {
			continue
		}
		path := MethodSourcePath{}
		if _, err := fmt.Sscanf(segs[0][8:len(segs[0])-1], "%d", &path.Service); err != nil {
			return fmt.Errorf("invalid service.method source path: %s", loc.Path.String())
		} else if _, err = fmt.Sscanf(segs[1][7:len(segs[1])-1], "%d", &path.Method); err != nil {
			return fmt.Errorf("invalid service.method source path: %s", loc.Path.String())
		}
		methodLocs[path] = loc
	}
	c.methodLocs = methodLocs
	return nil
}

func (c *Cleaner) findRevLocations() error {
	p := c.params
	removeLocs := make([]protoreflect.SourceLocation, 0)
	findRevs := make([]string, 0)
	for srvIdx := 0; srvIdx < c.file.Services().Len(); srvIdx++ {
		srv := c.file.Services().Get(srvIdx)
		for i := 0; i < srv.Methods().Len(); i++ {
			mtd := srv.Methods().Get(i)
			paths := c.getMethodAPIPath(mtd)
			hitMethods := gslice.Intersect(paths, p.Methods)
			if len(hitMethods) == 0 {
				continue // 不是需要删除的方法
			}
			path := MethodSourcePath{Service: int32(srv.Index()), Method: int32(mtd.Index())}
			loc, ok := c.methodLocs[path]
			if !ok { // 内部异常，未能正确解析方法
				return fmt.Errorf("内部解析异常，未能定位方法(%v)的文件位置", mtd.FullName())
			}
			c.locPathToName[loc.Path.String()] = string(mtd.FullName())
			removeLocs = append(removeLocs, loc)
			findRevs = append(findRevs, hitMethods...)
		}
	}
	c.result = c.result.SetRemoved(findRevs).SetMiss(gslice.Diff(p.Methods, findRevs))
	c.removeLocs = removeLocs
	return nil
}

func (c *Cleaner) getFile(p *idl_cleaner.Params) error {
	cpl := protocompile.Compiler{
		Resolver: &protocompile.SourceResolver{
			ImportPaths: []string{p.Root},
		},
		SourceInfoMode: protocompile.SourceInfoStandard,
	}
	files, err := cpl.Compile(context.Background(), p.File)
	if err != nil {
		return err
	}
	c.file = files.FindFileByPath(p.File)
	if c.file == nil {
		return fmt.Errorf("file not found: %s", p.File)
	}
	return nil
}
func (c *Cleaner) getMethodAPIPath(mtd protoreflect.MethodDescriptor) []string {
	v := proto.GetExtension(mtd.Options(), byteapi.E_Any)
	if v != nil && v != "" {
		return []string{v.(string)}
	}
	paths := make([]string, 0)
	exists := map[string]bool{}
	for _, c := range []*protoimpl.ExtensionInfo{
		byteapi.E_Get,
		byteapi.E_Post,
	} {
		pathObj := proto.GetExtension(mtd.Options(), c)
		if pathObj == nil || pathObj == "" {
			continue
		}
		path := pathObj.(string)
		if exists[path] {
			continue
		}
		exists[path] = true
		paths = append(paths, path)
	}
	return paths
}
func (c *Cleaner) applyComment(root string, file linker.File, locs []protoreflect.SourceLocation) error {
	cnt, err := os.ReadFile(filepath.Join(root, file.Path()))
	if err != nil {
		return fmt.Errorf("read file failed: %w", err)
	}
	gslice.SortBy(locs, func(loc1, loc2 protoreflect.SourceLocation) bool { return loc1.StartLine < loc2.StartLine })
	buf := []byte{}
	lines := strings.Split(string(cnt), "\n")
	needHandle := locs
	for i := 0; i < len(lines); i++ {
		span := needHandle[0]
		if i != span.StartLine {
			buf = append(buf, []byte(lines[i])...)
			buf = append(buf, []byte("\n")...)
			continue
		}
		if span.StartColumn > 0 {
			buf = append(buf, []byte(lines[i][:span.StartColumn])...)
		}
		buf = append(buf, commentPrefix...)
		if span.EndLine == span.StartLine {
			buf = append(buf, []byte(lines[i][span.StartColumn:span.EndColumn])...)
			buf = append(buf, commentSuffix...)
			buf = append(buf, []byte(lines[i][span.EndColumn:])...)
			buf = append(buf, []byte("\n")...)
		} else if span.EndLine < span.StartLine+1 {
			return fmt.Errorf("invalid method(%s) location: %d#L%d->%d#L%d", c.locPathToName[span.Path.String()],
				span.StartLine, span.StartColumn, span.EndLine, span.EndColumn)
		} else {
			buf = append(buf, []byte(lines[i][span.StartColumn:])...)
			buf = append(buf, []byte("\n")...)
			for j := i + 1; j < span.EndLine; j++ {
				buf = append(buf, []byte(lines[j])...)
				buf = append(buf, []byte("\n")...)
			}
			buf = append(buf, []byte(lines[span.EndLine][:span.EndColumn])...)
			buf = append(buf, commentSuffix...)
			buf = append(buf, []byte(lines[span.EndLine][span.EndColumn:])...)
			buf = append(buf, []byte("\n")...)
			i = span.EndLine // skip to end line
		}
		needHandle = needHandle[1:]
		if len(needHandle) == 0 {
			for j := i + 1; j < len(lines); j++ {
				buf = append(buf, []byte(lines[j])...)
				buf = append(buf, []byte("\n")...)
			}
			break
		}
	}
	buf = buf[:len(buf)-1] // 最后会多一次换行符，这里移除掉
	return c.writeOut(root, c.params.File, c.removeManySpaceLines(buf))
}
func (c *Cleaner) applyRemove(root string, file linker.File, locs []protoreflect.SourceLocation) error {
	cnt, err := os.ReadFile(filepath.Join(root, file.Path()))
	if err != nil {
		return fmt.Errorf("IDL文件读取失败: %w", err)
	}
	gslice.SortBy(locs, func(loc1, loc2 protoreflect.SourceLocation) bool { return loc1.StartLine < loc2.StartLine })
	buf := []byte{}
	lines := strings.Split(string(cnt), "\n")
	needHandle := locs
	for i := 0; i < len(lines); i++ {
		span := needHandle[0]
		if i != span.StartLine {
			buf = append(buf, []byte(lines[i])...)
			buf = append(buf, []byte("\n")...)
			continue
		}
		if span.StartColumn > 0 {
			buf = append(buf, []byte(lines[i][:span.StartColumn])...)
		}
		if span.EndLine == span.StartLine {
			buf = append(buf, []byte(lines[i][span.EndColumn:])...)
			buf = append(buf, []byte("\n")...)
		} else if span.EndLine < span.StartLine+1 {
			return fmt.Errorf("invalid method(%s) location: %d#L%d->%d#L%d", c.locPathToName[span.Path.String()],
				span.StartLine, span.StartColumn, span.EndLine, span.EndColumn)
		} else {
			buf = append(buf, []byte("\n")...)
			buf = append(buf, []byte(lines[span.EndLine][span.EndColumn:])...)
			buf = append(buf, []byte("\n")...)
			i = span.EndLine // skip to end line
		}
		needHandle = needHandle[1:]
		if len(needHandle) == 0 {
			for j := i + 1; j < len(lines); j++ {
				buf = append(buf, []byte(lines[j])...)
				buf = append(buf, []byte("\n")...)
			}
			break
		}
	}
	buf = buf[:len(buf)-1] // 最后会多一次换行符，这里移除掉
	return c.writeOut(root, c.params.File, c.removeManySpaceLines(buf))
}

func (c *Cleaner) removeManySpaceLines(buf []byte) []byte {
	lines := strings.Split(string(buf), "\n")
	filtered := make([]string, 0)
	prevSpaceLine := false
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			if prevSpaceLine {
				continue
			} else {
				prevSpaceLine = true
				filtered = append(filtered, line)
			}
		} else {
			prevSpaceLine = false
			filtered = append(filtered, line)
		}
	}
	return []byte(strings.Join(filtered, "\n"))
}

func (c *Cleaner) writeOut(root string, file string, buf []byte) error {
	if c.params.Write != nil {
		return c.params.Write(filepath.Join(root, file), buf)
	}
	err := os.WriteFile(filepath.Join(root, file), buf, 0777)
	if err != nil {
		return fmt.Errorf("IDL写出失败: %w", err)
	}
	return nil
}
