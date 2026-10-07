// Package web embeds the built Svelte UI (web/dist, produced by `make ui`).
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

// Dist is the built UI rooted at dist/.
func Dist() fs.FS {
	sub, _ := fs.Sub(dist, "dist")
	return sub
}
