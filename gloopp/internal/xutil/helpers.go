// Package xutil provides low-level, cross-cutting utility helpers shared
// across the gloop internal packages. It is intentionally a leaf package:
// xutil must never import any other gloop internal package to avoid cycles.
package xutil

import (
	"encoding/json"
	"fmt"
	"unicode/utf8"
)

// Truncate truncates s to at most n runes, appending a single-character
// ellipsis "…" when truncation occurs. It is safe for CJK and emoji.
// If n <= 0 the result is always "".
func Truncate(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	r := []rune(s)
	return string(r[:n]) + "…"
}

// TruncateASCII is like Truncate but uses three ASCII dots "..." instead of
// the Unicode ellipsis. Prefer this when the target renderer (e.g. Lark
// cards) may not render Unicode punctuation well.
func TruncateASCII(s string, n int) string {
	if n <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "..."
}

// ToJSON marshals v to a JSON string. On marshal failure it falls back to
// fmt's %+v representation so the helper always returns something printable.
func ToJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%+v", v)
	}
	return string(b)
}

// Int64FromAny safely extracts an int64 from a loosely-typed value (commonly
// found in map[string]any payloads decoded from JSON). Types accepted:
// int, int32, int64, float32, float64, and anything implementing an
// Int64() (int64, error) method (covers json.Number). Unrecognised types
// yield 0.
func Int64FromAny(v any) int64 {
	switch x := v.(type) {
	case int:
		return int64(x)
	case int64:
		return x
	case int32:
		return int64(x)
	case float64:
		return int64(x)
	case float32:
		return int64(x)
	case jsonNumber:
		n, _ := x.Int64()
		return n
	default:
		return 0
	}
}

// jsonNumber abstracts json.Number without forcing an import of encoding/json
// at the call site for every package that uses Int64FromAny.
type jsonNumber interface {
	Int64() (int64, error)
}

// StringFromAny safely extracts a string from a loosely-typed value. It
// accepts plain strings and anything implementing fmt.Stringer. Other types
// yield the empty string.
func StringFromAny(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case fmt.Stringer:
		return x.String()
	default:
		return ""
	}
}

// StringSliceFromAny converts a loosely-typed slice to a []string. It handles
// both native []string and heterogeneous []any (picking only the string
// entries). Other input types yield nil.
func StringSliceFromAny(v any) []string {
	switch x := v.(type) {
	case []string:
		return append([]string(nil), x...)
	case []any:
		out := make([]string, 0, len(x))
		for _, item := range x {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}
