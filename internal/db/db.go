// Package db provides the SQLite-backed persistence layer for OpenLogs.
package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// DB wraps a *sql.DB connection to the OpenLogs SQLite database.
type DB struct {
	*sql.DB
}

// Open opens (or creates) the SQLite database at path, enables WAL mode and
// foreign keys, and applies the schema. WAL allows concurrent reads during
// writes, which suits the ingest-while-viewing workload.
func Open(path string) (*DB, error) {
	// _pragma query params are honoured by modernc.org/sqlite on each connection.
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)", path)
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	if _, err := sqlDB.Exec(schema); err != nil {
		return nil, fmt.Errorf("apply schema: %w", err)
	}

	return &DB{sqlDB}, nil
}
