package skills

import (
	"testing"
)

// ==================== ParseVersion tests ====================

func TestParseVersion_Valid(t *testing.T) {
	cases := []struct {
		in   string
		want Version
	}{
		{"1.0.0", Version{1, 0, 0}},
		{"0.1.0", Version{0, 1, 0}},
		{"0.0.1", Version{0, 0, 1}},
		{"1.2.3", Version{1, 2, 3}},
		{"10.20.30", Version{10, 20, 30}},
		{"v1.2.3", Version{1, 2, 3}},    // leading v
		{"  2.3.4  ", Version{2, 3, 4}}, // whitespace
		{"0.0.0", Version{0, 0, 0}},     // zero version
	}
	for _, c := range cases {
		got, err := ParseVersion(c.in)
		if err != nil {
			t.Errorf("ParseVersion(%q) unexpected error: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("ParseVersion(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestParseVersion_Invalid(t *testing.T) {
	cases := []string{
		"",
		"1",
		"1.0",
		"1.0.0.0",
		"a.b.c",
		"1.b.3",
		"1.2.c",
		"-1.0.0",
		"1.-2.3",
		"01.2.3", // leading zero
		"1.02.3", // leading zero
		"1.2.03", // leading zero
		"v",
		"v1.0",
		"1.0.0-beta", // pre-release not supported
	}
	for _, s := range cases {
		_, err := ParseVersion(s)
		if err == nil {
			t.Errorf("ParseVersion(%q) expected error but got nil", s)
		}
	}
}

// ==================== String tests ====================

func TestVersion_String(t *testing.T) {
	cases := []struct {
		v    Version
		want string
	}{
		{Version{1, 0, 0}, "1.0.0"},
		{Version{0, 1, 0}, "0.1.0"},
		{Version{0, 0, 0}, "0.0.0"},
		{Version{10, 20, 30}, "10.20.30"},
	}
	for _, c := range cases {
		got := c.v.String()
		if got != c.want {
			t.Errorf("%v.String() = %q, want %q", c.v, got, c.want)
		}
	}
}

// ==================== IsZero / IsValid tests ====================

func TestVersion_IsZero(t *testing.T) {
	if !(Version{0, 0, 0}).IsZero() {
		t.Error("0.0.0 should be zero")
	}
	if (Version{0, 0, 1}).IsZero() {
		t.Error("0.0.1 should not be zero")
	}
	if (Version{0, 1, 0}).IsZero() {
		t.Error("0.1.0 should not be zero")
	}
	if (Version{1, 0, 0}).IsZero() {
		t.Error("1.0.0 should not be zero")
	}
}

func TestVersion_IsValid(t *testing.T) {
	if (Version{0, 0, 0}).IsValid() {
		t.Error("0.0.0 should not be valid")
	}
	if !(Version{0, 0, 1}).IsValid() {
		t.Error("0.0.1 should be valid")
	}
	if !(Version{1, 0, 0}).IsValid() {
		t.Error("1.0.0 should be valid")
	}
}

// ==================== Compare tests ====================

func TestVersion_Compare(t *testing.T) {
	cases := []struct {
		a, b Version
		want int
	}{
		// Equal
		{Version{1, 0, 0}, Version{1, 0, 0}, 0},
		{Version{0, 0, 0}, Version{0, 0, 0}, 0},
		{Version{2, 3, 4}, Version{2, 3, 4}, 0},

		// Less than — major
		{Version{1, 0, 0}, Version{2, 0, 0}, -1},
		{Version{0, 9, 9}, Version{1, 0, 0}, -1},

		// Less than — minor
		{Version{1, 0, 0}, Version{1, 1, 0}, -1},
		{Version{1, 2, 9}, Version{1, 3, 0}, -1},

		// Less than — patch
		{Version{1, 0, 0}, Version{1, 0, 1}, -1},
		{Version{1, 0, 5}, Version{1, 0, 10}, -1},

		// Greater than — major
		{Version{2, 0, 0}, Version{1, 0, 0}, 1},
		{Version{1, 0, 0}, Version{0, 9, 9}, 1},

		// Greater than — minor
		{Version{1, 1, 0}, Version{1, 0, 0}, 1},
		{Version{1, 3, 0}, Version{1, 2, 9}, 1},

		// Greater than — patch
		{Version{1, 0, 1}, Version{1, 0, 0}, 1},
		{Version{1, 0, 10}, Version{1, 0, 5}, 1},
	}
	for _, c := range cases {
		got := c.a.Compare(c.b)
		if got != c.want {
			t.Errorf("%v.Compare(%v) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

// ==================== Convenience comparison tests ====================

func TestVersion_LessThan(t *testing.T) {
	if !(Version{1, 0, 0}).LessThan(Version{2, 0, 0}) {
		t.Error("1.0.0 < 2.0.0 should be true")
	}
	if (Version{1, 0, 0}).LessThan(Version{1, 0, 0}) {
		t.Error("1.0.0 < 1.0.0 should be false")
	}
	if (Version{2, 0, 0}).LessThan(Version{1, 0, 0}) {
		t.Error("2.0.0 < 1.0.0 should be false")
	}
}

func TestVersion_LessThanOrEqual(t *testing.T) {
	if !(Version{1, 0, 0}).LessThanOrEqual(Version{2, 0, 0}) {
		t.Error("1.0.0 <= 2.0.0 should be true")
	}
	if !(Version{1, 0, 0}).LessThanOrEqual(Version{1, 0, 0}) {
		t.Error("1.0.0 <= 1.0.0 should be true")
	}
	if (Version{2, 0, 0}).LessThanOrEqual(Version{1, 0, 0}) {
		t.Error("2.0.0 <= 1.0.0 should be false")
	}
}

func TestVersion_GreaterThan(t *testing.T) {
	if (Version{1, 0, 0}).GreaterThan(Version{2, 0, 0}) {
		t.Error("1.0.0 > 2.0.0 should be false")
	}
	if (Version{1, 0, 0}).GreaterThan(Version{1, 0, 0}) {
		t.Error("1.0.0 > 1.0.0 should be false")
	}
	if !(Version{2, 0, 0}).GreaterThan(Version{1, 0, 0}) {
		t.Error("2.0.0 > 1.0.0 should be true")
	}
}

func TestVersion_GreaterThanOrEqual(t *testing.T) {
	if (Version{1, 0, 0}).GreaterThanOrEqual(Version{2, 0, 0}) {
		t.Error("1.0.0 >= 2.0.0 should be false")
	}
	if !(Version{1, 0, 0}).GreaterThanOrEqual(Version{1, 0, 0}) {
		t.Error("1.0.0 >= 1.0.0 should be true")
	}
	if !(Version{2, 0, 0}).GreaterThanOrEqual(Version{1, 0, 0}) {
		t.Error("2.0.0 >= 1.0.0 should be true")
	}
}

func TestVersion_Equal(t *testing.T) {
	if (Version{1, 0, 0}).Equal(Version{2, 0, 0}) {
		t.Error("1.0.0 == 2.0.0 should be false")
	}
	if !(Version{1, 0, 0}).Equal(Version{1, 0, 0}) {
		t.Error("1.0.0 == 1.0.0 should be true")
	}
}

// ==================== IsCompatible tests ====================

func TestVersion_IsCompatible(t *testing.T) {
	cases := []struct {
		a, b Version
		want bool
		desc string
	}{
		// Same major >= 1.0.0 → compatible
		{Version{1, 0, 0}, Version{1, 0, 0}, true, "same version 1.0.0"},
		{Version{1, 0, 0}, Version{1, 1, 0}, true, "same major, minor diff"},
		{Version{1, 2, 3}, Version{1, 5, 0}, true, "same major, minor+patch diff"},
		{Version{1, 9, 9}, Version{1, 0, 0}, true, "same major, backwards check"},

		// Different major → incompatible
		{Version{1, 0, 0}, Version{2, 0, 0}, false, "different major"},
		{Version{2, 0, 0}, Version{1, 0, 0}, false, "different major (reverse)"},
		{Version{0, 1, 0}, Version{1, 0, 0}, false, "0.x vs 1.x"},

		// 0.x: same minor → compatible
		{Version{0, 1, 0}, Version{0, 1, 0}, true, "0.x same version"},
		{Version{0, 1, 0}, Version{0, 1, 5}, true, "0.x same minor, patch diff"},
		{Version{0, 1, 9}, Version{0, 1, 0}, true, "0.x same minor, reverse"},

		// 0.x: different minor → incompatible
		{Version{0, 1, 0}, Version{0, 2, 0}, false, "0.x different minor"},
		{Version{0, 2, 0}, Version{0, 1, 0}, false, "0.x different minor (reverse)"},
		{Version{0, 0, 1}, Version{0, 1, 0}, false, "0.x minor upgrade"},

		// Zero version — compatible with everything
		{Version{0, 0, 0}, Version{1, 0, 0}, true, "zero version vs 1.0.0"},
		{Version{1, 0, 0}, Version{0, 0, 0}, true, "1.0.0 vs zero version"},
		{Version{0, 0, 0}, Version{0, 0, 0}, true, "zero vs zero"},
	}
	for _, c := range cases {
		got := c.a.IsCompatible(c.b)
		if got != c.want {
			t.Errorf("%s: %v.IsCompatible(%v) = %v, want %v",
				c.desc, c.a, c.b, got, c.want)
		}
	}
}

// ==================== Bump tests ====================

func TestVersion_BumpMajor(t *testing.T) {
	v := Version{1, 2, 3}.BumpMajor()
	if v != (Version{2, 0, 0}) {
		t.Errorf("BumpMajor(1.2.3) = %v, want 2.0.0", v)
	}
}

func TestVersion_BumpMinor(t *testing.T) {
	v := Version{1, 2, 3}.BumpMinor()
	if v != (Version{1, 3, 0}) {
		t.Errorf("BumpMinor(1.2.3) = %v, want 1.3.0", v)
	}
}

func TestVersion_BumpPatch(t *testing.T) {
	v := Version{1, 2, 3}.BumpPatch()
	if v != (Version{1, 2, 4}) {
		t.Errorf("BumpPatch(1.2.3) = %v, want 1.2.4", v)
	}
}
