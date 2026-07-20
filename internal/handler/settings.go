package handler

import (
	"log"
	"net/http"

	"github.com/jmstewart1127/openlogs/internal/db"
	"golang.org/x/crypto/bcrypt"
)

// settingsView is the data for the account settings page.
type settingsView struct {
	Theme          string
	InviteLink     string      // set once after a successful invite creation
	PendingInvites []db.Invite // active (unused, unexpired) invites
}

// Settings renders the account settings page (password change + theme selector).
func (a *App) Settings(w http.ResponseWriter, r *http.Request) {
	a.renderSettings(w, r, "", nil, "")
}

// renderSettings renders the settings page with optional invite link, pending
// invites override, and flash message. Pass nil for invites to load them fresh.
func (a *App) renderSettings(w http.ResponseWriter, r *http.Request, inviteLink string, pending []db.Invite, flash string) {
	if pending == nil {
		var err error
		pending, err = a.DB.ListPendingInvites()
		if err != nil {
			log.Printf("settings: list invites failed: %v", err)
			pending = nil
		}
	}
	a.render(w, r, "settings", settingsView{
		Theme:          a.currentTheme(),
		InviteLink:     inviteLink,
		PendingInvites: pending,
	}, flash)
}

// ChangePassword validates the current password and updates it to a new one.
func (a *App) ChangePassword(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	current := r.FormValue("current_password")
	next := r.FormValue("new_password")
	confirm := r.FormValue("confirm_password")

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(current)); err != nil {
		a.renderSettings(w, r, "", nil, "Current password is incorrect")
		return
	}
	if len(next) < 8 {
		a.renderSettings(w, r, "", nil, "New password must be at least 8 characters")
		return
	}
	if next != confirm {
		a.renderSettings(w, r, "", nil, "New passwords do not match")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(next), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("settings: hash password failed: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if err := a.DB.UpdatePassword(user.ID, string(hash)); err != nil {
		log.Printf("settings: update password failed: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	a.renderSettings(w, r, "", nil, "Password updated")
}

// ChangeTheme swaps the active UI theme for the running process.
func (a *App) ChangeTheme(w http.ResponseWriter, r *http.Request) {
	theme := r.FormValue("theme")
	if theme != "modern" && theme != "terminal" {
		a.renderSettings(w, r, "", nil, "Unknown theme")
		return
	}
	a.setTheme(theme)
	http.Redirect(w, r, "/settings", http.StatusSeeOther)
}
