package db

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// Project represents a log source with its own ingest API key.
type Project struct {
	ID        string
	Name      string
	APIKey    string
	CreatedAt string
}

// ErrDuplicateName is returned when a project name collides with an existing one.
var ErrDuplicateName = errors.New("a project with that name already exists")

// generateAPIKey returns a 32-byte cryptographically random key, hex-encoded.
func generateAPIKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate api key: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// CreateProject inserts a new project with a freshly generated API key.
func (d *DB) CreateProject(name string) (*Project, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("project name must not be empty")
	}

	key, err := generateAPIKey()
	if err != nil {
		return nil, err
	}

	p := &Project{ID: uuid.NewString(), Name: name, APIKey: key}
	_, err = d.Exec(
		`INSERT INTO projects (id, name, api_key) VALUES (?, ?, ?)`,
		p.ID, p.Name, p.APIKey,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrDuplicateName
		}
		return nil, fmt.Errorf("insert project: %w", err)
	}
	return d.GetProjectByID(p.ID)
}

// ListProjects returns all projects ordered by creation time.
func (d *DB) ListProjects() ([]Project, error) {
	rows, err := d.Query(`SELECT id, name, api_key, created_at FROM projects ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	var projects []Project
	for rows.Next() {
		var p Project
		if err := rows.Scan(&p.ID, &p.Name, &p.APIKey, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

// GetProjectByAPIKey looks up a project by its ingest API key.
func (d *DB) GetProjectByAPIKey(key string) (*Project, error) {
	return d.getProject(`SELECT id, name, api_key, created_at FROM projects WHERE api_key = ?`, key)
}

// GetProjectByID looks up a project by its ID.
func (d *DB) GetProjectByID(id string) (*Project, error) {
	return d.getProject(`SELECT id, name, api_key, created_at FROM projects WHERE id = ?`, id)
}

func (d *DB) getProject(query, arg string) (*Project, error) {
	var p Project
	err := d.QueryRow(query, arg).Scan(&p.ID, &p.Name, &p.APIKey, &p.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get project: %w", err)
	}
	return &p, nil
}

// RenameProject updates a project's name.
func (d *DB) RenameProject(id, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("project name must not be empty")
	}
	_, err := d.Exec(`UPDATE projects SET name = ? WHERE id = ?`, name, id)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrDuplicateName
		}
		return fmt.Errorf("rename project: %w", err)
	}
	return nil
}

// RegenerateAPIKey issues a new API key for a project, invalidating the old one.
func (d *DB) RegenerateAPIKey(id string) (string, error) {
	key, err := generateAPIKey()
	if err != nil {
		return "", err
	}
	if _, err := d.Exec(`UPDATE projects SET api_key = ? WHERE id = ?`, key, id); err != nil {
		return "", fmt.Errorf("regenerate api key: %w", err)
	}
	return key, nil
}

// DeleteProject removes a project and all its logs. Logs are deleted explicitly
// (rather than relying solely on ON DELETE CASCADE) so the FTS delete trigger
// fires and the search index stays consistent.
func (d *DB) DeleteProject(id string) error {
	tx, err := d.Begin()
	if err != nil {
		return fmt.Errorf("delete project: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM logs WHERE project_id = ?`, id); err != nil {
		return fmt.Errorf("delete project logs: %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM projects WHERE id = ?`, id); err != nil {
		return fmt.Errorf("delete project: %w", err)
	}
	return tx.Commit()
}

// isUniqueViolation reports whether err is a SQLite UNIQUE constraint failure.
func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}
