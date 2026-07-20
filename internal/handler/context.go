package handler

import (
	"net/http"

	"github.com/jmstewart1127/openlogs/internal/db"
	"github.com/jmstewart1127/openlogs/internal/middleware"
)

// currentUser returns the authenticated user for the request, or nil.
func currentUser(r *http.Request) *db.User {
	return middleware.UserFromContext(r.Context())
}

// currentProject returns the API-key-authenticated project for the request, or nil.
func currentProject(r *http.Request) *db.Project {
	return middleware.ProjectFromContext(r.Context())
}
