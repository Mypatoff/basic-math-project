// Package web bundles the HTML templates and static assets into the
// compiled binary, so deploying is just copying one executable.
package web

import "embed"

// go:embed can't reach outside this directory, which is why this
// file lives in web/ next to the folders it embeds.
//
//go:embed templates static
var FS embed.FS
