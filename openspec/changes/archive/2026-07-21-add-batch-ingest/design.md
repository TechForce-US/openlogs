## Context

The existing `Ingest` handler in `internal/handler/ingest.go` decodes one `ingestPayload` from the request body, validates it, calls `db.InsertLog` (one `INSERT` statement), and calls `broker.Publish`. All validation and normalization logic — `parseDatetime`, `jsonOrDefault`, level normalization — is already expressed as package-level helpers and can be reused directly.

`db.InsertLog` issues one `INSERT INTO logs ... VALUES (...)` and returns the generated ID via `LastInsertId`. For a batch this would mean N round-trips to SQLite, which defeats the purpose. SQLite supports multi-row inserts: `INSERT INTO logs ... VALUES (...),(...),...` and the FTS5 `AFTER INSERT` trigger fires per-row even in multi-row form, keeping the FTS index correct.

## Goals / Non-Goals

**Goals:**
- Single endpoint, single transaction, single SQL statement for the whole batch.
- Reuse all existing validation/normalization logic without duplication.
- Return correct IDs for every inserted entry (needed for SSE broadcast, which publishes `db.Log` with ID).
- Additive only — no changes to the existing single-entry path.

**Non-Goals:**
- Per-entry partial success responses (all-or-nothing keeps client behavior simple).
- Async or queued ingest (synchronous insert, same as today).
- Client-specific batching libraries or SDK changes.

## Decisions

**Separate endpoint `POST /api/ingest/batch`, not a content-sniffing variant of `/api/ingest`.**
The existing endpoint contract (single object, 201 with no body) is preserved without any conditional logic. The new endpoint has a distinct URL and response shape, making both easy to document and test independently.

**All-or-nothing validation before any writes.**
Iterate the decoded array, validate every entry, accumulate `db.Log` values — only after all pass does `InsertLogs` run. This keeps the transaction clean and gives callers an unambiguous signal: either everything is in or nothing is.

**New `db.InsertLogs(logs []*Log) ([]*Log, error)` method using a single multi-row INSERT inside an explicit transaction.**
Builds `INSERT INTO logs (project_id, channel, level, level_num, message, context, extra, logged_at) VALUES (?,?,?,?,?,?,?,?),(?,?,?,?,?,?,?,?),...` with all args flattened. Wraps in `BEGIN`/`COMMIT` so the FTS triggers for all rows fire atomically. Returns the slice with `ID` populated.

**ID recovery via `LastInsertId` + sequential offset.**
SQLite's `LastInsertId` after a multi-row INSERT returns the rowid of the **last** inserted row. Because SQLite assigns contiguous autoincrement IDs within a single INSERT in a transaction, the IDs for N rows are `[lastID-N+1, lastID-N+2, ..., lastID]`. This lets us populate IDs for all entries without N additional SELECTs.

**Body limit: 10 MiB; entry count limit: 1000.**
10 MiB is set via `http.MaxBytesReader`, consistent with how the 1 MiB cap is applied in `Ingest`. 1000 entries is enforced after JSON decode. A 10 MiB JSON array of typical log payloads holds roughly 5000–10000 entries, so 1000 is a meaningful cap well within the body limit. The 400 response for limit violations is returned before any DB work.

**SSE broadcast: publish each entry individually after insert.**
`broker.Publish` takes a single `db.Log`. Broadcasting each entry from the stored slice keeps live-tail behavior identical to multiple sequential single ingests. The broker's per-subscriber buffer of 16 may drop entries for large batches; this is acceptable — the full data is in the DB and clients re-sync on page load.

## Risks / Trade-offs

- [Large batch holding DB write lock for extended time] → Mitigated by the 1000-entry cap and 10 MiB body limit; typical batches are well under both. SQLite WAL mode allows concurrent reads during the write.
- [LastInsertId + sequential ID assumption] → This is documented SQLite behavior for autoincrement within a single statement/transaction. If the DB ever migrates away from SQLite, `InsertLogs` is the single place to update.
- [SSE buffer overflow for large batches] → Drop-and-resync is the existing SSE contract; no new risk introduced.
