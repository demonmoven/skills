package cerr

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/sirupsen/logrus"
)

const (
	unknownErrExitCode              = 126 // 内部默认错误码，建议业务按实际错误语义定义错误码，为此该错误码定义为私有变量
	AppRootNotSupportExitCode       = 127
	ToolOutOfMemoryExitCode         = 137
	GoModFileParsingErrExitCode int = iota
	GoModTidyErrExitCode
	GoModDirNotFoundExitCode
	ToolPrecheckUnknownRepoStructureExitCode
	GoParserSyntaxErrExitCode
	ToolPreloadInternalErrExitCode
	ToolPreloadPackageContainsErrExitCode
	ToolPreloadIgnoreTestErrExitCode
	ToolPreloadIgnoredByGoListExitCode
	ToolAnalysisInternalErrExitCode
	ToolAnalysisPackageContainsErrExitCode
)

var (
	// 1. 可能发生在预处理过程中的错误：
	// 1.1、我们无法正确处理go.mod文件，请检查你的go.mod文件是否正确
	// 1.2、我们无法完成go mod tidy任务，请检查你的go.mod文件是否正确，以及依赖是否可访问
	// 1.3、我们无法找到go.mod文件，这个仓库可能不包含go.mod, 请联系我们的开发人员
	// 1.4、我们无法从Go文件中解析出语法树，请联系我们的开发人员
	// 1.5、不支持在该仓库查找指定应用入口（main.go），该仓库可能使用了非kitex框架，请联系我们的开发人员
	GoModFileParsingErr  = New(GoModFileParsingErrExitCode, "Unable to parse the go.mod file. Please check whether your go.mod file is correct.")
	GoModTidyErr         = New(GoModTidyErrExitCode, "We are unable to complete the go mod tidy task. Please check whether your go.mod file is correct and whether the dependencies are accessible.")
	GoModDirNotFoundErr  = New(GoModDirNotFoundExitCode, "Can't find the go.mod file. This repository may not contain a go.mod.")
	GoParserSyntaxErr    = New(GoParserSyntaxErrExitCode, "We are unable to parse the syntax tree from Go files. Please contact on-call developers.")
	AppRootNotSupportErr = New(AppRootNotSupportExitCode, "only kitex has supported specify app root(the directory of main.go). Please contact on-call developers.")

	// 2. 可能发生在预加载过程中的错误：
	// 2.1、我们在分析之前尝试加载仓库，但是加载机制出现了内部错误，这可能是由于测试文件存在编译错误导致的，请确认你的测试文件可以编译成功，或者直接使用--test_mode=none选项
	// 2.2、我们在分析之前尝试加载仓库，但是发现了编译错误，请修复它们
	// 2.3、我们在分析之前尝试加载仓库，并且发现了测试中的编译错误，在试图忽略它们的时候发生了内部错误，请确认你的测试文件可以编译成功，或者直接使用--test_mode=none选项
	// 2.4、我们在分析之前尝试加载仓库，但是发现仓库存在可能被go list忽略的源文件，它们的名称或者所在的目录可能是以下划线'_'或者'.'为前缀的，请尝试去除这些前缀
	ToolPreloadInternalErr        = New(ToolPreloadInternalErrExitCode, "We attempted to load the repository before the analysis, but an internal error occurred during the loading. This might be caused by test symbols. Please confirm that the tests can be compiled successfully, or add the `--test_mod=none` option.")
	ToolPreloadPackageContainsErr = New(ToolPreloadPackageContainsErrExitCode, "We attempted to load the repository before the analysis, but compilation errors were found. Please fix them.")
	ToolPreloadIgnoreTestErr      = New(ToolPreloadIgnoreTestErrExitCode, "We attempted to load the repository before the analysis, and there were compilation errors in the tests. An internal error occurred when trying to ignore them. Please confirm that the tests can be compiled successfully, or add the `--test_mod=none` option.")
	ToolPreloadIgnoredByGoListErr = New(ToolPreloadIgnoredByGoListExitCode, "We attempted to load the repository before the analysis, but some source files were ignored by `go list`. Their names or directories may start with underscores '_' or '.'. Please try to remove these prefixes.")

	// 3. 可能发生在分析过程中的错误：
	// 3.1、我们在分析过程中发现了内部错误，请联系我们的开发人员
	// 3.2、我们在分析过程中发现了编译错误，请联系我们的开发人员
	ToolAnalysisInternalErr        = New(ToolAnalysisInternalErrExitCode, "We encountered an internal error during the analysis. Please contact on-call developers.")
	ToolAnalysisPackageContainsErr = New(ToolAnalysisPackageContainsErrExitCode, "We encountered compilation errors during the analysis. Please contact on-call developers.")

	// 4. 可能发生在执行过程中的错误：
	// 4.1、我们在执行过程中发现了内存不足的错误，请尝试减少分析的范围或者增加机器的内存
	ToolOutOfMemoryErr = New(ToolOutOfMemoryExitCode, "We encountered an out-of-memory error during execution. Please try the task later or contact on-call developers.")
)

type Error interface {
	error
	ExitCode() int
}

type codeErr struct {
	exitCode int
	message  string
}

func (e *codeErr) ExitCode() int {
	return e.exitCode
}
func (e *codeErr) Error() string {
	return fmt.Sprintf("[error %d] %s", e.exitCode, e.message)
}
func New(code int, msg string) *codeErr {
	return &codeErr{
		exitCode: code,
		message:  msg,
	}
}
func Newf(code int, format string, a ...interface{}) *codeErr {
	return &codeErr{
		exitCode: code,
		message:  fmt.Sprintf(format, a...),
	}
}

func Exit(err error) {
	if err == nil {
		return
	}

	logrus.SetOutput(os.Stderr)
	logrus.Infof(err.Error())

	if ce, ok := err.(Error); ok {
		fmt.Println(string(debug.Stack()))
		os.Exit(ce.ExitCode())
	} else {
		fmt.Println(string(debug.Stack()))
		os.Exit(unknownErrExitCode)
	}
}

func ExitWithDetail(detail error, err error) {
	if err == nil {
		return
	}

	if detail != nil {
		fmt.Fprintf(os.Stderr, "%s\n", detail.Error())
	}

	Exit(err)
}

func NewErrorWithDetail(detail error, err error) Error {
	if err == nil {
		return New(unknownErrExitCode, "unknown error")
	}

	e, ok := err.(*codeErr)
	if !ok {
		return New(unknownErrExitCode, err.Error())
	}

	if detail != nil {
		e.message = fmt.Sprintf("%s\n%s", detail.Error(), e.Error())
	}

	return e
}
