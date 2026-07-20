## ADDED Requirements

### Requirement: Filter logs by date/time range
The log viewer SHALL allow filtering entries to a date/time range using an optional lower bound (`from`) and an optional upper bound (`to`). Each bound is independent: a `from` bound alone SHALL match entries at or after that instant, a `to` bound alone SHALL match entries at or before that instant, and both together SHALL match entries within the closed range. Bounds are compared against each entry's `logged_at` timestamp. The range filter SHALL be combined as an AND condition with the active level, channel, and full-text search filters, and SHALL be preserved across pagination ("Load more") requests. An absent, empty, or unparseable bound SHALL be ignored (treated as no bound) rather than producing an error.

#### Scenario: Lower bound only
- **WHEN** a user sets a `from` bound and no `to` bound
- **THEN** only log entries with `logged_at` at or after the `from` instant are displayed

#### Scenario: Upper bound only
- **WHEN** a user sets a `to` bound and no `from` bound
- **THEN** only log entries with `logged_at` at or before the `to` instant are displayed

#### Scenario: Closed range
- **WHEN** a user sets both a `from` and a `to` bound
- **THEN** only log entries whose `logged_at` falls within the inclusive range are displayed

#### Scenario: Range combined with other filters
- **WHEN** a user sets a date range together with a level, channel, or search filter
- **THEN** only entries matching all active filters, including the range, are displayed

#### Scenario: Range preserved across pagination
- **WHEN** a date range is active and the user loads the next page via "Load more"
- **THEN** the next page of older entries is constrained to the same date range

#### Scenario: Empty or invalid bound ignored
- **WHEN** a bound is left blank or cannot be parsed as a valid date/time
- **THEN** that bound is treated as absent and does not restrict or error the query
