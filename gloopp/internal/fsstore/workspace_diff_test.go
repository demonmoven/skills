package fsstore

import (
	"testing"
)

// --- parsePatchStats tests ---

func TestParsePatchStats_SimpleDiff(t *testing.T) {
	raw := `diff --git a/file.txt b/file.txt
index 1111111..2222222 100644
--- a/file.txt
+++ b/file.txt
@@ -1,3 +1,3 @@
 line1
-old
+new
 line3
`
	files, adds, dels := parsePatchStats(raw)
	if len(files) != 1 {
		t.Fatalf("files len = %d, want 1", len(files))
	}
	if files[0].Path != "file.txt" {
		t.Errorf("Path = %q, want file.txt", files[0].Path)
	}
	if files[0].Status != "modified" {
		t.Errorf("Status = %q, want modified", files[0].Status)
	}
	if files[0].Additions != 1 {
		t.Errorf("Additions = %d, want 1", files[0].Additions)
	}
	if files[0].Deletions != 1 {
		t.Errorf("Deletions = %d, want 1", files[0].Deletions)
	}
	if adds != 1 || dels != 1 {
		t.Errorf("totals adds=%d dels=%d, want 1/1", adds, dels)
	}
}

func TestParsePatchStats_NewFile(t *testing.T) {
	raw := `diff --git a/new.go b/new.go
new file mode 100644
index 0000000..3333333
--- /dev/null
+++ b/new.go
@@ -0,0 +1,2 @@
+line1
+line2
`
	files, adds, dels := parsePatchStats(raw)
	if len(files) != 1 {
		t.Fatalf("files len = %d, want 1", len(files))
	}
	if files[0].Status != "added" {
		t.Errorf("Status = %q, want added", files[0].Status)
	}
	if files[0].Additions != 2 {
		t.Errorf("Additions = %d, want 2", files[0].Additions)
	}
	if adds != 2 || dels != 0 {
		t.Errorf("totals adds=%d dels=%d, want 2/0", adds, dels)
	}
}

func TestParsePatchStats_DeletedFile(t *testing.T) {
	raw := `diff --git a/old.txt b/old.txt
deleted file mode 100644
index 4444444..0000000
--- a/old.txt
+++ /dev/null
@@ -1,3 +0,0 @@
-a
-b
-c
`
	files, adds, dels := parsePatchStats(raw)
	if len(files) != 1 {
		t.Fatalf("files len = %d, want 1", len(files))
	}
	if files[0].Status != "deleted" {
		t.Errorf("Status = %q, want deleted", files[0].Status)
	}
	if files[0].Deletions != 3 {
		t.Errorf("Deletions = %d, want 3", files[0].Deletions)
	}
	if adds != 0 || dels != 3 {
		t.Errorf("totals adds=%d dels=%d, want 0/3", adds, dels)
	}
}

func TestParsePatchStats_MultipleFiles(t *testing.T) {
	raw := `diff --git a/a.go b/a.go
index 111..222 100644
--- a/a.go
+++ b/a.go
@@ -1,3 +1,3 @@
-aa
+AA
 bb
diff --git a/b.go b/b.go
new file mode 100644
index 000..333
--- /dev/null
+++ b/b.go
@@ -0,0 +1,1 @@
+new
`
	files, adds, dels := parsePatchStats(raw)
	if len(files) != 2 {
		t.Fatalf("files len = %d, want 2", len(files))
	}
	if adds != 2 || dels != 1 {
		t.Errorf("totals adds=%d dels=%d, want 2/1", adds, dels)
	}
}

func TestParsePatchStats_Empty(t *testing.T) {
	files, adds, dels := parsePatchStats("")
	if len(files) != 0 || adds != 0 || dels != 0 {
		t.Errorf("empty patch: got files=%d adds=%d dels=%d, all should be 0", len(files), adds, dels)
	}
}

// --- parseDiffGitPath ---

