// Package handler implements the HTTP handlers for the OpenLogs web UI and
// ingest API.
package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/jmstewart1127/openlogs/internal/broker"
	"github.com/jmstewart1127/openlogs/internal/config"
	"github.com/jmstewart1127/openlogs/internal/db"
	"github.com/jmstewart1127/openlogs/web"
)

// App bundles the dependencies shared by all handlers.
type App struct {
	DB     *db.DB
	Broker *broker.Broker
	Config *config.Config
	tmpl   map[string]*template.Template

	// theme is the active UI theme. It is seeded from config on startup and can
	// be swapped at runtime via the settings page. Guarded by mu because it is
	// read on every render and may be written concurrently.
	mu    sync.RWMutex
	theme string
}

// New constructs an App and parses all templates. It returns an error if any
// template fails to parse.
func New(database *db.DB, b *broker.Broker, cfg *config.Config) (*App, error) {
	tmpl, err := buildTemplates()
	if err != nil {
		return nil, fmt.Errorf("build templates: %w", err)
	}
	return &App{DB: database, Broker: b, Config: cfg, tmpl: tmpl, theme: cfg.Theme}, nil
}

// currentTheme returns the active UI theme.
func (a *App) currentTheme() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.theme
}

// setTheme updates the active UI theme for the running process.
func (a *App) setTheme(theme string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.theme = theme
}

// customCSSFiles returns the sorted names of .css files in the configured custom
// CSS directory. A missing, empty, or unreadable directory yields no files and no
// error, so custom theming is a pure no-op when unused. The directory is read on
// each call so operators can drop in files without restarting.
func (a *App) customCSSFiles() []string {
	dir := a.Config.CustomCSSDir
	if dir == "" {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(strings.ToLower(e.Name()), ".css") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)
	return files
}

// funcMap holds template helper functions.
var funcMap = template.FuncMap{
	// prettyJSON indents a raw JSON string for readable display. On any error it
	// returns the original string unchanged.
	"prettyJSON": func(raw string) string {
		if raw == "" {
			return ""
		}
		var v any
		if err := json.Unmarshal([]byte(raw), &v); err != nil {
			return raw
		}
		out, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return raw
		}
		return string(out)
	},
	// levelClass maps a severity level to a CSS class suffix.
	"levelClass": func(level string) string {
		switch level {
		case "DEBUG":
			return "debug"
		case "INFO", "NOTICE":
			return "info"
		case "WARNING":
			return "warning"
		case "ERROR":
			return "error"
		case "CRITICAL", "ALERT", "EMERGENCY":
			return "critical"
		default:
			return "default"
		}
	},
	// hasContext reports whether a JSON object string holds any data.
	"hasData": func(raw string) bool {
		return raw != "" && raw != "{}" && raw != "[]" && raw != "null"
	},
}

// pages lists the full-page templates, each composed with the base layout and
// all partials.
var pages = []string{"login", "projects", "logs", "project-settings", "settings", "invite"}

func buildTemplates() (map[string]*template.Template, error) {
	m := make(map[string]*template.Template, len(pages)+1)
	for _, p := range pages {
		t, err := template.New(p).Funcs(funcMap).ParseFS(
			web.TemplatesFS,
			"templates/base.html",
			"templates/partials/*.html",
			"templates/"+p+".html",
		)
		if err != nil {
			return nil, fmt.Errorf("parse page %q: %w", p, err)
		}
		m[p] = t
	}

	// Standalone partial set for HTMX/SSE fragment rendering.
	pt, err := template.New("partials").Funcs(funcMap).ParseFS(
		web.TemplatesFS, "templates/partials/*.html",
	)
	if err != nil {
		return nil, fmt.Errorf("parse partials: %w", err)
	}
	m["_partials"] = pt
	return m, nil
}

// pageData is the top-level data passed to every full page render. Templates read
// .Theme and .User for the shared layout and .Data for page-specific content.
type pageData struct {
	Theme     string
	CustomCSS []string
	User      *db.User
	Flash     string
	Data      any
}

// render executes a full page template through the base layout.
func (a *App) render(w http.ResponseWriter, r *http.Request, page string, data any, flash string) {
	t, ok := a.tmpl[page]
	if !ok {
		log.Printf("render: unknown page %q", page)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	pd := pageData{Theme: a.currentTheme(), CustomCSS: a.customCSSFiles(), User: currentUser(r), Flash: flash, Data: data}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(w, "base", pd); err != nil {
		log.Printf("render %q: %v", page, err)
	}
}

// renderPartial renders a named partial to a single line of HTML, suitable for
// SSE payloads (which are newline-delimited) and HTMX swaps.
func (a *App) renderPartial(name string, data any) (string, error) {
	var buf bytes.Buffer
	if err := a.tmpl["_partials"].ExecuteTemplate(&buf, name, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
