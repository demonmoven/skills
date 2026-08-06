package platformtools

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

)

// ==================== code_search 搜索能力测试 ====================
//
// v0.3.4 起原 platform tool 分发层（codeSearchToolDef / codeSearchHandler）已移除，
// 这里只测保留下来的搜索能力本身（RunCodeSearch / runCodeSearch / 纯 Go 回退）。

func TestRunCodeSearch_Basic(t *testing.T) {
	// 创建临时工作区，放几个测试文件
	workDir := t.TempDir()

	// 写一个 Go 文件
	goContent := `package foo

type User struct {
	Name string
	Age  int
}

func NewUser(name string) *User {
	return &User{Name: name, Age: 1}
}

func (u *User) Hello() string {
	return "Hello, " + u.Name
}

const MaxUsers = 100

var DefaultUser = &User{}
`
	err := os.WriteFile(filepath.Join(workDir, "user.go"), []byte(goContent), 0644)
	if err != nil {
		t.Fatalf("write test file failed: %v", err)
	}

	// 写一个测试文件
	testContent := `package foo

import "testing"

func TestNewUser(t *testing.T) {
	u := NewUser("test")
	if u.Name != "test" {
		t.Fail()
	}
}
`
	err = os.WriteFile(filepath.Join(workDir, "user_test.go"), []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("write test file failed: %v", err)
	}

	t.Run("definition mode", func(t *testing.T) {
		results, _, err := runCodeSearch(workDir, "NewUser", "definition", "go", false, 10)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d: %+v", len(results), results)
		}
		if results[0].SymbolType != "function" {
			t.Errorf("symbol_type = %s, want function", results[0].SymbolType)
		}
		if results[0].SymbolName != "NewUser" {
			t.Errorf("symbol_name = %s, want NewUser", results[0].SymbolName)
		}
	})

	t.Run("exclude tests by default", func(t *testing.T) {
		results, _, err := runCodeSearch(workDir, "TestNewUser", "all", "go", false, 10)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("默认应该排除测试文件，但搜到了 %d 条", len(results))
		}
	})

	t.Run("include tests", func(t *testing.T) {
		results, _, err := runCodeSearch(workDir, "TestNewUser", "all", "go", true, 10)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		if len(results) == 0 {
			t.Error("include_tests=true 时应该能搜到测试文件")
		}
	})

	t.Run("type definition", func(t *testing.T) {
		results, _, err := runCodeSearch(workDir, "User", "definition", "go", false, 10)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		// 应该搜到 type User struct
		found := false
		for _, r := range results {
			if r.SymbolType == "type" && r.SymbolName == "User" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("没找到 User 类型定义，结果: %+v", results)
		}
	})

	t.Run("limit truncation", func(t *testing.T) {
		results, _, err := runCodeSearch(workDir, "User", "all", "go", false, 1)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		if len(results) > 1 {
			t.Errorf("limit=1 但返回了 %d 条", len(results))
		}
	})

	t.Run("no matches", func(t *testing.T) {
		results, _, err := runCodeSearch(workDir, "NonExistentSymbol", "definition", "go", false, 10)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("应该搜不到，结果: %+v", results)
		}
	})
}

func TestIdentifyGoSymbol(t *testing.T) {
	cases := []struct {
		line     string
		wantType string
		wantName string
	}{
		{"func Foo() {}", "function", "Foo"},
		{"func (r *Receiver) Bar(x int) string {", "function", "Bar"},
		{"type User struct {", "type", "User"},
		{"type Handler interface {", "interface", "Handler"},
		{"const MaxCount = 100", "const", "MaxCount"},
		{"var ErrNotFound = errors.New(\"not found\")", "var", "ErrNotFound"},
		{"import \"fmt\"", "import", ""},
		{"    fmt.Println(\"hello\")", "other", ""},
	}

	for _, c := range cases {
		symType, symName, _ := identifyGoSymbol(c.line)
		if symType != c.wantType {
			t.Errorf("%q: type = %s, want %s", c.line, symType, c.wantType)
		}
		if symName != c.wantName {
			t.Errorf("%q: name = %s, want %s", c.line, symName, c.wantName)
		}
	}
}

func TestRgLanguageType(t *testing.T) {
	cases := map[string]string{
		"go":   "go",
		"ts":   "ts",
		"js":   "js",
		"py":   "py",
		"java": "java",
		"rs":   "rust",
		"c":    "c",
		"cpp":  "cpp",
		"sh":   "sh",
		"md":   "markdown",
		"xx":   "",
	}
	for lang, want := range cases {
		got := rgLanguageType(lang)
		if got != want {
			t.Errorf("rgLanguageType(%q) = %q, want %q", lang, got, want)
		}
	}
}

func TestSymbolTypePriority(t *testing.T) {
	// function 优先级最高（排最前），other 最低
	if symbolTypePriority("function") >= symbolTypePriority("type") {
		t.Error("function 优先级应该高于 type")
	}
	if symbolTypePriority("type") >= symbolTypePriority("var") {
		t.Error("type 优先级应该高于 var")
	}
	if symbolTypePriority("var") >= symbolTypePriority("other") {
		t.Error("var 优先级应该高于 other")
	}
}

