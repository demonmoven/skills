package platformtools

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// ==================== code_search 搜索能力 ====================
//
// L1 层代码搜索：在工作区内做符号级代码搜索，输出结构化结果。
// 纯机制层能力——只提供"找得到什么"，不提供"怎么用"。
// 比 shell grep 多了语义过滤（只搜定义 / 排除测试等）和结构化输出。
// 比 LSP 轻量得多，不需要构建索引。
//
// v0.3.4 起原 platform tool 分发层已移除；这里只保留搜索能力本身，
// 供 CLI `gloop code search` 直接调用。
//
// 底层实现：优先 ripgrep（快），自动回退到纯 Go 文件扫描（兼容性好）。

type CodeSearchResult struct {
	File       string `json:"file"`        // 文件路径（相对工作区）
	Line       int    `json:"line"`        // 行号（1-based）
	SymbolType string `json:"symbol_type"` // function / type / interface / const / var / import / other
	SymbolName string `json:"symbol_name"` // 符号名（如函数名、类型名），识别不出来则为空
	Signature  string `json:"signature"`   // 签名/定义行，识别不出来则为该行文本
	Snippet    string `json:"snippet"`     // 该行原文
}

const defaultCodeSearchLimit = 50

// RunCodeSearch 执行代码搜索（对外导出，供 CLI 调用）。
// 优先使用 ripgrep（快），不可用时自动回退到纯 Go 文件扫描（兼容）。
func RunCodeSearch(workDir, query, mode, language string, includeTests bool, limit int) ([]CodeSearchResult, string, error) {
	return runCodeSearch(workDir, query, mode, language, includeTests, limit)
}

// runCodeSearch 执行实际的代码搜索。
// 优先使用 ripgrep（快），不可用时自动回退到纯 Go 文件扫描（兼容）。
func runCodeSearch(workDir, query, mode, language string, includeTests bool, limit int) ([]CodeSearchResult, string, error) {
	rgPath, err := exec.LookPath("rg")
	if err == nil && rgPath != "" {
		results, err := runCodeSearchWithRg(rgPath, workDir, query, mode, language, includeTests, limit)
		return results, "ripgrep", err
	}
	// 回退到纯 Go 实现
	results, err := runCodeSearchPureGo(workDir, query, mode, language, includeTests, limit)
	return results, "pure_go", err
}

// runCodeSearchWithRg 使用 ripgrep 执行搜索（快速路径）。
func runCodeSearchWithRg(rgPath, workDir, query, mode, language string, includeTests bool, limit int) ([]CodeSearchResult, error) {
	// 构建 rg 参数
	rgArgs := []string{
		"--line-number",
		"--no-heading",
		"--color=never",
		"--sort", "path",
		"--max-count", fmt.Sprintf("%d", limit*5), // 多搜一些，过滤后再截断
	}

	// 语言过滤
	if language != "" {
		if langType := rgLanguageType(language); langType != "" {
			rgArgs = append(rgArgs, "--type", langType)
		}
	}

	// 排除测试文件
	if !includeTests {
		rgArgs = append(rgArgs,
			"--glob", "!*_test.go",
			"--glob", "!*.test.*",
			"--glob", "!*.spec.*",
			"--glob", "!**/__tests__/**",
			"--glob", "!**/test/**",
			"--glob", "!**/tests/**",
		)
	}

	// 排除常见非源码目录
	rgArgs = append(rgArgs,
		"--glob", "!.git/**",
		"--glob", "!node_modules/**",
		"--glob", "!vendor/**",
		"--glob", "!dist/**",
		"--glob", "!build/**",
		"--glob", "!.gloop/**",
	)

	// 搜索模式：definition 模式下用更精准的正则
	searchPattern := query
	if mode == "definition" {
		if language == "go" {
			searchPattern = goDefinitionPattern(query)
			rgArgs = append(rgArgs, "--regexp")
		}
		// 其他语言暂时回退到普通搜索，后处理识别符号类型
	}

	rgArgs = append(rgArgs, searchPattern, ".")

	// 执行 rg
	cmd := exec.Command(rgPath, rgArgs...)
	cmd.Dir = workDir
	output, err := cmd.Output()
	if err != nil {
		// rg 没找到匹配会返回 exit code 1，不算错误
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return []CodeSearchResult{}, nil
		}
		return nil, err
	}

	// 解析输出
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	results := make([]CodeSearchResult, 0, len(lines))

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		result := parseRgLine(line, language)
		if result == nil {
			continue
		}

		// reference 模式：排除定义行
		if mode == "reference" && result.SymbolType != "other" && result.SymbolType != "import" {
			continue
		}

		results = append(results, *result)

		if len(results) >= limit {
			break
		}
	}

	// 按符号类型排序（定义排在前面）
	sort.SliceStable(results, func(i, j int) bool {
		return symbolTypePriority(results[i].SymbolType) < symbolTypePriority(results[j].SymbolType)
	})

	return results, nil
}

