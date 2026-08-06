package fsstore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- countLinesFromBytes ---

func TestCountLinesFromBytes(t *testing.T) {
	tests := []struct {
		data []byte
		want int
	}{
		{[]byte(""), 0},
		{[]byte("\n"), 1},
		{[]byte("a"), 1},
		{[]byte("a\n"), 1},
		{[]byte("a\nb"), 2},
		{[]byte("a\nb\n"), 2},
		{[]byte("a\nb\nc"), 3},
		{[]byte("a\nb\nc\n"), 3},
		{[]byte("single line without newline"), 1},
	}
	for i, tc := range tests {
		got := countLinesFromBytes(tc.data)
		if got != tc.want {
			t.Errorf("[%d] countLinesFromBytes(%q) = %d, want %d", i, string(tc.data), got, tc.want)
		}
	}
}

// --- countLines (file-based) ---

func TestCountLines(t *testing.T) {
	dir := t.TempDir()

	// Non-existent file: returns 1 (fallback)
	if n := countLines(filepath.Join(dir, "nope.txt")); n != 1 {
		t.Errorf("non-existent: got %d, want 1", n)
	}

	tests := []struct {
		content string
		want    int
	}{
		{"", 0},
		{"\n", 1},
		{"hello", 1},
		{"a\nb\nc", 3},
		{"a\nb\nc\n", 3},
	}
	for i, tc := range tests {
		path := filepath.Join(dir, "f"+string(rune('0'+i))+".txt")
		if err := os.WriteFile(path, []byte(tc.content), 0o644); err != nil {
			t.Fatal(err)
		}
		if got := countLines(path); got != tc.want {
			t.Errorf("[%d] countLines = %d, want %d (content=%q)", i, got, tc.want, tc.content)
		}
	}
}

// --- diffFileLines ---

func TestDiffFileLines(t *testing.T) {
	dir := t.TempDir()

	writeFile := func(name, content string) string {
		p := filepath.Join(dir, name)
		os.WriteFile(p, []byte(content), 0o644)
		return p
	}

	// Same file
	p1 := writeFile("same.txt", "a\nb\nc\n")
	add, del := diffFileLines(p1, p1)
	if add != 0 || del != 0 {
		t.Errorf("same file: add=%d del=%d, want 0/0", add, del)
	}

	// Added lines
	p2 := writeFile("base.txt", "a\nb\n")
	p3 := writeFile("new.txt", "a\nb\nc\nd\n")
	add, del = diffFileLines(p2, p3)
	if add != 2 {
		t.Errorf("2 added: got add=%d, want 2", add)
	}
	if del != 0 {
		t.Errorf("2 added: got del=%d, want 0", del)
	}

	// Deleted lines
	add, del = diffFileLines(p3, p2)
	if add != 0 {
		t.Errorf("2 deleted: got add=%d, want 0", add)
	}
	if del != 2 {
		t.Errorf("2 deleted: got del=%d, want 2", del)
	}

	// Non-existent base
	add, del = diffFileLines(filepath.Join(dir, "nope.txt"), p1)
	// Should use countLines fallback
	if add == 0 {
		t.Errorf("missing base: add should be >0")
	}
}

// --- filesEqual ---

func TestFilesEqual(t *testing.T) {
	dir := t.TempDir()
	writeFile := func(name, content string) string {
		p := filepath.Join(dir, name)
		os.WriteFile(p, []byte(content), 0o644)
		return p
	}

	a := writeFile("a.txt", "hello world")
	b := writeFile("b.txt", "hello world")
	c := writeFile("c.txt", "different")
	d := writeFile("d.txt", "hello world!!") // longer

	ai, _ := os.Stat(a)
	bi, _ := os.Stat(b)
	ci, _ := os.Stat(c)
	di, _ := os.Stat(d)

	eq, err := filesEqual(a, b, ai, bi)
	if err != nil || !eq {
		t.Errorf("a==b: got eq=%v err=%v", eq, err)
	}
	eq, err = filesEqual(a, c, ai, ci)
	if err != nil || eq {
		t.Errorf("a==c (diff content same size): got eq=%v err=%v", eq, err)
	}
	// d has different size
	eq, err = filesEqual(a, d, ai, di)
	if err != nil || eq {
		t.Errorf("a==d (diff size): got eq=%v err=%v", eq, err)
	}
}

// --- copyFile & copyDirWithIgnore ---

func TestCopyFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	content := strings.Repeat("x", 5000) + "\n"
	if err := os.WriteFile(src, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(src, dst); err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}
	got, _ := os.ReadFile(dst)
	if string(got) != content {
		t.Errorf("copyFile content mismatch: len got=%d want=%d", len(got), len(content))
	}

	// Source doesn't exist
	if err := copyFile(filepath.Join(dir, "nope"), dst); err == nil {
		t.Error("copy nonexistent src should fail")
	}
}

// --- PrepareSandboxCopy validation ---

func TestPrepareSandboxCopy_Validation(t *testing.T) {
	qs := NewQuestStore(nil) // nil root is ok for validation path
	_, err := qs.PrepareSandboxCopy("", "sid", "/tmp")
	if err == nil {
		t.Error("empty qid should error")
	}
	_, err = qs.PrepareSandboxCopy("qid", "", "/tmp")
	if err == nil {
		t.Error("empty sid should error")
	}
	_, err = qs.PrepareSandboxCopy("qid", "sid", "")
	if err == nil {
		t.Error("empty sourceDir should error")
	}
}

// --- SandboxDiff is a wrapper around diffDirs ---

func TestSandboxDiff(t *testing.T) {
	base := t.TempDir()
	sandbox := t.TempDir()
	// Create a file in both (same)
	os.WriteFile(filepath.Join(base, "a.txt"), []byte("a\n"), 0o644)
	os.WriteFile(filepath.Join(sandbox, "a.txt"), []byte("a\n"), 0o644)
	// New file in sandbox
	os.WriteFile(filepath.Join(sandbox, "b.txt"), []byte("new\n"), 0o644)

	files, add, del, err := SandboxDiff(base, sandbox)
	if err != nil {
		t.Fatalf("SandboxDiff failed: %v", err)
	}
	foundNew := false
	for _, f := range files {
		if f.Path == "b.txt" && f.Status == "added" {
			foundNew = true
		}
	}
	if !foundNew {
		t.Errorf("b.txt added not found in diff: %v", files)
	}
	_ = add
	_ = del
}

// --- RemoveSandbox ---

func TestRemoveSandbox(t *testing.T) {
	qs := NewQuestStore(nil)
	// Empty ids: no-op (nil quest dir)
	if err := qs.RemoveSandbox("", ""); err != nil {
		t.Errorf("RemoveSandbox with empty IDs should be no-op, got err=%v", err)
	}
}
