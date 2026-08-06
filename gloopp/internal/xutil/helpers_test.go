package xutil

import (
	"testing"
)

func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		n        int
		expected string
	}{
		{"empty", "", 10, ""},
		{"zero_limit", "hello", 0, ""},
		{"negative_limit", "hello", -1, ""},
		{"no_truncation_ascii", "hello", 10, "hello"},
		{"exact_boundary_ascii", "hello", 5, "hello"},
		{"truncate_ascii", "hello world", 5, "hello…"},
		{"no_truncation_cjk", "你好世界", 4, "你好世界"},
		{"truncate_cjk", "你好世界测试", 3, "你好世…"},
		{"mixed_ascii_cjk", "hello你好世界", 7, "hello你好…"},
		{"emoji", "👋🌍abc", 3, "👋🌍a…"},
		{"emoji_exact", "👋🌍", 2, "👋🌍"},
		{"emoji_truncated", "👋🌍abc", 4, "👋🌍ab…"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Truncate(tt.input, tt.n)
			if got != tt.expected {
				t.Errorf("Truncate(%q, %d) = %q, want %q", tt.input, tt.n, got, tt.expected)
			}
		})
	}
}

func TestTruncateASCII(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		n        int
		expected string
	}{
		{"empty", "", 10, ""},
		{"zero_limit", "hello", 0, ""},
		{"truncate_uses_three_dots", "hello world", 5, "hello..."},
		{"no_truncation", "你好世界", 4, "你好世界"},
		{"truncate_cjk", "你好世界测试", 3, "你好世..."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TruncateASCII(tt.input, tt.n)
			if got != tt.expected {
				t.Errorf("TruncateASCII(%q, %d) = %q, want %q", tt.input, tt.n, got, tt.expected)
			}
		})
	}
}

func TestToJSON(t *testing.T) {
	// Struct
	type point struct{ X, Y int }
	got := ToJSON(point{1, 2})
	if got != `{"X":1,"Y":2}` {
		t.Errorf("ToJSON(point) = %q, want JSON object", got)
	}
	// String
	if s := ToJSON("hello"); s != `"hello"` {
		t.Errorf("ToJSON(string) = %q", s)
	}
	// Nil
	if s := ToJSON(nil); s != "null" {
		t.Errorf("ToJSON(nil) = %q, want null", s)
	}
}

func TestInt64FromAny(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected int64
	}{
		{"int", int(42), 42},
		{"int64", int64(42), 42},
		{"int32", int32(42), 42},
		{"float64", float64(42.7), 42},
		{"float32", float32(42.3), 42},
		{"zero_float", float64(0), 0},
		{"string_unhandled", "42", 0},
		{"nil", nil, 0},
		{"bool_unhandled", true, 0},
		{"negative", int(-5), -5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Int64FromAny(tt.input)
			if got != tt.expected {
				t.Errorf("Int64FromAny(%v) = %d, want %d", tt.input, got, tt.expected)
			}
		})
	}
}

func TestStringFromAny(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{"plain_string", "hello", "hello"},
		{"empty_string", "", ""},
		{"stringer", customStringer{"x"}, "custom:x"},
		{"int_unhandled", 42, ""},
		{"nil", nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StringFromAny(tt.input)
			if got != tt.expected {
				t.Errorf("StringFromAny(%v) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestStringSliceFromAny(t *testing.T) {
	// []string
	in1 := []string{"a", "b", "c"}
	got1 := StringSliceFromAny(in1)
	if len(got1) != 3 || got1[0] != "a" || got1[2] != "c" {
		t.Errorf("StringSliceFromAny([]string) = %v", got1)
	}
	// Mutating the returned slice must not affect input (copy)
	got1[0] = "X"
	if in1[0] != "a" {
		t.Error("StringSliceFromAny did not copy []string input")
	}
	// []any with mixed
	got2 := StringSliceFromAny([]any{"x", 42, "y", nil})
	if len(got2) != 2 || got2[0] != "x" || got2[1] != "y" {
		t.Errorf("StringSliceFromAny([]any) = %v, want [x y]", got2)
	}
	// nil
	if StringSliceFromAny(nil) != nil {
		t.Error("nil input should return nil")
	}
	// int
	if StringSliceFromAny(42) != nil {
		t.Error("non-slice should return nil")
	}
}

type customStringer struct{ v string }

func (c customStringer) String() string { return "custom:" + c.v }