// runCodeSearchPureGo 纯 Go 文件扫描实现（无 rg 时的兼容性回退）。
// 功能等价但速度较慢，适合没有 ripgrep 的环境。
func runCodeSearchPureGo(workDir, query, mode, language string, includeTests bool, limit int) ([]CodeSearchResult, error) {
	results := make([]CodeSearchResult, 0, limit)

	// 编译搜索模式
	var searchRe *regexp.Regexp
	var err error
	if mode == "definition" && language == "go" {
		searchRe, err = regexp.Compile(goDefinitionPattern(query))
	} else {
		searchRe, err = regexp.Compile(regexp.QuoteMeta(query))
	}
	if err != nil {
		return nil, fmt.Errorf("无效的搜索模式: %v", err)
	}

	// 语言扩展名过滤
	extFilter := languageExtensions(language)

	// 遍历文件
	err = filepath.WalkDir(workDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // 跳过读不了的目录/文件
		}
		if len(results) >= limit*5 {
			return filepath.SkipAll // 够了，提前终止
		}

		// 跳过非源码目录
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "node_modules" || name == "vendor" ||
				name == "dist" || name == "build" || name == ".gloop" {
				return filepath.SkipDir
			}
			if !includeTests && (name == "__tests__" || name == "test" || name == "tests") {
				return filepath.SkipDir
			}
			return nil
		}

		// 跳过测试文件
		if !includeTests && isTestFile(d.Name()) {
			return nil
		}

		// 扩展名过滤
		if !extMatches(d.Name(), extFilter) {
			return nil
		}

		// 读取文件（只搜文本文件，过大的跳过）
		info, err := d.Info()
		if err != nil || info.Size() > 2*1024*1024 { // 2MB 以上跳过
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		// 逐行匹配
		relPath, _ := filepath.Rel(workDir, path)
		relPath = filepath.ToSlash(relPath)
		lines := strings.Split(string(content), "\n")
		for lineNum, line := range lines {
			if searchRe.MatchString(line) {
				result := &CodeSearchResult{
					File:    relPath,
					Line:    lineNum + 1,
					Snippet: strings.TrimSpace(line),
				}
				// 识别符号类型
				symType, symName, signature := identifySymbol(line, language)
				result.SymbolType = symType
				result.SymbolName = symName
				result.Signature = signature

				// reference 模式：排除定义行
				if mode == "reference" && result.SymbolType != "other" && result.SymbolType != "import" {
					continue
				}

				results = append(results, *result)
				if len(results) >= limit*5 {
					break
				}
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// 截断并排序
	if len(results) > limit {
		results = results[:limit]
	}
	sort.SliceStable(results, func(i, j int) bool {
		return symbolTypePriority(results[i].SymbolType) < symbolTypePriority(results[j].SymbolType)
	})

	return results, nil
}

// languageExtensions 返回指定语言的常见扩展名列表，空列表表示不过滤。
func languageExtensions(language string) []string {
	switch language {
	case "go":
		return []string{".go"}
	case "ts":
		return []string{".ts", ".tsx"}
	case "js":
		return []string{".js", ".jsx", ".mjs", ".cjs"}
	case "py":
		return []string{".py"}
	case "java":
		return []string{".java"}
	case "rs":
		return []string{".rs"}
	case "c":
		return []string{".c", ".h"}
	case "cpp":
		return []string{".cpp", ".cc", ".cxx", ".hpp", ".hh", ".h"}
	case "sh":
		return []string{".sh", ".bash", ".zsh"}
	case "md":
		return []string{".md", ".mdx", ".markdown"}
	default:
		return nil // 不过滤
	}
}

// extMatches 检查文件名是否匹配扩展名过滤列表（空列表一律通过）。
func extMatches(filename string, exts []string) bool {
	if len(exts) == 0 {
		return true
	}
	lower := strings.ToLower(filename)
	for _, ext := range exts {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}

// isTestFile 判断文件是否是测试文件（粗略匹配）。
func isTestFile(name string) bool {
	return strings.HasSuffix(name, "_test.go") ||
		strings.Contains(name, ".test.") ||
		strings.Contains(name, ".spec.")
}

// parseRgLine 解析 rg 的 "file:line:text" 输出行，并尝试识别符号类型。
func parseRgLine(line, language string) *CodeSearchResult {
	// 格式：path/to/file.go:42:	func Foo() { ... }
	parts := strings.SplitN(line, ":", 3)
	if len(parts) < 3 {
		return nil
	}

	file := parts[0]
	lineNum := 0
	fmt.Sscanf(parts[1], "%d", &lineNum)
	if lineNum == 0 {
		return nil
	}
	snippet := parts[2]

	result := &CodeSearchResult{
		File:    filepath.ToSlash(file),
		Line:    lineNum,
		Snippet: strings.TrimSpace(snippet),
	}

	// 尝试识别符号类型
	symbolType, symbolName, signature := identifySymbol(snippet, language)
	result.SymbolType = symbolType
	result.SymbolName = symbolName
	result.Signature = signature

	return result
}

// identifySymbol 尝试从一行代码中识别符号类型和名称。
// 返回 (类型, 名称, 签名)。识别不出来时返回 "other", "", 原行文本。
func identifySymbol(line, language string) (string, string, string) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return "other", "", line
	}

	switch language {
	case "go":
		return identifyGoSymbol(trimmed)
	case "ts", "js":
		return identifyJsSymbol(trimmed)
	case "py":
		return identifyPySymbol(trimmed)
	default:
		// 通用启发式：根据开头关键字粗略判断
		if strings.HasPrefix(trimmed, "func ") || strings.HasPrefix(trimmed, "function ") || strings.HasPrefix(trimmed, "def ") {
			return "function", extractFirstWordAfter(trimmed, 1), trimmed
		}
		if strings.HasPrefix(trimmed, "type ") || strings.HasPrefix(trimmed, "interface ") || strings.HasPrefix(trimmed, "class ") {
			return "type", extractFirstWordAfter(trimmed, 1), trimmed
		}
		if strings.HasPrefix(trimmed, "const ") || strings.HasPrefix(trimmed, "let ") || strings.HasPrefix(trimmed, "var ") {
			return "var", extractFirstWordAfter(trimmed, 1), trimmed
		}
		return "other", "", trimmed
	}
}

// Go 符号识别
var goFuncPattern = regexp.MustCompile(`^func\s+(\([^)]*\)\s+)?([A-Za-z_][A-Za-z0-9_]*)\s*\(`)
var goTypePattern = regexp.MustCompile(`^type\s+([A-Za-z_][A-Za-z0-9_]*)\s+(type|struct|interface|func|map|\[)`)
var goConstPattern = regexp.MustCompile(`^const\s+([A-Za-z_][A-Za-z0-9_]*)`)
var goVarPattern = regexp.MustCompile(`^var\s+([A-Za-z_][A-Za-z0-9_]*)`)

func identifyGoSymbol(line string) (string, string, string) {
	if m := goFuncPattern.FindStringSubmatch(line); m != nil {
		return "function", m[2], line
	}
	if m := goTypePattern.FindStringSubmatch(line); m != nil {
		typeKind := "type"
		if strings.Contains(line, "interface") {
			typeKind = "interface"
		} else if strings.Contains(line, "struct") {
			typeKind = "type"
		}
		return typeKind, m[1], line
	}
	if m := goConstPattern.FindStringSubmatch(line); m != nil {
		return "const", m[1], line
	}
	if m := goVarPattern.FindStringSubmatch(line); m != nil {
		return "var", m[1], line
	}
	if strings.HasPrefix(line, "import ") {
		return "import", "", line
	}
	return "other", "", line
}

// JS/TS 符号识别
var jsFuncPattern = regexp.MustCompile(`^(export\s+)?(async\s+)?function\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*\(`)
var jsClassPattern = regexp.MustCompile(`^(export\s+)?class\s+([A-Za-z_$][A-Za-z0-9_$]*)`)
var jsConstPattern = regexp.MustCompile(`^(export\s+)?(const|let|var)\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*[=:]`)

func identifyJsSymbol(line string) (string, string, string) {
	if m := jsFuncPattern.FindStringSubmatch(line); m != nil {
		return "function", m[3], line
	}
	if m := jsClassPattern.FindStringSubmatch(line); m != nil {
		return "type", m[2], line
	}
	if m := jsConstPattern.FindStringSubmatch(line); m != nil {
		return "var", m[3], line
	}
	return "other", "", line
}

// Python 符号识别
var pyFuncPattern = regexp.MustCompile(`^def\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(`)
var pyClassPattern = regexp.MustCompile(`^class\s+([A-Za-z_][A-Za-z0-9_]*)`)

func identifyPySymbol(line string) (string, string, string) {
	if m := pyFuncPattern.FindStringSubmatch(line); m != nil {
		return "function", m[1], line
	}
	if m := pyClassPattern.FindStringSubmatch(line); m != nil {
		return "type", m[1], line
	}
	return "other", "", line
}

// extractFirstWordAfter 提取第 n 个空格后的第一个单词（用于粗略识别）
func extractFirstWordAfter(line string, skipWords int) string {
	fields := strings.Fields(line)
	if len(fields) > skipWords {
		// 去掉可能的括号、冒号等
		name := fields[skipWords]
		name = strings.TrimRight(name, "(){},:;=")
		return name
	}
	return ""
}

// symbolTypePriority 返回符号类型的优先级（用于排序，定义在前）
func symbolTypePriority(t string) int {
	switch t {
	case "function":
		return 0
	case "type":
		return 1
	case "interface":
		return 2
	case "const":
		return 3
	case "var":
		return 4
	case "import":
		return 5
	default:
		return 10
	}
}

// rgLanguageType 返回 ripgrep 识别的 --type 参数值
func rgLanguageType(lang string) string {
	switch lang {
	case "go":
		return "go"
	case "ts":
		return "ts"
	case "js":
		return "js"
	case "py":
		return "py"
	case "java":
		return "java"
	case "rs":
		return "rust"
	case "c":
		return "c"
	case "cpp":
		return "cpp"
	case "sh":
		return "sh"
	case "md":
		return "markdown"
	default:
		return ""
	}
}

// goDefinitionPattern 生成 Go 定义搜索的正则模式
func goDefinitionPattern(query string) string {
	// 匹配 func/type/const/var 后面跟符号名的行
	// 转义正则特殊字符（简单处理）
	escaped := regexp.QuoteMeta(query)
	return fmt.Sprintf(`^(func\s+(\([^)]*\)\s+)?|type\s+|const\s+|var\s+)%s\b`, escaped)
}
