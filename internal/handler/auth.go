package handler

import (
	"log"
	"net/http"
	"time"

	"github.com/jmstewart1127/openlogs/internal/middleware"
	"golang.org/x/crypto/bcrypt"
)

// sessionTTL is how long a login session remains valid.
const sessionTTL = 24 * time.Hour

// Login renders the login form.
func (a *App) Login(w http.ResponseWriter, r *http.Request) {
	flash := ""
	if r.URL.Query().Get("invited") == "1" {
		flash = "Account created! Log in with your new password."
	}
	a.render(w, r, "login", nil, flash)
}

// LoginSubmit validates credentials and, on success, creates a session and sets
// the session cookie. Failures return a generic error to avoid user enumeration.
func (a *App) LoginSubmit(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	password := r.FormValue("password")

	if email == "" || password == "" {
		a.render(w, r, "login", nil, "Invalid email or password")
		return
	}

	user, err := a.DB.GetUserByEmail(email)
	if err != nil {
		log.Printf("login: user lookup failed: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if user == nil {
		a.render(w, r, "login", nil, "Invalid email or password")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		a.render(w, r, "login", nil, "Invalid email or password")
		return
	}

	session, err := a.DB.CreateSession(user.ID, sessionTTL)
	if err != nil {
		log.Printf("login: create session failed: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     middleware.SessionCookieName,
		Value:    session.ID,
		Path:     "/",
		HttpOnly: true,
		Secure:   isHTTPS(r),
		SameSite: http.SameSiteStrictMode,
		Expires:  session.ExpiresAt,
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Logout deletes the current session and clears the cookie.
func (a *App) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(middleware.SessionCookieName); err == nil && cookie.Value != "" {
		if err := a.DB.DeleteSession(cookie.Value); err != nil {
			log.Printf("logout: delete session failed: %v", err)
		}
	}

	http.SetCookie(w, &http.Cookie{
		Name:     middleware.SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   isHTTPS(r),
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// isHTTPS reports whether the request arrived over TLS, either directly or via a
// TLS-terminating reverse proxy (e.g. Caddy setting X-Forwarded-Proto).
func isHTTPS(r *http.Request) bool {
	return r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
}
