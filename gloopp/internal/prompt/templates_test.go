package prompt

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestTemplateStore_Version_BaseOnly(t *testing.T) {
	base := fstest.MapFS{
		"warrior_role.md": &fstest.MapFile{Data: []byte("warrior role text")},
		"mage_role.md":    &fstest.MapFile{Data: []byte("mage role text")},
	}
	store := NewTemplateStore(base, nil)
	ver := store.Version()

	if ver.BaseHash == "" {
		t.Fatal("base hash should not be empty")
	}
	if len(ver.Overrides) != 0 {
		t.Fatalf("no overrides expected, got %v", ver.Overrides)
	}
	short := ver.Short()
	if !strings.HasPrefix(short, "base-") {
		t.Fatalf("short should start with base-, got %s", short)
	}
}

func TestTemplateStore_Version_WithOverlay(t *testing.T) {
	base := fstest.MapFS{
		"warrior_role.md": &fstest.MapFile{Data: []byte("base warrior")},
		"mage_role.md":    &fstest.MapFile{Data: []byte("base mage")},
	}
	overlay := fstest.MapFS{
		"warrior_role.md": &fstest.MapFile{Data: []byte("custom warrior")},
		"extra_thing.md":  &fstest.MapFile{Data: []byte("not in base = not counted as override")},
	}
	store := NewTemplateStore(base, overlay)
	ver := store.Version()

	if len(ver.Overrides) != 1 {
		t.Fatalf("expected 1 override, got %d: %v", len(ver.Overrides), ver.Overrides)
	}
	if ver.Overrides[0] != "warrior_role.md" {
		t.Fatalf("expected warrior_role.md override, got %s", ver.Overrides[0])
	}
	if len(ver.OverrideHashes) != 1 {
		t.Fatalf("expected 1 override hash, got %d", len(ver.OverrideHashes))
	}
	short := ver.Short()
	if !strings.Contains(short, "+1overrides") {
		t.Fatalf("short should mention 1 override, got %s", short)
	}
}

func TestTemplateStore_Version_Consistent(t *testing.T) {
	base := fstest.MapFS{
		"a.md": &fstest.MapFile{Data: []byte("content a")},
		"b.md": &fstest.MapFile{Data: []byte("content b")},
	}
	s1 := NewTemplateStore(base, nil)
	s2 := NewTemplateStore(base, nil)
	v1 := s1.Version()
	v2 := s2.Version()
	if v1.BaseHash != v2.BaseHash {
		t.Fatalf("hash not consistent: %s vs %s", v1.BaseHash, v2.BaseHash)
	}
	// 调用两次也应相同（缓存生效）
	v1b := s1.Version()
	if v1.BaseHash != v1b.BaseHash {
		t.Fatalf("cached hash changed")
	}
}

func TestTemplateStore_IsOverridden(t *testing.T) {
	base := fstest.MapFS{
		"warrior_role.md": &fstest.MapFile{Data: []byte("base")},
	}
	overlay := fstest.MapFS{
		"warrior_role.md": &fstest.MapFile{Data: []byte("custom")},
	}
	store := NewTemplateStore(base, overlay)
	if !store.IsOverridden("warrior_role.md") {
		t.Fatal("warrior_role.md should be overridden")
	}
	if store.IsOverridden("nonexistent.md") {
		t.Fatal("nonexistent.md should not be overridden")
	}
}
