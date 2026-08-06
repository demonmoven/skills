package skills

import (
	"fmt"
	"strconv"
	"strings"
)

// Version is a semantic version (MAJOR.MINOR.PATCH).
// It supports comparison and compatibility checking.
//
// Semantic versioning rules:
//   - MAJOR: breaking changes (contract changes that would break agents relying on the skill)
//   - MINOR: new capabilities, backward-compatible
//   - PATCH: documentation fixes, clarifications, no behavior change
//
// Special handling for 0.x versions (initial development):
//   - 0.MINOR.PATCH: MINOR changes may be breaking
//   - Two 0.x versions are only compatible if they share the same MINOR version
type Version struct {
	Major int
	Minor int
	Patch int
}

// ParseVersion parses a semver string like "1.2.3" into a Version struct.
// Returns an error if the string is not a valid semver.
func ParseVersion(s string) (Version, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Version{}, fmt.Errorf("version is empty")
	}

	// Strip leading "v" if present (common convention)
	s = strings.TrimPrefix(s, "v")

	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return Version{}, fmt.Errorf("invalid version %q: expected MAJOR.MINOR.PATCH", s)
	}

	var v Version
	var err error

	v.Major, err = parseVersionNumber(parts[0], "major")
	if err != nil {
		return Version{}, err
	}
	v.Minor, err = parseVersionNumber(parts[1], "minor")
	if err != nil {
		return Version{}, err
	}
	v.Patch, err = parseVersionNumber(parts[2], "patch")
	if err != nil {
		return Version{}, err
	}

	return v, nil
}

func parseVersionNumber(s string, name string) (int, error) {
	if s == "" {
		return 0, fmt.Errorf("%s version component is empty", name)
	}
	// Reject leading zeros (e.g. "01" is not valid) except for "0" itself
	if len(s) > 1 && s[0] == '0' {
		return 0, fmt.Errorf("%s version component has leading zero: %q", name, s)
	}
	// All characters must be digits
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return 0, fmt.Errorf("%s version component contains non-digit characters: %q", name, s)
		}
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("%s version component is not a valid integer: %q", name, s)
	}
	if n < 0 {
		return 0, fmt.Errorf("%s version component is negative: %d", name, n)
	}
	return n, nil
}

// String returns the version as a string in "MAJOR.MINOR.PATCH" format.
func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// IsZero returns true if the version is the zero value (0.0.0).
func (v Version) IsZero() bool {
	return v.Major == 0 && v.Minor == 0 && v.Patch == 0
}

// Compare returns:
//
//	-1 if v < other
//	 0 if v == other
//	+1 if v > other
func (v Version) Compare(other Version) int {
	if v.Major != other.Major {
		if v.Major < other.Major {
			return -1
		}
		return 1
	}
	if v.Minor != other.Minor {
		if v.Minor < other.Minor {
			return -1
		}
		return 1
	}
	if v.Patch != other.Patch {
		if v.Patch < other.Patch {
			return -1
		}
		return 1
	}
	return 0
}

// LessThan returns true if v < other.
func (v Version) LessThan(other Version) bool {
	return v.Compare(other) < 0
}

// LessThanOrEqual returns true if v <= other.
func (v Version) LessThanOrEqual(other Version) bool {
	return v.Compare(other) <= 0
}

// GreaterThan returns true if v > other.
func (v Version) GreaterThan(other Version) bool {
	return v.Compare(other) > 0
}

// GreaterThanOrEqual returns true if v >= other.
func (v Version) GreaterThanOrEqual(other Version) bool {
	return v.Compare(other) >= 0
}

// Equal returns true if v == other.
func (v Version) Equal(other Version) bool {
	return v.Compare(other) == 0
}

// IsCompatible returns true if v is compatible with other, i.e., they can
// communicate without breaking changes.
//
// Rules:
//   - For versions >= 1.0.0: same MAJOR version means compatible
//   - For 0.x versions (initial development): same MINOR version means compatible
//     (since MINOR bumps may be breaking in 0.x)
//   - The zero version (0.0.0) is treated as un-versioned and is compatible with everything
func (v Version) IsCompatible(other Version) bool {
	// Zero version means un-versioned — compatible with everything
	if v.IsZero() || other.IsZero() {
		return true
	}
	// Different major versions are never compatible
	if v.Major != other.Major {
		return false
	}
	// For 0.x, compatibility requires same minor version
	if v.Major == 0 {
		return v.Minor == other.Minor
	}
	// For >= 1.0.0, same major is sufficient
	return true
}

// IsValid returns true if the version is a valid semver (not the zero value).
func (v Version) IsValid() bool {
	return !v.IsZero()
}

// BumpMajor returns a new version with the major number incremented and minor/patch reset to 0.
func (v Version) BumpMajor() Version {
	return Version{Major: v.Major + 1, Minor: 0, Patch: 0}
}

// BumpMinor returns a new version with the minor number incremented and patch reset to 0.
func (v Version) BumpMinor() Version {
	return Version{Major: v.Major, Minor: v.Minor + 1, Patch: 0}
}

// BumpPatch returns a new version with the patch number incremented.
func (v Version) BumpPatch() Version {
	return Version{Major: v.Major, Minor: v.Minor, Patch: v.Patch + 1}
}
