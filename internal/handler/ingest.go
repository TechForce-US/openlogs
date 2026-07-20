package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jmstewart1127/openlogs/internal/db"
)

// maxIngestBody caps the accepted request body size to guard against oversized payloads.
const maxIngestBody = 1 << 20 // 1 MiB

// ingestPayload mirrors the language-agnostic wire format, compatible with
// Monolog's JsonFormatter output.
type ingestPayload struct {
	Message   string          `json:"message"`
	LevelName string          `json:"level_name"`
	Level     *int            `json:"level"`
	Channel   string          `json:"channel"`
	Datetime  string          `json:"datetime"`
	Context   json.RawMessage `json:"context"`
	Extra     json.RawMessage `json:"extra"`
}

// monologLevels maps Monolog level names to their numeric severity.
var monologLevels = map[string]int{
	"DEBUG":     100,
	"INFO":      200,
	"NOTICE":    250,
	"WARNING":   300,
	"ERROR":     400,
	"CRITICAL":  500,
	"ALERT":     550,
	"EMERGENCY": 600,
}

// datetimeLayouts are tried in order when parsing the incoming timestamp.
var datetimeLayouts = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02T15:04:05.999999-07:00",
	"2006-01-02 15:04:05",
}

// Ingest accepts a single structured log entry, stores it, and broadcasts it to
// live subscribers. The project is resolved from the API key by middleware.
func (a *App) Ingest(w http.ResponseWriter, r *http.Request) {
	project := currentProject(r)
	if project == nil {
		// Should not happen: APIKey middleware guarantees a project.
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxIngestBody)
	var p ingestPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(p.Message) == "" || strings.TrimSpace(p.LevelName) == "" {
		http.Error(w, "message and level_name are required", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(p.Datetime) == "" {
		http.Error(w, "datetime is required", http.StatusBadRequest)
		return
	}

	loggedAt, err := parseDatetime(p.Datetime)
	if err != nil {
		http.Error(w, "datetime is not a recognised timestamp", http.StatusBadRequest)
		return
	}

	level := strings.ToUpper(strings.TrimSpace(p.LevelName))
	levelNum := 0
	if p.Level != nil {
		levelNum = *p.Level
	} else if n, ok := monologLevels[level]; ok {
		levelNum = n
	}

	entry := &db.Log{
		ProjectID: project.ID,
		Channel:   p.Channel,
		Level:     level,
		LevelNum:  levelNum,
		Message:   p.Message,
		Context:   jsonOrDefault(p.Context),
		Extra:     jsonOrDefault(p.Extra),
		LoggedAt:  loggedAt.UTC().Format("2006-01-02 15:04:05"),
	}

	stored, err := a.DB.InsertLog(entry)
	if err != nil {
		log.Printf("ingest: insert failed: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	a.Broker.Publish(project.ID, *stored)
	w.WriteHeader(http.StatusCreated)
}

// parseDatetime tries each supported layout and returns the parsed time.
func parseDatetime(s string) (time.Time, error) {
	var lastErr error
	for _, layout := range datetimeLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		} else {
			lastErr = err
		}
	}
	return time.Time{}, lastErr
}

// jsonOrDefault returns the raw JSON as a string, or "{}" when empty/null.
func jsonOrDefault(raw json.RawMessage) string {
	s := strings.TrimSpace(string(raw))
	if s == "" || s == "null" {
		return "{}"
	}
	return s
}
