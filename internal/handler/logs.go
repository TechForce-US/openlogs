package handler

import (
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jmstewart1127/openlogs/internal/db"
)

// allLevels is the fixed set of severity levels offered as filter checkboxes.
var allLevels = []string{"DEBUG", "INFO", "NOTICE", "WARNING", "ERROR", "CRITICAL", "ALERT", "EMERGENCY"}

// pageSize is the number of log entries shown per page.
const pageSize = 100

// logPageView is the data for the log-page partial: a batch of rows, an optional
// load-more URL (empty when there is no further page), and whether this is the
// first page (controls the empty-state message).
type logPageView struct {
	Logs        []db.Log
	LoadMoreURL string
	FirstPage   bool
}

// logsView is the data for the full log viewer page.
type logsView struct {
	Project        *db.Project
	Q              string
	Channel        string
	From           string // datetime-local value for re-populating the input
	To             string // datetime-local value for re-populating the input
	SelectedLevels map[string]bool
	AllLevels      []string
	Page           logPageView
}

// Logs renders the log viewer for a project. It serves three response shapes from
// one route:
//   - a load-more request (cursor present) returns the log-page fragment for an
//     outerHTML swap of the load-more control;
//   - an HTMX filter change returns the log-page fragment for the #log-list inner
//     HTML;
//   - a normal navigation returns the full page.
func (a *App) Logs(w http.ResponseWriter, r *http.Request) {
	project := a.loadProject(w, r)
	if project == nil {
		return
	}

	q := r.URL.Query()
	levels := q["level"]
	channel := q.Get("channel")
	search := q.Get("q")
	fromInput := q.Get("from")
	toInput := q.Get("to")

	// Normalize datetime-local inputs to canonical UTC form for the DB query.
	// Unparseable or empty values are dropped (treated as unbounded).
	fromDB := parseDatetimeLocal(fromInput)
	toDB := parseDatetimeLocal(toInput)

	beforeTime := q.Get("before_time")
	beforeIDStr := q.Get("before_id")
	isLoadMore := beforeIDStr != ""
	var beforeID int64
	if isLoadMore {
		beforeID, _ = strconv.ParseInt(beforeIDStr, 10, 64)
	}

	logs, hasMore, err := a.DB.QueryLogsPage(db.LogFilter{
		ProjectID:      project.ID,
		Levels:         levels,
		Channel:        channel,
		Search:         sanitizeFTS(search),
		From:           fromDB,
		To:             toDB,
		Limit:          pageSize,
		BeforeLoggedAt: beforeTime,
		BeforeID:       beforeID,
	})
	if err != nil {
		log.Printf("logs: query failed: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	var loadMoreURL string
	if hasMore && len(logs) > 0 {
		last := logs[len(logs)-1]
		loadMoreURL = buildLoadMoreURL(project.ID, levels, channel, search, fromInput, toInput, last.LoggedAt, last.ID)
	}

	page := logPageView{Logs: logs, LoadMoreURL: loadMoreURL, FirstPage: !isLoadMore}

	// Load-more and filter changes both return the log-page fragment; the client
	// decides how to swap it (outerHTML of the control vs innerHTML of the list).
	if isLoadMore || r.Header.Get("HX-Request") == "true" {
		html, err := a.renderPartial("log-page", page)
		if err != nil {
			log.Printf("logs: render partial failed: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(html))
		return
	}

	selected := make(map[string]bool, len(levels))
	for _, l := range levels {
		selected[strings.ToUpper(l)] = true
	}

	a.render(w, r, "logs", logsView{
		Project:        project,
		Q:              search,
		Channel:        channel,
		From:           fromInput,
		To:             toInput,
		SelectedLevels: selected,
		AllLevels:      allLevels,
		Page:           page,
	}, "")
}

// buildLoadMoreURL constructs the next-page URL carrying the active filters plus
// the keyset cursor. Values are escaped via net/url.
func buildLoadMoreURL(projectID string, levels []string, channel, search, from, to, beforeTime string, beforeID int64) string {
	v := url.Values{}
	for _, l := range levels {
		v.Add("level", l)
	}
	if strings.TrimSpace(channel) != "" {
		v.Set("channel", channel)
	}
	if strings.TrimSpace(search) != "" {
		v.Set("q", search)
	}
	if strings.TrimSpace(from) != "" {
		v.Set("from", from)
	}
	if strings.TrimSpace(to) != "" {
		v.Set("to", to)
	}
	v.Set("before_time", beforeTime)
	v.Set("before_id", strconv.FormatInt(beforeID, 10))
	return "/projects/" + projectID + "?" + v.Encode()
}

// parseDatetimeLocal converts a datetime-local input value ("2006-01-02T15:04")
// into the canonical UTC form used by the DB ("2006-01-02 15:04:05"). The input
// is interpreted as local wall-clock time. Returns "" if the input is empty or
// cannot be parsed, so the caller can treat it as unbounded.
func parseDatetimeLocal(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	t, err := time.ParseInLocation("2006-01-02T15:04", s, time.Local)
	if err != nil {
		// Also try with seconds, in case some browsers include them.
		t, err = time.ParseInLocation("2006-01-02T15:04:05", s, time.Local)
		if err != nil {
			return ""
		}
	}
	return t.UTC().Format("2006-01-02 15:04:05")
}

// sanitizeFTS converts free-form user input into a safe FTS5 MATCH expression.
// Each whitespace-separated term is quoted (escaping embedded quotes) and given a
// prefix wildcard, so "checkout err" becomes `"checkout"* "err"*`. This avoids
// FTS5 syntax errors from user-entered special characters. Empty input yields "".
func sanitizeFTS(q string) string {
	terms := strings.Fields(q)
	if len(terms) == 0 {
		return ""
	}
	parts := make([]string, 0, len(terms))
	for _, t := range terms {
		escaped := strings.ReplaceAll(t, `"`, `""`)
		parts = append(parts, `"`+escaped+`"*`)
	}
	return strings.Join(parts, " ")
}
