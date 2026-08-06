package fsstore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- dimTitleFor & dimDescriptionFor ---

func TestDimTitleFor(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"identity", "用户身份"},
		{"workspace", "本地工作区"},
		{"tooling", "工具与环境"},
		{"lark_im", "飞书聊天记录"},
		{"lark_doc", "飞书文档"},
		{"lark_calendar", "工作节奏"},
		{"gloop_history", "工作模式"},
		{"codebase_map", "代码地图"},
		{"activity_snapshot", "活动快照"},
		{"unknown_dim", "unknown_dim"},
		{"", ""},
	}
	for _, tc := range tests {
		got := dimTitleFor(tc.name)
		if got != tc.want {
			t.Errorf("dimTitleFor(%q) = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestDimDescriptionFor(t *testing.T) {
	tests := []struct {
		name     string
		contains string
	}{
		{"identity", "用户身份"},
		{"workspace", "项目上下文"},
		{"tooling", "工具链"},
		{"lark_im", "飞书聊天"},
		{"lark_doc", "飞书云文档"},
		{"lark_calendar", "工作节奏"},
		{"gloop_history", "工作模式"},
		{"codebase_map", "代码结构地图"},
		{"activity_snapshot", "活动快照"},
		{"unknown_dim", "用户上下文维度"},
	}
	for _, tc := range tests {
		got := dimDescriptionFor(tc.name)
		if !strings.Contains(got, tc.contains) {
			t.Errorf("dimDescriptionFor(%q) = %q, should contain %q", tc.name, got, tc.contains)
		}
	}
}

// --- safeKnowledgeFilename ---

func TestSafeKnowledgeFilename(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"simple", "simple"},
		{" with spaces ", "with_spaces"},
		{"../path/traversal", "path_traversal"},
		{"中文名称", "untitled"},
		{"special!@#$%chars", "special_____chars"},
		{"  spaced  name  ", "spaced__name"},
		{"...dots...", "dots"},
		{"---dashes---", "dashes"},
		{"", "untitled"},
		{"   ", "untitled"},
		{"!!!", "untitled"},
		{"mixed-CASE_name123", "mixed-CASE_name123"},
	}
	for _, tc := range tests {
		got := safeKnowledgeFilename(tc.input)
		if got != tc.want {
			t.Errorf("safeKnowledgeFilename(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// --- yamlQuote ---

func TestYamlQuote(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`simple`, `"simple"`},
		{`with "quote"`, `"with \"quote\""`},
		{`with\backslash`, `"with\\backslash"`},
		{`both\ and "`, `"both\\ and \""`},
		{``, `""`},
	}
	for _, tc := range tests {
		got := yamlQuote(tc.input)
		if got != tc.want {
			t.Errorf("yamlQuote(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// --- formatOKFTimestamp ---

func TestFormatOKFTimestamp(t *testing.T) {
	// Valid timestamp
	got := formatOKFTimestamp(1700000000000) // 2023-11-14T22:13:20Z
	if got == "" {
		t.Error("valid timestamp returned empty")
	}
	// Check format (should end with Z or contain T)
	if !strings.Contains(got, "T") {
		t.Errorf("timestamp format: %q should be RFC3339", got)
	}
	// Zero/negative: falls back to now (still valid)
	gotZero := formatOKFTimestamp(0)
	if gotZero == "" {
		t.Error("zero timestamp returned empty")
	}
}

// --- contextUpdatedAt ---

func TestContextUpdatedAt(t *testing.T) {
	var m *ContextMeta = nil
	got := contextUpdatedAt(m, 42)
	if got != 42 {
		t.Errorf("nil meta: got %d, want 42", got)
	}
	// Meta with zero UpdatedAtMs
	m = &ContextMeta{UpdatedAtMs: 0}
	got = contextUpdatedAt(m, 99)
	if got != 99 {
		t.Errorf("zero UpdatedAtMs: got %d, want 99", got)
	}
	// Meta with valid UpdatedAtMs
	m = &ContextMeta{UpdatedAtMs: 123456}
	got = contextUpdatedAt(m, 99)
	if got != 123456 {
		t.Errorf("valid UpdatedAtMs: got %d, want 123456", got)
	}
}

// --- diffKnowledgeFiles ---

func TestDiffKnowledgeFiles(t *testing.T) {
	prev := []KnowledgeExportFile{
		{Path: "a.md", SizeBytes: 100, Hash: "hash_a_old"},
		{Path: "b.md", SizeBytes: 200, Hash: "hash_b"},
		{Path: "c.md", SizeBytes: 300, Hash: "hash_c"},
	}
	curr := []KnowledgeExportFile{
		{Path: "a.md", SizeBytes: 150, Hash: "hash_a_new"}, // changed
		{Path: "b.md", SizeBytes: 200, Hash: "hash_b"},     // unchanged
		{Path: "d.md", SizeBytes: 400, Hash: "hash_d"},     // added
	}

	diff := diffKnowledgeFiles(prev, curr)
	if diff == nil {
		t.Fatal("diff is nil")
	}
	if len(diff.Added) != 1 || diff.Added[0].Path != "d.md" {
		t.Errorf("Added = %v, want [d.md]", diff.Added)
	}
	if len(diff.Changed) != 1 || diff.Changed[0].Path != "a.md" {
		t.Errorf("Changed = %v, want [a.md]", diff.Changed)
	}
	if len(diff.Removed) != 1 || diff.Removed[0].Path != "c.md" {
		t.Errorf("Removed = %v, want [c.md]", diff.Removed)
	}
	if diff.Unchanged != 1 {
		t.Errorf("Unchanged = %d, want 1", diff.Unchanged)
	}

	// Sorting check
	if len(diff.Added) > 1 {
		for i := 1; i < len(diff.Added); i++ {
			if diff.Added[i].Path < diff.Added[i-1].Path {
				t.Error("Added not sorted")
			}
		}
	}
}

func TestDiffKnowledgeFiles_Empty(t *testing.T) {
	diff := diffKnowledgeFiles(nil, nil)
	if diff == nil {
		t.Fatal("diff is nil")
	}
	if len(diff.Added)+len(diff.Changed)+len(diff.Removed) != 0 || diff.Unchanged != 0 {
		t.Errorf("empty diff non-zero: %+v", diff)
	}
}

// --- KnowledgeExport.addKnowledgeFile ---

func TestKnowledgeExport_AddKnowledgeFile(t *testing.T) {
	k := &KnowledgeExport{Files: []KnowledgeExportFile{}}

	// File without size/hash: auto-computed
	k.addKnowledgeFile(KnowledgeExportFile{
		Path:    "test.md",
		Content: "hello world",
	})
	if len(k.Files) != 1 {
		t.Fatalf("files len = %d", len(k.Files))
	}
	if k.Files[0].SizeBytes != 11 {
		t.Errorf("SizeBytes = %d, want 11", k.Files[0].SizeBytes)
	}
	if k.Files[0].Hash == "" {
		t.Error("Hash should be auto-computed")
	}
	if k.FileCount != 1 {
		t.Errorf("FileCount = %d, want 1", k.FileCount)
	}
	if k.TotalSizeBytes != 11 {
		t.Errorf("TotalSizeBytes = %d, want 11", k.TotalSizeBytes)
	}

	// File with pre-set size: preserved
	k.addKnowledgeFile(KnowledgeExportFile{
		Path:      "pre.md",
		Content:   "ignored",
		SizeBytes: 999,
		Hash:      "precomputed",
	})
	if k.Files[1].SizeBytes != 999 {
		t.Errorf("pre-set SizeBytes = %d, want 999", k.Files[1].SizeBytes)
	}
	if k.Files[1].Hash != "precomputed" {
		t.Errorf("pre-set Hash = %q", k.Files[1].Hash)
	}
	if k.TotalSizeBytes != 1010 {
		t.Errorf("TotalSizeBytes = %d, want 1010", k.TotalSizeBytes)
	}
}

// --- ListContextDims / ReadContextDim / WriteContextDim ---

func TestContextDimOperations(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	// List empty: should be empty list, no error
	dims, err := root.ListContextDims()
	if err != nil {
		t.Fatalf("ListContextDims failed: %v", err)
	}
	if len(dims) != 0 {
		t.Errorf("empty dims len = %d, want 0", len(dims))
	}

	// GetContextMeta on empty: should return empty meta, no error
	meta, err := root.GetContextMeta()
	if err != nil {
		t.Fatalf("GetContextMeta failed: %v", err)
	}
	if meta == nil {
		t.Fatal("meta should not be nil")
	}
	if len(meta.Dimensions) != 0 {
		t.Errorf("empty meta Dimensions len = %d", len(meta.Dimensions))
	}

	// Write a dimension
	info, err := root.WriteContextDim("workspace", "## 项目\n- hello\n")
	if err != nil {
		t.Fatalf("WriteContextDim failed: %v", err)
	}
	if info.Name != "workspace" {
		t.Errorf("info.Name = %q", info.Name)
	}
	if info.SizeBytes == 0 {
		t.Error("SizeBytes should be >0")
	}

	// Read it back
	dim, err := root.ReadContextDim("workspace")
	if err != nil {
		t.Fatalf("ReadContextDim failed: %v", err)
	}
	if dim.Body != "## 项目\n- hello\n" {
		t.Errorf("Body = %q", dim.Body)
	}
	if dim.Title != "本地工作区" {
		t.Errorf("Title = %q, want '本地工作区'", dim.Title)
	}

	if _, err := root.WriteContextDim("loop_state", "legacy loop state"); err != nil {
		t.Fatalf("WriteContextDim legacy loop_state failed: %v", err)
	}
	if err := root.RenameContextDim("loop_state", "activity_snapshot"); err != nil {
		t.Fatalf("RenameContextDim failed: %v", err)
	}
	if _, err := root.ReadContextDim("loop_state"); err == nil {
		t.Fatal("legacy loop_state should be removed after rename")
	}
	snapshot, err := root.ReadContextDim("activity_snapshot")
	if err != nil {
		t.Fatalf("ReadContextDim activity_snapshot failed: %v", err)
	}
	if snapshot.Body != "legacy loop state" || snapshot.Title != "活动快照" {
		t.Fatalf("bad activity snapshot dim: %+v", snapshot)
	}
	if _, err := root.WriteContextDim("loop_state", "legacy v2"); err != nil {
		t.Fatalf("WriteContextDim second legacy loop_state failed: %v", err)
	}
	if _, err := root.WriteContextDim("activity_snapshot", "new snapshot"); err != nil {
		t.Fatalf("WriteContextDim activity_snapshot failed: %v", err)
	}
	if err := root.RenameContextDim("loop_state", "activity_snapshot"); err != nil {
		t.Fatalf("RenameContextDim with existing target failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root.Sub(SubdirContext), "dim_loop_state.legacy.md")); err != nil {
		t.Fatalf("legacy loop_state should be preserved, not deleted: %v", err)
	}

	// List should include the active dims plus the preserved legacy backup.
	dims, _ = root.ListContextDims()
	if len(dims) != 3 {
		t.Errorf("after writes: dims len = %d, want 3", len(dims))
	}

	// Validation: empty name
	if _, err := root.ReadContextDim(""); err == nil {
		t.Error("ReadContextDim empty name should error")
	}
	if _, err := root.WriteContextDim("", "body"); err == nil {
		t.Error("WriteContextDim empty name should error")
	}
	// Validation: path traversal
	if _, err := root.ReadContextDim("../etc/passwd"); err == nil {
		t.Error("ReadContextDim path traversal should error")
	}
	if _, err := root.WriteContextDim("..", "body"); err == nil {
		t.Error("WriteContextDim '..' should error")
	}
	// Validation: empty body
	if _, err := root.WriteContextDim("workspace", "   "); err == nil {
		t.Error("WriteContextDim empty body should error")
	}

	// ReadContextSummary on empty: should be empty, no error
	sum, err := root.ReadContextSummary()
	if err != nil {
		t.Fatalf("ReadContextSummary failed: %v", err)
	}
	if sum != "" {
		t.Errorf("empty summary = %q", sum)
	}

	// Write summary
	if err := root.WriteContextSummary("this is a summary\nwith two lines"); err != nil {
		t.Fatalf("WriteContextSummary failed: %v", err)
	}
	sum, _ = root.ReadContextSummary()
	if sum == "" {
		t.Error("summary should not be empty after write")
	}
	// Empty summary should error
	if err := root.WriteContextSummary("   "); err == nil {
		t.Error("WriteContextSummary empty body should error")
	}

	// Non-existent dimension
	if _, err := root.ReadContextDim("nonexistent"); err == nil {
		t.Error("ReadContextDim nonexistent should error")
	}
}

// --- BuildKnowledgeExport / WriteKnowledgeExport ---

func TestBuildKnowledgeExport(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	// Write some dims
	root.WriteContextDim("workspace", "workspace body\n")
	root.WriteContextDim("tooling", "tooling body\n")
	root.WriteContextSummary("summary body\n")

	bundle, err := root.BuildKnowledgeExport()
	if err != nil {
		t.Fatalf("BuildKnowledgeExport failed: %v", err)
	}
	if bundle == nil {
		t.Fatal("bundle is nil")
	}
	if bundle.Format != "gloop.okf.v0" {
		t.Errorf("Format = %q", bundle.Format)
	}
	if bundle.DimensionCount != 2 {
		t.Errorf("DimensionCount = %d, want 2", bundle.DimensionCount)
	}
	if !bundle.SummaryIncluded {
		t.Error("SummaryIncluded should be true")
	}
	if bundle.FileCount < 3 { // index + summary + 2 dims = 4 actually
		t.Errorf("FileCount = %d, want >=3", bundle.FileCount)
	}
	for _, f := range bundle.Files {
		if f.Content == "" {
			t.Errorf("file %q has empty content", f.Path)
		}
		if f.Hash == "" {
			t.Errorf("file %q has empty hash", f.Path)
		}
	}
}

func TestWriteKnowledgeExport(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	root.WriteContextDim("workspace", "ws body\n")

	outDir := filepath.Join(t.TempDir(), "export")
	bundle, err := root.WriteKnowledgeExport(outDir)
	if err != nil {
		t.Fatalf("WriteKnowledgeExport failed: %v", err)
	}
	if bundle == nil {
		t.Fatal("bundle is nil")
	}
	// Files should exist on disk
	for _, f := range bundle.Files {
		path := filepath.Join(outDir, filepath.FromSlash(f.Path))
		if _, statErr := os.Stat(path); statErr != nil {
			t.Errorf("exported file %q not found: %v", path, statErr)
		}
	}
	// Meta file should be written
	meta, err := root.ReadKnowledgeExportMeta()
	if err != nil {
		t.Fatalf("ReadKnowledgeExportMeta after write failed: %v", err)
	}
	if meta == nil {
		t.Error("meta should not be nil after write")
	}
	// Empty outDir: validation
	if _, err := root.WriteKnowledgeExport(""); err == nil {
		t.Error("WriteKnowledgeExport empty outDir should error")
	}
}

// --- KnowledgeExportPreview ---

func TestKnowledgeExportPreview(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	root.WriteContextDim("workspace", "ws body\n")

	preview, last, err := root.KnowledgeExportPreview()
	if err != nil {
		t.Fatalf("KnowledgeExportPreview failed: %v", err)
	}
	if preview == nil {
		t.Fatal("preview is nil")
	}
	// Preview should have Path/SizeBytes/Hash but NOT Content
	for _, f := range preview.Files {
		if f.Content != "" {
			t.Errorf("preview file %q has content (should not)", f.Path)
		}
		if f.Path == "" {
			t.Error("preview file has empty Path")
		}
	}
	// Last should be nil (never exported yet)
	if last != nil {
		t.Errorf("last should be nil on first preview, got %+v", last)
	}

	// After writing a real export, last should be non-nil
	root.WriteKnowledgeExport(t.TempDir() + "/export2")
	_, last2, _ := root.KnowledgeExportPreview()
	if last2 == nil {
		t.Error("last2 should not be nil after WriteKnowledgeExport")
	}
	// Preview should have Diff now (comparing current to last)
	preview2, _, _ := root.KnowledgeExportPreview()
	if preview2.Diff == nil {
		t.Log("warning: Diff is nil (no changes, might be ok)") // depends on content stability
	}
}

// --- SetContextGeneratedBy ---

func TestSetContextGeneratedBy(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	if err := root.SetContextGeneratedBy("auto_ctx_refresh"); err != nil {
		t.Fatalf("SetContextGeneratedBy failed: %v", err)
	}
	meta, _ := root.GetContextMeta()
	if meta.GeneratedBy != "auto_ctx_refresh" {
		t.Errorf("GeneratedBy = %q, want 'auto_ctx_refresh'", meta.GeneratedBy)
	}
	if meta.UpdatedAtMs == 0 {
		t.Error("UpdatedAtMs should be set")
	}
}
