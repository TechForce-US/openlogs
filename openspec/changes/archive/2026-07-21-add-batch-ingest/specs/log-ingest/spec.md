## ADDED Requirements

### Requirement: Accept batches of log entries via HTTP
The system SHALL expose a `POST /api/ingest/batch` endpoint that accepts a JSON array of log entry objects in a single request. Authentication SHALL use the same `X-API-Key` header as the single-entry endpoint, identifying the target project. The request body SHALL be capped at 10 MiB. The array SHALL contain at most 1000 entries; requests exceeding this limit SHALL be rejected with HTTP 400. An empty array SHALL be rejected with HTTP 400. Validation SHALL be all-or-nothing: if any entry in the array is invalid, the entire request SHALL be rejected with HTTP 400 and no entries SHALL be written. On success, all entries SHALL be inserted in a single database transaction and HTTP 201 SHALL be returned with a JSON body `{"accepted": N}` where N is the number of entries stored.

#### Scenario: Valid batch is stored
- **WHEN** a client POSTs a valid JSON array of 1–1000 log entries with a valid `X-API-Key` header
- **THEN** all entries are persisted to the database and the server returns HTTP 201 with `{"accepted": N}`

#### Scenario: Empty array is rejected
- **WHEN** a client POSTs an empty JSON array `[]`
- **THEN** the server returns HTTP 400 and no entries are stored

#### Scenario: Array exceeding 1000 entries is rejected
- **WHEN** a client POSTs a JSON array with more than 1000 entries
- **THEN** the server returns HTTP 400 and no entries are stored

#### Scenario: Any invalid entry rejects the whole batch
- **WHEN** a client POSTs a JSON array where at least one entry is missing a required field or has an unparseable datetime
- **THEN** the server returns HTTP 400 and no entries are stored

#### Scenario: Invalid API key is rejected
- **WHEN** a client POSTs to `/api/ingest/batch` with a missing or invalid `X-API-Key`
- **THEN** the server returns HTTP 401 and no entries are stored

#### Scenario: All entries are broadcast to live-tail subscribers
- **WHEN** a valid batch is stored and there are active SSE subscribers for the project
- **THEN** each stored entry is published to the broker individually

#### Scenario: Each entry in the batch follows the single-entry schema
- **WHEN** a client POSTs a batch where each object has the same fields as a single ingest payload
- **THEN** all normalisation rules (level uppercasing, level_num derivation, context/extra defaults) apply per-entry identically to the single-entry endpoint
