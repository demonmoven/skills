package fsstore

import "testing"

func TestDefaultCommandAllowlist(t *testing.T) {
	items := DefaultCommandAllowlist()
	if _, ok := FindAllowedCommand(items, "go-test"); !ok {
		t.Fatal("default allowlist should include go-test")
	}
	if _, ok := FindAllowedCommand(items, "npm-test"); !ok {
		t.Fatal("default allowlist should include npm-test")
	}
	if _, ok := FindAllowedCommand(items, "bytedcli"); ok {
		t.Fatal("default allowlist must not include bytedcli")
	}
	if err := ValidateAllowedCommands(items); err != nil {
		t.Fatalf("default allowlist should validate: %v", err)
	}
}

func TestValidateAllowedCommandsRejectsPathCommand(t *testing.T) {
	err := ValidateAllowedCommands([]AllowedCommand{{
		ID:      "bad",
		Command: "/bin/sh",
	}})
	if err == nil {
		t.Fatal("path command should be rejected")
	}
}

func TestValidateAllowedCommandsRejectsInvalidSideEffectLevel(t *testing.T) {
	err := ValidateAllowedCommands([]AllowedCommand{{
		ID:              "bad-side-effect",
		Command:         "go",
		SideEffectLevel: "L9",
	}})
	if err == nil {
		t.Fatal("expected invalid side_effect_level to be rejected")
	}
}
