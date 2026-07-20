# live-tail Specification

## Purpose
TBD - created by archiving change build-openlogs. Update Purpose after archive.
## Requirements
### Requirement: Real-time log streaming via SSE
The system SHALL expose a `GET /projects/{id}/stream` endpoint that returns a `text/event-stream` response. Each new log entry ingested for the project SHALL be pushed as an SSE event to all active subscribers. The HTMX SSE extension SHALL be used on the client to receive events and prepend new log rows to the log list without a page reload.

#### Scenario: New log appears in real time
- **WHEN** a log entry is ingested for a project
- **THEN** all browsers viewing that project's log viewer receive the new entry via SSE within one second and it appears at the top of the log list

#### Scenario: No active subscribers
- **WHEN** a log entry is ingested for a project with no active SSE subscribers
- **THEN** the log is stored normally and no broadcast error occurs

### Requirement: Per-project SSE subscription
The SSE broker SHALL maintain independent subscription channels per project. A subscriber MUST only receive events for the project they are viewing.

#### Scenario: Subscriber isolation
- **WHEN** logs are ingested for project A and project B simultaneously
- **THEN** a browser viewing project A receives only project A events, and a browser viewing project B receives only project B events

### Requirement: Subscription cleanup on disconnect
When an SSE client disconnects, its subscription MUST be removed from the in-memory broker immediately to prevent goroutine leaks.

#### Scenario: Browser tab closed
- **WHEN** a user closes the browser tab with an active SSE connection
- **THEN** the server detects the closed connection via context cancellation and removes the subscriber from the broker

### Requirement: Client auto-reconnect
The browser's native `EventSource` API provides automatic reconnect behaviour. The server SHALL NOT implement custom reconnect logic. Logs that arrive during a disconnect gap are not replayed via SSE; the client fetches current state via the normal page load on reconnect.

#### Scenario: Reconnect after server restart
- **WHEN** the OpenLogs server restarts
- **THEN** the browser's EventSource reconnects automatically and the page displays the latest logs from the database

