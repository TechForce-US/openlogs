package db

import (
	"path/filepath"
	"testing"
	"time"
)

func newTestDB(t *testing.T) *DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	database, err := Open(path)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return database
}

func mustProject(t *testing.T, d *DB, name string) *Project {
	t.Helper()
	p, err := d.CreateProject(name)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	return p
}

func insert(t *testing.T, d *DB, projectID, level, channel, msg, ctx, loggedAt string) {
	t.Helper()
	num := monologNum(level)
	if ctx == "" {
		ctx = "{}"
	}
	_, err := d.InsertLog(&Log{
		ProjectID: projectID, Level: level, LevelNum: num, Channel: channel,
		Message: msg, Context: ctx, Extra: "{}", LoggedAt: loggedAt,
	})
	if err != nil {
		t.Fatalf("insert log: %v", err)
	}
}

func monologNum(level string) int {
	m := map[string]int{"DEBUG": 100, "INFO": 200, "WARNING": 300, "ERROR": 400, "CRITICAL": 500}
	return m[level]
}

func now() string { return time.Now().UTC().Format("2006-01-02 15:04:05") }

func TestInsertAndQueryOrder(t *testing.T) {
	d := newTestDB(t)
	p := mustProject(t, d, "app")

	insert(t, d, p.ID, "INFO", "web", "first", "", "2026-01-01 10:00:00")
	insert(t, d, p.ID, "ERROR", "web", "second", "", "2026-01-01 11:00:00")

	logs, err := d.QueryLogs(LogFilter{ProjectID: p.ID})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(logs) != 2 {
		t.Fatalf("want 2 logs, got %d", len(logs))
	}
	if logs[0].Message != "second" {
		t.Errorf("want newest-first (second), got %q", logs[0].Message)
	}
}

func TestLevelFilter(t *testing.T) {
	d := newTestDB(t)
	p := mustProject(t, d, "app")
	insert(t, d, p.ID, "INFO", "", "info msg", "", now())
	insert(t, d, p.ID, "ERROR", "", "error msg", "", now())
	insert(t, d, p.ID, "WARNING", "", "warn msg", "", now())

	logs, err := d.QueryLogs(LogFilter{ProjectID: p.ID, Levels: []string{"ERROR", "WARNING"}})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(logs) != 2 {
		t.Fatalf("want 2 (ERROR+WARNING), got %d", len(logs))
	}
	for _, l := range logs {
		if l.Level != "ERROR" && l.Level != "WARNING" {
			t.Errorf("unexpected level %q", l.Level)
		}
	}
}

func TestChannelFilterCaseInsensitive(t *testing.T) {
	d := newTestDB(t)
	p := mustProject(t, d, "app")
	insert(t, d, p.ID, "INFO", "Billing", "a", "", now())
	insert(t, d, p.ID, "INFO", "web", "b", "", now())

	logs, err := d.QueryLogs(LogFilter{ProjectID: p.ID, Channel: "billing"})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(logs) != 1 || logs[0].Message != "a" {
		t.Fatalf("want 1 billing log 'a', got %+v", logs)
	}
}

