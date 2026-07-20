## 1. Database layer

- [x] 1.1 Extend `db.LogFilter` in `internal/db/logs.go` with a keyset cursor (`BeforeLoggedAt string`, `BeforeID int64`); remove the now-unused `Offset`
- [x] 1.2 In `QueryLogs`, when the cursor is set, add `AND (logged_at < ? OR (logged_at = ? AND id < ?))` (composing with the FTS join and other filters); keep ordering `logged_at DESC, id DESC`
- [x] 1.3 Fetch `Limit+1` rows internally so the caller can detect whether a further page exists (`QueryLogsPage` returns rows + `hasMore`; shared `queryLogs` builder)

## 2. Handler

- [x] 2.1 In `internal/handler/logs.go`, parse `before_time` and `before_id` query params; treat their presence as a "load more" request
- [x] 2.2 Query with the cursor + current filters; trim to page size and compute `hasMore` and the next cursor (last kept row's `logged_at`/`id`)
- [x] 2.3 Build the load-more URL with `net/url` from the current filters (levels, channel, `q`) plus `before_time`/`before_id`
- [x] 2.4 Choose response mode: cursor present → `log-page` partial (for `outerHTML` swap of the control); else `HX-Request` → filter/initial fragment; else full page

## 3. Templates

- [x] 3.1 Add `web/templates/partials/log-page.html`: renders the batch of `log-row`s and, when there is a next page, a trailing `<li class="load-more">` with `hx-get`, `hx-target="closest .load-more"`, `hx-swap="outerHTML"`
- [x] 3.2 Update the initial render and filter-swap paths to use `log-page` (removed the now-unused `log-list` partial and `logListView`); empty state is first-page-only
- [x] 3.3 Load-more control lives inside `#log-list` so appends land in place and coexist with SSE `afterbegin` prepends

## 4. Styling

- [x] 4.1 Style the `.load-more` control in both `web/static/modern.css` and `web/static/terminal.css`

## 5. Verification

- [x] 5.1 Unit test `QueryLogs`/`QueryLogsPage` cursor: 250 same-timestamp entries paged in 100s — no overlap/gaps, descending order, exact page count (`TestQueryLogsPagePaginates`)
- [x] 5.2 HTTP test the handler: first page 100 + control, load-more returns the next 50 with the control gone, filters carried into the load-more URL (`TestLogPagination`)
- [x] 5.3 `hasMore` boundary: exactly 100 → no control; 101 → control present (`TestQueryLogsPageHasMoreBoundary`)
- [x] 5.4 Live check via the compiled binary: seeded 150 filtered entries, paged 100 → 50 with the control removed at the end; `go build`/`go vet`/`go test ./...` pass
