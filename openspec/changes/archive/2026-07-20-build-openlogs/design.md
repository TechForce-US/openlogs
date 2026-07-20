## Context

OpenLogs is a new project with no existing codebase. It is a self-hosted log management server: applications POST structured JSON log entries over HTTP, and OpenLogs stores them in SQLite, indexes them with FTS5, and surfaces them through a real-time HTMX web UI. The system is designed for single-server, self-hosted deployment where operational simplicity is the primary constraint. The first reference adapter targets Monolog (PHP), but the ingest API is language-agnostic.

## Goals / Non-Goals

**Goals:**
- Single binary deployment with no runtime dependencies
- Language-agnostic HTTP ingest API with JSON payload
- Real-time log streaming to the browser via SSE
- Multi-project support with per-project API key auth
- Full-text search over message and context fields
- 30-day automatic retention (configurable)
- Two swappable CSS themes (terminal, modern)
- Docker Compose + Caddy for zero-config TLS deployment

**Non-Goals:**
- Horizontal scaling / clustering (single SQLite file)
- Log transformation, parsing, or enrichment beyond field storage
- Alerting or notifications (v1)
- Client libraries beyond documentation
- Log shipping in the pull direction (server never fetches logs)
- User roles or permissions beyond authenticated/unauthenticated

## Decisions

### 1. SQLite with FTS5 over a separate search engine

SQLite ships in the binary via `modernc.org/sqlite`. FTS5 provides full-text search over message and context with no additional process. Retention is a single `DELETE` statement. The trade-off is no horizontal scaling, but self-hosted log volumes rarely exceed what SQLite handles comfortably (tens of millions of rows). WAL mode is enabled to allow concurrent reads during writes.

**Alternative considered**: PostgreSQL — requires a separate process, complicates the "single binary" deployment story significantly.

### 2. modernc.org/sqlite over mattn/go-sqlite3

Pure Go driver — no CGO, no gcc dependency. Cross-compilation and Docker builds work with a plain `go build`. Performance difference at self-hosted log volumes is negligible.

**Alternative considered**: `mattn/go-sqlite3` — more battle-tested but requires CGO, which complicates CI, Docker images, and Windows builds.

### 3. HTMX + html/template over a JS framework

No build step. HTMX ships as a single vendored JS file (~14KB). Server-rendered HTML via `html/template` integrates naturally with Go. HTMX's SSE extension handles live tail with no custom JS. The result is a UI with zero frontend tooling.

**Alternative considered**: React/Vue — introduces a build pipeline, node_modules, and a separate API layer. Excessive for a focused internal tool.

### 4. SSE over WebSockets for live tail

Log streaming is unidirectional (server → client). SSE is native HTTP/1.1, requires no upgrade handshake, and HTMX supports it natively via `hx-ext="sse"`. Auto-reconnect is built into the browser's `EventSource`. WebSockets would add complexity for no functional benefit.

### 5. In-memory pub/sub broker over DB polling

A `sync.Map` of `chan LogEntry` per project_id. When a log is ingested, it is written to SQLite and simultaneously broadcast to all active SSE subscribers for that project. Latency is near-zero. The trade-off: logs ingested between a client disconnect and reconnect are not replayed via SSE — the client fetches the last N logs on reconnect via the normal page load path.

**Alternative considered**: Polling the DB every N seconds — adds latency, unnecessary read pressure, and complexity in tracking "last seen" IDs client-side.

### 6. API key identifies the project

`POST /api/ingest` with `X-API-Key: <key>` — the key is a lookup into the projects table, so the caller needs only one config value. No project ID required in the URL. This simplifies adapter configuration to a single endpoint + key pair.

### 7. Embedded assets via //go:embed

`web/static/` and `web/templates/` are embedded at compile time. The binary is self-contained — no asset path configuration, no risk of missing files at runtime.

### 8. Session cookies over JWT

Single-server deployment with SQLite-backed sessions. Session ID is a UUID stored in an `httpOnly`, `Secure`, `SameSite=Strict` cookie. No token rotation complexity. Sessions expire after 24 hours. JWTs add no benefit in a single-server context.

### 9. CLI `create-user` over a first-run web wizard

`openlogs create-user admin@example.com` — prompts for a password and writes the bcrypt hash to the DB. Works cleanly with Docker (`docker exec`), is scriptable in init scripts, and avoids a stateful setup flow in the web server.

### 10. Two CSS themes via env var

`OPENLOGS_THEME=terminal|modern` selects which stylesheet is served. Both are hand-written with CSS custom properties. No external CSS framework. Users swap by changing one env var and restarting.

## Risks / Trade-offs

- **SQLite write contention under burst ingest** → WAL mode allows concurrent readers; ingest is serialized per SQLite connection but self-hosted volumes are typically low enough that this is not a bottleneck. A connection pool with a single write connection is used.
- **FTS5 out-of-sync on partial failure** → FTS5 is kept in sync via INSERT/DELETE triggers. If a trigger fires but the parent transaction rolls back, the FTS index may drift. Mitigation: run ingest and FTS trigger in the same transaction; document an `openlogs reindex` command for recovery.
- **In-memory broker lost on restart** → SSE subscribers are dropped on restart. Mitigation: `EventSource` auto-reconnects; the reconnect causes a full page reload which fetches the latest logs from DB.
- **No horizontal scaling** → Single binary, single SQLite file. Acceptable for v1 self-hosted use. If scaling is needed in the future, the ingest API and web UI can be split, but this requires a shared DB or log forwarding layer — out of scope.
- **Log volume growth** → FTS5 index grows proportionally with log volume. The 30-day retention keeps the DB size bounded. Users needing longer retention are directed to rotate/archive SQLite files.

## Open Questions

- None — all decisions are resolved for v1.
