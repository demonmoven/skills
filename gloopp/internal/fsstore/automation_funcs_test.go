package fsstore

import (
	"testing"
)

func TestIsOfficial(t *testing.T) {
	c := &AutomationConfig{Source: SourceOfficial}
	if !c.IsOfficial() {
		t.Error("SourceOfficial should be official")
	}
	c2 := &AutomationConfig{Source: SourceUser}
	if c2.IsOfficial() {
		t.Error("SourceUser should not be official")
	}
	c3 := &AutomationConfig{Source: "unknown"}
	if c3.IsOfficial() {
		t.Error("unknown source should not be official")
	}
	c4 := &AutomationConfig{}
	if c4.IsOfficial() {
		t.Error("empty source should not be official")
	}
}

func TestHasTag(t *testing.T) {
	c := &AutomationConfig{Tags: []string{"official", "template", "context"}}
	cases := map[string]bool{
		"official": true,
		"template": true,
		"context":  true,
		"foo":      false,
		"":         false,
		"OFFICIAL": false,
	}
	for tag, want := range cases {
		if got := c.HasTag(tag); got != want {
			t.Errorf("HasTag(%q)=%v, want %v", tag, got, want)
		}
	}
	// nil tags
	c2 := &AutomationConfig{}
	if c2.HasTag("official") {
		t.Error("nil tags HasTag should be false")
	}
}

func TestNormalizeCronPreset(t *testing.T) {
	cases := map[string]string{
		"daily":         "0 0 * * *",
		"daily9":        "0 9 * * *",
		"weekday":       "0 9 * * 1-5",
		"weekly":        "0 0 * * 1",
		"":              "",
		"0 5 * * *":     "0 5 * * *",
		"unknown_preset": "unknown_preset",
	}
	for in, want := range cases {
		if got := NormalizeCronPreset(in); got != want {
			t.Errorf("NormalizeCronPreset(%q)=%q, want %q", in, got, want)
		}
	}
}

func TestNormalizeAutomationAfterLoad(t *testing.T) {
	// source 为空：tags 有 official → SourceOfficial
	c := &AutomationConfig{Tags: []string{"official"}}
	normalizeAutomationAfterLoad(c)
	if c.Source != SourceOfficial {
		t.Errorf("after load, tags has official -> source=%s, want %s", c.Source, SourceOfficial)
	}
	// source 为空：tags 无 official → SourceUser
	c2 := &AutomationConfig{Tags: []string{"template"}}
	normalizeAutomationAfterLoad(c2)
	if c2.Source != SourceUser {
		t.Errorf("after load, no official tag -> source=%s, want %s", c2.Source, SourceUser)
	}
	// source 非空：保持
	c3 := &AutomationConfig{Source: SourceUser, Tags: []string{"official"}}
	normalizeAutomationAfterLoad(c3)
	if c3.Source != SourceUser {
		t.Errorf("after load, existing user source should be kept")
	}
}

func TestNormalizeAutomationBeforeSave(t *testing.T) {
	// source 空：有 official tag → official，并同步 tags
	c := &AutomationConfig{Tags: []string{"official"}}
	normalizeAutomationBeforeSave(c)
	if c.Source != SourceOfficial {
		t.Errorf("before save, tags official -> source=%s", c.Source)
	}
	// source 空：无 official tag → user
	c2 := &AutomationConfig{}
	normalizeAutomationBeforeSave(c2)
	if c2.Source != SourceUser {
		t.Errorf("before save, empty -> source=%s, want %s", c2.Source, SourceUser)
	}
	// source 设为 official，tags 应自动补齐 official
	c3 := &AutomationConfig{Source: SourceOfficial, Tags: []string{"template"}}
	normalizeAutomationBeforeSave(c3)
	found := false
	for _, tg := range c3.Tags {
		if tg == "official" {
			found = true
		}
	}
	if !found {
		t.Errorf("before save, source=official should sync tags, got %+v", c3.Tags)
	}
	// source 已是 official 且 tags 已有 official，不重复
	c4 := &AutomationConfig{Source: SourceOfficial, Tags: []string{"official"}}
	normalizeAutomationBeforeSave(c4)
	count := 0
	for _, tg := range c4.Tags {
		if tg == "official" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("before save, official tag duplicated: count=%d, tags=%+v", count, c4.Tags)
	}
}
