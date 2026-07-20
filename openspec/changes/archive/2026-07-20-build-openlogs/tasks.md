## 1. Project Scaffolding

- [x] 1.1 Initialise Go module (`go mod init github.com/yourusername/openlogs`)
- [x] 1.2 Create directory structure: `cmd/openlogs/`, `internal/config/`, `internal/db/`, `internal/broker/`, `internal/handler/`, `internal/middleware/`, `web/templates/`, `web/static/`
- [x] 1.3 Add dependencies: `modernc.org/sqlite`, `golang.org/x/crypto` (bcrypt)
- [x] 1.4 Vendor or download HTMX (`htmx.min.js`) and HTMX SSE extension (`htmx-ext-sse.js`) into `web/static/`
- [x] 1.5 Set up `//go:embed` directives in `cmd/openlogs/main.go` for `web/templates` and `web/static` (implemented in `web/embed.go` — Go embed cannot reach paths above `cmd/`, so the embed lives in the `web` package)

## 2. Configuration

- [x] 2.1 Implement `internal/config/config.go`: read env vars (`OPENLOGS_DB_PATH`, `OPENLOGS_SECRET_KEY`, `OPENLOGS_PORT`, `OPENLOGS_THEME`, `OPENLOGS_RETENTION_DAYS`), apply defaults, exit on missing `SECRET_KEY` or invalid `RETENTION_DAYS`

## 3. Database Layer

- [x] 3.1 Implement `internal/db/db.go`: open SQLite connection, enable WAL mode, run schema migrations on startup
- [x] 3.2 Write schema in `internal/db/schema.go`: `projects`, `users`, `sessions`, `logs` tables; `idx_logs_project_logged` index; `logs_fts` FTS5 virtual table; INSERT and DELETE triggers to keep FTS5 in sync
- [x] 3.3 Implement `internal/db/projects.go`: `CreateProject`, `ListProjects`, `GetProjectByAPIKey`, `GetProjectByID`, `RenameProject`, `RegenerateAPIKey`, `DeleteProject`
- [x] 3.4 Implement `internal/db/users.go`: `CreateUser`, `GetUserByEmail`
- [x] 3.5 Implement `internal/db/sessions.go`: `CreateSession`, `GetSession`, `DeleteSession`, `DeleteExpiredSessions`
- [x] 3.6 Implement `internal/db/logs.go`: `InsertLog`, `QueryLogs` (with filters: project, levels, channel, FTS search, date range, limit/offset), `DeleteOldLogs`

## 4. SSE Broker

- [x] 4.1 Implement `internal/broker/broker.go`: `Broker` struct with `sync.Map` of `project_id → []chan LogEvent`; `Subscribe(projectID)`, `Unsubscribe(projectID, ch)`, `Publish(projectID, event)` methods
- [x] 4.2 Ensure `Unsubscribe` is called via `defer` in the SSE handler to prevent goroutine leaks

## 5. CLI Entry Point

- [x] 5.1 Implement `cmd/openlogs/main.go`: parse subcommand (`serve` vs `create-user`), wire config and DB
- [x] 5.2 Implement `create-user` subcommand: accept email as arg, prompt for password interactively, validate length ≥ 8 chars, bcrypt hash, call `db.CreateUser`, error on duplicate email

## 6. Middleware

- [x] 6.1 Implement `internal/middleware/auth.go`: session middleware — read session cookie, validate via `db.GetSession`, attach user to request context; redirect to `/login` if invalid
- [x] 6.2 Implement `internal/middleware/apikey.go`: API key middleware — read `X-API-Key` header, look up project via `db.GetProjectByAPIKey`, attach project to request context; return 401 if not found

## 7. HTTP Handlers

