package handler

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jmstewart1127/openlogs/internal/broker"
	"github.com/jmstewart1127/openlogs/internal/config"
	"github.com/jmstewart1127/openlogs/internal/db"
	"golang.org/x/crypto/bcrypt"
)

// newTestServerWithCustom builds a server whose custom-CSS directory is customDir.
func newTestServerWithCustom(t *testing.T, customDir string) *httptest.Server {
	t.Helper()
	database, err := db.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	cfg := &config.Config{Theme: "modern", RetentionDays: 30, SecretKey: "test", Port: "0", CustomCSSDir: customDir}
	app, err := New(database, broker.New(), cfg)
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	router, err := app.Router()
	if err != nil {
		t.Fatalf("router: %v", err)
	}
	srv := httptest.NewServer(router)
	t.Cleanup(func() { srv.Close(); database.Close() })
	return srv
}

func getBody(t *testing.T, url string) string {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("get %s: %v", url, err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return string(b)
}

func TestCustomCSSAbsent(t *testing.T) {
	// Non-existent custom dir → theme link only, no custom links, no error.
	srv := newTestServerWithCustom(t, filepath.Join(t.TempDir(), "does-not-exist"))
	body := getBody(t, srv.URL+"/login")
	if !strings.Contains(body, "/static/modern.css") {
		t.Fatal("expected built-in theme link")
	}
	if strings.Contains(body, "/static/custom/") {
		t.Fatal("expected no custom links when dir absent")
	}
}

func TestCustomCSSInjectedAfterTheme(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "00-vars.css", ":root{--accent:#f00}")
	writeFile(t, dir, "10-layout.css", ".container{max-width:1200px}")
	writeFile(t, dir, "notes.txt", "not css")

	srv := newTestServerWithCustom(t, dir)
	body := getBody(t, srv.URL+"/login")

	themeIdx := strings.Index(body, "/static/modern.css")
	varsIdx := strings.Index(body, "/static/custom/00-vars.css")
	layoutIdx := strings.Index(body, "/static/custom/10-layout.css")

	if themeIdx < 0 || varsIdx < 0 || layoutIdx < 0 {
		t.Fatalf("missing expected links: theme=%d vars=%d layout=%d", themeIdx, varsIdx, layoutIdx)
	}
	// Custom links come after the theme, and sorted order is preserved.
	if !(themeIdx < varsIdx && varsIdx < layoutIdx) {
		t.Fatalf("wrong order: theme=%d vars=%d layout=%d", themeIdx, varsIdx, layoutIdx)
	}
	// Non-.css file is ignored.
	if strings.Contains(body, "notes.txt") {
		t.Fatal("non-css file should not be linked")
	}

	// The custom file is served from the filesystem.
	resp, err := http.Get(srv.URL + "/static/custom/00-vars.css")
	if err != nil {
		t.Fatalf("get custom css: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("custom css: want 200, got %d", resp.StatusCode)
	}
	b, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(b), "--accent:#f00") {
		t.Fatalf("custom css content not served, got %q", string(b))
	}

	// A missing custom file 404s.
	miss, _ := http.Get(srv.URL + "/static/custom/nope.css")
	if miss.StatusCode != http.StatusNotFound {
		t.Fatalf("missing custom css: want 404, got %d", miss.StatusCode)
	}
	miss.Body.Close()
}

