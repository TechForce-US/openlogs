# deployment Specification

## Purpose
TBD - created by archiving change build-openlogs. Update Purpose after archive.
## Requirements
### Requirement: Docker Compose deployment with Caddy
The repository SHALL include a `docker-compose.yml` and `Caddyfile` that together run OpenLogs behind Caddy with automatic TLS via ACME/Let's Encrypt. The OpenLogs binary SHALL listen on port 8080 internally; Caddy SHALL handle TLS termination on ports 80 and 443 and reverse-proxy to the OpenLogs container.

#### Scenario: Stack starts with docker compose up
- **WHEN** a user runs `docker compose up -d` in the repository root
- **THEN** both the `openlogs` and `caddy` services start, Caddy obtains a TLS certificate, and the UI is accessible over HTTPS

#### Scenario: SQLite data persists across restarts
- **WHEN** the `openlogs` container is restarted
- **THEN** the SQLite database file is preserved via the `./data` volume mount and no log data is lost

### Requirement: Environment variable configuration
All runtime configuration SHALL be expressible via environment variables. The `docker-compose.yml` SHALL include an example `environment` block with all supported variables and their defaults.

| Variable | Default | Description |
|---|---|---|
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

### Requirement: Manual reverse proxy documentation
The repository README SHALL document how to run OpenLogs behind a manual reverse proxy (nginx, Apache, Traefik, etc.) without Docker Compose. The documentation SHALL cover: running the binary directly, pointing the reverse proxy at `localhost:8080`, and setting required environment variables.

#### Scenario: Manual setup documented
- **WHEN** a user reads the README
- **THEN** they can follow the manual setup instructions to deploy OpenLogs without Docker

### Requirement: Caddyfile template
The repository SHALL include a `Caddyfile` with a placeholder domain that users replace with their own. The Caddyfile SHALL configure Caddy to reverse-proxy to the `openlogs` container using the Docker internal hostname.

#### Scenario: Caddyfile is self-documenting
- **WHEN** a user opens the Caddyfile
- **THEN** the domain placeholder and proxy target are clearly identified with inline comments

