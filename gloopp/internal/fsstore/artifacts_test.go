package fsstore

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestArtifactKind(t *testing.T) {
	tests := []struct {
		mime string
		name string
		want string
	}{
		{"image/png", "logo.png", "image"},
		{"image/jpeg", "photo.jpg", "image"},
		{"text/plain", "log.txt", "log"},
		{"text/markdown", "readme.md", "log"},
		{"", "run.log", "log"},
		{"application/pdf", "spec.pdf", "document"},
		{"", "paper.pdf", "document"},
		{"", "notes.docx", "document"},
		{"", "design.md", "document"},
		{"application/zip", "archive.zip", "archive"},
		{"", "bundle.tar.gz", "archive"},
		{"", "code.tar", "archive"},
		{"application/json", "data.json", "file"},
		{"application/octet-stream", "unknown.bin", "unknown"},
		{"", "nofile", "unknown"},
	}
	for _, tc := range tests {
		got := artifactKind(tc.mime, tc.name)
		if got != tc.want {
			t.Errorf("artifactKind(%q, %q) = %q, want %q", tc.mime, tc.name, got, tc.want)
		}
	}
}

func TestSanitizeArtifactName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"simple.txt", "simple.txt"},
		{"   spaced.txt   ", "spaced.txt"},
		{"path/to/file.txt", "file.txt"},
		{"../../../etc/passwd", "passwd"},
		{".", ""},
		{"/", ""},
		{"", ""},
		{"has\x00null.txt", "hasnull.txt"},
		{"has\x1fgood.txt", "hasgood.txt"},
		{"has\x7fbad.txt", "hasbad.txt"},
		// length truncation (with extension preserved)
		{strings.Repeat("a", 200) + ".txt", strings.Repeat("a", 156) + ".txt"},
		// very long extension
		{"name." + strings.Repeat("x", 50), "name." + strings.Repeat("x", 50)},
	}
	for _, tc := range tests {
		got := sanitizeArtifactName(tc.input)
		if got != tc.want {
			t.Errorf("sanitizeArtifactName(%q) = %q, want %q (len got=%d want=%d)", tc.input, got, tc.want, len(got), len(tc.want))
		}
	}
}

func TestIsLikelySafeArtifactName(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"safe.txt", true},
		{"my file name.md", true},
		{"archive-v1.2.3.tar.gz", true},
		{"under_score.log", true},
		{"../escape", false},
		{"path/name.txt", false},
		{"name\x00.txt", false},
		{"", false},
		{"../../etc/passwd", false},
		{"$(rm -rf /)", false},
	}
	for _, tc := range tests {
		got := IsLikelySafeArtifactName(tc.name)
		if got != tc.want {
			t.Errorf("IsLikelySafeArtifactName(%q) = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestSaveAndOpenArtifact(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	qs := NewQuestStore(root)
	q := &QuestMeta{
		ID:     "qst_art_test",
		Status: "running",
	}
	if err := qs.CreateQuest(q); err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}

	content := "hello artifact world\nline 2\n"
	input := ArtifactInput{
		Name:   "test-output.txt",
		MIME:   "text/plain",
		Source: "warrior_phase",
		Reader: strings.NewReader(content),
	}
	art, err := qs.SaveArtifact("qst_art_test", input)
	if err != nil {
		t.Fatalf("SaveArtifact failed: %v", err)
	}
	if art.ID == "" {
		t.Error("artifact ID should not be empty")
	}
	if art.Size != int64(len(content)) {
		t.Errorf("Size = %d, want %d", art.Size, len(content))
	}
	if art.Kind != "log" {
		t.Errorf("Kind = %q, want 'log'", art.Kind)
	}
	if art.Source != "warrior_phase" {
		t.Errorf("Source = %q, want 'warrior_phase'", art.Source)
	}
	if art.SHA256 == "" {
		t.Error("SHA256 should not be empty")
	}

	// Save as quest output
	q, _ = qs.LoadQuest("qst_art_test")
	q.Outputs = append(q.Outputs, art)
	if err := qs.SaveQuest(q); err != nil {
		t.Fatalf("SaveQuest failed: %v", err)
	}

	// Open it back
	f, gotMeta, err := qs.OpenArtifact("qst_art_test", art.ID)
	if err != nil {
		t.Fatalf("OpenArtifact failed: %v", err)
	}
	defer f.Close()
	if gotMeta.ID != art.ID {
		t.Errorf("OpenArtifact returned wrong ID: %q vs %q", gotMeta.ID, art.ID)
	}
	buf := new(bytes.Buffer)
	buf.ReadFrom(f)
	if buf.String() != content {
		t.Errorf("OpenArtifact content = %q, want %q", buf.String(), content)
	}

	// Non-existent artifact
	_, _, err = qs.OpenArtifact("qst_art_test", "art_nonexistent")
	if err == nil {
		t.Error("OpenArtifact for non-existent artifact should fail")
	}
}

func TestAddDeclaredOutput(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	qs := NewQuestStore(root)
	q := &QuestMeta{ID: "qst_decl_test", Status: "running"}
	if err := qs.CreateQuest(q); err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}

	decl := DeclaredOutput{
		Name:        "飞书设计文档",
		Kind:        "document",
		URL:         "https://lark.example.com/doc/abc123",
		Description: "Gloop v2 架构方案",
		Source:      "mage_review",
	}
	art, err := qs.AddDeclaredOutput("qst_decl_test", decl)
	if err != nil {
		t.Fatalf("AddDeclaredOutput failed: %v", err)
	}
	if art.ID == "" {
		t.Error("declared output ID should not be empty")
	}
	if art.Kind != "document" {
		t.Errorf("Kind = %q, want 'document'", art.Kind)
	}
	if art.StoragePath != decl.URL {
		t.Errorf("StoragePath = %q, want %q", art.StoragePath, decl.URL)
	}
	if art.Source != "mage_review" {
		t.Errorf("Source = %q, want 'mage_review'", art.Source)
	}

	// Deduplication: add the same should return existing
	art2, err := qs.AddDeclaredOutput("qst_decl_test", decl)
	if err != nil {
		t.Fatalf("AddDeclaredOutput (dedup) failed: %v", err)
	}
	if art2.ID != art.ID {
		t.Errorf("dedup returned different ID: %q vs %q", art2.ID, art.ID)
	}
	loaded, _ := qs.LoadQuest("qst_decl_test")
	if len(loaded.Outputs) != 1 {
		t.Errorf("Outputs count = %d, want 1 (dedup)", len(loaded.Outputs))
	}

	// Open external link should fail
	_, _, err = qs.OpenArtifact("qst_decl_test", art.ID)
	if err == nil {
		t.Error("OpenArtifact for external link should fail")
	}

	// Validation: empty qid
	if _, err := qs.AddDeclaredOutput("", decl); err == nil {
		t.Error("empty qid should error")
	}
	// Validation: empty name
	if _, err := qs.AddDeclaredOutput("qst_decl_test", DeclaredOutput{}); err == nil {
		t.Error("empty name should error")
	}
}

