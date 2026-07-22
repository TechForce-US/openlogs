package db

import (
	"fmt"
	"strings"
)

// Log represents a single stored log entry. Context and Extra are stored as raw
// JSON text exactly as received. LoggedAt is stored in canonical UTC form
// ("2006-01-02 15:04:05") so ordering and retention comparisons are reliable.
type Log struct {
	ID        int64
	ProjectID string
	Channel   string
	Level     string
	LevelNum  int
	Message   string
	Context   string
	Extra     string
	LoggedAt  string
	CreatedAt string
}

// LogFilter describes the optional filters applied when querying logs.
type LogFilter struct {
	ProjectID string
	Levels    []string // OR-matched; empty means all levels
	Channel   string   // exact match, case-insensitive; empty means all
	Search    string   // FTS5 MATCH expression over message + context
	Limit     int

	// Date/time range bounds in canonical UTC form ("2006-01-02 15:04:05").
	// Empty means unbounded on that side; both may be set independently.
	From string // inclusive lower bound: logged_at >= From
	To   string // inclusive upper bound: logged_at <= To

	// Keyset pagination cursor. When BeforeID > 0, only entries strictly older
	// than (BeforeLoggedAt, BeforeID) in the (logged_at DESC, id DESC) ordering
	// are returned. This is stable under inserts at the top (live tail), unlike
	// OFFSET-based paging.
	BeforeLoggedAt string
	BeforeID       int64
}

// defaultLimit is the number of entries returned per page when none is specified.
const defaultLimit = 100

