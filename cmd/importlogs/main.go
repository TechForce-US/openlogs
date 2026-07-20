// Command importlogs parses a Laravel text log file (the default LineFormatter
// output, e.g. storage/logs/laravel.log) and inserts every entry into an OpenLogs
// SQLite database. It is a local testing/seeding helper — it writes directly to
// the database via the internal db package, so no running server is required.
//
// Usage:
//
//	go run ./cmd/importlogs [flags]
//
// Flags:
//
//	-file      Path to the Laravel log file       (default "laravel.log")
//	-db        Path to the OpenLogs SQLite file    (default $OPENLOGS_DB_PATH or "data/openlogs.db")
//	-project   Project name or ID to import into   (default "laravel"; created if absent)
//	-recent    Shift timestamps so the newest entry lands at "now", preserving
//	           relative spacing, so entries survive the 30-day retention window.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/jmstewart1127/openlogs/internal/db"
)

// headerRe matches a Laravel log record header:
//
//	[2025-03-06 02:27:33] local.ERROR: message and optional context...
var headerRe = regexp.MustCompile(`^\[(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2})\] ([^.\s]+)\.([A-Z]+): (.*)$`)

const timeLayout = "2006-01-02 15:04:05"

// psrLevels maps PSR-3 / Monolog level names to their numeric severity.
var psrLevels = map[string]int{
	"DEBUG": 100, "INFO": 200, "NOTICE": 250, "WARNING": 300,
	"ERROR": 400, "CRITICAL": 500, "ALERT": 550, "EMERGENCY": 600,
}

// record is one parsed log entry before it is written to the database.
type record struct {
	when    time.Time
	channel string
	level   string
	message string
	detail  string // full raw body (message line remainder + following lines)
}

func main() {
	file := flag.String("file", "laravel.log", "path to the Laravel log file")
	dbPath := flag.String("db", defaultDBPath(), "path to the OpenLogs SQLite database")
	projectName := flag.String("project", "laravel", "project name or ID to import into (created if it does not exist)")
	recent := flag.Bool("recent", false, "shift timestamps so the newest entry lands at now (survives retention)")
	flag.Parse()

	records, err := parseFile(*file)
	if err != nil {
		log.Fatalf("importlogs: %v", err)
	}
	if len(records) == 0 {
		log.Fatalf("importlogs: no log records found in %s", *file)
	}

	if *recent {
		shiftToRecent(records)
	}

	database, err := db.Open(*dbPath)
	if err != nil {
		log.Fatalf("importlogs: open db: %v", err)
	}
	defer database.Close()

	project, err := resolveProject(database, *projectName)
	if err != nil {
		log.Fatalf("importlogs: %v", err)
	}

	inserted := 0
	for _, r := range records {
		ctx, _ := json.Marshal(map[string]string{"detail": r.detail})
		if _, err := database.InsertLog(&db.Log{
			ProjectID: project.ID,
			Channel:   r.channel,
			Level:     r.level,
			LevelNum:  psrLevels[r.level],
			Message:   r.message,
			Context:   string(ctx),
			Extra:     "{}",
			LoggedAt:  r.when.UTC().Format(timeLayout),
		}); err != nil {
			log.Fatalf("importlogs: insert log: %v", err)
		}
		inserted++
	}

	fmt.Printf("Imported %d log entries into project %q (%s) from %s\n",
		inserted, project.Name, project.ID, *file)
	oldest, newest := records[0].when, records[len(records)-1].when
	fmt.Printf("Timestamp range: %s → %s\n", oldest.Format(timeLayout), newest.Format(timeLayout))
	if !*recent && time.Since(newest) > 30*24*time.Hour {
		fmt.Println("Note: all entries predate the 30-day retention window and will be pruned on the next")
		fmt.Println("      server start. Re-run with -recent to shift them into the recent window.")
	}
}

// parseFile reads the log file and groups lines into records. Each record begins
// with a header line; subsequent non-header lines (stack traces, etc.) are
// appended to the current record's detail.
func parseFile(path string) ([]record, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var (
		records []record
		cur     *record
		body    strings.Builder
	)

	flush := func() {
		if cur != nil {
			cur.detail = strings.TrimRight(body.String(), "\n")
			records = append(records, *cur)
		}
		body.Reset()
	}

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1024*1024), 8*1024*1024) // allow long stacktrace lines
	for sc.Scan() {
		line := sc.Text()
		if m := headerRe.FindStringSubmatch(line); m != nil {
			flush()
			when, err := time.Parse(timeLayout, m[1])
			if err != nil {
				// Skip records with an unparseable timestamp rather than aborting.
				cur = nil
				continue
			}
			rest := m[4]
			cur = &record{
				when:    when,
				channel: m[2],
				level:   m[3],
				message: extractMessage(rest),
			}
			body.WriteString(rest)
			body.WriteByte('\n')
			continue
		}
		if cur != nil {
			body.WriteString(line)
			body.WriteByte('\n')
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	flush()
	return records, nil
}

// extractMessage returns the human-readable message from a header line remainder,
// stripping the trailing JSON/array context that Laravel appends.
func extractMessage(rest string) string {
	msg := rest
	// Context is appended as " {" (object) or " [" (array); cut at the first one.
	if i := strings.Index(rest, " {"); i >= 0 {
		msg = rest[:i]
	} else if i := strings.Index(rest, " ["); i >= 0 {
		msg = rest[:i]
	}
	msg = strings.TrimSpace(msg)
	if msg == "" || msg == "[]" {
		return "(no message)"
	}
	return msg
}

// shiftToRecent moves every timestamp forward by the same delta so the newest
// entry lands at the current time, preserving the relative spacing between entries.
func shiftToRecent(records []record) {
	var newest time.Time
	for _, r := range records {
		if r.when.After(newest) {
			newest = r.when
		}
	}
	delta := time.Since(newest)
	for i := range records {
		records[i].when = records[i].when.Add(delta)
	}
}

// resolveProject finds a project by ID or name, creating it by name if absent.
func resolveProject(database *db.DB, nameOrID string) (*db.Project, error) {
	if p, err := database.GetProjectByID(nameOrID); err == nil && p != nil {
		return p, nil
	}
	projects, err := database.ListProjects()
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	for i := range projects {
		if strings.EqualFold(projects[i].Name, nameOrID) {
			return &projects[i], nil
		}
	}
	p, err := database.CreateProject(nameOrID)
	if err != nil {
		return nil, fmt.Errorf("create project %q: %w", nameOrID, err)
	}
	fmt.Printf("Created project %q\n", nameOrID)
	return p, nil
}

func defaultDBPath() string {
	if p := os.Getenv("OPENLOGS_DB_PATH"); p != "" {
		return p
	}
	return "data/openlogs.db"
}