func TestClearOutputs(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	qs := NewQuestStore(root)
	q := &QuestMeta{ID: "qst_clear_test", Status: "running"}
	if err := qs.CreateQuest(q); err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}

	decl1 := DeclaredOutput{Name: "doc1", Kind: "document", URL: "u1", Source: "warrior"}
	decl2 := DeclaredOutput{Name: "doc2", Kind: "link", URL: "u2", Source: "mage_review"}
	if _, err := qs.AddDeclaredOutput("qst_clear_test", decl1); err != nil {
		t.Fatal(err)
	}
	if _, err := qs.AddDeclaredOutput("qst_clear_test", decl2); err != nil {
		t.Fatal(err)
	}

	// Clear only warrior source
	if err := qs.ClearOutputs("qst_clear_test", "warrior"); err != nil {
		t.Fatal(err)
	}
	loaded, _ := qs.LoadQuest("qst_clear_test")
	if len(loaded.Outputs) != 1 {
		t.Fatalf("after clear warrior: outputs = %d, want 1", len(loaded.Outputs))
	}
	if loaded.Outputs[0].Source != "mage_review" {
		t.Errorf("remaining source = %q, want 'mage_review'", loaded.Outputs[0].Source)
	}

	// Clear all
	if err := qs.ClearOutputs("qst_clear_test", ""); err != nil {
		t.Fatal(err)
	}
	loaded, _ = qs.LoadQuest("qst_clear_test")
	if len(loaded.Outputs) != 0 {
		t.Errorf("after clear all: outputs = %d, want 0", len(loaded.Outputs))
	}
}

func TestSaveOutputArtifact(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	qs := NewQuestStore(root)
	q := &QuestMeta{ID: "qst_out_art", Status: "running"}
	if err := qs.CreateQuest(q); err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}

	_, err = qs.SaveOutputArtifact("qst_out_art", ArtifactInput{
		Name:   "report.md",
		Reader: strings.NewReader("# report\n"),
		Source: "warrior",
	})
	if err != nil {
		t.Fatalf("SaveOutputArtifact failed: %v", err)
	}
	loaded, _ := qs.LoadQuest("qst_out_art")
	if len(loaded.Outputs) != 1 {
		t.Fatalf("outputs count = %d, want 1", len(loaded.Outputs))
	}
	if loaded.Outputs[0].Name != "report.md" {
		t.Errorf("name = %q, want report.md", loaded.Outputs[0].Name)
	}
	// Artifact file should exist on disk
	artPath := qs.ArtifactPath("qst_out_art", loaded.Outputs[0])
	if _, err := os.Stat(artPath); err != nil {
		t.Errorf("artifact file should exist at %q: %v", artPath, err)
	}
	// Empty StoragePath returns ""
	if p := qs.ArtifactPath("qst_out_art", QuestArtifact{}); p != "" {
		t.Errorf("empty StoragePath should return '', got %q", p)
	}
}

func TestArtifactPathAbsolute(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	qs := NewQuestStore(root)

	// Relative storage path: should be joined with quest dir
	rel := QuestArtifact{StoragePath: "artifacts/abc.txt"}
	got := qs.ArtifactPath("q1", rel)
	want := filepath.Join(root.Sub(SubdirQuests, "q1"), "artifacts", "abc.txt")
	if got != want {
		t.Errorf("relative: got %q, want %q", got, want)
	}

	// Absolute storage path: should be returned as-is
	absPath := filepath.Join(t.TempDir(), "outside", "x.txt")
	absArt := QuestArtifact{StoragePath: filepath.ToSlash(absPath)}
	got = qs.ArtifactPath("q1", absArt)
	if got != absPath {
		t.Errorf("absolute: got %q, want %q", got, absPath)
	}
}
