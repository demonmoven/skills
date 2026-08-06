package skills

import (
	"fmt"
	"regexp"
	"strings"
)

// Valid skill kinds.
const (
	KindCapability    = "capability"
	KindOrchestration = "orchestration"
)

// validKinds is the set of allowed skill kind values.
// Empty string is also allowed for backward compatibility.
var validKinds = map[string]bool{
	KindCapability:    true,
	KindOrchestration: true,
}

// Required sections that every skill body must contain.
var requiredSections = []string{
	"## Trigger Examples",
	"## CLI Contract",
	"## Discipline",
}

var staleLoopStatePattern = regexp.MustCompile(`(?i)\bloop_state\b`)

// Semver parses the skill's version string into a Version struct.
// Returns an error if the version is not a valid semver.
func (sk Skill) Semver() (Version, error) {
	return ParseVersion(sk.Version)
}

// ValidationIssue represents a single validation problem found in a skill.
type ValidationIssue struct {
	Field   string // field or aspect that failed validation
	Message string // human-readable description of the issue
}

func (vi ValidationIssue) String() string {
	return fmt.Sprintf("%s: %s", vi.Field, vi.Message)
}

// Validate checks the skill for correctness and returns a list of issues.
// An empty/nil slice means the skill is valid.
//
// Validations performed:
//   - Name is non-empty
//   - Version is a valid semantic version
//   - Description is non-empty
//   - Kind is one of the valid values (or empty for backward compatibility)
//   - Body contains all required sections (# <name>, ## Trigger Examples, ## CLI Contract, ## Discipline)
func (sk Skill) Validate() []ValidationIssue {
	var issues []ValidationIssue

	// Name
	if strings.TrimSpace(sk.Name) == "" {
		issues = append(issues, ValidationIssue{
			Field:   "name",
			Message: "name must not be empty",
		})
	}

	// Version
	if sk.Version == "" {
		issues = append(issues, ValidationIssue{
			Field:   "version",
			Message: "version must not be empty",
		})
	} else if _, err := ParseVersion(sk.Version); err != nil {
		issues = append(issues, ValidationIssue{
			Field:   "version",
			Message: fmt.Sprintf("invalid version %q: %v", sk.Version, err),
		})
	}

	// Description
	if strings.TrimSpace(sk.Description) == "" {
		issues = append(issues, ValidationIssue{
			Field:   "description",
			Message: "description must not be empty",
		})
	}

	// Kind
	if sk.Kind != "" && !validKinds[sk.Kind] {
		validList := make([]string, 0, len(validKinds))
		for k := range validKinds {
			validList = append(validList, k)
		}
		issues = append(issues, ValidationIssue{
			Field:   "kind",
			Message: fmt.Sprintf("invalid kind %q; must be one of: %s (or empty)", sk.Kind, strings.Join(validList, ", ")),
		})
	}

	// Required sections in body
	if sk.Name != "" {
		h1Title := "# " + sk.Name
		if !strings.Contains(sk.Body, h1Title) {
			issues = append(issues, ValidationIssue{
				Field:   "body.h1",
				Message: fmt.Sprintf("body must contain H1 title %q", h1Title),
			})
		}
	}

	for _, section := range requiredSections {
		if !strings.Contains(sk.Body, section) {
			issues = append(issues, ValidationIssue{
				Field:   "body.sections",
				Message: fmt.Sprintf("body is missing required section %q", section),
			})
		}
	}

	if staleLoopStatePattern.MatchString(sk.Body) {
		issues = append(issues, ValidationIssue{
			Field:   "body.terminology",
			Message: "use Loop State Spine / loop_state_spine instead of stale loop_state",
		})
	}

	return issues
}

// IsValid returns true if the skill passes all validation checks.
func (sk Skill) IsValid() bool {
	return len(sk.Validate()) == 0
}

// ValidateAll validates all skills in the registry and returns a map of
// skill name to validation issues. Skills with no issues are not included.
func (r *Registry) ValidateAll() map[string][]ValidationIssue {
	result := map[string][]ValidationIssue{}
	for _, sk := range r.list {
		issues := sk.Validate()
		if len(issues) > 0 {
			result[sk.Name] = issues
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
