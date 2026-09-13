// Package web embeds the browser application in the Go executable.
package web

import "embed"

//go:embed index.html app.js styles.css
var Files embed.FS
