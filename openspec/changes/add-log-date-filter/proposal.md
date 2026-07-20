## Why

The log viewer lets users filter by level, channel, and full-text search, but there is no way to scope results to a point in time. Investigating an incident ("what happened between 14:00 and 15:00 last Tuesday?") currently means scrolling and loading page after page. Users need to filter logs to a specific date or date range.

## What Changes

- Add **date/time range filtering** to the log viewer: an optional "from" and "to" bound. Either bound may be supplied independently (from-only = everything since; to-only = everything up to; both = a closed range).
- The range filter is applied as an **AND** condition alongside the existing level, channel, and search filters, and is **preserved across pagination** ("Load more") the same way other filters are.
- Bounds are entered via native `datetime-local` inputs in the filter bar and submitted as query parameters (`from`, `to`).
- Inputs are interpreted in the user's local timezone and compared against the canonical UTC `logged_at` values in the database.
- Invalid or empty bounds are ignored rather than erroring, consistent with how empty channel/search inputs behave today.
- Live tail (SSE) behavior is unchanged; newly streamed entries continue to prepend regardless of the active range (the range constrains the queried history, not the live stream).

## Capabilities

### New Capabilities

<!-- none -->

### Modified Capabilities

- `log-search`: Adds a new requirement for filtering log entries by a date/time range (`from`/`to` bounds), combinable with existing filters and preserved across pagination.

## Impact

- **Code**: `internal/db` (`LogFilter` gains `From`/`To` bounds; `queryLogs` adds `logged_at >= ?` / `logged_at <= ?` clauses), `internal/handler/logs.go` (parse and normalize `from`/`to` params, carry them in `buildLoadMoreURL`, pass to the view), `web/templates/logs.html` (two `datetime-local` inputs in the filter form).
- **Behaviour**: additive — with no bounds supplied the viewer behaves exactly as today. No API (ingest) changes, no config changes, no schema/migration changes (`logged_at` is already indexed for ordering/retention).
