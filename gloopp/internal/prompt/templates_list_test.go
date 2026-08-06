package prompt

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestTemplateStore_ListTemplates(t *testing.T) {
	base := fstest.MapFS{
		"warrior_role.md": &fstest.MapFile{Data: []byte("Warrior role description. Long text for preview testing.")},
		"mage_role.md":    &fstest.MapFile{Data: []byte("Mage role description.")},
	}
	overlay := fstest.MapFS{
		"warrior_role.md": &fstest.MapFile{Data: []byte("Custom warrior role.")},
	}
	store := NewTemplateStore(base, overlay)
	list, err := store.ListTemplates()
	if err != nil {
		t.Fatalf("ListTemplates failed: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 templates, got %d", len(list))
	}
	// 字典序：mage_role < warrior_role
	if list[0].Name != "mage_role.md" {
		t.Fatalf("first should be mage_role.md, got %s", list[0].Name)
	}
	if list[0].Source != "builtin" {
		t.Fatalf("mage_role should be builtin, got %s", list[0].Source)
	}
	if list[1].Name != "warrior_role.md" {
		t.Fatalf("second should be warrior_role.md, got %s", list[1].Name)
	}
	if list[1].Source != "user" {
		t.Fatalf("warrior_role should be user-overridden, got %s", list[1].Source)
	}
	if !list[1].IsOverridden {
		t.Fatal("warrior_role should be marked as overridden")
	}
	if list[0].IsOverridden {
		t.Fatal("mage_role should NOT be marked as overridden")
	}
	if !strings.Contains(list[0].Preview, "Mage role") {
		t.Fatalf("preview should contain content, got %q", list[0].Preview)
	}
}

func TestTemplateStore_GetTemplate(t *testing.T) {
	base := fstest.MapFS{
		"warrior_role.md": &fstest.MapFile{Data: []byte("base warrior")},
	}
	overlay := fstest.MapFS{
		"warrior_role.md": &fstest.MapFile{Data: []byte("custom warrior")},
	}
	store := NewTemplateStore(base, overlay)
	content, info, err := store.GetTemplate("warrior_role.md")
	if err != nil {
		t.Fatalf("GetTemplate failed: %v", err)
	}
	if !strings.Contains(content, "custom warrior") {
		t.Fatalf("should return custom content, got %q", content)
	}
	if info.Source != "user" {
		t.Fatalf("source should be user, got %s", info.Source)
	}
	if !info.IsOverridden {
		t.Fatal("should be overridden")
	}
}

func TestTemplateStore_GetTemplate_NotFound(t *testing.T) {
	base := fstest.MapFS{}
	store := NewTemplateStore(base, nil)
	_, _, err := store.GetTemplate("nonexistent.md")
	if err == nil {
		t.Fatal("expected error for nonexistent template")
	}
}
