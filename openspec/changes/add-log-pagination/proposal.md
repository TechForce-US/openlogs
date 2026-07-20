## Why

The log viewer only ever shows the 100 most recent entries — there is no way to see older logs from the UI. Once a project has more than 100 entries (or a filter/search matches more than 100), everything past the first page is invisible. Users need to page back through history without losing the current filters.

## What Changes

- Add a **"Load more"** control at the bottom of the log list that fetches the next page of older entries and appends them, without a full page reload (HTMX).
- Use **keyset (cursor) pagination** keyed on `(logged_at, id)` rather than offset, so paging stays correct and efficient even as new logs arrive at the top via live tail.
- Pagination **preserves the active filters** (levels, channel, full-text search): each "Load more" request carries the same filters plus the cursor.
- The control appears only when more entries may exist and disappears when the end is reached.
- Live tail (SSE) continues to prepend new entries at the top, independent of paging.

## Capabilities

### New Capabilities

### Modified Capabilities

- `log-search`: The initial view remains the 100 newest entries, but the viewer now supports loading successive older pages via a cursor-based "Load more" control that respects the active filters.

## Impact

- **Code**: `internal/db` (`LogFilter` gains a keyset cursor; `QueryLogs` supports it), `internal/handler/logs.go` (parse cursor, build next-page link, choose response mode), `web/templates` (a `log-page` partial + load-more control; `logs.html` and filter/initial renders use it).
- **Behaviour**: additive — with ≤100 matching entries the UI looks the same (no load-more control). No API or config changes.