func TestFTSSearch(t *testing.T) {
	d := newTestDB(t)
	p := mustProject(t, d, "app")
	insert(t, d, p.ID, "ERROR", "billing", "checkout failed", `{"order_ref":"abc123"}`, now())
	insert(t, d, p.ID, "INFO", "web", "user logged in", `{"user":"kim"}`, now())

	// Match on message.
	logs, err := d.QueryLogs(LogFilter{ProjectID: p.ID, Search: `"checkout"*`})
	if err != nil {
		t.Fatalf("search message: %v", err)
	}
	if len(logs) != 1 || logs[0].Message != "checkout failed" {
		t.Fatalf("message search: want 1 'checkout failed', got %+v", logs)
	}

	// Match on context value.
	logs, err = d.QueryLogs(LogFilter{ProjectID: p.ID, Search: `"abc123"*`})
	if err != nil {
		t.Fatalf("search context: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("context search: want 1, got %d", len(logs))
	}

	// No match.
	logs, err = d.QueryLogs(LogFilter{ProjectID: p.ID, Search: `"zzzznope"*`})
	if err != nil {
		t.Fatalf("search none: %v", err)
	}
	if len(logs) != 0 {
		t.Fatalf("want 0, got %d", len(logs))
	}
}

func TestRetentionPrunesAndSyncsFTS(t *testing.T) {
	d := newTestDB(t)
	p := mustProject(t, d, "app")
	insert(t, d, p.ID, "INFO", "", "ancient event", "", "2000-01-01 00:00:00")
	insert(t, d, p.ID, "INFO", "", "recent event", "", now())

	n, err := d.DeleteOldLogs(30)
	if err != nil {
		t.Fatalf("delete old: %v", err)
	}
	if n != 1 {
		t.Fatalf("want 1 pruned, got %d", n)
	}

	all, _ := d.QueryLogs(LogFilter{ProjectID: p.ID})
	if len(all) != 1 || all[0].Message != "recent event" {
		t.Fatalf("want only recent event, got %+v", all)
	}

	// FTS must no longer return the pruned row.
	hits, err := d.QueryLogs(LogFilter{ProjectID: p.ID, Search: `"ancient"*`})
	if err != nil {
		t.Fatalf("fts after prune: %v", err)
	}
	if len(hits) != 0 {
		t.Fatalf("FTS still returns pruned row: %+v", hits)
	}
}

func TestCreateUserDuplicate(t *testing.T) {
	d := newTestDB(t)
	if _, err := d.CreateUser("a@b.com", "hash"); err != nil {
		t.Fatalf("first create: %v", err)
	}
	_, err := d.CreateUser("A@B.com", "hash2") // case-insensitive
	if err != ErrDuplicateEmail {
		t.Fatalf("want ErrDuplicateEmail, got %v", err)
	}
}

func TestSessionExpiry(t *testing.T) {
	d := newTestDB(t)
	u, _ := d.CreateUser("a@b.com", "hash")

	// Already-expired session.
	s, err := d.CreateSession(u.ID, -time.Minute)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	got, err := d.GetSession(s.ID)
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if got != nil {
		t.Fatalf("expired session should be nil, got %+v", got)
	}

	// Valid session.
	s2, _ := d.CreateSession(u.ID, time.Hour)
	got2, _ := d.GetSession(s2.ID)
	if got2 == nil {
		t.Fatal("valid session should be returned")
	}
}

func TestDeleteProjectCascadesAndSyncsFTS(t *testing.T) {
	d := newTestDB(t)
	p := mustProject(t, d, "app")
	insert(t, d, p.ID, "ERROR", "", "boom disaster", "", now())

	if err := d.DeleteProject(p.ID); err != nil {
		t.Fatalf("delete project: %v", err)
	}

	// Project gone.
	gp, _ := d.GetProjectByID(p.ID)
	if gp != nil {
		t.Fatal("project should be deleted")
	}

	// Recreate a project and ensure the FTS index has no stale "boom disaster".
	p2 := mustProject(t, d, "app2")
	hits, err := d.QueryLogs(LogFilter{ProjectID: p2.ID, Search: `"disaster"*`})
	if err != nil {
		t.Fatalf("fts: %v", err)
	}
	if len(hits) != 0 {
		t.Fatalf("stale FTS rows after project delete: %+v", hits)
	}
}

func TestQueryLogsPagePaginates(t *testing.T) {
	d := newTestDB(t)
	p := mustProject(t, d, "app")

	// 250 entries sharing the same timestamp exercises the (logged_at, id)
	// tiebreak in the keyset cursor.
	const total = 250
	ts := now()
	for i := 0; i < total; i++ {
		insert(t, d, p.ID, "INFO", "", "msg", "", ts)
	}

	seen := make(map[int64]bool)
	var lastLogged string
	var lastID int64
	pages := 0
	for {
		f := LogFilter{ProjectID: p.ID, Limit: 100}
		if lastID > 0 {
			f.BeforeLoggedAt = lastLogged
			f.BeforeID = lastID
		}
		logs, hasMore, err := d.QueryLogsPage(f)
		if err != nil {
			t.Fatalf("page %d: %v", pages, err)
		}
		pages++
		if len(logs) == 0 {
			t.Fatal("unexpected empty page")
		}
		// Verify descending id order within the page and no duplicates across pages.
		for i, l := range logs {
			if seen[l.ID] {
				t.Fatalf("duplicate id %d across pages", l.ID)
			}
			seen[l.ID] = true
			if i > 0 && logs[i-1].ID <= l.ID {
				t.Fatalf("not descending: %d then %d", logs[i-1].ID, l.ID)
			}
		}
		last := logs[len(logs)-1]
		lastLogged, lastID = last.LoggedAt, last.ID
		if !hasMore {
			break
		}
	}

	if len(seen) != total {
		t.Fatalf("want %d distinct entries across pages, got %d", total, len(seen))
	}
	if pages != 3 { // 100 + 100 + 50
		t.Fatalf("want 3 pages, got %d", pages)
	}
}

func TestQueryLogsPageHasMoreBoundary(t *testing.T) {
	d := newTestDB(t)
	p := mustProject(t, d, "app")
	ts := now()

	insertN := func(n int) {
		for i := 0; i < n; i++ {
			insert(t, d, p.ID, "INFO", "", "m", "", ts)
		}
	}

	// Exactly 100 → no further page.
	insertN(100)
	logs, hasMore, err := d.QueryLogsPage(LogFilter{ProjectID: p.ID, Limit: 100})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(logs) != 100 || hasMore {
		t.Fatalf("100 entries: want 100 rows/hasMore=false, got %d/%v", len(logs), hasMore)
	}

	// 101 → a further page exists.
	insertN(1)
	logs, hasMore, err = d.QueryLogsPage(LogFilter{ProjectID: p.ID, Limit: 100})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(logs) != 100 || !hasMore {
		t.Fatalf("101 entries: want 100 rows/hasMore=true, got %d/%v", len(logs), hasMore)
	}
}

func TestDateRangeFilter(t *testing.T) {
	d := newTestDB(t)
	p := mustProject(t, d, "app")

	insert(t, d, p.ID, "INFO", "", "early", "", "2026-01-01 08:00:00")
	insert(t, d, p.ID, "INFO", "", "mid", "", "2026-01-01 12:00:00")
	insert(t, d, p.ID, "ERROR", "", "late", "", "2026-01-01 20:00:00")

	// From-only: at or after noon.
	logs, err := d.QueryLogs(LogFilter{ProjectID: p.ID, From: "2026-01-01 12:00:00"})
	if err != nil {
		t.Fatalf("from-only: %v", err)
	}
	if len(logs) != 2 {
		t.Fatalf("from-only: want 2, got %d", len(logs))
	}
	for _, l := range logs {
		if l.Message == "early" {
			t.Errorf("from-only: unexpected 'early' in results")
		}
	}

	// To-only: at or before noon.
	logs, err = d.QueryLogs(LogFilter{ProjectID: p.ID, To: "2026-01-01 12:00:00"})
	if err != nil {
		t.Fatalf("to-only: %v", err)
	}
	if len(logs) != 2 {
		t.Fatalf("to-only: want 2, got %d", len(logs))
	}
	for _, l := range logs {
		if l.Message == "late" {
			t.Errorf("to-only: unexpected 'late' in results")
		}
	}

	// Closed range: noon only (exact match).
	logs, err = d.QueryLogs(LogFilter{ProjectID: p.ID, From: "2026-01-01 12:00:00", To: "2026-01-01 12:00:00"})
	if err != nil {
		t.Fatalf("closed-range: %v", err)
	}
	if len(logs) != 1 || logs[0].Message != "mid" {
		t.Fatalf("closed-range: want 1 'mid', got %+v", logs)
	}

	// Range combined with level filter.
	logs, err = d.QueryLogs(LogFilter{ProjectID: p.ID, From: "2026-01-01 08:00:00", Levels: []string{"ERROR"}})
	if err != nil {
		t.Fatalf("range+level: %v", err)
	}
	if len(logs) != 1 || logs[0].Message != "late" {
		t.Fatalf("range+level: want 1 'late' ERROR, got %+v", logs)
	}
}

func TestInviteLifecycle(t *testing.T) {
	d := newTestDB(t)

	// Create supersedes a prior pending invite for the same email.
	inv1, err := d.CreateInvite("user@example.com", "token-a", time.Hour)
	if err != nil {
		t.Fatalf("first invite: %v", err)
	}
	if inv1.Token != "token-a" {
		t.Errorf("unexpected token: %s", inv1.Token)
	}
	inv2, err := d.CreateInvite("user@example.com", "token-b", time.Hour)
	if err != nil {
		t.Fatalf("second invite: %v", err)
	}
	_ = inv2

	// token-a should no longer be valid (superseded).
	got, err := d.GetValidInvite("token-a")
	if err != nil {
		t.Fatalf("get superseded: %v", err)
	}
	if got != nil {
		t.Fatal("superseded invite should not be valid")
	}

	// token-b should be valid.
	valid, err := d.GetValidInvite("token-b")
	if err != nil {
		t.Fatalf("get valid: %v", err)
	}
	if valid == nil {
		t.Fatal("current invite should be valid")
	}

	// MarkInviteUsed makes it invalid.
	if err := d.MarkInviteUsed("token-b"); err != nil {
		t.Fatalf("mark used: %v", err)
	}
	used, err := d.GetValidInvite("token-b")
	if err != nil {
		t.Fatalf("get after used: %v", err)
	}
	if used != nil {
		t.Fatal("used invite should not be valid")
	}
}

func TestInviteExpired(t *testing.T) {
	d := newTestDB(t)
	if _, err := d.CreateInvite("a@b.com", "tok-expired", -time.Minute); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := d.GetValidInvite("tok-expired")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got != nil {
		t.Fatal("expired invite should not be valid")
	}
}

func TestInviteListPending(t *testing.T) {
	d := newTestDB(t)
	d.CreateInvite("a@b.com", "tok1", time.Hour)
	d.CreateInvite("b@b.com", "tok2", time.Hour)
	d.CreateInvite("c@b.com", "tok3", -time.Minute) // expired

	pending, err := d.ListPendingInvites()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(pending) != 2 {
		t.Fatalf("want 2 pending, got %d", len(pending))
	}
}

func TestRegenerateAPIKeyInvalidatesOld(t *testing.T) {
	d := newTestDB(t)
	p := mustProject(t, d, "app")
	oldKey := p.APIKey

	newKey, err := d.RegenerateAPIKey(p.ID)
	if err != nil {
		t.Fatalf("regenerate: %v", err)
	}
	if newKey == oldKey {
		t.Fatal("new key should differ from old")
	}
	if got, _ := d.GetProjectByAPIKey(oldKey); got != nil {
		t.Fatal("old key should no longer resolve")
	}
	if got, _ := d.GetProjectByAPIKey(newKey); got == nil {
		t.Fatal("new key should resolve")
	}
}
