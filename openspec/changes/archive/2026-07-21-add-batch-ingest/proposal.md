## Why

The existing `POST /api/ingest` endpoint accepts one log entry per request. Applications that generate bursts of log entries (e.g. a Laravel job that logs 50 events before completing) must issue one HTTP round-trip per entry, which is costly at scale and increases ingest latency under load. A batch endpoint allows senders to buffer entries and deliver them in a single request, reducing connection overhead and allowing the database layer to insert many rows in one statement.

## What Changes

- Add a new `POST /api/ingest/batch` endpoint that accepts a JSON array of log entries, up to 1000 entries per request and 10 MiB per request body.
- All entries must belong to the same project (resolved from the `X-API-Key` header, same as the single-entry endpoint).
- The batch is validated all-or-nothing: if any entry is invalid the entire request is rejected with HTTP 400 and no rows are written.
- On success, all valid entries are inserted in a single multi-row SQL transaction and HTTP 201 is returned with `{"accepted": N}`.
- Each inserted entry is published to the SSE broker for live-tail, identical to the single-entry path.
- The existing `POST /api/ingest` endpoint is unchanged.

## Capabilities

### New Capabilities

<!-- none — this extends existing ingest -->

### Modified Capabilities

- `log-ingest`: Adds a batch ingest endpoint as a second supported ingest mechanism, alongside the existing single-entry endpoint.

## Impact

- **Code**: `internal/db` (new `InsertLogs` method with multi-row INSERT in a transaction), `internal/handler/ingest.go` (new `IngestBatch` handler reusing validation/normalization helpers from `Ingest`), `internal/handler/router.go` (register `POST /api/ingest/batch` with existing API key middleware).
- **Behaviour**: additive — the existing single-entry endpoint is unaffected. No schema, config, or auth changes required.
