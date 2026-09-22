// Package web embeds the built React application in the Go executable.
//
// The dist/ directory is produced by `npm run build` in web/frontend (see
// scripts/check.ps1 and the Dockerfile); it does not exist in a fresh
// checkout, so the frontend must be built at least once before this package
// compiles.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var files embed.FS

// Dist is the built frontend's static file tree, rooted at dist/ so paths
// match what the browser requests (e.g. "index.html", "assets/app.js").
var Dist = mustSub(files, "dist")

func mustSub(f embed.FS, dir string) fs.FS {
	sub, err := fs.Sub(f, dir)
	if err != nil {
		panic(err)
	}
	return sub
}
