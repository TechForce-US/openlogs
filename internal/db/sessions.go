package db

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// sessionTimeFormat is the layout used to store session expiry timestamps.
const sessionTimeFormat = "2006-01-02 15:04:05"

// Session represents an authenticated browser session.
type Session struct {
	ID        string
	UserID    string
	ExpiresAt time.Time
}

// CreateSession creates a session for a user that expires after the given duration.
func (d *DB) CreateSession(userID string, ttl time.Duration) (*Session, error) {
	s := &Session{
		ID:        uuid.NewString(),
		UserID:    userID,
		ExpiresAt: time.Now().UTC().Add(ttl),
	}
	_, err := d.Exec(
		`INSERT INTO sessions (id, user_id, expires_at) VALUES (?, ?, ?)`,
		s.ID, s.UserID, s.ExpiresAt.Format(sessionTimeFormat),
	)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	return s, nil
}

// GetSession returns a session by ID if it exists and has not expired. An expired
// or missing session yields (nil, nil).
func (d *DB) GetSession(id string) (*Session, error) {
	var s Session
	var expires string
	err := d.QueryRow(
		`SELECT id, user_id, expires_at FROM sessions WHERE id = ?`, id,
	).Scan(&s.ID, &s.UserID, &expires)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	s.ExpiresAt, err = time.Parse(sessionTimeFormat, expires)
	if err != nil {
		return nil, fmt.Errorf("parse session expiry: %w", err)
	}

	if time.Now().UTC().After(s.ExpiresAt) {
		// Expired: clean it up and treat as absent.
		_ = d.DeleteSession(id)
		return nil, nil
	}
	return &s, nil
}

// DeleteSession removes a session by ID.
func (d *DB) DeleteSession(id string) error {
	if _, err := d.Exec(`DELETE FROM sessions WHERE id = ?`, id); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

// DeleteExpiredSessions removes all sessions whose expiry is in the past.
func (d *DB) DeleteExpiredSessions() error {
	if _, err := d.Exec(
		`DELETE FROM sessions WHERE expires_at < ?`,
		time.Now().UTC().Format(sessionTimeFormat),
	); err != nil {
		return fmt.Errorf("delete expired sessions: %w", err)
	}
	return nil
}
