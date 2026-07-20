package middleware

import (
	"log"
	"net/http"

	"github.com/jmstewart1127/openlogs/internal/db"
)

// APIKey returns middleware that authenticates ingest requests via the X-API-Key
// header. The key identifies the target project, which is attached to the request
// context. Missing or unknown keys receive HTTP 401.
func APIKey(database *db.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("X-API-Key")
			if key == "" {
				http.Error(w, "missing API key", http.StatusUnauthorized)
				return
			}

			project, err := database.GetProjectByAPIKey(key)
			if err != nil {
				log.Printf("apikey: project lookup failed: %v", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			if project == nil {
				http.Error(w, "invalid API key", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, withProject(r, project))
		})
	}
}
