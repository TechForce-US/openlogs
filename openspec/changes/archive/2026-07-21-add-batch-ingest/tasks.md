## 1. Database layer

- [x] 1.1 Add `InsertLogs(logs []*Log) ([]*Log, error)` to `internal/db/logs.go`: open an explicit transaction, build a single multi-row INSERT with all args flattened, commit, recover IDs from `LastInsertId` using the sequential-offset formula (`lastID - N + 1 + i`), and return the slice with IDs populated.

## 2. Handler

- [x] 2.1 Add `IngestBatch` handler in `internal/handler/ingest.go`: apply a 10 MiB `MaxBytesReader`, decode the body as `[]ingestPayload`, reject with 400 if empty or length > 1000.
- [x] 2.2 Validate and normalize every entry in a first pass (reuse `parseDatetime`, `jsonOrDefault`, level normalization from `Ingest`); accumulate into a `[]*db.Log` slice; return 400 with a descriptive message on the first invalid entry.
- [x] 2.3 Call `db.InsertLogs` with the validated slice; on success call `broker.Publish` for each stored entry and return `201` with `{"accepted": N}`.

## 3. Routing

- [x] 3.1 Register `POST /api/ingest/batch` in `internal/handler/router.go` using the existing `apiKeyMW` middleware, pointing to `a.IngestBatch`.

## 4. Tests

- [x] 4.1 Add `internal/db` test for `InsertLogs`: insert a batch of N entries, verify all rows exist with correct fields and that IDs are unique and sequential.
- [x] 4.2 Add handler tests for `IngestBatch`: valid batch returns 201 with correct `accepted` count; empty array returns 400; array over 1000 entries returns 400; any entry missing a required field returns 400 and stores nothing; invalid API key returns 401.
- [x] 4.3 Run `go test ./...` and `go vet ./...`.
