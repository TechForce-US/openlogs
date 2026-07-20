package db

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

const inviteTimeFormat = "2006-01-02 15:04:05"

// Invite represents a pending one-time invitation.
type Invite struct {
	Token     string
	Email     string
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt string
}

// CreateInvite deletes any existing unused invite for email, then inserts a new
// one with the given token expiring after ttl.
func (d *DB) CreateInvite(email, token string, ttl time.Duration) (*Invite, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	expiresAt := time.Now().UTC().Add(ttl)

	// Supersede any prior pending invite for this email.
	if _, err := d.Exec(
		`DELETE FROM invites WHERE email = ? AND used_at IS NULL`, email,
	); err != nil {
		return nil, fmt.Errorf("delete prior invite: %w", err)
	}

	if _, err := d.Exec(
		`INSERT INTO invites (token, email, expires_at) VALUES (?, ?, ?)`,
		token, email, expiresAt.Format(inviteTimeFormat),
	); err != nil {
		return nil, fmt.Errorf("insert invite: %w", err)
	}

	return &Invite{Token: token, Email: email, ExpiresAt: expiresAt}, nil
}

// GetValidInvite returns the invite for the given token only if it exists, is
// unused, and has not expired. Returns (nil, nil) for any invalid state.
func (d *DB) GetValidInvite(token string) (*Invite, error) {
	var inv Invite
	var expiresStr string
	var usedAt sql.NullString

	err := d.QueryRow(
		`SELECT token, email, expires_at, used_at, created_at FROM invites WHERE token = ?`, token,
	).Scan(&inv.Token, &inv.Email, &expiresStr, &usedAt, &inv.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get invite: %w", err)
	}

	// Already used.
	if usedAt.Valid {
		return nil, nil
	}

	inv.ExpiresAt, err = time.Parse(inviteTimeFormat, expiresStr)
	if err != nil {
		return nil, fmt.Errorf("parse invite expiry: %w", err)
	}

	if time.Now().UTC().After(inv.ExpiresAt) {
		return nil, nil
	}

	return &inv, nil
}

// MarkInviteUsed sets used_at to now for the given token.
func (d *DB) MarkInviteUsed(token string) error {
	if _, err := d.Exec(
		`UPDATE invites SET used_at = ? WHERE token = ?`,
		time.Now().UTC().Format(inviteTimeFormat), token,
	); err != nil {
		return fmt.Errorf("mark invite used: %w", err)
	}
	return nil
}

// DeleteInvite removes an invite by token (used to revoke a pending invite).
func (d *DB) DeleteInvite(token string) error {
	if _, err := d.Exec(`DELETE FROM invites WHERE token = ?`, token); err != nil {
		return fmt.Errorf("delete invite: %w", err)
	}
	return nil
}

// ListPendingInvites returns all invites that are unused and not yet expired,
// ordered newest first.
func (d *DB) ListPendingInvites() ([]Invite, error) {
	rows, err := d.Query(
		`SELECT token, email, expires_at, created_at FROM invites
		 WHERE used_at IS NULL AND expires_at > ?
		 ORDER BY created_at DESC`,
		time.Now().UTC().Format(inviteTimeFormat),
	)
	if err != nil {
		return nil, fmt.Errorf("list invites: %w", err)
	}
	defer rows.Close()

	var invites []Invite
	for rows.Next() {
		var inv Invite
		var expiresStr string
		if err := rows.Scan(&inv.Token, &inv.Email, &expiresStr, &inv.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan invite: %w", err)
		}
		inv.ExpiresAt, err = time.Parse(inviteTimeFormat, expiresStr)
		if err != nil {
			return nil, fmt.Errorf("parse invite expiry: %w", err)
		}
		invites = append(invites, inv)
	}
	return invites, rows.Err()
}
