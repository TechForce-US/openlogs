package middleware

import (
	"log"
	"net/http"

	"github.com/jmstewart1127/openlogs/internal/db"
)

// Auth returns middleware that requires a valid, unexpired session. It reads the
// session cookie, validates it against the database, loads the associated user,
// and attaches the user to the request context. Requests without a valid session
// are redirected to /login.
func Auth(database *db.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(SessionCookieName)
			if err != nil || cookie.Value == "" {
				redirectToLogin(w, r)
				return
			}

			session, err := database.GetSession(cookie.Value)
			if err != nil {
				log.Printf("auth: session lookup failed: %v", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			if session == nil {
				redirectToLogin(w, r)
				return
			}

			user, err := database.GetUserByID(session.UserID)
			if err != nil {
				log.Printf("auth: user lookup failed: %v", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			if user == nil {
				redirectToLogin(w, r)
				return
			}

			next.ServeHTTP(w, withUser(r, user))
		})
	}
}

func redirectToLogin(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
