package skills

import (
	"strings"
	"testing"
	"testing/fstest"
)

// ==================== Skill.Validate tests ====================

func TestSkill_Validate_Valid(t *testing.T) {
	sk := Skill{
		Name:        "test-skill",
		Version:     "1.0.0",
		Description: "A test skill",
		Kind:        KindCapability,
		Body: `# test-skill

## Trigger Examples

example

## CLI Contract

noop

## Discipline

none
`,
	}
	issues := sk.Validate()
	if len(issues) > 0 {
		t.Errorf("expected valid skill, got issues: %+v", issues)
	}
}

func TestSkill_Validate_ValidEmptyKind(t *testing.T) {
	// Empty kind is allowed for backward compatibility
	sk := Skill{
		Name:        "test-skill",
		Version:     "1.0.0",
		Description: "A test skill",
		Body: `# test-skill

## Trigger Examples

ex

## CLI Contract

no

## Discipline

nada
`,
	}
	issues := sk.Validate()
	if len(issues) > 0 {
		t.Errorf("empty kind should be valid (backward compat), got issues: %+v", issues)
	}
}

func TestSkill_Validate_OrchestrationKind(t *testing.T) {
	sk := Skill{
		Name:        "test-skill",
		Version:     "1.0.0",
		Description: "A test skill",
		Kind:        KindOrchestration,
		Body: `# test-skill

## Trigger Examples

ex

## CLI Contract

no

## Discipline

nada
`,
	}
	issues := sk.Validate()
	if len(issues) > 0 {
		t.Errorf("orchestration kind should be valid, got issues: %+v", issues)
	}
}

func TestSkill_Validate_EmptyName(t *testing.T) {
	sk := Skill{
		Name:        "",
		Version:     "1.0.0",
		Description: "desc",
		Body:        "## Trigger Examples\n\n## CLI Contract\n\n## Discipline\n",
	}
	issues := sk.Validate()
	found := false
	for _, iss := range issues {
		if iss.Field == "name" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected name validation issue, got: %+v", issues)
	}
}

func TestSkill_Validate_EmptyVersion(t *testing.T) {
	sk := Skill{
		Name:        "foo",
		Version:     "",
		Description: "desc",
		Body:        "# foo\n\n## Trigger Examples\n\n## CLI Contract\n\n## Discipline\n",
	}
	issues := sk.Validate()
	found := false
	for _, iss := range issues {
		if iss.Field == "version" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected version validation issue, got: %+v", issues)
	}
}

func TestSkill_Validate_InvalidVersion(t *testing.T) {
	sk := Skill{
		Name:        "foo",
		Version:     "not-a-version",
		Description: "desc",
		Body:        "# foo\n\n## Trigger Examples\n\n## CLI Contract\n\n## Discipline\n",
	}
	issues := sk.Validate()
	found := false
	for _, iss := range issues {
		if iss.Field == "version" {
			found = true
			if !strings.Contains(iss.Message, "invalid") {
				t.Errorf("version issue should mention 'invalid', got: %s", iss.Message)
			}
			break
		}
	}
	if !found {
		t.Errorf("expected version validation issue, got: %+v", issues)
	}
}

func TestSkill_Validate_EmptyDescription(t *testing.T) {
	sk := Skill{
		Name:        "foo",
		Version:     "1.0.0",
		Description: "",
		Body:        "# foo\n\n## Trigger Examples\n\n## CLI Contract\n\n## Discipline\n",
	}
	issues := sk.Validate()
	found := false
	for _, iss := range issues {
		if iss.Field == "description" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected description validation issue, got: %+v", issues)
	}
}

func TestSkill_Validate_InvalidKind(t *testing.T) {
	sk := Skill{
		Name:        "foo",
		Version:     "1.0.0",
		Description: "desc",
		Kind:        "bogus",
		Body:        "# foo\n\n## Trigger Examples\n\n## CLI Contract\n\n## Discipline\n",
	}
	issues := sk.Validate()
	found := false
	for _, iss := range issues {
		if iss.Field == "kind" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected kind validation issue, got: %+v", issues)
	}
}

