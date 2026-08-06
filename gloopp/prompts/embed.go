// Package builtinprompts embeds the official Gloop prompt templates.
//
// Prompt templates live outside internal/ so they are visible as versioned
// product content, while internal packages can still consume them through
// fs.FS injection.
package builtinprompts

import (
	"embed"
	"io/fs"
)

//go:embed *.md blocks/*.md
var content embed.FS

// FS returns the embedded official prompt tree.
//
// Paths are rooted at this package directory, for example
// "warrior_permissions.md" or "blocks/design_note.md".
func FS() fs.FS { return content }