// ==================== pure_go fallback 测试 ====================

func TestRunCodeSearch_PureGoFallback(t *testing.T) {
	workDir := t.TempDir()

	// 写一个 Go 文件
	goContent := `package foo

type User struct {
	Name string
	Age  int
}

func NewUser(name string) *User {
	return &User{Name: name, Age: 1}
}

func (u *User) Hello() string {
	return "Hello, " + u.Name
}

const MaxUsers = 100

var DefaultUser = &User{}
`
	err := os.WriteFile(filepath.Join(workDir, "user.go"), []byte(goContent), 0o644)
	if err != nil {
		t.Fatalf("write test file failed: %v", err)
	}

	t.Run("definition mode", func(t *testing.T) {
		results, err := runCodeSearchPureGo(workDir, "NewUser", "definition", "go", false, 10)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d: %+v", len(results), results)
		}
		if results[0].SymbolType != "function" {
			t.Errorf("symbol_type = %s, want function", results[0].SymbolType)
		}
		if results[0].SymbolName != "NewUser" {
			t.Errorf("symbol_name = %s, want NewUser", results[0].SymbolName)
		}
	})

	t.Run("exclude tests by default", func(t *testing.T) {
		// 写一个测试文件
		testContent := `package foo

import "testing"

func TestNewUser(t *testing.T) {
	u := NewUser("test")
	if u.Name != "test" {
		t.Fail()
	}
}
`
		err := os.WriteFile(filepath.Join(workDir, "user_test.go"), []byte(testContent), 0o644)
		if err != nil {
			t.Fatalf("write test file failed: %v", err)
		}
		results, err := runCodeSearchPureGo(workDir, "TestNewUser", "all", "go", false, 10)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("默认应该排除测试文件，但搜到了 %d 条", len(results))
		}
	})

	t.Run("include tests", func(t *testing.T) {
		results, err := runCodeSearchPureGo(workDir, "TestNewUser", "all", "go", true, 10)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		if len(results) == 0 {
			t.Error("include_tests=true 时应该能搜到测试文件")
		}
	})

	t.Run("no matches", func(t *testing.T) {
		results, err := runCodeSearchPureGo(workDir, "NonExistentSymbol", "definition", "go", false, 10)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("应该搜不到，结果: %+v", results)
		}
	})

	t.Run("language filter", func(t *testing.T) {
		// 写一个 Python 文件
		pyContent := `def hello():
    return "world"

class MyClass:
    pass
`
		err := os.WriteFile(filepath.Join(workDir, "demo.py"), []byte(pyContent), 0o644)
		if err != nil {
			t.Fatalf("write py file failed: %v", err)
		}

		// go 语言过滤应该搜不到 py 文件
		results, err := runCodeSearchPureGo(workDir, "hello", "all", "go", false, 10)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("go 语言过滤应该排除 py 文件，但搜到了 %d 条", len(results))
		}

		// py 语言过滤应该能搜到
		results, err = runCodeSearchPureGo(workDir, "hello", "all", "py", false, 10)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		if len(results) == 0 {
			t.Error("py 语言过滤应该能搜到 py 文件")
		}
	})

	t.Run("skips hidden dirs", func(t *testing.T) {
		// 写一个 .git 目录里的文件
		gitDir := filepath.Join(workDir, ".git")
		os.MkdirAll(gitDir, 0o755)
		os.WriteFile(filepath.Join(gitDir, "config"), []byte("name = test"), 0o644)

		results, err := runCodeSearchPureGo(workDir, "name", "all", "", false, 10)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		for _, r := range results {
			if strings.Contains(r.File, ".git/") {
				t.Errorf("不应该搜到 .git 目录下的文件: %s", r.File)
			}
		}
	})
}

// TestCodeSearch_BackendConsistency 验证 rg 和 pure_go 两种后端结果一致（基础场景）
func TestCodeSearch_BackendConsistency(t *testing.T) {
	if _, err := exec.LookPath("rg"); err != nil {
		t.Skip("ripgrep not available, skipping consistency test")
	}

	workDir := t.TempDir()
	goContent := `package foo

type User struct {
	Name string
}

func NewUser(name string) *User {
	return &User{Name: name}
}

const MaxUsers = 100
`
	os.WriteFile(filepath.Join(workDir, "user.go"), []byte(goContent), 0o644)

	cases := []struct {
		query    string
		mode     string
		language string
	}{
		{"NewUser", "definition", "go"},
		{"User", "definition", "go"},
		{"MaxUsers", "all", "go"},
	}

	for _, c := range cases {
		t.Run(c.query+"_"+c.mode, func(t *testing.T) {
			rgResults, _, err := runCodeSearch(workDir, c.query, c.mode, c.language, false, 10)
			if err != nil {
				t.Fatalf("rg search failed: %v", err)
			}
			pgResults, err := runCodeSearchPureGo(workDir, c.query, c.mode, c.language, false, 10)
			if err != nil {
				t.Fatalf("pure_go search failed: %v", err)
			}
			if len(rgResults) != len(pgResults) {
				t.Errorf("结果数不一致: rg=%d, pure_go=%d", len(rgResults), len(pgResults))
			}
		})
	}
}
