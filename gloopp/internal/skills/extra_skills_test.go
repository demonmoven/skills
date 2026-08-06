package skills

import (
	"strings"
	"testing"
	"testing/fstest"

	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

func testRegistry(t *testing.T) *Registry {
	t.Helper()
	fsys := fstest.MapFS{
		"warrior-skill/SKILL.md": &fstest.MapFile{Data: []byte(`---
name: warrior-skill
version: "1.0"
description: Warrior only skill
metadata:
  class: warrior
---

Warrior skill body.
`)},
		"shared-skill/SKILL.md": &fstest.MapFile{Data: []byte(`---
name: shared-skill
version: "1.0"
description: Shared skill for both classes
metadata:
  class: both
---

Shared skill body.
`)},
		"mage-only/SKILL.md": &fstest.MapFile{Data: []byte(`---
name: mage-only
version: "1.0"
description: Mage only skill
metadata:
  class: mage
---

Mage only body.
`)},
	}
	reg, err := Load(fsys)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	return reg
}

func TestBuildIndexLine_WarriorOnly(t *testing.T) {
	reg := testRegistry(t)
	line := reg.BuildIndexLine(model.ClassWarrior)
	if !strings.Contains(line, "warrior-skill") {
		t.Fatal("warrior should see warrior-skill")
	}
	if !strings.Contains(line, "shared-skill") {
		t.Fatal("warrior should see shared-skill")
	}
	if strings.Contains(line, "mage-only") {
		t.Fatal("warrior should NOT see mage-only")
	}
}

func TestBuildIndexLineWithExtras_AddsMageSkillToWarrior(t *testing.T) {
	reg := testRegistry(t)
	line := reg.BuildIndexLineWithExtras(model.ClassWarrior, []string{"mage-only"})
	if !strings.Contains(line, "warrior-skill") {
		t.Fatal("should still see default warrior skills")
	}
	if !strings.Contains(line, "mage-only") {
		t.Fatal("should see extra mage-only skill")
	}
	if !strings.Contains(line, "[自定义]") {
		t.Fatal("extra skills should be marked as custom")
	}
	// 验证 mage-only 出现在 [自定义] 标记后
	if strings.Contains(line, "`mage-only`: ") && !strings.Contains(line, "`mage-only` [自定义]: ") {
		t.Fatal("extra skill should have [自定义] marker")
	}
}

func TestBuildIndexLineWithExtras_Dedups(t *testing.T) {
	reg := testRegistry(t)
	// warrior-skill 本来就是战士的，加在 extra 里不应该重复
	line := reg.BuildIndexLineWithExtras(model.ClassWarrior, []string{"warrior-skill"})
	count := strings.Count(line, "warrior-skill")
	if count > 1 {
		t.Fatalf("warrior-skill should appear at most once, got %d", count)
	}
}

func TestBuildIndexLineWithExtras_NonexistentIgnored(t *testing.T) {
	reg := testRegistry(t)
	line := reg.BuildIndexLineWithExtras(model.ClassWarrior, []string{"nonexistent-skill"})
	if strings.Contains(line, "nonexistent") {
		t.Fatal("nonexistent skills should be silently ignored")
	}
	// 正常技能仍然显示
	if !strings.Contains(line, "warrior-skill") {
		t.Fatal("default skills still present")
	}
}

func TestBuildIndexLineWithExtras_EmptyExtrasSameAsDefault(t *testing.T) {
	reg := testRegistry(t)
	def := reg.BuildIndexLine(model.ClassWarrior)
	withEmpty := reg.BuildIndexLineWithExtras(model.ClassWarrior, nil)
	if def != withEmpty {
		t.Fatalf("empty extras should produce same result as BuildIndexLine\n--- default ---\n%s\n--- withEmpty ---\n%s", def, withEmpty)
	}
}
