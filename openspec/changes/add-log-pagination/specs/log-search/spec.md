## MODIFIED Requirements

### Requirement: Display logs in reverse chronological order
The log viewer SHALL display log entries for a given project ordered by `logged_at` descending, with `id` descending as a tiebreaker (newest first). On initial page load, the 100 most recent entries SHALL be shown; older entries are reachable via the load-more control (see "Load older entries with pagination").

#### Scenario: Initial log viewer load
- **WHEN** an authenticated user navigates to `/projects/{id}`
- **THEN** up to 100 log entries for that project are displayed, newest first

#### Scenario: Empty project
- **WHEN** an authenticated user navigates to a project with no log entries
- **THEN** an empty state message is displayed

#### Scenario: Stable ordering for equal timestamps
- **WHEN** multiple entries share the same `logged_at`
- **THEN** they are ordered by `id` descending so the overall order is deterministic

## ADDED Requirements

### Requirement: Load older entries with pagination
When more entries exist than are currently shown, the log viewer SHALL present a "Load more" control that fetches the next page of older entries and appends them to the list without a full page reload. Paging SHALL use a keyset cursor based on the last shown entry's `(logged_at, id)` — requesting entries strictly older than that cursor — so pages remain correct and duplicate-free while new entries arrive at the top via live tail. The control SHALL be shown only when a further page may exist and SHALL be removed once no older entries remain.

#### Scenario: Load the next page
- **WHEN** a project has more matching entries than are currently displayed and the user activates "Load more"
- **THEN** the next page of older entries is appended below the current ones, newest-of-the-batch first, without reloading the page

#### Scenario: End of results
- **WHEN** the user loads a page and no older entries remain
- **THEN** no further entries are appended and the "Load more" control is removed

#### Scenario: No control when everything fits
- **WHEN** a project (or the current filter/search) has 100 or fewer matching entries
- **THEN** no "Load more" control is shown

#### Scenario: No duplicates while live tail prepends
- **WHEN** new entries are ingested and prepended via live tail after the first page has loaded
- **THEN** activating "Load more" still returns only older entries and never duplicates an already-shown entry

### Requirement: Pagination preserves active filters
Each "Load more" request SHALL carry the currently active filters — severity levels, channel, and full-text search — so paginated results reflect the same query as the first page. Changing any filter SHALL reset pagination to the first page.

#### Scenario: Load more within a filtered view
- **WHEN** a level, channel, or search filter is active and the user activates "Load more"
- **THEN** the appended entries match the same filters as the currently displayed entries

#### Scenario: Changing a filter resets paging
- **WHEN** the user changes a filter or search term after having loaded additional pages
- **THEN** the list is replaced with the first page of results for the new filter
