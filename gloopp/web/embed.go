// Package webdist embeds the built React/Vite dashboard.
//
// Run `npm --prefix web run build` before `go build`; go:embed requires
// web/dist to exist at compile time.
package webdist

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var content embed.FS

// FS returns the embedded dashboard rooted at dist.
func FS() fs.FS {
	sub, err := fs.Sub(content, "dist")
	if err != nil {
		panic(err)
	}
	return sub
}