// InsertLog stores a log entry and returns it with its generated ID populated.
func (d *DB) InsertLog(l *Log) (*Log, error) {
	res, err := d.Exec(
		`INSERT INTO logs (project_id, channel, level, level_num, message, context, extra, logged_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		l.ProjectID, l.Channel, l.Level, l.LevelNum, l.Message, l.Context, l.Extra, l.LoggedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert log: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("insert log id: %w", err)
	}
	l.ID = id
	return l, nil
}

// InsertLogs stores a slice of log entries in a single multi-row INSERT inside
// an explicit transaction and returns the slice with IDs populated.
// SQLite guarantees contiguous autoincrement IDs within a single INSERT, so
// we recover all IDs from LastInsertId via lastID - N + 1 + i.
func (d *DB) InsertLogs(logs []*Log) ([]*Log, error) {
	if len(logs) == 0 {
		return logs, nil
	}

	// Build INSERT ... VALUES (?,?,?,?,?,?,?,?),(?,?,?,?,?,?,?,?), ...
	const cols = 8
	rowPlaceholder := "(?,?,?,?,?,?,?,?)"
	placeholders := make([]string, len(logs))
	args := make([]any, 0, len(logs)*cols)
	for i, l := range logs {
		placeholders[i] = rowPlaceholder
		args = append(args, l.ProjectID, l.Channel, l.Level, l.LevelNum, l.Message, l.Context, l.Extra, l.LoggedAt)
	}

	query := `INSERT INTO logs (project_id, channel, level, level_num, message, context, extra, logged_at) VALUES ` +
		strings.Join(placeholders, ",")

	tx, err := d.Begin()
	if err != nil {
		return nil, fmt.Errorf("insert logs: begin: %w", err)
	}
	res, err := tx.Exec(query, args...)
	if err != nil {
		_ = tx.Rollback()
		return nil, fmt.Errorf("insert logs: exec: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("insert logs: commit: %w", err)
	}

	lastID, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("insert logs: last id: %w", err)
	}
	n := int64(len(logs))
	for i, l := range logs {
		l.ID = lastID - n + 1 + int64(i)
	}
	return logs, nil
}

// QueryLogs returns up to the filter's limit of logs matching the filter, newest
// first.
func (d *DB) QueryLogs(f LogFilter) ([]Log, error) {
	return d.queryLogs(f, effectiveLimit(f))
}

// QueryLogsPage returns a page of logs plus whether at least one older entry
// exists beyond the page. It fetches one extra row to determine hasMore without a
// separate COUNT, then trims the result to the requested limit.
func (d *DB) QueryLogsPage(f LogFilter) (logs []Log, hasMore bool, err error) {
	limit := effectiveLimit(f)
	logs, err = d.queryLogs(f, limit+1)
	if err != nil {
		return nil, false, err
	}
	if len(logs) > limit {
		return logs[:limit], true, nil
	}
	return logs, false, nil
}

func effectiveLimit(f LogFilter) int {
	if f.Limit <= 0 {
		return defaultLimit
	}
	return f.Limit
}

// queryLogs runs the filtered, ordered query fetching at most fetchLimit rows.
func (d *DB) queryLogs(f LogFilter, fetchLimit int) ([]Log, error) {
	var (
		sb   strings.Builder
		args []any
	)

	// Full-text search requires joining the FTS index; otherwise query logs directly.
	if strings.TrimSpace(f.Search) != "" {
		sb.WriteString(`SELECT l.id, l.project_id, l.channel, l.level, l.level_num, l.message, l.context, l.extra, l.logged_at, l.created_at
			FROM logs l JOIN logs_fts f ON l.id = f.rowid
			WHERE l.project_id = ? AND logs_fts MATCH ?`)
		args = append(args, f.ProjectID, f.Search)
	} else {
		sb.WriteString(`SELECT id, project_id, channel, level, level_num, message, context, extra, logged_at, created_at
			FROM logs l
			WHERE l.project_id = ?`)
		args = append(args, f.ProjectID)
	}

	if len(f.Levels) > 0 {
		placeholders := make([]string, len(f.Levels))
		for i, lvl := range f.Levels {
			placeholders[i] = "?"
			args = append(args, strings.ToUpper(lvl))
		}
		sb.WriteString(" AND l.level IN (" + strings.Join(placeholders, ",") + ")")
	}

	if strings.TrimSpace(f.Channel) != "" {
		sb.WriteString(" AND l.channel = ? COLLATE NOCASE")
		args = append(args, strings.TrimSpace(f.Channel))
	}

	if f.From != "" {
		sb.WriteString(" AND l.logged_at >= ?")
		args = append(args, f.From)
	}

	if f.To != "" {
		sb.WriteString(" AND l.logged_at <= ?")
		args = append(args, f.To)
	}

	// Keyset cursor: entries strictly older than (BeforeLoggedAt, BeforeID) under
	// the logged_at DESC, id DESC ordering.
	if f.BeforeID > 0 && f.BeforeLoggedAt != "" {
		sb.WriteString(" AND (l.logged_at < ? OR (l.logged_at = ? AND l.id < ?))")
		args = append(args, f.BeforeLoggedAt, f.BeforeLoggedAt, f.BeforeID)
	}

	sb.WriteString(" ORDER BY l.logged_at DESC, l.id DESC LIMIT ?")
	args = append(args, fetchLimit)

	rows, err := d.Query(sb.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("query logs: %w", err)
	}
	defer rows.Close()

	var logs []Log
	for rows.Next() {
		var l Log
		if err := rows.Scan(
			&l.ID, &l.ProjectID, &l.Channel, &l.Level, &l.LevelNum,
			&l.Message, &l.Context, &l.Extra, &l.LoggedAt, &l.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan log: %w", err)
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

// DeleteOldLogs deletes logs older than the retention window and returns the
// number of rows removed. The logs_ad trigger keeps the FTS index in sync.
func (d *DB) DeleteOldLogs(retentionDays int) (int64, error) {
	res, err := d.Exec(
		fmt.Sprintf(`DELETE FROM logs WHERE logged_at < datetime('now', '-%d days')`, retentionDays),
	)
	if err != nil {
		return 0, fmt.Errorf("delete old logs: %w", err)
	}
	return res.RowsAffected()
}
