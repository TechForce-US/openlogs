package handler

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jmstewart1127/openlogs/internal/db"
	"golang.org/x/crypto/bcrypt"
)

const inviteTTL = 7 * 24 * time.Hour

// newInviteToken generates a cryptographically random, URL-safe token.
func newInviteToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// inviteView is the data passed to the public accept-invite page.
type inviteView struct {
	Token   string
	Invalid bool // true when the token is not valid (show error, no form)
}

// InviteAccept renders the public set-password page for a valid invite token,
// or an error page for an invalid/expired/used token.
func (a *App) InviteAccept(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	inv, err := a.DB.GetValidInvite(token)
	if err != nil {
		log.Printf("invite: lookup failed: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if inv == nil {
		a.renderInvite(w, r, token, true, "This invite link is invalid, has expired, or has already been used.")
		return
	}
	a.renderInvite(w, r, token, false, "")
}

// InviteAcceptSubmit handles the set-password form for an invite.
func (a *App) InviteAcceptSubmit(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")

	// Re-validate on every POST to guard against races and replay.
	inv, err := a.DB.GetValidInvite(token)
	if err != nil {
		log.Printf("invite submit: lookup failed: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if inv == nil {
		a.renderInvite(w, r, token, true, "This invite link is invalid, has expired, or has already been used.")
		return
	}

	pw := r.FormValue("new_password")
	confirm := r.FormValue("confirm_password")
	if len(pw) < 8 {
		a.renderInviteWithFlash(w, r, token, "Password must be at least 8 characters.")
		return
	}
	if pw != confirm {
		a.renderInviteWithFlash(w, r, token, "Passwords do not match.")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("invite submit: hash failed: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if _, err := a.DB.CreateUser(inv.Email, string(hash)); err != nil {
		if errors.Is(err, db.ErrDuplicateEmail) {
			a.renderInviteWithFlash(w, r, token, "An account with that email already exists.")
			return
		}
		log.Printf("invite submit: create user failed: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if err := a.DB.MarkInviteUsed(token); err != nil {
		log.Printf("invite submit: mark used failed: %v", err)
	}

	http.Redirect(w, r, "/login?invited=1", http.StatusSeeOther)
}

// CreateInvite handles POST /settings/invite — generates a one-time link.
func (a *App) CreateInvite(w http.ResponseWriter, r *http.Request) {
	email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
	if email == "" {
		a.renderSettings(w, r, "", nil, "Email address is required.")
		return
	}

	existing, err := a.DB.GetUserByEmail(email)
	if err != nil {
		log.Printf("create invite: user lookup failed: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if existing != nil {
		a.renderSettings(w, r, "", nil, fmt.Sprintf("A user with email %q already exists.", email))
		return
	}

	token, err := newInviteToken()
	if err != nil {
		log.Printf("create invite: token generation failed: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if _, err := a.DB.CreateInvite(email, token, inviteTTL); err != nil {
		log.Printf("create invite: db failed: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	inviteLink := inviteLinkFor(r, token)
	a.renderSettings(w, r, inviteLink, nil, "")
}

// RevokeInvite handles POST /settings/invite/revoke — deletes a pending invite.
func (a *App) RevokeInvite(w http.ResponseWriter, r *http.Request) {
	token := r.FormValue("token")
	if token != "" {
		if err := a.DB.DeleteInvite(token); err != nil {
			log.Printf("revoke invite: %v", err)
		}
	}
	http.Redirect(w, r, "/settings#invite", http.StatusSeeOther)
}

// inviteLinkFor builds the absolute one-time invite URL from the request context.
func inviteLinkFor(r *http.Request, token string) string {
	scheme := "http"
	if isHTTPS(r) {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s/invite/%s", scheme, r.Host, token)
}

func (a *App) renderInvite(w http.ResponseWriter, r *http.Request, token string, invalid bool, flash string) {
	a.render(w, r, "invite", inviteView{Token: token, Invalid: invalid}, flash)
}

func (a *App) renderInviteWithFlash(w http.ResponseWriter, r *http.Request, token, flash string) {
	a.render(w, r, "invite", inviteView{Token: token, Invalid: false}, flash)
}
