package db

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// User represents an authenticated web UI user.
type User struct {
	ID           string
	Email        string
	PasswordHash string
	CreatedAt    string
}

// ErrDuplicateEmail is returned when creating a user whose email already exists.
var ErrDuplicateEmail = errors.New("a user with that email already exists")

// CreateUser inserts a new user with an already-hashed password.
func (d *DB) CreateUser(email, passwordHash string) (*User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return nil, errors.New("email must not be empty")
	}

	u := &User{ID: uuid.NewString(), Email: email, PasswordHash: passwordHash}
	_, err := d.Exec(
		`INSERT INTO users (id, email, password_hash) VALUES (?, ?, ?)`,
		u.ID, u.Email, u.PasswordHash,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrDuplicateEmail
		}
		return nil, fmt.Errorf("insert user: %w", err)
	}
	return u, nil
}

// GetUserByEmail looks up a user by email. Returns (nil, nil) if not found.
func (d *DB) GetUserByEmail(email string) (*User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	var u User
	err := d.QueryRow(
		`SELECT id, email, password_hash, created_at FROM users WHERE email = ?`, email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return &u, nil
}

// GetUserByID looks up a user by ID. Returns (nil, nil) if not found.
func (d *DB) GetUserByID(id string) (*User, error) {
	var u User
	err := d.QueryRow(
		`SELECT id, email, password_hash, created_at FROM users WHERE id = ?`, id,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return &u, nil
}

// UpdatePassword sets a new password hash for a user.
func (d *DB) UpdatePassword(id, passwordHash string) error {
	if _, err := d.Exec(`UPDATE users SET password_hash = ? WHERE id = ?`, passwordHash, id); err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	return nil
}
