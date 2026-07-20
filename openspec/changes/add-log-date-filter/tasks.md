## 1. Database layer

- [x] 1.1 Add `From` and `To` string fields (canonical UTC `logged_at` form) to `db.LogFilter` in `internal/db/logs.go`, documenting that empty means unbounded.
- [x] 1.2 In `queryLogs`, append `AND l.logged_at >= ?` when `From` is non-empty and `AND l.logged_at <= ?` when `To` is non-empty, adding the corresponding bound params before the cursor/ORDER BY clauses.

## 2. Handler layer

- [x] 2.1 In `handler.Logs`, read `from` and `to` query params and normalize each `datetime-local` value (`2006-01-02T15:04`) into canonical UTC (`2006-01-02 15:04:05`) via a helper, dropping unparseable/empty values.
- [x] 2.2 Pass the normalized bounds into the `db.LogFilter` and add `From`/`To` to the `logsView` (and template data) so the inputs re-populate after a filter change.
- [x] 2.3 Extend `buildLoadMoreURL` to carry the active `from`/`to` params so pagination preserves the range.

## 3. Template / UI

- [x] 3.1 Add two `datetime-local` inputs (`name="from"`, `name="to"`) to the filter form in `web/templates/logs.html`, wired to the existing HTMX trigger and pre-filled from the view data.

## 4. Tests & verification

- [x] 4.1 Add `internal/db` tests covering from-only, to-only, closed-range, and range-combined-with-level filtering (extend the patterns in `db_test.go`).
- [x] 4.2 Add a handler test asserting `from`/`to` are parsed, applied, and echoed into `buildLoadMoreURL` (or the rendered load-more URL).
- [x] 4.3 Run `go test ./...` and `go vet ./...`; manually verify from-only, to-only, and closed-range filtering plus "Load more" in the running app.
