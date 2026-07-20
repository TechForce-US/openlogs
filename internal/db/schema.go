package db

// schema defines the full database schema. It is idempotent (IF NOT EXISTS) so it
// can be run on every startup to migrate a fresh or existing database.
//
// The FTS5 virtual table is kept in sync with the logs table via triggers. We use
// an external-content FTS5 table (content='logs') so the indexed text is not stored
// twice; the triggers mirror inserts and deletes into the index.
const schema = `
CREATE TABLE IF NOT EXISTS projects (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL UNIQUE,
    api_key    TEXT NOT NULL UNIQUE,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS users (
    id            TEXT PRIMARY KEY,
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at    TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS sessions (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS invites (
    token      TEXT PRIMARY KEY,
    email      TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    used_at    TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_invites_email
    ON invites(email);

CREATE TABLE IF NOT EXISTS logs (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    channel    TEXT NOT NULL DEFAULT '',
    level      TEXT NOT NULL DEFAULT '',
    level_num  INTEGER NOT NULL DEFAULT 0,
    message    TEXT NOT NULL,
    context    TEXT NOT NULL DEFAULT '{}',
    extra      TEXT NOT NULL DEFAULT '{}',
    logged_at  TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_logs_project_logged
    ON logs(project_id, logged_at DESC);

CREATE INDEX IF NOT EXISTS idx_sessions_expires
    ON sessions(expires_at);

-- Full-text index over message + context. External content table references logs
-- by rowid so we only store the index, not a copy of the data.
CREATE VIRTUAL TABLE IF NOT EXISTS logs_fts USING fts5(
    message,
    context,
    content='logs',
    content_rowid='id'
);

CREATE TRIGGER IF NOT EXISTS logs_ai AFTER INSERT ON logs BEGIN
    INSERT INTO logs_fts(rowid, message, context)
    VALUES (new.id, new.message, new.context);
END;

CREATE TRIGGER IF NOT EXISTS logs_ad AFTER DELETE ON logs BEGIN
    INSERT INTO logs_fts(logs_fts, rowid, message, context)
    VALUES ('delete', old.id, old.message, old.context);
END;
`
