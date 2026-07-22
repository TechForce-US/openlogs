# log-ingest Specification

## Purpose
TBD - created by archiving change build-openlogs. Update Purpose after archive.
## Requirements
### Requirement: Accept structured log entries via HTTP
The system SHALL expose a `POST /api/ingest` endpoint that accepts structured JSON log entries from any HTTP client. Authentication is performed via an `X-API-Key` header whose value identifies the target project.

#### Scenario: Valid log entry is stored
- **WHEN** a client POSTs a valid JSON payload with a valid `X-API-Key` header
- **THEN** the log entry is persisted to the database and the server returns HTTP 201

#### Scenario: Invalid API key is rejected
- **WHEN** a client POSTs with an `X-API-Key` value that does not match any project
- **THEN** the server returns HTTP 401 and no log entry is stored

#### Scenario: Missing API key is rejected
- **WHEN** a client POSTs without an `X-API-Key` header
- **THEN** the server returns HTTP 401

#### Scenario: Malformed JSON is rejected
- **WHEN** a client POSTs a body that is not valid JSON
- **THEN** the server returns HTTP 400

#### Scenario: Missing required field is rejected
- **WHEN** a client POSTs a payload that omits the `message` or `level_name` field
- **THEN** the server returns HTTP 400

### Requirement: JSON payload schema
The ingest endpoint SHALL accept a JSON object with the following fields. This schema is compatible with Monolog's `JsonFormatter` output and serves as the reference wire format for adapter authors.

| Field | Type | Required | Description |
|---|---|---|---|
| `message` | string | yes | Human-readable log message |
| `level_name` | string | yes | Severity name: DEBUG, INFO, NOTICE, WARNING, ERROR, CRITICAL, ALERT, EMERGENCY |
| `level` | integer | no | Numeric severity (Monolog convention: 100–600); stored for sort ordering |
| `channel` | string | no | Logical source name (e.g. `billing`, `auth`); defaults to empty string |
| `datetime` | string | yes | ISO 8601 timestamp of when the event occurred |
| `context` | object | no | Arbitrary key/value structured data; stored as JSON text; defaults to `{}` |
| `extra` | object | no | Additional metadata added by log processors; stored as JSON text; defaults to `{}` |

#### Scenario: Optional fields default gracefully
- **WHEN** a client POSTs a payload omitting `context`, `extra`, `level`, and `channel`
- **THEN** the log entry is stored with `context={}`, `extra={}`, `level_num=0`, and `channel=""`

#### Scenario: Monolog JsonFormatter payload is accepted
- **WHEN** a Monolog-configured application POSTs a `JsonFormatter`-encoded log entry
- **THEN** all fields are stored correctly without transformation

### Requirement: Level normalisation
The system SHALL normalise `level_name` to uppercase and store the corresponding `level_num` integer for sort ordering. If the `level` integer field is present it SHALL be stored as `level_num`; otherwise `level_num` is derived from `level_name` using the Monolog convention (DEBUG=100, INFO=200, NOTICE=250, WARNING=300, ERROR=400, CRITICAL=500, ALERT=550, EMERGENCY=600). Unrecognised level names SHALL be stored as-is with `level_num=0`.

#### Scenario: level_name is normalised to uppercase
- **WHEN** a client sends `level_name: "error"`
- **THEN** it is stored as `level="ERROR"` and `level_num=400`

#### Scenario: Unknown level_name is stored without error
- **WHEN** a client sends `level_name: "CUSTOM"`
- **THEN** it is stored as `level="CUSTOM"` and `level_num=0`, and HTTP 201 is returned

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

