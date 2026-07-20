## Why

Developers need a lightweight, self-hosted log aggregation tool that is simple to deploy and doesn't require external services or heavy infrastructure. OpenLogs is a language and framework agnostic log management server — applications send structured log entries over HTTP, and OpenLogs stores, indexes, and surfaces them through a real-time web UI. The first supported adapter targets PHP/Monolog, with a documented JSON schema that makes it straightforward to build adapters for any language or framework.

## What Changes

- New Go project (`openlogs`) providing a self-contained log management server
- HTTP ingest API accepting a structured JSON log format (language/framework agnostic)
- First-party adapter documentation for Monolog (PHP), with the wire format designed to accommodate other ecosystems
- Multi-project support with per-project API keys
- Real-time log streaming via Server-Sent Events (SSE)
- Full-text search over log messages and context via SQLite FTS5
- Two interchangeable UI themes (terminal and modern) with hand-written CSS
- 30-day automatic log retention with configurable TTL
- Docker Compose deployment with Caddy for automatic TLS
- CLI command for initial user creation

## Capabilities

### New Capabilities

- `log-ingest`: Accept log entries via HTTP POST authenticated by project API key; parse and store structured JSON log payloads; wire format is compatible with Monolog's JsonFormatter as the first reference implementation
- `log-search`: Query logs by project, level, channel, date range, and full-text search against message and context fields
- `live-tail`: Stream new log entries to the browser in real time via SSE, with HTMX-driven DOM updates
- `project-management`: Create and manage projects, each with a generated API key used for ingest authentication
- `user-auth`: Session-based authentication for web UI users; CLI command to create the initial admin user
- `log-retention`: Daily background job pruning logs older than the configured retention window
- `deployment`: Docker Compose configuration with Caddy reverse proxy for automatic TLS; documentation for manual reverse proxy setup

### Modified Capabilities

## Impact

- **New repository**: `openlogs` — Go module, no existing code affected
- **Dependencies**: `modernc.org/sqlite` (pure Go SQLite driver), HTMX (vendored JS, no build step)
- **External systems**: Any application capable of making an HTTP POST with a JSON body can send logs; Monolog adapter documented as the first reference
- **Deployment**: Docker Compose (openlogs + caddy), or any reverse proxy fronting port 8080
