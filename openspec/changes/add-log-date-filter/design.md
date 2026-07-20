## Context

`logged_at` is stored in canonical UTC form (`2006-01-02 15:04:05`) and drives ordering, retention, and keyset pagination. Filtering is assembled in `db.queryLogs` by appending `AND` clauses onto a `strings.Builder` with bound parameters, then ordering by `logged_at DESC, id DESC`. `handler.Logs` parses query params into a `db.LogFilter`, renders the full page or the `log-page` partial (for HTMX filter changes and "Load more"), and `buildLoadMoreURL` re-encodes the active filters plus the keyset cursor for the next page.

Adding a date range means: two new optional bounds on `LogFilter`, two conditional `WHERE` clauses, param parsing/normalization in the handler, propagation through `buildLoadMoreURL`, and two inputs in the filter form. No schema change is needed — string comparison on the canonical `logged_at` format is chronologically correct, and the existing ordering already exercises that column.

## Goals / Non-Goals

**Goals:**
- Filter logs by an optional `from` and/or `to` bound, combinable with existing filters.
- Preserve the range across pagination like every other filter.
- Zero schema/migration/config/API impact; fully additive behavior.

**Non-Goals:**
- Relative/preset ranges ("last 15 minutes", "today"). Can layer on later purely in the template/handler.
- Filtering or windowing the live-tail SSE stream by date.
- Per-user timezone configuration or server-side timezone storage.

## Decisions

**Wire format: two params `from` and `to`, `datetime-local` inputs.**
Native `datetime-local` inputs need no JS and submit `YYYY-MM-DDTHH:MM` (local wall-clock). The handler converts to the DB's `YYYY-MM-DD HH:MM:SS` UTC form. Alternative — separate date and time fields, or a JS date-picker library — rejected: more surface area, a new dependency, and no functional gain for a developer-facing tool.

**Timezone: interpret input as the browser's local time, convert to UTC.**
`datetime-local` has no timezone. The cleanest correct approach is for the client to send an offset-aware value; but to keep the form dependency-free, the handler parses the wall-clock value and treats it as local to the server process (`time.ParseInLocation` with `time.Local`), then formats as UTC for comparison. This matches the single-operator/self-hosted deployment model where server and viewer are typically the same timezone. Documented as a known trade-off below. Alternative — a hidden JS-populated offset field — deferred to keep this change JS-free; the parsing seam makes it easy to add later.

**Comparison: inclusive bounds via `logged_at >= ?` and `logged_at <= ?`.**
Lexical string comparison on the fixed-width canonical format is chronologically correct, so no `datetime()` wrapping or index changes are required. Inclusive on both ends is the least surprising for a range picker. Bounds are only appended when non-empty and successfully parsed, mirroring how `Channel`/`Search` are conditionally added.

**Normalization at the handler boundary.** The handler owns parsing `datetime-local` → canonical UTC and drops unparseable values (ignored, not errored), keeping `db.LogFilter` string-typed and the DB layer free of presentation formats — consistent with the existing separation.

## Risks / Trade-offs

- [Wall-clock input interpreted in server timezone may mismatch a viewer in a different timezone] → Acceptable for the self-hosted, single-operator model; the handler parsing seam isolates the conversion so a client-supplied offset can be added without touching the DB layer.
- [User enters `from` later than `to`, yielding zero results] → Not an error; the empty state already communicates "no matching logs". Could add client-side validation later.
- [`datetime-local` browser support/UX varies] → Widely supported in evergreen browsers; degrades to a text input accepting the same `YYYY-MM-DDTHH:MM` format, which the handler still parses.

## Open Questions

- Should we add relative presets ("Last hour", "Today") in a follow-up? Out of scope here but the `from`/`to` params are the natural substrate.
