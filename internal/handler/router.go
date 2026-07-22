package handler

import (
	"io/fs"
	"net/http"

	"github.com/jmstewart1127/openlogs/internal/middleware"
	"github.com/jmstewart1127/openlogs/web"
)

// Router builds the complete HTTP handler with all routes and middleware wired up.
func (a *App) Router() (http.Handler, error) {
	mux := http.NewServeMux()
	authMW := middleware.Auth(a.DB)
	apiKeyMW := middleware.APIKey(a.DB)

	// protect wraps a handler with session authentication.
	protect := func(h http.HandlerFunc) http.Handler { return authMW(h) }

	// Static assets.
	staticSub, err := fs.Sub(web.StaticFS, "static")
	if err != nil {
		return nil, err
	}
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(staticSub)))

	// Custom CSS overrides served from the runtime filesystem (not embedded), so
	// operators can drop in .css files without rebuilding. More specific than the
	// /static/ route above, so it takes precedence.
	customFS := http.FileServer(http.Dir(a.Config.CustomCSSDir))
	mux.Handle("GET /static/custom/", http.StripPrefix("/static/custom/", customFS))

	// Public routes.
	mux.HandleFunc("GET /login", a.Login)
	mux.HandleFunc("POST /login", a.LoginSubmit)
	mux.HandleFunc("POST /logout", a.Logout)
	mux.HandleFunc("GET /invite/{token}", a.InviteAccept)
	mux.HandleFunc("POST /invite/{token}", a.InviteAcceptSubmit)

	// Ingest API (API-key authenticated).
	mux.Handle("POST /api/ingest", apiKeyMW(http.HandlerFunc(a.Ingest)))
	mux.Handle("POST /api/ingest/batch", apiKeyMW(http.HandlerFunc(a.IngestBatch)))

	// Protected web routes.
	mux.Handle("GET /{$}", protect(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/projects", http.StatusSeeOther)
	}))
	mux.Handle("GET /projects", protect(a.Projects))
	mux.Handle("POST /projects", protect(a.CreateProject))
	mux.Handle("GET /projects/{id}", protect(a.Logs))
	mux.Handle("GET /projects/{id}/stream", protect(a.Stream))
	mux.Handle("GET /projects/{id}/settings", protect(a.ProjectSettings))
	mux.Handle("POST /projects/{id}/settings", protect(a.RenameProject))
	mux.Handle("POST /projects/{id}/regenerate-key", protect(a.RegenerateKey))
	mux.Handle("POST /projects/{id}/delete", protect(a.DeleteProject))
	mux.Handle("GET /settings", protect(a.Settings))
	mux.Handle("POST /settings/password", protect(a.ChangePassword))
	mux.Handle("POST /settings/theme", protect(a.ChangeTheme))
	mux.Handle("POST /settings/invite", protect(a.CreateInvite))
	mux.Handle("POST /settings/invite/revoke", protect(a.RevokeInvite))

	return mux, nil
}
