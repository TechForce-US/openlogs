## ADDED Requirements

### Requirement: Display logs in reverse chronological order
The log viewer SHALL display log entries for a given project ordered by `logged_at` descending (newest first). On initial page load, the 100 most recent entries SHALL be shown.

#### Scenario: Initial log viewer load
- **WHEN** an authenticated user navigates to `/projects/{id}`
- **THEN** up to 100 log entries for that project are displayed, newest first

#### Scenario: Empty project
- **WHEN** an authenticated user navigates to a project with no log entries
- **THEN** an empty state message is displayed

### Requirement: Filter logs by severity level
The log viewer SHALL allow filtering entries by one or more severity levels. Level filters are applied as an AND condition with other active filters. Selecting no levels SHALL show all levels.

#### Scenario: Single level filter applied
- **WHEN** a user selects only the ERROR level filter
- **THEN** only log entries with `level="ERROR"` for that project are displayed

#### Scenario: Multiple level filters applied
- **WHEN** a user selects ERROR and WARNING level filters
- **THEN** log entries with `level="ERROR"` or `level="WARNING"` are displayed

#### Scenario: All level filters cleared
- **WHEN** a user clears all level filter selections
- **THEN** log entries of all levels are displayed

### Requirement: Filter logs by channel
The log viewer SHALL allow filtering entries by channel name. The channel filter accepts a plain text string and matches entries whose `channel` field equals that value (case-insensitive). An empty channel filter shows all channels.

#### Scenario: Channel filter applied
- **WHEN** a user enters "billing" in the channel filter
- **THEN** only log entries with `channel="billing"` (case-insensitive) are displayed

### Requirement: Full-text search over message and context
The log viewer SHALL support full-text search using SQLite FTS5 over the `message` and `context` fields. Search is triggered on user input (debounced). The search query is passed as an FTS5 MATCH expression.

#### Scenario: Search term matches message
- **WHEN** a user enters a search term that appears in one or more log messages
- **THEN** only matching log entries are displayed

#### Scenario: Search term matches context value
- **WHEN** a user enters a search term that appears in the JSON context of a log entry
- **THEN** that log entry is included in results

#### Scenario: No results
- **WHEN** a search term matches no log entries
- **THEN** an empty state message is displayed

### Requirement: View log entry detail
Each log entry in the list SHALL be expandable to show full detail including the raw `context` and `extra` JSON, formatted for readability.

#### Scenario: Log entry expanded
- **WHEN** a user clicks a log entry row
- **THEN** the `context` and `extra` JSON are displayed in a formatted, readable view below the row