func TestLogPagination(t *testing.T) {
	srv, database := newTestServer(t, "modern")
	project, _ := database.CreateProject("app")
	hash, _ := bcrypt.GenerateFromPassword([]byte("supersecret"), bcrypt.DefaultCost)
	database.CreateUser("admin@example.com", string(hash))

	// 150 entries in channel "web" → first page 100 + load-more, second page 50.
	for i := 0; i < 150; i++ {
		if _, err := database.InsertLog(&db.Log{
			ProjectID: project.ID, Level: "INFO", LevelNum: 200, Channel: "web",
			Message: "event", Context: "{}", Extra: "{}",
			LoggedAt: "2026-01-01 00:00:00",
		}); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	jar := newJarClient()
	doLogin(t, jar, srv.URL, "admin@example.com", "supersecret").Body.Close()

	get := func(path string) string {
		t.Helper()
		resp, err := jar.Get(srv.URL + path)
		if err != nil {
			t.Fatalf("get %s: %v", path, err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return string(b)
	}
	countRows := func(s string) int { return strings.Count(s, `class="log-row`) }

	// First page (full navigation) with an active channel filter.
	body := get("/projects/" + project.ID + "?channel=web")
	if n := countRows(body); n != 100 {
		t.Fatalf("first page: want 100 rows, got %d", n)
	}
	if !strings.Contains(body, `class="load-more"`) {
		t.Fatal("first page: expected a load-more control")
	}

	// Extract the load-more URL and confirm it carries the cursor + filter.
	url := extractLoadMoreURL(t, body)
	if !strings.Contains(url, "before_id=") || !strings.Contains(url, "before_time=") {
		t.Fatalf("load-more URL missing cursor: %s", url)
	}
	if !strings.Contains(url, "channel=web") {
		t.Fatalf("load-more URL should preserve channel filter: %s", url)
	}

	// Fetch the second page via the load-more URL.
	frag := get(url)
	if n := countRows(frag); n != 50 {
		t.Fatalf("second page: want 50 rows, got %d", n)
	}
	if strings.Contains(frag, `class="load-more"`) {
		t.Fatal("second page: load-more control should be gone at the end")
	}
}

// extractLoadMoreURL pulls the hx-get value out of the load-more control.
func extractLoadMoreURL(t *testing.T, body string) string {
	t.Helper()
	marker := `class="load-more"`
	i := strings.Index(body, marker)
	if i < 0 {
		t.Fatal("no load-more control in body")
	}
	seg := body[i:]
	start := strings.Index(seg, `hx-get="`)
	if start < 0 {
		t.Fatal("load-more has no hx-get")
	}
	seg = seg[start+len(`hx-get="`):]
	end := strings.Index(seg, `"`)
	if end < 0 {
		t.Fatal("malformed hx-get")
	}
	// The attribute is HTML-escaped (&amp;); a browser decodes it before HTMX uses
	// it, so do the same here.
	return html.UnescapeString(seg[:end])
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func newTestServer(t *testing.T, theme string) (*httptest.Server, *db.DB) {
	t.Helper()
	database, err := db.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	cfg := &config.Config{Theme: theme, RetentionDays: 30, SecretKey: "test", Port: "0"}
	app, err := New(database, broker.New(), cfg)
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	router, err := app.Router()
	if err != nil {
		t.Fatalf("router: %v", err)
	}
	srv := httptest.NewServer(router)
	t.Cleanup(func() { srv.Close(); database.Close() })
	return srv, database
}

func TestIngestFlow(t *testing.T) {
	srv, database := newTestServer(t, "modern")
	project, _ := database.CreateProject("app")

	body := `{"message":"Payment failed","level_name":"error","channel":"billing",
		"datetime":"2026-07-18T14:23:00+00:00","context":{"user_id":42}}`

	// Missing key -> 401.
	resp := postIngest(t, srv.URL, "", body)
	if resp != http.StatusUnauthorized {
		t.Fatalf("missing key: want 401, got %d", resp)
	}
	// Bad key -> 401.
	if got := postIngest(t, srv.URL, "nope", body); got != http.StatusUnauthorized {
		t.Fatalf("bad key: want 401, got %d", got)
	}
	// Malformed JSON -> 400.
	if got := postIngest(t, srv.URL, project.APIKey, "{not json"); got != http.StatusBadRequest {
		t.Fatalf("bad json: want 400, got %d", got)
	}
	// Missing required field -> 400.
	if got := postIngest(t, srv.URL, project.APIKey, `{"message":"x"}`); got != http.StatusBadRequest {
		t.Fatalf("missing field: want 400, got %d", got)
	}
	// Valid -> 201.
	if got := postIngest(t, srv.URL, project.APIKey, body); got != http.StatusCreated {
		t.Fatalf("valid: want 201, got %d", got)
	}

	// Verify stored with normalised level.
	logs, err := database.QueryLogs(db.LogFilter{ProjectID: project.ID})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("want 1 log, got %d", len(logs))
	}
	if logs[0].Level != "ERROR" || logs[0].LevelNum != 400 {
		t.Errorf("level normalisation: got %q/%d", logs[0].Level, logs[0].LevelNum)
	}
	if logs[0].Message != "Payment failed" {
		t.Errorf("message: got %q", logs[0].Message)
	}
}

func postIngest(t *testing.T, base, key, body string) int {
	t.Helper()
	req, _ := http.NewRequest("POST", base+"/api/ingest", strings.NewReader(body))
	if key != "" {
		req.Header.Set("X-API-Key", key)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

func TestProtectedRouteRedirectsWhenUnauthenticated(t *testing.T) {
	srv, _ := newTestServer(t, "modern")
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	resp, err := client.Get(srv.URL + "/projects")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("want 303 redirect, got %d", resp.StatusCode)
	}
	if loc := resp.Header.Get("Location"); loc != "/login" {
		t.Fatalf("want redirect to /login, got %q", loc)
	}
}

func TestLoginLogoutSession(t *testing.T) {
	srv, database := newTestServer(t, "modern")
	hash, _ := bcrypt.GenerateFromPassword([]byte("supersecret"), bcrypt.DefaultCost)
	database.CreateUser("admin@example.com", string(hash))

	jar := newJarClient()

	// Wrong password: no session cookie, stays on login.
	resp := doLogin(t, jar, srv.URL, "admin@example.com", "wrong")
	if hasSessionCookie(jar, srv.URL) {
		t.Fatal("should not have session after failed login")
	}
	resp.Body.Close()

	// Correct password: sets session, can reach protected route.
	resp = doLogin(t, jar, srv.URL, "admin@example.com", "supersecret")
	resp.Body.Close()
	if !hasSessionCookie(jar, srv.URL) {
		t.Fatal("expected session cookie after login")
	}

	// Protected route now returns 200.
	r, _ := jar.Get(srv.URL + "/projects")
	if r.StatusCode != http.StatusOK {
		t.Fatalf("authenticated /projects: want 200, got %d", r.StatusCode)
	}
	r.Body.Close()

	// Logout clears session.
	lr, _ := jar.Post(srv.URL+"/logout", "application/x-www-form-urlencoded", nil)
	lr.Body.Close()
	r2, _ := jar.Get(srv.URL + "/projects")
	if r2.StatusCode != http.StatusSeeOther {
		t.Fatalf("after logout: want 303, got %d", r2.StatusCode)
	}
	r2.Body.Close()
}

func TestThemeCSSServed(t *testing.T) {
	for _, theme := range []string{"modern", "terminal"} {
		srv, database := newTestServer(t, theme)
		hash, _ := bcrypt.GenerateFromPassword([]byte("supersecret"), bcrypt.DefaultCost)
		database.CreateUser("a@b.com", string(hash))

		// Login page references the right stylesheet.
		resp, _ := http.Get(srv.URL + "/login")
		buf := make([]byte, 4096)
		n, _ := resp.Body.Read(buf)
		resp.Body.Close()
		if !strings.Contains(string(buf[:n]), "/static/"+theme+".css") {
			t.Errorf("theme %q: login page missing stylesheet link", theme)
		}

		// The stylesheet itself is served.
		cssResp, _ := http.Get(srv.URL + "/static/" + theme + ".css")
		if cssResp.StatusCode != http.StatusOK {
			t.Errorf("theme %q: css not served, got %d", theme, cssResp.StatusCode)
		}
		cssResp.Body.Close()
	}
}

func TestDateFilterParsedAndPreservedInLoadMore(t *testing.T) {
	srv, database := newTestServer(t, "modern")
	project, _ := database.CreateProject("app")
	hash, _ := bcrypt.GenerateFromPassword([]byte("supersecret"), bcrypt.DefaultCost)
	database.CreateUser("admin@example.com", string(hash))

	// Insert 150 entries so there is a load-more control.
	for i := 0; i < 150; i++ {
		database.InsertLog(&db.Log{
			ProjectID: project.ID, Level: "INFO", LevelNum: 200,
			Message: "event", Context: "{}", Extra: "{}",
			LoggedAt: "2026-06-01 12:00:00",
		})
	}

	jar := newJarClient()
	doLogin(t, jar, srv.URL, "admin@example.com", "supersecret").Body.Close()

	get := func(path string) string {
		t.Helper()
		resp, err := jar.Get(srv.URL + path)
		if err != nil {
			t.Fatalf("get %s: %v", path, err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return string(b)
	}

	from := "2026-06-01T00:00"
	to := "2026-06-01T23:59"
	body := get("/projects/" + project.ID + "?from=" + url.QueryEscape(from) + "&to=" + url.QueryEscape(to))

	// The load-more URL should carry the date params.
	loadMoreURL := extractLoadMoreURL(t, body)
	if !strings.Contains(loadMoreURL, "from=") {
		t.Errorf("load-more URL missing 'from': %s", loadMoreURL)
	}
	if !strings.Contains(loadMoreURL, "to=") {
		t.Errorf("load-more URL missing 'to': %s", loadMoreURL)
	}
}

func TestInviteFlow(t *testing.T) {
	srv, database := newTestServer(t, "modern")
	hash, _ := bcrypt.GenerateFromPassword([]byte("supersecret"), bcrypt.DefaultCost)
	database.CreateUser("admin@example.com", string(hash))

	jar := newJarClient()
	doLogin(t, jar, srv.URL, "admin@example.com", "supersecret").Body.Close()

	postBody := func(path, form string) string {
		t.Helper()
		resp, err := jar.Post(srv.URL+path, "application/x-www-form-urlencoded", strings.NewReader(form))
		if err != nil {
			t.Fatalf("post %s: %v", path, err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return string(b)
	}

	// Invite for an existing user: error shown on POST response.
	errPage := postBody("/settings/invite", "email=admin%40example.com")
	if !strings.Contains(errPage, "already exists") {
		t.Errorf("expected existing-user error, got: %q", errPage[:min(300, len(errPage))])
	}

	// Valid invite: POST response contains the generated link.
	invPage := postBody("/settings/invite", url.Values{"email": {"new@example.com"}}.Encode())
	if !strings.Contains(invPage, "/invite/") {
		t.Fatal("expected invite link in POST response")
	}

	// Extract token from the value attribute of the invite link input.
	marker := `/invite/`
	linkIdx := strings.Index(invPage, marker)
	if linkIdx < 0 {
		t.Fatal("no invite path in page")
	}
	rest := invPage[linkIdx+len(marker):]
	endIdx := strings.IndexAny(rest, `"' `)
	if endIdx < 0 {
		t.Fatal("could not parse token from page")
	}
	token := rest[:endIdx]

	// Invalid token shows error, not form.
	anonClient := &http.Client{}
	r, _ := anonClient.Get(srv.URL + "/invite/bad-token")
	b, _ := io.ReadAll(r.Body)
	r.Body.Close()
	if strings.Contains(string(b), "new_password") {
		t.Fatal("invalid token should not show password form")
	}

	// Valid token shows set-password form.
	r, _ = anonClient.Get(srv.URL + "/invite/" + token)
	b, _ = io.ReadAll(r.Body)
	r.Body.Close()
	if !strings.Contains(string(b), "new_password") {
		t.Fatalf("valid token should show password form, got: %q", string(b)[:min(300, len(b))])
	}

	// Weak password is rejected — user not created.
	anonJar, _ := cookiejar.New(nil)
	anonWithJar := &http.Client{Jar: anonJar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	weakResp, _ := anonWithJar.Post(srv.URL+"/invite/"+token, "application/x-www-form-urlencoded",
		strings.NewReader("new_password=short&confirm_password=short"))
	weakBody, _ := io.ReadAll(weakResp.Body)
	weakResp.Body.Close()
	if weakResp.StatusCode == http.StatusSeeOther {
		t.Fatal("weak password should not redirect to success")
	}
	_ = weakBody
	u, _ := database.GetUserByEmail("new@example.com")
	if u != nil {
		t.Fatal("user should not be created with weak password")
	}

	// Valid password creates user and consumes the invite.
	acceptResp, _ := anonWithJar.Post(srv.URL+"/invite/"+token, "application/x-www-form-urlencoded",
		strings.NewReader("new_password=strongpassword1&confirm_password=strongpassword1"))
	acceptResp.Body.Close()
	if acceptResp.StatusCode != http.StatusSeeOther {
		t.Fatalf("accept: want 303, got %d", acceptResp.StatusCode)
	}
	newUser, err := database.GetUserByEmail("new@example.com")
	if err != nil || newUser == nil {
		t.Fatalf("user not created: %v", err)
	}

	// Token is now consumed — reuse should show error, not success.
	reuse, _ := anonWithJar.Post(srv.URL+"/invite/"+token, "application/x-www-form-urlencoded",
		strings.NewReader("new_password=strongpassword1&confirm_password=strongpassword1"))
	reuseBody, _ := io.ReadAll(reuse.Body)
	reuse.Body.Close()
	if reuse.StatusCode == http.StatusSeeOther {
		t.Fatal("used invite should not redirect to success")
	}
	if strings.Contains(string(reuseBody), "new_password") {
		t.Fatal("used invite should show error, not form")
	}
}

func TestParseDatetimeLocal(t *testing.T) {
	cases := []struct {
		input string
		empty bool
	}{
		{"2026-06-01T14:30", false},
		{"2026-06-01T14:30:00", false},
		{"", true},
		{"not-a-date", true},
		{"   ", true},
	}
	for _, tc := range cases {
		got := parseDatetimeLocal(tc.input)
		if tc.empty && got != "" {
			t.Errorf("parseDatetimeLocal(%q): want empty, got %q", tc.input, got)
		}
		if !tc.empty && got == "" {
			t.Errorf("parseDatetimeLocal(%q): want non-empty result", tc.input)
		}
	}
}

// --- helpers ---

func newJarClient() *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{
		Jar: jar,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func doLogin(t *testing.T, c *http.Client, base, email, password string) *http.Response {
	t.Helper()
	form := url.Values{"email": {email}, "password": {password}}
	resp, err := c.Post(base+"/login", "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	return resp
}

func hasSessionCookie(c *http.Client, base string) bool {
	u, _ := url.Parse(base)
	for _, ck := range c.Jar.Cookies(u) {
		if ck.Name == "openlogs_session" && ck.Value != "" {
			return true
		}
	}
	return false
}

func TestIngestBatchFlow(t *testing.T) {
	srv, database := newTestServer(t, "modern")
	project, _ := database.CreateProject("batchapp")

	validEntry := func(msg string) string {
		return fmt.Sprintf(`{"message":%q,"level_name":"INFO","channel":"web","datetime":"2026-01-01T00:00:00Z"}`, msg)
	}

	postBatch := func(key, body string) *http.Response {
		t.Helper()
		req, _ := http.NewRequest("POST", srv.URL+"/api/ingest/batch", strings.NewReader(body))
		if key != "" {
			req.Header.Set("X-API-Key", key)
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("do request: %v", err)
		}
		return resp
	}

	// Invalid API key → 401.
	r := postBatch("bad-key", "["+validEntry("test")+"]")
	if r.StatusCode != http.StatusUnauthorized {
		t.Fatalf("bad key: want 401, got %d", r.StatusCode)
	}
	r.Body.Close()

	// Missing API key → 401.
	r = postBatch("", "["+validEntry("test")+"]")
	if r.StatusCode != http.StatusUnauthorized {
		t.Fatalf("no key: want 401, got %d", r.StatusCode)
	}
	r.Body.Close()

	// Empty array → 400.
	r = postBatch(project.APIKey, `[]`)
	if r.StatusCode != http.StatusBadRequest {
		t.Fatalf("empty: want 400, got %d", r.StatusCode)
	}
	r.Body.Close()

	// Missing required field (level_name) → 400, nothing stored.
	r = postBatch(project.APIKey, `[{"message":"x","datetime":"2026-01-01T00:00:00Z"}]`)
	if r.StatusCode != http.StatusBadRequest {
		t.Fatalf("missing field: want 400, got %d", r.StatusCode)
	}
	r.Body.Close()
	logs, _ := database.QueryLogs(db.LogFilter{ProjectID: project.ID})
	if len(logs) != 0 {
		t.Fatalf("missing field: want 0 stored, got %d", len(logs))
	}

	// Array over 1000 entries → 400, nothing stored.
	entries := make([]string, 1001)
	for i := range entries {
		entries[i] = validEntry(fmt.Sprintf("msg %d", i))
	}
	r = postBatch(project.APIKey, "["+strings.Join(entries, ",")+"]")
	if r.StatusCode != http.StatusBadRequest {
		t.Fatalf("over limit: want 400, got %d", r.StatusCode)
	}
	r.Body.Close()
	logs, _ = database.QueryLogs(db.LogFilter{ProjectID: project.ID})
	if len(logs) != 0 {
		t.Fatalf("over limit: want 0 stored, got %d", len(logs))
	}

	// Valid batch → 201 with correct accepted count and all entries stored.
	batch := "[" + validEntry("alpha") + "," + validEntry("beta") + "," + validEntry("gamma") + "]"
	r = postBatch(project.APIKey, batch)
	if r.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(r.Body)
		r.Body.Close()
		t.Fatalf("valid: want 201, got %d: %s", r.StatusCode, b)
	}
	var result map[string]int
	json.NewDecoder(r.Body).Decode(&result)
	r.Body.Close()
	if result["accepted"] != 3 {
		t.Fatalf("accepted: want 3, got %d", result["accepted"])
	}
	logs, _ = database.QueryLogs(db.LogFilter{ProjectID: project.ID})
	if len(logs) != 3 {
		t.Fatalf("stored: want 3, got %d", len(logs))
	}
}
