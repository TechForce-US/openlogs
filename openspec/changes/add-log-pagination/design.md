## Context

`GET /projects/{id}` renders the log viewer. The handler calls `db.QueryLogs` with `Limit: 100` and renders newest-first. HTMX filter changes re-request the same route with `HX-Request: true` and swap the `#log-list` inner HTML with a `log-list` partial. Live tail streams new entries via SSE and prepends them to `#log-list` (`sse-swap="log"`, `hx-swap="afterbegin"`). `db.QueryLogs`/`LogFilter` already order by `logged_at DESC, id DESC` and technically expose an unused `Offset`.

There is no way to see entries beyond the first 100. We need to page backwards through older entries while keeping filters and coexisting with live prepends.

## Goals / Non-Goals

**Goals:**
- Load successive pages of older entries from the UI, appended in place (no full reload).
- Correct, duplicate-free paging even while new logs arrive at the top.
- Preserve active filters (levels, channel, search) across pages.
- Purely additive: unchanged look when there are ≤100 matching entries.

**Non-Goals:**
- Infinite auto-scroll (a click-to-load button is enough for v1; auto-load-on-scroll can layer on later using the same endpoint).
- Jump-to-page / numbered pages or total counts.
- Paging the live-tail stream itself (SSE keeps prepending newest).
- A configurable page size (fixed at 100 to match today; can be revisited).

## Decisions

### 1. Keyset (cursor) pagination, not offset

Page boundaries are defined by the last row's sort key `(logged_at, id)`; the next page asks for rows strictly "older than" that cursor. Because the list is ordered `logged_at DESC, id DESC`, the predicate is:

```
logged_at < :cursor_time
   OR (logged_at = :cursor_time AND id < :cursor_id)
```

This is stable under inserts at the top (live tail): a cursor into older history is unaffected by new rows, so pages never shift, skip, or duplicate. It's also index-friendly — `idx_logs_project_logged (project_id, logged_at DESC)` supports it directly.

**Alternative considered**: `LIMIT/OFFSET`. Rejected — offsets drift as new rows are inserted at the top (live tail makes this constant), causing skipped/duplicated rows across "Load more" clicks; also `OFFSET` scans and discards skipped rows. The existing unused `Offset` field will be removed in favour of the cursor.

### 2. Detect "has more" with limit+1

`QueryLogs` fetches `Limit+1` rows; if it returns more than `Limit`, there is a next page. The handler trims to `Limit` and uses the last kept row as the next cursor. This avoids a separate `COUNT` and never renders a load-more control that would fetch zero rows.

### 3. One `log-page` partial, reused for initial / filter / load-more

The partial renders the batch of `log-row`s and, when there is a next page, a trailing **load-more control**. It is used three ways, differing only in how the client swaps it:

| Trigger | Target / swap | Result |
|---|---|---|
| Initial page load | rendered inside `logs.html` `#log-list` | first page + load-more |
| Filter/search change | `hx-target="#log-list" hx-swap="innerHTML"` | list replaced with page 1 |
| Load more click | control's `hx-target="closest .load-more"` `hx-swap="outerHTML"` | control replaced by next rows + next control |

Making the load-more control the **last `<li>` of the `#log-list`** keeps rows and control in one list, so `outerHTML` replacement appends the next batch exactly where the control was and installs the next control (or nothing at the end). This composes cleanly with SSE prepends at the top.

### 4. Cursor + filters travel in the load-more URL

The handler builds the control's `hx-get` URL from the current request's filters plus `before_time` and `before_id`:

```
/projects/{id}?level=ERROR&channel=billing&q=...&before_time=<logged_at>&before_id=<id>
```

The same handler parses `before_time`/`before_id`; when present it's a load-more request and returns the `log-page` partial for `outerHTML` swap; when absent but `HX-Request`, it's a filter change (`innerHTML`); otherwise it's the full page. Query-string building uses `net/url` so values are escaped.

### 5. Empty state stays first-page-only

The "no logs" empty state renders only for the first page (no cursor). A load-more request that yields no rows simply returns an empty fragment, removing the control.

## Risks / Trade-offs

- **Same `logged_at` across many rows** → the `(logged_at, id)` tiebreaker keeps ordering total and the cursor unambiguous, so equal-timestamp rows page correctly.
- **Filter changes mid-paging** → changing any filter re-requests without a cursor (page 1), resetting pagination; expected and desired.
- **Load-more shows entries also newly prepended by SSE** → cursor paging is over older history, disjoint from the top; duplicates are not possible because the cursor is strictly older than page 1's oldest row at first load. (A row could appear once via SSE at top and never again below, since ingest is newest.)
- **Response-mode detection** → keyed off `before_id` presence, which is unambiguous and not user-facing.

## Open Questions

- None for v1. Auto-load-on-scroll (IntersectionObserver or HTMX `revealed` trigger) could later reuse the exact same endpoint and partial with no server change.