func TestParseDiffGitPath(t *testing.T) {
	tests := []struct {
		line string
		want string
	}{
		{"diff --git a/file.txt b/file.txt", "file.txt"},
		{"diff --git a/path/to/file.go b/path/to/file.go", "path/to/file.go"},
		{"diff --git a/only b/only", "only"},
		{"short line", ""},
	}
	for _, tc := range tests {
		got := parseDiffGitPath(tc.line)
		if got != tc.want {
			t.Errorf("parseDiffGitPath(%q) = %q, want %q", tc.line, got, tc.want)
		}
	}
}

// --- parseNumstat & parseNameStatus & parseCount ---

func TestParseNumstat(t *testing.T) {
	raw := "1\t2\tfile.go\n-\t-\tbinary.bin\n10\t0\tnew.md\n"
	files, adds, dels := parseNumstat(raw)
	if len(files) != 3 {
		t.Fatalf("files len = %d, want 3", len(files))
	}
	if adds != 11 || dels != 2 {
		t.Errorf("totals adds=%d dels=%d, want 11/2", adds, dels)
	}
	// binary file => 0/0
	for _, f := range files {
		if f.Path == "binary.bin" {
			if f.Additions != 0 || f.Deletions != 0 {
				t.Errorf("binary: adds=%d dels=%d, want 0/0", f.Additions, f.Deletions)
			}
			if f.Status != "modified" {
				t.Errorf("binary status = %q", f.Status)
			}
		}
	}
}

func TestParseNameStatus(t *testing.T) {
	raw := "A\tnew.go\nM\tmodified.go\nD\tdeleted.go\n"
	status := parseNameStatus(raw)
	if status["new.go"] != "added" {
		t.Errorf("new.go = %q", status["new.go"])
	}
	if status["modified.go"] != "modified" {
		t.Errorf("modified.go = %q", status["modified.go"])
	}
	if status["deleted.go"] != "deleted" {
		t.Errorf("deleted.go = %q", status["deleted.go"])
	}
	if len(status) != 3 {
		t.Errorf("len = %d, want 3", len(status))
	}
	// Empty
	empty := parseNameStatus("")
	if len(empty) != 0 {
		t.Errorf("empty parseNameStatus len = %d", len(empty))
	}
}

func TestParseCount(t *testing.T) {
	if n := parseCount("-"); n != 0 {
		t.Errorf("'-' = %d, want 0", n)
	}
	if n := parseCount("0"); n != 0 {
		t.Errorf("'0' = %d", n)
	}
	if n := parseCount("42"); n != 42 {
		t.Errorf("'42' = %d, want 42", n)
	}
	if n := parseCount("123abc"); n != 0 {
		t.Errorf("'123abc' = %d, want 0", n)
	}
	if n := parseCount("abc"); n != 0 {
		t.Errorf("'abc' = %d, want 0", n)
	}
	if n := parseCount(""); n != 0 {
		t.Errorf("'' = %d, want 0", n)
	}
}

// --- ApplyConflictError.Error ---

func TestApplyConflictErrorString(t *testing.T) {
	// No files
	e := &ApplyConflictError{}
	if e.Error() == "" {
		t.Error("no files: Error() should not be empty")
	}
	// With files
	e2 := &ApplyConflictError{Files: []string{"a.go", "b.go"}}
	msg := e2.Error()
	if msg == "" {
		t.Error("with files: Error() should not be empty")
	}
}

// --- parseNumstatWithStatus integration ---

func TestParseNumstatWithStatus(t *testing.T) {
	numstat := "0\t1\tdel.go\n5\t0\tnew.go\n"
	statusByPath := map[string]string{
		"del.go":   "deleted",
		"new.go":   "added",
		"extra.go": "modified", // in nameStatus but not numstat
	}
	files, adds, dels := parseNumstatWithStatus(numstat, statusByPath)
	if adds != 5 || dels != 1 {
		t.Errorf("totals adds=%d dels=%d, want 5/1", adds, dels)
	}
	// extra.go should be appended
	foundExtra := false
	for _, f := range files {
		if f.Path == "extra.go" {
			foundExtra = true
			if f.Status != "modified" {
				t.Errorf("extra.go status = %q", f.Status)
			}
		}
	}
	if !foundExtra {
		t.Error("extra.go not found in result (should be appended from statusByPath)")
	}
}
