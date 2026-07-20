// Package web embeds the HTML templates and static assets so the compiled binary
// is fully self-contained.
package web

import "embed"

// TemplatesFS holds the HTML templates under templates/.
//
//go:embed all:templates
var TemplatesFS embed.FS

// StaticFS holds static assets (CSS, JS) under static/.
//
//go:embed static
var StaticFS embed.FS