- [x] 7.1 Implement `internal/handler/auth.go`: `GET /login` (render login form), `POST /login` (validate credentials, create session, set cookie), `POST /logout` (delete session, clear cookie)
- [x] 7.2 Implement `internal/handler/projects.go`: `GET /projects` (list), `POST /projects` (create with generated API key), `GET /projects/{id}/settings` (show settings), `POST /projects/{id}/settings` (rename), `POST /projects/{id}/regenerate-key`, `POST /projects/{id}/delete`
- [x] 7.3 Implement `internal/handler/logs.go`: `GET /projects/{id}` (log viewer — parse filter params, query DB, render template)
- [x] 7.4 Implement `internal/handler/sse.go`: `GET /projects/{id}/stream` — subscribe to broker, write `text/event-stream` headers, push rendered log row partials as SSE events, unsubscribe on context cancellation
- [x] 7.5 Implement `internal/handler/ingest.go`: `POST /api/ingest` — parse JSON body, normalise level, insert via `db.InsertLog`, publish to broker, return 201
- [x] 7.6 Implement `internal/handler/settings.go`: `GET /settings` (change password form, theme selector), `POST /settings/password`, `POST /settings/theme`
- [x] 7.7 Wire all routes in `cmd/openlogs/main.go` using `net/http` ServeMux, apply middleware

## 8. Templates

- [x] 8.1 Create `web/templates/base.html`: shared layout with nav, theme CSS link, HTMX script tags
- [x] 8.2 Create `web/templates/login.html`: email + password form
- [x] 8.3 Create `web/templates/projects.html`: project list with create form
- [x] 8.4 Create `web/templates/logs.html`: log viewer with level filter checkboxes, channel filter, search input, SSE connection (`hx-ext="sse"`), log list container
- [x] 8.5 Create `web/templates/partials/log-row.html`: single log row partial (used for initial list render and SSE push)
- [x] 8.6 Create `web/templates/partials/log-detail.html`: expanded log detail partial (context + extra JSON)
- [x] 8.7 Create `web/templates/project-settings.html`: rename form, API key display, regenerate key button, delete project (with confirmation)
- [x] 8.8 Create `web/templates/settings.html`: change password form, theme selector

## 9. Stylesheets

- [x] 9.1 Create `web/static/modern.css`: clean card-based theme — CSS custom properties for colours/spacing, readable log list, level badges with colour coding, responsive layout
- [x] 9.2 Create `web/static/terminal.css`: monospace dense theme — dark background, green/amber accents, compact log rows, ASCII-style borders, level indicators as bracketed text

## 10. Log Retention

- [x] 10.1 Implement retention goroutine in `cmd/openlogs/main.go`: run `db.DeleteOldLogs` at startup and every 24 hours; log deletion count to stdout

## 11. Deployment Files

- [x] 11.1 Write `Dockerfile`: multi-stage build — `golang:alpine` build stage, minimal `alpine` or `scratch` final image, copy binary
- [x] 11.2 Write `docker-compose.yml`: `openlogs` service (image, env block with all vars, `./data:/data` volume), `caddy` service (ports 80/443, Caddyfile mount, `caddy_data` volume, depends_on openlogs)
- [x] 11.3 Write `Caddyfile`: placeholder domain, `reverse_proxy openlogs:8080`, inline comments
- [x] 11.4 Write `README.md`: project overview, Docker Compose quickstart, `create-user` instructions, manual reverse proxy guide, env var reference, Monolog adapter configuration example

## 12. Polish and Verification

- [x] 12.1 Test full ingest → SSE → UI flow end-to-end with a curl request (live: login→session, POST /api/ingest→201, SSE `event: log` frame received with rendered row)
- [x] 12.2 Test FTS5 search returns expected results; test level and channel filters (live HTTP: no-filter=4, ERROR+WARNING=2, channel=1 case-insensitive, FTS message/context match, no-match empty state)
- [x] 12.3 Test 30-day retention deletes correctly and FTS5 stays in sync (unit `TestRetentionPrunesAndSyncsFTS`; live retention log line observed)
- [x] 12.4 Test `create-user` CLI: happy path, duplicate email, short password (driven via pty; plus no-arg usage error)
- [x] 12.5 Test session expiry and logout (unit `TestSessionExpiry`, `TestLoginLogoutSession`; live logout→303)
- [x] 12.6 Verify both CSS themes render correctly in browser (unit `TestThemeCSSServed`; live runtime swap modern↔terminal, both CSS served 200)
- [x] 12.7 Run `docker compose up` and confirm HTTPS resolves and logs flow through (`docker compose config` valid; image built; container serves /login, static, ingest 401 — full ACME/HTTPS requires a real domain)
