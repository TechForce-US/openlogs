# log-retention Specification

## Purpose
TBD - created by archiving change build-openlogs. Update Purpose after archive.
## Requirements
### Requirement: Automatic log pruning
The system SHALL run a retention job at startup and every 24 hours thereafter. The job SHALL delete all log entries whose `logged_at` timestamp is older than the configured retention window.

#### Scenario: Logs pruned on schedule
- **WHEN** the retention job runs
- **THEN** all log entries with `logged_at < now - retention_days` are deleted from the database

#### Scenario: Retention job runs at startup
- **WHEN** the OpenLogs server starts
- **THEN** the retention job runs immediately before the server begins accepting requests (or concurrently on startup)

### Requirement: Configurable retention window
The retention window SHALL be configurable via the `OPENLOGS_RETENTION_DAYS` environment variable. The default value SHALL be 30 days. The value SHALL be a positive integer; invalid values SHALL cause the server to exit with an error on startup.

#### Scenario: Default retention applied
- **WHEN** `OPENLOGS_RETENTION_DAYS` is not set
- **THEN** logs older than 30 days are pruned

#### Scenario: Custom retention applied
- **WHEN** `OPENLOGS_RETENTION_DAYS=7` is set
- **THEN** logs older than 7 days are pruned

#### Scenario: Invalid retention value rejected
- **WHEN** `OPENLOGS_RETENTION_DAYS=abc` is set
- **THEN** the server exits with a clear error message on startup

### Requirement: FTS5 index consistency after pruning
After deleting log entries, the system SHALL ensure the FTS5 index does not retain orphaned entries. This SHALL be handled by the DELETE trigger on the `logs` table which fires for each deleted row.

#### Scenario: FTS5 entries removed with logs
- **WHEN** the retention job deletes log entries
- **THEN** corresponding FTS5 index entries are removed via the DELETE trigger

### Requirement: Retention job observability
The retention job SHALL log the number of rows deleted to stdout on each run, including when zero rows were deleted.

#### Scenario: Deletion count logged
- **WHEN** the retention job completes
- **THEN** a line such as `retention: deleted 142 log entries older than 30 days` is written to stdout

