## MODIFIED Requirements

### Requirement: Caddyfile template
The repository SHALL include a `Caddyfile` that reads the public hostname from the `OPENLOGS_URL` environment variable using Caddy's native `{env.OPENLOGS_URL}` substitution. The Caddyfile SHALL configure Caddy to reverse-proxy to the `openlogs` container using the Docker internal hostname. Operators SHALL set `OPENLOGS_URL` in their `.env` file; they SHALL NOT need to edit the checked-in `Caddyfile`.

#### Scenario: Domain is configurable without editing the Caddyfile
- **WHEN** an operator sets `OPENLOGS_URL=logs.example.com` in their `.env` file
- **THEN** Caddy uses that domain as its hostname without any changes to the tracked `Caddyfile`

#### Scenario: Caddyfile is self-documenting
- **WHEN** a user opens the Caddyfile
- **THEN** the env var placeholder and proxy target are clearly identified with inline comments

### Requirement: Environment variable configuration
All runtime configuration SHALL be expressible via environment variables. The `docker-compose.yml` SHALL include an example `environment` block with all supported variables and their defaults.

| Variable | Default | Description |
|---|---|---|
| `OPENLOGS_URL` | `logs.example.com` | Public hostname Caddy serves; set in `.env`, never edit `Caddyfile` directly |
| `OPENLOGS_DB_PATH` | `/data/openlogs.db` | Path to the SQLite database file |
| `OPENLOGS_SECRET_KEY` | *(required)* | Secret used for session signing; server exits if unset |
| `OPENLOGS_PORT` | `8080` | Port the HTTP server listens on |
| `OPENLOGS_THEME` | `modern` | UI theme: `modern` or `terminal` |
| `OPENLOGS_RETENTION_DAYS` | `30` | Log retention window in days |

#### Scenario: Missing SECRET_KEY causes startup failure
- **WHEN** the server starts without `OPENLOGS_SECRET_KEY` set
- **THEN** the server exits immediately with a clear error message

#### Scenario: Theme switched via env var
- **WHEN** `OPENLOGS_THEME=terminal` is set and the server restarts
- **THEN** the terminal CSS theme is served to all clients

#### Scenario: URL configured via env var
- **WHEN** an operator adds `OPENLOGS_URL=my-domain.com` to their `.env`
- **THEN** Caddy serves OpenLogs at `my-domain.com` without any edits to `Caddyfile`
