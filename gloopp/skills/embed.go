// Package builtinskills embeds the official Gloop skill assets.
//
// Skill markdown lives outside internal/ so it is visible as versioned product
// content, while internal packages can still consume it through fs.FS injection.
package builtinskills

import (
	"embed"
	"io/fs"
)

//go:embed */SKILL.md
var content embed.FS

// FS returns the embedded official skill tree.
//
// Paths are rooted at this package directory, for example
// "gloop-quest-execution/SKILL.md".
func FS() fs.FS { return content }
