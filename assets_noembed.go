//go:build !embed

package main

import "io/fs"

// webAssets returns nil without the embed tag, so `go build` and `go test`
// work before web/dist exists. A nil FS makes `cred web` fall back to serving
// web/dist from disk, or a stub page when that is absent too.
func webAssets() fs.FS { return nil }
