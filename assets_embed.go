//go:build embed

package main

import (
	"embed"
	"io/fs"
)

// distFS carries the built SPA. `all:` keeps files Vite may emit with a leading
// dot or underscore. The embed directive lives at the repo root because
// go:embed cannot reach through `..`, so it must sit at or above web/dist.
//
//go:embed all:web/dist
var distFS embed.FS

// webAssets returns the embedded dist subtree, so callers see index.html at the
// root rather than under a web/dist prefix.
func webAssets() fs.FS {
	sub, err := fs.Sub(distFS, "web/dist")
	if err != nil {
		panic(err)
	}
	return sub
}
