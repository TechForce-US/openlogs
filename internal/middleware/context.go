// Package middleware provides HTTP middleware for session and API-key
// authentication, plus typed helpers for reading the authenticated principal
// out of the request context.
package middleware

import (
	"context"
	"net/http"

	"github.com/jmstewart1127/openlogs/internal/db"
)

// SessionCookieName is the name of the cookie holding the session UUID.
const SessionCookieName = "openlogs_session"

type contextKey int

const (
	userKey contextKey = iota
	projectKey
)

// UserFromContext returns the authenticated user attached by the Auth middleware,
// or nil if none is present.
func UserFromContext(ctx context.Context) *db.User {
	u, _ := ctx.Value(userKey).(*db.User)
	return u
}

// ProjectFromContext returns the project attached by the APIKey middleware,
// or nil if none is present.
func ProjectFromContext(ctx context.Context) *db.Project {
	p, _ := ctx.Value(projectKey).(*db.Project)
	return p
}

func withUser(r *http.Request, u *db.User) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), userKey, u))
}

func withProject(r *http.Request, p *db.Project) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), projectKey, p))
}