func TestSkill_Validate_MissingH1(t *testing.T) {
	sk := Skill{
		Name:        "foo",
		Version:     "1.0.0",
		Description: "desc",
		Body:        "## Trigger Examples\n\n## CLI Contract\n\n## Discipline\n",
	}
	issues := sk.Validate()
	found := false
	for _, iss := range issues {
		if iss.Field == "body.h1" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected body.h1 validation issue, got: %+v", issues)
	}
}

func TestSkill_Validate_MissingSections(t *testing.T) {
	sk := Skill{
		Name:        "foo",
		Version:     "1.0.0",
		Description: "desc",
		Body:        "# foo\n\nSome content but no sections.",
	}
	issues := sk.Validate()
	sectionIssues := 0
	for _, iss := range issues {
		if iss.Field == "body.sections" {
			sectionIssues++
		}
	}
	if sectionIssues < 3 {
		t.Errorf("expected at least 3 body section issues, got %d: %+v", sectionIssues, issues)
	}
}

func TestSkill_IsValid(t *testing.T) {
	valid := Skill{
		Name:        "foo",
		Version:     "1.0.0",
		Description: "d",
		Body:        "# foo\n\n## Trigger Examples\n\n## CLI Contract\n\n## Discipline\n",
	}
	if !valid.IsValid() {
		t.Error("valid skill should return IsValid() = true")
	}

	invalid := Skill{Name: ""}
	if invalid.IsValid() {
		t.Error("invalid skill should return IsValid() = false")
	}
}

func TestValidationIssue_String(t *testing.T) {
	iss := ValidationIssue{Field: "version", Message: "is bad"}
	s := iss.String()
	if !strings.Contains(s, "version") || !strings.Contains(s, "is bad") {
		t.Errorf("unexpected issue string: %q", s)
	}
}

// ==================== Skill.Semver tests ====================

func TestSkill_Semver(t *testing.T) {
	sk := Skill{Name: "foo", Version: "1.2.3"}
	v, err := sk.Semver()
	if err != nil {
		t.Fatalf("Semver() error: %v", err)
	}
	if v.Major != 1 || v.Minor != 2 || v.Patch != 3 {
		t.Errorf("Semver() = %v, want 1.2.3", v)
	}
}

func TestSkill_Semver_Invalid(t *testing.T) {
	sk := Skill{Name: "foo", Version: "nope"}
	_, err := sk.Semver()
	if err == nil {
		t.Error("expected error for invalid version")
	}
}

// ==================== Registry.ValidateAll tests ====================

func TestRegistry_ValidateAll_Builtin(t *testing.T) {
	// All built-in skills should pass validation
	reg := Default()
	issues := reg.ValidateAll()
	if issues != nil {
		for name, iss := range issues {
			t.Errorf("%s has validation issues:", name)
			for _, i := range iss {
				t.Errorf("  - %s", i.String())
			}
		}
	}
}

func TestRegistry_ValidateAll(t *testing.T) {
	fsys := fstest.MapFS{
		"good-skill/SKILL.md": {
			Data: []byte(`---
name: good-skill
version: 1.0.0
description: a good skill
metadata:
  class: both
  category: test
  kind: capability
---
# good-skill

## Trigger Examples

ex

## CLI Contract

no

## Discipline

nada
`),
		},
		"bad-skill/SKILL.md": {
			Data: []byte(`---
name: bad-skill
version: 1
description:
metadata:
  class: both
  kind: bogus
---
bad body with no sections
`),
		},
	}
	reg, err := Load(fsys)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	issues := reg.ValidateAll()
	if issues == nil {
		t.Fatal("expected validation issues for bad-skill")
	}
	if _, ok := issues["bad-skill"]; !ok {
		t.Error("bad-skill should have issues")
	}
	if _, ok := issues["good-skill"]; ok {
		t.Error("good-skill should have no issues")
	}
}
